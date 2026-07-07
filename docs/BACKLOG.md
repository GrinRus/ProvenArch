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
- `9D legacy /api/qa/ask deprecation plan`

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
- целевой UI baseline зафиксирован в `docs/UI_CONSOLE_V2_DESIGN.md`
- approved visual references сохранены в `docs/assets/ui-console-v2/*.png`
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

Follow-up note (2026-04-22):
- статус `docs/BACKLOG.md` как active planning surface vs reference/history требует отдельного owner decision; этот cleanup-slice синхронизирует только terminology, не меняя роль документа.
