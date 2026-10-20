## Jira
SMM-<n>: <link>

## Summary
<!-- What and why, in 2-5 lines -->

## Acceptance criteria
- [ ] AC1
- [ ] AC2

## How it was tested
- Unit:
- Integration:
- E2E / manual:

## Observability
- Metrics added/changed:
- Spans/logs:
- Alerts/runbooks:

## Risk & rollback
- Risk: low | medium | high. Why:
- Rollback: revert PR | feature flag `...` | migration down-path

## Checklist
- [ ] Conventional commit title with Jira key
- [ ] `buf breaking` passes (if proto changed)
- [ ] Error catalog updated (if new `SMS-xxxx` codes)
- [ ] Docs / README / ADR updated
- [ ] No secrets, real phone numbers, or internal hostnames
