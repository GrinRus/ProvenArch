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
