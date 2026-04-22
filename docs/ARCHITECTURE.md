# ARCHITECTURE.md (Go monorepo MVP)

Этот документ описывает целевую архитектуру реализации ACP для local-first MVP.
В текущем состоянии реализован runnable baseline: `workspace` foundations, docs-first staged runtime pipeline для `init|refresh`, API endpoints `/api/*`, validator-gated promotion в `reports/*`/`proposals/*`, derived model materialization `model/*` и fake-runner default для required CI без live dependencies.

## Scope (MVP)
- Local-first: всё работает на машине разработчика
- Тот же entrypoint поддерживает non-interactive batch execution в GitHub/GitLab CI jobs
- Runtime (analysis): **headless multi-provider** (`claude-code` default, `qwen-code` optional, `codex-code` release peer) + deterministic `fake` baseline
- Реализация продукта: **Go backend/orchestrator + embedded React UI**
- Единая workspace-конвенция MVP: central `arch-workspace` git-репозиторий (Variant 2)
- Agent operating model MVP: идеи `1,2,3,5,7` (domain-first cards/agents/changelog/Q&A)
- Нет hosted режима и нет security/compliance enforcement в MVP
- Autodocs и Jira manager agents остаются post-MVP (Wave 1+)

## Компоненты
1) **Go entrypoint (`cmd/acp`)** *(implemented baseline)*
   - exact CLI flag/help surface остаётся canonical в `acp --help` / `acp <command> --help` и `cmd/acp/main.go`; этот документ фиксирует behavior boundary, а не второй help-manual
   - `init-workspace` создаёт/обновляет `workspace.yaml`, bootstrap-ит fixed layout/baseline bundle и выполняет dry validation для первого старта
   - Раздаёт UI (embedded static assets из `ui/dist`)
   - Экспортирует API под `/api/*`
   - `serve` поднимает single-workspace-per-process local API+UI service
   - `serve --auto-init` bootstrap-ит workspace manifest/layout при отсутствии `workspace.yaml`
   - bootstrap (`init-workspace`/`serve --auto-init`) автоматически делает `git init` для workspace root при отсутствии `.git`
   - startup для `serve` lenient: без блокирующего repo preflight; readiness diagnostics доступны через `/api/workspace/validate`
   - Поддерживает batch/non-interactive режим для CI jobs
   - `run` выполняет deterministic `init|refresh` pipeline в local/batch/non-interactive execution
   - `qa` даёт read-only ответы по артефактам workspace
   - runtime selector process-scoped: `fake` default для required CI, `headless` opt-in
   - global provider selector остаётся process-level fallback: `--runtime-provider` > `ACP_RUNTIME_PROVIDER` > `claude-code`
   - effective provider resolution внутри run step-scoped: `workspace.yaml.runtime.profile.steps.<step>.provider` переопределяет global fallback только для выбранного шага
   - timeout и execution control остаются process/workspace-aware: persisted профиль живёт в `workspace.yaml.runtime.profile.*`, а точные precedence/API surfaces удерживаются в `docs/spec/API_SPEC.md` вместо дублирования здесь
   - Используется как локально, так и из SCM-triggered pipeline jobs/manual buttons
   - Internal API trigger остаётся optional trusted-mode capability, а не обязательной CI/CD поверхностью
   - Раздаёт embedded UI shell и API в одном процессе `acp serve`

