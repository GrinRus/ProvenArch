# Schemas

Эта папка содержит machine-readable контракты.

## Файлы
- `workspace.schema.json` — JSON Schema для `workspace.yaml`.
- `shard-pack-manifest.schema.json` — collect/runtime authored shard manifest.
- `final-run-index.schema.json` — canonical staged final set index.
- `citation-index.schema.json` — citation graph for promoted reports.
- `validator-verdict.schema.json` — validator/findings primary verdict surface.
- `effective-verdict.schema.json` — orchestrator-owned technical promotion authority.
- `qa-answer.schema.json` — async Ask runtime answer artifact.
- `source-qa-answer.schema.json` — immutable provenance record for an explicit Ask-to-Proposal draft.
- `task.schema.json` — durable product Task intent and explicit unavailable/linkage states.
- `attempt.schema.json` — immutable admitted Task Attempt snapshot.
- `task-history.schema.json` — versioned Task/Attempt registry with bounded diagnostics.

## Правила
Меняя schema, обновляйте также:
- `docs/APPENDIX_SCHEMAS.md`
- `docs/spec/*`
- `examples/*`
- валидаторы и фикстуры
