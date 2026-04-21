# Документ для стейкхолдеров и лидов (v1.0 implementation-aligned)

> **Название:** AI-native Architecture Control Plane (Local-first MVP)  
> **Версия:** v1.0 (implementation-aligned)  
> **Дата:** 05 Apr 2026  
> **Аудитория:** tech leads, staff/principal engineers, архитекторы, platform teams, engineering managers  
> **Важно:** required CI и deterministic baseline работают на process-scoped runtime policy: `fake` default, `headless` opt-in для реальных локальных прогонов.  
> **Q&A boundary (beta):** workspace-backed capability доступна как internal service + CLI `acp qa`; публичный endpoint `POST /api/qa/ask` остаётся Epic 11 follow-up.

---

## 0. Canonical Stakeholder Matrix (source of truth)

Эта матрица — канонический источник статуса `implemented vs planned` для stakeholder-plan.
README/ARCHITECTURE/PLANS/PIPELINE_SPEC должны ссылаться на неё и не противоречить ей.

| Stakeholder requirement | Implementation status | Evidence (artifact/test) |
|---|---|---|
| Runtime policy `fake` default + `headless` opt-in | done | `cmd/acp/main.go` (`--runtime ...`, `--runtime-provider ...`, `ACP_RUNTIME_PROVIDER`, `ACP_CLAUDE_CMD`, `ACP_QWEN_CMD`), `cmd/acp/main_test.go`, `internal/api/server_test.go` |
| Baseline flow `validate -> init|refresh -> inspect` (CLI/API/UI) | done | `scripts/smoke-cli.sh`, `scripts/smoke-api.sh`, `ui/src/App.test.tsx` |
| Schema-driven workspace/taskresult validation + actionable diagnostics | done | `internal/workspace/validation.go`, `internal/contracts/taskresult.go`, `internal/api/server_test.go` |
| Domain-first per-domain execution with raw taskruns + domain outputs | done | `internal/orchestrator/orchestrator.go`, `internal/orchestrator/orchestrator_test.go`, `internal/orchestrator/scenario_test.go` |
| Architect aggregation deterministic output | done | `reports/agent-outputs/architect/summary.md`, `TestArchitectSummaryIsDeterministicAcrossRuns`, scenario golden snapshot |
| Q&A capability without public beta API surface | done (boundary-complete) | `internal/qa/service.go`, `cmd/acp/main.go` (`acp qa`), `docs/spec/API_SPEC.md` (`/api/qa/ask` follow-up) |
| Public `POST /api/qa/ask` | follow-up (Epic 11) | not in current beta API surface |

Epic matrix:
- done: 1, 2, 3, 4, 5, 6, 7, 8, 9 (within boundary), 10, 14, 15
- follow-up: 11
- out of MVP: 12, 13

---

## 1. Краткое резюме

Мы предлагаем создать **dev-first архитектурный сервис**, который автоматически строит и поддерживает **as-is архитектуру** для множества репозиториев, даже когда сервисы **не описаны или описаны плохо**. Ключевое отличие: архитектура хранится не как “набор схем”, а как **версионируемая модель**, из которой компилируются отчёты и представления.

**В MVP мы сознательно делаем local-first режим**: пользователь разворачивает сервис локально, локально же у него доступны checkout-папки и/или GitHub/GitLab URL, а все git-операции идут через локальный `git` и уже настроенный доступ пользователя. Документы лежат в workspace (например, вручную выгруженные из Confluence). Это снижает барьер внедрения и откладывает вопросы безопасности/комплаенса на Wave 1+.

**Дополнительно для MVP предусматриваем полноценную интеграцию standalone сервиса с CI/CD**: тот же orchestrator/CLI запускается из GitHub/GitLab webhook-triggered workflow и/или manual pipeline button/job, без hosted control plane.

**В MVP используем step-scoped headless runtime providers**: `claude-code` (default fallback) и `qwen-code`.

**Техническое решение (принято):**
- реализация продукта: **Go** (orchestrator/server) + UI (React/TypeScript, локально, с встраиванием в Go-бинарь);
- рантайм анализа (MVP): **headless multi-provider** (`claude-code` default, `qwen-code` optional).

---

## 1.1. 10‑минутный walkthrough (демо‑история MVP)

Ниже — ожидаемая “история” использования MVP от нуля до результата:

