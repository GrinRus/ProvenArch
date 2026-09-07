# SWE Artifact Quality Assessment

> Manual SWE-agent artifact review over completed live E2E evidence. Start from promoted,
> user-visible evidence and its selected-run snapshot. Runtime/taskrun telemetry is diagnostic
> context only: it may explain how output was produced, but it cannot substitute for reviewing
> what the user reads. This report is release evidence, but it does not change
> `release_verdict_<matrix-id>.json`.

## Decision
- matrix_id:
- decision: accepted|rejected|inconclusive|blocked
- source_sha:
- assessed_by:
- assessed_at_utc:

## Evidence Inspected
- promoted overview/findings/coverage:
- promoted model/entities/edges:
- promoted diagrams:
- promoted proposals/changelog/publish artifacts:
- selected-run final index and staged snapshot:
- citations and broken/unresolved references:
- frontend evidence rendering (Review/Knowledge/Publish):
- Ask evidence:

## Diagnostic Execution Context (not artifact acceptance)
- release verdict: release_verdict_<matrix-id>.json
- execution reports and run matrices:
- taskrun quality telemetry:
- repair/stall/provider diagnostics:

## Artifact Findings
- architecture truthfulness:
- evidence density/readability:
- C4/Mermaid usefulness:
- decision-ready summary:
- proposal/actionability:
- misleading output risk:
- missing/weak evidence:

## Residual Risk
- visual regression:
- accessibility:
- trend history:
- mobile/non-happy-path coverage:

## Final Notes
- release artifact decision:
