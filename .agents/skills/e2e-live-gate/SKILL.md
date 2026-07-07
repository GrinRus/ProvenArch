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
   - `examples/e2e-matrix.diagnostic.temporal.yaml` (diagnostic selector only)
   - `examples/e2e-matrix.diagnostic.backstage.yaml` (diagnostic selector only)
   - `examples/e2e-matrix.diagnostic.airflow.yaml` (diagnostic selector only)
   - `examples/e2e-matrix.diagnostic.appwrite.yaml` (diagnostic selector only)
   - `examples/e2e-matrix.diagnostic.saleor.yaml` (diagnostic selector only)
   - `examples/e2e-matrix.release-fast.yaml`
   - `examples/e2e-matrix.release-long.yaml`
   - `examples/e2e-matrix.release-full.ftgo-sentry.yaml`

Не подменяй canonical profile taxonomy ad-hoc matrix-файлами, если пользователь явно не просит diagnostic/custom run.

## Black-box operator protocol
Live E2E skill теперь работает как step-by-step black-box evaluator, а не как report-first cookbook.

Layering для local `manual-live-e2e workflow`: live-e2e skill -> local trusted-machine operator procedure -> direct public harness commands -> ACP runtime/provider/UI evidence. Scripts produce facts and machine execution verdicts only; запускающий SWE-agent writes separate UX and artifact-quality assessments. Это operator procedure on trusted host, not GitHub Actions workflow.

После каждой фазы фиксируй короткий step report в формате:

```text
goal: <что проверяем>
action: <какую публичную поверхность вызвали/прочитали>
observed evidence: <команды, UI/API/log/report/artifact/verifier paths>
status: passed|failed|skipped|blocked
primary classification: none|operational_host_preflight_failed|precheck_failed|runtime_timeout|runner_unavailable|runtime_contract_failed|runtime_flow_failed|frontend_failed|infra_incomplete_cycle|infra_signal_terminated|release_verdict_FAIL|...
next decision: <continue|stop|rerun diagnostic|verify verdict|final report>
```

Разрешённые поверхности только публичные/operator-facing:
- direct harness commands (`scripts/full-run-batch-matrix.sh`, `scripts/full-run-batch.sh`, `scripts/live-e2e-plan.py --format shell`);
- UI/API surfaces;
- generated reports under `reports/*`;
- taskrun artifacts/logs/raw metadata under workspace/report roots;
- matrix inventories/status files;
- `scripts/verify-release-verdict.py` output.

Запрещено чинить прогон изменением canonical matrix files, curated repo files, command shims, compatibility aliases или test-only override env. Host/provider/path blockers надо остановить и классифицировать как operational blocker. Не добавлять GitHub-hosted `manual-live-e2e` workflow для live providers в этом flow.

Harness больше не пишет `blackbox_e2e_steps_*`: эти файлы были machine-authored pseudo-reasoning и не являются evidence. Release readiness требует `reports/swe_ux_assessment_<matrix-id>.md` и `reports/swe_artifact_quality_assessment_<matrix-id>.md` с matching matrix id и `decision: accepted`; optional operator notes можно вести по `docs/templates/LIVE_E2E_OPERATOR_ASSESSMENT.md`.

