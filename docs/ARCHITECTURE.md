# ARCHITECTURE.md (Go monorepo MVP)

Этот документ описывает целевую архитектуру реализации ACP для local-first MVP.
В текущем состоянии реализован runnable baseline: `workspace` foundations, docs-first staged runtime pipeline для `init|refresh`, API endpoints `/api/*`, validator-gated promotion в `reports/*`/`proposals/*`, derived model materialization `model/*` и fake-runner default для required CI без live dependencies.

## Scope (MVP)
- Local-first: всё работает на машине разработчика
- Тот же entrypoint поддерживает non-interactive batch execution в GitHub/GitLab CI jobs
- Runtime (analysis): **headless multi-provider** (`claude-code` default, `qwen-code` optional, `codex-code` release peer) + deterministic `fake` baseline
- Реализация продукта: **Go backend/orchestrator + embedded React UI**
- Distribution: public GitHub Releases single-binary `acp` для macOS/Linux `amd64/arm64` под Apache-2.0; `acp version` показывает release metadata; Homebrew tap planned after first stable release; Docker не является primary happy path в MVP
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
   - `serve` без `--workspace` поднимает loopback launcher/onboarding UI: workspace выбирается или создаётся в UI, затем server session attaches ровно один active workspace без restart процесса
   - `serve --workspace <path>` сохраняет direct-mode single-workspace-per-process local API+UI service для scripts, CI, live E2E и опытных пользователей
   - launcher API поддерживает `/api/onboarding/status`, `/api/onboarding/workspace`, `/api/onboarding/runtime`, `/api/onboarding/path-suggestions` и локальное forget действие для Recent workspaces; workspace-bound endpoints до выбора workspace возвращают `428 workspace_not_selected`
   - Recent workspaces хранятся как local-only user config metadata вне `workspace.yaml` и не являются переносимым workspace contract
   - `serve --auto-init` bootstrap-ит workspace manifest/layout при отсутствии `workspace.yaml`
   - bootstrap (`init-workspace`/`serve --auto-init`) автоматически делает `git init` для workspace root при отсутствии `.git`
   - startup для `serve` lenient: без блокирующего repo preflight; readiness diagnostics доступны через `/api/workspace/validate`
   - Поддерживает batch/non-interactive режим для CI jobs
   - `run` выполняет deterministic `init|refresh` pipeline в local/batch/non-interactive execution
   - `qa` даёт deterministic compatibility ответы по артефактам workspace; UI target Ask использует async runtime-backed Q&A runs
   - `doctor` выполняет read-only readiness checks для local install/workspace/repo/runtime/UI; CLI exit codes: `0` ready, `1` user-fixable issues, `2` invalid flags/internal request error
   - `version` / `--version` печатает build metadata (`version`, `commit`, `built`) для проверки release binary без workspace; UI получает те же runtime metadata через `GET /api/system/version`, доступный до выбора workspace
   - runtime selector process-scoped: `fake` default для required CI, `headless` opt-in
   - global provider selector остаётся process-level fallback: `--runtime-provider` > `ACP_RUNTIME_PROVIDER` > `claude-code`
   - effective provider resolution внутри run step-scoped: `workspace.yaml.runtime.profile.steps.<step>.provider` переопределяет global fallback только для выбранного шага
   - timeout, execution и permission control остаются process/workspace-aware: persisted профиль живёт в `workspace.yaml.runtime.profile.*`, а точные precedence/API surfaces удерживаются в `docs/spec/API_SPEC.md` вместо дублирования здесь
   - Используется как локально, так и из SCM-triggered pipeline jobs/manual buttons
   - Internal API trigger остаётся optional trusted-mode capability, а не обязательной CI/CD поверхностью
   - Раздаёт embedded UI shell и API в одном процессе `acp serve`

