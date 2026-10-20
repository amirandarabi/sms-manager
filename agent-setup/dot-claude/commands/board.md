---
description: "Show the current Jira board and PR status for SMS Manager (read-only)"
argument-hint: "[optional phase-number]"
---

Give me a read-only status report. **Change nothing.**

1. JQL on the project from `CLAUDE.md` (filter `labels = phase:$ARGUMENTS` if a phase is given). Group the issues by status: To Do, In Progress, In Review, QA, Blocked, Done this sprint.
2. For In Review/QA issues, show the PR number, CI state (`gh pr checks`) and review label.
3. List Blocked issues with the reason (from the latest comment) and who is needed.
4. Show points done vs remaining, and the top 3 risks.
Output as compact tables.
