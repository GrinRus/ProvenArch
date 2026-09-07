# PLANS_ARCHIVE_2026-07.md

Closed ExecPlans archived from `docs/PLANS.md` in July 2026.

This archive preserves implementation-complete plan evidence; residual trusted-host, live-quality, owner/admin and owner-decision workstreams remain active in `docs/PLANS.md`.

### Plan ID
EP-20260721-epic18-valid-artifact-stall-accounting

### Result
- PR #170 merged at `a99f3099` and made paired `retry scheduled / terminate_and_validate` with
  `manifest_state=valid` part of the single valid-artifact controlled-stop lifecycle.
- Invalid/missing artifacts remain actual stall pressure. Exact valid/invalid provider-free
  regressions passed 20 repetitions and the full deterministic DoD passed.
- Clean qualification `smoke-tiny-bank-20260721T004350Z` confirmed the accounting fix: five valid
  controlled stops were no longer counted as stalls; only two actual invalid/repair stalls remained.

---

### Plan ID
EP-20260728-async-shutdown-quiescence

### Result
- PR #180 backend CI exposed two async workspace-cleanup races after terminal status publication:
  `TestGitCommitBlockedWhileRunIsActive` and
  `TestAsyncRunPanicReleasesSlotAndStartsPendingRun`.
- `Service.Shutdown` now waits for terminal state plus release of the selected run's active/cancel
  ownership; the package-local lifecycle helper observes the same real quiescence condition.
- Both live-observed tests passed 50 consecutive repetitions. Full pinned deterministic DoD and
  `make offline-closure` passed, including API/orchestrator race suites, 263 Python tests, 158 UI
  tests, 7/7 mock E2E and deterministic embedded `ui_dist`.
- PR #181 passed all 11 required checks and merged as
  `cb126be6cdfaf7766d88694ed80a6a2d72c845dc`. The remote branch remains available during R3.

---

### Plan ID
EP-20260728-r3-qwen-atomic-pair-write

### Result
- PR #179 required Qwen to compose both final payloads and issue markdown plus manifest
  `write_file` calls in the same assistant response, with 3000/6000-character payload limits and a
  bounded semantic subset.
- Provider-free prompt/adapter tests preserved the exact provenance enum, refresh minima,
  Claude/Codex isolation and the 6 KiB Qwen prompt limit.
- Full DoD/offline closure and all 11 PR checks passed; PR #179 merged as
  `0fe8793ce179e668b2cb5cd43a51ac1cc09500d6`, and the merge SHA passed fresh detached-worktree
  `make offline-closure`.
- Fresh smoke `smoke-tiny-bank-20260728T111000Z` confirmed the atomic pair write, then exposed
  scalar `repo_scopes`/`path_scopes` in one Qwen manifest. The runtime scheduled manifest-only
  repair, so the matrix was stopped and no partial evidence was promoted.
- Follow-up is `EP-20260728-r3-qwen-scope-array-contract`.

---

### Plan ID
EP-20260720-epic18-step2-first-pass-marker-free

### Context
The post-PR #167 qualification `smoke-tiny-bank-20260720T172737Z` completed init/refresh collect
`10/10`, strict validation and the ProductShell frontend flow with machine `PASS`. Release-quality
acceptance still stopped because `init.step2.asis_docs` needed three focused repairs and one actual
stall. The first provider work unit authored useful content, but `architect-summary.md` described a
"bounded evidence read" and the following attempt put runtime/process narration in Architecture
Home. These are ordinary operator-document contract defects, independent of live matrix identity.

### Goals (must have)
- [x] Make the normal step2 first work unit scan all three authored Markdown targets for the exact
      runtime/recovery marker classes before writing `asis-draft-manifest.json` last.
- [x] Require offending sentences to be rewritten as architecture facts, coverage gaps or operator
      decisions in the same provider command, without relying on focused repair.
- [x] Add a reduced provider-free authored fixture and prompt/validator regressions.
- [x] Synchronize runtime/testing/live-gate documentation and pass full deterministic DoD.
- [x] Merge the remediation and restart R3 from the new clean merge commit.

### Non-goals
- [x] Do not sanitize, synthesize or manually repair provider-authored Markdown.
- [x] Do not change schemas, HTTP APIs, provider contracts, timeouts or canonical matrices.
- [x] Do not expose matrix IDs, verdicts, assessments or live-only environment to product code.

### Acceptance criteria
- [x] The first normal step2 command checks `overview.md`, `summary.md` and
      `architect-summary.md`, not only Architecture Home, before manifest completion.
- [x] Live-observed `bounded evidence read` / runtime-assembly narration is rejected by a
      provider-free fixture while equivalent operator-facing architecture language remains valid.
- [x] Focused tests and `make contracts`, `make test`, `make lint`, `make build` pass.

### Progress log
- 2026-07-20: Stopped before standalone release-fast despite machine `PASS`; final matrix evidence
  reported `runtime_quality.repair_heavy=1` and `runtime_quality.stall_pressure=1`, both isolated to
  init step2 first-pass runtime/recovery narration. Main and the pinned Bank checkout remained clean.
- 2026-07-20: Added the same-command three-target marker scan, a reduced provider-free architect
  summary fixture and prompt/validator coverage. Focused stress passed 20/20; full DoD passed with
  261 Python and 142 UI tests, lint, typecheck and embedded UI build.
- 2026-07-20: PR #168 merged as `ff140f44`; the next clean smoke exposed the separate inline-program
  command construction defect before step2 artifacts were authored.

---

## EP-20260720-epic18-step2-direct-literal-first-pass

### Context
- Clean qualification `smoke-tiny-bank-20260720T201011Z` completed init collect `10/10`, then
  `init.step2.asis_docs` exhausted five repairs after a provider-authored inline Python generator
  failed with a nested f-string `SyntaxError` before producing the required write set.

### Results
- Normal step2 startup now permits at most one bounded evidence read/list command and immediately
  requires one mechanically simple direct-literal single-quoted-heredoc write of all Markdown
  targets, marker check and manifest-last output.
- Provider-free prompt regressions preserve the same Claude/Qwen/Codex contract and prohibit inline
  generators, templates and nested quote tricks before the complete write set exists.
- Focused tests passed 20 consecutive runs and full deterministic DoD passed with Go 1.25.10,
  Node 22.21.1, npm 10.9.4, 261 Python tests and 142 UI tests.
- Merged as PR #169. The next clean smoke completed init and refresh step2 on the normal first pass
  with zero repairs; R3 then stopped on a separate valid-artifact stall-accounting defect.

---

### Plan ID
EP-20260720-epic18-architecture-home-concrete-evidence-refs

### Context
The clean post-remediation Codex qualification smoke `smoke-tiny-bank-20260720T143329Z` passed
init/refresh, collect `10/10`, strict validation and the ProductShell frontend gate. The matrix still
reported release-quality blockers because provider-authored Architecture Home drafts used
`bank-of-anthos:.` and `bank-of-anthos:src/*`. Both are rejected by the existing public evidence
contract, but normal and focused prompts described only "existing paths" and did not explicitly
exclude repository-root shorthand or wildcard syntax. This is a product prompt-contract defect;
the regression and remediation must remain provider-free and independent of live matrix identity.

### Goals (must have)
- [x] Require exact non-root repo:path references in normal, repair, compact and command-text step2
      prompts; explicitly prohibit root shorthand and wildcard/glob syntax.
- [x] Preserve strict validation and improve its deterministic diagnostics for those two invalid
      reference classes without weakening containment, existence or symlink checks.
- [x] Add the reduced authored fixture plus exact-valid/invalid validator and prompt-contract tests.
- [x] Synchronize product/runtime/testing/live-gate docs and pass full deterministic DoD.
- [x] Merge the remediation and restart R3 from the new clean merge commit.

### Non-goals
- [x] Do not rewrite or guess provider-authored evidence references.
- [x] Do not change schemas, HTTP APIs, provider contracts, timeouts or canonical matrices.
- [x] Do not expose matrix IDs, verdicts, assessments or live-only environment to product code.

### Acceptance criteria
- [x] `repo:.`, `repo:./` and wildcard/glob repo references are explicitly prohibited by every
      step2 authoring/recovery prompt surface.
- [x] Exact existing file/directory references continue to validate.
- [x] Provider-free fixtures reproduce both live-observed invalid reference forms.
- [x] Focused tests and `make contracts`, `make test`, `make lint`, `make build` pass.

### Progress log
- 2026-07-20: Stopped the release sequence after the new smoke passed machine/frontend gates but
  reported `runtime_quality.repair_heavy` and `runtime_quality.stall_pressure`; taskrun diagnostics
  isolated the causes to repository-root shorthand and wildcard Architecture Home references.
- 2026-07-20: Added exact-path prompt contracts and deterministic validator diagnostics with a
  reduced provider-free authored fixture. Focused exact/invalid reference coverage passed 20/20;
  full deterministic DoD passed with 261 Python and 142 UI tests plus lint and embedded UI build.
- 2026-07-20: PR #167 merged as `4caa5cd0`; the fresh qualification proved exact repository refs
  on init and refresh, then exposed the separate first-pass runtime-narration defect.

---

### Plan ID

EP-20260708-console-source-hydration-recovery

### Context
Follow-up medium-depth rendered UX/UI audit found two operator issues after the initial Console V2 hardening slice. Source could render the default `my-service` sample draft instead of the backend auto-init `workspace.yaml` repo when Go-authored YAML used four-space list indentation; saving from that state could overwrite a valid manifest with sample values. Manifest reload failures also kept the shell visible but did not surface a clear recoverable error. A post-commit audit then found one remaining P2 mobile readability issue: long workspace/source paths in validation status blocks could create internal clipped overflow even though page-level overflow was absent.

### Goals (must have)
- [x] Hydrate guided Source repos/docs imports from both UI-authored and Go-authored `workspace.yaml` indentation.
- [x] Preserve hydrated repo values when the operator saves Source without making edits.
- [x] Surface workspace manifest reload failures as visible recoverable console errors while keeping Refresh and current shell controls available.
- [x] Run focused UI tests, full UI test/typecheck, exact-node build, full DoD and bounded fake live smoke before the first commit.
- [x] Run a follow-up medium live UX/UI audit and close the remaining rendered P2 path readability finding in a second commit.

