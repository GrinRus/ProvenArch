# Architecture Change Review — superseded UX/UI exploration

Статус: **superseded historical exploration; not implementation acceptance**, 2026-07-15.

Единственный актуальный planned UX baseline —
[`UI_TASK_FIRST_PRODUCT_DESIGN.md`](UI_TASK_FIRST_PRODUCT_DESIGN.md). Этот документ сохранён для
decision traceability Epic 20; он не задаёт новые acceptance criteria и не должен использоваться
как визуальный источник реализации.

Письменные UX-правила, state model и contract boundaries ниже авторитетнее PNG-референсов.
Изображения задают композицию и визуальную иерархию, но не являются pixel-perfect контрактом.

## 1. Решение в одной фразе

ACP должен ощущаться как **Git-native review архитектурного знания**, а не как постоянная
панель внутреннего pipeline:

```text
read-only sources
  -> analysis run
  -> immutable run snapshot
  -> validator-approved automatic promotion -> dirty Current workspace
  -> Architecture Change Review (human Git acceptance, not a promotion gate)
  -> full workspace Git publication
  -> versioned architecture knowledge
```

Runtime promotion и human acceptance — разные границы. После validator-approved `init|refresh`
канонические artifacts автоматически продвигаются в Current workspace до пользовательского
review. Architecture Change Review не approve-ит runtime output и не управляет promotion; он
помогает понять уже promoted changes и принять осознанное решение о Git publication.

Главный объект интерфейса после анализа — проверяемый пакет архитектурных изменений. Runtime,
shards, logs и provider internals остаются важными, но открываются в специализированном Run
Studio. Текущая architecture knowledge base доступна в Knowledge, а evidence viewer работает
как общий contextual workbench.

## 2. Current и target surfaces

| Surface | Current implementation | Target responsibility |
| --- | --- | --- |
| Onboarding | Один pre-console экран с четырьмя одновременно видимыми секциями | Последовательная Guided Setup session, показывающая один текущий prerequisite |
| Primary navigation | `Source / Readiness / Charter / Analysis / Review / Proposals / Ask / Publish` | `Home / Runs / Knowledge / Changes` |
| Setup | Распределён между onboarding, Source, Readiness и Charter | Контекстная session до первого run; после настройки доступна из workspace menu |
| Runtime monitoring | Analysis плюс global active strip, inspector и activity drawer | Focused Run Studio; logs и raw telemetry только здесь или в Diagnostics |
| Review | Evidence queue, artifact explorer, Domain Map и Proposals на разных stages | Один Architecture Change Review с contextual Evidence Studio |
| Current architecture | Часть Review/Domain Map | Самостоятельный Knowledge destination с Atlas и list fallback |
| Ask | Обязательный stage | Global read-only utility с явным `Current workspace` context |
| Publish | Отдельный stage, selection выглядит как commit scope | Последний режим Changes; full workspace Git inventory и честное confirmation |

Старые Console V2 surfaces сохраняются до инкрементальной миграции. Реализация не должна
создавать скрытый параллельный shell или compatibility DOM только ради старых selectors.

## 3. Product framing и пользователи

### 3.1 Primary user — architect / tech lead

Top jobs:

- подключить один или несколько source repositories;
- запустить `init` или `refresh`;
- понять, что обнаружено, что неизвестно и чем это подтверждено;
- сравнить результат run с опубликованным Git baseline настолько, насколько позволяют
  authoritative artifact/Git surfaces;
- изучить findings, questions и proposals;
- опубликовать все workspace changes осознанным Git-действием.

### 3.2 Operator / maintainer

Top jobs:

- увидеть active/pending run, текущий step и реальный blocker;
- восстановиться после provider, permission, timeout или artifact contract failure;
- открыть shards, logs и retained evidence;
- не смешать execution failure с качеством последнего валидного snapshot.

### 3.3 Reviewer / stakeholder

Top jobs:

- читать архитектурный результат без знания pipeline internals;
- перейти от finding/entity/proposal к citations и source paths;
- отличить validated, partial, demo и unavailable evidence;
- понять publication risk без просмотра raw logs.

### 3.4 First-time evaluator

Top jobs:

- пройти deterministic `fake` walkthrough;
- понять read-only source и workspace write boundaries;
- не принять demo output за live architecture evidence;
- завершить first run без знания `step0..step4`.

## 4. Design principles

1. **Architecture knowledge is the product.** Run — способ получить знание, а не основная
   информационная архитектура.
2. **Evidence before telemetry.** Сначала вывод, coverage и provenance; runtime mechanics — по
   запросу.
3. **Context is never implicit.** Changes/Knowledge/Publish всегда называют источник: `Run
   snapshot`, `Current workspace` или `Published Git baseline` там, где он действительно
   доступен.
4. **Execution, evidence и publication независимы.** Failed active run не обнуляет last-good
   knowledge; succeeded run не означает human approval или Git publication.
5. **Documents look like documents.** Markdown и proposals рендерятся как читаемые документы,
   а не как raw `<pre>` внутри вложенных cards.
6. **One viewport — one decision.** Один dominant context и одна primary action; secondary
   diagnostics раскрываются постепенно.
7. **Honest affordances only.** Нет `Approve`, пока отсутствует persisted decision contract; нет
   selected-file commit, пока backend выполняет `git add -A`.
