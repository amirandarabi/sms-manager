# pkg/smserr

The unified error catalog (`SMS-xxxx` codes) and its gRPC/REST mapping (ARCHITECTURE.md §12). Every service returns errors from this package only — new codes must be added here with doc generation, never ad-hoc.

## Usage

```go
e, err := smserr.New("SMS-9001", "db write failed")
if err != nil {
    // code not in the catalog — this is a programmer error, make it loud
}
e = e.WithTraceID(traceID)

// REST: respond with RFC 9457 application/problem+json.
json.NewEncoder(w).Encode(e.ProblemJSON(includeProvider))

// gRPC: attach the rich-error-model detail.
info := e.GRPCErrorInfo()
```

- `smserr.Catalog` is the single source of truth for code → class/title/status/retryable. Add new codes there, sorted by code.
- `go generate ./pkg/smserr/...` regenerates `docs/error-catalog.md` from `Catalog` — run it after every catalog change and commit the diff.
- A code not present in `Catalog` is never silently defaulted: `Lookup`/`New` return an error wrapping `smserr.ErrUnknownCode` (ARCHITECTURE.md §12.3: gaps must be loud).
- `ProblemJSON(includeProvider)` only includes raw provider fields when `includeProvider` is true — reserve that for `sms.read.all` responses and log sinks, never default clients.

**Scope note (SMM-6):** this is a placeholder catalog (`SMS-1xxx` auth, `SMS-9xxx` internal) to prove the pattern. `SMS-2xxx`..`SMS-5xxx` and the per-provider mapping YAMLs (`pkg/providers/mappings/*.yaml`) land with each Phase 1 provider/API story.
