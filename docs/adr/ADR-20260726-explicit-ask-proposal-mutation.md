# ADR-20260726: Make Ask-to-Proposal an explicit digest-bound mutation

## Status

Accepted on 2026-07-26.

## Context

Ask taskruns are immutable read-only evidence. Turning an answer into a proposal changes the
current workspace and therefore cannot be an implicit side effect of asking or viewing an answer.
The source answer may also become stale between display and confirmation.

## Decision

A succeeded validated QA read exposes SHA-256 `answer_digest`. The operator may explicitly call
`POST /api/qa/runs/<run_id>/proposal-draft` with a title and expected digest. The server re-reads
that exact run's answer/context, validates citations and digest, then publishes an exclusive
`proposals/qa-synthesis-<run-id>-<slug>/` directory atomically.

The package contains human-readable proposal/evidence Markdown and a closed
`source-qa-answer.json` provenance record. Existing packages are never overwritten. Write or rename
failure leaves no visible partial package. The shared admission lease rejects the mutation while
runtime work is active or the workspace generation changes.

ProductShell shows the action only for a succeeded answer with a digest, uses a focus-managed
confirmation, refreshes Git inventory, opens current Changes→Proposals and retains Return to Ask.
No commit or proposal branch is created implicitly.

## Consequences

Ask execution remains read-only. Proposal creation is auditable, optimistic-concurrency safe and
restart-independent because all source authority is persisted in the selected taskrun.