2) **UI (`ui/`)** *(operator console shell)*
   - React + TypeScript + Vite
   - Dev: `npm run dev` с proxy на backend
   - Prod: `npm run build` → `ui/dist` встраивается в Go бинарь
   - Live browser e2e: Playwright optional smoke (`ui/e2e/live-flow.spec.ts`, `npm run e2e:live --prefix ui`)
   - UI first-run entrypoint использует pre-console `OnboardingShell`, а не девятую product stage: `Workspace -> Sources -> Runner -> Ready`, затем переход в Console V2
   - UI shell организован как Proven Arch console: top health strip (actual build/version metadata, workspace path, repo count, runtime/provider, permission mode, Git publication state), product-flow rail `Source / Readiness / Charter / Analysis / Review / Proposals / Ask / Publish`, центральная рабочая область, right inspector (`Next action`, blockers, evidence refs, workspace health, runtime safety, Git publication) и bottom activity drawer для logs/events
   - Guided setup поддерживает multi-repo (`repos[]`) с add/remove rows и optional `ref`; Source показывает repo table с name/source/ref, validation state и явным advanced-only статусом для analysis include/exclude
   - Readiness показывает summary cards для workspace, repositories, runtime provider, permissions и artifacts, а также compact runtime profile summary перед advanced settings
   - Показывает repo overview в validate surface: `resolved_repos` + diagnostics, сгруппированные по repo
   - `Charter` показывает wizard summary, domain/team card overview, baseline prompt bundle status, explicit partial state для missing card artifacts и editor для baseline bundle artifacts (`charter/*`, `skills/*`, prompt packs, `skills/subagents.yaml`)
   - `Analysis` показывает run mission control, canonical `step0..step4` timeline, shard/log-derived table, warning/error drilldown и existing pending permissions/run history controls without adding backend contracts
   - `App.tsx` остаётся route shell, а крупные sections вынесены в dedicated stage panels (`SourceStagePanel`, `ReadinessStagePanel`, `CharterStagePanel`, `AnalysisStagePanel`, `ReviewStagePanel`, `ProposalsStagePanel`, `AskStagePanel`, `PublishStagePanel`); shell components отвечают за top bar, rail, inspector и activity drawer
   - setup/baseline/wizard/git state и actions остаются за facade `useWorkspaceSetup`, но внутри разделены на `useManifestEditor`, `useBaselineEditor`, `useWizardEditor` и `useGitActions`; runtime settings живут в отдельном hook, а run explorer разделён на `useRunSelection`, `useRunPolling`, `useRunActions`, `useRunArtifacts` и `useRunLogs`
   - Runtime profile (`timeouts` + `execution` + `permissions`) доступен в `Readiness -> Advanced runtime settings`, включая effective per-step providers; это не отдельная primary stage
   - Показывает run dashboard (queued/running/succeeded/failed), включая завершённые run'ы из persisted history
   - При bootstrap авто-выбирает newest active run (`queued/running`), иначе newest completed run в history
   - При bootstrap UI остаётся на `Source` для пустого workspace, но автоматически открывает `Analysis` для active run и `Review` для выбранного completed run с уже доступными artifacts
   - Если выбранный run исчезает из history и есть новый доступный run, UI переключается на него; если history временно пуста, но status endpoint ещё возвращает выбранный run, UI сохраняет текущий selection и не делает ложный auto-switch
   - Показывает `Run status` выбранного run с полным warnings list (`RunInfo.warnings`), `error_code` и `error`
   - Bottom activity drawer показывает compact recent logs для выбранного run (`timestamp/level/step/domain/message`) с dual-view `event timeline | raw agent stream | all`, переключателем `line | line+fields`, collapsed runtime execution artifact refs и quick actions `Copy logs`, `Download logs`
   - `Review` объединяет evidence tabs, grouped artifact explorer, markdown/Mermaid preview, coverage/open-question/trust summary и artifact-derived Domain Map на основе `model/entities/*`, `model/edges/*`, `reports/agent-outputs/domains/*` с explicit partial states для sparse model data; `Proposals` показывает review room для proposal/changelog artifacts с package list, preview/evidence/changelog/diff tabs, quality blockers и publication path
   - `Ask` stage вызывает async `POST /api/qa/runs`, poll-ит `GET /api/qa/runs/<run_id>`, загружает history через `GET /api/qa/runs?limit=20` и показывает selected answer, runtime/provider identity, confidence, citations, unresolved assumptions, related-entity partial state and read-only safety/audit artifact links; optional frontend UX smoke покрывает этот flow только как diagnostic evidence; legacy deterministic `POST /api/qa/ask` остаётся compatibility endpoint для CLI/API consumers
   - `Publish` stage показывает Git Review Room: folder-level artifact summary from selected-run refs, selected artifact preview, explicit partial state for unavailable line-level Git diff, publish gate/checklist, commit plan, prepared commit-message copy action and existing commit/proposal-branch mutations
   - Поддерживает `Cancel selected run` для active run через `POST /api/pipeline/runs/<run_id>/cancel`
   - Runtime Timeouts settings panel:
     - load/save/reset через `GET/PUT /api/runtime/timeouts`
     - показывает persisted/effective/source для каждого timeout поля
   - Runtime Execution settings panel:
     - load/save/reset через `GET/PUT /api/runtime/execution`
     - показывает persisted/effective/source для strategy/parallelism/failure/discovery
   - Runtime Permissions settings panel:
     - load/save/reset через `GET/PUT /api/runtime/permissions`
     - показывает persisted/effective/source для `trusted_full_access|managed` и `fail_fast|ui`
   - `Analysis` показывает pending runtime permission requests выбранного run вместе с `decision/rule_id`; approve/deny broker остаётся отдельным будущим slice
   - runtime profile patch validation/merge/manifest rewrite живёт в shared internal package `internal/runtimeprofile`, а API handlers остаются только HTTP adapter layer
   - live e2e poll timeout-ы берутся из effective config (`/api/runtime/timeouts`) с env override
   - Критичные UI-контролы для live e2e снабжены стабильными `data-testid` (`validate/run/status/artifacts/logs`)
   - First-run flow начинается до Console V2: пользователь выбирает/создаёт workspace или открывает Recent workspace, добавляет один или несколько repos через существующее `workspace.yaml.repos[]`, выбирает runner (`fake` recommended, live providers explicit opt-in), проходит readiness summary и только затем входит в stages `Source / Readiness / Charter / Analysis`
   - После onboarding `Source` остаётся editable для изменения repos/imports; GitHub/GitLab URL является default entry, local folder остаётся supported mode, raw `workspace.yaml` editor спрятан в Advanced, readiness checklist берётся из `GET /api/system/doctor`, первый `init` запускается кнопкой `Run first analysis`

