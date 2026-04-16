---
name: acp-e2e-live-gate
description: Используй для trusted-machine pre-release live E2E gate, matrix harness по wave1/wave2, additional non-release smoke/diagnostic прогонов и manual backend_only acceptance audit.
---

## Когда использовать
- Нужен `PASS|FAIL` pre-release live verdict.
- Нужно прогнать `scripts/full-run-batch-matrix.sh` без wrapper.
- Нужно проверить новый runtime behavior beyond release verdict: parallel smoke, forced-incomplete diagnostic, `backend_only` acceptance.

## Source of truth
1) `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
2) `examples/e2e-matrix.release-wave1.yaml`
3) `examples/e2e-matrix.release-wave2.yaml`

Не подменяй эти источники ad-hoc matrix-файлами, если пользователь явно не просит diagnostic/custom run.
Каталог target repos в runbook нужен для понимания scope и pinned presets, но не заменяет official wave1/wave2 matrices для release verdict.

## Operational rules
1) Release gate запускать только на trusted машине, где доступны canonical `path` checkout'ы из curated presets под `/tmp/provenarch-live-e2e/...`.
2) Перед release run проверить, что локальные `path` checkout'ы существуют и совпадают с pinned SHA из curated/github presets.
3) Для release verdict использовать только `reports/release_verdict_<matrix-id>.json`.
4) Не использовать diagnostic timeout overrides в official release matrix.
5) Для non-release execution overrides задавать effective execution profile через `ACP_EXECUTION_*` и `ACP_REPO_SELECTION`, а не через `BATCH_*`.
6) Не добавлять wrapper-скрипт поверх `scripts/full-run-batch-matrix.sh`.
7) Не редактировать canonical wave files или curated `repos_file`, чтобы адаптировать release gate под неподходящий хост; если текущая машина не удовлетворяет prerequisites, остановить прогон и зафиксировать operational blocker.

## Fail-Fast Host Check
Перед DoD и matrix run сначала проверить, подходит ли хост для official release gate:

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

- Если `parent_writable=false`, или existing `root` не является writable directory, official release wave1/wave2 на этом хосте запускать нельзя.
- В таком случае skill должен остановить release gate как operational blocker, а не пытаться править matrix/curated files под текущую машину.

Если хост подходит, но curated `path` checkout'ы ещё не подготовлены:
- выполнить one-time bootstrap из `docs/RELEASE_LIVE_E2E_RUNBOOK.md` → `2.2 One-time canonical path bootstrap`;
- подготовить exact local checkouts или symlink-targets по canonical `/tmp/provenarch-live-e2e/...` путям;
- выставить каждый checkout на pinned SHA из curated/github presets до запуска official wave matrices.

## Canonical scope
- Wave 1: `posthog/posthog`, `microservices-patterns/ftgo-application`, Open edX ecosystem, Sentry ecosystem.
- Wave 2: `GoogleCloudPlatform/bank-of-anthos`, OpenStack ecosystem.

## Required flow
1) Выполнить preflight:
   `make contracts test lint build`
   `npm ci --prefix ui`
   `npm exec --prefix ui playwright install chromium`
2) Выполнить official wave1 matrix.
3) Выполнить official wave2 matrix.
4) Выполнить additional non-release checks:
   - parallel smoke: два параллельных `full-run-batch-5x2.sh` с разными `BATCH_ID` и `BATCH_PROVIDER_FILTER=qwen-code|claude-code`
   - forced-incomplete diagnostic run с `ACP_EXECUTION_STRATEGY=parallel`, `ACP_MAX_PARALLEL_TASKS=4`, `ACP_FAILURE_POLICY=best_effort`, `ACP_SHARD_DISCOVERY_MODE=semantic`, `ACP_REPO_SELECTION=backend_only`
5) Выполнить manual acceptance audit по `repo-selection-summary.json` under `scale-backend`.

## Acceptance focus
- Во всех `profile+sweep`: `strict_status=passed`
- `backend_total_runs=10`, `backend_hard_pass=10`
- Ноль `runtime_parse`, `runner_unavailable`, `runtime_timeout`, `infra_*`, `summary_missing`, `precheck_failed`
- Frontend init/cancel passed для обоих провайдеров
- `artifact_source` только `snapshot`
- Нет `analysis:evidence-scope` и `analysis:cross-repo-missing`

## backend_only audit
- Проверять `reports/taskruns/*-repo-selection-summary.json`
- Ожидание: любой repo с `effective_role=frontend` under `backend_only` должен быть `included=false`
- Дополнительно frontend-like repo names тоже должны быть явно отмечены, даже если ACP оставил им `effective_role=unknown`
- Если frontend-like repo остаётся `included=true` с `effective_role=unknown`, это product gap, даже если official strict verdict прошёл

## Common blockers
- `repos[1] path does not exist: /tmp/provenarch-live-e2e/...`
  Причина: path profiles не подготовлены.
- `SHA_MISMATCH ... expected=<sha> got=<sha>`
  Причина: local clone существует, но не закреплён на pinned release SHA.
- timeout + `runner_unavailable` в одном run:
  primary triage class = `runtime_timeout`, если summary/classifier явно фиксирует timeout.
