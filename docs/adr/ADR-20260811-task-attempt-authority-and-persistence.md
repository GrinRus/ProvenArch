# Task and Attempt authority and persistence

- **ADR ID:** ADR-20260811-task-attempt-authority-and-persistence
- **Status:** accepted
- **Date:** 2026-08-11
- **Owners:** ACP maintainers

## Context

The current pipeline `RunInfo` and runtime `Task` describe execution mechanics. They do not provide
the durable user object required by the Task-first product: one intent across retries, immutable
execution configuration, archive lifecycle and exact result/publication linkage. Inferring a Task
from a run title, brief or recency would make identity unstable and would strand existing run
history during the shell cutover.

## Decision

Introduce a separate product `Task` aggregate and immutable child `Attempt` records.

- A Task owns server-generated `task_id`, mutable current intent (`title`, `goal`, optional context,
  repository/scope selection and desired runner preset), monotonic `revision`, timestamps and
  explicit archive metadata.
- One Attempt maps to exactly one pipeline `run_id`. Admission snapshots the exact Task intent,
  scope and effective runner configuration. Retry of an unsuccessful terminal Attempt and rerun of
  a successful terminal Attempt create a new child Attempt with explicit parent lineage and never
  mutate the parent; a Task cannot create a second root Attempt.
- Task grouping such as `Needs attention`, `Running`, `Ready` and `Completed` is a read model derived
  from Task archive state plus linked Attempt/run state; it is not an independently persisted
  lifecycle authority.
- Persist Task/Attempt identity and immutable snapshots in the versioned workspace control-plane
  registry `reports/taskruns/task-history.json`, with the same atomic current-then-`.last-good`
  recovery policy as run history. Automatic run-artifact retention must not delete Task identity or
  Attempt summaries. No automatic Task deletion is added in the MVP; archive/unarchive is the only
  visibility lifecycle.
- The Task registry is excluded from promoted Architecture, analysis/QA context and analyzed source
  repositories. Detailed execution lifecycle remains authoritative in run history while retained;
  the Task registry keeps the stable linkage, admitted snapshot and terminal summary required after
  detailed run evidence expires.
- Pre-contract runs remain readable legacy execution evidence and are never assigned a synthetic
  Task. An explicit future `Create Task from this run` action may copy intent, but it creates a new
  identity and must not rewrite history.

## Alternatives considered

- Rename pipeline runs to Tasks: rejected because retry lineage, mutable user intent and immutable
  execution configuration would remain conflated.
- Infer Tasks from existing run names or briefs: rejected because the mapping is ambiguous and not
  restart-stable.
- Store Task data in promoted `model/*` or canonical reports: rejected because product workflow
  history is not architecture knowledge.
- Use one file per Task in the first slice: deferred because the existing crash-safe registry pattern
  gives a smaller atomic implementation. The public Task/Attempt contract remains independent of
  the internal registry encoding.

## Consequences

- Task/Attempt schemas and APIs must be implemented before the Task-first shell can become primary.
- Admission and terminalization must coordinate Task and run registries and expose recovery
  diagnostics rather than silently publishing a partially linked identity.
- Existing runs remain accessible without polluting the new Task Inbox with fabricated Tasks.
- The registry can grow with archived Tasks; later retention/export policy requires a separate
  decision and cannot silently delete history.

## Links

- Related epics: 23A, 23C, 23D, 23O
- Related docs: `docs/spec/TASK_SPEC.md`, `docs/spec/API_SPEC.md`,
  `docs/spec/WORKSPACE_SPEC.md`, `docs/UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md`
