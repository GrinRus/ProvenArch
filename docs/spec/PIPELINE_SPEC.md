# Спецификация пайплайна (MVP v0)

Документ описывает pipeline ACP через input/output контракты и expected artifacts.

## Общие понятия

- **Workspace**: единый central git-репозиторий `arch-workspace/` (каноническая MVP-конвенция, Variant 2) с `workspace.yaml`, `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/`.
- **Workspace manifest**: `workspace.yaml`, валидируемый по `schemas/workspace.schema.json` и описанный в `docs/spec/WORKSPACE_SPEC.md`.
- **Orchestrator**: управляет шагами, готовит PromptPack/ContextPack, выдаёт runtime staged-write envelope, валидирует manifests/indexes/verdicts и persisted runtime execution metadata.
- **Runtime (MVP)**: headless multi-provider (`claude-code` default, `qwen-code` optional, `codex-code` release peer) + deterministic fake harness (default for required CI/testing).
- **Runtime execution metadata**: internal metadata artifact runtime-шага (`task_id`, `run_id`, `step_id`, `provider`, write roots, status, raw output refs), не semantic source of truth для live docs-first flows.

## Docs-first runtime contract

Docs-first contract schemas для live pipeline:
- `schemas/shard-pack-manifest.schema.json`
- `schemas/final-run-index.schema.json`
- `schemas/citation-index.schema.json`
- `schemas/validator-verdict.schema.json`
- `schemas/qa-answer.schema.json`

Artifact ownership:
- provider-authored runtime artifacts: `shard-pack-manifest.json`, runtime draft manifests/files и `validator-verdict.json`
- orchestrator-authored staged artifacts: `final-run-index.json`, `citation-index.json`, shard plans/summaries и run logs/history
- compiler-derived promoted artifacts: `model/*`, diagrams, normalized coverage/findings renderers и approved `reports/*` / `proposals/*`

Staged write model:
- runtime шаги пишут только в step-local `write_root` и `draft_final_root`
- shard analysts пишут только в `reports/taskruns/<run_id>/staging/shards/<shard_id>/`
- aggregator/orchestrator materialize-ит staged final set в `reports/taskruns/<run_id>/staging/final/`
- validator пишет только в `reports/taskruns/<run_id>/validator/`
- promotion копирует только approved final set в стабильные `reports/*` и `proposals/*`

Runtime write policy:
- `workspace root` больше не трактуется как implicit write target
- runtime получает explicit `artifact_root`, `write_root`, `draft_final_root`, `read_context_roots[]`, `step_contract`, `expected_artifacts[]`
- runtime не имеет права писать в `workspace.yaml`, `schemas/*`, `docs/spec/*`, `charter/*` и анализируемые user repos
- `runtime.profile.permissions.mode=trusted_full_access` сохраняет текущие provider full-access flags; opt-in `managed` отключает их и auto-approves только reads внутри `read_context_roots` и writes внутри `write_root`/`draft_final_root`
- managed requests для shell/network/package install/unknown tools не auto-approved; в non-interactive `approval_channel=fail_fast` это terminal `runtime_permission_required`
- Runtime write audit is an execution contract in live/headless mode: unexpected mutations of protected workspace surfaces or analyzed repo working trees are surfaced as run warnings/logs (`runtime_write_audit_unexpected_mutation`, `runtime_write_audit_repo_skipped`) and turn an otherwise-successful provider step into `runtime_contract_failed`; read-only isolated repo clones use a lightweight HEAD/index/mode integrity snapshot with `GIT_OPTIONAL_LOCKS=0` instead of writable `git status`, while writable repos still use porcelain status; if a repo status/snapshot cannot be read after runtime, ACP reports `runtime_write_audit_repo_skipped` with `status_unavailable_after_runtime` and also fails the step because the read input can no longer be trusted.

> MVP policy фиксирует step-scoped runtime provider contract: effective provider для шага выбирается как `workspace step override > CLI/env global provider > claude-code`; semantic stdout payloads не поддерживаются.
> CLI/process runtime mode задаётся флагом `--runtime fake|headless` (`fake` default, `headless` opt-in), global fallback provider — `--runtime-provider claude-code|qwen-code|codex-code` (env fallback `ACP_RUNTIME_PROVIDER`).
> В `--runtime fake` fallback provider валидируется как config surface, но execution metadata пишет neutral provider `fake`; live provider command не запускается.

Live matrix harness является manual trusted-machine surface, а не product runtime contract. Перед запуском child batches он выполняет host/provider/path preflight: selected-provider readiness, writable roots и минимальное свободное место (`E2E_MATRIX_MIN_FREE_KB`, default 5 GiB) для `E2E_TMP_ROOT`, `REPORTS_ROOT` и `MATRIX_ROOT`. Provider readiness probe/artifact-smoke subprocesses are bounded and run in their own process group; timeout terminates the full group before stdout/stderr collection so child processes cannot keep pipes open and hang preflight. For canonical `path` inputs the child backend cycle verifies the source checkout and rewrites the active repos file to run-local isolated detached clones under the batch temp root; canonical `/tmp/provenarch-live-e2e/...` checkouts remain prerequisites/source inputs, not provider write surfaces. Child batch затем выполняет DoD/UI precheck before headless runtime; DoD (`make contracts test lint build`) is bounded by `ACP_LIVE_PRECHECK_DOD_TIMEOUT_SEC`, while UI dependency/browser commands are bounded by `ACP_LIVE_PRECHECK_UI_TIMEOUT_SEC`. A bounded precheck timeout materializes as `precheck_failed` with `precheck-make.log`, `precheck-ui-npm.log`, or `precheck-playwright.log` evidence, not as provider/runtime or artifact-quality failure. The baseline API init simulation uses `api_init_timeout_sec` as the base budget, but the poll loop has bounded progress grace (`max(60s, 2x heartbeat)`) when run status/current step/artifact/warning counters advance; no-progress API init still fails with timeout evidence. Такие blockers materialize-ятся как `operational_host_preflight_failed` или `precheck_failed` и не смешиваются с runtime/artifact quality; если ресурсный сбой возникает уже после успешного preflight, execution report классифицирует это как infra-level failure, а не artifact-quality verdict.

## Repo source manifest (MVP)

В `workspace.yaml`:
- `version` обязателен и сейчас поддерживает только `1`
- `repos[]` обязателен
- `docs.imports_path` optional, default `./docs/imports`

В `repos[]` каждая запись содержит:
- `name`
- ровно одно из:
  - `path`
  - `git_url`
- optional `ref`

MVP policy:
- `path` используется для already-cloned локальных репозиториев;
- `git_url` допускает GitHub/GitLab-style sources, которые clone/fetch-ятся на той же машине через локальный `git` и текущий user/runner auth context;
- ACP в MVP не хранит отдельный credential store и не реализует собственный git access plane;
- имена репозиториев в одном workspace должны быть уникальными;
- layout `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/` не конфигурируется через manifest и считается fixed MVP convention;
- GitHub/GitLab hooks и manual pipeline button/job должны в итоге запускать тот же batch mode и те же step IDs.

## Charter, Cards и Skills (MVP format)

### Charter
Хранится в `charter/`, минимально:
- `charter/overview.md`
- `charter/glossary.yaml`
- `charter/nfr.yaml`
- `charter/rules.yaml`
- `charter/templates/`
- `charter/cards/domains/<domain-id>.md`
- `charter/cards/teams/<team-id>.md`

### Cards ownership model
- Step 0 wizard создаёт initial canonical `charter/cards/domains/*` и `charter/cards/teams/*`.
- Эти cards являются human-owned source of truth для domain/team IDs.
- Runtime pipeline не пишет в canonical `charter/cards/*`.
- Step 1 не создаёт и не переименовывает canonical domain/team cards автоматически.
- Если runtime обнаруживает новый домен, новую команду или неразрешимый owner gap, он создаёт `question` и/или `finding`, а не materialize-ит новый canonical card.
- `owner_team_id` в model должен ссылаться на существующий `team.<slug>`. Неизвестный owner фиксируется как unknown/question.

### Skills
Хранятся в `skills/` и редактируются в UI:

```text
skills/subagents.yaml
skills/<skill_name>/
  manifest.yaml
  prompts/
    system.md
    task.md
  templates/
    adr.md
    rfc.md
```

`manifest.yaml` (minimum):
- `name`
- `version`
- `applies_to`
- `inputs`
- `outputs`

`skills/subagents.yaml` в MVP обязателен и задаёт baseline agent roles и привязку skills/prompt packs.

## Baseline Agents/Skills/Prompts (MVP)

Обязательный baseline bundle:
- agents:
  - `domain-analyst`
  - `architect-aggregator`
  - `system-analyst-qa`
- skills:
  - `service-inventory`
  - `interface-extraction`
  - `integration-mapping`
  - `datastore-mapping`
  - `cicd-mapping`
  - `ownership-coverage`
  - `findings`
  - `proposals`
  - `qa`
- prompt packs:
  - `constitution`
  - `collect-context`
  - `findings`
  - `proposals`
  - `qa`

Bundle поставляется вместе с продуктом, хранится в workspace и может редактироваться пользователем через UI/git workflow.

## Docs imports metadata (MVP)

Для imported docs используется metadata index:
- `<docs.imports_path>/index.yaml` (default `docs/imports/index.yaml`)

Required поля записи:
- `id`
- `path`

Optional поля записи:
- `source`
- `checksum`
- `imported_at`
- `source_updated_at`
- `status`

Отсутствие index допустимо и не создаёт diagnostic. Malformed/index semantic issues (`duplicate id`, path traversal/absolute/outside imports root, missing referenced file) surfacing как warning-only workspace diagnostics.

## Canonical step IDs

### Init pipeline
- `init.step0.constitution`
- `init.step1.collect`
- `init.step2.asis_docs`
- `init.step3.findings`
- `init.step4.proposals`

### Refresh pipeline (manual)
- `refresh.step1.collect`
- `refresh.step2.asis_docs`
- `refresh.step3.findings`
- `refresh.step4.proposals`

### Q&A run
- `qa.ask`

## Runtime execution semantics

