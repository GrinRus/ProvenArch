---
name: acp-docs-sync
description: Обновить затронутые behavior/spec/status документы ACP после изменения реализации, контракта, testing или agent workflow; проверить ссылки и согласованность источников.
---

## Маршрутизация

Прочитать [DOCS_POLICY](../../../docs/DOCS_POLICY.md): там находятся language policy, владельцы
содержания и карта «тип изменения → документы». [AGENT_DEVELOPMENT](../../../docs/AGENT_DEVELOPMENT.md)
связывает код со specs и checks. Не дублировать эти карты в новых инструкциях.

Определить, что изменилось для пользователя или maintainer, и обновить только соответствующие
разделы. Runtime change обычно затрагивает PIPELINE_SPEC/ARCHITECTURE; API spec меняется при
изменении API. Для schema diff использовать `acp-schema-guardian`; user-visible behavior отражать
в CHANGELOG. Documentation-only slice не должен попутно менять runtime prompts или release policy.

Статус брать из evidence active plan и canonical matrix в
[STAKEHOLDER_DOC](../../../docs/STAKEHOLDER_DOC.md); [BACKLOG](../../../docs/BACKLOG.md) хранит
acceptance. Расхождение документов требует проверки evidence, а не предположения о закрытом epic.
Не переписывать статусы соседней remediation программы при локальной правке routing.

## Проверка

Выполнить `make verify-agent-guidance` для структуры skills/plans и локальных ссылок
(прямой эквивалент — `./scripts/run-go.sh test ./internal/docsync`). Проверить смысл ссылок и claims вручную: green check
не доказывает implementation status. Для изменённого контракта дополнительно выполнить
`make contracts-check`; installation/toolchain route — [CONTRIBUTING](../../../CONTRIBUTING.md).

Историю переносить по правилам PLANS/DOCS_POLICY, сохраняя ссылки и evidence. Не поддерживать
вручную вторую stakeholder matrix или копии catalog/runbook constants.
