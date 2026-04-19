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
EP-20260419-qwen-collect-stall-recovery

### Context
Live `regres small` показал, что `qwen-code` на `init.step1.collect` может дойти до authored collect artifacts (`shard-pack-manifest.json` + draft docs), но продолжать бессмысленный repo sweep и не возвращать финальный `TaskResult` до внешнего hard-stop. Из-за этого existing repair path не запускался, а batch/matrix harness оставляли run history без финальной классификации.

### Goals (must have)
- [x] Добавить ранний stall watchdog для `qwen` collect steps и forced artifact-repair retry до общего step timeout
- [x] Усилить collect/retry prompt discipline правилом "no explore after write"
- [x] Пробросить explicit runtime diagnostic events про stall/retry в orchestrator logs
- [x] Сохранить raw stdout/stderr artifacts и normal `runner_parse_failed` semantics при failed stall recovery
- [x] Устранить blind spot batch/matrix harness: incomplete child cycles получают classification/record даже без `session-summary.md`

### Non-goals
- [x] Не менять public schemas/API
- [x] Не добавлять user-facing timeout/stall knobs
- [x] Не чинить OS-level external `SIGKILL` всего matrix process tree

### Approach
1) Ввести provider-local collect stall monitor в `internal/runtime/qwencode/runner.go`, основанный на реальной pipe activity и мутациях `write_root`.
2) При stall завершать provider process, писать diagnostics и сразу запускать forced `RETRY RECOVERY MODE` с existing artifact repair contract.
3) Пробросить diagnostics в orchestrator event logs без расширения `TaskResult`.
4) Добавить per-run sentinel/status files в batch harness и заставить matrix/report path работать на partially completed roots.
5) Зафиксировать поведение тестами и минимально синхронизировать docs/runbook.

### Files expected to change
- `internal/runtime/runtime.go`
- `internal/runtime/qwencode/*`
- `internal/orchestrator/*`
- `scripts/full-run-batch-5x2.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`

### Acceptance criteria
- [x] qwen collect stall после authored artifacts уходит в forced retry и не ждёт step timeout
- [x] failed stall recovery оставляет `reports/taskruns/raw/*` и `runner_parse_failed`, а не `runner_unavailable`
- [x] orchestrator logs содержат `runtime task stalled after artifacts` / `retry scheduled` / `retry completed`
- [x] batch/matrix классифицируют incomplete child cycle даже без `session-summary.md`
- [x] regression tests покрывают runtime, logs и batch/matrix failure accounting

### Risks
- Основной риск — ложное stall detection на legitimately long collect shards. Снижение риска: watchdog включён только для qwen collect steps и срабатывает только после authored artifacts плюс двойного idle сигнала (`stdout/stderr` и `write_root`).

### Progress log
- 2026-04-19: добавлены qwen collect stall watchdog, forced recovery retry, runtime diagnostics, batch/matrix sentinels/classification и regression tests.

### Plan ID
EP-20260419-step-scoped-agent-pipeline

### Context
Нужно заменить legacy process-scoped runner flow на step-scoped agent-first pipeline без отдельного compatibility mode. Canonical workspace по-прежнему должен писаться только через deterministic compile/validate/publish слой, а выбор provider теперь делается на уровне шага с fallback на глобальные CLI/env настройки.

### Goals (must have)
- [x] Ввести `runtime.profile.steps.*.provider` в manifest/schema/API и сохранить precedence `workspace step override > CLI/env global > claude-code`
- [x] Перевести `step0..step4` на step-scoped runtime resolution с provider cache/preflight внутри одного run
- [x] Добавить runtime contract поля `draft_final_root`, `step_contract`, `expected_artifacts` и staged draft manifests для `step0/2/4`
- [x] Сохранить compile/validate/publish как единственную canonical write surface и убрать обязательный human gate на promotion
- [x] Обновить fake/test/docs/examples/fixtures под новую базовую архитектуру и прогнать DoD

### Non-goals
- [x] Не менять `schemas/taskresult.schema.json` в этом slice
- [x] Не добавлять per-step execution knobs beyond provider selection
- [x] Не расширять provider list beyond `claude-code|qwen-code`

