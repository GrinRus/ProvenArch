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
- минимальные merge-конфликты
- читаемые diffs
- проще поддерживать детерминированность

## 2) Core types (MVP)

### Entities
- `service`
- `api.http`
- `api.grpc`
- `event.topic`
- `datastore`
- `external.system`
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

MVP-практика:
- внешние интеграции желательно представлять отдельными `external.system` entities;
- `owner_team_id` должен ссылаться на существующий `team.<slug>`;
- CI/CD topology пока фиксируется в `reports/as-is/ci-cd.md` и `reports/coverage/*`, а не как отдельный core type модели.

### Read-only Architecture projection

`GET /api/architecture` не вводит вторую модель. Он проецирует только validator-promoted
entity-per-file YAML и immutable semantic snapshot source run в четыре read-only уровня:
Context, Container, Component и Advanced Code. Последний является доступной детализацией
существующих API entities, а не отдельным каноническим code graph. Узлы/связи сохраняют stable ID,
owner, repository evidence, confidence и related finding/question IDs. Если валидированной
детализации нет, API возвращает unavailable reason, а UI не синтезирует C4 элементы из имён файлов.

Семантическое сравнение между promoted generations использует stable IDs сущностей, связей и
findings. Coverage gaps получают детерминированную identity из нормализованного текста только для
comparison presentation; это не новый persisted model ID и не изменяет `schemas/*`.

## 4) Формат evidence

Каждый элемент `evidence[]`:
- `repo`
- `path`
- optional `ref`
- optional `lines` как объект `{start, end}`
- optional `excerpt_hash` / `excerpt`

Для `provenance.kind = observation` evidence должен быть непустым.

## 5) Canonical stable IDs (MVP)

### Entity ID patterns
- `service`: `svc.<slug>`
- `team`: `team.<slug>`
- `repo`: `repo.<slug>`
- `external.system`: `ext.<slug>`
- `datastore`: `db.<engine>.<slug>`
- `api.http`: `api.http.<service-slug>.<method>.<path-slug>`
- `api.grpc`: `api.grpc.<service-slug>.<service>.<method>`
- `event.topic`: `topic.<slug>`

### Edge ID pattern
- `edge.<from>.<type>.<to>`

`from` и `to` в edge ID используют уже нормализованные canonical IDs сущностей.

## 6) Normalization rules

### General slug rules
- lowercase ASCII only
- separator: `-`
- удаляем ведущие/хвостовые separator symbols
- подряд идущие не-alnum символы схлопываются в один `-`
- для `service-slug` используется slug service name без префикса `svc.`

### Service / team / repo / external / datastore slug
- основа slug строится из human-readable canonical name или стабильного anchor
- `db.<engine>.<slug>` использует отдельный engine slug, например `db.postgres.payments`

### HTTP path slug
Для `api.http.<service-slug>.<method>.<path-slug>`:
- HTTP method → lowercase (`get`, `post`, `put`, `delete`, ...)
- путь `/` нормализуется в `root`
- сегменты пути нормализуются по одному
- static segment `/payments` → `payments`
- parameter segment `{id}` или `:id` → `by-id`
- итоговые segments join-ятся через `-`

Примеры:
- `GET /payments` → `api.http.payments.get.payments`
- `GET /payments/{id}` → `api.http.payments.get.payments-by-id`
- `POST /` → `api.http.payments.post.root`

### gRPC service/method slug
Для `api.grpc.<service-slug>.<service>.<method>`:
- proto service и method нормализуются теми же general slug rules
- package prefix в canonical ID не включается, если он не нужен для устранения коллизии

### Collision rule
- если canonical slug уже занят другим entity того же logical pattern, добавляется suffix `.repo-<repo-slug>`
- collision resolution должна быть детерминированной и опираться на repo source, а не на случайный порядок обхода

## 7) Rename / move policy

- canonical ID автоматически не меняется при rename/move
- для миграций используются `aliases[]`
- смена canonical ID требует явной human/manual migration
- orchestrator/runtime не должны silently re-key существующую сущность

## 8) Пример entity YAML

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

## 9) Пример edge YAML

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
