# Release Live E2E Runbook (Agent, no wrapper)

Этот runbook фиксирует manual pre-release gate на trusted локальной машине.
Новый wrapper-скрипт не используется: агент запускает существующий matrix harness напрямую (`full-run-batch-matrix.sh` -> `full-run-batch-5x2.sh` -> `e2e_batch_report.py`).

## 1) Scope и ограничения

- Gate покрывает backend `5x2` + frontend `init-inspect` + frontend `cancel-refresh`.
- Product API/schema contracts не меняются.
- Gate режим: manual pre-release, не required CI merge gate.
- Verdict policy: strict zero-failure (`PASS|FAIL`).
- В release-mode matrix harness включает timeout safety guard: диагностические timeout override запрещены по умолчанию.

## 2) Prerequisites

Проверить на машине:
- `go`
- `npm`
- `python3`
- `curl`
- `qwen`
- `claude`
- доступ к `path` repos
- доступ к `git_url` repos с pinned `ref`

## 3) Matrix input contract

`E2E_MATRIX_FILE` содержит:
- `profiles[]` (обязательные 4 профиля):
  - `single-path`
  - `single-git_url`
  - `multi-path`
  - `multi-git_url`
- `sweeps[]` (optional, backward-compatible):
  - если отсутствует -> implicit `baseline`
  - release-ready harness использует 2 sweep-профиля:
    - `baseline`: `strategy=sequential`, `max_parallel_tasks=1`, `failure_policy=best_effort`, `shard_discovery_mode=heuristics`, `repo_selection=all`
    - `scale-backend`: `strategy=parallel`, `max_parallel_tasks=4`, `failure_policy=best_effort`, `shard_discovery_mode=semantic`, `repo_selection=backend_only`

Готовый шаблон:
- `examples/e2e-matrix.example.yaml`
- curated profile presets: `examples/repos/curated/*.repos.yaml`
- pinned GitHub presets: `examples/repos/github/*.repos.yaml`

### 3.1) GitHub catalog для выбора target repos (3 monorepo + 3 multi-repo)

Причина: release matrix выше использует 4 fixed-профиля (`single-path`, `single-git_url`, `multi-path`, `multi-git_url`), поэтому для GitHub-only сценария выбор делается через `repos_file`:
- `single-path`/`single-git_url`: выбрать один monorepo;
- `multi-path`/`multi-git_url`: выбрать один multi-repo проект (2+ repos).

Pinned presets (commit SHA) добавлены в:
- `examples/repos/github/mono-ftgo-application.repos.yaml`
- `examples/repos/github/mono-posthog.repos.yaml`
- `examples/repos/github/mono-bank-of-anthos.repos.yaml`
- `examples/repos/github/multi-sentry-ecosystem.repos.yaml`
- `examples/repos/github/multi-openstack-ecosystem.repos.yaml`
- `examples/repos/github/multi-openedx-ecosystem.repos.yaml`

Monorepo выбор (single):
- `microservices-patterns/ftgo-application` (`master`, pinned SHA в preset)
- `posthog/posthog` (`master`, pinned SHA в preset)
- `GoogleCloudPlatform/bank-of-anthos` (`main`, pinned SHA в preset)

Multi-repo выбор (multi):
- Sentry ecosystem: `getsentry/sentry`, `getsentry/self-hosted`, `getsentry/snuba`, `getsentry/relay`, `getsentry/symbolicator`
- OpenStack ecosystem: `openstack/openstack`, `openstack/nova`, `openstack/neutron`, `openstack/cinder`, `openstack/keystone`
- Open edX ecosystem: `openedx/openedx-platform`, `openedx/frontend-platform`, `openedx/course-discovery`, `openedx/credentials`, `openedx/devstack`

Примечание по Open edX:
- в preset используется canonical `git_url` для devstack: `openedx-unsupported/devstack` (репозиторий `openedx/devstack` редиректит туда).

Пример wiring в release matrix:

```yaml
profiles:
  - id: single-path
    source_kind: path
    expected_repo_count: 1
    repos_file: /abs/path/to/local-clones/mono-ftgo-application.path.repos.yaml
  - id: single-git_url
    source_kind: git_url
    expected_repo_count: 1
    repos_file: ./examples/repos/github/mono-posthog.repos.yaml
  - id: multi-path
    source_kind: path
    expected_repo_count: 5
    repos_file: /abs/path/to/local-clones/multi-openedx.path.repos.yaml
  - id: multi-git_url
    source_kind: git_url
    expected_repo_count: 5
    repos_file: ./examples/repos/github/multi-sentry-ecosystem.repos.yaml
```

Важно:
- для `git_url` использовать только pinned `ref` (commit SHA);
- для `path` использовать локальные checkout тех же репозиториев на тех же SHA;
- перед релизным прогоном при необходимости обновить SHA в preset-файлах отдельным коммитом (без изменения harness логики).
- запасной monorepo-кандидат: `GoogleCloudPlatform/microservices-demo` (если нужно заменить `bank-of-anthos`).
- рекомендуемый first-run набор: `posthog/posthog` + `microservices-patterns/ftgo-application` + `getsentry/*` + `Open edX ecosystem`.
- рекомендуемый second-pass набор: `GoogleCloudPlatform/bank-of-anthos` + `OpenStack ecosystem`.

## 4) Порядок запуска

1. Preflight ACP quality:

```bash
make contracts test lint build
npm ci --prefix ui
npm exec --prefix ui playwright install chromium
```

2. Matrix live harness:

```bash
E2E_MATRIX_FILE=/abs/path/to/e2e-matrix.release.yaml \
MATRIX_ID=release-$(date -u +%Y%m%dT%H%M%SZ) \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh
```

