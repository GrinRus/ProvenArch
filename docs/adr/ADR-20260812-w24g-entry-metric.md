# W24G entry metric and mechanical-envelope decision

- **ADR ID:** ADR-20260812-w24g-entry-metric
- **Status:** accepted
- **Date:** 2026-08-12
- **Owners:** ACP maintainers

## Context

W24G is conditional: it may remove model-authored mechanical fields only after the completed
W24A–F authority chain demonstrates first-pass success below 95%, more than 10% of otherwise-valid
tasks entering provider repair, or p95 provider invocations above two because of mechanical
identity/path/link/shape failures. The metric must be recorded before selecting a new envelope and
must not turn a provider-free conformance corpus into a live-provider claim.

## Measurement

`fixtures/conformance/w24g-entry-metric.json` is the retained provider-free measurement for the
W24I incident corpus. `internal/conformance.MeasureEntryMetric` computes the three backlog
thresholds from immutable sample fields and ignores non-positive invocation observations. Negative
incident cases remain fail-closed conformance tests; they are not counted as otherwise-valid
successes or repair pressure.

Recorded result:

- 20/20 first-pass-valid observations (100%);
- 0/20 otherwise-valid observations entered provider repair (0%);
- p95 provider invocations = 2;
- 0 mechanical contract failures in the valid trace;
- entry condition = false; decision = defer W24G.

This result is a deterministic conformance baseline, not a substitute for future production/live
telemetry. If a trusted run later crosses a threshold, the next change must add a before/after
measurement and revisit this decision before changing the public envelope.

## Decision

Keep W24G deferred. Do not introduce schema v2, a compiler envelope, prompt changes, or a new
mechanical-field ownership split from this measurement alone. Retain the current strict validator,
provider-authored draft, orchestrator technical candidate, pre-promotion audit and effective
verdict chain.

When the entry condition becomes true, compare these two designs in a follow-up ADR:

1. an orchestrator base envelope plus a bounded, whitelisted semantic patch; or
2. an internal provider-draft format compiled into the existing public artifact contract.

Both designs must preserve provider-authored semantic bytes/provenance, reject arbitrary patch
paths and unknown fields, and replay through the existing full validator and audit.

## Consequences

- W24G has a recorded, reviewable entry decision without weakening validation or release gates.
- W24H's hard three-start budget and canonical live matrices remain unchanged.
- Future telemetry may trigger W24G, but cannot silently change the envelope; it must produce a new
  metric, before/after conformance result and ADR.

## Links

- Related epics: 24G, 24H, 24I
- Related ADR: `ADR-20260811-validation-audit-effective-verdict-authority.md`
- Related docs: `docs/BACKLOG.md`, `docs/PLANS.md`, `docs/spec/PIPELINE_SPEC.md`
