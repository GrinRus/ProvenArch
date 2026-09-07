# Karpathy LLM Wiki vs ProvenArch

> Archived 2026-09-05 from `docs/KARPATHY_LLM_WIKI_COMPARISON_2026-06-18.md`. This dated research/design proposal is retained as rationale, not an active implementation plan or current contract. Implemented health, citation and Ask-to-Proposal behavior is described in the [current architecture](../../ARCHITECTURE.md) and [status matrix](../../STAKEHOLDER_DOC.md); future proposals still need an explicit slice.

Date: 2026-06-18

## Scope

This report compares Andrej Karpathy's "LLM Wiki" pattern with the current ProvenArch / ACP state.

Sources reviewed:
- Karpathy gist: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
- GitHub gist comments through the public comments API and rendered gist page.
- Local project sources of truth:
  - `README.md`
  - `docs/ARCHITECTURE.md`
  - `docs/spec/PIPELINE_SPEC.md`
  - `docs/spec/WORKSPACE_SPEC.md`
  - `docs/spec/MODEL_SPEC.md`
  - `docs/spec/API_SPEC.md`
  - `docs/TESTING_STRATEGY.md`
  - `docs/BACKLOG.md`

API note: during the first GitHub API pass I observed 899 comments, first comment `2026-04-04T16:49:23Z`, latest observed comment `2026-06-18T00:35:25Z`, 607 unique users, and 386 comments longer than 1000 characters. Later repeated API calls hit the unauthenticated GitHub rate limit, so this report uses the successfully collected aggregate pass plus the rendered gist page for source-visible examples. The comment thread contains a lot of thanks, product announcements, duplicate promotions and unrelated noise; the analysis below focuses on recurring architectural substance.

## Karpathy's Core Pattern

Karpathy describes a local, persistent knowledge base where raw sources remain immutable, an LLM-maintained wiki becomes the compiled knowledge layer, and a schema/instructions file such as `CLAUDE.md` or `AGENTS.md` tells the agent how to maintain it.

The key contrast is against classic RAG:
- RAG repeatedly retrieves raw chunks at query time.
- LLM Wiki compiles knowledge once into durable markdown pages, then keeps the compiled layer current.
- Query results can themselves be filed back into the wiki, so exploration compounds instead of disappearing into chat history.

The operational primitives are:
- ingest: read a new source, write/update summaries, entity pages, concept pages, index and log;
- query: answer against the wiki with citations, optionally filing useful synthesis back as a new page;
- lint: periodically check contradictions, stale claims, orphan pages, missing links and gaps;
- index/log: `index.md` for content navigation and `log.md` for chronological evolution.

The framing is intentionally abstract. Karpathy is proposing a pattern, not a product: Obsidian is a common frontend, markdown/git are the storage surface, search tools like qmd are optional, and the exact structure is expected to co-evolve with the user and domain.

## Comment Thread Signal

The thread is best read as a distributed design review. The repeated themes are consistent:

- Implementations appeared quickly. Many commenters shared CLI tools, Obsidian vault workflows, MCP servers, React UIs, SQLite-backed variants, skills, coding-agent memory systems and team wiki setups.
- Local-first is a dominant expectation. Many implementations emphasize no cloud requirement, plain markdown, Obsidian compatibility, Ollama/LM Studio support, and Git versioning.
- Provenance is the main hardening pressure. Multiple commenters argue that every claim needs source metadata, citation identity, hashes, spans, timestamps, lineage or auditability. The idea is not just "summarize sources", but "make each synthesized statement accountable".
- Markdown alone starts to strain at scale. Several commenters describe moving to SQLite/FTS, transaction logs, derived indexes, stable concept IDs, or block-level ASTs while keeping markdown as the human-readable surface.
- Contradictions need policy, not just detection. A substantive critique points out that the base pattern treats contradiction as a defect. In dialogic domains, disagreement can be the content. Commenters propose typed relationships such as `supports`, `contradicts`, `extends`, `supersedes`, plus policies that preserve tension instead of flattening it.
- Human verification remains load-bearing. The stronger implementations gate autonomous writes, route ambiguous contradictions to review queues, require citations, or treat the human as editor-in-chief.
- Agent memory is a major adjacent use case. Several comments adapt the pattern to coding agents: `.memory`, `.brain`, `hot.md`, project decision logs, architecture patterns, task handoffs and durable context across sessions.
- Query and token efficiency become real problems. Commenters note that `index.md` works at modest scale, but repeated context loading and large page counts push systems toward routing files, search indexes, caches, compact handoffs and queryable derived state.
- Team sharing pushes toward Git branches, PR review, MCP/API layers and path-scoped rules. A team wiki cannot rely only on one user's Obsidian workflow.

Rough keyword counts from the successful 899-comment API pass:
- RAG/search/vector/BM25 related: 391 comments
- structured DB/frontmatter/schema/typed graph related: 248
- Obsidian/vault/Dataview related: 243
- contradiction/conflict/lint/divergence related: 225
- MCP/API/server related: 185
- local/Ollama/no-cloud/offline related: 176
- provenance/citation/evidence/lineage related: 136
- verification/hallucination/trust/editor related: 120

