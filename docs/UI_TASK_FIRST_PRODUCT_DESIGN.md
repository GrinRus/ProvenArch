# Task-first ProvenArch — целевой UX/UI baseline

Статус: **authoritative planned target**, 2026-08-11.

Этот документ определяет следующий целевой UX ProvenArch. Он заменяет прежние направления
Console V2 и Architecture Change Review как источник UI-решений. Текущий binary пока сохраняет
реализованный shell `Home / Analyze / Architecture / Changes`; переход выполняется отдельными
reviewable slices из Epic 23.

Письменные правила, продуктовая модель и state contracts авторитетнее PNG. Изображения в
`docs/assets/ui-task-first-product/` фиксируют композицию, плотность и визуальный характер, но не
являются pixel-perfect, copy или data-contract oracle.

## 1. Решение

ProvenArch становится **task-first инструментом архитектурного исследования**:

```text
user goal + scope + runner preset
              |
              v
          durable Task
              |
              v
     immutable execution Attempt
              |
              v
 validator-approved workspace knowledge
              |
              +--> Architecture: read and trace evidence
              +--> Changes: understand semantic/Git changes
              +--> Publish: explicit full-workspace Git action
```

Главный пользовательский объект — `Task`: что нужно понять и зачем. Runtime `run` становится
`Attempt`: одной аудируемой попыткой выполнить Task. Pipeline steps, shards, raw logs и permission
telemetry не исчезают, но живут только внутри Attempt/Pipeline Studio.

## 2. Почему прежний UI больше не является target

- `Home` повторяет Tasks, Architecture и Changes, но не даёт собственной основной работы.
- `Analyze` смешивает запуск, историю, terminal outcome, retry, artifacts и diagnostics.
- runner выбирается в setup/settings, а не в момент постановки задачи.
- terminal run продолжает выглядеть как активный pipeline.
- review copy иногда предлагает решить, заменять ли current snapshot, хотя validator-approved
  promotion уже произошёл до human Git review.
- Markdown, YAML, Mermaid, evidence, semantic model и Git diff показаны как соседние технические
  поверхности вместо единого knowledge workflow.

Новый baseline не является визуальным rebrand поверх старой IA. Он меняет ownership экранов и
последовательность решений.

## 3. Users и top jobs

### Architect / tech lead

- сформулировать архитектурный вопрос или цель;
- выбрать репозитории, scope и runner;
- понять semantic outcome раньше файлов и logs;
- проверить evidence, unknowns, findings и proposals;
- опубликовать полный architecture workspace осознанным Git-действием.

### Operator / maintainer

- видеть active/pending Attempt и один реальный blocker;
- различать provider activity и durable pipeline progress;
- повторить только failed scope либо следующий attempt с другим runner;
- сохранить last-good Architecture при failed Attempt.

### Reviewer / stakeholder

- читать Architecture Home, документы, схемы и findings без runtime vocabulary;
- переходить от claim/entity/finding к точным repo/path citations;
- понимать demo/live/partial/current/run-snapshot authority.

### First-time evaluator

- пройти guided setup и deterministic demo без знания pipeline step IDs;
- понять read-only source boundary и workspace write boundary;
- создать первую Task и отличить demo evidence от live analysis.

## 4. Product object model

| Object | Пользовательский смысл | Lifecycle и authority |
| --- | --- | --- |
| Workspace | Git-tracked база архитектурного знания | один attached workspace на local service session |
| Task | устойчивое намерение: goal, scope, runner preset, attempts, review state | новый contract; до реализации не выводится из run name или brief эвристиками |
| Attempt | immutable execution конкретной Task | соответствует pipeline run; config snapshot не меняется после admission |
| Runner preset | demo/live provider, model/effort и optional per-step overrides | выбирается до Attempt; изменение действует только на следующий Attempt |
| Architecture | validator-approved current workspace knowledge | автоматически promoted pipeline; не зависит от human Git review |
| Change set | semantic changes и full workspace Git inventory | semantic review и Git publication остаются разными представлениями |
| Artifact | Markdown, YAML, JSON или Mermaid workspace file | current может быть editable; run snapshot всегда read-only |
| Evidence authority | `promoted_current`, `run_snapshot`, `qa_snapshot`, `qa_audit` | всегда видима; нет silent fallback между authorities |

`Task` требует отдельного schema/API slice. До его принятия frontend не должен симулировать Task
через local-only labels поверх runs.

## 5. Information architecture

### Primary navigation

1. `Tasks` — default destination, New Task, inbox, active/attention/completed work.
2. `Architecture` — current validated map, documents, model and findings.
3. `Changes` — task/run-pinned semantic review, evidence, files and publication.

### Global utilities

