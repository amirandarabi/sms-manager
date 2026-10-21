// Package observability provides the shared OpenTelemetry setup, slog JSON
// logging, and PII-masking helpers used by every service, per
// ARCHITECTURE.md §18.3. This file covers the "core" half of that story
// (SMM-18): OTel provider setup and the logger. RED metrics middleware,
// health checks, and Pyroscope bootstrap are added on top of this in a
// follow-up story (SMM-19); they are intentionally not here.
package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Env vars read by Setup. These follow the OpenTelemetry autoinstrumentation
// convention (https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/)
// so the same collector endpoint config works for every language's SDK.
const (
	// EnvOTLPEndpoint is the OTLP collector endpoint, e.g.
	// "otel-collector.observability.svc:4317". Unset (or empty) means: use
	// the no-op exporter fallback (AC1) — no panics, no network calls.
	EnvOTLPEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
	// EnvOTLPProtocol selects the exporter transport: "grpc" (default) or
	// "http/protobuf".
	EnvOTLPProtocol = "OTEL_EXPORTER_OTLP_PROTOCOL"
	// EnvOTLPInsecure, when "true", disables TLS on the OTLP connection.
	// Intended for local dev / providersim only — never set in production.
	EnvOTLPInsecure = "OTEL_EXPORTER_OTLP_INSECURE"
)

// Config controls Setup. Every field has a safe zero value: a zero Config
// (plus a required ServiceName) produces fully no-op meter/tracer
// providers that never dial out, per AC1.
type Config struct {
	// ServiceName is the fixed "service" resource attribute / log field.
	// Required.
	ServiceName string
	// ServiceVersion is recorded on the OTel resource; optional.
	ServiceVersion string
	// Env is the deployment environment (e.g. "dev", "staging", "prod");
	// optional, recorded on the OTel resource.
	Env string

	// OTLPEndpoint overrides EnvOTLPEndpoint for tests/callers that don't
	// want to mutate process env. Empty means "read from env", and empty
	// after that means "no-op exporters".
	OTLPEndpoint string
	// OTLPProtocol overrides EnvOTLPProtocol ("grpc" or "http/protobuf").
	OTLPProtocol string
	// OTLPInsecure overrides EnvOTLPInsecure.
	OTLPInsecure bool

	// LogWriter is where the slog JSON handler writes. Defaults to
	// os.Stdout, per ARCHITECTURE.md §18.3 ("slog JSON to stdout").
	LogWriter io.Writer
}

// Observability bundles everything Setup produces: a structured logger and
// the OTel providers a service registers globally and instruments against.
type Observability struct {
	Logger         *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  otelmetric.MeterProvider

	shutdownFuncs []func(context.Context) error
}

