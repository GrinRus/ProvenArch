---
name: acp-schema-guardian
description: Синхронизировать изменения контрактов ACP — workspace, Task/Attempt, model, runtime manifests и validator verdict — со specs, Go validation, примерами и regression tests.
---

## Найти владельцев контракта

Начать с [schemas](../../../schemas) и [APPENDIX_SCHEMAS](../../../docs/APPENDIX_SCHEMAS.md).
Выбрать семантическую spec: [WORKSPACE_SPEC](../../../docs/spec/WORKSPACE_SPEC.md),
[TASK_SPEC](../../../docs/spec/TASK_SPEC.md), [MODEL_SPEC](../../../docs/spec/MODEL_SPEC.md),
[API_SPEC](../../../docs/spec/API_SPEC.md) или [PIPELINE_SPEC](../../../docs/spec/PIPELINE_SPEC.md).

Проверить реальный consumer: `internal/contracts` для artifacts/model, `internal/tasks` для
Task/Attempt, `internal/workspace` для manifest, `internal/runtimedrafts` для draft artifacts,
`internal/api` и `ui/src/lib` для API. JSON Schema задаёт форму; Go validators также обеспечивают
identity, references, containment и semantic invariants.

## Синхронный slice

- Зафиксировать изменение shape/семантики и поведение existing/invalid/unsupported-version inputs.
  Artifact-only runtime читает required files; stdout/stderr не являются semantic result contract.
- Обновить только затронутые schema/spec, Go types/validators, examples, fixtures и
  APPENDIX_SCHEMAS. Записать rationale в relevant [ADR](../../../docs/adr/README.md).
- Для `workspace.yaml` держать вместе `schemas/workspace.schema.json`, WORKSPACE_SPEC и
  `internal/workspace` validation; для API проверить frontend consumer и typed unavailable states.
- Добавить regression на конкретный invalid case, сохранив valid baseline. Не ослаблять parser,
  identity или evidence validation ради прохождения нового fixture/provider output.

## Проверка

После [bootstrap](../../../CONTRIBUTING.md) запустить `make contracts-check`, затем tests владельца
контракта, например `./scripts/run-go.sh test ./internal/contracts` или
`./scripts/run-go.sh test ./internal/tasks ./internal/workspace`. Для materialization/golden изменений
применить `acp-test-fixtures`; для документации — `acp-docs-sync`. Завершить implementation slice
полным DoD из [AGENTS](../../../AGENTS.md).
