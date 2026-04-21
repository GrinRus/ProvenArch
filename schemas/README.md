# Schemas

Эта папка содержит machine-readable контракты.

## Файлы
- `workspace.schema.json` — JSON Schema для `workspace.yaml`.
- `shard-pack-manifest.schema.json` — collect/runtime authored shard manifest.
- `final-run-index.schema.json` — canonical staged final set index.
- `citation-index.schema.json` — citation graph for promoted reports.
- `validator-verdict.schema.json` — validator/findings primary verdict surface.

## Правила
Меняя schema, обновляйте также:
- `docs/APPENDIX_SCHEMAS.md`
- `docs/spec/*`
- `examples/*`
- валидаторы и фикстуры
