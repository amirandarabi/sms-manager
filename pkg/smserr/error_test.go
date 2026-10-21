package smserr

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestLookup_CodeToClassMapping(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantClass Class
		wantHTTP  int
		wantRetry bool
	}{
		{"auth missing token", "SMS-1001", ClassClient, 401, false},
		{"auth insufficient role", "SMS-1002", ClassClient, 403, false},
		{"auth service unavailable", "SMS-1003", ClassRetryable, 503, true},
		{"internal error", "SMS-9001", ClassRetryable, 500, true},
		{"dependency unavailable", "SMS-9002", ClassRetryable, 503, true},
		{"all providers unavailable", "SMS-9003", ClassRetryable, 502, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := Lookup(tt.code)
			if err != nil {
				t.Fatalf("Lookup(%q) returned unexpected error: %v", tt.code, err)
			}
			if entry.Class != tt.wantClass {
				t.Errorf("Class = %q, want %q", entry.Class, tt.wantClass)
			}
			if entry.HTTPStatus != tt.wantHTTP {
				t.Errorf("HTTPStatus = %d, want %d", entry.HTTPStatus, tt.wantHTTP)
			}
			if entry.Retryable != tt.wantRetry {
				t.Errorf("Retryable = %v, want %v", entry.Retryable, tt.wantRetry)
			}
			if entry.Title == "" {
				t.Error("Title is empty")
			}
		})
	}
}

func TestLookup_UnknownCodeIsLoud(t *testing.T) {
	_, err := Lookup("SMS-4999")
	if err == nil {
		t.Fatal("Lookup(unmapped code) = nil error, want an error wrapping ErrUnknownCode")
	}
	if !errors.Is(err, ErrUnknownCode) {
		t.Errorf("err = %v, want it to wrap ErrUnknownCode", err)
	}
}

func TestNew_UnknownCodeIsLoud(t *testing.T) {
	_, err := New("SMS-0000", "not a real code")
	if err == nil {
		t.Fatal("New(unknown code) = nil error, want an error")
	}
	if !errors.Is(err, ErrUnknownCode) {
		t.Errorf("err = %v, want it to wrap ErrUnknownCode", err)
	}
}

func TestNew_KnownCode(t *testing.T) {
	e, err := New("SMS-9001", "db write failed")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if e.Code != "SMS-9001" {
		t.Errorf("Code = %q, want SMS-9001", e.Code)
	}
	if e.Class != ClassRetryable {
		t.Errorf("Class = %q, want RETRYABLE", e.Class)
	}
	if !e.Retryable {
		t.Error("Retryable = false, want true")
	}
	if e.Detail != "db write failed" {
		t.Errorf("Detail = %q, want %q", e.Detail, "db write failed")
	}
}

// TestProblemJSON_MatchesArchitectureExampleShape builds an Error for
// SMS-9001 and checks the marshaled problem+json document carries exactly
// the fields (and nothing silently missing) called out in
// ARCHITECTURE.md §12.1: type, title, status, code, class, retryable,
// detail, trace_id.
func TestProblemJSON_MatchesArchitectureExampleShape(t *testing.T) {
	e, err := New("SMS-9001", "all eligible providers failed; last: kavenegar")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	e = e.WithTraceID("4bf92f3577b34da6a3ce929d0e0e4736")

	raw, err := json.Marshal(e.ProblemJSON(false))
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	wantFields := map[string]interface{}{
		"type":      "https://sms.internal/errors/SMS-9001",
		"title":     "Internal error",
		"status":    float64(500),
		"code":      "SMS-9001",
		"class":     "RETRYABLE",
		"retryable": true,
		"detail":    "all eligible providers failed; last: kavenegar",
		"trace_id":  "4bf92f3577b34da6a3ce929d0e0e4736",
	}
	for field, want := range wantFields {
		got, ok := got[field]
		if !ok {
			t.Errorf("field %q missing from problem+json output", field)
			continue
		}
		if got != want {
			t.Errorf("field %q = %v, want %v", field, got, want)
		}
	}

	// Provider details must not leak when includeProvider is false, even
	// if the Error carries them.
	if _, present := got["provider"]; present {
		t.Error(`"provider" present in output despite includeProvider=false`)
	}
}

func TestProblemJSON_ProviderVisibility(t *testing.T) {
	e, err := New("SMS-9003", "no healthy providers")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	e = e.WithProvider("kavenegar", "418", "insufficient credit")

	privileged := e.ProblemJSON(true)
	if privileged.Provider == nil {
		t.Fatal("ProblemJSON(true).Provider = nil, want populated")
	}
	if privileged.Provider.RawCode != "418" {
		t.Errorf("Provider.RawCode = %q, want 418", privileged.Provider.RawCode)
	}

	public := e.ProblemJSON(false)
	if public.Provider != nil {
		t.Error("ProblemJSON(false).Provider is populated, want nil (must not leak raw provider details)")
	}
}

func TestError_ImplementsErrorInterface(t *testing.T) {
	e, err := New("SMS-1001", "token expired")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	var _ error = e
	if e.Error() == "" {
		t.Error("Error() returned empty string")
	}
}

func TestGRPCErrorInfo(t *testing.T) {
	e, err := New("SMS-1002", "missing sms.send role")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	e = e.WithTraceID("trace-123").WithMessageID("msg-456")

	info := e.GRPCErrorInfo()
	if info.Reason != "SMS-1002" {
		t.Errorf("Reason = %q, want SMS-1002", info.Reason)
	}
	if info.Domain != "sms.internal" {
		t.Errorf("Domain = %q, want sms.internal", info.Domain)
	}
	if info.Metadata["class"] != string(ClassClient) {
		t.Errorf(`Metadata["class"] = %q, want %q`, info.Metadata["class"], ClassClient)
	}
	if info.Metadata["retryable"] != "false" {
		t.Errorf(`Metadata["retryable"] = %q, want "false"`, info.Metadata["retryable"])
	}
	if info.Metadata["trace_id"] != "trace-123" {
		t.Errorf(`Metadata["trace_id"] = %q, want "trace-123"`, info.Metadata["trace_id"])
	}
	if info.Metadata["message_id"] != "msg-456" {
		t.Errorf(`Metadata["message_id"] = %q, want "msg-456"`, info.Metadata["message_id"])
	}
}

func TestCatalog_NoDuplicateCodes(t *testing.T) {
	seen := make(map[string]bool)
	for _, e := range Catalog {
		if seen[e.Code] {
			t.Errorf("duplicate catalog entry for code %q", e.Code)
		}
		seen[e.Code] = true
	}
}

func TestTypeURL(t *testing.T) {
	got := TypeURL("SMS-9001")
	want := "https://sms.internal/errors/SMS-9001"
	if got != want {
		t.Errorf("TypeURL = %q, want %q", got, want)
	}
}
