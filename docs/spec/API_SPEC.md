# Спецификация API (implemented baseline, v0)

Этот документ фиксирует фактический wire-contract HTTP API для local-first ACP standalone сервера.

> MVP режим: `acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ... [--docs-imports-path <path>]]`.
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

Примеры repo diagnostics в `warnings[]`/`errors[]`:
- `workspace.repo.ref.invalid` (error, для `path` source)
- `workspace.repo.ref.resolved_via_remote` (warning, `ref` разрешён через `origin/*`)
- `workspace.repo.ref.head_mismatch` (warning, локальный `HEAD` отличается от ожидаемого `ref`)

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

### GET `/api/runtime/timeouts`
Возвращает timeout-конфиг для текущего workspace в трёх представлениях:
- `persisted` — значения из `workspace.yaml` (`runtime.profile.timeouts`);
- `effective` — значения после precedence-resolve (`env > workspace > defaults`);
- `source` — источник каждого effective значения (`env|workspace|default`).

**200**
```json
{
  "ok": true,
  "persisted": {
    "step_timeout_sec": 1800
  },
  "effective": {
    "step_timeout_sec": 1800,
    "heartbeat_sec": 30,
    "pipeline_timeout_sec": 2400,
    "pipeline_kill_grace_sec": 30,
    "api_ready_timeout_sec": 60,
    "api_init_timeout_sec": 120,
    "ui_init_poll_timeout_sec": 900,
    "ui_cancel_poll_timeout_sec": 420
  },
  "source": {
    "step_timeout_sec": "workspace",
    "heartbeat_sec": "default",
    "pipeline_timeout_sec": "default",
    "pipeline_kill_grace_sec": "default",
    "api_ready_timeout_sec": "default",
    "api_init_timeout_sec": "default",
    "ui_init_poll_timeout_sec": "default",
    "ui_cancel_poll_timeout_sec": "default"
  }
}
```

### PUT `/api/runtime/timeouts`
Partial update persisted timeout-полей в `workspace.yaml`.

**Request**
```json
{
  "timeouts": {
    "pipeline_timeout_sec": 3000,
    "ui_init_poll_timeout_sec": 1200
  }
}
```

Правила:
- payload должен содержать хотя бы одно поле в `timeouts`;
- поддерживается partial update (изменяются только переданные поля);
- каждое поле должно быть целым `> 0`.

**200**
```json
{
  "ok": true,
  "persisted": {
    "pipeline_timeout_sec": 3000,
    "ui_init_poll_timeout_sec": 1200
  },
  "effective": {
    "step_timeout_sec": 1800,
    "heartbeat_sec": 30,
    "pipeline_timeout_sec": 3000,
    "pipeline_kill_grace_sec": 30,
    "api_ready_timeout_sec": 60,
    "api_init_timeout_sec": 120,
    "ui_init_poll_timeout_sec": 1200,
    "ui_cancel_poll_timeout_sec": 420
  },
  "source": {
    "pipeline_timeout_sec": "workspace",
    "ui_init_poll_timeout_sec": "workspace"
  }
}
```

**400**
- `invalid_request_body`
- `runtime_timeouts_empty`
- `runtime_timeouts_invalid`

**500**
- `runtime_timeouts_render_failed`
- `runtime_timeouts_write_failed`
- `runtime_timeouts_reopen_failed`

### GET `/api/runtime/execution`
Возвращает execution-профиль для текущего workspace:
- `persisted` — значения из `workspace.yaml` (`runtime.profile.execution`);
- `effective` — значения после precedence-resolve (`CLI > env > workspace > defaults`);
- `source` — источник каждого effective значения (`override|env|workspace|default`).

**200**
```json
{
  "ok": true,
  "persisted": {
    "strategy": "parallel",
    "max_parallel_tasks": 4,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "heuristics"
  },
  "effective": {
    "strategy": "parallel",
    "max_parallel_tasks": 4,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "heuristics"
  },
  "source": {
    "strategy": "workspace",
    "max_parallel_tasks": "workspace",
    "failure_policy": "workspace",
    "shard_discovery_mode": "workspace"
  }
}
```

