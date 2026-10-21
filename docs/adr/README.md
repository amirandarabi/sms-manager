# docs/adr

Architecture Decision Records — one Markdown file per significant technical decision (e.g. Kafka over RabbitMQ, ConnectRPC). New ADRs follow [`0000-adr-template.md`](./0000-adr-template.md) and require principal-engineer (and human, for stack changes) sign-off.

## Index

| # | Title | Decision in one line | Status |
|---|---|---|---|
| [0000](./0000-adr-template.md) | ADR template | Not a decision — the template and conventions every ADR in this directory follows. | Template |
| [0001](./0001-gen-directory-policy.md) | Commit generated code (`gen/`) to the repository | `gen/` (proto Go, sqlc) is committed, never hand-edited, and CI drift-checks it. | Accepted |
| [0002](./0002-go-single-module-monorepo.md) | Go, in a single-module monorepo | Services are Go, in one `go.mod` at the repo root, with shared code in `pkg/*`. | Accepted |
| [0003](./0003-connectrpc-vanguard-http3.md) | ConnectRPC + Vanguard + HTTP/3 for the public API | Proto is the contract; one connect-go handler serves gRPC/gRPC-Web/Connect, REST via Vanguard, HTTP/3 via quic-go. | Accepted |
| [0004](./0004-kafka-via-strimzi.md) | Kafka (via Strimzi, KRaft mode) as the message broker | Kafka on Strimzi with franz-go; priority lanes are separate topics, consumers idempotent. | Accepted |
| [0005](./0005-postgresql-via-cnpg.md) | PostgreSQL (via CloudNativePG) as the system of record | Postgres on CNPG with pgx/sqlc/goose, monthly partitions; Redis holds ephemeral state. | Accepted |

ADRs 0002–0005 are **backfills**: they record decisions already locked in
[`../ARCHITECTURE.md`](../ARCHITECTURE.md) §14 and explain the reasoning. ARCHITECTURE.md
remains the source of truth — if an ADR and §14 ever disagree, that is a bug in the ADR.

## Conventions

- Filename: `NNNN-kebab-case-title.md`, numbers assigned in order and never reused.
- Copy [`0000-adr-template.md`](./0000-adr-template.md), fill every section, add a row above.
- Accepted ADRs are immutable: supersede them with a new ADR rather than editing the decision.
- Decisions that change the tech stack in ARCHITECTURE.md §14 need explicit human sign-off.

## See also

- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) — system design and the decided stack (§14).
- [`../AGENT_WORKFLOW.md`](../AGENT_WORKFLOW.md) — how work is planned, reviewed and shipped.
- [`../security/threat-model.md`](../security/threat-model.md) — the project's threat model and security posture.
- [`../../SECURITY.md`](../../SECURITY.md) — the vulnerability reporting process.
