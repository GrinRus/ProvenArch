# Karpathy LLM Wiki x ProvenArch: integration design

> Archived 2026-09-05 from `docs/KARPATHY_PROVENARCH_INTEGRATION_DESIGN_2026-07-09.md`. This dated research/design proposal is retained as rationale, not an active implementation plan or current contract. Implemented health, citation and Ask-to-Proposal behavior is described in the [current architecture](../../ARCHITECTURE.md) and [status matrix](../../STAKEHOLDER_DOC.md); future proposals still need an explicit slice.

Дата: 2026-07-09

## Назначение

Этот документ повторно анализирует подход Karpathy LLM Wiki и текущее состояние ProvenArch/ACP, затем фиксирует, что именно стоит встраивать, куда, зачем и в каком порядке.

Документ не меняет product contract сам по себе. Он является decision/design surface для следующих implementation slices.

Связанные документы:
- `docs/archive/research/KARPATHY_LLM_WIKI_COMPARISON_2026-06-18.md`
- `docs/archive/research/KARPATHY_ADOPTION_ANALYSIS_2026-06-18.md`
- `docs/PLANS.md` -> `EP-20260623-karpathy-adoption-roadmap`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/spec/MODEL_SPEC.md`
- `docs/spec/WORKSPACE_SPEC.md`

## 1. Что на самом деле предлагает Karpathy

Karpathy LLM Wiki не столько про Obsidian или markdown, сколько про иную форму работы с LLM и накоплением знания.

Классический RAG:

```text
raw documents -> retrieve chunks at query time -> answer
```

LLM Wiki:

```text
raw sources -> LLM-maintained compiled wiki -> query/lint/update loop
```

Смысловой сдвиг:
- знание не пересобирается заново на каждый вопрос;
- source остается immutable;
- LLM поддерживает промежуточный compiled layer;
- query outputs могут становиться новыми knowledge artifacts;
- lint/health pass поддерживает wiki живой;
- index/log помогают навигации и continuity между сессиями.

Три слоя Karpathy:

1. **Raw sources**
   - curated source documents;
   - source of truth;
   - LLM читает, но не правит.

2. **Wiki**
   - LLM-generated markdown;
   - summaries, entities, concepts, comparisons, overview;
   - maintained and cross-linked by LLM.

3. **Schema**
   - `CLAUDE.md`, `AGENTS.md` or equivalent;
   - conventions, workflows, page shape, lint rules;
   - makes the agent a disciplined maintainer.

Операции:
- ingest;
- query;
- lint;
- index/log maintenance;
- optional search/index tools as scale grows.

Свежая важная поправка из обсуждения gist: по мере роста wiki начинает болеть не только hallucination, но и token/context efficiency. Повторное чтение index, manifests, handoffs and source files становится дорогим. Это усиливает аргумент за deterministic routing/index layers и за то, чтобы LLM занимался synthesis/judgment, а не каждую сессию сканировал весь workspace.

## 2. Что уже есть в ProvenArch

ACP уже реализует более строгий вариант той же идеи, но для architecture workspace.

Текущий ACP flow:

```text
source repos / imported docs
  -> Go orchestrator
  -> runtime provider in staged write roots
  -> artifact contracts and validator
  -> promoted arch-workspace files
  -> UI review / Ask / Publish
