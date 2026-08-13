# Task and Attempt contract (W23A1–W23A4 foundation)

Status: **W23A1–W23A4 schemas, durable registry, public Task APIs, Attempt admission and W23
Task-first surfaces are implemented. W24 authority and W25 trusted live evidence remain gated.**

This document fixes the product identity and persistence boundary for the Task-first shell. Current
`/api/pipeline/runs*` behavior remains readable for runtime lifecycle and legacy evidence, while
new Task/Attempt and publication identity is authoritative through the Task registry.

## 1) Goals

- Make Task the durable user object across retry/rerun and service restart.
- Keep Attempt as an immutable admitted execution linked one-to-one with a pipeline run.
- Preserve exact runner, scope, result and publication identities without frontend inference.
- Keep Task history out of promoted Architecture, provider context and analyzed repositories.

## 2) Non-goals

- Hosted/multi-user collaboration, assignment or approval workflows.
- Writing Task state into analyzed source repositories or canonical architecture model files.
- Replacing pipeline run evidence or inferring Tasks for pre-contract runs.
- Unlimited scheduling, selected-file publication or deletion/retention policy for archived Tasks.

## 3) Authority model

| Object | Authority |
| --- | --- |
| Task | Current user intent, revision and archive lifecycle in the Task registry |
| Attempt | Immutable admitted intent/scope/runner snapshot and parent lineage in the Task registry |
| Run | Detailed execution lifecycle, artifacts, logs, progress and runtime diagnostics |
| Architecture | Current validator-promoted workspace knowledge; independent of Task archive/review state |
| Change set | Exact run-snapshot semantic comparison plus separate full-workspace Git inventory |
| Publication | Server-authored association created only after a successful Git mutation |

Task/Attempt identity never derives from run name, recency, current route or brief text. Explicit
unknown/stale identity fails visibly and never falls back to another Task, Attempt or run.

## 4) Persistence

The planned authoritative registry paths are:

```text
reports/taskruns/task-history.json
reports/taskruns/task-history.json.last-good
```

The current file is a versioned registry written atomically. A valid current file is authoritative;
missing, unreadable, malformed or unsupported-version current state may recover from valid
`.last-good` with a bounded service diagnostic. Candidate state is published in memory only after
the current file is durable. A `.last-good` write failure does not roll back a durable current write
but remains visible in diagnostics.

Task history is workspace-owned project data. It is excluded from promoted Architecture, semantic
compilation, Ask/QA context and provider read roots. Automatic run retention may remove detailed
run evidence, but it must retain Task identity, admitted Attempt snapshot and terminal summary.
There is no automatic Task deletion in the MVP.

## 5) Task v1 public shape

The 23A schema slice must define at least:

- `version`;
- opaque server-generated `task_id`;
- `title`, `goal`, optional `context`;
- explicit desired repository/scope selection;
- desired runner preset reference;
- monotonic `revision`;
- `created_at`, `updated_at`, `last_activity_at`, optional `archived_at`;
- ordered Attempt references/summaries;
- explicit result/review/publication linkage or unavailable state.

Only `open|archived` is persisted as Task visibility lifecycle. Inbox groups such as
`needs_attention|running|ready|completed` are derived from archive state and linked Attempt/run
state. `updated_at` and list ordering must not depend on browser polling frequency.

Task intent is editable between Attempts through optimistic concurrency (`expected_revision`). An
accepted edit changes only the Task's desired state and a later Attempt; it never changes an
existing Attempt snapshot.

## 6) Attempt v1 public shape

Every Attempt includes at least:

- opaque server-generated `attempt_id`, exact `task_id` and exact one-to-one `run_id`;
- optional `parent_attempt_id` plus retry/rerun reason;
- immutable `pipeline`, client `idempotency_key` and canonical request fingerprint;
- admitted Task revision and immutable goal/context/scope snapshot;
- desired runner preset plus immutable effective runtime mode, provider, model, effort,
  permissions, timeouts, execution settings, per-step overrides and resolution sources;
- admitted/queued/started/finished timestamps as applicable;
- terminal summary and retained-evidence state;
- exact run-snapshot/result/change/publication references when available.

For queued and running Attempts, `terminal_summary` is explicitly `null`. Once terminal, it is an
object containing the terminal status, bounded error/summary fields and retained-evidence state.
An available outcome repeats the exact `attempt_id` and `run_id` linkage so a standalone Attempt
payload remains self-describing.

Attempt status follows the linked run lifecycle while that run is retained. Terminalization copies
a bounded immutable summary into Task history so archive/history remains useful after detailed run
retention. A retry/rerun always creates a new child Attempt.

