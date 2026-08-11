# Стратегия тестирования ACP MVP

Этот документ фиксирует baseline testing strategy для ACP MVP.

## 1) Цели и принципы

- Required CI должен проходить локально и в CI без live network dependencies.
- Required CI не зависит от live headless providers (`claude-code`/`qwen-code`/`codex-code`), GitHub, GitLab или реальных пользовательских репозиториев.
- Любые изменения schema/spec/examples должны сопровождаться обновлением fixtures и golden outputs в том же PR.
- Synthetic fixtures считаются baseline regression surface.
- Live headless providers проверяются только optional smoke на trusted machine/runner и не блокируют merge.
- Отдельно от merge-gates используется manual pre-release live gate:
  - `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
  - machine execution verdict `PASS|FAIL` с policy strict zero-failure.
  - final release readiness = execution `PASS` + accepted SWE UX assessment + accepted SWE artifact-quality assessment.

## 2) Тестовая пирамида MVP

### Contract tests
- `workspace.yaml` валидируется по `schemas/workspace.schema.json`
- docs-first contracts валидируются по:
  - `schemas/shard-pack-manifest.schema.json`
  - `schemas/final-run-index.schema.json`
  - `schemas/citation-index.schema.json`
  - `schemas/validator-verdict.schema.json`
  - `schemas/source-revisions.schema.json`
  - `schemas/refresh-impact-plan.schema.json`
  - `schemas/qa-answer.schema.json`
  - `schemas/source-qa-answer.schema.json`
- persisted `runtime-execution.json` metadata и artifact-only step contracts проходят parse/semantic validation
- examples и fixture cases должны парситься и проходить contract validation, где это ожидается
- Ask-to-Proposal tests cover digest staleness, citation authority, exclusive atomic directory
  publication, injected write/rename rollback and ProductShell focus/routing/Git refresh

### Semantic validator tests
- правила, которые не выражаются чистой JSON Schema
- deterministic canonicalization top-level `questions/coverage`
- stable ID normalization и collision rules
- ownership/card linkage constraints
- W24B verdict fixtures reject `PASS` plus technical errors, provider-authored `fixed_paths`, empty
  effective `FAIL`, duplicate/conflicting issue identities, unstable issue order and dangling
  selected-run document/citation/path references while preserving advisory owner/source gaps until
  explicit reconciliation.
- W24D semantic fixtures reject unknown nested fields, conflicting cross-shard ID collisions and dangling edge
  endpoints while preserving unresolved finding/question references as advisory coverage gaps.
- W24E promotion fixtures run the public selected-run scanner before activation, fail closed on
  contaminated staged output, and verify the previous canonical generation remains unchanged.
- W24F authority fixtures keep provider `validator-verdict.json` byte-identical, persist a separate
  effective verdict, prove provider PASS/FAIL cannot override deterministic technical status, retain
  unmatched provider issues as warning-level advisory records, and expose legacy/unavailable status
  when an effective artifact is absent. Retry fixtures rebind effective paths to child runs.
- W24C provider-free evidence fixtures exercise one shared bounded implementation across collect,
  staged validation and selected-run audit: 1-based inclusive ranges, CRLF/CR-to-LF normalization,
  exact whitespace/Unicode preservation, excerpt bytes and SHA-256 identity.
- Negative evidence fixtures cover invalid UTF-8, empty/reversed/out-of-range ranges, excerpt/hash
  mismatch, missing explicit range, oversized source/line/excerpt limits and symlink/path escapes;
  semantic provenance evidence with no bounded line assertion remains advisory path-only input.

### Golden/regression tests
- docs-first staged + promoted outputs (`reports/*`, `proposals/*`)
- model store materialization как derived layer
- diagrams/compat outputs как thin-code layer
- deterministic comparisons against recorded golden outputs для `fake` + artifact-fixture baseline
- hash-based snapshot compare against `fixtures/scenarios/*/golden/snapshot.sha256`
- для live/headless acceptance больше не требуется byte-identical narrative markdown; обязательны structural contracts: shard-plan shape, manifest/index schemas, publish invariants и absence of direct canonical writes from runtime

### Scenario integration tests
- pipeline runs на synthetic repos и fixture workspaces
- artifact fixtures without live providers в required tests
- Workspace Health fixtures cover every advisory v1 issue class, deterministic ordering,
  workspace containment and byte-identical read-only scans; response version remains `1`
- fixture contract gate проверяет parse/semantics recorded artifacts (`meta.step_id`, `repo_scopes`)
- `git_url` freshness проверяется только на local bare remotes: unpinned cache должен fetch/reset-иться на новый remote default `HEAD`, pinned SHA/ref остаётся выбранным ref, а `path` checkout не мутируется
- collect contract fixtures must include at least one authored document and one repo-backed
  citation; sparse `documents: []`, `citations: []`, empty document/citation binding arrays,
  unknown citation document IDs, and one-way document/citation bindings are negative fixtures,
  not valid minimal examples

### Smoke tests
- CLI smoke
- API smoke
- UI smoke

### Optional live-runner smoke
- только manual/opt-in
- не входит в required CI gates

### Headless provider conformance
- required tests используют stub provider adapters без live network dependencies
- общий process engine проверяется на success by valid artifacts, controlled stop after valid artifacts, qwen draft valid-artifact stop after continued stream/mutation, focused repair valid-artifact stop after provider overrun, collect pair recovery, collect pair recovery after exhausted fully silent collect retry, collect pair repair invalid/no-artifact failure, collect pair repair fresh-mutation threshold plus default 5-minute pre/post/partial artifact windows, focused collect-pair retry after a silent no-fresh pre-artifact repair stall, exhausted silent no-fresh pair repair classified as `runner_unavailable`, structural missing repo evidence in existing markdown escalating to collect pair repair with required markdown rewrite, process-contaminated collect markdown escalating to collect pair repair with required markdown rewrite, stale/noop markdown pair repair remaining terminal, partial collect pair repair markdown-only stall chaining into explicit manifest runtime recovery, invalid observed artifact stall despite active stdout when there is no fresh mutation, pre-artifact wall-clock stall despite active stdout/stderr before any authored artifact, collect manifest-only write-first repair success/failure без read-only preflight-only success, deterministic collect manifest runtime recovery from temp authored-doc trees before provider repair for missing/scaffold-only manifests, structural-invalid collect manifest repair exhaustion as terminal `runtime_contract_failed` without deterministic post-repair recovery, transient provider API/transport failure retry during qwen collect-pair and draft-artifact repair, validator-verdict-only repair, draft-artifact repair, direct bootstrap-only draft validation -> `draft_artifact_enrichment`, scaffold-only draft repair stall -> `draft_artifact_enrichment`, enrichment success only when every referenced markdown actually changes and is marker-free while preserving manifest `outputs[].path/canonical_path`, enrichment noop/scaffold/manifest-alias failure -> `draft_artifact_enrichment_noop_or_scaffold` or `runtime_contract_failed`, one-shot provider-authored `draft_artifact_enrichment_markdown_syntax` retry after fresh enrichment fails only malformed inline-code/code-fence syntax, stale draft files ignored until fresh enrichment mutation, step2 enrichment write-first coverage for all three targets (`overview.md`, `summary.md`, `architect-summary.md`) with shard completeness and operator decision content, rejection of draft markdown that copies bootstrap manifest summary markers such as `Drafted required runtime artifacts` and rejection of recovery-process narration such as `current draft manifest`, `manifest target remains`, `bounded staged evidence`, `bounded evidence read`, `bounded read roots`, `bounded read pass`, `recovery pass`, or `enrichment read`, acceptance of operator-facing step0 coverage gaps that say a repo surface was not inspected in a bounded read without describing recovery mechanics, bounded draft enrichment include scope that excludes whole workspace/repo for `step2/step4`, bounded pre-artifact and repair stall windows, qwen recovered zero-output pre-artifact retry warning, scoped Claude constitution/collect/validator/proposals zero-output retry warning, exhausted silent/API no-artifact classification after collect repair is unavailable/exhausted, interrupted temporary collect markdown (`first bounded evidence read was attempted` / `initial artifact records only` / `will be repaired with concrete`) and process-contaminated collect markdown (`bounded read/pass`, guessed path/file/evidence, expected-missing concrete path checks) as contract-invalid artifacts, invalid artifact contract failures, deadline timeout и raw stdout/stderr + redacted lifecycle diagnostics, включая resolved timeout profile and `pre_artifact_wall_clock_window_ms`
- draft enrichment regression tests also cover provider-authored missing interpreter commands (`python: command not found` / `command not found: python`): runtime may do one `draft_artifact_enrichment_python3_retry`, but only provider-authored fresh markdown rewrites can pass validation.
- draft enrichment regression tests cover terminal noop/scaffold enrichment: if enrichment leaves bootstrap/noop markdown and validation reports `draft_artifact_enrichment_noop_or_scaffold`, runtime must emit exhausted draft enrichment telemetry and fail as `runtime_contract_failed` without scheduling a generic no-action retry. The first no-fresh exception is a single provider-authored `draft_artifact_enrichment_write_first_retry` after a silent pre-artifact stall with empty stdout/stderr, no fresh markdown mutation and any already-proven strict draft validation failure, including truthfulness or repository-reference failures in otherwise substantive markdown. The only follow-on no-fresh exception is step2-only `draft_artifact_enrichment_compact_step2_retry` when that write-first retry also stalls silently before mutating markdown; tests require it to be non-recursive, limited to `init|refresh.step2.asis_docs`, and to use a compact evidence set before overwriting `overview.md`, `summary.md`, and `architect-summary.md`. Repeated silent/noop/scaffold/invalid output after those narrow retries remains terminal. Prompt/validation tests still require `init.step0.constitution` to name the exact `charter-overview.md` target and bounded repo entrypoint candidates in the focused enrichment prompt, reject downstream/final/shard/validator/proposal/coverage/taskrun/runtime-provider leakage in constitution markdown, require exact typed shard-summary completeness for `step2` when `items[]` is visible, and reject metadata-key dumps (`meta`, `step_id`, `domain_id`, `strategy`, `max_parallel_tasks`, `failure_policy`, `shard_discovery_mode`), false zero-shard/zero-file claims, and empty shell-substituted evidence slots such as `from  and`, `checked:  and`, or `under .`. For `step4`, focused enrichment and command-text retry require bounded evidence reads before writing both `proposal.md` and `changelog.md`, including non-empty required sections, the exact literal `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` shard-completeness string plus `no-shard-coverage-blocker` in both files when typed status is visible, concrete current-run evidence refs, no placeholder actionable Finding ID values such as `none` when current-run findings are non-empty, and a decision-ready no-actionable-proposal gap when evidence is sparse.
- Normal step2 first-pass tests require at most one bounded evidence read/list command followed immediately by one mechanically simple direct-literal write command. Before all four targets exist, the prompt prohibits Python/Node/Ruby/Perl/awk/jq inline generators, template programs and nested quote tricks; it requires single-quoted heredocs, all three Markdown targets first, a case-insensitive runtime/recovery marker check, and `asis-draft-manifest.json` last. Reduced provider-free tests protect both the live-observed inline-Python `SyntaxError` path and the earlier `bounded evidence read` / runtime-assembly narration regression without adding matrix identity or live-only behavior to production code.
- draft enrichment regression tests also cover the step4-only compact retry: if `init|refresh.step4.proposals` repeats a silent no-fresh failure during `draft_artifact_enrichment_write_first_retry`, runtime may schedule exactly one provider-authored `draft_artifact_enrichment_compact_step4_retry`. The prompt is non-recursive, avoids heredoc/bootstrap scaffolds, reads only compact current-run proposal evidence, and must overwrite both `proposal.md` and `changelog.md`; repeated silent/noop/scaffold output remains terminal `runtime_contract_failed`. Step4 contract tests also use a live-observed Claude fixture to reject `reports/taskruns/**`, `staging/final/**` and `staging/shards/**` input locators in user-visible proposal/changelog markdown while accepting canonical findings/coverage paths, exact finding/citation IDs and stable repo:path evidence.
- draft enrichment regression tests also cover printed-command retry output: if provider-authored `draft_artifact_enrichment` prints shell/Python command text such as `python3 - <<...` without file mutation, runtime may do one `draft_artifact_enrichment_command_text_retry`; success still requires provider-authored fresh rewrites, and repeated printed-command/noop/scaffold output remains terminal. Normal, repair, compact and command-text `step2.asis_docs` prompts are each contract-tested to require the exact eight non-empty Architecture Home sections as standalone H2 lines and exact non-root repo evidence paths; repository-root shorthand and wildcard/glob syntax are explicitly prohibited, and the normal first-pass self-check must scan each `repo:path` token for glob metacharacters before manifest write. A provider-free Bank-shaped fixture proves that valid current-run nested evidence `bank-of-anthos:src/ledger/cloudbuild.yaml` is exposed byte-for-byte in a deterministic allowlist while inferred root `bank-of-anthos:cloudbuild.yaml` is never produced. Missing-reference repair is contract-tested to use direct literal heredocs and forbid Python/Node/template assembly. Legacy generic `Architecture Surface` / `Evidence Used` / `Coverage Gaps` headings are not accepted substitutes. A provider-free fixture covers the narrow all-eight inline-heading shape: normalization preserves every authored body, inserts only heading/body separators, is deterministic, skips provider repair after strict success, and rejects partial/duplicate/out-of-order/empty/fenced or multiply-invalid input. Failed candidate revalidation restores the original document byte-identically. Reduced authored fixtures also reproduce `repo:.` and `repo:src/*`; validator tests reject both while accepting an exact existing file/directory reference. Live-observed Claude fixtures verify that Architecture Home containing `reports/taskruns/**/staging/**`, absolute runtime checkout / `.acp/repos` paths, run/current-run identity, any typed-shard/shard-pack recap, exact `planned/succeeded/failed/incomplete` counters, or missing `<repo>:<path>` evidence is rejected before promotion. Repository-reference tests cover existing files/directories, missing paths, traversal and symlink escape; canonical artifact references remain valid.
- draft enrichment regression tests also cover manifest-shape drift: if enrichment fresh-rewrites evidence-backed markdown but adds unknown manifest fields (`status`, `content_digest`, `enriched_at`, aliases), runtime may do one provider-authored `draft_artifact_enrichment_manifest_shape` retry; strict parser behavior stays unchanged and repeated drift remains `runtime_contract_failed`.
- draft enrichment regression tests also cover stale downstream final/citation-index claims: if enrichment fresh-rewrites evidence-backed markdown but only fails because `step2` claims current-run final/citation indexes are unavailable, not yet present/not yet available, or a draft claims unvalidated zero-document final-index evidence, runtime may do one provider-authored `draft_artifact_enrichment_downstream_index_retry`; repeated stale claims remain terminal. Strict draft validation also rejects metadata-only evidence bullets and mismatched non-zero final/citation index counts when current-run index files are present.
- downstream-index retry selection is tested as an exclusive validation class: a fixture containing both Architecture Home process narration and a stale downstream-index sentence must route to `draft_artifact_enrichment_architecture_home_cleanup`, never the downstream-only retry. The regression uses provider-free filesystem scripts and validates the same public draft contract; it contains no matrix IDs or live-harness branches.
- draft enrichment regression tests cover both live-observed Architecture Home paths. After every step2 markdown target is freshly rewritten, a remaining strict `runtime/process narration, manifest recap, or unsupported confidence` failure may schedule exactly one provider-authored `draft_artifact_enrichment_architecture_home_cleanup`. Separately, when that exact error is the initial/repair cause and only `overview.md` is freshly rewritten, runtime may preserve byte-identical valid `summary.md`/`architect-summary.md` only after strict full-set validation passes. The exception is step2-only, requires a changed overview, and does not apply to any other validation error or partial/noop enrichment. Fixtures include the rejected sentence `scoped to the current run` and 20-run focused stress coverage.
- draft enrichment regression tests also cover marker cleanup: if enrichment fresh-rewrites every markdown target but strict validation rejects only process/scaffold/downstream wording (`bounded read roots`, `current draft manifest`, `validator output`, step0 later-pipeline wording), runtime may do one provider-authored `draft_artifact_enrichment_marker_cleanup` retry; repeated marker contamination or unchanged scaffold remains terminal.
- draft validation regression tests distinguish recovery placeholder narration from domain findings: scaffold wording such as `replace placeholder content` remains invalid, but evidence-backed proposals may recommend replacing placeholder credentials when they cite concrete current-run repo paths or citations.
- draft enrichment regression tests also cover semantic repair routing, exact shard-status cleanup and write-set cleanup: if draft repair creates proposal/changelog markdown that is structurally present but semantically invalid (for example missing proposal sections or exact proposal shard completeness), runtime must route to provider-authored `draft_artifact_enrichment` instead of terminal scaffold failure. If step2 enrichment fresh-rewrites markdown but leaves generic conditional shard-gap wording, runtime may do one `draft_artifact_enrichment_shard_status_cleanup` retry that rewrites all step2 markdown targets with exact typed shard counts. If step4 proposal/changelog fresh-rewrites markdown but omits exact proposal shard completeness, the same cleanup path rewrites `proposal.md` and `changelog.md` with exact `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` plus `no-shard-coverage-blocker` while preserving finding IDs/actionability. If the same shard-completeness wording appears inside a mixed `draft_artifact_enrichment_noop_or_scaffold` failure, cleanup must not run because there was no fresh markdown to clean up; runtime must use the no-fresh write-first/compact path or exhaust as `runtime_contract_failed`. If enrichment writes byte-identical referenced markdown duplicates into `write_root`, runtime may do one `draft_artifact_enrichment_write_set_cleanup` retry that deletes only those misplaced duplicates while keeping valid markdown under `draft_final_root`; for step0 constitution, the same cleanup path may delete only a mistaken `draft_final_root/skills/subagents.yaml` canonical-path duplicate while preserving `baseline-subagents.yaml`. A live-observed `amp` fixture also proves that a newly created regular zero-byte sidecar is rolled back before strict revalidation and a subsequent Architecture Home cleanup; non-empty extra files and every other forbidden mutation remain terminal.
- draft enrichment regression tests also cover the exact validator phrase `does not report exact current-run shard completeness from typed shard summary`; it must schedule the same `draft_artifact_enrichment_shard_status_cleanup` path only for fresh markdown rewrites and must not recursively schedule itself.
- collect recovery regression tests cover process-contaminated pair repair followed by manifest-only repair when the provider fresh-rewrites clean markdown but leaves only manifest schema binding invalid, such as missing `citations[].document_ids`; they also cover the live-observed `citations[].claim_ids: []` case, where deterministic `collect_manifest_runtime_recovery` reconstructs the manifest from clean provider-authored markdown and emits recovery diagnostics/warnings. A tracked rich-manifest fixture without `semantic.findings` verifies the narrower shape recovery: only an empty findings array is inserted, existing values stay structurally identical, output digest is deterministic, provider repair is skipped on success, and any second defect restores the original bytes before the normal fail-closed path. Stale/noop/process-contaminated markdown still remains terminal.
- Qwen prompt regressions pin live-observed compact-contract drift: normal bounded collect names the closed root allowlist before atomic pair write, enumerates validator-rejected process phrases including `bounded read`, and restricts every minimal semantic provenance kind to exact `observation`; unsupported inference becomes a coverage gap. The root `claims` fixture additionally proves that an exact empty-instance-path/root-schema `additionalProperties 'claims'` diagnostic selects a compact prompt below 2.4 KiB with exactly one manifest read and one same-target write. Nested `questions[].citation_ids`, unknown root fields and non-Qwen providers retain the full manifest repair contract; all rewrites remain subject to complete backend validation.
- collect recovery regression tests also cover digest-only partial collect output: if the provider writes only a non-contract file such as `collect-digest.txt` and no authored `.md`/`.markdown`, runtime must schedule `collect_pair_repair` with a fresh markdown+manifest pair and must not schedule manifest-only repair first.
- runtime write audit regression tests require protected workspace and analyzed repo mutations after provider execution to fail otherwise-successful steps as `runtime_contract_failed`; read-only isolated repo clones must pass through a lightweight HEAD/index/mode snapshot without writable `git status`; initial non-git read roots remain warning-only diagnostics, but an audited repo that becomes unreadable after runtime is fatal.
- provider-specific tests проверяют только adapter policy/args/env: `qwen` использует stream-json activity output без semantic stdout contract, не передаёт JSON task stdin при `-p` invocation, нормализует custom prompt args к artifact prompt, не отключает artifact/pre-artifact monitoring при custom args, включает bounded transient provider-unavailable focused repair retry for collect-pair/draft-artifact no-artifact cases и применяет bounded valid-artifact stop к normal draft steps; default `codex-code` args включают disabled plugin/app suggestion surfaces (`plugins`, `remote_plugin`, `plugin_sharing`, `apps`, `enable_mcp_apps`, `tool_suggest`, `skill_mcp_dependency_install`) и `--ignore-user-config`/`--ignore-rules`, а `ACP_CODEX_MODEL`/`ACP_CODEX_REASONING_EFFORT` добавляют explicit `--model`/`model_reasoning_effort` args для live E2E pin без изменения qwen/claude defaults; command spec создает isolated auth-only `CODEX_HOME` без пользовательских `config.toml`, MCP/plugins, app tools, `.tmp/plugins` и rules, collect activity policy имеет 5-minute initial/retry pre-artifact window, а normal draft activity policy имеет 180s first-command pre-artifact window aligned with Claude/Qwen; `claude` collect activity policy has a 5-minute initial/retry pre-artifact window while draft/enrichment windows remain unchanged; collect artifact tasks для live adapters получают extended post-/partial-artifact enrichment window перед repair, но unchanged normal collect seed pair не является valid artifact-only success; shared focused repair policy добавляет bounded valid-artifact stop для repair attempts независимо от provider; `claude` retry policy включает zero-output pre-artifact warning/retry только для constitution/collect/validator/proposals steps; `claude`/`codex` machine-mode flags остаются diagnostic transport mode
  - prompt contract tests покрывают constitution command-first manifest+draft-file heredoc targets, валидный YAML first-action `baseline-subagents.yaml` для canonical `skills/subagents.yaml`, normal collect `COLLECT EVIDENCE-FIRST ARTIFACT PAIR` до broad filesystem contract, `FIRST COLLECT BOUNDED WRITE ACTION` как bounded read/list + direct literal write work unit, bounded reads по entrypoint/path scopes, file-level citation/provenance checks (`test -f`, `rg --files`, portable `find ... -type f -print`) вместо syntax-only JSON proof, запрет normal analysis-only prose / second read-only preflight / seed-only heredoc / temporary “will be repaired” prose / process narration in final markdown / Ruby/Node/Python/Perl/awk/jq generated inline writers before both collect targets exist, skeleton-copy rejection, legacy marker `ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT` rejection, validation blocker для unchanged marker-free seed/recovery fallback collect docs, validation blocker для interrupted temporary collect docs, validation blocker для process-contaminated collect docs, validation blocker для scaffold-only collect semantic, no pair-repair success для bootstrap-only authored doc, collect-specific entrypoint hints, collect pair recovery write-first/evidence-bounded prompt without seed heredoc or separate `collect_pair_repair_preflight`, compact collect pair prompt for no-artifact stalls, directory-only evidence refs and process-contaminated markdown, required non-empty `citations[].claim_ids` and `citations[].document_ids` in compact collect pair manifest guidance, validation-specific pair-repair focus for directory-only repo evidence refs, process-contaminated markdown, and no-artifact stalls, exact final targets, existing authored markdown target selection for missing repo evidence or process-contaminated repair, bounded evidence candidates, banned recovery/process markers and final self-check, prompt rule that the first pair-repair item is one filesystem command that writes markdown plus manifest before prose, prompt rule that plan/status/analysis-only phrases like “I have enough evidence” or “I am now writing” are invalid before filesystem writes, candidate filtering for lockfiles/generated baselines/test-duration indexes/large files, low-signal `Recovery Summary`/`Recovery Bootstrap`/`Recovery Evidence Summary` scaffold rejection, collect manifest-only repair `COLLECT MANIFEST EVIDENCE-FIRST REPAIR`, `FIRST COLLECT MANIFEST REPAIR COMMAND`, literal manifest JSON skeleton as schema guide only, write-first provider-authored manifest command без read-only preflight-only success, runtime recovery tests that build temp authored markdown trees instead of persistent bad-run fixtures, placeholder/skeleton-copy rejection, anti-collapse semantic extraction requirements for evidence-rich authored docs, as-is/proposals evidence-first first-action prompts with exact manifest/markdown targets and no bootstrap heredoc in the normal happy path, as-is first-action `FIRST AS-IS DRAFT COMMAND` as bounded evidence read/list followed by validation-ready `overview.md`/`summary.md`/`architect-summary.md` writes before manifest-last `asis-draft-manifest.json`, and a step2 rule that the first provider item must be command execution rather than assistant/status prose, validator first-action command-first verdict heredoc target, validator issue canonical shape/legacy bans, proposals first-action `FIRST PROPOSALS DRAFT COMMAND` as bounded current-run findings/coverage/index read/list followed by validation-ready proposal/changelog writes before manifest-last `proposals-draft-manifest.json`, required proposal sections (`Decision / recommended operator action`, `Evidence used`, `Proposed changes or follow-up plan`, `Risks, gaps, and out-of-scope notes`) and required changelog sections (`Updated architecture/proposal surfaces`, `Findings/proposals summary`, `Evidence index or citation references`, `Residual coverage gaps`), draft recovery command-first manifest+draft-file heredocs, draft enrichment prompt без heredoc scaffold, с fresh mutation requirement, command-first/write-first bounded evidence command, step0 constitution enrichment write-first target `charter-overview.md` with bounded repo entrypoint evidence, no dependency on later pipeline evidence, and validation rejection for downstream/runtime-only leaks, step0 include-dir coverage for resolved `git_url` repo roots from `read_context_roots`, focused enrichment minimum 3-minute pre-artifact window, banned analysis-only phrases like “I have enough evidence” before mutation, banned bootstrap markers including copied manifest summaries, manifest `outputs[]` preservation plus bans for `logical_path`, `status`, `content_digest`, `enriched_at` and other unknown manifest fields, manifest-shape retry prompt coverage, downstream-index retry prompt coverage, marker-cleanup retry prompt coverage for process/downstream wording after fresh rewrites, exact root-variable targets that avoid retyping long slash-separated provider paths, step2 as-is enrichment write-first targets `overview.md`/`summary.md`/`architect-summary.md` with typed/observed shard completeness instead of lexical failed/error counts, shard-completeness cleanup focus for the validation-flagged target, coverage gaps and operator decision summary, step4 proposals enrichment write-first targets `proposal.md`/`changelog.md` with decision/evidence/gap structure, no manifest-summary-as-finding contamination, command-text retry bounded write-before-analysis behavior, terminal noop/scaffold enrichment exhaustion, bounded draft enrichment include-dir tests without whole workspace/repo for `step2/step4`, and root-file shard hints без live network dependency
- batch preflight tests покрывают selected-provider readiness без live network dependency, включая live E2E Codex default `ACP_CODEX_MODEL=gpt-5.6-luna` + `ACP_CODEX_REASONING_EFFORT=high`, CLI version mismatch guard, artifact smoke pass/fail/timeout paths, full process-group termination for timed-out probe children that keep pipes open, и low-disk guard, который materialize-ит `operational_host_preflight_failed` до child batch
- batch precheck script tests покрывают bounded DoD/UI prechecks: зависший `make contracts test lint build`, `npm ci --prefix ui` или `playwright install chromium` должен завершать batch как `precheck_failed` с `[precheck-timeout]` evidence в соответствующем precheck log, а не оставлять profile/matrix бесконечно `running`.

## 3) Обязательная структура test assets

- `fixtures/workspace/` — manifest и validator cases
- `examples/*.example.json` + contract tests — docs-first fixtures (manifest/index/citation/verdict)
- `fixtures/refresh-planning/*` + `internal/refreshplan` tests — revision baseline, complete Git name-status parsing, advisory mapping, legacy baseline rejection, dirty/history/unmapped fallback and exact 10,001-path safety limit
- `fixtures/scenarios/<name>/workspace/` — central workspace inputs
- `fixtures/scenarios/<name>/repos/<repo-name>/` — synthetic repos
- `fixtures/scenarios/<name>/golden/` — expected deterministic snapshot (hash list) + fixture docs

Baseline scenario set:
- `single-service-http-postgres-gitlabci`
- `two-services-http-call-and-queue`
- `missing-owner-and-missing-cicd`

## 4) Обязательные semantic checks

- duplicate `repo.name` rejected
- unsupported manifest fields rejected
- top-level `questions/coverage` canonicalize deterministically
- legacy `add_question` / `set_coverage` rejected contract validation
- `observation` without evidence rejected
- semantic stdout payload не используется как content write path
- `owner_team_id` должен ссылаться на существующий `team.<slug>`
- stable ID normalization использует canonical slug rules
- collision suffix `.repo-<repo-slug>` применяется детерминированно
- rename/move проходит через `aliases[]`, а не silent re-key
- Step 1 runtime не auto-create-ит canonical domain/team cards
- Step 0 wizard contract wiring: valid contract влияет на charter/cards; missing/invalid contract даёт fallback + run warning
- workspace validate выдаёт layout readiness diagnostics (`missing`/`not_dir`/`unreadable`)
- async lifecycle operability:
  - `CancelRun` для pending run даёт immediate terminal `failed` + `error_code=run_canceled`
  - `CancelRun` для active run даёт cooperative cancel + `failed` + `error_code=run_canceled`, очередь продолжает работать
  - workspace-owned persistence использует fault-injection tests для atomic write failure points: before write, before rename и parent directory sync; failed writes must not leave partial current JSON or stale temp files
  - run-history persistence пишет `.last-good`; service startup recovers from malformed current `reports/taskruns/run-history.json` when the last-good copy is valid and records a recovery diagnostic path
  - async run panic isolation covers terminal `failed/internal_failure` history, service survival after a panicking runner, active-slot/cancel cleanup and pending-run continuation; direct `Service.Run` panic tests continue to require caller-visible re-panic
  - server-owned shutdown tests cover active run context cancellation, queued pending run `run_canceled` terminalization without runner start, post-shutdown `ErrServiceClosed`, and API `Serve` context cancellation waiting for orchestrator shutdown
  - coherent API session generation tests cover request-scoped workspace/service/runtime snapshots, direct/onboarding effective runtime readback, `409` conflicts for workspace switch/runtime switch/runtime profile mutation during active async work, unchanged manifest on conflict, and concurrent polling plus mutation attempts under `go test -race ./internal/api`
  - initial async run queueing returns a history persistence error before launching the background run when `run-history.json` cannot be written
  - stale persisted `queued` run при старте сервиса reconciled в `failed` + `error_code=run_reconciled_after_restart`
  - stale persisted `running` run auto-resume-ится с тем же `run_id`, если присутствуют resumable shard artifacts; иначе reconciled в `failed` + `error_code=run_reconciled_after_restart`
- runtime timeout control:
  - persisted profile в `workspace.yaml.runtime.profile.timeouts`
  - effective precedence `env > workspace > defaults`
  - новые API endpoints `GET/PUT /api/runtime/timeouts`
  - runtime profile patch service покрывается через API characterization: validation, merge/prune, manifest rewrite/reopen and unchanged error-code surface
- runtime sharding control:
  - heuristics planner (module markers + leaf-pruning) и `analysis.include/exclude` фильтры
  - structural coalescing для больших repos сохраняет module marker leaf shard groups внутри top-level dirs, если итоговый shard count остаётся в `maxAutoShardsPerRepo`, и детерминированно merge-ит excess top-level groups в bounded buckets
  - root-marker-only repos планируются как root-file group + top-level directory shards, а не single `"."` shard, если структура repo большая
  - fallback warning + root shard `.` при пустом результате фильтров
  - scheduler semantics `sequential|parallel` (`max_parallel_tasks`) и deterministic apply order
  - `fail_fast` останавливает step/pipeline на первой shard error без перехода в downstream runtime steps
- runtime provider model control:
  - manifest/schema validation covers provider keys, model bounds, provider-specific effort values,
    and rejects unsupported Qwen effort overrides;
  - resolver tests cover provider-default fallback and independent env-over-workspace precedence;
  - adapter tests assert omitted values preserve native defaults while explicit values are forwarded;
  - API tests cover `GET/PUT /api/runtime/models`, replacement/prune semantics, capabilities, and
    effective/source readback; accepted run/history snapshots retain resolved settings.
- `best_effort` partial shard failures: pipeline completes terminal reporting with status `failed` + `error_code=run_partial_failed`; partial collect skips live downstream `step2/3/4` runtime and records `*_skipped_due_to_partial_collect` so collect remains an explicit execution blocker/counter.
- sequential `best_effort` collect aborts after five consecutive `runner_unavailable` shard failures, preserves the provider error class on undispatched shards, and keeps the root cause visible in reports instead of spending the full medium window on repeated provider outages.
- batch/report tests фиксируют, что `partial_failure_count>0` / `run_partial_failed` остаётся видимым `runtime_flow_failed` signal в `run_matrix`/`execution_report`, но primary `failure_class` выбирается из конкретного terminal/provider classifier (`runner_unavailable` или `runtime_contract_failed`), если он есть; это разделяет primary terminal failure и secondary partial shard evidence.
- frontend report tests фиксируют, что `frontend_e2e_matrix_<batch-id>.md` для `runtime_run_failed` содержит runtime details (`run_id`, `last_run_status`, `error_code`, `current_step`, screenshot count/result directory) без изменения summary status taxonomy.
- docflow builder seam:
  - staged artifacts, citation index, final run index and semantic snapshot remain characterization-covered before promotion
  - promotion builds a complete run-scoped generation, validates indexed files, activates managed canonical roots with journaled rollback, and regression tests inject failures across copy/model/diagram/activation operations to prove canonical state is either the previous complete generation or the new complete generation
  - `reports/changelog/*` draft files are activated through the journaled file path while preserving existing changelog history; stale managed artifact registry entries are removed only after successful activation
- UI route-shell seams:
  - feature-owned Changes/Knowledge/Publish view models receive authoritative route/data inputs;
  - W23B1 route codec tests cover Task Inbox/New/Detail/Attempt deep links, optional Task context on
    Architecture/Changes and fail-closed unsafe identities; the target container is tested to avoid
    legacy/latest-run fallback;
  - W23C component tests cover displayed scope submission, exact created Task identity routing and
    fail-closed runner mode/provider readiness;
  - W23D component/route tests cover public Task/Attempt loading, five derived Inbox groups,
    URL-restorable lifecycle/runner/repository/time filters, keyboard-safe row activation, exact
    detail/history identities and explicit empty/error recovery without legacy/latest-run fallback;
  - W23E outcome fixtures cover succeeded/failed/canceled Task detail states, exact Attempt→run
    review-summary binding, semantic delta counts, independent current Architecture availability and
    explicit missing-comparison partial state without fabricated zero deltas;
  - W23F fixtures cover exact Attempt-bound Pipeline Studio deep links, canonical step rendering,
    bounded blocker/diagnostics disclosure and the absence of provider-output-derived percentages or
    latest/global run fallback;
  - W23G fixtures cover the exact opaque Task identity in the Architecture context, current promoted
    authority wording, read-only/no-latest-run guardrail and return navigation without starting a
    second analysis;
  - W23H fixtures cover lossless Markdown draft editing, allowlisted `charter/*` write admission,
    explicit save status and read-only promoted report behavior;
  - W23I fixtures cover canonical entity identity, schema/version labels, path-linked model
    validation status, line-numbered Advanced source loading and the explicit no-structured-save
    guard until a lossless YAML/JSON round-trip proof exists;
    authority matrix coverage separates `promoted_current`, `run_snapshot`, `qa_snapshot` and
    `qa_audit`, rejects selected-QA fallback and prevents `reports/taskruns/**` from entering
    Knowledge; ProductShell, semantic primitives and ContextDrawer have focused component coverage
  - Evidence Viewer regressions cover relative links, traversal/cross-run rejection, safe raw HTML,
    explicit diff identity, unknown provenance, typed partial state, long lines and oversized
    read/render fallback
  - responsive/a11y coverage includes 1440, 1280, 1024 and 390×844 rendered scenarios, global
    overflow/console checks, critical axe, safe-area CSS, 44 px controls, keyed card tables and
    focus trap/Escape/outside-click/return-focus/orientation transitions
  - historical snapshot tests inject a foreign-run final index before the selected-run index and
    require the UI to open only the exact selected-run staged paths; the live ProductShell flow
    accepts an empty optional Diagrams group but must still open and inspect substantive indexed
    Reports from the selected snapshot
  - artifact-quality tests distinguish standalone/scaffold placeholder markers from substantive
    safe-change guidance that names an observed repository value as a placeholder
- docs truth-sync gate проверяет:
  - согласованность runtime policy/Q&A boundary и ссылок на canonical stakeholder matrix;
  - prompt-layer truth: exact merge order (`provider header -> artifact-only/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer`) и invariant `workspace prompt pack = editable content layer only`;
  - active-only `docs/PLANS.md` не возвращает уже закрытые cleanup/refactor планы в current ExecPlan surface;
  - отсутствие stale-маркеров в ключевых surfaces (`future`, `skeleton`, `placeholder`, устаревшие version-маркеры);
  - CLI docs parity: базовые `acp serve|run|qa` usage и runtime flags в help и документации совпадают

## 5) Обязательные internal test seams

- fake runner + artifact fixtures вместо live headless providers в required tests
- injectable clock/run-id provider для deterministic golden outputs
- injectable git executor/repo resolver для local test doubles
- workspace sandbox root для integration tests без записи вне test workspace
- internal runtime/orchestration seams:
  - `internal/runtimeprofile` keeps runtime profile patch validation/merge/manifest rewrite shared below API adapters
  - `RuntimeTaskExecutor` keeps task envelope/timeout/heartbeat/provider execution behavior characterization-covered without coupling it to sharding planner tests
  - `run_finalization.go`, `step_handlers.go` and `artifact_registry.go` keep terminal status, step dispatch and artifact list behavior in narrow files while existing async/docflow/sharding tests preserve external run contracts
  - `sharding_coordinator.go`, `sharding_scheduler.go`, `sharding_summary_store.go`, `sharding_artifacts.go` and `sharding_planner.go` keep planning, scheduling, summary/checkpoint persistence, artifact materialization and apply/replay coordination in separate files while preserving the existing sharding characterization tests
  - `ShardSummaryStore` keeps persisted shard-summary/checkpoint behavior covered separately from scheduler ordering and apply/replay coordinator behavior
  - `artifactquality` remains canonical wording source for collect/validator prompt snippets reused by runtime prompt contracts and baseline prompt packs
- UI hook facades stay stable while internal hooks isolate run selection/polling/actions and workspace manifest/baseline/wizard/git actions; App tests preserve route-shell behavior and stable `data-testid` surfaces

## 6) Required CI jobs

Toolchain policy:
- Go module compatibility remains `go 1.20`, but required CI, release builds, and Makefile entrypoints use the exact Go version from `.go-version` to avoid shipping binaries built with an unsupported/vulnerable standard library.
- UI/source-build jobs require exact Node.js version from `.node-version`.

Implemented required jobs:
- `contracts`
  - `make contracts`
  - locked validator toolchain from `tools/contracts/package-lock.json`; required CI must not
    resolve mutable `latest` packages during validation
  - schema validation
  - parse examples/fixtures
- `backend`
  - `go test ./...`
  - `./scripts/run-python.sh -m unittest discover -s scripts/tests -p '*_test.py'`
  - includes docs-consistency gate (`internal/docsync`) для truth-sync/stale-marker/CLI-docs parity checks
  - includes harness regression fixtures for batch failure classification (`scripts/tests/*`)
  - `make test-stress` (coordinator explicit-queue and pending-supersession regression loop)
  - `go build ./cmd/acp`
- `ui`
  - `./scripts/run-npm.sh ci --prefix ui`
  - `./scripts/run-npm.sh run typecheck --prefix ui`
  - `./scripts/run-npm.sh run test --prefix ui -- --run`
  - `./scripts/run-npm.sh run build --prefix ui`
  - `make verify-ui-determinism` builds the exact checked-out commit in two independent
    temp roots and compares sorted `ui/dist` path/digest manifests
  - `make verify-ui-dist` rebuilds and re-embeds `internal/api/ui_dist`, then fails if the
    tracked embedded bundle is stale

Implemented additional jobs:
- `lint`
  - installs UI dependencies, then runs canonical `make lint`
  - covers Go formatting, ShellCheck for production shell scripts and UI typecheck in one
    local/CI-equivalent entrypoint
- `golden`
  - `TestScenarioFixturesDeterministicInitPipeline`
  - `TestScenarioFixtureLayoutExists`
  - `TestScenarioRunnerFixturesContractAndSemantics`
  - `TestScenarioDomainTaskEnvelopesDeterministic`
  - `TestDeterministicSnapshotScopeExcludesRunSpecificArtifacts`
- `smoke-cli`
  - `acp run --workspace ... --pipeline init --runtime fake --non-interactive`
  - `acp run --workspace ... --pipeline refresh --runtime fake --non-interactive`
  - deterministic fake runner only
- `smoke-api`
  - `acp serve --workspace ... --runtime fake`
  - `/api/workspace/validate`
  - pipeline status/artifacts endpoints
  - run logs endpoint with explicit `cursor`/`limit`, second-page pagination and invalid cursor
    response validation
  - dynamic free port + explicit fail on run polling timeout
- `ui`
  - runs UI tests, production build, deterministic build verification and embedded bundle
    freshness; UI typecheck is owned by canonical `make lint` in the `lint` workflow
  - installs Chromium and runs `npm run e2e:mock --prefix ui`, which executes eight local
    provider-free Playwright scenarios and fails on skipped scenarios, console errors or critical
    horizontal overflow
  - optional local coverage is available through `npm run coverage --prefix ui`; it uses locked
    `@vitest/coverage-v8`, includes all `ui/src` implementation files and writes ignored
    `ui/coverage/coverage-summary.json` / `coverage-final.json`

Security/advisory workflows:
- `dependency-review` runs on pull requests and blocks newly introduced vulnerable dependencies.
- `codeql` runs Go and JavaScript/TypeScript analysis on pull requests, pushes to `main`, and weekly schedule.
- `scorecard` runs OpenSSF Scorecard on push/schedule with top-level read-only workflow permissions; the scorecard job alone gets `id-token: write` and `security-events: write` for result publishing/SARIF upload, and the action is pinned to the upstream tag's peeled commit so Scorecard publish verification can resolve the action owner correctly. Full Scorecard publish verification is confirmed on default-branch push/schedule; PR branches rely on ordinary required checks.

Release workflow hardening:
- tag-only release workflow uses job-level write permissions, an explicit `github-release` environment, pinned actions, timeouts, and provenance/SBOM artifact generation.
- tag release publication is split behind a read-only `verify-release-evidence` job that runs
  `scripts/verify-release-verdict.py` against exactly one configured mode:
  `ACP_RELEASE_MATRIX_IDS` for composite release evidence, or the compatible single-matrix
  `ACP_RELEASE_VERDICT_PATH` / `ACP_RELEASE_MATRIX_ID`; the write-enabled GoReleaser/provenance job has
  `needs: verify-release-evidence`.
- GitHub environment required reviewers, protected tags, branch protection, Dependabot alerts/security updates, secret scanning, and push protection are repository settings and must be enforced by owners/admins.
## 7) Базовый набор тестов

### Contract tests
- valid `workspace.yaml`
- invalid `workspace.yaml`
- valid docs-first contracts (`shard-pack-manifest`, `final-run-index`, `citation-index`, `validator-verdict`)
- negative docs-first contract cases (missing citations, duplicate claim/topic ids, broken topic refs)
- docflow index assembly with repeated provider-authored document ids across distinct canonical paths; final/citation indexes must remap them to globally unique canonical document ids and keep citation/document links reciprocal after remap
- valid persisted runtime execution metadata
- invalid runtime execution metadata
- invalid artifact contracts (`shard-pack-manifest`, `validator-verdict`, draft manifests)
- UI Review regression covers two selected runs with identical canonical artifact paths but
  different staged bytes; historical Review preview, coverage and questions must read the
  selected run's `reports/taskruns/<run_id>/staging/final/...` bytes rather than current
  canonical workspace files.
- runtime draft manifest metadata: optional `updated_at` is accepted, while legacy/envelope fields such as `repo_scopes`, `compatibility`, `generated_at`, `pipeline`, output aliases such as `logical_path`, or `proposals[]` remain invalid
- strict collect validation:
  - artifact-root-prefixed, absolute, missing-file, directory and hidden provider/tool `documents[].path` (`.qwen/`, `.claude/`, `.codex/`, `.git/`, `node_modules/`) fail-ятся без rewrite
  - `fixtures/scenarios/collect-manifest-wrong-artifact-root` проверяет identity drift без live-specific данных: structurally valid manifest с существующим authored document, но foreign `artifact_root`, fail-ится до downstream use. `fixtures/scenarios/collect-manifest-task-identity-typo` фиксирует live-observed односивольный `shard_id` drift: providercommon может детерминированно восстановить только assigned top-level task identity после успешной task-independent validation, затем обязан пройти полный task-aware contract; fixture с дополнительной broken evidence path доказывает отсутствие мутации. Ordinary provider-authored manifest repair остаётся fallback, а ACP не нормализует nested identity/content и не считает recovery artifact-quality acceptance.
  - missing, guessed or directory-only repo evidence paths in `citations[].path` and `semantic.*[].provenance.evidence[].path` fail when resolved repo roots are available, including generated repo root suffix aliases
  - referenced authored collect docs с `ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT`, unchanged marker-free seed prose, recovery fallback prose или failure-only prose (`no repository evidence was emitted` / bounded collection failure) fail-ятся до apply и не маскируются pair-repair/runtime-recovery success
  - collect artifact monitor snapshot использует тот же strict repo-root-aware validation и не считает parse-only manifest, bootstrap markdown, missing evidence path или directory-only evidence path валидным controlled-stop сигналом
  - deterministic fake runtime выбирает реально существующий repo evidence file из `read_context_roots`/`path_scopes` (например `README.adoc`), а не hard-coded `README.md`, чтобы live baseline API simulation не блокировала git_url profiles до headless provider run
  - missing required metadata fail-ится без autofill
  - normal collect prompt требует первый bounded work unit: короткий read/list по allowed evidence, concrete per-path-scope file candidates, затем direct literal heredoc/printf/tee write suggested authored doc + `shard-pack-manifest.json`; second read-only preflight, analysis-only prose, broad sweep, Ruby/Node/Python/Perl/awk/jq inline writer, generated template programs, nested quote tricks и reliance on focused repair как штатный success path должны ловиться prompt contract tests
  - collect pair recovery запускается один раз при no authored artifacts + non-empty provider diagnostics или exhausted fully silent collect fresh retry, разрешает писать только suggested authored doc + `shard-pack-manifest.json`; runtime policy requires fresh mutation and default 5-minute pre-artifact/post-artifact/partial-artifact windows for collect pair repair. Если focused collect-pair repair сам stalls pre-artifact with empty stdout/stderr and no fresh authored mutation, policy-enabled providers get one bounded focused retry; second silent no-fresh exhaustion is `runner_unavailable`, while a fresh but invalid mutation remains `runtime_contract_failed`. Qwen-specific stream-only pre-artifact exhaustion без authored files также получает ровно одну retry с compact stall-focused prompt; повторный stream stall остаётся `runtime_contract_failed`. Prompt является write-first final repair без heredoc seed/fallback, запрещает separate read-only `collect_pair_repair_preflight`, печать capped evidence packet как единственное действие, analysis-only prose до записи artifacts, provider-invented exact phrase gates (`required = [...]`, `missing expected evidence`), oversized-evidence aborts (`read file exceeds size limit` при доступных других snippets), semantic pre-write aborts и fragile f-string/`.format(...)` template writers перед записью. One-shot bounded filesystem command читает listed evidence prefixes, truncates/skips oversized candidates, and writes markdown + manifest before returning; planned claims без observed snippets удаляются или становятся coverage gaps, semantic sufficiency проверяет backend validation, а `Recovery Summary`/`Recovery Bootstrap`/`Recovery Evidence Summary` scaffold reject-ится. Pair repair prompt tests also require validation-specific focus for directory-only repo evidence refs, process-contaminated markdown, and no-artifact stalls; rejection guidance for runtime-process gap wording such as `not examined in this bounded pass`, canonical semantic shape for `coverage.notes[]`, `entities[].id/name/type/provenance`, `edges[].id/type/from/to/provenance`, `findings[].id/severity/title/description/provenance`, `provenance.kind/confidence/evidence[]`, legacy top-level `claims`/`claim_map`/metadata wrappers, legacy semantic aliases (`relation/source/target`, entity-level `repo/path/evidence`, direct repo/path provenance, `evidence_citation_ids`, finding `summary`) and unreplaced claim-id template tokens such as `SHARD`, `<shard>`, `<claim>`, `TODO`, or `REPLACE_ME`. Candidate tests require directory-scope fairness so later path scopes still contribute real files.
  - Qwen stream-only collect-pair retry carries an explicit `collect_pair_repair_stream_retry` marker and selects a separate tool-call-first prompt: `run_shell_command` must be the first response block, evidence is capped at four files/4000 bytes, bulky canonical-shape/checklist/skeleton sections are absent, and the whole prompt remains at most 4 KiB. Existing validation/write-set gates remain authoritative and a second stream stall is still `runtime_contract_failed`.
  - collect pair and manifest repair prompt tests require exact stable `documents[].path -> documents[].canonical_path` mapping for provider-authored markdown. Compact no-artifact repair must name the exact `documents[0].canonical_path` and reject `reports/taskruns/**`, `/staging/`, absolute `write_root`, duplicated `artifact_root`, raw runtime metadata/logs and provider side-effect paths as canonical publish surfaces.
  - manifest-only runtime repair запускается один раз только при authored docs + structural-invalid `shard-pack-manifest.json`; if structural-invalid cause is missing repo evidence and authored markdown still names the missing path, or if authored markdown is empty/whitespace-only, runtime uses collect pair repair targeting that existing markdown and requiring a fresh rewrite before manifest-only repair can run; missing/empty manifests, scaffold-only manifests, and clean manifests whose only binding error is empty/missing `citations[].claim_ids` can use deterministic `collect_manifest_runtime_recovery` from temp authored-doc trees. Provider repair prompt still requires `FIRST COLLECT MANIFEST REPAIR COMMAND` as next filesystem action, and that first provider-authored command must read bounded authored docs/evidence and write `shard-pack-manifest.json` before returning; evidence-packet-only output or status prose without a manifest write is a no-op repair failure. Exhausted/no-op structural manifest repair remains terminal `runtime_contract_failed` and must not fall back to deterministic runtime recovery. Recovered runtime manifests carry explicit warning/diagnostic markers and evidence-rich docs must produce concrete semantic entities/edges/findings beyond repo/shard wrappers and owner mapping; prompt contract tests require manifest-only repair to carry the same canonical semantic-shape block and to forbid top-level `claims`/`claim_map`/metadata wrappers, legacy semantic aliases and placeholder or empty claim-id tokens.
- draft enrichment include-scope tests require current-run shard status JSON visibility for `step2/step4` while keeping whole workspace/source repo and old sibling taskrun histories out of the focused read surface. Prompt contract tests require target identity from `repo_scope`/`repo_scopes`/`domain_id`, typed shard-summary `items[].status` counting, current-run final-index counts from `canonical_documents[]`, citation counts from `citations[]`, exact final/citation index count consistency when index files are present, no matrix-folder target contamination, no placeholder/recovery wording in proposals, runtime draft validation rejection for generic placeholder-replacement narration such as `replaced placeholder proposal content`, rejection of step0 downstream/runtime-only constitution leaks, rejection of foreign live `run_*` taskrun IDs in current-run markdown, rejection of generic conditional shard-gap wording such as `failed shards require rerun`, acceptance of exact current-run no-shard-coverage-blocker statements when tied to the typed summary/status, rejection of step2 `summary.md` that omits exact typed shard-summary completeness, rejection of step2 `overview.md` without concrete repo/path, citation, or staged artifact evidence refs when typed shard status is visible, rejection of step2 `architect-summary.md` without decision-ready operator next-action/inspection cues when typed shard status is visible, rejection of stale current-run final/citation-index unavailable or not-yet-present claims in `step2` markdown, rejection of false empty-shard evidence claims such as `Shard pack manifests: none observed`, rejection of stale `No current-run final-run-index document list was available`, `final-run-index.json contains 0 observed document entries` and `final-run-index.json (0 observed document entries)` claims in `step4` proposal/changelog markdown, rejection of empty proposal/changelog sections, dangling `findings above` references and generic follow-up plans without a proposed-plan no-actionable-proposal gap, rejection of metadata-only evidence bullets such as `"version": 1` and unquoted shard-summary key dumps, rejection of raw structured evidence dumps (`{'id': ...}`, `documents=[{...}]`, `citations=[{...}]`) and malformed markdown backtick/fence syntax in operator-facing markdown, provider-authored markdown syntax retry prompt without scaffold heredoc after a malformed enrichment attempt, command-text retry prompt/engine behavior, terminal noop/scaffold enrichment exhaustion, and no false `0 authored markdown shard docs` / `0 staging shard files` claim.
  - live-shaped runtime-draft section tests accept substantive H3 Sprint phases under the required
    H2 proposal plan, reject a heading-only nested phase, and prove that actionable content in the
    next sibling H2 cannot satisfy an empty required H2
  - bidirectional boundary tests scan Go/Python/TypeScript ownership: live/manual release identity and
    assessment names cannot enter product runtime/orchestrator/UI sources; the live flow cannot
    import production-internal selectors/validators or synthesize `run-history.json`; provider env
    fixtures prove ambient orchestration identity is removed before child process start
  - backend-cycle script tests assert that failed `/api/pipeline/init` baseline simulation writes a 17-field fake init row into `run-results.tsv` before stopping, preserving failed run telemetry for `execution_report_*`
  - successful Epic 21 no-op refresh is counted as a completed run without legacy `<run_id>-quality.json` only when its matching run-scoped `refresh-execution.json` validates `unchanged_candidate`, `no_op` and skipped providers; malformed, mismatched or absent audit evidence remains a gate failure
  - manifest-only runtime repair fail-ится, если provider пишет что-либо кроме `shard-pack-manifest.json`
  - repair include dirs исключают broader workspace `reports/taskruns`/sibling manifests и оставляют только current write root + repo evidence roots
- strict validator normal prompt и repair prompt используют command-first absolute heredoc skeleton для `validator-verdict.json`; repair запускается максимум один раз, пишет только `validator-verdict.json`, указывает `checked_paths` на staged final artifacts, требует canonical `issues[]` shape, reject-ит legacy issue fields и закрывает `questions[]` до `id/text/priority/related_ids` без `citation_ids`
- strict draft repair запускается максимум один раз, пишет только step manifest в `write_root` и draft files под `draft_final_root`; draft artifact monitor учитывает nested draft files inside `draft_final_root`
- strict draft validation fail-ится, если referenced `outputs[].path` отсутствует, даже когда файл существует только по `outputs[].canonical_path`
- active compatibility inventory отсутствует; tests не должны ожидать compatibility rule ids
- validator repair stage проверяется отдельно на atomicity: при write failure staged state не мутируется
- UI ownership split держится unit/integration coverage-ом поверх route shell `App.tsx`, `useWorkspaceSetup`, `useRunExplorer`, `useRunLogs`, `useRunArtifacts`

### Semantic tests
- duplicate repo names
- unsupported manifest fields
- `observation` without evidence
- unknown `owner_team_id`
- canonical top-level coverage/questions dedupe

### Golden tests
- stage-then-promote deterministic flow for canonical docs-first surfaces
- derived `model/*` extraction determinism
- stable slug normalization and collision handling
- Step 4 changelog determinism

### Scenario integration tests
- one-service happy path
- multi-repo dependency extraction
- missing owner / missing CI-CD evidence path
- unresolved domain/team becomes question/finding, not new card
- deterministic Step 1 enrichment включает `evidence_refs` в domain/team cards
- sharded runtime regression:
  - step1/step3 materialize runtime-execution metadata + shard-plan/shard-summary artifacts
  - shard-summary statuses cover `pending/checkpointed/succeeded/failed` and survive restart recovery
  - parallel scheduler keeps deterministic merge/apply order despite out-of-order shard completion
  - runtime execution metadata (`shard_id`, `repo_scopes`, `path_scopes`) сохраняется в persisted `runtime-execution.json`
  - service restart recovery resumes same `run_id` from persisted shard artifacts without rerunning already persisted runtime executions

### Smoke tests
- `acp run --workspace ... --pipeline init --runtime fake --non-interactive`
- `acp run --workspace ... --pipeline refresh --runtime fake --non-interactive`
- `acp serve --workspace ... --runtime fake`
- `/api/workspace/validate` без request body
- pipeline endpoints не принимают `workspace_path`
- run logs endpoint:
  - `GET /api/pipeline/runs/<run_id>/logs?cursor=<n>&limit=<n>`
  - required API smoke validates first page, second page from `next_cursor`, payload shape,
    non-2xx/malformed failure handling and `400 invalid_cursor`
  - deeper Go/API coverage keeps invalid params + run_not_found cases
  - structured failure diagnostics в `fields` (`stdout_snippet`/`stderr_snippet`, `task_id`, `provider`, counters)
  - mixed wire-shape (`kind=event|runtime_output`, optional `stream=stdout|stderr`)
- run cancel endpoint:
  - `POST /api/pipeline/runs/<run_id>/cancel`
  - happy-path `202`, `404 run_not_found`, `409 run_not_cancelable`, `400 invalid_request_body`
- UI path: open Guided Setup, inspect Runs, select completed run evidence in Changes and open Publish through destination/view controls. In matrix live smoke this uses the backend refresh snapshot; direct `UI_E2E_ARTIFACT_SOURCE=live` diagnostics may still start a fresh UI init.
- Required mock Playwright gate: `npm run e2e:mock --prefix ui` starts a local Vite UI through
  `ui/playwright.mock.config.ts` and runs exactly eight deterministic scenarios:
  initial-analysis-to-refresh happy path, source recovery, onboarding recovery, permission recovery,
  provider stream, failed-shard analysis, Publish Git recovery and QA recovery. These scenarios use mocked `/api/**` responses only; live
  providers, external repositories and network runtime checks stay out of required CI.
- UI run diagnostics surface:
  - Runs Step review tabs for artifacts/logs/evidence/diff
  - log polling/append without duplicates and selected-step filtering
  - local non-modal Diagnostics disclosure for shards, raw runtime output, permissions and telemetry
  - runtime execution artifact refs remain secondary to workflow acceptance
- UI results diagrams surface:
  - navigation through `Review`
  - diagram artifact listing and Mermaid preview render
  - deterministic UI coverage for failed latest run + previous successful run recovery in Review, so
    operators can reach the complete artifact set after a failed refresh
  - live `init-inspect` reads selected diagram content through `/api/artifacts` and fails gap-only C4 output instead of accepting a rendered placeholder diagram
- UI version/readability surface:
  - deterministic UI/API tests cover `GET /api/system/version` before workspace selection and top-bar display of actual build metadata
  - live `init-inspect` rejects hard-coded public release labels and placeholder/scaffold markdown previews in Review
- UI domain-map surface:
  - deterministic UI/fake-fixture tests cover artifact-derived entity/edge rendering, sparse-model partial states, edge navigation and proposal/evidence drilldown
  - provider-live domain-map execution is not part of release readiness until a separate owner-approved live-gate slice defines stable semantics
- UI Ask UX smoke:
  - live frontend shell defaults to `UI_E2E_QA_SMOKE=1`; it checks run history, read-only safety, answer panel, citations panel, confidence/unresolved signals and context-pack/runtime-execution links
  - async Ask polling uses a bounded `ACP_UI_QA_POLL_TIMEOUT_SEC` budget (default 300s); timeout emits `ACTIVE_RUN_TIMEOUT`, is classified as `active_run_timeout`, and the shell best-effort cancels the active QA run before tearing down the server so provider children are not orphaned
  - explicit `UI_E2E_QA_SMOKE=0` is allowed only for diagnostic speed runs and must be recorded as UX residual risk, not accepted Ask evidence
  - screenshot refs are evidence-only and do not influence machine execution verdicts
  - release UX readiness still requires Ask-flow evidence in `swe_ux_assessment_<matrix-id>.md`
- UI Publish gate coverage:
  - deterministic UI tests check folder summary, selected artifact preview, explicit diff partial state, publish gate/checklist, commit plan and existing Git actions
  - Changes route table covers distinct overview/evidence/findings/diff/proposals/publish models,
    popstate restoration and server-authored Git `clean|dirty|stale|blocked|unknown`; mutation
    confirmation is unavailable for non-actionable Git truth
  - URL/request identity tests cover delayed artifact selection, run/source/path/viewer generations,
    invalid explicit enum cleanup through notice + `replaceState`, and user `pushState` navigation
- UI run lifecycle operability:
  - bootstrap auto-select newest active run
  - если выбранный run исчезает из list endpoint и replacement доступен, UI переключается на следующий run; если list endpoint временно пуст, но status endpoint ещё жив, selection сохраняется
  - `Run status` показывает полный warnings list выбранного run
  - `Cancel selected run` корректно обрабатывает `202/404/409`
- UI runtime settings surface:
  - save/reset `Runtime Timeouts`
  - save/reset `Runtime Execution`
- UI quick actions:
  - collapsed runtime execution artifact action открывает persisted taskrun artifact без live e2e-only допущений
- Подробный command cookbook по trusted-machine live/release gate intentionally вынесен в `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.

### Optional live-runner smoke
- Local `manual-live-e2e workflow` is the trusted-machine operator procedure from `docs/RELEASE_LIVE_E2E_RUNBOOK.md`, not a GitHub Actions workflow.
- Scripts produce machine execution evidence and verifier-backed execution verdicts only. Запускающий SWE-agent produces separate UX and artifact-quality assessments over public evidence; harness no longer generates `blackbox_e2e_steps_*`.
- `scripts/live-e2e-plan.py` — catalog-driven command generator for direct matrix harness invocations:
  - does not execute the harness and does not replace `scripts/full-run-batch-matrix.sh`
  - supports flexible selectors `smoke tiny`, `regres fast|long|full|complex`, `release fast|long|full`
  - `smoke tiny` is `1 repo × 1 run × 1 provider` for fastest trusted-machine signal
  - `regres complex` is diagnostic-only product/feature rotation over Temporal, Backstage, Airflow, Appwrite and Saleor; it is not a release readiness signal
  - generated `regres`/`release` commands rely on execution reporting: `execution_report_<batch-id>.md`; `reports/taskruns/<run_id>-quality.json.quality_signals[]` remains the public black-box evidence channel, and any `artifact_quality.*` signal fails strict batch/matrix/release artifact-quality status while keeping runtime contract status separate
- `scripts/full-run-batch.sh` — canonical live batch + frontend live e2e:
  - canonical input: `TARGET_REPOS_FILE`
  - direct-only runtime commands: `claude`, `qwen`, `codex`
  - for canonical `path` inputs, child backend cycle verifies the original checkout and writes a generated repos file that points to run-local isolated detached clones; script tests cover accepted isolated clones, corrupt/non-git checkout blockers, and restoring write bits before temp cleanup
  - selected-provider readiness записывается в `preflight.json`; version + provider-specific bounded headless probe where stable + artifact smoke ловят missing binary, auth/quota, codex CLI compatibility и no-write host/provider failures до deep run. Qwen не использует text-only `ACP_READY`: после `qwen --version` одна runtime-like artifact-smoke попытка с canonical `120s` budget обязана завершиться с exact sentinel; timeout, non-zero exit, missing/invalid sentinel остаются `operational_host_preflight_failed`, а process-group cleanup не оставляет provider children. Live E2E Codex readiness/runtime invocation по умолчанию использует `gpt-5.6-luna` with `model_reasoning_effort="high"`, while qwen/claude stay on CLI defaults; для `claude` artifact smoke является основным headless readiness gate, allowlist-ит temp write dir через `--add-dir` и получает один bounded retry на timeout/no-output; provider `model`/`modelUsage` telemetry не является blocker
  - DoD/UI precheck failures generate execution evidence before runtime starts. `make contracts test lint build` is bounded by `ACP_LIVE_PRECHECK_DOD_TIMEOUT_SEC`; `npm ci --prefix ui` and `playwright install chromium` are bounded by `ACP_LIVE_PRECHECK_UI_TIMEOUT_SEC`. Script tests cover hung DoD/npm commands producing `precheck_failed` plus `[precheck-timeout]` log evidence, not a silent running batch.
  - baseline API init simulation uses `api_init_timeout_sec` as the base budget with bounded progress grace when status/current-step/artifact/warning counters advance; script assertions keep this progress-aware poll path visible without increasing canonical timeout presets.
  - each backend `acp run` is bounded by `pipeline_timeout_sec` through `scripts/pipeline-watchdog.py`: the watchdog starts the command in its own process group, kills the full group on deadline, writes `runtime_timeout` terminal status, records deadline/last-progress fields, separates `last_output_activity_at` from useful `last_progress_at`, and marks host sleep/clock jumps as contributing infra evidence instead of changing the primary class.
  - backend execution source-of-truth: только `snapshots/<run_id>/reports/*`; `reports/taskruns/<run_id>-quality.json.artifact_inventory` должен каждый раз заново описывать фактические promoted/taskrun surfaces текущего run, а не ссылаться на сохранённую bad-run fixture
  - failed headless `init`/`refresh` rows must be appended to `run-results.tsv` even when `acp run` exits before printing `run_id`; helper falls back to newest workspace `run_*-quality.json` / raw runtime metadata and reports missing-row runtime logs as evidence, not as the primary root cause
  - `artifact_quality.*` signals emitted by product quality summaries block strict live E2E artifact-quality status; reports must show split cases as `runtime passed, artifact quality failed` instead of collapsing them into runtime/provider failures
  - placeholder promoted-artifact checks distinguish bootstrap/generic proposal text from evidence-backed proposal/changelog artifacts: boilerplate validation notes alone are not blockers when the artifact carries concrete finding/question ids and findings/coverage traceability
  - artifact-quality report tests cover misleading/off-topic interpretation, weak evidence density, useless C4/Mermaid and missing cross-repo truthfulness; machine hard-fails are limited to public product signals and documented analysis blockers, while broader SWE review remains a companion assessment
  - frontend smoke работает на отдельной `frontend-workspace` копии backend workspace, merge-ит выбранный run snapshot reports и captured product-authored history поверх copied reports tree и не мутирует backend baseline; harness не создает и не переписывает product run history, а Playwright выбирает completed snapshot run instead of starting a second provider-backed init; `snapshot_reports_missing` после terminal backend failure записывается как dependent skipped frontend status, а не independent frontend regression; shell tests закрепляют, что failed headless refresh row remains ineligible for frontend smoke even if a triage snapshot directory exists
  - terminal-success backend runs (`result=passed`, `run-status.env state=completed process_exit=0`) остаются `failure_class=none`, даже если raw provider logs содержат recovered `runner_unavailable`/429 diagnostics
  - batch/report tests cover `runtime_contract_status`, `artifact_quality_status`, `quality_gates_failed_failures` and `artifact_quality_failed_failures`; `runtime_quality.stall_pressure` stays non-blocking but caps the display label below `Excellent`; step-level blocker fixtures preserve both `first_validation_error_excerpt` and additive `terminal_validation_error_excerpt` so final repair exhaustion causes are visible without raw-log drilldown
  - execution report/matrix telemetry counters агрегируют `repair_attempts`, `repair_exhausted`, `fresh_retries`, `focused_repairs`, `stall_count`, `pre_artifact_stalls`, `post_artifact_stalls`, `zero_output_pre_artifact_stalls`, `partial_failure_count`, `provider_invocations`, `provider_invocation_budget_max`, `provider_invocation_remaining`, `provider_budget_exhausted`, `provider_last_transition`, `provider_terminal_exhaustion_reason`, `validation_first_pass_valid`, `validation_first_pass_invalid`, `validation_issue_classes`, `effective_verdict_source`, `promotion_audit_result` и `quality_alerts`; counters читаются из snapshot quality JSON, а для failed/non-snapshot runs — из actual workspace `reports/taskruns/<run_id>-quality.json` без превращения такого run в snapshot hard-pass; non-exhausted repair/stall pressure visible but non-blocking, partial failures remain blockers; provider-free event-sequence tests prove that a valid provider-command controlled stop and its paired `retry scheduled / terminate_and_validate` diagnostic count once as `valid_artifact_controlled_stops` and never emit `stall_pressure`, while invalid/missing artifact state remains an actual post-artifact stall; malformed provider-authored semantic artifacts such as markdown written to `shard-pack-manifest.json` are surfaced as `analysis:malformed-semantic-json` issue details instead of crashing report generation; runtime Go tests cover the single provider-authored `collect_manifest_shape_cleanup` retry for missing question text / duplicate citation IDs and prove repeated invalid cleanup remains `runtime_contract_failed`; `internal/conformance` incident tables cover foreign identity, schema drift, missing evidence, invalid ranges/hashes, graph collisions, contradictory verdicts, stale provider repairs and audit failures with zero false accepts; provider-free parity tests prove Claude/Qwen/Codex envelopes produce the same ordered issue codes and cannot exceed three provider starts per runtime unit, while the deterministic conformance trace records p95 starts at two.
  - provider lifecycle metadata tests assert that timeout/stall diagnostics include `last_pipe_activity_at`, artifact mutation/state fields and `no_progress_duration_ms`; watchdog tests assert that heartbeat stdout, `reports/taskruns/**/logs`, `reports/taskruns/**/raw` and `run-history.json` are diagnostic activity only, while staged artifact files are useful progress.
  - stall termination drains provider pipes before forcibly closing local readers, and stream activity is recorded only after the same bytes enter the captured result. Repeated stress coverage protects stream-only collect-pair retry from a terminate/capture race.
  - provider activity timeout overrides (`ACP_PROVIDER_*_ARTIFACT_STALL_SEC`, `ACP_PROVIDER_VALID_ARTIFACT_STOP_SEC`) are covered by focused unit tests and counted as diagnostic timeout overrides by the matrix release guard; release-mode tests must keep them blocked.
  - batch report evidence tests проверяют, что `collect_partial_shard_failures`, focused recovery exhaustion/write-set violations и missing headless rows with runtime logs surfaced as per-run issue details, а не теряются за aggregate failure class
- `scripts/full-run-batch-matrix.sh` — официальный local trusted-machine harness:
  - canonical input: `E2E_MATRIX_FILE`
  - approved profile ids: `single-path`, `single-git_url`, `multi-path`, `multi-git_url`
  - non-release slices: `examples/e2e-matrix.regres-*.yaml`
  - diagnostic slices for generated selectors: `examples/e2e-matrix.smoke-tiny.bank.yaml`, `examples/e2e-matrix.diagnostic.sentry.yaml`, `examples/e2e-matrix.diagnostic.temporal.yaml`, `examples/e2e-matrix.diagnostic.backstage.yaml`, `examples/e2e-matrix.diagnostic.airflow.yaml`, `examples/e2e-matrix.diagnostic.appwrite.yaml`, `examples/e2e-matrix.diagnostic.saleor.yaml`
  - diagnostic live evidence should rotate product domains and feature areas between runs where feasible, so success history is not based on a single product or one repeated feature path
  - release-specific slices, `baseline` + `parallel-default`, strict blockers и release verdict policy живут только в `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
  - matrix invariant: для одного `profile_id` shard-plan должен совпадать между `baseline` и `parallel-default`
  - для `source_kind=git_url` refs должны быть pinned
  - child batch stdin is detached from the planned profile/sweep combinations file; regression coverage forces a dummy child to drain stdin and still requires all matrix rows to execute
  - release-mode пишет machine execution verdict `reports/release_verdict_<matrix-id>.json/.md`; non-release/diagnostic mode пишет neutral `reports/matrix_result_<matrix-id>.json/.md` без `release_state`
