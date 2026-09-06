# Archived planning evidence — September 2026 reconciliation

This archive preserves completed engineering/audit/design plans and superseded tracker snapshots
moved from `docs/PLANS.md` on 2026-09-05. It is historical evidence, not the active queue or current
product/release status. See the [active plan index](../PLANS.md#active-plan-index) and
[canonical stakeholder matrix](../STAKEHOLDER_DOC.md#0-canonical-stakeholder-matrix-source-of-truth).

The merged Epic 19 program and its child scopes are closed by the final `19Z` reconciliation
recording merge `02716bb`; old “continue the next slice” or archive-only checkboxes below describe
that earlier sequence. Other moved scopes have explicit completed acceptance, supersession or
archive-only follow-up. Original bodies/checklists/logs are retained, including historical failures.
One relative `docs/archive/audits/DESIGN_QA_2026-08-21.md` link is rebased for the new directory. Archiving implementation evidence
does not turn failed/stopped diagnostics into qualification or close any parent R3/W25 gate.
Owner-review/admin decisions and incomplete release/DoD criteria remain in the active file.
The 2026-09-05 remediation program, its dependencies and REM-25 remain unchanged and unexecuted.

## Archive index

| Original plan or note | Reason for relocation |
| --- | --- |
| [EP-20260905-agent-development-revision](#ep-20260905-agent-development-revision) | Completed owner-approved local revision; full DoD and eight rendered mock scenarios passed. |
| [EP-20260804-agents-gpt-5-6](#ep-20260804-agents-gpt-5-6) | Superseded by the owner-approved agent-development revision; implementation had passed its DoD and only archive/review bookkeeping remained. |
| [EP-20260801-api-test-server-shutdown](#ep-20260801-api-test-server-shutdown) | Explicit completed/superseded scope; only archive bookkeeping remained. |
| [EP-20260719-epic18-draft-empty-sidecar-rollback](#ep-20260719-epic18-draft-empty-sidecar-rollback) | Explicit completed/superseded scope; only archive bookkeeping remained. |
| [EP-20260715-architecture-change-review-ui-migration-wave](#ep-20260715-architecture-change-review-ui-migration-wave) | Explicit completed/superseded scope; only archive bookkeeping remained. |
| [EP-20260715-architecture-change-review-design-baseline](#ep-20260715-architecture-change-review-design-baseline) | Explicit completed/superseded scope; only archive bookkeeping remained. |
| [EP-20260714-ui-ux-redesign-exploration](#ep-20260714-ui-ux-redesign-exploration) | Explicit completed/superseded scope; only archive bookkeeping remained. |
| [EP-20260713-live-e2e-codex-model-pin](#ep-20260713-live-e2e-codex-model-pin) | Explicitly superseded by the retained runtime-model-selection plan. |
| [EP-20260717-epic18-r3-missing-findings-shape-recovery](#ep-20260717-epic18-r3-missing-findings-shape-recovery) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260713-epic-19-pr1-remediation](#ep-20260713-epic-19-pr1-remediation) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19b-transactional-promotion](#ep-20260713-epic-19-19b-transactional-promotion) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19c-async-panic-isolation](#ep-20260713-epic-19-19c-async-panic-isolation) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19d-server-owned-shutdown](#ep-20260713-epic-19-19d-server-owned-shutdown) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19e-coherent-service-generation](#ep-20260713-epic-19-19e-coherent-service-generation) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19f-fresh-git-url-resolution](#ep-20260713-epic-19-19f-fresh-git-url-resolution) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19g-minimum-collect-evidence-contract](#ep-20260713-epic-19-19g-minimum-collect-evidence-contract) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19h-symmetric-document-citation-validation](#ep-20260713-epic-19-19h-symmetric-document-citation-validation) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19i-historical-run-artifact-snapshots](#ep-20260713-epic-19-19i-historical-run-artifact-snapshots) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260713-epic-19-19j-request-scoped-ui-detail-state](#ep-20260713-epic-19-19j-request-scoped-ui-detail-state) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19k1-run-mutation-acknowledgement](#ep-20260714-epic-19-19k1-run-mutation-acknowledgement) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19k2-qa-provisional-run-ordering](#ep-20260714-epic-19-19k2-qa-provisional-run-ordering) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19l-editor-revision-safety](#ep-20260714-epic-19-19l-editor-revision-safety) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19m-deterministic-embedded-ui-bundle](#ep-20260714-epic-19-19m-deterministic-embedded-ui-bundle) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19o-locked-contract-validator-toolchain](#ep-20260714-epic-19-19o-locked-contract-validator-toolchain) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19n-composite-release-verdict-gate](#ep-20260714-epic-19-19n-composite-release-verdict-gate) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19p-step1-card-enrichment](#ep-20260714-epic-19-19p-step1-card-enrichment) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19q-generic-refresh-semantic-guard](#ep-20260714-epic-19-19q-generic-refresh-semantic-guard) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19r1-accessible-tabs-controller](#ep-20260714-epic-19-19r1-accessible-tabs-controller) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19r2-keyboard-path-combobox](#ep-20260714-epic-19-19r2-keyboard-path-combobox) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19r3-accessible-async-announcements](#ep-20260714-epic-19-19r3-accessible-async-announcements) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19s1-shell-dead-code-cleanup](#ep-20260714-epic-19-19s1-shell-dead-code-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19s2-shellcheck-lint](#ep-20260714-epic-19-19s2-shellcheck-lint) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19s3-required-pr-lint](#ep-20260714-epic-19-19s3-required-pr-lint) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19t-logs-smoke-coverage](#ep-20260714-epic-19-19t-logs-smoke-coverage) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19u-deterministic-mock-playwright-ci](#ep-20260714-epic-19-19u-deterministic-mock-playwright-ci) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19u2-ui-v8-coverage-baseline](#ep-20260714-epic-19-19u2-ui-v8-coverage-baseline) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19v-python-runtime-pinning](#ep-20260714-epic-19-19v-python-runtime-pinning) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w1-runtime-draft-wrapper-cleanup](#ep-20260714-epic-19-19w1-runtime-draft-wrapper-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w2-sharding-wrapper-cleanup](#ep-20260714-epic-19-19w2-sharding-wrapper-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w3-provider-argument-wrapper-cleanup](#ep-20260714-epic-19-19w3-provider-argument-wrapper-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w4-docflow-compatibility-helper-cleanup](#ep-20260714-epic-19-19w4-docflow-compatibility-helper-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w5a-review-diff-residual-cleanup](#ep-20260714-epic-19-19w5a-review-diff-residual-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w5b-model-store-residual-cleanup](#ep-20260714-epic-19-19w5b-model-store-residual-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w5c-orchestrator-quality-residual-cleanup](#ep-20260714-epic-19-19w5c-orchestrator-quality-residual-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w5d-reports-compiler-residual-cleanup](#ep-20260714-epic-19-19w5d-reports-compiler-residual-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19w5e-prompt-contract-residual-cleanup](#ep-20260714-epic-19-19w5e-prompt-contract-residual-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19x-ui-dead-surface-cleanup](#ep-20260714-epic-19-19x-ui-dead-surface-cleanup) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260714-epic-19-19z-final-reconciliation](#ep-20260714-epic-19-19z-final-reconciliation) | Merged Epic 19; 19Z records 02716bb and completed reconciliation. |
| [EP-20260715-21A-architecture-home](#ep-20260715-21a-architecture-home) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-21B-source-revisions](#ep-20260715-21b-source-revisions) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-21C-refresh-impact-plan](#ep-20260715-21c-refresh-impact-plan) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [Live E2E Step2 Typed Shard Completeness Hardening](#live-e2e-step2-typed-shard-completeness-hardening) | All implementation and acceptance items checked; broader live follow-ups retained. |
| [Artifact Quality Excellent Gate: Proposals/Findings Linkage](#artifact-quality-excellent-gate-proposalsfindings-linkage) | All implementation and acceptance items checked; broader live follow-ups retained. |
| [UX/UI Iteration: Live Shard Scope Mapping](#uxui-iteration-live-shard-scope-mapping) | All implementation and acceptance items checked; broader live follow-ups retained. |
| [UX/UI Iteration: Active Provider Stream Diagnostics](#uxui-iteration-active-provider-stream-diagnostics) | All implementation and acceptance items checked; broader live follow-ups retained. |
| [UX/UI Iteration: Onboarding Recovery Rendered QA](#uxui-iteration-onboarding-recovery-rendered-qa) | All implementation and acceptance items checked; broader live follow-ups retained. |
| [Implemented vs Planned (operational mirror)](#implemented-vs-planned-operational-mirror) | Obsolete duplicate status snapshot; canonical status stays in STAKEHOLDER_DOC. |
| [EP-20260710-code-quality-audit](#ep-20260710-code-quality-audit) | All audit/planning items checked; results and acceptance preserved below. |
| [EP-20260715-20F2-deliberate-queue-ui](#ep-20260715-20f2-deliberate-queue-ui) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-20I2-deep-url-context](#ep-20260715-20i2-deep-url-context) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-20J1-home-guided-setup-runs](#ep-20260715-20j1-home-guided-setup-runs) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-20J2-changes-knowledge-ask](#ep-20260715-20j2-changes-knowledge-ask) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-21D-explainable-no-op](#ep-20260715-21d-explainable-no-op) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-21E-affected-collect](#ep-20260715-21e-affected-collect) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-21F-surgical-materialization](#ep-20260715-21f-surgical-materialization) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260715-21G-operator-explanation](#ep-20260715-21g-operator-explanation) | Completed implementation acceptance; Epic 18 release qualification stays open. |
| [EP-20260716-epic18-r3-qwen-artifact-readiness](#ep-20260716-epic18-r3-qwen-artifact-readiness) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260716-epic18-r3-qwen-stream-repair](#ep-20260716-epic18-r3-qwen-stream-repair) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260716-epic18-r3-step2-home-repair](#ep-20260716-epic18-r3-step2-home-repair) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260716-epic18-r3-qwen-tool-first-retry](#ep-20260716-epic18-r3-qwen-tool-first-retry) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260710-code-audit-remediation-backlog](#ep-20260710-code-audit-remediation-backlog) | All audit/planning items checked; results and acceptance preserved below. |
| [EP-20260718-epic18-r3-architecture-home-staging-reference](#ep-20260718-epic18-r3-architecture-home-staging-reference) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260718-epic18-r3-architecture-home-run-narration](#ep-20260718-epic18-r3-architecture-home-run-narration) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260718-epic18-r3-architecture-home-runtime-checkout](#ep-20260718-epic18-r3-architecture-home-runtime-checkout) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260718-epic18-r3-command-stream-drain](#ep-20260718-epic18-r3-command-stream-drain) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260718-epic18-r3-architecture-home-shard-recap](#ep-20260718-epic18-r3-architecture-home-shard-recap) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260718-epic18-r3-stream-only-fixture-budget](#ep-20260718-epic18-r3-stream-only-fixture-budget) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260718-epic18-r3-architecture-home-evidence-paths](#ep-20260718-epic18-r3-architecture-home-evidence-paths) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260718-epic18-r3-orchestrator-lifecycle-fixture-budget](#ep-20260718-epic18-r3-orchestrator-lifecycle-fixture-budget) | Implementation/tests complete in recorded evidence; parent R3 gate stays open. |
| [EP-20260821-task-first-design-qa-closure](#ep-20260821-task-first-design-qa-closure) | Q10 acceptance and 2026-08-23/25 closure logs; new remediation program remains separate. |
| [Historical continuous backlog queue policy (July 2026)](#historical-continuous-backlog-queue-policy-july-2026) | Superseded queue policy; active work and status now route to their owning documents. |

## Preserved source blocks

<a id="ep-20260801-api-test-server-shutdown"></a>

### Plan ID
EP-20260801-api-test-server-shutdown

### Context
Parser fix PR #195 passed the complete local provider-free gate, but two independent backend CI
attempts failed while Go test cleanup removed `reports/taskruns`: the assertions had completed, yet
the shared test helper had left its orchestrator service alive after closing only the HTTP test
server. A background terminal history write could therefore race with `t.TempDir` cleanup. This is
a provider-free test lifecycle defect and is intentionally isolated from the parser change.

### Goals (must have)
- [x] Make every shared API test server stop its orchestrator before its temporary workspace is
      removed.
- [x] Repeat the affected parallel endpoint tests and complete the deterministic DoD/offline gate.
- [x] Merge the lifecycle fix, update PR #195 onto the new `main`, and restart its CI.
- [ ] Archive this completed lifecycle slice during the post-R3 tracker cleanup.

### Non-goals
- [x] Do not change production lifecycle behavior, API contracts, schemas, timeouts, canonical live
      matrices or release evidence.
- [x] Do not weaken TempDir cleanup or hide shutdown errors.

### Approach
1. Register one cleanup callback in the shared API test-server constructors.
2. Rely on `testing.T.Cleanup` LIFO ordering so service shutdown runs before the already-registered
   TempDir removal.
3. Bound shutdown with the production service close budget and report any shutdown error as a test
   failure.

### Files expected to change
- `internal/api/server_test.go`
- `docs/PLANS.md`

### Acceptance criteria
- [x] The two CI-failing cancellation endpoint tests pass repeatedly and without cleanup errors.
- [x] The complete API package, provider-free DoD and offline closure pass.
- [x] PR #195 receives green backend CI after incorporating the merged fix.

### Risks
- Some tests already call `Shutdown` explicitly. Service shutdown is idempotent; the registered
  cleanup still provides a uniform final lifecycle boundary for every helper user.

### Progress log
- 2026-08-01: Classified two different `unlinkat reports/taskruns: directory not empty` backend CI
  failures as the same missing test-server shutdown boundary; the corresponding assertions and
  focused local repetitions passed.
- 2026-08-01: Both affected endpoint scenarios passed 20 repetitions; the complete API package
  passed three race-enabled repetitions. Exact Go 1.25.10 / Node 22.21.1 / npm 10.9.4
  `make contracts`, `make test`, `make lint`, `make build`, and a separate
  `make offline-closure` completed with exit code 0. Embedded UI is deterministic, the fix
  worktree contains only the two expected test/docs files, and all 12 pinned source repositories
  remain clean at their curated revisions.
- 2026-08-01: PR #196 passed all 11 CI checks and merged as
  `0d1e146435d61595becd82fd7f8b173467a508ea`. Parser PR #195 incorporated the resulting `main`;
  its post-integration deterministic gate and CI were then repeated from the combined tree.
- 2026-08-01: The integrated parser branch passed the complete local DoD/offline closure again,
  then PR #195 passed all 11 CI checks, including backend in 5m54s, with no review threads.
<a id="ep-20260719-epic18-draft-empty-sidecar-rollback"></a>

### Plan ID
EP-20260719-epic18-draft-empty-sidecar-rollback

### Context
Post-PR #161 Claude smoke `smoke-tiny-bank-20260718T225118Z` proved the new Architecture Home
cleanup during init and completed refresh collect `10/10`. Refresh step2 then failed closed before
that cleanup could run because the provider-authored focused rewrite created an unreferenced empty
file named `amp` under `draft_final_root`. The write-set guard correctly rejected it, but a newly
created zero-byte sidecar can be safely rolled back to the pre-command state without accepting it.

### Goals (must have)
- [x] Roll back only newly created regular zero-byte sidecars inside `draft_final_root`.
- [x] Re-run the full write-set and strict artifact validation after rollback.
- [x] Preserve terminal failure for non-empty, modified/deleted, directory, symlink, write-root,
      traversal and out-of-root mutations.
- [x] Add the live-observed `amp` recovery regression and focused stress coverage.
- [x] Synchronize architecture, pipeline, testing, release-runbook and tracker documentation.
- [x] Pass full deterministic DoD.
- [x] Merge the remediation and restart qualification from the new clean merge commit.
- [ ] Archive this completed remediation during the next tracker reconciliation.

### Non-goals
- [x] Do not add `amp` or arbitrary sidecars to the allowed output set.
- [x] Do not accept invalid Architecture Home content or skip provider-authored cleanup.
- [x] Do not change schemas, HTTP APIs, provider contracts, timeout policy or canonical matrices.
- [x] Do not reuse the stopped smoke as release evidence.

### Acceptance criteria
- [x] A created empty sidecar is absent after rollback and the next strict recovery stage can run.
- [x] Existing forbidden non-empty-file regression remains `runtime_contract_failed`.
- [x] Focused recovery sequence passes 20 consecutive runs.
- [x] `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-19: Preserved bounded failure evidence, confirmed refresh collect `10/10`, and isolated
  the terminal write-set mutation to newly created zero-byte `draft_final_root/amp`.
- 2026-07-19: Added rollback plus strict revalidation and proved the empty-sidecar → Architecture
  Home cleanup sequence 20/20 while the non-empty extra-file case remained terminal.
- 2026-07-19: Full deterministic DoD passed with Go 1.25.10, Node 22.21.1 and npm 10.9.4:
  contracts, full Go, 261 Python, 142 UI, lint and embedded UI build are green.

<a id="ep-20260715-architecture-change-review-ui-migration-wave"></a>

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
- `docs/archive/design/UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md`
- `docs/archive/design/UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md`
- `docs/archive/design/UI_CONSOLE_V2_DESIGN.md`
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
  `docs/archive/design/UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md` with authority rules, all internal and
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

<a id="ep-20260715-architecture-change-review-design-baseline"></a>

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
- `docs/archive/design/UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md`
- `docs/assets/ui-architecture-change-review/*.png`
- `docs/archive/design/UI_CONSOLE_V2_DESIGN.md`
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
- 2026-07-15: Located the historical V2 design/current-shell baseline at `docs/archive/design/UI_CONSOLE_V2_DESIGN.md`, historical
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

<a id="ep-20260714-ui-ux-redesign-exploration"></a>

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

<a id="ep-20260713-live-e2e-codex-model-pin"></a>

### Plan ID
EP-20260713-live-e2e-codex-model-pin

Status: superseded by `EP-20260804-runtime-model-selection`; retained as historical evidence of
the earlier Codex-only environment pin.

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

<a id="ep-20260717-epic18-r3-missing-findings-shape-recovery"></a>

## EP-20260717-epic18-r3-missing-findings-shape-recovery

### Context
- Standalone `release-fast-20260717T114416Z` passed Qwen collection and reached Claude Bank
  collection before the first product blocker.
- Claude authored a non-bootstrap evidence document and otherwise rich manifest, but omitted the
  schema-required `semantic.findings` collection. The generic manifest-only repair then stalled.
- The schema deliberately permits an empty findings array, so this is a narrow shape defect; it
  does not authorize ACP to invent findings or normalize any other provider content.

### Goals
- [x] Add a deterministic fixture for a valid collect artifact set whose only defect is missing
  `semantic.findings`.
- [x] Atomically add `semantic.findings: []` only for that exact failure and repeat full strict
  collect validation, including documents and repo evidence paths.
- [x] Restore the original manifest and continue existing provider repair/fail-closed behavior if
  any other validation error remains.
- [x] Emit before/after digests, recovery diagnostics and an explicit manual artifact-quality
  warning; do not treat shape recovery as artifact acceptance.
- [x] Add validation-specific provider repair guidance as a fallback without inventing findings.
- [x] Pass focused tests and the full deterministic DoD.
- [x] Merge the slice; repeat a direct non-release Claude smoke after the ProductShell gate slice
  so all live qualification uses one final clean commit.

### Non-goals
- No schema, HTTP API, workspace, provider command, retry, timeout, canonical matrix or curated
  repository change.
- No normalization of malformed JSON, wrong types, missing coverage, citations, evidence paths,
  document content or nested semantic objects.
- No accepted release evidence from the remediation branch.

### Acceptance
- Existing manifest values are structurally identical before and after recovery except for the
  inserted empty `semantic.findings` array.
- Recovery writes only `shard-pack-manifest.json`; candidate validation failure restores the exact
  original bytes and leaves provider-authored repair authoritative.
- The tracked fixture and focused engine/prompt tests run offline and deterministically.
- `make contracts`, `make test`, `make lint`, and `make build` pass.

### Progress log
- 2026-07-17: The tracked missing-findings fixture, engine integration, rollback/determinism,
  atomic-write cleanup and validation-specific prompt tests pass. Focused tests also passed with
  repeated counts (`providercommon` 20x, `promptcontract` 10x).
- 2026-07-17: Full deterministic DoD passed with Go `1.25.10`, Node `22.21.1` and npm `10.9.4`:
  contracts, complete Go suite, 261 Python tests, 141 UI tests, lint/typecheck and embedded UI
  production build are green.
- 2026-07-17: PR #148 passed required GitHub checks and merged to `main` as `59a94cab`.

<a id="ep-20260713-epic-19-pr1-remediation"></a>

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

<a id="ep-20260713-epic-19-19b-transactional-promotion"></a>

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

<a id="ep-20260713-epic-19-19c-async-panic-isolation"></a>

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

<a id="ep-20260713-epic-19-19d-server-owned-shutdown"></a>

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

<a id="ep-20260713-epic-19-19e-coherent-service-generation"></a>

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

<a id="ep-20260713-epic-19-19f-fresh-git-url-resolution"></a>

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

<a id="ep-20260713-epic-19-19g-minimum-collect-evidence-contract"></a>

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

<a id="ep-20260713-epic-19-19h-symmetric-document-citation-validation"></a>

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

<a id="ep-20260713-epic-19-19i-historical-run-artifact-snapshots"></a>

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

<a id="ep-20260713-epic-19-19j-request-scoped-ui-detail-state"></a>

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

<a id="ep-20260714-epic-19-19k1-run-mutation-acknowledgement"></a>

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

<a id="ep-20260714-epic-19-19k2-qa-provisional-run-ordering"></a>

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

<a id="ep-20260714-epic-19-19l-editor-revision-safety"></a>

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

<a id="ep-20260714-epic-19-19m-deterministic-embedded-ui-bundle"></a>

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

<a id="ep-20260714-epic-19-19o-locked-contract-validator-toolchain"></a>

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

<a id="ep-20260714-epic-19-19n-composite-release-verdict-gate"></a>

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

<a id="ep-20260714-epic-19-19p-step1-card-enrichment"></a>

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

<a id="ep-20260714-epic-19-19q-generic-refresh-semantic-guard"></a>

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

<a id="ep-20260714-epic-19-19r1-accessible-tabs-controller"></a>

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

<a id="ep-20260714-epic-19-19r2-keyboard-path-combobox"></a>

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

<a id="ep-20260714-epic-19-19r3-accessible-async-announcements"></a>

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

<a id="ep-20260714-epic-19-19s1-shell-dead-code-cleanup"></a>

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

<a id="ep-20260714-epic-19-19s2-shellcheck-lint"></a>

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

<a id="ep-20260714-epic-19-19s3-required-pr-lint"></a>

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

<a id="ep-20260714-epic-19-19t-logs-smoke-coverage"></a>

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

<a id="ep-20260714-epic-19-19u-deterministic-mock-playwright-ci"></a>

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

<a id="ep-20260714-epic-19-19u2-ui-v8-coverage-baseline"></a>

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

<a id="ep-20260714-epic-19-19v-python-runtime-pinning"></a>

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

<a id="ep-20260714-epic-19-19w1-runtime-draft-wrapper-cleanup"></a>

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

<a id="ep-20260714-epic-19-19w2-sharding-wrapper-cleanup"></a>

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

<a id="ep-20260714-epic-19-19w3-provider-argument-wrapper-cleanup"></a>

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

<a id="ep-20260714-epic-19-19w4-docflow-compatibility-helper-cleanup"></a>

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

<a id="ep-20260714-epic-19-19w5a-review-diff-residual-cleanup"></a>

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

<a id="ep-20260714-epic-19-19w5b-model-store-residual-cleanup"></a>

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

<a id="ep-20260714-epic-19-19w5c-orchestrator-quality-residual-cleanup"></a>

### Plan ID
EP-20260714-epic-19-19w5c-orchestrator-quality-residual-cleanup

### Context
`19W5c` follows committed `19W5b`. `docs/archive/audits/CODE_AUDIT_2026-07-10.md` identifies the
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

<a id="ep-20260714-epic-19-19w5d-reports-compiler-residual-cleanup"></a>

### Plan ID
EP-20260714-epic-19-19w5d-reports-compiler-residual-cleanup

### Context
`19W5d` follows committed `19W5c`. `docs/archive/audits/CODE_AUDIT_2026-07-10.md` identifies the
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

<a id="ep-20260714-epic-19-19w5e-prompt-contract-residual-cleanup"></a>

### Plan ID
EP-20260714-epic-19-19w5e-prompt-contract-residual-cleanup

### Context
`19W5e` follows committed `19W5d` and closes the `DEAD-007` package-local residual track.
`docs/archive/audits/CODE_AUDIT_2026-07-10.md` identifies the residual at
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

<a id="ep-20260714-epic-19-19x-ui-dead-surface-cleanup"></a>

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

<a id="ep-20260714-epic-19-19z-final-reconciliation"></a>

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

<a id="ep-20260715-21a-architecture-home"></a>

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

<a id="ep-20260715-21b-source-revisions"></a>

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

<a id="ep-20260715-21c-refresh-impact-plan"></a>

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

<a id="live-e2e-step2-typed-shard-completeness-hardening"></a>

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

<a id="artifact-quality-excellent-gate-proposalsfindings-linkage"></a>

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

<a id="uxui-iteration-live-shard-scope-mapping"></a>

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

<a id="uxui-iteration-active-provider-stream-diagnostics"></a>

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

<a id="uxui-iteration-onboarding-recovery-rendered-qa"></a>

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

<a id="implemented-vs-planned-operational-mirror"></a>

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
| 18 Live E2E black-box boundary | open (paused behind Epic 22) | PR #171/#172 are merged; fresh R3 qualification and composite evidence restart only after the provider-free Epic 22 closure gate |
| 19 Code quality remediation | done | Merged into `main` at `02716bb`; deterministic DoD and cleanup evidence are recorded in the active/archive plans and `docs/archive/audits/CODE_AUDIT_2026-07-10.md` |
| 20 Console UX trust and IA reset | done | 20A–20N deliver trustworthy contracts, four-destination shell, feature seams, semantic primitives and responsive task gates; Epic 18 trusted-machine release evidence remains separate |
| 21 Evidence-backed Architecture Home + impact-aware refresh | done (Wave 1) | Architecture Home, source revision/impact evidence, provider-free no-op, affected-only collect, surgical materialization and operator explanation are implemented |
| 22 Post-implementation correctness/trust audit | open (release blocker) | Ordered provider-free remediation `22A..22O`: runtime/path/snapshot/refresh correctness, typed recovery, live/product isolation and ProductShell offline closure before R3 |

---

<a id="ep-20260710-code-quality-audit"></a>

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
- [x] Publish `docs/archive/audits/CODE_AUDIT_2026-07-10.md` with prioritized remediation roadmap and confirmed dead-code register.
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
- Final worktree verification found only `docs/PLANS.md` and `docs/archive/audits/CODE_AUDIT_2026-07-10.md`.

---

<a id="ep-20260715-20f2-deliberate-queue-ui"></a>

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

<a id="ep-20260715-20i2-deep-url-context"></a>

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

<a id="ep-20260715-20j1-home-guided-setup-runs"></a>

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

<a id="ep-20260715-20j2-changes-knowledge-ask"></a>

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

<a id="ep-20260715-21d-explainable-no-op"></a>

## EP-20260715-21D-explainable-no-op

### Goal
- Turn safe `unchanged_candidate` plans into successful provider-free refreshes without canonical rewrites.

### Plan
- [x] Persist factual refresh execution audit and additive run summary.
- [x] Fail closed for every planner fallback and leave validator-promoted baseline selection unchanged.
- [x] Surface no-op identity in CLI and Runs and cover schema/orchestrator behavior.

<a id="ep-20260715-21e-affected-collect"></a>

## EP-20260715-21E-affected-collect

### Goal
- Execute only affected collect shards when all unaffected baseline packs can be replayed safely.

### Plan
- [x] Preflight and copy baseline packs into current taskrun staging before provider execution.
- [x] Reuse the existing checkpoint parser/apply path and preserve deterministic scheduling semantics.
- [x] Fall back to full execution on missing packs or shard-plan mismatch and provide bounded Git intent context.

<a id="ep-20260715-21f-surgical-materialization"></a>

## EP-20260715-21F-surgical-materialization

### Goal
- Make downstream publication decisions explicit and validator-gated.

### Plan
- [x] Add schema-validated materialization decisions with provenance and digest.
- [x] Record byte-identical no-op preservation and full-refresh rebuild decisions.
- [x] Keep final/citation validation and atomic promotion as the only canonical mutation boundary.

<a id="ep-20260715-21g-operator-explanation"></a>

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

<a id="ep-20260716-epic18-r3-qwen-artifact-readiness"></a>

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

<a id="ep-20260716-epic18-r3-qwen-stream-repair"></a>

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

<a id="ep-20260716-epic18-r3-step2-home-repair"></a>

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

<a id="ep-20260716-epic18-r3-qwen-tool-first-retry"></a>

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

<a id="ep-20260710-code-audit-remediation-backlog"></a>

## EP-20260710-code-audit-remediation-backlog

### Context
- The completed audit at `docs/archive/audits/CODE_AUDIT_2026-07-10.md` identified 29 reportable findings and 13 confirmed dead-code groups.
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

---

<a id="ep-20260718-epic18-r3-architecture-home-staging-reference"></a>

## EP-20260718-epic18-r3-architecture-home-staging-reference

### Context
- Canonical Claude `smoke tiny` from clean ProductShell baseline `5c93ff61` completed init collect
  and promoted a structurally complete Architecture Home, but its operator guidance linked to
  `reports/taskruns/run_20260718_063551_001/staging/shards/`.
- Promoted and staged-final overview bytes were identical, proving that strict draft validation
  allowed an internal execution path onto the user-visible canonical surface.
- The release sequence stopped before accepting the smoke; no matrix result from that run is
  reusable release evidence.

### Goals
- [x] Add the live-observed Architecture Home as a deterministic contract-rejection fixture.
- [x] Reject taskrun staging references in canonical Architecture Home before promotion while
  continuing to allow canonical artifact and concrete repository paths.
- [x] Harden first-pass, normal enrichment, compact retry and command-text retry prompts so staged
  evidence remains input-only and operator navigation uses canonical/repository references.
- [x] Synchronize architecture, testing and live-gate documentation.
- [x] Pass focused regressions and full deterministic DoD.
- [x] Merge the remediation, rerun canonical Claude smoke from the new clean commit, inspect
  promoted/UI-visible artifacts, then restart Codex qualification and the full R3 sequence.

### Non-goals
- No deterministic rewriting or sanitizing of provider-authored Architecture Home content.
- No schema, HTTP API, workspace, provider-contract, retry-budget, canonical matrix or curated
  repository change.
- No acceptance of runtime success as artifact-quality success; the promoted/UI-visible evidence
  remains the assessment source of truth.

### Acceptance
- Any `reports/taskruns/**/staging/**` reference in `reports/as-is/overview.md` fails the existing
  strict runtime draft contract before promotion and routes through existing provider repair or
  terminal fail-closed behavior.
- Canonical `reports/`, `model/`, `proposals/` references and concrete repository paths validate.
- Prompt contract tests cover first-pass and every supported step2 enrichment mode.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned Go/Node toolchains.

### Progress log
- 2026-07-18: Reproduced the live contamination from matrix
  `smoke-tiny-bank-20260718T062735Z`; terminated the run at the first artifact-quality blocker and
  started the minimal fail-closed remediation on a dedicated branch.
- 2026-07-18: Focused runtime-draft, prompt and step-policy suites pass. The two affected
  providercommon lifecycle tests each passed 20 consecutive runs after replacing the obsolete
  synthetic staging reference. Full deterministic DoD passed with Go `1.25.10`, Node `22.21.1`
  and npm `10.9.4`: contracts, full Go, 261 Python, 142 UI, shellcheck/typecheck and embedded UI
  build are green.
- 2026-07-18: Merged as PR #150. The post-merge Claude smoke reached collect `10/10` and exposed
  the next independent Architecture Home narration defect; that smoke was stopped and not accepted.

---

<a id="ep-20260718-epic18-r3-architecture-home-run-narration"></a>

## EP-20260718-epic18-r3-architecture-home-run-narration

### Context
- After PR #150 merged, canonical Claude smoke `smoke-tiny-bank-20260718T075924Z` completed
  collect `10/10` and step2 strict validation, but the staged-final Architecture Home said it was
  “derived from the run `run_20260718_080853_001` collect pass”.
- All eight required sections and concrete repository references were present and no staging path
  leaked, but user-visible process narration remained. The smoke stopped before promotion and is
  not reusable release evidence.

### Goals
- [x] Add the second live-observed Claude output as a deterministic rejection fixture.
- [x] Reject run IDs and pipeline/collect/repair narration in canonical Architecture Home before
  promotion while preserving direct repository facts and evidence gaps.
- [x] Harden first-pass, normal enrichment, compact retry and command-text retry prompts.
- [x] Synchronize architecture, testing and live-gate documentation.
- [x] Pass focused regressions and full deterministic DoD.
- [x] Merge the remediation and restart Claude smoke from the new clean merge commit.

### Non-goals
- No sanitizer, deterministic narrative rewrite, schema/API/provider-contract change, retry/timeout
  adjustment, matrix edit or accepted release evidence.

### Acceptance
- The exact live phrase and stable `run_YYYYMMDD_HHMMSS_NNN` identity are strict draft-contract
  failures in `reports/as-is/overview.md`.
- Canonical documents and concrete repository evidence remain valid.
- All step2 prompt variants explicitly prohibit execution narration.
- Full deterministic DoD passes with pinned toolchains before merge.

### Progress log
- 2026-07-18: Stopped the post-#150 Claude smoke at the first remaining artifact-quality blocker,
  preserved the staged-final overview under `/tmp/provenarch-remediation-evidence/`, and started a
  second minimal fail-closed remediation branch.
- 2026-07-18: Focused draft-validator, prompt, step-policy, providercommon and docsync suites pass.
  Full deterministic DoD passed with Go `1.25.10`, Node `22.21.1` and npm `10.9.4`: contracts,
  full Go, 261 Python, 142 UI, shellcheck/typecheck and embedded UI build are green.
- 2026-07-18: Merged as PR #151 at `f0d9dead`; canonical Claude smoke
  `smoke-tiny-bank-20260718T090734Z` restarted from that clean commit.

---

<a id="ep-20260718-epic18-r3-architecture-home-runtime-checkout"></a>

## EP-20260718-epic18-r3-architecture-home-runtime-checkout

### Context
- Post-PR #151 Claude smoke `smoke-tiny-bank-20260718T090734Z` completed collect `10/10` and
  produced all eight required Architecture Home sections, but step2 embedded absolute
  `/tmp/.../.acp/repos/...` checkout paths throughout the operator-facing document.
- The same document narrated current-run assembly through typed shard plan/summary, shard-pack
  manifests and completeness counters. Those execution facts belong in coverage/diagnostics, not
  the canonical architecture entry point.
- The matrix was terminated at this first artifact-quality blocker before acceptance or release use.

### Goals
- [x] Add a minimized fixture preserving the live absolute checkout and current-run recap patterns.
- [x] Reject `.acp/repos` runtime checkout references and the observed typed-shard/current-run
  narration before Architecture Home promotion.
- [x] Require stable `<repo>:<path>` or canonical document references in first-pass, normal,
  compact and command-text step2 prompt modes.
- [x] Synchronize architecture, pipeline, testing and live-gate documentation.
- [x] Pass focused regressions and full deterministic DoD.
- [x] Merge the remediation and restart Claude qualification from the new clean merge commit.

### Non-goals
- No sanitizer, hidden deterministic rewrite, schema/API/provider-contract change, retry/timeout
  adjustment, canonical matrix edit or accepted release evidence.
- Shard completeness remains valid in coverage/architect summary; only Architecture Home rejects it.

### Acceptance
- The live fixture fails strict draft validation for runtime checkout references and process recap.
- Canonical artifact links and stable repo-relative refs continue to validate.
- All step2 prompt modes prohibit runtime checkout paths and current-run typed-shard mechanics.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-18: Stopped the canonical Claude smoke after staged-final inspection exposed absolute
  runtime checkout paths and current-run shard mechanics; no result from that matrix is reusable.
- 2026-07-18: Focused runtime-draft/prompt/step-policy tests and full deterministic DoD passed
  with Go `1.25.10`, Node `22.21.1` and npm `10.9.4`: contracts, full Go, 261 Python, 142 UI,
  shellcheck/typecheck and embedded UI build are green.
- 2026-07-18: Merged as PR #152 at `3df8ecd3`; canonical Claude smoke
  `smoke-tiny-bank-20260718T102026Z` restarted from that clean commit.

---

<a id="ep-20260718-epic18-r3-command-stream-drain"></a>

## EP-20260718-epic18-r3-command-stream-drain

### Context
- Full deterministic DoD for the next Architecture Home remediation reproduced a provider
  lifecycle race: collect-pair stdout activity was observed, but termination could close the local
  pipe reader before those bytes entered `Result.Stdout`.
- The false zero-output result skipped the one allowed stream-only focused retry and made the
  timing-sensitive contract test fail repeatedly. This is an independent runtime defect.

### Goals
- [x] Capture a stream chunk before advancing the activity timestamp used by the stall monitor.
- [x] Allow the capture goroutines a bounded drain before forcibly closing local pipe readers.
- [x] Make both stream-only retry fixtures emit sustained diagnostics until termination.
- [x] Pass the focused pair-repair lifecycle tests for 20 consecutive runs.
- [x] Pass full deterministic DoD.
- [x] Merge the isolated remediation.

### Non-goals
- No new retry, timeout, provider, schema, API or canonical matrix behavior.
- No acceptance of missing/invalid artifacts; the second stream-only stall remains a strict
  `runtime_contract_failed` result.

### Acceptance
- Observed pipe activity cannot be published as zero captured output because of local ordering.
- The successful fixture performs exactly two focused calls; repeated stalls exhaust after exactly
  two calls and remain contract failures.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-18: Reproduced the failure in full DoD and focused stress runs, isolated the capture-order
  race, and passed both affected providercommon cases for 20 consecutive runs after the fix.
- 2026-07-18: Full deterministic DoD passed with Go `1.25.10`, Node `22.21.1` and npm `10.9.4`:
  contracts, full Go, 261 Python, 142 UI, shellcheck/typecheck and embedded UI build are green.
- 2026-07-18: Merged as PR #153 at `4958382d`; the Architecture Home shard-recap slice resumed
  from that clean merge commit.

---

<a id="ep-20260718-epic18-r3-architecture-home-shard-recap"></a>

## EP-20260718-epic18-r3-architecture-home-shard-recap

### Context
- Post-PR #152 Claude smoke `smoke-tiny-bank-20260718T102026Z` completed collect `10/10` and
  produced stable repository-relative evidence references, but Architecture Home still described
  the current run, typed shard packs and exact planned/succeeded/failed/incomplete counters.
- The same recap falsely reported zero files for shard-pack manifests despite successful collect.
  The matrix was stopped before promotion; none of its results are reusable release evidence.

### Goals
- [x] Add a minimized fixture preserving the live semantic recap variants.
- [x] Reject current-run, typed-shard, shard-pack and exact completeness-counter language in
  Architecture Home while keeping coverage and architect summaries unchanged.
- [x] Harden first-pass, normal, compact and command-text step2 prompt modes.
- [x] Synchronize architecture, pipeline, testing and live-gate documentation.
- [x] Pass focused regressions and full deterministic DoD.
- [x] Merge the remediation and restart Claude qualification from the new clean merge commit.

### Non-goals
- No sanitizer, deterministic narrative rewrite, schema/API/provider-contract change, retry/timeout
  adjustment, canonical matrix edit or accepted release evidence.
- Shard execution details remain valid in coverage and architect summary artifacts.

### Acceptance
- The live fixture fails strict draft validation for process narration.
- Stable repository references and the same wording outside Architecture Home remain valid.
- All step2 prompt modes explicitly prohibit the semantic recap variants.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-18: Stopped the post-#152 Claude smoke at the first remaining artifact-quality blocker
  after collect `10/10`; terminated the matrix and orphan provider process groups cleanly.
- 2026-07-18: After PR #153 removed the independent stream-capture race, focused draft/prompt/
  step-policy/providercommon suites and full deterministic DoD passed with Go `1.25.10`, Node
  `22.21.1` and npm `10.9.4`: contracts, full Go, 261 Python, 142 UI, lint and build are green.
- 2026-07-18: Merged as PR #154 at `d2018a03`; canonical Claude smoke
  `smoke-tiny-bank-20260718T121334Z` restarted from that clean merge commit.

---

<a id="ep-20260718-epic18-r3-stream-only-fixture-budget"></a>

## EP-20260718-epic18-r3-stream-only-fixture-budget

### Context
- Full DoD for the Architecture Home evidence-path slice reproduced one failure in the
  stream-only collect-pair exhaustion fixture: the shell process did not emit its first line
  inside the test-only 500 ms wall-clock, so the fixture observed a silent rather than stream-only
  stall and correctly skipped the stream-only retry.
- A focused 20-count stress run reproduced the timing miss once. Production focused repair budgets
  and behavior are not implicated.
- The next loaded full-suite run exposed the same fixture-class issue in the zero-output collect
  recovery test: its successful retry could be stopped in the 20 ms interval between authored
  markdown and manifest writes. The isolated slice also gives that test-only partial pair a bounded
  loaded-suite write window.

### Goals
- [x] Give only the repeated stream-only test fixture enough startup budget to emit diagnostics
  reliably under loaded-suite scheduling.
- [x] Give only the successful zero-output collect retry fixture enough time to finish its two-file
  pair after the first file appears.
- [x] Pass both stream-only lifecycle cases for 20 consecutive runs.
- [x] Pass full deterministic DoD.
- [x] Merge the isolated test remediation.

### Non-goals
- No production retry, timeout, lifecycle, provider, schema, API or canonical matrix change.
- No weakening of repeated-stall failure semantics; the fixture must still exhaust as
  `runtime_contract_failed` after exactly two focused calls.

### Acceptance
- Focused stress observes exactly two pair-repair calls in both success and repeated-stall cases.
- Full DoD is green with pinned Go/Node toolchains.

### Progress log
- 2026-07-18: Stream-only success/exhaustion passed 20 consecutive focused runs; zero-output
  collect retry passed 50 consecutive focused runs after its test-only partial-write window fix.
- 2026-07-18: Full deterministic DoD passed with Go `1.25.10`, Node `22.21.1` and npm `10.9.4`:
  contracts, full Go, 261 Python, 142 UI, lint and embedded UI build are green.
- 2026-07-18: Merged as PR #155 at `89dec41e`; the Architecture Home evidence-path slice resumed
  from that clean merge commit.

---

<a id="ep-20260718-epic18-r3-architecture-home-evidence-paths"></a>

## EP-20260718-epic18-r3-architecture-home-evidence-paths

### Context
- Post-PR #154 Claude smoke `smoke-tiny-bank-20260718T121334Z` completed collect `10/10` and
  passed step2 structural/content validation, but its staged Architecture Home published four
  plausible `bank-of-anthos:<path>` references that do not exist in the pinned repository.
- The matrix was stopped before promotion/acceptance; none of its results are reusable release
  evidence. This is a product truthfulness gap, not a provider-readiness or host failure.

### Goals
- [x] Preserve the live-observed missing repository references in a minimized fixture.
- [x] Resolve operator-visible Architecture Home `repo:path` references against current task repo
  roots, accepting existing files/directories and rejecting missing or escaping paths.
- [x] Harden every normal/focused step2 prompt path to state missing evidence as a gap rather than
  publishing a guessed path.
- [x] Synchronize architecture, pipeline, testing and live-gate documentation.
- [x] Pass focused regressions and full deterministic DoD.
- [x] Merge the remediation and restart Claude qualification from the new clean merge commit.

### Non-goals
- No schema, HTTP API, provider contract, retry/timeout or canonical matrix change.
- No ACP-authored correction of provider narrative and no acceptance of missing evidence paths.
- Canonical-document references and repository-root directory references remain valid.

### Acceptance
- The live fixture fails with the exact missing `repo:path` identities before promotion.
- Existing files and directories pass; absolute, traversal and symlink-escaped paths fail closed.
- Validation is read-only and does not mutate source repositories or runtime drafts.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-18: Inspected the staged step2 draft immediately after strict validation, confirmed four
  missing repository references, stopped the smoke during step4 and terminated all matrix/provider
  process groups without orphan processes or workspace mutation.
- 2026-07-18: After PR #155 stabilized the independent loaded-suite fixtures, focused draft,
  providercommon, step-policy and prompt suites passed on `89dec41e`. Full deterministic DoD passed
  with Go `1.25.10`, Node `22.21.1` and npm `10.9.4`: contracts, full Go, 261 Python, 142 UI, lint
  and embedded UI build are green. One shutdown-lifecycle test flaked once under load, passed 50/50
  focused and passed in the required repeated full suite, so no unrelated timeout change was made.
- 2026-07-18: Merged as PR #156 at `9e1c93ad`; canonical Claude smoke
  `smoke-tiny-bank-20260718T142040Z` restarted from that clean merge commit.

---

<a id="ep-20260718-epic18-r3-orchestrator-lifecycle-fixture-budget"></a>

## EP-20260718-epic18-r3-orchestrator-lifecycle-fixture-budget

### Context
- The canonical Claude qualification from PR #156 stopped during its deterministic precheck before
  provider execution. Under the loaded full Go suite, both async lifecycle fixtures timed out after
  one second before their runner goroutine was scheduled (`calls=0`).
- The shutdown fixture had already passed 50 consecutive focused runs after one earlier loaded-suite
  miss. The paired panic fixture now showed the same scheduling-sensitive test budget; production
  orchestration did not start and no live artifact was produced.

### Goals
- [x] Give only the two runner-start assertions a bounded loaded-suite scheduling budget.
- [x] Preserve the existing runner call-count, pending-run, panic and shutdown assertions.
- [x] Pass focused stress and full deterministic DoD.
- [x] Merge the isolated test remediation and restart Claude qualification from the new merge commit.

### Non-goals
- No production lifecycle, queue, shutdown, retry, timeout, schema, API, provider or matrix change.
- No acceptance of a runner that never starts; the fixture remains bounded and fail-closed.

### Acceptance
- Both lifecycle cases pass repeated focused execution and still prove the pending-run invariants.
- The full loaded suite no longer depends on a one-second goroutine scheduling window.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-18: `smoke-tiny-bank-20260718T142040Z` stopped in precheck with both runner-start
  assertions at `calls=0`; harness classified the matrix as failed before provider execution.
- 2026-07-18: Both lifecycle cases passed 100 consecutive focused runs. Full deterministic DoD
  passed with Go `1.25.10`, Node `22.21.1` and npm `10.9.4`: contracts, full Go, 261 Python,
  142 UI, lint and embedded UI build are green.
- 2026-07-18: Merged as PR #157 at `727f6d26`; canonical Claude smoke
  `smoke-tiny-bank-20260718T144339Z` restarted from that clean merge commit.

---

<a id="ep-20260821-task-first-design-qa-closure"></a>

## EP-20260821-task-first-design-qa-closure

### Current layer

Implementation/review. Q0–Q10 implementation slices are complete; deterministic Q10 checks,
embedded-asset parity, responsive/accessibility smoke and final live UI comparison are accepted.

### Context

- Epic 23 contracts, routes and additive surfaces are implemented, but the fresh comparison in
  [`docs/archive/audits/DESIGN_QA_2026-08-21.md`](audits/DESIGN_QA_2026-08-21.md) found six P1 and five P2 gaps against the authoritative
  task-first UX and target composition references.
- The current UI is operational and deterministic tests pass, but it still reads as a dense
  operator console. The highest-risk gaps are contradictory Task/Attempt state, diagnostic-first
  routing, incomplete Architecture/Findings workbenches, duplicated Changes review, modal Ask and
  raw runtime Settings.
- Written product behavior remains authoritative over PNGs. The corrective work must preserve
  automatic validator-gated promotion, immutable Attempt identity, current Architecture authority,
  full-workspace Git publication and read-only analyzed repositories.
- Existing implementation-status claims in Epic 23 describe additive contract/surface completion;
  they do not constitute visual or UX closure. Status docs are corrected only after rendered
  acceptance passes.

### Goal

Make the task-first UI trustworthy, calm and outcome-first: a user can create a Task, understand its
result, trace current Architecture evidence, review already-promoted changes, publish the complete
workspace and use Ask/Settings without needing runtime vocabulary or hidden legacy surfaces.

### Non-goals

- No hosted mode, multi-user collaboration, persisted approval workflow or security enforcement.
- No new headless provider beyond `fake`, `claude-code`, `qwen-code`, `codex-code`.
- No pipeline/schema change made only to imitate reference screenshots.
- No structured YAML/JSON editing until the accepted lossless round-trip requirement is proven.
- No visual Mermaid drag editor or selected-file Git commit.
- No new runtime model/reasoning defaults, live matrix changes or release gate expansion.
- No write to analyzed source repositories and no hidden compatibility UI.

### Source of truth

1. `schemas/*`, validators and `docs/spec/*`, including `TASK_SPEC.md` and `PIPELINE_SPEC.md`.
2. `docs/UI_TASK_FIRST_PRODUCT_DESIGN.md`.
3. Epic 23 acceptance in `docs/BACKLOG.md`.
4. `docs/UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md`.
5. `docs/archive/audits/DESIGN_QA_2026-08-21.md` and the normalized comparison evidence.
6. PNGs under `docs/assets/ui-task-first-product/` for composition, density and visual character.

### Product brief

- **Product type:** local-first responsive web/desktop developer tool embedded in the Go binary.
- **Primary users:** architect/tech lead, operator/maintainer, reviewer/stakeholder and first-time
  evaluator.
- **Primary job:** turn an architecture question into validated, evidence-backed, Git-reviewable
  current workspace knowledge.
- **Primary path:** Setup -> New Task -> Running/Outcome -> Architecture/Findings -> Changes ->
  Publish.
- **Recovery path:** failed/canceled Attempt -> retained last-good Architecture -> one blocker ->
  child Attempt retry/change runner.
- **Quality bar:** no contradictory terminal state, no silent identity fallback, no P0/P1/P2 design
  QA findings, keyboard-complete primary jobs and deterministic offline closure.

### Assumptions and required decisions

- The PNGs are composition targets, not pixel/copy/data-contract oracles; written behavior wins.
- Structured model inspection remains read-only in this epic.
- Until the exact target display font is identified, implementation may use an approved system
  serif fallback for display/document roles; the choice must be recorded before Q2 is accepted.
- The current UI has no coherent icon package. Adding one production icon dependency requires
  explicit owner approval; otherwise licensed source icons must be supplied. Text glyphs are not an
  accepted closure path.

### Dependency order

```text
Q0 baseline decisions/fixtures
  |--> Q1 state truth and identity
  |--> Q2 visual foundation

Q1 + Q2 --> Q3 Setup and New Task --> Q4 Inbox and Outcome --> Q5 Pipeline Studio
Q1 + Q2 --> Q6 Architecture Map/Documents/Diagrams --> Q7 Model/Schema/Findings
Q1 + Q2 + Q7 --> Q8 Changes and Publish
Q2 + Q6 --> Q9 Ask and Runner Settings
Q3..Q9 --> Q10 responsive/accessibility/design-QA/docs closure
```

### Reviewable slices

#### Q0 — Baseline decisions and deterministic visual fixtures (`S`)

**Outcome:** every later slice compares the same route, state, data and viewport.

- Record the approved display-font strategy and icon source/dependency decision.
- Add/refresh deterministic fixture identities for Setup ready, New Task empty, populated Inbox,
  running/succeeded/failed/canceled Attempt, current/partial Architecture, selected finding,
  initial/update Changes, Publish confirmation, Ask answered/failed and runner readiness.
- Capture the missing Pipeline Recovery and answered Ask implementation states.
- Define comparison viewports: source-sized design QA captures plus the canonical responsive matrix
  `1440x980`, `1024x768`, `768x1024`, `390x844`, DPR 1.
- Keep current-behavior captures ephemeral; only target references stay committed.

**Acceptance:** every target screen has one matched deterministic implementation state or an
explicitly documented contract blocker; source and implementation can be compared side by side
without density, browser-chrome or state ambiguity.

#### Q1 — Task/Attempt state truth, refresh and route identity (`M`)

**Outcome:** no user-visible state contradicts the authoritative Task/Attempt/run contracts.

- Route a newly admitted Attempt to the Task running/outcome surface, not Pipeline Studio.
- Derive terminal status, structured steps, blocker and recovery from one exact Attempt/run review
  identity. A succeeded Attempt cannot retain active/pending/failed presentation.
- Invalidate/refetch current Architecture automatically after successful promotion; no manual global
  Refresh is required to see the new authority.
- Remove direct-Changes latest-run fallback. An exact Task/Attempt/run is restored from the URL, or
  the UI shows an explicit chooser/unavailable state.
- Keep failed/canceled child-retry lineage and last-good Architecture visible without mutating the
  terminal parent.
- Start with frontend/view-model/API-consumption fixes. If the public contract cannot express a
  coherent state, invoke `acp-schema-guardian` before changing schemas or validators.

**Acceptance:** fixtures cover queued, running, stalled, succeeded, failed, canceled and timed-out
Attempts; no terminal row or detail uses active language; Architecture refreshes after promotion;
stale/missing identities fail closed and never substitute latest/current data.

#### Q2 — Visual foundation and reusable component contract (`M`)

**Outcome:** screen work consumes one target-aligned design system instead of one-off overrides.

- Make `ui/src/styles/tokens.css` the semantic source for warm canvas/surfaces, forest primary,
  rare orange decision/recovery accent, text, borders, focus and semantic statuses.
- Add type roles for display, document, body, label, metadata and code; retain sans UI text and use
  the approved serif only where the target uses display/document hierarchy.
- Preserve the existing spacing scale but define relaxed workbench/onboarding and compact repeated-
  operations density modes.
- Define component contracts and states for ProductShell/Nav, PageHeader, Workbench/SplitPane,
  Drawer/Sheet, TaskRow, EvidenceCard/Chain, Status/Authority, FilterBar, Disclosure, Empty/Loading/
  Error/Partial panels, buttons, fields and icon buttons.
- Replace text glyphs with one approved vector icon family; no handcrafted inline SVG/CSS art.
- Remove raw screen-specific colors/radii when an existing semantic token can express the role.

**Acceptance:** tokens have real consumers and no duplicate semantic meanings; all touched
components cover hover/active/focus-visible/disabled/loading/selected/invalid states; light desktop
nav and mobile bottom nav use the same semantic roles; target title/body/meta hierarchy is visible
in rendered fixtures.

#### Q3 — Guided Setup review and New Task composer (`M`)

**Outcome:** a first-time evaluator understands boundaries, runner readiness and expected output
before starting work.

- Setup uses the target stepper, exact workspace/source/runner summary, readiness, read/write
  boundaries, quality warning and “What happens next”.
- `Create first task` opens New Task and never starts analysis implicitly.
- New Task is goal-first with optional context, compact scope, runner preset/readiness, effective
  model/effort and a “What you’ll get” pane.
- Preserve draft across navigation, provider readiness recovery, offline/reconnect and demo fallback.
- Keep one primary `Start task` action adjacent to the real blocker/reason.

**Acceptance:** ready/partial/blocked/save-error/offline Setup and empty/draft/invalid/checking/
unavailable/conflict/queued New Task states pass; displayed scope equals submitted scope; the first
Task can be started keyboard-only and at `390x844` without hiding the primary action.

#### Q4 — Task Inbox, selection, Attempt history and outcome (`L`)

**Outcome:** Tasks is an efficient daily workspace rather than a filter-heavy administrative list.

- Desktop uses lifecycle list + selected Task preview; mobile uses a route split with preserved
  filters and list position.
- Collapse advanced filters behind one control; keep search and primary lifecycle filters visible.
- Show goal, meaningful state, runner and last activity without raw IDs on each compact row.
- Outcome-first Task detail shows validated result, semantic counts, questions/gaps, current
  Architecture availability and recommended next action before Attempt telemetry.
- Keep Attempt identity/config/lineage in history/detail and diagnostics, not the primary outcome.
- Preserve archive/unarchive, empty/filtered-empty/loading/error/offline/retry behavior and focus.

**Acceptance:** the first useful Task appears above the fold on mobile; 1000-row fixture remains
scannable and keyboard-safe; Task list/detail/history counts reconcile; succeeded/failed/canceled/
recovered/demo/live outcomes are truthful and URL-restorable.

#### Q5 — Focused Pipeline Studio and recovery (`M`)

**Outcome:** diagnostics are one level deeper and show one real recovery decision.

- Open Studio only from an exact Attempt.
- Show canonical durable steps, scopes/artifacts and last useful progress; provider activity never
  advances completion.
- Select one blocker with cause, retained data and recommended action.
- Put raw logs, JSON, permission telemetry and technical IDs into bounded Diagnostics disclosure.
- Implement/verify Retry failed scope, Change runner for next Attempt and cooperative Stop states
  against authoritative admission/retry contracts.

**Acceptance:** active/retrying/stalled/permission-required/failed/canceled captures match their
contracts; retry creates a child and leaves the parent immutable; diagnostics cannot create an
unbounded page; a missing matching recovery capture blocks slice acceptance.

#### Q6 — Architecture Map, Documents, Diagrams and shared evidence (`L`)

**Outcome:** current Architecture is the independent knowledge product and remains usable during
or after failed Attempts.

- Architecture Map uses a semantic catalog/map/inspector layout with validated nodes/edges, search,
  filters, zoom/fit, current authority and an accessible list alternative.
- Selection links exact document, model entity, finding and evidence identities.
- Documents default to Architecture Home or Task-linked semantic document, not an alphabetical
  proposal path; use semantic tree, rendered/source modes, outline and evidence/context pane.
- Keep allowlisted current Markdown edit separate from promoted/run-snapshot read-only readers.
- Diagrams add fit/zoom, raw fallback, exact selected identity and accessible relation parity.
- Preserve explicit unavailable/partial/stale/broken/oversized/offline states without inventing
  topology or evidence.

**Acceptance:** current knowledge survives active/failed Attempts; default selection is useful and
authoritative; every visible claim/relation can reach exact evidence; map/list/document/diagram
keyboard and containment fixtures pass.

#### Q7 — Model/Schema workbench and Findings evidence chain (`L`)

**Outcome:** structured knowledge and review decisions are distinct, evidence-first workbenches.

- Split Model/Schema from the map into semantic tree, structured inspector and validation/evidence
  pane; keep Source under Advanced and editing disabled until lossless patching exists.
- Surface schema/version, field help, logical IDs, semantic diagnostics and related entity/edge
  navigation without using filenames as facts.
- Findings uses list/detail/evidence-chain composition with observation, why it matters, suggested
  direction, exact repo/path/line refs, confidence and coverage gaps.
- Resolve `related_ids` into actionable evidence links; never show only a reference count.
- Expose guarded review/proposal draft actions only where the persisted decision contract permits.

**Acceptance:** invalid/unavailable structured source retains the last valid rendering; comments/
unknown keys/order/multiline fixtures keep read-only truth; finding counters reconcile with Task and
Changes; one keyboard path completes finding -> evidence -> source -> returned focus.

#### Q8 — Changes truth and full-workspace Publish (`L`)

**Outcome:** semantic review and Git mutation are clear, distinct decisions.

- Replace stacked semantic + legacy review with one workbench: grouped change set, selected rendered/
  source diff and review notes/evidence.
- State explicitly that validator-approved knowledge already updated current Architecture; review
  here does not approve/reject analysis and Publish only performs the Git action.
- Keep Summary/Evidence/Files/Publish materially distinct and pinned to exact Task/Attempt/run.
- Show semantic delta first and Git inventory second; never equate their counters.
- Publish confirmation defaults to folder/file counts, validation, open risks, commit message and
  branch/HEAD; move fingerprint and individual paths into Technical details/All files disclosures.
- Preserve authoritative stale HEAD/inventory, dirty/clean, conflict, busy, failure and retry gates.

**Acceptance:** no latest-run fallback, duplicate H1/review surface or “replace/promote current
snapshot” copy remains; the exact full-workspace scope is clear; stale confirmation cannot mutate
Git; successful commit refreshes Task/publication state and cannot double-submit.

#### Q9 — Contextual Ask and Runner Settings (`M`)

**Outcome:** Ask preserves Architecture context, while everyday runner configuration is preset-first.

- Replace centered Ask modal with a right contextual drawer that keeps Architecture visible,
  restores focus and uses internal/sticky scrolling without clipped controls.
- Cover empty/running/answered/unresolved/failed/retry/provider-outage Ask states with impact summary,
  architecture path, citations, assumptions and explicit proposal action.
- Make Runner Settings a preset list + selected runner/model/effort/execution + readiness card + task
  defaults.
- Put raw timeout, permission, precedence and per-step keys inside Advanced; preserve effective vs
  desired/source values and immutable active Attempt history.

**Acceptance:** Ask never mutates current knowledge implicitly and remains usable at desktop/mobile;
provider outage clearly distinguishes blocked new Ask/Attempt from readable existing Architecture;
preset invalid/save/conflict/readiness states fail inline and cannot rewrite an admitted Attempt.

#### Q10 — Responsive, accessibility, deterministic design QA and docs closure (`L`)

**Outcome:** the implemented UI, embedded binary and documentation make the same truthful claim.

- Run the full state/viewport matrix with overflow, console-error, critical axe, keyboard-only,
  focus-return, 200% zoom, reduced-motion, long-path and touch-target checks.
- Compare matched implementation captures against every target reference using full-view and focused
  side-by-side evidence; append iteration history to `docs/archive/audits/DESIGN_QA_2026-08-21.md`.
- Remove obsolete/duplicated components, CSS, routes, test IDs and compatibility DOM exposed by the
  corrective slices.
- Run `acp-docs-sync` only after behavior is implemented; synchronize README, ARCHITECTURE,
  STAKEHOLDER, TESTING_STRATEGY, Epic 23 status and the migration-plan status.
- Verify source repositories are unchanged and embedded `ui_dist` matches the tested frontend.
- Execute full DoD: `make contracts`, `make test`, `make lint`, `make build`; deterministic UI mock
  E2E remains offline and required. Live provider gate remains a separate trusted-machine step via
  `acp-e2e-live-gate` after deterministic closure.

**Acceptance:** critical axe violations are zero; no global overflow or clipped primary action at
all supported viewports; every primary job is keyboard-complete; no actionable P0/P1/P2 design QA
finding remains; `docs/archive/audits/DESIGN_QA_2026-08-21.md` ends with exact `final result: passed`; full DoD and embedded parity
pass from a clean tree.

### State matrix required for closure

| Surface | Required states |
| --- | --- |
| Setup | default, partial, blocked, ready, save error, offline/reconnect |
| New Task | empty, draft, invalid, readiness checking/unavailable, admission conflict, starting, queued, offline |
| Inbox | loading, empty, filtered empty, populated, error/retry, offline, archived |
| Task/Attempt | queued, running, stalled, succeeded, failed, canceled, timeout, retained last-good, child retry |
| Architecture | loading, unavailable, partial, current, stale identity, error, active/failed Attempt alongside current |
| Document/Model/Diagram | no selection, loading, loaded, read-only, editable where allowed, dirty, invalid, oversized, render error, save error |
| Findings | empty, filtered empty, selected, missing evidence, unresolved, proposal unavailable/available, error |
| Changes | no exact context, initial review, update with baseline, unavailable comparison, stale identity, semantic/file views |
| Publish | loading, ready, warnings, blocked, stale HEAD/inventory, conflict, busy, success, failure/retry |
| Ask | empty, running, answered, unresolved, failed, retrying, provider outage, proposal confirmation |
| Settings | loading, preset selected, dirty, invalid, checking, ready, save conflict, advanced expanded |

### Cross-slice interaction rules

- One dominant action per primary surface; diagnostics and raw IDs are secondary disclosures.
- Explicit Task/Attempt/artifact/evidence authority stays in the URL and never silently falls back.
- Validation is adjacent to the field/action; async work announces one bounded status and prevents
  duplicate submission.
- Destructive or Git-mutating actions require exact scope preview and stale-state rejection.
- Dialog/drawer close returns focus to the invoker; route transitions focus the new `h1` or explicit
  surface heading.
- Mobile touch targets are at least 44px; status never relies on color alone.
- Source-repository paths are evidence links only; editable workspace boundaries remain explicit.

### Milestones

1. **Trustworthy Task loop:** Q0–Q5 accepted.
2. **Evidence-first Architecture:** Q6–Q7 accepted.
3. **Truthful review, publication and utilities:** Q8–Q9 accepted.
4. **Product closure:** Q10 accepted and design QA passed.

### Stop condition

Stop implementation planning expansion when Q0–Q10 acceptance is satisfied. Do not add unrelated
providers, hosted/security scope, structured editing or new runtime defaults merely to close visual
references. If Q1 proves the public state contract is insufficient, pause that slice for an explicit
schema/contract decision instead of inventing frontend state.

### Progress log

- 2026-08-21: Baseline plan created from `docs/archive/audits/DESIGN_QA_2026-08-21.md`, Epic 23, the target UX and migration plan.
  No implementation started. Current QA remains `blocked` with six P1 and five P2 findings.
- 2026-08-21: Q1 implementation started. New Task admission now lands on exact Task detail;
  terminal Pipeline Studio states suppress stale blockers/active language; Architecture refreshes
  after a promoted Task outcome; direct Changes navigation no longer chooses a latest run. Narrow
  component tests pass. Live verification is in progress against a fresh fake workspace; the first
  fixture exposed a separate empty-analysis-exclude backend fixture failure, then a non-empty
  exclude fixture produced a successful promoted outcome and updated Architecture authority.
- 2026-08-21: Q2–Q3 implemented. Shared warm semantic tokens, display/document typography,
  responsive shell states, Guided Setup stepper, goal-first New Task layout and the “What you’ll
  get” boundary panel are live; focused component tests and typecheck pass.
- 2026-08-21: Q4–Q9 implemented. Inbox now has a selected-task preview; Architecture exposes the
  Map workbench and actionable Findings evidence chains; Changes is explicit-review-package-first;
  Ask is a contextual read-only drawer; Settings is preset-first with collapsed advanced controls.
  Focused Knowledge/Ask/Task/Changes suites pass, followed by the full UI suite (47 files,
  256 tests) and App suite (103 tests).
- 2026-08-21: Documentation synchronized across README, ARCHITECTURE, STAKEHOLDER_DOC and the
  task-first UX baseline. Q10 verification remains open pending the final deterministic DoD,
  embedded UI parity and design-qa result.
- 2026-08-21: Q10 verification found and fixed an Architecture auto-selection history regression:
  initial document selection now uses `replaceState`, while explicit document changes remain
  navigable. Git Publish confirmation now leads with grouped folder/count scope and keeps HEAD,
  fingerprint and the full file inventory under disclosures. The full mock E2E is green (8/8),
  embedded parity is green, and live desktop/mobile smoke is clean. Design QA remains blocked only
  on the owner-approved icon family decision plus the not-yet-recorded axe/keyboard/zoom and
  dedicated recovery/answered-Ask evidence matrix; canonical `make build` also remains pinned to
  unavailable Node.js 22.21.1 on this host, while the equivalent version-check-disabled build and
  embedded parity pass.
- 2026-08-21: Publish confirmation was refined after the QA pass to show grouped folder/count scope
  first and keep technical Git identity plus the full file list in disclosures. Targeted Publish
  tests and the full UI suite remain green after the refinement.
- 2026-08-21: Q10 live closure completed its responsive/accessibility pass. Map, Setup and Settings
  controls were reflowed for tablet/mobile; all visible mobile controls are at least 44px; Ask
  now restores focus to its invoker; Findings controls have explicit form identities; `/robots.txt`
  and the shell meta description are valid. The live matrix covered 28 route × viewport cases with
  no overflow or clipping, Lighthouse desktop/mobile scored 100/100/100 with zero failed audits,
  answered Ask was verified with three citations, and the deterministic recovery matrix remains
  8/8 green. Full UI (47 files/256 tests), Go, Python (274 tests), contracts, lint, embedded build
  and `ui_dist` parity pass. Q10 remains blocked only by the owner-approved icon/font decisions,
  unavailable native 200%/reduced-motion emulation in the current browser surface, and the exact
  Node.js 22.21.1 release toolchain required by canonical `make build`.
- 2026-08-21: Q10 final closure completed. The MVP icon decision is now explicit: the shell uses a
  self-contained CSS primitive icon family, with no unapproved production dependency; the target
  font metadata was not present in the references, so the documented Georgia/system-serif fallback
  is accepted for this slice. Exact Node.js 22.21.1 was resolved through the repository toolchain
  candidate path. A fresh full DoD passed (`make contracts`, `make test`, `make lint`, `make build`),
  including Go, Python (274 tests), UI (47 files/256 tests), embedded parity and unchanged source
  fixtures. Playwright/CDP verified reduced-motion and page scale 200% across all seven routes;
  the final live matrix is 28/28, mobile touch targets are 0 below 44px, Lighthouse is 100/100/100
  on desktop and mobile, and mock E2E remains 8/8. Q10 is accepted and the product-design QA result
  is `passed`.
- 2026-08-23: Q10 priority closure completed against the fresh UI audit. Changes now exposes one
  review workbench without a duplicate H1; Architecture Documents select `exports.home_path` (or
  the named/canonical Architecture Home) before alphabetical fallback; Task outcomes show
  `Unavailable`/explicit produced counts when semantic comparison is unavailable instead of a zero
  delta; route transitions reset scroll and focus the new heading; Inbox advanced filters and empty
  lifecycle groups are collapsed; Settings Advanced runtime controls are closed by default. Focused
  component coverage, full UI (47 files/258 tests), mock E2E (8/8), desktop/mobile overflow and
  critical axe checks passed. Source repositories remain unchanged.
- 2026-08-23: Final task-first audit remediation completed. Guided Setup now hands off only to New
  Task; New Task is goal-first with a visible Start Task action; Architecture defaults to semantic
  Map; Findings expose selected evidence-rich detail; Changes keeps one compact selected change-set
  workbench; Publish confirmation and refresh share the authoritative full-workspace Git inventory;
  Settings runner presets are selectable and saveable. Focused suites (128 tests), full UI (47
  files/258 tests), deterministic mock E2E (8/8), contracts, Go, Python (274 tests), lint, build and
  embedded parity pass with the pinned Node.js 22.21.1 candidate at
  `/tmp/provenarch-node-22.21.1-bin`.
- 2026-08-25: Post-closure UI cycle completed. Repository evidence keeps source authority separate
  from workspace artifacts; Changes preserves exact Task/Attempt identity; legacy null repository
  paths normalize to schema-valid arrays on restart. ArchitectureMap is lazy-loaded so the initial
  Knowledge route is 47 KB instead of 1.5 MB, and map rendering waits for a measurable container
  while suppressing the recoverable React Flow zero-size warning. Full UI tests (47 files/253 tests),
  targeted typecheck/build, desktop/mobile Architecture captures and mock E2E (8/8) pass; the
  remaining large map chunk is route-local and does not block the initial shell.
<a id="historical-continuous-backlog-queue-policy-july-2026"></a>

### Continuous Backlog Queue Policy

Epics 19, 20 and 21 are implementation-complete in `main`. `docs/BACKLOG.md` remains the
reference/acceptance backlog; this file selects focused active slices. The current release blocker
is provider-free Epic 22. Epic 18 R3 restarts only after the `22O` closure gate records one clean
qualification SHA. The post-R3 product queue is fixed as `K2b -> K4 -> K3A -> K3B -> 9D -> cleanup`.

Allowed next workstreams:
- The next ordered Epic 22 slice, with a decision-complete child ExecPlan and provider-free DoD.
- Epic 19/20/21 archive or reconciliation bookkeeping only; no original-scope engineering slice remains.
- Epic 18 R3 release validation only after `22O`, with trusted-machine/provider/path prerequisites.
- `EP-20260508-oss-readiness-hardening`: owner/admin verification for residual GitHub repository settings only.
- Post-R3 K-roadmap work only in the recorded order, with a fresh decision-complete ExecPlan.
- Cleanup applies the accepted readable-fixture retain decision and archives stale trackers; it does
  not remove or deduplicate `golden/readable`.

Task selection rules:
- Completed plans whose only remaining item is owner review, merge/archive bookkeeping, or historical evidence retention are not next engineering work.
- Trusted-host/live-release items remain explicit blockers; do not run them as normal backlog tasks
  or before the provider-free closure prerequisite.
- Each selected slice gets a decision-complete ExecPlan/update before implementation, one focused implementation pass, self-review/fix loops, Full DoD (`make contracts`, `make test`, `make lint`, `make build`), then one commit.


<!-- Additional relocation during the approved 2026-09-05 revision; original plan body retained. -->

## EP-20260804-agents-gpt-5-6

Status: review — recorded owner/admin or merge acceptance remains open.

Next action: Resolve the recorded open criterion: Archive this plan after repository-owner review.

### Context
Root `AGENTS.md` should remain a compact repository guidance surface while taking advantage of
current GPT-5.6 prompting behavior: outcome-oriented instructions, explicit autonomy boundaries,
targeted context routing and evidence-backed completion. Provider model pins and release behavior
remain separate runtime contracts.

### Goals (must have)
- [x] Replace full-corpus startup reading with task-specific source routing.
- [x] Add concise autonomy, success, stop-condition and validation rules.
- [x] Add model-aware guidance without pinning a model or enabling optional GPT-5.6 features.
- [x] Keep ProvenArch MVP, contract, deterministic CI and live-gate invariants intact.
- [ ] Archive this plan after repository-owner review.

### Non-goals
- No `codex-code` model/reasoning default migration.
- No runtime, schema, product behavior, release matrix or provider-list changes.

### Approach
1) Compare repository guidance with current official Codex `AGENTS.md` and GPT-5.6 prompting docs.
2) Deduplicate process rules and route detailed procedures to specs, skills and runbooks.
3) Validate the resulting Markdown, diff and repository DoD.

### Files expected to change
- `AGENTS.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] `AGENTS.md` keeps durable repo rules and avoids duplicated release workflow detail.
- [x] Model-specific runtime choices remain outside `AGENTS.md`.
- [x] `make contracts`, `make test`, `make lint` and `make build` pass.

### Risks
- Over-compression could hide a release invariant; the live skill/runbook remain authoritative.
- Model-version wording can age quickly; only durable behavior guidance belongs in `AGENTS.md`.

### Progress log
- 2026-08-04: Reviewed official Codex customization/AGENTS guidance and GPT-5.6 prompting guidance.
- 2026-08-04: Reworked root guidance around source routing, scope, autonomy and validation.
- 2026-08-04: Full provider-free DoD passed with pinned Node.js 22.21.1: contracts, Go/Python/UI
  tests, lint/typecheck and production build.

---

## EP-20260905-agent-development-revision

Status: complete — owner-approved local instruction/tooling revision validated and archived.

Closure: Local implementation and verification are complete; no further action in this slice.

### Context
Owner-approved follow-up to the repository agent-development audit on `91509c08`.
Current layer: implementation. This local slice improves development instructions and tooling;
it does not execute or close the independent `EP-20260905-audit-remediation-program` queue.

### Goals (must have)
- [x] Route agents to current architecture, contracts, active work and focused verification.
- [x] Update all five developer skills and distinguish them from ACP workspace/runtime prompts.
- [x] Archive demonstrably completed plans and reconcile proven documentation/status drift.
- [x] Make fresh-worktree setup and mock UI verification reproducible and isolated.
- [x] Validate instruction structure/references and run focused checks plus the full local DoD.

### Non-goals
- No product runtime/prompt/schema changes, provider/model defaults, release matrices or live runs.
- No push, publication, repository administration or execution of the separate remediation queue.
- Preserve incomplete release qualification, owner-review gates and neighboring stabilization work.

### Acceptance criteria
- Agent guidance identifies the owning spec/code/test for each supported development area.
- Skills have valid metadata and resolvable local references; current plans remain discoverable.
- Concurrent mock checks cannot reuse another worktree's server or remove its evidence.
- UI freshness verification inspects the intended source state without rewriting tracked assets.
- `make contracts`, `make test`, `make lint`, `make build` pass, or exact environmental blockers
  are recorded without changing pinned versions or weakening checks.

### Progress log
- 2026-09-05: Started from clean worktree `91509c08`; split independent instruction, tracker and
  UI-tooling work. Root owns setup, structural validation, integration and final review.
- 2026-09-05: Updated all five skills, root guidance and change-to-spec/check routing. Archived 79
  historical source blocks losslessly except a documented relative-link adjustment; preserved all
  incomplete qualification/owner gates and the separate remediation program.
- 2026-09-05: Added worktree setup/preflight, nonmutating UI freshness checks, explicit source-state
  determinism verification and isolated mock evidence/server allocation. Independent review found
  and fixed parallel Make install/check ordering and base-Python versus runtime-override ambiguity.
- 2026-09-05: Focused setup/UI regression tests, skill metadata/link checks and three independent
  workflow forward-checks passed. Full Go suite and canonical lint passed; remaining DoD and
  rendered mock checks are in progress using installed pinned Node 22.21.1 and local Python venv.
- 2026-09-05: Final local DoD passed: `make contracts`, `make test`, `make lint`, `make build`.
  The successful complete test run covered the Go suite, 291 Python tests and 253 UI tests in 47 files.
  An earlier test process received SIGTERM during Python discovery; no failing assertion was observed,
  its focused boundary test passed, and the subsequent full run completed successfully.
- 2026-09-05: `make preflight`, `make verify-ui-determinism verify-ui-dist` and rendered mock E2E
  passed (`8 passed / 0 skipped`). Mock evidence used a fresh private directory and loopback port;
  no live provider was invoked. Independent review and three workflow forward-checks passed.
- 2026-09-05: Preserved source fixtures, production dependency declarations and embedded UI assets
  without diff. Archived this completed plan and the superseded August instruction-revision plan;
  existing remediation dependencies and trusted release/owner gates remain open in their own scopes.
