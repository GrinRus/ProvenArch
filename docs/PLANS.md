# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон, текущие активные планы и инженерный operational mirror.

Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
- `docs/archive/PLANS_ARCHIVE_2026-05.md`
- `docs/archive/PLANS_ARCHIVE_2026-06.md`
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

Current engineering queue has no active engineering slice after PR #110; the remaining `v0.1.7` work is release metadata/tagging bookkeeping. The next implementation workstream must be selected by the owner from the reference backlog or a new ExecPlan; trusted live validation and release-fast remain manual owner-triggered gates, not default backlog work.

Task selection rules:
- Completed plans whose only remaining item is owner review, merge/archive bookkeeping, or historical evidence retention are not next engineering work.
- Owner-decision and trusted-host/live-release items remain explicit blockers; do not run or edit them as normal backlog tasks without the required owner/trusted-machine prerequisites.
- Each selected slice gets a decision-complete ExecPlan/update before implementation, one focused implementation pass, self-review/fix loops, Full DoD (`make contracts`, `make test`, `make lint`, `make build`), then one commit.

### Plan ID
EP-20260616-live-e2e-recovery-rerun-loop

### Context
Diagnostic `claude-code` medium (`regres long`) live runs exposed harness/runtime issues before a clean strict-medium acceptance loop can continue: `draft_artifact_enrichment` for `step2.asis_docs` did not rewrite all bootstrap draft files, batch reporting let a stale `runner_unavailable` classifier row override collect partial root cause, and the `f2e962f` rerun showed fully silent collect fresh retry exhaustion falling straight to `runner_unavailable` without a focused collect-pair repair attempt. The user requested fix -> DoD -> commit -> live rerun loop across `claude-code`/`codex-code` with target rotation, without product UI/API changes or canonical matrix/repo edits.

### Goals (must have)
- [ ] Harden step2 draft enrichment prompt/contract so enrichment reads bounded staged evidence and write-first overwrites `overview.md`, `summary.md`, and `architect-summary.md`.
- [ ] Keep bootstrap/noop enrichment as `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`; do not add deterministic synthesis as a hidden success path.
- [ ] Run one provider-authored collect-pair repair after exhausted fully silent collect retry before terminal `runner_unavailable`, while keeping invalid/noop repair terminal.
- [ ] Make collect-pair repair evidence-first without seed/fallback heredoc, and stop invalid observed artifacts by no-fresh-mutation window even when provider stdout remains active.
- [ ] Fix report classification so collect partial failures remain primary `runtime_flow_failed` while provider class is secondary evidence.
- [ ] Update tests and live E2E docs/runbooks for the changed runtime/reporting behavior.
- [ ] Run full DoD, commit, then rerun strict medium live E2E from a clean tree.

### Non-goals
- [ ] Do not change product UI/API behavior.
- [ ] Do not edit canonical matrix files, curated repo files, or provider command shims.
- [ ] Do not treat diagnostic non-release evidence as release readiness.

### Approach
1) Patch promptcontract and reporting classification with focused unit/script tests.
2) Synchronize `docs/RELEASE_LIVE_E2E_RUNBOOK.md`, `docs/spec/PIPELINE_SPEC.md`, `docs/ARCHITECTURE.md`, and `docs/TESTING_STRATEGY.md`.
3) Run targeted tests, then full DoD (`make contracts`, `make test`, `make lint`, `make build`).
4) Commit the fix slice.
5) Run `regres long` medium through direct `scripts/full-run-batch-matrix.sh`, starting with `codex-code` and then `claude-code`; evaluate execution, artifact and UX quality; continue the requested provider loop if blockers remain.

