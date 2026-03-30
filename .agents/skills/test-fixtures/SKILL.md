---
name: acp-test-fixtures
description: Используй при изменении логики извлечения/модели. Добавляет/обновляет фикстуры и golden outputs для защиты от регрессий.
---

## Инструкции
1) Создай/обнови fixture workspace под `fixtures/`.
2) Зафиксируй golden outputs (model + reports).
3) Добавь тест, сравнивающий actual с golden.
4) Тесты должны запускаться локально без сети.
