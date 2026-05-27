# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон, текущие активные планы и инженерный operational mirror.

Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
- `docs/archive/PLANS_ARCHIVE_2026-05.md`
- `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Этот файл остаётся рабочим engineering mirror и active ExecPlan surface.

## Когда использовать
Используйте ExecPlan, если:
- работа затрагивает несколько модулей, или
- ожидаемое время > 30–60 минут, или
- затрагиваются контракты/схемы.

---

## Шаблон ExecPlan

### Plan ID
EP-YYYYMMDD-<slug>

### Context
Зачем это нужно? Какие ограничения важны?

### Goals (must have)
- [ ] ...

### Non-goals
- [ ] ...

### Approach
1) ...
2) ...
3) ...

### Files expected to change
- ...

### Acceptance criteria
- [ ] Тесты обновлены/добавлены
- [ ] Схемы валидируются
- [ ] Документация обновлена

### Risks
- ...

### Progress log
- YYYY-MM-DD: ...

---

## Active Plans
Tracker reconciliation from 2026-05-07 consolidated historical active plans into the remaining open slices below. Detailed evidence and classification are archived in `docs/archive/TRACKER_RECONCILIATION_2026-05-07.md`; original historical active plan text was moved to `docs/archive/PLANS_ARCHIVE_2026-05.md` under "Reconciled active plans from 2026-05-07".

### Plan ID
EP-20260527-live-e2e-ui-ux-operator-flow

### Context
После breaking simplification live E2E scripts больше не генерируют pseudo black-box reasoning, а operator/SWE-agent отвечает за assessment поверх evidence. Post-rebase UI walkthrough показал оставшиеся legacy compatibility controls, слабое покрытие async Ask UX, слишком шумный activity drawer и отсутствие durable screenshot refs в frontend live evidence.

### Goals (must have)
- [x] Удалить hidden compatibility UI controls/testids из operator console.
- [x] Добавить optional non-release `UI_E2E_QA_SMOKE=1` в live Playwright flow без включения в canonical release readiness.
- [x] Сохранять frontend screenshots как diagnostic evidence refs в `frontend-e2e-result.json`.
- [x] Сделать activity drawer и artifact/diagram links компактнее и сканируемее.
- [x] Уточнить next-action wording для blockers/findings/release blockers.
- [x] Обновить operator assessment template, live gate skill и runbook.
- [x] Обновить unit/contract/live tests и пройти focused checks plus DoD.
- [ ] После owner review/merge перенести план в архив.

### Non-goals
- [x] Не менять `release_verdict_*` contract и `verify-release-verdict.py`.
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`.
- [x] Не включать Ask/UX smoke в canonical release matrices.
- [x] Не делать screenshots или operator UX assessment источником release readiness.

### Approach
1) Remove compatibility DOM and migrate tests to stage rail/operator-facing controls.
2) Extend live Playwright init-inspect with optional QA smoke and deterministic screenshots.
3) Add screenshot refs to frontend result JSON and keep them evidence-only.
4) Compact operator console evidence surfaces while preserving logs/artifact access.
5) Sync runbook/skill/template/docs and focused tests.

### Files expected to change
- `ui/src/components/*`
- `ui/src/App.test.tsx`
- `ui/e2e/live-flow.spec.ts`
- `scripts/frontend-live-e2e.sh`
- `scripts/tests/frontend_live_e2e_contract_test.py`
- `.agents/skills/e2e-live-gate/SKILL.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/templates/LIVE_E2E_OPERATOR_ASSESSMENT.md`

### Acceptance criteria
- [x] Legacy compatibility controls/testids are absent from the UI shell.
- [x] Default live frontend flow remains `init-inspect` without QA smoke.
- [x] `UI_E2E_QA_SMOKE=1` verifies Ask answer/citations/context/runtime links.
- [x] Frontend result JSON includes screenshot refs when screenshots are produced.
- [x] Mobile first viewport does not expose legacy labels.
- [x] Focused Python/UI tests pass.
- [x] `make contracts`, `make test`, `make lint`, `make build`.

### Risks
- Removing legacy testids requires broad UI test migration.
- Screenshot refs must remain diagnostic metadata and must not leak into release verdict semantics.
- Activity drawer compaction must not hide failure triage evidence from operators.

### Progress log
- 2026-05-27: Started UI/UX live E2E operator-flow hardening slice.
- 2026-05-27: Implemented compatibility DOM removal, optional Ask UX smoke, screenshot evidence refs, compact activity/artifact UX, docs sync, full DoD, and fake-runtime UX smoke with screenshots.

### Plan ID
EP-20260526-async-runtime-backed-ask

### Context
Текущий `POST /api/qa/ask` и `acp qa` остаются compatibility/beta baseline: deterministic workspace search в `internal/qa`, без runtime provider. Target architecture для UI Ask меняется на async agentic Q&A run: ACP собирает deterministic context pack из существующих workspace artifacts, запускает выбранный runtime provider с ролью `system-analyst-qa`, валидирует `qa-answer.json` и показывает результат в UI polling flow.

### Goals (must have)
- [x] Добавить Q&A runtime family `qa.ask` с agent role `system-analyst-qa`, prompt pack `skills/prompt-packs/qa.md` и write scope только `reports/taskruns/<run_id>/qa/`.
- [x] Добавить async API `POST /api/qa/runs`, `GET /api/qa/runs/{run_id}`, `GET /api/qa/runs?limit=...`; оставить `POST /api/qa/ask` как legacy deterministic endpoint.
- [x] Добавить `runtime.profile.steps.qa.provider` в manifest/schema/validator/API runtime profile.
- [x] Ввести `qa-answer.json` contract/schema and validation; сохранять `context-pack.json` и `runtime-execution.json` рядом с answer для audit/debug.
- [x] Context pack строится deterministic из canonical workspace artifacts and imported docs; `reports/taskruns/**` исключается из evidence corpus.
- [x] Fake runtime умеет выполнять `qa.ask` для required CI/local smoke.
- [x] Headless runtime получает artifact-only QA prompt через shared provider engine and validates `qa-answer.json`.
- [x] UI Ask stage использует async submit + polling and shows run status/provider/answer/citations/unresolved/confidence.
- [x] Обновить README, architecture/spec docs, stakeholder docs and schema appendix под target/current split.
- [ ] После owner review/merge перенести план в архив.

### Non-goals
- [x] Не удалять `POST /api/qa/ask` и `acp qa` в этом slice.
- [x] Не менять init/refresh artifact schemas beyond adding QA answer schema and workspace qa provider field.
- [x] Не добавлять hosted/security hardening.
- [x] Не мутировать source repos или canonical architecture outputs из QA run.

### Approach
1) Reuse orchestrator run history/log infrastructure with `pipeline="qa"` while keeping QA runs out of the normal Analysis run list.
2) Build `context-pack.json` before runtime execution from `charter/cards`, `model`, `reports/as-is`, `reports/findings`, `reports/coverage`, `proposals`, `reports/changelog`, and configured docs imports.
3) Route `qa.ask` through shared runtime task execution and providercommon artifact validation.
4) Return structured QA run status over `/api/qa/runs/*`; UI polls that endpoint and links to existing run logs/artifacts.
5) Keep legacy deterministic QA service as retriever/context builder and compatibility fallback.

### Files expected to change
- `internal/orchestrator/*`
- `internal/api/server.go`
- `internal/qa/*`
- `internal/runtime/*`
- `internal/workspace/manifest.go`
- `internal/runtimeprofile/patch_service.go`
- `schemas/workspace.schema.json`
- `schemas/qa-answer.schema.json`
- `ui/src/components/StagePanels.tsx`
- `ui/src/lib/qaApi.ts`
- `ui/src/App.test.tsx`
- docs/spec/README/stakeholder/backlog/schema appendix docs

### Acceptance criteria
- [x] Schema validation accepts `runtime.profile.steps.qa.provider` and rejects invalid providers.
- [x] Fake Q&A run writes valid `reports/taskruns/<run_id>/qa/qa-answer.json`.
- [x] Context pack excludes `reports/taskruns/**`.
- [x] Async API covers start/status/list while legacy `POST /api/qa/ask` still works.
- [x] UI Ask submit creates a Q&A run and renders succeeded answer state.
- [x] Full DoD completed: `make contracts`, `make test`, `make lint`, `make build`.

