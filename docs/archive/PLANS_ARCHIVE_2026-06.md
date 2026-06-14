# PLANS_ARCHIVE_2026-06.md

Closed ExecPlans archived from `docs/PLANS.md` in June 2026.

### Plan ID
EP-20260601-live-e2e-complex-diagnostics

### Context
Canonical live E2E release slices уже покрывают стабильные repo sets, но diagnostic/regression coverage не должна зацикливаться на одних и тех же продуктах. Нужен отдельный non-release selector для более сложных публичных продуктов и явная operator policy: в ручных live E2E прогонах желательно ротировать продукты и feature focus, чтобы успешность не измерялась только привычным happy path.

### Goals (must have)
- [x] Добавить diagnostic-only complex repo presets с pinned GitHub SHA.
- [x] Добавить runnable non-release matrix files и catalog selector для прямого `scripts/full-run-batch-matrix.sh`.
- [x] Зафиксировать rotation guidance: каждый diagnostic прогон по возможности выбирает разные продукты и feature areas.
- [x] Обновить planner/tests/docs без изменения canonical release verdict contract.

### Non-goals
- [x] Не менять canonical `release fast|long|full` matrices, curated path presets или release verdict policy.
- [x] Не добавлять новые headless providers или hosted/security/compliance enforcement.
- [x] Не делать complex diagnostic selector required CI или release readiness signal.

### Approach
1) Добавить GitHub `repos.yaml` presets для Temporal, Backstage, Airflow, Appwrite и Saleor.
2) Добавить отдельные `examples/e2e-matrix.diagnostic.*.yaml`, чтобы не нарушать unique profile id contract внутри matrix.
3) Расширить `examples/e2e-profile-catalog.yaml` и `scripts/live-e2e-plan.py` selector size `complex`.
4) Обновить runbook/testing docs и focused tests.