- итоговый release decision составной: `release_verdict_<matrix-id>.json = PASS`, `swe_ux_assessment_<matrix-id>.md = accepted`, `swe_artifact_quality_assessment_<matrix-id>.md = accepted`
- SWE artifact assessment начинается с promoted/user-visible selected-run evidence; taskrun
  quality JSON и runtime counters проверяются только как execution diagnostics и не являются
  surrogate artifact acceptance.
  - pre-tag/offline verifier: `python3 scripts/verify-release-verdict.py reports/release_verdict_<matrix-id>.json`; скрипт проверяет release-mode contract/providers/run indexes/records plus matching accepted SWE reports и не запускает live harness
- `scripts/frontend-live-e2e.sh` и `npm run e2e:live --prefix ui` используют Playwright:
  - local wrapper поддерживает `claude-code`, `qwen-code`, `codex-code`
  - canonical toggles: `UI_E2E_EXPECTED_REPO_COUNT`, `UI_E2E_SCENARIO=init-inspect`, `UI_E2E_ARTIFACT_SOURCE=snapshot`, `UI_E2E_SNAPSHOT_RUN_ID`, `UI_E2E_OUTPUT_DIR`
  - release-facing `init-inspect` validates the current ProductShell over the copied backend refresh snapshot: contextual Setup, `Home / Runs / Knowledge / Changes`, run deep-link reload/Back restoration, snapshot-isolated Evidence/Publish and global Ask citation return
  - rendered acceptance covers `1440`, `1280`, `1024`, `390x844`, global overflow, first-viewport state/action, keyboard focus/Escape/return focus, critical axe results and browser console errors
  - durable ProductShell screenshot refs are diagnostic evidence only: `frontend-home-desktop.png`, `frontend-setup-desktop.png`, `frontend-runs-desktop.png`, `frontend-knowledge-desktop.png`, `frontend-changes-evidence-desktop.png`, `frontend-changes-publish-desktop.png`, default Ask evidence `frontend-ask-desktop.png`, `frontend-changes-publish-mobile.png`, and `frontend-changes-evidence-mobile.png`
  - cancellation/page-close behavior проверяется deterministic fake-runtime UI/API tests, а не live provider release gate
  - init inspect обязан различать `active_run_timeout`, `runtime_run_failed`, `browser_closed`, `api_unreachable`, `server_exited` и fallback `playwright_failed`, чтобы backend run failure, browser lifecycle, API/server lifecycle и productive runtime timeout не выглядели одним failure class
  - long-running run polling использует independent API request context и не зависит от lifetime browser page, которая нужна только для UI assertions
  - failed Playwright runs with a still-active selected/QA run must request cancel through `/api/pipeline/runs/<run_id>/cancel` before server teardown; remaining cancellation evidence is secondary to the frontend reason
  - init poll budget берётся из effective runtime timeouts and is not raised to `ACP_PIPELINE_TIMEOUT_SEC+30` by default; canonical snapshot mode uses this budget for UI/API/artifact inspection, not a fresh provider-backed init; follow-pipeline mode requires explicit `UI_E2E_INIT_TIMEOUT_FOLLOW_PIPELINE=1`, and `UI_E2E_INIT_TIMEOUT_CAP_SEC` is an optional diagnostic cap
  - `frontend-e2e-result.json` keeps scenario, reason, run id, last run status/error/current step and diagnostic refs stable; screenshots, traces and videos remain evidence metadata and do not change release verdict semantics
  - live UI smoke раскрывает Activity / Events drawer before interacting with log-mode controls, and Playwright action timeouts are bounded so hidden controls fail as `frontend_failed` instead of consuming the full runtime polling budget
