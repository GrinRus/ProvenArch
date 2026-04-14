# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон и текущие активные планы.

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

### Plan ID
EP-20260414-batch-parallel-shards

### Context
`scripts/full-run-batch-5x2.sh` выполняет backend/frontend циклы строго последовательно (`2 providers x 5 runs`), что делает локальный trusted-machine re-audit слишком долгим. Нужен безопасный split-run режим для параллельного запуска независимых shard-процессов без изменения default `5x2` контракта.

### Goals (must have)
- [x] Добавить shard selector по provider (`BATCH_PROVIDER_FILTER`) и run index (`BATCH_RUN_SELECTION`)
- [x] Добавить `BATCH_SKIP_PRECHECK` для secondary shard'ов
- [x] Добавить frontend execution mode (`BATCH_FRONTEND_MODE=auto|always|never`) с auto-skip без `run1`
- [x] Сохранить backward-compatible default поведение (`all providers`, `runs 1..5`, precheck enabled)
- [x] Обновить integration tests и документацию runbook/testing strategy

### Non-goals
- [x] Изменение quality rubric/report contracts (`scripts/e2e_batch_report.py`)
- [x] Изменение required CI gate policy (live smoke остаётся optional)
- [x] Изменение набора runtime providers beyond MVP

### Approach
1) Расширить `full-run-batch-5x2.sh` валидируемыми shard/env controls и перевести все loops на resolved selections.
2) Сделать deterministic frontend skip semantics для shard'ов без `run1` (без ложного `frontend_failed`).
3) Добавить regression tests для precheck-classification subset и frontend auto-skip semantics.
4) Синхронизировать `README`, `LOCAL_FULL_RUN_AI_ADVENT`, `TESTING_STRATEGY`.