## 7) Admission and coordination

- Runner and scope validation completes before provider execution.
- One service owns at most one active pipeline Attempt and one queued Attempt.
- A further start returns a typed conflict and cannot replace another Task's queued Attempt.
- Async Q&A starts and workspace/session/runtime/Git mutations use the existing shared admission
  lease. Q&A does not become a Task Attempt or consume the single queued pipeline Attempt slot, but
  an active/queued conflicting owner still blocks admission explicitly.
- Attempt creation requires a client idempotency token. Repeating an accepted token returns the same
  Task/Attempt/run identity; a token cannot represent a different request.
- Queue/readiness/conflict responses expose exact Task, Attempt and run identities.

## 8) Planned API surface

The implemented 23A surface exposes versioned JSON contracts for:

- `POST /api/tasks` — create Task intent;
- `GET /api/tasks` — stable cursor pagination and filters;
- `GET /api/tasks/<task_id>` — exact Task detail;
- `PATCH /api/tasks/<task_id>` — update desired intent with `expected_revision`;
- `POST /api/tasks/<task_id>/archive` and `/unarchive`;
- `POST /api/tasks/<task_id>/attempts` — admit/start an Attempt idempotently;
- `GET /api/tasks/<task_id>/attempts/<attempt_id>` — exact Attempt detail;
- `POST /api/tasks/<task_id>/attempts/<attempt_id>/retry` — create an explicit child Attempt.

Attempt admission accepts `{idempotency_key, pipeline?: init|refresh, intent?: start|queue}`. Retry
accepts the same fields plus an optional `reason`; a retry requires a terminal parent and always
creates a new child identity. Duplicate keys with the same canonical request fingerprint return the
existing Attempt, while a reused key for a different Task revision/options returns
`409 idempotency_conflict`. Capacity errors are typed (`run_active` or `attempt_queue_full`) and
never supersede another Task's queued Attempt.

There is no Task delete endpoint in the MVP. List ordering is stable by
`(last_activity_at desc, task_id)`; pagination cursors must not be offset-only. Filters and explicit
identities are URL-restorable. Exact request/response JSON remains part of the schema-first 23A
implementation and must not be invented independently in frontend code.

## 9) Legacy runs

Runs created before the Task contract remain readable through existing run/snapshot APIs and an
explicit read-only legacy UI surface. They are not listed as fabricated Tasks. A legacy route may
show a migration notice or offer an explicit `Create Task from this run` action, but it cannot
silently select or create a Task and cannot rewrite the historical run.

## 10) Result and publication linkage

Automatic validator-approved promotion updates current Architecture independently of Task review or
Git publication. Task Outcome may link an Attempt to its exact run snapshot and semantic comparison.

`Published` is never inferred from a clean worktree, branch, latest commit or recency. A successful
Git mutation creates a server-authored publication association containing the initiating
Task/Attempt/run identities when present, action, branch/base/head identity, exact full-workspace
inventory fingerprint and resulting commit/branch identity. The association does not claim that all
committed files belong exclusively to one Task.

## 11) Compatibility and failure behavior

- Current run APIs remain readable during migration and for legacy history.
- Invalid Task/Attempt/scope/runner identity fails before provider execution.
- Registry persistence failure leaves the previously durable Task view authoritative.
- Partial Task/run linkage is surfaced as a recovery diagnostic; API/UI cannot silently attach the
  Attempt to a different run.
- Historical Task registry versions require an explicit dual-read decision before a writer version
  change.

## 12) Required implementation verification

- Schema/API round trips for create/list/read/update/archive/unarchive.
- Restart and `.last-good` recovery, primary/last-good write faults and no partial in-memory publish.
- Parent-child Attempt lineage and immutable effective configuration after Settings/env changes.
- Concurrent start, duplicate idempotency token and queue-full behavior.
- Run retention with retained Task/Attempt terminal summary.
- Legacy runs remain readable and are never synthesized into Tasks.
- Task/run/snapshot/publication identity never falls back across authorities.
- No Task bytes enter analyzed repositories, promoted Architecture or provider/QA context.

## Links

- `docs/adr/ADR-20260811-task-attempt-authority-and-persistence.md`
- `docs/adr/ADR-20260811-per-attempt-runner-admission.md`
- `docs/adr/ADR-20260811-task-change-publication-linkage.md`
- `docs/UI_TASK_FIRST_PRODUCT_DESIGN.md`
- `docs/UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md`
- `docs/BACKLOG.md` Epic 23