### Non-goals
- [x] Do not change public API response shapes, workspace schema/spec, runtime contracts, or backend run behavior.
- [x] Do not write to analyzed source repositories.
- [x] Do not run release/regres matrices or create `release_verdict_*` artifacts.

### Files changed
- `ui/src/lib/workspaceSetupState.ts`
- `ui/src/hooks/useManifestEditor.ts`
- `ui/src/App.test.tsx`
- `ui/src/styles.css`
- `docs/PLANS.md`
- `docs/archive/PLANS_ARCHIVE_2026-07.md`
- `internal/api/ui_dist/*`

### Acceptance criteria
- [x] Source form/table hydrate `repos[]` from Go-style `workspace.yaml` with four-space list indentation.
- [x] Saving the hydrated Source form preserves the loaded repo and does not persist the sample `my-service` draft.
- [x] Manifest reload failure shows a visible `Error: ...` message while TopStatusBar, Refresh, and current shell remain mounted.
- [x] Live fake Source screenshot shows the real auto-init repo path/name before any manual Source edit.
- [x] Mobile Source/Readiness validation status blocks wrap long path/code values without page-level overflow or sub-44px actionable controls.

### Progress log
- 2026-07-08: Implemented parser/recovery regression fixes and added focused component tests. Full UI suite, typecheck, exact-node build, `make contracts`, `make test`, `make lint`, and bounded fake live smoke passed before commit `bd1dc18`.
- 2026-07-08: Post-commit fake live smoke passed at `http://127.0.0.1:18180` with workspace `/tmp/provenarch-ui-ux-postcommit-workspace.szRzFG`; smoke screenshots were written to `/tmp/provenarch-ui-ux-postcommit-results.DUvVSX` and manual audit evidence to `/tmp/provenarch-ui-ux-postcommit-manual.RKz9lE`. Source hydration evidence showed `provenarch`, no `my-service`, and no sample Git URL.
- 2026-07-08: Manual desktop/mobile audit found no console errors, failed requests, viewport overflow, sub-threshold controls or running reduced-motion animations. The only remaining finding was P2 internal clipped overflow for long path/code values in status/summary blocks.
- 2026-07-08: Added status/repo summary path wrapping and stabilized the Publish filter regression test. Full UI suite and typecheck passed; exact-node `make build` passed. Focused live mobile check at `http://127.0.0.1:18182` with workspace `/tmp/provenarch-ui-ux-finalfix-workspace.Qi8h13` wrote evidence to `/tmp/provenarch-ui-ux-finalfix-manual.mQnFg8` and confirmed zero viewport overflow, zero small controls, zero console errors and zero failed requests.

### Plan ID
EP-20260708-ux-ui-quality-loop

### Context
The user requested an iterative UX/UI quality loop for the ACP operator console: understand the product and user journey, inspect every screen, run live E2E on a medium task when host prerequisites allow it, write a UX/UI quality report, plan improvements, commit, and repeat until a first-time operator can complete the happy path and recovery paths clearly. Existing diagnostic evidence in `docs/archive/audits/ux_current_state_20260707.md` shows the fake-runtime UI flow passes, but the rendered console is still too dense around shared shell chrome: Activity drawer, right inspector, and active run summary compete with the stage workbench.

### Goals (must have)
- [x] Map the current user journey across onboarding, Source, Readiness, Charter, Analysis, Review, Proposals, Ask and Publish.
- [x] Capture rendered UI evidence for desktop and mobile with the deterministic fake-runtime live smoke.
- [x] Attempt the requested medium live E2E path only through public runbook surfaces; classify host/provider blockers instead of changing canonical matrices or curated repos.
- [x] Write a current UX/UI quality assessment and actionable improvement plan.
- [x] Implement the first focused UI polish slice without schema/API/runtime contract changes.
- [x] Add or update UI tests for changed shell behavior.
- [x] Run relevant verification and commit the slice.
- [x] Implement the Review hierarchy slice so the review queue and selected preview are primary while the full artifact explorer is secondary.
- [x] Add mobile Review section jumps for long first-time review sessions.
- [x] Simplify first-run Readiness by keeping advanced runtime tools compact and closed during the primary readiness path.
- [x] Add an explicit Analysis failed-run recovery path with retry and blocker drilldown.
- [x] Add an explicit Ask/Q&A failed-run recovery path with audit refs and same-question retry.
- [x] Keep the secondary Review artifact explorer stable while switching filters during evidence review.
- [x] Add Publish readiness summary and mobile section jumps for the final Git handoff path.
- [x] Add onboarding setup summary with current blocker, next action and explicit disabled-action reasons.
- [x] Restore trusted-host PostHog checkout outside the repo and execute the medium qwen-only baseline matrix through `scripts/full-run-batch-matrix.sh`.
- [x] Classify the live medium result with report/profile/status evidence instead of treating it as a visual UI regression.
- [x] Implement a live diagnostics UX slice for shard collection, focused repair and partial failure recovery.
- [x] Implement pending permission triage so managed-mode `runtime_permission_required` failures are understandable before raw table inspection.
- [x] Mirror pending permission step/rule/target/reason detail in the right inspector hard blocker.
- [x] Add cooperative cancel guidance for active selected runs without changing cancel API behavior.
- [x] Distinguish terminal canceled/reconciled runs from runtime failures in Analysis recovery and inspector copy.
- [x] Distinguish terminal canceled/reconciled Q&A runs from answer validation/runtime failures in Ask recovery.
- [x] Show terminal canceled/reconciled Analysis runs as `canceled`/`recovered` across status, mission control, active run summary and history.
- [x] Make the shared Activity drawer distinguish terminal canceled/reconciled runs from generic failures when log entries are absent.
- [x] Make Analysis recovery and global next action use retained-evidence actions for terminal canceled/reconciled runs.
- [x] Route provider-unavailable Analysis, Publish and Ask recovery copy to Readiness provider checks instead of generic failed-shard retry guidance.
- [x] Add a Readiness provider recovery surface so provider-unavailable reroutes land on command/auth/quota guidance before retry.
- [x] Add onboarding runner recovery guidance so first-time headless setup shows expected command/env override before first analysis.
- [x] Add Source validation recovery guidance so repo/source validation errors are actionable before raw diagnostics.
- [x] Add Charter baseline recovery guidance so prompt/charter bundle warnings are actionable before Analysis.
- [x] Add Proposals package recovery guidance so incomplete proposal/changelog packages are actionable before Publish.
- [x] Summarize active provider JSON stream chunks in the Activity drawer and label active Analysis provider stream as a run signal.
- [x] Add rendered browser QA for the Activity drawer provider-stream summary on desktop and narrow mobile.
- [x] Add rendered browser QA and UI polish for failed artifact-handoff shard drilldown.
- [x] Add rendered browser QA and mobile-card polish for pending runtime permission recovery.
- [x] Add rendered browser QA and layout polish for failed Ask/Q&A recovery.
- [x] Add rendered browser QA and recovery polish for failed Publish Git mutations.
- [x] Add stable rendered browser QA for Source validation recovery with long repo/source diagnostics.
- [x] Hand off remaining structural trust, IA, evidence-viewer, responsive and design-system
      work to Epic 20 and `EP-20260711-run-pinned-evidence-review` instead of extending this
      UI-only recovery-polish loop.

### Non-goals
- [ ] Do not change workspace, runtime, artifact, schema, or API contracts in this UX polish slice.
- [ ] Do not edit canonical release matrices, curated repo files, or add a wrapper over the matrix harness.
- [ ] Do not treat deterministic fake-runtime UX evidence as release readiness.
- [ ] Do not redesign the console visual language; preserve the approved dense operator-console direction.

### Approach
1) Study project docs, current UI implementation, design baseline and existing UX evidence.
2) Run fake-runtime rendered QA and inspect screenshots across primary screens.
3) Run live E2E host/tool/provider preflight for the medium path; continue only if runbook prerequisites are satisfied.
4) Write a UX/UI report with findings, state coverage, recovery gaps and prioritized fixes.
5) Implement the smallest coherent improvement slice: context-aware Activity drawer, progressive-disclosure inspector, and successful-run review emphasis.
6) Implement the next Review-focused slice: primary review queue, selected evidence preview, secondary artifact explorer and mobile section jumps.
7) Implement the next Readiness-focused slice: keep exact runtime controls available while preserving the first-run readiness path and screenshot hierarchy.
8) Implement non-happy-path recovery polish for failed Analysis and Ask runs.
9) Implement Publish handoff polish: publication readiness summary, mobile section jumps and rendered mobile Publish evidence.
10) Implement onboarding first-run polish: current setup blocker, next action and disabled-action reasons.
11) Use the medium matrix result to design the next live diagnostics slice: shard-plan progress, succeeded/failed/repairing counts, repair attempt state, terminal validation excerpt, raw-output refs and concrete rerun/triage actions.
12) Improve managed permission recovery UX using existing pending request fields: blocked step, operation, decision/rule, target/reason and safe next actions.
13) Mirror managed permission blocker context in the right inspector so the global hard-blocker surface stays actionable.
14) Clarify active-run cancellation as cooperative stop with taskrun evidence/history preserved.
15) Distinguish terminal `run_canceled` and restart-reconciled runs from runtime/provider failures.
16) Extend the same terminal canceled/reconciled language to Ask/Q&A answer runs without changing the async QA API.
17) Promote the same terminal outcome labels into Analysis status, mission control, active run summary and history counts while leaving generic runtime failures as `failed`.
18) Extend terminal outcome labels into the shared Activity drawer summary/empty-log state so stopped/reconciled runs do not read as generic failures.
19) Extend retained-evidence action copy into Analysis recovery and global next action for terminal stopped/reconciled runs.
20) Make Readiness the actionable landing surface for provider outages by showing selected provider, doctor status, command override and binary/auth/quota recovery actions.
21) Make onboarding Runner actionable for headless provider setup by showing expected command, env override, doctor status and fake fallback before first analysis.
22) Make Source validation actionable by showing the first blocking repo/source diagnostic, current source/ref and save-then-validate recovery actions before raw diagnostics.
23) Make Charter baseline warnings actionable by showing affected artifact/category/prompt usage, severity, message, suggested fix and save-then-analyze actions before the raw editor warnings.
24) Make Proposals package blockers actionable by showing proposal/changelog package state, primary blocker, suggested fix and publication path before the review room.
25) Make active provider stream readable in shared Activity by summarizing JSON stream chunks while preserving full raw log/export access.
26) Add rendered browser QA for the provider-stream Activity summary with a stable mocked running-run fixture.
27) Add rendered browser QA for failed artifact-handoff shard drilldown and fix any visual defects exposed by it.
28) Add rendered browser QA for pending runtime permission recovery and use mobile-first request cards when the raw request table is too dense.
29) Add rendered browser QA for failed Ask/Q&A recovery and keep recovery metrics readable on desktop/mobile.
30) Add rendered browser QA for failed Publish Git mutations and keep recovery local to the commit plan/inspector.
31) Add stable rendered browser QA for Source validation recovery with long Git URL/ref diagnostics, readiness blocking, desktop/mobile overflow checks and screenshots.
32) Verify with component tests, UI build, rendered smoke when feasible, and DoD commands proportional to the slice.
33) Commit and use the report to drive the next iteration.