8. **Density follows the task.** Setup и review используют comfortable density; tables, diffs и
   logs — compact density.
9. **Color reinforces meaning.** Каждый status имеет text/icon/structure и не зависит только от
   цвета.
10. **Recovery is a first-class route.** Error state всегда сохраняет контекст и даёт реальное
    действие, а не generic `Try again`.

## 5. Object and source model

| UI object | Meaning | Source of truth |
| --- | --- | --- |
| Workspace | Local Git-tracked architecture workspace | selected server workspace/session |
| Published baseline | Workspace state at Git `HEAD` | authoritative Git read/diff surface; partial until sufficient API exists |
| Current workspace | Promoted local files, possibly dirty/unpublished | canonical workspace files + Git status |
| Analysis run | One `init` or `refresh` execution; primary Runs surface | persisted pipeline run history/status |
| QA run | Read-only `qa.ask` audit execution; shown in Ask history/Diagnostics, not as an architecture review package | persisted QA run history/status |
| Run snapshot | Immutable staged final evidence for one selected run | run-bounded `staged_path` and final index |
| Review package | UI composition of snapshot artifacts, findings, gaps, proposals and Git risk | derived frontend view; **not** a new persisted artifact contract |
| Evidence item | Artifact, citation, claim, entity, edge, finding or question | selected snapshot/current workspace artifacts |
| Architecture entity | Current Knowledge uses canonical `model/*`; historical Changes/Atlas uses selected run final-index semantic data | explicit current or snapshot source, never a silent fallback |
| Publication inventory | Every changed workspace path covered by commit, including new/modified/deleted/untracked/renamed/copied/changed statuses | authoritative full-workspace Git inventory |

### 5.1 Source identity rules

- Historical review defaults to `Run snapshot`.
- `Current workspace` is an explicit source switch, never an error fallback.
- Missing indexed staged content renders `Snapshot unavailable` and keeps run/path context.
- Optional content absent from the selected run renders `Not produced for this run`.
- `Published Git baseline` is shown only when backend data can prove the comparison. Otherwise
  the UI shows artifact/Git diff availability as partial and does not invent an entity-level delta.
- Current Git `HEAD` and clean/dirty state are workspace-global. Until a persisted `run_id ->
  commit` association exists, a row cannot claim that a particular run was published; its
  per-run publication label is `Unknown`.
- Before Epic 21 impact planning, `Changes` means workspace/run artifact changes. It must not
  claim causal source-code impact by file/domain unless an authoritative contract supplies it.

## 6. Target information architecture

```text
ACP workspace
├── Home
│   ├── Needs attention
│   ├── Current architecture summary
│   ├── Latest review package
│   └── Recent runs and publication state
├── Runs
│   ├── Active and pending
│   ├── History
│   └── Run Studio
├── Knowledge
│   ├── Overview
│   ├── Architecture Atlas
│   ├── Entities and relations
│   └── Artifacts
└── Changes
    ├── Review packages
    ├── Evidence and findings
    ├── Proposals
    ├── Diff
    └── Publish

Global utilities
├── Ask current workspace
├── Quick switcher (future; outside Epic 20 acceptance)
├── Workspace switcher and Setup
└── Settings / Diagnostics
```

### 6.1 Naming decisions

- `Home` означает operator home/attention surface. Это не artifact
  `reports/as-is/overview.md`, который в Epic 21 также называется Architecture Home.
- `Knowledge` означает текущую promoted architecture knowledge base. Dirty/unpublished state
  должен быть заметен.
- `Changes` означает review и publication выбранного run package относительно доступного
  workspace/Git context.
- `Setup` не занимает primary navigation после готовности workspace. До first run Guided Setup
  временно заменяет основной shell; позднее открывается из workspace menu.
- `Ask` не влияет на completion и не является обязательным destination.
- `Quick switcher` не является обязательной частью Epic 20. До отдельного scope GlobalHeader
  содержит только direct navigation/Ask; implementation не создаёт новый search index или source
  code search неявно.

### 6.2 Minimum URL state

Планируемые routes или equivalent history state:

```text
/setup?step=<workspace|sources|brief|runner|review>
/home
/runs
/runs/<run_id>
/knowledge?view=atlas&entity=<entity_id>&source=current
/changes?run=<run_id>&view=overview&source=snapshot
/changes?run=<run_id>&view=evidence&artifact=<document_id>&mode=rendered
/changes?run=<run_id>&view=publish
```

Back/Forward и reload должны восстанавливать setup step, destination, run, source mode,
artifact/entity и viewer mode. Missing IDs очищаются с visible notice; silent source fallback
запрещён. До first run `/setup` является top-level route без primary shell. Незаписанные drafts
остаются в существующей session/editor boundary; reload не обещает persistence, пока draft не
сохранён authoritative workspace API.

## 7. End-to-end journeys

### 7.1 First analysis

1. Пользователь создаёт или открывает workspace.
2. Guided Setup последовательно проходит `Workspace -> Sources -> Analysis brief -> Runner &
   readiness -> Review & start`.
3. `fake` рекомендован как walkthrough и постоянно маркируется `Deterministic demo`.
4. Analysis brief рекомендован, но `Run without brief` остаётся явным quality trade-off.
5. Readiness показывает effective server runtime/provider/permission mode, а не только client
   selection.
