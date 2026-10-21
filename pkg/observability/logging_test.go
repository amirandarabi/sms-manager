package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestNewLogger_MasksMSISDNInMessageAndAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "sms-api")

	logger.InfoContext(context.Background(), "sending to 09120001234",
		"recipient", "09120001234",
		"message_id", "01JB0EXAMPLE",
	)

	out := buf.String()
	if strings.Contains(out, "09120001234") {
		t.Fatalf("expected MSISDN to be masked out of log line, got: %s", out)
	}
	if !strings.Contains(out, "0912***1234") {
		t.Fatalf("expected masked MSISDN 0912***1234 in log line, got: %s", out)
	}

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, out)
	}

	if line["service"] != "sms-api" {
		t.Errorf("expected service=sms-api field, got %v", line["service"])
	}
	if line["recipient"] != "0912***1234" {
		t.Errorf("expected recipient attr masked, got %v", line["recipient"])
	}
	if line["msg"] != "sending to 0912***1234" {
		t.Errorf("expected message masked, got %v", line["msg"])
	}
}

func TestWithHelpers_AttachMandatoryFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "otp-service")
	logger = WithMessageID(logger, "01JB0MSG")
	logger = WithClientID(logger, "client-42")
	logger = WithErrorCode(logger, "SMS-4102")

	logger.InfoContext(context.Background(), "delivery failed")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	for key, want := range map[string]string{
		FieldMessageID: "01JB0MSG",
		FieldClientID:  "client-42",
		FieldErrorCode: "SMS-4102",
		FieldService:   "otp-service",
	} {
		if line[key] != want {
			t.Errorf("field %s = %v, want %s", key, line[key], want)
		}
	}
}

func TestNewLogger_NeverPanicsOnNilLikeInput(_ *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "svc")
	// Group attrs and non-string attrs must not panic the masking handler.
	logger.InfoContext(context.Background(), "ok",
		"count", 3,
		"meta", "group-ish-09120001234",
	)
}

// testSpanContext builds a valid, deterministic OTel SpanContext so tests
// can assert on exact trace_id/span_id values without standing up a real
// tracer provider.
func testSpanContext(t *testing.T) trace.SpanContext {
	t.Helper()
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("TraceIDFromHex: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatalf("SpanIDFromHex: %v", err)
	}
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})
}

// TestNewLogger_InjectsTraceAndSpanIDFromContext covers the §18.3
// requirement that every log line carry trace_id/span_id from the active
// span, and that they land at the top level of the JSON object.
func TestNewLogger_InjectsTraceAndSpanIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "sms-api")

	sc := testSpanContext(t)
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	logger.InfoContext(ctx, "message accepted")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	if line[FieldTraceID] != sc.TraceID().String() {
		t.Errorf("trace_id = %v, want %s", line[FieldTraceID], sc.TraceID().String())
	}
	if line[FieldSpanID] != sc.SpanID().String() {
		t.Errorf("span_id = %v, want %s", line[FieldSpanID], sc.SpanID().String())
	}
}

// TestNewLogger_NoSpanContext_OmitsTraceFields ensures we don't fabricate
// trace_id/span_id when there is no active span — logging without a
// context-propagated span is common in Phase 0 (no services wired yet).
func TestNewLogger_NoSpanContext_OmitsTraceFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "sms-api")

	logger.InfoContext(context.Background(), "no span here")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}
	if _, ok := line[FieldTraceID]; ok {
		t.Errorf("expected no trace_id field without an active span, got %v", line[FieldTraceID])
	}
	if _, ok := line[FieldSpanID]; ok {
		t.Errorf("expected no span_id field without an active span, got %v", line[FieldSpanID])
	}
}

