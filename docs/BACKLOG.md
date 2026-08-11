# BACKLOG (baseline)

Этот backlog описывает эпики реализации, критерии приёмки MVP и рекомендуемую PR-level нарезку.
Для MVP-эпиков `Suggested PR slices` зафиксированы прямо в этом файле.
Required CI для MVP опирается на schema/contracts, synthetic fixtures, fake runner + recorded artifacts и не требует live headless provider binaries.

Статус выполнения и текущие активные engineering slices ведутся в `docs/STAKEHOLDER_DOC.md` (Canonical Stakeholder Matrix) и `docs/PLANS.md`; этот файл остаётся reference/acceptance backlog, а не единственным active tracker.

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
- orchestrator запускает headless runtime adapter для workspace (`claude-code` default, `qwen-code` optional, `codex-code` release peer)
- поддерживается передача PromptPack + subagents + skills
- adapter работает поверх baseline bundle agents/skills/prompts
- persisted runtime execution metadata сохраняется в `reports/taskruns/`
- required tests используют fake runner + artifact fixtures вместо live provider binaries

Suggested PR slices:
- `3A runner interface`
- `3B process execution + stdout/stderr`
- `3C taskrun persistence`
- `3D fake runner + artifact-fixture harness`

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
- target on-demand Q&A capability доступна как async runtime-backed UI/API flow: `POST /api/qa/runs`, `GET /api/qa/runs/<run_id>`, step id `qa.ask`, agent role `system-analyst-qa`
- deterministic workspace-backed read-only service остаётся compatibility/fake baseline для CLI `acp qa` + public read-only `POST /api/qa/ask`
- ответы содержат ссылки на evidence/артефакты workspace
- runtime-backed capability использует `skills/prompt-packs/qa.md`, пишет только `reports/taskruns/<run_id>/qa/{context-pack.json,qa-answer.json,runtime-execution.json}` и не мутирует source repos/canonical workspace outputs
- context pack excludes `reports/taskruns/**` by default

Suggested PR slices:
- `9A workspace-backed QA service`
- `9B public read-only Q&A API`
- `9C async runtime-backed QA runs`
- `9D compatibility policy through v1`
  - keep `POST /api/qa/ask` and `acp qa` response/behavior compatible through v1;
  - document migration to async `POST /api/qa/runs` + polling without runtime deprecation headers;
  - permit removal only through a separately approved v1 breaking-change plan.

## Epic 10 — Iteration Changelog (MVP)
Acceptance:
- на каждую итерацию формируется `reports/changelog/<yyyy-mm-dd>-<iteration-id>.md`
- changelog отражает изменения модели, findings, proposals и agent outputs
- changelog детерминирован при одинаковых входах

Suggested PR slices:
- `10A changelog compiler integration`

## Epic 11 — Q&A API Contracting Step (implemented beta)
Acceptance:
- read-only endpoint `POST /api/qa/ask` реализован поверх deterministic workspace-backed QA service
- response shape содержит `answer`, `citations`, `unresolved`, `confidence`
- endpoint не меняет workspace и не требует изменения runtime artifact contracts
- endpoint remains compatibility-only while UI target moves to async `/api/qa/runs`

