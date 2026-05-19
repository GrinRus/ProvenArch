# Спецификация пайплайна (MVP v0)

Документ описывает pipeline ACP через input/output контракты и expected artifacts.

## Общие понятия

- **Workspace**: единый central git-репозиторий `arch-workspace/` (каноническая MVP-конвенция, Variant 2) с `workspace.yaml`, `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/`.
- **Workspace manifest**: `workspace.yaml`, валидируемый по `schemas/workspace.schema.json` и описанный в `docs/spec/WORKSPACE_SPEC.md`.
- **Orchestrator**: управляет шагами, готовит PromptPack/ContextPack, выдаёт runtime staged-write envelope, валидирует manifests/indexes/verdicts и persisted runtime execution metadata.
- **Runtime (MVP)**: headless multi-provider (`claude-code` default, `qwen-code` optional, `codex-code` release peer) + deterministic fake harness (default for required CI/testing).
- **Runtime execution metadata**: internal metadata artifact runtime-шага (`task_id`, `run_id`, `step_id`, `provider`, write roots, status, raw output refs), не semantic source of truth для live docs-first flows.

## Docs-first runtime contract

Docs-first contract schemas для live pipeline:
- `schemas/shard-pack-manifest.schema.json`
- `schemas/final-run-index.schema.json`
- `schemas/citation-index.schema.json`
- `schemas/validator-verdict.schema.json`

Artifact ownership:
- provider-authored runtime artifacts: `shard-pack-manifest.json`, runtime draft manifests/files и `validator-verdict.json`
- orchestrator-authored staged artifacts: `final-run-index.json`, `citation-index.json`, shard plans/summaries и run logs/history
- compiler-derived promoted artifacts: `model/*`, diagrams, normalized coverage/findings renderers и approved `reports/*` / `proposals/*`

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
- MVP runtime write audit is detect-only: unexpected mutations of protected workspace surfaces or analyzed repo working trees are surfaced as run warnings/logs (`runtime_write_audit_unexpected_mutation`, `runtime_write_audit_repo_skipped`) and do not fail, restore, or sandbox the provider process

> MVP policy фиксирует step-scoped runtime provider contract: effective provider для шага выбирается как `workspace step override > CLI/env global provider > claude-code`; semantic stdout payloads не поддерживаются.
> CLI/process runtime mode задаётся флагом `--runtime fake|headless` (`fake` default, `headless` opt-in), global fallback provider — `--runtime-provider claude-code|qwen-code|codex-code` (env fallback `ACP_RUNTIME_PROVIDER`).
> В `--runtime fake` fallback provider валидируется как config surface, но execution metadata пишет neutral provider `fake`; live provider command не запускается.

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
  - `qa`
- prompt packs:
  - `constitution`
  - `collect-context`
  - `findings`
  - `proposals`
  - `qa`

Bundle поставляется вместе с продуктом, хранится в workspace и может редактироваться пользователем через UI/git workflow.

## Docs imports metadata (MVP)

Для imported docs используется metadata index:
- `<docs.imports_path>/index.yaml` (default `docs/imports/index.yaml`)

Required поля записи:
- `id`
- `path`

Optional поля записи:
- `source`
- `checksum`
- `imported_at`
- `source_updated_at`
- `status`

Отсутствие index допустимо и не создаёт diagnostic. Malformed/index semantic issues (`duplicate id`, path traversal/absolute/outside imports root, missing referenced file) surfacing как warning-only workspace diagnostics.

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
- `claude-code`, `qwen-code` и `codex-code` используют общий artifact-only process engine; provider adapters задают CLI args/stdin/workdir, unavailable markers и bounded activity/recovery policy
- `qwen-code` adapter invocation передаёт artifact prompt только через CLI `-p`; JSON task stdin не используется, custom qwen args нормализуются, чтобы не смешивать или подменять machine envelope с пользовательским prompt, а default invocation использует `stream-json` activity output без semantic stdout contract. `claude-code` и `codex-code` сохраняют свои transport-specific stdin/machine-mode surfaces.
- zero-output `pre_artifact` recovery остаётся adapter policy: `qwen-code` может сделать один warning/retry для любого artifact-monitored шага, а `claude-code` делает такой warning/retry только для validator/proposals steps (`init|refresh.step3.findings`, `init|refresh.step4.proposals`); exhausted silent no-artifact retry остаётся `runner_unavailable`
- stdout/stderr provider transcripts сохраняются как diagnostics/raw-output refs и никогда не трактуются как semantic success payload
- selected-provider readiness фиксируется в batch preflight; provider command/probe/auth/quota/version и codex CLI compatibility blockers являются operational failures до deep run. Provider `model`/`modelUsage` telemetry не является readiness contract и не блокирует release сама по себе
- semantic source of truth для `step1` — `shard-pack-manifest.json.semantic`
- `step1` semantic provenance evidence (`entities/edges/findings[].provenance.evidence[]`) обязаны содержать non-empty `repo` + `path`; citation-only semantic evidence objects невалидны
- semantic source of truth для `step3` — `validator-verdict.json.findings[]` и `validator-verdict.json.questions[]`
- runtime execution metadata сохраняют только execution context, status, warnings и raw-output references
- raw provider prompt argv payloads and stdout/stderr diagnostics are redacted before persistence and run-log streaming; lifecycle diagnostics keep prompt size, and prompt argv payloads are replaced with byte count + hash when present; stdout/stderr remain diagnostic only
- deterministic fake runtime пишет provider `fake`; headless adapters пишут `claude-code`, `qwen-code` или `codex-code`

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
- `<docs.imports_path>/index.yaml` (если есть) + configured imports files
- `docs/rfcs/*`, `docs/meetings/*`, `docs/decisions/*`
- `charter/*`
- `skills/*`

