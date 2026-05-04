# PLANS_ARCHIVE_2026-05.md

Closed ExecPlans archived from `docs/PLANS.md` in May 2026.

### Plan ID
EP-20260504-public-oss-release-readiness

### Context
Репозиторий переведён в public visibility, а release workflow уже умеет публиковать single-binary `acp` по тегам `v*`. Перед первым public release нужно было закрыть OSS-readiness gaps: license, release metadata в binary, user-facing release status и smoke проверка installed binary без workspace.

### Goals (must have)
- [x] Добавить Apache-2.0 `LICENSE`
- [x] Добавить `acp version` / `acp --version`
- [x] Inject release metadata через GoReleaser ldflags
- [x] Обновить README/install/architecture/stakeholder/CLI docs
- [x] Обновить installer/release tests под version smoke
- [x] Выполнить local DoD и release snapshot перед PR
- [x] После merge опубликовать `v0.1.0`, проверить GitHub Release artifacts и public install smoke

### Non-goals
- [x] Не менять pipeline/runtime schemas или workspace contract
- [x] Не добавлять Homebrew tap в этот slice
- [x] Не менять primary distribution beyond GitHub Releases + `install.sh`

### Approach
1) Добавить release metadata command и ldflags.
2) Добавить Apache-2.0 license и синхронизировать user-facing docs.
3) Обновить тесты installer/release distribution.
4) Прогнать DoD/snapshot, открыть PR, дождаться CI, merge, затем опубликовать `v0.1.0`.

### Files expected to change
- `LICENSE`
- `cmd/acp/main.go`, `cmd/acp/main_test.go`, `cmd/acp/README.md`
- `.goreleaser.yml`, `scripts/tests/*release*_test.py`, `README.md`, `docs/INSTALL.md`, `docs/ARCHITECTURE.md`, `docs/STAKEHOLDER_DOC.md`

### Acceptance criteria
- [x] `acp version` works from source build and snapshot release archive
- [x] Local release snapshot contains 4 platform archives and `checksums.txt`
- [x] Local install smoke from snapshot passes
- [x] GitHub identifies repository license as Apache-2.0 after merge
- [x] Public `install.sh` path is reachable after merge
- [x] `v0.1.0` GitHub Release contains 4 platform archives and `checksums.txt`
- [x] Public install smoke and fake first-run smoke pass from released binary

### Risks
- Release tag workflow is first-run only; unexpected GitHub token or GoReleaser publish issue may require a follow-up fix and retag.
- README mentioned `v0.1.0` before the tag existed during the short PR-to-release window.

### Progress log
- 2026-05-04: Started public OSS release readiness after repository visibility changed to public.
- 2026-05-04: Added Apache-2.0 license, version command, GoReleaser ldflags, docs/tests sync; local DoD, snapshot release and snapshot install smoke passed.
- 2026-05-04: Merged PR #49, published `v0.1.0`, verified release workflow, public release assets/checksums, public `install.sh`, `acp version`, `acp doctor`, embedded UI and fake first-run smoke from released binary.

### Plan ID
EP-20260504-user-friendly-distribution

### Context
MVP feature matrix закрыта, но текущий GitHub-only путь остаётся developer-oriented: пользователь должен ставить Go/Node, собирать бинарь из исходников и понимать `workspace.yaml` до первого результата. Для beta adoption нужен user-first local-first flow: установить готовый `acp`, открыть embedded UI, подключить GitHub/GitLab URL или локальный checkout, пройти readiness checks и запустить первый `init`.

### Goals (must have)
- [x] Добавить release packaging через GoReleaser для macOS/Linux `amd64/arm64`
- [x] Добавить GitHub Actions release workflow для tag `v*`
- [x] Добавить `install.sh` с checksum verification и тестами
- [x] Добавить read-only `acp doctor` для install/workspace/repo/runtime/UI readiness
- [x] Добавить `GET /api/system/doctor` для UI readiness checklist
- [x] Переработать Setup UI в first-run stepper `Source -> Workspace -> Runtime -> Validate -> Run`
- [x] Спрятать raw `workspace.yaml` editor в Advanced и добавить first-run CTA
- [x] Обновить README/install/troubleshooting/API/architecture docs
- [x] Прогнать full DoD (`make contracts`, `make test`, `make lint`, `make build`)

### Non-goals
- [x] Не добавлять hosted mode, credential store или запись в user repos
- [x] Не менять runtime artifact schemas/contracts
- [x] Не делать Docker primary distribution path
- [x] Не публиковать Homebrew tap в этом slice; зафиксировать как next release follow-up

### Approach
1) Добавить native release artifacts: GoReleaser config, tag-based GitHub release workflow, checksum-aware installer.
2) Вынести readiness checks в reusable `internal/doctor`, подключить CLI `doctor` и API `/api/system/doctor`.
3) Обновить Setup UI до guided first-run flow поверх существующих hooks/API без изменения process-scoped runtime semantics.
4) Синхронизировать docs и добавить focused Go/UI/script tests.

### Files expected to change
- `cmd/acp/main.go`, `internal/doctor/*`, `internal/api/*`
- `ui/src/components/SetupWorkspacePanel.tsx`, `ui/src/App.tsx`, `ui/src/lib/*`, `ui/src/hooks/*`, `ui/src/styles.css`
- `.goreleaser.yml`, `.github/workflows/release.yml`, `install.sh`, `scripts/tests/install_script_test.py`, `scripts/tests/release_distribution_test.py`
- `README.md`, `docs/INSTALL.md`, `docs/TROUBLESHOOTING.md`, `docs/ARCHITECTURE.md`, `docs/spec/API_SPEC.md`, `cmd/acp/README.md`, `docs/PLANS.md`, `docs/archive/PLANS_ARCHIVE_2026-05.md`

### Acceptance criteria
- [x] `acp doctor --json` reports pass/warn/fail checks and exits `0` when ready, `1` for user-fixable issues, `2` for invalid flags/internal request errors
- [x] `/api/system/doctor` returns the same readiness report shape without mutating workspace
- [x] UI first-run flow supports GitHub/GitLab URL default, local folder mode, validation diagnostics, doctor checklist and `Run first analysis`
- [x] Installer downloads expected release archive, verifies `checksums.txt`, installs `acp` into `INSTALL_DIR`
- [x] Release config targets `darwin/linux` `amd64/arm64` with embedded UI
- [x] DoD passes

### Risks
- `git ls-remote` for private repos depends on local git auth and may fail until the user configures SSH/token credentials.
- Headless readiness only verifies provider command presence; provider auth/model/runtime failures are still surfaced during live runs.
- Homebrew needs a stable first release artifact before creating a tap formula.

### Progress log
- 2026-05-04: Implemented GoReleaser release config, tag release workflow, installer, `doctor` CLI/API, first-run Setup UI and focused tests/docs.
- 2026-05-04: Post-implementation audit fixed remaining gaps: release distribution contract tests, CLI text output coverage, UI local-folder/validation-suggestion coverage, installer version/repo URL coverage, supported GoReleaser `archives.ids`, local `goreleaser check`, and snapshot release build without publish.
- 2026-05-04: Final UI audit fixed stale readiness state after setup edits and hydrated guided first-run form from loaded `workspace.yaml`; full DoD, release snapshot and fake live UI smoke passed.

### Plan ID
EP-20260504-qa-api-imports-hardening

### Context
Аудит покрытия user story показал два закрываемых beta gap: UI wording по editable prompt pack layer отстал от runtime step policy, а `docs.imports_path/index.yaml` описан как metadata index, но не валидируется. Пользователь также выбрал включить follow-up Epic 11: public read-only `POST /api/qa/ask` поверх существующего deterministic `internal/qa.Service`.

