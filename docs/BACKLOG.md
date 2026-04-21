# BACKLOG (baseline)

Этот backlog описывает эпики реализации, критерии приёмки MVP и рекомендуемую PR-level нарезку.
Для MVP-эпиков `Suggested PR slices` зафиксированы прямо в этом файле.
Required CI для MVP опирается на schema/contracts, synthetic fixtures, fake/recorded runner artifacts и не требует live headless provider binaries.

## Epic 1 — Управление workspace
Acceptance:
- `workspace.yaml` валидируется по `schemas/workspace.schema.json`
- читается `workspace.yaml`
- валидируются repo entries с `path` или `git_url`
- `git_url` clone/fetch-ится через локальный `git` текущего пользователя/runner без отдельного credential store ACP
- валидируется структура central `arch-workspace` (`charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/imports/`)
- в UI отображается статус workspace (репозитории подключены, local git access доступен, папка docs доступна)
- fixture-driven negative cases покрывают invalid manifest combinations и duplicate `repo.name`

Suggested PR slices:
- `1A workspace schema/spec`
- `1B manifest parser`
- `1C local git resolver`
- `1D workspace layout validator`
- `1E workspace validator negative cases`

## Epic 2 — Artifact contracts + runtime execution metadata
Acceptance:
- orchestrator принимает только required step artifacts и persisted `runtime-execution.json`
- collect semantic state приходит только через `shard-pack-manifest.json.semantic`
- findings приходят только через `validator-verdict.json`
- invalid artifact contract завершается с понятной ошибкой `runtime_contract_failed`
- semantic tests покрывают `observation without evidence`, mixed normalized forms и explicit error surfacing

Suggested PR slices:
- `2A schema validation wiring`
- `2B semantic validation rules`
- `2C validator fixtures/errors`
- `2D runtime execution metadata`
- `2E artifact-only error tests`

## Epic 3 — Headless runtime adapters (historически стартовал как Claude Code adapter)
Acceptance:
- orchestrator запускает headless runtime adapter для workspace (`claude-code` default, `qwen-code` optional)
- поддерживается передача PromptPack + subagents + skills
- adapter работает поверх baseline bundle agents/skills/prompts
- persisted runtime execution metadata сохраняется в `reports/taskruns/`
- required tests используют fake/recorded runner harness вместо live provider binaries

Suggested PR slices:
- `3A runner interface`
- `3B process execution + stdout/stderr`
- `3C taskrun persistence`
- `3D fake runner + recorded artifact-only harness`

## Epic 4 — Model store (entity-per-file)
Acceptance:
- сущности и связи создаются/обновляются как YAML-файлы
- поддерживаются поля provenance/confidence
- есть минимальный resolver для stable IDs + aliases
- stable ID normalization и collision policy детерминированы и защищены fixtures
- golden regressions защищают materialized entity/edge outputs и stable ID behavior

Suggested PR slices:
- `4A entity/edge file store`
- `4B alias resolver`
- `4C semantic snapshot apply`
- `4D stable ID normalization`
- `4E golden stable-ID regressions`

## Epic 5 — Init pipeline 0–4 (MVP)
Acceptance:
- Step 0: Charter wizard (template-based) сохраняет артефакты в `charter/`
- Step 0: Charter wizard создаёт initial canonical domain/team cards
- Step 1: Collect context формирует модель и coverage по architecture/integrations/datastores/CI-CD
- Step 1: extraction не ограничен фиксированным whitelist языков/стэков; используется LLM runner + baseline prompt bundle
- Step 1: при нехватке evidence создаются `questions` и coverage gaps вместо выдуманных фактов
- Step 1: existing domain/team cards только enrich-ятся derived sections; auto-create/rename canonical cards не допускается
- Step 2: As-is docs формируются в `reports/as-is/`, включая service dossiers, integrations, datastores и CI-CD views
- Step 3: Findings формируются в `reports/findings/`
- Step 4: Proposals формируются в `proposals/<topic>/`
- synthetic scenario repos и golden outputs покрывают pipeline results end-to-end без live network

Suggested PR slices:
- `5A step0 charter bootstrap`
- `5B step1 collect`
- `5C coverage/questions persistence`
- `5D as-is compiler`
- `5E findings runtime`
- `5F proposals compiler`
- `5G changelog generation`
- `5H scenario pipeline golden tests`

## Epic 6 — UI baseline
Acceptance:
- workspace setup принимает локальные папки и GitHub/GitLab URL
- экраны: Charter wizard/editor, Skills editor, Prompt bundle editor, Run pipeline, Results viewer
- coverage/questions по missing info доступны в results viewer
- git helper: Commit changes + Create proposal branch (baseline decision)
- UI smoke покрывает open workspace -> validate -> run -> inspect results

Suggested PR slices:
- `6A workspace setup screen`
- `6B charter editor`
- `6C skills/prompt bundle editor`
- `6D run/results viewer`
- `6E UI smoke coverage`

## Epic 7 — Domain/Team Cards (MVP)
Acceptance:
- source-of-truth карточки доменов ведутся в `charter/cards/domains/*.md`
- source-of-truth карточки команд ведутся в `charter/cards/teams/*.md`
- cards bootstrap-ятся wizard-ом и считаются human-owned canonical IDs
- карточки версионируются в Git и связаны с model/findings/proposals артефактами
- unknown domain/team не создаётся автоматически runtime’ом, а фиксируется как question/finding
- описаны минимум: ссылки, описание домена/команды, люди/ресурсы, ключевые технологии