Canonical MVP runtime shape:
- `claude-code`, `qwen-code` и `codex-code` используют общий artifact-only process engine; provider adapters задают CLI args/stdin/workdir, unavailable markers и bounded activity/recovery policy
- `qwen-code` adapter invocation передаёт artifact prompt только через CLI `-p`; JSON task stdin не используется, custom qwen args нормализуются, чтобы не смешивать или подменять machine envelope с пользовательским prompt, а default invocation использует `stream-json` activity output без semantic stdout contract. `claude-code` и `codex-code` сохраняют свои transport-specific stdin/machine-mode surfaces. Default `codex-code` invocation disables plugin/app suggestion surfaces (`plugins`, `remote_plugin`, `plugin_sharing`, `apps`, `enable_mcp_apps`, `tool_suggest`, `skill_mcp_dependency_install`), добавляет `--ignore-user-config`/`--ignore-rules` и запускает процесс с auth-only isolated `CODEX_HOME`, куда копируются только `auth.json` и optional `installation_id`; пользовательские `config.toml`, MCP/plugins, app tools, `.tmp/plugins` и rules не входят в runtime input. Custom `HeadlessRunner.Args` остаются explicit caller-owned override.
- `codex-code` collect steps use a 5-minute initial and retry pre-artifact stall window. `claude-code` collect steps use a 5-minute initial and retry pre-artifact window for medium-slice shard recovery. These are provider policies, not matrix timeout overrides: if no authored collect artifacts appear within those windows, the step still follows the normal stall/focused-repair classification path. Write-first collect and focused repair commands also have an absolute pre-artifact wall-clock cap, so stdout/stderr-only stream activity cannot indefinitely extend a no-artifact command.
- `init.step0.constitution` normal prompt начинается с `FIRST CONSTITUTION DRAFT COMMAND`: `constitution-draft.json` и referenced draft files под `draft_final_root` должны быть first authored artifact set до broad analysis; `baseline-subagents.yaml` должен сразу быть валидным `skills/subagents.yaml` bundle (`agents:`), иначе следующий `refresh` блокируется workspace validation.
- `init.step0.constitution` normal prompt must not rely on focused repair as the happy path: after the first skeleton exists, the same provider turn must perform a bounded repo-entrypoint enrichment rewrite of `charter-overview.md` before successful exit.
- `init.step0.constitution` draft enrichment отличается от `step2/step4`: later pipeline evidence на этом шаге ещё не существует. Если bootstrap-only `charter-overview.md` дошел до enrichment, focused prompt обязан как следующий filesystem action прочитать текущий draft manifest/current charter overview плюс bounded repo entrypoint evidence from selected repo roots, затем fresh-overwrite exact `draft_final_root/charter-overview.md` before optional analysis. Валидный enrichment содержит target identity, repo/path evidence references, architecture scope, operating constraints, coverage gaps and decision-ready operator summary; sparse repo evidence записывается как explicit gap. `baseline-subagents.yaml` сохраняется, если он уже является валидным baseline bundle. Focused draft enrichment uses a minimum 3-minute pre-artifact wall-clock/stall window, and the step0 focused prompt names the exact `charter-overview.md` target plus bounded repo entrypoint candidates. Step0 final markdown is invalid if it mentions downstream/final/shard/validator/proposal/coverage/taskrun/runtime-provider evidence or generated timestamps. Для `git_url` repo resolved checkout может быть доступен только как `read_context_roots` path under `.acp/repos`; step0 enrichment includes that selected repo root directly while excluding workspace root and taskrun/report roots. If step0 markdown was freshly rewritten but still leaks process/downstream terms such as `draft manifest`, `validator output`, `bounded read roots`, or `later passes`, shared runtime may do one provider-authored `draft_artifact_enrichment_marker_cleanup` retry that rewrites `charter-overview.md` again. Noop, stall или scaffold-preserving output остаются `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`; ACP не синтезирует constitution markdown вместо provider and does not run a generic no-action enrichment retry.
- zero-output `pre_artifact` recovery остаётся adapter policy: `qwen-code` может сделать один warning/retry для любого artifact-monitored шага и один bounded focused collect-pair/draft-artifact retry после transient provider transport/API transcript (`[API Error: Premature close]`, `Connection error` with network socket/TLS disconnect, connection reset/closed, transient 5xx/529 stream errors) без artifacts, а `claude-code` делает zero-output warning/retry только для constitution/collect/validator/proposals steps (`init.step0.constitution`, `init|refresh.step1.collect`, `init|refresh.step3.findings`, `init|refresh.step4.proposals`); exhausted silent/API no-artifact collect retry with no authored artifacts gets one focused `collect_pair_repair` attempt before terminal classification. If that collect-pair repair stalls pre-artifact with empty stdout/stderr and no fresh authored mutation, policy-enabled providers get one bounded focused retry; repeated silent no-fresh exhaustion is `runner_unavailable`, while fresh-but-invalid repair output remains `runtime_contract_failed`. Exhausted non-collect silence remains `runner_unavailable`.
- diagnostic live runs may override provider activity windows with `ACP_PROVIDER_PRE_ARTIFACT_STALL_SEC`, `ACP_PROVIDER_PRE_ARTIFACT_WALL_CLOCK_SEC`, `ACP_PROVIDER_RETRY_PRE_ARTIFACT_STALL_SEC`, `ACP_PROVIDER_POST_ARTIFACT_STALL_SEC`, `ACP_PROVIDER_PARTIAL_ARTIFACT_STALL_SEC` and `ACP_PROVIDER_VALID_ARTIFACT_STOP_SEC`; these values are recorded in provider lifecycle `activity_policy` diagnostics and are treated as diagnostic timeout overrides, not release-mode evidence inputs.
- live batch harness enforces `pipeline_timeout_sec` as an outer hard deadline around each `acp run` pipeline. This deadline is separate from provider activity/stall windows: if it fires, the harness kills the whole `acp run` process group, reconciles active run history once with `error_code=runtime_timeout`, writes a failed `run-results.tsv` row when a run id can be recovered, and stops the remaining pipeline sequence. Provider heartbeat text alone is not liveness; useful progress is fresh non-heartbeat stdout/stderr bytes, fresh artifact file mutation outside raw/log/history surfaces, valid artifact controlled stop, or terminal provider exit. The watchdog records `last_output_activity_at` separately from `last_progress_at`, and ignores `reports/taskruns/**/logs`, `reports/taskruns/**/raw` and `run-history.json` for artifact progress so heartbeat/log churn cannot mask a stalled pipeline. Provider lifecycle diagnostics include `last_pipe_activity_at`, `last_artifact_mutation_at`, `artifact_observed`, `artifact_valid`, `artifact_state`, and `no_progress_duration_ms`.
- frontend live `init-inspect` uses the effective `ui_init_poll_timeout_sec` as its own execution budget and does not silently extend it to `pipeline_timeout_sec+30`. In the matrix harness, frontend smoke runs with `UI_E2E_ARTIFACT_SOURCE=snapshot`: the child batch copies the successful backend workspace into an isolated `frontend-workspace`, merges `snapshots/<refresh-run-id>/reports` over its reports tree, writes a temporary `reports/taskruns/run-history.json` for that `refresh_run_id`, and Playwright inspects the existing completed run instead of starting a second provider-backed `init`. `UI_E2E_ARTIFACT_SOURCE=live` remains a direct-harness diagnostic mode. Trusted diagnostic runs may opt into follow-pipeline behavior with `UI_E2E_INIT_TIMEOUT_FOLLOW_PIPELINE=1`; such timeout overrides are diagnostic evidence only, not canonical release readiness. Optional Ask smoke has its own bounded `ACP_UI_QA_POLL_TIMEOUT_SEC` budget; timeout is reported as `ACTIVE_RUN_TIMEOUT` / `active_run_timeout`, and the frontend shell best-effort cancels the active QA run before server teardown so provider processes do not linger after a failed smoke.
- `qwen-code` draft steps (`step0.constitution`, `step2.asis_docs`, `step4.proposals`) могут остановиться после bounded valid-artifact window, если draft manifest и все referenced draft files уже валидны, но provider продолжает стримить или мутировать draft files; focused provider-authored repair attempts для collect/validator/draft artifacts также могут остановить still-running provider после появления валидного repair artifact и принять результат только через тот же validation gate. Для collect это означает strict `ValidateCollectManifestInRoot`, включая referenced markdown document checks; parse-only `shard-pack-manifest.json` не является valid-artifact stop signal. Это artifact-only controlled completion, не расширение timeout; partial/invalid artifacts сохраняют обычную post-artifact stale-signal policy и остаются failure после exhausted repair.
- `step2/step4` draft markdown must be readable operator-facing content, not raw structured evidence dumps. Provider-authored markdown that pastes Python/JSON object fragments (`{'id': ...}`, `documents=[{...}]`, `citations=[{...}]`) fails draft validation; it must summarize counts, selected document titles, citation ids and repo/path refs. Normal draft prompts may include bounded current-run evidence hints computed from public artifacts, including exact typed shard completeness (`planned/succeeded/failed/incomplete`) and a preview of exact current-run finding IDs, so the provider can copy observed values instead of guessing. `step2.asis_docs` must not claim current-run `final-run-index.json` / `citation-index.json` are missing, not observed, unavailable, not yet present, or not yet available, because those indexes are downstream final staging artifacts; when absent during step2, index availability should be omitted from the report. When a readable current-run typed shard-summary `items[]` is visible, `step2` `summary.md` and `architect-summary.md` should carry the exact literal `planned=<n> succeeded=<n> failed=<n> incomplete=<n>`; if all current-run shards succeeded, both should include an explicit `no-shard-coverage-blocker` statement instead of generic failed/incomplete caveats. `overview.md` must include concrete repo/path, citation, or staged artifact refs, and `architect-summary.md` must include decision-ready operator next-action/inspection cues. Step2 markdown writes must preserve literal backticks and paths through shell-safe single-quoted heredocs or literal Python writes; empty evidence slots such as `from  and`, `checked:  and`, `under .`, or `Use  and` are runtime draft validation failures. Metadata-only bullets such as `meta:`, `step_id:`, `domain_id:`, `strategy:`, `max_parallel_tasks:`, `failure_policy:` or `shard_discovery_mode:` and false empty-shard claims such as `staging shard directory contains 0 files`, `Shard pack manifests: none observed`, or `no shard manifests observed` are runtime draft contract failures. `step4.proposals` must not claim the current-run final index document list is unavailable; when `final-run-index.json` is present it must summarize the observed canonical document count, and when absent it should omit that index status. `step4` proposal/changelog required sections must be non-empty, include concrete current-run evidence refs, and use the validator section names: `proposal.md` includes `Decision / recommended operator action`, `Evidence used`, `Proposed changes or follow-up plan`, and `Risks, gaps, and out-of-scope notes`; `changelog.md` includes `Updated architecture/proposal surfaces`, `Findings/proposals summary`, `Evidence index or citation references`, and `Residual coverage gaps`. When typed status is visible both files must contain the exact literal shard completeness string `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` plus an explicit `no-shard-coverage-blocker` statement; dangling `findings above` references or generic follow-up plans are invalid unless the same draft contains substantive linked findings/proposals or the proposed plan explicitly records a no-actionable-proposal gap tied to observed evidence.
- `reports/as-is/overview.md` is the canonical Architecture Home. Step2 validation requires non-empty `System at a glance`, `Analyzed scope`, `Domains and ownership`, `Key flows`, `Integrations and datastores`, `Where to start`, `Safe-change guidance`, and `Evidence gaps and open questions` sections. Each required heading MUST occupy its own Markdown H2 line and its substantive body MUST begin on following lines. When the sole contract error is that every exact required H2 has one non-empty inline authored body, in canonical order, the runtime MAY atomically insert only the heading/body line boundary and MUST repeat the complete strict draft validation. Partial, duplicate, reordered, empty, malformed, or multiply-invalid input is not eligible; failed revalidation MUST restore the original bytes and continue through the existing provider-authored repair or fail-closed path. Canonical-document or stable repo-relative (`<repo>:<path>`) references are required where evidence exists. Repo references MUST resolve to an exact existing non-root file or directory; repository-root shorthand (`<repo>:.`, `<repo>:./`) and wildcard/glob syntax (`<repo>:src/*`) are invalid evidence references and unresolved candidates MUST be stated only as gaps. Manifest recap, runtime/provider narration, run/current-run identity, any typed-shard/shard-pack recap, exact `planned/succeeded/failed/incomplete` counters, taskrun staging and absolute runtime checkout / `.acp/repos` paths are not acceptable architecture content. Shard completeness belongs in coverage/architect summary rather than Architecture Home.
- draft enrichment manifest shape is strict. If provider-authored enrichment already fresh-rewrote evidence-backed markdown but inserted unknown manifest fields such as `status`, `content_digest`, `enriched_at`, `metadata`, `validation`, `confidence`, `source`, `logical_path`, `target`, `output_path` or `publish_path`, runtime may run one provider-authored `draft_artifact_enrichment_manifest_shape` retry. The retry must restore allowed manifest keys (`version`, `run_id`, `step_id`, `step_contract`, `agent_role`, optional `summary`/`updated_at`, `outputs[]` with `path`/`canonical_path`/`kind`/`title`) and keep or fresh-overwrite every markdown target; parser strictness is not relaxed and repeated manifest drift remains `runtime_contract_failed`.
- stdout/stderr provider transcripts сохраняются как diagnostics/raw-output refs и никогда не трактуются как semantic success payload
- selected-provider readiness фиксируется в batch preflight; provider command/probe/auth/quota/version и codex CLI compatibility blockers являются operational failures до deep run. Live E2E по умолчанию запускает только `codex-code` с явными `ACP_CODEX_MODEL=gpt-5.5` и `ACP_CODEX_REASONING_EFFORT=xhigh`; `qwen-code` и `claude-code` остаются на provider CLI defaults. Provider `model`/`modelUsage` telemetry не является readiness contract и не блокирует release сама по себе
- semantic source of truth для `step1` — `shard-pack-manifest.json.semantic`
- `step1` semantic provenance evidence (`entities/edges/findings[].provenance.evidence[]`) обязаны содержать non-empty `repo` + `path`; citation-only semantic evidence objects невалидны
- semantic source of truth для `step3` — `validator-verdict.json.findings[]` и `validator-verdict.json.questions[]`
- для multi-repo `init|refresh.step3.findings` first-action `validator-verdict.json` skeleton обязан включать минимум один PASS-compatible cross-repo finding и один question с repo/path provenance и `related_ids` по нескольким repo scopes; пустой валидный verdict skeleton допустим только для single-repo validator tasks
- для multi-repo release profiles cross-repo signal считается валидным только через explicit `semantic.edges[]`, finding provenance по нескольким repos или question `related_ids` по нескольким repo scopes при наличии repo-specific citation coverage; простое перечисление repos без связи остаётся `analysis:cross-repo-missing`
- runtime execution metadata сохраняют только execution context, status, warnings и raw-output references
- `qa.ask` is a runtime task family, not an init/refresh promotion step: ACP prepares `context-pack.json`, selected provider writes `qa-answer.json`, and canonical workspace artifacts/source repos are not mutated.
- raw provider prompt argv payloads and stdout/stderr diagnostics are redacted before persistence and run-log streaming; lifecycle diagnostics keep prompt size, and prompt argv payloads are replaced with byte count + hash when present; stdout/stderr remain diagnostic only
- deterministic fake runtime пишет provider `fake`; headless adapters пишут `claude-code`, `qwen-code` или `codex-code`

