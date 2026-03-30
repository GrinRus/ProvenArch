# AGENTS.md (Codex)

Codex читает `AGENTS.md` перед началом работы. Держите этот файл коротким и устойчивым.

## Mission (MVP)
Собрать ACP как **local-first** инструмент:
- runtime анализа: **Claude Code headless** (только в MVP)
- стек реализации: **Go backend/orchestrator + React UI**
- outputs: Git‑версионируемые файлы workspace (entity-per-file модель)

## Жёсткие ограничения
- Нет hosted режима в MVP
- Нет security/compliance enforcement в MVP (будущие волны)
- Не менять схемы/контракты без синхронизации: docs + валидаторы + тесты
- Предпочитать Git-friendly diffs и детерминированность

## Всегда делать
- Читать: README.md, docs/ARCHITECTURE.md, schemas/taskresult.schema.json
- Для нетривиальной задачи: ExecPlan в docs/PLANS.md
- Добавлять/обновлять тесты/фикстуры при изменении core поведения
- Обновлять документацию при изменении поведения
- Использовать skills (`.agents/skills/*`) когда применимо

## Не делать
- Не добавлять другие runtimes в MVP (только Claude Code)
- Не “выдумывать” форматы данных/модели
- Не писать в пользовательские репозитории; писать только в workspace
