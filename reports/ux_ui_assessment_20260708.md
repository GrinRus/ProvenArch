# UX/UI Assessment - 2026-07-08

This report is product UX evidence for the ACP operator console. It is not release readiness evidence and does not replace `release_verdict_<matrix-id>.json`.

## Product Context

- Product type: local-first developer/operator tool.
- Primary users: architects, tech leads and local operators running ACP on their own machine.
- Primary job: configure an architecture workspace, run analysis, inspect evidence, ask read-only questions and publish Git-versioned artifacts.
- Main flow: `Workspace -> Sources -> Runner -> Ready`, then `Source -> Readiness -> Charter -> Analysis -> Review -> Proposals -> Ask -> Publish`.
- Quality bar: a first-time user should understand the next action, current blocker, generated evidence and recovery path without reading raw logs first.

## Evidence Inspected

- Project docs: `README.md`, `docs/ARCHITECTURE.md`, `docs/spec/PIPELINE_SPEC.md`, `docs/UI_CONSOLE_V2_DESIGN.md`, `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.
- Current implementation: `ui/src/components/AppShell.tsx`, `RightInspector.tsx`, `ActivityDrawer.tsx`, `ActiveRunStrip.tsx`, `StagePanels.tsx`, `styles.css`.
- Existing UX report: `reports/ux_current_state_20260707.md`.
- Fresh fake-runtime rendered smoke:
  - build: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build`
  - server: `./bin/acp serve --workspace /tmp/provenarch-ui-ux-smoke-workspace.JRJ01W --auto-init --repo-name provenarch --repo-path <repo> --runtime fake --listen 127.0.0.1:18180`
  - Playwright: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-ux-smoke-results.3lR2MF npm run --prefix ui e2e:live`
  - init run: `run_20260708_094849_001`
  - QA run: `run_20260708_094856_002`
  - screenshots: `/tmp/provenarch-ui-ux-smoke-results.3lR2MF/frontend-*.png`

## Live E2E Medium Gate

goal: run the requested medium live E2E path through public runbook surfaces.
action: selected canonical non-release `regres long` (`examples/e2e-matrix.regres-long.yaml`) and generated direct command with `python3 scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell`.
observed evidence: host root `/tmp/provenarch-live-e2e` is writable; `/tmp/provenarch-test_arch_project` report/matrix roots are writable; exact Node toolchain resolves; provider binaries report `qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`; current worktree is dirty because this UX plan/report is in progress; curated PostHog path `/tmp/provenarch-live-e2e/posthog/posthog` exists but is not a Git checkout and `git rev-parse HEAD` fails.
status: blocked
primary classification: operational_host_preflight_failed
next decision: do not run matrix from this host state; prepare the exact pinned PostHog checkout or symlink per runbook section `2.2`, then rerun from a clean committed tree or clean worktree.

2026-07-08 slice 2 retry:
goal: re-check whether the requested medium live E2E can start after the Review UX slice.
action: ran fail-fast host checks, generated the direct command with `python3 scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell`, checked provider binaries and probed `/tmp/provenarch-live-e2e/posthog/posthog` with `git rev-parse HEAD`.
observed evidence: `/tmp/provenarch-live-e2e` and `/tmp/provenarch-test_arch_project` are writable directories; generated command still targets `examples/e2e-matrix.regres-long.yaml` with qwen-only baseline and `scripts/full-run-batch-matrix.sh`; binaries report `qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`; PostHog canonical path still returns `fatal: not a git repository`.
status: blocked
primary classification: operational_host_preflight_failed
next decision: stop before matrix execution; fix the canonical PostHog checkout/pinned SHA on a trusted host, then rerun from a clean committed tree.

2026-07-08 slice 3 retry:
goal: re-check whether the requested medium live E2E can start after the Readiness UX slice.
action: ran fail-fast host checks, generated the direct command with `python3 scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell`, checked provider binaries and probed `/tmp/provenarch-live-e2e/posthog/posthog` with `git rev-parse HEAD`.
observed evidence: `/tmp/provenarch-live-e2e` and `/tmp/provenarch-test_arch_project` are writable directories; generated command still targets `examples/e2e-matrix.regres-long.yaml` with qwen-only baseline and `scripts/full-run-batch-matrix.sh`; binaries report `qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`; PostHog canonical path still returns `fatal: not a git repository`.
status: blocked
primary classification: operational_host_preflight_failed
next decision: stop before matrix execution; fix the canonical PostHog checkout/pinned SHA on a trusted host, then rerun from a clean committed tree.

2026-07-08 slice 4 retry:
goal: re-check whether the requested medium live E2E can start after the failed-run recovery UX slice.
action: ran fail-fast host checks, generated the direct command with `python3 scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell`, checked provider binaries and probed `/tmp/provenarch-live-e2e/posthog/posthog` with `git rev-parse HEAD`.
observed evidence: `/tmp/provenarch-live-e2e` and `/tmp/provenarch-test_arch_project` are writable directories; generated command still targets `examples/e2e-matrix.regres-long.yaml` with qwen-only baseline and `scripts/full-run-batch-matrix.sh`; binaries report `qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`; PostHog canonical path still returns `fatal: not a git repository`; current worktree is dirty because slice 4 is in progress and not an acceptance run.
status: blocked
primary classification: operational_host_preflight_failed
next decision: stop before matrix execution; fix the canonical PostHog checkout/pinned SHA on a trusted host, then rerun from a clean committed tree.

2026-07-08 slice 5 retry:
goal: re-check whether the requested medium live E2E can start after the Ask/Q&A recovery UX slice.
action: ran fail-fast host checks, generated the direct command with `python3 scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell`, checked provider binaries and probed `/tmp/provenarch-live-e2e/posthog/posthog` with `git rev-parse HEAD`.
observed evidence: `/tmp/provenarch-live-e2e` and `/tmp/provenarch-test_arch_project` are writable directories; generated command still targets `examples/e2e-matrix.regres-long.yaml` with qwen-only baseline and `scripts/full-run-batch-matrix.sh`; binaries report `qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`; PostHog canonical path still returns `fatal: not a git repository`; current worktree is dirty because slice 5 is in progress and not an acceptance run.
status: blocked
primary classification: operational_host_preflight_failed
next decision: stop before matrix execution; fix the canonical PostHog checkout/pinned SHA on a trusted host, then rerun from a clean committed tree.

2026-07-08 slice 6 retry:
goal: re-check whether the requested medium live E2E can start after the Publish handoff UX slice.
action: ran fail-fast host checks, generated the direct command with `python3 scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell`, checked provider binaries and probed `/tmp/provenarch-live-e2e/posthog/posthog` with `git rev-parse HEAD`.
observed evidence: `/tmp/provenarch-live-e2e` and `/tmp/provenarch-test_arch_project` are writable directories; generated command still targets `examples/e2e-matrix.regres-long.yaml` with qwen-only baseline and `scripts/full-run-batch-matrix.sh`; binaries report `qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`; PostHog canonical path still returns `fatal: not a git repository`; current worktree is dirty because slice 6 is in progress and not an acceptance run.
status: blocked
primary classification: operational_host_preflight_failed
next decision: stop before matrix execution; fix the canonical PostHog checkout/pinned SHA on a trusted host, then rerun from a clean committed tree.

2026-07-08 slice 7 retry:
goal: re-check whether the requested medium live E2E can start after the onboarding blocker-summary UX slice.
action: ran fail-fast host checks, generated the direct command with `python3 scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell`, checked provider binaries and probed `/tmp/provenarch-live-e2e/posthog/posthog` with `git rev-parse HEAD`.
observed evidence: `/tmp/provenarch-live-e2e` and `/tmp/provenarch-test_arch_project` are writable directories; generated command still targets `examples/e2e-matrix.regres-long.yaml` with qwen-only baseline and `scripts/full-run-batch-matrix.sh`; binaries report `qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`; PostHog canonical path still returns `fatal: not a git repository`; current worktree is dirty because slice 7 is in progress and not an acceptance run.
status: blocked
primary classification: operational_host_preflight_failed
next decision: stop before matrix execution; fix the canonical PostHog checkout/pinned SHA on a trusted host, then rerun from a clean committed tree.

2026-07-08 slice 8 live matrix run:
goal: run the requested medium live E2E after restoring the canonical PostHog checkout on this trusted host.
action: moved the non-Git placeholder at `/tmp/provenarch-live-e2e/posthog/posthog` aside, restored a real Git checkout at pinned ref `14d29a548d63665d60b506cf13bd5cfb2de7c743`, then ran the direct harness command without a wrapper:
`MATRIX_ID=regres-long-posthog-ftgo-20260708T130309Z E2E_MATRIX_FILE=examples/e2e-matrix.regres-long.yaml E2E_MATRIX_RELEASE_MODE=0 RUN_COUNT=1 BATCH_PROVIDER_FILTER=qwen-code ./scripts/full-run-batch-matrix.sh` with exact Node `v22.21.1` and provider binaries `qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`.
observed evidence: matrix result `/tmp/provenarch-test_arch_project/reports/matrix_result_regres-long-posthog-ftgo-20260708T130309Z.json` is `FAIL`, non-release, `strict_pass_runs=0/2`.
profile evidence:
- PostHog path profile `single-path/baseline` failed with `runtime_contract_failed=1`, `quality_gates_failed=1`, `partial_failure_count=9`, raw output refs `9`, excellent blockers `runtime_quality.partial_failures`, `runtime_quality.repair_exhausted`, `runtime_quality.repair_heavy`, `runtime_quality.stall_pressure`.
- FTGO git_url profile `single-git_url/baseline` failed with `runtime_contract_failed=1`, `quality_gates_failed=1`, `partial_failure_count=8`, raw output refs `8`, `repair_attempts=11`, `repair_exhausted=8`, `focused_repairs=11`, `stall_count=17`, `pre_artifact_stalls=17`, `valid_artifact_controlled_stops=3`; the first blocker excerpt was `documents[0].path references process-contaminated collect document file "root-overview.md"` and the terminal excerpt was `runtime_stalled_before_artifacts`.
status: failed
primary classification: live_runtime_provider_artifact_handoff_failed
UX classification: the live product UI screens beyond backend/API baseline were not reached because headless qwen failed during `init.step1.collect`; the actionable UX gap is live diagnostics clarity for long shard collection, focused repair, partial shard failures and raw-output handoff, not visual polish of Review/Ask/Publish screens.
next decision: implement a focused live-diagnostics UX slice before another medium run: show shard-plan progress, succeeded/failed/repairing counts, current repair reason, terminal validation excerpt, raw-output refs and concrete rerun/triage actions in the console and reports.

2026-07-08 slice 9 diagnostics UX implementation:
goal: make the failed Analysis screen explain live shard collection, focused repair, partial failures, stall pressure and raw-output refs before the operator reads raw logs.
action: added a derived `analysis-live-diagnostics` panel inside Analysis failed-run recovery, using existing `RunLogEntry.fields`, selected artifacts and run warnings; no backend API, schema or runtime contract changed.
status: implemented; full DoD passed; next medium rerun remains the follow-up.

2026-07-08 slice 10 live rerun preflight:
goal: rerun the requested medium live E2E after committing Slice 9 diagnostics.
action: verified clean tree, provider binaries (`qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`), writable reports root and generated direct `regres long` qwen-only command. `/tmp/provenarch-live-e2e` was missing, so the canonical PostHog path checkout had to be restored per runbook `2.2`. A partial clone/refetch of `posthog/posthog` to pinned `14d29a548d63665d60b506cf13bd5cfb2de7c743` failed with `fatal: write error: No space left on device` after the object store reached about `3.3G`; `/tmp` had only `116MiB` free at failure. The temporary partial checkout was removed, restoring `/tmp` to about `3.4GiB` free.
observed evidence: direct selector output still targets `examples/e2e-matrix.regres-long.yaml` with qwen-only baseline and `scripts/full-run-batch-matrix.sh`; no matrix id was started; no canonical matrix or curated repo file was changed.
status: blocked
primary classification: operational_host_preflight_failed
next decision: stop before matrix execution; rerun only on a trusted host or volume with enough free space for the canonical PostHog checkout pinned to the runbook SHA.

## UX Findings

1. Shared shell chrome still competes with the stage task.
   Top status, active run strip, stage rail, right inspector and bottom drawer are individually useful, but together they leave too little room for Review, Ask and Publish work. The issue is priority, not visual style.

2. Activity drawer has the wrong default after success.
   During Analysis, open logs are useful. After a succeeded run, the drawer remains open in Review, Ask and Publish and consumes the lower half of the viewport with historical init logs instead of current task content.

3. Right inspector has too many equal-weight empty sections.
   `Hard blockers`, `Review warnings`, `Open questions`, `Evidence refs`, `Workspace health`, `Runtime safety` and `Git publication` are all rendered as full panels even when most are empty. This makes `Next action` less decisive.

4. Active run strip uses execution wording after execution is finished.
   A succeeded run still emphasizes `Current step init.step4.proposals`. For review tasks the useful summary is artifact count, warnings, errors and whether evidence is ready for human review.

5. Readiness advanced settings are too easy to foreground.
   The live smoke opens them intentionally, but for a first-time operator they can push the core readiness result and first-run action down the viewport.

6. Review has a strong evidence preview but too many inventories at once.
   The review queue, artifact explorer, citation coverage and right inspector all describe overlapping concepts. The first improvement should reduce shell noise before reworking Review internals.

7. Mobile Review is functional but long.
   The screen avoids hard overlap, but users scroll through preview, queue, explorer, citations, inspector and activity. Collapsing secondary shell sections improves mobile immediately.

## First Improvement Slice

P0:
- Make `ActivityDrawer` context-aware: open by default for active/failed runs, collapsed after success, with a compact summary of last event and runtime artifact count.
- Make `RightInspector` progressive: keep `Next action` and non-empty/high-priority sections visible; group secondary empty/health/safety/publication sections into compact disclosure by default.
- Make `ActiveRunStrip` switch to review language after success: show `Review state`, artifact count and warning/error count instead of stale `Current step`.

P1:
- Keep advanced runtime settings collapsed by default outside explicit operator interaction or warnings. Completed in Slice 3 for the Readiness live-smoke path.
- Rebalance Review internals so `Needs review` queue and selected preview are primary, while full artifact explorer is secondary. Completed in Slice 2 for Review.
- Add mobile section jump controls for Review/Publish if the page still feels too long after shell disclosure changes. Completed in Slice 2 for Review; Publish remains a later candidate if rendered evidence shows the same friction.

## Acceptance Criteria For Slice 1

- Fake-runtime live smoke still passes with Ask enabled and mobile Review screenshot.
- Component tests prove active/failed logs remain discoverable while succeeded runs do not open the drawer by default.
- Component tests prove hard blockers/warnings/evidence remain visible, while empty secondary inspector sections are collapsed or summarized.
- Active run strip for a succeeded run names review/artifact state clearly.
- No API, schema, runtime or artifact contract changes.

## Slice 1 Result

Implemented:
- `ActivityDrawer` now opens automatically for queued/running/failed diagnostics and collapses after a succeeded run, keeping a compact summary with the latest event and an explicit `open logs` affordance.
- `RightInspector` keeps `Next action` prominent, shows non-empty warning/evidence sections, and collapses empty or secondary status sections while preserving existing `data-testid` selectors.
- `ActiveRunStrip` now uses review-oriented language after success: `Review state`, artifact count and `evidence ready for review` replace stale current-step emphasis.

Post-change evidence:
- targeted component test: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin npm run --prefix ui test -- ConsoleShellPrimitives.test.tsx --run` -> `9 passed`
- UI typecheck: `npm run --prefix ui typecheck` -> passed
- UI test suite: `npm run --prefix ui test -- --run` -> `82 passed`
- build: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build` -> passed
- rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-ux-polish-results.mM08Mw npm run --prefix ui e2e:live` -> `1 passed`
- updated init run: `run_20260708_095725_001`
- updated QA run: `run_20260708_095731_002`
- updated screenshots: `/tmp/provenarch-ui-ux-polish-results.mM08Mw/frontend-*.png`

