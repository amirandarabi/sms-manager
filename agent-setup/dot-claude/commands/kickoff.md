---
description: "Plan a delivery phase into Jira (epics, stories, AC, dependencies) via the product-owner agent"
argument-hint: "[phase-number e.g. 0]"
---

Kick off **Phase $ARGUMENTS** of SMS Manager.

1. Read `docs/ARCHITECTURE.md` §19 (delivery plan) and the sections that phase depends on.
2. Delegate to the `product-owner` agent:
   - Create or refresh the Phase $ARGUMENTS epic(s) and stories in Jira per `docs/AGENT_WORKFLOW.md` §3–§4.
   - Search before creating, so nothing is duplicated.
   - Include the cross-cutting stories (observability, CI/CD, security, QA, docs).
   - Link dependencies and move Ready items to To Do.
3. Ask `principal-engineer` to sanity-check the story breakdown: contract-first order, PR-sized stories, missing technical tasks. Feed corrections back to `product-owner`.
4. Show me:
   - the epic link
   - a table of stories in dependency order (key, summary, role, points, blocked-by)
   - open questions needing my decision
5. Wait for my confirmation before starting `/sprint`.
