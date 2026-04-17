---
name: acp-e2e-live-gate
description: Используй для trusted-machine pre-release live E2E gate, canonical 5-profile taxonomy (`regres fast|long`, `release fast|long|full`) и дополнительных non-release smoke/diagnostic прогонов поверх baseline/parallel-default sweeps.
---

## Когда использовать
- Нужен `PASS|FAIL` pre-release live verdict.
- Нужно прогнать `scripts/full-run-batch-matrix.sh` без wrapper.
- Нужно проверить новый runtime behavior beyond release verdict: parallel smoke, forced-incomplete diagnostic и shard-plan invariant.

## Source of truth
1) `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
2) `examples/e2e-profile-catalog.yaml`
3) canonical runnable slices:
   - `examples/e2e-matrix.regres-fast.bank-openedx.yaml`
   - `examples/e2e-matrix.regres-fast.openstack.yaml`
   - `examples/e2e-matrix.regres-long.yaml`
   - `examples/e2e-matrix.release-fast.yaml`
   - `examples/e2e-matrix.release-long.yaml`
   - `examples/e2e-matrix.release-full.ftgo-sentry.yaml`

Legacy compatibility only:
- `examples/e2e-matrix.regression-wave1.yaml`
- `examples/e2e-matrix.release-wave1.yaml`
- `examples/e2e-matrix.release-wave2.yaml`

Не подменяй canonical profile taxonomy ad-hoc matrix-файлами, если пользователь явно не просит diagnostic/custom run.

## Operational rules
1) Release gate запускать только на trusted машине, где доступны canonical `path` checkout'ы из curated presets под `/tmp/provenarch-live-e2e/...`.
2) Перед release run проверить, что локальные `path` checkout'ы существуют и совпадают с pinned SHA из curated/github presets.
3) Для release verdict использовать только `reports/release_verdict_<matrix-id>.json`.
4) Не использовать diagnostic timeout overrides в official release slices.
5) Для non-release execution overrides задавать effective execution profile через `ACP_EXECUTION_*`, а не через `BATCH_*`.
6) Не добавлять wrapper-скрипт поверх `scripts/full-run-batch-matrix.sh`.
7) В release-mode matrix обязан иметь explicit `sweeps[]` с ровно `baseline` + `parallel-default`, ровно один `single-*` и один `multi-*` профиль, и `RUN_COUNT=1`; implicit baseline допустим только для non-release/diagnostic.
8) Не редактировать canonical release slices или curated `repos_file`, чтобы адаптировать release gate под неподходящий хост; если текущая машина не удовлетворяет prerequisites, остановить прогон и зафиксировать operational blocker.
9) Дополнительная отладка на `claude` остаётся ручной фазой и не входит в expected backend totals для `regres*` профилей.

## Fail-Fast Host Check
Перед DoD и matrix run сначала проверить, подходит ли хост для canonical release slices:

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

- Если `parent_writable=false`, или existing `root` не является writable directory, canonical release slices на этом хосте запускать нельзя.
- В таком случае skill должен остановить release gate как operational blocker, а не пытаться править matrix/curated files под текущую машину.

Если хост подходит, но curated `path` checkout'ы ещё не подготовлены:
- выполнить one-time bootstrap из `docs/RELEASE_LIVE_E2E_RUNBOOK.md` → `2.2 One-time canonical path bootstrap`;
- подготовить exact local checkouts или symlink-targets по canonical `/tmp/provenarch-live-e2e/...` путям;
- выставить каждый checkout на pinned SHA из curated/github presets до запуска canonical release slices.

## Canonical profile taxonomy
- `regres fast`: qwen-only, implicit baseline, composite из `bank-of-anthos + openedx` и отдельного `openstack` slice, `3` backend runs total.
- `regres long`: qwen-only, implicit baseline, `posthog + ftgo`, `2` backend runs total.
- `release fast`: dual-provider, explicit `baseline + parallel-default`, `bank-of-anthos + openedx`, `8` backend runs total.
- `release long`: dual-provider, explicit `baseline + parallel-default`, `posthog + openstack`, `8` backend runs total.
- `release full`: composite из `release fast` + `release long` + `ftgo + sentry-ecosystem`, `24` backend runs total.

## Required flow
1) Для базовой отладки/regression по умолчанию выполнить `regres fast` или `regres long` через catalog-approved slices с `BATCH_PROVIDER_FILTER=qwen-code`.
2) Если нужен full pre-release verdict, выполнить preflight:
   `make contracts test lint build`
   `npm ci --prefix ui`
   `npm exec --prefix ui playwright install chromium`
3) Выполнить нужный canonical release slice:
   - `release fast`
   - `release long`
   - или весь `release full` как три последовательных matrix invocation
4) При дополнительной отладке повторить нужный regression/release slice с `BATCH_PROVIDER_FILTER=claude-code`.
5) Выполнить additional non-release checks:
   - parallel smoke: два параллельных `full-run-batch-5x2.sh` с разными `BATCH_ID` и `BATCH_PROVIDER_FILTER=qwen-code|claude-code`
   - forced-incomplete diagnostic run с `ACP_EXECUTION_STRATEGY=parallel`, `ACP_MAX_PARALLEL_TASKS=4`, `ACP_FAILURE_POLICY=best_effort`, `ACP_SHARD_DISCOVERY_MODE=heuristics`
6) Проверить matrix invariant: для одного `profile_id` sweeps `baseline` и `parallel-default` дают одинаковый shard-plan.

## Acceptance focus
- Во всех release slice verdicts: `strict_status=passed`
- `artifact_source` только `snapshot`
- Нет `analysis:evidence-scope`, `analysis:cross-repo-missing`, `runtime_parse`, `runner_unavailable`, `runtime_timeout`, `infra_*`, `summary_missing`, `precheck_failed`
- Frontend init/cancel passed для обоих провайдеров
- Для одного `profile_id` shard-plan invariant между `baseline` и `parallel-default` = `passed`
- Для `release full` все constituent `release_verdict_<matrix-id>.json` должны иметь `PASS`

## Common blockers
- `repos[1] path does not exist: /tmp/provenarch-live-e2e/...`
  Причина: path profiles не подготовлены.
- `SHA_MISMATCH ... expected=<sha> got=<sha>`
  Причина: local clone существует, но не закреплён на pinned release SHA.
- timeout + `runner_unavailable` в одном run:
  primary triage class = `runtime_timeout`, если summary/classifier явно фиксирует timeout.
