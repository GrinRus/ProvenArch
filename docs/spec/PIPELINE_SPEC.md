# Спецификация пайплайна (MVP v0)

Документ описывает pipeline ACP через input/output контракты и expected artifacts.

## Общие понятия

- **Workspace**: единый central git-репозиторий `arch-workspace/` (каноническая MVP-конвенция, Variant 2) с `workspace.yaml`, `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/`.
- **Workspace manifest**: `workspace.yaml`, валидируемый по `schemas/workspace.schema.json` и описанный в `docs/spec/WORKSPACE_SPEC.md`.
- **Orchestrator**: управляет шагами, готовит PromptPack/ContextPack, выдаёт runtime staged-write envelope, валидирует manifests/indexes/verdicts и persisted runtime execution metadata.
- **Runtime (MVP)**: headless multi-provider (`claude-code` default, `qwen-code` optional, `codex-code` release peer) + deterministic fake harness (default for required CI/testing).
- **Runtime execution metadata**: internal metadata artifact runtime-шага (`task_id`, `run_id`, `step_id`, `provider`, write roots, status, raw output refs), не semantic source of truth для live docs-first flows.

## Docs-first runtime contract

Primary runtime outputs для live pipeline:
- `schemas/shard-pack-manifest.schema.json`
- `schemas/final-run-index.schema.json`
- `schemas/citation-index.schema.json`
- `schemas/validator-verdict.schema.json`

Staged write model:
- runtime шаги пишут только в step-local `write_root` и `draft_final_root`
- shard analysts пишут только в `reports/taskruns/<run_id>/staging/shards/<shard_id>/`
- aggregator/orchestrator materialize-ит staged final set в `reports/taskruns/<run_id>/staging/final/`
- validator пишет только в `reports/taskruns/<run_id>/validator/`
- promotion копирует только approved final set в стабильные `reports/*` и `proposals/*`

Runtime write policy:
- `workspace root` больше не трактуется как implicit write target
- runtime получает explicit `artifact_root`, `write_root`, `draft_final_root`, `read_context_roots[]`, `step_contract`, `expected_artifacts[]`
- runtime не имеет права писать в `workspace.yaml`, `schemas/*`, `docs/spec/*`, `charter/*` и анализируемые user repos

> MVP policy фиксирует step-scoped runtime provider contract: effective provider для шага выбирается как `workspace step override > CLI/env global provider > claude-code`; semantic stdout payloads не поддерживаются.
> CLI/process runtime mode задаётся флагом `--runtime fake|headless` (`fake` default, `headless` opt-in), global fallback provider — `--runtime-provider claude-code|qwen-code|codex-code` (env fallback `ACP_RUNTIME_PROVIDER`).

## Repo source manifest (MVP)

В `workspace.yaml`:
- `version` обязателен и сейчас поддерживает только `1`
- `repos[]` обязателен
- `docs.imports_path` optional, default `./docs/imports`

В `repos[]` каждая запись содержит:
- `name`
- ровно одно из:
  - `path`
  - `git_url`
- optional `ref`

MVP policy:
- `path` используется для already-cloned локальных репозиториев;
- `git_url` допускает GitHub/GitLab-style sources, которые clone/fetch-ятся на той же машине через локальный `git` и текущий user/runner auth context;
- ACP в MVP не хранит отдельный credential store и не реализует собственный git access plane;
- имена репозиториев в одном workspace должны быть уникальными;
- layout `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` не конфигурируется через manifest и считается fixed MVP convention;
- GitHub/GitLab hooks и manual pipeline button/job должны в итоге запускать тот же batch mode и те же step IDs.

## Charter, Cards и Skills (MVP format)

### Charter
Хранится в `charter/`, минимально:
- `charter/overview.md`
- `charter/glossary.yaml`
- `charter/nfr.yaml`
- `charter/rules.yaml`
- `charter/templates/`
- `charter/cards/domains/<domain-id>.md`
- `charter/cards/teams/<team-id>.md`

