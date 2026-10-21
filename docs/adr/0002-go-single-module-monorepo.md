# ADR 0002: Go, in a single-module monorepo

**Status:** Accepted
**Date:** 2026-10-20 (backfilled 2026-10-21)
**Deciders:** principal-engineer (stack locked with human sign-off in ARCHITECTURE.md v0.3)
**Related:** SMM-2, SMM-16, ARCHITECTURE.md §14, §18.1

## Context

The platform is a set of cooperating services (gateway, dispatcher, DLR
ingester, callback dispatcher, scheduler, admin-api, content-guard) that share
a large amount of code: the error model (`pkg/smserr`), observability wiring
(`pkg/observability`), provider adapters, generated proto stubs and DB access.
They are developed by one (agent) team, released together, and deployed to the
same Kubernetes cluster.

Options considered:

1. **Go, one module, one repo** — all services under `cmd/<service>`, shared
   code in `pkg/*`, one `go.mod`. Shared code changes atomically with its
   callers; one `go build ./...`, one lint config, one CI matrix.
2. **Go, multi-module monorepo** — a `go.mod` per service. Buys independent
   dependency sets at the cost of `replace` directives or version tagging for
   every shared-package change, which is pure friction at this team size.
3. **Polyrepo** — maximum independence, but cross-cutting changes (a new
   `SMS-xxxx` error code, a proto change) become multi-PR coordination.
4. **Other languages** — rejected: Go's concurrency model, static binaries,
   first-class ConnectRPC/franz-go/pgx support and small container images fit
   a high-throughput SMS dispatcher better than the alternatives available to
   this team.

## Decision

Per ARCHITECTURE.md §14: services are written in **Go** (latest stable), with
a **single Go module in the monorepo**, using `log/slog`, `net/http` and
`context` everywhere.

Rules that follow:

- One `go.mod` at the repo root. Service entrypoints live in `cmd/<service>`;
  shared code in `pkg/*` and `internal/*`.
- `context.Context` is the first parameter of any call that does I/O; errors
  are wrapped with `%w`; no global mutable state.
- Structured logging via `log/slog` through `pkg/observability` — never
  `fmt.Println` or a third-party logger.
- The repository layout in §18.1 is normative; new top-level directories need
  an ADR.

## Consequences

- Cross-cutting changes (error codes, proto contracts, observability helpers)
  land in one atomic PR and are built and tested against every consumer.
- All services share one dependency graph: a dependency bump affects
  everything at once. That is a feature for CVE response and a constraint on
  adding niche dependencies — hence the "new deps need an ADR" rule.
- CI must build affected packages only (§18.5), so the single module does not
  make every PR pay for a full-repo build.
- Not a one-way door: splitting out a module later is mechanical. Revisit if
  an independent release cadence or a conflicting dependency set is needed.
