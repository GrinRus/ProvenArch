# cmd/acp/

Go CLI entrypoint for ACP.

Usage surface:
- `acp init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports] [--force]` for first-time workspace bootstrap (`workspace.yaml` + fixed layout + baseline bundle + dry validation)
- `acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path ./docs/imports]]` for local UI/API bound to a single workspace per process
- `acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]` for batch/non-interactive execution, including GitHub/GitLab CI jobs, hook-triggered workflows, and manual pipeline buttons
- `acp qa --workspace <abs-path> --question "<text>"` for read-only Q&A over workspace artifacts

Notes:
- quickest local MVP setup: `acp serve --workspace ... --auto-init --repo-name ... --repo-path ... --runtime fake` -> `acp run --workspace ... --pipeline init --runtime fake --non-interactive`
- required CI/CD surface in MVP: CLI batch mode
- internal API trigger is optional and only valid for trusted local/private long-running deployments
- runtime policy: `fake` default for required deterministic CI, `headless` is opt-in for real local runs
- headless providers in MVP: `claude-code` (default) and `qwen-code`
- provider selection precedence: `--runtime-provider` > `ACP_RUNTIME_PROVIDER` > `claude-code`
- provider commands: `ACP_CLAUDE_CMD` (default `claude-code`), `ACP_QWEN_CMD` (default `qwen`)
- in `--runtime fake` mode provider value валидируется, но runner фактически не используется
- run logs retention:
  - `serve` and `run` support `--run-logs-ttl-hours` and `--run-logs-max-runs`
  - env equivalents: `ACP_RUN_LOGS_TTL_HOURS`, `ACP_RUN_LOGS_MAX_RUNS`
  - defaults: `168` hours and `200` run-log files
- bootstrap policy:
  - if workspace root is not a git repo, ACP runs `git init` automatically
  - ACP never auto-commits/auto-pushes
- multi-repo bootstrap:
  - `--repos-file` accepts YAML with `repos:` (or top-level array)
  - `path` entries in `--repos-file` can be relative to repos-file location
- current baseline behavior:
  - `run` executes deterministic `init|refresh` flow and materializes artifacts in workspace
  - `serve` starts local API server (`/api/*`), `--dry-run` validates wiring without listener start
  - `serve` работает в lenient startup mode: сервис не блокируется на repo preflight; readiness diagnostics доступны через `/api/workspace/validate`