### Files expected to change
- `internal/runtime/promptcontract/draft_repair.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `internal/runtime/providercommon/artifact_recovery.go`
- `internal/runtime/providercommon/engine_test.go`
- `scripts/e2e_report_classifiers.py`
- `scripts/e2e_batch_report.py`
- `scripts/tests/batch_failure_classification_test.py`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Prompt tests prove step2 enrichment has exact write-first targets and banned bootstrap scaffold.
- [ ] Providercommon tests prove exhausted fully silent collect retry invokes collect-pair repair and invalid/noop repair remains terminal.
- [ ] Script tests prove partial collect root cause is not overridden by stale provider classifier rows.
- [ ] Full DoD passes before rerun.
- [ ] Strict medium rerun evidence is reported across execution quality, artifact quality and UX quality.

### Risks
- Live providers can still fail due to quota/auth/timeout, which must be classified separately from ACP regressions.
- The stricter prompt can increase provider work in medium/full runs; bounded staged evidence limits the blast radius.
- Failed backend refresh can still leave triage snapshots; frontend smoke must require a succeeded refresh row before using a snapshot, otherwise it can launch a heavy UI-side live run after an already terminal backend failure.

### Progress log
- 2026-06-16: Started fix slice from the failed `regres-long-posthog-ftgo-20260616T033256Z` diagnostic evidence.
- 2026-06-16: Committed draft enrichment/reporting fix `f2e962f`; strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260616T062005Z` still failed with partial collect (`posthog` 2 failed shards, `ftgo` 8 failed shards). Reports correctly kept primary `runtime_flow_failed`, selected-provider totals, `execution_report_*`, and no legacy mixed quality artifacts, but runtime showed repeated fully silent collect retry exhaustion without focused `collect_pair_repair`. Current fix slice adds that recovery path plus classifier/docs coverage.
- 2026-06-16: Committed collect silent-retry repair fix `532c967`; strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260616T100303Z` proved `collect_pair_repair` now schedules, but PostHog failed 14/16 shards because the pair-repair prompt still wrote seed/recovery fallback markdown and let Codex continue without fresh target mutation. The FTGO profile was manually interrupted after the same no-fresh-mutation pattern appeared in `step0.constitution` and would otherwise wait for the full medium step timeout. Current slice removes seed pair-repair prompt logic and makes invalid observed artifacts stop on the bounded partial-artifact window despite active stdout.
- 2026-06-18: After commits `5587298`, `6e84a9f`, and `8c452d0`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260618T083335Z` progressed PostHog init and refresh collect to 16/16 shards, preserving selected-provider totals and failed headless refresh row. It exposed a new primary blocker at `refresh.step2.asis_docs`: `draft_artifact_enrichment` read bounded evidence and then stopped after analysis-only prose (“I have enough bounded evidence to rewrite now”) without fresh writes, leaving `overview.md` and `architect-summary.md` bootstrap-only. Runtime correctly failed as `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`. The same run showed a secondary frontend harness issue: failed refresh had a triage snapshot directory, so frontend smoke started a heavy UI live run. Current slice makes enrichment command-first/write-first and makes frontend snapshot eligibility require latest headless refresh status `succeeded`.
- 2026-06-18: Started Live E2E Quality Loop fix slice for `codex-code`/`claude-code` medium reruns. Scope: make manifest-only collect repair provider-authored instead of hidden writer command, mark deterministic `collect_manifest_runtime_recovery` as explicit runtime warning/diagnostic evidence, preserve execution/artifact telemetry separation in `execution_report_*`, and add AOR boundary tests so live/manual release evidence names do not enter core runtime/orchestrator logic.
- 2026-06-18: Committed first loop slice `5587298`; diagnostic `codex-code` strict medium run `regres-long-posthog-ftgo-20260618T065744Z` reached PostHog collect and exposed a new collect-repair blocker. The first shard wrote evidence-backed `root-overview.md` during `collect_pair_repair` but then read `.test_durations` and stalled before `shard-pack-manifest.json`; the second shard exhausted pair repair with no authored output. The run was interrupted after two independent P0 collect failures to avoid spending the remaining medium window on repeated shard failures. Current slice narrows pair-repair evidence candidates, forbids extra repo reads after markdown write, and lets markdown-only pair repair partials chain into explicit manifest runtime recovery while no-artifact pair repair remains terminal.
- 2026-06-18: Committed second loop slice `6e84a9f`; diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T074654Z` confirmed the markdown-only manifest recovery path, but exposed a no-artifact collect-pair repair failure on the first PostHog shard. Raw stdout showed Codex created its own broad evidence-read command, read many root files plus `docker-compose.dev.yml`, and stalled before writing either `root-overview.md` or `shard-pack-manifest.json`. The run was interrupted as `infra_signal_terminated` after the P0 blocker was captured. The later exact read-only `collect_pair_repair_preflight` attempt also stalled preflight-only, so the current slice makes collect-pair repair one-shot write-first instead of two-phase preflight/write.
- 2026-06-18: After commit `28b5937`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T142635Z` proved the one-shot write-first prompt made Codex attempt a provider-authored filesystem command, but the provider added a brittle exact-phrase precondition (`required = [...]`, `missing expected evidence`) and aborted before writing any authored artifacts when current PostHog text did not match guessed snippets. Current slice keeps provider-authored recovery but forbids hard-coded expected-phrase gates before writes; repair may only precheck target containment, at least one allowed evidence file read, and size limits, while unsupported planned claims must be omitted or reported as coverage gaps.
- 2026-06-18: After commit `df771e5`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T145933Z` showed the root-file collect pair repair now writes valid artifacts, but the next directory shard (`posthog-bin`) failed before writing because Codex generated an over-complex Python writer with an invalid f-string (`f-string: empty expression not allowed`). Current slice keeps provider-authored recovery but tightens the first repair command to a mechanically simple writer: no Python f-strings, `.format(...)` template writers, generated Python source strings, nested quote tricks, or self-invented semantic pre-write aborts; backend validation remains the semantic gate.
- 2026-06-18: After commit `2edf6cf`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T153313Z` confirmed the f-string failure was removed and the root-file shard still recovered, but `posthog-bin` exhausted collect pair repair with no artifacts because Codex added a file-size hard gate (`read file exceeds size limit`) after reading a candidate larger than the per-file budget. The run was interrupted once this acceptance blocker was captured. Current slice keeps the evidence-bounded contract but changes the repair instruction to read bounded prefixes or skip oversized candidates and continue; terminal no-write failure is allowed only when no allowed evidence yields bytes.

