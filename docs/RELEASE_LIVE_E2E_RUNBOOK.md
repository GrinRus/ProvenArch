# Release Live E2E Runbook (Agent, no wrapper)

Этот runbook фиксирует manual pre-release gate на trusted локальной машине.
Новый wrapper-скрипт не используется: агент запускает существующий matrix harness напрямую (`full-run-batch-matrix.sh` -> `full-run-batch-5x2.sh` -> `e2e_batch_report.py`).

## 0) Canonical release targets (official waves)

Release gate в проекте использует два canonical matrix-файла:
- `examples/e2e-matrix.release-wave1.yaml`
- `examples/e2e-matrix.release-wave2.yaml`

Для official release verdict использовать только эти matrix-файлы как source of truth. Каталог repo targets ниже помогает понимать scope/pinned presets, но не заменяет canonical waves.

Wave 1:
- `single-path`: `posthog/posthog`
- `single-git_url`: `microservices-patterns/ftgo-application`
- `multi-path`: Open edX ecosystem (`openedx-platform`, `frontend-platform`, `course-discovery`, `credentials`, `devstack`)
- `multi-git_url`: Sentry ecosystem (`sentry`, `self-hosted`, `snuba`, `relay`, `symbolicator`)

Wave 2:
- `single-path`: `GoogleCloudPlatform/bank-of-anthos`
- `single-git_url`: `GoogleCloudPlatform/bank-of-anthos`
- `multi-path`: OpenStack ecosystem (`openstack`, `nova`, `neutron`, `cinder`, `keystone`)
- `multi-git_url`: OpenStack ecosystem (`openstack`, `nova`, `neutron`, `cinder`, `keystone`)

Release verdict для readiness берётся только из:
- `reports/release_verdict_<matrix-id>.json`

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

### 2.1) Fail-fast host eligibility

Эту проверку выполнять до DoD/preflight и до старта official wave matrices:

```bash
python3 - <<'PY'
from pathlib import Path
import os
root = Path("/tmp/provenarch-live-e2e")
parent = root.parent
print(f"root={root}")
print(f"root_exists={root.exists()}")
print(f"root_is_dir={root.is_dir() if root.exists() else False}")
print(f"parent_writable={parent.exists() and os.access(parent, os.W_OK)}")
print(f"root_writable={os.access(root, os.W_OK) if root.exists() else 'n/a'}")
PY
```

Если результат показывает `parent_writable=false`, либо existing `root` не является writable directory:
- текущий хост не подходит для official release gate с canonical curated `path` profiles под `/tmp/provenarch-live-e2e`;
- `wave1`/`wave2` release matrices здесь запускать нельзя;
- нужно перенести прогон на другой trusted host, а не переписывать matrix/curated files под локальную машину.

### 2.2) One-time canonical path bootstrap

Если хост подходит по `2.1`, но curated `path` checkout'ы ещё не существуют, их нужно подготовить один раз заранее.

Минимальное требование:
- каждый `path` из `examples/repos/curated/*.repos.yaml` должен существовать локально;
- каждый checkout должен быть ровно на pinned SHA из соответствующего `github` preset;
- official release gate запускается только после этого bootstrap.

Есть два устойчивых варианта:
- preferred: реальные локальные clone в exact canonical путях;
- acceptable: symlink в canonical `/tmp/provenarch-live-e2e/...` path на уже существующий локальный clone, если symlink ведёт в рабочую git-директорию и checkout тоже зафиксирован на pinned SHA.

Важно:
- `/tmp` удобен для локального bootstrap, но может очищаться системой;
- если нужен truly stable trusted host setup, bootstrap должен быть либо легко повторяемым, либо вынесенным на persistent volume, смонтированный в `/tmp/provenarch-live-e2e`.

Пример one-time bootstrap команд для path profiles:

```bash
# canonical parent dirs
mkdir -p \
  /tmp/provenarch-live-e2e/posthog \
  /tmp/provenarch-live-e2e/openedx \
  /tmp/provenarch-live-e2e/openedx-unsupported \
  /tmp/provenarch-live-e2e/GoogleCloudPlatform \
  /tmp/provenarch-live-e2e/openstack

# wave1 single-path
git clone https://github.com/posthog/posthog.git /tmp/provenarch-live-e2e/posthog/posthog
git -C /tmp/provenarch-live-e2e/posthog/posthog checkout --detach 14d29a548d63665d60b506cf13bd5cfb2de7c743

# wave1 multi-path (Open edX)
git clone https://github.com/openedx/openedx-platform.git /tmp/provenarch-live-e2e/openedx/openedx-platform
git -C /tmp/provenarch-live-e2e/openedx/openedx-platform checkout --detach 01dc3c84ea58d2e8b8181a90e89d6c9017aceee8

git clone https://github.com/openedx/frontend-platform.git /tmp/provenarch-live-e2e/openedx/frontend-platform
git -C /tmp/provenarch-live-e2e/openedx/frontend-platform checkout --detach 44cc02429404ed3547c20abd834faf4e487d2c00

git clone https://github.com/openedx/course-discovery.git /tmp/provenarch-live-e2e/openedx/course-discovery
git -C /tmp/provenarch-live-e2e/openedx/course-discovery checkout --detach 127ea98b0ed6d05b955ea0fc8c57b9c0c285a9a5

git clone https://github.com/openedx/credentials.git /tmp/provenarch-live-e2e/openedx/credentials
git -C /tmp/provenarch-live-e2e/openedx/credentials checkout --detach 88767bc79e8908a2c731f8c099e917fb8454bd5e

git clone https://github.com/openedx-unsupported/devstack.git /tmp/provenarch-live-e2e/openedx-unsupported/devstack
git -C /tmp/provenarch-live-e2e/openedx-unsupported/devstack checkout --detach 28f6d7ea1fa30fd7e0bdc10f269999f15f7f8876

# wave2 single-path
git clone https://github.com/GoogleCloudPlatform/bank-of-anthos.git /tmp/provenarch-live-e2e/GoogleCloudPlatform/bank-of-anthos
git -C /tmp/provenarch-live-e2e/GoogleCloudPlatform/bank-of-anthos checkout --detach 7f0589c7aaf0e009aacb4cd9e2e8f26bd30061e1

# wave2 multi-path (OpenStack)
git clone https://github.com/openstack/openstack.git /tmp/provenarch-live-e2e/openstack/openstack
git -C /tmp/provenarch-live-e2e/openstack/openstack checkout --detach 32e939e38f8ff7f91d593e3be94240590afd4db2

git clone https://github.com/openstack/nova.git /tmp/provenarch-live-e2e/openstack/nova
git -C /tmp/provenarch-live-e2e/openstack/nova checkout --detach b8340c9361b2a2e9473ba4e8abaabc372360c6ee

git clone https://github.com/openstack/neutron.git /tmp/provenarch-live-e2e/openstack/neutron
git -C /tmp/provenarch-live-e2e/openstack/neutron checkout --detach f27e72c44246a3281f882201f91c88896e2adbe6

git clone https://github.com/openstack/cinder.git /tmp/provenarch-live-e2e/openstack/cinder
git -C /tmp/provenarch-live-e2e/openstack/cinder checkout --detach 04f5a13c0376859711e69f3916409cb40f700e0f

git clone https://github.com/openstack/keystone.git /tmp/provenarch-live-e2e/openstack/keystone
git -C /tmp/provenarch-live-e2e/openstack/keystone checkout --detach 80d5b7bf50448073223723cf1f6001a367695e80
```

Если репозитории уже выкачаны в другом месте, можно использовать symlink bootstrap вместо повторных clone:

```bash
mkdir -p /tmp/provenarch-live-e2e/posthog
ln -s /real/local/path/posthog /tmp/provenarch-live-e2e/posthog/posthog
git -C /real/local/path/posthog checkout --detach 14d29a548d63665d60b506cf13bd5cfb2de7c743
```

После bootstrap обязательно прогнать path/SHA verify script из `4) Порядок запуска` перед официальными matrices.

