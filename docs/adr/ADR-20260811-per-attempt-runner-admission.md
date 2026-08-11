# Per-Attempt runner admission

- **ADR ID:** ADR-20260811-per-attempt-runner-admission
- **Status:** accepted
- **Date:** 2026-08-11
- **Owners:** ACP maintainers

## Context

The target Task composer lets the user choose deterministic demo or a supported headless runner
without navigating to Settings. The current service resolves process/workspace runtime settings and
snapshots them at run admission, but a frontend-only seamless runner switch would misrepresent that
boundary. Concurrent Task starts must also preserve the existing local single-owner execution model.

## Decision

- Runner selection is admitted per Attempt. A Task stores the desired runner preset; every Attempt
  stores immutable effective runtime mode, provider, model, effort, permissions, execution profile,
  timeout profile, per-step provider overrides and value sources.
- Later Task edits, Settings changes, workspace profile updates or environment changes affect only a
  future Attempt. They cannot mutate queued, active or terminal Attempt history.
- Scope and runner readiness are validated before a provider process starts. Demo/fake identity is
  explicit and never presented as live repository analysis.
- The MVP keeps one global active pipeline Attempt and at most one queued Attempt under the existing
  service admission lease. A further start returns a typed conflict; a queued Attempt belonging to
  another Task is never silently replaced or superseded.
- Async Q&A starts and workspace, runtime/session and Git mutations continue to participate in the
  same lease. Q&A is not a Task Attempt and does not consume the queued pipeline Attempt slot, but a
  conflicting lease owner blocks admission explicitly. Queue and admission results expose exact
  Task, Attempt and run identities.
- A client request/idempotency token protects Attempt creation against double submission. Replaying
  the same accepted token returns the same Attempt; a different start request cannot reuse it.

## Alternatives considered

- Immutable service-session runner with restart/re-attach: rejected as the target because it makes
  ordinary runner selection a process-management workflow and cannot satisfy the accepted composer
  UX. It may remain visible as a temporary limitation until the backend slice lands.
- Unlimited Task queue: rejected because it expands the MVP into a new scheduler and complicates
  workspace/runtime mutation ownership.
- Let Settings update an active run: rejected because it destroys reproducibility and retry audit.

## Consequences

- Orchestrator admission must accept a per-Attempt resolved runner snapshot rather than reading
  mutable settings throughout execution.
- UI readiness, API responses, run history and Task history must show desired and effective values
  separately.
- The single-active/single-queued limit remains an explicit product state, not an implementation
  detail hidden behind disabled controls.

## Links

- Related epics: 23A, 23C, 23F, 23M
- Related ADR: `ADR-20260804-runtime-provider-model-selection.md`
- Related docs: `docs/spec/TASK_SPEC.md`, `docs/spec/API_SPEC.md`,
  `docs/UI_TASK_FIRST_PRODUCT_DESIGN.md`