2) **UI (`ui/`)** *(baseline shell)*
   - React + TypeScript + Vite
   - Dev: `npm run dev` с proxy на backend
   - Prod: `npm run build` → `ui/dist` встраивается в Go бинарь
   - Live browser e2e: Playwright optional smoke (`ui/e2e/live-flow.spec.ts`, `npm run e2e:live --prefix ui`)
   - Guided setup поддерживает multi-repo (`repos[]`) с add/remove rows и optional `ref`
   - Показывает repo overview в validate surface: `resolved_repos` + diagnostics, сгруппированные по repo
   - Редактирует baseline bundle artifacts через guided selector (`charter/*`, `skills/*`, prompt packs, `skills/subagents.yaml`)
   - UI разбит на top-level tabs `Setup / Baseline / Runs / Results / Settings`
   - `App.tsx` остаётся route shell, а крупные sections вынесены в dedicated panels (`SetupWorkspacePanel`, `WizardContractPanel`, `BaselineEditorsPanel`, `RunPanels`, `ResultsPanels`)
   - setup/baseline/wizard/git state и actions вынесены в `useWorkspaceSetup`; runtime settings живут в отдельном hook, а внутри run explorer logs/artifacts разделены на internal hook modules
   - Runtime profile (`timeouts` + `execution`) полностью вынесен в вкладку `Settings`, включая effective per-step providers
   - Показывает run dashboard (queued/running/succeeded/failed), включая завершённые run'ы из persisted history
   - При bootstrap авто-выбирает newest active run (`queued/running`), иначе первый run в history
   - Если выбранный run исчезает из history и есть новый доступный run, UI переключается на него; если history временно пуста, но status endpoint ещё возвращает выбранный run, UI сохраняет текущий selection и не делает ложный auto-switch
   - Показывает `Run status` выбранного run с полным warnings list (`RunInfo.warnings`), `error_code` и `error`
   - Показывает `Runs: Logs` для выбранного run (`timestamp/level/step/domain/message`) с dual-view `event timeline | raw agent stream | all`, переключателем `line | line+fields` и quick actions `Copy logs`, `Download logs`, `Open runtime execution artifact`
   - `Results` включает sub-tabs `Coverage / Artifacts / Diagrams`, где `Diagrams` рендерит Mermaid previews для `reports/diagrams/*`
   - Поддерживает `Cancel selected run` для active run через `POST /api/pipeline/runs/<run_id>/cancel`
   - Runtime Timeouts settings panel:
     - load/save/reset через `GET/PUT /api/runtime/timeouts`
     - показывает persisted/effective/source для каждого timeout поля
   - Runtime Execution settings panel:
     - load/save/reset через `GET/PUT /api/runtime/execution`
     - показывает persisted/effective/source для strategy/parallelism/failure/discovery
   - live e2e poll timeout-ы берутся из effective config (`/api/runtime/timeouts`) с env override
   - Критичные UI-контролы для live e2e снабжены стабильными `data-testid` (`validate/run/status/artifacts/logs`)

