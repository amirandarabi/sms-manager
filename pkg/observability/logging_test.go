package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
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
