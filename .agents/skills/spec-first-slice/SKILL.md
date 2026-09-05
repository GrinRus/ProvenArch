---
name: acp-spec-first-slice
description: Спланировать новую фичу ACP как минимальный проверяемый slice с контрактом, границами кода и acceptance. Для read-only review не запускает реализацию.
---

## Выбор slice

Прочитать relevant row в [карте разработки](../../../docs/AGENT_DEVELOPMENT.md#карта-изменений),
затем existing active plan в [PLANS](../../../docs/PLANS.md) и acceptance в
[BACKLOG](../../../docs/BACKLOG.md). Новый запрос может не иметь backlog ID; не подменять его
старым epic и не переоткрывать завершённый slice. Проверить ownership/dependencies соседних работ.

Сформулировать наблюдаемый before/after результат, non-goals и stop condition. Указать затронутые
модули, authoritative spec/schema и проверку результата. Для Task/Attempt начать с
[TASK_SPEC](../../../docs/spec/TASK_SPEC.md), для pipeline/runtime — с
[PIPELINE_SPEC](../../../docs/spec/PIPELINE_SPEC.md); прочие маршруты находятся в карте.

## План реализации

- Если меняется public contract, сначала определить compatibility и синхронный набор
  schema/spec/type/validator/fixture edits через `acp-schema-guardian`.
- Выбрать существующие focused tests и нужный negative/recovery case. Model/materialization changes
  маршрутизировать через `acp-test-fixtures`; runtime prompt changes отделять от SWE guidance.
- Для многошаговой задачи обновить или создать ExecPlan по PLANS; записать acceptance, зависимости,
  ожидаемые файлы и evidence. Небольшая локальная правка не требует отдельного большого плана.
- При implementation запросе перейти к согласованному локальному slice; при design/review запросе
  вернуть план. Не добавлять release, новые dependencies или чужую remediation работу к scope.

Проверки и подготовка среды: [CONTRIBUTING](../../../CONTRIBUTING.md). Полный DoD относится к
завершённому implementation slice; создание плана само по себе не требует live run.
