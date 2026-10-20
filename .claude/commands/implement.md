---
description: "Implement a single Jira issue end to end (implement → PR → review → QA → merge)"
argument-hint: "[JIRA-KEY e.g. SMM-42]"
---

Implement Jira issue **$ARGUMENTS** end to end.

1. Fetch the issue (atlassian MCP) and check that its blockers are Done. If they aren't, report and stop.
2. Spawn the agent named in its `role:*` label with a self-contained brief (see `engineering-manager` instructions).
3. When the PR is open, spawn `principal-engineer` to review (plus `security-engineer` if the PR touches auth, secrets, crypto, CI, deps or public endpoints).
4. If changes are requested, send the implementer back to the same branch (`gh pr checkout`) with the review notes. Repeat up to 2 rounds, then escalate to me.
5. Spawn `qa-engineer` to verify the AC.
6. If all merge conditions are met (`docs/AGENT_WORKFLOW.md` §6), squash-merge, transition to Done, and post the Done comment.
7. Report: PR link, review rounds, QA result, final Jira status.
