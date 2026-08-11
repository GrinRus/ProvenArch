# Документ для стейкхолдеров и лидов (v1.2 implementation-aligned)

> **Название:** AI-native Architecture Control Plane (Local-first MVP)  
> **Версия:** v1.2 (implementation-aligned)
> **Дата:** 11 August 2026
> **Аудитория:** tech leads, staff/principal engineers, архитекторы, platform teams, engineering managers  
> **Важно:** required CI и deterministic baseline работают на process-scoped runtime policy: `fake` default, `headless` opt-in для реальных локальных прогонов; live provider permission mode по умолчанию `trusted_full_access`, `managed` включается явно в `workspace.yaml`.
> **Q&A boundary (target/current split):** UI stage `Ask` target — async runtime-backed `qa.ask` run over existing workspace artifacts via `POST /api/qa/runs`; deterministic `acp qa` + read-only `POST /api/qa/ask` остаются compatibility/fake baseline surfaces.
> **Refresh maintenance:** ACP объясняет no-op/full/selective execution, переиспользует только validator-promoted baseline shard evidence и не считает commit messages источником архитектурной истины.

---

## 0. Canonical Stakeholder Matrix (source of truth)

Эта матрица — канонический источник статуса `implemented vs planned` для stakeholder-plan.
README/ARCHITECTURE/PLANS/PIPELINE_SPEC должны ссылаться на неё и не противоречить ей.

| Stakeholder requirement | Implementation status | Evidence (artifact/test) |
|---|---|---|
| Runtime policy `fake` default + `headless` opt-in | done | `cmd/acp/main.go` (`--runtime ...`, `--runtime-provider ...`, `ACP_RUNTIME_PROVIDER`, `ACP_CLAUDE_CMD`, `ACP_QWEN_CMD`, `ACP_CODEX_CMD`), `cmd/acp/main_test.go`, `internal/api/server_test.go` |
| Runtime permission policy `trusted_full_access` default + opt-in `managed` auto-approve envelope | done | `schemas/workspace.schema.json`, `internal/runtime/permissions.go`, `internal/orchestrator/runtime_logging.go`, `internal/api/server.go`, `ui/src/App.test.tsx` |
| Baseline flow `validate -> init|refresh -> inspect` (CLI/API/UI) | done | `scripts/smoke-cli.sh`, `scripts/smoke-api.sh`, `ui/src/App.test.tsx` |
| Schema-driven workspace/runtime artifact validation + actionable diagnostics | done | `internal/workspace/validation.go`, `internal/contracts/runtimeexecution.go`, `internal/api/server_test.go` |
| Domain-first per-domain execution with persisted runtime execution metadata + domain outputs | done | `internal/orchestrator/orchestrator.go`, `internal/orchestrator/orchestrator_test.go`, `internal/orchestrator/scenario_test.go` |
| Architect aggregation deterministic output | done | `reports/agent-outputs/architect/summary.md`, `TestArchitectSummaryIsDeterministicAcrossRuns`, scenario golden snapshot |
| Q&A capability with UI + CLI + public API surface | target upgraded | UI uses async `/api/qa/runs`; deterministic `internal/qa` + `acp qa` + `POST /api/qa/ask` remain compatibility/fake baseline |
| Public `POST /api/qa/ask` | done (Epic 11) | read-only wrapper over deterministic workspace-backed QA service |
| User-friendly install + first-run readiness surface | done (trust shell cutover) | `.goreleaser.yml`, `.github/workflows/release.yml`, `install.sh`, `LICENSE`, `cmd/acp/main.go` (`acp version`, `acp doctor`), `internal/api/server.go`, `ui/src/components/ProductShell.tsx`, `ui/src/components/StagePanels.tsx`, `ui/src/App.test.tsx` |
| Onboarding-first workspace/source/runner setup | done (usability hardening) | `acp serve` without `--workspace` starts Guided Setup with Workspace, Sources, Analysis brief, Runner/readiness and Review/start; direct `acp serve --workspace` remains compatibility path. |
| Code quality audit remediation | done (Epic 19 merged at `02716bb`) | `docs/CODE_AUDIT_2026-07-10.md` + `docs/BACKLOG.md` Epic 19: slices `19A..19X` landed crash consistency, lifecycle/shutdown, contract/citation correctness, UI stale-state/editor safety, deterministic build/tooling/release gates, semantic restoration, accessibility primitives and confirmed dead-code cleanup. Required deterministic DoD remains `make contracts`, `make test`, `make lint`, `make build`; live providers remain trusted-machine release gate only. Local frontend security hardening remains Wave 1+ non-goal |
| Console evidence trust and IA reset | in progress (truth + shell slices) | The native History shell now provides `Home / Analyze / Architecture / Changes` plus first-class `/settings`; Documents is the default Architecture mode with Diagrams/Model/Findings siblings, while distinct Run Studio routes, contextual evidence, deep URL context, run-pinned review, server-authored coordination/runtime identity, safe Git publication, responsive navigation/context drawer, current-workspace authority and global read-only Ask remain preserved. |
| Task-first UI, runner presets and content-aware artifact workbenches | planned; authority decisions accepted (Epic 23) | [`spec/TASK_SPEC.md`](spec/TASK_SPEC.md), [`UI_TASK_FIRST_PRODUCT_DESIGN.md`](UI_TASK_FIRST_PRODUCT_DESIGN.md), [`UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md`](UI_TASK_FIRST_PRODUCT_MIGRATION_PLAN.md) and 2026-08-11 Task/Attempt ADRs fix the target identity/persistence/admission/publication boundary; no current schema/API/UI implementation is claimed. |
| Weak-model validation and promotion hardening | in progress; W24A–W24F and W24H runtime-unit budget implemented | [`ADR-20260811-validation-audit-effective-verdict-authority.md`](adr/ADR-20260811-validation-audit-effective-verdict-authority.md) fixes provider draft → technical candidate → provider-free pre-promotion audit → effective verdict; W24H bounds each runtime unit to three provider starts with persisted diagnostics; metric-gated W24G and conformance closure W24I remain. |
| Task-first live E2E and hardened runtime evidence alignment | planned; release boundary accepted (Epic 25) | Cross-epic release-gate task after `23O`/`24I`: exact existing Task/Attempt snapshot inspection and public audit/verdict/budget evidence consumption without a second analysis or canonical profile expansion. |
| Evidence-backed architecture home + impact-aware refresh | done (Epic 21) | `reports/as-is/overview.md` is the validated Architecture Home; refresh records source revisions, impact, actual execution and materialization evidence, supports provider-free no-op and fail-closed affected-only collect, and explains preserved/updated output in Runs and Changes. |
| Post-implementation correctness and trust-boundary audit remediation | done; R3 pending | Epic 22 slices `22A`–`22O` and provider-free closure are complete. The latest diagnostic R3 attempt used clean SHA `aa69d16191f93311560190fc467edd534ed5e567` and stopped at fresh smoke when a provider incorrectly treated source-repository security observations as blocking validator issues. A provider-free technical-only validator-boundary fix is in qualification; no stopped live evidence is accepted. |
| Advisory Workspace Health completion | done (K2b implementation) | Read-only v1 now reports broken/escaping links, model endpoint/alias/owner defects, orphan domain/team outputs, malformed canonical text, unlinked findings and missing proposal evidence; current Knowledge shows the summary without turning health into historical Changes evidence or a publication gate. |
| Outcome-first Architecture and recovery UX | implemented | Home/Architecture expose the promoted result and interactive C4 projection through the Architecture API with a legacy Knowledge fallback; Changes compares individual entities, relations, findings and evidence gaps from immutable promoted semantics; Runs exposes structured useful progress, outcome/recovery and audited child-run retry/rerun for every terminal analysis with revalidated shard/aggregate inputs and validator-gated promotion. |
| Explicit Ask-to-Proposal handoff | done (K3A/K3B implementation) | A succeeded immutable Ask answer exposes a digest; operator confirmation atomically creates a traceable proposal/evidence/source package, refreshes Git truth and opens current Changes→Proposals with Return to Ask. |

