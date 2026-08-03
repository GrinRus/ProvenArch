# UX/UI Assessment - 2026-07-24

This report is product UX evidence for the ACP operator console. It is not release readiness evidence and does not replace `release_verdict_<matrix-id>.json`.

## Product Context

- Product: Architecture Control Plane (ACP), local-first CLI+UI tool that builds a Git-versioned, validated architecture knowledge base from one or more repositories.
- Pipeline: `operator CLI/UI -> Go orchestrator -> runtime provider (fake | claude-code | qwen-code | codex-code) -> staged artifacts -> validator -> arch-workspace files`.
- Primary users: architects, tech leads and local operators running ACP on their own machine.
- Current shell: primary navigation `Home / Runs / Knowledge / Changes`, contextual Guided Setup (`Workspace -> Sources -> Analysis brief -> Runner & readiness -> Review & start`), global read-only `Ask`, Git mutations only in `Publish`.
- Quality bar (from `docs/UI_CONSOLE_V2_DESIGN.md`): a first-time user understands the next action, current blocker, generated evidence and recovery path without reading raw logs first.
- Prior UX lineage: `reports/ux_current_state_20260707.md` (shell density findings), `reports/ux_ui_assessment_20260708.md` (33 implemented improvement slices). This assessment re-audits the redesigned shell fresh, as a black-box user.

## Evidence Inspected

- Docs: `README.md`, `docs/ARCHITECTURE.md`, `docs/UI_CONSOLE_V2_DESIGN.md`, prior UX reports (2026-07-07, 2026-07-08).
- Mock e2e gate: `npm run e2e:mock` (7 scenarios) — 7 passed / 0 skipped.
- Build: `ACP_NODE_TOOL_CANDIDATES=$HOME/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make build`.
- Server: `./bin/acp serve --workspace /tmp/acp-ux-audit-ws --auto-init --repo-name provenarch --repo-path <repo> --runtime fake --listen 127.0.0.1:18180`.
- Live flow: `UI_E2E_BASE_URL=http://127.0.0.1:18180 UI_E2E_QA_SMOKE=1 UI_E2E_OUTPUT_DIR=/tmp/acp-ux-audit-results npm run --prefix ui e2e:live` — 1 passed; init run `run_20260724_135822_001`, QA run `run_20260724_135827_002`.
- Black-box walkthrough: standalone Playwright script over every primary route (`/home`, `/runs`, `/runs/<id>`, `/knowledge` x4 views, `/changes` x6 views, `/setup` x5 steps, Ask flow) at 1440x980 plus narrow 1100x850 passes; per-screen axe scan, console-error and failed-request capture. 24 captures, results in `/tmp/acp-ux-audit/shots/` (`walkthrough-results.json`).
- Launcher mode: `acp serve --listen 127.0.0.1:18181` without workspace — first-run onboarding captures.
- DOM consistency probe: heading/scroll comparison across `changes` views (evidence vs findings vs diff vs proposals).

## Step Reports

goal: verify trusted-machine preflight for rendered UX evidence.
action: fail-fast host check from `acp-e2e-live-gate` skill; provider/binary versions.
observed evidence: `/tmp/provenarch-live-e2e` exists and is writable; `qwen 0.19.11`, `Claude Code 2.1.85`, `codex-cli 0.144.1`; exact Node `v22.21.1` via project toolchain dir.
status: passed
primary classification: none
next decision: continue

goal: build current product surfaces.
action: `make build` with exact Node toolchain.
observed evidence: UI bundle + `bin/acp` built clean; `acp version dev`.
status: passed
primary classification: none
next decision: continue

goal: collect deterministic rendered evidence over the real product shell.
action: fake-runtime `acp serve --auto-init` + Playwright `e2e:live` with `UI_E2E_QA_SMOKE=1`.
observed evidence: live flow passed in 12.5s; init `run_20260724_135822_001` succeeded (5/5 steps, 18 artifacts); QA `run_20260724_135827_002` succeeded; screenshots under `/tmp/acp-ux-audit-results`.
status: passed
primary classification: none
next decision: continue

goal: audit every primary screen as a first-time black-box user.
action: standalone Playwright walkthrough of 17 routes + Ask modal flow, desktop and narrow viewports, per-screen axe/console/network capture.
observed evidence: 24 captures; 0 axe violations; 0 page console errors; only navigation-aborted fetches (`ERR_ABORTED` on route change, SPA-normal); screenshots and `walkthrough-results.json` under `/tmp/acp-ux-audit/shots`.
status: passed
primary classification: none
next decision: continue

goal: audit first-run onboarding without a preselected workspace.
action: launcher-mode `acp serve` (no `--workspace`), capture entry and setup steps.
observed evidence: all paths canonize to `/setup?step=workspace`; blocker, step map, Recent workspaces with `missing` states and disabled `Open` render correctly.
status: passed
primary classification: none
next decision: continue

