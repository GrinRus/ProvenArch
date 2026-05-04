# Установка ProvenArch

> Primary distribution для beta: native single-binary `acp` из GitHub Releases. Go/Node нужны только разработчикам, которые собирают проект из исходников.

## Быстрый путь через install.sh

```bash
curl -fsSL https://raw.githubusercontent.com/GrinRus/ProvenArch/main/install.sh | sh
```

По умолчанию installer ставит бинарь в `~/.local/bin/acp`.
Если этой директории нет в `PATH`, добавьте её:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Опциональные переменные:

```bash
ACP_VERSION=v0.1.0 INSTALL_DIR=/usr/local/bin sh install.sh
```

Installer скачивает archive `acp_<os>_<arch>.tar.gz`, проверяет `checksums.txt` и устанавливает только бинарь `acp`.

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
acp doctor --workspace "$HOME/acp-workspaces/my-service" --repo-git-url https://github.com/org/my-service.git

acp serve \
  --workspace "$HOME/acp-workspaces/my-service" \
  --auto-init \
  --repo-name my-service \
  --repo-git-url https://github.com/org/my-service.git \
  --runtime fake
```

Откройте [http://127.0.0.1:8080](http://127.0.0.1:8080), пройдите `Setup -> First run`, нажмите `Save and validate workspace.yaml`, затем `Run first analysis`.

## Реальный headless runtime

Для первого walkthrough используйте `--runtime fake`.
Для live анализа установите один из provider commands и перезапустите сервис:

```bash
acp serve \
  --workspace "$HOME/acp-workspaces/my-service" \
  --runtime headless \
  --runtime-provider claude-code
```

Provider commands:
- `claude-code` или `ACP_CLAUDE_CMD`
- `qwen` или `ACP_QWEN_CMD`
- `codex` или `ACP_CODEX_CMD`

## Установка из исходников

Этот путь нужен разработчикам ProvenArch:

```bash
git clone https://github.com/GrinRus/ProvenArch.git
cd ProvenArch
make bootstrap
make build
./bin/acp doctor --workspace "$HOME/acp-workspaces/my-service"
```
