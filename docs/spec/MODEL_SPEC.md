# Спецификация модели (MVP v0)

Этот документ фиксирует минимальную каноническую модель ACP для MVP.

## 1) Формат и размещение

Каноническая модель хранится как **entity-per-file YAML**:

```text
model/
  entities/
  edges/
```

Почему так:
- минимальные merge-конфликты,
- читаемые diffs,
- проще поддерживать детерминированность.

## 2) Core types (MVP)

### Entities
- `service`
- `api.http`
- `api.grpc`
- `event.topic`
- `datastore`
- `repo` (optional)
- `team` (optional)

### Edges
- `calls`
- `publishes`
- `subscribes`
- `reads`
- `writes`
- `exposes`

## 3) Обязательные поля

### Entity
- `id`
- `type`
- `name`
- `provenance`

### Edge
- `id`
- `type`
- `from`
- `to`
- `provenance`

`provenance`:
- `kind`: `observation | inference | assertion`
- `confidence`: `0..1`
- `evidence[]`: список ссылок на исходные артефакты

Рекомендуемые поля:
- `aliases[]`
- `tags[]`
- `attributes` (map)
- `owner_team_id` (для entity)

## 4) Формат evidence

Каждый элемент `evidence[]`:
- `repo`
- `path`
- optional `ref`
- optional `lines` как объект `{start, end}`
- optional `excerpt_hash` / `excerpt`

Для `provenance.kind = observation` evidence должен быть непустым.

## 5) Пример entity YAML

```yaml
id: svc.payments
type: service
name: Payments Service
aliases:
  - payment-api
tags:
  - critical
attributes:
  language: go
owner_team_id: team.payments
provenance:
  kind: observation
  confidence: 0.92
  evidence:
    - repo: payments-service
      ref: main@abc123
      path: cmd/payments/main.go
      lines:
        start: 1
        end: 40
      excerpt_hash: sha256:deadbeef
```

## 6) Пример edge YAML

```yaml
id: edge.svc.payments.calls.svc.users
type: calls
from: svc.payments
to: svc.users
attributes:
  protocol: http
provenance:
  kind: inference
  confidence: 0.7
  evidence:
    - repo: payments-service
      path: internal/clients/users/client.go
      lines:
        start: 10
        end: 44
```

## 7) Стратегия stable IDs (baseline)

Используем гибрид:
- человеко-читаемые canonical IDs (`svc.<name>`, `api.<name>.<protocol>`, `db.<name>.<engine>`),
- `aliases[]` для миграций rename/move,
- предпочтение evidence-backed anchors (entrypoints, module/package path, ingress host, image name).
