# PLANS_ARCHIVE_2026-06.md

Closed ExecPlans archived from `docs/PLANS.md` in June 2026.

### Plan ID
EP-20260601-live-e2e-complex-diagnostics

### Context
Canonical live E2E release slices уже покрывают стабильные repo sets, но diagnostic/regression coverage не должна зацикливаться на одних и тех же продуктах. Нужен отдельный non-release selector для более сложных публичных продуктов и явная operator policy: в ручных live E2E прогонах желательно ротировать продукты и feature focus, чтобы успешность не измерялась только привычным happy path.

### Goals (must have)
- [x] Добавить diagnostic-only complex repo presets с pinned GitHub SHA.
- [x] Добавить runnable non-release matrix files и catalog selector для прямого `scripts/full-run-batch-matrix.sh`.
- [x] Зафиксировать rotation guidance: каждый diagnostic прогон по возможности выбирает разные продукты и feature areas.
- [x] Обновить planner/tests/docs без изменения canonical release verdict contract.

### Non-goals
- [x] Не менять canonical `release fast|long|full` matrices, curated path presets или release verdict policy.
- [x] Не добавлять новые headless providers или hosted/security/compliance enforcement.
- [x] Не делать complex diagnostic selector required CI или release readiness signal.

### Approach
1) Добавить GitHub `repos.yaml` presets для Temporal, Backstage, Airflow, Appwrite и Saleor.
2) Добавить отдельные `examples/e2e-matrix.diagnostic.*.yaml`, чтобы не нарушать unique profile id contract внутри matrix.
3) Расширить `examples/e2e-profile-catalog.yaml` и `scripts/live-e2e-plan.py` selector size `complex`.
4) Обновить runbook/testing docs и focused tests.

### Files changed
- `.agents/skills/e2e-live-gate/SKILL.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/TESTING_STRATEGY.md`
- `examples/e2e-profile-catalog.yaml`
- `examples/e2e-matrix.diagnostic.*.yaml`
- `examples/repos/github/*.repos.yaml`
- `scripts/live-e2e-plan.py`
- `scripts/tests/live_e2e_plan_test.py`
- `scripts/tests/matrix_release_contract_test.py`

### Acceptance criteria
- [x] `python3 -m unittest scripts.tests.live_e2e_plan_test scripts.tests.matrix_release_contract_test`
- [x] `python3 scripts/live-e2e-plan.py --mode regres --size complex --providers qwen --format shell`
- [x] Документация явно говорит, что complex selector diagnostic-only и требует ротации products/features.
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`

### Risks
- Первые shard counts для новых публичных repo sets неизвестны до trusted-machine diagnostic run; catalog маркирует их как `unmeasured`.

### Progress log
- 2026-06-01: Started as diagnostic-only catalog/planner/docs update.
- 2026-06-01: Added `regres complex`, pinned complex product presets, docs rotation policy and tests.
- 2026-06-01: Review pass updated drifted `temporalio/temporal` and `apache/airflow` pins to current remote HEAD and moved the completed ExecPlan to this archive.
