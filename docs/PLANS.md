# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон, текущие активные планы и инженерный operational mirror.

Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
- `docs/archive/PLANS_ARCHIVE_2026-05.md`
- `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Этот файл остаётся рабочим engineering mirror и active ExecPlan surface.

## Когда использовать
Используйте ExecPlan, если:
- работа затрагивает несколько модулей, или
- ожидаемое время > 30–60 минут, или
- затрагиваются контракты/схемы.

---

## Шаблон ExecPlan

### Plan ID
EP-YYYYMMDD-<slug>

### Context
Зачем это нужно? Какие ограничения важны?

### Goals (must have)
- [ ] ...

### Non-goals
- [ ] ...

### Approach
1) ...
2) ...
3) ...

### Files expected to change
- ...

### Acceptance criteria
- [ ] Тесты обновлены/добавлены
- [ ] Схемы валидируются
- [ ] Документация обновлена

### Risks
- ...

### Progress log
- YYYY-MM-DD: ...

---

## Active Plans
Tracker reconciliation from 2026-05-07 consolidated historical active plans into the remaining open slices below. Detailed evidence and classification are archived in `docs/archive/TRACKER_RECONCILIATION_2026-05-07.md`; original historical active plan text was moved to `docs/archive/PLANS_ARCHIVE_2026-05.md` under "Reconciled active plans from 2026-05-07".

### Plan ID
EP-20260507-trusted-live-validation

### Context
Historical active plans from 2026-04-20 through 2026-05-06 repeatedly ended with the same unresolved gate: trusted-host live reruns through the canonical matrix harness. Local behavior fixes and DoD evidence are present for those slices, but live confidence still requires direct `scripts/full-run-batch-matrix.sh` execution from a clean committed trusted workspace with `qwen`, `claude`, and `codex` available.

### Goals (must have)
- [ ] Prepare a clean committed trusted-host worktree with required provider binaries/auth and pinned repo/path prerequisites
- [ ] Run qwen `regres fast` for `examples/e2e-matrix.regres-fast.bank-openedx.yaml` and `examples/e2e-matrix.regres-fast.openstack.yaml`
- [ ] Run qwen+codex diagnostic rerun for bank/openedx and openstack scenarios
- [ ] Run qwen+claude rerun for the current small live bugfix matrix
- [ ] Run claude bank/openedx verification and qwen-focused `regres fast`
- [ ] Run full fresh-all-6 only after smaller slices pass or produce narrower blocker reports
- [ ] Record matrix ids, driver logs, profile/release reports, quality reports, taskrun raw diagnostics, and verifier output where a release verdict is used
- [ ] Update `docs/PLANS.md`/archive with pass/fail evidence and any follow-up fix slices created from live failures

### Non-goals
- [x] Do not add wrapper scripts around `scripts/full-run-batch-matrix.sh`
- [x] Do not wire live matrix into required CI
- [x] Do not edit canonical matrix/profile/timeout files to fit the current machine
- [x] Do not treat live release readiness as passed without `reports/release_verdict_<matrix-id>.json` and `scripts/verify-release-verdict.py` when release verdict is used

### Approach
1) Verify trusted-host prerequisites and clean committed tree.
2) Execute each matrix by setting `E2E_MATRIX_FILE`, provider command envs, and `BATCH_PROVIDER_FILTER`, then invoking `./scripts/full-run-batch-matrix.sh` directly.
3) Monitor driver/status/report artifacts until terminal state.
4) Classify failures into product/runtime/harness/operational buckets and open a focused fix slice before broad reruns when needed.
5) Close historical live-gate tasks only with concrete matrix/report evidence.

### Files expected to change
- `docs/PLANS.md`
- `docs/archive/*` for execution reports or reconciliation notes
- focused code/docs/tests only if live evidence creates a new fix slice

### Acceptance criteria
- [ ] No unresolved `runtime_contract_failed`, `runner_unavailable`, `semantic_hard_fail`, independent frontend failure, stale `running`, or unexpected `analysis:cross-repo-missing` remains for fixed scenarios
- [ ] Every live run has a recorded `matrix_id`, `driver.log`, `profile_matrix_<matrix-id>.*`, quality reports, and taskrun diagnostics
- [ ] Release readiness uses only `reports/release_verdict_<matrix-id>.json`; verifier exits 0 only for `PASS` / `RELEASE READY` / `passed`
- [ ] Any live failure produces a narrower follow-up fix slice instead of ad-hoc matrix edits

### Risks
- Provider outages, auth/quota limits, or missing trusted-host path prerequisites can block live gates without indicating product regression.
- Full fresh-all-6 can be expensive; run it only after smaller gates either pass or isolate residual blockers.

### Progress log
- 2026-05-07: Created by tracker reconciliation. Consolidates live rerun gates from historical active plans; no live matrix was run in this reconciliation slice.

### Plan ID
EP-20260507-cleanup-owner-decisions

### Context
Safe cleanup is already complete, but `docs/PLANS.md` and `docs/BACKLOG.md` retain owner-gated cleanup decisions where deleting or moving tracked surfaces could damage discoverability, review UX, or hidden tooling usage. These items must remain explicit and blocked until owner intent is known.

### Goals (must have)
- [ ] Gather usage evidence for whether `internal/docsync` and `internal/scriptsmeta` should remain test-only packages under `internal/*`
- [ ] Gather evidence and owner decision on retaining both `docs/archive/PLANS_ARCHIVE_2026-04.md` and `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`
- [ ] Gather evidence and owner decision on persisted `fixtures/scenarios/*/golden/readable`
- [ ] Gather evidence and owner decision on duplicated readable scenario fixtures
- [ ] Gather evidence and owner decision on retaining `docs/LOCAL_FULL_RUN_AI_ADVENT.md` as a separate convenience runbook vs pointer/appendix
- [ ] If approved by owners, implement each cleanup as a separate small change set with tests/docs sync

### Non-goals
- [x] Do not move `internal/docsync` or `internal/scriptsmeta` without owner approval
- [x] Do not delete archive snapshots, readable fixtures, duplicated fixtures, or local runbooks without owner approval
- [x] Do not mix destructive cleanup with runtime/live fixes

### Approach
1) Search usage and references for each cleanup candidate.
2) Write a short evidence table with retain/remove/dedupe options and recommended default.
3) Request owner decisions; retain by default if no answer is available.
4) Implement approved cleanup items one at a time with focused tests and full DoD when tracked files change.

### Files expected to change
- `docs/PLANS.md`
- `docs/BACKLOG.md` only if owner-gated backlog wording changes
- candidate files only after explicit owner approval

### Acceptance criteria
- [ ] Each owner-gated item has usage evidence and an explicit retain/remove/dedupe recommendation
- [ ] No destructive cleanup happens without owner approval
- [ ] Approved cleanup items are implemented in separate commits
- [ ] Active plan surface keeps owner-gated decisions visible until resolved

### Risks
- Removing historical docs or readable fixtures can degrade review workflows even if tests do not fail.
- Keeping everything can preserve drift; retained items need explicit ownership or periodic review.

### Progress log
- 2026-05-07: Created by tracker reconciliation from `EP-20260421-cleanup-owner-followups` and `docs/BACKLOG.md` cleanup follow-ups.

---

## Implemented vs Planned (operational mirror)

Канонический stakeholder статус находится в `docs/STAKEHOLDER_DOC.md` → **Canonical Stakeholder Matrix (source of truth)**.
Таблица ниже — инженерный mirror и должна оставаться синхронизированной с канонической матрицей.

| Epic | Статус | Комментарий |
|---|---|---|
| 1 Workspace/contracts | done (beta baseline) | Schema-driven + semantic validation, resolver `path/git_url`, diagnostics API |
| 2 Runtime artifact contracts | done (beta baseline) | Validation + artifact-only runtime execution contract, contract tests |
| 3 Runtime/orchestration seam | done (beta baseline) | Fake default + opt-in headless runtime selector with provider choice (`claude-code` default, `qwen-code`, `codex-code` release peer), persisted runtime execution metadata |
| 4 Model deterministic core | done (beta baseline) | Canonical IDs/collision rules + deterministic regression tests |
| 5 Pipeline 0–4 | done (beta baseline) | `init|refresh` runnable через CLI/API |
| 6 UI baseline | done (beta baseline) | Setup/validate/run/inspect + editors + git helpers |
| 7 Domain-first layer | done (beta baseline) | Per-domain contracts + deterministic Step 1 enrichment canonical domain/team cards without auto-create |
| 8 Baseline bundle | done (beta baseline) | `skills/subagents.yaml` + prompt packs + validation |
| 9 Q&A capability | done (beta boundary) | Workspace-backed QA service + read-only CLI `acp qa` + public read-only `POST /api/qa/ask` |
| 10 Changelog compilers | done (beta baseline) | Iteration changelog materialization в `reports/changelog/*` |
| 11 `POST /api/qa/ask` | done (beta baseline) | Read-only API wrapper over deterministic Q&A service |
| 12–13 | out of MVP | Вне текущего beta scope |
| 14 CI trigger mode | done (beta baseline) | CLI batch required, smoke/golden jobs без live network deps |
| 15 Domain/baseline pack hardening | done (beta baseline) | Baseline skills/prompts wired и versioned в workspace |