### Cards ownership model
- Step 0 wizard создаёт initial canonical `charter/cards/domains/*` и `charter/cards/teams/*`.
- Эти cards являются human-owned source of truth для domain/team IDs.
- Runtime pipeline не пишет в canonical `charter/cards/*`.
- Step 1 не создаёт и не переименовывает canonical domain/team cards автоматически.
- Если runtime обнаруживает новый домен, новую команду или неразрешимый owner gap, он создаёт `question` и/или `finding`, а не materialize-ит новый canonical card.
- `owner_team_id` в model должен ссылаться на существующий `team.<slug>`. Неизвестный owner фиксируется как unknown/question.

### Skills
Хранятся в `skills/` и редактируются в UI:

```text
skills/subagents.yaml
skills/<skill_name>/
  manifest.yaml
  prompts/
    system.md
    task.md
  templates/
    adr.md
    rfc.md
```

`manifest.yaml` (minimum):
- `name`
- `version`
- `applies_to`
- `inputs`
- `outputs`

`skills/subagents.yaml` в MVP обязателен и задаёт baseline agent roles и привязку skills/prompt packs.

## Baseline Agents/Skills/Prompts (MVP)

Обязательный baseline bundle:
- agents:
  - `domain-analyst`
  - `architect-aggregator`
  - `system-analyst-qa`
- skills:
  - `service-inventory`
  - `interface-extraction`
  - `integration-mapping`
  - `datastore-mapping`
  - `cicd-mapping`
  - `ownership-coverage`
  - `findings`
  - `proposals`
- prompt packs:
  - `constitution`
  - `collect-context`
  - `findings`
  - `proposals`
  - `qa`

Bundle поставляется вместе с продуктом, хранится в workspace и может редактироваться пользователем через UI/git workflow.

## Docs imports metadata (MVP)

Для imported docs используется metadata index:
- `docs/imports/index.yaml`

Минимальные поля записи:
- `id`
- `title`
- `source_kind`
- `path`
- `checksum`
- `imported_at`
- optional `source_url`
- optional `source_updated_at`
- optional `status`
- optional `tags`

## Canonical step IDs

### Init pipeline
- `init.step0.constitution`
- `init.step1.collect`
- `init.step2.asis_docs`
- `init.step3.findings`
- `init.step4.proposals`

### Refresh pipeline (manual)
- `refresh.step1.collect`
- `refresh.step2.asis_docs`
- `refresh.step3.findings`
- `refresh.step4.proposals`

## Runtime execution semantics

Canonical MVP runtime shape:
- semantic source of truth для `step1` — `shard-pack-manifest.json.semantic`
- `step1` semantic provenance evidence (`entities/edges/findings[].provenance.evidence[]`) обязаны содержать non-empty `repo` + `path`; citation-only semantic evidence objects невалидны
- semantic source of truth для `step3` — `validator-verdict.json.findings[]` и `validator-verdict.json.questions[]`
- runtime execution metadata сохраняют только execution context, status, warnings и raw-output references

## Init pipeline

### Step 0 — Constitution (human-in-the-loop)
Inputs:
- шаблоны charter
- baseline prompt pack `constitution`
- пользовательские правки в UI
- persisted structured wizard contract `charter/wizard/step0-contract.json` (optional)

Outputs:
- обновлённые `charter/*`
- initial canonical `charter/cards/domains/*`
- initial canonical `charter/cards/teams/*`
- runtime draft manifest `constitution-draft.json` + optional draft finals в `draft_final_root`

Step 0 materialization policy:
- если `charter/wizard/step0-contract.json` валиден, его поля детерминированно влияют на `charter/*` и canonical cards;
- если contract отсутствует/невалиден, применяется baseline fallback materialization;
- fallback фиксируется warning-сообщением в run diagnostics (`GET /api/pipeline/runs/<run_id>.warnings`).
- runtime draft не пишет canonical `charter/*` напрямую; compile/materialization layer остаётся единственной publish surface.
- support-only baseline bundle seeding не пишет canonical `skills/subagents.yaml`; этот файл публикуется только из validated `constitution-draft.json`.

### Step 1 — Collect context (runtime step)
Inputs:
- `workspace.yaml` из корня central `arch-workspace`
- локальные checkout репозиториев, полученные из `path` и/или local git resolution of `git_url` на той же машине
- `docs/imports/index.yaml` (если есть) + `docs/imports/*`
- `docs/imports/*`, `docs/rfcs/*`, `docs/meetings/*`, `docs/decisions/*`
- `charter/*`
- `skills/*`

