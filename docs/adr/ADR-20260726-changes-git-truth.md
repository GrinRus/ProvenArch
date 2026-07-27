# Changes route semantics and server-authored Git truth

- **ADR ID:** ADR-20260726-changes-git-truth
- **Status:** accepted
- **Date:** 2026-07-26
- **Owners:** ACP maintainers

## Context

Changes URL views previously selected one shared review panel, while publication readiness was
inferred from client loading text and whether a Git inventory happened to be empty.

## Decision

Represent every Changes route with a discriminated view model and distinct route container.
The Git diff endpoint is authoritative for `clean`, `dirty`, `stale`, `blocked` and `unknown`.
Optional fingerprint comparison identifies stale confirmation; active/pending work is blocked; Git
inventory failure is explicit unknown. The client never opens mutation confirmation from
`stale`, `blocked` or `unknown`.

## Consequences

- Deep links and Back/Forward restore meaningful view semantics.
- Publication cannot be declared ready from a missing or failed request.
- Existing full-inventory fingerprint remains the mutation precondition.

## Links

- Related issue: Epic 22J
- Related docs: `docs/spec/API_SPEC.md`, `docs/ARCHITECTURE.md`, `docs/PLANS.md`