3) **Orchestrator (`internal/orchestrator`)** *(implemented baseline)*
   - Step registry (шаги init pipeline)
   - Step 0 support-artifacts materialization читает persisted wizard contract `charter/wizard/step0-contract.json`
   - при missing/invalid wizard contract применяется deterministic baseline fallback только для support artifacts, а warning фиксируется в run diagnostics
   - baseline bundle seeding выполняется create-if-missing, без перезаписи пользовательских правок; support-only bundle не пишет canonical `skills/subagents.yaml`, поэтому source of truth для него остаётся validated `constitution-draft.json`; constitution first-action `baseline-subagents.yaml` обязан быть валидным `skills/subagents.yaml` YAML bundle, а не markdown-заглушкой
   - Готовит ContextPack/PromptPack
   - Загружает baseline bundle agents/skills/prompts из workspace
   - workspace prompt packs подключаются к runtime prompt composition как editable content layer по фиксированному merge order: provider header -> task-specific first-action artifact command -> artifact-only/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer; содержимое prompt pack не может ослаблять enforced contract rules. Editable prompt pack layer применяется к `step0.constitution`, `step1.collect`, `step3.findings` и `step4.proposals`; `step2.asis_docs` остаётся enforced-policy-only и не имеет отдельного editable `as-is` prompt pack.
   - `internal/orchestrator/orchestrator.go` остаётся entry shell/pipeline glue; `pipelineExecution` state сгруппирован во вложенные run-progress/artifact-registry/runtime/quality/semantic-docflow/draft buckets, а run finalization, step handlers, artifact registry methods, service run-registry/history lifecycle и semantic/card enrichment вынесены в dedicated package files
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
   - Collect validation read-only: ACP не нормализует provider manifest после выполнения и не логирует compatibility rule ids
   - Collect recovery имеет два слоя: non-silent no-artifact collect может выполнить одну provider-authored pair-recovery попытку (`suggested overview doc + shard-pack-manifest.json`), а collect с non-bootstrap authored docs и missing manifest или scaffold-only semantic manifest сначала выполняет deterministic manifest recovery из уже provider-authored markdown docs и bounded `repo/path` evidence; provider manifest-only repair остается для structural-invalid existing manifest и fallback после failed runtime recovery. Оба пути пишут ровно `write_root/shard-pack-manifest.json` с diagnostic `collect_manifest_runtime_recovery` для runtime path. Unchanged seed/bootstrap authored docs не маскируются repair success; write-set guards запрещают любые лишние файлы
   - Сохраняет raw runtime execution metadata и shard summaries в `reports/taskruns/*` для recovery/auditability
   - Runtime sharding planner (heuristics/semantic) materialize-ит deterministic shard-plan artifacts `reports/taskruns/*-shard-plan*.json` и shard-summary artifacts `reports/taskruns/*-shard-summary*.json`; internal `ShardPlanner` принимает workspace manifest + resolved repo paths + execution profile и возвращает plans/warnings/semantic graph без scheduling side effects
   - shard-plan публикует полный неперекрывающийся coverage partition repo через `path_scopes` (directory/file scopes); для больших repo применяется structural coalescing по filesystem ancestry, module marker leaf shards внутри крупных top-level dirs сохраняются только пока итоговый plan не превышает `maxAutoShardsPerRepo`, а excess top-level groups детерминированно merge-ятся в bounded buckets; repos только с root marker (`pyproject.toml`, `pom.xml`, etc. в корне) планируются как root-file group + top-level directory shards, а не как один `"."`
   - Per-shard persistence crash-safe: shard-summary materialize-ится сразу со status=`pending`; после validated runtime execution metadata internal `runtime-execution.json` пишется до `apply`, shard переходит в `checkpointed`, после успешного `apply` — в `succeeded`; runtime/apply failure фиксируется как `failed` без ожидания конца шага
   - Internal shard-summary contract: `taskrun_path` обязателен для `checkpointed/succeeded`; он должен ссылаться на persisted `runtime-execution.json` с `shard_id/repo_scopes/path_scopes`
   - Internal shard-plan/shard-summary artifacts materialize-ят non-empty `meta.runtime.name/meta.runtime.version`, чтобы internal batch/contract checks не трактовали их как runtime-name drift
   - Scheduler поддерживает `sequential|parallel` execution с worker-pool (`max_parallel_tasks`) и `fail_fast|best_effort` failure-policy; internal `ShardScheduler` отвечает за dispatch ordered runtime results, `ShardSummaryStore` держит summary/checkpoint persistence, а semantic/model apply остаётся в orchestration coordinator после scheduler result. Ownership физически разделён по файлам: coordinator, scheduler, summary store, artifact persistence и planner.
   - При `best_effort` partial shard failures итог run фиксируется как `failed` с `error_code=run_partial_failed`; если `step1.collect` становится `partial` или `unusable`, live downstream runtime для `step2/3/4` не вызывается, markdown artifacts (`as-is/findings/coverage/proposals/agent-outputs`) materialize-ятся в `report_mode=incomplete` с явным banner/triage-only wording, а batch/report primary class остаётся collect-level `runtime_flow_failed`; provider-class details из failed shards (`runner_unavailable`/`runtime_contract_failed`) остаются secondary evidence и не переписывают root cause
   - Вызывает runtime adapter через `StepRunnerResolver` с per-provider cache/preflight внутри одного run
   - `step0..step4` становятся agent-first шагами, но runtime получает только staged surfaces (`write_root`, `draft_final_root`, `read_context_roots`, `step_contract`, `expected_artifacts`)
   - canonical publish для `step0/2/4` выполняется только из validated runtime draft artifacts через deterministic compile/publish path; direct orchestrator writer больше не является альтернативным source of truth
   - Собирает staged final doc set в `reports/taskruns/<run_id>/staging/final/`; internal docflow builder принимает `DocflowBuildInput` и возвращает `DocflowBuildResult` (`artifacts`, `citation-index`, `final-run-index`, `semantic snapshot`), после чего execution state mutates только в thin adapter/promotion path
   - Генерирует и валидирует `final-run-index.json` и `citation-index.json`
   - `final-run-index.json` и `citation-index.json` используют один deterministic `document_id` mapping: canonical staged document ids берутся из `manifest.Documents[*].id`, а не пересобираются независимо на citation/final-index сторонах
   - Runtime proposal/changelog draft outputs stage into the final doc set before promotion, so `final-run-index.json` represents both analysis reports and published `proposals/*` / `reports/changelog/*`
   - `citation-index.json.claim_ids` трактуются как global staged-final namespace; duplicate claim ids в validator scope детерминированно repair-ятся на index/reference уровне с shard suffix без semantic rewrite authored docs
   - Artifact ownership разделён явно: provider-authored artifacts включают normal shard manifests, runtime draft manifests/files и validator verdict; runtime-recovered collect manifests являются contract-recovery artifacts, derived только из provider-authored shard docs и bounded repo/path evidence, и помечаются в diagnostics; orchestrator-authored artifacts включают staged indexes, run logs/history, shard plans/summaries; compiler-derived artifacts включают `model/*`, diagrams и normalized report/proposal renderers после validator-gated promotion. `model/*` filenames are deterministic and bounded with a hash suffix when canonical ids exceed filesystem filename limits; full ids remain inside YAML.
   - Step 3 runtime primary output: `validator-verdict.json`
   - `validator-verdict.json` использует canonical contract с обязательными `version=1`, `run_id`, `generated_at`, `verdict`, `checked_paths`; validator findings сохраняют `title + description + provenance`, а observation evidence требует non-empty `repo/path`
   - staged semantic snapshot нормализует `evidence.repo` к логическому repo scope, сводит generated checkout-dir aliases и детерминированно дедуплицирует entity/edge/finding references до validator/promotion
   - Validator repair остаётся явной internal stage между load/parse verdict и финальной staged validation; repair write path остаётся atomic и не превращается в hidden mutation validation path
   - owner-gap остаётся visible signal в `coverage/findings/questions`, но owner-only residual после технических repair stages может быть reconciled из `FAIL` в `PASS` без скрытия самих findings/questions
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
   - Синхронный `acp run` использует signal-aware context (`SIGTERM`/`SIGINT`/`SIGHUP`) и terminal guard: после записи `running` любой error/panic best-effort переводит history в terminal `failed` с `finished_at`/`error_code`
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
   - Runtime repo scopes для all-repo шагов вычисляются напрямую из `workspace.yaml`; legacy repo-selection mode/summary artifact не входят в active contract
   - trusted batch/matrix harness не оставляет terminal-less child runs: если per-run `run-status.env` отсутствует или остаётся `running` после завершения child batch, outer reconciliation переводит его в terminal `failed` с `failure_reason=infra_incomplete_cycle`
   - trusted batch/matrix host preflight проверяет writable roots и минимальное свободное место (`E2E_MATRIX_MIN_FREE_KB`, default 5 GiB) для `E2E_TMP_ROOT`, `REPORTS_ROOT` и `MATRIX_ROOT`; low-disk blocker останавливает matrix как `operational_host_preflight_failed` до child batch
   - child batch публикует `batch-owner.env` heartbeat в `BATCH_ROOT`; stale `profile-status/*.json = running` без живого owner pid или со stale owner heartbeat reconciles-ится в terminal `failed/infra_incomplete_cycle`
   - terminal `validator verdict is FAIL` классифицируется как `runtime_flow_failed`; `runtime_contract_failed` остаётся только для active runtime artifact/manifest/required-output failures
   - generic `codex` plugin/Cloudflare/state-db warnings (`plugins/featured`, Cloudflare HTML, cache/state-db permission noise) считаются secondary telemetry и не должны сами поднимать `runner_unavailable`; default `codex-code` headless invocation disables plugin/app suggestion surfaces (`plugins`, `remote_plugin`, `plugin_sharing`, `apps`, `enable_mcp_apps`, `tool_suggest`, `skill_mcp_dependency_install`), uses `--ignore-user-config` + `--ignore-rules`, and runs with an auth-only isolated `CODEX_HOME` copied from the caller's `auth.json`/`installation_id`, so user config, MCP/plugins, app tools and rules are not part of ACP artifact-only tasks before model actions
   - batch report telemetry records semantic/artifact findings such as `analysis:cross-repo-missing` when multi-repo signal is weak; release execution verdicts do not fail on this artifact-truthfulness telemetry, and final acceptance is handled by `swe_artifact_quality_assessment_<matrix-id>.md`
   - multi-repo validator prompts write a first-action `validator-verdict.json` skeleton that already carries one cross-repo finding/question with repo/path provenance; single-repo validator skeletons remain empty except for technical `issues[]`
   - Run logs retention policy (TTL + max runs) запускается при старте сервиса, перед run и после run
   - (опционально) делает git commit

