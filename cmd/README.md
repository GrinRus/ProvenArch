# cmd/

Здесь живут CLI entrypoints ACP.

`cmd/acp` реализует рабочие команды:
- `init-workspace` — bootstrap `workspace.yaml` + fixed layout для первого запуска.
- `serve` — локальный API+UI сервер для одного workspace (`--auto-init` поддерживает bootstrap при отсутствии manifest).
- `run` — non-interactive/interactive запуск `init|refresh`.
- `qa` — read-only вопросы по артефактам workspace.
