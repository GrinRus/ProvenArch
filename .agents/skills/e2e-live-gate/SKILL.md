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
   - `examples/e2e-matrix.smoke-tiny.bank.yaml` (diagnostic selector only)
   - `examples/e2e-matrix.regres-fast.bank-openedx.yaml`
   - `examples/e2e-matrix.regres-fast.openstack.yaml`
   - `examples/e2e-matrix.regres-long.yaml`
   - `examples/e2e-matrix.diagnostic.sentry.yaml` (diagnostic selector only)
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
9) Дополнительная отладка на `claude`/`codex` остаётся ручной фазой и не входит в canonical expected backend totals для `regres*` профилей; generated diagnostic selectors считают totals по фактически выбранным providers/run indexes.
10) Canonical acceptance запускать только из clean committed tree или отдельного clean worktree без unrelated локальных правок; `BATCH_SKIP_PRECHECK=1` допустим только для diagnostic/triage run.
11) Если canonical run идёт из отдельного clean worktree, сначала установить локальные UI deps в этом worktree (`npm ci --prefix ui`), иначе precheck на `make test` сломается ещё до live batch execution.
12) Canonical matrix slices уже несут native `timeout_profile`; не задавай `ACP_*TIMEOUT*` вручную для штатного запуска. В non-release manual diagnostic внешние timeout env допустимы, в release-mode они остаются blocked-by-default.
13) Batch/profile reports нужно читать только в рамках реально выбранной поверхности (`selected_providers`, `selected_run_indexes`); qwen-only `run1` regression run не должен интерпретироваться как synthetic `2x5` matrix.
14) Для collect steps canonical runtime теперь делает одну artifact-repair попытку для skeletal/generic-only manifests; если repair не улучшил artifact fidelity, исходный `write_root` откатывается, а step классифицируется как `runner_parse_failed` / `runtime_parse`, а не как nominal success.
15) Frontend cancel smoke должен идти из свежей копии backend `arch-workspace`, а terminal cancel verdict обязан сохранять `error_code=run_canceled`, даже если рядом всплыл validation/layout failure.
16) Для flexible combinations можно использовать `python3 scripts/live-e2e-plan.py ... --format shell`; этот tool только печатает прямые `full-run-batch-matrix.sh` команды и не заменяет release harness.
17) Regress/release acceptance всегда включает artifact quality: `reports/taskruns/<run_id>-quality.json`, `quality_report_<batch-id>.md`, `quality_gates_failed=0`, отсутствие `artifact_quality:*`.

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
- `smoke tiny`: diagnostic-only, exactly one provider, one `bank-of-anthos` repo set, `RUN_COUNT=1`, frontend off by default, `1` backend run total.
- `regres fast`: qwen-only default, provider-selectable for generated diagnostic runs, implicit baseline, composite из `bank-of-anthos + openedx` и отдельного `openstack` slice, `3 × providers × RUN_COUNT` backend runs total.
- `regres long`: qwen-only default, provider-selectable for generated diagnostic runs, implicit baseline, `posthog + ftgo`, `2 × providers × RUN_COUNT` backend runs total.
- `regres full`: diagnostic-only provider-selectable baseline over all 6 canonical repo sets, including Sentry, `6 × providers × RUN_COUNT` backend runs total.
- `release fast`: three-provider, explicit `baseline + parallel-default`, `bank-of-anthos + openedx`, `12` backend runs total.
- `release long`: three-provider, explicit `baseline + parallel-default`, `posthog + openstack`, `12` backend runs total.
- `release full`: composite из `release fast` + `release long` + `ftgo + sentry-ecosystem`, `36` backend runs total.
- canonical timeout presets:
  - `short-window`: step `3600s`, pipeline `7200s`, ui-init `1200s`
  - `medium-window`: step `5400s`, pipeline `14400s`, ui-init `1500s`
  - `extended-window`: step `10800s`, pipeline `21600s`, ui-init `1800s`

## Required flow
1) Для самого быстрого trusted-machine signal выполнить generated `smoke tiny`, например `python3 scripts/live-e2e-plan.py --mode smoke --size tiny --providers qwen --format shell`, затем запустить напечатанную direct command.
2) Для базовой отладки/regression по умолчанию выполнить `regres fast` или `regres long` через catalog-approved slices с `BATCH_PROVIDER_FILTER=qwen-code` или сгенерировать direct commands через `scripts/live-e2e-plan.py`.
3) Если нужен full pre-release verdict, выполнить preflight:
   `make contracts test lint build`
   `npm ci --prefix ui`
   `npm exec --prefix ui playwright install chromium`
4) Выполнить нужный canonical release slice:
   - `release fast`
   - `release long`
   - или весь `release full` как три последовательных matrix invocation
5) При дополнительной отладке повторить нужный regression/non-release diagnostic slice с `BATCH_PROVIDER_FILTER=claude-code` или `BATCH_PROVIDER_FILTER=codex-code`, если нужен isolated provider diagnostic вне canonical regression totals; release-mode provider subsets не использовать.
6) Выполнить additional non-release checks:
   - parallel smoke: два параллельных `full-run-batch-5x2.sh` с разными `BATCH_ID` и разными single-provider `BATCH_PROVIDER_FILTER` (например, `qwen-code` и `claude-code`; при необходимости заменить один из них на `codex-code`)
   - forced-incomplete diagnostic run с `ACP_EXECUTION_STRATEGY=parallel`, `ACP_MAX_PARALLEL_TASKS=4`, `ACP_FAILURE_POLICY=best_effort`, `ACP_SHARD_DISCOVERY_MODE=heuristics`
7) Проверить matrix invariant: для одного `profile_id` sweeps `baseline` и `parallel-default` дают одинаковый shard-plan.

## Acceptance focus
- Во всех release slice verdicts: `strict_status=passed`
- `artifact_source` только `snapshot`
- Нет `analysis:evidence-scope`, `analysis:cross-repo-missing`, `runtime_parse`, `runner_unavailable`, `runtime_timeout`, `infra_*`, `summary_missing`, `precheck_failed`
- Frontend init/cancel passed для всех трёх release providers (`qwen`, `claude`, `codex`)
- Нет `artifact_quality:*`; bank-like collapse к одному `cite.runtime-summary` должен либо починиться provider-side repair, либо остаться явным blocker
- Для одного `profile_id` shard-plan invariant между `baseline` и `parallel-default` = `passed`
- Для `release full` все constituent `release_verdict_<matrix-id>.json` должны иметь `PASS`

## Common blockers
- `repos[1] path does not exist: /tmp/provenarch-live-e2e/...`
  Причина: path profiles не подготовлены.
- `SHA_MISMATCH ... expected=<sha> got=<sha>`
  Причина: local clone существует, но не закреплён на pinned release SHA.
- timeout + `runner_unavailable` в одном run:
  primary triage class = `runtime_timeout`, если summary/classifier явно фиксирует timeout.
- `runner_parse_failed` на `single-git_url`/`qwen-code` в `regres fast`
  Причина: live provider ушёл в tool chatter + partial TaskResult drafting вместо одного финального JSON; canonical reference incident зафиксирован 2026-04-17, Open edX companion run тогда был прерван вручную и не считается отдельным regression signal.
- `runtime_timeout` на clean canonical slice
  Причина: либо запускается legacy matrix без committed `timeout_profile`, либо даже native time budget оказался недостаточным; сначала проверить `timeout_profile` matrix-файла и `full-run.log`, затем уже считать это реальной runtime/provider деградацией.
