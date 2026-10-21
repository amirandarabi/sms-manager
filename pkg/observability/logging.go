package observability

import (
	"context"
	"fmt"
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
// auto-populated from the context's active span (via the *Context slog
// methods), and every attribute value — including nested groups, errors,
// structs and fmt.Stringers, not just plain strings — is masked for PII
// (ARCHITECTURE.md §10) before being written.
//
// trace_id/span_id/service are always emitted at the top level of the
// JSON object, per §18.3, even when the caller has nested other fields
// under logger.WithGroup(...) — see logHandler below for why that needs
// its own bookkeeping instead of delegating to slog.JSONHandler's native
// WithGroup.
//
// w is the output writer; pass os.Stdout in production code. Tests pass a
// bytes.Buffer to assert on the emitted JSON.
func NewLogger(w io.Writer, service string) *slog.Logger {
	// base is used strictly flat: logHandler never calls base.WithGroup,
	// so base's own group machinery is never engaged. Group nesting asked
	// for via (*slog.Logger).WithGroup is instead implemented by
	// logHandler itself, by wrapping the relevant attrs in an explicit
	// slog.Group attribute value at Handle time — which base still
	// renders correctly nested, since that's just an ordinary attribute
	// of Kind Group, not handler-level grouping state.
	base := slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
	})
	return slog.New(&logHandler{next: base, service: service})
}

// withAttrsEntry records one (*slog.Logger).WithAttrs(...) call together
// with the full group path that was open when it was made, so Handle can
// later nest it correctly. groupDepth is len(groups) at call time, i.e. an
// index into the handler's groups slice: depth 0 means "top level, no
// group open yet".
type withAttrsEntry struct {
	groupDepth int
	attrs      []slog.Attr
}

// logHandler is the single slog.Handler backing every logger from
// NewLogger. It combines what used to be two separate handlers (one for
// trace context, one for masking) because both need to see the *final*,
// fully-assembled attribute set for a record, and because mandatory-field
// placement (the fix for the WithGroup bug below) requires owning the
// group stack directly rather than delegating it to the wrapped handler.
//
// Group handling: standard slog.Handler.WithGroup is normally implemented
// by telling the *wrapped* handler "nest everything from here on", so
// that handler's own group state controls where every subsequently added
// attribute — including ones a middleware layer injects itself, like our
// mandatory fields — ends up in the output. That means if the mandatory
// fields are injected via the wrapped handler after a group has been
// opened, they get nested under that group too, which breaks §18.3 (Loki
// queries expect trace_id/span_id/service at the top level always).
//
// So logHandler tracks `groups` itself and never calls next.WithGroup: at
// Handle time it rebuilds the correct nested shape by hand (via
// nestAttrs) for everything that came from the caller, and adds the
// mandatory fields as separate, always-top-level attrs on the flat
// Record it hands to `next`.
type logHandler struct {
	next    slog.Handler
	service string

	groups    []string         // open group names, outermost first
	withAttrs []withAttrsEntry // accumulated WithAttrs calls, in call order
}

func (h *logHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *logHandler) Handle(ctx context.Context, r slog.Record) error {
	// The record's own call-site attrs ("logger.Info(msg, k, v, ...)")
	// are logically one more WithAttrs entry at the *current* group
	// depth, added last.
	var callSiteAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		callSiteAttrs = append(callSiteAttrs, maskAttr(a))
		return true
	})
	entries := h.withAttrs
	if len(callSiteAttrs) > 0 {
		entries = append(entries, withAttrsEntry{groupDepth: len(h.groups), attrs: callSiteAttrs})
	}

	nested := nestAttrs(h.groups, entries)

	out := slog.NewRecord(r.Time, r.Level, MaskMSISDN(r.Message), r.PC)
	// Mandatory fields always go on the flat, ungrouped record — never
	// inside `nested` — so they stay top-level no matter what groups the
	// caller opened.
	out.AddAttrs(slog.String(FieldService, h.service))
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		out.AddAttrs(
			slog.String(FieldTraceID, sc.TraceID().String()),
			slog.String(FieldSpanID, sc.SpanID().String()),
		)
	}
	out.AddAttrs(nested...)

	return h.next.Handle(ctx, out)
}

func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	masked := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		masked[i] = maskAttr(a)
	}
	return &logHandler{
		next:      h.next,
		service:   h.service,
		groups:    h.groups,
		withAttrs: append(append([]withAttrsEntry{}, h.withAttrs...), withAttrsEntry{groupDepth: len(h.groups), attrs: masked}),
	}
}

func (h *logHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	// Deliberately NOT calling h.next.WithGroup(name): see the type
	// comment above. The underlying handler stays flat; nesting is
	// reproduced manually in Handle via nestAttrs.
	return &logHandler{
		next:      h.next,
		service:   h.service,
		groups:    append(append([]string{}, h.groups...), name),
		withAttrs: h.withAttrs,
	}
}

// nestAttrs rebuilds the nested slog.Group shape implied by groups and
// entries. Because slog.Handler.WithGroup can only ever push a new group
// (there is no "pop" — a shallower view is just a different, earlier
// handler value in the chain), entries' groupDepth values are
// non-decreasing in call order relative to a single handler chain, so a
// simple innermost-out fold is sufficient: start from the deepest depth
// present and wrap outward one group per level, attaching that level's
// own directly-added attrs alongside the wrapped child.
func nestAttrs(groups []string, entries []withAttrsEntry) []slog.Attr {
	if len(groups) == 0 {
		var flat []slog.Attr
		for _, e := range entries {
			flat = append(flat, e.attrs...)
		}
		return flat
	}

	byDepth := make([][]slog.Attr, len(groups)+1)
	for _, e := range entries {
		d := e.groupDepth
		if d < 0 || d > len(groups) {
			d = len(groups) // defensive clamp; should not happen
		}
		byDepth[d] = append(byDepth[d], e.attrs...)
	}

	cur := byDepth[len(groups)]
	for d := len(groups); d > 0; d-- {
		groupName := groups[d-1]
		level := append([]slog.Attr{}, byDepth[d-1]...)
		if len(cur) > 0 {
			level = append(level, slog.Attr{Key: groupName, Value: slog.GroupValue(cur...)})
		}
		cur = level
	}
	return cur
}

// maskAttr returns a copy of a with any PII masked, per ARCHITECTURE.md
// §10. It handles every slog.Kind, not just strings:
//
//   - KindString: masked directly.
//   - KindGroup: every member is masked recursively (so attrs nested via
//     slog.Group(...) are covered too).
//   - everything else (KindAny — errors, structs, fmt.Stringers, slices,
//     maps, pointers — as well as the numeric/bool/time/duration kinds):
//     rendered with fmt.Sprintf("%v", ...) and masked as a string. This is
//     intentionally lossy (e.g. an error becomes a masked string instead
//     of keeping its original type) because the alternative — walking
//     arbitrary struct fields via reflection — is unbounded in scope for
//     this story, and a PII leak (e.g. a provider adapter's error message
//     echoing the recipient MSISDN via `slog.Any("err", err)`) is worse
//     than losing the attr's original Go type in the log. Values that
//     clearly cannot contain PII (bool, numeric, time.Duration) still go
//     through this path for simplicity; it's a no-op for them in
//     practice since MaskMSISDN only rewrites digit runs shaped like an
//     MSISDN.
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
		return slog.String(a.Key, MaskMSISDN(fmt.Sprintf("%v", a.Value.Any())))
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
