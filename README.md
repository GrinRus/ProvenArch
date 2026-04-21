# Architecture Control Plane (Local-first MVP)

> **Статус:** MVP beta foundation / runnable local docs-first pipeline baseline + strict contracts
> **Принятый стек реализации:** Go (backend/orchestrator) + React/TypeScript UI (embedded), runtime анализа в MVP: **headless multi-provider** (`claude-code` default, `qwen-code` optional, `codex-code` release peer)
> **Последняя ревизия:** 2026-04-19

## Что это

Architecture Control Plane (ACP) — **local-first** инструмент, который строит и поддерживает **as-is архитектурную модель** multi-repo системы через agentic runtime.

ACP не является "рисовалкой диаграмм". Архитектура трактуется как **версионируемый набор runtime-authored документов + индексов в Git**. Совместимый `model/*` в MVP остаётся производным слоем, а не primary source of truth для live pipeline.

---

## Статус репозитория

Сейчас репозиторий содержит:
- набор документов для стейкхолдеров и инженеров,
- контракты и схемы,
- рабочий local-first backend/API/CLI baseline (`init|refresh` execution path),
- staged docs-first runtime pipeline для `reports/taskruns/*` с validator-gated promotion в стабильные `reports/*` и `proposals/*`,
- deterministic derived materialization для `model/*`, `reports/*`, `proposals/*`, `changelog`,
- UI shell + `make` entrypoints + repo CI.

Реализация остаётся incremental по `docs/BACKLOG.md`, но базовый e2e поток уже исполним: `workspace validate -> run pipeline -> inspect artifacts`.

Канонический статус stakeholder-plan (`implemented vs planned`) зафиксирован в [docs/STAKEHOLDER_DOC.md](docs/STAKEHOLDER_DOC.md), секция **Canonical Stakeholder Matrix (source of truth)**.

---

## Scope MVP (явно)

✅ В MVP включено:
- step-scoped runtime provider selection для headless режима: `claude-code` (default fallback), `qwen-code` и `codex-code` (release peer)
- local-first режим (всё запускается локально)
- запуск того же standalone orchestrator в CI/CD через GitHub/GitLab hooks и/или manual pipeline/job trigger
- единый формат хранения: central `arch-workspace` git-репозиторий (Variant 2)
- источники репозиториев: локальные checkout-папки и/или GitHub/GitLab `git_url`, разрешаемые через локальный `git` контекст пользователя/runner
- локально импортированные документы
- интерактивный wizard "Конституции проекта"
- deterministic Step 0 support-artifacts materialization из `charter/wizard/step0-contract.json`, а canonical `charter/overview.md` / `skills/subagents.yaml` публикуются только из валидных runtime draft artifacts (`constitution-draft.json` + draft finals) с warning в run diagnostics при missing/invalid wizard contract
- встроенный baseline bundle agents/skills/prompts + редактируемые в UI prompt packs, версионируемые в Git
- domain-first иерархия агентов (domain analysts + architect aggregator)
- docs-first runtime contract: shard analysts пишут dossier packs в run-scoped staging, validator даёт canonical verdict, promotion переносит только approved final set
- markdown-карточки доменов/команд как source-of-truth в `charter/cards`
- internal Q&A capability системного аналитика поверх артефактов workspace (`internal/qa` + `acp qa`, без публичного API endpoint в beta surface)
- итерационный changelog в `reports/changelog`
- детальный анализ каждого сервиса: архитектура, внешние интеграции, БД, CI/CD
- анализ arbitrary stacks через выбранный headless provider (`claude-code|qwen-code|codex-code`) + baseline prompt bundle, без фиксированного whitelist парсеров в MVP
- headless runtime получает не только `arch-workspace`, но и resolved source repo directories из `workspace.yaml` / `workspace-validate.json`, чтобы live analysis опирался на реальные checkout-ы, а не только на ACP-generated scaffold
- явная фиксация недостатка информации через `coverage`, `questions` и findings
- semantic guard в refresh-цикле: фильтрация нерелевантных placeholder-операций, fallback finding при owner-gap, канонизация/дедуп coverage+questions
- Git-based versioning/branching для модели, правил, отчётов и proposal-пакетов
- строгий runtime contract: staged filesystem artifact packs + `shard-pack-manifest` / `final-run-index` / `citation-index` / `validator-verdict`
- persisted runtime execution metadata сохраняется как internal audit/replay surface для fake/live runners

## Docs-First Runtime Pipeline

