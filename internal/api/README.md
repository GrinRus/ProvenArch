# internal/api/

HTTP API handlers (implemented baseline).

## Реализованные endpoints
- `GET /api/health`
- `POST /api/workspace/validate`
- `GET /api/workspace/manifest`
- `PUT /api/workspace/manifest`
- `GET /api/artifacts?path=<relative>`
- `POST /api/artifacts/write`
- `POST /api/git/commit`
- `POST /api/git/proposal-branch`
- `POST /api/pipeline/init`
- `POST /api/pipeline/refresh`
- `GET /api/pipeline/runs/<run_id>`
- `GET /api/pipeline/runs/<run_id>/artifacts`

## Wire-contract notes
- Единый error envelope: `{ "error": { "code": "...", "message": "..." } }`
- `POST /api/workspace/validate` при ошибке возвращает envelope + diagnostics:
  - `ok=false`
  - `errors[]`
  - `warnings[]`
  - `resolved_repos[]`
- `POST /api/workspace/validate` всегда включает pre-run layout readiness diagnostics:
  - `workspace.layout.dir.missing` (warning: будет создано на run)
  - `workspace.layout.dir.not_dir` / `workspace.layout.dir.unreadable` (errors)
- Для `POST /api/pipeline/init|refresh`:
  - `trigger` должен быть одним из `ui|manual|hook|automation`
  - `commit` и `create_proposal_branch` пока не поддержаны в pipeline start API и возвращают `501 not_supported`
  - headless runtime startup ошибки возвращаются как actionable code `runner_unavailable`
- `runner_parse_failed` не должен приходить из start endpoint; этот код возвращается только как run-level `error_code` после `202`.
- `GET /api/pipeline/runs/<run_id>` возвращает optional `error_code` при failed run (`runner_parse_failed` и др. runtime-level коды после async start).

## UI fallback
- `GET /` и любые non-API routes обслуживаются embedded `ui_dist` SPA.

Каноническая публичная спецификация: `docs/spec/API_SPEC.md`.
