# ADR-20260726: Validate final/citation identity as one run-scoped graph

## Status

Accepted on 2026-07-26.

## Context

The final-run and citation-index schemas describe each file independently. Correct publication also
depends on cross-file properties: one run identity, reciprocal document links, globally unique
claim/citation ids and evidence that resolves to an actual file in the selected repository.
Selective refresh can otherwise preserve valid bytes whose references were removed from the new
graph.

## Decision

Keep the existing JSON shapes and validate the assembled pair as one run-scoped graph:

- both run ids, the citation-index path and staged paths must belong to the active run;
- citation and claim ids are globally unique and document links are reciprocal;
- every citation repo/path resolves after symlink evaluation to a regular file inside the resolved
  repository root;
- key Architecture Home/findings/proposal documents cannot claim complete citation coverage with an
  empty citation set;
- selective refresh verifies that preserved document citations and citation document ids still
  exist before copying immutable baseline bytes.

Duplicate provider-authored claim ids continue to use deterministic shard-suffixed repair. Low
explicit coverage is exposed separately by advisory Workspace Health and does not become a new
publication policy.

## Consequences

Foreign-run, dangling and out-of-root evidence fails before promotion without a schema migration.
Future claim-ledger semantics remain a separate K5 design.
