# Validation, audit and effective verdict authority

- **ADR ID:** ADR-20260811-validation-audit-effective-verdict-authority
- **Status:** accepted
- **Date:** 2026-08-11
- **Owners:** ACP maintainers

## Context

The provider-authored `validator-verdict.json` currently combines technical PASS/FAIL language with
semantic findings/questions. The completed provider-free auditor is read-only and on-demand, not a
mandatory pre-promotion gate. Epic 24 must prevent provider opinion from overriding deterministic
integrity while avoiding a circular dependency where audit requires a final verdict that itself must
include audit errors.

## Decision

Use a four-stage authority chain:

1. Preserve provider-authored `validator-verdict.json` as immutable draft evidence. Task-aware
   admission and consistency validation must succeed before any advisory content is considered.
2. Build an internal orchestrator-owned technical candidate from the ordered deterministic issue set
   after assembly and allowed deterministic repairs. Any candidate error fails before audit.
3. For a candidate PASS, run the provider-free selected-run audit over the exact current-run indexes,
   staged documents and resolved repository roots before the first canonical write. The audit reads
   the candidate authority but never repairs or rewrites it.
4. Persist an orchestrator-owned effective verdict that includes deterministic validation and audit
   issues. Promotion is allowed only when that effective verdict is PASS.

The effective verdict owns final PASS/FAIL, checked paths, orchestrator fixed paths, technical issue
codes, audit summary and recovery-budget evidence. Provider findings/questions remain advisory
semantic candidates and become visible only after their evidence and graph references validate.
Public API/report surfaces identify provider draft and effective authorities separately; they never
compare provider prose to determine technical status.

The effective persisted shape is versioned separately from the provider draft. Historical v1
provider verdicts remain readable and are never rewritten; lack of an effective verdict is reported
as legacy/unavailable rather than inferred.

## Alternatives considered

- Rewrite the provider verdict in place: rejected because it destroys authorship/provenance and
  makes fixed paths and technical fields ambiguous.
- Let audit consume the final effective verdict: rejected because audit errors must contribute to
  that final verdict, creating circular authority.
- Keep audit on-demand only: rejected because a validator PASS is necessary but not sufficient to
  protect canonical promotion.
- Let provider FAIL veto a deterministic clean snapshot: rejected because unsupported technical
  opinion belongs in advisory findings/questions, not the effective gate.

## Consequences

- Epic 24 keeps delivery order `24A -> 24B -> 24C -> 24D -> 24E -> 24F`: 24E introduces the
  pre-audit candidate/gate, and 24F persists/exposes the final effective verdict.
- Promotion must prove zero canonical writes after audit error and preserve the previous generation.
- New effective-verdict schemas, fixtures, API fields and historical interpretation require full
  schema-guardian synchronization during implementation.
- W25 consumes only public effective authority and audit/budget fields; provider draft opinion
  cannot satisfy the release gate.

## Links

- Related epics: 24A–24I, 25B
- Related ADRs: `ADR-20260726-read-only-artifact-auditor.md`,
  `ADR-20260726-typed-recovery-routing.md`
- Related docs: `docs/spec/PIPELINE_SPEC.md`, `docs/spec/API_SPEC.md`, `docs/BACKLOG.md`
