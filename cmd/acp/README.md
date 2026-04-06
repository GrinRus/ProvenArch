# cmd/acp/

Go CLI entrypoint for ACP.

Usage surface:
- `acp init-workspace --workspace <abs-path> --repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>] [--docs-imports-path ./docs/imports] [--force]` for first-time workspace bootstrap (`workspace.yaml` + fixed layout + dry validation)
- `acp serve --workspace <abs-path> [--runtime fake|headless] [--auto-init --repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>] [--docs-imports-path ./docs/imports]]` for local UI/API bound to a single workspace per process
- `acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--non-interactive]` for batch/non-interactive execution, including GitHub/GitLab CI jobs, hook-triggered workflows, and manual pipeline buttons
- `acp qa --workspace <abs-path> --question "<text>"` for read-only Q&A over workspace artifacts

Notes:
- quickest local MVP setup: `acp serve --workspace ... --auto-init --repo-name ... --repo-path ... --runtime fake` -> `acp run --workspace ... --pipeline init --runtime fake --non-interactive`
- required CI/CD surface in MVP: CLI batch mode
- internal API trigger is optional and only valid for trusted local/private long-running deployments
- runtime policy: `fake` default for required deterministic CI, `headless` is opt-in for real local runs
- current baseline behavior:
  - `run` executes deterministic `init|refresh` flow and materializes artifacts in workspace
  - `serve` starts local API server (`/api/*`), `--dry-run` validates wiring without listener start
  - `serve` работает в lenient startup mode: сервис не блокируется на repo preflight; readiness diagnostics доступны через `/api/workspace/validate`
