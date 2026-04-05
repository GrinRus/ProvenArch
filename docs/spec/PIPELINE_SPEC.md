# Спецификация пайплайна (MVP v0)

Документ описывает pipeline ACP через input/output контракты и expected artifacts.

## Общие понятия

- **Workspace**: единый central git-репозиторий `arch-workspace/` (каноническая MVP-конвенция, Variant 2) с `workspace.yaml`, `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/`.
- **Workspace manifest**: `workspace.yaml`, валидируемый по `schemas/workspace.schema.json` и описанный в `docs/spec/WORKSPACE_SPEC.md`.
- **Orchestrator**: управляет шагами, готовит PromptPack/ContextPack, вызывает runtime, валидирует TaskResult.
- **Runtime (MVP)**: Claude Code headless (opt-in) + deterministic fake harness (default for required CI/testing).
- **TaskResult**: структурированный JSON ответа runtime-шага (`schemas/taskresult.schema.json`).

> Несмотря на schema-гибкость, MVP policy фиксирует production/runtime target как `claude-code`, а MVP changeset contract считаем замороженным без `write_file`.
> CLI/process runtime mode задаётся флагом `--runtime fake|headless` (`fake` default, `headless` opt-in).

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
- Step 1 может обновлять только derived references, coverage links и аналитические секции в существующих cards.
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

## TaskResult semantics (MVP)

Canonical MVP runtime shape:
- `questions[]` пишутся на top-level
- `coverage` пишется на top-level
- `changeset` содержит model/findings/doc metadata operations

Legacy-compatible forms:
- `add_question`
- `set_coverage`

Orchestrator normalization policy:
- operation-form `add_question` и `set_coverage` нормализуются в canonical top-level representation до persistence/materialization
- если в одном TaskResult присутствуют и top-level поля, и operation-form:
  - source of truth для persisted artifacts — merged result after normalization
  - duplicate questions dedupe по `id`
  - coverage merge для `observed`, `missing`, `notes` идёт по unique string

`add_doc_artifact` в MVP:
- трактуется только как metadata registration op
- может ссылаться на уже существующий или позднее materialized orchestrator artifact
- не несёт content payload
- не является скрытым `write_file`
- не является required dependency path для обязательных outputs Step 1 или Step 3

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

Step 0 materialization policy:
- если `charter/wizard/step0-contract.json` валиден, его поля детерминированно влияют на `charter/*` и canonical cards;
- если contract отсутствует/невалиден, применяется baseline fallback materialization;
- fallback фиксируется warning-сообщением в run diagnostics (`GET /api/pipeline/runs/<run_id>.warnings`).

### Step 1 — Collect context (runtime step)
Inputs:
- `workspace.yaml` из корня central `arch-workspace`
- локальные checkout репозиториев, полученные из `path` и/или local git resolution of `git_url` на той же машине
- `docs/imports/index.yaml` (если есть) + `docs/imports/*`
- `docs/imports/*`, `docs/rfcs/*`, `docs/meetings/*`, `docs/decisions/*`
- `charter/*`
- `skills/*`

Runtime focuses on:
- arbitrary stacks через Claude Code + baseline skill/prompt bundle, без фиксированного whitelist parser implementations в MVP
- service topology и entrypoints
- interfaces (HTTP/gRPC/events)
- external systems/integrations
- datastores и storage usage
- CI/CD evidence (`.gitlab-ci.yml`, Dockerfile, deploy manifests, helm/k8s, scripts)
- ownership hints и явные unknowns

Runtime output (TaskResult):
- `changeset`: `upsert_entity`, `upsert_edge`, optional `add_doc_artifact`
- optional top-level `questions`
- optional top-level `coverage`
- legacy-compatible `add_question` / `set_coverage` допустимы, но не являются canonical MVP form

Orchestrator applies:
- валидирует TaskResult schema
- нормализует legacy question/coverage ops в canonical top-level representation
- выполняет runtime `init.step1.collect`/`refresh.step1.collect` отдельно для каждой canonical domain card (`charter/cards/domains/*`)
- materialize-ит отдельный raw taskrun на каждый домен в `reports/taskruns/*-step1-collect-domain-<domain>.json`
- обновляет `model/*`
- сохраняет taskrun under `reports/taskruns/*`
- формирует `reports/coverage/summary.md`
- формирует `reports/coverage/open-questions.md`
- enrich существующие `charter/cards/domains/*` и `charter/cards/teams/*` через детерминированную секцию `## Derived (ACP Step1)`:
  - related model IDs / findings / questions
  - coverage missing summary
  - evidence refs (для domain и team cards)
- не создаёт и не переименовывает canonical cards автоматически
- сохраняет outputs domain-агентов в `reports/agent-outputs/domains/*`

### Step 2 — As-is docs (compiler step in MVP)
Inputs:
- `model/*`
- `charter/*`
- `skills/templates/*`

Outputs:
- `reports/as-is/overview.md`
- `reports/as-is/service-catalog.md`
- `reports/as-is/services/<service-id>.md`
- `reports/as-is/dependencies.md` (optional)
- `reports/as-is/integrations.md`
- `reports/as-is/datastores.md`
- `reports/as-is/ci-cd.md`

> В MVP это детерминированная компиляция из модели.

### Step 3 — Findings (runtime step)
Inputs:
- `model/*`
- `charter/rules.yaml`
- `skills/*`

Runtime output (TaskResult):
- `changeset`: `add_finding`
- optional top-level `questions`
- optional top-level `coverage`
- legacy-compatible `add_question` / `set_coverage` допустимы, но не являются canonical MVP form

Orchestrator applies:
- нормализует legacy question/coverage ops в canonical top-level representation
- обновляет `reports/findings/*`
- обновляет `reports/agent-outputs/architect/summary.md` через детерминированную агрегацию фактических domain outputs
- materializes critical unknowns как findings, если отсутствуют owner/integration/database/CI-CD evidence

### Step 4 — Proposals (compiler/templates in MVP)
Inputs:
- `model/*`
- `reports/findings/*`
- `charter/*`
- `skills/templates/*`

Outputs:
- `proposals/<proposal-id>/proposal.md`
- `proposals/<proposal-id>/ADR.md`
- `proposals/<proposal-id>/RFC.md`
- `proposals/<proposal-id>/migration-checklist.md`

> В MVP proposals формируются без write-file операции в TaskResult, через orchestrator templates/compiler.

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