Обязательное условие для path-профилей:
- локальные checkout должны быть на тех же pinned SHA, что и соответствующие `github` presets;
- canonical пути из `examples/repos/curated/*.repos.yaml` должны реально существовать на trusted машине;
- если `/tmp/provenarch-live-e2e` не может быть создан или системно очищается слишком агрессивно, использовать persistent mount/volume под этим canonical root.
- не переписывать canonical `examples/e2e-matrix.release-wave*.yaml` или `examples/repos/curated/*.repos.yaml` только ради обхода ограничений текущей машины.

## 3) Matrix input contract

`E2E_MATRIX_FILE` содержит:
- `profiles[]` (обязательные 4 профиля):
  - `single-path`
  - `single-git_url`
  - `multi-path`
  - `multi-git_url`
- `sweeps[]` (optional):
  - если отсутствует -> implicit `baseline`
  - release-ready harness использует 2 sweep-профиля:
    - `baseline`: `strategy=sequential`, `max_parallel_tasks=1`, `failure_policy=best_effort`, `shard_discovery_mode=heuristics`
    - `parallel-default`: `strategy=parallel`, `max_parallel_tasks=4`, `failure_policy=best_effort`, `shard_discovery_mode=heuristics`

Готовый шаблон:
- `examples/e2e-matrix.example.yaml`
- `examples/repos/*.repos.yaml`

### 3.1) GitHub catalog для выбора target repos (3 monorepo + 3 multi-repo)

Причина: release matrix выше использует 4 fixed-профиля (`single-path`, `single-git_url`, `multi-path`, `multi-git_url`), поэтому для GitHub-only сценария выбор делается через `repos_file`:
- `single-path`/`single-git_url`: выбрать один monorepo;
- `multi-path`/`multi-git_url`: выбрать один multi-repo проект (2+ repos).

Важно:
- этот каталог фиксирует approved target set и pinned preset-файлы;
- для official release gate решение всё равно принимается только по `examples/e2e-matrix.release-wave1.yaml` и `examples/e2e-matrix.release-wave2.yaml`.

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
    repos_file: ./repos/curated/single-path.posthog.repos.yaml
  - id: single-git_url
    source_kind: git_url
    expected_repo_count: 1
    repos_file: ./repos/github/mono-ftgo-application.repos.yaml
  - id: multi-path
    source_kind: path
    expected_repo_count: 5
    repos_file: ./repos/curated/multi-path.openedx.repos.yaml
  - id: multi-git_url
    source_kind: git_url
    expected_repo_count: 5
    repos_file: ./repos/github/multi-sentry-ecosystem.repos.yaml