These counts are approximate because they are keyword-based and include promotions, but they correctly show where the design pressure is.

## Current ProvenArch State

ProvenArch / ACP is a local-first architecture analysis control plane. It writes a Git-versioned architecture workspace for one or more local repositories. It is not a generic personal wiki, and it is not a hosted control plane.

Current documented baseline:
- Stack: Go backend/orchestrator plus embedded React/TypeScript UI.
- Runtime: deterministic `fake` baseline plus headless providers `claude-code`, `qwen-code`, `codex-code`.
- Workspace: central `arch-workspace` with fixed layout: `workspace.yaml`, `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/`.
- Output: normal files, including reports, diagrams, findings, proposals, agent outputs and entity-per-file YAML model.
- Contracts: JSON schemas for workspace and docs-first runtime artifacts.
- Pipeline:
  - step 0 constitution/charter;
  - step 1 collect context into shard-authored dossier packs and manifests;
  - step 2 as-is docs and indexes;
  - step 3 validator/findings;
  - step 4 proposals and promotion.
- Q&A: async runtime-backed `qa.ask` writes run-scoped answer artifacts; compatibility deterministic QA remains read-only.
- UI: stage-based console with Source, Readiness, Charter, Analysis, Review, Proposals, Ask and Publish.
- Release posture: required CI is deterministic and avoids live provider dependencies; live provider matrix remains manual trusted-machine pre-release gate.

The codebase matches the docs structurally:
- `internal/orchestrator` is the largest area and owns pipeline, sharding, task execution, docflow, promotion and run lifecycle.
- `internal/workspace`, `internal/runtime`, `internal/api`, `internal/model`, `internal/reports`, `internal/contracts` and `internal/runtimedrafts` map directly to the documented architecture.
- UI is already split into shell/stages/hooks: `OnboardingShell`, `StagePanels`, `ActiveRunStrip`, runtime settings, run review, git diff and API clients.
- Scripts and fixtures are substantial: contract validation, live E2E planning, release verdict verification, matrix harness tests, scenario fixtures and golden outputs.

Verification performed:
- `git status --short`: clean before adding this report file.
- `./scripts/run-go.sh test ./...`: passed.
- `python3 -m unittest discover -s scripts/tests -p '*_test.py'`: passed, 183 tests.
- `bash ./scripts/validate-contracts.sh`: blocked locally because `js-yaml` was not available outside the npm-driven Makefile path.
- `./scripts/run-npm.sh --version`: blocked because `.node-version` requires Node.js `22.21.1`, while the first discovered local Node was `25.9.0`; Codex bundled Node was `24.14.0` and did not include npm.

## Direct Comparison

### 1. Same underlying philosophy

ProvenArch is already aligned with the strongest part of Karpathy's pattern: durable, Git-friendly artifacts beat ephemeral chat. ACP's `reports/*`, `model/*`, `charter/*`, `proposals/*`, indexes and taskrun artifacts are the architecture equivalent of a compiled wiki.

Both systems value:
- local-first operation;
- raw/source inputs separate from generated knowledge;
- markdown/git as reviewable collaboration surface;
- persistent context across sessions;
- LLMs doing summarization, cross-reference work and bookkeeping;
- human curation and review as the final authority.

### 2. ProvenArch is much stricter than the gist pattern

Karpathy's schema is primarily an instruction document. ProvenArch turns that idea into explicit contracts:
- `schemas/*` define machine-readable boundaries;
- `docs/spec/*` define behavior;
- runtime writes are staged into `write_root` / `draft_final_root`;
- promotion is gated by validator verdicts and indexes;
- semantic outputs flow through manifests, not stdout;
- canonical cards are human-owned;
- runtime cannot silently create canonical domain/team cards;
- source repos are intended as read-only inputs.

This is the largest difference. Karpathy's pattern trusts a disciplined agent. ProvenArch assumes the agent will drift unless the orchestrator, schemas, validators and tests constrain it.

### 3. ACP has stronger provenance than the base LLM Wiki

The gist recommends citations and source truth, but leaves details open. ACP already requires provenance in the model, citation indexes, run indexes, repo/path evidence for observations, and strict semantic manifest fields. This directly answers a major concern from the comments.

Gap: ACP's provenance is strong for architecture entities, edges, findings and reports, but there is not yet a generalized claim ledger where every narrative claim has stable identity, supersession state, contradiction edges and review lifecycle. If ACP grows into a broader architecture knowledge memory, that will become the next useful hardening layer.

### 4. ACP treats relationships as architecture topology, not general knowledge graph semantics

Karpathy's wiki uses wikilinks and informal cross-references. Commenters push toward typed edges for support/contradiction/supersession. ACP already has typed edges, but the vocabulary is architecture-specific: `calls`, `publishes`, `subscribes`, `reads`, `writes`, `exposes`.

That is correct for MVP. It would be a mistake to import an open-ended ontology into the current model without a slice. If needed later, add a separate claim/evidence relationship layer rather than overloading service topology edges.

### 5. ACP has a clearer write authority boundary

