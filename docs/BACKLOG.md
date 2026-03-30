# BACKLOG (baseline)

Этот backlog описывает эпики реализации и критерии приёмки MVP.
Каждый эпик дальше декомпозируется на reviewable PR.

## Epic 1 — Управление workspace
Acceptance:
- читается `workspace.yaml`
- валидируются локальные пути
- валидируется структура central `arch-workspace` (`charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/imports/`)
- в UI отображается статус workspace (репозитории подключены, папка docs доступна)

## Epic 2 — TaskResult schema + validator
Acceptance:
- `schemas/taskresult.schema.json` зафиксирована как контракт
- orchestrator валидирует TaskResult до применения изменений
- невалидный TaskResult отклоняется с понятной ошибкой

## Epic 3 — Claude Code adapter (headless)
Acceptance:
- orchestrator запускает Claude Code headless для workspace
- поддерживается передача PromptPack + subagents + skills
- raw TaskResult сохраняется в `reports/taskruns/`

## Epic 4 — Model store (entity-per-file)
Acceptance:
- сущности и связи создаются/обновляются как YAML-файлы
- поддерживаются поля provenance/confidence
- есть минимальный resolver для stable IDs + aliases

## Epic 5 — Init pipeline 0–4 (MVP)
Acceptance:
- Step 0: Charter wizard (template-based) сохраняет артефакты в `charter/`
- Step 1: Collect context формирует модель и coverage
- Step 2: As-is docs формируются в `reports/as-is/`
- Step 3: Findings формируются в `reports/findings/`
- Step 4: Proposals формируются в `proposals/<topic>/`

## Epic 6 — UI baseline
Acceptance:
- экраны: Charter wizard/editor, Skills editor, Run pipeline, Results viewer
- git helper: Commit changes + Create proposal branch (baseline decision)
