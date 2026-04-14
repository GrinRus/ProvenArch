# AGENTS.md (Codex)

Codex читает `AGENTS.md` перед началом работы. Держите этот файл коротким и устойчивым.

## Mission (MVP)
Собрать ACP как **local-first** инструмент:
- runtime анализа: **headless multi-provider** (`claude-code` default, `qwen-code` optional) + deterministic `fake` baseline
- стек реализации: **Go backend/orchestrator + React UI**
- outputs: Git‑версионируемые файлы workspace (entity-per-file модель)

## Жёсткие ограничения
- Нет hosted режима в MVP
- Нет security/compliance enforcement в MVP (будущие волны)
- Не менять схемы/контракты без синхронизации: docs + валидаторы + тесты
- Предпочитать Git-friendly diffs и детерминированность

## Всегда делать
- Читать: README.md, docs/ARCHITECTURE.md, schemas/taskresult.schema.json
- Для pre-release live gate использовать `docs/RELEASE_LIVE_E2E_RUNBOOK.md` как source of truth
- Для нетривиальной задачи: ExecPlan в docs/PLANS.md
- Добавлять/обновлять тесты/фикстуры при изменении core поведения
- Обновлять документацию при изменении поведения
- При конфликте источников правды использовать приоритет:
  `schemas/*` -> `docs/spec/*` -> `README.md`/`docs/ARCHITECTURE.md` -> `docs/STAKEHOLDER_DOC.md`
- При изменении `workspace.yaml` contract синхронизировать `docs/spec/WORKSPACE_SPEC.md`, `schemas/workspace.schema.json`, examples и fixtures
- При изменении testing baseline синхронизировать `docs/TESTING_STRATEGY.md`, fixtures и golden outputs
- Для каждого завершённого slice выполнять DoD:
  `make contracts`, `make test`, `make lint`, `make build`
- Для live matrix harness запускать только `scripts/full-run-batch-matrix.sh` (без wrapper-скрипта)
- Для release matrix использовать 4 профиля: `single-path`, `single-git_url`, `multi-path`, `multi-git_url`; sweep baseline: `baseline`, `scale-backend`
- Если `sweeps[]` отсутствует в `E2E_MATRIX_FILE`, считать implicit `baseline`
- Release verdict брать только из `reports/release_verdict_<matrix-id>.json` (`PASS` -> ready, иначе blocked)
- В release gate требовать strict zero-failure и оба provider в PATH: `qwen`, `claude`
- При изменении `schemas/*` или `docs/spec/*` обязательно синхронизировать:
  `docs/APPENDIX_SCHEMAS.md`, `examples/*`, `fixtures/*`, валидаторы/тесты и ADR rationale
- Использовать skills (`.agents/skills/*`) когда применимо

## AI workflow
- Брать небольшой reviewable slice из `docs/BACKLOG.md` и `docs/spec/*`
- Держать diff маленьким и не расшатывать контракты без явной необходимости
- При изменении core поведения синхронизировать docs/spec и тесты/фикстуры
- Required CI проектировать без live network dependencies; live runner checks оставлять optional
- Full live matrix harness остаётся manual trusted-machine pre-release gate, а не required CI merge gate

## Не делать
- Не расширять список headless providers в MVP beyond `claude-code` и `qwen-code` без отдельного slice
- Не “выдумывать” форматы данных/модели
- Не писать в пользовательские репозитории; писать только в workspace
- Не добавлять wrapper-скрипт поверх matrix harness для release gate
