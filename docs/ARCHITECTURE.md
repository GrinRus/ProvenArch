# ARCHITECTURE.md (Go monorepo MVP)

Этот документ описывает целевую архитектуру реализации ACP для local-first MVP.
В текущем состоянии реализован runnable baseline: `workspace`/`TaskResult` foundations, рабочий `init|refresh` execution path, API endpoints `/api/*`, deterministic materialization `model/reports/proposals/changelog` и fake-runner default для required CI без live dependencies.

## Scope (MVP)
- Local-first: всё работает на машине разработчика
- Тот же entrypoint поддерживает non-interactive batch execution в GitHub/GitLab CI jobs
- Runtime (analysis): **headless multi-provider** (`claude-code` default, `qwen-code` optional) + deterministic `fake` baseline
- Реализация продукта: **Go backend/orchestrator + embedded React UI**
- Единая workspace-конвенция MVP: central `arch-workspace` git-репозиторий (Variant 2)
- Agent operating model MVP: идеи `1,2,3,5,7` (domain-first cards/agents/changelog/Q&A)
- Нет hosted режима и нет security/compliance enforcement в MVP
- Autodocs и Jira manager agents остаются post-MVP (Wave 1+)

## Компоненты
1) **Go entrypoint (`cmd/acp`)** *(implemented baseline)*
   - `init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>)` создаёт/обновляет `workspace.yaml`, bootstrap-ит fixed layout/baseline bundle и выполняет dry validation для первого старта
   - Раздаёт UI (embedded static assets из `ui/dist`)
   - Экспортирует API под `/api/*`
   - `serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path <path>]]` поднимает single-workspace-per-process service
   - `serve --auto-init` bootstrap-ит workspace manifest/layout при отсутствии `workspace.yaml`
   - bootstrap (`init-workspace`/`serve --auto-init`) автоматически делает `git init` для workspace root при отсутствии `.git`
   - startup для `serve` lenient: без блокирующего repo preflight; readiness diagnostics доступны через `/api/workspace/validate`
   - Поддерживает batch/non-interactive режим для CI jobs
   - `run --workspace <abs-path> --pipeline init|refresh [--refresh-mode incremental|full] [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]`
   - runtime selector process-scoped: `fake` default для required CI, `headless` opt-in
   - provider selector process-scoped: `--runtime-provider` > `ACP_RUNTIME_PROVIDER` > `claude-code`
   - timeout control process/workspace-aware:
     - persisted profile в `workspace.yaml.runtime.profile.timeouts`
     - effective precedence: `env > workspace > defaults`
     - canonical envs: `ACP_RUNTIME_*`, `ACP_PIPELINE_*`, `ACP_API_*`, `ACP_UI_*`
     - deprecated aliases поддержаны для обратной совместимости (`READY_TIMEOUT_SEC`, `UI_E2E_*`, full-run `ACP_FULL_RUN_PIPELINE_*`)
   - execution control process/workspace-aware:
     - persisted profile в `workspace.yaml.runtime.profile.execution`
     - effective precedence: `CLI > env > workspace > defaults`
     - CLI overrides: `--execution-strategy`, `--max-parallel-tasks`, `--failure-policy`
     - repo filtering policy: `runtime.profile.execution.repo_selection` (`all|backend_only`) + `repos[].analysis.role`
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
   - UI разбит на явные секции `Setup / Baseline / Runs / Results`
   - Показывает run dashboard (queued/running/succeeded/failed), включая завершённые run'ы из persisted history
   - При bootstrap авто-выбирает newest active run (`queued/running`), иначе первый run в history; после ручного выбора run auto-switch не выполняется
   - Если выбранный run исчезает из history (например, retention/restart race), UI очищает stale `Run status`/logs для этого run и не auto-switch-ится на другой run
   - Показывает `Run status` выбранного run с полным warnings list (`RunInfo.warnings`), `error_code` и `error`
   - Показывает `Runs: Logs` для выбранного run (`timestamp/level/step/domain/message`) с переключателем `line | line+fields` и quick actions `Copy logs`, `Download logs`, `Open taskrun artifact`
   - Поддерживает `Cancel selected run` для active run через `POST /api/pipeline/runs/<run_id>/cancel`
   - Runtime Timeouts settings panel:
     - load/save/reset через `GET/PUT /api/runtime/timeouts`
     - показывает persisted/effective/source для каждого timeout поля
   - Runtime Execution settings panel:
     - load/save через `GET/PUT /api/runtime/execution`
     - показывает persisted/effective/source для strategy/parallelism/failure/discovery
   - live e2e poll timeout-ы берутся из effective config (`/api/runtime/timeouts`) с env override
   - Критичные UI-контролы для live e2e снабжены стабильными `data-testid` (`validate/run/status/artifacts/logs`)

