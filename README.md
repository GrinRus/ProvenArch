# Architecture Control Plane (Local-first MVP)

> **Статус:** MVP beta foundation / runnable local pipeline baseline + strict contracts
> **Принятый стек реализации:** Go (backend/orchestrator) + React/TypeScript UI (embedded), runtime анализа в MVP: **headless multi-provider** (`claude-code` default, `qwen-code` optional)
> **Последняя ревизия:** 2026-04-13

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
- process-scoped runtime provider selection для headless режима: `claude-code` (default) и `qwen-code`
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
- анализ arbitrary stacks через выбранный headless provider (`claude-code|qwen-code`) + baseline prompt bundle, без фиксированного whitelist парсеров в MVP
- явная фиксация недостатка информации через `coverage`, `questions` и findings
- semantic guard в refresh-цикле: фильтрация нерелевантных placeholder-операций, fallback finding при owner-gap, канонизация/дедуп coverage+questions
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

Bundle bootstrap policy:
- `init-workspace` и `serve --auto-init` создают baseline artifacts по стратегии create-if-missing;
- существующие пользовательские правки в baseline файлах не перезаписываются.

---

## Ключевые понятия (trust model)

ACP разделяет три типа фактов:
- **Observation**: факт с evidence из артефактов.
- **Inference**: гипотеза на основе косвенных сигналов.
- **Assertion**: факт, подтверждённый человеком/организацией.

MVP policy: Observation + Assertion отображаются как рабочая истина, Inference требует review.

---

## Быстрый старт (minimal local MVP)

### Минимальные prerequisites для первого запуска
- Git
- Go 1.20.x
- локальный клон хотя бы одного анализируемого репозитория
- Node.js 22.21.1 + npm 10.x (нужно для UI dev/build в этом репозитории)

Для первого запуска достаточно `--runtime fake`.
Для реальных запусков `--runtime headless` нужен установленный provider command (`claude-code`/`qwen`) либо env override (`ACP_CLAUDE_CMD`/`ACP_QWEN_CMD`).
Direct режим `ACP_CLAUDE_CMD=claude` поддерживается нативно (без wrapper).

### 1) Поднимите сервис одной командой (auto-init)

```bash
acp serve --workspace /path/to/arch-workspace --auto-init --repo-name payments-service --repo-path /path/to/payments-service --runtime fake
```

Эта команда:
- создаёт `workspace.yaml`, если он отсутствует,
- создаёт fixed MVP layout,
- автоматически выполняет `git init` в workspace root, если `.git` отсутствует,
- поднимает backend + embedded UI без блокирующего preflight repo-resolution на старте.

Для multi-repo bootstrap вместо single-repo флагов можно использовать:

```bash
acp serve --workspace /path/to/arch-workspace --auto-init --repos-file /path/to/repos.yaml --runtime fake
```

Опционально для `serve --auto-init` можно задать `--docs-imports-path <path>` (default `./docs/imports` в `workspace.yaml`).

### 2) Запустите первый анализ

Можно из UI (кнопка `Run init`) или CLI:

```bash
acp run --workspace /path/to/arch-workspace --pipeline init --runtime fake --non-interactive
```

После запуска UI отображает dashboard со всеми run'ами (`queued/running/succeeded/failed`), включая уже завершённые запуски.
Для operability UI автоматически выбирает newest active run (или первый из списка), показывает полный warnings list выбранного run, позволяет переключать log view `line/line+fields` и поддерживает `Cancel selected run`.
История сохраняется в workspace: `reports/taskruns/run-history.json`.

### 3) Альтернативный явный bootstrap через `init-workspace`

```bash
acp init-workspace --workspace /path/to/arch-workspace --repo-name payments-service --repo-path /path/to/payments-service
```

Команда:
- создаёт/обновляет `workspace.yaml`,
- создаёт fixed MVP layout (`charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/`),
- автоматически выполняет `git init` в workspace root, если `.git` отсутствует,
- валидирует manifest и repo source.

