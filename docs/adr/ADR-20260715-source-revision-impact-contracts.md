# ADR: Persist source revisions and advisory refresh impact before execution

- Status: Accepted
- Date: 2026-07-15

## Context

The refresh pipeline previously ran the complete analysis without a durable, deterministic account of which source revisions and analysis inputs were compared. Future no-op, affected-only execution and surgical promotion require an explicit baseline and fail-closed dependency plan; inferring either from the current workspace or from a legacy run would make stale evidence look authoritative.

## Decision

Every `init|refresh` persists a schema-validated `source-revisions.json` before provider execution. Its baseline can only be the newest prior successful `init|refresh` with a matching valid final index, PASS validator verdict and valid source revision artifact. Legacy or unreadable runs are not inferred. The analysis-input fingerprint covers repo source/scope configuration, imported docs, charter and skills, and excludes generated outputs and process provider settings. Resolved absolute checkout paths are never persisted.

Every refresh also persists a schema-validated `refresh-impact-plan.json` before collect. It uses a complete rename/copy-aware Git delta, a hard 10,000-path mapping limit, prior shard/domain/citation/final-index dependencies and conservative typed fallback reasons. Dirty worktrees, missing/unavailable revisions, rewritten history, changed/unreadable analysis inputs, unreadable prior evidence and unmapped in-scope paths require `full_refresh_required`.

The impact contract is explicitly `advisory`. No-op and selective decisions are applied only by
the factual execution layer. Every successful collect shard receives an orchestrator-owned
internal integrity sidecar with full repo/domain/shard/path/source identity and a deterministic
SHA-256 inventory. Selective replay requires the sidecar, manifest and successful runtime metadata
to agree with the current plan and source range. Preserved final documents are copied from the
baseline taskrun staging snapshot, never from mutable canonical workspace paths.

## Consequences

- Advisory planning and factual execution are separate persisted contracts: `refresh-impact-plan.json` is immutable, while `refresh-execution.json` records no-op/selective/full behavior and conservative fallback.
- Selective collect reuses prior shard packs only after full identity and digest verification;
  unavailable, legacy, symlinked or mismatched baseline evidence forces full execution before
  providers start.
- Publication decisions are independently auditable through `refresh-materialization.json`; preserved content carries baseline provenance and digest.

- Each run has reviewable baseline evidence and deterministic planning output.
- Initial and legacy workspaces safely fall back without blocking the existing full pipeline.
- Required CI remains provider-independent and can exercise all decisions with synthetic Git output and fixtures.
- Taskrun storage grows by the public planning/execution artifacts plus one small internal integrity
  sidecar per successful collect shard.
- Current stale/preserved lists are candidates, not proof that an artifact was actually retained or republished.
