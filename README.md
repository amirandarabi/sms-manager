# SMS Manager

A unified SMS platform: one API (REST, gRPC, Connect over HTTP/3) for sending SMS/OTP through
multiple Iranian providers, with routing and failover, delivery tracking, a uniform error model,
Keycloak auth, an LLM content guard, generated client SDKs, and a full admin panel.

This repo is built by a team of Claude agents coordinated through a public Jira board
(project `SMM`) and this public GitHub repo. See:

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design, tech stack decisions, repo
  layout (§18.1), CI/CD pipeline, observability, scalability and reliability. The source of
  truth for how this system is built.
- [`docs/AGENT_WORKFLOW.md`](docs/AGENT_WORKFLOW.md) — how the agent team plans, tracks, builds,
  reviews and ships work: Jira lifecycle, GitHub conventions, comment templates, and the
  orchestration loop.
- [`docs/adr/README.md`](docs/adr/README.md) — the Architecture Decision Record index: why the
  stack in ARCHITECTURE.md §14 is what it is, plus the template every new ADR follows.
- [`CLAUDE.md`](CLAUDE.md) — project config, tech stack, agent roster, golden workflow, and
  safety rules.

## Status

This is Phase 0: the repository and Go module scaffold. Later stories fill in `api/proto`,
`pkg/observability`, `pkg/smserr`, the CI pipeline, local dev stack, Helm charts and GitOps
config — tracked on the [Jira board](https://andarabiamirhosein.atlassian.net/jira/software/projects/SMM/boards/1).

## Layout

See `docs/ARCHITECTURE.md` §18.1 for the full repository layout and the rationale behind it.
Each top-level directory has its own `README.md` explaining its purpose.

## Contributing

No code without a Jira issue in `SMM`. Read `CLAUDE.md` and `docs/AGENT_WORKFLOW.md` before
opening a PR.