### Plan ID
EP-20260608-medium-live-e2e-quality-ui

### Context
Run a medium live E2E assessment on a trusted local host after a user-reported weak local result: sparse artifacts and missing C4 diagrams. This is an operator evaluation, not a product code slice. The run must follow `acp-e2e-live-gate` and `docs/RELEASE_LIVE_E2E_RUNBOOK.md`: use direct public harness commands, avoid wrapper scripts, avoid canonical matrix edits, and classify host/provider/path blockers separately from ACP product defects.

The medium non-release profile is `regres long`: `posthog` (`single-path`, medium shard bucket) plus `ftgo-application` (`single-git_url`, medium shard bucket), default qwen-only baseline with `RUN_COUNT=1`. Because the current worktree may need this plan update, execute the harness from a separate clean worktree.

### Goals (must have)
- [ ] Run host/tree/provider/path preflight for the medium live E2E profile.
- [ ] Launch `regres long` through `scripts/full-run-batch-matrix.sh` using the public planner/harness path.
- [ ] Keep frontend init inspection enabled so UI artifact readability can be assessed from real UI/API evidence.
- [ ] Inspect `run_matrix_*`, `execution_report_*`, `reports/taskruns/*-quality.json` telemetry, promoted reports, diagrams and raw taskrun metadata.
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

### Progress notes
- 2026-06-11: Re-ran medium diagnostics with extended activity windows on the selected live
  providers. One provider stopped at readiness, while another produced a substantive single-path
  `init` artifact set before a later `refresh` stopped on provider availability. Manual UI review
  found the root operator issue: Review defaulted to the latest failed partial run artifact set,
  hiding the complete successful init artifacts unless the operator switched runs from Analysis. The
  UI now keeps the failed-run blocker visible and offers a Review recovery action to open the latest
  successful artifacts; C4 preview also uses a scrollable canvas so large diagrams are not reduced to
  unreadable thumbnails.

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
4) Add prompt snapshot/contract tests that verify normal collect uses evidence-first bounded reads before writing artifacts, unchanged collect seeds/recovery fallbacks and scaffold-only manifest semantics are rejected before apply, draft first-action commands remain bootstrap-only until enriched, and final success is forbidden from generic placeholder text.
5) Tighten runtime prompt policy in `steppolicy`/`promptcontract`: normal collect must write evidence-backed doc+manifest pairs instead of seed-only first-action artifacts, draft first-action skeletons may bootstrap required files, and single-repo validator prompts must not return `PASS` with zero findings/questions when coverage gaps remain.
6) Extend `scripts/e2e_batch_report.py` and Python tests so dynamically inspected bad artifact shapes remain telemetry/manual-assessment evidence, while execution failures stay limited to host/preflight/runtime/frontend/infra classes.
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
- 2026-06-08: Medium diagnostic on selected live providers reproduced product-quality blockers before
  full provider readiness: collect manifests had citations/docs but no semantic entities/edges/findings,
  C4 was gap-only, promoted reports/proposals contained placeholder text, and frontend inspection
  failed on gap-only C4 before Review/mobile readability could be assessed. Current fix slice makes
  collect skeleton/repair artifacts carry minimal evidence-backed semantic signal, removes
  provider-placeholder draft text from normal and focused repair paths, and makes backend-cycle fail
  fast on fresh `artifact_quality:*` warnings before dependent frontend.