// Shutdown flushes and closes every exporter/provider Setup created, in
// reverse order. Safe to call even when Setup fell back to no-op
// providers (it's then a no-op itself). Services should call this on
// graceful shutdown (ARCHITECTURE.md §18.4).
func (o *Observability) Shutdown(ctx context.Context) error {
	var firstErr error
	for i := len(o.shutdownFuncs) - 1; i >= 0; i-- {
		if err := o.shutdownFuncs[i](ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Setup builds the OTel meter/tracer providers and the slog logger for a
// service, per ARCHITECTURE.md §18.3. When no OTLP endpoint is configured
// (via cfg.OTLPEndpoint or the OTEL_EXPORTER_OTLP_ENDPOINT env var), it
// falls back to no-op exporters: the SDK providers are still constructed
// and registered globally (satisfying AC1 — "registers an OTel
// meter/tracer provider without panicking"), they just never export
// anything over the network.
//
// Setup registers the returned providers as the process-global OTel
// providers (otel.SetTracerProvider / otel.SetMeterProvider), so
// instrumented libraries that call otel.Tracer()/otel.Meter() pick them up
// without every call site threading *Observability through.
func Setup(ctx context.Context, cfg Config) (*Observability, error) {
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("observability: Config.ServiceName is required")
	}

	endpoint := cfg.OTLPEndpoint
	if endpoint == "" {
		endpoint = os.Getenv(EnvOTLPEndpoint)
	}
	protocol := cfg.OTLPProtocol
	if protocol == "" {
		protocol = os.Getenv(EnvOTLPProtocol)
	}
	insecure := cfg.OTLPInsecure || os.Getenv(EnvOTLPInsecure) == "true"

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Env),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("observability: building resource: %w", err)
	}

	o := &Observability{}

	tp, err := newTracerProvider(ctx, res, endpoint, protocol, insecure, o)
	if err != nil {
		return nil, fmt.Errorf("observability: tracer provider: %w", err)
	}
	o.TracerProvider = tp
	otel.SetTracerProvider(tp)

	mp, err := newMeterProvider(ctx, res, endpoint, protocol, insecure, o)
	if err != nil {
		return nil, fmt.Errorf("observability: meter provider: %w", err)
	}
	o.MeterProvider = mp
	otel.SetMeterProvider(mp)

	// ARCHITECTURE.md §14.1 requires W3C traceparent propagation through
	// Kafka headers (and HTTP/gRPC). The OTel SDK's global propagator
	// defaults to a no-op, so without this, every Inject()/Extract() call
	// anywhere in the process silently does nothing and trace context
	// never crosses a service/Kafka boundary.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	w := cfg.LogWriter
	if w == nil {
		w = os.Stdout
	}
	o.Logger = NewLogger(w, cfg.ServiceName)

	return o, nil
}

// newTracerProvider builds an sdktrace.TracerProvider. With no endpoint
// configured it has no span processor/exporter attached at all — spans are
// still created (satisfying AC1) but are dropped rather than exported.
func newTracerProvider(ctx context.Context, res *resource.Resource, endpoint, protocol string, insecure bool, o *Observability) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}

	if endpoint != "" {
		exp, err := newTraceExporter(ctx, endpoint, protocol, insecure)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
		o.shutdownFuncs = append(o.shutdownFuncs, exp.Shutdown)
	}

	tp := sdktrace.NewTracerProvider(opts...)
	o.shutdownFuncs = append(o.shutdownFuncs, tp.Shutdown)
	return tp, nil
}

// newMeterProvider builds a metric.MeterProvider. With no endpoint
// configured it has no metric reader/exporter attached — instruments can
// still be created and recorded against (satisfying AC1), the data just
// isn't collected anywhere.
func newMeterProvider(ctx context.Context, res *resource.Resource, endpoint, protocol string, insecure bool, o *Observability) (*metric.MeterProvider, error) {
	opts := []metric.Option{metric.WithResource(res)}

	if endpoint != "" {
		exp, err := newMetricExporter(ctx, endpoint, protocol, insecure)
		if err != nil {
			return nil, err
		}
		reader := metric.NewPeriodicReader(exp, metric.WithInterval(15*time.Second))
		opts = append(opts, metric.WithReader(reader))
		o.shutdownFuncs = append(o.shutdownFuncs, reader.Shutdown)
	}

	mp := metric.NewMeterProvider(opts...)
	o.shutdownFuncs = append(o.shutdownFuncs, mp.Shutdown)
	return mp, nil
}

func newTraceExporter(ctx context.Context, endpoint, protocol string, insecure bool) (sdktrace.SpanExporter, error) {
	if protocol == "http/protobuf" {
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
		if insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)
	}
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	return otlptracegrpc.New(ctx, opts...)
}

func newMetricExporter(ctx context.Context, endpoint, protocol string, insecure bool) (metric.Exporter, error) {
	if protocol == "http/protobuf" {
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
		if insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		return otlpmetrichttp.New(ctx, opts...)
	}
	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
	if insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	return otlpmetricgrpc.New(ctx, opts...)
}