Epic matrix:
- done: 1, 2, 3, 4, 5, 6, 7, 8, 9 (within boundary), 10, 11, 14, 15, 16, 17, 19
- done: Epic 20 and Epic 21 implementation complete
- planned: Epic 23, Epic 24 and cross-epic release-gate alignment Epic 25
- completed release prerequisite: Epic 22 qualification handoff. The prior
  step2 fixes are merged in PR #171 (`ca8c3f67`) and PR #172 (`a633e3ce`), but a subsequent static
  and historical-artifact audit demonstrated additional blockers in run/history concurrency,
  filesystem and selected-run containment, refresh preservation, recovery routing, live/product
  isolation and ProductShell state correctness. Slices `22A`–`22O` completed on 2026-07-26 with
  focused race/fault/restart/symlink/snapshot/glob/baseline-integrity/admission-lease and typed
  recovery coverage, deterministic artifact auditing, bidirectional live/product isolation and
  Changes route/Git truth, stale-response suppression, explicit Knowledge/QA evidence authorities
  bounded authority-safe Evidence Viewer behavior, responsive/a11y completion and one deterministic
  provider-free closure command. `make offline-closure` passed from isolated clean commit
  `e8055d65699ed63623f62ad99c3b8406f79c030d`,
  including race/fault/path/boundary suites, 263 Python tests, 158 UI tests, 7 rendered mock
  scenarios, contracts, lint, build and embedded-UI/source-repository drift checks, leaving no
  tracked drift.
- open after Epic 22: Epic 18 R3 trusted-machine composite evidence. The full smoke -> standalone
  fast/long -> fresh three-constituent release-full sequence restarts from the exact clean merge
  commit after the validator-boundary remediation. Diagnostic matrix
  `smoke-tiny-bank-20260729T075044Z` on `aa69d161` is stopped/failed evidence only and is not
  reusable.
- implemented locally by explicit owner request while composite R3 remains pending:
  `K2b -> K4 -> K3A -> K3B -> 9D -> cleanup`. The implementation preserves deterministic
  `acp qa` and `POST /api/qa/ask` compatibility through v1 and retains the tracked
  `golden/readable` human-review baseline. This ordering exception does not satisfy or replace R3.
- out of MVP: 12, 13

---

## 1. Краткое резюме

Мы предлагаем создать **dev-first архитектурный сервис**, который автоматически строит и поддерживает **evidence-backed as-is architecture workspace** для множества репозиториев, даже когда сервисы **не описаны или описаны плохо**. `refresh` сохраняет source revision/impact evidence, безопасно завершает no-op без provider execution и переиспользует только валидные unaffected shard packs. Ключевое отличие: архитектура хранится не как “набор схем”, а как **версионируемая модель**, из которой компилируются отчёты и представления.

**В MVP мы сознательно делаем local-first режим**: пользователь разворачивает сервис локально, локально же у него доступны checkout-папки и/или GitHub/GitLab URL, а все git-операции идут через локальный `git` и уже настроенный доступ пользователя. Документы лежат в workspace (например, вручную выгруженные из Confluence). Это снижает барьер внедрения и откладывает вопросы безопасности/комплаенса на Wave 1+.

**Дополнительно для MVP предусматриваем полноценную интеграцию standalone сервиса с CI/CD**: тот же orchestrator/CLI запускается из GitHub/GitLab webhook-triggered workflow и/или manual pipeline button/job, без hosted control plane.

