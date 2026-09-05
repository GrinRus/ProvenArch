# AGENTS.md

Устойчивые правила разработки ProvenArch. Процедуры и маршруты к коду живут в
[agent development guide](docs/AGENT_DEVELOPMENT.md), specs и skills.

## Продукт

ACP — local-first инструмент: Go backend/orchestrator + embedded React UI и отдельный
Git-версионируемый architecture workspace с entity-per-file моделью. `fake` — deterministic
baseline; live runtime opt-in: `claude-code` default fallback, `qwen-code` optional,
`codex-code` release peer. Hosted режим и security/compliance enforcement вне MVP.

## Где искать ответ

- Начать с [README](README.md), затем выбрать затронутую строку
  [карты кода, specs и проверок](docs/AGENT_DEVELOPMENT.md#карта-изменений).
  Читать относящиеся к задаче разделы, а не весь корпус документации.
- Контракты: `schemas/*` задают форму данных, `docs/spec/*` — семантику и инварианты.
  Код и проверки показывают фактическое поведение; расхождение со spec фиксировать как drift,
  а не разрешать молчаливым переписыванием контракта.
- Статус реализации: canonical matrix в [STAKEHOLDER_DOC](docs/STAKEHOLDER_DOC.md).
  Текущая работа и зависимости: [PLANS](docs/PLANS.md). [BACKLOG](docs/BACKLOG.md) — acceptance
  reference; завершённый slice не переоткрывать по старому описанию. Несогласованные статусы
  перепроверять по evidence; новая задача пользователя может не иметь backlog ID.
- Для pipeline/runtime обязателен [PIPELINE_SPEC](docs/spec/PIPELINE_SPEC.md); для Task/Attempt
  и UI — [TASK_SPEC](docs/spec/TASK_SPEC.md) и маршруты из guide.
- Использовать релевантные `.agents/skills/*`: `acp-spec-first-slice` для новой фичи,
  `acp-schema-guardian` для контрактов, `acp-test-fixtures` для core/model logic,
  `acp-docs-sync` для behavior docs, `acp-e2e-live-gate` для live/release gate.
  Это навыки разработки; workspace runtime prompts — отдельная продуктовая поверхность.

## Рабочий контракт

- В начале назвать слой: research, design, implementation, review или release; смену слоя отмечать.
- Для answer/review/diagnose исследовать и сообщить результат без правок кода. Для change/build/fix
  самостоятельно выполнить локальный in-scope slice и его проверки.
- До правок определить проверяемый результат, non-goals и stop condition. Для многошаговой работы
  использовать существующий ExecPlan или добавить новый по [PLANS](docs/PLANS.md); держать
  актуальными зависимости, evidence и следующее действие. Read-only audit не требует правки tracker.
- Сохранять пользовательские изменения и делать reviewable diff. Параллельные задачи должны иметь
  независимые границы файлов/ответственности и финальный синтез; не менять чужой active slice.
- Release/push/external writes, destructive действия, новые production dependencies и существенное
  расширение scope требуют явного разрешения. Уже данное разрешение не запрашивать повторно.

## Инварианты реализации и review

- Анализируемые репозитории — read-only inputs. Все outputs пишутся в разрешённые корни отдельного
  workspace; проверять containment и identity до чтения, мутации и promotion.
- Artifact-only runtime: успех подтверждают валидные step artifacts, stdout/stderr — diagnostics.
  Сохранять staged validation → promotion и last-good knowledge при неуспешной попытке.
- Task/Attempt/run identity и evidence authority задаёт backend. Не подменять explicit missing/stale
  identity другим run, mutable current state или предположением frontend.
- Сохранять пользовательские workspace values и editable baseline files. Изменение runtime prompts,
  provider/model/reasoning defaults или списка providers — отдельный измеримый slice с тестами;
  возможности новой модели сами по себе не основание добавлять orchestration или defaults.
- При изменении контракта синхронизировать затронутые schemas/specs, validators, examples,
  tests/fixtures, `docs/APPENDIX_SCHEMAS.md` и ADR rationale через schema-guardian. Core behavior и
  testing baseline требуют соответствующих fixtures и behavior docs, а не механической правки всех docs.

## Проверка и release

- Подготовка среды и точные toolchains: [CONTRIBUTING](CONTRIBUTING.md). Во время работы запускать
  узкий релевантный check; для завершённого implementation slice выполнить полный DoD:
  `make contracts`, `make test`, `make lint`, `make build`. Дополнительные UI/fixture checks — в guide.
- Required CI остаётся deterministic, без live provider/network execution. Проверки структуры
  agent guidance выполняет `make verify-agent-guidance`.
- Live/release gate выполнять через `acp-e2e-live-gate` и
  [RELEASE_LIVE_E2E_RUNBOOK](docs/RELEASE_LIVE_E2E_RUNBOOK.md): canonical harness
  `scripts/full-run-batch-matrix.sh` без wrapper, manual trusted-machine gate вне required CI.
  Не менять canonical matrices или curated `repos_file` для обхода ограничений хоста.
- Завершение подтверждать фактическими проверками; blocker сообщать с evidence и минимальным
  следующим действием. Implementation complete и release accepted — разные результаты.