Runtime focuses on:
- arbitrary stacks через выбранный headless provider (`claude-code|qwen-code|codex-code`) + baseline skill/prompt bundle, без фиксированного whitelist parser implementations в MVP
- workspace prompt packs участвуют в composed prompt как editable content layer; merge order фиксирован: provider header -> artifact-only/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer. Enforced runtime policy/invariants задаются internal shared step-policy слоем и не могут быть ослаблены содержимым prompt pack
- service topology и entrypoints
- interfaces (HTTP/gRPC/events)
- external systems/integrations
- datastores и storage usage
- CI/CD evidence (`.gitlab-ci.yml`, Dockerfile, deploy manifests, helm/k8s, scripts)
- ownership hints и явные unknowns

Primary runtime output:
- authored shard docs inside shard `write_root`
- `shard-pack-manifest.json`
- optional shard-local draft finals рядом с manifest, но только внутри shard staging root

Orchestrator applies:
- валидирует `shard-pack-manifest.json`
- требует полного `semantic` block в `shard-pack-manifest.json`: `coverage`, `questions`, `entities`, `edges`, `findings` обязательны, а `questions/entities/edges/findings` materialize-ятся массивами даже when empty
- collect manifest принимает только canonical vocabulary: `semantic.coverage.observed`, `semantic.questions[*].text`, `semantic.edges[*].type`, object-shaped `provenance`, numeric `confidence`; aliases вроде `covered_topics`, `question`, `relation`, array provenance, string confidence, `evidence_citation_ids`, top-level `step_contract` и `compatibility` считаются contract-invalid legacy drift
- требует global uniqueness для `citations[].claim_ids` в assembled staged final set; provider должен формировать `claim_id` как semantic stem + shard slug и добавлять deterministic numeric suffix при остаточной коллизии
- collect runtime до финальной validation может выполнять только explicit repair rules для path normalization и draft-root reconcile; semantic stdout repair отсутствует
- для `step1.collect` runtime использует только selected repo roots, `write_root` и explicit `read_context_roots`; `reports/taskruns/**`, raw logs и archive docs не считаются schema source of truth и не должны использоваться как manifest examples
- выполняет runtime `init.step1.collect`/`refresh.step1.collect` отдельно для каждой canonical domain card (`charter/cards/domains/*`)
- materialize-ит отдельный `runtime-execution.json` на каждый shard/domain under `reports/taskruns/<run_id>/...`
- для sharded runtime ведёт shard-summary state machine `pending | checkpointed | succeeded | failed`; raw per-shard taskrun materialize-ится до `apply`, чтобы restart recovery мог replay-ить shard из persisted artifact
- internal shard-summary contract: `taskrun_path` обязателен для `checkpointed/succeeded` item и должен ссылаться на persisted `runtime-execution.json`
- materialize-ит raw runtime execution metadata и shard summaries в `reports/taskruns/*`
- извлекает semantic snapshot из manifests/verdicts для derived `model/*`/semantic guards
- обновляет derived `model/*`
- enrich существующие `charter/cards/domains/*` и `charter/cards/teams/*` через детерминированную секцию `## Derived (ACP Step1)`:
  - related model IDs / findings / questions
  - coverage missing summary
  - evidence refs (для domain и team cards)
- не создаёт и не переименовывает canonical cards автоматически
- сохраняет outputs domain-агентов в `reports/agent-outputs/domains/*`
- не изменяет canonical `charter/cards/*` напрямую; допускается только controlled enrichment секции `## Derived (ACP Step1)`

### Step 2 — As-is docs (aggregator staged assembly)
Inputs:
- collected shard manifests + authored shard docs
- `charter/*`
- `skills/templates/*`

Outputs:
- staged final docs в `reports/taskruns/<run_id>/staging/final/`
- `final-run-index.json`
- `citation-index.json`
- staged derived `model/*` для diagram builders и deterministic projections
- если live runtime step выполняется, его primary output — `asis-draft-manifest.json` + optional draft docs внутри `draft_final_root`

