# Release Live E2E Runbook (Agent, no wrapper)

Этот runbook фиксирует manual pre-release gate на trusted локальной машине.
Wrapper-скрипт не используется: запуск только через существующий matrix harness
`full-run-batch-matrix.sh -> full-run-batch-5x2.sh -> e2e_batch_report.py`.

## 1) Scope и ограничения

- Gate покрывает backend `5x2` + frontend `init-inspect-service-first` + frontend `cancel-refresh`.
- Для релизного решения выполняются две волны matrix-прогонов (строго последовательно).
- Product API/schema contracts не меняются.
- Gate режим: manual pre-release, не required CI merge gate.
- Verdict policy: strict zero-failure (`PASS|FAIL`).
- Release-mode matrix harness включает timeout safety guard:
  диагностические timeout override запрещены по умолчанию.

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
- активная GUI-сессия (headed browser для frontend e2e)

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

Готовые шаблоны:
- `examples/e2e-matrix.example.yaml`
- `examples/e2e-matrix.release-wave1.yaml`
- `examples/e2e-matrix.release-wave2.yaml`
- curated profile presets: `examples/repos/curated/*.repos.yaml`
- pinned GitHub presets: `examples/repos/github/*.repos.yaml`

### 3.1) GitHub catalog для выбора target repos (3 monorepo + 3 multi-repo)

Matrix использует 4 fixed-профиля (`single-path`, `single-git_url`, `multi-path`, `multi-git_url`),
поэтому выбор target repos делается через `repos_file`:
- `single-path`/`single-git_url`: один monorepo;
- `multi-path`/`multi-git_url`: один multi-repo проект (2+ repos).

Pinned GitHub presets (commit SHA):
- `examples/repos/github/mono-ftgo-application.repos.yaml`
- `examples/repos/github/mono-posthog.repos.yaml`
- `examples/repos/github/mono-bank-of-anthos.repos.yaml`
- `examples/repos/github/multi-sentry-ecosystem.repos.yaml`
- `examples/repos/github/multi-openstack-ecosystem.repos.yaml`
- `examples/repos/github/multi-openedx-ecosystem.repos.yaml`

Path presets с теми же pinned SHA:
- `examples/repos/curated/single-path.posthog.repos.yaml`
- `examples/repos/curated/multi-path.openedx.repos.yaml`
- `examples/repos/curated/single-path.bank-of-anthos.repos.yaml`
- `examples/repos/curated/multi-path.openstack.repos.yaml`

Важно:
- для `git_url` использовать только pinned `ref` (commit SHA);
- для `path` использовать локальные checkout тех же репозиториев на тех же SHA;
- перед релизным прогоном при необходимости обновить SHA в preset-файлах отдельным коммитом (без изменения harness логики).

### 3.2) Двухволновой release набор

Wave 1 (first-run queue):
- `single-path`: `posthog/posthog` (path preset)
- `single-git_url`: `microservices-patterns/ftgo-application`
- `multi-path`: `Open edX ecosystem` (path preset)
- `multi-git_url`: `getsentry/*`

Wave 2 (second-pass queue):
- `single-path`: `GoogleCloudPlatform/bank-of-anthos` (path preset)
- `single-git_url`: `GoogleCloudPlatform/bank-of-anthos`
- `multi-path`: `OpenStack ecosystem` (path preset)
- `multi-git_url`: `OpenStack ecosystem`

### 3.3) Подготовка path presets перед запуском

Path presets в `examples/repos/curated/*.repos.yaml` содержат placeholder-пути и требуют
локальных checkout’ов на вашей машине.

Обязательная проверка перед запуском Wave 1/2:

```bash
python3 - <<'PY'
import yaml
import subprocess
from pathlib import Path

files = [
    Path("examples/repos/curated/single-path.posthog.repos.yaml"),
    Path("examples/repos/curated/multi-path.openedx.repos.yaml"),
    Path("examples/repos/curated/single-path.bank-of-anthos.repos.yaml"),
    Path("examples/repos/curated/multi-path.openstack.repos.yaml"),
]
ok = True
for f in files:
    data = yaml.safe_load(f.read_text(encoding="utf-8")) or {}
    for i, repo in enumerate(data.get("repos", []), start=1):
        ref = str(repo.get("ref", "")).strip()
        p = str(repo.get("path", "")).strip()
        if not p:
            continue
        path = Path(p).expanduser().resolve()
        if not path.is_dir():
            ok = False
            print(f"{f}: repos[{i}] path does not exist: {path}")
            continue
        if not ref:
            ok = False
            print(f"{f}: repos[{i}] missing pinned ref for path repo: {path}")
            continue
        try:
            head = subprocess.check_output(["git", "-C", str(path), "rev-parse", "HEAD"], text=True).strip()
            ref_head = subprocess.check_output(["git", "-C", str(path), "rev-parse", f"{ref}^{{commit}}"], text=True).strip()
        except subprocess.CalledProcessError as exc:
            ok = False
            print(f"{f}: repos[{i}] git rev-parse failed for {path}: {exc}")
            continue
        if head != ref_head:
            ok = False
            print(f"{f}: repos[{i}] HEAD mismatch for {path}: head={head} ref={ref} resolved={ref_head}")
        status = subprocess.check_output(["git", "-C", str(path), "status", "--porcelain"], text=True).strip()
        if status:
            ok = False
            print(f"{f}: repos[{i}] dirty checkout for {path}")
        tracked = subprocess.check_output(["git", "-C", str(path), "ls-files"], text=True)
        if len([line for line in tracked.splitlines() if line.strip()]) <= 0:
            ok = False
            print(f"{f}: repos[{i}] zero tracked files for {path}")
if not ok:
    raise SystemExit(1)
print("path presets are valid (exists + pinned-ref + clean + tracked files)")
PY
```

Если path-checkout’ов нет, используйте fallback `git_url-only` matrix (ниже) и сохраните
этот прогон как preflight evidence, а не final release-ready verdict.

### 3.4) Fallback: git_url-only matrix для Wave 1/2

Когда `path`-checkout’ы недоступны, можно временно запускать волны на публичных `git_url`
репозиториях с сохранением профильных id (`single-path`, `multi-path`) для совместимости
matrix-отчётов:

```bash
# wave1
cat > /tmp/e2e-matrix.release-wave1.giturl-only.yaml <<'YAML'
version: 1
sweeps:
  - id: baseline
    strategy: sequential
    max_parallel_tasks: 1
    failure_policy: best_effort
    shard_discovery_mode: heuristics
    repo_selection: all
  - id: scale-backend
    strategy: parallel
    max_parallel_tasks: 4
    failure_policy: best_effort
    shard_discovery_mode: semantic
    repo_selection: backend_only
profiles:
  - id: single-path
    source_kind: git_url
    expected_repo_count: 1
    repos_file: ./examples/repos/github/mono-posthog.repos.yaml
  - id: single-git_url
    source_kind: git_url
    expected_repo_count: 1
    repos_file: ./examples/repos/github/mono-ftgo-application.repos.yaml
  - id: multi-path
    source_kind: git_url
    expected_repo_count: 5
    repos_file: ./examples/repos/github/multi-openedx-ecosystem.repos.yaml
  - id: multi-git_url
    source_kind: git_url
    expected_repo_count: 5
    repos_file: ./examples/repos/github/multi-sentry-ecosystem.repos.yaml
YAML

# wave2
cat > /tmp/e2e-matrix.release-wave2.giturl-only.yaml <<'YAML'
version: 1
sweeps:
  - id: baseline
    strategy: sequential
    max_parallel_tasks: 1
    failure_policy: best_effort
    shard_discovery_mode: heuristics
    repo_selection: all
  - id: scale-backend
    strategy: parallel
    max_parallel_tasks: 4
    failure_policy: best_effort
    shard_discovery_mode: semantic
    repo_selection: backend_only
profiles:
  - id: single-path
    source_kind: git_url
    expected_repo_count: 1
    repos_file: ./examples/repos/github/mono-bank-of-anthos.repos.yaml
  - id: single-git_url
    source_kind: git_url
    expected_repo_count: 1
    repos_file: ./examples/repos/github/mono-bank-of-anthos.repos.yaml
  - id: multi-path
    source_kind: git_url
    expected_repo_count: 5
    repos_file: ./examples/repos/github/multi-openstack-ecosystem.repos.yaml
  - id: multi-git_url
    source_kind: git_url
    expected_repo_count: 5
    repos_file: ./examples/repos/github/multi-openstack-ecosystem.repos.yaml
YAML
```

## 4) Порядок запуска (строго последовательно)

1. Preflight ACP quality:

```bash
make contracts test lint build
npm ci --prefix ui
npm exec --prefix ui playwright install chromium
```

2. Запуск Wave 1:

```bash
MATRIX_ID=release-wave1-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=./examples/e2e-matrix.release-wave1.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
MATRIX_MAX_PARALLEL_COMBINATIONS=2 \
BATCH_MAX_PARALLEL_RUNS=2 \
BATCH_FRONTEND_MAX_PARALLEL=1 \
BATCH_FRONTEND_MODE=per_run \
BATCH_FRONTEND_CANCEL_MODE=once_per_provider \
UI_E2E_HEADED=1 \
DOCUMENTATION_AUDIT_MANUAL_STATUS=passed \
DOCUMENTATION_AUDIT_MANUAL_NOTES="manual checklist completed for wave1" \
./scripts/full-run-batch-matrix.sh
```