Observed improvement:
- Review, Ask and Publish no longer lose the lower half of the viewport to old init logs after success.
- The first visible run summary now says `33 artifacts ready` and `evidence ready for review`, which matches the operator's task after the pipeline succeeds.
- Empty inspector sections are compact but still discoverable; open questions and evidence refs remain visible because they affect review/publish decisions.

Residual UX work for next iteration:
- Non-happy path UX still needs a dedicated pass across failed runs, retry paths, provider unavailable states, QA failures and publish blockers.
- Medium live E2E remains blocked by host preflight until `/tmp/provenarch-live-e2e/posthog/posthog` is a valid pinned Git checkout and the tree is clean.

## Slice 2 Result

Implemented:
- `ReviewEvidenceWorkbench` now separates the left side into a primary review task lane and a secondary full artifact explorer disclosure.
- `ReviewQueuePanel` highlights the currently selected queue item, making the selected evidence preview traceable back to the operator action.
- Review mobile layout now exposes section jumps for `Preview`, `Queue`, `Artifacts` and `Trust` before the long review stack.
- Live Playwright opens the secondary artifact explorer explicitly before selecting diagram artifacts, matching the new UI hierarchy.

Post-change evidence:
- targeted component test: `npm run --prefix ui test -- App.test.tsx --run` -> `66 passed`
- UI typecheck: `npm run --prefix ui typecheck` -> passed
- UI test suite: `npm run --prefix ui test -- --run` -> `82 passed`
- build: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build` -> passed
- rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-review-slice-results.Y78xBZ npm run --prefix ui e2e:live` -> `1 passed`
- updated init run: `run_20260708_102054_001`
- updated screenshots: `/tmp/provenarch-ui-review-slice-results.Y78xBZ/frontend-review-desktop.png`, `/tmp/provenarch-ui-review-slice-results.Y78xBZ/frontend-review-mobile.png`