Step 2 policy:
- domain outputs и финальные analysis/proposal surfaces собираются как authored/staged docs-first set из shard packs
- orchestrator compiler допускается только как deterministic renderer/materializer для technical derived surfaces
- `model/*` и диаграммы остаются derived layer
- `asis-draft-manifest.json` обязан использовать strict canonical contract: `version=1`, `run_id`, `step_id`, `step_contract="as_is"`, `agent_role`, `outputs[]`
- для `step2.asis_docs` обязательны canonical publish mappings `reports/as-is/overview.md`, `reports/coverage/summary.md`, `reports/agent-outputs/architect/summary.md`; дополнительные outputs разрешены только под `reports/as-is/<domain>/overview.md`
- runtime draft parsing для `step2` strict: unknown top-level fields и legacy envelopes (`repo_scopes`, `compatibility`, `step_contract=null`, неканонические coverage outputs) reject-ятся до publish
- `final-run-index.json` и `citation-index.json` используют один deterministic `document_id` mapping, который наследует `manifest.Documents[*].id`; provider-authored `document_ids` remap-ятся в этот canonical namespace до validator
- staged semantic assembly нормализует `evidence.repo` к логическому repo scope, сводит generated checkout-dir aliases и дедуплицирует entity aliases/related references до validator
- если evidence incomplete, staged reports materialize-ятся с incomplete banner, но не promote-ятся без validator `PASS`
- если collect status = `unusable`, live runtime для `step2.asis_docs`, `step3.findings` и `step4.proposals` не запускается; orchestrator детерминированно пересобирает triage-only staged docflow только из persisted collect artifacts, а terminal/root cause остаётся collect failure
- non-collect runtime шаги не стартуют из workspace root: `step2` использует `draft_final_root` как cwd, а live harness разводит headless и baseline workspaces по разным temp roots, чтобы sibling baseline artifacts не были implicit template source
- provider-side hard sandbox в текущих headless CLI отсутствует; practical isolation для runtime обеспечивается только через separated temp roots и step-local `cwd`

### Step 3 — Findings / Validation (runtime step)
Inputs:
- staged final docs
- `final-run-index.json`
- `citation-index.json`
- `charter/rules.yaml`
- `skills/*`

Primary runtime output:
- `validator-verdict.json`
- `validator-verdict.json` обязан содержать canonical metadata trio `version=1`, `run_id`, `generated_at` вместе с `verdict` и `checked_paths`
- validator findings сохраняют canonical `title + description + provenance`; observation evidence внутри `findings[*].provenance.evidence[]` обязано содержать `repo/path`

Orchestrator applies:
- для sharded runtime replay-ит persisted `succeeded/checkpointed` shard taskruns без повторного provider execution; `checkpointed` shard повторно `apply`-ится, `succeeded` shard только восстанавливает orchestrator in-memory state
- пересобирает staged final set после validator verdict
- валидирует `validator-verdict.json`
- блокирует promotion при verdict != `PASS` или при broken staged indexes
- validator может править только index/reference/technical issues внутри validator scope; смысл authored docs не переписывается wholesale
- duplicate `claim_id` внутри `citation-index.json` считаются validator-scope technical drift: orchestrator repair-ит поздние коллизии по правилу `<claim_id>.<shard_slug>[.<n>]` и фиксирует это в `validator-verdict.json.fixed_paths`
- обновляет `reports/findings/*`
- обновляет `reports/agent-outputs/architect/summary.md` через детерминированную агрегацию фактических domain outputs
- materializes critical unknowns как findings, если отсутствуют owner/integration/database/CI-CD evidence
- owner-gap остаётся surfaced через `coverage/findings/questions`, но owner-only residual без технических validator issues не должен сам по себе держать `validator-verdict = FAIL`; после repair stages orchestrator может детерминированно reconcile-ить такой verdict в `PASS`, не скрывая сам gap

### Step 4 — Promotion / Proposals
Inputs:
- staged final doc set
- `final-run-index.json`
- `citation-index.json`
- `validator-verdict.json`
- `skills/templates/*`

Outputs:
- `proposals/<proposal-id>/proposal.md`
- `proposals/<proposal-id>/ADR.md`
- `proposals/<proposal-id>/RFC.md`
- `proposals/<proposal-id>/migration-checklist.md`

> В MVP proposals формируются только через runtime-authored staged docs; compiler допускается только как deterministic renderer/materializer для derived technical outputs.

