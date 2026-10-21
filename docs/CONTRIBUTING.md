# Contributing

See `CLAUDE.md` and `docs/AGENT_WORKFLOW.md` for the full golden workflow
(Jira → branch → commits → PR → review → QA → merge). This page only covers
local commit-message tooling.

## Conventional Commits

Every commit must follow [Conventional Commits](https://www.conventionalcommits.org/)
with a scope, matching ARCHITECTURE.md §18.2:

```
<type>(<scope>): <summary> (<JIRA-KEY>)

feat(providers/kavenegar): add send adapter (SMM-42)
fix(dispatcher): retry on provider timeout (SMM-51)
chore(ci): configure release-please and commitlint (SMM-9)
```

Allowed `type`s: `feat`, `fix`, `perf`, `revert`, `docs`, `build`, `chore`,
`refactor`, `test`, `ci`, `style` (see `commitlint.config.js`). The scope
should name the component you changed (`api`, `dispatcher`,
`providers/<name>`, `pkg/<name>`, `web/admin`, `deploy`, `docs`, `ci`,
`repo`, ...) so release-please can attribute the change to the right
per-component changelog.

This is enforced in CI by the `commitlint` GitHub Actions workflow
(`.github/workflows/commitlint.yml`) on every pull request — a commit that
violates the convention fails that check with the specific commitlint rule
that was violated.

## Local pre-commit hook (recommended)

Install commitlint once per clone and wire it into git's `commit-msg` hook
so violations are caught before you push, not just in CI:

```bash
# from the repo root
npm install --no-save @commitlint/cli@19 @commitlint/config-conventional@19

mkdir -p .git/hooks
cat > .git/hooks/commit-msg <<'EOF'
#!/usr/bin/env bash
npx --no -- commitlint --edit "$1"
EOF
chmod +x .git/hooks/commit-msg
```

If you use [Husky](https://typicode.github.io/husky/) for `web/admin`
already, an equivalent `commit-msg` hook calling
`npx commitlint --edit "$1"` works the same way and is the preferred
approach once Husky is wired up repo-wide.

To check a range of existing commits manually (what CI runs):

```bash
npx commitlint --from <base-sha> --to HEAD --verbose
```

## Releases and changelogs

Do not hand-edit any `CHANGELOG.md` — `release-please` (configured in
`release-please-config.json` / `.release-please-manifest.json`, monorepo
mode) generates one per component from merged Conventional Commits and opens
a release PR. Per `CLAUDE.md`, actually merging a release PR (which cuts a
tag/GitHub Release) requires a human — see
`.github/workflows/release-please.yml` for how the human gate is enforced.
