package observability

import (
	"bytes"
	"context"
	"testing"
	"time"
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
