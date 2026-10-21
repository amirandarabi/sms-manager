# ADR 0004: Kafka (via Strimzi, KRaft mode) as the message broker, with franz-go

**Status:** Accepted
**Date:** 2026-10-20 (backfilled 2026-10-21)
**Deciders:** principal-engineer (stack locked with human sign-off in ARCHITECTURE.md v0.3)
**Related:** SMM-16, ARCHITECTURE.md §4, §14, §14.1

## Context

The send path is asynchronous: the API accepts a message, a dispatcher sends it
to a provider, retries on failure, and emits status events consumed by the
callback dispatcher, analytics and the admin live view (§4). Requirements:
priority lanes (OTP must never queue behind a bulk campaign), bounded retry
tiers, a DLQ with replay, high throughput with ordering per message, and
replayable raw provider webhooks for post-incident reprocessing.

Options considered:

1. **RabbitMQ** — real per-queue priorities and delayed exchanges, but
   messages are consumed destructively; replaying `sms.dlr.raw` after a
   normalization bug would be impossible without a separate store.
2. **Postgres-only queue (outbox + `SKIP LOCKED`)** — fewest moving parts, and
   we do use an outbox for the DB-to-broker handoff, but it does not give us
   fan-out of `sms.status.events` to several independent consumer groups, and
   that fan-out load would land on the primary database.
3. **Kafka** — a retained, replayable log; consumer groups give independent
   fan-out; partition-per-key gives per-message ordering. No per-message
   priority, which we solve with one topic per lane.

## Decision

Per ARCHITECTURE.md §14 and §14.1:

- **Apache Kafka**, operated by the **Strimzi** operator in **KRaft mode**, is
  the message broker. Event schemas are Protobuf in a schema registry
  (Apicurio or Confluent SR).
- The Kafka client is **franz-go** (pure Go, fast, good transaction and
  idempotent-producer support).
- Topic design follows the §14.1 table, including its rules:
  - Kafka has no per-message priority, so **lanes are separate topics with
    separate consumer groups** (`sms.send.otp`, `sms.send.transactional`,
    `sms.send.bulk`).
  - Scheduled messages wait in **PostgreSQL** and are released by the
    Scheduler. Kafka is not used for long delays; `sms.retry.5s` / `.30s` /
    `.5m` are short delayed-retry tiers where the consumer pauses until the
    `not_before` header.
  - Consumers are **idempotent**: `attempts` has a unique key on
    (message_id, attempt_no) and status updates are monotonic, so
    at-least-once delivery is safe.
  - Producers use `acks=all`, idempotence on, `min.insync.replicas=2`, RF=3.
  - Trace context is propagated in Kafka headers (W3C `traceparent`).

## Consequences

- Kafka plus its operator is real operational surface: Strimzi upgrades,
  rebalances, partition sizing, consumer-lag SLOs and alerts.
- Every consumer must be idempotent and keyed as specified in §14.1; reviews
  block on this. Changing a topic's key or partition count is a migration,
  not a config tweak.
- Topics are declared as Strimzi `KafkaTopic` resources under `deploy/`, so
  topic changes are GitOps changes and reviewable.
- `sms.dlr.raw` retention is the window in which a normalization bug can be
  fixed by replay — changing that retention is a reliability decision, not a
  cost tweak.
- Adding a lane is cheap; replacing Kafka itself would be a §14 stack change
  needing its own ADR and human sign-off.