Suggested PR slices:
- `11A /api/qa/ask contract + read-only semantics` (done)

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
- `acp serve --workspace <abs-path>` поднимает direct-mode single-workspace-per-process service для scripts/CI, while `acp serve` without workspace starts UI onboarding
- batch mode работает с тем же `workspace.yaml` и теми же pipeline step IDs
- hook-triggered workflow и manual pipeline button/job запускают тот же ACP flow
- SCM hooks обрабатываются CI provider; native ACP webhook listener / external SCM app integration остаются вне MVP
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
- baseline skills фиксированы: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`, `qa`
- baseline prompt packs фиксированы: `constitution`, `collect-context`, `findings`, `proposals`, `qa`
- bundle редактируется пользователем и версионируется в Git

Suggested PR slices:
- `15A subagents.yaml`
- `15B baseline skill packages`
- `15C baseline prompt packs`
- `15D UI editing/validation integration`

## Epic 16 — ACP Console V2 UX + Live E2E Coverage

Acceptance:
- исторический UI baseline зафиксирован в `docs/UI_CONSOLE_V2_DESIGN.md`; его PNG references
  удалены после supersession и не являются текущим target
- implementation baseline после rebase на `origin/main` `3aa458a`: текущий UI уже имеет
  stage shell, компактный activity drawer/artifact links, optional `UI_E2E_QA_SMOKE=1`,
  screenshot refs в frontend result JSON и удалённые hidden compatibility controls
- UI остаётся stage-based console с 8 стадиями `Source / Readiness / Charter / Analysis / Review / Proposals / Ask / Publish`
- общий shell включает top health strip, compact stage rail, center workbench, right inspector и bottom activity drawer
- `Analysis` показывает run timeline, shard table, blockers, evidence refs, runtime safety и live logs на одном экране
- `Review` получает две первичные рабочие поверхности: evidence workbench и domain map
- `Proposals` показывает proposal/changelog review, linked evidence, unresolved blockers и publication path
- `Ask` показывает async read-only Q&A run history, answer trust, citations, unresolved assumptions и runtime safety
- `Publish` показывает Git review room: folder diff summary, preview/diff/evidence tabs, publish gate, checklist, commit plan и proposal branch
- существующие backend API, runtime artifact schemas, CLI flags и workspace contracts не меняются в UI-only slice
- screen data source map зафиксирован в design baseline; недостающие данные показываются как explicit partial/empty state вместо неявного backend/API изменения
- стабильные operator-facing `data-testid` либо сохраняются, либо мигрируются вместе с UI/unit/live E2E tests; hidden compatibility controls не возвращаются
- live E2E логика обновлена под новый shell/stages и проверяет evidence/logs/runtime safety/Git publication surfaces внутри текущего `init-inspect` release-facing flow
- Ask/cancel/page-close/domain-map coverage остаётся deterministic или optional diagnostic, пока отдельный owner-approved slice не расширит public live shell за пределы `UI_E2E_SCENARIO=init-inspect`
- V2 live E2E сохраняет существующую reason taxonomy (`active_run_timeout`, `runtime_run_failed`, `browser_closed`, `api_unreachable`, `server_exited`, `playwright_failed`) и не вводит новые failure classes без отдельного slice
- frontend result JSON и Playwright diagnostics содержат scenario, reason, run id, last run status/error/current step, screenshots/traces/log refs и black-box evidence refs when available
- required CI остаётся deterministic/fake; live provider checks остаются manual trusted-machine release gate по runbook

Suggested PR slices:
- `16A UI V2 shell foundation`
  - refine current `AppShell`/`TopStatusBar`/`StageRail`/`RightInspector`/`ActivityDrawer` primitives instead of restoring removed legacy panels
  - shared stage status model and responsive density rules
  - baseline a11y/keyboard focus contract for rail, inspector and drawer
  - data adapters stay on existing hooks/API clients; no new backend contract in this slice
- `16B Source + Readiness consolidation`
  - repo table, Git URL/local path mode, docs imports and advanced `workspace.yaml`
  - readiness cards for workspace/repos/runtime/permissions/artifacts
  - runtime profile summary without making advanced settings the primary flow
- `16C Charter workbench`
  - charter wizard summary + artifact preview/editor
  - domain/team card overview and baseline prompt bundle status
  - charter readiness and Git path in inspector
- `16D Analysis mission control`
  - run timeline for canonical step IDs
  - shard table with failed shard drilldown
  - blocker/evidence/runtime safety inspector and log adjacency
- `16E Review evidence workbench`
  - artifact explorer grouped by workspace folders
  - markdown/diagram preview with claims/citations/coverage
  - findings, coverage gaps, trust status and approve/flag actions
- `16F Review domain map`
  - domain/service map using existing derived model and edge artifacts
  - ownership/coverage/cross-repo blocker inspector
  - navigation from map entity to evidence/proposal/Git diff
- `16G Proposals review room`
  - ADR/RFC/proposal list and preview/diff/evidence/changelog tabs
  - proposal quality, linked evidence and unresolved blocker handling
  - proposal branch path surfaced before Publish
- `16H Ask read-only Q&A console`
  - async Q&A run history and selected answer view
  - confidence/citations/unresolved/related entities and edges
  - explicit no-mutation runtime safety messaging
- `16I Publish gate`
  - folder-level diff summary and selected artifact preview
  - publish gate/checklist/blockers/commit plan/proposal branch
  - prepared commit message copy/export actions
- `16J UI V2 unit and accessibility coverage`
  - stage rendering, inspector priority, drawer controls, keyboard navigation
  - empty/loading/error/blocked states per critical stage
  - responsive desktop/tablet collapse behavior
- `16K Live E2E selector migration`
  - update Playwright navigation to stage rail and V2 primary actions
  - preserve or migrate only visible operator-facing `data-testid`; do not add hidden compatibility controls
  - keep regression coverage that hidden compatibility controls are absent from the UI shell
  - update `ui/e2e/live-flow.spec.ts`, `ui/playwright.live.config.ts` only if artifact behavior changes, `scripts/frontend-live-e2e.sh` only if the public live shell changes, `scripts/tests/frontend_live_e2e_contract_test.py` and selector docs together
- `16L Live E2E operator journey`
  - fake-runtime required flow: Source -> Readiness -> Analysis -> Review -> Publish
  - assert run status, blocker/evidence/logs/runtime safety/Git path surfaces
  - capture diagnostic screenshots for each major stage on failure
  - preserve trace/screenshot/video failure artifacts under Playwright `UI_E2E_OUTPUT_DIR`
  - keep release-facing frontend live shell on `UI_E2E_SCENARIO=init-inspect`; cancellation/page-close coverage remains deterministic fake-runtime UI/API tests outside the live release gate
- `16M Live E2E Ask and domain-map diagnostics`
  - async Ask submit/poll/result/citation checks through optional `UI_E2E_QA_SMOKE=1` on `init-inspect`, not a separate required shell scenario
  - domain-map render and selected domain evidence link checks through unit/component/fake-fixture coverage first
  - keep provider-live variants optional diagnostics and classified by existing reason taxonomy
  - do not add `ask-readonly` or `domain-map-diagnostic` to `scripts/frontend-live-e2e.sh` allowlist without a separate release-gate owner decision
- `16N Docs/runbook sync`
  - update README/ARCHITECTURE UI description only after implementation
  - update `docs/TESTING_STRATEGY.md` and `docs/RELEASE_LIVE_E2E_RUNBOOK.md` with V2 live E2E flow
  - archive or update the active ExecPlan after owner review/merge

## Epic 17 — Onboarding-first Workspace, Sources and Runner Setup

Status: implemented as beta baseline in the current onboarding-first branch; provider-live release gate remains unchanged.

Acceptance:
- `acp serve` can start a local onboarding UI without requiring `--workspace` or repo flags up front.
- Existing direct mode remains supported: `acp serve --workspace <path>` opens the normal single-workspace console path for scripts, CI, existing users and live E2E.
- Onboarding is a pre-console setup surface, not a ninth product stage.
- The first onboarding decision is architecture workspace selection: create a new workspace path or reopen an existing workspace.
- Workspace create/open validates writable path, fixed layout readiness and git initialization, with actionable inline errors.
- A new workspace can exist in draft setup state before `workspace.yaml` is valid, but pipeline actions remain disabled until a valid manifest is saved.
- Target repositories are configured in UI through existing `workspace.yaml.repos[]` semantics.
- One or more target repos are supported; each repo has unique `name`, exactly one source (`path` or `git_url`) and optional `ref`.
- Repo validation supports local checkout paths and GitHub/GitLab-style URLs through local `git` auth context; ACP does not add a credential store.
- Runner selection is mandatory before the first analysis.
- `fake` is shown as the recommended first walkthrough runner and needs no external provider command.
- Live runners remain explicit opt-in: `claude-code`, `qwen-code`, `codex-code`; provider command/auth readiness is surfaced before run.
- No source repository mutation happens during onboarding validation or fake analysis.
- Existing workspace schema, runtime artifact contracts and CLI batch `acp run` behavior are unchanged unless a later slice performs full schema/docs/fixtures sync.
- Deterministic onboarding UI/e2e coverage exists, while release-facing provider-live frontend shell remains `UI_E2E_SCENARIO=init-inspect` unless a separate owner-approved live-gate slice changes it.

Suggested PR slices:
- `17A Launcher + workspace selection foundation`
  - allow `acp serve` without `--workspace` to start loopback onboarding UI
  - preserve `acp serve --workspace <path>` direct mode
  - add onboarding status and workspace create/open APIs
  - support workspace layout/git init before valid `workspace.yaml`
  - add first UI screen for workspace path validation and create/open
- `17B Onboarding source repositories`
  - move first-run repo selection into onboarding while reusing Source-stage repo table semantics
  - support add/remove multiple repos, Git URL/local path mode, optional `ref`, duplicate-name checks and docs imports path
  - save valid repo setup through existing `workspace.yaml`
  - show row-level validation and recovery states before enabling readiness/run
- `17C Mandatory runner selection`
  - add onboarding runner step with `fake`, `claude-code`, `qwen-code`, `codex-code`
  - show provider command override hints and availability diagnostics
  - keep `fake` deterministic and always available
  - wire selected runner into first-run service/runtime state without changing artifact contracts
- `17D Onboarding to Console V2 transition`
  - add final readiness summary and transition into existing `Source -> Readiness -> Charter -> Analysis -> Review -> Proposals -> Ask -> Publish` shell
  - keep Source editable after onboarding for repo changes
  - ensure first analysis is disabled until workspace, repos and runner are valid
  - preserve activity drawer/inspector/top-strip state after transition
- `17E Onboarding tests, docs and live E2E hardening`
  - deterministic Playwright/UI flow from blank `acp serve` to fake first analysis
  - direct-mode `init-inspect` regression for existing live E2E
  - README/INSTALL/TROUBLESHOOTING updates distinguishing UI-first setup, direct mode and multi-repo setup
  - no new required provider-live release scenario without owner-approved release-gate decision

## Epic 18 — Live E2E Black-box Artifact Boundary

Status (2026-07-26): trusted-machine R3 release evidence remains open. The two provider-free
remediations that were still pending in the previous status are merged: exact step2 evidence
references landed in PR #171 (`ca8c3f67`) and mixed recovery routing landed in PR #172
(`a633e3ce`). A subsequent static, deterministic and historical-artifact audit identified
additional correctness and live/product trust-boundary blockers recorded in Epic 22. Their
provider-free implementation and working-tree closure gate now pass; R3 remains paused until the
same gate is tied to one clean reviewed qualification commit. Stopped or earlier live matrices
remain diagnostic only and cannot be reused as release evidence.

Context:
- latest strict medium diagnostics validated the execution/artifact-quality split shape
  (`execution_report_*`, selected-provider totals, no legacy mixed `quality_report_*` artifacts),
  but did not fully validate black-box artifact quality;
- `regres-long-posthog-ftgo-20260630T172821Z` left `ftgo` incomplete because `claude-code`
  hit provider quota/permission limits during refresh, so artifact and UX decisions stayed
  inconclusive for that target;
- promoted operator-facing artifacts from the same evidence showed boundary leaks:
  `reports/as-is/overview.md` included `run_*` taskrun ids, `reports/taskruns/.../staging/shards`,
  `runtime-execution.json` snippets and stale final/citation-index availability claims;
- promoted coverage markdown also included runtime policy wording such as
  `Downstream quality gates still decide whether recovered artifacts are complete enough for acceptance`;
- artifact quality assessment must evaluate ProvenArch as a black box: the decision is based on
  promoted workspace artifacts and UI-visible results, while `reports/taskruns/*-quality.json`,
  raw logs, runtime metadata and matrix reports are execution telemetry only.

Acceptance:
- machine execution evidence remains separate from manual quality decisions:
  `release_verdict_<matrix-id>.json` / `execution_report_<batch-id>.md` decide only execution,
  while `swe_ux_assessment_<matrix-id>.md` and
  `swe_artifact_quality_assessment_<matrix-id>.md` decide release UX/artifact acceptance;
- black-box artifact assessment has an explicit allowed surface:
  `reports/as-is/**`, `reports/coverage/**`, `reports/findings/**`,
  `reports/diagrams/**`, `reports/agent-outputs/**`, `proposals/**`, `model/**`,
  `charter/**` only where promoted as user-facing output, plus UI preview/screenshots;
- artifact assessment does not use `reports/taskruns/*-quality.json`, raw provider logs,
  runtime metadata, matrix inventories or execution counters as source-of-truth for artifact
  acceptance; those files may be cited only in the execution-quality section;
- promoted operator-facing markdown does not leak live/runtime internals:
  `reports/taskruns`, `/staging/`, `runtime-execution.json`, raw stdout/stderr/log refs,
  `task-run_`, foreign/current `run_*` taskrun IDs, `write_root`, `artifact_root`,
  recovery-process narration, scaffold/bootstrap markers or live-gate policy terms are rejected
  unless the file is explicitly internal telemetry under `reports/taskruns/**`;
- provider-authored docs do not publish stale downstream-index claims: `step2.asis_docs`
  must omit final/citation-index availability if those indexes are not yet in-scope, and
  `step4.proposals` must summarize current-run `final-run-index.json.canonical_documents[]`
  and `citation-index.json.citations[]` only when they are actually present and validated;
- `collect_manifest_runtime_recovery` and fake-runtime generated user-facing artifacts do not
  inject release/live E2E policy phrases or `reports/taskruns/**` staging paths into promoted
  reports;
- runtime/prompt validation treats operator-facing contamination as `runtime_contract_failed`
  when it is machine-checkable, not as a subjective manual artifact-quality warning;
- boundary tests cover both directions:
  live release evidence names stay out of core AOR runtime/orchestrator logic, and promoted
  artifact fixtures/golden outputs stay free of live/runtime internals;
- docs/templates make the black-box rule explicit: SWE artifact-quality reports must start from
  promoted artifacts and UI-visible results, not from telemetry counters;
- strict medium rerun evidence after implementation includes at least one clean `claude-code`
  and one clean `codex-code` diagnostic when provider readiness allows; if a provider is blocked
  by quota/auth/capacity, the report classifies it as an external `runner_unavailable` blocker
  and does not count it as artifact-quality acceptance;
- no canonical matrix files, curated repo files, product UI/API behavior or runtime artifact
  schemas are changed merely to make the live E2E evidence pass.

Suggested PR slices:
- `18A Black-box artifact assessment contract`
  - update `docs/RELEASE_LIVE_E2E_RUNBOOK.md`, `docs/templates/LIVE_E2E_OPERATOR_ASSESSMENT.md`
    and `docs/TESTING_STRATEGY.md` with allowed black-box artifact surfaces
  - state that `reports/taskruns/*-quality.json` is telemetry-only for artifact decisions
  - add checklist items for truthfulness, completeness, readability, C4/Mermaid usefulness,
    citations/indexes, proposals and operator decision readiness
- `18B Promoted artifact contamination validation`
  - add shared detector for operator-facing markdown contamination markers
  - reject `reports/taskruns`, `/staging/`, `runtime-execution.json`, task ids, raw log refs,
    `write_root`, `artifact_root`, recovery/bootstrap narration and live-gate policy wording
    outside internal telemetry surfaces
  - add focused fixtures for PostHog/FTGO-style stale overview and coverage-summary leaks
- `18C Runtime recovery wording cleanup`
  - remove `Downstream quality gates...` and similar policy text from runtime-recovered
    provider-authored/promoted content
  - keep recovery diagnostics in telemetry/logs instead of user-facing architecture reports
  - ensure fake runtime user-facing reports reference promoted paths, not
    `reports/taskruns/<run_id>/staging/final/...`
- `18D Downstream index truthfulness hardening`
  - keep `step2.asis_docs` from claiming final/citation indexes are missing before downstream
    indexes exist
  - verify `step4.proposals` counts documents/citations only from current-run top-level
    `canonical_documents[]` and `citations[]`
  - retain provider-authored retry paths without adding deterministic hidden synthesis
- `18E Boundary regression tests`
  - extend `scripts/tests/aor_live_boundary_test.py` or add a companion test for promoted
    artifact contamination markers
  - add Go/runtime tests for rejected contaminated markdown and accepted clean operator-facing
    markdown with concrete repo/path evidence
  - add script/report tests proving artifact telemetry remains non-blocking for machine
    execution verdicts
- `18F Trusted medium rerun and black-box report`
  - rerun `regres long` from a clean worktree with direct `scripts/full-run-batch-matrix.sh`
  - run `claude-code` and `codex-code` selected-provider diagnostics when host/provider
    readiness permits
  - produce a chat/report summary split into execution quality, black-box artifact quality and
    UX quality, with leakage scan results over promoted artifacts
  - do not commit generated live E2E evidence unless it is converted into an intentional
    fixture/golden file
- `18G Step2 first-pass Excellent blocker`
  - harden `step2.asis_docs` normal prompt/validation so first-pass `overview.md`,
    `summary.md`, `architect-summary.md` and `asis-draft-manifest.json` are validation-ready
    without `draft_artifact_enrichment`
  - reject and prevent stale downstream final/citation-index availability wording in step2
    operator-facing markdown; when those downstream indexes are absent during step2, omit their
    status instead of publishing `unavailable` / `not yet present` claims
  - preserve exact typed shard completeness (`planned=<n> succeeded=<n> failed=<n>
    incomplete=<n>`), concrete repo/path or staged artifact refs and decision-ready operator
    summary in the first provider filesystem work unit
  - add prompt/validation regressions from the `smoke-tiny-bank-20260707T053308Z` evidence:
    first-pass downstream-index wording must fail, clean first-pass step2 output must pass
    without focused repair
- `18H Step4 first-pass actionability blocker`
  - harden `step4.proposals` normal prompt/validation so first-pass proposal/changelog output
    links medium/high current-run findings to exact finding IDs, copied severities, affected
    surfaces/paths and concrete recommended operator actions
  - keep the bullet-only actionable finding format and reject generic `inspect` / `review` /
    `decide`-only plans when medium/high findings exist
  - ensure first-pass proposal/changelog text does not rely on focused repair to become
    operator-actionable, while keeping ACP from deterministically synthesizing proposal content
  - add live-shaped regression from `smoke-tiny-bank-20260707T053308Z`: `low_actionability`
    first-pass proposal output must fail before promotion, and exact linked actionable bullets
    must pass without `runtime_quality.repair_heavy`

## Epic 19 — Code Quality Audit Remediation (Local-first MVP)

Status: implementation-complete and merged into `main` at `02716bb`.
Source-of-truth findings — `docs/CODE_AUDIT_2026-07-10.md` at baseline
`122e4c9b5a91b29e243677c0dac0fe2ebfca226b`.

### Goal

- закрыть все 19 Major/P1 findings до следующего public release;
- разложить Normal/P2 и confirmed dead code на независимые reviewable slices;
- усилить crash consistency, run lifecycle, contract correctness, UI state consistency,
  reproducibility и deterministic CI без расширения product scope;
- сохранить local-first MVP, синхронно эволюционировать затронутые public contracts и
  оставить required CI без live network dependencies.

### Explicit non-goals and local frontend boundary

- frontend остаётся локальной loopback/trusted-operator surface, встроенной в Go binary;
- этот epic **не добавляет** frontend authentication/authorization, multi-user isolation,
  hosted exposure, CSP/CORS hardening program, browser sandboxing, security/compliance policy
  enforcement или отдельный secrets boundary;
- security baseline остаётся Wave 1+ и должен планироваться отдельным epic/threat-model только
  если deployment меняется с local/trusted на shared, remote или hosted;
- UI slices ниже исправляют correctness, stale state, data loss, accessibility и deterministic
  QA; они не должны маскироваться под frontend security work;
- live provider matrix не становится required per-PR CI: она остаётся manual trusted-machine
  pre-release gate по `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.

### Priority and sequencing

| Phase | Exit criterion | Slices |
|---|---|---|
| P1-A Crash/lifecycle | canonical state crash-safe; async runs имеют одного owner | 19A–19E |
| P1-B Contract/source correctness | refresh и collect используют fresh, symmetric evidence | 19F–19H |
| P1-C UI correctness | historical/active state не смешивается и edits не теряются | 19I–19L |
| P1-D Release reproducibility | embed, verdict и contract tools воспроизводимы | 19M–19O |
| P2 Quality hardening | заявленное semantic/UI/CI behavior покрыто deterministic tests | 19P–19V |
| P3 Cleanup | confirmed dead code удалён после зависимых behavior decisions | 19W–19X |

P1 slices допускают параллельную работу только внутри независимых групп. Обязательные
dependencies: `19A -> 19B`, `19C -> 19D -> 19E`, `19G -> 19H`,
`19J -> {19K1, 19K2, 19L}`, `19M + 19O -> 19N`. Slices `19W*/19X` выполняются последними,
чтобы не удалять код, который ещё нужен для восстановления заявленного behavior.

### Epic acceptance

- все `BUG-001..BUG-019` закрыты regression tests или явно superseded одним принятым
  behavior/ADR решением;
- `REF-001..REF-003` и `QUAL-001..QUAL-007` имеют implemented deterministic gates;
- `DEAD-001..DEAD-013` удалены либо оставлены только с documented active call site;
- required CI выполняет canonical `make contracts`, `make test`, `make lint` и build/drift
  checks без live providers;
- каждый completed slice проходит `make contracts`, `make test`, `make lint`, `make build`;
- schema/contract slices синхронизируют `schemas/*`, `docs/spec/*`,
  `docs/APPENDIX_SCHEMAS.md`, examples, fixtures, validators/tests и ADR rationale;
- behavior slices обновляют README/ARCHITECTURE/STAKEHOLDER/TESTING_STRATEGY только там,
  где фактическое operator-facing или testing behavior меняется;
- завершение P1 подтверждается deterministic CI; manual live release gate выполняется
  отдельно и не блокирует review отдельных non-live slices.

Implementation note: local slice commits `19A..19X` completed the program and were merged into
`main` at `02716bb` after the reconciled branch passed `make contracts`, `make test`, `make lint`
and `make build`. Epic 20 is unblocked; its first implementation slice starts with a current-code
sufficiency audit because Epic 19 already delivered part of the snapshot foundation.

### Suggested PR slices

#### `19A Atomic recovery-state persistence`

- Priority/effort/findings: P1 / M / `BUG-005`.
- Deliverable: отдельный atomic write primitive для run history, shard summaries и runtime
  checkpoints: temp file, fsync, rename, error propagation и last-good/recovery diagnostic.
- Modules: `internal/workspace`, `internal/orchestrator/service_runs*`,
  `restart_reconcile*`, `sharding_artifacts*`.
- Tests: fault injection до/после write/rename, malformed current + valid last-good,
  restart replay; ошибки persistence не должны silently становиться empty history.
- Docs: ARCHITECTURE/TESTING_STRATEGY — только crash/recovery semantics.
- Dependency: первая backend slice; prerequisite для 19B.

#### `19B Transactional canonical promotion`

- Priority/effort/findings: P1 / L / `BUG-001`; depends on 19A.
- Deliverable: build/validate managed generation в sibling staging tree, затем atomic activation
  или journaled rollback; live canonical tree не мутируется до готовности generation.
- Modules: `internal/orchestrator/docflow_promotion*`, model/diagram materialization,
  workspace generation helpers.
- Tests: fail each N-th copy/remove/model/diagram operation; after failure workspace соответствует
  ровно previous либо new complete generation, mixed artifacts запрещены.
- Docs: ARCHITECTURE + recovery operator guidance; public artifact schemas не меняются.

#### `19C Async panic isolation`

- Priority/effort/findings: P1 / S / `BUG-003`.
- Deliverable: recover только на outer async goroutine boundary; terminal internal failure и
  `finishAsyncRun` гарантируются через defer, synchronous panic semantics не меняются.
- Modules: `internal/orchestrator/service_runs*` и async lifecycle tests.
- Tests: panic runner для init/refresh/QA; API process жив, slot освобождён, pending run продолжен.
- Docs: testing strategy only; API contract shape unchanged.

#### `19D Server-owned shutdown`

- Priority/effort/findings: P1 / M / `BUG-002`; depends on 19C.
- Deliverable: signal-aware server context и bounded `Service.Close/Shutdown`, отменяющий
  active/pending runs, provider process groups и post-shutdown writes.
- Modules: `cmd/acp`, `internal/api`, `internal/orchestrator`, provider process-group adapters.
- Tests: context cancellation и process-level SIGTERM with blocking fake provider; terminal
  history persisted, no orphan process, bounded exit.
- Docs: CLI/ARCHITECTURE shutdown semantics; no frontend security changes.

#### `19E Coherent service/session generation`

- Priority/effort/findings: P1 / M / `BUG-006`, `BUG-007`; depends on 19D.
- Deliverable: immutable workspace/service/config snapshot per request; coordinated service
  generation swap; runtime reselection returns conflict while active/pending or performs
  explicit quiesce before replacement.
- Modules: `internal/api/server*`, onboarding runtime selection, orchestrator registry.
- Tests: concurrent polling + workspace/runner mutation under race detector; direct-mode
  reported/effective runner equality; no stale goroutine ownership.
- Docs: API/ARCHITECTURE conflict behavior if a new 409 surface is introduced.

#### `19F Fresh unpinned git_url resolution`

- Priority/effort/findings: P1 / M / `BUG-004`.
- Deliverable: after fetch resolve remote default HEAD, reset ACP-owned cache to exact SHA and
  persist resolved SHA in run evidence; pinned ref behavior remains unchanged.
- Modules: `internal/workspace/resolver*` and resolver fixtures.
- Tests: local bare remote receives new default-branch commit; second resolve without ref reads
  new content; pinned ref remains stable; no live network.
- Docs: WORKSPACE_SPEC/README source freshness semantics without schema change.

#### `19G Minimum collect evidence contract`

- Priority/effort/findings: P1 / M / `BUG-008`.
- Deliverable: non-empty documents/citations and authored document citation IDs become explicit
  collect contract requirements before checkpoint/apply.
- Modules: shard-pack schema, contracts/docflow validators, runtime apply gate.
- Tests/fixtures: empty arrays, missing citation IDs and sparse semantic packets fail and enter
  existing repair/terminal path; valid minimal fixture passes.
- Docs: full schema-guardian sync, Appendix Schemas, examples, fixtures and ADR rationale.
- Dependency: run before 19H to avoid two competing schema migrations.

#### `19H Symmetric document/citation validation`

- Priority/effort/findings: P1 / M / `BUG-009`; depends on 19G.
- Deliverable: every `citation.document_ids` resolves to a current document and reverse links
  remain symmetric before and after document-ID remap.
- Modules: contracts/docflow validation and staged index/remap validation.
- Tests/fixtures: unknown ID, typo, asymmetric binding and post-remap dangling ID fail;
  valid duplicate-ID remap remains green.
- Docs: PIPELINE_SPEC + schema appendix/fixtures through schema-guardian workflow.

#### `19I Historical run artifact snapshots`

- Priority/effort/findings: P1 / M / `BUG-011`.
- Deliverable: UI model keeps canonical label and run-scoped staged read path separately;
  historical preview/coverage/questions always read selected-run snapshot.
- Modules: `ui/src/lib/appContracts.ts`, `useRunArtifacts` and artifact API adapter.
- Tests: two runs share canonical paths but contain different staged content; old selection never
  reads current canonical file.
- Docs: UI behavior docs only; no auth, browser perimeter or backend security scope.

#### `19J Request-scoped UI detail state`

- Priority/effort/findings: P1 / M / `BUG-012`, `BUG-014`.
- Deliverable: reusable request generation/AbortController primitive and keyed
  `{requestKey,status,data}` state for run status, logs, artifacts, previews and Git diff.
- Modules: run/log/artifact/diff hooks and shared UI async-state helper.
- Tests: deferred A then B responses; late A cannot update any B panel; unmount abort is clean.
- Docs: none unless operator-visible loading/error behavior changes.

#### `19K1 Run mutation acknowledgement`

- Priority/effort/findings: P1 / S / `BUG-013`; depends on 19J.
- Deliverable: accepted start/cancel response materializes provisional state immediately;
  follow-up GET failure is recoverable reconciliation, not mutation failure.
- Modules: `useRunActions` and run polling/recovery state.
- Tests: POST success + first GET failure does not duplicate start/cancel; polling recovers the
  same run ID and preserves accepted action copy.
- Docs: UI recovery copy only if labels change; no frontend security work.

#### `19K2 Q&A provisional run and selection ordering`

- Priority/effort/findings: P1 / M / `BUG-015`; depends on 19J.
- Deliverable: accepted Q&A response creates provisional run before detail GET; history selection
  uses request generation and last selection wins.
- Modules: Ask/Q&A StagePanels state and QA API adapter.
- Tests: first detail GET may fail and recover the same ID; delayed A history response cannot
  replace selected B; duplicate submit is impossible while accepted run reconciles.
- Docs: Ask recovery behavior only; no frontend security work.

#### `19L Editor revision safety`

- Priority/effort/findings: P1 / M / `BUG-016`, `BUG-017`; depends on 19J.
- Deliverable: form revision/snapshot checks for manifest save and single-owner, per-path dirty
  draft loading for charter/baseline editor.
- Modules: `useManifestEditor`, `useBaselineEditor` and App selection effect.
- Tests: edit during deferred save remains dirty; duplicate/late load cannot overwrite typed text.
- Docs: none unless save/recovery UX changes.

#### `19M Deterministic embedded UI bundle`

- Priority/effort/findings: P1 / M / `BUG-018`.
- Deliverable: stabilize Vite chunk/file generation, add clean exact-commit build comparison and
  fail PR when `internal/api/ui_dist` is stale.
- Modules: Vite config, Makefile generated-copy target, UI workflow and embedded bundle manifest.
- Tests: two independent temp-root builds have identical paths/digests; stale embed fixture/check
  fails while unchanged build produces empty diff.
- Docs: TESTING_STRATEGY and release build instructions.

#### `19N Release composite-verdict gate`

- Priority/effort/findings: P1 / M / `BUG-019`; depends on 19M and 19O.
- Deliverable: release job receives canonical verdict + accepted UX/artifact assessments and runs
  offline verifier before GoReleaser/write permissions.
- Modules: release workflow and existing `verify-release-verdict.py` contract tests.
- Tests: missing, FAIL, matrix-mismatched or unaccepted SWE inputs prevent GoReleaser; complete
  accepted evidence allows dry-run publication path.
- Docs: RELEASE_LIVE_E2E_RUNBOOK/release process; full live matrix remains manual trusted-machine.

#### `19O Locked contract validator toolchain`

- Priority/effort/findings: P1 / M / `REF-002`.
- Deliverable: versioned lockfile/package for ajv-cli, ajv-formats and js-yaml; contract target
  runs installed exact versions and does not resolve mutable registry latest.
- Modules: contract tooling package/lockfile, Makefile and contracts workflow.
- Tests: offline contract validation after clean install; version update requires explicit
  lockfile diff; current positive/negative contract fixtures unchanged.
- Docs: CONTRIBUTING/TESTING_STRATEGY toolchain bootstrap.

#### `19P Restore Step 1 card enrichment`

- Priority/effort/findings: P2 / M / `BUG-010`, `DEAD-001`.
- Decision: retain the existing PIPELINE_SPEC behavior and restore one contract-safe enrichment
  call after semantic apply; do not delete the cluster.
- Modules: Step 1 handler and semantic card rendering/enrichment.
- Tests: init/refresh with domain/team cards creates one idempotent Derived section with evidence
  refs and never auto-creates/renames human-owned cards.
- Docs: no contract change; update architecture only if exact execution point is documented.

#### `19Q Generic refresh semantic guard`

- Priority/effort/findings: P2 / M / `REF-001`, `DEAD-002`.
- Decision: preserve the documented refresh guard, but replace/rework narrow unreachable helpers
  into a generic policy that marks or rejects runtime/provider metadata and off-topic candidates
  without domain-specific heuristics.
- Modules: semantic normalization/diagnostics and refresh tests.
- Tests: runtime metadata/off-topic candidate is marked/filtered, legitimate same-domain entity
  survives, init behavior is unchanged, diagnostic is deterministic.
- Docs: ARCHITECTURE + ADR explaining generic policy and no hidden domain whitelist.

#### `19R1 ARIA tabs controller`

- Priority/effort/findings: P2 / S / `QUAL-001`.
- Deliverable: reusable roving-tabindex controller with Arrow/Home/End navigation and stable
  tab-to-tabpanel relationships.
- Modules: `TabNav` and tabbed stage panels.
- Tests: only active tab is tabbable; keyboard focus/selection and labelled panel behavior.
- Docs: UI accessibility contract; usability scope only.

#### `19R2 Keyboard path combobox`

- Priority/effort/findings: P2 / S / `QUAL-002`.
- Deliverable: active option state, aria-activedescendant, Arrow/Enter/Escape behavior and
  pointer/keyboard parity.
- Modules: `LocalPathCombobox` and onboarding/source consumers.
- Tests: full keyboard-only selection, Escape close and active descendant assertions.
- Docs: UI accessibility contract; no security boundary change.

#### `19R3 Accessible async announcements`

- Priority/effort/findings: P2 / S / `QUAL-003`.
- Deliverable: shared alert/live-status primitive plus aria-invalid/describedby wiring for
  asynchronous validation, save and run errors.
- Modules: App/onboarding status and error surfaces.
- Tests: error alert, polite progress/success and field-linked diagnostic behavior.
- Docs: UI accessibility contract; desktop/mobile mock QA remains green.

#### `19S1 Confirmed shell dead-code cleanup`

- Priority/effort/findings: P2 / S / `DEAD-011`, `DEAD-012`, `DEAD-013`.
- Deliverable: remove unused matrix/batch helpers and frontend status assignments without
  changing active result classification.
- Modules: batch and matrix shell scripts.
- Tests: Python/shell contract tests and bash syntax remain green; targeted reference search empty.
- Dependency: perform before enabling ShellCheck in 19S2.

#### `19S2 ShellCheck in canonical lint`

- Priority/effort/findings: P2 / S / `QUAL-005`; depends on 19S1.
- Deliverable: Makefile lint executes ShellCheck for production scripts with only narrow,
  documented suppressions for intentional indirect trap callbacks/export idioms.
- Modules: Makefile, shell lint configuration and script test fixtures.
- Tests: current scripts clean; a test probe with unused variable or invalid shell pattern fails.
- Docs: TESTING_STRATEGY canonical lint baseline.

#### `19S3 Required PR lint job`

- Priority/effort/findings: P2 / S / `QUAL-004`; depends on 19S2.
- Deliverable: required PR workflow invokes canonical `make lint` rather than duplicating a
  partial gofmt/typecheck subset.
- Modules: backend/UI workflow composition and Makefile entrypoint.
- Tests: intentionally unformatted Go or ShellCheck violation makes required check red;
  valid branch remains provider-free.
- Docs: TESTING_STRATEGY required CI matrix.

#### `19T Logs endpoint smoke coverage`

- Priority/effort/findings: P2 / S / `QUAL-006`.
- Deliverable: smoke-api requests run logs with cursor/limit and validates response shape and
  pagination cursor.
- Modules: `scripts/smoke-api.sh` and its deterministic stubs/tests.
- Tests: 5xx, malformed payload and invalid cursor fail; normal empty/non-empty pages pass.
- Docs: TESTING_STRATEGY smoke baseline synchronized.

#### `19U Deterministic mock Playwright CI`

- Priority/effort/findings: P2 / M / `QUAL-007`.
- Deliverable: separate `e2e:mock` runner with local Vite server and explicit seven-scenario
  matrix; no live providers, external repos or network dependencies.
- Modules: Playwright mock config/runner, UI package scripts, Makefile and UI workflow.
- Tests: 7 passed / 0 skipped; broken selector fails required check; desktop/mobile overflow and
  console-error assertions remain active.
- Docs: TESTING_STRATEGY; release-facing live scenario allowlist unchanged.

#### `19U2 Optional V8 coverage baseline`

- Priority/effort/source: P2 optional / S / residual audit test gap, not a reportable finding.
- Deliverable: lock `@vitest/coverage-v8` and publish a deterministic text/JSON coverage summary
  that includes all `ui/src` files.
- Modules: UI package/lockfile and deterministic unit-test workflow; no Playwright coupling.
- Tests: clean install produces coverage without prompt/download; initial thresholds are recorded
  baseline and may only ratchet upward.
- Docs: TESTING_STRATEGY; this optional coverage slice is independent from the required
  browser-CI finding.

#### `19V Pinned Python tooling runtime`

- Priority/effort/findings: P2 / M / `REF-003`.
- Deliverable: exact supported Python version, setup-python in workflows and one
  version-checking wrapper used by Makefile/scripts.
- Modules: CI workflows, developer version file/wrapper and Python test entrypoints.
- Tests: wrong interpreter fails before suite; all jobs report same version; 230 tests pass.
- Docs: CONTRIBUTING/TESTING_STRATEGY bootstrap.

#### `19W1 Runtime-draft wrapper cleanup`

- Priority/effort/findings: P3 / S / `DEAD-003`.
- Deliverable: remove six orchestrator runtime-draft wrappers and use canonical
  `internal/runtimedrafts` call sites only.
- Tests: runtime-draft/orchestrator package tests + Staticcheck 2026.1.
- Dependency: after all P1 backend slices.

#### `19W2 Sharding wrapper cleanup`

- Priority/effort/findings: P3 / S / `DEAD-004`.
- Deliverable: remove four legacy sharding planner/artifact wrappers without adding aliases.
- Tests: deterministic sharding tests + Staticcheck 2026.1.
- Dependency: after all P1 backend slices.

#### `19W3 Provider argument wrapper cleanup`

- Priority/effort/findings: P3 / S / `DEAD-005`.
- Deliverable: remove three legacy default-argument entry points and retain the
  permission-aware builders.
- Tests: Claude/Qwen/Codex adapter argument tests + Staticcheck 2026.1.
- Dependency: after lifecycle/runtime P1 slices.

#### `19W4 Docflow compatibility helper cleanup`

- Priority/effort/findings: P3 / S / `DEAD-006`.
- Deliverable: remove two local helpers superseded by the artifact-quality layer.
- Tests: docflow/artifact-quality package tests + Staticcheck 2026.1.
- Dependency: after 19G/19H contract work.

#### `19W5 Package-local residual cleanup track`

- Priority/effort/findings: P3 / S each / `DEAD-007`.
- Rule: one PR per package; no cross-package forwarding wrapper.

| Sub-slice | Package/location | Required test |
|---|---|---|
| `19W5a` | `internal/api/review_diff.go` | API/review diff package tests |
| `19W5b` | `internal/model/store.go` | model store/golden tests |
| `19W5c` | `internal/orchestrator/quality.go` | orchestrator quality tests |
| `19W5d` | `internal/reports/compiler.go` | reports compiler tests |
| `19W5e` | `internal/runtime/promptcontract/collect_repair.go` | prompt-contract tests |

Каждый sub-slice дополнительно запускает Staticcheck 2026.1 и full DoD. Track выполняется
после 19P/19Q decisions и всех P1 backend slices.

#### `19X UI dead-surface cleanup`

- Priority/effort/findings: P3 / S / `DEAD-008`, `DEAD-009`, `DEAD-010`.
- Deliverable: remove unused Diagnostic import/props, legacy QA client and facade members, unless
  a preceding correctness slice has created an explicit active consumer.
- Modules: App/StagePanels, QA client and run-review/diff/explorer hooks.
- Tests: noUnusedLocals/noUnusedParameters, Vitest and mock Playwright all green.
- Dependency: after 19I–19L so request-state refactor owns the final hook surface.

## Epic 20 — Console UX trust, evidence workflow and IA reset (post-beta)

Context:
- Epic 16 и Epic 17 дали рабочий beta baseline и сильное recovery-покрытие, но последующий
  task-based аудит показал системный разрыв: console хорошо объясняет runtime diagnostics,
  однако недостаточно надёжно отвечает на вопросы «какой run я смотрю», «какие данные
  проверены», «что именно будет закоммичено» и «что считается завершённым».
- Это corrective successor для UI/IA частей Epics 16–17. Его historical target описан в
  [`UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md`](UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md) и
  superseded task-first Epic 23. После
  реализации Epic 20 primary IA `Home / Runs / Knowledge / Changes`, contextual Setup и global
  Ask заменяют требование о восьми обязательных numbered stages; backend/runtime contracts,
  local-first boundary, deterministic required CI и release live-E2E guardrails сохраняются.
- Historical delivery order, обязательные contract-first PR, code/test map, cutover/rollback и
  reference matrix были зафиксированы в
  [`UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md`](UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md).
  План отображает эти же `20A–20N` и не является вторым roadmap.
- Исходный trust audit подтвердил несколько defects: selected historical run терял обязательный
  `staged_path`, UI обещает `Commit selected artifacts`, хотя backend выполняет `git add -A`,
  client-only runtime selector может расходиться с effective server runtime, а fake walkthrough
  может выглядеть как publication-ready evidence. Epic 19 уже доставил typed staged-path mapping,
  containment/run-identity validation и run-keyed loading; для `20A` остаются sufficiency audit,
  explicit source-mode UX, fail-closed edge semantics и rendered same-path multi-run proof.
- Оставшийся UI debt — raw Markdown через `<pre>` в Review/Proposals/Publish, presence-based
  stage statuses, permanently disabled approval actions, обязательный вид `Ask` в rail,
  перегруженный first viewport, неполные keyboard contracts и монолитные
  `StagePanels.tsx` / `styles.css`.

Product decisions for this epic:
- historical run review is an immutable run snapshot; current canonical workspace is a
  separate, explicitly labelled mode and is never a silent fallback;
- MVP Git action remains full-workspace commit because that is the current backend contract;
  UI calls it `Commit all workspace changes`, shows the full change inventory and requires
  confirmation; commit/branch commands carry the confirmed inventory/HEAD identity and fail with
  typed `409` if it is stale before mutation; exact file/folder-scoped commit needs a separate
  owner-approved API/schema slice;
- Git mutation is available only in Publish; Charter and other workbenches link to Publish
  instead of bypassing its gate;
- fake stays the recommended deterministic walkthrough, but its outputs are labelled
  `Demo evidence` and never inherit normal live-evidence readiness; an intentional fake
  commit uses a distinct `Commit all demo workspace changes` confirmation;
- runtime/provider identity is process- and run-scoped: current effective values come from
  server readback; historical runs use persisted run-start `runtime_mode` plus
  `step_providers` (shown as one provider or `Mixed`), never the current client selection;
  the existing runtime switch is limited to launcher/first-run Setup before console entry and
  without active/pending runs; an in-console desired choice remains a Settings-session draft,
  never changes effective service state and shows exact restart guidance until a new process
  readback confirms it;
- Charter/analysis brief remains recommended, not mandatory: skipping it is an explicit
  choice with a quality warning;
- Review and Proposals stay honest read-only inspection surfaces until a persisted review
  decision contract exists; disabled `Approve` affordances are removed;
- unresolved open questions are visible publication risks and require explicit confirmation,
  but are not silently converted into an undocumented hard blocker;
- Ask remains async and read-only over the current canonical workspace, is labelled with that
  context and becomes a global utility rather than a mandatory step;
- guided operator content is the default; raw paths, runtime telemetry, permission internals
  and advanced settings move behind an expert/diagnostics disclosure.
- any slice that introduces tabs, dialogs, async states or navigation must create/reuse the
  minimum semantic primitive in that slice; `20K` consolidates and migrates the remaining UI,
  not postpones component contracts until after new screens are built.
- `20B`, `20C` and `20F` start with a current-payload sufficiency check. If required Git,
  runtime or queue state is absent, the first PR is contract-only with docs, validators,
  fixtures and backend tests; UI work follows only after readback is authoritative.

Priority and dependency order:
- P0 trust foundation: `20A` first; `20B` and `20C` can then proceed independently;
  `20D` depends on `20C`.
- P1 decision workflow: `20E` depends on `20A–20D`; `20F` depends on `20C`;
  `20G` depends on `20A`; `20H` can run after `20B`.
- P1/P2 structure and craft: `20I` depends on `20E`; `20J1` on `20I + 20F`, `20J2` on
  `20I + 20G`;
  `20K` follows the trust components from `20G/20H/20I`; `20L` depends on `20I + 20K`;
  `20M` depends on `20E + 20G + 20J`.
- `20N` closes the program, but every preceding slice must add its own focused tests rather
  than deferring coverage to the final slice.

Acceptance:
- selecting two runs with the same `canonical_path` and different content always renders each
  run's own staged bytes, coverage and open questions; missing/corrupt snapshot data becomes
  an explicit error and never falls back to latest canonical content;
- Changes and Publish identify `run_id`, persisted run runtime/provider set, generated time and
  `Run snapshot` versus `Current workspace` context in the primary header;
- every full-workspace commit path accurately describes its scope, shows the complete
  new/modified/deleted/untracked/renamed/copied/changed inventory covered by the action, handles
  binary/unavailable diffs honestly and requires confirmation; the backend rejects a changed
  inventory/HEAD with typed `409` before mutation; proposal-branch creation has a separate branch
  name/base confirmation with the same stale-state protection and both mutation types are
  unavailable outside Publish;
- effective runtime/provider/permission mode comes from server readback; a desired setting
  cannot be presented as active until a restarted process confirms it through readback;
- fake output is visibly demo/wiring evidence in Run Studio, Changes and Publish and requires a
  distinct intentional confirmation before commit;
- one shared workflow selector drives PrimaryNav status, Home/PageHeader/ContextDrawer next action,
  Changes blockers and Publish gate without contradictory ready/blocked states;
- normal run commands return typed `409` while a run is active; only an explicit queue intent may
  create/replace the single pending run, names its pipeline and exposes typed supersession identity;
- Markdown artifacts are readable by default with safe Rendered/Raw/Diff modes, source
  provenance and deterministic loading/empty/error/large-content states;
- primary IA is grouped as `Home / Runs / Knowledge / Changes`; Setup is contextual, Ask and
  Settings/Diagnostics are utilities, browser Back/Forward works, and URL state restores at least
  view, run, source mode and artifact/entity;
- on `1440`, `1280`, `1024` and `390x844` viewports there is no document-level horizontal
  overflow; destination/page title, current state and primary next action fit in the first mobile
  viewport;
- tabs, combobox, forms, async status/error announcements and destructive confirmations meet
  their keyboard/screen-reader contracts and automated accessibility checks have no critical
  violations;
- required CI remains fake/fixture-driven and network-free; release-facing provider-live
  coverage stays on the canonical runbook and existing scenario/reason taxonomy unless a
  separate release-gate slice is approved;
- each implementation slice updates tests and relevant docs and completes
  `make contracts`, `make test`, `make lint`, `make build` before commit.

Non-goals:
- no hosted mode, security/compliance enforcement or source-repository writes;
- no artifact schema change for run-pinned review: `staged_path` already exists in the
  canonical final-run index and must be consumed rather than reinvented;
- no persisted evidence/proposal approval workflow in this epic;
- no exact selected-file Git commit hidden inside a UI-only change;
- no one-shot visual rebrand or big-bang rewrite of `StagePanels.tsx`; keep the current calm
  palette/dark navigation identity and migrate touched vertical slices incrementally;
- no canonical release matrix, curated repo list, live reason taxonomy or provider set changes.

Suggested PR slices:
- `20A Run-pinned evidence snapshot truth` (P0, first selected slice)
  - Epic 19 уже доставил базовые typed snapshot contracts и run-keyed artifact loading; перед
    реализацией этого slice провести sufficiency audit и оставить здесь только недостающие
    source-mode, fail-closed, rendered multi-run и migration-proof requirements
  - preserve top-level `run_id`/`generated_at` and document `staged_path` in the TypeScript
    final-index contract and build selected-run refs from staged paths, including coverage and
    open-question documents
  - require index `run_id` to equal the selected run and every staged document path to remain
    under `reports/taskruns/<run_id>/staging/final/`; cross-run/out-of-root paths fail closed
  - load preview/coverage/questions as one run-id-keyed snapshot transaction with abort or
    stale-response suppression so rapid `A -> B -> A` switching cannot mix responses
  - add explicit `Run snapshot` / `Current workspace` source mode; never silently mix modes
  - default History/Review selection to `Run snapshot`; `Current workspace` is an explicit user
    action, never an error fallback
  - distinguish `Not produced for this run` (optional document absent from the index) from
    `Snapshot unavailable` (indexed staged file cannot be read)
  - show run id, generated timestamp and source mode beside selected evidence; historical
    runtime/provider identity follows in `20C`
  - tests: two runs, same canonical paths, different bytes; rapid A/B/A switching; mismatched
    index run id; cross-run staged path; missing indexed file; optional absent document; stale
    canonical file; selected-run coverage/open-questions isolation
  - expected code surface: `ui/src/lib/appContracts.ts`,
    `ui/src/hooks/useRunArtifacts.ts`, `ui/src/hooks/useRunActions.ts`,
    `ui/src/hooks/useRunExplorer.ts`, Review/Publish headers, `ui/src/App.test.tsx` and a named
    deterministic `historical-run-snapshot` rendered scenario
  - exit: rendered multi-run proof shows the older run's bytes after a newer promotion

- `20B Honest full-workspace Publish boundary` (P0; depends on `20A` for snapshot labelling)
  - PR `20B1`: add an authoritative Git change-inventory read contract covering
    modified/deleted/untracked paths, with API docs, validators/fixtures and backend tests
  - PR `20B2`: implement the truthful Publish UI only after the inventory contract is sufficient
  - rename the action to `Commit all workspace changes` and label artifact selection as preview,
    not commit scope
  - show branch, tracked modifications, deletions and untracked paths covered by `git add -A`;
    block commit if that inventory cannot be loaded
  - fingerprint branch/HEAD, normalized status/path/original path, mode/binary state, index
    identity and content hashes; counts are not a confirmation identity
  - require a confirmation summary with change counts, unresolved-question warning and the
    selected evidence context; cancel performs no mutation
  - keep proposal-branch creation as a separate confirmation naming branch/base; it must not
    inherit commit-scope copy
  - send expected fingerprint/HEAD with commit and expected fingerprint/source branch/base/HEAD
    with proposal-branch; serialize backend validation and return `409 stale_git_confirmation`
    without side effects when the confirmation is stale
  - remove commit/branch mutation controls from Charter and route users to Publish
  - tests: dirty file outside selected run is visible; untracked file is visible; cancel is
    side-effect free; inventory/HEAD change after dialog open is rejected; concurrent confirmation
    is serialized; Charter has no mutation path; successful commit refreshes real Git state
  - expected code surface: `BaselineGitPanel.tsx`, `StagePanels.tsx`, `useGitActions.ts`,
    workspace Git API/handler tests; contract docs are updated if inventory fields are added

- `20C Effective runtime identity and restart boundary` (P0)
  - PR `20C1`: add/confirm an authoritative read contract for current effective runtime,
    provider/step-provider set and permission mode, plus persisted per-run `runtime_mode` and
    `step_providers`; use one provider label or `Mixed` for multi-provider runs
  - PR `20C2`: consume that contract in the UI after contract docs/fixtures/backend tests pass
  - make server readback the source of truth for current effective runtime/provider/permissions
    and persisted run metadata the source for historical Changes context
  - display desired and effective values separately; use `Pending restart` until a restarted
    process reports the desired configuration
  - make run preflight/GlobalHeader/Run Studio use effective values, never a client-only selection
  - preserve immediate runtime selection only as a launcher/first-run Setup exception before
    console entry and without active/pending runs; outside it `/api/onboarding/runtime` returns
    `409 runtime_switch_requires_restart`
  - do not call that launcher endpoint from in-console Settings; keep the desired choice as an
    explicit session draft, provide an exact restart command/instruction and refresh readback after
    reconnect; do not promise reload persistence without a future persisted process-preference
    contract
  - tests: UI selects headless while server remains fake; simulated server restart/readback
    success; mixed step providers; missing historical metadata; readback/reconnect error;
    direct mode and onboarding first-run identity
  - expected code surface: server status/readiness API, `internal/api/onboarding.go`,
    `internal/api/server_test.go`, `ui/src/lib/onboardingApi.ts`, `appContracts.ts`, runtime hooks,
    `TopStatusBar.tsx`, Readiness and Analysis; invoke schema/contract synchronization if the
    public payload changes

- `20D Fake/demo evidence boundary` (P0; depends on `20C`)
  - add persistent `Deterministic demo` identity to run summary, Changes trust status and Publish;
    pair it with the separate `Demo evidence` trust label wherever evidence readiness is shown
  - replace `Evidence ready` / normal `READY` semantics for fake with `Demo evidence`
  - keep fake publication available only as `Commit all demo workspace changes`
    with a warning that no live architecture analysis occurred
  - tests: fake can complete walkthrough, never appears live-ready, normal headless run keeps
    live evidence semantics, demo confirmation is required before Git mutation
  - expected migration inputs: shared run identity selector, current
    Analysis/Review/Publish/inspector copy, fake Playwright fixtures and README walkthrough wording;
    target outputs are Run Studio, Changes, Publish and ContextDrawer

- `20E Shared workflow status and publication gate` (P1; depends on `20A–20D`)
  - replace presence-based `done` with `available / needs_review / blocked / complete` semantics
  - derive one typed gate model from selected snapshot health, validation/doctor state,
    UI-visible promoted evidence blockers, proposal package, open questions, fake/live identity
    and Git mutation errors
  - preserve Epic 18 black-box boundary: `reports/taskruns/*-quality.json`, runtime telemetry
    and matrix counters never decide artifact acceptance in the product gate
  - use the same derivation/precedence table in PrimaryNav, Home, PageHeader, ContextDrawer and
    Publish; prevent a green
    top-level status when a detailed gate is blocked
  - remove permanently disabled evidence/proposal approval buttons and offer only real
    navigation/recovery actions
  - tests: table-driven state matrix for partial artifacts, proposal blocker, open questions,
    fake, failed Git action, clean post-commit workspace and stale selected run
  - expected migration inputs: `stageModel.ts`, `App.tsx`, current Review/Proposals/Publish and
    inspector components; target output is a pure workflow selector consumed by Home/Changes/Publish

- `20F Deliberate run start, refresh and queue semantics` (P1; depends on `20C`)
  - PR `20F1`: expose active identity and the single replaceable pending run
    identity/pipeline plus typed `superseded_by_run_id` through a contract-first read model with
    docs/fixtures/backend tests
  - make the command contract deliberate: ordinary start while active returns `409 run_active`;
    only explicit queue intent can create or replace pending, and supersession has a typed
    `error_code` rather than free-form text parsing
  - PR `20F2`: implement deliberate queue controls against that authoritative read model
  - disable ordinary init/refresh actions while a run is active across Run Studio, Home and
    ContextDrawer
  - expose `Queue refresh after current run` only when existing backend queue semantics apply;
    show pending run id/pipeline and replacement/supersession rule before confirmation
  - keep last accepted evidence selected if a start request fails
  - tests: stale UI/double click cannot silently enqueue, explicit queue, active run, pending
    replacement with typed supersession, request failure, cancellation and post-terminal queue
    transition
  - expected migration inputs: run service/API read model, run hooks, current Analysis mission
    control, active-run strip and inspector; target output is Runs/Run Studio while preserving
    single-active/debounce backend invariants

- `20G Shared evidence-first ArtifactViewer` (P1; depends on `20A`)
  - render Markdown safely by default with headings, lists, tables, fenced code, local evidence
    links and Mermaid; raw HTML stays disabled
  - provide consistent `Rendered / Raw / Diff` modes and source metadata in Changes,
    Knowledge/Atlas and Publish through shared Evidence Studio
  - add loading, empty, unavailable snapshot, parse error and large-content behavior without
    replacing content with generic cards
  - tests: tables/code/links/Mermaid, script injection, broken artifact, long lines, diff view,
    keyboard mode switch and snapshot/current-workspace source badge
  - expected code surface: new `ArtifactViewer`/Evidence Studio component, artifact-link resolver,
    current Review/Proposals/Publish migration integrations and focused component tests

- `20H Accessibility-critical interaction contracts` (P1; can follow `20B`)
  - implement WAI-ARIA tab keyboard behavior: roving tabindex, arrows, Home/End,
    `aria-controls` and linked tabpanels
  - make local-path combobox support ArrowUp/Down, Enter, Escape, active option and predictable
    focus/blur behavior
  - connect field errors with `aria-invalid`/`aria-describedby` and announce async errors,
    completion and destructive-confirmation results through live regions
  - fix text contrast roles, remove or implement permanently disabled shell controls and make
    local time update correctly or remove it
  - tests: keyboard-only onboarding and artifact viewer, focus return after dialogs/drawers,
    screen-reader names and automated axe checks
  - expected code surface: `TabNav.tsx`, `LocalPathCombobox.tsx`, onboarding fields,
    async feedback primitives, top bar and token styles

- `20I Navigation model and URL-restorable context` (P1; depends on `20E`)
  - PR `20I1 Navigation/IA`: replace the mandatory numbered rail with grouped destinations,
    relocate utilities and add only path-level `/setup|home|runs|knowledge|changes` navigation,
    `popstate` and direct-load SPA fallback
  - replace the mandatory numbered rail with grouped primary destinations:
    `Home / Runs / Knowledge / Changes`
  - land a minimal attention-first Home over the shared selector and authoritative current hooks;
    the new shell is not accepted until all four routes have visible, honest temporary compositions
  - keep first-run Setup as a guided session before the primary shell and expose it later from
    the workspace menu; move Ask to a global read-only utility and Settings/Diagnostics to expert
    access without changing their runtime/API semantics
  - `/knowledge` may remain an explicit partial/unavailable temporary composition until `20J2`;
    it must not derive architecture truth from filenames
  - synchronize README/ARCHITECTURE/current UI baseline/STAKEHOLDER current-shell status in the
    same `20I1` cutover
  - PR `20I2 URL context`: persist and restore deep navigation context after `20I1` is accepted
  - persist at least view, selected run, source mode and selected artifact/entity in URL/history;
    Back/Forward and reload restore a valid context and sanitize missing ids
  - preserve/migrate operator-facing `data-testid` values with unit/fake E2E coverage; do not
    add hidden compatibility controls
  - expected code surface: navigation model, shell/rail, `App.tsx` state boundary, URL tests and
    `internal/api/server_test.go` direct GET SPA-fallback coverage for target routes

- `20J Home, Guided Setup, Runs, Knowledge and Changes composition` (P1/P2; `20J1` depends on
  `20I + 20F`, `20J2` depends on `20I + 20G`)
  - PR `20J1 Home + Guided Setup + Runs`: change attention summary, first-run preparation and run
    mission control only
  - make Home answer workspace/readiness, active/pending execution, latest evidence and current
    workspace publication state with one non-contradictory next action
  - compose Setup from Workspace, Sources, recommended Analysis brief and Runner/Readiness;
    make `Run without brief` an explicit quality trade-off
  - make Runs answer first: what is running, current step/blocker, next action and history;
    keep shard/raw telemetry behind diagnostics
  - PR `20J2 Changes + Knowledge + Ask context`: recompose evidence/decision work, current
    architecture knowledge and contextual utility
  - compose Architecture Change Review around evidence queue, findings/gaps, proposals and
    publication using the shared viewer and gate; compose Knowledge around Overview, Atlas,
    entities and artifacts with explicit partial states
  - make Changes list route successful `init|refresh` snapshots to review, failed/canceled/recovered
    runs to retained-evidence Run Studio, and QA runs to Ask history; per-run publication remains
    `Unknown` without an authoritative run-to-commit association
  - label Ask as `Current workspace` context until run-scoped Q&A has an explicit contract
  - tests: first-time guided walkthrough, explicit Charter skip, failed run recovery,
    historical review, proposal blocker and contextual Ask entry
  - expected migration inputs: onboarding transition, current stage composition and
    Analysis/Review/Ask containers; target outputs are Home, Runs, Knowledge, Changes and global
    Ask while pipeline semantics stay unchanged

- `20K Semantic UI consolidation and density` (P2; after primary trust components stabilize)
  - define semantic roles for text, surface, border, action, status and focus plus a compact
    type scale, 6–8 spacing steps, three radii and compact/comfortable density
  - consolidate the `Button`, `Tabs`, `PageHeader`, `ContextBar`, `RecoveryPanel`,
    `Metric/DefinitionList`, `DataTable/CardList` and async state primitives introduced/reused
    by earlier vertical slices; do not rebuild accepted interactions
  - migrate only components touched by Epic 20; remove nested card borders where section
    heading, alignment and whitespace express the hierarchy better
  - bundle the intended UI font or make the system-font contract explicit and deterministic
  - tests: token reference/lint, component variants and states, focus/contrast checks and
    representative visual fixtures
  - expected code surface: token/style layers, `ConsolePrimitives.tsx` and touched stage modules

- `20L Responsive shell and first-viewport budget` (P2; depends on `20I` and `20K`)
  - at `1024–1439px` use compact navigation and an inspector drawer instead of reserving three
    permanent columns
  - after idle/success collapse active-run chrome to one summary line; on mobile show one health
    summary with collapsible run detail before destination content
  - convert decision tables to mobile cards where comparison is not essential; keep intentional
    scroll only for true tabular comparison
  - viewport gates: no document overflow at `1440/1280/1024/390`, workbench at least `720px`
    wide at `1024`, and mobile destination title/state/primary action visible above `y=520`
  - tests: settled and loading screenshots, drawer focus/escape behavior, long paths/text,
    Changes/Run Studio/Publish first viewport and orientation changes
  - expected migration inputs: shell/rail/inspector/active-run components and responsive styles;
    target output uses PrimaryNav and modal/non-modal ContextDrawer variants

- `20M Changes/Knowledge/Publish module seam and maintainability` (P2; depends on `20E`, `20G`, `20J`)
  - extract target Changes, Knowledge and Publish containers/view models from the monolithic
    `StagePanels.tsx`, using current Review/Publish panels as migration inputs while preserving
    accepted public props and operator-facing selectors
  - centralize recovery/gate/viewer composition instead of duplicating panel anatomy
  - keep the refactor behavior-neutral beyond already accepted Epic 20 UX decisions; split
    other stages only when their own vertical slice is touched
  - tests: existing Review/Publish suite remains green, pure selectors receive table-driven
    coverage and bundle/typecheck show no circular stage dependencies

- `20N Task-based UX regression gate, docs and rollout` (P2; closes the epic)
  - add deterministic fixtures for two-run snapshot isolation, dirty files outside the selected
    artifact, fake/live identity, desired/effective runtime mismatch, active/pending queue and
    partial artifact packages
  - add task assertions, not only no-overflow checks: first viewport answers current state,
    blocker and next action; commit confirmation names full scope; Back/Forward restores context
  - run component/a11y tests and rendered fake scenarios at `1440/1280/1024/390`; keep provider
    live runs manual and classify their evidence through the existing release runbook
  - update README, ARCHITECTURE, implemented UI baseline, TESTING_STRATEGY, stakeholder matrix and
    implementation screenshots only after the corresponding behavior is implemented; planned
    design references remain clearly labelled as non-runtime artifacts
  - exit: a first-time user completes guided fake walkthrough without mistaking demo output for
    live evidence, and an experienced operator can inspect raw diagnostics without cluttering
    the default decision flow
## Epic 21 — Evidence-backed Architecture Home + Impact-aware Refresh (Wave 1)

Status (2026-07-15): complete. `21A`–`21G` implemented with schema-validated planning/execution/materialization evidence, provider-free no-op, fail-closed affected-only collect, byte-preserving promotion decisions and operator explanation. Epic 18 R3 remains a separate trusted-host release gate.

Context:
- ProvenArch already has the stronger trust and governance foundation: a separate Git-versioned
  architecture workspace, read-only source repositories, staged runtime outputs, validator-gated
  promotion, provenance/evidence, domain-first execution and entity-per-file model artifacts;
- the current `reports/as-is/overview.md` is the default Review entrypoint, but it is still treated
  primarily as one generated report rather than as the stable home page for navigating the whole
  architecture workspace;
- the current `refresh` pipeline reuses the init step shape and does not first persist a
  deterministic explanation of which source revisions/files changed, which domains/shards are
  affected, which canonical artifacts may be stale, or why a refresh can safely be a no-op;
- this epic adopts the useful documentation-maintenance ideas commonly associated with wiki agents
  without changing ProvenArch into a source-repo-writing docs bot or weakening its evidence and
  validation boundaries.

Goals:
- make `reports/as-is/overview.md` the canonical human-readable architecture home for Review and
  Ask navigation;
- add concise documentation-quality rules to `step2.asis_docs` and compatible answer-quality rules
  to `qa.ask`;
- capture a deterministic source revision baseline for each resolved repository and compare refresh
  inputs with the last successful validator-promoted `init|refresh` run;
- use bounded recent Git history as secondary intent evidence while keeping current source files and
  validated repository evidence authoritative;
- compute and persist a deterministic refresh impact plan before `refresh.step1.collect`;
- support explainable no-op refreshes and affected-only collect execution;
- preserve unaffected canonical artifacts byte-for-byte once selective downstream promotion is safe;
- expose what changed, why it matters, what was refreshed, what was preserved and what remains
  uncertain through existing Analysis/Review/Publish surfaces.

Non-goals:
- do not write generated documentation or commits into analyzed source repositories;
- do not replace the central `arch-workspace`, entity-per-file model, provenance, staged promotion,
  validators, domain/team cards or multi-provider runtime model;
- do not mine the full Git history, treat commit messages as stronger evidence than source code, or
  ask the provider to infer the refresh scope without a deterministic plan;
- do not classify an in-scope but unmapped source change as safe no-op; fall back to conservative
  refresh and record the mapping gap;
- do not add hosted scheduling, webhook delivery, external documentation sources or new providers;
- do not rename the product to a wiki product; `architecture home` / `wiki-like workspace` is a UX
  explanation over the existing evidence-backed architecture workspace.

Acceptance:
- `reports/as-is/overview.md` answers: what the system is, analyzed repo/scope, domains, where to
  start reading, key flows, datastores/integrations, safe-change guidance and evidence gaps; it links
  to deeper canonical artifacts instead of duplicating their full contents;
- operator-facing docs avoid raw file inventories, manifest recaps, runtime/process narration and
  unsupported certainty; they use short sections, explicit gaps, concrete repo/path references and
  `what changed / why it matters` where refresh evidence exists;
- Ask continues to be read-only, cites workspace artifacts, distinguishes confirmed facts from gaps,
  and uses the architecture home as a high-priority navigation source without treating it as the
  only evidence source;
- each resolved repo in a refresh has an exact current commit identity and a baseline identity from
  the last successful promoted architecture run; missing baseline, dirty local worktree, rewritten
  history, unavailable commit or ambiguous mapping is explicit and forces conservative behavior;
- the deterministic planner computes the complete changed path set for a valid baseline range,
  maps it through effective include/exclude scope plus previous shard/domain/evidence ownership, and
  never silently truncates planning input; oversized or unresolvable ranges fall back to full refresh;
- provider context receives only bounded history evidence for affected scope (recent commit summaries
  plus relevant changed paths), and prompts state that commit text explains possible intent but does
  not override current source evidence;
- a persisted refresh impact plan identifies source deltas, affected/unmapped domains and shards,
  potentially stale canonical artifacts, planned actions, preserved artifacts and fallback reasons;
- safe no-op is limited to unchanged clean revisions or changes wholly outside effective analysis
  scope; it succeeds without provider execution or canonical artifact rewrites and retains taskrun
  evidence explaining the decision;
- affected-only collect dispatch preserves existing deterministic shard ordering and failure policy;
  an unmapped in-scope change triggers conservative collect instead of being ignored;
- selective downstream promotion carries forward unaffected artifacts only when their baseline run,
  evidence dependencies and validator status are known; preserved files remain byte-identical and
  final indexes record a coherent complete publication set;
- deterministic fake/runtime fixtures cover unchanged, out-of-scope-only, one-domain, multi-domain,
  dirty worktree, missing baseline, history rewrite and conservative fallback cases;
- required CI remains provider-independent; live providers are used only for optional/manual quality
  validation after deterministic tests pass.

Suggested PR slices:
- `21A Architecture home + documentation quality baseline`
  - **done:** canonical Architecture Home sections, validation, fake output and QA navigation priority
  - redefine the existing `reports/as-is/overview.md` authoring policy as the architecture workspace
    home while keeping its canonical path and current Review default selection
  - add the required navigation/content sections and concise human-readable quality rules to
    `step2.asis_docs`; extend strict validation only for machine-checkable failures
  - prioritize the overview as a QA navigation document and align `qa.ask` answer guidance with
    explicit gaps, concrete citations and concise answers
  - update fake artifacts plus focused prompt/runtime-draft/QA tests without schema changes
- `21B Source revision baseline contract`
  - **done:** persisted schema-validated source revisions, conservative baseline selection and analysis-input fingerprint
  - specify and persist per-repo current revision, previous successful promoted revision, source kind,
    effective scope and conservative-fallback reason under taskrun scope
  - define the exact baseline selection rule: latest successful validator-promoted `init|refresh`, not
    an inferred human Git acceptance event
  - treat dirty local worktrees, missing commits and non-ancestor/history-rewrite cases explicitly
  - synchronize schemas, `docs/spec/*`, appendix, examples, fixtures, validators and ADR rationale
- `21C Deterministic refresh impact plan`
  - **done:** persisted advisory plan with complete Git delta accounting, mapping and fail-closed fallback
  - compute changed paths before `refresh.step1.collect`
  - map paths to effective include/exclude scope, prior shard path scopes, domain IDs, citations/model
    provenance and canonical document dependencies
  - persist affected/unmapped shards/domains, stale/preserved artifact candidates, planned actions and
    conservative fallback reasons as a validated taskrun artifact
  - process the complete changed path set for planning; if safety limits are exceeded, mark full
    refresh instead of silently dropping paths
- `21D Explainable no-op refresh`
  - **done:** safe provider-free no-op, factual execution audit and CLI/API/UI explanation
  - skip provider steps only for unchanged clean revisions or out-of-scope-only changes
  - finish the refresh successfully with taskrun impact evidence and no canonical report/model/
    proposal/changelog rewrites
  - surface the no-op reason and source range in CLI/API/UI run status using existing extension
    patterns or a separately synchronized contract change
- `21E Affected-only collect execution + bounded Git intent evidence`
  - **done:** validated baseline checkpoint replay, fail-closed fallback and bounded secondary Git intent
  - dispatch only impacted existing shards/domains while preserving deterministic ordering and current
    failure semantics
  - pass affected-path context plus bounded recent commit summaries to collect tasks
  - fall back to full collect for any in-scope unmapped or ambiguous change
  - prove that source evidence wins over stale/incorrect commit messages in prompt and fixture tests
- `21F Surgical downstream materialization and promotion`
  - **done:** dependency candidates, byte-identity preservation and updated/preserved/removed/uncertain audit contract
  - rebuild global summaries only when their dependency set changes
  - update affected domain docs/model entities/edges/findings/proposals while carrying forward known-good
    unaffected artifacts from the explicit baseline run
  - keep preserved canonical files byte-identical and validate the merged final index/citation set
  - record `updated / preserved / removed / uncertain` artifact decisions for Review and Publish
- `21G Operator explanation, product language and release validation`
  - **done:** Runs/Changes summaries, legacy state, synchronized docs and deterministic UI gates
  - show impact summary, no-op reason and artifact decisions in Analysis/Review/Publish without adding a
    new product stage
  - keep pre-maintenance copy truthful (`builds an evidence-backed architecture workspace`) and use
    `builds and maintains` only after no-op/selective refresh behavior is implemented and tested
  - update README/ARCHITECTURE/STAKEHOLDER docs and run deterministic UI plus optional trusted-machine
    live artifact-quality validation

## Epic 22 — Post-implementation correctness and trust-boundary audit remediation

Status (2026-07-27): **complete; Epic 18 R3 remains the release blocker.** Slices `22A`–`22O` and
the combined provider-free closure pass at clean qualification commit
`e8055d65699ed63623f62ad99c3b8406f79c030d`. No stopped live matrix is reinterpreted as release
evidence.

### Goal

Make ordinary product execution, evidence selection, refresh preservation and ProductShell state
fail closed under races, filesystem aliases, stale requests and malformed artifacts. Prove that the
trusted-machine harness and product remain independent in both directions before restarting R3.

### Delivery order

`22A -> 22B -> 22C -> 22D -> 22E -> 22F -> 22G -> 22H -> 22I -> 22J -> 22K -> 22L -> 22M -> 22N -> 22O -> Epic 18 R3`

Each slice is a separate reviewable implementation unit with its own ExecPlan. By explicit owner
request, the product queue `K2b -> K4 -> K3A -> K3B -> 9D -> cleanup` was also completed locally
before R3 without treating that work as live qualification. Epics 12/13 and K5–K7 remain deferred.

### Boundaries

- Required CI and all Epic 22 acceptance are provider-free and have no live network dependency.
- No Epic 22 test or product branch may run or adapt the canonical live matrices.
- Source repositories remain read-only; all writes stay inside the selected workspace.
- Product runtime/API/UI must not know matrix IDs, release verdicts, assessment filenames,
  profile/sweep/batch labels or live-only environment variables.
- Live E2E may use only the canonical harness and public CLI/API/UI/artifact surfaces. It must not
  import production internals, author product state or repair provider/workspace artifacts.
- Public schema/API changes are allowed only when a slice proves they are necessary and performs
  the full schema/spec/appendix/examples/fixtures/validator/ADR synchronization.

### Entry prerequisite

Before `22A`, use a host/worktree with the required pinned Go/Node/npm toolchains and enough free
space for caches plus parallel test/build output (minimum 5 GiB on the relevant workspace/temp
volume). Run the deterministic preflight and leave the tree clean. Low disk or a missing pinned
toolchain is an operational blocker; it is not permission to skip a slice DoD or change product
behavior.

### 22A — Immutable run registry and transactional history

Status (2026-07-26): **complete**. Run snapshots are deep-cloned at storage/read/history boundaries;
registry candidates persist primary current history before in-memory publication; pending
replacement is one transaction; persistence/recovery diagnostics are bounded and exposed through
the runs list; cancellation uses the existing terminal `canceled` status. Focused race/fault/restart
coverage and full deterministic DoD pass without live E2E. `22B` is next.

What:
- return deep immutable copies of run records, nested slices, coordination and refresh summaries;
- serialize run-registry transitions and make queued/running/terminal history updates transactional;
- stop ignoring terminal persistence failures and preserve a visible diagnostic without publishing
  an in-memory state that cannot be recovered after restart;
- represent cancellation consistently as a terminal canceled outcome instead of an incidental
  generic failure; synchronize the public contract first if a status-shape change is required.

Why: concurrent polling and completion currently have a reproducible shared-memory race, while a
failed history write can leave API state and restart state disagreeing.

Acceptance:
- race detector coverage for polling, refresh-summary mutation, cancel, pending replacement and
  completion is clean;
- injected write/rename/fsync failures never leave malformed current or `.last-good` history;
- restart reconstructs exactly one active/pending/terminal truth and never resurrects a canceled run.

### 22B — Symlink-safe workspace containment and atomic manifest writes

Status (2026-07-26): **complete**. Workspace-owned reads, layout creation and atomic writes use
Go's descriptor-backed `os.Root`: relative symlinks are accepted only when they remain inside the
workspace, while absolute, dangling and escaping links fail closed. Workspace creation now writes
`workspace.yaml` through the same temp/fsync/rename/dir-sync primitive. Focused symlink and race
tests pass; `22C` is next.

What:
- enforce containment against resolved filesystem identity, including existing symlink parents,
  final symlinks and creation through a symlinked ancestor;
- centralize the safe read/write/open primitive used by workspace-owned paths;
- write `workspace.yaml` with the same same-directory temp, sync, rename and directory-sync contract
  as other critical workspace files.

Why: lexical path checks alone allow a path inside the workspace spelling to resolve outside it;
direct manifest truncation can leave the workspace unreadable after an interrupted write.

Acceptance:
- symlink escape, traversal, dangling-link, replacement and concurrent rename fixtures fail closed;
- valid in-root symlinks follow one documented policy consistently;
- failed manifest writes preserve the previous bytes and source repositories remain unchanged.

### 22C — Server-owned selected-run snapshot resolver

Status (2026-07-26): **complete**. `GET /api/pipeline/runs/<run_id>/snapshot` owns exact
final-index selection, run identity, staged containment, inventory membership and canonical
mapping checks. It returns typed states/issues; the browser no longer parses final indexes or
constructs staged paths. Late proposal documents are persisted in run inventory. `22D` is next.

What:
- move final-index discovery, identity checks and artifact resolution behind one server-side resolver;
- require exact selected `run_id`, normalized in-root staged paths, final-index inventory membership
  and an unambiguous canonical-path-to-staged-path mapping;
- return typed `available | partial | not_produced | unavailable | error` status and issues;
- forbid snapshot fallback to current workspace, another run, a suffix-matched foreign index or an
  unindexed file.

Why: client-side composition and permissive path resolution can expose cross-run or newer workspace
bytes while the UI claims to show the selected historical run.

Acceptance:
- missing/mismatched `run_id`, traversal, cross-run path, duplicate canonical mapping, stale index,
  missing artifact and out-of-root target all produce deterministic typed failures;
- A -> B -> A selection and reload always return the exact same-run bytes or an explicit issue.

### 22D — One recursive glob dialect

Status (2026-07-26): **complete**. `internal/pathscope` is the only compiler/matcher for repository
analysis scopes. Refresh impact mapping and shard planning share the same slash-normalized
segment dialect (`*` one segment, standalone `**` recursive, literal directory includes its
subtree); invalid/absolute/traversal patterns fail manifest validation before run admission.
`22E` is next.

What:
- define one documented recursive include/exclude matcher for workspace validation, shard planning,
  source fingerprints, refresh impact mapping and QA/import collection;
- reject unsupported or ambiguous patterns before execution;
- normalize separators and ordering without silently changing the user's scope.

Why: inconsistent `**` handling can classify an in-scope change as out of scope and incorrectly
select no-op or preserved evidence.

Acceptance:
- a shared conformance table covers root files, nested files, `**`, exclusions, Windows separators,
  invalid patterns and multi-repo scope;
- every consumer produces identical matches and no false no-op is possible from matcher drift.

### 22E — Immutable selective-refresh baseline and complete shard identity

Status (2026-07-26): **complete**. Successful collect shards now receive an orchestrator-owned
`baseline-integrity.json` sidecar with full logical identity, source revision ranges and a
deterministic SHA-256 inventory. Selective reuse validates that sidecar plus manifest/runtime
identity, rejects symlinks/special files, rewrites only the new run identity and falls back to full
before provider execution on any mismatch. Preserved canonical documents are copied from the
baseline run's staged final snapshot, never from mutable current canonical paths. `22F` is next.

What:
- preserve unaffected artifacts only from validator-promoted immutable taskrun staging bytes and
  recorded digests, never from mutable current canonical files;
- match baseline shards by the full identity: repo scopes, domain, shard ID, path scopes, source
  revisions and artifact digests;
- use canonical model paths and bounded filenames while retaining full logical IDs.

Why: current-workspace bytes may have changed after the baseline run, and partial shard identity can
replay evidence authored for a different scope.

Acceptance:
- post-baseline canonical edits, shard-ID collisions, changed path scopes, long IDs, missing packs and
  digest mismatch all force a full fallback before provider execution;
- genuinely preserved artifacts remain byte-identical and carry verifiable baseline provenance.

### 22F — Session-generation lease and Git/run coordination

Status (2026-07-26): **complete**. One server admission lease now serializes pipeline/QA start
commit, workspace/runtime/session replacement and Git commit/proposal-branch mutations. Starts
hold the lease through workspace validation, generation revalidation and service registration;
Git mutations reject active/queued work and revalidate full confirmation state under the same
lease immediately before mutation. Deterministic barrier and unchanged-HEAD tests cover both race
directions. `22G` is next.

What:
- acquire one serialized start lease spanning session-generation validation, run registration and
  active/pending publication;
- prevent workspace/runtime replacement while a start is being admitted or work is active/pending;
- block Git commit/proposal-branch mutations while active/pending work exists;
- revalidate session generation, branch, HEAD, base and full inventory immediately before mutation.

Why: a switch/start race can orphan a run under a previous service owner, and Git confirmation can
become stale between UI confirmation and mutation.

Acceptance:
- deterministic barriers reproduce switch/start and Git/start races without orphan runs or mixed
  workspace ownership;
- close/shutdown, queued replacement and concurrent confirmations have one serializable outcome.

### 22G — Typed validation issues and explicit recovery state machine

Status (2026-07-26): **complete**. Provider artifact validation is normalized once into an internal
ordered `validationIssueSet` with stable code/class/path fields. Collect/draft repair routing and
provider-unavailable classification consume issue codes rather than `error.Error()` fragments;
recovery transitions name their target stage and one-attempt budget. Claude/Qwen/Codex
order-shuffled fixtures prove equivalent issue sets select the same path, mixed-class draft errors
cannot enter downstream-only cleanup and repeated transitions terminate. `22H` is next.

What:
- replace routing based on fragments of `error.Error()` with typed issue codes, paths and classes;
- define recovery states, class-exclusive transitions, priority, retry/transition budget and an
  auditable terminal reason;
- add preserved Claude/Codex/Qwen incident fixtures plus paraphrase and metamorphic regressions.

Why: historical live failures showed that two simultaneous validation errors can select the wrong
repair even when both individual paths are tested.

Acceptance:
- equivalent issue sets route identically regardless of message wording or order;
- mixed-class failures cannot enter a specialized single-class repair;
- repeated/no-op recovery terminates within a deterministic budget and records the complete path.

### 22H — Provider-free artifact integrity auditor

Status (2026-07-26): **complete**. A read-only `internal/artifactaudit` scanner now audits an exact
selected-run staging graph or compares that graph with promoted-current canonical bytes. It
requires matching final/citation/validator identities, a `PASS` validator verdict, contained staged
paths, reciprocal document/citation references, real contained repository evidence, complete
Architecture Home sections and bounded actionable content. Reports are deterministic, redacted and
bounded by issue, inventory, message and per-file read budgets; the HTTP surface is
`GET /api/pipeline/runs/<run_id>/audit`. Clean, incident, foreign-run, oversized and promoted-digest
fixtures prove read-only behavior. `22I` is next.

What:
- add a read-only auditor over promoted and selected-run evidence;
- verify exact run/index identity, path containment, final/citation reciprocity, concrete evidence
  existence, digests, inventory, Architecture Home completeness and finding/proposal actionability;
- detect foreign-run refs, taskrun staging paths, absolute runtime paths, execution narration,
  scaffold text and fake/live identity contamination;
- emit a bounded redacted audit package suitable for later assessment without raw provider logs.

Why: previous accepted-looking outputs contained staging paths, run narration, false shard recap and
plausible but nonexistent repository references; the most recent raw workspaces are no longer
available for independent reinspection.

Acceptance:
- every preserved historical incident fixture fails with a stable issue code;
- a clean deterministic fixture passes, repeated scans are byte-identical and scanning never writes
  the workspace or source repositories.

### 22I — Bidirectional live E2E/product isolation

Status (2026-07-26): **complete**. Provider subprocesses now receive a filtered ambient environment
that excludes orchestration/release identity segments while preserving normal tool, credential,
locale and proxy variables. Production prompts describe only canonical repo/domain/staged evidence
and generic ambient labels. Live frontend preparation preserves the backend product-authored
`run-history.json` in its read-only snapshot copy instead of synthesizing state, and its diagram
assessment helper lives under `ui/e2e`, outside production ownership. Bidirectional Python source
scans plus Go env and TypeScript helper tests enforce both boundaries. `22J` is next.

What:
- scrub live-only environment and present stable canonical repo/evidence identities to providers;
- remove matrix/profile/batch vocabulary and temporary checkout paths from production prompts and
  fake product artifacts;
- stop frontend live preparation from synthesizing or rewriting product run history/state;
- move any live assessment oracle/helper out of production `ui/src` ownership;
- expand Go/Python/TypeScript boundary tests in both directions.

Why: historical artifacts prove that providers copy ambient runtime paths and execution vocabulary;
the harness must also not become an author of the product state it is supposed to inspect.

Acceptance:
- product code does not read `ACP_RELEASE_*`, `BATCH_*`, matrix/profile/sweep identity or assessment
  files, and promoted documents cannot receive those values through provider context;
- the harness imports no production-internal selector/validator and modifies no canonical workspace
  artifact or run history;
- live fixtures interact only through public CLI/API/UI/artifact contracts.

### 22J — Changes views and workflow truth

Status (2026-07-26): **complete**. Changes uses a discriminated route model for
Overview/Evidence/Findings/Diff/Proposals/Publish and renders named route containers with
route-specific review semantics. `GET /api/git/diff` returns server-authored
`clean|dirty|stale|blocked|unknown`; optional confirmation fingerprint detects stale inventory and
active/pending analysis produces `blocked`. Git actions refuse confirmation for non-actionable
states. Route/state/popstate and backend inventory tests pass. `22K` is next.

What:
- make `Overview | Evidence | Findings | Proposals | Diff | Publish` materially distinct views whose
  URL route/view is the sole selection source;
- derive topology only from validated entity/edge fields, never filenames;
- make the shared workflow selector consume server coordination, the latest authoritative accepted
  evidence and explicit Git `clean | dirty | stale | blocked | unknown` state.

Why: current composition can show duplicate content under different Changes tabs, infer topology
from filenames and choose a next action from stale or incomplete state.

Acceptance:
- route/view table tests prove distinct content and Back/Forward restoration;
- active/pending, no evidence, partial evidence, unknown publication and stale Git confirmation each
  produce one deterministic next action.

### 22K — URL/request identity and stale-response suppression

Status (2026-07-26): **complete**. Artifact snapshot and preview requests use abortable generation
gates; preview identity now includes exact run, source authority, canonical/read paths and viewer
mode. Invalid explicit Changes `view/source/mode` values are removed with a visible notice and
`replaceState` on bootstrap and PopState, while user navigation continues through `pushState`.
Delayed selection, route-table and invalid identity regressions pass. `22L` is next.

What:
- treat `(run_id, source_mode, artifact/entity, viewer_mode)` as one request identity;
- cancel or ignore older requests with generation tokens/abort signals;
- canonicalize defaults and invalid params with `replaceState`, while user navigation uses
  `pushState`; make `popstate` restoration idempotent.

Why: slow responses can overwrite the context chosen later and make the header, bytes and URL refer
to different runs or sources.

Acceptance:
- delayed A -> B -> A fetches, reload, direct URLs and Back/Forward never display stale bytes;
- invalid explicit IDs are sanitized with a notice and never silently fall back to current workspace.

### 22L — Knowledge and QA evidence authorities

Status (2026-07-26): **complete**. Knowledge is explicitly `promoted_current` and excludes every
`reports/taskruns/**` artifact. QA responses name exact `qa_snapshot` and `qa_audit` roots, expose
`answer_status`, validate citations against the selected context pack and return
`qa_answer_unavailable` for a succeeded run whose own snapshot is missing or invalid. Existing
configured-import, historical-selection and canceled/failed regressions remain green. `22M` is next.

What:
- keep current promoted Knowledge free of `reports/taskruns/**` and run-scoped diagnostics;
- define explicit evidence authorities `promoted_current | run_snapshot | qa_snapshot | qa_audit`;
- build QA context from the exact authority, include configured imports, retain selected QA run/return
  context and surface `qa_answer_unavailable` instead of substituting another answer/workspace state.

Why: current-workspace Knowledge, historical Changes and QA audit evidence have different trust and
lifecycle semantics and must not be merged implicitly.

Acceptance:
- authority matrix tests cover current, historical, partial, missing, canceled and legacy runs;
- citations resolve only inside the selected pack and taskrun files never appear as promoted
  Knowledge entities/artifacts.

### 22M — Evidence Viewer correctness

Status (2026-07-26): **complete**. Relative links retain the selected canonical directory, traversal
and taskrun/cross-run escape are blocked, and run navigation refuses paths absent from the selected
snapshot inventory. The viewer exposes `Demo | Live | Unknown`, typed evidence states/issues and
explicit two-sided diff identity. API reads are bounded at 2 MiB and rendered Markdown/Mermaid at
512 KiB with Raw/unavailable fallback. XSS/Mermaid isolation remains strict. `22N` is next.

What:
- resolve local links through the selected evidence authority and preserve the source run;
- make Raw/Rendered/Diff use an explicit and truthful comparison source;
- show `Demo | Live | Unknown`, typed issues and unavailable/partial states consistently;
- impose a tested file-size/render budget with readable fallback for large or broken artifacts.

Why: a polished renderer is still misleading if links, diff baseline or identity header refer to a
different evidence source.

Acceptance:
- link traversal, broken links, cross-run links, XSS, Mermaid failure, long lines and oversized files
  are isolated without replacing the selected document or crashing the page.

### 22N — Responsive and accessibility completion

Status (2026-07-26): **complete**. Tablet collapse controls are removed from focus order, mobile
header/navigation/content/fullscreen sheets honor device safe areas, utility and primary controls
meet the 44 px target, and Run/Knowledge read-only tables become keyed labeled cards on phones.
Modal/drawer tests cover focus trap, Escape, outside click, return focus and live breakpoint
transitions. Offline rendered scenarios exercise the required viewports, overflow and axe. The
combined `22O` working-tree gate also passes.

What:
- remove focusable hidden controls and remaining legacy shell CSS;
- implement safe-area aware mobile navigation/sheets, 44px touch targets and keyed card fallbacks for
  non-comparison tables;
- verify focus trap, Escape, outside click, return focus, long paths and orientation changes.

Why: static review found reachable hidden focus targets and responsive assumptions that are not
fully protected by the current component tests.

Acceptance:
- rendered ProductShell scenarios pass at `1440`, `1280`, `1024` and `390x844` with no global
  horizontal overflow, console errors or critical axe violations;
- keyboard-only flows reach Home, Runs, Knowledge, Changes, Setup and global Ask with logical focus.

### 22O — Deterministic offline closure gate

Status (2026-07-27): **complete at qualification SHA
`e8055d65699ed63623f62ad99c3b8406f79c030d`.** `make offline-closure` passed from an isolated clean
detached worktree without provider binaries or live/network execution: race/fault/path and boundary
suites, readable-fixture drift, 263 Python tests, 158 UI tests, 7 rendered mock scenarios,
contracts, lint, build, embedded UI comparison and source-repository cleanliness all passed. The
worktree remained free of tracked drift.

What:
- run the combined race, fault-injection, path-attack, incident-fixture and artifact-auditor suites;
- run route/view-model/UI mock/rendered/keyboard/axe and live/product boundary suites;
- run `make contracts`, `make test`, `make lint`, `make build` with enough local disk for reliable
  output and verify tracked embedded UI bundle plus clean source repositories;
- reconcile active plans and trackers only after every preceding slice is merged.

Why: Epic 18 must restart from one reviewed clean commit with no unresolved provider-free blocker.

Acceptance:
- all offline gates pass repeatedly without provider binaries or network access;
- no race/path/fault fixture is quarantined, and no live-specific condition exists in product code;
- the exact clean merge commit is recorded as the new R3 qualification SHA.

### Epic acceptance

- Run/session/history and Git coordination are race-clean and restart-safe.
- Workspace and snapshot paths fail closed across symlinks, traversal and cross-run identities.
- Refresh cannot produce a false no-op or preserve mutable/foreign baseline bytes.
- Validation/recovery is typed, bounded and independent of human error text.
- Product and live harness share only public contracts; neither authors or imports the other's logic.
- ProductShell evidence, routing, responsive layout, keyboard and accessibility gates pass offline.
- Provider-free artifact audit rejects every preserved contamination incident and accepts the clean
  reference corpus.
- Full deterministic DoD passes on an adequately provisioned host before Epic 18 R3 restarts.

## Post-R3 K-roadmap product queue

Status (2026-07-26): **implementation complete in the requested order; R3 evidence remains
pending.** The owner explicitly requested local implementation of
`K2b -> K4 -> K3A -> K3B -> 9D -> cleanup` without waiting for live E2E. This does not waive or
replace the trusted-machine R3 release gate. K5 claim ledger, K6 contradictions and K7 search
projection remain outside this cycle.

### K2b — Advisory Workspace Health completion

Status (2026-07-26): **implementation complete; provider-free DoD evidence recorded in the active
ExecPlan.** Delivery was performed locally without claiming the still-blocked Epic 18 R3 live gate.

What:
- retain the computed read-only `GET /api/workspace/health`; do not persist `reports/health/*` and
  do not change the response shape/version;
- add deterministic checks for broken local Markdown artifact links, missing entity endpoints,
  duplicate aliases, unlinked findings, proposal evidence that does not exist, malformed canonical
  model/proposal/report files and orphan domain/team outputs;
- show current-workspace summary in Knowledge and existing readiness/inspector surfaces only;
  historical Changes must not display current workspace health as historical evidence.

Acceptance:
- clean/warn/fail, path containment and every new issue class have provider-free fixtures with stable
  IDs and deterministic ordering;
- scanning is byte-identical/read-only and remains advisory: it never blocks run, Review or Publish.

### K4 — Citation and claim identity hardening

Status (2026-07-26): **implementation complete; provider-free DoD tracked in the active ExecPlan.**
The existing public schema shapes remain unchanged.

What:
- prove global citation/claim ID uniqueness, document/citation reciprocity, run isolation, concrete
  in-root evidence paths and deterministic claim IDs for identical input;
- reject key Architecture Home/findings/proposals documents that claim citation completeness with
  empty coverage;
- prevent selective-refresh preserved artifacts from referencing a removed citation/document;
- expose low coverage through advisory Workspace Health rather than a new publication blocker.

Acceptance:
- duplicate IDs, unresolved/foreign-run references, broken reciprocity and out-of-root paths remain
  hard failures under the existing public schema shapes;
- if implementation proves a shape change is necessary, stop and create a separate schema-first PR
  with full contract synchronization instead of hiding it in K4.

### K3A — Explicit Ask to Proposal backend mutation

Status (2026-07-26): **implementation complete** with atomic rollback and synchronized additive
schema/API contracts.

What:
- add additive `POST /api/qa/runs/<run_id>/proposal-draft` with required title and
  `expected_answer_digest`, optional slug/operator note, and optional `answer_digest` on succeeded
  QA run reads;
- allow creation only from a succeeded immutable QA run with a valid `qa-answer.json`, matching
  digest and resolvable citations;
- atomically create `proposals/qa-synthesis-<run-id>-<slug>/` containing `proposal.md`, `evidence.md`
  and schema-validated `source-qa-answer.json`; never overwrite an existing package;
- return typed not-found/not-succeeded/unavailable/stale/unresolved/invalid-slug/already-exists
  errors without mutating the QA taskrun or source repositories.

Acceptance:
- success, stale digest, duplicate, path traversal, broken citation, write/rename failure and restart
  fixtures prove atomic rollback and source/canonical QA non-mutation;
- API/spec/appendix/examples/fixtures/validators/ADR rationale are synchronized in the same PR.

### K3B — Ask to Proposal ProductShell flow

Status (2026-07-26): **implementation complete** with focus-managed confirmation, current
Changes→Proposals routing, Git refresh and Return to Ask context.

What:
- show `Create proposal draft` only for a succeeded answer with `answer_digest`;
- use a focus-managed confirmation showing title/path, citations, unresolved assumptions and the
  explicit fact that Ask remains read-only;
- after `201`, navigate through the existing URL codec to current-workspace Changes -> Proposals and
  preserve Return to Ask context;
- keep Ask open on stale/duplicate/citation error and offer reload of the selected answer.

Acceptance:
- focus trap, Escape, return focus, routing/reload and stale-answer tests pass with no critical axe
  violations;
- the new proposal immediately appears in full Git inventory and makes an already-open Git
  confirmation stale; no commit or branch is created implicitly.

### 9D — Q&A compatibility policy through v1

Status (2026-07-26): **implementation/documentation lock complete.** Deterministic response-shape
and repeatability tests protect the compatibility endpoint; removal still requires a separate
approved v1 breaking-change plan.

What:
- keep `POST /api/qa/ask` and `acp qa` deterministic and response-compatible through v1;
- document async start/poll migration examples without adding runtime deprecation headers;
- require a separate approved v1 breaking-change plan before removal.

Acceptance:
- API spec, README, architecture/changelog policy and contract tests all describe the same boundary;
- UI remains on async `/api/qa/runs`, while existing CLI/API consumers are not broken.

## Cleanup follow-up (post-R3, retain decision accepted)

Owner decision (2026-07-22): retain the 90 tracked files under
`fixtures/scenarios/*/golden/readable` as a versioned human-review deterministic export. Machine
fixtures remain the execution source; readable exports remain review/release evidence. Do not
delete, deduplicate or switch them to generate-on-demand in this cycle.

Remaining cleanup tasks:

1) Readable fixture drift protection
- keep a deterministic drift check between scenario generation/machine fixtures and readable export;
- document ownership consistently in `fixtures/README.md`, `docs/BASELINE_POLICY.md` and
  `docs/TESTING_STRATEGY.md`;
- preserve current files and review-friendly Git diffs.

Status (2026-07-26): **complete** via `make verify-readable-fixtures`; all retained readable files
must be a digest-matching subset of the adjacent machine snapshot.

2) Tracker and ExecPlan archive
- archive implementation-complete ExecPlans using the monthly archive convention;
- reconcile stale K-roadmap checkboxes and merged PR/commit references;
- record `docs/BACKLOG.md` as acceptance/reference backlog, with active execution state remaining in
  the stakeholder matrix and `docs/PLANS.md`;
- do not mix archive cleanup with product behavior changes.

Status (2026-07-27): **complete**. Implementation-complete child plans, including `22O`, were moved
to the July archive. Only the program-level Epic 18 R3 evidence work remains active.

Resolved (2026-04-05):
- `slugify` дедупликация между подсистемами выполнена через `internal/slugutil` + regression tests.
- `.codex/model_instructions.md` удалён из tracked surface в cleanup-срезе.

Follow-up note (2026-04-22):
- статус `docs/BACKLOG.md` как active planning surface vs reference/history требует отдельного owner decision; этот cleanup-slice синхронизирует только terminology, не меняя роль документа.

## Epic 23 — Task-first Product UI and Artifact Workbench

Status (2026-08-11): **target design and pre-implementation authority decisions accepted; 23B1
through 23L additive slices are implemented without shell cutover; the remaining UI wave and
Tasks-primary closure are pending.**

Authoritative UX: [`UI_TASK_FIRST_PRODUCT_DESIGN.md`](UI_TASK_FIRST_PRODUCT_DESIGN.md).
Delivery order: [`UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md`](UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md).

### Product outcome

ProvenArch перестаёт показывать pipeline run как основную работу пользователя. Пользователь создаёт
устойчивую Task с goal/scope/runner, видит immutable Attempts, получает outcome-first result,
работает с Markdown/YAML/Mermaid/evidence в content-appropriate workbench и публикует полный
architecture workspace через явное Git confirmation.

### Epic non-goals

- hosted/multi-user collaboration и persisted approval workflow;
- source-repository writes;
- новый provider вне `fake|claude-code|qwen-code|codex-code`;
- visual Mermaid drag editor;
- selected-file Git commit при текущем full-workspace publication contract;
- frontend-only Task façade без schema/API authority;
- hidden compatibility UI ради старых selectors.

### 23A — Task, Attempt and runner admission contracts

**Goal:** дать UI устойчивый пользовательский Task object и immutable execution Attempt вместо
эвристического переименования runs.

Accepted decisions:

- Task is a separate durable aggregate; Attempt maps one-to-one to an exact pipeline run and is
  immutable after admission;
- planned persistence is versioned `reports/taskruns/task-history.json` plus `.last-good`, excluded
  from promoted Architecture/provider/QA context; pre-contract runs remain explicit legacy evidence
  and are never synthesized into Tasks;
- runner selection is per-Attempt, with immutable effective config/sources and one global active plus
  at most one queued Attempt under the shared admission lease;
- authoritative details live in `docs/spec/TASK_SPEC.md` and the 2026-08-11 Task/Attempt ADRs.

What:

- определить schema/spec для Task: ID, title, goal/context, repo/scope, runner preset, lifecycle,
  attempts, created/updated/archive metadata;
- связать Attempt с existing run identity, parent/child retry lineage и immutable effective config;
- определить storage/read APIs, pagination/filtering и restart persistence;
- решить per-Attempt fake/headless admission либо документированный service-session restart flow;
- snapshot-ить provider, model, effort, permissions and per-step overrides в Attempt history;
- определить authoritative task/run/change/publication linkage без ложного `Published` inference;
- выполнить `acp-schema-guardian`: specs, schemas, validators, examples, fixtures, appendix and ADR.

Acceptance:

- Task переживает restart и остаётся той же сущностью после retry/rerun;
- admitted Attempt config не меняется от последующего Settings/workspace/env update;
- concurrent Task start/session switch остаётся serialized admission lease;
- invalid scope/runner/task identity fail-closed до provider execution;
- API and schema round-trip fixtures cover create/list/read/archive and parent-child attempts;
- no Task data is written to analyzed source repositories or promoted Architecture surfaces.

### 23B — Task-first shell and navigation cutover

**Goal:** единственный shell `Tasks / Architecture / Changes`, global Ask and contextual Settings.

Delivery boundary: implement as two reviewable PRs. `23B1` adds typed routes and truthful target
containers without claiming cutover complete. `23B2` runs after Tasks, Architecture and Changes are
ready, makes Tasks primary and removes old routes/components/selectors in the same accepted diff.

What:

- добавить typed routes `/tasks`, `/tasks/new`, Task/Attempt detail and target Architecture/Changes;
- сделать Tasks default destination; удалить Home and Analyze primary destinations;
- сохранить explicit authority, selected identities and viewer mode in URL;
- migrate Back/Forward/reload and invalid/stale identity notices;
- introduce quiet architectural desk tokens, vector icon set and three density modes;
- cut over without hidden legacy shell or duplicate controls.

Acceptance:

- every primary destination has one purpose and one dominant action;
- deep links restore exact Task/Attempt/artifact/source context;
- invalid explicit identity never falls back to another run/current workspace;
- mobile uses bottom nav and safe-area; desktop supports collapsed semantic nav;
- old Home/Analyze route components/selectors are removed in the accepted cutover diff;
- route/component/rendered tests cover 1440/1024/390 widths and keyboard navigation.

Implementation status (2026-08-11): `23B1` is implemented on the additive UI route surface. The
codec now preserves `/tasks`, `/tasks/new`, exact Task detail and Task/Attempt identities, plus
optional Task context on Architecture/Changes. The target container is explicit and read-only:
it does not query or mutate Task data and never substitutes a legacy/latest run. `23B2`, the
primary-navigation cutover and removal of Home/Analyze routes, remain pending until the vertical
Task, Architecture and Changes slices are ready.

### 23C — New Task composer and inline runner readiness

**Goal:** пользователь формулирует анализ и выбирает runner без перехода в Settings.

What:

- goal-first composer with optional context, repositories and include/exclude scope summary;
- runner picker: Deterministic demo, Claude Code, Qwen Code, Codex, Advanced mix;
- show effective model/effort, demo/live identity, readiness and last check in picker;
- preserve draft across in-app navigation, runner recovery and temporary offline state;
- block start only on real contract/readiness blockers with adjacent reason;
- start protection against double click and active/pending admission conflict.

Acceptance:

- first-time user can explain read/write boundary and effective runner before Start Task;
- ordinary provider choice needs no Settings navigation;
- demo copy never implies live architecture analysis;
- provider unavailable flow offers actionable recovery and demo fallback without losing draft;
- create/start/error/queued/offline states pass component and mock E2E tests;
- scope is submitted exactly as displayed and never inferred from UI-only filters.

Implementation status (2026-08-11): `23C` is implemented as an additive New Task composer. It
submits the displayed repository scope to the authoritative Task API, snapshots the selected
runtime mode/provider in the desired runner preset, blocks mode/provider/scope mismatches with an
adjacent reason, and routes to the exact created Task identity. `23D` now adds the additive
authoritative Inbox/detail/Attempt read surfaces, URL-restorable filters, derived lifecycle groups,
keyboard-safe rows and archive/unarchive controls; Attempt admission remains server-owned and later
outcome/workbench slices remain pending.

### 23D — Task Inbox, filters and Attempt history

**Goal:** сделать ежедневную работу с несколькими задачами быстрой и сканируемой.

What:

- groups `Needs attention`, `Running`, `Ready`, `Completed`, `Archived`;
- search and URL-restorable filters by lifecycle, runner, repo and time;
- compact rows show user goal, current state, runner and meaningful last activity;
- selected Task opens detail without losing list position/filter;
- Attempt history shows terminal status, runner snapshot, duration, parent/child lineage;
- archive/unarchive uses confirmation/undo and never deletes run evidence.

Acceptance:

- grouping/counts derive from authoritative Task state and reconcile with detail;
- large lists remain keyboard navigable and virtualizable without focus loss;
- empty/filtered-empty/loading/error/offline states offer one clear recovery;
- no `current step` appears on terminal Task rows;
- list/detail/history component tests and 1000-row performance fixture pass.

Implementation status (2026-08-11): `23D` loads only public Task/Attempt endpoints, derives Inbox
groups from Task lifecycle and linked Attempt summaries, keeps exact identities through detail/history
routes, restores filters through the URL, and exposes explicit loading/error/empty/recovery states.
It does not synthesize Tasks from legacy runs or infer a result from recency.

### 23E — Outcome-first Task detail and semantic result

**Goal:** после terminal Attempt сначала показать полученное знание и решения, не telemetry.

What:

- terminal success header: outcome, validated scope, semantic counts and next action;
- semantic rows for added/changed/removed entities/edges/findings/gaps;
- decisions/questions separated from generated files;
- current Architecture availability shown independently from Attempt lifecycle;
- retry/rerun moves to secondary menu; plan preview explains downstream closure;
- failed/canceled outcomes explain retained evidence and last-good Architecture.

Acceptance:

- succeeded Attempt renders no active/current-step language;
- failed Attempt never hides or invalidates last-good Architecture;
- semantic summary binds to exact run snapshot and does not compare mutable bytes implicitly;
- missing semantic comparison becomes explicit partial state, not fabricated zero delta;
- success/failure/canceled/recovered/demo/live fixtures pass.

Implementation status (2026-08-11): the additive Task detail now binds terminal outcome rendering to
the exact Attempt run's public review-summary endpoint. Success shows the snapshot identity,
promotion/current-Architecture availability, semantic entity/edge/finding/question/gap counts and
recommended next action; failed/canceled Attempts retain explicit evidence and never hide last-good
Architecture. Missing semantic comparison is rendered as partial/unavailable rather than zero.

### 23F — Focused Pipeline Studio and recovery

**Goal:** дать операторам глубокую диагностику без превращения всего продукта в dashboard.

What:

- open from a specific Attempt only; no global nav item;
- pipeline track shows canonical steps and durable progress only;
- current step shows scope table, artifacts and last useful progress;
- one selected blocker panel explains cause, retained data and recommended action;
- raw logs, JSON, permissions and telemetry live in Diagnostics disclosure;
- `Retry failed scope`, `Change runner for next attempt`, cooperative `Stop task` flows.

Acceptance:

- percentage is never derived from stdout/heartbeat;
- retry creates child Attempt and never mutates terminal parent;
- changing runner cannot alter active Attempt;
- failed step/log line deep links preserve Attempt identity;
- active/retrying/stalled/permission-required/failed/canceled rendered scenarios pass;
- raw output cannot expand the document into an unbounded scrolling wall.

Implementation status (2026-08-11): the additive Attempt route now exposes an exact
`/tasks/<task_id>/attempts/<attempt_id>/studio` Pipeline Studio. It reads the public Attempt and
run review-summary identities, renders canonical structured steps, shows a bounded selected blocker
and diagnostics disclosure, and explicitly avoids deriving progress percentages from provider output
or selecting a latest/global run. Retry and runner mutation remain later admission-owned work.

### 23G — Architecture Map as current knowledge home

**Goal:** открывать продукт на понятной текущей архитектуре, а не на runtime dashboard.

What:

- map/list views from validator-approved entities/edges only;
- type-specific node shapes/icons, ownership, coverage and confidence states;
- selection inspector links document, model entity, findings and evidence;
- update banner routes to exact task/run Changes review;
- search/filter by name, type, domain, owner and evidence status;
- mandatory list/table equivalent and bounded layout performance.

Acceptance:

- filenames/layout positions are never semantic facts;
- partial/missing evidence is at least as visible as validated topology;
- map remains available after active/failed Attempts;
- empty map explains required analysis/artifacts and one next action;
- keyboard selection, zoom, list parity and large-graph fixtures pass.

Implementation status (2026-08-11): the additive Architecture surface now accepts an exact Task
context, labels the current promoted workspace authority, exposes a return path to that Task and
states that it is read-only without latest-run inference. Remaining map/workbench parity and
responsive closure continue in W23H–W23O.

Implementation status (2026-08-11, W23H): Architecture Documents now provide a bounded Markdown
reader/editor. Only backend-allowlisted `charter/*` and `skills/*` Markdown can be edited and saved;
promoted reports remain explicitly read-only evidence, preserving snapshot authority and lossless
text until save.

### 23H — Markdown Document Workbench

**Goal:** сделать Architecture Home и authored docs первоклассным читаемым продуктом.

What:

- semantic document tree plus secondary folder/path explorer;
- Rendered default, document outline, search, safe relative links and citations;
- Source mode with line numbers and bounded large-file fallback;
- explicit Edit for current workspace only; run/QA snapshots always read-only;
- unsaved draft, validation, atomic Save and Git dirty feedback;
- preserve scroll/focus when opening/closing evidence drawer.

Acceptance:

- Markdown XSS/link traversal/cross-run protections stay intact;
- edit/save cannot target source repo, taskrun snapshot or unsupported file;
- broken links/render failure/oversized/empty/offline states are actionable;
- save failure preserves draft and original workspace bytes;
- headings/tables/code/long paths/citations fixtures and keyboard E2E pass.

### 23I — YAML/JSON Model and Schema Workbench

**Goal:** безопасно работать с entity-per-file model, charter and schema-governed data.

What:

- intentional structured inspectors for entities, edges, charter, workspace and indexes;
- schema name/version, field descriptions, evidence and validation summary;
- Source mode in Advanced with field/line-linked diagnostics;
- lossless patch editor requirement for structured mutations;
- structural key-path diff plus raw diff fallback;
- batch navigation between related entity/edge files without file identity drift.

Acceptance:

- no structured save until comments, unknown keys, ordering and multiline scalars are preserved;
- schema + semantic validation occurs before atomic write;
- invalid YAML/JSON never destroys last valid rendered state or draft;
- entity IDs/relations remain exact and long bounded filenames display full logical ID;
- fixtures cover comments, aliases, unknown keys, invalid types, large files and conflicts.

Implementation status (2026-08-11, W23I): the Model view now exposes a structured read-only
entity/edge inspector with canonical logical identity, `architecture.entity|edge` schema/version,
promoted validation status, path-linked issues and an Advanced line-numbered source view. It
explicitly withholds structured Save until comments, unknown keys, ordering and multiline scalars
have a proven lossless round trip; malformed or unavailable source keeps the last valid promoted
structure visible.

### 23J — Mermaid and shared Evidence Studio

**Goal:** объединить diagram viewing и claim-to-source provenance без ложной visual authority.

What:

- Rendered/Source modes, zoom/fit and accessible relation list;
- edit source with bounded live preview for current workspace only;
- node/entity/document/evidence navigation using validated IDs;
- evidence drawer: authority, claim, repo/path/ref, confidence and coverage issue;
- Markdown/YAML/Mermaid citations use one shared identity/navigation model;
- Mermaid source diff only until deterministic visual diff contract exists.

Implementation status (2026-08-11, W23J): the Diagrams surface now exposes Rendered/Raw source
tabs, actionable render-error fallback and an accessible validated relation list. Promoted Mermaid
source remains read-only; relation navigation uses canonical model edge IDs and explicitly does not
derive semantics from layout or arrow placement.

Acceptance:

- broken/oversized diagram exposes actionable source fallback;
- diagram layout is never persisted or described as semantic relation;
- evidence authority never silently changes across navigation;
- drawer focus/return, cross-run containment and citation reciprocity tests pass;
- every map/diagram relation has accessible list/table equivalent.

### 23K — Findings, questions and proposal decision workflow

**Goal:** превратить findings/gaps/questions в понятную очередь решений, не approval theater.

What:

- separate findings, questions, gaps and proposals with severity/priority/evidence count;
- detail shows observation, why it matters, evidence and suggested direction;
- filters by severity/domain/owner/status with URL restoration;
- explicit proposal-draft action where existing contract permits it;
- no `Approved` status until persisted human decision contract exists;
- unresolved items remain visible in Task outcome and publication risk summary.

Acceptance:

- counters reconcile across Task, Architecture and Changes;
- missing evidence is not shown as low-confidence fact;
- proposal links exact findings/evidence or shows disconnected state;
- empty/partial/error/filtered states and keyboard list/detail flow pass;
- no runtime logs required to understand any operator-facing finding.

Implementation status (2026-08-11, W23K): the Findings view now separates findings, questions and
coverage gaps with search/severity filtering, bounded selected-item detail, linked-reference counts
and explicit unresolved status. It does not invent `Approved` or mutate proposals before a
persisted human decision contract exists.

Implementation status (2026-08-11, W23L): Changes now carries the exact Task identity through
snapshot/current routes, provides a return path to that Task and states that selected run/Attempt
identity is authoritative. Current workspace evidence remains read-only and separate from
historical snapshot publication; no latest-run fallback is introduced.

### 23L — Changes truth and full-workspace Publish

**Goal:** честно разделить automatic promotion, semantic review and Git publication.

What:

- copy states that validated knowledge is already Current workspace;
- summary/evidence/files/publish are materially distinct routes;
- semantic delta first; Git file diff second; exact baselines named;
- authoritative full workspace inventory with new/modified/deleted/untracked/renamed/copied;
- confirmation includes branch, HEAD, inventory fingerprint, demo/live and open risks;
- commit/proposal branch results, stale confirmation and retry behavior.

Acceptance:

- no copy asks whether current snapshot should be replaced;
- no selected preview implies selected-file commit scope;
- active/pending work and stale HEAD/inventory block mutation authoritatively;
- successful commit refreshes state and cannot be double-submitted;
- semantic and Git counters are labelled and never presented as the same metric;
- full success/stale/conflict/blocked/unknown/demo fixtures pass.

### 23M — Global Ask and Runner Settings

**Goal:** сохранить Ask быстрым, а advanced runner configuration — доступным, но не обязательным.

What:

- global `Ask this architecture…` over current workspace with explicit read-only authority;
- history, confidence, citations, unresolved and retry/recovery states;
- Runner Settings presets with provider/model/effort/readiness and per-step overrides;
- effective vs desired/source values and last check;
- explicit proposal draft mutation remains separate confirmation;
- remove duplicate runtime controls from ordinary Task/Architecture screens.

Acceptance:

- Ask never changes workspace without explicit proposal action;
- provider outage explains whether it blocks Ask/new Attempt but not existing Architecture review;
- preset edits cannot mutate active Attempt history;
- unsupported effort/provider values fail inline before save;
- modal/sheet focus trap, return context and mobile fullscreen tests pass.

Implementation status (2026-08-11, W23M): the global Ask surface now repeats its current-workspace
read-only authority inside the panel, records that Q&A cannot mutate canonical architecture or an
admitted Attempt, and explains that provider outage blocks only new Ask/Attempt admission while
existing evidence remains reviewable. Runner Settings now labels editable fields as desired draft
configuration, keeps effective/source resolution and immutable admitted Attempt history explicit,
and routes the latest readiness check to Setup → Runner. Proposal creation remains a separate,
digest-bound confirmation action.

### 23N — Responsive, accessibility and complete state coverage

**Goal:** закрыть quality bar до release, не после visual polish.

What:

- 1440/1280/1024/768/390 layouts; bottom nav and safe areas;
- Task list/detail route split, overlay inspectors and fullscreen sheets;
- 44px touch targets, visible focus, one `h1`, logical landmarks;
- keyboard contracts for lists, tabs, dialogs, comboboxes, editors and map alternative;
- reduced motion, 200% zoom, long paths and local source/diff scrolling;
- fixture matrix for every state from target design.

Acceptance:

- no global horizontal overflow, hidden focus targets or clipped primary actions;
- critical axe violations = 0 in all deterministic rendered scenarios;
- every primary job completes keyboard-only;
- status uses text/icon/shape and live regions do not announce duplicates;
- mobile title/authority/state/action fit before the first long content section.

### 23O — Deterministic UI closure, old-shell removal and docs sync

**Goal:** не объявлять новый UI законченным, пока старый shell и противоречивые docs реально живы.

What:

- run all target mock E2E scenarios and component/contract suites;
- remove obsolete routes/components/CSS/testids and hidden compatibility surfaces;
- ensure current-behavior screenshots are ephemeral, target PNGs are the only committed UI refs;
- synchronize README, ARCHITECTURE, STAKEHOLDER, TESTING_STRATEGY and runbooks;
- verify embedded UI bundle parity and clean source repositories;
- full DoD before trusted-machine live gate.

Acceptance:

- `make contracts`, `make test`, `make lint`, `make build` pass;
- target rendered scenarios pass at desktop/tablet/mobile with no skips;
- source scans find no Home/Analyze legacy navigation or obsolete UI asset paths;
- current binary behavior and planned/implemented docs status are consistent;
- release live gate runs only through `acp-e2e-live-gate` after deterministic closure.

### Epic acceptance

- Task is the stable user object; Attempt is immutable execution detail.
- runner choice is available at Task creation with truthful readiness.
- terminal outcomes are result-first and Pipeline Studio is contextual.
- Architecture is the independent current knowledge product.
- Markdown/YAML/JSON/Mermaid have content-appropriate, authority-safe workbenches.
- Changes explains automatic promotion; Publish confirms full workspace Git mutation.
- responsive, keyboard, accessibility and state fixtures pass deterministically.
- no obsolete UI screenshots, conflicting target docs or hidden legacy shell remains.

## Epic 24 — Weak-model runtime validation and promotion hardening

Status (2026-08-11): **authority decision accepted; W24A–W24F, W24H and W24I implemented; metric-gated W24G remains pending.**
This epic extends the completed `22G` typed recovery state machine and `22H`
provider-free auditor. It does not reopen or replace those slices.

### Goal

Make weak but supported headless models fail predictably at the earliest authoritative boundary,
without accepting plausible-but-unsupported architecture and without forcing a provider retry for
mechanical metadata that ACP can own deterministically.

The runtime must accept sparse truthful output with explicit coverage gaps, but reject foreign-run
artifacts, contradictory validator verdicts, unverifiable evidence, broken semantic graphs and any
selected-run snapshot that fails provider-free integrity checks. Technical `PASS | FAIL` authority
must move toward the orchestrator; provider-authored findings and questions remain advisory semantic
input and never bypass deterministic validation.

### Confirmed gaps on current `main`

- validator artifact admission parses `validator-verdict.json`, but does not bind its `run_id` or
  `checked_paths` to the current runtime task;
- the verdict contract accepts `verdict=PASS` together with one or more `severity=error` issues;
- semantic/citation evidence checks concrete repository file existence and containment, but do not
  verify optional line ranges, excerpts or excerpt hashes against the referenced bytes;
- several semantic `evidence`, `provenance`, entity, edge and finding schema objects allow unknown
  properties, so an unsupported model alias can pass schema validation and then disappear during Go
  decoding;
- the provider-free auditor is exposed as a read-only selected-run/promoted-current API, but is not
  yet the mandatory final gate immediately before canonical promotion;
- provider-authored verdict findings/questions are merged before every deterministic selected-run
  integrity decision has completed;
- current recovery is typed and bounded per transition, but the total number of possible
  provider-authored repair transitions is still large and step/provider asymmetric.

### Epic boundaries and non-goals

- No hosted mode, source-repository writes, security/compliance enforcement or new provider.
- No default provider/model/reasoning change; model compatibility is measured separately.
- No deterministic invention of architecture prose, entities, relations, findings or evidence.
- No weakening of schema, evidence, write-set, containment, final-index or promotion checks to make
  a weak model appear successful.
- No live network dependency in required CI; `fake`, synthetic fixtures and recorded incident-shaped
  artifacts remain the required baseline.
- Historical taskruns remain readable. A strict writer/runtime contract may evolve before v1, but
  any schema-version change requires a dual-read compatibility decision and full
  `acp-schema-guardian` synchronization.
- Epic 23 may later render the typed diagnostics, but Epic 24 backend correctness does not depend on
  the Task-first UI migration.
- The canonical live matrices and curated repository list are not changed by this epic.

### Delivery order

`24A -> 24B -> 24C -> 24D -> 24E -> 24F -> 24H -> 24I`

`24G` is conditional and starts after `24F` only when the recorded first-pass/repair metrics meet its
entry condition. `24H` records a shared hard budget at the provider process-start seam and `24I`
closes the incident corpus and conformance evidence. Each slice is a separate reviewable
implementation unit with its own ExecPlan, focused tests and documentation synchronization.

### Cross-slice invariants

- equivalent invalid output yields the same ordered typed issue set across
  `claude-code | qwen-code | codex-code`;
- runtime validation is read-only; repair is a separate explicit transition with before/after bytes;
- provider-authored output never mutates stable `reports/*`, `model/*` or `proposals/*` directly;
- no merge into execution semantic state occurs before the artifact passes its task-aware boundary;
- no canonical promotion starts while any deterministic issue or selected-run audit issue has
  `severity=error`;
- missing support becomes a coverage gap/question, not an ACP-authored fact and not an automatic
  technical failure;
- warnings and advisory findings remain visible but cannot override a deterministic technical gate.

### 24A — Task-bound validator identity and checked-path admission

**Goal:** reject a structurally valid validator artifact that belongs to another run, checks the
wrong snapshot or points outside the validator's assigned read surface.

What:

- replace parse-only validator admission with task-aware validation in the shared provider runtime;
- require exact normalized `verdict.run_id == task.run_id` before apply or semantic merge;
- normalize and deduplicate `checked_paths`, rejecting empty entries and ambiguous aliases;
- require every checked path to resolve to a regular file inside the exact current
  `reports/taskruns/<run_id>/staging/final/` snapshot;
- require the exact current `final-run-index.json` and `citation-index.json` paths, not only a parent
  directory, suffix match, validator write root, canonical workspace path or another run's index;
- reject absolute paths, traversal, symlink escape, missing files, directories and foreign-run paths;
- preserve the validator write-set rule: provider writes only `validator-verdict.json` inside its
  task-local write root and treats checked paths as read-only inputs;
- emit stable issue codes with artifact/JSON path rather than routing on error text.

Expected modules:

- `internal/runtime/providercommon/artifacts.go` and focused tests;
- shared validation issue mapping used by `providercommon` recovery;
- validator task builders/fake runtime fixtures where exact checked paths are authored;
- `docs/spec/PIPELINE_SPEC.md` and `docs/TESTING_STRATEGY.md`.

Tests:

- wrong/empty/case-changed `run_id`;
- checked path under another run, validator root, canonical path and source repository;
- traversal, absolute path, symlink escape, missing target and directory target;
- missing one required index, duplicate path and parent-directory-only check;
- clean current-run validator artifact for local `path` and resolved `git_url` repositories;
- provider adapter metamorphic fixtures prove identical typed admission failure.

Acceptance:

- every invalid variant terminates before findings/questions merge and before staged reassembly;
- no rejected validator task changes canonical workspace bytes;
- clean fake/current-run artifacts continue to pass without an additional provider call;
- the public verdict JSON shape remains unchanged in this slice.

### 24B — Verdict consistency and technical-field ownership

**Goal:** make the existing verdict shape internally coherent before changing the larger verdict
authority model.

What:

- add the invariant `PASS` cannot contain any `issues[].severity=error`;
- require technical `FAIL` to contain at least one error issue; unsupported provider concerns belong
  in advisory findings/questions rather than an empty opaque failure;
- reject provider-authored non-empty `fixed_paths`; only the orchestrator may add a path after a
  successful deterministic repair and full revalidation;
- validate issue `path`, `document_id` and `citation_id` against the current selected-run inventory
  when present;
- deterministically collapse exact duplicate issues by `(code,path,document_id,citation_id)`;
  conflicting duplicates fail consistency validation, and orchestrator ordering is stable;
- persist before/after digests and exact repair code for every orchestrator-added `fixed_path`;
- keep provider issues/findings/questions isolated until the full verdict consistency check passes.

Expected modules:

- `internal/contracts/docflow.go` and schema fixtures;
- `internal/runtime/providercommon/artifacts.go`;
- `internal/orchestrator/docflow_repair.go` and `runtime_task_apply.go`;
- `internal/runtime/steppolicy` and `internal/runtime/fakeruntime` verdict builders.

Tests:

- `PASS + error`, `PASS + warning`, empty `FAIL`, `FAIL + error`;
- provider-authored `fixed_paths`, orchestrator-authored repaired paths and stale fixed path;
- issue refs to absent document/citation/foreign path;
- duplicate/reordered issue sets and JSON round trip;
- retry/replay preserves effective repaired verdict identity without accepting parent-run paths.

Acceptance:

- no contradictory verdict reaches `applyValidatorRuntimeExecution`;
- deterministic repair is the only source of non-empty `fixed_paths`;
- the same inconsistency maps to one stable code across normal, focused repair and replay paths;
- historical valid v1 verdict fixtures remain readable.

### 24C — Evidence locator and claim-integrity validation

**Goal:** prove that a syntactically valid evidence locator actually identifies the claimed source
bytes rather than merely naming an existing repository file.

What:

- introduce one reusable evidence validator shared by collect artifact admission, validator
  findings, staged validation and provider-free audit;
- resolve logical repo names and generated checkout aliases to the exact current task repository
  roots without accepting an ambiguous basename match;
- keep current relative-path, regular-file, containment and symlink rules;
- when `lines` is present, require `start <= end`, both values within file bounds and a bounded read;
- when `excerpt` is present, compare it with the documented normalized line-range bytes;
- when `excerpt_hash` is present, define one SHA-256 normalization algorithm and verify the digest;
- reject excerpt/hash fields that cannot be tied to an explicit line range, unless the contract
  deliberately defines and tests a whole-file mode;
- apply identical rules to `citations[]` and
  `entities|edges|findings[].provenance.evidence[]`;
- preserve inference/assertion only when its evidence policy is explicit; unsupported observations
  are removed by the provider or surfaced as questions/coverage gaps, never silently promoted;
- bound file size, line count, excerpt length and issue count so malicious or accidental large files
  cannot make validation unbounded.

Accepted normalization decision: line ranges are 1-based inclusive UTF-8 text; CRLF normalizes to
LF, while whitespace and Unicode content are not trimmed or normalized. Excerpt compares to those
exact normalized line bytes: selected logical line contents joined by LF with no synthetic trailing
LF. `excerpt_hash` is SHA-256 of the same bytes. Invalid UTF-8, excerpt/hash without an explicit line
range and whole-file excerpt mode are not supported in the MVP.

Expected modules:

- new shared evidence-quality module under `internal/artifactquality` or another cycle-safe package;
- collect validation in `internal/artifactquality/canonicalize.go`;
- staged citation validation in `internal/orchestrator/docflow.go`;
- validator finding admission and `internal/artifactaudit/audit.go`;
- schemas/spec/examples/fixtures if evidence normalization semantics change.

Tests:

- missing/ambiguous repo, missing file, directory, traversal and symlink escape;
- reversed, zero, negative and out-of-range line ranges;
- valid LF and CRLF excerpts; whitespace/Unicode normalization decision fixtures;
- wrong excerpt, wrong hash, oversized file/range and unreadable file;
- the same valid evidence reused by collect, validator and auditor;
- sparse repository output represented as a gap passes without fabricated evidence.

Acceptance:

- all consumers return the same evidence issue codes and normalized locator identity;
- no observation with an invalid locator/excerpt/hash enters the promoted semantic snapshot;
- validation remains read-only and bounded;
- required CI uses temporary repository trees and deterministic bytes only.

### 24D — Strict semantic envelope and cross-shard graph integrity

**Goal:** prevent weak-model aliases, silently dropped fields and dangling cross-shard relations from
surviving aggregation.

What:

- inventory currently allowed extension fields in semantic evidence, provenance, entity, edge and
  finding objects and classify each as product data, intentional extension or unsupported drift;
- stop silently dropping unsupported fields: reject them at the authoritative contract boundary or
  preserve them losslessly under an explicitly documented extension object;
- if tightening `additionalProperties` is required, perform an explicit schema/version decision with
  dual-read behavior for historical taskruns and full `acp-schema-guardian` synchronization;
- require unique normalized entity, edge and finding IDs after all shards are aggregated;
- detect normalization collisions separately from exact duplicates; never merge distinct semantic
  objects only because weak-model spelling normalizes to the same token;
- require each edge `from` and `to` to resolve to an entity in the aggregated selected-run graph;
- validate aliases and `related_ids` against their documented namespace, while allowing explicitly
  typed external/unresolved references only if the contract defines them;
- require merged provenance/evidence to remain attributable to the winning object and record any
  deterministic alias/remap decision;
- reject provider/runtime/taskrun paths embedded as semantic evidence or canonical identifiers.

Accepted compatibility policy: new writes never silently discard unsupported properties; intentional
extensions require a typed namespace. First inventory current v1 fixtures. If strict writer rules
would invalidate supported historical v1 bytes, introduce a v2 writer with explicit v1 read-only
compatibility; never rewrite historical taskruns automatically.

Expected modules:

- `schemas/shard-pack-manifest.schema.json` and `schemas/validator-verdict.schema.json` when needed;
- `internal/contracts`, `internal/artifactquality` and orchestrator semantic assembly;
- model materialization/resolver fixtures and `docs/APPENDIX_SCHEMAS.md`;
- ADR for extension and collision policy if the public contract changes.

Tests:

- unknown aliases at every semantic nesting level;
- exact duplicates, order-only duplicates and normalized-ID collisions;
- edge to missing, foreign-run and ambiguous entity;
- cross-shard valid edge, alias remap and multi-repo same-name entities;
- replay/selective-refresh preserved shards mixed with new shards;
- lossless historical v1 read when a new strict writer version is introduced.

Acceptance:

- no unsupported semantic field is silently discarded;
- every promoted edge resolves deterministically in the selected-run graph;
- collision behavior is stable, documented and protected by golden fixtures;
- schema, parser, validator, examples, appendix and ADR remain synchronized.

### 24E — Mandatory provider-free pre-promotion audit gate

**Goal:** reuse the completed `22H` auditor as the final read-only technical gate before any
canonical mutation.

What:

- invoke selected-run audit after final-index/citation reconciliation and deterministic validator
  repairs, but before the first promotion write;
- pass resolved repository roots into the audit so local `path` and managed `git_url` checkouts use
  the same containment/evidence authority;
- fail promotion on any audit `severity=error`; warnings remain visible and non-blocking;
- keep the HTTP audit endpoint read-only and compatible, while sharing the same scanner/options with
  the promotion path;
- record the bounded audit summary and issue codes in run diagnostics without copying raw provider
  logs or writing an additional model-authored artifact;
- prove that failed audit leaves the previous canonical generation and Git state byte-identical;
- avoid circular authority: selected-run audit consumes an internal orchestrator technical candidate
  plus exact indexes after deterministic validation/repair; it never consumes the not-yet-final
  effective verdict and never repairs or rewrites inputs;

Expected modules:

- `internal/artifactaudit/audit.go` and fixtures;
- orchestrator finalization/promotion transaction;
- resolved repository source plumbing;
- run diagnostics/quality summary and pipeline specification.

Tests:

- foreign validator/index identity, staged path escape and missing indexed file;
- asymmetric document/citation refs and promoted digest mismatch;
- invalid evidence locator, execution/scaffold contamination and broken graph signal;
- local path and `git_url` selected-run audit;
- injected promotion write failure and audit failure preserve prior canonical bytes;
- repeated scans are byte-identical and create no files.

Acceptance:

- zero canonical writes occur after any selected-run audit error;
- clean fake and deterministic incident-clean fixtures promote successfully;
- API/on-demand and promotion-gate scans use the same issue codes;
- auditor remains bounded, redacted, deterministic and provider-free.

### 24F — Orchestrator-owned effective technical verdict

**Goal:** ensure that a weak provider cannot green a broken snapshot or fail a clean snapshot through
an unsupported technical opinion.

What:

- preserve provider-authored `validator-verdict.json` as immutable draft evidence and persist a
  separately versioned orchestrator-owned effective technical verdict;
- compute the effective technical verdict from the ordered deterministic issue set after assembly,
  repair and the mandatory selected-run audit: any error means `FAIL`, no errors means `PASS`;
- make `checked_paths`, `fixed_paths`, technical issue codes and final `PASS | FAIL`
  orchestrator-owned;
- admit provider findings/questions only as advisory semantic candidates after evidence and graph
  validation;
- preserve a provider issue as an advisory warning when it does not match a deterministic issue;
- define a stable matching key for provider/deterministic issue correlation without comparing prose;
- remove or collapse special owner-gap/evidence-advisory verdict reconciliation branches once the
  common effective-verdict rule covers them;
- use the accepted versioned draft/effective split from
  `ADR-20260811-validation-audit-effective-verdict-authority.md`; historical provider verdicts
  remain readable and are never rewritten or treated as inferred effective PASS;
- keep terminal technical failure distinct from `runtime_contract_failed` and
  `runner_unavailable`.

Expected modules:

- `internal/orchestrator/runtime_task_apply.go`, staged validation and repair code;
- contracts/schema/ADR if draft/effective representation changes;
- provider step policy and fake runtime artifact generation;
- run history/quality diagnostics and retry-input rebinding.

Tests:

- provider `PASS` with deterministic graph/evidence/index errors;
- provider `FAIL` with a clean deterministic snapshot;
- matching and non-matching provider advisory issues;
- owner-only/evidence-gap scenarios retain findings/questions without false technical failure;
- replay, child retry and selective refresh preserve exact effective authority;
- fake runtime remains deterministic and provider-free.

Acceptance:

- provider-authored `PASS | FAIL` cannot override the deterministic technical result;
- advisory semantic content remains visible only after its own validation;
- one common verdict rule replaces provider/model-specific reconciliation behavior;
- historical persisted verdicts have a documented read/interpretation policy.

### 24G — Conditional reduction of model-authored mechanical envelope

**Entry condition:** start only after `24A`–`24F` metrics show first-pass success below 95%, more than
10% of otherwise valid tasks entering provider repair, or p95 provider invocations above two because
of identity/path/link/shape errors. Record the measurement before choosing a contract design.

**Goal:** remove mechanically derivable fields from the weak model's responsibility without letting
ACP synthesize semantic meaning.

What:

- move `run_id`, `step_id`, `shard_id`, `domain_id`, `artifact_root`, assigned repo/scope identity,
  output mappings, reciprocal index bindings, checked paths and technical verdict to orchestrator
  ownership where derivable;
- keep provider ownership of authored Markdown, candidate semantic objects, evidence candidates,
  questions and coverage gaps;
- compare two explicit designs in an ADR: orchestrator base envelope plus bounded semantic patch, or
  an internal provider-draft format compiled into the public artifact contract;
- forbid generic map merge and arbitrary JSON Patch paths; whitelist model-authored semantic fields
  and validate the compiled result through the full existing contract;
- retain provider-authored bytes and compiler provenance so the final artifact is explainable;
- introduce schema v2 only when necessary, with v1/v2 reader compatibility and v2 writer fixtures;
- update prompts to show one minimal semantic payload rather than requiring the provider to repeat
  long runtime paths and identity metadata.

Tests:

- weak-model payload omits every orchestrator-owned field yet compiles deterministically;
- provider cannot override task identity, output path, checked path or verdict;
- forbidden patch paths, unknown fields and conflicting IDs fail closed;
- compiled artifact round trip, replay and historical v1 read;
- no generated semantic entities/findings appear when provider returns only gaps/questions.

Acceptance:

- measured mechanical contract failures decrease without relaxing semantic/evidence checks;
- ACP-authored fields are exclusively factual task metadata or deterministic bindings;
- provider-authored semantic bytes and evidence remain attributable;
- the entry metric, ADR and before/after conformance result are retained with the slice.

### 24H — Global bounded recovery budget and provider parity

**Goal:** make failure cost and terminal behavior predictable after validation has become
authoritative.

Terminology decision: the hard maximum applies to one runtime provider task envelope
(step/shard/target execution unit), not the durable product Task and not the whole multi-step
Attempt. Attempt/run diagnostics aggregate counters across those runtime units; a new product
Attempt receives new runtime-unit budgets.

What:

- add a global per-task provider invocation budget shared by all recovery transitions;
- target a default hard maximum of three invocations: normal attempt, one focused
  contract/semantic repair and one transport retry only for silent/unavailable execution;
- count every provider process start against the budget, including chained specialized cleanup;
- keep deterministic normalization/repair outside the provider count but log its transition and
  before/after digest;
- route by typed issue class and target artifact, never provider stderr wording;
- collapse specialized shape/marker/index repair branches that are made redundant by `24A`–`24G`;
- keep provider adapters limited to invocation, stream parsing, activity policy and unavailable
  classification; shared recovery owns transition semantics;
- require fresh target mutation for repair success and reject unchanged, stale, wrong-target or
  out-of-write-set output;
- persist attempts used/remaining, selected transition and terminal exhaustion reason.

Implementation status (2026-08-11): the shared provider process-start seam now enforces the default
three-start budget per runtime execution unit, including normal, focused-repair and transport-retry
transitions. Diagnostics and `reports/taskruns/<run_id>-quality.json` persist used/remaining counts,
the last transition and an explicit exhaustion reason. The conformance corpus and p95 measurement
are recorded by `24I`.

Tests:

- normal success, focused success and transport retry success;
- mixed issue classes cannot enter a single-class cleanup;
- silent, printed-command, no-op, stale bytes and wrong-target mutation;
- issue ordering/message paraphrase metamorphic cases;
- equivalent Claude/Qwen/Codex fixtures consume the same budget and reach the same terminal code;
- cancellation and controlled valid-artifact stop do not start an extra repair process.

Acceptance:

- hard provider invocation budget cannot be exceeded by chained transitions;
- p95 provider invocations is at most two on the deterministic conformance corpus;
- repeated/no-op repair is terminal with complete recovery evidence;
- no provider-specific branch changes the semantic acceptance result.

### 24I — Weak-model conformance corpus, diagnostics and closure

**Goal:** make the hardening measurable and prevent future prompts/models from reopening the same
false-accept and repair-loop classes.

What:

- build a provider-free incident-shaped corpus covering foreign identity, schema drift, missing
  evidence, invalid ranges/hashes, graph collisions, contradictory verdicts, stale repairs and audit
  failures;
- use table-driven temporary repository/workspace trees for invalid-run cases; add persistent
  fixtures/goldens only where byte-stable public outputs are the product contract;
- run the same semantic payload through Claude/Qwen/Codex adapter envelopes without invoking live
  binaries and compare typed issues, recovery transitions and terminal status;
- add diagnostic counters for first-pass validity, issue class, repair count, provider invocations,
  effective verdict source and promotion audit result;
- expose bounded issue code/stage/path and attempts used/remaining through existing diagnostics/run
  surfaces; raw provider text remains secondary disclosure;
- optionally add an explicitly invoked provider/model artifact canary to `doctor` or diagnostics,
  keyed by provider/model/CLI/config fingerprint; it is never required CI and never changes model
  defaults automatically;
- synchronize README/ARCHITECTURE/PIPELINE_SPEC/TESTING_STRATEGY/runbooks after implemented behavior,
  then run full deterministic DoD before any trusted-machine live diagnostic.

Implementation status (2026-08-11): `internal/conformance` now runs provider-free incident-shaped
tables for foreign identity, schema drift, missing evidence, invalid ranges/hashes, graph collisions,
contradictory verdicts, stale provider repairs and audit failures. The same semantic payload is wrapped
for Claude/Qwen/Codex adapter identities without starting a provider; ordered issue-code parity and
zero false accepts are asserted. Runtime quality totals persist first-pass validity, issue-class
counts, effective-verdict source and promotion-audit result alongside the W24H invocation counters;
the deterministic conformance trace records p95 provider starts at two.

Acceptance:

- false accepts = 0 for the incident corpus;
- equivalent adapter inputs produce the same ordered typed issues and terminal code;
- clean and sparse-with-gaps fixtures pass without recovery;
- no audit error permits promotion;
- `make contracts`, `make test`, `make lint` and `make build` pass with pinned toolchains;
- live validation, when requested after deterministic closure, uses only `acp-e2e-live-gate` and the
  canonical harness without changing release matrices or curated repos.

### Epic acceptance

- Foreign-run, wrong-snapshot and contradictory validator artifacts fail before semantic merge.
- Evidence line/excerpt/hash claims are verified against contained repository bytes.
- Unsupported semantic fields are never silently discarded and every promoted edge resolves.
- Provider-free selected-run audit is a mandatory fail-closed pre-promotion gate.
- Effective technical verdict is orchestrator-owned; provider findings/questions remain validated
  advisory input.
- A global recovery budget bounds provider invocations and is provider-parity tested.
- Sparse truthful analysis remains successful with explicit gaps; unsupported facts do not.
- Required CI remains deterministic, offline and independent of live provider binaries.

## Epic 25 — Task-first Live E2E and Hardened Runtime Evidence Alignment

Status (2026-08-11): **release boundary accepted; implementation not started.** This is a cross-epic
release-gate alignment task. The Task-first frontend migration starts only after `23O`; hardened
runtime evidence assertions start only after the public diagnostics from `24I` are stable.

### Goal

Keep the existing canonical trusted-machine release gate truthful after Epics 23 and 24 land: the
release-facing frontend `init-inspect` must follow the public Task-first journey, and live execution
reports must consume the final public audit/verdict/recovery evidence without depending on product
internals or preserving the retired shell.

Accepted boundary decisions: `init-inspect` selects one exact existing Task/Attempt/run snapshot and
never starts a second provider analysis; batch/frontend harnesses consume public API/report fields
only; optional `smoke tiny` Task-start remains a separate non-release owner decision and does not
expand canonical taxonomy by default.

### Non-goals

- no new canonical matrix profile, provider, curated repository, sweep or timeout taxonomy;
- no wrapper around `scripts/full-run-batch-matrix.sh` and no hosted live-provider CI workflow;
- no hidden legacy routes, selectors or compatibility DOM for `/home`, `/runs` or `/knowledge`;
- no second provider-backed analysis started by frontend snapshot inspection;
- no attempt to force live models to emit deterministic invalid artifacts; Epic 24 negative cases
  remain provider-free fixture/fault-injection coverage;
- no product API/schema invented by the harness; it consumes only contracts accepted by Epics 23
  and 24.

### 25A — Task-first frontend `init-inspect` cutover

**Depends on:** `23O` deterministic closure and accepted Task/Attempt routes/contracts.

What:

- keep the release-facing scenario ID `UI_E2E_SCENARIO=init-inspect` and existing frontend failure
  taxonomy, but replace the retired `Home / Runs / Knowledge / Changes` journey;
- inspect the exact Task and immutable Attempt bound to the selected backend snapshot;
- verify terminal Outcome, effective runner/config identity and contextual Pipeline Studio;
- traverse current Architecture map/document/Mermaid/evidence surfaces with explicit authority;
- review task/run-pinned semantic Changes and the authoritative full-workspace Publish gate;
- exercise global read-only Ask and citation return without mutating canonical architecture;
- retain desktop/tablet/mobile, keyboard/focus, critical axe, overflow and console-error checks;
- migrate operator-facing selectors, screenshots, result JSON evidence refs and contract tests in
  the same slice; do not retain hidden compatibility controls.

Expected files/modules:

- `ui/e2e/live-flow.spec.ts`, `ui/playwright.live.config.ts` only if artifact behavior changes;
- `scripts/frontend-live-e2e.sh`, `scripts/tests/frontend_live_e2e_contract_test.py`;
- Task/Attempt public UI/API clients and operator-facing test IDs introduced by Epic 23;
- `docs/TESTING_STRATEGY.md` and `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.

Tests:

- snapshot-bound Task/Attempt selection, reload and Back/Forward identity restoration;
- terminal success opens Outcome rather than active pipeline language;
- Pipeline Studio preserves Attempt identity and exposes bounded diagnostics;
- Architecture, evidence, Changes, Publish and Ask never silently switch authority;
- old shell routes/selectors are absent after cutover;
- frontend result reason mapping, active-run cleanup and dependent-skip behavior remain stable;
- deterministic fake/mock E2E passes before any trusted-machine run.

Acceptance:

- all three release providers pass the Task-first frontend `init-inspect` over `artifact_source=snapshot`;
- no release screenshot or assertion refers to the retired Home/Analyze/Knowledge shell;
- the scenario does not launch a second live analysis and does not mutate source repositories;
- exact Task, Attempt, run snapshot and publication identities remain consistent through the flow;
- existing `active_run_timeout`, `runtime_run_failed`, `browser_closed`, `api_unreachable`,
  `server_exited` and `playwright_failed` classifications remain distinguishable.

### 25B — Epic 24 public evidence consumption

**Depends on:** `24I` deterministic closure and stable public diagnostic fields.

What:

- extend batch/matrix/report inspection to consume the public pre-promotion audit result, effective
  technical verdict authority and global recovery-budget counters established by Epic 24;
- keep `runtime_contract_status`, artifact-quality status and release execution verdict separate;
- fail closed on any public deterministic audit error or inconsistent effective verdict;
- retain repair/invocation pressure as explicit evidence and enforce the accepted hard budget;
- prove identical report interpretation for Claude/Qwen/Codex without importing orchestrator or
  validator internals.

Expected files/modules:

- `scripts/full-run-batch.sh`, report helpers and focused script tests;
- `scripts/full-run-batch-matrix.sh` only where public report aggregation requires it;
- `scripts/verify-release-verdict.py` only if the accepted release-verdict contract changes;
- `docs/TESTING_STRATEGY.md`, `docs/RELEASE_LIVE_E2E_RUNBOOK.md` and operator assessment template.

Tests:

- clean public audit/verdict/budget fixture passes for all provider envelopes;
- audit error, foreign authority, contradictory effective verdict and exhausted budget fail with
  stable public classifications;
- reordered diagnostics and provider-specific prose do not change interpretation;
- historical report fixtures retain their documented read behavior;
- canonical profile totals, baseline/parallel-default invariant and release companion assessment
  requirements remain unchanged.

Acceptance:

- live release evidence proves the promoted snapshot passed the mandatory public technical gate;
- provider-authored opinion cannot be mistaken for the effective technical verdict;
- recovery-budget exhaustion is visible and cannot be hidden by a terminal provider success;
- no product-internal package, selector or validation helper is imported by the harness;
- required CI remains deterministic/provider-free; trusted live execution remains manual.

### 25C — Closure and optional Task-start diagnostic decision

What:

- run focused shell/Python/Playwright contract suites, deterministic UI scenarios, full DoD and
  embedded UI parity before trusted-machine validation;
- synchronize current behavior in testing strategy/runbook and remove retired screenshot/selector
  references;
- decide explicitly whether a small non-release `smoke tiny` UI Task-start diagnostic is worth its
  provider cost; absence of that owner decision does not expand canonical release taxonomy;
- execute canonical live validation only through `acp-e2e-live-gate` from a clean committed tree.

Acceptance:

- `make contracts`, `make test`, `make lint` and `make build` pass;
- frontend live contract tests prove only the accepted Task-first `init-inspect` release scenario;
- canonical release matrices and curated repos are byte-unchanged unless separately approved;
- release readiness still requires machine execution `PASS` plus accepted SWE UX and artifact-quality
  assessments for the same matrix ID.
