---
name: acp-e2e-live-gate
description: Провести trusted-machine live E2E gate ACP или явно выбранную non-release диагностику через canonical direct harness, проверив execution verdict и отдельные SWE UX/artifact assessments.
---

## Источники и выбор режима

Прочитать [RELEASE_LIVE_E2E_RUNBOOK](../../../docs/RELEASE_LIVE_E2E_RUNBOOK.md): operator protocol,
prerequisites и раздел выбранного режима. Profile taxonomy, constituents, repo sets, provider/run
counts и native timeout profiles брать из [catalog](../../../examples/e2e-profile-catalog.yaml)
и указанных в нём matrices. Incident triage и host bootstrap находятся в runbook, не в этом skill.

Canonical profiles: `regres fast|long`, `release fast|long|full`. Дополнительные smoke/diagnostic
selectors не доказывают release readiness. Не подменять запрошенный profile ad-hoc matrix.
Planner `./scripts/run-python.sh scripts/live-e2e-plan.py ... --format shell` только печатает прямые
команды, исполнение — `scripts/full-run-batch-matrix.sh` без wrapper.

## Black-box operator protocol

Это local `manual-live-e2e workflow`: operator procedure на trusted host, not GitHub Actions workflow.
Scripts produce machine execution verdicts only; SWE-agent отдельно оценивает UX и качество artifacts.
Работать через public/operator surfaces: direct harness, UI/API, generated reports, taskrun artifacts
и raw metadata/logs, matrix/profile status и inventories, driver heartbeat/logs, verifier output.

После каждой фазы фиксировать короткий шаг по
[operator assessment template](../../../docs/templates/LIVE_E2E_OPERATOR_ASSESSMENT.md):

```text
goal: что проверяем
action: вызванная публичная поверхность
observed evidence: команды, UI/API и artifact/report paths
status: passed|failed|skipped|blocked
primary classification: evidence-backed failure class или none
next decision: continue|stop|rerun diagnostic|verify verdict|final report
```

Harness больше не пишет `blackbox_e2e_steps_*`; machine-authored pseudo-reasoning не является evidence.
Optional operator notes не заменяют accepted SWE UX and artifact-quality reports.

## Preflight и границы исполнения

- Перед DoD/matrix проверить trusted host, clean committed tree либо clean worktree, writable roots,
  provider binaries/readiness и canonical path checkouts с pinned SHA. Для нового worktree сначала
  установить local dependencies по [CONTRIBUTING](../../../CONTRIBUTING.md); использовать exact
  Go/Node/Python resolvers, включая `./scripts/run-npm.sh ci --prefix ui`.
- Если host/provider/path prerequisites не выполняются, остановить run и классифицировать
  operational blocker. Если хост подходит, подготовить checkouts по runbook bootstrap. Не править
  canonical matrices/curated `repos_file`, shims, aliases или overrides для обхода blocker.
- Не добавлять wrapper или GitHub-hosted live workflow. `BATCH_SKIP_PRECHECK=1` допустим только в
  test-only harness mode с явным `ACP_TEST_ALLOW_BATCH_SKIP_PRECHECK=1`.
- Release matrices требуют explicit `baseline` + `parallel-default`, один `single-*` и один
  `multi-*` profile, `RUN_COUNT=1`. Не задавать manual timeout overrides в official release slices;
  использовать native `timeout_profile`. Non-release execution overrides — `ACP_EXECUTION_*`,
  timeout diagnostics — только по runbook.
- Читать totals в контексте `selected_providers`/`selected_run_indexes`. Дополнительные manual
  provider diagnostics не включать в canonical regression totals или release evidence.

## Исполнение и принятие результата

1. Выбрать catalog slice, записать direct command и запустить harness. Отслеживать durable status,
   inventories, owner heartbeat и driver logs; public provider stdout/stderr — diagnostics,
   runtime success подтверждается валидными artifacts.
2. После профиля оценить execution reports, run matrix и taskrun quality JSON. Разделять
   `runtime_contract_status` и `artifact_quality_status`; `artifact_quality.*` в `quality_signals[]`
   и `artifact_quality:` в warnings остаются strict artifact-quality blockers.
3. Проверить frontend `init-inspect` UI/API evidence всех release providers. Dependent frontend skips
   после backend failure не считать отдельной UI regression. Cancellation покрывается deterministic
   fake tests; Ask evidence относится к SWE UX assessment. Optional `UI_E2E_QA_SMOKE=1` для
   non-release/fake UX не является machine execution verdict input.
4. Подготовить отдельные `reports/swe_ux_assessment_<matrix-id>.md` и
   `reports/swe_artifact_quality_assessment_<matrix-id>.md` с matching matrix id и evidence-backed
   `decision: accepted` либо residual blockers. Проверить execution JSON и companion reports через
   `./scripts/run-python.sh scripts/verify-release-verdict.py reports/release_verdict_<matrix-id>.json`.
5. Release readiness требует machine `PASS`, `strict_status=passed`, snapshot-only artifacts,
   passed frontend для трёх release providers, passed shard-plan invariant между sweeps и оба
   accepted SWE reports. Полный список acceptance/failure classes — в runbook. Для `release full`
   это требуется для каждого constituent matrix, а не только последнего запуска.

При missing/failed execution или assessment evidence результат остаётся blocked/failed с причиной
и следующим действием. Gate verification не означает разрешения на tag, push или публикацию release.
