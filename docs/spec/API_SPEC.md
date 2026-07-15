# Спецификация API (implemented baseline, v0)

Этот документ фиксирует фактический wire-contract HTTP API для local-first ACP standalone сервера.

> Этот документ описывает только HTTP API wire-contract.
> Exact CLI flag/help surface canonical в `acp --help` и `cmd/acp/main.go`; quickstart и runbook usage удерживаются в `README.md` и профильных runbook docs.
> Service работает с одним bound workspace на процесс. Required CI/CD surface: CLI batch mode (`acp run ... --non-interactive`), API-trigger остаётся optional для trusted local/private deployment.

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

### GET `/api/system/version`
Возвращает metadata запущенного ACP binary/UI bundle. Endpoint доступен и в launcher mode до выбора workspace, чтобы onboarding/live screenshots показывали фактическую сборку под тестом.

**200**
```json
{
  "version": "dev",
  "commit": "none",
  "built": "unknown",
  "ui_bundle": "embedded"
}
```

### GET `/api/system/doctor`
Возвращает read-only readiness checklist для first-run UI и локальной диагностики.
Endpoint не меняет workspace.

Query params optional:
- `runtime=fake|headless`
- `runtime_provider=claude-code|qwen-code|codex-code`
- `repo_path=/abs/path/to/repo`
- `repo_git_url=https://github.com/org/repo.git`

`repo_path` и `repo_git_url` взаимоисключающие. Для `repo_git_url` проверка использует локальный `git ls-remote --heads` и текущий auth context пользователя/runner.

**200**
```json
{
  "ok": true,
  "summary": "ready",
  "checks": [
    {
      "id": "git",
      "label": "Git",
      "status": "pass",
      "message": "git found at /usr/bin/git"
    },
    {
      "id": "runtime_provider",
      "label": "Runtime provider",
      "status": "pass",
      "message": "fake runtime selected; no headless provider command required"
    }
  ]
}
```

`checks[].status`:
- `pass` — готово
- `warn` — не блокирует первый fake walkthrough, но пользователь может улучшить конфигурацию
- `fail` — user-fixable blocker

Headless provider readiness reports stable Provider ID separately from executable command.
Command resolution:
- `claude-code`: `ACP_CLAUDE_CMD` -> `claude` -> legacy `claude-code`
- `qwen-code`: `ACP_QWEN_CMD` -> `qwen`
- `codex-code`: `ACP_CODEX_CMD` -> `codex`

Example fail message:
```json
{
  "id": "runtime_provider",
  "label": "Runtime provider",
  "status": "fail",
  "message": "Provider ID: claude-code; executable not found; checked: claude, claude-code",
  "suggestion": "Install claude or set ACP_CLAUDE_CMD to the provider command. Legacy command claude-code is also supported."
}
```

**400**
- `invalid_doctor_request`
- `doctor_failed`

### GET `/api/onboarding/status`

Возвращает launcher/Console boundary вместе с workspace и runtime readiness. Additive поля
`console_entered` и `can_switch_runtime` являются серверной истиной: launcher начинает с
`console_entered=false`, обычный `acp serve --workspace ...` — с `true`.

`can_switch_runtime=true` только до входа в Console и при отсутствии active/pending analysis run.
После входа смена runtime через `POST /api/onboarding/runtime` возвращает
`409 runtime_switch_requires_restart`; UI должен показывать restart-guided command, а не менять
effective runtime работающего процесса.

### POST `/api/onboarding/enter-console`

Фиксирует однонаправленную границу onboarding → Console для текущего server process. Body может
быть пустым или `{}`. Успешный ответ совпадает с onboarding status и содержит
`console_entered=true`, `can_switch_runtime=false`.

**428**
- `console_entry_not_ready` — workspace/service/runtime ещё не готовы.

**400**
- `invalid_request_body`

### GET `/api/onboarding/path-suggestions`
Local-only launcher endpoint for searchable path comboboxes. Endpoint is available before
workspace selection, never writes to target repos, never clones repos and does not change
`workspace.yaml`.

