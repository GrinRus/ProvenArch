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

Не подменяй canonical profile taxonomy ad-hoc matrix-файлами, если пользователь явно не просит diagnostic/custom run.

## Black-box operator protocol
Live E2E skill теперь работает как step-by-step black-box evaluator, а не как report-first cookbook.

Layering для local `manual-live-e2e workflow`: live-e2e skill -> local trusted-machine manual workflow -> internal evaluator helper -> existing project flow. Это operator procedure on trusted host, not GitHub Actions workflow. `scripts/internal/live-e2e-evaluator.sh` является source-only implementation detail для durable step evidence; он не является public entrypoint и не заменяет direct harness commands.

После каждой фазы фиксируй короткий step report в формате:

```text
goal: <что проверяем>
action: <какую публичную поверхность вызвали/прочитали>
observed evidence: <команды, UI/API/log/report/artifact/verifier paths>
status: passed|failed|skipped|blocked
primary classification: none|operational_host_preflight_failed|precheck_failed|runtime_timeout|runner_unavailable|runtime_contract_failed|runtime_flow_failed|quality_gates_failed|release_verdict_FAIL|...
next decision: <continue|stop|rerun diagnostic|verify verdict|final report>
```

Разрешённые поверхности только публичные/operator-facing:
- direct harness commands (`scripts/full-run-batch-matrix.sh`, `scripts/full-run-batch.sh`, `scripts/live-e2e-plan.py --format shell`);
- UI/API surfaces;
- generated reports under `reports/*`;
- taskrun artifacts/logs/raw metadata under workspace/report roots;
- matrix inventories/status files;
- `scripts/verify-release-verdict.py` output.

Запрещено чинить прогон изменением canonical matrix files, curated repo files или compatibility aliases. Host/provider/path blockers надо остановить и классифицировать как operational blocker. Не добавлять GitHub-hosted `manual-live-e2e` workflow для live providers в этом flow.

Harness через internal evaluator helper дополнительно пишет durable evidence:
- `reports/blackbox_e2e_steps_<batch-id>.jsonl`
- `reports/blackbox_e2e_steps_<batch-id>.md`
- `reports/blackbox_e2e_steps_<matrix-id>.jsonl`
- `reports/blackbox_e2e_steps_<matrix-id>.md`

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
14) Для headless providers runtime использует общий artifact-only process engine и тонкие adapters; stdout/stderr являются diagnostics, а success берётся только из валидных artifacts.
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
1) Host/tree/provider/path preflight: проверить trusted host, clean tree/worktree, provider binaries, writable roots, canonical path checkout'ы и pinned SHA. Зафиксировать step report и остановиться на blockers.
2) Selector and direct command planning: выбрать catalog slice или сгенерировать direct command через `scripts/live-e2e-plan.py --format shell`; не запускать wrapper и не править canonical matrices. Зафиксировать planned command/evidence.
3) Matrix execution monitoring: запускать только `scripts/full-run-batch-matrix.sh`, отслеживать matrix/profile status, batch owner heartbeat, driver logs и durable inventories.
4) Backend artifact and quality inspection: читать `run_matrix_*`, `quality_report_*`, taskrun quality JSON, raw metadata/logs и batch black-box step report; классифицировать primary failure после каждого профиля.
5) Frontend UI/cancel inspection: читать frontend init/cancel result JSON/MD reports, Playwright/server logs и UI/API evidence; dependent skips после backend failure не считать independent frontend regression.
6) Release verdict verification: readiness брать только из `reports/release_verdict_<matrix-id>.json`; проверить `python3 scripts/verify-release-verdict.py reports/release_verdict_<matrix-id>.json`.
7) Final black-box report: свести по шагам `goal / action / observed evidence / status / primary classification / next decision`; для `release full` все constituent verdict JSON должны иметь `PASS`.

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
  Исторический legacy incident: до удаления semantic stdout/TaskResult contract live provider мог уйти в tool chatter + partial JSON drafting. В актуальном artifact-only runtime такой сигнал должен проявляться как `runtime_contract_failed`, `runner_unavailable` или `runtime_timeout` с raw-output refs; не использовать этот старый инцидент как текущую qwen-specific диагностику.
- `runner_unavailable` на qwen-only smoke без raw stdout/stderr
  Причина: qwen adapter policy классифицирует fully silent no-artifact path или silent retry exhaustion как provider availability incident. Сначала проверить raw metadata и artifact state; valid artifacts after controlled stop должны считаться success.
- `shard-pack-manifest.json is missing` / `collect_manifest_missing` на `init.step1.collect`
  Это не qwen-only signal. Если authored docs уже есть, shared engine должен сделать один manifest-only repair через тот же provider adapter; repair читает только current write_root + repo evidence roots, не sibling `reports/taskruns`, и write-set guard разрешает менять только `shard-pack-manifest.json`; failure после repair = `runtime_contract_failed`. Если authored docs отсутствуют и provider silent/no-artifact, pre-artifact monitor должен bounded-stop all live adapters; qwen может остаться `runner_unavailable`, остальные обычно `runtime_contract_failed`, если нет auth/rate-limit markers.
- `operational_host_preflight_failed` с codex model/version текстом
  Это host/provider readiness blocker до deep run. Обновить `codex` или явно задать `ACP_CODEX_CMD_BIN` на совместимый binary; не считать product verdict.
- `runtime_timeout` на clean canonical slice
  Причина: либо native time budget оказался недостаточным, либо provider/runtime завис; сначала проверить `timeout_profile` matrix-файла, `blackbox_e2e_steps_*`, `full-run.log` и taskrun raw logs, затем считать это runtime/provider деградацией.
