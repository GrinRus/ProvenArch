# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон, текущие активные планы и инженерный operational mirror.

Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
- `docs/archive/PLANS_ARCHIVE_2026-05.md`
- `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Этот файл остаётся рабочим engineering mirror и active ExecPlan surface.

## Когда использовать
Используйте ExecPlan, если:
- работа затрагивает несколько модулей, или
- ожидаемое время > 30–60 минут, или
- затрагиваются контракты/схемы.

---

## Шаблон ExecPlan

### Plan ID
EP-YYYYMMDD-<slug>

### Context
Зачем это нужно? Какие ограничения важны?

### Goals (must have)
- [ ] ...

### Non-goals
- [ ] ...

### Approach
1) ...
2) ...
3) ...

### Files expected to change
- ...

### Acceptance criteria
- [ ] Тесты обновлены/добавлены
- [ ] Схемы валидируются
- [ ] Документация обновлена

### Risks
- ...

### Progress log
- YYYY-MM-DD: ...

---

## Active Plans
### Plan ID
EP-20260506-regres-small-live-bugfix-matrix

### Context
Live `regres fast` small на `qwen-code` и `claude-code` от 2026-05-06 дал подтверждённые runtime/product blockers:
`regres-fast-bank-openedx-20260506T052600Z` и `regres-fast-openstack-20260506T112333Z`.
Основные failure classes: silent no-artifact provider stalls, `runtime_contract_failed` на draft repair, incomplete collect semantic shape,
`analysis:cross-repo-missing`, productive frontend `active_run_timeout`, provider/model attribution drift и dependent frontend `snapshot_reports_missing`.

### Goals (must have)
- [x] Усилить selected-provider readiness и unavailable diagnostics для `qwen-code` без обхода artifact-only success contract
- [x] Привести `claude-code` missing/no-artifact recovery lane к shared provider behavior, сохранив malformed/partial artifacts как `runtime_contract_failed`
- [x] Ужесточить draft artifact repair: manifest + files под `draft_final_root`, absolute `test -s` checks, validation remains final authority
- [x] Синхронизировать collect prompt checklist/repair hints с canonical semantic required fields
- [x] Усилить multi-repo semantic prompt/report details для `analysis:cross-repo-missing`
- [x] Согласовать frontend init timeout с effective timeout profile и сохранить `active_run_timeout` как отдельный reason
- [x] Добавить provider/model audit diagnostics в runtime/preflight/reporting
- [x] Добавить fail-fast selected-provider artifact smoke до batch/matrix, чтобы silent host/provider failures становились `operational_host_preflight_failed`
- [x] Гарантировать terminal run-history cleanup для CLI timeout/cancel/error/panic и harness fallback reconciliation
- [x] Изолировать frontend cancel smoke от stale cloned run-history/logs/raw и передавать симметричные provider command envs
- [x] Добавить runtime repair/stall/partial counters в quality summary, batch reports и release verdict aggregation
- [x] Закрепить dependent `snapshot_reports_missing` after backend terminal failure как skipped/blocked evidence
- [ ] Выполнить trusted-machine live rerun через direct `scripts/full-run-batch-matrix.sh` на clean committed workspace

### Non-goals
- [x] Не менять public provider list, workspace schema, release verdict JSON shape или canonical matrix files
- [x] Не добавлять wrapper поверх `scripts/full-run-batch-matrix.sh`
- [x] Не синтезировать provider artifacts ACP-side и не ослаблять runtime validators
- [x] Не чинить `codex-code` runtime behavior вне provider/model audit surface

### Approach
1) Обновить shared provider policies/classification и thin adapters для bounded silent missing-artifact recovery.
2) Harden focused draft repair prompt/write-set diagnostics so stdout success never overrides missing files.
3) Update collect semantic instructions/checklists and batch report details without schema changes.
4) Update frontend timeout guard to consume effective matrix/API timeout budget without hiding productive timeout failures.
5) Add provider/model mismatch detection in preflight/report/runtime log diagnostics.
6) Add targeted Go/Python regression tests, sync runbook/testing docs, then run DoD checks.

### Files expected to change
- `internal/runtime/{providercommon,promptcontract,steppolicy,qwencode,claudecode}/*`
- `internal/artifactquality/policy.go`
- `internal/orchestrator/runtime_logging.go`
- `scripts/{write-batch-preflight.py,e2e_batch_report.py,frontend-live-e2e.sh,full-run-ai-advent.sh,full-run-batch.sh,full-run-batch-matrix.sh}`
- `scripts/tests/*`, `docs/RELEASE_LIVE_E2E_RUNBOOK.md`, `docs/TESTING_STRATEGY.md`

### Acceptance criteria
- [x] Targeted Go tests cover silent no-artifact, partial/malformed artifacts, draft repair missing/valid files and provider/model log audit
- [x] Go tests cover zero-output pre-artifact fail-fast, qwen custom-args monitoring, provider lifecycle metadata, run-history terminal guard on cancellation/panic and quality recovery counters
- [x] Python tests cover deeper preflight readiness/model mismatch/artifact smoke, cross-repo-missing details, frontend dependent skip/isolation/env symmetry, timeout cleanup and quality counter aggregation
- [x] `make contracts`, `make test`, `make lint`, `make build` complete or any blocker is documented
- [ ] Trusted-machine rerun uses only direct `scripts/full-run-batch-matrix.sh`; expected verdict has no `runtime_contract_failed`, `runner_unavailable`, `semantic_hard_fail`, or independent frontend failure for the fixed scenarios

### Risks
- Provider preflight can become too expensive if it performs real model calls too often; keep it bounded and selected-provider scoped.
- Overbroad provider/model mismatch matching could flag benign CLI telemetry; only known cross-family model tokens should block.
- Timeout budget increases can hide real UI hangs; keep `active_run_timeout` explicit when backend remains productive but non-terminal.

### Progress log
- 2026-05-06: Plan opened from live matrix triage and bug-fix matrix.
- 2026-05-06: Local implementation complete; targeted Go/Python regressions and DoD checks passed. Live rerun remains pending for a clean trusted-machine workspace.
- 2026-05-07: Implemented live reliability hardening slice: qwen/claude zero-output pre-artifact stalls fail fast as `runner_unavailable`, provider lifecycle diagnostics are persisted/redacted, preflight uses selected-provider artifact smoke, CLI/harness terminalize stale `running` history, frontend cancel uses a sanitized cloned workspace, and quality/batch/matrix reports expose repair/stall/partial counters without blocking successful runs on non-exhausted repair pressure.
- 2026-05-07: Audit pass tightened context cancellation classification, argv/env secret redaction, zero-output fail-fast ordering before draft/validator repair, actual-stall counter filtering, and full pre/post stall counter reporting in quality/provider/matrix outputs. Local DoD passed; trusted rerun remains pending for a clean committed workspace.

### Plan ID
EP-20260502-qwen-codex-regres-fast-live-hardening

### Context
Diagnostic `regres fast` на `qwen-code,codex-code` дошёл до harness/provider execution, но failed на runtime artifact production. Qwen чаще всего зависал в `step3.findings` без `validator-verdict.json`; Codex на OpenStack/Nova collect читал evidence, но проверял/писал relative paths вместо absolute `write_root`, а validator verdict использовал legacy issue-shaped `issues[]`. Дополнительно reports/classifiers переучитывали raw provider text как `runner_unavailable`, а frontend snapshot absence после backend failure выглядел как отдельная frontend regression.

### Goals (must have)
- [x] Усилить artifact-only prompts: exact absolute `write_root`/`draft_final_root`, validator skeleton, canonical `issues[]`, legacy issue bans
- [x] Добавить provider-authored collect pair recovery для non-silent no-artifact collect stalls без ACP-side artifact synthesis
- [x] Добавить provider-authored focused recovery для missing/invalid `validator-verdict.json`
- [x] Добавить focused draft-artifact recovery для `step0/2/4` с write-set guard
- [x] Сохранить strict validation: без ACP-side manifest/verdict/draft synthesis и без silent legacy shape acceptance
- [x] Починить root-marker-only sharding: root-file group + top-level dirs вместо single `"."` shard на больших repos; many-top-level repos enforce `maxAutoShardsPerRepo` через deterministic buckets
- [x] Уточнить report classifiers: raw provider text не создаёт secondary `runner_unavailable` без real availability signal; backend-failed snapshot absence становится dependent skipped frontend status
- [x] Добавить targeted prompt/provider/engine/sharding/report tests
- [x] Прогнать full DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Выполнить trusted-machine diagnostic rerun `regres-fast.bank-openedx` + `regres-fast.openstack` на `qwen-code,codex-code`

### Non-goals
- [x] Не менять public schemas, provider IDs, matrix files, release verdict shape или live harness interfaces
- [x] Не добавлять wrapper над `scripts/full-run-batch-matrix.sh`
- [x] Не ослаблять artifact validation и не делать ACP-side artifact autofill

### Approach
1) Обновить enforced prompt contracts и generated baseline findings prompt pack.
2) Расширить shared provider engine focused recovery adapters: collect получает pair-recovery для non-silent no-artifact stalls и manifest-only repair для authored docs; validator and draft recovery get separate prompts and write-set guards.
3) Обновить thin adapters (`qwen`, `codex`, `claude`) для validator/draft repair command specs and include dirs.
4) Исправить sharding discovery for root-marker-only repos and keep marker-preserving coalescing bounded by cap.
5) Сузить batch/report classifier raw scanning and mark dependent frontend snapshot absence as skipped after backend failure.
6) Синхронизировать docs/tests, затем выполнить DoD и live diagnostic rerun.

### Files expected to change
- `internal/runtime/{providercommon,promptcontract,steppolicy,qwencode,codexcode,claudecode}/*`
- `internal/orchestrator/sharding*`
- `internal/workspace/baseline*`
- `scripts/full-run-batch.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/batch_failure_classification_test.py`
- `README.md`, `docs/ARCHITECTURE.md`, `docs/spec/PIPELINE_SPEC.md`, `docs/TESTING_STRATEGY.md`, `docs/RELEASE_LIVE_E2E_RUNBOOK.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] Prompt tests cover absolute writes, validator skeleton/canonical issues and draft recovery targets
- [x] Engine tests cover collect pair recovery, missing/invalid validator repair, write-set guard, draft repair and strict failure lanes
- [x] Qwen adapter repair specs keep empty stdin and prompt via CLI `-p`
- [x] Sharding tests prove root-marker-only large repo does not collapse to `"."` and many-top-level super-repos stay within `maxAutoShardsPerRepo`
- [x] Report tests prove raw category-word noise and backend-failed snapshot absence do not become independent failures
- [x] DoD passes
- [ ] Diagnostic live rerun reaches `release_verdict_<matrix-id>.json.verdict == "PASS"` or records a narrower residual provider blocker

### Risks
- Focused recovery can still fail if provider is truly silent/unavailable; qwen fully silent no-artifact remains `runner_unavailable`.
- Root-marker-only splitting may increase live runtime duration; cap/coalescing is enforced by deterministic bucket coalescing, which can reduce per-project granularity for very large super-repos.
- Classifier narrowing must not hide real capacity/rate-limit incidents; generic 429/rate-limit/capacity lines still count after noise filtering.

### Refactoring follow-up (non-blocking)
- [ ] После trusted live rerun решить, делать refactor до merge или отдельным PR; если rerun still fails on product behavior, сначала чинить blocker, а не косметику.
- [ ] Снизить повторение focused recovery lifecycle в `internal/runtime/providercommon/engine.go`: выделить общий internal helper для `snapshot -> command spec -> run -> write-set guard -> validate -> classify`, сохранив mode-specific eligibility, diagnostics, write-set guards и error codes без изменения поведения.
- [ ] Дедуплицировать include-dir helpers в `internal/runtime/headless_include_dirs.go`: общий small helper для ordered unique existing dirs, без расширения validator repair read surface beyond `write_root + /staging/final`.
- [ ] Разделить prompt contract code по доменам (`collect`, `validator`, `draft`) или выделить small prompt-builder helpers, но не менять literal prompt intent/tokens без обновления focused prompt tests.
- [ ] Зафиксировать reporting cleanup как отдельный behavior-preserving slice: вынести runner-noise/terminal-success/focused-recovery classifiers из `scripts/e2e_batch_report.py` в локальные helper functions/modules only if tests keep `release_verdict_*`, `profile_matrix_*`, `run_matrix_*` and `quality_report_*` shapes identical.
- [ ] Для каждого refactor step сначала добавить/оставить characterization tests на current behavior, затем гонять `make contracts`, `make test`, `make lint`, `make build`; live harness interfaces and public schemas remain unchanged.

### Progress log
- 2026-05-02: Implemented prompt hardening, validator/draft focused recovery adapters, write-set guards, qwen/codex/claude repair specs, root-marker-only sharding, classifier/frontend dependent-status fixes and targeted tests.
- 2026-05-02: Full DoD passed: `make contracts`, `make test`, `make lint`, `make build`. Generated `internal/api/ui_dist` remained pre-existing unrelated churn and is excluded from this implementation slice.
- 2026-05-02: Follow-up audit tightened report evidence details for `collect_partial_shard_failures`, focused recovery exhaustion/write-set violations and missing headless rows with runtime logs present.
- 2026-05-02: Added sharding-level invariant coverage proving `baseline` and `parallel-default` profiles produce identical shard plan items for a Nova-like root-marker repo.
- 2026-05-02: Live `regres-fast.bank-openedx` exposed two residual blockers before provider artifact validation: flaky controlled-stop unit precheck (`PartialArtifactStallWindow` too small for draft writes) and legacy overlong coalesced shard IDs causing `file name too long` on Open edX root-file groups. Fixed both with bounded hashed shard IDs and a stable partial-artifact test window.
- 2026-05-02: Second live diagnostic reached qwen collect manifest repair and exposed residual repair stall: qwen wrote authored shard docs but did not materialize `shard-pack-manifest.json` before the 90s repair window. Tightened collect repair prompt to put the exact task JSON skeleton at the top, removed the second generic manifest JSON example from repair mode and extended the collect repair window to 3 minutes while keeping strict provider-authored manifest validation.
- 2026-05-02: Third live diagnostic confirmed focused draft repair works in qwen (`step0.constitution` recovered), bounded shard IDs are effective, and collect still fails specifically on provider-authored manifest JSON production: qwen writes `root-overview.md`, then ignores manifest-only repair with zero stdout/stderr until `runtime_contract_failed`. Removed generic `payments` manifest examples from initial collect prompts too and replaced repair-mode canonical bulk with a compact validation checklist so the task-specific skeleton is the only JSON template.
- 2026-05-02: Fourth live diagnostic (`regres-fast-bank-openedx-20260502T145323Z`) confirmed the generic examples are gone, but qwen still stalled in collect repair: root shard wrote `root-overview.md`, then the manifest-only repair produced zero stdout/stderr for 3 minutes and failed `runtime_contract_failed`; a direct qwen probe with a short command-first heredoc prompt wrote `shard-pack-manifest.json` in 21 seconds. Updated collect repair to present one preferred absolute heredoc write command around the task-specific skeleton before any contract text, while preserving provider-authored write-set guard and no ACP-side manifest synthesis.
- 2026-05-02: Fifth live diagnostic (`regres-fast-bank-openedx-20260502T152857Z`) proved command-first repair fixes missing manifests and some invalid manifests (`root`, `docs`, `extras` reached valid manifests), but `iac` still stalled when an invalid manifest already existed. Tightened repair again to overwrite invalid `shard-pack-manifest.json` from the heredoc without reading/diffing/patching the existing JSON and without factual edits before validation.
- 2026-05-02: Sixth live diagnostic (`regres-fast-bank-openedx-20260502T162238Z`) proved collect hardening across all 10 Bank shards, including missing-manifest and invalid-manifest repairs, but exposed the same prompt-shape weakness in `init.step2.asis_docs`: focused draft repair stalled with zero stdout/stderr and no draft artifacts. Tightened draft repair to start with command-first heredoc writes for the step manifest plus referenced `draft_final_root` files, removed full generic draft manifest examples from repair mode, and kept the write-set guard as the source of truth.
- 2026-05-02: Seventh live diagnostic (`regres-fast-bank-openedx-20260502T173735Z`) confirmed the draft repair patch compiled and prechecked, but qwen collect repair still had a nondeterministic stall on `extras`: initial manifest failed on `semantic.edges[0].provenance.kind`, then repair saw the validation error and produced zero stdout/stderr for 3 minutes. Tightened collect repair to be command-only: authored docs/evidence remain encoded in the skeleton, validation-error details are not repeated, and the final instruction is to write the heredoc JSON exactly without factual edits.
- 2026-05-02: Eighth live diagnostic (`regres-fast-bank-openedx-20260502T192507Z`) proved qwen collect/validator/draft focused repair paths across the bank/openedx slice, but exposed a codex gap: no-artifact collect stalls with provider diagnostics had no focused recovery because manifest-only repair only applies after authored docs exist. Added provider-authored collect pair recovery that writes only suggested overview doc + `shard-pack-manifest.json`, keeps fully silent qwen no-artifact in `runner_unavailable`, and added prompt/engine/qwen adapter regression coverage.
- 2026-05-02: Ninth live diagnostic (`regres-fast-bank-openedx-20260502T215450Z`) used an older verification worktree and showed qwen still stalling in focused `validator_verdict_repair` after valid staged final artifacts existed. Tightened validator repair to command-first heredoc `validator-verdict.json`, added qwen/prompt tests for that compact repair contract, made draft artifact monitoring recursive under `draft_final_root`, and split structured report scanning so `kind=runtime_output` provider narration cannot create a secondary `runner_unavailable` without a real availability signal.
- 2026-05-03: Fresh final-snapshot live diagnostic exposed a reporting metadata gap: qwen refresh failed after terminal runtime artifacts/logs existed, but `run-results.tsv` missed the failed headless refresh row because `full-run-ai-advent.sh` wrote rows only after successful CLI status/quality gates. Added idempotent failed-row persistence before every post-run `die` path with known `run_id`, preserving the existing 17-field TSV shape and making reports show failed headless rows instead of missing rows.
- 2026-05-03: Post-DoD review found one more failed-row gap: known-`run_id` cycles could still die before row persistence when `reports/taskruns/<run_id>-quality.json` was missing or invalid. Made failed-row append best-effort around quality parsing and snapshot known runtime artifacts/logs before writing the row, then reran `make contracts`, `make test`, `make lint`, and `make build`.
- 2026-05-03: Live run then exposed remaining legacy classifier behavior: a terminal-success `codex-code` backend (`result=passed`, `quality_gates=passed`, `run-status.env state=completed`) was still marked `runner_unavailable` by `full-run-batch.sh` from recovered raw provider diagnostics. Updated shell and Python classifiers so terminal-success runs keep `failure_class=none`; added regression tests and confirmed the current classifier returns `none` on the live codex artifact.
- 2026-05-03: Live qwen validator repair exposed one more prompt skeleton bug: repair include dirs put validator `write_root` before staged final roots, so generated `checked_paths` could point at `.../validator/final-run-index.json` while still schema-valid. Updated the validator skeleton to skip `write_root` and prefer `/staging/final` roots; added regression coverage.
- 2026-05-03: Final audit found the validator repair include-dir helper still passed the full `ReadContextRoots` set into focused verdict recovery, including workspace and repo roots. Narrowed it to `write_root + /staging/final` only, added regression coverage, and reran full DoD successfully.
- 2026-05-03: Final live report replay found one more stale-classifier path: Python report regeneration ignored raw runner noise for terminal success but still trusted an old `run-classifications.tsv` row with `failure_class=runner_unavailable`/`cancellation_like`. Terminal-success report evaluation now discards stale classified failure/subclass values and has regression coverage.
- 2026-05-03: The same replay exposed legacy `analysis:cross-repo-missing` logic on the successful codex backend: staged artifacts cited all 5 Open edX repos and had a multi-repo owner-gap finding, but the evaluator required explicit `semantic.edges[]`. Updated the report gate to count `citations[].repo` coverage plus multi-repo finding provenance as valid cross-repo signal, preserving the hard fail when both explicit edges and multi-repo finding evidence are absent.
- 2026-05-03: Follow-up audit tightened the Python terminal-success classifier override to match the shell classifier: summary success must be paired with `run-status.env state=completed process_exit=0`. A passed summary without run-status no longer suppresses stale failure classes; regression coverage added.
- 2026-05-04: Post-merge trusted-machine `regres fast` rerun was executed without wrapper on `qwen-code,codex-code`. `regres-fast-bank-openedx-20260503T143529Z` remained `FAIL` with qwen fully silent no-artifact collect retries (`runner_unavailable`), codex single-git_url `runtime_flow_failed` from unresolved authoring text in staged docs, and codex frontend `active_run_timeout`. `regres-fast-openstack-20260503T200727Z` remained `FAIL` with both providers timing out in `init.step1.collect`; root cause was `openstack/openstack` shard explosion (`687` collect shards for that repo, `735` total backend signal).
- 2026-05-04: Follow-up fix enforces `maxAutoShardsPerRepo` for many-top-level repos by deterministic bucket coalescing instead of warning-only overflow, removes persisted authoring instructions from collect early-pair doc/manifest skeletons, and restores the dependent `snapshot_reports_missing` frontend blocker suppression in matrix strict verdicts.
### Plan ID
EP-20260425-runtime-providers-stabilization

### Context
Live `regres fast` triage показал подтверждённый ACP product defect: `step4.proposals` prompt/step policy был слабее `collect/as_is/findings`, поэтому provider смог записать legacy proposal envelope, а strict parser корректно завершил run как `runtime_contract_failed`. Параллельно нужен меньший provider-specific drift между `claude/qwen/codex`, non-empty codex runtime metadata и durable evidence после cleanup temp roots.

### Goals (must have)
- [x] Усилить enforced `proposals-draft-manifest.json` contract без ослабления strict parser
- [x] Добавить parser-level guard для proposals publish surface
- [x] Вынести shared provider validation/raw diagnostic helpers; full process lifecycle alignment superseded by `EP-20260425-provider-runtime-adapter-alignment`
- [x] Исправить `codex-code` runtime meta до `codex-code/headless`
- [x] Добавить durable matrix inventory с bounded raw-output refs
- [x] Синхронизировать docs/spec/runbook и fixtures/tests
- [ ] Перепроверить live `claude-code × bank-openedx`, затем qwen-focused `regres fast`, затем full fresh-all-6 на trusted host

### Non-goals
- [x] Не менять публичные API/schemas/release taxonomy
- [x] Не добавлять compatibility alias layer для legacy proposals manifests
- [x] Не менять canonical matrix/profile/timeout files под текущую машину
- [x] Не добавлять wrapper scripts поверх matrix harness

### Approach
1) Добавить proposals contract helpers/example и подключить их в enforced prompt, step policy, repair hints и baseline prompt bundle.
2) Расширить runtime draft validator для `step4.proposals`: allowed canonical targets `proposals/*` + `reports/changelog/*`, duplicate target rejection.
3) Создать `internal/runtime/providercommon` для artifact validation, stream capture и raw failure message persistence; qwen переиспользует только safe helpers.
4) Зафиксировать codex runtime metadata и расширить raw failure metadata/inventory для post-cleanup triage.
5) Обновить docs/spec/runbook/README и покрыть unit/script tests.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/promptcontract/*`
- `internal/runtime/steppolicy/*`
- `internal/runtimedrafts/*`
- `internal/runtime/{providercommon,claudecode,codexcode,qwencode,runnerdiag}/*`
- `internal/orchestrator/sharding*`
- `internal/workspace/baseline*`
- `scripts/full-run-batch-matrix.sh`
- `scripts/resolve-repos-meta.py`
- `scripts/tests/*`
- `docs/*`

### Acceptance criteria
- [x] Targeted Go runtime/provider tests pass
- [x] Resolver and matrix durable-inventory script tests pass
- [x] Full DoD (`make contracts`, `make test`, `make lint`, `make build`) passes in this worktree
- [ ] Live verification completed on trusted host

### Risks
- Provider-runtime outages (`qwen runner_unavailable`, `codex runtime_timeout`) may still block canonical live confidence even after ACP contract fixes.
- Existing dirty WIP in matrix/preflight/docflow files must not be accidentally reverted or mixed into unrelated semantic changes.

### Progress log
- 2026-04-25: Implemented proposals contract hardening, parser guard, shared providercommon helpers, codex runtime meta, raw diagnostic metadata, durable matrix inventory, fixtures/tests, and docs sync. Full DoD passed; live verification remains pending.
- 2026-04-25: Closed remaining operational blocker surface gap: missing selected provider binary and path pinned-SHA mismatch now materialize terminal `operational_host_preflight_failed` status/inventory/verdict before child batch execution. Full DoD passed again.
- 2026-04-25: Final audit fixed raw failure artifact overwrite risk by switching runnerdiag names from second-resolution stamps to nanosecond-resolution stamp + pid + atomic sequence and exclusive file creation; rapid-failure regression coverage added. Live verification remains pending.
### Plan ID
EP-20260423-regres-fast-failure-taxonomy-hardening

### Context
Сравнительный live triage по `qwen` и `codex` показал четыре новых дефекта: `step4.proposals` всё ещё может запускаться после unusable collect, `qwen` местами смешивает provider unavailable и artifact contract failures, `full-run-ai-advent.sh` может ложно раздувать `completed_runs` из-за malformed `run-results.tsv`, а frontend live E2E не различает explicit failure и productive timeout.

### Goals (must have)
- [x] Не запускать `init|refresh.step4.proposals` после unusable collect
- [x] Сохранить collect/runtime contract как primary root cause вместо downstream proposal parse failures
- [x] Развести `runtime_contract_failed` и `runner_unavailable` в `qwen` artifact-validation paths
- [x] Исправить `run-results.tsv` accounting так, чтобы malformed rows не превращали успешный backend в `infra_incomplete_cycle`
- [x] Поднять raw `runtime_contract_failed` выше heuristic `runner_unavailable` в shell/python batch reporting
- [x] Развести `playwright_failed` и `active_run_timeout` для frontend live E2E
- [x] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Перепроверить live `regres fast` минимум на `qwen bank/openedx` и `codex bank/openedx`

### Non-goals
- [x] Не менять публичные schemas/contracts
- [x] Не ослаблять strict runtime draft manifest validator
- [x] Не трактовать codex plugin/state/cache noise как primary blocker

### Approach
1) Закрыть orchestrator path для proposals после unusable collect и добавить regression coverage.
2) Уточнить qwen artifact-failure classification на missing-artifacts vs malformed-artifacts.
3) Починить structured accounting в `full-run-ai-advent.sh`.
4) Выровнять precedence между raw runtime/session-summary и grep/signature overrides в shell/python reporting.
5) Развести frontend active timeout и explicit Playwright/backend failure, затем синхронизировать docs.

### Files expected to change
- `internal/orchestrator/*`
- `internal/runtime/qwencode/*`
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch.sh`
- `scripts/e2e_batch_report.py`
- `scripts/frontend-live-e2e.sh`
- `scripts/frontend-status-reasons.sh`
- `ui/e2e/live-flow.spec.ts`
- `scripts/tests/*`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Unusable collect не запускает `step4.proposals`
- [x] Malformed runtime draft/shard artifacts остаются `runtime_contract_failed`, даже если рядом есть capacity/429 markers
- [x] Missing artifacts + explicit provider unavailable markers остаются `runner_unavailable`
- [x] `run-results.tsv` считает только валидные строки с фиксированным числом полей
- [x] Batch/reporting держат `runtime_contract_failed` выше `runner_unavailable`, если raw runtime уже дал contract failure
- [x] Frontend live E2E пишет `active_run_timeout` для продуктивного long-running run вместо generic `playwright_failed`

### Risks
- Главный риск: переагрессивно поднять `runtime_contract_failed` и потерять реальные provider incidents. Снижение риска: provider-unavailable остаётся только для explicit markers + missing-artifact shapes, а malformed manifests/unknown fields сохраняются в contract lane.

### Progress log
- 2026-04-23: implementation started for proposals skip gating, qwen artifact classification split, structured ai-advent run accounting, batch/report precedence hardening, and frontend active timeout differentiation.
- 2026-04-23: completed DoD (`make contracts`, `make test`, `make lint`, snapshot `make build`) and hardened shell/python classifiers to ignore `codex` plugin/Cloudflare/state-db noise as primary `runner_unavailable`; live `regres fast` reruns remain pending.
### Plan ID
EP-20260423-regres-fast-qwen-hardening

### Context
Последний live diagnostic выявил четыре blocker класса: semantic duplicates (`user-service` vs `userservice` + owner-gap finding clones), неверный frontend workspace handoff (`headless/arch-workspace` не подхватывался), недоклассификацию provider capacity/rate-limit как `runner_unavailable`, и слабый host preflight до matrix запуска.

### Goals (must have)
- [x] Усилить semantic dedupe для service token variants и finding signature merge
- [x] Ввести единый workspace resolver (`headless/arch-workspace` -> `arch-workspace` -> `workspace`) в shell/python harness
- [x] Добавить capacity/rate-limit сигнатуры в shell/python failure classifiers с сохранением приоритета `runtime_timeout > runner_unavailable > runtime_flow_failed > runtime_contract_failed`
- [x] Не эскалировать `best_effort` partial в `runtime:execution-semantics`, если подтверждён provider unavailable signal
- [x] Усилить host preflight и пометку operational blocker до live matrix run
- [ ] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Выполнить canonical live `regres fast` на `qwen-code` (оба matrix slices) и выпустить execution report

### Non-goals
- [x] Не менять публичные schemas/contracts (`release_verdict_*`, `profile_matrix_*`, `run_matrix_*`, `quality_report_*`)
- [x] Не менять canonical matrix/curated presets под текущую машину

### Approach
1) Исправить dedupe в orchestrator (`semanticEntityDedupKey`, findings-by-signature merge).
2) Унифицировать workspace resolution в `full-run-batch.sh` и `e2e_batch_report.py`.
3) Расширить runtime/provider classification на capacity/rate-limit сигналы в shell/python.
4) Усилить matrix host preflight (`qwen` binary/version + writable roots + path SHA checks для pinned refs).
5) Прогнать DoD, затем mandatory qwen `regres fast` и собрать evidence-based blocker report.

### Files expected to change
- `internal/orchestrator/docflow.go`
- `internal/orchestrator/docflow_test.go`
- `scripts/full-run-batch.sh`
- `scripts/e2e_batch_report.py`
- `scripts/resolve-repos-meta.py`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/batch_failure_classification_test.py`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] `user-service/userservice` и дубли owner-gap findings схлопываются детерминированно
- [ ] Frontend init/cancel не получает false `*_workspace_missing`, если workspace есть в `headless/arch-workspace`
- [ ] Capacity/rate-limit инциденты классифицируются как `runner_unavailable` (при отсутствии explicit timeout)
- [ ] `best_effort` partial не получает runtime semantics blocker только из-за отсутствующего `run_partial_failed`, если есть provider unavailable signal
- [ ] Host preflight ловит невалидный qwen/writable roots/pinned path SHA mismatch как operational blocker до live run

### Risks
- Главный риск: переагрессивный finding dedupe может объединить разные инциденты. Снижение риска: signature ограничен `rule_id + normalized related_ids + normalized title` и merge остаётся deterministic.

### Progress log
- 2026-04-23: implementation started for semantic dedupe hardening, workspace resolver, classifier capacity signals, and host preflight enforcement.
### Plan ID
EP-20260422-headless-legacy-cleanup

### Context
Live `regres fast` на `codex-code` показал, что collect provider может писать legacy `shard-pack-manifest` shape и подхватывать schema drift из runtime workspace (`reports/taskruns`, raw logs, archived artifacts). Параллельно обнаружены downstream gaps: `step2.asis_docs` стартует даже при unusable collect, а batch/matrix harness может оставлять child/profile в stale `running`.

### Goals (must have)
- [x] Зафиксировать shared canonical-only collect contract для `claude-code`, `qwen-code`, `codex-code`
- [x] Убрать workspace-root fallback и legacy schema scavenging из collect runtime surface
- [x] Добавить explicit legacy precheck перед strict collect canonicalization
- [x] Не запускать live `step2.asis_docs` при unusable collect
- [x] Переводить terminal-less child/profile runs в `infra_incomplete_cycle`
- [ ] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить новый canonical live `regres fast` matrix и зафиксировать следующий triage cycle

### Non-goals
- [x] Не расширять schema/contracts под legacy aliases
- [x] Не добавлять permissive semantic compatibility repair
- [x] Не менять release taxonomy или canonical matrix harness command

### Approach
1) Усилить shared prompt/policy/baseline contract для collect и явно запретить legacy aliases.
2) Ограничить collect runtime surface до `write_root` + selected repo roots + explicit `read_context_roots`, включая collect cwd.
3) Добавить raw legacy detector в artifactquality и сохранить strict parse/validate path без semantic repair.
4) Исправить orchestrator skip path для unusable collect и harness reconciliation для stale `running`.
5) Синхронизировать spec/runbook/docs и затем прогнать DoD + новый live matrix cycle.

### Files expected to change
- `internal/runtime/*`
- `internal/artifactquality/*`
- `internal/orchestrator/*`
- `internal/workspace/*`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/*`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Collect prompts/contracts for all headless providers encode canonical-only vocabulary and legacy bans
- [ ] `step1.collect` no longer sees workspace-root by default and uses `write_root` as working directory
- [ ] Legacy collect payloads fail with explicit precheck diagnostics before strict parse
- [ ] `step2.asis_docs` is skipped when collect evidence is unusable
- [ ] Batch/matrix status files never remain silently `running` after child completion
- [ ] DoD passes and a new live `regres fast` cycle is launched for triage

### Risks
- Основной риск — пережать runtime surface и случайно отрезать нужные repo reads. Снижение риска: collect access остаётся на selected repo roots, а `step2/3` продолжают использовать explicit `read_context_roots`.

### Progress log
- 2026-04-22: implementation started for shared prompt cleanup, collect surface hardening, explicit legacy precheck, `step2` unusable-collect skip and batch/matrix stale-running reconciliation.
### Plan ID
EP-20260422-headless-legacy-residuals

### Context
После первого live rerun остались residual failure classes: collect semantic evidence всё ещё может дрейфовать в citation-only shape, findings verdict prompt недоописывает canonical metadata trio, `qwen-code` маскирует quota/auth как `runtime_contract_failed`, а matrix reconciliation не добивает stale `profile-status=running` по owner heartbeat.

### Goals (must have)
- [x] Дожать shared collect contract до обязательного `repo/path` внутри semantic provenance evidence
- [x] Добавить shared canonical contract/example для `validator-verdict.json`
- [x] Переклассифицировать `qwen-code` post-run quota/auth failures в `runner_unavailable`
- [x] Добавить durable child owner sidecar + matrix stale-profile reconciliation
- [ ] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить следующий live `regres fast` cycle и зафиксировать новый triage

### Non-goals
- [x] Не менять `schemas/*` и public contracts
- [x] Не добавлять semantic compatibility repair для legacy payloads
- [x] Не вводить provider-specific codex footer до завершения shared prompt wave

### Approach
1) Обновить shared artifact/prompt policy: collect evidence требует `repo/path`, findings contract задаёт canonical validator verdict example.
2) Сохранить strict legacy reject в `artifactquality` и расширить diagnostics на citation-only semantic evidence.
3) Исправить `qwen` post-run classification без ослабления artifact validation.
4) Добавить `batch-owner.env` heartbeat и matrix stale sweep поверх `profile-status/*.json`.
5) Синхронизировать docs, прогнать DoD и затем повторить live `regres fast`.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/*`
- `internal/workspace/*`
- `scripts/full-run-batch.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/*`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Collect examples/prompts больше не показывают citation-only semantic evidence и явно требуют `repo/path`
- [ ] Findings prompts/policies требуют `version/run_id/generated_at` в `validator-verdict.json`
- [ ] `qwen` quota/auth post-run failures классифицируются как `runner_unavailable`
- [ ] Matrix умеет перевести stale `profile-status=running` в `failed/infra_incomplete_cycle` по owner sidecar
- [ ] DoD проходит и запускается новый live rerun

### Risks
- Главный риск — переусилить stale reconciliation и ложно добивать живой профиль. Снижение риска: reconciliation использует `batch-owner.env pid + updated_at` и оставляет свежий live heartbeat нетронутым.

### Progress log
- 2026-04-22: started residual wave for canonical semantic evidence `repo/path`, validator verdict metadata trio, qwen post-run reclassification, and batch-owner-based stale profile reconciliation.
- 2026-04-22: closed residual literal gaps after implementation audit: added strict `step_contract:null` reject coverage, explicit owner-gap `PASS` prompt/policy coverage, entrypoint hint tests for root `CODEOWNERS`/`MAINTAINERS*`, and docs wording that current runtime isolation relies on temp-root layout + step-local `cwd` because provider-side hard sandbox is unavailable.
- 2026-04-22: fixed an additional qwen runtime bug found during acceptance audit: stall monitors now terminate the stuck provider process, draft/collect post-artifact stalls recover through artifact-only validation/repair, and pre-artifact stalls get one fresh-process retry before final failure classification.
- 2026-04-22: fresh clean bank acceptance rerun surfaced one more qwen residual: the pre-artifact fresh-process retry could still run silently for too long after the initial stall trigger; fixed by keeping a bounded pre-artifact sentinel on the second attempt, adding explicit `retry exhausted` diagnostics, and extending regression coverage around retry stall windows.
- 2026-04-22: a follow-up clean bank rerun showed that bounded qwen retry exhaustion now behaves correctly but still surfaced one semantic misclassification: a fully silent second collect attempt was landing as `runtime_contract_failed`; fixed by reclassifying silent pre-artifact retry exhaustion to `runner_unavailable`, while preserving `runtime_contract_failed` for retries that still emit provider output before stalling.
- 2026-04-22: the next clean bank rerun immediately exposed the remaining edge-case of the same classifier: fully silent retry exhaustion could resurface as `post_artifact` once the fresh process wrote a partial artifact and then froze again; widened the reclassification so any fully silent collect retry exhaustion now lands as `runner_unavailable`, regardless of whether the second attempt died in `pre_artifact` or `post_artifact`.
### Plan ID
EP-20260422-docflow-runtime-residuals

### Context
После второго live `regres fast` collect перестал быть главным blocker, но surfaced новые residual drift classes: `step2.asis_docs` мог писать loose draft manifest и читать sibling baseline artifacts, `step3.findings` ломался на расходящемся `document_id` mapping и semantic alias duplicates, owner-gap оставался release-blocking даже без технических validator issues, а batch classifier путал terminal `validator verdict is FAIL` с `runtime_contract_failed`.

### Goals (must have)
- [x] Зафиксировать strict canonical contract для `step2.asis_docs`
- [x] Убрать sibling baseline leakage через step-local cwd + separated temp roots
- [x] Нормализовать staged docflow document IDs и semantic repo/entity aliases до validator
- [x] Перевести owner-only residual `FAIL` в non-blocking `PASS` без потери findings/questions
- [x] Классифицировать terminal `validator verdict is FAIL` как `runtime_flow_failed`
- [x] Синхронизировать README/spec/runbook/architecture с новым поведением
- [x] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить новый canonical live `regres fast` matrix и зафиксировать следующий triage cycle

### Non-goals
- [x] Не менять `schemas/*` и `internal/contracts/*`
- [x] Не добавлять новые verdict states или provider-specific compatibility repair
- [x] Не ослаблять validator или collect schemas под legacy semantic payloads

### Approach
1) Усилить shared prompt/policy/runtime-manifest validation для `step2.asis_docs`.
2) Перевести non-collect runtime cwd на step-local roots и развести headless/baseline temp roots.
3) Нормализовать `document_id` mapping, `evidence.repo` identity и semantic alias duplicates в docflow assembly/model layer.
4) Добавить owner-gap-only verdict reconciliation и скорректировать batch/report failure classification.
5) Синхронизировать docs, прогнать DoD и затем повторить canonical live `regres fast`.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/*`
- `internal/runtimedrafts/*`
- `internal/orchestrator/*`
- `internal/model/*`
- `internal/workspace/*`
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/PLANS.md`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

### Acceptance criteria
- [ ] `step2.asis_docs` принимает только canonical draft manifest с required outputs и без legacy top-level fields
- [ ] non-collect runtime шаги больше не стартуют из workspace root и не видят sibling baseline workspace как лёгкий template source
- [ ] staged `citation-index.json` и `final-run-index.json` используют согласованный deterministic `document_id` mapping
- [ ] semantic assembly нормализует repo aliases и детерминированно дедуплицирует entity aliases до validator/promotion
- [ ] owner-gap-only residual больше не держит `validator-verdict = FAIL`, но signal остаётся visible в findings/questions
- [ ] terminal `validator verdict is FAIL` классифицируется как `runtime_flow_failed`
- [ ] DoD проходит и следующий live `regres fast` cycle запускается

### Risks
- Главный риск — переусилить semantic dedupe и случайно слить разные сущности. Снижение риска: canonical key включает type + normalized repo identity + normalized name + primary evidence path, а alias remap остаётся детерминированным и тестируется на bank-style duplicates.

### Progress log
- 2026-04-22: started runtime/docflow residual wave for strict `step2` manifest contract, isolated non-collect cwd/layout, deterministic staged document IDs, semantic alias folding, owner-gap downgrade, and validator-fail classifier correction.
### Plan ID
EP-20260421-cleanup-owner-followups

### Context
Safe cleanup уже внедрён: убраны доказуемые code/doc хвосты, сокращены дубли и ослаблены repo-policy проверки под canonical-source model. Остались только owner-gated решения, которые нельзя принимать автоматически без риска снести intentional discoverability/history surfaces.

### Goals (must have)
- [ ] Подтвердить, должны ли `internal/docsync` и `internal/scriptsmeta` оставаться test-only пакетами под `internal/*`
- [ ] Подтвердить, нужно ли одновременно хранить `docs/archive/PLANS_ARCHIVE_2026-04.md` и `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`
- [ ] Не смешать follow-up owner decision с текущим cleanup change set

### Non-goals
- [ ] Не переносить `internal/docsync` и `internal/scriptsmeta` в этом проходе
- [ ] Не удалять `docs/archive/PLANS_SNAPSHOT_2026-04-21.md` без явного подтверждения владельца

### Approach
1) Зафиксировать спорные места как owner-gated follow-up после завершённого cleanup.
2) Не менять code/doc layout дополнительно, пока владелец не подтвердит intent.
3) После подтверждения сделать отдельный малый change set на package placement/archive retention.

### Files expected to change
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Спорные cleanup-решения зафиксированы отдельно от внедрённого safe cleanup
- [ ] Активная секция не держит уже закрытый audit-plan

### Risks
- Главный риск — ошибочно снести intentional discoverability/history surfaces. Снижение риска: держать эти вопросы owner-gated и не менять их без явного решения.

### Progress log
- 2026-04-21: cleanup внедрён; спорные решения по test-only package placement и archive retention вынесены в отдельный follow-up.
### Plan ID
EP-20260420-regres-small-live-triage

### Context
Нужно выполнить canonical live `regres fast` через `scripts/full-run-batch-matrix.sh` на trusted host, непрерывно мониторить статус прогонов и после завершения собрать детальный triage-отчёт по фактическим runtime/harness проблемам без изменения public contract.

### Goals (must have)
- [ ] Запустить canonical `regres fast` matrix slice из clean worktree с нужным provider filter
- [ ] Непрерывно мониторить matrix/batch progress до terminal state
- [ ] Собрать run artifacts, classifications и release/profile reports
- [ ] Зафиксировать продуктовые/runtime/harness баги и отделить их от operational noise
- [ ] Составить итоговый triage-отчёт: что сломалось, где именно, что нужно чинить дальше

### Non-goals
- [ ] Не менять matrix taxonomy, timeout profiles и public schemas во время самого прогона
- [ ] Не подменять canonical harness wrapper-скриптом или ad-hoc env overrides
- [ ] Не пытаться чинить live run до завершения triage цикла

### Approach
1) Проверить trusted-host prerequisites, pinned path checkouts и clean worktree.
2) Выполнить `regres fast` через `scripts/full-run-batch-matrix.sh` c canonical sweep set.
3) Во время выполнения регулярно читать batch/matrix logs, status files и intermediate reports.
4) После завершения разобрать terminal artifacts: `profile-runs.jsonl`, matrix status files, batch classifications, session summaries, release/profile verdicts и raw taskrun diagnostics.
5) Сформировать детальный triage-отчёт и список необходимых follow-up fixes.

### Files expected to change
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Matrix slice дошёл до terminal state (`passed|failed`) с сохранёнными batch/matrix breadcrumbs
- [ ] Есть собранный список фактических failure classes и их root cause
- [ ] Итоговый отчёт отделяет runtime/provider проблемы от harness/reporting проблем

### Risks
- Основной риск — long-running live provider hangs или внешние transport/API failures. Снижение риска: canonical watchdog/reporting уже встроены; мониторинг идёт по нескольким источникам (`driver log`, batch status, profile status, taskrun artifacts), а не по одному summary-файлу.

### Progress log
- 2026-04-20: prerequisites и pinned path SHA проверены, подготовлен trusted-host flow для live triage.

---

## Implemented vs Planned (operational mirror)

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Таблица ниже — инженерный mirror и должна оставаться синхронизированной с канонической матрицей.

| Epic | Статус | Комментарий |
|---|---|---|
| 1 Workspace/contracts | done (beta baseline) | Schema-driven + semantic validation, resolver `path/git_url`, diagnostics API |
| 2 Runtime artifact contracts | done (beta baseline) | Validation + artifact-only runtime execution contract, contract tests |
| 3 Runtime/orchestration seam | done (beta baseline) | Fake default + opt-in headless runtime selector with provider choice (`claude-code` default, `qwen-code`, `codex-code` release peer), persisted runtime execution metadata |
| 4 Model deterministic core | done (beta baseline) | Canonical IDs/collision rules + deterministic regression tests |
| 5 Pipeline 0–4 | done (beta baseline) | `init|refresh` runnable через CLI/API |
| 6 UI baseline | done (beta baseline) | Setup/validate/run/inspect + editors + git helpers |
| 7 Domain-first layer | done (beta baseline) | Per-domain contracts + deterministic Step 1 enrichment canonical domain/team cards without auto-create |
| 8 Baseline bundle | done (beta baseline) | `skills/subagents.yaml` + prompt packs + validation |
| 9 Q&A capability | done (beta boundary) | Workspace-backed QA service + read-only CLI `acp qa` + public read-only `POST /api/qa/ask` |
| 10 Changelog compilers | done (beta baseline) | Iteration changelog materialization в `reports/changelog/*` |
| 11 `POST /api/qa/ask` | done (beta baseline) | Read-only API wrapper over deterministic Q&A service |
| 12–13 | out of MVP | Вне текущего beta scope |
| 14 CI trigger mode | done (beta baseline) | CLI batch required, smoke/golden jobs без live network deps |
| 15 Domain/baseline pack hardening | done (beta baseline) | Baseline skills/prompts wired и versioned в workspace |
