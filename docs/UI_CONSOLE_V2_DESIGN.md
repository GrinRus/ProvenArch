# UI Console V2 Design Baseline

Статус: approved design direction, 2026-05-27.

Этот документ фиксирует целевое UX/UI-видение ACP Console V2. Он описывает design baseline
для будущей реализации и не утверждает, что все экраны уже реализованы в коде.

## Visual references

Approved PNG references are stored in `docs/assets/ui-console-v2/`:

- [01 Source](assets/ui-console-v2/01-source.png)
- [02 Readiness](assets/ui-console-v2/02-readiness.png)
- [03 Charter](assets/ui-console-v2/03-charter.png)
- [04 Analysis](assets/ui-console-v2/04-analysis.png)
- [05 Review - Evidence](assets/ui-console-v2/05-review-evidence.png)
- [06 Review - Domain Map](assets/ui-console-v2/06-review-domain-map.png)
- [07 Proposals](assets/ui-console-v2/07-proposals.png)
- [08 Ask](assets/ui-console-v2/08-ask.png)
- [09 Publish](assets/ui-console-v2/09-publish.png)

The PNGs are design references, not product runtime assets. Implementation should translate them
into native React/CSS components and preserve the behavior contracts below.

## Product framing

ACP UI должен ощущаться как плотная local-first operator console для доверенного pipeline:

```text
Source -> Readiness -> Charter -> Analysis -> Review -> Proposals -> Ask -> Publish
```

Основной пользователь - local operator, architect или tech lead, который запускает ACP на своей
машине, ревьюит evidence и публикует Git-версионируемые architecture artifacts.

Ключевые вопросы первого экрана:
- готов ли workspace;
- что следующий пользовательский шаг;
- что сейчас делает pipeline;
- где блокер;
- какие artifacts/evidence можно ревьюить или публиковать.

Pre-console onboarding (`Workspace -> Sources -> Runner -> Ready`) должен показывать setup summary
с current step, next action, current blocker и причинами disabled actions для `Open console` /
`Run first analysis`, чтобы первый запуск не требовал читать raw diagnostics.

## Shell contract

Все основные экраны используют один shell:

- **Top health strip**: system health, workspace path, repo count, runtime/provider, permission mode,
  Git status, current date/time.
- **Left stage rail**: ровно 8 стадий `Source`, `Readiness`, `Charter`, `Analysis`, `Review`,
  `Proposals`, `Ask`, `Publish`.
- **Central workbench**: stage-specific dense operator surface, без landing/hero layout.
- **Right inspector**: `Next action`, blockers/warnings, evidence refs, runtime safety, Git path
  в зависимости от stage.
- **Bottom activity drawer**: `Event timeline`, `Raw agent stream`, `All`, copy/download/filter
  actions where applicable.

Visual direction:
- light dense ops-console, warm white/gray surfaces, charcoal text;
- teal primary action, blue evidence links, amber warning, red blocker;
- small radius 6-8px;
- no decorative hero art, no gradients, no stock imagery;
- stable dimensions for rail, top strip, inspector and activity drawer.

## Screen inventory

### 1. Source

Purpose: configure source repositories and imported docs before readiness.

Primary action: `Save and validate`.

Must show:
- repo table with `name`, `source`, `ref`, `analysis include/exclude`, `status`;
- Git URL/local path source mode;
- docs imports path;
- advanced `workspace.yaml` surface;
- source diagnostics and Git publication path.

### 2. Readiness

Purpose: validate local prerequisites before analysis.

Primary action: `Run readiness check`.

Must show:
- grouped checks for workspace, repositories, runtime provider, permissions and artifacts;
- runtime profile summary: timeouts, execution, permissions, step providers;
- blocking vs warning distinction;
- runtime safety status.

### 3. Charter

Purpose: prepare human-owned scope, rules, domain/team cards and baseline prompts.

Primary action: `Save charter`.

Must show:
- wizard summary for project, scope, NFRs and rules;
- domain/team cards;
- markdown artifact preview/editor for `charter/*`;
- baseline prompt bundle status;
- charter readiness and Git path.

### 4. Analysis

Purpose: run and monitor `init|refresh` pipeline.

Primary action: `Review blocker` when blocked, otherwise `Run init` or `Run refresh`.

Must show:
- run id, runtime/provider and progress;
- step timeline for `init.step0.constitution` through `init.step4.proposals`;
- shard table with repo/path/provider/status/artifacts/duration;
- failed-run live diagnostics for shard counters, focused repair, stall pressure, terminal excerpt and raw-output refs;
- pending permission triage with blocked step, operation, decision, policy rule, target/reason and safe next actions;
- blocker, evidence refs and runtime safety in inspector, including permission blocker step/rule/target/reason detail;
- live logs with highlighted error/warning rows.

### 5. Review - Evidence

Purpose: review artifacts as evidence, not raw paths.

Primary action: `Approve selected evidence`.

Must show:
- grouped artifact explorer;
- markdown/diagram preview;
- citations and claim coverage;
- findings, coverage gaps and trust status;
- Git publication status.
- if the selected/latest run failed with a partial artifact set, a recovery action that opens the
  most recent successful run artifacts without hiding the failed run blocker.

