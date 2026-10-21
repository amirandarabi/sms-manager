package observability

import (
	"context"
	"io"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// Mandatory structured-log field keys, per ARCHITECTURE.md §18.3
// ("Mandatory fields: trace_id, span_id, service, message_id, client_id,
// error_code"). Services pass MessageID/ClientID/ErrorCode as needed via
// the With* helpers below; trace_id/span_id are injected automatically
// from the context's active span, and service is fixed at logger
// construction time.
const (
	FieldTraceID   = "trace_id"
	FieldSpanID    = "span_id"
	FieldService   = "service"
	FieldMessageID = "message_id"
	FieldClientID  = "client_id"
	FieldErrorCode = "error_code"
)

// NewLogger builds the slog.Logger every service uses for structured JSON
// logging to stdout (ARCHITECTURE.md §18.3: "slog JSON to stdout → Loki").
// Every record gets a fixed "service" field, trace_id/span_id are
// auto-populated from the context's active span (via LogAttrsContext /
// log/slog's native context-aware methods), and every string attribute
// value is masked for PII (ARCHITECTURE.md §10) before being written.
//
// w is the output writer; pass os.Stdout in production code. Tests pass a
// bytes.Buffer to assert on the emitted JSON.
func NewLogger(w io.Writer, service string) *slog.Logger {
	base := slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
	})
	return slog.New(&maskingHandler{
		next:    &tracingHandler{next: base},
		service: service,
	})
}

// tracingHandler injects trace_id/span_id from the active OTel span in ctx
// (if any) into every record, so every log line can be correlated with the
// trace that produced it (ARCHITECTURE.md §18.3).
type tracingHandler struct {
	next slog.Handler
}

func (h *tracingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *tracingHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String(FieldTraceID, sc.TraceID().String()),
			slog.String(FieldSpanID, sc.SpanID().String()),
		)
	}
	return h.next.Handle(ctx, r)
}

func (h *tracingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &tracingHandler{next: h.next.WithAttrs(attrs)}
}

func (h *tracingHandler) WithGroup(name string) slog.Handler {
	return &tracingHandler{next: h.next.WithGroup(name)}
}

// maskingHandler adds the fixed "service" field and masks PII (Iranian
// MSISDNs, ARCHITECTURE.md §10) in every string-valued attribute, in the
// log message itself, and in grouped attributes, before delegating to the
// wrapped handler.
type maskingHandler struct {
	next    slog.Handler
	service string
	groups  []string
}

func (h *maskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *maskingHandler) Handle(ctx context.Context, r slog.Record) error {
	masked := slog.NewRecord(r.Time, r.Level, MaskMSISDN(r.Message), r.PC)
	masked.AddAttrs(slog.String(FieldService, h.service))
	r.Attrs(func(a slog.Attr) bool {
		masked.AddAttrs(maskAttr(a))
		return true
	})
	return h.next.Handle(ctx, masked)
}

func (h *maskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	masked := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		masked[i] = maskAttr(a)
	}
	return &maskingHandler{
		next:    h.next.WithAttrs(masked),
		service: h.service,
		groups:  h.groups,
	}
}

func (h *maskingHandler) WithGroup(name string) slog.Handler {
	return &maskingHandler{
		next:    h.next.WithGroup(name),
		service: h.service,
		groups:  append(append([]string{}, h.groups...), name),
	}
}

// maskAttr returns a copy of a with any string value (including nested
// group members) PII-masked.
func maskAttr(a slog.Attr) slog.Attr {
	a.Value = a.Value.Resolve()
	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(a.Key, MaskMSISDN(a.Value.String()))
	case slog.KindGroup:
		members := a.Value.Group()
		masked := make([]slog.Attr, len(members))
		for i, m := range members {
			masked[i] = maskAttr(m)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(masked...)}
	default:
		return a
	}
}

// WithMessageID attaches the §18.3 mandatory message_id field to a logger
// for the scope of a request/message.
func WithMessageID(l *slog.Logger, messageID string) *slog.Logger {
	return l.With(slog.String(FieldMessageID, messageID))
}

// WithClientID attaches the §18.3 mandatory client_id field to a logger
// for the scope of a request/message.
func WithClientID(l *slog.Logger, clientID string) *slog.Logger {
	return l.With(slog.String(FieldClientID, clientID))
}

// WithErrorCode attaches the §18.3 mandatory error_code field to a logger
// for the scope of a request/message.
func WithErrorCode(l *slog.Logger, errorCode string) *slog.Logger {
	return l.With(slog.String(FieldErrorCode, errorCode))
}