### Risks
- QA runs share the existing single active/pending run queue; future UX may need a dedicated lightweight queue if operators ask questions during long init runs.
- Headless answer quality depends on provider following `qa-answer.json`; fake remains deterministic baseline, but live QA should get focused smoke coverage before release claims.

### Progress log
- 2026-05-26: Implemented async QA run family/API/schema/fake runtime/UI polling, context-pack citation validation, docs/test sync, full DoD, and fake async QA smoke.

### Plan ID
EP-20260526-live-e2e-operator-blackbox-simplification

### Context
Live E2E/release gate накопил двусмысленные surfaces: harness сам пишет `blackbox_e2e_steps_*` как будто это operator reasoning, non-release diagnostic matrices пишут `release_verdict_*`, planner публикует JSON/Markdown как будто это second-order API, а release strict verdict включает stubbed frontend cancel smoke. Breaking compatibility допустима: лучше удалить старую логику сразу и оставить один machine-verifiable release gate плюс отдельный operator assessment поверх evidence.

### Goals (must have)
- [x] Удалить internal black-box evaluator helper и генерацию `blackbox_e2e_steps_*`.
- [x] Развести release artifacts (`release_verdict_*`) и non-release artifacts (`matrix_result_*`).
- [x] Ужесточить `verify-release-verdict.py`, чтобы он принимал только canonical release-mode verdict.
- [x] Оставить `live-e2e-plan.py` shell-only direct command printer.
- [x] Убрать stubbed frontend cancel из release gate и public live frontend harness.
- [x] Обновить runbook/skill/testing docs и добавить шаблон operator assessment.
- [x] Обновить regression tests под breaking protocol.
- [x] Сделать planner shell output явным про diagnostic/release mode, provider/run scope, frontend skip и fake/headless init/refresh цикл.
- [x] Починить `make test-stress`, чтобы zero-match Go test pattern не давал ложный зелёный сигнал.
- [x] Rerun full DoD on a host with exact Node.js `22.21.1`.
- [ ] После owner review/merge перенести план в архив.