3) **Orchestrator (`internal/orchestrator`)** *(implemented baseline)*
   - Step registry (шаги init pipeline)
   - Step 0 support-artifacts materialization читает persisted wizard contract `charter/wizard/step0-contract.json`
   - при missing/invalid wizard contract применяется deterministic baseline fallback только для support artifacts, а warning фиксируется в run diagnostics
   - baseline bundle seeding выполняется create-if-missing, без перезаписи пользовательских правок; support-only bundle не пишет canonical `skills/subagents.yaml`, поэтому source of truth для него остаётся validated `constitution-draft.json`
   - Готовит ContextPack/PromptPack
   - Загружает baseline bundle agents/skills/prompts из workspace
   - workspace prompt packs подключаются к runtime prompt composition как editable content layer по фиксированному merge order: provider header -> artifact-only/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer; содержимое prompt pack не может ослаблять enforced contract rules
   - `internal/orchestrator/orchestrator.go` остаётся entry shell/pipeline glue; service run-registry/history lifecycle и semantic/card enrichment вынесены в dedicated package files
   - Работает с единым central workspace (`arch-workspace`) как корнем артефактов MVP
   - Валидирует `workspace.yaml` по `schemas/workspace.schema.json`
   - Разрешает repo sources (`path`/`git_url`) в локальные checkout перед анализом через системный `git` текущего пользователя/runner
   - Оркестрирует domain-first слой агентов
   - Materialize-ит per-domain execution contracts (`reports/agent-outputs/domains/*.task-envelope.json`) для canonical domain cards
   - Step1 repo binding: источник истины `repo_scope` в domain card; fallback только slug-match `domain_id` ↔ `repo.name`
   - Runtime step1 и enrich domain cards используют общий resolver `repo_scope` (declared -> slug fallback -> empty)
   - Проверяет согласованность canonical domain card: filename `<domain-id>.md` vs поле `- id:`; mismatch фиксируется high-priority question, runtime остаётся filename-based
   - Монолитный сценарий many-domains-to-one-repo поддержан через общий `repo_scope`; unknown scope фиксируется вопросом `q.domain.<id>.unknown-repo-scope`
   - Выполняет runtime collect-step per-domain и выдаёт каждому shard runtime-aware envelope:
     - `artifact_root` (workspace-relative)
     - `write_root` (absolute run-scoped staging dir)
     - `read_context_roots[]`
   - Step 1 runtime primary output: authored shard dossier pack + `shard-pack-manifest.json`
   - Collect repair возвращает structured repair report (`changed`, `applied_rule_ids`) и логирует compatibility rule ids через runtime diagnostics вместо silent normalization
   - Active compatibility inventory ограничен двумя safe rules из `internal/runtime/compatibilityregistry`: `collect.documents_path_normalization` и `drafts.reconcile_existing_canonical_outputs`
   - Сохраняет raw runtime execution metadata и shard summaries в `reports/taskruns/*` для recovery/auditability
   - Runtime sharding planner (heuristics/semantic) materialize-ит deterministic shard-plan artifacts `reports/taskruns/*-shard-plan*.json` и shard-summary artifacts `reports/taskruns/*-shard-summary*.json`
   - shard-plan публикует полный неперекрывающийся coverage partition repo через `path_scopes` (directory/file scopes); для больших repo применяется только structural coalescing по filesystem ancestry
   - Per-shard persistence crash-safe: shard-summary materialize-ится сразу со status=`pending`; после validated runtime execution metadata internal `runtime-execution.json` пишется до `apply`, shard переходит в `checkpointed`, после успешного `apply` — в `succeeded`; runtime/apply failure фиксируется как `failed` без ожидания конца шага
   - Internal shard-summary contract: `taskrun_path` обязателен для `checkpointed/succeeded`; он должен ссылаться на persisted `runtime-execution.json` с `shard_id/repo_scopes/path_scopes`
   - Internal shard-plan/shard-summary artifacts materialize-ят non-empty `meta.runtime.name/meta.runtime.version`, чтобы internal batch/contract checks не трактовали их как runtime-name drift
   - Scheduler поддерживает `sequential|parallel` execution с worker-pool (`max_parallel_tasks`) и `fail_fast|best_effort` failure-policy
   - При `best_effort` downstream шаги продолжаются на partial model, но итог run фиксируется как `failed` с `error_code=run_partial_failed`; если `step1.collect` становится `unusable`, live `step3.findings` не выполняется, а downstream markdown artifacts (`as-is/findings/coverage/proposals/agent-outputs`) materialize-ятся в `report_mode=incomplete` с явным banner/triage-only wording
   - Вызывает runtime adapter через `StepRunnerResolver` с per-provider cache/preflight внутри одного run
   - `step0..step4` становятся agent-first шагами, но runtime получает только staged surfaces (`write_root`, `draft_final_root`, `read_context_roots`, `step_contract`, `expected_artifacts`)
   - canonical publish для `step0/2/4` выполняется только из validated runtime draft artifacts через deterministic compile/publish path; direct orchestrator writer больше не является альтернативным source of truth
   - Собирает staged final doc set в `reports/taskruns/<run_id>/staging/final/`
   - Генерирует и валидирует `final-run-index.json` и `citation-index.json`
   - `citation-index.json.claim_ids` трактуются как global staged-final namespace; duplicate claim ids в validator scope детерминированно repair-ятся на index/reference уровне с shard suffix без semantic rewrite authored docs
   - Step 3 runtime primary output: `validator-verdict.json`
   - Validator repair остаётся явной internal stage между load/parse verdict и финальной staged validation; repair write path остаётся atomic и не превращается в hidden mutation validation path
   - Promotion копирует только approved final set в stable `reports/as-is/*`, `reports/findings/*`, `reports/coverage/*`, `reports/agent-outputs/*`, `proposals/*`
   - обязательный human gate перед publish отсутствует; после успешных compile/validator gates promotion происходит автоматически
   - Валидирует только required step artifacts и persisted runtime execution metadata, а не semantic stdout payload
   - Нормализует canonical top-level `questions`/`coverage` (dedupe/canonicalization) без ingestion из legacy operations
   - Применяет semantic guard для refresh-taskruns: фильтрует placeholder/off-topic артефакты в `refresh.step1.collect`, добавляет deterministic fallback finding при owner-gap в `refresh.step3.findings`, канонизирует/дедуплицирует coverage/question semantics
   - Поддерживает derived model layer: `model/*` rebuild-ится из `final run index + citation index` после успешного promotion
   - Не auto-create/rename canonical domain/team cards и не разрешает runtime напрямую писать в `charter/*`
   - Для incomplete/failed terminal paths materialize-ит fallback markdown surfaces без promotion approved set
   - Поддерживает async run coordination: single active run + debounce queue (`last event wins`)
   - Поддерживает управляемую отмену run:
     - pending run в debounce queue отменяется immediate (`failed`, `error_code=run_canceled`)
     - active run отменяется cooperative через `context cancel` (`failed`, `error_code=run_canceled`)
     - если cancel request пришёл раньше конкурирующего layout/validation failure, terminal surface сохраняет `error_code=run_canceled`, а validation error остаётся в logs/warnings
   - На старте сервиса stale `queued` run по-прежнему reconcile-ится в `failed` с `error_code=run_reconciled_after_restart`
   - Для async service-managed run stale `running` run auto-resume-ится с тем же `run_id`, если есть resumable shard artifacts; resume cursor стартует с persisted `runtime-execution.json` для `step1.collect`, `step2.asis_docs` при resume после более позднего шага может быть детерминированно пересобран из persisted collect artifacts без live provider rerun
   - Ведёт persisted run history в `reports/taskruns/run-history.json` (versioned index, retention 500)
   - Ведёт run-level logs в `reports/taskruns/logs/<run_id>.ndjson` с cursor query API (`GET /api/pipeline/runs/<run_id>/logs`)
   - Runtime seam поддерживает live forwarding stdout/stderr от headless providers в run logs (`kind=runtime_output`, `stream=stdout|stderr`), event stream и raw stream сосуществуют
   - Internal safeguard ограничивает raw runtime stream hard-cap и публикует явный truncation marker (`fields.output_truncated=true`)
   - При runtime/parse fail логирует structured diagnostics snippets (`stdout_snippet`/`stderr_snippet`) в `RunLogEntry.fields` (sanitize + truncate)
   - Пробрасывает runtime warnings из execution metadata в run diagnostics (`RunInfo.Warnings`) и логирует warning events
   - Runtime step execution:
     - `executeRuntimeTask` выполняет runner под `context.WithTimeout(step_timeout_sec)`
     - heartbeat-log `runtime task heartbeat` публикуется раз в `heartbeat_sec`
     - timeout/cancel причины добавляются в error message без изменения `error_code` контракта
   - Materialize-ит per-run quality summary `reports/taskruns/<run_id>-quality.json` (signal metrics/runtime versions + `evidence_state.collect/findings/report_mode/reasons`)
   - Materialize-ит per-run repo selection summary `reports/taskruns/<run_id>-repo-selection-summary.json` (mode + selected scopes + include/exclude reasons)
   - Run logs retention policy (TTL + max runs) запускается при старте сервиса, перед run и после run
   - (опционально) делает git commit