Observed improvement:
- First-time Review now starts with the next review tasks and selected evidence, not a full file catalog.
- Full artifact browsing remains discoverable through an explicit disclosure and keeps the existing `review-artifact-explorer` test surface.
- Mobile Review has stable section-level navigation before preview, queue, artifacts and trust/citation sections.

Residual UX work after Slice 2:
- Publish should receive the same mobile/long-form review treatment only if screenshots show comparable navigation friction.
- Error, retry and provider-unavailable paths still need a dedicated non-happy-path UX pass.

## Slice 3 Result

Implemented:
- `ReadinessStagePanel` now gives advanced runtime settings a compact operator-tools disclosure with descriptive summary copy.
- Advanced runtime panels remain closed by default in the first-run Readiness path, while the exact timeout/execution/permissions/provider override panels remain one click away.
- Live Playwright still opens the advanced disclosure to prove the settings are reachable, then closes it before the Readiness screenshot so first-run evidence reflects the intended hierarchy.

Post-change evidence:
- targeted component test: `npm run --prefix ui test -- App.test.tsx --run` -> `66 passed`
- UI typecheck: `npm run --prefix ui typecheck` -> passed
- UI test suite: `npm run --prefix ui test -- --run` -> `82 passed`
- build: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed
- rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-readiness-slice-results.Py1ckv npm run --prefix ui e2e:live` -> `1 passed`
- updated init run: `run_20260708_104434_001`
- updated screenshot: `/tmp/provenarch-ui-readiness-slice-results.Py1ckv/frontend-readiness-desktop.png`

Observed improvement:
- First-run Readiness foregrounds validation, local readiness and run-first-analysis actions instead of detailed runtime tuning panels.
- Operator controls remain discoverable with concrete scope text: timeouts, execution policy, permissions and per-step providers.

Residual UX work after Slice 3:
- Error, retry and provider-unavailable paths still need a dedicated non-happy-path UX pass.
- Publish should receive the same mobile/long-form review treatment only if screenshots show comparable navigation friction.

## Slice 4 Result

Implemented:
- `AnalysisStagePanel` now renders a dedicated `Recovery path` panel for failed selected runs.
- The recovery panel summarizes error classification, blocked step, retained evidence refs or diagnostic rows, warning count and runtime/provider.
- Failed runs now expose a primary same-pipeline retry action (`Retry init` / `Retry refresh`) and a secondary blocker drilldown action without adding API or contract surface.
- Recovery guidance now distinguishes permission-required, provider-unavailable, timeout, runtime-contract and incomplete/infra-style failures with short operator-facing copy.

Post-change evidence:
- targeted component test: `npm run --prefix ui test -- App.test.tsx --run` -> `66 passed`
- UI typecheck: `npm run --prefix ui typecheck` -> passed
- UI test suite: `npm run --prefix ui test -- --run` -> `82 passed`
- build: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed
- rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-recovery-slice-results.Ikr22T npm run --prefix ui e2e:live` -> `1 passed`
- updated init run: `run_20260708_111016_001`
- updated screenshots: `/tmp/provenarch-ui-recovery-slice-results.Ikr22T/frontend-analysis-desktop.png`, `/tmp/provenarch-ui-recovery-slice-results.Ikr22T/frontend-review-mobile.png`

Observed improvement:
- A failed Analysis run now has an explicit recovery path instead of scattering the decision across status text, timeline, history and logs.
- First-time operators can see what failed, where it failed, what evidence remains available and which retry action is appropriate before reading raw logs.
- Existing blocker drilldown, pending permissions, run status and history remain available for deeper diagnosis.

Residual UX work after Slice 4:
- QA failure states need the same recovery-path treatment as pipeline failures. Resolved in Slice 5.
- Publish blockers and mobile Publish navigation remain candidates for the next non-happy-path pass.

## Slice 5 Result

Implemented:
- `AskStagePanel` now renders a dedicated `Recovery path` panel for failed Q&A runs.
- The Q&A recovery panel summarizes error classification, blocked step, `reports/taskruns/<run_id>/qa/` audit refs, warning count, error/warning details and read-only safety context.
- Failed Q&A runs now expose a primary same-question retry action while preserving history selection and existing async `POST /api/qa/runs` semantics.
- Rendered smoke exposed a Review regression where switching the secondary artifact explorer to `Diagrams` could close the disclosure or leave the diagram list hidden. `ReviewEvidenceWorkbench` now keeps the explorer controlled and filter actions keep it open.
- The live smoke path now switches back to `Reports` before checking markdown text preview, matching the UI distinction between Mermaid diagram preview and markdown artifact preview.

Post-change evidence:
- targeted component test: `npm run --prefix ui test -- App.test.tsx --run` -> `67 passed`
- UI typecheck: `npm run --prefix ui typecheck` -> passed
- build: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build` -> passed
- rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-qa-recovery-results.TF7gHz npm run --prefix ui e2e:live` -> `1 passed`
- updated init run: `run_20260708_115053_001`
- intermediate smoke findings fixed in this slice: `run_20260708_114506_001` exposed hidden `run-diagrams-list`; `run_20260708_114847_001` exposed that the smoke needed to return from diagram preview to report preview before asserting markdown text.
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed

Observed improvement:
- A failed Q&A run now has an explicit recovery path instead of relying on raw status/error text and history alone.
- First-time operators can see what failed, which QA step is blocked, where audit artifacts live and how to retry the same question before reading raw logs.
- Review secondary artifact browsing remains stable while changing filters; diagram preview and markdown preview are now tested as distinct user actions.

Residual UX work after Slice 5:
- Publish blockers and mobile Publish navigation remain candidates for the next non-happy-path pass. Resolved in Slice 6.
- Medium live E2E remains blocked by trusted-host repo setup until `/tmp/provenarch-live-e2e/posthog/posthog` is restored as the pinned Git checkout.

## Slice 6 Result

Implemented:
- `PublishStagePanel` now shows a publication readiness summary above the fold: publication set, gate state, open-question count and Git action state.
- Publish blocker details in the summary include the gate classification label, so `runtime_contract_failed` and similar blockers remain visible without opening raw logs or scrolling to the gate section.
- Mobile Publish now has section jumps for `Diff`, `Preview`, `Gate` and `Commit`, reusing the Review jump treatment for long final-handoff screens.
- Live Playwright now asserts the Publish summary/jumps and captures `frontend-publish-mobile.png`.

