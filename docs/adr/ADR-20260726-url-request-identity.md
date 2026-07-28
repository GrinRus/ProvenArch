# Immutable URL and request identity

- **ADR ID:** ADR-20260726-url-request-identity
- **Status:** accepted
- **Date:** 2026-07-26
- **Owners:** ACP maintainers

## Context

An artifact path alone does not identify its bytes: the same canonical path can exist in multiple
runs, authorities and viewer states. Browser history can also restore invalid explicit enums.

## Decision

Bind preview requests to run, source authority, canonical/read paths and viewer mode plus a monotonic
generation. Starting a request aborts the previous generation and only the current token may publish.
Canonicalize invalid explicit route enums through visible notice and `replaceState`; reserve
`pushState` for user navigation.

## Consequences

- Late A→B→A responses cannot overwrite current evidence.
- Back/Forward restoration is idempotent and invalid URL state does not accumulate history entries.
- New artifact consumers must include every authority/selection dimension in their request key.

## Links

- Related issue: Epic 22K
- Related docs: `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md`, `docs/PLANS.md`