## Source revision and advisory impact artifacts

Before every non-QA pipeline execution, ACP writes `reports/taskruns/<run_id>/source-revisions.json` using `schemas/source-revisions.schema.json`. The baseline is the newest prior successful `init|refresh` whose matching final index and PASS validator verdict are valid and whose source revision artifact is valid. Legacy runs without the artifact are not inferred. The SHA-256 input fingerprint covers configured repo source/scope, `docs.imports_path` content, `charter/**` and `skills/**`; generated reports/model/proposals/taskruns and process-level provider settings are excluded. Resolved absolute checkout paths are never persisted.

Before `refresh.step1.collect`, ACP writes immutable advisory `refresh-impact-plan.json` and factual `refresh-execution.json`. Git changes come from `git diff --name-status -z -M -C <baseline>..<current>` and preserve rename/copy identity. Planning accepts at most 10,000 complete changed paths; larger, dirty, unavailable, diverged, unreadable or unmapped input requires full refresh. Safe unchanged/out-of-scope candidates succeed without provider execution or canonical rewrites. Selective execution replays only validator-promoted baseline shard packs whose current shard identity matches; any missing pack or mismatch switches to full before the first provider call. Collect prompts receive bounded affected paths and at most 20 commit subjects per repo (200 characters each, 64 KiB total); source evidence is authoritative over Git intent text. `refresh-materialization.json` records publication decisions and content hashes.

## Init pipeline

### Step 0 — Constitution (human-in-the-loop)
Inputs:
- шаблоны charter
- baseline prompt pack `constitution`
- пользовательские правки в UI
- persisted structured wizard contract `charter/wizard/step0-contract.json` (optional)

Outputs:
- обновлённые `charter/*`
- initial canonical `charter/cards/domains/*`
- initial canonical `charter/cards/teams/*`
- runtime draft manifest `constitution-draft.json` + optional draft finals в `draft_final_root`

Step 0 materialization policy:
- если `charter/wizard/step0-contract.json` валиден, его поля детерминированно влияют на `charter/*` и canonical cards;
- если contract отсутствует/невалиден, применяется baseline fallback materialization;
- fallback фиксируется warning-сообщением в run diagnostics (`GET /api/pipeline/runs/<run_id>.warnings`).
- runtime draft не пишет canonical `charter/*` напрямую; compile/materialization layer остаётся единственной publish surface.
- support-only baseline bundle seeding не пишет canonical `skills/subagents.yaml`; этот файл публикуется только из validated `constitution-draft.json`.

### Step 1 — Collect context (runtime step)
Inputs:
- `workspace.yaml` из корня central `arch-workspace`
- локальные checkout репозиториев, полученные из `path` и/или local git resolution of `git_url` на той же машине
- `<docs.imports_path>/index.yaml` (если есть) + configured imports files
- `docs/rfcs/*`, `docs/meetings/*`, `docs/decisions/*`
- `charter/*`
- `skills/*`

Runtime focuses on:
- arbitrary stacks через выбранный headless provider (`claude-code|qwen-code|codex-code`) + baseline skill/prompt bundle, без фиксированного whitelist parser implementations в MVP
- workspace prompt packs участвуют в composed prompt как editable content layer; merge order фиксирован: provider header -> task-specific first-action artifact command -> artifact-only/filesystem policy -> step-specific policy -> workspace prompt pack -> provider completion footer. Enforced runtime policy/invariants задаются internal shared step-policy слоем и не могут быть ослаблены содержимым prompt pack. Editable prompt pack layer подключён к `step0.constitution`, `step1.collect`, `step3.findings` и `step4.proposals`; `step2.asis_docs` работает через enforced policy only без отдельного editable `as-is` prompt pack.
- service topology и entrypoints
- interfaces (HTTP/gRPC/events)
- external systems/integrations
- datastores и storage usage
- CI/CD evidence (`.gitlab-ci.yml`, Dockerfile, deploy manifests, helm/k8s, scripts)
- ownership hints и явные unknowns

Primary runtime output:
- authored shard docs inside shard `write_root`
- `shard-pack-manifest.json`
- optional shard-local draft finals рядом с manifest, но только внутри shard staging root

Orchestrator applies:
- валидирует `shard-pack-manifest.json`
- требует non-empty `documents[]` и `citations[]`; collect без authored documents или без
  repo-backed citations является invalid artifact до checkpoint/apply
