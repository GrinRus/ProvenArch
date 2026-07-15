# Trustworthy console state contracts

- **ADR ID:** ADR-20260715-trustworthy-console-state-contracts
- **Status:** accepted
- **Date:** 2026-07-15
- **Owners:** ACP maintainers

## Context
The old console could combine current workspace files with historical run selection, present a
filtered Git preview for a full-workspace commit, and imply that a client-side runtime choice was
already effective. These are trust-boundary defects rather than visual inconsistencies.

## Decision
Persist runtime mode per run; expose active/pending coordination explicitly; load historical
evidence only from the selected run staging index; make Git reads return one full inventory and
require its SHA-256 fingerprint plus repository identity for every Publish mutation. Runtime
switching is immediate only before the launcher enters Console; afterward it requires restart.

The UI consumes these public contracts through a five-path shell and one pure workflow selector.
Fake runtime and its artifacts are always identified as deterministic demo evidence.

## Alternatives considered
- Keep UI-only guards: rejected because another client or workspace change can invalidate them.
- Commit only the selected preview: rejected because it changes the existing `git add -A` contract.
- Fall back from a missing snapshot to current files: rejected because it silently crosses run boundaries.

## Consequences
Clients must provide Git confirmation identity and handle typed `409` conflicts. Legacy runs
without persisted runtime metadata display `Unknown`. Deep URL context now extends the five stable
destination paths with explicit setup/run/source/artifact/entity/viewer identity; invalid identity is
sanitized with a notice and never changes evidence source.

## Links
- Related issues/epics: Epic 20 (20A–20I2)
- Related docs: `docs/spec/API_SPEC.md`, `docs/ARCHITECTURE.md`, `docs/PLANS.md`
