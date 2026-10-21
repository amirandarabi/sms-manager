# ADR NNNN: <short, decision-shaped title>

**Status:** Proposed | Accepted | Superseded by [ADR NNNN](./NNNN-....md) | Deprecated
**Date:** YYYY-MM-DD
**Deciders:** principal-engineer (+ human sign-off if this changes the stack)
**Related:** SMM-<n>, ARCHITECTURE.md §<n>, ADR NNNN

## Context

What problem are we solving, and what constraints apply? State the forces in
play (scale, team size, phase, operational cost, existing decisions) and the
options considered, with their trade-offs. Keep it factual — no decision yet.

## Decision

The decision, in the present tense and in one sentence if possible ("We use
X for Y"), followed by the concrete rules that follow from it.

If this records a decision already stated in `docs/ARCHITECTURE.md`, the
Decision section must match it exactly — ARCHITECTURE.md stays the source of
truth and this ADR explains *why*.

## Consequences

What becomes easier, what becomes harder, and what other work this obliges
(CI steps, migrations, docs, follow-up ADRs). Include the exit path: is this a
one-way door, and if not, what would trigger revisiting it?

---

**How to use this template**

1. Copy to `docs/adr/NNNN-kebab-case-title.md` with the next free number.
2. Fill every section. Short is fine — one screen is a good target.
3. Add an index row in [`README.md`](./README.md).
4. Decisions that change the tech stack in ARCHITECTURE.md §14 need explicit
   human sign-off; say so in the ADR and do not mark it Accepted before it.
5. Never edit an Accepted ADR's Decision. Supersede it with a new ADR and set
   this one's Status to "Superseded by ADR NNNN".
