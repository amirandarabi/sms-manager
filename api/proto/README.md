# api/proto

Protobuf contracts — the source of truth for every service's public API (REST via Vanguard, gRPC, Connect over HTTP/3).
Holds `buf.yaml`, `buf.gen.yaml`, and per-domain `.proto` files (e.g. `sms/v1`).

## Contract-first rule

Per `CLAUDE.md` and ARCHITECTURE.md §14/§15: **API changes start here.** No
service code is written against a new/changed RPC until the `.proto` change
has:

1. Passed `buf lint`.
2. Passed `buf breaking` against `main` (proves the change doesn't break
   existing consumers — a new story/ADR is required to make an intentional
   breaking change).
3. Been regenerated into `gen/` via `buf generate`, with the regenerated
   code committed in the same PR (see `docs/adr/0001-gen-directory-policy.md`
   — `gen/` is committed, not CI-only).

CI (SMM-8) enforces this automatically; until then, run the two commands
below manually before opening a PR that touches `api/proto/**`.

### Commands

Run from `api/proto/`. Note `.git` lives at the repo root, two levels up
from here, so the breaking-change reference needs `../../.git` plus
`subdir=api/proto` to point back at this module within that checkout:

```bash
# 1. Lint the contracts — must pass with zero warnings.
buf lint

# 2. Check for breaking changes against the main branch baseline.
buf breaking --against '../../.git#branch=main,subdir=api/proto'
```

Then regenerate the Go stubs:

```bash
buf generate
```

This writes Protobuf + ConnectRPC Go code into `gen/` (paths relative to
`api/proto`, mirrored under `gen/`). Verify it compiles with:

```bash
go build ./gen/...
```

`task gen`/`task lint` (SMM-7) will wire these up as repo-wide targets once
the Taskfile lands; the raw commands above work today without it.

## Current contents

- `sms/v1/health.proto` — a minimal `HealthService` placeholder proving the
  Buf lint/breaking/generate pipeline end to end (SMM-3). The full
  `MessageService` / `OtpService` / `ContentService` contracts from §15 land
  in later Phase 1a/1b stories.