**В MVP используем step-scoped headless runtime providers**: `claude-code` (default fallback), `qwen-code` и `codex-code` (release peer).

**Техническое решение (принято):**
- реализация продукта: **Go** (orchestrator/server) + UI (React/TypeScript, локально, с встраиванием в Go-бинарь);
- рантайм анализа (MVP): **headless multi-provider** (`claude-code` default, `qwen-code` optional, `codex-code` release peer).
- выбор model/effort доступен provider-scoped профилем `runtime.profile.providers`; если override не
  задан, каждый CLI сохраняет native default, а effective source явно показывается как
  `provider_default`. Live E2E Codex pin может задаваться отдельно через harness env.

---

## 1.1. 10‑минутный walkthrough (демо‑история MVP)

Ниже — ожидаемая “история” использования MVP от нуля до результата:

1) **Подготовка workspace**
- пользователь поднимает сервис одной командой `acp serve`; default runtime остаётся `fake`;
- UI открывает onboarding: выбирает или создаёт `arch-workspace`, либо открывает Recent workspace; ACP готовит fixed layout и `git init` для workspace root;
- в шаге `Sources` пользователь добавляет один или несколько target repos через local checkout path или Git URL; sources сохраняются в существующий `workspace.yaml.repos[]`;
- onboarding summary показывает текущий шаг, главный blocker и next action, а `Ready` объясняет, почему `Open console` или `Run first analysis` ещё disabled;
- складывает выгрузки docs (например из Confluence) в `docs.imports_path` (default `docs/imports/`);
- ведёт `<docs.imports_path>/index.yaml` как metadata index импортированных материалов;
- в шаге `Runner` выбирает `fake` для deterministic walkthrough или explicit live provider; для headless provider видит expected command, `ACP_*_CMD` override и readiness blocker до первого live analysis.
- pipeline/QA start, смена workspace/runner и Git publication сериализованы одной admission lease; пока есть active или queued run, session/runtime/profile/Git mutations возвращают явный conflict, а UI/API продолжают показывать effective runtime текущей service generation до terminal state.

2) **Шаг 0: Конституция проекта**
- открывает UI → мастер (wizard) по “Конституции”:
  - цель/границы, глоссарий, NFR/FT, правила/анти‑паттерны;
- (опционально) агент предлагает черновик, пользователь подтверждает/редактирует;
- UI сохраняет изменения в workspace и предлагает/выполняет git commit.

3) **Запуск Init pipeline**
- пользователь нажимает “Run Init (1–4)”.

4) **Результат**
- в `model/` появляется каноническая as‑is модель (entity-per-file);
- в `reports/as-is/` появляются service dossiers, интеграции, базы данных и CI/CD описание;
- в `reports/coverage/` появляется coverage report и список открытых вопросов по недостающей информации;
- в `reports/findings/` — список провалов/анти‑паттернов с evidence;
- в `proposals/` — 1–3 “proposal пакета” улучшений (to‑be) + черновики ADR/RFC.
- в UI dashboard видны все run'ы анализа (queued/running/succeeded/failed), включая уже завершённые; terminal `run_canceled` и restart-reconciled failed states показываются пользователю как `canceled`/`recovered`;
- при повторном открытии UI выбирает newest active run и ведёт в `Analysis`, иначе newest completed artifact run и ведёт в `Review`;
- для выбранного run UI показывает полный warnings/error контекст, live logs (в т.ч. structured fields) и поддерживает cancel active run с пояснением cooperative stop, terminal canceled/restart-reconciled status/history/activity labels, retained-evidence recovery actions/copy и сохранения taskrun evidence/history.

5) **Git‑ветка proposal**
- пользователь создаёт `proposal/<topic>` из UI (MVP) или вручную;
- предложения фиксируются как git diff, который можно review/merge стандартными средствами.

---

## 1.2. Конвенция хранения в MVP (зафиксировано)

### Вариант 2 — Один central “architecture workspace” repo (выбран для MVP)
- В MVP используем один отдельный git‑репозиторий `arch-workspace/` как единый рабочий контур ACP.
- `workspace.yaml` валидируется по отдельному schema-contract и хранит только repo sources + `docs.imports_path`.
- В `workspace.yaml` хранятся локальные пути к продуктовым репозиториям и/или GitHub/GitLab URL.
- `repos[].analysis.role` удалён из active contract; workspace manifest хранит только source metadata и optional `analysis.include/exclude`.
- Если указан `git_url`, clone/fetch выполняется через локальный `git` на устройстве пользователя или в runner-контексте CI; unpinned source перед анализом refresh-ится на exact remote default `HEAD` SHA в ACP-owned cache, а пользовательские `path` checkout-ы не изменяются.
- В `docs/imports/` лежат вручную импортированные документы (например, выгрузки из Confluence).
- Layout `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` не конфигурируется через manifest и считается fixed MVP convention.

Почему именно этот формат для MVP:
- единый golden source для charter/skills/model/reports/proposals;
- единая история изменений и предсказуемый Git workflow для proposal-веток;
- более детерминированные прогоны pipeline и проще воспроизводимость результатов;
- позволяет одинаково запускать анализ локально и в GitHub/GitLab-triggered pipeline без отдельного hosted контура.

Рекомендуемый layout:
```text
arch-workspace/
  workspace.yaml
  charter/
    cards/
      domains/
      teams/
  skills/
  model/
  reports/
    as-is/
    findings/
    coverage/
    taskruns/
    agent-outputs/
      domains/
      architect/
    changelog/
  proposals/
  docs/
    imports/
    rfcs/
    meetings/
    decisions/
```

---

## 1.3. Agent Operating Model (MVP)

