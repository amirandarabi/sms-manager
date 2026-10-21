# pkg/observability

Shared OpenTelemetry setup, `slog` JSON logging, and PII-masking helpers used by every service, per ARCHITECTURE.md §18.3.

This README covers the **core** half of the package, scaffolded in SMM-18: OTel provider setup and the structured logger. RED metrics middleware, `/livez`/`/readyz`, and Pyroscope bootstrap are a separate follow-up (SMM-19) and are not covered here.

## Setup

Every service calls `observability.Setup` once at startup:

```go
import (
    "context"
    "os/signal"
    "syscall"

    "github.com/amirandarabi/sms-manager/pkg/observability"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    obs, err := observability.Setup(ctx, observability.Config{
        ServiceName:    "sms-api",
        ServiceVersion: buildVersion,
        Env:            "dev",
    })
    if err != nil {
        panic(err)
    }
    defer obs.Shutdown(context.Background())

    logger := obs.Logger
    logger.InfoContext(ctx, "service starting")

    // Instrumented code calls otel.Tracer("...") / otel.Meter("...") as
    // usual — Setup registers obs.TracerProvider / obs.MeterProvider as
    // the process-global OTel providers.
}
```

`Setup` always returns a usable `*Observability` (never partially
initialized) and never panics:

- If `OTEL_EXPORTER_OTLP_ENDPOINT` (or `Config.OTLPEndpoint`) is set, it
  configures a real OTLP exporter (gRPC by default; set
  `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf` for HTTP) for both traces and
  metrics.
- If it is unset, the SDK meter/tracer providers are still constructed and
  registered globally, but with **no exporter attached** — spans and
  metric instruments can be created and used safely in tests/local dev,
  they just aren't sent anywhere. This is the "no-op fallback" required by
  AC1.

| Env var | Purpose | Default |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector address, e.g. `otel-collector.observability.svc:4317` | unset → no-op exporters |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `grpc` or `http/protobuf` | `grpc` |
| `OTEL_EXPORTER_OTLP_INSECURE` | `true` to disable TLS (local dev / providersim only) | `false` |

Call `obs.Shutdown(ctx)` on graceful shutdown (ARCHITECTURE.md §18.4) to
flush pending spans/metrics and close exporters.

## Logging

`obs.Logger` is a `*slog.Logger` that:

- Writes JSON to the configured writer (`os.Stdout` by default).
- Always includes a fixed `service` field.
- Automatically injects `trace_id`/`span_id` from the active OTel span in
  the context passed to `*Context` logging methods (e.g. `InfoContext`).
- Masks any Iranian MSISDN found in the message or in any string-valued
  attribute, per ARCHITECTURE.md §10 (e.g. `0912***1234`) — see "PII
  masking" below.

Attach the remaining ARCHITECTURE.md §18.3 mandatory fields
(`message_id`, `client_id`, `error_code`) with the `With*` helpers:

```go
logger := observability.WithMessageID(obs.Logger, messageID)
logger = observability.WithClientID(logger, clientID)
logger.InfoContext(ctx, "message accepted")

// On failure, attach the SMS-xxxx error code (never a raw provider error):
observability.WithErrorCode(logger, err.Code).ErrorContext(ctx, "send failed")
```

Never log OTP codes or full message bodies of `otp`-type messages — the
masking helper only protects phone numbers, not arbitrary secrets.

## PII masking

`observability.MaskMSISDN(s string) string` replaces every Iranian MSISDN
in `s` — in local (`0912...`), `+98...`, or `0098...` form — with the
masked form `0912***1234` (first 4 / last 4 digits kept, middle replaced
with `***`), per ARCHITECTURE.md §10. It's used internally by the logger,
and can be called directly wherever a string with a possible phone number
needs to be sanitized before it leaves the process (e.g. provider error
messages surfaced to clients).

## What's out of scope here (see SMM-19)

- RED metrics middleware for HTTP/gRPC endpoints and Kafka consumers.
- `/livez` / `/readyz` health-check aggregation.
- Pyroscope continuous-profiling bootstrap.