```

Важно:
- для `git_url` использовать только pinned `ref` (commit SHA);
- для `path` использовать локальные checkout тех же репозиториев на тех же SHA;
- перед релизным прогоном при необходимости обновить SHA в preset-файлах отдельным коммитом (без изменения harness логики).
- запасной monorepo-кандидат: `GoogleCloudPlatform/microservices-demo` (если нужно заменить `bank-of-anthos`).
- рекомендуемый first-run набор для exploratory/live shakeout: `posthog/posthog` + `microservices-patterns/ftgo-application` + `getsentry/*` + `Open edX ecosystem`; он не заменяет official release gate по двум wave matrices.

## 4) Порядок запуска

1. Preflight ACP quality:

```bash
make contracts test lint build
npm ci --prefix ui
npm exec --prefix ui playwright install chromium
```

2. Проверка path targets (exist + pinned SHA):

```bash
python3 - <<'PY'
import json, subprocess, sys
from pathlib import Path

files = [
    "examples/repos/curated/single-path.posthog.repos.yaml",
    "examples/repos/curated/multi-path.openedx.repos.yaml",
    "examples/repos/curated/single-path.bank-of-anthos.repos.yaml",
    "examples/repos/curated/multi-path.openstack.repos.yaml",
]

ok = True
for rel in files:
    payload = Path(rel).read_text(encoding="utf-8")
    # lightweight YAML parse without deps: keep runbook script zero-dependency
    repos = []
    current = {}
    for line in payload.splitlines():
        s = line.strip()
        if s.startswith("- name:"):
            if current:
                repos.append(current)
            current = {"name": s.split(":", 1)[1].strip()}
        elif s.startswith("path:"):
            current["path"] = s.split(":", 1)[1].strip()
        elif s.startswith("ref:"):
            current["ref"] = s.split(":", 1)[1].strip()
    if current:
        repos.append(current)
    for r in repos:
        p = Path(r["path"])
        if not p.exists():
            ok = False
            print(f"MISSING path: {p} ({r.get('name')})")
            continue
        try:
            head = subprocess.check_output(
                ["git", "-C", str(p), "rev-parse", "HEAD"],
                text=True,
            ).strip()
        except Exception as exc:
            ok = False
            print(f"NOT_GIT_OR_UNREADABLE: {p} ({exc})")
            continue
        ref = r.get("ref", "")
        if ref and head != ref:
            ok = False
            print(f"SHA_MISMATCH: {p} expected={ref} got={head}")
        else:
            print(f"OK {p} @ {head}")

if not ok:
    sys.exit(1)
PY
```

Если этот preflight не проходит:
- не править canonical matrix/curated preset под текущий хост;
- остановить release gate как operational blocker и перенести прогон на подходящую trusted машину.

3. Official release matrix (wave1 + wave2):

```bash
MATRIX_ID=release-$(date -u +%Y%m%dT%H%M%SZ)-wave1 \
E2E_MATRIX_FILE=examples/e2e-matrix.release-wave1.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh

MATRIX_ID=release-$(date -u +%Y%m%dT%H%M%SZ)-wave2 \
E2E_MATRIX_FILE=examples/e2e-matrix.release-wave2.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh
```

В release-mode (`MATRIX_ID=release-*`) matrix harness автоматически выставляет:
- `BATCH_FRONTEND_MODE=per_run`
- `BATCH_FRONTEND_CANCEL_MODE=once_per_provider`
- `UI_E2E_HEADED=1`

Release guard rules:
- Не задавать diagnostic timeout env для релизного запуска:
  - `ACP_RUNTIME_*`, `ACP_PIPELINE_*`, `ACP_API_*`, `ACP_UI_*`.
- Если нужен диагностический прогон с override, явно включать:
  - `E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES=1`
  - Такой прогон не использовать как release verdict.

4. Сбор артефактов из:
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

### 6.2 Structural shard-plan invariant

Проверяем:
- один и тот же profile под `baseline` и `parallel-default` даёт идентичный shard-plan;
- planner покрывает весь repo неперекрывающимися `path_scopes`, без дыр и residual overlap;
- для больших repo structural coalescing не смешивает разные top-level subtree.

Blocking signals:
- `shard_plan_invariant=failed`
- `analysis:cross-repo-missing`

### 6.3 Execution profile semantics

Проверяем:
- consistency `strategy`, `max_parallel_tasks`, `failure_policy`, `shard_discovery_mode` по effective surfaces + shard artifacts + run results;
- headless provider command line включает не только `arch-workspace`, но и source repo directories для task repo scopes;
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
- preflight cancel-timeout guard соблюдён: `ACP_UI_CANCEL_POLL_TIMEOUT_SEC >= UI_E2E_CANCEL_STUB_SLEEP_SEC + UI_E2E_CANCEL_TIMEOUT_MARGIN_SEC`;
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

Дополнительно:
- если run завершился `run_partial_failed` и `reports/taskruns/<run_id>-quality.json.evidence_state.report_mode=incomplete`, generated markdown artifacts (`as-is/findings/coverage/proposals/agent-outputs`) читать только как triage-only artifacts; banner/triage-only wording обязаны явно указывать на incomplete analysis, а не имитировать пустой успешный verdict.

### 6.7 Triage rule for runtime timeout/infra signals

Если в любом `profile+sweep` появляются `runtime_timeout` или `infra_*`:
- считать это blocking runtime incident для релизного verdict;
- сначала проверить `driver.log` (matrix + batch), затем `session-summary.md` и `full-run.log` в `runs/<batch-id>/<provider>/runN/`, затем `arch-workspace/reports/taskruns/logs/*.ndjson` и `reports/taskruns/raw/*` для первичного runtime signal;
- отделить induced failures (например, debug timeout override) от реальной runtime/provider деградации;
- если raw/taskrun logs показывают `runner_parse_failed` или `runner_unavailable`, считать их primary failure class даже если `session-summary.md` дополнительно фиксирует `infra_incomplete_cycle`;
- для release decision использовать только прогон без diagnostic timeout overrides.

### 6.8 Additional Non-Release Checks

После official release matrices выполнить отдельно:
- parallel shard smoke: два параллельных `full-run-batch-5x2.sh` с разными `BATCH_ID` и `BATCH_PROVIDER_FILTER=qwen-code|claude-code`, `BATCH_RUN_SELECTION=1`, `BATCH_FRONTEND_MODE=never`;
- forced-incomplete diagnostic run: large multi-repo batch с diagnostic timeout override, чтобы проверить `report_mode=incomplete`, triage-only wording и failure-class precedence.

Эти прогоны не использовать для release verdict; они нужны только как additional evidence по новому функционалу.

Рекомендуемые команды (без wrapper):

```bash
# parallel smoke (example: Sentry multi-git_url)
TS="$(date -u +%Y%m%dT%H%M%SZ)"
(
  BATCH_ID=parallel-smoke-${TS}-qwen \
  TARGET_REPOS_FILE=examples/repos/github/multi-sentry-ecosystem.repos.yaml \
  BATCH_PROVIDER_FILTER=qwen-code \
  BATCH_RUN_SELECTION=1 \
  BATCH_FRONTEND_MODE=never \
  ./scripts/full-run-batch-5x2.sh
) &
(
  BATCH_ID=parallel-smoke-${TS}-claude \
  TARGET_REPOS_FILE=examples/repos/github/multi-sentry-ecosystem.repos.yaml \
  BATCH_PROVIDER_FILTER=claude-code \
  BATCH_RUN_SELECTION=1 \
  BATCH_FRONTEND_MODE=never \
  BATCH_SKIP_PRECHECK=1 \
  ./scripts/full-run-batch-5x2.sh
) &
wait || true

# forced-incomplete diagnostic (example: OpenStack multi-path)
# ВАЖНО: execution overrides задаются через ACP_EXECUTION_*, а не BATCH_*.
BATCH_ID=forced-incomplete-$(date -u +%Y%m%dT%H%M%SZ)-qwen \
TARGET_REPOS_FILE=examples/repos/curated/multi-path.openstack.repos.yaml \
BATCH_PROVIDER_FILTER=qwen-code \
BATCH_RUN_SELECTION=1 \
BATCH_FRONTEND_MODE=never \
BATCH_SKIP_PRECHECK=1 \
ACP_EXECUTION_STRATEGY=parallel \
ACP_MAX_PARALLEL_TASKS=4 \
ACP_FAILURE_POLICY=best_effort \
ACP_SHARD_DISCOVERY_MODE=heuristics \
ACP_RUNTIME_STEP_TIMEOUT_SEC=15 \
ACP_PIPELINE_TIMEOUT_SEC=300 \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-5x2.sh || true
```

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
9. Для каждого `profile_id` invariant `baseline == parallel-default` по shard-plan выполнен.

Любое нарушение => `RELEASE BLOCKED`.

## 8) Agent verdict output

Источник истины для финального решения:
- `release_verdict_<matrix-id>.md/.json`

## 9) Common blockers (операционный triage)

1. `repos[1] path does not exist: /tmp/provenarch-live-e2e/...`
- Причина: path profiles не подготовлены на pinned SHA.
- Действие: подготовить локальные checkout в точных absolute paths из curated файлов.

2. `SHA_MISMATCH: ... expected=<sha> got=<sha>`
- Причина: локальный clone существует, но checkout не закреплён на pinned release SHA.
- Действие: выполнить `git -C <repo> checkout --detach <expected_sha>` и повторить verify script.

3. `runtime_timeout` вместе с `runner_unavailable` в том же run
- Политика triage: primary incident class = `runtime_timeout` при явном timeout signal в summary/classifier; `runner_unavailable` остаётся secondary evidence.

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