- 2026-06-08: Follow-up fresh run proved the empty-model blocker was fixed, but operator quality was
  still not acceptable: semantic output was scaffold-like repo/shard containment with repeated
  owner-gap findings, and C4 context still showed gap-only context nodes. Added dynamic
  `artifact_quality.semantic_scaffold_only` and `artifact_quality.c4_context_scaffold_only` blockers
  from temporary in-test artifact trees, not from a persisted bad-run fixture.
- 2026-06-08/09: Several fresh diagnostics exposed collect bootstrap and repair-routing defects:
  unchanged first-action artifacts, bootstrap markdown routed to manifest-only repair, recovery-only
  low-signal docs, and early post-artifact termination before targeted evidence enrichment. The fixes
  add collect bootstrap detection, route bootstrap-only docs through pair repair, extend live adapter
  enrichment windows, and reject scaffold recovery prose in collect validation.
- 2026-06-09: Follow-up diagnostics narrowed the collect prompt issue: normal collect still depended
  on a seed heredoc and later enrichment that live execution did not reliably reach. The fix removes
  the normal seed heredoc helper, makes collect evidence-first, and rejects unchanged scaffold semantic
  skeletons even when markdown no longer contains seed prose.
- 2026-06-09: Manifest-only repair then proved too easy to accept syntactically while preserving
  wrapper-only semantic output. The fix makes manifest repair evidence-first, treats the embedded
  skeleton as schema guidance only, requires concrete named entities and non-container relationships,
  and keeps backend validation as the only success surface.
- 2026-06-09: Follow-up diagnostics showed evidence-first repair still lacked a reliable filesystem
  write surface and could stall after authored markdown already existed. The fix makes manifest repair
  command-first and adds deterministic `collect_manifest_runtime_recovery` that derives a manifest
  from provider-authored markdown under the existing write-set guard when provider repair stalls.
- 2026-06-09: Runtime recovery removed the hard contract failure but exposed inefficient ordering and
  scaffold-only recovered semantic. The fix runs deterministic recovery before provider manifest
  repair for missing/scaffold-only manifests with non-bootstrap authored docs, and strengthens recovered
  manifests with concrete service/component/datastore entities plus usage/dependency/configuration
  edges from authored markdown.
- 2026-06-09: Medium diagnostic with extended provider activity windows proved the timeout override
  slice works for long-running collect enrichment, while a later partial diagnostic showed preflight
  timeout changes did not control runtime activity windows. The fix adds diagnostic-only provider
  activity timeout env overrides that flow through matrix timeout env handling and remain blocked in
  release mode.
- 2026-06-09: Medium diagnostic with extended live activity windows proved artifacts can become
  substantive and frontend can complete, but a nontrivial target failed quality on
  `artifact_quality.c4_context_scaffold_only`: C4 context stayed gap-only despite a populated model.
  Operator screenshot review also found Review/Publish preview readability was under-tested. The fix
  adds C4 context internal-relation fallback plus viewport-visible Review/Publish preview assertions
  and layout hardening.
- 2026-06-10: Medium diagnostic with provider stall windows raised further proved old limits still cut
  valid work, but a traceable changelog hit a false `artifact_quality.placeholder_artifact` because it
  included a generic validation note. The fix narrows placeholder detection so bootstrap/generic
  proposal text remains blocked while traceable finding/question-backed changelog content passes.