### Files expected to change
- `ui/src/components/ActivityDrawer.tsx`
- `ui/src/components/RightInspector.tsx`
- `ui/src/components/ActiveRunStrip.tsx`
- `ui/src/components/OnboardingShell.tsx`
- `ui/src/App.tsx`
- `ui/src/components/StagePanels.tsx`
- `ui/src/components/ConsoleShellPrimitives.test.tsx`
- `ui/src/App.test.tsx`
- `ui/e2e/live-flow.spec.ts`
- `ui/e2e/provider-stream-mock.spec.ts`
- `ui/e2e/analysis-failed-shard-mock.spec.ts`
- `ui/e2e/permission-recovery-mock.spec.ts`
- `ui/e2e/qa-recovery-mock.spec.ts`
- `ui/e2e/publish-git-recovery-mock.spec.ts`
- `ui/e2e/source-recovery-mock.spec.ts`
- `ui/src/hooks/useGitActions.ts`
- `ui/src/styles.css`
- `docs/archive/audits/ux_ui_assessment_20260708.md`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/archive/design/UI_CONSOLE_V2_DESIGN.md`
- `docs/STAKEHOLDER_DOC.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Activity drawer keeps active/failed run diagnostics easy to open, but does not consume review/publish attention after success by default.
- [x] Right inspector keeps `Next action` prominent and collapses or summarizes empty/secondary sections without hiding hard blockers, warnings, evidence, runtime safety or Git publication access.
- [x] Active run strip highlights review/artifact state after a successful run instead of making stale `current_step` the dominant signal.
- [x] Review makes the queue and selected evidence preview the primary path, while the full artifact explorer is an explicit secondary disclosure.
- [x] Mobile Review exposes section jumps before the long evidence/queue/artifact/trust stack.
- [x] Readiness keeps advanced runtime panels out of the first-run visual path while preserving direct operator access to timeouts/execution/permissions/provider overrides.
- [x] Analysis failed runs show classification, blocked step, retained evidence, warnings and a retry action for the same pipeline.
- [x] Ask failed runs show classification, blocked step, QA audit refs, warnings and a retry action for the same question.
- [x] Ask terminal canceled/reconciled runs show canceled/recovered outcome, stopped/recovered step, retained QA audit evidence and an Ask-again action instead of answer-validation failure wording.
- [x] Review artifact explorer stays open and keeps filtered artifact lists visible while switching between diagram/report/proposal/runtime groups.
- [x] Publish shows publication set, gate, open-question and Git-action state above the fold and exposes mobile jumps for `Diff`, `Preview`, `Gate` and `Commit`.
- [x] Onboarding shows the current setup step, current blocker, next action and disabled-action reason before the user enters Console V2.
- [x] Primary fake-runtime flow remains passable through live Playwright smoke, including Ask and mobile Review evidence.
- [x] Medium live E2E is either executed through `scripts/full-run-batch-matrix.sh` or explicitly blocked with runbook classification and evidence.
- [x] Live medium execution evidence is captured from `matrix_result_regres-long-posthog-ftgo-20260708T130309Z.json`, profile status JSON and execution reports.
- [x] Live failed-run diagnostics expose shard progress and recovery state without requiring raw log inspection first.
- [x] Live diagnostics distinguish provider activity, focused repair, partial collect continuation and terminal runtime contract failure.
- [x] Pending permission requests expose a triage summary, target/reason, policy rule, decision and safe next actions before the raw request table.
- [x] Right inspector hard blockers expose pending permission step, rule, target and reason instead of a generic pending message.
- [x] Active selected runs explain that cancel is cooperative and preserves taskrun evidence/history.
- [x] Terminal canceled runs show stopped-step/run-again/History evidence copy instead of generic runtime-failure recovery.
- [x] Terminal canceled/restart-reconciled Analysis runs show `canceled`/`recovered` in status, mission control, active run summary and History without reclassifying generic runtime failures.
- [x] Activity drawer empty-log states show canceled/recovered recovery copy with retained History evidence instead of `Run failed before log entries`.
- [x] Analysis recovery and right-inspector next action say run again/review retained evidence for terminal canceled/reconciled runs instead of generic retry/blocker wording.
- [x] Readiness provider recovery shows selected provider, doctor status/message, command override and last `runner_unavailable` blocker before retry.
- [x] Onboarding Runner shows selected provider, expected command, command override, doctor status/message and fake fallback before first live analysis.
- [x] Readiness and Onboarding provider recovery classify headless probe timeout, artifact-smoke, auth/quota and command-unavailable readiness blockers before retry.
- [x] Source validation recovery shows affected repo/workspace, diagnostic, source type, current source/ref and save-then-validate actions before raw validation details.
- [x] Charter baseline recovery shows affected artifact, category, runtime use, diagnostic, suggested fix and save-selected-artifact actions before raw editor warnings.
- [x] Proposals package recovery shows proposal/changelog package state, primary blocker, suggested fix and publication path before Publish.
- [x] Analysis live diagnostics classify pre-artifact artifact-handoff stalls, show the recovery stage and tell the operator to confirm both markdown and `shard-pack-manifest.json` before retry.
- [x] Analysis shard drilldown shows per-shard artifact-pair state, distinguishing runtime-only evidence from authored markdown + `shard-pack-manifest.json`.
- [x] Activity drawer summarizes provider JSON stream chunks without hiding full raw payload access.
- [x] Active provider-stream Analysis diagnostics use `Run signal` instead of terminal `Failure mode` wording.
- [x] Rendered provider-stream browser QA verifies desktop and narrow-mobile Activity summary readability, raw payload disclosure, and no horizontal overflow.
- [x] Rendered failed-shard browser QA verifies artifact-handoff recovery, shard-scoped drilldown, long-classification wrapping and no horizontal overflow.
- [x] Rendered permission recovery browser QA verifies Analysis triage, right-inspector blocker detail, Readiness runtime-permission settings access, mobile request cards and no horizontal overflow.
- [x] Rendered Ask recovery browser QA verifies failed answer guidance, QA audit refs, retry behavior, read-only safety, no citations state and no horizontal overflow.
- [x] Rendered Publish Git recovery browser QA verifies failed commit/proposal branch mutations, local recovery copy, inspector state, retryability and no horizontal overflow.

### Risks
- Collapsing shared shell sections can hide useful diagnostic context if the summary copy is weak; tests should verify critical sections remain discoverable.
- Live provider runs can fail on quota/auth/toolchain/path prerequisites; those must be classified as operational blockers, not product UX regressions.
- UI density improvements can break existing live E2E selectors if test IDs or visible controls move unexpectedly.