- Этот документ фиксирует policy, invariants и required gates; пошаговые live/release cookbook команды не дублируются здесь.

## 8) Acceptance для testing strategy

- любой required CI run проходит без live network dependencies
- любое изменение schema/spec/examples требует update fixtures/golden в том же PR
- live headless provider smoke не блокирует merge; для обязательного CI используется только `contracts`, `backend`, `ui`, `golden`, `smoke-cli`, `smoke-api`
- release gate выполняется вручную перед релизом на trusted машине по `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- pre-tag release check использует `scripts/verify-release-verdict.py` поверх уже созданного `reports/release_verdict_<matrix-id>.json`; это не required CI и не live runner
- scenario fixtures и golden outputs считаются канонической regression surface до появления production-scale test corpus
- optional readable golden export доступен для review-diff:
  - `ACP_EXPORT_SCENARIO_GOLDEN=1 go test ./internal/orchestrator -run TestScenarioFixturesDeterministicInitPipeline -count=1`
- tracked generated artifacts policy:
  - `internal/api/ui_dist/*` и `fixtures/scenarios/*/golden/readable/*` остаются versioned в git как часть baseline/release surface
  - `make verify-readable-fixtures` checks every readable export path/digest against its adjacent
    machine `snapshot.sha256`; machine-only snapshot entries remain valid
  - UI source changes must leave `internal/api/ui_dist/*` fresh: run `make build` to regenerate
    the embedded bundle and `make verify-ui-dist` to prove the committed bundle matches the
    current Vite output.
  - controlled snapshot refresh:
  - `ACP_UPDATE_SCENARIO_GOLDEN=1 go test ./internal/orchestrator -run TestScenarioFixturesDeterministicInitPipeline -count=1`

## 9) Технологические defaults

- Public product APIs и schema contracts этим документом не меняются
- для schema validation в CI используется Draft 2020-12 compatible validator
- contract validation tools (`ajv-cli`, `ajv-formats`, `js-yaml`) live in
  `tools/contracts`; version changes require an explicit `package.json`/`package-lock.json`
  review and `make contracts` must run from the lockfile-backed install.
- Python tooling runtime is pinned by `.python-version` (`3.10.8`). Required CI installs it via
  `actions/setup-python`, and Makefile/script-test entrypoints use `scripts/run-python.sh` so a
  wrong interpreter fails before Python suites or verifier scripts run.
- основной backend test loop предполагает `go test`
- canonical `make lint` проверяет Go formatting, ShellCheck для production `scripts/**/*.sh`
  и UI TypeScript typecheck; ShellCheck baseline — `0.11.x`, live provider/network access не
  требуется.
- UI smoke стек: `React + Vite + Vitest + Playwright`; required `e2e:mock` is deterministic and
  provider-free, while `e2e:live` remains a trusted-machine diagnostic/release-gate tool.
- UI V8 coverage baseline is informational only, not a required CI threshold. Baseline from
  `2026-07-14` (`npm run coverage --prefix ui`): statements `85.36%` (`3220/3772`), branches
  `77.03%` (`3200/4154`), functions `88.54%` (`812/917`), lines `85.22%` (`3087/3622`). Future
  threshold ratchets should only move upward and must be explicit package/config/docs changes.
- Balanced timeout defaults:
  - step `1800s`, heartbeat `30s`, pipeline `2400s`, kill-grace `30s`
  - api-ready `60s`, api-init `120s`, ui-init poll `900s`, ui-cancel poll `420s`
- Canonical live matrix timeout presets:
  - `short-window`: step `3600s`, pipeline `7200s`, ui-init `1200s`
  - `medium-window`: step `5400s`, pipeline `14400s`, ui-init `1500s`
  - `extended-window`: step `10800s`, pipeline `21600s`, ui-init `1800s`

## 10) Developer entrypoints

- `make bootstrap`
- `make contracts`
- `make test`
- `make lint` (gofmt + ShellCheck + UI typecheck)
- `make build`
- `make offline-closure` (complete provider-free Epic 22 closure: race/fault/path/boundary,
  readable-fixture drift, UI unit/rendered mock suites, deterministic DoD, embedded UI and
  source-repository cleanliness; never runs live providers or canonical matrices)
- `make verify-ui-determinism`
- `make verify-ui-dist`
- `make run-backend WORKSPACE=/abs/path/to/arch-workspace`
- `make run-ui`
- `./scripts/full-run-batch.sh`
- `./scripts/full-run-batch-matrix.sh`
- `./scripts/frontend-live-e2e.sh`
- runtime live log seam:
  - mixed `event` + `runtime_output` entries в run logs
  - `runtime_output.stream` (`stdout|stderr`) сохраняется и не ломает pagination
  - hard-cap truncation marker фиксируется как `fields.output_truncated=true`
- Step 2 diagram compiler regression:
  - deterministic C4 artifacts + stable index ordering
  - strict evidence gap markers (`Gap:*`) при недостатке данных
  - sanitized Mermaid node ids and generated diagram paths remain collision-free for distinct entity ids that normalize to the same slug