### PUT `/api/runtime/execution`
Partial update persisted execution-полей в `workspace.yaml`.

**Request**
```json
{
  "execution": {
    "strategy": "parallel",
    "max_parallel_tasks": 6,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "semantic"
  }
}
```

Правила:
- payload должен содержать хотя бы одно поле в `execution`;
- поддерживается partial update (изменяются только переданные поля);
- `strategy` в `sequential|parallel`;
- `max_parallel_tasks` должен быть целым `> 0`;
- `failure_policy` в `fail_fast|best_effort`;
- `shard_discovery_mode` в `heuristics|semantic`.

**400**
- `invalid_request_body`
- `runtime_execution_empty`
- `runtime_execution_invalid`

**500**
- `runtime_execution_render_failed`
- `runtime_execution_write_failed`
- `runtime_execution_reopen_failed`

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
- runtime mode/provider фиксируются конфигурацией процесса (`acp serve --runtime ... --runtime-provider ...`) и не задаются per-request.

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

### POST `/api/pipeline/runs/<run_id>/cancel`
Запрашивает отмену async run.

Отмена не вводит новый `status` enum:
- pending (`queued` в debounce queue) переводится в terminal `failed` немедленно;
- active (`queued/running` текущий active run) отменяется cooperative через `context cancel` и завершается `failed`.

**Request body**
- пустой body допустим;
- optional `{}` допустим;
- неизвестные поля → `400 invalid_request_body`.

**202**
```json
{
  "run_id": "run_20260403_001",
  "status": "cancel_requested"
}
```

**404**
- `run_not_found`

**409**
- `run_not_cancelable` (run уже terminal: `succeeded|failed`)

**400**
- `invalid_request_body`

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

Для lifecycle сценариев используются дополнительные `error_code`:
- `run_canceled` — run отменён пользователем;
- `run_reconciled_after_restart` — stale run был reconciled в `failed` при старте сервиса после рестарта (всегда для `queued`, и для `running` без resumable shard artifacts).
- `run_partial_failed` — run завершён после `best_effort` shard execution, но один или более shard-ов завершились ошибкой.

### GET `/api/pipeline/runs?limit=<n>`
Возвращает список запусков pipeline (queued/running/succeeded/failed), отсортированный по `started_at desc`.

Параметры:
- `limit` optional, default `50`, max `500`

**200**
```json
{
  "items": [
    {
      "run_id": "run_20260403_001",
      "pipeline": "init",
      "status": "succeeded",
      "started_at": "2026-04-03T12:00:00Z",
      "finished_at": "2026-04-03T12:00:02Z",
      "current_step": "init.step4.proposals",
      "warnings": [],
      "error_code": null,
      "error": null
    }
  ]
}
```

**400**
- `invalid_limit`

### GET `/api/pipeline/runs/<run_id>/logs?cursor=<n>&limit=<n>`
Возвращает run-level лог-стрим (cursor pagination) для выбранного run.

Логи сохраняются в workspace:
- `reports/taskruns/logs/<run_id>.ndjson`

Параметры:
- `cursor` optional, default `0`, должен быть `>= 0`
- `limit` optional, default `200`, max `500`

**200**
```json
{
  "run_id": "run_20260403_001",
  "items": [
    {
      "cursor": 0,
      "timestamp": "2026-04-03T12:00:00Z",
      "level": "info",
      "kind": "event",
      "step_id": "init.step1.collect",
      "domain_id": "payments-service",
      "message": "runtime task started",
      "taskrun_path": "reports/taskruns/run_20260403_001-step1-collect-domain-payments-service.json",
      "fields": {
        "task_id": "task-run_20260403_001-init-step1-collect-payments-service",
        "provider": "claude-code",
        "shard_id": "payments-service-services-api",
        "repo_scope": "payments-service",
        "repo_scopes": ["payments-service"],
        "path_scopes": ["services/api"],
        "stderr_snippet": "json parse error ... [truncated]"
      }
    },
    {
      "cursor": 1,
      "timestamp": "2026-04-03T12:00:01Z",
      "level": "info",
      "kind": "runtime_output",
      "stream": "stdout",
      "step_id": "init.step1.collect",
      "domain_id": "payments-service",
      "message": "Agent runtime line from stdout",
      "fields": {
        "output_truncated": false
      }
    }
  ],
  "next_cursor": 2,
  "eof": false
}
```

