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
EP-20260420-qwen-step0-draft-contract-regressions

### Context
Live canonical `regres fast` показал универсальный hard-stop на `qwen-code` в `init.step0.constitution`: provider возвращает schema-valid `TaskResult`, но пишет `constitution-draft.json` в legacy constitution shapes (`schema_version`, `version: "0.1.0"`, `services/...`) вместо runtime draft manifest contract. Текущий orchestrator gate корректно ловит ошибку, но runtime не валидирует/не чинит draft-only artifacts до возврата, а qwen prompt/retry helpers остаются collect-centric.

### Goals (must have)
- [x] Вынести runtime draft manifest contract в shared internal package и использовать его в writer + validator
- [x] Сделать `qwen` step0 prompt/retry contract-aware для `constitution-draft.json` и убрать legacy model-first drift
- [x] Добавить runtime-side validation и один artifact-repair retry для draft-only steps (`step0/2/4`)
- [x] Расширить safe normalization legacy `add_doc_artifact` drift на `step0` без silent legacy payload conversion
- [x] Зафиксировать live step0 regression fixtures/tests и синхронизировать docs

### Non-goals
- [ ] Не менять public JSON schemas, API endpoints и `workspace.yaml`
- [ ] Не трогать harness/reporting/incomplete-cycle logic в этом slice
- [ ] Не добавлять broad auto-conversion legacy constitution payloads в canonical draft manifest

### Approach
1) Вынести manifest structs/validation/file-existence checks в shared draft-contract package и подключить его в orchestrator/runtime writer paths.
2) Переписать qwen draft-only prompt/retry helpers: exact `constitution-draft.json` example, draft-aware recovery phrasing, `changeset: []` template для `step0/2/4`.
3) После parse/binding success валидировать required draft artifacts в qwen runtime и при invalid manifest запускать один constrained repair retry поверх `write_root + draft_final_root`.
4) Расширить compatibility normalization только на duplicate `add_doc_artifact` drift при уже валидном step0 draft manifest.
5) Добавить recorded fixtures/tests на observed live legacy step0 shapes и обновить docs/architecture notes.

### Files expected to change
- `internal/orchestrator/*`
- `internal/runtime/qwencode/*`
- `internal/runtime/claudecode/*`
- `internal/runtime/taskresultcompat/*`
- `internal/runtime/testdata/*`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Canonical `constitution-draft.json` contract валидируется из shared package и одинаково используется writer/validator
- [x] `qwen` step0 prompt и retry prompt больше не тянут collect/shard wording и synthetic `upsert_entity` template
- [x] Invalid step0 draft manifest идёт в один runtime repair retry и при неудаче падает как `runner_parse_failed` с raw artifacts
- [x] Valid step0 draft manifest + legacy `add_doc_artifact` ops нормализуются без silent semantic payload conversion
- [x] Regression fixtures/tests покрывают observed live step0 legacy shapes

### Risks
- Основной риск — случайно расширить safe normalization до semantic auto-conversion legacy constitution payloads. Снижение риска: shared validator остаётся строгим, а runtime normalization ограничена только duplicate `add_doc_artifact` drift при уже валидном manifest.

### Progress log
- 2026-04-20: зафиксирован implementation slice по `qwen` step0 draft-contract regressions; подтверждены observed live legacy manifest shapes (`schema_version`, `version:"0.1.0"`) и collect-centric prompt drift в qwen runtime.
- 2026-04-20: live rerun после основного фикса выявил дополнительный qwen-specific post-artifact stall на `step0`, когда provider писал canonical `constitution-draft.json`, но оставлял draft files под `draft_final_root/<canonical_path>` и зависал без финального JSON; добавлены safe file-layout reconciliation, draft-step stall watchdog/retry и regression tests.

### Plan ID
EP-20260420-regres-small-live-triage

### Context
Нужно выполнить canonical live `regres fast` (`bank-of-anthos + openedx`, затем `openstack`) через `scripts/full-run-batch-matrix.sh` на trusted host, непрерывно мониторить статус прогонов и после завершения собрать детальный triage-отчёт по фактическим багам/недоработкам без изменения public contract.

### Goals (must have)
- [ ] Запустить оба canonical `regres fast` matrix slice из clean worktree с `BATCH_PROVIDER_FILTER=qwen-code`
- [ ] Непрерывно мониторить matrix/batch progress до terminal state каждого slice
- [ ] Собрать run artifacts, classifications и release/profile reports для каждого slice
- [ ] Зафиксировать продуктовые/runtime/harness баги и отделить их от operational noise
- [ ] Составить итоговый отчёт: что сломалось, где именно, что нужно чинить дальше

