# Appendix: Schemas (MVP v0)

Этот документ — companion к machine-readable схемам в `schemas/`.
Цель: дать человеко-читаемое описание контрактов без дублирования всей JSON Schema.

## 1) Workspace manifest schema

- **Source of truth:** `schemas/workspace.schema.json`
- Человеко-читаемая спецификация: `docs/spec/WORKSPACE_SPEC.md`

### Top-level shape
- required:
  - `version`
  - `repos`
- optional:
  - `docs`
  - `runtime`

### MVP constraints
- сейчас поддерживается только `version: 1`
- `repos[]` должен содержать как минимум одну запись
- каждая запись repo содержит:
  - required `name`
  - ровно одно из `path | git_url`
  - optional `ref`
  - optional `analysis.include[] | analysis.exclude[]` (glob overrides для shard planner)
  - optional `analysis.role` (`backend|frontend|mixed|unknown`)
- `repo.name` значения должны быть уникальными; это semantic validation rule workspace validator-а поверх JSON Schema
- `docs.imports_path` optional, default `./docs/imports`
- `runtime.profile.timeouts.*` optional persisted timeout profile:
  - `step_timeout_sec`, `heartbeat_sec`, `pipeline_timeout_sec`, `pipeline_kill_grace_sec`
  - `api_ready_timeout_sec`, `api_init_timeout_sec`, `ui_init_poll_timeout_sec`, `ui_cancel_poll_timeout_sec`
  - если поле задано, значение должно быть integer `> 0`
- `runtime.profile.execution.*` optional persisted execution profile:
  - `strategy: sequential|parallel`
  - `max_parallel_tasks > 0`
  - `failure_policy: fail_fast|best_effort`
  - `shard_discovery.mode: heuristics|semantic`
  - `repo_selection: all|backend_only`
- precedence:
  - timeouts: `env > workspace.yaml(runtime.profile.timeouts) > defaults`
  - execution: `CLI > env > workspace.yaml(runtime.profile.execution) > defaults`

`repo_selection` policy в MVP:
- `all`: включаются все repos.
- `backend_only`: исключаются только repos с `analysis.role=frontend`.
- `analysis.role=unknown` остаётся включённым и даёт warning `workspace.repo.selection.role_unknown`.

### Важное ограничение
`workspace.yaml` не конфигурирует workspace layout beyond repo sources и imports path.
Папки `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` фиксированы MVP convention.

## 2) TaskResult JSON Schema

- **Source of truth:** `schemas/taskresult.schema.json`
- Контракт между **orchestrator** и **runtime** (MVP runtime: headless providers `claude-code|qwen-code` + fake baseline).
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
  - `shard_id`
  - `repo_scope`
  - `repo_scopes[]`
  - `path_scopes[]`

> Политика MVP: `runtime.name` остаётся provider-aware (`claude-code` или `qwen-code` для headless, `claude-code` для fake baseline), при этом схема по-прежнему требует только непустую строку.
> `repo_scope` — primary repo context shard-а (удобен для prompt/diagnostics и обратной совместимости single-scope шагов).
> `repo_scopes[]` соответствует repo entries, заданным в `workspace.yaml`, и использует их `name`.
> `shard_id`/`path_scopes[]` используются runtime shard planner/scheduler для per-shard диагностики и воспроизводимости taskrun-артефактов.

### Changeset operations (MVP)
- `upsert_entity`
- `remove_entity`
- `upsert_edge`
- `remove_edge`
- `add_finding`
- `add_doc_artifact`

### Canonical MVP semantics
- runtime по умолчанию пишет `questions[]` и `coverage` на top-level
- `changeset[].op` поддерживает только canonical operations из schema
- legacy forms `add_question` и `set_coverage` отклоняются на contract validation

Conflict policy:
- duplicate questions dedupe по canonical `id` и нормализованному `text`
- `coverage.observed`, `coverage.missing`, `coverage.notes` canonicalize-ятся (snake/kebab/spaced variants) с дедупликацией по нормализованной форме

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
Дополнительно Step 2 materialize-ит `reports/diagrams/*` (C4 Mermaid set) как compiler outputs; это не расширяет `TaskResult` schema.

`add_doc_artifact` в MVP:
- metadata registration op
- не содержит content payload
- не инициирует свободную запись файлов runtime’ом
- может ссылаться только на уже существующий или позднее materialized orchestrator artifact
- не является required dependency path для обязательных outputs Step 1 или Step 3

Фиксация unknowns происходит через `questions`, `coverage` и subsequent findings, а не через свободную запись произвольных файлов runtime.

## 3) Model conventions

- **Source of truth:** `docs/spec/MODEL_SPEC.md`
- Каноническая модель хранится как entity-per-file:
  - `model/entities/*`
  - `model/edges/*`
- Stable ID patterns и normalization rules зафиксированы в `MODEL_SPEC`.
- Канонические patterns в MVP: `svc.<slug>`, `team.<slug>`, `repo.<slug>`, `ext.<slug>`, `db.<engine>.<slug>`, `api.http.<service-slug>.<method>.<path-slug>`, `api.grpc.<service-slug>.<service>.<method>`, `topic.<slug>`, `edge.<from>.<type>.<to>`.

## 4) Charter и skills conventions

- **Source of truth:** `docs/spec/PIPELINE_SPEC.md`
- Charter хранится в `charter/`.
- Cards `charter/cards/domains/*` и `charter/cards/teams/*` являются canonical human-owned source of truth.
- Skills хранятся в `skills/` в версионируемом формате (manifest + prompts + templates).

## 5) Изменения схем/контрактов

Любые изменения в `schemas/` и контрактах сопровождаются:
- обновлением `docs/spec/*` и `docs/APPENDIX_SCHEMAS.md`
- обновлением примеров/фикстур
- кратким rationale в PR