- `Ask this architecture…` — current workspace, read-only until explicit proposal action.
- `Settings` — workspace, sources, runner presets, runtime advanced, Git and diagnostics.
- workspace switch/attach — launcher/setup context, не primary daily navigation.

`Home` и `Analyze` удаляются как primary destinations. New Task является явной action, а
Pipeline Studio открывается из Task Attempt, не из global nav.

### Target routes

```text
/setup?step=workspace|sources|runner|review
/tasks
/tasks/new
/tasks/<task_id>
/tasks/<task_id>/attempts/<run_id>
/architecture?view=map|documents|model|findings
/architecture/document?path=<workspace-path>&mode=rendered|source|diff
/architecture/entity?id=<entity-id>
/changes?task=<task_id>&run=<run_id>&view=summary|evidence|files|publish
/settings?section=workspace|sources|runners|runtime|git|diagnostics
```

Selected Task, Attempt, artifact/entity, evidence authority, viewer mode и filters восстанавливаются
через URL. Explicit stale identity fail-closed: показывается notice, но не подставляется другой run
или current workspace.

## 6. Canonical journeys

### 6.1 First analysis

1. Setup выбирает/создаёт workspace.
2. Sources добавляет один или несколько read-only repositories.
3. Runner предлагает `Deterministic demo`, `Claude Code`, `Qwen Code`, `Codex`.
4. Review показывает точные read/write boundaries и readiness.
5. `Create first task` открывает New Task; setup не запускает analysis неявно.
6. Пользователь задаёт goal, optional context, scope и runner preset.
7. `Start task` создаёт Task и первый immutable Attempt.
8. После terminal success Task открывается на Outcome; не остаётся в Pipeline Studio.

### 6.2 Daily task flow

1. Tasks открывается на `Needs attention`, затем `Running`, `Ready`, `Completed`.
2. New Task composer требует goal; scope и runner имеют понятные defaults.
3. Inline readiness проверяет runner до start и объясняет recovery рядом с picker.
4. Active Task показывает plain-language phase, completed scopes и last useful progress.
5. Pipeline Studio открывается только через `View attempt details`.
6. Terminal outcome сначала показывает semantic result, decisions и next action.

### 6.3 Recovery

1. Failed/retrying scope подсвечивается один раз в Task summary.
2. Recovery panel отвечает: что произошло, что сохранено, что произойдёт дальше.
3. `Retry failed scope` создаёт child Attempt; terminal Attempt не мутируется.
4. `Change runner for next attempt` открывает runner preset; active Attempt остаётся immutable.
5. Raw logs и runtime JSON доступны только через Diagnostics disclosure.

### 6.4 Knowledge review

1. Successful validator-approved Attempt уже обновил Current Architecture.
2. Task Outcome показывает semantic delta и review decisions.
3. Architecture открывает current map/document/model независимо от active/failed Attempts.
4. Evidence chain всегда показывает authority, repository, path/ref и coverage state.

### 6.5 Publication

1. Changes объясняет уже promoted workspace delta; human review не является promotion gate.
2. Publish загружает authoritative full workspace Git inventory.
3. Confirmation называет branch, HEAD, file/folder counts, open questions и demo/live identity.
4. Единственная mutation — `Commit all workspace changes` или явная proposal-branch action.

## 7. Runner UX contract

### Presets

- `Deterministic demo` — fake, no external calls, никогда не называется live analysis.
- `Claude Code` — headless live provider.
- `Qwen Code` — headless live provider.
- `Codex` — headless live provider.
- `Advanced mix` — per-step overrides, доступен через disclosure, не в default picker list.

### Picker row

Каждый runner показывает:

- provider label и demo/live identity;
- `Ready`, `Needs setup`, `Auth/quota not verified`, `Unavailable`;
- effective model/effort либо `Provider default`;
- last readiness check;
- короткое отличие: reliable default, deterministic demo, current workspace preset.

Task сохраняет immutable effective runner snapshot для каждого Attempt. Изменение workspace preset
не меняет историю. Если fake/headless остаётся process-scoped, UI обязан честно показывать restart
requirement. Accepted target decision — per-Attempt admission из
`ADR-20260811-per-attempt-runner-admission.md`; seamless picker не считается реализованным до
backend acceptance `23A`.

## 8. Artifact and file workbench

### 8.1 Common file identity

Каждый viewer показывает в одном context bar:

- semantic title;
- exact workspace-relative path;
- authority (`Current workspace`, `Run snapshot`, `QA snapshot`);
- validation state;
- demo/live identity, если artifact run-bound;
- Rendered/Source/Diff modes, только когда режим честно доступен.

Raw taskrun paths не появляются в Architecture navigation.

### 8.2 Markdown (`*.md`)

