# Что ACP может взять из подхода Karpathy LLM Wiki

Дата: 2026-06-18

Источник контекста:
- `docs/KARPATHY_LLM_WIKI_COMPARISON_2026-06-18.md`
- Karpathy gist: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
- текущие ACP specs: `docs/spec/PIPELINE_SPEC.md`, `docs/spec/MODEL_SPEC.md`, `docs/spec/WORKSPACE_SPEC.md`

## Короткий вывод

ACP не нужно становиться generic Obsidian/wiki-системой. Сильная сторона ProvenArch уже ближе к "industrial LLM Wiki", чем к обычному RAG: source repos и imports читаются как raw sources, runtime компилирует архитектурное знание, orchestrator валидирует и публикует результат как Git-friendly workspace.

Главное, что стоит взять у Karpathy:

1. Явно описать `arch-workspace` как compiled architecture knowledge base.
2. Сделать полезные Q&A/analysis результаты не одноразовыми, а promote-able через review/proposal flow.
3. Добавить health/lint слой для архитектурной памяти: stale claims, contradictions, duplicate entities, orphan reports, aging open questions.
4. В будущем ввести claim-level модель поверх текущих reports/citations/model: stable claim IDs, support/contradict/supersede relations, review status.
5. Если появится проблема масштаба поиска/context loading, строить rebuildable derived search index, но не делать его source of truth.

## Фильтр принятия идей

Любая идея из Karpathy-подхода должна проходить четыре вопроса:

1. Усиливает ли она Git-versioned architecture workspace, а не заменяет его чатовой памятью?
2. Сохраняет ли она правило: runtime пишет staged artifacts, canonical outputs публикует orchestrator?
3. Может ли required CI остаться deterministic/fake, без live provider и network dependency?
4. Есть ли понятный source of truth: schema/spec/file contract, а не "agent should remember"?

Если ответ "нет" хотя бы на один вопрос, это не MVP slice.

## Матрица применимости

| Идея Karpathy / thread | Польза для ACP | Стоимость | Риск | Решение |
| --- | --- | --- | --- | --- |
| "Wiki as compiled artifact" terminology | Улучшает product framing: ACP outputs не reports dump, а поддерживаемая архитектурная память | Низкая | Только wording drift, если не синхронизировать docs | Взять при ближайшем docs/UX slice |
| Raw sources immutable | Уже совпадает с source repos read-only и imports | Низкая | Нет | Сохранить как invariant |
| `index.md` + `log.md` split | ACP уже имеет indexes/taskruns/changelog, но human navigation можно усилить | Средняя | Дублирование с existing indexes | Взять выборочно, не как новый source of truth |
| Query results file back into wiki | Делает Ask не одноразовым; полезно для architecture exploration | Средняя | Автоматическая мутация canonical docs может загрязнить workspace | Взять только как explicit user-promoted proposal/review artifact |
| Periodic lint/health check | Очень полезно: stale docs, orphan artifacts, old questions, duplicate model entities | Средняя | Может смешаться с validator step3 | Взять как deterministic workspace health report, отдельно от provider validator |
| Claim-level provenance | Закрывает главный risk: narrative claims без стабильной identity | Высокая | Schema/model expansion, migration cost | Wave 1 candidate после MVP stabilization |
| Typed claim relationships (`supports`, `contradicts`, `supersedes`) | Позволяет не "исправлять" противоречия, а хранить decision tension | Высокая | Ontology drift, edge vocabulary governance | Только после claim ledger, не смешивать с topology edges |
| Derived SQLite/FTS/BM25 index | Ускоряет Ask/context selection на больших workspaces | Средняя | Index может стать hidden source of truth | Взять как rebuildable cache/projection, не committable canonical state |
| MCP memory server | Удобно для multi-agent ecosystems | Высокая | Hosted/API/security creep, не MVP | Не брать в MVP |
| Agent-owned free-form wiki pages | Быстро растит знания | Низкая сначала, высокая потом | Silent corruption, hallucinated notes, schema drift | Не брать; ACP должен оставаться orchestrator-owned |
| Autonomous self-heal/nightly rewrite | Может держать wiki свежей | Высокая | Тихая порча корректных решений | Не брать без review queue |
| Generic Obsidian compatibility | Привлекательно для personal PKM | Средняя | Размывает product focus | Не целиться специально; markdown/git уже достаточно |
| Stable concept identity / merge/split tools | Полезно для duplicate services/domains/entities | Средняя | Needs UX + migration policy | Взять в ограниченном виде для model/entity dedupe |

## Что взять немедленно

### 1. Product framing: compiled architecture knowledge base