### Approach
1) Заменить single-runner seam на `StepRunnerResolver` с per-provider cache/preflight и runtime metadata per step.
2) Добавить step-scoped runtime contract и staged draft manifests; запретить runtime прямую запись в canonical workspace.
3) Перевести `step0/2/4` на agent-first execution с compile/publish gating и auto-promotion после validator/schema checks.
4) Синхронизировать API/spec/docs/examples/goldens и подтвердить поведение через contract/orchestrator/API tests.

### Files expected to change
- `internal/orchestrator/*`
- `internal/runtime/*`
- `internal/api/*`
- `internal/workspace/*`
- `cmd/acp/*`
- `schemas/workspace.schema.json`
- `docs/spec/*`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/APPENDIX_SCHEMAS.md`
- `docs/STAKEHOLDER_DOC.md`
- `docs/adr/ADR-20260410-headless-runtime-multi-provider.md`
- `examples/workspace.example.yaml`
- `fixtures/scenarios/*/golden/snapshot.sha256`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Schema/API/runtime contracts синхронизированы
- [x] Документация и examples/fixtures обновлены

### Risks
- Основной риск — потерять canonical compile invariants при появлении draft-first шагов. Снижение риска: runtime drafts остаются staging-only surface, а coverage/findings/as-is/proposals канонизируются compile/publish слоем.

### Progress log
- 2026-04-19: введены step provider resolution, staged draft manifests, agent-first `step0/2/4`, auto-publish без human gate и API/runtime profile surfaces.
- 2026-04-19: синхронизированы tests/goldens/docs; full DoD прогон запланирован после финальной contract/docs ревизии.

### Plan ID
EP-20260418-regres-fast-artifact-quality

### Context
Canonical `regres fast` всё ещё краснеет не только из-за реальной bank-like деградации refresh-артефактов, но и из-за ложных blockers в batch report, `contract:runtime-name` и frontend cancel/init smoke. Публичные схемы менять нельзя; нужно довести runtime/harness/report/frontend поведение до уже зафиксированной policy.

### Goals (must have)
- [x] Добавить provider-side artifact-fidelity repair для collect steps и синхронизировать qwen/claude guardrails
- [x] Исправить batch report, чтобы non-release verdict считался только по реально выбранным provider/run slots
- [x] Убрать ложный `contract:runtime-name` для internal shard-plan/shard-summary артефактов
- [x] Стабилизировать frontend init-inspect/cancel smoke и сохранить `run_canceled` при конкурирующем terminal failure
- [x] Синхронизировать docs/skill и прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)

### Non-goals
- [ ] Не менять five-profile taxonomy
- [ ] Не менять public schemas (`TaskResult`, `validator-verdict`, `final-run-index`, `citation-index`)
- [ ] Не добавлять wrapper-скрипт поверх matrix harness

### Approach
1) Вынести shared helper для rich/skeletal manifest assessment и использовать его в runtime retry + docflow quality warnings.
2) Добавить post-success artifact-repair attempt для collect steps, не ломая existing parse-retry contract.
3) Перевести batch quality report на selected-provider/selected-run aware aggregation и убрать hardcoded `10/10`.
4) Проставить runtime metadata в internal shard-plan/shard-summary JSON и починить frontend smoke/cancel path.
5) Обновить docs/skill только по реально изменившемуся поведению и повторно прогнать non-live DoD.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/qwencode/runner.go`
- `internal/runtime/qwencode/runner_test.go`
- `internal/runtime/claudecode/runner.go`
- `internal/runtime/claudecode/runner_test.go`
- `internal/orchestrator/docflow.go`
- `internal/orchestrator/sharding.go`
- `internal/orchestrator/*_test.go`
- `scripts/e2e_batch_report.py`
- `scripts/full-run-batch-5x2.sh`
- `scripts/write-batch-preflight.py`
- `scripts/tests/batch_failure_classification_test.py`
- `ui/e2e/live-flow.spec.ts`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] Bank-like collapse получает automatic repair attempt и остаётся blocker только если repair не улучшил artifacts
- [x] Openstack-like rich reuse не считается ложным дефектом
- [x] Batch report не генерирует фантомные `backend_total_runs=10` и `summary_missing=9` для qwen-only non-release runs
- [x] Internal shard-plan/shard-summary больше не триггерят `contract:runtime-name`
- [x] Frontend init-inspect и cancel smoke больше не падают на stale test id / missing `run_canceled`

### Risks
- Основной риск в artifact-repair path: не ухудшить already-good manifests повторным retry. Для этого repair должен запускаться только для collect steps с poor manifest и иметь rollback на исходный write_root при неудачном retry.

### Progress log
- 2026-04-18: старт slice на стабилизацию `regres fast` после frozen-state findings; scope зафиксирован как artifact-quality first, затем batch/runtime/frontend false blockers.
- 2026-04-18: реализованы shared artifact-quality heuristics, collect repair retry/rollback, selected-surface batch aggregation, runtime metadata stamping и cancel precedence; `make contracts`, `make test`, `make lint`, `make build` прошли.

### Plan ID
EP-20260417-live-e2e-profile-taxonomy

### Context
Нужно заменить wave-centric live E2E narrative на более гранулированную 5-профильную таксономию без изменения текущего runner contract. Concrete `profile_id` остаются прежними, а named профили (`regres fast|long`, `release fast|long|full`) вводятся как checked-in composite presets поверх прямых вызовов `full-run-batch-matrix.sh`.

### Goals (must have)
- [x] Добавить checked-in catalog с sizing policy, repo-set shard classification и expected backend totals
- [x] Добавить runnable matrix slices для новых high-level профилей без изменения matrix parser/release-mode contract
- [x] Синхронизировать runbook/testing docs/skill под новую canonical taxonomy и пометить старые wave files как legacy/compat
- [x] Добавить tests на новые release/non-release slice shapes и catalog integrity

### Non-goals
- [x] Не менять approved concrete profile ids (`single-path`, `single-git_url`, `multi-path`, `multi-git_url`)
- [x] Не добавлять wrapper-скрипт поверх `scripts/full-run-batch-matrix.sh`

### Approach
1) Зафиксировать canonical taxonomy в `examples/e2e-profile-catalog.yaml`.
2) Разбить named профили на минимальные runnable matrix slice-файлы.
3) Обновить docs/skill так, чтобы canonical source of truth ссылался на catalog/slices, а legacy wave matrices остались только для compatibility.
4) Расширить matrix tests на новые slice shapes и catalog expansion counts.

### Files expected to change
- `examples/e2e-profile-catalog.yaml`
- `examples/e2e-matrix.*.yaml`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `.agents/skills/e2e-live-gate/SKILL.md`
- `scripts/tests/matrix_release_contract_test.py`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Catalog покрывает все 6 canonical repo sets и 5 named profiles
- [x] Документация и skill синхронизированы с новой taxonomy

### Risks
- Главное место риска — смешение старых wave файлов с новой canonical taxonomy. Их нужно сохранить рабочими, но явно вывести из основного release narrative.

### Progress log
- 2026-04-17: добавлены catalog и canonical matrix slices для `regres fast|long`, `release fast|long|full`.
- 2026-04-17: runbook/testing docs/skill переведены на 5-profile taxonomy; wave matrices оставлены как legacy compatibility.
- 2026-04-17: добавлены slice-shape и catalog integrity regression tests, выполнена exploratory fake validation shard counts для canonical repo sets.

### Plan ID
EP-20260417-live-e2e-baseline-vs-full

### Context
Нужно развести два live E2E контура: быстрый baseline regression для ежедневной отладки (`wave1`, `qwen`, implicit baseline, 2 backend runs total) и полный trusted-machine прогон для release/debug escalation.

### Goals (must have)
- [x] Ввести отдельный regression matrix example для `wave1`
- [x] Зафиксировать в docs/skill, что baseline regression = `qwen-only`, а release/full run остаётся полным прогоном
- [x] Добавить regression test на 2-profile non-release matrix

### Non-goals
- [ ] Не менять release-mode matrix contract
- [ ] Не менять provider list (`claude-code`, `qwen-code`)

### Approach
1) Добавить non-release matrix example с `single-path + multi-git_url`.
2) Документировать запуск baseline regression через `BATCH_PROVIDER_FILTER=qwen-code`.
3) Оставить official release wave1/wave2 как full run и закрепить это отдельной проверкой.

### Files expected to change
- `examples/e2e-matrix.*.yaml`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `.agents/skills/e2e-live-gate/SKILL.md`
- `scripts/tests/matrix_release_contract_test.py`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Baseline regression и full run описаны без двусмысленности
- [x] Новый regression matrix example добавлен

### Risks
- Если не развести baseline/full run достаточно явно, команда будет путать быстрый qwen-only smoke с release-ready verdict.

### Progress log
- 2026-04-17: старт slice на разделение quick baseline regression и full run/release gate.
- 2026-04-17: добавлен `examples/e2e-matrix.regression-wave1.yaml`, docs/skill разведены на quick baseline vs full run, пройдены `python3 -m unittest scripts.tests.matrix_release_contract_test` и `go test ./internal/docsync`.

### Plan ID
EP-20260417-canonical-live-e2e-stabilization

### Context
После перехода на 5-profile taxonomy canonical `regres fast` должен исполняться из clean committed tree без `BATCH_SKIP_PRECHECK`. Для этого нужно синхронизировать committed harness с новым matrix contract, усилить qwen retry/JSON discipline на live shard output и закрепить clean-tree preflight в docs/skill.

### Goals (must have)
- [x] Синхронизировать harness и tests с новым non-release/release matrix contract
- [x] Усилить qwen prompt/retry discipline и добавить regression fixture на реальный invalid live stdout pattern
- [x] Обновить runbook/testing docs/skill под canonical clean-tree preflight
- [x] Повторно прогнать non-live DoD после фиксов harness/tests
- [ ] Довести canonical `regres fast` live acceptance до финального verdict без `BATCH_SKIP_PRECHECK`

### Non-goals
- [x] Не менять 5-profile taxonomy, catalog и curated repo sets
- [x] Не добавлять wrapper-скрипт поверх `scripts/full-run-batch-matrix.sh`
- [x] Не менять product API и `schemas/taskresult.schema.json`

### Approach
1) Убрать из tracked harness legacy-ожидание "в матрице должны быть все 4 concrete profile id" и привести release verdict math к реальному expansion.
2) Зафиксировать qwen live parse regression fixture и усилить prompt/retry contract только под подтверждённые event-stream patterns.
3) Синхронизировать runbook/skill/strategy на clean committed tree или отдельный clean worktree.
4) Повторить canonical `regres fast` из clean worktree без `BATCH_SKIP_PRECHECK` и проверить verdict files.

### Files expected to change
- `scripts/full-run-batch-matrix.sh`
- `scripts/full-run-batch-5x2.sh`
- `scripts/tests/matrix_release_contract_test.py`
- `internal/runtime/qwencode/runner.go`
- `internal/runtime/qwencode/runner_test.go`
- `internal/runtime/taskresultextractor/extractor_test.go`
- `internal/runtime/testdata/*`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] Matrix contract tests покрывают release/non-release slices и implicit baseline
- [x] Qwen regression fixture и retry path покрыты тестами
- [x] Документация и skill описывают canonical clean-tree preflight
- [x] `make contracts test lint build` проходит после фиксов
- [ ] Canonical `regres fast` live rerun завершён с PASS verdict без `precheck_failed` и `runner_parse_failed`

### Risks
- Основной риск остался runtime-level: qwen может продолжать выдавать длинный prose/tool-planning перед финальным `TaskResult`, из-за чего canonical acceptance придётся подтверждать реальным trusted-host rerun, а не только unit coverage.

### Progress log
- 2026-04-17: synced committed harness with new matrix contract, including non-release positive `RUN_COUNT` and release-family validation.
- 2026-04-17: added qwen live invalid stdout fixture, strengthened prompt/retry discipline, and expanded runner/extractor regression tests.
- 2026-04-17: documented clean-tree canonical preflight in runbook, testing strategy, local full-run guide, and `acp-e2e-live-gate` skill.
- 2026-04-17: reran `make contracts test lint build` successfully after fixing `REPORTS_ROOT`/`MATRIX_ROOT` env leakage in matrix contract tests.
- 2026-04-17: canonical `regres fast` acceptance rerun started from clean worktree without `BATCH_SKIP_PRECHECK`; qwen advanced past the original immediate parse-failure path and entered retry-backed shard execution.
- 2026-04-17: clean rerun showed the remaining canonical blocker moved from `runner_parse_failed` to `runtime_timeout` under legacy `pipeline_timeout=2400s`; follow-up slice adds matrix-native `timeout_profile` presets to canonical matrices and harness.

### Plan ID
EP-20260417-live-e2e-matrix-downsize

### Context
Нужно сократить manual live E2E release surface без потери базового coverage: оставить по одному `single` и `multi` профилю на wave, запускать по одному backend run на provider и сохранить release sweeps `baseline` + `parallel-default`.

### Goals (must have)
- [x] Перевести release-mode matrix contract на `RUN_COUNT=1`
- [x] Требовать в official release matrix ровно один `single-*` и один `multi-*` профиль
- [x] Обновить official wave1/wave2 matrices и release verdict expectations
- [x] Синхронизировать skill/runbook/testing docs и regression tests

### Non-goals
- [ ] Не менять provider list (`claude-code`, `qwen-code`)
- [ ] Не убирать canonical sweeps `baseline` и `parallel-default`

### Approach
1) Ослабить generic matrix parser до approved profile ids, а release-mode сузить до `single + multi`.
2) Сделать expected backend totals динамическими от `RUN_COUNT`, зафиксировав release-mode на `1`.
3) Пересобрать official wave matrices и docs под новый minimal live baseline.

### Files expected to change
- `scripts/full-run-batch-matrix.sh`
- `scripts/full-run-batch-5x2.sh`
- `scripts/tests/*`
- `examples/e2e-matrix.release-wave*.yaml`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Release contract валидирует новый minimal matrix
- [x] Документация и skill синхронизированы

### Risks
- Слишком жёсткий/неочевидный release contract может сломать существующие trusted-host процедуры; нужно держать ошибки fail-fast и явно описать allowed profile families.

### Progress log
- 2026-04-17: старт slice на уменьшение live E2E matrix до minimal single+multi baseline с `RUN_COUNT=1`.
- 2026-04-17: release-mode matrix переведён на `RUN_COUNT=1`, official wave matrices сужены до `single + multi`, обновлены skill/runbooks/docs и пройдены `python3 -m unittest scripts.tests.matrix_release_contract_test`, `go test ./internal/docsync`, `bash -n scripts/full-run-batch-matrix.sh scripts/full-run-batch-5x2.sh`.

### Plan ID
EP-20260416-doc-first-runtime-pipeline

### Context
Нужно перевести ACP с model-first применения `TaskResult` на docs-first staged runtime pipeline: shard writers пишут только в run-scoped staging surface, orchestrator собирает/проверяет staged artifacts, а promotion в canonical `reports/*` и `proposals/*` выполняется только после validator verdict.

### Goals (must have)
- [x] Ввести staged runtime contracts (`shard pack manifest`, `final run index`, `citation index`, `validator verdict`)
- [x] Добавить explicit `read_context_roots` / `write_root` / `artifact_root` в runtime task metadata
- [x] Перевести orchestrator steps `step1.collect` и `step3.findings` на docs-first artifacts вместо primary `TaskResult` application
- [x] Сохранить `model/*` только как derived compatibility layer
- [x] Синхронизировать schemas/docs/tests/fixtures

### Non-goals
- [x] Не менять provider list (`claude-code`, `qwen-code`)
- [x] Не разрешать runtime запись в `workspace.yaml`, `schemas/*`, `docs/spec/*`, `charter/*` или user repos

### Approach
1) Добавить новые schema/contracts и staged runtime task metadata.
2) На `step1.collect` materialize-ить shard packs в `reports/taskruns/<run_id>/staging/shards/*`.
3) Собрать staged final set + indexes, прогнать validator verdict и only-then promote canonical outputs.
4) Держать `model/*` как derived extraction из final index/citations для compatibility.

### Files expected to change
- `internal/contracts/*`
- `internal/runtime/*`
- `internal/orchestrator/*`
- `internal/reports/*`
- `schemas/*`
- `fixtures/scenarios/*`
- `README.md`
- `docs/*`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена

### Risks
- Переход задевает почти все pipeline fixtures и scenario golden outputs; риск регрессий в compatibility surface (`model/*`, diagrams, UI results) нужно страховать derived extraction и scenario tests.

### Progress log
- 2026-04-16: старт исполнения docs-first staged runtime slice.
- 2026-04-16: primary path переведён на runtime-authored staged docs; compiler layer оставлен как compatibility fallback для отсутствующих surfaces, добавлены path policy guard и docflow negative tests.

### Plan ID
EP-20260416-structural-sharding-reset

### Context
Нужно вернуть sharding к structural full-repo coverage contract: `heuristics` должен строить deterministic non-overlapping partition, `semantic` должен стать metadata-only, а runtime execution contract должен перестать зависеть от `repo_selection/backend_only`.

### Goals (must have)
- [x] Переписать planner на structural coverage roots + bounded coalescing
- [x] Убрать `repo_selection/backend_only` из schema/runtime/API/UI/reporting
- [x] Перевести release harness на sweeps `baseline` + `parallel-default`
- [x] Добавить invariant `baseline == parallel-default` по shard-plan
- [x] Синхронизировать tests/docs/skills

### Non-goals
- [x] Не менять public `schemas/taskresult.schema.json`
- [x] Не вводить новый explicit repo filtering contract

### Approach
1) Упростить workspace/runtime execution до always-all-repos semantics.
2) Пересобрать shard planner вокруг structural coverage и metadata-only semantic graph.
3) Обновить matrix/reporting/docs под новый release contract и invariant.

### Files expected to change
- `internal/orchestrator/*`
- `internal/runtime/*`
- `internal/workspace/*`
- `internal/api/*`
- `scripts/*`
- `ui/*`
- `schemas/workspace.schema.json`
- `examples/*`
- `README.md`
- `docs/*`
- `.agents/skills/e2e-live-gate/*`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы/спеки синхронизированы
- [x] Legacy `repo_selection/backend_only` убран из active contract surfaces

### Risks
- Structural coalescing должен сохранять full coverage и не смешивать top-level subtree; release harness дополнительно страхует это matrix invariant-ом.

### Progress log
- 2026-04-16: реализованы structural full-coverage shard planner, metadata-only semantic mode, removal of runtime repo filtering, updated matrix/reporting contract и docs sync.

### Plan ID
EP-20260416-zero-signal-hardening

### Context
Нужно убрать ложную семантику empty reports после shard/runtime сбоев и поднять первичный runtime incident в batch triage, не меняя public TaskResult schema.

### Goals (must have)
- [x] Поднять shard outcomes в run-level `evidence_state`
- [x] Перевести markdown artifacts на explicit incomplete/partial semantics
- [x] Исправить batch failure attribution для raw `runner_parse_failed`
- [x] Добавить regression tests и синхронизировать docs

### Non-goals
- [x] Не менять public `schemas/taskresult.schema.json`
- [x] Не менять UI code

### Approach
1) Расширить orchestrator quality/report context на основе shard outcomes.
2) Сменить compiler semantics для `as-is/findings/coverage/proposals/agent-outputs`.
3) Обновить harness triage precedence и добавить script-level fixtures.

### Files expected to change
- `internal/orchestrator/*`
- `internal/reports/*`
- `scripts/full-run-batch-5x2.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена

### Risks
- `make build` в основном дереве затрагивает уже грязный `internal/api/ui_dist/*`; build verification безопасно выполнять в изолированной временной копии.

### Progress log
- 2026-04-16: реализованы `evidence_state`, incomplete/partial report semantics, batch failure precedence, regression tests и docs sync.

### Archived
- Completed historical plans moved to `docs/archive/PLANS_ARCHIVE_2026-04.md` (archived on 2026-04-15).

---

## Implemented vs Planned (operational mirror)

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Таблица ниже — инженерный mirror и должна оставаться синхронизированной с канонической матрицей.

| Epic | Статус | Комментарий |
|---|---|---|
| 1 Workspace/contracts | done (beta baseline) | Schema-driven + semantic validation, resolver `path/git_url`, diagnostics API |
| 2 TaskResult foundations | done (beta baseline) | Validation + canonical-only TaskResult contract, contract tests |
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
| 14 CI trigger mode | done (beta baseline) | CLI batch required, smoke/golden jobs без live network deps |
| 15 Domain/baseline pack hardening | done (beta baseline) | Baseline skills/prompts wired и versioned в workspace |
