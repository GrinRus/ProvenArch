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
  - optional `analysis.include[] | analysis.exclude[]`; shape остаётся string arrays, semantic
    validator компилирует единый slash-normalized dialect (`*` one segment, standalone `**`
    recursive, literal directory includes subtree) и fail-fast отклоняет ambiguous/invalid paths
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
- `runtime.profile.providers.<provider>` optionally persists provider-scoped `model` and
  `effort` for `claude-code`, `qwen-code`, or `codex-code`.
- each field resolves independently as `provider environment > workspace.yaml > provider-native default`;
  omitted values are not forwarded as CLI arguments. Runtime history snapshots effective
  values and their source (`env`, `workspace`, or `provider_default`) at run acceptance.
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
- `documents[]` и `citations[]` должны быть non-empty; каждый authored document обязан иметь
  non-empty `citation_ids[]`, а каждая citation — non-empty `claim_ids[]` и `document_ids[]`;
  document/citation bindings должны быть reciprocal: every cited citation lists the document and
  every citation `document_id` resolves back to a document that lists that citation
- несёт semantic snapshot для derived model layer
- schema-level rejection блокирует known legacy aliases: `covered_topics`, `question`, `relation`, `source`, `target`, finding `summary`/`inference`, `evidence_citation_ids`, top-level `step_contract`/`compatibility`
- validator-level path hygiene дополнительно reject-ит `documents[].path`, если он указывает на hidden/provider/tool side-effect directory (`.qwen/`, `.claude/`, `.codex/`, `.git/`, `node_modules/`), даже если файл физически существует под shard `write_root`
- runtime collect repair/recovery policy is behavioral, not a separate schema; current manifest-only repair rules, deterministic `collect_manifest_runtime_recovery`, and the narrowly allowlisted missing-`semantic.findings` shape recovery are specified in `docs/spec/PIPELINE_SPEC.md`. The latter inserts only an empty array when it is the sole defect, revalidates the complete artifact set and does not change the schema.

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
- semantic pair validation (without changing either JSON shape) requires global citation/claim
  uniqueness, reciprocal document bindings, matching run identity and concrete in-root repository
  evidence files; foreign-run or unresolved references are invalid
- даёт deterministic bridge между authored docs и evidence-backed claims

## 6) Validator Verdict Schema

- **Source of truth:** `schemas/validator-verdict.schema.json`
- Provider-free selected-run audit requires `checked_paths[]` to contain the exact current
  `final-run-index.json` and `citation-index.json` paths; duplicate or foreign paths fail closed.
- Primary runtime output для `step3.findings` / validator phase

Required fields:
- `version`
- `run_id`
- `generated_at`
- `verdict`
- `checked_paths[]`

Evidence locators with `lines`, `excerpt` or `excerpt_hash` use the shared bounded W24C validator:
1-based inclusive UTF-8 lines, CRLF/CR-to-LF normalization, exact whitespace/Unicode preservation,
no synthetic trailing LF, and SHA-256 over the selected bytes. Excerpt/hash fields require an
explicit line range; oversized, invalid-UTF-8 or out-of-range sources fail closed.

W24B admission keeps provider `fixed_paths` empty, rejects `PASS` with technical errors, requires
an effective `FAIL` to carry a technical error, and enforces unique deterministic issue identities
and selected-run document/citation/path references.

W24D adds strict semantic-envelope admission for new writes: object keys outside the documented
entity/edge/finding/question/provenance/evidence surface are rejected, conflicting IDs across shard
snapshots are rejected, and graph edge endpoints resolve to entities. Repeated shard observations
may reuse an ID only when core identity fields agree; provenance evidence remains mergeable.
Historical v1 reads remain compatible.

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
- `updated_at` (string metadata timestamp; ignored by orchestration decisions)
- `outputs[].kind`
- `outputs[].title`

Output mapping rules:
- `outputs[].path` must stay relative to `draft_final_root`
- `outputs[].canonical_path` must stay workspace-relative
- `outputs[].canonical_path` values must be unique when a step-specific publish surface is constrained

`asis-draft-manifest.json` specifics:
- normal step2 authoring writes all referenced Markdown first, verifies all three files are free of
  runtime/recovery narration, and writes the manifest last; this is content validation over the
  existing manifest shape, not an additional persisted field;
- the normal provider-independent prompt allows at most one bounded evidence read/list command,
  followed immediately by direct single-quoted heredoc writes. Inline language generators,
  templates and nested quote construction are prohibited until the complete write set exists;
  this changes execution guidance only and adds no schema field;
- `step_contract="as_is"`
- required canonical mappings:
  - `overview.md` -> `reports/as-is/overview.md`
  - `summary.md` -> `reports/coverage/summary.md`
  - `architect-summary.md` -> `reports/agent-outputs/architect/summary.md`
- additional outputs allowed only under `reports/as-is/<domain>/overview.md`
- canonical Architecture Home content requires eight exact standalone Markdown H2 lines defined by
  Pipeline Spec. The JSON manifest shape does not change. A deterministic recovery may only split
  an exact all-eight `H2 + inline authored body` form and must revalidate the complete draft set;
  it does not add sections or content.