Source of truth для manifest contract:
- `docs/spec/WORKSPACE_SPEC.md`
- `schemas/workspace.schema.json`
- `examples/workspace.example.yaml`

Для remote source можно использовать:

```bash
acp init-workspace --workspace /path/to/arch-workspace --repo-name users-service --repo-git-url https://gitlab.example.com/platform/users-service.git --repo-ref main
```

Для 2+ репозиториев:

```bash
acp init-workspace --workspace /path/to/arch-workspace --repos-file /path/to/repos.yaml
```

`repos.yaml` поддерживает формат:
- `repos: [...]`
- или top-level массив записей `repos[]`

Если `repos-file` содержит блок `runtime.profile.timeouts`, `init-workspace`/`serve --auto-init` переносят его в `workspace.yaml` (persisted timeout profile).

### 3.1) Read-only QA по артефактам workspace (опционально)

```bash
acp qa --workspace /path/to/arch-workspace --question "Who owns payments-service?"
```

### 4) Самый короткий локальный flow через Makefile

```bash
make quickstart-local WORKSPACE=/path/to/arch-workspace REPO_PATH=/path/to/payments-service REPO_NAME=payments-service
```

Команда выполняет `init-workspace` + первый `init` pipeline. После неё можно сразу запускать `acp serve`.

### 5) Импортируйте документы вручную (MVP)

Документы (например, выгрузки из Confluence) кладутся в `docs/imports/`.
Для импортов рекомендуется вести `docs/imports/index.yaml` с metadata: источник, путь, checksum, imported_at, source_updated_at.

### 6) Когда переходить на headless runtime

- `--runtime fake` — default для required deterministic CI surface и первого локального старта.
- `--runtime headless` — opt-in для реального анализа через выбранный provider.
- `--runtime-provider` поддерживает `claude-code` (default) и `qwen-code`.
- precedence выбора provider: CLI `--runtime-provider` > `ACP_RUNTIME_PROVIDER` > `claude-code`.
- command env:
  - `ACP_CLAUDE_CMD` (default `claude-code`)
  - `ACP_QWEN_CMD` (default `qwen`)
- direct `claude` режим: `ACP_CLAUDE_CMD=claude` (native one-shot invocation с envelope parse).
- в `--runtime fake` provider проходит валидацию, но фактически не используется runner’ом.

### 6.1) Runtime timeouts (persisted + effective)

Timeout-конфиг хранится в `workspace.yaml` (`runtime.profile.timeouts`) и используется backend/full-run/frontend e2e.

Balanced defaults:
- `step_timeout_sec=1800`
- `heartbeat_sec=30`
- `pipeline_timeout_sec=2400`
- `pipeline_kill_grace_sec=30`
- `api_ready_timeout_sec=60`
- `api_init_timeout_sec=120`
- `ui_init_poll_timeout_sec=900`
- `ui_cancel_poll_timeout_sec=420`

Precedence:
- `env > workspace.yaml > defaults`

Каноничные env overrides:
- `ACP_RUNTIME_STEP_TIMEOUT_SEC`
- `ACP_RUNTIME_HEARTBEAT_SEC`
- `ACP_PIPELINE_TIMEOUT_SEC`
- `ACP_PIPELINE_KILL_GRACE_SEC`
- `ACP_API_READY_TIMEOUT_SEC`
- `ACP_API_INIT_TIMEOUT_SEC`
- `ACP_UI_INIT_POLL_TIMEOUT_SEC`
- `ACP_UI_CANCEL_POLL_TIMEOUT_SEC`

Deprecated fallback aliases:
- `READY_TIMEOUT_SEC`
- `UI_E2E_INIT_TIMEOUT_SEC`
- `UI_E2E_CANCEL_TIMEOUT_SEC`
- full-run script aliases: `ACP_FULL_RUN_PIPELINE_TIMEOUT_SEC`, `ACP_FULL_RUN_PIPELINE_KILL_GRACE_SEC`

API управления timeout-профилем:
- `GET /api/runtime/timeouts` (persisted + effective + source)
- `PUT /api/runtime/timeouts` (partial update persisted values)