Post-change evidence:
- targeted component test: `npm run --prefix ui test -- App.test.tsx --run` -> `67 passed`
- UI typecheck: `npm run --prefix ui typecheck` -> passed
- frontend live E2E contract test: `python3 -m unittest scripts.tests.frontend_live_e2e_contract_test.FrontendLiveE2EContractTest` -> `19 passed`
- build: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build` -> passed
- rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-publish-nav-results.VBvGYh npm run --prefix ui e2e:live` -> `1 passed`
- updated init run: `run_20260708_121704_001`
- updated screenshots: `/tmp/provenarch-ui-publish-nav-results.VBvGYh/frontend-publish-desktop.png`, `/tmp/provenarch-ui-publish-nav-results.VBvGYh/frontend-publish-mobile.png`
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed

Observed improvement:
- First-time operators can see whether Publish is ready, under review or blocked before reading the lower gate/commit panels.
- Mobile Publish now exposes direct jumps to the review inventory, preview, gate and commit plan instead of forcing a single long scroll.
- Publish blocker classification is visible in the top summary, not only inside the detailed gate checklist.

Residual UX work after Slice 6:
- Medium live E2E remains blocked by trusted-host repo setup until `/tmp/provenarch-live-e2e/posthog/posthog` is restored as the pinned Git checkout and the run starts from a clean committed tree.
- Onboarding error states and provider permission UI remained future audit candidates before Slice 7.

## Slice 7 Result

Implemented:
- `OnboardingShell` now shows an above-the-fold setup progress summary before Console V2: current step, next action, current blocker and per-step status details.
- The `Ready` card now explains why `Open console` and `Run first analysis` are disabled, using existing workspace/source/runtime/doctor state.
- Source setup errors such as `repo_name_duplicate` and runner readiness blockers such as missing provider commands are elevated into the top onboarding summary instead of living only in lower diagnostics.
- Desktop and mobile onboarding layout keep the progress summary readable without adding backend API, schema or runtime contract changes.

Post-change evidence:
- targeted component test: `npm run --prefix ui test -- App.test.tsx --run` -> `67 passed`
- UI typecheck: `npm run --prefix ui typecheck` -> passed
- UI test suite: `npm run --prefix ui test -- --run` -> `83 passed`
- build: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build` -> passed
- rendered onboarding smoke: `/tmp/provenarch-ui-onboarding-summary-results.KbB5En/onboarding-initial-desktop.png`, `/tmp/provenarch-ui-onboarding-summary-results.KbB5En/onboarding-initial-mobile.png`, `/tmp/provenarch-ui-onboarding-summary-results.KbB5En/onboarding-source-error-desktop.png`
- fake-runtime rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-onboarding-slice-results.TSpfmo npm run --prefix ui e2e:live` -> `1 passed`
- updated init run: `run_20260708_124024_001`
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed

Observed improvement:
- First-time operators now see the current setup blocker before interacting with lower cards.
- Disabled primary actions now tell the user what to fix next, including source validation diagnostics and readiness blockers.
- Recoverable onboarding source errors are visible in three aligned places: top blocker summary, affected repo row and Ready action hint.

Residual UX work after Slice 7:
- Medium live E2E is no longer blocked by trusted-host repo setup; Slice 8 executed the matrix and found live runtime/provider artifact-handoff failures during `init.step1.collect`.
- Provider permission approval/denial UI remains a future deeper non-happy-path audit area once a trusted-host live run can produce representative pending-permission evidence.

## Slice 8 Result

Implemented:
- Restored the local PostHog trusted-host checkout outside the repo so the canonical `regres long` matrix could run through the public harness instead of stopping at host preflight.
- Ran the qwen-only baseline matrix with `scripts/full-run-batch-matrix.sh` and captured machine evidence for both medium profiles.
- Classified the result as a live runtime/provider artifact-handoff failure rather than a visual UI regression, because both profiles failed before the live UI could exercise Review, Ask or Publish on provider-authored artifacts.

Post-change evidence:
- matrix result: `/tmp/provenarch-test_arch_project/reports/matrix_result_regres-long-posthog-ftgo-20260708T130309Z.json` -> `FAIL`
- PostHog profile status: `/tmp/provenarch-test_arch_project/matrix/regres-long-posthog-ftgo-20260708T130309Z/profile-status/single-path--baseline.json` -> `failed`, `failure_reason=child_failed`
- FTGO profile status: `/tmp/provenarch-test_arch_project/matrix/regres-long-posthog-ftgo-20260708T130309Z/profile-status/single-git-url--baseline.json` -> `failed`, `failure_reason=child_failed`
- FTGO execution report: `/tmp/provenarch-test_arch_project/reports/execution_report_regres-long-posthog-ftgo-20260708T130309Z-single-git-url-baseline.md`
- FTGO run status: `state=process_failed`, `failure_reason=pipeline command failed for runtime=headless:qwen-code pipeline=init`

Observed improvement:
- The previous trusted-host blocker is gone; the matrix now reaches real qwen `init.step1.collect` execution on both medium targets.
- The existing reporting already exposes useful raw refs and quality blocker codes for partial shard failures.

Residual UX work after Slice 8:
- The console still needs a first-class live diagnostics view for long shard collection: current shard, total shards, succeeded/failed/repairing counts, repair attempt count, stall pressure, terminal validation excerpt and raw-output refs.
- Recovery copy should distinguish "provider still working", "focused repair running", "partial collect can continue best-effort" and "terminal runtime contract failure" before asking the operator to inspect raw logs.
- A repeat medium run should happen only after the live-diagnostics UX slice, so the next audit can inspect whether a first-time operator understands the failure and retry path.

## Slice 9 Result

Implemented:
- `AnalysisStagePanel` now renders a compact `Live diagnostics` panel inside failed-run recovery.
- The panel derives shard state, focused repair schedule/completion/exhaustion, stall pressure, provider refs, terminal validation excerpt and raw-output refs from existing selected-run logs and artifacts.
- Partial collect failures are deduplicated by `shard_id`, so repeated error/repair/stall rows for one shard do not inflate failed-shard counts.
- Recovery actions now distinguish failed shard inspection, provider/artifact readiness and raw-output stdout/stderr comparison before retry.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `67 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`83` UI tests, `230` Python tests)

Observed improvement:
- A failed live Analysis run can now show `1/2 ok`, focused repair exhaustion, pre-artifact stall pressure, `collect_pair_repair`, `runtime_stalled_before_artifacts` and raw-output metadata in one recovery block.
- First-time operators get a triage summary before opening the activity drawer or external execution report.

Residual UX work after Slice 9:
- Re-run the medium live matrix from a clean committed tree and inspect whether the new failed-run diagnostics explain the qwen collect failures well enough.
- Provider permission approval/denial UI remains a separate future non-happy-path slice when live evidence produces representative pending permission requests.

## Slice 10 Live Rerun Attempt

Implemented:
- No product/UI change. This was a live-gate preflight attempt after Slice 9.
- Confirmed the repo worktree was clean and provider binaries were available.
- Confirmed the canonical direct command remained `regres long` qwen-only baseline through `scripts/full-run-batch-matrix.sh`.

Observed evidence:
- Host root `/tmp/provenarch-live-e2e` was absent, so the run required restoring canonical PostHog path input.
- PostHog restore could not complete on this machine: Git refetch failed with `No space left on device`; `/tmp` was at `100%` with `116MiB` free at failure.
- The temporary partial checkout was removed; `/tmp/provenarch-live-e2e` is absent again and matrix execution did not start.

UX conclusion:
- Slice 9 is ready for the next medium run, but this host cannot currently satisfy the canonical PostHog path prerequisite.
- This is an operational trusted-host blocker, not evidence against the new `analysis-live-diagnostics` UI.

## Slice 11 Result

Implemented:
- Added a `Permission triage` panel to `Analysis -> Pending permissions` for managed-mode runtime permission stops.
- The panel summarizes blocked step, operation, decision, policy rule, primary target/reason and safe next actions before the raw request table.
- Existing pending permission request data remains the only input; there is no backend/API/schema change and no new approve/deny broker.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `67 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `83` UI tests, Vite build and Go build)