4) **Agent Topology (domain-first, baseline)**
   - Domain Analyst Agent (per domain)
   - Team overlay через `charter/cards/teams/*`
   - Architect Aggregator Agent (анализ outputs domain-агентов)
   - System Analyst Q&A Agent (on-demand ответы по артефактам workspace, internal capability + `acp qa`)
   - Базовые skill/prompt bundles поставляются вместе с продуктом и versioned в workspace

5) **Runtime providers (`internal/runtime/*`)** *(implemented baseline)*
   - headless providers: `claude-code` (`internal/runtime/claudecode`), `qwen-code` (`internal/runtime/qwencode`) и `codex-code` (`internal/runtime/codexcode`)
   - общий runtime layer + provider factory: `internal/runtime/runtime.go`, `internal/runtime/providers/factory.go`
   - каждый provider получает explicit staged-write contract (`artifact_root`, `write_root`, `draft_final_root`, `read_context_roots`, `step_contract`, `expected_artifacts`) и должен писать runtime-authored artifacts только внутрь `write_root`/`draft_final_root`
   - live headless providers считаются успешными только при completed process + valid required artifacts; missing/invalid artifacts классифицируются как `runtime_contract_failed`
   - `shard-pack-manifest.json.documents[].path` — strict `artifact_root`-relative contract; workspace-level prefixes (`reports/...`, `charter/...`, `proposals/...`) и duplicated `artifact_root` prefix считаются invalid collect artifact drift. Runtime canonicalizer может детерминированно переписать такой path только если он однозначно указывает на существующий файл внутри `write_root`
   - для `init.step1.collect` / `refresh.step1.collect` runtime выполняет максимум одну post-success artifact-repair попытку, если `shard-pack-manifest.json` выглядит skeletal/generic-only; write-root-only retry разрешён только для already-rich manifests, иначе retry сохраняет repo roots и откатывает `write_root`, если fidelity не улучшилась
   - artifact-repair и provider retry разведены: invalid manifest получает один explicit artifact-repair retry с canonical manifest skeleton; semantic stdout parse больше не является success surface
   - `qwen-code` для `init.step1.collect` / `refresh.step1.collect` имеет internal stall watchdog в двух фазах:
     - `pre_artifact`: если process жив, но до первого authored artifact одновременно молчат stdout/stderr и мутации `write_root`, runtime принудительно завершает provider process и запускает один forced fresh retry до общего step timeout
     - `post_artifact`: если в `write_root` уже есть `shard-pack-manifest.json` + authored docs, но provider перестал писать в stdout/stderr и перестал мутировать `write_root`, runtime принудительно завершает provider process, пишет diagnostic events (`runtime task stalled before artifacts` / `runtime task stalled after artifacts` / `retry scheduled` / `retry completed`) и запускает forced artifact-repair retry до step timeout
   - transcript outputs с provider transport/API failures (например `[API Error: ... SSL ...]`) не считаются generic `runtime_contract_failed`: runtime сохраняет raw stdout/stderr и классифицирует их как `runner_unavailable`
   - collect step не считается успешным, если после единственной artifact-repair попытки `shard-pack-manifest.json` остаётся missing/invalid/skeletal; такой случай поднимается как runtime contract failure (`runtime_contract_failed`)
   - collect contract требует полного `semantic` block в `shard-pack-manifest.json` (`coverage/questions/entities/edges/findings`) и repo-specific citation surface; generic-only `cite.runtime-summary` допустим только вне multi-document refresh evidence collapse
   - `init.step0.constitution`, `init|refresh.step2.asis_docs` и `init|refresh.step4.proposals` проходят provider-agnostic required-artifact gate: runtime принимает шаг только если draft manifest валиден и все referenced draft files существуют под `draft_final_root`
   - runtime draft manifest contract для `step0/2/4` (`version=1`, `run_id`, `step_id`, `step_contract`, `agent_role`, `outputs[]`) вынесен в shared internal source of truth и используется и writer-ами, и validator-ами без дублирования структур
   - validators для collect manifests и runtime draft manifests read-only по умолчанию: hidden filesystem mutation внутри validation не допускается
   - filesystem reconciliation разрешён только как явная runtime repair/canonicalization стадия до финальной validation; сама validation лишь проверяет manifest contract и наличие referenced files
   - active repair surface зафиксирована internal registry и ограничена двумя safe drift paths: artifact-root-relative `documents[].path` normalization и draft-root reconcile только для уже записанных canonical draft files в явной repair stage
   - `qwen` для draft-only шагов (`step0/2/4`) валидирует required draft artifacts до возврата в orchestrator и делает один constrained artifact-repair retry (`write_root + draft_final_root`) вместо silent acceptance legacy draft schemas
   - `qwen` для draft-only шагов также имеет post-artifact stall recovery: если canonical draft manifest и draft files уже появились, но provider перестал писать в stdout/stderr и перестал мутировать `write_root`/`draft_final_root`, runtime принудительно завершает process и запускает один constrained artifact-only retry
   - `claude-code`, `qwen-code` и `codex-code` используют shared provider-agnostic step-policy/prompt layer для required artifacts, retry bans и explicit negative rules; provider-specific остаются только command/process execution, pipe monitoring, transcript extraction и provider failure classification
   - headless provider scope включает `arch-workspace` и resolved repo directories для текущих `repo_scope/repo_scopes`, чтобы provider видел source evidence из реальных checkout-ов
   - command overrides:
     - `ACP_CLAUDE_CMD` (default `claude-code`)
     - `ACP_QWEN_CMD` (default `qwen`)
     - `ACP_CODEX_CMD` (default `codex`)

