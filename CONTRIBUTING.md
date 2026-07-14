# Contributing

ACP — **AI-native** и **spec-first** проект.

## Start here
1) Прочитайте `AGENTS.md`
2) Прочитайте `README.md` и `docs/ARCHITECTURE.md`
3) Относитесь к `schemas/*` и `docs/spec/*` как к контрактам

## Local bootstrap
- Требуемый стек: Go exact version из `.go-version`, Node exact version из `.node-version`,
  Python exact version из `.python-version`, npm 10.x, Git.
- Репозиторные Python tests/scripts запускайте через `./scripts/run-python.sh`; wrapper fail-fast
  остановит suite до запуска, если найденный `python3` не совпадает с `.python-version`.
- `go.mod` остаётся на language compatibility level `go 1.20`; не используйте это как разрешение собирать release устаревшим Go toolchain.
- Установите зависимости: `make bootstrap`
- Contract validation uses a separate locked npm toolchain in `tools/contracts`. `make contracts`
  runs `npm ci` for that package and then validates schemas/examples with the repo-local
  `ajv-cli`, `ajv-formats` and `js-yaml` versions from `tools/contracts/package-lock.json`.
  Tool version updates must include an explicit `tools/contracts/package.json` and
  `tools/contracts/package-lock.json` diff.
- Прогоните DoD-проверки:
  - `make contracts`
  - `make test`
  - `make lint`
  - `make build`
- Для изменений в `ui/src`, `ui/vite.config.ts` или embedded UI assets дополнительно проверьте:
  - `make verify-ui-determinism`
  - `make verify-ui-dist`

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
UI source changes must include the deterministic `internal/api/ui_dist` refresh produced by `make build`; stale embedded bundles are rejected by `make verify-ui-dist`.
