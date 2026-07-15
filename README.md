# Architecture Control Plane (Local-first MVP)

[![License](https://img.shields.io/github/license/GrinRus/ProvenArch)](LICENSE)
[![Release](https://img.shields.io/github/v/release/GrinRus/ProvenArch?include_prereleases)](https://github.com/GrinRus/ProvenArch/releases)
[![backend](https://github.com/GrinRus/ProvenArch/actions/workflows/backend.yml/badge.svg)](https://github.com/GrinRus/ProvenArch/actions/workflows/backend.yml)
[![ui](https://github.com/GrinRus/ProvenArch/actions/workflows/ui.yml/badge.svg)](https://github.com/GrinRus/ProvenArch/actions/workflows/ui.yml)

ACP строит Git-версионируемую compiled architecture knowledge base для одного или
нескольких локальных репозиториев. Результат - не картинка и не чат-лог, а набор
файлов с evidence, отчетами, диаграммами, findings, proposals, derived model и
read-only health snapshot по опубликованным workspace artifacts.

В одну строку:

```text
operator CLI / UI -> ACP Go orchestrator -> runtime provider -> staged artifacts -> validator -> arch-workspace files
```

> **Статус:** MVP beta / pre-v1 foundation. Public API, artifact contracts и UX могут меняться до `v1.0.0`.
> **Стек реализации:** Go backend/orchestrator + embedded React/TypeScript UI.
> **Runtime анализа:** deterministic `fake` baseline или headless providers `claude-code`, `qwen-code`, `codex-code`.
> **Последняя ревизия README:** 2026-07-15.

## Что это

Architecture Control Plane (ACP) - **local-first инструмент анализа архитектуры** и
поддержания проверяемой architecture knowledge base.
Вы указываете один или несколько репозиториев и, при необходимости, импортированные
документы. ACP запускает staged pipeline, просит runtime provider собрать архитектурные
evidence, валидирует созданные артефакты и записывает принятый результат в отдельный
architecture workspace. LLM/runtime drafts remain staged, orchestrator validates/promotes,
human review принимает решения, а Git хранит accepted architecture knowledge.

`reports/as-is/overview.md` является каноническим Architecture Home: он ведёт к областям,
потокам, интеграциям, safe-change guidance и явно отмеченным evidence gaps. Каждый
`init|refresh` также сохраняет `source-revisions.json`; `refresh` дополнительно пишет
schema-validated `refresh-impact-plan.json`, `refresh-execution.json` и `refresh-materialization.json`.
Безопасный unchanged/out-of-scope refresh завершается без provider и canonical rewrites; selective
refresh переиспользует только валидные baseline shard packs, иначе fail-closed выполняет полный pipeline.

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
staging directories, managed permission policy, live-harness isolated checkouts for canonical
`path` inputs и runtime write audit, который завершает otherwise-successful provider step
как `runtime_contract_failed`, если protected workspace surfaces или analyzed repo working
tree изменились.
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

- latest public release: `v0.1.9`;
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

Самый простой путь - запустить локальную UI-консоль без заранее выбранного workspace:

```bash
mkdir -p "$HOME/acp-workspaces"

acp serve
```

Откройте [http://127.0.0.1:8080](http://127.0.0.1:8080).
`acp serve` обрабатывает SIGINT/SIGTERM/SIGHUP через bounded shutdown: HTTP listener
останавливается gracefully, active runtime run получает context cancellation, queued pending
runs завершаются как canceled, а новые async starts после shutdown отклоняются.

В onboarding UI:

Верхний setup summary показывает текущий шаг, главный blocker и next action; disabled actions в `Ready` объясняют, чего именно не хватает. Для headless runner onboarding дополнительно показывает expected command, env override и provider readiness guidance до первого live analysis.

1. `Workspace`: создайте или откройте `arch-workspace`, например `$HOME/acp-workspaces/my-service`. ACP инициализирует fixed layout и git в workspace. Успешно открытые workspaces попадают в локальный список Recent workspaces; missing entries можно забыть без изменения самого workspace.
2. `Sources`: добавьте один или несколько target repos через GitHub/GitLab URL или local checkout path, optional `ref`, guided analysis include/exclude globs и `docs.imports_path`.
3. `Analysis brief`: сохраните project name и scope либо подтвердите запуск без brief с явным quality warning.
4. `Runner & readiness`: выберите runner. Для первого walkthrough используйте default `fake`; live providers (`claude-code`, `qwen-code`, `codex-code`) включаются явно. Если provider command/auth/quota не готов, recovery panel показывает expected executable, `ACP_*_CMD` override и безопасный fallback на `fake`.
5. `Review & start`: проверьте summary и запустите первый `init` после successful local readiness check.

Если вы открываете уже существующий workspace, onboarding загружает repos из `workspace.yaml`.
Валидный manifest можно сразу довести до `Ready` после выбора runner; manifest с ошибками вернёт
оператора к `Sources` с actionable diagnostics.

После setup основной shell использует четыре destination: `Home / Runs / Knowledge / Changes`.
Home показывает четыре authoritative оси и одну next action; Runs объединяет history, Run Studio и
явную refresh queue; Knowledge читает только current promoted workspace; Changes открывает immutable
run snapshot через Evidence/Findings/Proposals/Diff/Publish. Setup остаётся contextual utility, Ask —
global `Current workspace · read-only` dialog, а diagnostics не влияют на workflow acceptance.
Back/Forward и reload восстанавливают run, source, view и artifact/entity context.

Опытные пользователи, scripts, CI и live E2E могут по-прежнему открыть direct-mode console сразу на известном workspace. Ниже используется уже склонированный локальный Git checkout. Для GitHub/GitLab URL замените `--repo-path "$HOME/src/my-service"` на `--repo-git-url https://github.com/org/my-service.git`.

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
| `--runtime fake` или default `acp serve` | не требуется | Первый walkthrough и deterministic baseline |
| `--runtime headless --runtime-provider claude-code` | `ACP_CLAUDE_CMD`, затем `claude`, затем legacy `claude-code` | Default live analysis provider |
| `--runtime headless --runtime-provider qwen-code` | `qwen`, либо `ACP_QWEN_CMD=<command>` | Optional live analysis provider |
| `--runtime headless --runtime-provider codex-code` | `codex`, либо `ACP_CODEX_CMD=<command>` | Optional live analysis provider |

`--runtime headless` - opt-in режим для live анализа; default локальный baseline остается
`fake`, поэтому для чистого UI onboarding достаточно `acp serve`.

Пример live CLI smoke через Claude CLI. Если ваш binary называется `claude`, env override
не нужен; если команда называется иначе, задайте `ACP_CLAUDE_CMD`. Explicit
`ACP_CLAUDE_CMD=claude` остаётся валидным, но для обычной установки больше не требуется.

```bash
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
в `Analysis -> Pending permissions` с triage summary, policy rule, target/reason и next actions,
а также в правом inspector hard blockers с step, rule, target и reason.

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

UI показывает то же состояние через path-based console:

- `/setup`: Guided Setup `Workspace → Sources → Analysis brief → Runner & readiness → Review & start`; brief сохраняется в существующий Step 0 contract, skip требует quality warning;
- `/home`: четыре authoritative workflow axes и одна primary next action;
- `/runs` и `/runs/<run_id>`: history/Run Studio, active/pending coordination, provider identity и retained recovery; shards/raw output/permissions остаются в локальном Diagnostics disclosure;
- `/knowledge`: только current promoted workspace через `GET /api/knowledge`, включая Overview, validated Atlas с table fallback, searchable Entities и Artifacts; malformed files дают explicit partial state, filename-derived topology не используется;
- `/changes`: historical Change Review с `Overview / Evidence / Findings / Proposals / Diff / Publish`, run-pinned snapshot source и честным `Unknown` publication status для отдельного run;
- global `Ask`: modal/sheet `Current workspace · read-only` с async run history, citations и возвратом из Evidence Viewer; QA status не меняет Changes/Publish gate.

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

- UI stage `Ask` uses async runtime-backed `POST /api/qa/runs` + polling `GET /api/qa/runs/<run_id>` and `GET /api/qa/runs?limit=20` history, with explicit failed-run recovery and same-question retry.
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

- provider-authored: normal runtime manifests, draft files, shard packs и validator outputs; collect manifest runtime recovery помечается отдельно и строится только из already-authored shard docs;
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
или CI runner. Непривязанный `git_url` (`ref` не задан) перед каждым анализом fetch-ится
в ACP-owned cache under `.acp/repos`, затем cache force-reset-ится на exact commit remote
default `HEAD`; resolved commit SHA возвращается в fetch-backed resolver/run evidence. Если
`ref` задан, он продолжает выбирать эту ветку/тег/SHA вместо remote default. ACP не хранит
отдельные repository credentials и не изменяет пользовательские `path` checkout-ы.

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

- [docs/STAKEHOLDER_DOC.md](docs/STAKEHOLDER_DOC.md) - canonical implemented/planned status matrix.
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
