# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон, текущие активные планы и инженерный operational mirror.

Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
- `docs/archive/PLANS_ARCHIVE_2026-05.md`
- `docs/archive/PLANS_ARCHIVE_2026-06.md`
- `docs/archive/PLANS_ARCHIVE_2026-07.md`
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
Tracker reconciliation from 2026-07-02 archived implementation-complete plans into `docs/archive/PLANS_ARCHIVE_2026-07.md`. Historical reconciliation evidence remains in `docs/archive/TRACKER_RECONCILIATION_2026-05-07.md`, with older closed plans in the monthly archives listed above.

### Plan ID
EP-20260715-architecture-change-review-ui-migration-wave

### Context
The Architecture Change Review target design and seven reference screens are accepted as the
post-beta direction, but implementation still needs one decision-complete migration wave. The
current React UI is organized around eight stages in `App.tsx`, `StageRail.tsx` and a large
`StagePanels.tsx`; several target trust states also need authoritative API readback before new
screens can present them. The detailed wave plan must map existing Epic `20A–20N` to contract-first
dependencies, current code owners, deterministic QA, one-shell cutover, rollback and the exact
internal/external references used during review.

### Goals (must have)
- [x] Audit current UI orchestration, hooks/components, API/runtime/Git read models and test/release
      infrastructure before fixing delivery order.
- [x] Record one source-of-truth/reference matrix covering target design, schemas/specs, current
      behavior, visual screens, UX evidence, QA policy and official external interaction patterns.
- [x] Convert Epic `20A–20N` into dependency-ordered contract, trust, shell, destination, craft and
      rollout phases without inventing competing slice IDs.
- [x] Fix exact code/test ownership, canonical fixtures, per-PR gates, wave-exit acceptance,
      cutover/rollback and documentation synchronization rules.
- [x] Record that `20B1`, `20C1` and `20F1` are required contract-first PR after current-code
      sufficiency inspection, while `20A` reuses its already landed snapshot foundations.
- [x] Cross-link the migration plan from target/current UI docs, Epic 20, README, architecture and
      stakeholder status without claiming that the new shell is implemented.
- [x] Complete documentation checks and the full deterministic repository DoD.
- [ ] Archive this completed planning slice during the next tracker reconciliation.

### Non-goals
- [x] Do not implement or restyle React UI in this planning slice.
- [x] Do not change API, schema, runtime, Git mutation, queue or artifact behavior.
- [x] Do not introduce a parallel hidden shell, production feature flag or big-bang rewrite.
- [x] Do not run provider-live/release matrices; they remain a trusted-machine pre-release gate.
- [x] Do not turn external console references or generated PNGs into product contracts.

### Approach
1) Reconcile mandatory project docs and inspect current UI/API/test ownership.
2) Resolve target trust assumptions against current payloads and code behavior.
3) Define source priority, delivery dependency graph and current-to-target route/module maps.
4) Specify every existing Epic 20 slice with deliverable, expected files, tests and exit condition.
5) Define deterministic fixtures, task anchors, viewport/a11y gates, embedded-bundle checks and
   live/release boundary.
6) Synchronize discovery surfaces and execute docs plus repository DoD checks.

### Files expected to change
- `docs/UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md`
- `docs/UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md`
- `docs/UI_CONSOLE_V2_DESIGN.md`
- `docs/BACKLOG.md`
- `docs/PLANS.md`
- `docs/STAKEHOLDER_DOC.md`
- `README.md`
- `docs/ARCHITECTURE.md`

### Acceptance criteria
- [x] Migration authority and conflict resolution are explicit; schemas/specs outrank UI plans,
      written target outranks PNG, and README/ARCHITECTURE remain implemented-behavior docs.
- [x] The wave starts with `20A`, then required `20B1`/`20C1`, and delays shell cutover until
      authoritative snapshot/Git/runtime/queue/workflow foundations exist.
- [x] `20I1` is an atomic one-shell cutover with four honest temporary compositions and no hidden
      Console V2 controls; rollback is a slice revert, not a dual-shell toggle.
- [x] Every `20A–20N` slice has an implementation/test handoff and preserves local-first,
      automatic promotion, read-only sources and full-workspace Git semantics.
- [x] Six canonical fixtures, stable task anchors, accessibility/viewport gates and required/live
      test boundaries are fixed.
- [x] The complete visual/internal/external reference set is linkable from one document.
- [x] `git diff --check`, link/path checks, docs synchronization tests and full DoD pass.

### Risks
- A detailed plan can drift into a competing roadmap; the plan keeps `BACKLOG.md` `20A–20N` as the
  only acceptance IDs and uses phases only for coordination.
- The shell can appear ready while underlying truth is client-derived; mandatory contract-first PR
  prevent `20I1` from landing before authoritative readback.
- Visual references can encourage brittle screenshot matching; written behavior, task assertions,
  viewports and deterministic fixtures remain the acceptance basis.
- Live E2E selectors are tied to Console V2; they migrate only at wave exit and never become an
  ordinary merge dependency on providers.

### Progress log
- 2026-07-15: Audited `App.tsx`, shell/stage components, hooks/contracts, Git diff/commit behavior,
  run history/queue persistence, Vitest/Playwright contracts and release runbook. Confirmed that
  full Git inventory, historical runtime mode and pending/supersession readback require
  contract-first PR before their target UI.
- 2026-07-15: Added
  `docs/UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md` with authority rules, all internal and
  official external references, six delivery phases, existing `20A–20N` slice cards, module/route
  maps, fixtures, QA gates, one-shell cutover/rollback and first-slice handoff; synchronized product
  discovery and stakeholder surfaces.
- 2026-07-15: Verified every local migration-plan link, `git diff --check` and
  `go test ./internal/docsync`; full deterministic DoD passed with exact Node 22.21.1:
  `make contracts`, `make test` (Go, 246 Python and 116 UI tests), `make lint`, `make build`.
  `make verify-ui-determinism` and `make verify-ui-dist` also passed; generated embedded UI remained
  unchanged because this planning slice does not edit product code.
- 2026-07-15: Final independent code/docs/QA review closed Git preview-to-mutation TOCTOU,
  launcher-only runtime switching, explicit single-pending queue command semantics, path-only
  `20I1` versus deep-context `20I2`, same-slice cutover docs, executable responsive/a11y fixtures
  and exact mock/live contract files. Epic 20 and target design were synchronized with those
  decisions rather than leaving the migration plan as a competing source.
- 2026-07-15: Re-ran the full DoD after review fixes: contracts, Go tests, 246 Python tests,
  116 UI tests, lint/typecheck and production build passed. WORKTREE UI determinism, embedded
  bundle freshness, local-link validation, `git diff --check` and docs-sync also passed; the final
  read-only code/docs/QA rechecks returned PASS.

### Plan ID
EP-20260715-architecture-change-review-design-baseline

### Context
The exploratory UX audit selected Architecture Change Review as the product backbone, with
attention-first Home, focused Run Studio, Knowledge/Atlas and contextual Evidence Studio modes.
The repository still treats the implemented eight-stage Console V2 design and its screenshots as
the only detailed design baseline. Before implementation starts, the chosen post-beta direction
needs one canonical target specification, project-owned visual references and explicit links from
the current baseline, architecture overview and Epic 20 backlog.

### Goals (must have)
- [x] Locate and reconcile the existing UI design baseline, screenshots, backlog decisions,
      architecture notes, UX evidence and React/CSS source-of-truth surfaces.
- [x] Define the detailed target product model, information architecture, journeys, screens,
      state matrix, interaction rules, responsive behavior and accessibility requirements.
- [x] Define a semantic visual system and reusable component families aligned with the current
      calm palette and plain-CSS implementation boundary.
- [x] Generate and persist seven reference screens for the selected hybrid concept under `docs/assets`.
- [x] Synchronize the V2 baseline, README, architecture overview, stakeholder matrix and Epic 20 with the new target
      concept without claiming that the target UI is already implemented.
- [ ] Archive this completed design-baseline plan during the next tracker reconciliation.

### Non-goals
- [x] Do not implement or restyle React UI code in this design-baseline slice.
- [x] Do not change API, artifact, workspace, runtime or Git mutation contracts.
- [x] Do not introduce persisted review approval or imply selected-file commit semantics.
- [x] Do not treat generated PNG references as pixel-perfect implementation contracts.

### Approach
1) Inspect the current design/doc/assets/code source-of-truth and record current-versus-target status.
2) Specify the Architecture Change Review backbone and supporting product modes from user jobs.
3) Define honest source, execution, evidence and publication state semantics.
4) Define semantic tokens, component anatomy, density and responsive/accessibility rules.
5) Generate one reference image for each key target surface and persist it in the repository.
6) Cross-link the target design from current docs/backlog and run docs/full repository checks.

### Files expected to change
- `docs/UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md`
- `docs/assets/ui-architecture-change-review/*.png`
- `docs/UI_CONSOLE_V2_DESIGN.md`
- `docs/BACKLOG.md`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/STAKEHOLDER_DOC.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] One canonical target document distinguishes implemented V2 behavior from planned post-beta UX.
- [x] Target IA uses `Home / Runs / Knowledge / Changes`, with Setup contextual and Ask global.
- [x] Full first-run, refresh, failure/recovery, review, Q&A and Git publication paths are specified.
- [x] Workspace, execution, evidence and publication axes have explicit empty/partial/error states.
- [x] Visual tokens and component contracts are implementation-ready but do not mutate CSS yet.
- [x] Seven project-owned reference screens are linked from the target design document.
- [x] Existing V2 screenshots remain available as historical design/current-shell references.
- [x] Documentation checks and full DoD pass without product-code or contract changes.

### Risks
- Generated UI references can contain minor text/rendering artifacts; written behavior and component
  contracts remain authoritative over pixels.
- Some target trust states depend on Epic 20 contract-first slices; the spec must label those
  dependencies instead of inventing data or presenting approvals that do not exist.
- Current docs describe implemented eight-stage navigation. Target links must be additive and
  explicitly planned until implementation and E2E migration are complete.

### Progress log
- 2026-07-15: Located the historical V2 design/current-shell baseline at `docs/UI_CONSOLE_V2_DESIGN.md`, historical
  visuals under `docs/assets/ui-console-v2/`, target trust decisions in Epic 20, current UX evidence
  under `reports/`, and implementation sources in `ui/src/components` plus `ui/src/styles.css`.
- 2026-07-15: Added the user-selected Architecture Change Review target spec with explicit
  promotion-vs-Git-acceptance boundary, source identity, active/historical routing, four-axis state
  derivation, migration bridge, semantic tokens, responsive/accessibility rules and seven-screen
  reference set; synchronized V2/README/ARCHITECTURE/BACKLOG/STAKEHOLDER status and IA.
- 2026-07-15: Generated and visually inspected seven consistent 1536x1024 reference screens for
  Guided Setup, Home, Run Studio, Change Review, Evidence Studio, Knowledge Atlas and Publish;
  corrected the only prominent generated-copy defect before persisting assets. Full deterministic
  DoD passed with exact Node 22.21.1: `make contracts`, `make test` (Go, 246 Python and 116 UI
  tests), `make lint`, `make build`; product code and public contracts remained unchanged.

### Plan ID
EP-20260714-ui-ux-redesign-exploration

### Context
The current operator console exposes the full local-first ACP workflow, but its information
architecture and visual hierarchy have accumulated around backend stages and diagnostics.
Before implementation, the product needs an evidence-backed redesign exploration grounded in
the actual end-to-end operator journey, rendered UI, and current patterns from comparable
developer and operations consoles.

### Goals (must have)
- [x] Reconstruct the product problem, primary users, and complete ACP/AOR operator journey from
      canonical docs, code, and rendered UI.
- [x] Audit the current console for task clarity, hierarchy, density, state coverage, recovery,
      accessibility, and trust/evidence workflow issues.
- [x] Research current comparable engineering consoles and extract transferable interaction
      patterns with source links.
- [x] Produce at least three genuinely distinct redesign concepts, including information
      architecture, core surfaces, strengths, risks, and recommendation criteria.
- [ ] Archive this completed research plan during the next tracker reconciliation.

### Non-goals
- [x] Do not implement UI code or change API/schema/workspace contracts in this exploration.
- [x] Do not expand the MVP provider list, hosted scope, or security/compliance scope.
- [x] Do not treat visual styling alone as a complete redesign.

### Approach
1) Read canonical product and pipeline documentation and map the current source architecture.
2) Inspect the rendered onboarding and console surfaces alongside their React implementation.
3) Model the primary, alternate, failure, recovery, and repeat-use journeys.
4) Compare relevant developer/operations consoles and synthesize reusable patterns.
5) Define and evaluate distinct redesign concepts against ACP constraints and operator jobs.

### Files expected to change
- `docs/PLANS.md` only for this research pass.

### Acceptance criteria
- [x] Product and user-job summary is grounded in canonical sources.
- [x] Current-state flow covers onboarding, readiness, analysis, review, Q&A, and publication.
- [x] Current UI findings include missing or weak loading/empty/error/partial/offline/recovery states.
- [x] At least three concepts differ structurally, not only by theme or color.
- [x] Each concept includes a screen model, interaction model, tradeoffs, and fit assessment.
- [x] Final recommendation identifies what to prototype and validate next without changing code.

### Risks
- Repository terminology uses `AOR` internally without a canonical user-facing expansion; this
  exploration treats it as the core ACP runtime/orchestrator flow unless contrary evidence appears.
- A static/code-only review can miss density and responsive defects, so rendered inspection is
  required before final synthesis.

### Progress log
- 2026-07-14: Started product, pipeline, UI, and reference-console research; no product code changes.
- 2026-07-14: Reconstructed the complete onboarding, readiness, init/refresh, review, Q&A,
  recovery, and Git publication journeys from canonical docs, React state, and API behavior.
- 2026-07-14: Inspected desktop and mobile renders and completed all seven deterministic UI mock
  scenarios; identified trust, state-model, density, responsive, and false-affordance defects.
- 2026-07-14: Compared official Temporal, Dagster, GitHub Actions, Prefect, Grafana, Backstage,
  Argo, and Sentry interaction patterns and synthesized five structurally distinct concepts.
- 2026-07-14: Selected an architecture-change-review backbone with attention-first Home, focused Run
  Studio, Knowledge/Atlas, and contextual Evidence workbench as the recommended prototype direction.

### Plan ID
EP-20260713-live-e2e-codex-model-pin

### Context
Live E2E/release runs should compare stable provider surfaces. The user asked to pin Codex runs to `gpt-5.5` with extra-high reasoning while leaving qwen and claude on their installed CLI defaults. Existing Codex runtime ignores user `config.toml` by design, so the pin must be explicit runtime/preflight argv, not ambient config.

### Goals (must have)
- [x] Add env-driven Codex `--model` / reasoning config args without changing qwen/claude defaults.
- [x] Make live E2E default Codex env `ACP_CODEX_MODEL=gpt-5.5` and `ACP_CODEX_REASONING_EFFORT=xhigh`.
- [x] Ensure preflight probes and artifact smoke use the same Codex model surface as runtime.
- [x] Update tests and live E2E docs.
- [ ] Archive this completed slice during the next tracker reconciliation.

### Non-goals
- [x] Do not extend `workspace.yaml` runtime profile schema in this slice.
- [x] Do not pin qwen or claude models.
- [x] Do not edit canonical matrix repo selections or add a wrapper over `scripts/full-run-batch-matrix.sh`.

### Approach
1) Add optional Codex model/reasoning args in the adapter from env.
2) Set live batch/matrix/generator defaults and pass them through child backend/frontend runs.
3) Align preflight compatibility checks with the explicit live Codex model.
4) Update targeted tests and docs.

### Files expected to change
- `internal/runtime/codexcode/runner.go`
- `scripts/write-batch-preflight.py`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/live-e2e-plan.py`
- focused tests and live E2E docs

### Acceptance criteria
- [x] Codex live E2E requests `gpt-5.5` with `model_reasoning_effort="xhigh"`.
- [x] qwen/claude invocations remain model-default.
- [x] Old Codex CLI compatibility blocker follows the explicit live Codex model, not user config.
- [x] Focused Go/Python tests pass.

### Risks
- A host with an older Codex CLI now fails earlier in live preflight because the default requested Codex model is explicit.

### Progress log
- 2026-07-13: Implemented slice in a focused patch; full DoD pending after targeted verification.
- 2026-07-13: Focused Go/Python/shell checks passed; full DoD (`make contracts test lint build`) passed with `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin` after installing UI deps in this worktree.

### Continuous Backlog Queue Policy

Epics 19, 20 and 21 are complete in `main`. `docs/BACKLOG.md` remains the reference/acceptance
backlog; this file selects focused active slices. The current release blocker is Epic 18 R3:
trusted-machine composite evidence over fresh canonical release matrices. New Wave 1 product work
requires a separate owner-selected ExecPlan after R3.

Allowed next workstreams:
- Epic 19/20/21 archive or reconciliation bookkeeping only; no additional engineering slice remains.
- Epic 18 R3 release validation, only when trusted-machine/provider/path prerequisites are satisfied.
- `EP-20260508-oss-readiness-hardening`: owner/admin verification for residual GitHub repository settings only.
- `EP-20260507-cleanup-owner-decisions`: owner-gated retain/remove/dedupe decisions only; retain by default until decisions exist.
- New Wave 1 or product work: create a fresh decision-complete ExecPlan before implementation.

Task selection rules:
- Completed plans whose only remaining item is owner review, merge/archive bookkeeping, or historical evidence retention are not next engineering work.
- Owner-decision and trusted-host/live-release items remain explicit blockers; do not run or edit them as normal backlog tasks without the required owner/trusted-machine prerequisites.
- Each selected slice gets a decision-complete ExecPlan/update before implementation, one focused implementation pass, self-review/fix loops, Full DoD (`make contracts`, `make test`, `make lint`, `make build`), then one commit.

### Plan ID
EP-20260713-epic-19-pr1-remediation

### Context
Epic 19 was implemented as one review program with backlog slices `19A..19X` used as internal
commit and checklist boundaries. It is merged into `main` at `02716bb`; remaining work for this
plan is archive/reconciliation bookkeeping only.

### Goals (must have)
- [x] Add a shared atomic persistence primitive for workspace-owned files: temporary file,
      flushed data, synced parent directory and atomic rename.
- [x] Preserve a last-good copy for critical persisted state so malformed or partial current
      files do not collapse into empty history.
- [x] Stop silently ignoring run-history persistence failures.
- [x] Move run history, shard summaries and runtime checkpoint writes onto the shared primitive
      where the existing ownership boundary is clear.
- [x] Add fault-injection coverage for write, sync and rename failure points.
- [x] Keep the PR deterministic: no live provider dependency and no release matrix changes.
- [x] Continue PR-1 with `19B` transactional canonical promotion after `19A` review/commit
      boundary is stable.
- [x] Continue PR-1 with `19C` async panic isolation after `19B` review/commit boundary
      is stable.
- [x] Continue PR-1 with `19D` shutdown coordination after `19C` review/commit boundary
      is stable.
- [x] Continue PR-1 with `19E` coherent service/session generation after `19D` review/commit
      boundary is stable.
- [x] Continue PR-1 with `19F` fresh unpinned git_url resolution after `19E` review/commit
      boundary is stable.
- [x] Continue PR-1 with `19G` minimum collect evidence contract after `19F` review/commit
      boundary is stable.
- [x] Continue PR-1 with `19H` symmetric document/citation validation after `19G`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19I` historical run artifact snapshots after `19H`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19J` request-scoped UI detail state after `19I`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19K1` run mutation acknowledgement after `19J`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19K2` Q&A provisional run ordering after `19K1`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19L` editor draft safety after `19K2`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19M` reproducible embedded UI build after `19L`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19O` locked contract validator tooling after `19M`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19N` composite release verdict gate after `19O`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19P` Step 1 card enrichment after `19N`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19Q` generic refresh semantic guard after `19P`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19R1` ARIA tabs controller after `19Q`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19R2` keyboard path combobox after `19R1`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19R3` accessible async announcements after `19R2`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19S1` confirmed shell dead-code cleanup after `19R3`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19S2` ShellCheck in canonical lint after `19S1`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19S3` required PR lint routing after `19S2`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19T` logs endpoint smoke coverage after `19S3`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19U` deterministic mock Playwright CI after `19T`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19U2` optional V8 coverage baseline after `19U`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19V` Python runtime pinning after `19U2`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W1` runtime-draft wrapper cleanup after `19V`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W2` sharding wrapper cleanup after `19W1`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W3` provider argument wrapper cleanup after `19W2`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W4` docflow compatibility helper cleanup after `19W3`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W5a` review diff residual cleanup after `19W4`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W5b` model store residual cleanup after `19W5a`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W5c` orchestrator quality residual cleanup after `19W5b`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W5d` reports compiler residual cleanup after `19W5c`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19W5e` prompt-contract residual cleanup after `19W5d`
      review/commit boundary is stable.
- [x] Continue PR-1 with `19X` UI dead-surface cleanup after `19W5e`
      review/commit boundary is stable.
- [x] Continue PR-1 with final Epic 19 docs/backlog reconciliation after `19X`
      review/commit boundary is stable.
- [x] Complete PR-1 review/push/merge into `main` before starting Epic 20.
- [ ] Archive this merged Epic 19 plan during the next tracker reconciliation.

### Non-goals
- [ ] Do not implement `19B` transactional canonical promotion in the first pass.
- [ ] Do not change public artifact schemas in `19A`.
- [ ] Do not change provider selection, live matrix inputs or hosted/security scope.
- [x] Do not open Epic 20 implementation work until Epic 19 is merged or explicitly rebased after merge.

### Approach
1) Add the persistence primitive in `internal/workspace` and expose it through the existing
   `workspace.Root` boundary.
2) Add focused unit tests for atomic write success, simulated failures and last-good recovery.
3) Replace the highest-risk direct writes in run-history/shard-checkpoint surfaces first.
4) Propagate persistence errors to callers where the current code drops them.
5) Run targeted Go tests, then expand verification as the touched surface grows.
6) Continue with `19B` only after `19A` has tests and no known crash-consistency regressions.

### Files expected to change
- `internal/workspace/fs.go`
- `internal/workspace/*_test.go`
- `internal/orchestrator/service_runs.go`
- `internal/orchestrator/sharding_artifacts.go`
- `internal/orchestrator/restart_reconcile*` if current checkpoint writes require migration
- `docs/PLANS.md`

### Acceptance criteria
- [x] Atomic writes never leave a partial target file after injected data-write or rename
      failures.
- [x] Parent directory sync failures are surfaced as errors in tests.
- [x] A malformed current run history with a valid last-good copy recovers the last-good state
      and records a diagnostic path instead of returning an empty history.
- [x] Run-history persistence errors are returned or logged through existing lifecycle status
      paths instead of being discarded.
- [x] Targeted packages pass under `go test`, and the full PR eventually passes
      `make contracts`, `make test`, `make lint`, `make build`.

### Risks
- Some call sites currently use best-effort persistence during terminal cleanup; turning those
  into hard errors must not strand active run slots.
- Atomic rename is only atomic within the same filesystem; temp files must stay beside the
  target.
- Last-good files are recovery aids, not a second source of truth. Reads must prefer current
  content when it is valid.

### Progress log
- 2026-07-13: Created branch `codex/epic-19-code-quality-remediation`; selected `19A` as the
  first implementation slice and recorded this ExecPlan before code changes.
- 2026-07-13: Implemented `19A` atomic workspace writes, run-history `.last-good` recovery,
  persistence error surfacing for async queue start, fault-injection tests and docs sync.
  Verification passed: `go test ./internal/...`, `go test ./...`, Python harness unit suite
  (230 tests), focused `go test -race` lifecycle tests and Go build. Full `make contracts`,
  `make test`, `make lint` and `make build` are blocked locally before repo checks by missing
  Node.js `22.21.1` (`/opt/homebrew/bin/node` is `25.9.0`).
- 2026-07-13: Installed exact Node.js `22.21.1` outside the repo and reran the canonical
  `19A` gate with `ACP_NODE_TOOL_CANDIDATES`. `make contracts`, `make test`, `make lint` and
  `make build` all pass; `make build` produced no tracked embedded-UI drift.

### Plan ID
EP-20260713-epic-19-19b-transactional-promotion

### Context
`19B` follows the committed `19A` atomic persistence foundation. Current docflow promotion
copies validated staged documents into canonical `reports/*`/`proposals/*`, removes stale
managed files, then rebuilds `model/*` and `reports/diagrams/*` directly in the live
workspace. A mid-promotion copy/remove/model/diagram failure can therefore leave mixed
old/new bytes visible under canonical paths.

### Goals (must have)
- [x] Make canonical promotion transactional for managed generated surfaces: prepare a complete
      promotion generation outside canonical paths, validate it, then activate it with a journaled
      rollback path so a failed promotion leaves either the previous complete generation or the new
      complete generation visible, never a mixed generation.
- [x] Continue PR-1 with `19C` async panic isolation after `19B` review/commit boundary is stable.
- [x] Continue PR-1 with `19D` shutdown coordination after `19C` review/commit boundary is stable.
- [x] Continue PR-1 with `19E` coherent service/session generation after `19D` review/commit boundary is stable.
- [ ] Continue PR-1 with `19F` fresh unpinned git_url resolution after `19E` review/commit boundary is stable.

### Non-goals
- [ ] Do not change public artifact schemas, final-run-index shape or provider contracts.
- [ ] Do not change live provider selection, release matrices or live E2E gates.
- [ ] Do not introduce user authentication, hosted mode or security enforcement.
- [ ] Do not refactor unrelated report/model compiler behavior beyond the promotion boundary.

### Implementation
1) Build a run-scoped promotion generation under `reports/taskruns/<run_id>/staging/` rather
   than writing any canonical managed path directly.
2) Populate the generation with all `finalRunIndex.CanonicalDocuments` from their staged paths,
   rebuild `model/entities` and `model/edges` inside that generation, and compile diagrams
   against the generated model before canonical activation.
3) Treat managed canonical replacement surfaces as generation roots:
   `reports/as-is`, `reports/coverage`, `reports/findings`, `reports/agent-outputs`,
   `reports/diagrams`, `proposals`, `model/entities`, `model/edges`.
4) Activate `reports/changelog/*` final-index draft files through the same journal as individual
   files, while preserving existing changelog history instead of replacing the whole directory.
5) Validate that every indexed canonical document exists in the generation and that no indexed
   document targets an unmanaged canonical surface.
6) Activate the generation by renaming existing managed roots to a journal backup and renaming
   generation roots into place. On any forward activation failure, rollback processed roots in
   reverse order before returning the error.
7) Remove the redundant post-promotion runtime draft copy in proposals handling; draft outputs
   are already staged into the final run index and must be activated only through the
   transactional promotion path.

### Interfaces
No public API or schema changes. Internal-only helpers may be added to
`internal/orchestrator/docflow_promotion.go` for promotion generation, validation, activation
and test fault injection.

### Tests
- Existing promotion tests keep proving FAIL verdict rejection and stale managed-file removal.
- Add failure-injection coverage across promotion copy/build/activation operations.
- For each injected failure point, assert canonical generated surfaces remain exactly the
  previous complete generation; when no injected failure fires, assert the new generation is
  complete.
- Include reports, proposals, model and diagrams in the old/new generation assertions.

### Docs/fixtures
- Update `docs/ARCHITECTURE.md` and `docs/STAKEHOLDER_DOC.md` for transactional managed
  promotion semantics and operator recovery expectations.
- No schema/example/fixture sync is required because public contracts do not change.

### Acceptance
- [x] `go test ./internal/orchestrator` covers rollback and successful activation.
- [x] `go test ./internal/...` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no canonical writes happen before generation validation and
      activation.

### Progress log
- 2026-07-13: Implemented `19B` transactional canonical promotion. Promotion now builds a
  run-scoped generation, validates indexed documents, rebuilds model/diagrams in staging,
  activates managed roots with journaled rollback, activates `reports/changelog/*` draft files
  through per-file journal entries, removes stale artifact registry entries only after successful
  activation, and removes the redundant post-promotion draft copy. Regression coverage injects
  failures across copy/model/diagram/activation operations and verifies old-or-new complete
  generation semantics. Verification passed: `go test ./internal/orchestrator`,
  `go test ./internal/...`, `go test ./internal/docsync`, and full DoD with exact Node 22.21.1:
  `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260713-epic-19-19c-async-panic-isolation

### Context
`19C` follows committed `19B`. Synchronous `Service.Run` currently terminalizes panic failures
and re-panics, preserving caller-visible panic semantics. The async path starts `runWithID` in a
goroutine and then calls `finishAsyncRun`; if the runner panics, the goroutine can skip
`finishAsyncRun`, leave `activeRunID`/cancel state occupied, block a queued pending run, and crash
the service process instead of recording a terminal async failure.

### Goals (must have)
- [x] Isolate async runner panics at the outer goroutine boundary so the service process stays
      alive, the panicked run gets terminal `failed/internal_failure` history, and
      `finishAsyncRun` always releases the slot and starts a pending run.
- [x] Preserve synchronous `Service.Run` panic behavior: panic still propagates to the caller
      while existing terminal history semantics remain intact.
- [x] Continue PR-1 with `19D` shutdown coordination after `19C` review/commit boundary is stable.
- [x] Continue PR-1 with `19E` coherent service/session generation after `19D` review/commit boundary is stable.
- [ ] Continue PR-1 with `19F` fresh unpinned git_url resolution after `19E` review/commit boundary is stable.

### Non-goals
- [ ] Do not change normal runtime error classification or cancellation semantics.
- [ ] Do not change queue/debounce policy beyond releasing slots after async panic.
- [ ] Do not change provider contracts, public schemas, UI contracts or live matrices.

### Implementation
1) Wrap only the goroutine body in `launchAsyncRun` with `defer`.
2) Ensure the defer calls `finishAsyncRun` exactly once for every async run, including panics.
3) Recover panics at the async goroutine boundary, terminalize the run as
   `internal_failure`/`run failed: panic` if `runWithID` has not already done so, and do not
   re-panic from the goroutine.
4) Keep `runWithID` synchronous panic semantics unchanged: its existing panic guard may
   terminalize and re-panic, so direct callers still observe a panic.
5) Add tests for async init panic and queued pending-run continuation after the active run
   panics.

### Interfaces
No public API/schema changes. Internal-only lifecycle helpers may be added to
`internal/orchestrator/service_runs.go` if needed.

### Tests
- Async panic runner: `StartAsyncRun` returns a run ID, the service remains usable, run history
  records terminal `failed/internal_failure`, and the active slot is released.
- Active async panic with queued pending run: pending run starts after the panicked active run is
  finalized.
- Existing synchronous panic test remains unchanged and continues to require caller-visible panic.

### Docs/fixtures
- Update `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md` and `docs/STAKEHOLDER_DOC.md`
  only for lifecycle/recovery behavior. No fixtures or schemas change.

### Acceptance
- [x] `go test ./internal/orchestrator` covers async panic terminalization and pending-run
      continuation.
- [x] `go test ./internal/...` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no synchronous panic masking and no double `finishAsyncRun`.

### Progress log
- 2026-07-13: Implemented `19C` async panic isolation. `launchAsyncRun` now wraps the outer
  goroutine with a defer that recovers runner panics, terminalizes the run as
  `failed/internal_failure` when needed, and always calls `finishAsyncRun` to release active
  slots/cancel state and launch pending work. Direct `Service.Run` panic behavior remains
  caller-visible through the existing runWithID terminal guard. Regression coverage verifies async
  panic terminalization, service reuse after panic, pending-run continuation, and existing sync
  re-panic semantics. Verification passed: targeted lifecycle tests, `go test ./internal/orchestrator`,
  `go test ./internal/...`, `go test ./internal/docsync`, and full DoD with exact Node 22.21.1:
  `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260713-epic-19-19d-server-owned-shutdown

### Context
`19D` follows committed `19C`. `cmd/acp run` already uses `signal.NotifyContext`, but
`cmd/acp serve` still passes `context.Background()` into `api.Server.Serve`. API serve shutdown
uses `http.Server.Shutdown(context.Background())`, and orchestrator `Service` has cancel maps for
active async runs but no closed/shutdown state. A process-level signal can therefore leave active
runs/provider processes outside a bounded service-owned shutdown path, and post-shutdown API
mutations can still enqueue new run writes.

### Goals (must have)
- [x] Add bounded service-owned shutdown for async runs: reject new starts after shutdown,
      cancel active run contexts, terminalize queued pending runs, release lifecycle state, and
      wait for active terminal state until the shutdown context expires.
- [x] Wire `api.Server.Serve` to call bounded HTTP shutdown and orchestrator shutdown on
      context cancellation.
- [x] Wire `cmd/acp serve` to a signal-aware context for SIGINT/SIGTERM/SIGHUP.
- [x] Continue PR-1 with `19E` coherent service/session generation after `19D` review/commit boundary is stable.
- [ ] Continue PR-1 with `19F` fresh unpinned git_url resolution after `19E` review/commit boundary is stable.

### Non-goals
- [ ] Do not change runtime provider selection, pipeline semantics or queue/debounce behavior
      except at shutdown.
- [ ] Do not redesign process execution; provider process groups remain owned by existing
      `providercommon` command context/process-group adapters.
- [ ] Do not change public API schemas or frontend behavior.

### Implementation
1) Add `ErrServiceClosed`, `Service.Shutdown(ctx)` and `Service.Close()` in
   `internal/orchestrator/service_runs.go`.
2) Add a `closed` flag under `Service.mu`; `StartAsyncRun` returns `ErrServiceClosed` once set.
3) During shutdown, mark pending queued run as `failed/run_canceled`, cancel active run contexts,
   and wait/poll for the active run to reach terminal state until `ctx.Done()`.
4) Ensure `finishAsyncRun` does not launch pending work after `closed=true`.
5) Add `api.Server.Shutdown(ctx)` and make `Serve` use a bounded timeout for both HTTP and
   orchestrator shutdown.
6) In `cmd/acp serve`, create `signal.NotifyContext` and pass it to `server.Serve` in both
   launcher and workspace modes.

### Interfaces
Internal API only: `orchestrator.Service.Shutdown`, `orchestrator.Service.Close` and
`api.Server.Shutdown`. Public HTTP/schema contracts do not change.

### Tests
- Service shutdown cancels an active blocking runner and persists terminal canceled history.
- Service shutdown fails a queued pending run and prevents it from starting.
- `StartAsyncRun` after shutdown returns `ErrServiceClosed`.
- API `Serve` returns after context cancellation and calls service shutdown.

### Docs/fixtures
- Update `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md` and `docs/STAKEHOLDER_DOC.md`
  for server-owned shutdown semantics. No fixtures or schemas change.

### Acceptance
- [x] `go test ./internal/orchestrator` covers service shutdown lifecycle.
- [x] `go test ./internal/api` covers serve-context shutdown.
- [x] `go test ./internal/...` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms post-shutdown starts are rejected and pending work does not launch
      after `closed=true`.

### Progress log
- 2026-07-13: Implemented `19D` server-owned shutdown. `acp serve` now passes a
  SIGINT/SIGTERM/SIGHUP-aware context into API serving; API `Serve` performs bounded HTTP and
  orchestrator shutdown and waits for service cleanup on context cancellation. Orchestrator
  `Service.Shutdown`/`Close` set a closed flag, reject post-shutdown starts, cancel active run
  contexts, terminalize queued pending runs as `run_canceled`, and prevent pending launch after
  `closed=true`. Existing provider process-group cleanup remains driven by runtime context
  cancellation. Verification passed: targeted orchestrator/API/docsync tests, `go test ./internal/...`,
  and full DoD with exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260713-epic-19-19e-coherent-service-generation

### Context
`19E` follows committed `19D`. Launcher mode can select a workspace, select a runtime and then
serve normal workspace-bound endpoints from one process. Current API handlers mostly read
`workspace`, `service` and `runtimeConfig` through independent getters, while onboarding/runtime
mutation handlers can replace session state. That leaves room for one request to observe a
mixed generation, for runtime/profile mutations to race with active async work, and for
onboarding/direct-mode status to report desired runtime state rather than the service generation
that will actually handle new runs.

### Goals (must have)
- [x] Add a request-scoped immutable API session snapshot containing the selected workspace,
      orchestrator service, runtime config and generation metadata.
- [x] Make workspace/runtime session replacement coordinated under one server lock so handlers
      cannot combine stale service ownership with fresh workspace/runtime state.
- [x] Reject workspace or runtime replacement while the current service has active or queued
      async work, returning an explicit conflict instead of silently swapping ownership.
- [x] Make onboarding/direct-mode runtime status report the effective runtime config for the
      current service generation.
- [x] Preserve existing local-first MVP boundaries: no provider-list changes, hosted mode,
      schema migration or Epic 20 UI redesign.
- [ ] Continue PR-1 with `19F` fresh unpinned git_url resolution after `19E` review/commit
      boundary is stable.

### Non-goals
- [ ] Do not introduce UI hot restart or process restart orchestration.
- [ ] Do not change pipeline semantics, queue/debounce rules or provider execution behavior.
- [ ] Do not change public artifact schemas or runtime provider IDs.
- [ ] Do not start Epic 20 before Epic 19 is complete and merged.

### Implementation
1) Add a small `serverSessionSnapshot` value in `internal/api` and use it in request handlers
   that need workspace/service/runtime state together.
2) Add an orchestrator lifecycle query that reports whether a service has active or pending async
   work without exposing mutable internals.
3) Replace independent workspace/service/runtime reads in high-risk handlers with one snapshot
   read, especially run start/status/log/artifact/review/Git/runtime/onboarding surfaces.
4) Gate onboarding workspace selection and runtime selection/profile mutations with a
   `serviceHasInFlightWork` conflict check before replacing session state or changing runtime
   config used by future service generations.
5) Keep direct-mode construction as a ready generation from process CLI config; launcher mode
   creates a new generation only after a workspace has been opened and runtime selected.
6) Update docs for the operator-visible conflict behavior only; no schema or fixture sync is
   expected unless implementation proves a public contract change is unavoidable.

### Interfaces
No schema changes. HTTP error surface may add conflict errors such as
`workspace_switch_conflict` or `runtime_switch_conflict` for mutations attempted during active or
queued runs.

### Tests
- API snapshot/readback: direct-mode onboarding status reports the effective runtime config from
  the current generation.
- Workspace switch conflict: selecting a different workspace while an async run is active is
  rejected and leaves the original service/run visible.
- Runtime switch conflict: selecting a new runtime while active or pending work exists is
  rejected and leaves effective runtime readback unchanged.