### Non-goals
- [ ] Не менять matrix taxonomy, timeout profiles и public schemas во время самого прогона
- [ ] Не подменять canonical harness wrapper-скриптом или ad-hoc env overrides
- [ ] Не пытаться "чинить на лету" live run до завершения triage цикла

### Approach
1) Проверить trusted-host prerequisites, pinned path checkouts и clean detached worktree.
2) Выполнить `regres fast` как два последовательных вызова `scripts/full-run-batch-matrix.sh` с qwen-only provider filter.
3) Во время выполнения регулярно читать batch/matrix logs, status files и intermediate reports, чтобы не пропустить stall/incomplete state.
4) После завершения разобрать terminal artifacts: `profile-runs.jsonl`, matrix status files, batch classifications, session summaries, release/profile verdicts и raw taskrun diagnostics.
5) Сформировать детальный triage-отчёт и список необходимых фиксов.

### Files expected to change
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Оба `regres fast` slice дошли до terminal state (`passed|failed`) с сохранёнными matrix/batch breadcrumbs
- [ ] Есть собранный список фактических failure classes и их root cause
- [ ] Итоговый отчёт отделяет runtime/provider проблемы от harness/reporting проблем

### Risks
- Основной риск — long-running live provider hangs или внешние transport/API failures. Снижение риска: canonical watchdog/reporting уже встроены; во время прогона мониторинг идёт по нескольким источникам (`driver log`, batch status, profile status, taskrun artifacts), а не только по одному summary-файлу.

### Progress log
- 2026-04-20: prerequisites и pinned path SHA проверены, prepared clean detached worktree `/private/tmp/provenarch-run-clean`, запуск `regres fast` начинается.

### Plan ID
EP-20260420-qwen-preartifact-stall-reporting

### Context
После фикса `collect_stalled_after_artifacts` live `regres small` всё ещё мог зависать раньше первого authored collect artifact: `qwen-code` на `step1.collect` оставался живым, но не писал ни stdout/stderr, ни `write_root`, из-за чего runtime не доходил до repair path. Параллельно batch/matrix harness оставляли слабые terminal breadcrumbs, если parent shell завершался до post-processing.

### Goals (must have)
- [x] Добавить отдельный `pre-artifact stall` watchdog для `qwen` collect steps с forced retry до общего step timeout
- [x] Сохранить существующий `after-artifacts stall` path без semantic regression
- [x] Усилить runtime diagnostics полями `stall_phase`, `last_pipe_activity_at`, `last_write_root_mutation_at`, `manifest_state`, `authored_file_count`
- [x] Сделать child-owned run sentinel в `full-run-ai-advent.sh` и использовать его как durable source of truth в batch/matrix/report paths
- [x] Убрать stale `queued|running` хвост через explicit reconcile helper в CLI/orchestrator и зафиксировать это тестами

### Non-goals
- [x] Не менять public schemas/API/workspace contract
- [x] Не вводить user-facing knobs для stall thresholds
- [x] Не чинить OS-level `SIGKILL` всего process tree

### Approach
1) Ввести в `internal/runtime/qwencode/runner.go` второй stall sentinel для collect steps, который срабатывает до manifest/doc artifacts при отсутствии pipe activity и `write_root` mutations.
2) При `pre-artifact stall` досрочно завершать process, писать diagnostics и запускать один fresh retry с ужесточённым collect prompt.
3) Перенести terminal sentinel ownership внутрь `full-run-ai-advent.sh`, а batch/matrix/report scripts научить классифицировать incomplete cycles по sentinel даже без summary/report artifacts.
4) Вынести restart reconciliation из implicit startup behavior в explicit helper и вызывать его из CLI/tests там, где это действительно требуется.
5) Закрыть slice regression tests на runtime, orchestrator resume и batch/matrix reconstruction.

