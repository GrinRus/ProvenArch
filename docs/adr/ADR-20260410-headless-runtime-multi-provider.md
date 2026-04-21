# ADR-20260410-headless-runtime-multi-provider.md

- **ADR ID:** ADR-20260410-headless-runtime-multi-provider
- **Status:** accepted
- **Date:** 2026-04-10
- **Owners:** ACP maintainers
- **Supersedes:** [ADR-20260329-implementation-stack-go](ADR-20260329-implementation-stack-go.md)

## Context

Current MVP implementation supports two production headless runtime providers:
- `claude-code` (default)
- `qwen-code` (optional)

Required CI baseline must stay deterministic and independent from live provider binaries (`--runtime fake`).
Provider selection keeps a global CLI/env fallback (`--runtime-provider` / `ACP_RUNTIME_PROVIDER`), but effective provider resolution inside a run is step-scoped.

## Decision

ACP MVP runtime policy is **headless multi-provider + fake baseline**:
- keep `fake` as required deterministic CI runtime mode;
- keep headless provider IDs fixed to `claude-code` and `qwen-code`;
- keep public API surface stable while moving runtime semantics to artifact-only contracts;
- resolve provider per pipeline step, with precedence `workspace step override > CLI/env global provider > claude-code`;
- support direct local binaries for providers through command envs:
  - `ACP_CLAUDE_CMD` (default `claude-code`, direct `claude` supported)
  - `ACP_QWEN_CMD` (default `qwen`)

## Rationale

- Runtime diversity is required for local-first usage across different developer environments.
- Step-scoped provider selection preserves API stability while allowing mixed-provider runs without adding per-request hosted routing semantics.
- Keeping the public API stable while removing semantic stdout payloads limits migration risk and protects deterministic fixtures/tests.

## Consequences

- Docs and runbooks must describe provider-aware behavior consistently.
- Required CI continues to rely on `--runtime fake`; live provider checks remain optional smoke/e2e surfaces.
- Batch quality reporting and semantic hard-fail checks are documented as evaluator behavior, not single-run runtime contract changes.

## Follow-ups

- Additional headless providers remain out of MVP scope and require a separate ADR/slice.
- Any future provider expansion must preserve the artifact-only runtime contract and API error-code surface.