- требует полного `semantic` block в `shard-pack-manifest.json`: `coverage`, `questions`, `entities`, `edges`, `findings` обязательны, а `questions/entities/edges/findings` materialize-ятся массивами даже when empty
- collect manifest принимает только canonical vocabulary: `semantic.coverage.observed`, `semantic.questions[*].text`, `semantic.edges[*].type/from/to`, object-shaped `provenance`, numeric `confidence`; aliases вроде `covered_topics`, `question`, `relation`, `source`, `target`, finding `summary`/`inference`, array provenance, string confidence, `evidence_citation_ids`, top-level `step_contract` и `compatibility` считаются contract-invalid drift и reject-ятся schema/contract validation-ом
- `documents[].path` должен быть artifact-root-relative и должен resolve-иться в существующий provider-authored file under collect `write_root`; missing file references, directory references, absolute paths, workspace-level prefixes, duplicated `artifact_root` prefixes и provider/tool side-effect dirs (`.qwen/`, `.claude/`, `.codex/`, `.git/`, `node_modules/`) являются invalid collect artifacts, поэтому `step2` не должен первым обнаруживать broken document reference
- manifest identity fields `run_id`, `step_id`, `shard_id`, `domain_id` и `artifact_root` должны точно совпадать с текущей collect task identity. Существующий authored document под фактическим `write_root` не делает manifest валидным, если manifest указывает foreign/stale task identity; такой drift отклоняется до `step2`, проходит обычный provider-authored repair path и повторную полную validation, но не нормализуется ACP.
- `documents[].canonical_path` должен быть stable promoted workspace path, обычно `reports/as-is/<shard>/<doc>.md`, и не должен ссылаться на live staging surface. `reports/taskruns/**`, `/staging/`, absolute `write_root`, duplicated `artifact_root`, raw runtime logs/metadata и provider side-effect paths являются contract drift. Collect pair/manifest repair prompts обязаны показывать exact mapping from authored `documents[].path` to stable `documents[].canonical_path`, чтобы live provider не копировал staging path из `write_root`.
- `documents[].citation_ids` must be non-empty, reference existing `citations[].id` values and be reciprocal with `citations[].document_ids`.
- `citations[].repo/path` и `semantic.entities|edges|findings[].provenance.evidence[].repo/path` должны resolve-иться к текущему repo scope/root и реально существующим relative file paths under that repo root when task repo roots are available. Directory-only evidence refs are invalid: directory scopes may guide bounded reads, but citations/provenance must name a concrete file. Missing/guessed repo evidence paths are invalid collect artifacts before `step2`; unsupported claims must be removed or recorded as coverage gaps rather than cited to non-existent files. `citations[].id` values must be unique; `citations[].claim_ids` and `citations[].document_ids` must be non-empty. Every `citations[].document_ids` value must resolve to a current `documents[].id`, and the referenced document must list that citation ID. When one authored markdown document cites multiple repo files, provider must derive citation IDs from shard/document stem plus repo path slug and update `documents[].citation_ids` to those exact IDs. Provider prompts require file-level checks (`test -f`, `rg --files`, or portable `find ... -type f -print`) plus local structural checks for `semantic.questions[*].id/text`, duplicate `citations[].id`, citation `claim_ids`/`document_ids`, and reciprocal document/citation references before successful collect exit; syntax-only checks such as `jq empty` / `python3 -m json.tool` are not sufficient proof.
- требует global uniqueness для `citations[].claim_ids` в assembled staged final set; provider должен формировать `claim_id` как semantic stem + shard slug и добавлять deterministic numeric suffix при остаточной коллизии
- `shard-pack-manifest.json` не допускает legacy top-level `claims`, `claim_map`, `metadata`, `validation`, `compatibility` или alternate semantic wrappers; claim identity is only non-empty `citations[].claim_ids`, and empty arrays or unreplaced template tokens such as `SHARD`, `<shard>`, `<claim>`, `TODO`, or `REPLACE_ME` are runtime contract failures
- collect prompt ставит suggested authored doc path и literal task-specific `shard-pack-manifest.json` skeleton в `COLLECT EVIDENCE-FIRST ARTIFACT PAIR` section сразу после provider identity; skeleton является schema/key/type guide, а не heredoc artifact. Первый collect filesystem work unit допускает только два mechanically simple действия: bounded evidence read/list по existing repo entrypoint hints, concrete path-scope file candidates и assigned `path_scopes` (до 8 representative files, до 6000 bytes per file; oversized files truncate/skip with continue), затем direct literal write provider-authored markdown + `shard-pack-manifest.json` через shell heredoc/printf/tee. Directory scopes may guide discovery, but generated prompt candidates are existing files from each path scope where possible so multi-scope shards do not starve later scopes. До появления обоих target-файлов provider не должен читать `reports/taskruns/**`, raw logs, sibling shard manifests или archive docs, не должен делать analysis-only/status/progress prose, todo/planning, broad repo sweep, second read-only preflight, Ruby/Node/Python/Perl/awk/jq inline writer, generated source-code strings, template programs или nested quote tricks. Если direct write command падает до создания обоих targets, provider обязан сразу повторить simpler direct literal write from observed evidence, а не ждать focused repair; normal collect не должен рассчитывать на focused repair как штатный success path. Normal collect больше не пишет seed-only pair как первый filesystem action. Strict collect validation fail-ит marker-bearing docs, marker-free unchanged seed docs, recovery-fallback docs и scaffold-only semantic snapshot (`contains scoped surface` + generic owner mapping gap) вместо `artifact_only` success. A clean operator-facing coverage gap such as “no OpenAPI/Swagger spec was observed under this scope” is allowed when it is not a guessed citation/provenance path or runtime process narration.
- collect runtime до финальной validation может выполнить focused recovery: если provider оставил stdout/stderr diagnostics, но не записал authored artifacts, или если fully silent zero-output collect fresh retry exhausted без authored artifacts, общий engine делает одну provider-authored collect pair-recovery попытку с write-set guard только на suggested authored doc + `write_root/shard-pack-manifest.json`; если provider уже записал bootstrap-only/seed-only/failure-only authored doc или temporary doc с interrupted read / `initial artifact` / `will be repaired with concrete...` wording, collect validation reject-ит такой referenced document и runtime не маскирует его pair-recovery success. Если collect-pair repair itself stalls before a fresh authored mutation with empty stdout/stderr, policy-enabled providers get one bounded focused retry of the same provider-authored repair contract; the first silent no-fresh stall is not emitted as exhausted until that retry also fails. Если pair-recovery stalled/failed after writing non-bootstrap authored markdown but before a valid manifest, engine may continue into manifest-only/runtime recovery instead of treating the markdown-only partial as immediate terminal failure; if pair-recovery produced no authored artifact after the allowed retry, exhaustion remains terminal. Если structural-invalid manifest содержит missing repo evidence path и existing authored markdown всё ещё упоминает этот path, runtime эскалирует в provider-authored collect pair repair вместо manifest-only repair: exact markdown target становится existing authored doc, success требует fresh markdown mutation plus valid manifest, а noop/stale markdown завершается `runtime_contract_failed` с `collect_pair_repair_noop_or_stale_markdown`. Если provider уже записал non-bootstrap authored docs, но `shard-pack-manifest.json` отсутствует, существующий manifest отвергнут как scaffold-only semantic snapshot, or existing clean manifest failed only because citation `claim_ids` are empty/missing, общий engine сначала применяет deterministic `collect_manifest_runtime_recovery`; если manifest structural-invalid по other links/schema, или deterministic recovery не смогла пройти validation, engine делает одну manifest-only provider repair попытку с write-set guard только на `write_root/shard-pack-manifest.json`. Единственное shape-only исключение до provider repair — existing process-clean manifest, у которого единственная contract-ошибка состоит в отсутствии schema-required `semantic.findings`: runtime может атомарно вставить только `semantic.findings: []`, сохранить before/after digest, повторить полную strict validation документов/evidence и вернуть исходные bytes при любой другой ошибке. Это не semantic synthesis и не artifact-quality acceptance; missing/wrong coverage, questions/entities/edges, citations, evidence paths, nested fields или malformed JSON не нормализуются. Если manifest-only provider repair после structural-invalid manifest failed only on clean manifest-shape errors (`semantic.questions[*].text` missing, duplicate `citations[].id`, missing citation `claim_ids`/`document_ids`, or broken `documents[].citation_ids` references), runtime may run exactly one additional provider-authored `collect_manifest_shape_cleanup` retry through the same manifest repair adapter, passing the terminal validation error and keeping the write-set guard limited to `write_root/shard-pack-manifest.json`. This cleanup is not deterministic ACP-side synthesis and is forbidden for missing repo evidence, process-contaminated/empty/bootstrap markdown, write-set violations, or no authored markdown. Если cleanup или original manifest-only provider repair exhausted/stalled или вышел без валидного manifest, run остается `runtime_contract_failed`; deterministic runtime recovery больше не применяется как fallback после такого repair exhaustion, чтобы guessed/missing repo evidence не превращался в normal succeeded shard. Runtime recovery читает только existing authored markdown under `write_root`, использует bounded regular-file `repo/path` evidence, пишет только `write_root/shard-pack-manifest.json`, логирует recovery diagnostic/runtime warning и снова проходит strict collect validation. ACP не нормализует `documents[].path`, не автозаполняет metadata вне runtime-recovered manifest и не использует semantic stdout repair
- все required runtime artifacts пишутся/проверяются по exact absolute `write_root`/`draft_final_root`; relative CWD checks/writes (`test -f validator-verdict.json`, `test -f nova-overview.md`) не являются валидным artifact target
- `step3.findings` при missing/invalid `validator-verdict.json` может выполнить одну focused provider-authored repair попытку: разрешённый write-set — только `write_root/validator-verdict.json`; repair prompt начинается с command-first absolute heredoc skeleton, но skeleton `checked_paths` ссылается на staged final artifacts, а не validator `write_root`; `issues[]` использует canonical shape `code`, `severity=error|warning`, `message`, optional `path/document_id/citation_id`; legacy finding-shaped fields внутри `issues[]` hard-invalid
- draft steps (`step0.constitution`, `step2.asis_docs`, `step4.proposals`) при missing/invalid draft artifacts могут выполнить одну focused provider-authored repair попытку: разрешённый write-set — step manifest в `write_root` и files under `draft_final_root`; repair prompt содержит command-first heredoc artifact set для manifest + referenced draft files, задаёт exact absolute `write_root`/`draft_root` один раз и дальше пишет через эти shell variables, запрещая broad analysis и ручное перепечатывание/переписывание path components до первого artifact set. Если validation уже показывает bootstrap-only draft content, runtime пропускает scaffold-style `draft_artifact_repair` и сразу выполняет focused `draft_artifact_enrichment`; если validation/repair result is a recoverable draft semantic/shape failure (`runtime draft manifest outputs are invalid`, malformed markdown, missing exact shard completeness, missing proposal sections, missing finding linkage/actionability, stale structured-finding denial, process/downstream wording), runtime also routes one provider-authored enrichment call instead of terminal scaffold failure. Structural missing draft files still use draft repair first. Enrichment не содержит heredoc scaffold, может менять только step manifest и referenced files под `draft_final_root`, читает bounded current-taskrun staging evidence и обязан fresh rewrite-нуть каждый referenced markdown draft evidence-backed content. Для `step2/step4` enrichment include scope не добавляет whole headless workspace или whole source repo; он ограничен current `write_root`, current `draft_final_root`, current taskrun `staging/shards` и `staging/final` when present. Prompt требует bounded evidence pass: all available `shard-pack-manifest.json`/indexes плюс небольшой representative authored-doc sample, а не чтение всего shard corpus. Draft manifest используется только как contract/output map; bootstrap summary (`Drafted required runtime artifacts for this step`), schema keys, canonical_path examples and validation/scaffold wording must not be copied into final markdown. Final markdown must read as operator-facing architecture/proposal content; process narration such as `current draft manifest`, `manifest target remains`, `bounded staged evidence`, `bounded evidence read`, `bounded read roots`, `bounded read pass`, `recovery pass`, or `enrichment read` is treated as scaffold/recovery contamination; an operator-facing gap should state the missing current-run evidence/finding directly instead of describing bounded roots or recovery mechanics. Draft validation также reject-ит malformed markdown evidence: unbalanced inline-code/path backticks, unclosed code fences, raw structured dumps, stale missing-index claims and stale zero-document final-index claims such as `final-run-index.json contains 0 observed document entries` or `final-run-index.json (0 observed document entries)` without validated current-run zero-document evidence. Activity monitor для enrichment игнорирует draft files, существовавшие до старта focused call, пока provider не сделает fresh mutation; stale bootstrap files не считаются `artifact_observed` progress для post-artifact stall window. No-op enrichment invalid: если любой referenced markdown после enrichment не изменился или содержит scaffold/recovery markers, результат — `runtime_contract_failed` с причиной `draft_artifact_enrichment_noop_or_scaffold`; if every markdown target changed and the only remaining failure is marker/process/downstream wording, runtime may do one provider-authored `draft_artifact_enrichment_marker_cleanup` retry that rewrites all markdown targets again. Normal `step0.constitution` prompt still begins with `FIRST CONSTITUTION DRAFT COMMAND`, which writes `constitution-draft.json` + `charter-overview.md`/`baseline-subagents.yaml`; only `charter-overview.md` must be immediately rewritten from constitution evidence before success. Normal `step2.asis_docs` and `step4.proposals` prompts now begin with evidence-first `FIRST AS-IS DRAFT COMMAND` / `FIRST PROPOSALS DRAFT COMMAND`: the first filesystem work unit reads bounded current-run staged evidence, writes all referenced markdown targets validation-ready first, and writes the draft manifest last before returning. A manifest-only first write before markdown is invalid prompt behavior and may be classified as pre-artifact stall/repair pressure. Shared runtime draft validation rejects referenced `step0/step2/step4` draft files if they remain unchanged first-action/recovery scaffold (`Draft surface initialized`, `Current run evidence should be reviewed`, `Runtime proposal surface initialized`, `Drafted required runtime artifacts`, etc.) or describe runtime recovery mechanics as final content, so adapter valid-artifact stop cannot accept such a set as final; downstream quality telemetry additionally records promoted placeholder/scaffold artifacts through `artifact_quality:*`; ACP does not synthesize validator verdicts or draft manifests/files.
- Normal `step0.constitution`, `step2.asis_docs`, and `step4.proposals` prompts carry a same-turn completion requirement. For step0, the provider writes the initial draft set and immediately rewrites `charter-overview.md` from repository-entrypoint evidence before success. For step2/step4, the first filesystem work unit itself is evidence-first: bounded read/list first, then write validation-ready markdown targets, then write the manifest last before any final/status/analysis-only response. For step2, the first provider item must be command execution rather than assistant/status prose. Headless draft adapters use bounded first-command pre-artifact windows, including 180s for Claude/Qwen/Codex normal draft steps; tripping that window is still runtime pressure, not an accepted excellent path. A successful normal happy path should not require focused repair/stall pressure.
- `runtime_quality.stall_pressure` describes actual runtime pressure before a valid artifact exists or while artifacts are invalid/no-fresh/retry/analysis-only. A provider lifecycle stall after `artifact_state=valid` / `artifact_valid=true` and without validation error is recorded as additive `valid_artifact_controlled_stop(s)` telemetry, not as `stall_pressure`, and does not block `Excellent` by itself.
- `draft_artifact_enrichment` должен сохранять текущую shape `outputs[]`: `path` и `canonical_path` нельзя переименовывать, пересобирать из aliases или заменять на `logical_path`/`target`/`output_path`. Если provider обновляет draft manifest, допустимы только top-level `summary`/`updated_at`; unknown top-level/output fields (`status`, `content_digest`, `enriched_at`, `metadata`, `validation`, `confidence`, `source`, provider aliases) остаются strict `runtime_contract_failed`.
- Если provider-authored `draft_artifact_enrichment` fresh rewrite-нул evidence-backed markdown, но strict validation отвергла draft manifest из-за unknown fields / manifest shape drift, shared runtime может выполнить ровно один provider-authored retry `draft_artifact_enrichment_manifest_shape`. Retry prompt требует восстановить strict manifest shape and allowed keys while keeping/fresh-overwriting markdown targets; это не parser relaxation и не deterministic ACP-side manifest rewrite. Повторный manifest drift остаётся terminal `runtime_contract_failed`.
- Если provider-authored `draft_artifact_enrichment` fresh rewrite-нул все markdown targets, но strict validation отвергла результат только из-за malformed markdown inline-code/code-fence syntax, shared runtime может выполнить ровно один дополнительный provider-authored retry `draft_artifact_enrichment_markdown_syntax`. Retry prompt сохраняет evidence-backed meaning, но требует переписать все referenced markdown targets с balanced backticks/fences, без raw/truncated shard snippets, без sampled first-paragraph dumps after evidence paths, без stale downstream-index claims in step2 and без generic shard-gap wording when typed shard-summary shows all shards succeeded. Это не deterministic synthesis: если retry оставляет malformed markdown, scaffold/noop content или invalid manifest, итог остаётся `runtime_contract_failed`.
- For `step4.proposals`, markdown-syntax retry must preserve bullet-only actionable finding format and must not introduce markdown tables; table-based high/medium actionable findings remain validation failures, not a formatting escape hatch.
- Если provider-authored `draft_artifact_enrichment` не сделал fresh markdown mutation because its filesystem command invoked missing `python` (`python: command not found` / `command not found: python`), shared runtime может выполнить ровно один дополнительный provider-authored retry `draft_artifact_enrichment_python3_retry`. Enrichment prompt requires `python3`, not `python`; retry не является deterministic synthesis и не ослабляет contract: success still requires marker-free fresh rewrites for every referenced markdown target under `draft_final_root`, while repeated missing-interpreter/noop/scaffold output remains `runtime_contract_failed`.
- Если provider-authored `draft_artifact_enrichment` завершился/stalled without a fresh valid rewrite and validation still reports `draft_artifact_enrichment_noop_or_scaffold`, shared runtime immediately exhausts focused draft enrichment as `runtime_contract_failed` and emits `repair_exhausted`. Это не deterministic synthesis and not a quality bypass: ACP does not run a generic no-action enrichment retry and does not synthesize draft markdown.
- Narrow exception: if the failed enrichment was a silent pre-artifact stall with empty stdout/stderr and no fresh markdown mutation, shared runtime may do exactly one provider-authored `draft_artifact_enrichment_write_first_retry`. This is still not a generic no-action/status retry: success requires the provider to execute one bounded filesystem command and fresh-overwrite every referenced markdown target with marker-free evidence-backed content.
- Narrow step2-only exception: if `init|refresh.step2.asis_docs` then repeats the same silent pre-artifact no-fresh failure during `draft_artifact_enrichment_write_first_retry`, shared runtime may do one provider-authored `draft_artifact_enrichment_compact_step2_retry`. The compact retry prompt limits the evidence set to the current draft manifest, current-run typed shard-plan/shard-summary, observed shard manifest counts, at most three authored shard docs/manifests, and current-run final/citation indexes only when present. It still must fresh-overwrite `overview.md`, `summary.md`, and `architect-summary.md` under `draft_final_root`; evidence bullets must be path plus paraphrased architecture signal rather than raw shard markdown excerpts. Repeated silent/noop/scaffold/invalid compact output remains `runtime_contract_failed`, and ACP still does not synthesize draft markdown.
- Narrow step4-only exception: if `init|refresh.step4.proposals` then repeats the same silent pre-artifact no-fresh failure during `draft_artifact_enrichment_write_first_retry`, shared runtime may do one provider-authored `draft_artifact_enrichment_compact_step4_retry`. The compact retry prompt limits the evidence set to the current proposals draft manifest, current-run typed shard-plan/shard-summary, validator/finding/proposal/coverage summaries, current-run final/citation indexes when present, and at most three staged shard docs/manifests. It still must fresh-overwrite `proposal.md` and `changelog.md` under `draft_final_root`; repeated silent/noop/scaffold/invalid compact output remains `runtime_contract_failed`, and ACP still does not synthesize proposal/changelog markdown.
- Если provider-authored `draft_artifact_enrichment` возвращает shell/Python command text (например `python3 - <<...` или fenced bash/python) без fresh filesystem mutation, shared runtime может выполнить ровно один provider-authored `draft_artifact_enrichment_command_text_retry`. Retry существует только для принудительного real command execution после printed-command output; success still requires marker-free fresh rewrites, and repeated printed-command/noop/scaffold output remains terminal `runtime_contract_failed`.
- Если provider-authored `draft_artifact_enrichment` уже fresh rewrite-нул evidence-backed markdown, но strict validation отвергла результат только из-за stale downstream final/citation-index availability claim или unvalidated zero-document final-index claim, shared runtime может выполнить ровно один provider-authored `draft_artifact_enrichment_downstream_index_retry`. Retry должен удалить premature index-availability sentence для `step2` or recount present current-run `final-run-index.json.canonical_documents[]` and `citation-index.json.citations[]`; повторный stale claim остаётся terminal `runtime_contract_failed`.
- Если provider-authored `draft_artifact_enrichment` уже fresh rewrite-нул evidence-backed markdown, но strict validation отвергла output только из-за generic conditional shard-gap wording (`any failed or incomplete shards`, `failed shards require rerun`, `if present above`) или missing exact current-run shard completeness despite readable current-run typed shard status, shared runtime может выполнить ровно один provider-authored `draft_artifact_enrichment_shard_status_cleanup` retry. Этот retry запрещен для mixed `draft_artifact_enrichment_noop_or_scaffold` failures: если markdown не был fresh-overwritten and still bootstrap/noop, runtime сначала использует silent write-first/compact no-fresh path or exhausts as `runtime_contract_failed`, not shard-status cleanup. Для step2 retry должен переписать все step2 markdown targets, especially the validation-flagged target (`summary.md`, `architect-summary.md` or `overview.md`), с exact planned/succeeded/failed/incomplete counts from current-run typed `items[].status` when visible and an explicit no-shard-coverage-blocker statement when all shards succeeded. Для step4 retry должен переписать `proposal.md` and `changelog.md`, preserve exact finding IDs/actionability, and when all current-run shards succeeded both files must contain the exact literal `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` plus explicit `no-shard-coverage-blocker`. Повторный generic/missing exact completeness wording remains terminal `runtime_contract_failed`.
- Если provider-authored `draft_artifact_enrichment` уже fresh rewrite-нул все referenced markdown targets, но strict validation отвергла результат только из-за process/scaffold/downstream wording (`bounded read roots`, `current draft manifest`, `validator output`, step0 `later passes`/pipeline wording), shared runtime может выполнить ровно один provider-authored `draft_artifact_enrichment_marker_cleanup`. Retry prompt требует переписать все markdown targets again, not only the flagged line; repeated marker contamination remains terminal `runtime_contract_failed`.
- Если provider-authored `draft_artifact_enrichment` wrote otherwise valid referenced draft markdown into `write_root` as byte-identical misplaced duplicates of files under `draft_final_root`, shared runtime может выполнить ровно один provider-authored `draft_artifact_enrichment_write_set_cleanup` retry. Cleanup may delete only those duplicate referenced markdown files from `write_root`; the step draft manifest stays in `write_root`, draft markdown stays under `draft_final_root`, and arbitrary extra writes such as `extra.md` or unreferenced draft files remain immediate `runtime_contract_failed`. For `init.step0.constitution`, the same retry may delete only the mistaken `draft_final_root/skills/subagents.yaml` canonical-path duplicate while preserving `draft_final_root/baseline-subagents.yaml`; `outputs[].canonical_path` is publish metadata, not an allowed draft write path. This is not deterministic synthesis: success still requires existing/provider-authored valid draft artifacts plus provider-authored cleanup mutation.
- collect prompt для root-file shard (`path_scopes` содержит root-level files вроде `README.md`, `pom.xml`, `Makefile`) явно ограничивает чтение перечисленными файлами, выбирает primary evidence в порядке `README` -> `Makefile` -> build/deploy manifests -> прочие root files, и требует один evidence-backed root overview doc без bootstrap marker + enriched `shard-pack-manifest.json` без recursive repo sweep.
- collect repair read surface уже collect: текущий `write_root` + repo evidence roots. Workspace-level `reports/taskruns`, sibling shard manifests, raw logs, archive docs, examples и filesystem schema scavenging не являются допустимым repair input; embedded prompt contract/schema fragment is authoritative for the repair attempt
- collect repair prompt contract explicitly carries the canonical semantic shape because live providers may drift to older manifest aliases. `semantic.coverage.notes` must be an array, `semantic.entities[]` must use `id/name/type/provenance` with evidence under `provenance.evidence[]`, `semantic.edges[]` must use `id/type/from/to/provenance` rather than `relation/source/target`, and every semantic provenance object must include `kind`, numeric `confidence`, and `evidence[]`. Direct entity-level `repo/path/evidence`, direct `{"repo":"...","path":"..."}` provenance, `evidence_citation_ids`, finding-level `confidence`, and `summary` as a finding alias are execution contract drift. If pair repair writes this legacy shape and manifest-only repair cannot replace it with canonical shape, the collect shard remains `runtime_contract_failed`.
- collect pair recovery prompt is write-first/evidence-bounded and has no command-first heredoc seed. Pair repair runs with a fresh-mutation threshold and default 5-minute pre-artifact, post-artifact and partial-artifact windows; stale invalid markdown/manifest files do not count as repair progress, and slow live providers still have a bounded window to perform the first write. A silent no-fresh pre-artifact pair-repair stall can schedule one focused retry when the provider policy allows zero-output repair retry; the retry must still produce provider-authored markdown+manifest and pass strict validation. It names the exact suggested or existing authored doc + `shard-pack-manifest.json` targets, the exact read roots and ranked/listed evidence candidates, the task-specific manifest skeleton as schema guide only, banned recovery/scaffold markers, and a final self-check. Provider must not run a separate read-only preflight or emit plan/status/analysis-only prose such as “I have enough evidence” or “I am now writing”: the next item must be one bounded filesystem command that reads at most 8 listed evidence files, at most the first 6000 bytes from each file, and writes markdown plus `shard-pack-manifest.json` before returning. When pair repair is triggered because existing markdown cites a missing repo path, the prompt targets that existing markdown and requires a complete rewrite from observed evidence; leaving stale missing-path claims unchanged is terminal. When validation reports directory-only repo evidence refs, process-contaminated markdown, or a no-artifact stall, pair repair includes a validation-specific first-objective block to replace directory citations/provenance with concrete existing files, rewrite contaminated markdown before manifest repair, or write both targets in the first command. Pair repair evidence candidates are capped and ranked toward README/AGENTS/build/deploy/config signals, and exclude lockfiles, generated baselines, test duration indexes, raw logs, taskrun history, and full reads of files larger than 96 KiB. Pair repair must not add provider-invented exact phrase gates (`required = [...]`, `missing expected evidence`, marketing-copy substring checks) before writing: pre-write checks are limited to target containment and at least one allowed evidence file yielding bytes after bounded prefix reads. Oversized candidates are truncated to the first 6000 bytes or skipped while the repair continues; `read file exceeds size limit` is not a valid terminal reason when other evidence is available. Unsupported planned claims are omitted or recorded as coverage gaps. Pair repair also must not add semantic pre-write exits for entity/edge/finding/citation counts or generated text shape; backend validation is the semantic gate. The first command should be mechanically simple: no Python f-strings, no `.format(...)` templates, no generated Python source strings, and no nested quote tricks for markdown/JSON assembly. Seed-only/recovery fallback prose is not an allowed intermediate success path. Marker-bearing docs, unchanged normal seed docs, `Recovery Bootstrap`/`Recovery Summary` scaffold and marker-free `Recovery Evidence Summary` fallback prose reject-ятся тем же collect validation, чтобы short valid-artifact stop не закреплял recovery-only docs. Invalid observed artifacts without fresh mutation are stopped by the bounded partial-artifact window even when stdout/stderr remains active; no-artifact write-first commands are stopped by the absolute pre-artifact wall-clock cap even when stdout/stderr remains active. Manifest-only repair prompt starts with `COLLECT MANIFEST EVIDENCE-FIRST REPAIR` and `FIRST COLLECT MANIFEST REPAIR COMMAND`; the first provider-authored filesystem command must read bounded authored docs/evidence and write the single allowed target `write_root/shard-pack-manifest.json` before returning. Evidence-packet-only output and status prose like “I have enough evidence / I am replacing” are no-op repair failures unless a valid manifest was actually written. Task-specific JSON skeleton остаётся schema/key/type guide, а copied placeholder/skeleton content and scaffold semantic invalid. Evidence-rich authored docs require concrete semantic extraction: named entities beyond repo/shard wrappers, non-contains relationships, and findings/questions beyond generic owner mapping when stack/deploy/config/test/security evidence exists; citations-only enrichment around generic semantic remains invalid. Root-file repair evidence использует тот же `README` -> `Makefile` -> build/deploy preference, validation-error details не повторяются как field-level patch cues, а success определяется backend validation, не stdout claim. Если manifest отсутствует после non-bootstrap authored docs, existing manifest failed only because semantic is scaffold-only, or clean manifest failed only because citation `claim_ids` are empty/missing, runtime применяет deterministic `collect_manifest_runtime_recovery` before provider repair; для other structural invalid existing manifests и для случаев, где deterministic recovery не валидируется, provider repair остается next fallback, кроме missing repo evidence path, still present in authored markdown, which must use pair repair. Exhausted/invalid provider repair после structural-invalid manifest является terminal `runtime_contract_failed`, а не runtime-recovery success. Recovered manifest derives from existing provider-authored markdown, uses concrete service/component/datastore entities with usage/dependency/configuration edges plus explicit runtime-recovery finding/question/coverage note, remains under the same write-set guard, emits recovery diagnostics/runtime warning so it cannot masquerade as normal provider-authored success, and is still accepted only through strict backend validation and fresh live quality gates.
- Empty provider-authored collect markdown is not a valid input for manifest-only repair. If a shard leaves whitespace-only markdown or an empty `shard-pack-manifest.json`, runtime routes to provider-authored `collect_pair_repair` with mandatory fresh markdown+manifest rewrite before any manifest-only repair can run. A non-empty provider-authored markdown doc with a missing/empty manifest may still use deterministic `collect_manifest_runtime_recovery`, but only from that existing markdown and only through the same strict collect validation and telemetry warning.
- Digest-only or transcript-only partial collect output is also not a valid input for manifest-only repair. If `write_root` contains non-contract files such as `collect-digest.txt` but no authored `.md`/`.markdown` collect document, runtime routes to provider-authored `collect_pair_repair` with `missing_authored_markdown` stage and requires a fresh evidence-backed markdown+manifest pair. `collect_manifest_runtime_recovery` may only derive a manifest from existing non-bootstrap, process-clean authored markdown; it must not treat non-markdown digests as hidden authored docs.
- Compact collect pair repair manifests must bind every citation back to authored documents and claims: each `citations[]` item must include non-empty `claim_ids` plus non-empty `document_ids` referencing `documents[].id`. If process-contaminated existing markdown was freshly rewritten but the pair repair leaves only manifest schema/semantic errors such as missing `document_ids`, shared runtime may chain into manifest-only repair; if the only remaining binding error is empty/missing `claim_ids`, runtime may reconstruct the manifest via `collect_manifest_runtime_recovery`. These fallbacks are allowed only after the required markdown target changed and is no longer process/bootstrap/stale-missing-evidence contaminated.
- Interrupted temporary collect markdown is contract-invalid even if the manifest semantic snapshot is otherwise rich: phrases such as `first bounded evidence read was attempted`, `initial artifact records only`, and `will be repaired with concrete...` must fail strict collect validation instead of being passed downstream as complete shard evidence.
- Process-contaminated collect markdown is also contract-invalid: final collect docs must not narrate bounded reads/passes, guessed paths/files/evidence, expected-missing concrete path checks, recovery attempts, or later repair. Missing expected concrete files such as `src/foo/Bar.java path was not present` are process contamination, and wording like `not examined in this bounded pass` is still runtime-process narration; write operator-facing gaps as `not confirmed in scoped repository evidence` or as a concrete missing evidence category. A decision-ready coverage gap that a named spec/scope was not observed is valid when it does not cite guessed evidence. When existing authored markdown has this contamination, runtime routes to provider-authored collect pair repair with mandatory fresh rewrite of the same markdown target; manifest-only runtime recovery must not turn that document into hidden deterministic success.
- markdown-only collect completion невалиден: hard pass невозможен, пока каждый required shard не имеет валидный `shard-pack-manifest.json` после разрешённых collect recovery попыток
- для `step1.collect` runtime использует selected repo roots, `write_root` и explicit `read_context_roots`; `reports/taskruns/**`, raw logs и archive docs не считаются schema source of truth и не должны использоваться как manifest examples. Manifest-only repair дополнительно исключает broader workspace read scope, чтобы sibling shard manifests не становились неявными examples
- если `step1.collect` завершился partial (`best_effort` succeeded+failed shards), downstream live draft/validator/proposal runtime не должен запускаться как будто collect complete. Orchestrator пишет incomplete docflow из persisted shard evidence, фиксирует `collect_partial_shard_failures` плюс `*_skipped_due_to_partial_collect`, а terminal run surface остается `run_partial_failed`. Sequential `best_effort` collect stops dispatch after five consecutive `runner_unavailable` shard failures and marks undispatched shards failed with the same provider class, so reports preserve the collect-level outage instead of masking it as downstream incompleteness. Batch/report layer keeps `runtime_flow_failed`/`partial_failure_count` as explicit execution signals, but primary `failure_class` must use concrete terminal/provider class (`runner_unavailable` or `runtime_contract_failed`) when the classifier or failed shard diagnostics provide one. If a provider leaves malformed semantic JSON in a collect artifact path, reporting must record `analysis:malformed-semantic-json` evidence and still emit `run_matrix_*`/`execution_report_*`; the malformed artifact remains artifact-quality telemetry unless the runtime itself failed the execution contract.
- root-marker-only repos (например только `pyproject.toml`/`pom.xml` в корне) не планируются как единый shard `"."`, если top-level структура большая: planner строит root-file group + top-level directory shards, затем применяет существующий cap/coalescing и сохраняет marker leaves where possible
- coalesced shard IDs are stable filesystem-safe bounded slugs; overlong root-file groups keep a readable prefix plus hash suffix so runtime artifact directories remain portable across local filesystems
- выполняет runtime `init.step1.collect`/`refresh.step1.collect` отдельно для каждой canonical domain card (`charter/cards/domains/*`)
- materialize-ит отдельный `runtime-execution.json` на каждый shard/domain under `reports/taskruns/<run_id>/...`
- для sharded runtime ведёт shard-summary state machine `pending | checkpointed | succeeded | failed`; raw per-shard taskrun materialize-ится до `apply`, чтобы restart recovery мог replay-ить shard из persisted artifact
- internal shard-summary contract: `taskrun_path` обязателен для `checkpointed/succeeded` item и должен ссылаться на persisted `runtime-execution.json`
- materialize-ит raw runtime execution metadata и shard summaries в `reports/taskruns/*`
- извлекает semantic snapshot из manifests/verdicts для derived `model/*`/semantic guards
- обновляет derived `model/*`
- enrich существующие `charter/cards/domains/*` и `charter/cards/teams/*` через детерминированную секцию `## Derived (ACP Step1)`:
  - related model IDs / findings / questions
  - coverage missing summary
  - evidence refs (для domain и team cards)
