# Historical documentation archive

These documents retain dated decisions, reviews and execution evidence. Their old status claims,
commands, source line numbers and machine-local paths describe the recorded revision; they do not
set current implementation acceptance or qualify a release. The 2026-09-05 cleanup moved the files
below and rebased repository links without removing their original findings or progress logs.

Use the [current architecture](../ARCHITECTURE.md), [specifications](../AGENT_DEVELOPMENT.md#карта-изменений),
[task-first UX](../UI_TASK_FIRST_PRODUCT_DESIGN.md), [canonical status matrix](../STAKEHOLDER_DOC.md)
and [active plans](../PLANS.md) for present work. The current 13-screen composition reference set
remains with the task-first UX specification.

## Superseded UI directions

| Historical record | Why retained |
| --- | --- |
| [Console V2](design/UI_CONSOLE_V2_DESIGN.md) | Decisions behind the earlier operator console; retired visual assets are not current references. |
| [Architecture Change Review design](design/UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md) | Epic 20 information architecture and evidence/publication authority rationale. |
| [Architecture Change Review migration](design/UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md) | Historical delivery dependencies and acceptance, superseded by the current task-first migration plan. |

## Research and proposals

| Historical record | Why retained |
| --- | --- |
| [Karpathy LLM Wiki comparison, 2026-06-18](research/KARPATHY_LLM_WIKI_COMPARISON_2026-06-18.md) | Dated external research and comparison baseline. |
| [Adoption analysis, 2026-06-18](research/KARPATHY_ADOPTION_ANALYSIS_2026-06-18.md) | Tradeoffs and boundaries for importing selected ideas. |
| [Integration design, 2026-07-09](research/KARPATHY_PROVENARCH_INTEGRATION_DESIGN_2026-07-09.md) | Rationale for health, citation and explicit Ask-to-Proposal work, plus future proposals. |

Health and Ask-to-Proposal implementation has since changed from these proposals. In particular,
current Workspace Health is a read-only API snapshot and does not persist the proposed
`reports/health/*` files. Unimplemented claim-ledger/search proposals require a separately selected
slice; their presence here does not authorize implementation.

## Audit and UX evidence

| Historical record | Scope and evidence availability |
| --- | --- |
| [Code quality audit, 2026-07-10](audits/CODE_AUDIT_2026-07-10.md) | Findings and DEAD register at the recorded `122e4c9…` baseline. Epic 19 remediation status is recorded in the canonical matrix; old identifiers are not a current deletion list. |
| [UX current state, 2026-07-07](audits/ux_current_state_20260707.md) | Console V2 fake smoke and UX findings. Original `/tmp` screenshots/workspace are not retained in this repository and were absent at archival. |
| [UX/UI assessment, 2026-07-08](audits/ux_ui_assessment_20260708.md) | Console V2 improvement and live-diagnostic history. Many screenshot/report paths under `/tmp` were absent at archival; retained text records their original observations. |
| [UX/UI assessment, 2026-07-24](audits/ux_ui_assessment_20260724.md) | Intermediate shell audit. `/tmp/acp-ux-audit/shots` and `/tmp/acp-ux-audit-results` were absent at archival. |
| [Task-first design QA, 2026-08-21 through 2026-08-25](audits/DESIGN_QA_2026-08-21.md) | Iterations from blocked findings to accepted Q10 and later verification. Initial private visualization directories existed on the audit host; later `/tmp` Lighthouse/Q10/Ask captures were absent. None of these external paths is a portable archive attachment. |

Missing local screenshots limit reproduction of a historical review; they neither establish a
current failure nor invalidate the dated written observation. Original local paths and commands
remain in the records for provenance. Release readiness still requires the separate evidence
specified by the [live runbook](../RELEASE_LIVE_E2E_RUNBOOK.md).

## Plan history

Monthly [April](PLANS_ARCHIVE_2026-04.md), [May](PLANS_ARCHIVE_2026-05.md),
[June](PLANS_ARCHIVE_2026-06.md), [July](PLANS_ARCHIVE_2026-07.md),
[August](PLANS_ARCHIVE_2026-08.md) and [September](PLANS_ARCHIVE_2026-09.md) archives preserve
completed implementation and old tracker snapshots. The [April 21 snapshot](PLANS_SNAPSHOT_2026-04-21.md)
and [May 7 reconciliation](TRACKER_RECONCILIATION_2026-05-07.md) retain their distinct historical
context. Unresolved live qualification and owner decisions remain in the active plan index.
