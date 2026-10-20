# SMS Manager — Claude Code project guide

This repo is built by a **team of Claude agents** coordinated through a **public Jira board** and a **public GitHub repo**.
Read this file fully before doing anything. Detailed process lives in `docs/AGENT_WORKFLOW.md`, and the system design lives in `docs/ARCHITECTURE.md`.

## Project config (fill these in once)
```yaml
GITHUB_REPO:        amirandarabi/sms-manager     # public
DEFAULT_BRANCH:     main
JIRA_SITE:          andarabiamirhosein.atlassian.net
JIRA_PROJECT_KEY:   SMM                          # NOT "SMS": that prefix is reserved for our error codes (SMS-xxxx)
JIRA_BOARD_URL:     https://andarabiamirhosein.atlassian.net/jira/software/projects/SMM/boards/1
JIRA_CLOUD_ID:      606dd9e6-ff7f-4f73-b92b-8fda8e2864f0
HUMAN_GATES:        [release, destructive, security-settings]   # see "Safety rules"
```

## What we are building
A unified SMS platform: one API (REST, gRPC, Connect over HTTP/3) for sending SMS/OTP through multiple Iranian providers, with routing, failover, delivery tracking, a uniform error model, Keycloak auth, LLM content guard, generated SDKs, and a full admin panel.
**Source of truth: `docs/ARCHITECTURE.md`.** If a task conflicts with it, stop and ask the `principal-engineer` (or open an ADR) instead of improvising.

## Tech stack (decided, don't change without an ADR)
- Go (single module), ConnectRPC + Vanguard (REST) + quic-go (HTTP/3), Protobuf + Buf
- Kafka (Strimzi, franz-go), PostgreSQL (pgx, sqlc, goose), Redis
- Keycloak (OIDC), vLLM/Ollama for Content Guard
- React + TypeScript + Vite admin panel (pnpm, Connect-ES, shadcn/ui, RTL/Jalali)
- OpenTelemetry → Prometheus, Tempo, Loki, Pyroscope, Grafana
- Kubernetes, Helm, ArgoCD, KEDA. CI: **GitHub Actions**, images: **GHCR**

## Agent team
| Agent | Owns |
|---|---|
| `engineering-manager` | Orchestration. Run it as the main session: `claude --agent engineering-manager` |
| `product-owner` | Jira: epics, stories, acceptance criteria, priorities, status hygiene, sprint reports |
| `principal-engineer` | Architecture decisions, ADRs, **PR review gate**, cross-cutting design |
| `backend-engineer` | Go services, proto, providers, Kafka, DB, OTP, routing |
| `frontend-engineer` | Admin panel (`web/admin`) |
| `devops-engineer` | CI/CD, Docker, Helm, ArgoCD, Strimzi, CNPG, Keycloak-as-code, local dev stack |
| `sre-engineer` | Observability, SLOs, alerts, runbooks, dashboards, load and chaos tests |
| `qa-engineer` | Test strategy, e2e tests, acceptance verification, provider simulator |
| `security-engineer` | Threat model, authz review, secret hygiene, dependency/image scanning, security review of PRs |
| `ai-engineer` | Content Guard: prompts, LLM client, eval datasets, suggestion API |

Slash commands: `/kickoff`, `/sprint`, `/implement <KEY>`, `/review <PR#>`, `/board`.

## Golden workflow (every task)
1. **Jira first.** No code without a Jira issue in `SMM`. If none exists, ask `product-owner` to create one.
2. Transition the issue **In Progress** and comment the plan (template in `docs/AGENT_WORKFLOW.md`).
3. Branch: `<type>/SMM-<n>-<short-slug>` (e.g. `feat/SMM-42-kavenegar-adapter`).
4. Commits: Conventional Commits with scope + key: `feat(providers/kavenegar): add send adapter (SMM-42)`.
5. Run `task lint test` (or the equivalent for the area) **before** pushing. Never push red.
6. Open a PR with `gh pr create`. Title `feat(scope): summary (SMM-42)`, body from `.github/pull_request_template.md`. Comment the PR link on Jira and transition the issue to **In Review**.
7. `principal-engineer` reviews (plus `security-engineer` if the PR touches auth, secrets, crypto, public endpoints or CI). The verdict is posted on the PR and in Jira.
8. Changes requested → back to **In Progress**. Approved + CI green → `qa-engineer` verifies the acceptance criteria → **QA**.
9. `engineering-manager` squash-merges, transitions to **Done**, and comments a summary.

**Status must always reflect reality.** If you stop, are blocked, or fail, transition and comment immediately.

## Engineering rules
- **Contract first:** API changes start in `api/proto`. Run `buf lint` and `buf breaking`, then regenerate (`task gen`).
- **Errors:** only return `pkg/smserr` errors with catalog codes. New code → update the catalog + docs in the same PR.
- **Observability is part of done:** new endpoint/consumer = RED metrics + spans + structured logs (`pkg/observability`). No PII in labels or logs.
- **Tests:** unit tests for logic, table-driven tests for provider mappings, integration tests with testcontainers for DB/Kafka. Coverage must not drop.
- **Small PRs:** one story = one PR, ideally under 400 changed lines (generated code excluded). Split otherwise.
- **Docs:** update `docs/` and component README when behavior changes. Record significant decisions as ADRs (`docs/adr/NNNN-title.md`).
- **Changelog:** do not edit CHANGELOG files by hand. release-please generates them from commits.
- Go style: `gofmt`, `golangci-lint`, `context` first param, no global state, errors wrapped with `%w`.
- Frontend style: TypeScript strict, ESLint + Prettier, components tested with Vitest + Testing Library, RTL-safe CSS (logical properties).

## Safety rules (public repo + public Jira)
- **Never commit or post secrets.** No API keys, tokens, passwords, kubeconfigs, or `.env` files in the repo, PRs, Jira, or logs. Use `.env` (gitignored) locally and GitHub Actions secrets in CI. If you find a leaked secret, stop and tell the human.
- **Public content only.** No real phone numbers (use the test range `0912000xxxx`), no customer data, no internal company hostnames/IPs, no non-public provider documents in Jira or the repo.
- **Human confirmation required** before:
  - force-push, history rewrite, or deleting branches/tags other than your own merged branch
  - creating releases or tags
  - changing repo settings, branch protection, GitHub Actions secrets, or Jira workflow/project settings
  - destructive DB migrations (drop/rename column/table) and anything that deletes data
  - adding new external services or paid dependencies
- Prefer `--dry-run` flags (helm, kubectl, terraform plan) where available.
- Never disable tests, linters, or security scans to make CI pass.
- Never bypass review: no direct pushes to `main`.

## Useful commands
```bash
task dev          # local stack (kafka, postgres, redis, keycloak, ollama, providersim, grafana)
task gen          # buf generate, sqlc, mocks, error catalog doc
task lint test    # all linters + unit tests (affected packages)
task e2e          # end-to-end against local stack
gh pr create / gh pr view / gh pr checks / gh pr diff
```
(Until Phase 0 creates the Taskfile, use the raw `go`, `buf`, and `pnpm` commands.)

## Repository map
See `docs/ARCHITECTURE.md` §18.1. Key dirs: `api/proto`, `cmd/<service>`, `pkg/*`, `migrations`, `web/admin`, `sdks`, `deploy`, `test`, `docs`.