- Race-focused coverage: concurrent polling/status reads and runtime/workspace mutation attempts
  are safe under `go test -race ./internal/api`.

### Docs/fixtures
- Update `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md` and `docs/STAKEHOLDER_DOC.md` for
  session-generation conflict semantics if the HTTP conflict behavior is implemented.
- No schemas, examples or fixtures should change.

### Acceptance
- [x] Targeted `go test ./internal/api` covers snapshot/coherence conflicts.
- [x] Race-focused `go test -race ./internal/api` passes for the new lifecycle tests.
- [x] `go test ./internal/...` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms handlers use coherent snapshots where workspace/service/runtime state
      must agree and no Epic 20 behavior leaked into this slice.

### Progress log
- 2026-07-13: Implemented `19E` coherent service/session generation. API handlers now take a
  request-scoped `serverSessionSnapshot` for workspace/service/runtime state, server session
  mutations advance a generation under one lock, onboarding workspace/runtime switches and
  runtime profile/manifest writes return conflict while the current service has active or queued
  async work, and direct/onboarding status reports the effective runtime config for the current
  generation. Orchestrator exposes read-only `HasInFlightRun` for API conflict checks.
  Regression coverage verifies workspace switch conflict, runtime switch conflict, unchanged
  manifest on runtime profile conflict, effective runtime readback and concurrent polling/mutation
  safety under the race detector. Verification passed: `go test ./internal/api ./internal/orchestrator`,
  `go test -race ./internal/api`, `go test ./internal/...`, and full DoD with exact Node 22.21.1:
  `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260713-epic-19-19f-fresh-git-url-resolution

### Context
`19F` follows committed `19E`. Workspace `git_url` sources are ACP-owned caches under
`.acp/repos`, but an unpinned source can currently keep reading the previously checked-out cache
after the remote default branch advances. That makes refresh/collect evidence stale even though
the operator declared a remote source rather than a pinned revision. The fix must keep
user-owned `path` checkouts non-mutating and must not change the `workspace.yaml` schema.

### Goals (must have)
- [x] For unpinned `git_url` repos, fetch the remote, resolve the remote default `HEAD`, and
      force the ACP-owned cache to the exact resolved commit before analysis reads it.
- [x] Preserve pinned `ref` behavior: an explicit branch/tag/SHA continues to select that ref
      rather than the remote default `HEAD`.
- [x] Expose the exact resolved commit SHA in resolver output and run evidence when a fetch
      occurred.
- [x] Keep `path` sources read-only: validation may compare refs and warn, but must not mutate
      user checkouts.
- [x] Add no live-network dependency; tests use local temporary repositories and bare remotes.
- [x] Continue PR-1 with `19G` minimum collect evidence contract after `19F` review/commit
      boundary is stable.
- [x] Continue PR-1 with `19H` symmetric document/citation validation after `19G`
      review/commit boundary is stable.
- [ ] Continue PR-1 with `19I` historical run artifact snapshots after `19H`
      review/commit boundary is stable.

### Non-goals
- [ ] Do not change `workspace.yaml` schema or provider list.
- [ ] Do not introduce a new repository credential store or hosted source resolution plane.
- [ ] Do not start `19G` collect contract work or Epic 20 UI trust work in this slice.
- [ ] Do not rewrite user-owned `path` repositories to configured refs.

### Implementation
1) Extend `workspace.ResolvedRepo` with optional `resolved_sha`, keeping it omitted for dry
   validation or unresolved sources.
2) In `resolveGitURLRepo`, after clone/fetch, resolve either the explicit `ref` or the remote
   default `HEAD` to an exact commit SHA.
3) For unpinned `git_url`, update the local `origin/HEAD` view from the remote default and
   `reset --hard` the ACP-owned cache to the resolved SHA. For pinned refs, checkout the explicit
   ref and record its resolved SHA without switching to remote default.
4) Add resolved repo evidence to run logs after fetch-backed validation, so run artifacts/logs
   preserve the exact source revisions used for analysis.
5) Update TypeScript validate response typing for the optional `resolved_sha` readback; UI
   display remains best-effort and does not become a new workflow gate in this slice.
6) Update README, `docs/spec/WORKSPACE_SPEC.md`, `docs/ARCHITECTURE.md` and testing docs only
   for actual source freshness/readback semantics.

### Interfaces
No schema changes. Public API readback adds optional `resolved_repos[].resolved_sha` in workspace
validation responses when ACP fetched a `git_url` source; run logs add `source_repos` metadata
containing the same resolved SHA evidence for execution runs.

### Tests
- Local bare remote: first unpinned resolve reads default-branch content, then after a new remote
  default-branch commit the second resolve reads the fresh content.
- Unpinned resolver records `resolved_sha` equal to cache `HEAD`.
- Pinned SHA/ref remains stable after the remote default branch advances.
- `path` source ref verification does not mutate the checkout `HEAD`.
- Execution run logs include fetched `source_repos[].resolved_sha` evidence.

### Docs/fixtures
- Update `README.md`, `docs/spec/WORKSPACE_SPEC.md`, `docs/ARCHITECTURE.md`,
  `docs/TESTING_STRATEGY.md` and `docs/STAKEHOLDER_DOC.md`.
- No JSON schemas, examples, golden fixtures or live matrices change.

### Acceptance
- [x] `go test ./internal/workspace` covers git_url freshness and pinned stability.
- [x] Targeted orchestrator/API test covers persisted run-log resolved SHA evidence.
- [x] `go test ./internal/...` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms only ACP-owned git_url caches are reset and no user checkout mutation
      path was introduced.
- [x] Commit `19F: refresh unpinned git_url sources`.

### Progress log
- 2026-07-13: Started `19F` after clean `19E` commit. Backlog scope confirmed: fresh unpinned
  `git_url` resolution, resolved SHA evidence, local bare-remote regressions and docs-only
  source freshness sync without `workspace.yaml` schema changes.
- 2026-07-13: Implemented `19F`. Unpinned `git_url` cache resolution now fetches the remote,
  resolves remote default `HEAD`, force-checkouts/resets/cleans the ACP-owned cache to the exact
  commit, and exposes `resolved_sha` in fetch-backed resolver output plus persisted run-log
  `source_repos` evidence. Pinned SHA/ref selection remains separate from remote default, and
  path-source ref verification remains non-mutating. Regression coverage uses only local bare
  remotes and verifies fresh default-branch content, pinned stability, path checkout safety and
  run-log SHA evidence. Verification passed: `go test ./internal/workspace ./internal/orchestrator`,
  `go test ./internal/...`, and full DoD with exact Node 22.21.1: `make contracts`, `make test`,
  `make lint`, `make build`.

### Plan ID
EP-20260713-epic-19-19g-minimum-collect-evidence-contract

### Context
`19G` follows committed `19F` and must land before `19H`. The active shard-pack schema requires
`documents`, `citations`, `documents[].citation_ids`, `citations[].claim_ids` and
`citations[].document_ids` fields, but schema arrays can still be empty and contract validation
does not reject authored documents without citation IDs. That lets sparse/no-evidence collect
packets reach checkpoint/apply paths instead of the existing collect repair/terminal path.

### Goals (must have)
- [x] Make non-empty `documents[]` and `citations[]` an explicit shard-pack schema and runtime
      contract requirement.
- [x] Make every authored `documents[].citation_ids` array non-empty and require its values to
      reference existing citation IDs.
- [x] Keep existing `citations[].claim_ids` and `citations[].document_ids` non-empty behavior,
      and mirror it at schema level with clear validation errors.
- [x] Ensure invalid sparse collect packs fail before checkpoint/apply and continue through the
      existing repair/terminal classification paths.
- [x] Synchronize schema docs, examples/fixtures and tests without changing `workspace.yaml`,
      provider list, live matrices or Epic 20 UI behavior.
- [x] Continue PR-1 with `19H` symmetric document/citation validation after `19G`
      review/commit boundary is stable.
- [ ] Continue PR-1 with `19I` historical run artifact snapshots after `19H`
      review/commit boundary is stable.

### Non-goals
- [ ] Do not implement `19H` reverse citation symmetry (`citation.document_ids` resolving to
      current documents and remap checks) in this slice.
- [ ] Do not change final-run-index or citation-index schemas.
- [ ] Do not weaken collect repair/recovery policy or synthesize provider-authored markdown.
- [ ] Do not require live providers or network access.

### Implementation
1) Update `schemas/shard-pack-manifest.schema.json` with `minItems: 1` for `documents`,
   `citations`, `documents[].citation_ids`, `citations[].claim_ids` and
   `citations[].document_ids`.
2) Tighten `internal/contracts` semantic validation so empty documents/citations and
   authored documents without citation IDs produce deterministic messages.
3) Adjust artifactquality/providercommon tests that previously treated empty collect packs as
   valid; add negative cases for empty arrays and missing document citation IDs plus a positive
   minimal manifest.
4) Verify runtime apply/monitor surfaces still use `ValidateCollectManifestInRoot` so invalid
   packs fail before checkpoint/apply.
5) Sync schema docs in `docs/APPENDIX_SCHEMAS.md`, `docs/spec/PIPELINE_SPEC.md`, examples,
   testing docs and ADR rationale.

### Interfaces
Public schema change: `shard-pack-manifest.schema.json` now rejects empty collect
documents/citations and empty citation binding arrays. Workspace/API schemas do not change.

### Tests
- Schema/contract test rejects `documents: []`.
- Schema/contract test rejects `citations: []`.
- Contract test rejects authored document with empty `citation_ids`.
- Contract test keeps rejecting missing/empty `citations[].claim_ids` and
  `citations[].document_ids`.
- Artifactquality/runtime validation test proves sparse collect pack fails before apply and a
  valid minimal pack passes.

### Docs/fixtures
- Update `schemas/shard-pack-manifest.schema.json`, `docs/APPENDIX_SCHEMAS.md`,
  `docs/spec/PIPELINE_SPEC.md`, `docs/TESTING_STRATEGY.md`, `examples/shard-pack-manifest.example.json`
  if needed, and affected test fixtures/golden snippets.

### Acceptance
- [x] `make contracts` passes with the stricter shard-pack schema.
- [x] `go test ./internal/contracts ./internal/artifactquality ./internal/runtime/providercommon`
      passes.
- [x] `go test ./internal/...` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms `19H` reverse-link/remap behavior was not implemented early.
- [x] Commit `19G: require collect evidence bindings`.

### Progress log
- 2026-07-13: Started `19G` after clean `19F` commit. Schema-guardian and test-fixtures rules
  apply because this slice changes shard-pack contract validation and its regression fixtures.
- 2026-07-13: Implemented `19G`. `shard-pack-manifest.schema.json` now requires non-empty
  `documents[]`, `citations[]`, `documents[].citation_ids`, `citations[].claim_ids` and
  `citations[].document_ids`; Go contract validation mirrors the minimum evidence requirement.
  Sparse collect packs fail strict validation before checkpoint/apply, while valid minimal
  evidence-backed packs still pass. Existing deterministic claim-binding recovery was updated to
  recognize the new schema `minItems` wording for empty `claim_ids`; no `19H` reverse
  `citation.document_ids` resolution/remap behavior was added. Verification passed:
  `make contracts`, `go test ./internal/contracts ./internal/artifactquality ./internal/runtime/providercommon`,
  `go test ./internal/...`, and full DoD with exact Node 22.21.1: `make contracts`,
  `make test`, `make lint`, `make build`.

### Plan ID
EP-20260713-epic-19-19h-symmetric-document-citation-validation

### Context
`19H` follows committed `19G`. The shard-pack schema and contract now require non-empty
documents, citations and binding arrays, and documents already reject unknown `citation_ids`.
However `citations[].document_ids` is still only shape-checked: it can point at an unknown
document, and document/citation links can be one-way. During citation-index aggregation,
document IDs are remapped to final canonical document IDs, but validation does not yet prove
that every remapped citation document ID resolves to the current final-run document set.

### Goals (must have)
- [x] Reject shard-pack citations whose `document_ids` do not resolve to documents in the same
      manifest.
- [x] Reject asymmetric shard-pack bindings in both directions: a document-citation link must
      be present in `documents[].citation_ids` and the matching `citations[].document_ids`.
- [x] Validate the staged final citation index against the final run index after document-ID
      remap so post-remap dangling document IDs and one-way links fail before promotion.
- [x] Keep valid duplicate source document ID remap behavior green: duplicate shard-local IDs may
      still map to unique final document IDs, and citation bindings must follow that remap.
- [x] Update pipeline/schema docs and ADR rationale for symmetric evidence bindings without
      changing provider list, `workspace.yaml`, live matrices or Epic 20 UI behavior.
- [x] Continue PR-1 with `19I` historical run artifact snapshots after `19H` review/commit
      boundary is stable.
- [x] Continue PR-1 with `19J` request-scoped UI detail state after `19I` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change JSON schema field names or add a new snapshot/citation schema version.
- [ ] Do not change collect retry policy except through existing invalid-manifest
      repair/terminal paths.
- [ ] Do not synthesize missing provider-authored shard-pack links in collect output; only
      orchestrator-generated `runtime-derived` final documents may be reconciled into the
      generated citation index so generated fallback links remain reciprocal.
- [ ] Do not implement Epic 20 snapshot UX or frontend request-state work.

### Implementation
1) Add contract-level symmetry validation in `internal/contracts/docflow.go`: build document and
   citation lookup maps, reject unknown `citation.document_ids`, reject
   `document.citation_ids` entries not reciprocated by the citation, and reject
   `citation.document_ids` entries not reciprocated by the document.
2) Keep schema JSON unchanged unless implementation proves a shape-level schema edit is needed;
   this slice strengthens semantic contract validation.
3) Extend staged artifact validation in `internal/orchestrator/docflow.go` so the generated
   `citation-index.json` and `final-run-index.json` are cross-checked after document-ID remap:
   every citation document ID must exist, every final document citation ID must exist, and the
   links must be reciprocal.
4) Add focused negative tests for unknown citation document ID, document-to-citation asymmetry,
   citation-to-document asymmetry and post-remap dangling final citation document ID.
5) Reconcile only orchestrator-generated `runtime-derived` final documents into
   `citation-index.json` when `buildFinalRunIndex` adds fallback citation IDs, so generated
   final docs do not create one-way links.
6) Keep the existing duplicate-document-ID remap test green and extend it, if needed, to prove
   remapped citation document IDs still match final canonical document IDs.

### Interfaces
No schema field changes. Public behavior changes by rejecting previously accepted semantically
invalid shard-pack manifests and staged citation indexes with dangling/asymmetric evidence links.

### Tests
- `ParseShardPackManifest` rejects `citations[].document_ids` that reference an unknown document.
- `ParseShardPackManifest` rejects a document listing a citation that does not list that document.
- `ParseShardPackManifest` rejects a citation listing a document that does not list that citation.
- `validateStagedArtifacts` rejects post-remap citation-index document IDs not present in the
  final run index.
- Duplicate source document ID remap remains green and produces reciprocal final document/citation
  links.

### Docs/fixtures
- Update `docs/spec/PIPELINE_SPEC.md`, `docs/APPENDIX_SCHEMAS.md`,
  `docs/TESTING_STRATEGY.md` and ADR rationale for symmetric collect evidence bindings.
- No example JSON change is expected because current examples already use reciprocal bindings.

### Acceptance
- [x] `go test ./internal/contracts ./internal/orchestrator` passes.
- [x] `go test ./internal/...` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19I` frontend snapshot behavior or Epic 20 UX work leaked into
      this slice.
- [x] Commit `19H: enforce symmetric citation bindings`.

### Progress log
- 2026-07-13: Started `19H` after clean `19G` commit. Spec-first, schema-guardian and docs-sync
  rules apply because this slice strengthens docflow contract semantics and the documented
  pipeline evidence contract.
- 2026-07-13: Implemented `19H`. Shard-pack validation now rejects unknown
  `citations[].document_ids` and one-way document/citation links; staged artifact validation
  cross-checks generated `final-run-index.json` and `citation-index.json` after document-ID
  remap. Runtime-derived final docs generated by the orchestrator are explicitly reconciled into
  the generated citation index so fallback citations are reciprocal without mutating
  provider-authored shard-pack links. Regression coverage includes unknown document IDs,
  document-to-citation asymmetry, citation-to-document asymmetry, post-remap dangling citation
  documents and valid duplicate-ID remap. Verification passed: `go test ./internal/contracts ./internal/orchestrator`,
  `go test ./internal/...`, and full DoD with exact Node 22.21.1: `make contracts`,
  `make test`, `make lint`, `make build`.

### Plan ID
EP-20260713-epic-19-19i-historical-run-artifact-snapshots

### Context
`19I` follows committed `19H`. The backend final-run index already carries
`canonical_path` and `staged_path`, but the frontend contract keeps only `canonical_path`.
`useRunArtifacts` converts selected-run final documents back into canonical paths and
`handleOpenArtifact` reads those canonical paths through `/api/artifacts`. Coverage summary and
open questions are also loaded from stable canonical paths. When a later run promotes new bytes
to the same canonical path, selecting an older run can show current workspace bytes under the
older run label.

### Goals (must have)
- [x] Preserve final-run-index `run_id`, `generated_at` and per-document `staged_path` in the
      TypeScript contract.
- [x] Keep artifact display/selection keyed by canonical label while reading selected-run bytes
      from run-scoped `staged_path`.
- [x] Reject mismatched final-run-index `run_id` and out-of-root `staged_path` values instead of
      falling back to canonical bytes.
- [x] Load coverage summary and open questions from selected-run staged documents when the
      final-run index contains those canonical documents.
- [x] Add deterministic UI regression with two runs sharing the same canonical path but different
      staged bytes; older selection must not read current canonical content.
- [x] Continue PR-1 with `19J` request-scoped UI detail state after `19I` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not implement Epic 20 source-mode UI copy (`Run snapshot` / `Current workspace`) in this
      slice.
- [ ] Do not add a new backend snapshot API or change final-run-index schema.
- [ ] Do not solve all stale async response races; `19J` owns request-generation and abort
      primitives.
- [ ] Do not change Publish commit scope or Git semantics.

### Implementation
1) Extend `ui/src/lib/appContracts.ts` so final-run-index payloads include top-level `run_id`,
   `generated_at` and required document `staged_path`.
2) Extend `Artifact` with optional internal read metadata (`read_path`, canonical path/source
   markers) while preserving `path` as the operator-facing canonical label used by existing UI
   selectors.
3) In `useRunArtifacts`, parse the selected run's final-run-index, validate `run_id` and
   `staged_path` root, derive display artifacts from `canonical_path`, and read previews from
   `read_path`.
4) Make final-run-index failure fail closed for snapshot mode: do not use canonical final
   document paths when the selected run index is corrupt, mismatched or cross-run.
5) Load coverage/open-question content through the same selected-run final-index mapping when
   available, with canonical fallback only for runs that have no final-run-index artifact.

### Interfaces
Frontend-only TypeScript contract extension. HTTP and JSON schemas do not change.

### Tests
- UI regression: run A and run B expose `reports/as-is/overview.md` with different staged bytes;
  selecting A reads A staged bytes, selecting B reads B staged bytes, and canonical current bytes
  are never displayed for either selected historical run.
- Coverage/open-question regression: selected-run staged coverage documents override canonical
  workspace files.
- Existing artifact filters and proposal/publish artifact lists continue to work with canonical
  display paths.

### Docs/fixtures
- Update `docs/TESTING_STRATEGY.md` and stakeholder/operator docs only for selected-run snapshot
  preview behavior. No schema/example changes are expected.

### Acceptance
- [x] `npm --prefix ui test -- --run` passes.
- [x] `npm --prefix ui run typecheck` passes.
- [x] `go test ./internal/docsync` passes after docs update.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19J` request-generation primitive or Epic 20 source-mode IA
      leaked into this slice.
- [x] Commit `19I: read historical artifacts from run snapshots`.

### Progress log
- 2026-07-13: Started `19I` after clean `19H` commit. Spec-first, docs-sync and
  UI-implementation-QA rules apply because this slice changes frontend artifact read behavior
  and operator-visible Review evidence correctness.
- 2026-07-13: Implemented `19I`. The frontend final-run-index contract now preserves
  `run_id`, `generated_at` and document `staged_path`; Review artifacts keep canonical display
  paths but read previews from selected-run staged paths. Final-run-index run mismatches and
  cross-run staged paths fail closed without canonical fallback, while runs without a final index
  keep the existing current-workspace read behavior. Coverage summary and open questions load
  from selected-run staged documents when indexed. UI regression covers two runs sharing
  `reports/as-is/overview.md` with different staged bytes and proves current canonical bytes are
  not displayed. Verification passed: UI typecheck, UI Vitest `94/94`, `go test ./internal/docsync`,
  and full DoD with exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260713-epic-19-19j-request-scoped-ui-detail-state

### Context
`19J` follows committed `19I`. Review artifacts now read selected-run staged bytes, but the
frontend still lets independent async detail requests write unkeyed state after selection changes:
run status, logs, artifact lists, artifact previews, coverage/open questions, run review summary
and Git diff can all be requested for run/path A and resolve after the operator has selected
run/path B. The UI must clear stale detail state immediately and only allow the latest matching
request generation to update the visible panel.

### Goals (must have)
- [x] Add a reusable frontend request generation / AbortController helper for abort-aware async
      detail loading.
- [x] Apply request-scoped state guards to run status, logs, artifacts, selected artifact preview,
      coverage/open questions, run review summary and Git diff.
- [x] Keep state writes keyed by the current request selection so a late response for run/path A
      cannot update panels for run/path B.
- [x] Clear visible detail surfaces on selection changes before replacement data lands.
- [x] Treat unmount/selection aborts as silent cancellation, not user-visible errors.
- [x] Continue PR-1 with `19K1` run mutation acknowledgement after `19J` review/commit boundary
      is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change backend APIs, JSON schemas, final-run-index semantics or provider/runtime
      behavior.
- [ ] Do not implement `19K1` mutation acknowledgement, `19K2` Q&A provisional ordering or
      `19L` editor safety.
- [ ] Do not add Epic 20 source-mode UX, Publish scope changes, IA changes or persisted approval
      workflow.
- [ ] Do not refactor unrelated stage layout, visual design, accessibility primitives or
      generated contract tooling.

### Implementation
1) Add a small shared hook in `ui/src/hooks` that starts a named request generation, aborts the
   previous generation for that surface, exposes `signal`, `requestKey`, `isCurrent()` and cleanup
   helpers, and suppresses ordinary `AbortError`.
2) Extend frontend API helpers where needed so existing fetches can receive an `AbortSignal`
   without changing backend endpoints.
3) Update `useRunActions` so selected-run status/detail loads are keyed by run ID and ignored if
   a newer selection supersedes them.
4) Update `useRunLogs` so reset/page/until-EOF log loads are keyed by run ID and cursor; late log
   pages cannot merge into a different selected run.
5) Update `useRunArtifacts` so artifact-list, coverage/open-question and preview requests have
   independent request generations; clearing artifacts aborts all three surfaces.
6) Update `useRunReview` and `useGitDiff` so late review-summary or diff responses cannot replace
   newer selected-run/path state.
7) Add targeted UI regressions using deferred fetch responses to prove A -> B switching ignores
   late A writes while existing `19I` snapshot behavior remains green.

### Interfaces
Frontend-only hook/API-helper types. No HTTP, backend, schema or public artifact contract change.

### Tests
- Deferred selected-run A status/log/artifact responses resolving after selected-run B do not
  update B panels.
- Switching selected run clears stale artifact/log/review detail state before B data lands.
- Late artifact preview for an older selected artifact cannot overwrite a newer selected artifact.
- Late Git diff for an older run/path cannot overwrite a newer run/path diff.
- Aborted requests during unmount or selection change do not surface user-visible errors.
- Existing `19I` historical snapshot tests stay green.

### Docs/fixtures
No operator documentation, schema, example or fixture changes are expected. `docs/PLANS.md` is
the only documentation update unless implementation changes visible loading/error copy.

### Acceptance
- [x] Targeted UI Vitest suite passes.
- [x] `npm --prefix ui run typecheck` passes.
- [x] Existing `19I` snapshot regression stays green.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19K1`, `19K2`, `19L` or Epic 20 work leaked into this slice.
- [x] Commit `19J: isolate UI detail requests`.

### Progress log
- 2026-07-13: Started `19J` after clean `19I` commit. Spec-first and UI-implementation-QA
  rules apply because this slice introduces a shared frontend async-state primitive and UI
  regressions for stale request ordering.
- 2026-07-13: Implemented `19J`. Added `useRequestGate` for request-keyed AbortController
  ownership and applied it to selected-run status, logs, artifacts, preview, coverage/open
  questions, review summary and Git diff. Selection changes now clear stale detail surfaces and
  late A responses are ignored after B becomes current. Added deferred-response regressions for
  artifact preview and Git diff ordering while keeping the `19I` historical snapshot regression
  green. Verification passed: UI typecheck, UI Vitest `96/96`, and full DoD with exact Node
  22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19k1-run-mutation-acknowledgement

### Context
`19K1` follows committed `19J`. The UI already receives accepted `POST /api/pipeline/init|refresh`
and `POST /api/pipeline/runs/<run_id>/cancel` responses, but follow-up detail/list/log requests
still execute inside the same mutation try/catch. If a post-acknowledgement GET fails, the UI can
present the accepted mutation as a failed start/cancel instead of a recoverable reconciliation
state. This creates duplicate-action risk and hides the server-accepted `run_id`.

### Goals (must have)
- [x] Preserve an accepted start response as provisional selected run state immediately.
- [x] Treat the first failed follow-up status/list/log request after accepted start as
      reconciliation status, not failed mutation.
- [x] Preserve accepted cancel acknowledgement after `202` even if follow-up list/status/log
      reconciliation fails.
- [x] Avoid duplicate start/cancel requests after accepted mutation acknowledgement.
- [x] Reuse the `19J` request-gated detail loading behavior without introducing another async
      state primitive.
- [x] Continue PR-1 with `19K2` Q&A provisional run ordering after `19K1` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change backend APIs, schemas or run start/cancel response formats.
- [ ] Do not implement Q&A provisional ordering; that remains `19K2`.
- [ ] Do not implement manifest/editor dirty-draft safety; that remains `19L`.
- [ ] Do not change queue semantics, active-run blocking or Epic 20 explicit queue workflow.

### Implementation
1) In `useRunActions`, split accepted mutation acknowledgement from follow-up reconciliation.
   `handleRunPipeline` should upsert the provisional run, select it and show accepted copy before
   any status/log/list reconciliation is attempted.
2) Add a small internal reconciliation helper for post-start detail/list/log loading. It may set
   `runActionStatus` to a recovery message when detail loading fails, but it must not set the
   global mutation error or return `false` after the start POST succeeded.
3) Apply the same boundary to cancel `202`: the accepted cancel message remains visible if
   follow-up list/status/log loading fails.
4) Keep existing 404 and 409 cancel handling unchanged except for making post-acknowledgement
   detail failures recoverable.
5) Add UI regressions where POST succeeds and the first follow-up GET fails, proving the accepted
   run/cancel state remains selected and no duplicate mutation is sent.

### Interfaces
Frontend-only behavior change in run explorer state. HTTP, backend, schemas and public artifact
contracts remain unchanged.

### Tests
- Start POST succeeds and first `GET /api/pipeline/runs/<run_id>` fails; UI still shows the
  accepted provisional `run_id`, reports reconciliation status and sends only one start request.
- Later polling/list recovery for the same `run_id` replaces provisional state with server status.
- Cancel `202` followed by failed list/status/log reconciliation keeps `Cancel requested for
  <run_id>` visible and sends only one cancel request.
- Existing missing-run cancel `404`, already-terminal `409`, `19J` stale-response and `19I`
  historical snapshot tests remain green.

### Docs/fixtures
No operator documentation, schema, example or fixture changes are expected. `docs/PLANS.md` is
the only documentation update unless implementation changes user-facing recovery copy.

### Acceptance
- [x] Targeted `ui/src/App.test.tsx` mutation acknowledgement tests pass.
- [x] `npm --prefix ui run typecheck` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19K2`, `19L`, queue semantics or Epic 20 work leaked into this
      slice.
- [x] Commit `19K1: acknowledge accepted run mutations`.

### Progress log
- 2026-07-14: Started `19K1` after clean `19J` commit. Spec-first and UI-implementation-QA
  rules apply because this slice changes frontend mutation acknowledgement and recovery states.
- 2026-07-14: Implemented `19K1`. Accepted pipeline starts now upsert and select a provisional
  run before follow-up status/log/list reconciliation, and accepted cancels preserve their
  acknowledgement when post-acknowledgement reconciliation fails. Regression coverage proves
  start/cancel POSTs are not duplicated and later polling reconciles the same accepted run ID.
  During full DoD, docs-sync exposed completed `19H`-`19J` active plans without open follow-up
  goals, so those plans now keep an explicit final-reconciliation archive goal. Verification
  passed: UI typecheck, targeted App mutation acknowledgement tests, UI Vitest `98/98`,
  `go test ./internal/docsync`, and full DoD with exact Node 22.21.1: `make contracts`,
  `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19k2-qa-provisional-run-ordering

### Context
`19K2` follows committed `19K1`. The Ask panel still treats accepted Q&A start and follow-up
detail/history loading as one mutation: `startQARun` clears the selected answer, waits for
`GET /api/qa/runs/<run_id>`, and reports any post-acknowledgement failure as a failed Q&A
request. Initial history load, manual history refresh and selected-run detail loads also use
local cancel booleans instead of request-key ownership. A late history/detail response can
therefore replace a newer selected Q&A run, and an accepted QA run can disappear until polling
recovers.

### Goals (must have)
- [x] Create and select a provisional Q&A run immediately after accepted
      `POST /api/qa/runs`.
- [x] Treat the first failed detail/history GET after accepted Q&A start as recoverable
      reconciliation status, not a failed submit.
- [x] Keep history, selected detail and polling writes keyed so the last selected Q&A run wins.
- [x] Disable double submit while an accepted Q&A run is reconciling or active.
- [x] Reuse the `19J` request gate primitive instead of adding another async-state model.
- [x] Continue PR-1 with `19L` editor draft safety after `19K2` review/commit boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change pipeline start/cancel acknowledgement; that was `19K1`.
- [ ] Do not change manifest/editor draft behavior; that remains `19L`.
- [ ] Do not change backend Q&A endpoints, schemas or response formats.
- [ ] Do not implement Epic 20 Ask IA/source-mode changes.

### Implementation
1) Extend `qaApi` helpers so history/detail requests can receive an `AbortSignal` without
   changing backend URLs.
2) In `AskStagePanel`, add request gates for Q&A history and selected-detail ownership.
   Initial history load, manual refresh, selected history row loads and accepted-start detail
   reconciliation must only write visible state when their token is current.
3) Build a provisional `QARunResponse` from the accepted start response plus the submitted
   question, upsert it into history, select it and display an accepted/reconciling status before
   detail GET.
4) If post-acknowledgement detail/history reconciliation fails, keep the provisional run selected
   and show a recoverable reconciliation message. The accepted submit returns without allowing a
   second submit until the detail/polling loop owns the same run.
5) Keep polling scoped to the currently selected active QA run, and ignore/abort stale polling
   results after a newer selection or submission.

### Interfaces
Frontend-only helper signatures and Ask state flow. HTTP, backend, schemas and public artifact
contracts remain unchanged.

### Tests
- Accepted Q&A start remains selected after the first detail GET fails; later polling reconciles
  the same run ID.
- Double submit is disabled while the accepted Q&A run is reconciling or active.
- A delayed old history response cannot replace a newer selected/submitted Q&A run.
- A delayed old selected-run detail response cannot overwrite a newer selected Q&A run.
- Existing Q&A answer, failure recovery, retry and nullable-evidence tests remain green.

### Docs/fixtures
No schema, example or fixture changes are expected. `docs/PLANS.md` is the only documentation
update unless implementation changes operator-visible recovery copy.

### Acceptance
- [x] Targeted `ui/src/App.test.tsx` Q&A ordering tests pass.
- [x] `npm --prefix ui run typecheck` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19L`, Git publication, queue semantics or Epic 20 work leaked
      into this slice.
- [x] Commit `19K2: preserve accepted QA run selection`.

### Progress log
- 2026-07-14: Started `19K2` after clean `19K1` commit. Spec-first and UI-implementation-QA
  rules apply because this slice changes frontend Q&A async mutation ordering and recovery
  states.
- 2026-07-14: Implemented `19K2`. Ask now creates a provisional selected QA run immediately
  after accepted `POST /api/qa/runs`, keeps accepted runs selected through first detail
  reconciliation failures, and keys history, selected-detail and polling writes so late older
  responses cannot replace the latest selection. QA history/detail helpers now accept
  `AbortSignal`, and regression coverage proves accepted-run recovery, delayed history ordering,
  delayed detail ordering and disabled double submit while accepted/active. Verification passed:
  UI typecheck, targeted App Q&A ordering tests, UI Vitest `101/101`, and full DoD with exact
  Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19l-editor-revision-safety

### Context
`19L` follows committed `19K2`. Source manifest saves and Charter baseline artifact editing still
trust late async completions too broadly. Manifest save handlers always mark setup dirty state
clean after `save -> validate`, even if the operator edited the form while the save was in
flight. Baseline artifact loading uses one selected content slot: a late load for an older path
can overwrite typed content, switching paths loses dirty drafts, and a save completion can report
the draft clean even when the operator changed the text after save started.

### Goals (must have)
- [x] Add form revision/snapshot checks to raw manifest and guided Source save paths.
- [x] Keep Source dirty state dirty when text/form changes after a save starts.
- [x] Add per-path single-owner draft state for Charter/Baseline editable artifacts.
- [x] Ignore late artifact loads for old paths or dirty newer drafts.
- [x] Keep edits made during a deferred artifact save dirty and visible after save resolves.
- [x] Continue PR-1 with `19M` reproducible embedded UI build after `19L` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change Git publication, Q&A, pipeline run or queue semantics.
- [ ] Do not change backend APIs, schemas or artifact write response formats.
- [ ] Do not redesign Charter editor UX beyond correctness/status copy needed for stale saves.
- [ ] Do not start Epic 20.

### Implementation
1) In `useManifestEditor`, track a monotonic form revision and current manifest content ref.
   Every raw text edit, guided repo edit/import path edit and guided apply increments the
   revision and marks setup dirty.
2) Manifest save handlers capture `{revision, manifest}` before the write. After
   `saveWorkspaceManifest` and validation return, only clear dirty state and accept validation
   if the current revision/content still match the saved snapshot.
3) In `useBaselineEditor`, replace single content ownership with a per-path draft map carrying
   content, loaded content, dirty flag and revision.
4) Baseline artifact loads capture path/load sequence and only write state if that path is still
   selected and the draft was not edited after the load started.
5) Baseline artifact saves capture path/content/revision and only mark the path clean when the
   draft is unchanged at save completion; otherwise preserve newer text and show status that
   unsaved edits remain.

### Interfaces
Frontend-only hook behavior. No HTTP, backend, schema or public artifact contract change.

### Tests
- Raw manifest edit during deferred save keeps the newer text and does not treat the newer draft
  as clean.
- Guided Source edit during deferred save keeps dirty state when form revision changes after save
  starts.
- Late baseline artifact load for path A cannot overwrite typed content for selected path B.
- Switching baseline paths preserves each path's dirty draft content.
- Editing a baseline artifact during deferred save preserves the newer dirty text after save
  resolves.

### Docs/fixtures
No schema, examples or fixtures are expected. `docs/PLANS.md` is the only documentation update
unless implementation changes operator-visible editor status copy.

### Acceptance
- [x] Targeted `ui/src/App.test.tsx` manifest/editor stale-write tests pass.
- [x] `npm --prefix ui run typecheck` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no Git publication, run/Q&A state, schema or Epic 20 work leaked into
      this slice.
- [x] Commit `19L: protect editor drafts from stale writes`.

### Progress log
- 2026-07-14: Started `19L` after clean `19K2` commit. Spec-first and UI-implementation-QA
  rules apply because this slice changes frontend editor async-state correctness and recovery
  status copy.
- 2026-07-14: Implemented `19L`. Source manifest saves now capture revision/content snapshots
  and only mark the form clean when the same draft is still current; stale completions surface
  `newer unsaved edits remain`. Charter/Baseline editing now owns per-path drafts with
  load/save revision guards, so late loads for old paths and save completions for superseded
  text cannot overwrite visible operator edits. Regression coverage includes raw manifest
  edit-during-save, late baseline load, per-path dirty draft switching and edit-during-save for
  baseline artifacts. Verification passed: UI typecheck, targeted App stale-write tests, UI
  Vitest `105/105`, and full DoD with exact Node 22.21.1: `make contracts`, `make test`,
  `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19m-deterministic-embedded-ui-bundle

### Context
`19M` follows committed `19L`. `make build` currently removes `ui/dist`, runs Vite and copies the
result into `internal/api/ui_dist`, but CI does not prove that the committed embedded bundle is
fresh. There is also no exact-commit double-build verifier that compares independent clean
builds by sorted path/digest. Vite output naming is implicit, so future bundler upgrades could
change chunk names/order without a dedicated guard.

### Goals (must have)
- [x] Make Vite output naming explicit for entry chunks, dynamic chunks and assets.
- [x] Add an exact-commit UI build determinism verifier that builds the same ref in two
      independent temp roots and compares sorted paths/digests.
- [x] Add an embedded UI freshness verifier that rebuilds/copies `internal/api/ui_dist` and fails
      if Git detects a diff.
- [x] Wire the new verifiers into provider-free UI CI.
- [x] Update testing/build docs for deterministic UI and stale embedded bundle checks.
- [x] Continue PR-1 with `19O` locked contract validator tooling after `19M` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change app UI behavior or design.
- [ ] Do not change backend APIs, schemas or live-provider requirements.
- [ ] Do not solve contract validator lockfiles; that remains `19O`.
- [ ] Do not make live provider checks required in PR CI.

### Implementation
1) Update `ui/vite.config.ts` with explicit deterministic `assetsDir`, `entryFileNames`,
   `chunkFileNames` and `assetFileNames`.
2) Add `scripts/verify-ui-deterministic-build.sh`: archive the requested Git ref into two temp
   roots, or copy the current `WORKTREE` for pre-commit self-review, reuse installed
   `ui/node_modules` when available, run the pinned `scripts/run-npm.sh run build --prefix ui`
   in both roots, and compare sorted SHA-256 path manifests.
3) Add `scripts/check-ui-dist-fresh.sh`: rebuild UI, copy `ui/dist` into `internal/api/ui_dist`
   using the same embed path as `make build`, and fail with a diff/stat when committed embedded
   assets are stale.
