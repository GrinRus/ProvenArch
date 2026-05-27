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
  - verdict `PASS|FAIL` с policy strict zero-failure.

## 2) Тестовая пирамида MVP

### Contract tests
- `workspace.yaml` валидируется по `schemas/workspace.schema.json`
- docs-first contracts валидируются по:
  - `schemas/shard-pack-manifest.schema.json`
  - `schemas/final-run-index.schema.json`
  - `schemas/citation-index.schema.json`
  - `schemas/validator-verdict.schema.json`
  - `schemas/qa-answer.schema.json`
- persisted `runtime-execution.json` metadata и artifact-only step contracts проходят parse/semantic validation
- examples и fixture cases должны парситься и проходить contract validation, где это ожидается

### Semantic validator tests
- правила, которые не выражаются чистой JSON Schema
- deterministic canonicalization top-level `questions/coverage`
- stable ID normalization и collision rules
- ownership/card linkage constraints

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
- fixture contract gate проверяет parse/semantics recorded artifacts (`meta.step_id`, `repo_scopes`)

### Smoke tests
- CLI smoke
- API smoke
- UI smoke

### Optional live-runner smoke
- только manual/opt-in
- не входит в required CI gates

### Headless provider conformance
- required tests используют stub provider adapters без live network dependencies
- общий process engine проверяется на success by valid artifacts, controlled stop after valid artifacts, qwen draft valid-artifact stop after continued stream/mutation, focused repair valid-artifact stop after provider overrun, collect pair recovery, collect manifest-only repair success/failure, transient provider API/transport failure retry during qwen collect-pair and draft-artifact repair, validator-verdict-only repair, draft-artifact repair, bounded pre-artifact and repair stall windows, qwen recovered zero-output pre-artifact retry warning, scoped Claude constitution/collect/validator/proposals zero-output retry warning, exhausted silent/API no-artifact classification, invalid artifact contract failures, deadline timeout и raw stdout/stderr + redacted lifecycle diagnostics, включая resolved timeout profile
- provider-specific tests проверяют только adapter policy/args: `qwen` использует stream-json activity output без semantic stdout contract, не передаёт JSON task stdin при `-p` invocation, нормализует custom prompt args к artifact prompt, не отключает artifact/pre-artifact monitoring при custom args, включает bounded transient provider-unavailable focused repair retry for collect-pair/draft-artifact no-artifact cases и применяет bounded valid-artifact stop к normal draft steps; shared focused repair policy добавляет bounded valid-artifact stop для repair attempts независимо от provider; `claude` retry policy включает zero-output pre-artifact warning/retry только для constitution/collect/validator/proposals steps; `claude`/`codex` machine-mode flags остаются diagnostic transport mode
- prompt contract tests покрывают constitution command-first manifest+draft-file heredoc targets, валидный YAML first-action `baseline-subagents.yaml` для canonical `skills/subagents.yaml`, normal collect first-action запрет на `read_file`/repo exploration до first pair, deferred collect entrypoint hints, collect pair recovery command-first doc+manifest heredoc targets, collect manifest-only repair `FIRST COLLECT MANIFEST REPAIR COMMAND`, literal manifest JSON skeleton, as-is first-action `FIRST AS-IS DRAFT COMMAND` для manifest + `overview.md`/`summary.md`/`architect-summary.md`, validator first-action command-first verdict heredoc target, validator issue canonical shape/legacy bans, proposals first-action command-first manifest+draft-file heredoc targets, draft recovery command-first manifest+draft-file heredocs, exact root-variable targets that avoid retyping long slash-separated provider paths, and root-file shard hints без live network dependency
- batch preflight tests покрывают selected-provider readiness без live network dependency, включая codex `gpt-5.5`/CLI version mismatch guard и artifact smoke pass/fail/timeout paths

## 3) Обязательная структура test assets

- `fixtures/workspace/` — manifest и validator cases
- `examples/*.example.json` + contract tests — docs-first fixtures (manifest/index/citation/verdict)
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
  - `best_effort` partial shard failures: pipeline продолжается, но итоговый status `failed` + `error_code=run_partial_failed`