### 6.2) Runtime execution profile (persisted + effective)

Execution-конфиг хранится в `workspace.yaml` (`runtime.profile.execution`) и управляет шардированием runtime-задач.

Default values:
- `strategy=sequential`
- `max_parallel_tasks=1`
- `failure_policy=best_effort`
- `shard_discovery.mode=heuristics`
- `repo_selection=all`

Precedence:
- `CLI > env > workspace.yaml > defaults`

CLI overrides (ограниченный набор):
- `--execution-strategy sequential|parallel`
- `--max-parallel-tasks <n>`
- `--failure-policy fail_fast|best_effort`

Env overrides:
- `ACP_EXECUTION_STRATEGY`
- `ACP_MAX_PARALLEL_TASKS`
- `ACP_FAILURE_POLICY`
- `ACP_SHARD_DISCOVERY_MODE`
- `ACP_REPO_SELECTION`

API управления execution-профилем:
- `GET /api/runtime/execution` (persisted + effective + source)
- `PUT /api/runtime/execution` (partial update persisted values)

`repo_selection` policy:
- `all`: анализируются все repos из `workspace.yaml`.
- `backend_only`: исключаются только repos с `analysis.role=frontend`; `backend|mixed|unknown` остаются включёнными (для `unknown` validator пишет warning).

### 7) Поднимите dev environment

Root entrypoints:
- `make bootstrap`
- `make contracts`
- `make test`
- `make lint`
- `make build`
- `make run-backend WORKSPACE=/abs/path/to/arch-workspace`
- `make run-ui`

### 8) Полный локальный прогон (scenario)

Готовый runbook и script:
- [docs/LOCAL_FULL_RUN_AI_ADVENT.md](docs/LOCAL_FULL_RUN_AI_ADVENT.md)
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch-5x2.sh` (batch `5x2` + frontend live e2e + quality report aggregation)
- `scripts/full-run-batch-matrix.sh` (multi-profile matrix orchestrator over `full-run-batch-5x2.sh`)
- `scripts/frontend-live-e2e.sh` (локальный live UI smoke для выбранного provider)

Быстрый запуск:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml ./scripts/full-run-ai-advent.sh
```

Канонический input для full-run/batch: `TARGET_REPOS_FILE` (`repos[]` в формате `workspace.yaml`).
Legacy compatibility сохранена:
- `TARGET_REPO=/path/to/repo` (single-path)
- `TARGET_REPO_GIT_URL + TARGET_REPO_NAME + TARGET_REPO_REF` (single-git_url)

Script делает strict полный цикл:
- API simulation + runtime `fake + headless`;
- anti-mock/anti-zero-signal проверки для headless run;
- quality regression guard по одинаковой паре `(runtime_mode, pipeline)` между итерациями;
- локальные semantic checks full-run: owner-gap+findings, canonical duplicates в coverage/questions, critical off-topic markers;
- completion invariants перед `result=passed`: ожидаемое число runtime/headless run, наличие headless `init+refresh` на каждую итерацию, отсутствие `running` в `run-history.json`;
- trap-handling для `TERM/INT/HUP/PIPE` с `failure_reason=infra_signal_terminated` и truthful summary semantics;
- per-run snapshots в `TMP_ROOT/snapshots/<run_id>/...`;
- гарантированные debug artifacts: `TMP_ROOT/full-run.log` и `TMP_ROOT/session-summary.md` даже при раннем fail.
- при `runner_parse_failed` runtime сохраняет raw-output evidence в `reports/taskruns/raw/*` (stdout/stderr + checksums + meta).

Если нужно сохранить временный workspace для ручного анализа:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml KEEP_TMP=1 ./scripts/full-run-ai-advent.sh
```

Опциональные параметры retention для run logs:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml RUN_LOGS_TTL_HOURS=168 RUN_LOGS_MAX_RUNS=200 ./scripts/full-run-ai-advent.sh
```

