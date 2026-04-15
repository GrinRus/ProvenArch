# Contributing

ACP — **AI-native** и **spec-first** проект.

## Start here
1) Прочитайте `AGENTS.md`
2) Прочитайте `README.md` и `docs/ARCHITECTURE.md`
3) Относитесь к `schemas/taskresult.schema.json` и `docs/spec/*` как к контрактам

## Local bootstrap
- Требуемый стек: Go 1.20.x, Node 22.21.1, npm 10.x, Git
- Установите зависимости: `make bootstrap`
- Прогоните DoD-проверки:
  - `make contracts`
  - `make test`
  - `make lint`
  - `make build`

## ADRs
Пишите ADR, если:
- выбираете/добавляете крупную зависимость
- меняете схемы/контракты
- меняете семантику пайплайна или правила модели

См. `docs/adr/`.

## Canonical references
- Repo CI и required jobs: `README.md` + `docs/TESTING_STRATEGY.md`
- Планирование крупных slice: `docs/PLANS.md`

## Planning
Используйте `docs/PLANS.md` для многосоставных задач.
