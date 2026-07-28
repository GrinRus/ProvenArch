# ADR-20260726: Complete advisory Workspace Health without a contract version change

## Status

Accepted on 2026-07-26.

## Context

The original computed Workspace Health endpoint reported only a small set of missing-evidence,
orphan-domain and proposal-shape signals. Canonical workspaces could still contain broken local
links, unresolved model endpoints, duplicate aliases, malformed Markdown or missing proposal
evidence without a stable operator-visible diagnostic.

Persisting a health report or making it a publication gate would create a new authority and change
the product lifecycle. Historical Changes also cannot truthfully reuse a health snapshot computed
from the current promoted workspace.

## Decision

Keep `GET /api/workspace/health` response version `1`, computed and read-only. Extend its stable
issue taxonomy to cover:

- broken or escaping local Markdown links;
- missing model endpoints and owner-team cards, plus duplicate aliases;
- orphan domain and team outputs;
- malformed canonical Markdown, unlinked findings and missing proposal evidence.

The scan follows workspace containment, excludes `reports/taskruns/**`, sorts output
deterministically and must leave workspace bytes unchanged. The current Knowledge and readiness
surfaces may show its summary. Historical Changes must not present it as run evidence.

`pass|warn|fail` remains an advisory health classification. It does not block runtime execution,
Review, Publish or Q&A.

## Consequences

Operators receive actionable current-workspace diagnostics without a schema migration or a new
persisted source of truth. A future blocking policy or stored health artifact requires a separate
schema-first decision.
