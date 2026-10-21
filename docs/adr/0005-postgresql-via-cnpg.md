# ADR 0005: PostgreSQL (via CloudNativePG) as the system of record, with pgx/sqlc/goose

**Status:** Accepted
**Date:** 2026-10-20 (backfilled 2026-10-21)
**Deciders:** principal-engineer (stack locked with human sign-off in ARCHITECTURE.md v0.3)
**Related:** SMM-16, ARCHITECTURE.md §7, §14, §18.1

## Context

The system of record holds clients, routes, providers, messages, attempts and
audit entries (§7). It needs transactional integrity on the send path (an
outbox row must commit together with the message row), high-volume
append-heavy tables (`messages`, `attempts`) that must stay prunable, and rich
querying for the admin panel's reporting views. Short-lived state —
idempotency keys, OTP codes, rate limits, provider health, JWKS — is
explicitly not this store's job and lives in Redis.

Options considered:

1. **PostgreSQL** — transactions, native partitioning, JSONB for provider
   payloads, mature Go tooling, and a good Kubernetes operator story.
2. **MySQL** — workable, but weaker partitioning ergonomics and no equivalent
   of Postgres' JSONB and extension ecosystem for our reporting needs.
3. **A managed cloud RDBMS** — less operational work, but the target
   deployment is self-hosted Kubernetes (§14 "Deploy"), so the operator route
   keeps dev, staging and prod identical and GitOps-managed.
4. **A time-series or wide-column store for `messages`** — premature; monthly
   partitions in Postgres cover the expected volume, and splitting stores
   would cost us transactional writes alongside the outbox.

## Decision

Per ARCHITECTURE.md §14:

- **PostgreSQL**, operated by the **CloudNativePG (CNPG)** operator, is the
  relational system of record.
- `messages` and `attempts` use **monthly partitions**.
- Access uses the **pgx** driver with **sqlc** for type-safe queries;
  migrations use **goose** (or Atlas).
- **Redis** (or Valkey) — not Postgres — holds idempotency keys, OTP state,
  rate limits, provider health and the JWKS cache.

## Consequences

- No hand-rolled SQL string building in service code: queries live in SQL
  files and are compiled to Go by sqlc, so schema and code cannot silently
  disagree. sqlc output follows [ADR 0001](./0001-gen-directory-policy.md).
- Migrations are forward-only in practice; destructive migrations (dropping or
  renaming a column or table) are a human gate per CLAUDE.md.
- Partition creation and retention must be automated and monitored before
  `messages` volume ramps up — a missing future partition is an outage.
- CNPG owns backups, PITR and failover; restore drills belong in the SRE
  runbooks rather than ad-hoc `pg_dump` scripts.
- Integration tests run against real Postgres via testcontainers, so DB
  behaviour (constraints, partitions) is covered rather than mocked.