1) **Подготовка workspace**
- пользователь поднимает сервис одной командой `acp serve --workspace ... --auto-init ((--repo-name ... (--repo-path ... | --repo-git-url ...) [--repo-ref ...]) | --repos-file ...) [--docs-imports-path ...]`;
- при `--auto-init` ACP создаёт `workspace.yaml` и fixed layout автоматически;
- при bootstrap ACP автоматически выполняет `git init` в workspace root, если `.git` отсутствует;
- складывает выгрузки docs (например из Confluence) в `docs/imports/`;
- ведёт `docs/imports/index.yaml` как metadata index импортированных материалов.

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
- в UI dashboard видны все run'ы анализа (queued/running/succeeded/failed), включая уже завершённые;
- для выбранного run UI показывает полный warnings/error контекст, live logs (в т.ч. structured fields) и поддерживает cancel active run.

5) **Git‑ветка proposal**
- пользователь создаёт `proposal/<topic>` из UI (MVP) или вручную;
- предложения фиксируются как git diff, который можно review/merge стандартными средствами.

---

## 1.2. Конвенция хранения в MVP (зафиксировано)

### Вариант 2 — Один central “architecture workspace” repo (выбран для MVP)
- В MVP используем один отдельный git‑репозиторий `arch-workspace/` как единый рабочий контур ACP.
- `workspace.yaml` валидируется по отдельному schema-contract и хранит только repo sources + `docs.imports_path`.
- В `workspace.yaml` хранятся локальные пути к продуктовым репозиториям и/или GitHub/GitLab URL.
- Если указан `git_url`, clone/fetch выполняется через локальный `git` на устройстве пользователя или в runner-контексте CI.
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
- System Analyst Q&A Agent, который отвечает на вопросы по артефактам `charter/cards + model + reports + docs/imports`.
- На каждую итерацию формируется markdown changelog.

Обязательный baseline bundle для MVP:
- agents: `domain-analyst`, `architect-aggregator`, `system-analyst-qa`
- skills: `service-inventory`, `interface-extraction`, `integration-mapping`, `datastore-mapping`, `cicd-mapping`, `ownership-coverage`, `findings`, `proposals`
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
- **Headless multi-provider runtime (`claude-code` + `qwen-code`)**.  
- **Локальный запуск** сервиса и UI как основной режим.  
- **Полноценная standalone интеграция с GitHub/GitLab CI/CD** тем же orchestrator/CLI, через hooks и/или manual pipeline triggers, без hosted control plane.  
- Пользователь указывает **локальные пути** к репозиториям или **GitHub/GitLab URL**, но git access всегда идёт через локальный `git` контекст устройства/runner.  
- **Обязательный baseline bundle agents/skills/prompts**, который поставляется вместе с продуктом и редактируется в workspace.  
- **Карточки доменов/команд** (markdown) как source-of-truth в `charter/cards/`.  
- **Domain-first иерархия агентов**:
  - Domain Analyst Agent per domain,
  - Team overlay через team cards,
  - 1 Architect Aggregator Agent.
- **System Analyst Q&A capability** (on-demand ответы по артефактам workspace).  
- **Итерационный changelog** в `reports/changelog/`.  
- **Подробный analysis scope на каждый сервис**:
  - архитектура и интерфейсы,
  - внешние интеграции,
  - базы данных и storage usage,
  - CI/CD pipeline, build/deploy flow, runtime clues.
- **All-stacks extraction strategy**:
  - MVP не фиксирует narrow whitelist языков/стэков,
  - headless providers (`claude-code|qwen-code`) + baseline prompts/skills пытаются анализировать arbitrary stacks,
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
   - wizard для “Конституции”  
   - настройка источников репозиториев (`path` или `git_url`)  
   - редактор baseline skills/prompts (с версионированием через git)  
   - top-level tabs: `Setup / Baseline / Runs / Results / Settings`  
   - runtime profile (`timeouts` + `execution`) в отдельной вкладке `Settings`  
   - запуск пайплайнов (init / update / report)  
   - `Runs: Logs` с dual-view (`event timeline` + `raw agent stream`)  
   - `Results -> Diagrams` для C4 Mermaid artifacts (filter/open/preview)  
   - просмотр остальных результатов (as‑is docs, findings, proposals)

