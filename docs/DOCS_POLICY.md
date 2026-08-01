# Политика документации

## Язык
- Repository entrypoint `README.md`: **EN** для open-source аудитории.
- Детальные stakeholder/engineering документы пока сохраняют primary язык **RU**.
- Для локализованных версий используем явный language suffix (`README.ru.md`, `*.en.md` и т.п.).
- Языковую миграцию документа фиксируем с явным rationale в PR/commit; она не означает
  автоматический перевод остальных canonical документов.

## Не терять контент
- Если документ становится слишком большим — выносим детали в `docs/spec/*` или отдельные файлы,
  а в исходном документе оставляем ссылки.
- Любые крупные сокращения фиксируем через явный комментарий или ссылку на commit в Git-истории.

## Версионирование
- Stakeholder doc: версия в заголовке (`vX.Y`) и дата ревизии.
- README: должен ссылаться на canonical stakeholder matrix в `docs/STAKEHOLDER_DOC.md`.
- При изменении реализации/runtime boundary обновляются синхронно:
  - `README.md`
  - `docs/ARCHITECTURE.md`
  - `docs/spec/API_SPEC.md`
  - `docs/spec/PIPELINE_SPEC.md`
  - `docs/PLANS.md` (лог изменений)
