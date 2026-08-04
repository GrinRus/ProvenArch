# Provider-scoped runtime model selection

- **ADR ID:** ADR-20260804-runtime-provider-model-selection
- **Status:** accepted
- **Date:** 2026-08-04
- **Owners:** ACP maintainers

## Context

ACP supports multiple headless providers, but model and reasoning effort were previously
implicit in each installed CLI. That made live runs difficult to reproduce and made a
provider-specific migration require ad-hoc environment changes without a visible workspace contract.

## Decision

Persist optional provider-scoped settings under `runtime.profile.providers`. Resolve each
field with the precedence `provider environment > workspace manifest > provider-native
default`, and snapshot effective values and their sources when a run is accepted. Omitted
values are not sent as CLI arguments. The first live E2E pin is Codex `gpt-5.6-luna` with
effort `high`; Claude and Qwen remain on native defaults unless explicitly configured.

## Consequences

- Workspace/API/UI can show persisted and effective settings.
- Run history is reproducible across later environment changes.
- Provider capability validation stays explicit and prevents unsupported effort values.
- Per-step model catalogs, fallback chains, and automatic model selection remain out of scope.
