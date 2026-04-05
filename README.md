# Architecture Control Plane (Local-first MVP)

> **Статус:** MVP beta foundation / runnable local pipeline baseline + strict contracts
> **Принятый стек реализации:** Go (backend/orchestrator) + React/TypeScript UI (embedded), runtime анализа в MVP: **Claude Code headless**
> **Последняя ревизия:** 2026-04-05

## Что это

Architecture Control Plane (ACP) — **local-first** инструмент, который строит и поддерживает **as-is архитектурную модель** multi-repo системы через agentic runtime.

ACP не является "рисовалкой диаграмм". Архитектура трактуется как **версионируемая модель в Git**, а диаграммы/отчёты/предложения компилируются из неё.

---

## Статус репозитория

Сейчас репозиторий содержит:
- набор документов для стейкхолдеров и инженеров,
- контракты и схемы,
- рабочий local-first backend/API/CLI baseline (`init|refresh` execution path),
- deterministic materialization baseline для `model/`, `reports/`, `proposals/`, `changelog`,
- UI shell + `make` entrypoints + repo CI.

Реализация остаётся incremental по `docs/BACKLOG.md`, но базовый e2e поток уже исполним: `workspace validate -> run pipeline -> inspect artifacts`.

Канонический статус stakeholder-plan (`implemented vs planned`) зафиксирован в [docs/STAKEHOLDER_DOC.md](docs/STAKEHOLDER_DOC.md), секция **Canonical Stakeholder Matrix (source of truth)**.

---

## Scope MVP (явно)

✅ В MVP включено:
- только Claude Code как runtime (headless/bare execution)
- local-first режим (всё запускается локально)
- запуск того же standalone orchestrator в CI/CD через GitHub/GitLab hooks и/или manual pipeline/job trigger
- единый формат хранения: central `arch-workspace` git-репозиторий (Variant 2)
- источники репозиториев: локальные checkout-папки и/или GitHub/GitLab `git_url`, разрешаемые через локальный `git` контекст пользователя/runner
- локально импортированные документы
- интерактивный wizard "Конституции проекта"
- deterministic Step 0 materialization из `charter/wizard/step0-contract.json` (с fallback baseline + warning в run diagnostics при missing/invalid contract)
- встроенный baseline bundle agents/skills/prompts + редактируемые в UI prompt packs, версионируемые в Git
- domain-first иерархия агентов (domain analysts + architect aggregator)
- markdown-карточки доменов/команд как source-of-truth в `charter/cards`
- internal Q&A capability системного аналитика поверх артефактов workspace (`internal/qa` + `acp qa`, без публичного API endpoint в beta surface)
- итерационный changelog в `reports/changelog`
- детальный анализ каждого сервиса: архитектура, внешние интеграции, БД, CI/CD
- анализ arbitrary stacks через Claude Code + baseline prompt bundle, без фиксированного whitelist парсеров в MVP
- явная фиксация недостатка информации через `coverage`, `questions` и findings
- Git-based versioning/branching для модели, правил, отчётов и proposal-пакетов
- строгий контракт TaskResult (JSON Schema) между runtime и orchestrator

❌ В MVP не включено:
- security/compliance enforcement
- hosted/multi-tenant режим
- автоматические интеграции Confluence/Jira/Notion (включая autodocs)
- manager-агенты по Jira/resource skew
- org-scale cost optimization/scheduling
- расширенные role-based UX поверхности

---

## Agent Operating Model (MVP)

`domain-first` модель:
- на каждый домен работает Domain Analyst Agent;
- Team overlay фиксируется отдельными team cards;
- 1 Architect Aggregator Agent собирает и нормализует результаты domain-агентов;
- System Analyst Q&A Agent отвечает на вопросы по артефактам `charter/cards + model + reports + docs/imports`;
- каждая итерация фиксируется в markdown changelog.

Q&A API follow-up в baseline зарезервирован как read-only endpoint `POST /api/qa/ask` (post-beta slice).
Полная матрица статусов epics и boundary зафиксирована в canonical stakeholder matrix: [docs/STAKEHOLDER_DOC.md](docs/STAKEHOLDER_DOC.md).

### Baseline Bundle (MVP)

В продукт поставляется обязательный baseline bundle, который хранится в workspace и редактируется как git-tracked assets:
- agents: `domain-analyst`, `architect-aggregator`, `system-analyst-qa`
- skills: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`
- prompt packs: `constitution`, `collect-context`, `findings`, `proposals`, `qa`

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
- Go 1.20.x
- Node.js 22.21.1
- npm 10.x
- локальный доступ пользователя к git на устройстве, где запускается ACP
- локальные клоны релевантных репозиториев и/или доступные через локальный `git` GitHub/GitLab `git_url`
- установленный Claude Code в PATH

### 1) Создайте architecture workspace

Для MVP это единственная каноническая конвенция хранения: central `arch-workspace` repo.

Рекомендуемый layout отдельного git-репозитория:

```text
arch-workspace/
  workspace.yaml
  charter/
    cards/
      domains/
      teams/
  skills/
    subagents.yaml
  model/
  reports/
    as-is/
    findings/
    coverage/
    taskruns/
    agent-outputs/
      domains/
      architect/
    changelog/
  proposals/
  docs/
    imports/
    rfcs/
    meetings/
    decisions/
