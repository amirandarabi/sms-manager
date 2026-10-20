---
description: "Run the orchestration loop over the current phase's Jira issues until done or a human gate is hit"
argument-hint: "[phase-number] [max-parallel default 3]"
---

Run the delivery loop for **phase $0** with up to **$1** implementers in parallel. If that value is empty, use 3.

Follow `docs/AGENT_WORKFLOW.md` §7 exactly:
- Pick unblocked To Do issues with label `phase:$0` in priority/rank order.
- Dispatch each to the agent in its `role:*` label with a self-contained brief (key, AC, ARCHITECTURE refs, branch name, allowed packages).
- Route PRs to `principal-engineer` (+ `security-engineer` when required), then to `qa-engineer`.
- Merge only when the merge conditions are met, then transition to Done and comment.
- Keep Jira statuses and comments current at every step.

Stop and ask me when:
- a human gate from `CLAUDE.md` is reached
- an issue fails twice
- requirements are ambiguous
- the phase backlog is empty

Give me a short status line after each merge, and a final sprint summary (also posted by `product-owner` on the epic).