Observed improvement:
- First-time operators no longer have to infer the meaning of `needs_user`, `ask_unsafe_operation` and `path_or_command` from a raw table.
- The recovery path is explicit: inspect command/path and reason, choose the intended permission mode/channel in Readiness, then retry the failed pipeline only when the request is expected.

Residual UX work after Slice 11:
- Medium live rerun remains blocked on the current host until the canonical PostHog checkout can be restored with enough `/tmp` space.
- Approve/deny broker remains future scope; the current UI makes the stop understandable but does not resolve requests in-place.

## Slice 12 Result

Implemented:
- Updated the right inspector hard-blocker copy for pending runtime permissions.
- Pending permission blockers now show `Permission: <action>` plus blocked step, decision/rule, target and reason.
- Existing pending permission request data remains the only input; there is no backend/API/schema change.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `67 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `83` UI tests, Vite build and Go build)

Observed improvement:
- The global hard-blocker surface now matches the dedicated `Permission triage` panel instead of reducing the issue to a generic pending message.
- A first-time operator can understand the permission stop from the right inspector even before scrolling to the pending permissions table.

Residual UX work after Slice 12:
- Medium live rerun remains blocked on the current host until the canonical PostHog checkout can be restored with enough `/tmp` space.
- Approve/deny broker remains future scope.

## Slice 13 Result

Implemented:
- Added active-run cancellation guidance to the shared active run strip.
- The hint appears only for selected queued/running runs and states that cancel requests a cooperative stop while taskrun evidence stays in History.
- Existing cancel API behavior remains unchanged; this is UI copy and layout only.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run src/components/ConsoleShellPrimitives.test.tsx` -> `10 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `84` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator can distinguish stopping the active run from deleting evidence or losing audit history.
- Succeeded review-ready runs remain focused on evidence review and do not show cancellation guidance.

Residual UX work after Slice 13:
- Medium live rerun remains blocked on the current host until the canonical PostHog checkout can be restored with enough `/tmp` space.

## Slice 14 Result

Implemented:
- Distinguished terminal `run_canceled` runs from runtime/provider failures in Analysis recovery.
- Canceled runs now show `Canceled run`, `Stopped step`, `Run <pipeline> again`, retained taskrun History evidence and a secondary `Review retained evidence` action.
- Right inspector and Publish blockers now use `Canceled run` / restart-reconciled copy instead of generic `Selected run failed` wording.
- Existing cancel API/runtime behavior remains unchanged; this is UI copy/state classification only.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `68 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `85` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator can understand that a canceled terminal run was stopped intentionally, not that the provider crashed.
- The post-cancel screen tells the operator where evidence remains and how to start a fresh run.

Residual UX work after Slice 14:
- Medium live rerun remains blocked on the current host: `/tmp/provenarch-live-e2e` is absent and `/tmp` has about `3.4GiB` free, below the 5 GiB matrix guard.

## Slice 15 Result

Implemented:
- Extended terminal `run_canceled` / restart-reconciled semantics to Ask/Q&A recovery without changing the async QA API or runtime contracts.
- Ask run history and selected run status now display `canceled` / `recovered` outcomes for terminal failed QA runs with those classifications instead of raw generic `failed`.
- The Ask recovery panel now shows `Canceled answer run` / `Recovered answer run`, `Stopped step` / `Recovered step`, retained `reports/taskruns/<run_id>/qa/` audit evidence and an `Ask again` action.
- Generic answer validation/runtime failure guidance remains unchanged for `runtime_contract_failed`, `runner_unavailable`, `runtime_timeout` and permission blockers.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `69 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed

Observed improvement:
- A first-time operator who cancels an Ask run can distinguish an intentional stop from a provider crash or invalid `qa-answer.json`.
- The recovery surface keeps the audit trail visible and makes the next action explicit: ask again when ready.

Residual UX work after Slice 15:
- Medium live rerun remains blocked on the current host: `/tmp/provenarch-live-e2e` is absent and `/tmp` has about `3.3GiB` free, below the 5 GiB matrix guard.

## Slice 16 Result

Implemented:
- Added shared run outcome helpers so terminal `run_canceled` and restart-reconciled failed runs render as `canceled` / `recovered`.
- Analysis header, run mission control, run status panel, active run strip and History now use those outcome labels instead of raw generic `failed`.
- History separates `Failed`, `Canceled` and `Recovered` counts while generic runtime/provider failures remain `failed`.
- Reused the same helper for Ask/Q&A run history and selected-run status so Analysis and Ask stay consistent.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx ConsoleShellPrimitives.test.tsx` -> `79 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `86` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator no longer sees a canceled or restart-reconciled run as a red generic runtime failure in the main Analysis status surfaces.
- The active run strip now labels terminal canceled/recovered runs with `Stopped step` / `Recovered step`, matching the recovery panel language.
- Run History makes operational stops auditable without inflating the generic failure count.

Residual UX work after Slice 16:
- Medium live rerun remains blocked on the current host: `/tmp/provenarch-live-e2e` is absent and `/tmp` remains below the 5 GiB matrix guard.
- Next deterministic slice should continue across remaining non-happy-path surfaces, especially activity drawer failure summaries and any retry states that still collapse user intent into generic failure copy.

## Slice 17 Result

Implemented:
- Made the shared Activity drawer outcome-aware for terminal `run_canceled` and restart-reconciled runs.
- The drawer summary now shows `canceled run` / `recovered run` instead of raw `failed run` when the selected run has those terminal classifications.
- Empty-log recovery copy now explains retained History evidence for canceled runs and restart reconciliation for recovered runs; generic runtime/provider failures still say `Run failed before log entries`.
- Split Activity drawer inputs so `error_code` drives classification while human error text remains separate.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx ConsoleShellPrimitives.test.tsx` -> `79 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `86` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator who opens Activity after a canceled/reconciled terminal run no longer sees the stop described as a provider/runtime crash.
- The shared shell now uses the same canceled/recovered language as Analysis recovery, History and the active run strip.

Residual UX work after Slice 17:
- Medium live rerun remains blocked on this host: canonical checkout root is absent and `/tmp` remains below the 5 GiB matrix guard.
- Continue deterministic polish on remaining retry states that still use generic failure copy or hide the next action behind raw logs.

## Slice 18 Result

Implemented:
- Made Analysis recovery actions retained-evidence aware for restart-reconciled runs, matching the canceled-run path.
- Terminal canceled/recovered Analysis runs now use `Run <pipeline> again` and `Review retained evidence` instead of generic retry/blocker wording.
- The global right-inspector next action now says `Review retained run evidence` for terminal canceled/recovered selected runs instead of `Review blocker`.
- Generic runtime/provider failures still keep the existing `Retry <pipeline>` and blocker-review language.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `69 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `86` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator who sees a restart-reconciled run no longer gets generic failed-run actions.
- The recovery panel and global inspector now consistently frame stopped/reconciled terminal runs as retained evidence plus a fresh run, not as a provider crash to retry blindly.

Residual UX work after Slice 18:
- Medium live rerun remains blocked on this host: canonical checkout root is absent and `/tmp` remains below the 5 GiB matrix guard.
- Continue deterministic polish on remaining publish/onboarding/provider-unavailable recovery states while waiting for a trusted host that can run the medium live matrix.

## Slice 19 Result

Implemented:
- Distinguished `runner_unavailable` from generic runtime failures in global blocker copy, Analysis recovery, live diagnostics, Ask recovery guidance and Publish gate copy.
- The global right-inspector next action now says `Check provider readiness` and opens `Readiness` without starting a new analysis, so quota/auth/binary outages are checked before retry.
- Analysis live diagnostics now labels provider outages as `provider check` and tells operators to verify Readiness provider setup, binary/auth/quota instead of treating the outage as a shard-quality failure.
- Existing run status, logs, artifacts and error codes remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `70 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- fake-runtime rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:49994 UI_E2E_QA_SMOKE=0 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-provider-recovery-results.ijdm004g ./scripts/run-npm.sh run --prefix ui e2e:live` -> `1 passed` (`run_20260708_194304_001`)
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `87` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator who hits provider quota/auth/toolchain outage now sees the correct recovery path: check provider readiness first, then retry the same pipeline.
- Publish no longer presents a provider outage as an opaque failed run before publication.
- Ask uses the same provider-readiness language as Analysis, reducing cross-stage terminology drift.