- не создаёт и не переименовывает canonical cards автоматически
- сохраняет outputs domain-агентов в `reports/agent-outputs/domains/*`
- не изменяет canonical `charter/cards/*` напрямую; допускается только controlled enrichment секции `## Derived (ACP Step1)`

### Step 2 — As-is docs (aggregator staged assembly)
Inputs:
- collected shard manifests + authored shard docs
- `charter/*`
- `skills/templates/*`

Outputs:
- staged final docs в `reports/taskruns/<run_id>/staging/final/`
- `final-run-index.json`
- `citation-index.json`
- staged derived `model/*` для diagram builders и deterministic projections; C4/Mermaid compiler обязан дедуплицировать sanitized node ids и generated diagram artifact slugs, чтобы разные model entity ids вроде `svc.foo-bar` и `svc.foo.bar` не перезаписывали друг друга и не создавали ambiguous duplicate Mermaid nodes
- если live runtime step выполняется, его primary output — `asis-draft-manifest.json` + optional draft docs внутри `draft_final_root`

Step 2 policy:
- domain outputs и финальные analysis/proposal surfaces собираются как authored/staged docs-first set из shard packs
- orchestrator compiler допускается только как deterministic renderer/materializer для technical derived surfaces
- `model/*` и диаграммы остаются derived layer
- `step2.asis_docs` не имеет отдельного editable workspace prompt pack; live runtime получает enforced step policy + artifact-only contract, а не `skills/prompt-packs/as-is.md`
- `asis-draft-manifest.json` обязан использовать strict canonical contract: `version=1`, `run_id`, `step_id`, `step_contract="as_is"`, `agent_role`, `outputs[]`; optional top-level metadata допускается только как `summary`/`updated_at`
- `asis-draft-manifest.json.outputs[]` item shape ограничен `path`, `canonical_path`, optional `kind`, optional `title`; `logical_path` и другие aliases запрещены strict parser-ом.
- для `step2.asis_docs` обязательны canonical publish mappings `reports/as-is/overview.md`, `reports/coverage/summary.md`, `reports/agent-outputs/architect/summary.md`; дополнительные outputs разрешены только под `reports/as-is/<domain>/overview.md`
- normal и focused repair prompts для `step2.asis_docs` начинают task surface с `FIRST AS-IS DRAFT COMMAND`; команда обязана создать `asis-draft-manifest.json` в `write_root` и все три referenced draft files в `draft_final_root` до broad as-is assembly
- runtime draft parsing для `step2` strict: unknown top-level fields и legacy envelopes (`repo_scopes`, `compatibility`, `generated_at`, `step_contract=null`, неканонические coverage outputs) reject-ятся до publish; harmless `updated_at` metadata не участвует в publish/quality verdict
- runtime draft validation для `step2` проверяет не только shape manifest и наличие referenced files, но и отсутствие unchanged first-action scaffold text в `overview.md`, `summary.md` и `architect-summary.md`; такой bootstrap остаётся invalid до provider-authored evidence-backed rewrite
- runtime task validation для canonical `reports/as-is/overview.md` разрешает каждую operator-visible inline-code ссылку `<repo>:<path>` только когда `<repo>` является current task repo scope, а `<path>` существует как файл или директория внутри resolved repo root после containment/symlink проверки; missing, absolute, traversal и symlink-escaped paths являются `runtime_contract_failed` до promotion. Generic manifest contract не получает repo roots и не меняет публичную shape; runtime adapter применяет дополнительную read-only truthfulness check из task context.
- `draft_artifact_enrichment` для `step2` является command-first/write-first focused recovery: first filesystem action выполняет bounded command, который читает текущий `asis-draft-manifest.json`, current-run typed shard status evidence (`reports/taskruns/<run_id>-*-step1-collect-shard-plan*.json` / `reports/taskruns/<run_id>-*-step1-collect-shard-summary*.json` when present), bounded current-taskrun staged evidence (`shard-pack-manifest.json` summaries, optional current-run `final-run-index.json`/`citation-index.json` and at most 6 high-signal shard manifests/authored docs) и в той же command до optional extra analysis fresh overwrite-ит все три exact markdown targets под `draft_final_root`: `overview.md`, `summary.md`, `architect-summary.md`. `overview.md` должен описывать architecture surface с repo/path/staged refs и coverage gaps; `summary.md` — planned/succeeded/failed shard completeness, evidence density/readability и gaps; completeness counts must come from typed shard-plan/shard-summary data when visible, including shard-summary `items[].status` (`planned=len(items)`, `succeeded=count(status=="succeeded")`, `failed=count(status=="failed")`), or from observed shard dirs plus `shard-pack-manifest.json` counts with explicit unknowns. Lexical counts of words like `failed`/`error` in manifests are misleading output and must not be used. If exact current-run status says failed=0 and no incomplete statuses, markdown must state no failed/incomplete shards rather than generic conditional caveats and must not publish false empty-shard claims such as `Shard pack manifests: none observed` when typed summaries or shard manifests are visible. Current target identity comes from `repo_scope`/`repo_scopes`/`domain_id`, not matrix/profile/batch names or workspace paths, and markdown must not cite foreign live `run_*` taskrun ids as current evidence. `architect-summary.md` — decision-ready operator summary with complete/missing/next inspection or decision. Provider не должен отвечать analysis-only фразами вроде “I have enough evidence” без filesystem mutation, не должен читать весь shard corpus и не должен останавливаться после перезаписи одного файла; unchanged files, scaffold markers, foreign run-id evidence, generic shard-gap wording, or context-overflow caused by broad workspace/repo reads remain runtime contract/recovery bugs, not artifact-quality-only warnings.
- `draft_artifact_enrichment` для `step2` должен сохранять `asis-draft-manifest.json.outputs[]` mappings; provider не должен добавлять `logical_path` или пересобирать manifest как final/citation index. Default acceptance по-прежнему требует fresh rewrite всех referenced markdown. Единственное targeted исключение действует для exact `outputs[0].path "overview.md" Architecture Home contains runtime/process narration, manifest recap, or unsupported confidence language`: fresh rewrite только `overview.md` принимается лишь после полной strict validation manifest и всех outputs; unchanged valid `summary.md`/`architect-summary.md` не переписываются ради искусственного byte change.
- focused draft recovery may remove only newly created regular zero-byte sidecar files inside `draft_final_root` that were absent from the pre-command snapshot. This is rollback of a forbidden mutation, not an allowed output: runtime immediately revalidates the full write-set and strict draft contract. Any non-empty, modified, deleted, directory, symlink, `write_root`, traversal or out-of-root mutation remains `runtime_contract_failed`.
- Если `step2` enrichment читает current-run final/citation indexes, document counts must come from top-level `final-run-index.json.canonical_documents[]` and citation counts from top-level `citation-index.json.citations[]`; missing `documents[]`, `checked_paths[]`, or validator checked paths are not evidence for zero current-run documents.
- `draft_artifact_enrichment` для `step4.proposals` обязан выполнять write-first sequence: прочитать `proposals-draft-manifest.json`, current-run typed shard-plan/shard-summary files when present, current-run staged `reports/taskruns/<run_id>/staging/final/reports/findings/findings.md`, `reports/taskruns/<run_id>/staging/final/reports/coverage/summary.md`, `final-run-index.json`/`citation-index.json`, validator/finding summaries и не более 6 high-signal shard manifests/docs, затем fresh overwrite-нуть `proposal.md` и `changelog.md` под `draft_final_root` до optional extra analysis. До publish current-run findings/coverage не живут под `reports/taskruns/<run_id>/reports/*`; provider must not use that path as evidence for zero findings. `proposal.md` должен содержать non-empty recommended operator action, evidence refs, proposed changes/follow-up plan и risks/gaps/out-of-scope; `changelog.md` должен содержать non-empty touched architecture/proposal surfaces, findings/proposals summary, evidence index/citation refs и residual coverage gaps. Если current-run `findings.md` содержит finding IDs, proposal/changelog markdown must cite current-run finding IDs and must not claim structured findings are absent; for medium/high findings, proposal.md must include severity summary, top actionable findings, affected surfaces/paths, recommended operator action and residual gaps. Provider must not report `0` authored markdown shard docs unless it actually globbed `staging/shards/**/*.md` in the allowed roots and found none, must not cite a different `run_*` taskrun id as proposal evidence, and must not ask operators to repair non-succeeded shards when current-run typed status is all-succeeded. All-succeeded typed status must be written as exact planned/succeeded/failed/incomplete counts plus an explicit no shard-coverage blocker in both proposal/changelog. Generic conditional phrases such as `failed shards require rerun`, non-negated `failed or incomplete shards remain`, stale index claims such as `No current-run final-run-index document list was available`, dangling references such as `prioritize each finding above` / `findings above`, empty findings/gaps sections, generic action-only plans, `No structured finding summary was present` when current-run findings are non-empty, `replaced placeholder proposal content`, `replace placeholder content`, or `replacing placeholders` are runtime scaffold/low-signal contamination even when repo/path evidence is present; domain statements such as replacing placeholder credentials are allowed when they cite concrete current-run findings/repo paths. An explicitly negated line such as `No failed or incomplete shards remain as a coverage blocker in the current-run typed shard summary` is accepted only when tied to exact current-run status, and an explicit no-actionable-proposal gap is accepted only when it appears in the findings/proposals summary and proposed plan with current-run evidence. The draft manifest summary and output metadata are not proposal evidence; copying bootstrap phrases such as `Drafted required runtime artifacts for this step` into markdown remains scaffold contamination. Если любой target unchanged или scaffold-only, runtime возвращает `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`; ACP не синтезирует proposal/changelog как hidden success path.
- For high/medium current-run findings, `proposal.md` must use a markdown-safe bullet-only `Top Actionable Findings` section. Each actionable bullet represents exactly one finding and must keep exact Finding ID, the copied Severity value from that finding block, Affected surface/path from Related IDs or Evidence, Recommended operator action using a concrete verb such as `update`, `add`, `document`, `assign`, or `remediate`, and Residual gap on the same bullet line. Splitting one finding into a Finding ID bullet followed by a Description/action bullet is invalid. `Severity: unspecified` is invalid when the referenced finding has a `- Severity:` field. The current-run findings file is nested at `staging/final/reports/findings/findings.md`; providers must not shorten it to `staging/final/reports/findings.md`, must parse backticked `- ID: ` lines, and must not emit synthetic placeholders such as `no-current-run-finding-id`, `no structured current-run finding ID`, or `finding unavailable`. If current-run findings are non-empty, `Finding ID: none`, `Finding ID: n/a`, `Finding ID: unavailable`, or equivalent placeholder finding IDs in any actionable bullet are strict draft validation failures even when an exact finding ID appears elsewhere in the proposal. Markdown tables, malformed inline-code/fence syntax, low-only citations when high/medium findings exist, synthetic finding placeholders, and generic `inspect`/`review`/`decide`-only actionability are strict draft validation failures before promotion.
- `step4.proposals` enrichment must count indexed documents from top-level `final-run-index.json.canonical_documents[]` and citations from top-level `citation-index.json.citations[]`; providers must not infer `0 observed document entries` from missing `documents[]`, `checked_paths[]`, or validator checked paths. If proposal/changelog markdown states final/citation index counts while the current-run index files are present, strict draft validation rejects mismatched counts. Metadata-only JSON keys such as `"version": 1`, `"run_id"`, `"pipeline"`, `"generated_at"`, or `"citation_index_path"` are not high-signal evidence bullets and are rejected in operator-facing draft markdown.
- `final-run-index.json` и `citation-index.json` используют один deterministic `document_id` mapping. Unique `manifest.Documents[*].id` values are preserved; when multiple distinct `canonical_path` values reuse the same provider-authored document id, orchestrator remaps those documents to stable canonical-path-derived ids before `final-run-index.json` validation and remaps citation `document_ids` into the same canonical namespace. Staged validation then checks the remapped indexes together: every citation document id must exist in `final-run-index.json.canonical_documents[]`, every final document citation id must exist in `citation-index.json.citations[]`, and the links must stay reciprocal after remap.
- staged semantic assembly нормализует `evidence.repo` к логическому repo scope, сводит generated checkout-dir aliases и дедуплицирует entity aliases/related references до validator
- derived `model/entities/*.yaml` и `model/edges/*.yaml` используют deterministic bounded filenames; при длинном canonical id filename обрезается с hash suffix, а полный `id` сохраняется внутри YAML
- если evidence incomplete, staged reports materialize-ятся с incomplete banner, но не promote-ятся без validator `PASS`
- run quality summary (`reports/taskruns/<run_id>-quality.json`) строит fresh artifact inventory по текущему promoted workspace + `reports/taskruns/<run_id>/staging/**`: expected/produced surfaces, final semantic counts, missing model files, placeholder reports/proposals, gap-only C4, empty findings при critical coverage gaps, proposals/findings disconnect (`artifact_quality.proposals_findings_disconnected`), low-actionability proposals for medium/high findings (`artifact_quality.proposals_low_actionability`) и hidden provider/tool document refs фиксируются как `artifact_quality:*` signals для succeeded normal runs. C4 `Context` считается gap-only blocker, если semantic model non-empty, но диаграмма не смогла показать ни external/team relation, ни bounded evidence-backed internal service/datastore relation.
- если collect status = `unusable`, live runtime для `step2.asis_docs`, `step3.findings` и `step4.proposals` не запускается; orchestrator детерминированно пересобирает triage-only staged docflow только из persisted collect artifacts, а terminal/root cause остаётся collect failure
- non-collect runtime шаги не стартуют из workspace root: `step2` использует `draft_final_root` как cwd, а live harness разводит headless и baseline workspaces по разным temp roots, чтобы sibling baseline artifacts не были implicit template source
- provider-side hard sandbox в текущих headless CLI отсутствует; practical isolation для runtime обеспечивается только через separated temp roots и step-local `cwd`