3) **Orchestrator (`internal/orchestrator`)** *(implemented baseline)*
   - Step registry (шаги init pipeline)
   - Step 0 materialization читает persisted wizard contract `charter/wizard/step0-contract.json`
   - при missing/invalid wizard contract применяется deterministic baseline fallback и warning фиксируется в run diagnostics
   - baseline bundle seeding выполняется create-if-missing, без перезаписи пользовательских правок
   - Готовит ContextPack/PromptPack
   - Загружает baseline bundle agents/skills/prompts из workspace
   - Работает с единым central workspace (`arch-workspace`) как корнем артефактов MVP
   - Валидирует `workspace.yaml` по `schemas/workspace.schema.json`
   - Разрешает repo sources (`path`/`git_url`) в локальные checkout перед анализом через системный `git` текущего пользователя/runner
   - Оркестрирует service-first pipeline:
     - `step1.service_inventory` (deterministic planner)
     - `step2.service_collect` (runtime fan-out по service shards)
     - `step3.asis_docs` (compiler)
     - `step4.service_findings` (runtime fan-out по service shards)
     - `step5.global_review` (single runtime aggregation task per run)
     - `step6.proposals` (compiler/changelog)
   - Service inventory planner:
     - marker-based service roots + leaf-pruning + fallback `.`
     - deterministic chunking для больших сервисов (`>500 files` или `>8MB` → chunks до `200 files`/`3MB`, cap `8`)
     - artifacts: `*-service-inventory-plan.json`, `*-service-inventory-summary.json`, stable snapshot `service-inventory-latest.json`
   - Refresh modes:
     - default `incremental` (previous snapshot + git diff + untracked/modified)
     - explicit `full` через CLI/API/UI
     - per-repo fallback to full при недоступном snapshot/git + warning diagnostics
   - Runtime шаги `step2/step4` сохраняют raw taskruns per service shard в `reports/taskruns/*-service_collect-domain-service-*.json` и `*-service_findings-domain-service-*.json`
   - Domain outputs остаются source-of-truth в `reports/agent-outputs/domains/*`, строятся детерминированно из service artifacts
   - Проверяет согласованность canonical domain card (`filename` vs `- id`) и repo_scope-resolver (`declared -> slug fallback -> empty`) с high-priority questions на mismatch/unknown/excluded
   - Runtime sharding planner (heuristics/semantic) materialize-ит deterministic shard-plan artifacts `reports/taskruns/*-shard-plan*.json` и shard-summary artifacts `reports/taskruns/*-shard-summary*.json`
   - Scheduler поддерживает `sequential|parallel` execution с worker-pool (`max_parallel_tasks`) и `fail_fast|best_effort` failure-policy
   - При `best_effort` downstream шаги продолжаются на partial model, но итог run фиксируется как `failed` с `error_code=run_partial_failed`
   - Вызывает runtime adapter
   - Валидирует TaskResult (schema)
   - Нормализует legacy `add_question` / `set_coverage` в canonical top-level form
   - Применяет semantic guard для refresh-taskruns: фильтрует placeholder/off-topic артефакты в `refresh.step2.service_collect`, добавляет deterministic fallback finding/edge в `refresh.step4.service_findings`, канонизирует/дедуплицирует coverage/question semantics
   - Применяет changeset к модели workspace
   - Не auto-create/rename canonical domain/team cards
   - Триггерит генерацию отчётов
   - Поддерживает async run coordination: single active run + debounce queue (`last event wins`)
   - Поддерживает управляемую отмену run:
     - pending run в debounce queue отменяется immediate (`failed`, `error_code=run_canceled`)
     - active run отменяется cooperative через `context cancel` (`failed`, `error_code=run_canceled`)
   - На старте сервиса делает reconciliation stale persisted run (`queued/running`) в `failed` с `error_code=run_reconciled_after_restart`
   - Ведёт persisted run history в `reports/taskruns/run-history.json` (versioned index, retention 500)
   - Ведёт run-level logs в `reports/taskruns/logs/<run_id>.ndjson` с cursor query API (`GET /api/pipeline/runs/<run_id>/logs`)
   - При runtime/parse fail логирует structured diagnostics snippets (`stdout_snippet`/`stderr_snippet`) в `RunLogEntry.fields` (sanitize + truncate)
   - Пробрасывает `TaskResult.warnings` в run diagnostics (`RunInfo.Warnings`) и логирует warning events
   - Runtime step execution:
     - `executeRuntimeTask` выполняет runner под `context.WithTimeout(step_timeout_sec)`
     - heartbeat-log `runtime task heartbeat` публикуется раз в `heartbeat_sec`
     - timeout/cancel причины добавляются в error message без изменения `error_code` контракта
   - Materialize-ит per-run quality summary `reports/taskruns/<run_id>-quality.json` (signal metrics/runtime versions)
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
   - headless providers: `claude-code` (`internal/runtime/claudecode`) и `qwen-code` (`internal/runtime/qwencode`)
   - общий runtime layer + provider factory: `internal/runtime/runtime.go`, `internal/runtime/providers/factory.go`
   - каждый provider возвращает TaskResult JSON; parse failures классифицируются как `runner_parse_failed`
   - command overrides:
     - `ACP_CLAUDE_CMD` (default `claude-code`)
     - `ACP_QWEN_CMD` (default `qwen`)

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
   - git_url cache key использует `slug(repo.name)+hash(git_url)`; legacy slug-only cache path поддержан через fallback warning
   - не хранит отдельные credentials внутри ACP
   - safe path joins (никогда не читаем вне workspace root)
   - `POST /api/workspace/validate` даёт pre-run readiness diagnostics по layout (`missing/will create on run`, `not_dir`, `unreadable`)
   - git helpers (shell out в `git`)

