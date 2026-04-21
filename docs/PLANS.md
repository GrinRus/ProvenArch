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
EP-20260421-repo-garbage-audit

### Context
Нужно провести двухэтапную ревизию репозитория на предмет лишнего/устаревшего кода и документации без поспешных выводов по неявно используемым entrypoints, fixtures, generated assets и runtime contracts.

### Goals (must have)
- [ ] Явно зафиксировать критерии "мусора" для кода и документации в контексте ACP
- [ ] Проверить репозиторий на лишние артефакты, дубли, dead code, docs drift и низкоценные комментарии
- [ ] Разделить находки на подтверждённые и требующие ручной верификации
- [ ] Сформировать приоритизированный отчёт с рисками изменения и рекомендациями по cleanup

### Non-goals
- [ ] Не удалять и не переписывать код в рамках самой ревизии
- [ ] Не объявлять неявно используемые entrypoints/fixtures/generated assets "мусором" без доказательств

### Approach
1) Прочитать канонические документы и зафиксировать критерии ревизии.
2) Проверить структуру репозитория, codepaths, docs surfaces, fixtures/examples/scripts и generated assets.
3) Для подозрительных мест собрать usage/context через symbol search, ripgrep и точечное чтение файлов.
4) Составить отчёт по четырём корзинам: точно лишнее, вероятно лишнее, устаревшие docs, быстрые улучшения.

### Files expected to change
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Есть явные критерии для code/docs garbage
- [ ] Каждая находка содержит путь, аргументацию, риск и рекомендованное действие
- [ ] Спорные места помечены как требующие подтверждения владельца/ручной проверки

### Risks
- Основной риск — принять за мусор generated/test/support assets или неявные runtime/CLI entrypoints. Снижение риска: проверять usage, учитывать контракты/fixtures и явно маркировать низкую уверенность.

### Progress log
- 2026-04-21: прочитаны README/ARCHITECTURE/PIPELINE_SPEC, зафиксирован план ревизии.

### Plan ID
EP-20260421-repo-cleanup-pr1-pr2

### Context
Нужно реализовать cleanup после ревизии мусора, не смешивая truth-sync документации и низкорискованный naming cleanup с owner-gated удалением скрытого `ai-advent` special-case в runtime scripts.

### Goals (must have)
- [x] Сжать duplicated runbook content в `README.md` и `docs/TESTING_STRATEGY.md`
- [x] Вернуть `docs/LOCAL_FULL_RUN_AI_ADVENT.md` к scenario/full-run scope без release cookbook
- [x] Убрать misleading legacy wording в canonical release examples и тестах без изменения поведения
- [x] Обновить docsync assertions под новую границу ответственности docs

### Non-goals
- [x] Не менять hidden `ai-advent` runtime behavior без owner confirmation
- [x] Не менять public API, schema contracts или `workspace.yaml`

### Approach
1) Сжать high-level docs и оставить подробный cookbook только в runbook'ах.
2) Обновить примеры, pointer-доки и docsync tests под новую docs boundary.
3) Переименовать legacy naming в тестах/сообщениях, не затрагивая runtime semantics.

### Files expected to change
- `README.md`
- `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
- `docs/TESTING_STRATEGY.md`
- `examples/e2e-matrix.release-fast.yaml`
- `examples/e2e-matrix.release-full.ftgo-sentry.yaml`
- `cmd/README.md`
- `internal/docsync/docsync_test.go`
- `scripts/tests/matrix_release_contract_test.py`
- `internal/orchestrator/coverage_merge_test.go`

### Acceptance criteria
- [x] README остаётся high-level entrypoint и не дублирует release cookbook
- [x] `docs/LOCAL_FULL_RUN_AI_ADVENT.md` больше не содержит release command blocks и release totals
- [x] `docs/TESTING_STRATEGY.md` фиксирует policy/invariants, а не operational cookbook
- [x] Docusync покрывает новую границу ответственности docs

### Risks
- Основной риск — случайно замаскировать действующее hidden `ai-advent` behavior как будто оно уже удалено. Снижение риска: не трогать scripts/runtime flow и оставлять release contract только в dedicated runbook.

### Progress log
- 2026-04-21: начата реализация PR1+PR2; PR3 явно отложен до owner confirmation.
- 2026-04-21: README/local runbook/testing strategy сжаты до truth-sync boundary; обновлены canonical release example comments, docsync assertions и low-risk legacy naming в тестах; verification: `go test ./internal/docsync`, targeted Python/Go tests, `make contracts`, `make test`, `make lint`, `make build`.
- 2026-04-21: `examples/e2e-matrix.regression-wave1.yaml` сознательно не менялся; файл остаётся deferred cleanup candidate до owner confirmation, как и hidden `ai-advent` script behavior.

2026-04-21: cleanup/refactor closure по prompt layering, compatibility inventory, explicit repair stages и ownership split завершён; active engineering surface больше не держит эти планы как open work. Зафиксированный результат: workspace prompt pack остаётся editable content layer only, merge order фиксирован как `provider header -> artifact-only/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer`, а enforced runtime policy/invariants не могут быть ослаблены содержимым prompt pack. Active compatibility inventory ограничен `collect.documents_path_normalization` и `drafts.reconcile_existing_canonical_outputs`. Текущий source of truth для результата — код, `docs/ARCHITECTURE.md` и `docs/TESTING_STRATEGY.md`.

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
- [x] Не переименовывать legacy `full-run-batch-5x2.sh`
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
- `scripts/full-run-batch-5x2.sh`
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
