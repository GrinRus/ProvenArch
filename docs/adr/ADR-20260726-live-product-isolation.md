# Bidirectional live harness and product isolation

- **ADR ID:** ADR-20260726-live-product-isolation
- **Status:** accepted
- **Date:** 2026-07-26
- **Owners:** ACP maintainers

## Context

An external qualification harness must observe the product, not author the run history it later
claims to assess. Conversely, ambient qualification identity inherited by provider subprocesses can
be copied into generated architecture content even when product prompts do not intentionally use it.

## Decision

Provider commands always receive a filtered ambient environment. Keys whose semantic segments denote
external test/release orchestration identity are omitted before provider-specific overrides are
merged. Normal toolchain, credential, locale and proxy environment remains available.

The live frontend snapshot captures and copies the product-authored run history byte-for-byte. It
does not reconstruct a succeeded run or artifact inventory. Live-only readability/assessment code
lives under `ui/e2e`, while production `ui/src` exposes only product behavior. Bidirectional static
tests cover Go, Python and TypeScript ownership.

## Consequences

- Provider output cannot inherit qualification labels from ambient environment.
- Snapshot UI inspection remains black-box and cannot turn a failed/absent product run into success.
- Harness and product can evolve independently across public CLI/API/UI/artifact contracts.
- New harness helpers must stay outside product source trees and may not import internal selectors.

## Links

- Related issue: Epic 22I
- Related docs: `docs/ARCHITECTURE.md`, `docs/spec/PIPELINE_SPEC.md`, `docs/TESTING_STRATEGY.md`
