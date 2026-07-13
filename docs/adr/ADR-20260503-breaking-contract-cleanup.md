# ADR-20260503-breaking-contract-cleanup

- **ADR ID:** ADR-20260503-breaking-contract-cleanup
- **Status:** accepted
- **Date:** 2026-05-03
- **Owners:** ACP maintainers

## Context
MVP contract surfaces had leftover compatibility affordances that increased orchestrator/runtime complexity: `repos[].analysis.role`, `repo_selection` execution branches, and collect-manifest legacy aliases that were detected by an ad hoc scanner before schema validation. The MVP constraint is deterministic, Git-friendly local-first output, so malformed legacy artifacts should fail fast instead of being repaired silently.

## Decision
Remove `repos[].analysis.role` from the active workspace schema and reject manifests that still contain it. Remove runtime repo-selection state from orchestrator execution and always derive repo scopes directly from `workspace.yaml`. Move known collect legacy alias rejection into `schemas/shard-pack-manifest.schema.json` and keep runtime artifact validation read-only.

2026-07-13 update: make minimum collect evidence bindings part of the active shard-pack
contract. A collect manifest must contain at least one authored document, at least one citation,
non-empty `documents[].citation_ids`, and non-empty `citations[].claim_ids` /
`citations[].document_ids`. Sparse/no-evidence collect packets are contract drift and must fail
before checkpoint/apply instead of being treated as valid minimal output.

## Alternatives considered
- Keep migration shims for old manifests. Rejected because it preserves hidden contract branches and weakens deterministic failure behavior.
- Keep the pre-schema collect scanner. Rejected because it duplicates schema responsibility and requires a second validation vocabulary.

## Consequences
Workspace manifests with `repos[].analysis.role` are invalid. Collect shard manifests using legacy aliases such as `covered_topics`, `question`, `relation`, `source`, `target`, finding `summary`/`inference`, `evidence_citation_ids`, top-level `step_contract`, or `compatibility` fail through schema/contract validation. Collect shard manifests with empty documents/citations or empty document/citation binding arrays also fail through schema/contract validation. Existing valid all-repo runtime behavior is unchanged.

## Links
- `schemas/workspace.schema.json`
- `schemas/shard-pack-manifest.schema.json`
- `docs/spec/WORKSPACE_SPEC.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/APPENDIX_SCHEMAS.md`
