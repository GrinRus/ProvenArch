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

Status (2026-07-17): deterministic boundary/quality remediation is complete; trusted-machine R3
release evidence remains open and is the only release-readiness blocker after Epic 21. The
artifact-readiness remediation is merged at `57155786`, and fresh Codex smoke
`smoke-tiny-bank-20260716T165233Z` passed with strict failures `0` and init/refresh collect
`10/10`. Standalone `release-fast-20260716T185527Z` then stopped before product execution:
`qwen --version` passed, but the single canonical runtime-like artifact smoke did not create the
exact sentinel and correctly produced `operational_host_preflight_failed`. Release long/full and
accepted evidence remain blocked until Qwen artifact readiness succeeds on this or another trusted
host and the complete R3 sequence is repeated from one clean commit. A later standalone
`release-fast-20260717T114416Z` passed Qwen collection and exposed a product contract-shape defect
in Claude collection; its bounded missing-findings remediation and the ProductShell live-gate
alignment are now merged. Proposal staging-path leakage found by promoted-evidence review was also
fixed in PR #160. Post-merge Claude smoke `smoke-tiny-bank-20260718T210050Z` completed init and
refresh collect `10/10`, then failed because a focused step2 rewrite fixed invalid repo refs but
left `scoped to the current run` in Architecture Home and had no bounded changed-error cleanup.
That narrow recovery remediation must merge before the complete sequence restarts; no evidence
from stopped matrices is accepted.

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
- Это corrective successor для UI/IA частей Epics 16–17. Канонический target описан в
  [`UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md`](UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md). После
  реализации Epic 20 primary IA `Home / Runs / Knowledge / Changes`, contextual Setup и global
  Ask заменяют требование о восьми обязательных numbered stages; backend/runtime contracts,
  local-first boundary, deterministic required CI и release live-E2E guardrails сохраняются.
- Детальный delivery order, обязательные contract-first PR, code/test map, cutover/rollback и
  reference matrix зафиксированы в
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
