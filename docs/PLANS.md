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
EP-20260416-live-e2e-release-hardening

### Context
Нужно перевести live E2E release plan из документа в реально исполнимый product/harness flow: release-mode matrix должен честно поддерживать per-run frontend smoke, headed Playwright, дополнительный parallel shard smoke, manual backend-only audit и auto-detection frontend repos без ручной `analysis.role` разметки.

### Goals (must have)
- [x] Добавить auto role inference для `repo_selection=backend_only`
- [x] Довести batch/matrix harness до promised release-mode frontend behavior (`per_run`, cancel mode, headed)
- [x] Обновить batch reporting для per-run frontend artifacts и backend-only audit evidence
- [x] Добавить regression tests и синхронизировать docs/runbooks

### Non-goals
- [x] Не менять public `schemas/taskresult.schema.json`
- [x] Не добавлять wrapper-скрипт поверх existing live harness

### Approach
1) Расширить workspace repo-selection на conservative source-based role inference.
2) Доработать `full-run-batch-5x2.sh` / `full-run-batch-matrix.sh` / `frontend-live-e2e.sh` под release-mode defaults и per-run frontend evidence.
3) Обновить `e2e_batch_report.py`, regression tests и docs/runbooks под новый live flow.

### Files expected to change
- `internal/workspace/*`
- `scripts/full-run-batch-5x2.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/frontend-live-e2e.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `docs/spec/WORKSPACE_SPEC.md`
- `docs/spec/API_SPEC.md`
- `docs/APPENDIX_SCHEMAS.md`

### Acceptance criteria
- [x] Тесты обновлены/добавлены
- [x] Схемы валидируются
- [x] Документация обновлена

### Risks
- Conservative repo-role inference может не покрыть все frontend naming conventions; unresolved repos всё ещё должны безопасно оставаться `unknown`, а не silently misclassify backend repos.

### Progress log
- 2026-04-16: добавлены repo role inference, release-mode frontend defaults/per-run evidence, batch report backend-only audit и docs sync.

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

### Plan ID
EP-20260416-release-runbook-ops-clarification

### Context
Нужно подготовить исполнимый release live E2E план под реальные trusted-machine ограничения: зафиксировать canonical wave targets, path checkout readiness, точные non-release команды и explicit triage policy для runtime incidents.

### Goals (must have)
- [x] Уточнить runbook на official wave1/wave2 как source of truth
- [x] Добавить preflight проверки path repos + pinned SHA
- [x] Зафиксировать команды для parallel smoke / forced-incomplete (без wrapper)
- [x] Добавить manual backend_only acceptance audit и common blockers

### Non-goals
- [x] Не менять matrix harness контракты
- [x] Не добавлять новые release scripts/wrappers

### Approach
1) Проанализировать текущий runbook vs фактический execution path.
2) Добавить операционные уточнения в `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.
3) Синхронизировать plan-log и подготовить run command package для trusted machine.

### Files expected to change
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`
- `AGENTS.md`
- `.agents/skills/e2e-live-gate/*`

### Acceptance criteria
- [x] Документация обновлена
- [x] План прогона воспроизводим end-to-end на trusted machine

### Risks
- Runbook intentionally опирается на canonical absolute paths для curated path profiles; среда без writable/access к этим путям остаётся operational blocker.

### Progress log
- 2026-04-16: добавлены canonical wave mapping, path SHA preflight, non-release command set, backend_only audit guidance и blocker triage.
- 2026-04-16: сокращён `AGENTS.md` до инвариантов, для live gate добавлен repo-local skill и закреплён запрет на подмену canonical matrix/curated files под неподходящий хост.

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
