# Contributing

ACP — **AI-native** и **spec-first** проект.

## Start here
1) Прочитайте `AGENTS.md`
2) Прочитайте `docs/ARCHITECTURE.md`
3) Относитесь к `schemas/taskresult.schema.json` как к контракту

## Local bootstrap
- Требуемый стек: Go 1.20.x, Node 22.21.1, npm 10.x, Git
- Установите зависимости:
  - `make bootstrap`
- Прогоните обязательные проверки:
  - `make contracts`
  - `make test`
  - `make lint`
  - `make build`

## Repo CI
- GitHub Actions является canonical repo CI для самого ACP репозитория
- Required jobs:
  - `contracts`
  - `backend`
  - `ui`
- Эти jobs не используют live Claude Code, live GitHub/GitLab API и реальные пользовательские репозитории

## ADRs
Пишите ADR, если:
- выбираете/добавляете крупную зависимость
- меняете схемы/контракты
- меняете семантику пайплайна или правила модели

См. `docs/adr/`.

## Planning
Используйте `docs/PLANS.md` для многосоставных задач.

## Branching
- feat/<slug>, fix/<slug>, chore/<slug>, proposal/<slug>
