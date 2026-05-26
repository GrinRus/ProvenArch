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
> **Последняя ревизия README:** 2026-05-25.

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
staging directories, managed permission policy и audit warnings, а не на hard sandbox.
Default остаётся `trusted_full_access`: ACP передаёт provider CLI текущие full-access flags.
Opt-in `runtime.profile.permissions.mode: managed` отключает эти flags и auto-approves
только операции внутри runtime task envelope; неизвестные запросы в non-interactive run
завершаются `runtime_permission_required`. Для чувствительных или рискованных прогонов
используйте disposable checkout или чистую branch и ревьюйте результат перед commit.

Workspace может содержать prompts, raw runtime logs, repository context, findings, questions
и другие evidence анализа. Считайте его project data и коммитьте только по политике вашей
команды.

## Установка

Обычный пользовательский путь - native single-binary `acp` из GitHub Releases.
Go и Node.js нужны только для разработки ProvenArch из исходников.

Release status:

- latest public release: `v0.1.1`;
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
mkdir -p "$HOME/acp-workspaces"

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

1. `Source`: выберите GitHub/GitLab URL или local checkout folder и docs imports folder.
2. `Readiness`: сохраните и провалидируйте `workspace.yaml`, затем запустите readiness checks; для первого walkthrough используйте `fake`.
3. `Charter`: проверьте стартовый architecture charter и baseline prompts.
4. `Analysis`: запустите первый `init` analysis и следите за run status/logs.
5. `Review` / `Proposals` / `Ask` / `Publish`: просмотрите coverage, artifacts, diagrams, proposals, задайте read-only Q&A и подготовьте git changes.

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
Provider ID - это значение для `--runtime-provider`; executable command - это реальная CLI
команда, установленная на вашей машине.

| Runtime mode/provider ID | Executable command | Typical use |
| --- | --- | --- |
| `--runtime fake` | не требуется | Первый walkthrough и deterministic baseline |
| `--runtime headless --runtime-provider claude-code` | `claude-code`, либо `ACP_CLAUDE_CMD=claude` | Default live analysis provider |
| `--runtime headless --runtime-provider qwen-code` | `qwen`, либо `ACP_QWEN_CMD=<command>` | Optional live analysis provider |
| `--runtime headless --runtime-provider codex-code` | `codex`, либо `ACP_CODEX_CMD=<command>` | Optional live analysis provider |

`--runtime headless` - opt-in режим для live анализа; default локальный baseline остается
`--runtime fake`.

Пример live CLI smoke через Claude CLI. Если ваш binary называется `claude-code`, строку
`export ACP_CLAUDE_CMD=claude` можно не задавать.

```bash
export ACP_CLAUDE_CMD=claude

acp doctor \
  --workspace "$HOME/acp-workspaces/my-service" \
  --repo-path "$HOME/src/my-service" \
  --runtime headless \
  --runtime-provider claude-code

acp run \
  --workspace "$HOME/acp-workspaces/my-service" \
  --pipeline init \
  --runtime headless \
  --runtime-provider claude-code \
  --non-interactive
```

Тот же workspace можно открыть в UI:

```bash
export ACP_CLAUDE_CMD=claude

acp serve \
  --workspace "$HOME/acp-workspaces/my-service" \
  --runtime headless \
  --runtime-provider claude-code
```

Provider selection precedence для каждого pipeline/QA step:

```text
workspace.yaml runtime.profile.steps.<step>.provider
  -> CLI --runtime-provider or ACP_RUNTIME_PROVIDER
  -> claude-code
```

For Ask, `<step>` is `qa`, mapped to runtime step id `qa.ask`.

В `--runtime fake` configured provider только валидируется как fallback config;
live provider command не запускается, а runtime metadata записывает `fake`.

Runtime permission mode хранится в `workspace.yaml`:

```yaml
runtime:
  profile:
    permissions:
      mode: trusted_full_access # default; managed is opt-in
      approval_channel: fail_fast # or ui
```

В UI это настраивается в `Readiness -> Advanced runtime settings -> Runtime Permissions`. В managed mode orchestrator
auto-approves reads under `read_context_roots` и writes under `write_root`/`draft_final_root`.
Shell/network/package-install/unknown requests не auto-approve-ятся; pending requests видны
в `Analysis -> Pending permissions` и правом inspector.

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

UI показывает то же состояние через stage-based console:

