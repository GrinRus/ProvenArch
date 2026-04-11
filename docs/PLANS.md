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
