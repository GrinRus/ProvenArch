# Architecture Control Plane (Local-first MVP)

[![License](https://img.shields.io/github/license/GrinRus/ProvenArch)](LICENSE)
[![Release](https://img.shields.io/github/v/release/GrinRus/ProvenArch?include_prereleases)](https://github.com/GrinRus/ProvenArch/releases)
[![backend](https://github.com/GrinRus/ProvenArch/actions/workflows/backend.yml/badge.svg)](https://github.com/GrinRus/ProvenArch/actions/workflows/backend.yml)
[![ui](https://github.com/GrinRus/ProvenArch/actions/workflows/ui.yml/badge.svg)](https://github.com/GrinRus/ProvenArch/actions/workflows/ui.yml)

ACP строит Git-версионируемый architecture workspace для одного или нескольких локальных
репозиториев. Результат - не картинка и не чат-лог, а набор файлов с evidence, отчетами,
диаграммами, findings, proposals и derived model.

В одну строку:

```text
operator CLI / UI -> ACP Go orchestrator -> runtime provider -> staged artifacts -> validator -> arch-workspace files
```

> **Статус:** MVP beta / pre-v1 foundation. Public API, artifact contracts и UX могут меняться до `v1.0.0`.
> **Стек реализации:** Go backend/orchestrator + embedded React/TypeScript UI.
> **Runtime анализа:** deterministic `fake` baseline или headless providers `claude-code`, `qwen-code`, `codex-code`.
> **Последняя ревизия README:** 2026-05-09.

## Что это

Architecture Control Plane (ACP) - **local-first инструмент анализа архитектуры**.
Вы указываете один или несколько репозиториев и, при необходимости, импортированные
документы. ACP запускает staged pipeline, просит runtime provider собрать архитектурные
evidence, валидирует созданные артефакты и записывает принятый результат в отдельный
architecture workspace.

ACP нужен, когда вы хотите получить:

- актуальный **as-is architecture overview** для repo или multi-repo системы;
- отчеты по coverage gaps, findings, proposals, diagrams и ownership hints;
- обычные файлы, которые можно ревьюить через Git, а не opaque chat history;
- повторяемый локальный pipeline, который также можно запускать в CI job без hosted ACP;
- deterministic fake runtime для тестов и первого walkthrough до подключения live AI CLI.

ACP не является:

- hosted SaaS control plane;
- редактором диаграмм;
- security/compliance enforcement engine;
- заменой human review архитектурных решений;
- credential store для GitHub/GitLab или AI providers.

## Статус и безопасность

ACP находится в beta-состоянии и рассчитан на локальную оценку и controlled operator use.

Go orchestrator пишет generated architecture state в настроенный `arch-workspace`.
Анализируемые source repositories задуманы как read-only inputs. Но в live headless mode
ACP запускает внешние provider CLI на той же машине. MVP isolation строится на explicit
staging directories и audit warnings, а не на hard sandbox. Для чувствительных или рискованных
прогонов используйте disposable checkout или чистую branch и ревьюйте результат перед commit.

Workspace может содержать prompts, raw runtime logs, repository context, findings, questions
и другие evidence анализа. Считайте его project data и коммитьте только по политике вашей
команды.

## Установка

Обычный пользовательский путь - native single-binary `acp` из GitHub Releases.
Go и Node.js нужны только для разработки ProvenArch из исходников.

Release status:

- latest public release: `v0.1.0`;
- license: Apache-2.0;
- maturity: MVP beta / pre-v1 foundation.

```bash
curl -fsSL https://raw.githubusercontent.com/GrinRus/ProvenArch/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
acp version
```

Installer резолвит последний GitHub Release, скачивает platform archive, проверяет
`checksums.txt` и устанавливает только binary `acp`.

Поддерживаемые release platforms:

| OS | Architectures |
| --- | --- |
| macOS | `amd64`, `arm64` |
| Linux | `amd64`, `arm64` |

Подробнее: [docs/INSTALL.md](docs/INSTALL.md).

## Первый анализ

Начинайте с deterministic `fake` runtime. Он проверяет install, workspace setup, UI,
pipeline wiring, validators и publication artifacts без AI provider.

Ниже используется уже склонированный локальный Git checkout. Для GitHub/GitLab URL замените
`--repo-path "$HOME/src/my-service"` на `--repo-git-url https://github.com/org/my-service.git`.

```bash
acp doctor \
  --workspace "$HOME/acp-workspaces/my-service" \
  --repo-path "$HOME/src/my-service"

acp serve \
  --workspace "$HOME/acp-workspaces/my-service" \
  --auto-init \
  --repo-name my-service \
  --repo-path "$HOME/src/my-service" \
  --runtime fake
```

Откройте [http://127.0.0.1:8080](http://127.0.0.1:8080).

В UI:

1. `Setup -> Source`: выберите GitHub/GitLab URL или local checkout folder.
2. `Workspace`: оставьте default `docs.imports_path` или выберите imports folder.
3. `Runtime`: для первого walkthrough используйте `fake`.
4. `Validate`: сохраните и провалидируйте `workspace.yaml`, затем запустите readiness checks.
5. `Run`: запустите первый `init` analysis.

Тот же первый анализ можно запустить из CLI:

```bash
acp run \
  --workspace "$HOME/acp-workspaces/my-service" \
  --pipeline init \
  --runtime fake \
  --non-interactive
```

Expected stable outputs после успешного walkthrough:

```text
workspace.yaml
charter/
skills/
reports/as-is/overview.md
reports/coverage/summary.md
reports/findings/findings.md
reports/agent-outputs/architect/summary.md
reports/diagrams/
reports/taskruns/<run_id>/
model/entities/*.yaml
model/edges/*.yaml        # если анализ нашёл связи
proposals/
```

ACP пишет эти файлы в `arch-workspace`, а не в анализируемый source repository.

## Выбор runtime

Используйте `acp doctor`, чтобы проверить local readiness и provider availability.

| Runtime mode/provider | External dependency | Typical use |
| --- | --- | --- |
| `--runtime fake` | нет | Первый walkthrough, deterministic tests, required CI baseline |
| `--runtime headless --runtime-provider claude-code` | `claude-code` command или `ACP_CLAUDE_CMD` | Default live analysis provider |
| `--runtime headless --runtime-provider qwen-code` | `qwen` command или `ACP_QWEN_CMD` | Optional live analysis provider |
| `--runtime headless --runtime-provider codex-code` | `codex` command или `ACP_CODEX_CMD` | Release peer live analysis provider |

`--runtime headless` - opt-in режим для live анализа; default локальный baseline остается
`--runtime fake`.

Пример live run:

```bash
acp doctor \
  --workspace "$HOME/acp-workspaces/my-service" \
  --runtime headless \
  --runtime-provider claude-code

acp serve \
  --workspace "$HOME/acp-workspaces/my-service" \
  --runtime headless \
  --runtime-provider claude-code
```

Provider selection precedence для каждого pipeline step:

```text
workspace.yaml runtime.profile.steps.<step>.provider
  -> CLI --runtime-provider or ACP_RUNTIME_PROVIDER
  -> claude-code
```

В `--runtime fake` configured provider только валидируется как fallback config;
live provider command не запускается, а runtime metadata записывает `fake`.

## Артефакты и логи

ACP хранит evidence анализа как обычные файлы внутри workspace:

```text
reports/taskruns/<run_id>/        # run staging, logs, indexes, raw diagnostics
reports/as-is/                    # promoted as-is architecture docs
reports/coverage/                 # coverage gaps and unknowns
reports/findings/                 # findings from pipeline
reports/diagrams/                 # generated Mermaid C4 views
proposals/                        # proposed follow-up changes
model/entities/, model/edges/     # derived entity-per-file model
```

UI показывает то же состояние через:

- `Runs`: run status, warnings, event timeline, raw agent stream и cancel;
- `Results`: coverage, artifacts и diagram previews;
- `Settings`: persisted runtime timeouts, execution profile и effective step providers.

Можно задавать read-only вопросы по generated workspace artifacts:

```bash
acp qa \
  --workspace "$HOME/acp-workspaces/my-service" \
  --question "Who owns payments-service?"

curl -fsS -X POST http://127.0.0.1:8080/api/qa/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Who owns payments-service?"}'
```

Q&A beta boundary: deterministic workspace-backed read-only service + CLI `acp qa` + public read-only `POST /api/qa/ask`; это не headless runtime agent и не consumer `skills/prompt-packs/qa.md`.

Troubleshooting: [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md).

## Как работает ACP

ACP использует docs-first pipeline. Runtime provider не становится source of truth просто
потому, что напечатал JSON в stdout. Provider обязан записать artifact files в run-scoped
staging directories. ACP валидирует эти artifacts и promoted-ит только approved files в
stable workspace paths.

Canonical init stages:

```text
step0 constitution -> step1 collect -> step2 as-is docs -> step3 findings -> step4 proposals
```

Refresh runs переиспользуют steps `1..4`.

Ключевые правила:

- `workspace.yaml` объявляет repositories, imported docs и runtime profile settings.
- `charter/` и `skills/` хранят human-owned project context и editable baseline prompts.
- Headless providers пишут staged shard/final artifacts, manifests и validator verdicts.
- Orchestrator валидирует manifests, строит indexes, выводит `model/*` и публикует reports.
- Promotion в stable `reports/*` и `proposals/*` требует validator approval.
- Required CI использует deterministic fixtures и fake runtime, без live network/provider dependencies.

Baseline bundle включает agents `domain-analyst`, `architect-aggregator`, `system-analyst-qa`; skills: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`, `qa`; prompt packs `constitution`, `collect-context`, `findings`, `proposals`, `qa`.

Editable prompt pack layer подключается к `step0.constitution`, `step1.collect`,
`step3.findings` и `step4.proposals`. `step2.asis_docs` использует enforced policy only и не имеет отдельного editable `as-is` prompt pack.

Artifact ownership taxonomy:

- provider-authored: runtime manifests, draft files, shard packs и validator verdicts;
- orchestrator-authored: run indexes, citation indexes, shard plans/summaries, logs/history;
- compiler-derived: promoted reports, diagrams, normalized renderers и `model/*`.

Подробные contracts:

- [docs/spec/WORKSPACE_SPEC.md](docs/spec/WORKSPACE_SPEC.md)
- [docs/spec/PIPELINE_SPEC.md](docs/spec/PIPELINE_SPEC.md)
- [docs/spec/MODEL_SPEC.md](docs/spec/MODEL_SPEC.md)
- [docs/spec/API_SPEC.md](docs/spec/API_SPEC.md)
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## Workspace model

ACP хранит generated architecture state в central Git workspace:

```text
arch-workspace/
  workspace.yaml
  charter/
  skills/
  docs/imports/
  reports/
  model/
  proposals/
```

Repository sources могут быть local checkout paths или GitHub/GitLab-style `git_url`.
Для `git_url` ACP использует local `git` command и auth context текущего пользователя
или CI runner. ACP не хранит отдельные repository credentials.

Imported documents, например выгрузки из Confluence, живут под `docs.imports_path` by default.
Optional `<docs.imports_path>/index.yaml` может хранить metadata импортированных файлов:
required поля `id` и `path`, optional поля `source`, `checksum`, `imported_at`,
`source_updated_at`, `status`. Отсутствие index допустимо, malformed/index semantic issues
surfacing как warning-only diagnostics.

## Разработка из исходников

Prerequisites:

- Go exact version из [.go-version](.go-version)
- Node.js exact version из [.node-version](.node-version)
- npm 10.x
- optional provider CLIs для live runtime development

Bootstrap и проверка repository:

```bash
make bootstrap
make contracts
make test
make lint
make build
./bin/acp version
```

Полезные local commands:

```bash
make quickstart-local WORKSPACE=/path/to/arch-workspace REPO_PATH=/path/to/repo REPO_NAME=my-service
make run-backend WORKSPACE=/path/to/arch-workspace
make run-ui
```

Root CLI commands:

```text
acp init-workspace
acp serve
acp run
acp qa
acp doctor
acp version
```

Contribution guide: [CONTRIBUTING.md](CONTRIBUTING.md).

## Test and release evidence

Required CI спроектирован без live AI provider dependencies и использует synthetic fixtures and recorded artifacts, включая artifact fixtures. Local Definition of Done для завершенного slice:

```bash
make contracts
make test
make lint
make build
```

Manual live E2E и release-gate работа отделены от required CI.
Локальный full-run описан в [docs/LOCAL_FULL_RUN_AI_ADVENT.md](docs/LOCAL_FULL_RUN_AI_ADVENT.md):
он использует `TARGET_REPOS_FILE` и entrypoint `scripts/full-run-ai-advent.sh`.
Канонический trusted-machine runbook:
[docs/RELEASE_LIVE_E2E_RUNBOOK.md](docs/RELEASE_LIVE_E2E_RUNBOOK.md). Для live matrix harness
используйте `E2E_MATRIX_FILE` и запускайте `scripts/full-run-batch-matrix.sh` напрямую; не
добавляйте wrapper поверх release gate. Release readiness берется только из
`reports/release_verdict_<matrix-id>.json`.

Testing strategy: [docs/TESTING_STRATEGY.md](docs/TESTING_STRATEGY.md).

## Docs map

- [docs/INSTALL.md](docs/INSTALL.md) - install paths и source build prerequisites.
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) - local diagnostics и common failures.
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - implementation architecture и boundaries.
- [docs/spec/PIPELINE_SPEC.md](docs/spec/PIPELINE_SPEC.md) - pipeline inputs, outputs и artifact ownership.
- [docs/spec/WORKSPACE_SPEC.md](docs/spec/WORKSPACE_SPEC.md) - `workspace.yaml` contract.
- [docs/spec/API_SPEC.md](docs/spec/API_SPEC.md) - local API contracts.
- [docs/spec/MODEL_SPEC.md](docs/spec/MODEL_SPEC.md) - derived model format.
- [docs/STAKEHOLDER_DOC.md](docs/STAKEHOLDER_DOC.md) - canonical stakeholder matrix и implemented/planned status.
- [docs/BACKLOG.md](docs/BACKLOG.md) - backlog и acceptance criteria.
- [docs/PLANS.md](docs/PLANS.md) - active engineering plans.
- [docs/RELEASE_LIVE_E2E_RUNBOOK.md](docs/RELEASE_LIVE_E2E_RUNBOOK.md) - trusted-machine live release gate.

## Repository map

- `cmd/acp/` - CLI entrypoint и command surface.
- `internal/orchestrator/` - pipeline, staging, validation, promotion, run lifecycle.
- `internal/runtime/` - fake runtime, headless provider adapters, runtime policies.
- `internal/workspace/` - workspace manifest, layout, repo resolution, baseline bundle.
- `internal/model/` - derived entity-per-file model store.
- `internal/reports/` - report и diagram materialization.
- `internal/api/` - local API и embedded UI server.
- `ui/` - React/TypeScript UI.
- `schemas/` - JSON Schemas для workspace и runtime artifacts.
- `examples/` - contract examples и matrix inputs.
- `fixtures/` - deterministic test fixtures и golden outputs.
- `scripts/` - smoke checks, release/live E2E harnesses и CI helpers.

## Known MVP limits

- No hosted or multi-tenant mode.
- No security/compliance enforcement engine.
- ACP не принимает public SCM webhooks сам: native webhook listener / external SCM app integration остаются вне MVP.
- No automatic Confluence/Jira/Notion integrations.
- No separate credential store для Git или provider CLIs.
- No hard provider-process sandbox в MVP headless mode.
- Docker/npm/PyPI/Maven/crates.io не являются primary distribution channels для ACP.

## Security and support

Используйте [SECURITY.md](SECURITY.md) для vulnerability reporting. Не открывайте public issues
с secrets, private repository content, raw provider logs или tokens.

Используйте [SUPPORT.md](SUPPORT.md) для support scope и evidence expectations. Release notes
и user-visible changes tracked в [CHANGELOG.md](CHANGELOG.md). Governance и release ownership
описаны в [GOVERNANCE.md](GOVERNANCE.md).

## License

Apache-2.0. См. [LICENSE](LICENSE).