3. Запуск Wave 2 (только после завершения Wave 1):

```bash
MATRIX_ID=release-wave2-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=./examples/e2e-matrix.release-wave2.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
MATRIX_MAX_PARALLEL_COMBINATIONS=2 \
BATCH_MAX_PARALLEL_RUNS=2 \
BATCH_FRONTEND_MAX_PARALLEL=1 \
BATCH_FRONTEND_MODE=per_run \
BATCH_FRONTEND_CANCEL_MODE=once_per_provider \
UI_E2E_HEADED=1 \
DOCUMENTATION_AUDIT_MANUAL_STATUS=passed \
DOCUMENTATION_AUDIT_MANUAL_NOTES="manual checklist completed for wave2" \
./scripts/full-run-batch-matrix.sh
```

4. Release guard rules:
- Не задавать diagnostic timeout env для релизного запуска:
  - `ACP_RUNTIME_*`, `ACP_PIPELINE_*`, `ACP_API_*`, `ACP_UI_*`, `READY_TIMEOUT_SEC`, `UI_E2E_*_TIMEOUT_SEC`.
- Если нужен диагностический прогон с override, явно включать:
  - `E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES=1`
  - Такой прогон не использовать как release verdict.

5. Release-ready решение принимается только если обе волны завершились `PASS`.

## 4.1) Live monitoring during run

Во время matrix-run держите отдельный терминал с мониторингом:

```bash
# matrix driver log (high-level progress)
tail -f /tmp/provenarch-test_arch_project/matrix/<matrix-id>/driver.log

# сколько profile+sweep уже завершилось
watch -n 5 'ls -1 /tmp/provenarch-test_arch_project/matrix/<matrix-id>/records/*.json 2>/dev/null | wc -l'

# активные процессы harness
ps -axo pid,ppid,command | rg "full-run-batch-matrix.sh|full-run-batch-5x2.sh|full-run-ai-advent.sh|frontend-live-e2e.sh"
```

Проверка parallel-факта по логам:
- в matrix log: `max_parallel_combinations=<N>`;
- в batch log: `backend_max_parallel_runs=<N> frontend_max_parallel_runs=<M>`.

## 4.2) Controlled stop of long run + failed evidence

Если run нужно прервать:

```bash
# 1) найти matrix pid(s) по matrix-id и отправить TERM
matrix_pids="$(pgrep -f "full-run-batch-matrix.sh.*<matrix-id>" || true)"
if [[ -n "$matrix_pids" ]]; then
  kill -TERM ${matrix_pids} || true
fi

# 2) дать 20-30s на завершение child jobs
sleep 30

# 3) если matrix pid всё ещё жив — force kill только их
if [[ -n "$matrix_pids" ]]; then
  for pid in ${matrix_pids}; do
    kill -KILL "$pid" 2>/dev/null || true
  done
fi
```

Если вы параллельно не запускаете другие harness-runs, и остались orphan child-процессы,
можно добавить аварийный cleanup:

```bash
pkill -9 -f "full-run-ai-advent.sh|frontend-live-e2e.sh" || true
```

После остановки зафиксируйте failed evidence:

```bash
stopped_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
cat >"/tmp/provenarch-test_arch_project/reports/manual_stop_<matrix-id>.md" <<EOF
# Manual Stop Evidence: <matrix-id>

- status: failed
- reason: manual_stop
- stopped_at_utc: ${stopped_at}
- matrix_driver_log: /tmp/provenarch-test_arch_project/matrix/<matrix-id>/driver.log
- partial_profile_matrix: /tmp/provenarch-test_arch_project/reports/profile_matrix_<matrix-id>.md
- partial_release_verdict: /tmp/provenarch-test_arch_project/reports/release_verdict_<matrix-id>.json
EOF
```

`manual_stop_<matrix-id>.md` считается обязательным evidence для прерванного run.

## 5) Release evidence artifacts

Обязательные:
- `profile_matrix_<matrix-id>.md`
- `profile_matrix_<matrix-id>.tsv`
- `release_verdict_<matrix-id>.md`
- `release_verdict_<matrix-id>.json`
- `run_matrix_<batch-id>.md`
- `run_matrix_<batch-id>.tsv`
- `quality_report_<batch-id>.md`
- `frontend_e2e_matrix_<batch-id>.md` (run-level: `provider + run`)
- `frontend_cancel_e2e_matrix_<batch-id>.md` (run-level / mode-dependent)
- `documentation_audit_<batch-id>.md`