### Step 3 — Findings / Validation (runtime step)
Inputs:
- staged final docs
- `final-run-index.json`
- `citation-index.json`
- `charter/rules.yaml`
- `skills/*`

Primary runtime output:
- `validator-verdict.json`
- `validator-verdict.json` обязан содержать canonical metadata trio `version=1`, `run_id`, `generated_at` вместе с `verdict` и `checked_paths`
- validator findings сохраняют canonical `title + description + provenance`; observation evidence внутри `findings[*].provenance.evidence[]` обязано содержать `repo/path`

Orchestrator applies:
- для sharded runtime replay-ит persisted `succeeded/checkpointed` shard taskruns без повторного provider execution; `checkpointed` shard повторно `apply`-ится, `succeeded` shard только восстанавливает orchestrator in-memory state
- пересобирает staged final set после validator verdict
- валидирует `validator-verdict.json`
- блокирует promotion при verdict != `PASS` или при broken staged indexes
- validator может править только index/reference/technical issues внутри validator scope; смысл authored docs не переписывается wholesale
- duplicate `claim_id` внутри `citation-index.json` считаются validator-scope technical drift: orchestrator repair-ит поздние коллизии по правилу `<claim_id>.<shard_slug>[.<n>]` и фиксирует это в `validator-verdict.json.fixed_paths`
- обновляет `reports/findings/*`
- обновляет `reports/agent-outputs/architect/summary.md` через детерминированную агрегацию фактических domain outputs
- materializes critical unknowns как findings, если отсутствуют owner/integration/database/CI-CD evidence
- owner-gap остаётся surfaced через `coverage/findings/questions`, но owner-only residual без технических validator issues не должен сам по себе держать `validator-verdict = FAIL`; после repair stages orchestrator может детерминированно reconcile-ить такой verdict в `PASS`, не скрывая сам gap

