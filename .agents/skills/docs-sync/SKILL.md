---
name: acp-docs-sync
description: Используй, когда изменения кода требуют обновления документации. Держит README/docs/ARCHITECTURE/docs/STAKEHOLDER_DOC в согласованном виде.
---

## Инструкции
1) Определи, какие документы затронуты.
2) Обнови docs так, чтобы они соответствовали поведению.
3) Убедись, что нет противоречий между README/docs/ARCHITECTURE/docs/STAKEHOLDER_DOC и схемами.

**Важно:** repository entrypoint `README.md` ведётся на EN. Детальные stakeholder/engineering
документы пока ведутся преимущественно на RU; localized variants используют явный language suffix
(`README.ru.md`, `*.en.md` и т.п.).