- 2026-06-11: PR #110 merged into `main` at `e68d35c` after green PR checks, green post-merge `main`
  CI, and local `make contracts`, `make test`, `make lint`, `make build`. Started `v0.1.7` CI-only
  beta release metadata branch to publish the live artifact quality hardening in downloadable release
  artifacts. Canonical live release gate remains blocked on this host because Open edX/OpenStack
  canonical path checkouts under `/tmp/provenarch-live-e2e` are missing; no `RELEASE READY` claim is
  made without a fresh `release_verdict_*.json`.

### Plan ID
EP-20260602-onboarding-first-startup

### Context
Clean UI startup in `v0.1.5` works, but first live-provider use exposed a confusing boundary: ACP provider IDs use stable adapter names (`claude-code`, `qwen-code`, `codex-code`), while local binaries are usually `claude`, `qwen`, and `codex`. `qwen-code` and `codex-code` already default to the expected binary names; `claude-code` still defaults to `claude-code`, so a normal Homebrew Claude install fails readiness until the operator sets `ACP_CLAUDE_CMD`. The onboarding screen also relies on typed paths only and can become hard to scan with long paths or diagnostics.

### Goals (must have)
- [x] Keep provider IDs unchanged while normalizing executable resolution: `claude-code` resolves `ACP_CLAUDE_CMD -> claude -> claude-code`, `qwen-code` resolves `ACP_QWEN_CMD -> qwen`, and `codex-code` resolves `ACP_CODEX_CMD -> codex`.
- [x] Add local-only onboarding path suggestions for workspace and local repo paths without writing to target repos or changing workspace schema.
- [x] Add searchable path comboboxes for workspace and local repo rows while preserving typed path entry and explicit create/open/save actions.
- [x] Polish onboarding rendering for desktop, narrow desktop and mobile so long paths, missing recents, duplicate repo names and runner command errors stay readable.
- [x] Sync README/install/API docs with provider ID vs executable wording and clean `acp serve` onboarding guidance.
- [x] Owner review/merge complete; archive remains post-release housekeeping after `v0.1.6`.
- [x] Publish `v0.1.6` release metadata/tag and verify install smoke.
- [ ] Archive this completed plan during post-release housekeeping.

### Non-goals
- [x] No provider ID, CLI flag, workspace schema, runtime artifact contract, source repo write-policy or provider-live release gate changes.
- [x] No native OS/browser directory picker, hosted picker, repo cloning, or source repo mutation in the path suggestion API.
- [x] No change to direct `acp serve --workspace ...` behavior.

### Approach
1) Add provider command-resolution helpers/tests so readiness can discover the installed `claude` binary while retaining legacy `claude-code`.
2) Add `/api/onboarding/path-suggestions?kind=workspace|repo&query=...` with bounded local directory suggestions under safe roots.
3) Add typed contracts/API client and a reusable `LocalPathCombobox`; wire it into `OnboardingShell` for workspace and `Local folder` source rows.
4) Adjust onboarding CSS for stable grids, wrapping/ellipsis and mobile one-column behavior.
5) Update README, `docs/INSTALL.md` and `docs/spec/API_SPEC.md`; validate with focused backend/UI tests and Full DoD.