6) **Workspace (`internal/workspace`)** *(implemented baseline)*
   - реализует/валидирует структуру central `arch-workspace` (Variant 2)
   - парсит `workspace.yaml`
   - валидирует manifest по `schemas/workspace.schema.json`
   - поддерживает `docs/imports/index.yaml` как metadata index для imported docs
   - поддерживает repo entries с `path` или `git_url` + optional `ref`
   - поддерживает optional `repos[].analysis.role` (`backend|frontend|mixed|unknown`) для runtime repo-selection policy
   - поддерживает optional persisted runtime profile в `runtime.profile` (`timeouts + execution`, см. `WORKSPACE_SPEC`)
   - verify `ref` для `path` source использует fallback (`ref` -> `origin/ref` -> `refs/remotes/origin/ref`) и выдаёт warning при `HEAD` mismatch
   - clone/fetch для `git_url` выполняет на той же машине через локальный `git` и текущий user/runner auth context
   - git_url cache key использует только `slug(repo.name)+hash(git_url)` (legacy slug-only cache fallback удалён из active behavior)
   - не хранит отдельные credentials внутри ACP
   - safe path joins (никогда не читаем вне workspace root)
   - `POST /api/workspace/validate` даёт pre-run readiness diagnostics по layout (`missing/will create on run`, `not_dir`, `unreadable`)
   - git helpers (shell out в `git`)