```

Ключевые свойства:
- local-first;
- no hosted MVP;
- source repos are read-only inputs;
- central Git `arch-workspace`;
- deterministic `fake` baseline for required CI;
- headless providers only through artifact-only contracts;
- strict manifests and schemas;
- `validator-verdict.json` gates promotion;
- `model/entities` and `model/edges` are derived YAML;
- async `qa.ask` writes only run-scoped audit artifacts.

У ACP уже есть аналоги Karpathy layers:

| Karpathy layer | ACP equivalent | Важное отличие |
| --- | --- | --- |
| Raw sources | `repos[]`, `docs.imports_path`, repo checkouts | ACP дополнительно resolver/validates sources and refs |
| Wiki | `reports/*`, `model/*`, `proposals/*`, `charter/*`, `reports/changelog/*` | ACP wiki layer is not free-form: generated through pipeline and promotion gates |
| Schema | `schemas/*`, `docs/spec/*`, `skills/*`, prompt packs, validators | ACP schema is executable/tested, not only instructions |
| Ingest | `init/refresh.step1.collect` | Sharded, manifest-backed, provider output is staged |
| Query | `qa.ask` and compatibility `acp qa` | Run-scoped, currently non-compounding |
| Lint | `step3.findings` and validators | Validator checks staged final set, not full published workspace health |
| Index/log | `final-run-index.json`, `citation-index.json`, `run-history`, `reports/changelog/*` | Mostly operational/artifact indexes, not yet human-friendly knowledge health/navigation layer |

Главный вывод: ACP не нужно строить wiki. ACP уже является contract-gated architecture wiki/compiler. Нужно встроить недостающие loops: health, curated Q&A promotion, claim/citation hardening, eventually claim graph/search projection.

## 3. Главная граница интеграции

Karpathy говорит "LLM owns the wiki layer". Для ACP это нельзя брать буквально.

Правило ACP:

```text
LLM discovers and drafts.
Schemas define valid shapes.
Orchestrator stages, validates and promotes.
Human reviews decisions.
Git stores accepted architecture knowledge.
Derived indexes accelerate but never become truth.
```

Это правило должно остаться неизменным.

Следовательно:
- runtime provider не должен писать canonical `reports/*`, `model/*`, `charter/*` напрямую;
- `qa.ask` не должен сам обновлять canonical workspace;
- health/lint может предлагать actions, но не должен silently rewrite;
- search projection не должна становиться source of truth;
- claim relations нельзя смешивать с topology edges без отдельного model contract.

## 4. Product mental model после интеграции

Целевой mental model:

```text
ACP is a compiled architecture knowledge base.

Inputs:
  source repos, imported docs, human-owned charter/cards.

Compilation:
  staged runtime artifacts, manifests, validator, deterministic compiler.

Accepted memory:
  reports, model, proposals, changelog, health reports.

Exploration:
  Ask answers are run-scoped first.
  Useful answers can be explicitly promoted into proposal drafts.

Maintenance:
  workspace health scans detect drift, stale claims, missing links and unresolved gaps.

Scale:
  rebuildable search/context projections help routing, but canonical truth remains files.
```

This explains "where Karpathy fits":
- Karpathy's "wiki" becomes ACP's accepted architecture workspace;
- Karpathy's "lint" becomes deterministic workspace health;
- Karpathy's "file query result back" becomes explicit Ask promotion;
- Karpathy's "index/search when scale grows" becomes derived search projection;
- Karpathy's "contradictions flagged" becomes claim/contradiction review, later.

## 5. Integration map by ACP product stage

### Source

Current role:
- configure `workspace.yaml`;
- attach repos/imports;
- validate source readiness.

Karpathy fit:
- Source is ACP's raw source layer.
- Imported docs should be treated like curated raw notes, not generated wiki pages.

What to add:
- wording only in near term: "Raw sources for compiled architecture workspace".
- later: source freshness indicators for imports, using `source_updated_at` / `checksum` from docs imports index.

Do not add:
- automatic web clipping;
- generic inbox folder;
- provider ingestion outside pipeline.

Why:
- ACP source model is explicit and reproducible;
- generic inbox would weaken `workspace.yaml` and imports contract.

### Readiness

Current role:
- workspace validation;
- doctor checks;
- runtime/profile readiness.

Karpathy fit:
- Readiness is the right home for "can this knowledge base be maintained safely?".

What to add first:
- workspace health summary once K2 exists:
  - latest health report status;
  - broken artifact refs;
  - unresolved question age;
  - provenance gaps;
  - duplicate candidates.

Why:
- health/lint should be visible before running or publishing;
- it is not a pipeline validator replacement, but workspace maintenance status.

### Charter

Current role:
- human-owned domain/team cards;
- baseline prompts/skills;
- project rules.

Karpathy fit:
- Charter is ACP's schema/human-governed intent layer.

What to add:
- no immediate behavior.
- later, if claim ledger is introduced, charter/rules should define claim policy:
  - when evidence supersedes old evidence;
  - what counts as contradiction;
  - who owns unresolved architecture claims.

Why:
- Karpathy "schema co-evolves with user" maps to ACP's charter/skills, but ACP must keep executable schemas separate.

### Analysis

Current role:
- run mission control;
- step timeline;
- artifacts/logs/evidence/diff.

Karpathy fit:
- Analysis is compilation runtime, not accepted memory.

What to add:
- for K2, show health scanner as separate "workspace health" task only if it is run independently or after promotion.
- avoid mixing health report with step3 validator.

Why:
- step3 validates staged final set;
- health validates already accepted workspace over time.

### Review

Current role:
- inspect reports/model/diagrams/evidence;
- artifact/domain map review.

Karpathy fit:
- Review is where compiled architecture memory is inspected.

What to add:
- health findings panel;
- stale/duplicate/orphan indicators;
- links from health items to artifacts/model entities/proposals;
- later: claim review workbench.

Why:
- Karpathy's graph/links/orphans become useful only if the operator can inspect and resolve them.

### Proposals

Current role:
- proposal/changelog review room.

Karpathy fit:
- This is the correct place for "query outputs file back into wiki", adapted to ACP.

What to add:
- `qa-synthesis-*` proposal packages from Ask promotion;
- evidence and unresolved assumptions copied from QA answer;
- quality blocker if proposal lacks citations.

Why:
- keeps compounding loop explicit and reviewable;
- avoids canonical mutation by Q&A runtime.

### Ask

Current role:
- async runtime-backed Q&A;
- run-scoped audit artifacts;
- no canonical mutation.

Karpathy fit:
- Ask is exploration.
- In Karpathy, useful exploration can become wiki content.

What to add:
- "Create proposal draft from this answer" action for succeeded QA runs.
- "This answer is audit-only until promoted" copy.

What not to add:
- "Save to reports";
- "Update model";
- automatic filing of every answer.

Why:
- preserve safety boundary;
- let high-signal exploration compound.

### Publish

Current role:
- Git Review Room.

Karpathy fit:
- Git is the accepted memory boundary.

What to add:
- health and promoted QA proposal artifacts visible in publication summary.
- possibly warning if publishing while health report has red status.

Why:
- accepted knowledge should pass operator review.

## 6. Integration map by backend module

### `internal/qa`

Current state:
- builds context pack;
- deterministic compatibility answer;
- parses/validates `qa-answer.json`;
- validates citations against context pack.

Integrate:
- no direct canonical mutation here.
- add helper/read model for promotion input only if needed, but not proposal writing.

Why:
- `qa` package should remain about asking and validating answers, not publishing.

### `internal/orchestrator/qa_runs.go`

Current state:
- orchestrates async QA run;
- writes `context-pack.json`;
- invokes runtime;
- validates answer;
- stores artifacts.

Integrate:
- keep unchanged for K3 except maybe add artifact metadata needed by promotion.
- promotion should be a separate API/service path that reads a completed QA run.

Why:
- `qa.ask` must stay non-mutating and repeatable.

### `internal/api/server.go`

Current state:
- owns HTTP routing for workspace, runtime, artifacts, git, QA and pipeline.

Integrate:
- K2: possible `GET /api/workspace/health` or `POST /api/workspace/health/run`.
- K3: possible `POST /api/qa/runs/<run_id>/promote`.

Why:
- these are operator actions over accepted workspace, not runtime pipeline steps.

Endpoint shape should be narrow:

```text
GET  /api/workspace/health
POST /api/workspace/health/run
POST /api/qa/runs/<run_id>/promote
```

Do not overload:
- `/api/qa/runs` response mutation;
- `/api/pipeline/init`;
- `/api/artifacts/write`.

### New `internal/workspacehealth`

Recommended K2 module.

Responsibilities:
- scan accepted workspace artifacts;
- produce deterministic JSON report;
- render markdown report if report is persisted;
- never call runtime providers;
- never mutate source repos;
- no network.

Inputs:
- `workspace.Root`;
- `reports/*`;
- `model/*`;
- `proposals/*`;
- `charter/cards/*`;
- `reports/taskruns/run-history.json` if useful.

Initial checks:
- broken artifact refs;
- stale unresolved questions;
- findings without related proposals;
- proposals without evidence/citation section;
- observation entities/edges without evidence;
- duplicate aliases / possible duplicate entity names;
- orphan domain outputs;
- missing final/citation index targets.

Outputs:

```text
reports/health/workspace-health.json
reports/health/workspace-health.md
```

Potential JSON shape:

```json
{
  "version": 1,
  "generated_at": "2026-07-09T00:00:00Z",
  "status": "pass|warn|fail",
  "summary": {
    "checks": 7,
    "pass": 4,
    "warn": 2,
    "fail": 1
  },
  "items": [
    {
      "id": "health.provenance.entity.svc-payments",
      "check": "model.provenance",
      "severity": "warning",
      "title": "Entity has observation provenance without evidence",
      "path": "model/entities/svc.payments.yaml",
      "related_paths": ["reports/as-is/overview.md"]
    }
  ]
}
```

Why new module:
- avoids mixing validator, reports compiler and QA;
- deterministic scanner can be tested with fixtures.

### `internal/reports`

Current state:
- deterministic render/materialize surfaces;
- writes coverage/findings/domain outputs/changelog/diagrams.

Integrate:
- render health markdown either here or in `workspacehealth`.
- If report is pure render from health JSON, `reports` can host renderer.

Why:
- report rendering fits here, but health rule logic should not.

### `internal/model`

Current state:
- entity-per-file YAML;
- stable IDs/aliases;
- semantic snapshot apply.

Integrate:
- K2 read-only duplicate/provenance checks.
- K4 claim/citation hardening can inspect model relationships.
- K5 claim ledger only after separate schema-first plan.

Do not add now:
- claim model directories without schemas/specs/fixtures.

### `internal/orchestrator/docflow_promotion.go`

Current state:
- promotes validator-approved artifacts;
- clears managed canonical prefixes;
- rebuilds derived model/diagrams.

Integrate:
- not for K2/K3 first cut.
- Later, optionally run health report after promotion.

Why:
- health should not block existing promotion until policy is defined.
- coupling health generation to promotion can make first slice too invasive.

### UI `StagePanels.tsx`

Current state:
- contains Source/Readiness/Analysis/Review/Proposals/Ask/Publish stage panels.
- Ask already has safety panel saying writes are limited to `reports/taskruns/<run_id>/qa/`.

Integrate:
- K2: health summary in Readiness and Review.
- K3: button in Ask answer panel: `Create proposal draft`.
- K3: show promoted proposal link after success.
- Proposals stage should naturally pick up new proposal files if artifact/git diff surfaces are wired.

Need eventually split large `StagePanels.tsx`, but not required for first slice.

## 7. Integration sequence

### Phase 0: clarify language

Goal:
- make product framing match actual architecture.

Change:
- docs/UX copy only.

Why:
- aligns team mental model before implementation;
- low risk.

Deliverable:
- README/ARCHITECTURE mention "validated compiled architecture knowledge base".

### Phase 1: deterministic workspace health

Goal:
- import Karpathy lint safely.

Change:
- new health scanner;
- optional API;
- UI summary;
- fixture tests.

Why first:
- direct value;
- no live provider;
- no new ontology;
- helps future slices find drift.

Do not:
- auto-repair;
- block promotion;
- call LLM.

### Phase 2: Ask promotion to proposal

Goal:
- import Karpathy compounding query loop without automatic writes.

Change:
- explicit promotion API/action;
- proposal package writer;
- UI button/link.

Why second:
- uses existing QA and proposal concepts;
- keeps human review boundary.

Do not:
- mutate reports/model;
- change `qa.ask` write scope.

### Phase 3: citation/claim identity hardening

Goal:
- improve provenance and narrative stability using existing artifacts.

Change:
- health rules around `citation-index.json`, claim IDs, report citation coverage.

Why before claim ledger:
- avoids designing ontology before seeing actual drift.

### Phase 4: claim ledger prototype

Goal:
- model durable architecture claims as first-class artifacts.

Change:
- new schema/spec/model dirs.

Why later:
- schema-heavy;
- needs model migration policy;
- risk of second source of truth.

### Phase 5: contradiction review

Goal:
- preserve meaningful conflicts instead of flattening them.

Change:
- claim relations and review queue.

Why after claim ledger:
- contradiction needs stable claim IDs.

### Phase 6: search/context projection

Goal:
- solve measured context/token efficiency issues.

Change:
- rebuildable local index.

Why last:
- current `qa.BuildContextPack` already does deterministic ranked selection with hard limits;
- optimize when scale evidence says it is necessary.

## 8. Why this order

K2 before K3:
- health tells us whether accepted workspace is maintainable before we add more proposal-producing flows.

K3 before K5:
- compounding via proposal drafts gives value without model/schema expansion.

K4 before K5:
- existing citations and claim IDs already reveal what claim model should look like.

K5 before K6:
- contradictions require stable claim nodes.

K7 after all:
- search infrastructure should solve real scaling pain, not become premature architecture.

## 9. Concrete first slice recommendation

First implementation slice:

```text
K2a: Workspace Health MVP
```

Minimal scope:
- backend-only deterministic scanner;
- no API if CLI/internal test is enough for first PR, but API is acceptable if small;
- persisted reports optional. If persisted, document as report artifact;
- no UI if it makes PR too large. UI can be K2b.

Suggested minimal checks:
1. `model/entities/*.yaml` and `model/edges/*.yaml` observation provenance must have evidence.
2. `reports/agent-outputs/domains/*.md` must correspond to a domain card, ignoring `.task-envelope.json`.
3. `proposals/*/proposal.md` should contain evidence/citation/unresolved section or linked artifacts.
4. `reports/coverage/open-questions.md` unresolved count should be surfaced.
5. `reports/taskruns/run-history.json` referenced selected/current run should not point to missing taskrun dirs, if applicable.

Why these:
- they fit existing artifacts;
- deterministic;
- no new schemas needed if JSON report is internal first;
- they map directly to Karpathy lint concepts:
  - schema integrity;
  - orphan check;
  - coverage gaps;
  - stale/operational drift.

Test fixtures:
- clean workspace fixture;
- missing evidence fixture;
- orphan domain output fixture;
- proposal without evidence fixture.

## 10. Where not to integrate

### Not inside runtime prompts first

Do not solve this by telling Claude/Qwen/Codex "please lint better" in prompt packs.

Why:
- non-deterministic;
- weak testability;
- provider-specific behavior;
- repeats context burn problem from the gist comments.

### Not inside `qa.ask`

Do not make successful QA runs write canonical proposal/report/model files.

Why:
- violates current Q&A boundary;
- makes exploration unexpectedly mutating.

### Not by adding a generic `wiki/` directory

Do not add:

```text
wiki/
  index.md
  log.md
  concepts/