### Files expected to change
- `internal/runtime/*`, `internal/api/*`
- `ui/src/components/*`, `ui/src/lib/*`, `ui/src/App.test.tsx`, `ui/src/styles.css`
- `README.md`, `docs/INSTALL.md`, `docs/spec/API_SPEC.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] Claude readiness works when only `claude` is on PATH; explicit `ACP_CLAUDE_CMD` still wins.
- [x] Qwen/Codex command defaults remain `qwen` and `codex`.
- [x] Workspace and local repo path dropdowns render suggestions, fill fields and keep manual typing intact.
- [x] Path suggestion API rejects invalid kind, NUL/traversal/root escape and unsafe symlink escape.
- [x] Onboarding has no horizontal overflow at `1440x960`, `1024x768`, `390x1200`.
- [x] Direct-mode server startup remains unchanged.

### Progress log
- 2026-06-05: Started implementation slice after owner reported normal `claude` install was not discovered by `claude-code` readiness and requested onboarding path dropdowns/rendering polish.
- 2026-06-05: Implemented provider command resolution, `/api/onboarding/path-suggestions`, workspace/repo path comboboxes, onboarding rendering polish and provider/executable readiness copy; focused backend/UI/doc sync tests passed before Full DoD.
- 2026-06-05: PR #104 merged into `main` at `47a691e` after green PR checks and green post-merge `main` CI; remote feature branch was deleted.
- 2026-06-06: Started `v0.1.6` CI-only beta release metadata branch to publish PR #104 changes in downloadable release artifacts. Fresh trusted `release-fast` remains skipped, so canonical `RELEASE READY` is not claimed.

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
- 2026-05-21: Clean-worktree Claude frontend diagnostic on `cc223d8` (`smoke-tiny-bank-claude-frontend-20260521T062833Z`) confirmed the backend `step2.asis_docs` and multi-repo semantic fixes: backend hard-passed with no `runtime_contract_failed`, `runner_unavailable`, `runtime_timeout`, old quality-gate, or `cross_repo_missing` hits. The remaining product blocker is frontend-triggered `init.step0.constitution`: focused repair wrote `constitution-draft.json` and `charter-overview.md` correctly, but wrote `baseline-subagents.yaml` under a sibling path where `/frontend/claude-code/` was rewritten to `/frontend-claude-code/`; runtime correctly rejected the missing in-write-set draft as `runtime_contract_failed`. The follow-up keeps artifact-only success and write-set validation unchanged, removes permissive "equivalent writes" wording, and makes normal/repair draft commands assign exact absolute `write_root`/`draft_root` once, then write through shell variables so providers do not manually retype long slash-separated paths.
- 2026-05-21: Canonical `release fast` on clean commit `d8ddf7d` (`release-fast-20260521T093333Z`) completed all backend and frontend surfaces with no runtime/provider/infra blockers, but verdict stayed `FAIL`: qwen/codex multi-path baseline and parallel-default runs had `analysis:cross-repo-missing` because the `FIRST VALIDATOR VERDICT COMMAND` wrote an empty `findings/questions` skeleton and runtime stopped after the valid artifact before late cross-repo instructions could take effect. The follow-up keeps `analysis:cross-repo-missing` strict and does not add ACP-side fallback; instead multi-repo validator first-action and focused repair skeletons now include one PASS-compatible cross-repo finding and one question with repo/path provenance, so qwen/codex first valid artifacts satisfy the provider-facing semantic contract.
- 2026-05-22: Targeted multi-path diagnostic on `38172b1` (`regres-fast-bank-openedx-20260522T021739Z`) passed for qwen/codex and confirmed the validator first-action cross-repo skeleton. Canonical `release fast` (`release-fast-20260522T065332Z`) then passed baseline backend for all providers and frontend init/cancel for Claude/Codex, but qwen frontend init failed because two `init.step1.collect` focused pair repairs returned qwen stream output ending in `[API Error: Premature close]` with process `success` and no artifacts. The run was stopped after the first failed sweep because release PASS was already impossible. The remediation keeps artifact-only success, canonical matrices, schemas, provider contracts, and timeouts unchanged: qwen now treats transient provider API/transport text during collect pair repair as a warning retry signal, retries that focused repair once, and classifies exhausted no-artifact API repair as `runner_unavailable` rather than stale `runtime_contract_failed`.
- 2026-05-23: Fresh `v0.1.1` release candidate gate on main `488e173` (`release-fast-20260523T101656Z`) failed: qwen passed init but `refresh.step2.asis_docs` focused draft repair returned `[API Error: Connection error ... network socket disconnected before secure TLS connection]` with process `success` and no `asis-draft-manifest.json`; the runtime correctly reported `runner_unavailable`, but the existing transient retry covered only collect-pair repairs and did not recognize the TLS/socket transcript as retryable. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: qwen transient provider-unavailable focused retry now also covers draft-artifact repairs (`step0/2/4`) and recognizes connection/TLS/socket API text; exhausted no-artifact repair still fails as `runner_unavailable`. The same release attempt also showed host/network operational failures (`git clone` TLS errors, qwen/codex provider connectivity, occasional Claude smoke timeout), so a new release tag remains blocked until this fix is committed and a fresh clean-worktree release-fast PASS is produced on a stable trusted host.
- 2026-05-23: Fresh gate on main `4147d2c` (`release-fast-20260523T113202Z`) advanced past the transient draft-repair failure but hit a new qwen frontend blocker in `init.step2.asis_docs`: required as-is draft artifacts (`asis-draft-manifest.json`, `overview.md`, `summary.md`, `architect-summary.md`) were valid, while qwen kept streaming and mutating `architect-summary.md` until the global step runtime timeout. This is not a silence/preflight/provider-auth failure; it is an active overrun after valid draft artifacts. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: only `qwen-code` draft steps (`step0/2/4`) get a bounded valid-artifact controlled stop, so continued stream/mutation after a valid manifest+file set is accepted through the existing artifact validation gate instead of waiting for `runtime_timeout`. Collect/validator, Claude, and Codex behavior remain unchanged.
- 2026-05-24: Fresh release-prep gate on `e4db0ac` (`release-fast-20260524T081756Z`) passed the single-repo sweeps and multi-path backend, then exposed a Claude frontend product blocker in `init.step1.collect`: manifest-only repair for `devstack-docs` wrote a schema-valid `shard-pack-manifest.json` but the provider kept running and mutating it, while repair policy had `valid_artifact_stop_window_ms=0`; the old runtime could wait until timeout instead of accepting the valid repair artifact through validation. The remediation keeps canonical matrices, schemas, provider contracts, and timeout budgets unchanged: focused repair policies now add a short valid-artifact controlled stop for collect/validator/draft repairs across providers; partial/invalid repair artifacts remain `runtime_contract_failed`.
- 2026-06-01: Published `v0.1.1` as a GitHub prerelease after PR #88 (`7548fdc4`) updated release notes for the owner-approved no-gate path. Main CI passed, release workflow `26761305996` succeeded, and install smoke passed for `ACP_VERSION=v0.1.1` plus authenticated `ACP_VERSION=latest`. Release URL: `https://github.com/GrinRus/ProvenArch/releases/tag/v0.1.1`. Fresh trusted-machine `release-fast` was skipped by owner decision; no final-tag `release_verdict_*.json` exists, so canonical `RELEASE READY` is not claimed.
- 2026-06-04: Published `v0.1.3` as a GitHub prerelease after PR #93 fixed clean UI/onboarding QA issues and PR #94 updated release metadata. Tag `v0.1.3` points to `11dc504bfdaad4e8cb14c0a843189d25ceb2f1f8`; release workflow `26935595541` passed after one rerun of a transient Linux `codexcode` stub test failure (`text file busy`). Install smoke passed for `ACP_VERSION=v0.1.3` and `ACP_VERSION=latest`; release URL: `https://github.com/GrinRus/ProvenArch/releases/tag/v0.1.3`. Fresh trusted-machine `release-fast` was skipped by owner decision; no final-tag `release_verdict_*.json` exists, so canonical `RELEASE READY` is not claimed.

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
EP-20260618-live-e2e-quality-loop

