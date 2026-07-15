# Current workspace knowledge read model

- **ADR ID:** ADR-20260715-current-knowledge-read-model
- **Status:** accepted
- **Date:** 2026-07-15
- **Owners:** ACP maintainers

## Context
Knowledge previously inferred a map from selected-run artifact file names. That could hide malformed
promoted files, invent topology from naming conventions and blur historical snapshots with the
current workspace.

## Decision
Expose a read-only `GET /api/knowledge` read model for current promoted workspace content. Parse
entities and edges from their canonical YAML fields, resolve every edge against validated entity
IDs, inventory readable model/report/proposal artifacts independently, and retain valid records when
another file is malformed. The response explicitly reports `available|partial|unavailable` and typed
issues. It does not claim a promoting run unless a future authoritative association contract exists.

Historical review remains run-snapshot based in Changes. Ask is a current-workspace read-only utility;
its citations open the same safe evidence viewer and preserve a return route without changing review
or publication acceptance.

## Alternatives considered
- Derive nodes and edges from paths: rejected because filenames are presentation metadata, not model truth.
- Fail the whole endpoint on one malformed file: rejected because valid promoted knowledge remains useful.
- Attach the latest successful run ID: rejected because recency does not prove promotion association.

## Consequences
Knowledge can be partial while still searchable and accessible through a table fallback. Consumers
must display typed issues and source identity. The response is additive API state, not a persisted
workspace schema.

## Links
- Related slice: Epic 20 / 20J2
- Related docs: `docs/spec/API_SPEC.md`, `docs/ARCHITECTURE.md`, `docs/PLANS.md`