- default mode — `Rendered` с document outline и readable measure `68–76ch`;
- headings, tables, lists, code blocks и relative workspace links рендерятся безопасно;
- citations рядом с claim открывают Evidence drawer без потери scroll/focus;
- `Source` использует text editor с line numbers, search и unsaved state;
- editing разрешён только для Current workspace и только через явный `Edit document`;
- run/QA snapshots read-only и не предлагают Save;
- Save валидирует links/contracts, пишет workspace atomically и переводит Git state в dirty;
- large/broken Markdown показывает bounded Source fallback, не пустой canvas.

### 8.3 YAML model and charter (`*.yaml`)

- default mode — structured inspector/form по известному schema/semantic type;
- entity, edge, charter и workspace manifest имеют разные intentional views;
- schema errors показываются у поля и в summary с line/path identity;
- `Source` остаётся доступным в Advanced;
- structured editing не имеет права терять comments, unknown keys, ordering или user formatting;
  до lossless patch editor structured mode остаётся read-only;
- Save проходит schema + semantic validation до atomic write.

### 8.4 JSON contracts and indexes (`*.json`)

- operator-facing JSON открывается как structured inspector с collapsible sections;
- schemas/validator/index metadata не смешиваются с authored knowledge;
- raw JSON доступен для diagnostics, но не является default reader;
- schema name/version и validation outcome показаны отдельно от content.

### 8.5 Mermaid (`*.mmd`)

- default mode — rendered diagram с zoom/fit и доступным list/table equivalent;
- `Source` показывает Mermaid text рядом с bounded live preview только в edit mode;
- node selection ведёт к entity/document/evidence, но layout не считается semantic truth;
- visual drag editing не входит в target; source text остаётся authoritative;
- broken/oversized diagram показывает source и actionable render error.

### 8.6 Diff

- Markdown: unified/side-by-side text diff; rendered preview не маскирует source changes.
- YAML/JSON: structural key-path summary плюс raw diff fallback.
- Mermaid: source diff; visual diagram diff не заявляется без отдельного deterministic contract.
- Diff всегда называет обе стороны: current/run/baseline/HEAD.

### 8.7 File navigation

Default tree группируется по смыслу:

```text
Architecture Home
Services and domains
Diagrams
Model
Findings and questions
Proposals
Changelog
```

Folder/path explorer остаётся secondary mode. Search ищет title, entity ID, path и document text.

## 9. Screen inventory

| ID | Screen | Primary job | Primary action |
| --- | --- | --- | --- |
| 00 | Guided Setup | attach workspace, sources and initial runner | `Create first task` |
| 01 | New Task | define goal, scope and runner | `Start task` |
| 02 | Task Inbox | choose active/attention/completed work | `New task` |
| 03 | Running Task | understand current progress without telemetry noise | `View attempt details` |
| 04 | Pipeline Recovery | diagnose one Attempt/scope and recover | `Retry failed scope` |
| 05 | Architecture Map | explore current validated system | `Review update` when delta exists |
| 06 | Document Workbench | read/edit Markdown and trace citations | `Edit document` or `Save changes` |
| 07 | Model & Schema Workbench | inspect/edit schema-governed YAML safely | `Edit structured data` |
| 08 | Findings & Evidence | resolve unknowns and trace claims | `Open evidence` / proposal action |
| 09 | Changes Review | understand semantic delta before Git | `Continue to publish` |
| 10 | Publish Confirmation | confirm full workspace Git mutation | `Commit all workspace changes` |
| 11 | Ask | ask current architecture read-only | `Ask` / explicit proposal draft |
| 12 | Runner Settings | manage presets, model/effort and readiness | `Save preset` |

Reference PNGs use the same IDs and one fictional `acme / payments` workspace fixture.

## 10. State matrix

### Task

`draft | blocked_readiness | queued | running | retrying | needs_attention | result_ready |
failed_retained | canceled | completed | archived`

### Attempt

`queued | provider_working | artifact_observed | validating | repairing | stalled | succeeded |
failed | canceled | recovered`

### Artifact

`loading | available | partial | stale | invalid | oversized | render_failed | missing | forbidden`

### Publication

`clean | dirty | loading | stale | blocked | unknown | committing | committed | failed`

Rules:

- last valid Architecture remains visible during refresh/offline/error;
- every empty/error/disabled state names cause and next action;
- no percentage is inferred from stdout or heartbeat;
- retrying is warning, terminal failure is danger;
- status never relies on color alone;
- offline preserves route, drafts and prepared commit message.

## 11. Interaction rules

- One dominant action per screen; recovery may replace it.
- Mutating Git controls exist only in Publish.
- Task/Attempt selection, filters and viewer mode are URL-restorable.
- Start, retry and commit are protected from double submission.
- Disabled controls have visible adjacent reasons.
- Unsaved Markdown/YAML drafts block accidental route exit and survive ordinary in-app navigation.
- Save announces validation/write result once and leaves it inspectable.
- Citation drawer preserves originating selection and returns focus.
- Command/search palette never hides the only way to complete a task.
- Destructive actions use confirmation; reversible local filters/selections do not.