4) **Agent Topology (domain-first, baseline)**
   - Domain Analyst Agent (per domain)
   - Team overlay через `charter/cards/teams/*`
   - Architect Aggregator Agent (анализ outputs domain-агентов)
   - System Analyst Q&A capability: target UI flow is async runtime-backed `qa.ask` with agent role `system-analyst-qa`; deterministic workspace-backed `acp qa` + `POST /api/qa/ask` remain compatibility/fake baseline surfaces
   - Базовые skill/prompt bundles поставляются вместе с продуктом и versioned в workspace

5) **Runtime providers (`internal/runtime/*`)** *(implemented baseline)*
   - headless providers: `claude-code` (`internal/runtime/claudecode`), `qwen-code` (`internal/runtime/qwencode`) и `codex-code` (`internal/runtime/codexcode`); deterministic baseline: `fake` (`internal/runtime/fakeruntime`)
   - общий runtime layer + provider factory: `internal/runtime/runtime.go`, `internal/runtime/providers/factory.go`
   - default permission mode `trusted_full_access` сохраняет существующие full-access flags для live providers (`bypassPermissions`, `--yolo`, `danger-full-access`); opt-in `managed` отключает эти flags и включает orchestrator policy decisions
   - managed permission policy auto-approves только envelope операции: reads под `read_context_roots`, writes под `write_root`/`draft_final_root`; writes в analyzed repos/protected workspace paths, path traversal и symlink escape deny; shell/network/package install/unknown tool become `needs_user` and non-interactive fail-fast produces `runtime_permission_required`
   - каждый provider получает explicit staged-write contract (`artifact_root`, `write_root`, `draft_final_root`, `read_context_roots`, `step_contract`, `expected_artifacts`) и должен писать runtime-authored artifacts только внутрь `write_root`/`draft_final_root`; required artifact checks/writes должны использовать exact absolute paths, а не relative CWD targets
   - live headless providers считаются успешными только по valid required artifacts: normal process exit или controlled stop после появления валидных artifacts оба допустимы; missing/invalid artifacts классифицируются как `runtime_contract_failed`, кроме явных provider availability incidents
   - `shard-pack-manifest.json.documents[].path` — strict `artifact_root`-relative contract and must reference an existing provider-authored file under `write_root`; workspace-level prefixes (`reports/...`, `charter/...`, `proposals/...`), duplicated `artifact_root` prefix, absolute paths, missing file references и directory references считаются invalid collect artifact drift и не нормализуются ACP
   - `shard-pack-manifest.json.citations[].path` и `semantic.*.provenance.evidence[].path` — strict repo-relative regular-file evidence refs when repo roots are resolved; directory scopes may drive sharding/bounded reads, but directory-only citation/provenance refs, guessed `README.md`/`pom.xml` paths, and failure-only markdown that admits no repository evidence was read remain collect contract failures before downstream AOR synthesis
   - collect citations and semantic provenance are repo-evidence contracts, not free-form claims: `citations[].repo/path` and `semantic.*[].provenance.evidence[].repo/path` must resolve to the current repo scope/root and an existing relative path under that repo root. Missing or guessed evidence paths fail collect validation before downstream draft steps can promote misleading artifacts. Provider collect/repair prompts require file-level proof (`test -f`, `rg --files`, or portable `find ... -type f -print`); syntax-only JSON checks such as `jq empty` are not enough.
   - collect prompt начинает normal `init.step1.collect` / `refresh.step1.collect` с `COLLECT EVIDENCE-FIRST ARTIFACT PAIR`: первый filesystem work unit provider может содержать только bounded evidence read/list по repo entrypoint hints / assigned `path_scopes` (до 8 representative files, до 6000 bytes per file; oversized files truncate/skip with continue), затем direct literal shell heredoc/printf/tee write marker-free evidence-backed authored doc + `shard-pack-manifest.json`. До появления обоих targets provider не читает `reports/taskruns/**`, raw logs, sibling shards или archive docs, не делает analysis-only/status/progress prose, todo/planning, broad repo sweep, second read-only preflight, Ruby/Node/Python/Perl/awk/jq inline writer, generated template/source-code program или nested quote tricks. Если direct write command падает до создания обоих targets, provider сразу повторяет simpler direct literal write from observed evidence, а не ждет focused repair; normal collect не должен рассчитывать на focused repair как штатный success path. Task-specific manifest skeleton остаётся в prompt как schema/key/type guide, но copying unchanged skeleton invalid; `ValidateCollectManifestInRoot` reject-ит bootstrap-only docs/recovery fallback docs, interrupted temporary docs (`first bounded evidence read was attempted`, `initial artifact records only`, `will be repaired with concrete...`) и scaffold-only semantic (`contains`-only repo/shard + generic owner-gap). Root-file shards читают только перечисленные root files first и пишут evidence-backed root overview без recursive sweep. Для `init.step1.collect` / `refresh.step1.collect` runtime выполняет максимум одну collect pair-recovery попытку, когда provider оставил stdout/stderr diagnostics без authored artifacts, а также после exhausted fully silent zero-output fresh retry with no authored artifacts. Pair repair теперь write-first/evidence-bounded: provider не запускает отдельный read-only preflight, не отвечает планом/status/analysis-only текстом вроде “I have enough evidence” или “I am now writing”, не ставит собственный exact-phrase gate (`required = [...]`, `missing expected evidence`) до записи, не пишет temporary/recovery prose вроде “will be repaired”, и не добавляет semantic pre-write abort по counts/shape. Следующий item обязан быть одним простым filesystem command, который читает только listed evidence candidates/read roots (до 8 файлов, bounded prefix до 6000 bytes each), выводит claims из реально прочитанных snippets и пишет markdown + `shard-pack-manifest.json` до возврата; oversized candidates нужно читать prefix-only или пропускать с продолжением, а не abort-ить repair из-за `read file exceeds size limit`. Python f-strings, `.format(...)` template writers, generated Python source strings и nested quote tricks для markdown/JSON запрещены. Planned claims без observed evidence нужно удалить или зафиксировать как gap, а semantic sufficiency проверяет backend validation. Если pair repair успел записать non-bootstrap authored markdown, но stalled/failed before valid manifest, shared engine может перейти в manifest-only/runtime recovery вместо immediate terminal failure; если authored artifacts отсутствуют, collect pair exhaustion остается terminal provider/runtime failure. Если provider уже записал non-bootstrap authored docs в `write_root`, но `shard-pack-manifest.json` отсутствует или existing manifest failed only как scaffold-only semantic, shared engine применяет deterministic `collect_manifest_runtime_recovery` before provider repair; если manifest structural-invalid, или deterministic recovery не прошла validation, runtime делает максимум одну manifest-only repair попытку. Manifest-only repair starts with `COLLECT MANIFEST EVIDENCE-FIRST REPAIR` and `FIRST COLLECT MANIFEST REPAIR COMMAND`; the first provider-authored filesystem command must read bounded authored markdown/evidence and write the single allowed target `write_root/shard-pack-manifest.json` before returning. Evidence-packet-only output and status prose without a manifest write are no-op repair failures. Если manifest-only provider repair exhausted/stalled или вышел без валидного manifest после structural-invalid provider manifest, run остается `runtime_contract_failed`; deterministic runtime recovery не используется как post-repair fallback. Runtime recovery читает только existing non-bootstrap authored markdown under `write_root`, строит manifest с concrete extracted service/component/datastore entities, usage/dependency/configuration edges, runtime-recovery finding/question/coverage note и bounded repo/path evidence, пишет только `write_root/shard-pack-manifest.json`, затем снова запускает strict collect validation and emits explicit recovery diagnostics/runtime warnings so recovered manifests are not normal provider-authored success. Engine write-set guards разрешают только expected pair или `write_root/shard-pack-manifest.json`
   - collect repair запускается с narrow include dirs: текущий `write_root` плюс repo evidence roots. Broader ACP workspace, sibling `reports/taskruns`, raw logs и старые shard manifests намеренно не входят в repair read surface; embedded prompt contract является authoritative schema text, если `schemas/*` или `docs/spec/*` отсутствуют внутри runtime workspace
   - artifact-repair и provider retry разведены: collect pair repair, manifest-only collect repair, validator-verdict-only repair, draft-artifact repair и draft-artifact enrichment общие для `claude-code`/`qwen-code`/`codex-code`, а fresh-process retry остаётся provider policy; focused repair command construction централизован в `providercommon`, provider adapters задают только command transport; focused repair attempts have a bounded valid-artifact stop window, so a provider that keeps running after writing a valid repair artifact is stopped and accepted only through validation; deterministic collect manifest recovery живёт в shared engine, runs first for missing or scaffold-only semantic manifests with existing non-bootstrap authored docs, and is logged as runtime recovery, not provider authorship; exhausted structural invalid-manifest provider repair remains terminal `runtime_contract_failed`; bootstrap-only draft validation skips scaffold-style draft repair and routes directly to `draft_artifact_enrichment`, while structural missing/invalid draft manifests still use draft repair first; collect/validator contract wording canonically живёт в `artifactquality` и переиспользуется runtime prompt contract + baseline prompt generation; semantic stdout parse больше не является success surface
   - `claude-code`, `qwen-code` и `codex-code` используют общий artifact-only process engine в `internal/runtime/providercommon`: launch, stdout/stderr capture, process-group kill, deadline handling, raw diagnostics, activity monitor, controlled stop и artifact validation находятся в одном lifecycle path; orchestrator вызывает этот lifecycle через internal `RuntimeTaskExecutor`, после чего отдельно выполняет persistence/apply/promotion decisions
   - provider-specific остаётся только в thin adapters: command/args/stdin/workdir/include dirs, unavailable markers, activity policy и recovery policy; stdout/stderr transcript сохраняется как diagnostics и не является semantic success payload
   - shared activity monitor отслеживает pipe activity вместе с мутациями `write_root`/`draft_final_root`; pre-artifact silent/no-artifact hangs bounded для всех live adapters, post-artifact stop разрешён когда оба сигнала stale, валидные required artifacts уже можно принять без повторного provider call, а partial artifacts могут иметь более длинное provider policy grace window; `codex-code` collect steps use a 3-minute initial pre-artifact window because disabled plugin/app startup removes noisy pipe activity and medium-slice collect prompts can legitimately take longer than the shared 75s default before the first model action
   - `qwen-code` и `claude-code` policies дополнительно разрешают один fresh retry для missing/invalid artifacts; `qwen-code` также делает первый zero-output `pre_artifact` stall (`stdout=0`, `stderr=0`, no observed artifacts, authored file count `0`) retryable warning, а transient provider transport/API failure во focused collect-pair или draft-artifact repair (`[API Error: Premature close]`, `Connection error` with network socket/TLS disconnect, connection reset/closed, transient 5xx/529 stream errors) получает один bounded focused-repair retry; для normal draft steps (`step0/2/4`) `qwen-code` может выполнить controlled stop после bounded valid-artifact window, если manifest и referenced draft files уже валидны, но provider продолжает активно стримить или мутировать draft files; focused repair attempts всех providers используют короткий valid-artifact stop window после валидного repair artifact; collect repair controlled stop считает artifact валидным только через strict collect validation, включая referenced markdown checks, а не по parse-only manifest; exhausted silent/API retry на collect сначала пробует focused `collect_pair_repair`, если authored artifacts отсутствуют; exhausted pair repair with non-bootstrap authored markdown can chain into manifest-only/runtime recovery, while exhausted repair with no authored artifact remains terminal provider/runtime failure; `claude-code` сохраняет zero-output fail-fast для non-scoped шагов, но на `init.step0.constitution`, `init|refresh.step1.collect`, `init|refresh.step3.findings` и `init|refresh.step4.proposals` первый fully silent pre-artifact stall retry-ится как warning; exhausted non-collect silent retry остаётся `runner_unavailable`, а partial authored artifacts без валидного manifest/verdict/draft manifest после разрешенной manifest recovery остаются `runtime_contract_failed`
   - transcript outputs с provider transport/API failures (например `[API Error: ... SSL ...]` или `[API Error: Premature close]`) не считаются generic `runtime_contract_failed`: runtime сохраняет raw stdout/stderr и классифицирует exhausted no-artifact path как `runner_unavailable`
   - collect step не считается успешным, если после разрешённых focused collect recovery попыток `shard-pack-manifest.json` остаётся missing/invalid, `documents[].path` ссылается на отсутствующий authored doc или referenced authored doc сохраняет bootstrap marker/scaffold/seed prose; такой случай поднимается как runtime contract failure (`runtime_contract_failed`) и hard pass невозможен. Collect pair-recovery prompt теперь write-first/evidence-bounded: provider в первом filesystem command читает capped evidence из exact read roots/listed evidence candidates и пишет final evidence-backed suggested authored doc + `shard-pack-manifest.json`; marker-free seed/recovery fallback heredoc больше не является допустимым repair path. Marker-free `Recovery Summary`/`Recovery Bootstrap`/`Recovery Evidence Summary` scaffold reject-ится, чтобы repair не закреплял low-signal recovery-only docs. Invalid observed artifacts без fresh mutation останавливаются bounded partial-artifact window даже при активном stdout/stderr, поэтому stale bootstrap drafts или collect placeholders не ждут полного step timeout. Fully silent no-artifact qwen/claude/codex collect path после bounded retry запускает one-shot `collect_pair_repair`; если repair тоже stalled/invalid/noop without non-bootstrap authored markdown, run остаётся failed и не маскируется deterministic synthesis
   - collect contract требует полного `semantic` block в `shard-pack-manifest.json` (`coverage/questions/entities/edges/findings`) и repo-specific citation surface; generic-only `cite.runtime-summary` допустим только вне multi-document refresh evidence collapse
   - canonical collect vocabulary жёсткая: `coverage.observed`, `questions[*].text`, `edges[*].type/from/to`, object-shaped `provenance`, numeric `confidence`; legacy aliases (`covered_topics`, `question`, `relation`, `source`, `target`, finding `summary`/`inference`, string confidence, `evidence_citation_ids`, top-level `step_contract`, `compatibility`) reject-ятся schema/contract validation-ом без отдельного compatibility scanner
   - `step1.collect` не использует `reports/taskruns/**`, raw logs, старые manifests или archive docs как schema/reference surface; headless provider получает selected repo roots, `write_root` и explicit `read_context_roots`, collect cwd фиксируется на `write_root`, root-file shard prompt ограничивает анализ перечисленными root files и выбирает primary evidence в порядке `README` -> `Makefile` -> build/deploy manifests -> прочие root files, а repair path дополнительно исключает workspace-level taskrun history
   - runtime write audit в MVP остаётся detect-only: после live provider call ACP сравнивает protected workspace surfaces (`workspace.yaml`, `schemas/*`, `docs/spec/*`, `charter/*`) и analyzed repo working trees, surfacing `runtime_write_audit_unexpected_mutation` / `runtime_write_audit_repo_skipped` as run warnings/logs without failing, restoring, or sandboxing the provider; after-runtime repo status unavailability is reported as `runtime_write_audit_repo_skipped` with `status_unavailable_after_runtime`, not as a synthetic mutation path
   - если collect evidence стал `partial` или `unusable`, live runtime для `init|refresh.step2.asis_docs`, `init|refresh.step3.findings` и `init|refresh.step4.proposals` не вызывается: orchestrator детерминированно пересобирает incomplete staged docflow из persisted shard packs, помечает triage reasons (`*_skipped_due_to_partial_collect` или `*_skipped_due_to_unusable_collect`) и не позволяет downstream draft errors перезаписать collect root cause
   - `init.step0.constitution`, `init|refresh.step2.asis_docs` и `init|refresh.step4.proposals` проходят provider-agnostic required-artifact gate: runtime принимает шаг только если draft manifest валиден, все referenced draft files существуют под `draft_final_root`, а referenced markdown не остался unchanged first-action/recovery scaffold; для constitution `charter-overview.md` обязан быть evidence-backed, тогда как `baseline-subagents.yaml` может оставаться baseline bundle
   - focused validator repair может писать только `write_root/validator-verdict.json`; repair prompt начинается с command-first absolute heredoc skeleton, но `checked_paths` ссылается на staged final artifacts, а не validator `write_root`; `issues[]` принимает только `code`, `severity=error|warning`, `message` и optional `path/document_id/citation_id`, без legacy `id/title/description/rule_id/related_paths`
   - focused draft repair может писать только step manifest в `write_root` и draft files под `draft_final_root`; repair prompt начинается с command-first heredoc write-set для manifest + referenced draft files, задаёт exact absolute `write_root`/`draft_root` один раз и дальше использует только `"$write_root/..."` / `"$draft_root/..."` targets, чтобы provider не перепечатывал длинные slash-separated пути вручную. Если repair оставил unchanged bootstrap scaffold и остановился/завершился до valid artifacts, shared engine делает один `draft_artifact_enrichment` focused call без heredoc scaffold: provider читает bounded current-taskrun staging evidence и fresh rewrite-ит каждый referenced markdown draft evidence-backed содержимым. Для `step2/step4` enrichment include scope intentionally excludes whole headless workspace and whole source repo; доступен current `write_root`, current `draft_final_root`, current taskrun `staging/shards` and `staging/final` when present, а prompt требует читать manifests/indexes plus a small authored-doc sample instead of every shard doc. Для `step2.asis_docs` enrichment теперь command-first/write-first: provider должен выполнить filesystem command, который читает `asis-draft-manifest.json`, shard manifest summaries, optional final/citation indexes и максимум 6 high-signal shard docs, затем в той же command fresh overwrite-ит `overview.md`, `summary.md` и `architect-summary.md` до optional extra analysis, включая architecture surface, shard completeness/evidence density, coverage gaps и decision-ready operator summary. Analysis-only текст вроде “I have enough evidence” до fresh mutation не считается progress и остается `runtime_contract_failed`. Для `step4.proposals` enrichment дополнительно write-first: после manifest/index/finding evidence pass provider обязан fresh overwrite-нуть `proposal.md` и `changelog.md` до optional extra analysis, включая operator action, evidence refs, proposal/follow-up plan, touched surfaces, citations and residual gaps. Для этого focused call activity monitor игнорирует draft files, существовавшие до старта enrichment, пока provider не сделает fresh mutation, чтобы stale bootstrap не выглядел как post-artifact progress. Если enrichment не меняет любой referenced markdown или оставляет scaffold markers, шаг завершается `runtime_contract_failed` с причиной `draft_artifact_enrichment_noop_or_scaffold`. Writes outside manifest target или `draft_final_root` остаются `runtime_contract_failed`, а activity monitor учитывает draft files рекурсивно внутри `draft_final_root`
   - draft enrichment не должен пересобирать `outputs[]` из provider-invented aliases: существующие `outputs[].path`/`outputs[].canonical_path` сохраняются, а `outputs[].logical_path`, `target`, `output_path` и похожие поля остаются strict `runtime_contract_failed`; допустимые metadata updates ограничены top-level `summary`/`updated_at`
   - collect manifest-only repair теперь write-first: первый focused command читает bounded authored markdown/evidence и записывает единственный allowed target `write_root/shard-pack-manifest.json` до возврата; read-only preflight-only output и status prose без manifest write считаются no-op repair, чтобы provider не зависал после “I have enough evidence”
   - `init|refresh.step2.asis_docs` использует strict shared draft contract: `step_contract="as_is"`, required canonical outputs `reports/as-is/overview.md`, `reports/coverage/summary.md`, `reports/agent-outputs/architect/summary.md`, optional top-level metadata ограничена `summary`/`updated_at`, а extra outputs разрешены только под `reports/as-is/<domain>/overview.md`; normal и repair prompts начинают as-is surface с единственного `FIRST AS-IS DRAFT COMMAND`, который пишет `asis-draft-manifest.json` в `write_root` и `overview.md`/`summary.md`/`architect-summary.md` под `draft_final_root`; scaffold-only drafts после repair обрабатываются только one-shot `draft_artifact_enrichment` и всё равно принимаются исключительно через strict validation
   - `init|refresh.step4.proposals` использует strict shared draft contract: `step_contract="proposals"`, top-level shape `version=1/run_id/step_id/step_contract/agent_role/summary?/updated_at?/outputs[]`, allowed `outputs[].canonical_path` только `proposals/*` и `reports/changelog/*`, legacy envelopes (`pipeline`, `step`, `generated_at`, `domain_id`, `proposals[]`, `info_findings_noted`, `orphan_coverage_gaps`) reject-ятся strict parser-ом; normal prompt начинается с единственного `FIRST PROPOSALS DRAFT COMMAND`, который пишет `proposals-draft-manifest.json` в `write_root` и referenced draft files под `draft_final_root` до broad proposal analysis
   - runtime draft manifest contract для `step0/2/4` (`version=1`, `run_id`, `step_id`, `step_contract`, `agent_role`, optional `summary`/`updated_at`, `outputs[]`) вынесен в shared internal source of truth и используется и writer-ами, и validator-ами без дублирования структур
   - validators для collect manifests и runtime draft manifests read-only: hidden filesystem mutation, path normalization и draft file reconciliation внутри validation не допускаются
   - draft validation проверяет manifest contract, наличие referenced files под `draft_final_root` и отсутствие unchanged bootstrap/recovery scaffold в referenced markdown; files, записанные только по `outputs[].canonical_path`, считаются contract drift
   - active compatibility registry удалён; backward-compatible artifact rewrites не входят в runtime success path
   - draft-only шаги (`step0/2/4`) проходят тот же shared engine: draft manifest и referenced draft files валидируются до возврата в orchestrator; legacy draft schemas не принимаются silently
   - `claude-code`, `qwen-code` и `codex-code` используют shared provider-agnostic step-policy/prompt layer для required artifacts, retry bans и explicit negative rules; `qwen` получает artifact prompt только через CLI `-p` без JSON task stdin, custom qwen args не могут подменить artifact prompt, `qwen` использует `--output-format stream-json --include-partial-messages` для activity telemetry без semantic stdout contract, а `claude`/`codex` machine-mode flags считаются только transport/diagnostic transcript mode
   - runtime metadata для `fake`, `claude-code`, `qwen-code`, `codex-code` обязана иметь non-empty `meta.runtime.name` и `meta.runtime.version`; для headless providers version = `headless`, для fake name/version = `fake`
   - non-collect headless шаги больше не стартуют из workspace root: draft steps используют `draft_final_root` как cwd, validator использует `write_root`, а live harness разводит headless и baseline workspaces по разным temp roots вместо sibling layout
   - provider-side hard sandbox в текущих CLI surface нет; isolation достигается layout-ом temp roots, step-local `cwd` и opt-in managed permission policy, а не security/compliance sandbox enforcement
   - live provider approve-loop включается только если provider даёт structured permission events; без stable protocol `managed` fail-fast без PTY/expect text parsing
   - headless provider scope включает `arch-workspace` и resolved repo directories для текущих `repo_scope/repo_scopes`, чтобы provider видел source evidence из реальных checkout-ов
   - command overrides:
     - `ACP_CLAUDE_CMD` (default resolution `claude`, then legacy `claude-code`)
     - `ACP_QWEN_CMD` (default `qwen`)
     - `ACP_CODEX_CMD` (default `codex`)
   - live batch preflight records selected-provider readiness before deep matrix execution; command/probe/auth/quota failures, codex CLI compatibility blockers (for example `gpt-5.5` on an old Codex CLI) and selected-provider artifact smoke failure are operational blockers, not product verdicts; provider `model`/`modelUsage` telemetry is diagnostic text and does not block readiness
   - raw provider failure metadata includes redacted lifecycle diagnostics: resolved command path, argv, cwd, include dirs, pid, duration/exit reason, stdout/stderr byte counts, selected provider, prompt byte count, resolved runtime timeout profile and allowlisted `ACP_*_CMD`/timeout env presence/hash; prompt payload argv values are replaced with byte count + hash when present, and stdout/stderr diagnostics are redacted before persistence/streaming