// TestNewLogger_WithGroup_KeepsMandatoryFieldsTopLevel is the regression
// test for the bug where logger.WithGroup(...) nested service/trace_id/
// span_id under the group instead of keeping them top-level per §18.3.
func TestNewLogger_WithGroup_KeepsMandatoryFieldsTopLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "sms-api")

	sc := testSpanContext(t)
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	grouped := logger.WithGroup("provider")
	grouped.InfoContext(ctx, "dispatch attempt", "name", "kavenegar", "recipient", "09120001234")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	// Mandatory fields must stay top-level, not nested under "provider".
	if line[FieldService] != "sms-api" {
		t.Errorf("service = %v, want sms-api at top level", line[FieldService])
	}
	if line[FieldTraceID] != sc.TraceID().String() {
		t.Errorf("trace_id = %v, want %s at top level", line[FieldTraceID], sc.TraceID().String())
	}
	if line[FieldSpanID] != sc.SpanID().String() {
		t.Errorf("span_id = %v, want %s at top level", line[FieldSpanID], sc.SpanID().String())
	}

	// The grouped, call-site attrs must be nested under "provider" and
	// still be masked.
	group, ok := line["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected a nested \"provider\" group object, got: %v (full line: %v)", line["provider"], line)
	}
	if group["name"] != "kavenegar" {
		t.Errorf("provider.name = %v, want kavenegar", group["name"])
	}
	if group["recipient"] != "0912***1234" {
		t.Errorf("provider.recipient = %v, want masked 0912***1234", group["recipient"])
	}
	// And the group itself must NOT also appear re-nested at a stray
	// top-level "provider" key duplicated some other way, nor swallow the
	// mandatory fields — covered implicitly by the assertions above, but
	// double check the group has exactly the two attrs we logged.
	if len(group) != 2 {
		t.Errorf("expected exactly 2 attrs in the provider group, got %d: %v", len(group), group)
	}
}

// TestNewLogger_WithGroup_AndWithAttrs_NestsCorrectly exercises a logger
// built via With(...) *before* WithGroup(...), which must stay top-level,
// combined with attrs added *after* WithGroup(...), which must nest.
func TestNewLogger_WithGroup_AndWithAttrs_NestsCorrectly(t *testing.T) {
	var buf bytes.Buffer
	base := NewLogger(&buf, "sms-api")

	logger := base.With("request_id", "req-1").
		WithGroup("provider").
		With("name", "kavenegar")

	logger.Info("sent")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	if line["request_id"] != "req-1" {
		t.Errorf("request_id = %v, want req-1 at top level", line["request_id"])
	}
	group, ok := line["provider"].(map[string]any)
	if !ok {
		t.Fatalf("expected a nested \"provider\" group object, got: %v", line["provider"])
	}
	if group["name"] != "kavenegar" {
		t.Errorf("provider.name = %v, want kavenegar", group["name"])
	}
}

// stringerPhone is a fmt.Stringer whose String() echoes an MSISDN, the
// same shape a provider adapter's error type might take.
type stringerPhone struct{ msisdn string }

func (s stringerPhone) String() string { return "recipient=" + s.msisdn }

// TestMaskAttr_MasksNonStringKinds is the regression test for the PII
// leak where slog.Any (errors, structs, fmt.Stringers) passed through the
// masking handler unmasked. This is the dominant Go idiom for logging an
// error (`logger.Error("failed", "err", err)`), so it's the most
// important case to cover.
func TestMaskAttr_MasksNonStringKinds(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, "sms-api")

	providerErr := fmt.Errorf("provider rejected %s", "09120001234")
	type providerDetail struct {
		Recipient string
	}

	logger.Error("send failed",
		"err", providerErr,
		"detail", providerDetail{Recipient: "09120001234"},
		"stringer", stringerPhone{msisdn: "09120001234"},
	)

	out := buf.String()
	if strings.Contains(out, "09120001234") {
		t.Fatalf("expected MSISDN to be masked out of non-string attrs, got: %s", out)
	}
	if !strings.Contains(out, "0912***1234") {
		t.Fatalf("expected masked MSISDN 0912***1234 to appear, got: %s", out)
	}

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, out)
	}
	for _, key := range []string{"err", "detail", "stringer"} {
		s, ok := line[key].(string)
		if !ok {
			t.Fatalf("expected field %q to be a masked string, got %T: %v", key, line[key], line[key])
		}
		if !strings.Contains(s, "0912***1234") {
			t.Errorf("field %q = %q, want it to contain masked 0912***1234", key, s)
		}
	}
}