### Progress log
- 2026-07-08: Started UX/UI quality loop. Read project mission, architecture, pipeline spec, live E2E runbook, design baseline, current UX smoke report and shell implementation.
- 2026-07-08: Fresh fake-runtime rendered smoke passed (`run_20260708_094849_001`, QA `run_20260708_094856_002`) and confirmed the primary UX debt: succeeded Review/Ask/Publish screens were dominated by old init logs and repeated empty inspector sections. Medium live E2E was blocked before execution as `operational_host_preflight_failed` because `/tmp/provenarch-live-e2e/posthog/posthog` exists but is not a Git checkout at the pinned SHA; no canonical matrix or curated repo file was changed.
- 2026-07-08: Implemented slice 1 shell polish: context-aware Activity drawer, progressive-disclosure right inspector and review-oriented active run strip. Targeted shell tests, full UI Vitest suite, typecheck, build and fake-runtime rendered smoke passed (`run_20260708_095725_001`, QA `run_20260708_095731_002`). Full DoD (`make contracts test lint build`) passed with exact Node `22.21.1`; commit prepared for this slice.
- 2026-07-08: Implemented slice 2 Review hierarchy polish: queue/selected preview are now the primary review path, full artifact explorer is a secondary disclosure, selected queue items are highlighted, and mobile Review exposes section jumps. Targeted App tests, full UI Vitest suite, typecheck, build and fake-runtime rendered smoke passed (`run_20260708_102054_001`). Full DoD (`make contracts test lint build`) passed with exact Node `22.21.1`.
- 2026-07-08: Re-ran the medium `regres long` preflight through `scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell`. Writable roots and provider binaries were present (`qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`), but `/tmp/provenarch-live-e2e/posthog/posthog` still failed `git rev-parse HEAD` because it is not a Git checkout. Matrix execution remains blocked as `operational_host_preflight_failed`; no canonical matrix or curated repo file was changed.
- 2026-07-08: Implemented slice 3 Readiness polish: advanced runtime settings now render as a compact operator-tools disclosure, default closed, with live smoke verifying the settings remain reachable but closing them before the readiness screenshot. Targeted App tests, full UI Vitest suite, typecheck, build and fake-runtime rendered smoke passed (`run_20260708_104434_001`). Full DoD (`make contracts test lint build`) passed with exact Node `22.21.1`.
- 2026-07-08: Re-ran the medium `regres long` preflight after slice 3. Writable roots and provider binaries were still present (`qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`), and `scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell` still generated the direct `full-run-batch-matrix.sh` command. Matrix execution remains blocked as `operational_host_preflight_failed` because `/tmp/provenarch-live-e2e/posthog/posthog` still fails `git rev-parse HEAD`; no canonical matrix or curated repo file was changed.
- 2026-07-08: Implemented slice 4 non-happy-path polish: failed Analysis runs now show a dedicated recovery path with error classification, blocked step, retained evidence, warning count, same-pipeline retry and blocker drilldown. Targeted App tests, full UI Vitest suite, typecheck, build, fake-runtime rendered smoke (`run_20260708_111016_001`) and full DoD (`make contracts test lint build`) passed.
- 2026-07-08: Re-ran the medium `regres long` preflight after slice 4. Writable roots and provider binaries were still present (`qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`), and `scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell` still generated the direct `full-run-batch-matrix.sh` command. Matrix execution remains blocked as `operational_host_preflight_failed` because `/tmp/provenarch-live-e2e/posthog/posthog` still fails `git rev-parse HEAD`; current UX changes also leave the main worktree dirty until committed, so this is not an acceptance run.
- 2026-07-08: Implemented slice 5 Ask/Q&A recovery polish: failed Q&A runs now show recovery guidance with classification, blocked step, `reports/taskruns/<run_id>/qa/` audit refs, warning detail and same-question retry. The rendered smoke exposed a Review artifact explorer regression where switching to `Diagrams` closed or hid the secondary artifact list; the explorer is now controlled and filter actions keep it open. Targeted App tests, typecheck, build, fake-runtime rendered smoke (`run_20260708_115053_001`) and full DoD (`make contracts test lint build`) passed with exact Node `22.21.1`.
- 2026-07-08: Re-ran the medium `regres long` preflight after slice 5. Writable roots and provider binaries were still present (`qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`), and `scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell` still generated the direct `full-run-batch-matrix.sh` command. Matrix execution remains blocked as `operational_host_preflight_failed` because `/tmp/provenarch-live-e2e/posthog/posthog` still fails `git rev-parse HEAD`; no canonical matrix, curated repo file or wrapper script was changed.
- 2026-07-08: Implemented slice 6 Publish handoff polish: Publish now shows an above-the-fold readiness summary for publication set, gate state, open questions and Git action; mobile Publish exposes sticky section jumps for `Diff`, `Preview`, `Gate` and `Commit`; live smoke now captures `frontend-publish-mobile.png`. Targeted App tests, typecheck, frontend live E2E contract tests, build, fake-runtime rendered smoke (`run_20260708_121704_001`) and full DoD (`make contracts test lint build`) passed with exact Node `22.21.1`.
- 2026-07-08: Re-ran the medium `regres long` preflight after slice 6. Writable roots and provider binaries were still present (`qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`), and `scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell` still generated the direct `full-run-batch-matrix.sh` command. Matrix execution remains blocked as `operational_host_preflight_failed` because `/tmp/provenarch-live-e2e/posthog/posthog` still fails `git rev-parse HEAD`; current UX changes also leave the main worktree dirty until committed, so this is not an acceptance run.
- 2026-07-08: Implemented slice 7 onboarding blocker-summary polish: the pre-console onboarding screen now shows current setup step, next action, current blocker, per-step status and disabled-action reasons for `Open console` / `Run first analysis`. Targeted App tests, typecheck, full UI Vitest suite, build, rendered onboarding smoke (`/tmp/provenarch-ui-onboarding-summary-results.KbB5En`), fake-runtime rendered smoke (`run_20260708_124024_001`) and full DoD (`make contracts test lint build`) passed with exact Node `22.21.1`.
- 2026-07-08: Re-ran the medium `regres long` preflight after slice 7. Writable roots and provider binaries were still present (`qwen 0.17.1`, `Claude Code 2.1.85`, `codex-cli 0.131.0`, Node `v22.21.1`), and `scripts/live-e2e-plan.py --mode regres --size long --providers qwen --format shell` still generated the direct `full-run-batch-matrix.sh` command. Matrix execution remains blocked as `operational_host_preflight_failed` because `/tmp/provenarch-live-e2e/posthog/posthog` still fails `git rev-parse HEAD`; current UX changes also leave the main worktree dirty until committed, so this is not an acceptance run.
- 2026-07-08: Restored the trusted-host PostHog checkout outside the repo at pinned ref `14d29a548d63665d60b506cf13bd5cfb2de7c743` and ran `regres long` qwen-only baseline through direct `scripts/full-run-batch-matrix.sh` with matrix id `regres-long-posthog-ftgo-20260708T130309Z`. Matrix result was `FAIL`: both `single-path` PostHog and `single-git_url` FTGO failed in `init.step1.collect` with `runtime_contract_failed`, `quality_gates_failed`, `runtime_flow_failed`, partial shard failures, repair exhaustion and stall pressure. PostHog recorded `partial_failure_count=9`; FTGO recorded `partial_failure_count=8`, `repair_attempts=11`, `repair_exhausted=8`, `focused_repairs=11`, `stall_count=17`, `pre_artifact_stalls=17`, and raw output refs `8`. UX conclusion: the next slice should improve live diagnostics and recovery clarity for shard collection/repair failures before rerunning medium live E2E.
- 2026-07-08: Implemented slice 9 live diagnostics in Analysis failed-run recovery without backend/schema changes. The panel derives shard state, focused repair counts, stall pressure, provider refs, terminal validation excerpt and raw-output refs from existing run logs/artifacts, with targeted App coverage for collect partial + `collect_pair_repair` + `runtime_stalled_before_artifacts` evidence.
- 2026-07-08: Attempted the next `regres long` qwen-only medium rerun from clean commit `8a7786a`. Preflight passed for clean tree, provider binaries and report root, but `/tmp/provenarch-live-e2e` was absent and restoring canonical PostHog path checkout failed with `No space left on device` after the partial object store reached about `3.3G` and `/tmp` had only `116MiB` free. Removed the temporary partial checkout and stopped before matrix execution as `operational_host_preflight_failed`; rerun requires a trusted host or volume with enough space for the pinned PostHog checkout.
- 2026-07-08: Implemented slice 11 pending permission triage in `Analysis -> Pending permissions` without backend/schema changes. The panel now summarizes blocked step, operation, decision, policy rule, primary target/reason and safe next actions before the raw request table; approve/deny broker remains future scope. Targeted App test (`67` tests), UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 12 right-inspector permission blocker copy without backend/schema changes. Pending permissions now surface as `Permission: <action>` with step, decision/rule, target and reason in the global hard-blocker panel; targeted App test (`67` tests), UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 13 active-run cancellation guidance without backend/API/schema changes. Active selected runs now explain that cancel requests a cooperative stop and keeps taskrun evidence in History; targeted primitive test (`10` tests), UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 14 terminal cancellation recovery copy without backend/API/schema changes. Terminal `run_canceled` runs now show canceled/stopped-step/run-again/retained-History evidence wording in Analysis recovery and right-inspector blockers, while restart-reconciled runs use preserved-evidence copy; targeted App test (`68` tests), UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 16 Analysis outcome-label polish without backend/API/schema changes. Terminal `run_canceled` and restart-reconciled failed runs now display as `canceled`/`recovered` in Analysis header, mission control, status panel, active run strip and History counts/table; generic runtime/provider failures remain `failed`. Targeted App/primitive tests (`79` tests), UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 17 Activity drawer outcome-label polish without backend/API/schema changes. Terminal canceled/recovered selected runs now show `canceled run`/`recovered run` summaries and empty-log recovery copy with retained History guidance; generic failures still say `Run failed before log entries`. Targeted App/primitive tests (`79` tests), UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 18 retained-evidence action polish without backend/API/schema changes. Analysis recovery and the global next-action inspector now use `Run <pipeline> again`, `Review retained evidence` and retained-History copy for terminal canceled/restart-reconciled runs, while generic failures still use retry/blocker wording. Targeted App test (`69` tests), UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 19 provider-unavailable recovery polish without backend/API/schema changes. `runner_unavailable` now shows `Provider unavailable`, routes the global next action to Readiness provider checks, explains binary/auth/quota before retry, updates Analysis live diagnostics away from generic failed-shard guidance, and uses the same Readiness guidance in Ask and Publish blockers. Targeted App test (`70` tests), UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 20 Readiness provider recovery without backend/API/schema changes. Readiness now shows selected provider, doctor status/message/suggested fix, command override and last run blocker for headless/provider-unavailable/doctor-fail states; the inspector route lands there without starting a retry. Targeted App test (`70` tests), UI typecheck, fake-runtime rendered smoke (`run_20260708_194304_001`) and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 21 onboarding runner recovery without backend/API/schema changes. Runner onboarding now shows selected provider, expected executable, `ACP_*_CMD` override, runtime-provider doctor status/message and fake-baseline fallback before first live analysis. Targeted App test (`70` tests), UI typecheck, rendered onboarding desktop/mobile check (`/tmp/provenarch-ui-onboarding-runner-manual.laqv2_2p`), fake-runtime rendered smoke (`run_20260708_195749_001`) and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-08: Implemented slice 22 Source validation recovery without backend/API/schema changes. Source now shows affected repo/workspace, diagnostic, source type, current source/ref, suggested fix and save-then-validate actions before raw validation details. Targeted App test (`71` tests), UI typecheck, rendered Source recovery desktop/mobile check (`/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-source-recovery-manual.Z9MEYk`), fake-runtime rendered smoke (`run_20260708_201043_001`) and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-09: Implemented slice 23 Charter baseline recovery without backend/API/schema changes. Charter now shows affected artifact, category, runtime use, diagnostic, suggested fix and exact `Save selected baseline artifact` action before raw editor warnings. Targeted App test (`72` tests), UI typecheck, rendered Charter recovery desktop/mobile check (`/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-charter-recovery-manual.zN6vmE`), fake-runtime rendered smoke (`run_20260709_062422_001`) and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-09: Implemented slice 24 Proposals package recovery without backend/API/schema changes. Proposals now shows proposal/changelog package state, primary blocker, suggested fix and publication path before the review room when generated proposal packages are partial, while changelog-only runs keep the changelog preview selected. Targeted App test (`72` tests), UI typecheck, rendered Proposals recovery desktop/mobile check (`/tmp/provenarch-ui-proposals-recovery-manual.TNgFwb`), fake-runtime rendered smoke (`run_20260709_064232_001`) and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed.
- 2026-07-09: Bootstrapped the canonical PostHog path checkout at pinned ref `14d29a548d63665d60b506cf13bd5cfb2de7c743` and ran the requested medium `regres long` qwen-only baseline through direct `scripts/full-run-batch-matrix.sh` with matrix id `regres-long-posthog-ftgo-20260709T065646Z`. Matrix result was `FAIL`: both `single-path/baseline` and `single-git_url/baseline` stopped before backend/frontend execution as `operational_host_preflight_failed` because `qwen` headless readiness timed out after 30s. This is host/provider readiness evidence, not a product UI/backend verdict.
- 2026-07-09: Implemented slice 25 provider readiness timeout guidance without backend/API/schema changes. Readiness and Onboarding now derive failure mode/probe stage/operator focus for headless probe timeout, artifact smoke, auth/quota and command-unavailable blockers from existing doctor/run error text. Targeted App provider-readiness tests, UI typecheck, rendered qwen-timeout desktop/mobile check (`/var/folders/0y/qkpd1n592qjgm3w3rcl_gs6m0000gn/T/provenarch-ui-provider-timeout-manual.1wiqoc`) and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `90` UI tests, Vite build and Go build).
- 2026-07-09: Re-ran the medium `regres long` qwen-only baseline from clean commit `3de030e` through direct `scripts/full-run-batch-matrix.sh` with matrix id `regres-long-posthog-ftgo-20260709T072237Z`. Matrix result was `FAIL`: `single-path/baseline` reached PostHog backend execution and failed in `init.step1.collect` with `runtime_contract_failed`, `quality_gates_failed`, `runtime_flow_failed`, `partial_failure_count=10`, `repair_attempts=14`, `repair_exhausted=10`, `stall_count=27`, `pre_artifact_stalls=24`, `raw_output_refs=10`, and shard summary `6 succeeded / 10 failed`; each failed shard reported `stage=collect_pair_repair` and `runtime_stalled_before_artifacts`. `single-git_url/baseline` failed earlier as `operational_host_preflight_failed` because qwen headless readiness timed out after 30s.
- 2026-07-09: Implemented slice 26 Analysis artifact-handoff guidance without backend/API/schema changes. Analysis live diagnostics now parse `stage=collect_pair_repair` and `runtime_stalled_before_artifacts` from existing run logs, classify the failure as `Artifact handoff stalled`, show the recovery stage, and put markdown + `shard-pack-manifest.json` verification before blind retry. Targeted App test, full UI Vitest suite, UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `90` UI tests, Vite build and Go build).
- 2026-07-09: Implemented slice 27 per-shard artifact-pair inspection without backend/API/schema changes. Analysis shard/log table and blocker drilldown now derive artifact-pair state from existing selected-run artifact refs, distinguishing `Runtime only`, `Markdown only`, `Manifest only`, `Artifact pair present` and missing shard-artifact states. Targeted App test, full UI Vitest suite, UI typecheck and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `90` UI tests, Vite build and Go build).
- 2026-07-09: Implemented slice 30 Activity provider-stream readability without backend/API/schema changes. The current medium run `regres-long-posthog-ftgo-20260709T112238Z` reached PostHog qwen collect and repeated `collect_pair_repair -> runtime_stalled_before_artifacts` (`4 failed / 12 pending` by DoD checkpoint), then was diagnostically stopped as `infra_signal_terminated`; the first 4 shard failures are genuine artifact-handoff failures, later `context canceled` rows came from the stop, and no top-level `matrix_result_*` was written. Activity now summarizes JSON stream chunks while preserving full raw log/export access, and active Analysis labels stream telemetry as `Run signal`. Targeted App/primitive tests and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `92` UI tests, Vite build and Go build).
- 2026-07-09: Implemented slice 31 rendered provider-stream QA without backend/API/schema changes. Medium rerun `regres-long-posthog-ftgo-20260709T120910Z` failed before backend/frontend execution in both profiles as `operational_host_preflight_failed` because `qwen` headless readiness timed out after 30s, so the product UI evidence came from a stable mocked running-run fixture. New Playwright coverage verifies the Analysis live diagnostics, Activity summary, raw-payload disclosure and no horizontal overflow on desktop and narrow mobile; screenshots are under `/tmp/provenarch-ui-provider-stream-rendered-20260709T1230Z`. Targeted UI checks and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `92` UI tests, Vite build and Go build).
- 2026-07-09: Implemented slice 32 rendered failed-shard QA and drilldown polish without backend/API/schema changes. Medium rerun `regres-long-posthog-ftgo-20260709T123421Z` again failed before backend/frontend execution in both profiles as `operational_host_preflight_failed` because `qwen` headless readiness timed out after 30s. New Playwright coverage verifies the live-shaped `domain_id != shard_id` artifact-handoff failure on desktop/detail/mobile, while UI polish wraps long recovery metric values and keeps blocker drilldown focused on shard-scoped failures when present. Screenshots are under `/tmp/provenarch-ui-analysis-failed-shard-rendered-20260709T1240Z`; targeted UI checks and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `92` UI tests, Vite build and Go build).
- 2026-07-09: Implemented slice 33 rendered permission-recovery QA without backend/API/schema changes. Medium rerun `regres-long-posthog-ftgo-20260709T125743Z` again failed before backend/frontend execution in both profiles as `operational_host_preflight_failed` because `qwen` headless readiness timed out after 30s. New Playwright coverage verifies Analysis permission triage, right-inspector blocker detail, Readiness runtime-permission settings access and narrow-mobile readability. Visual QA exposed that the raw pending-permissions table was too dense on mobile, so pending permission requests now render mobile-first cards while retaining the desktop raw table. Screenshots are under `/tmp/provenarch-ui-permission-recovery-rendered-20260709T1310Z`; targeted UI checks and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `92` UI tests, Vite build and Go build).
- 2026-07-09: Implemented slice 34 rendered Ask/Q&A recovery QA without backend/API/schema changes. Medium rerun `regres-long-posthog-ftgo-20260709T131747Z` again failed before backend/frontend execution in both profiles as `operational_host_preflight_failed` because `qwen` headless readiness timed out after 30s. New Playwright coverage verifies failed `qa.ask` recovery guidance, audit refs, retrying the original question, read-only safety, no-citations state and no horizontal overflow. Visual QA exposed narrow desktop recovery cards and a wrapped `Refresh` action in Q&A history, so Ask recovery metrics now use responsive grid tracks and the history refresh action keeps a stable single-line label. Screenshots are under `/tmp/provenarch-ui-qa-recovery-rendered-20260709T1329Z`; targeted UI checks and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `92` UI tests, UI typecheck, Vite build and Go build).
- 2026-07-09: Implemented slice 35 rendered Publish Git recovery QA without backend/API/schema changes. Medium rerun `regres-long-posthog-ftgo-20260709T133940Z` again failed before backend/frontend execution in both profiles as `operational_host_preflight_failed` because `qwen` headless readiness timed out after 30s. New Playwright coverage verifies failed commit/proposal branch mutations, local Commit plan recovery, Git publication inspector error state, retryability and no horizontal overflow. Visual QA exposed that the global next-action inspector still looked ready after a failed Git mutation, so Publish now changes next action to blocked recovery guidance until the operator reviews the Git failure. Screenshots are under `/tmp/provenarch-ui-publish-git-recovery-rendered-20260709T1352Z`; targeted UI checks and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `93` UI tests, UI typecheck, Vite build and Go build).
- 2026-07-09: Implemented slice 36 stable Source recovery rendered QA without backend/API/schema changes. Medium rerun `regres-long-posthog-ftgo-20260709T141636Z` produced real PostHog runtime evidence in `single-path/baseline`: matrix `FAIL`, shard summary `6 succeeded / 10 failed`, repeated `collect_pair_repair -> runtime_stalled_before_artifacts`, `partial_failure_count=10`, and excellent blockers on `init.step1.collect`; `single-git_url/baseline` stopped at qwen `headless_probe_timeout`. New Playwright coverage verifies Source validation recovery for a long Git URL/ref, blocked readiness state, desktop/mobile readability and no horizontal overflow. Screenshots are under `/tmp/provenarch-ui-source-recovery-rendered-20260709T1614Z`; rendered QA and full DoD (`make contracts test lint build` with exact Node `22.21.1`) passed (`230` Python tests, `93` UI tests, UI typecheck, Vite build and Go build).
- 2026-07-11: Closed the open-ended polish continuation with a structural audit handoff.
  Recovery/rendered-QA slices remain regression constraints; new trust, IA, viewer, responsive
  and design-system work is sequenced in Epic 20, starting with
  `EP-20260711-run-pinned-evidence-review`.
