package smserr

import "fmt"

// typeBaseURL is the stable namespace for the "type" field of the
// problem+json document, per ARCHITECTURE.md §12.1
// (https://sms.internal/errors/SMS-xxxx).
const typeBaseURL = "https://sms.internal/errors/"

// ProblemJSON is the RFC 9457 `application/problem+json` representation of
// an Error, matching the shape in ARCHITECTURE.md §12.1 exactly:
//
//	{
//	  "type": "https://sms.internal/errors/SMS-4102",
//	  "title": "Insufficient provider credit",
//	  "status": 502,
//	  "code": "SMS-4102",
//	  "class": "PROVIDER_CONFIG",
//	  "retryable": true,
//	  "message_id": "01JB...",
//	  "detail": "All eligible providers failed; last: kavenegar",
//	  "provider": {...},
//	  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
//	}
//
// Field order below matches the architecture doc's example for readability;
// JSON object key order is not semantically significant.
type ProblemJSON struct {
	Type      string        `json:"type"`
	Title     string        `json:"title"`
	Status    int           `json:"status"`
	Code      string        `json:"code"`
	Class     Class         `json:"class"`
	Retryable bool          `json:"retryable"`
	MessageID string        `json:"message_id,omitempty"`
	Detail    string        `json:"detail,omitempty"`
	Provider  *ProviderInfo `json:"provider,omitempty"`
	TraceID   string        `json:"trace_id,omitempty"`
}

// ProblemJSON renders e as the RFC 9457 document described above.
// includeProvider must only be true for responses visible to the
// sms.read.all role or for log sinks (ARCHITECTURE.md §12.1) — normal
// clients must never receive raw provider fields.
func (e *Error) ProblemJSON(includeProvider bool) *ProblemJSON {
	p := &ProblemJSON{
		Type:      typeBaseURL + e.Code,
		Title:     e.Title,
		Status:    e.Status,
		Code:      e.Code,
		Class:     e.Class,
		Retryable: e.Retryable,
		MessageID: e.MessageID,
		Detail:    e.Detail,
		TraceID:   e.TraceID,
	}
	if includeProvider {
		p.Provider = e.Provider
	}
	return p
}

// TypeURL returns the "type" URI for code without requiring an Error value,
// useful for documentation/tooling that only has the code string.
func TypeURL(code string) string {
	return fmt.Sprintf("%s%s", typeBaseURL, code)
}