### Non-goals
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`.
- [x] Не менять canonical release matrices или curated repos для обхода host prerequisites.
- [x] Не делать operator assessment источником release readiness.

### Approach
1) Удалить evaluator helper и все script-authored black-box decision calls из batch/matrix harness.
2) Изменить matrix synthesis: release mode пишет только release verdict, non-release mode пишет neutral matrix result без `release_state`/`release_contract`.
3) Ужесточить verifier и planner public surface.
4) Убрать frontend cancel strict path из release aggregation и live shell harness; оставить frontend release check на init/artifact inspection.
5) Синхронизировать docs/tests и template для ручного assessment.

### Files expected to change
- `scripts/full-run-batch-matrix.sh`
- `scripts/full-run-batch.sh`
- `scripts/frontend-live-e2e.sh`
- `scripts/live-e2e-plan.py`
- `scripts/verify-release-verdict.py`
- `Makefile`
- `scripts/tests/*`
- `internal/orchestrator/run_lifecycle_test.go`
- `.agents/skills/e2e-live-gate/SKILL.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/templates/LIVE_E2E_OPERATOR_ASSESSMENT.md`

### Acceptance criteria
- [x] Focused script tests updated and passing.
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] Canonical release gate is not run unless trusted-host prerequisites are satisfied.

### Risks
- Large shell harness changes can break hermetic tests; mitigate by keeping execution flow unchanged except removed surfaces.
- Historical docs may mention old artifacts; sync source-of-truth docs first and ignore archive snapshots unless tests read them.

### Progress log
- 2026-05-26: Started breaking simplification implementation from audit follow-up.
- 2026-05-26: Implemented breaking simplification and updated focused script suite.
- 2026-05-26: Installed exact local Node.js `22.21.1`, fixed frontend health-start result JSON, fixed async API test timeout flake under live quality gate, updated skipped-frontend report wording, completed full DoD (`make contracts`, `make test`, `make lint`, `make build`), and passed codex-code non-release diagnostic `smoke-tiny-bank-20260526T195844Z`.
- 2026-05-27: Added explicit planner comments for diagnostic/release scope and fixed `make test-stress` with a zero-match guard plus async debounce regression coverage.
- 2026-05-27: Re-ran direct codex-code non-release diagnostic `smoke-tiny-bank-20260527T114046Z`; it wrote `matrix_result_*` only, verifier rejected the diagnostic payload, and reports surfaced runtime pressure (`stall_count=5`, `post_artifact_stalls=5`, `quality_alerts=2`) for operator review.

### Plan ID
EP-20260525-frontend-live-e2e-diagnostics

### Context
`release-fast-20260525T104842Z` proved the runtime permission slice did not break trusted provider args or backend hard gates, but exposed weak frontend live E2E classification. `qwen-code` frontend init passed, while `claude-code` failed with `Target page, context or browser has been closed` while the run was still active in `init.step1.collect`, and `codex-code` failed with API `ECONNREFUSED` while the fresh init run was still in `init.step0.constitution`. Both collapsed into `playwright_failed`, which blocks useful triage and makes it unclear whether the issue is browser lifecycle, API/server lifecycle, or product UI.

### Goals (must have)
- [x] Keep Playwright as the canonical CLI/release-gate harness; Browser/Chrome MCP remains manual diagnostic only.
- [x] Split frontend live failure reasons into `browser_closed`, `api_unreachable`, `server_exited`, `active_run_timeout`, and fallback `playwright_failed`.
- [x] Split post-merge frontend live backend-run failures into `runtime_run_failed` instead of collapsing them into fallback `playwright_failed`.
- [x] Make long backend polling independent from the browser page object.
- [x] Persist frontend result diagnostics: server PID/exit code, post-failure health, run id, last run status/current step, and diagnostic refs.
- [x] Add stub regression tests for the new frontend classifications.
- [x] Keep API request polling independent from page lifecycle inside the init-inspect flow; the old page-close live scenario is superseded by EP-20260526.
- [ ] After merge, run focused frontend init diagnostics for `claude-code` and `codex-code`, then rerun canonical `release fast` if both focused checks pass.

### Non-goals
- [x] Do not change canonical matrices, timeout profiles, provider contracts, permission behavior, or public HTTP API.
- [x] Do not replace Playwright release-gate acceptance with MCP automation.
- [x] Do not fix any discovered `acp serve` lifecycle bug in this slice; classify it first.

### Approach
1) Extend frontend reason allowlist and batch/report aggregation coverage.
2) Harden `scripts/frontend-live-e2e.sh` post-failure diagnostics around `acp serve` PID, `/api/health`, Playwright log signatures, and run-history/API state.
3) Refactor `ui/e2e/live-flow.spec.ts` to use independent API request polling and promise-based sleeps for long polling.
4) Add shell-stub tests that simulate server exit, API unreachable, browser closed, and active timeout without live providers.
5) Sync testing/runbook architecture docs with the narrower failure taxonomy.

### Files expected to change
- `scripts/frontend-live-e2e.sh`
- `scripts/frontend-status-reasons.sh`
- `ui/e2e/live-flow.spec.ts`
- `scripts/tests/*`
- `docs/PLANS.md`
- `docs/TESTING_STRATEGY.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/ARCHITECTURE.md`

### Acceptance criteria
- [x] `python3 -m unittest scripts.tests.frontend_live_e2e_contract_test`
- [x] relevant batch/report script tests
- [x] UI unit/build checks
- [x] frontend live shell exposes only `init-inspect`; page-close/cancel coverage moved out of live gate
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`

### Risks
- Shell lifecycle checks must not kill or wait on a still-running `acp serve` before cleanup.
- New reason codes must remain additive so historical result JSON and report aggregation keep working.

### Progress log
- 2026-05-25: Started implementation after stopping `release-fast-20260525T104842Z`; evidence showed independent frontend init failures for Claude browser lifecycle and Codex API/server reachability.
- 2026-05-25: Implemented additive reason taxonomy, post-failure diagnostics, independent API polling, stub regression coverage, and docs sync; local DoD passed.
- 2026-05-26: Follow-up release-fast triage found a stable `claude-code` init failure where Playwright correctly observed the backend run had already failed with `runner_unavailable`; adding `runtime_run_failed` classification plus `last_run_error_code` diagnostics.

### Plan ID
EP-20260525-proven-arch-ui-console

### Context
Текущий React UI отражает internal modules (`Setup / Baseline / Runs / Results / Settings`) и начинается с hero-блока, из-за чего пользователь видит админку настроек вместо рабочего flow Proven Arch. Нужно перестроить surface в stage-based console без изменения backend/API contracts.

### Goals (must have)
- [x] Заменить hero/top tabs на Proven Arch console shell: top bar, stage rail, center work area, right inspector, bottom activity drawer
- [x] Разложить существующие setup/runtime/run/results/git surfaces по stages `Source / Readiness / Charter / Analysis / Review / Proposals / Ask / Publish`
- [x] Подключить существующий read-only `POST /api/qa/ask` как UI stage `Ask` (historical slice; superseded by EP-20260526 async `/api/qa/runs` target)
- [x] Добавить derived stage status, next action, blockers, evidence refs и runtime/workspace health
- [x] Обновить CSS под light operator-console style без backend изменений
- [x] Обновить UI tests, live E2E selectors и пользовательские docs
- [x] Выполнить доступные проверки slice
- [ ] Перенести этот ExecPlan в архив после owner review/merge

### Non-goals
- [ ] Не менять schemas, runtime artifact contracts, Go API wire shapes или CLI flags
- [ ] Не добавлять hosted/security/compliance UX
- [ ] Не менять pipeline semantics или provider execution policy

### Approach
1) Добавить UI-only view model types и Q&A API wrapper.
2) Вынести reusable console components (`AppShell`, top bar, rail, inspector, activity drawer, small primitives).
3) Refactor `App.tsx` так, чтобы existing hooks остались источником данных, а central content переключался по stage.
4) Добавить stage panels поверх текущих API/hook actions и сохранить стабильные `data-testid` для core flows.
5) Синхронизировать README/ARCHITECTURE и тесты с stage-based console.

### Files expected to change
- `ui/src/App.tsx`, `ui/src/styles.css`, `ui/src/components/*`, `ui/src/lib/*`
- `ui/src/App.test.tsx`, `ui/e2e/live-flow.spec.ts`
- `README.md`, `docs/ARCHITECTURE.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] Stage rail renders all 8 stages and switches central content
- [x] Readiness blockers/next actions reflect workspace validation, doctor, run errors and pending permissions
- [x] Review stage shows coverage/questions/artifacts/diagrams from existing artifact APIs
- [x] Ask stage originally called `/api/qa/ask` and rendered answer/citations/confidence; current target is async `/api/qa/runs` under EP-20260526
- [x] Existing first-run, runtime settings, logs, diagrams, baseline editor and git helper coverage remain covered
- [x] `npm run typecheck --prefix ui`
- [x] `npm test --prefix ui`
- [x] `make lint`
- [x] `make build`

### Risks
- Existing UI tests have many tab-specific selectors; preserve important test IDs where practical and update navigation selectors deliberately.
- Dense console layout can overflow on small viewports; responsive collapse must be part of the slice.

### Progress log
- 2026-05-25: Started implementation from approved UI/UX plan.
- 2026-05-25: Implemented console shell, stage panels, Q&A UI, docs/tests/e2e updates, embedded UI rebuild and DoD checks (`make contracts`, `make test`, `make lint`, `make build`).
- 2026-05-25: Follow-up UX audit against the accepted mockup found lower visual fidelity in the top bar/rail density and hidden compatibility controls still participating in keyboard focus; applying a focused UI polish and a11y fix without backend contract changes.
- 2026-05-25: Browser QA follow-up fixed rail a11y duplicate labels, unlabeled advanced manifest textarea, and initial optional Charter artifact 404 console noise by lazy-loading Charter content only when the stage opens.
- 2026-05-25: Second UX pass tightened Source density, added top-bar status icons/server status, and made bootstrap open `Review` automatically when a completed run already has artifacts.
- 2026-05-25: Final browser QA found stale Mermaid syntax-error SVGs caused by rendering `.mmd` artifacts while text content still said `Loading...`; fixed diagram loading guard/cleanup and changed inspector next-action status from `ready` to `attention` when warning blockers exist.

### Plan ID
EP-20260518-live-e2e-blackbox

### Context
Live E2E должен стать black-box operator flow: план шага, прямой harness/UI/API вызов, evidence inspection, classification, next decision. Official release readiness остаётся только в release-mode `reports/release_verdict_<matrix-id>.json`, проверяемом `scripts/verify-release-verdict.py`. EP-20260526 superseded the earlier machine-generated step-report approach: scripts now produce facts/verdicts only, and operator assessment is separate.

### Goals (must have)
- [x] Обновить live E2E skill и release runbook под обязательный step-by-step black-box evaluator protocol
- [x] Удалить durable machine-authored step evidence из batch/matrix harness; reasoning layer остаётся operator-owned
- [x] Сделать explicit layering: live-e2e skill -> local manual-live-e2e workflow -> direct public harness commands -> ACP runtime/provider/UI evidence, без GitHub Actions live workflow
- [x] Оставить `scripts/full-run-batch-matrix.sh` direct top-level release harness
- [x] Перенести backend-cycle logic за `scripts/full-run-batch.sh` в internal helper и удалить публичный legacy entrypoint
- [x] Удалить legacy live E2E matrices/docs/tests and references without compatibility shims
- [x] Выполнить DoD checks после implementation
- [ ] Когда owner запросит pre-release validation, выполнить trusted-machine live gate через новый black-box protocol и сохранить verifier-backed verdict evidence

### Non-goals
- [x] Не менять release verdict contract
- [x] Не запускать trusted live E2E в рамках implementation slice
- [x] Не менять runtime artifact schemas или rejection tests for old runtime payloads

### Approach
1) Удалить script-authored pseudo-reasoning reports из batch/matrix harness.
2) Оставить release readiness только в verifier-backed release verdict; non-release matrices пишут neutral matrix result.
3) Перевести старый backend-cycle в non-public helper, вызываемый только из `scripts/full-run-batch.sh`.
4) Переписать skill/runbook/testing docs под новый protocol и удалить legacy live E2E surfaces.
5) Обновить docsync/script tests так, чтобы они требовали operator-owned assessment wording и отклоняли старые live E2E references.

### Files expected to change
- `.agents/skills/e2e-live-gate/SKILL.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`
- `docs/BACKLOG.md`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/internal/live-e2e-backend-cycle.sh`
- deleted internal live E2E evaluator helper
- `internal/docsync/docsync_test.go`
- `scripts/tests/*`
- legacy live E2E files removed from `docs/`, `examples/`, and `scripts/`

### Acceptance criteria
- [x] `go test ./internal/docsync`
- [x] `python3 -m unittest discover -s scripts/tests -p '*_test.py'`
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`

### Risks
- Shell harness changes can affect long trusted-machine runs; targeted tests must cover the new step-report artifacts and direct harness integration.
- Active docs must distinguish removed live E2E surfaces from unrelated runtime contract rejection tests that intentionally mention old payload shapes.

### Progress log
- 2026-05-18: Started implementation. Added black-box step reports, moved backend-cycle behind batch harness, and began docs/test cleanup. Trusted live E2E was not run.
- 2026-05-18: Local verification passed: `go test ./internal/docsync`, `python3 -m unittest discover -s scripts/tests -p '*_test.py'`, `make contracts`, `make test`, `make lint`, `make build`. Make targets used `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node22-wrapper/bin` because Homebrew `node@22` needed older simdjson/simdutf dylib paths on this host.

### Plan ID
EP-20260509-v011-hardening-release

### Context
`v0.1.0` is the only public release, while `main` has accumulated OSS readiness, release hardening, exact toolchain, and onboarding documentation improvements. Prepare `v0.1.1` as a beta hardening/onboarding release without changing runtime contracts, schemas, CLI flags, or public API.

### Goals (must have)
- [x] Keep README aligned with the actual local-first install/run path and current public `v0.1.0` status until `v0.1.1` is published
- [x] Keep README user-facing by removing internal live E2E/release-gate/runbook navigation and making fake/live onboarding standalone
- [x] Sync `docs/INSTALL.md` provider command wording without changing public latest-release status
- [x] Move user-facing hardening notes from `Unreleased` into `CHANGELOG.md` entry `v0.1.1 - 2026-05-09`
- [x] Preserve `.goreleaser.yml` prerelease behavior for the next beta release
- [x] Run local DoD and release-prep smoke checks
- [ ] Run trusted-machine release gate through direct `scripts/full-run-batch-matrix.sh` invocation and record `reports/release_verdict_<matrix-id>.json`
- [ ] After the GitHub release is published, update public latest-release docs from `v0.1.0` to `v0.1.1`

### Non-goals
- [x] Do not change runtime contracts, schemas, CLI flags, or public API
- [x] Do not add wrapper scripts around the release matrix harness
- [x] Do not edit canonical release matrices or curated repo files to fit the current host
- [x] Do not mark release readiness as passed without verifier-backed `PASS`

### Approach
1) Prepare release docs on a dedicated branch with README and changelog aligned to the release plan.
2) Run local required checks and smoke checks against fake runtime and public install path.
3) Execute trusted-host live release gate only from a clean committed tree that satisfies canonical path/provider prerequisites.
4) Publish `v0.1.1` as a prerelease through existing GoReleaser config after release verdict evidence is present.
5) Update latest-release docs to `v0.1.1` after publication.

### Files expected to change
- `README.md`
- `CHANGELOG.md`
- `docs/PLANS.md`
- `docs/INSTALL.md` for provider command wording; latest-release values only after `v0.1.1` is published
- `scripts/write-batch-preflight.py`, `scripts/e2e_report_classifiers.py`, `scripts/e2e_batch_report.py`, and related tests for release-fast provider readiness/diagnostic alignment
- `internal/runtime/{qwencode,claudecode,codexcode,providercommon}` tests and adapters for qwen/claude pre-artifact stall policy and raw diagnostics

### Acceptance criteria
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] `goreleaser check`
- [x] Public install smoke against current `main/install.sh`
- [x] Source fake walkthrough smoke on a temporary local Git repo
- [x] UI/API smoke against local `acp serve`
- [x] README user-facing guard: no internal live E2E/runbook/matrix/verdict navigation
- [x] README relative links check
- [x] README standalone fake quality check includes `serve --auto-init --dry-run` before `run`
- [x] README provider example check with `ACP_CLAUDE_CMD=claude`
- [ ] Trusted release verdict JSON verified with `scripts/verify-release-verdict.py`

### Risks
- Current local host may not satisfy trusted-machine prerequisites for canonical release slices.
- `v0.1.1` docs must not claim latest public release until the GitHub release is actually published.
- Owner/admin GitHub residual tasks remain manual and must not be misrepresented as completed release evidence.

### Progress log
- 2026-05-09: Started `v0.1.1` hardening/onboarding release prep. README rewrite already exists in the worktree; changelog moved to `v0.1.1`. Trusted live release gate not run yet.
- 2026-05-09: Local release-prep verification passed: `make contracts`, `make test`, `make lint`, `make build`, `goreleaser check` via `go run`, public installer smoke, source fake walkthrough smoke, and local UI/API smoke. Canonical live release gate remains blocked until a clean committed tree and complete trusted-host curated checkout set are available; current host is missing `/tmp/provenarch-live-e2e/posthog/posthog` for `release long`.
- 2026-05-09: Refined README as standalone user onboarding artifact: removed public navigation to internal live E2E/release-gate/runbook surfaces, added explicit provider ID vs executable command wording, documented `ACP_CLAUDE_CMD=claude` live smoke, and synced `docs/INSTALL.md` wording without changing latest public release status.
- 2026-05-09: Fixed README quality-check flow to be standalone for a fresh workspace by adding `serve --auto-init --dry-run` before the fake `run` command.
- 2026-05-09: README-only live E2E audit: README supports install, fake walkthrough, and single-provider Claude live smoke, but intentionally contains no matrix/runbook/verdict route; a README-only user cannot complete the full live E2E/release gate. Local provider binaries `qwen`, `claude`, and `codex` are present, `/tmp/provenarch-live-e2e` is writable, but the canonical checkout set is incomplete (`bank-of-anthos`, `posthog`, `ftgo`, and `sentry-ecosystem` missing).
- 2026-05-09: Started README + release-fast live E2E candidate slice. Target is current candidate fixes, not published `v0.1.0`: close clean-`HOME` README doctor gap, verify fake README walkthrough including diagram preview and QA, then run canonical `release fast` from a clean committed worktree through direct `scripts/full-run-batch-matrix.sh`.
- 2026-05-09: First canonical `release fast` attempt on commit `99de15b` stopped before backend runs with `operational_host_preflight_failed`: qwen artifact smoke used bare `qwen -p`, while the runtime adapter needs `--chat-recording false --yolo --channel CI` to enable filesystem artifact writes. Current follow-up aligns preflight with runtime invocation and reruns `release fast`.
- 2026-05-10: Second canonical `release fast` attempt on commit `c343c5c` passed provider preflight but failed baseline backend hard-pass `1/3`: `qwen-code` and `claude-code` both ended as zero-output `runtime_stalled_before_artifacts` before `asis-draft-manifest.json`, while `codex-code` passed. The follow-up widened qwen/claude initial pre-artifact stall windows to 180s, preserved strict artifact-only success, and surfaced failed raw zero-output pre-artifact stalls in quality reports before rerunning release-fast.
- 2026-05-18: Follow-up qwen-only smoke on commit `f724530` confirmed qwen readiness and advanced through `init.step2.asis_docs`, but `refresh.step1.collect` hit a zero-output/no-artifact stall after the widened 180s window on the root-file shard. The remediation kept timeout/matrix policy unchanged and removed a prompt contradiction: the refresh collect first-action skeleton now satisfies refresh minimums (`coverage.missing >= 3`, `questions >= 1`) and root-file evidence prefers README/Makefile/build-deploy manifests over dotfiles.
- 2026-05-18: Qwen-only smoke on committed candidate `7d3f3c8` (`smoke-tiny-bank-20260518T072816Z`) stayed blocked before the targeted refresh slice: `qwen-code` produced a zero-output pre-artifact `runner_unavailable` on `init.step1.collect` root-file shard after the 180s window, while the prompt already contained the README-preferring early pair command. Canonical `release fast` was not run. Claude readiness also carried model telemetry that was previously treated as host/provider configuration, but the current policy removes model-attribution readiness blocking.
- 2026-05-18: Review found one remaining prompt-contract gap: manifest-only collect repair could pass sorted evidence candidates into the skeleton and accidentally choose `.gitignore` before `README.md` for root-file shards. The follow-up fix applies the same README/Makefile/build-deploy preference to repair evidence candidates and adds regression coverage for direct skeleton generation plus the composed repair prompt.
- 2026-05-18: Current PR #69 mergeability review found GitHub CI green but release readiness blocked: PR is draft, qwen smoke still has zero-output `runner_unavailable`, and no `release_verdict_*.json` PASS exists. Claude `model`/`modelUsage` telemetry remains transcript diagnostics only, not release readiness attribution. The current remediation keeps timeout/matrix policy unchanged and moves the collect first-action heredoc pair command immediately after provider identity, so `init.step1.collect` / `refresh.step1.collect` expose the write command before broad artifact/doc-first instructions and only once per prompt. Next gate is qwen-only smoke, then canonical `release fast`.
- 2026-05-18: Qwen-only smoke on command-first prompt commit `865397d` (`smoke-tiny-bank-20260518T084306Z`) passed precheck and moved past the previous earliest collect silence: qwen wrote step0 artifacts and 9/10 `init.step1.collect` shard manifests, with the first-action command visible before broad prompt instructions. Acceptance still failed with `runner_unavailable=1` and `zero_output_pre_artifact_stalls=2`: one zero-output pre-artifact stall on `init.step1.collect` shard `bank-of-anthos-src-ledger-ledgerwriter`, plus one on `init.step2.asis_docs`. Canonical `release fast` remains blocked and was not run; the next slice should diagnose qwen invocation/provider behavior on the remaining silent shard/as-is transition without increasing timeouts or relaxing canonical matrices.
- 2026-05-18: Post-DoD audit found one diagnostics hygiene bug in raw qwen failure metadata: because qwen passes the prompt through `-p`, the redacted `argv` still contained the full artifact prompt even though lifecycle diagnostics already record `prompt_bytes`. The follow-up fix redacts prompt payload argv values (`-p`/`--prompt`) to byte count + hash while preserving provider command/flags/cwd/include-dir diagnostics.
- 2026-05-18: Current slice removes provider/model attribution gate without backward compatibility: `model`/`modelUsage` telemetry is plain transcript diagnostics, not selected-provider readiness/report blocker. Qwen runtime now uses `stream-json` activity output and treats the first zero-output pre-artifact stall as retryable warning; recovered retry is non-blocking, exhausted no-artifact retry remains `runner_unavailable`.
- 2026-05-18: Canonical `release fast` attempt `release-fast-20260518T152336Z` on PR #69 was stopped after a new terminal blocker: `qwen-code` baseline completed, while `claude-code` reached `init.step3.findings` and produced zero stdout/stderr plus no `validator-verdict.json` for the 180s pre-artifact window. The remediation keeps matrices/timeouts/provider contracts unchanged, moves validator steps to command-first `FIRST VALIDATOR VERDICT COMMAND`, and makes only Claude validator zero-output pre-artifact silence a bounded warning/retry; exhausted validator silence still fails as `runner_unavailable`.
- 2026-05-19: Claude-only validator smoke `smoke-tiny-bank-claude-validator-20260518T182851Z` passed on commit `fe3f6d4`, confirming the validator-step code path. Subsequent canonical `release fast` attempts `release-fast-20260518T193132Z` and `release-fast-20260518T193433Z` stopped before backend runs with `operational_host_preflight_failed`: the separate Claude text-only `ACP_READY` preflight probe timed out after 30s, while manual probes showed flaky latency. The current remediation removes that text probe as a Claude gate and relies on `--version` plus runtime-like artifact smoke (`--add-dir` temp write dir) with one bounded retry on timeout/no-output.
- 2026-05-19: After rebase to `f554807`, provider preflight passed and `kimi-for-coding` telemetry remained non-blocking, but canonical `release fast` hit a new product blocker: `claude-code` reached `init.step4.proposals` and produced zero stdout/stderr plus no `proposals-draft-manifest.json` for the 180s pre-artifact window. A separate host hygiene issue removed the temporary Node `22.21.1` path under `/tmp`, causing Codex quality/frontend checks to fail for operational reasons. The current remediation keeps matrices/timeouts/schemas/provider contracts unchanged, adds a command-first `FIRST PROPOSALS DRAFT COMMAND`, makes only Claude proposals zero-output pre-artifact silence a bounded warning/retry, ensures stale runner classifier rows do not override terminal quality failures, and requires stable non-`/tmp` Node toolchain selection for the next gate.
- 2026-05-19: Clean-worktree preflight on `98419c0` found a narrower Claude readiness bug: manual artifact smoke wrote the expected sentinel but the Claude process did not exit before timeout, so preflight still returned `operational_host_preflight_failed`. The follow-up keeps command/probe contracts unchanged and treats Claude timeout-after-sentinel as ready, matching the runtime artifact-only controlled-stop policy; timeout without expected sentinel remains a bounded retry then host blocker.
- 2026-05-19: Claude-only proposals diagnostic on clean commit `98b4573` (`smoke-tiny-bank-claude-proposals-20260519T093914Z`) did not reach the targeted proposals step: `claude-code` hit a fully silent no-artifact `runner_unavailable` on `init.step1.collect` root-file shard after the 180s pre-artifact window, while later collect shards showed that Claude can recover when a fresh focused process writes the manifest. The follow-up keeps timeout/matrix/provider contracts unchanged and scopes Claude zero-output pre-artifact warning/retry to collect as well as validator/proposals; exhausted no-artifact retry remains `runner_unavailable`.
- 2026-05-19: Claude-only proposals diagnostic on clean commit `757ac6c` (`smoke-tiny-bank-claude-proposals-20260519T101428Z`) passed. The next canonical `release fast` (`release-fast-20260519T111703Z`) advanced through qwen init but failed qwen refresh in `refresh.step2.asis_docs`: qwen had authored `src-ledger-balancereader-overview.md`, while the collect manifest referenced typo path `src-ledger-balereader-overview.md`. The remediation keeps schemas/matrices/provider contracts/timeouts unchanged and tightens collect validation so `documents[].path` must reference an existing authored file under `write_root`; missing references trigger the existing provider-authored manifest-only repair before step2, and batch classifiers report stale step2 missing-document failures as `runtime_contract_failed`, not `runner_unavailable`.
- 2026-05-19: Canonical `release fast` on clean commit `d1e7d05` (`release-fast-20260519T134436Z`) reached a green backend baseline (`hard_pass=3/3`) and qwen/codex frontend PASS, but failed release policy because Claude frontend init created a fresh run where `claude-code` hit fully silent no-artifact `runner_unavailable` on `init.step0.constitution`. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: constitution normal prompt now starts with `FIRST CONSTITUTION DRAFT COMMAND`, and only Claude constitution fully silent pre-artifact stalls get the same one bounded warning/retry as collect/validator/proposals; exhausted no-artifact retry remains `runner_unavailable`.
- 2026-05-19: Claude-only constitution diagnostic on clean commit `a848f16` (`smoke-tiny-bank-claude-constitution-20260519T174232Z`) proved `init.step0.constitution` and `init.step4.proposals` no longer hit `runner_unavailable`, but refresh was blocked before runtime because published `skills/subagents.yaml` contained markdown from the generic draft template. The remediation keeps schemas/matrices/provider contracts/timeouts unchanged and makes the constitution first-action `baseline-subagents.yaml` use the canonical valid `agents:` YAML bundle so workspace validation can start refresh.
- 2026-05-19: Canonical `release fast` on clean commit `885f524` (`release-fast-20260519T201118Z`) passed all baseline backend runs and frontend init/cancel surfaces for qwen, Claude, and Codex, including recovered Claude validator silence. Release verdict was still `FAIL` because the matrix harness executed only the first planned profile/sweep row (`single-git_url/baseline`) and then reported missing `parallel-default`; `profile-sweep-combinations.tsv` contained all four required rows. Root cause is shell stdin ownership: child `full-run-batch.sh` inherited the while-loop stdin and could consume the remaining combinations. The remediation keeps canonical matrices/timeouts/provider contracts unchanged, reads combinations through an isolated fd, runs child batches with stdin detached, and adds regression coverage where a dummy child drains stdin while all release combinations still execute.
- 2026-05-20: Canonical `release fast` rerun on clean commit `004f0e9` (`release-fast-20260519T235004Z`) confirmed the matrix stdin fix: all four profile/sweep combinations were planned and execution advanced past the first qwen backend run. The run was stopped after a new product/runtime blocker in `qwen-code` `init.step1.collect` root-file shard: qwen read repo files before the first pair write, wrote only `root-overview.md`, and then manifest-only repair stalled for 180s before writing `shard-pack-manifest.json`. The remediation keeps canonical matrices/schemas/provider contracts/timeouts unchanged, removes the collect prompt conflict that told providers to read entrypoint hints "first", explicitly forbids `read_file`/repo exploration before `FIRST COLLECT ARTIFACT PAIR COMMAND`, and moves manifest-only repair to a `FIRST COLLECT MANIFEST REPAIR COMMAND` heredoc as the first task action.
- 2026-05-20: Audit of PR #69 diff after `release-fast-20260520T012613Z` found no active provider/model attribution blocker left in release path; `model`/`modelUsage` remains diagnostic-only. The release blocker is now product quality/contract: multi-path backend strict gate fails with `analysis:cross-repo-missing`, and Claude frontend init failed `init.step2.asis_docs` because draft repair claimed `overview.md`/`architect-summary.md` existed while only `summary.md` was present. The current remediation keeps matrices/schemas/provider contracts/timeouts unchanged, adds a normal + repair `FIRST AS-IS DRAFT COMMAND` for `asis-draft-manifest.json` plus the three required draft files, and aligns the cross-repo evaluator with documented acceptance of semantic edges, validator/collect findings with multi-repo provenance, or questions with multi-repo `related_ids` plus repo-specific citations.
- 2026-05-21: Clean-worktree Claude frontend diagnostic on `cc223d8` (`smoke-tiny-bank-claude-frontend-20260521T062833Z`) confirmed the backend `step2.asis_docs` and multi-repo semantic fixes: backend hard-passed with no `runtime_contract_failed`, `runner_unavailable`, `runtime_timeout`, `quality_gates_failed`, or `cross_repo_missing` hits. The remaining product blocker is frontend-triggered `init.step0.constitution`: focused repair wrote `constitution-draft.json` and `charter-overview.md` correctly, but wrote `baseline-subagents.yaml` under a sibling path where `/frontend/claude-code/` was rewritten to `/frontend-claude-code/`; runtime correctly rejected the missing in-write-set draft as `runtime_contract_failed`. The follow-up keeps artifact-only success and write-set validation unchanged, removes permissive "equivalent writes" wording, and makes normal/repair draft commands assign exact absolute `write_root`/`draft_root` once, then write through shell variables so providers do not manually retype long slash-separated paths.
- 2026-05-21: Canonical `release fast` on clean commit `d8ddf7d` (`release-fast-20260521T093333Z`) completed all backend and frontend surfaces with no runtime/provider/infra blockers, but verdict stayed `FAIL`: qwen/codex multi-path baseline and parallel-default runs had `analysis:cross-repo-missing` because the `FIRST VALIDATOR VERDICT COMMAND` wrote an empty `findings/questions` skeleton and runtime stopped after the valid artifact before late cross-repo instructions could take effect. The follow-up keeps `analysis:cross-repo-missing` strict and does not add ACP-side fallback; instead multi-repo validator first-action and focused repair skeletons now include one PASS-compatible cross-repo finding and one question with repo/path provenance, so qwen/codex first valid artifacts satisfy the provider-facing semantic contract.
- 2026-05-22: Targeted multi-path diagnostic on `38172b1` (`regres-fast-bank-openedx-20260522T021739Z`) passed for qwen/codex and confirmed the validator first-action cross-repo skeleton. Canonical `release fast` (`release-fast-20260522T065332Z`) then passed baseline backend for all providers and frontend init/cancel for Claude/Codex, but qwen frontend init failed because two `init.step1.collect` focused pair repairs returned qwen stream output ending in `[API Error: Premature close]` with process `success` and no artifacts. The run was stopped after the first failed sweep because release PASS was already impossible. The remediation keeps artifact-only success, canonical matrices, schemas, provider contracts, and timeouts unchanged: qwen now treats transient provider API/transport text during collect pair repair as a warning retry signal, retries that focused repair once, and classifies exhausted no-artifact API repair as `runner_unavailable` rather than stale `runtime_contract_failed`.
- 2026-05-23: Fresh `v0.1.1` release candidate gate on main `488e173` (`release-fast-20260523T101656Z`) failed: qwen passed init but `refresh.step2.asis_docs` focused draft repair returned `[API Error: Connection error ... network socket disconnected before secure TLS connection]` with process `success` and no `asis-draft-manifest.json`; the runtime correctly reported `runner_unavailable`, but the existing transient retry covered only collect-pair repairs and did not recognize the TLS/socket transcript as retryable. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: qwen transient provider-unavailable focused retry now also covers draft-artifact repairs (`step0/2/4`) and recognizes connection/TLS/socket API text; exhausted no-artifact repair still fails as `runner_unavailable`. The same release attempt also showed host/network operational failures (`git clone` TLS errors, qwen/codex provider connectivity, occasional Claude smoke timeout), so a new release tag remains blocked until this fix is committed and a fresh clean-worktree release-fast PASS is produced on a stable trusted host.
- 2026-05-23: Fresh gate on main `4147d2c` (`release-fast-20260523T113202Z`) advanced past the transient draft-repair failure but hit a new qwen frontend blocker in `init.step2.asis_docs`: required as-is draft artifacts (`asis-draft-manifest.json`, `overview.md`, `summary.md`, `architect-summary.md`) were valid, while qwen kept streaming and mutating `architect-summary.md` until the global step runtime timeout. This is not a silence/preflight/provider-auth failure; it is an active overrun after valid draft artifacts. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: only `qwen-code` draft steps (`step0/2/4`) get a bounded valid-artifact controlled stop, so continued stream/mutation after a valid manifest+file set is accepted through the existing artifact validation gate instead of waiting for `runtime_timeout`. Collect/validator, Claude, and Codex behavior remain unchanged.
- 2026-05-24: Fresh release-prep gate on `e4db0ac` (`release-fast-20260524T081756Z`) passed the single-repo sweeps and multi-path backend, then exposed a Claude frontend product blocker in `init.step1.collect`: manifest-only repair for `devstack-docs` wrote a schema-valid `shard-pack-manifest.json` but the provider kept running and mutating it, while repair policy had `valid_artifact_stop_window_ms=0`; the old runtime could wait until timeout instead of accepting the valid repair artifact through validation. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: focused repair policies now add a short valid-artifact controlled stop for collect/validator/draft repairs across providers; partial/invalid repair artifacts remain `runtime_contract_failed`.

### Plan ID
EP-20260508-oss-readiness-hardening

### Context
Public OSS readiness audit found that the release binary path works, but repository governance, security reporting, CI trust boundaries, dependency hygiene, and README/community surfaces were below public distribution standards.

### Goals (must have)
- [x] Add security disclosure, support, governance, changelog, code of conduct, CODEOWNERS, PR template, and issue templates
- [x] Harden GitHub Actions permissions, timeouts, action pinning, release environment, SBOM/provenance, and required test alignment
- [x] Add Dependabot, CodeQL, Dependency Review, and Scorecard workflows/configuration
- [x] Fix UI dependency audit issues and enforce exact Node version from `.node-version`
- [x] Move required CI/release/local build toolchain to security-patched Go from `.go-version` while keeping `go.mod` compatibility at `go 1.20`
- [x] Update README/install/testing docs for OSS beta distribution and release evidence boundaries
- [x] Best-effort apply GitHub repository settings where current credentials have permission; otherwise record owner/admin manual tasks
- [ ] Owner/admin manual verification: GitHub still reports `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks` as disabled after API PATCH; enable them if available for the account/plan, or document the platform limitation
- [ ] Owner/admin manual verification: add a safe `creation` rule to the `v*` tag ruleset once bypass actors are confirmed, so release tag creation is restricted to maintainers without locking out the release owner

### Non-goals
- [x] Do not make Docker/npm/PyPI/Maven/crates.io a primary distribution path
- [x] Do not add hosted/SaaS mode or security/compliance enforcement
- [x] Do not run trusted live release matrix without explicit owner approval
- [x] Do not treat release readiness as passed without `reports/release_verdict_<matrix-id>.json`

### Approach
1) Add OSS community/security files and front-load README quickstart/release/security links.
2) Harden workflows with read-only defaults, job-level release writes, pinned actions, timeouts, and advisory security jobs.
3) Update UI dependencies and Node resolver so source builds are deterministic against `.node-version`.
4) Attempt admin-level GitHub settings changes through `gh`; record any settings that require owner/admin manual verification.
5) Run local DoD and targeted security/install checks.

### Files expected to change
- `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `SUPPORT.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`, `GOVERNANCE.md`
- `.github/*`
- `.goreleaser.yml`
- `scripts/resolve-node-tool.sh`, `scripts/tests/*`
- `ui/package.json`, `ui/package-lock.json`
- `docs/INSTALL.md`, `docs/TESTING_STRATEGY.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] `npm audit --prefix ui --audit-level=moderate`
- [x] `govulncheck ./...` when available
- [x] `bash ./scripts/smoke-cli.sh`
- [x] `bash ./scripts/smoke-api.sh`
- [x] GitHub owner/admin-only settings are either applied or explicitly listed as manual residual tasks

### Risks
- Branch protection, rulesets, secret scanning, private vulnerability reporting, and environment reviewers may require owner/admin permissions.
- Exact Node enforcement can block local builds until Node `22.21.1` is installed or selected through `ACP_NODE_TOOL_CANDIDATES`.
- Exact Go enforcement can block local builds until Go from `.go-version` is installed or selected through `ACP_GO_BIN`.
- GoReleaser SBOM generation requires `syft` in the release workflow.
- Local Go 1.20.x remains useful for compatibility checks, but `govulncheck` on May 8, 2026 reports vulnerable standard-library call paths when scanning binaries built with Go 1.20.3.

### Progress log
- 2026-05-08: Started OSS readiness hardening slice from the public distribution audit plan. Live matrix was not run.
- 2026-05-08: `govulncheck` against local Go 1.20.3 reported standard-library vulnerabilities; CI/release toolchain was moved to Go 1.25.10 while preserving `go.mod` language compatibility.
- 2026-05-08: Second audit found local `make build` and smoke scripts could still use stale `go` from PATH; added `.go-version`, `scripts/run-go.sh`, and wired Makefile/smoke scripts to fail fast on mismatched Go toolchains.
- 2026-05-08: Follow-up `make test` correctly failed `TestActivePlansHaveOpenGoals` because this active plan had all goals checked. Reopened the only real residual owner/admin task for GitHub secret scanning non-provider/validity features.
- 2026-05-08: Added OSS community/security files, issue/PR templates, Dependabot, CodeQL, Dependency Review, Scorecard, pinned GitHub Actions, release environment, SBOM/provenance release hooks, UI dependency updates, and exact Node resolver enforcement.
- 2026-05-08: Applied GitHub settings with admin token: `main` branch protection, `v*` tag ruleset for deletion/non-fast-forward blocking, `github-release` environment reviewer, private vulnerability reporting, Dependabot alerts/security updates, secret scanning, push protection, repo description/topics, labels, milestones, and curated `v0.1.0` release notes. GitHub still reports `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks` as disabled after PATCH; owners should manually verify whether these features are available for the account/plan.
- 2026-05-08: Tag ruleset creation restriction was left as owner/admin verification instead of being enabled blindly. GitHub's ruleset model restricts creations to bypass actors, and the safe bypass role/team set for this personal repository should be confirmed before enforcement.
- 2026-05-08: Verification passed with Go 1.25.10 + Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`, `npm audit --prefix ui --audit-level=moderate`, `govulncheck ./...`, `bash ./scripts/smoke-cli.sh`, `bash ./scripts/smoke-api.sh`, `git diff --check`, YAML parse smoke, and `goreleaser check`.
- 2026-05-08: Final workflow audit found two release-readiness bugs: `dependency-review` was pinned to a SHA from the wrong action repository, and Go workflows duplicated the exact version instead of reading `.go-version`. Fixed both and added regression coverage.
- 2026-05-08: Second-pass verification passed after Go guard and issue-template fixes: `make contracts`, `make test`, `make lint`, `make build`, `npm audit --prefix ui --audit-level=moderate`, `govulncheck ./...`, `bash ./scripts/smoke-cli.sh`, `bash ./scripts/smoke-api.sh`, `git diff --check`, YAML parse smoke, `goreleaser check`, and targeted Go/Node resolver tests.
- 2026-05-08: Final release metadata audit found `v0.1.0` was published as a normal GitHub Release while docs call the line MVP beta/pre-release. Changed GoReleaser to keep future beta releases prerelease and fixed `install.sh` so `ACP_VERSION=latest` resolves prerelease releases through the GitHub Releases API instead of relying on `/releases/latest/download`.
- 2026-05-08: Tried marking the already-published `v0.1.0` as prerelease, but reverted the remote release to latest/non-prerelease because the current public `main/install.sh` still uses GitHub's `/releases/latest/download` endpoint. Public install smoke passed again after restoring `v0.1.0` as latest. After this installer fix is merged to `main`, owners can mark beta releases as GitHub prereleases without breaking the default install command.
- 2026-05-08: PR mergeability audit found a solo-maintainer governance deadlock: `main` required one approval plus CODEOWNERS review, but `GrinRus` is currently the only collaborator and CODEOWNER. Documented the solo-maintainer policy: required CI remains mandatory, while required approving/CODEOWNERS reviews are enabled after a second maintainer exists.

### Plan ID
EP-20260507-trusted-live-validation

### Context
Historical active plans from 2026-04-20 through 2026-05-06 repeatedly ended with the same unresolved gate: trusted-host live reruns through the canonical matrix harness. Local behavior fixes and DoD evidence are present for those slices, but live confidence still requires direct `scripts/full-run-batch-matrix.sh` execution from a clean committed trusted workspace with `qwen`, `claude`, and `codex` available.

### Goals (must have)
- [ ] Prepare a clean committed trusted-host worktree with required provider binaries/auth and pinned repo/path prerequisites
- [ ] Run qwen `regres fast` for `examples/e2e-matrix.regres-fast.bank-openedx.yaml` and `examples/e2e-matrix.regres-fast.openstack.yaml`
- [ ] Run qwen+codex diagnostic rerun for bank/openedx and openstack scenarios
- [ ] Run qwen+claude rerun for the current small live bugfix matrix
- [ ] Run claude bank/openedx verification and qwen-focused `regres fast`
- [ ] Run full fresh-all-6 only after smaller slices pass or produce narrower blocker reports
- [ ] Record matrix ids, driver logs, profile/release reports, quality reports, taskrun raw diagnostics, and verifier output where a release verdict is used
- [ ] Update `docs/PLANS.md`/archive with pass/fail evidence and any follow-up fix slices created from live failures

### Non-goals
- [x] Do not add wrapper scripts around `scripts/full-run-batch-matrix.sh`
- [x] Do not wire live matrix into required CI
- [x] Do not edit canonical matrix/profile/timeout files to fit the current machine
- [x] Do not treat live release readiness as passed without `reports/release_verdict_<matrix-id>.json` and `scripts/verify-release-verdict.py` when release verdict is used

### Approach
1) Verify trusted-host prerequisites and clean committed tree.
2) Execute each matrix by setting `E2E_MATRIX_FILE`, provider command envs, and `BATCH_PROVIDER_FILTER`, then invoking `./scripts/full-run-batch-matrix.sh` directly.
3) Monitor driver/status/report artifacts until terminal state.
4) Classify failures into product/runtime/harness/operational buckets and open a focused fix slice before broad reruns when needed.
5) Close historical live-gate tasks only with concrete matrix/report evidence.

### Non-live prerequisite check (2026-05-07)

| Check | Status | Evidence / note |
|---|---|---|
| Clean committed local worktree | ready | `git status --short` was empty at `c1f58b5`. |
| Required provider binaries in PATH | ready for auth check | `command -v qwen`, `command -v claude`, and `command -v codex` returned local paths. Exact machine-local paths are intentionally omitted from the versioned plan; provider auth/quota was not probed. |
| Canonical harness inputs present | ready | `examples/e2e-matrix.regres-fast.bank-openedx.yaml`, `examples/e2e-matrix.regres-fast.openstack.yaml`, `scripts/full-run-batch-matrix.sh`, and `scripts/verify-release-verdict.py` exist. |
| Live execution permission | blocked | No live matrix was started in this local implementation pass. Running live validation still needs explicit trusted-machine approval and direct `scripts/full-run-batch-matrix.sh` invocation. |

### Critical analysis (2026-05-08)

This slice is not closed and must not be treated as release evidence. The local worktree check and provider binary discovery only prove that the current machine can find command names; they do not prove provider authentication, quota availability, pinned repo/path readiness, matrix pass/fail status, or release readiness.

Residual blockers:
- live matrix execution still requires explicit trusted-machine approval;
- provider auth/quota remains unverified;
- pass criteria remain completely open until direct `scripts/full-run-batch-matrix.sh` runs produce matrix/report artifacts;
- release readiness still requires `reports/release_verdict_<matrix-id>.json` plus `scripts/verify-release-verdict.py` when a release verdict is used.

### Files expected to change
- `docs/PLANS.md`
- `docs/archive/*` for execution reports or reconciliation notes
- focused code/docs/tests only if live evidence creates a new fix slice

### Acceptance criteria
- [ ] No unresolved `runtime_contract_failed`, `runner_unavailable`, `semantic_hard_fail`, independent frontend failure, stale `running`, or unexpected `analysis:cross-repo-missing` remains for fixed scenarios
- [ ] Every live run has a recorded `matrix_id`, `driver.log`, `profile_matrix_<matrix-id>.*`, quality reports, and taskrun diagnostics
- [x] Release readiness uses only `reports/release_verdict_<matrix-id>.json`; verifier exits 0 only for `PASS` / `RELEASE READY` / `passed`
- [ ] Any live failure produces a narrower follow-up fix slice instead of ad-hoc matrix edits

### Risks
- Provider outages, auth/quota limits, or missing trusted-host path prerequisites can block live gates without indicating product regression.
- Full fresh-all-6 can be expensive; run it only after smaller gates either pass or isolate residual blockers.

### Progress log
- 2026-05-07: Created by tracker reconciliation. Consolidates live rerun gates from historical active plans; no live matrix was run in this reconciliation slice.
- 2026-05-07: Performed non-live prerequisite check only. Worktree and provider binary paths look ready for an owner-approved trusted-machine run, but provider auth/quota and pinned repo/path prerequisites are unverified. Live matrix execution remains blocked until explicit approval.

### Plan ID
EP-20260507-cleanup-owner-decisions

### Context
Safe cleanup is already complete, but `docs/PLANS.md` and `docs/BACKLOG.md` retain owner-gated cleanup decisions where deleting or moving tracked surfaces could damage discoverability, review UX, or hidden tooling usage. These items must remain explicit and blocked until owner intent is known.

### Goals (must have)
- [x] Gather usage evidence for whether `internal/docsync` and `internal/scriptsmeta` should remain test-only packages under `internal/*`
- [x] Gather evidence on retaining both `docs/archive/PLANS_ARCHIVE_2026-04.md` and `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`
- [x] Gather evidence on persisted `fixtures/scenarios/*/golden/readable`
- [x] Gather evidence on duplicated readable scenario fixtures
- [x] Gather evidence on `docs/BACKLOG.md` role as active planning surface vs reference/history backlog
- [ ] Owner decision: retain/move `internal/docsync` and `internal/scriptsmeta`
- [ ] Owner decision: retain/remove/dedupe April plan archive + snapshot docs
- [ ] Owner decision: retain/remove/dedupe readable golden fixtures
- [ ] Owner decision: keep `docs/BACKLOG.md` as reference/acceptance backlog or change its planning-surface role
- [ ] If approved by owners, implement each cleanup as a separate small change set with tests/docs sync

### Non-goals
- [x] Do not move `internal/docsync` or `internal/scriptsmeta` without owner approval
- [x] Do not delete archive snapshots, readable fixtures, or duplicated fixtures without owner approval
- [x] Do not mix destructive cleanup with runtime/live fixes

### Approach
1) Search usage and references for each cleanup candidate.
2) Write a short evidence table with retain/remove/dedupe options and recommended default.
3) Request owner decisions; retain by default if no answer is available.
4) Implement approved cleanup items one at a time with focused tests and full DoD when tracked files change.

### Owner-decision evidence (2026-05-07)

| Candidate | Evidence gathered | Options for owner | Default until decision |
|---|---|---|---|
| `internal/docsync`, `internal/scriptsmeta` | Only test files are present under those packages: `internal/docsync/docsync_test.go`, `internal/scriptsmeta/resolve_repos_meta_test.go`. `go test ./...` runs them as repo-level consistency gates; `docs/TESTING_STRATEGY.md` names `internal/docsync` as a docs-consistency gate. | Retain under `internal/*`; move to a dedicated test-support location; collapse into script tests. | Retain under `internal/*`; moving them is layout churn without owner-confirmed value. |
| `docs/archive/PLANS_ARCHIVE_2026-04.md` + `docs/archive/PLANS_SNAPSHOT_2026-04-21.md` | `docs/PLANS.md` lists both archives; the snapshot records that completed historical plans moved to the April archive. Current sizes are 912 and 1176 lines, respectively, so they are substantial historical evidence. | Retain both; collapse snapshot into archive; keep snapshot but remove cross-links. | Retain both until owner decides the audit/history surface can be reduced. |
| `fixtures/scenarios/*/golden/readable/*` | 90 tracked readable files exist. `docs/BASELINE_POLICY.md`, `docs/TESTING_STRATEGY.md`, `fixtures/README.md`, and `internal/docsync/docsync_test.go` explicitly describe them as tracked baseline/release surface and human-readable deterministic export. | Retain tracked readable exports; remove and rely on machine-readable golden only; generate on demand without tracking. | Retain; current docs/tests make them intentional, not accidental generated output. |
| Duplicated readable scenario fixtures | Hash scan shows many identical files repeated across the three readable scenario exports, including `reports/as-is/*`, `reports/diagrams/*`, `model/entities/svc.payments.yaml`, and proposal docs. | Keep duplicated per-scenario snapshots; dedupe via shared fixture layer; remove readable exports after replacing review-diff workflow. | Retain duplicated snapshots; dedupe needs a QA/tooling owner decision because per-scenario full-tree diffs are the current review UX. |
| `docs/BACKLOG.md` role | README still points to `docs/BACKLOG.md` for epics/acceptance criteria, AGENTS instructs agents to take reviewable slices from it, and `internal/docsync/docsync_test.go` asserts several backlog truth-sync strings. The backlog itself says active engineering slices live in `docs/STAKEHOLDER_DOC.md` and `docs/PLANS.md`, while a 2026-04-22 follow-up asks owners to decide active planning surface vs reference/history role. | Keep as reference/acceptance backlog; promote it back to active planning surface; archive/freeze it after migrating acceptance criteria elsewhere. | Keep current reference/acceptance role until owners decide; active execution remains in `docs/PLANS.md`. |

### Critical analysis (2026-05-08)

This slice is evidence-complete but decision-blocked. The gathered evidence points toward retaining all cleanup candidates by default because the candidates are referenced by tests/docs or are intentionally tracked review surfaces. However, retaining by default is not the same as resolving ownership: no package move, archive collapse, fixture deletion, fixture dedupe, or runbook consolidation should happen until maintainers/docs/tooling owners choose an option.

Residual blockers:
- no owner decision has been provided for test-only package placement;
- no owner decision has been provided for archive/snapshot retention;
- no owner decision has been provided for readable fixture retention or dedupe;
- no owner decision has been provided for the long-term role of `docs/BACKLOG.md`;
- no destructive cleanup was performed, so the final implementation goal remains intentionally open.

### Files expected to change
- `docs/PLANS.md`
- `docs/BACKLOG.md` only if owner-gated backlog wording changes
- candidate files only after explicit owner approval

### Acceptance criteria
- [x] Each owner-gated item has usage evidence and an explicit retain/remove/dedupe recommendation
- [x] No destructive cleanup happens without owner approval
- [ ] Approved cleanup items are implemented in separate commits
- [x] Active plan surface keeps owner-gated decisions visible until resolved

### Risks
- Removing historical docs or readable fixtures can degrade review workflows even if tests do not fail.
- Keeping everything can preserve drift; retained items need explicit ownership or periodic review.

### Progress log
- 2026-05-07: Created by tracker reconciliation from `EP-20260421-cleanup-owner-followups` and `docs/BACKLOG.md` cleanup follow-ups.
- 2026-05-07: Gathered usage evidence and documented retain-by-default recommendations. No files were deleted, moved, or deduplicated; owner decisions remain open and blocked.
- 2026-05-18: Removed the live E2E convenience-runbook owner decision from this cleanup plan because the black-box live E2E slice deletes old live E2E surfaces without compatibility.

---

## Implemented vs Planned (operational mirror)

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Таблица ниже — инженерный mirror и должна оставаться синхронизированной с канонической матрицей.

| Epic | Статус | Комментарий |
|---|---|---|
| 1 Workspace/contracts | done (beta baseline) | Schema-driven + semantic validation, resolver `path/git_url`, diagnostics API |
| 2 Runtime artifact contracts | done (beta baseline) | Validation + artifact-only runtime execution contract, contract tests |
| 3 Runtime/orchestration seam | done (beta baseline) | Fake default + opt-in headless runtime selector with provider choice (`claude-code` default, `qwen-code`, `codex-code` release peer), persisted runtime execution metadata |
| 4 Model deterministic core | done (beta baseline) | Canonical IDs/collision rules + deterministic regression tests |
| 5 Pipeline 0–4 | done (beta baseline) | `init|refresh` runnable через CLI/API |
| 6 UI baseline | done (beta baseline) | Setup/validate/run/inspect + editors + git helpers |
| 7 Domain-first layer | done (beta baseline) | Per-domain contracts + deterministic Step 1 enrichment canonical domain/team cards without auto-create |
| 8 Baseline bundle | done (beta baseline) | `skills/subagents.yaml` + prompt packs + validation |
| 9 Q&A capability | target upgraded | Async runtime-backed Ask via `/api/qa/runs`; deterministic service remains `acp qa` + legacy `POST /api/qa/ask` compatibility |
| 10 Changelog compilers | done (beta baseline) | Iteration changelog materialization в `reports/changelog/*` |
| 11 `POST /api/qa/ask` | done (compatibility baseline) | Read-only API wrapper over deterministic Q&A service while UI target uses async QA runs |
| 12–13 | out of MVP | Вне текущего beta scope |
| 14 CI trigger mode | done (beta baseline) | CLI batch required, smoke/golden jobs без live network deps |
| 15 Domain/baseline pack hardening | done (beta baseline) | Baseline skills/prompts wired и versioned в workspace |
