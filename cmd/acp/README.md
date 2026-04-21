# cmd/acp/

Go CLI entrypoint for ACP.

High-level commands:
- `init-workspace` — bootstrap `workspace.yaml`, fixed layout, baseline bundle и dry validation
- `serve` — локальный backend + embedded UI для одного workspace на процесс
- `run` — deterministic `init|refresh` pipeline для local/batch execution
- `qa` — read-only вопросы по артефактам workspace

Canonical sources:
- exact flag/help surface: `acp --help`, `acp <command> --help`, [cmd/acp/main.go](main.go)
- local bootstrap и quickstart: [README.md](../../README.md)
- HTTP API wire contract: [docs/spec/API_SPEC.md](../../docs/spec/API_SPEC.md)
- system boundaries и runtime behavior: [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md)

Operational notes:
- required CI/CD surface в MVP: CLI batch mode через `acp run`
- runtime policy: `fake` default, `headless` opt-in
- headless providers в MVP: `claude-code` (default), `qwen-code`, `codex-code`
- provider selection precedence: `--runtime-provider` > `ACP_RUNTIME_PROVIDER` > `claude-code`
- provider commands: `ACP_CLAUDE_CMD`, `ACP_QWEN_CMD`, `ACP_CODEX_CMD`
- `serve` остаётся lenient startup surface; readiness diagnostics идут через `/api/workspace/validate`
