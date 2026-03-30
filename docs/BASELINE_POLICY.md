# Политика baseline

Этот репозиторий следует **baseline-first** подходу:
- сначала спецификации и контракты,
- затем реализация, синхронная с контрактами.

## Правила
1) Любое изменение в `schemas/` или `docs/spec/` сопровождается коротким rationale в PR.
2) MVP scope фиксирован: local-first + Claude Code headless как единственный runtime.
3) Hosted/security/compliance фичи не добавляются до Wave 1 без явного одобрения.
4) Конвенция хранения MVP фиксирована: central `arch-workspace` repo (Variant 2) как единственный канонический формат active docs.