6) **Workspace (`internal/workspace`)** *(implemented baseline)*
   - реализует/валидирует структуру central `arch-workspace` (Variant 2)
   - парсит `workspace.yaml`
   - валидирует manifest по `schemas/workspace.schema.json`
   - поддерживает `<docs.imports_path>/index.yaml` как metadata index для imported docs; отсутствие index silent, malformed/semantic issues warning-only
   - поддерживает repo entries с `path` или `git_url` + optional `ref`
   - поддерживает optional `repos[].analysis.include/exclude` для shard planner; legacy `repos[].analysis.role` удалён из active workspace contract
   - поддерживает optional persisted runtime profile в `runtime.profile` (`timeouts + execution + permissions`, см. `WORKSPACE_SPEC`)
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
   - в derived layer генерирует evidence-first C4 Mermaid set: `Context`, `Container`, per-service `Component`, per-service `Code`; `Context` сначала показывает external/team relations, а при их отсутствии использует bounded fallback из evidence-backed internal service/datastore relations, чтобы non-empty semantic model не превращался в gap-only context
   - materialize-ит индекс диаграмм `reports/diagrams/index.md` для UI filtering/open flow
   - формирует `reports/changelog/*` по итерациям

9) **Runtime Profile API (`internal/api`)** *(implemented baseline)*
   - `GET /api/runtime/timeouts`: persisted + effective + source
   - `PUT /api/runtime/timeouts`: partial update persisted timeout profile, write-through в `workspace.yaml`
   - `GET /api/runtime/execution`: persisted + effective + source
   - `PUT /api/runtime/execution`: partial update persisted execution profile, write-through в `workspace.yaml`
   - `GET /api/runtime/permissions`: persisted + effective + source
   - `PUT /api/runtime/permissions`: partial update persisted permission profile, write-through в `workspace.yaml`
   - `GET /api/runtime/profile`: aggregate view `timeouts + execution + permissions + step_providers`
   - runtime profile PUT handlers используют общий internal patch service для validate/merge/prune/render/write/reopen, чтобы API route code не дублировал workspace manifest mutation lifecycle
   - active run не прерывается при изменении timeout settings; новые значения применяются к следующим run
   - frontend live E2E differentiates productive timeout (`active_run_timeout`), backend run terminal failure observed by UI polling (`runtime_run_failed`), browser/page/context closure (`browser_closed`), post-failure API health loss (`api_unreachable`), early `acp serve` exit (`server_exited`) and fallback Playwright assertion failure (`playwright_failed`), while long backend polling uses an API request context independent from the browser page; init poll budget comes from effective runtime timeout profile and can follow `pipeline_timeout+30s` without a default fixed cap

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
- Target Q&A capability использует async runtime-backed run `pipeline="qa"` / step id `qa.ask`: orchestrator собирает deterministic `reports/taskruns/<run_id>/qa/context-pack.json` из `charter/cards`, `model`, `reports/as-is`, `reports/findings`, `reports/coverage`, `proposals`, `reports/changelog` и configured `docs.imports_path`, исключая `reports/taskruns/**`; runtime provider с role `system-analyst-qa` и `skills/prompt-packs/qa.md` пишет только `reports/taskruns/<run_id>/qa/qa-answer.json`.
- Current compatibility surfaces `acp qa` и `POST /api/qa/ask` остаются deterministic workspace-backed read-only service без runtime invocation до полной миграции внешних consumers.

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

## Boundary notes
- Native GitHub/GitLab webhook listener, hosted control plane and external SCM app integration остаются вне MVP; required integration surface — CLI batch job, optional trusted internal API trigger.
- Async Q&A API writes only run-scoped audit artifacts under `reports/taskruns/<run_id>/qa/` and must not mutate source repos or canonical architecture outputs. Legacy `POST /api/qa/ask` remains read-only and deterministic.

## Progress tracking
- Каноническая матрица stakeholder-статусов: `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
- `docs/PLANS.md` содержит инженерный ExecPlan и синхронизированный operational mirror статусов.
