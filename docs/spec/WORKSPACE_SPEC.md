# Спецификация workspace manifest (MVP v0)

Этот документ фиксирует канонический контракт `workspace.yaml` для ACP MVP.

## 1) Source of truth

- человеко-читаемое описание: этот файл
- machine-readable контракт: `schemas/workspace.schema.json`

`workspace.yaml` описывает repo sources, imports path и persisted runtime timeouts.
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

Правила:
- `name` используется как stable repo scope identifier в `TaskResult.meta.repo_scopes[]`, warnings и evidence references
- имена репозиториев в одном workspace должны быть уникальными
- uniqueness `name` проверяется workspace validator-ом как semantic rule поверх JSON Schema
- `path` означает локально доступный checkout
- `git_url` означает remote source, который ACP разрешает через локальный `git` context пользователя/runner
- `ref` задаёт желаемую ветку/тег/sha, если repo source поддерживает checkout/fetch semantics
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

## 5) `runtime.timeouts`

Поддерживаемый optional блок:
- `runtime.timeouts.step_timeout_sec`
- `runtime.timeouts.heartbeat_sec`
- `runtime.timeouts.pipeline_timeout_sec`
- `runtime.timeouts.pipeline_kill_grace_sec`
- `runtime.timeouts.api_ready_timeout_sec`
- `runtime.timeouts.api_init_timeout_sec`
- `runtime.timeouts.ui_init_poll_timeout_sec`
- `runtime.timeouts.ui_cancel_poll_timeout_sec`

Ограничения:
- каждое поле optional;
- если поле задано, значение должно быть целым `> 0`.

Назначение:
- persisted source-of-truth для timeout-профиля workspace;
- effective значения резолвятся с precedence: `env > workspace.yaml > defaults`.

Balanced defaults (если поле не задано и env override отсутствует):
- `1800/30/2400/30/60/120/900/420` (в порядке полей выше).

## 6) Пример

```yaml
version: 1
repos:
  - name: payments-service
    path: /absolute/path/to/payments-service
  - name: users-service
    git_url: https://gitlab.example.com/platform/users-service.git
    ref: main
docs:
  imports_path: ./docs/imports
runtime:
  timeouts:
    step_timeout_sec: 1800
    heartbeat_sec: 30
    pipeline_timeout_sec: 2400
    pipeline_kill_grace_sec: 30
    api_ready_timeout_sec: 60
    api_init_timeout_sec: 120
    ui_init_poll_timeout_sec: 900
    ui_cancel_poll_timeout_sec: 420
```

## 7) Validation expectations

Manifest считается невалидным, если:
- отсутствует `version`
- `version` не равен `1`
- отсутствует `repos[]`
- запись repo не содержит `name`
- запись repo содержит одновременно `path` и `git_url`
- запись repo не содержит ни `path`, ни `git_url`
- любое значение `runtime.timeouts.* <= 0`
- manifest пытается конфигурировать workspace layout beyond supported fields

Дополнительные runtime diagnostics для `POST /api/workspace/validate`:
- `workspace.repo.ref.invalid` (error): `ref` не резолвится ни локально, ни через origin-tracking fallback
- `workspace.repo.ref.resolved_via_remote` (warning): `ref` был разрешён через remote-tracking ref
- `workspace.repo.ref.head_mismatch` (warning): `ref` и текущий `HEAD` указывают на разные коммиты