Primary execution path для `step0..step4`:
- runtime пишет только в `reports/taskruns/<run_id>/...` внутри своего `write_root` и `draft_final_root`
- provider резолвится на уровне шага, а не всего run; global `--runtime-provider` / `ACP_RUNTIME_PROVIDER` остаются fallback только если step override не задан
- `step0/2/4` materialize-ят staged draft manifests (`constitution-draft.json`, `asis-draft-manifest.json`, `proposals-draft-manifest.json`)
- `step0/2/4` считаются successful только если draft manifest contract валиден и все referenced draft files реально существуют под `draft_final_root`
- publish для `step0/2/4` идёт только из validated runtime draft artifacts через deterministic compile/publish path; direct orchestrator writer больше не является альтернативным source-of-truth для canonical outputs
- runtime draft manifest contract (`version=1`, `run_id`, `step_id`, `step_contract`, `agent_role`, `outputs[]`) является единым internal source of truth для writer + validator; `qwen` дополнительно делает один step-aware artifact-repair retry для draft-only шагов до возврата в orchestrator
- runtime validators для collect manifests и draft manifests read-only по умолчанию; допустимая reconciliation вынесена в явную runtime repair/canonicalization стадию до финальной validation
- если `qwen` на draft-only шаге уже записал draft manifest + draft files, но завис до завершения процесса, runtime принудительно завершает provider process и делает один constrained artifact-only retry.
- shard agents materialize-ят authored docs + `shard-pack-manifest.json`
- persisted collect manifests с workspace-level `documents[].path` drift (`reports/...`, `charter/...`, `proposals/...` или duplicated `artifact_root`) отбрасываются до `step2`, а qwen repair может детерминированно нормализовать только safe path drift
- orchestrator/aggregator собирает staged final doc set в `reports/taskruns/<run_id>/staging/final/`
- validator пишет `validator-verdict.json`
- только schema/semantic/validator gates разрешают promotion в канонические `reports/as-is/*`, `reports/findings/*`, `reports/coverage/*`, `reports/agent-outputs/*`, `proposals/*`
- обязательного human gate перед publish больше нет

Machine-readable канон для дальнейшего рендера:
- `final run index`
- `citation index`

Protected read-only surfaces для runtime:
- `workspace.yaml`
- `schemas/*`
- `docs/spec/*`
- `charter/*`
- анализируемые user repos

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
- workspace prompt packs участвуют в runtime prompt composition как editable content layer; merge order фиксирован: provider header -> artifact-only contract/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer. Enforced safety/contract rules приходят из internal runtime step policy и не могут быть ослаблены содержимым prompt pack.
- baseline prompt defaults структурированы по обязательным секциям (`Goal`, `Inputs`, `Required Output Shape`, `Evidence Policy`, `Forbidden Behavior`, `Fallback When Unknown`) и покрыты quality-тестом на минимальную насыщенность.

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
Для реальных запусков `--runtime headless` нужен установленный provider command (`claude-code`/`qwen`/`codex`) либо env override (`ACP_CLAUDE_CMD`/`ACP_QWEN_CMD`/`ACP_CODEX_CMD`).
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
Для operability UI автоматически выбирает newest active run (или первый из списка), показывает полный warnings list выбранного run, поддерживает top-level navigation `Setup / Baseline / Runs / Results / Settings`, в `Runs: Logs` позволяет переключать `event timeline | raw agent stream | all` и `line | line+fields`, и поддерживает `Cancel selected run`.
`Results` включает отдельную поверхность `Diagrams` для `reports/diagrams/*` (Mermaid preview + index).
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
- `--runtime-provider` поддерживает `claude-code` (default), `qwen-code` и `codex-code`.
- effective precedence выбора provider для каждого шага: `workspace.yaml.runtime.profile.steps.<step>.provider` > CLI `--runtime-provider` / `ACP_RUNTIME_PROVIDER` > `claude-code`.
- command env:
  - `ACP_CLAUDE_CMD` (default `claude-code`)
  - `ACP_QWEN_CMD` (default `qwen`)
  - `ACP_CODEX_CMD` (default `codex`)
- direct `claude` режим: `ACP_CLAUDE_CMD=claude` (native one-shot invocation с envelope parse).
- в `--runtime fake` provider проходит валидацию, но фактически не используется runner’ом.

### 6.1) Runtime timeouts (persisted + effective)

