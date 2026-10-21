# ADR 0003: ConnectRPC + Vanguard + HTTP/3 for the public API

**Status:** Accepted
**Date:** 2026-10-20 (backfilled 2026-10-21)
**Deciders:** principal-engineer (stack locked with human sign-off in ARCHITECTURE.md v0.3)
**Related:** SMM-3, SMM-16, ARCHITECTURE.md §14, §15

## Context

The platform promises one API surface reachable three ways: REST for the long
tail of integrators, gRPC for high-throughput server-to-server senders, and
HTTP/3 for mobile clients on lossy mobile networks (§15). Hand-writing and
keeping three implementations consistent is exactly the kind of drift we
cannot afford in a billing-relevant send path.

Options considered:

1. **grpc-go + grpc-gateway** — the classic pairing. Needs a separate
   reverse-proxy process or in-process gateway for REST, no browser support
   without grpc-web, and gRPC's HTTP/2-only transport rules out HTTP/3.
2. **REST-first (chi/echo) with a hand-written gRPC facade** — duplicated
   validation and authz per transport; the contract drifts from the code.
3. **ConnectRPC (connect-go) + Vanguard + quic-go** — a connect-go handler is
   an ordinary `http.Handler`, so it serves gRPC, gRPC-Web and the Connect
   protocol over HTTP/1.1, HTTP/2 and HTTP/3 from one implementation.
   Vanguard transcodes REST from `google.api.http` annotations in the same
   process, and quic-go's `http3.Server` mounts the same handler on UDP.

## Decision

Per ARCHITECTURE.md §14 and §15:

- **Protobuf is the source of truth** for the API contract, managed with
  **Buf** (`buf lint` and `buf breaking` in CI); REST/OpenAPI is generated
  from proto, never hand-written.
- **ConnectRPC (connect-go)** is the RPC framework: one handler serves
  **gRPC, gRPC-Web and Connect** over HTTP/1.1, HTTP/2 and **HTTP/3**.
- **REST** comes from **Vanguard** transcoding `google.api.http` annotations —
  not from a separate gateway process.
- **HTTP/3 / QUIC** uses **quic-go** (`http3.Server`): the same handlers are
  mounted on HTTP/2 (TCP) and HTTP/3 (UDP), with `Alt-Svc` advertising.

Consequently authentication, authorization, validation, error mapping (§12)
and observability are implemented **once**, as connect-go interceptors, and
apply to every transport automatically.

## Consequences

- API changes always start in `api/proto` (contract-first); adding a REST
  route means adding an annotation, not writing a handler.
- Clients that only speak plain gRPC still work, because connect-go speaks the
  gRPC wire protocol — but only over HTTP/2, so gRPC clients do not get QUIC.
- Two listeners to operate (TCP and UDP/443). Load balancers, Kubernetes
  Services and NetworkPolicies must pass UDP; HTTP/3 needs its own synthetic
  probes and metrics labelled by protocol.
- quic-go's HTTP/3 support is younger than `net/http`'s HTTP/2. HTTP/3 is an
  additive fast path: if it misbehaves, dropping the `Alt-Svc` header makes
  clients fall back to HTTP/2 with no contract change.
- The commit policy for the generated proto stubs is
  [ADR 0001](./0001-gen-directory-policy.md).