- docflow builder seam:
  - staged artifacts, citation index, final run index and semantic snapshot remain characterization-covered before promotion
  - promotion still copies only the validated final set into canonical `reports/*`/`proposals/*` and rebuilds derived `model/*`
- UI route-shell seams:
  - `RunPanels` receives grouped `model/actions`, while run selection, stale artifact clearing, logs polling and stable `data-testid` controls remain covered by UI tests
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
  - schema validation
  - parse examples/fixtures
- `backend`
  - `go test ./...`
  - `python3 -m unittest discover -s scripts/tests -p '*_test.py'`
  - includes docs-consistency gate (`internal/docsync`) для truth-sync/stale-marker/CLI-docs parity checks
  - includes harness regression fixtures for batch failure classification (`scripts/tests/*`)
  - `make test-stress` (coordinator debounce/queue regression loop)
  - `go build ./cmd/acp`
- `ui`
  - `./scripts/run-npm.sh ci --prefix ui`
  - `./scripts/run-npm.sh run typecheck --prefix ui`
  - `./scripts/run-npm.sh run test --prefix ui -- --run`
  - `./scripts/run-npm.sh run build --prefix ui`

Implemented additional jobs:
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
  - pipeline status/artifacts/logs endpoints
  - dynamic free port + explicit fail on run polling timeout

Security/advisory workflows:
- `dependency-review` runs on pull requests and blocks newly introduced vulnerable dependencies.
- `codeql` runs Go and JavaScript/TypeScript analysis on pull requests, pushes to `main`, and weekly schedule.
- `scorecard` runs OpenSSF Scorecard on push/schedule with top-level read-only workflow permissions; the scorecard job alone gets `id-token: write` and `security-events: write` for result publishing/SARIF upload, and the action is pinned to the upstream tag's peeled commit so Scorecard publish verification can resolve the action owner correctly. Full Scorecard publish verification is confirmed on default-branch push/schedule; PR branches rely on ordinary required checks.

Release workflow hardening:
- tag-only release workflow uses job-level write permissions, an explicit `github-release` environment, pinned actions, timeouts, and provenance/SBOM artifact generation.
- GitHub environment required reviewers, protected tags, branch protection, Dependabot alerts/security updates, secret scanning, and push protection are repository settings and must be enforced by owners/admins.
## 7) Базовый набор тестов

### Contract tests
- valid `workspace.yaml`
- invalid `workspace.yaml`
- valid docs-first contracts (`shard-pack-manifest`, `final-run-index`, `citation-index`, `validator-verdict`)
- negative docs-first contract cases (missing citations, duplicate claim/topic ids, broken topic refs)
- valid persisted runtime execution metadata
- invalid runtime execution metadata
- invalid artifact contracts (`shard-pack-manifest`, `validator-verdict`, draft manifests)
- strict collect validation:
  - artifact-root-prefixed, absolute, missing-file, and directory `documents[].path` fail-ятся без rewrite
  - missing required metadata fail-ится без autofill
  - collect pair recovery запускается один раз только при no authored artifacts + non-empty provider diagnostics и разрешает писать только suggested authored doc + `shard-pack-manifest.json`
  - manifest-only runtime repair запускается один раз только при authored docs + missing/invalid `shard-pack-manifest.json`
  - manifest-only runtime repair fail-ится, если provider пишет что-либо кроме `shard-pack-manifest.json`
  - repair include dirs исключают broader workspace `reports/taskruns`/sibling manifests и оставляют только current write root + repo evidence roots
- strict validator normal prompt и repair prompt используют command-first absolute heredoc skeleton для `validator-verdict.json`; repair запускается максимум один раз, пишет только `validator-verdict.json`, указывает `checked_paths` на staged final artifacts, требует canonical `issues[]` shape и reject-ит legacy issue fields
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
  - pagination + invalid params + run_not_found
  - structured failure diagnostics в `fields` (`stdout_snippet`/`stderr_snippet`, `task_id`, `provider`, counters)
  - mixed wire-shape (`kind=event|runtime_output`, optional `stream=stdout|stderr`)