### Step 4 — Promotion / Proposals
Inputs:
- staged final doc set
- `final-run-index.json`
- `citation-index.json`
- `validator-verdict.json`
- `skills/templates/*`

Outputs:
- `proposals/<proposal-id>/proposal.md`
- `proposals/<proposal-id>/ADR.md`
- `proposals/<proposal-id>/RFC.md`
- `proposals/<proposal-id>/migration-checklist.md`

> В MVP proposals формируются только через runtime-authored staged docs; compiler допускается только как deterministic renderer/materializer для derived technical outputs.

Runtime proposal contract:
- если live runtime step выполняется, primary output — `proposals-draft-manifest.json` + optional proposal/changelog drafts внутри `draft_final_root`;
- manifest обязан использовать strict canonical contract: `version=1`, `run_id`, `step_id`, `step_contract="proposals"`, `agent_role`, optional `summary`, `outputs[]`;
- `outputs[].path` relative только к `draft_final_root`, `outputs[].canonical_path` workspace-relative и unique;
- `outputs[]` допускает только `path`, `canonical_path`, optional `kind`, optional `title`; `logical_path` и другие aliases invalid.
- allowed publish surface для `outputs[].canonical_path`: только `proposals/*` и `reports/changelog/*`;
- normal provider prompt должен показывать command-first heredoc write-set (`FIRST PROPOSALS DRAFT COMMAND`) для exact `write_root/proposals-draft-manifest.json` и referenced files под `draft_final_root`; command block встречается один раз, поздние sections ссылаются на canonical shape без повторного heredoc;
- legacy/final-index-like envelopes (`pipeline`, `step`, `generated_at`, `domain_id`, `proposals[]`, `info_findings_noted`, `orphan_coverage_gaps`) являются contract drift и reject-ятся strict parser-ом до promotion;
- successful proposal/changelog draft outputs stage into `reports/taskruns/<run_id>/staging/final/` and are included in the same `final-run-index.json` before validator-gated promotion copies them to `proposals/*` and `reports/changelog/*`.

