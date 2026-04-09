# Политика baseline

Этот репозиторий следует **baseline-first** подходу:
- сначала спецификации и контракты,
- затем реализация, синхронная с контрактами.

## Правила
1) Любое изменение в `schemas/` или `docs/spec/` сопровождается коротким rationale в PR.
2) MVP scope фиксирован: local-first как основной режим + headless multi-provider runtime (`claude-code` default, `qwen-code` optional) при обязательном deterministic baseline `--runtime fake` для required CI.
3) Hosted/security/compliance фичи не добавляются до Wave 1 без явного одобрения.
4) Конвенция хранения MVP фиксирована: central `arch-workspace` repo (Variant 2) как единственный канонический формат active docs.
5) `workspace.yaml` имеет отдельный schema-contract (`schemas/workspace.schema.json`) и не конфигурирует workspace layout beyond repo sources и imports path.
6) Repo sources в MVP задаются через `path` или `git_url`, но git access всегда идёт через локальный `git` контекст пользователя/runner; GitHub/GitLab hooks и manual pipeline/job triggers допустимы, если не появляется hosted control plane.
7) При недостатке evidence система должна фиксировать gaps через coverage/questions/findings, а не выдумывать факты; canonical MVP shape для runtime — top-level `questions` и `coverage`.
8) MVP extraction strategy не ограничивается фиксированным whitelist языков/стэков; используются headless providers (`claude-code|qwen-code`) + baseline prompt/skill bundle, а при нехватке evidence фиксируются unknowns.
9) В MVP обязателен встроенный baseline bundle agents/skills/prompts, versioned в workspace.
10) Q&A API follow-up фиксируется как read-only `POST /api/qa/ask` без mutation surface.

## Policy для generated артефактов в репозитории

Следующие generated артефакты намеренно tracked в git и считаются частью baseline/release surface:
- `internal/api/ui_dist/*` (embed UI assets, используемые `acp serve`)
- `fixtures/scenarios/*/golden/readable/*` (human-readable deterministic export для review-диффов)

Rationale:
- фиксированная локальная воспроизводимость UI/fixtures без дополнительных шагов в ревью;
- предсказуемые diffs при изменениях deterministic surface;
- прозрачная верификация CI gates для smoke/golden контуров.

Правила обновления:
- UI embed артефакты обновляются через `make build`.
- Readable golden обновляется только контролируемым запуском golden refresh flow и коммитится вместе с изменениями baseline логики.
