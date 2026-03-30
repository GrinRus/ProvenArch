# Appendix: Schemas (MVP v0)

Этот документ — companion к `schemas/taskresult.schema.json`.
Цель: дать человеко-читаемое описание контрактов без дублирования всей JSON Schema.

## 1) TaskResult JSON Schema

- **Source of truth:** `schemas/taskresult.schema.json`
- Контракт между **orchestrator** и **runtime** (MVP runtime: Claude Code headless).
- Orchestrator обязан валидировать TaskResult до применения изменений.

### Top-level поля
- required:
  - `meta`
  - `summary`
  - `changeset`
- optional:
  - `questions`
  - `coverage`
  - `warnings`
  - `debug`

### `meta` (минимум)
- required:
  - `task_id`
  - `step_id`
  - `runtime.name` (non-empty string)
  - `started_at`
- optional:
  - `runtime.version`
  - `finished_at`
  - `run_id`
  - `workspace`
  - `repo_scopes[]`

> Политика MVP: фактически используем `runtime.name = "claude-code"`, даже если схема допускает любое непустое значение.

### Changeset operations (MVP)
- `upsert_entity`
- `remove_entity`
- `upsert_edge`
- `remove_edge`
- `add_finding`
- `add_doc_artifact`
- `add_question`
- `set_coverage`

### Provenance и evidence
- `provenance.kind`: `observation | inference | assertion`
- `provenance.confidence`: `0..1`
- `provenance.evidence[]`:
  - `repo`
  - `path`
  - optional `ref`
  - optional `lines` в формате объекта `{ "start": <int>, "end": <int> }`

### Ограничение контракта
Операции `write_file(content)` в TaskResult нет.
Поэтому генерация `reports/as-is/*` и упаковка `proposals/*` в MVP реализуются orchestrator/compiler шагами (см. `docs/spec/PIPELINE_SPEC.md`).

---

## 2) Model conventions

- **Source of truth:** `docs/spec/MODEL_SPEC.md`
- Каноническая модель хранится как entity-per-file:
  - `model/entities/*`
  - `model/edges/*`

---

## 3) Charter и skills conventions

- **Source of truth:** `docs/spec/PIPELINE_SPEC.md`
- Charter хранится в `charter/`.
- Skills хранятся в `skills/` в версионируемом формате (manifest + prompts + templates).

---

## 4) Изменения схем/контрактов

Любые изменения в `schemas/` и контрактах сопровождаются:
- обновлением `docs/spec/*` и `docs/APPENDIX_SCHEMAS.md`,
- обновлением примеров/фикстур,
- кратким rationale в PR.
