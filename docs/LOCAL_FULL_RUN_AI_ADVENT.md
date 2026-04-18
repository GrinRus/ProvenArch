# Local Full-Run: scenario profile (`ai_advent`-like)

Этот runbook описывает воспроизводимый полный прогон ProvenArch «как пользователь» в временном `tmp` workspace (`/tmp`) против целевого репозитория.

Сценарий используется и для первого локального запуска, и для итеративного цикла улучшений backend/frontend.
Для pre-release решения по принципу strict gate (`PASS|FAIL`) используйте отдельный агентский runbook:
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

## 1) Что проверяет сценарий

- Bootstrap workspace в `tmp` через `acp init-workspace`.
- Автоматический `git init` для workspace root при отсутствии `.git`.
- API simulation: `validate -> init -> artifacts`.
- API simulation run logs: `GET /api/pipeline/runs/<run_id>/logs`.
- Полный runtime цикл на каждой итерации:
  - `init/refresh` в `fake`
  - `init/refresh` в `headless`
- Quality guardrails для headless:
  - запрет mock/fake runtime version,
  - fail при zero-signal quality summary,
  - fail при регрессии сигнала относительно предыдущей итерации для той же пары `(runtime_mode, pipeline)`,
  - fail при `No findings reported` в headless refresh при owner-related gaps в coverage,
  - fail при canonical duplicates в `coverage.missing`,
  - fail при duplicate open-question texts после нормализации,
  - fail при critical off-topic markers в headless refresh artifacts (local semantic check),
  - ai-advent profile checks для минимально содержательного сигнала.
- Snapshot artifacts per run: `TMP_ROOT/snapshots/<run_id>/...`.
- Проверка ключевых артефактов (`as-is/findings/coverage`) и run quality summaries.
- (Опционально) quality gates: `make contracts`, `make test`, `make lint`, `make build`.
- (Опционально) live frontend e2e через Playwright: `validate -> run init -> inspect artifacts`.

Batch-only semantic hard-fail checks (в `scripts/e2e_batch_report.py`):
- `analysis:off-topic`
- `analysis:evidence-scope`
- `analysis:cross-doc`
- `analysis:cross-repo-missing` (для multi-profile, `expected_repo_count >= 2`)

## 2) Переменные скрипта

`./scripts/full-run-ai-advent.sh` поддерживает:

- `PROVENARCH_ROOT` (default: текущий repo ProvenArch)
- `TARGET_REPOS_FILE` (canonical: YAML с `repos[]`, как в `workspace.yaml`)
  - если файл содержит `runtime.profile.timeouts`, `init-workspace` переносит профиль в `workspace.yaml`
- legacy single inputs запрещены: script делает fail-fast с migration hint на `TARGET_REPOS_FILE`
- `TMP_ROOT` (default: auto `mktemp -d -t provenarch-ai-advent.XXXXXX`)
- `ACP_RUNTIME_PROVIDER` (headless provider: `claude-code` default или `qwen-code`)
- `ACP_CLAUDE_CMD` (команда для provider `claude-code`; default `claude-code`, поддержан direct `claude` без wrapper)
- `ACP_QWEN_CMD` (команда для provider `qwen-code`; default `qwen`)
- `KEEP_TMP` (`0/1`, default `0`)
- `ITERATIONS` (default `1`)
- `RUN_QUALITY_GATES` (`0/1`, default `1`)
- `RUN_LOGS_TTL_HOURS` (default `168`)
- `RUN_LOGS_MAX_RUNS` (default `200`)
- timeout env overrides (canonical):
  - `ACP_RUNTIME_STEP_TIMEOUT_SEC` (default `1800`)
  - `ACP_RUNTIME_HEARTBEAT_SEC` (default `30`)
  - `ACP_PIPELINE_TIMEOUT_SEC` (default `2400`)
  - `ACP_PIPELINE_KILL_GRACE_SEC` (default `30`)
  - `ACP_API_READY_TIMEOUT_SEC` (default `60`)
  - `ACP_API_INIT_TIMEOUT_SEC` (default `120`)
  - `ACP_UI_INIT_POLL_TIMEOUT_SEC` (default `900`)
  - `ACP_UI_CANCEL_POLL_TIMEOUT_SEC` (default `420`)
