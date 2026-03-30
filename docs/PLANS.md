# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.

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

## ExecPlan

### Plan ID
EP-20260330-baseline-v0-4-migration

### Context
Требуется логически перенести документы и контракты из исходной baseline-папки v0.4 в основной репозиторий как новую baseline-версию,
сохранив RU-first подачу в основных документах и не оставив дублирующий каталог.

### Goals (must have)
- [x] Перенести и синхронизировать документы baseline v0.4 в канонические пути репозитория.
- [x] Обновить TaskResult schema и пример под новую контрактную версию.
- [x] Заархивировать предыдущий `docs/STAKEHOLDER_DOC.md` v0.3 и заменить root-файл на v0.4.
- [x] Удалить исходный каталог baseline v0.4 и убрать все ссылки на него.

### Non-goals
- [x] Не добавлять код реализации продукта (`.go`, `.ts`, `.tsx`) в рамках этой миграции.
- [x] Не менять scope MVP (по-прежнему local-first + Claude Code headless).

### Approach
1) Синхронизировать core docs (`README`, `BACKLOG`, `BASELINE_POLICY`, `STAKEHOLDER_DOC`).
2) Синхронизировать контракты и specs (`schemas/taskresult.schema.json`, `examples/taskresult.example.json`, `docs/spec/*`, `docs/APPENDIX_SCHEMAS.md`).
3) Провести проверку ссылочной консистентности, затем удалить исходный каталог baseline v0.4.

### Files expected to change
- `docs/PLANS.md`
- `README.md`
- `docs/BACKLOG.md`
- `docs/BASELINE_POLICY.md`
- `docs/STAKEHOLDER_DOC.md`
- исторический `STAKEHOLDER_DOC` (предыдущая версия)
- `schemas/taskresult.schema.json`
- `examples/taskresult.example.json`
- `docs/spec/MODEL_SPEC.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/APPENDIX_SCHEMAS.md`

### Acceptance criteria
- [x] `rg -n "<baseline-source-folder>"` не находит совпадений.
- [x] `schemas/taskresult.schema.json` и `examples/taskresult.example.json` валидно парсятся.
- [x] `README`, `ARCHITECTURE`, `STAKEHOLDER_DOC`, `APPENDIX_SCHEMAS` не противоречат новой схеме.
- [x] `examples/workspace.example.yaml` сохраняет переносимый generic path формат.

### Risks
- Потеря согласованности между docs/spec и schema при неполном переносе семантики.
- Случайное сохранение ссылок на удаляемый каталог.

### Progress log
- 2026-03-30: ExecPlan добавлен, перенос baseline v0.4 выполнен.

---

### Plan ID
EP-20260330-variant2-workspace-docs-sync

### Context
Нужно зафиксировать в active docs единый формат хранения MVP: central `arch-workspace` (Variant 2),
вынести решение из appendix stakeholder-документа в основной поток и убрать альтернативные форматы из канонической документации.

### Goals (must have)
- [x] Обновить `docs/STAKEHOLDER_DOC.md` до v0.5 и перенести выбранный Variant 2 в основной раздел.
- [x] Удалить из активного stakeholder-дока развернутые альтернативы (Variant 1/3, Hybrid).
- [x] Синхронизировать `README`, `ARCHITECTURE`, `docs/spec/PIPELINE_SPEC.md`, `docs/spec/API_SPEC.md`, `BACKLOG`, `docs/BASELINE_POLICY.md`.
- [x] Сохранить неизменными схемы/контракты (`schemas/taskresult.schema.json` и API wire-shape).

### Non-goals
- [x] Не редактировать historical docs вне active baseline.
- [x] Не менять runtime scope MVP (по-прежнему только Claude Code headless).
- [x] Не вводить новые операции/поля в TaskResult schema.

### Approach
1) Обновить stakeholder-док: версия, перенос Variant 2 в основную часть, appendix без альтернатив.
2) Внести единообразные формулировки по central workspace в core-docs/spec/policy/backlog.
3) Прогнать grep-проверки на отсутствие альтернатив и на наличие canonical markers.

### Files expected to change
- `docs/PLANS.md`
- `docs/STAKEHOLDER_DOC.md`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/spec/API_SPEC.md`
- `docs/BACKLOG.md`
- `docs/BASELINE_POLICY.md`

### Acceptance criteria
- [x] `rg -n "Вариант 1|Вариант 3|Hybrid" docs/STAKEHOLDER_DOC.md README.md docs/ARCHITECTURE.md docs/BACKLOG.md docs/BASELINE_POLICY.md docs/spec/*.md` не находит совпадений в active docs.
- [x] `rg -n "arch-workspace|workspace.yaml|docs/imports" docs/STAKEHOLDER_DOC.md README.md docs/ARCHITECTURE.md docs/spec/*.md docs/BACKLOG.md` подтверждает канонические маркеры.
- [x] `rg -n "v0\\.4|v0\\.5" docs/STAKEHOLDER_DOC.md README.md` показывает `v0.5` в stakeholder и обновлённую ссылку в README.
- [x] Формулировки про central workspace (Variant 2) согласованы между stakeholder и инженерными документами.

### Risks
- Неполная синхронизация формулировок между stakeholder и инженерными docs.
- Случайные остатки альтернативных вариантов в active docs.

### Progress log
- 2026-03-30: Создан и реализован план синхронизации active docs под Variant 2.
- 2026-03-30: Acceptance checks выполнены (`rg`), противоречий не найдено.