### Files expected to change
- `internal/runtime/qwencode/*`
- `internal/orchestrator/*`
- `cmd/acp/*`
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch-5x2.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/PLANS.md`

### Acceptance criteria
- [x] `qwen` collect shard без stdout/stderr и без artifacts детектится как `collect_stalled_before_artifacts` и идёт в forced retry
- [x] failed retry после pre-artifact stall отдаёт `runner_parse_failed` и сохраняет raw runner diagnostics
- [x] batch/matrix/report классифицируют incomplete cycle даже при missing `session-summary.md` и неполном `profile-runs.jsonl`
- [x] interrupted/orphaned CLI history больше не остаётся в вечном `running`
- [x] `make contracts`, `make test`, `make lint`, `make build` проходят

### Risks
- Основной риск — ложный pre-artifact stall на legitimately slow collect shard. Снижение риска: watchdog включён только для `init/refresh.step1.collect`, использует двойной idle сигнал (`stdout/stderr` + `write_root`) и conservative threshold `75s`.

### Progress log
- 2026-04-20: реализованы pre-artifact stall watchdog, forced fresh retry, durable run/profile sentinels, batch/matrix/report reconstruction и explicit stale-run reconciliation; полный DoD прогон зелёный.

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
EP-20260420-qwen-live-regress-stabilization

### Context
Нужно стабилизировать `qwen` live regress small без изменения public contracts: закрыть doubled-path drift в collect manifests, ужесточить collect-repair/runtime diagnostics, правильно классифицировать provider transport failures и перестать маркировать terminal pipeline failures как `infra_incomplete_cycle`.

### Goals (must have)
- [x] Сделать shard manifest contract artifact-root-aware и нормализовать repaired `documents[].path`
- [x] Исправить `qwen` collect repair hints/diagnostics и transport error classification
- [x] Исправить batch/report reconstruction для terminal pipeline failures
- [x] Добавить regression fixtures/tests и синхронизировать docs

### Non-goals
- [x] Не менять public JSON schemas / API / `workspace.yaml`
- [x] Не менять release matrix composition или timeout profiles

### Approach
1) Ужесточить manifest validation и canonicalization вокруг `artifact_root`.
2) Усилить `qwen` repair prompt и выделить provider transport transcript errors в `runner_unavailable`.
3) Исправить batch/report precedence: terminal summary + terminal sentinel не равны incomplete cycle.
4) Добавить recorded fixtures и regression tests под live findings.

### Files expected to change
- `internal/contracts/*`
- `internal/artifactquality/*`
- `internal/runtime/qwencode/*`
- `internal/runtime/taskresultextractor/*`
- `scripts/full-run-batch-5x2.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/ARCHITECTURE.md`

### Acceptance criteria
- [x] `step2` больше не читает doubled paths из collect manifests
- [x] `qwen` transport/API transcript errors классифицируются как `runner_unavailable`
- [x] terminal pipeline failures больше не попадают в `infra_incomplete_cycle`
- [x] tests/fixtures/docs обновлены

### Risks
- Слишком агрессивная manifest normalization может скрыть реальные contract violations; исправлять только случаи, которые однозначно указывают внутрь `write_root/artifact_root`.

### Progress log
- 2026-04-20: старт slice по стабилизации `qwen` live regress small; подтверждены three live failure classes (`documents[].path`, collect-repair drift, false incomplete-cycle classification) и отдельный step0 SSL transport path.
- 2026-04-20: реализованы artifact-root-aware manifest validation + canonical path normalization, `qwen` transport/API classification в `runner_unavailable`, richer collect repair hints/diagnostics, corrected terminal batch/report classification; regression fixtures/tests и docs sync завершены, DoD (`make contracts test lint build`) пройден в изолированной копии.

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

### Plan ID
EP-20260420-qwen-live-regress-stabilization

### Context
Нужно закрыть подтвержденные live-regress баги `qwen` runtime/harness после прогона `regres fast`: поздний stall retry из-за pipe drain, неполный collect repair, отсутствие draft-artifact gate для step0/2/4 и ложный `running` в batch/matrix при аварийном завершении.

### Goals (must have)
- [ ] Ускорить forced retry после collect stall без ожидания полного EOF stdout/stderr
- [ ] Дожать collect manifest contract enforcement после retry
- [ ] Добавить required-artifact gate для `constitution/asis/proposals` draft manifests
- [ ] Сделать terminal batch/matrix status durable даже при partial/abnormal exit
- [ ] Обновить tests/fixtures/docs и прогнать DoD

### Non-goals
- [ ] Не менять public JSON schemas, API endpoints и `workspace.yaml`
- [ ] Не добавлять user-facing knobs для stall thresholds или draft validation

### Approach
1) Перестроить `qwen` stall path вокруг short drain + immediate retry sentinel.
2) Усилить runtime contract validation для collect и draft-step artifacts.
3) Добавить durable parent/child terminal breadcrumbs в live harness и reconstruction.

### Files expected to change
- `internal/runtime/qwencode/*`
- `internal/runtime/taskresultcompat/*`
- `internal/orchestrator/runtime_drafts.go`
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch-5x2.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`

### Acceptance criteria
- [ ] Stall retry реально ранний и покрыт тестом с зависшим stdout pipe
- [ ] Invalid collect/draft artifacts падают как contract/runtime failures с raw diagnostics
- [ ] Batch/matrix partial exit больше не оставляет stale `running`
- [ ] Документация и regression tests синхронизированы

### Risks
- В основном дереве уже есть unrelated изменения `internal/api/ui_dist/*`; не затрагивать их и прогонять verification аккуратно, при необходимости в изолированной копии.

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
