# Typed validation issues and bounded recovery routing

- **ADR ID:** ADR-20260726-typed-recovery-routing
- **Status:** accepted
- **Date:** 2026-07-26
- **Owners:** ACP maintainers

## Context

Provider artifact validators produce detailed composed errors. Recovery previously searched those
strings again to select collect shape repair, draft enrichment cleanup or provider-unavailable
classification. Reordering simultaneous problems could therefore change the selected transition.

## Decision

Normalize each validation failure once into a deterministic internal set of issues with stable
`code`, `class` and optional bounded `path`. Recovery predicates consume only this set. Specialized
single-class transitions reject competing issue classes. Each focused transition names its exact
target stage and has a one-attempt budget; seeing the same target stage again terminates through the
existing `runtime_contract_failed` surface.

Provider command availability markers and authored document content checks remain text analysis,
because they are inputs rather than validator repair routing.

## Consequences

- Equivalent Claude/Qwen/Codex issue sets route identically regardless of diagnostic order.
- Missing evidence paths retain structured, sorted and deduplicated identities.
- Public provider and artifact schemas do not change.
- New validation categories must add one boundary classification code and tests before they can
  participate in recovery.

## Links

- Related issue: Epic 22G
- Related docs: `docs/spec/PIPELINE_SPEC.md`, `docs/ARCHITECTURE.md`, `docs/PLANS.md`