goal: verify that Changes tabs deliver distinct content.
action: DOM probe comparing `changes?view=evidence|findings|diff|proposals` for run `run_20260724_135822_001`.
observed evidence: evidence/findings/diff produce identical heading sets, identical text length (3267) and identical scroll height (2538) — the same Review room under three tab labels; proposals renders its own room.
status: failed
primary classification: frontend_failed
next decision: record as UX finding; final report

## UX Findings

1. **(major) Changes tabs `Evidence`, `Findings` and `Diff` render identical content.**
   The tab bar promises three different tools, but all three show the same generic Review room (queue + evidence preview). DOM probe: same headings, same text length, same scroll height. A user clicking `Diff` expects the workspace diff, `Findings` expects the findings list; both get the same screen they already saw under `Evidence`. This is a false affordance and breaks the task model of the destination.

2. **(minor) `Run mission control` uses execution wording after execution finished.**
   A succeeded run still leads with `Current step init.step4.proposals`. The regression of the old "execution wording after success" finding (fixed earlier for the active run strip) reappears in mission control. For a completed run the useful summary is review state, artifact count and warning/error count.

3. **(minor) Empty `Review blocker` bar renders as dead chrome on a succeeded run.**
   On Runs, an empty gray `Review blocker` strip is always visible. Empty states should either say "no blockers" positively or not render; the current bar reads as a disabled/broken element.

4. **(minor) Counters do not reconcile across screens.**
   Runs shows `Warnings/errors 3 / 0`, Home shows `1 open question(s)`, Changes shows `13` queue items and `18 artifacts`. None of these numbers explain themselves in place, and a user cannot tell how they relate (3 warnings vs 1 open question vs 13 queue items).

5. **(minor) Home duplicates its status message and underuses the viewport.**
   `needs review — 1 open question(s) require review` appears both in the global strip and inside the Home card. Below the single summary card the page is empty; for the "one authoritative view" claim it could carry the next-action context one level deeper (e.g. top review queue items, latest run summary).

6. **(minor) Disabled actions lack reasons in the console.**
   `Approve selected evidence` (Review) is disabled with no visible explanation. Onboarding does this well (disabled `Open` explains `missing`); the console should match that standard.

7. **(minor) Naming inconsistencies between setup steps and stage panels.**
   Setup step `Workspace` renders a page titled `Source`; the onboarding top chips show 3 steps (`Workspace/Sources/Runner`) while the body map shows 4 (`.../Ready`). Small, but it blurs the mental model of where you are.

8. **(minor) Atlas empty state is a dead end.**
   `No validated relationships are available` states a fact without why or what to do (run analysis with edges? check entity files?). Compare with Readiness health items, which cite paths and codes.

9. **(minor) Top bar leaks build noise.**
   `dev · none` (version/commit) is shown to users in dev builds. Harmless in releases, but in dogfooding it reads as broken metadata.

## Design Verdict

The current design **fundamentally holds**: the shell is coherent, navigation is learnable (4 destinations + contextual Setup + global Ask), next action and blockers are visible without raw logs, recovery and partial states are honest and specific, accessibility is clean (0 axe violations on 17 routes), and the restrained operator-console visual language matches the product. The 2026-07-07/08 density problems are resolved — Runs, Review, Publish and Ask are now genuinely usable workspaces.

It does **not** need a redesign. It needs one consistency fix (finding 1) and a polish pass on state communication (findings 2-9): counters that reconcile, empty states that speak, disabled actions that explain themselves, and tabs that keep their promises.

## Proposed Improvements

P0:
- Give `Evidence`, `Findings`, `Diff` tabs distinct content or collapse them into one `Evidence` tab. `Findings` should list findings; `Diff` should show the run/workspace diff inventory directly (it exists behind `Load full workspace diff` in Publish).

P1:
- Switch `Run mission control` to review language for terminal runs (review state, artifacts, warnings/errors), mirroring the earlier active-run-strip fix.
- Reconcile counters: one tooltip or legend per surface explaining what is counted (warnings vs open questions vs queue items vs artifacts).
- Replace the empty `Review blocker` bar with a positive `No blockers` state or hide it.
- Add disabled-reason hints to console actions (`Approve selected evidence`), reusing the onboarding pattern.

P2:
- Enrich Home with the top of the review queue / latest run summary instead of the duplicated status line.
- Align setup step names with stage titles; add `Ready` to the onboarding top chips.
- Give Atlas empty state a cause and next action; hide or stylize `dev · none` build metadata.

## Acceptance Notes For The P0 Slice

- Each Changes tab renders distinct, task-appropriate content, verified by a component test and one mock e2e assertion per tab.
- No API, schema, runtime or artifact contract changes.
- Existing mock e2e suite (7 scenarios) and live flow stay green.