### Files changed
- `.agents/skills/e2e-live-gate/SKILL.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `examples/e2e-profile-catalog.yaml`
- `examples/e2e-matrix.diagnostic.*.yaml`
- `examples/repos/github/*.repos.yaml`
- `scripts/live-e2e-plan.py`
- `scripts/tests/live_e2e_plan_test.py`
- `scripts/tests/matrix_release_contract_test.py`

### Acceptance criteria
- [x] `python3 -m unittest scripts.tests.live_e2e_plan_test scripts.tests.matrix_release_contract_test`
- [x] `python3 scripts/live-e2e-plan.py --mode regres --size complex --providers qwen --format shell`
- [x] Документация явно говорит, что complex selector diagnostic-only и требует ротации products/features.
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`

### Risks
- Первые shard counts для новых публичных repo sets неизвестны до trusted-machine diagnostic run; catalog маркирует их как `unmeasured`.

### Progress log
- 2026-06-01: Started as diagnostic-only catalog/planner/docs update.
- 2026-06-01: Added `regres complex`, pinned complex product presets, docs rotation policy and tests.
- 2026-06-01: Review pass updated drifted `temporalio/temporal` and `apache/airflow` pins to current remote HEAD and moved the completed ExecPlan to this archive.
---

### Archive note
Archived 2026-06-04 after owner accepted the completed v0.1.4/v0.1.5 CI-only beta release cycle; no fresh trusted `release-fast` verdict was produced, so canonical `RELEASE READY` is not claimed.

### Plan ID
EP-20260604-v014-ci-product-release

### Context
After `v0.1.3`, the next patch release should remove GitHub Actions deprecation warnings and polish the clean UI startup path used by first-time and returning local operators. The product scope is deliberately small: recent workspace shortcuts, safer reopen/resume behavior and run bootstrap selection. `v0.1.4` shipped that product scope, and the only remaining cleanup is `v0.1.5`: a metadata/workflow patch so the next public release run uses Node 24-compatible Anchore/Goreleaser release actions. Release mode remains owner-approved CI-only beta; trusted `release-fast` is not part of this plan unless the owner starts a separate gate.

### Goals (must have)
- [x] Modernize pinned GitHub Actions to Node 24-compatible action versions and CodeQL v4 without changing permissions, gates, SBOM/provenance or pinned-SHA policy.
- [x] Add local-only Recent workspaces to onboarding with newest-first ordering, limit 10, missing-path state, `Open` and `Forget`.
- [x] Reopen existing workspaces by hydrating `workspace.yaml` sources, allowing valid manifests to proceed to `Ready` after runner selection and routing invalid manifests to `Sources` diagnostics.
- [x] Resume newest active run on console bootstrap; otherwise select newest completed run and open `Review` when artifacts exist.
- [x] Prepare `v0.1.4` release metadata and tag final green `main` after metadata PR/main CI.
- [x] Merge release workflow follow-up PR #100 so future tag workflows use Node 24-compatible Anchore/Goreleaser actions.
- [x] Prepare `v0.1.5` release metadata and tag final green `main` after metadata PR/main CI.
- [ ] After owner review, archive this completed release/product plan as bookkeeping-only work.

### Non-goals
- [x] No workspace schema, runtime artifact schema, provider contract, CLI `acp run`, direct `acp serve --workspace` or provider-live release-gate changes.
- [x] No hosted workspace picker, browser directory picker or source repository mutation.
- [x] No canonical `RELEASE READY` claim without a fresh verifier-backed `release_verdict_*.json`.

### Acceptance criteria
- [x] CI hygiene PR and main CI are green with no Node 20 / CodeQL v3 deprecation annotations.
- [x] Release workflow follow-up PR #100 is merged and `main` CI is green after updating Anchore/Goreleaser release actions to Node 24-compatible pinned versions.
- [x] `v0.1.5` release workflow succeeds without Node 20 / CodeQL v3 deprecation annotations.
- [x] `ACP_VERSION=v0.1.5` and `ACP_VERSION=latest` install smokes resolve version `0.1.5`; `acp serve --runtime fake --dry-run` reports launcher readiness.
- [x] Onboarding status exposes local recents; create/open records recents; forget removes a recent entry; missing paths are visible and not openable.
- [x] UI tests cover recents, reopen existing workspace, missing recent forget, runner-required state and active/completed run bootstrap.
- [x] Deterministic fake UI walkthrough from clean launcher to Analysis/Review/Publish remains passable.
- [x] `git diff --check`, `make contracts`, `make test`, `make lint`, `make build` pass before commit.

### Progress log
- 2026-06-04: Merged CI hygiene PR #97 into `main` (`bc45d1c`); PR and main CI passed after pinned GitHub Actions moved to Node 24-compatible versions and CodeQL v4.
- 2026-06-04: Started product polish branch `codex/onboarding-resume-polish`: added local Recent workspaces API/UI, reopen/resume copy and newest active/newest completed run bootstrap tests. During rendered fake walkthrough found and fixed a premature `/api/workspace/bundle` call that produced `428 workspace_not_selected` in browser console after selecting a draft workspace. Validation passed: `git diff --check`, `go test ./internal/docsync`, focused `go test ./internal/api`, UI unit suite `69/69`, Full DoD (`make contracts`, `make test`, `make lint`, `make build`) and Playwright rendered smoke from clean launcher to Analysis/Review/Publish with no browser console issues or HTTP >=400 responses. Product PR #98 merged into `main` at `5337e61`, and `main` CI passed; `v0.1.4` release metadata/tag still pending.
- 2026-06-04: Started `v0.1.4` release metadata branch after green `main`; scope is changelog/latest-release docs only, with CI-only beta/no trusted `release-fast` wording.
- 2026-06-04: Published `v0.1.4` as a GitHub prerelease from tag commit `9de4abaf6549510e45b2616ba8742b01c1912b03`; release workflow `26942766883` succeeded, release URL is `https://github.com/GrinRus/ProvenArch/releases/tag/v0.1.4`, and install smoke passed for explicit `ACP_VERSION=v0.1.4` plus `ACP_VERSION=latest`. Fresh trusted-machine `release-fast` was not run, so canonical `RELEASE READY` is not claimed.
- 2026-06-04: The `v0.1.4` release run still emitted Node 20 deprecation annotations for `anchore/sbom-action/download-syft@v0.20.6` and `goreleaser/goreleaser-action@v6`. Follow-up PR #100 updated them to Node 24-compatible pinned commits for `anchore/sbom-action/download-syft@v0.24.0` and `goreleaser/goreleaser-action@v7.2.2`; PR #100 merged into `main` at `afa3ff8`, and `main` CI passed. `v0.1.5` is the planned tiny public patch to publish a release run with that cleanup.
- 2026-06-04: Published `v0.1.5` as a GitHub prerelease from tag commit `fb3071e66212942ba2dcc899e3e6c4292ddf9fa4`; release workflow `26946062981` succeeded without Node 20 / CodeQL v3 deprecation strings in release logs. Release URL is `https://github.com/GrinRus/ProvenArch/releases/tag/v0.1.5`; assets include four platform archives, per-archive SBOM JSON files and `checksums.txt`; provenance attestation verified for `acp_darwin_arm64.tar.gz`. Install smoke passed for explicit `ACP_VERSION=v0.1.5` and `ACP_VERSION=latest`, both resolving `acp version 0.1.5` at commit `fb3071e`, and `acp serve --runtime fake --dry-run` reported launcher readiness. Fresh trusted-machine `release-fast` was not run, so canonical `RELEASE READY` is not claimed.

---

### Archive note
Archived 2026-06-04 after owner accepted the completed onboarding-first startup implementation; remaining work is future backlog selection, not this active plan.

### Plan ID
EP-20260602-onboarding-first-startup

### Context
Current `v0.1.1` onboarding is too CLI-first: the operator must choose `--workspace` before the UI exists and must often seed at least one repo through `--auto-init` before the `Source` screen can take over. That contradicts the desired first-run UX for a local-first operator console: start ACP, choose or create an architecture workspace in the UI, add one or more target repositories from Git URL or local checkout path, choose the runner, validate readiness, then run analysis.

The current architecture still has important constraints that should be preserved:
- one active architecture workspace per served console session;
- `repos[]` is already the multi-repo source of truth in `workspace.yaml`;
- source repositories remain read-only inputs;
- deterministic `fake` stays the required-CI default and recommended first walkthrough runner;
- live providers remain explicit opt-in (`claude-code`, `qwen-code`, `codex-code`);
- existing `acp serve --workspace ...` direct mode must remain for CI, scripts, existing live E2E and users who already have a workspace.

### UX spec
Users:
- local operator / architect / tech lead starting ACP for the first time;
- existing ACP user reopening a known architecture workspace;
- multi-repo system owner who needs to analyze several local checkouts and/or Git URLs together.

Flow:
1) Operator runs `acp serve` with no workspace, or opens a direct-mode `acp serve --workspace ...` session.
2) If no workspace is selected, the UI opens an onboarding screen instead of the normal stage shell.
3) Step 1 `Workspace`: create a new architecture workspace path or reopen a recent/existing one; validate writable path, git availability and fixed layout readiness.
4) Step 2 `Sources`: add one or more target repos using `GitHub/GitLab URL` or `Local folder`, optional `ref`, and docs imports path; save `workspace.yaml` only after repo validation can produce a valid manifest.
5) Step 3 `Runner`: choose runner before analysis. `fake` is the recommended default and requires no provider command; `claude-code`, `qwen-code`, `codex-code` show command/auth availability via readiness checks and require explicit selection.
6) Step 4 `Ready`: show workspace/repo/runner/permission summary, then enter the existing `Source -> Readiness -> Charter -> Analysis -> Review -> Proposals -> Ask -> Publish` console.

Screens:
- `Welcome / Workspace`: local-only explanation, create/reopen workspace, recent workspaces, path validation, no marketing hero.
- `Repository Sources`: dense multi-repo table/cards with add/remove, source type, path/URL, ref, per-row diagnostics and docs imports path.
- `Runner`: segmented runner choice, provider availability, command override hints, permission mode summary, fake-first recommendation.
- `Ready`: compact checklist and primary action `Open console` / `Run first analysis`.
- Existing console shell: unchanged after onboarding completion, with `Source` still editable for later repo changes.

States:
- no workspace selected: only onboarding APIs are active; pipeline/run/publish actions disabled;
- new workspace without manifest: layout can exist, but `workspace.yaml` is draft until at least one repo is configured;
- existing workspace: parse manifest, hydrate onboarding fields, allow skip to console after validation;
- invalid path/not writable/not git-initable: inline error with remedy near workspace path;
- repo path missing, duplicate repo name, invalid URL/ref, private git auth failure: row-level diagnostics and no run action;
- runner unavailable: selectable only as draft with explicit blocker, or disabled until command/auth is fixed; `fake` remains always available;
- direct mode with `--workspace`: bypass onboarding unless manifest is missing and explicit onboarding mode is enabled.

Information architecture:
- Onboarding is a pre-console setup surface, not a ninth stage.
- `arch-workspace` path is process/session-level app context.
- target repos and docs imports are workspace manifest data.
- runner choice is setup-time execution context plus persisted runtime profile where applicable; live provider selection must be visible before the first run.

Risks:
- letting the UI choose arbitrary local paths is acceptable only because ACP is local-first and bound to loopback, but errors must be explicit and no source repo mutation should happen;
- browser directory pickers are not portable enough for the MVP path, so typed paths/recent workspaces are the safe baseline;
- adding no-workspace `serve` changes public CLI behavior and live E2E startup assumptions; direct `--workspace` compatibility must be tested;
- runtime mode is partly process-scoped today, so onboarding runner selection requires a deliberate service lifecycle design instead of only editing `workspace.yaml`.

### Goals (must have)
- [x] Add an onboarding-first startup path where `acp serve` can start a local UI without a preselected workspace.
- [x] Let the UI create or reopen an architecture workspace before entering the normal console.
- [x] Let the UI configure one or more target repositories in `repos[]` using Git URL or local checkout path, with optional `ref`.
- [x] Require an explicit runner choice before first analysis; show `fake` as recommended and live providers as opt-in with availability diagnostics.
- [x] Keep existing `acp serve --workspace ...`, `--auto-init`, `--repos-file`, `acp init-workspace`, `acp run` and CI/batch flows working.
- [x] Keep backend API/schema/runtime artifact/workspace manifest contracts unchanged unless a slice explicitly updates docs, validators, fixtures and tests in the same change.
- [x] Update README/INSTALL/TROUBLESHOOTING only after implementation matches the new behavior.
- [x] Add deterministic CLI/API/UI coverage for onboarding and retain existing direct-mode `init-inspect` live E2E behavior without adding a new provider-live release scenario.
- [ ] After owner review/merge, move this completed implementation plan to archive.

### Non-goals
- [x] No hosted workspace picker, cloud storage, remote credential store or SCM OAuth flow.
- [x] No file browser integration that enumerates arbitrary local directories server-side.
- [x] No change to source repo write policy; target repositories stay read-only inputs.
- [x] No expansion of required live provider release gates.
- [x] No removal of direct CLI/bootstrap mode.

### Approach
1) Implement a launcher/no-workspace server mode behind `acp serve` while preserving direct `--workspace` service construction.
2) Add onboarding APIs for workspace path validation/create/open, recent workspace metadata, repo manifest draft save/validate and runner readiness.
3) Refactor service construction so the backend can attach a selected workspace/runtime after onboarding without restarting the process, or use a narrowly scoped launcher server that swaps into the existing workspace server.
4) Build the onboarding UI as a pre-console surface; keep the current stage shell unchanged after setup completion.
5) Persist valid repo sources through existing `workspace.yaml` rendering/validation; do not introduce a second source-of-truth file for repos.
6) Treat runner selection as required UI state before `Run first analysis`; use `fake` for deterministic onboarding e2e and show provider diagnostics for headless choices.
7) Update live/deterministic e2e carefully: add onboarding coverage without breaking release-facing direct-mode `UI_E2E_SCENARIO=init-inspect`.
8) Sync docs after behavior lands, making README/INSTALL distinguish UI-first setup, direct workspace mode and multi-repo setup.

### Files expected to change
- `cmd/acp/main.go`, `cmd/acp/main_test.go`
- `internal/api/*`
- `internal/workspace/*` only if workspace open/create helpers need narrow extraction
- `internal/doctor/*` only if runner/workspace checks need reusable launcher diagnostics
- `ui/src/App.tsx`, `ui/src/components/*`, `ui/src/hooks/*`, `ui/src/lib/*`, `ui/src/App.test.tsx`
- `ui/e2e/live-flow.spec.ts`, `scripts/frontend-live-e2e.sh` only if startup flow/test harness changes
- `docs/PLANS.md`, `docs/BACKLOG.md`, `docs/STAKEHOLDER_DOC.md`, then README/INSTALL/TROUBLESHOOTING after implementation

### Acceptance criteria
- [x] `acp serve` without `--workspace` starts a loopback onboarding UI and does not require a repo CLI flag.
- [x] `acp serve --workspace <path>` still starts the existing single-workspace console path.
- [x] Onboarding can create a new workspace, initialize fixed layout/git, configure multiple repos, choose `fake`, validate readiness and enter the console.
- [x] Onboarding can reopen an existing workspace and either skip to console or edit sources/runner before continuing.
- [x] The first analysis cannot start until workspace, at least one repo and runner are valid.
- [x] Multi-repo `repos[]` names are unique, path/git URL modes are validated, and row-level errors are recoverable in the UI.
- [x] Runner selection distinguishes `fake` availability from live provider command/auth blockers.
- [x] Source repos are not mutated during onboarding validation or fake analysis.
- [x] Existing direct-mode live E2E contract remains compatible; onboarding gets deterministic/fake CLI/API/UI coverage.
- [x] `git diff --check`, focused tests and Full DoD pass.

### Test plan
- Go CLI tests:
  - `serve` without workspace starts launcher/dry-run successfully;
  - `serve --workspace` direct mode still validates and serves;
  - invalid workspace paths and missing git/writable surfaces return actionable errors;
  - auto-init and repos-file compatibility are unchanged.
- API tests:
  - onboarding status, workspace create/open, repo draft validate/save and runner readiness;
  - no pipeline actions are allowed before workspace selection;
  - selected workspace uses the same service/API behavior as direct mode.
- UI unit tests:
  - onboarding workspace path states;
  - multi-repo add/remove/path/git URL/ref validation;
  - runner required state and provider availability copy;
  - transition into existing Console V2 shell.
- E2E:
  - deterministic onboarding fake flow from blank launch to Source/Readiness/Analysis;
  - existing `UI_E2E_SCENARIO=init-inspect` direct-mode regression;
  - no new required provider-live scenario without owner-approved release-gate slice.
- Full DoD:
  - `git diff --check`
  - `make contracts`
  - `make test`
  - `make lint`
  - `make build`

### Slice ExecPlan - 17A-17E Onboarding-first beta baseline

Status: done.

Goals:
- [x] Add `acp serve` launcher mode when `--workspace` is omitted, with `--workspace` direct mode preserved.
- [x] Add onboarding status/workspace create-or-open API, runtime selection API and session service reattachment.
- [x] Support a new workspace draft state where layout/git can exist before `workspace.yaml` is valid.
- [x] Add a UI pre-console onboarding shell for workspace, sources, runner and ready summary.
- [x] Save multi-repo sources through existing `workspace.yaml` semantics and keep Source editable after onboarding.

Non-goals:
- [x] Do not change workspace schema, runtime artifact contracts, CLI `run` behavior or provider contracts.
- [x] Do not remove or weaken `--workspace` direct mode.
- [x] Do not add browser directory picker, hosted mode or new provider-live release scenario.

Implementation notes:
- Prefer reusing existing `workspace.InitLayout`/`EnsureLayout`/git-init helpers rather than adding a parallel workspace model.
- Treat recent workspaces as optional local UI metadata only if it can be done without new cross-machine contracts; otherwise defer recents to 17B.
- Keep launcher APIs loopback-only with the same local server assumption as the rest of ACP.

Validation:
- [x] Focused Go tests for launcher/direct serve modes.
- [x] Focused UI tests for workspace/source/runner onboarding and Console V2 transition.
- [x] `git diff --check`.
- [x] Full DoD after the slice implementation.

### Progress log
- 2026-06-02: Created onboarding-first startup plan after UX review found that current first-run flow is too CLI-first: workspace is chosen before UI, while target repos and runner should be selected in UI. No code implemented in this planning slice.
- 2026-06-02: Implemented onboarding-first beta baseline: `acp serve` launcher mode, optional `--workspace` direct mode, `/api/onboarding/status|workspace|runtime`, workspace draft/session service state, pre-console OnboardingShell, multi-repo source diagnostics, mandatory runner selection and transition into Console V2. Updated README/INSTALL/TROUBLESHOOTING/ARCHITECTURE/STAKEHOLDER docs. Validation passed: focused Go/API tests, focused UI tests, `git diff --check`, `make contracts`, `make test`, `make lint` with Go toolchain in PATH, and `make build`. Provider-live release gate was not expanded.

---

### Archive note
Archived 2026-06-04 after owner accepted the completed long-run review and Git diff UX implementation; no runtime/schema contract changes are pending from this plan.

### Plan ID
EP-20260602-long-run-review-diff

### Context
Console V2 made long runs observable through mission control, stage timeline and activity logs, but the daily operator workflow still has a gap: during a long provider run the user needs to understand which step is producing reviewable evidence, which artifacts are already available, and what changed in the workspace Git tree without leaving the UI. The approved follow-up scope is local UI/API only: expose run review summaries and real workspace Git diffs, then wire them into Analysis, Review, Proposals and Publish.

Constraints:
- runtime artifact schemas, `workspace.yaml`, CLI `acp run`, provider contracts and public live E2E shell stay unchanged;
- Git diff source of truth is the architecture workspace Git repository, never target repos;
- long-run states must be reviewable through artifacts, logs, evidence and diff, not only through raw logs;
- existing operator-facing selectors remain stable where possible.

### Goals (must have)
- [x] Add `GET /api/pipeline/runs/<run_id>/review-summary` with canonical `step0..step4` summaries built from existing run status, logs, artifacts and taskrun refs.
- [x] Add `GET /api/git/diff` for workspace-only Git status, folder summaries, selected file status and text hunks with strict path normalization.
- [x] Add a persistent active-run strip across Console V2 stages with run status, current step, provider, progress, warnings/errors and cancel.
- [x] Rework `Analysis` toward step-level review: step cards and selected-step tabs for artifacts, logs, evidence and diff.
- [x] Add a Review Queue and use real diff data in `Review`, `Proposals` and `Publish`.
- [x] Turn `Publish` into a real Git Review Room: folder summary, changed file list, selected hunks and existing publish gate/actions.
- [x] Cover queued/no logs/partial artifacts/failed/canceled/stale/no changes/binary diff states with explicit UI/API states where data is available.
- [x] Keep direct-mode live E2E contract unchanged.
- [ ] After owner review/merge, move this completed follow-up plan to archive.

### Non-goals
- [x] Do not change workspace manifest schema, runtime artifact schemas, provider contracts, CLI flags or `acp run` behavior.
- [x] Do not add approval persistence, hosted mode, source repo mutation, new provider-live release gate or new release-facing UI E2E scenario.
- [x] Do not introduce a second artifact source of truth beyond existing workspace files, run logs, run artifacts and Git status.

### Approach
1) Add narrow backend local APIs for run review summaries and workspace Git diff, with tests for path safety, empty diffs, binary files and step mapping.
2) Add frontend contracts and hooks for review summary and Git diff loading, reusing current run selection/polling.
3) Add `ActiveRunStrip` into the existing shell without changing the stage rail or onboarding boundary.
4) Update Analysis, Review, Proposals and Publish to expose artifacts/logs/evidence/diff in consistent tabs.
5) Add UI tests for the new surfaces and preserve existing selectors/live-shell expectations.
6) Run focused tests and Full DoD before committing the follow-up slice.

### Files expected to change
- `internal/api/server.go`
- `internal/api/review_diff.go`
- `internal/api/server_test.go`
- `ui/src/lib/*`
- `ui/src/hooks/*`
- `ui/src/components/*`
- `ui/src/App.tsx`, `ui/src/App.test.tsx`, `ui/src/styles.css`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Review summary returns five canonical steps for init runs and handles no logs, failed/canceled/stale and partial artifact states without panics.
- [x] Git diff returns valid empty diffs, rejects unsafe paths, reports untracked/modified/deleted/binary files and filters by folder/file/run/step where applicable.
- [x] Active run strip remains visible after console entry across stages.
- [x] Analysis exposes current/failed step details with artifacts, logs, evidence and diff.
- [x] Review Queue is visible and can navigate to reviewable artifacts.
- [x] Proposals and Publish no longer show the old partial line-level diff placeholder when real Git diff data is available.
- [x] Publish shows real workspace folder/file/hunk diff data while preserving hard blockers, warnings, open questions and explicit Git actions.
- [x] `git diff --check`, focused tests and Full DoD pass.

### Test plan
- Go/API:
  - review summary step mapping from run status/logs/artifacts;
  - review summary no-log failed/partial cases;
  - Git diff untracked/modified/deleted/folder filter/file filter/empty/binary/invalid traversal.
- UI:
  - active run strip persistence;
  - step cards and selected-step tabs;
  - Review Queue counts/navigation;
  - real diff state in Review/Proposals/Publish.
- Deterministic checks:
  - focused Go/API tests;
  - focused UI test suite;
  - `git diff --check`;
  - Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Progress log
- 2026-06-02: Started long-run review/diff follow-up after UX review found long provider runs were observable through logs but not sufficiently reviewable through step-level artifacts and workspace Git diff. Scope is local UI/API only; no schema/runtime/provider/live-shell changes.
- 2026-06-02: Implemented long-run review/diff follow-up: local `review-summary` and workspace Git diff APIs, persistent active-run strip, Analysis step review tabs, Review Queue, real diff tabs in Review/Proposals/Publish and Git Review Room folder/file/hunk view. Found and fixed a real infinite-update bug when opening Diff tabs by stabilizing the Git diff callback. Validation passed: `git diff --check`, focused `./scripts/run-go.sh test ./internal/api`, UI suite `61/61`, `make contracts`, `make test`, `make lint`, `make build`, plus rendered in-app browser smoke on a temporary fake workspace through Analysis/Review/Proposals/Publish with no browser console errors. Runtime schemas, `workspace.yaml`, CLI `acp run`, provider contracts and live shell were unchanged.

---

### Archive note
Archived 2026-06-04 after owner accepted the completed Console V2 implementation and follow-up polish; trusted live validation remains a separate owner-triggered gate.

### Plan ID
EP-20260527-ui-console-v2

### Context
После утверждения 9-screen UI vision нужно зафиксировать Console V2 как реализуемый backlog поверх текущего stage-based UI baseline. Цель - собрать лучший вариант из Mission Control, Evidence Review Workbench, Domain Map, Run Timeline и Git Review Room без изменения backend/API contracts. В этот же feature wave должна войти миграция live E2E логики, иначе новый shell может пройти code review, но сломать trusted-machine release diagnostics.

### Goals (must have)
- [x] Зафиксировать целевой design baseline в `docs/UI_CONSOLE_V2_DESIGN.md`.
- [x] Сохранить approved PNG references в `docs/assets/ui-console-v2/`.
- [x] Добавить Epic 16 в `docs/BACKLOG.md` с кодовыми, UX и live E2E slices.
- [x] Зафиксировать V2 live E2E contract: scenarios, selectors, diagnostics and reason taxonomy.
- [x] Зафиксировать screen data source map so implementation stays on existing APIs unless a later slice changes scope.
- [x] Реализовать unified shell V2: top health strip, stage rail, center workbench, right inspector, bottom activity drawer.
- [x] Реализовать stage surfaces: Source, Readiness, Charter, Analysis, Review Evidence, Review Domain Map, Proposals, Ask, Publish.
- [x] Сохранить или явно мигрировать стабильные operator-facing `data-testid` для UI/unit/live E2E; hidden compatibility controls не возвращать.
- [x] Обновить Playwright live E2E operator journey под V2 shell/stages, включая evidence/logs/runtime safety/Git publication assertions.
- [x] Сохранить текущий public live shell как `UI_E2E_SCENARIO=init-inspect`; Ask покрывать optional `UI_E2E_QA_SMOKE=1`, а cancel/page-close/domain-map держать в deterministic или optional diagnostic coverage до отдельного live-gate решения.
- [x] Сохранить frontend live E2E diagnostics: screenshots/traces refs, stage context and existing reason taxonomy.
- [x] Обновить testing docs/runbook после implementation.
- [x] Выполнить full DoD для implementation slice: `make contracts`, `make test`, `make lint`, `make build`.
- [ ] После owner review/merge перенести план в архив.

### Non-goals
- [x] Не менять backend API, schemas, runtime artifact contracts, workspace contracts или CLI flags в UI-only slice.
- [x] Не добавлять hosted/security/compliance enforcement.
- [x] Не делать provider-live checks required CI; live provider matrix остаётся manual trusted-machine gate.
- [x] Не коммитить сгенерированные PNG mockups как product assets без отдельного owner decision.

### Approach
1) Use `docs/UI_CONSOLE_V2_DESIGN.md` as the product/UX source for screen inventory, shell contract, state model and acceptance checklist.
2) Implement in reviewable PR slices from `docs/BACKLOG.md` Epic 16, starting with shared shell primitives and stage status model.
3) Reuse existing hooks/API clients; keep backend/API schemas unchanged unless a later slice explicitly changes scope.
4) Use the data source map in `docs/UI_CONSOLE_V2_DESIGN.md`; render explicit empty/partial states for missing data instead of silently adding API fields.
5) Move stage surfaces one by one, preserving existing critical operator-facing selectors or migrating tests deliberately; do not reintroduce hidden compatibility DOM.
6) Update `ui/e2e/live-flow.spec.ts` in the same wave:
   - `init-inspect`: validate Source/Readiness, run Analysis, inspect Review Evidence, check Publish gate, and assert top strip/right inspector/activity drawer on each transition.
   - optional `UI_E2E_QA_SMOKE=1`: submit async QA inside `init-inspect`, poll result, assert confidence/citations/unresolved/read-only safety and context/runtime artifact links.
   - cancellation/page-close: keep deterministic fake-runtime UI/API coverage outside the provider-live release gate.
   - domain map: add unit/component/fake-fixture diagnostics first; add a live diagnostic only after stable model fixtures and owner-approved release-gate semantics exist.
7) Preserve Playwright failure artifacts from `ui/playwright.live.config.ts` (`trace=retain-on-failure`, `screenshot=only-on-failure`, `video=retain-on-failure`) and make sure result JSON diagnostic refs lead to the `UI_E2E_OUTPUT_DIR` results.
8) Update `scripts/frontend-live-e2e.sh` and script tests only if the public live shell or diagnostic refs change; do not add `ask-readonly`/`domain-map-diagnostic`/page-close scenarios or new frontend reason codes unless a separate classification/release-gate slice is approved.
9) Sync README/ARCHITECTURE/testing/runbook docs only when the implemented UI behavior changes.

### Slice ExecPlan - 16A UI V2 shell foundation

Status: done.

Goals:
- [x] Refine existing shell primitives (`AppShell`, `TopStatusBar`, `StageRail`, `RightInspector`, `ActivityDrawer`) toward the approved V2 shell without restoring legacy panels.
- [x] Extract a shared stage status model for the eight-stage rail.
- [x] Add baseline keyboard/focus contract for rail navigation and shared inspector/drawer surfaces.
- [x] Surface runtime permission mode and Git publication path in the shared shell using existing UI state.

Non-goals:
- [x] No backend API, schema, runtime contract, CLI flag or workspace contract changes.
- [x] No stage-specific redesign beyond shared shell foundation.
- [x] No `UI_E2E_SCENARIO` expansion, no new frontend reason taxonomy, no hidden compatibility controls.

Implementation notes:
- Use current React/Vite components and hooks only.
- Add/keep visible operator-facing `data-testid` selectors for V2 shared shell surfaces.
- Render missing Git/publication data as explicit partial/empty state in the inspector.

Validation:
- [x] Focused UI unit tests for stage rail keyboard navigation and shell shared surfaces.
- [x] Browser visual QA for desktop and narrow viewport on embedded build.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Slice ExecPlan - 16B Source + Readiness consolidation

Status: done.

Goals:
- [x] Add a Source repo table with name, source, ref, analysis include/exclude and validation status.
- [x] Preserve Git URL/local path editing, docs imports path and advanced `workspace.yaml` editor.
- [x] Add Readiness cards for workspace, repositories, runtime provider, permissions and artifacts.
- [x] Add a compact runtime profile summary without making advanced runtime settings the primary flow.

Non-goals:
- [x] No backend API, schema, runtime contract, CLI flag or workspace contract changes.
- [x] Do not add guided include/exclude editing until a separate workspace-contract slice approves it.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Use existing `ValidateResponse`, `DoctorResponse`, runtime settings state and artifact counters.
- Show include/exclude as an explicit advanced-only/partial state because `GuidedRepo` does not own that contract today.
- Keep existing Source/Readiness actions and `data-testid` selectors stable.

Validation:
- [x] Focused UI unit tests for Source table and Readiness summary cards.
- [x] Browser visual QA for Source and Readiness on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Slice ExecPlan - 16C Charter workbench

Status: done.

Goals:
- [x] Add a Charter wizard summary for project, scope, NFR priorities and rules before the editable form.
- [x] Add domain/team card overview using existing baseline bundle artifact metadata, with explicit empty/partial states.
- [x] Add baseline prompt bundle status covering prompt packs, live-consumed prompts, reference-only prompts and bundle warnings.
- [x] Keep the existing charter artifact editor and Git helper actions available without changing backend APIs.

Non-goals:
- [x] No backend API, schema, runtime contract, CLI flag or workspace contract changes.
- [x] Do not add a new cards API or infer domain/team ownership beyond existing artifact paths/metadata.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse current wizard state, baseline bundle response and Git helper state.
- Preserve existing baseline editor labels/buttons/tests while adding V2 summary surfaces.
- Render missing card artifacts as explicit empty/partial state rather than inventing data.

Validation:
- [x] Focused UI unit tests for Charter summary, card overview and prompt bundle status.
- [x] Browser visual QA for Charter on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Slice ExecPlan - 16D Analysis mission control

Status: done.

Goals:
- [x] Add Analysis run progress summary with selected run id, runtime/provider, status, current step and warnings/errors.
- [x] Add canonical step timeline for `step0..step4` using existing run status and logs.
- [x] Add shard/log-derived table with step, scope, provider, status, artifact/log reference and failed-row drilldown.
- [x] Keep existing run actions, pending permissions, run history and bottom activity drawer behavior.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not add a new shard API; derive table rows from existing run logs/status/artifacts and show partial state when data is sparse.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse `RunStatusResponse`, `RunLogEntry`, current artifacts and existing runtime setup state.
- Add stable V2 selectors: `analysis-run-progress`, `analysis-run-timeline`, `analysis-shard-table`, `analysis-review-blocker-btn`.
- Keep logs adjacent through the existing activity drawer and add warning/error summary in the Analysis workbench.

Validation:
- [x] Focused UI unit tests for run progress, timeline and shard/log table.
- [x] Browser visual QA for Analysis on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Analysis mission control using existing run status/log/artifact data, added V2 selectors (`analysis-run-progress`, `analysis-run-timeline`, `analysis-shard-table`, `analysis-review-blocker-btn`), verified terminal fake runs do not leave active shard rows, and kept warning-only runs from enabling the blocker action.

### Slice ExecPlan - 16E Review evidence workbench

Status: done.

Goals:
- [x] Add Review Evidence tabs for evidence workbench and domain map placeholder routing, preserving the single `Review` stage.
- [x] Add grouped artifact explorer by workspace folder using existing selected-run artifacts.
- [x] Add primary evidence preview that handles markdown/text and existing Mermaid diagram rendering without raw-path-first layout.
- [x] Add citation/coverage/trust summary surfaces derived from existing coverage summary, open questions, findings/diagram artifacts and selected artifact state.
- [x] Keep current artifact open flow, selected artifact selectors and diagram preview behavior available for unit/live E2E migration.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not implement the interactive domain map; reserve `review-domain-map` as an explicit future/partial state for `16F`.
- [x] Do not add approve/flag persistence or Git publish gating mutations in this slice.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse `nonDiagramArtifacts`, `diagramArtifacts`, `selectedArtifact`, `selectedArtifactContent`, `coverageSummary`, `openQuestions` and `onOpenArtifact`.
- Add stable V2 selectors: `review-view-evidence-tab`, `review-view-domain-map-tab`, `review-artifact-explorer`, `review-evidence-preview`, `review-domain-map`, `review-citation-coverage`.
- Render missing data as explicit empty/partial states; do not infer citations or claim coverage beyond existing artifacts/text.

Validation:
- [x] Focused UI unit tests for artifact grouping, evidence preview, citation/coverage/trust panels and domain-map partial state.
- [x] Browser visual QA for Review on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Review Evidence Workbench with visible V2 selectors (`review-view-evidence-tab`, `review-view-domain-map-tab`, `review-artifact-explorer`, `review-evidence-preview`, `review-domain-map`, `review-citation-coverage`), grouped artifacts by workspace folder, preserved artifact/diagram open flow, and kept Domain Map as an explicit 16F partial state.

### Slice ExecPlan - 16F Review domain map

Status: done.

Goals:
- [x] Replace the Review Domain Map placeholder with a useful map workbench derived from existing selected-run artifacts.
- [x] Show domain/service/model nodes from `model/entities/*` and domain agent outputs from `reports/agent-outputs/domains/*`.
- [x] Show model edges from `model/edges/*` with relationship direction, relation type and evidence navigation.
- [x] Add an ownership/coverage/cross-repo inspector that makes sparse or missing model data explicit.
- [x] Preserve Review Evidence tab behavior and artifact open flow for model/entity/domain artifacts.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not parse or validate full model YAML in the browser; derive the map from artifact paths/labels and show partial states where content would be required.
- [x] Do not add graph editing, approval persistence, proposal branch mutations or Publish gate behavior in this slice.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse `nonDiagramArtifacts`, `diagramArtifacts`, `coverageSummary`, `openQuestions` and `onOpenArtifact`.
- Add stable V2 selectors: `review-domain-map-canvas`, `review-domain-map-inspector`, `review-domain-map-node`, `review-domain-map-edge-list`, `review-domain-map-empty`.
- Keep navigation artifact-backed via existing `ArtifactPathButton` and render missing ownership/cross-repo data as explicit partial state.

Validation:
- [x] Focused UI unit tests for populated model map, edge list, artifact navigation and sparse-model empty state.
- [x] Browser visual QA for Review Domain Map on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Review Domain Map with visible V2 selectors (`review-domain-map-canvas`, `review-domain-map-inspector`, `review-domain-map-node`, `review-domain-map-edge-list`, `review-domain-map-empty`), model/entity/edge/domain artifact navigation, explicit sparse-model partial states, and no backend/API/schema/live-shell changes.

### Slice ExecPlan - 16G Proposals review room

Status: done.

Goals:
- [x] Replace the basic Proposals artifact list with a review-room layout for proposal/changelog artifacts.
- [x] Add proposal package list and preview tabs for proposal body, linked evidence, changelog and explicit diff partial state.
- [x] Add proposal quality, unresolved blocker and publication path panels using existing artifacts, `openQuestions`, `proposalBranch` and `gitStatus`.
- [x] Preserve existing artifact open flow and proposal/changelog routing from Review/inspector.
- [x] Surface proposal branch path before Publish without adding Git mutations to this slice.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not implement proposal approval persistence, Git diff API, commit actions or branch creation in Proposals; keep those in Publish.
- [x] Do not parse full proposal markdown for semantic quality; derive package/type/status from artifact paths and labels.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse `nonDiagramArtifacts`, `selectedArtifact`, `selectedArtifactContent`, `openQuestions`, `proposalBranch`, `gitStatus` and `onOpenArtifact`.
- Add stable V2 selectors: `proposals-review-room`, `proposals-artifact-list`, `proposal-preview-tabs`, `proposal-preview-panel`, `proposal-quality-panel`, `proposal-publication-path`.
- Render missing ADR/RFC/changelog/diff data as explicit partial states.

Validation:
- [x] Focused UI unit tests for proposal grouping, preview tabs, evidence/changelog partial states and publication path panel.
- [x] Browser visual QA for Proposals on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Proposals Review Room with V2 selectors (`proposals-review-room`, `proposals-artifact-list`, `proposal-preview-tabs`, `proposal-preview-panel`, `proposal-quality-panel`, `proposal-publication-path`), artifact-derived package grouping, preview/evidence/changelog/diff tabs, publication path handoff to Publish, and no backend/API/schema/live-shell changes.

### Slice ExecPlan - 16H Ask read-only Q&A console

Status: done.

Goals:
- [x] Add an Ask workbench with async Q&A run history and selected-run answer view.
- [x] Show answer confidence, citations, unresolved assumptions and explicit related-entity/edge partial state using existing QA response fields only.
- [x] Add a read-only runtime safety panel that explains taskrun-only QA writes and links context-pack/runtime-execution audit artifacts.
- [x] Preserve current async Ask flow, visible operator selectors and optional `UI_E2E_QA_SMOKE=1` assertions.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not add mutation actions, canonical artifact writes, approval persistence or new QA result fields in this slice.
- [x] Do not expand release-facing live E2E scenarios or reason taxonomy.
- [x] Do not restore hidden compatibility controls.

Implementation notes:
- Reuse `POST /api/qa/runs`, `GET /api/qa/runs/<run_id>` and `GET /api/qa/runs?limit=20` via the existing QA API client.
- Add stable V2 selectors: `qa-run-history`, `qa-answer-panel`, `qa-citations-panel`, `qa-readonly-safety-panel`.
- Keep existing visible selectors (`qa-panel`, `qa-question-input`, `qa-ask-btn`, `qa-run-status`, `qa-answer`) so current unit/live diagnostics remain valid.
- Render missing citations/unresolved/related entities as explicit empty/partial state rather than inventing API data.

Validation:
- [x] Focused UI unit tests for QA run history, selected answer, citations/unresolved/confidence and read-only safety/audit links.
- [x] Browser visual QA for Ask on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Ask read-only Q&A console with V2 selectors (`qa-run-history`, `qa-answer-panel`, `qa-citations-panel`, `qa-readonly-safety-panel`), existing async QA APIs, selected-run audit links (`context-pack.json`, `qa-answer.json`, `runtime-execution.json`), optional QA-smoke assertions and explicit partial state for structured related entities/edges. No backend/API/schema/runtime contract/live-shell expansion changes.

### Slice ExecPlan - 16I Publish gate

Status: done.

Goals:
- [x] Replace the basic Publish Git helper surface with a Git Review Room layout.
- [x] Show folder-level workspace artifact summary, selected artifact preview and explicit diff partial state using existing artifact data only.
- [x] Add publish gate/checklist/blockers, commit plan, proposal branch path and prepared commit-message copy action.
- [x] Preserve existing Git commit/proposal branch mutations and visible operator selectors while adding V2 selectors.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not implement a Git diff API, artifact selection persistence or approval workflow in this slice.
- [x] Do not expand release-facing live E2E scenarios or reason taxonomy.
- [x] Do not restore hidden compatibility controls.

Implementation notes:
- Reuse existing selected-run artifact index, selected artifact content, open questions, Git message/status and proposal branch state.
- Add stable V2 selectors: `publish-diff-summary`, `publish-preview-tabs`, `publish-gate-panel`, `publish-commit-plan`, `publish-commit-selected-btn`.
- Keep current Git helper actions visible and mutation-compatible with `/api/git/commit` and `/api/git/proposal-branch`.
- Render missing real Git diff/dirty folder data as explicit partial state until a separate backend slice exposes it.

Validation:
- [x] Focused UI unit tests for folder summary, selected artifact preview, publish gate/checklist, commit plan and Git actions.
- [x] Browser visual QA for Publish on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Publish Git Review Room with folder artifact summary, selected artifact preview, advisory publish gate, commit plan, prepared message copy action and preserved Git commit/proposal branch mutations/selectors. Added V2 selectors (`publish-diff-summary`, `publish-preview-tabs`, `publish-gate-panel`, `publish-commit-plan`, `publish-commit-selected-btn`), explicit partial state for real Git diff data and desktop/mobile browser QA screenshots. No backend/API/schema/runtime contract/live-shell expansion changes.

### Slice ExecPlan - 16J UI V2 unit and accessibility coverage

Status: done.

Goals:
- [x] Add focused UI tests for V2 shell primitives: stage rail rendering/keyboard/collapse, right inspector priority/disabled action/empty sections and activity drawer controls/export/log states.
- [x] Cover critical empty/blocked/operator-visible states without adding hidden compatibility controls.
- [x] Keep tests fast and component-level where possible so they complement the heavier App integration tests from 16A-16I.

Non-goals:
- [x] No product UI redesign, backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not expand release-facing live E2E scenarios, Playwright reason taxonomy or provider-live release gates.
- [x] Do not add hidden DOM controls solely for test compatibility.

Implementation notes:
- Add a small component test file around `StageRail`, `RightInspector` and `ActivityDrawer`.
- Reuse existing visible labels/selectors and verify accessibility-facing attributes (`aria-current`, `aria-pressed`, disabled primary actions, accessible drawer labels).
- Treat responsive collapse as the operator-facing rail collapse state; CSS breakpoint visual QA remains covered by implementation slices and live/Playwright slices.

Validation:
- [x] Focused UI component tests for shell primitives.
- [x] Existing App focused tests still pass.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Added fast component-level coverage for V2 shell primitives: stage rail accessible labels/keyboard/collapse state, right inspector blocked/attention priority with empty sections and evidence links, and activity drawer empty/populated/export/log-control states. Existing App-focused tests still pass; no product UI, backend/API/schema/runtime contract/live-shell changes.

### Slice ExecPlan - 16K Live E2E selector migration

Status: done.

Goals:
- [x] Migrate `ui/e2e/live-flow.spec.ts` assertions from legacy/raw-path-first surfaces to visible V2 operator selectors where those selectors now exist.
- [x] Keep the release-facing shell on `UI_E2E_SCENARIO=init-inspect` and preserve optional `UI_E2E_QA_SMOKE=1`.
- [x] Add contract-test coverage that the live flow uses V2 visible selectors and does not rely on hidden compatibility controls.

Non-goals:
- [x] Do not add new public live scenarios, provider-live release gates, failure reason classes or wrapper scripts.
- [x] Do not change backend API, schemas, runtime artifacts, CLI flags or workspace contracts.
- [x] Do not add hidden compatibility controls or hidden test-only DOM.

Implementation notes:
- Keep `scripts/frontend-live-e2e.sh` unchanged unless the public shell or result JSON changes.
- Update only Playwright selectors and Python script contract tests needed to protect the selector migration.
- Leave deeper Source -> Readiness -> Analysis -> Review -> Publish operator journey assertions for 16L.

Validation:
- [x] Focused Playwright/spec contract tests for selector migration.
- [x] UI typecheck/full UI tests through DoD.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Migrated `init-inspect` live Playwright assertions to visible V2 selectors (`source-repo-table`, readiness cards/runtime summary, analysis progress/timeline, activity table, review artifact explorer/evidence/citation panels) and added an explicit hidden-compatibility-control absence check. Added Python contract coverage for the selector migration. Public shell remains `UI_E2E_SCENARIO=init-inspect`; optional `UI_E2E_QA_SMOKE=1`, reason taxonomy, result JSON shape, backend/API/schema/runtime contracts and scripts remain unchanged.

### Slice ExecPlan - 16L Live E2E operator journey

Status: done.

Goals:
- [x] Extend the existing `init-inspect` Playwright flow to assert the V2 operator journey: Source -> Readiness -> Analysis -> Review -> Publish.
- [x] Assert operator-critical surfaces: run status, blockers/evidence refs, activity logs, runtime safety and Git publication path.
- [x] Capture durable stage screenshots for Source, Readiness, Analysis, Review and Publish diagnostics while preserving Playwright trace/screenshot/video failure artifacts.

Non-goals:
- [x] Do not add a new public `UI_E2E_SCENARIO`, provider-live release gate, reason taxonomy value or wrapper script.
- [x] Do not move Ask/domain-map/cancel/page-close into release readiness; Ask remains optional `UI_E2E_QA_SMOKE=1`.
- [x] Do not change backend API, schemas, runtime artifacts, CLI flags or workspace contracts.

Implementation notes:
- Keep `scripts/frontend-live-e2e.sh` result JSON shape stable; it already discovers `frontend-*.png` under `UI_E2E_OUTPUT_DIR`.
- Update the fake harness contract fixture to emit the same screenshot names the Playwright spec now captures on success.
- Leave deeper optional Ask/domain-map diagnostics for 16M.

Validation:
- [x] Focused frontend live E2E contract tests.
- [x] UI typecheck/full UI tests through DoD.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Extended `init-inspect` live Playwright flow into the required V2 operator journey Source -> Readiness -> Analysis -> Review -> Publish, with assertions for run status/logs, blockers/evidence refs, runtime safety and Git publication path. Added durable `frontend-source-desktop.png`, `frontend-readiness-desktop.png`, `frontend-analysis-desktop.png`, `frontend-review-desktop.png`, `frontend-publish-desktop.png` and `frontend-review-mobile.png` screenshot refs and updated the fake harness contract fixture accordingly. No new public scenario, reason taxonomy, release gate, wrapper script or backend/API/schema/runtime contract changes.

### Slice ExecPlan - 16M Live E2E Ask and domain-map diagnostics

Status: done.

Goals:
- [x] Keep async Ask checks inside optional `UI_E2E_QA_SMOKE=1` on `init-inspect`, with explicit answer/citation/confidence/read-only audit assertions.
- [x] Add deterministic domain-map diagnostics for render, edge navigation and proposal/evidence links through UI unit/fake-fixture coverage before any live shell expansion.
- [x] Add contract coverage that `ask-readonly` and `domain-map-diagnostic` are not public `scripts/frontend-live-e2e.sh` scenarios.

Non-goals:
- [x] Do not add new public live scenarios, provider-live release gates, reason classes or wrapper scripts.
- [x] Do not make Ask/domain-map required release readiness; Ask remains optional and domain-map remains deterministic/fake-fixture coverage.
- [x] Do not change backend API, schemas, runtime artifacts, CLI flags or workspace contracts.

Implementation notes:
- Update `ui/e2e/live-flow.spec.ts` only within the existing optional `UI_E2E_QA_SMOKE=1` block.
- Add focused `App.test.tsx` domain-map fixture assertions rather than provider-live domain-map execution.
- Update script contract tests, not the shell allowlist itself.

Validation:
- [x] Focused UI unit tests for domain-map diagnostics.
- [x] Focused frontend live E2E contract tests.
- [x] UI typecheck/full UI tests through DoD.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Tightened optional `UI_E2E_QA_SMOKE=1` assertions for citations panel and read-only QA audit path, added deterministic App-level domain-map diagnostics for edge navigation and proposal artifact drilldown, and added shell contract coverage that `ask-readonly`/`domain-map-diagnostic` remain absent from the public frontend live allowlist. Ask remains optional; domain-map remains deterministic/fake-fixture coverage; no new public scenario, reason taxonomy, release gate, wrapper script or backend/API/schema/runtime contract changes.

### Slice ExecPlan - 16N Docs/runbook sync

Status: done.

Goals:
- [x] Sync the active plan acceptance and progress log with the completed Console V2 implementation slices.
- [x] Update testing strategy docs so the required UI/live surfaces describe the V2 Source -> Readiness -> Analysis -> Review -> Publish journey, optional Ask smoke and deterministic domain-map coverage.
- [x] Update the release live E2E runbook so trusted-machine operators understand which V2 screenshots, diagnostics and non-release Ask/domain-map signals are evidence-only.
- [x] Confirm README, ARCHITECTURE and stakeholder docs remain consistent with the implemented UI behavior.

Non-goals:
- [x] No product code, backend API, schema, runtime contract, CLI flag, workspace contract or public live shell changes.
- [x] Do not add release-gate scenarios for Ask, domain map, cancel or page-close behavior.
- [x] Do not alter release verdict semantics, provider matrices, curated repo presets or matrix harness wrappers.

Implementation notes:
- Keep `UI_E2E_SCENARIO=init-inspect` as the only release-facing frontend live scenario.
- Keep `UI_E2E_QA_SMOKE=1` optional/non-release and document screenshots as diagnostics, not release readiness inputs.
- Keep domain-map coverage deterministic/fake-fixture until a separate owner-approved live-gate slice exists.

Validation:
- [x] Focused docs sync test: `go test ./internal/docsync`.
- [x] Focused frontend live E2E contract test.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Synced Console V2 testing strategy and release live E2E runbook with the implemented Source -> Readiness -> Analysis -> Review -> Publish journey, evidence-only screenshot refs, optional `UI_E2E_QA_SMOKE=1`, deterministic domain-map diagnostics and unchanged release-facing `UI_E2E_SCENARIO=init-inspect`. README, ARCHITECTURE and stakeholder docs were checked as already consistent with the implemented UI behavior. No code/API/schema/runtime contract, public live shell, reason taxonomy, release verdict, provider matrix or wrapper-script changes.

### Slice ExecPlan - 16O Post-implementation UI/UX alignment audit

Status: done.

Goals:
- [x] Re-audit the implemented Console V2 against `docs/BACKLOG.md` Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and the saved PNG references.
- [x] Fix operator-flow mismatches found by rendered UI QA without changing backend API, schemas, runtime contracts, CLI flags or workspace contracts.
- [x] Keep Source, Readiness and Analysis primary paths aligned with the target `Source -> Readiness -> Analysis -> Review -> Publish` journey.

Non-goals:
- [x] Do not add approval persistence, Git diff API, new live E2E scenarios, provider-live release gates or new reason taxonomy.
- [x] Do not reintroduce hidden compatibility controls or test-only DOM.
- [x] Do not convert saved design PNGs into runtime product assets.

Implementation notes:
- Source inspector primary action must persist and validate sources, not only update the unsaved `workspace.yaml` draft.
- `Run first analysis` must route to Analysis mission control so operators can watch progress and logs before Review.
- Analysis shard/log table must show duration as available data or an explicit partial state.
- Toolchain-only live E2E precheck tests must not fall back to ambient `node`/`npm` when `ACP_NODE_TOOL_CANDIDATES_ONLY=1`; otherwise validation can recurse into full DoD instead of materializing expected precheck evidence.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server and saved design baseline.
- [x] Focused UI tests for first-run routing, source save semantics and Analysis shard duration.
- [x] Focused resolver and batch-precheck tests for hermetic live E2E node/npm toolchain evidence.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Re-audited Console V2 against Epic 16, the approved design baseline and saved PNG references. Fixed Source primary action so it saves and validates guided source settings in one step, kept raw `workspace.yaml` save inside Advanced, routed `Run first analysis` to Analysis mission control instead of empty Review, added explicit duration/partial-state column to the Analysis shard/log table, tuned the table layout for desktop/mobile visual QA, and fixed the node/npm resolver so `ACP_NODE_TOOL_CANDIDATES_ONLY=1` remains hermetic for live E2E precheck evidence. No backend/API/schema/runtime contract, live E2E public shell, reason taxonomy, Git diff API or approval persistence changes.

### Slice ExecPlan - 16P Post-audit UI/UX interaction hardening

Status: done.

Goals:
- [x] Re-audit the current Console V2 implementation against Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and rendered local UI evidence.
- [x] Fix confirmed operator-flow bugs in Source, Analysis, Ask and Publish without changing backend API, schemas, runtime contracts, CLI flags or workspace contracts.
- [x] Keep the first viewport aligned with the target Mission Control questions: workspace health, current pipeline status, blocker, reviewable evidence/logs and Git publication path.

Non-goals:
- [x] Do not add new API fields, approval persistence, Git diff APIs, live E2E scenarios, reason taxonomy, provider-live release gates or hidden compatibility controls.
- [x] Do not rework the approved visual baseline, saved PNG references or domain model.
- [x] Do not change release readiness semantics; provider-live checks remain manual trusted-machine gates.

Implementation notes:
- Source secondary action must be named as draft preview work, so it does not compete with the primary save-and-validate path.
- Analysis blocker review must keep the operator in Analysis and focus the failed-shard/log drilldown instead of navigating away from the run context.
- Ask inspector primary action must submit the visible Q&A form when Ask is already active, including normal validation for empty questions.
- Publish commit actions, including the inspector primary, must stay disabled while the publish gate has hard blockers such as missing generated artifacts.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Source, Analysis, Ask and Publish interactions.
- [x] Focused UI tests for Source wording, Analysis blocker focus, Ask inspector submit and Publish no-artifacts gate.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started post-audit interaction hardening after comparing the current branch with Epic 16, the target design baseline and rendered local UI screenshots.
- 2026-05-31: Completed 16P after rendered QA found and verified four interaction gaps: Source draft-preview wording, Analysis blocker focus, Ask inspector submit, and Publish hard-blocker disabled state. Verified no backend/API/schema/runtime contract, public live E2E shell, reason taxonomy or release-readiness changes.

### Slice ExecPlan - 16Q Post-audit recovery path hardening

Status: done.

Goals:
- [x] Re-audit Console V2 recovery actions against Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and rendered local UI evidence.
- [x] Fix confirmed operator-flow bugs where an empty-stage recovery action is shown but blocked, or readiness can be bypassed by a competing local action.
- [x] Keep Source -> Readiness -> Analysis -> Review recovery behavior aligned with the approved Mission Control design.

Non-goals:
- [x] Do not add backend API fields, schemas, runtime contracts, CLI flags, workspace contract changes, live E2E scenarios, reason taxonomy or hidden compatibility controls.
- [x] Do not change provider-live release readiness or trusted-machine gate semantics.
- [x] Do not add approval persistence, Git diff APIs or new publication mutations.

Implementation notes:
- Review empty state must make `Run analysis first` a real recovery action and route the operator into Analysis mission control.
- Readiness stage-level `Run first analysis` must require both workspace validation and local readiness pass, matching the right-inspector validate -> doctor -> analysis sequence.
- Analysis direct actions remain available from the Analysis stage for operators who intentionally jump there.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Review empty recovery and Readiness doctor gating.
- [x] Focused UI tests for Review recovery action routing and Readiness first-run gating.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started recovery path hardening after rendered QA showed Review empty-state primary action disabled and Readiness `Run first analysis` enabled before local doctor pass.
- 2026-05-31: Completed 16Q: Review empty-state `Run analysis first` now starts `init` and routes to Analysis mission control, while Readiness `Run first analysis` requires both valid workspace and successful local doctor. Verified with rendered desktop/mobile QA, focused UI tests and full DoD. No backend/API/schema/runtime contract, public live E2E shell, reason taxonomy or release-readiness changes.

### Slice ExecPlan - 16R Post-audit publication gate hardening

Status: done.

Goals:
- [x] Re-audit Publish Git Review Room against Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and the approved saved PNG baseline.
- [x] Fix confirmed publication-gate UI/UX bugs so commit and proposal-branch Git mutations are blocked consistently when hard publish blockers exist.
- [x] Keep operator recovery clear: publication blockers must be visible in the gate panel, right inspector and action disabled states.

Non-goals:
- [x] Do not add backend API fields, schemas, runtime contracts, CLI flags, workspace contract changes, Git diff APIs, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not change Charter/Baseline Git helper behavior; this slice only hardens the Publish room.
- [x] Do not add approval persistence or convert saved design PNGs into product runtime assets.

Implementation notes:
- Publish gate must treat generated artifact absence and shell hard blockers consistently for all publication mutations.
- The proposal branch action is a Git publication mutation in the Publish room, so it must not remain enabled while the gate is blocked.
- Gate copy must not describe checks as merely advisory if the UI blocks Git mutations.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for empty/blocked Publish states.
- [x] Focused UI tests for Publish commit/proposal-branch disabled states and allowed state after artifacts exist.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started publication gate hardening after code audit found Publish commit and proposal-branch actions were not gated consistently with the approved Git Review Room design.
- 2026-05-31: Completed 16R: Publish now carries shell hard blockers into the gate panel, disables both commit and proposal-branch Git mutations under hard blockers, removes misleading advisory gate copy, and makes disabled inspector primary actions visually non-primary. Verified with focused UI tests, rendered desktop/mobile QA, `git diff --check` and full DoD. No backend/API/schema/runtime contract, public live E2E shell, reason taxonomy, Git diff API or provider-live release gate changes.

### Slice ExecPlan - 16S Target UI reference alignment audit

Status: done.

Goals:
- [x] Re-audit all saved `docs/assets/ui-console-v2/*.png` references against the current rendered Console V2 screens.
- [x] Fix confirmed target-design mismatches that are UI-only and can be solved without backend/API/schema/runtime contract changes.
- [x] Keep the first screen of Review and Publish immediately reviewable when artifacts exist.

Non-goals:
- [x] Do not add approval persistence, Git diff APIs, new backend fields, schemas, runtime contracts, CLI flags, workspace contract changes, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not convert saved PNG references into runtime assets.
- [x] Do not change fake runtime artifact generation or provider behavior.

Implementation notes:
- Review Evidence should auto-load the first reviewable artifact when selected-run artifacts exist and no artifact is selected, matching the saved Review Evidence reference.
- Publish preview tabs must match the saved Publish reference and design contract: `Preview`, `Diff`, `Evidence`, `Changelog`; checklist remains in the side publish gate.
- The shared workflow rail should visually align with the saved reference screens as a dark compact ops rail while preserving selectors, keyboard behavior and responsive collapse.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Source, Review Evidence, Review Domain Map, Publish and shared shell.
- [x] Focused UI tests for Review auto-selection and Publish Changelog tab.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started target UI reference alignment after comparing the saved 9-screen PNG baseline with current rendered Source, Analysis, Review, Proposals, Ask and Publish screens.
- 2026-05-31: Completed 16S target alignment: Review Evidence auto-selects the first reviewable artifact, Publish uses `Preview`/`Diff`/`Evidence`/`Changelog`, raw proposal/changelog artifacts survive final-run-index normalization, and the shared workflow rail matches the saved dark ops-console references. No backend/API/schema/runtime/live-shell changes.

### Slice ExecPlan - 16T Stage contextual action and review defaults audit

Status: done.

Goals:
- [x] Re-audit the rendered Console V2 screens against the saved `docs/assets/ui-console-v2/*.png` references after 16S.
- [x] Keep each stage's right-inspector primary action contextual to that stage while still surfacing global blockers in the blocker panel.
- [x] Make Review and Proposals immediately reviewable when selected-run artifacts exist.

Non-goals:
- [x] Do not add Git diff APIs, approval persistence, new backend fields, schemas, runtime contracts, CLI flags, workspace contract changes, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not convert design PNGs into runtime assets or rewrite the existing shell architecture.
- [x] Do not change fake runtime artifact generation or provider behavior.

Implementation notes:
- Open-question blockers should remain visible in `Blockers`, but should not hijack Source, Charter, Proposals or Ask primary next actions.
- Review artifact explorer should be evidence-first (`reports/as-is`, coverage, findings, diagrams, model/domain outputs) with proposal/changelog artifacts later in the list.
- Proposals should auto-load the first proposal/ADR/RFC/checklist artifact when artifacts exist and no proposal artifact is selected, matching the saved Proposals review-room reference.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Source, Review Evidence, Proposals, Ask, Publish and shared shell.
- [x] Focused UI tests for contextual next action, Review explorer order/selection and Proposals auto-preview.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started 16T after rendered audit found Source/Charter/Proposals/Ask next actions were globally overridden by review findings, Proposals opened with an empty preview despite available proposal artifacts, and Review grouped proposal artifacts before evidence groups.
- 2026-05-31: Completed 16T: right-inspector next action is stage-contextual while blockers remain visible, Review explorer is evidence-first with visible selected artifact state, and Proposals auto-loads the first proposal artifact for immediate review. Verified with focused UI tests, rendered desktop/mobile QA, `git diff --check` and full DoD. Also restored local Python 3.9 compatibility for the matrix release contract test and matrix harness type hints so DoD imports and scripted matrix paths run cleanly, with no release behavior change. No backend/API/schema/runtime/live-shell changes.

### Slice ExecPlan - 16U Desktop shell and stage artifact-context audit

Status: done.

Goals:
- [x] Re-audit all saved `docs/assets/ui-console-v2/*.png` target references against rendered desktop/mobile Console V2 screens after 16T.
- [x] Keep the shared activity drawer visible in the desktop first viewport, matching the approved bottom ops-console drawer pattern.
- [x] Prevent proposal/changelog artifact selection from leaking into the Review Evidence first screen when operators return from Proposals, Ask or Publish.

Non-goals:
- [x] Do not add backend API fields, schemas, runtime contracts, CLI flags, workspace contract changes, Git diff APIs, approval persistence, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not convert saved PNG references into runtime assets or rewrite the shell architecture.
- [x] Do not remove access to proposal/changelog artifacts from the Review explorer; only restore stage-appropriate default context on stage entry.

Implementation notes:
- Desktop shell should behave like a fixed-height operator console: top health strip, rail/workbench/inspector and bottom activity drawer all visible, with workbench/inspector scrolling internally.
- Mobile should remain page-scroll friendly and must not introduce horizontal overflow.
- Review stage entry from another stage should re-open the preferred evidence artifact when the selected artifact belongs to Proposals/Changelog; user selection inside Review remains available afterward.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Source, Readiness, Charter, Analysis, Review Evidence, Review Domain Map, Proposals, Ask, Publish and shared shell.
- [x] Focused UI tests for returning to Review from Proposals after proposal auto-preview.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-06-01: Started 16U after rendered reference audit found the desktop activity drawer below the first viewport and stale proposal artifact context leaking into Review Evidence after cross-stage navigation.
- 2026-06-01: Completed 16U: wide desktop now behaves as a fixed-height operator console with rail, workbench, inspector and activity drawer visible in the first viewport; 901-1180px keeps the same shell without clipping the inspector or rail; Review stage entry restores evidence-first artifact context after Proposals/Changelog navigation. Verified against all saved target references with rendered desktop/mobile QA, focused UI tests, `git diff --check` and full DoD. No backend/API/schema/runtime/live-shell changes.

### Slice ExecPlan - 16V Narrow desktop workbench overflow audit

Status: done.

Goals:
- [x] Re-audit current Console V2 against `docs/BACKLOG.md` Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and all saved `docs/assets/ui-console-v2/*.png` references after 16U.
- [x] Fix confirmed narrow desktop UI overflow where the fixed three-column shell leaves the central workbench too narrow for Review/Proposals/Publish stage grids.
- [x] Preserve wide desktop first-viewport shell behavior and mobile page-scroll behavior.

Non-goals:
- [x] Do not add backend API fields, schemas, runtime contracts, CLI flags, workspace contract changes, Git diff APIs, approval persistence, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not convert saved PNG references into runtime assets or rewrite the shell architecture.
- [x] Do not hide operator-facing controls or restore hidden compatibility DOM.

Implementation notes:
- Keep the shared rail, inspector and activity drawer visible in the fixed desktop shell.
- At narrow desktop widths, stage-specific workbench grids should collapse inside the central work area before content clips or creates confusing horizontal work-area scroll.
- Shared section heading rows should wrap controls instead of forcing action buttons outside their card.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for all 9 target screens plus narrow desktop breakpoints.
- [x] Focused UI/type tests for changed frontend behavior where applicable.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-06-01: Started 16V after rendered breakpoint audit found Review evidence controls clipping and central work-area horizontal overflow at 901px while the shared shell itself remained visible.
- 2026-06-01: Completed 16V: stage workbench grids now adapt before fixed desktop shells get too narrow, shared heading rows wrap controls, and Source/Readiness/Charter/Analysis/Review/Proposals/Ask/Publish render without document or central work-area horizontal overflow at 1536px, 1440px, 1366px, 1180px, 1100px, 1024px and the 901px desktop boundary. Mobile/tablet page-scroll behavior remains unchanged below 901px, and no backend/API/schema/runtime/live E2E contract changed.

### Files expected to change
- `ui/src/App.tsx`, `ui/src/styles.css`, `ui/src/components/*`, `ui/src/hooks/*`, `ui/src/lib/*`
- `ui/src/App.test.tsx`, `ui/e2e/live-flow.spec.ts`, `ui/playwright.live.config.ts` only if artifact output behavior changes
- `scripts/frontend-live-e2e.sh` only if the public live shell changes; `scripts/frontend-status-reasons.sh` only if a separate classification slice changes reason taxonomy; `scripts/tests/frontend_live_e2e_contract_test.py`
- `docs/UI_CONSOLE_V2_DESIGN.md`, `docs/BACKLOG.md`, `docs/PLANS.md`
- implementation docs: `README.md`, `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md`, `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

### Acceptance criteria
- [x] All 8 rail stages render and route to the V2 surfaces.
- [x] First viewport shows workspace readiness, current run status, blocker/next action and evidence/Git path where relevant.
- [x] Analysis keeps run timeline, failed shard details and logs adjacent.
- [x] Review supports evidence workbench and domain map without raw-path-first review.
- [x] Ask is explicit read-only Q&A and shows confidence/citations/unresolved assumptions.
- [x] Publish gate shows changed folders, blocker checklist, commit plan and proposal branch.
- [x] Each screen uses existing APIs/data hooks or renders an explicit partial/empty state; no implicit backend contract changes.
- [x] UI tests cover stage navigation, inspector priority and drawer behavior.
- [x] Live E2E `init-inspect` validates Source -> Readiness -> Analysis -> Review -> Publish and captures diagnostics on failure.
- [x] Async Ask is covered by optional `UI_E2E_QA_SMOKE=1` and deterministic tests; domain-map checks are covered by fake-fixture/unit/component diagnostics before any live shell expansion.
- [x] `frontend-e2e-result.json` still records scenario, reason, run id, last run status/error/current step and diagnostic refs.
- [x] Playwright trace/screenshot/video artifacts remain available for failed V2 live E2E stages.
- [x] Existing reason taxonomy remains additive-compatible with release report aggregation.
- [x] Hidden compatibility controls remain absent.

### Risks
- Dense shell can regress responsive layout; visual QA must include desktop and narrow viewport checks.
- Selector migration can weaken release diagnostics if compatibility aliases are dropped without E2E updates.
- Domain map may overpromise if model/edge data is sparse; empty and partial states must be explicit.

### Progress log
- 2026-05-27: Approved unified 9-screen design vision and fixed design baseline, visual references, data-source map, backlog, ExecPlan and V2 live E2E contract without product code changes.
- 2026-05-27: Rebased on `origin/main` `3aa458a`; latest main already removed compatibility controls, compacted activity/artifact surfaces, added optional `UI_E2E_QA_SMOKE=1` and screenshot refs, so V2 backlog was corrected to keep release-facing live frontend on `init-inspect` only.
- 2026-05-28: Continuous backlog queue normalized this plan to the first active workstream; next implementation slice is `16A UI V2 shell foundation`.
- 2026-05-28: Completed `16A UI V2 shell foundation`: shared stage status model, top strip runtime/Git metadata, V2 inspector selectors, Git publication panel, rail keyboard navigation and drawer a11y baseline. No backend/API/schema/live-shell changes.
- 2026-05-28: Completed `16B Source + Readiness consolidation`: Source now has a repo table with explicit advanced-only analysis scope state, Readiness has workspace/repo/runtime/permission/artifact summary cards plus a compact runtime profile summary. No backend/API/schema/live-shell changes.
- 2026-05-28: Completed `16C Charter workbench`: Charter now shows wizard summary, domain/team card overview, baseline prompt bundle status, explicit partial states for missing cards, and a no-404 empty editor selection before existing artifact editor/Git helper actions. No backend/API/schema/live-shell changes.
- 2026-05-28: Completed `16D` through `16N`, finishing Analysis mission control, Review evidence/domain map, Proposals review room, Ask read-only console, Publish gate, UI coverage, live E2E selector/operator-journey migration, optional Ask/domain-map diagnostics and testing/runbook sync. Epic 16 implementation acceptance is complete; remaining item is owner review/merge/archive bookkeeping.

### Plan ID
EP-20260613-live-e2e-execution-ux-artifact-gate

Status: done.

Context:
Live E2E release evidence mixed machine execution quality with artifact/UX quality. The new release model keeps product behavior unchanged, removes the old mixed gate, and makes release readiness composite: machine execution `PASS`, accepted SWE UX report, and accepted SWE artifact-quality report.

Goals:
- [x] Convert live E2E harness/reporting to execution-only machine verdicts.
- [x] Preserve `reports/taskruns/<run_id>-quality.json` as telemetry/evidence only.
- [x] Add required SWE UX and artifact-quality assessment templates and verifier checks.
- [x] Update live E2E docs/runbooks/catalog/agent instructions.
- [x] Update script tests for execution reports, telemetry-only artifact findings, and composite release evidence.

Validation:
- [x] Targeted script tests: `python3 -m unittest scripts.tests.verify_release_verdict_test scripts.tests.live_e2e_plan_test scripts.tests.live_e2e_blackbox_report_test scripts.tests.frontend_live_e2e_contract_test scripts.tests.matrix_release_contract_test scripts.tests.batch_failure_classification_test`
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build` with exact Node `22.21.1`.

Progress log:
- 2026-06-13: Implemented execution-only harness/reporting, required SWE UX/artifact-quality reports, docs/runbook/catalog updates, verifier checks and tests.
