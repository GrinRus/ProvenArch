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
- `repo.name` значения должны быть уникальными; это semantic validation rule workspace validator-а поверх JSON Schema
- `docs.imports_path` optional, default `./docs/imports`
- `<docs.imports_path>/index.yaml` не входит в `workspace.yaml` schema; validator проверяет его warning-only как imports metadata artifact (`id`/`path` required)
- `runtime.profile.timeouts.*` optional persisted timeout profile:
  - `step_timeout_sec`, `heartbeat_sec`, `pipeline_timeout_sec`, `pipeline_kill_grace_sec`
  - `api_ready_timeout_sec`, `api_init_timeout_sec`, `ui_init_poll_timeout_sec`, `ui_cancel_poll_timeout_sec`
  - если поле задано, значение должно быть integer `> 0`
- `runtime.profile.execution.*` optional persisted execution profile:
  - `strategy: sequential|parallel`
  - `max_parallel_tasks > 0`
  - `failure_policy: fail_fast|best_effort`
  - `shard_discovery.mode: heuristics|semantic`
- `runtime.profile.permissions.*` optional persisted permission profile:
  - `mode: trusted_full_access|managed`
  - `approval_channel: fail_fast|ui`
  - default `trusted_full_access/fail_fast`, so existing live-provider behavior remains unchanged unless managed mode is explicitly selected
- `runtime.profile.steps.*.provider` optional step-scoped provider override:
  - `step0_constitution|step1_collect|step2_as_is|step3_findings|step4_proposals|qa`
  - allowed values: `claude-code|qwen-code|codex-code`
- precedence:
  - timeouts: `env > workspace.yaml(runtime.profile.timeouts) > defaults`
  - execution: `CLI > env > workspace.yaml(runtime.profile.execution) > defaults`
  - permissions: `workspace.yaml(runtime.profile.permissions) > defaults`
- step providers: `workspace step override > CLI/env global provider > claude-code`
- `analysis.role` удалён из active workspace contract; manifests с этим legacy полем reject-ятся schema validation-ом

Sharding policy в MVP:
- `heuristics` строит structural full-coverage partition repo без overlap в `path_scopes`.
- `semantic` больше не меняет shard boundaries и остаётся metadata-only surface поверх того же shard-plan.
- runtime execution всегда анализирует все repo scopes из workspace; frontend/backend filtering в execution contract отсутствует.

### Важное ограничение
`workspace.yaml` не конфигурирует workspace layout beyond repo sources и imports path.
Папки `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` фиксированы MVP convention.

## 2) Runtime execution metadata

- **Source of truth:** `internal/contracts/runtimeexecution.go`
- Internal execution metadata между **runtime** и **orchestrator**.
- Semantic success surface полностью artifact-only: stdout/stderr используются только для diagnostics/classification и raw-output forensics.

### Top-level поля
- required:
  - `version`
  - `task_id`
  - `run_id`
  - `step_id`
  - `provider`
  - `started_at`
  - `finished_at`
  - `status`
- optional:
  - `shard_id`
  - `domain_id`
  - `runtime_version`
  - `repo_scope`
  - `repo_scopes[]`
  - `path_scopes[]`
  - `artifact_root`
  - `write_root`
  - `draft_final_root`
  - `required_artifacts[]`
  - `warnings[]`
  - `raw_output_refs`

### Canonical MVP semantics
- provider success = process completed + required step artifacts passed contract validation
- semantic state не приходит через stdout/stderr или отдельный JSON envelope
- execution metadata используются для replay/recovery, diagnostics и linking to raw stdout/stderr artifacts
- `status` поддерживает только `succeeded | failed | canceled | timeout`

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
- `semantic`

Semantic role:
- описывает authored shard docs внутри shard staging root
- связывает документы с canonical stable paths, topics и citation ids
- несёт semantic snapshot для derived model layer
- schema-level rejection блокирует known legacy aliases: `covered_topics`, `question`, `relation`, `source`, `target`, finding `summary`/`inference`, `evidence_citation_ids`, top-level `step_contract`/`compatibility`
- validator-level path hygiene дополнительно reject-ит `documents[].path`, если он указывает на hidden/provider/tool side-effect directory (`.qwen/`, `.claude/`, `.codex/`, `.git/`, `node_modules/`), даже если файл физически существует под shard `write_root`
- runtime collect repair/recovery policy is behavioral, not a separate schema; current manifest-only command-first repair prompt rules and deterministic `collect_manifest_runtime_recovery` missing-manifest/fallback behavior are specified in `docs/spec/PIPELINE_SPEC.md`.

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
- `semantic`

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

