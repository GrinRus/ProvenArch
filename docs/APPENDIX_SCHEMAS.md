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
- `runtime.profile.steps.*.provider` optional step-scoped provider override:
  - `step0_constitution|step1_collect|step2_as_is|step3_findings|step4_proposals`
  - allowed values: `claude-code|qwen-code`
- precedence:
  - timeouts: `env > workspace.yaml(runtime.profile.timeouts) > defaults`
  - execution: `CLI > env > workspace.yaml(runtime.profile.execution) > defaults`
  - step providers: `workspace step override > CLI/env global provider > claude-code`

Sharding policy в MVP:
- `heuristics` строит structural full-coverage partition repo без overlap в `path_scopes`.
- `semantic` больше не меняет shard boundaries и остаётся metadata-only surface поверх того же shard-plan.
- runtime execution всегда анализирует все repo scopes из workspace; frontend/backend filtering в execution contract отсутствует.

### Важное ограничение
`workspace.yaml` не конфигурирует workspace layout beyond repo sources и imports path.
Папки `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` фиксированы MVP convention.

## 2) TaskResult JSON Schema

- **Source of truth:** `schemas/taskresult.schema.json`
- Compatibility-контракт между **orchestrator** и **runtime** (MVP runtime: headless providers `claude-code|qwen-code` + fake baseline).
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

### Ограничение compatibility-контракта
`TaskResult` не поддерживает `write_file(content)` и не является primary writer surface.

Primary docs-first path:
- runtime пишет authored docs только в run-scoped `write_root`
- каноника живёт в staged/promoted doc set + `final-run-index` + `citation-index`
- `validator-verdict` является release gate

`TaskResult` в MVP:
- compatibility envelope для semantic guards/derived model/taskrun diagnostics
- `add_doc_artifact` остаётся metadata registration op без content payload
- не заменяет runtime-authored docs-first artifact pack
- per-step provider/runtime resolution не добавляется в `TaskResult` schema этого slice и фиксируется в run metadata / logs / runtime profile API

## 3) Shard Pack Manifest Schema

- **Source of truth:** `schemas/shard-pack-manifest.schema.json`
- Primary runtime output для `step1.collect`

Top-level required fields:
- `version`
- `run_id`
- `step_id`
- `shard_id`
- `agent_role`
- `artifact_root`
- `documents[]`
- `citations[]`
- `compatibility`

Semantic role:
- описывает authored shard docs внутри shard staging root
- связывает документы с canonical stable paths, topics и citation ids
- несёт compatibility snapshot для derived model layer

## 4) Final Run Index Schema

- **Source of truth:** `schemas/final-run-index.schema.json`
- Aggregator/orchestrator output для staged final set

Required fields:
- `version`
- `run_id`
- `pipeline`
- `generated_at`
- `citation_index_path`
- `canonical_documents[]`
- `topics[]`
- `compatibility`

Semantic role:
- canonical machine-readable index для UI/API/results surfaces
- перечисляет stable docs, staged paths, topics, citation bindings и source shards

## 5) Citation Index Schema

- **Source of truth:** `schemas/citation-index.schema.json`
- Required machine-readable evidence layer для docs-first pipeline

Required fields:
- `version`
- `run_id`
- `generated_at`
- `citations[]`

Semantic role:
- нормализует citation ids / claim ids / document ids
- даёт deterministic bridge между authored docs и evidence-backed claims

## 6) Validator Verdict Schema

- **Source of truth:** `schemas/validator-verdict.schema.json`
- Primary runtime output для `step3.findings` / validator phase

Required fields:
- `version`
- `run_id`
- `generated_at`
- `verdict`
- `checked_paths[]`

Allowed `verdict` values:
- `PASS`
- `FAIL`

Semantic role:
- canonical gate для promotion staged final set в стабильные `reports/*` и `proposals/*`
- validator может фиксировать только technical/index/reference issues, а не переписывать authored смысл документов

## 7) Runtime Draft Manifests (stage-only contract)

В этом slice runtime пишет staged draft manifests для agent-first шагов:
- `constitution-draft.json` (`step0.constitution`)
- `asis-draft-manifest.json` (`step2.asis_docs`)
- `proposals-draft-manifest.json` (`step4.proposals`)

Инварианты:
- runtime пишет manifests только в step `write_root`;
- почти финальные документы пишет только в `draft_final_root`;
- canonical workspace остаётся publish-only surface orchestrator/compiler/promoter;
- обязательный human gate на promotion отсутствует: publish происходит автоматически после schema/semantic/validator gates.

## 8) Model conventions

- **Source of truth:** `docs/spec/MODEL_SPEC.md`
- Каноническая модель хранится как entity-per-file:
  - `model/entities/*`
  - `model/edges/*`
- Stable ID patterns и normalization rules зафиксированы в `MODEL_SPEC`.
- Канонические patterns в MVP: `svc.<slug>`, `team.<slug>`, `repo.<slug>`, `ext.<slug>`, `db.<engine>.<slug>`, `api.http.<service-slug>.<method>.<path-slug>`, `api.grpc.<service-slug>.<service>.<method>`, `topic.<slug>`, `edge.<from>.<type>.<to>`.

## 9) Charter и skills conventions

- **Source of truth:** `docs/spec/PIPELINE_SPEC.md`
- Charter хранится в `charter/`.
- Cards `charter/cards/domains/*` и `charter/cards/teams/*` являются canonical human-owned source of truth; runtime pipeline не пишет в них напрямую.
- Skills хранятся в `skills/` в версионируемом формате (manifest + prompts + templates).

## 10) Изменения схем/контрактов

Любые изменения в `schemas/` и контрактах сопровождаются:
- обновлением `docs/spec/*` и `docs/APPENDIX_SCHEMAS.md`
- обновлением примеров/фикстур
- кратким rationale в PR