6. `Run init` открывает Run Studio.
7. После terminal success Home показывает новый review package и primary action
   `Review architecture changes`.
8. Changes открывает выбранный `Run snapshot`, findings, gaps, proposals и evidence.
9. Пользователь может открыть Knowledge Atlas и Evidence Studio, сохраняя исходный review context.
10. Publish загружает полный Git inventory.
11. Пользователь подтверждает `Commit all workspace changes`; для fake используется отдельный
    `Commit all demo workspace changes` confirmation.
12. После commit Home показывает clean workspace Git state и следующий возможный refresh.

### 7.2 Repeat refresh

1. Home показывает current knowledge, authoritative workspace `HEAD` when available, dirty state
   и latest analysis run; он не приписывает commit конкретному run без persisted association.
2. Пользователь выбирает `Refresh analysis`.
3. Run Studio становится dominant destination для active execution.
4. После success пользователь возвращается в новый Architecture Change Review.
5. Last-good Knowledge остаётся доступным во время нового run и при его failure.
6. После review пользователь публикует full workspace changes.

### 7.3 Historical review

1. Runs -> выбрать analysis run.
2. Successful `init|refresh` с valid final index открывает Changes с `source=snapshot` и identity
   выбранного run.
3. Failed/canceled/reconciled analysis открывает Run Studio с retained evidence; partial review
   доступен только при реально существующем authoritative index и маркируется `Partial`.
4. QA run открывается в Ask history, а не в Architecture Change Review.
5. Snapshot bytes, coverage и questions не смешиваются с current canonical files.
6. Если indexed file отсутствует, review остаётся на выбранном run и показывает
   `Snapshot unavailable`.

### 7.4 Ask

1. Ask открывается из global header, entity, finding или artifact.
2. Context всегда подписан `Current workspace · read-only` до появления отдельного run-scoped
   Q&A contract.
3. Answer показывает confidence, citations и unresolved assumptions.
4. Citation открывает Evidence Studio и возвращает focus/selection после закрытия.
5. QA status не меняет Changes или Publish gate.

## 8. Recovery journeys

### 8.1 Setup and runtime

- Invalid manifest -> показать field/repo diagnostics -> исправить -> повторить validation.
- Provider unavailable -> exact executable/auth/quota guidance -> `Recheck readiness` или
  `Use fake walkthrough`.
- Desired runtime отличается от effective -> `Pending restart`, exact restart instruction и
  readback после reconnect.
- Immediate runtime selection остаётся только launcher/first-run Setup exception до входа в
  Console и без active/pending run. Внутри Console desired value — Settings-session draft, а не
  effective switch; reload persistence не обещается без отдельного persisted process-preference
  contract.
- API unavailable -> persistent offline banner; сохранить route, selected run, form drafts и
  prepared commit message.

### 8.2 Run Studio

| Condition | Default explanation | Primary recovery |
| --- | --- | --- |
| `runner_unavailable` | Provider command/auth/quota недоступен | `Open readiness` |
| `runtime_permission_required` | Request не разрешён policy | `Review permission settings`; не показывать approve/deny без broker contract |
| `runtime_timeout` | Step/shard превысил effective budget | `Review retained evidence` или `Run again` |
| `runtime_contract_failed` | Required artifact отсутствует или невалиден | `Open artifact diagnostics` |
| partial shard failure | Succeeded и failed shards разделены; output triage-only | `Inspect failed shards` |
| `run_canceled` | Cooperative cancel; taskrun evidence сохранено | `Start again` |
| restart reconciliation | Running run auto-resumed by the service or terminalized with retained evidence | `Observe resumed run`, `Review retained evidence` или `Run again` according to authoritative status |

Start request failure не очищает last-good evidence selection. Active run не допускает обычный
double-start. Queue action показывается только при authoritative pending/supersession readback.

### 8.3 Evidence

- Initial load -> skeleton сохраняет ожидаемую geometry.
- Refresh -> уже прочитанный valid content остаётся видимым с inline pending indicator.
- Parse/render error -> `Raw` доступен, если bytes успешно загружены.
- Large content -> progressive rendering/virtualization; copy/download без молчаливого truncation.
- Entity/edge without evidence -> explicit gap/question; Atlas не создаёт guessed linkage.

### 8.4 Publish

- Git inventory load failed -> commit blocked; `Retry inventory`.
- Dirty file вне выбранного run -> обязательно входит в full-workspace inventory.
- Open questions -> visible risk + commit-attempt-scoped acknowledgement, не скрытый hard blocker.
  Acknowledgement сбрасывается при изменении inventory, branch, source context или question count
  и не сохраняется как review/approval decision.
- Commit failed -> сохранить message/confirmation context; показать error и retry.
- No changes -> `Workspace already clean`.
- Proposal branch failure не изменяет commit state и имеет собственное recovery.

## 9. Four-axis state model

Ни одна ось не сворачивается в общий недостоверный `READY`.

