package smserr

import "fmt"

// ProviderInfo carries the raw, provider-specific error details behind a
// mapped SMS-xxxx code. Per ARCHITECTURE.md §12.1, these fields must only
// be surfaced to callers with the sms.read.all role and to logs — normal
// clients see the unified Code/Class/Title/Detail fields only. Callers
// control this by passing includeProvider=false to Error.ProblemJSON for
// any response that is not already privileged.
type ProviderInfo struct {
	Name       string `json:"name"`
	RawCode    string `json:"raw_code"`
	RawMessage string `json:"raw_message"`
}

// Error is the single error type every service, adapter, and handler in
// this repo must return (ARCHITECTURE.md §12). It is built from a
// catalog-defined code via New, never constructed with an arbitrary code.
type Error struct {
	Code      string
	Class     Class
	Title     string
	Status    int
	Retryable bool

	Detail    string
	MessageID string
	TraceID   string
	Provider  *ProviderInfo
}

// New builds an Error for code, filling Class/Title/Status/Retryable from
// the catalog. It returns an error (wrapping ErrUnknownCode) instead of a
// silent default if code is not in Catalog — see ARCHITECTURE.md §12.3.
func New(code string, detail string) (*Error, error) {
	entry, err := Lookup(code)
	if err != nil {
		return nil, err
	}
	return &Error{
		Code:      entry.Code,
		Class:     entry.Class,
		Title:     entry.Title,
		Status:    entry.HTTPStatus,
		Retryable: entry.Retryable,
		Detail:    detail,
	}, nil
}

// WithMessageID returns e with MessageID set, for chaining after New.
func (e *Error) WithMessageID(id string) *Error {
	e.MessageID = id
	return e
}

// WithTraceID returns e with TraceID set, for chaining after New.
func (e *Error) WithTraceID(id string) *Error {
	e.TraceID = id
	return e
}

// WithProvider attaches raw provider error details. These are rendered in
// ProblemJSON only when includeProvider is true (see ProviderInfo).
func (e *Error) WithProvider(name, rawCode, rawMessage string) *Error {
	e.Provider = &ProviderInfo{Name: name, RawCode: rawCode, RawMessage: rawMessage}
	return e
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Title, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Title)
}