Residual UX work after Slice 19:
- Medium live rerun remains blocked on this host: canonical checkout root is absent and `/tmp` remains below the 5 GiB matrix guard.
- Continue deterministic polish on onboarding/provider-selection and any remaining retry states while waiting for a trusted host that can run the medium live matrix.

## Slice 20 Result

Implemented:
- Added a dedicated `Provider readiness recovery` panel to Readiness for headless runtime, `runner_unavailable` reroutes and runtime-provider doctor failures.
- The panel shows selected provider, doctor status/message, suggested fix, command override env var, expected command and the last run blocker before retry.
- The global inspector route from provider-unavailable Analysis opens Readiness without starting `init` or `refresh`.
- Existing selected run status and doctor response remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `70 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `87` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator who lands in Readiness from a provider outage now sees binary/auth/quota recovery in the first viewport instead of generic readiness cards only.
- The recovery surface explains the exact env override (`ACP_CODEX_CMD`, `ACP_QWEN_CMD` or `ACP_CLAUDE_CMD`) and doctor suggestion before telling the operator to retry Analysis.
- The retry path is safer because the Readiness action does not accidentally start a new run.

Residual UX work after Slice 20:
- Medium live rerun remains blocked on this host: `/tmp/provenarch-live-e2e` is absent and `/tmp` remains below the 5 GiB matrix guard.
- Continue deterministic polish on first-run provider selection and remaining retry/error states while waiting for a trusted host that can run the medium live matrix.

## Slice 21 Result

Implemented:
- Added an onboarding `Provider setup for first analysis` recovery block for headless runner setup and runtime-provider doctor failures.
- The block shows selected provider, expected executable, `ACP_*_CMD` command override, readiness check status, doctor message/suggestion and a safe fallback to `fake` for the deterministic first walkthrough.
- Reused the same provider command/env guidance as Readiness so onboarding and Console V2 do not drift.
- Existing onboarding status, selected runtime/provider state and doctor response remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `70 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- rendered onboarding check: Playwright opened launcher mode, switched Runner to `headless`, asserted `onboarding-runner-recovery` content and no horizontal overflow on `1440x980` and `390x900`; screenshots: `/tmp/provenarch-ui-onboarding-runner-manual.laqv2_2p/onboarding-runner-recovery-desktop.png`, `/tmp/provenarch-ui-onboarding-runner-manual.laqv2_2p/onboarding-runner-recovery-mobile.png`
- fake-runtime rendered smoke: `UI_E2E_BASE_URL=http://127.0.0.1:51846 UI_E2E_QA_SMOKE=0 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-onboarding-runner-results.hi5mptcr ./scripts/run-npm.sh run --prefix ui e2e:live` -> `1 passed` (`run_20260708_195749_001`)
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `87` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator opting into a live provider no longer has to infer `claude`, `qwen`, `codex` or `ACP_*_CMD` from raw doctor text before the first analysis.
- The Runner step now makes the safe fallback explicit: use `fake` for the first deterministic walkthrough if live auth/quota/binary setup is not ready.
- The first-run path now has provider recovery in both places a new user can encounter it: onboarding before Console V2 and Readiness after a provider-unavailable run.

Live medium status:
- Current preflight remains `operational_host_preflight_failed`: `/tmp/provenarch-live-e2e` is absent, canonical PostHog/FTGO path inputs are absent, and `/tmp` has about `2.95GiB` free, below the 5 GiB matrix guard.
- No canonical matrix, curated repos file or wrapper script was changed.

Residual UX work after Slice 21:
- Medium live rerun requires a trusted host or volume with enough space for canonical PostHog/FTGO checkouts.
- Continue deterministic polish on remaining retry/error states while live medium is operationally blocked.

## Slice 22 Result

Implemented:
- Added a dedicated `Source validation recovery` panel above the Source repo table and raw validation result.
- The panel shows affected repo/workspace scope, diagnostic, source type, current source, ref, message, suggested fix and save-then-validate next actions.
- Draft source errors such as missing repo name, duplicate repo name or missing Git URL/local path now surface immediately before saving; server-side `ValidateResponse` diagnostics remain the source of truth after validation.
- Existing Source repo form state, `ValidateResponse` and diagnostics remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `71 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- rendered Source recovery check: Playwright opened direct fake Console, cleared `Local checkout path`, asserted `source-validation-recovery` copy and no horizontal overflow on `1440x980` and `390x900`; screenshots: `/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-source-recovery-manual.Z9MEYk/source-recovery-desktop.png`, `/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-source-recovery-manual.Z9MEYk/source-recovery-mobile.png`
- fake-runtime rendered smoke: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin UI_E2E_BASE_URL=http://127.0.0.1:51848 UI_E2E_QA_SMOKE=0 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-source-recovery-results.sAner5at ./scripts/run-npm.sh run --prefix ui e2e:live` -> `1 passed` (`run_20260708_201043_001`)
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `88` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator who breaks a repo source now sees the exact Source recovery path before scrolling to raw `workspace.yaml` diagnostics.
- The Source stage now matches Readiness/Analysis/Ask recovery quality: blocking state, affected scope and next action are visible in the first workbench viewport.
- Git URL/local folder problems are no longer presented as only a table badge plus low-level validation text.

Live medium status:
- Current preflight remains `operational_host_preflight_failed`: `/tmp/provenarch-live-e2e/posthog/posthog` and `/tmp/provenarch-live-e2e/ftgo/ftgo-application` are absent, and `/tmp` has about `3.09GiB` free, below the 5 GiB matrix guard.
- No canonical matrix, curated repos file or wrapper script was changed.

Residual UX work after Slice 22:
- Medium live rerun requires a trusted host or volume with enough space for canonical PostHog/FTGO checkouts.
- Continue deterministic polish on remaining retry/error states while live medium is operationally blocked.

## Slice 23 Result

Implemented:
- Added a dedicated `Charter baseline recovery` panel above the Charter workbench when baseline bundle diagnostics are present.
- The panel shows affected artifact, category, runtime use (`live consumed`, `reference only`, `charter context`), diagnostic code, severity, message, suggested fix and exact `Save selected baseline artifact` next action.
- The panel maps diagnostics to editable artifact metadata using existing `BaselineBundleResponse.warnings` and `editable_artifacts`; raw warning lines remain in the editor for detailed inspection.
- Existing baseline bundle diagnostics and editor artifact metadata remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `npm --prefix ui test -- --run App.test.tsx` -> `72 passed`
- UI typecheck: `npm --prefix ui run typecheck` -> passed
- rendered Charter recovery check: Playwright opened the Vite UI with mocked `/api/*`, asserted `charter-baseline-recovery` content and no horizontal overflow on `1440x980` and `390x900`; screenshots: `/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-charter-recovery-manual.zN6vmE/charter-recovery-desktop.png`, `/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-charter-recovery-manual.zN6vmE/charter-recovery-mobile.png`
- fake-runtime rendered smoke: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin UI_E2E_BASE_URL=http://127.0.0.1:51850 UI_E2E_QA_SMOKE=0 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-charter-recovery-results.dNBmXDjg ./scripts/run-npm.sh run --prefix ui e2e:live` -> `1 passed` (`run_20260709_062422_001`)
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `89` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator who lands on Charter with prompt/charter bundle diagnostics no longer has to infer impact from raw warning lines below the fold.
- The Charter stage now explains whether the affected file is live-consumed runtime context or reference-only seed content before Analysis.
- Recovery copy names the exact editor action and keeps Git review in the same stage.

Live medium status:
- Current preflight remains `operational_host_preflight_failed`: `/tmp/provenarch-live-e2e` is absent, canonical PostHog/FTGO path inputs are absent, and `/tmp` has about `3.19GiB` free, below the 5 GiB matrix guard.
- No canonical matrix, curated repos file or wrapper script was changed.