### 6. Review - Domain Map

Purpose: inspect domain/service ownership, coverage and cross-repo edges.

Primary action: `Open affected evidence`.

Must show:
- domain cards with services, owners and coverage;
- dependency and cross-repo blocker edges;
- selected-domain inspector with ownership, findings and evidence refs;
- route back to evidence/proposal/Git diff.

### 7. Proposals

Purpose: review generated ADR/RFC/proposal artifacts before publication.

Primary action: `Approve proposal`.

Must show:
- proposal/changelog list;
- formatted proposal preview with diff/evidence/changelog tabs;
- linked findings and evidence coverage;
- unresolved owner/questions blockers;
- proposal branch/Git publication path.

### 8. Ask

Purpose: ask read-only architecture questions over analyzed workspace data.

Primary action: `Ask workspace`.

Must show:
- Q&A run history;
- answer with confidence, citations, unresolved assumptions and related entities/edges;
- explicit read-only runtime safety;
- note that Q&A does not mutate canonical artifacts.

### 9. Publish

Purpose: publish reviewed workspace artifacts to Git.

Primary action: `Commit selected artifacts`.

Must show:
- diff summary by workspace folder;
- selected artifact preview with `Preview`, `Diff`, `Evidence`, `Changelog`;
- publish gate, checklist, blockers, commit plan and proposal branch;
- prepared commit message actions.

## State model

Every stage must have explicit surfaces for:
- loading and stale data;
- empty workspace / no run / no artifacts;
- warning vs blocking issue;
- active run, cancelled run, failed run and succeeded run;
- managed runtime permission mode and pending permission requests;
- read-only Ask runs;
- Git dirty/clean/ready-after-review states;
- API/server unreachable and browser/live E2E diagnostic states.

## Implementation constraints

- Do not change backend API, artifact schemas, runtime contracts or CLI flags unless a separate
  slice explicitly requires it.
- Preserve or intentionally migrate stable operator-facing `data-testid` selectors used by UI and
  live E2E; do not reintroduce hidden compatibility controls.
- Prefer current React/Vite component model and existing hooks; avoid inventing a second data layer.
- Treat right inspector and activity drawer as shared shell primitives, not per-screen one-off panels.
- Live E2E must be updated with the UI changes in the same feature wave.

## Current code baseline after latest main

Rebase on `origin/main` commit `3aa458a` ("Improve live E2E operator flow") changed the
starting point for V2 implementation:

- current code already has `AppShell`, `TopStatusBar`, `StageRail`, `RightInspector`,
  `ActivityDrawer`, `StagePanels` and `ConsolePrimitives`; V2 should refine these surfaces rather
  than recreate the removed legacy panels;
- `ResultsPanels.tsx`, `RunPanels.tsx` and `SetupWorkspacePanel.tsx` have been deleted, so future
  implementation slices must not plan work against those files or restore their compatibility DOM;
- hidden compatibility controls/testids were removed from the UI shell; selector migration must use
  visible operator controls only;
- the live frontend harness currently supports only `UI_E2E_SCENARIO=init-inspect`; Ask coverage
  is collected by default through `UI_E2E_QA_SMOKE=1` and is diagnostic/non-release UX evidence;
- `frontend-e2e-result.json` already includes screenshot refs when screenshots are produced, while
  Playwright trace/screenshot/video retention stays in `ui/playwright.live.config.ts`.

## Data source map

Console V2 is expected to use existing API surfaces and current React hooks/facades. If a screen
needs richer data than the current API exposes, the first implementation should show an explicit
empty/partial state rather than changing backend contracts implicitly.

- Source:
  - workspace manifest editor state from existing workspace setup hooks;
  - `POST /api/workspace/validate` for resolved repos and diagnostics;
  - `GET /api/workspace/manifest` / manifest save flow for `workspace.yaml`.
- Readiness:
  - `GET /api/system/doctor`;
  - `POST /api/workspace/validate`;
  - `GET/PUT /api/runtime/timeouts`, `GET/PUT /api/runtime/execution`,
    `GET/PUT /api/runtime/permissions`, `GET /api/runtime/profile`.
- Charter:
  - baseline bundle/editor artifacts from existing workspace setup/baseline hooks;
  - `charter/*`, `skills/*`, `skills/subagents.yaml`, prompt packs;
  - step0 wizard contract under `charter/wizard/step0-contract.json`.
- Analysis:
  - `POST /api/pipeline/init`, `POST /api/pipeline/refresh`;
  - `GET /api/pipeline/runs`, `GET /api/pipeline/runs/<run_id>`;
  - `GET /api/pipeline/runs/<run_id>/logs`, including `fields` such as `shard_id`, `provider`, `recovery_mode`, `stall_phase`, `validation_error`, `partial_failure_count` and raw-output metadata when emitted;
  - `GET /api/pipeline/runs/<run_id>/artifacts`;
  - pending permission request data already exposed on selected run status.