4) Add Make targets for both checks without changing the local `make build` behavior that updates
   embedded assets for the current slice.
5) Update the UI workflow to run UI tests, standalone Vite build, deterministic double-build and
   embedded bundle freshness checks after `npm ci`.
6) Update `docs/TESTING_STRATEGY.md` and build guidance to explain local usage and CI behavior.

### Interfaces
Tooling-only changes: Make targets, CI workflow and scripts. No runtime API/schema changes.

### Tests
- `scripts/verify-ui-deterministic-build.sh HEAD` produces identical sorted path/digest manifests
  for two clean temp roots.
- `scripts/check-ui-dist-fresh.sh` exits zero when `internal/api/ui_dist` matches the source UI
  build.
- A deliberately stale embedded asset would make `scripts/check-ui-dist-fresh.sh` exit non-zero
  in CI.
- Existing UI tests/typecheck/build remain green.

### Docs/fixtures
- Update `docs/TESTING_STRATEGY.md`.
- Update `CONTRIBUTING.md` and architecture build guidance for the new UI verification commands.

### Acceptance
- [x] `scripts/verify-ui-deterministic-build.sh WORKTREE` passes before commit; CI runs the same
      verifier against `HEAD` through `make verify-ui-determinism`.
- [x] `scripts/check-ui-dist-fresh.sh` passes on the current bundle state.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no UI behavior, schema, contract-toolchain or live-provider scope
      leaked into this slice.
- [x] Commit `19M: make embedded UI builds reproducible`.

### Progress log
- 2026-07-14: Started `19M` after clean `19L` commit. Spec-first and docs-sync rules apply
  because this slice changes deterministic build tooling and required CI behavior.
- 2026-07-14: Implemented explicit Vite output names, UI determinism/freshness scripts, Make
  targets and provider-free UI workflow checks. Pre-commit targeted verification passed:
  `scripts/verify-ui-deterministic-build.sh WORKTREE`, `make build`, and
  `scripts/check-ui-dist-fresh.sh`.
- 2026-07-14: Full DoD passed with exact Node 22.21.1: `make contracts`, `make test`,
  `make lint`, `make build`. A docs-sync active-plan invariant was fixed by adding the same
  final-reconciliation archive goal to completed `19L` that earlier completed slices use.

### Plan ID
EP-20260714-epic-19-19o-locked-contract-validator-toolchain

### Context
`19O` follows committed `19M` and closes the remaining release-reproducibility tooling gap before
`19N`. The current `make contracts` target executes:
`npm exec --yes --package=ajv-cli --package=ajv-formats --package=js-yaml ...`, which lets npm
resolve mutable registry versions at validation time. That makes local/CI contract validation
depend on whatever `latest` resolves to, rather than a reviewed lockfile diff.

### Goals (must have)
- [x] Add a versioned, lockfile-backed contract validator tooling package for `ajv-cli`,
      `ajv-formats` and `js-yaml`.
- [x] Make `make contracts` install/use that locked toolchain instead of mutable
      `npm exec --package=...` resolution.
- [x] Keep contract validation offline-capable after the locked package has been installed.
- [x] Keep current schema/example/fixture validation behavior unchanged.
- [x] Update developer/testing docs so toolchain version changes require explicit package/lockfile
      review.
- [x] Continue PR-1 with `19N` composite release verdict gate after `19O` review/commit boundary
      is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change schemas, examples, fixtures or semantic contract rules.
- [ ] Do not change UI package dependencies or the embedded UI build.
- [ ] Do not introduce live provider/network dependencies into required CI.
- [ ] Do not implement release verifier behavior; that remains `19N`.

### Implementation
1) Add `tools/contracts/package.json` with exact dependency ranges for the validator CLI
   toolchain and generate a committed `package-lock.json`.
2) Update `Makefile` with a `CONTRACT_TOOLS_DIR` and make `contracts` run
   `scripts/run-npm.sh ci --prefix tools/contracts` before validation, so installed versions come
   from the lockfile.
3) Update `scripts/validate-contracts.sh` to execute `tools/contracts/node_modules/.bin/ajv` and
   `tools/contracts/node_modules/.bin/js-yaml` directly, so global tools are never used as a
   fallback, and fail with a clear bootstrap message if the locked install is missing.
4) Leave schema/example/fixture validation loops intact so positive/negative behavior does not
   move in this slice.
5) Update `CONTRIBUTING.md` and `docs/TESTING_STRATEGY.md` to document the locked contract
   toolchain and explicit lockfile review requirement.

### Interfaces
Tooling-only changes: a new contract-tool npm package/lockfile, `make contracts` behavior and
validation script bootstrap. No product HTTP/API/schema/artifact contract change.

### Tests
- `make contracts` installs the locked toolchain and validates the existing positive/negative
  fixtures.
- Running `scripts/validate-contracts.sh` after `tools/contracts` install works without
  `npm exec --package` or mutable latest resolution.
- Removing the installed tool binaries makes the script fail with a bootstrap hint rather than
  falling back to global tools.
- Existing full DoD remains green.

### Docs/fixtures
- Update `CONTRIBUTING.md`.
- Update `docs/TESTING_STRATEGY.md`.
- No schema/example/fixture updates expected because validator semantics are unchanged.

### Acceptance
- [x] `ACP_NODE_TOOL_CANDIDATES="$HOME/.cache/provenarch-toolchains/node-v22.21.1-darwin-arm64/bin" make contracts` passes with the locked toolchain.
- [x] `PATH="$HOME/.cache/provenarch-toolchains/node-v22.21.1-darwin-arm64/bin:/usr/bin:/bin" scripts/validate-contracts.sh` passes after locked install by using repo-local tool binaries
      plus only the pinned Node runtime.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no schema, fixture, UI build or release-verifier behavior leaked into
      this slice.
- [x] Commit `19O: lock contract validation tooling`.

### Progress log
- 2026-07-14: Started `19O` after clean `19M` commit. Spec-first and docs-sync rules apply
  because this slice changes required contract-validation tooling and developer docs.
- 2026-07-14: Implemented `tools/contracts` exact dependency package/lockfile, including a
  patched `fast-json-patch` override to keep the new lockfile free of high audit advisories.
  `make contracts` now performs lockfile-backed `npm ci` and `scripts/validate-contracts.sh`
  executes repo-local `ajv`/`js-yaml` binaries directly instead of falling back to globals.
  Verification passed: clean locked install, direct pinned-Node/offline script run, missing-tools
  negative bootstrap, `npm audit --audit-level=high`, and full DoD with exact Node 22.21.1:
  `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19n-composite-release-verdict-gate

### Context
`19N` follows committed `19M` and `19O`. `scripts/verify-release-verdict.py` already verifies a
canonical `reports/release_verdict_<matrix-id>.json` plus companion
`swe_ux_assessment_<matrix-id>.md` and `swe_artifact_quality_assessment_<matrix-id>.md`, including
missing files, `FAIL`, matrix-id mismatch and unaccepted manual assessments. The GitHub release
workflow still grants GoReleaser/write permissions in the publishing job without a preceding
read-only composite-evidence verifier job.

### Goals (must have)
- [x] Add a read-only release evidence verification job that runs
      `scripts/verify-release-verdict.py` before any GoReleaser/write-permission job can start.
- [x] Make release publication depend on that verifier job with `needs`.
- [x] Keep GoReleaser and provenance write permissions scoped only to the publishing job.
- [x] Keep release verification offline: no live provider or matrix harness execution in the
      workflow.
- [x] Add workflow regression tests for missing verifier, missing dependency, and write
      permissions appearing before verification.
- [x] Update release runbook/docs with the workflow inputs and composite evidence gate.
- [x] Continue PR-1 with `19P` Step 1 card enrichment after `19N` review/commit boundary is
      stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change canonical release matrices, providers, sweeps or live matrix harness.
- [ ] Do not generate live evidence in CI.
- [ ] Do not weaken `verify-release-verdict.py` payload/manual-assessment requirements.
- [ ] Do not implement semantic enrichment; that remains `19P`.

### Implementation
1) Split `.github/workflows/release.yml` into a read-only `verify-release-evidence` job and the
   existing write-enabled `release` job.
2) The verifier job checks out the exact tag commit, resolves a verdict path from either
   `ACP_RELEASE_VERDICT_PATH` or `ACP_RELEASE_MATRIX_ID`, and runs
   `python3 scripts/verify-release-verdict.py "$verdict_path"`.
3) The release job keeps `contents/id-token/attestations: write`, keeps GoReleaser/attestation
   steps unchanged, and adds `needs: verify-release-evidence`.
4) Update release workflow tests to assert the verifier job has read-only permissions, the release
   job depends on it, and no GoReleaser/write-permission job can run without the verifier.
5) Update `docs/RELEASE_LIVE_E2E_RUNBOOK.md` and `docs/TESTING_STRATEGY.md` for the composite
   gate and required evidence variables.

### Interfaces
CI/release workflow surface only: `ACP_RELEASE_VERDICT_PATH` or `ACP_RELEASE_MATRIX_ID` must point
the release workflow at already-created evidence files in the checked-out tag. No product API,
schema or live harness interface changes.

### Tests
- Existing `scripts/tests/verify_release_verdict_test.py` continues covering missing, `FAIL`,
  matrix-mismatched and unaccepted SWE evidence.
- `scripts/tests/release_distribution_test.py` covers release workflow job ordering and
  permissions.
- Full DoD remains green.

### Docs/fixtures
- Update `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.
- Update `docs/TESTING_STRATEGY.md`.
- No fixtures/schema changes expected.

### Acceptance
- [x] Targeted release verifier/distribution tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no live matrix/provider, schema, UI or semantic enrichment scope leaked
      into this slice.
- [x] Commit `19N: require composite release verdict`.

### Progress log
- 2026-07-14: Started `19N` after clean `19O` commit. Spec-first and docs-sync rules apply
  because this slice changes release workflow policy and release docs.
- 2026-07-14: Implemented read-only `verify-release-evidence` workflow job, wired the
  write-enabled `release` job through `needs`, kept GoReleaser/provenance write permissions only
  on the publishing job, and documented `ACP_RELEASE_MATRIX_ID` / `ACP_RELEASE_VERDICT_PATH`.
  Verification passed: targeted release verifier/distribution tests and full DoD with exact Node
  22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19p-step1-card-enrichment

### Context
`19P` follows committed `19N` and starts the P2 quality-hardening band. The semantic card
enrichment implementation already exists in `internal/orchestrator/semantic_cards.go`, but Step 1
currently finishes after loading canonical team cards and never runs the enrichment pass. That
creates a mismatch with `docs/spec/PIPELINE_SPEC.md`: existing human-owned domain/team cards should
receive one deterministic `## Derived (ACP Step1)` section after Step 1 semantic apply, while ACP
must not auto-create or rename canonical cards.

### Goals (must have)
- [x] Restore exactly one Step 1 enrichment call after all domain semantic apply work completes.
- [x] Enrich only existing canonical `charter/cards/domains/*` and `charter/cards/teams/*`.
- [x] Keep `## Derived (ACP Step1)` idempotent across repeated init/refresh runs.
- [x] Preserve the question path for model owner teams without a matching canonical team card.
- [x] Cover evidence refs, related entities/services, findings/questions and coverage summaries in
      deterministic tests.
- [x] Continue PR-1 with `19Q` generic refresh semantic guard after `19P` review/commit boundary is
      stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change public schemas, runtime provider contracts, final-run-index/citation contracts
      or workspace manifest shape.
- [ ] Do not auto-create, delete, rename or otherwise take ownership of human-authored canonical
      cards.
- [ ] Do not implement `19Q` semantic guard filtering or any Epic 20 UI behavior.
- [ ] Do not change promotion, release, UI generated bundle or live provider matrix behavior.

### Implementation
1) In `runStepCollectByDomain`, keep the existing domain/team-card discovery and missing-card
   questions, then call `enrichCanonicalCards(domainIDs, teamCards)` once before Step 1 returns.
2) Preserve the existing renderer semantics: merge/replace only the managed
   `## Derived (ACP Step1)` block and leave human-authored card content before that heading intact.
3) Add orchestrator unit tests that seed canonical cards plus semantic model entities, findings,
   questions and coverage, run enrichment twice, and assert the managed section is present exactly
   once with evidence refs.
4) Add a regression for an owner team in the semantic model that lacks a canonical team card:
   enrichment must not create a new card and must add the existing high-priority question.
5) Run targeted orchestrator tests, then full DoD.

### Interfaces
Internal orchestrator behavior only. No HTTP API, TypeScript, JSON schema or workspace manifest
interfaces change.

### Tests
- Existing domain/team cards receive one `## Derived (ACP Step1)` section with related model IDs,
  findings/questions, coverage gaps and `repo:path` evidence refs.
- Re-running enrichment replaces the managed section instead of appending duplicates.
- A service owned by an unknown/missing canonical team card records a deterministic question and
  does not create `charter/cards/teams/<missing>.md`.
- Step 1 returns enrichment errors if card reads/writes fail instead of silently skipping the
  documented behavior.

### Docs/fixtures
- No schema/spec changes expected because this restores the existing `PIPELINE_SPEC` behavior.
- No scenario golden refresh is expected unless existing deterministic scenario tests cover this
  path and fail after the restored call.

### Acceptance
- [x] Targeted orchestrator tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19Q`, UI, contract, release or live-provider scope leaked into this
      slice.
- [x] Commit `19P: restore idempotent card enrichment`.

### Progress log
- 2026-07-14: Started `19P` after clean `19N` commit. Spec-first and test-fixtures rules apply
  because this slice restores documented model/card behavior and must add deterministic
  regression coverage.
- 2026-07-14: Restored the Step 1 enrichment call, added idempotency/no-autocreate/full-pipeline
  regression tests, and completed full DoD with exact Node 22.21.1:
  `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19q-generic-refresh-semantic-guard

### Context
`19Q` follows committed `19P`. The current semantic guard helpers are narrow and effectively
unreachable: refresh collect apply stores provider semantic entities directly, while the helper set
contains a domain-specific off-topic term list and a power-domain whitelist. The backlog decision is
to preserve the documented refresh guard, but make it generic: runtime/provider metadata and
off-scope semantic candidates are filtered or marked by deterministic diagnostics without a hidden
domain whitelist.

### Goals (must have)
- [x] Activate refresh-only semantic guard logic in `refresh.step1.collect` before model apply and
      before shard packs feed staged final indexes.
- [x] Filter runtime/provider/process metadata candidates generically from the semantic model.
- [x] Filter explicit off-scope candidates whose repo evidence does not match the assigned refresh
      repo scopes.
- [x] Emit deterministic diagnostic findings/warnings for every filtered candidate class.
- [x] Preserve legitimate same-repo/same-domain entities and leave init collect behavior unchanged.
- [x] Document the generic policy in architecture/ADR without schema or provider-list changes.
- [x] Continue PR-1 with `19R1` ARIA tabs controller after `19Q` review/commit boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not add or change shard-pack, final-index, citation-index or workspace schemas.
- [ ] Do not introduce a domain/business-term blacklist, whitelist or special-case corpus.
- [ ] Do not change provider prompts, live matrix inputs, release gates or required CI live checks.
- [ ] Do not implement accessibility slices `19R1..19R3`.

### Implementation
1) Replace the unused narrow helper cluster in `semantic_utils.go` with a generic
   `guardRefreshCollectSemantic(stepID, task, semantic)` function.
2) The guard is a no-op unless `stepID == refresh.step1.collect`.
3) Runtime metadata detection uses generic runtime/process markers from entity type/id/name/tags
   and runtime artifact evidence paths; it does not inspect product-domain words.
4) Off-scope detection uses task repo scopes versus semantic provenance evidence repos. A candidate
   with explicit evidence outside the assigned scopes is filtered; a same-scope candidate survives.
5) Edges that reference filtered entities are dropped; findings/questions tied only to filtered
   IDs are dropped; deterministic diagnostic findings are appended to the guarded semantic snapshot.
6) Apply the guarded semantic snapshot before `e.shardPacks` append and before `model.Store` apply,
   so promoted model/final indexes use the same filtered data.

### Interfaces
Internal orchestrator behavior and docs only. No public API, schema, workspace manifest, TypeScript
contract or provider list changes.

### Tests
- Refresh collect filters runtime/provider metadata entities and emits deterministic diagnostics.
- Refresh collect filters explicit off-scope repo-evidence candidates and drops edges tied to them.
- Legitimate same-scope refresh entities survive.
- Init collect with the same semantic payload is unchanged.
- Apply path uses the guarded semantic snapshot before model apply and shard-pack aggregation.

### Docs/fixtures
- Update `docs/ARCHITECTURE.md` refresh semantic guard wording.
- Add ADR rationale for generic evidence-scope policy and no hidden domain whitelist.
- No schema/example/golden fixture changes expected.

### Acceptance
- [x] Targeted orchestrator semantic guard tests pass.
- [x] `go test ./internal/docsync` passes after docs changes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no schema, live matrix, provider-list, UI accessibility or Epic 20 scope
      leaked into this slice.
- [x] Commit `19Q: generalize refresh semantic guard`.

### Progress log
- 2026-07-14: Started `19Q` after clean `19P` commit. Spec-first, test-fixtures and docs-sync
  rules apply because this slice changes documented semantic behavior and regression coverage.
- 2026-07-14: Replaced the narrow/unreached refresh helper cluster with an active generic
  evidence-scope guard, added pure/apply-path tests, documented architecture/ADR rationale, and
  completed full DoD with exact Node 22.21.1: `make contracts`, `make test`, `make lint`,
  `make build`.

### Plan ID
EP-20260714-epic-19-19r1-accessible-tabs-controller

### Context
`19R1` follows committed `19Q` and starts the accessibility primitives band. The UI already has a
shared `TabNav` component used by Analysis, Review, Proposal and Publish surfaces, but it only sets
`role="tab"` / `aria-selected`. It does not implement roving tabindex, Arrow/Home/End keyboard
navigation, or stable `tab` -> `tabpanel` relationships. Epic 20 will reuse this primitive, so the
fix should be centralized rather than duplicated per stage.

### Goals (must have)
- [x] Add reusable roving-tabindex keyboard behavior to `TabNav`.
- [x] Support ArrowLeft/ArrowRight/ArrowUp/ArrowDown plus Home/End.
- [x] Expose stable tab and tabpanel IDs through a small shared helper.
- [x] Wire current `TabNav` consumers to matching active tabpanel relationships without changing
      user-visible IA or tab labels.
- [x] Cover keyboard navigation, single tabbable tab, and aria-controls/labelledby links in Vitest.
- [x] Continue PR-1 with `19R2` keyboard path combobox after `19R1` review/commit boundary is
      stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not redesign Review/Publish IA; that remains Epic 20.
- [ ] Do not implement path combobox keyboard behavior; that remains `19R2`.
- [ ] Do not add async alert/live-region behavior; that remains `19R3`.
- [ ] Do not introduce a new UI library or change backend/API contracts.

### Implementation
1) Extend `ui/src/components/TabNav.tsx` with an `idBase` prop, exported tab/panel ID helpers,
   roving `tabIndex`, and keyboard focus/selection handling.
2) Add `tabPanelProps(idBase, value)` helper so consumers use one source of truth for
   `role="tabpanel"`, `id` and `aria-labelledby`.
3) Update existing `TabNav` call sites in `StagePanels.tsx` with stable `idBase` values and wrap
   active tab content/filter result regions with matching tabpanel props.
4) Add focused component tests for keyboard navigation and ARIA relationships, and keep existing
   App tests/selectors stable.

### Interfaces
Frontend-only component API change. No backend, schema, workspace, artifact or public HTTP
interfaces change.

### Tests
- Only the selected tab has `tabIndex=0`; inactive tabs have `tabIndex=-1`.
- Arrow/Home/End keys move focus and selected value deterministically.
- Each tab has `aria-controls` pointing at the active panel ID; each active panel has
  `role="tabpanel"` and `aria-labelledby` pointing back to the selected tab.
- Existing Review/Publish/Analysis tab tests remain green.

### Docs/fixtures
- No product docs changes expected beyond this ExecPlan; the accessibility contract is captured by
  tests and the shared primitive.
- Regenerate embedded UI bundle through `make build` if UI source changes update
  `internal/api/ui_dist`.

### Acceptance
- [x] Targeted TabNav/App Vitest tests pass.
- [x] `npm --prefix ui run typecheck` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19R2`, `19R3`, backend, schema or Epic 20 IA scope leaked into this
      slice.
- [x] Commit `19R1: add accessible tabs controller`.

### Progress log
- 2026-07-14: Started `19R1` after clean `19Q` commit. Spec-first and UI implementation QA rules
  apply because this slice changes shared UI interaction and accessibility behavior.
- 2026-07-14: Added roving-tabindex keyboard support, stable tab/panel helpers, wired current
  `TabNav` consumers to active tabpanels, added focused Vitest coverage, regenerated embedded UI
  assets, and completed targeted UI tests plus full DoD with exact Node 22.21.1:
  `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19r2-keyboard-path-combobox

### Context
`19R2` follows committed `19R1` and continues the accessibility primitives band. The onboarding
local path picker already exposes `role="combobox"` and renders suggestion buttons in a listbox,
but keyboard users cannot move through suggestions, select one with Enter, close with Escape, or
observe an active descendant. The fix should stay inside the existing `LocalPathCombobox` surface
and keep pointer selection behavior unchanged.

### Goals (must have)
- [x] Add active option state for local path suggestions.
- [x] Wire `aria-activedescendant` from the input to the active option.
- [x] Support ArrowDown/ArrowUp navigation with wraparound.
- [x] Support Enter selection of the active option and Escape close/clear active option.
- [x] Preserve pointer selection parity and existing onboarding `onSelect` side effects.
- [x] Cover keyboard-only selection, Escape close and active descendant behavior in Vitest.
- [x] Continue PR-1 with `19R3` accessible async announcements after `19R2` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change onboarding/source API contracts or path suggestion payloads.
- [ ] Do not change run/Q&A/editor async state; those were `19J..19L`.
- [ ] Do not add async alert/live-region behavior; that remains `19R3`.
- [ ] Do not redesign onboarding layout, path suggestion ranking or source validation.

### Implementation
1) Extend `LocalPathCombobox` with `activeIndex`, stable option IDs and active-option reset rules
   when the popover closes, suggestions change or the input value changes.
2) Add input `onKeyDown` handling for ArrowDown, ArrowUp, Enter and Escape. Arrow keys open the
   popover and move the active option; Enter applies the active suggestion through the same
   `onChange`/`onSelect` path as pointer clicks; Escape closes without selecting.
3) Add `aria-activedescendant` only while the popover is open and an active option exists, and mark
   the active option with a visual/semantic selected state.
4) Keep existing focus/blur timeout behavior, loading/error helper copy and pointer `onMouseDown`
   behavior intact.
5) Add focused component tests with mocked path-suggestion API and no backend/schema changes.

### Interfaces
Frontend-only component behavior change. No backend, schema, workspace, artifact, HTTP or
TypeScript app-contract payload changes.

### Tests
- Keyboard-only ArrowDown + Enter selects a suggestion, calls `onChange`, and invokes `onSelect`
  with the selected item.
- ArrowUp/ArrowDown update `aria-activedescendant` and wrap through suggestions predictably.
- Escape closes the popup and removes `aria-activedescendant` without selecting.
- Pointer click still selects the clicked suggestion and closes the popup.

### Docs/fixtures
- No product docs changes expected beyond this ExecPlan; the accessibility behavior is enforced by
  focused component tests.
- Regenerate embedded UI bundle through `make build` if UI source changes update
  `internal/api/ui_dist`.

### Acceptance
- [x] Targeted `LocalPathCombobox` Vitest tests pass.
- [x] `npm --prefix ui run typecheck` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19R3`, backend, schema or Epic 20 IA scope leaked into this slice.
- [x] Commit `19R2: make path combobox keyboard accessible`.

### Progress log
- 2026-07-14: Started `19R2` after clean `19R1` commit. Spec-first and UI implementation QA rules
  apply because this slice changes keyboard/focus behavior in an existing onboarding control.
- 2026-07-14: Added active option state, `aria-activedescendant`, ArrowUp/Down wraparound,
  Enter/Escape handling, pointer parity coverage, regenerated embedded UI assets, and completed
  targeted UI tests plus full DoD with exact Node 22.21.1: `make contracts`, `make test`,
  `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19r3-accessible-async-announcements

### Context
`19R3` follows committed `19R2` and completes the Epic 19 accessibility primitives band. The UI
has scattered `.status`, `.error-text` and `.error-banner` messages for onboarding validation,
doctor checks, first-run start and path-suggestion loading/error states. These messages are visible
but not consistently announced to assistive technology, and repo field diagnostics are not linked
back to the offending inputs.

### Goals (must have)
- [x] Add a shared async status primitive for assertive error alerts and polite progress/success
      announcements.
- [x] Use the primitive for onboarding top-level errors, source validation status, readiness status,
      first-run status and local path suggestion helper states.
- [x] Link repo name/source field errors with `aria-invalid` and `aria-describedby`.
- [x] Preserve existing copy, selectors, onboarding flow and backend/API contracts.
- [x] Cover alert, polite status and field diagnostic linkage in Vitest.
- [x] Continue PR-1 with `19S1` confirmed shell dead-code cleanup after `19R3`
      review/commit boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not redesign onboarding, Review/Publish IA or stage composition.
- [ ] Do not change validation/doctor API payloads or backend behavior.
- [ ] Do not implement global toast infrastructure, destructive confirmations or Epic 20 dialogs.
- [ ] Do not change path combobox keyboard behavior beyond helper announcement wiring.

### Implementation
1) Add `AsyncStatusMessage` in a shared component file. Error tone uses `role="alert"` /
   assertive semantics; progress/success/info/warning tones use polite `role="status"` semantics.
2) Replace onboarding top-level `error-banner`, source validation result, doctor result and
   first-run status messages with the shared primitive while preserving existing class names and
   text.
3) Extend `LocalPathCombobox` with optional `invalid`/`describedBy` props and announce its helper
   text through the shared primitive.
4) In onboarding repo rows, derive diagnostics per row, add stable diagnostics IDs, wire local
   repo-name and repo-source errors to `aria-invalid` / `aria-describedby`, and keep diagnostic
   text visible.
5) Add focused component tests for alert semantics, polite live status and field-linked
   diagnostics.

### Interfaces
Frontend-only component behavior change. No backend, schema, workspace, artifact, HTTP or
TypeScript app-contract payload changes. `LocalPathCombobox` receives optional internal UI props
only.

### Tests
- Error messages render as assertive alerts.
- Progress/success messages render as polite statuses.
- Repo name/source inputs with local diagnostics have `aria-invalid="true"` and
  `aria-describedby` pointing to visible diagnostic text.
- Local path combobox helper errors are announced while preserving existing keyboard tests.

### Docs/fixtures
- No product docs changes expected beyond this ExecPlan; the accessibility contract is enforced by
  focused component tests.
- Regenerate embedded UI bundle through `make build` if UI source changes update
  `internal/api/ui_dist`.

### Acceptance
- [x] Targeted accessibility primitive/onboarding/combobox Vitest tests pass.
- [x] `npm --prefix ui run typecheck` passes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no backend, schema, Epic 20 IA or `19S*` cleanup scope leaked into this
      slice.
- [x] Commit `19R3: announce async UI state accessibly`.

### Progress log
- 2026-07-14: Started `19R3` after clean `19R2` commit. Spec-first and UI implementation QA rules
  apply because this slice changes shared UI announcement semantics and field accessibility.
- 2026-07-14: Added shared async status announcements, wired onboarding/path-combobox statuses,
  linked repo field diagnostics with invalid/describedby attributes, added focused accessibility
  regression tests, regenerated embedded UI assets, and completed full DoD with exact Node
  22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19s1-shell-dead-code-cleanup

### Context
`19S1` follows committed `19R3` and starts deterministic quality-gate cleanup before ShellCheck is
enabled in `19S2`. The code audit confirms three shell/frontend-batch dead-code items:
`DEAD-011` unused `normalize_binary_flag`, `DEAD-012` unused `run_dod_precheck_make`, and
`DEAD-013` unused frontend status/reason assignments in active frontend E2E result paths.

### Goals (must have)
- [x] Remove the unused `normalize_binary_flag` helper from `scripts/full-run-batch-matrix.sh`.
- [x] Remove the unused `run_dod_precheck_make` helper from `scripts/full-run-batch.sh`.
- [x] Remove unused `frontend_status` / `frontend_reason` assignments while preserving
      `frontend_result_summary` validation side effects.
- [x] Verify targeted reference search no longer finds removed identifiers or assignments.
- [x] Keep active batch/frontend result classification unchanged.
- [x] Continue PR-1 with `19S2` ShellCheck in canonical lint after `19S1` review/commit boundary
      is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not enable ShellCheck yet; that is `19S2`.
- [ ] Do not rewrite batch/matrix harness control flow.
- [ ] Do not change release matrices, live provider requirements or frontend result taxonomy.
- [ ] Do not remove unrelated Go/UI dead code; later `19W*`/`19X` slices cover those.

### Implementation
1) Delete `normalize_binary_flag()` from `scripts/full-run-batch-matrix.sh`.
2) Delete `run_dod_precheck_make()` from `scripts/full-run-batch.sh`.
3) Replace `frontend_summary` parsing assignments with a validation-only call or scoped discard
   that keeps malformed/missing frontend result detection behavior unchanged.
4) Run targeted reference searches for removed function names and assignment names.
5) Run targeted bash syntax and existing script tests before full DoD.

### Interfaces
Shell cleanup only. No backend, schema, UI source, workspace, artifact or HTTP interfaces change.

### Tests
- `rg` finds no active references to `normalize_binary_flag` or `run_dod_precheck_make`.
- `rg` finds no `frontend_status=` / `frontend_reason=` assignments in `scripts/full-run-batch.sh`.
- Bash syntax checks pass for touched scripts.
- Existing Python script tests pass.

### Docs/fixtures
- No product docs changes expected beyond this ExecPlan; audit/backlog status is reconciled in the
  final Epic 19 reconciliation slice.

### Acceptance
- [x] Targeted reference search and bash syntax checks pass.
- [x] Python script tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no ShellCheck enablement, release-matrix, live-provider or unrelated
      cleanup scope leaked into this slice.
- [x] Commit `19S1: remove unused shell helpers`.

### Progress log
- 2026-07-14: Started `19S1` after clean `19R3` commit. Spec-first rules apply because this is a
  focused backlog cleanup slice with explicit audit IDs and deterministic test gates.
- 2026-07-14: Removed the confirmed unused shell helpers and frontend status/reason assignments,
  verified reference search and bash syntax, ran Python script tests, and completed full DoD with
  exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19s2-shellcheck-lint

### Context
`19S2` follows committed `19S1`. The audit requires canonical `make lint` to run ShellCheck for
production shell scripts before PR lint routing is changed in `19S3`. A current full ShellCheck pass
finds only intentional indirect trap callbacks (`SC2329`) and one assignment-export idiom
(`SC2163`), so this slice can add the gate with narrow documented suppressions and no harness
behavior changes.

### Goals (must have)
- [x] Add ShellCheck invocation to canonical `make lint` for production `scripts/**/*.sh`.
- [x] Require ShellCheck availability with an actionable local setup error.
- [x] Keep suppressions narrow and documented for trap callbacks and intentional assignment export.
- [x] Verify current scripts pass ShellCheck.
- [x] Verify an intentionally broken shell probe fails ShellCheck.
- [x] Update testing documentation for the new lint baseline.
- [x] Continue PR-1 with `19S3` required PR lint routing after `19S2` review/commit boundary is
      stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change PR workflow routing yet; that is `19S3`.
- [ ] Do not refactor batch/matrix shell logic beyond ShellCheck-required suppressions.
- [ ] Do not add live provider or network dependencies to required CI.
- [ ] Do not change release matrices, runtime behavior or shell script taxonomy.

### Implementation
1) Define a deterministic `SHELL_FILES` list in `Makefile` from production `scripts/*.sh` files.
2) Add ShellCheck availability check and `shellcheck $(SHELL_FILES)` to `make lint`.
3) Add documented `SC2329` suppressions for trap callback helper chains and fix/suppress the
   assignment export idiom without changing semantics.
4) Update `docs/TESTING_STRATEGY.md` to state that canonical lint includes gofmt, ShellCheck and
   UI typecheck.
5) Run current ShellCheck and a temporary failing probe to prove the gate catches shell defects.

### Interfaces
Tooling/docs only. No backend, schema, UI runtime, workspace, artifact, HTTP or live-provider
interfaces change.

### Tests
- `make lint` fails if `shellcheck` is missing.
- `shellcheck $(find scripts -name '*.sh')` passes on the current tree.
- A temporary script with a ShellCheck violation fails ShellCheck.
- Full DoD remains provider-free.

### Docs/fixtures
- Update `docs/TESTING_STRATEGY.md` canonical lint baseline.
- No schema/example/golden fixture changes.

### Acceptance
- [x] Targeted ShellCheck pass and failing probe pass.
- [x] `make lint` runs ShellCheck and UI typecheck.
- [x] `go test ./internal/docsync` passes after docs changes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no `19S3`, live-provider, release-matrix or unrelated shell refactor
      scope leaked into this slice.
- [x] Commit `19S2: add ShellCheck to lint`.

### Progress log
- 2026-07-14: Started `19S2` after clean `19S1` commit. Spec-first and docs-sync rules apply
  because this slice changes canonical lint behavior and documented testing baseline.
- 2026-07-14: Added ShellCheck to canonical `make lint`, documented the testing baseline, added
  narrow ShellCheck suppressions/fix for existing intentional shell idioms, verified current
  scripts plus a failing probe, and completed full DoD with exact Node 22.21.1: `make contracts`,
  `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19s3-required-pr-lint

### Context
`19S3` follows committed `19S2`. Canonical `make lint` now runs gofmt, ShellCheck and UI
typecheck, but PR CI still checks those surfaces only partially and independently. The backlog goal
is to make required PR lint call the canonical local target without adding live provider or network
provider dependencies.

### Goals (must have)
- [x] Add a provider-free PR/push workflow that sets up Go + Node and invokes `make lint`.
- [x] Install UI dependencies before `make lint` so the canonical UI typecheck path is used.
- [x] Remove duplicated partial UI typecheck from the UI workflow where safe.
- [x] Keep UI tests/build/determinism checks in the UI workflow.
- [x] Update testing documentation to list the canonical PR lint workflow.
- [x] Continue PR-1 with `19T` logs endpoint smoke coverage after `19S3` review/commit boundary is
      stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change branch protection settings or GitHub repository configuration.
- [ ] Do not add live provider checks to required PR CI.
- [ ] Do not change release workflow permissions or release evidence gates.
- [ ] Do not change `make lint` behavior itself; that was `19S2`.

### Implementation
1) Add `.github/workflows/lint.yml` with checkout, setup-go from `.go-version`, setup-node from
   `.node-version`, npm cache for `ui/package-lock.json`, `npm ci --prefix ui`, and `make lint`.
2) Remove the `Typecheck UI` step from `.github/workflows/ui.yml` because canonical lint now owns
   UI typecheck in PR CI; keep UI test/build/determinism/fresh-dist steps unchanged.
3) Update `docs/TESTING_STRATEGY.md` implemented jobs section to include the lint workflow and
   clarify UI workflow no longer duplicates typecheck.
4) Validate workflow YAML parsing and local `make lint`/full DoD.

### Interfaces
CI/tooling/docs only. No backend, schema, UI runtime, workspace, artifact, HTTP or live-provider
interfaces change.

### Tests
- YAML parse smoke passes for changed workflows.
- `make lint` passes locally and is the command used by the new workflow.
- Full DoD remains provider-free.

### Docs/fixtures
- Update `docs/TESTING_STRATEGY.md` CI job list.
- No schema/example/golden fixture changes.

### Acceptance
- [x] Workflow YAML parses.
- [x] New PR lint workflow invokes canonical `make lint`.
- [x] UI workflow no longer duplicates `npm run typecheck --prefix ui`.
- [x] `go test ./internal/docsync` passes after docs changes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no live-provider, branch-protection, release-permission or unrelated CI
      scope leaked into this slice.
- [x] Commit `19S3: route PR lint through make lint`.

### Progress log
- 2026-07-14: Started `19S3` after clean `19S2` commit. Spec-first and docs-sync rules apply
  because this slice changes required PR CI composition and documented testing baseline.
- 2026-07-14: Added provider-free `lint` workflow that runs canonical `make lint`, removed
  duplicated UI typecheck from the UI workflow while preserving tests/build/drift checks, updated
  testing docs, verified workflow YAML parsing, and completed full DoD with exact Node 22.21.1:
  `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19t-logs-smoke-coverage

### Context
`19T` follows committed `19S3`. The backend logs endpoint already has Go coverage for cursor,
limit, missing run and mixed event/runtime-output payloads, but required API smoke currently only
touches status, artifacts, run list and cancel. The backlog asks for provider-free smoke coverage
that fails on missing/malformed logs pagination behavior before PR merge.

### Goals (must have)
- [x] Extend `scripts/smoke-api.sh` to request run logs with explicit `cursor` and `limit`.
- [x] Validate logs response shape: matching `run_id`, array `items`, integer `next_cursor`,
      boolean `eof`, per-item cursor/message/kind and legal runtime stream values.
- [x] Validate pagination by fetching a second page from the first page `next_cursor`.
- [x] Validate invalid cursor returns `400 invalid_cursor`.
- [x] Add deterministic script tests for normal empty/non-empty pages, malformed payload,
      server/error status and invalid-cursor error handling without requiring a live ACP server.
- [x] Update smoke baseline documentation.
- [x] Continue PR-1 with `19U` deterministic mock Playwright CI after `19T` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change the HTTP logs endpoint contract or backend API routing.
- [ ] Do not add live-provider, external-network or hosted dependencies to required smoke.
- [ ] Do not change UI log rendering; that was covered by earlier UI slices.
- [ ] Do not change release workflows or release evidence gates.

### Implementation
1) Refactor `scripts/smoke-api.sh` into reusable helper functions plus a `main` entrypoint guarded
   by a lib-only environment variable for deterministic tests.
2) Add helpers that validate HTTP status/error code and logs JSON payloads with Python, failing on
   malformed shape, stale `run_id`, bad cursor progression, unknown `kind`, illegal `stream`, empty
   messages or non-boolean `eof`.
