# Спецификация workspace manifest (MVP v1 contract)

Этот документ фиксирует канонический контракт `workspace.yaml` для ACP MVP.

## 1) Source of truth

- человеко-читаемое описание: этот файл
- machine-readable контракт: `schemas/workspace.schema.json`

`workspace.yaml` описывает repo sources, manual analysis overrides, imports path и persisted runtime profile.
Layout `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` не конфигурируется через manifest и считается fixed MVP convention.

## 2) Top-level shape

Обязательные поля:
- `version`
- `repos`

Опциональные поля:
- `docs`
- `runtime`

Текущий MVP поддерживает только:
- `version: 1`

`repos[]` должен содержать как минимум одну запись.

## 3) `repos[]`

Каждая запись `repos[]` содержит:
- `name` required
- ровно одно из:
  - `path`
  - `git_url`
- `ref` optional
- `analysis` optional:
  - `include[]` glob-паттерны shard include
  - `exclude[]` glob-паттерны shard exclude
  - `role` optional enum: `backend|frontend|mixed|unknown`

Правила:
- `name` используется как stable repo scope identifier в persisted runtime execution metadata, warnings и evidence references
- имена репозиториев в одном workspace должны быть уникальными
- uniqueness `name` проверяется workspace validator-ом как semantic rule поверх JSON Schema
- `path` означает локально доступный checkout
- `git_url` означает remote source, который ACP разрешает через локальный `git` context пользователя/runner
- `ref` задаёт желаемую ветку/тег/sha, если repo source поддерживает checkout/fetch semantics
- `analysis.role` сохранён как legacy metadata (`backend|frontend|mixed|unknown`) и не влияет на runtime execution/sharding contract
- для `path`-source ACP не делает checkout (non-mutating policy в пользовательском репозитории)
- verify `ref` для `path`-source использует fallback-резолвинг: `<ref>` -> `origin/<ref>` -> `refs/remotes/origin/<ref>`
- при fallback/расхождении с `HEAD` validator возвращает warnings (не блокирующие), а не изменяет checkout

ACP в MVP не хранит отдельные git credentials и не вводит собственный credential plane.

## 4) `docs`

Поддерживаемое поле:
- `imports_path` optional

Default:
- `./docs/imports`

`imports_path` указывает только папку raw imports.
Пути `docs/rfcs/`, `docs/meetings/`, `docs/decisions/` считаются фиксированной частью workspace layout и не конфигурируются через manifest.

## 5) `runtime.profile`

Поддерживается optional блок `runtime.profile`:
- `runtime.profile.timeouts`
- `runtime.profile.execution`
- `runtime.profile.steps`

### 5.1) `runtime.profile.timeouts`

Поддерживаемые optional поля:
- `step_timeout_sec`
- `heartbeat_sec`
- `pipeline_timeout_sec`
- `pipeline_kill_grace_sec`
- `api_ready_timeout_sec`
- `api_init_timeout_sec`
- `ui_init_poll_timeout_sec`
- `ui_cancel_poll_timeout_sec`

Ограничения:
- каждое поле optional;
- если поле задано, значение должно быть целым `> 0`.

Назначение:
- persisted source-of-truth для timeout-профиля workspace;
- effective значения резолвятся с precedence: `env > workspace.yaml > defaults`.

Balanced defaults (если поле не задано и env override отсутствует):
- `1800/30/2400/30/60/120/900/420` (в порядке полей выше).

### 5.2) `runtime.profile.execution`

Поддерживаемые optional поля:
- `strategy`: `sequential|parallel`
- `max_parallel_tasks`: integer `> 0`
- `failure_policy`: `fail_fast|best_effort`
- `shard_discovery.mode`: `heuristics|semantic`

Default values:
- `strategy=sequential`
- `max_parallel_tasks=1`
- `failure_policy=best_effort`
- `shard_discovery.mode=heuristics`

Effective precedence:
- `CLI > env > workspace.yaml > defaults`

