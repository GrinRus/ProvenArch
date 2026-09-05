# Агентская разработка ProvenArch

Этот guide помогает выбрать код, контракт и проверку для конкретной задачи. Обязательные границы
работы находятся в [AGENTS.md](../AGENTS.md), подготовка среды — в
[CONTRIBUTING.md](../CONTRIBUTING.md). Он не задаёт текущий статус эпиков: активную работу и
handoff ведёт [PLANS.md](PLANS.md), статус продукта — canonical matrix в
[STAKEHOLDER_DOC.md](STAKEHOLDER_DOC.md).

## Карта изменений

Пути к коду и команды ниже считаются от корня репозитория. Выбрать затронутые пакеты и существующие
тесты; строка карты не требует запуска всех перечисленных suites при каждой правке.

| Изменение | Код и источники контракта | Первая проверка |
| --- | --- | --- |
| Workspace, manifest, bootstrap, runtime config | `internal/workspace`, `internal/runtimeprofile`, `internal/doctor`; [WORKSPACE_SPEC](spec/WORKSPACE_SPEC.md), [workspace schema](../schemas/workspace.schema.json) | `./scripts/run-go.sh test ./internal/workspace ./internal/runtimeprofile ./internal/doctor`; при schema diff — `make contracts-check` |
| Task/Attempt, admission, API, restart | `internal/tasks`, `internal/api`, `internal/orchestrator`; [TASK_SPEC](spec/TASK_SPEC.md), [API_SPEC](spec/API_SPEC.md), [Task ADRs](adr/README.md) | Соответствующий package test; concurrency/lifecycle change — focused `-race` test |
| Pipeline, provider execution, recovery, prompts | `internal/orchestrator`, `internal/runtime`, `internal/runtimedrafts`; [PIPELINE_SPEC](spec/PIPELINE_SPEC.md), runtime sections [ARCHITECTURE](ARCHITECTURE.md) | `./scripts/run-go.sh test ./internal/runtime/promptcontract ./internal/runtime/steppolicy`; затем затронутые engine/adapter/orchestrator tests |
| Artifact contracts, validation, promotion, model | `internal/contracts`, `internal/validation`, `internal/artifactaudit`, `internal/artifactquality`, `internal/model`, `internal/reports`; [MODEL_SPEC](spec/MODEL_SPEC.md), [PIPELINE_SPEC](spec/PIPELINE_SPEC.md), [APPENDIX_SCHEMAS](APPENDIX_SCHEMAS.md) | `make contracts-check` и package tests; при изменении materialized outputs — scenario/golden и `make verify-readable-fixtures` |
| Refresh, reuse, source/evidence paths | `internal/refreshplan`, `internal/orchestrator`, `internal/evidence`, `internal/pathscope`; refresh/evidence sections [PIPELINE_SPEC](spec/PIPELINE_SPEC.md) | Existing refresh preservation, baseline integrity, path/identity tests; fault/race tests при изменении persistence/lifecycle |
| UI flow, routes, state, artifact viewer | `ui/src`, `internal/api`; [TASK_SPEC](spec/TASK_SPEC.md), [API_SPEC](spec/API_SPEC.md), [UI_TASK_FIRST_PRODUCT_DESIGN](UI_TASK_FIRST_PRODUCT_DESIGN.md), [migration plan](UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md) | Targeted Vitest + typecheck; для user flow — rendered mock E2E, keyboard/focus, loading/error/stale states; embedded checks ниже |
| CLI, tooling, deterministic CI | `cmd/acp`, `scripts`, `Makefile`, `.github/workflows`; [README](../README.md), [CONTRIBUTING](../CONTRIBUTING.md), [TESTING_STRATEGY](TESTING_STRATEGY.md) | CLI/package tests или соответствующий Python/shell test; live commands не входят в required CI |
| Agent guidance и documentation | `AGENTS.md`, `.agents/skills`, этот guide; [DOCS_POLICY](DOCS_POLICY.md), [PLANS](PLANS.md) | `make verify-agent-guidance`, `./scripts/run-go.sh test ./internal/docsync` |

По изменённой границе review должен установить конкретный результат: Task/Attempt сохраняет exact
identity после restart/retry; explicit stale path не читает другую evidence authority; promotion не
обходит validation; refresh переиспользует только проверенный baseline; UI показывает missing/unknown
вместо выведенного из diagnostics успеха. Соответствующие specs выше содержат полные инварианты.

## Development skills и runtime prompt surfaces

