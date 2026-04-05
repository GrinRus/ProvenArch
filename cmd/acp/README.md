# cmd/acp/

Go CLI entrypoint for ACP.

Usage surface:
- `acp serve --workspace <abs-path> [--runtime fake|headless]` for local UI/API bound to a single workspace per process
- `acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--non-interactive]` for batch/non-interactive execution, including GitHub/GitLab CI jobs, hook-triggered workflows, and manual pipeline buttons
- `acp qa --workspace <abs-path> --question "<text>"` for read-only Q&A over workspace artifacts

Notes:
- required CI/CD surface in MVP: CLI batch mode
- internal API trigger is optional and only valid for trusted local/private long-running deployments
- runtime policy: `fake` default for required deterministic CI, `headless` is opt-in for real local runs
- current baseline behavior:
  - `run` executes deterministic `init|refresh` flow and materializes artifacts in workspace
  - `serve` starts local API server (`/api/*`), `--dry-run` validates wiring without listener start
