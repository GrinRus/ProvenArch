# Architecture Control Plane (Local-first MVP)

> **Статус:** Draft / strict baseline (документы/контракты/структура без реализации кода)
> **Принятый стек реализации:** Go (backend/orchestrator) + React/TypeScript UI (embedded), runtime анализа в MVP: **Claude Code headless**

## Что это

Architecture Control Plane (ACP) — **local-first** инструмент, который строит и поддерживает **as-is архитектурную модель** multi-repo системы через agentic runtime.

ACP не является "рисовалкой диаграмм". Архитектура трактуется как **версионируемая модель в Git**, а диаграммы/отчёты/предложения компилируются из неё.

---

## Статус репозитория

Сейчас репозиторий содержит **строгий baseline**:
- набор документов для стейкхолдеров и инженеров,
- контракты и схемы,
- структуру каталогов и соглашения.

Реализованного runnable кода продукта пока нет. Реализация запускается отдельными PR из `docs/BACKLOG.md`.

---

## Scope MVP (явно)

✅ В MVP включено:
- только Claude Code как runtime (headless/bare execution)
- local-first режим (всё запускается локально)
- единый формат хранения: central `arch-workspace` git-репозиторий (Variant 2)
- локальные репозитории и локально импортированные документы
- интерактивный wizard "Конституции проекта"
- subagents + skills (prompt packs), редактируемые в UI и версионируемые в Git
- Git-based versioning/branching для модели, правил, отчётов и proposal-пакетов
- строгий контракт TaskResult (JSON Schema) между runtime и orchestrator

❌ В MVP не включено:
- security/compliance enforcement
- hosted/multi-tenant режим
- автоматические интеграции Confluence/Jira/Notion
- org-scale cost optimization/scheduling
- расширенные role-based UX поверхности

---

## Ключевые понятия (trust model)

ACP разделяет три типа фактов:
- **Observation**: факт с evidence из артефактов.
- **Inference**: гипотеза на основе косвенных сигналов.
- **Assertion**: факт, подтверждённый человеком/организацией.

MVP policy: Observation + Assertion отображаются как рабочая истина, Inference требует review.

---

## Быстрый старт (baseline)

### Prerequisites
- Git
- локальные клоны релевантных репозиториев
- установленный Claude Code в PATH

### 1) Создайте architecture workspace

Для MVP это единственная каноническая конвенция хранения: central `arch-workspace` repo.

Рекомендуемый layout отдельного git-репозитория:

```text
arch-workspace/
  workspace.yaml
  charter/
  skills/
  model/
  reports/
  proposals/
  docs/
    imports/
    rfcs/
    meetings/
    decisions/
```

### 2) Заполните `workspace.yaml`

См. [examples/workspace.example.yaml](examples/workspace.example.yaml):

```yaml
version: 1
repos:
  - name: payments-service
    path: /absolute/path/to/payments-service
  - name: infra-k8s
    path: /absolute/path/to/infra-k8s
docs:
  imports_path: ./docs/imports
```

### 3) Импортируйте документы вручную (MVP)

Документы (например, выгрузки из Confluence) кладутся в `docs/imports/`.

### 4) Запускайте pipeline (planned)

Планируемые интерфейсы:
- локальный UI (через Go server)
- минимальный CLI

Пример целевых команд:
- `acp serve --workspace /path/to/arch-workspace`
- `acp run --workspace ... --pipeline init`
- `acp run --workspace ... --pipeline update`

---

## High-level архитектура (локально)

- **arch-workspace/**: charter, skills, model, reports, proposals
- **UI**: редактирование charter/skills, запуск pipeline, просмотр результатов
- **Orchestrator (Go)**: шаги pipeline, context/prompt packs, вызов runtime
- **Claude Code runner**: headless jobs анализа
- **Model store**: `model/` в формате entity-per-file
- **Reports/Proposals**: `reports/` и `proposals/`

### Data flow (MVP)

```mermaid
flowchart LR
  U[User] --> UI[Local UI]
  UI --> ORCH[Orchestrator (Go)]
  ORCH --> WS[arch-workspace (git)]
  ORCH --> CC[Claude Code (headless)]
  CC --> REPOS[Local repos paths]
  CC --> DOCS[Local docs/imports]
  CC --> OUT[TaskResult JSON (changeset + evidence)]
  OUT --> ORCH
  ORCH --> WS
  UI --> WS
```

---

## Каноническая модель (MVP)

Модель хранится в **entity-per-file YAML**:

```text
model/
  entities/
  edges/
```

Минимальные требования к сущностям/связям:
- `id`
- `type`
- `name` (для entity)
- `provenance.kind`: `observation | inference | assertion`
- `provenance.confidence`: `0..1`
- `provenance.evidence[]`

Подробнее: `docs/spec/MODEL_SPEC.md`.

---

## Контракт runtime output: TaskResult (обязателен)

Orchestrator принимает выход runtime **только** как TaskResult JSON и валидирует по `schemas/taskresult.schema.json`.

### Top-level поля
- required: `meta`, `summary`, `changeset`
- optional: `questions`, `coverage`, `warnings`, `debug`

### Changeset operations (MVP)
- `upsert_entity`
- `remove_entity`
- `upsert_edge`
- `remove_edge`
- `add_finding`
- `add_doc_artifact`
- `add_question`
- `set_coverage`

### Evidence format (MVP)

Каждый evidence item ссылается на локальный артефакт:

```json
{
  "repo": "payments-service",
  "ref": "main@<commit>",
  "path": "internal/http/routes.go",
  "lines": { "start": 120, "end": 148 },
  "excerpt_hash": "sha256:..."
}
```

Пример: `examples/taskresult.example.json`.

---

## Пайплайны (MVP)

### Init pipeline
0. Charter (wizard)
1. Collect context
2. As-is docs
3. Findings
4. Proposals

### Continuous loop (manual)
- обновление локальных репозиториев/документов
- повторный запуск pipeline
- обновление model/reports/proposals

Подробная спецификация: `docs/spec/PIPELINE_SPEC.md`.

---

## UI требования (MVP baseline)

UI в MVP должен покрывать минимум:
- wizard для Step 0 (charter);
- редактор skills/prompts;
- запуск pipeline (init/update);
- просмотр результатов (`as-is`, findings, proposals);
- вызовы backend через `/api/*` (см. `docs/spec/API_SPEC.md`).

---

## Стратегия тестирования (baseline)

- fixture repos (синтетические)
- golden outputs (TaskResult + model files)
- проверки в CI:
  - schema validation TaskResult
  - regressions по stable IDs
  - базовые lint/check правила для model artifacts

---

## Ключевые файлы

- `docs/STAKEHOLDER_DOC.md` — stakeholder vision и MVP рамка (v0.5)
- `docs/spec/MODEL_SPEC.md` — каноническая модель v0
- `docs/spec/PIPELINE_SPEC.md` — pipeline I/O и expected artifacts
- `docs/APPENDIX_SCHEMAS.md` — человеко-читаемые правила для schema/contracts
- `schemas/taskresult.schema.json` — JSON Schema контракта runtime output
- `examples/workspace.example.yaml` — пример workspace config
- `examples/taskresult.example.json` — пример TaskResult
- `docs/BACKLOG.md` — эпики и acceptance criteria
- `docs/BASELINE_POLICY.md` — правила сопровождения baseline

---

## Порядок реализации

1) финализировать schema/examples для TaskResult
2) реализовать orchestrator + Claude Code adapter
3) реализовать model store (entity-per-file)
4) реализовать UI (charter/skills/run/results)
