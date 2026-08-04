# AGENTS.md (Codex)

Короткие устойчивые правила для работы в репозитории. Детальные процедуры живут в specs, skills и
runbooks, а не дублируются здесь.

## Mission (MVP)

Собрать ACP как local-first инструмент:
- runtime анализа: headless multi-provider (`claude-code` default, `qwen-code` optional,
  `codex-code` release peer) + deterministic `fake` baseline;
- реализация: Go backend/orchestrator + embedded React UI;
- результат: Git-версионируемые workspace-файлы с entity-per-file моделью.

## Source routing

- Начать с `README.md`, затем читать только относящиеся к текущему slice разделы
  `docs/ARCHITECTURE.md` и `docs/spec/*`; для pipeline/runtime обязателен
  `docs/spec/PIPELINE_SPEC.md`.
- При конфликте источников правды использовать приоритет:
  `schemas/*` -> `docs/spec/*` -> `README.md`/`docs/ARCHITECTURE.md` ->
  `docs/STAKEHOLDER_DOC.md`.
- Для нетривиальной задачи вести ExecPlan в `docs/PLANS.md` с целью, non-goals, acceptance и
  progress log.
- Использовать релевантные `.agents/skills/*`: `acp-spec-first-slice` для новой фичи,
  `acp-schema-guardian` для контрактов, `acp-test-fixtures` для core/model logic,
  `acp-docs-sync` для behavior docs и `acp-e2e-live-gate` для live/release gate.

## Working contract

- В начале зафиксировать текущий слой работы: research, design, implementation, review или release;
  не переходить между слоями молча.
- Выбирать минимальный reviewable slice из `docs/BACKLOG.md` и релевантной spec. До правок
  сформулировать проверяемый результат, ограничения и stop condition.
- Для answer/review/diagnose: исследовать и сообщить результат, не менять код без запроса. Для
  change/build/fix: самостоятельно внести локальные in-scope изменения и выполнить безопасную
  проверку. Запрашивать подтверждение для release/push/external writes, destructive действий,
  новых production dependencies и существенного расширения scope.
- Сохранять пользовательские изменения, не трогать unrelated файлы и делать Git-friendly diff.
- Не объявлять работу завершённой без фактической проверки; blocker описывать вместе с evidence и
  минимальным следующим действием.

## Guidance for current reasoning models

- Давать модели цель, доменный контекст, hard constraints, acceptance criteria и границы
  автономности; не предписывать каждый промежуточный шаг без необходимости.
- Каждое правило формулировать один раз. Удалять повторяющийся process/style scaffolding, но
  сохранять продуктовые инварианты, форматы результата и review requirements.
- Сохранять пользовательские значения и существующее поведение. Существенную неоднозначность в
  контракте или необратимом результате выносить на уточнение; в локальном обратимом slice делать
  явно отмеченное разумное допущение.
- Для длинной работы держать актуальными текущий слой, зависимости, критерий остановки и краткий
  handoff. Параллелить только независимые задачи с обязательным финальным синтезом.
- Не закреплять модель или reasoning effort в `AGENTS.md`. Изменение `codex-code` model/reasoning
  defaults — отдельный измеримый slice через runtime config, tests и live runbook. Не включать Pro,
  multi-agent, persisted reasoning, explicit caching или programmatic tool calling только потому,
  что новая модель это поддерживает.

## Product invariants

- Нет hosted режима и security/compliance enforcement в MVP.
- Не расширять список headless providers за `claude-code`, `qwen-code`, `codex-code` без отдельного
  slice.
- Не выдумывать форматы данных и не писать в анализируемые пользовательские репозитории.
- При изменении схемы/контракта синхронизировать specs, validators, tests/fixtures, examples,
  `docs/APPENDIX_SCHEMAS.md` и ADR rationale. Для `workspace.yaml` также синхронизировать
  `docs/spec/WORKSPACE_SPEC.md` и `schemas/workspace.schema.json`.
- При изменении core поведения обновлять tests/fixtures и behavior docs. При изменении testing
  baseline синхронизировать `docs/TESTING_STRATEGY.md`, fixtures и golden outputs.
- Required CI должен оставаться deterministic и без live network dependencies.

## Validation and release

- Во время работы запускать самый узкий релевантный check; для завершённого slice выполнить полный
  DoD: `make contracts`, `make test`, `make lint`, `make build`.
- Для pre-release live gate использовать только skill `acp-e2e-live-gate` и
  `docs/RELEASE_LIVE_E2E_RUNBOOK.md`; canonical harness — `scripts/full-run-batch-matrix.sh` без
  wrapper. Full live matrix остаётся manual trusted-machine gate, не required CI merge gate.
- Не менять canonical release matrices или curated `repos_file` ради обхода ограничений машины;
  переносить gate на подходящий trusted host.
