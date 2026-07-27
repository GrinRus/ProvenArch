# ADR-20260726: Evidence Viewer identity and resource budgets

## Status

Accepted.

## Context

Relative evidence links previously discarded the selected document directory, unknown sources were
presented as Live, and artifact reads/rendering had no explicit size budget.

## Decision

- Local links resolve relative to the selected canonical document. Traversal above the authority
  root and links into `reports/taskruns/**` are disabled in promoted/run-snapshot views.
- Run-snapshot navigation only opens canonical paths present in the selected run inventory; it does
  not fall back to a raw workspace path.
- Provenance is the closed UI state `Demo | Live | Unknown`; absence of runtime provenance means
  Unknown, not Live.
- Diff UI is exposed only with explicit content and identifies both sources.
- Generic artifact reads are limited to 2 MiB and return `artifact_too_large` (HTTP 413). Rendered
  Markdown/Mermaid is limited to 512 KiB; larger bounded text remains available in Raw.
- Partial/unavailable/error state and typed issues are displayed with the selected evidence.

## Consequences

Broken or malicious links cannot silently switch run/source identity. Oversized and malformed
artifacts degrade to readable typed states without replacing the selected document or crashing the
viewer.