- `Source` / `Readiness`: repo sources, `workspace.yaml`, validation diagnostics, doctor checklist и runtime profile;
- `Analysis`: run status, warnings, event timeline, raw agent stream, pending permissions и cancel;
- `Review` / `Proposals`: coverage, artifacts, diagram previews, changelog/proposal artifacts;
- `Ask`: async agent-backed Q&A поверх existing workspace artifacts через `POST /api/qa/runs`; legacy deterministic `POST /api/qa/ask` остаётся compatibility endpoint;
- `Publish`: git commit/proposal branch helper actions.

Можно задавать вопросы по generated workspace artifacts. В UI целевой путь — async Q&A run: ACP собирает deterministic `context-pack.json`, запускает runtime step `qa.ask` через selected provider/fake baseline, валидирует `qa-answer.json` и сохраняет audit artifacts только в `reports/taskruns/<run_id>/qa/`.

```bash
acp qa \
  --workspace "$HOME/acp-workspaces/my-service" \
  --question "Who owns payments-service?"

curl -fsS -X POST http://127.0.0.1:8080/api/qa/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Who owns payments-service?"}'
```

Current/compatibility split:

- UI stage `Ask` uses async runtime-backed `POST /api/qa/runs` + polling `GET /api/qa/runs/<run_id>`.
- `acp qa` and public read-only `POST /api/qa/ask` remain deterministic workspace-backed compatibility surfaces.
- QA runs do not mutate source repos or canonical `charter/`, `model/`, `reports/*`, or `proposals/*`; they write only taskrun/audit artifacts under `reports/taskruns/<run_id>/qa/`.

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
- Headless providers пишут staged shard/final artifacts, manifests и validator outputs.
- Orchestrator валидирует manifests, строит indexes, выводит `model/*` и публикует reports.
- Promotion в stable `reports/*` и `proposals/*` требует validator approval.
- Required CI использует deterministic fixtures и fake runtime, без live network/provider dependencies.

Baseline bundle включает agents `domain-analyst`, `architect-aggregator`, `system-analyst-qa`; skills: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`, `qa`; prompt packs `constitution`, `collect-context`, `findings`, `proposals`, `qa`.

Editable prompt pack layer подключается к `step0.constitution`, `step1.collect`,
`step3.findings` и `step4.proposals`. `step2.asis_docs` использует enforced policy only и не имеет отдельного editable `as-is` prompt pack.

Artifact ownership taxonomy:

- provider-authored: runtime manifests, draft files, shard packs и validator outputs;
- orchestrator-authored: run indexes, citation indexes, shard plans/summaries, logs/history;
- compiler-derived: promoted reports, diagrams, normalized renderers и `model/*`.

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

## Проверка установки и качества

Минимальная проверка установленного binary:

```bash
acp version

acp doctor \
  --workspace "$HOME/acp-workspaces/my-service" \
  --repo-path "$HOME/src/my-service"

acp serve \
  --workspace "$HOME/acp-workspaces/my-service" \
  --auto-init \
  --repo-name my-service \
  --repo-path "$HOME/src/my-service" \
  --runtime fake \
  --dry-run

acp run \
  --workspace "$HOME/acp-workspaces/my-service" \
  --pipeline init \
  --runtime fake \
  --non-interactive
```

После этого проверьте, что появились ключевые файлы:

```bash
test -f "$HOME/acp-workspaces/my-service/reports/as-is/overview.md"
test -f "$HOME/acp-workspaces/my-service/reports/coverage/summary.md"
test -f "$HOME/acp-workspaces/my-service/reports/findings/findings.md"
```

Installer проверяет `checksums.txt` перед установкой binary. Для hardening-релизов GitHub
Release artifacts также могут включать SBOM/provenance files; используйте их как
дополнительный supply-chain evidence при внутренней политике вашей команды.

## Пользовательские документы

- [docs/INSTALL.md](docs/INSTALL.md) - install paths и source build prerequisites.
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) - local diagnostics и common failures.
- [SUPPORT.md](SUPPORT.md) - support scope и evidence expectations.
- [SECURITY.md](SECURITY.md) - vulnerability reporting.
- [CHANGELOG.md](CHANGELOG.md) - user-visible release notes.
- [LICENSE](LICENSE) - Apache-2.0 license.

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
и user-visible changes tracked в [CHANGELOG.md](CHANGELOG.md).

## License

Apache-2.0. См. [LICENSE](LICENSE).