Query params:
- `kind=workspace|repo`
- `query=<typed path or filter text>`

Workspace suggestions include Recent workspaces, `$HOME/acp-workspaces`, `$HOME`, system temp
root and safe discovered child directories under allowed launcher roots. Repo suggestions include
paths already present in the selected/draft `workspace.yaml`, common source roots (`$HOME/src`,
`$HOME/Projects`, `$HOME/code`), current process cwd when safe, and discovered `.git` directories.

Each item:
- `path`: absolute local path
- `label`: display label, usually basename
- `exists`: whether the path currently exists as a directory
- `kind`: `workspace`, `git_repo` or `directory`
- `source`: `recent`, `manifest`, `common`, `cwd`, `query` or `discovered`

**200**
```json
{
  "ok": true,
  "kind": "repo",
  "query": "/Users/me/src",
  "items": [
    {
      "path": "/Users/me/src/payments",
      "label": "payments",
      "exists": true,
      "kind": "git_repo",
      "source": "discovered"
    }
  ]
}
```

**400**
- `path_suggestion_kind_invalid`
- `path_suggestion_query_invalid`
- `path_suggestion_query_traversal`
- `path_suggestion_query_root`

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

### GET `/api/workspace/health`
Возвращает read-only health snapshot по уже опубликованным workspace artifacts.
Endpoint вычисляет отчет на запрос, не пишет `reports/health/*`, не запускает runtime
provider и не блокирует run/review/publish/Q&A flows.

Initial K2a checks:
- observation provenance в `model/entities/*.yaml` и `model/edges/*.yaml` без evidence;
- `reports/agent-outputs/domains/*.md` без matching `charter/cards/domains/<domain-id>.md`;
- `proposals/*/proposal.md` без review sections `Evidence`, `Citations`, `Unresolved`/`Open questions`;
- количество unresolved questions в `reports/coverage/open-questions.md` как info item.

**200**
```json
{
  "version": 1,
  "generated_at": "2026-07-10T00:00:00Z",
  "status": "warn",
  "summary": {
    "info": 1,
    "warning": 1,
    "error": 0
  },
  "items": [
    {
      "id": "model.observation.missing_evidence",
      "severity": "warning",
      "title": "Observation entity \"svc.payments\" has no evidence",
      "path": "model/entities/svc.payments.yaml",
      "related_paths": []
    }
  ]
}
```

`status`:
- `pass` — health findings отсутствуют;
- `warn` — есть advisory warnings/info, но это не promotion gate;
- `fail` — scanner нашел health item severity `error`, например malformed model YAML. HTTP status при этом остается `200`, потому что это состояние workspace artifacts, а не ошибка API lifecycle.

`items[].severity`:
- `info`
- `warning`
- `error`

**500**
- `workspace_health_failed`

### GET `/api/knowledge`
Возвращает read-only read model текущей promoted knowledge base bound workspace. Endpoint ничего
не пишет, не запускает provider и не связывает данные с историческим run без authoritative
association. Поэтому поле `promoted_from_run_id` отсутствует в контракте.

`entities[]` и `edges[]` читаются из `model/entities/*.{yaml,yml}` и
`model/edges/*.{yaml,yml}`. В ответ попадают только записи с обязательными semantic fields;
edge включается только когда обе ссылки разрешаются в набор validated entities. `path` всегда
является canonical workspace-relative path. `artifacts[]` — inventory читаемых файлов в
`model/`, `reports/` и `proposals/`; имена файлов не интерпретируются как topology.

