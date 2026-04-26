# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон, текущие активные планы и инженерный operational mirror.

Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
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
### Plan ID
EP-20260426-strict-runtime-no-compatibility-shims

### Context
После выравнивания live provider adapters в runtime остались compatibility-шымы, которые могли молча переписать provider artifacts после выполнения: collect manifest path/metadata canonicalization и draft file reconciliation из `outputs[].canonical_path`. Обратная совместимость с такими malformed artifacts больше не требуется; success source of truth должен быть strict artifact-only validation.

### Goals (must have)
- [x] Удалить active compatibility registry и rule-id diagnostics
- [x] Сделать collect validation read-only: без autofill metadata и без `documents[].path` normalization
- [x] Сделать draft validation read-only: без копирования draft files из `outputs[].canonical_path` в `outputs[].path`
- [x] Сохранить только manifest-only provider repair для collect shards с authored docs + missing/invalid manifest
- [x] Перенести deterministic fake runtime в provider-neutral package
- [x] Переименовать child batch harness в нейтральное имя без wrapper для старого пути
- [x] Обновить tests/docs под no-compat behavior

### Non-goals
- [x] Не менять public artifact schemas
- [x] Не добавлять backward-compat wrapper для старого имени batch script

### Approach
1) Заменить локальный collect repair/canonicalization на strict read-only manifest validation с legacy precheck.
2) Удалить draft-root reconciliation path и tests, которые ожидали compatibility mutations.
3) Обновить provider adapters так, чтобы qwen не имел отдельной repair-named artifact validation обёртки.
4) Перенести fake runtime из `claudecode` в `fakeruntime`, сохранив deterministic artifacts.
5) Переименовать child batch harness с legacy имени на `full-run-batch.sh` и синхронизировать matrix harness/docs/tests.
6) Добавить engine write-set guard для manifest-only collect repair: всё кроме `shard-pack-manifest.json` остаётся contract failure.
7) Синхронизировать docs/spec/testing с strict no-compat runtime behavior.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/providercommon/*`
- `internal/runtimedrafts/*`
- `internal/runtime/qwencode/*`
- `internal/runtime/fakeruntime/*`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `docs/*`

### Acceptance criteria
- [x] Artifact validation never rewrites collect manifests or draft files
- [x] Artifact-root-prefixed/absolute collect document paths fail strict validation
- [x] Draft files written only at `outputs[].canonical_path` fail strict validation
- [x] Manifest-only provider repair remains available and engine-enforced to write only `shard-pack-manifest.json`
- [x] Deterministic fake runtime is no longer implemented inside `claudecode`
- [x] Active docs/tests/scripts no longer reference the old child batch script name
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`

### Risks
- Existing live providers may fail more often after hidden normalization is removed. That is intentional: failures should surface as `runtime_contract_failed` and be fixed via prompt/adapter behavior, not post-hoc mutation.

### Progress log
- 2026-04-26: Implemented strict collect/draft validation, removed compatibility registry/reconciliation shims, moved deterministic fake runtime to `fakeruntime`, renamed the child batch harness, added collect repair write-set guard, updated tests/docs, and completed full DoD plus `git diff --check`.

### Plan ID
EP-20260426-live-provider-collect-contract-stabilization

### Context
Smoke tiny live triage показал общий collect-contract failure surface: `qwen-code`/`claude-code` доходят до `init.step1.collect`, но оставляют `shard-pack-manifest.json` missing/invalid, а `codex-code` после обновления CLI снова должен участвовать как полноценный peer. Success source остаётся artifact-only; stdout/stderr являются diagnostics.

### Goals (must have)
- [x] Добавить общий manifest-only repair path для collect steps после authored docs + missing/invalid `shard-pack-manifest.json`
- [x] Расширить artifact-state diagnostics: manifest state, authored artifact count, raw stdout/stderr refs
- [x] Сузить qwen `runner_unavailable` до fully silent/no-artifact paths; partial artifacts без валидного manifest остаются `runtime_contract_failed`
- [x] Сохранить thin adapters для `claude-code`, `qwen-code`, `codex-code`
- [x] Добавить selected-provider readiness guard, включая codex `gpt-5.5`/CLI version mismatch
- [x] Обновить docs/spec/testing/runbook/live-e2e skill

### Non-goals
- [x] Не менять product API, workspace schema, public artifact schemas или release matrices
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`
- [x] Не расширять MVP provider set

### Approach
1) В `providercommon` добавить optional collect repair adapter interface, shared diagnostics и классификацию partial collect artifacts как contract failure.
2) Подключить manifest-only repair prompt ко всем live adapters; qwen сохраняет fresh retry только для no-artifact/missing-invalid paths.
3) Усилить collect prompt policy против markdown-only completion.
4) В batch preflight записывать selected provider readiness и блокировать известный codex model/version mismatch до deep run.
5) Синхронизировать docs/skill и покрыть runtime/preflight tests.

### Files expected to change
- `internal/runtime/providercommon/*`
- `internal/runtime/{claudecode,codexcode,qwencode,promptcontract,steppolicy}/*`
- `scripts/full-run-batch.sh`
- `scripts/write-batch-preflight.py`
- `docs/*`, `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] Unit tests cover manifest-only collect repair success/failure
- [x] qwen partial collect artifacts without valid manifest classify as `runtime_contract_failed`
- [x] qwen no-output/no-artifact retry exhaustion remains `runner_unavailable`
- [x] codex `0.125.0` + `gpt-5.5` passes readiness guard; old `0.118.0` is blocked
- [x] Top-level `release_verdict_*.json.backend` aggregate exists for canonical acceptance checks
- [x] `claude-code`/`codex-code` adapters use shared pre-artifact stall monitoring instead of waiting for full hard timeout on silent/no-artifact hangs
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`
- [x] Trusted-machine smoke tiny rerun for `qwen-code` captured residual live collect repair failure
- [x] Trusted-machine smoke tiny rerun for `codex-code` confirmed updated CLI readiness and the same collect-manifest residual on `bank-of-anthos-extras`
- [x] Trusted-machine smoke tiny rerun for `claude-code`
- [x] Post-tightening trusted smoke tiny rerun for `qwen-code`
- [x] Post-pre-artifact-monitor trusted smoke tiny rerun for `codex-code`/`claude-code`

### Risks
- Manifest-only repair can still fail if provider wrote no authored docs at all; that must stay explicit `runtime_contract_failed`/`runner_unavailable`, not be hidden as artifact quality.
- Provider auth/rate-limit remains operational and must surface as `runner_unavailable` with raw diagnostics.

### Progress log
- 2026-04-26: Implemented shared collect manifest repair path, adapter repair prompts, artifact-state diagnostics, qwen partial-artifact classification guard, codex readiness guard, and targeted tests. Full DoD and live reruns pending.
- 2026-04-26: Full DoD passed. Trusted `qwen-code` smoke tiny `smoke-tiny-bank-qwen-20260426T104225Z` failed as expected on residual live behavior: authored collect docs were present, manifest-only repair stalled without producing `shard-pack-manifest.json`; retry partial-artifact classification was tightened after this run.
- 2026-04-26: Trusted `codex-code` smoke tiny `smoke-tiny-bank-codex-20260426T112719Z` passed selected-provider readiness on `codex-cli 0.125.0`/`gpt-5.5`, then failed as `runtime_contract_failed` on `bank-of-anthos-extras`: authored `extras-overview.md` existed, but manifest-only repair still stalled without `shard-pack-manifest.json`. The residual exposed repair-surface drift, so repair include dirs were narrowed to current `write_root` + repo evidence, repair prompt now treats embedded schema text as authoritative, and repair watchdog uses a bounded 90s repair window instead of the normal 20s post-artifact stall.
- 2026-04-26: Post-tightening `qwen-code` smoke tiny `smoke-tiny-bank-qwen-20260426T130440Z` completed with verdict `FAIL`, mode `non-release`, selected provider `qwen-code`, run index `1`, backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=0`, `runtime_timeout=0`; frontend `never` remained skipped/non-blocking. Task logs confirmed manifest-only repair scheduled/exhausted per partial collect shard.
- 2026-04-26: During post-tightening `codex-code` smoke tiny `smoke-tiny-bank-codex-20260426T135500Z`, updated CLI readiness passed and initial step/first collect shards succeeded, but `bank-of-anthos-extras` exposed a shared lifecycle gap: `claude-code`/`codex-code` lacked pre-artifact stall monitoring for silent/no-artifact hangs. The diagnostic run was terminated after capturing the issue; adapters now use shared artifact-step pre-artifact monitoring and release verdict JSON now has a top-level backend aggregate for canonical acceptance checks.
- 2026-04-26: Post-pre-artifact-monitor smoke tiny reruns completed for all three live providers: `codex-code` `smoke-tiny-bank-codex-postfix-20260426T144025Z` (`FAIL`, non-release, backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=0`, `runtime_timeout=0`), `claude-code` `smoke-tiny-bank-claude-postfix-20260426T150624Z` (`FAIL`, non-release, backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=0`, `runtime_timeout=0`), and `qwen-code` `smoke-tiny-bank-qwen-postfix-20260426T152149Z` (`FAIL`, non-release, backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=1`, `runtime_timeout=0`). The qwen `runner_unavailable` came from a later silent/no-artifact non-collect path; collect partial artifacts remained classified as `runtime_contract_failed`.

### Plan ID
EP-20260425-provider-runtime-adapter-alignment

### Context
Qwen live smoke показал, что provider уже успевает записать валидные artifacts, но старый qwen-only watchdog мог убить процесс и классифицировать artifact-only success как `runner_unavailable`. У `claude-code` и `codex-code` не было такого же watchdog, но был общий риск: разные process lifecycle paths при одинаковом artifact-only runtime contract.

### Goals (must have)
- [x] Вынести process lifecycle для `claude-code`, `qwen-code`, `codex-code` в общий `providercommon` engine
- [x] Оставить provider differences в adapters: command/args/stdin/workdir, unavailable markers, activity/recovery policy
- [x] Принять valid artifacts after controlled stop как success
- [x] Убрать qwen dependency на `--output-format json`
- [x] Сохранить `qwen` silent missing-artifact/retry exhaustion как `runner_unavailable`, а malformed artifacts как `runtime_contract_failed`
- [x] Исправить non-release frontend `never` verdict semantics без ослабления release-mode strict frontend checks

### Non-goals
- [x] Не менять product API, workspace schema или public artifact schemas
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`
- [x] Не расширять MVP provider set

### Approach
1) Добавить shared `providercommon` process engine с raw diagnostics, activity monitor, controlled stop/retry и artifact validation.
2) Переключить `claudecode`, `qwencode`, `codexcode` на thin adapters поверх engine и удалить qwen-only process executor.
3) Добавить conformance tests для artifact-only success/failure/timeout/unavailable paths и provider args tests.
4) Синхронизировать architecture/spec/testing/runbook/live-e2e skill.

### Files expected to change
- `internal/runtime/providercommon/*`
- `internal/runtime/{claudecode,codexcode,qwencode}/*`
- `scripts/full-run-batch-matrix.sh`
- `docs/*`, `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] `go test ./internal/runtime/...`
- [x] targeted matrix release-contract tests for non-release frontend `never` and release strict frontend blockers
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] Trusted-machine smoke tiny qwen rerun executed; verdict remained blocked by live `qwen-code` partial/missing artifact behavior (`runner_unavailable`), with frontend `never` correctly non-blocking

### Risks
- Live CLI behavior can still fail due external auth/rate limits; such failures must stay explicit `runner_unavailable` with raw-output refs.
- Release-mode frontend checks must remain strict even though non-release diagnostic `frontend=never` is now non-applicable.

### Progress log
- 2026-04-25: Implemented shared provider engine/adapters, removed qwen-only process executor, fixed qwen output args, added conformance tests, fixed non-release frontend-never verdict semantics, and synchronized docs/runbook/skill.
- 2026-04-25: Follow-up audit closed residual gaps: adapter contract now exposes include dirs, `claude`/`codex` adapter conformance tests cover noninteractive diagnostic flags and shared unavailable markers, and shared engine now applies explicit adapter retry policy after normal-exit missing/invalid artifact validation failures.
- 2026-04-25: Full local DoD passed (`make contracts`, `make test`, `make lint`, `make build`). Trusted smoke tiny qwen diagnostic `smoke-tiny-bank-20260425T133718Z` produced reports but failed with backend `runner_unavailable`; raw diagnostics show qwen left five collect shards partial/missing while valid artifacts-after-stop were accepted for the first shard.
- 2026-04-25: Final defect audit fixed retry classification edge case: an initial structurally malformed artifact contract cannot be masked by a later silent retry as `runner_unavailable`; added regression coverage and reran full DoD.

### Plan ID
EP-20260425-flexible-live-e2e-selector

### Context
Live E2E surface сейчас имеет canonical release taxonomy и low-level env selectors (`BATCH_PROVIDER_FILTER`, `BATCH_RUN_SELECTION`), но нет удобного catalog-driven способа получить прямые команды для комбинаций вроде `regres + codex + fast`, `regres + claude + full` или супербыстрого `1 repo × 1 run × 1 provider` smoke. Нужно добавить этот слой без превращения его в wrapper поверх release harness и без изменения official release verdict contract.

### Goals (must have)
- [x] Добавить command generator, который только печатает direct `scripts/full-run-batch-matrix.sh` commands
- [x] Добавить `smoke tiny` selector (`bank-of-anthos`, one provider, one run, frontend off)
- [x] Добавить diagnostic `regres full` selector на все 6 canonical repo sets, включая Sentry
- [x] Сохранить canonical release taxonomy и release verdict source unchanged
- [x] Явно зафиксировать artifact quality как обязательный gate для regress/release
- [x] Синхронизировать runbook/testing strategy/live-e2e skill
- [x] Прогнать targeted tests и DoD (`make contracts`, `make test`, `make lint`, `make build`)

### Non-goals
- [x] Не добавлять executable wrapper, который сам запускает release matrix
- [x] Не менять public schemas/product APIs
- [x] Не делать standalone artifact-eval command в этом slice

### Approach
1) Расширить `examples/e2e-profile-catalog.yaml` отдельным `selectors[]` слоем, не меняя canonical `profiles[]`.
2) Добавить diagnostic matrix files для `smoke tiny` и Sentry baseline.
3) Реализовать `scripts/live-e2e-plan.py` с выводом `shell|json|markdown`.
4) Покрыть selector normalization, provider shorthand, release subset rejection, run counts и direct command output unit tests.
5) Обновить docs/skill и выполнить DoD.

### Files expected to change
- `scripts/live-e2e-plan.py`
- `scripts/tests/live_e2e_plan_test.py`
- `examples/e2e-profile-catalog.yaml`
- `examples/e2e-matrix.smoke-tiny.bank.yaml`
- `examples/e2e-matrix.diagnostic.sentry.yaml`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] `smoke tiny` generates exactly one backend run for exactly one provider
- [x] `regres full` is non-release and includes all six canonical repo sets
- [x] `release fast|long|full` generated commands keep all three release providers
- [x] Provider subset is rejected for release selectors
- [x] Full release provider set is accepted in any CLI order and normalized to canonical order
- [x] `frontend-mode=never` disables both init and cancel frontend smoke in generated commands
- [x] Generated regress/release metadata declares existing artifact-quality gate path

### Risks
- Главный риск: оператор может принять generated diagnostic selector за official release verdict. Mitigation: docs/skill/script metadata помечают `smoke`/`regres full` как diagnostic/non-release, а release readiness остаётся только `release_verdict_<matrix-id>.json`.

### Progress log
- 2026-04-25: Started implementation of catalog-driven live E2E command generator, smoke tiny, diagnostic regres full, docs/skill sync, and unit tests.
- 2026-04-25: Added generator/catalog/matrix/docs/skill changes and passed targeted tests, full `scripts/tests` discovery, and DoD (`make contracts`, `make test`, `make lint`, `make build`).
- 2026-04-25: Audit fixed release provider ordering bug (`codex,qwen,claude` now accepted as the full release provider set) and added regression coverage.
- 2026-04-25: Follow-up audit fixed `frontend-mode=never` command generation to set `BATCH_FRONTEND_CANCEL_MODE=never` as well, and tightened release provider diagnostic guidance.

### Plan ID
EP-20260425-runtime-providers-stabilization

### Context
Live `regres fast` triage показал подтверждённый ACP product defect: `step4.proposals` prompt/step policy был слабее `collect/as_is/findings`, поэтому provider смог записать legacy proposal envelope, а strict parser корректно завершил run как `runtime_contract_failed`. Параллельно нужен меньший provider-specific drift между `claude/qwen/codex`, non-empty codex runtime metadata и durable evidence после cleanup temp roots.

### Goals (must have)
- [x] Усилить enforced `proposals-draft-manifest.json` contract без ослабления strict parser
- [x] Добавить parser-level guard для proposals publish surface
- [x] Вынести shared provider validation/raw diagnostic helpers; full process lifecycle alignment superseded by `EP-20260425-provider-runtime-adapter-alignment`
- [x] Исправить `codex-code` runtime meta до `codex-code/headless`
- [x] Добавить durable matrix inventory с bounded raw-output refs
- [x] Синхронизировать docs/spec/runbook и fixtures/tests
- [ ] Перепроверить live `claude-code × bank-openedx`, затем qwen-focused `regres fast`, затем full fresh-all-6 на trusted host

### Non-goals
- [x] Не менять публичные API/schemas/release taxonomy
- [x] Не добавлять compatibility alias layer для legacy proposals manifests
- [x] Не менять canonical matrix/profile/timeout files под текущую машину
- [x] Не добавлять wrapper scripts поверх matrix harness

### Approach
1) Добавить proposals contract helpers/example и подключить их в enforced prompt, step policy, repair hints и baseline prompt bundle.
2) Расширить runtime draft validator для `step4.proposals`: allowed canonical targets `proposals/*` + `reports/changelog/*`, duplicate target rejection.
3) Создать `internal/runtime/providercommon` для artifact validation, stream capture и raw failure message persistence; qwen переиспользует только safe helpers.
4) Зафиксировать codex runtime metadata и расширить raw failure metadata/inventory для post-cleanup triage.
5) Обновить docs/spec/runbook/README и покрыть unit/script tests.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/promptcontract/*`
- `internal/runtime/steppolicy/*`
- `internal/runtimedrafts/*`
- `internal/runtime/{providercommon,claudecode,codexcode,qwencode,runnerdiag}/*`
- `internal/orchestrator/sharding*`
- `internal/workspace/baseline*`
- `scripts/full-run-batch-matrix.sh`
- `scripts/resolve-repos-meta.py`
- `scripts/tests/*`
- `docs/*`

### Acceptance criteria
- [x] Targeted Go runtime/provider tests pass
- [x] Resolver and matrix durable-inventory script tests pass
- [x] Full DoD (`make contracts`, `make test`, `make lint`, `make build`) passes in this worktree
- [ ] Live verification completed on trusted host

### Risks
- Provider-runtime outages (`qwen runner_unavailable`, `codex runtime_timeout`) may still block canonical live confidence even after ACP contract fixes.
- Existing dirty WIP in matrix/preflight/docflow files must not be accidentally reverted or mixed into unrelated semantic changes.

### Progress log
- 2026-04-25: Implemented proposals contract hardening, parser guard, shared providercommon helpers, codex runtime meta, raw diagnostic metadata, durable matrix inventory, fixtures/tests, and docs sync. Full DoD passed; live verification remains pending.
- 2026-04-25: Closed remaining operational blocker surface gap: missing selected provider binary and path pinned-SHA mismatch now materialize terminal `operational_host_preflight_failed` status/inventory/verdict before child batch execution. Full DoD passed again.
- 2026-04-25: Final audit fixed raw failure artifact overwrite risk by switching runnerdiag names from second-resolution stamps to nanosecond-resolution stamp + pid + atomic sequence and exclusive file creation; rapid-failure regression coverage added. Live verification remains pending.

### Plan ID
EP-20260423-regres-fast-failure-taxonomy-hardening

### Context
Сравнительный live triage по `qwen` и `codex` показал четыре новых дефекта: `step4.proposals` всё ещё может запускаться после unusable collect, `qwen` местами смешивает provider unavailable и artifact contract failures, `full-run-ai-advent.sh` может ложно раздувать `completed_runs` из-за malformed `run-results.tsv`, а frontend live E2E не различает explicit failure и productive timeout.

### Goals (must have)
- [x] Не запускать `init|refresh.step4.proposals` после unusable collect
- [x] Сохранить collect/runtime contract как primary root cause вместо downstream proposal parse failures
- [x] Развести `runtime_contract_failed` и `runner_unavailable` в `qwen` artifact-validation paths
- [x] Исправить `run-results.tsv` accounting так, чтобы malformed rows не превращали успешный backend в `infra_incomplete_cycle`
- [x] Поднять raw `runtime_contract_failed` выше heuristic `runner_unavailable` в shell/python batch reporting
- [x] Развести `playwright_failed` и `active_run_timeout` для frontend live E2E
- [x] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Перепроверить live `regres fast` минимум на `qwen bank/openedx` и `codex bank/openedx`

### Non-goals
- [x] Не менять публичные schemas/contracts
- [x] Не ослаблять strict runtime draft manifest validator
- [x] Не трактовать codex plugin/state/cache noise как primary blocker

### Approach
1) Закрыть orchestrator path для proposals после unusable collect и добавить regression coverage.
2) Уточнить qwen artifact-failure classification на missing-artifacts vs malformed-artifacts.
3) Починить structured accounting в `full-run-ai-advent.sh`.
4) Выровнять precedence между raw runtime/session-summary и grep/signature overrides в shell/python reporting.
5) Развести frontend active timeout и explicit Playwright/backend failure, затем синхронизировать docs.

### Files expected to change
- `internal/orchestrator/*`
- `internal/runtime/qwencode/*`
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch.sh`
- `scripts/e2e_batch_report.py`
- `scripts/frontend-live-e2e.sh`
- `scripts/frontend-status-reasons.sh`
- `ui/e2e/live-flow.spec.ts`
- `scripts/tests/*`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Unusable collect не запускает `step4.proposals`
- [x] Malformed runtime draft/shard artifacts остаются `runtime_contract_failed`, даже если рядом есть capacity/429 markers
- [x] Missing artifacts + explicit provider unavailable markers остаются `runner_unavailable`
- [x] `run-results.tsv` считает только валидные строки с фиксированным числом полей
- [x] Batch/reporting держат `runtime_contract_failed` выше `runner_unavailable`, если raw runtime уже дал contract failure
- [x] Frontend live E2E пишет `active_run_timeout` для продуктивного long-running run вместо generic `playwright_failed`

### Risks
- Главный риск: переагрессивно поднять `runtime_contract_failed` и потерять реальные provider incidents. Снижение риска: provider-unavailable остаётся только для explicit markers + missing-artifact shapes, а malformed manifests/unknown fields сохраняются в contract lane.

### Progress log
- 2026-04-23: implementation started for proposals skip gating, qwen artifact classification split, structured ai-advent run accounting, batch/report precedence hardening, and frontend active timeout differentiation.
- 2026-04-23: completed DoD (`make contracts`, `make test`, `make lint`, snapshot `make build`) and hardened shell/python classifiers to ignore `codex` plugin/Cloudflare/state-db noise as primary `runner_unavailable`; live `regres fast` reruns remain pending.

### Plan ID
EP-20260423-regres-fast-qwen-hardening

### Context
Последний live diagnostic выявил четыре blocker класса: semantic duplicates (`user-service` vs `userservice` + owner-gap finding clones), неверный frontend workspace handoff (`headless/arch-workspace` не подхватывался), недоклассификацию provider capacity/rate-limit как `runner_unavailable`, и слабый host preflight до matrix запуска.

### Goals (must have)
- [x] Усилить semantic dedupe для service token variants и finding signature merge
- [x] Ввести единый workspace resolver (`headless/arch-workspace` -> `arch-workspace` -> `workspace`) в shell/python harness
- [x] Добавить capacity/rate-limit сигнатуры в shell/python failure classifiers с сохранением приоритета `runtime_timeout > runner_unavailable > runtime_flow_failed > runtime_contract_failed`
- [x] Не эскалировать `best_effort` partial в `runtime:execution-semantics`, если подтверждён provider unavailable signal
- [x] Усилить host preflight и пометку operational blocker до live matrix run
- [ ] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Выполнить canonical live `regres fast` на `qwen-code` (оба matrix slices) и выпустить execution report

### Non-goals
- [x] Не менять публичные schemas/contracts (`release_verdict_*`, `profile_matrix_*`, `run_matrix_*`, `quality_report_*`)
- [x] Не менять canonical matrix/curated presets под текущую машину

### Approach
1) Исправить dedupe в orchestrator (`semanticEntityDedupKey`, findings-by-signature merge).
2) Унифицировать workspace resolution в `full-run-batch.sh` и `e2e_batch_report.py`.
3) Расширить runtime/provider classification на capacity/rate-limit сигналы в shell/python.
4) Усилить matrix host preflight (`qwen` binary/version + writable roots + path SHA checks для pinned refs).
5) Прогнать DoD, затем mandatory qwen `regres fast` и собрать evidence-based blocker report.

### Files expected to change
- `internal/orchestrator/docflow.go`
- `internal/orchestrator/docflow_test.go`
- `scripts/full-run-batch.sh`
- `scripts/e2e_batch_report.py`
- `scripts/resolve-repos-meta.py`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/batch_failure_classification_test.py`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] `user-service/userservice` и дубли owner-gap findings схлопываются детерминированно
- [ ] Frontend init/cancel не получает false `*_workspace_missing`, если workspace есть в `headless/arch-workspace`
- [ ] Capacity/rate-limit инциденты классифицируются как `runner_unavailable` (при отсутствии explicit timeout)
- [ ] `best_effort` partial не получает runtime semantics blocker только из-за отсутствующего `run_partial_failed`, если есть provider unavailable signal
- [ ] Host preflight ловит невалидный qwen/writable roots/pinned path SHA mismatch как operational blocker до live run

### Risks
- Главный риск: переагрессивный finding dedupe может объединить разные инциденты. Снижение риска: signature ограничен `rule_id + normalized related_ids + normalized title` и merge остаётся deterministic.

### Progress log
- 2026-04-23: implementation started for semantic dedupe hardening, workspace resolver, classifier capacity signals, and host preflight enforcement.

### Plan ID
EP-20260422-headless-legacy-cleanup

### Context
Live `regres fast` на `codex-code` показал, что collect provider может писать legacy `shard-pack-manifest` shape и подхватывать schema drift из runtime workspace (`reports/taskruns`, raw logs, archived artifacts). Параллельно обнаружены downstream gaps: `step2.asis_docs` стартует даже при unusable collect, а batch/matrix harness может оставлять child/profile в stale `running`.

### Goals (must have)
- [x] Зафиксировать shared canonical-only collect contract для `claude-code`, `qwen-code`, `codex-code`
- [x] Убрать workspace-root fallback и legacy schema scavenging из collect runtime surface
- [x] Добавить explicit legacy precheck перед strict collect canonicalization
- [x] Не запускать live `step2.asis_docs` при unusable collect
- [x] Переводить terminal-less child/profile runs в `infra_incomplete_cycle`
- [ ] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить новый canonical live `regres fast` matrix и зафиксировать следующий triage cycle

### Non-goals
- [x] Не расширять schema/contracts под legacy aliases
- [x] Не добавлять permissive semantic compatibility repair
- [x] Не менять release taxonomy или canonical matrix harness command

### Approach
1) Усилить shared prompt/policy/baseline contract для collect и явно запретить legacy aliases.
2) Ограничить collect runtime surface до `write_root` + selected repo roots + explicit `read_context_roots`, включая collect cwd.
3) Добавить raw legacy detector в artifactquality и сохранить strict parse/validate path без semantic repair.
4) Исправить orchestrator skip path для unusable collect и harness reconciliation для stale `running`.
5) Синхронизировать spec/runbook/docs и затем прогнать DoD + новый live matrix cycle.

### Files expected to change
- `internal/runtime/*`
- `internal/artifactquality/*`
- `internal/orchestrator/*`
- `internal/workspace/*`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/*`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Collect prompts/contracts for all headless providers encode canonical-only vocabulary and legacy bans
- [ ] `step1.collect` no longer sees workspace-root by default and uses `write_root` as working directory
- [ ] Legacy collect payloads fail with explicit precheck diagnostics before strict parse
- [ ] `step2.asis_docs` is skipped when collect evidence is unusable
- [ ] Batch/matrix status files never remain silently `running` after child completion
- [ ] DoD passes and a new live `regres fast` cycle is launched for triage

### Risks
- Основной риск — пережать runtime surface и случайно отрезать нужные repo reads. Снижение риска: collect access остаётся на selected repo roots, а `step2/3` продолжают использовать explicit `read_context_roots`.

### Progress log
- 2026-04-22: implementation started for shared prompt cleanup, collect surface hardening, explicit legacy precheck, `step2` unusable-collect skip and batch/matrix stale-running reconciliation.

### Plan ID
EP-20260422-headless-legacy-residuals

### Context
После первого live rerun остались residual failure classes: collect semantic evidence всё ещё может дрейфовать в citation-only shape, findings verdict prompt недоописывает canonical metadata trio, `qwen-code` маскирует quota/auth как `runtime_contract_failed`, а matrix reconciliation не добивает stale `profile-status=running` по owner heartbeat.

### Goals (must have)
- [x] Дожать shared collect contract до обязательного `repo/path` внутри semantic provenance evidence
- [x] Добавить shared canonical contract/example для `validator-verdict.json`
- [x] Переклассифицировать `qwen-code` post-run quota/auth failures в `runner_unavailable`
- [x] Добавить durable child owner sidecar + matrix stale-profile reconciliation
- [ ] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить следующий live `regres fast` cycle и зафиксировать новый triage

### Non-goals
- [x] Не менять `schemas/*` и public contracts
- [x] Не добавлять semantic compatibility repair для legacy payloads
- [x] Не вводить provider-specific codex footer до завершения shared prompt wave

### Approach
1) Обновить shared artifact/prompt policy: collect evidence требует `repo/path`, findings contract задаёт canonical validator verdict example.
2) Сохранить strict legacy reject в `artifactquality` и расширить diagnostics на citation-only semantic evidence.
3) Исправить `qwen` post-run classification без ослабления artifact validation.
4) Добавить `batch-owner.env` heartbeat и matrix stale sweep поверх `profile-status/*.json`.
5) Синхронизировать docs, прогнать DoD и затем повторить live `regres fast`.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/*`
- `internal/workspace/*`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/*`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Collect examples/prompts больше не показывают citation-only semantic evidence и явно требуют `repo/path`
- [ ] Findings prompts/policies требуют `version/run_id/generated_at` в `validator-verdict.json`
- [ ] `qwen` quota/auth post-run failures классифицируются как `runner_unavailable`
- [ ] Matrix умеет перевести stale `profile-status=running` в `failed/infra_incomplete_cycle` по owner sidecar
- [ ] DoD проходит и запускается новый live rerun

### Risks
- Главный риск — переусилить stale reconciliation и ложно добивать живой профиль. Снижение риска: reconciliation использует `batch-owner.env pid + updated_at` и оставляет свежий live heartbeat нетронутым.

### Progress log
- 2026-04-22: started residual wave for canonical semantic evidence `repo/path`, validator verdict metadata trio, qwen post-run reclassification, and batch-owner-based stale profile reconciliation.
- 2026-04-22: closed residual literal gaps after implementation audit: added strict `step_contract:null` reject coverage, explicit owner-gap `PASS` prompt/policy coverage, entrypoint hint tests for root `CODEOWNERS`/`MAINTAINERS*`, and docs wording that current runtime isolation relies on temp-root layout + step-local `cwd` because provider-side hard sandbox is unavailable.
- 2026-04-22: fixed an additional qwen runtime bug found during acceptance audit: stall monitors now terminate the stuck provider process, draft/collect post-artifact stalls recover through artifact-only validation/repair, and pre-artifact stalls get one fresh-process retry before final failure classification.
- 2026-04-22: fresh clean bank acceptance rerun surfaced one more qwen residual: the pre-artifact fresh-process retry could still run silently for too long after the initial stall trigger; fixed by keeping a bounded pre-artifact sentinel on the second attempt, adding explicit `retry exhausted` diagnostics, and extending regression coverage around retry stall windows.
- 2026-04-22: a follow-up clean bank rerun showed that bounded qwen retry exhaustion now behaves correctly but still surfaced one semantic misclassification: a fully silent second collect attempt was landing as `runtime_contract_failed`; fixed by reclassifying silent pre-artifact retry exhaustion to `runner_unavailable`, while preserving `runtime_contract_failed` for retries that still emit provider output before stalling.
- 2026-04-22: the next clean bank rerun immediately exposed the remaining edge-case of the same classifier: fully silent retry exhaustion could resurface as `post_artifact` once the fresh process wrote a partial artifact and then froze again; widened the reclassification so any fully silent collect retry exhaustion now lands as `runner_unavailable`, regardless of whether the second attempt died in `pre_artifact` or `post_artifact`.

### Plan ID
EP-20260422-docflow-runtime-residuals

### Context
После второго live `regres fast` collect перестал быть главным blocker, но surfaced новые residual drift classes: `step2.asis_docs` мог писать loose draft manifest и читать sibling baseline artifacts, `step3.findings` ломался на расходящемся `document_id` mapping и semantic alias duplicates, owner-gap оставался release-blocking даже без технических validator issues, а batch classifier путал terminal `validator verdict is FAIL` с `runtime_contract_failed`.

### Goals (must have)
- [x] Зафиксировать strict canonical contract для `step2.asis_docs`
- [x] Убрать sibling baseline leakage через step-local cwd + separated temp roots
- [x] Нормализовать staged docflow document IDs и semantic repo/entity aliases до validator
- [x] Перевести owner-only residual `FAIL` в non-blocking `PASS` без потери findings/questions
- [x] Классифицировать terminal `validator verdict is FAIL` как `runtime_flow_failed`
- [x] Синхронизировать README/spec/runbook/architecture с новым поведением
- [x] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить новый canonical live `regres fast` matrix и зафиксировать следующий triage cycle

### Non-goals
- [x] Не менять `schemas/*` и `internal/contracts/*`
- [x] Не добавлять новые verdict states или provider-specific compatibility repair
- [x] Не ослаблять validator или collect schemas под legacy semantic payloads

### Approach
1) Усилить shared prompt/policy/runtime-manifest validation для `step2.asis_docs`.
2) Перевести non-collect runtime cwd на step-local roots и развести headless/baseline temp roots.
3) Нормализовать `document_id` mapping, `evidence.repo` identity и semantic alias duplicates в docflow assembly/model layer.
4) Добавить owner-gap-only verdict reconciliation и скорректировать batch/report failure classification.
5) Синхронизировать docs, прогнать DoD и затем повторить canonical live `regres fast`.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/*`
- `internal/runtimedrafts/*`
- `internal/orchestrator/*`
- `internal/model/*`
- `internal/workspace/*`
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/PLANS.md`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

### Acceptance criteria
- [ ] `step2.asis_docs` принимает только canonical draft manifest с required outputs и без legacy top-level fields
- [ ] non-collect runtime шаги больше не стартуют из workspace root и не видят sibling baseline workspace как лёгкий template source
- [ ] staged `citation-index.json` и `final-run-index.json` используют согласованный deterministic `document_id` mapping
- [ ] semantic assembly нормализует repo aliases и детерминированно дедуплицирует entity aliases до validator/promotion
- [ ] owner-gap-only residual больше не держит `validator-verdict = FAIL`, но signal остаётся visible в findings/questions
- [ ] terminal `validator verdict is FAIL` классифицируется как `runtime_flow_failed`
- [ ] DoD проходит и следующий live `regres fast` cycle запускается

### Risks
- Главный риск — переусилить semantic dedupe и случайно слить разные сущности. Снижение риска: canonical key включает type + normalized repo identity + normalized name + primary evidence path, а alias remap остаётся детерминированным и тестируется на bank-style duplicates.

### Progress log
- 2026-04-22: started runtime/docflow residual wave for strict `step2` manifest contract, isolated non-collect cwd/layout, deterministic staged document IDs, semantic alias folding, owner-gap downgrade, and validator-fail classifier correction.

### Plan ID
EP-20260421-cleanup-owner-followups

### Context
Safe cleanup уже внедрён: убраны доказуемые code/doc хвосты, сокращены дубли и ослаблены repo-policy проверки под canonical-source model. Остались только owner-gated решения, которые нельзя принимать автоматически без риска снести intentional discoverability/history surfaces.

### Goals (must have)
- [ ] Подтвердить, должны ли `internal/docsync` и `internal/scriptsmeta` оставаться test-only пакетами под `internal/*`
- [ ] Подтвердить, нужно ли одновременно хранить `docs/archive/PLANS_ARCHIVE_2026-04.md` и `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`
- [ ] Не смешать follow-up owner decision с текущим cleanup change set

### Non-goals
- [ ] Не переносить `internal/docsync` и `internal/scriptsmeta` в этом проходе
- [ ] Не удалять `docs/archive/PLANS_SNAPSHOT_2026-04-21.md` без явного подтверждения владельца

### Approach
1) Зафиксировать спорные места как owner-gated follow-up после завершённого cleanup.
2) Не менять code/doc layout дополнительно, пока владелец не подтвердит intent.
3) После подтверждения сделать отдельный малый change set на package placement/archive retention.

### Files expected to change
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Спорные cleanup-решения зафиксированы отдельно от внедрённого safe cleanup
- [ ] Активная секция не держит уже закрытый audit-plan

### Risks
- Главный риск — ошибочно снести intentional discoverability/history surfaces. Снижение риска: держать эти вопросы owner-gated и не менять их без явного решения.

### Progress log
- 2026-04-21: cleanup внедрён; спорные решения по test-only package placement и archive retention вынесены в отдельный follow-up.

### Plan ID
EP-20260421-codex-runtime-provider

### Context
Нужно расширить MVP runtime/provider surface и trusted-host live release gate новым headless provider `codex-code`, сохранив deterministic `fake` baseline, default fallback `claude-code`, 5-profile taxonomy и shard-plan invariant между `baseline` и `parallel-default`.

### Goals (must have)
- [x] Добавить `codex-code` в runtime/config contracts, CLI/API/workspace validation и runner factory
- [x] Реализовать artifact-only headless runner через `codex exec` без codex-specific retry/watchdog policy
- [x] Расширить live batch harness/reporting/release verdict до `qwen-code + claude-code + codex-code`
- [x] Синхронизировать schema/spec/docs/examples/ADR и live runbook
- [x] Обновить Go и Python/script tests под новый provider

### Non-goals
- [x] Не добавлять hosted mode
- [x] Не менять default provider с `claude-code`
- [x] Не добавлять wrapper-скрипты поверх canonical matrix harness
- [x] Не добавлять wrapper-скрипты поверх canonical matrix harness

### Approach
1) Завершить runtime/config slice: provider enum/parser, runner factory, Codex runner, workspace/API validation и schema surface.
2) Расширить live harness/reporting/preflight и release strict gate без большого generic refactor.
3) Синхронизировать ADR/docs/examples/fixtures и пересчитать release catalog totals.
4) Прогнать целевые Go и Python/script tests, затем при возможности `make contracts`, `make test`, `make lint`, `make build`.

### Files expected to change
- `internal/runtime/*`
- `internal/workspace/*`
- `internal/api/*`
- `cmd/acp/*`
- `schemas/*`
- `scripts/full-run-batch-matrix.sh`
- `scripts/full-run-batch.sh`
- `scripts/frontend-live-e2e.sh`
- `scripts/write-batch-preflight.py`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/*`
- `examples/*`

### Acceptance criteria
- [x] `codex-code` accepted everywhere `claude-code|qwen-code` were previously required
- [x] Release-mode matrix expects all three providers and reports frontend/backend verdicts for Codex
- [x] Regression profiles remain qwen-only unless manually filtered otherwise
- [x] Docs/runbook/ADR/examples remain synchronized with schema and tests

### Risks
- Основной риск — оставить несовпадение между runtime surface и trusted-host live gate/reporting. Снижение риска: держать provider allowlists/order explicit, расширять report fields инкрементально (`frontend_codex_status`, `frontend_cancel_codex_status`, `runtimes.codex`) и проверять это script tests.

### Progress log
- 2026-04-21: started implementation; runtime provider enum/factory/Codex runner/tests partially wired, remaining work is harness/reporting/docs synchronization.
- 2026-04-21: runtime/config, harness/reporting, docs/examples/ADR and Go/Python test slices completed; `make contracts`, `make test`, `make lint`, `make build` passed on the implementation tree.
- 2026-04-21: post-implementation audit found residual runbook drift (`release` totals and frontend three-provider acceptance wording); synchronized the runbook and started trusted-host live preflight.
- 2026-04-21: repaired canonical `/tmp/provenarch-live-e2e` path checkouts to valid pinned-SHA git heads without changing curated matrices, created a detached clean verification worktree from the implementation snapshot, and launched manual `codex-code` regression smoke through `scripts/full-run-batch-matrix.sh`.
- 2026-04-21: reran DoD on an isolated git-backed snapshot of the current working tree and attempted canonical `release fast`; the new `codex-code` path advanced through `init.step1.collect` and materialized shard outputs inside the canonical harness, while overall release verification remained blocked by trusted-host provider issues outside this slice (`claude-code` returned quota/auth `403`, `qwen-code` failed live draft contract on `single-git_url`).

### Plan ID
EP-20260420-regres-small-live-triage

### Context
Нужно выполнить canonical live `regres fast` через `scripts/full-run-batch-matrix.sh` на trusted host, непрерывно мониторить статус прогонов и после завершения собрать детальный triage-отчёт по фактическим runtime/harness проблемам без изменения public contract.

### Goals (must have)
- [ ] Запустить canonical `regres fast` matrix slice из clean worktree с нужным provider filter
- [ ] Непрерывно мониторить matrix/batch progress до terminal state
- [ ] Собрать run artifacts, classifications и release/profile reports
- [ ] Зафиксировать продуктовые/runtime/harness баги и отделить их от operational noise
- [ ] Составить итоговый triage-отчёт: что сломалось, где именно, что нужно чинить дальше

### Non-goals
- [ ] Не менять matrix taxonomy, timeout profiles и public schemas во время самого прогона
- [ ] Не подменять canonical harness wrapper-скриптом или ad-hoc env overrides
- [ ] Не пытаться чинить live run до завершения triage цикла

### Approach
1) Проверить trusted-host prerequisites, pinned path checkouts и clean worktree.
2) Выполнить `regres fast` через `scripts/full-run-batch-matrix.sh` c canonical sweep set.
3) Во время выполнения регулярно читать batch/matrix logs, status files и intermediate reports.
4) После завершения разобрать terminal artifacts: `profile-runs.jsonl`, matrix status files, batch classifications, session summaries, release/profile verdicts и raw taskrun diagnostics.
5) Сформировать детальный triage-отчёт и список необходимых follow-up fixes.

### Files expected to change
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Matrix slice дошёл до terminal state (`passed|failed`) с сохранёнными batch/matrix breadcrumbs
- [ ] Есть собранный список фактических failure classes и их root cause
- [ ] Итоговый отчёт отделяет runtime/provider проблемы от harness/reporting проблем

### Risks
- Основной риск — long-running live provider hangs или внешние transport/API failures. Снижение риска: canonical watchdog/reporting уже встроены; мониторинг идёт по нескольким источникам (`driver log`, batch status, profile status, taskrun artifacts), а не по одному summary-файлу.

### Progress log
- 2026-04-20: prerequisites и pinned path SHA проверены, подготовлен trusted-host flow для live triage.

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
| 9 Q&A capability | done (beta boundary) | Workspace-backed QA service + read-only CLI `acp qa`; публичный endpoint остаётся follow-up |
| 10 Changelog compilers | done (beta baseline) | Iteration changelog materialization в `reports/changelog/*` |
| 11 `POST /api/qa/ask` | follow-up (post-beta) | Не входит в required beta surface |
| 12–13 | out of MVP | Вне текущего beta scope |
| 14 CI trigger mode | done (beta baseline) | CLI batch required, smoke/golden jobs без live network deps |
| 15 Domain/baseline pack hardening | done (beta baseline) | Baseline skills/prompts wired и versioned в workspace |
