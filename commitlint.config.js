// Conventional Commits enforcement for sms-manager (ARCHITECTURE.md §18.2).
//
// Scopes are expected to match a component in the monorepo (api, dispatcher,
// providers/<name>, pkg/<name>, web/admin, deploy, docs, ci, repo, ...) so
// release-please and per-component changelogs stay meaningful. The scope
// list below is intentionally permissive (free-form, lower-case,
// kebab/slash-safe) rather than a hard enum, because new components land
// every phase — see docs/CONTRIBUTING.md for the convention and examples.
//
// Usage:
//   npx commitlint --from <sha> --to HEAD   (CI, see .github/workflows/commitlint.yml)
//   npx --no -- commitlint --edit "$1"      (local pre-commit/commit-msg hook)

/** @type {import('@commitlint/types').UserConfig} */
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Require a scope on every commit so release-please can attribute the
    // change to a component, e.g. `feat(api): add health check endpoint`.
    'scope-empty': [2, 'never'],
    'scope-case': [2, 'always', 'kebab-case'],
    'subject-case': [0],
    'header-max-length': [2, 'always', 100],
    'type-enum': [
      2,
      'always',
      [
        'feat',
        'fix',
        'perf',
        'revert',
        'docs',
        'build',
        'chore',
        'refactor',
        'test',
        'ci',
        'style',
      ],
    ],
  },
};