3) In the live smoke path, after the run succeeds and artifacts are fetched, request
   `/api/pipeline/runs/<run_id>/logs?cursor=0&limit=2`, then request a second page from
   `next_cursor`, and assert invalid cursor `-1` returns `400 invalid_cursor`.
4) Add `scripts/tests/smoke_api_logs_test.py` that sources the shell helpers in lib-only mode and
   covers empty/non-empty success, malformed payload, non-2xx status, wrong error code and invalid
   cursor validation.
5) Update `docs/TESTING_STRATEGY.md` smoke baseline to state that API smoke validates logs
   cursor/limit pagination and invalid params.

### Interfaces
Script/test/docs only. No schema, backend API, TypeScript, workspace, artifact, provider or release
interfaces change.

### Tests
- `python3 -m unittest scripts.tests.smoke_api_logs_test`.
- `bash -n scripts/smoke-api.sh`.
- `shellcheck scripts/smoke-api.sh`.
- `bash ./scripts/smoke-api.sh` under the fake runtime smoke environment.
- Full DoD remains provider-free.

### Docs/fixtures
- Update `docs/TESTING_STRATEGY.md`.
- No schema/example/golden fixture changes.

### Acceptance
- [x] Smoke validates first and second logs pages for the started run.
- [x] Smoke fails on malformed logs payloads.
- [x] Smoke fails on 5xx/non-2xx logs responses.
- [x] Smoke validates invalid cursor as `400 invalid_cursor`.
- [x] Script unit tests pass.
- [x] `go test ./internal/docsync` passes after docs changes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no backend contract, live-provider or unrelated CI scope leaked into
      this slice.
- [x] Commit `19T: smoke test run logs pagination`.

### Progress log
- 2026-07-14: Started `19T` after clean `19S3` commit. Spec-first and docs-sync rules apply
  because this slice changes required smoke coverage and documented testing baseline.
- 2026-07-14: Added logs page validation to API smoke, covered helper failure paths with
  deterministic Python tests, verified the live fake-server smoke path, and completed full DoD with
  exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19u-deterministic-mock-playwright-ci

### Context
`19U` follows committed `19T`. The repository already contains seven provider-free mock Playwright
specs under `ui/e2e/*mock*.spec.ts`, and each spec exercises console errors plus desktop/mobile
overflow checks. The gap is that required UI CI only runs Vitest/build/determinism checks; there is
no canonical `e2e:mock` runner, no CI step that guarantees `7 passed / 0 skipped`, and no local
script that runs the seven scenarios explicitly without live providers.

### Goals (must have)
- [x] Add a deterministic mock Playwright config that serves the Vite UI locally and excludes live
      specs.
- [x] Add a canonical `e2e:mock` package script/runner that runs exactly seven named mock
      scenarios.
- [x] Fail when any scenario is skipped, so a broken selector/scenario gate cannot silently pass.
- [x] Keep console-error and horizontal-overflow assertions active in the existing mock specs.
- [x] Wire the UI workflow to run the mock Playwright gate after build/determinism checks.
- [x] Update testing documentation for the required mock browser gate.
- [x] Continue PR-1 with `19U2` optional V8 coverage baseline after `19U` review/commit boundary is
      stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not add live provider checks or external repository/network dependencies.
- [ ] Do not change `e2e:live` or the live matrix harness.
- [ ] Do not redesign the Console UI; only wire existing deterministic mock scenarios.
- [ ] Do not introduce screenshot golden assertions in this slice.

### Implementation
1) Add `ui/playwright.mock.config.ts` with a local Vite `webServer`, Chromium project, single
   worker, no retries and `testMatch` limited to `**/*mock.spec.ts`.
2) Add `scripts/ui-mock-e2e.sh` that loops the seven explicit scenario names, sets
   `UI_E2E_SCENARIO`, runs the matching `ui/e2e/<scenario>.spec.ts` file through the mock config,
   captures list output, and fails unless all seven scenarios pass and zero skipped tests are
   observed.
3) Add `npm run e2e:mock --prefix ui` as the canonical package entrypoint for the script.
4) Add the required UI workflow step after deterministic build/fresh-dist checks.
5) Update `docs/TESTING_STRATEGY.md` implemented jobs and UI smoke baseline.

### Interfaces
Tooling/CI/docs only. No backend API, schema, TypeScript runtime contract, workspace, provider,
release or live matrix interfaces change.

### Tests
- `npm run e2e:mock --prefix ui` reports seven passed scenarios and zero skipped.
- A script contract test proves the scenario list stays at seven and each named spec exists.
- Workflow YAML parses.
- Full DoD remains provider-free.

### Docs/fixtures
- Update `docs/TESTING_STRATEGY.md`.
- No schema/example/golden fixture changes.

### Acceptance
- [x] `npm run e2e:mock --prefix ui` completes with `7 passed / 0 skipped`.
- [x] A missing/broken scenario selection would fail the runner.
- [x] UI workflow invokes the canonical mock gate.
- [x] `go test ./internal/docsync` passes after docs changes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no live-provider, live-harness or unrelated UI redesign scope leaked
      into this slice.
- [x] Commit `19U: add deterministic mock Playwright gate`.

### Progress log
- 2026-07-14: Started `19U` after clean `19T` commit. Spec-first, UI implementation QA and
  docs-sync rules apply because this slice changes required browser QA and documented testing
  baseline.
- 2026-07-14: Added mock Playwright config, canonical `e2e:mock` runner, UI workflow gate and
  contract tests for the seven explicit scenarios. Verified local `e2e:mock` as `7 passed / 0
  skipped` after installing local Chromium, then completed full DoD with exact Node 22.21.1:
  `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19u2-ui-v8-coverage-baseline

### Context
`19U2` follows committed `19U`. Vitest unit coverage is currently runnable only through the normal
test command, so there is no locked V8 coverage provider, deterministic coverage output or recorded
informational baseline that includes all `ui/src` files. This slice is explicitly optional and must
not become a hard quality threshold gate.

### Goals (must have)
- [x] Lock `@vitest/coverage-v8` in `ui/package.json`/`ui/package-lock.json` at the Vitest version.
- [x] Add a canonical UI coverage script that emits deterministic text and JSON summaries.
- [x] Configure coverage to include all `ui/src` files and exclude tests/setup type-only noise.
- [x] Record current line/statement/function/branch percentages as informational baseline only.
- [x] Update testing documentation for the optional coverage baseline.
- [ ] Continue PR-1 with `19V` Python runtime pinning after `19U2` review/commit boundary is
      stable.

### Non-goals
- [ ] Do not make coverage thresholds fail required CI in this slice.
- [ ] Do not add Playwright/browser coverage coupling.
- [ ] Do not change UI runtime behavior or test assertions.
- [ ] Do not change backend, schema, provider or live matrix behavior.

### Implementation
1) Install/lock `@vitest/coverage-v8` matching the repository's Vitest version.
2) Extend `ui/vite.config.ts` test coverage config with provider `v8`, text/json reporters,
   deterministic output directory and `include: ["src/**/*.{ts,tsx}"]`.
3) Add `coverage` package script that runs Vitest coverage in one-shot mode.
4) Add a lightweight Python contract test that confirms the script/dependency/config remain wired.
5) Run coverage once, capture the summary percentages in `docs/TESTING_STRATEGY.md`, and keep them
   informational.

### Interfaces
UI tooling/docs only. No backend API, schema, workspace, artifact, provider, Playwright or release
interfaces change.

### Tests
- `npm run coverage --prefix ui` completes after clean install state and writes text/json coverage.
- `python3 -m unittest scripts/tests/ui_coverage_contract_test.py`.
- `npm run typecheck --prefix ui`.
- Full DoD remains provider-free.

### Docs/fixtures
- Update `docs/TESTING_STRATEGY.md`.
- No schema/example/golden fixture changes.

### Acceptance
- [x] `@vitest/coverage-v8` is locked and version changes require package/lockfile diff.
- [x] Coverage includes all `ui/src` files.
- [x] Coverage summary is deterministic enough for local comparison and emits JSON.
- [x] No coverage thresholds gate CI.
- [x] `go test ./internal/docsync` passes after docs changes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no Playwright, runtime behavior or hard-threshold scope leaked into
      this slice.
- [x] Commit `19U2: add deterministic UI coverage baseline`.

### Progress log
- 2026-07-14: Started `19U2` after clean `19U` commit. Spec-first and docs-sync rules apply
  because this slice changes UI test tooling and documented testing baseline.
- 2026-07-14: Locked `@vitest/coverage-v8`, added `npm run coverage --prefix ui`, configured
  V8 text/JSON coverage over `ui/src`, recorded the informational baseline
  (`85.36/77.03/88.54/85.22` statements/branches/functions/lines), ignored generated coverage
  artifacts, and completed full DoD with exact Node 22.21.1: `make contracts`, `make test`,
  `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19v-python-runtime-pinning

### Context
`19V` follows committed `19U2`. Python tests and release verifier scripts currently use bare
`python3`; required CI does not install a repo-declared Python version, so the Python script/test
surface can drift across developer machines and runners. The current green local baseline is
Python `3.10.8`, so this slice pins that exact version and routes repository-owned Python test
entrypoints through a version-checking wrapper.

### Goals (must have)
- [x] Add `.python-version` with the exact supported Python version.
- [x] Add `scripts/run-python.sh` that discovers only the required Python version and fails before
      running commands when no matching interpreter is available.
- [x] Route `make test` Python unittest discovery through the wrapper.
- [x] Add setup-python and wrapper usage to workflows that run Python scripts/tests.
- [x] Add regression tests for matching and wrong Python interpreter discovery.
- [x] Update CONTRIBUTING and TESTING_STRATEGY bootstrap documentation.
- [x] Continue PR-1 with `19W1` runtime-draft wrapper cleanup after `19V` review/commit boundary
      is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not rewrite every shell heredoc that invokes `python3`; this slice pins repository test
      and workflow entrypoints, not provider-authored runtime commands.
- [ ] Do not add Python package dependency management.
- [ ] Do not change release evidence semantics or live provider gates.
- [ ] Do not change Go/Node version policy.

### Implementation
1) Add `.python-version` containing `3.10.8`.
2) Add `scripts/run-python.sh` modeled after `scripts/run-go.sh`, supporting `ACP_PYTHON_BIN` and
   `ACP_PYTHON_TOOL_CANDIDATES` for local/toolchain overrides.
3) Add `PYTHON ?= ./scripts/run-python.sh` to `Makefile` and use it for Python unittest discovery.
4) Update backend/release workflows with pinned `actions/setup-python` configured from
   `.python-version`; route Python commands through `./scripts/run-python.sh`.
5) Add `scripts/tests/run_python_test.py` for correct/wrong version behavior.
6) Update docs to state Python `3.10.8` is required for tests/scripts.

### Interfaces
Tooling/CI/docs only. No backend API, schema, UI runtime, provider, workspace, artifact or release
matrix interfaces change.

### Tests
- `python3 -m unittest scripts/tests/run_python_test.py`.
- Wrong interpreter fixture fails before the requested command runs.
- Correct interpreter fixture runs normally.
- Workflow YAML parses.
- Full DoD remains provider-free.

### Docs/fixtures
- Update `CONTRIBUTING.md`.
- Update `docs/TESTING_STRATEGY.md`.
- No schema/example/golden fixture changes.

### Acceptance
- [x] `.python-version` is present and workflows read it.
- [x] Makefile Python tests use `scripts/run-python.sh`.
- [x] Wrong Python version fails before unittest discovery.
- [x] Correct Python version runs script tests.
- [x] `go test ./internal/docsync` passes after docs changes.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms no provider/live/runtime behavior scope leaked into this slice.
- [x] Commit `19V: pin Python test runtime`.

### Progress log
- 2026-07-14: Started `19V` after clean `19U2` commit. Spec-first and docs-sync rules apply
  because this slice changes developer/CI runtime bootstrap and documented testing baseline.
- 2026-07-14: Added `.python-version`, `scripts/run-python.sh`, wrapper tests, setup-python
  workflow steps and docs. Verified wrong-version fail-fast behavior and completed full DoD with
  exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19w1-runtime-draft-wrapper-cleanup

### Context
`19W1` follows committed `19V`. `docs/BACKLOG.md` marks `DEAD-003` as an orchestrator
runtime-draft cleanup: remove local forwarding wrappers and use the canonical
`internal/runtimedrafts` package directly. The behavior must remain unchanged; this is a
dead-surface deletion slice after the P1 backend correctness work has stabilized runtime draft
validation and publication.

### Goals (must have)
- [x] Remove orchestrator-local runtime-draft type aliases and forwarding wrappers.
- [x] Replace active orchestrator wrapper call sites with direct `runtimedrafts` calls.
- [x] Keep orchestrator-specific publication helpers that copy validated draft outputs into the
      workspace/final staging surface.
- [x] Verify runtime-draft and orchestrator packages still pass targeted tests.
- [x] Verify no removed runtime-draft wrapper identifiers remain.
- [x] Continue PR-1 with `19W2` sharding wrapper cleanup after `19W1` review/commit boundary is
      stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change runtime draft manifest schema, validation rules or prompt contracts.
- [ ] Do not change provider recovery behavior or artifact quality policy.
- [ ] Do not change fake runtime draft output generation.
- [ ] Do not start `19W2` sharding cleanup in this slice.

### Implementation
1) Convert orchestrator state/test types from `runtimeDraftManifest` and `runtimeDraftOutput` to
   `runtimedrafts.Manifest` and `runtimedrafts.Output`.
2) Replace `validateRequiredRuntimeDraftArtifacts(...)` call sites with
   `runtimedrafts.ValidateRequiredManifest(...)` using the current task fields.
3) Delete unused forwarding wrappers in `internal/orchestrator/runtime_drafts.go`:
   manifest load/validation wrappers, required-manifest-file wrapper, required-artifacts wrapper
   and task-validation wrapper.
4) Keep `applyRuntimeDraftOutputs(...)` and `draftManifestHasPrefix(...)` because they contain
   orchestrator-specific publish/index behavior rather than package forwarding.
5) Run gofmt and focused searches for removed identifiers.

### Interfaces
Internal Go package cleanup only. No backend API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/runtimedrafts ./internal/orchestrator -count=1`.
- `./scripts/run-go.sh test ./internal/runtime/providercommon ./internal/runtime/fakeruntime -count=1`
  as regression coverage for runtime draft consumers outside orchestrator.
- Reference search for removed wrapper identifiers returns no active code hits.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Orchestrator active call sites import and call `internal/runtimedrafts` directly.
- [x] Removed wrapper identifiers have no remaining code references.
- [x] Targeted runtime draft/orchestrator tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this slice is behavior-neutral dead-code cleanup and does not include
      `19W2` sharding work.
- [x] Commit `19W1: remove runtime draft wrappers`.

### Progress log
- 2026-07-14: Started `19W1` after clean `19V` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes internal
  forwarding dead surfaces only.
- 2026-07-14: Replaced orchestrator runtime-draft aliases/wrappers with direct
  `runtimedrafts` imports/calls, kept orchestrator-specific draft publication helpers, verified
  removed identifiers have no `internal/orchestrator` references, and completed targeted tests plus
  full DoD with exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19w2-sharding-wrapper-cleanup

### Context
`19W2` follows committed `19W1`. `docs/BACKLOG.md` marks `DEAD-004` as removal of four legacy
sharding planner/artifact wrappers. Current code still has unused forwarding helpers around the
canonical sharding planner input functions and taskrun path builder. This slice removes only those
dead wrappers; active planner/scheduler/store entrypoints remain unchanged.

### Goals (must have)
- [x] Remove unused sharding planner wrappers `resolveRepoPath`, `planScopePaths` and
      `discoverHeuristicShardPaths`.
- [x] Remove unused sharding artifact wrapper `singleShardTaskrunPath`.
- [x] Keep canonical `*ForInput` planner functions and active taskrun path helpers unchanged.
- [x] Verify deterministic sharding tests still pass.
- [x] Verify removed wrapper identifiers have no remaining code references.
- [x] Continue PR-1 with `19W3` provider argument wrapper cleanup after `19W2` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change shard planning heuristics, semantic graph discovery, path filtering or shard
      IDs.
- [ ] Do not change shard summary/plan JSON contracts.
- [ ] Do not change scheduler, checkpoint replay or best-effort failure policy behavior.
- [ ] Do not start `19W3` provider argument cleanup in this slice.

### Implementation
1) Delete the unused `pipelineExecution` forwarding methods in `sharding_planner.go`.
2) Delete the unused `discoverHeuristicShardPaths` wrapper and keep
   `discoverHeuristicShardPathsWithMeta` as the canonical implementation.
3) Delete the unused `singleShardTaskrunPath` artifact helper and keep `shardTaskrunPath`.
4) Run gofmt and reference search for the removed identifiers.

### Interfaces
Internal Go dead-code cleanup only. No public API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/orchestrator -run 'Shard|Sharding' -count=1`.
- `./scripts/run-go.sh test ./internal/orchestrator -count=1`.
- Reference search for removed wrapper identifiers returns no active code hits.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Four legacy sharding wrapper functions are removed without aliases/replacements.
- [x] Canonical sharding planner/artifact helpers remain unchanged.
- [x] Deterministic sharding tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is behavior-neutral dead-code cleanup and does not include
      `19W3` provider argument work.
- [x] Commit `19W2: remove sharding wrappers`.

### Progress log
- 2026-07-14: Started `19W2` after clean `19W1` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes internal
  sharding forwarding dead surfaces only.
- 2026-07-14: Removed the four legacy sharding wrappers, confirmed no remaining
  `internal/orchestrator` references, ran deterministic sharding/orchestrator tests, and completed
  full DoD with exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19w3-provider-argument-wrapper-cleanup

### Context
`19W3` follows committed `19W2`. `docs/BACKLOG.md` marks `DEAD-005` as provider argument wrapper
cleanup: remove legacy default-argument entry points and keep the permission-aware builders as the
only implementation surface. The active Claude/Qwen/Codex adapters already call
`build*ArgsWithPermissions(...)` when no custom args are supplied.

### Goals (must have)
- [x] Remove unused `buildDefaultClaudeArgs`, `buildDefaultQwenArgs` and `buildDefaultCodexArgs`.
- [x] Keep permission-aware builders for Claude, Qwen and Codex unchanged.
- [x] Keep custom-argument sanitization behavior unchanged.
- [x] Verify Claude/Qwen/Codex adapter argument tests still pass.
- [x] Verify removed default entry point identifiers have no remaining code references.
- [x] Continue PR-1 with `19W4` docflow compatibility helper cleanup after `19W3` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change provider command arguments, sandbox/permission modes or defaults.
- [ ] Do not change provider list or runtime selection.
- [ ] Do not change prompt contracts, recovery policy or process execution behavior.
- [ ] Do not start `19W4` docflow compatibility cleanup in this slice.

### Implementation
1) Delete the three unused `buildDefault*Args` functions from Claude/Qwen/Codex adapters.
2) Keep `build*ArgsWithPermissions(...)` and existing `build*ArgsWithIncludeDirectories(...)`
   helpers because tests and explicit include-dir coverage still use them.
3) Run gofmt and reference search for the removed identifiers.

### Interfaces
Internal Go dead-code cleanup only. No backend API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/runtime/claudecode ./internal/runtime/qwencode ./internal/runtime/codexcode -count=1`.
- Reference search for removed default entry point identifiers returns no active code hits.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Three legacy provider default-argument entry points are removed without aliases/replacements.
- [x] Permission-aware builders and custom-arg stripping paths remain unchanged.
- [x] Claude/Qwen/Codex argument tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is behavior-neutral dead-code cleanup and does not include
      `19W4` docflow work.
- [x] Commit `19W3: remove provider argument wrappers`.

### Progress log
- 2026-07-14: Started `19W3` after clean `19W2` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes internal
  provider argument forwarding dead surfaces only.
- 2026-07-14: Removed the three legacy default provider argument functions, confirmed no remaining
  `internal/runtime` references, ran Claude/Qwen/Codex adapter tests, and completed full DoD with
  exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19w4-docflow-compatibility-helper-cleanup

### Context
`19W4` follows committed `19W3`. `docs/BACKLOG.md` marks `DEAD-006` as docflow compatibility
helper cleanup: two local orchestrator helpers have been superseded by the artifact-quality layer.
Current docflow already calls `artifactquality.HasRepoSpecificCitationSurface(...)` and
`artifactquality.IsGenericRuntimeSummaryCitation(...)` directly; the remaining local helpers are
unused wrappers.

### Goals (must have)
- [x] Remove unused `manifestHasRepoSpecificCitationSurface`.
- [x] Remove unused `isGenericRuntimeSummaryCitation`.
- [x] Keep the artifact-quality package as the only implementation surface for those checks.
- [x] Verify docflow and artifact-quality tests still pass.
- [x] Verify removed helper identifiers have no remaining code references.
- [x] Continue PR-1 with `19W5a` review diff residual cleanup after `19W4` review/commit boundary
      is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change artifact-quality warning semantics or wording.
- [ ] Do not change final/citation index generation.
- [ ] Do not change docflow rendering, promotion or collect manifest validation.
- [ ] Do not start `19W5a` package-local residual cleanup in this slice.

### Implementation
1) Delete the two unused docflow wrapper functions.
2) Run gofmt and reference search for the removed identifiers.

### Interfaces
Internal Go dead-code cleanup only. No public API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/orchestrator ./internal/artifactquality -count=1`.
- Reference search for removed helper identifiers returns no active code hits.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Two local docflow compatibility helpers are removed without aliases/replacements.
- [x] Existing artifact-quality direct calls remain unchanged.
- [x] Docflow and artifact-quality tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is behavior-neutral dead-code cleanup and does not include
      `19W5a` work.
- [x] Commit `19W4: remove docflow compatibility helpers`.

### Progress log
- 2026-07-14: Started `19W4` after clean `19W3` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes internal
  docflow compatibility dead surfaces only.
- 2026-07-14: Removed the two unused docflow compatibility helpers, confirmed no remaining
  references, ran orchestrator/artifact-quality tests, and completed full DoD with exact Node
  22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19w5a-review-diff-residual-cleanup

### Context
`19W5a` follows committed `19W4`. The `DEAD-007` package-local residual track starts in
`internal/api/review_diff.go`. Current API Git diff tests initialize repositories through
`initGitWorkspaceForDiffTest`; the production file still contains an unused `ensureGitDiffTestRepo`
helper that duplicates test-only setup behavior.

### Goals (must have)
- [x] Remove unused `ensureGitDiffTestRepo` from `internal/api/review_diff.go`.
- [x] Keep Git diff endpoint behavior and test helpers unchanged.
- [x] Verify API/review diff tests still pass.
- [x] Verify the removed helper identifier has no remaining code references.
- [x] Continue PR-1 with `19W5b` model store residual cleanup after `19W5a` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change Git diff response shape, filtering, binary handling or hunk parsing.
- [ ] Do not move test helpers between packages.
- [ ] Do not change Git commit/proposal branch endpoints.
- [ ] Do not start `19W5b` model store cleanup in this slice.

### Implementation
1) Delete `ensureGitDiffTestRepo`.
2) Run gofmt and reference search for the removed identifier.

### Interfaces
Internal Go dead-code cleanup only. No public API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/api -run 'GitDiff|RunReview' -count=1`.
- `./scripts/run-go.sh test ./internal/api -count=1`.
- Reference search for `ensureGitDiffTestRepo` returns no active code hits.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Review diff residual helper is removed without alias/replacement.
- [x] API Git diff/run review tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is behavior-neutral dead-code cleanup and does not include
      `19W5b` work.
- [x] Commit `19W5a: remove review diff residuals`.

### Progress log
- 2026-07-14: Started `19W5a` after clean `19W4` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes one internal
  API dead helper only.
- 2026-07-14: Removed unused `ensureGitDiffTestRepo`, confirmed no remaining references, ran
  targeted API Git diff/run review tests plus full API package tests, and completed full DoD with
  exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19w5b-model-store-residual-cleanup

### Context
`19W5b` follows committed `19W5a`. The model store package-local residual is the unused
`Store.removeFile` helper in `internal/model/store.go`; current model store behavior writes and
lists entity/edge YAML files but has no active delete path.

### Goals (must have)
- [x] Remove unused `Store.removeFile`.
- [x] Keep model store apply/list behavior unchanged.
- [x] Verify model store/golden tests still pass.
- [x] Verify the removed helper identifier has no remaining code references.
- [x] Continue PR-1 with `19W5c` orchestrator quality residual cleanup after `19W5b`
      review/commit boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not add model deletion behavior.
- [ ] Do not change entity/edge ID normalization, collision remapping or owner-team validation.
- [ ] Do not change YAML layout or schema.
- [ ] Do not start `19W5c` orchestrator quality cleanup in this slice.

### Implementation
1) Delete the unused `Store.removeFile` method.
2) Run gofmt and reference search for the removed identifier.

### Interfaces
Internal Go dead-code cleanup only. No public API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/model -count=1`.
- Reference search for `removeFile` in `internal/model` returns no active code hits.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Model store residual helper is removed without alias/replacement.
- [x] Model store tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is behavior-neutral dead-code cleanup and does not include
      `19W5c` work.
- [x] Commit `19W5b: remove model store residuals`.

### Progress log
- 2026-07-14: Started `19W5b` after clean `19W5a` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes one internal
  model store dead helper only.
- 2026-07-14: Removed unused `Store.removeFile`, confirmed no remaining references, ran model
  store tests, and completed full DoD with exact Node 22.21.1: `make contracts`, `make test`,
  `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19w5c-orchestrator-quality-residual-cleanup

### Context
`19W5c` follows committed `19W5b`. `docs/CODE_AUDIT_2026-07-10.md` identifies the
package-local residual at `internal/orchestrator/quality.go:241`. In the current tree this is the
unused `assessLiveReportSurfaceWarnings` wrapper; active code and tests call
`assessLiveReportSurfaceSignals` directly.

### Goals (must have)
- [x] Remove unused `assessLiveReportSurfaceWarnings`.
- [x] Keep run quality summary generation and artifact-quality signal behavior unchanged.
- [x] Verify orchestrator quality tests still pass.
- [x] Verify the removed helper identifier has no remaining code references.
- [x] Continue PR-1 with `19W5d` reports compiler residual cleanup after `19W5c`
      review/commit boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change artifact-quality signal thresholds, messages or ordering.
- [ ] Do not change report render context behavior or run warning synthesis.
- [ ] Do not change runtime recovery counters or failure classification.
- [ ] Do not start `19W5d` reports compiler cleanup in this slice.

### Implementation
1) Delete the unused `assessLiveReportSurfaceWarnings` helper.
2) Run gofmt and reference search for the removed identifier.

### Interfaces
Internal Go dead-code cleanup only. No public API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/orchestrator -run 'AssessLiveReportSurfaceSignals|RuntimeDiagnosticCounters|AssessRunArtifactInventory' -count=1`.
- `./scripts/run-go.sh test ./internal/orchestrator -count=1`.
- Reference search for `assessLiveReportSurfaceWarnings` returns no active code hits.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Orchestrator quality residual helper is removed without alias/replacement.
- [x] Targeted orchestrator quality tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is behavior-neutral dead-code cleanup and does not include
      `19W5d` work.
- [x] Commit `19W5c: remove orchestrator quality residuals`.

### Progress log
- 2026-07-14: Started `19W5c` after clean `19W5b` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes one internal
  orchestrator quality dead helper only.
- 2026-07-14: Removed unused `assessLiveReportSurfaceWarnings`, confirmed no remaining code
  references in `internal/orchestrator`, ran targeted orchestrator quality tests and full
  orchestrator package tests, and completed full DoD with exact Node 22.21.1: `make contracts`,
  `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19w5d-reports-compiler-residual-cleanup

### Context
`19W5d` follows committed `19W5c`. `docs/CODE_AUDIT_2026-07-10.md` identifies the
package-local residual at `internal/reports/compiler.go:266`. In the current tree this is the
unused `writeStringList` helper; active coverage rendering uses `writeStringListWithFallback`.

### Goals (must have)
- [x] Remove unused `writeStringList`.
- [x] Keep report compiler output unchanged.
- [x] Verify reports compiler tests still pass.
- [x] Verify the removed helper identifier has no remaining code references.
- [x] Continue PR-1 with `19W5e` prompt-contract residual cleanup after `19W5d`
      review/commit boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change coverage/findings/changelog markdown content.
- [ ] Do not change incomplete-analysis fallback wording.
- [ ] Do not change report artifact paths, labels or kinds.
- [ ] Do not start `19W5e` prompt-contract cleanup in this slice.

### Implementation
1) Delete the unused `writeStringList` helper.
2) Run gofmt and reference search for the removed identifier.

### Interfaces
Internal Go dead-code cleanup only. No public API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/reports -count=1`.
- Reference search for `writeStringList` confirms only `writeStringListWithFallback` remains as an
  active helper.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Reports compiler residual helper is removed without alias/replacement.
- [x] Reports compiler tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is behavior-neutral dead-code cleanup and does not include
      `19W5e` work.
- [x] Commit `19W5d: remove reports compiler residuals`.

### Progress log
- 2026-07-14: Started `19W5d` after clean `19W5c` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes one internal
  reports compiler dead helper only.
- 2026-07-14: Removed unused `writeStringList`, confirmed the active
  `writeStringListWithFallback` helper remains unchanged, ran reports compiler tests, and
  completed full DoD with exact Node 22.21.1: `make contracts`, `make test`, `make lint`,
  `make build`.

### Plan ID
EP-20260714-epic-19-19w5e-prompt-contract-residual-cleanup

### Context
`19W5e` follows committed `19W5d` and closes the `DEAD-007` package-local residual track.
`docs/CODE_AUDIT_2026-07-10.md` identifies the residual at
`internal/runtime/promptcontract/collect_repair.go:213`. In the current tree this is the unused
`firstNonEmpty` helper; collect repair prompt composition uses explicit path/evidence selection
logic instead.

### Goals (must have)
- [x] Remove unused `firstNonEmpty`.
- [x] Keep collect manifest and collect artifact-pair repair prompts unchanged.
- [x] Verify prompt-contract tests still pass.
- [x] Verify the removed helper identifier has no remaining code references.
- [x] Continue PR-1 with `19X` UI dead-surface cleanup after `19W5e` review/commit boundary is
      stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change collect repair prompt wording, evidence ranking or fallback behavior.
- [ ] Do not change runtime draft validation or repair retry policy.
- [ ] Do not change schemas, fixtures or live-provider gates.
- [ ] Do not start `19X` UI dead-surface cleanup in this slice.

### Implementation
1) Delete the unused `firstNonEmpty` helper.
2) Run gofmt and reference search for the removed identifier.

### Interfaces
Internal Go dead-code cleanup only. No public API, schema, workspace, UI, provider or release
interfaces change.

### Tests
- `./scripts/run-go.sh test ./internal/runtime/promptcontract -count=1`.
- Reference search for `firstNonEmpty` returns no active code hits.
- Staticcheck/canonical lint remains green through full DoD.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or golden fixtures should change.

### Acceptance
- [x] Prompt-contract residual helper is removed without alias/replacement.
- [x] Prompt-contract tests pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is behavior-neutral dead-code cleanup and does not include `19X`
      work.
- [x] Commit `19W5e: remove prompt contract residuals`.

### Progress log
- 2026-07-14: Started `19W5e` after clean `19W5d` commit. Spec-first rule applies; no schema,
  model-fixture or docs-visible behavior skill is required because the slice removes one internal
  prompt-contract dead helper only.
- 2026-07-14: Removed unused `firstNonEmpty`, confirmed no remaining code references, ran
  prompt-contract tests, and completed full DoD with exact Node 22.21.1: `make contracts`,
  `make test`, `make lint`, `make build`.

### Plan ID
EP-20260714-epic-19-19x-ui-dead-surface-cleanup

### Context
`19X` follows committed `19W5e` and closes the UI dead-surface cleanup group
(`DEAD-008`, `DEAD-009`, `DEAD-010`). A strict TypeScript probe with
`--noUnusedLocals --noUnusedParameters` still reports the original `DEAD-008` entries:
`Diagnostic` in `App.tsx`, `issueCount` in `AnalysisRunProgress`, and `nonDiagramArtifacts` in
`ReviewEvidenceWorkbench`. Reference search also confirms the legacy synchronous QA wrapper
(`QAAskResponse`, `QAAskRawResponse`, `askArchitectureQuestion`) and hook facade cleanup members
(`clearRunReviewSummary`, `clearGitDiff`, `selectedDiffPath`, `setSelectedDiffPath`) have no active
consumers outside internal propagation.

### Goals (must have)
- [x] Remove the unused `Diagnostic` import from `App.tsx`.
- [x] Remove unused `issueCount` and `nonDiagramArtifacts` props/call-site wiring from
      `StagePanels.tsx`.
- [x] Remove the legacy synchronous QA ask client surface from `qaApi.ts`.
- [x] Remove unused run-review/Git-diff facade cleanup members from hook return surfaces.
- [x] Enable TypeScript `noUnusedLocals` and `noUnusedParameters` so these regressions fail
      canonical UI typecheck.
- [x] Keep request-gated run review, Git diff, async QA and artifact review behavior unchanged.
- [x] Continue PR-1 with final Epic 19 docs/backlog reconciliation after `19X` review/commit
      boundary is stable.
- [ ] Keep this plan active until final Epic 19 reconciliation archives completed PR-1 slice
      plans.

### Non-goals
- [ ] Do not change Review/Publish IA, Git publication semantics or Epic 20 UX scope.
- [ ] Do not remove active `Diagnostic` type consumers in validation/onboarding/editor surfaces.
- [ ] Do not change API endpoints or backend QA compatibility.
- [ ] Do not refactor StagePanels beyond deleting confirmed dead props/surfaces.

### Implementation
1) Remove `Diagnostic` from `App.tsx` imports.
2) Remove `issueCount` from `AnalysisRunProgress` props and call site.
3) Remove `nonDiagramArtifacts` from `ReviewEvidenceWorkbench` props and call site.
4) Delete `QAAskResponse`, `QAAskRawResponse`, and `askArchitectureQuestion` from `qaApi.ts`.
5) Stop returning unused cleanup members from `useRunReview`, `useGitDiff`, and `useRunExplorer`;
   keep internal request abort/clear behavior where it is actively used.
6) Add `noUnusedLocals` and `noUnusedParameters` to `ui/tsconfig.json`.

### Interfaces
Frontend-only cleanup. No backend API, schema, workspace, provider or release interface changes.
The public TypeScript typecheck contract becomes stricter for unused locals/parameters.

### Tests
- `ACP_NODE_TOOL_CANDIDATES=... ./scripts/run-npm.sh run typecheck --prefix ui`.
- `ACP_NODE_TOOL_CANDIDATES=... ./scripts/run-npm.sh run test --prefix ui -- --run`.
- `ACP_NODE_TOOL_CANDIDATES=... ./scripts/run-npm.sh run e2e:mock --prefix ui`.
- Reference search for removed UI identifiers shows no active source consumers.
- Full DoD remains green.

### Docs/fixtures
- `docs/PLANS.md` only. No behavior docs, schemas, examples or fixtures should change.

### Acceptance
- [x] `noUnusedLocals`/`noUnusedParameters` are enabled and UI typecheck passes.
- [x] Legacy synchronous QA client and unused hook facade members are removed.
- [x] UI Vitest and deterministic mock Playwright pass.
- [x] Full slice DoD passes with exact Node `22.21.1`:
      `make contracts`, `make test`, `make lint`, `make build`.
- [x] Self-review confirms this is dead-surface deletion only and does not include Epic 20 UX
      changes.
- [x] Commit `19X: remove UI dead surfaces`.

### Progress log
- 2026-07-14: Started `19X` after clean `19W5e` commit. Using `ui-implementation-qa` for UI
  implementation and verification. No schema/model-fixture/docs-visible behavior skill is required
  because the slice deletes confirmed frontend dead surfaces and tightens typecheck only.
