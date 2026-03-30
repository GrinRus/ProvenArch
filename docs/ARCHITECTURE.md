# ARCHITECTURE.md (Go monorepo MVP)

Этот документ описывает **целевую** (planned) архитектуру реализации ACP для local-first MVP.

## Scope (MVP)
- Local-first: всё работает на машине разработчика
- Runtime (analysis): **Claude Code headless** (только в MVP)
- Реализация продукта: **Go backend/orchestrator + embedded React UI**
- Единая workspace-конвенция MVP: central `arch-workspace` git-репозиторий (Variant 2)
- Нет hosted режима и нет security/compliance enforcement в MVP

## Компоненты
1) **Go Server (`cmd/acp`)** *(planned)*
   - Раздаёт UI (embedded static assets из `ui/dist`)
   - Экспортирует API под `/api/*`

2) **UI (`ui/`)** *(planned)*
   - React + TypeScript + Vite
   - Dev: `npm run dev` с proxy на backend
   - Prod: `npm run build` → `ui/dist` встраивается в Go бинарь

3) **Orchestrator (`internal/orchestrator`)** *(planned)*
   - Step registry (шаги init pipeline)
   - Готовит ContextPack/PromptPack
   - Работает с единым central workspace (`arch-workspace`) как корнем артефактов MVP
   - Вызывает runtime adapter
   - Валидирует TaskResult (schema)
   - Применяет changeset к модели workspace
   - Триггерит генерацию отчётов
   - (опционально) делает git commit

4) **Claude Code adapter (`internal/runtime/claudecode`)** *(planned)*
   - headless запуск процесса
   - возвращает TaskResult JSON

5) **Workspace (`internal/workspace`)** *(planned)*
   - реализует/валидирует структуру central `arch-workspace` (Variant 2)
   - парсит `workspace.yaml`
   - safe path joins (никогда не читаем вне workspace root)
   - git helpers (shell out в `git`)

6) **Model store (`internal/model`)** *(planned)*
   - entity-per-file YAML
   - stable IDs + aliases
   - apply changeset operations

7) **Reports (`internal/reports`)** *(planned)*
   - генерирует `reports/as-is/*` и findings/proposals индексы

## Pipeline (MVP)
0) Конституция (charter)
1) Collect context
2) As-is docs
3) Findings (gaps/anti-patterns)
4) Proposals (improvements)