| Поверхность | Назначение и владелец |
| --- | --- |
| `.agents/skills/*/SKILL.md` | Навыки SWE-агента, который меняет этот репозиторий. Не исполняются ACP как analysis prompts. Optional `agents/openai.yaml` задаёт UI metadata; его отсутствие не ошибка. |
| `internal/workspace/baseline.go` | Embedded defaults для создаваемого architecture workspace: prompt packs, skill manifests, reference prompts и templates. |
| Workspace `skills/prompt-packs/*.md` | Editable live-consumed content layer. Его использование и merge order задаёт [PIPELINE_SPEC](spec/PIPELINE_SPEC.md); `step2.asis_docs` не имеет отдельного editable pack. |
| Workspace `skills/*/prompts/*.md` | Reference-only материалы; правка этих файлов сама по себе не меняет live prompt. Inventory и usage labels задаёт `internal/workspace/baseline_bundle_manifest.go`. |
| Workspace `skills/subagents.yaml` | Bundle agent/skill references, которые проверяет `internal/workspace/bundle.go`. Наличие роли в bundle не даёт разрешения обходить step policy или запускать дополнительные provider agents. |
| `internal/runtime/steppolicy`, `internal/runtime/promptcontract`, `internal/runtime/providercommon` | Enforced policy, first-action/artifact/recovery contracts и shared execution policy; adapters добавляют provider transport/policy. Editable packs не могут ослаблять эти инварианты. |

Baseline seeding создаёт editable files только при отсутствии: `writeFileIfMissing` сохраняет
пользовательские изменения. Generated `skills/bundle-manifest.json` обновляется при изменении
inventory. Поэтому правка embedded prompt не мигрирует автоматически уже существующий workspace.
При изменении baseline проверить и fresh workspace, и сохранение user-edited файла через
`internal/workspace/baseline_test.go`.

Правки runtime prompts и model/reasoning defaults — продуктовые изменения. Их цель, changed prompt
surface и наблюдаемый результат должны быть сформулированы отдельно от уборки development guidance.
Использовать bounded fixture/contract/adapter tests; если требуется доказать live behavior, заранее
определить diagnostic или release profile и пройти соответствующий gate. Не переносить текст
отдельного live incident в универсальные инструкции и не снижать validation ради нового prompt.

## Локальная проверка

`make preflight` проверяет готовность точных toolchains и зависимостей без установки;
`make bootstrap` устанавливает locked dependencies. Подробности и prerequisites принадлежат
[CONTRIBUTING](../CONTRIBUTING.md), а не каждому skill. Repository Go/Node/Python команды запускать
через `scripts/run-go.sh`, `scripts/run-npm.sh`, `scripts/run-python.sh`.

После установки зависимостей `make contracts-check` проверяет schemas/examples без повторной
установки contract tools. `make contracts` сохраняет совместимый install-and-check путь.
Для изменения текста guidance достаточно relevant structural/docs checks; полный implementation DoD
и required CI определены в [AGENTS](../AGENTS.md) и [TESTING_STRATEGY](TESTING_STRATEGY.md).

UI changes требуют обновлённого tracked `internal/api/ui_dist`, которое генерирует `make build`.
`make verify-ui-determinism UI_SOURCE=WORKTREE` проверяет текущие source edits;
`UI_SOURCE=HEAD` — committed source. `make verify-ui-dist` проверяет соответствие source и embedded
bundle, не исправляя его. Browser checks выбирать по затронутому flow; fake/mock execution отделено
от paid/live providers.

Сценарии и readable golden policy описаны в [fixtures/README](../fixtures/README.md) и
[scenario README](../fixtures/scenarios/README.md). Live evidence проверяется только по
[live skill](../.agents/skills/e2e-live-gate/SKILL.md) и [runbook](RELEASE_LIVE_E2E_RUNBOOK.md).

## План и handoff

Сначала найти существующий active plan и его ownership/dependencies. Backlog задаёт acceptance, но
не даёт разрешения менять соседний slice. При отдельной задаче без backlog ID достаточно её
проверяемой цели и подходящего ExecPlan по [PLANS](PLANS.md).

Handoff должен позволять продолжить работу без чтения всей переписки: текущий слой и состояние,
что изменено, что фактически проверено, что осталось, blocker/dependency и следующее действие.
Закрытый implementation slice переносится в архив по правилам PLANS; отдельный незавершённый
release gate сохраняет свои evidence и условия завершения. Read-only review сообщает findings и
план в ответе, если пользователь не запросил сохранение отчёта.
