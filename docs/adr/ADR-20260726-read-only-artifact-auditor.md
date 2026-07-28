# Read-only selected and promoted artifact integrity auditor

- **ADR ID:** ADR-20260726-read-only-artifact-auditor
- **Status:** accepted
- **Date:** 2026-07-26
- **Owners:** ACP maintainers

## Context

Schema-valid, accepted-looking output can still be untrustworthy when it references another run,
leaks execution narration, carries scaffold text, breaks reciprocal citation links or cites a
plausible repository path that does not exist. Live provider assessment cannot be the only way to
reinspect these invariants.

## Decision

Add a provider-free `internal/artifactaudit` scanner. Selected-run mode reads only the exact
run-scoped final index, citation index, validator PASS verdict and indexed staged documents.
Promoted-current mode starts from the same immutable graph and additionally compares canonical
document bytes with their staged source.

The scan is read-only and deterministic. It uses descriptor-backed bounded workspace reads,
symlink-aware repository containment, stable issue codes and fixed budgets for issues, artifacts,
messages and bytes. Reports contain only logical/workspace-relative identities and digests, never
absolute repository paths, timestamps or raw provider diagnostics. The report is a transient HTTP
read model rather than a persisted workspace artifact.

## Consequences

- Historical and current promotion evidence can be rechecked offline and byte-identically.
- Missing/foreign/oversized evidence fails closed with a stable issue code.
- A validator PASS remains necessary but is no longer treated as sufficient integrity evidence.
- New audit classes require focused incident fixtures and must remain provider-free/read-only.

## Links

- Related issue: Epic 22H
- Related docs: `docs/spec/API_SPEC.md`, `docs/ARCHITECTURE.md`, `docs/PLANS.md`
