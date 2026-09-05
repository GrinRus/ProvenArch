# Contributing

ACP — **AI-native** и **spec-first** проект.

## Start here
1) Прочитайте `AGENTS.md`
2) Прочитайте `README.md`, затем только относящиеся к задаче разделы по
   [карте разработки](docs/AGENT_DEVELOPMENT.md)
3) Относитесь к `schemas/*` и `docs/spec/*` как к контрактам

## Local bootstrap
- Требуемый стек: Go exact version из `.go-version`, Node exact version из `.node-version`,
  Python exact version из `.python-version`, npm 10.x, Git, ShellCheck 0.11.x или новее.
- Репозиторные Python tests/scripts запускайте через `./scripts/run-python.sh`; wrapper fail-fast
  остановит suite до запуска, если найденный `python3` не совпадает с `.python-version`.
- `go.mod` остаётся на language compatibility level `go 1.20`; не используйте это как разрешение собирать release устаревшим Go toolchain.
- Для каждого нового checkout/worktree выполните `make bootstrap`: preflight проверяет toolchain,
  затем setup загружает Go modules без `go mod tidy`, устанавливает UI и contract tools по lockfiles
  и Python dependencies из `scripts/requirements-dev.txt` в игнорируемый локальный `.venv`.
  `run-python.sh` использует этот venv после явно заданных `ACP_PYTHON_BIN` /
  `ACP_PYTHON_TOOL_CANDIDATES` runtime overrides; активировать
  venv в shell не требуется. Setup не устанавливает и не меняет версии самих toolchains.
- `make preflight` повторно проверяет toolchain и наличие зависимостей без установки и без сети.
  При недоступном exact Node используйте `ACP_NODE_TOOL_CANDIDATES=/path/to/pinned-node/bin`;
  для Go — `ACP_GO_BIN`, для базового Python при создании venv — `ACP_PYTHON_BASE_BIN`.
  Этот base locator уступает готовому `.venv`. `ACP_PYTHON_BIN` и legacy
  `ACP_PYTHON_TOOL_CANDIDATES` выбирают тестовое окружение явно; bootstrap остановится до установки,
  если такой override направляет проверки вне локального `.venv`. Для обычного setup снимите runtime
  override и используйте base locator. Не отключайте version checks.
- Contract validation uses a separate locked npm toolchain in `tools/contracts`. `make contracts`
  runs `npm ci` for that package and then validates schemas/examples with the repo-local
  `ajv-cli`, `ajv-formats` and `js-yaml` versions from `tools/contracts/package-lock.json`.
  Tool version updates must include an explicit `tools/contracts/package.json` and
  `tools/contracts/package-lock.json` diff.
- Для повторной локальной проверки установленных contract tools используйте `make contracts-check`.
  `make test` выполняет эту проверку и Go/Python/UI suites без переустановки npm packages.
  Если изменился dependency lockfile, сначала повторите `make bootstrap`.
- Прогоните DoD-проверки:
  - `make contracts`
  - `make test`
  - `make lint`
  - `make build`
- Для изменений в `ui/src`, `ui/vite.config.ts` или embedded UI assets дополнительно проверьте:
  - `make verify-ui-determinism` — текущие файлы, включая незакоммиченные (`UI_SOURCE=WORKTREE`);
    для проверки commit укажите `make verify-ui-determinism UI_SOURCE=HEAD` или точный Git ref.
  - `make build`, затем `make verify-ui-dist` — сравнение временной сборки с embedded assets
    текущего worktree без перезаписи проверяемых assets и без требования staging/commit.
  - Для rendered behavior: `./scripts/run-npm.sh exec --prefix ui -- playwright install chromium`,
    затем `bash scripts/ui-mock-e2e.sh`. Harness создаёт отдельный results directory и сервер
    для запуска; сохраняйте напечатанный путь к evidence. Live providers для этой проверки не нужны.

## Проверки по типу изменения

| Изменение | Узкая локальная проверка | CI surface |
| --- | --- | --- |
| Agent guidance, skills, локальные ссылки | `make verify-agent-guidance` | `backend` / `internal/docsync` |
| JSON/YAML artifact contracts | `make contracts-check` + tests затронутого validator | `contracts`, `backend` |
| Go behavior | `./scripts/run-go.sh test ./internal/<package>` | `backend` |
| Scripts | `./scripts/run-python.sh -m unittest scripts.tests.<module>` | `backend`; ShellCheck в `lint` |
| React behavior | `./scripts/run-npm.sh run test --prefix ui -- --run <test-file>` | `ui` |
| TypeScript / Go formatting / shell syntax | `make lint` | `lint` |
| Embedded UI | `make build`, `make verify-ui-determinism`, `make verify-ui-dist` | `ui` |

Узкие проверки помогают во время работы; завершённый implementation slice проходит полный DoD
выше. Research/review без правок не требует запуска полного build. Фактические GitHub required
settings проверяются отдельно от этой карты workflow jobs.
В одном вызове Make targets исполняются последовательно даже с `-j`: installation, verification
и build используют общие зависимости/assets. Между отдельными процессами Make общей блокировки
нет; координатор должен разделять setup/build и использующие те же файлы проверки.
Параллельную работу вести в независимых worktrees.

## ADRs
Пишите ADR, если:
- выбираете/добавляете крупную зависимость
- меняете схемы/контракты
- меняете семантику пайплайна или правила модели

См. `docs/adr/`.

## Canonical references
- Repo CI и required jobs: `README.md` + `docs/TESTING_STRATEGY.md`
- Планирование крупных slice: `docs/PLANS.md`

## Planning
Используйте `docs/PLANS.md` для многосоставных задач.

## Community and security

- Следуйте [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
- Уязвимости не публикуйте в issues; используйте [SECURITY.md](SECURITY.md).
- Пользовательские вопросы и bug reports оформляйте по [SUPPORT.md](SUPPORT.md).
- User-facing изменения отражайте в [CHANGELOG.md](CHANGELOG.md) и release notes.

## Review expectations

Pull request должен быть небольшим, воспроизводимо проверенным и не должен добавлять live network/provider dependency в required CI. Изменения `schemas/*` и `docs/spec/*` требуют синхронного обновления docs, examples/fixtures, валидаторов и тестов.
UI source changes must include the deterministic `internal/api/ui_dist` refresh produced by `make build`; stale embedded bundles are rejected by `make verify-ui-dist`.
