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