Residual UX work after Slice 23:
- Medium live rerun requires a trusted host or volume with enough space for canonical PostHog/FTGO checkouts.
- Continue deterministic polish on remaining retry/error states while live medium is operationally blocked.

## Slice 24 Result

Implemented:
- Added a dedicated `Proposal package recovery` panel above the Proposals review room when proposal/changelog package blockers are present.
- The panel shows package state, proposal doc count, ADR/RFC count, changelog count, evidence refs, primary blocker, suggested fix and publication path before the operator reaches Publish.
- Changelog-only runs keep the changelog preview selected, but now clearly explain that `proposals/*` artifacts are missing before publication review.
- Existing selected run artifact refs and open questions remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin ./scripts/run-npm.sh --prefix ui test -- --run App.test.tsx` -> `72 passed`
- UI typecheck: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin ./scripts/run-npm.sh --prefix ui run typecheck` -> passed
- rendered Proposals recovery check: Playwright opened the Vite UI with mocked `/api/*`, asserted `proposal-package-recovery` content and no horizontal overflow on `1440x980` and `390x900`; screenshots: `/tmp/provenarch-ui-proposals-recovery-manual.TNgFwb/proposal-recovery-desktop.png`, `/tmp/provenarch-ui-proposals-recovery-manual.TNgFwb/proposal-recovery-mobile.png`
- fake-runtime rendered smoke: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin UI_E2E_BASE_URL=http://127.0.0.1:51852 UI_E2E_QA_SMOKE=0 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-proposals-smoke-results.eUZV5i ./scripts/run-npm.sh run --prefix ui e2e:live` -> `1 passed` (`run_20260709_064232_001`)
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `89` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator who lands on Proposals with only a changelog or partial package no longer has to infer publication risk from the side blocker list.
- The Proposals stage now matches Source/Readiness/Charter recovery quality: the first viewport names the blocker, counts missing package pieces and states the safe path before Publish.
- The existing changelog preview remains useful evidence while the missing proposal package is explicit.

Live medium status:
- Current preflight remains `operational_host_preflight_failed`: `/tmp/provenarch-live-e2e` is absent and canonical PostHog/FTGO path inputs are absent. `/tmp` currently has more than the 5 GiB guard available, but the required checkouts are not prepared.
- No canonical matrix, curated repos file or wrapper script was changed.

Residual UX work after Slice 24:
- Medium live rerun requires the canonical `/tmp/provenarch-live-e2e/posthog/posthog` and `/tmp/provenarch-live-e2e/ftgo/ftgo-application` checkouts on a trusted host.
- Continue deterministic polish on remaining retry/error states while live medium is operationally blocked.

## Slice 25 Result

Live medium retry:
- goal: run the requested medium live E2E after preparing the canonical PostHog checkout.
- action: restored `/tmp/provenarch-live-e2e/posthog/posthog` as a real Git checkout at pinned ref `14d29a548d63665d60b506cf13bd5cfb2de7c743`, confirmed exact Node `v22.21.1`, provider binaries and free disk, then ran direct `scripts/full-run-batch-matrix.sh` with matrix id `regres-long-posthog-ftgo-20260709T065646Z`.
- observed evidence: `/tmp/provenarch-test_arch_project/reports/matrix_result_regres-long-posthog-ftgo-20260709T065646Z.json` is `FAIL`, non-release, `strict_pass_runs=0/2`; both `single-path/baseline` and `single-git_url/baseline` stopped before backend/frontend execution with `operational_host_preflight_failed`.
- primary blocker: `qwen: headless_probe_timeout: qwen headless probe timed out after 30s`.
- UX conclusion: this is host/provider readiness evidence, not a product backend/UI verdict. The useful product gap is that provider recovery must distinguish a text readiness probe timeout from generic binary/auth/quota failure before the operator retries Analysis.

Implemented:
- Added shared provider readiness guidance for headless probe timeout, artifact-smoke failure, auth/quota blocker and command-unavailable cases.
- Readiness `Provider readiness recovery` now shows failure mode, probe stage and operator focus next to selected provider, doctor status, command override and last run blocker.
- Onboarding `Provider setup for first analysis` uses the same guidance before a first-time operator enters Console V2.
- Existing doctor response and selected-run error text remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin npm run --prefix ui test -- App.test.tsx --run -t "provider readiness|provider probe timeouts|runner unavailable"` -> `2 passed`
- UI typecheck: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin npm run --prefix ui typecheck` -> passed
- rendered qwen-timeout check: Playwright opened the Vite UI with mocked `/api/*`, asserted `provider-readiness-recovery` content and no horizontal overflow on `1440x980` and `390x900`; screenshots: `/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-provider-timeout-manual.1wiqoc/provider-timeout-readiness-desktop.png`, `/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-provider-timeout-manual.1wiqoc/provider-timeout-readiness-mobile.png`
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `90` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator who sees `qwen headless_probe_timeout` no longer gets only generic provider setup copy.
- The screen now names the failed probe (`Text readiness probe`), explains that the bounded readiness response did not arrive, and keeps retry blocked on `Check local readiness` before Analysis.
- Onboarding and Readiness now share the same provider-recovery taxonomy, reducing terminology drift before and after Console entry.

Residual UX work after Slice 25:
- Medium live rerun remains blocked on this host by provider readiness: `qwen` headless probe timeout before backend/frontend execution.
- Continue deterministic polish on remaining retry/error states while live medium is operationally blocked by provider readiness.

## Slice 26 Result

Live medium retry:
- goal: rerun the medium live E2E after Slice 25 clarified qwen readiness timeouts.
- action: ran direct `scripts/full-run-batch-matrix.sh` from clean commit `3de030e` with matrix id `regres-long-posthog-ftgo-20260709T072237Z`.
- observed evidence: `/tmp/provenarch-test_arch_project/reports/matrix_result_regres-long-posthog-ftgo-20260709T072237Z.json` is `FAIL`, non-release, `strict_pass_runs=0/2`.
- `single-path/baseline` reached backend execution and failed in `init.step1.collect`: `partial_failure_count=10`, `repair_attempts=14`, `repair_exhausted=10`, `focused_repairs=14`, `stall_count=27`, `pre_artifact_stalls=24`, `post_artifact_stalls=3`, `valid_artifact_controlled_stops=5`, `raw_output_refs=10`, and shard summary `6 succeeded / 10 failed`.
- every failed PostHog shard had the same terminal pattern: `stage=collect_pair_repair`, `collect pair recovery stalled before valid artifacts were available`, `runtime_stalled_before_artifacts`.
- `single-git_url/baseline` failed before backend/frontend as `operational_host_preflight_failed`: `qwen: headless_probe_timeout: qwen headless probe timed out after 30s`.

Implemented:
- Analysis live diagnostics now parse recovery stage from either structured fields or `stage=...` log text.
- The panel classifies this live pattern as `Artifact handoff stalled` instead of only showing generic failed shard counts.
- The summary explains that collect repair was reached, but valid shard artifacts were not written before the pre-artifact stall.
- Next actions now tell the operator to open the failed shard row and raw-output ref, then confirm whether both authored markdown and `shard-pack-manifest.json` were written before retrying or switching provider.
- Existing run logs, artifacts and selected-run status remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `PATH=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin:$PATH npm --prefix ui test -- --run App.test.tsx -t "renders Analysis V2 run progress"` -> `1 passed`
- full UI suite: `PATH=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin:$PATH npm --prefix ui test -- --run` -> `90 passed`
- UI typecheck: `PATH=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin:$PATH npm --prefix ui run typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `90` UI tests, Vite build and Go build)

Observed improvement:
- A first-time operator no longer has to infer from `repair exhausted`, `stall pressure` and raw-output refs that the provider failed to hand off the required artifact pair.
- The Analysis recovery panel now names the failure mode, the failed recovery stage and the exact artifact pair to verify.
- Mixed collect outcomes are clearer: successful shards can be understood as durable evidence while failed shards are specifically artifact-handoff failures.

Residual UX work after Slice 26:
- The execution blocker is still runtime/provider behavior, not a visual UI blocker: qwen produced 10 PostHog collect artifact-handoff failures and later timed out readiness for the FTGO profile.
- The next iteration should decide whether to improve runtime/provider prompt behavior or add deeper per-shard artifact-pair inspection in the UI.

## Slice 27 Result

Implemented:
- Added per-shard artifact-pair state to the Analysis shard/log table.
- The table and blocker drilldown now distinguish `Runtime only`, `Markdown only`, `Manifest only`, `Artifact pair present` and missing selected-run shard artifacts.
- The blocker drilldown shows separate refs for runtime execution metadata, authored markdown and `shard-pack-manifest.json`, so a first-time operator can verify the collect handoff without inferring it from raw logs.
- Existing selected-run artifact refs and run logs remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin npm run --prefix ui test -- App.test.tsx --run` -> `73 passed`
- full UI suite: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin npm run --prefix ui test -- --run` -> `90 passed`
- UI typecheck: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin npm run --prefix ui typecheck` -> passed
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `90` UI tests, Vite build and Go build)

