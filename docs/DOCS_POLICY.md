# Политика документации

## Язык и сохранение содержания

- Repository entrypoint `README.md`: **EN**; detailed stakeholder/engineering документы
  преимущественно на **RU**. Localized variants используют явный suffix (`README.ru.md`, `*.en.md`).
- Смену языка документа объяснять в PR/commit; это не требует перевода остальных документов.
- Крупный документ разделять по теме, сохраняя ссылки. Завершённые планы архивировать по
  [PLANS.md](PLANS.md); большие сокращения объяснять ссылкой на архив или Git history.

## Источники и обновления

Контрактная форма принадлежит `schemas/*`, семантика — `docs/spec/*`. Код и тесты подтверждают
фактическое поведение. Canonical matrix в [STAKEHOLDER_DOC](STAKEHOLDER_DOC.md) хранит статус
реализации; [PLANS](PLANS.md) — active work/dependencies, [BACKLOG](BACKLOG.md) — acceptance.
Обнаруженное противоречие не разрешать предположением о завершённости работы или release readiness.

Обновлять документы, смысл которых меняет slice; одна правка кода не требует каскадного
переписывания всех overview/spec документов.

| Изменение | Документы и сопутствующие surfaces |
| --- | --- |
| Public schema/contract | Relevant spec, validators/types, examples и fixtures, [APPENDIX_SCHEMAS](APPENDIX_SCHEMAS.md), ADR rationale. Для `workspace.yaml` — [WORKSPACE_SPEC](spec/WORKSPACE_SPEC.md) и [schema](../schemas/workspace.schema.json). |
| Runtime/pipeline boundary | Соответствующие разделы [PIPELINE_SPEC](spec/PIPELINE_SPEC.md), [ARCHITECTURE](ARCHITECTURE.md), behavior tests; [API_SPEC](spec/API_SPEC.md), если меняется API. |
| CLI/install/user flow | [README](../README.md), [INSTALL](INSTALL.md), [CONTRIBUTING](../CONTRIBUTING.md) или UI design/spec по затронутой поверхности; user-visible changes — [CHANGELOG](../CHANGELOG.md). |
| Testing/toolchain/fixture policy | [TESTING_STRATEGY](TESTING_STRATEGY.md), [CONTRIBUTING](../CONTRIBUTING.md), relevant fixture README/golden. Live policy — [runbook](RELEASE_LIVE_E2E_RUNBOOK.md) и catalog. |
| Implementation milestone | Evidence и active plan; canonical matrix только при подтверждённом изменении статуса. Не дублировать matrix вручную в других документах. |
| Agent workflow | [AGENTS](../AGENTS.md), [AGENT_DEVELOPMENT](AGENT_DEVELOPMENT.md), relevant `.agents/skills/*/SKILL.md`; runtime prompts не входят в обычную правку guidance. |

`README.md` сохраняет ссылку на canonical stakeholder matrix. Версия и дата stakeholder документа
обновляются при его содержательной ревизии. ExecPlan progress log хранит evidence своей задачи,
а не заменяет changelog или полный журнал каждой документационной правки.

## Проверка

Для guidance/links/plans выполнить `make verify-agent-guidance`; semantic docs consistency проверяет
`./scripts/run-go.sh test ./internal/docsync`. При contract diff дополнительно выполнить
`make contracts-check` и relevant package tests. Эти проверки дополняют review смысла: совпадение
слов или существование ссылки сами по себе не доказывают актуальность статуса и поведения.