Timeout-конфиг хранится в `workspace.yaml` (`runtime.profile.timeouts`) и используется backend/full-run/frontend e2e.
Канонический contract для persisted/effective/source representation и API surfaces удерживается в [docs/spec/API_SPEC.md](docs/spec/API_SPEC.md); runtime semantics и ownership boundary описаны в [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

### 6.2) Runtime execution profile (persisted + effective)

Execution-конфиг хранится в `workspace.yaml` (`runtime.profile.execution`) и управляет шардированием runtime-задач.
Step provider overrides живут в `workspace.yaml.runtime.profile.steps.*.provider`.
Точный wire/API contract, precedence и update surface удерживаются в [docs/spec/API_SPEC.md](docs/spec/API_SPEC.md), а sharding/runtime behavior boundary — в [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

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

Канонические runbook-документы:
- [docs/LOCAL_FULL_RUN_AI_ADVENT.md](docs/LOCAL_FULL_RUN_AI_ADVENT.md)
- [docs/RELEASE_LIVE_E2E_RUNBOOK.md](docs/RELEASE_LIVE_E2E_RUNBOOK.md) — агентский pre-release live gate (`PASS|FAIL`, strict zero-failure)

Доступные entrypoint-скрипты:
- `scripts/full-run-ai-advent.sh` — полный локальный cycle over `TARGET_REPOS_FILE`
- `scripts/full-run-batch-5x2.sh` — batch re-audit + frontend live e2e
- `scripts/full-run-batch-matrix.sh` — multi-profile matrix orchestrator
- `scripts/frontend-live-e2e.sh` — локальный UI smoke для выбранного provider

Быстрый локальный запуск:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml ./scripts/full-run-ai-advent.sh
```

Быстрый matrix запуск:

```bash
E2E_MATRIX_FILE=/abs/path/to/e2e-matrix.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
./scripts/full-run-batch-matrix.sh
```

Ключевые правила:
- канонический input для full-run и batch: `TARGET_REPOS_FILE`
- release matrix и trusted-machine gate не входят в required CI merge gate
- подробные matrix/sweep/shard/release rules, output contracts и troubleshooting удерживаются только в runbook docs выше, а не в `README.md`

Для release decision и детального live gate используйте:
- [docs/RELEASE_LIVE_E2E_RUNBOOK.md](docs/RELEASE_LIVE_E2E_RUNBOOK.md)

Repo CI по умолчанию живёт в GitHub Actions:
- `contracts`
- `backend`
- `ui`
- `golden`
- `smoke-cli`
- `smoke-api`

---

## High-level архитектура (локально)

- **arch-workspace/**: charter, skills, model, reports, proposals
- **Repo source resolver**: локальные `path` или `git_url`, разрешаемые в локальные checkout через системный `git` текущего пользователя/runner
- **Agent topology**: domain analysts, architect aggregator, system analyst Q&A + baseline bundle skills/prompts
- **UI**: guided workspace setup + baseline editors для `charter/*` и `skills/*`, запуск pipeline, просмотр результатов
- **Orchestrator (Go)**: шаги pipeline, context/prompt packs, вызов runtime, local execution и CI/CD trigger execution
- **Runtime providers**: headless jobs анализа через `claude-code|qwen-code|codex-code`; `fake` остаётся deterministic baseline, default fallback остаётся `claude-code`
- **Model store**: `model/` в формате entity-per-file, включая внешние системы и datastores
- **Reports/Proposals**: `reports/` (включая `agent-outputs/` и `changelog/`) и `proposals/`

### Data flow (MVP)

```mermaid
flowchart LR
  U[User] --> UI[Local UI]
  SCM[GitHub/GitLab hooks or pipeline button] --> ORCH
  UI --> ORCH[Orchestrator (Go)]
  ORCH --> WS[arch-workspace (git)]
  ORCH --> CC[Headless Runtime Provider (claude-code or qwen-code or codex-code)]
  ORCH --> SRC[Repo sources from workspace.yaml]
  SRC --> REPOS[Local checkout paths]
  SRC --> GITLAB[GitHub/GitLab git_url via local git]
  GITLAB --> REPOS
  CC --> DOCS[Local docs/imports]
  CC --> REPOS
  CC --> OUT["Docs-first artifact pack (shard/final indexes + verdict)"]
  CC --> META["runtime-execution.json (internal metadata)"]
  OUT --> ORCH
  META --> ORCH
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

## Контракт runtime output: artifact-only docs-first

Primary runtime contract для live `step1.collect`/`step3.findings`:
- `shard-pack-manifest.json`
- `final-run-index.json`
- `citation-index.json`
- `validator-verdict.json`

Docs-first semantic rules:
- `citation-index.json.claim_ids` образуют глобальное пространство имён в пределах assembled staged final set; один и тот же `claim_id` нельзя переиспользовать между разными shard/citation surfaces.
- `shard-pack-manifest.json.semantic` всегда materialize-ится полностью (`coverage`, `questions`, `entities`, `edges`, `findings`), а коллекционные retry не считаются успешными, если manifest остаётся missing/invalid/skeletal.
- `shard-pack-manifest.json.documents[].path` всегда strict `artifact_root`-relative; runtime может детерминированно нормализовать duplicated `artifact_root` prefix только если файл реально существует внутри `write_root`, но persisted workspace-relative staging paths считаются contract-invalid drift.
- validator path может чинить только technical/reference drift в staged indexes; дублирующиеся `claim_id` детерминированно переименовываются в citation index без semantic rewrite authored docs.

Persisted runtime execution metadata:
- сериализуется как internal `runtime-execution.json` payload рядом с taskrun artifacts
- используется для replay/recovery, taskrun diagnostics и raw-output linking
- provider/API transport transcripts (например `[API Error: ... SSL ...]`) классифицируются как `runner_unavailable` с обязательным сохранением raw stdout/stderr artifacts
- не является semantic source of truth для canonical `reports/*`/`proposals/*` promotion path

Primary promotion gate:
- только `validator-verdict = PASS` разрешает promotion staged final docs в canonical stable paths.

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

Пример: `examples/validator-verdict.example.json`.

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
- явные top-level секции `Setup / Baseline / Runs / Results / Settings`;
- `Settings` как отдельная вкладка для runtime profile (`timeouts` + `execution`) и effective step providers;
- `Runs: Logs` с dual stream surface (`event timeline` + `raw agent stream`) и quick actions;
- `Results` с отдельной вкладкой `Diagrams` (filter/open/preview C4 Mermaid artifacts);
- просмотр остальных результатов (`as-is`, findings, proposals) и repo validation overview (`resolved_repos` + diagnostics по repo);
- просмотр coverage/questions по недостающим данным;
- вызовы backend через `/api/*` (см. `docs/spec/API_SPEC.md`).

---

## Стратегия тестирования (baseline)

- source of truth: `docs/TESTING_STRATEGY.md`
- required CI использует synthetic fixtures, recorded runner outputs и не зависит от live headless providers / live network
- baseline layers:
  - contract tests для `workspace.yaml`, `shard-pack-manifest`, `final-run-index`, `citation-index`, `validator-verdict` и persisted runtime execution metadata
  - semantic validator tests
  - golden/regression tests для docs-first staged/promoted outputs и derived model layer
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
- runtime contract/runtime и lifecycle ошибки после async start отражаются в `GET /api/pipeline/runs/<run_id>.error_code` (например, `runtime_contract_failed`, `run_canceled`, `run_reconciled_after_restart`)

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
- `schemas/shard-pack-manifest.schema.json` — JSON Schema primary shard runtime contract
- `schemas/final-run-index.schema.json` — JSON Schema staged final run index
- `schemas/citation-index.schema.json` — JSON Schema citation normalization layer
- `schemas/validator-verdict.schema.json` — JSON Schema validator release gate
- `schemas/workspace.schema.json` — JSON Schema для `workspace.yaml`
- `examples/workspace.example.yaml` — пример workspace config
- `examples/shard-pack-manifest.example.json` — пример shard manifest
- `examples/final-run-index.example.json` — пример final index
- `examples/citation-index.example.json` — пример citation index
- `examples/validator-verdict.example.json` — пример validator verdict
- `cmd/acp/main.go` — CLI entrypoint (`serve`, `run`, `qa`)
- `ui/package.json` — UI toolchain + scripts
- `fixtures/README.md` — baseline fixtures и regression surface
- `docs/BACKLOG.md` — эпики и acceptance criteria
- `docs/BASELINE_POLICY.md` — правила сопровождения baseline

---

## Порядок реализации

1) финализировать artifact-only docs-first runtime contracts + derived model rebuild
2) реализовать baseline bundle agents/skills/prompts
3) реализовать CI/CD trigger surface: hooks/manual pipeline button/job + batch mode
4) реализовать orchestrator + runtime provider adapters (`claude-code`, `qwen-code`, `codex-code`)
5) реализовать model store (entity-per-file) и extraction coverage for integrations/datastores/CI-CD
6) реализовать UI (workspace setup, charter/skills/run/results)