Runtime focuses on:
- arbitrary stacks через выбранный headless provider (`claude-code|qwen-code|codex-code`) + baseline skill/prompt bundle, без фиксированного whitelist parser implementations в MVP
- workspace prompt packs участвуют в composed prompt как editable content layer; merge order фиксирован: provider header -> artifact-only/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer. Enforced runtime policy/invariants задаются internal shared step-policy слоем и не могут быть ослаблены содержимым prompt pack. Editable prompt pack layer подключён к `step0.constitution`, `step1.collect`, `step3.findings` и `step4.proposals`; `step2.asis_docs` работает через enforced policy only без отдельного editable `as-is` prompt pack.
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
- collect manifest принимает только canonical vocabulary: `semantic.coverage.observed`, `semantic.questions[*].text`, `semantic.edges[*].type/from/to`, object-shaped `provenance`, numeric `confidence`; aliases вроде `covered_topics`, `question`, `relation`, `source`, `target`, finding `summary`/`inference`, array provenance, string confidence, `evidence_citation_ids`, top-level `step_contract` и `compatibility` считаются contract-invalid drift и reject-ятся schema/contract validation-ом
- требует global uniqueness для `citations[].claim_ids` в assembled staged final set; provider должен формировать `claim_id` как semantic stem + shard slug и добавлять deterministic numeric suffix при остаточной коллизии
- collect prompt ставит suggested authored doc path и literal task-specific `shard-pack-manifest.json` skeleton в command-first section сразу после provider identity, чтобы provider писал manifest сразу после первого authored shard doc, а не после broad repository sweep; эта heredoc command встречается один раз, а поздний docs-first block только ссылается на first-action skeleton; для `refresh.step1.collect` skeleton должен сразу удовлетворять refresh policy minimums (`coverage.missing >= 3`, `questions >= 1`), чтобы first-action heredoc не требовал последующей смысловой перестройки перед первым artifact write
- collect runtime до финальной validation может выполнить focused provider-authored recovery без ACP-side synthesis: если provider оставил stdout/stderr diagnostics, но не записал authored artifacts, общий engine делает одну collect pair-recovery попытку с write-set guard только на suggested authored doc + `write_root/shard-pack-manifest.json`; если provider уже записал authored docs, но `shard-pack-manifest.json` missing/invalid, общий engine делает одну manifest-only repair попытку с write-set guard только на `write_root/shard-pack-manifest.json`; ACP не нормализует `documents[].path`, не автозаполняет metadata и не использует semantic stdout repair
- все required runtime artifacts пишутся/проверяются по exact absolute `write_root`/`draft_final_root`; relative CWD checks/writes (`test -f validator-verdict.json`, `test -f nova-overview.md`) не являются валидным artifact target
- `step3.findings` при missing/invalid `validator-verdict.json` может выполнить одну focused provider-authored repair попытку: разрешённый write-set — только `write_root/validator-verdict.json`; repair prompt начинается с command-first absolute heredoc skeleton, но skeleton `checked_paths` ссылается на staged final artifacts, а не validator `write_root`; `issues[]` использует canonical shape `code`, `severity=error|warning`, `message`, optional `path/document_id/citation_id`; legacy finding-shaped fields внутри `issues[]` hard-invalid
- draft steps (`step0.constitution`, `step2.asis_docs`, `step4.proposals`) при missing/invalid draft artifacts могут выполнить одну focused provider-authored repair попытку: разрешённый write-set — step manifest в `write_root` и files under `draft_final_root`; repair prompt содержит command-first heredoc artifact set для manifest + referenced draft files и запрещает broad analysis до первого валидного draft set; normal `step4.proposals` prompt также начинается с единственного `FIRST PROPOSALS DRAFT COMMAND`, который пишет proposals manifest + draft files до broad proposal analysis; ACP не синтезирует manifest/verdict/draft files
- collect prompt для root-file shard (`path_scopes` содержит root-level files вроде `README.md`, `pom.xml`, `Makefile`) явно ограничивает чтение перечисленными файлами, выбирает primary evidence в порядке `README` -> `Makefile` -> build/deploy manifests -> прочие root files, и требует один concise root overview doc + `shard-pack-manifest.json` без recursive repo sweep.
- collect repair read surface уже collect: текущий `write_root` + repo evidence roots. Workspace-level `reports/taskruns`, sibling shard manifests, raw logs, archive docs, examples и filesystem schema scavenging не являются допустимым repair input; embedded prompt contract/schema fragment is authoritative for the repair attempt
- collect pair recovery prompt содержит command-first absolute heredoc write targets для suggested authored doc + task-specific manifest skeleton; manifest-only repair prompt содержит command-first absolute heredoc write target с task-specific JSON skeleton, overwrite-only instruction для existing invalid manifests и compact validation checklist; authored docs/evidence уже encoded в skeleton, root-file repair evidence использует тот же `README` -> `Makefile` -> build/deploy preference, а validation-error details не повторяются как field-level patch cues. Это не ACP-side manifest synthesis, а provider instruction surface под write-set guard.
- markdown-only collect completion невалиден: hard pass невозможен, пока каждый required shard не имеет валидный `shard-pack-manifest.json` после разрешённых collect recovery попыток
- для `step1.collect` runtime использует selected repo roots, `write_root` и explicit `read_context_roots`; `reports/taskruns/**`, raw logs и archive docs не считаются schema source of truth и не должны использоваться как manifest examples. Manifest-only repair дополнительно исключает broader workspace read scope, чтобы sibling shard manifests не становились неявными examples
- root-marker-only repos (например только `pyproject.toml`/`pom.xml` в корне) не планируются как единый shard `"."`, если top-level структура большая: planner строит root-file group + top-level directory shards, затем применяет существующий cap/coalescing и сохраняет marker leaves where possible
- coalesced shard IDs are stable filesystem-safe bounded slugs; overlong root-file groups keep a readable prefix plus hash suffix so runtime artifact directories remain portable across local filesystems
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
- `step2.asis_docs` не имеет отдельного editable workspace prompt pack; live runtime получает enforced step policy + artifact-only contract, а не `skills/prompt-packs/as-is.md`
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