7) **Model store (`internal/model`)** *(implemented baseline)*
   - entity-per-file YAML
   - stable IDs + aliases
   - детерминированная slug normalization и collision policy
   - apply changeset operations
   - хранит service/API/datastore/external system model артефакты

8) **Reports (`internal/reports`)** *(implemented baseline)*
   - генерирует `reports/as-is/*`, включая per-service dossiers, integrations/datastores/ci-cd views
   - сохраняет `reports/coverage/*` для unknowns/questions
   - сохраняет `reports/agent-outputs/*`
   - формирует `reports/changelog/*` по итерациям

9) **Runtime Profile API (`internal/api`)** *(implemented baseline)*
   - `GET /api/runtime/timeouts`: persisted + effective + source
   - `PUT /api/runtime/timeouts`: partial update persisted timeout profile, write-through в `workspace.yaml`
   - `GET /api/runtime/execution`: persisted + effective + source
   - `PUT /api/runtime/execution`: partial update persisted execution profile, write-through в `workspace.yaml`
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
1) Service inventory (planner + snapshot)
2) Service collect (runtime fan-out)
3) As-is docs
4) Service findings (runtime fan-out)
5) Global review (single runtime aggregation)
6) Proposals (improvements)

On-demand capability:
- Q&A агент использует `charter/cards + model + reports + docs/imports`; в beta доступен как internal service + CLI `acp qa` без публичного API endpoint.

Execution modes:
- local interactive: UI + local process
- local/batch: CLI without UI
- GitHub/GitLab CI: required integration surface через тот же batch mode внутри job/runner, запускаемый из webhook-triggered workflow и/или manual pipeline button, без hosted control plane
- optional internal API trigger: только для trusted local/private long-running deployment

## Deterministic scope (beta baseline)
- Stable artifacts (при одинаковом input + recorded runner):
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