`items[].kind`:
- `event` — orchestrator lifecycle/event logs (default when field is omitted)
- `runtime_output` — raw runtime stdout/stderr stream forwarded during task execution

`items[].stream`:
- optional; only for `kind=runtime_output`
- values: `stdout`, `stderr`

`output_truncated` policy:
- внутренний hard-cap safeguard применяется к raw runtime stream;
- при срабатывании публикуется `runtime_output` entry с `fields.output_truncated=true`.

**400**
- `invalid_cursor`
- `invalid_limit`

**404**
- `run_not_found`

**500**
- `run_logs_unavailable`

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
- `reports/taskruns/run-history.json`
- `reports/taskruns/logs/*`
- runtime run registry/status (`/api/pipeline/runs/*`)

## 7) Follow-up boundary (post-beta)

`POST /api/qa/ask` не входит в текущий beta required surface и остаётся follow-up slice (Epic 11), если не появится отдельное release-требование.
Каноническая фиксация runtime/Q&A boundary: `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.

## 8) Deployment boundary
- `acp init-workspace --workspace <abs-path> ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>)`: explicit bootstrap.
- `acp serve --workspace <abs-path> [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--auto-init ((--repo-name <name> (--repo-path <path> | --repo-git-url <url>) [--repo-ref <ref>]) | --repos-file <path>) [--docs-imports-path <path>]]`: local interactive и trusted local/private deployment.
- bootstrap behavior: если workspace root не является git-репозиторием, ACP автоматически выполняет `git init` (без auto-commit/auto-push).
- `serve` startup работает в lenient mode: сервис стартует без блокирующего repo preflight; readiness diagnostics доступны через `POST /api/workspace/validate`.
- `acp run --workspace <abs-path> --pipeline init|refresh [--runtime fake|headless] [--runtime-provider claude-code|qwen-code] [--execution-strategy sequential|parallel] [--max-parallel-tasks <n>] [--failure-policy fail_fast|best_effort] [--non-interactive]`.
- run logs retention knobs:
  - CLI flags: `--run-logs-ttl-hours`, `--run-logs-max-runs` (для `serve` и `run`)
  - env overrides: `ACP_RUN_LOGS_TTL_HOURS`, `ACP_RUN_LOGS_MAX_RUNS`
  - defaults: `168h` TTL и `200` run log files
- default runtime mode: `fake` (required deterministic CI surface), `headless` — opt-in.
- default runtime provider: `claude-code`; fallback env `ACP_RUNTIME_PROVIDER`; CLI override `--runtime-provider`.
- provider-specific command envs: `ACP_CLAUDE_CMD` и `ACP_QWEN_CMD`.
- timeout control envs:
  - `ACP_RUNTIME_STEP_TIMEOUT_SEC`
  - `ACP_RUNTIME_HEARTBEAT_SEC`
  - `ACP_PIPELINE_TIMEOUT_SEC`
  - `ACP_PIPELINE_KILL_GRACE_SEC`
  - `ACP_API_READY_TIMEOUT_SEC`
  - `ACP_API_INIT_TIMEOUT_SEC`
  - `ACP_UI_INIT_POLL_TIMEOUT_SEC`
  - `ACP_UI_CANCEL_POLL_TIMEOUT_SEC`
- timeout precedence: `env > workspace.yaml(runtime.profile.timeouts) > defaults`.
- execution precedence: `CLI > env > workspace.yaml(runtime.profile.execution) > defaults`.
- execution env overrides: `ACP_EXECUTION_STRATEGY`, `ACP_MAX_PARALLEL_TASKS`, `ACP_FAILURE_POLICY`, `ACP_SHARD_DISCOVERY_MODE`.
- при `--runtime fake` provider value валидируется, но live provider command не выполняется.
- GitHub/GitLab hooks/manual jobs для required CI/CD должны использовать CLI batch mode с deterministic defaults (`--runtime fake`).
- API-trigger не должен превращаться в hosted control plane в рамках MVP.
