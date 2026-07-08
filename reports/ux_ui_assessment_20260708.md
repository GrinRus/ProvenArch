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
- Keep advanced runtime settings collapsed by default outside explicit operator interaction or warnings.
- Rebalance Review internals so `Needs review` queue and selected preview are primary, while full artifact explorer is secondary.
- Add mobile section jump controls for Review/Publish if the page still feels too long after shell disclosure changes.

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
- Readiness still exposes advanced runtime settings too prominently during the live smoke path.
- Review still shows queue and full artifact explorer with similar visual weight; a later slice should make the queue the primary lane and put the full explorer behind a secondary disclosure/tab.
- Mobile Review remains long even after shell compaction; add section jump controls if next visual pass still shows navigation friction.