- Review - Evidence:
  - selected run artifacts from `GET /api/pipeline/runs/<run_id>/artifacts`;
  - artifact content from existing open-artifact flow;
  - coverage/open questions derived from promoted reports and run artifacts;
  - Mermaid preview remains frontend rendering over `.mmd` artifact content, with large C4 diagrams
    presented in a scrollable canvas rather than shrunk to an unreadable thumbnail;
  - failed selected runs may expose only partial artifacts; the UI must keep those blockers visible
    while offering direct navigation to the latest successful run artifacts when available.
- Review - Domain Map:
  - derived model artifacts under `model/entities/*.yaml` and `model/edges/*.yaml`;
  - domain agent outputs under `reports/agent-outputs/domains/*.md`;
  - if model artifacts are missing or sparse, render an explicit partial/empty map state.
- Proposals:
  - proposal/changelog artifacts under `proposals/*` and `reports/changelog/*`;
  - linked findings/evidence from artifact content and current run artifact index.
- Ask:
  - async QA APIs `POST /api/qa/runs`, `GET /api/qa/runs/<run_id>`,
    `GET /api/qa/runs?limit=...`;
  - legacy `POST /api/qa/ask` remains compatibility-only, not the target UI path.
- Publish:
  - existing Git helper flows for commit and proposal branch;
  - workspace artifact list/diff summary should be derived from current Git helper surfaces where
    possible, otherwise rendered as partial state until a separate backend slice is approved.

## Live E2E contract

Console V2 implementation must update the current Playwright surface instead of leaving live E2E
on legacy panel assumptions.

Release-facing live scenario:
- `init-inspect`: headless/fake run journey from Source/Readiness through Analysis, Review and
  Publish. The public `scripts/frontend-live-e2e.sh` shell should remain `init-inspect`-only unless
  a separate owner-approved live-gate slice expands the release surface.

Non-release and deterministic UI coverage:
- `UI_E2E_QA_SMOKE=1`: default Ask smoke layered on `init-inspect`; it checks async `qa.ask`,
  run history, read-only safety, answer/citations panels, context-pack/runtime-execution links and
  screenshots. It is UX evidence rather than a standalone release verdict source; explicit
  `UI_E2E_QA_SMOKE=0` is diagnostic-only and must be recorded as residual risk in
  `swe_ux_assessment_<matrix-id>.md`.
- cancellation/page-close behavior: cover through deterministic fake-runtime UI/API tests and
  frontend reason taxonomy, not through provider-live release scenarios.
- Review Domain Map: cover with unit/component/fake-fixture diagnostics first; do not add a live
  shell allowlist entry until stable model fixtures and release-gate semantics are approved.

V2 selectors should be stable and explicit:
- shell: `console-shell`, `top-status-bar`, `stage-rail`, `right-inspector`, `activity-drawer`;
- stages: `stage-source`, `stage-readiness`, `stage-charter`, `stage-analysis`, `stage-review`,
  `stage-proposals`, `stage-ask`, `stage-publish`;
- shared surfaces: `next-action-panel`, `blockers-panel`, `evidence-refs-panel`,
  `runtime-safety-panel`, `git-publication-panel`;
- Analysis: `analysis-run-timeline`, `analysis-shard-table`, `analysis-run-progress`,
  `analysis-live-diagnostics`,
  `runtime-permission-recovery`,
  `analysis-review-blocker-btn`;
- Review: `review-view-evidence-tab`, `review-view-domain-map-tab`, `review-artifact-explorer`,
  `review-evidence-preview`, `review-domain-map`, `review-citation-coverage`;
- Ask: `qa-run-history`, `qa-answer-panel`, `qa-citations-panel`, `qa-readonly-safety-panel`;
- Publish: `publish-diff-summary`, `publish-gate-panel`, `publish-commit-plan`,
  `publish-commit-selected-btn`.

The live E2E wrapper should continue using the existing frontend reason taxonomy:
`active_run_timeout`, `runtime_run_failed`, `browser_closed`, `api_unreachable`, `server_exited`,
and fallback `playwright_failed`. V2 visual assertion failures should remain `playwright_failed`
unless they reveal one of the more specific infrastructure/runtime states.

Failure diagnostics must include:
- `frontend-e2e-result.json` with scenario, reason, run id, last run status/error/current step and
  diagnostic refs, including screenshot refs when screenshots were produced;
- Playwright screenshots/traces/videos for the failing stage where available; current
  `ui/playwright.live.config.ts` already uses `trace=retain-on-failure`,
  `screenshot=only-on-failure` and `video=retain-on-failure`;
- server log and Playwright log paths;
- black-box step evidence references when invoked through batch/matrix harness.

## Acceptance checklist

- All 8 stages render in the rail and preserve keyboard/screen-reader navigation.
- First viewport answers workspace readiness, run status, blocker and next action.
- Review shows evidence/citations/coverage without forcing raw-path inspection first.
- Analysis keeps logs and failed shard context adjacent.
- Runtime safety is visible on every stage where runtime, QA or publication decisions are made.
- Publish has a clear path from review gate to commit/proposal branch.
- Required CI uses deterministic fake/runtime fixtures; live provider checks remain optional/manual
  unless the release runbook says otherwise.