**200**
```json
{
  "version": 1,
  "generated_at": "2026-07-15T00:00:00Z",
  "source_mode": "current_workspace",
  "status": "partial",
  "entities": [
    {
      "id": "svc.payments",
      "type": "service",
      "name": "Payments",
      "provenance": { "kind": "inference", "confidence": 0.9 },
      "path": "model/entities/svc.payments.yaml"
    }
  ],
  "edges": [],
  "artifacts": [
    { "path": "model/entities/svc.payments.yaml", "kind": "entity", "name": "svc.payments.yaml" }
  ],
  "issues": [
    {
      "code": "knowledge.edge_reference_missing",
      "path": "model/edges/edge.payments.calls.missing.yaml",
      "message": "edge references missing entities: from=\"svc.payments\" to=\"svc.missing\""
    }
  ]
}
```

`status`:
- `available` — есть promoted content и все обнаруженные model files валидны;
- `partial` — часть файлов malformed/unreadable, содержит duplicate entity ID или broken edge reference; остальные валидные данные сохраняются;
- `unavailable` — promoted model/report/proposal content отсутствует.

Typed issue codes: `knowledge.entity_malformed`, `knowledge.entity_duplicate`,
`knowledge.edge_malformed`, `knowledge.edge_reference_missing`, `knowledge.file_unreadable`,
`knowledge.inventory_unreadable`.

**405** — любой метод кроме `GET`.

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

Execution payload также включает `steps`:
- `persisted.steps` — step-scoped provider overrides из `workspace.yaml.runtime.profile.steps`;
- `effective.steps` — effective provider per step;
- `source.steps` — источник effective provider per step (`workspace|override|env|default`).

**200**
```json
{
  "ok": true,
  "persisted": {
    "strategy": "parallel",
    "max_parallel_tasks": 4,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "heuristics",
    "steps": {
      "step2_as_is": "qwen-code"
    }
  },
  "effective": {
    "strategy": "parallel",
    "max_parallel_tasks": 4,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "heuristics",
    "steps": {
      "step0_constitution": "claude-code",
      "step1_collect": "claude-code",
      "step2_as_is": "qwen-code",
      "step3_findings": "claude-code",
      "step4_proposals": "claude-code"
    }
  },
  "source": {
    "strategy": "workspace",
    "max_parallel_tasks": "workspace",
    "failure_policy": "workspace",
    "shard_discovery_mode": "workspace",
    "steps": {
      "step2_as_is": "workspace"
    }
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
    "shard_discovery_mode": "semantic",
    "steps": {
      "step2_as_is": "qwen-code",
      "step4_proposals": "claude-code"
    }
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
- `execution.steps.*` в `claude-code|qwen-code|codex-code`.

**400**
- `invalid_request_body`
- `runtime_execution_empty`
- `runtime_execution_invalid`

**500**
- `runtime_execution_render_failed`
- `runtime_execution_write_failed`
- `runtime_execution_reopen_failed`

### GET `/api/runtime/permissions`
Возвращает permission-профиль для текущего workspace:
- `persisted` — значения из `workspace.yaml` (`runtime.profile.permissions`);
- `effective` — значения после resolve (`workspace > defaults`);
- `source` — источник каждого effective значения (`workspace|default`).

**200**
```json
{
  "ok": true,
  "persisted": {
    "mode": "managed"
  },
  "effective": {
    "mode": "managed",
    "approval_channel": "fail_fast"
  },
  "source": {
    "mode": "workspace",
    "approval_channel": "default"
  }
}
```

### PUT `/api/runtime/permissions`
Partial update persisted permission-полей в `workspace.yaml`.

**Request**
```json
{
  "permissions": {
    "mode": "managed",
    "approval_channel": "ui"
  }
}
```

Правила:
- payload должен содержать хотя бы одно поле в `permissions`;
- `mode` в `trusted_full_access|managed`;
- `approval_channel` в `fail_fast|ui`.

**200**
```json
{
  "ok": true,
  "persisted": {
    "mode": "managed",
    "approval_channel": "ui"
  },
  "effective": {
    "mode": "managed",
    "approval_channel": "ui"
  },
  "source": {
    "mode": "workspace",
    "approval_channel": "workspace"
  }
}
```

**400**
- `invalid_request_body`
- `runtime_permissions_empty`
- `runtime_permissions_invalid`

**500**
- `runtime_permissions_render_failed`
- `runtime_permissions_write_failed`
- `runtime_permissions_reopen_failed`

### GET `/api/runtime/profile`
Возвращает aggregate runtime profile:
- `runtime_mode` — effective process mode (`fake|headless`);
- `runtime_provider` — effective process provider ID;
- `provider_source` — источник выбора provider (`cli|env|default`);
- `timeouts`
- `execution`
- `permissions`
- `step_providers`

`step_providers` дублируется отдельно для UI/CLI, чтобы не требовать парсинг execution payload:
- `persisted`
- `effective`
- `source`

**200**
```json
{
  "ok": true,
  "runtime_mode": "headless",
  "runtime_provider": "claude-code",
  "provider_source": "cli",
  "timeouts": {
    "persisted": {},
    "effective": {
      "step_timeout_sec": 1800
    },
    "source": {
      "step_timeout_sec": "default"
    }
  },
  "execution": {
    "persisted": {
      "strategy": "parallel",
      "steps": {
        "step2_as_is": "qwen-code"
      }
    },
    "effective": {
      "strategy": "parallel",
      "steps": {
        "step0_constitution": "claude-code",
        "step1_collect": "claude-code",
        "step2_as_is": "qwen-code",
        "step3_findings": "claude-code",
        "step4_proposals": "claude-code",
        "qa": "claude-code"
      }
    },
    "source": {
      "strategy": "workspace",
      "steps": {
        "step2_as_is": "workspace"
      }
    }
  },
  "permissions": {
    "persisted": {
      "mode": "managed"
    },
    "effective": {
      "mode": "managed",
      "approval_channel": "fail_fast"
    },
    "source": {
      "mode": "workspace",
      "approval_channel": "default"
    }
  },
  "step_providers": {
    "persisted": {
      "step2_as_is": "qwen-code"
    },
    "effective": {
      "step0_constitution": "claude-code",
      "step1_collect": "claude-code",
      "step2_as_is": "qwen-code",
      "step3_findings": "claude-code",
      "step4_proposals": "claude-code",
      "qa": "claude-code"
    },
    "source": {
      "step2_as_is": "workspace"
    }
  }
}
```

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
- server normalizes path before authorization; absolute paths, `..` traversal and bare `charter`/`skills` roots return `artifact_path_forbidden`

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
  "trigger": "ui",
  "intent": "start"
}
```