Дополнительные для triage:
- `/tmp/provenarch-test_arch_project/runs/<batch-id>/<provider>/runN/*`
- `backend-run-classifications.tsv`
- `preflight.json`
- `driver.log`, `full-run.log`, `session-summary.md`
- `manual_stop_<matrix-id>.md` (обязательно для вручную прерванного run)

## 6) Documentation audit contract

`documentation_audit_<batch-id>.md` включает:
- auto rubric (на каждый backend run):
  - `as-is`
  - `findings`
  - `coverage`
  - `open-questions`
  - `service-inventory`
  - `global-review`
  - `evidence-scope`
  - `cross-doc`
- manual checklist completion status (`pending|passed|failed`);
- implementation audit section с traceability-проверками новой логики.

Strict gate использует только scalar-поля из audit:
- `auto_status`
- `manual_status`
- `implementation_audit_status`

## 7) Traceability matrix (new logic -> PASS criterion -> evidence)

| Feature | PASS criterion | Evidence artifact |
|---|---|---|
| service-first step contract | Для init/refresh есть `service-inventory-plan` и `global-review` taskrun artifacts | `documentation_audit_<batch-id>.md` / `reports/taskruns/*` |
| shard artifacts/metadata | Нет `runtime:shard-artifacts` и `runtime:shard-metadata` | `run_matrix_<batch-id>.md/.tsv` |
| repo selection semantics | Нет `runtime:repo-selection` и `analysis:cross-repo-missing` | `run_matrix_<batch-id>.md/.tsv`, `quality_report_<batch-id>.md` |
| execution profile semantics | Нет `runtime:execution-semantics` и `runtime_flow_failed` | `run_matrix_<batch-id>.md/.tsv`, `profile_matrix_<matrix-id>.md` |
| timeout precedence/guard | Нет `runtime_timeout` и `precheck_failed` | `run_matrix_<batch-id>.md/.tsv`, `release_verdict_<matrix-id>.json` |
| path checkout health guardrail | Для `source=path` нет dirty/mismatched/empty checkout | `preflight.json`, `backend-run-classifications.tsv`, `release_verdict_<matrix-id>.json` |
| cancel lifecycle | Frontend `cancel-refresh` status=`passed` для обоих провайдеров | `frontend_cancel_e2e_matrix_<batch-id>.md` |
| semantic hard-fail checks | `semantic_hard_fail=0`, `off_topic_hits=0`, нет `analysis:evidence-scope` | `run_matrix_<batch-id>.md/.tsv`, `quality_report_<batch-id>.md` |
| snapshot-only baseline | Для всех backend run `artifact_source=snapshot` | `run_matrix_<batch-id>.md/.tsv` |

Targeted code-audit этих зон фиксируется в implementation-audit части release evidence.

## 8) Strict release acceptance

Release `PASS` только если одновременно:
1. Во всех `profile+sweep` строках `strict_status=passed`.
2. Для каждого `profile+sweep`: `backend_total_runs=10`, `backend_hard_pass=10`.
3. `runtime_parse/runner_unavailable/runtime_timeout/infra_signal_terminated/infra_incomplete_cycle/quality_gates_failed/summary_missing/precheck_failed = 0`.
4. `semantic_hard_fail=0`, `off_topic_hits=0`.
5. `artifact_source` только `snapshot` (без `workspace-fallback`).
6. Нет `analysis:evidence-scope` и `analysis:cross-repo-missing`.
7. Нет runtime flow violations (`runtime:*`, `runtime_flow_failed`).
   - включая `runtime:empty-service-inventory`.
8. Frontend init smoke:
  - любой run-level status в `frontend_e2e_matrix` должен быть `passed` (`provider + run`),
  - provider summary status должен быть `passed`.
9. Frontend cancel smoke: provider summary status=`passed` для `qwen-code` и `claude-code`.
10. Documentation audit: `auto_status=passed` и `manual_status=passed`.

Любое нарушение => `RELEASE BLOCKED`.

## 9) Agent verdict output

Источник истины для финального решения:
- `release_verdict_<matrix-id>.json`

Минимальный формат публикации агентом:

```text
VERDICT: PASS|FAIL
Matrix ID: <matrix-id>
Release State: RELEASE READY|RELEASE BLOCKED
Blocking reasons:
- ...
Evidence:
- /tmp/provenarch-test_arch_project/reports/release_verdict_<matrix-id>.json
- /tmp/provenarch-test_arch_project/reports/profile_matrix_<matrix-id>.md
- /tmp/provenarch-test_arch_project/reports/documentation_audit_<batch-id>.md
```
