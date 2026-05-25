# Release Live E2E Runbook (Agent, no wrapper)

Этот runbook фиксирует manual pre-release gate на trusted локальной машине.
Новый wrapper-скрипт не используется: local `manual-live-e2e workflow` является operator procedure on trusted host, not GitHub Actions workflow. Агент запускает существующий matrix harness напрямую (`full-run-batch-matrix.sh` -> `full-run-batch.sh` -> internal backend-cycle helper -> `e2e_batch_report.py`).
Layering: live-e2e skill -> local manual-live-e2e workflow -> internal evaluator helper -> existing project flow. `scripts/internal/live-e2e-evaluator.sh` является source-only implementation detail для durable step evidence; он не является public release command и не вызывает matrix harness.

Canonical source of truth для live profile taxonomy:
- `examples/e2e-profile-catalog.yaml`
- runnable slice-файлы `examples/e2e-matrix.regres-*.yaml` и `examples/e2e-matrix.release-*.yaml`
- diagnostic selector slice-файлы `examples/e2e-matrix.smoke-tiny.bank.yaml` и `examples/e2e-matrix.diagnostic.sentry.yaml`

## Black-box evaluator protocol

Canonical live E2E flow теперь является operator-driven black-box evaluation. Агент не начинает с чтения итогового отчёта: он планирует шаг, запускает или инспектирует только публичную поверхность, фиксирует evidence, классифицирует результат и принимает следующее решение.

После каждой фазы в ответе оператора и в durable artifacts должен быть шаг:

```text
goal: <что доказываем>
action: <какую публичную поверхность вызвали/прочитали>
observed evidence: <команды, UI/API/log/report/artifact/verifier paths>
status: passed|failed|skipped|blocked
primary classification: none|operational_host_preflight_failed|precheck_failed|runtime_timeout|runner_unavailable|runtime_contract_failed|runtime_flow_failed|quality_gates_failed|release_verdict_FAIL|...
next decision: <continue|stop|rerun diagnostic|verify verdict|final report>
```

Разрешённые evidence surfaces:
- direct harness commands (`scripts/full-run-batch-matrix.sh`, `scripts/full-run-batch.sh`, `scripts/live-e2e-plan.py --format shell`);
- UI/API состояния;
- generated reports under `reports/*`;
- taskrun artifacts/logs/raw metadata;
- batch/matrix status, inventories and driver logs;
- verifier output from `scripts/verify-release-verdict.py`.

Запрещено использовать compatibility aliases, command shims или править canonical matrix/curated repo files под текущую машину. Host/provider/path blockers останавливают прогон как `operational_host_preflight_failed`.

Harness через internal evaluator helper пишет step evidence в existing report roots:
- `reports/blackbox_e2e_steps_<batch-id>.jsonl`
- `reports/blackbox_e2e_steps_<batch-id>.md`
- `reports/blackbox_e2e_steps_<matrix-id>.jsonl`
- `reports/blackbox_e2e_steps_<matrix-id>.md`

Canonical flow:
1. host/tree/provider/path preflight;
2. selector and direct command planning;
3. matrix execution monitoring;
4. backend artifact and quality inspection;
5. frontend UI/cancel inspection;
6. release verdict verification;
7. final black-box report.

## 0) Canonical profile catalog

Пять high-level профилей задаются как composite presets поверх одного или нескольких прямых вызовов `scripts/full-run-batch-matrix.sh`.

| Profile | Mode | Constituent matrix files | Target repo sets | Shard bucket | Backend runs | Rough time band |
|---|---|---|---|---|---:|---|
| `regres fast` | `regres` | `e2e-matrix.regres-fast.bank-openedx.yaml`, `e2e-matrix.regres-fast.openstack.yaml` | `bank-of-anthos`, `openedx`, `openstack` | `small` | 3 | `short-window` |
| `regres long` | `regres` | `e2e-matrix.regres-long.yaml` | `posthog`, `ftgo` | `medium` | 2 | `medium-window` |
| `release fast` | `release` | `e2e-matrix.release-fast.yaml` | `bank-of-anthos`, `openedx` | `small` | 12 | `short-window` |
| `release long` | `release` | `e2e-matrix.release-long.yaml` | `posthog`, `openstack` | `medium` | 12 | `medium-window` |
| `release full` | `release` | `e2e-matrix.release-fast.yaml`, `e2e-matrix.release-long.yaml`, `e2e-matrix.release-full.ftgo-sentry.yaml` | все 6 canonical repo sets | `full` | 36 | `extended-window` |

Sizing policy по текущему planner:
- `small <= 16 shards`
- `medium = 17..64 shards`
- `full >= 65 shards`

Current shard classification:
- `small`: `bank-of-anthos=6`, `openedx=5`, `openstack=5`
- `medium`: `posthog=24`, `ftgo=37`
- `full`: `sentry-ecosystem=104`

Release verdict для readiness берётся только из `reports/release_verdict_<matrix-id>.json`.
Для `release full` composite readiness означает, что все constituent `release_verdict_<matrix-id>.json` имеют `PASS`.

### 0.1) Flexible command generator (no wrapper)

Для быстрых diagnostic/non-release комбинаций можно сгенерировать прямые команды harness:

```bash
python3 scripts/live-e2e-plan.py --mode smoke --size tiny --providers codex --format shell
python3 scripts/live-e2e-plan.py --mode regres --size fast --providers codex --format shell
python3 scripts/live-e2e-plan.py --mode regres --size full --providers claude --frontend-mode never --format shell
python3 scripts/live-e2e-plan.py --mode release --size full --format shell
```

`scripts/live-e2e-plan.py` только печатает команды `scripts/full-run-batch-matrix.sh`; он не запускает batch и не является release wrapper.
Оператор копирует/запускает напечатанные прямые команды, поэтому official release verdict contract не меняется.
Если selector задаёт `--frontend-mode never`, generator выставляет и `BATCH_FRONTEND_MODE=never`, и `BATCH_FRONTEND_CANCEL_MODE=never`, чтобы не запускать ни init, ни cancel frontend smoke.
В non-release diagnostic verdict такие frontend statuses считаются non-applicable; в release-mode strict frontend `passed` для всех release providers остаётся обязательным.

Flexible selectors:

| Selector | Mode | Coverage | Providers | Sweeps | Backend runs |
|---|---|---|---|---|---:|
| `smoke tiny` | diagnostic | `bank-of-anthos` | exactly 1 selected provider | implicit baseline | 1 |
| `regres fast` | diagnostic/regression | `bank-of-anthos`, `openedx`, `openstack` | selected provider(s) | implicit baseline | `3 × providers × RUN_COUNT` |
| `regres long` | diagnostic/regression | `posthog`, `ftgo` | selected provider(s) | implicit baseline | `2 × providers × RUN_COUNT` |
| `regres full` | diagnostic/regression | all 6 canonical repo sets, including Sentry | selected provider(s) | implicit baseline | `6 × providers × RUN_COUNT` |
| `release fast|long|full` | release | canonical release slices | all three providers only | `baseline + parallel-default` | unchanged |