| Axis | States | Meaning |
| --- | --- | --- |
| Workspace/readiness | `unconfigured`, `draft`, `validating`, `invalid`, `doctor_unknown`, `checking`, `doctor_failed`, `provider_unavailable`, `pending_restart`, `ready` | Возможность настроить workspace и стартовать новый analysis run |
| Execution coordination | `none`, `starting`, `queued`, `running`, `canceling`, `succeeded`, `partial_failed`, `failed`, `canceled`, `recovered` | Active/pending workspace execution; outcome выбранного historical run живёт отдельно в ContextBar |
| Evidence | `not_produced`, `loading`, `unavailable`, `error`, `partial`, `available`, `blocked`, `package_complete`, `stale` | Доступность и validator/package trust selected review evidence; не human approval |
| Publication | `unknown`, `inventory_loading`, `clean`, `changes`, `ready_with_risks`, `blocked`, `confirming`, `committing`, `committed`, `failed` | Full-workspace Git boundary |

Cross-cutting facets, которые могут сочетаться с каждой осью:

```text
source identity: Run snapshot {run_id, generated_at, outcome, runtime_mode, provider|Mixed}
              | Current workspace {git state, promoted_from_run where known}
              | Published Git baseline {commit, only when authoritative}
transport:       online | reconnecting | offline
attention:       none | info | warning | blocker
evidence origin: live | deterministic_demo
```

`stale` означает только «selected snapshot не является latest known promoted snapshot». Он не
утверждает, что source architecture устарела: такой вывод требует impact/source revision contract.
При `offline` все server-derived значения маркируются `Last known`, но не переписываются в другой
workspace/evidence lifecycle state.

### 9.1 Derivation and precedence

Один pure typed selector получает только authoritative read models: workspace validation/doctor,
active/pending run, selected run identity/final index, validator result, required artifact presence,
open-question count, effective live/demo identity и Git inventory/mutation result.

| Authoritative condition | Evidence result | Publication result | Primary attention/action |
| --- | --- | --- | --- |
| No configured workspace | `not_produced` | `unknown` | blocker -> `Open Setup` |
| Validation/doctor is unknown, running or failed | keep last known evidence | `blocked` only for new mutation/start where required | `Validate`, `Wait` or exact recovery |
| Selected snapshot index mismatches run, escapes staging or indexed bytes cannot load | `unavailable` or `error` | `blocked` for snapshot-dependent handoff | `Review snapshot diagnostics` |
| Required final artifact/validator contract fails | `blocked` | `blocked` | `Open artifact diagnostics` |
| Valid required package has missing optional material | `partial` | `ready_with_risks` if Git inventory is authoritative | `Review gaps` |
| Valid required package has open questions | `available` or `package_complete` | `ready_with_risks` | `Review questions` |
| Valid live package and authoritative dirty inventory | `package_complete` | `changes` then `confirming` | `Continue to Publish` |
| Valid fake package and authoritative dirty inventory | `package_complete` + `deterministic_demo` | `ready_with_risks` | distinct demo confirmation |
| Git inventory unavailable | keep evidence state | `blocked` | `Retry inventory` |
| Authoritative workspace is clean | keep evidence state | `clean` | no commit action |

Composition rules:

- Home показывает четыре оси одновременно, но выделяет одну primary next action.
- Navigation badges, Home, Changes и Publish используют один typed workflow selector.
- Failed run не скрывает last-good Knowledge или historical review.
- `package_complete` никогда не отображается как `Approved` или `Reviewed`.
- `Deterministic demo` identity и `Demo evidence` trust label показываются вместе; demo никогда
  не наследует live-ready semantics.
- Open questions обычно дают `ready_with_risks`; hard blocker требует явного documented rule.
- Runtime telemetry, matrix verdict files и release counters не решают product publication gate.

## 10. Screen inventory

### 10.1 Guided Setup session

Purpose: подготовить один анализ без показа всей конфигурации одновременно.

Steps:

1. Workspace.
2. Sources and imports.
3. Recommended analysis brief: scope, NFR, domains/teams, rules.
4. Runner and readiness.
5. Review & start.

Каждый шаг показывает один blocker, one primary action, completed summary предыдущих шагов и
`Expert settings` disclosure. После first run Setup открывается из workspace menu и не занимает
primary navigation. URL/history сохраняет текущий step; Back не теряет уже persisted workspace
values. Несохранённые field drafts живут только в существующей editor/session boundary и должны
явно предупреждать перед destructive navigation.

### 10.2 Home — attention and architecture summary

Purpose: за 10 секунд ответить:

- какой workspace открыт;
- есть ли active/pending run;
- что требует внимания;
- какое architecture knowledge сейчас доступно;
- есть ли unpublished workspace changes;
- что делать дальше.

Sections:

- compact four-axis summary;
- ordered `Needs attention` queue: hard blockers -> active run -> review risks -> publication;
- current architecture summary and Knowledge entry;
- latest review package;
- recent runs and publication activity.

Не содержит full logs, shard table или raw paths.

### 10.3 Runs list

Purpose: найти active, pending или historical execution.

Must show:

- `init|refresh` pipeline, run ID, start/finish, effective runtime/provider, result;
- filters by lifecycle, pipeline and provider;
- active/pending identity and queue semantics;
- newest active selection, otherwise latest terminal.

QA audit runs остаются в Ask history и Diagnostics; primary Runs не ведёт их в architecture
pipeline/shard UI.

### 10.4 Run Studio

Purpose: наблюдать, диагностировать и восстановить execution.

Must show:

- sticky run identity header;
- accessible ordered pipeline track;
- current step/shard and last useful progress;
- one blocker/recovery panel;
- shard matrix and artifact counts;
- cancel and explicit queue actions;
- contextual technical drawer for logs, raw output, permissions and runtime internals.

После idle/success Run Studio не оставляет global log drawer открытым на других destinations.

### 10.5 Knowledge

Purpose: читать текущее architecture knowledge независимо от выполнения.

Views:

- `Overview`: rendered as-is and coverage summary;
- `Atlas`: domain/entity graph with coverage/confidence overlays;
- `Entities`: searchable list/table equivalent;
- `Artifacts`: expert explorer.

Knowledge header показывает current workspace Git state и promoted-from-run identity when known.
Sparse model получает partial banner; graph не считается доказательством полноты.

### 10.6 Architecture Atlas

Must show:

- search and filters by entity type/domain/repo/owner;
- domain groups, services, datastores, external systems and teams;
- selected entity purpose, owner, relations, findings, questions and provenance;
- coverage and confidence overlays;
- conditional change overlay только когда authoritative semantic diff contract существует;
  до этого control disabled с reason и доступен лишь artifact/Git-level diff в Changes;
- explicit validated/partial/missing legend;
- full list/table alternative для keyboard and screen-reader use.

### 10.7 Changes list

Purpose: выбрать review package.

Each row:

- run identity and generated time;
- live/demo identity;
- execution outcome;
- artifact/finding/question/proposal counts;
- workspace publication relevance: current dirty/clean state where authoritative, otherwise
  per-run `Unknown`; не показывать `Published` без persisted run-to-commit association;
- honest partial/unavailable snapshot state.

### 10.8 Architecture Change Review

Purpose: понять результат run и подготовить решение о публикации.

Subviews:

- `Overview` — architecture synopsis, coverage, what needs attention;
- `Evidence` — artifact and finding queue;
- `Findings` — severity, affected surfaces, provenance;
- `Proposals` — linked recommendations and residual gaps;
- `Diff` — artifact/Git diff at authoritative available granularity;
- `Publish` — full workspace handoff.

Header always shows source mode, run ID, generated time and live/demo identity. No disabled
`Approve` controls. Real actions: `Open evidence`, `Investigate`, `Open proposal`,
`Continue to Publish`.

### 10.9 Evidence Studio

Reusable contextual workbench opened from Changes, Knowledge, Atlas, Ask or Publish.

Anatomy:

- human title and object type;
- source context and run/current identity;
- `Rendered / Raw / Diff` modes;
- rendered Markdown/Mermaid/YAML view;
- citations and provenance chain;
- linked entities, findings, questions and proposals;
- full-screen mode without losing underlying route/selection.

Logs appear only in Run Diagnostics, not beside normal architecture reading.

### 10.10 Publish

Purpose: make the full-workspace Git mutation explicit.

Must show:

- evidence context and separate commit scope;
- shared publication gate;
- complete new/modified/deleted/untracked/renamed/copied/changed inventory with explicit counts;
- binary/no-hunk, unavailable diff and bounded large-diff states without hiding commit scope;
- branch and base identity;
- selected artifact preview labelled as preview only;
- open-question/demo warnings;
- prepared commit message;
- separate proposal-branch action;
- confirmation and mutation result.

Primary action: `Commit all workspace changes`. Fake variant: `Commit all demo workspace
changes`.

### 10.11 Global Ask

Desktop: modal contextual side sheet. Mobile: full-screen modal dialog. Ask всегда имеет scrim,
focus trap, Escape, explicit close и focus return; wide non-modal Evidence/Atlas panes используют
другой drawer variant без focus trap.

Must show:

- `Ask current workspace · read-only`;
- composer, history, selected answer;
- runtime/provider identity;
- confidence, citations and unresolved assumptions;
- same-question retry and retained audit evidence after failure.

### 10.12 Settings and Diagnostics

Contains runtime profile, effective/desired values, permissions, timeouts, execution strategy,
server/version information and expert diagnostics. It is not a mandatory workflow destination.

## 11. Interaction contract

- Each destination has one primary action; recovery can replace it.
- Mutating Git controls exist only in Publish.
- Disabled controls show a visible reason adjacent to the control, not only in `title`.
- Snapshot/current switch is explicit, URL-restorable and aborts or suppresses stale responses.
- During content refresh, last valid content remains visible.
- Run start is protected against double-click; active run blocks ordinary start actions.
- Backend отклоняет ordinary start while active; только explicit queue intent может создать или
  заменить единственный pending run. Queue confirmation names active run, pending pipeline and
  typed last-event-wins replacement identity.
- Cancel confirmation explains cooperative cancellation and retained taskrun evidence.
- Citation navigation preserves origin selection and focus.
- Atlas selection never makes the visual layout itself a semantic fact.
- Internal workspace links route within the application; source refs show repo/path/ref.
- Raw HTML in rendered Markdown stays disabled.
- Offline/reconnect preserves workspace, route, source mode, drafts and prepared Git message.
- Commit-risk acknowledgement is attempt-scoped and resets when inventory, branch, source context
  or open-question count changes.
- Modal sheet/dialog uses scrim, focus trap, Escape and focus return. Wide persistent/non-modal
  pane has no scrim or focus trap and remains in normal page focus order.
