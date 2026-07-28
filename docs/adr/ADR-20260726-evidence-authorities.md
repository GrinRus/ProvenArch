# ADR-20260726: Explicit evidence authorities

## Status

Accepted.

## Context

Promoted workspace knowledge, historical run snapshots, immutable QA inputs/answers and QA runtime
diagnostics have different lifecycle and trust semantics. Treating all four as generic workspace
paths permits accidental cross-run or current-state fallback.

## Decision

Public read models use the closed authority vocabulary
`promoted_current | run_snapshot | qa_snapshot | qa_audit`.

- Knowledge inventories only promoted canonical `model/`, `reports/` and `proposals/`, excluding
  `reports/taskruns/**`.
- A QA detail response names the exact run-scoped answer and audit roots.
- A succeeded QA run must have a valid answer and context pack in its own `qa_snapshot`; missing or
  invalid content returns `qa_answer_unavailable`.
- Queued, running, failed and canceled QA runs expose `answer_status=not_produced` and never borrow
  an answer from another run or promoted state.
- Citations are valid only when present in the selected run's context pack.

## Consequences

Clients can preserve selected-run identity and render unavailable states without guessing paths.
Legacy taskrun files remain readable as explicit audit evidence but cannot leak into promoted
Knowledge.