`intent` optional, default `start`. Обычный `start` при active analysis run возвращает
`409 run_active` и не создаёт history item. `intent=queue` разрешён только для `refresh`:
без active run refresh стартует сразу, а при active run создаёт или заменяет единственный
pending refresh. Заменённый pending run получает terminal `status=canceled`,
`error_code=run_superseded` и `superseded_by_run_id`.

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
- `queue_unsupported`
- `run_start_failed`

**409**
- `run_active`

`runner_unavailable` и `runtime_contract_failed` не возвращаются start-endpoint'ом; эти коды появляются только в run status (`GET /api/pipeline/runs/<run_id>`) после `202 Accepted`, когда конкретный step-scoped provider проходит lazy preflight или runtime execution.

**501**
- `not_supported`

### POST `/api/pipeline/runs/<run_id>/cancel`
Запрашивает отмену async run.

Отмена использует terminal `status=canceled`:
- pending run переводится в `canceled` немедленно;
- active run отменяется cooperative через `context cancel` и завершается `canceled` с
  `error_code=run_canceled`.

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
  "runtime_mode": "headless",
  "step_providers": {
    "step0_constitution": "claude-code",
    "step1_collect": "claude-code",
    "step2_as_is": "qwen-code",
    "step3_findings": "claude-code",
    "step4_proposals": "claude-code"
  },
  "pending_permissions": [],
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
- `canceled`

Для runtime parse/runtime ошибок после успешного async start используется run-level статус:
- `error_code: "runtime_contract_failed"` (или другой actionable code) в `failed` run.

