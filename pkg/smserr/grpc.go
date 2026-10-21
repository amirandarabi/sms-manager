package smserr

import "strconv"

// grpcErrorDomain is the "domain" field every ErrorInfo carries, so clients
// can disambiguate our codes from another service's error namespace.
const grpcErrorDomain = "sms.internal"

// ErrorInfo mirrors the fields of google.rpc.ErrorInfo — the standard gRPC
// "rich error model" detail message (reason, domain, metadata) — without
// pulling in the grpc/genproto dependency. ARCHITECTURE.md §15 requires
// every transport to map errors to the §12 catalog; gRPC services attach a
// real google.rpc.ErrorInfo detail to their status built from these fields
// once that dependency is introduced (Phase 1 API wiring), so the mapping
// logic can be written and tested here first.
type ErrorInfo struct {
	// Reason is the machine-readable SMS-xxxx code, e.g. "SMS-9001".
	Reason string
	// Domain identifies this service's error namespace.
	Domain string
	// Metadata carries class/retryable/trace_id/message_id as strings,
	// matching google.rpc.ErrorInfo.metadata's map[string]string shape.
	Metadata map[string]string
}

// GRPCErrorInfo builds the ErrorInfo detail for e, per ARCHITECTURE.md §15.
func (e *Error) GRPCErrorInfo() *ErrorInfo {
	metadata := map[string]string{
		"class":     string(e.Class),
		"retryable": strconv.FormatBool(e.Retryable),
	}
	if e.TraceID != "" {
		metadata["trace_id"] = e.TraceID
	}
	if e.MessageID != "" {
		metadata["message_id"] = e.MessageID
	}
	return &ErrorInfo{
		Reason:   e.Code,
		Domain:   grpcErrorDomain,
		Metadata: metadata,
	}
}