```

### 2) Заполните `workspace.yaml`

Source of truth:
- `docs/spec/WORKSPACE_SPEC.md`
- `schemas/workspace.schema.json`
- `examples/workspace.example.yaml`

В manifest описываются только repo sources и optional `docs.imports_path`.
Layout `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` не конфигурируется через `workspace.yaml` и считается fixed MVP convention.

См. [examples/workspace.example.yaml](examples/workspace.example.yaml):

```yaml
version: 1
repos:
  - name: payments-service
    path: /absolute/path/to/payments-service
  - name: users-service
    git_url: https://gitlab.example.com/platform/users-service.git
    ref: main
docs:
  imports_path: ./docs/imports
```

Для каждой записи в `repos[]` задаётся либо `path`, либо `git_url`.
Если указан `git_url`, workspace layer выполняет clone/fetch на той же машине через локальный `git` и текущий git-контекст пользователя/runner.
Отдельное хранилище credentials внутри ACP в MVP не вводится.
Имена репозиториев в `repos[]` должны быть уникальными, потому что используются как repo scopes и evidence references.

### 3) Импортируйте документы вручную (MVP)

Документы (например, выгрузки из Confluence) кладутся в `docs/imports/`.
Для импортов рекомендуется вести `docs/imports/index.yaml` с metadata: источник, путь, checksum, imported_at, source_updated_at.

### 4) Запускайте CLI

Доступные интерфейсы в baseline slice:
- локальный UI/API (через `acp serve --workspace <abs-path>`)
- минимальный CLI
- batch mode для GitHub/GitLab CI/CD
- запуск через SCM hook и/или manual pipeline button/job

`acp serve` в MVP поднимает single-workspace-per-process service.
Поэтому pipeline endpoints работают с уже привязанным workspace и не принимают `workspace_path` в request body.
Для CI/CD required surface — CLI batch mode; internal API trigger остаётся optional и допустим только для trusted local/private deployment.

`acp run` выполняет рабочий baseline pipeline (`init`/`refresh`) и материализует артефакты в workspace.
`acp serve` поднимает локальный API backend и embedded UI (SPA fallback на non-API routes); для быстрой проверки wiring доступен `--dry-run`.
Runtime selector фиксируется на уровне процесса:
- `--runtime fake` (default, deterministic required CI surface)
- `--runtime headless` (opt-in для реальных локальных прогонов через Claude Code headless)

Пример доступных команд:
- `acp serve --workspace /path/to/arch-workspace --runtime fake`
- `acp serve --workspace /path/to/arch-workspace --runtime headless`
- `acp run --workspace ... --pipeline init --runtime fake`
- `acp run --workspace ... --pipeline refresh --runtime fake --non-interactive`
- `acp qa --workspace ... --question "Who owns payments-service?"`

### 5) Поднимите dev environment

Root entrypoints:
- `make bootstrap`
- `make contracts`
- `make test`
- `make lint`
- `make build`
- `make run-backend WORKSPACE=/abs/path/to/arch-workspace`
- `make run-ui`

Repo CI по умолчанию живёт в GitHub Actions:
- `contracts`
- `backend`
- `ui`
- `golden`
- `smoke-cli`
- `smoke-api`
- `ui-smoke`
- `live-runner-smoke` (manual/optional)

---

## High-level архитектура (локально)

- **arch-workspace/**: charter, skills, model, reports, proposals
- **Repo source resolver**: локальные `path` или `git_url`, разрешаемые в локальные checkout через системный `git` текущего пользователя/runner
- **Agent topology**: domain analysts, architect aggregator, system analyst Q&A + baseline bundle skills/prompts
- **UI**: guided workspace setup + baseline editors для `charter/*` и `skills/*`, запуск pipeline, просмотр результатов
- **Orchestrator (Go)**: шаги pipeline, context/prompt packs, вызов runtime, local execution и CI/CD trigger execution
- **Claude Code runner**: headless jobs анализа
- **Model store**: `model/` в формате entity-per-file, включая внешние системы и datastores
- **Reports/Proposals**: `reports/` (включая `agent-outputs/` и `changelog/`) и `proposals/`

### Data flow (MVP)

```mermaid
flowchart LR
  U[User] --> UI[Local UI]
  SCM[GitHub/GitLab hooks or pipeline button] --> ORCH
  UI --> ORCH[Orchestrator (Go)]
  ORCH --> WS[arch-workspace (git)]
  ORCH --> CC[Claude Code (headless)]
  ORCH --> SRC[Repo sources from workspace.yaml]
  SRC --> REPOS[Local checkout paths]
  SRC --> GITLAB[GitHub/GitLab git_url via local git]
  GITLAB --> REPOS
  CC --> DOCS[Local docs/imports]
  CC --> REPOS
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

MVP-модель должна покрывать как минимум:
- сервисы и их интерфейсы,
- внешние интеграции,
- datastores,
- ownership hints,
- CI/CD evidence в reports/coverage/findings, если это не выносится в core model.

Подробнее: `docs/spec/MODEL_SPEC.md`.

---

## Контракт runtime output: TaskResult (обязателен)

Orchestrator принимает выход runtime **только** как TaskResult JSON и валидирует по `schemas/taskresult.schema.json`.

### Top-level поля
- required: `meta`, `summary`, `changeset`
- optional: `questions`, `coverage`, `warnings`, `debug`

MVP canonical runtime shape:
- `questions[]` и `coverage` пишутся на top-level
- legacy operation forms `add_question` / `set_coverage` допускаются только для backward-compatible normalization внутри orchestrator
- `add_doc_artifact` трактуется как metadata registration op, а не как content write op

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

### CI/CD mode (MVP)
- тот же `acp run ... --non-interactive` может выполняться в GitHub/GitLab pipeline job
- запуск инициируется через SCM hooks и/или manual pipeline button/job
- входы: workspace repo + локальные checkout и/или доступ к declared `git_url` через локальный `git` контекст пользователя/runner
- ACP не хранит отдельные git credentials и не требует hosted control plane
- выходы: обновлённые артефакты workspace и явные gaps по недостающей информации
- GitLab template примеры (push + manual trigger): `scripts/templates/gitlab/`

Подробная спецификация: `docs/spec/PIPELINE_SPEC.md`.

---

## UI требования (MVP baseline)

UI в MVP должен покрывать минимум:
- wizard для Step 0 (charter);
- настройку источников репозиториев: локальные папки и GitHub/GitLab URL;
- baseline-wide редактор `charter/*` + `skills/*` (prompt packs, `subagents.yaml`, skill prompts);
- запуск pipeline (init/update);
- просмотр результатов (`as-is`, findings, proposals);
- просмотр coverage/questions по недостающим данным;
- вызовы backend через `/api/*` (см. `docs/spec/API_SPEC.md`).

---

## Стратегия тестирования (baseline)

- source of truth: `docs/TESTING_STRATEGY.md`
- required CI использует synthetic fixtures, recorded runner outputs и не зависит от live Claude Code / live network
- baseline layers:
  - contract tests для `workspace.yaml` и `TaskResult`
  - semantic validator tests
  - golden/regression tests для model/compiler outputs
  - scenario integration tests на synthetic repos
  - smoke tests для CLI/API/UI
- optional `live-runner-smoke` остаётся manual/opt-in и не блокирует merge

---

## Deterministic scope (beta baseline)

При одинаковом input + recorded runner expected stable surface:
- `charter/`
- `model/`
- `reports/as-is/`
- `reports/findings/`
- `reports/coverage/`
- `reports/agent-outputs/`
- `proposals/`

Run-specific поверхность (исключена из strict golden compare):
- `reports/changelog/*`
- `reports/taskruns/*`
- runtime run registry/status (`/api/pipeline/runs/*`)
- runtime parse/runtime ошибки после успешного async start отражаются в `GET /api/pipeline/runs/<run_id>.error_code` (например, `runner_parse_failed`)

Статус покрытия epics (single source): `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.

---

## Ключевые файлы

- `go.mod` — root Go module
- `Makefile` — единые developer entrypoints
- `.github/workflows/*` — repo CI
- `docs/STAKEHOLDER_DOC.md` — stakeholder source-of-truth и canonical matrix статусов (v1.0 implementation-aligned)
- `docs/spec/WORKSPACE_SPEC.md` — канонический контракт `workspace.yaml`
- `docs/spec/MODEL_SPEC.md` — каноническая модель v0
- `docs/spec/PIPELINE_SPEC.md` — pipeline I/O и expected artifacts
- `docs/TESTING_STRATEGY.md` — baseline strategy для contract/golden/smoke tests
- `docs/APPENDIX_SCHEMAS.md` — человеко-читаемые правила для schema/contracts
- `schemas/taskresult.schema.json` — JSON Schema контракта runtime output
- `schemas/workspace.schema.json` — JSON Schema для `workspace.yaml`
- `examples/workspace.example.yaml` — пример workspace config
- `examples/taskresult.example.json` — пример TaskResult
- `cmd/acp/main.go` — CLI entrypoint (`serve`, `run`, `qa`)
- `ui/package.json` — UI toolchain + scripts
- `fixtures/README.md` — baseline fixtures и regression surface
- `docs/BACKLOG.md` — эпики и acceptance criteria
- `docs/BASELINE_POLICY.md` — правила сопровождения baseline

---

## Порядок реализации

1) финализировать baseline model + TaskResult contract
2) реализовать baseline bundle agents/skills/prompts
3) реализовать CI/CD trigger surface: hooks/manual pipeline button/job + batch mode
4) реализовать orchestrator + Claude Code adapter
5) реализовать model store (entity-per-file) и extraction coverage for integrations/datastores/CI-CD
6) реализовать UI (workspace setup, charter/skills/run/results)