Runtime proposal contract:
- если live runtime step выполняется, primary output — `proposals-draft-manifest.json` + optional proposal/changelog drafts внутри `draft_final_root`;
- manifest обязан использовать strict canonical contract: `version=1`, `run_id`, `step_id`, `step_contract="proposals"`, `agent_role`, optional `summary`, `outputs[]`;
- `outputs[].path` relative только к `draft_final_root`, `outputs[].canonical_path` workspace-relative и unique;
- allowed publish surface для `outputs[].canonical_path`: только `proposals/*` и `reports/changelog/*`;
- normal provider prompt должен показывать command-first heredoc write-set (`FIRST PROPOSALS DRAFT COMMAND`) для exact `write_root/proposals-draft-manifest.json` и referenced files под `draft_final_root`; command block встречается один раз, поздние sections ссылаются на canonical shape без повторного heredoc;
- legacy/final-index-like envelopes (`pipeline`, `step`, `generated_at`, `domain_id`, `proposals[]`, `info_findings_noted`, `orphan_coverage_gaps`) являются contract drift и reject-ятся strict parser-ом до promotion.

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
- System Analyst Q&A capability в текущей beta работает как deterministic workspace-backed read-only service + CLI `acp qa` + public read-only `POST /api/qa/ask` поверх:
  - `charter/cards/*`
  - `model/*`
  - `reports/*`
  - configured `docs.imports_path`
- Capability не вызывает headless runtime provider, не использует `skills/prompt-packs/qa.md` как runtime prompt и не меняет workspace.
- API response shape: `answer`, `citations`, `unresolved`, `confidence`.
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
- Webhook принимает CI provider, не ACP: native SCM webhook listener / external SCM app integration остаются вне MVP.
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
- strict `proposals-draft-manifest.json` contract обязателен для live runtime path: `version=1`, `run_id`, `step_id`, `step_contract="proposals"`, `agent_role`, optional `summary`, `outputs[]`;
- `outputs[].path` relative только к `draft_final_root`, `outputs[].canonical_path` workspace-relative, unique и разрешён только под `proposals/*` или `reports/changelog/*`;
- legacy/final-index-like envelopes запрещены: top-level `pipeline`, `step`, `generated_at`, `domain_id`, `proposals[]`, `info_findings_noted`, `orphan_coverage_gaps` должны hard-fail-иться strict parser-ом;
- deterministic promoter проверяет schema/semantic/validator gates;
- обязательного human approve нет;
- canonical `proposals/*` и `reports/changelog/*` публикуются автоматически только после successful gates.