- Success/error results are announced once and remain inspectable.

## 12. Visual system

### 12.1 Visual character

The visual system evolves the current identity:

- warm light canvas;
- deep navy navigation;
- restrained teal action/trust accent;
- graphite editorial documents;
- thin separators and whitespace instead of nested cards;
- no gradients, glow, stock imagery or sci-fi dashboard styling.

The primary visual unit is a context-labelled evidence/review object, not a pipeline stage card.

### 12.2 Semantic color aliases

Current `ui/src/styles.css` variables remain the implementation source until a code slice migrates
them. Target semantic aliases:

```css
--color-bg-canvas: var(--bg);                 /* #f6f8fa */
--color-bg-surface: var(--surface);           /* #ffffff */
--color-bg-subtle: var(--surface-muted);      /* #f9fafb */
--color-bg-inverse: var(--sidebar-bg);        /* #061923 */

--color-text-primary: var(--ink);             /* #17202a */
--color-text-secondary: var(--ink-muted);     /* #5f6b7a */
--color-text-tertiary: var(--ink-faint);      /* #8894a3 */
--color-text-inverse: var(--sidebar-ink);

--color-border-default: var(--line);          /* #d8dee4 */
--color-border-subtle: var(--line-soft);      /* #eaeef2 */
--color-border-strong: #b9c2cc;

--color-action-primary: var(--accent);        /* #0f766e */
--color-action-primary-hover: #0b655f;
--color-action-primary-active: #084c47;
--color-action-on-primary: #ffffff;
--color-link: var(--info);                    /* #2563eb */

--color-state-success: var(--ok);
--color-state-success-bg: var(--ok-soft);
--color-state-success-border: var(--ok-line);
--color-state-warning: var(--warn);
--color-state-warning-bg: var(--warn-soft);
--color-state-warning-border: var(--warning-panel-line);
--color-state-danger: var(--err);
--color-state-danger-bg: var(--err-soft);
--color-state-info: var(--info);
--color-state-info-bg: var(--info-soft);
--color-focus: var(--focus-ring);
--color-scrim: rgba(6, 25, 35, 0.42);
```

Context roles:

```css
--color-context-snapshot: var(--info);
--color-context-snapshot-bg: var(--info-soft);
--color-context-current: var(--ink-muted);
--color-context-current-bg: var(--surface-muted);
--color-context-published: var(--ok);
--color-context-published-bg: var(--ok-soft);
--color-context-live: var(--accent);
--color-context-live-bg: var(--accent-soft);
--color-context-demo: var(--warn);
--color-context-demo-bg: var(--warn-soft);
```

Atlas roles add service, datastore, external, team and unknown colors, but entity type must also
use icon/shape/text. Coverage and confidence overlay precedence is structural: selection outline
first, missing/partial pattern second, confidence text/icon third. Future change overlay must not
replace those trust signals.

### 12.3 Typography

```text
font-sans   Inter, IBM Plex Sans, Segoe UI, Arial, sans-serif
font-mono   SFMono-Regular, Consolas, Liberation Mono, monospace

12px / 16  caption, status metadata
13px / 18  compact table/list labels
14px / 21  default UI body
16px / 26  rendered evidence body
20px / 28  section title
24px / 32  page title
28px / 34  rare primary metric
```

- No text below 12px.
- Default weights: 400, 600, 700.
- Rendered evidence maximum measure: `76ch`.
- Paths, IDs and timestamps use mono only as supporting metadata.
- Sentence case; uppercase limited to short system labels.

### 12.4 Spacing, density and sizing

4px base scale:

```text
4, 8, 12, 16, 20, 24, 32, 40, 48
```

- desktop page gutter: 24px; tablet 16px; phone 12px;
- section gap: 24–32px;
- related control gap: 8px;
- default control height: 40px;
- compact operational control: 32px;
- mobile/touch control target: 44px;
- default list row: 48–56px;
- compact data row: 36px.

Comfortable density: Guided Setup, Home, Knowledge, review documents. Compact density: Run
Studio matrices, logs, Git inventory and diff.

### 12.5 Radius and elevation

```text
control 6px
panel   8px
sheet  12px
pill  999px
```

Default sections use borders and whitespace. Shadows are reserved for sticky drawers/dialogs.
Bordered cards must not nest unless the inner object has an independent interaction or lifecycle.

### 12.6 Motion

```text
fast    120ms
normal  180ms
slow    240ms
easing  cubic-bezier(.2, 0, 0, 1)
```

No route fly-ins. Looping motion is reserved for real indeterminate activity. Reduced-motion
mode removes transforms, loops and smooth scrolling.

## 13. Component families