Artifact-quality policy для generated regress/release команд остаётся штатной:
- каждый backend run должен иметь `reports/taskruns/<run_id>-quality.json`;
- `quality_report_<batch-id>.md` должен агрегировать только реально выбранные providers/run indexes;
- `artifact_quality:*` warning поднимается в `quality_gates_failed` и блокирует strict verdict.
- `totals.repair_attempts`, `fresh_retries`, `focused_repairs`, `repair_exhausted`, `stall_count`, `pre_artifact_stalls`, `post_artifact_stalls`, `zero_output_pre_artifact_stalls` и `partial_failure_count` являются обязательной visible telemetry в quality/matrix reports;
- non-exhausted repair/stall pressure не превращает successful backend run в failure само по себе, но `partial_failure_count > 0` остаётся strict blocker.

## 1) Scope и ограничения

- Canonical `regres*` профили по умолчанию идут как `qwen-only` non-release baseline с implicit `baseline`; flexible diagnostic selectors могут явно выбрать provider set через generator/`BATCH_PROVIDER_FILTER`.
- `release*` профили идут как three-provider run (`qwen + claude + codex`) с explicit `baseline + parallel-default`.
- Дополнительная ручная debug-фаза на `claude` или `codex` остаётся вне regression profile definition и запускается через `BATCH_PROVIDER_FILTER=<provider>`.
- `smoke tiny` и `regres full` являются generated diagnostic selectors, а не новыми release verdict профилями.
- Gate покрывает backend `3 providers × RUN_COUNT=1` на каждый `profile+sweep` + frontend `init-inspect` + frontend `cancel-refresh` для release slices.
- Product API/schema contracts не меняются.
- Gate режим: manual pre-release, не required CI merge gate.
- Verdict policy: strict zero-failure (`PASS|FAIL`).
- В release-mode matrix harness включает timeout safety guard: диагностические timeout override запрещены по умолчанию.

## 2) Prerequisites

Проверить на машине:
- Go exact version from `.go-version`
- Node.js exact version from `.node-version` and npm from the same toolchain
- `python3`
- `curl`
- `qwen`
- `claude`
- `codex`
- доступ к `path` repos
- доступ к `git_url` repos с pinned `ref`

### 2.0) Clean tree requirement

Canonical profile runs нужно выполнять из clean committed tree или из отдельного clean worktree без unrelated локального drift.

Практическое правило:
- если в основном worktree есть незакоммиченные изменения в harness/runtime/docs, сначала вынести canonical прогон в отдельный clean worktree;
- если используется отдельный clean worktree, заранее установить локальные UI deps в этом worktree через exact Node resolver: минимум `./scripts/run-npm.sh ci --prefix ui`; для frontend live surface дополнительно `./scripts/run-npm.sh exec --prefix ui playwright install chromium`;
- не использовать `BATCH_SKIP_PRECHECK=1` как способ обойти локальное расхождение между committed contract и текущими незакоммиченными правками;
- diagnostic прогон с `BATCH_SKIP_PRECHECK=1` допустим только как triage-only evidence и не считается canonical acceptance run.

### 2.0.1) Exact Node/npm toolchain

Live harness выполняет DoD/UI precheck только на exact Node.js из `.node-version`. Совместимая minor версия не считается валидной: например, если `.node-version` требует `22.21.1`, то Homebrew `node@22` с `22.22.3` остаётся `precheck_failed`.

Проверить выбранный toolchain до запуска:

```bash
./scripts/resolve-node-tool.sh node
./scripts/resolve-node-tool.sh npm
```

Если exact версия установлена не первой в `PATH`, передать её явно:

```bash
ACP_NODE_TOOL_CANDIDATES=/path/to/node-22.21.1/bin ./scripts/full-run-batch-matrix.sh
```

Для release evidence использовать стабильный toolchain path вне `/tmp`: временные распаковки Node могут исчезнуть между diagnostic и release runs и превратить backend/frontend checks в `quality_gates_failed`/`precheck_failed` без продуктового ACP дефекта.

Не использовать `ACP_NODE_VERSION_CHECK=0` или `BATCH_SKIP_PRECHECK=1` для canonical acceptance. Эти обходы допустимы только как triage-only evidence.

### 2.1) Fail-fast host eligibility

Эту проверку выполнять до DoD/preflight и до старта canonical release slices:

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
- текущий хост не подходит для canonical `release fast|long|full` path slices под `/tmp/provenarch-live-e2e`;
- canonical release profiles здесь запускать нельзя;
- нужно перенести прогон на другой trusted host, а не переписывать matrix/curated files под локальную машину.

Дополнительный runtime host preflight для `regres/release` matrix:

```bash
export PATH=/opt/homebrew/bin:$PATH
qwen --version
python3 - <<'PY'
from pathlib import Path
for path in [
    Path("/tmp/provenarch-test_arch_project"),
    Path("/tmp/provenarch-test_arch_project/reports"),
    Path("/tmp/provenarch-test_arch_project/matrix"),
]:
    path.mkdir(parents=True, exist_ok=True)
    probe = path / ".write-probe"
    probe.write_text("ok", encoding="utf-8")
    probe.unlink()
    print(f"OK writable: {path}")
PY
```

Если любой шаг выше падает, фиксировать как `operational_host_preflight_failed` и не интерпретировать как продуктовый ACP дефект.

Matrix preflight также выполняет selected-provider live smoke перед deep batch:
- `--version` probe;
- короткий headless `ACP_READY` probe для providers, где он остаётся устойчивым readiness signal;
- artifact smoke: выбранный provider должен записать sentinel-файл в temp write dir. Для `claude` artifact smoke является основным headless readiness gate, temp write dir передаётся через runtime-like `--add-dir`, потому что отдельный text-only probe может флейкать по latency без проверки filesystem write path; первый timeout/no-output artifact-smoke attempt допускает один bounded retry.
- Для `claude` artifact smoke timeout после записи expected sentinel считается ready: это доказывает headless response + filesystem write path, а runtime artifact-only engine умеет controlled stop after valid artifacts. Timeout без expected sentinel остаётся blocker и допускает только bounded retry.

Artifact smoke failure или timeout считается `operational_host_preflight_failed` и должен останавливать batch/matrix до запуска дорогой runtime matrix. Это host/provider readiness blocker, а не ACP product verdict.

### 2.2) One-time canonical path bootstrap

Если хост подходит по `2.1`, но curated `path` checkout'ы ещё не существуют, их нужно подготовить один раз заранее.