Batch re-audit `5x2` (direct-only `claude`/`qwen`, без wrapper):

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
./scripts/full-run-batch-5x2.sh
```

Параллельный shard-run (например, по провайдерам):

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
BATCH_ID=batch-qwen \
BATCH_PROVIDER_FILTER=qwen-code \
./scripts/full-run-batch-5x2.sh &

TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
BATCH_ID=batch-claude \
BATCH_PROVIDER_FILTER=claude-code \
./scripts/full-run-batch-5x2.sh &

wait
```

Shard controls:
- `BATCH_PROVIDER_FILTER`: `all` (default) или CSV из `qwen-code,claude-code`
- `BATCH_RUN_SELECTION`: `all` (default), CSV (`1,3,5`) или диапазоны (`1-3,5`)
- `BATCH_SKIP_PRECHECK`: `0|1` (default `0`); полезно для secondary shard'ов
- `BATCH_FRONTEND_MODE`: `auto|always|never` (default `auto`)
  - `auto`: frontend smoke выполняется только если в `BATCH_RUN_SELECTION` есть `run1`
  - `always`: всегда запускать frontend smoke (требует `run1` workspace)
  - `never`: полностью пропускать frontend smoke
- в shard-режиме требуются бинари только выбранных провайдеров из `BATCH_PROVIDER_FILTER`
- для параллельных shard-процессов используйте разные `BATCH_ID` (иначе конфликт output paths)
- рекомендуемый split: один shard с `BATCH_SKIP_PRECHECK=0`, остальные shard'ы с `BATCH_SKIP_PRECHECK=1`

Matrix runbook `4` профиля (`single-path`, `single-git_url`, `multi-path`, `multi-git_url`):

```bash
E2E_MATRIX_FILE=/abs/path/to/e2e-matrix.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
./scripts/full-run-batch-matrix.sh
```

`E2E_MATRIX_FILE` поддерживает `profiles[]`:
- `id`
- `repos_file`
- `expected_repo_count`
- `source_kind` (`path|git_url`)
- обязательный набор профилей: `single-path`, `single-git_url`, `multi-path`, `multi-git_url`
- для `git_url` профилей refs должны быть pinned в `repos_file`
- относительные пути `repos_file` резолвятся относительно директории `E2E_MATRIX_FILE`

Готовый шаблон: `examples/e2e-matrix.example.yaml` (+ `examples/repos/*.repos.yaml`).

Скрипт сохраняет:
- run artifacts (default): `/tmp/provenarch-test_arch_project/runs/<batch-id>/<provider>/runN/...`
- quality reports:
  - `/tmp/provenarch-test_arch_project/reports/run_matrix_<batch-id>.md`
  - `/tmp/provenarch-test_arch_project/reports/frontend_e2e_matrix_<batch-id>.md`
  - `/tmp/provenarch-test_arch_project/reports/quality_report_<batch-id>.md`
- backend quality считается только по snapshot-артефактам (`snapshots/<run_id>/reports/*`), frontend smoke запускается на отдельной `frontend-workspace` копии и не мутирует backend baseline
- batch evaluator добавляет semantic hard-fail checks: `analysis:off-topic`, `analysis:evidence-scope`, `analysis:cross-doc`; для multi-profile (`expected_repo_count >= 2`) обязателен `cross-repo` сигнал (`analysis:cross-repo-missing` при отсутствии)
- в `run_matrix`/`quality_report` дополнительно фиксируются `artifact_source`, `semantic_hard_fail`, `off_topic_hits` и failure classes (`runtime_parse`, `runner_unavailable`, `runtime_timeout`, `infra_signal_terminated`, `infra_incomplete_cycle`, `quality_gates_failed`, `summary_missing`)
- direct `npm run --prefix ui e2e:live`: Playwright output default `/tmp/provenarch-ui-e2e/test-results` (override: `UI_E2E_OUTPUT_DIR`)
- `scripts/frontend-live-e2e.sh`: Playwright output сохраняется в `$OUTPUT_DIR/playwright-results`
- frontend live e2e ожидает число resolved repos из `UI_E2E_EXPECTED_REPO_COUNT` (default `1`)
- `UI_E2E_SCENARIO` переключает live flow:
  - `init-inspect` (default): validate -> run init -> inspect artifacts
  - `cancel-refresh`: validate -> run refresh -> cancel selected run -> expect `failed + run_canceled`