Observed improvement:
- A failed collect shard with only `runtime-execution.json` now reads as `Runtime only` and explicitly says authored markdown plus manifest are missing.
- A neighboring shard with markdown and manifest reads as `Artifact pair present`, making mixed collect results easier to scan.
- The retry decision is clearer: raw output remains useful for provider diagnosis, but the operator can first check whether the required authored artifact pair exists.

Residual UX work after Slice 27:
- Rendered failed-run visual QA should be added when a stable mocked failed-run browser fixture is available; current verification is component-level plus DoD.
- The execution blocker remains runtime/provider behavior: qwen must reliably write the collect artifact pair before the medium live matrix can pass.

## Slice 28 Result

Live medium retry:
- goal: rerun medium `regres long` qwen-only after Slice 27 from a clean committed tree.
- action: confirmed `/tmp/provenarch-live-e2e` writable, PostHog checkout pinned at `14d29a548d63665d60b506cf13bd5cfb2de7c743`, exact Node `v22.21.1`, `qwen 0.17.1`, then ran direct `scripts/full-run-batch-matrix.sh` with matrix id `regres-long-posthog-ftgo-20260709T094207Z`.
- observed evidence: `single-path/baseline` stopped before backend/frontend as `operational_host_preflight_failed` because `qwen headless probe timed out after 30s`; reports exist under `/tmp/provenarch-test_arch_project/reports/*regres-long-posthog-ftgo-20260709T094207Z-single-path-baseline*`.
- observed evidence: `single-git_url/baseline` reached FTGO headless init collect; after repeated artifact-handoff stalls the diagnostic run was manually interrupted and classified as `infra_signal_terminated`, not a release verdict. Public shard summary at `/tmp/provenarch-test_arch_project/runs/regres-long-posthog-ftgo-20260709T094207Z-single-git-url-baseline/qwen-code/run1/headless/arch-workspace/reports/taskruns/run_20260709_094918_001-init-step1-collect-shard-summary-ftgo-application.json` ended `15 failed / 1 succeeded`.
- observed evidence: first four FTGO failures were genuine `stage=collect_pair_repair` / `runtime_stalled_before_artifacts`; the only succeeded shard had authored markdown + `shard-pack-manifest.json` + runtime metadata. Later `context canceled` failures came from the deliberate stop and are not counted as product defects.
- status: failed diagnostic / interrupted after repeated evidence.
- primary classification: runtime_contract_failed for repeated qwen collect artifact handoff, plus operational_host_preflight_failed for PostHog provider readiness; final FTGO profile status is `infra_signal_terminated` because of the manual stop.
- next decision: fix the UI mapping bug exposed by the live evidence, then rerun deterministic UI verification and full DoD.

Implemented:
- Fixed Analysis shard/log grouping to prefer `fields.shard_id` or `staging/shards/<shard_id>` path segments over `domain_id` when matching selected-run artifacts.
- Grouped same-shard runtime, repair and terminal log rows together even when later repair rows do not carry `taskrun_path`.
- Preserved shard duration from the latest grouped log entry that actually contains duration fields, so terminal repair messages do not erase useful timing.
- Added a live-shaped regression fixture where `domain_id=ftgo-application` but artifacts are scoped by distinct shard IDs.

Post-change evidence:
- targeted component test: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin npm run --prefix ui test -- App.test.tsx --run -t "renders Analysis V2 run progress"` -> `1 passed`
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `90` UI tests, UI typecheck, Vite build and Go build)

Observed improvement:
- Real medium-run rows like `ftgo-application-gitattributes-...` now map to `Runtime only` instead of losing artifact-pair state under the broader `ftgo-application` domain.
- The one successful FTGO shard maps to `Artifact pair present`, making mixed collect outcomes visible without reading raw output.
- The blocker drilldown now keeps the terminal repair message while retaining the earlier runtime record and duration for the same shard.

Residual UX work after Slice 28:
- Add rendered failed-run visual QA around this exact `domain_id != shard_id` fixture.
- The execution blocker remains runtime/provider behavior: qwen must reliably write collect markdown + manifest before the medium matrix can pass.

## Slice 29 Result

Live medium retry:
- goal: rerun medium `regres long` qwen-only after Slice 28 from clean commit `c2c1633`.
- action: confirmed `/tmp/provenarch-live-e2e` writable, PostHog checkout pinned at `14d29a548d63665d60b506cf13bd5cfb2de7c743`, FTGO git URL preset pinned at `558dfc53b11d30a5f1d995c0c6d58d5106c28189`, exact Node `v22.21.1`, `qwen 0.17.1`, then ran direct `scripts/full-run-batch-matrix.sh` with matrix id `regres-long-posthog-ftgo-20260709T104135Z`.
- observed evidence: `single-path/baseline` stopped before backend/frontend as `operational_host_preflight_failed` because `qwen headless probe timed out after 30s`; reports exist under `/tmp/provenarch-test_arch_project/reports/*regres-long-posthog-ftgo-20260709T104135Z-single-path-baseline*`.
- observed evidence: `single-git_url/baseline` passed provider preflight, bounded precheck, fake init and fake refresh, then reached FTGO qwen init collect. The first collect shard failed as `runtime_contract_failed` with `stage=collect_pair_repair` and `runtime_stalled_before_artifacts`; the public shard summary had `1 failed / 15 pending` when enough diagnostic evidence had been captured.
- observed evidence: the selected run log grew to multi-megabyte `runtime_output` JSON stream chunks before a durable authored artifact pair appeared; the run was manually interrupted and therefore final status is `infra_signal_terminated`, not a release verdict.
- UX conclusion: while the active run is still streaming, a first-time operator needs a compact "provider stream active, artifact pair pending" state instead of interpreting raw provider chunks as collect progress.

Implemented:
- Analysis now renders `Live diagnostics` for an active selected run when live telemetry is present, not only after terminal failure.
- Runtime output JSON stream events are summarized as provider stream chunk count, JSON stream event count and signal types.
- A running collect step with provider stream telemetry but no authored markdown or `shard-pack-manifest.json` is classified as `provider stream` / `Artifact pair pending`.
- Next actions tell the operator to wait for authored markdown plus manifest before treating collect as complete, and to use raw-output metadata instead of reading the full provider stream if collect stalls or repair starts.
- Existing run logs, selected-run artifacts and run status remain the only inputs; no backend/API/schema/runtime contract changed.

Post-change evidence:
- targeted component test: `PATH=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin:$PATH ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin npm run --prefix ui test -- App.test.tsx --run -t "surfaces active provider stream"` -> `1 passed`
- full DoD: `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts test lint build` -> passed (`230` Python tests, `91` UI tests, UI typecheck, Vite build and Go build)

Observed improvement:
- A live FTGO/qwen operator no longer has to infer active collect state from multi-megabyte raw provider stream logs.
- The Analysis screen now separates "provider is talking" from "collect artifact pair exists", which is the critical decision boundary before retry or wait.
- Terminal failed-shard triage from Slices 26-28 remains intact; Slice 29 fills the pre-terminal active-run state.

Residual UX work after Slice 29:
- Add rendered browser visual QA for active provider stream diagnostics once a stable mocked running-run fixture is available.
- The execution blocker remains runtime/provider behavior: qwen must reliably write collect markdown + manifest before the medium matrix can pass.
