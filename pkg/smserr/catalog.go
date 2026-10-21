// Package smserr implements the uniform SMS-xxxx error catalog described in
// ARCHITECTURE.md §12. Every service and provider adapter must return errors
// built from this package only — never a raw provider or library error —
// so that clients can branch on a stable code/class across REST, gRPC, and
// Connect transports (§15).
package smserr

import "fmt"

//go:generate go run ./internal/gencatalog

// Class is the fixed failover category of an error code (ARCHITECTURE.md
// §12.2). It is what routing/failover logic branches on — never the code's
// text or the HTTP status alone.
type Class string

const (
	// ClassRetryable means the same request can be retried, typically
	// against a different provider or after a backoff.
	ClassRetryable Class = "RETRYABLE"
	// ClassPermanent means retrying will not help; the request itself is
	// unprocessable (e.g. a provider-confirmed invalid recipient).
	ClassPermanent Class = "PERMANENT"
	// ClassProviderConfig means the failure is specific to how this
	// provider is configured (credentials, sender ID, credit) — routing
	// should fail over to another provider rather than retry the same one.
	ClassProviderConfig Class = "PROVIDER_CONFIG"
	// ClassAmbiguous means the mapping could not determine retryability
	// with confidence (e.g. an unmapped provider code); treated
	// conservatively by callers.
	ClassAmbiguous Class = "AMBIGUOUS"
	// ClassClient means the caller must fix the request; retrying the
	// exact same request will never succeed.
	ClassClient Class = "CLIENT"
)

// CatalogEntry documents one SMS-xxxx code: its fixed class, human title,
// default HTTP status for REST problem+json responses, default
// retryability, and a short description. Entries are the single source of
// truth for both runtime behavior (Lookup) and the generated docs
// (docs/error-catalog.md, produced by `go generate ./pkg/smserr/...`).
type CatalogEntry struct {
	Code        string
	Class       Class
	Title       string
	HTTPStatus  int
	Retryable   bool
	Description string
}

// Catalog is the authoritative, ordered list of currently defined SMS-xxxx
// codes (ARCHITECTURE.md §12.2). This is a deliberately small placeholder
// covering only the SMS-1xxx (auth) and SMS-9xxx (internal) ranges, enough
// to prove the pattern; SMS-2xxx..SMS-5xxx fill in as Phase 1 provider and
// API stories land (SMM-6 scope — see Jira).
//
// Adding a code: append an entry here, keep it sorted by Code, then run
// `go generate ./pkg/smserr/...` to refresh docs/error-catalog.md.
var Catalog = []CatalogEntry{
	{
		Code:        "SMS-1001",
		Class:       ClassClient,
		Title:       "Missing or invalid token",
		HTTPStatus:  401,
		Retryable:   false,
		Description: "The request has no bearer token, or the token is malformed/expired/invalid.",
	},
	{
		Code:        "SMS-1002",
		Class:       ClassClient,
		Title:       "Insufficient role",
		HTTPStatus:  403,
		Retryable:   false,
		Description: "The caller is authenticated but lacks the role/scope required for this operation.",
	},
	{
		Code:        "SMS-1003",
		Class:       ClassRetryable,
		Title:       "Auth service unavailable",
		HTTPStatus:  503,
		Retryable:   true,
		Description: "Keycloak (or the token introspection path) could not be reached in time.",
	},
	{
		Code:        "SMS-9001",
		Class:       ClassRetryable,
		Title:       "Internal error",
		HTTPStatus:  500,
		Retryable:   true,
		Description: "An unexpected internal failure occurred; see trace_id for correlation.",
	},
	{
		Code:        "SMS-9002",
		Class:       ClassRetryable,
		Title:       "Dependency unavailable",
		HTTPStatus:  503,
		Retryable:   true,
		Description: "A required dependency (database, queue, cache) is unavailable or timed out.",
	},
	{
		Code:        "SMS-9003",
		Class:       ClassRetryable,
		Title:       "All providers unavailable",
		HTTPStatus:  502,
		Retryable:   true,
		Description: "Every eligible SMS provider failed or was unavailable for this send.",
	},
}

// catalogIndex is Catalog keyed by Code, built once from the slice above so
// Lookup is O(1) and duplicate codes are caught immediately at init time
// (a programmer error, not a runtime condition).
var catalogIndex = buildCatalogIndex(Catalog)

func buildCatalogIndex(entries []CatalogEntry) map[string]CatalogEntry {
	idx := make(map[string]CatalogEntry, len(entries))
	for _, e := range entries {
		if _, dup := idx[e.Code]; dup {
			panic(fmt.Sprintf("smserr: duplicate catalog entry for code %q", e.Code))
		}
		idx[e.Code] = e
	}
	return idx
}

// ErrUnknownCode is wrapped by Lookup (and New) when a code is not present
// in Catalog. Per ARCHITECTURE.md §12.3's mapping philosophy, catalog gaps
// must be loud — callers must never silently fall back to a default class.
var ErrUnknownCode = fmt.Errorf("smserr: unknown code")

// Lookup returns the CatalogEntry for code, or an error wrapping
// ErrUnknownCode if the code is not defined in Catalog.
func Lookup(code string) (CatalogEntry, error) {
	entry, ok := catalogIndex[code]
	if !ok {
		return CatalogEntry{}, fmt.Errorf("%w: %q is not a defined SMS-xxxx code — add it to pkg/smserr.Catalog", ErrUnknownCode, code)
	}
	return entry, nil
}
