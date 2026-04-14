# Стратегия тестирования ACP MVP

Этот документ фиксирует baseline testing strategy для ACP MVP.

## 1) Цели и принципы

- Required CI должен проходить локально и в CI без live network dependencies.
- Required CI не зависит от live headless providers (`claude-code`/`qwen-code`), GitHub, GitLab или реальных пользовательских репозиториев.
- Любые изменения schema/spec/examples должны сопровождаться обновлением fixtures и golden outputs в том же PR.
- Synthetic fixtures считаются baseline regression surface.
- Live headless providers проверяются только optional smoke на trusted machine/runner и не блокируют merge.
- Отдельно от merge-gates используется manual pre-release live gate:
  - `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
  - verdict `PASS|FAIL` с policy strict zero-failure.

## 2) Тестовая пирамида MVP

### Contract tests
- `workspace.yaml` валидируется по `schemas/workspace.schema.json`
- `TaskResult` валидируется по `schemas/taskresult.schema.json`
- examples и fixture cases должны парситься и проходить contract validation, где это ожидается

### Semantic validator tests
- правила, которые не выражаются чистой JSON Schema
- deterministic normalization legacy `questions/coverage`
- stable ID normalization и collision rules
- ownership/card linkage constraints

### Golden/regression tests
- model store materialization
- compiler outputs (`reports/as-is/*`, findings, proposals, changelog)
- deterministic comparisons against recorded golden outputs
- hash-based snapshot compare against `fixtures/scenarios/*/golden/snapshot.sha256`

### Scenario integration tests
- pipeline runs на synthetic repos и fixture workspaces
- recorded raw TaskResult вместо live runner в required tests
- fixture contract gate проверяет parse/semantics recorded runner outputs (`meta.step_id`, `repo_scopes`)

### Smoke tests
- CLI smoke
- API smoke
- UI smoke

### Optional live-runner smoke
- только manual/opt-in
- не входит в required CI gates

## 3) Обязательная структура test assets

- `fixtures/workspace/` — manifest и validator cases
- `fixtures/taskresult/` — raw и normalized TaskResult cases
- `fixtures/scenarios/<name>/workspace/` — central workspace inputs
- `fixtures/scenarios/<name>/repos/<repo-name>/` — synthetic repos
- `fixtures/scenarios/<name>/runner/` — recorded raw TaskResult per step
- `fixtures/scenarios/<name>/golden/` — expected deterministic snapshot (hash list) + fixture docs

Baseline scenario set:
- `single-service-http-postgres-gitlabci`
- `two-services-http-call-and-queue`
- `missing-owner-and-missing-cicd`

## 4) Обязательные semantic checks

- duplicate `repo.name` rejected
- unsupported manifest fields rejected
- mixed top-level `questions/coverage` + legacy ops normalize deterministically
- `observation` without evidence rejected
- `add_doc_artifact` не используется как content write path
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
  - stale persisted `queued/running` run при старте сервиса reconciled в `failed` + `error_code=run_reconciled_after_restart`
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
  - отсутствие stale-маркеров в ключевых surfaces (`future`, `skeleton`, `placeholder`, устаревшие version-маркеры);
  - CLI docs parity: базовые `acp serve|run|qa` usage и runtime flags в help и документации совпадают

## 5) Обязательные internal test seams

- fake/recorded runner вместо live headless providers в required tests
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
  - includes docs-consistency gate (`internal/docsync`) для truth-sync/stale-marker/CLI-docs parity checks
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
- `ui-smoke`
  - workspace setup
  - baseline editor save (`charter/*`/`skills/*`)
  - validate
  - run pipeline
  - results viewer
## 7) Базовый набор тестов

### Contract tests
- valid `workspace.yaml`
- invalid `workspace.yaml`
- valid canonical TaskResult
- valid legacy-compatible TaskResult with normalization
- invalid TaskResult with schema violations

### Semantic tests
- duplicate repo names
- unsupported manifest fields
- `observation` without evidence
- unknown `owner_team_id`
- mixed top-level and legacy coverage/questions merge

### Golden tests
- entity/edge file materialization
- stable slug normalization and collision handling
- Step 2 `reports/as-is/*`
- Step 3 findings materialization
- Step 4 proposals/changelog determinism

### Scenario integration tests
- one-service happy path
- multi-repo dependency extraction
- missing owner / missing CI-CD evidence path
- unresolved domain/team becomes question/finding, not new card
- deterministic Step 1 enrichment включает `evidence_refs` в domain/team cards
- sharded runtime regression:
  - step1/step3 materialize per-shard taskruns + shard-plan/shard-summary artifacts
  - parallel scheduler keeps deterministic merge/apply order despite out-of-order shard completion
  - TaskResult shard metadata (`meta.shard_id`, `meta.repo_scopes`, `meta.path_scopes`) сохраняется в persisted taskruns

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
- run cancel endpoint:
  - `POST /api/pipeline/runs/<run_id>/cancel`
  - happy-path `202`, `404 run_not_found`, `409 run_not_cancelable`, `400 invalid_request_body`
- UI path: open workspace, validate, run, inspect coverage/questions
- UI run logs surface:
  - log panel render (`Runs: Logs`)
  - log polling/append without duplicates
  - view toggle `line | line+fields`
  - quick action `Open taskrun artifact`
- UI run lifecycle operability:
  - bootstrap auto-select newest active run
  - если выбранный run исчезает из list endpoint, UI очищает stale run details/logs и не auto-switch-ится на другой run
  - `Run status` показывает полный warnings list выбранного run
  - `Cancel selected run` корректно обрабатывает `202/404/409`
- локальный full-run regression сценарий на реальном репозитории:
  - `scripts/full-run-ai-advent.sh`
  - bootstrap в `tmp`, API simulation, runtime циклы `fake + headless`
  - strict quality checks: anti-mock + anti-zero-signal + no last-run degradation
  - completion invariants: expected/completed runtime counts, per-iteration headless `init+refresh`, отсутствие `running` в run-history
  - signal handling: `TERM/INT/HUP/PIPE` => `infra_signal_terminated`, `result=passed` запрещён при неполном цикле
  - full-run semantic checks ограничены локальным скриптом (owner-gap/findings, coverage/questions dedupe, critical off-topic markers) и не включают batch-only `analysis:evidence-scope`/`analysis:cross-doc`
  - summary/log/snapshots: `TMP_ROOT/session-summary.md`, `TMP_ROOT/full-run.log`, `TMP_ROOT/snapshots/*`
  - при parse-fail runtime сохраняет raw-output diagnostics в `reports/taskruns/raw/*` (stdout/stderr/meta with checksum)
- batch regression `5x2` + frontend live e2e:
  - `scripts/full-run-batch-5x2.sh`
  - canonical input: `TARGET_REPOS_FILE` (`repos[]` format)
  - legacy compatibility: `TARGET_REPO` или `TARGET_REPO_GIT_URL+TARGET_REPO_NAME+TARGET_REPO_REF`
  - direct-only runtime commands (`claude`, `qwen`)
  - frontend live e2e работает на отдельной `frontend-workspace` копии run snapshot, не мутируя backend baseline
  - backend quality source-of-truth: snapshot reports (`snapshots/<run_id>/reports/*`), fallback помечается как `reliability:snapshot-missing`
  - semantic hard-fail checks в batch evaluator: `analysis:off-topic`, `analysis:evidence-scope`, `analysis:cross-doc`
  - multi-profile hard-fail: `analysis:cross-repo-missing` при `expected_repo_count >= 2` и отсутствии cross-repo сигнала
  - runtime-flow hard-fail checks: `runtime:shard-artifacts`, `runtime:shard-metadata`, `runtime:repo-selection`, `runtime:execution-semantics`, `runtime_flow_failed`
  - hard-pass учитывает semantic hard-fail и snapshot source validity
  - run artifacts default: `/tmp/provenarch-test_arch_project/runs/<batch-id>/<provider>/runN/*`
  - reports: `run_matrix_<batch-id>.md/.tsv`, `frontend_e2e_matrix_<batch-id>.md`, `frontend_cancel_e2e_matrix_<batch-id>.md`, `quality_report_<batch-id>.md` (+ fields `artifact_source`, `semantic_hard_fail`, `off_topic_hits`, runtime-flow checks, failure classes `runtime_parse/runner_unavailable/runtime_timeout/infra_signal_terminated/infra_incomplete_cycle/quality_gates_failed/summary_missing/precheck_failed`)
- profile matrix regression (local official runbook, non-required CI):
  - `scripts/full-run-batch-matrix.sh`
  - `E2E_MATRIX_FILE` обязателен (`profiles[]`: `id`, `repos_file`, `expected_repo_count`, `source_kind`)
  - `sweeps[]` optional (если отсутствует -> implicit `baseline`)
  - release-ready sweep set: `baseline` + `scale-backend`
  - относительные `repos_file` пути резолвятся от директории `E2E_MATRIX_FILE`
  - официальные профили: `single-path`, `single-git_url`, `multi-path`, `multi-git_url`
  - для `source_kind=git_url` refs должны быть pinned
  - агрегированные отчёты: `profile_matrix_<matrix-id>.md/.tsv`, `release_verdict_<matrix-id>.md/.json`
- release live harness (manual pre-release gate, no wrapper):
  - source-of-truth runbook: `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
  - использует текущий matrix контур (`full-run-batch-matrix.sh` + `full-run-batch-5x2.sh` + `e2e_batch_report.py`)
  - release-mode guard (auto при `MATRIX_ID=release-*`) блокирует diagnostic timeout overrides; debug bypass только через `E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES=1`
  - обязательны оба провайдера (`qwen-code`, `claude-code`) и оба frontend сценария (`init-inspect`, `cancel-refresh`)
  - strict acceptance: только `PASS`, любое нарушение quality/failure-class критериев = `RELEASE BLOCKED`
- optional frontend live smoke:
  - `scripts/frontend-live-e2e.sh` (local)
  - direct `npm run e2e:live --prefix ui`: Playwright output default `/tmp/provenarch-ui-e2e/test-results` (override: `UI_E2E_OUTPUT_DIR`)
  - `scripts/frontend-live-e2e.sh`: Playwright output в `$OUTPUT_DIR/playwright-results`
  - `scripts/frontend-live-e2e.sh` читает effective UI poll timeouts из `GET /api/runtime/timeouts` (если env override не задан)
  - `UI_E2E_EXPECTED_REPO_COUNT` задаёт ожидаемое количество resolved repos (default `1`)
  - `UI_E2E_SCENARIO=init-inspect|cancel-refresh` переключает live flow:
    - `init-inspect`: validate -> run init -> inspect artifacts
    - `cancel-refresh`: validate -> run refresh -> cancel selected run -> expect `failed + run_canceled`
  - `UI_E2E_CANCEL_STUB_SLEEP_SEC` задаёт длительность controlled slow stub runner для `cancel-refresh`
  - cancel preflight guard: `UI_E2E_CANCEL_TIMEOUT_SEC >= UI_E2E_CANCEL_STUB_SLEEP_SEC + UI_E2E_CANCEL_TIMEOUT_MARGIN_SEC`; при нарушении сценарий fail-fast до Playwright
  - `ui/e2e/live-flow.spec.ts` + `npm run e2e:live --prefix ui`
  - batch shard controls (`scripts/full-run-batch-5x2.sh`):
    - `BATCH_PROVIDER_FILTER` (`all` или CSV `qwen-code,claude-code`)
    - `BATCH_RUN_SELECTION` (`all`, CSV `1,3,5` или диапазоны `1-3,5`)
    - `BATCH_SKIP_PRECHECK=1` для secondary shard'ов
    - `BATCH_FRONTEND_MODE=auto|always|never` (default `auto`, auto-skip frontend если `run1` не выбран)
    - параллельные shard-процессы должны использовать разные `BATCH_ID`

## 8) Acceptance для testing strategy

- любой required CI run проходит без live network dependencies
- любое изменение schema/spec/examples требует update fixtures/golden в том же PR
- live headless provider smoke не блокирует merge; для обязательного CI используется только `contracts`, `backend`, `ui`, `golden`, `smoke-cli`, `smoke-api`, `ui-smoke`
- workflow `ui-live-smoke-optional` запускается только вручную (`workflow_dispatch`) и не является required gate
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
- UI smoke предполагает React test stack; конкретный framework выбирается при реализации UI
- Balanced timeout defaults:
  - step `1800s`, heartbeat `30s`, pipeline `2400s`, kill-grace `30s`
  - api-ready `60s`, api-init `120s`, ui-init poll `900s`, ui-cancel poll `420s`

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