Сейчас README говорит правильную вещь: результат ACP - не картинка и не чат-лог, а набор файлов. Karpathy дает более сильный язык: "compiled knowledge artifact".

Для ACP это можно сформулировать так:

```text
source repos / imported docs -> runtime analysis -> validated compiled architecture workspace
```

Практическая польза:
- оператор лучше понимает, зачем нужны `reports/`, `model/`, `proposals/`, `changelog`;
- объясняет отличие от RAG/chat upload tools;
- согласуется с local-first и Git-friendly diffs.

Где применить:
- README intro;
- onboarding copy;
- `Review` / `Ask` empty states;
- docs/ARCHITECTURE overview.

Это docs/UX wording change. Контракты не менять.

### 2. Ask answer promotion, но только через review

Karpathy считает важным, что хороший ответ не должен исчезать в chat history. В ACP `qa.ask` сейчас безопасно пишет run-scoped `qa-answer.json` и не мутирует canonical workspace. Это правильный baseline, но можно добавить curated compounding loop.

Правильная форма для ACP:

```text
Ask answer -> user selects "Promote" -> proposal/review draft -> Git review -> canonical promotion
```

Не делать:
- автоматическую запись ответа в `reports/as-is/*`;
- автоматическое изменение `model/*`;
- provider-authored canonical markdown без validator/review.

Возможный artifact:

```text
proposals/qa-synthesis-<slug>/
  proposal.md
  evidence.md
  source-qa-answer.json
```

Польза:
- Ask становится не только read-only interrogation, но и способом открыть follow-up architecture work;
- Q&A exploration начинает аккумулироваться;
- всё остается reviewable через Git.

Риск:
- proposal spam.

Guardrail:
- promotion только явным user action;
- proposal должен сохранять citations/unresolved assumptions;
- no direct model/report mutation.

### 3. Workspace health/lint report

Karpathy lint ищет contradictions, stale claims, orphan pages, missing links, gaps. У ACP уже есть validator, но validator привязан к run final set. Нужен отдельный deterministic health pass по workspace state.

Первый полезный scope без schema churn:

- unresolved questions older than N runs;
- findings without linked proposal;
- proposals without evidence refs;
- reports referencing missing model entity;
- model entities without observation evidence;
- duplicate aliases / possible duplicate services;
- orphan `reports/agent-outputs/domains/*` without domain card;
- stale `reports/taskruns` selected in UI but not linked from run history;
- broken artifact references in final/citation indexes.

Artifact:

```text
reports/health/workspace-health.md
reports/health/workspace-health.json
```

Важно: это не replacement for `validator-verdict.json`. Это separate health surface over already published workspace.

Польза:
- прямо переносит Karpathy lint в ACP language;
- хорошо ложится в `Readiness` / `Review` / `Publish`;
- можно сделать deterministic и covered by tests.

### 4. Better chronological memory

Karpathy разделяет content index и chronological log. У ACP есть `reports/changelog/*`, `reports/taskruns/*`, `run-history.json`, logs. Но для human navigation они сейчас больше operational, чем "memory timeline".

Что можно взять:
- human-readable "workspace evolution log" как view поверх existing changelog/run-history;
- не новый canonical source, а compiler-rendered summary.

Вариант:

```text
reports/changelog/index.md
```

Содержит:
- run id / date / pipeline;
- changed report/model/proposal surfaces;
- warnings/error highlights;
- links to promoted artifacts.

Польза:
- упрощает "что изменилось за последние N прогонов";
- поддерживает agent/operator continuity.

Не надо создавать отдельный `log.md` вручную maintained by runtime. Это должен собирать orchestrator.

## Что взять после MVP foundation

### 5. Claim ledger

Самая сильная идея из комментариев: не просто хранить page/report citations, а дать каждому значимому утверждению stable identity.

Сейчас ACP уже требует `claim_ids` в citation index и provenance в model. Но это еще не полноценная claim memory:
- нет отдельного claim lifecycle;
- нет `superseded_by`;
- нет explicit contradiction relation;
- нет status `active|challenged|superseded|rejected`;
- нет review ownership.

Возможная модель:

```text
model/claims/
  claim.<slug>.yaml
model/claim-edges/
  edge.claim-a.supports.claim-b.yaml
```

Пример:

```yaml
id: claim.svc-payments.uses-postgres
subject_id: svc.payments
predicate: uses_datastore
object_id: db.postgres.payments
status: active
provenance:
  kind: observation
  confidence: 0.88
  evidence:
    - repo: payments-service
      path: docker-compose.yml
      lines:
        start: 12
        end: 24
```

Claim edge:

```yaml
id: edge.claim.a.contradicts.claim.b
type: contradicts
from: claim.a
to: claim.b
status: needs_review
provenance:
  kind: inference
  confidence: 0.7
  evidence: [...]
```

Почему это не MVP-fast:
- меняет model contract;
- требует schemas + fixtures + docs + validators + UI;
- нужен vocabulary governance.

Правильный порядок:
1. сначала health report использует существующие citations/model;
2. затем claim ledger prototype только для limited claim types;
3. только потом typed claim edges.

### 6. Contradiction policy

Karpathy base pattern иногда трактует contradiction как defect. Для ACP это тоже опасно: две ветки, два env, старый deploy и новый deploy могут давать разные факты, и это не всегда ошибка.

Нужна политика:

- contradiction between observations should not be auto-resolved;
- newer evidence can supersede older evidence only under explicit rule;
- ambiguous conflicts become review items;
- objective topology conflict может быть finding;
- historical/current mismatch может быть `supersedes`;
- env-specific differences should be modeled as attributes, not contradictions.

Минимально это можно начать без claim ledger:
- validator/health report emits `conflict_candidate`;
- no auto-rewrite;
- UI routes to review/proposal.

### 7. Rebuildable search projection

Karpathy говорит, что `index.md` хватает до умеренного масштаба. Комментарии подтверждают: дальше нужен search/index layer.

Для ACP правильная архитектура:

```text
canonical workspace files -> derived search projection -> Ask/context selection
```

Проекция может включать:
- FTS over reports/model/proposals/charter/imports;
- citation-aware chunks;
- entity aliases;
- recency/run metadata;
- artifact path filters.

Нельзя:
- делать vector DB canonical memory;
- хранить непроверенные generated summaries только в index;
- требовать external service в required CI.

Хороший MVP-later подход:
- local SQLite under `.acp/cache/` or `reports/search/` depending on commit policy;
- deterministic rebuild command;
- tests verify rebuild from fixtures;
- UI can show "search index stale/rebuild".

Я бы не коммитил binary SQLite в Git by default. Лучше derived cache с deterministic rebuild.

### 8. Entity merge/split curation

Karpathy-thread много говорит о stable concept identity. Для ACP это проявится как:
- duplicate services;
- repo rename vs service rename;
- same datastore named differently;
- external system aliases;
- domain card mismatch.

ACP уже имеет stable IDs и aliases. Следующий шаг - curation workflow:

- show duplicate candidates;
- accept merge by adding aliases or manual migration;
- keep canonical ID stable;
- never silently re-key.

Это хорошо ложится на `Review` или `Publish`:

```text
Possible duplicate entities:
- svc.payment-api
- svc.payments
Action: keep separate | add alias | create migration proposal
```

## Что не брать

### 1. Free-form provider-owned wiki

Karpathy говорит "LLM owns wiki layer". Для ACP это нельзя брать буквально.

Почему:
- текущая сила ACP в staged artifacts and validator-gated promotion;
- provider CLI может ошибиться путём, форматом, scope;
- canonical workspace должен оставаться orchestrator-controlled.

Верная адаптация:
- provider owns staged draft only;
- orchestrator owns promotion;
- human owns canonical cards and publish decision.

### 2. Autonomous self-healing canonical docs

Ночной агент, который сам переписывает stale architecture docs, звучит полезно, но для ACP опасен.

Допустимо:
- health report;
- proposed patch;
- review queue;
- explicit user promotion.

Недопустимо:
- silent rewrite of `reports/as-is/*`;
- silent model re-key;
- auto-resolve contradictions.

### 3. Generic memory platform / MCP server in MVP

MCP memory service и multi-agent shared memory - интересное направление, но это другой продуктовый boundary.

Почему не MVP:
- security/API surface;
- auth/network questions;
- может конфликтовать с no hosted mode;
- отвлекает от local-first architecture pipeline.

Можно вернуться позже, если появится четкий use case:
- external agents ask read-only questions against an ACP workspace;
- no canonical mutation;
- loopback/trusted mode only.

### 4. Vector DB as source of truth

Search index может быть полезен. Но source of truth должен оставаться:

```text
schemas/* + docs/spec/* + workspace files + model YAML + report artifacts
```

Vector/BM25/SQLite projection должен быть rebuildable and disposable.

## Рекомендуемые slices

### Slice A - Terminology and UX framing

Scope:
- README/ARCHITECTURE/UI copy: `compiled architecture workspace`, `architecture knowledge base`, `not chat history`.
- No schema/API changes.

Acceptance:
- docs describe raw sources -> staged artifacts -> validated compiled workspace.
- no change to pipeline behavior.

Risk: низкий.

### Slice B - Ask promotion to proposal draft