- run cancel endpoint:
  - `POST /api/pipeline/runs/<run_id>/cancel`
  - happy-path `202`, `404 run_not_found`, `409 run_not_cancelable`, `400 invalid_request_body`
- UI path: open workspace, validate, run, inspect coverage/questions
- UI run logs surface:
  - log panel render (`Runs: Logs`)
  - log polling/append without duplicates
  - view toggle `line | line+fields`
  - mode toggle `event timeline | raw agent stream | all`
  - quick action `Open runtime execution artifact`
- UI results diagrams surface:
  - navigation `Results -> Diagrams`
  - diagram artifact listing and Mermaid preview render
- UI run lifecycle operability:
  - bootstrap auto-select newest active run
  - если выбранный run исчезает из list endpoint и replacement доступен, UI переключается на следующий run; если list endpoint временно пуст, но status endpoint ещё жив, selection сохраняется
  - `Run status` показывает полный warnings list выбранного run
  - `Cancel selected run` корректно обрабатывает `202/404/409`
- UI runtime settings surface:
  - save/reset `Runtime Timeouts`
  - save/reset `Runtime Execution`
- UI quick actions:
  - `Open runtime execution artifact` открывает persisted taskrun artifact без live e2e-only допущений
- Подробный command cookbook по trusted-machine live/release gate intentionally вынесен в `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.

### Optional live-runner smoke
- Local `manual-live-e2e workflow` is the trusted-machine operator procedure from `docs/RELEASE_LIVE_E2E_RUNBOOK.md`, not a GitHub Actions workflow.
- `scripts/internal/live-e2e-evaluator.sh` is a source-only internal evaluator helper for durable step evidence; it is not a public entrypoint and must not call `scripts/full-run-batch-matrix.sh`.
- `scripts/live-e2e-plan.py` — catalog-driven command generator for direct matrix harness invocations:
  - does not execute the harness and does not replace `scripts/full-run-batch-matrix.sh`
  - supports flexible selectors `smoke tiny`, `regres fast|long|full`, `release fast|long|full`
  - `smoke tiny` is `1 repo × 1 run × 1 provider` for fastest trusted-machine signal
  - generated `regres`/`release` commands rely on the existing quality path: `reports/taskruns/<run_id>-quality.json`, `quality_report_<batch-id>.md`, `quality_gates_failed=0`, no `artifact_quality:*`
- `scripts/full-run-batch.sh` — canonical live batch + frontend live e2e:
  - canonical input: `TARGET_REPOS_FILE`
  - direct-only runtime commands: `claude`, `qwen`, `codex`
  - selected-provider readiness записывается в `preflight.json`; version + provider-specific bounded headless probe + artifact smoke ловят missing binary, auth/quota, codex CLI compatibility и no-write host/provider failures до deep run; для `claude` artifact smoke является основным headless readiness gate, allowlist-ит temp write dir через `--add-dir` и получает один bounded retry на timeout/no-output; provider `model`/`modelUsage` telemetry не является blocker
  - backend quality source-of-truth: только `snapshots/<run_id>/reports/*`
  - hard-fail checks: `analysis:off-topic`, `analysis:evidence-scope`, `analysis:cross-doc`, `analysis:cross-repo-missing`; cross-repo presence can be satisfied by explicit `semantic.edges[]`, by findings with multi-repo provenance evidence, or by questions with multi-repo `related_ids` when repo-wide `citations[].repo` coverage exists, and report details must name the missing dimension. Prompt-contract tests cover the multi-repo validator first-action skeleton so qwen/codex cannot complete `step3` with an empty valid verdict before cross-repo policy is applied.
  - frontend smoke работает на отдельной `frontend-workspace` копии run snapshot и не мутирует backend baseline; `snapshot_reports_missing` после terminal backend failure записывается как dependent skipped frontend status, а не independent frontend regression
  - terminal-success backend runs (`result=passed`, `quality_gates=passed`, `run-status.env state=completed process_exit=0`) остаются `failure_class=none`, даже если raw provider logs содержат recovered `runner_unavailable`/429 diagnostics
  - terminal quality failures (`failure_reason=quality` или `quality_gates=failed`) классифицируются как `quality_gates_failed`, даже если stale classifier rows/raw logs содержат `runner_unavailable`
  - quality summary/matrix counters агрегируют `repair_attempts`, `repair_exhausted`, `fresh_retries`, `focused_repairs`, `stall_count`, `pre_artifact_stalls`, `post_artifact_stalls`, `zero_output_pre_artifact_stalls`, `partial_failure_count` и `quality_alerts`; non-exhausted repair/stall pressure visible but non-blocking, partial failures remain blockers
  - batch report evidence tests проверяют, что `collect_partial_shard_failures`, focused recovery exhaustion/write-set violations и missing headless rows with runtime logs surfaced as per-run issue details, а не теряются за aggregate failure class
  - black-box step evidence через internal evaluator helper пишется в `reports/blackbox_e2e_steps_<batch-id>.jsonl/.md` после preflight, backend run, frontend init/cancel, report synthesis и final classification
- `scripts/full-run-batch-matrix.sh` — официальный local trusted-machine harness:
  - canonical input: `E2E_MATRIX_FILE`
  - approved profile ids: `single-path`, `single-git_url`, `multi-path`, `multi-git_url`
  - non-release slices: `examples/e2e-matrix.regres-*.yaml`
  - diagnostic slices for generated selectors: `examples/e2e-matrix.smoke-tiny.bank.yaml`, `examples/e2e-matrix.diagnostic.sentry.yaml`
  - release-specific slices, `baseline` + `parallel-default`, strict blockers и release verdict policy живут только в `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
  - matrix invariant: для одного `profile_id` shard-plan должен совпадать между `baseline` и `parallel-default`
  - для `source_kind=git_url` refs должны быть pinned
  - child batch stdin is detached from the planned profile/sweep combinations file; regression coverage forces a dummy child to drain stdin and still requires all matrix rows to execute
  - black-box matrix evidence через internal evaluator helper пишется в `reports/blackbox_e2e_steps_<matrix-id>.jsonl/.md` после preflight, planning, каждого profile/sweep и verdict verification
  - итоговый release decision брать только из `reports/release_verdict_<matrix-id>.json`
  - pre-tag/offline verifier: `python3 scripts/verify-release-verdict.py reports/release_verdict_<matrix-id>.json`; скрипт только проверяет существующий verdict JSON и не запускает live harness
- `scripts/frontend-live-e2e.sh` и `npm run e2e:live --prefix ui` используют Playwright:
  - local wrapper поддерживает `claude-code`, `qwen-code`, `codex-code`
  - canonical toggles: `UI_E2E_EXPECTED_REPO_COUNT`, `UI_E2E_SCENARIO=init-inspect|cancel-refresh`, `UI_E2E_OUTPUT_DIR`
  - diagnostic-only `UI_E2E_SCENARIO=api-context-page-close-smoke` proves Playwright API polling survives a closed page; it is not part of release acceptance
  - cancel flow остаётся guarded сценарием с явным `run_canceled`
  - init inspect обязан различать `active_run_timeout`, `runtime_run_failed`, `browser_closed`, `api_unreachable`, `server_exited` и fallback `playwright_failed`, чтобы backend run failure, browser lifecycle, API/server lifecycle и productive runtime timeout не выглядели одним failure class
  - long-running run polling использует independent API request context и не зависит от lifetime browser page, которая нужна только для UI assertions
  - init poll budget берётся из effective runtime timeouts and may be raised to `ACP_PIPELINE_TIMEOUT_SEC+30`; fixed cap is opt-in diagnostic only
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
  - controlled snapshot refresh:
  - `ACP_UPDATE_SCENARIO_GOLDEN=1 go test ./internal/orchestrator -run TestScenarioFixturesDeterministicInitPipeline -count=1`

## 9) Технологические defaults

- Public product APIs и schema contracts этим документом не меняются
- для schema validation в CI используется Draft 2020-12 compatible validator
- основной backend test loop предполагает `go test`
- UI smoke стек: `React + Vite + Vitest + Playwright`
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
- `make lint`
- `make build`
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