- 2026-07-14: Removed the unused `Diagnostic` import, stale `AnalysisRunProgress.issueCount` and
  `ReviewEvidenceWorkbench.nonDiagramArtifacts` props, legacy synchronous QA ask wrapper, and
  unused run-review/Git-diff facade members. Enabled `noUnusedLocals`/`noUnusedParameters`, ran UI
  typecheck, Vitest, deterministic mock Playwright (`7 passed / 0 skipped`), and completed full DoD
  with exact Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`. `make build`
  regenerated `internal/api/ui_dist` for the UI source change.

### Plan ID
EP-20260714-epic-19-19z-final-reconciliation

### Context
`19Z` follows committed `19X` and reconciles PR-1 status after all Epic 19 implementation slices
are complete. The branch has also been deliberately merged with `main`, preserving the concurrent
workspace-health/Karpathy planning work from `main` and the complete Epic 19 slice evidence from
this branch. Current update: the reconciled program later merged into `main` at `02716bb`.

### Goals (must have)
- [x] Record the then-current Epic 19 status as implementation-complete pending PR review/merge.
- [x] Update stakeholder status so PR-1 no longer claims the stale `19I` current-work marker.
- [x] Preserve all `19A..19X` plan evidence and `main` active plan evidence after merge conflict
      resolution.
- [x] Record that Epic 20 remains blocked until PR-1 merges into `main`.
- [x] Complete PR-1 review/push/merge into `main`.
- [ ] Archive this completed reconciliation plan during the next tracker reconciliation.

### Non-goals
- [ ] Do not start Epic 20 in this reconciliation slice.
- [ ] Do not run live providers or trusted-machine release gates.
- [ ] Do not change canonical release matrices or provider lists.
- [x] Do not mark PR-1 as merged before it is actually merged into `main`.

### Implementation
1) Sync `docs/BACKLOG.md`, `docs/STAKEHOLDER_DOC.md` and the Epic 19 active plan status.
2) Resolve `main` merge conflict in `docs/PLANS.md` by preserving both Epic 19 and concurrent
   `main` active plans.
3) Run docs-sync checks and full deterministic DoD after the merge and status updates.

### Interfaces
Documentation/status reconciliation only. No public API, schema, workspace, UI, provider or
release interface changes.

### Tests
- `./scripts/run-go.sh test ./internal/docsync -count=1`.
- `git diff --check`.
- Full deterministic DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Docs/fixtures
- `docs/BACKLOG.md`, `docs/PLANS.md`, `docs/STAKEHOLDER_DOC.md`.

### Acceptance
- [x] At the `19Z` commit boundary, backlog/stakeholder/plan status consistently said Epic 19 was
      implementation-complete but pending PR review/merge.
- [x] Full DoD passes after merging `main`.
- [x] Commit `19Z: reconcile Epic 19 completion`.
- [x] Current backlog/stakeholder/plan status records the later merge at `02716bb`.

### Progress log
- 2026-07-14: Reconciled Epic 19 backlog/stakeholder/plan status, preserved both Epic 19 and
  `main` active-plan evidence during merge conflict resolution, removed the merged unused
  `doctorWarnings` local exposed by the new noUnused gate, regenerated `internal/api/ui_dist`, and
  completed full DoD with exact Node 22.21.1: `make contracts`, `make test`, `make lint`,
  `make build`.
- 2026-07-15: Recorded the later merge into `main` at `02716bb`; archive remains for the next
  tracker reconciliation.

### Plan ID
EP-20260623-karpathy-adoption-roadmap

### Context
The Karpathy LLM Wiki analysis in `docs/KARPATHY_LLM_WIKI_COMPARISON_2026-06-18.md` and `docs/KARPATHY_ADOPTION_ANALYSIS_2026-06-18.md` concluded that ACP should not become a free-form personal/wiki system. The useful import is narrower: treat accepted architecture knowledge as a maintained, Git-versioned, provenance-backed artifact that compounds through explicit review.

Current ACP already has the stronger foundation needed for this:
- source repos and imported docs are raw/read-only inputs;
- runtime writes staged artifacts under explicit write roots;
- orchestrator validates schemas/manifests and controls canonical promotion;
- stable architecture knowledge lives in `reports/*`, `model/*`, `proposals/*`, `charter/*`, `reports/changelog/*` and run-scoped taskrun artifacts;
- `qa.ask` is intentionally run-scoped and does not mutate canonical reports/model.

The implementation roadmap must preserve MVP constraints:
- no hosted mode;
- no security/compliance enforcement expansion;
- required CI remains deterministic/fake and does not require live providers or network;
- canonical artifacts are not mutated directly by runtime providers;
- schema/contract changes require synchronized docs, validators, fixtures and tests.

This plan is a roadmap for owner selection. It does not override the existing queue unless the owner chooses one of its slices as the next engineering workstream.

### Goals (must have)
- [ ] Adopt Karpathy's "compiled knowledge artifact" framing for ACP without changing runtime contracts.
- [ ] Add a deterministic workspace health/lint surface over already published workspace artifacts.
- [ ] Add an explicit Ask-answer-to-proposal promotion path that keeps Q&A compounding reviewable and non-automatic.
- [ ] Harden citation/claim identity checks before introducing any new claim model.
- [ ] Defer claim ledger, contradiction policy and search projection until smaller slices prove the need and contracts are clear.
- [ ] Keep every slice reviewable, deterministic and covered by focused tests/fixtures.

### Non-goals
- [ ] Do not build a generic Obsidian-compatible personal wiki.
- [ ] Do not let runtime providers own or silently rewrite canonical workspace files.
- [ ] Do not auto-promote `qa.ask` answers into `reports/as-is/*`, `model/*` or `charter/*`.
- [ ] Do not add MCP memory server, hosted sharing, background autonomous rewrite, or vector DB as source of truth in MVP.
- [ ] Do not overload topology edges (`calls`, `reads`, `writes`, etc.) with general claim relations (`supports`, `contradicts`, `supersedes`).

### Approach
1) **Slice K1 - Product framing / terminology**
   - Update README, architecture overview and selected UI copy to describe `arch-workspace` as a validated compiled architecture knowledge base.
   - Keep this wording-only: no API/schema/runtime behavior changes.
   - Sync wording with the existing "not chat history" positioning.

2) **Slice K2 - Deterministic workspace health/lint**
   - Add a deterministic health scanner over current workspace artifacts, separate from `validator-verdict.json`.
   - Initial checks should use existing data only: broken artifact refs, old unresolved questions, findings without proposals, proposals without evidence refs, observation provenance gaps, duplicate entity alias candidates, orphan domain outputs and broken final/citation index links.
   - Publish a small report surface, for example `reports/health/workspace-health.json` plus `reports/health/workspace-health.md`, only if the artifact contract is documented and fixture-backed.
   - Surface the summary in `Readiness`/`Review` before adding any live-provider lint.

3) **Slice K3 - Ask answer promotion to proposal draft**
   - Add an explicit user action from a selected async QA answer to create a proposal draft package.
   - Preserve `qa.ask` read-only/canonical-non-mutating semantics.
   - Suggested output package:
     - `proposals/qa-synthesis-<run-id>-<slug>/proposal.md`
     - `proposals/qa-synthesis-<run-id>-<slug>/evidence.md`
     - `proposals/qa-synthesis-<run-id>-<slug>/source-qa-answer.json`
   - Ensure citations and unresolved assumptions are copied into the proposal package.
   - Route review through existing `Proposals`/`Publish` surfaces.

4) **Slice K4 - Citation/claim identity hardening**
   - Strengthen checks around existing `citation-index.json` and report claim IDs before adding new model directories.
   - Detect duplicate/unstable claim IDs, missing citation coverage for key report surfaces and citation refs that no longer resolve.
   - Prefer health-report warnings first; promote to validator/blocking behavior only with explicit policy.

5) **Slice K5 - Claim ledger prototype (post-MVP candidate)**
   - Only after K2/K4, consider `model/claims/*.yaml` for a narrow fact set.
   - Start with architecture facts that already map to existing model semantics: service uses datastore, service calls service/external, service exposes API, service publishes/subscribes topic.
   - Add schema/spec/fixtures/tests in the same slice; use `acp-schema-guardian`.

6) **Slice K6 - Contradiction review queue (post-claim-ledger)**
   - Add typed claim relations only after claims have stable IDs.
   - Treat contradiction as reviewable knowledge, not automatic cleanup.
   - Keep relation vocabulary small and policy-owned: `supports`, `contradicts`, `supersedes`.

7) **Slice K7 - Rebuildable search projection**
   - Add only if QA/context loading becomes a measured pain.
   - Keep search index disposable and rebuildable from canonical workspace files.
   - Do not make SQLite/FTS/vector output canonical or required external infrastructure.

### Minimal first deliverable
The first implementation slice should be K2, not K5/K6/K7. A deterministic health/lint report imports the most useful Karpathy maintenance loop while preserving ACP's current architecture and avoiding schema-heavy claim ontology work.

K1 can be bundled with K2 if the diff stays small; otherwise keep K1 as a docs/UX-only precursor.

### Files expected to change
K1:
- `README.md`
- `docs/ARCHITECTURE.md`
- selected UI copy in `ui/src/components/*` / `ui/src/lib/*` only if needed

K2:
- new or existing internal package for health scanning, for example `internal/workspacehealth/*`
- `internal/api/*` if exposing health through API
- `internal/orchestrator/*` only if health report is generated as part of run/promotion
- `ui/src/components/*`, `ui/src/hooks/*`, `ui/src/lib/*` for `Readiness`/`Review` summary
- `fixtures/scenarios/*` and/or dedicated health fixtures
- `docs/spec/PIPELINE_SPEC.md`, `docs/TESTING_STRATEGY.md`, `docs/APPENDIX_SCHEMAS.md` if a new report contract is introduced

K3:
- `internal/qa/*` and/or `internal/api/*`
- proposal package writer under orchestrator/API boundary
- `ui/src/components/*`, `ui/src/hooks/*`, `ui/src/lib/*`
- fixtures for QA answer promotion
- docs/spec/API and pipeline docs if a new endpoint is added

K4-K7:
- `schemas/*`, `docs/spec/*`, `fixtures/*`, `examples/*`, validators and ADR rationale as required by schema/contract policy

### Acceptance criteria
- [ ] K1 wording explains ACP as a compiled architecture knowledge base without weakening existing boundaries.
- [ ] K2 health checks are deterministic, fixture-backed and run without live providers/network.
- [ ] K2 does not replace step3 validator and does not block promotion unless a later policy says so.
- [ ] K3 creates proposal drafts only by explicit user action.
- [ ] K3 preserves canonical non-mutation of `qa.ask`.
- [ ] K4 detects claim/citation identity drift without introducing an unreviewed claim ontology.
- [ ] K5-K7 are not implemented without fresh schema/spec-first plans.
- [ ] Full DoD passes for each implementation slice: `make contracts`, `make test`, `make lint`, `make build`.

### Test plan
- K1:
  - `./scripts/run-go.sh test ./internal/docsync`
  - `git diff --check`
- K2:
  - unit tests for each health rule;
  - fixture workspace tests for clean/warn/error health reports;
  - API tests if health endpoint is added;
  - UI tests for summary rendering and empty/partial states;
  - docs-sync test for terminology and source-of-truth consistency.
- K3:
  - API tests for valid QA promotion, missing answer, stale run and path-safety;
  - proposal package golden/fixture tests;
  - UI tests for explicit promotion action and no automatic mutation;
  - regression asserting `qa.ask` still writes only run-scoped artifacts.
- K4-K7:
  - schema/contract tests, fixture/golden updates and `acp-schema-guardian` workflow when new contracts are introduced.

### Risks
- Health report can duplicate validator responsibilities if the scope is not explicit.
- Ask promotion can create proposal spam unless the UI makes it a deliberate review action.
- Claim ledger can become a second model if introduced before citation identity hardening.
- Search projection can become hidden source of truth unless it is rebuildable and disposable.
- Terminology changes can overpromise "memory" unless docs keep the staged/validated promotion boundary clear.

### Progress log
- 2026-06-23: Created adoption roadmap from Karpathy analysis. Selected deterministic workspace health/lint as the recommended first implementation slice, with Ask promotion and claim/citation hardening as follow-ups.

### Plan ID
EP-20260710-workspace-health-snapshot

### Context
This is the concrete K2a implementation slice from `EP-20260623-karpathy-adoption-roadmap`.
It imports the Karpathy maintenance loop as a deterministic, read-only health snapshot over
accepted workspace artifacts, without turning ACP into a free-form wiki and without adding
canonical `reports/health/*` artifacts in this slice.

### Goals (must have)
- [x] Add deterministic `internal/workspacehealth` scanner over existing workspace files.
- [x] Expose `GET /api/workspace/health` as a read-only API endpoint.
- [x] Surface health status/counts in `Readiness` and detailed items in the right inspector.
- [x] Keep health advisory: no run/review/publish/Q&A blocking and no runtime/provider calls.
- [x] Update README/ARCHITECTURE/STAKEHOLDER/API docs for the implemented K2a behavior.
- [ ] Complete full DoD for the implementation slice and archive/close the plan after owner review.

### Non-goals
- [x] Do not persist `reports/health/workspace-health.json|md`.
- [x] Do not change `qa.ask` or add Ask-to-Proposal promotion in this slice.
- [x] Do not add claim ledger, contradiction policy, vector DB or search projection.
- [x] Do not introduce schemas for health output until/unless a persisted artifact is added.

### Approach
1) Implement scanner rules for observation model provenance without evidence, orphan domain outputs, proposal review sections and unresolved open-question count.
2) Add API route and tests that confirm pass/warn responses and read-only behavior.
3) Add typed frontend client/state and reuse existing `Readiness`/right-inspector surfaces.
4) Update docs and run focused backend/frontend/docs checks before full DoD.

### Files expected to change
- `internal/workspacehealth/*`
- `internal/api/server.go`, `internal/api/server_test.go`
- `ui/src/App.tsx`, `ui/src/components/StagePanels.tsx`, `ui/src/lib/*`, `ui/src/App.test.tsx`
- `README.md`, `docs/ARCHITECTURE.md`, `docs/STAKEHOLDER_DOC.md`, `docs/spec/API_SPEC.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] Clean workspace returns `status=pass` with no findings.
- [x] Missing observation evidence, orphan domain output and proposal review-section gaps return warning items.
- [x] Open questions return info item only.
- [x] Malformed model YAML returns health `status=fail` as report content, not API lifecycle failure.
- [x] UI states cover no findings, warning items and scan failed.
- [x] Health scan is read-only and does not mutate workspace files.

### Risks
- Health can duplicate validator semantics; K2a keeps rules advisory and outside promotion gates.
- Proposal section check is intentionally simple heading-based lint; richer proposal quality belongs in a later slice.
- Persisted health artifacts would require a new contract/schema/docs fixture pass and are deferred to K2b.

### Progress log
- 2026-07-10: Implemented K2a read-only workspace health snapshot, API endpoint, Readiness/right-inspector UI surface and docs sync. Focused Go/UI checks and full Go suite passed; canonical `make contracts|test|lint|build` remains blocked on this host until Node.js 22.21.1 is available.
### Plan ID
EP-20260711-run-pinned-evidence-review

### Context
Epic 20 originally started with the highest-risk trust defect found in Console V2: historical
review could lose staged identity and expose current canonical bytes. Epic 19 subsequently landed
typed `staged_path`, selected-run identity/containment checks and run-keyed artifact loading. This
slice now begins with a sufficiency audit and closes only the remaining explicit source-mode,
fail-closed edge semantics, atomic view-state and rendered same-path multi-run proof gaps, without
changing the canonical artifact schema.

Re-baseline note (2026-07-15): Epic 19 delivered typed staged snapshot foundations in
`appContracts.ts` and `useRunArtifacts.ts`. Before implementation, audit current behavior against
the goals below and retain only missing explicit source-mode, fail-closed, transaction and rendered
multi-run proof work; do not duplicate landed code.

### Goals (must have)
- [ ] Preserve top-level `run_id`/`generated_at` and document `staged_path` in the frontend
      final-run index contract.
- [ ] Resolve every selected-run final document from its staged path, including coverage and
      open-question artifacts when present in that run's index.
- [ ] Require index `run_id` to match the selected run and every `staged_path` to remain inside
      `reports/taskruns/<run_id>/staging/final/`.
- [ ] Load preview, coverage and open questions as one run-id-keyed snapshot transaction with
      stale-response suppression or abort behavior.
- [ ] Model `Run snapshot` and `Current workspace` as explicit, non-interchangeable source modes.
- [ ] Default selected History/Review evidence to `Run snapshot` and expose
      `Current workspace` only as an explicit user action.
- [ ] Show selected `run_id`, generated timestamp and source mode in the Review/Publish evidence
      context; runtime/provider authority remains scoped to Epic `20C`.
- [ ] Treat a missing, unreadable or incomplete staged snapshot as an explicit selected-run
      error; never fall back to canonical bytes.
- [ ] Add deterministic two-run regression coverage and rendered proof with the same canonical
      paths but different content.

### Non-goals
- [ ] Do not change `schemas/final-run-index.schema.json` or synthesize a second snapshot format.
- [ ] Do not redesign the full navigation, artifact viewer or Publish gate in this slice.
- [ ] Do not add persisted review approvals or exact file-scoped Git commit behavior.
- [ ] Do not change runtime promotion, canonical workspace writes or live E2E matrices.
- [ ] Do not require a live provider; required acceptance uses fake/recorded deterministic data.

### Approach
1) Extend the TypeScript final-index document contract with required `staged_path` and keep
   its top-level `run_id`/`generated_at` parsing aligned with the existing JSON schema.
2) Introduce a small evidence-source view model that distinguishes a selected run snapshot from
   the current canonical workspace.
3) Validate that the index belongs to the selected run and that normalized staged paths are
   contained under that run's `staging/final` root before issuing artifact reads.
4) Build one run-id-keyed snapshot load for preview, coverage and open questions; abort or ignore
   stale responses when selection changes. Do not retain stable-path fallback in snapshot mode.
5) Distinguish an optional document absent from the index (`Not produced for this run`) from an
   indexed document that cannot be read (`Snapshot unavailable`).
6) Add primary-context copy and explicit unavailable/corrupt snapshot recovery near the
   selected evidence rather than as a remote global error.
7) Add two run fixtures sharing canonical paths while staging different bytes; assert rapid
   run A -> run B -> run A switching never leaks canonical, cross-run or stale async content.
8) Run focused UI tests, typecheck/build, rendered multi-run QA and the repository Full DoD.
9) Update behavior documentation only where run-history/current-workspace semantics are
   described, then commit this slice independently before selecting `20B`.

### Files expected to change
- `ui/src/lib/appContracts.ts`
- `ui/src/hooks/useRunArtifacts.ts`
- `ui/src/hooks/useRunActions.ts`
- `ui/src/hooks/useRunExplorer.ts`
- `ui/src/App.tsx` and/or the touched Review/Publish context component
- `ui/src/App.test.tsx`
- deterministic `historical-run-snapshot` UI/Playwright scenario for same-path multi-run evidence
- `docs/ARCHITECTURE.md`, `docs/UI_CONSOLE_V2_DESIGN.md` and testing docs only where the
  implemented source-mode behavior needs synchronization

### Acceptance criteria
- [ ] Run A and Run B use the same canonical overview/coverage/questions paths but distinct
      staged content; selecting either run shows only its own bytes and metadata.
- [ ] A final index whose `run_id` differs from the selected run is rejected, and normalized
      staged paths outside `reports/taskruns/<selected-run>/staging/final/` are never read.
- [ ] Rapid A -> B -> A selection cannot let a late response from B overwrite the final A
      snapshot.
- [ ] Changing the current canonical file after both runs does not alter either historical
      preview.
- [ ] Missing staged content renders `Snapshot unavailable` (or equivalent) with run/path
      context and does not issue a canonical-path read.
- [ ] An optional document absent from a run index renders `Not produced for this run` and is
      not treated as corrupt snapshot data.
- [ ] Review and Publish evidence context display run id, generated time and `Run snapshot`;
      explicit current-workspace mode displays `Current workspace`.
- [ ] Focused component tests, UI typecheck/build and rendered multi-run QA pass without console
      errors or document-level horizontal overflow.
- [ ] Full DoD passes: `make contracts`, `make test`, `make lint`, `make build`.
- [ ] Documentation and fixtures remain aligned with the unchanged final-run index schema.

### Risks
- Existing tests with different canonical paths per run can pass while hiding the bug; same-path,
  different-byte fixtures are mandatory.
- Some run indexes may not contain optional coverage/question documents. Snapshot mode must show
  `Not produced for this run` rather than borrowing current workspace files.
- A staged path can be syntactically present but point at another run or outside staging; path
  normalization and selected-run containment are part of the trust boundary.
- Independent async requests can resolve out of order during run switching; the selected run id
  must key the complete snapshot transaction, not just the primary preview.
- A missing staged file can be tempting to recover through canonical fallback; that would
  recreate the trust defect and is explicitly forbidden.
- Publish currently mixes run-scoped preview with workspace-scoped Git state. This slice labels
  those contexts but leaves full publication-boundary correction to `20B`.

### Progress log
- 2026-07-11: Selected `20A` as the first P0 slice after the full UX/UI trust and craft audit;
  recorded the dependency-ordered Epic 20 program in `docs/BACKLOG.md`. Implementation has not
  started.
- 2026-07-15: Epic 19 merged into `main` at `02716bb`; snapshot foundations now require a
  sufficiency audit before this implementation plan is activated.

### Plan ID
EP-20260712-evidence-backed-architecture-refresh

### Context
ProvenArch already creates validated, Git-versioned architecture artifacts in a separate workspace,
but its primary overview is not yet an intentional navigation home and refresh does not first explain
what source changed or why particular domains/artifacts must be regenerated. This Wave 1 plan adds an
evidence-backed architecture home followed by deterministic source revision and impact planning,
explainable no-op behavior, affected-only collect execution and finally safe surgical promotion.

The implementation must retain the current trust boundaries: source repos remain read-only; current
source evidence outranks Git messages; runtime outputs remain staged and validator-gated; unknown
impact causes conservative refresh; required CI remains deterministic and provider-independent.

### Goals (must have)
- [ ] Make `reports/as-is/overview.md` the architecture workspace home without changing its path.
- [ ] Add machine-checkable documentation-quality rules to `step2.asis_docs` and aligned guidance to `qa.ask`.
- [ ] Persist an exact per-repo source revision baseline for successful promoted architecture runs.
- [ ] Compute a deterministic refresh impact plan before provider collect execution.
- [ ] Add bounded recent Git intent evidence without treating commit messages as source truth.
- [ ] Support safe, explainable no-op refreshes.
- [ ] Dispatch collect only for affected shards/domains with conservative fallback for ambiguity.
- [ ] Preserve unaffected canonical artifacts byte-for-byte after dependency-aware promotion exists.
- [ ] Expose changed/preserved/uncertain decisions in existing operator surfaces.

### Non-goals
- [ ] Do not write into analyzed source repositories.
- [ ] Do not read the full Git history or let an LLM choose refresh scope without deterministic planning.
- [ ] Do not add hosted scheduling, external docs integrations, providers or security enforcement.
- [ ] Do not infer human acceptance from workspace Git history; use successful validator promotion until a separate acceptance contract exists.
- [ ] Do not call an unmapped in-scope change a no-op.
- [ ] Do not rename ProvenArch or replace `arch-workspace` with an OpenWiki-compatible layout.

### Approach
1) Deliver `21A` as the minimal useful slice: architecture home policy, docs/QA quality guidance, fake output and focused tests, with no schema changes.
2) Deliver `21B` as a contract slice for source revision baselines; use `acp-schema-guardian` and `acp-test-fixtures` and synchronize all required schema/spec/example/fixture/ADR surfaces.
3) Deliver `21C` as a pure deterministic planner over complete changed paths, prior shard/domain scopes and evidence dependencies; ambiguous input produces an explicit full-refresh plan.
4) Deliver `21D` only after planner fixtures prove no-op safety; no-op skips providers and canonical rewrites but persists taskrun rationale.
5) Deliver `21E` by filtering existing collect dispatch with the validated impact plan and passing only bounded affected-scope Git intent evidence to providers.
6) Deliver `21F` by merging freshly generated affected artifacts with explicitly selected known-good baseline artifacts, then validating the complete staged publication set before promotion.
7) Deliver `21G` after behavior exists: add operator-facing explanations, update product language and perform deterministic plus optional trusted-machine quality validation.

### Files expected to change
- `docs/BACKLOG.md`
- `docs/PLANS.md`
- `docs/STAKEHOLDER_DOC.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/spec/WORKSPACE_SPEC.md` only if persisted workspace configuration changes
- `docs/APPENDIX_SCHEMAS.md`, `schemas/*`, `examples/*`, `fixtures/*` for `21B/21C` contracts
- `internal/workspace/*` or a focused internal source-revision package
- `internal/orchestrator/*`
- `internal/runtime/steppolicy/*`
- `internal/runtimedrafts/*`
- `internal/runtime/fakeruntime/*`
- `internal/qa/*`
- `internal/artifactquality/*`
- focused Go/UI/scenario fixture tests for each slice

### Acceptance criteria
- [ ] Every implementation slice has focused fixtures/tests and passes Full DoD: `make contracts`, `make test`, `make lint`, `make build`.
- [ ] Contract slices synchronize schemas, specs, appendix, examples, fixtures, validators, tests and ADR rationale.
- [ ] No-op is possible only for unchanged clean revisions or out-of-scope-only changes.
- [ ] Dirty worktree, missing baseline, history rewrite, oversized range and unmapped in-scope changes are explicit conservative fallbacks.
- [ ] Planner input is never silently truncated; only provider history context is bounded.
- [ ] Commit messages remain secondary intent evidence and cannot override current source evidence.
- [ ] Selective promotion preserves unaffected canonical files byte-for-byte and validates one coherent final/citation index set.
- [ ] Required CI has no live provider or network dependency.

### Risks
- Git revision state and workspace artifact state can diverge; baseline selection must bind both to one successful promoted run.
- Path-to-domain mapping can be incomplete; correctness requires conservative fallback instead of optimistic skipping.
- Carrying forward artifacts can preserve stale evidence unless dependency and baseline provenance are explicit and validator-checked.
- Global overview/coverage/findings may depend on multiple domains, so surgical refresh must model aggregate dependencies rather than only copy files by path.
- Adding impact artifacts creates contract maintenance cost; schemas and examples must be designed once the required fields are proven by the planner slice.

### Progress log
- 2026-07-12: Created Wave 1 Epic 21 and decision-complete implementation sequence. Selected `21A` as the minimal first slice; no runtime or schema behavior changed.

---

## EP-20260715-21A-architecture-home

### Goal
- Make `reports/as-is/overview.md` the evidence-backed navigation home and align fake/runtime/QA behavior without changing public schemas or routes.

### Implementation
- [x] Add the required non-empty architecture-home sections to step2 policy and machine validation.
- [x] Update deterministic fake output and QA context ranking while preserving relevance-first evidence selection.
- [x] Add focused prompt, draft-validation, fake-runtime and QA regressions.
- [x] Synchronize pipeline, architecture and stakeholder documentation.

### Acceptance
- Complete architecture-home output passes; missing sections, runtime narration and evidence-free output fail before promotion.

---

## EP-20260715-21B-source-revisions

### Goal
- Persist a deterministic per-run source revision baseline without changing `workspace.yaml` or pipeline execution semantics.

### Implementation
- [x] Add and register `source-revisions.schema.json` plus parser, example and fixtures.
- [x] Capture current revisions, worktree state, effective scopes and analysis-input fingerprint before execution.
- [x] Select only a prior succeeded validator-promoted run with a valid matching source revision artifact as baseline.
- [x] Register the taskrun artifact and cover conservative fallback states.

### Acceptance
- Revision/fingerprint output is deterministic, contains no absolute checkout paths and never blocks the existing full pipeline merely because selective planning is unsafe.

---

## EP-20260715-21C-refresh-impact-plan

### Goal
- Persist a deterministic advisory refresh impact decision before collect while retaining full refresh execution.

### Implementation
- [x] Add and register `refresh-impact-plan.schema.json` plus parser, example and fixtures.
- [x] Compute complete Git deltas with rename/copy identity and a fail-closed 10,000-path planning limit.
- [x] Map in-scope changes through prior shards, domains, citations and final-index dependencies.
- [x] Persist advisory decision/candidates and keep provider dispatch unchanged.

### Progress log
- 2026-07-15: Implemented 21A–21C. Revision and impact artifacts are advisory inputs only; refresh still executes the complete pipeline. Model provenance remains represented through final-index source-shard dependencies until 21F defines a persisted selective-promotion dependency contract.

### Acceptance
- Unchanged/out-of-scope inputs can be identified as candidates; every dirty, ambiguous, incomplete or unmapped in-scope input produces `full_refresh_required`.

### Plan ID
EP-20260629-live-e2e-artifact-summary-finalization

### Context
The latest accepted `claude-code` strict medium diagnostic proved the new execution gate shape (`execution_report_*`, selected-provider totals, no legacy mixed quality gate artifacts), but manual artifact review rejected FTGO artifact quality. The machine execution verdict was correctly separate from manual artifact quality, yet key FTGO markdown could still claim `Shard pack manifests: none observed` or stale final/citation index availability while current-run shard summaries and downstream indexes existed. The same run also showed that `step4.proposals` could pass with formally present but weak proposal/changelog markdown: empty findings/gaps sections, dangling references to findings "above", and generic follow-up actions. This slice closes those machine-checkable contradictions before opening the PR, while leaving broader truthfulness/readability to the SWE artifact-quality report.

### Goals (must have)
- [ ] Reject step2 as-is markdown that claims shard manifests/evidence are absent when current-run typed shard summary has planned shards.
- [ ] Reject stale `final-run-index` / `citation-index` availability claims, including `not yet present` variants, instead of promoting them as operator evidence.
- [ ] Require step2 overview to include concrete repo/path, citation, or staged artifact references when typed shard status is visible.
- [ ] Require step2 architect summary to include a decision-ready operator cue when typed shard status is visible.
- [ ] Require step4 proposal/changelog drafts to have non-empty required sections, current-run evidence refs and exact shard completeness when typed status is visible.
- [ ] Reject dangling proposal wording such as `findings above` and generic follow-up plans unless the proposed plan contains an explicit no-actionable-proposal gap tied to current-run evidence.
- [ ] Route these validation failures to the existing provider-authored shard-status/downstream-index enrichment retries; repeated noop/scaffold/stale output remains `runtime_contract_failed`.
- [ ] Keep `release_verdict_*` execution-only; artifact/UX acceptance remains separate SWE reports.
- [ ] Treat current `codex-code` quota/permission/readiness as an external operational blocker, not a repo-code blocker for this PR.

### Non-goals
- [ ] Do not change product UI/API behavior.
- [ ] Do not edit canonical matrix files or curated repo files.
- [ ] Do not add deterministic synthesis as a hidden success path.
- [ ] Do not commit generated live E2E evidence unless it is an intentional fixture/golden.

### Approach
1) Tighten shared runtime draft validation for FTGO-style stale/contradictory as-is markdown.
2) Update prompt contract tests and runtime draft tests for the observed failure shape.
3) Synchronize live E2E runbook, pipeline spec, architecture and testing strategy.
4) Run targeted tests, then full DoD.
5) Commit, run Claude strict medium diagnostic from a clean tree if host/provider preflight passes, push branch, open draft PR, and squash-merge only after CI is green.

### Files expected to change
- `internal/runtimedrafts/manifest.go`
- `internal/runtimedrafts/manifest_test.go`
- `internal/runtime/promptcontract/draft_repair.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `docs/PLANS.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`

### Acceptance criteria
- [ ] Targeted `go test ./internal/runtimedrafts ./internal/runtime/promptcontract` passes.
- [ ] Full DoD passes with exact Node toolchain.
- [ ] Claude strict medium diagnostic either passes with accepted manual artifact/UX assessment or records a concrete remaining blocker.
- [ ] PR body documents Codex readiness as external operational blocker if it is still not runnable.

### Risks
- Stricter markdown validation can convert previously passing provider output into `runtime_contract_failed`; this is intentional for machine-checkable contradictions, while subjective truthfulness remains manual artifact quality.
- Live rerun can still fail on host/provider quota, auth, network or timeout; classify those separately and do not bypass with test-only flags.

### Progress log
- 2026-06-29: Started finalization slice after manual review rejected the latest Claude FTGO artifacts despite clean machine execution separation.
- 2026-06-29: Added the observed weak FTGO `step4` proposal/changelog shape to the plan: proposal sections must be non-empty, evidence-backed and actionable, or explicitly state a no-actionable-proposal gap.
### Plan ID
EP-20260616-live-e2e-recovery-rerun-loop

### Context
Diagnostic `claude-code` medium (`regres long`) live runs exposed harness/runtime issues before a clean strict-medium acceptance loop can continue: `draft_artifact_enrichment` for `step2.asis_docs` did not rewrite all bootstrap draft files, batch reporting let a stale `runner_unavailable` classifier row override collect partial root cause, and the `f2e962f` rerun showed fully silent collect fresh retry exhaustion falling straight to `runner_unavailable` without a focused collect-pair repair attempt. The user requested fix -> DoD -> commit -> live rerun loop across `claude-code`/`codex-code` with target rotation, without product UI/API changes or canonical matrix/repo edits.

### Goals (must have)
- [ ] Harden step2 draft enrichment prompt/contract so enrichment reads bounded staged evidence and write-first overwrites `overview.md`, `summary.md`, and `architect-summary.md`.
- [ ] Keep bootstrap/noop enrichment as `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`; do not add deterministic synthesis as a hidden success path.
- [ ] Run one provider-authored collect-pair repair after exhausted fully silent collect retry before terminal `runner_unavailable`, while keeping invalid/noop repair terminal.
- [ ] Make collect-pair repair evidence-first without seed/fallback heredoc, and stop invalid observed artifacts by no-fresh-mutation window even when provider stdout remains active.
- [ ] Give collect-pair repair a fresh-mutation threshold, minimum 5-minute pre/post/partial artifact windows, and validation-specific prompt focus for directory-only evidence refs, process-contaminated markdown, and no-artifact stalls.
- [ ] Fix report classification so collect partial failures stay visible as `runtime_flow_failed`/`partial_failure_count`, while primary `failure_class` uses the concrete terminal/provider class when available.
- [ ] Ensure live UI dependency/browser precheck cannot hang indefinitely; timeout must become `precheck_failed` with log/report evidence before headless runtime starts.
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
- [ ] Script tests prove partial collect stays visible as a runtime-flow signal while provider classifier rows can set primary `runner_unavailable` / `runtime_contract_failed`.
- [ ] Script tests prove a hung UI precheck command is terminated and reported as `precheck_failed`.
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
- 2026-06-19: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T190444Z` exposed that `posthog-ee` could still become a checkpointed shard after structural-invalid repo evidence paths (`ee/celery.py`, `ee/billing/__init__.py`, `ee/plugin-server/...`) plus exhausted `collect_manifest_repair`, because deterministic `collect_manifest_runtime_recovery` rebuilt a README-only manifest and marked the shard `succeeded`. Current slice removes deterministic post-repair fallback for structural-invalid manifests: missing/scaffold-only manifests may still use runtime recovery before provider repair, but exhausted structural manifest repair now remains terminal `runtime_contract_failed`.
- 2026-06-22: Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260622T045732Z` confirmed the step0 draft enrichment blocker was gone and selected-provider reporting/no-legacy execution reports remained correct, but both profiles failed during collect partials: PostHog primary classifier `runner_unavailable`, FTGO primary classifier `runtime_contract_failed`, with `runtime_flow_failed`/`partial_failure_count` as execution signals. Current slice persists per-shard `error_code` into shard summary JSON and aligns Python reports/docs so primary `failure_class` no longer collapses these cases back to generic `runtime_flow_failed`.
- 2026-06-22: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260622T081027Z` passed PostHog backend/frontend and completed FTGO collect 16/16 with no shard errors, but failed FTGO `init.step2.asis_docs` as `runtime_contract_failed`: `draft_artifact_enrichment` generated a filesystem command using missing `python`, so no markdown target was rewritten and strict validation rejected bootstrap-only `overview.md`/`summary.md`/`architect-summary.md`. Current slice adds a prompt-level `python3` requirement plus one provider-authored `draft_artifact_enrichment_python3_retry` for this missing-interpreter case, without deterministic draft synthesis or validation weakening.
- 2026-06-22: After commit `bebbd87`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260622T133647Z` preserved selected-provider totals, emitted `execution_report_*`, skipped frontend only because backend snapshots were missing, and kept artifact quality outside the machine verdict. It still failed both selected slots as `runtime_contract_failed`: PostHog `step4.proposals` rewrote proposal/changelog but claimed `final-run-index.json` had 0 observed documents because the provider counted a nonexistent `documents[]` field instead of current-run `canonical_documents[]`; FTGO refresh `step2.asis_docs` no-action retry printed a `python3 - <<'PY'` command as assistant text and left bootstrap markdown unchanged. Current slice adds explicit final/citation index schema instructions, rejects short `Draft surface initialized` scaffold markers, and adds one provider-authored `draft_artifact_enrichment_command_text_retry` for printed-command no-action output, without deterministic draft synthesis or validation weakening.
- 2026-06-22: Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260622T160036Z` ran from clean commit `9f921a6`. PostHog failed in `init.step0.constitution`: first enrichment and no-action retry made no fresh mutation, leaving bootstrap `charter-overview.md`; FTGO passed step0 enrichment but then hit repeated fully silent collect shard failures, with 1/16 collect shards succeeded and five consecutive `runner_unavailable` failures before the run was operator-interrupted to avoid burning the full medium window. Current slice keeps provider-authored-only success, extends step0 enrichment to a bounded 2-minute pre-artifact window with explicit repo-entrypoint targets in the compact retry, and makes sequential best-effort collect abort after five consecutive `runner_unavailable` failures while marking undispatched shards failed with the same provider class.
- 2026-06-22: Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260622T173826Z` ran from clean commit `e4208fb` and confirmed the new execution/artifact split still held (`execution_report_*`, no legacy quality gate artifacts, artifact quality not driving machine verdict), but both profiles failed in `init.step0.constitution` as `runtime_contract_failed`. PostHog successfully rewrote `charter-overview.md` with evidence-backed content after the compact retry, then drifted `constitution-draft.json` by adding unknown `status`, `content_digest`, and `enriched_at` fields. FTGO kept `charter-overview.md` bootstrap-only because its `git_url` resolved checkout existed only in `read_context_roots` under `.acp/repos`, while step0 enrichment include dirs added repo roots only from `workspace.yaml` path/validate fallback. Current slice adds one provider-authored `draft_artifact_enrichment_manifest_shape` retry without relaxing strict parsing, explicitly bans unknown manifest fields in enrichment prompts, and lets step0 enrichment include selected `git_url` repo roots from `read_context_roots` while keeping `step2/step4` bounded.
- 2026-06-18: Committed first loop slice `5587298`; diagnostic `codex-code` strict medium run `regres-long-posthog-ftgo-20260618T065744Z` reached PostHog collect and exposed a new collect-repair blocker. The first shard wrote evidence-backed `root-overview.md` during `collect_pair_repair` but then read `.test_durations` and stalled before `shard-pack-manifest.json`; the second shard exhausted pair repair with no authored output. The run was interrupted after two independent P0 collect failures to avoid spending the remaining medium window on repeated shard failures. Current slice narrows pair-repair evidence candidates, forbids extra repo reads after markdown write, and lets markdown-only pair repair partials chain into explicit manifest runtime recovery while no-artifact pair repair remains terminal.
- 2026-06-18: Committed second loop slice `6e84a9f`; diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T074654Z` confirmed the markdown-only manifest recovery path, but exposed a no-artifact collect-pair repair failure on the first PostHog shard. Raw stdout showed Codex created its own broad evidence-read command, read many root files plus `docker-compose.dev.yml`, and stalled before writing either `root-overview.md` or `shard-pack-manifest.json`. The run was interrupted as `infra_signal_terminated` after the P0 blocker was captured. The later exact read-only `collect_pair_repair_preflight` attempt also stalled preflight-only, so the current slice makes collect-pair repair one-shot write-first instead of two-phase preflight/write.
- 2026-06-18: After commit `28b5937`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T142635Z` proved the one-shot write-first prompt made Codex attempt a provider-authored filesystem command, but the provider added a brittle exact-phrase precondition (`required = [...]`, `missing expected evidence`) and aborted before writing any authored artifacts when current PostHog text did not match guessed snippets. Current slice keeps provider-authored recovery but forbids hard-coded expected-phrase gates before writes; repair may only precheck target containment, at least one allowed evidence file read, and size limits, while unsupported planned claims must be omitted or reported as coverage gaps.
- 2026-06-18: After commit `df771e5`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T145933Z` showed the root-file collect pair repair now writes valid artifacts, but the next directory shard (`posthog-bin`) failed before writing because Codex generated an over-complex Python writer with an invalid f-string (`f-string: empty expression not allowed`). Current slice keeps provider-authored recovery but tightens the first repair command to a mechanically simple writer: no Python f-strings, `.format(...)` template writers, generated Python source strings, nested quote tricks, or self-invented semantic pre-write aborts; backend validation remains the semantic gate.
- 2026-06-18: After commit `2edf6cf`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T153313Z` confirmed the f-string failure was removed and the root-file shard still recovered, but `posthog-bin` exhausted collect pair repair with no artifacts because Codex added a file-size hard gate (`read file exceeds size limit`) after reading a candidate larger than the per-file budget. The run was interrupted once this acceptance blocker was captured. Current slice keeps the evidence-bounded contract but changes the repair instruction to read bounded prefixes or skip oversized candidates and continue; terminal no-write failure is allowed only when no allowed evidence yields bytes.
- 2026-06-19: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T074026Z` passed PostHog machine execution plus frontend smoke, but failed FTGO refresh `step2.asis_docs` as `runtime_contract_failed`: `draft_artifact_enrichment` rewrote the three markdown drafts, then rewrote `asis-draft-manifest.json.outputs[]` with an invalid `logical_path` alias. The same run still showed repeated collect manifest repair no-write/status-only behavior that runtime recovered as explicit telemetry. Current slice keeps strict parsing, forbids output aliases in draft contracts, tells enrichment to preserve existing `outputs[]` mappings, and changes manifest-only collect repair from read-only preflight to a write-first provider-authored command contract.
- 2026-06-19: After commit `9f2820d`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T172527Z` confirmed bootstrap-only draft validation now routes straight to `draft_artifact_enrichment`, but PostHog refresh accepted `posthog-bin/bin-overview.md` as a completed collect shard even though the markdown said the first bounded evidence read was interrupted, the artifact was initial-only, and it "will be repaired with concrete file-level evidence". The run was interrupted during frontend-triggered PostHog collect after this acceptance blocker was captured. Current slice makes interrupted temporary collect markdown contract-invalid regardless of semantic richness, adds live-pattern regression tests, and updates collect prompt/runbook wording so these artifacts fail before downstream `step2` can mask them.
- 2026-06-20: After commit `25bb2d1`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T064659Z` did not reach headless runtime: `npm ci --prefix ui` in UI precheck hung with no log progress while registry was reachable, requiring manual interrupt. Current slice bounds UI dependency/browser precheck via `ACP_LIVE_PRECHECK_UI_TIMEOUT_SEC`, materializes timeout as `precheck_failed`, and adds script/docs coverage so precheck hangs cannot silently block the live loop.
- 2026-06-20: After commit `b03fe11`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T071243Z` reached machine `PASS` for both `posthog` and `ftgo` with selected provider totals only and no legacy mixed quality artifacts. Manual artifact review rejected the evidence: FTGO `draft_artifact_enrichment` reported planned/failed shard counts as unknown despite current-run collect summaries showing 16/16 succeeded, and several FTGO markdown drafts mentioned PostHog/matrix context as if it were target evidence. Current slice exposes current-run typed shard-plan/summary JSON to focused draft enrichment, instructs providers to count shard-summary `items[].status`, makes target identity come from `repo_scope`/`repo_scopes`/`domain_id` rather than matrix folder names, and keeps live/manual release evidence names out of core AOR packages.
- 2026-06-20: After commit `438b1aa`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T114135Z` reached machine `PASS`, fixed current-run shard completeness and target contamination, and preserved selected-provider execution reporting. Manual artifact review still rejected the evidence because several operator-facing draft files described the enrichment as replacing placeholder content. Current slice makes generic placeholder-replacement narration a shared runtime draft validation failure and strengthens prompt/docs contracts so this remains `draft_artifact_enrichment_noop_or_scaffold`, not an accepted artifact-quality issue.
- 2026-06-20: After commit `703cb66`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T160443Z` reached non-release machine `PASS` for selected provider only (`backend 2/2`, frontend PASS for both profiles, no legacy mixed quality-gate artifacts). Manual artifact review rejected the result because current-run final markdown still contained generic failed/incomplete shard caveats despite `16/16` succeeded summaries, and refresh as-is markdown cited prior taskrun final/citation indexes as if they were current-run evidence. Current slice rejects foreign live `run_*` references and generic conditional shard-gap wording in shared draft validation and prompt/docs contracts.
- 2026-06-21: Committed `6c91550` to isolate `codex-code` preflight readiness with the same auth-only `CODEX_HOME`, disabled plugin/app suggestion surfaces and `--ignore-user-config`/`--ignore-rules` used by runtime. Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260621T174452Z` reached real headless execution for both PostHog and FTGO, preserved selected-provider totals, and failed both profiles as `runtime_contract_failed` because provider-authored `draft_artifact_enrichment` produced evidence-backed markdown with malformed inline-code/code-fence syntax. Current slice adds a one-shot provider-authored `draft_artifact_enrichment_markdown_syntax` retry and prompt/docs tests without deterministic artifact synthesis or validation weakening.
- 2026-06-22: Claude strict medium rerun `regres-long-posthog-ftgo-20260622T183346Z` ran from clean commit `694278f` with selected provider `claude-code` only. PostHog failed at `init.step0.constitution` because no-action enrichment rewrote `charter-overview.md` with runtime/provider/generated/final-index narration instead of pure repo-entrypoint constitution evidence; FTGO proved `git_url` repo roots and collect recovery by completing 16/16 collect shards, then failed at `init.step4.proposals` because focused enrichment/no-action retry did not mutate bootstrap `proposal.md`/`changelog.md` before the 90s pre-artifact wall-clock. Current slice separates step0 prompt/validation from downstream evidence, rejects step0 downstream/runtime-only leaks, makes step4 no-action retry bounded write-before-analysis, and raises focused draft enrichment to a minimum 3-minute pre-artifact window.
- 2026-06-22: Claude strict medium rerun `regres-long-posthog-ftgo-20260622T202736Z` from clean commit `86251c2` validated the step0/FTGO read-root fixes but failed both profiles in `init.step1.collect`: PostHog had directory-only repo evidence refs (`livestream`) and process-contaminated markdown (`posthog-share-staticfiles`), while FTGO reproduced process-contaminated markdown and one fully silent no-artifact collect repair path. Reports correctly preserved selected-provider totals and execution/artifact-quality separation, but collect-pair repair used the generic 90s focused repair window and did not give validation-specific immediate rewrite guidance. Current slice gives collect-pair repair a fresh-mutation threshold, minimum 3-minute pre/post/partial windows, and prompt focus for directory-only citations, process-contaminated markdown and no-artifact stalls.
- 2026-06-23: Claude strict medium rerun `regres-long-posthog-ftgo-20260622T224209Z` ran from clean commit `0a2cc01` with selected provider totals only and no legacy mixed quality-gate artifacts. PostHog completed collect but failed `refresh.step2.asis_docs` because evidence-backed enrichment still claimed current-run final/citation indexes were unavailable; FTGO completed collect best-effort only partially (`9/16` checkpointed, `7` collect-pair repair failures) after repeated no-artifact/directory/process-contamination recovery stalls. Current slice adds one provider-authored `draft_artifact_enrichment_downstream_index_retry` for stale downstream-index claims and switches live-observed no-artifact/directory/process collect pair repair to a compact field-contract prompt without deterministic artifact synthesis.
- 2026-06-23: Codex strict medium rerun `regres-long-posthog-ftgo-20260623T024519Z` from commit `70fbadc` reached non-release machine `PASS` for selected provider only (`2/2` backend, both frontend smoke runs passed, no legacy mixed quality artifacts). Manual artifact review still found proposal-quality defects: `proposal.md` self-reported stale non-zero final-index document counts (`77/55`) while current `final-run-index.json.canonical_documents[]` had `79/57`, and "high-signal" evidence bullets included metadata-only JSON lines such as `"version": 1`. Current slice rejects metadata-only evidence bullets and mismatched current-run final/citation index counts in strict runtime draft validation, and strengthens step4 enrichment prompts/docs to compute counts from parsed JSON variables or omit them.
- 2026-06-23: Claude strict medium rerun `regres-long-posthog-ftgo-20260623T135008Z` from clean commit `1475969` proved the marker/window fixes on PostHog backend (`init` and `refresh` succeeded with 16/16 collect shards), but frontend auto triggered a UI-side live run that failed at `init.step2.asis_docs`: focused enrichment left bootstrap markdown unchanged, then runtime incorrectly scheduled a generic `draft_artifact_enrichment_no_action_retry` before eventually failing as `draft_artifact_enrichment_noop_or_scaffold`. Current slice removes the generic no-action enrichment retry entirely; noop/scaffold enrichment now exhausts immediately as `runtime_contract_failed`, while the narrow `draft_artifact_enrichment_command_text_retry` remains only for providers that print shell/Python command text instead of mutating files.
- 2026-06-25: The latest interrupted `codex-code` strict medium diagnostic exposed an outer harness reliability defect: `pipeline_timeout_sec=14400` was not a hard boundary and refresh reached about 32700 seconds before manual interruption. Current slice adds a process-group `pipeline-watchdog.py`, timeout terminal status as `runtime_timeout`, deadline/last-progress/clock-gap reporting, and provider lifecycle progress fields without increasing canonical medium timeouts or changing product UI/API behavior.
- 2026-06-25: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260625T184127Z` confirmed the new watchdog/liveness path on PostHog backend, but frontend failed because the canonical PostHog path checkout became an unreadable Git repo (`workspace.repo.ref.invalid` for pinned SHA). Taskrun quality already recorded `runtime_write_audit_unexpected_mutation`, but runtime accepted it as warning-only. Current slice makes post-provider protected workspace/analyzed repo mutation a terminal `runtime_contract_failed` for otherwise-successful steps and makes the live backend cycle rewrite canonical `path` inputs to run-local isolated detached clones before `init-workspace`, keeping canonical `/tmp/provenarch-live-e2e/...` as a verified source prerequisite only.
- 2026-06-25: After commit `bca7093`, `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260625T205043Z` proved path-repo isolation (`target-repos.live-isolated.yaml` and clean canonical PostHog), but PostHog failed before headless runtime because baseline fake API init hit the fixed 120s `api_init_timeout_sec` while actively progressing through `init.step4.proposals`. Current slice adds bounded progress-aware grace to the API init poll loop without increasing canonical timeout presets.
- 2026-06-25: After commit `2b6c7a9`, `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260625T211921Z` proved API init progress grace but failed during fake `init` before headless runtime: `runtime_write_audit_unexpected_mutation` reported `repo status unavailable after runtime` for the run-local read-only PostHog clone. The failure was a write-audit false positive caused by using writable/full `git status` against an isolated read-only large checkout. Current slice keeps read-only path-repo isolation but makes audit use a lightweight HEAD/index/mode snapshot with `GIT_OPTIONAL_LOCKS=0` for read-only clones, while preserving terminal mutation failures for protected workspace/writable repo changes.
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
- 2026-07-07: PR #120 merged into `main` at `7f65499` after green PR checks. Started `v0.1.8`
  beta release metadata branch for artifact-quality acceptance hardening, live E2E Excellent
  diagnostics and the recorded `smoke-tiny-bank-20260707T053308Z` Codex diagnostic evidence. The
  diagnostic reached strict `PASS` and artifact-quality `passed`, but stayed `Needs review` because
  `init.step2.asis_docs` and `init.step4.proposals` still required focused repair. The release can be
  prepared as beta/RC evidence, but must not claim canonical `RELEASE READY` without a fresh
  `release_verdict_*.json` plus accepted SWE UX/artifact-quality assessments.
- 2026-07-09: PR #123 squash-merged into `main` at `90c2931` after green PR checks. Started
  `v0.1.9` beta release metadata branch for Console V2 UX recovery polish, rendered recovery-state
  QA coverage and recorded medium `regres long` diagnostic evidence. The latest medium diagnostic
  remains a non-release `FAIL` because of provider/runtime reliability blockers (`qwen` headless
  readiness timeout and FTGO collect artifact handoff stalls), so this release must not claim
  canonical `RELEASE READY` without a fresh verified release verdict plus accepted SWE assessments.
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
- [x] Preserve execution/UX/artifact quality separation; `artifact_quality.*` is a public black-box artifact gate, separate from runtime contract status
- [x] Fix live-observed draft enrichment no-op/scaffold failure without ACP-side deterministic product synthesis
- [x] Fix frontend snapshot eligibility so failed headless refresh rows cannot start heavy UI smoke as if they were valid snapshots
- [x] Fix live-observed collect pair repair preflight-only/status-only no-op so post-preflight provider output must mutate markdown + manifest before prose
- [x] Replace two-phase collect pair repair preflight/write contract with one-shot provider-authored write-first command
- [x] Tighten normal collect so its first bounded work unit reads limited evidence, writes doc + manifest directly, and does not rely on repair/recovery as the normal success path
- [x] Isolate default `codex-code` headless invocation from user Codex config/rules so local MCP/plugins cannot stall artifact-only tasks before model actions
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
- 2026-06-18: After commit `35eb476`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T181545Z` showed normal collect improved but still not clean: PostHog collect advanced through multiple shards, while `posthog-cli-common`, `posthog-frontend-funnel-udf`, and `posthog-nodejs-packages` each failed the normal first write because Codex generated brittle inline Ruby/Node writer scripts that crashed before artifacts (`NoMethodError`, invalid regexp flags, template literal syntax). `collect_pair_repair` recovered the first two and was running on the third when the known-bad run was stopped. Current slice changes normal collect from a single generated read+write script requirement to a bounded read/list followed by direct literal shell heredoc/printf/tee writes, forbidding Ruby/Node/Python/Perl/awk/jq inline writers before both targets exist and requiring immediate simpler literal retry if the direct write fails.
- 2026-06-18: After commit `60449b2`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T190906Z` reached the first PostHog collect shard but emitted only local Codex plugin/MCP/state diagnostics (`MCP tools/list`, stream 504, plugin manifest/rules noise) and no model command items before the runtime scheduled `collect_pair_repair`. The run was stopped as known-bad because this is not prompt content failure; it is headless Codex environment leakage. Commit `1862ba7` added `--ignore-user-config` and `--ignore-rules`, but the next rerun proved those flags only ignore `$CODEX_HOME/config.toml` and user/project rules, not plugin/skill loading from `$CODEX_HOME`.
- 2026-06-18: Codex strict medium rerun `regres-long-posthog-ftgo-20260618T193336Z` started with `--ignore-user-config --ignore-rules` but still loaded `/Users/griogrii_riabov/.codex/.tmp/plugins/...` and `codex_core_skills` before any model action; the first PostHog collect shard wrote no artifacts for 108s and went to `collect_pair_repair`. The run was stopped as known-bad. Current slice adds shared `CommandSpec.Env` support and makes default `codex-code` run under isolated auth-only `CODEX_HOME` copied from caller `auth.json`/`installation_id`, excluding `config.toml`, MCP/plugins, skills, `.tmp/plugins`, and rules; custom runner args remain caller-owned overrides.
- 2026-06-18: After commit `4a3d95f`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T195848Z` proved `CODEX_HOME` override was applied, but Codex CLI self-created `config.toml`, `.tmp/plugins`, plugin cache, system skills and sqlite state inside that isolated home; `ngs-analysis` plugin manifest warnings still appeared from the isolated path. The run was stopped as known-bad before collect acceptance because isolated auth home alone does not disable remote plugin/app sync. Current slice keeps the isolated home and adds default `codex exec --disable` flags for plugin/app suggestion surfaces (`plugins`, `remote_plugin`, `plugin_sharing`, `apps`, `enable_mcp_apps`, `tool_suggest`, `skill_mcp_dependency_install`), verified manually on a short `codex exec` prompt to avoid `.tmp/plugins` creation.
- 2026-06-18: After commit `6cbffc8`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T201715Z` proved plugin/app surfaces stayed disabled (`plugin=0`, `skill=0`, no MCP startup), but the first collect shard produced no model/action events before the shared 75s pre-artifact window and was prematurely routed into `collect_pair_repair`. The run was stopped as known-bad. Current slice keeps strict stall semantics but sets a Codex-specific 3-minute initial pre-artifact window for collect steps, matching observed medium-slice prompt latency after removing plugin/app stderr noise.
- 2026-06-18: After commit `6929237`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T203446Z` proved backend collect no longer prematurely repaired and `step2/step4` enrichment replaced bootstrap scaffold, but frontend `init-inspect` hung after runtime success because the Activity / Events drawer was closed while the live spec tried `run-logs-mode-select.selectOption("events")`; Playwright had no bounded action timeout and would have waited for the full raised init budget. Current slice opens the drawer before log mode actions and bounds live Playwright action timeouts so this class is reported as `frontend_failed` instead of a multi-hour hang.
- 2026-06-19: Codex strict medium rerun `regres-long-posthog-ftgo-20260619T211450Z` ran from clean commit `476a109`. PostHog init succeeded and refresh collect reached 16/16 shard manifests, proving the stale missing-evidence markdown repair fix for `posthog-share-staticfiles`. The profile then failed at `refresh.step4.proposals`: `draft_artifact_enrichment` rewrote `proposal.md`/`changelog.md` but copied the bootstrap draft manifest summary string `"Drafted required runtime artifacts for this step."` into final markdown, so shared draft validation correctly rejected it as `draft_artifact_enrichment_noop_or_scaffold` / `runtime_contract_failed`. The same run exposed artifact-quality drift in refresh `step2` summary: shard completeness was reported as lexical `planned=0, succeeded=0, failed/error=2` despite observed 16/16 manifests. Current slice bans copied bootstrap manifest summaries in enrichment markdown and requires typed/observed shard completeness instead of lexical failed/error counts.
- 2026-06-20: Codex strict medium rerun `regres-long-posthog-ftgo-20260619T224351Z` ran from clean commit `1bc3bc5` with selected provider `codex-code` only. Backend execution passed `2/2` and preserved selected-provider totals/no legacy mixed quality-gate artifacts, but the matrix failed because PostHog frontend smoke launched a UI-side run `run_20260619_235435_001` that failed as `run_partial_failed` at `init.step4.proposals` after shard `posthog-frontend-funnel-udf` exhausted `collect_pair_repair` with only a thread-start/no-output artifact. FTGO frontend passed, but manual artifact review rejected final markdown that narrated runtime recovery mechanics (`current draft manifest`, `manifest target remains`, `bounded staged evidence`, `enrichment read`) instead of operator-facing architecture/proposal conclusions. Current slice rejects recovery-process narration in shared draft validation/prompt contracts and adds frontend report `runtime_details` so UI-run failures surface `run_id`, error code, current step, screenshots, and Playwright result directory without digging through raw metadata.
- 2026-06-21: Codex strict medium rerun `regres-long-posthog-ftgo-20260621T004731Z` ran from clean commit `df46cda` with selected provider `codex-code` only. PostHog init succeeded and refresh collect completed `16/16` without partial failures, but `refresh.step4.proposals` failed as `runtime_contract_failed`: enrichment wrote exact all-succeeded counts, but shared draft validation over-matched the positive line `No failed or incomplete shards remain as a coverage blocker in the current-run typed shard summary` as generic shard-gap wording. Current slice narrows validation so exact negated current-run no-blocker statements pass, while conditional/rerun shard caveats remain rejected, and strengthens step4 prompts to require exact counts plus no-shard-coverage-blocker wording in both proposal and changelog.
- 2026-06-22: Codex strict medium rerun `regres-long-posthog-ftgo-20260622T111245Z` ran from clean commit `8bb1517` with selected provider `codex-code` only. PostHog init succeeded, PostHog refresh and FTGO init both reached `step4.proposals`, then failed as `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`: focused enrichment either started no filesystem command or produced diagnostics without fresh valid `proposal.md`/`changelog.md` rewrites, leaving bootstrap proposal scaffold in place. Current slice adds one compact provider-authored `draft_artifact_enrichment_no_action_retry` that requires a single `python3` filesystem command over bounded current-run evidence; repeated noop/scaffold output remains terminal and no deterministic ACP-side proposal synthesis is added.
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

## Live E2E Draft Manifest Metadata Tolerance

### Context
Strict medium `codex-code` diagnostic run `regres-long-posthog-ftgo-20260618T225252Z` reached PostHog `refresh.step2.asis_docs` after successful collect, but failed as `runtime_contract_failed` during `draft_artifact_enrichment`. The provider rewrote evidence-backed markdown and produced a valid publish mapping, but added top-level `updated_at` to `asis-draft-manifest.json`; the shared draft parser rejected it as an unknown field.

### Plan
- [x] Keep runtime draft manifests strict and continue rejecting legacy/envelope fields such as `repo_scopes`, `compatibility`, `generated_at`, `pipeline`, and `proposals[]`.
- [x] Allow only bounded optional metadata `updated_at` alongside existing `summary`.
- [x] Update prompt contract wording, schema docs, runbook, and tests.
- [ ] Rerun strict medium `codex-code`; then run strict medium `claude-code` if Codex produces clean evidence.

### Acceptance
- [x] `updated_at` in `asis-draft-manifest.json` validates without changing execution/artifact quality separation.
- [x] Existing unknown-field tests still reject legacy manifest drift.
- [ ] Latest strict medium `codex-code` and `claude-code` runs pass with artifact/UX quality accepted or explicitly non-applicable with no blocker.

### Progress log
- 2026-06-19: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T002751Z` reached non-release machine `PASS` for selected provider totals (`2/2`) with frontend PASS and no legacy mixed quality-gate artifacts, proving execution/artifact quality separation. Manual artifact review rejected the result because promoted FTGO docs cited non-existent Maven `pom.xml` files in a Gradle repo; `artifact_quality:*` telemetry surfaced evidence-scope warnings but correctly did not flip the machine verdict.
- 2026-06-19: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T194014Z` was stopped during PostHog refresh after `posthog-share-staticfiles` reproduced a narrower blocker: manifest-only repair changed invalid citations from missing `share/Caddyfile` / `staticfiles/admin/tailwind.css` to existing files, but left the authored markdown claiming those missing paths. The fix escalates structural missing-repo-evidence manifests to collect pair repair when existing markdown still names the missing path, requires fresh markdown rewrite, and treats stale/noop markdown as terminal `runtime_contract_failed`.

---

## Live E2E Collect Repo Evidence Path Strictness

### Context
The same strict medium `codex-code` run proved that missing repo evidence paths can leak past collect validation into promoted `step2` markdown: FTGO final docs referenced `ftgo-order-service/pom.xml` and `ftgo-order-service-api/pom.xml`, while the pinned repo only contains Gradle build files for those modules. This is not a manual quality preference; it is a broken runtime evidence contract because collect citations/provenance pointed to files that do not exist under the resolved repo root.

### Plan
- [x] Validate `citations[].repo/path` against resolved collect repo roots when task context is available.
- [x] Validate semantic provenance evidence paths for entities, edges, and findings against the same resolved repo roots.
- [x] Keep existing validation behavior for callers that do not have repo-root context, so deterministic fixtures and offline contract tests remain scoped.
- [x] Update collect prompt/contract wording so providers remove unsupported claims or record coverage gaps instead of citing guessed files.
- [x] Add targeted unit coverage for missing citation paths, missing semantic evidence paths, generated repo-root suffix aliases, and runtime collect task validation.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

### Acceptance
- [x] A collect manifest that cites a missing repo file fails as `runtime_contract_failed` / repair input before downstream `step2` can promote the claim.
- [x] Existing `artifact_quality:*` telemetry remains non-gating for machine execution verdict; this fix only enforces concrete runtime evidence paths in collect contracts.
- [ ] Latest strict medium `codex-code` rerun no longer promotes missing repo/path evidence into final artifacts.
- [ ] Latest strict medium `claude-code` rerun remains pending after Codex produces clean strict medium evidence.

---

## Live E2E Collect Process Narration Strictness

### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T023512Z` preserved selected-provider execution reporting but exposed a narrower collect artifact-quality contract gap: some accepted/derived markdown still narrated runtime collection mechanics (`bounded read`, guessed or expected-missing paths) instead of clean operator-facing architecture evidence. Those strings can later contaminate `step2` as-is docs even when the machine verdict separation is working.

### Plan
- [x] Reject process-contaminated collect markdown in strict collect validation.
- [x] Route existing process-contaminated authored markdown to provider-authored `collect_pair_repair` with mandatory fresh rewrite of the same markdown target.
- [x] Keep deterministic `collect_manifest_runtime_recovery` limited to process-clean authored markdown so it cannot turn runtime narration into hidden success.
- [x] Update prompt contracts, docs, and tests for process-narration/guessed-path bans.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

### Acceptance
- [x] Final collect docs mentioning bounded reads/passes, guessed paths/files/evidence, expected-missing path checks, recovery attempts, or later repair fail validation.
- [x] Manifest-only recovery does not accept process-contaminated markdown.
- [ ] Latest strict medium `codex-code` rerun no longer promotes process-narrated collect evidence into final artifacts.

---

## Live E2E Collect Path-Scope Evidence Fairness

### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T051657Z` ran from clean commit `6057fbb` with selected provider `codex-code` only. The new execution/UX/artifact split held: `matrix_result_*` stayed non-release, selected-provider totals were `0/2`, `execution_report_*` replaced legacy quality reports, frontend was `skipped` because snapshots were missing, and no old `quality_report_*` / `quality_gates_failed` / `failure_reason=quality` evidence appeared. Backend failed in collect for both targets: PostHog `posthog-cli-common` cited stale missing paths (`cli/package.json`, `cli/docker-compose.yml`, `common/hogvm/python/README.md`) and `collect_pair_repair` produced only Codex lifecycle output; FTGO had 15/16 succeeded shards but `ftgo-restaurant-service-api...` was falsely routed to process-contamination repair because a legitimate missing Swagger spec coverage gap said an expected resource path was not present.

### Plan
- [x] Narrow process-contamination detection so concrete missing path claims remain invalid, but operator-facing missing spec/scope coverage gaps are allowed.
- [x] Add concrete path-scope file candidates to normal collect prompts, selected fairly per assigned directory scope.
- [x] Apply the same per-scope candidate fairness to collect pair/manifest repair prompts.
- [x] Add tests for `cli/common`-style multi-scope candidates and the missing-spec coverage-gap false positive.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

### Acceptance
- [x] Multi-scope candidates include real files from later scopes and do not list nonexistent stale paths.
- [x] `expected src/test/... path was not present` remains invalid process contamination.
- [x] `no OpenAPI/Swagger spec was observed under this scope` remains a valid coverage gap when not used as citation/provenance evidence.
- [ ] Latest strict medium `codex-code` rerun reaches collect completion without the `posthog-cli-common` stale-path blocker or FTGO missing-spec false positive.

---

## Live E2E Draft Artifact Readability And Index Timing

### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T203021Z` reached non-release machine `PASS` for selected-provider totals (`2/2`) with frontend PASS and no legacy mixed quality-gate artifacts. Manual artifact review still rejected the result: some final `step2.asis_docs` markdown claimed current-run `final-run-index.json` / `citation-index.json` were unavailable even though final staging later contained those indexes, and some `step4` proposal/changelog content pasted raw Python-style citation dictionaries/truncated JSON fragments instead of readable operator evidence.

### Plan
- [x] Reject stale current-run final/citation-index unavailable claims in runtime draft markdown.
- [x] Reject raw structured evidence dumps in operator-facing draft markdown.
- [x] Update draft enrichment prompts so providers omit downstream index availability during step2 when indexes are not yet present, and summarize index/citation evidence instead of pasting raw objects.
- [x] Update docs and targeted prompt/validation tests.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

### Acceptance
- [x] `step2` no longer promotes stale `final-run-index/citation-index not found` claims.
- [x] `step4` proposal/changelog drafts cannot pass validation with raw `{'id': ...}` / `documents=[{...}]` / `citations=[{...}]` dumps.
- [ ] Latest strict medium `codex-code` rerun has accepted artifact quality for readability, index truthfulness and decision readiness.
- [ ] Latest strict medium `claude-code` rerun remains pending after Codex produces clean strict medium evidence.

---

## Live E2E Proposal Final-Index Truthfulness

### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260621T022237Z` reached non-release machine `PASS` for selected-provider totals (`2/2`) and frontend PASS. Runtime recovery behaved correctly for bootstrap-only drafts, but manual artifact review rejected FTGO proposals: `proposals/runtime-recommendations.md` and `reports/changelog/runtime-proposals.md` stated `No current-run final-run-index document list was available` even though `final-run-index.json` was present with `51` canonical documents.

### Plan
- [x] Extend runtime draft validation to reject stale final-index document-list availability claims.
- [x] Update `step4.proposals` enrichment prompt to require canonical document count summaries when `final-run-index.json` is present and omission when it is absent.
- [x] Add regression coverage using the observed FTGO stale phrase.
- [x] Update live E2E docs/spec/architecture wording.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

### Acceptance
- [x] Proposal/changelog drafts cannot pass validation with `No current-run final-run-index document list was available`.
- [ ] Latest strict medium `codex-code` rerun has accepted artifact quality for proposal index truthfulness and decision readiness.
- [ ] Latest strict medium `claude-code` rerun remains pending after Codex produces clean strict medium evidence.

---

## Live E2E Proposal Count And Mermaid Collision Follow-Up

### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260621T193026Z` reached non-release machine `PASS` for selected-provider totals (`2/2`) with frontend PASS and no legacy mixed quality-gate artifacts. Manual artifact review still found two artifact-quality issues: FTGO proposal/changelog used the variant `final-run-index.json (0 observed document entries)` even though the current `final-run-index.json` had `51` `canonical_documents`, and FTGO C4 container output duplicated a Mermaid node id for distinct service ids that normalized to the same slug.

### Plan
- [x] Reject parenthesized/styled stale zero-document `final-run-index` claims in runtime draft markdown.
- [x] Make C4 diagram generation collision-free for sanitized Mermaid node ids.
- [x] Make generated component/code diagram artifact paths collision-free for distinct entity ids that normalize to the same slug.
- [x] Update spec/runbook/testing docs and targeted tests.
- [x] Run full DoD.
- [ ] Commit and rerun affected strict medium evidence.

### Acceptance
- [x] Proposal/changelog drafts cannot pass validation with `final-run-index.json (0 observed document entries)` without validated zero-document evidence.
- [x] Distinct model entity ids such as `svc.foo-bar` and `svc.foo.bar` produce distinct Mermaid node ids and distinct diagram artifact paths.
- [ ] Latest strict medium `codex-code` rerun has accepted artifact quality for proposal index truthfulness and C4/Mermaid usefulness.
- [ ] Latest strict medium `claude-code` rerun remains pending after Codex produces clean strict medium evidence.

---

## Live E2E Collect Repair Canonical Semantic Shape

### Context
Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T071328Z` reproduced a collect recovery failure before a full profile completed. The first PostHog shard failed as `runtime_contract_failed`; later shards showed the same provider behavior but were sometimes rescued by manifest-only repair. The repeated root cause was `collect_pair_repair` writing legacy `shard-pack-manifest.json` semantic shape: `semantic.coverage.notes` as a string, entities with direct `repo/path/evidence` or missing `provenance`, edges with `relation/source/target`, findings without `description`, and provenance as direct `{repo,path}` or missing `kind/confidence/evidence[]`.

### Plan
- [x] Strengthen collect manifest contract lines and compact checklist with canonical semantic object shapes.
- [x] Add canonical semantic-shape guidance to collect pair repair and manifest-only repair prompts.
- [x] Add prompt contract tests for the exact live failure fields and forbidden aliases.
- [x] Update live E2E runbook/spec/architecture/testing docs.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium `claude-code`.

### Acceptance
- [x] Prompt contracts explicitly reject legacy collect semantic aliases that appeared in the failed `claude-code` run.
- [ ] Latest strict medium `claude-code` rerun no longer fails from unchanged/legacy collect repair manifest shape.
- [ ] Latest strict medium `claude-code` and `codex-code` evidence both reach clean execution and accepted manual artifact/UX decisions.

---

## Live E2E Collect Repair Canonical Path Mapping

### Context
Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T082911Z` confirmed that the canonical semantic-shape fix worked for `posthog-bin`, but exposed the next collect repair drift. The compact `collect_pair_repair` wrote provider-authored markdown and canonical semantic objects, yet set `documents[0].canonical_path` to the live staging path `reports/taskruns/.../staging/shards/posthog-bin/bin-overview.md`. Strict contract validation correctly rejected it, then `collect_manifest_repair` stalled/no-op without replacing the manifest. The run was stopped after terminal shard failure because the matrix could no longer become acceptance evidence.

### Plan
- [x] Centralize stable collect canonical path generation from shard slug + authored doc path.
- [x] Add exact `documents[].path -> documents[].canonical_path` mapping to manifest-only repair prompts.
- [x] Add exact `documents[0].canonical_path` to compact collect pair repair prompts and explicitly forbid `reports/taskruns/**`, `/staging/`, absolute `write_root`, raw runtime paths and duplicated artifact-root canonical paths.
- [x] Add prompt/skeleton contract tests for the live-observed drift.
- [x] Update live E2E runbook/spec/architecture/testing docs.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium `claude-code`.

### Acceptance
- [x] Compact collect pair repair prompt no longer leaves canonical path inference open.
- [x] Manifest-only repair prompt lists stable canonical-path mappings for existing authored docs.
- [ ] Latest strict medium `claude-code` rerun no longer fails on staging/taskrun `documents[].canonical_path`.
- [ ] Latest strict medium `claude-code` and `codex-code` evidence both reach clean execution and accepted manual artifact/UX decisions.

---

## Live E2E Step2 Typed Shard Completeness Hardening

### Context
- 2026-06-23 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T091625Z` confirmed the collect `canonical_path` fix across 16 PostHog shard manifests, including the previously failing `posthog-bin` shard.
- The same run was stopped during `init.step2.asis_docs` after manual artifact-quality inspection found misleading enriched draft markdown: typed shard-summary evidence had `items=16` with succeeded statuses, but `summary.md` dumped metadata-like lines (`meta: 1 keys`, `step_id`, `domain_id`, `strategy`) and `architect-summary.md` claimed `Staging shard directory contains 0 files`.
- This is a runtime draft contract problem, not product functionality and not artifact-quality telemetry: when typed current-run shard status is readable, step2 coverage must report exact planned/succeeded/failed/incomplete counts or fail validation.

### Plan
- [x] Add strict `step2.asis_docs` draft validation against current-run typed shard-summary files.
- [x] Reject metadata-only shard-summary key dumps and false zero-shard/zero-file claims when typed/shard manifest evidence exists.
- [x] Strengthen compact `draft_artifact_enrichment_no_action_retry` prompt for step2 typed completeness.
- [x] Add unit and prompt contract regression tests.
- [x] Run full DoD.
- [x] Commit and rerun strict medium `claude-code`.

### Acceptance
- [x] Valid step2 markdown with exact `N/N succeeded` and no failed/incomplete statement passes when typed summary shows all succeeded.
- [x] Live-observed misleading markdown fails as `runtime_contract_failed`.
- [x] Latest strict medium rerun no longer accepts metadata-dump/false-zero step2 coverage.

---

## Live E2E Step0 Bounded-Read Marker And Claude Collect Window

### Context
- 2026-06-23 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T113848Z` validated the step2 hardening: downstream `step2/3/4` no longer masked partial collect, and machine execution reports kept artifact quality telemetry out of the execution verdict.
- PostHog failed at collect as execution-only `runner_unavailable`: two shards produced no stdout/stderr and no authored files across normal collect, fresh retry and focused collect-pair repair, each bounded by the 180s pre-artifact window.
- FTGO failed at `init.step0.constitution` due a draft-validation false positive: provider-authored `charter-overview.md` was evidence-backed and decision-ready, but the hard scaffold marker `bounded read` rejected the valid coverage-gap sentence “not inspected in this bounded read.”

### Plan
- [x] Narrow draft scaffold markers from generic `bounded read` to process-specific markers such as `bounded read roots`, `bounded read pass`, `bounded evidence read` and `bounded staged evidence`.
- [x] Add regression coverage that accepts evidence-backed step0 coverage gaps using `bounded read` and still rejects recovery-process `bounded read roots`.
- [x] Extend `claude-code` collect initial/retry pre-artifact window to 5 minutes while leaving draft/enrichment windows unchanged.
- [x] Update prompt contract wording and live E2E docs/spec/testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium `claude-code`.

### Acceptance
- [x] Evidence-backed `charter-overview.md` with a normal bounded-read coverage gap passes draft validation.
- [x] Recovery/process bounded-read markers remain contract-invalid.
- [x] Claude collect policy exposes 5-minute initial/retry pre-artifact windows for `init|refresh.step1.collect`.
- [ ] Latest strict medium `claude-code` rerun no longer fails from the FTGO false-positive marker and has enough collect budget to test PostHog shard recovery.

---

## Live E2E Collect Pair Silent No-Fresh Retry

### Context
- 2026-06-23 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T182236Z` ran from clean commit `f5c35cc` with selected provider `claude-code` only.
- PostHog backend passed init/refresh, frontend desktop smoke passed, `execution_report_*` replaced legacy quality reports, and artifact-quality telemetry stayed out of the machine verdict.
- FTGO failed in `init.step1.collect`: 15/16 shards succeeded, but shard `ftgo-application-ftgo-kitchen-service-api-ftgo-kitchen-service-contracts-ftgo-order-cc671c29a0aa` wrote stale/invalid collect files, then focused `collect_pair_repair` stalled pre-artifact with `stdout=0`, `stderr=0` and no fresh authored mutation. Reports preserved the root as collect partial execution failure (`runtime_contract_failed`, `partial_failure_count=1`), with frontend skipped for that profile.
- Root issue: collect-pair repair treated a silent no-fresh pre-artifact repair stall over stale artifacts as final exhaustion. It needed one bounded provider-authored retry, without deterministic artifact synthesis and without treating artifact quality as a machine gate.

### Plan
- [x] Give collect-pair repair its own activity policy so production/default collect repair windows remain bounded for live providers while tests can exercise explicit wall-clock caps.
- [x] Add one focused retry when collect-pair repair stalls before fresh authored mutation with empty stdout/stderr and the provider policy allows zero-output retry.
- [x] Keep success provider-authored only: retry must write markdown + `shard-pack-manifest.json` and pass strict validation.
- [x] Classify repeated silent no-fresh collect-pair repair exhaustion as `runner_unavailable`; classify fresh-but-invalid repair output as `runtime_contract_failed`.
- [x] Add providercommon regression tests for successful silent/no-fresh retry and exhausted silent/no-fresh classification.
- [x] Update live E2E runbook, pipeline spec, architecture and testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

### Acceptance
- [x] First silent/no-fresh collect-pair repair does not emit final exhausted telemetry before the allowed retry.
- [x] Second provider-authored repair can recover stale/process-contaminated collect artifacts only by fresh rewriting the markdown and manifest.
- [x] Repeated silent/no-fresh repair exhaustion is `runner_unavailable`.
- [ ] Latest strict medium `claude-code` rerun no longer fails from the FTGO silent/no-fresh collect-pair repair stall.

---

## Live E2E Draft Enrichment Marker Cleanup Retry

### Context
- 2026-06-24 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T230003Z` proved the silent collect-pair retry and draft enrichment paths were active, but still failed both selected backend slots.
- PostHog `refresh.step4.proposals` fresh-rewrote `proposal.md` and `changelog.md`, but `changelog.md` leaked process wording (`bounded read roots`) into operator-facing markdown. Runtime correctly rejected it as `runtime_contract_failed`, but had no targeted cleanup retry after a real markdown mutation.
- FTGO `init.step0.constitution` fresh-rewrote `charter-overview.md`, but leaked step0-invalid process/downstream wording (`validator output`, `draft manifest`, `later passes`). Runtime correctly rejected it as `runtime_contract_failed`.

### Plan
- [x] Add one provider-authored `draft_artifact_enrichment_marker_cleanup` retry when every referenced markdown target changed, but strict validation only rejects scaffold/process/downstream wording.
- [x] Keep unchanged scaffold/noop enrichment terminal: cleanup retry is not available when markdown targets did not fresh-mutate.
- [x] Extend enrichment prompts with marker-cleanup instructions for step0 and step4.
- [x] Add providercommon and promptcontract regression tests for proposals marker cleanup and step0 downstream/process cleanup.
- [x] Update live E2E runbook, pipeline spec, architecture and testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

### Acceptance
- [x] Fresh-rewritten proposal/changelog content with process marker contamination gets one provider-authored cleanup retry.
- [x] Fresh-rewritten step0 constitution content with downstream/process wording gets one provider-authored cleanup retry.
- [x] Repeated marker contamination or unchanged scaffold remains `runtime_contract_failed`.
- [ ] Latest strict medium `claude-code` rerun no longer fails from PostHog `bounded read roots` or FTGO step0 marker leakage.

---

## Live E2E Draft Enrichment Shard Status And Write-Set Cleanup

### Context
- 2026-06-24 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260624T055541Z` ran selected provider `claude-code` only with selected-provider totals preserved and no legacy mixed quality artifacts, but both backend slots failed as `runtime_contract_failed`.
- PostHog `refresh.step2.asis_docs` completed collect 16/16 and fresh-rewrote step2 drafts, but `architect-summary.md` still used generic conditional shard-gap wording instead of exact current-run typed shard status.
- FTGO completed collect 16/16 and wrote valid step2 markdown under `draft_final_root`, but duplicated `overview.md`, `summary.md`, and `architect-summary.md` in `write_root`, causing a write-set failure even though the final draft files were otherwise valid.

### Plan
- [x] Add one provider-authored `draft_artifact_enrichment_shard_status_cleanup` retry for fresh step2 markdown rejected only by generic conditional shard-gap wording.
- [x] Add one provider-authored `draft_artifact_enrichment_write_set_cleanup` retry that deletes only byte-identical referenced markdown duplicates from `write_root` while keeping arbitrary extra writes terminal.
- [x] Keep success provider-authored only: no ACP-side synthesis of step2 draft markdown or manifest shape.
- [x] Add providercommon and promptcontract regression tests for both retry modes.
- [x] Stabilize the existing silent/no-fresh collect-pair retry test window so package tests are not timing-sensitive on loaded trusted hosts.
- [x] Update live E2E runbook, pipeline spec, architecture and testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

### Acceptance
- [x] Fresh step2 markdown with generic shard-gap caveats gets one provider-authored shard-status cleanup retry.
- [x] Byte-identical draft markdown duplicates in `write_root` get one cleanup retry that deletes only those misplaced duplicates.
- [x] `extra.md`/unreferenced write-set violations remain `runtime_contract_failed`.
- [ ] Latest strict medium `claude-code` rerun no longer fails from PostHog generic shard wording or FTGO duplicated write_root drafts.

---

## Live E2E Final Index Document ID Collision

### Context
- 2026-06-24 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260624T101459Z` preserved selected-provider totals and no legacy mixed quality-gate artifacts, but still failed both backend slots.
- PostHog reached `init.step4.proposals` and correctly failed scaffold/noop proposal enrichment: `proposal.md` and `changelog.md` still contained bootstrap placeholder content.
- FTGO completed collect `16/16` and step2 enrichment produced readable, decision-ready as-is markdown with exact shard completeness, but staged final assembly failed before writing `final-run-index.json`: two distinct shard documents reused provider id `doc.overview`, causing `canonical_documents[12].id must be unique`.

### Plan
- [x] Keep unique provider-authored `manifest.Documents[*].id` values when they identify one canonical path.
- [x] Remap repeated provider-authored document ids across distinct `canonical_path` values to deterministic canonical-path-derived ids before final index validation.
- [x] Remap citation `document_ids` into the same canonical document namespace.
- [x] Add docflow regression coverage for repeated `doc.overview` across distinct canonical paths.
- [x] Update pipeline/runbook/testing docs.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

### Acceptance
- [x] Duplicate source document ids no longer produce duplicate `final-run-index.json.canonical_documents[].id`.
- [x] Citation index references the remapped unique document ids.
- [ ] Latest strict medium `claude-code` rerun no longer fails FTGO at final-index assembly.

---

## Live E2E DoD Precheck Timeout

### Context
- 2026-06-26 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260625T224145Z` completed the PostHog backend/frontend slot, then the FTGO profile stalled before runtime in `make contracts test lint build`.
- Existing harness hard timeout covered provider pipelines and UI dependency/browser prechecks, but DoD precheck still ran as an unbounded direct shell command.
- This left the matrix profile `running` until manual interrupt and produced no useful headless taskrun evidence because runtime had not started.

### Plan
- [x] Add `ACP_LIVE_PRECHECK_DOD_TIMEOUT_SEC` and run `make contracts test lint build` through the existing process-group precheck timeout helper.
- [x] Keep terminal classification as `precheck_failed`, with `[precheck-timeout]` evidence in `precheck-make.log`.
- [x] Keep DoD timeout separate from provider runtime timeout and artifact/UX quality.
- [x] Add a script regression test for a hung DoD precheck.
- [x] Update runbook, pipeline spec, architecture and testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

### Acceptance
- [x] A hung `make contracts test lint build` no longer leaves a child batch/profile indefinitely running.
- [x] Reports retain `execution_report_<batch-id>.md`; legacy mixed quality-gate artifacts stay absent.
- [ ] Latest strict medium rerun reaches runtime or fails with a bounded, classified precheck/provider/runtime reason.

---

## Live E2E Step2 Enrichment And Ask Smoke Cleanup

### Context
- 2026-06-30 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260630T003528Z` preserved the new execution/UX/artifact separation: `matrix_result` used selected provider totals only, `execution_report_<batch-id>.md` was produced, and no legacy mixed `quality_report_*`/`quality_gates_failed` release gate evidence was observed.
- PostHog backend still failed in `init.step2.asis_docs`: after multiple no-op/scaffold enrichment attempts, compact step2 retry finally fresh-wrote markdown, but copied sampled shard prose with unbalanced inline backticks; the syntax cleanup retry then stalled before valid artifacts and the run correctly ended as `runtime_contract_failed`.
- FTGO backend passed both init and refresh after focused collect/draft repairs, but frontend failed in default Ask smoke: the async `qa.ask` run remained running after the fixed 120s Playwright poll, was classified as generic `playwright_failed`, and teardown left an orphan `claude` provider process until manually killed.

### Plan
- [x] Keep `release_verdict_*` execution-only and leave artifact/UX quality as separate SWE report inputs.
- [x] Add bounded QA smoke polling with `ACP_UI_QA_POLL_TIMEOUT_SEC` and `ACTIVE_RUN_TIMEOUT` marker classification.
- [x] Best-effort cancel active frontend QA runs after Playwright failure before `acp serve` teardown, preventing orphan provider processes.
- [x] Tighten step2 markdown syntax retry prompt: rewrite every referenced markdown target in one bounded command, remove sampled shard prose snippets, keep exact typed shard completeness, omit stale downstream-index claims, and avoid generic shard-gap wording.
- [x] Tighten compact step2 retry prompt so evidence bullets are path plus paraphrased architecture signal only, not copied first paragraphs.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

### Acceptance
- [x] QA smoke timeout is `active_run_timeout`, not generic `playwright_failed`.
- [x] Failed QA smoke does not leave an unmanaged provider process after harness teardown.
- [x] Step2 syntax cleanup is provider-authored and does not introduce deterministic ACP-side artifact synthesis.
- [ ] Latest strict medium `claude-code` rerun no longer fails PostHog step2 on malformed markdown/no-op enrichment.
- [ ] Latest strict medium frontend Ask smoke either succeeds or fails with bounded `active_run_timeout` plus cancellation evidence.

---

## Artifact Quality Excellent Gate: Proposals/Findings Linkage

### Context
- Current live diagnostics can produce non-empty structured findings while `proposals/runtime-recommendations.md` or `reports/changelog/runtime-proposals.md` still says no structured finding summary was present.
- Runtime/contract success and artifact usefulness must stay separated: ProvenArch emits generic product facts through `quality_signals[]`; live E2E consumes public reports and `*-quality.json` as black-box evidence.
- `runtime_quality.repair_heavy` / `runtime_quality.stall_pressure` are not strict blockers by themselves, but they must prevent an `Excellent` label.

### Plan
- [x] Add generic product `artifact_quality.proposals_findings_disconnected` and `artifact_quality.proposals_low_actionability` signals for succeeded normal runs.
- [x] Tighten `step4.proposals` draft validation and enrichment prompts so non-empty current-run findings must be linked by finding ID and must not be denied as absent.
- [x] Add live E2E `Excellent Blockers` reporting and cap `Excellent` on repair/stall/runtime-quality and `analysis:*` issues.
- [x] Add regression tests for product signals, draft validation, live E2E classification and diagram datastore deduplication.
- [x] Run full DoD.
- [x] Rerun `smoke tiny` on `codex-code`.

### Acceptance
- [x] Weak proposals/changelog that deny non-empty findings produce artifact-quality failure.
- [x] Repair/stall pressure remains non-blocking but no longer reports `Excellent`.
- [x] No live E2E profile/provider/matrix logic leaks into ProvenArch product quality signals.

---

## Runtime Stability To Excellent

### Context
- Current `smoke tiny` live evidence can reach `strict PASS` and `artifact_quality_status=passed`, but still receive `Needs review` because good artifacts were obtained through repair/stall pressure.
- Goal is not to weaken acceptance: `Excellent` still requires runtime contract pass, artifact-quality pass, no `runtime_quality.*`, no `analysis:*`, no repair/stall pressure, and applicable frontend evidence.
- Product and live layers remain black-box separated: ProvenArch improves generic prompt/validation contracts; live E2E reports public evidence only.

### Plan
- [x] Strengthen normal `step0.constitution`, `step2.asis_docs`, and `step4.proposals` prompts so the same provider turn completes evidence-backed drafts before successful exit.
- [x] Make `step4.proposals` high/medium actionability markdown-safe and bullet-only: exact Finding ID, copied Severity value, Affected surface/path, Recommended operator action with concrete verb, Residual gap; reject tables and generic inspect/review/decide-only proposals.
- [x] Add live E2E `Excellent blockers by step` diagnostics from public taskrun logs/raw metadata and propagate additive `excellent_blockers_by_step` into batch/matrix JSON reports.
- [x] Update `docs/spec/PIPELINE_SPEC.md` and `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.
- [x] Run targeted tests and full DoD: `make contracts`, `make test`, `make lint`, `make build`.
- [x] Rerun diagnostic `smoke tiny` on `codex-code`.

### Acceptance
- [x] Valid step4 actionable bullets pass validation; malformed tables/inline-code and generic review-only proposals fail before promotion.
- [x] Live reports identify which `step_id` caused repair/stall pressure that blocks `Excellent`.
- [x] No provider/profile/matrix/repo-specific logic is added to ProvenArch product code.

### Live validation notes
- 2026-07-04: `smoke-tiny-bank-stability2-20260704T095509Z` failed with `init.step4.proposals` `low_actionability`; fixed by requiring copied high/medium Severity values and clearer validation hints.
- 2026-07-04: `smoke-tiny-bank-stability3-20260704T104303Z` still failed before promotion with `init.step4.proposals` `finding_linkage`; reports correctly identify step-level `repair_exhausted`, `repair_heavy`, and `stall_pressure`. Added exact nested findings path/backticked-ID guidance as follow-up hardening. `Excellent` remains blocked until a subsequent live run avoids focused repair/stall and passes runtime contract.
- 2026-07-04: `smoke-tiny-bank-runtime-pass-20260704T121128Z` reached non-release matrix `PASS` with `runtime_contract_status=passed`, `artifact_quality_status=passed`, `quality_gates_failed=0`, and `artifact_quality_failed=0`. Verdict remains `Needs review` because `repair_attempts=5`, `focused_repairs=5`, `stall_count=12`, and `Excellent Blockers By Step` still point at init `step0`/`step1`/`step2`/`step3`/`step4` plus refresh `step2`/`step4`. The remaining P1 blocker is same-turn provider completion: current artifacts are useful after focused repair, but the happy path still relies on repair/stall pressure.

### Follow-up Plan: Runtime Contract PASS
- [x] Treat synthetic current-run finding placeholders (`no-current-run-finding-id`, `no structured current-run finding ID`, `finding unavailable`) as explicit `step4.proposals` draft validation failures when current-run `findings.md` contains exact IDs.
- [x] Add a live-shaped regression where nested staged `findings.md` has backticked IDs but proposal/changelog use a synthetic placeholder.
- [x] Keep live E2E classification black-box by mapping the public validation text to `finding_linkage` without importing product internals.
- [x] Sync pipeline spec and live runbook for the stricter finding-linkage contract.
- [x] Rerun `smoke tiny` on `codex-code`; observed intermediate result is `runtime_contract_status=passed`, `artifact_quality_status=passed`, strict `PASS`, and `Needs review` because repair/stall pressure remains.

---

### Plan ID
EP-20260704-needs-review-to-excellent-no-repair-pressure

### Context
The latest `smoke tiny` diagnostic (`smoke-tiny-bank-runtime-pass-20260704T121128Z`) reached strict machine `PASS` with `runtime_contract_status=passed` and `artifact_quality_status=passed`, but live E2E correctly capped the label at `Needs review`. The remaining blockers are runtime-quality signals, not artifact-quality failures: first-pass `step2.asis_docs` and `step4.proposals` drafts still need focused repair, `step0.constitution` bootstrap text leaks downstream/runtime-only wording, and valid post-artifact controlled stops are over-attributed as stall pressure for `step1.collect` and `step3.findings`.

### Goals (must have)
- [x] Make normal `step0.constitution`, `step2.asis_docs`, and `step4.proposals` prompts require same-turn completion, with no final/analysis-only response before evidence-backed draft markdown exists.
- [x] Remove downstream/runtime-only wording from the initial `charter-overview.md` scaffold while preserving strict validation that blocks unchanged bootstrap drafts.
- [x] Make normal `step2.asis_docs` prompts enumerate current-run typed shard/index evidence and require exact `planned/succeeded/failed/incomplete` counts when available.
- [x] Make normal `step4.proposals` prompts require exact nested staged findings file reads and exact finding-ID linkage/actionability in the first-pass proposal/changelog.
- [x] Make normal and focused `step4.proposals` guidance require both `proposal.md` and `runtime-proposals.md`/`changelog.md` to contain the exact literal `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` string plus an explicit `no-shard-coverage-blocker` statement when typed shard status is visible.
- [x] Treat valid-artifact controlled stop as diagnostic telemetry, not `runtime_quality.stall_pressure`; keep invalid/no-fresh/pre-artifact/repair stalls as real Excellent blockers.
- [x] Add `first_validation_error_excerpt`, `stop_kind`, and `initial_artifact_state` to live `excellent_blockers_by_step` without schema-breaking removals.
- [x] Update tests, fixtures, docs/spec/runbook and run full DoD.
- [x] Rerun diagnostic `smoke tiny` on `codex-code` after the final validation/reporting fixes and classify strict/artifact/runtime pressure status.
- [x] Route recoverable draft semantic/shape repair failures, including step4 missing proposal sections and missing proposal shard completeness, into provider-authored `draft_artifact_enrichment` instead of terminal scaffold repair failure.
- [x] Harden normal `step2.asis_docs` / `step4.proposals` prompts after live evidence so first-pass draft markdown is written before the manifest and `step4` includes validator-required proposal/changelog sections by name.
- [ ] Remove the remaining normal first-pass scaffold/finding-linkage repair path for `step2.asis_docs` and `step4.proposals` before archiving this plan as `Excellent`-ready.
- [ ] Rerun diagnostic `smoke tiny` on `codex-code` after the 2026-07-05 manifest-last/required-sections prompt hardening; continue only if the result is `strict PASS`, `artifact_quality_status=passed`, no actual `runtime_quality.repair_heavy`, no actual `runtime_quality.stall_pressure`, and verdict `Excellent`.
- [ ] Fix or explicitly route the live-observed `init.step1.collect` no-artifact pre-artifact stall on root-file shards before rerunning Phase A, because partial collect prevents `step2`/`step4` validation and blocks `Excellent`.
- [x] Harden collect manifest normal/repair prompts for structural self-checks: `semantic.questions[*].id/text`, unique `citations[].id`, non-empty citation `claim_ids`/`document_ids`, `documents[].citation_ids` references and file-level repo/provenance paths.
- [x] Add one provider-authored `collect_manifest_shape_cleanup` retry after failed manifest-only repair for clean manifest-shape errors such as missing question text, duplicate citation IDs and citation/document binding drift; repeated invalid output remains `runtime_contract_failed`.
- [x] Add `terminal_validation_error_excerpt` to live `excellent_blockers_by_step` while preserving `first_validation_error_excerpt`, so repair exhaustion reports show the final terminal cause.
- [x] Rerun diagnostic `smoke tiny` on `codex-code` after the collect manifest shape-cleanup slice and classify strict/artifact/runtime pressure status.
- [x] Remove the `init.step2.asis_docs` first-pass missing-manifest/exact-shard-completeness/operator-summary repair path: normal Codex now writes all four step2 targets and passes strict draft validation on init and refresh.
- [x] Harden normal `step2.asis_docs` first-pass prompt with a compact same-command write sequence: exact `write_root`/`draft_root`, all three markdown targets, `asis-draft-manifest.json` last, and `test -s` checks for all four targets.
- [x] Require normal `step2.asis_docs` prompt guidance to place exact typed shard completeness and `no-shard-coverage-blocker` in both `summary.md` and `architect-summary.md` when all current-run shards succeeded.
- [x] Reject shell-substituted `step2.asis_docs` markdown that contains empty evidence reference slots such as `from  and`, `checked:  and`, `under .`, or `Use  and`.
- [x] Reject `step4.proposals` actionable bullets with placeholder finding IDs such as `Finding ID: none` when current-run findings are non-empty, even if an exact finding ID appears elsewhere in the file.
- [x] Rerun diagnostic `smoke tiny codex-code` after the shell-safe step2 / placeholder finding-ID validation patch from a clean committed tree and capture the next blocker.
- [x] Align `codex-code` normal draft pre-artifact window with the existing Claude/Qwen draft budget and require `step2.asis_docs` first provider item to be command execution, not assistant/status prose.
- [x] Rerun diagnostic `smoke tiny codex-code` after the Codex draft pre-command stall fix from a clean committed tree and classify the outcome.
- [x] Rerun diagnostic `smoke tiny codex-code` after Codex quota/auth is available again; current provider usage-limit blocker prevented reaching `step2`.
- [x] Remove the remaining init-only `step2.asis_docs` first-pass stale downstream-index wording: when current-run final/citation indexes are not present yet, overview must omit downstream index status instead of saying they are unavailable.
- [x] Remove the remaining init-only `step4.proposals` first-pass low-actionability repair path: medium/high findings must link exact finding ID, affected surface/path and concrete operator action before repair.
- [ ] Run broader trusted validation only after Phase A is `Excellent` from a clean committed tree with `qwen`, `claude`, and `codex` available in `PATH`.

2026-07-15 deterministic follow-up: fake as-is evidence no longer leaks taskrun staging paths;
normal step2 first-pass guidance now explicitly omits absent downstream-index status; normal
step4 self-check requires exact same-line medium/high finding linkage before manifest write.
The two remaining prompt/fake-output implementation items above are complete in code and focused
tests; the clean-tree trusted rerun remains open and is intentionally separate from deterministic DoD.

### Non-goals
- [ ] Do not weaken draft validation or artifact-quality gates.
- [ ] Do not synthesize draft markdown deterministically inside ACP.
- [ ] Do not add live profile/provider/matrix/repo-specific strings to ProvenArch product/runtime code.
- [ ] Do not treat diagnostic smoke as release readiness.

### Acceptance criteria
- [x] Focused prompt/validation tests cover same-turn draft completion, exact step2 shard counts, exact step4 finding linkage/actionability, and clean step0 scaffold wording.
- [x] Runtime quality tests prove valid controlled stops do not emit `runtime_quality.stall_pressure`, while invalid/no-fresh/pre-artifact stalls still do.
- [x] Python batch fixtures prove `valid_artifact_controlled_stop` does not block `Excellent`, while actual repair-heavy/finding-linkage repair still does.
- [x] `make contracts`, `make test`, `make lint`, `make build` pass with exact Node toolchain.
- [x] Diagnostic `smoke tiny` on `codex-code` is rerun and evaluated for strict PASS, artifact quality PASS, repair/stall pressure and label.
- [x] Latest P1 exact shard-completeness patch has full DoD evidence.
- [x] Latest P1 exact shard-completeness patch has diagnostic `smoke tiny codex-code` evidence before any broader trusted matrix.
- [x] Latest P1 step4 repair-to-enrichment route patch has full DoD and diagnostic `smoke tiny codex-code` evidence before any broader trusted matrix.
- [x] Latest P1 manifest-last/required-sections prompt hardening has full DoD and diagnostic `smoke tiny codex-code` evidence before any broader trusted matrix.

### Progress Log
- 2026-07-04: Full DoD passed before live validation, then diagnostic `smoke-tiny-bank-20260704T135401Z` ran selected `codex-code` only. The run failed strict runtime contract at `init.step4.proposals`; artifact quality did not fail, but promotion stopped before refresh. New live diagnostics correctly exposed step-level blockers: `step4.proposals` repair exhausted, `step2.asis_docs` invalid first-pass scaffold, `step0.constitution` downstream wording, and valid controlled stops for non-blocking lifecycle telemetry.
- 2026-07-04: Follow-up patch after that live evidence made step4 actionable-finding prompts explicit that one high/medium finding must be represented by one bullet with all required fields on the same bullet line, because the provider split ID/severity/path and action/residual-gap across separate bullets. The same patch fixed live classification so raw provider stdout and placeholder-hint wording no longer mask `low_actionability` as `markdown_syntax` or `finding_linkage`.
- 2026-07-04: Step0 normal prompts now explicitly ban `later collection steps` / later-analysis / future-pipeline wording in `charter-overview.md`; unknowns must be phrased as current constitution evidence gaps.
- 2026-07-04: Diagnostic `smoke-tiny-bank-20260704T145052Z` still failed strict at `init.step4.proposals`, but the final provider-authored proposal/changelog after cleanup contained exact current-run finding IDs, severity, affected surfaces and recommended actions. The remaining terminal validation was a false positive on `findings above`; shared draft validation now allows that wording when the same draft contains substantive linked findings/proposals, while still rejecting dangling references without linked content.
- 2026-07-04: Live report aggregation now treats `retry scheduled` with action `terminate_and_validate` as valid-artifact controlled completion, not repair pressure. Normal step2/step4 prompts now include bounded public evidence hints: exact shard completeness parsed from current-run shard-summary `items[].status` and a current-run finding ID/severity preview. Full DoD passed with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`.
- 2026-07-04: Diagnostic `smoke-tiny-bank-20260704T154848Z` stopped at precheck because this active ExecPlan had no open goal after documentation updates; this was classified as docs bookkeeping, not runtime evidence, and `go test ./internal/docsync` passed after adding the open live-validation goal.
- 2026-07-04: Diagnostic `smoke-tiny-bank-20260704T155014Z` completed with non-release matrix `PASS`: `hard_pass=1`, `runtime_contract_status=passed`, `artifact_quality_status=passed`, `quality_gates_failed=0`, `artifact_quality_failed=0`, and no product `artifact_quality.*` signals. Verdict remains `Needs review` with `runtime_quality.repair_heavy` and `runtime_quality.stall_pressure`: `repair_attempts=4`, `focused_repairs=4`, `repair_exhausted=0`, `stall_count=6`, `valid_artifact_controlled_stops=6`. Step-level blockers are `init.step2.asis_docs` and `refresh.step2.asis_docs` (`placeholder_or_scaffold`: first-pass overview lacks concrete repo/path/citation/staged refs) plus `init.step4.proposals` and `refresh.step4.proposals` (`finding_linkage`: first-pass proposal lacks exact current-run finding IDs). This confirms strict/artifact-quality success and identifies the remaining no-repair hardening target for `Excellent`.
- 2026-07-05: Started the no-repair hardening slice for the remaining `step2`/`step4` blockers. Normal `step2.asis_docs` and `step4.proposals` prompts now use evidence-first first-action contracts: bounded current-run evidence read/list and validation-ready manifest/markdown writes happen inside the first filesystem work unit, while bootstrap heredoc remains limited to focused repair.
- 2026-07-05: Full DoD passed for the initial P1 prompt slice, then diagnostic `smoke-tiny-bank-20260705T104741Z` ran selected `codex-code` only through `scripts/full-run-batch-matrix.sh`. Result: strict `FAIL` with `runtime_contract_failed`; artifact quality stayed passed (`artifact_quality_failed=0`). The useful signal is that `step2.asis_docs` first-pass output became valid without focused repair, while `init.step4.proposals` exhausted repair as `manifest_shape` because both final proposal files omitted the exact current-run shard completeness literal `planned=10 succeeded=10 failed=0 incomplete=0`. The same run also showed a real `init.step1.collect` pre-artifact stall, so broader trusted matrices remain blocked until Phase A reaches `Excellent`.
- 2026-07-05: Follow-up patch tightened normal and focused `step4.proposals` prompts/repair hints so both proposal and changelog outputs must carry the exact `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` literal and explicit `no-shard-coverage-blocker` statement when typed shard status is readable. This remains provider-authored only; ACP still validates and guides retry instead of synthesizing markdown.
- 2026-07-05: Full DoD passed after the exact shard-completeness patch with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`. Coverage included contracts, Go tests, 229 Python batch/script tests, UI Vitest/typecheck/build and final Go build; Vite reported only existing large-chunk warnings.
- 2026-07-05: Diagnostic `smoke-tiny-bank-20260705T114209Z` ran selected `codex-code` after the exact shard-completeness patch. Result: strict `FAIL` with `runtime_contract_failed`, `artifact_quality_failed=0`, clean collect (`10/10` succeeded) and valid `step2.asis_docs`; remaining blocker is `init.step4.proposals` `draft_artifact_repair` exhaustion. Raw provider output did read staged findings and wrote substantive linked proposal/changelog, but validation failed because both files still omitted exact proposal shard completeness. Runtime classified this as terminal repair failure instead of routing to enrichment/cleanup, so broader trusted matrices remain blocked.
- 2026-07-05: Follow-up patch broadened draft recovery routing for recoverable semantic/shape draft failures: `runtime draft manifest outputs are invalid`, malformed markdown, missing exact shard completeness, missing proposal sections, missing finding linkage/actionability, stale structured-finding denial and process/downstream wording now route to provider-authored `draft_artifact_enrichment`; structural missing draft files still use draft repair first. The shard-status cleanup retry now also recognizes step4 `proposal shard completeness` and instructs provider to rewrite `proposal.md` and `changelog.md` with exact `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` plus `no-shard-coverage-blocker`, preserving finding IDs/actionability.
- 2026-07-05: Full DoD passed after the recovery-routing patch, then diagnostic `smoke-tiny-bank-20260705T123819Z` ran selected `codex-code` through `scripts/full-run-batch-matrix.sh`. Result: non-release `PASS`, `strict_pass_runs=1`, `strict_fail_runs=0`, `artifact_quality_failed_runs=0`, and no `artifact_quality.*` blockers. It is still not `Excellent`: live reported `runtime_quality.repair_exhausted=1`, `runtime_quality.repair_heavy=1`, and `runtime_quality.stall_pressure=1`; telemetry totals were `repair_attempts=4`, `focused_repairs=4`, `repair_exhausted=1`, `stall_count=5`, `pre_artifact_stalls=1`, `post_artifact_stalls=4`, and `valid_artifact_controlled_stops=5`. Step-level blockers: `refresh.step2.asis_docs` produced a manifest-first/pre-markdown path classified as `actual_stall`, `placeholder_or_scaffold`, `repair_attempts=3`; both `init.step4.proposals` and `refresh.step4.proposals` still needed one repair retry because first-pass `proposal.md` was missing validator-required substantive sections (`Decision / recommended operator action`, `Proposed changes or follow-up plan`, etc.). Broader trusted matrices remain blocked.
- 2026-07-05: Follow-up prompt hardening after `smoke-tiny-bank-20260705T123819Z`: normal `step2.asis_docs` now requires markdown targets to be written before the manifest and explicitly calls manifest-only first writes invalid; normal `step4.proposals` does the same for proposal/changelog before `proposals-draft-manifest.json` and names all validator-required proposal/changelog sections in the first-pass contract. This remains provider-authored only; ACP does not synthesize markdown.
- 2026-07-05: Full DoD passed after the manifest-last/required-sections prompt hardening with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`. Coverage included contracts, Go tests, 229 Python batch/script tests, UI Vitest/typecheck/build and final Go build; Vite reported only existing large-chunk warnings.
- 2026-07-05: Diagnostic `smoke-tiny-bank-20260705T141233Z` ran selected `codex-code` through `scripts/full-run-batch-matrix.sh` after the manifest-last/required-sections patch. Result: matrix `FAIL`, strict `FAIL`, `runtime_contract_status=failed`, `artifact_quality_failed=0`, `artifact_quality_status=needs_review`, and no product `artifact_quality.*` blockers. The run did not validate the new `step2`/`step4` happy path because init collect stopped partial: typed collect summary was `planned=10 succeeded=9 failed=1`, with `findings/as-is/proposals` skipped under `report_mode=incomplete`. The failed shard was the root-file shard `bank-of-anthos-gitignore-pylintrc-license-makefile-readme-md-mvnw-mvnw-cmd-pom-xml-a70095e6d2df`; raw stdout only contained Codex `thread.started` / `turn.started`, stderr was empty, no authored collect files were written, and runtime ended as `collect_pair_repair` `runtime_stalled_before_artifacts`. Live reports correctly exposed `init.step1.collect` as the only step-level `Excellent` blocker (`runtime_quality.repair_exhausted`, `runtime_quality.stall_pressure`, `final_validation_class=manifest_shape`). Broader trusted matrices remain blocked until this collect no-artifact path is fixed and Phase A reaches `Excellent`.
- 2026-07-06: Scoped the timeout fix for that Codex root-file collect failure: `codex-code` collect now uses a 5-minute initial and retry pre-artifact stall window, and shared collect-pair repair uses 5-minute pre/post/partial artifact windows. This does not weaken artifact-quality gates or synthesize artifacts; it only gives live Codex collect/recovery enough bounded time after transient stream reconnects. Phase A still requires a fresh `smoke tiny codex-code` rerun to prove `Excellent`.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T062358Z` proved the 5-minute window was applied and the root-file collect shard wrote artifacts; failure moved to manifest shape. Initial manifest missed `semantic.questions[2].text`; provider-authored `collect_manifest_repair` fixed questions but wrote repeated `citations[].id` for multiple repo files. Follow-up slice hardens normal/repair prompt self-checks, passes terminal validation error into manifest repair focus, adds one provider-authored `collect_manifest_shape_cleanup` retry for clean manifest-shape errors, and surfaces terminal validation excerpts in live reports.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T074548Z` ran selected `codex-code` from clean commit `efb849d` after the collect manifest shape-cleanup slice. Result: non-release matrix `PASS`, `strict_pass_runs=1`, `runtime_contract_status=passed`, `artifact_quality_status=passed`, `quality_gates_failed=0`, `artifact_quality_failed=0`, and no product `artifact_quality.*` signals. The collect blocker is fixed for this smoke: init and refresh collect both completed `10/10`, including the previously failing root-file shard. Verdict remains `Needs review`, not `Excellent`, because `init.step2.asis_docs` still used focused repair/enrichment: the normal provider turn missed `runtime/step2_as_is/asis-draft-manifest.json`, the repair pass then missed exact current-run shard completeness (`planned=10 succeeded=10 failed=0 incomplete=0`) and a decision-ready operator summary, and enrichment recovered the final artifacts. Broader trusted matrices remain blocked until this step2 first-pass repair path is removed.
- 2026-07-06: Started the `init.step2.asis_docs` first-pass completion slice. The target is not new validation policy or live acceptance; it is to make the normal provider turn mechanically finish the existing contract by writing `overview.md`, `summary.md`, `architect-summary.md`, and `asis-draft-manifest.json` in one bounded evidence-first command, with exact typed shard completeness and `no-shard-coverage-blocker` visible in the as-is markdown before any repair path is needed.
- 2026-07-06: Implemented the step2 first-pass prompt hardening. Normal `step2.asis_docs` prompt now includes a compact same-command write sequence with exact `write_root`/`draft_root`, all three markdown targets, manifest-last write and `test -s` checks for all four required files; as-is guidance now asks both `summary.md` and `architect-summary.md` to carry exact typed shard completeness plus `no-shard-coverage-blocker` when all current-run shards succeeded. Full DoD passed with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`; Vite reported only the existing large-chunk warning. Phase A still requires a clean-tree `smoke tiny codex-code` rerun before `claude-code` cross-check.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T094200Z` ran selected `codex-code` from clean commit `33c9c16`. Init collect completed `10/10`, and `init.step2.asis_docs` did create all four required targets in the first filesystem work unit, but provider-authored markdown lost path/index refs because backticked Markdown was written through shell-expanded text; validator accepted empty slots such as `Typed shard evidence checked:  and .`, so this slice exposed a product validation gap. `init.step4.proposals` then scheduled focused repair because first-pass `Top Actionable Findings` used `Finding ID: none` despite non-empty current-run findings. Since `Excellent` was already impossible after focused repair, the diagnostic matrix was stopped during refresh collect (`2/10` checkpointed) instead of waiting for a full non-Excellent report.
- 2026-07-06: Follow-up patch after `smoke-tiny-bank-20260706T094200Z` made normal step2 prompt shell-safe by requiring single-quoted heredocs or literal Python writes and self-checking for empty evidence slots. Shared draft validation now rejects empty evidence reference slots, and shared step4 validation rejects placeholder actionable Finding ID values such as `none` when current-run findings are non-empty. Targeted Go tests cover both live-shaped failures.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T104631Z` ran selected `codex-code` from clean commit `3fe91d2`. `init.step0.constitution` completed cleanly and `init.step1.collect` completed `10/10`, confirming the collect timeout/manifest-shape fixes remained effective. The run then hit `init.step2.asis_docs`: normal Codex output only contained `thread.started` / `turn.started`, no command execution, and the default 75s draft pre-artifact window terminated it as `runtime_stalled_before_artifacts`; focused repair was scheduled with `validation_error=.../runtime/step2_as_is/asis-draft-manifest.json: no such file or directory`, so `Excellent` was already impossible and the matrix was stopped.
- 2026-07-06: Follow-up patch after `smoke-tiny-bank-20260706T104631Z` aligns `codex-code` normal draft steps with the existing 180s Claude/Qwen draft pre-artifact window while preserving collect's 5-minute window and repair/excellent blockers. Normal `step2.asis_docs` prompt now explicitly says the first provider item must be command execution and forbids a preliminary assistant/status message before the filesystem command.
- 2026-07-06: Full DoD passed after the Codex draft pre-command stall fix with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`. Coverage included contracts, Go tests, 230 Python batch/script tests, UI Vitest/typecheck/build and final Go build; Vite reported only the existing large-chunk warning. The batch DoD timeout fixture was also made less timing-flaky by using a 2s fake-make timeout so child evidence is consistently written before timeout classification.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T114544Z` ran selected `codex-code` from clean commit `3330d86`. The run did not reach `step2`: `init.step1.collect` completed 4/10 shards, then five consecutive collect shards failed as `runner_unavailable` with Codex stdout `You've hit your usage limit. Visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at Jul 7th, 2026 5:43 AM.` The scheduler aborted the remaining shard after the repeated provider failure threshold. Matrix result was strict `FAIL`, `runtime_contract_status=failed`, `artifact_quality_failed=0`, `artifact_quality_status=needs_review`, `quality_gates_failed=1`, primary `failure_class=runner_unavailable`, and `partial_failure_count=6`. This is an operational provider quota blocker, not evidence against the step2 pre-command fix; Phase A remains blocked until Codex quota/auth is available and the clean-tree smoke can reach `step2`.
- 2026-07-07: Diagnostic `smoke-tiny-bank-20260707T053308Z` ran selected `codex-code` from clean commit `91000a7` after the Codex quota window. The `/tmp/provenarch-node-22.21.1/...` toolchain had a broken `npm` symlink, so the canonical smoke used the same exact Node `22.21.1` via `/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin`; harness precheck reported Node/npm ready and completed `make contracts test lint build`. Result: non-release matrix `PASS`, `strict_pass_runs=1`, `runtime_contract_status=passed`, `artifact_quality_status=passed`, `quality_gates_failed=0`, `artifact_quality_failed=0`, and no product `artifact_quality.*` signals. Init and refresh collect both completed `10/10`; refresh was clean (`repair_attempts=0`, `stall_count=0`). The run is still `Needs review`, not `Excellent`, because init quality emitted `runtime_quality.repair_heavy` and `runtime_quality.stall_pressure`: totals `repair_attempts=3`, `focused_repairs=3`, `stall_count=2`, `post_artifact_stalls=2`, `valid_artifact_controlled_stops=4`. Step-level blockers are now narrow and init-only: `init.step2.asis_docs` required two repair attempts plus one stall after first-pass `overview.md` claimed current-run final/citation indexes were unavailable instead of omitting downstream index status (`final_validation_class=manifest_shape`), and `init.step4.proposals` required one repair for low actionability because first-pass proposal did not link medium/high findings to recommended operator action and affected surface/path. Broader trusted matrices and `claude-code` cross-check remain blocked until Codex Phase A reaches `Excellent`.

---

## UX/UI Iteration: Live Shard Scope Mapping

### Context
- 2026-07-09 medium `regres long` diagnostic `regres-long-posthog-ftgo-20260709T094207Z` exposed a UI mapping gap after Slice 27.
- Real FTGO collect logs use `domain_id=ftgo-application` while selected-run artifacts are stored under `staging/shards/<shard_id>/...`.
- Analysis shard/log rows previously preferred `domain_id` for scope matching, so live rows could miss `Runtime only` / `Artifact pair present` state even when the artifact files were present.

### Plan
- [x] Prefer `fields.shard_id` and `staging/shards/<shard_id>` path segments for Analysis shard grouping and artifact-pair matching.
- [x] Keep same-shard runtime, repair and terminal log rows grouped even when later rows do not carry `taskrun_path`.
- [x] Preserve duration from the latest grouped row with duration fields.
- [x] Add a live-shaped UI regression where `domain_id` differs from `shard_id`.
- [x] Run full DoD with exact Node toolchain.
- [x] Commit the slice.

### Acceptance
- [x] A failed live-shaped shard with only `runtime-execution.json` renders as `Runtime only`.
- [x] A neighboring live-shaped shard with markdown + manifest renders as `Artifact pair present`.
- [x] The blocker drilldown keeps terminal repair copy while retaining the shard runtime record ref.
- [x] Full DoD passes: `make contracts`, `make test`, `make lint`, `make build`.

---

## UX/UI Iteration: Active Provider Stream Diagnostics

### Context
- 2026-07-09 medium `regres long` diagnostic `regres-long-posthog-ftgo-20260709T104135Z` produced two distinct live signals after Slice 28.
- `single-path/baseline` remained an operational host/provider preflight failure: `qwen headless probe timed out after 30s`.
- `single-git_url/baseline` reached FTGO qwen init collect, then the first shard failed `collect_pair_repair` with `runtime_stalled_before_artifacts`; before terminal evidence, the active run log was dominated by provider `runtime_output` JSON stream chunks.
- The UI already explains terminal artifact-handoff failures, but an active selected run needed a compact "provider stream active, artifact pair pending" state before a first-time operator opens raw stream diagnostics.

### Plan
- [x] Render Analysis live diagnostics for active selected runs when telemetry is loaded.
- [x] Summarize `runtime_output` JSON stream chunks into count, stream-event count and signal types.
- [x] Classify running collect telemetry with no authored markdown or `shard-pack-manifest.json` as provider stream / artifact pair pending.
- [x] Add a UI regression for active qwen-like stream output before authored shard artifacts exist.
- [x] Update UX report with live evidence and residual work.
- [x] Run full DoD with exact Node toolchain.
- [x] Commit the slice.

### Acceptance
- [x] Active Analysis shows `provider stream` diagnostics before terminal failure when provider output is streaming.
- [x] The diagnostic copy distinguishes provider chatter from durable collect progress by naming missing authored markdown plus `shard-pack-manifest.json`.
- [x] Next actions route raw stream inspection to stall/repair cases instead of asking the operator to read the full stream.
- [x] Full DoD passes: `make contracts`, `make test`, `make lint`, `make build`.

---

## UX/UI Iteration: Onboarding Recovery Rendered QA

### Context
- 2026-07-09 medium `regres long` diagnostic `regres-long-posthog-ftgo-20260709T162033Z` ran from clean commit `a71cc9d` with direct `scripts/full-run-batch-matrix.sh`.
- Matrix result: `FAIL`, non-release, `strict_pass_runs=0/2`.
- `single-path/baseline` stopped at operational host preflight because `qwen` headless probe timed out after 30s.
- `single-git_url/baseline` reached FTGO headless init collect, then failed as `runtime_contract_failed` with `partial_failure_count=5`, `repair_attempts=12`, `repair_exhausted=5`, `stall_count=18`, `pre_artifact_stalls=16`, `post_artifact_stalls=2`, `valid_artifact_controlled_stops=7`, and step-level Excellent blockers on `init.step1.collect`.
- Rendered QA already covers provider stream, failed shard, permissions, Ask, Publish Git and Source recovery. The first-time onboarding/recovery surface still lacks stable browser evidence.

### Plan
- [x] Add a mocked Playwright rendered QA scenario for onboarding source/runner recovery.
- [x] Cover duplicate source-name recovery, provider readiness failure guidance, recent workspace affordances, disabled first-analysis action, console-error absence and no horizontal overflow.
- [x] Fix any narrow rendered layout defects found during the QA pass without changing backend/API/schema/runtime contracts.
- [x] Update UX report with live evidence, implemented slice result and residual work.
- [x] Run rendered QA plus full DoD with the exact Node toolchain.
- [x] Commit the slice.

### Acceptance
- [x] A first-time operator can see what blocks onboarding and which setup step to fix next without entering Console V2.
- [x] Duplicate source diagnostics and provider command/auth/quota guidance remain readable on desktop and narrow mobile.
- [x] First analysis stays disabled until workspace, sources, runner and local readiness are ready.
- [x] Full DoD passes: `make contracts`, `make test`, `make lint`, `make build`.

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
| 19 Code quality remediation | done | Merged into `main` at `02716bb`; deterministic DoD and cleanup evidence are recorded in the active/archive plans and `docs/CODE_AUDIT_2026-07-10.md` |
| 20 Console UX trust and IA reset | done | 20A–20N deliver trustworthy contracts, four-destination shell, feature seams, semantic primitives and responsive task gates; Epic 18 trusted-machine release evidence remains separate |
| 21 Evidence-backed Architecture Home + impact-aware refresh | done (Wave 1) | Architecture Home, source revision/impact evidence, provider-free no-op, affected-only collect, surgical materialization and operator explanation are implemented |

---

## EP-20260710-code-quality-audit

### Context
- Baseline: commit `122e4c9b5a91b29e243677c0dac0fe2ebfca226b` with a clean starting worktree.
- Goal: evidence-backed quality audit of production Go/React code, deterministic tests, CI/tooling, contracts/spec drift and ordinary lifecycle/recovery behavior.
- Findings taxonomy: `DEAD`, `BUG`, `REF`, `QUAL`; only `Confirmed` and `High` confidence entries are reportable.

### Plan
- [x] Lock scope, finding contract, impact/priority/effort scales and excluded checks.
- [x] Run four isolated review streams: Static, Backend, UI and Tooling.
- [x] Execute allowed deterministic checks and classify environment limitations separately from code findings.
- [x] Deduplicate sanitized fragments without returning to source during synthesis.
- [x] Publish `docs/CODE_AUDIT_2026-07-10.md` with prioritized remediation roadmap and confirmed dead-code register.
- [x] Verify the final worktree changes only this plan and the audit report.

### Non-goals
- No production-code fixes, schema/API/contract changes or dependency upgrades.
- No live providers/matrices, fuzzing, dependency advisory scans or analysis outside the explicit quality scope.
- No raw logs, generated inputs or lower-confidence conjectures in the report.

### Acceptance
- Every finding has `file:line`, normal scenario, expected/actual, evidence, root cause, recommendation and acceptance test.
- Tool signals are manually validated; unsupported or host-limited checks are recorded as limitations, not findings.
- Final diff is documentation-only and contains no public-interface or production behavior change.

### Results
- Main register: 29 findings — 19 BUG, 3 REF, 7 QUAL; 19 Major/P1 and 10 Normal/P2; no Blocker.
- Dead-code register: 13 confirmed groups covering 60 Go/TypeScript/shell identifiers or assignments.
- Deterministic baseline passed: contracts, Go tests, race/shuffle, Python 230 tests, Vitest 93 tests, UI mock Playwright 7/7 and lint.
- Confirmed quality gaps include generated UI drift, ShellCheck coverage, missing V8 coverage dependency and missing failure-injection/request-order tests.
- No production code, API, schema or contract changes were made.
- Final worktree verification found only `docs/PLANS.md` and `docs/CODE_AUDIT_2026-07-10.md`.

---

## EP-20260715-console-trust-shell

### Context
- Epic 20 starts with contract truth before the Console IA cutover: selected-run evidence, Git inventory, runtime identity and run coordination must be authoritative.
- Epic 18 quality closure remains an independent release-readiness track and must not leak live-provider dependencies into required CI.

### Goals
- Deliver 20A–20I1 as small reviewable slices, ending with one path-based shell at `/setup`, `/home`, `/runs`, `/knowledge` and `/changes`.
- Close deterministic Epic 18 quality/contamination gaps and run the canonical trusted-machine live gate only after deterministic DoD.
- Keep API changes additive where compatibility permits and synchronize docs, TypeScript contracts, validators, fixtures and tests.

### Non-goals
- No 20F2 queue UI, 20I2 deep URL context, 20J–20N or Epic 21 implementation.
- No new runtime providers, hosted mode or security/compliance enforcement.
- No live network/provider dependency in required CI.

### Plan
- [x] 20A: make selected-run evidence an atomic, fail-closed `RunEvidenceSnapshot` with explicit source mode.
- [x] 20B1/20B2: publish complete Git inventory/identity/fingerprint and reject stale mutations before side effects.
- [x] 20C1/20C2: persist runtime identity and separate effective runtime from restart-only desired settings after Console entry.
- [x] 20F1: make ordinary starts fail with typed `run_active`; expose active/pending coordination and typed supersession.
- [x] 20D/20G/20H/20E: add demo identity, safe evidence rendering, accessible primitives and one workflow selector.
- [x] 20I1: atomically replace StageRail with the five-destination native History API shell.
- [x] Epic 18: synchronize quality contracts, remove contamination/actionability defects and add deterministic fixtures.
- [x] Run `make contracts`, `make test`, `make lint`, `make build` and `npm run e2e:mock --prefix ui`.
- [ ] Perform trusted-host preflight; run direct canonical matrix commands only when all prerequisites pass.

### Acceptance
- Selected-run Review/Publish never falls back to current workspace content.
- Git confirmation covers the exact full-workspace mutation set and stale confirmation has no side effects.
- Historical run identity never derives from current UI selection; active/pending coordination is public and typed.
- Only the new shell is interactive after 20I1, with direct URL and Back/Forward coverage.
- Required CI remains deterministic; release readiness is reported only from PASS verdicts plus accepted SWE UX and artifact-quality assessments.

### Results
- The agreed Epic 20 foundation scope through 20I1 is complete: snapshot evidence is fail-closed, Git mutations are fingerprint-confirmed, runtime/coordination state is server-authored, and the legacy StageRail shell is absent from the product DOM. Epic 20 as a whole remains open.
- Epic 18 R1/R2 are complete with promoted-evidence-first assessment language, contamination fixes and regression fixtures.
- Deterministic DoD passed on 2026-07-15: contracts, full Go suite, 246 Python tests, 127 Vitest tests, ShellCheck/typecheck, production build and 7/7 Playwright mock scenarios with critical axe checks.
- Production dependency audit reports zero vulnerabilities after pinning the Mermaid transitive DOMPurify dependency to `3.4.12`.
- R3 is intentionally not executed from this worktree: the canonical trusted-machine gate requires reviewed committed inputs, all three release providers in `PATH`, direct matrix invocation and accepted SWE assessment reports.
- 2026-07-15 Epic 21 preparation audit: exact Go `1.25.10` and Node `22.21.1` toolchains are available; `qwen 0.17.1`, `claude 2.1.85` and `codex-cli 0.144.1` are in `PATH`; `/tmp` is writable with more than the 5 GiB minimum. The current tree is intentionally dirty during implementation and `/tmp/provenarch-live-e2e` canonical pinned checkouts are not provisioned, so smoke/release execution remains deferred until a clean merged commit on a prepared trusted host.

---

## EP-20260715-20F2-deliberate-queue-ui

### Goal
- Consume authoritative run coordination in Runs and make queueing a confirmed refresh-only action without losing last-good evidence.

### Non-goals
- No multi-item queue, init queueing, scheduling policy changes or Home/Run Studio recomposition.

### Plan
- [x] Persist `coordination` in the frontend run model and send explicit `start|queue` intent.
- [x] Disable ordinary starts during an active run; confirm queue creation/replacement and expose pending cancellation.
- [x] Preserve selected evidence until an ordinary start is accepted and refresh coordination after mutations.
- [x] Pass focused tests and full deterministic DoD.

### Acceptance
- A stale click cannot silently enqueue; the dialog names active and replaced pending identities.
- Queue failure or success does not replace the selected last-good snapshot.

### Results
- Full deterministic DoD passed on 2026-07-15: contracts, Go, 246 Python tests, 128 Vitest tests, lint/typecheck, production build and 7/7 mock Playwright scenarios.

---

## EP-20260715-20I2-deep-url-context

### Goal
- Make route, setup step, run/source selection, artifact/entity identity and viewer mode reloadable and navigable through native History API URLs.

### Non-goals
- No router dependency, persisted draft contract, queue UI expansion or product-page recomposition from 20J.

### Plan
- [x] Add a typed codec for Setup, Runs, Knowledge and Changes deep context.
- [x] Restore run and artifact identity without cross-run/current-workspace fallback; canonicalize defaults and sanitize stale context with a visible notice.
- [x] Preserve viewer mode through reload/Back/Forward and warn before leaving an unsaved Setup/editor draft.
- [x] Cover nested SPA fallback paths and focused URL-state scenarios.
- [x] Pass full deterministic DoD and mock E2E.

### Acceptance
- Direct GET, reload and Back/Forward recover the same valid product context.
- An explicitly stale run or artifact is removed from the URL with an explanation and never replaced silently by another source.
- User navigation uses `pushState`; only defaults, invalid context and canonical paths use `replaceState`.

### Results
- Full deterministic DoD passed on 2026-07-15: contracts, full Go suite, 246 Python tests, 132 Vitest tests, ShellCheck/typecheck, production build and 7/7 mock Playwright scenarios with critical axe checks.
- Run recovery navigation updates URL identity before async evidence loading, removing a race where the old deep link could reselect the failed run.

---

## EP-20260715-20J1-home-guided-setup-runs

### Goal
- Recompose Home, Guided Setup and Runs around the shared workflow selector and authoritative workspace/run state without a broad 20M refactor.

### Non-goals
- No Knowledge API, Changes package composition, global Ask modal, persisted contract additions or QA runs in primary history.

### Plan
- [x] Extract dedicated Home, Guided Setup and Runs page containers from the temporary `App.tsx` composition.
- [x] Give Home four authoritative axes and exactly one selector-derived primary action.
- [x] Implement the five-step Guided Setup and persist the brief through the existing Step 0 contract, with an explicit quality-warning confirmation when skipped.
- [x] Separate Run Studio context/history from a local Diagnostics disclosure and keep coordination/recovery evidence visible.
- [x] Pass focused tests, full deterministic DoD and mock E2E.

### Acceptance
- Home never derives a second competing next action outside the shared selector.
- A saved non-empty project name and scope satisfy brief readiness; an unsaved brief can only be skipped through explicit confirmation.
- Runs shows server-authored active/pending and historical runtime identity while diagnostic telemetry remains non-authoritative.

### Results
- Full deterministic DoD passed on 2026-07-15: contracts, full Go suite, 246 Python tests, 133 Vitest tests, ShellCheck/typecheck, production build and 7/7 mock Playwright scenarios with critical axe checks.
- Mock browser coverage now explicitly opens the local Runs Diagnostics disclosure before asserting shard telemetry; workflow recovery remains visible outside it.

---

## EP-20260715-20J2-changes-knowledge-ask

### Goal
- Complete the primary product composition with historical Change Review packages, an authoritative current-workspace Knowledge read model and global read-only Ask context.

### Non-goals
- No persisted schema changes, filename-derived topology, run-scoped Ask, 20K–20N refactor/responsive budget or live network dependency.

### Plan
- [x] Add read-only `GET /api/knowledge` with validated entity/edge parsing, artifact inventory and typed partial/unavailable issues.
- [x] Compose Changes around review packages and six URL-backed modes while routing non-reviewable runs back to Run Studio.
- [x] Build Knowledge Overview/Atlas/Entities/Artifacts exclusively from current promoted workspace data with a searchable keyboard-accessible fallback.
- [x] Wrap Ask as a focus-managed global dialog; open citations in the shared current-workspace Evidence Viewer and preserve return context.
- [x] Synchronize API spec, appendix, examples/fixtures, ADR and stakeholder architecture docs; pass focused tests and full deterministic DoD.

### Acceptance
- A malformed entity or broken edge produces `partial` without hiding other valid knowledge; an empty workspace produces `unavailable`.
- Atlas topology is based only on validated entity/edge fields and remains usable as a table without pointer input.
- Historical Changes never claims a known publication status for a run, and current-workspace evidence never silently falls back to a run snapshot.
- Ask owns focus while open, closes with Escape, returns focus, and citation drilldown can return to the original Ask/route context.

### Results
- `GET /api/knowledge` now exposes current-workspace validated entities/edges, readable artifact inventory and deterministic `available|partial|unavailable` state without inferred promotion identity.
- Changes admits only successful `init|refresh` runs with their own authoritative final index as Change Review packages; other analysis outcomes route to Run Studio and QA remains Ask history.
- Global Ask is a focus-managed read-only dialog; citations open current-workspace Evidence Viewer and preserve an explicit Return to Ask route.
- Full deterministic DoD passed on 2026-07-15: contracts, full Go suite, 246 Python tests, 141 Vitest tests, ShellCheck/typecheck, production build and 7/7 mock Playwright scenarios with critical axe checks. Ask was rendered without horizontal overflow at 1440, 1280, 1024 and 390 px.

---

Completed Epic 20 exit plans `20M`, `20K`, `20L` and `20N` are archived in
`docs/archive/PLANS_ARCHIVE_2026-07.md`.

---

## EP-20260715-21D-explainable-no-op

### Goal
- Turn safe `unchanged_candidate` plans into successful provider-free refreshes without canonical rewrites.

### Plan
- [x] Persist factual refresh execution audit and additive run summary.
- [x] Fail closed for every planner fallback and leave validator-promoted baseline selection unchanged.
- [x] Surface no-op identity in CLI and Runs and cover schema/orchestrator behavior.

## EP-20260715-21E-affected-collect

### Goal
- Execute only affected collect shards when all unaffected baseline packs can be replayed safely.

### Plan
- [x] Preflight and copy baseline packs into current taskrun staging before provider execution.
- [x] Reuse the existing checkpoint parser/apply path and preserve deterministic scheduling semantics.
- [x] Fall back to full execution on missing packs or shard-plan mismatch and provide bounded Git intent context.

## EP-20260715-21F-surgical-materialization

### Goal
- Make downstream publication decisions explicit and validator-gated.

### Plan
- [x] Add schema-validated materialization decisions with provenance and digest.
- [x] Record byte-identical no-op preservation and full-refresh rebuild decisions.
- [x] Keep final/citation validation and atomic promotion as the only canonical mutation boundary.

## EP-20260715-21G-operator-explanation

### Goal
- Explain refresh behavior in existing Runs and Changes surfaces and synchronize product documentation.

### Plan
- [x] Show no-op, affected-only, full and legacy-unavailable states without a new product stage.
- [x] Synchronize README, architecture, specs, appendix, stakeholder and ADR sources.
- [x] Close Epic 21 after complete deterministic DoD and mock E2E gates.

### Results
- Full deterministic DoD passed on 2026-07-15: contracts, complete Go suite, 246 Python tests, 141 Vitest tests, lint/typecheck, embedded production build and 7/7 mock Playwright scenarios.
- Epic 18 R3 remains the release-readiness gate. The implementation is committed/merged and
  canonical `/tmp/provenarch-live-e2e` pinned checkouts are bootstrapped; current progress and
  operational blockers are recorded in `EP-20260715-epic18-r3-composite-release`.

---

## EP-20260715-epic18-r3-composite-release

### Context
- Epic 21 is complete at `b95e72f`; Epic 18 R3 is the remaining release-readiness gate.
- `release full` is three fresh constituent matrices, while the tag workflow currently accepts only
  one verdict path. The trusted host has provider CLIs, writable `/tmp` and sufficient disk, but its
  canonical pinned path checkouts are not bootstrapped.

### Goals
- [x] Add a backward-compatible composite verifier/workflow interface through
  `ACP_RELEASE_MATRIX_IDS` and reject ambiguous or duplicate configuration.
- [x] Reconcile Epic 21 and Epic 18 status across planning, stakeholder, testing and runbook docs.
- [x] Merge the composite-gate slice after full deterministic DoD.
- [x] Bootstrap and verify canonical pinned checkouts on the trusted host.
- [ ] Run direct Codex `smoke tiny`, then standalone release fast/long, then fresh release-full
  fast/long/ftgo-sentry constituents; stop on the first product/provider/host failure.
- [ ] Persist only bounded final release evidence, verify every constituent plus the composite, and
  close Epic 18 only after all verdicts are `PASS` with accepted SWE UX/artifact assessments.

### Non-goals
- No HTTP API, persisted product schema, workspace contract, provider contract or canonical matrix changes.
- No timeout overrides, wrapper scripts, hosted live workflow, release tag or new Wave 1 product epic.
- No commit of raw taskrun directories, provider logs or temporary canonical checkouts.

### Acceptance
- Single-matrix release settings remain compatible; exactly one release-evidence selection mode is required.
- Any missing/failed/duplicate constituent or missing/rejected companion assessment blocks GoReleaser.
- R3 uses only direct `scripts/full-run-batch-matrix.sh` commands from a clean merged commit.
- Release readiness requires exact providers, run index 1, baseline/parallel shard-plan equality,
  strict zero-failure, frontend evidence and three accepted full-release evidence packages.
- Each code/docs PR passes `make contracts`, `make test`, `make lint`, `make build`.

### Progress log
- 2026-07-15: Confirmed exact Go/Node toolchains and qwen/claude/codex CLIs; `/tmp` is writable with
  sufficient disk. Canonical `/tmp/provenarch-live-e2e` checkouts remain the preflight blocker.
- 2026-07-15: Implemented multi-path verifier input, duplicate matrix rejection, mutually exclusive
  workflow settings and focused deterministic tests; synchronized stale Epic 21 tracker language.
- 2026-07-15: Merged the composite gate as `89b4bb49`, bootstrapped and verified all canonical
  pinned checkouts, exact Go/Node/npm toolchains and three release provider CLIs. Direct Codex
  `smoke tiny` matrix `smoke-tiny-bank-20260715T172229Z` stopped before provider execution because
  the live backend-cycle helper treated a successful Epic 21 `no_op` refresh as missing telemetry.
  Opened a bounded remediation: accept the absent legacy quality summary only when the matching
  run-scoped `refresh-execution.json` proves `unchanged_candidate`, `no_op` and skipped providers;
  every missing, malformed or mismatched audit remains a gate failure.
- 2026-07-16: No-op harness remediation merged at `e9ee719c`; Codex smoke
  `smoke-tiny-bank-20260715T181251Z` passed. Standalone matrix
  `release-fast-20260716T043815Z` was stopped on the first live product blocker:
  `qwen-code` shard `bank-of-anthos-extras` failed with `runtime_contract_failed` after focused
  `collect_pair_repair` emitted active stream telemetry for the full pre-artifact wall-clock but
  wrote no authored files. Follow-up `EP-20260716-epic18-r3-qwen-stream-repair` is the only active
  remediation before repeating smoke and release acceptance from a new clean commit.
- 2026-07-16: Qwen stream-only remediation merged at `cae4653a`. The required fresh Codex smoke
  `smoke-tiny-bank-20260716T053113Z` then completed all 10 collect shards but stopped at
  `init.step2.asis_docs`: focused enrichment replaced bootstrap content with the pre-21A generic
  overview headings, so strict Architecture Home validation rejected every required section.
  `EP-20260716-epic18-r3-step2-home-repair` is the active bounded remediation; no release matrix
  or accepted evidence follows until it is merged and smoke passes from the new clean commit.
- 2026-07-16: Step2 Architecture Home remediation merged at `435c2ac8`; fresh Codex smoke
  `smoke-tiny-bank-20260716T065516Z` passed with strict failures `0`. Standalone release-fast
  `release-fast-20260716T085301Z` then stopped on the first terminal blocker: Qwen shard
  `bank-of-anthos-extras` exhausted the allowed stream-only collect-pair retry after ~2.49 MiB of
  thinking telemetry and zero filesystem actions. `EP-20260716-epic18-r3-qwen-tool-first-retry`
  is the only active remediation; release long/full and accepted evidence remain blocked.
- 2026-07-16: Qwen tool-call-first remediation merged at `4eddf559`. The first post-merge Codex
  smoke `smoke-tiny-bank-20260716T095549Z` stopped at refresh `9/10` with
  `runner_unavailable` after Codex exhausted WebSocket TLS reconnects and HTTP fallback; raw
  evidence classified this as an external provider transport incident, so runtime retry policy
  was not expanded.
- 2026-07-16: Requalification smoke `smoke-tiny-bank-20260716T122802Z` ran from the same clean
  `4eddf559` and passed with `strict_pass_runs=1`, `strict_fail_runs=0`, init/refresh collect
  `10/10`, no runtime/provider/contract/flow blockers and no partial failures. Standalone
  `release-fast-20260716T142658Z` then stopped before product execution:
  Qwen's bounded read-only `ACP_READY` probe timed out after 30 seconds. A separate bounded
  readiness recheck reproduced `headless_probe_timeout`; Claude and Codex readiness passed.
  This is an `operational_host_preflight_failed` provider blocker. Release long/full, SWE
  assessments, bounded evidence PR and Epic 18 closure remain blocked until Qwen readiness is
  restored on this or another trusted host.
- 2026-07-16: Qwen artifact-readiness remediation merged as `57155786`. Fresh Codex smoke
  `smoke-tiny-bank-20260716T165233Z` passed with `strict_pass_runs=1`, `strict_fail_runs=0`,
  init/refresh collect `10/10` and no execution blocker. Standalone
  `release-fast-20260716T185527Z` then stopped on the first profile before product execution:
  `qwen --version` passed, but the single canonical runtime-like artifact smoke exited without
  creating the exact sentinel. The batch correctly materialized
  `operational_host_preflight_failed`; the automatically started next sweep was interrupted, and
  release long/full plus accepted evidence remain blocked. No retry, timeout-success exception or
  product-runtime change is justified by this provider/host readiness result.

---

## EP-20260717-epic18-r3-ui-route-preflight

### Context
- The clean `ba98798a` deterministic R3 preflight passed contracts, Go and Python tests but the
  full UI suite exposed a route race in the provider-unavailable recovery test.
- The test opened Changes while async run canonicalization was still settling, navigated to Runs,
  and could be returned to Changes before `run-status-panel` rendered. The focused test passed
  `3/3`; production behavior and provider/runtime policy are unchanged.

### Goals
- Start the recovery scenario directly on its stable `/runs/<run_id>` deep link.
- Preserve the existing Runs recovery, Changes publication gate and Setup readiness assertions.
- Pass the full deterministic DoD before restarting R3 from the merged clean commit.

### Non-goals
- No production UI, API, runtime, schema, canonical matrix or timeout change.
- No live provider execution from the remediation branch.

---

## EP-20260716-epic18-r3-qwen-artifact-readiness

### Context
- Post-requalification standalone `release fast` stopped before product execution because Qwen's
  bounded read-only `ACP_READY` probe timed out twice.
- The Qwen runtime uses an artifact-capable command surface with filesystem writes, and preflight
  already has a runtime-like sentinel smoke for that surface.

### Goals
- [x] Remove only the text-only Qwen `ACP_READY` probe while preserving the Codex probe.
- [x] Require successful `qwen --version` and one runtime-like artifact-smoke attempt with
  `--chat-recording false`, `--yolo`, `--channel CI` and an exact sentinel.
- [x] Keep the canonical `120s` budget and strict blockers for auth/quota, non-zero exit, timeout,
  missing sentinel and invalid sentinel.
- [x] Cover process-group cleanup so a timed-out Qwen smoke cannot leave provider children.
- [x] Synchronize runbook, testing strategy and Epic 18 tracker status.
- [x] Pass focused tests and complete deterministic DoD.
- [x] Merge the remediation, then restart R3 from direct Codex smoke on the new clean commit.

### Non-goals
- No runtime/provider contract, HTTP API, schema, workspace, canonical matrix or timeout change.
- No timeout-success exception, retry expansion, wrapper script or live result claim in this PR.
- No Epic 18 closure before fresh standalone and composite release evidence passes.

### Acceptance
- A Qwen text-only invocation may hang without affecting readiness when the runtime-like artifact
  smoke writes the exact sentinel and exits successfully.
- Missing binary, auth/quota marker, non-zero command, timeout without sentinel and invalid
  sentinel remain blockers before product runtime and release verdict execution.
- Qwen artifact smoke runs once with the canonical `120s` budget and full process-group cleanup.
- `make contracts`, `make test`, `make lint`, and `make build` pass.

### Progress log
- 2026-07-16: Qwen readiness now skips the unstable text-only probe and requires the existing
  runtime-like artifact smoke. Focused readiness tests passed `32/32`, batch preflight/failure
  classification passed `94/94`, and the complete deterministic DoD passed with 259 Python tests,
  141 UI tests, the full Go suite, lint/typecheck and the embedded production build.
- 2026-07-16: Merged as `57155786`. Codex smoke `smoke-tiny-bank-20260716T165233Z` passed from the
  clean merge commit. Standalone `release-fast-20260716T185527Z` proved the new fail-closed
  contract: Qwen version detection succeeded, but the only artifact-smoke attempt produced no
  sentinel, so preflight stopped as `operational_host_preflight_failed` before any release runtime.
  Epic 18 remains open pending provider readiness and a fresh complete R3 sequence.

---

## EP-20260716-epic18-r3-qwen-stream-repair

### Context
- Canonical standalone `release fast` reached Qwen live collect after clean host/provider/DoD
  preflight, then failed one Bank of Anthos shard before any accepted release evidence existed.
- Raw lifecycle evidence showed a bounded 180-second pre-artifact wall-clock, about 4.1 MiB of
  stream diagnostics, zero authored files and terminal `runtime_stalled_before_artifacts`.
- Silent/no-fresh and transient-provider repair retries already exist; active stream-only repair
  stalls currently skip the compact retry path even though a stall-focused compact prompt exists.

### Goals
- [x] Add a Qwen-only one-shot retry for focused collect-pair pre-artifact stalls that emitted
  stream diagnostics but produced no authored files.
- [x] Build that retry with the existing compact `runtime_stalled_before_artifacts` write-first
  prompt; do not rerun the same broad repair prompt.
- [x] Keep the first wall-clock cap, write-set validation and terminal second-stall behavior.
- [x] Add focused engine, Qwen policy and compact-prompt size regressions.
- [x] Pass full deterministic DoD; commit/merge the remediation, then repeat Codex smoke before
  restarting standalone release fast.

### Non-goals
- No timeout override, canonical matrix/curated repo change, deterministic artifact synthesis,
  additional provider, schema/API/workspace contract change or release evidence acceptance.
- No retry after a partial authored write; existing validation and manifest-recovery paths remain
  authoritative for partial/invalid artifacts.

### Acceptance
- Stream-only Qwen repair stalls receive exactly one compact retry and can succeed only through
  provider-authored validated markdown plus `shard-pack-manifest.json`.
- Silent exhaustion keeps its `runner_unavailable` classification; repeated stream-only exhaustion
  remains `runtime_contract_failed`.
- `make contracts`, `make test`, `make lint`, `make build` pass and the tracked embedded UI bundle
  remains deterministic.

### Progress log
- 2026-07-16: Focused providercommon/Qwen/prompt-contract tests and full deterministic DoD passed
  with exact Go `1.25.10` and Node `22.21.1`; tracked embedded UI output remained unchanged.

---

## EP-20260716-epic18-r3-step2-home-repair

### Context
- The post-remediation Codex smoke passed collect with 10/10 shards and failed in focused
  `step2.asis_docs` draft enrichment.
- Strict validation correctly required the 21A Architecture Home sections, but the focused,
  compact and command-text enrichment prompts still described only the older generic architecture
  surface/coverage shape.
- The provider therefore wrote fresh evidence-backed markdown with `Architecture Surface`,
  `Evidence Used` and `Coverage Gaps`; this was a product prompt-contract drift, not a host blocker.

### Goals
- [x] Require the exact Architecture Home section set in every step2 draft enrichment prompt mode.
- [x] Explicitly reject generic substitute headings in prompt guidance without weakening validation.
- [x] Add focused prompt-contract regressions for normal, compact and command-text retry modes.
- [x] Pass full deterministic DoD, merge the remediation, then repeat Codex smoke from the new
  clean commit before restarting standalone release fast.

### Non-goals
- No validator relaxation, deterministic ACP-authored Architecture Home, timeout override, matrix
  or curated repo change, schema/API/workspace contract change, or accepted release evidence.
- No change to step2 evidence scope, write-set, retry count or provider transport.

### Acceptance
- Normal, compact and command-text step2 enrichment prompts require non-empty `System at a glance`,
  `Analyzed scope`, `Domains and ownership`, `Key flows`, `Integrations and datastores`,
  `Where to start`, `Safe-change guidance`, and `Evidence gaps and open questions` sections.
- Existing strict runtime draft validation remains authoritative and focused tests plus
  `make contracts`, `make test`, `make lint`, `make build` pass.

### Progress log
- 2026-07-16: Root cause reproduced from bounded smoke evidence; prompt contract and focused tests
  updated for all three step2 enrichment modes.
- 2026-07-16: Full deterministic DoD passed with exact Go `1.25.10`, Node `22.21.1` and npm
  `10.9.4`; contracts, Go, 256 Python, 141 UI, lint/typecheck and embedded UI build are green.

---

## EP-20260716-epic18-r3-qwen-tool-first-retry

### Context
- Post-step2-remediation Codex smoke passed, so standalone release fast restarted from clean
  commit `435c2ac8` with canonical host/toolchain/provider prerequisites.
- Qwen collect pair recovery succeeded for two Bank of Anthos shards only after its allowed retry,
  then shard `bank-of-anthos-extras` exhausted the retry: 180 seconds, ~2.49 MiB stream-thinking,
  zero authored files and no filesystem tool call.
- The retry already used the compact path, but its 11.9 KiB schema/checklist surface still let the
  model reason about the full manifest until the wall-clock cap.

### Goals
- [x] Mark the stream-only retry explicitly so prompt composition can distinguish it from the
  first compact collect-pair repair attempt.
- [x] For Qwen only, replace that retry prompt with an ultra-compact tool-call-first contract under
  4 KiB: `run_shell_command` first, four evidence candidates, minimal canonical manifest shape.
- [x] Preserve the single retry, existing wall-clock/write-set/validation gates and terminal
  repeated-stall classification.
- [x] Add prompt-size/content and engine marker regressions; synchronize architecture, testing and
  live-runbook contracts.
- [x] Pass full deterministic DoD; merge the remediation, then repeat Codex smoke and standalone
  release fast from the new clean commit.

### Non-goals
- No timeout increase/override, additional retry, deterministic ACP-authored evidence, canonical
  matrix/curated repo change, schema/API/workspace/provider contract change or accepted evidence.
- No relaxation of strict collect validation or write-set enforcement.

### Acceptance
- The second Qwen stream-only attempt receives the tool-call-first prompt and no bulky canonical
  shape/checklist/skeleton sections.
- Success still requires provider-authored markdown plus manifest to pass existing validation;
  repeated stream-only exhaustion remains `runtime_contract_failed`.
- `make contracts`, `make test`, `make lint`, `make build` pass with the exact pinned toolchains.

### Progress log
- 2026-07-16: Focused prompt/providercommon regressions pass. Full deterministic DoD passed with
  Go `1.25.10`, Node `22.21.1` and npm `10.9.4`: contracts, Go, 256 Python, 141 UI,
  lint/typecheck and tracked embedded UI build are green.
- 2026-07-16: Merged as `4eddf559`. Post-merge Codex requalification eventually passed as
  `smoke-tiny-bank-20260716T122802Z`; the following standalone release-fast was blocked in
  provider readiness before runtime execution because Qwen twice timed out on the bounded
  headless probe. No further product remediation is inferred from this operational blocker.

---

## EP-20260710-code-audit-remediation-backlog

### Context
- The completed audit at `docs/CODE_AUDIT_2026-07-10.md` identified 29 reportable findings and 13 confirmed dead-code groups.
- The project remains a local-first MVP with a loopback/trusted-operator React UI embedded into the Go binary.
- Security/compliance enforcement and hosted/multi-user frontend hardening remain Wave 1+.

### Goals
- Convert every audit finding into a small, dependency-ordered implementation slice.
- Make all Major/P1 remediation release-blocking while keeping required CI deterministic and provider-free.
- Synchronize the reference backlog, canonical stakeholder matrix and operational mirror.

### Non-goals
- No production-code, API, schema or contract implementation in this planning slice.
- No frontend auth/authz, CSP/CORS hardening program, hosted exposure or security policy work.
- No live provider/matrix execution.

### Plan
- [x] Add Epic 19 goals, scope boundary, priority phases and epic-level acceptance to `docs/BACKLOG.md`.
- [x] Map `BUG-001..019`, `REF-001..003`, `QUAL-001..007` and `DEAD-001..013` to reviewable slices.
- [x] Record dependencies, affected modules, deterministic tests, docs obligations and DoD for every slice.
- [x] Keep UI remediation limited to correctness, accessibility and deterministic QA.
- [x] Synchronize `docs/STAKEHOLDER_DOC.md` and the operational mirror.

### Acceptance
- Every audit ID appears in exactly one primary Epic 19 slice; related IDs may be referenced as dependencies only.
- P1 ordering prevents promotion/lifecycle/schema/UI/release work from racing incompatible foundations.
- Schema-changing slices explicitly require schema-guardian, fixtures and documentation synchronization.
- P3 deletion slices run only after behavior-restoration decisions that could reuse dead code.
- Existing local-first and Wave 1+ security boundaries remain unchanged.

### Results
- Epic 19 contains 38 PR-sized slices/sub-slices: 16 P1, 12 P2 (including one optional
  V8-coverage gap slice) and 10 P3.
- P1 is grouped into crash/lifecycle, contract/source, UI correctness and release reproducibility phases.
- Frontend security hardening was deliberately excluded; local UI work covers stale state, data loss, accessibility and deterministic browser QA.
- Documentation verification passed: `git diff --check`, exact one-to-one audit-ID mapping and
  slice-count checks. Full DoD was not repeated because this slice changes no code,
  schemas/spec contracts or fixtures; the same exact code baseline DoD is recorded as passing
  in the immediately preceding audit ExecPlan.