Suggested PR slices:
- `7A domain card template`
- `7B team card template`
- `7C card bootstrap/linkage rules`

## Epic 8 — Domain Agent Layer + Architect Aggregation (MVP)
Acceptance:
- domain-first слой: Domain Analyst Agent запускается per domain
- outputs domain-агентов сохраняются в `reports/agent-outputs/domains/*.md`
- Architect Aggregator Agent анализирует domain outputs и формирует `reports/agent-outputs/architect/summary.md`
- orchestrator поддерживает детерминированный порядок запуска/сборки для повторяемых результатов
- fan-out опирается на существующие canonical domain cards; unresolved domains/owners surface как questions/findings

Suggested PR slices:
- `8A domain fan-out orchestration`
- `8B architect aggregation artifact`
- `8C unresolved domain/team escalation`

## Epic 9 — System Analyst Q&A Capability (MVP)
Acceptance:
- on-demand Q&A capability доступна поверх `charter/cards + model + reports + docs/imports`
- ответы содержат ссылки на evidence/артефакты workspace
- capability работает без расширения API-контракта на текущем шаге

Suggested PR slices:
- `9A workspace-backed QA service`

## Epic 10 — Iteration Changelog (MVP)
Acceptance:
- на каждую итерацию формируется `reports/changelog/<yyyy-mm-dd>-<iteration-id>.md`
- changelog отражает изменения модели, findings, proposals и agent outputs
- changelog детерминирован при одинаковых входах

Suggested PR slices:
- `10A changelog compiler integration`

## Epic 11 — Q&A API Contracting Step (separate follow-up)
Acceptance:
- отдельный design/contract шаг в backlog для будущего API Q&A
- фиксируется read-only endpoint `POST /api/qa/ask`
- response shape содержит `answer`, `citations`, `unresolved`, `confidence`
- endpoint не меняет workspace и не требует изменения runtime artifact contracts

Suggested PR slices:
- `11A /api/qa/ask contract + read-only semantics`

## Epic 12 — Autodocs Integration (Wave 1+)
Acceptance:
- определены интеграции autodocs с внешними источниками
- зафиксированы guardrails по deterministic imports и provenance
- вынесено за пределы MVP

Status:
- horizon item; PR slicing отложен до планирования Wave 1

## Epic 13 — Jira Manager Agents (Wave 1+)
Acceptance:
- manager-агенты анализируют Jira backlog per domain/team
- отчёт показывает перекосы по ресурсам и bottlenecks
- вынесено за пределы MVP

Status:
- horizon item; PR slicing отложен до планирования Wave 1

## Epic 14 — GitHub/GitLab CI Trigger Mode (MVP)
Acceptance:
- `cmd/acp` поддерживает non-interactive запуск без UI для GitHub/GitLab CI jobs
- `acp serve --workspace <abs-path>` поднимает single-workspace-per-process service
- batch mode работает с тем же `workspace.yaml` и теми же pipeline step IDs
- hook-triggered workflow и manual pipeline button/job запускают тот же ACP flow
- default auto-trigger: `push` в default branch
- `merge request` / `pull request` updates в MVP идут как manual/preview trigger
- одновременно активен только один run на workspace; debounce window 5 минут, policy `last event wins`
- git access использует локальный runner/user context и surface как actionable warnings/errors
- job не пишет в пользовательские репозитории, только в workspace и local checkout/cache
- required CI/CD surface: CLI batch mode; internal API trigger остаётся optional trusted-mode capability
- smoke tests покрывают CLI/API trigger path и locking/debounce behavior без live SCM dependencies

Suggested PR slices:
- `14A CLI batch mode`
- `14B single-workspace serve + API boundary`
- `14C hook/manual trigger integration`
- `14D run locking + debounce`
- `14E CI error surfacing`
- `14F CLI/API trigger smoke tests`

## Epic 15 — Baseline Agent/Skill/Prompt Bundle (MVP)
Acceptance:
- product поставляется с baseline `skills/subagents.yaml`
- baseline agents фиксированы: `domain-analyst`, `architect-aggregator`, `system-analyst-qa`
- baseline skills фиксированы: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`
- baseline prompt packs фиксированы: `constitution`, `collect-context`, `findings`, `proposals`, `qa`
- bundle редактируется пользователем и версионируется в Git

Suggested PR slices:
- `15A subagents.yaml`
- `15B baseline skill packages`
- `15C baseline prompt packs`
- `15D UI editing/validation integration`

## Cleanup follow-up (post-beta, owner confirmation required)

Открытые пункты после cleanup slice:

1) Пересмотр необходимости persisted слоя `fixtures/scenarios/*/golden/readable`
- Owner: ACP maintainers + QA/testing owner
- Risk: medium (влияние на review diffability и developer UX)
- Next step: собрать usage evidence, затем принять решение retain/remove отдельным PR

2) Owner confirmation для cleanup-кандидатов с неявным usage risk
- Scope:
  - duplicated readable scenario fixtures (possible dedupe, policy-sensitive)
- Owner: ACP maintainers + docs owner + tooling owner
- Risk: medium/high (риск удалить скрыто используемые файлы или ухудшить regression/review UX)
- Next step: подтвердить explicit ownership и usage contracts, затем принять retain/remove/dedupe решение отдельным PR.

Resolved (2026-04-05):
- `slugify` дедупликация между подсистемами выполнена через `internal/slugutil` + regression tests.
- `.codex/model_instructions.md` удалён из tracked surface в cleanup-срезе.
