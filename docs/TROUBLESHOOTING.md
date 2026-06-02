# Troubleshooting

## Проверить установку

```bash
acp doctor --workspace "$HOME/acp-workspaces/my-service" --repo-git-url https://github.com/org/my-service.git
```

`acp doctor --json` возвращает machine-readable report с checks:
- `git`
- `workspace`
- `embedded_ui`
- `port`
- `repo_source`
- `runtime_provider`

Exit codes:
- `0` — ready
- `1` — есть user-fixable issues
- `2` — invalid flags или internal request error

## Git не может прочитать private repo

ACP не хранит credentials. `git_url` работает через локальный `git` и текущий auth context пользователя или CI runner.

Проверьте:

```bash
git ls-remote --heads git@github.com:org/private-repo.git
git ls-remote --heads https://github.com/org/private-repo.git
```

Если эти команды не проходят, настройте SSH key, credential helper или token для локального `git`.

## Workspace path is not writable

Выберите директорию, где пользователь может создавать файлы:

```bash
mkdir -p "$HOME/acp-workspaces"
acp serve --runtime fake
```

Workspace является git-репозиторием с generated architecture artifacts. ACP не пишет в анализируемый user repo.
Если вы используете direct-mode, проверьте путь через doctor:

```bash
acp doctor --workspace "$HOME/acp-workspaces/my-service"
```

## API вернул workspace_not_selected

`acp serve` без `--workspace` стартует launcher/onboarding режим. До выбора workspace доступны `/api/health` и `/api/onboarding/*`; workspace-bound endpoints возвращают `428 workspace_not_selected`.

Решение: откройте UI, выберите или создайте workspace в шаге `Workspace`, затем сохраните sources и runner. Для scripts/CI используйте direct-mode:

```bash
acp serve --workspace "$HOME/acp-workspaces/my-service" --runtime fake
```

## Port 8080 занят

Проверьте другой порт:

```bash
acp doctor --listen 127.0.0.1:8081
acp serve --listen 127.0.0.1:8081
```

## Embedded UI missing

Это означает, что бинарь собран без `ui/dist`.
Используйте release binary или пересоберите из исходников:

```bash
make build
./bin/acp serve
```

## Headless provider command not found

Для fake runtime provider command не нужен.
Для live анализа установите provider или задайте override. В UI-first режиме runner выбирается в onboarding; direct-mode можно запускать сразу с workspace:

```bash
ACP_CLAUDE_CMD=claude acp serve --runtime headless --runtime-provider claude-code

ACP_CLAUDE_CMD=claude acp serve --workspace "$HOME/acp-workspaces/my-service" --runtime headless --runtime-provider claude-code
ACP_QWEN_CMD=qwen acp serve --workspace "$HOME/acp-workspaces/my-service" --runtime headless --runtime-provider qwen-code
ACP_CODEX_CMD=codex acp serve --workspace "$HOME/acp-workspaces/my-service" --runtime headless --runtime-provider codex-code
```

## UI показывает validation errors

В onboarding откройте `Sources`; после входа в Console V2 откройте `Source`. Проверьте:
- repo source mode: `GitHub/GitLab URL` или `Local folder`
- `Repo name` уникален внутри workspace
- для local folder указан абсолютный путь к git checkout
- для URL локальный `git` может выполнить `ls-remote`

После исправления нажмите `Save and validate sources`.