В этой версии фиксируем `domain-first` operating model:
- Domain Analyst Agent на каждый домен.
- Team overlay через markdown team cards.
- 1 Architect Aggregator Agent, который анализирует outputs domain-агентов и формирует общий синтез.
- System Analyst Q&A capability: target UI flow creates async `qa.ask` runtime run over deterministic context pack; deterministic workspace-backed `acp qa` remains compatibility CLI.
- На каждую итерацию формируется markdown changelog.

Обязательный baseline bundle для MVP:
- agents: `domain-analyst`, `architect-aggregator`, `system-analyst-qa`
- skills: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`, `qa`
- prompt packs: `constitution`, `collect-context`, `findings`, `proposals`, `qa`
- prompt packs остаются редактируемыми Git-tracked artifacts, но используются как content layer; enforced runtime policy и contract guardrails задаются internal step-policy слоем

Статус epics и границы scope фиксируются только в секции
`0. Canonical Stakeholder Matrix (source of truth)` этого документа.
Дополнительные “горизонты по идеям” не считаются источником статуса.

Важно: текущее ограничение сохраняется — внешние автоинтеграции Confluence/Jira/Notion не входят в MVP.

---

## 2. Проблема и почему это важно

### 2.1. Что болит
- Архитектурные знания размазаны: часть в головах, часть в коде/инфре/CI, часть в устаревших схемах/доках.
- Сервисы часто **вообще не описаны** (или описание не соответствует реальности).
- Перед изменениями сложно оценить **blast radius**: кого затронет, где сломается, кого звать на review.
- Онбординг и cross-team коммуникации дорогие: “где правда?” и “как оно работает?”.

### 2.2. Почему подход “просто диаграммы” не работает
Ручные схемы умирают быстро. Нужен механизм, который:
- собирает архитектуру из артефактов и кода,
- фиксирует доказательства (откуда факт),
- умеет обновляться при изменениях,
- даёт управляемый workflow улучшений.

---

## 3. Что это за сервис и для чего нужен

### 3.1. Определение

**Architecture Control Plane (ACP)** — это сервис, который:

1) **Собирает контекст** по проекту/системе из локальных репозиториев, GitHub/GitLab-репозиториев и локально добавленных документов.  
2) Строит **каноническую as‑is модель**: компоненты, интерфейсы, зависимости, ownership, внешние интеграции, базы данных, инфраструктурные и CI/CD следы.  
3) Компилирует отчёты/представления:
   - “как работает система” (as‑is docs + dossiers по сервисам),
   - список провалов/анти‑паттернов,
   - предложения улучшений (to‑be) как ветки/пакеты изменений.

### 3.2. Что даёт командам (в бизнес‑терминах)
- Быстрее понять систему и последствия изменений.
- Снизить риск “скрытых зависимостей”.
- Иметь единый источник архитектурного контекста, который не устаревает в день создания.

---

## 4. Основной продуктовый тезис, принципы и доверие к данным

### 4.1. Тезис
**Архитектура — это модель, а не диаграмма.** Диаграммы, ADR и отчёты — производные, компилируемые представления.

### 4.2. Принципы (MVP‑важные)
1) **Facts-first:** сначала факты из кода/артефактов, потом интерпретации.  
2) **Provenance & confidence:** каждый факт помечен источником и уверенностью.  
3) **AI предлагает — человек утверждает:** особенно на спорных связях и улучшениях.  
4) **Local-first:** всё запускается и хранится локально в MVP.  
5) **Git как источник версионирования:** модель, правила и выводы — файлы в git, ветки/диффы/мерджи — через git.

### 4.3. Таксономия фактов (обязательное определение)
Чтобы команда доверяла as‑is карте, мы разделяем факты на три категории:

- **Observation** — наблюдение, доказуемое конкретным артефактом (файл/строки/манифест/спека).  
  Пример: “в repo X найден OpenAPI с endpoint /payments”.

- **Inference** — гипотеза/вывод агента на основании косвенных сигналов.  
  Пример: “похоже, сервис относится к домену Payments” (по структуре кода, именам, docs).

- **Assertion** — утверждение, подтверждённое человеком (или принятое как правило организации).  
  Пример: “svc.payments принадлежит bounded context Payments” (подтверждено архитектором).

**Политика отображения MVP:** по умолчанию показываем Observations + Assertions, а Inference отображаем как “needs review” (с пониженной уверенностью и явной маркировкой).

---

## 5. Что входит в MVP и что сознательно НЕ входит

### 5.1. MVP — ключевой функционал
- **Headless multi-provider runtime (`claude-code` + `qwen-code` + `codex-code`)**.  
- **Локальный запуск** сервиса и UI как основной режим.  
- **Полноценная standalone интеграция с GitHub/GitLab CI/CD** тем же orchestrator/CLI, через hooks и/или manual pipeline triggers, без hosted control plane.  
- Пользователь указывает **локальные пути** к репозиториям или **GitHub/GitLab URL**, но git access всегда идёт через локальный `git` контекст устройства/runner.  
- **Обязательный baseline bundle agents/skills/prompts**, который поставляется вместе с продуктом и редактируется в workspace.  
- **Карточки доменов/команд** (markdown) как source-of-truth в `charter/cards/`.  
- **Domain-first иерархия агентов**:
  - Domain Analyst Agent per domain,
  - Team overlay через team cards,
  - 1 Architect Aggregator Agent.
- **System Analyst Q&A capability** (UI stage `Ask` + async runtime-backed `/api/qa/runs`; deterministic CLI `acp qa` + `POST /api/qa/ask` compatibility).
- **Итерационный changelog** в `reports/changelog/`.  
- **Подробный analysis scope на каждый сервис**:
  - архитектура и интерфейсы,
  - внешние интеграции,
  - базы данных и storage usage,
  - CI/CD pipeline, build/deploy flow, runtime clues.
- **All-stacks extraction strategy**:
  - MVP не фиксирует narrow whitelist языков/стэков,
  - headless providers (`claude-code|qwen-code|codex-code`) + baseline prompts/skills пытаются анализировать arbitrary stacks,
  - при нехватке evidence система пишет unknowns, а не придумывает факты.
- **Явная фиксация unknowns**:
  - `reports/coverage/*`,
  - `questions`,
  - findings по отсутствующим доказательствам/артефактам.  
- **Init pipeline**:
  0) интерактивная “Конституция проекта” (шаблоны + редактирование)  
  1) сбор контекста из кода/артефактов  
  2) генерация as‑is документов  
  3) анализ провалов и анти‑паттернов  
  4) предложения улучшений  
- **Subagents + Skills**:
  - субагенты выполняют специализированные роли,
  - skills — пакеты промптов/правил/шаблонов,
  - редактирование skills и принципов — через UI с версионированием в git.
- **Git-based ветвление**: as‑is и предложения улучшений как ветки/диффы/мерджи.

### 5.2. В MVP НЕ делаем (осознанно)
- Security policy/комплаенс (только оговорки “не кладите секреты”).  
- Масштабирование стоимости (budget/caching/приоритизация на орг‑уровне) — позже.  
- Ролевые поверхности и сложный UX для всех ролей — позже (в MVP UI упрощённый).  
- Авто‑интеграции с Confluence/Notion/Jira и пр. (включая autodocs) — позже.
- Manager‑агенты по Jira/resource skew — позже.

### 5.3. Не‑цели (чтобы не завысить ожидания)
- ACP **не** является enterprise architecture suite (ArchiMate/EA‑репозиторий).  
- ACP **не** является whiteboard/diagram editor.  
- ACP **не** является security/compliance платформой в MVP.

---

## 6. Верхнеуровневое устройство (архитектура MVP)

### 6.1. Компоненты MVP (локально)
1) **Central Architecture Workspace (git репозиторий `arch-workspace/`)**  
   хранит: “Конституцию”, rules, skills, model, отчёты, findings, предложения

2) **UI (локальный web-интерфейс)**  
   - Proven Arch console с top status bar, product-flow rail `Source / Readiness / Charter / Analysis / Review / Proposals / Ask / Publish`, центральной рабочей областью, правым inspector и bottom activity drawer  
   - wizard summary, domain/team card overview, baseline prompt bundle status и charter baseline recovery для “Конституции” в `Charter`
   - настройка источников репозиториев (`path` или `git_url`) в `Source` с repo table для source/ref/validation state и source validation recovery для blocking repo/source diagnostics
   - readiness validation, summary cards, provider readiness recovery, doctor checklist, read-only workspace health snapshot и runtime profile (`timeouts` + `execution` + `permissions`) в `Readiness`
   - редактор baseline skills/prompts (с версионированием через git)  
   - запуск пайплайнов (init / refresh) в `Analysis` с run mission control, canonical step timeline, failed-run recovery path, terminal canceled/recovered status/history labels, retained-evidence recovery actions, provider-unavailable Readiness recovery, live diagnostics для shard/repair/stall/raw-output сигналов, shard/log table, warning/error drilldown и pending permission triage
   - logs activity drawer с dual-view (`event timeline` + `raw agent stream`) и terminal canceled/recovered empty-log copy
   - `Review` для evidence tabs, grouped artifact explorer, markdown/Mermaid preview, coverage/open-question/trust summary и artifact-derived Domain Map по `model/entities/*`, `model/edges/*`, `reports/agent-outputs/domains/*` с explicit partial states
   - `Proposals` для review room по proposal/changelog packages: proposal package recovery для incomplete proposal/changelog packages, preview/evidence/changelog/diff tabs, quality blockers и publication path перед `Publish`
   - `Ask` для async agent-backed Q&A через `POST /api/qa/runs`, с history/citations/safety/audit links, failed-run recovery и deterministic `POST /api/qa/ask` compatibility API
   - `Publish` для Git Review Room: publication readiness summary, mobile section jumps, folder-level artifact summary, selected artifact preview, explicit diff partial state, publish gate/checklist, commit plan, prepared commit-message copy action, failed Git action recovery и proposal branch

3) **Orchestrator (локальный сервис, Go)**  
   - управляет шагами pipeline  
   - готовит контекст и PromptPack перед запуском каждого шага  
   - загружает baseline bundle agents/skills/prompts из workspace  
   - разрешает repo sources (`path`/`git_url`) в локальные checkout перед анализом через системный `git` текущего пользователя/runner; для unpinned `git_url` фиксирует exact resolved SHA в run evidence
   - принимает только required step artifacts + runtime execution metadata и сохраняет их в workspace
   - работает в interactive local mode и non-interactive CI mode
   - вызывается как напрямую пользователем, так и из CI/CD trigger flows

4) **Runtime Providers (headless)**  
   - запускается orchestrator’ом  
   - использует subagents + skills  
   - опирается на baseline prompts/skills для arbitrary stacks  
   - читает локальные checkout репозиториев и документы

5) **Agent Topology (MVP)**  
   - Domain Analyst Agent (per domain)  
   - Team overlay через team cards  
   - Architect Aggregator Agent (1 на workspace)  
   - System Analyst Q&A capability (`qa.ask` runtime run + deterministic `acp qa` compatibility)

6) **Model & Reports (файлы)**  
   - каноническая модель как файлы (git-tracked)  
   - отчёты, evidence, agent outputs и changelog как файлы (git-tracked)  
   - coverage/questions как отдельные артефакты для unknowns

### 6.2. Артефакты на диске (MVP контракт)
MVP должен стабильно производить следующий набор артефактов в `arch-workspace/`:

- `workspace.yaml` — список локальных репозиториев + настройки путей, валидируемый по отдельной schema  
- `charter/` — Конституция проекта (шаблоны + отредактированные значения)  
- `charter/cards/domains/<domain-id>.md` — карточки доменов (source-of-truth)  
- `charter/cards/teams/<team-id>.md` — карточки команд (team overlay)  
- `skills/` — skills/prompts/templates (редактируемые в UI, git‑tracked)  
  - `skills/subagents.yaml` — baseline agent mapping  
  - baseline skill directories — versioned prompt bundles  
- `model/` — каноническая модель (entity-per-file):
  - `model/entities/...` (services/apis/datastores/…)  
  - `model/edges/...` (calls/publishes/subscribes/reads/writes/…)  
- `reports/as-is/` — as‑is документы (overview, catalog, flows, etc.)  
- `reports/as-is/services/<service-id>.md` — подробный разбор каждого сервиса  
- `reports/as-is/integrations.md` — внешние интеграции и dependency surface  
- `reports/as-is/datastores.md` — базы данных, storage usage и data hints  
- `reports/as-is/ci-cd.md` — как устроены build/test/deploy workflows  
- `reports/findings/` — findings (anti‑patterns, gaps)  
- `reports/coverage/summary.md` — coverage отчёты (что извлечено, что не найдено)  
- `reports/coverage/open-questions.md` — открытые вопросы по недостающей информации  
- `proposals/<topic>/` — пакеты улучшений + ADR/RFC drafts  
- `reports/taskruns/` — сохранённые runtime execution metadata и raw failure artifacts (для воспроизводимости и дебага)
- `reports/agent-outputs/domains/<domain-id>.md` — outputs domain-агентов  
- `reports/agent-outputs/architect/summary.md` — синтез Architect Aggregator Agent  
- `reports/changelog/<yyyy-mm-dd>-<iteration-id>.md` — changelog по итерациям
- `<docs.imports_path>/index.yaml` — metadata index импортированных документов

---

## 7. Git workflow (решение для MVP)

**Решение для MVP:** UI предоставляет “git helper actions” и выполняет минимум:

- **Commit changes** (кнопка): коммитит изменения в `arch-workspace/` с понятным сообщением.  
- **Create proposal branch** (кнопка): создаёт ветку `proposal/<topic>` от текущего состояния.

**Merge/PR‑подобный процесс в MVP:** остаётся стандартным git‑процессом (CLI/IDE). В Wave 1 можно добавить PR‑подобный UI.

**CI/CD сценарий MVP:** standalone ACP должен полноценно работать с CI/CD в двух mode:
- GitHub/GitLab webhook инициирует native workflow/job, который запускает ACP batch mode;
- manual pipeline button/job запускает тот же ACP flow вручную.

Во всех случаях git access идёт через локальный runner/user context; ACP не хранит отдельные credentials и не требует hosted control plane.

---

## 8. Этап 0 — “Конституция проекта” (интерактивно)

### 8.1. Как выглядит в UI (MVP)
Wizard из блоков-шаблонов:
1) Project purpose / scope  
2) Key domains / glossary  
3) System constraints (например: данные, регуляторика, латентность)  
4) NFR/FT (SLO, availability, reliability expectations)  
5) Architectural rules (allowed / forbidden patterns)  
6) Output expectations (что хотим видеть в as‑is и report)

### 8.2. Авто‑помощь агента
- агент предлагает черновик Конституции (по структуре реп и имеющимся docs),
- пользователь подтверждает/редактирует,
- wizard создаёт initial domain/team cards,
- Конституция и initial cards сохраняются в git и становятся “golden source”.

---

## 9. Init pipeline (первичный запуск)

### 9.1. Шаг 1 — Сбор контекста из локальных репозиториев и документов
**Вход:**
- локальные пути к репозиториям и/или GitHub/GitLab URL (пользователь указывает)  
- локальная папка документов (RFC, meeting notes, выгрузки из Confluence и пр.)  
- Конституция + skills/rules

**Действия рантайма:**
- анализ arbitrary stacks через headless providers (`claude-code|qwen-code|codex-code`) + baseline prompts/skills (без фиксированного whitelist language adapters в MVP)  
- инвентаризация сервисов/юнитов  
- извлечение интерфейсов (HTTP/gRPC/events), зависимостей, инфраструктурных следов  
- извлечение внешних интеграций и third-party/system dependencies  
- извлечение баз данных, storage usage, migration/runtime hints  
- извлечение CI/CD конфигурации (`.gitlab-ci.yml`, Dockerfile, deploy manifests, helm/k8s, scripts)  
- поиск ownership hints (CODEOWNERS/структура/история)  
- формирование evidence (файлы/фрагменты) и confidence  
- при недостатке данных: явная фиксация gaps через coverage/questions/findings вместо выдумывания фактов

**Выход:**
- as‑is модель (файлы)  
- evidence index (файлы)  
- coverage report (что не нашли)  
- явные вопросы по ownership / integrations / databases / CI-CD gaps
- новый домен/команда или неизвестный owner surface-ятся как `question` и/или `finding`
- enrich существующих domain/team cards derived references и coverage links
- automatic create/rename canonical cards не допускается
- outputs domain-агентов в `reports/agent-outputs/domains/*`

### 9.2. Шаг 2 — Документы “как работает as‑is”
- overview системы  
- сервисный каталог  
- dossiers по каждому сервису  
- основные потоки (по возможности)  
- зависимости и внешние интеграции  
- datastores и storage footprint  
- как устроен CI/CD в каждом сервисе  
- “что важно знать” (onboarding)
- full C4 Mermaid set (`Context`, `Container`, per-service `Component`, per-service `Code`) + `reports/diagrams/index.md`
- strict evidence policy: если данных недостаточно, диаграммы показывают явные `Gap:*` маркеры, без выдуманных узлов

### 9.3. Шаг 3 — Провалы и анти‑паттерны
- findings с severity, объяснением и ссылками на evidence  
- gaps: “нет owner”, “неясные интерфейсы”, “не найдена DB схема/миграции”, “неясный deploy flow”, “подозрение на циклы зависимостей”, “нарушение правила X”
- architect synthesis report в `reports/agent-outputs/architect/summary.md`

### 9.4. Шаг 4 — Улучшения
- предложение 1–3 улучшений в виде “proposal пакетов”  
- черновики ADR/RFC  
- список шагов (миграционный чеклист уровня MVP)

### 9.5. On-demand Q&A capability (MVP)
- Target System Analyst Q&A capability отвечает через UI stage `Ask` + async runtime-backed `POST /api/qa/runs`, используя deterministic context pack из:
  - `charter/cards/*`
  - `model/*`
  - `reports/as-is/*`, `reports/findings/*`, `reports/coverage/*`
  - `proposals/*`, `reports/changelog/*`
  - configured `docs.imports_path` (`docs/imports/*` по умолчанию)
- Runtime step id: `qa.ask`; agent role: `system-analyst-qa`; prompt pack: `skills/prompt-packs/qa.md`; write scope: только `reports/taskruns/<run_id>/qa/`.
- QA runs не меняют source repos или canonical workspace outputs; они пишут `context-pack.json`, `qa-answer.json` и `runtime-execution.json` в taskrun scope.
- UI `Ask` показывает async run history, selected answer, confidence, citations, unresolved assumptions, explicit related-entity partial state, failed-run recovery/retry guidance, provider-unavailable Readiness guidance, terminal canceled/restart-reconciled answer copy и read-only safety/audit artifact links; отсутствующие structured related entities/edges не домысливаются поверх текущего API.
- Compatibility: deterministic `acp qa` + public read-only `POST /api/qa/ask` остаются temporary fallback surfaces.
- API возвращает `answer`, `citations`, `unresolved`, `confidence`; empty/invalid request идёт через standard API error envelope.

---

## 10. Регулярная работа (итерационный цикл)

### 10.1. Ручной режим MVP
В MVP обновления инициируются вручную:
- самый короткий старт: `acp serve`, затем onboarding UI выбирает workspace, target repos и runner; default runtime остаётся `fake`
- direct compatibility старт: `acp serve --workspace ... --auto-init ((--repo-name ... (--repo-path ... | --repo-git-url ...) [--repo-ref ...]) | --repos-file ...) [--docs-imports-path ...] --runtime fake`
- первый bootstrap workspace выполняется через `acp init-workspace --workspace ... ((--repo-name ... (--repo-path ... | --repo-git-url ...) [--repo-ref ...]) | --repos-file ...)`
- первый materialization запуск: `acp run --workspace ... --pipeline init --runtime fake --non-interactive`
- для live запуска: `acp run --workspace ... --pipeline init --runtime headless --runtime-provider qwen-code --non-interactive` (или `claude-code` / `codex-code`)
- пользователь добавил новые документы в workspace  
- пользователь обновил репозитории (git pull)  
- пользователь нажал “Rebuild as‑is / Re-run analysis”
- по итогам итерации формируется `reports/changelog/<date>-<iteration-id>.md`

### 10.2. Batch режим в CI/CD (MVP)
- required integration surface: `acp run --workspace ... --pipeline ... --non-interactive`
- тот же orchestrator может запускаться без UI внутри GitHub/GitLab job/runner  
- запуск идёт через webhook-triggered workflow и/или manual pipeline button/job  
- default auto-trigger: `push` в default branch  
- `merge request` / `pull request` updates идут как manual/preview trigger  
- runner использует локальные checkout и/или свой локальный `git` context для `git_url` из `workspace.yaml`  
- результатом остаются те же git-tracked артефакты workspace  
- если данных недостаточно, job не придумывает ответ, а пишет gaps/questions/coverage
- одновременно активен только один run на workspace; debounce window 5 минут, policy `last event wins`
- workspace/runtime session replacement и runtime profile writes запрещены во время active/pending work, чтобы queued/running analysis всегда оставался привязан к одной service generation
- internal API trigger допустим только как optional trusted local/private deployment mode и не является обязательной CI/CD поверхностью MVP

### 10.3. Позже (после MVP)
- автоподтягивание изменений  
- интеграции с Confluence/Jira/трекерами  
- автоматические nightly scans  
- governance и security

---

## 11. Дорожная карта (MVP → Wave 1 → Wave N)

### MVP (Local-first, multi-provider headless)
- central `arch-workspace` (git)  
- repo sources через локальные папки и/или GitHub/GitLab URL, разрешаемые через локальный `git` context  
- интерактивная Конституция  
- standalone CI/CD integration через hooks и manual pipeline triggers  
- skills/subagents и их редактура в UI (git versioning)  
- domain/team cards в `charter/cards/*`  
- domain-first агентная иерархия (domain analysts + architect aggregator)  
- Q&A capability системного аналитика  
- итерационный changelog в `reports/changelog/*`
- init pipeline 0–4  
- as‑is docs + service dossiers + CI/CD/integrations/datastores reports  
- explicit unknowns через coverage/questions/findings  
- git-based branching (commit + proposal branch)

### Wave 1 (после MVP)
- evidence-backed architecture home и impact-aware incremental refresh с explainable no-op
- интеграции с внешними источниками (Confluence/Jira/Notion/etc)  
- autodocs integration  
- manager-агенты по Jira/resource skew  
- автоматические обновления (webhooks/nightly)  
- улучшенный UX (PR‑подобный review, role views)  
- cost/scaling (кеши, приоритизация)  
- security baseline (policies, audit)

### Wave N
- расширение списка runtime providers (через RuntimeProvider)  
- governance “как продукт” (policy engine + exceptions)  
- drift detection по runtime/observability  
- org-scale аналитика, scorecards, compliance overlays

### Критерии перехода MVP → Wave 1 (предложение)
Переход имеет смысл, когда:
- **Coverage:** ≥70% сервисов имеют выявленные интерфейсы и ≥60% имеют owner hints (или явные unknown с вопросами)  
- **Use:** команда использует ACP на каждом существенном изменении (или хотя бы на design review)  
- **Trust:** доля inference‑связей, которые требуют ручного исправления, снижается (или стабильно управляется через inbox)  
- **Workflow:** proposals реально живут как ветки и проходят review (не только “сгенерировали и забыли”)

---

## 12. Схема процесса (MVP)

```mermaid
flowchart TD
  A[Workspace: repo paths or GitHub/GitLab URLs + docs/imports] --> B[Step 0: Charter Wizard]
  B --> C[Step 1: Collect Context (Headless Provider)]
  X[GitHub/GitLab hook or pipeline button] --> C
  C --> D[Step 2: As-is Docs]
  D --> E[Step 3: Findings / Anti-patterns]
  E --> F[Step 4: Proposals (to-be packages)]
  F --> G[Git: commit / proposal branch]

  G --> H[Manual loop: update repos/docs]
  H --> C
```

---

## 13. Зафиксированные baseline-решения (2026-04-02)

### 13.1. Каноническая модель и ID
- В MVP фиксируем минимальную entity-per-file модель без дальнейшего расширения core types на этом этапе.
- Canonical ID patterns фиксируются по типам: `svc.<slug>`, `team.<slug>`, `repo.<slug>`, `ext.<slug>`, `db.<engine>.<slug>`, `api.http.<service-slug>.<method>.<path-slug>`, `api.grpc.<service-slug>.<service>.<method>`, `topic.<slug>`, `edge.<from>.<type>.<to>`.
- Slug normalization: lowercase ASCII + kebab-case; для path params используется stable replacement rule (`{id}` -> `by-id`).
- При коллизии добавляется suffix `.repo-<repo-slug>`.
- `owner_team_id` должен ссылаться на существующий `team.<slug>`; неизвестный owner не создаёт auto-team entity.
- Stable IDs не пересчитываются автоматически при rename/move; для миграций используются `aliases[]` и явная модельная правка.

### 13.2. Контракт runtime execution
- MVP использует только artifact-only runtime contract.
- Runtime пишет required step artifacts в `write_root` / `draft_final_root` и завершает процесс без semantic JSON на stdout.
- Orchestrator принимает шаг только после read-only validation artifacts и persisted runtime execution metadata.
- В `managed` permission mode orchestrator auto-approves только reads под `read_context_roots` и writes под `write_root`/`draft_final_root`; shell/network/package install/unknown requests не auto-approved и в non-interactive режиме завершаются `runtime_permission_required`. UI показывает такие pending requests в `Analysis` как triage summary с target/reason, rule/decision и next actions; правый inspector mirrors step/rule/target/reason в hard blocker; approve/deny broker остаётся future scope.
- Observation без evidence запрещён policy и examples.

### 13.3. Skills/prompts editing через UI
- Минимальный UX MVP: edit → validate → commit → history → rollback.
- Валидация prompt bundle опирается на manifest checks и быстрые fixture dry-runs.

### 13.4. Extraction strategy
- MVP не ограничивается narrow whitelist языков/стэков.
- Анализ arbitrary stacks выполняется headless providers (`claude-code|qwen-code|codex-code`) + baseline prompt/skill bundle.
- Если стек или артефакт не удаётся надёжно интерпретировать, система фиксирует gaps через coverage/questions/findings.

### 13.5. Документы и metadata index
- Raw imports хранятся в `docs.imports_path` (default `docs/imports/`).
- Metadata фиксируется в `<docs.imports_path>/index.yaml`: required `id`/`path`, optional `source`, `checksum`, `imported_at`, `source_updated_at`, `status`.
- Отсутствие metadata index допустимо; malformed/semantic index issues surface как warning-only workspace diagnostics.
- `workspace.yaml` получает отдельный schema-contract; layout workspace beyond repo sources и imports path не конфигурируется через manifest.

### 13.6. GitHub/GitLab trigger policy
- Auto-trigger по умолчанию: `push` в default branch.
- `merge request` / `pull request` updates идут как manual/preview trigger.
- Manual pipeline button/job обязателен.
- Required MVP CI/CD surface: CLI batch mode.
- Internal API trigger optional и допустим только для trusted local/private deployment.
- Debounce policy: один активный run на workspace, окно 5 минут, `last event wins`.

### 13.7. Q&A API baseline
- Target endpoint: `POST /api/qa/runs` creates async QA run; `GET /api/qa/runs/<run_id>` returns status, provider identity, answer, citations, unresolved and confidence.
- Compatibility endpoint: `POST /api/qa/ask` remains read-only deterministic fallback.
- QA run writes only `reports/taskruns/<run_id>/qa/*`; it does not mutate source repos or canonical `charter/`, `model/`, `reports/*`, `proposals/*` outputs.

---

## 14. Appendix: reference layout central workspace (MVP)

Ниже — референс структуры для выбранной MVP-конвенции (central `arch-workspace`):

```text
arch-workspace/
  workspace.yaml
  charter/
    cards/
      domains/
      teams/
  skills/
  model/
  reports/
    as-is/
    findings/
    coverage/
    taskruns/
    agent-outputs/
      domains/
      architect/
    changelog/
  proposals/
  docs/
    imports/
    rfcs/
    meetings/
    decisions/
```