Для non-interactive managed permission запросов, которые не auto-approved:
- `error_code: "runtime_permission_required"` в `failed` run;
- `pending_permissions[]` содержит normalized requests с resolved `decision` для UI/diagnostics.

Для lifecycle сценариев используются дополнительные `error_code`:
- `run_canceled` — run отменён пользователем;
- `run_reconciled_after_restart` — stale run был reconciled в `failed` при старте сервиса после рестарта (всегда для `queued`, и для `running` без resumable shard artifacts).
- `run_partial_failed` — run завершён после `best_effort` shard execution, но один или более shard-ов завершились ошибкой.

### GET `/api/pipeline/runs?limit=<n>`
Возвращает список запусков analysis pipeline (`init|refresh`, queued/running/succeeded/failed/canceled), отсортированный по `started_at desc`. Q&A runs (`pipeline="qa"`) имеют отдельный endpoint `/api/qa/runs` и в этот список не включаются. Ответ также содержит authoritative `coordination`: active run и единственный pending refresh, если они есть. Additive boolean `authoritative_index` сообщает, зарегистрирован ли у run его собственный staged `final-run-index.json`; только `succeeded init|refresh` с этим флагом являются Change Review packages. Финальная content/readability validation всё равно выполняется snapshot loader-ом при выборе run.

Параметры:
- `limit` optional, default `50`, max `500`

**200**
```json
{
  "coordination": {
    "active_run_id": null,
    "pending": null
  },
  "items": [
    {
      "run_id": "run_20260403_001",
      "pipeline": "init",
      "status": "succeeded",
      "started_at": "2026-04-03T12:00:00Z",
      "finished_at": "2026-04-03T12:00:02Z",
      "current_step": "init.step4.proposals",
      "authoritative_index": true,
      "runtime_mode": "headless",
      "step_providers": {
        "step0_constitution": "claude-code",
        "step1_collect": "claude-code",
        "step2_as_is": "qwen-code",
        "step3_findings": "claude-code",
        "step4_proposals": "claude-code",
        "qa": "claude-code"
      },
      "pending_permissions": [],
      "warnings": [],
      "error_code": null,
      "error": null
    }
  ]
}
```

**400**
- `invalid_limit`

### GET `/api/pipeline/runs/<run_id>/permissions`
Возвращает normalized pending permission requests для выбранного run.

**200**
```json
{
  "run_id": "run_20260403_001",
  "requests": [
    {
      "request_id": "perm-1",
      "run_id": "run_20260403_001",
      "step_id": "init.step1.collect",
      "provider": "claude-code",
      "action": "shell",
      "path_or_command": "npm install",
      "reason": "package install requires review",
      "decision": {
        "request_id": "perm-1",
        "decision": "needs_user",
        "rule_id": "ask_unsafe_operation",
        "message": "operation requires explicit user approval"
      }
    }
  ]
}
```

**404**
- `run_not_found`

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
      "taskrun_path": "reports/taskruns/run_20260403_001/staging/shards/payments-service/runtime-execution.json",
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

Для `init|refresh` inventory включает `reports/taskruns/<run_id>/source-revisions.json`; для
`refresh` также включает `reports/taskruns/<run_id>/refresh-impact-plan.json`. Это обычные
readable taskrun artifacts через существующий endpoint, отдельного HTTP endpoint нет.
Impact plan имеет `enforcement="advisory"` и не означает, что текущий refresh пропустил шаги.

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

### GET `/api/git/diff`
Возвращает authoritative полный inventory Git-состояния workspace. Query `path`, `folder`,
`run_id` и `step_id` выбирают только preview; `files[]` и `fingerprint` всегда описывают весь
workspace scope, соответствующий будущему `git add -A`.

Identity содержит `branch`, `head_oid`, `base_ref`, `base_oid`. Для каждого файла возвращаются
normalized `status`, `index_status`, `worktree_status`, `path`, nullable `original_path`,
old/new mode, HEAD/index object identity, worktree SHA-256, additions/deletions и flags
`binary|unavailable`. Rename/copy не теряют source path.

