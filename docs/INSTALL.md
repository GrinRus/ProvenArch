# Установка ProvenArch

> Primary distribution для beta: native single-binary `acp` из GitHub Releases. Go/Node нужны только разработчикам, которые собирают проект из исходников.

## Release status

- Latest public release: `v0.1.9`
- Next prepared candidate: `v0.1.10` (unpublished; the tag remains blocked on fresh trusted-machine
  composite release evidence)
- Supported platforms: macOS/Linux on `amd64` and `arm64`
- License: Apache-2.0
- Primary install command:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/GrinRus/ProvenArch/main/install.sh | sh
  ```

## Быстрый путь через install.sh

```bash
curl -fsSL https://raw.githubusercontent.com/GrinRus/ProvenArch/main/install.sh | sh
```

По умолчанию installer ставит бинарь в `~/.local/bin/acp`.
Если этой директории нет в `PATH`, добавьте её:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Проверьте установленный binary:

```bash
acp version
```

Опциональные переменные:

```bash
ACP_VERSION=v0.1.9 INSTALL_DIR=/usr/local/bin sh install.sh
```

По умолчанию `ACP_VERSION=latest`: installer разрешает последний GitHub Release через Releases API, включая beta/prerelease releases, затем скачивает archive `acp_<os>_<arch>.tar.gz`, проверяет `checksums.txt` и устанавливает только бинарь `acp`.
Начиная с `v0.1.1` release workflow также публикует SBOM/provenance artifacts; installer продолжает доверять checksum verification как обязательному install gate.

## Ручная установка из GitHub Release

1. Откройте GitHub Releases проекта.
2. Скачайте archive для вашей платформы:
   - `acp_darwin_amd64.tar.gz`
   - `acp_darwin_arm64.tar.gz`
   - `acp_linux_amd64.tar.gz`
   - `acp_linux_arm64.tar.gz`
3. Проверьте checksum из `checksums.txt`.
4. Распакуйте `acp` в директорию из `PATH`.

## Первый запуск

```bash
mkdir -p "$HOME/acp-workspaces"
acp serve
```

Откройте [http://127.0.0.1:8080](http://127.0.0.1:8080).
При запуске без `--workspace` UI откроет onboarding:

1. `Workspace`: выберите или создайте architecture workspace, например `$HOME/acp-workspaces/my-service`. Успешно открытые workspaces сохраняются в локальный список Recent workspaces; missing entries можно удалить кнопкой `Forget`.
2. `Sources`: добавьте один или несколько target repositories через Git URL или local checkout path.
3. `Runner`: выберите default `fake` для первого deterministic walkthrough.
4. `Ready`: откройте Console V2 или запустите первый analysis.

При reopening существующего workspace UI загружает repos из `workspace.yaml`. Если manifest валиден,
после выбора runner можно сразу перейти в Console V2; если manifest невалиден, onboarding оставит
оператора в `Sources` с actionable diagnostics.

Launcher workspace path должен быть dedicated directory под текущим `$HOME` или system temp
directory; используйте direct-mode `--workspace` для automation/advanced paths.

Direct-mode для scripts, CI и опытных пользователей остаётся доступным:

```bash
acp doctor --workspace "$HOME/acp-workspaces/my-service" --repo-git-url https://github.com/org/my-service.git

acp serve \
  --workspace "$HOME/acp-workspaces/my-service" \
  --auto-init \
  --repo-name my-service \
  --repo-git-url https://github.com/org/my-service.git \
  --runtime fake
```

## Реальный headless runtime

Для первого walkthrough достаточно `acp serve`: default runtime остаётся `fake`.
Для live анализа установите один из provider commands и перезапустите сервис.
Provider ID передается через `--runtime-provider`, а executable command можно переопределить
через env var:

| Provider ID | Default/override command |
| --- | --- |
| `claude-code` | `ACP_CLAUDE_CMD`, затем `claude`, затем legacy `claude-code` |
| `qwen-code` | `qwen`, либо `ACP_QWEN_CMD=<command>` |
| `codex-code` | `codex`, либо `ACP_CODEX_CMD=<command>` |

Перед live-анализом проверьте provider. Для Claude CLI с binary `claude` env override не нужен;
если команда называется иначе, задайте `ACP_CLAUDE_CMD`:

```bash
acp doctor \
  --workspace "$HOME/acp-workspaces/my-service" \
  --repo-git-url https://github.com/org/my-service.git \
  --runtime headless \
  --runtime-provider claude-code

acp serve \
  --workspace "$HOME/acp-workspaces/my-service" \
  --runtime headless \
  --runtime-provider claude-code
```

## Установка из исходников

Этот путь нужен разработчикам ProvenArch:

Prerequisites:
- Go exact version из `.go-version` для security-patched local/release builds.
- Node.js exact version из `.node-version`.
- npm 10.x.

```bash
git clone https://github.com/GrinRus/ProvenArch.git
cd ProvenArch
make bootstrap
make build
./bin/acp version
./bin/acp doctor --workspace "$HOME/acp-workspaces/my-service"
```

Source build требует exact Node.js version из `.node-version`. Если локальный `node` отличается, `make bootstrap`, `make test`, `make lint` и `make build` завершаются до генерации UI assets с ошибкой resolver-а.
Если exact Node установлен не первым в `PATH`, укажите каталог с matching `node` и `npm` через `ACP_NODE_TOOL_CANDIDATES=/path/to/node-22.21.1/bin`. Minor drift не принимается: `22.22.3` не заменяет требуемый `22.21.1`.
`go.mod` сохраняет language compatibility level `go 1.20`, но release/local builds должны выполняться Go version из `.go-version`.