### Goals (must have)
- [x] Добавить public read-only `POST /api/qa/ask` без headless runtime, workspace writes, git operations или pipeline run side effects
- [x] Добавить warning-only validation для `<docs.imports_path>/index.yaml` и использовать configured imports path в Q&A indexing
- [x] Синхронизировать UI wording, README/ARCHITECTURE/STAKEHOLDER/API/PIPELINE/BACKLOG docs и docsync tests с implemented Q&A API boundary
- [x] Добавить API/workspace/Q&A/UI/docsync regression coverage и выполнить DoD

### Non-goals
- [x] Не менять `workspace.yaml` schema, runtime provider contract, model schema, pipeline request shape или CLI flags
- [x] Не добавлять hosted mode, webhook listener, external SCM app integration, auth model или headless-QA runtime
- [x] Не делать imports index absence blocking diagnostic

### Approach
1) Реализовать imports index validator как non-blocking workspace warning surface.
2) Перевести Q&A indexing на configured `docs.imports_path`.
3) Добавить `/api/qa/ask` как thin strict-json HTTP wrapper над `internal/qa.Service`.
4) Обновить UI hint, docs/spec and docsync tests.
5) Прогнать regression gate and archive this ExecPlan after completion.

### Files expected to change
- `internal/api/server.go`, `internal/api/server_test.go`
- `internal/workspace/*`, `internal/qa/*`
- `ui/src/components/BaselineEditorsPanel.tsx`, `ui/src/App.test.tsx`
- `README.md`, `docs/ARCHITECTURE.md`, `docs/STAKEHOLDER_DOC.md`, `docs/BACKLOG.md`, `docs/spec/API_SPEC.md`, `docs/spec/PIPELINE_SPEC.md`, `docs/PLANS.md`, `internal/docsync/docsync_test.go`

### Acceptance criteria
- [x] `POST /api/qa/ask` returns `answer`, `citations`, `unresolved`, `confidence` and rejects empty/unknown/malformed requests with existing API error envelope
- [x] Imports index malformed/semantic issues are warnings only; missing index is silent
- [x] Custom `docs.imports_path` works for validation and Q&A citations
- [x] Docs no longer mark `/api/qa/ask` as post-beta/follow-up and still say Q&A is deterministic/read-only/non-headless

### Risks
- Public API docs can drift from route behavior if tests only cover happy path.
- Warning-only imports validation must not accidentally turn existing workspaces red.
- Q&A custom imports path support must avoid path traversal while preserving existing `docs/imports/*` citations.

### Progress log
- 2026-05-04: Implemented read-only `/api/qa/ask`, warning-only imports index validation, configured imports Q&A indexing, UI/docs/docsync sync and full regression gate.

### Plan ID
EP-20260504-architecture-docsync-audit

### Context
Архитектурная ревизия показала, что код в целом соответствует MVP architecture, но active docs drift-или по baseline `qa` skill, Q&A beta wording, prompt-pack coverage для `step2.asis_docs`, artifact ownership taxonomy и закрытым Active-планам.

### Goals (must have)
- [x] Синхронизировать README/ARCHITECTURE/STAKEHOLDER/PIPELINE/BACKLOG/fixtures wording с текущим кодом без изменения public API/schema/CLI contracts
- [x] Зафиксировать `qa` как baseline skill рядом с `system-analyst-qa` и `qa` prompt pack
- [x] Уточнить, что Q&A beta — deterministic workspace-backed read-only service + CLI, а не headless runtime agent/public API
- [x] Уточнить prompt-pack coverage: editable workspace prompt pack layer есть у step0/step1/step3/step4, а step2 остаётся enforced-policy-only без отдельного editable `as-is` pack
- [x] Очистить Active Plans от закрытых completed plans и перенести их в майский архив
- [x] Добавить docsync tests, которые ловят повторный drift по этим surfaces

### Non-goals
- [x] Не менять HTTP API, CLI flags, workspace/model/runtime schemas или provider contracts
- [x] Не добавлять public `/api/qa/ask` или prompt-backed/headless QA runtime в этом slice
- [x] Не добавлять отдельный `as-is` prompt pack
- [x] Не запускать live provider matrix, потому что runtime behavior не меняется

### Approach
1) Принять код как source of truth для docs-only alignment: `internal/workspace/baseline.go`, `internal/qa/service.go`, `internal/runtime/steppolicy/policy.go`, orchestrator/docflow ownership.
2) Обновить active docs wording и revision metadata.
3) Заархивировать completed Active ExecPlans, оставив в `docs/PLANS.md` только планы с открытыми goals.
4) Добавить docsync tests на baseline inventory, Q&A beta boundary, prompt-pack coverage, artifact ownership wording и Active Plans hygiene.
5) Выполнить docs/tests DoD.

