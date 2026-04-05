# Спецификация API (implemented baseline, v0)

Этот документ фиксирует фактический wire-contract HTTP API для local-first ACP standalone сервера.

> MVP режим: `acp serve --workspace <abs-path>`.
> Service работает с одним bound workspace на процесс.
> Required CI/CD surface: CLI batch mode (`acp run ... --non-interactive`). API-trigger остаётся optional для trusted local/private deployment.

## 1) Конвенции
- Base path: `/api`
- Content type для JSON: `application/json`
- Унифицированная ошибка: `{ "error": { "code": "...", "message": "..." } }`
- В MVP нет встроенной app-level аутентификации; контроль доступа обеспечивается deployment/network boundary.

### Error envelope example
```json
{
  "error": {
    "code": "invalid_request_body",
    "message": "invalid request body"
  }
}
```

## 2) Workspace endpoints

### GET `/api/health`
**200**
```json
{ "status": "ok" }
```

### POST `/api/workspace/validate`
Проверяет bound workspace:
- `workspace.yaml` contract + semantic validation
- layout/manifest checks + pre-run readiness diagnostics (что отсутствует и будет создано на run)
- repo source resolution diagnostics (`path` и `git_url`), без обязательного live network fetch в required flow

Body: отсутствует.

**200**
```json
{
  "ok": true,
  "workspace": "/abs/path/to/arch-workspace",
  "warnings": [],
  "resolved_repos": [
    {
      "name": "payments-service",
      "source": "path",
      "path": "/abs/path/to/payments-service",
      "ref": "main"
    }
  ]
}
```

**400**
```json
{
  "ok": false,
  "workspace": "/abs/path/to/arch-workspace",
  "error": {
    "code": "workspace_validation_failed",
    "message": "workspace validation failed"
  },
  "errors": [],
  "warnings": [],
  "resolved_repos": []
}
```

Примеры layout diagnostics в `warnings[]`/`errors[]`:
- `workspace.layout.dir.missing` (warning: директория отсутствует и будет создана при run)
- `workspace.layout.dir.not_dir` (error: конфликтующий file path вместо required directory)
- `workspace.layout.dir.unreadable` (error: нет доступа к required directory path)

### GET `/api/workspace/manifest`
Возвращает текущее содержимое `workspace.yaml`.

**200**
```json
{ "content": "version: 1\nrepos:\n..." }
```

### PUT `/api/workspace/manifest`
Обновляет `workspace.yaml` после contract parse-check.

**Request**
```json
{ "content": "version: 1\nrepos:\n..." }
```

**200**
```json
{ "ok": true }
```

**400**
- `invalid_request_body`
- `manifest_empty`
- `manifest_invalid`

**500**
- `manifest_write_failed`
- `manifest_reopen_failed`

## 3) Artifacts endpoints

### GET `/api/artifacts?path=<relative>`
Безопасное чтение файла из bound workspace (safe-join, без выхода за root).

**200**
- file content (content-type определяется по расширению)

**400**
- `bad_request` (path отсутствует)
- `path_invalid` (absolute/traversal)

**404**
- `artifact_not_found`

### POST `/api/artifacts/write`
Безопасная запись editable артефактов.

Ограничения:
- разрешены только `charter/*` и `skills/*`

**Request**
```json
{
  "path": "charter/overview.md",
  "content": "# Updated charter"
}
```

**200**
```json
{ "ok": true }
```

**400**
- `invalid_request_body`
- `artifact_path_required`
- `artifact_path_forbidden`
- `artifact_write_failed`

## 4) Pipeline endpoints

### POST `/api/pipeline/init`
### POST `/api/pipeline/refresh`
Запускает async run для соответствующего pipeline.

**Request (optional body)**
```json
{
  "commit": false,
  "create_proposal_branch": false,
  "trigger": "ui"
}
```

`trigger` поддерживает только enum:
- `ui`
- `manual`
- `hook`
- `automation`

Поведение:
- при пустом body используются defaults (`trigger` по endpoint, flags=false)
- `commit=true` или `create_proposal_branch=true` в этом slice возвращают `501 not_supported`
- runtime mode фиксируется конфигурацией процесса (`acp serve --runtime ...`) и не задаётся per-request.

**202**
```json
{
  "run_id": "run_20260403_001",
  "status": "started"
}
```

**400**
- `invalid_request_body`
- `trigger_unsupported`
- `run_start_failed`

**503**
- `runner_unavailable`

`runner_parse_failed` не возвращается start-endpoint'ом; этот код появляется только в run status (`GET /api/pipeline/runs/<run_id>`) после `202 Accepted`.

**501**
- `not_supported`

### GET `/api/pipeline/runs/<run_id>`
Возвращает run status.

**200**
```json
{
  "run_id": "run_20260403_001",
  "pipeline": "init",
  "status": "running",
  "started_at": "2026-04-03T12:00:00Z",
  "finished_at": null,
  "current_step": "init.step1.collect",
  "warnings": [],
  "error_code": null,
  "error": null
}
```

`status` enum:
- `queued`
- `running`
- `succeeded`
- `failed`

Для runtime parse/runtime ошибок после успешного async start используется run-level статус:
- `error_code: "runner_parse_failed"` (или другой actionable code) в `failed` run.

### GET `/api/pipeline/runs/<run_id>/artifacts`
Возвращает список materialized artifacts для run.

**200**
```json
{
  "run_id": "run_20260403_001",
  "artifacts": [
    {
      "path": "reports/as-is/overview.md",
      "kind": "report",
      "label": "System Overview"
    }
  ]
}
```

**404**
- `run_not_found`

## 5) Git helper endpoints

### POST `/api/git/commit`
Коммитит текущие изменения в bound workspace repo.

**Request**
```json
{ "message": "chore: update ACP workspace artifacts" }
```

**200 (committed)**
```json
{
  "ok": true,
  "status": "committed",
  "output": "[main abc123] chore: update ..."
}
```

**200 (no changes)**
```json
{
  "ok": true,
  "status": "no_changes",
  "message": "nothing to commit"
}
```

### POST `/api/git/proposal-branch`
Создаёт или переключает proposal-branch в bound workspace repo.

**Request**
```json
{ "name": "proposal/beta-refresh" }
```

**200**
```json
{ "ok": true, "branch": "proposal/beta-refresh" }
```

## 6) Deterministic scope

Детерминированно сравниваемые артефакты (при одинаковом input и recorded runner):
- `charter/`
- `model/`
- `reports/as-is/`
- `reports/findings/`
- `reports/coverage/`
- `reports/agent-outputs/`
- `proposals/`

Run-specific поверхность (не входит в strict deterministic golden compare):
- `reports/changelog/*`
- `reports/taskruns/*`
- runtime run registry/status (`/api/pipeline/runs/*`)

## 7) Follow-up boundary (post-beta)

`POST /api/qa/ask` не входит в текущий beta required surface и остаётся follow-up slice (Epic 11), если не появится отдельное release-требование.
Каноническая фиксация runtime/Q&A boundary: `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.

## 8) Deployment boundary
- `acp serve --workspace <abs-path> [--runtime fake|headless]`: local interactive и trusted local/private deployment.
- `acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--non-interactive]`.
- default runtime mode: `fake` (required deterministic CI surface), `headless` — opt-in.
- GitHub/GitLab hooks/manual jobs для required CI/CD должны использовать CLI batch mode с deterministic defaults (`--runtime fake`).
- API-trigger не должен превращаться в hosted control plane в рамках MVP.