## Iteration changelog (MVP)
- На каждую итерацию orchestrator формирует:
  - `reports/changelog/<yyyy-mm-dd>-<iteration-id>.md`
- Changelog агрегирует изменения по model/findings/proposals/agent-outputs/coverage.

## Missing information handling (MVP)
- Runtime не должен выдумывать отсутствующие данные.
- Если evidence недостаточно для архитектуры, интеграции, базы данных, owner linkage или CI/CD:
  - добавляются `questions`,
  - заполняется `coverage.missing` / `coverage.notes`,
  - при необходимости создаются findings про критичные пробелы.
- Observation без evidence не должен порождаться.
- Unknown owner не создаёт auto-team entity и не создаёт auto-card; это question/finding path.

## On-demand Q&A capability (MVP)
- System Analyst Q&A Agent работает поверх:
  - `charter/cards/*`
  - `model/*`
  - `reports/*`
  - `docs/imports/*`
- В текущем beta surface доступен internal service слой + CLI `acp qa` (read-only).
- Follow-up API endpoint: read-only `POST /api/qa/ask`.
- Planned response shape: `answer`, `citations`, `unresolved`, `confidence`.
- Канонический stakeholder статус/границы по Q&A API фиксируются в `docs/STAKEHOLDER_DOC.md` (Canonical Stakeholder Matrix).

## Нефункциональные требования (MVP)

- детерминированность на одинаковых входах
- безопасный filesystem scope (без выхода за workspace root)
- runtime предлагает, человек подтверждает спорные решения
- запись только в workspace, не в пользовательские репозитории
- git access использует локальный `git` контекст пользователя/runner, а не отдельный credential plane ACP
- один и тот же pipeline должен быть воспроизводим локально и в GitHub/GitLab CI/CD trigger mode

## CI/CD trigger modes (MVP)
- Required MVP integration surface: `acp run --workspace ... --pipeline ... --non-interactive`.
- SCM hook mode: GitHub/GitLab webhook инициирует native pipeline/job, который запускает ACP batch mode.
- Default auto-trigger: `push` в default branch.
- `merge request` / `pull request` updates в MVP идут как manual/preview trigger, а не auto-write trigger.
- Manual trigger mode: пользователь запускает ту же job через manual pipeline button/job.
- Long-running standalone mode: при наличии поднятого ACP service внутренняя automation может вызывать тот же refresh flow через API/CLI без hosted control plane.
- Internal API trigger optional и допустим только для trusted local/private deployment.
- Debounce policy: одновременно активен только один run на workspace; события в окне 5 минут схлопываются, policy `last event wins`.
### Step 2 — As-Is docs (agent-first + compiler)

Inputs:
- persisted Step 1 shard manifests / authored docs
- semantic snapshot из collect artifacts

Primary runtime output:
- `asis-draft-manifest.json`
- optional draft docs inside `draft_final_root`

Publish policy:
- orchestrator/compiler нормализует layout, coverage/findings links и ordering;
- runtime draft docs не считаются опубликованным результатом до compile/publish stage;
- strict `asis-draft-manifest.json` contract обязателен для live runtime path: `step_contract="as_is"`, required overview/coverage/architect outputs и отсутствие legacy top-level fields;
- при resume после более позднего шага orchestrator может детерминированно пересобрать staged final docflow из persisted collect artifacts без live provider rerun.

### Step 3 — Findings / Validator

Primary runtime output:
- `validator-verdict.json`

Publish policy:
- validator остаётся единственным schema/semantic gate для staged final set;
- richer synthesis/ranking/evidence shaping разрешены, но итог проходит через validator verdict и compile checks.
- terminal `validator verdict is FAIL` трактуется как completed runtime flow failure, а не как draft/runtime-contract failure; owner-gap-only verdict после technical repairs может быть downgraded в `PASS` при сохранении findings/questions.

### Step 4 — Proposals (agent-first + auto-publish)

Primary runtime output:
- `proposals-draft-manifest.json`
- optional proposal/changelog drafts inside `draft_final_root`

Publish policy:
- deterministic promoter проверяет schema/semantic/validator gates;
- обязательного human approve нет;
- canonical `proposals/*` и `reports/changelog/*` публикуются автоматически только после successful gates.