### Files expected to change
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/STAKEHOLDER_DOC.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/BACKLOG.md`
- `docs/PLANS.md`
- `docs/archive/PLANS_ARCHIVE_2026-05.md`
- `fixtures/README.md`
- `internal/docsync/docsync_test.go`

### Acceptance criteria
- [x] Public interfaces stay unchanged
- [x] Active docs no longer advertise Q&A as current headless runtime agent/public API
- [x] Baseline docs include `qa` in skills and prompt packs
- [x] Active docs state exact prompt-pack coverage and step2 no-pack boundary
- [x] Active Plans contain only plans with open goals
- [x] `make contracts`, `make test`, `make lint`, `make build` are expected final gates for the implementation slice

### Risks
- `make build` may refresh tracked embedded UI assets; this is expected release-surface behavior if it occurs.

### Progress log
- 2026-05-04: Architecture/docs audit completed and synchronized as docs-only alignment; completed Active plans moved to `docs/archive/PLANS_ARCHIVE_2026-05.md`.

### Plan ID
EP-20260503-refactor-boundary-cleanup

### Context
Запрошенный behavior-preserving срез снижает текущую сложность ACP MVP без изменения public schemas, API routes, CLI flags, provider IDs, release verdict shape или workspace contract. Главная цель — закрепить internal ownership seams вокруг runtime profile patching, provider repair command construction, docflow assembly, sharding planning/scheduling и UI run panels.

### Goals (must have)
- [x] Вынести runtime profile patch/merge/render/reopen из API handlers в internal helper/service
- [x] Разделить `GET /api/pipeline/runs*` handler на list/status/artifacts/logs helpers без изменения route payloads
- [x] Дедуплицировать provider repair command specs для `claude-code`, `qwen-code`, `codex-code`
- [x] Выделить `DocflowBuildInput`/`DocflowBuildResult`, чтобы staged docflow assembly возвращал явный result и не размазывал state mutation по builder logic
- [x] Выделить `ShardPlanner`/`ShardScheduler` seams вокруг deterministic planning и worker-pull scheduling
- [x] Добавить `RuntimeTaskExecutor` seam для runtime task lifecycle и `ShardSummaryStore` seam для summary/checkpoint persistence
- [x] Разбить плоский `pipelineExecution` state на вложенные группы: run-progress, artifact registry, runtime, quality, semantic/docflow, draft
- [x] Физически вынести из `orchestrator.go` run finalization, step handlers и artifact registry methods в dedicated files
- [x] Централизовать compact collect/validator wording в `artifactquality` и переиспользовать его в promptcontract/baseline prompt generation
- [x] Сгруппировать UI `RunPanels` props в `model/actions`, сохранив route shell и test ids
- [x] Прогнать full DoD после документации

### Non-goals
- [x] Не менять public schemas, API routes, CLI flags, provider IDs, release verdict format или workspace contract
- [x] Не добавлять hosted mode, security/compliance enforcement, новых providers или live network required CI gates
- [x] Не добавлять wrapper над `scripts/full-run-batch-matrix.sh`
- [x] Не удалять tracked embedded UI release surface `internal/api/ui_dist`

### Approach
1) Сначала оставить characterization surface неизменным: API error codes/routes, provider command transport, qwen `-p`, codex stdin, sharding fail-fast/best-effort and staged docflow outputs.
2) В API вынести patch lifecycle в `RuntimeProfilePatchService`, а route dispatcher разбить на малые helpers.
3) В runtime adapters оставить thin provider-specific transport, а focused repair prompt/include-dir selection централизовать в `providercommon`.
4) В orchestrator ввести internal builder/planner/scheduler seams без смены promotion, validation, history/log contracts.
5) В UI уменьшить prop-sprawl на route boundary через grouped `model/actions`.
6) Синхронизировать docs и выполнить required checks.

### Files expected to change
- `internal/api/server.go`
- `internal/runtimeprofile/patch_service.go`
- `internal/orchestrator/docflow.go`
- `internal/orchestrator/artifact_registry.go`
- `internal/orchestrator/run_finalization.go`
- `internal/orchestrator/runtime_task_executor.go`
- `internal/orchestrator/sharding.go`
- `internal/orchestrator/sharding_artifacts.go`
- `internal/orchestrator/sharding_coordinator.go`
- `internal/orchestrator/sharding_planner.go`
- `internal/orchestrator/sharding_scheduler.go`
- `internal/orchestrator/sharding_summary_store.go`
- `internal/orchestrator/step_handlers.go`
- `internal/artifactquality/*`
- `internal/workspace/baseline.go`
- `internal/runtime/providercommon/repair_command.go`
- `internal/runtime/{claudecode,qwencode,codexcode}/runner.go`
- `ui/src/App.tsx`
- `ui/src/components/RunPanels.tsx`
- `ui/src/hooks/useRunActions.ts`
- `ui/src/hooks/useRunPolling.ts`
- `ui/src/hooks/useRunSelection.ts`
- `ui/src/hooks/useManifestEditor.ts`
- `ui/src/hooks/useBaselineEditor.ts`
- `ui/src/hooks/useWizardEditor.ts`
- `ui/src/hooks/useGitActions.ts`
- `ui/src/hooks/useRunExplorer.ts`
- `ui/src/hooks/useWorkspaceSetup.ts`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Public API route shapes and error codes remain unchanged
- [x] Provider adapter repair specs preserve include-dir surfaces, qwen prompt-via-`-p`, codex stdin behavior and write-set guard ownership
- [x] Staged docflow still emits artifacts, citation index, final run index and semantic snapshot before existing promotion flow
- [x] Shard planner output remains deterministic and scheduler keeps fail-fast worker-pull semantics
- [x] Runtime task execution, shard summary/checkpoint persistence, run finalization, step handlers, artifact registry and pipeline execution state are split behind internal seams without public API changes
- [x] Baseline prompt packs and runtime prompt contracts reuse `artifactquality` policy wording for collect/validator contract text
- [x] UI run selection/log/artifact controls keep stable `data-testid` surfaces
- [x] `make contracts`, `make test`, `make lint`, `make build` pass

### Risks
- Internal seam extraction can accidentally change exact transient error text even when error code stays stable; API tests should prefer code/status assertions for public behavior.
- `make build` may refresh tracked embedded UI assets because `internal/api/ui_dist` is intentional release output.

### Progress log
- 2026-05-03: Implemented runtime profile patch service, pipeline run route helpers, shared focused repair command builder, docflow builder result, sharding planner/scheduler seams and UI `RunPanels` model/actions grouping. Targeted Go checks for API/orchestrator/runtime adapters passed; full DoD pending.
- 2026-05-03: Full verification passed: `make contracts`, `go test ./...`, `python3 -m unittest discover -s scripts/tests -p '*_test.py'`, `npm ci --prefix ui`, `npm run typecheck --prefix ui`, `npm run test --prefix ui -- --run`, `make test`, `make lint`, `make build`, `git diff --check`. `make build` refreshed tracked embedded UI assets in `internal/api/ui_dist`.
- 2026-05-03: Follow-up plan audit found missing internal seams from the requested plan. Added `RuntimeTaskExecutor`, `ShardSummaryStore`, embedded pipeline execution state groups and artifactquality-backed baseline/prompt wording reuse. Fresh verification passed: `make contracts`, `go test ./...`, `python3 -m unittest discover -s scripts/tests -p '*_test.py'`, `npm ci --prefix ui`, `npm run typecheck --prefix ui`, `npm run test --prefix ui -- --run`, `make test`, `make lint`, `make build`, `git diff --check`; `make build` refreshed tracked embedded UI assets.
- 2026-05-03: Second follow-up audit found that `run finalization`, `step handlers` and artifact registry methods were still physically in `orchestrator.go`, and quality metrics were grouped under runtime state. Moved these layers to `run_finalization.go`, `step_handlers.go`, `artifact_registry.go`, added `pipelineQualityState`, rebuilt `ast-index` (222 files / 8.1k symbols) and passed `go test ./internal/orchestrator`; full DoD rerun pending.
- 2026-05-03: Runtime profile patch service was still API-package local. Moved patch service, validation and merge logic to shared internal package `internal/runtimeprofile`, leaving API handlers as HTTP adapters; `go test ./internal/api ./internal/runtimeprofile` passed, full DoD rerun pending.
- 2026-05-03: UI hook split was still partial. Split run explorer into selection/polling/actions facade hooks and workspace setup into manifest/baseline/wizard/git hooks while preserving `useRunExplorer`/`useWorkspaceSetup` facade exports; `npm run typecheck --prefix ui` passed, full DoD rerun pending.
- 2026-05-03: Final verification passed after second follow-up: `make contracts`, `go test ./...`, `python3 -m unittest discover -s scripts/tests -p '*_test.py'`, `npm ci --prefix ui`, `npm run typecheck --prefix ui`, `npm run test --prefix ui -- --run`, `make test`, `make lint`, `make build`, `git diff --check`; `ast-index` rebuilt with 229 files / 8130 symbols. One parallel `npm ci` + Vitest race was rerun sequentially and passed. `make build` refreshed tracked embedded UI assets.
- 2026-05-03: Final plan audit found sharding ownership was still physically concentrated in `sharding.go` despite internal seams. Split coordinator, scheduler, summary/checkpoint store, shard artifacts and planner into dedicated files without behavior changes; `go test ./internal/orchestrator` passed, full DoD rerun pending.
- 2026-05-03: Final verification after sharding file split passed: `git diff --check`, `go test ./...`, `python3 -m unittest discover -s scripts/tests -p '*_test.py'`, `npm run typecheck --prefix ui`, `npm run test --prefix ui -- --run`, `make contracts`, `make test`, `make lint`, `make build`; `ast-index rebuild` indexed 234 files. `make build` refreshed tracked embedded UI assets and emitted only the existing Vite chunk-size warning for Mermaid bundles.
- 2026-05-03: Re-audit found baseline prompt generation still had direct copies of legacy-ban wording after `artifactquality` became canonical. Removed the remaining baseline-local copies for collect/findings/proposals prompt packs and proposal skill prompt generation; tests now assert the canonical `artifactquality` wording. Targeted `go test ./internal/workspace ./internal/artifactquality ./internal/runtime/promptcontract ./internal/runtime/steppolicy` passed with sandbox-local `GOCACHE`; full DoD rerun pending.
- 2026-05-03: Final verification after baseline prompt dedupe passed with sandbox-local caches: `git diff --check`, `go test ./...`, `npm run typecheck --prefix ui`, `npm run test --prefix ui -- --run`, `make contracts`, `make test`, `make lint`, `make build`; `make test` covered 110 script regression tests and 18 UI tests. `ast-index rebuild` indexed 234 files. `make build` refreshed tracked embedded UI assets and emitted only the existing Vite Mermaid chunk-size warning.
### Plan ID
EP-20260502-qwen-smoke-tiny-marker-collect-hardening

### Context
Diagnostic `smoke tiny` на `qwen-code` (`smoke-tiny-bank-20260501T172519Z`) снова подтвердил, что release/matrix selector корректный, но backend hard pass блокируется в `init.step1.collect`: root/src shards писали useful markdown без `shard-pack-manifest.json` и затем exhausting manifest-only repair (`runtime_contract_failed`), а `kubernetes-manifests` оставался fully silent/no-artifact (`runner_unavailable`). Дополнительно heuristic coverage roots для Bank of Anthos схлопывались с 26 roots до 6 top-level groups, превращая весь `src` в один тяжёлый collect shard.

### Goals (must have)
- [x] Усилить collect prompt до early pair-write: suggested overview doc + `shard-pack-manifest.json` до broad second-pass sweep
- [x] Сфокусировать manifest-only repair prompt: authored docs + exact JSON skeleton first, затем compact constraints/canonical reference
- [x] Сохранить strict artifact-only policy: ACP не синтезирует manifest и не нормализует provider artifacts
- [x] Сделать structural coalescing marker-aware: сохранять module marker leaf shards внутри крупных top-level dirs, пока итоговый план остаётся в `maxAutoShardsPerRepo`
- [x] Добавить regression coverage для prompt contracts, qwen repair command spec и marker-preserving sharding
- [x] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [x] Выполнить trusted-machine `smoke tiny` qwen diagnostic и зафиксировать verdict/evidence

### Non-goals
- [x] Не менять public schemas/API/report formats/provider IDs
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`
- [x] Не менять `best_effort partial` semantics: partial downstream может продолжаться, но terminal run остаётся failed

### Approach
1) Обновить shared collect step policy: early pair-write wording, tiny-smoke target shape, manifest skeleton remains provider instruction only.
2) Переставить repair prompt вокруг exact skeleton-first flow и заменить длинный repair hint tail компактными repair constraints.
3) В sharding planner вынести discovery marker leaves в reusable helper и раскрывать top-level coalesced groups на marker leaves + residual group только в пределах shard cap.
4) Добавить targeted tests для prompt order/content, qwen repair args и marker-preserving coalescing.
5) Синхронизировать README/architecture/runbook/testing docs и затем прогнать DoD + live diagnostic.

### Files expected to change
- `internal/runtime/{steppolicy,promptcontract,qwencode}/*`
- `internal/orchestrator/sharding*`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Collect prompt contains explicit early pair-write requirement
- [x] Repair prompt remains manifest-only, lists authored docs and puts exact JSON skeleton before repair constraints
- [x] Qwen repair command still uses `-p` prompt and empty stdin
- [x] Marker-leaf code shards are preserved under coalescing without exceeding `maxAutoShardsPerRepo`
- [x] DoD passes
- [x] `smoke tiny` qwen rerun has `release_verdict_<matrix-id>.json.verdict == "PASS"` or records a narrower residual provider blocker

### Risks
- Marker-aware coalescing can increase shard count for large repos. Mitigation: expansion is bounded by `maxAutoShardsPerRepo`; top-level groups that do not fit stay coalesced and log a warning.
- Prompt hardening can still fail under provider auth/rate-limit/CLI regressions. Those remain explicit `runner_unavailable` incidents with raw diagnostics.

### Progress log
- 2026-05-02: Started implementation after triage of `smoke-tiny-bank-20260501T172519Z`; added early pair-write prompt wording, compact skeleton-first repair prompt, marker-preserving structural coalescing, and targeted regression tests.
- 2026-05-02: DoD passed: `make contracts`, `make test`, `make lint`, `make build`.
- 2026-05-02: Ran direct `scripts/full-run-batch-matrix.sh` diagnostic as `smoke-tiny-bank-qwen-hardening-20260502T055955Z`; verdict `FAIL`/`RELEASE BLOCKED`, release contract selector passed (`qwen-code`, run `1`, `baseline`, snapshot artifacts, frontend skipped) but backend hard pass stayed `0/1` with `runtime_contract_failed=1` and `runner_unavailable=1`.
- 2026-05-02: Live evidence confirmed marker-preserving coalescing worked (`26` coverage roots -> `10` shard groups, including `4` preserved module marker leaves); collect improved from the prior `3/6` pass shape to `4/10` succeeded shards, but residual qwen stalls remained: markdown-only collect shards still exhausted manifest-only repair with empty stdout/stderr, `iac` had no artifacts after retry, and `step3.findings` ended `runner_unavailable`.
### Plan ID
EP-20260503-breaking-complexity-cleanup

### Context
Аудит текущего кода показал несколько подтверждённых источников сложности: unused legacy `repos[].analysis.role` в workspace contract, недостижимые `repo_selection` ветки в orchestrator, ad hoc legacy collect-manifest scanner поверх уже strict schemas, provider metadata через adapter type-switch imports, монолитные run/shard/providercommon flows и дублированные headless test stubs. Срез намеренно breaking: старые manifests/artifacts с legacy полями должны fail-fast через schema/contract validation без migration shims.

### Goals (must have)
- [x] Удалить `repos[].analysis.role` из workspace schema/parser/examples/docs
- [x] Удалить недостижимое repo-selection состояние из orchestrator
- [x] Перенести known legacy collect alias rejection в schema-level constraints
- [x] Сократить сложность run finalization, sharded execution и domain collect preparation
- [x] Добавить runner metadata seam и убрать provider adapter imports из orchestrator
- [x] Разнести `providercommon` engine по focused файлам без изменения lifecycle semantics
- [x] Дедуплицировать Go headless runner test stubs и синхронизировать UI/docs/tests

### Non-goals
- [ ] Не добавлять backward compatibility/migration shims для удалённых legacy fields
- [ ] Не расширять provider set, hosted mode, security/compliance или public API endpoints
- [ ] Не делать live provider matrix required CI gate

### Approach
1) Обновить workspace schema/spec/examples/parser/tests под removal `analysis.role`.
2) Убрать repo-selection fields/branches из `pipelineExecution`, сохранив all-repos behavior.
3) Ужесточить shard-pack schema для known alias fields и удалить raw legacy scanner.
4) Вынести helpers вокруг run finalization, domain collect preparation и sharded scheduler/apply flow.
5) Добавить runtime metadata interface на runners и разнести providercommon engine internals по файлам.
6) Перенести повторяющиеся headless runner stubs в `internal/testutil`, обновить UI helpers/docs.
7) Прогнать contract, targeted и full DoD checks.

### Files expected to change
- `schemas/*`, `examples/*`, `fixtures/*`
- `internal/workspace/*`, `internal/contracts/*`, `internal/artifactquality/*`
- `internal/orchestrator/*`, `internal/runtime/*`, `internal/testutil/*`
- `ui/src/*`
- `README.md`, `docs/*`

### Acceptance criteria
- [x] `analysis.role` rejected as unsupported schema field
- [x] Legacy collect aliases rejected by schema/contract validation
- [x] Sharded execution preserves deterministic all-repo behavior
- [x] Orchestrator no longer imports provider adapter packages for metadata
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`

### Risks
- Breaking schema change can invalidate user workspaces that still contain `analysis.role`; this is intentional for the cleanup.
- Moving rejection from raw scanner to schema can change exact error text; tests should assert actionable contract failure, not brittle full messages.

### Progress log
- 2026-05-03: Started implementation from requested Breaking Complexity Cleanup Plan.
- 2026-05-03: Removed `analysis.role`, repo-selection state, raw collect legacy scanner, provider metadata type-switching; split sharded scheduler/finalization/domain collect prep and providercommon internals; moved repeated script stubs into `internal/testutil`; UI fetch details moved to typed API helpers. Targeted Go checks are green; full DoD pending dependency bootstrap.
- 2026-05-03: `make bootstrap`, `make contracts`, `make test`, `make lint` и `make build` passed. `make build` refreshed embedded UI assets in `internal/api/ui_dist`.
- 2026-05-03: Follow-up audit closed remaining plan gaps: command process execution and artifact recovery moved out of `providercommon/engine.go`, UI hook state moved behind focused reducers/actions, and explicit regressions now cover all-workspace repo scopes plus deterministic sharded scheduler result ordering. Re-ran `make contracts`, `make test`, `make lint`, `make build`, and `git diff --check` successfully.
- 2026-05-03: Final checklist audit found one stale active-doc reference to removed `repo-selection-summary`; synchronized `docs/ARCHITECTURE.md` with all-repo direct workspace scope behavior. Re-ran full requested verification: `make bootstrap`, `make contracts`, targeted Go tests, UI typecheck/tests, `make test`, `make lint`, and `make build`.
- 2026-05-03: Final bug sweep fixed stale run artifact UI state when manually selecting another run; `handleSelectRun` now clears artifacts before loading the new run. Re-ran UI typecheck/tests, `make test`, `make lint`, and `make build`.
- 2026-05-03: Additional scheduler audit found a fail-fast race where sequential sharded execution could dispatch one extra shard after the first failure. Reworked `scheduleRuntimeShardRuns` to worker-pull scheduling and added a regression that keeps the next sequential shard undispatched after fail-fast terminal error.
### Plan ID
EP-20260501-qwen-smoke-tiny-collect-manifest-hardening

### Context
Diagnostic `smoke tiny` на `qwen-code` (`smoke-tiny-bank-20260501T093108Z`) дошёл до backend run, но завершился `FAIL`: один collect shard прошёл, четыре shards написали authored markdown без `shard-pack-manifest.json`, manifest-only repair stalled без manifest, а root-file shard остался fully silent/no-artifact и был корректно classified как `runner_unavailable`. Нужно стабилизировать qwen invocation/prompt surface без ослабления strict artifact-only validation и без изменения public schemas/API.

### Goals (must have)
- [x] Убрать JSON task stdin из `qwen-code` invocation; prompt остаётся только в CLI `-p`, включая custom qwen args
- [x] Усилить collect manifest repair prompt task-specific scaffold-ом и literal JSON skeleton
- [x] Добавить root-file collect shard hint против recursive sweep и silent/no-artifact hang
- [x] Сохранить `runtime_contract_failed` для markdown-only collect после одной repair попытки и `runner_unavailable` для fully silent qwen retry exhaustion
- [x] Синхронизировать docs/runbook/testing и покрыть tests

### Non-goals
- [x] Не менять public schemas/API/workspace contracts
- [x] Не добавлять ACP-side manifest synthesis/autofill
- [x] Не менять `best_effort partial` orchestration semantics

### Approach
1) Перевести `qwencode` command specs на prompt-only stdin-empty invocation.
2) Расширить shared prompt contract: repair prompt перечисляет authored docs, exact metadata, repo/path scopes, evidence candidates и task-specific manifest JSON skeleton.
3) Расширить collect step policy: root-file shards читают только перечисленные root files, пишут один concise overview doc и manifest; все collect shards получают suggested doc path и manifest skeleton до canonical example.
4) Добавить adapter/prompt/policy tests и синхронизировать docs.
5) Прогнать targeted tests, DoD и затем clean-worktree `smoke tiny` qwen diagnostic.

### Files expected to change
- `internal/runtime/qwencode/*`
- `internal/runtime/{promptcontract,steppolicy}/*`
- `docs/*`, `README.md`

### Acceptance criteria
- [x] `qwen-code` command specs keep stdin empty and pass artifact prompt through CLI `-p`
- [x] Repair prompt exposes task-specific manifest scaffold, literal JSON skeleton and no broader schema/history scavenging
- [x] Root-file collect shard prompt forbids recursive repo sweep
- [x] Targeted runtime tests pass
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`
- [x] Trusted-machine `smoke tiny` qwen rerun captured with backend `1/1` or residual blocker evidence

### Risks
- Live provider behavior can still fail due auth/rate limits or qwen CLI changes; such failures must remain explicit `runner_unavailable` with raw diagnostics.
- Prompt hardening may reduce broad context collection on root-file shards; remaining uncertainty should be represented as coverage gaps, not hidden success.

### Progress log
- 2026-05-01: Started implementation after `smoke-tiny-bank-20260501T093108Z` log triage.
- 2026-05-01: Implemented qwen prompt-only invocation, collect repair scaffold, root-file shard hints, stub test parsing for prompt-only qwen, and synchronized README/architecture/spec/runbook/testing docs. Targeted runtime tests and full DoD passed.
- 2026-05-01: Final audit found that custom `HeadlessRunner.Args` could drop the qwen artifact prompt after stdin removal, or keep a caller-supplied prompt. Fixed qwen args normalization so custom args still receive exactly one artifact prompt through CLI `-p`/`--prompt`, with stdin empty; added adapter coverage and synchronized docs.
- 2026-05-01: Final audit also found the root-file hint detector was too broad for multiple top-level directory scopes such as `docs, src` and service dirs like `.github`. Tightened it so root-file mode requires every scope to look like a root-level file, and added regression coverage.
- 2026-05-01: Follow-up audit found collect repair only counted/listed top-level authored files even though `documents[].path` may be nested. Made collect authored-file snapshots and repair prompt document discovery recursive under `write_root`, while still excluding runtime metadata/manifest files and keeping manifest-only write-set guard unchanged.
- 2026-05-01: Added a skeleton parse regression and fixed duplicate scaffold IDs when authored docs share the same basename in different directories, e.g. `overview.md` and `docs/overview.md`.
- 2026-05-01: Clean-worktree diagnostic rerun `smoke-tiny-bank-qwen-fix-20260501T105754Z` used qwen `0.15.2`, selected only `qwen-code` run `1`, frontend `never`, and did not skip precheck. Verdict stayed `FAIL`: backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=1`, frontend skipped/non-blocking. Evidence: `/tmp/provenarch-test_arch_project/reports/release_verdict_smoke-tiny-bank-qwen-fix-20260501T105754Z.json` and `/tmp/provenarch-test_arch_project/reports/profile_matrix_smoke-tiny-bank-qwen-fix-20260501T105754Z.md`.
- 2026-05-01: Live residuals narrowed: `docs` collect shard now wrote both authored doc and `shard-pack-manifest.json`; root-file shard still exhausted as silent/no-artifact `runner_unavailable`, `extras` hit qwen CLI/API TLS `runner_unavailable`, and `iac`/`kubernetes-manifests`/`src` wrote markdown but manifest-only repair still stalled as `runtime_contract_failed`. Step2 then aborted as silent/no-artifact `runner_unavailable`.
- 2026-05-01: Revalidated current diff from a clean temp clone with installed UI deps using direct harness matrix `smoke-tiny-bank-20260501T121025Z`. Precheck passed, selected provider/run were `qwen-code`/`1`, frontend init/cancel were skipped by `never`, and verdict stayed `FAIL`: backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=1`, `precheck_failed=0`. Collect improved to one valid shard (`bank-of-anthos-src`) and five authored-docs-without-manifest shards that exhausted manifest-only repair; downstream `step2.asis_docs` assembled from the one valid shard, then `step3.findings` failed as silent no-artifact `runner_unavailable`.
- 2026-05-01: Follow-up audit found the remaining prompt gap: qwen still wrote authored docs before manifest. Added suggested authored doc paths plus literal task-specific manifest JSON skeletons to normal collect and manifest-only repair prompts; this remains provider instruction only, with no ACP-side manifest synthesis/autofill.
- 2026-05-01: Clean temp-clone rerun `smoke-tiny-bank-20260501T131747Z` used direct `./scripts/full-run-batch-matrix.sh` with qwen `0.15.2`, selected only `qwen-code` run `1`, frontend init/cancel `never`, and no `BATCH_SKIP_PRECHECK`. Verdict stayed diagnostic `FAIL`: backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=1`, `precheck_failed=0`, frontend skipped. The new manifest skeleton improved collect to two valid shard packs (`bank-of-anthos-docs`, `bank-of-anthos-extras`); root-file shard remained fully silent/no-artifact `runner_unavailable`; `iac`, `kubernetes-manifests`, and `src` wrote overview markdown but exhausted manifest-only repair without `shard-pack-manifest.json`; `step2.asis_docs` recovered on fresh retry; `step3.findings` then exhausted as silent/no-artifact `runner_unavailable`. Evidence: `/tmp/provenarch-test_arch_project/reports/release_verdict_smoke-tiny-bank-20260501T131747Z.json`, `/tmp/provenarch-test_arch_project/reports/profile_matrix_smoke-tiny-bank-20260501T131747Z.md`, and `/tmp/provenarch-test_arch_project/reports/quality_report_smoke-tiny-bank-20260501T131747Z-single-git-url-baseline.md`.
### Plan ID
EP-20260426-strict-runtime-no-compatibility-shims

### Context
После выравнивания live provider adapters в runtime остались compatibility-шымы, которые могли молча переписать provider artifacts после выполнения: collect manifest path/metadata canonicalization и draft file reconciliation из `outputs[].canonical_path`. Обратная совместимость с такими malformed artifacts больше не требуется; success source of truth должен быть strict artifact-only validation.

### Goals (must have)
- [x] Удалить active compatibility registry и rule-id diagnostics
- [x] Сделать collect validation read-only: без autofill metadata и без `documents[].path` normalization
- [x] Сделать draft validation read-only: без копирования draft files из `outputs[].canonical_path` в `outputs[].path`
- [x] Сохранить только manifest-only provider repair для collect shards с authored docs + missing/invalid manifest
- [x] Перенести deterministic fake runtime в provider-neutral package
- [x] Переименовать child batch harness в нейтральное имя без wrapper для старого пути
- [x] Обновить tests/docs под no-compat behavior

### Non-goals
- [x] Не менять public artifact schemas
- [x] Не добавлять backward-compat wrapper для старого имени batch script

### Approach
1) Заменить локальный collect repair/canonicalization на strict read-only manifest validation с legacy precheck.
2) Удалить draft-root reconciliation path и tests, которые ожидали compatibility mutations.
3) Обновить provider adapters так, чтобы qwen не имел отдельной repair-named artifact validation обёртки.
4) Перенести fake runtime из `claudecode` в `fakeruntime`, сохранив deterministic artifacts.
5) Переименовать child batch harness с legacy имени на `full-run-batch.sh` и синхронизировать matrix harness/docs/tests.
6) Добавить engine write-set guard для manifest-only collect repair: всё кроме `shard-pack-manifest.json` остаётся contract failure.
7) Синхронизировать docs/spec/testing с strict no-compat runtime behavior.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/providercommon/*`
- `internal/runtimedrafts/*`
- `internal/runtime/qwencode/*`
- `internal/runtime/fakeruntime/*`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `docs/*`

### Acceptance criteria
- [x] Artifact validation never rewrites collect manifests or draft files
- [x] Artifact-root-prefixed/absolute collect document paths fail strict validation
- [x] Draft files written only at `outputs[].canonical_path` fail strict validation
- [x] Manifest-only provider repair remains available and engine-enforced to write only `shard-pack-manifest.json`
- [x] Deterministic fake runtime is no longer implemented inside `claudecode`
- [x] Active docs/tests/scripts no longer reference the old child batch script name
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`

### Risks
- Existing live providers may fail more often after hidden normalization is removed. That is intentional: failures should surface as `runtime_contract_failed` and be fixed via prompt/adapter behavior, not post-hoc mutation.

### Progress log
- 2026-04-26: Implemented strict collect/draft validation, removed compatibility registry/reconciliation shims, moved deterministic fake runtime to `fakeruntime`, renamed the child batch harness, added collect repair write-set guard, updated tests/docs, and completed full DoD plus `git diff --check`.
### Plan ID
EP-20260426-live-provider-collect-contract-stabilization

### Context
Smoke tiny live triage показал общий collect-contract failure surface: `qwen-code`/`claude-code` доходят до `init.step1.collect`, но оставляют `shard-pack-manifest.json` missing/invalid, а `codex-code` после обновления CLI снова должен участвовать как полноценный peer. Success source остаётся artifact-only; stdout/stderr являются diagnostics.

### Goals (must have)
- [x] Добавить общий manifest-only repair path для collect steps после authored docs + missing/invalid `shard-pack-manifest.json`
- [x] Расширить artifact-state diagnostics: manifest state, authored artifact count, raw stdout/stderr refs
- [x] Сузить qwen `runner_unavailable` до fully silent/no-artifact paths; partial artifacts без валидного manifest остаются `runtime_contract_failed`
- [x] Сохранить thin adapters для `claude-code`, `qwen-code`, `codex-code`
- [x] Добавить selected-provider readiness guard, включая codex `gpt-5.5`/CLI version mismatch
- [x] Обновить docs/spec/testing/runbook/live-e2e skill

### Non-goals
- [x] Не менять product API, workspace schema, public artifact schemas или release matrices
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`
- [x] Не расширять MVP provider set

### Approach
1) В `providercommon` добавить optional collect repair adapter interface, shared diagnostics и классификацию partial collect artifacts как contract failure.
2) Подключить manifest-only repair prompt ко всем live adapters; qwen сохраняет fresh retry только для no-artifact/missing-invalid paths.
3) Усилить collect prompt policy против markdown-only completion.
4) В batch preflight записывать selected provider readiness и блокировать известный codex model/version mismatch до deep run.
5) Синхронизировать docs/skill и покрыть runtime/preflight tests.

### Files expected to change
- `internal/runtime/providercommon/*`
- `internal/runtime/{claudecode,codexcode,qwencode,promptcontract,steppolicy}/*`
- `scripts/full-run-batch.sh`
- `scripts/write-batch-preflight.py`
- `docs/*`, `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] Unit tests cover manifest-only collect repair success/failure
- [x] qwen partial collect artifacts without valid manifest classify as `runtime_contract_failed`
- [x] qwen no-output/no-artifact retry exhaustion remains `runner_unavailable`
- [x] codex `0.125.0` + `gpt-5.5` passes readiness guard; old `0.118.0` is blocked
- [x] Top-level `release_verdict_*.json.backend` aggregate exists for canonical acceptance checks
- [x] `claude-code`/`codex-code` adapters use shared pre-artifact stall monitoring instead of waiting for full hard timeout on silent/no-artifact hangs
- [x] Full DoD: `make contracts`, `make test`, `make lint`, `make build`
- [x] Trusted-machine smoke tiny rerun for `qwen-code` captured residual live collect repair failure
- [x] Trusted-machine smoke tiny rerun for `codex-code` confirmed updated CLI readiness and the same collect-manifest residual on `bank-of-anthos-extras`
- [x] Trusted-machine smoke tiny rerun for `claude-code`
- [x] Post-tightening trusted smoke tiny rerun for `qwen-code`
- [x] Post-pre-artifact-monitor trusted smoke tiny rerun for `codex-code`/`claude-code`

### Risks
- Manifest-only repair can still fail if provider wrote no authored docs at all; that must stay explicit `runtime_contract_failed`/`runner_unavailable`, not be hidden as artifact quality.
- Provider auth/rate-limit remains operational and must surface as `runner_unavailable` with raw diagnostics.

### Progress log
- 2026-04-26: Implemented shared collect manifest repair path, adapter repair prompts, artifact-state diagnostics, qwen partial-artifact classification guard, codex readiness guard, and targeted tests. Full DoD and live reruns pending.
- 2026-04-26: Full DoD passed. Trusted `qwen-code` smoke tiny `smoke-tiny-bank-qwen-20260426T104225Z` failed as expected on residual live behavior: authored collect docs were present, manifest-only repair stalled without producing `shard-pack-manifest.json`; retry partial-artifact classification was tightened after this run.
- 2026-04-26: Trusted `codex-code` smoke tiny `smoke-tiny-bank-codex-20260426T112719Z` passed selected-provider readiness on `codex-cli 0.125.0`/`gpt-5.5`, then failed as `runtime_contract_failed` on `bank-of-anthos-extras`: authored `extras-overview.md` existed, but manifest-only repair still stalled without `shard-pack-manifest.json`. The residual exposed repair-surface drift, so repair include dirs were narrowed to current `write_root` + repo evidence, repair prompt now treats embedded schema text as authoritative, and repair watchdog uses a bounded 90s repair window instead of the normal 20s post-artifact stall.
- 2026-04-26: Post-tightening `qwen-code` smoke tiny `smoke-tiny-bank-qwen-20260426T130440Z` completed with verdict `FAIL`, mode `non-release`, selected provider `qwen-code`, run index `1`, backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=0`, `runtime_timeout=0`; frontend `never` remained skipped/non-blocking. Task logs confirmed manifest-only repair scheduled/exhausted per partial collect shard.
- 2026-04-26: During post-tightening `codex-code` smoke tiny `smoke-tiny-bank-codex-20260426T135500Z`, updated CLI readiness passed and initial step/first collect shards succeeded, but `bank-of-anthos-extras` exposed a shared lifecycle gap: `claude-code`/`codex-code` lacked pre-artifact stall monitoring for silent/no-artifact hangs. The diagnostic run was terminated after capturing the issue; adapters now use shared artifact-step pre-artifact monitoring and release verdict JSON now has a top-level backend aggregate for canonical acceptance checks.
- 2026-04-26: Post-pre-artifact-monitor smoke tiny reruns completed for all three live providers: `codex-code` `smoke-tiny-bank-codex-postfix-20260426T144025Z` (`FAIL`, non-release, backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=0`, `runtime_timeout=0`), `claude-code` `smoke-tiny-bank-claude-postfix-20260426T150624Z` (`FAIL`, non-release, backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=0`, `runtime_timeout=0`), and `qwen-code` `smoke-tiny-bank-qwen-postfix-20260426T152149Z` (`FAIL`, non-release, backend `0/1`, `runtime_contract_failed=1`, `runner_unavailable=1`, `runtime_timeout=0`). The qwen `runner_unavailable` came from a later silent/no-artifact non-collect path; collect partial artifacts remained classified as `runtime_contract_failed`.
### Plan ID
EP-20260425-provider-runtime-adapter-alignment

### Context
Qwen live smoke показал, что provider уже успевает записать валидные artifacts, но старый qwen-only watchdog мог убить процесс и классифицировать artifact-only success как `runner_unavailable`. У `claude-code` и `codex-code` не было такого же watchdog, но был общий риск: разные process lifecycle paths при одинаковом artifact-only runtime contract.

### Goals (must have)
- [x] Вынести process lifecycle для `claude-code`, `qwen-code`, `codex-code` в общий `providercommon` engine
- [x] Оставить provider differences в adapters: command/args/stdin/workdir, unavailable markers, activity/recovery policy
- [x] Принять valid artifacts after controlled stop как success
- [x] Убрать qwen dependency на `--output-format json`
- [x] Сохранить `qwen` silent missing-artifact/retry exhaustion как `runner_unavailable`, а malformed artifacts как `runtime_contract_failed`
- [x] Исправить non-release frontend `never` verdict semantics без ослабления release-mode strict frontend checks

### Non-goals
- [x] Не менять product API, workspace schema или public artifact schemas
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`
- [x] Не расширять MVP provider set

### Approach
1) Добавить shared `providercommon` process engine с raw diagnostics, activity monitor, controlled stop/retry и artifact validation.
2) Переключить `claudecode`, `qwencode`, `codexcode` на thin adapters поверх engine и удалить qwen-only process executor.
3) Добавить conformance tests для artifact-only success/failure/timeout/unavailable paths и provider args tests.
4) Синхронизировать architecture/spec/testing/runbook/live-e2e skill.

### Files expected to change
- `internal/runtime/providercommon/*`
- `internal/runtime/{claudecode,codexcode,qwencode}/*`
- `scripts/full-run-batch-matrix.sh`
- `docs/*`, `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] `go test ./internal/runtime/...`
- [x] targeted matrix release-contract tests for non-release frontend `never` and release strict frontend blockers
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] Trusted-machine smoke tiny qwen rerun executed; verdict remained blocked by live `qwen-code` partial/missing artifact behavior (`runner_unavailable`), with frontend `never` correctly non-blocking

### Risks
- Live CLI behavior can still fail due external auth/rate limits; such failures must stay explicit `runner_unavailable` with raw-output refs.
- Release-mode frontend checks must remain strict even though non-release diagnostic `frontend=never` is now non-applicable.

### Progress log
- 2026-04-25: Implemented shared provider engine/adapters, removed qwen-only process executor, fixed qwen output args, added conformance tests, fixed non-release frontend-never verdict semantics, and synchronized docs/runbook/skill.
- 2026-04-25: Follow-up audit closed residual gaps: adapter contract now exposes include dirs, `claude`/`codex` adapter conformance tests cover noninteractive diagnostic flags and shared unavailable markers, and shared engine now applies explicit adapter retry policy after normal-exit missing/invalid artifact validation failures.
- 2026-04-25: Full local DoD passed (`make contracts`, `make test`, `make lint`, `make build`). Trusted smoke tiny qwen diagnostic `smoke-tiny-bank-20260425T133718Z` produced reports but failed with backend `runner_unavailable`; raw diagnostics show qwen left five collect shards partial/missing while valid artifacts-after-stop were accepted for the first shard.
- 2026-04-25: Final defect audit fixed retry classification edge case: an initial structurally malformed artifact contract cannot be masked by a later silent retry as `runner_unavailable`; added regression coverage and reran full DoD.
### Plan ID
EP-20260425-flexible-live-e2e-selector

### Context
Live E2E surface сейчас имеет canonical release taxonomy и low-level env selectors (`BATCH_PROVIDER_FILTER`, `BATCH_RUN_SELECTION`), но нет удобного catalog-driven способа получить прямые команды для комбинаций вроде `regres + codex + fast`, `regres + claude + full` или супербыстрого `1 repo × 1 run × 1 provider` smoke. Нужно добавить этот слой без превращения его в wrapper поверх release harness и без изменения official release verdict contract.

### Goals (must have)
- [x] Добавить command generator, который только печатает direct `scripts/full-run-batch-matrix.sh` commands
- [x] Добавить `smoke tiny` selector (`bank-of-anthos`, one provider, one run, frontend off)
- [x] Добавить diagnostic `regres full` selector на все 6 canonical repo sets, включая Sentry
- [x] Сохранить canonical release taxonomy и release verdict source unchanged
- [x] Явно зафиксировать artifact quality как обязательный gate для regress/release
- [x] Синхронизировать runbook/testing strategy/live-e2e skill
- [x] Прогнать targeted tests и DoD (`make contracts`, `make test`, `make lint`, `make build`)

### Non-goals
- [x] Не добавлять executable wrapper, который сам запускает release matrix
- [x] Не менять public schemas/product APIs
- [x] Не делать standalone artifact-eval command в этом slice

### Approach
1) Расширить `examples/e2e-profile-catalog.yaml` отдельным `selectors[]` слоем, не меняя canonical `profiles[]`.
2) Добавить diagnostic matrix files для `smoke tiny` и Sentry baseline.
3) Реализовать `scripts/live-e2e-plan.py` с выводом `shell|json|markdown`.
4) Покрыть selector normalization, provider shorthand, release subset rejection, run counts и direct command output unit tests.
5) Обновить docs/skill и выполнить DoD.

### Files expected to change
- `scripts/live-e2e-plan.py`
- `scripts/tests/live_e2e_plan_test.py`
- `examples/e2e-profile-catalog.yaml`
- `examples/e2e-matrix.smoke-tiny.bank.yaml`
- `examples/e2e-matrix.diagnostic.sentry.yaml`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `.agents/skills/e2e-live-gate/SKILL.md`

### Acceptance criteria
- [x] `smoke tiny` generates exactly one backend run for exactly one provider
- [x] `regres full` is non-release and includes all six canonical repo sets
- [x] `release fast|long|full` generated commands keep all three release providers
- [x] Provider subset is rejected for release selectors
- [x] Full release provider set is accepted in any CLI order and normalized to canonical order
- [x] `frontend-mode=never` disables both init and cancel frontend smoke in generated commands
- [x] Generated regress/release metadata declares existing artifact-quality gate path

### Risks
- Главный риск: оператор может принять generated diagnostic selector за official release verdict. Mitigation: docs/skill/script metadata помечают `smoke`/`regres full` как diagnostic/non-release, а release readiness остаётся только `release_verdict_<matrix-id>.json`.

### Progress log
- 2026-04-25: Started implementation of catalog-driven live E2E command generator, smoke tiny, diagnostic regres full, docs/skill sync, and unit tests.
- 2026-04-25: Added generator/catalog/matrix/docs/skill changes and passed targeted tests, full `scripts/tests` discovery, and DoD (`make contracts`, `make test`, `make lint`, `make build`).
- 2026-04-25: Audit fixed release provider ordering bug (`codex,qwen,claude` now accepted as the full release provider set) and added regression coverage.
- 2026-04-25: Follow-up audit fixed `frontend-mode=never` command generation to set `BATCH_FRONTEND_CANCEL_MODE=never` as well, and tightened release provider diagnostic guidance.
### Plan ID
EP-20260421-codex-runtime-provider

### Context
Нужно расширить MVP runtime/provider surface и trusted-host live release gate новым headless provider `codex-code`, сохранив deterministic `fake` baseline, default fallback `claude-code`, 5-profile taxonomy и shard-plan invariant между `baseline` и `parallel-default`.

### Goals (must have)
- [x] Добавить `codex-code` в runtime/config contracts, CLI/API/workspace validation и runner factory
- [x] Реализовать artifact-only headless runner через `codex exec` без codex-specific retry/watchdog policy
- [x] Расширить live batch harness/reporting/release verdict до `qwen-code + claude-code + codex-code`
- [x] Синхронизировать schema/spec/docs/examples/ADR и live runbook
- [x] Обновить Go и Python/script tests под новый provider

### Non-goals
- [x] Не добавлять hosted mode
- [x] Не менять default provider с `claude-code`
- [x] Не добавлять wrapper-скрипты поверх canonical matrix harness
- [x] Не добавлять wrapper-скрипты поверх canonical matrix harness

### Approach
1) Завершить runtime/config slice: provider enum/parser, runner factory, Codex runner, workspace/API validation и schema surface.
2) Расширить live harness/reporting/preflight и release strict gate без большого generic refactor.
3) Синхронизировать ADR/docs/examples/fixtures и пересчитать release catalog totals.
4) Прогнать целевые Go и Python/script tests, затем при возможности `make contracts`, `make test`, `make lint`, `make build`.

### Files expected to change
- `internal/runtime/*`
- `internal/workspace/*`
- `internal/api/*`
- `cmd/acp/*`
- `schemas/*`
- `scripts/full-run-batch-matrix.sh`
- `scripts/full-run-batch.sh`
- `scripts/frontend-live-e2e.sh`
- `scripts/write-batch-preflight.py`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/*`
- `examples/*`

### Acceptance criteria
- [x] `codex-code` accepted everywhere `claude-code|qwen-code` were previously required
- [x] Release-mode matrix expects all three providers and reports frontend/backend verdicts for Codex
- [x] Regression profiles remain qwen-only unless manually filtered otherwise
- [x] Docs/runbook/ADR/examples remain synchronized with schema and tests

### Risks
- Основной риск — оставить несовпадение между runtime surface и trusted-host live gate/reporting. Снижение риска: держать provider allowlists/order explicit, расширять report fields инкрементально (`frontend_codex_status`, `frontend_cancel_codex_status`, `runtimes.codex`) и проверять это script tests.

### Progress log
- 2026-04-21: started implementation; runtime provider enum/factory/Codex runner/tests partially wired, remaining work is harness/reporting/docs synchronization.
- 2026-04-21: runtime/config, harness/reporting, docs/examples/ADR and Go/Python test slices completed; `make contracts`, `make test`, `make lint`, `make build` passed on the implementation tree.
- 2026-04-21: post-implementation audit found residual runbook drift (`release` totals and frontend three-provider acceptance wording); synchronized the runbook and started trusted-host live preflight.
- 2026-04-21: repaired canonical `/tmp/provenarch-live-e2e` path checkouts to valid pinned-SHA git heads without changing curated matrices, created a detached clean verification worktree from the implementation snapshot, and launched manual `codex-code` regression smoke through `scripts/full-run-batch-matrix.sh`.
- 2026-04-21: reran DoD on an isolated git-backed snapshot of the current working tree and attempted canonical `release fast`; the new `codex-code` path advanced through `init.step1.collect` and materialized shard outputs inside the canonical harness, while overall release verification remained blocked by trusted-host provider issues outside this slice (`claude-code` returned quota/auth `403`, `qwen-code` failed live draft contract on `single-git_url`).
