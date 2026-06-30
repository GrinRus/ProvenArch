# Live E2E Operator Assessment

> Optional phase-by-phase operator notes over machine evidence. Release readiness now requires `scripts/verify-release-verdict.py` plus the two accepted SWE assessment reports.

## Preflight
- host/tree status:
- provider binaries:
- canonical path checkouts:
- operational blockers:

## Selected Command
```bash
# direct public command only
```

## Matrix/Profile Evidence
- matrix id:
- matrix/profile status files:
- inventories:
- selected providers:
- selected run indexes:

## Backend Execution Evidence
- run matrix:
- execution report:
- taskrun quality JSON telemetry:
- raw metadata/logs:
- classification:

## Frontend Evidence
- frontend matrix:
- result JSON:
- Playwright/server logs:
- UI/API observation:
- classification:

## UI/UX Evidence
- desktop review screenshot:
- desktop Ask screenshot (if QA smoke ran):
- mobile review screenshot:
- stage walkthrough:
- artifact readability:
- activity/log usability:
- Ask evidence classification:
- operator UX decision:

## Required SWE Companion Reports
- UX assessment: `reports/swe_ux_assessment_<matrix-id>.md`
- artifact quality assessment: `reports/swe_artifact_quality_assessment_<matrix-id>.md`
- required decision in both reports: `accepted`

## Verifier Output
```text
# python3 scripts/verify-release-verdict.py reports/release_verdict_<matrix-id>.json
```

## Classification
- status: passed|failed|skipped|blocked
- primary classification:
- false-pass risk:
- false-fail risk:

## Final Decision
- execution readiness source:
- UX readiness source:
- artifact readiness source:
- operator assessment:
- next decision: continue|stop|rerun diagnostic|verify verdict|final report