- deprecated timeout aliases запрещены: script делает fail-fast с migration hint

Effective timeout precedence для full-run/batch/frontend live:
- `env > workspace.yaml(runtime.profile.timeouts) > defaults`

Batch/Frontend scripts:
- `scripts/full-run-batch-5x2.sh`
  - `BATCH_ID` (default `batch-<UTC timestamp>`)
  - `TARGET_REPOS_FILE` (canonical; единственный вход)
  - optional profile metadata:
    - `PROFILE_ID`
    - `PROFILE_SOURCE_KIND` (`path|git_url`, optional: если не задан, auto-detect из `TARGET_REPOS_FILE`)
    - `EXPECTED_REPO_COUNT`
    - `SWEEP_ID`
  - execution profile env (обычно выставляются matrix sweep'ом):
    - `ACP_EXECUTION_STRATEGY`
    - `ACP_MAX_PARALLEL_TASKS`
    - `ACP_FAILURE_POLICY`
    - `ACP_SHARD_DISCOVERY_MODE`
  - `E2E_TMP_ROOT` (default `/tmp/provenarch-test_arch_project`)
  - `BATCH_ROOT` (default `${E2E_TMP_ROOT}/runs/${BATCH_ID}`)
  - `REPORTS_ROOT` (default `${E2E_TMP_ROOT}/reports`)
  - `ACP_CLAUDE_CMD_BIN` (default `claude`, direct binary)
  - `ACP_QWEN_CMD_BIN` (default `qwen`, direct binary)
  - shard controls:
    - `BATCH_PROVIDER_FILTER` (`all` или CSV `qwen-code,claude-code`)
    - `BATCH_RUN_SELECTION` (`all`, CSV `1,3,5` или диапазоны `1-3,5`)
    - `BATCH_SKIP_PRECHECK` (`0|1`; default `0`)
    - `BATCH_FRONTEND_MODE` (`auto|always|never|per_run`; default `auto`; `auto` skip если `run1` не выбран, `always` использует первый выбранный backend run)
    - `BATCH_FRONTEND_CANCEL_MODE` (`once_per_provider|per_run|never`; default `once_per_provider`)
    - `UI_E2E_HEADED` (`0|1`; default `0`)
- `scripts/full-run-batch-matrix.sh`
  - `E2E_MATRIX_FILE` (required; YAML `profiles[]`, optional `sweeps[]`)
  - approved profile ids: `single-path`, `single-git_url`, `multi-path`, `multi-git_url`
  - если `sweeps[]` отсутствует -> implicit `baseline` sweep (только non-release/diagnostic)
  - canonical acceptance запускать из clean committed tree или отдельного clean worktree без unrelated локальных правок
  - canonical high-level profile catalog: `examples/e2e-profile-catalog.yaml`
  - canonical non-release slices: `examples/e2e-matrix.regres-*.yaml`
  - canonical release slices: `examples/e2e-matrix.release-*.yaml`
  - legacy compatibility slices: `examples/e2e-matrix.regression-wave1.yaml`, `examples/e2e-matrix.release-wave1.yaml`, `examples/e2e-matrix.release-wave2.yaml`
  - release-ready sweeps: `baseline`, `parallel-default`
  - `RUN_COUNT` (default `1` для matrix driver; release-mode фиксирует `RUN_COUNT=1`)
  - release-mode (`MATRIX_ID=release-*` или `E2E_MATRIX_RELEASE_MODE=1`) требует explicit `sweeps[]` с ровно `baseline` + `parallel-default` и ровно два профиля: один `single-*`, один `multi-*`; иначе matrix driver завершится fail-fast до batch execution
  - `repos_file` в matrix-профилях: относительные пути резолвятся от директории `E2E_MATRIX_FILE`
  - `MATRIX_ID` (default `matrix-<UTC timestamp>`)
  - `MATRIX_ROOT` (default `${E2E_TMP_ROOT}/matrix/${MATRIX_ID}`)
  - профильный запуск делегируется в `full-run-batch-5x2.sh` (`profiles × sweeps`)
- `scripts/frontend-live-e2e.sh`
  - `WORKSPACE` (required)
  - `RUNTIME_PROVIDER` (required: `claude-code|qwen-code`)
  - `UI_E2E_EXPECTED_REPO_COUNT` (optional; default `1`)
  - `OUTPUT_DIR` (optional; default `mktemp`)
  - `LISTEN` (optional; default free local port)
  - Playwright output path для wrapper фиксируется как `$OUTPUT_DIR/playwright-results`
  - `UI_E2E_HEADED=1` добавляет `--headed` к Playwright запуску

Direct Playwright запуск (без wrapper) использует:
- `UI_E2E_OUTPUT_DIR` (optional; default `/tmp/provenarch-ui-e2e/test-results`)

## 3) Быстрый запуск (script)

```bash
cd /path/to/ProvenArch

# Вариант 1: default headless provider (claude-code в PATH)
TARGET_REPOS_FILE=/abs/path/to/repos.yaml ./scripts/full-run-ai-advent.sh

# Вариант 2: direct claude без wrapper
TARGET_REPOS_FILE=/abs/path/to/repos.yaml ACP_RUNTIME_PROVIDER=claude-code ACP_CLAUDE_CMD=claude ./scripts/full-run-ai-advent.sh

# Вариант 3: явно задать provider=qwen-code
TARGET_REPOS_FILE=/abs/path/to/repos.yaml ACP_RUNTIME_PROVIDER=qwen-code ./scripts/full-run-ai-advent.sh

# Вариант 4: явно задать команду для выбранного provider
TARGET_REPOS_FILE=/abs/path/to/repos.yaml ACP_RUNTIME_PROVIDER=qwen-code ACP_QWEN_CMD=/abs/path/to/qwen ./scripts/full-run-ai-advent.sh

# Вариант 5: оставить tmp workspace для ручного анализа
TARGET_REPOS_FILE=/abs/path/to/repos.yaml KEEP_TMP=1 ./scripts/full-run-ai-advent.sh

# Вариант 6: batch 5x2 + frontend live e2e + агрегированный quality report
TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
./scripts/full-run-batch-5x2.sh

# Вариант 7: canonical `regres fast` (3 backend runs total)
# matrix file already carries canonical timeout_profile=short-window
E2E_MATRIX_FILE=./examples/e2e-matrix.regres-fast.bank-openedx.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
BATCH_PROVIDER_FILTER=qwen-code \
./scripts/full-run-batch-matrix.sh

E2E_MATRIX_FILE=./examples/e2e-matrix.regres-fast.openstack.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
BATCH_PROVIDER_FILTER=qwen-code \
./scripts/full-run-batch-matrix.sh

# Вариант 7.1: canonical `regres long` (2 backend runs total)
# matrix file already carries canonical timeout_profile=medium-window
E2E_MATRIX_FILE=./examples/e2e-matrix.regres-long.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
BATCH_PROVIDER_FILTER=qwen-code \
./scripts/full-run-batch-matrix.sh

# Вариант 7.2: дополнительная отладка того же regression slice на claude
E2E_MATRIX_FILE=./examples/e2e-matrix.regres-long.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
BATCH_PROVIDER_FILTER=claude-code \
BATCH_SKIP_PRECHECK=1 \
./scripts/full-run-batch-matrix.sh

# Вариант 8: произвольный matrix run (approved profiles × sweeps)
E2E_MATRIX_FILE=/abs/path/to/e2e-matrix.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
./scripts/full-run-batch-matrix.sh

# Вариант 8.1: canonical `release fast`
# matrix file already carries canonical timeout_profile=short-window
MATRIX_ID=release-fast-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=./examples/e2e-matrix.release-fast.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh

# Вариант 8.2: canonical `release long`
# matrix file already carries canonical timeout_profile=medium-window
MATRIX_ID=release-long-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=./examples/e2e-matrix.release-long.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh

# Вариант 8.3: canonical `release full` addon slice (`ftgo + sentry`)
# matrix file already carries canonical timeout_profile=extended-window
MATRIX_ID=release-full-ftgo-sentry-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=./examples/e2e-matrix.release-full.ftgo-sentry.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh

# Вариант 9: параллельные shard-runs (по провайдерам)
TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
BATCH_ID=batch-qwen \
BATCH_PROVIDER_FILTER=qwen-code \
./scripts/full-run-batch-5x2.sh &

TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
BATCH_ID=batch-claude \
BATCH_PROVIDER_FILTER=claude-code \
./scripts/full-run-batch-5x2.sh &

wait
```

Canonical regression/release profile taxonomy задаётся в `examples/e2e-profile-catalog.yaml`:
- `regres fast` = `3` backend runs total
- `regres long` = `2` backend runs total
- `release fast` = `8` backend runs total
- `release long` = `8` backend runs total
- `release full` = `24` backend runs total

Canonical live matrices также несут checked-in `timeout_profile`, который matrix driver разворачивает без внешних `ACP_*TIMEOUT*` override:
- `short-window` = step `3600s`, pipeline `7200s`, ui-init `1200s`
- `medium-window` = step `5400s`, pipeline `14400s`, ui-init `1500s`
- `extended-window` = step `10800s`, pipeline `21600s`, ui-init `1800s`

Legacy `regression-wave1` / `release-wave1` / `release-wave2` остаются только compatibility slices для ad-hoc diagnostics и не считаются canonical profile taxonomy.

Правила shard-run:
- параллельные shard-процессы обязаны использовать разные `BATCH_ID`;
- precheck рекомендуется выполнять только в одном shard (`BATCH_SKIP_PRECHECK=0`), для остальных shard'ов использовать `BATCH_SKIP_PRECHECK=1`.
- для canonical regression/release acceptance `BATCH_SKIP_PRECHECK=1` не использовать; это diagnostic-only bypass.
- в shard-режиме требуются runtime-бинари только выбранных провайдеров (`BATCH_PROVIDER_FILTER`).

`full-run-batch-matrix.sh` — официальный локальный (trusted machine) runbook и не входит в required CI gates.
При запуске из отдельного clean worktree сначала подготовьте локальные UI deps в этом worktree (`npm ci --prefix ui`), иначе precheck на `make test` остановит batch до runtime phase.
Если цель запуска — release verdict, используйте критерии и формат решения из:
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

Script всегда формирует:
- `TMP_ROOT/full-run.log`
- `TMP_ROOT/session-summary.md`
- `TMP_ROOT/snapshots/<run_id>/...`

`session-summary.md` верифицирует полноту цикла и содержит:
- `expected_runs` / `completed_runs`
- `expected_headless_runs` / `completed_headless_runs`
- `running_runs_detected`
- `termination_signal`

При ошибке script всегда сохраняет `TMP_ROOT` для дебага независимо от `KEEP_TMP`.

Для batch-скрипта отчёты сохраняются в:
- `/tmp/provenarch-test_arch_project/reports/run_matrix_<batch-id>.md`
- `/tmp/provenarch-test_arch_project/reports/run_matrix_<batch-id>.tsv`
- `/tmp/provenarch-test_arch_project/reports/frontend_e2e_matrix_<batch-id>.md`
- `/tmp/provenarch-test_arch_project/reports/frontend_cancel_e2e_matrix_<batch-id>.md`
- `/tmp/provenarch-test_arch_project/reports/quality_report_<batch-id>.md`
- `/tmp/provenarch-test_arch_project/reports/profile_matrix_<matrix-id>.md`
- `/tmp/provenarch-test_arch_project/reports/profile_matrix_<matrix-id>.tsv`
- `/tmp/provenarch-test_arch_project/reports/release_verdict_<matrix-id>.md`
- `/tmp/provenarch-test_arch_project/reports/release_verdict_<matrix-id>.json`

Batch evaluator source-of-truth:
- backend quality берётся из snapshot-артефактов `snapshots/<run_id>/reports/*`;
- если snapshot недоступен, evaluator фиксирует issue `reliability:snapshot-missing` (без fallback к `arch-workspace/reports`);
- frontend live e2e запускается на отдельной копии workspace (`frontend-workspace`) и не влияет на backend quality content score.
- frontend/cancel matrix формируются в run-level формате (`provider + run_index`), strict matrix gate блокирует любой init run-status != `passed`.
- `quality_report_<batch-id>.md` и `profile_matrix_<matrix-id>.md` считают только реально выбранные `selected_providers` и `selected_run_indexes`, а не synthetic `claude+qwen x run1..run5` поверхность.
- для multi-profile (`EXPECTED_REPO_COUNT >= 2`) batch hard-fail включает `analysis:cross-repo-missing`.
- backend run-matrix дополнительно классифицирует failure classes: `runtime_parse`, `runner_unavailable`, `runtime_timeout`, `infra_signal_terminated`, `infra_incomplete_cycle`, `quality_gates_failed`, `summary_missing`, `precheck_failed`.
- runtime flow checks в evaluator: `runtime:shard-artifacts`, `runtime:shard-metadata`, `runtime:execution-semantics`, `runtime_flow_failed`.
- collect runtime делает максимум одну post-success artifact-repair попытку для skeletal/generic-only `shard-pack-manifest.json`; если repair не улучшил artifact fidelity, исходный `write_root` восстанавливается.
- schema-invalid `TaskResult` получает отдельный direct-JSON retry с whitelist допустимых `changeset[].op`; invalid manifest после schema-valid result идёт в отдельный artifact-repair retry.
- если после этой repair попытки `shard-pack-manifest.json` остаётся missing/invalid/skeletal, collect step больше не считается успешным и должен выйти как `runner_parse_failed` / `runtime_parse`.
- collect contract требует полного `compatibility` block и global uniqueness для `citation-index.claim_ids`; duplicate claim ids разрешается чинить только как validator-scope index/reference repair, без semantic rewrite authored docs.
- `artifact_quality:*` в `reports/taskruns/<run_id>-quality.json.run_warnings` эскалируется batch evaluator'ом в `quality_gates_failed`; canonical live gate не принимает refresh artifacts, схлопнувшиеся до одного generic `cite.runtime-summary` без rich repo-specific shard evidence.

При `runner_parse_failed` raw stdout/stderr сохраняются в:
- `WORKSPACE/reports/taskruns/raw/*-stdout.log`
- `WORKSPACE/reports/taskruns/raw/*-stderr.log`
- `WORKSPACE/reports/taskruns/raw/*-meta.json` (bytes/hash + task context)

## 4) CLI/API поток вручную (без скрипта)

```bash
cd /path/to/ProvenArch
make build

TMP_ROOT="$(mktemp -d -t provenarch-ai-advent.XXXXXX)"
WORKSPACE="$TMP_ROOT/arch-workspace"

./bin/acp init-workspace \
  --workspace "$WORKSPACE" \
  --repos-file /abs/path/to/repos.yaml

# API simulation
PORT=18080
./bin/acp serve --workspace "$WORKSPACE" --runtime fake --listen "127.0.0.1:$PORT"
# отдельным терминалом:
# curl -X POST http://127.0.0.1:$PORT/api/workspace/validate
# curl -X POST -H 'Content-Type: application/json' -d '{"trigger":"manual"}' http://127.0.0.1:$PORT/api/pipeline/init
# curl http://127.0.0.1:$PORT/api/pipeline/runs/<run_id>
# curl http://127.0.0.1:$PORT/api/pipeline/runs/<run_id>/artifacts

# Runtime cycle: fake + headless
./bin/acp run --workspace "$WORKSPACE" --pipeline init --runtime fake --non-interactive
./bin/acp run --workspace "$WORKSPACE" --pipeline refresh --runtime fake --non-interactive
./bin/acp run --workspace "$WORKSPACE" --pipeline init --runtime headless --runtime-provider claude-code --non-interactive
./bin/acp run --workspace "$WORKSPACE" --pipeline refresh --runtime headless --runtime-provider claude-code --non-interactive
```

Ключевые артефакты после прогона:

- `reports/as-is/overview.md`
- `reports/findings/findings.md`
- `reports/coverage/open-questions.md`

## 5) Manual UI checklist

После запуска `acp serve --workspace ...` откройте UI и проверьте минимум по секциям.

### Setup

- `Setup: Workspace`: виден guided setup для `repos[]`.
- Можно добавить/удалить repo row (`name + path|git_url + ref`).
- `Validate workspace` возвращает `resolved_repos` и diagnostics.
- Diagnostics сгруппированы по repo.

### Baseline

- `Baseline: Editors`: можно открыть/сохранить baseline artifacts.
- `Setup: Step 0 Wizard Contract`: сохраняется `charter/wizard/step0-contract.json`.

### Runs

- `Runs: Pipeline Control`: запускаются `Run init` и `Run refresh`.
- `Runs: History`: отображаются queued/running/succeeded/failed и выбор run.
- `Runs: Logs`: для выбранного run отображается live stream с полями `timestamp/level/step/domain/message`.
- В `Runs: Logs` работают quick actions:
  - `Copy logs`
  - `Download logs`
  - `Open taskrun artifact` (если путь найден в log events)

### Results

- `Results: Coverage & Questions`: показываются coverage/open questions.
- `Results: Run Artifacts`: можно открыть артефакты run.

### Optional automation (Playwright live smoke)

```bash
cd /path/to/ProvenArch
make build
npm ci --prefix ui
npm exec --prefix ui playwright install chromium

WORKSPACE=/path/to/arch-workspace \
RUNTIME_PROVIDER=qwen-code \
ACP_QWEN_CMD=qwen \
./scripts/frontend-live-e2e.sh
```

Для `claude-code`:

```bash
WORKSPACE=/path/to/arch-workspace \
RUNTIME_PROVIDER=claude-code \
ACP_CLAUDE_CMD=claude \
./scripts/frontend-live-e2e.sh
```

Output semantics:
- direct `npm run --prefix ui e2e:live`: default `/tmp/provenarch-ui-e2e/test-results`, override `UI_E2E_OUTPUT_DIR`;
- `scripts/frontend-live-e2e.sh`: output в `$OUTPUT_DIR/playwright-results`;
- `scripts/frontend-live-e2e.sh` берёт poll timeouts из effective runtime config (`GET /api/runtime/timeouts`) с env override (`ACP_UI_INIT_POLL_TIMEOUT_SEC`/`ACP_UI_CANCEL_POLL_TIMEOUT_SEC`);
- `UI_E2E_EXPECTED_REPO_COUNT` задаёт ожидаемое число resolved repos в live e2e (default `1`).
- `UI_E2E_SCENARIO`:
  - `init-inspect` (default): validate -> run init -> inspect artifacts;
  - `cancel-refresh`: validate -> run refresh -> cancel selected run -> verify `failed + run_canceled`.
- для `UI_E2E_SCENARIO=cancel-refresh` script использует controlled slow stub runner;
  длительность задаётся `UI_E2E_CANCEL_STUB_SLEEP_SEC` (default `90`).
- при `BATCH_FRONTEND_MODE=auto` frontend smoke помечается `skipped`, если `run1` не входит в `BATCH_RUN_SELECTION`.

## 6) Continuous Improvement Loop (balanced backend/frontend)

Используйте script как базовый повторяемый цикл.

1. Запустить полный прогон (`full-run-ai-advent.sh`) в новом `tmp` workspace.
2. Снять findings и разложить по корзинам:
   - backend
   - frontend
3. Приоритизировать задачи: P1/P2 first, без перекоса в один слой.
4. Внести изменения в код.
5. Повторить полный прогон script'ом с нуля.
6. Повторять до условия остановки.

Критерий остановки:

- зелёные `make contracts`, `make test`, `make lint`, `make build`
- отсутствие P1/P2 находок по последнему полному прогону;
- headless runs проходят strict quality checks (non-mock, non-zero-signal, no degradation).
- headless refresh проходит semantic checks (owner-gap+findings, coverage/question dedupe).

## 7) Диагностика типовых проблем

- Ошибка `headless runtime command ... is unavailable`:
  - проверить `ACP_RUNTIME_PROVIDER`;
  - для `claude-code`: установить `claude-code` или использовать direct `ACP_CLAUDE_CMD=claude` (либо задать `ACP_CLAUDE_CMD=/abs/path/to/runner`);
  - для `qwen-code`: установить `qwen` или задать `ACP_QWEN_CMD=/abs/path/to/runner`.
- Ошибки bootstrap (`workspace.yaml/.git/skills/subagents.yaml` не созданы):
  - проверить вывод `logs/init-workspace.log`.
- Timeout/зависание pipeline:
  - проверить `session-summary.md` (`failure_reason=runtime_timeout`, `termination_signal=timeout`);
  - проверить `full-run.log` на watchdog progress/TERM/KILL;
  - сверить effective timeout values в summary и/или `GET /api/runtime/timeouts`.
- Quality regression / zero-signal fail:
  - проверить `session-summary.md` (секция failure reason),
  - сравнить `snapshots/<run_id>/...` между последними run,
  - посмотреть `reports/taskruns/<run_id>-quality.json`.
- Провал quality gates:
  - проверить `quality-gates.log`.