### Context
Medium live E2E validation now uses the canonical non-release `regres long` matrix (`posthog + ftgo`, `RUN_COUNT=1`) with one selected provider at a time. The current loop starts with `codex-code`, then `claude-code`, and repeats the affected provider until execution quality, artifact completeness, and UX/manual evidence are clean. Generated live evidence is diagnostic and must not be committed.

### Goals (must have)
- [x] Keep product UI/API behavior, canonical matrix files, and curated repo files unchanged
- [x] Preserve execution/UX/artifact quality separation; `artifact_quality:*` remains telemetry/evidence only
- [x] Fix live-observed draft enrichment no-op/scaffold failure without ACP-side deterministic product synthesis
- [x] Fix frontend snapshot eligibility so failed headless refresh rows cannot start heavy UI smoke as if they were valid snapshots
- [x] Fix live-observed collect pair repair preflight-only/status-only no-op so post-preflight provider output must mutate markdown + manifest before prose
- [x] Replace two-phase collect pair repair preflight/write contract with one-shot provider-authored write-first command
- [x] Tighten normal collect so its first filesystem action writes bounded evidence-backed doc + manifest instead of relying on repair/recovery as the normal success path
- [ ] Produce clean strict medium `codex-code` evidence
- [ ] Produce clean strict medium `claude-code` evidence
- [ ] Record execution, artifact completeness/truthfulness, and UX findings for each rerun

### Non-goals
- [x] Do not add wrapper scripts around `scripts/full-run-batch-matrix.sh`
- [x] Do not bypass provider/path preflight with test-only env flags
- [x] Do not edit `examples/e2e-matrix.regres-long.yaml` or curated repo files to fit the current machine
- [x] Do not treat SWE manual UX/artifact reports as core AOR runtime inputs

