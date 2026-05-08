# Contributing

ACP — **AI-native** и **spec-first** проект.

## Start here
1) Прочитайте `AGENTS.md`
2) Прочитайте `README.md` и `docs/ARCHITECTURE.md`
3) Относитесь к `schemas/*` и `docs/spec/*` как к контрактам

## Local bootstrap
- Требуемый стек: Go exact version из `.go-version`, Node exact version из `.node-version`, npm 10.x, Git
- `go.mod` остаётся на language compatibility level `go 1.20`; не используйте это как разрешение собирать release устаревшим Go toolchain.
- Установите зависимости: `make bootstrap`
- Прогоните DoD-проверки:
  - `make contracts`
  - `make test`
  - `make lint`
  - `make build`

## ADRs
Пишите ADR, если:
- выбираете/добавляете крупную зависимость
- меняете схемы/контракты
- меняете семантику пайплайна или правила модели

См. `docs/adr/`.

## Canonical references
- Repo CI и required jobs: `README.md` + `docs/TESTING_STRATEGY.md`
- Планирование крупных slice: `docs/PLANS.md`

## Planning
Используйте `docs/PLANS.md` для многосоставных задач.

## Community and security

- Следуйте [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
- Уязвимости не публикуйте в issues; используйте [SECURITY.md](SECURITY.md).
- Пользовательские вопросы и bug reports оформляйте по [SUPPORT.md](SUPPORT.md).
- User-facing изменения отражайте в [CHANGELOG.md](CHANGELOG.md) и release notes.

## Review expectations

Pull request должен быть небольшим, воспроизводимо проверенным и не должен добавлять live network/provider dependency в required CI. Изменения `schemas/*` и `docs/spec/*` требуют синхронного обновления docs, examples/fixtures, валидаторов и тестов.