3) **Orchestrator (локальный сервис, Go)**  
   - управляет шагами pipeline  
   - готовит контекст и PromptPack перед запуском каждого шага  
   - загружает baseline bundle agents/skills/prompts из workspace  
   - разрешает repo sources (`path`/`git_url`) в локальные checkout перед анализом через системный `git` текущего пользователя/runner  
   - принимает структурированный результат (TaskResult) и сохраняет в workspace
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
   - System Analyst Q&A Agent (on-demand capability)

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
- `reports/taskruns/` — сохранённые TaskResult (для воспроизводимости и дебага)
- `reports/agent-outputs/domains/<domain-id>.md` — outputs domain-агентов  
- `reports/agent-outputs/architect/summary.md` — синтез Architect Aggregator Agent  
- `reports/changelog/<yyyy-mm-dd>-<iteration-id>.md` — changelog по итерациям
- `docs/imports/index.yaml` — metadata index импортированных документов

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
- анализ arbitrary stacks через headless providers (`claude-code|qwen-code`) + baseline prompts/skills (без фиксированного whitelist language adapters в MVP)  
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
- System Analyst Q&A Agent отвечает на вопросы, используя:
  - `charter/cards/*`
  - `model/*`
  - `reports/*`
  - `docs/imports/*`
- Follow-up API фиксируется как read-only `POST /api/qa/ask`, который возвращает `answer`, `citations`, `unresolved`, `confidence`.

---

## 10. Регулярная работа (итерационный цикл)

### 10.1. Ручной режим MVP
В MVP обновления инициируются вручную:
- самый короткий старт: `acp serve --workspace ... --auto-init ((--repo-name ... (--repo-path ... | --repo-git-url ...) [--repo-ref ...]) | --repos-file ...) [--docs-imports-path ...] --runtime fake`
- первый bootstrap workspace выполняется через `acp init-workspace --workspace ... ((--repo-name ... (--repo-path ... | --repo-git-url ...) [--repo-ref ...]) | --repos-file ...)`
- первый materialization запуск: `acp run --workspace ... --pipeline init --runtime fake --non-interactive`
- для live запуска: `acp run --workspace ... --pipeline init --runtime headless --runtime-provider qwen-code --non-interactive` (или `claude-code`)
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

### 13.2. Контракт TaskResult
- Текущий changeset contract считаем замороженным для MVP.
- `write_file` в MVP не добавляется; Step 2 и Step 4 становятся agent-first шагами, но canonical publish по-прежнему выполняет только compiler/promoter слой.
- Observation без evidence запрещён policy и examples.
- Canonical MVP shape: runtime по умолчанию пишет `questions[]` и `coverage` на top-level.
- Legacy operations `add_question` и `set_coverage` удалены из active contract; принимается только canonical top-level representation.
- `add_doc_artifact` трактуется как metadata registration op, а не как content write op.

### 13.3. Skills/prompts editing через UI
- Минимальный UX MVP: edit → validate → commit → history → rollback.
- Валидация prompt bundle опирается на manifest checks и быстрые fixture dry-runs.

### 13.4. Extraction strategy
- MVP не ограничивается narrow whitelist языков/стэков.
- Анализ arbitrary stacks выполняется headless providers (`claude-code|qwen-code`) + baseline prompt/skill bundle.
- Если стек или артефакт не удаётся надёжно интерпретировать, система фиксирует gaps через coverage/questions/findings.

### 13.5. Документы и metadata index
- Raw imports хранятся в `docs/imports/`.
- Metadata фиксируется в `docs/imports/index.yaml` с source/checksum/timestamps/status.
- `workspace.yaml` получает отдельный schema-contract; layout workspace beyond repo sources и imports path не конфигурируется через manifest.

### 13.6. GitHub/GitLab trigger policy
- Auto-trigger по умолчанию: `push` в default branch.
- `merge request` / `pull request` updates идут как manual/preview trigger.
- Manual pipeline button/job обязателен.
- Required MVP CI/CD surface: CLI batch mode.
- Internal API trigger optional и допустим только для trusted local/private deployment.
- Debounce policy: один активный run на workspace, окно 5 минут, `last event wins`.

### 13.7. Q&A API follow-up
- Read-only endpoint резервируется как `POST /api/qa/ask`.
- Планируемый ответ: `answer`, `citations`, `unresolved`, `confidence`.

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