```

Why:
- duplicates `reports`, `model`, `charter`, `proposals`;
- creates unclear source of truth;
- invites provider-owned free-form writes.

### Not by storing accepted memory in SQLite/vector DB

Search index can come later as cache/projection.

Canonical truth remains:

```text
workspace.yaml
charter/
skills/
reports/
model/
proposals/
schemas/
docs/spec/
```

## 11. Resulting architecture if all selected ideas land

```text
Raw source layer
  workspace.yaml repos[]
  docs.imports_path
  human-owned charter/cards

Compilation layer
  init/refresh pipeline
  staged runtime artifacts
  final-run-index.json
  citation-index.json
  validator-verdict.json

Accepted architecture memory
  reports/as-is
  reports/coverage
  reports/findings
  reports/agent-outputs
  reports/diagrams
  model/entities
  model/edges
  proposals
  reports/changelog

Maintenance layer
  reports/health/workspace-health.json
  reports/health/workspace-health.md
  later: claim ledger and contradiction review

Exploration layer
  reports/taskruns/<run_id>/qa/context-pack.json
  reports/taskruns/<run_id>/qa/qa-answer.json
  explicit promotion -> proposals/qa-synthesis-*

Acceleration layer
  rebuildable search/context index
  not canonical
```

## 12. Final position

Karpathy's approach is valuable for ACP because it names the thing ACP is already becoming: a maintained compiled knowledge artifact rather than a chat transcript or one-shot RAG answer.

But the implementation must be ACP-native:
- deterministic before agentic;
- staged before canonical;
- proposals before mutation;
- health reports before self-healing;
- citation hardening before claim graph;
- rebuildable projections before search infrastructure.

The most useful immediate integration is not "build a wiki". It is:

```text
Workspace Health MVP + Ask-to-Proposal Promotion
```

Those two features import Karpathy's maintenance loop and compounding loop while preserving ProvenArch's strict local-first architecture control plane.
