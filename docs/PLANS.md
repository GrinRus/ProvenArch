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
EP-20260421-flow-audit-followups

### Context
Полный аудит flow/project/tests/prompts/artifacts после closure step-scoped runtime parity показал, что core pipeline уже соответствует docs-first модели, но остаются четыре системных класса риска: prompt/source-of-truth drift между editable workspace prompt packs и enforced runtime policy, неполная модульная декомпозиция orchestrator/UI, transitional compatibility logic в runtime repair/canonicalization, и ограниченное интеграционное покрытие UI/harness относительно критичности live gate.

### Goals (must have)
- [ ] Зафиксировать явный layered contract между workspace prompt packs, enforced runtime step policy и provider-specific wrappers
- [ ] Довести orchestrator lifecycle до более жёстких ownership boundaries без скрытых cross-cutting side effects
- [ ] Сузить transitional compatibility/reconciliation logic до time-boxed, наблюдаемого слоя с понятным removal path
- [ ] Укрепить test surface там, где реальное поведение всё ещё опирается на крупные stateful компоненты и shell harness
- [ ] Сохранить текущие public contracts и release verdict semantics без расширения user-facing knobs

### Non-goals
- [ ] Не менять `workspace.yaml`, runtime profile API или release verdict/report contract
- [ ] Не добавлять новых live providers beyond `claude-code` / `qwen-code`
- [ ] Не превращать manual live matrix gate в required CI

### Approach
1) Вынести prompt composition contract в отдельный internal слой: editable workspace prompt packs становятся declarative content layer, а hardcoded Go policy остаётся explicit enforcement layer с проверяемым merge order.
2) Продолжить modular split orchestrator/UI: выделить runtime dispatch, compile/publish, run-registry/logging и semantic guards из `internal/orchestrator/*`; в UI вынести API/polling state в hooks/client layer, оставив `App.tsx` route shell.
3) Инвентаризировать transitional compatibility logic (`draft reconcile`, collect repair normalization), задать removal criteria и перевести каждое исключение в наблюдаемый compatibility registry/test matrix.
4) Усилить integration tests для active run lifecycle, git actions UI, shell harness interruption/recovery и prompt/source-of-truth parity между providers.

### Files expected to change
- `internal/orchestrator/*`
- `internal/runtime/*`
- `internal/runtimedrafts/*`
- `internal/api/*`
- `ui/src/*`
- `scripts/*`
- `scripts/tests/*`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Runtime prompt behavior имеет один documented source-of-truth layering и тест на composition order
- [ ] `internal/orchestrator/orchestrator.go` и `ui/src/App.tsx` заметно уменьшаются за счёт выделения lifecycle/state modules без functional drift
- [ ] Transitional compatibility paths имеют явный inventory, tests и removal notes вместо неявного накопления
- [ ] UI/harness integration coverage закрывает active running, interruption/recovery и git action flows
- [ ] DoD проходит: `make contracts`, `make test`, `make lint`, `make build`

### Risks
- Главный риск — сломать текущую стабильность ради structural cleanup. Снижение риска: делать только reviewable slices с parity tests и без изменения public contracts.

### Progress log
- 2026-04-21: архитектурный аудит завершён; выделены четыре follow-up класса риска: prompt layering, ownership boundaries, compatibility cleanup и integration coverage.

### Plan ID
EP-20260421-artifact-only-cleanup-followthrough

### Context
Нужно довести репозиторий до чистой docs-first artifact-only архитектуры без остаточного legacy-шума в active code paths, source-of-truth docs и harness/test assumptions. Historical материалы сохраняются, но выводятся из active surfaces и явно маркируются как архив.

### Goals (must have)
- [ ] Перевести `docs/PLANS.md` в active-only режим и убрать stale legacy markers из source-of-truth surfaces
- [ ] Добавить static/docsync guard, который не допускает возврат legacy markers вне allowlist архива и rejection fixtures
- [ ] Сделать validator-scope repair явной internal stage с наблюдаемыми логами и отдельными tests
- [ ] Вынести prompt composition в shared contract layer с provider parity test
- [ ] Сузить active compatibility inventory до двух safe repair rules без silent broad normalization

### Non-goals
- [ ] Не менять public schemas/API/runtime profile surface
- [ ] Не возвращать artifact pipeline к semantic stdout или runtime envelopes
- [ ] Не удалять historical evidence из `docs/archive/*` и negative regression fixtures

### Approach
1) Перенести исторический `docs/PLANS.md` snapshot в archive и пересобрать active-only плановый документ.
2) Добавить guard на legacy markers в active docs/code с allowlist только для `docs/archive/**` и rejection fixtures.
3) Вынести validator repair в explicit internal stage между validator verdict load и final staged validation.
4) Убрать provider-specific дубли prompt composition в shared builder и закрыть composition-order parity tests.
5) Закрепить compatibility inventory/tests и прогнать DoD.

### Files expected to change
- `docs/PLANS.md`
- `docs/archive/*`
- `internal/docsync/*`
- `internal/orchestrator/*`
- `internal/runtime/*`
- `internal/runtime/compatibilityregistry/*`
- `docs/ARCHITECTURE.md`

### Acceptance criteria
- [ ] В active docs/code больше нет legacy runtime markers удалённой wire-surface модели
- [ ] Validator repair больше не выглядит скрытой мутацией внутри validator apply path
- [ ] Provider prompt composition использует один shared order и parity test
- [ ] Compatibility registry и tests описывают только safe repair rules
- [ ] DoD проходит: `make contracts`, `make test`, `make lint`, `make build`

### Risks
- Главный риск — перепутать historical evidence с active source-of-truth и удалить полезный forensic context. Снижение риска: archive snapshot остаётся в репозитории, а active surfaces получают только текущие contracts и backlog.

### Progress log
- 2026-04-21: plan создан как implementation slice для parity cleanup, legacy cleanup и ownership refactor без смены публичных контрактов.

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
| 3 Runtime/orchestration seam | done (beta baseline) | Fake default + opt-in headless runtime selector with provider choice (`claude-code` default, `qwen-code`), persisted runtime execution metadata |
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