### Plan ID
EP-20260708-console-v2-ux-ui-remediation

### Context
Medium-depth rendered UX/UI audit follow-up for Console V2. This completed remediation slice locked the first round of audited behavior: recoverable refresh handling, onboarding recent-workspace density, Review/Publish artifact filters, operator-facing microcopy, semantic CSS aliases, shared control sizing and regression tests. Backend APIs, schemas, runtime contracts and source-repo write boundaries remained unchanged.

### Goals (must have)
- [x] Keep P0 criteria documented as a no-op class for this slice.
- [x] Add UI recovery coverage for transient API/bootstrap failures so an already-open console stays usable and Refresh remains available.
- [x] Preserve readiness gating, fake runtime labels, source include/exclude scope persistence, StageRail keyboard/mobile behavior, TabNav ARIA semantics and CSS token guard coverage.
- [x] Collapse onboarding Recent workspaces to three entries by default with a reveal-all affordance.
- [x] Add client-side Review and Publish artifact filters over existing artifact data only.
- [x] Remove user-facing raw step-id copy from Proposals empty state and rename Readiness rail helper copy to operator-facing language.
- [x] Add semantic CSS aliases for recurring focus/sidebar/warning colors while preserving the current visual palette.

### Non-goals
- [x] Do not change public API response shapes, workspace schema/spec, runtime contracts or backend behavior.
- [x] Do not run release/regres matrix or create `release_verdict_*` artifacts.
- [x] Do not write to analyzed source repos.
- [x] Do not introduce a new UI library or redesign the Console shell.