### Files expected to change
- `scripts/full-run-batch-5x2.sh`
- `scripts/tests/test_full_run_stability.py`
- `README.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы/контракты не изменены
- [x] Документация обновлена

### Risks
- Неправильный shard selection может создавать неполные локальные отчёты; mitigated explicit env docs + явный `skipped` статус frontend smoke.
- Параллельные shard'ы с одинаковым `BATCH_ID` будут конфликтовать по output paths; mitigated runbook рекомендацией задавать уникальный `BATCH_ID`.

### Progress log
- 2026-04-14: Добавлены `BATCH_PROVIDER_FILTER`, `BATCH_RUN_SELECTION`, `BATCH_SKIP_PRECHECK`, `BATCH_FRONTEND_MODE` в `full-run-batch-5x2.sh`; loops переведены на resolved shard sets.
- 2026-04-14: Добавлены integration tests для shard precheck classification и frontend auto-skip без `run1`; обновлены README/runbook/testing docs.
- 2026-04-14: Добавлены негативные regression tests для invalid provider/run shard selection, `BATCH_SKIP_PRECHECK` bypass и `BATCH_FRONTEND_MODE=never`; в runbooks явно зафиксированы `unique BATCH_ID` и рекомендация single-precheck shard.

### Plan ID
EP-20260413-backend-repo-selection-hardening

### Context
Нужно укрепить backend-focused selection в multi-repo/monorepo: синхронизировать `repo_scope` resolver между runtime и enrich, добавить explicit frontend/backend policy, убрать `git_url` cache collisions и повысить диагностируемость.

### Goals (must have)
- [x] Единый resolver `repo_scope` для step1.collect + enrich
- [x] Проверка mismatch filename `<domain-id>.md` vs `- id:` с high-priority question, без изменения filename-based runtime id
- [x] Контракт `workspace.yaml`: `repos[].analysis.role` + `runtime.profile.execution.repo_selection`
- [x] Effective selection policy `all|backend_only` + skip domain tasks при excluded scope
- [x] Применение effective scopes к runtime шагам (step1/step2/step3/step4) и shard planning
- [x] `git_url` cache key `slug+hash(source)` + legacy fallback warning
- [x] API/UI surfaces: execution `repo_selection`, validate decisions (`effective_role`, `included/excluded`, `reason`)
- [x] Артефакт `reports/taskruns/<run_id>-repo-selection-summary.json`
- [x] Unit/integration/regression tests + docs/schema/examples sync

### Non-goals
- [x] Изменение default поведения для существующих пользователей (`repo_selection` default остаётся `all`)
- [x] Добавление новых runtime providers beyond MVP

### Approach
1) Обновить workspace/runtime contracts и selection evaluator.
2) Привязать selection к orchestrator domain/runtime execution path.
3) Укрепить resolver/cache diagnostics и добавить repo-selection summary artifact.
4) Обновить API/UI surfaces и тесты.
5) Синхронизировать docs/spec/examples.

### Files expected to change
- `internal/workspace/*`
- `internal/runtime/*`
- `internal/orchestrator/*`
- `internal/api/*`
- `ui/src/*`
- `schemas/workspace.schema.json`
- `docs/*`, `README.md`, `examples/workspace.example.yaml`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена

### Risks
- `backend_only` при пустом selected scope может unintentionally скрыть анализ; mitigated explicit questions/warnings + summary artifact.
- Legacy git cache fallback может задержать миграцию на hashed key; mitigated warning diagnostics.

### Progress log
- 2026-04-13: Реализованы repo-selection policy hardening, unified domain repo_scope resolver, git_url cache-key migration fallback, API/UI wiring, summary artifact и regression suite.

### Plan ID
EP-20260413-sharding-execution-profile

### Context
Нужно масштабировать runtime для больших монореп/мульти-репо: добавить управляемое shard-дробление с parallel execution внутри одного run, сохранить global single-run lock, и перенести runtime-конфиг в новый `runtime.profile` контракт.

### Goals (must have)
- [x] Ввести новый `workspace.yaml` контракт: `runtime.profile.timeouts`, `runtime.profile.execution`, `repos[].analysis.include/exclude`
- [x] Добавить execution resolution (`defaults/workspace/env/CLI`) и CLI overrides (`--execution-strategy`, `--max-parallel-tasks`, `--failure-policy`)
- [x] Реализовать shard planner + scheduler (sequential/parallel, best_effort/fail_fast) для runtime step1/step3
- [x] Добавить per-shard taskruns + shard summary artifact + deterministic apply order
- [x] Зафиксировать partial failure semantics (`run_partial_failed`) с агрегированными diagnostics
- [x] Добавить API `GET/PUT /api/runtime/execution`
- [x] Добавить UI panel для runtime execution profile (load/save/reset)
- [x] Синхронизировать docs/examples/fixtures/tests и прогнать DoD (`make contracts/test/lint/build`)

### Non-goals
- [x] Изменение global queue policy (single active run + debounce)
- [x] Добавление новых runtime providers beyond MVP (`claude-code`, `qwen-code`)
- [x] Hosted/security-compliance расширения вне MVP

### Approach
1) Обновить schema/manifest/runtime resolution контракты.
2) Интегрировать shard planning/scheduling в orchestrator runtime steps.
3) Добавить API/UI surfaces для execution profile.
4) Обновить regression suite + docs/spec/examples/fixtures, зафиксировать deterministic golden.

### Files expected to change
- `schemas/workspace.schema.json`
- `internal/workspace/*`
- `internal/runtime/*`
- `internal/orchestrator/*`
- `internal/api/*`
- `cmd/acp/*`
- `ui/src/*`
- `docs/*`, `examples/*`, `fixtures/*`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена

### Risks
- Breaking update `workspace.yaml` (`runtime.timeouts` -> `runtime.profile.timeouts`) без compatibility-layer.
- Увеличение числа taskrun artifacts из-за sharding требует аккуратной фильтрации в тестах/интеграциях.

### Progress log
- 2026-04-13: Реализованы schema/manifest/runtime profile changes + execution resolver + CLI overrides.
- 2026-04-13: Добавлены shard planner/scheduler, best-effort partial semantics и shard artifacts.
- 2026-04-13: Добавлены API/UI execution profile surfaces, обновлены docs/examples/fixtures, пройдены `make contracts`, `make test`, `make lint`, `make build`.
- 2026-04-13: Дозакрыт gap-аудит: добавлены unit/integration regression tests для planner filters/fallback, sequential-vs-parallel scheduler, deterministic apply order, `run_partial_failed`, synthetic large-monorepo + multi-repo shard scenarios; в shard-plan artifacts добавлен deterministic `semantic_graph` dump.
- 2026-04-13: Дозакрыт контрактный и тестовый хвост: добавлен backward-compatible alias `meta.repo_scope` (в schema/runtime prompts/diagnostics), добавлены CLI/env precedence tests для execution profile и refresh step1/step3 sharding regression; повторно пройден DoD (`make contracts/test/lint/build`).
- 2026-04-13: Добавлен отдельный fail-fast regression (`step` останавливается на первой shard error, без `run_partial_failed`), повторно перепроверены `go test ./...`, `make contracts`, `make lint`, `make build`.

### Plan ID
EP-20260413-postfix-matrix-runtime-stability

### Context
После post-fix matrix остались падения в direct runtime (`qwen` invocation/`claude` parse), а также нечёткая классификация quality-vs-infra в batch отчётах. Дополнительно `cancel-refresh` frontend e2e флакал из-за transient banner-проверки.

### Goals (must have)
- [x] Перевести default `qwen` invocation на `--prompt` при `--include-directories`
- [x] Усилить extractor/runtime diagnostics: явный `parse_stage`, точные причины envelope errors, richer unavailable message при пустом stderr
- [x] Расширить `claude` retry для malformed envelope `result`
- [x] Добавить отдельную batch/matrix классификацию `quality_gates_failed` (без смешивания с `infra_incomplete_cycle`)
- [x] Стабилизировать frontend `cancel-refresh` e2e без зависимости от transient текста

### Non-goals
- [x] Изменение CLI/API/schema contracts
- [x] Изменение набора runtime providers MVP

### Approach
1) Обновить runtime invocation/parsing (`qwencode`, `claudecode`, `taskresultextractor`, `runnerdiag`).
2) Обновить batch/matrix quality classification (`full-run-batch-5x2`, `e2e_batch_report`, `full-run-batch-matrix`).
3) Упростить `cancel-refresh` frontend live assertion до устойчивых run-state checks.
4) Обновить docs + покрыть изменённое поведение unit/integration regression tests.

### Files expected to change
- `internal/runtime/{qwencode,claudecode,taskresultextractor,runnerdiag}/*`
- `scripts/full-run-batch-5x2.sh`
- `scripts/e2e_batch_report.py`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/test_e2e_batch_report.py`
- `ui/e2e/live-flow.spec.ts`
- `README.md`, `docs/LOCAL_FULL_RUN_AI_ADVENT.md`, `docs/TESTING_STRATEGY.md`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы/контракты не изменены
- [x] Документация синхронизирована

### Risks
- Разные версии внешних CLI (`qwen`, `claude`) могут давать дополнительные carrier-форматы output; для них нужен регулярный regression run на trusted machine.

### Progress log
- 2026-04-13: Реализованы runtime/batch/frontend фиксы и добавлены regression tests для `--prompt`, parse-stage/schema path, quality-vs-infra classification и cancel-refresh stability.
- 2026-04-14: Дозакрыт integration gap `runtime_parse > infra_incomplete_cycle` (batch classifier priority) и выполнен повторный canary multi (`multi-path`/`multi-git_url`, `qwen`/`claude`) + frontend `cancel-refresh` single `2/2`.

### Plan ID
EP-20260413-trash-cleanup-safe-pass

### Context
Нужен безопасный cleanup репозитория: удалить только доказанный мусор без риска скрытых регрессий и без изменения API/контрактов.

### Goals (must have)
- [x] Удалить неиспользуемые импорты в Python scripts/tests
- [x] Удалить unreferenced исторические review-документы
- [x] Зафиксировать, что policy-tracked generated surface и legacy compatibility не удаляются в этом slice
- [x] Подготовить owner follow-up для спорных cleanup-пунктов

### Non-goals
- [x] Изменение public API/CLI wire-contracts
- [x] Изменение `schemas/*` и `docs/spec/*` контрактов
- [x] Рефакторинг/дедупликация scenario readable fixtures

### Approach
1) Применить только high-confidence удаления (`unused imports`, unreferenced historical docs).
2) Прогнать quality gates (`ruff`, `go test`, `go vet`, UI tests/typecheck, `shellcheck`).
3) Зафиксировать cleanup changelog и owner follow-up в docs.

### Files expected to change
- `scripts/e2e_batch_report.py`
- `scripts/tests/test_full_run_stability.py`
- `docs/reviews/*` (remove)
- `docs/BACKLOG.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] `ruff check --select F401,F841 scripts` не показывает новых нарушений
- [x] `go test ./...` зелёный
- [x] `go vet ./...` зелёный
- [x] `npm --prefix ui run test -- --run` зелёный
- [x] `npm --prefix ui run typecheck` зелёный
- [x] `shellcheck scripts/*.sh` без block-level проблем

### Risks
- Удаление исторических docs может убрать контекст прошлых решений; mitigated через запись cleanup/follow-up в operational docs.
- Часть кандидатов на cleanup может использоваться неявно; такие пункты оставлены в owner follow-up и не удаляются.

### Progress log
- 2026-04-13: Удалены `unused import` в `scripts/e2e_batch_report.py` (`math`) и `scripts/tests/test_full_run_stability.py` (`json`).
- 2026-04-13: Удалены unreferenced historical docs: `docs/reviews/CONSISTENCY_AUDIT_2026-03-30.md`, `docs/reviews/DOCS_HISTORY_AND_GAPS_2026-03-30.md`.
- 2026-04-13: Подтверждён cleanup boundary: `internal/api/ui_dist/*`, `fixtures/scenarios/*/golden/readable/*`, legacy full-run inputs и core CLI/runtime/schema surfaces не удаляются в этом slice.
- 2026-04-13: Создан owner follow-up для спорных cleanup-кандидатов в `docs/BACKLOG.md`.

---

### Plan ID
EP-20260411-e2e-stability-hardening

### Context
Нужно устранить ложные pass/fail в локальном e2e контуре: процессные сигналы и неполные циклы не должны завершаться как `passed`, а runtime parse-fail должен оставлять воспроизводимые raw diagnostics. Дополнительно нужно разнести failure classes в batch/matrix отчётах.

### Goals (must have)
- [x] Укрепить `scripts/full-run-ai-advent.sh` (signal traps, truthful summary, completion invariants)
- [x] Укрепить `scripts/full-run-batch-5x2.sh` post-run validation и классификацию failure classes
- [x] Расширить `scripts/full-run-batch-matrix.sh` агрегированными колонками failure classes
- [x] Добавить raw parse-fail artifacts в runtime runners (`claude-code`, `qwen-code`)
- [x] Усилить extractor mixed/noisy output парсинг без изменения schema contracts
- [x] Обновить тесты и документацию

### Non-goals
- [x] Изменение CLI/API wire-contracts
- [x] Изменение `schemas/taskresult.schema.json`
- [x] Добавление новых runtime providers

### Approach
1) Исправить truthfulness и completion-валидацию single full-run.
2) Добавить batch-level post-run validation и отдельные классы сбоев.
3) Усилить runtime parse diagnostics (raw outputs + checksum) и extractor.
4) Синхронизировать docs и прогнать DoD проверки.

### Files expected to change
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch-5x2.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/e2e_batch_report.py`
- `internal/runtime/{claudecode,qwencode,taskresultextractor}/*`
- `internal/runtime/runnerdiag/*`
- `scripts/tests/test_e2e_batch_report.py`
- `README.md`, `docs/LOCAL_FULL_RUN_AI_ADVENT.md`, `docs/TESTING_STRATEGY.md`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена

### Risks
- Дополнительная строгость completion-инвариантов может выявить ранее скрытые инфраструктурные проблемы в локальных окружениях.
- Расширенный extractor может начать принимать ранее невалидные шумные carrier-форматы; это требует regression coverage.

### Progress log
- 2026-04-11: Реализованы signal traps + completion invariants + truthful summary в `full-run-ai-advent.sh`.
- 2026-04-11: Добавлены post-run validation и failure-class aggregation в `full-run-batch-5x2.sh`, расширен `profile_matrix` в `full-run-batch-matrix.sh`.
- 2026-04-11: Добавлен `internal/runtime/runnerdiag` и raw parse-fail artifacts в claude/qwen runners.
- 2026-04-11: Усилен `taskresultextractor` для noisy NDJSON/mixed output; обновлены unit tests и docs.

---

### Plan ID
EP-20260403-acp-mvp-beta-foundation

### Context
Нужно перевести ACP из bootstrap skeleton в рабочий MVP beta foundation: валидируемый workspace, применяемый TaskResult, рабочие `init/refresh` run paths для CLI/API, deterministic materialization артефактов в workspace и required локальные проверки без live network dependencies.

### Goals (must have)
- [x] Обновить AGENTS baseline правила (source-of-truth priority, DoD, contract-sync rule)
- [x] Реализовать workspace manifest loading + semantic validation
- [x] Реализовать TaskResult validation + normalization legacy forms
- [x] Реализовать рабочий orchestrator run path для `init`/`refresh`
- [x] Реализовать API baseline endpoints `/api/health`, `/api/workspace/validate`, `/api/artifacts`, `/api/pipeline/*`
- [x] Реализовать deterministic materialization `model/`, `reports/`, `proposals/`, `changelog`, `taskruns`
- [x] Добавить/обновить тесты и прогнать `make contracts test lint build`

### Non-goals
- [x] Hosted/multi-tenant режим
- [x] Security/compliance enforcement
- [x] Wave 1+ интеграции (autodocs, Jira manager agents)
- [x] Не включать `POST /api/qa/ask` в required release surface

### Approach
1) Сделать документарный baseline hardening и зафиксировать этот план.
2) Реализовать contracts/workspace/runtime/orchestrator минимально, но полностью исполнимыми.
3) Реализовать API/CLI execution flow и deterministic artifact materialization.
4) Закрыть тестовый слой (contract + semantic + run-path smoke) и синхронизировать документацию.

### Files expected to change
- `AGENTS.md`
- `docs/PLANS.md`
- `cmd/acp/*`
- `internal/workspace/*`
- `internal/orchestrator/*`
- `internal/runtime/claudecode/*`
- `internal/model/*`
- `internal/reports/*`
- `internal/api/*`
- `fixtures/*` (при необходимости)
- `README.md`, `docs/ARCHITECTURE.md`, `docs/spec/*` (при изменении поведения/контракта)

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена

### Risks
- Большой объём изменений для одного slice может увеличить риск регрессий.
- Нужна аккуратная балансировка между "рабочий MVP foundation" и "не раздувать scope".

### Progress log
- 2026-04-03: План создан. Начат Phase A (baseline hardening).
- 2026-04-03: Реализованы Phase A/B/C/D/E foundation: workspace+TaskResult validation/normalization, runnable orchestrator init|refresh, API `/api/*`, model/reports/proposals/changelog materialization, fake/recorded runtime harness.
- 2026-04-03: Добавлены/обновлены тесты и пройдены `make contracts`, `make test`, `make lint`, `make build`.
- 2026-04-03: Начат gap-closing beta pass: schema-driven runtime validation, workspace diagnostics, repo source resolver (`path` + `git_url` cache/fetch), run coordinator/debounce, deterministic artifact hardening, embedded UI/API + editor/git-helper endpoints.
- 2026-04-03: Добавлены scenario fixtures, deterministic scenario integration tests, smoke scripts и CI workflows (`golden`, `smoke-cli`, `smoke-api`, `ui-smoke`, optional `live-runner-smoke`).
- 2026-04-07: `trash-cleanup` slice: удален dead-code в `scripts/full-run-ai-advent.sh` (`QUALITY_GATES_NOTE`), очищены hardcoded defaults для full-run, удалён placeholder `live-runner-smoke` workflow и обновлены централизованные инструкции для scenario run.

---

### Plan ID
EP-20260403-acp-beta-gap-closing

### Context
Закрыть выявленные после аудита расхождения между фактической реализацией и документированными контрактами, а также усилить deterministic/smoke контур до beta-ready required CI без live network dependencies.

### Goals (must have)
- [x] Привести API docs к фактическому wire-contract (`error.code/error.message`, diagnostics shape, trigger/not_supported cases)
- [x] Усилить API тесты на негативные кейсы (`invalid_request_body`, `trigger_unsupported`, `not_supported`)
- [x] Усилить smoke/API и scenario fixtures (dynamic port, timeout fail, fixture contract/semantic gate)
- [x] Зафиксировать golden deterministic baseline через snapshot hashes
- [x] Расширить UI smoke до реального flow `open -> validate -> run -> inspect` с mocked API
- [x] Обновить README/ARCHITECTURE/TESTING_STRATEGY/PLANS до согласованного состояния

### Non-goals
- [x] Не включать `/api/qa/ask` в beta required surface
- [x] Не добавлять новые runtime кроме `claude-code`
- [x] Не вводить hosted/multi-tenant control plane

### Approach
1) Закрыть compile/test/lint regressions в изменённых slice.
2) Зафиксировать deterministic scenario baseline и fixture semantics.
3) Синхронизировать docs/spec/API с реальным поведением серверных handler-ов.
4) Прогнать release gates (`contracts/test/lint/build/smoke/race`).

### Files expected to change
- `internal/api/*`
- `internal/orchestrator/scenario_test.go`
- `fixtures/scenarios/*`
- `scripts/smoke-api.sh`
- `ui/src/App.test.tsx`
- `docs/spec/API_SPEC.md`
- `internal/api/README.md`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] `make contracts` зелёный
- [x] `make test` зелёный
- [x] `make lint` зелёный
- [x] `make build` зелёный
- [x] `bash ./scripts/smoke-cli.sh` зелёный
- [x] `bash ./scripts/smoke-api.sh` зелёный
- [x] `go test -race ./...` зелёный

### Risks
- Документация может снова устаревать без дисциплины синхронизации при следующих slice.
- Run-specific артефакты могут ошибочно попадать в deterministic compare при расширении pipeline.

### Progress log
- 2026-04-03: Добавлен unified error envelope для `POST /api/workspace/validate` и API tests на negative cases.
- 2026-04-03: Усилен `scripts/smoke-api.sh` (dynamic port + timeout fail), исправлен `copyScenarioRoot`, добавлен fixture semantic gate.
- 2026-04-03: Зафиксирован scenario golden baseline через `golden/snapshot.sha256` и сравнение в тестах.
- 2026-04-03: Расширен UI smoke test до полного mocked flow `open -> validate -> run -> inspect`.
- 2026-04-03: Пройдены `make contracts`, `make test`, `make lint`, `make build`, `smoke-cli`, `smoke-api`, `go test -race ./...`.
- 2026-04-04: Добавлен process-scoped runtime selector (`--runtime fake|headless`) для `acp run|serve`, actionable runner diagnostics (`runner_unavailable`, `runner_parse_failed`) и `error_code` в run status API.
- 2026-04-04: Domain fan-out переведён на per-domain execution contracts (`*.task-envelope.json`) с deterministic tests и unresolved-domain questions.
- 2026-04-04: Усилен internal QA indexing (charter/cards + model + reports + docs/imports), добавлены ranking/citations tests, добавлены GitLab trigger templates и docs sync pass.
- 2026-04-04: Устранён async coordinator gap: rejected start вне debounce window больше не создаёт orphan `queued` run-record.
- 2026-04-04: API/docs truth-sync: `runner_parse_failed` зафиксирован как run-level `error_code` после `202` async start, а не как start-time HTTP ошибка.
- 2026-04-04: Закрыт residual gap: `runner_parse_failed` исключён из start-time API mapping, Step 1 domain unresolved questions materialize-ятся в coverage сразу, async polling в API/orchestrator тестах унифицирован через общий wait helper.
- 2026-04-04: Добавлен read-only CLI consumption layer для QA (`acp qa --workspace ... --question ...`) поверх `internal/qa`.
- 2026-04-04: Step 1 domain/team card enrichment доведён до deterministic derived section updates для существующих canonical cards без auto-create/rename.
- 2026-04-04: UI baseline editor расширен до selectable набора `charter/*` и `skills/*` артефактов; добавлен UI regression test на save flow.
- 2026-04-04: `golden` workflow зафиксирован на 5 deterministic scenario tests (включая domain envelopes и deterministic-scope exclusion).
- 2026-04-04: Закрыт Step0 wiring gap: `init.step0.constitution` читает `charter/wizard/step0-contract.json`, при missing/invalid contract использует baseline fallback и пишет warning в run diagnostics.
- 2026-04-04: `POST /api/workspace/validate` усилен pre-run layout readiness diagnostics (`workspace.layout.dir.missing|not_dir|unreadable`) + добавлены API/workspace regression tests.
- 2026-04-04: Усилены deterministic derived sections для team cards (`evidence_refs`) и обновлён scenario golden snapshot hash baseline.
- 2026-04-05: Step 1 переведён на реальное per-domain runtime execution с отдельными raw taskruns по каждому canonical domain card; architect summary теперь агрегируется из фактических domain outputs с детерминированной сортировкой.
- 2026-04-05: Усилен internal QA слой (explainable citation reasons, стабилизированная confidence policy, deterministic ranking) и добавлены дополнительные QA regression tests.
- 2026-04-05: Добавлен docs-consistency gate (`internal/docsync`) и синхронизирована каноническая stakeholder matrix (`docs/STAKEHOLDER_DOC.md`) с README/ARCHITECTURE/PLANS/PIPELINE/API docs.
- 2026-04-05: Выполнен cleanup truth-sync slice: удалены stale `future/skeleton/placeholder` формулировки в CLI/help и ключевых docs, добавлены docsync проверки stale markers + CLI docs parity, зафиксирована policy tracked generated artifacts (`internal/api/ui_dist`, `fixtures/scenarios/*/golden/readable`) и добавлены post-beta follow-up cleanup items в backlog.
- 2026-04-05: Закрыт cleanup follow-up по `slugify`: дубли в orchestrator/model/runtime объединены в `internal/slugutil` с unit-tests без изменения публичных контрактов.

---

### Plan ID
EP-20260406-mvp-onboarding-simplification

### Context
Нужно сократить количество ручных действий для первого локального запуска ACP MVP. Сейчас пользователь обязан вручную создавать `arch-workspace` и `workspace.yaml`, из-за чего onboarding получается длинным и хрупким.

### Goals (must have)
- [x] Выполнить repo audit и зафиксировать фактический MVP execution path
- [x] Проверить, что README отражает текущее состояние реализации, и устранить найденные расхождения
- [x] Добавить минимальный bootstrap entrypoint для первичного создания workspace (`workspace.yaml` + layout)
- [x] Обновить быстрый локальный запуск MVP до минимального набора шагов
- [x] Добавить/обновить тесты и smoke coverage под новый onboarding flow

### Non-goals
- [x] Изменение schema/contracts (`schemas/*`, `docs/spec/*`) без строгой необходимости
- [x] Добавление новых runtime в MVP (кроме `fake|headless`)
- [x] Изменение hosted/security boundary (остаётся вне MVP)

### Approach
1) Провести audit реализации (`cmd/internal/ui/scripts`) и сверить с README/ARCHITECTURE/spec.
2) Ввести CLI bootstrap команду для инициализации workspace с минимальным набором обязательных аргументов.
3) Обновить Make/docs quickstart так, чтобы первый локальный запуск был максимально коротким и повторяемым.
4) Закрыть slice тестами (`cmd/acp`, smoke scripts, docs sync).

### Files expected to change
- `cmd/acp/main.go`
- `cmd/acp/main_test.go`
- `Makefile`
- `scripts/smoke-cli.sh`
- `scripts/smoke-api.sh`
- `README.md`
- `docs/STAKEHOLDER_DOC.md`
- `cmd/README.md`
- `cmd/acp/README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/API_SPEC.md`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `internal/orchestrator/orchestrator.go`
- `internal/orchestrator/orchestrator_test.go`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `ui/src/styles.css`
- `docs/PLANS.md`
- `internal/docsync/docsync_test.go`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена
- [x] Первый локальный MVP запуск документирован как минимальный и практически применимый flow

### Risks
- Усложнение CLI surface при добавлении нового bootstrap subcommand.
- Риск документарного рассинхрона между README, `cmd/acp/README.md` и `docs/ARCHITECTURE.md`.

### Progress log
- 2026-04-06: План создан, начат audit текущего состояния и onboarding pain points.
- 2026-04-06: Добавлена CLI команда `acp init-workspace` (bootstrap manifest + fixed layout + dry validation) и обновлён root help surface.
- 2026-04-06: Добавлен `make quickstart-local` для минимального локального потока (`init-workspace` + `run init`), smoke CLI переведён на новый onboarding path.
- 2026-04-06: Обновлены `README`/`cmd` docs/`ARCHITECTURE`/`STAKEHOLDER_DOC` и docsync assertions под новый CLI onboarding.
- 2026-04-06: Пройдены DoD проверки: `make contracts`, `make test`, `make lint`, `make build`, плюс `bash ./scripts/smoke-cli.sh` и `bash ./scripts/smoke-api.sh`.
- 2026-04-06: `acp serve` расширен `--auto-init` и repo flags; startup переведён в lenient mode без блокирующего repo preflight.
- 2026-04-06: Добавлен persisted run history (`reports/taskruns/run-history.json`, retention 500) + API list endpoint `GET /api/pipeline/runs?limit=<n>`.
- 2026-04-06: UI расширен run dashboard (queued/running/succeeded/failed, counters, persisted history selection) + добавлены API/UI/orchestrator regression tests.

---

### Plan ID
EP-20260409-runtime-provider-selection-mvp

### Context
Нужно добавить официальный выбор headless runtime provider в MVP без изменения API/TaskResult контрактов и без нарушения required deterministic CI baseline (`--runtime fake`).

### Goals (must have)
- [x] Ввести provider-aware runtime layer и factory для `claude-code`/`qwen-code`
- [x] Добавить CLI/env contract для provider selection (`--runtime-provider`, `ACP_RUNTIME_PROVIDER`)
- [x] Реализовать native `qwen-code` runner с actionable error mapping (`runner_unavailable`, `runner_parse_failed`)
- [x] Сохранить backward compatibility (default provider `claude-code`, fake baseline неизменён)
- [x] Синхронизировать runtime policy/docs/runbooks и обновить тесты

### Non-goals
- [x] Перевод required CI на live runtime provider checks
- [x] Расширение provider списка beyond `claude-code` и `qwen-code`
- [x] Изменение `TaskResult` schema/model contracts

### Approach
1) Добавить shared runtime contracts/errors и provider factory.
2) Подключить provider selection в `acp run|serve` (flag/env/default precedence).
3) Реализовать `qwen-code` headless runner + parse strategy + tests.
4) Обновить docs/policy/runbooks и прогнать DoD.

### Files expected to change
- `internal/runtime/runtime.go`
- `internal/runtime/providers/*`
- `internal/runtime/claudecode/*`
- `internal/runtime/qwencode/*`
- `internal/orchestrator/*`
- `internal/api/*`
- `cmd/acp/*`
- `README.md`
- `cmd/acp/README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/API_SPEC.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/BASELINE_POLICY.md`
- `docs/STAKEHOLDER_DOC.md`
- `docs/APPENDIX_SCHEMAS.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `scripts/full-run-ai-advent.sh`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена

### Risks
- Drift между фактическими CLI флагами и docs usage.
- Live provider команды могут отсутствовать в среде разработчика.

### Progress log
- 2026-04-09: Добавлен shared runtime layer (`Task/Result/Runner`, provider parse/resolve, unified runner error classifier).
- 2026-04-09: Внедрён provider-aware factory и подключение в CLI/orchestrator/API.
- 2026-04-09: Добавлен native `qwen-code` headless runner с strict TaskResult parse/validation.
- 2026-04-09: Обновлены unit/integration tests для provider selection и headless provider paths.
- 2026-04-09: Синхронизированы README/spec/policy/runbook документы и full-run script под provider-aware runtime contract.

---

### Plan ID
EP-20260409-direct-claude-stabilization

### Context
Нужно стабилизировать прямой headless запуск `claude` без wrapper (`ACP_CLAUDE_CMD=claude`) и убрать шум в quality-артефактах (`runtime_versions`, coverage term drift), сохранив существующие контракты.

### Goals (must have)
- [x] Добавить native direct flow для `claude` в `claudecode` runner (без wrapper)
- [x] Вынести extraction mixed/envelope output в shared helper и использовать в `qwen`/`claude`
- [x] Убрать trailing `@` в `runtime_versions` при пустой версии
- [x] Канонизировать coverage missing terms (`owner/ci-cd/delta/dependency/runtime`) перед дедупом
- [x] Обновить тесты и runbook docs под direct `claude`

### Non-goals
- [x] Изменение TaskResult schema/model contracts
- [x] Изменение CLI/API wire contracts
- [x] Расширение списка providers beyond `claude-code|qwen-code`

### Approach
1) Реализовать dual-mode `claudecode` runner: legacy passthrough + native direct `claude --output-format json -p`.
2) Добавить shared taskresult extractor и подключить в `claudecode`/`qwencode`.
3) Нормализовать quality aggregation (`runtime_versions`) и coverage merge canonicalization.
4) Синхронизировать docs/runbook и прогнать DoD + e2e acceptance.

### Progress log
- 2026-04-09: Добавлен shared extractor `internal/runtime/taskresultextractor` и интеграция в `qwen`/`claude` runner paths.
- 2026-04-09: `claudecode` runner переведён на dual-mode, добавлен native direct flow с retry при parse-fail.
- 2026-04-09: Обновлены orchestrator normalization rules для `runtime_versions` и coverage missing canonicalization.
- 2026-04-09: Добавлены/обновлены unit tests (`claudecode`, extractor, orchestrator quality/coverage), обновлены runtime usage + full-run runbook docs.

---

### Plan ID
EP-20260410-orchestration-prompt-quality

### Context
Нужно снизить семантический шум в refresh-артефактах и стабилизировать содержательный результат headless рантаймов (`claude-code`/`qwen-code`) без изменения публичных контрактов.

### Goals (must have)
- [x] Ввести post-normalize semantic guard для refresh (`step1` фильтр placeholder entities, `step3` fallback finding при owner-gap)
- [x] Усилить step-specific prompt policy для `claude` и `qwen` (allowed/forbidden entities, finding requirement, anti-noise rules)
- [x] Расширить канонизацию/дедуп coverage и questions (ID suffix collapse + text-level dedupe)
- [x] Усилить full-run quality gates semantic checks (owner-gap + empty findings, canonical duplicates)
- [x] Обновить отчётный рендер findings (rule_id, related_ids, evidence refs)

### Non-goals
- [x] Изменение CLI/API/env surface
- [x] Изменение `schemas/taskresult.schema.json`
- [x] Добавление новых runtime providers

### Approach
1) Добавить semantic guard в orchestrator после `NormalizeTaskResult`.
2) Сделать prompts step-aware и provider-consistent для `claude`/`qwen`.
3) Нормализовать merge coverage/questions и report rendering.
4) Добавить semantic проверки в `scripts/full-run-ai-advent.sh` и закрыть тестами.

### Progress log
- 2026-04-10: В `orchestrator` добавлен semantic guard: фильтрация runtime placeholders в `refresh.step1.collect`, fallback findings в `refresh.step3.findings` (включая generic fallback без service candidate).
- 2026-04-10: Обновлены merge правила вопросов/coverage: canonical question IDs, text dedupe, semantic dedupe/канонизация coverage missing+notes.
- 2026-04-10: Hardened prompts для `claude`/`qwen`: step policies, canonical dictionaries, anti-noise contract, retry strict-mode hints.
- 2026-04-10: Усилен full-run quality gate (`check_headless_refresh_semantic_quality`) и обновлены docs (`README`, `ARCHITECTURE`, `LOCAL_FULL_RUN_AI_ADVENT`).
- 2026-04-10: Добавлены/обновлены unit+integration tests и golden fixtures для findings/report changes.

---

### Plan ID
EP-20260410-batch-5x2-frontend-reaudit

### Context
Нужно выполнить повторный e2e re-audit на target repo в формате `5x2` (5 run для `qwen-code` + 5 run для `claude-code`) в direct-only режиме (`qwen`/`claude`, без wrapper), добавить live frontend e2e automation и выпустить агрегированные quality-отчёты.

### Goals (must have)
- [x] Добавить batch runner `5x2` с фиксированным layout артефактов (default: `/tmp/provenarch-test_arch_project/runs/<batch-id>/...`)
- [x] Добавить frontend live e2e automation на Playwright + стабильные `data-testid` в UI
- [x] Добавить локальный script для live UI smoke against running `acp serve` с provider-aware runtime
- [x] Добавить optional CI workflow для manual live UI smoke (не required gate)
- [x] Сформировать агрегированные отчёты: run matrix, frontend matrix, quality report

### Non-goals
- [x] Изменение CLI/API/TaskResult публичных контрактов
- [x] Добавление новых runtime providers beyond `claude-code|qwen-code`
- [x] Перевод live runtime smoke в required CI

### Approach
1) Добавить Playwright e2e test (`validate -> run init -> artifacts`) и `data-testid` для стабильного селектора.
2) Добавить `scripts/frontend-live-e2e.sh` для поднятия backend и запуска Playwright smoke с выбранным provider.
3) Добавить `scripts/full-run-batch-5x2.sh` для preflight + 10 full-run + 2 frontend e2e.
4) Добавить `scripts/e2e_batch_report.py` для rubric-based quality aggregation и markdown отчётов.
5) Обновить README/ARCHITECTURE/TESTING_STRATEGY/LOCAL_FULL_RUN runbook под новые entrypoints.

### Files expected to change
- `ui/src/App.tsx`
- `ui/package.json`
- `ui/package-lock.json`
- `ui/playwright.live.config.ts`
- `ui/e2e/live-flow.spec.ts`
- `scripts/frontend-live-e2e.sh`
- `scripts/full-run-batch-5x2.sh`
- `scripts/e2e_batch_report.py`
- `.github/workflows/ui-live-smoke-optional.yml`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `docs/PLANS.md`

### Progress log
- 2026-04-10: Добавлены `data-testid` для критичных UI секций (`validate/run/status/logs/artifacts`) и Playwright live spec/config/scripts.
- 2026-04-10: Добавлены automation scripts `frontend-live-e2e.sh`, `full-run-batch-5x2.sh`, `e2e_batch_report.py` с provider-aware direct-only режимом.
- 2026-04-10: Добавлен optional manual workflow `ui-live-smoke-optional` (workflow_dispatch, non-required).
- 2026-04-10: Обновлены README/ARCHITECTURE/TESTING_STRATEGY/LOCAL_FULL_RUN runbooks для batch `5x2` и frontend live smoke.

---

### Plan ID
EP-20260411-multi-repo-e2e-matrix

### Context
Нужно расширить локальный e2e контур с single-repo на multi-repo связки без изменения публичных API/CLI/schema контрактов. Канонический input для run scripts должен перейти на `repos-file`, при сохранении backward compatibility.

### Goals (must have)
- [x] Поддержать canonical `TARGET_REPOS_FILE` в `full-run-ai-advent.sh` и `full-run-batch-5x2.sh`
- [x] Сохранить legacy single inputs (`TARGET_REPO`, `TARGET_REPO_GIT_URL+TARGET_REPO_NAME+TARGET_REPO_REF`)
- [x] Добавить profile matrix orchestrator (`full-run-batch-matrix.sh`) для `single-path|single-git_url|multi-path|multi-git_url`
- [x] Усилить batch evaluator: multi-root evidence scope + hard-fail `analysis:cross-repo-missing`
- [x] Обновить frontend live e2e на expected repo count (`UI_E2E_EXPECTED_REPO_COUNT`)
- [x] Синхронизировать README/runbook/testing docs

### Non-goals
- [x] Изменение runtime provider contracts (`claude-code|qwen-code`)
- [x] Изменение CLI/API wire contracts
- [x] Изменение `TaskResult` schema

### Approach
1) Нормализовать target input в scripts через repos-file adapter.
2) Добавить matrix-level launcher с profile metadata.
3) Расширить batch quality evaluator на multi-repo semantics.
4) Обновить frontend live e2e ожидания и doc truth-sync.

### Progress log
- 2026-04-11: Добавлен canonical `TARGET_REPOS_FILE` flow с legacy adapters в `full-run-ai-advent.sh` и `full-run-batch-5x2.sh`.
- 2026-04-11: Добавлен `scripts/full-run-batch-matrix.sh` (`E2E_MATRIX_FILE` profiles) с агрегированным `profile_matrix` отчётом.
- 2026-04-11: `scripts/e2e_batch_report.py` обновлён на multi-root evidence validation и hard-fail `analysis:cross-repo-missing`.
- 2026-04-11: Frontend live e2e расширен `UI_E2E_EXPECTED_REPO_COUNT`; docs (`README`, `LOCAL_FULL_RUN_AI_ADVENT`, `TESTING_STRATEGY`) синхронизированы.

---

### Plan ID
EP-20260411-orchestrator-runner-frontend-operability

### Context
По результатам детального аудита run-flow нужно закрыть операционные пробелы: прозрачность текущего прогресса на UI, полезность runtime логов, предсказуемость restart/recovery на существующем workspace и явные механизмы управления long-running run.

### Goals (must have)
- [x] Улучшить run observability в UI: показывать не только `status/current_step`, но и детальные warnings/error причины для выбранного run
- [x] Улучшить run logs UX: выводить структурированные `fields` (task_id, counters, repo scopes, warning/error payload) и упростить навигацию по step/domain
- [x] Добавить restart reconciliation: при старте сервиса детерминированно переводить "зависшие" `queued/running` run из persisted history в terminal failed state с понятной причиной
- [x] Добавить управляемость run lifecycle: API+orchestrator cancel для активного/pending run
- [x] Улучшить runner failure diagnostics: писать в run logs безопасный срез `stdout/stderr` при parse/runtime fail без утечки лишнего контента
- [x] Синхронизировать docs/spec и покрыть изменения тестами (orchestrator/api/ui)

### Non-goals
- [ ] Не менять `schemas/taskresult.schema.json` и model/entity contracts
- [ ] Не добавлять runtime providers beyond `claude-code` и `qwen-code`
- [ ] Не переводить required CI на live network dependencies

### Approach
1) Backend lifecycle hardening:
   - добавить `CancelRun(run_id)` в orchestrator service (active + pending cases);
   - добавить startup reconciliation pass после `loadHistory()` для stale non-terminal runs;
   - ввести отдельные `error_code` для lifecycle событий (`run_canceled`, `run_reconciled_after_restart`) и использовать их в `RunInfo`.
2) API surface:
   - добавить `POST /api/pipeline/runs/<run_id>/cancel`;
   - сохранить backward compatibility `GET /api/pipeline/runs*`, расширив только допустимые `error_code` semantics и docs.
3) Runtime diagnostics:
   - при runtime fail/parse fail логировать structured snippet полей (`stderr_snippet`, `stdout_snippet`, `task_id`, `provider`) в `RunLogEntry.fields`;
   - ограничить размер snippet и sanitize multiline output для стабильного UI отображения.
4) Frontend run UX:
   - на `Runs: History`/`Run status` показывать warnings list и full error reason выбранного run;
   - в `Runs: Logs` показывать переключаемый structured view (`line` + `fields`);
   - авто-выбирать newest active run при загрузке dashboard, чтобы live polling сразу показывал "что сейчас происходит".
5) Verification + docs sync:
   - добавить/обновить unit/integration/UI tests;
   - обновить `docs/spec/API_SPEC.md`, `docs/ARCHITECTURE.md`, `README.md`, `docs/TESTING_STRATEGY.md`.

### Files expected to change
- `internal/orchestrator/orchestrator.go`
- `internal/orchestrator/runlogs.go`
- `internal/orchestrator/orchestrator_test.go`
- `internal/orchestrator/runlogs_test.go`
- `internal/api/server.go`
- `internal/api/server_test.go`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `docs/spec/API_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`
- `README.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Активный run можно отменить через API, статус и логи отражают cancel reason детерминированно
- [x] После рестарта сервиса stale `queued/running` run не остаются в подвешенном состоянии
- [x] UI для выбранного run показывает полный контекст: status, current step, warnings, error details, логи с structured fields
- [x] Runner parse/runtime fail содержит actionable diagnostics в run logs без избыточного шума
- [x] Обновлены API/docs и пройдены DoD проверки: `make contracts`, `make test`, `make lint`, `make build`

### Risks
- Расширение lifecycle logic может повлиять на текущие async/debounce инварианты и требует аккуратной синхронизации mutex/state.
- Неверно подобранные log snippets могут либо быть слишком шумными, либо потерять полезную диагностику.
- Добавление cancel/reconciliation без четкой error-code policy может ухудшить читаемость run history.

### Progress log
- 2026-04-11: План создан на основе code-level аудита orchestrator/runtime/UI run-flow, логирования и restart behavior.
- 2026-04-12: Реализованы orchestrator cancel/reconciliation, API cancel endpoint, structured runtime diagnostics snippets в run logs, UI auto-select + warnings + log fields view + cancel flow, добавлены unit/API/UI тесты и docs sync.

---

## EP-20260414-runtime-streaming-ux-c4-prompts

### Context
- Требуется закрыть 5 пользовательских болей: raw runtime logs в `Runs: Logs`, non-single-page UX с переносом settings, рендер C4 диаграмм, качество baseline prompts, общий UX redesign.
- Ограничения: не менять `TaskResult` schema и runtime provider list (`claude-code|qwen-code`) в MVP.

### Slice Plan
1. **Slice A (runtime stream):** live forwarding stdout/stderr в run logs, совместное существование event/raw stream, hard-cap safeguard + truncation marker.
2. **Slice B (reports/C4):** Step 2 materialize full C4 Mermaid set (`Context/Container/Component/Code`) + `reports/diagrams/index.md`, strict evidence-first gaps.
3. **Slice C (frontend UX):** top tabs `Setup/Baseline/Runs/Results/Settings`, settings relocation, `Results -> Diagrams`, `Runs: Logs` dual mode.
4. **Slice D (baseline prompts):** rewrite prompt packs + skill prompts в структурированные deterministic defaults, quality guard tests against short placeholders.

### Contract/Surface Changes
- `GET /api/pipeline/runs/<run_id>/logs` wire shape расширен полями `kind` и `stream`.
- Step 2 artifacts расширены `reports/diagrams/*` и artifact kinds `diagram`, `diagram-index`.
- UI surface intentionally breaking (updated tab/navigation + selectors in tests).

### Progress Log
- 2026-04-14: Slice A реализован (runtime `OnOutput` seam, provider stream forwarding, orchestrator raw log entries + truncation event).
- 2026-04-14: Slice B реализован (compiler `CompileC4Diagrams`, Step2 wiring, diagrams index/materialization, deterministic tests).
- 2026-04-14: Slice C реализован (tabbed UX, settings relocation, logs dual-mode, diagrams preview, Vitest+Playwright updates).
- 2026-04-14: Slice D реализован (structured prompt defaults rewrite + quality tests, create-if-missing policy сохранена).

### Verification
- Backend: `go test ./internal/runtime/... ./internal/orchestrator ./internal/api ./internal/reports ./internal/workspace`
- Frontend: `npm --prefix ui run typecheck`, `npm --prefix ui run test -- --run`
- DoD gates: `make contracts`, `make test`, `make lint`, `make build`

---

## Implemented vs Planned (operational mirror)

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Таблица ниже — инженерный mirror и должна оставаться синхронизированной с канонической матрицей.

| Epic | Статус | Комментарий |
|---|---|---|
| 1 Workspace/contracts | done (beta baseline) | Schema-driven + semantic validation, resolver `path/git_url`, diagnostics API |
| 2 TaskResult foundations | done (beta baseline) | Validation + normalization legacy forms, contract tests |
| 3 Runtime/orchestration seam | done (beta baseline) | Fake default + opt-in headless runtime selector with provider choice (`claude-code` default, `qwen-code`), raw taskruns materialization |
| 4 Model deterministic core | done (beta baseline) | Canonical IDs/collision rules + deterministic regression tests |
| 5 Pipeline 0–4 | done (beta baseline) | `init|refresh` runnable через CLI/API |
| 6 UI baseline | done (beta baseline) | Setup/validate/run/inspect + editors + git helpers |
| 7 Domain-first layer | done (beta baseline) | Per-domain contracts + deterministic Step 1 enrichment canonical domain/team cards without auto-create |
| 8 Baseline bundle | done (beta baseline) | `skills/subagents.yaml` + prompt packs + validation |
| 9 Q&A capability | done (beta boundary) | Workspace-backed QA service + read-only CLI `acp qa`; публичный endpoint остаётся follow-up |
| 10 Changelog compilers | done (beta baseline) | Iteration changelog materialization в `reports/changelog/*` |
| 11 `POST /api/qa/ask` | follow-up (post-beta) | Не входит в required beta surface |
| 12–13 | out of MVP | Вне текущего beta scope |
| 14 CI trigger mode | done (beta baseline) | CLI batch required, smoke/golden/ui-smoke jobs без live network deps |
| 15 Domain/baseline pack hardening | done (beta baseline) | Baseline skills/prompts wired и versioned в workspace |
