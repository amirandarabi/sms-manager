package observability

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TestSetup_NoOpFallbackWhenNoOTLPEndpoint covers AC1: a minimal harness
// calling Setup registers an OTel meter/tracer provider without panicking
// when no OTLP endpoint is configured, and those providers are usable
// (creating instruments/spans doesn't panic or error either).
func TestSetup_NoOpFallbackWhenNoOTLPEndpoint(t *testing.T) {
	t.Setenv(EnvOTLPEndpoint, "")

	var logBuf bytes.Buffer
	ctx := context.Background()

	o, err := Setup(ctx, Config{
		ServiceName: "test-service",
		LogWriter:   &logBuf,
	})
	if err != nil {
		t.Fatalf("Setup() returned unexpected error: %v", err)
	}
	if o == nil {
		t.Fatal("Setup() returned nil Observability")
	}
	if o.TracerProvider == nil {
		t.Fatal("expected a non-nil TracerProvider")
	}
	if o.MeterProvider == nil {
		t.Fatal("expected a non-nil MeterProvider")
	}
	if o.Logger == nil {
		t.Fatal("expected a non-nil Logger")
	}

	// Exercise both providers to make sure the no-op exporter path doesn't
	// panic when actually used, not just when constructed.
	tracer := o.TracerProvider.Tracer("smoke-test")
	_, span := tracer.Start(ctx, "test-span")
	span.End()

	meter := o.MeterProvider.Meter("smoke-test")
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Int64Counter() returned unexpected error: %v", err)
	}
	counter.Add(ctx, 1)

	o.Logger.InfoContext(ctx, "setup smoke test ok")

	if err := o.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() returned unexpected error: %v", err)
	}
}

// TestSetup_RegistersW3CTraceContextPropagator covers ARCHITECTURE.md
// §14.1 ("Trace context is propagated in Kafka headers (W3C
// traceparent)"): Setup must register a real text-map propagator
// globally, or Inject() silently produces an empty carrier and trace
// context never crosses a service/Kafka boundary.
func TestSetup_RegistersW3CTraceContextPropagator(t *testing.T) {
	t.Setenv(EnvOTLPEndpoint, "")

	o, err := Setup(context.Background(), Config{ServiceName: "propagator-test"})
	if err != nil {
		t.Fatalf("Setup() returned unexpected error: %v", err)
	}
	defer o.Shutdown(context.Background()) //nolint:errcheck // best-effort cleanup

	prop := otel.GetTextMapPropagator()

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: [16]byte{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36},
		SpanID:  [8]byte{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7},
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	carrier := propagation.MapCarrier{}
	prop.Inject(ctx, carrier)

	got := carrier.Get("traceparent")
	if got == "" {
		t.Fatal("expected Setup() to register a propagator that injects a non-empty traceparent header; got empty carrier (still the default no-op propagator)")
	}
	if !strings.Contains(got, sc.TraceID().String()) {
		t.Errorf("traceparent = %q, want it to contain trace id %s", got, sc.TraceID().String())
	}
}

func TestSetup_RequiresServiceName(t *testing.T) {
	if _, err := Setup(context.Background(), Config{}); err == nil {
		t.Fatal("expected Setup() to error when ServiceName is empty")
	}
}

func TestSetup_OTLPExporterPathConstructsWithoutError(t *testing.T) {
	// With an endpoint configured, Setup must still construct the
	// exporters/providers without erroring or panicking. OTLP gRPC/HTTP
	// exporters in this SDK connect lazily, so this does not require a
	// live collector. Shutdown is expected to fail fast here (there is no
	// collector at this address to flush to) — we only assert it returns
	// promptly rather than hanging, not that it succeeds.
	o, err := Setup(context.Background(), Config{
		ServiceName:  "svc-with-otlp",
		OTLPEndpoint: "127.0.0.1:4317",
		OTLPInsecure: true,
	})
	if err != nil {
		t.Fatalf("Setup() returned unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = o.Shutdown(ctx) // error expected (no live collector); must not hang or panic.
}