7) **Model store (`internal/model`)** *(implemented baseline, derived layer)*
   - entity-per-file YAML
   - stable IDs + aliases
   - детерминированная slug normalization и collision policy
   - apply semantic snapshots
   - хранит derived model view для диаграмм и deterministic projections

8) **Reports (`internal/reports`)** *(implemented baseline)*
   - primary narrative surfaces в docs-first path приходят из runtime-authored staged docs
   - compiler layer используется только как deterministic renderer/materializer для derived technical surfaces
   - в derived layer генерирует evidence-first C4 Mermaid set: `Context`, `Container`, per-service `Component`, per-service `Code`
   - materialize-ит индекс диаграмм `reports/diagrams/index.md` для UI filtering/open flow
   - формирует `reports/changelog/*` по итерациям

9) **Runtime Profile API (`internal/api`)** *(implemented baseline)*
   - `GET /api/runtime/timeouts`: persisted + effective + source
   - `PUT /api/runtime/timeouts`: partial update persisted timeout profile, write-through в `workspace.yaml`
   - `GET /api/runtime/execution`: persisted + effective + source
   - `PUT /api/runtime/execution`: partial update persisted execution profile, write-through в `workspace.yaml`
   - `GET /api/runtime/profile`: aggregate view `timeouts + execution + step_providers`
   - active run не прерывается при изменении timeout settings; новые значения применяются к следующим run

## Agent Topology Artifacts (MVP)
- `charter/cards/domains/<domain-id>.md`
- `charter/cards/teams/<team-id>.md`
- `reports/agent-outputs/domains/<domain-id>.md`
- `reports/agent-outputs/architect/summary.md`
- `reports/changelog/<yyyy-mm-dd>-<iteration-id>.md`

## Baseline Bundle Artifacts (MVP)
- `skills/subagents.yaml`
- `skills/service-inventory/*`
- `skills/interface-extraction/*`
- `skills/integration-mapping/*`
- `skills/datastore-mapping/*`
- `skills/cicd-mapping/*`
- `skills/ownership-coverage/*`
- `skills/findings/*`
- `skills/proposals/*`

## Pipeline (MVP)
0) Конституция (charter)
1) Collect context -> shard-authored dossier packs + shard manifests
2) As-is docs -> agent-authored drafts + compiler-normalized staged final doc assembly + indexes
3) Findings -> validator verdict over staged final set
4) Proposals -> agent-authored drafts + automatic promotion + derived model rebuild

On-demand capability:
- Q&A агент использует `charter/cards + model + reports + docs/imports`; в beta доступен как internal service + CLI `acp qa` без публичного API endpoint.

Execution modes:
- local interactive: UI + local process
- local/batch: CLI without UI
- GitHub/GitLab CI: required integration surface через тот же batch mode внутри job/runner, запускаемый из webhook-triggered workflow и/или manual pipeline button, без hosted control plane
- optional internal API trigger: только для trusted local/private long-running deployment

## Deterministic scope (beta baseline)
- Stable artifacts (при одинаковом input + одинаковом наборе artifact fixtures):
  - `charter/`
  - `model/`
  - `reports/as-is/`
  - `reports/findings/`
  - `reports/coverage/`
  - `reports/agent-outputs/`
  - `proposals/`
- Run-specific artifacts:
  - `reports/changelog/*`
  - `reports/taskruns/*`
  - `reports/taskruns/run-history.json`
  - run status registry (`/api/pipeline/runs/*`)

## Follow-up boundary
- `POST /api/qa/ask` остаётся post-beta slice (Epic 11) до отдельного release-требования.

## Progress tracking
- Каноническая матрица stakeholder-статусов: `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
- `docs/PLANS.md` содержит инженерный ExecPlan и синхронизированный operational mirror статусов.