### Approach
1) Implement the smallest runtime/reporting/docs/test fix slice for the current live blocker.
2) Run `make contracts`, `make test`, `make lint`, and `make build`.
3) Commit the fix before any live rerun.
4) Run direct `scripts/full-run-batch-matrix.sh` from a clean tree using planner-generated `regres long` provider env.
5) Inspect matrix/profile/run/execution/frontend reports, taskrun quality JSON, raw provider metadata, staged manifests/docs, final/citation indexes, and screenshots/logs.
6) If any P0/P1 execution bug, artifact incompleteness/misleading output, or UX blocker remains, open the next narrow fix slice and repeat.

### Acceptance criteria
- [ ] Latest strict medium `codex-code` run passes selected-provider backend totals with no synthetic provider deficits
- [ ] Latest strict medium `claude-code` run passes selected-provider backend totals with no synthetic provider deficits
- [ ] No unresolved `runtime_contract_failed`, `runner_unavailable`, `runtime_timeout`, `runtime_flow_failed`, or `frontend_failed` remains for the accepted medium evidence
- [ ] No old mixed quality gate artifacts (`quality_report_*`, `quality_gates_failed`, `failure_reason=quality`, `RUN_QUALITY_GATES`) appear in generated evidence
- [ ] Artifact completeness includes planned/succeeded/failed shard accounting, final/citation indexes, proposals, publish readiness, and decision-ready summaries
- [ ] UX evidence is accepted when frontend runs, or explicitly inconclusive/not applicable with no hidden blocker when frontend is skipped

### Progress log
- 2026-06-18: Codex strict medium run `regres-long-posthog-ftgo-20260618T083335Z` failed in `refresh.step2.asis_docs`: `draft_artifact_enrichment` printed analysis-only status and left bootstrap markdown unchanged. Fixed write-first draft enrichment prompt/contract, reporting tests, frontend snapshot eligibility, docs, and committed `7c1df8e`.
- 2026-06-18: Codex rerun `regres-long-posthog-ftgo-20260618T131738Z` proved the draft enrichment fix for `init.step0.constitution`, then exposed a collect pair repair no-op: `collect_pair_repair_preflight` returned bounded evidence, provider printed “I am now writing”/status prose, but did not execute the required filesystem write before stall. The known-bad diagnostic was stopped early to implement the next narrow fix.
- 2026-06-18: Tightened collect pair repair prompt/contract so post-preflight plan/status/analysis-only prose is invalid and the next item must be a filesystem command that writes the evidence-backed markdown plus `shard-pack-manifest.json`. Targeted prompt/adapter tests passed; full DoD passed with pinned Node 22.21.1 (`make contracts`, `make test`, `make lint`, `make build`).
- 2026-06-18: Codex rerun `regres-long-posthog-ftgo-20260618T135110Z` showed the previous fix was still insufficient: `posthog-bin` `collect_pair_repair` executed only the read-only preflight command, emitted no follow-up command or agent message, and stalled before artifacts. The run was stopped as known-bad; next fix removes the separate pair-repair preflight and requires a one-shot write-first provider-authored command.
- 2026-06-18: Replaced collect pair repair's two-phase preflight/write prompt with one-shot write-first provider-authored repair. The prompt now forbids separate `collect_pair_repair_preflight`, requires the first filesystem command to read bounded evidence and write both final artifacts before returning, and keeps manifest-only repair as the only read-only preflight path. Full DoD passed with pinned Node 22.21.1 (`make contracts`, `make test`, `make lint`, `make build`).
- 2026-06-18: Codex rerun `regres-long-posthog-ftgo-20260618T161004Z` proved init could complete only with heavy repair/recovery reliance (`repair_attempts=19`, `pre_artifact_stalls=13`, `post_artifact_stalls=25`), and refresh immediately repeated a normal collect `runtime_stalled_before_artifacts` before `collect_pair_repair`. The run was interrupted as known-bad diagnostic evidence. Current slice tightens normal collect prompt/contract so the first filesystem action must read bounded evidence and write the authored doc + `shard-pack-manifest.json` together, with no read-only preflight, analysis-only prose, broad sweep, fragile templated writer, or reliance on focused repair as the expected path.

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
