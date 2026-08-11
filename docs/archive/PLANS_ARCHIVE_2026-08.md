# Closed ExecPlans — August 2026

## EP-20260811-task-first-ui-target-baseline

### Result

- Established one authoritative planned task-first UX baseline with `Task -> Attempt -> Outcome`,
  runner selection next to task start and independent Architecture/Changes/Publish ownership.
- Defined authority-safe Markdown/YAML/JSON/Mermaid/evidence workbenches, full recovery and
  publication journeys, responsive/accessibility rules and one 13-screen visual reference set.
- Added Epic 23 slices `23A`–`23O` with contract, UI, state, test, cutover and closure acceptance.
- Synchronized README, ARCHITECTURE and STAKEHOLDER planned/current status, marked both older UI
  design waves as superseded and removed their committed PNG/mock-log asset sets after link audit.
- Targeted link/assets checks and `git diff --check` passed. Full DoD passed after moving the
  completed plan here: contracts, Go, 267 Python tests, 231 UI tests, shellcheck, TypeScript,
  production UI build and Go binary build.

### Non-goals preserved

- No Task schema/API/backend/frontend implementation was added.
- No current ProductShell, provider/runtime policy or release matrix behavior changed.
- No live provider or release gate was executed.

## EP-20260811-weak-model-validation-backlog

### Result

- Fetched `origin/main` at `64b42251` and created the planning branch from that exact main commit.
- Reconciled the proposed hardening with completed Epic 22 typed recovery/auditor work and current
  runtime, schema and pipeline behavior.
- Added Epic 24 slices `24A`–`24I` for task-bound validator admission, verdict consistency,
  evidence/graph integrity, mandatory pre-promotion audit, orchestrator-owned effective verdict,
  conditional mechanical-envelope reduction, bounded recovery and provider-free conformance.
- Recorded goals, non-goals, dependencies, expected modules, focused tests, compatibility boundaries
  and measurable acceptance for every slice.
- `git diff --check` and the focused `internal/docsync` test passed after archiving this completed
  plan outside the active-plan surface.

### Non-goals preserved

- No runtime, schema, provider, UI or release-matrix implementation was added.
- No default model/provider change and no live E2E execution occurred.
- No incomplete historical live run was reinterpreted as acceptance evidence.
