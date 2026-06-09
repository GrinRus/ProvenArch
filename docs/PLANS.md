# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон, текущие активные планы и инженерный operational mirror.

Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
- `docs/archive/PLANS_ARCHIVE_2026-05.md`
- `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Этот файл остаётся рабочим engineering mirror и active ExecPlan surface.

## Когда использовать
Используйте ExecPlan, если:
- работа затрагивает несколько модулей, или
- ожидаемое время > 30–60 минут, или
- затрагиваются контракты/схемы.

---

## Шаблон ExecPlan

### Plan ID
EP-YYYYMMDD-<slug>

### Context
Зачем это нужно? Какие ограничения важны?

### Goals (must have)
- [ ] ...

### Non-goals
- [ ] ...

### Approach
1) ...
2) ...
3) ...

### Files expected to change
- ...

### Acceptance criteria
- [ ] Тесты обновлены/добавлены
- [ ] Схемы валидируются
- [ ] Документация обновлена

### Risks
- ...

### Progress log
- YYYY-MM-DD: ...

---

## Active Plans
Tracker reconciliation from 2026-05-07 consolidated historical active plans into the remaining open slices below. Detailed evidence and classification are archived in `docs/archive/TRACKER_RECONCILIATION_2026-05-07.md`; original historical active plan text was moved to `docs/archive/PLANS_ARCHIVE_2026-05.md` under "Reconciled active plans from 2026-05-07".

### Continuous Backlog Queue Policy

Current engineering queue starts with `EP-20260602-onboarding-first-startup`; implement Epic 17 slices in `docs/BACKLOG.md` order (`17A` -> `17E`) unless a blocker creates a narrower prerequisite slice. `EP-20260527-ui-console-v2` remains visible for owner review/archive bookkeeping, but it is not the next engineering workstream.

Task selection rules:
- Completed plans whose only remaining item is owner review, merge/archive bookkeeping, or historical evidence retention are not next engineering work.
- Owner-decision and trusted-host/live-release items remain explicit blockers; do not run or edit them as normal backlog tasks without the required owner/trusted-machine prerequisites.
- Each selected slice gets a decision-complete ExecPlan/update before implementation, one focused implementation pass, self-review/fix loops, Full DoD (`make contracts`, `make test`, `make lint`, `make build`), then one commit.

### Plan ID
EP-20260608-medium-live-e2e-quality-ui

### Context
Run a medium live E2E assessment on a trusted local host after a user-reported weak local result: sparse artifacts and missing C4 diagrams. This is an operator evaluation, not a product code slice. The run must follow `acp-e2e-live-gate` and `docs/RELEASE_LIVE_E2E_RUNBOOK.md`: use direct public harness commands, avoid wrapper scripts, avoid canonical matrix edits, and classify host/provider/path blockers separately from ACP product defects.

The medium non-release profile is `regres long`: `posthog` (`single-path`, medium shard bucket) plus `ftgo-application` (`single-git_url`, medium shard bucket), default qwen-only baseline with `RUN_COUNT=1`. Because the current worktree may need this plan update, execute the harness from a separate clean worktree.

### Goals (must have)
- [ ] Run host/tree/provider/path preflight for the medium live E2E profile.
- [ ] Launch `regres long` through `scripts/full-run-batch-matrix.sh` using the public planner/harness path.
- [ ] Keep frontend init inspection enabled so UI artifact readability can be assessed from real UI/API evidence.
- [ ] Inspect `run_matrix_*`, `quality_report_*`, `reports/taskruns/*-quality.json`, promoted reports, diagrams and raw taskrun metadata.
- [ ] Evaluate whether the resulting artifacts are substantive, evidence-backed and complete enough for architecture review.
- [ ] Evaluate whether the UI makes the generated artifacts discoverable and readable without relying only on filesystem inspection.

### Non-goals
- [ ] Do not change runtime contracts, schemas, canonical matrix files or curated repo files.
- [ ] Do not treat this non-release diagnostic as release readiness.
- [ ] Do not bypass provider/toolchain/path prechecks with test-only overrides.
- [ ] Do not add a new wrapper around the matrix harness.

### Approach
1) Record this ExecPlan and keep the live run itself in a separate clean worktree.
2) Run fail-fast host checks: writable roots, clean tree, exact Go/Node toolchain, provider command, canonical path repo availability for `posthog`.
3) Generate the medium command with `scripts/live-e2e-plan.py --mode regres --size long --providers qwen --frontend-mode per_run --format shell`.
4) Execute the printed `scripts/full-run-batch-matrix.sh` command and monitor profile status, driver logs, inventories and generated reports.
5) Inspect backend quality outputs and promoted workspace artifacts, including C4 diagram presence/content.
6) Inspect frontend live E2E outputs and, where possible, open the resulting UI workspace for manual readability checks.
7) Report the black-box assessment with evidence paths, failure class if any, and concrete gaps.

### Files expected to change
- `docs/PLANS.md`
- Optional operator assessment under `reports/` if a durable report is useful after the run

### Acceptance criteria
- [ ] Preflight result is classified as passed or a concrete operational blocker.
- [ ] Medium harness command is recorded and launched only if preflight passes.
- [ ] Artifact quality is assessed from generated machine reports and direct artifact inspection.
- [ ] UI readability is assessed from frontend init evidence and manual UI/API inspection.
- [ ] Final answer names the primary failure class or confirms no blocking quality issue was found.

### Risks
- Medium live runs can take hours and can fail due to provider auth/quota/timeout rather than ACP defects.
- `posthog` path checkout may be missing or not pinned on the current host.
- Frontend inspection depends on successful backend snapshots; if backend fails, UI evidence may be dependent-skip rather than an independent UI result.

### Progress log
- 2026-06-08: Started medium live E2E artifact quality and UI readability assessment plan.
- 2026-06-08: Medium `regres long` diagnostic was mixed: `single-path/posthog` stopped as `operational_host_preflight_failed` because qwen headless probe timed out; `single-git_url/ftgo` backend and frontend hard-passed but produced operator-unusable artifacts. Root-cause triage points at enforced runtime prompt/quality policy, not C4 rendering: command-first collect/as-is/validator skeletons are valid with empty semantic model or placeholder drafts, qwen draft valid-artifact stop can accept those files, validator can return `PASS` with empty findings/questions for single-repo owner/dependency gaps, and live quality gates score/report this as warnings/analysis loss rather than hard failure. C4 diagrams are gap-only because promoted `final-run-index.json` has `semantic.entities=[]` and `semantic.edges=[]`.
- 2026-06-08: Added the remaining live-loop hardening surface: `/api/system/version` exposes actual running build metadata before workspace selection, Console V2 top bar no longer hard-codes `v0.1.1 beta`, and frontend live `init-inspect` now checks selected Review markdown and C4 Mermaid content for readability/placeholder/gap-only failures through `/api/artifacts`.

### Plan ID
EP-20260608-live-artifact-quality-hardening

### Context
Follow-up to `EP-20260608-medium-live-e2e-quality-ui`. The medium live run showed that a provider can produce structurally valid but substantively unusable analysis artifacts: placeholder top-level reports, empty findings, empty semantic model, gap-only C4 diagrams, generic proposal/changelog files, and at least one taskrun/staging document sourced from a provider tool directory instead of repository evidence. The frontend review UI then presents these artifacts as reviewable/ready because it mostly checks artifact presence and open-question counts.

This is a product-quality slice. It must keep the existing local-first/entity-per-file contracts intact unless explicitly synchronized through schemas, docs, validators, fixtures and tests.

### Goals (must have)
- [ ] Define and enforce an artifact-quality rubric across all analysis outputs: charter, collect shard manifests/docs, as-is reports, coverage/questions, findings, semantic model, C4 diagrams, proposals/changelog, taskrun indexes, citation indexes, and frontend review surfaces.
- [ ] Audit prompt correctness for each artifact-producing step (`collect`, `as-is`, `findings/validator`, `proposals`) so prompts require evidence-backed, non-placeholder content before final acceptance.
- [ ] Make placeholder or skeleton-only promoted artifacts fail quality gates for nontrivial live targets instead of passing with a low signal score.
- [ ] Make an empty semantic model a hard quality failure when collect/as-is completed against a non-empty repository.
- [ ] Make gap-only C4 diagrams a hard quality failure when the semantic model is empty or unusable.
- [ ] Reject hidden/provider/tooling files such as `.qwen/`, `.claude/`, `.codex/`, `.git/`, and similar generated side-effects from shard document manifests and promoted canonical docs.
- [ ] Require findings/questions to reflect critical coverage gaps. For example, owner/dependency/operational gaps must not coexist with `No findings reported.` in normal report mode.
- [ ] Update frontend review/readiness so the UI surfaces artifact-quality blockers and keeps generated artifacts readable without raw runtime noise dominating the review surface.
- [ ] Replace the hard-coded UI `v0.1.1 beta` label with runtime/build metadata so live screenshots show the actual binary/UI version under test.
- [ ] Update live E2E scoring so backend and frontend hard-pass require quality-readable artifacts, not only completed runs and screenshot capture.
- [ ] Evaluate artifact quality from the actual artifacts produced by each run. Do not rely on a copied bad-run fixture as the release/live verdict source.

### Non-goals
- [ ] Do not tune the curated release matrices or repo fixtures to hide the observed qwen behavior.
- [ ] Do not create a persistent `ftgo` bad-output fixture as the main quality gate. The live gate must inspect the current run's artifact tree every time.
- [ ] Do not remove the gap-node diagram fallback; it remains useful diagnostic output, but it must not be counted as a successful C4 result.
- [ ] Do not require live network/provider calls in required CI.
- [ ] Do not introduce hosted-mode or security/compliance enforcement scope.

### Approach
1) Implement a per-run artifact inventory/evaluator that walks the actual promoted workspace and taskrun/staging outputs for the selected backend/frontend run. It must score every artifact surface produced by that run, including missing surfaces.
2) Harden artifact canonicalization and docflow validation so shard documents must be inside the allowed workspace artifact surface and must not come from hidden/provider/tool/runtime side-effect directories.
3) Extend backend run-quality evaluation to emit `artifact_quality:` warnings or blocking signals for sparse top-level reports, placeholder text (`Provider wrote...`), empty semantic entities/edges, gap-only diagrams, missing model files, empty findings with critical coverage gaps, and placeholder proposals/changelog.
4) Add prompt snapshot/contract tests that verify collect first-action commands write marker-free validation-ready artifact pairs, draft first-action commands remain bootstrap-only until enriched, and final success is forbidden from generic placeholder text.
5) Tighten runtime prompt policy in `steppolicy`/`promptcontract`: collect first-action pairs must be minimally valid scoped evidence immediately, draft first-action skeletons may bootstrap required files, and single-repo validator prompts must not return `PASS` with zero findings/questions when coverage gaps remain.
6) Extend `scripts/e2e_batch_report.py` and Python tests so dynamically inspected bad artifact shapes classify as `quality_gates_failed`/`semantic_hard_fail`, including backend and frontend-selected provider surfaces. Tests may build small temporary artifact trees inline, but must not depend on a persistent copied live-run fixture.
7) Extend review API/UI with artifact-quality status per step and review surface. The UI should show hard blockers for empty model/gap-only C4/placeholders, distinguish baseline support files from analysis outputs, and keep artifact previews legible on desktop/mobile.
8) Update docs/runbook/testing strategy to document the quality rubric and rerun the medium live profile only after the dynamic evaluator and local tests pass.

### Files expected to change
- `internal/artifactquality/canonicalize.go`
- `internal/artifactquality/collect_bootstrap.go`
- `internal/contracts/docflow.go`
- `internal/orchestrator/docflow.go`
- `internal/orchestrator/quality.go`
- `internal/runtime/promptcontract/promptcontract.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `internal/runtime/steppolicy/policy.go`
- `internal/runtime/steppolicy/policy_test.go`
- `internal/runtime/qwencode/runner.go`
- `internal/reports/compiler.go`
- `scripts/e2e_batch_report.py`
- `scripts/tests/batch_failure_classification_test.py`
- `scripts/tests/frontend_live_e2e_contract_test.py`
- `internal/api/review_diff.go`
- `ui/src/components/TopStatusBar.tsx`
- `ui/src/components/StagePanels.tsx`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/TESTING_STRATEGY.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- Related unit/integration tests that construct temporary artifact trees or inspect generated run outputs

### Acceptance criteria
- [ ] Every live/backend/frontend run quality report includes a fresh artifact inventory and quality verdict for all produced and expected analysis surfaces.
- [ ] The observed sparse `ftgo` artifact shape fails when detected by the dynamic evaluator, without relying on a persisted copied fixture.
- [ ] Hidden/provider tool document paths in shard manifests are rejected before promotion and covered by unit tests.
- [ ] Backend quality JSON contains explicit `artifact_quality:` signals for every blocked artifact class found in the current run inventory.
- [ ] Prompt contract tests prove artifact-producing prompts require evidence-backed content, semantic entities/edges where applicable, and explicit non-placeholder final criteria.
- [ ] C4 gap-only output is visible as diagnostic output but is not counted as a successful C4 diagram in quality/readiness.
- [ ] Frontend review UI shows hard blockers for empty semantic model, gap-only diagrams, placeholder reports/proposals, and empty findings with critical coverage gaps.
- [ ] Frontend screenshots display version metadata from the running app/build, not a hard-coded release string.
- [ ] Frontend live E2E contract verifies not only screenshots/selectors but also artifact readability and quality blocker surfacing.
- [ ] Docs and runbook explain the rubric and how to inspect all resulting artifacts after a live run.
- [ ] Full DoD for the slice passes: `make contracts`, `make test`, `make lint`, `make build`.

### Risks
- Providers may need prompt changes that increase runtime duration; keep hard gates deterministic and let live gates validate provider-specific behavior.
- Empty findings can be valid for tiny deterministic or toy targets, so quality rules need an explicit nontrivial-target threshold instead of a blanket ban.
- Some generated support artifacts under `skills/` are expected baseline files. The quality rubric must separate baseline support surfaces from analysis outputs.

### Progress log
- 2026-06-08: Audited the failed medium live run artifact surfaces. Backend/frontend promoted workspaces have no `model/` files, C4 is gap-only, findings/proposals/changelog/top-level overviews are placeholders, shard manifests have citations but no entities/edges/findings, and one init shard staged a `.qwen/skills/.../SKILL.md` provider-side-effect file as an as-is document. UI review/readiness currently surfaces artifact presence but not these substantive quality failures. The top status bar also hard-codes `v0.1.1 beta`, so screenshots can mislead operators about the tested build version.
- 2026-06-08: Updated the plan to avoid making a persistent bad-run fixture the quality source of truth. The live gate must dynamically inspect and score the current run's generated artifacts every time; tests can use temporary in-test artifact trees only to protect evaluator behavior.
- 2026-06-08: Started implementation slice for dynamic backend artifact quality. Added fresh run artifact inventory to quality summary, blocking `artifact_quality:*` signals for sparse current-run surfaces, and collect manifest rejection for provider/tool side-effect document paths; no persistent bad-run fixture added.
- 2026-06-08: Completed the first backend/prompt hardening slice. Prompt policy now treats collect/as-is/proposals first-action skeletons as bootstrap-only, not final-acceptable content. Full local DoD passed with exact Node candidate: `make contracts`, `make test`, `make lint`, `make build`.
- 2026-06-08: Medium `regres long` diagnostic on `claude-code,codex-code` reproduced product-quality blockers before qwen recovery: collect manifests had citations/docs but no semantic entities/edges/findings, C4 was gap-only, promoted reports/proposals contained placeholder text, and frontend `init-inspect` failed on gap-only C4 before Review/mobile readability could be assessed. Current fix slice makes collect skeleton/repair artifacts carry minimal evidence-backed semantic signal, removes provider-placeholder draft text from normal and focused repair paths, and makes backend-cycle fail fast on fresh `artifact_quality:*` warnings before dependent frontend.
- 2026-06-08: Follow-up `claude-code` fresh run proved the empty-model blocker was fixed (`entities=17`, `edges=16`, `findings=16+`), but operator quality was still not acceptable: semantic output was scaffold-like repo/shard containment with repeated owner-gap findings, and C4 context still showed gap-only context nodes. Added dynamic `artifact_quality.semantic_scaffold_only` and `artifact_quality.c4_context_scaffold_only` blockers from temporary in-test artifact trees, not from a persisted bad-run fixture.
- 2026-06-08: Next fresh `claude-code,codex-code` diagnostic (`regres-long-posthog-ftgo-20260608T223130Z`) was stopped after direct staging inspection showed unchanged collect first-action artifacts: markdown still carried generic repository/path/owner-gap prose and manifests kept repo/shard `contains` plus generic owner-gap findings. The fix adds a collect bootstrap detector, emits `artifact_quality:*` when accepted collect manifests retain unchanged bootstrap docs/semantic, and strengthens the normal collect prompt with post-command targeted evidence enrichment plus `ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT` removal before successful exit.
- 2026-06-08: Follow-up diagnostic (`regres-long-posthog-ftgo-20260608T225807Z`) found a narrower repair-routing defect: Claude wrote only bootstrap markdown with `ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT`, then runtime selected manifest-only repair, which cannot rewrite the bad markdown. The next fix routes bootstrap-only authored docs through collect pair repair, keeps manifest-only repair for non-bootstrap docs only, and adds post-command enrichment requirements to the focused pair repair prompt.
- 2026-06-09: Fresh `claude-code,codex-code` diagnostic (`regres-long-posthog-ftgo-20260609T002218Z`) proved pair recovery no longer preserves the bootstrap marker, but exposed a quality defect before the matrix could finish: normal collect wrote bootstrap pairs and the default 20s post-artifact monitor terminated providers before targeted evidence enrichment. Recovery-only docs were validation-acceptable but low-signal, so the fix extends collect post-/partial-artifact enrichment windows for live adapters before recovery is attempted.
- 2026-06-09: Follow-up diagnostic (`regres-long-posthog-ftgo-20260609T004556Z`) showed the extended collect window lets several shards enrich successfully, but `posthog-devenv` still fell into pair recovery and the marker-free recovery heredoc was accepted as a 503-byte low-signal `Recovery Summary`. The next fix makes collect pair recovery a marker-bearing bootstrap surface again, requires targeted evidence enrichment before successful exit, and rejects marker-free Recovery Summary/Recovery Bootstrap scaffold docs in collect validation.
- 2026-06-09: Follow-up diagnostic (`regres-long-posthog-ftgo-20260609T012003Z`) showed strict collect validation now rejects marker-bearing first-action docs and routes bootstrap authored docs to pair recovery, but Claude executed only the recovery heredoc, left the marker-bearing `Recovery Bootstrap`, then stalled into `runtime_contract_failed`. The next fix changes focused collect pair recovery from a two-phase marker-bearing bootstrap into a marker-free validation-ready fallback pair with scoped path evidence and explicit recovery semantic signal; richer targeted enrichment remains optional if provider execution continues after the first valid repair write.
- 2026-06-09: Follow-up diagnostic (`regres-long-posthog-ftgo-20260609T014741Z`) showed pair recovery is now validation-ready, but normal Claude collect still writes legacy bootstrap-only docs first and would make the run recovery-dominated. The next fix removes `ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT` from normal collect first-action output and makes the primary collect pair marker-free, validation-ready scoped evidence so stalled providers can validate without dropping into recovery-only artifacts.

### Plan ID
EP-20260602-onboarding-first-startup

### Context
Current `v0.1.1` onboarding is too CLI-first: the operator must choose `--workspace` before the UI exists and must often seed at least one repo through `--auto-init` before the `Source` screen can take over. That contradicts the desired first-run UX for a local-first operator console: start ACP, choose or create an architecture workspace in the UI, add one or more target repositories from Git URL or local checkout path, choose the runner, validate readiness, then run analysis.

The current architecture still has important constraints that should be preserved:
- one active architecture workspace per served console session;
- `repos[]` is already the multi-repo source of truth in `workspace.yaml`;
- source repositories remain read-only inputs;
- deterministic `fake` stays the required-CI default and recommended first walkthrough runner;
- live providers remain explicit opt-in (`claude-code`, `qwen-code`, `codex-code`);
- existing `acp serve --workspace ...` direct mode must remain for CI, scripts, existing live E2E and users who already have a workspace.

### UX spec
Users:
- local operator / architect / tech lead starting ACP for the first time;
- existing ACP user reopening a known architecture workspace;
- multi-repo system owner who needs to analyze several local checkouts and/or Git URLs together.

Flow:
1) Operator runs `acp serve` with no workspace, or opens a direct-mode `acp serve --workspace ...` session.
2) If no workspace is selected, the UI opens an onboarding screen instead of the normal stage shell.
3) Step 1 `Workspace`: create a new architecture workspace path or reopen a recent/existing one; validate writable path, git availability and fixed layout readiness.
4) Step 2 `Sources`: add one or more target repos using `GitHub/GitLab URL` or `Local folder`, optional `ref`, and docs imports path; save `workspace.yaml` only after repo validation can produce a valid manifest.
5) Step 3 `Runner`: choose runner before analysis. `fake` is the recommended default and requires no provider command; `claude-code`, `qwen-code`, `codex-code` show command/auth availability via readiness checks and require explicit selection.
6) Step 4 `Ready`: show workspace/repo/runner/permission summary, then enter the existing `Source -> Readiness -> Charter -> Analysis -> Review -> Proposals -> Ask -> Publish` console.

Screens:
- `Welcome / Workspace`: local-only explanation, create/reopen workspace, recent workspaces, path validation, no marketing hero.
- `Repository Sources`: dense multi-repo table/cards with add/remove, source type, path/URL, ref, per-row diagnostics and docs imports path.
- `Runner`: segmented runner choice, provider availability, command override hints, permission mode summary, fake-first recommendation.
- `Ready`: compact checklist and primary action `Open console` / `Run first analysis`.
- Existing console shell: unchanged after onboarding completion, with `Source` still editable for later repo changes.

States:
- no workspace selected: only onboarding APIs are active; pipeline/run/publish actions disabled;
- new workspace without manifest: layout can exist, but `workspace.yaml` is draft until at least one repo is configured;
- existing workspace: parse manifest, hydrate onboarding fields, allow skip to console after validation;
- invalid path/not writable/not git-initable: inline error with remedy near workspace path;
- repo path missing, duplicate repo name, invalid URL/ref, private git auth failure: row-level diagnostics and no run action;
- runner unavailable: selectable only as draft with explicit blocker, or disabled until command/auth is fixed; `fake` remains always available;
- direct mode with `--workspace`: bypass onboarding unless manifest is missing and explicit onboarding mode is enabled.

Information architecture:
- Onboarding is a pre-console setup surface, not a ninth stage.
- `arch-workspace` path is process/session-level app context.
- target repos and docs imports are workspace manifest data.
- runner choice is setup-time execution context plus persisted runtime profile where applicable; live provider selection must be visible before the first run.

Risks:
- letting the UI choose arbitrary local paths is acceptable only because ACP is local-first and bound to loopback, but errors must be explicit and no source repo mutation should happen;
- browser directory pickers are not portable enough for the MVP path, so typed paths/recent workspaces are the safe baseline;
- adding no-workspace `serve` changes public CLI behavior and live E2E startup assumptions; direct `--workspace` compatibility must be tested;
- runtime mode is partly process-scoped today, so onboarding runner selection requires a deliberate service lifecycle design instead of only editing `workspace.yaml`.

### Goals (must have)
- [x] Add an onboarding-first startup path where `acp serve` can start a local UI without a preselected workspace.
- [x] Let the UI create or reopen an architecture workspace before entering the normal console.
- [x] Let the UI configure one or more target repositories in `repos[]` using Git URL or local checkout path, with optional `ref`.
- [x] Require an explicit runner choice before first analysis; show `fake` as recommended and live providers as opt-in with availability diagnostics.
- [x] Keep existing `acp serve --workspace ...`, `--auto-init`, `--repos-file`, `acp init-workspace`, `acp run` and CI/batch flows working.
- [x] Keep backend API/schema/runtime artifact/workspace manifest contracts unchanged unless a slice explicitly updates docs, validators, fixtures and tests in the same change.
- [x] Update README/INSTALL/TROUBLESHOOTING only after implementation matches the new behavior.
- [x] Add deterministic CLI/API/UI coverage for onboarding and retain existing direct-mode `init-inspect` live E2E behavior without adding a new provider-live release scenario.
- [ ] After owner review/merge, move this completed implementation plan to archive.

### Non-goals
- [x] No hosted workspace picker, cloud storage, remote credential store or SCM OAuth flow.
- [x] No file browser integration that enumerates arbitrary local directories server-side.
- [x] No change to source repo write policy; target repositories stay read-only inputs.
- [x] No expansion of required live provider release gates.
- [x] No removal of direct CLI/bootstrap mode.

### Approach
1) Implement a launcher/no-workspace server mode behind `acp serve` while preserving direct `--workspace` service construction.
2) Add onboarding APIs for workspace path validation/create/open, recent workspace metadata, repo manifest draft save/validate and runner readiness.
3) Refactor service construction so the backend can attach a selected workspace/runtime after onboarding without restarting the process, or use a narrowly scoped launcher server that swaps into the existing workspace server.
4) Build the onboarding UI as a pre-console surface; keep the current stage shell unchanged after setup completion.
5) Persist valid repo sources through existing `workspace.yaml` rendering/validation; do not introduce a second source-of-truth file for repos.
6) Treat runner selection as required UI state before `Run first analysis`; use `fake` for deterministic onboarding e2e and show provider diagnostics for headless choices.
7) Update live/deterministic e2e carefully: add onboarding coverage without breaking release-facing direct-mode `UI_E2E_SCENARIO=init-inspect`.
8) Sync docs after behavior lands, making README/INSTALL distinguish UI-first setup, direct workspace mode and multi-repo setup.

### Files expected to change
- `cmd/acp/main.go`, `cmd/acp/main_test.go`
- `internal/api/*`
- `internal/workspace/*` only if workspace open/create helpers need narrow extraction
- `internal/doctor/*` only if runner/workspace checks need reusable launcher diagnostics
- `ui/src/App.tsx`, `ui/src/components/*`, `ui/src/hooks/*`, `ui/src/lib/*`, `ui/src/App.test.tsx`
- `ui/e2e/live-flow.spec.ts`, `scripts/frontend-live-e2e.sh` only if startup flow/test harness changes
- `docs/PLANS.md`, `docs/BACKLOG.md`, `docs/STAKEHOLDER_DOC.md`, then README/INSTALL/TROUBLESHOOTING after implementation

### Acceptance criteria
- [x] `acp serve` without `--workspace` starts a loopback onboarding UI and does not require a repo CLI flag.
- [x] `acp serve --workspace <path>` still starts the existing single-workspace console path.
- [x] Onboarding can create a new workspace, initialize fixed layout/git, configure multiple repos, choose `fake`, validate readiness and enter the console.
- [x] Onboarding can reopen an existing workspace and either skip to console or edit sources/runner before continuing.
- [x] The first analysis cannot start until workspace, at least one repo and runner are valid.
- [x] Multi-repo `repos[]` names are unique, path/git URL modes are validated, and row-level errors are recoverable in the UI.
- [x] Runner selection distinguishes `fake` availability from live provider command/auth blockers.
- [x] Source repos are not mutated during onboarding validation or fake analysis.
- [x] Existing direct-mode live E2E contract remains compatible; onboarding gets deterministic/fake CLI/API/UI coverage.
- [x] `git diff --check`, focused tests and Full DoD pass.

### Test plan
- Go CLI tests:
  - `serve` without workspace starts launcher/dry-run successfully;
  - `serve --workspace` direct mode still validates and serves;
  - invalid workspace paths and missing git/writable surfaces return actionable errors;
  - auto-init and repos-file compatibility are unchanged.
- API tests:
  - onboarding status, workspace create/open, repo draft validate/save and runner readiness;
  - no pipeline actions are allowed before workspace selection;
  - selected workspace uses the same service/API behavior as direct mode.
- UI unit tests:
  - onboarding workspace path states;
  - multi-repo add/remove/path/git URL/ref validation;
  - runner required state and provider availability copy;
  - transition into existing Console V2 shell.
- E2E:
  - deterministic onboarding fake flow from blank launch to Source/Readiness/Analysis;
  - existing `UI_E2E_SCENARIO=init-inspect` direct-mode regression;
  - no new required provider-live scenario without owner-approved release-gate slice.
- Full DoD:
  - `git diff --check`
  - `make contracts`
  - `make test`
  - `make lint`
  - `make build`

### Slice ExecPlan - 17A-17E Onboarding-first beta baseline

Status: done.

Goals:
- [x] Add `acp serve` launcher mode when `--workspace` is omitted, with `--workspace` direct mode preserved.
- [x] Add onboarding status/workspace create-or-open API, runtime selection API and session service reattachment.
- [x] Support a new workspace draft state where layout/git can exist before `workspace.yaml` is valid.
- [x] Add a UI pre-console onboarding shell for workspace, sources, runner and ready summary.
- [x] Save multi-repo sources through existing `workspace.yaml` semantics and keep Source editable after onboarding.

Non-goals:
- [x] Do not change workspace schema, runtime artifact contracts, CLI `run` behavior or provider contracts.
- [x] Do not remove or weaken `--workspace` direct mode.
- [x] Do not add browser directory picker, hosted mode or new provider-live release scenario.

Implementation notes:
- Prefer reusing existing `workspace.InitLayout`/`EnsureLayout`/git-init helpers rather than adding a parallel workspace model.
- Treat recent workspaces as optional local UI metadata only if it can be done without new cross-machine contracts; otherwise defer recents to 17B.
- Keep launcher APIs loopback-only with the same local server assumption as the rest of ACP.

Validation:
- [x] Focused Go tests for launcher/direct serve modes.
- [x] Focused UI tests for workspace/source/runner onboarding and Console V2 transition.
- [x] `git diff --check`.
- [x] Full DoD after the slice implementation.

### Progress log
- 2026-06-02: Created onboarding-first startup plan after UX review found that current first-run flow is too CLI-first: workspace is chosen before UI, while target repos and runner should be selected in UI. No code implemented in this planning slice.
- 2026-06-02: Implemented onboarding-first beta baseline: `acp serve` launcher mode, optional `--workspace` direct mode, `/api/onboarding/status|workspace|runtime`, workspace draft/session service state, pre-console OnboardingShell, multi-repo source diagnostics, mandatory runner selection and transition into Console V2. Updated README/INSTALL/TROUBLESHOOTING/ARCHITECTURE/STAKEHOLDER docs. Validation passed: focused Go/API tests, focused UI tests, `git diff --check`, `make contracts`, `make test`, `make lint` with Go toolchain in PATH, and `make build`. Provider-live release gate was not expanded.

### Plan ID
EP-20260602-long-run-review-diff

### Context
Console V2 made long runs observable through mission control, stage timeline and activity logs, but the daily operator workflow still has a gap: during a long provider run the user needs to understand which step is producing reviewable evidence, which artifacts are already available, and what changed in the workspace Git tree without leaving the UI. The approved follow-up scope is local UI/API only: expose run review summaries and real workspace Git diffs, then wire them into Analysis, Review, Proposals and Publish.

Constraints:
- runtime artifact schemas, `workspace.yaml`, CLI `acp run`, provider contracts and public live E2E shell stay unchanged;
- Git diff source of truth is the architecture workspace Git repository, never target repos;
- long-run states must be reviewable through artifacts, logs, evidence and diff, not only through raw logs;
- existing operator-facing selectors remain stable where possible.

### Goals (must have)
- [x] Add `GET /api/pipeline/runs/<run_id>/review-summary` with canonical `step0..step4` summaries built from existing run status, logs, artifacts and taskrun refs.
- [x] Add `GET /api/git/diff` for workspace-only Git status, folder summaries, selected file status and text hunks with strict path normalization.
- [x] Add a persistent active-run strip across Console V2 stages with run status, current step, provider, progress, warnings/errors and cancel.
- [x] Rework `Analysis` toward step-level review: step cards and selected-step tabs for artifacts, logs, evidence and diff.
- [x] Add a Review Queue and use real diff data in `Review`, `Proposals` and `Publish`.
- [x] Turn `Publish` into a real Git Review Room: folder summary, changed file list, selected hunks and existing publish gate/actions.
- [x] Cover queued/no logs/partial artifacts/failed/canceled/stale/no changes/binary diff states with explicit UI/API states where data is available.
- [x] Keep direct-mode live E2E contract unchanged.
- [ ] After owner review/merge, move this completed follow-up plan to archive.

### Non-goals
- [x] Do not change workspace manifest schema, runtime artifact schemas, provider contracts, CLI flags or `acp run` behavior.
- [x] Do not add approval persistence, hosted mode, source repo mutation, new provider-live release gate or new release-facing UI E2E scenario.
- [x] Do not introduce a second artifact source of truth beyond existing workspace files, run logs, run artifacts and Git status.

### Approach
1) Add narrow backend local APIs for run review summaries and workspace Git diff, with tests for path safety, empty diffs, binary files and step mapping.
2) Add frontend contracts and hooks for review summary and Git diff loading, reusing current run selection/polling.
3) Add `ActiveRunStrip` into the existing shell without changing the stage rail or onboarding boundary.
4) Update Analysis, Review, Proposals and Publish to expose artifacts/logs/evidence/diff in consistent tabs.
5) Add UI tests for the new surfaces and preserve existing selectors/live-shell expectations.
6) Run focused tests and Full DoD before committing the follow-up slice.

### Files expected to change
- `internal/api/server.go`
- `internal/api/review_diff.go`
- `internal/api/server_test.go`
- `ui/src/lib/*`
- `ui/src/hooks/*`
- `ui/src/components/*`
- `ui/src/App.tsx`, `ui/src/App.test.tsx`, `ui/src/styles.css`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Review summary returns five canonical steps for init runs and handles no logs, failed/canceled/stale and partial artifact states without panics.
- [x] Git diff returns valid empty diffs, rejects unsafe paths, reports untracked/modified/deleted/binary files and filters by folder/file/run/step where applicable.
- [x] Active run strip remains visible after console entry across stages.
- [x] Analysis exposes current/failed step details with artifacts, logs, evidence and diff.
- [x] Review Queue is visible and can navigate to reviewable artifacts.
- [x] Proposals and Publish no longer show the old partial line-level diff placeholder when real Git diff data is available.
- [x] Publish shows real workspace folder/file/hunk diff data while preserving hard blockers, warnings, open questions and explicit Git actions.
- [x] `git diff --check`, focused tests and Full DoD pass.

### Test plan
- Go/API:
  - review summary step mapping from run status/logs/artifacts;
  - review summary no-log failed/partial cases;
  - Git diff untracked/modified/deleted/folder filter/file filter/empty/binary/invalid traversal.
- UI:
  - active run strip persistence;
  - step cards and selected-step tabs;
  - Review Queue counts/navigation;
  - real diff state in Review/Proposals/Publish.
- Deterministic checks:
  - focused Go/API tests;
  - focused UI test suite;
  - `git diff --check`;
  - Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Progress log
- 2026-06-02: Started long-run review/diff follow-up after UX review found long provider runs were observable through logs but not sufficiently reviewable through step-level artifacts and workspace Git diff. Scope is local UI/API only; no schema/runtime/provider/live-shell changes.
- 2026-06-02: Implemented long-run review/diff follow-up: local `review-summary` and workspace Git diff APIs, persistent active-run strip, Analysis step review tabs, Review Queue, real diff tabs in Review/Proposals/Publish and Git Review Room folder/file/hunk view. Found and fixed a real infinite-update bug when opening Diff tabs by stabilizing the Git diff callback. Validation passed: `git diff --check`, focused `./scripts/run-go.sh test ./internal/api`, UI suite `61/61`, `make contracts`, `make test`, `make lint`, `make build`, plus rendered in-app browser smoke on a temporary fake workspace through Analysis/Review/Proposals/Publish with no browser console errors. Runtime schemas, `workspace.yaml`, CLI `acp run`, provider contracts and live shell were unchanged.

### Plan ID
EP-20260527-ui-console-v2

### Context
После утверждения 9-screen UI vision нужно зафиксировать Console V2 как реализуемый backlog поверх текущего stage-based UI baseline. Цель - собрать лучший вариант из Mission Control, Evidence Review Workbench, Domain Map, Run Timeline и Git Review Room без изменения backend/API contracts. В этот же feature wave должна войти миграция live E2E логики, иначе новый shell может пройти code review, но сломать trusted-machine release diagnostics.

### Goals (must have)
- [x] Зафиксировать целевой design baseline в `docs/UI_CONSOLE_V2_DESIGN.md`.
- [x] Сохранить approved PNG references в `docs/assets/ui-console-v2/`.
- [x] Добавить Epic 16 в `docs/BACKLOG.md` с кодовыми, UX и live E2E slices.
- [x] Зафиксировать V2 live E2E contract: scenarios, selectors, diagnostics and reason taxonomy.
- [x] Зафиксировать screen data source map so implementation stays on existing APIs unless a later slice changes scope.
- [x] Реализовать unified shell V2: top health strip, stage rail, center workbench, right inspector, bottom activity drawer.
- [x] Реализовать stage surfaces: Source, Readiness, Charter, Analysis, Review Evidence, Review Domain Map, Proposals, Ask, Publish.
- [x] Сохранить или явно мигрировать стабильные operator-facing `data-testid` для UI/unit/live E2E; hidden compatibility controls не возвращать.
- [x] Обновить Playwright live E2E operator journey под V2 shell/stages, включая evidence/logs/runtime safety/Git publication assertions.
- [x] Сохранить текущий public live shell как `UI_E2E_SCENARIO=init-inspect`; Ask покрывать optional `UI_E2E_QA_SMOKE=1`, а cancel/page-close/domain-map держать в deterministic или optional diagnostic coverage до отдельного live-gate решения.
- [x] Сохранить frontend live E2E diagnostics: screenshots/traces refs, stage context and existing reason taxonomy.
- [x] Обновить testing docs/runbook после implementation.
- [x] Выполнить full DoD для implementation slice: `make contracts`, `make test`, `make lint`, `make build`.
- [ ] После owner review/merge перенести план в архив.

### Non-goals
- [x] Не менять backend API, schemas, runtime artifact contracts, workspace contracts или CLI flags в UI-only slice.
- [x] Не добавлять hosted/security/compliance enforcement.
- [x] Не делать provider-live checks required CI; live provider matrix остаётся manual trusted-machine gate.
- [x] Не коммитить сгенерированные PNG mockups как product assets без отдельного owner decision.

### Approach
1) Use `docs/UI_CONSOLE_V2_DESIGN.md` as the product/UX source for screen inventory, shell contract, state model and acceptance checklist.
2) Implement in reviewable PR slices from `docs/BACKLOG.md` Epic 16, starting with shared shell primitives and stage status model.
3) Reuse existing hooks/API clients; keep backend/API schemas unchanged unless a later slice explicitly changes scope.
4) Use the data source map in `docs/UI_CONSOLE_V2_DESIGN.md`; render explicit empty/partial states for missing data instead of silently adding API fields.
5) Move stage surfaces one by one, preserving existing critical operator-facing selectors or migrating tests deliberately; do not reintroduce hidden compatibility DOM.
6) Update `ui/e2e/live-flow.spec.ts` in the same wave:
   - `init-inspect`: validate Source/Readiness, run Analysis, inspect Review Evidence, check Publish gate, and assert top strip/right inspector/activity drawer on each transition.
   - optional `UI_E2E_QA_SMOKE=1`: submit async QA inside `init-inspect`, poll result, assert confidence/citations/unresolved/read-only safety and context/runtime artifact links.
   - cancellation/page-close: keep deterministic fake-runtime UI/API coverage outside the provider-live release gate.
   - domain map: add unit/component/fake-fixture diagnostics first; add a live diagnostic only after stable model fixtures and owner-approved release-gate semantics exist.
7) Preserve Playwright failure artifacts from `ui/playwright.live.config.ts` (`trace=retain-on-failure`, `screenshot=only-on-failure`, `video=retain-on-failure`) and make sure result JSON diagnostic refs lead to the `UI_E2E_OUTPUT_DIR` results.
8) Update `scripts/frontend-live-e2e.sh` and script tests only if the public live shell or diagnostic refs change; do not add `ask-readonly`/`domain-map-diagnostic`/page-close scenarios or new frontend reason codes unless a separate classification/release-gate slice is approved.
9) Sync README/ARCHITECTURE/testing/runbook docs only when the implemented UI behavior changes.

### Slice ExecPlan - 16A UI V2 shell foundation

Status: done.

Goals:
- [x] Refine existing shell primitives (`AppShell`, `TopStatusBar`, `StageRail`, `RightInspector`, `ActivityDrawer`) toward the approved V2 shell without restoring legacy panels.
- [x] Extract a shared stage status model for the eight-stage rail.
- [x] Add baseline keyboard/focus contract for rail navigation and shared inspector/drawer surfaces.
- [x] Surface runtime permission mode and Git publication path in the shared shell using existing UI state.

Non-goals:
- [x] No backend API, schema, runtime contract, CLI flag or workspace contract changes.
- [x] No stage-specific redesign beyond shared shell foundation.
- [x] No `UI_E2E_SCENARIO` expansion, no new frontend reason taxonomy, no hidden compatibility controls.

Implementation notes:
- Use current React/Vite components and hooks only.
- Add/keep visible operator-facing `data-testid` selectors for V2 shared shell surfaces.
- Render missing Git/publication data as explicit partial/empty state in the inspector.

Validation:
- [x] Focused UI unit tests for stage rail keyboard navigation and shell shared surfaces.
- [x] Browser visual QA for desktop and narrow viewport on embedded build.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Slice ExecPlan - 16B Source + Readiness consolidation

Status: done.

Goals:
- [x] Add a Source repo table with name, source, ref, analysis include/exclude and validation status.
- [x] Preserve Git URL/local path editing, docs imports path and advanced `workspace.yaml` editor.
- [x] Add Readiness cards for workspace, repositories, runtime provider, permissions and artifacts.
- [x] Add a compact runtime profile summary without making advanced runtime settings the primary flow.

Non-goals:
- [x] No backend API, schema, runtime contract, CLI flag or workspace contract changes.
- [x] Do not add guided include/exclude editing until a separate workspace-contract slice approves it.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Use existing `ValidateResponse`, `DoctorResponse`, runtime settings state and artifact counters.
- Show include/exclude as an explicit advanced-only/partial state because `GuidedRepo` does not own that contract today.
- Keep existing Source/Readiness actions and `data-testid` selectors stable.

Validation:
- [x] Focused UI unit tests for Source table and Readiness summary cards.
- [x] Browser visual QA for Source and Readiness on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Slice ExecPlan - 16C Charter workbench

Status: done.

Goals:
- [x] Add a Charter wizard summary for project, scope, NFR priorities and rules before the editable form.
- [x] Add domain/team card overview using existing baseline bundle artifact metadata, with explicit empty/partial states.
- [x] Add baseline prompt bundle status covering prompt packs, live-consumed prompts, reference-only prompts and bundle warnings.
- [x] Keep the existing charter artifact editor and Git helper actions available without changing backend APIs.

Non-goals:
- [x] No backend API, schema, runtime contract, CLI flag or workspace contract changes.
- [x] Do not add a new cards API or infer domain/team ownership beyond existing artifact paths/metadata.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse current wizard state, baseline bundle response and Git helper state.
- Preserve existing baseline editor labels/buttons/tests while adding V2 summary surfaces.
- Render missing card artifacts as explicit empty/partial state rather than inventing data.

Validation:
- [x] Focused UI unit tests for Charter summary, card overview and prompt bundle status.
- [x] Browser visual QA for Charter on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Slice ExecPlan - 16D Analysis mission control

Status: done.

Goals:
- [x] Add Analysis run progress summary with selected run id, runtime/provider, status, current step and warnings/errors.
- [x] Add canonical step timeline for `step0..step4` using existing run status and logs.
- [x] Add shard/log-derived table with step, scope, provider, status, artifact/log reference and failed-row drilldown.
- [x] Keep existing run actions, pending permissions, run history and bottom activity drawer behavior.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not add a new shard API; derive table rows from existing run logs/status/artifacts and show partial state when data is sparse.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse `RunStatusResponse`, `RunLogEntry`, current artifacts and existing runtime setup state.
- Add stable V2 selectors: `analysis-run-progress`, `analysis-run-timeline`, `analysis-shard-table`, `analysis-review-blocker-btn`.
- Keep logs adjacent through the existing activity drawer and add warning/error summary in the Analysis workbench.

Validation:
- [x] Focused UI unit tests for run progress, timeline and shard/log table.
- [x] Browser visual QA for Analysis on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Analysis mission control using existing run status/log/artifact data, added V2 selectors (`analysis-run-progress`, `analysis-run-timeline`, `analysis-shard-table`, `analysis-review-blocker-btn`), verified terminal fake runs do not leave active shard rows, and kept warning-only runs from enabling the blocker action.

### Slice ExecPlan - 16E Review evidence workbench

Status: done.

Goals:
- [x] Add Review Evidence tabs for evidence workbench and domain map placeholder routing, preserving the single `Review` stage.
- [x] Add grouped artifact explorer by workspace folder using existing selected-run artifacts.
- [x] Add primary evidence preview that handles markdown/text and existing Mermaid diagram rendering without raw-path-first layout.
- [x] Add citation/coverage/trust summary surfaces derived from existing coverage summary, open questions, findings/diagram artifacts and selected artifact state.
- [x] Keep current artifact open flow, selected artifact selectors and diagram preview behavior available for unit/live E2E migration.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not implement the interactive domain map; reserve `review-domain-map` as an explicit future/partial state for `16F`.
- [x] Do not add approve/flag persistence or Git publish gating mutations in this slice.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse `nonDiagramArtifacts`, `diagramArtifacts`, `selectedArtifact`, `selectedArtifactContent`, `coverageSummary`, `openQuestions` and `onOpenArtifact`.
- Add stable V2 selectors: `review-view-evidence-tab`, `review-view-domain-map-tab`, `review-artifact-explorer`, `review-evidence-preview`, `review-domain-map`, `review-citation-coverage`.
- Render missing data as explicit empty/partial states; do not infer citations or claim coverage beyond existing artifacts/text.

Validation:
- [x] Focused UI unit tests for artifact grouping, evidence preview, citation/coverage/trust panels and domain-map partial state.
- [x] Browser visual QA for Review on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Review Evidence Workbench with visible V2 selectors (`review-view-evidence-tab`, `review-view-domain-map-tab`, `review-artifact-explorer`, `review-evidence-preview`, `review-domain-map`, `review-citation-coverage`), grouped artifacts by workspace folder, preserved artifact/diagram open flow, and kept Domain Map as an explicit 16F partial state.

### Slice ExecPlan - 16F Review domain map

Status: done.

Goals:
- [x] Replace the Review Domain Map placeholder with a useful map workbench derived from existing selected-run artifacts.
- [x] Show domain/service/model nodes from `model/entities/*` and domain agent outputs from `reports/agent-outputs/domains/*`.
- [x] Show model edges from `model/edges/*` with relationship direction, relation type and evidence navigation.
- [x] Add an ownership/coverage/cross-repo inspector that makes sparse or missing model data explicit.
- [x] Preserve Review Evidence tab behavior and artifact open flow for model/entity/domain artifacts.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not parse or validate full model YAML in the browser; derive the map from artifact paths/labels and show partial states where content would be required.
- [x] Do not add graph editing, approval persistence, proposal branch mutations or Publish gate behavior in this slice.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse `nonDiagramArtifacts`, `diagramArtifacts`, `coverageSummary`, `openQuestions` and `onOpenArtifact`.
- Add stable V2 selectors: `review-domain-map-canvas`, `review-domain-map-inspector`, `review-domain-map-node`, `review-domain-map-edge-list`, `review-domain-map-empty`.
- Keep navigation artifact-backed via existing `ArtifactPathButton` and render missing ownership/cross-repo data as explicit partial state.

Validation:
- [x] Focused UI unit tests for populated model map, edge list, artifact navigation and sparse-model empty state.
- [x] Browser visual QA for Review Domain Map on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Review Domain Map with visible V2 selectors (`review-domain-map-canvas`, `review-domain-map-inspector`, `review-domain-map-node`, `review-domain-map-edge-list`, `review-domain-map-empty`), model/entity/edge/domain artifact navigation, explicit sparse-model partial states, and no backend/API/schema/live-shell changes.

### Slice ExecPlan - 16G Proposals review room

Status: done.

Goals:
- [x] Replace the basic Proposals artifact list with a review-room layout for proposal/changelog artifacts.
- [x] Add proposal package list and preview tabs for proposal body, linked evidence, changelog and explicit diff partial state.
- [x] Add proposal quality, unresolved blocker and publication path panels using existing artifacts, `openQuestions`, `proposalBranch` and `gitStatus`.
- [x] Preserve existing artifact open flow and proposal/changelog routing from Review/inspector.
- [x] Surface proposal branch path before Publish without adding Git mutations to this slice.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not implement proposal approval persistence, Git diff API, commit actions or branch creation in Proposals; keep those in Publish.
- [x] Do not parse full proposal markdown for semantic quality; derive package/type/status from artifact paths and labels.
- [x] No live E2E shell expansion beyond existing `init-inspect`.

Implementation notes:
- Reuse `nonDiagramArtifacts`, `selectedArtifact`, `selectedArtifactContent`, `openQuestions`, `proposalBranch`, `gitStatus` and `onOpenArtifact`.
- Add stable V2 selectors: `proposals-review-room`, `proposals-artifact-list`, `proposal-preview-tabs`, `proposal-preview-panel`, `proposal-quality-panel`, `proposal-publication-path`.
- Render missing ADR/RFC/changelog/diff data as explicit partial states.

Validation:
- [x] Focused UI unit tests for proposal grouping, preview tabs, evidence/changelog partial states and publication path panel.
- [x] Browser visual QA for Proposals on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Proposals Review Room with V2 selectors (`proposals-review-room`, `proposals-artifact-list`, `proposal-preview-tabs`, `proposal-preview-panel`, `proposal-quality-panel`, `proposal-publication-path`), artifact-derived package grouping, preview/evidence/changelog/diff tabs, publication path handoff to Publish, and no backend/API/schema/live-shell changes.

### Slice ExecPlan - 16H Ask read-only Q&A console

Status: done.

Goals:
- [x] Add an Ask workbench with async Q&A run history and selected-run answer view.
- [x] Show answer confidence, citations, unresolved assumptions and explicit related-entity/edge partial state using existing QA response fields only.
- [x] Add a read-only runtime safety panel that explains taskrun-only QA writes and links context-pack/runtime-execution audit artifacts.
- [x] Preserve current async Ask flow, visible operator selectors and optional `UI_E2E_QA_SMOKE=1` assertions.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not add mutation actions, canonical artifact writes, approval persistence or new QA result fields in this slice.
- [x] Do not expand release-facing live E2E scenarios or reason taxonomy.
- [x] Do not restore hidden compatibility controls.

Implementation notes:
- Reuse `POST /api/qa/runs`, `GET /api/qa/runs/<run_id>` and `GET /api/qa/runs?limit=20` via the existing QA API client.
- Add stable V2 selectors: `qa-run-history`, `qa-answer-panel`, `qa-citations-panel`, `qa-readonly-safety-panel`.
- Keep existing visible selectors (`qa-panel`, `qa-question-input`, `qa-ask-btn`, `qa-run-status`, `qa-answer`) so current unit/live diagnostics remain valid.
- Render missing citations/unresolved/related entities as explicit empty/partial state rather than inventing API data.

Validation:
- [x] Focused UI unit tests for QA run history, selected answer, citations/unresolved/confidence and read-only safety/audit links.
- [x] Browser visual QA for Ask on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Ask read-only Q&A console with V2 selectors (`qa-run-history`, `qa-answer-panel`, `qa-citations-panel`, `qa-readonly-safety-panel`), existing async QA APIs, selected-run audit links (`context-pack.json`, `qa-answer.json`, `runtime-execution.json`), optional QA-smoke assertions and explicit partial state for structured related entities/edges. No backend/API/schema/runtime contract/live-shell expansion changes.

### Slice ExecPlan - 16I Publish gate

Status: done.

Goals:
- [x] Replace the basic Publish Git helper surface with a Git Review Room layout.
- [x] Show folder-level workspace artifact summary, selected artifact preview and explicit diff partial state using existing artifact data only.
- [x] Add publish gate/checklist/blockers, commit plan, proposal branch path and prepared commit-message copy action.
- [x] Preserve existing Git commit/proposal branch mutations and visible operator selectors while adding V2 selectors.

Non-goals:
- [x] No backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not implement a Git diff API, artifact selection persistence or approval workflow in this slice.
- [x] Do not expand release-facing live E2E scenarios or reason taxonomy.
- [x] Do not restore hidden compatibility controls.

Implementation notes:
- Reuse existing selected-run artifact index, selected artifact content, open questions, Git message/status and proposal branch state.
- Add stable V2 selectors: `publish-diff-summary`, `publish-preview-tabs`, `publish-gate-panel`, `publish-commit-plan`, `publish-commit-selected-btn`.
- Keep current Git helper actions visible and mutation-compatible with `/api/git/commit` and `/api/git/proposal-branch`.
- Render missing real Git diff/dirty folder data as explicit partial state until a separate backend slice exposes it.

Validation:
- [x] Focused UI unit tests for folder summary, selected artifact preview, publish gate/checklist, commit plan and Git actions.
- [x] Browser visual QA for Publish on desktop and narrow viewport.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Implemented Publish Git Review Room with folder artifact summary, selected artifact preview, advisory publish gate, commit plan, prepared message copy action and preserved Git commit/proposal branch mutations/selectors. Added V2 selectors (`publish-diff-summary`, `publish-preview-tabs`, `publish-gate-panel`, `publish-commit-plan`, `publish-commit-selected-btn`), explicit partial state for real Git diff data and desktop/mobile browser QA screenshots. No backend/API/schema/runtime contract/live-shell expansion changes.

### Slice ExecPlan - 16J UI V2 unit and accessibility coverage

Status: done.

Goals:
- [x] Add focused UI tests for V2 shell primitives: stage rail rendering/keyboard/collapse, right inspector priority/disabled action/empty sections and activity drawer controls/export/log states.
- [x] Cover critical empty/blocked/operator-visible states without adding hidden compatibility controls.
- [x] Keep tests fast and component-level where possible so they complement the heavier App integration tests from 16A-16I.

Non-goals:
- [x] No product UI redesign, backend API, schema, runtime artifact contract, CLI flag or workspace contract changes.
- [x] Do not expand release-facing live E2E scenarios, Playwright reason taxonomy or provider-live release gates.
- [x] Do not add hidden DOM controls solely for test compatibility.

Implementation notes:
- Add a small component test file around `StageRail`, `RightInspector` and `ActivityDrawer`.
- Reuse existing visible labels/selectors and verify accessibility-facing attributes (`aria-current`, `aria-pressed`, disabled primary actions, accessible drawer labels).
- Treat responsive collapse as the operator-facing rail collapse state; CSS breakpoint visual QA remains covered by implementation slices and live/Playwright slices.

Validation:
- [x] Focused UI component tests for shell primitives.
- [x] Existing App focused tests still pass.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Added fast component-level coverage for V2 shell primitives: stage rail accessible labels/keyboard/collapse state, right inspector blocked/attention priority with empty sections and evidence links, and activity drawer empty/populated/export/log-control states. Existing App-focused tests still pass; no product UI, backend/API/schema/runtime contract/live-shell changes.

### Slice ExecPlan - 16K Live E2E selector migration

Status: done.

Goals:
- [x] Migrate `ui/e2e/live-flow.spec.ts` assertions from legacy/raw-path-first surfaces to visible V2 operator selectors where those selectors now exist.
- [x] Keep the release-facing shell on `UI_E2E_SCENARIO=init-inspect` and preserve optional `UI_E2E_QA_SMOKE=1`.
- [x] Add contract-test coverage that the live flow uses V2 visible selectors and does not rely on hidden compatibility controls.

Non-goals:
- [x] Do not add new public live scenarios, provider-live release gates, failure reason classes or wrapper scripts.
- [x] Do not change backend API, schemas, runtime artifacts, CLI flags or workspace contracts.
- [x] Do not add hidden compatibility controls or hidden test-only DOM.

Implementation notes:
- Keep `scripts/frontend-live-e2e.sh` unchanged unless the public shell or result JSON changes.
- Update only Playwright selectors and Python script contract tests needed to protect the selector migration.
- Leave deeper Source -> Readiness -> Analysis -> Review -> Publish operator journey assertions for 16L.

Validation:
- [x] Focused Playwright/spec contract tests for selector migration.
- [x] UI typecheck/full UI tests through DoD.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Migrated `init-inspect` live Playwright assertions to visible V2 selectors (`source-repo-table`, readiness cards/runtime summary, analysis progress/timeline, activity table, review artifact explorer/evidence/citation panels) and added an explicit hidden-compatibility-control absence check. Added Python contract coverage for the selector migration. Public shell remains `UI_E2E_SCENARIO=init-inspect`; optional `UI_E2E_QA_SMOKE=1`, reason taxonomy, result JSON shape, backend/API/schema/runtime contracts and scripts remain unchanged.

### Slice ExecPlan - 16L Live E2E operator journey

Status: done.

Goals:
- [x] Extend the existing `init-inspect` Playwright flow to assert the V2 operator journey: Source -> Readiness -> Analysis -> Review -> Publish.
- [x] Assert operator-critical surfaces: run status, blockers/evidence refs, activity logs, runtime safety and Git publication path.
- [x] Capture durable stage screenshots for Source, Readiness, Analysis, Review and Publish diagnostics while preserving Playwright trace/screenshot/video failure artifacts.

Non-goals:
- [x] Do not add a new public `UI_E2E_SCENARIO`, provider-live release gate, reason taxonomy value or wrapper script.
- [x] Do not move Ask/domain-map/cancel/page-close into release readiness; Ask remains optional `UI_E2E_QA_SMOKE=1`.
- [x] Do not change backend API, schemas, runtime artifacts, CLI flags or workspace contracts.

Implementation notes:
- Keep `scripts/frontend-live-e2e.sh` result JSON shape stable; it already discovers `frontend-*.png` under `UI_E2E_OUTPUT_DIR`.
- Update the fake harness contract fixture to emit the same screenshot names the Playwright spec now captures on success.
- Leave deeper optional Ask/domain-map diagnostics for 16M.

Validation:
- [x] Focused frontend live E2E contract tests.
- [x] UI typecheck/full UI tests through DoD.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Extended `init-inspect` live Playwright flow into the required V2 operator journey Source -> Readiness -> Analysis -> Review -> Publish, with assertions for run status/logs, blockers/evidence refs, runtime safety and Git publication path. Added durable `frontend-source-desktop.png`, `frontend-readiness-desktop.png`, `frontend-analysis-desktop.png`, `frontend-review-desktop.png`, `frontend-publish-desktop.png` and `frontend-review-mobile.png` screenshot refs and updated the fake harness contract fixture accordingly. No new public scenario, reason taxonomy, release gate, wrapper script or backend/API/schema/runtime contract changes.

### Slice ExecPlan - 16M Live E2E Ask and domain-map diagnostics

Status: done.

Goals:
- [x] Keep async Ask checks inside optional `UI_E2E_QA_SMOKE=1` on `init-inspect`, with explicit answer/citation/confidence/read-only audit assertions.
- [x] Add deterministic domain-map diagnostics for render, edge navigation and proposal/evidence links through UI unit/fake-fixture coverage before any live shell expansion.
- [x] Add contract coverage that `ask-readonly` and `domain-map-diagnostic` are not public `scripts/frontend-live-e2e.sh` scenarios.

Non-goals:
- [x] Do not add new public live scenarios, provider-live release gates, reason classes or wrapper scripts.
- [x] Do not make Ask/domain-map required release readiness; Ask remains optional and domain-map remains deterministic/fake-fixture coverage.
- [x] Do not change backend API, schemas, runtime artifacts, CLI flags or workspace contracts.

Implementation notes:
- Update `ui/e2e/live-flow.spec.ts` only within the existing optional `UI_E2E_QA_SMOKE=1` block.
- Add focused `App.test.tsx` domain-map fixture assertions rather than provider-live domain-map execution.
- Update script contract tests, not the shell allowlist itself.

Validation:
- [x] Focused UI unit tests for domain-map diagnostics.
- [x] Focused frontend live E2E contract tests.
- [x] UI typecheck/full UI tests through DoD.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Tightened optional `UI_E2E_QA_SMOKE=1` assertions for citations panel and read-only QA audit path, added deterministic App-level domain-map diagnostics for edge navigation and proposal artifact drilldown, and added shell contract coverage that `ask-readonly`/`domain-map-diagnostic` remain absent from the public frontend live allowlist. Ask remains optional; domain-map remains deterministic/fake-fixture coverage; no new public scenario, reason taxonomy, release gate, wrapper script or backend/API/schema/runtime contract changes.

### Slice ExecPlan - 16N Docs/runbook sync

Status: done.

Goals:
- [x] Sync the active plan acceptance and progress log with the completed Console V2 implementation slices.
- [x] Update testing strategy docs so the required UI/live surfaces describe the V2 Source -> Readiness -> Analysis -> Review -> Publish journey, optional Ask smoke and deterministic domain-map coverage.
- [x] Update the release live E2E runbook so trusted-machine operators understand which V2 screenshots, diagnostics and non-release Ask/domain-map signals are evidence-only.
- [x] Confirm README, ARCHITECTURE and stakeholder docs remain consistent with the implemented UI behavior.

Non-goals:
- [x] No product code, backend API, schema, runtime contract, CLI flag, workspace contract or public live shell changes.
- [x] Do not add release-gate scenarios for Ask, domain map, cancel or page-close behavior.
- [x] Do not alter release verdict semantics, provider matrices, curated repo presets or matrix harness wrappers.

Implementation notes:
- Keep `UI_E2E_SCENARIO=init-inspect` as the only release-facing frontend live scenario.
- Keep `UI_E2E_QA_SMOKE=1` optional/non-release and document screenshots as diagnostics, not release readiness inputs.
- Keep domain-map coverage deterministic/fake-fixture until a separate owner-approved live-gate slice exists.

Validation:
- [x] Focused docs sync test: `go test ./internal/docsync`.
- [x] Focused frontend live E2E contract test.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-28: Synced Console V2 testing strategy and release live E2E runbook with the implemented Source -> Readiness -> Analysis -> Review -> Publish journey, evidence-only screenshot refs, optional `UI_E2E_QA_SMOKE=1`, deterministic domain-map diagnostics and unchanged release-facing `UI_E2E_SCENARIO=init-inspect`. README, ARCHITECTURE and stakeholder docs were checked as already consistent with the implemented UI behavior. No code/API/schema/runtime contract, public live shell, reason taxonomy, release verdict, provider matrix or wrapper-script changes.

### Slice ExecPlan - 16O Post-implementation UI/UX alignment audit

Status: done.

Goals:
- [x] Re-audit the implemented Console V2 against `docs/BACKLOG.md` Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and the saved PNG references.
- [x] Fix operator-flow mismatches found by rendered UI QA without changing backend API, schemas, runtime contracts, CLI flags or workspace contracts.
- [x] Keep Source, Readiness and Analysis primary paths aligned with the target `Source -> Readiness -> Analysis -> Review -> Publish` journey.

Non-goals:
- [x] Do not add approval persistence, Git diff API, new live E2E scenarios, provider-live release gates or new reason taxonomy.
- [x] Do not reintroduce hidden compatibility controls or test-only DOM.
- [x] Do not convert saved design PNGs into runtime product assets.

Implementation notes:
- Source inspector primary action must persist and validate sources, not only update the unsaved `workspace.yaml` draft.
- `Run first analysis` must route to Analysis mission control so operators can watch progress and logs before Review.
- Analysis shard/log table must show duration as available data or an explicit partial state.
- Toolchain-only live E2E precheck tests must not fall back to ambient `node`/`npm` when `ACP_NODE_TOOL_CANDIDATES_ONLY=1`; otherwise validation can recurse into full DoD instead of materializing expected precheck evidence.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server and saved design baseline.
- [x] Focused UI tests for first-run routing, source save semantics and Analysis shard duration.
- [x] Focused resolver and batch-precheck tests for hermetic live E2E node/npm toolchain evidence.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Re-audited Console V2 against Epic 16, the approved design baseline and saved PNG references. Fixed Source primary action so it saves and validates guided source settings in one step, kept raw `workspace.yaml` save inside Advanced, routed `Run first analysis` to Analysis mission control instead of empty Review, added explicit duration/partial-state column to the Analysis shard/log table, tuned the table layout for desktop/mobile visual QA, and fixed the node/npm resolver so `ACP_NODE_TOOL_CANDIDATES_ONLY=1` remains hermetic for live E2E precheck evidence. No backend/API/schema/runtime contract, live E2E public shell, reason taxonomy, Git diff API or approval persistence changes.

### Slice ExecPlan - 16P Post-audit UI/UX interaction hardening

Status: done.

Goals:
- [x] Re-audit the current Console V2 implementation against Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and rendered local UI evidence.
- [x] Fix confirmed operator-flow bugs in Source, Analysis, Ask and Publish without changing backend API, schemas, runtime contracts, CLI flags or workspace contracts.
- [x] Keep the first viewport aligned with the target Mission Control questions: workspace health, current pipeline status, blocker, reviewable evidence/logs and Git publication path.

Non-goals:
- [x] Do not add new API fields, approval persistence, Git diff APIs, live E2E scenarios, reason taxonomy, provider-live release gates or hidden compatibility controls.
- [x] Do not rework the approved visual baseline, saved PNG references or domain model.
- [x] Do not change release readiness semantics; provider-live checks remain manual trusted-machine gates.

Implementation notes:
- Source secondary action must be named as draft preview work, so it does not compete with the primary save-and-validate path.
- Analysis blocker review must keep the operator in Analysis and focus the failed-shard/log drilldown instead of navigating away from the run context.
- Ask inspector primary action must submit the visible Q&A form when Ask is already active, including normal validation for empty questions.
- Publish commit actions, including the inspector primary, must stay disabled while the publish gate has hard blockers such as missing generated artifacts.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Source, Analysis, Ask and Publish interactions.
- [x] Focused UI tests for Source wording, Analysis blocker focus, Ask inspector submit and Publish no-artifacts gate.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started post-audit interaction hardening after comparing the current branch with Epic 16, the target design baseline and rendered local UI screenshots.
- 2026-05-31: Completed 16P after rendered QA found and verified four interaction gaps: Source draft-preview wording, Analysis blocker focus, Ask inspector submit, and Publish hard-blocker disabled state. Verified no backend/API/schema/runtime contract, public live E2E shell, reason taxonomy or release-readiness changes.

### Slice ExecPlan - 16Q Post-audit recovery path hardening

Status: done.

Goals:
- [x] Re-audit Console V2 recovery actions against Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and rendered local UI evidence.
- [x] Fix confirmed operator-flow bugs where an empty-stage recovery action is shown but blocked, or readiness can be bypassed by a competing local action.
- [x] Keep Source -> Readiness -> Analysis -> Review recovery behavior aligned with the approved Mission Control design.

Non-goals:
- [x] Do not add backend API fields, schemas, runtime contracts, CLI flags, workspace contract changes, live E2E scenarios, reason taxonomy or hidden compatibility controls.
- [x] Do not change provider-live release readiness or trusted-machine gate semantics.
- [x] Do not add approval persistence, Git diff APIs or new publication mutations.

Implementation notes:
- Review empty state must make `Run analysis first` a real recovery action and route the operator into Analysis mission control.
- Readiness stage-level `Run first analysis` must require both workspace validation and local readiness pass, matching the right-inspector validate -> doctor -> analysis sequence.
- Analysis direct actions remain available from the Analysis stage for operators who intentionally jump there.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Review empty recovery and Readiness doctor gating.
- [x] Focused UI tests for Review recovery action routing and Readiness first-run gating.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started recovery path hardening after rendered QA showed Review empty-state primary action disabled and Readiness `Run first analysis` enabled before local doctor pass.
- 2026-05-31: Completed 16Q: Review empty-state `Run analysis first` now starts `init` and routes to Analysis mission control, while Readiness `Run first analysis` requires both valid workspace and successful local doctor. Verified with rendered desktop/mobile QA, focused UI tests and full DoD. No backend/API/schema/runtime contract, public live E2E shell, reason taxonomy or release-readiness changes.

### Slice ExecPlan - 16R Post-audit publication gate hardening

Status: done.

Goals:
- [x] Re-audit Publish Git Review Room against Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and the approved saved PNG baseline.
- [x] Fix confirmed publication-gate UI/UX bugs so commit and proposal-branch Git mutations are blocked consistently when hard publish blockers exist.
- [x] Keep operator recovery clear: publication blockers must be visible in the gate panel, right inspector and action disabled states.

Non-goals:
- [x] Do not add backend API fields, schemas, runtime contracts, CLI flags, workspace contract changes, Git diff APIs, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not change Charter/Baseline Git helper behavior; this slice only hardens the Publish room.
- [x] Do not add approval persistence or convert saved design PNGs into product runtime assets.

Implementation notes:
- Publish gate must treat generated artifact absence and shell hard blockers consistently for all publication mutations.
- The proposal branch action is a Git publication mutation in the Publish room, so it must not remain enabled while the gate is blocked.
- Gate copy must not describe checks as merely advisory if the UI blocks Git mutations.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for empty/blocked Publish states.
- [x] Focused UI tests for Publish commit/proposal-branch disabled states and allowed state after artifacts exist.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started publication gate hardening after code audit found Publish commit and proposal-branch actions were not gated consistently with the approved Git Review Room design.
- 2026-05-31: Completed 16R: Publish now carries shell hard blockers into the gate panel, disables both commit and proposal-branch Git mutations under hard blockers, removes misleading advisory gate copy, and makes disabled inspector primary actions visually non-primary. Verified with focused UI tests, rendered desktop/mobile QA, `git diff --check` and full DoD. No backend/API/schema/runtime contract, public live E2E shell, reason taxonomy, Git diff API or provider-live release gate changes.

### Slice ExecPlan - 16S Target UI reference alignment audit

Status: done.

Goals:
- [x] Re-audit all saved `docs/assets/ui-console-v2/*.png` references against the current rendered Console V2 screens.
- [x] Fix confirmed target-design mismatches that are UI-only and can be solved without backend/API/schema/runtime contract changes.
- [x] Keep the first screen of Review and Publish immediately reviewable when artifacts exist.

Non-goals:
- [x] Do not add approval persistence, Git diff APIs, new backend fields, schemas, runtime contracts, CLI flags, workspace contract changes, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not convert saved PNG references into runtime assets.
- [x] Do not change fake runtime artifact generation or provider behavior.

Implementation notes:
- Review Evidence should auto-load the first reviewable artifact when selected-run artifacts exist and no artifact is selected, matching the saved Review Evidence reference.
- Publish preview tabs must match the saved Publish reference and design contract: `Preview`, `Diff`, `Evidence`, `Changelog`; checklist remains in the side publish gate.
- The shared workflow rail should visually align with the saved reference screens as a dark compact ops rail while preserving selectors, keyboard behavior and responsive collapse.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Source, Review Evidence, Review Domain Map, Publish and shared shell.
- [x] Focused UI tests for Review auto-selection and Publish Changelog tab.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started target UI reference alignment after comparing the saved 9-screen PNG baseline with current rendered Source, Analysis, Review, Proposals, Ask and Publish screens.
- 2026-05-31: Completed 16S target alignment: Review Evidence auto-selects the first reviewable artifact, Publish uses `Preview`/`Diff`/`Evidence`/`Changelog`, raw proposal/changelog artifacts survive final-run-index normalization, and the shared workflow rail matches the saved dark ops-console references. No backend/API/schema/runtime/live-shell changes.

### Slice ExecPlan - 16T Stage contextual action and review defaults audit

Status: done.

Goals:
- [x] Re-audit the rendered Console V2 screens against the saved `docs/assets/ui-console-v2/*.png` references after 16S.
- [x] Keep each stage's right-inspector primary action contextual to that stage while still surfacing global blockers in the blocker panel.
- [x] Make Review and Proposals immediately reviewable when selected-run artifacts exist.

Non-goals:
- [x] Do not add Git diff APIs, approval persistence, new backend fields, schemas, runtime contracts, CLI flags, workspace contract changes, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not convert design PNGs into runtime assets or rewrite the existing shell architecture.
- [x] Do not change fake runtime artifact generation or provider behavior.

Implementation notes:
- Open-question blockers should remain visible in `Blockers`, but should not hijack Source, Charter, Proposals or Ask primary next actions.
- Review artifact explorer should be evidence-first (`reports/as-is`, coverage, findings, diagrams, model/domain outputs) with proposal/changelog artifacts later in the list.
- Proposals should auto-load the first proposal/ADR/RFC/checklist artifact when artifacts exist and no proposal artifact is selected, matching the saved Proposals review-room reference.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Source, Review Evidence, Proposals, Ask, Publish and shared shell.
- [x] Focused UI tests for contextual next action, Review explorer order/selection and Proposals auto-preview.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-05-31: Started 16T after rendered audit found Source/Charter/Proposals/Ask next actions were globally overridden by review findings, Proposals opened with an empty preview despite available proposal artifacts, and Review grouped proposal artifacts before evidence groups.
- 2026-05-31: Completed 16T: right-inspector next action is stage-contextual while blockers remain visible, Review explorer is evidence-first with visible selected artifact state, and Proposals auto-loads the first proposal artifact for immediate review. Verified with focused UI tests, rendered desktop/mobile QA, `git diff --check` and full DoD. Also restored local Python 3.9 compatibility for the matrix release contract test and matrix harness type hints so DoD imports and scripted matrix paths run cleanly, with no release behavior change. No backend/API/schema/runtime/live-shell changes.

### Slice ExecPlan - 16U Desktop shell and stage artifact-context audit

Status: done.

Goals:
- [x] Re-audit all saved `docs/assets/ui-console-v2/*.png` target references against rendered desktop/mobile Console V2 screens after 16T.
- [x] Keep the shared activity drawer visible in the desktop first viewport, matching the approved bottom ops-console drawer pattern.
- [x] Prevent proposal/changelog artifact selection from leaking into the Review Evidence first screen when operators return from Proposals, Ask or Publish.

Non-goals:
- [x] Do not add backend API fields, schemas, runtime contracts, CLI flags, workspace contract changes, Git diff APIs, approval persistence, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not convert saved PNG references into runtime assets or rewrite the shell architecture.
- [x] Do not remove access to proposal/changelog artifacts from the Review explorer; only restore stage-appropriate default context on stage entry.

Implementation notes:
- Desktop shell should behave like a fixed-height operator console: top health strip, rail/workbench/inspector and bottom activity drawer all visible, with workbench/inspector scrolling internally.
- Mobile should remain page-scroll friendly and must not introduce horizontal overflow.
- Review stage entry from another stage should re-open the preferred evidence artifact when the selected artifact belongs to Proposals/Changelog; user selection inside Review remains available afterward.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for Source, Readiness, Charter, Analysis, Review Evidence, Review Domain Map, Proposals, Ask, Publish and shared shell.
- [x] Focused UI tests for returning to Review from Proposals after proposal auto-preview.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-06-01: Started 16U after rendered reference audit found the desktop activity drawer below the first viewport and stale proposal artifact context leaking into Review Evidence after cross-stage navigation.
- 2026-06-01: Completed 16U: wide desktop now behaves as a fixed-height operator console with rail, workbench, inspector and activity drawer visible in the first viewport; 901-1180px keeps the same shell without clipping the inspector or rail; Review stage entry restores evidence-first artifact context after Proposals/Changelog navigation. Verified against all saved target references with rendered desktop/mobile QA, focused UI tests, `git diff --check` and full DoD. No backend/API/schema/runtime/live-shell changes.

### Slice ExecPlan - 16V Narrow desktop workbench overflow audit

Status: done.

Goals:
- [x] Re-audit current Console V2 against `docs/BACKLOG.md` Epic 16, `docs/UI_CONSOLE_V2_DESIGN.md` and all saved `docs/assets/ui-console-v2/*.png` references after 16U.
- [x] Fix confirmed narrow desktop UI overflow where the fixed three-column shell leaves the central workbench too narrow for Review/Proposals/Publish stage grids.
- [x] Preserve wide desktop first-viewport shell behavior and mobile page-scroll behavior.

Non-goals:
- [x] Do not add backend API fields, schemas, runtime contracts, CLI flags, workspace contract changes, Git diff APIs, approval persistence, live E2E scenarios, reason taxonomy or provider-live release gates.
- [x] Do not convert saved PNG references into runtime assets or rewrite the shell architecture.
- [x] Do not hide operator-facing controls or restore hidden compatibility DOM.

Implementation notes:
- Keep the shared rail, inspector and activity drawer visible in the fixed desktop shell.
- At narrow desktop widths, stage-specific workbench grids should collapse inside the central work area before content clips or creates confusing horizontal work-area scroll.
- Shared section heading rows should wrap controls instead of forcing action buttons outside their card.

Validation:
- [x] Rendered desktop/mobile UI QA against the local fake server for all 9 target screens plus narrow desktop breakpoints.
- [x] Focused UI/type tests for changed frontend behavior where applicable.
- [x] `git diff --check`.
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`.

Progress log:
- 2026-06-01: Started 16V after rendered breakpoint audit found Review evidence controls clipping and central work-area horizontal overflow at 901px while the shared shell itself remained visible.
- 2026-06-01: Completed 16V: stage workbench grids now adapt before fixed desktop shells get too narrow, shared heading rows wrap controls, and Source/Readiness/Charter/Analysis/Review/Proposals/Ask/Publish render without document or central work-area horizontal overflow at 1536px, 1440px, 1366px, 1180px, 1100px, 1024px and the 901px desktop boundary. Mobile/tablet page-scroll behavior remains unchanged below 901px, and no backend/API/schema/runtime/live E2E contract changed.

### Files expected to change
- `ui/src/App.tsx`, `ui/src/styles.css`, `ui/src/components/*`, `ui/src/hooks/*`, `ui/src/lib/*`
- `ui/src/App.test.tsx`, `ui/e2e/live-flow.spec.ts`, `ui/playwright.live.config.ts` only if artifact output behavior changes
- `scripts/frontend-live-e2e.sh` only if the public live shell changes; `scripts/frontend-status-reasons.sh` only if a separate classification slice changes reason taxonomy; `scripts/tests/frontend_live_e2e_contract_test.py`
- `docs/UI_CONSOLE_V2_DESIGN.md`, `docs/BACKLOG.md`, `docs/PLANS.md`
- implementation docs: `README.md`, `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md`, `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

### Acceptance criteria
- [x] All 8 rail stages render and route to the V2 surfaces.
- [x] First viewport shows workspace readiness, current run status, blocker/next action and evidence/Git path where relevant.
- [x] Analysis keeps run timeline, failed shard details and logs adjacent.
- [x] Review supports evidence workbench and domain map without raw-path-first review.
- [x] Ask is explicit read-only Q&A and shows confidence/citations/unresolved assumptions.
- [x] Publish gate shows changed folders, blocker checklist, commit plan and proposal branch.
- [x] Each screen uses existing APIs/data hooks or renders an explicit partial/empty state; no implicit backend contract changes.
- [x] UI tests cover stage navigation, inspector priority and drawer behavior.
- [x] Live E2E `init-inspect` validates Source -> Readiness -> Analysis -> Review -> Publish and captures diagnostics on failure.
- [x] Async Ask is covered by optional `UI_E2E_QA_SMOKE=1` and deterministic tests; domain-map checks are covered by fake-fixture/unit/component diagnostics before any live shell expansion.
- [x] `frontend-e2e-result.json` still records scenario, reason, run id, last run status/error/current step and diagnostic refs.
- [x] Playwright trace/screenshot/video artifacts remain available for failed V2 live E2E stages.
- [x] Existing reason taxonomy remains additive-compatible with release report aggregation.
- [x] Hidden compatibility controls remain absent.

### Risks
- Dense shell can regress responsive layout; visual QA must include desktop and narrow viewport checks.
- Selector migration can weaken release diagnostics if compatibility aliases are dropped without E2E updates.
- Domain map may overpromise if model/edge data is sparse; empty and partial states must be explicit.

### Progress log
- 2026-05-27: Approved unified 9-screen design vision and fixed design baseline, visual references, data-source map, backlog, ExecPlan and V2 live E2E contract without product code changes.
- 2026-05-27: Rebased on `origin/main` `3aa458a`; latest main already removed compatibility controls, compacted activity/artifact surfaces, added optional `UI_E2E_QA_SMOKE=1` and screenshot refs, so V2 backlog was corrected to keep release-facing live frontend on `init-inspect` only.
- 2026-05-28: Continuous backlog queue normalized this plan to the first active workstream; next implementation slice is `16A UI V2 shell foundation`.
- 2026-05-28: Completed `16A UI V2 shell foundation`: shared stage status model, top strip runtime/Git metadata, V2 inspector selectors, Git publication panel, rail keyboard navigation and drawer a11y baseline. No backend/API/schema/live-shell changes.
- 2026-05-28: Completed `16B Source + Readiness consolidation`: Source now has a repo table with explicit advanced-only analysis scope state, Readiness has workspace/repo/runtime/permission/artifact summary cards plus a compact runtime profile summary. No backend/API/schema/live-shell changes.
- 2026-05-28: Completed `16C Charter workbench`: Charter now shows wizard summary, domain/team card overview, baseline prompt bundle status, explicit partial states for missing cards, and a no-404 empty editor selection before existing artifact editor/Git helper actions. No backend/API/schema/live-shell changes.
- 2026-05-28: Completed `16D` through `16N`, finishing Analysis mission control, Review evidence/domain map, Proposals review room, Ask read-only console, Publish gate, UI coverage, live E2E selector/operator-journey migration, optional Ask/domain-map diagnostics and testing/runbook sync. Epic 16 implementation acceptance is complete; remaining item is owner review/merge/archive bookkeeping.

### Plan ID
EP-20260527-live-e2e-ui-ux-operator-flow

### Context
После breaking simplification live E2E scripts больше не генерируют pseudo black-box reasoning, а operator/SWE-agent отвечает за assessment поверх evidence. Post-rebase UI walkthrough показал оставшиеся legacy compatibility controls, слабое покрытие async Ask UX, слишком шумный activity drawer и отсутствие durable screenshot refs в frontend live evidence.

### Goals (must have)
- [x] Удалить hidden compatibility UI controls/testids из operator console.
- [x] Добавить optional non-release `UI_E2E_QA_SMOKE=1` в live Playwright flow без включения в canonical release readiness.
- [x] Сохранять frontend screenshots как diagnostic evidence refs в `frontend-e2e-result.json`.
- [x] Сделать activity drawer и artifact/diagram links компактнее и сканируемее.
- [x] Уточнить next-action wording для blockers/findings/release blockers.
- [x] Обновить operator assessment template, live gate skill и runbook.
- [x] Обновить unit/contract/live tests и пройти focused checks plus DoD.
- [ ] После owner review/merge перенести план в архив.

### Non-goals
- [x] Не менять `release_verdict_*` contract и `verify-release-verdict.py`.
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`.
- [x] Не включать Ask/UX smoke в canonical release matrices.
- [x] Не делать screenshots или operator UX assessment источником release readiness.

### Approach
1) Remove compatibility DOM and migrate tests to stage rail/operator-facing controls.
2) Extend live Playwright init-inspect with optional QA smoke and deterministic screenshots.
3) Add screenshot refs to frontend result JSON and keep them evidence-only.
4) Compact operator console evidence surfaces while preserving logs/artifact access.
5) Sync runbook/skill/template/docs and focused tests.

### Files expected to change
- `ui/src/components/*`
- `ui/src/App.test.tsx`
- `ui/e2e/live-flow.spec.ts`
- `scripts/frontend-live-e2e.sh`
- `scripts/tests/frontend_live_e2e_contract_test.py`
- `.agents/skills/e2e-live-gate/SKILL.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/templates/LIVE_E2E_OPERATOR_ASSESSMENT.md`

### Acceptance criteria
- [x] Legacy compatibility controls/testids are absent from the UI shell.
- [x] Default live frontend flow remains `init-inspect` without QA smoke.
- [x] `UI_E2E_QA_SMOKE=1` verifies Ask answer/citations/context/runtime links.
- [x] Frontend result JSON includes screenshot refs when screenshots are produced.
- [x] Mobile first viewport does not expose legacy labels.
- [x] Focused Python/UI tests pass.
- [x] `make contracts`, `make test`, `make lint`, `make build`.

### Risks
- Removing legacy testids requires broad UI test migration.
- Screenshot refs must remain diagnostic metadata and must not leak into release verdict semantics.
- Activity drawer compaction must not hide failure triage evidence from operators.

### Progress log
- 2026-05-27: Started UI/UX live E2E operator-flow hardening slice.
- 2026-05-27: Implemented compatibility DOM removal, optional Ask UX smoke, screenshot evidence refs, compact activity/artifact UX, docs sync, full DoD, and fake-runtime UX smoke with screenshots.
- 2026-05-28: Queue normalization classified this plan as implementation-complete/archive-only; do not pick it as the next engineering slice.

### Plan ID
EP-20260526-async-runtime-backed-ask

### Context
Текущий `POST /api/qa/ask` и `acp qa` остаются compatibility/beta baseline: deterministic workspace search в `internal/qa`, без runtime provider. Target architecture для UI Ask меняется на async agentic Q&A run: ACP собирает deterministic context pack из существующих workspace artifacts, запускает выбранный runtime provider с ролью `system-analyst-qa`, валидирует `qa-answer.json` и показывает результат в UI polling flow.

### Goals (must have)
- [x] Добавить Q&A runtime family `qa.ask` с agent role `system-analyst-qa`, prompt pack `skills/prompt-packs/qa.md` и write scope только `reports/taskruns/<run_id>/qa/`.
- [x] Добавить async API `POST /api/qa/runs`, `GET /api/qa/runs/{run_id}`, `GET /api/qa/runs?limit=...`; оставить `POST /api/qa/ask` как legacy deterministic endpoint.
- [x] Добавить `runtime.profile.steps.qa.provider` в manifest/schema/validator/API runtime profile.
- [x] Ввести `qa-answer.json` contract/schema and validation; сохранять `context-pack.json` и `runtime-execution.json` рядом с answer для audit/debug.
- [x] Context pack строится deterministic из canonical workspace artifacts and imported docs; `reports/taskruns/**` исключается из evidence corpus.
- [x] Fake runtime умеет выполнять `qa.ask` для required CI/local smoke.
- [x] Headless runtime получает artifact-only QA prompt через shared provider engine and validates `qa-answer.json`.
- [x] UI Ask stage использует async submit + polling and shows run status/provider/answer/citations/unresolved/confidence.
- [x] Обновить README, architecture/spec docs, stakeholder docs and schema appendix под target/current split.
- [ ] После owner review/merge перенести план в архив.

### Non-goals
- [x] Не удалять `POST /api/qa/ask` и `acp qa` в этом slice.
- [x] Не менять init/refresh artifact schemas beyond adding QA answer schema and workspace qa provider field.
- [x] Не добавлять hosted/security hardening.
- [x] Не мутировать source repos или canonical architecture outputs из QA run.

### Approach
1) Reuse orchestrator run history/log infrastructure with `pipeline="qa"` while keeping QA runs out of the normal Analysis run list.
2) Build `context-pack.json` before runtime execution from `charter/cards`, `model`, `reports/as-is`, `reports/findings`, `reports/coverage`, `proposals`, `reports/changelog`, and configured docs imports.
3) Route `qa.ask` through shared runtime task execution and providercommon artifact validation.
4) Return structured QA run status over `/api/qa/runs/*`; UI polls that endpoint and links to existing run logs/artifacts.
5) Keep legacy deterministic QA service as retriever/context builder and compatibility fallback.

### Files expected to change
- `internal/orchestrator/*`
- `internal/api/server.go`
- `internal/qa/*`
- `internal/runtime/*`
- `internal/workspace/manifest.go`
- `internal/runtimeprofile/patch_service.go`
- `schemas/workspace.schema.json`
- `schemas/qa-answer.schema.json`
- `ui/src/components/StagePanels.tsx`
- `ui/src/lib/qaApi.ts`
- `ui/src/App.test.tsx`
- docs/spec/README/stakeholder/backlog/schema appendix docs

### Acceptance criteria
- [x] Schema validation accepts `runtime.profile.steps.qa.provider` and rejects invalid providers.
- [x] Fake Q&A run writes valid `reports/taskruns/<run_id>/qa/qa-answer.json`.
- [x] Context pack excludes `reports/taskruns/**`.
- [x] Async API covers start/status/list while legacy `POST /api/qa/ask` still works.
- [x] UI Ask submit creates a Q&A run and renders succeeded answer state.
- [x] Full DoD completed: `make contracts`, `make test`, `make lint`, `make build`.

### Risks
- QA runs share the existing single active/pending run queue; future UX may need a dedicated lightweight queue if operators ask questions during long init runs.
- Headless answer quality depends on provider following `qa-answer.json`; fake remains deterministic baseline, but live QA should get focused smoke coverage before release claims.

### Progress log
- 2026-05-26: Implemented async QA run family/API/schema/fake runtime/UI polling, context-pack citation validation, docs/test sync, full DoD, and fake async QA smoke.

### Plan ID
EP-20260526-live-e2e-operator-blackbox-simplification

### Context
Live E2E/release gate накопил двусмысленные surfaces: harness сам пишет `blackbox_e2e_steps_*` как будто это operator reasoning, non-release diagnostic matrices пишут `release_verdict_*`, planner публикует JSON/Markdown как будто это second-order API, а release strict verdict включает stubbed frontend cancel smoke. Breaking compatibility допустима: лучше удалить старую логику сразу и оставить один machine-verifiable release gate плюс отдельный operator assessment поверх evidence.

### Goals (must have)
- [x] Удалить internal black-box evaluator helper и генерацию `blackbox_e2e_steps_*`.
- [x] Развести release artifacts (`release_verdict_*`) и non-release artifacts (`matrix_result_*`).
- [x] Ужесточить `verify-release-verdict.py`, чтобы он принимал только canonical release-mode verdict.
- [x] Оставить `live-e2e-plan.py` shell-only direct command printer.
- [x] Убрать stubbed frontend cancel из release gate и public live frontend harness.
- [x] Обновить runbook/skill/testing docs и добавить шаблон operator assessment.
- [x] Обновить regression tests под breaking protocol.
- [x] Сделать planner shell output явным про diagnostic/release mode, provider/run scope, frontend skip и fake/headless init/refresh цикл.
- [x] Починить `make test-stress`, чтобы zero-match Go test pattern не давал ложный зелёный сигнал.
- [x] Rerun full DoD on a host with exact Node.js `22.21.1`.
- [ ] После owner review/merge перенести план в архив.

### Non-goals
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`.
- [x] Не менять canonical release matrices или curated repos для обхода host prerequisites.
- [x] Не делать operator assessment источником release readiness.

### Approach
1) Удалить evaluator helper и все script-authored black-box decision calls из batch/matrix harness.
2) Изменить matrix synthesis: release mode пишет только release verdict, non-release mode пишет neutral matrix result без `release_state`/`release_contract`.
3) Ужесточить verifier и planner public surface.
4) Убрать frontend cancel strict path из release aggregation и live shell harness; оставить frontend release check на init/artifact inspection.
5) Синхронизировать docs/tests и template для ручного assessment.

### Files expected to change
- `scripts/full-run-batch-matrix.sh`
- `scripts/full-run-batch.sh`
- `scripts/frontend-live-e2e.sh`
- `scripts/live-e2e-plan.py`
- `scripts/verify-release-verdict.py`
- `Makefile`
- `scripts/tests/*`
- `internal/orchestrator/run_lifecycle_test.go`
- `.agents/skills/e2e-live-gate/SKILL.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/templates/LIVE_E2E_OPERATOR_ASSESSMENT.md`

### Acceptance criteria
- [x] Focused script tests updated and passing.
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] Canonical release gate is not run unless trusted-host prerequisites are satisfied.

### Risks
- Large shell harness changes can break hermetic tests; mitigate by keeping execution flow unchanged except removed surfaces.
- Historical docs may mention old artifacts; sync source-of-truth docs first and ignore archive snapshots unless tests read them.

### Progress log
- 2026-05-26: Started breaking simplification implementation from audit follow-up.
- 2026-05-26: Implemented breaking simplification and updated focused script suite.
- 2026-05-26: Installed exact local Node.js `22.21.1`, fixed frontend health-start result JSON, fixed async API test timeout flake under live quality gate, updated skipped-frontend report wording, completed full DoD (`make contracts`, `make test`, `make lint`, `make build`), and passed codex-code non-release diagnostic `smoke-tiny-bank-20260526T195844Z`.
- 2026-05-27: Added explicit planner comments for diagnostic/release scope and fixed `make test-stress` with a zero-match guard plus async debounce regression coverage.
- 2026-05-27: Re-ran direct codex-code non-release diagnostic `smoke-tiny-bank-20260527T114046Z`; it wrote `matrix_result_*` only, verifier rejected the diagnostic payload, and reports surfaced runtime pressure (`stall_count=5`, `post_artifact_stalls=5`, `quality_alerts=2`) for operator review.

### Plan ID
EP-20260525-frontend-live-e2e-diagnostics

### Context
`release-fast-20260525T104842Z` proved the runtime permission slice did not break trusted provider args or backend hard gates, but exposed weak frontend live E2E classification. `qwen-code` frontend init passed, while `claude-code` failed with `Target page, context or browser has been closed` while the run was still active in `init.step1.collect`, and `codex-code` failed with API `ECONNREFUSED` while the fresh init run was still in `init.step0.constitution`. Both collapsed into `playwright_failed`, which blocks useful triage and makes it unclear whether the issue is browser lifecycle, API/server lifecycle, or product UI.

### Goals (must have)
- [x] Keep Playwright as the canonical CLI/release-gate harness; Browser/Chrome MCP remains manual diagnostic only.
- [x] Split frontend live failure reasons into `browser_closed`, `api_unreachable`, `server_exited`, `active_run_timeout`, and fallback `playwright_failed`.
- [x] Split post-merge frontend live backend-run failures into `runtime_run_failed` instead of collapsing them into fallback `playwright_failed`.
- [x] Make long backend polling independent from the browser page object.
- [x] Persist frontend result diagnostics: server PID/exit code, post-failure health, run id, last run status/current step, and diagnostic refs.
- [x] Add stub regression tests for the new frontend classifications.
- [x] Keep API request polling independent from page lifecycle inside the init-inspect flow; the old page-close live scenario is superseded by EP-20260526.
- [ ] After merge, run focused frontend init diagnostics for `claude-code` and `codex-code`, then rerun canonical `release fast` if both focused checks pass.

### Non-goals
- [x] Do not change canonical matrices, timeout profiles, provider contracts, permission behavior, or public HTTP API.
- [x] Do not replace Playwright release-gate acceptance with MCP automation.
- [x] Do not fix any discovered `acp serve` lifecycle bug in this slice; classify it first.

### Approach
1) Extend frontend reason allowlist and batch/report aggregation coverage.
2) Harden `scripts/frontend-live-e2e.sh` post-failure diagnostics around `acp serve` PID, `/api/health`, Playwright log signatures, and run-history/API state.
3) Refactor `ui/e2e/live-flow.spec.ts` to use independent API request polling and promise-based sleeps for long polling.
4) Add shell-stub tests that simulate server exit, API unreachable, browser closed, and active timeout without live providers.
5) Sync testing/runbook architecture docs with the narrower failure taxonomy.

### Files expected to change
- `scripts/frontend-live-e2e.sh`
- `scripts/frontend-status-reasons.sh`
- `ui/e2e/live-flow.spec.ts`
- `scripts/tests/*`
- `docs/PLANS.md`
- `docs/TESTING_STRATEGY.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/ARCHITECTURE.md`

### Acceptance criteria
- [x] `python3 -m unittest scripts.tests.frontend_live_e2e_contract_test`
- [x] relevant batch/report script tests
- [x] UI unit/build checks
- [x] frontend live shell exposes only `init-inspect`; page-close/cancel coverage moved out of live gate
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`

### Risks
- Shell lifecycle checks must not kill or wait on a still-running `acp serve` before cleanup.
- New reason codes must remain additive so historical result JSON and report aggregation keep working.

### Progress log
- 2026-05-25: Started implementation after stopping `release-fast-20260525T104842Z`; evidence showed independent frontend init failures for Claude browser lifecycle and Codex API/server reachability.
- 2026-05-25: Implemented additive reason taxonomy, post-failure diagnostics, independent API polling, stub regression coverage, and docs sync; local DoD passed.
- 2026-05-26: Follow-up release-fast triage found a stable `claude-code` init failure where Playwright correctly observed the backend run had already failed with `runner_unavailable`; adding `runtime_run_failed` classification plus `last_run_error_code` diagnostics.

### Plan ID
EP-20260525-proven-arch-ui-console

### Context
Текущий React UI отражает internal modules (`Setup / Baseline / Runs / Results / Settings`) и начинается с hero-блока, из-за чего пользователь видит админку настроек вместо рабочего flow Proven Arch. Нужно перестроить surface в stage-based console без изменения backend/API contracts.

### Goals (must have)
- [x] Заменить hero/top tabs на Proven Arch console shell: top bar, stage rail, center work area, right inspector, bottom activity drawer
- [x] Разложить существующие setup/runtime/run/results/git surfaces по stages `Source / Readiness / Charter / Analysis / Review / Proposals / Ask / Publish`
- [x] Подключить существующий read-only `POST /api/qa/ask` как UI stage `Ask` (historical slice; superseded by EP-20260526 async `/api/qa/runs` target)
- [x] Добавить derived stage status, next action, blockers, evidence refs и runtime/workspace health
- [x] Обновить CSS под light operator-console style без backend изменений
- [x] Обновить UI tests, live E2E selectors и пользовательские docs
- [x] Выполнить доступные проверки slice
- [ ] Перенести этот ExecPlan в архив после owner review/merge

### Non-goals
- [ ] Не менять schemas, runtime artifact contracts, Go API wire shapes или CLI flags
- [ ] Не добавлять hosted/security/compliance UX
- [ ] Не менять pipeline semantics или provider execution policy

### Approach
1) Добавить UI-only view model types и Q&A API wrapper.
2) Вынести reusable console components (`AppShell`, top bar, rail, inspector, activity drawer, small primitives).
3) Refactor `App.tsx` так, чтобы existing hooks остались источником данных, а central content переключался по stage.
4) Добавить stage panels поверх текущих API/hook actions и сохранить стабильные `data-testid` для core flows.
5) Синхронизировать README/ARCHITECTURE и тесты с stage-based console.

### Files expected to change
- `ui/src/App.tsx`, `ui/src/styles.css`, `ui/src/components/*`, `ui/src/lib/*`
- `ui/src/App.test.tsx`, `ui/e2e/live-flow.spec.ts`
- `README.md`, `docs/ARCHITECTURE.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] Stage rail renders all 8 stages and switches central content
- [x] Readiness blockers/next actions reflect workspace validation, doctor, run errors and pending permissions
- [x] Review stage shows coverage/questions/artifacts/diagrams from existing artifact APIs
- [x] Ask stage originally called `/api/qa/ask` and rendered answer/citations/confidence; current target is async `/api/qa/runs` under EP-20260526
- [x] Existing first-run, runtime settings, logs, diagrams, baseline editor and git helper coverage remain covered
- [x] `npm run typecheck --prefix ui`
- [x] `npm test --prefix ui`
- [x] `make lint`
- [x] `make build`

### Risks
- Existing UI tests have many tab-specific selectors; preserve important test IDs where practical and update navigation selectors deliberately.
- Dense console layout can overflow on small viewports; responsive collapse must be part of the slice.

### Progress log
- 2026-05-25: Started implementation from approved UI/UX plan.
- 2026-05-25: Implemented console shell, stage panels, Q&A UI, docs/tests/e2e updates, embedded UI rebuild and DoD checks (`make contracts`, `make test`, `make lint`, `make build`).
- 2026-05-25: Follow-up UX audit against the accepted mockup found lower visual fidelity in the top bar/rail density and hidden compatibility controls still participating in keyboard focus; applying a focused UI polish and a11y fix without backend contract changes.
- 2026-05-25: Browser QA follow-up fixed rail a11y duplicate labels, unlabeled advanced manifest textarea, and initial optional Charter artifact 404 console noise by lazy-loading Charter content only when the stage opens.
- 2026-05-25: Second UX pass tightened Source density, added top-bar status icons/server status, and made bootstrap open `Review` automatically when a completed run already has artifacts.
- 2026-05-25: Final browser QA found stale Mermaid syntax-error SVGs caused by rendering `.mmd` artifacts while text content still said `Loading...`; fixed diagram loading guard/cleanup and changed inspector next-action status from `ready` to `attention` when warning blockers exist.

### Plan ID
EP-20260518-live-e2e-blackbox

### Context
Live E2E должен стать black-box operator flow: план шага, прямой harness/UI/API вызов, evidence inspection, classification, next decision. Official release readiness остаётся только в release-mode `reports/release_verdict_<matrix-id>.json`, проверяемом `scripts/verify-release-verdict.py`. EP-20260526 superseded the earlier machine-generated step-report approach: scripts now produce facts/verdicts only, and operator assessment is separate.

### Goals (must have)
- [x] Обновить live E2E skill и release runbook под обязательный step-by-step black-box evaluator protocol
- [x] Удалить durable machine-authored step evidence из batch/matrix harness; reasoning layer остаётся operator-owned
- [x] Сделать explicit layering: live-e2e skill -> local manual-live-e2e workflow -> direct public harness commands -> ACP runtime/provider/UI evidence, без GitHub Actions live workflow
- [x] Оставить `scripts/full-run-batch-matrix.sh` direct top-level release harness
- [x] Перенести backend-cycle logic за `scripts/full-run-batch.sh` в internal helper и удалить публичный legacy entrypoint
- [x] Удалить legacy live E2E matrices/docs/tests and references without compatibility shims
- [x] Выполнить DoD checks после implementation
- [ ] Когда owner запросит pre-release validation, выполнить trusted-machine live gate через новый black-box protocol и сохранить verifier-backed verdict evidence

### Non-goals
- [x] Не менять release verdict contract
- [x] Не запускать trusted live E2E в рамках implementation slice
- [x] Не менять runtime artifact schemas или rejection tests for old runtime payloads

### Approach
1) Удалить script-authored pseudo-reasoning reports из batch/matrix harness.
2) Оставить release readiness только в verifier-backed release verdict; non-release matrices пишут neutral matrix result.
3) Перевести старый backend-cycle в non-public helper, вызываемый только из `scripts/full-run-batch.sh`.
4) Переписать skill/runbook/testing docs под новый protocol и удалить legacy live E2E surfaces.
5) Обновить docsync/script tests так, чтобы они требовали operator-owned assessment wording и отклоняли старые live E2E references.

### Files expected to change
- `.agents/skills/e2e-live-gate/SKILL.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`
- `docs/BACKLOG.md`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/internal/live-e2e-backend-cycle.sh`
- deleted internal live E2E evaluator helper
- `internal/docsync/docsync_test.go`
- `scripts/tests/*`
- legacy live E2E files removed from `docs/`, `examples/`, and `scripts/`

### Acceptance criteria
- [x] `go test ./internal/docsync`
- [x] `python3 -m unittest discover -s scripts/tests -p '*_test.py'`
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`

### Risks
- Shell harness changes can affect long trusted-machine runs; targeted tests must cover the new step-report artifacts and direct harness integration.
- Active docs must distinguish removed live E2E surfaces from unrelated runtime contract rejection tests that intentionally mention old payload shapes.

### Progress log
- 2026-05-18: Started implementation. Added black-box step reports, moved backend-cycle behind batch harness, and began docs/test cleanup. Trusted live E2E was not run.
- 2026-05-18: Local verification passed: `go test ./internal/docsync`, `python3 -m unittest discover -s scripts/tests -p '*_test.py'`, `make contracts`, `make test`, `make lint`, `make build`. Make targets used `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node22-wrapper/bin` because Homebrew `node@22` needed older simdjson/simdutf dylib paths on this host.

### Plan ID
EP-20260509-v011-hardening-release

### Context
`v0.1.1` is now published as a beta prerelease. The final tag was cut from `7548fdc4` after owner-approved no-gate release handling, GitHub main CI, release workflow success, and install smoke checks. Fresh trusted-machine `release-fast` was intentionally skipped, so this plan records the release but does not claim canonical `RELEASE READY` status.

### Goals (must have)
- [x] Keep README aligned with the actual local-first install/run path and public release status across `v0.1.1` publication
- [x] Keep README user-facing by removing internal live E2E/release-gate/runbook navigation and making fake/live onboarding standalone
- [x] Sync `docs/INSTALL.md` provider command wording and post-publication latest-release status
- [x] Move user-facing hardening notes from `Unreleased` into `CHANGELOG.md` entry `v0.1.1`
- [x] Preserve `.goreleaser.yml` prerelease behavior for the next beta release
- [x] Run local DoD and release-prep smoke checks
- [ ] Optional follow-up: run trusted-machine release gate through direct `scripts/full-run-batch-matrix.sh` invocation and record `reports/release_verdict_<matrix-id>.json` if canonical release-ready status is needed later
- [x] After the GitHub release is published, update public latest-release docs from `v0.1.0` to `v0.1.1`

### Non-goals
- [x] Do not change runtime contracts, schemas, CLI flags, or public API
- [x] Do not add wrapper scripts around the release matrix harness
- [x] Do not edit canonical release matrices or curated repo files to fit the current host
- [x] Do not mark release readiness as passed without verifier-backed `PASS`

### Approach
1) Prepare release docs on a dedicated branch with README and changelog aligned to the release plan.
2) Run local required checks and smoke checks against fake runtime and public install path.
3) Execute trusted-host live release gate only from a clean committed tree that satisfies canonical path/provider prerequisites when a canonical release-ready verdict is required.
4) For the owner-approved no-gate path, publish `v0.1.1` as a prerelease through existing GoReleaser config after release metadata is explicit and main CI is green.
5) Keep latest-release docs at `v0.1.1` after publication.

### Files expected to change
- `README.md`
- `CHANGELOG.md`
- `docs/PLANS.md`
- `docs/INSTALL.md` for provider command wording and post-publication latest-release values
- `scripts/write-batch-preflight.py`, `scripts/e2e_report_classifiers.py`, `scripts/e2e_batch_report.py`, and related tests for release-fast provider readiness/diagnostic alignment
- `internal/runtime/{qwencode,claudecode,codexcode,providercommon}` tests and adapters for qwen/claude pre-artifact stall policy and raw diagnostics

### Acceptance criteria
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] `goreleaser check`
- [x] Public install smoke against current `main/install.sh`
- [x] Source fake walkthrough smoke on a temporary local Git repo
- [x] UI/API smoke against local `acp serve`
- [x] README user-facing guard: no internal live E2E/runbook/matrix/verdict navigation
- [x] README relative links check
- [x] README standalone fake quality check includes `serve --auto-init --dry-run` before `run`
- [x] README provider example check with `ACP_CLAUDE_CMD=claude`
- [ ] Trusted release verdict JSON verified with `scripts/verify-release-verdict.py`

### Risks
- Current local host may not satisfy trusted-machine prerequisites for canonical release slices.
- `v0.1.1` docs may claim latest public release after publication, but must not imply canonical `RELEASE READY` without a verifier-backed release verdict.
- Owner/admin GitHub residual tasks remain manual and must not be misrepresented as completed release evidence.

### Progress log
- 2026-05-09: Started `v0.1.1` hardening/onboarding release prep. README rewrite already exists in the worktree; changelog moved to `v0.1.1`. Trusted live release gate not run yet.
- 2026-05-09: Local release-prep verification passed: `make contracts`, `make test`, `make lint`, `make build`, `goreleaser check` via `go run`, public installer smoke, source fake walkthrough smoke, and local UI/API smoke. Canonical live release gate remains blocked until a clean committed tree and complete trusted-host curated checkout set are available; current host is missing `/tmp/provenarch-live-e2e/posthog/posthog` for `release long`.
- 2026-05-09: Refined README as standalone user onboarding artifact: removed public navigation to internal live E2E/release-gate/runbook surfaces, added explicit provider ID vs executable command wording, documented `ACP_CLAUDE_CMD=claude` live smoke, and synced `docs/INSTALL.md` wording without changing latest public release status.
- 2026-05-09: Fixed README quality-check flow to be standalone for a fresh workspace by adding `serve --auto-init --dry-run` before the fake `run` command.
- 2026-05-09: README-only live E2E audit: README supports install, fake walkthrough, and single-provider Claude live smoke, but intentionally contains no matrix/runbook/verdict route; a README-only user cannot complete the full live E2E/release gate. Local provider binaries `qwen`, `claude`, and `codex` are present, `/tmp/provenarch-live-e2e` is writable, but the canonical checkout set is incomplete (`bank-of-anthos`, `posthog`, `ftgo`, and `sentry-ecosystem` missing).
- 2026-05-09: Started README + release-fast live E2E candidate slice. Target is current candidate fixes, not published `v0.1.0`: close clean-`HOME` README doctor gap, verify fake README walkthrough including diagram preview and QA, then run canonical `release fast` from a clean committed worktree through direct `scripts/full-run-batch-matrix.sh`.
- 2026-05-09: First canonical `release fast` attempt on commit `99de15b` stopped before backend runs with `operational_host_preflight_failed`: qwen artifact smoke used bare `qwen -p`, while the runtime adapter needs `--chat-recording false --yolo --channel CI` to enable filesystem artifact writes. Current follow-up aligns preflight with runtime invocation and reruns `release fast`.
- 2026-05-10: Second canonical `release fast` attempt on commit `c343c5c` passed provider preflight but failed baseline backend hard-pass `1/3`: `qwen-code` and `claude-code` both ended as zero-output `runtime_stalled_before_artifacts` before `asis-draft-manifest.json`, while `codex-code` passed. The follow-up widened qwen/claude initial pre-artifact stall windows to 180s, preserved strict artifact-only success, and surfaced failed raw zero-output pre-artifact stalls in quality reports before rerunning release-fast.
- 2026-05-18: Follow-up qwen-only smoke on commit `f724530` confirmed qwen readiness and advanced through `init.step2.asis_docs`, but `refresh.step1.collect` hit a zero-output/no-artifact stall after the widened 180s window on the root-file shard. The remediation kept timeout/matrix policy unchanged and removed a prompt contradiction: the refresh collect first-action skeleton now satisfies refresh minimums (`coverage.missing >= 3`, `questions >= 1`) and root-file evidence prefers README/Makefile/build-deploy manifests over dotfiles.
- 2026-05-18: Qwen-only smoke on committed candidate `7d3f3c8` (`smoke-tiny-bank-20260518T072816Z`) stayed blocked before the targeted refresh slice: `qwen-code` produced a zero-output pre-artifact `runner_unavailable` on `init.step1.collect` root-file shard after the 180s window, while the prompt already contained the README-preferring early pair command. Canonical `release fast` was not run. Claude readiness also carried model telemetry that was previously treated as host/provider configuration, but the current policy removes model-attribution readiness blocking.
- 2026-05-18: Review found one remaining prompt-contract gap: manifest-only collect repair could pass sorted evidence candidates into the skeleton and accidentally choose `.gitignore` before `README.md` for root-file shards. The follow-up fix applies the same README/Makefile/build-deploy preference to repair evidence candidates and adds regression coverage for direct skeleton generation plus the composed repair prompt.
- 2026-05-18: Current PR #69 mergeability review found GitHub CI green but release readiness blocked: PR is draft, qwen smoke still has zero-output `runner_unavailable`, and no `release_verdict_*.json` PASS exists. Claude `model`/`modelUsage` telemetry remains transcript diagnostics only, not release readiness attribution. The current remediation keeps timeout/matrix policy unchanged and moves the collect first-action heredoc pair command immediately after provider identity, so `init.step1.collect` / `refresh.step1.collect` expose the write command before broad artifact/doc-first instructions and only once per prompt. Next gate is qwen-only smoke, then canonical `release fast`.
- 2026-05-18: Qwen-only smoke on command-first prompt commit `865397d` (`smoke-tiny-bank-20260518T084306Z`) passed precheck and moved past the previous earliest collect silence: qwen wrote step0 artifacts and 9/10 `init.step1.collect` shard manifests, with the first-action command visible before broad prompt instructions. Acceptance still failed with `runner_unavailable=1` and `zero_output_pre_artifact_stalls=2`: one zero-output pre-artifact stall on `init.step1.collect` shard `bank-of-anthos-src-ledger-ledgerwriter`, plus one on `init.step2.asis_docs`. Canonical `release fast` remains blocked and was not run; the next slice should diagnose qwen invocation/provider behavior on the remaining silent shard/as-is transition without increasing timeouts or relaxing canonical matrices.
- 2026-05-18: Post-DoD audit found one diagnostics hygiene bug in raw qwen failure metadata: because qwen passes the prompt through `-p`, the redacted `argv` still contained the full artifact prompt even though lifecycle diagnostics already record `prompt_bytes`. The follow-up fix redacts prompt payload argv values (`-p`/`--prompt`) to byte count + hash while preserving provider command/flags/cwd/include-dir diagnostics.
- 2026-05-18: Current slice removes provider/model attribution gate without backward compatibility: `model`/`modelUsage` telemetry is plain transcript diagnostics, not selected-provider readiness/report blocker. Qwen runtime now uses `stream-json` activity output and treats the first zero-output pre-artifact stall as retryable warning; recovered retry is non-blocking, exhausted no-artifact retry remains `runner_unavailable`.
- 2026-05-18: Canonical `release fast` attempt `release-fast-20260518T152336Z` on PR #69 was stopped after a new terminal blocker: `qwen-code` baseline completed, while `claude-code` reached `init.step3.findings` and produced zero stdout/stderr plus no `validator-verdict.json` for the 180s pre-artifact window. The remediation keeps matrices/timeouts/provider contracts unchanged, moves validator steps to command-first `FIRST VALIDATOR VERDICT COMMAND`, and makes only Claude validator zero-output pre-artifact silence a bounded warning/retry; exhausted validator silence still fails as `runner_unavailable`.
- 2026-05-19: Claude-only validator smoke `smoke-tiny-bank-claude-validator-20260518T182851Z` passed on commit `fe3f6d4`, confirming the validator-step code path. Subsequent canonical `release fast` attempts `release-fast-20260518T193132Z` and `release-fast-20260518T193433Z` stopped before backend runs with `operational_host_preflight_failed`: the separate Claude text-only `ACP_READY` preflight probe timed out after 30s, while manual probes showed flaky latency. The current remediation removes that text probe as a Claude gate and relies on `--version` plus runtime-like artifact smoke (`--add-dir` temp write dir) with one bounded retry on timeout/no-output.
- 2026-05-19: After rebase to `f554807`, provider preflight passed and `kimi-for-coding` telemetry remained non-blocking, but canonical `release fast` hit a new product blocker: `claude-code` reached `init.step4.proposals` and produced zero stdout/stderr plus no `proposals-draft-manifest.json` for the 180s pre-artifact window. A separate host hygiene issue removed the temporary Node `22.21.1` path under `/tmp`, causing Codex quality/frontend checks to fail for operational reasons. The current remediation keeps matrices/timeouts/schemas/provider contracts unchanged, adds a command-first `FIRST PROPOSALS DRAFT COMMAND`, makes only Claude proposals zero-output pre-artifact silence a bounded warning/retry, ensures stale runner classifier rows do not override terminal quality failures, and requires stable non-`/tmp` Node toolchain selection for the next gate.
- 2026-05-19: Clean-worktree preflight on `98419c0` found a narrower Claude readiness bug: manual artifact smoke wrote the expected sentinel but the Claude process did not exit before timeout, so preflight still returned `operational_host_preflight_failed`. The follow-up keeps command/probe contracts unchanged and treats Claude timeout-after-sentinel as ready, matching the runtime artifact-only controlled-stop policy; timeout without expected sentinel remains a bounded retry then host blocker.
- 2026-05-19: Claude-only proposals diagnostic on clean commit `98b4573` (`smoke-tiny-bank-claude-proposals-20260519T093914Z`) did not reach the targeted proposals step: `claude-code` hit a fully silent no-artifact `runner_unavailable` on `init.step1.collect` root-file shard after the 180s pre-artifact window, while later collect shards showed that Claude can recover when a fresh focused process writes the manifest. The follow-up keeps timeout/matrix/provider contracts unchanged and scopes Claude zero-output pre-artifact warning/retry to collect as well as validator/proposals; exhausted no-artifact retry remains `runner_unavailable`.
- 2026-05-19: Claude-only proposals diagnostic on clean commit `757ac6c` (`smoke-tiny-bank-claude-proposals-20260519T101428Z`) passed. The next canonical `release fast` (`release-fast-20260519T111703Z`) advanced through qwen init but failed qwen refresh in `refresh.step2.asis_docs`: qwen had authored `src-ledger-balancereader-overview.md`, while the collect manifest referenced typo path `src-ledger-balereader-overview.md`. The remediation keeps schemas/matrices/provider contracts/timeouts unchanged and tightens collect validation so `documents[].path` must reference an existing authored file under `write_root`; missing references trigger the existing provider-authored manifest-only repair before step2, and batch classifiers report stale step2 missing-document failures as `runtime_contract_failed`, not `runner_unavailable`.
- 2026-05-19: Canonical `release fast` on clean commit `d1e7d05` (`release-fast-20260519T134436Z`) reached a green backend baseline (`hard_pass=3/3`) and qwen/codex frontend PASS, but failed release policy because Claude frontend init created a fresh run where `claude-code` hit fully silent no-artifact `runner_unavailable` on `init.step0.constitution`. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: constitution normal prompt now starts with `FIRST CONSTITUTION DRAFT COMMAND`, and only Claude constitution fully silent pre-artifact stalls get the same one bounded warning/retry as collect/validator/proposals; exhausted no-artifact retry remains `runner_unavailable`.
- 2026-05-19: Claude-only constitution diagnostic on clean commit `a848f16` (`smoke-tiny-bank-claude-constitution-20260519T174232Z`) proved `init.step0.constitution` and `init.step4.proposals` no longer hit `runner_unavailable`, but refresh was blocked before runtime because published `skills/subagents.yaml` contained markdown from the generic draft template. The remediation keeps schemas/matrices/provider contracts/timeouts unchanged and makes the constitution first-action `baseline-subagents.yaml` use the canonical valid `agents:` YAML bundle so workspace validation can start refresh.
- 2026-05-19: Canonical `release fast` on clean commit `885f524` (`release-fast-20260519T201118Z`) passed all baseline backend runs and frontend init/cancel surfaces for qwen, Claude, and Codex, including recovered Claude validator silence. Release verdict was still `FAIL` because the matrix harness executed only the first planned profile/sweep row (`single-git_url/baseline`) and then reported missing `parallel-default`; `profile-sweep-combinations.tsv` contained all four required rows. Root cause is shell stdin ownership: child `full-run-batch.sh` inherited the while-loop stdin and could consume the remaining combinations. The remediation keeps canonical matrices/timeouts/provider contracts unchanged, reads combinations through an isolated fd, runs child batches with stdin detached, and adds regression coverage where a dummy child drains stdin while all release combinations still execute.
- 2026-05-20: Canonical `release fast` rerun on clean commit `004f0e9` (`release-fast-20260519T235004Z`) confirmed the matrix stdin fix: all four profile/sweep combinations were planned and execution advanced past the first qwen backend run. The run was stopped after a new product/runtime blocker in `qwen-code` `init.step1.collect` root-file shard: qwen read repo files before the first pair write, wrote only `root-overview.md`, and then manifest-only repair stalled for 180s before writing `shard-pack-manifest.json`. The remediation keeps canonical matrices/schemas/provider contracts/timeouts unchanged, removes the collect prompt conflict that told providers to read entrypoint hints "first", explicitly forbids `read_file`/repo exploration before `FIRST COLLECT ARTIFACT PAIR COMMAND`, and moves manifest-only repair to a `FIRST COLLECT MANIFEST REPAIR COMMAND` heredoc as the first task action.
- 2026-05-20: Audit of PR #69 diff after `release-fast-20260520T012613Z` found no active provider/model attribution blocker left in release path; `model`/`modelUsage` remains diagnostic-only. The release blocker is now product quality/contract: multi-path backend strict gate fails with `analysis:cross-repo-missing`, and Claude frontend init failed `init.step2.asis_docs` because draft repair claimed `overview.md`/`architect-summary.md` existed while only `summary.md` was present. The current remediation keeps matrices/schemas/provider contracts/timeouts unchanged, adds a normal + repair `FIRST AS-IS DRAFT COMMAND` for `asis-draft-manifest.json` plus the three required draft files, and aligns the cross-repo evaluator with documented acceptance of semantic edges, validator/collect findings with multi-repo provenance, or questions with multi-repo `related_ids` plus repo-specific citations.
- 2026-05-21: Clean-worktree Claude frontend diagnostic on `cc223d8` (`smoke-tiny-bank-claude-frontend-20260521T062833Z`) confirmed the backend `step2.asis_docs` and multi-repo semantic fixes: backend hard-passed with no `runtime_contract_failed`, `runner_unavailable`, `runtime_timeout`, `quality_gates_failed`, or `cross_repo_missing` hits. The remaining product blocker is frontend-triggered `init.step0.constitution`: focused repair wrote `constitution-draft.json` and `charter-overview.md` correctly, but wrote `baseline-subagents.yaml` under a sibling path where `/frontend/claude-code/` was rewritten to `/frontend-claude-code/`; runtime correctly rejected the missing in-write-set draft as `runtime_contract_failed`. The follow-up keeps artifact-only success and write-set validation unchanged, removes permissive "equivalent writes" wording, and makes normal/repair draft commands assign exact absolute `write_root`/`draft_root` once, then write through shell variables so providers do not manually retype long slash-separated paths.
- 2026-05-21: Canonical `release fast` on clean commit `d8ddf7d` (`release-fast-20260521T093333Z`) completed all backend and frontend surfaces with no runtime/provider/infra blockers, but verdict stayed `FAIL`: qwen/codex multi-path baseline and parallel-default runs had `analysis:cross-repo-missing` because the `FIRST VALIDATOR VERDICT COMMAND` wrote an empty `findings/questions` skeleton and runtime stopped after the valid artifact before late cross-repo instructions could take effect. The follow-up keeps `analysis:cross-repo-missing` strict and does not add ACP-side fallback; instead multi-repo validator first-action and focused repair skeletons now include one PASS-compatible cross-repo finding and one question with repo/path provenance, so qwen/codex first valid artifacts satisfy the provider-facing semantic contract.
- 2026-05-22: Targeted multi-path diagnostic on `38172b1` (`regres-fast-bank-openedx-20260522T021739Z`) passed for qwen/codex and confirmed the validator first-action cross-repo skeleton. Canonical `release fast` (`release-fast-20260522T065332Z`) then passed baseline backend for all providers and frontend init/cancel for Claude/Codex, but qwen frontend init failed because two `init.step1.collect` focused pair repairs returned qwen stream output ending in `[API Error: Premature close]` with process `success` and no artifacts. The run was stopped after the first failed sweep because release PASS was already impossible. The remediation keeps artifact-only success, canonical matrices, schemas, provider contracts, and timeouts unchanged: qwen now treats transient provider API/transport text during collect pair repair as a warning retry signal, retries that focused repair once, and classifies exhausted no-artifact API repair as `runner_unavailable` rather than stale `runtime_contract_failed`.
- 2026-05-23: Fresh `v0.1.1` release candidate gate on main `488e173` (`release-fast-20260523T101656Z`) failed: qwen passed init but `refresh.step2.asis_docs` focused draft repair returned `[API Error: Connection error ... network socket disconnected before secure TLS connection]` with process `success` and no `asis-draft-manifest.json`; the runtime correctly reported `runner_unavailable`, but the existing transient retry covered only collect-pair repairs and did not recognize the TLS/socket transcript as retryable. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: qwen transient provider-unavailable focused retry now also covers draft-artifact repairs (`step0/2/4`) and recognizes connection/TLS/socket API text; exhausted no-artifact repair still fails as `runner_unavailable`. The same release attempt also showed host/network operational failures (`git clone` TLS errors, qwen/codex provider connectivity, occasional Claude smoke timeout), so a new release tag remains blocked until this fix is committed and a fresh clean-worktree release-fast PASS is produced on a stable trusted host.
- 2026-05-23: Fresh gate on main `4147d2c` (`release-fast-20260523T113202Z`) advanced past the transient draft-repair failure but hit a new qwen frontend blocker in `init.step2.asis_docs`: required as-is draft artifacts (`asis-draft-manifest.json`, `overview.md`, `summary.md`, `architect-summary.md`) were valid, while qwen kept streaming and mutating `architect-summary.md` until the global step runtime timeout. This is not a silence/preflight/provider-auth failure; it is an active overrun after valid draft artifacts. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: only `qwen-code` draft steps (`step0/2/4`) get a bounded valid-artifact controlled stop, so continued stream/mutation after a valid manifest+file set is accepted through the existing artifact validation gate instead of waiting for `runtime_timeout`. Collect/validator, Claude, and Codex behavior remain unchanged.
- 2026-05-24: Fresh release-prep gate on `e4db0ac` (`release-fast-20260524T081756Z`) passed the single-repo sweeps and multi-path backend, then exposed a Claude frontend product blocker in `init.step1.collect`: manifest-only repair for `devstack-docs` wrote a schema-valid `shard-pack-manifest.json` but the provider kept running and mutating it, while repair policy had `valid_artifact_stop_window_ms=0`; the old runtime could wait until timeout instead of accepting the valid repair artifact through validation. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: focused repair policies now add a short valid-artifact controlled stop for collect/validator/draft repairs across providers; partial/invalid repair artifacts remain `runtime_contract_failed`.
- 2026-06-01: Published `v0.1.1` as a GitHub prerelease after PR #88 (`7548fdc4`) updated release notes for the owner-approved no-gate path. Main CI passed, release workflow `26761305996` succeeded, and install smoke passed for `ACP_VERSION=v0.1.1` plus authenticated `ACP_VERSION=latest`. Release URL: `https://github.com/GrinRus/ProvenArch/releases/tag/v0.1.1`. Fresh trusted-machine `release-fast` was skipped by owner decision; no final-tag `release_verdict_*.json` exists, so canonical `RELEASE READY` is not claimed.

### Plan ID
EP-20260508-oss-readiness-hardening

### Context
Public OSS readiness audit found that the release binary path works, but repository governance, security reporting, CI trust boundaries, dependency hygiene, and README/community surfaces were below public distribution standards.

### Goals (must have)
- [x] Add security disclosure, support, governance, changelog, code of conduct, CODEOWNERS, PR template, and issue templates
- [x] Harden GitHub Actions permissions, timeouts, action pinning, release environment, SBOM/provenance, and required test alignment
- [x] Add Dependabot, CodeQL, Dependency Review, and Scorecard workflows/configuration
- [x] Fix UI dependency audit issues and enforce exact Node version from `.node-version`
- [x] Move required CI/release/local build toolchain to security-patched Go from `.go-version` while keeping `go.mod` compatibility at `go 1.20`
- [x] Update README/install/testing docs for OSS beta distribution and release evidence boundaries
- [x] Best-effort apply GitHub repository settings where current credentials have permission; otherwise record owner/admin manual tasks
- [ ] Owner/admin manual verification: GitHub still reports `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks` as disabled after API PATCH; enable them if available for the account/plan, or document the platform limitation
- [ ] Owner/admin manual verification: add a safe `creation` rule to the `v*` tag ruleset once bypass actors are confirmed, so release tag creation is restricted to maintainers without locking out the release owner

### Non-goals
- [x] Do not make Docker/npm/PyPI/Maven/crates.io a primary distribution path
- [x] Do not add hosted/SaaS mode or security/compliance enforcement
- [x] Do not run trusted live release matrix without explicit owner approval
- [x] Do not treat release readiness as passed without `reports/release_verdict_<matrix-id>.json`

### Approach
1) Add OSS community/security files and front-load README quickstart/release/security links.
2) Harden workflows with read-only defaults, job-level release writes, pinned actions, timeouts, and advisory security jobs.
3) Update UI dependencies and Node resolver so source builds are deterministic against `.node-version`.
4) Attempt admin-level GitHub settings changes through `gh`; record any settings that require owner/admin manual verification.
5) Run local DoD and targeted security/install checks.

### Files expected to change
- `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `SUPPORT.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`, `GOVERNANCE.md`
- `.github/*`
- `.goreleaser.yml`
- `scripts/resolve-node-tool.sh`, `scripts/tests/*`
- `ui/package.json`, `ui/package-lock.json`
- `docs/INSTALL.md`, `docs/TESTING_STRATEGY.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] `npm audit --prefix ui --audit-level=moderate`
- [x] `govulncheck ./...` when available
- [x] `bash ./scripts/smoke-cli.sh`
- [x] `bash ./scripts/smoke-api.sh`
- [x] GitHub owner/admin-only settings are either applied or explicitly listed as manual residual tasks

### Risks
- Branch protection, rulesets, secret scanning, private vulnerability reporting, and environment reviewers may require owner/admin permissions.
- Exact Node enforcement can block local builds until Node `22.21.1` is installed or selected through `ACP_NODE_TOOL_CANDIDATES`.
- Exact Go enforcement can block local builds until Go from `.go-version` is installed or selected through `ACP_GO_BIN`.
- GoReleaser SBOM generation requires `syft` in the release workflow.
- Local Go 1.20.x remains useful for compatibility checks, but `govulncheck` on May 8, 2026 reports vulnerable standard-library call paths when scanning binaries built with Go 1.20.3.

### Progress log
- 2026-05-08: Started OSS readiness hardening slice from the public distribution audit plan. Live matrix was not run.
- 2026-05-08: `govulncheck` against local Go 1.20.3 reported standard-library vulnerabilities; CI/release toolchain was moved to Go 1.25.10 while preserving `go.mod` language compatibility.
- 2026-05-08: Second audit found local `make build` and smoke scripts could still use stale `go` from PATH; added `.go-version`, `scripts/run-go.sh`, and wired Makefile/smoke scripts to fail fast on mismatched Go toolchains.
- 2026-05-08: Follow-up `make test` correctly failed `TestActivePlansHaveOpenGoals` because this active plan had all goals checked. Reopened the only real residual owner/admin task for GitHub secret scanning non-provider/validity features.
- 2026-05-08: Added OSS community/security files, issue/PR templates, Dependabot, CodeQL, Dependency Review, Scorecard, pinned GitHub Actions, release environment, SBOM/provenance release hooks, UI dependency updates, and exact Node resolver enforcement.
- 2026-05-08: Applied GitHub settings with admin token: `main` branch protection, `v*` tag ruleset for deletion/non-fast-forward blocking, `github-release` environment reviewer, private vulnerability reporting, Dependabot alerts/security updates, secret scanning, push protection, repo description/topics, labels, milestones, and curated `v0.1.0` release notes. GitHub still reports `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks` as disabled after PATCH; owners should manually verify whether these features are available for the account/plan.
- 2026-05-08: Tag ruleset creation restriction was left as owner/admin verification instead of being enabled blindly. GitHub's ruleset model restricts creations to bypass actors, and the safe bypass role/team set for this personal repository should be confirmed before enforcement.
- 2026-05-08: Verification passed with Go 1.25.10 + Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`, `npm audit --prefix ui --audit-level=moderate`, `govulncheck ./...`, `bash ./scripts/smoke-cli.sh`, `bash ./scripts/smoke-api.sh`, `git diff --check`, YAML parse smoke, and `goreleaser check`.
- 2026-05-08: Final workflow audit found two release-readiness bugs: `dependency-review` was pinned to a SHA from the wrong action repository, and Go workflows duplicated the exact version instead of reading `.go-version`. Fixed both and added regression coverage.
- 2026-05-08: Second-pass verification passed after Go guard and issue-template fixes: `make contracts`, `make test`, `make lint`, `make build`, `npm audit --prefix ui --audit-level=moderate`, `govulncheck ./...`, `bash ./scripts/smoke-cli.sh`, `bash ./scripts/smoke-api.sh`, `git diff --check`, YAML parse smoke, `goreleaser check`, and targeted Go/Node resolver tests.
- 2026-05-08: Final release metadata audit found `v0.1.0` was published as a normal GitHub Release while docs call the line MVP beta/pre-release. Changed GoReleaser to keep future beta releases prerelease and fixed `install.sh` so `ACP_VERSION=latest` resolves prerelease releases through the GitHub Releases API instead of relying on `/releases/latest/download`.
- 2026-05-08: Tried marking the already-published `v0.1.0` as prerelease, but reverted the remote release to latest/non-prerelease because the current public `main/install.sh` still uses GitHub's `/releases/latest/download` endpoint. Public install smoke passed again after restoring `v0.1.0` as latest. After this installer fix is merged to `main`, owners can mark beta releases as GitHub prereleases without breaking the default install command.
- 2026-05-08: PR mergeability audit found a solo-maintainer governance deadlock: `main` required one approval plus CODEOWNERS review, but `GrinRus` is currently the only collaborator and CODEOWNER. Documented the solo-maintainer policy: required CI remains mandatory, while required approving/CODEOWNERS reviews are enabled after a second maintainer exists.

### Plan ID
EP-20260507-trusted-live-validation

### Context
Historical active plans from 2026-04-20 through 2026-05-06 repeatedly ended with the same unresolved gate: trusted-host live reruns through the canonical matrix harness. Local behavior fixes and DoD evidence are present for those slices, but live confidence still requires direct `scripts/full-run-batch-matrix.sh` execution from a clean committed trusted workspace with `qwen`, `claude`, and `codex` available.

### Goals (must have)
- [ ] Prepare a clean committed trusted-host worktree with required provider binaries/auth and pinned repo/path prerequisites
- [ ] Run qwen `regres fast` for `examples/e2e-matrix.regres-fast.bank-openedx.yaml` and `examples/e2e-matrix.regres-fast.openstack.yaml`
- [ ] Run qwen+codex diagnostic rerun for bank/openedx and openstack scenarios
- [ ] Run qwen+claude rerun for the current small live bugfix matrix
- [ ] Run claude bank/openedx verification and qwen-focused `regres fast`
- [ ] Run full fresh-all-6 only after smaller slices pass or produce narrower blocker reports
- [ ] Record matrix ids, driver logs, profile/release reports, quality reports, taskrun raw diagnostics, and verifier output where a release verdict is used
- [ ] Update `docs/PLANS.md`/archive with pass/fail evidence and any follow-up fix slices created from live failures

### Non-goals
- [x] Do not add wrapper scripts around `scripts/full-run-batch-matrix.sh`
- [x] Do not wire live matrix into required CI
- [x] Do not edit canonical matrix/profile/timeout files to fit the current machine
- [x] Do not treat live release readiness as passed without `reports/release_verdict_<matrix-id>.json` and `scripts/verify-release-verdict.py` when release verdict is used

### Approach
1) Verify trusted-host prerequisites and clean committed tree.
2) Execute each matrix by setting `E2E_MATRIX_FILE`, provider command envs, and `BATCH_PROVIDER_FILTER`, then invoking `./scripts/full-run-batch-matrix.sh` directly.
3) Monitor driver/status/report artifacts until terminal state.
4) Classify failures into product/runtime/harness/operational buckets and open a focused fix slice before broad reruns when needed.
5) Close historical live-gate tasks only with concrete matrix/report evidence.

### Non-live prerequisite check (2026-05-07)

| Check | Status | Evidence / note |
|---|---|---|
| Clean committed local worktree | ready | `git status --short` was empty at `c1f58b5`. |
| Required provider binaries in PATH | ready for auth check | `command -v qwen`, `command -v claude`, and `command -v codex` returned local paths. Exact machine-local paths are intentionally omitted from the versioned plan; provider auth/quota was not probed. |
| Canonical harness inputs present | ready | `examples/e2e-matrix.regres-fast.bank-openedx.yaml`, `examples/e2e-matrix.regres-fast.openstack.yaml`, `scripts/full-run-batch-matrix.sh`, and `scripts/verify-release-verdict.py` exist. |
| Live execution permission | blocked | No live matrix was started in this local implementation pass. Running live validation still needs explicit trusted-machine approval and direct `scripts/full-run-batch-matrix.sh` invocation. |

### Critical analysis (2026-05-08)

This slice is not closed and must not be treated as release evidence. The local worktree check and provider binary discovery only prove that the current machine can find command names; they do not prove provider authentication, quota availability, pinned repo/path readiness, matrix pass/fail status, or release readiness.

Residual blockers:
- live matrix execution still requires explicit trusted-machine approval;
- provider auth/quota remains unverified;
- pass criteria remain completely open until direct `scripts/full-run-batch-matrix.sh` runs produce matrix/report artifacts;
- release readiness still requires `reports/release_verdict_<matrix-id>.json` plus `scripts/verify-release-verdict.py` when a release verdict is used.

### Files expected to change
- `docs/PLANS.md`
- `docs/archive/*` for execution reports or reconciliation notes
- focused code/docs/tests only if live evidence creates a new fix slice

### Acceptance criteria
- [ ] No unresolved `runtime_contract_failed`, `runner_unavailable`, `semantic_hard_fail`, independent frontend failure, stale `running`, or unexpected `analysis:cross-repo-missing` remains for fixed scenarios
- [ ] Every live run has a recorded `matrix_id`, `driver.log`, `profile_matrix_<matrix-id>.*`, quality reports, and taskrun diagnostics
- [x] Release readiness uses only `reports/release_verdict_<matrix-id>.json`; verifier exits 0 only for `PASS` / `RELEASE READY` / `passed`
- [ ] Any live failure produces a narrower follow-up fix slice instead of ad-hoc matrix edits

### Risks
- Provider outages, auth/quota limits, or missing trusted-host path prerequisites can block live gates without indicating product regression.
- Full fresh-all-6 can be expensive; run it only after smaller gates either pass or isolate residual blockers.

### Progress log
- 2026-05-07: Created by tracker reconciliation. Consolidates live rerun gates from historical active plans; no live matrix was run in this reconciliation slice.
- 2026-05-07: Performed non-live prerequisite check only. Worktree and provider binary paths look ready for an owner-approved trusted-machine run, but provider auth/quota and pinned repo/path prerequisites are unverified. Live matrix execution remains blocked until explicit approval.

### Plan ID
EP-20260507-cleanup-owner-decisions

### Context
Safe cleanup is already complete, but `docs/PLANS.md` and `docs/BACKLOG.md` retain owner-gated cleanup decisions where deleting or moving tracked surfaces could damage discoverability, review UX, or hidden tooling usage. These items must remain explicit and blocked until owner intent is known.

### Goals (must have)
- [x] Gather usage evidence for whether `internal/docsync` and `internal/scriptsmeta` should remain test-only packages under `internal/*`
- [x] Gather evidence on retaining both `docs/archive/PLANS_ARCHIVE_2026-04.md` and `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`
- [x] Gather evidence on persisted `fixtures/scenarios/*/golden/readable`
- [x] Gather evidence on duplicated readable scenario fixtures
- [x] Gather evidence on `docs/BACKLOG.md` role as active planning surface vs reference/history backlog
- [ ] Owner decision: retain/move `internal/docsync` and `internal/scriptsmeta`
- [ ] Owner decision: retain/remove/dedupe April plan archive + snapshot docs
- [ ] Owner decision: retain/remove/dedupe readable golden fixtures
- [ ] Owner decision: keep `docs/BACKLOG.md` as reference/acceptance backlog or change its planning-surface role
- [ ] If approved by owners, implement each cleanup as a separate small change set with tests/docs sync

### Non-goals
- [x] Do not move `internal/docsync` or `internal/scriptsmeta` without owner approval
- [x] Do not delete archive snapshots, readable fixtures, or duplicated fixtures without owner approval
- [x] Do not mix destructive cleanup with runtime/live fixes

### Approach
1) Search usage and references for each cleanup candidate.
2) Write a short evidence table with retain/remove/dedupe options and recommended default.
3) Request owner decisions; retain by default if no answer is available.
4) Implement approved cleanup items one at a time with focused tests and full DoD when tracked files change.

### Owner-decision evidence (2026-05-07)

| Candidate | Evidence gathered | Options for owner | Default until decision |
|---|---|---|---|
| `internal/docsync`, `internal/scriptsmeta` | Only test files are present under those packages: `internal/docsync/docsync_test.go`, `internal/scriptsmeta/resolve_repos_meta_test.go`. `go test ./...` runs them as repo-level consistency gates; `docs/TESTING_STRATEGY.md` names `internal/docsync` as a docs-consistency gate. | Retain under `internal/*`; move to a dedicated test-support location; collapse into script tests. | Retain under `internal/*`; moving them is layout churn without owner-confirmed value. |
| `docs/archive/PLANS_ARCHIVE_2026-04.md` + `docs/archive/PLANS_SNAPSHOT_2026-04-21.md` | `docs/PLANS.md` lists both archives; the snapshot records that completed historical plans moved to the April archive. Current sizes are 912 and 1176 lines, respectively, so they are substantial historical evidence. | Retain both; collapse snapshot into archive; keep snapshot but remove cross-links. | Retain both until owner decides the audit/history surface can be reduced. |
| `fixtures/scenarios/*/golden/readable/*` | 90 tracked readable files exist. `docs/BASELINE_POLICY.md`, `docs/TESTING_STRATEGY.md`, `fixtures/README.md`, and `internal/docsync/docsync_test.go` explicitly describe them as tracked baseline/release surface and human-readable deterministic export. | Retain tracked readable exports; remove and rely on machine-readable golden only; generate on demand without tracking. | Retain; current docs/tests make them intentional, not accidental generated output. |
| Duplicated readable scenario fixtures | Hash scan shows many identical files repeated across the three readable scenario exports, including `reports/as-is/*`, `reports/diagrams/*`, `model/entities/svc.payments.yaml`, and proposal docs. | Keep duplicated per-scenario snapshots; dedupe via shared fixture layer; remove readable exports after replacing review-diff workflow. | Retain duplicated snapshots; dedupe needs a QA/tooling owner decision because per-scenario full-tree diffs are the current review UX. |
| `docs/BACKLOG.md` role | README still points to `docs/BACKLOG.md` for epics/acceptance criteria, AGENTS instructs agents to take reviewable slices from it, and `internal/docsync/docsync_test.go` asserts several backlog truth-sync strings. The backlog itself says active engineering slices live in `docs/STAKEHOLDER_DOC.md` and `docs/PLANS.md`, while a 2026-04-22 follow-up asks owners to decide active planning surface vs reference/history role. | Keep as reference/acceptance backlog; promote it back to active planning surface; archive/freeze it after migrating acceptance criteria elsewhere. | Keep current reference/acceptance role until owners decide; active execution remains in `docs/PLANS.md`. |

### Critical analysis (2026-05-08)

This slice is evidence-complete but decision-blocked. The gathered evidence points toward retaining all cleanup candidates by default because the candidates are referenced by tests/docs or are intentionally tracked review surfaces. However, retaining by default is not the same as resolving ownership: no package move, archive collapse, fixture deletion, fixture dedupe, or runbook consolidation should happen until maintainers/docs/tooling owners choose an option.

Residual blockers:
- no owner decision has been provided for test-only package placement;
- no owner decision has been provided for archive/snapshot retention;
- no owner decision has been provided for readable fixture retention or dedupe;
- no owner decision has been provided for the long-term role of `docs/BACKLOG.md`;
- no destructive cleanup was performed, so the final implementation goal remains intentionally open.

### Files expected to change
- `docs/PLANS.md`
- `docs/BACKLOG.md` only if owner-gated backlog wording changes
- candidate files only after explicit owner approval

### Acceptance criteria
- [x] Each owner-gated item has usage evidence and an explicit retain/remove/dedupe recommendation
- [x] No destructive cleanup happens without owner approval
- [ ] Approved cleanup items are implemented in separate commits
- [x] Active plan surface keeps owner-gated decisions visible until resolved

### Risks
- Removing historical docs or readable fixtures can degrade review workflows even if tests do not fail.
- Keeping everything can preserve drift; retained items need explicit ownership or periodic review.

### Progress log
- 2026-05-07: Created by tracker reconciliation from `EP-20260421-cleanup-owner-followups` and `docs/BACKLOG.md` cleanup follow-ups.
- 2026-05-07: Gathered usage evidence and documented retain-by-default recommendations. No files were deleted, moved, or deduplicated; owner decisions remain open and blocked.
- 2026-05-18: Removed the live E2E convenience-runbook owner decision from this cleanup plan because the black-box live E2E slice deletes old live E2E surfaces without compatibility.

---

## Implemented vs Planned (operational mirror)

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Таблица ниже — инженерный mirror и должна оставаться синхронизированной с канонической матрицей.

| Epic | Статус | Комментарий |
|---|---|---|
| 1 Workspace/contracts | done (beta baseline) | Schema-driven + semantic validation, resolver `path/git_url`, diagnostics API |
| 2 Runtime artifact contracts | done (beta baseline) | Validation + artifact-only runtime execution contract, contract tests |
| 3 Runtime/orchestration seam | done (beta baseline) | Fake default + opt-in headless runtime selector with provider choice (`claude-code` default, `qwen-code`, `codex-code` release peer), persisted runtime execution metadata |
| 4 Model deterministic core | done (beta baseline) | Canonical IDs/collision rules + deterministic regression tests |
| 5 Pipeline 0–4 | done (beta baseline) | `init|refresh` runnable через CLI/API |
| 6 UI baseline | done (beta baseline) | Setup/validate/run/inspect + editors + git helpers |
| 7 Domain-first layer | done (beta baseline) | Per-domain contracts + deterministic Step 1 enrichment canonical domain/team cards without auto-create |
| 8 Baseline bundle | done (beta baseline) | `skills/subagents.yaml` + prompt packs + validation |
| 9 Q&A capability | target upgraded | Async runtime-backed Ask via `/api/qa/runs`; deterministic service remains `acp qa` + legacy `POST /api/qa/ask` compatibility |
| 10 Changelog compilers | done (beta baseline) | Iteration changelog materialization в `reports/changelog/*` |
| 11 `POST /api/qa/ask` | done (compatibility baseline) | Read-only API wrapper over deterministic Q&A service while UI target uses async QA runs |
| 12–13 | out of MVP | Вне текущего beta scope |
| 14 CI trigger mode | done (beta baseline) | CLI batch required, smoke/golden jobs без live network deps |
| 15 Domain/baseline pack hardening | done (beta baseline) | Baseline skills/prompts wired и versioned в workspace |
| 16 Console V2 UX | done (beta baseline) | Mission-control shell, Source/Readiness/Review/Publish surfaces, live E2E selector migration and fake/direct-mode coverage |
| 17 Onboarding-first setup | done (beta baseline) | `acp serve` launcher/onboarding, workspace create/open, multi-repo sources, mandatory runner choice and direct `--workspace` compatibility path |