`fingerprint` — SHA-256 канонического отсортированного manifest identity + полного inventory;
он меняется при смене branch/HEAD/base, status/path/mode/index blob или рабочего содержимого.

**200**
```json
{
  "scope": "full_workspace",
  "branch": "main",
  "head_oid": "abc123",
  "base_ref": "HEAD",
  "base_oid": "abc123",
  "fingerprint": "<sha256>",
  "empty": false,
  "files": [
    {
      "path": "reports/as-is/overview.md",
      "original_path": null,
      "status": "modified",
      "index_status": " ",
      "worktree_status": "M",
      "old_mode": "100644",
      "new_mode": "100644",
      "head_oid": "<blob>",
      "index_oid": "<blob>",
      "worktree_sha256": "<sha256>",
      "binary": false,
      "unavailable": false,
      "additions": 1,
      "deletions": 0
    }
  ],
  "selected_file": null,
  "hunks": []
}
```

### POST `/api/git/commit`
Коммитит текущие изменения в bound workspace repo с честной семантикой `git add -A`.

**Request**
```json
{
  "message": "chore: update ACP workspace artifacts",
  "expected_fingerprint": "<sha256>",
  "expected_head_oid": "abc123"
}
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
{
  "name": "proposal/beta-refresh",
  "expected_fingerprint": "<sha256>",
  "expected_source_branch": "main",
  "expected_base_ref": "HEAD",
  "expected_base_oid": "abc123",
  "expected_head_oid": "abc123"
}
```

**200**
```json
{ "ok": true, "branch": "proposal/beta-refresh" }
```

Обе mutation операции сериализованы. Несовпадение подтверждённых branch/HEAD/base/inventory
возвращает `409 stale_git_confirmation` до любой Git mutation. Пустой fingerprint возвращает
`400 git_confirmation_required`.

## 6) Q&A endpoints

### POST `/api/qa/runs`
Starts an async agent-backed Q&A run over existing workspace artifacts.

Target flow:
- API creates run with `pipeline="qa"` and current step `qa.ask`;
- orchestrator writes `reports/taskruns/<run_id>/qa/context-pack.json`;
- selected runtime provider/fake baseline writes `reports/taskruns/<run_id>/qa/qa-answer.json`;
- ACP validates `qa-answer.json` and returns the structured answer via `GET /api/qa/runs/<run_id>`.

Request:
```json
{ "question": "Who owns payments-service?" }
```

Unknown fields and malformed JSON are rejected.

**202**
```json
{
  "run_id": "run_20260403_001",
  "status": "queued"
}
```

**400**
- `invalid_request_body`
- `question_required`
- `qa_run_start_failed`

### GET `/api/qa/runs/<run_id>`
Returns Q&A run status and answer fields when ready.

**200**
```json
{
  "run_id": "run_20260403_001",
  "pipeline": "qa",
  "status": "succeeded",
  "started_at": "2026-04-03T12:00:00Z",
  "finished_at": "2026-04-03T12:00:02Z",
  "question": "Who owns payments-service?",
  "current_step": "qa.ask",
  "step_providers": {
    "qa": "codex-code"
  },
  "runtime_provider": "codex-code",
  "provider": "codex-code",
  "answer": "The available workspace evidence identifies Platform Architecture as owner.",
  "citations": [
    {
      "path": "reports/as-is/overview.md",
      "reason": "ownership evidence"
    }
  ],
  "unresolved": [],
  "confidence": 0.82,
  "generated_at": "2026-04-03T12:00:02Z",
  "pending_permissions": [],
  "warnings": [],
  "error_code": null,
  "error": null
}
```

While queued/running, answer fields are `null`/empty and provider identity comes from resolved `runtime.profile.steps.qa.provider`. Failed runs return the failed run envelope without parsing partial `qa-answer.json` content. Succeeded runs expose citations only after the runtime answer has passed schema validation and citation paths have been cross-checked against `context-pack.json` `documents[].path`.