Минимальное требование:
- каждый `path`, используемый canonical release slices, должен существовать локально;
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
  /tmp/provenarch-live-e2e/openstack

# release-long single-path
git clone https://github.com/posthog/posthog.git /tmp/provenarch-live-e2e/posthog/posthog
git -C /tmp/provenarch-live-e2e/posthog/posthog checkout --detach 14d29a548d63665d60b506cf13bd5cfb2de7c743

# release-fast multi-path (Open edX)
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

# release-fast/release-long multi-path (OpenStack)
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

После bootstrap обязательно прогнать path/SHA verify script из `4) Порядок запуска` перед canonical release slices.

Обязательное условие для path-профилей:
- локальные checkout должны быть на тех же pinned SHA, что и соответствующие `github` presets;
- canonical пути из `examples/repos/curated/*.repos.yaml` должны реально существовать на trusted машине;
- если `/tmp/provenarch-live-e2e` не может быть создан или системно очищается слишком агрессивно, использовать persistent mount/volume под этим canonical root.
- не переписывать canonical `examples/e2e-matrix.release-*.yaml` или `examples/repos/curated/*.repos.yaml` только ради обхода ограничений текущей машины.

## 3) Matrix input contract

`E2E_MATRIX_FILE` содержит:
- `profiles[]`:
  - `single-path`
  - `single-git_url`
  - `multi-path`
  - `multi-git_url`
  - release-mode требует ровно 2 профиля: один `single-*` и один `multi-*`
- `sweeps[]` (optional):
  - если отсутствует -> implicit `baseline`
  - release-ready harness использует 2 sweep-профиля:
    - `baseline`: `strategy=sequential`, `max_parallel_tasks=1`, `failure_policy=best_effort`, `shard_discovery_mode=heuristics`
    - `parallel-default`: `strategy=parallel`, `max_parallel_tasks=4`, `failure_policy=best_effort`, `shard_discovery_mode=heuristics`
  - release-mode фиксирует `RUN_COUNT=1`

Готовый шаблон:
- `examples/e2e-matrix.example.yaml`
- `examples/e2e-profile-catalog.yaml`
- `examples/e2e-matrix.smoke-tiny.bank.yaml`
- `examples/e2e-matrix.regres-fast.bank-openedx.yaml`
- `examples/e2e-matrix.regres-fast.openstack.yaml`
- `examples/e2e-matrix.regres-long.yaml`
- `examples/e2e-matrix.diagnostic.sentry.yaml`
- `examples/e2e-matrix.release-fast.yaml`
- `examples/e2e-matrix.release-long.yaml`
- `examples/e2e-matrix.release-full.ftgo-sentry.yaml`
- `examples/repos/*.repos.yaml`

### 3.1) Canonical repo-set catalog

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
  - id: multi-git_url
    source_kind: git_url
    expected_repo_count: 5
    repos_file: ./repos/github/multi-sentry-ecosystem.repos.yaml
```

Важно:
- для `git_url` использовать только pinned `ref` (commit SHA);
- для `path` использовать локальные checkout тех же репозиториев на тех же SHA;
- перед релизным прогоном при необходимости обновить SHA в preset-файлах отдельным коммитом (без изменения harness логики).
- catalog `examples/e2e-profile-catalog.yaml` фиксирует shard bucket, expected backend runs и runnable matrix files для каждого named profile.

## 4) Порядок запуска

0. Super-fast smoke через generated direct command (`1 repo × 1 run × 1 provider`):

```bash
python3 scripts/live-e2e-plan.py --mode smoke --size tiny --providers qwen --format shell
```

Проверить напечатанную команду и запустить её напрямую. Этот smoke не является release readiness signal.

1. `regres fast` (qwen-first small repos, 3 backend runs total):

```bash
E2E_MATRIX_FILE=examples/e2e-matrix.regres-fast.bank-openedx.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
BATCH_PROVIDER_FILTER=qwen-code \
./scripts/full-run-batch-matrix.sh

E2E_MATRIX_FILE=examples/e2e-matrix.regres-fast.openstack.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
BATCH_PROVIDER_FILTER=qwen-code \
./scripts/full-run-batch-matrix.sh
```

Обе canonical matrix в этом профиле уже несут `timeout_profile=short-window`; штатный запуск не требует внешних `ACP_*TIMEOUT*`.

2. `regres long` (qwen-first medium repos, 2 backend runs total):

```bash
E2E_MATRIX_FILE=examples/e2e-matrix.regres-long.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
BATCH_PROVIDER_FILTER=qwen-code \
./scripts/full-run-batch-matrix.sh
```

Этот slice несёт `timeout_profile=medium-window`.

3. Manual Codex diagnostic для regression slice (не часть canonical regres verdict, только targeted smoke):

```bash
E2E_MATRIX_FILE=examples/e2e-matrix.regres-long.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
BATCH_PROVIDER_FILTER=codex-code \
BATCH_SKIP_PRECHECK=1 \
./scripts/full-run-batch-matrix.sh
```

Если после regression нужна дополнительная debug-фаза на `claude`, повторить нужный regression slice с:

```bash
BATCH_PROVIDER_FILTER=claude-code \
BATCH_SKIP_PRECHECK=1
```

Альтернативно для provider/size combinations использовать generator:

```bash
python3 scripts/live-e2e-plan.py --mode regres --size fast --providers codex --format shell
python3 scripts/live-e2e-plan.py --mode regres --size full --providers claude --frontend-mode never --format shell
```

`regres full` добавляет diagnostic Sentry baseline slice и покрывает все 6 canonical repo sets, но остаётся non-release.

3. Preflight ACP quality для release slices:

```bash
make contracts test lint build
./scripts/run-npm.sh ci --prefix ui
./scripts/run-npm.sh exec --prefix ui playwright install chromium
```

4. Проверка path targets (exist + pinned SHA):

```bash
python3 - <<'PY'
import json, subprocess, sys
from pathlib import Path