## Iteration changelog (MVP)
- На каждую итерацию orchestrator формирует:
  - `reports/changelog/<yyyy-mm-dd>-<iteration-id>.md`
- Changelog агрегирует изменения по model/findings/proposals/agent-outputs/coverage.

## Missing information handling (MVP)
- Runtime не должен выдумывать отсутствующие данные.
- Если evidence недостаточно для архитектуры, интеграции, базы данных, owner linkage или CI/CD:
  - добавляются `questions`,
  - заполняется `coverage.missing` / `coverage.notes`,
  - при необходимости создаются findings про критичные пробелы.
- Observation без evidence не должен порождаться.
- Unknown owner не создаёт auto-team entity и не создаёт auto-card; это question/finding path.

## On-demand Q&A capability (MVP)
- Target System Analyst Q&A capability работает как async runtime-backed run через API `POST /api/qa/runs` + `GET /api/qa/runs/<run_id>`:
  - step id: `qa.ask`
  - agent role: `system-analyst-qa`
  - prompt pack: `skills/prompt-packs/qa.md`
  - write scope: `reports/taskruns/<run_id>/qa/`
  - required runtime artifact: `qa-answer.json`
- Перед runtime invocation ACP собирает deterministic context pack из:
  - `charter/cards/*`
  - `model/*`
  - `reports/as-is/*`
  - `reports/findings/*`
  - `reports/coverage/*`
  - `proposals/*`
  - `reports/changelog/*`
  - configured `docs.imports_path`
- `reports/taskruns/**` is excluded from the evidence corpus by default.
- Runtime answer contract: `qa-answer.json` includes `version`, `run_id`, `question`, `answer`, `citations`, `unresolved`, `confidence`, `provider`, `generated_at`.
- Runtime validator additionally rejects citation paths that are not workspace-relative paths present in `context-pack.json` `documents[].path`; `citations` may be empty when evidence is insufficient.
- Current compatibility surfaces `acp qa` and public read-only `POST /api/qa/ask` remain deterministic workspace-backed service/fallback during migration.
- Канонический stakeholder статус/границы по Q&A API фиксируются в `docs/STAKEHOLDER_DOC.md` (Canonical Stakeholder Matrix).

## Нефункциональные требования (MVP)

- детерминированность на одинаковых входах
- безопасный filesystem scope (без выхода за workspace root)
- runtime предлагает, человек подтверждает спорные решения
- запись только в workspace, не в пользовательские репозитории
- git access использует локальный `git` контекст пользователя/runner, а не отдельный credential plane ACP
- один и тот же pipeline должен быть воспроизводим локально и в GitHub/GitLab CI/CD trigger mode

## CI/CD trigger modes (MVP)
- Required MVP integration surface: `acp run --workspace ... --pipeline ... --non-interactive`.
- SCM hook mode: GitHub/GitLab webhook инициирует native pipeline/job, который запускает ACP batch mode.
- Webhook принимает CI provider, не ACP: native SCM webhook listener / external SCM app integration остаются вне MVP.
- Default auto-trigger: `push` в default branch.
- `merge request` / `pull request` updates в MVP идут как manual/preview trigger, а не auto-write trigger.
- Manual trigger mode: пользователь запускает ту же job через manual pipeline button/job.
- Long-running standalone mode: при наличии поднятого ACP service внутренняя automation может вызывать тот же refresh flow через API/CLI без hosted control plane.
- Internal API trigger optional и допустим только для trusted local/private deployment.
- Debounce policy: одновременно активен только один run на workspace; события в окне 5 минут схлопываются, policy `last event wins`.
### Step 2 — As-Is docs (agent-first + compiler)

Inputs:
- persisted Step 1 shard manifests / authored docs
- semantic snapshot из collect artifacts

Primary runtime output:
- `asis-draft-manifest.json`
- optional draft docs inside `draft_final_root`

Publish policy:
- orchestrator/compiler нормализует layout, coverage/findings links и ordering;
- runtime draft docs не считаются опубликованным результатом до compile/publish stage;
- strict `asis-draft-manifest.json` contract обязателен для live runtime path: `step_contract="as_is"`, required overview/coverage/architect outputs и отсутствие legacy top-level fields;
- при resume после более позднего шага orchestrator может детерминированно пересобрать staged final docflow из persisted collect artifacts без live provider rerun.

### Step 3 — Findings / Validator

Primary runtime output:
- `validator-verdict.json`

Publish policy:
- validator остаётся единственным schema/semantic gate для staged final set;
- richer synthesis/ranking/evidence shaping разрешены, но итог проходит через validator verdict и compile checks.
- terminal `validator verdict is FAIL` трактуется как completed runtime flow failure, а не как draft/runtime-contract failure; owner-gap-only verdict после technical repairs может быть downgraded в `PASS` при сохранении findings/questions.

### Step 4 — Proposals (agent-first + auto-publish)

Primary runtime output:
- `proposals-draft-manifest.json`
- optional proposal/changelog drafts inside `draft_final_root`

Publish policy:
- strict `proposals-draft-manifest.json` contract обязателен для live runtime path: `version=1`, `run_id`, `step_id`, `step_contract="proposals"`, `agent_role`, optional `summary`/`updated_at`, `outputs[]`;
- `outputs[].path` relative только к `draft_final_root`, `outputs[].canonical_path` workspace-relative, unique и разрешён только под `proposals/*` или `reports/changelog/*`;
- legacy/final-index-like envelopes запрещены: top-level `pipeline`, `step`, `generated_at`, `domain_id`, `proposals[]`, `info_findings_noted`, `orphan_coverage_gaps` и output aliases вроде `logical_path` должны hard-fail-иться strict parser-ом; `updated_at` является единственным timestamp metadata key, разрешённым в runtime draft manifest;
- deterministic promoter проверяет schema/semantic/validator gates;
- обязательного human approve нет;
- canonical `proposals/*` и `reports/changelog/*` публикуются автоматически только после successful gates and are represented in `final-run-index.json`.
