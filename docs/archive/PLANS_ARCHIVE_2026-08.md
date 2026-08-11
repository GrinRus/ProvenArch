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
