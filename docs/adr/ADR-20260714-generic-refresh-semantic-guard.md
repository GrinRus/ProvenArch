# ADR-20260714-generic-refresh-semantic-guard

- **ADR ID:** ADR-20260714-generic-refresh-semantic-guard
- **Status:** accepted
- **Date:** 2026-07-14
- **Owners:** ACP maintainers

## Context

Refresh collect runs can receive semantic candidates that are not architecture model facts for the
assigned repo/domain. The risky cases are runtime/provider/process metadata leaking into
`semantic.entities` and explicit off-scope candidates whose evidence points at a different repo than
the refresh shard was assigned to analyze.

The previous guard helpers were too narrow and not reliably active in the apply path: they carried a
business-domain term list plus a special-case domain whitelist. That approach is brittle for a
local-first architecture tool because valid customer domains can use any vocabulary.

## Decision

Use a generic evidence-scope semantic guard for `refresh.step1.collect` only.

Before model apply and before shard packs are aggregated into staged final indexes, ACP filters:

- runtime/provider/process metadata candidates identified by generic runtime markers or runtime
  artifact evidence paths;
- candidates whose explicit semantic provenance repo does not match the assigned refresh
  `repo_scopes`.

Filtered candidates produce deterministic `semantic_guard.refresh.*` findings and run warnings. The
guard does not inspect product-domain terms and does not use hidden business-domain whitelists.
`init.step1.collect` behavior remains unchanged.

## Alternatives considered

- Keep the existing term-list guard. Rejected because it encodes one observed off-topic corpus and
  can falsely reject valid customer domains.
- Reject the whole shard when any candidate is off-scope. Rejected because a refresh shard can
  contain mostly valid evidence and should keep same-scope facts while surfacing diagnostics.
- Move the policy into schemas. Rejected because repo-scope assignment is runtime context, not a
  static shard-pack JSON shape rule.

## Consequences

Refresh semantic model apply and staged final indexes now use the same guarded snapshot. Legitimate
same-scope entities survive. Runtime metadata and explicit off-scope facts are removed from the
model instead of polluting derived diagrams/cards, while deterministic findings/warnings make the
filtering visible to operators and tests.

No shard-pack schema, workspace schema, provider list or live release matrix changes are required.

## Links

- `docs/BACKLOG.md` — Epic 19 `19Q Generic refresh semantic guard`
- `docs/ARCHITECTURE.md`
- `internal/orchestrator/semantic_utils.go`
- `internal/orchestrator/runtime_task_apply.go`