### Acceptance criteria
- [x] UI tests cover refresh recovery, collapsed recents, Review/Publish filters and existing regression behavior.
- [x] `npm run --prefix ui test -- --run` passes.
- [x] `npm run --prefix ui typecheck` passes.
- [x] Exact-node `make contracts`, `make test`, `make lint`, and `make build` pass.
- [x] Bounded fake live smoke passes without `release_verdict_*` artifacts.

### Progress log
- 2026-07-08: Implemented and verified the first Console V2 UX/UI hardening slice. Bounded fake live smoke passed at `http://127.0.0.1:18180` with workspace `/tmp/provenarch-ui-ux-remediation.IEYY48`; automated screenshots were written under `/tmp/provenarch-ui-ux-remediation-results`, manual rendered screenshots/report under `/tmp/provenarch-ui-ux-remediation-manual`. Build still reported the known large lazy dependency chunk warning.
- 2026-07-08: Archived after follow-up audit opened `EP-20260708-console-source-hydration-recovery` for the remaining Source hydration/recovery P1 issues.

### Plan ID
EP-20260707-console-v2-ux-hardening

### Context
Follow-up P1 slice from the Console V2 UX audit. The target was a reviewable frontend-only hardening pass: readiness gating must not imply the first analysis is runnable before local doctor checks pass, fake runtime labels must stay unambiguous, source analysis include/exclude should be editable without raw YAML, and mobile/navigation controls need consistent sizing and affordance. Backend APIs, schemas, runtime contracts, fixtures, canonical matrices and source repositories remained unchanged.

### Goals (must have)
- [x] Gate onboarding/Readiness `Run first analysis` behind successful local readiness while keeping `Open console` available after workspace/source/runtime validation.
- [x] Display `fake` / `fake baseline` consistently across readiness, analysis, Ask/run and top-status surfaces without changing response shapes.
- [x] Add guided include/exclude editing for existing `repos[].analysis.include/exclude` and replace source table advanced-only copy with scope counts.
- [x] Normalize shared interactive sizing/tokens, TabNav accessibility semantics and mobile StageRail active-stage visibility.
- [x] Update focused UI tests, including CSS token guard and StageRail keyboard/scroll behavior.
- [x] Run full DoD plus bounded fake-runtime frontend live smoke.

### Non-goals
- [x] Do not change public API, backend runtime behavior, workspace schema/spec, schemas, fixtures, canonical matrices or release verdict artifacts.
- [x] Do not include P2 visual polish or broader redesign in this slice.

### Approach
1) Add internal UI helpers/types for runtime labels and analysis scope drafts while serializing the existing `workspace.yaml` contract.
2) Apply guided Source/onboarding scope controls and readiness/doctor gating copy in the shell and stage panels.
3) Consolidate tab semantics and control sizing in shared CSS/component primitives.
4) Preserve keyboard behavior and add narrow-viewport StageRail scroll visibility.
5) Update tests, docs and verification evidence.

