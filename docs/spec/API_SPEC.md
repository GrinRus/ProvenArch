# Спецификация API (planned, v0)

Этот файл описывает стартовую поверхность HTTP API для local-first ACP сервера.

> В MVP все endpoints доступны только на localhost.

## 1) Конвенции
- Base path: `/api`
- Content-Type: `application/json`
- Ошибки: `{ "error": "..." }`
- В MVP аутентификации нет (local-only).

## 2) Endpoints (MVP базовые)

### GET `/api/health`
**Ответ 200**
```json
{ "status": "ok" }
```

### POST `/api/workspace/validate`
Проверяет, что workspace корректно задан, соответствует central `arch-workspace` конвенции MVP (Variant 2), и манифест валиден.

**Ответ 200**
```json
{ "ok": true, "results": { "warnings": [] } }
```

**Ответ 400**
```json
{ "ok": false, "error": "..." }
```

### GET `/api/artifacts?path=<relative>`
Читает файл из workspace по относительному пути (без выхода за корень workspace).

**Ответ 200**
- возвращает содержимое файла (content-type по расширению)

**Ответ 400/404**
```json
{ "error": "..." }
```

### POST `/api/pipeline/init`
Запускает init pipeline (шаги 0–4).

**Ответ 202 (planned)**
```json
{ "run_id": "run_...", "status": "started" }
```

### GET `/api/pipeline/runs/<run_id>` (planned)
Статус и лог прогона.

### GET `/api/pipeline/runs/<run_id>/artifacts` (planned)
Список сгенерированных артефактов.

## 3) Нефункциональные требования API (MVP)
- Детерминированность: одинаковые входные данные → одинаковые артефакты (насколько возможно)
- Безопасность путей: никакого чтения вне workspace
- Логи и ошибки должны быть “actionable”