## Operational rules
1) Release gate запускать только на trusted машине, где доступны canonical `path` checkout'ы из curated presets под `/tmp/provenarch-live-e2e/...`.
2) Перед release run проверить, что локальные `path` checkout'ы существуют и совпадают с pinned SHA из curated/github presets.
3) `reports/release_verdict_<matrix-id>.json` использовать только как machine execution verdict; final release readiness additionally requires accepted SWE UX and artifact-quality reports.
4) Не использовать diagnostic timeout overrides в official release slices.
5) Для non-release execution overrides задавать effective execution profile через `ACP_EXECUTION_*`, а не через `BATCH_*`.
6) Не добавлять wrapper-скрипт поверх `scripts/full-run-batch-matrix.sh`.
7) В release-mode matrix обязан иметь explicit `sweeps[]` с ровно `baseline` + `parallel-default`, ровно один `single-*` и один `multi-*` профиль, и `RUN_COUNT=1`; implicit baseline допустим только для non-release/diagnostic.
8) Не редактировать canonical release slices или curated `repos_file`, чтобы адаптировать release gate под неподходящий хост; если текущая машина не удовлетворяет prerequisites, остановить прогон и зафиксировать operational blocker.
9) Дополнительная отладка на `claude`/`codex` остаётся ручной фазой и не входит в canonical expected backend totals для `regres*` профилей; generated diagnostic selectors считают totals по фактически выбранным providers/run indexes.
10) Canonical acceptance запускать только из clean committed tree или отдельного clean worktree без unrelated локальных правок; `BATCH_SKIP_PRECHECK=1` допустим только в test-only harness mode с явным `ACP_TEST_ALLOW_BATCH_SKIP_PRECHECK=1`.
11) Если canonical run идёт из отдельного clean worktree, сначала установить локальные UI deps в этом worktree через exact Node resolver (`./scripts/run-npm.sh ci --prefix ui`), иначе precheck на `make test` сломается ещё до live batch execution.
12) Live harness требует exact Node.js из `.node-version` и npm из того же toolchain; если `.node-version` требует `22.21.1`, то Homebrew `node@22` с `22.22.3` остаётся `precheck_failed`. Для trusted smoke задавай `ACP_NODE_TOOL_CANDIDATES=/path/to/node-22.21.1/bin`, а не отключай version check.
13) Canonical matrix slices уже несут native `timeout_profile`; не задавай `ACP_*TIMEOUT*` вручную для штатного запуска. В non-release manual diagnostic внешние timeout env допустимы, в release-mode они остаются blocked-by-default.
14) Batch/profile reports нужно читать только в рамках реально выбранной поверхности (`selected_providers`, `selected_run_indexes`); qwen-only `run1` regression run не должен интерпретироваться как synthetic `2x5` matrix.
15) Для headless providers runtime использует общий artifact-only process engine и тонкие adapters; stdout/stderr являются diagnostics, а success берётся только из валидных artifacts.
16) Frontend live release check покрывает только real user-facing `init-inspect` artifact/UI/API inspection. Cancellation coverage должна жить в deterministic fake-runtime UI/API tests, не в trusted live provider release gate. Optional `UI_E2E_QA_SMOKE=1` допускается только для non-release/fake-runtime UX evidence и не является machine execution verdict input; release UX readiness still requires Ask-flow evidence in `swe_ux_assessment_<matrix-id>.md`.
17) Для flexible combinations можно использовать `python3 scripts/live-e2e-plan.py ... --format shell`; этот tool только печатает прямые `full-run-batch-matrix.sh` команды и не заменяет release harness.
18) `reports/taskruns/<run_id>-quality.json` is public black-box evidence. Any `quality_signals[].code` with prefix `artifact_quality.` and legacy `artifact_quality:` run warning fails artifact-quality status and strict batch/matrix/release verdicts while preserving separate `runtime_contract_status`; broader artifact judgment still belongs in `swe_artifact_quality_assessment_<matrix-id>.md`.

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
- `regres complex`: diagnostic-only provider-selectable baseline over product/feature rotation targets (Temporal, Backstage, Airflow, Appwrite, Saleor), `5 × providers × RUN_COUNT` backend runs total; rotate products and feature areas between runs where feasible and do not treat this as release readiness.
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
4) Backend execution evidence inspection: читать `run_matrix_*`, `execution_report_*`, taskrun quality JSON public signals, raw metadata/logs и inventory/status evidence; классифицировать primary execution failure после каждого профиля, отдельно фиксируя `runtime_contract_status` и `artifact_quality_status`.
5) Frontend UI/API inspection: читать frontend init result JSON/MD reports, Playwright/server logs, screenshot refs и UI/API evidence; dependent skips после backend failure не считать independent frontend regression. Ask UX smoke evidence учитывать как UX assessment evidence, не как machine execution verdict source.
6) Release execution verdict verification: проверить `python3 scripts/verify-release-verdict.py reports/release_verdict_<matrix-id>.json`; verifier additionally requires accepted SWE UX and artifact-quality reports next to the verdict JSON.
7) Final black-box report: свести по шагам `goal / action / observed evidence / status / primary classification / next decision`; для `release full` все constituent verdict JSON должны иметь `PASS` and accepted companion SWE reports.

## Acceptance focus
- Во всех release slice verdicts: `strict_status=passed`
- `artifact_source` только `snapshot`
- Нет `runtime_parse`, `runner_unavailable`, `runtime_timeout`, `infra_*`, `summary_missing`, `precheck_failed`
- Frontend init-inspect passed для всех трёх release providers (`qwen`, `claude`, `codex`)
- Accepted SWE UX assessment and accepted SWE artifact-quality assessment; product-emitted `artifact_quality.*` signals such as bank-like collapse to one generic `cite.runtime-summary` are strict artifact-quality blockers, while broader SWE judgment remains a companion assessment
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
  Причина: либо native time budget оказался недостаточным, либо provider/runtime завис; сначала проверить `timeout_profile` matrix-файла, `full-run.log`, `execution_report_*`, `run_matrix_*`, inventories/status files и taskrun raw logs, затем считать это runtime/provider деградацией.