Shared required fields:
- `version` — integer `1`
- `run_id`
- `step_id`
- `step_contract`
- `agent_role`
- `outputs[]`

Allowed common optional fields:
- `summary`
- `outputs[].kind`
- `outputs[].title`

Output mapping rules:
- `outputs[].path` must stay relative to `draft_final_root`
- `outputs[].canonical_path` must stay workspace-relative
- `outputs[].canonical_path` values must be unique when a step-specific publish surface is constrained

`asis-draft-manifest.json` specifics:
- `step_contract="as_is"`
- required canonical mappings:
  - `overview.md` -> `reports/as-is/overview.md`
  - `summary.md` -> `reports/coverage/summary.md`
  - `architect-summary.md` -> `reports/agent-outputs/architect/summary.md`
- additional outputs allowed only under `reports/as-is/<domain>/overview.md`

`proposals-draft-manifest.json` specifics:
- `step_contract="proposals"`
- allowed canonical publish surface only:
  - `proposals/*`
  - `reports/changelog/*`
- forbidden legacy top-level fields:
  - `pipeline`
  - `step`
  - `generated_at`
  - `domain_id`
  - `proposals`
  - `info_findings_noted`
  - `orphan_coverage_gaps`
- canonical example:

```json
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "summary": "Drafted remediation proposals from validated findings.",
  "outputs": [
    {
      "path": "proposal.md",
      "canonical_path": "proposals/proposal-baseline/proposal.md",
      "kind": "proposal",
      "title": "Baseline Remediation Proposal"
    },
    {
      "path": "changelog.md",
      "canonical_path": "reports/changelog/run-1.md",
      "kind": "changelog",
      "title": "Proposal Changelog"
    }
  ]
}
```

## 8) QA Answer Schema

- **Source of truth:** `schemas/qa-answer.schema.json`
- Primary runtime output для async Ask step `qa.ask`.

Required fields:
- `version` — integer `1`
- `run_id`
- `question`
- `answer`
- `citations[]`
- `unresolved[]`
- `confidence` (`0..1`)
- `provider`
- `generated_at`

Semantic role:
- structured answer contract consumed by `GET /api/qa/runs/<run_id>`;
- schema validates the answer shape; runtime validation also requires citations to point to workspace-relative paths from the generated context pack;
- file is written only under `reports/taskruns/<run_id>/qa/qa-answer.json`;
- it is run/audit output, not a promoted canonical architecture artifact.

## 9) Model conventions

- **Source of truth:** `docs/spec/MODEL_SPEC.md`
- Каноническая модель хранится как entity-per-file:
  - `model/entities/*`
  - `model/edges/*`
- Stable ID patterns и normalization rules зафиксированы в `MODEL_SPEC`.
- Канонические patterns в MVP: `svc.<slug>`, `team.<slug>`, `repo.<slug>`, `ext.<slug>`, `db.<engine>.<slug>`, `api.http.<service-slug>.<method>.<path-slug>`, `api.grpc.<service-slug>.<service>.<method>`, `topic.<slug>`, `edge.<from>.<type>.<to>`.

## 10) Charter и skills conventions

- **Source of truth:** `docs/spec/PIPELINE_SPEC.md`
- Charter хранится в `charter/`.
- Cards `charter/cards/domains/*` и `charter/cards/teams/*` являются canonical human-owned source of truth; runtime pipeline не пишет в них напрямую.
- Skills хранятся в `skills/` в версионируемом формате (manifest + prompts + templates).

## 11) Изменения схем/контрактов

Любые изменения в `schemas/` и контрактах сопровождаются:
- обновлением `docs/spec/*` и `docs/APPENDIX_SCHEMAS.md`
- обновлением примеров/фикстур
- кратким rationale в PR