## 12. Visual system

Character: **quiet architectural desk** — warm neutral canvas, graphite text, restrained forest
green trust/action accent and terracotta attention accent. No deep navy permanent sidebar, gradients,
glow, glassmorphism, nested card dashboards or Unicode pseudo-icons.

```text
canvas          #F7F5F0
surface         #FFFFFF
surface-subtle  #F1EFE9
text            #18201C
text-muted      #657069
border          #D8D6CF
action          #2F6648
action-hover    #25543A
link            #315D89
warning         #B85C38
danger          #B33A35
success         #347A52
focus           #2B6CB0
```

- app body: `15px/22px`; metadata: `12–13px`; document body: `16px/26px`;
- page title: `28–32px`; section title: `18–20px`;
- default control 40px; touch target 44px; compact rows 40–48px;
- 4px spacing scale; 6px controls; 8px panels; shadows only for modal/drawer;
- icons come from one accessible vector set; entity types also use shape/text;
- motion 120–180ms, transform/opacity only, reduced-motion respected.

## 13. Responsive behavior

- `>=1280`: Task Inbox and workbenches may use three columns.
- `901–1279`: two columns; inspector becomes overlay drawer.
- `681–900`: one content column, bottom primary navigation, local lists open as sheets.
- `<=680`: Task list/detail are separate routes; map defaults to entity list; Ask/inspectors are
  fullscreen sheets; no document-level horizontal overflow.

Primary action, title, authority and state fit in the first mobile viewport. Dense tables become
keyed rows except source/diff content, which uses bounded local horizontal scroll.

## 14. Accessibility

- WCAG 2.2 AA target; keyboard completion for setup, Task, recovery, document edit and Publish.
- One `h1`, logical landmarks and focus order.
- Visible focus on every surface; icon-only controls have names.
- Tabs follow WAI-ARIA keyboard pattern; modal sheets trap focus and return it.
- Map has list/table equivalent for entities and relations.
- Status has icon/text/shape; charts/coverage have textual summary.
- Zoom 200%, long paths and 390x844 preserve task completion.

## 15. Reference screens

1. [Guided Setup](assets/ui-task-first-product/00-guided-setup.png)
2. [New Task](assets/ui-task-first-product/01-new-task.png)
3. [Task Inbox](assets/ui-task-first-product/02-task-inbox.png)
4. [Running Task](assets/ui-task-first-product/03-task-running.png)
5. [Pipeline Recovery](assets/ui-task-first-product/04-pipeline-recovery.png)
6. [Architecture Map](assets/ui-task-first-product/05-architecture-map.png)
7. [Document Workbench](assets/ui-task-first-product/06-document-workbench.png)
8. [Model and Schema Workbench](assets/ui-task-first-product/07-model-schema-workbench.png)
9. [Findings and Evidence](assets/ui-task-first-product/08-findings-evidence.png)
10. [Changes Review](assets/ui-task-first-product/09-changes-review.png)
11. [Publish Confirmation](assets/ui-task-first-product/10-publish-confirmation.png)
12. [Ask Current Architecture](assets/ui-task-first-product/11-ask-current-architecture.png)
13. [Runner Settings](assets/ui-task-first-product/12-runner-settings.png)

## 16. Acceptance criteria

- A first-time evaluator can state what ACP reads, where it writes, which runner is effective and
  what output will exist before starting the first Task.
- New Task requires no navigation to Settings for ordinary runner choice.
- Task remains the stable object across retry/rerun; every Attempt retains exact runner/config.
- Terminal success opens Outcome, not active pipeline language.
- Current Architecture remains available after failed/canceled Attempts.
- Markdown/YAML/Mermaid default readers match their content type; raw source is progressive detail.
- Editing cannot write run snapshots or source repositories and cannot silently lose YAML content.
- Changes copy states that validator-approved promotion already occurred; human action is Git review.
- Publish uses authoritative full-workspace inventory and one explicit confirmation.
- Required state matrix is covered by component/fixture tests and rendered desktop/mobile scenarios.
- No hidden compatibility shell, duplicate navigation or stale old visual reference remains.

## 17. Non-goals

- hosted, multi-tenant or collaborative approval workflow;
- source-repository writes;
- visual Mermaid drag editor;
- automatic human approval of findings/proposals;
- selected-file Git commit while full-workspace publication remains canonical;
- new providers beyond `claude-code`, `qwen-code`, `codex-code` and deterministic `fake`;
- schema/API changes hidden inside frontend-only work.