- Architecture Home `repo:path` references are content-level evidence identities, not additional
  manifest fields. They must name an exact existing non-root file or directory; root shorthand and
  wildcard/glob syntax are invalid and are never expanded or normalized by the runtime.

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

## 8) Task/Attempt contracts

- **Source of truth:** `schemas/task.schema.json`, `schemas/attempt.schema.json` and `schemas/task-history.schema.json`.
- `Task` is durable user intent with a server-generated opaque `task_id`, monotonic revision,
  explicit repository/path scope, desired runner preset and explicit outcome/publication states.
- `Attempt` is an immutable admitted snapshot with its own opaque id, exact `task_id`/`run_id`
  linkage, parent retry lineage, immutable pipeline/idempotency key and request fingerprint,
  effective provider/model/permission/scope snapshot and terminal evidence summary.
- Admission records the client token against the canonical Task revision/options request. Repeating
  the same token returns the same Attempt/run identity; a different fingerprint is a typed conflict.
  The shared lease permits one active and one queued pipeline Attempt and never supersedes another
  Task's queued identity.
- `task-history.json` is a versioned registry. Its semantic validator checks unique identities,
  task-to-attempt membership, exact run/revision/status summary joins and same-task parent lineage.
- Historical pipeline runs are not synthesized as Tasks; legacy run identity remains outside this
  registry and is read-only until an explicit user action creates a Task.

## 9) Source Revisions Schema

- **Source of truth:** `schemas/source-revisions.schema.json`
- Per-run, pre-execution audit artifact at `reports/taskruns/<run_id>/source-revisions.json`.
- Records `run_id`, pipeline, capture time, nullable validated baseline, analysis-input fingerprint and per-repo configured source identity, current/baseline revisions, worktree state, effective include/exclude, comparison and typed fallback reasons.
- Absolute resolved checkout paths are not persisted. A configured absolute local path is represented by a stable redacted `external/<name>-<hash>` identity.
- Dirty/unavailable/non-ancestor states are valid conservative results, not parse failures.

## 10) Refresh Impact Plan Schema

- **Source of truth:** `schemas/refresh-impact-plan.schema.json`
- Pre-collect refresh audit artifact at `reports/taskruns/<run_id>/refresh-impact-plan.json`.
- `enforcement` is fixed to `advisory`; decisions are `unchanged_candidate`, `selective_candidate`, or `full_refresh_required`.
- `repo_deltas[]` preserve complete changed-file status including rename/copy original path, scope and mapped shards/domains. More than 10,000 changed paths records the exact count with `changes_complete=false` and never maps a partial list.
- Stale/preserved artifacts are candidates only; this schema does not authorize selective execution or promotion.

## 11) Refresh Execution Schema

- **Source of truth:** `schemas/refresh-execution.schema.json`
- Фиксирует фактический режим refresh (`no_op`, `affected_only`, `full`), исходное решение planner, source ranges, причины fallback и реально сохранённые/затронутые shards.
- `refresh-impact-plan.json` остаётся неизменяемым advisory input; execution audit не переписывает исходное решение planner.

## 12) Refresh Materialization Schema

- **Source of truth:** `schemas/refresh-materialization.schema.json`
- Фиксирует решения `updated`, `preserved`, `removed`, `uncertain`, baseline provenance и SHA-256 доступного содержимого.
- `uncertain` не разрешает selective promotion; полный объединённый staged set обязан пройти validator до атомарной promotion.

Selective shard reuse additionally requires orchestrator-owned
`reports/taskruns/<run_id>/staging/shards/<bounded-shard>/baseline-integrity.json`. This is an
internal persistence sidecar, not a provider/public schema: it records the full logical shard
identity, source ranges and sorted SHA-256/size inventory. Missing/legacy/invalid sidecars force
full refresh; they are never inferred from mutable canonical files. Its parser and fixtures live
with orchestrator tests so no public example or schema is added.

## 13) QA Answer Schema

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

## 14) Source QA Answer Schema

- **Source of truth:** `schemas/source-qa-answer.schema.json`
- Immutable provenance record inside an explicitly created Ask-to-Proposal package.

Required fields:
- `version` — integer `1`
- `source_run_id`, `answer_digest`, `proposal_title`, `question`
- `answer_generated_at`, `citations[]`, `unresolved[]`, `created_at`

Optional `operator_note` records the confirming operator context. The closed schema is validated
before atomic directory publication. It is a provenance record, not a replacement for the
immutable taskrun answer.

## 15) Model conventions

- **Source of truth:** `docs/spec/MODEL_SPEC.md`
- Каноническая модель хранится как entity-per-file:
  - `model/entities/*`
  - `model/edges/*`
- Stable ID patterns и normalization rules зафиксированы в `MODEL_SPEC`.
- Канонические patterns в MVP: `svc.<slug>`, `team.<slug>`, `repo.<slug>`, `ext.<slug>`, `db.<engine>.<slug>`, `api.http.<service-slug>.<method>.<path-slug>`, `api.grpc.<service-slug>.<service>.<method>`, `topic.<slug>`, `edge.<from>.<type>.<to>`.

