---
description: "Principal (and security, if needed) review of a GitHub PR, with verdict posted to the PR and Jira"
argument-hint: "[PR-number]"
---

Review PR **#$ARGUMENTS**.

1. Delegate to `principal-engineer`: follow its review procedure, post the verdict on the PR and the linked Jira issue, set the `review:*` label, and transition Jira accordingly.
2. If the PR touches auth, secrets, crypto, OTP, webhooks, public endpoints, CI/CD, dependencies or Dockerfiles (or has the `security-review` label), also delegate to `security-engineer`.
3. Summarize to me: the verdict, the must-fix items, and the resulting Jira status.
