# Tracker Reconciliation 2026-05-07

## Summary

This note records the evidence-backed reconciliation of historical unchecked items from `docs/PLANS.md` and cleanup follow-ups from `docs/BACKLOG.md`.

Result:
- historical active ExecPlans were moved to `docs/archive/PLANS_ARCHIVE_2026-05.md` under "Reconciled active plans from 2026-05-07";
- active work is now represented by three consolidated plans:
  - `EP-20260507-trusted-live-validation`;
  - `EP-20260507-provider-reporting-refactor`;
  - `EP-20260507-cleanup-owner-decisions`.

No live matrix was run as part of this reconciliation.

## Classification Matrix

| Source | Classification | Evidence | Follow-up |
|---|---|---|---|
| `EP-20260506-regres-small-live-bugfix-matrix` | needs trusted live rerun | Local implementation and DoD are recorded in its progress log; active unchecked item was only trusted-machine rerun. Runtime/provider diagnostics are documented in `README.md` and `docs/RELEASE_LIVE_E2E_RUNBOOK.md`. | Consolidated into `EP-20260507-trusted-live-validation`. |
| `EP-20260502-qwen-codex-regres-fast-live-hardening` | mixed: trusted live rerun + refactor follow-up | Local acceptance has targeted tests: `internal/runtime/providercommon/engine_test.go`, `internal/runtime/qwencode/runner_test.go`, `internal/runtime/promptcontract/promptcontract_test.go`, `internal/runtime/headless_include_dirs_test.go`, `scripts/tests/batch_failure_classification_test.py`, `scripts/tests/matrix_release_contract_test.py`. Refactor follow-up remains explicitly non-blocking. | Live gate consolidated into `EP-20260507-trusted-live-validation`; refactors consolidated into `EP-20260507-provider-reporting-refactor`. |
| `EP-20260425-runtime-providers-stabilization` | needs trusted live rerun | Proposals contract hardening is covered by `internal/runtimedrafts/manifest_test.go`, `internal/artifactquality/policy_test.go`, `internal/runtime/steppolicy/policy_test.go`, and docs in `README.md` / `docs/RELEASE_LIVE_E2E_RUNBOOK.md`; unchecked item is live verification. | Consolidated into `EP-20260507-trusted-live-validation`. |
| `EP-20260423-regres-fast-failure-taxonomy-hardening` | needs trusted live rerun | Local behavior is covered by orchestrator/runtime/script tests, including `TestDocFlowSkipsAsIsRuntimeWhenCollectEvidenceIsUnusable`, qwen artifact-classification tests, and batch/frontend classifier tests. Unchecked item is live rerun. | Consolidated into `EP-20260507-trusted-live-validation`. |
| `EP-20260423-regres-fast-qwen-hardening` | already implemented locally; needs trusted live rerun | Evidence includes `TestNormalizeSemanticSnapshotDedupesServiceTokenVariantsAndFindingSignatures`, `test_python_report_reads_runner_logs_from_headless_workspace_candidate`, `test_shell_classifier_reads_capacity_signal_from_headless_workspace`, `test_python_report_detects_runner_unavailable_capacity_signal`, `test_runtime_flow_best_effort_partial_skips_run_partial_enforcement_on_runner_unavailable`, and matrix preflight tests in `scripts/tests/matrix_release_contract_test.py`. | Consolidated into `EP-20260507-trusted-live-validation`. |
| `EP-20260422-headless-legacy-cleanup` | already implemented locally; needs trusted live rerun | Evidence includes `internal/runtime/headless_include_dirs_test.go`, `internal/artifactquality/canonicalize_test.go`, `internal/contracts/docflow_test.go`, `TestDocFlowSkipsAsIsRuntimeWhenCollectEvidenceIsUnusable`, and stale-profile reconciliation tests. | Consolidated into `EP-20260507-trusted-live-validation`. |
| `EP-20260422-headless-legacy-residuals` | already implemented locally; needs trusted live rerun | Evidence includes citation-only semantic evidence rejects in `internal/artifactquality/canonicalize_test.go`, validator verdict metadata tests in `internal/runtime/steppolicy/policy_test.go` and `internal/runtime/promptcontract/promptcontract_test.go`, qwen unavailable tests, and `batch-owner.env`/stale profile reconciliation tests. | Consolidated into `EP-20260507-trusted-live-validation`. |
| `EP-20260422-docflow-runtime-residuals` | already implemented locally; needs trusted live rerun | Evidence includes strict as-is manifest tests in `internal/runtimedrafts/manifest_test.go`, non-collect working directory tests in `internal/runtime/headless_include_dirs_test.go`, document-id and semantic alias tests in `internal/orchestrator/docflow_test.go`, owner-gap downgrade in `internal/orchestrator/docflow_pipeline_test.go`, and `runtime_flow_failed` classifier tests. | Consolidated into `EP-20260507-trusted-live-validation`. |
| `EP-20260421-cleanup-owner-followups` | owner-gated | Its open items require explicit owner decisions before moving packages or deleting archive/history surfaces. | Consolidated into `EP-20260507-cleanup-owner-decisions`. |
| `EP-20260420-regres-small-live-triage` | superseded by later triage/fix cycles; remaining work is live validation | Later active plans record multiple live triage cycles and local fixes; the original triage process is no longer the active unit of work. | Consolidated into `EP-20260507-trusted-live-validation`. |
| `docs/BACKLOG.md` cleanup follow-ups | owner-gated | Items explicitly name owners and risks for readable fixtures, duplicated readable fixtures, and `docs/LOCAL_FULL_RUN_AI_ADVENT.md`. | Consolidated into `EP-20260507-cleanup-owner-decisions`. |

## Critical Analysis

The main issue was tracker drift, not a newly discovered local runtime contract failure. Historical plans had unchecked boxes for behaviors that are now covered by code, docs, and tests, while the actual remaining work was concentrated in live validation, refactor debt, and owner-gated cleanup decisions.

Residual risk:
- evidence is test/docs based for local behavior; it does not replace trusted-provider live evidence;
- consolidated live validation can still produce product or operational blockers and must open narrower fix slices instead of closing by assumption;
- owner-gated cleanup remains intentionally unresolved until owners choose retain/remove/dedupe.

Acceptance for this reconciliation slice:
- active plan surface now contains only genuinely open consolidated slices;
- original historical active content is preserved in the archive;
- local runtime/docflow residuals are not treated as closed without evidence references;
- no live matrix was run or implied as passed.