## 16) Charter и skills conventions

- **Source of truth:** `docs/spec/PIPELINE_SPEC.md`
- Charter хранится в `charter/`.
- Cards `charter/cards/domains/*` и `charter/cards/teams/*` являются canonical human-owned source of truth; runtime pipeline не пишет в них напрямую.
- Skills хранятся в `skills/` в версионируемом формате (manifest + prompts + templates).

## 17) Изменения схем/контрактов

Public HTTP contracts, не представленные JSON Schema, фиксируются в
`docs/spec/API_SPEC.md` и проверяются handler tests + TypeScript response types. Для Epic 20
canonical fixture полного Git confirmation/read contract находится в
`fixtures/api/git-state-confirmation.json`, а user-facing example — в
`examples/git-state-confirmation.example.json`. Run coordination и persisted runtime identity
остаются additive API fields и не изменяют workspace artifact schemas. Epic 22F не меняет payload:
existing pipeline/QA and Git handlers now share a server admission lease; deterministic handler
tests prove `session_generation_changed`/`run_active` conflict behavior and unchanged Git HEAD.

Любые изменения в `schemas/` и контрактах сопровождаются:
- обновлением `docs/spec/*` и `docs/APPENDIX_SCHEMAS.md`
- обновлением примеров/фикстур
- кратким rationale в PR

Epic 22G changes no public schema. Provider validation diagnostics are normalized into an internal
typed issue set before recovery selection; Go incident fixtures are the compatibility surface.
Public terminal runner error codes remain unchanged.

Epic 22H likewise adds no persisted schema. `GET /api/pipeline/runs/<run_id>/audit` is a transient
read model specified in `docs/spec/API_SPEC.md` and checked by Go handler/package fixtures. Its
versioned, bounded JSON report is computed from existing final-index, citation-index and validator
contracts; it is never promoted or written into the workspace.

Epic 22I changes no product schema/API. It narrows provider process environment and live-harness
ownership: captured product history is copied byte-for-byte for inspection and never synthesized.

Epic 22J adds transient `state` to the existing Git diff HTTP read model; it is specified in
`docs/spec/API_SPEC.md` and does not alter a persisted workspace schema.

Epic 22K changes no API or persisted schema; immutable request keys and URL canonicalization are
client execution invariants covered by deterministic UI tests.

## 18) Current knowledge API read model

- **Source of truth:** `docs/spec/API_SPEC.md` (`GET /api/knowledge`) и Go response types в `internal/api/knowledge.go`.
- Это transient read model, а не новая persisted schema: `schemas/*` не меняются.
- `source_mode` всегда `promoted_current`; historical run snapshot, immutable QA answer/context
  (`qa_snapshot`) и QA diagnostics (`qa_audit`) являются отдельными authorities без fallback.
- `entities[]`/`edges[]` сохраняют canonical model fields и добавляют workspace-relative `path`.
- Только parsed fields задают topology; filename-derived IDs/edges запрещены.
- Malformed/unreadable/broken-reference файл фиксируется в typed `issues[]` и переводит общий `status` в `partial`, не скрывая валидные записи.
- Contract examples: `examples/knowledge-current-workspace.example.json` и `fixtures/api/knowledge-current-workspace.json`.

## 19) Architecture, progress and retry read models

- `GET /api/architecture`, run `progress/result/recovery/retry` and retry-plan responses are
  transient API read models documented in `docs/spec/API_SPEC.md`; canonical `schemas/*` remain
  unchanged.
- Architecture topology is projected only from validated entity/edge contracts and includes source
  paths/evidence, explicit export links and prior-promoted semantic comparison; Mermaid is
  inventory/export, never a parsed source of truth.
- Run-history additions are optional for backward compatibility. Progress uses known step/unit
  counters, persisted elapsed time and separate activity/useful-progress clocks. Retry lineage records
  immutable parent ID, requested/effective start and reused inputs; retry planning is admitted only
  for terminal `succeeded|failed|canceled` analysis runs, validates reusable shard contracts plus
  aggregated final/citation indexes and hashes every parent staging file.
  Architecture review and coverage are sourced from the immutable version-2 promoted snapshot
  manifest and retain explicit related IDs. The manifest version is an internal audit format, not a
  new canonical workspace model schema; `semantic_source_run_id` preserves no-op baseline lineage.
- Contract behavior is protected by Go API/orchestrator tests and TypeScript response types; unknown
  legacy fields remain safely absent rather than inferred.
- `review-summary.review` is an additive transient run-pinned read model. Its contract is documented
  in `docs/spec/API_SPEC.md`, represented by `fixtures/api/run-review-contract.json`, and covered by
  Go API fixture tests plus the TypeScript `RunReviewContract` type. It does not alter persisted
  `schemas/*` because the payload is derived from immutable run snapshots at request time.
- Combined public-shape example/fixture: `examples/outcome-workflow.example.json` and
  `fixtures/api/outcome-workflow.json`.
