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
- runtime sharding control:
  - heuristics planner (module markers + leaf-pruning) и `analysis.include/exclude` фильтры
  - fallback warning + root shard `.` при пустом результате фильтров
  - scheduler semantics `sequential|parallel` (`max_parallel_tasks`) и deterministic apply order
  - `fail_fast` останавливает step/pipeline на первой shard error без перехода в downstream runtime steps
  - `best_effort` partial shard failures: pipeline продолжается, но итоговый status `failed` + `error_code=run_partial_failed`
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

## 6) Required CI jobs

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
  - `npm ci --prefix ui`
  - `npm run typecheck --prefix ui`
  - `npm run test --prefix ui -- --run`
  - `npm run build --prefix ui`

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
## 7) Базовый набор тестов

### Contract tests
- valid `workspace.yaml`
- invalid `workspace.yaml`
- valid docs-first contracts (`shard-pack-manifest`, `final-run-index`, `citation-index`, `validator-verdict`)
- negative docs-first contract cases (missing citations, duplicate claim/topic ids, broken topic refs)
- valid persisted runtime execution metadata
- invalid runtime execution metadata
- invalid artifact contracts (`shard-pack-manifest`, `validator-verdict`, draft manifests)
- collect manifest repair report:
  - safe normalization возвращает `applied_rule_ids=["collect.documents_path_normalization"]`
  - ambiguous/unsafe path не repair-ится и не расширяет compatibility inventory
- compatibility inventory ограничен двумя safe rules (`collect.documents_path_normalization`, `drafts.reconcile_existing_canonical_outputs`)
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
- Подробный command cookbook по live/full-run intentionally вынесен в runbook'и:
  - `docs/LOCAL_FULL_RUN_AI_ADVENT.md`
  - `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

### Optional live-runner smoke
- `scripts/live-e2e-plan.py` — catalog-driven command generator for direct matrix harness invocations:
  - does not execute the harness and does not replace `scripts/full-run-batch-matrix.sh`
  - supports flexible selectors `smoke tiny`, `regres fast|long|full`, `release fast|long|full`
  - `smoke tiny` is `1 repo × 1 run × 1 provider` for fastest trusted-machine signal
  - generated `regres`/`release` commands rely on the existing quality path: `reports/taskruns/<run_id>-quality.json`, `quality_report_<batch-id>.md`, `quality_gates_failed=0`, no `artifact_quality:*`
- `scripts/full-run-ai-advent.sh` — canonical local scenario/full-run loop:
  - supported headless providers: `claude-code`, `qwen-code`, `codex-code`
  - canonical input: `TARGET_REPOS_FILE`
  - bootstrap в `tmp`, runtime циклы `fake + headless`, strict anti-mock/anti-zero-signal guardrails
  - completion invariants: expected/completed runtime counts, per-iteration headless `init+refresh`, отсутствие `running` в `run-history`
  - signal handling: `TERM/INT/HUP/PIPE` => `infra_signal_terminated`
  - debug artifacts и raw diagnostics: `TMP_ROOT/session-summary.md`, `TMP_ROOT/full-run.log`, `TMP_ROOT/snapshots/*`, `reports/taskruns/raw/*`
- `scripts/full-run-batch-5x2.sh` — canonical batch `5x2` + frontend live e2e:
  - canonical input: `TARGET_REPOS_FILE`
  - direct-only runtime commands: `claude`, `qwen`, `codex`
  - backend quality source-of-truth: только `snapshots/<run_id>/reports/*`
  - hard-fail checks: `analysis:off-topic`, `analysis:evidence-scope`, `analysis:cross-doc`, `analysis:cross-repo-missing`
  - frontend smoke работает на отдельной `frontend-workspace` копии run snapshot и не мутирует backend baseline
- `scripts/full-run-batch-matrix.sh` — официальный local trusted-machine harness:
  - canonical input: `E2E_MATRIX_FILE`
  - approved profile ids: `single-path`, `single-git_url`, `multi-path`, `multi-git_url`
  - non-release slices: `examples/e2e-matrix.regres-*.yaml`
  - diagnostic slices for generated selectors: `examples/e2e-matrix.smoke-tiny.bank.yaml`, `examples/e2e-matrix.diagnostic.sentry.yaml`
  - release-specific slices, `baseline` + `parallel-default`, strict blockers и release verdict policy живут только в `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
  - matrix invariant: для одного `profile_id` shard-plan должен совпадать между `baseline` и `parallel-default`
  - для `source_kind=git_url` refs должны быть pinned
  - итоговый release decision брать только из `reports/release_verdict_<matrix-id>.json`
- `scripts/frontend-live-e2e.sh` и `npm run e2e:live --prefix ui` используют Playwright:
  - local wrapper поддерживает `claude-code`, `qwen-code`, `codex-code`
  - canonical toggles: `UI_E2E_EXPECTED_REPO_COUNT`, `UI_E2E_SCENARIO=init-inspect|cancel-refresh`, `UI_E2E_OUTPUT_DIR`
  - cancel flow остаётся guarded сценарием с явным `run_canceled`
  - init inspect обязан различать `playwright_failed` и `active_run_timeout`, если run остаётся продуктивным, но не доходит до `succeeded` в UI poll budget
- Этот документ фиксирует policy, invariants и required gates; пошаговые live/release cookbook команды не дублируются здесь.

## 8) Acceptance для testing strategy

- любой required CI run проходит без live network dependencies
- любое изменение schema/spec/examples требует update fixtures/golden в том же PR
- live headless provider smoke не блокирует merge; для обязательного CI используется только `contracts`, `backend`, `ui`, `golden`, `smoke-cli`, `smoke-api`
- release gate выполняется вручную перед релизом на trusted машине по `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
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
- `./scripts/full-run-ai-advent.sh`
- `./scripts/full-run-batch-5x2.sh`
- `./scripts/full-run-batch-matrix.sh`
- `./scripts/frontend-live-e2e.sh`
- runtime live log seam:
  - mixed `event` + `runtime_output` entries в run logs
  - `runtime_output.stream` (`stdout|stderr`) сохраняется и не ломает pagination
  - hard-cap truncation marker фиксируется как `fields.output_truncated=true`
- Step 2 diagram compiler regression:
  - deterministic C4 artifacts + stable index ordering
  - strict evidence gap markers (`Gap:*`) при недостатке данных