### Files changed
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/PLANS.md`
- `docs/archive/PLANS_ARCHIVE_2026-07.md`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `ui/src/components/*`
- `ui/src/hooks/useManifestEditor.ts`
- `ui/src/lib/*`
- `ui/src/styles.css`

### Acceptance criteria
- [x] UI tests and typecheck pass.
- [x] Full DoD passes with exact Node toolchain.
- [x] Fake runtime live smoke passes and no `release_verdict_*` artifacts are created.
- [x] Source include/exclude persists to existing manifest fields without schema/spec changes.
- [x] Checked core screens have no actionable interactive controls below desktop/mobile minimums.

### Risks
- Live smoke and manual QA depend on local browser/toolchain availability; classify host blockers separately instead of weakening the harness.
- CSS sizing changes can affect dense tables/tabs; rendered desktop/mobile QA must inspect overflow and wrapping.

### Progress log
- 2026-07-07: Started P1 UX hardening slice; implemented guided analysis scope, readiness gating, fake runtime labels, TabNav semantics, control tokens and StageRail scroll behavior.
- 2026-07-07: Full DoD passed with `ACP_NODE_TOOL_CANDIDATES=/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin make contracts`, `make test`, `make lint`, `make build`.
- 2026-07-07: Final bounded fake live smoke passed against `http://127.0.0.1:18191` with `UI_E2E_OUTPUT_DIR=/tmp/provenarch-ui-ux-fix-final2-results` and run `run_20260707_182109_001`; manual rendered audit evidence is in `/tmp/provenarch-ui-ux-fix-final2-manual/manual-rendered-audit.json`.
- 2026-07-07: Archived the completed plan so `docs/PLANS.md` keeps only active workstreams.

### Plan ID
EP-20260707-ux-live-smoke-review

### Context
The current task is a diagnostic UX review pass over Console V2: capture present rendered behavior with a small live browser E2E smoke, classify host/tool blockers separately from product UX, and propose focused design improvements. This is not a release readiness run and must not edit canonical live matrices, contracts, schemas, or provider presets.

### Goals (must have)
- [x] Run or explicitly classify the minimal frontend UX live smoke from `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.
- [x] Capture current UI state evidence for Source, Readiness, Analysis, Review, Publish and mobile Review where available.
- [x] Inspect the rendered UI enough to identify high-impact workflow, hierarchy, state-coverage and responsive issues.
- [x] Produce a concise current-state report and prioritized UX/design fixes.

### Non-goals
- [ ] Do not change product UI/API behavior in this pass.
- [ ] Do not run a canonical release gate or treat diagnostic smoke as release readiness.
- [ ] Do not edit canonical release matrices, curated repo presets, schemas or runtime contracts.

### Approach
1) Read required project/runbook context and UI smoke contract.
2) Check host prerequisites, especially exact Node/npm, Playwright browser availability and fake-runtime UI path.
3) Run the smallest feasible frontend live smoke or record the precise operational blocker.
4) Review screenshots/test evidence and current UI implementation shape.
5) Write current-state findings and prioritize a small reviewable UX improvement slice.

### Files changed
- `docs/PLANS.md`
- `docs/archive/audits/ux_current_state_20260707.md`

### Acceptance criteria
- [x] Host/tool readiness and smoke result are recorded with evidence paths.
- [x] UX findings are prioritized by operator workflow impact.
- [x] Proposed fixes are scoped and do not require schema/contract changes unless explicitly called out.

### Risks
- Exact Node.js `22.21.1` may be missing on the host; classify as `precheck_failed`/host blocker rather than bypassing exact-toolchain policy.
- Existing generated reports/screenshots may be local diagnostic artifacts and should not be treated as release evidence.

### Progress log
- 2026-07-07: Started diagnostic UX live smoke review. Initial preflight found current PATH Node.js `25.9.0`, while `.node-version` requires `22.21.1`.
- 2026-07-07: Downloaded exact diagnostic Node.js `22.21.1` to `/tmp/provenarch-node-v22.21.1`, ran `make build`, started fake-runtime `acp serve`, and passed `ui/e2e/live-flow.spec.ts` with `UI_E2E_QA_SMOKE=1`.
- 2026-07-07: Captured current state in `docs/archive/audits/ux_current_state_20260707.md` with screenshot paths, API evidence, findings and proposed UX fixes.
- 2026-07-07: Archived the completed diagnostic plan during the Console V2 UX hardening slice so `docs/PLANS.md` keeps only active workstreams.

### Plan ID
EP-20260702-active-plan-reconciliation

### Context
MVP product work is complete within the current beta boundary, but `docs/PLANS.md` still listed several implementation-complete plans as active because their only remaining item was archive or post-release bookkeeping. This reconciliation archives those completed plans while preserving newer open live-quality/release-validation plans from `origin/main`.

### Goals (must have)
- [x] Replace stale queue wording that treated completed implementation work as ordinary next backlog.
- [x] Move implementation-complete/archive-only plans out of the active section.
- [x] Keep open live-quality/release-validation, GitHub admin residuals and cleanup owner decisions visible as active residual workstreams.
- [x] Avoid product code, schemas, runtime contracts, fixtures, canonical matrices, curated repos and runbook behavior changes.

### Non-goals
- [x] Do not run trusted live validation.
- [x] Do not imply owner/admin cleanup decisions.
- [x] Do not claim canonical release readiness without verifier-backed release verdict evidence.

### Approach
1) Reclassify completed implementation plans as archived evidence.
2) Keep open live-quality/release-validation and owner/admin plans active.
3) Preserve active rules that owner/trusted-host tasks are not ordinary backlog work.
4) Run docs-sync and diff checks.

### Files changed
- `docs/PLANS.md`
- `docs/archive/PLANS_ARCHIVE_2026-07.md`

### Acceptance criteria
- [x] Active queue no longer points agents at completed Epic 17 or historical UI implementation work as next engineering work.
- [x] Completed implementation plans are archived in July 2026 archive.
- [x] Active plans have real open residual goals.
- [x] Trusted live/release gates remain blocked until owner/trusted-host prerequisites are satisfied.

### Progress log
- 2026-07-02: Archived implementation-complete plans from `docs/PLANS.md`, kept open live-quality/release-validation and owner/admin plans active, and recorded this reconciliation slice as closed archive evidence.

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

---

## EP-20260720-epic18-architecture-home-inline-heading-repair

### Context
Qualification smoke from clean merge SHA `8889f059` completed collect `10/10`, then step2 rejected a
substantive Architecture Home because the provider placed each required H2 label and its body on the
same physical line. Markdown correctly parsed the entire line as a heading, so all eight canonical
sections appeared missing and the subsequent provider enrichment stalled. This is a product-level
document-shape defect exposed by live evidence; the regression and remediation remain entirely
provider-free and independent of matrix identity or live-harness state.

### Completed
- [x] Recover only the exact all-eight inline H2/body shape in canonical order with no other error.
- [x] Preserve authored content, insert only Markdown boundaries, write atomically and revalidate.
- [x] Restore original bytes and retain provider repair/fail-closed behavior on every ambiguity.
- [x] Add provider-free fixture, focused/stress regressions, diagnostics and synchronized docs.
- [x] Pass full deterministic DoD, merge PR #166 and restart R3 from merge SHA `5718f3f4`.

### Results
- Focused recovery passed 20/20; full deterministic DoD passed with pinned Node `22.21.1`.
- The post-merge smoke reached init/refresh collect `10/10`, validating the remediation before the
  next independent concrete-evidence-reference defect was isolated.

---

### Plan ID
EP-20260720-epic18-collect-manifest-task-identity

### Context
Standalone `release fast` from qualification SHA `aea3950a` reached Qwen collect `10/10`, but one
otherwise valid provider-authored shard manifest carried an `artifact_root` for a different spelling
of the public task run identity. The referenced document existed under the actual task `write_root`,
so file validation passed and downstream step2 later tried to resolve the foreign root. This is a
product contract gap: a collect manifest must describe the task that authored it. The live harness
only exposed the defect and remains outside the product implementation and provider-free regression.

### Goals (must have)
- [x] Require exact manifest `run_id`, `step_id`, `shard_id`, `domain_id`, and `artifact_root`
      equality with the current collect task before provider success.
- [x] Repeat the same validation defensively before orchestrator materialization.
- [x] Add a provider-free fixture and regressions for rejection plus ordinary provider repair.
- [x] Synchronize architecture, pipeline, testing, and live-gate documentation.
- [x] Pass focused stress coverage and full deterministic DoD.
- [x] Merge the remediation and restart qualification from the new clean merge commit.

### Non-goals
- [x] Do not change schemas, HTTP API, provider contracts, retry budgets, or canonical matrices.
- [x] Do not normalize or manually rewrite a provider manifest in the live harness.
- [x] Do not expose matrix IDs, release verdicts, assessments, or live environment in product code.

### Acceptance criteria
- [x] A structurally valid manifest with a foreign task identity is rejected before step2.
- [x] Focused provider repair can replace the manifest and pass the same strict validation.
- [x] Existing manifest values and document content are not synthesized or normalized by ACP.
- [x] Product/runtime boundary tests and full deterministic DoD pass.

### Progress log
- 2026-07-20: Stopped `release-fast-20260720T104426Z` after the first run failure and isolated an
  exact `artifact_root` identity mismatch while the authored document remained present under the
  actual task write root. No live artifact was edited or copied into product behavior.
- 2026-07-20: Added shared task-identity validation at the provider acceptance and orchestrator
  materialization boundaries with a provider-free regression fixture.
- 2026-07-20: Proved provider-authored repair 20/20, preserved selective-refresh baseline packs
  byte-identically by validating them against their persisted authoring execution, and passed full
  deterministic DoD: contracts, Go/API, 261 Python boundary tests, 142 UI tests, lint, and build.
- 2026-07-20: PR #165 squash-merged; qualification restarted from clean merge SHA `8889f059`.

---

## EP-20260718-epic18-proposal-staging-path-guard

### Goal
- Prevent current taskrun staging locators from leaking into user-visible proposal/changelog navigation.

### Implementation
- [x] Reject `reports/taskruns/**`, `staging/final/**` and `staging/shards/**` in step4 markdown.
- [x] Keep staging paths as provider input while requiring canonical report paths, finding/citation IDs and stable repo evidence in output.
- [x] Add live-observed fixture coverage and a bounded provider-authored marker-cleanup retry.
- [x] Synchronize architecture, testing and release-runbook contracts.

### Results
- Focused recovery stress and the full deterministic DoD passed on Node 22.21.1.
- PR #160 merged as `cea5fe99`; the subsequent independent Architecture Home recovery gap is tracked by the active R3 remediation plan.

---

## EP-20260718-epic18-architecture-home-repair

### Goal
- Preserve strict rejection of runtime/process narration while allowing one non-recursive,
  provider-authored cleanup after every step2 markdown target was freshly rewritten.

### Results
- Added the live-observed `scoped to the current run` fixture and strict recovery regression.
- Focused recovery passed 20/20 and full deterministic DoD passed with Go 1.25.10, Node 22.21.1
  and npm 10.9.4 (261 Python tests, 142 UI tests).
- PR #161 merged; post-merge smoke proved the cleanup stage during init. The independent empty
  draft-sidecar write-set blocker is tracked by the active R3 remediation plan.

---

## EP-20260718-epic18-step2-silent-enrichment-retry

### Context
Canonical Claude Bank smoke `smoke-tiny-bank-20260718T161023Z` produced substantive step2 drafts
that strict validation rejected for runtime-process narration and nonexistent repository references.
The first focused enrichment stalled silently before artifact mutation, but existing bounded
write-first and compact retries were reachable only for bootstrap/no-op errors.

### Completed scope
- Routed silent pre-artifact, no-fresh-mutation enrichment stalls through the existing bounded
  write-first and compact step2 retries for any proven strict draft validation failure.
- Preserved strict validation, retry counts, activity budgets, schemas, provider contracts and
  canonical matrices.
- Added non-recursion, valid-artifact, provider-output, post-artifact and fresh-mutation exclusions.
- Synchronized architecture/testing documentation and completed full deterministic DoD.

### Results
- Focused helper stress passed 100/100; providercommon and promptcontract packages passed.
- Full DoD passed on pinned Node 22.21.1: contracts, Go tests, 261 Python tests, 142 UI tests,
  shellcheck/typecheck, deterministic UI build and Go build.
- PR #159 merged as `4febe970`; canonical Claude smoke
  `smoke-tiny-bank-20260718T174204Z` passed with init and refresh collect both at 10/10.

---

### Plan ID
## EP-20260715-20M-workbench-module-seams

### Goal
- Move Changes/Knowledge/Publish composition behind typed feature view models while keeping HTTP, Git and workflow behavior unchanged.

### Implementation
- [x] Add pure Change Review, Knowledge and Publish view models with table-driven tests.
- [x] Make Changes URL view/source the composition source of truth and remove panel selection from `App.tsx`.
- [x] Preserve existing data loading/actions in `App.tsx` and current operator-visible behavior.
- [x] Correct current-shell documentation drift.

### Acceptance
- Typecheck and focused workbench tests pass; no feature module imports `App.tsx`; no public contract changes.

---

## EP-20260715-20K-semantic-ui-density

### Goal
- Consolidate the accepted Epic 20 UI around semantic tokens, shared hierarchy/state primitives and comfortable/compact density.

### Implementation
- [x] Add semantic color/context, type, spacing, radius, control and motion tokens with an explicit system-font contract.
- [x] Add Button, PageHeader, ContextBar, RecoveryPanel, Metric/DefinitionList, RouteTabs and AsyncState primitives.
- [x] Apply shared hierarchy to Home, Guided Setup and Changes without changing workflow semantics.

### Acceptance
- Primitive state/route tests, typecheck, focus styling and reduced-motion behavior pass.

---

## EP-20260715-20L-responsive-shell

### Goal
- Deliver wide/collapsed/bottom navigation and modal/non-modal context behavior without document-level overflow.

### Implementation
- [x] Limit PrimaryNav to Home/Runs/Knowledge/Changes and keep Setup contextual.
- [x] Add expanded/collapsed desktop, compact desktop and tablet/phone bottom navigation layouts.
- [x] Add persistent wide ContextDrawer and focus-managed modal/fullscreen variants below 1280px.
- [x] Add first-viewport hierarchy, long-content wrapping and local-scroll rules.

### Acceptance
- Component tests cover drawer modality/Escape; rendered viewport gates run in mock Playwright.

---

## EP-20260715-20N-epic20-task-exit

### Goal
- Retire the unreachable Console V2 shell and close Epic 20 through deterministic task, accessibility, responsive and documentation gates.

### Implementation
- [x] Delete unused AppShell/StageRail/top-status/active-strip/inspector/activity modules and their legacy primitive tests.
- [x] Preserve deterministic scenarios for snapshot isolation, runtime/demo identity, coordination, partial evidence, Git scope and Ask citation return.
- [x] Synchronize README, ARCHITECTURE, TESTING_STRATEGY and stakeholder status with the implemented shell.
- [x] Complete full DoD and mock viewport evidence before marking the epic exit verified.

### Acceptance
- Full deterministic DoD and mock E2E pass; Epic 18 R3 remains a separate trusted-machine release gate.

### Results
- `make contracts`, `make test`, `make lint` and `make build` pass with the pinned Node 22 toolchain; the UI unit suite reports 140 passing tests.
- The deterministic mock E2E gate reports 7 passed / 0 skipped and writes current-behavior evidence under `docs/assets/ui-product-shell/` without replacing target-reference assets.
- Epic 20 is complete. Epic 18 R3 remains the independent trusted-machine release-readiness gate.

---

### Plan ID
EP-20260721-epic18-step2-mixed-recovery-routing

### Context
Clean standalone release-fast `release-fast-20260721T094341Z` completed Qwen init/refresh and
Claude collect `10/10`, then failed closed at Claude `init.step2.asis_docs`. Provider-authored
markdown simultaneously retained Architecture Home process narration and a stale downstream-index
availability claim. The shared retry selector incorrectly treated the mixed validation result as a
downstream-index-only failure; that focused retry made no fresh writes and exhausted after the
canonical pre-artifact window. This is provider-independent recovery routing, not live-harness
behavior.

### Goals (must have)
- [x] Require the specialized downstream-index retry to receive only downstream-index validation
      problems.
- [x] Route a mixed Architecture Home + downstream-index result to the existing Architecture Home
      cleanup path, which still performs a provider-authored rewrite and full strict validation.
- [x] Preserve the live-observed mixed failure as a provider-free fixture and regression.
- [x] Pass focused stress and the full deterministic DoD.
- [x] Merge the remediation in PR #172 (`a633e3ce`); R3 restart is now gated by Epic 22.

### Non-goals
- [x] Do not sanitize or synthesize provider markdown in ACP.
- [x] Do not weaken draft validation or accept stale downstream/runtime narration.
- [x] Do not change schemas, HTTP API, provider contracts, timeout/retry budgets or matrices.
- [x] Do not add matrix identity, release verdicts or live environment behavior to product code.

### Acceptance criteria
- [x] Homogeneous downstream-index errors still select `draft_artifact_enrichment_downstream_index_retry`.
- [x] Mixed Architecture Home/downstream errors do not select the downstream-only retry and recover
      through `draft_artifact_enrichment_architecture_home_cleanup` in a provider-free test.
- [x] Focused routing tests pass 20 consecutive runs.
- [x] `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-21: Stopped the failed release-fast immediately after Claude terminal failure; a Codex
  slot started by the batch harness was interrupted and is not evidence. Product and all pinned
  canonical source checkouts remained clean.
- 2026-07-21: Added homogeneous-error selection and a mixed step2 fixture; no live identifiers or
  provider-specific conditions are present in the implementation.
- 2026-07-21: Focused mixed/pure routing regression passed 20 consecutive runs in 91.3 seconds.
- 2026-07-21: Full deterministic DoD passed: contracts, Go/Python/UI tests, lint/typecheck and
  production build all completed successfully with the pinned toolchains.
- 2026-07-21: PR #172 merged as `a633e3ce`; no stopped matrix was promoted to release evidence.

### Plan ID
EP-20260721-epic18-step2-exact-evidence-references

### Context
After PR #170 fixed valid-artifact stall accounting, clean qualification
`smoke-tiny-bank-20260721T004350Z` reached init collect `10/10` and reported only actual invalid/
repair stalls. `init.step2.asis_docs` then failed closed because the provider shortened the observed
nested evidence path `src/ledger/cloudbuild.yaml` to unavailable root `cloudbuild.yaml`; focused
repair subsequently emitted an unterminated inline Python string. This is a general evidence-
identity and command-construction defect, not live-harness behavior.

### Goals (must have)
- [x] Build a deterministic bounded allowlist from exact repo/path identities in valid current-run
      shard-manifest citations and semantic provenance.
- [x] Put the same allowlist and no-shortening rule in normal, repair, enrichment, compact and
      command-text step2 prompt paths.
- [x] Require repository-reference repair to use direct literal heredocs rather than inline
      generated programs.
- [x] Preserve the Bank-shaped nested-path failure in provider-free regressions.
- [x] Pass focused stress and the full deterministic DoD.
- [x] Merge the remediation in PR #171 (`ca8c3f67`); R3 restart is now gated by Epic 22.

### Non-goals
- [x] Do not synthesize or rewrite Architecture Home prose in ACP.
- [x] Do not weaken strict repository-reference validation or accept guessed paths.
- [x] Do not change schemas, HTTP API, provider contracts, timeout/retry policy or matrices.
- [x] Do not add live matrix identity or environment behavior to product code/tests.

### Acceptance criteria
- [x] Exact nested citation `bank-of-anthos:src/ledger/cloudbuild.yaml` appears in guidance while
      inferred `bank-of-anthos:cloudbuild.yaml` does not.
- [x] Allowlist output is deterministic and ignores invalid/foreign-run manifests.
- [x] Missing-reference focused repair forbids Python/Node/template assembly and requires literal
      writes plus full strict revalidation.
- [x] `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-21: Stopped the failed matrix immediately after step2. Product and pinned Bank trees
  remained clean; the failed matrix is not reusable release evidence.
- 2026-07-21: Added exact current-run evidence-reference guidance and provider-free prompt tests;
  strict artifact validation remains unchanged and authoritative.
- 2026-07-21: New step-policy/prompt regressions passed 20 repetitions; the complete required DoD
  passed with Go 1.25.10, Node 22.21.1, npm 10.9.4, 261 Python tests and 142 UI tests. A deliberately
  over-broad 20x run of the entire long providercommon package reached its aggregate 10-minute
  test timeout; the package passed normally in both full DoD runs, so no unrelated runtime/test
  budget was changed.

---

### Plan IDs
- `EP-20260726-epic22a-transactional-run-history`
- `EP-20260726-epic22b-symlink-safe-workspace`
- `EP-20260726-epic22c-server-snapshot-resolver`
- `EP-20260726-epic22d-one-pathscope-dialect`
- `EP-20260726-epic22e-immutable-refresh-baseline`
- `EP-20260726-epic22f-session-git-admission-lease`
- `EP-20260726-epic22g-typed-recovery-routing`
- `EP-20260726-epic22h-provider-free-artifact-auditor`
- `EP-20260726-epic22i-live-product-boundary`
- `EP-20260726-epic22j-changes-workflow-truth`
- `EP-20260726-epic22k-url-request-identity`
- `EP-20260726-epic22l-evidence-authorities`
- `EP-20260726-epic22m-evidence-viewer-correctness`
- `EP-20260726-epic22n-responsive-accessibility`

### Result
- Epic 22 implementation slices `22A`–`22N` closed the run-history, filesystem, snapshot,
  path-scope, refresh-baseline, admission, recovery, artifact-audit, live/product isolation,
  Changes, request-identity, evidence-authority, Evidence Viewer and responsive/accessibility
  findings.
- Every slice received focused provider-free regression coverage and documentation synchronization.
  The combined `make offline-closure` subsequently passed; live E2E was intentionally not run.
- `EP-20260726-epic22o-offline-closure` remains active only to record a clean reviewed qualification
  SHA before Epic 18 R3.

---

### Plan ID
EP-20260726-k2b-advisory-workspace-health

### Result
- Workspace Health v1 remains read-only/advisory and now reports deterministic link, model,
  ownership, orphan, malformed-document, finding, proposal-evidence and low-citation-coverage issues.
- Current Knowledge exposes the summary; historical Changes does not import current health.

---

### Plan ID
EP-20260726-k4-citation-claim-hardening

### Result
- Existing public shapes now enforce run isolation, global citation/claim identity and reciprocity,
  concrete contained repository evidence, key-document completeness and safe selective preservation.
- Focused fault/path fixtures and the complete provider-free DoD passed without adding a claim ledger
  or publication policy.

---

### Plan ID
EP-20260726-k3a-qa-proposal-draft

### Result
- Succeeded immutable QA answers expose a digest and can create one exclusive atomic
  proposal/evidence/source package through the typed proposal-draft endpoint.
- The additive `source-qa-answer` schema, parser, examples, fixtures, API/spec/appendix and ADR are
  synchronized; stale, duplicate, traversal, citation and rollback cases are covered.
- K3B completed the focus-managed confirmation, stale-answer recovery, Git refresh,
  Changes→Proposals navigation and Return to Ask flow on top of that mutation.

---

### Plan ID
EP-20260726-9d-qa-v1-compatibility

### Result
- Deterministic `acp qa` and `POST /api/qa/ask` retain their exact compatibility contract through
  v1, with async start/poll migration documented and no removal or deprecation headers scheduled.

---

### Plan ID
EP-20260726-cleanup-readable-trackers

### Result
- `make verify-readable-fixtures` protects all 90 retained human-readable fixture exports by path
  and digest while machine snapshots remain authoritative.
- Implementation-complete child plans were archived and backlog/stakeholder/program state was
  reconciled. The clean qualification SHA was subsequently recorded by the archived `22O` handoff;
  live R3 remains explicitly open.

---

### Plan ID
EP-20260726-epic22o-offline-closure

### Result
- Added one provider-free `make offline-closure` command covering race/fault/path/auditor,
  live-product boundary, readable-fixture drift, UI unit/rendered mock, contracts, full tests, lint,
  build, embedded UI and source-repository cleanliness.
- The gate passed from isolated clean commit
  `e8055d65699ed63623f62ad99c3b8406f79c030d` with 263 Python tests, 158 UI tests and 7 Playwright
  scenarios and left no tracked drift.
- Live E2E was intentionally not run. Epic 18 R3 remains active and must use this exact code input.

---

### Plan ID
EP-20260728-r3-provenance-kind-shape-recovery

### Result
- Fresh smoke `smoke-tiny-bank-20260728T065407Z` exposed otherwise valid collect semantic objects
  using the unambiguous lexical aliases `observed`, `inferred` and `asserted`.
- PR #177 added exact-alias-only atomic recovery under unchanged full schema/repository-evidence
  validation, rollback, digest/count diagnostics, runtime warning and readable input/golden fixtures.
- Full DoD/offline closure and all 11 PR checks passed; PR #177 merged as
  `e54b4ce6b2d809d56d2de8c1c369e19724a3b7b3`.
- The merge SHA passed a fresh detached-worktree `make offline-closure`, and R3 restarted as
  `smoke-tiny-bank-20260728T083406Z`. That smoke found a separate Qwen tool-first prompt defect, so
  no stopped/partial matrix evidence was promoted.

---

### Plan ID
EP-20260728-r3-qwen-tool-first-collect

### Result
- PR #178 replaced Qwen normal and first pre-artifact focused collect prompts with a bounded
  `read_file -> write_file` contract, exact provenance enum and provider-isolation regressions.
- Full DoD/offline closure and all 11 PR checks passed; PR #178 merged as
  `e9025647d13690b5ea236d14fc476bc89b556f12`.
- The merge SHA passed fresh detached-worktree `make offline-closure`. Fresh smoke
  `smoke-tiny-bank-20260728T101010Z` proved the bounded reads and markdown write, but Qwen waited for
  that tool result and then exhausted the partial-artifact window before writing the manifest.
- Runtime reconstruction made the shard recovery-heavy, so the smoke was stopped and no partial
  evidence was promoted. Same-response atomic pair writing supersedes this plan.

---