| Family | Purpose and important variants |
| --- | --- |
| `GlobalHeader` | Brand, workspace switcher, Ask, server indicator, utility menu; quick switcher only in a future scoped slice |
| `PrimaryNav` | Four destinations; expanded/collapsed desktop, bottom mobile |
| `ContextBar` | Snapshot/current/published source, run identity, live/demo, stale/partial |
| `PageHeader` | Title, purpose, state and one primary action |
| `ContextDrawer` | `persistent_nonmodal`, `overlay_modal`, `fullscreen_modal`; explicit scrim, outside-click and focus behavior per variant |
| `AttentionList` | Ordered blocker/run/review/publication items; empty/loading/error states |
| `RunList` | Compact history with filters and lifecycle/provider identity |
| `PipelineTrack` | Accessible ordered steps; horizontal desktop, vertical mobile |
| `ShardMatrix` | Sort/filter, compact rows, mobile keyed cards, partial/error states |
| `ArtifactViewer` | Rendered/Raw/Diff, Markdown/Mermaid/YAML, large/error/unavailable states |
| `ProvenanceChain` | Claim -> citation -> repo/path; confidence and missing evidence |
| `ArchitectureAtlas` | Map plus mandatory list/table equivalent and partial banner |
| `ReviewQueue` | Findings/questions/proposals/artifacts with severity and evidence count |
| `ChangeInventory` | New/modified/deleted/untracked/renamed/copied/changed groups, binary/large/unavailable diff; full-scope semantics |
| `PublishGate` | Blockers, risks, demo identity and readiness from shared selector |
| `ConfirmationDialog` | Scope, branch, counts, warning acknowledgement, focus return |
| `AskPanel` | Current workspace identity, answer/citations/unresolved/history |
| `AsyncState` | Loading, empty, partial, stale, offline, error, recovered |

All interactive families specify hover, active, focus-visible, disabled, loading and selected
states. Async/destructive results use labelled live regions. StatusBadge and ContextBadge are
separate: source identity must not be confused with success/failure.

## 14. Responsive contract

### 14.1 Wide desktop — 1280px and above

- header: 56px;
- navigation: 216px expanded or 64px collapsed;
- content gutter: 24px;
- contextual drawer: 340–400px;
- application max width: 1600px;
- three-pane review appears only when the user opens provenance; it is not the default shell.

### 14.2 Compact desktop/tablet — 901–1279px

- collapsed 64px navigation by default;
- contextual drawer overlays content;
- at most two persistent content columns;
- pipeline remains horizontal only while labels remain readable;
- workbench target width at 1024px: at least 720px.

### 14.3 Tablet — 681–900px

- bottom navigation with four destinations; no side rail at this breakpoint;
- reserve bottom safe area so primary actions and sheets do not overlap navigation;
- one main column;
- queue/inventory open as drawers;
- pipeline becomes vertical;
- data tables become keyed rows unless comparison requires internal scroll;
- code/log/diff scroll is local, never document-level.

### 14.4 Phone — up to 680px, canonical 390x844

- header: 56px;
- bottom navigation: four destinations, 60–64px;
- touch targets: minimum 44px;
- primary action stays visible above bottom navigation;
- review list and selected review object become separate routes;
- Atlas defaults to searchable entity list; map is optional;
- Ask, context drawer and confirmation use full-screen sheets;
- title, source context, state and primary action fit before `y=520`;
- long paths wrap or middle-truncate with accessible full-value/copy action;
- no document-level horizontal overflow.

## 15. Accessibility contract

- WCAG 2.2 AA minimum.
- Landmarks, skip link, one `h1`, logical heading order.
- Tabs: roving `tabindex`, Arrow keys, Home/End, `aria-controls` and linked panels.
- Modal dialogs/drawers: labelled title/description, scrim, focus trap, Escape and focus return.
- Non-modal persistent panes: landmark/label, normal focus order, no focus trap; close returns focus
  only when the pane was explicitly opened by a trigger.
- Forms: `aria-invalid`, `aria-describedby`, field errors plus summary.
- Async completion/error: restrained `aria-live`, no duplicate announcements.
- Status uses icon/text/shape in addition to color.
- Atlas has keyboard-operable list/table equivalent for every entity and relation.
- Visible focus survives dark and light surfaces.
- Zoom to 200% preserves task completion.
- Logs do not auto-scroll without pause when the operator has moved away from the end.
- Long paths remain copyable and do not create page overflow.

## 16. Retired reference screens

The former seven-screen PNG set was removed on 2026-08-11 to prevent it competing with the
task-first target. The current 13-screen inventory is maintained only in
[`UI_TASK_FIRST_PRODUCT_DESIGN.md`](UI_TASK_FIRST_PRODUCT_DESIGN.md).

## 17. Implementation sequence and Epic 20 mapping

This design refines Epic 20 rather than creating a competing roadmap.
Detailed delivery phases, contract-first decisions, code/test ownership, cutover rules and the
complete comparison-link matrix are fixed in
[`UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md`](UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md).
That plan sequences the existing `20A–20N`; it does not create a second backlog.

| Foundation | Design dependency |
| --- | --- |
| Run-pinned snapshot truth | Source identity and historical Changes review |
| Honest Git inventory | Publish and full-scope confirmation |
| Effective runtime identity | ContextBar, Run Studio and demo/live trust |
| Shared workflow selector | Four-axis summary and non-contradictory gates |
| Evidence-first viewer | Changes, Knowledge, Ask citations and Publish preview |
| Navigation/URL state | `Home / Runs / Knowledge / Changes` and source restoration |
| Semantic components/tokens | Visual system and density modes |
| Responsive shell | Context drawer, mobile routes and bottom navigation |

Recommended vertical prototype order:

1. Shared state/source view models and truthful snapshot/Git/runtime identities.
2. New shell with Home and URL-restorable four-destination navigation.
3. Run Studio migration with contextual diagnostics.
4. Architecture Change Review plus shared ArtifactViewer/Evidence Studio.
5. Knowledge Overview/Atlas with list fallback and partial states.
6. Publish confirmation and Git recovery.
7. Guided Setup migration, global Ask and final responsive/accessibility gate.

No big-bang rewrite is required. Touched vertical slices should extract view models and containers
from `StagePanels.tsx` while preserving accepted backend/runtime behavior.

### 17.1 Incremental shell bridge

Ни один релиз не содержит два скрытых shell. Migration landing заменяет navigation только после
того, как все четыре target destinations имеют честный временный composition:

| Migration phase | Target route | Temporary implementation input | Retirement condition |
| --- | --- | --- | --- |
| Before `20I1` | none | Console V2 remains the only shell | shared workflow/source models accepted |
| `20I1` shell landing | `/home` | new attention summary over existing authoritative hooks | Home state matrix + responsive tests pass |
| `20I1` shell landing | `/runs` | current Analysis mission-control container, relabelled only where behavior matches | Run Studio extraction accepted |
| `20I1` shell landing | `/knowledge` | authoritative current-workspace inventory/list when available, otherwise explicit partial/unavailable state; no filename-derived Domain Map | Knowledge/Atlas container accepted |
| `20I1` shell landing | `/changes` | current Review + Proposals + Publish composition with honest source/Git labels | Change Review/Evidence Studio/Publish containers accepted |
| `20J–20M` | all four | replace temporary containers one vertical slice at a time | no legacy stage-only selector remains |
| `20N` | all four | target shell only | old rail/stage routing and migrated tests removed |

Guided Setup remains the only pre-shell entry, and current Ask behavior is wrapped as a global
utility only when its route/history/focus behavior is preserved. Temporary compositions are visible
product surfaces, not hidden compatibility DOM.

## 18. Contract dependencies and open decisions

P0 before truthful implementation:

1. Confirm full new/modified/deleted/untracked/renamed/copied/changed Git inventory,
   binary/unavailable-diff behavior, branch/base readback and content-bearing inventory
   fingerprint; commit/branch mutations must reject stale confirmed fingerprint/HEAD before side
   effects. Otherwise land a contract-only slice before Publish UI.
2. Confirm effective server runtime/provider/permission readback and persisted historical run
   identity.
3. Confirm authoritative single-pending run/pipeline/supersession readback and explicit queue
   command intent before showing queue controls.
4. Keep run-bounded staged snapshot reads fail-closed and expose the source mode visibly.
5. Treat publication as workspace-global until a persisted run-to-commit association exists;
   historical rows remain `Unknown` rather than inferring acceptance from Git state.

P1 for richer change review:

1. Choose the comparison baseline for future semantic diff: Git `HEAD`, previous successful run
   or explicit user-selected run.
2. Decide whether MVP Changes shows artifact/Git diff only or adds a deterministic
   entity/edge/finding diff contract.
3. Define which proposal artifacts are required versus legitimately not produced.
4. Decide whether provider outage blocks only new runs or also affects publication of already
   valid evidence.

Future capability:

- persisted human review decision;
- exact file-scoped commit;
- run-scoped Ask;
- UI approve/deny permission broker;
- source-impact planning from Epic 21.
- run-to-commit publication association and per-run published state.

Until those contracts exist, the UI must not use `Approved`, selected-file commit copy,
run-scoped Ask labels or source-impact claims.

## 19. Acceptance checklist

- Primary IA is `Home / Runs / Knowledge / Changes`; Setup is contextual and Ask is global.
- First-time fake walkthrough always shows `Deterministic demo` after run completion.
- Home shows four independent axes and one non-contradictory next action.
- Failed active run does not make last-good Knowledge or historical snapshots unavailable.
- Two runs with identical canonical paths render their own staged bytes.
- Snapshot failure never falls back to current workspace.
- Changes, Knowledge/Atlas and Publish share one Rendered/Raw/Diff Evidence Studio.
- No evidence/proposal approval control exists without persisted review state.
- Atlas partial/missing evidence is at least as prominent as validated topology.
- Ask always says `Current workspace · read-only` and does not affect workflow completion.
- Publish shows full new/modified/deleted/untracked/renamed/copied/changed inventory and calls the mutation
  `Commit all workspace changes`.
- Demo commit uses distinct wording and confirmation.
- No Git mutation is available outside Publish.
- Back/Forward/reload restore destination, run, source mode and selected evidence/entity.
- Offline/reconnect preserves route and unsent local UI drafts.
- No document-level overflow at 1440, 1280, 1024 or 390x844.
- Mobile title, source identity, state and primary action fit in the first viewport.
- Setup, Run recovery, Evidence Studio and Publish are completable keyboard-only.
- Automated accessibility tests have no critical violations.
- Required CI remains deterministic, fixture-driven and network-free.
- Source repositories remain read-only; product writes only to the architecture workspace.

## 20. Non-goals

- No hosted or multi-tenant mode.
- No security/compliance enforcement UI.
- No source-repository writes.
- No hidden schema/API change inside a visual implementation slice.
- No generated PNG treated as production CSS specification.
- No full visual rebrand detached from trust, source and state semantics.
- No automatic acceptance of runtime-authored architecture decisions.