Release guard rules:
- Не задавать diagnostic timeout env для релизного запуска:
  - `ACP_RUNTIME_*`, `ACP_PIPELINE_*`, `ACP_API_*`, `ACP_UI_*`, `READY_TIMEOUT_SEC`, `UI_E2E_*_TIMEOUT_SEC`.
- Если нужен диагностический прогон с override, явно включать:
  - `E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES=1`
  - Такой прогон не использовать как release verdict.

3. Сбор артефактов из:
- `/tmp/provenarch-test_arch_project/reports/`

## 5) Release evidence artifacts

Обязательные:
- `profile_matrix_<matrix-id>.md`
- `profile_matrix_<matrix-id>.tsv`
- `release_verdict_<matrix-id>.md`
- `release_verdict_<matrix-id>.json`
- `run_matrix_<batch-id>.md`
- `run_matrix_<batch-id>.tsv`
- `quality_report_<batch-id>.md`
- `frontend_e2e_matrix_<batch-id>.md`
- `frontend_cancel_e2e_matrix_<batch-id>.md`

Дополнительные для triage:
- `/tmp/provenarch-test_arch_project/runs/<batch-id>/<provider>/runN/*`
- `backend-run-classifications.tsv`
- `preflight.json`
- `driver.log`, `full-run.log`, `session-summary.md`

## 6) Flow проверки новой функциональности

### 6.1 Runtime sharding

Проверяем:
- shard artifacts для `init`/`refresh`: `shard-plan`, `shard-summary`, per-shard taskruns;
- taskrun metadata: `meta.shard_id`, `meta.repo_scopes`, `meta.path_scopes`.

Blocking signals:
- `runtime:shard-artifacts`
- `runtime:shard-metadata`

### 6.2 Repo selection hardening

Проверяем:
- `repo_selection=all|backend_only` vs `analysis.role` decisions;
- наличие include/exclude reason в repo-selection summary;
- отсутствие включения frontend-only repos в `backend_only`.

Blocking signals:
- `runtime:repo-selection`
- `analysis:cross-repo-missing`

### 6.3 Execution profile semantics

Проверяем:
- consistency `strategy`, `max_parallel_tasks`, `failure_policy`, `shard_discovery_mode`, `repo_selection` по effective surfaces + shard artifacts + run results;
- корректную реакцию на partial shard failures по policy.

Blocking signals:
- `runtime:execution-semantics`
- `runtime_flow_failed`
- `run_partial_failed` policy mismatch в details/issue evidence.

### 6.4 Timeout control

Проверяем:
- `ACP_APPLY_TIMEOUTS_VIA_API=1` применяет timeout profile без ошибок;
- precedence `env > workspace > defaults` отражается в preflight/runtime surfaces;
- в release-mode не используются diagnostic timeout overrides (guard fail-fast до batch start);
- отсутствуют ложные timeout failures.

Blocking signals:
- `runtime_timeout`

### 6.5 Run lifecycle + cancel

Проверяем:
- frontend `init-inspect` = `passed` для `qwen-code` и `claude-code`;
- frontend `cancel-refresh` = `passed` для `qwen-code` и `claude-code`;
- preflight cancel-timeout guard соблюдён: `UI_E2E_CANCEL_TIMEOUT_SEC >= UI_E2E_CANCEL_STUB_SLEEP_SEC + UI_E2E_CANCEL_TIMEOUT_MARGIN_SEC`;
- cancel сценарий завершается `failed + run_canceled`.

Blocking signals:
- любой frontend status != `passed`

### 6.6 Runtime parsing/diagnostics stability

Zero tolerance:
- `runtime_parse`
- `runner_unavailable`
- `summary_missing`
- `infra_signal_terminated`
- `infra_incomplete_cycle`
- `quality_gates_failed`
- `precheck_failed`

### 6.7 Triage rule for runtime timeout/infra signals

Если в любом `profile+sweep` появляются `runtime_timeout` или `infra_*`:
- считать это blocking runtime incident для релизного verdict;
- сначала проверить `driver.log` (matrix + batch), затем `session-summary.md` и `full-run.log` в `runs/<batch-id>/<provider>/runN/`;
- отделить induced failures (например, debug timeout override) от реальной runtime/provider деградации;
- для release decision использовать только прогон без diagnostic timeout overrides.

## 7) Strict release acceptance

Release `PASS` только если одновременно:
1. Во всех `profile+sweep` строках `strict_status=passed`.
2. Для каждого `profile+sweep`: `backend_total_runs=10`, `backend_hard_pass=10`.
3. `runtime_parse/runner_unavailable/runtime_timeout/infra_signal_terminated/infra_incomplete_cycle/quality_gates_failed/summary_missing/precheck_failed = 0`.
4. `semantic_hard_fail=0`, `off_topic_hits=0`.
5. `artifact_source` только `snapshot` (без `workspace-fallback`).
6. Нет `analysis:evidence-scope` и `analysis:cross-repo-missing`.
7. Нет runtime flow violations (`runtime:*`, `runtime_flow_failed`).
8. Frontend live/cancel smoke: `passed` для обоих провайдеров.

Любое нарушение => `RELEASE BLOCKED`.

## 8) Agent verdict output

Источник истины для финального решения:
- `release_verdict_<matrix-id>.md/.json`

Минимальный формат публикации агентом:

```text
VERDICT: PASS|FAIL
Matrix ID: <matrix-id>
Release State: RELEASE READY|RELEASE BLOCKED
Blocking reasons:
- ...
Evidence:
- /tmp/provenarch-test_arch_project/reports/release_verdict_<matrix-id>.md
- /tmp/provenarch-test_arch_project/reports/profile_matrix_<matrix-id>.md
```