files = [
    "examples/repos/curated/single-path.posthog.repos.yaml",
    "examples/repos/curated/multi-path.openedx.repos.yaml",
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

5. `release fast` (small repos, 12 backend runs total):

```bash
MATRIX_ID=release-fast-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=examples/e2e-matrix.release-fast.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh
```

`examples/e2e-matrix.release-fast.yaml` несёт `timeout_profile=short-window`.

6. `release long` (medium repos, 12 backend runs total):

```bash
MATRIX_ID=release-long-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=examples/e2e-matrix.release-long.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh
```

`examples/e2e-matrix.release-long.yaml` несёт `timeout_profile=medium-window`.

7. `release full` (all canonical repo sets, 36 backend runs total):

```bash
MATRIX_ID=release-full-fast-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=examples/e2e-matrix.release-fast.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh

MATRIX_ID=release-full-long-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=examples/e2e-matrix.release-long.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh

MATRIX_ID=release-full-ftgo-sentry-$(date -u +%Y%m%dT%H%M%SZ) \
E2E_MATRIX_FILE=examples/e2e-matrix.release-full.ftgo-sentry.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch-matrix.sh
```

Addon slice `examples/e2e-matrix.release-full.ftgo-sentry.yaml` несёт `timeout_profile=extended-window`.

Composite readiness rule:
- `release full` считается готовым только если все три constituent `release_verdict_<matrix-id>.json` имеют `PASS`.
- `release fast` и `release long` можно использовать как самостоятельные slice verdicts.

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
- Native `timeout_profile` внутри checked-in canonical matrix не считается diagnostic override и является частью штатного release/regression surface.

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
- `blackbox_e2e_steps_<batch-id>.jsonl/.md`
- `blackbox_e2e_steps_<matrix-id>.jsonl/.md`

Pre-tag/offline check:
```bash
python3 scripts/verify-release-verdict.py reports/release_verdict_<matrix-id>.json
```

Скрипт только проверяет уже созданный `release_verdict_<matrix-id>.json` (`verdict=PASS`, `release_state=RELEASE READY`, `release_contract.contract_status=passed`). Он не запускает live harness и не является wrapper-скриптом поверх `scripts/full-run-batch-matrix.sh`.

Дополнительные для triage:
- `/tmp/provenarch-test_arch_project/runs/<batch-id>/<provider>/runN/*`
- `backend-run-classifications.tsv`
- `preflight.json`
- `driver.log`, `full-run.log`, `session-summary.md`

## 6) Flow проверки новой функциональности

### 6.1 Runtime sharding

Проверяем:
- shard artifacts для `init`/`refresh`: `shard-plan`, `shard-summary`, per-shard `runtime-execution.json`;
- shard-summary status progression: `pending -> checkpointed -> succeeded` и `failed` path при runtime/apply ошибке;
- runtime execution metadata: `shard_id`, `repo_scopes`, `path_scopes`.

Blocking signals:
- `runtime:shard-artifacts`
- `runtime:shard-metadata`

### 6.2 Cross-repo coverage hardening

Проверяем:
- для multi-profile (`expected_repo_count >= 2`) нет `analysis:cross-repo-missing`;
- evidence и authored artifacts действительно ссылаются на несколько repo scopes, а не схлопываются в single-repo narrative;
- cross-repo signal считается присутствующим, если есть explicit `semantic.edges[]`, либо repo coverage через `citations[].repo` плюс finding provenance по нескольким repos, либо question `related_ids` по нескольким repo scopes при наличии repo-specific citations; старый edge-only report check слишком узок для focused recovery outputs;
- `path` и `git_url` profiles сохраняют одинаковую strict semantics по shard-plan/runtime-flow checks.

Blocking signals:
- `analysis:cross-repo-missing`

### 6.3 Execution profile semantics

Проверяем:
- consistency `strategy`, `max_parallel_tasks`, `failure_policy`, `shard_discovery_mode` по effective surfaces + shard artifacts + run results;
- headless provider command line включает не только `arch-workspace`, но и source repo directories для selected repo scopes;
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
- frontend `init-inspect` = `passed` для всех трёх release providers (`qwen-code`, `claude-code`, `codex-code`);
- frontend `cancel-refresh` = `passed` для всех трёх release providers (`qwen-code`, `claude-code`, `codex-code`);
- frontend cancel smoke стартует из свежей копии `arch-workspace` выбранного backend run; reuse уже мутированного `frontend-workspace` из init-smoke не допускается;
- preflight cancel-timeout guard соблюдён: `ACP_UI_CANCEL_POLL_TIMEOUT_SEC >= UI_E2E_CANCEL_STUB_SLEEP_SEC + UI_E2E_CANCEL_TIMEOUT_MARGIN_SEC`;
- cancel сценарий завершается `failed + run_canceled`;
- если cancel request пришёл раньше конкурирующего layout/validation failure, terminal `error_code` остаётся `run_canceled`, а competing validation signal уходит только в logs/warnings.

Blocking signals:
- любой frontend status != `passed`

### 6.6 Runtime parsing/diagnostics stability

Zero tolerance:
- `runtime_contract_failed`
- `runner_unavailable`
- `summary_missing`
- `infra_signal_terminated`
- `infra_incomplete_cycle`
- `quality_gates_failed`
- `precheck_failed`

Дополнительно:
- если run завершился `run_partial_failed` и `reports/taskruns/<run_id>-quality.json.evidence_state.report_mode=incomplete`, generated markdown artifacts (`as-is/findings/coverage/proposals/agent-outputs`) читать только как triage-only artifacts; banner/triage-only wording обязаны явно указывать на incomplete analysis, а не имитировать пустой успешный verdict.
- для `init.step1.collect` и `refresh.step1.collect` collect prompt начинает task-specific surface с `FIRST COLLECT ARTIFACT PAIR COMMAND` сразу после provider identity: suggested authored doc path + literal task-specific manifest skeleton должны быть записаны как focused artifact pair до broad second-pass repository sweep. До этой команды provider не должен вызывать `read_file`, `list_directory`, `grep_search`, `glob` или repo exploration; repo entrypoint hints в collect prompt применяются только после того, как first artifact pair уже существует. Generic manifest examples не добавляются в collect prompts; task-specific skeleton является единственным JSON template, а поздний docs-first block не дублирует heredoc command. Общий runtime может сделать одну collect pair-recovery попытку, если provider оставил stdout/stderr diagnostics, но не записал authored artifacts, и одну manifest-only repair попытку, если provider уже записал authored docs, но `write_root/shard-pack-manifest.json` missing/invalid. Pair recovery пишет только suggested doc + manifest; manifest-only repair пишет только `shard-pack-manifest.json` и начинается с `FIRST COLLECT MANIFEST REPAIR COMMAND` plus absolute heredoc write target вокруг exact skeleton. Repair prompts instruct overwrite for invalid manifests rather than read/diff/patch, не повторяют validation-error field cues, а engine write-set guards запрещают лишние writes.
- все required artifacts должны писаться/проверяться по exact absolute `write_root`/`draft_final_root`; relative CWD checks вроде `test -f validator-verdict.json` или `test -f nova-overview.md` не считаются валидным runtime artifact target.
- `init|refresh.step3.findings` начинает normal prompt с `FIRST VALIDATOR VERDICT COMMAND` сразу после provider identity: provider пишет минимальный валидный `write_root/validator-verdict.json` skeleton до broad validation instructions. Для multi-repo validator tasks этот first-action skeleton уже содержит PASS-compatible cross-repo finding и question с `related_ids` плюс repo/path evidence, чтобы первый валидный artifact не конфликтовал с `analysis:cross-repo-missing`; `issues[]` остаётся только для technical validator problems. Шаг может выполнить одну focused validator-verdict repair попытку при missing/invalid verdict; repair prompt использует тот же command-first absolute heredoc skeleton, `checked_paths` указывает на staged final artifacts, а не validator `write_root`; `issues[]` использует canonical `code/severity/message` shape, legacy finding-shaped issue fields остаются `runtime_contract_failed`.
- `init.step0.constitution` начинает normal prompt с `FIRST CONSTITUTION DRAFT COMMAND` сразу после provider identity: provider пишет `write_root/constitution-draft.json` и referenced draft files под `draft_final_root` до broad workspace analysis. `baseline-subagents.yaml` в этом first-action set должен быть валидным `skills/subagents.yaml` YAML bundle (`agents:`), иначе следующий refresh обязан остановиться на workspace validation.
- draft steps (`step0/2/4`) могут выполнить одну focused draft-artifact repair попытку: provider пишет только step manifest в `write_root` и referenced files под `draft_final_root`; repair prompt начинается с command-first heredoc artifact set для manifest + draft files, задаёт `write_root`/`draft_root` из exact absolute paths один раз и дальше пишет через `"$write_root/..."` / `"$draft_root/..."`, выполняет `test -s` checks для manifest и каждого referenced draft file, запрещает broad analysis и ручное перепечатывание/переписывание path components до первого валидного draft set; ACP-side manifest/verdict/draft autofill запрещён.
- `init|refresh.step2.asis_docs` normal и repair prompts начинают as-is surface с единственного `FIRST AS-IS DRAFT COMMAND`: команда пишет `write_root/asis-draft-manifest.json` и три required draft files под `draft_final_root` (`overview.md`, `summary.md`, `architect-summary.md`) до broad as-is analysis; manifest success без реально существующих referenced files остаётся `runtime_contract_failed`.
- collect repair intentionally runs with narrow read scope: current `write_root` + repo evidence roots only. Broader ACP workspace, sibling `reports/taskruns`, raw logs, archive docs and old shard manifests are excluded; embedded prompt contract/schema text is authoritative when runtime workspace does not contain `schemas/*` or `docs/spec/*`.
- root-file collect shards должны читать только перечисленные root-level files, писать один concise root overview doc и сразу завершать `shard-pack-manifest.json`, без recursive sweep по top-level directories.
- root-marker-only repos (`pyproject.toml`, `pom.xml`, etc. только в корне) не должны превращаться в один `"."` shard на больших repo: expected plan shape — root-file group + top-level directory shards с дальнейшим cap/coalescing.
- coalesced shard IDs должны оставаться filesystem-safe bounded names; большие root-file groups получают стабильный shortened slug + hash, чтобы `reports/taskruns/.../staging/shards/<shard_id>` не падал на `file name too long`.
- `shard-pack-manifest.json.documents[].path` должен быть строго relative к `artifact_root` и указывать на реально существующий provider-authored file under `write_root`; workspace-level staging prefixes (`reports/...`, `charter/...`, `proposals/...`), duplicated `artifact_root` prefix, missing file references и directory references считаются contract-invalid collect drift и не должны доходить до `step2`.
- collect manifest допускает только canonical vocabulary: `semantic.coverage.observed`, `semantic.questions[*].id/text`, `semantic.findings[*].id/severity/title/provenance`, `semantic.edges[*].type`, object-shaped `provenance`, numeric `confidence`; aliases вроде `covered_topics`, `question`, `relation`, array provenance, string confidence, `evidence_citation_ids`, top-level `step_contract` и `compatibility` считаются hard contract drift.
- для `step1.collect` runtime не должен использовать `reports/taskruns/**`, raw logs, архивные планы и старые `shard-pack-manifest.json` как schema/reference surface; headless provider опирается на embedded schema/spec/prompt contract, selected repo roots, `write_root` и explicit `read_context_roots`.
- `init.step0.constitution`, `init|refresh.step2.asis_docs` и `init|refresh.step4.proposals` считаются successful только если runtime draft manifest валиден и все referenced draft files реально существуют под `draft_final_root`.
- `init|refresh.step2.asis_docs` использует strict shared draft contract: `step_contract="as_is"`, required outputs `reports/as-is/overview.md`, `reports/coverage/summary.md`, `reports/agent-outputs/architect/summary.md`; дополнительные outputs допустимы только под `reports/as-is/<domain>/overview.md`, а legacy top-level fields вроде `repo_scopes` или `compatibility` должны hard-fail-иться.
- `init|refresh.step4.proposals` использует strict shared draft contract: `step_contract="proposals"`, required top-level shape `version=1/run_id/step_id/step_contract/agent_role/summary?/outputs[]`; `outputs[].canonical_path` допустим только под `proposals/*` или `reports/changelog/*`, duplicate canonical paths запрещены, а legacy fields `pipeline`, `step`, `generated_at`, `domain_id`, `proposals[]`, `info_findings_noted`, `orphan_coverage_gaps` должны hard-fail-иться как `runtime_contract_failed`.
- non-collect runtime шаги не должны стартовать из workspace root: draft steps используют `draft_final_root` как cwd, validator использует `write_root`, а backend-cycle helper под `scripts/full-run-batch.sh` разводит headless и baseline workspaces по разным temp roots, чтобы sibling baseline artifacts не были implicit template source.
- provider-side hard sandbox в текущих headless CLI нет; поэтому runtime isolation надо оценивать через temp-root layout и step-local `cwd`, а не ожидать отдельного sandbox enforcement от provider tooling.
- `claude-code`, `qwen-code` и `codex-code` используют общий artifact-only process engine: stdout/stderr capture, process-group termination, timeout/cancel handling, raw diagnostics, activity monitor и artifact validation находятся в shared `providercommon` path.
- pre/post-artifact recovery provider-agnostic: до появления artifacts silent/no-artifact hangs bounded для всех live adapters; после появления required artifacts runtime обычно ждёт stale pipe activity и stale mutations `write_root`/`draft_final_root`; если artifacts уже валидны, controlled stop считается successful artifact-only completion, а не provider failure. Partial artifacts могут ждать отдельное более длинное provider policy grace window.
- provider-specific recovery задаётся adapter policy: `qwen-code` adapter invocation передаёт artifact prompt только через CLI `-p` без JSON task stdin, нормализует custom qwen prompt args к artifact prompt и использует `stream-json` activity output. `qwen-code` и `claude-code` разрешают один fresh retry для missing/invalid artifacts; `qwen-code` дополнительно делает первый zero-output pre-artifact stall retryable warning и один bounded focused collect-pair/draft-artifact retry, если repair transcript содержит transient provider transport/API failure (`[API Error: Premature close]`, `Connection error` with network socket/TLS disconnect, connection reset/closed, transient 5xx/529 stream errors) без artifacts. Для qwen normal draft steps (`step0/2/4`) valid manifest + referenced draft files могут быть accepted через bounded valid-artifact controlled stop, если provider продолжает активно стримить или мутировать draft files после валидного set; focused repair attempts всех providers также имеют short valid-artifact stop window после валидного repair artifact и принимаются только через validation gate. `claude-code` сохраняет zero-output fail-fast для non-scoped шагов, но на `init.step0.constitution`, `init|refresh.step1.collect`, `init|refresh.step3.findings` и `init|refresh.step4.proposals` первый fully silent pre-artifact stall является retryable warning; exhausted silent/API no-artifact retry остаётся `runner_unavailable`. Partial authored artifacts без валидного manifest, malformed validator verdict и malformed manifest/draft contract остаются `runtime_contract_failed` после focused repair exhaustion.
- если после разрешённых collect recovery попыток `shard-pack-manifest.json` всё ещё missing/invalid, collect step обязан завершиться runtime contract failure (`runtime_contract_failed`), а не продолжать прогон как nominal success.
- transcript outputs с provider transport/API errors (например `[API Error: ... SSL ...]` или `[API Error: Premature close]`) считаются `runner_unavailable`, а не `runtime_contract_failed`, если после разрешённого bounded retry нет валидных artifacts; raw stdout/stderr artifacts в `reports/taskruns/raw/*` обязательны.
- collect contract требует полного `semantic` block и global uniqueness для `citations[].claim_ids`; staged duplicate claim ids считаются blocking contract drift, если validator-scope repair не смог детерминированно снять коллизию на index/reference surface.
- staged `citation-index.json` и `final-run-index.json` должны использовать один deterministic `document_id` namespace, наследующий `manifest.Documents[*].id`; semantic assembly перед validator обязана нормализовать `evidence.repo` к логическому repo scope и дедуплицировать alias entities/related refs.
- если collect evidence = `unusable`, `init|refresh.step2.asis_docs`, `init|refresh.step3.findings` и `init|refresh.step4.proposals` не должны запускать live provider: staged final set собирается только из persisted collect artifacts и дальше остаётся triage-only incomplete surface.
- owner-gap остаётся visible signal в `coverage/findings/questions`, но owner-only residual без технических validator issues не должен сам по себе блокировать verdict; такой кейс допустимо увидеть как `validator-verdict = PASS` с сохранёнными findings/questions.
- `reports/taskruns/<run_id>-quality.json.run_warnings` с префиксом `artifact_quality:` считаются canonical live gate blocker даже при schema-valid `validator-verdict.json = PASS`; типовой пример — refresh final set с несколькими canonical docs и единственным generic `cite.runtime-summary`.
- acceptable reuse-pattern допускается только если frozen refresh artifacts сохраняют хотя бы один rich collect shard с repo-specific citations; reuse-only manifests без такого shard'а считаются low-signal collapse.
- `profile_matrix_<matrix-id>` и `quality_report_<batch-id>` агрегируют только реально выбранные `selected_providers` и `selected_run_indexes`; qwen-only `run1` regression run не должен материализовать synthetic `2x5` deficits.
- internal shard-plan/shard-summary JSON обязаны содержать non-empty `meta.runtime.name` / `meta.runtime.version`; пустой runtime meta считается contract drift, а не допустимым partial state.
- structural shard coalescing для больших repos сохраняет module marker leaf shard groups внутри top-level dirs, пока итоговый shard count остаётся в `maxAutoShardsPerRepo`; если top-level groups не помещаются в cap, они детерминированно merge-ятся в bounded buckets и получают warning.
- live triage от `2026-04-17` зафиксировал один надёжный blocker для canonical `regres fast`: `single-git_url` на `qwen-code` завершился runtime contract failure после event-stream chatter и неполного artifact-only collect recovery; последующий `multi-path`/Open edX run был прерван вручную и не считается самостоятельным продуктовым failure signal.
- subsequent clean rerun от `2026-04-17` подтвердил, что после фикса qwen prompt/retry + `cwd/chat-recording` этот parse blocker снимается; оставшийся canonical blocker сместился в устаревший `pipeline_timeout=2400s`, поэтому canonical matrix slices получили checked-in `timeout_profile` с matrix-native budget.

### 6.7 Triage rule for runtime timeout/infra signals

Если в любом `profile+sweep` появляются `runtime_timeout` или `infra_*`:
- считать это blocking runtime incident для релизного verdict;
- сначала проверить `driver.log` (matrix + batch), затем `session-summary.md` и `full-run.log` в `runs/<batch-id>/<provider>/runN/`, затем `arch-workspace/reports/taskruns/logs/*.ndjson` и `reports/taskruns/raw/*` для первичного runtime signal;
- отделить induced failures (например, debug timeout override) от реальной runtime/provider деградации;
- если raw/taskrun logs показывают `runtime_contract_failed` или `runner_unavailable`, считать их primary failure class даже если `session-summary.md` дополнительно фиксирует `infra_incomplete_cycle`;
- если `session-summary.md`/taskrun artifacts уже дают explicit `runtime_contract_failed`, batch/report layer не должен переопределять это в `runner_unavailable` только из-за grep/signature capacity/429 маркеров в тех же логах;
- raw provider stdout/stderr не должен создавать secondary `runner_unavailable` только из-за текста `runner_unavailable`; raw scan учитывает только реальные availability/capacity/rate-limit сигналы после noise filtering.
- если terminal logs показывают `read shard document ... no such file or directory` или collect validation фиксирует missing `documents[].path` reference, primary failure class должен быть `runtime_contract_failed`, даже если stale classifier row или raw provider diagnostics содержат `runner_unavailable` markers;
- если terminal logs показывают `parse runtime draft manifest` вместе с `unknown field`, primary failure class для batch/reporting должен быть `runtime_contract_failed` даже при одновременных `runner_unavailable` capacity/429 маркерах;
- если terminal logs показывают `validator verdict is FAIL`, primary failure class для batch/reporting должен быть `runtime_flow_failed`, но только когда run не классифицирован как terminal runtime/provider failure (`runtime_timeout`, `runner_unavailable`, `runtime_contract_failed`);
- если `session-summary.md` фиксирует terminal success (`result=passed`, `quality_gates=passed`, API `succeeded`) и `run-status.env state=completed process_exit=0`, shell/Python classifiers не должны поднимать `runner_unavailable`/`runtime_contract_failed` только из-за raw provider diagnostics from recovered attempts;
- если `session-summary.md` фиксирует terminal `failure_reason=quality` или `quality_gates=failed`, shell/Python classifiers должны считать это `quality_gates_failed` и игнорировать stale `runner_unavailable` rows/noise от более ранних raw provider attempts;
- для terminal runtime/provider failures (`runtime_timeout`, `runner_unavailable`, `runtime_contract_failed`) не эскалировать secondary `runtime_flow_failed`/`analysis:cross-repo-missing` только из-за неполных refresh/runtime artifacts;
- per-run evidence в batch report должен явно показывать `collect_partial_shard_failures`, focused recovery exhaustion/write-set violations и наличие runtime logs/metadata, если `run-results.tsv` не содержит ожидаемые headless rows;
- backend-cycle helper под `scripts/full-run-batch.sh` должен писать failed headless `init`/`refresh` rows в существующем 17-field `run-results.tsv` формате, когда CLI уже вернул `run_id`, и делать best-effort snapshot даже при missing/invalid quality summary; runtime artifacts/logs после terminal provider/runtime failure не должны выглядеть как missing-row infra gap;
- terminal `session-summary.md` вместе с `run-status.env state=process_failed summary_written=yes` считать завершившимся deterministic pipeline failure; такой run не должен переопределяться в `infra_incomplete_cycle` только из-за mismatch `completed_*`, неполного `run-results.tsv` или classifier fallback.
- backend-cycle helper обязан поддерживать running-heartbeat в `run-status.env` (`updated_at`, `last_pipeline_stage`, `last_runtime_provider`, `last_progress_at`) и сам писать terminal sentinel при `completed|process_failed|signal_terminated`.
- если `session-summary.md` отсутствует, но batch shell успел дойти до classifier или завершился через `EXIT` trap, trusted harness обязан materialize-ить `infra_incomplete_cycle` или `infra_signal_terminated` через per-run `run-status.env`; отсутствие summary больше не считается допустимым silent gap.
- если child batch завершился, а `run-status.env` отсутствует или остаётся `state=running`, outer batch обязан синтезировать terminal `process_failed` с `failure_reason=infra_incomplete_cycle`; `profile-status/*.json` должны отражать тот же terminal reason, а не generic `child_failed`.
- child `full-run-batch.sh` обязан публиковать `batch-owner.env` heartbeat в `BATCH_ROOT`; lingering `profile-status/*.json = running` без живого owner pid или со stale owner heartbeat считаются terminal `infra_incomplete_cycle`.
- `full-run-batch-matrix.sh` обязан держать durable `profile-status/*.json`, выполнять stale sweep на старте/перед report synthesis и переводить lingering `running` в terminal `failed`; reconstruction в `e2e_batch_report.py` использует `run-status.env`, `profile-status/*.json`, `batch-owner.env` и `run-history.json` как равноправные источники истины для partial roots.
- `full-run-batch-matrix.sh` обязан писать durable inventory per started profile/sweep (`matrix/<matrix-id>/inventory/<batch-id>.json`) с `matrix_id`, matrix file, selected providers/run indexes, `batch_id`, output root, terminal status, key report/log paths и bounded `raw_output_refs` metadata (provider, run_id/task_id/step_id, stdout/stderr bytes/hash/truncation). Этот inventory является decision-support evidence после cleanup temp roots, но не заменяет terminal status/verdict fields.
- frontend live E2E должен различать productive timeout (`active_run_timeout`), browser/page/context closure (`browser_closed`), post-failure API health loss (`api_unreachable`), early `acp serve` exit (`server_exited`) и fallback explicit Playwright assertion failure (`playwright_failed`).
- frontend result JSON должен сохранять server PID/exit code, post-failure health, run id, last run status/current step и diagnostic refs, чтобы release blocker можно было классифицировать без повторного запуска.
- Diagnostic-only `UI_E2E_SCENARIO=api-context-page-close-smoke` проверяет, что Playwright API polling не зависит от закрытой browser page; canonical release gate по-прежнему использует только `init-inspect` и `cancel-refresh`.
- frontend init-inspect budget берётся из effective runtime timeout profile/API и, если задан `ACP_PIPELINE_TIMEOUT_SEC`, может быть поднят до `pipeline_timeout+30s`; fixed cap не применяется по умолчанию. Diagnostic `UI_E2E_INIT_TIMEOUT_CAP_SEC` допустим только как явное manual ограничение и не должен использоваться в canonical release slices.
- `snapshot_reports_missing` после terminal backend failure считается dependent frontend skipped/blocked evidence, а не independent frontend regression.
- provider `model` / `modelUsage` telemetry в stdout/stderr считается обычной diagnostic transcript частью: readiness/reporting не блокируют release по model-family attribution, если command probe, auth/quota checks и artifact smoke успешны.
- generic `codex` plugin/Cloudflare/state-db noise (`chatgpt.com/backend-api/plugins/featured`, Cloudflare HTML, `failed to renew cache TTL`, `state db`, `Operation not permitted`) учитывать как secondary telemetry; сами по себе такие строки не должны поднимать `runner_unavailable`, если raw runtime/session-summary не зафиксировали terminal provider failure.
- для release decision использовать только прогон без diagnostic timeout overrides.

### 6.8 Additional Non-Release Checks

После official release matrices выполнить отдельно:
- parallel shard smoke: два параллельных `full-run-batch.sh` с разными `BATCH_ID`, разными single-provider `BATCH_PROVIDER_FILTER` (например, `qwen-code` и `claude-code`; при необходимости можно заменить один из них на `codex-code`), `BATCH_RUN_SELECTION=1`, `BATCH_FRONTEND_MODE=never`, `BATCH_FRONTEND_CANCEL_MODE=never`;
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
  BATCH_FRONTEND_CANCEL_MODE=never \
  ./scripts/full-run-batch.sh
) &
(
  BATCH_ID=parallel-smoke-${TS}-claude \
  TARGET_REPOS_FILE=examples/repos/github/multi-sentry-ecosystem.repos.yaml \
  BATCH_PROVIDER_FILTER=claude-code \
  BATCH_RUN_SELECTION=1 \
  BATCH_FRONTEND_MODE=never \
  BATCH_FRONTEND_CANCEL_MODE=never \
  BATCH_SKIP_PRECHECK=1 \
  ./scripts/full-run-batch.sh
) &
wait || true