- `UI_E2E_CANCEL_STUB_SLEEP_SEC` задаёт длительность controlled slow stub runner для сценария `cancel-refresh`
- при shard-режиме `BATCH_FRONTEND_MODE=auto` frontend smoke помечается `skipped`, если `run1` не входит в `BATCH_RUN_SELECTION`

`TARGET_REPOS_FILE` — основной batch-контракт; single legacy env поддерживаются для обратной совместимости.
Full matrix (`full-run-batch-matrix.sh`) — локальный trusted-machine runbook, не required CI gate.

Repo CI по умолчанию живёт в GitHub Actions:
- `contracts`
- `backend`
- `ui`
- `golden`
- `smoke-cli`
- `smoke-api`
- `ui-smoke`
- `ui-live-smoke-optional` (workflow_dispatch, не required gate)

---

## High-level архитектура (локально)

- **arch-workspace/**: charter, skills, model, reports, proposals
- **Repo source resolver**: локальные `path` или `git_url`, разрешаемые в локальные checkout через системный `git` текущего пользователя/runner
- **Agent topology**: domain analysts, architect aggregator, system analyst Q&A + baseline bundle skills/prompts
- **UI**: guided workspace setup + baseline editors для `charter/*` и `skills/*`, запуск pipeline, просмотр результатов
- **Orchestrator (Go)**: шаги pipeline, context/prompt packs, вызов runtime, local execution и CI/CD trigger execution
- **Runtime providers**: headless jobs анализа через `claude-code|qwen-code`
- **Model store**: `model/` в формате entity-per-file, включая внешние системы и datastores
- **Reports/Proposals**: `reports/` (включая `agent-outputs/` и `changelog/`) и `proposals/`

### Data flow (MVP)

```mermaid
flowchart LR
  U[User] --> UI[Local UI]
  SCM[GitHub/GitLab hooks or pipeline button] --> ORCH
  UI --> ORCH[Orchestrator (Go)]
  ORCH --> WS[arch-workspace (git)]
  ORCH --> CC[Headless Runtime Provider (claude-code or qwen-code)]
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
- настройку `repos[]` (multi-repo) для локальных папок и GitHub/GitLab URL;
- baseline-wide редактор `charter/*` + `skills/*` (prompt packs, `subagents.yaml`, skill prompts);
- запуск pipeline (init/update);
- явные секции `Setup / Baseline / Runs / Results`;
- просмотр результатов (`as-is`, findings, proposals) и repo validation overview (`resolved_repos` + diagnostics по repo);
- просмотр coverage/questions по недостающим данным;
- вызовы backend через `/api/*` (см. `docs/spec/API_SPEC.md`).

---

## Стратегия тестирования (baseline)

- source of truth: `docs/TESTING_STRATEGY.md`
- required CI использует synthetic fixtures, recorded runner outputs и не зависит от live headless providers / live network
- baseline layers:
  - contract tests для `workspace.yaml` и `TaskResult`
  - semantic validator tests
  - golden/regression tests для model/compiler outputs
  - scenario integration tests на synthetic repos
  - smoke tests для CLI/API/UI
- local live-runner smoke выполняется вручную (не через required CI) через
  `scripts/full-run-ai-advent.sh` с реально доступным headless runner на доверенной машине

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
- runtime parse/runtime и lifecycle ошибки после async start отражаются в `GET /api/pipeline/runs/<run_id>.error_code` (например, `runner_parse_failed`, `run_canceled`, `run_reconciled_after_restart`)

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
4) реализовать orchestrator + runtime provider adapters (`claude-code`, `qwen-code`)
5) реализовать model store (entity-per-file) и extraction coverage for integrations/datastores/CI-CD
6) реализовать UI (workspace setup, charter/skills/run/results)