Scope:
- UI action from selected QA answer: `Create proposal draft`.
- Backend copies selected `qa-answer.json` citations/unresolved assumptions into proposal draft package.
- No direct report/model mutation.

Possible output:

```text
proposals/qa-synthesis-<run-id>-<slug>/
  proposal.md
  evidence.md
  source-qa-answer.json
```

Acceptance:
- explicit user action only;
- draft includes citations and unresolved assumptions;
- Publish/Git Review sees created files;
- tests cover no canonical report/model mutation.

Risk: medium. Main risk is proposal clutter.

### Slice C - Deterministic workspace health report

Scope:
- Add local deterministic health scanner over published workspace artifacts.
- Output `reports/health/workspace-health.{json,md}` or expose through API first.
- Checks are structural/semantic, not live LLM.

Initial checks:
- broken artifact refs;
- old unresolved questions;
- findings with no proposal link;
- proposals with missing evidence;
- duplicate entity alias candidates;
- orphan domain outputs;
- observation provenance gaps.

Acceptance:
- fixture-backed tests;
- no live provider dependency;
- no mutation except health report;
- UI displays health summary in Review/Readiness.

Risk: medium. Need avoid overlapping/confusing with step3 validator.

### Slice D - Citation/claim identity hardening

Scope:
- Before a full claim ledger, harden existing `citation-index.json` and narrative claim usage.
- Add checks for stable claim IDs, duplicate claim IDs, citation coverage in key reports.

Acceptance:
- validator or health scanner detects claim drift;
- examples/fixtures updated;
- no new top-level model contract yet.

Risk: medium-low.

### Slice E - Claim ledger prototype

Scope:
- Introduce limited `model/claims/*.yaml` for a narrow set of architecture facts.
- Derive from existing semantic manifests/citation index.
- Add schemas/docs/tests.

Initial claim types:
- service owns/exposes API;
- service uses datastore;
- service calls service/external;
- service publishes/subscribes topic.

Acceptance:
- stable claim IDs;
- evidence required for observation;
- no contradiction edges yet, only status/lifecycle.

Risk: high. Requires schema guardian and fixtures.

### Slice F - Contradiction review queue

Scope:
- Build on claim ledger or health conflict candidates.
- Add `supports|contradicts|supersedes` vocabulary only for claim edges, not topology edges.

Acceptance:
- contradictions never auto-resolve;
- user-visible review item;
- policy documented;
- fixtures cover objective conflict vs historical supersession.

Risk: high. Ontology governance is real work.

### Slice G - Derived search projection

Scope:
- Rebuildable local FTS index over workspace files.
- Used by QA/context selection only after deterministic fallback exists.

Acceptance:
- index is rebuildable from workspace;
- not canonical;
- required CI can rebuild from fixtures without network;
- UI/API surfaces stale/rebuild state.

Risk: medium-high. Need avoid hidden source of truth.

## Suggested priority

Recommended order:

1. Slice A - terminology/UX framing.
2. Slice C - deterministic workspace health report.
3. Slice B - Ask promotion to proposal draft.
4. Slice D - citation/claim identity hardening.
5. Slice E - claim ledger prototype.
6. Slice F - contradiction review queue.
7. Slice G - derived search projection, only when context loading/search becomes a measured pain.

Why this order:
- A is cheap and clarifies product direction.
- C gives immediate value and imports Karpathy lint safely.
- B adds compounding behavior without automatic canonical mutation.
- D/E/F address the deeper provenance/contradiction problem in controlled steps.
- G should be driven by scale evidence, not premature infrastructure.

## MVP line

Safe for MVP:
- wording/framing;
- deterministic health/lint report;
- explicit QA-to-proposal promotion;
- stronger citation checks.

Probably post-MVP:
- claim ledger;
- contradiction edge vocabulary;
- entity merge/split UI;
- rebuildable search projection.

Not for MVP:
- MCP memory service;
- provider-owned canonical wiki;
- autonomous self-heal;
- vector DB as memory source;
- generic personal knowledge system scope.

## Final recommendation

Karpathy's strongest contribution for ACP is not the folder structure. It is the discipline that knowledge should compound into maintained artifacts.

ACP should take that discipline, but keep its current stricter architecture:

```text
LLM can discover and draft.
Schemas define valid shapes.
Orchestrator stages and validates.
Human reviews decisions.
Git records accepted knowledge.
Derived indexes can accelerate, but never become truth.
```

The first practical implementation should be `workspace health/lint` plus `Ask answer promotion to proposal draft`. Together they import the two most valuable Karpathy loops - maintenance and compounding - without weakening ACP's core contract model.
