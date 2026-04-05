# Спецификация workspace manifest (MVP v0)

Этот документ фиксирует канонический контракт `workspace.yaml` для ACP MVP.

## 1) Source of truth

- человеко-читаемое описание: этот файл
- machine-readable контракт: `schemas/workspace.schema.json`

`workspace.yaml` описывает только repo sources и imports path.
Layout `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` не конфигурируется через manifest и считается fixed MVP convention.

## 2) Top-level shape

Обязательные поля:
- `version`
- `repos`

Опциональные поля:
- `docs`

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

ACP в MVP не хранит отдельные git credentials и не вводит собственный credential plane.

## 4) `docs`

Поддерживаемое поле:
- `imports_path` optional

Default:
- `./docs/imports`

`imports_path` указывает только папку raw imports.
Пути `docs/rfcs/`, `docs/meetings/`, `docs/decisions/` считаются фиксированной частью workspace layout и не конфигурируются через manifest.

## 5) Пример

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
```

## 6) Validation expectations

Manifest считается невалидным, если:
- отсутствует `version`
- `version` не равен `1`
- отсутствует `repos[]`
- запись repo не содержит `name`
- запись repo содержит одновременно `path` и `git_url`
- запись repo не содержит ни `path`, ни `git_url`
- manifest пытается конфигурировать workspace layout beyond supported fields