`shard_discovery.mode` поведение:
- `heuristics`: детерминированное structural sharding с полным покрытием repo и неперекрывающимися `path_scopes`.
- `semantic`: metadata-only mode; использует тот же shard-plan, но дополняет его graph/debug metadata и не меняет count/boundaries shard-ов.
- runtime execution всегда использует все repo scopes из `workspace.yaml`; frontend/backend filtering не входит в execution contract этого slice.

### 5.3) `runtime.profile.steps`

Поддерживаемые optional поля:
- `step0_constitution.provider`
- `step1_collect.provider`
- `step2_as_is.provider`
- `step3_findings.provider`
- `step4_proposals.provider`

Допустимые значения provider:
- `claude-code`
- `qwen-code`

Назначение:
- step-scoped override для headless provider resolution;
- позволяет смешивать providers между соседними шагами одного run, не меняя global execution knobs.

Precedence effective provider для конкретного шага:
- `runtime.profile.steps.<step>.provider`
- CLI `--runtime-provider` или `ACP_RUNTIME_PROVIDER`
- default `claude-code`

Ограничения:
- шаг не auto-fallback-ится на другой provider при недоступности выбранного provider;
- per-step overrides в этом slice не вводят отдельные `max_parallel/failure_policy/shard_discovery` knobs;
- canonical workspace outputs не пишутся runtime-шагом напрямую, даже если provider override задан.

## 6) Пример

```yaml
version: 1
repos:
  - name: payments-service
    path: /absolute/path/to/payments-service
    analysis:
      include:
        - services/**
      exclude:
        - services/legacy/**
  - name: users-service
    git_url: https://gitlab.example.com/platform/users-service.git
    ref: main
docs:
  imports_path: ./docs/imports
runtime:
  profile:
    timeouts:
      step_timeout_sec: 1800
      heartbeat_sec: 30
      pipeline_timeout_sec: 2400
      pipeline_kill_grace_sec: 30
      api_ready_timeout_sec: 60
      api_init_timeout_sec: 120
      ui_init_poll_timeout_sec: 900
      ui_cancel_poll_timeout_sec: 420
    execution:
      strategy: parallel
      max_parallel_tasks: 4
      failure_policy: best_effort
      shard_discovery:
        mode: heuristics
    steps:
      step0_constitution:
        provider: claude-code
      step2_as_is:
        provider: qwen-code
      step4_proposals:
        provider: claude-code
```

## 7) Validation expectations

Manifest считается невалидным, если:
- отсутствует `version`
- `version` не равен `1`
- отсутствует `repos[]`
- запись repo не содержит `name`
- запись repo содержит одновременно `path` и `git_url`
- запись repo не содержит ни `path`, ни `git_url`
- `analysis.include[]`/`analysis.exclude[]` содержит пустые элементы
- `analysis.role` не в `backend|frontend|mixed|unknown`
- любое значение `runtime.profile.timeouts.* <= 0`
- `runtime.profile.execution.max_parallel_tasks <= 0`
- `runtime.profile.execution.strategy` не в `sequential|parallel`
- `runtime.profile.execution.failure_policy` не в `fail_fast|best_effort`
- `runtime.profile.execution.shard_discovery.mode` не в `heuristics|semantic`
- `runtime.profile.steps.*.provider` не в `claude-code|qwen-code|codex-code`
- manifest пытается использовать legacy path `runtime.timeouts` (breaking change, intentional)
- manifest пытается конфигурировать workspace layout beyond supported fields

Дополнительные runtime diagnostics для `POST /api/workspace/validate`:
- `workspace.repo.ref.invalid` (error): `ref` не резолвится ни локально, ни через origin-tracking fallback
- `workspace.repo.ref.resolved_via_remote` (warning): `ref` был разрешён через remote-tracking ref
- `workspace.repo.ref.head_mismatch` (warning): `ref` и текущий `HEAD` указывают на разные коммиты