**404**
- `qa_run_not_found`

**500**
- `qa_answer_unavailable`

### GET `/api/qa/runs?limit=<n>`
Lists Q&A runs only, sorted by `started_at desc`.

Parameters:
- `limit` optional, default `20`, max `100`

**200**
```json
{
  "items": [
    {
      "run_id": "run_20260403_001",
      "pipeline": "qa",
      "status": "succeeded",
      "question": "Who owns payments-service?",
      "runtime_provider": "codex-code",
      "provider": "codex-code"
    }
  ]
}
```

**400**
- `invalid_limit`

### POST `/api/qa/ask`
Legacy compatibility endpoint for deterministic read-only Q&A over workspace artifacts. Endpoint uses deterministic `internal/qa` service and does not call headless runtime providers, git helpers or pipeline runs. New UI Ask uses `/api/qa/runs`.

**Request**
```json
{ "question": "Who owns payments-service?" }
```

Unknown fields and malformed JSON are rejected.

**200**
```json
{
  "answer": "Workspace evidence matched 2 artifact(s): reports/coverage/summary.md, docs/imports/architecture-notes.md",
  "citations": [
    {
      "path": "reports/coverage/summary.md",
      "reason": "matched 2 keyword(s): owner, payments"
    }
  ],
  "unresolved": [],
  "confidence": 0.82
}
```

**400**
- `invalid_request_body`
- `question_required`

**500**
- `qa_failed`

## 7) Deterministic scope

Детерминированно сравниваемые артефакты (при одинаковом input и одинаковом наборе artifact fixtures):
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
- `reports/taskruns/<run_id>/qa/context-pack.json`
- `reports/taskruns/<run_id>/qa/qa-answer.json`
- `reports/taskruns/run-history.json`
- `reports/taskruns/logs/*`
- runtime run registry/status (`/api/pipeline/runs/*`)
- Q&A run registry/status (`/api/qa/runs/*`)

## 8) Boundary notes

`POST /api/qa/runs` / `GET /api/qa/runs/*` are the target UI Ask API surface. `POST /api/qa/ask` remains in the beta API surface as a read-only deterministic compatibility endpoint.
Каноническая фиксация runtime/Q&A boundary: `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Native SCM webhook listener/hosted control plane остаются вне MVP; required CI/CD surface — CLI batch mode, optional trusted API trigger — local/private deployment only.

## 9) Deployment boundary
- CLI deployment surface остаётся explicit: bootstrap через `acp init-workspace`, local service через `acp serve`, required CI/CD через `acp run`.
- bootstrap behavior: если workspace root не является git-репозиторием, ACP автоматически выполняет `git init` (без auto-commit/auto-push).
- `serve` startup работает в lenient mode: сервис стартует без блокирующего repo preflight; readiness diagnostics доступны через `POST /api/workspace/validate`.
- default runtime mode: `fake` (required deterministic CI surface), `headless` — opt-in.
- effective provider per step: `workspace.yaml.runtime.profile.steps.<step>.provider > CLI/env global provider > claude-code`.
- provider-specific command envs: `ACP_CLAUDE_CMD`, `ACP_QWEN_CMD`, `ACP_CODEX_CMD`; without env override ACP resolves `claude-code` to `claude` then legacy `claude-code`, `qwen-code` to `qwen`, and `codex-code` to `codex`.
- при `--runtime fake` provider value валидируется как config fallback, live provider command не выполняется, а runtime execution metadata пишет provider `fake`.
- GitHub/GitLab hooks/manual jobs для required CI/CD должны использовать CLI batch mode с deterministic defaults (`--runtime fake`).
- API-trigger не должен превращаться в hosted control plane в рамках MVP.
- Exact CLI flags, run log retention knobs, env precedence и local runbook examples намеренно не дублируются здесь; canonical source of truth — CLI help, `README.md` quickstart и профильные runbook docs.