# forced-incomplete diagnostic (example: OpenStack multi-path)
# ВАЖНО: execution overrides задаются через ACP_EXECUTION_*, а не BATCH_*.
BATCH_ID=forced-incomplete-$(date -u +%Y%m%dT%H%M%SZ)-qwen \
TARGET_REPOS_FILE=examples/repos/curated/multi-path.openstack.repos.yaml \
BATCH_PROVIDER_FILTER=qwen-code \
BATCH_RUN_SELECTION=1 \
BATCH_FRONTEND_MODE=never \
BATCH_FRONTEND_CANCEL_MODE=never \
BATCH_SKIP_PRECHECK=1 \
ACP_EXECUTION_STRATEGY=parallel \
ACP_MAX_PARALLEL_TASKS=4 \
ACP_FAILURE_POLICY=best_effort \
ACP_SHARD_DISCOVERY_MODE=heuristics \
ACP_RUNTIME_STEP_TIMEOUT_SEC=15 \
ACP_PIPELINE_TIMEOUT_SEC=300 \
ACP_APPLY_TIMEOUTS_VIA_API=1 \
./scripts/full-run-batch.sh || true
```

## 7) Strict release acceptance

Release `PASS` только если одновременно:
1. Во всех `profile+sweep` строках `strict_status=passed`.
2. Для каждого `profile+sweep`: `backend_total_runs=3`, `backend_hard_pass=3`.
3. `runtime_contract_failed/runner_unavailable/runtime_timeout/infra_signal_terminated/infra_incomplete_cycle/quality_gates_failed/summary_missing/precheck_failed = 0`.
4. `semantic_hard_fail=0`, `off_topic_hits=0`.
5. `artifact_source` только `snapshot` (без `workspace-fallback`).
6. Нет `analysis:evidence-scope` и `analysis:cross-repo-missing`.
7. Нет runtime flow violations (`runtime:*`, `runtime_flow_failed`).
8. Frontend live/cancel smoke: `passed` для всех трёх release providers (`qwen`, `claude`, `codex`).
9. Нет artifact-quality blockers (`artifact_quality:*` в run warnings / batch quality report).

Любое нарушение => `RELEASE BLOCKED`.

## 8) Agent verdict output

Источник истины для финального решения:
- `release_verdict_<matrix-id>.md/.json`

Перед tag/release выполнить offline verifier:
```bash
python3 scripts/verify-release-verdict.py reports/release_verdict_<matrix-id>.json
```

## 9) Common blockers (операционный triage)

1. `repos[1] path does not exist: /tmp/provenarch-live-e2e/...`
- Причина: path profiles не подготовлены на pinned SHA.
- Действие: подготовить локальные checkout в точных absolute paths из curated файлов.

2. `SHA_MISMATCH: ... expected=<sha> got=<sha>`
- Причина: локальный clone существует, но checkout не закреплён на pinned release SHA.
- Действие: выполнить `git -C <repo> checkout --detach <expected_sha>` и повторить verify script.

3. `runtime_timeout` вместе с `runner_unavailable` в том же run
- Политика triage: primary incident class = `runtime_timeout` при явном timeout signal в summary/classifier; `runner_unavailable` остаётся secondary evidence.
- Если `qwen-code` draft step (`step0/2/4`) уже имеет валидный draft manifest и все referenced files, но процесс продолжает actively stream/mutate until runtime timeout, это product bug in valid-artifact stop policy: runtime должен bounded-stop and validate artifacts instead of waiting for the global step timeout. `release-fast-20260523T113202Z` зафиксировал такой случай на frontend `init.step2.asis_docs`.

4. `operational_host_preflight_failed` до старта backend runs
- Причина: невалидный runtime binary surface (`qwen`/`claude`/`codex` в PATH), provider readiness blocker, known codex CLI compatibility issue, либо нерелевантный writable state/tmp surface.
- Действие: починить host prerequisites и повторить запуск; не считать это продуктовым ACP багом.
- Для `claude` отдельный text-only `ACP_READY` timeout не является release blocker: readiness должна подтверждаться `--version` + artifact smoke с sentinel write. Exhausted artifact-smoke timeout остаётся host/provider blocker.

5. `collect_manifest_missing` / `shard-pack-manifest.json is missing` на `init.step1.collect`
- Политика triage: если provider оставил diagnostics, но не написал authored artifacts, общий engine должен сделать одну collect pair-recovery попытку с command-first suggested doc + manifest skeleton targets; если в `write_root` есть authored docs, общий engine должен сделать один manifest-only repair, где `FIRST COLLECT MANIFEST REPAIR COMMAND` является первым task action и пишет только `write_root/shard-pack-manifest.json`. Если repair provider продолжает работать после валидного `shard-pack-manifest.json`, short valid-artifact stop завершает процесс и success всё равно зависит только от validation. Writes outside allowed collect repair set или failure после repair остаются `runtime_contract_failed`.
- Если authored docs отсутствуют и provider был полностью silent, `qwen-code` делает один bounded fresh retry с warning telemetry; если focused collect-pair или draft-artifact repair вернул transient provider API/transport текст без artifacts, `qwen-code` делает один bounded focused-repair retry. Exhausted silent/API retry остаётся `runner_unavailable`. `claude-code` делает bounded silent retry для constitution/collect/validator/proposals steps (`init.step0.constitution`, `init|refresh.step1.collect`, `init|refresh.step3.findings`, `init|refresh.step4.proposals`) и сохраняет zero-output fail-fast для as-is/non-scoped steps; exhausted scoped silence остаётся `runner_unavailable`. Partial authored artifacts у всех providers остаются `runtime_contract_failed`, если manifest/draft/verdict невалиден после focused repair.

6. `Selected model is at capacity` / `429` / `rate limited` в task logs
- Политика triage: классифицировать как `runner_unavailable` (если одновременно нет explicit timeout signal).
- Для `best_effort` partial shard run это считается provider-availability incident, а не runtime execution-semantics drift.

7. `backend_workspace_missing` / `frontend_workspace_missing`
- Harness workspace resolver обязан проверять в порядке `run_dir/headless/arch-workspace` -> `run_dir/arch-workspace` -> `run_dir/workspace`.
- Если workspace найден в одном из candidate roots, это не blocker; если отсутствует во всех roots, инцидент остаётся operational.

8. `parse runtime draft manifest ... unknown field ...` вместе с `runner_unavailable`/capacity сигналами
- Политика triage: primary incident class = `runtime_contract_failed` (parse-signature override); capacity/429 остаются secondary evidence.

9. Provider `model` / `modelUsage` telemetry выглядит неожиданно
- Политика triage: не считать это release blocker само по себе. ACP запускает выбранную provider command surface (`qwen`, `claude`, `codex`) и проверяет command/probe/artifact-smoke/auth/quota behavior; модель под капотом остаётся provider CLI configuration detail.
- Действие: если это мешает операторскому доверию, проверить CLI wrapper/config (`ACP_QWEN_CMD`, `ACP_CLAUDE_CMD`, provider auth profile`) вне release verdict; не править canonical matrices ради model telemetry.

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

В execution report для смешанных сигналов обязательно указывать `primary failure class` отдельно от secondary evidence (например `runtime_contract_failed` + `runner_unavailable`).