Karpathy says the LLM "owns" the wiki layer. ACP deliberately does not give the provider direct ownership over canonical workspace surfaces. Runtime-authored outputs are staged and validated; canonical publication is orchestrator-controlled.

This is a major practical improvement over many LLM Wiki implementations. It reduces silent corruption, path mistakes, schema drift and accidental writes into protected surfaces.

### 6. ACP is less exploratory by design

Karpathy's pattern encourages exploratory compounding: a useful answer becomes a new page. ACP's async QA writes run-scoped `qa-answer.json` and does not mutate canonical outputs. That is safer, but it means good Q&A synthesis does not yet compound into the stable architecture workspace unless converted through proposals or a future curated promotion flow.

This is a real product choice:
- Karpathy optimizes for personal knowledge growth.
- ACP optimizes for controlled architecture evidence and reviewable outputs.

### 7. ACP already solves many comment-thread failure modes

| Comment-thread concern | ACP state |
| --- | --- |
| Hallucinated notes become truth | Validator gates, provenance, fake baseline, required tests, staged promotion. |
| Source lineage is lost | Evidence/provenance, citation index, final run index, taskrun metadata. |
| Agent writes corrupt files | Staged write roots, write-set guards, runtime write audit, protected surfaces. |
| Index/frontmatter drift | JSON schemas, semantic validation, fixtures, golden tests. |
| Scale needs deterministic routing | Sharding planner, context packs, run-scoped artifacts, model store. |
| Team review needed | Git workspace, Publish/Git Review Room, proposals, changelog. |
| Agent memory across sessions | Workspace artifacts, run history, reports, model and charter act as durable state. |

### 8. ACP still has gaps if judged as an LLM Wiki system

These are gaps only if ACP wants to absorb more LLM Wiki behavior:
- no general-purpose wiki page ontology for arbitrary concepts;
- no stable claim registry with claim-level contradiction/supersession state;
- no derived FTS/SQLite search index as a rebuildable query accelerator;
- no MCP memory service surface;
- no curated "promote this QA answer into canonical synthesis" workflow;
- no generalized lint for orphan/stale/duplicate narrative pages beyond current contract/validator checks.

None of these are required for the current MVP mission. Several would be harmful if rushed into the existing architecture model.

## How Different Are They?

Short answer: philosophically close, operationally very different.

Karpathy's pattern is a lightweight, agent-instantiated memory architecture:

```text
raw sources -> LLM-maintained markdown wiki -> index/log -> query/lint
```

ProvenArch is a productized, domain-specific, contract-gated control plane:

```text
repos/imported docs -> orchestrated runtime steps -> staged artifacts -> schemas/validator -> promoted architecture workspace
```

The overlap is high in storage philosophy and compounding-artifact thinking. The divergence is high in governance, validation, domain specificity and runtime control.

I would not describe ProvenArch as "different from Karpathy". I would describe it as one of the stricter possible industrializations of the same core idea, specialized for software architecture evidence.

## Recommendations

1. Do not pivot ACP toward a free-form Obsidian-style LLM Wiki.

ACP's advantage is contracts, provenance, staged promotion and deterministic CI. Keep that.

2. Adopt the "compiled knowledge artifact" language more explicitly.

README already says ACP outputs are files, not chat logs. It may be worth describing `arch-workspace` as a compiled architecture knowledge base, with source repos/imports as raw inputs and reports/model/proposals as promoted derived state.

3. Add future backlog consideration for a claim/evidence ledger.

A minimal future slice could model claim identity, source support, contradiction, supersession and review status. This should be separate from service topology edges.

4. Consider a rebuildable derived search index, not a new source of truth.

If QA/context loading becomes expensive, follow the better comment-thread pattern: keep markdown/YAML canonical, build SQLite/FTS/BM25/graph projections as disposable derived state. Do not make the index authoritative.

5. Keep Q&A canonical mutation gated.

The Karpathy "file good answers back into the wiki" loop is valuable, but ACP should implement it as explicit curated promotion, likely through proposals or a review queue, not automatic writes from `qa.ask`.

6. Strengthen semantic health checks separately from contract checks.

ACP has strong schema validation. A future "workspace health" pass could check stale architecture claims, duplicate entities, orphan reports, missing backlinks, unresolved questions aging, contradictory findings and citation decay.

7. Preserve current MVP boundaries.

The gist and comments mention hosted sharing, MCP services, autonomous agents and broad memory systems. Those are not MVP requirements. ACP should stay local-first, non-hosted, and not become a generic memory platform in this phase.

## Bottom Line

Karpathy's approach and ProvenArch share the same core thesis: useful LLM work should accumulate into durable, inspectable, Git-friendly artifacts rather than vanish into chat or be recomputed from raw RAG each time.

The difference is that Karpathy presents a flexible pattern for personal/team knowledge bases, while ProvenArch already implements a much more constrained architecture-analysis pipeline with schemas, validators, staged writes, model materialization, runtime profiles and release gates.

The strongest import from the Karpathy thread is not "add a wiki". ACP already has one in a stricter form. The strongest import is to treat future architecture knowledge as claim-level, provenance-backed, reviewable memory: stable identities, typed evidence relationships, explicit contradiction/supersession policy, and rebuildable search projections.
