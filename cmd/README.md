# cmd/

Здесь живут CLI entrypoints ACP.

`cmd/acp` реализует рабочие команды:
- `serve` — локальный API+UI сервер для одного workspace.
- `run` — non-interactive/interactive запуск `init|refresh`.
- `qa` — read-only вопросы по артефактам workspace.
