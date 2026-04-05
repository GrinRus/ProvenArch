# Стратегия тестирования ACP MVP

Этот документ фиксирует baseline testing strategy для ACP MVP.

## 1) Цели и принципы

- Required CI должен проходить локально и в CI без live network dependencies.
- Required CI не зависит от live Claude Code, GitHub, GitLab или реальных пользовательских репозиториев.
- Любые изменения schema/spec/examples должны сопровождаться обновлением fixtures и golden outputs в том же PR.
- Synthetic fixtures считаются baseline regression surface.
- Live Claude Code проверяется только optional smoke на trusted machine/runner и не блокирует merge.

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
- docs truth-sync gate проверяет:
  - согласованность runtime policy/Q&A boundary и ссылок на canonical stakeholder matrix;
  - отсутствие stale-маркеров в ключевых surfaces (`future`, `skeleton`, `placeholder`, устаревшие version-маркеры);
  - CLI docs parity: базовые `acp serve|run|qa` usage и runtime flags в help и документации совпадают

## 5) Обязательные internal test seams

- fake/recorded runner вместо live Claude Code в required tests
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
  - pipeline status/artifacts endpoints
  - dynamic free port + explicit fail on run polling timeout
- `ui-smoke`
  - workspace setup
  - baseline editor save (`charter/*`/`skills/*`)
  - validate
  - run pipeline
  - results viewer
- `live-runner-smoke`
  - manual/opt-in only
  - never required for merge

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

### Smoke tests
- `acp run --workspace ... --pipeline init --runtime fake --non-interactive`
- `acp run --workspace ... --pipeline refresh --runtime fake --non-interactive`
- `acp serve --workspace ... --runtime fake`
- `/api/workspace/validate` без request body
- pipeline endpoints не принимают `workspace_path`
- UI path: open workspace, validate, run, inspect coverage/questions

## 8) Acceptance для testing strategy

- любой required CI run проходит без live network dependencies
- любое изменение schema/spec/examples требует update fixtures/golden в том же PR
- live Claude Code smoke не блокирует merge и запускается только вручную/по opt-in
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

## 10) Developer entrypoints

- `make bootstrap`
- `make contracts`
- `make test`
- `make lint`
- `make build`
- `make run-backend WORKSPACE=/abs/path/to/arch-workspace`
- `make run-ui`
