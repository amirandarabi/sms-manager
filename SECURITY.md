# Security Policy

SMS Manager is built in the open: this repository and its Jira board are public. This
file is the baseline vulnerability-reporting policy. See
[`docs/security/threat-model.md`](docs/security/threat-model.md) for the project's threat
model and current security posture.

## Reporting a vulnerability

**Do not open a public GitHub issue or a public Jira ticket for a security
vulnerability.** This board and repo are public, and exploit details or leaked secrets
posted there are visible to everyone.

Instead:

1. Use **GitHub's private vulnerability reporting**: open the repository's
   **Security → Advisories → Report a vulnerability** form. This creates a private
   advisory visible only to maintainers.
2. If that isn't available, contact a maintainer directly (see repository owner) rather
   than filing anything public.

Please include: affected component/endpoint, impact, and reproduction steps. We'll
acknowledge the report, assess severity, and coordinate a fix and disclosure timeline
with you before any public write-up.

## What happens next

- Confirmed vulnerabilities are tracked as a Jira bug, but **the public Jira description
  only ever states impact and fix status — never exploit steps, secrets, or PoC
  payloads** (see `CLAUDE.md` safety rules).
- If a leaked secret is found (in this repo, in CI logs, or anywhere public), it is
  revoked/rotated immediately; the finding is not posted publicly with the secret value.

## Supported scope

This project is pre-release (Phase 0/1). There are no published releases yet; report
against the `main` branch.

## Scanning

Automated secret/dependency/image scanning (gitleaks, govulncheck, Trivy) in CI is
tracked separately (SMM-10) and will extend this policy once it lands — this file
intentionally stays minimal until then.
