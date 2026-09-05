# UX Current State - 2026-07-07

> Archived 2026-09-05 from `reports/ux_current_state_20260707.md`. This is a dated UX/QA evidence log, not the current product specification or release verdict. Use the [current task-first UX](../../UI_TASK_FIRST_PRODUCT_DESIGN.md) and [current architecture](../../ARCHITECTURE.md) and [status matrix](../../STAKEHOLDER_DOC.md). Historical commands and local evidence paths below are preserved; `/tmp` captures are expired or untracked and private `/Users` paths are nonportable. Their availability is not a prerequisite or a claim of this archive.

Diagnostic pass over Console V2. This is UX/live-smoke evidence only, not release readiness and not a machine release verdict.

## Smoke Evidence

goal: verify the rendered local console can complete the primary fake-runtime operator flow.
action: ran embedded UI build, started `acp serve` against a temporary workspace, then ran `ui/e2e/live-flow.spec.ts` with Ask smoke enabled.
observed evidence:
- build: `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-v22.21.1/bin make build`
- server: `./bin/acp serve --workspace /tmp/provenarch-ui-ux-smoke-workspace.J6Js01 --auto-init --repo-name provenarch --repo-path <repo> --runtime fake --listen 127.0.0.1:18180`
- Playwright: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-ux-smoke-results-20260707T140758Z npm run --prefix ui e2e:live`
- init run: `run_20260707_140931_001`
- QA run: `run_20260707_140936_002`
- screenshots: `/tmp/provenarch-ui-ux-smoke-results-20260707T140758Z/frontend-*.png`
- workspace artifacts: `/tmp/provenarch-ui-ux-smoke-workspace.J6Js01/reports/taskruns/`
status: passed
primary classification: none
next decision: use as current-state UX evidence; do not treat as release gate evidence.

## Host Notes

- Initial host preflight found PATH Node.js `25.9.0`, while `.node-version` requires `22.21.1`.
- For this diagnostic run, exact Node.js `22.21.1` was downloaded to `/tmp/provenarch-node-v22.21.1` and passed `scripts/resolve-node-tool.sh`.
- This `/tmp` toolchain is acceptable for local diagnostic evidence only. Release evidence still needs a stable exact toolchain path.

## Product State

- Build passed with embedded UI bundle.
- Playwright live UI smoke passed: `1 passed`.
- Init run status: `succeeded`; `failure.class=none`; `report_mode=normal`.
- Collect evidence: `planned_shards=10`, `succeeded_shards=10`, `failed_shards=0`.
- QA flow status: `succeeded`; fake runtime returned citations from workspace artifacts.
- Run warnings were non-frontend blockers:
  - missing wizard contract fallback for baseline constitution support artifacts;
  - structural shard coalescing reduced 27 roots to 10 shard groups;
  - structural shard coalescing preserved one module marker leaf group.

## UX Findings

1. The console is functionally complete but visually over-dense at 1280px.
   Top status, active run strip, stage rail, right inspector and open Activity drawer compete for first-screen attention. The work area often has less usable height than the surrounding chrome.

2. The Activity drawer is too prominent during review tasks.
   It is valuable during a run, but after success it remains open in the captured flow and consumes about one quarter of the viewport. Review, Ask and Publish screens lose their primary content above the fold.

3. Right inspector sections are always expanded.
   `Hard blockers`, `Review warnings`, `Open questions`, `Evidence refs`, `Workspace health`, `Runtime safety` and `Git publication` are useful, but empty or secondary sections repeat across every stage. This dilutes the actual next action.

4. Stage completion semantics are hard to scan.
   The rail mixes checkmarks, dots and numeric counts. `Review 19` is useful, but `Ask` remains visually pending after a successful QA run, and counts are not explained in place.

5. Readiness exposes advanced runtime settings too early.
   The smoke needs them for coverage, but for an operator this pushes the core readiness result down and makes the first actionable decision less obvious.

6. Review artifact browsing works, but the queue and explorer feel like separate inventories.
   The selected preview is readable, yet the left-side review queue, artifact explorer and right-side evidence refs overlap conceptually. Operators need a clearer "what needs attention now" lane.

7. Publish action is clear, but the diff summary and preview hierarchy is cramped.
   The main commit action is discoverable in the inspector. The center preview is useful, but the left diff column and right inspector make the page feel narrower than the task requires.

8. Mobile is usable but very long.
   The rail collapses and content wraps without obvious overlap. However Review becomes a long sequence of preview, queue, explorer, citations, inspector and activity sections; the operator loses local navigation.

## Proposed Fixes

P0/P1:
- Make Activity drawer state context-aware: auto-collapse after a run succeeds, keep it open for active/failed runs, and show a compact "last event + open logs" summary when collapsed.
- Add progressive disclosure to the right inspector: keep `Next action` pinned, collapse empty sections by default, and group secondary health/safety/publication under a compact `Details` area.
- Split the active run strip into "execution status" and "review state" emphasis: after success, surface `Review required`, warning count and artifact count instead of keeping `current_step=init.step4.proposals` as the dominant line.

P1:
- Rework Review into a three-part task flow: `Needs review` queue, `Selected artifact`, `Evidence/trust`. Avoid showing both queue and full artifact explorer with equal visual weight at the same time.
- In Readiness, keep advanced runtime settings collapsed unless there is a runtime warning, headless mode, or explicit operator interaction.
- Give stage rail statuses stronger labels/tooltips: explain counts, mark Ask done after successful QA, and reduce decorative status glyph ambiguity.

P2:
- Add mobile in-page jump controls for Review and Publish (`Preview`, `Queue`, `Artifacts`, `Inspector`, `Activity`) or sticky section tabs after the rail.
- Reduce repeated card chrome in inspector and repeated panels. The current style is disciplined, but there are many equal-weight bordered boxes competing for attention.
- Preserve the current restrained palette. Avoid a broad redesign; the strongest gains are layout/disclosure changes rather than new visual decoration.

## Recommended Next Slice

Implement a focused UX polish slice without schema/API changes:

1. Activity drawer context behavior and collapsed summary.
2. Right inspector disclosure model for empty/secondary sections.
3. Review stage hierarchy cleanup around queue vs artifact explorer.
4. Update/extend UI tests for drawer state, inspector collapsed sections, and mobile Review navigation.

Suggested verification:
- `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-v22.21.1/bin make build`
- `UI_E2E_BASE_URL=<local serve> UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=<tmp> npm run --prefix ui e2e:live`
- Add targeted React tests for shell primitives if state/disclosure logic changes.
