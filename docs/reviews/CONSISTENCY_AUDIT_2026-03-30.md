# Consistency Audit (2026-03-30)

Цель: убедиться, что верхнеуровневые документы **не противоречат**:
- `docs/spec/*`
- `schemas/taskresult.schema.json`

Проверяли: `docs/STAKEHOLDER_DOC.md`, `README.md`, `docs/spec/API_SPEC.md`, `docs/spec/PIPELINE_SPEC.md`, `docs/spec/MODEL_SPEC.md`, `docs/APPENDIX_SCHEMAS.md`.

---

## Найденные расхождения и исправления

### 1) `docs.imports_path`: абсолютный vs относительный путь
**Проблема:** в README пример был с абсолютным путём, а в `examples/workspace.example.yaml` — относительный (`./docs/imports`).  
**Риск:** пользователи копируют README и получают разные ожидания.

✅ **Исправление:**
- README пример приведён к `imports_path: ./docs/imports`.
- Добавлено пояснение, что путь может быть абсолютным, но **рекомендуется относительный** для переносимости.

---

### 2) Provenance/Confidence: разная формулировка
**Проблема:** README описывал `provenance: evidence[]` и отдельно `confidence`, что расходилось с модельной и schema‑логикой (confidence живёт внутри `provenance`).  
**Риск:** неверные ожидания структуры entity/edge.

✅ **Исправление:**
- README теперь явно фиксирует:
  - `provenance.kind`
  - `provenance.confidence`
  - `provenance.evidence[]`

---

### 3) TaskResult: “changeset+evidence” vs фактическая схема
**Проблема:** в README TaskResult упоминался упрощённо, без обязательных полей `meta/questions/coverage`.  
**Риск:** runtime/agent начнёт возвращать “почти то же самое”, но невалидное.

✅ **Исправление:**
- В README добавлен раздел **“Контракт TaskResult (обязателен)”** с перечнем обязательных ключей.
- В `docs/APPENDIX_SCHEMAS.md` добавлено человеко‑читаемое описание TaskResult.

---

### 4) Пайплайн: шаги 2 и 4 требуют “писать файлы”, но TaskResult не умеет write_file
**Проблема:** ожидались `reports/as-is/*` и `proposals/*`, но TaskResult схема не содержит операции, которая передаёт содержимое файлов (write_file).  
**Риск:** скрытая запись файлов рантаймом “в обход” orchestrator или неявные каналы.

✅ **Исправление (MVP-совместимое):**
- `docs/spec/PIPELINE_SPEC.md` уточнён:
  - Step 2 и Step 4 — **compiler/templates** шаги orchestrator’а (детерминированные, без runtime), чтобы не расширять контракт преждевременно.
  - Runtime‑генерация narrative/proposal‑контента переносится в Wave 1+ и требует отдельного решения по контракту.

---

### 5) Step IDs
**Проблема:** не было зафиксировано, какие строки использовать в `TaskResult.meta.step_id`.  
**Риск:** невозможность сравнивать прогоны и строить историю/диффы.

✅ **Исправление:**
- В `docs/spec/PIPELINE_SPEC.md` добавлена секция с canonical step ids.

---

## Оставшиеся открытые вопросы (не блокируют baseline, но блокируют реализацию MVP)

1) **Нужен ли write_file контракт в TaskResult?**  
   Варианты:
   - A) оставить Step 2/4 как compiler‑шаги (шаблоны, без LLM контента) — проще, быстрее
   - B) расширить TaskResult новой операцией `write_file` (path + content + provenance) — мощнее, но требует строгих guardrails
   - C) отдельный “artifact channel” вне TaskResult (например, tarball/dir output от runtime) — рискованнее в плане provenance и безопасности

2) **Стратегия стабильных IDs** (semantic IDs + aliases vs hashing).

3) **Как exactly хранить coverage/questions артефакты** (какие файлы, где лежат).

---

## Итог
После правок документы:
- не противоречат JSON Schema,
- согласованы по workspace.yaml,
- имеют явную позицию по “compiler steps” vs runtime steps, не вводя в заблуждение.
