# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит только шаблон, текущие активные планы и инженерный operational mirror.

Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
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
### Plan ID
EP-20260422-headless-legacy-cleanup

### Context
Live `regres fast` на `codex-code` показал, что collect provider может писать legacy `shard-pack-manifest` shape и подхватывать schema drift из runtime workspace (`reports/taskruns`, raw logs, archived artifacts). Параллельно обнаружены downstream gaps: `step2.asis_docs` стартует даже при unusable collect, а batch/matrix harness может оставлять child/profile в stale `running`.

### Goals (must have)
- [x] Зафиксировать shared canonical-only collect contract для `claude-code`, `qwen-code`, `codex-code`
- [x] Убрать workspace-root fallback и legacy schema scavenging из collect runtime surface
- [x] Добавить explicit legacy precheck перед strict collect canonicalization
- [x] Не запускать live `step2.asis_docs` при unusable collect
- [x] Переводить terminal-less child/profile runs в `infra_incomplete_cycle`
- [ ] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить новый canonical live `regres fast` matrix и зафиксировать следующий triage cycle

### Non-goals
- [x] Не расширять schema/contracts под legacy aliases
- [x] Не добавлять permissive semantic compatibility repair
- [x] Не менять release taxonomy или canonical matrix harness command

### Approach
1) Усилить shared prompt/policy/baseline contract для collect и явно запретить legacy aliases.
2) Ограничить collect runtime surface до `write_root` + selected repo roots + explicit `read_context_roots`, включая collect cwd.
3) Добавить raw legacy detector в artifactquality и сохранить strict parse/validate path без semantic repair.
4) Исправить orchestrator skip path для unusable collect и harness reconciliation для stale `running`.
5) Синхронизировать spec/runbook/docs и затем прогнать DoD + новый live matrix cycle.

### Files expected to change
- `internal/runtime/*`
- `internal/artifactquality/*`
- `internal/orchestrator/*`
- `internal/workspace/*`
- `scripts/full-run-batch-5x2.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/*`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Collect prompts/contracts for all headless providers encode canonical-only vocabulary and legacy bans
- [ ] `step1.collect` no longer sees workspace-root by default and uses `write_root` as working directory
- [ ] Legacy collect payloads fail with explicit precheck diagnostics before strict parse
- [ ] `step2.asis_docs` is skipped when collect evidence is unusable
- [ ] Batch/matrix status files never remain silently `running` after child completion
- [ ] DoD passes and a new live `regres fast` cycle is launched for triage

### Risks
- Основной риск — пережать runtime surface и случайно отрезать нужные repo reads. Снижение риска: collect access остаётся на selected repo roots, а `step2/3` продолжают использовать explicit `read_context_roots`.

### Progress log
- 2026-04-22: implementation started for shared prompt cleanup, collect surface hardening, explicit legacy precheck, `step2` unusable-collect skip and batch/matrix stale-running reconciliation.

### Plan ID
EP-20260422-headless-legacy-residuals

### Context
После первого live rerun остались residual failure classes: collect semantic evidence всё ещё может дрейфовать в citation-only shape, findings verdict prompt недоописывает canonical metadata trio, `qwen-code` маскирует quota/auth как `runtime_contract_failed`, а matrix reconciliation не добивает stale `profile-status=running` по owner heartbeat.

### Goals (must have)
- [x] Дожать shared collect contract до обязательного `repo/path` внутри semantic provenance evidence
- [x] Добавить shared canonical contract/example для `validator-verdict.json`
- [x] Переклассифицировать `qwen-code` post-run quota/auth failures в `runner_unavailable`
- [x] Добавить durable child owner sidecar + matrix stale-profile reconciliation
- [ ] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить следующий live `regres fast` cycle и зафиксировать новый triage

### Non-goals
- [x] Не менять `schemas/*` и public contracts
- [x] Не добавлять semantic compatibility repair для legacy payloads
- [x] Не вводить provider-specific codex footer до завершения shared prompt wave

### Approach
1) Обновить shared artifact/prompt policy: collect evidence требует `repo/path`, findings contract задаёт canonical validator verdict example.
2) Сохранить strict legacy reject в `artifactquality` и расширить diagnostics на citation-only semantic evidence.
3) Исправить `qwen` post-run classification без ослабления artifact validation.
4) Добавить `batch-owner.env` heartbeat и matrix stale sweep поверх `profile-status/*.json`.
5) Синхронизировать docs, прогнать DoD и затем повторить live `regres fast`.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/*`
- `internal/workspace/*`
- `scripts/full-run-batch-5x2.sh`
- `scripts/full-run-batch-matrix.sh`
- `scripts/tests/*`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Collect examples/prompts больше не показывают citation-only semantic evidence и явно требуют `repo/path`
- [ ] Findings prompts/policies требуют `version/run_id/generated_at` в `validator-verdict.json`
- [ ] `qwen` quota/auth post-run failures классифицируются как `runner_unavailable`
- [ ] Matrix умеет перевести stale `profile-status=running` в `failed/infra_incomplete_cycle` по owner sidecar
- [ ] DoD проходит и запускается новый live rerun

### Risks
- Главный риск — переусилить stale reconciliation и ложно добивать живой профиль. Снижение риска: reconciliation использует `batch-owner.env pid + updated_at` и оставляет свежий live heartbeat нетронутым.

### Progress log
- 2026-04-22: started residual wave for canonical semantic evidence `repo/path`, validator verdict metadata trio, qwen post-run reclassification, and batch-owner-based stale profile reconciliation.
- 2026-04-22: closed residual literal gaps after implementation audit: added strict `step_contract:null` reject coverage, explicit owner-gap `PASS` prompt/policy coverage, entrypoint hint tests for root `CODEOWNERS`/`MAINTAINERS*`, and docs wording that current runtime isolation relies on temp-root layout + step-local `cwd` because provider-side hard sandbox is unavailable.
- 2026-04-22: fixed an additional qwen runtime bug found during acceptance audit: stall monitors now terminate the stuck provider process, draft/collect post-artifact stalls recover through artifact-only validation/repair, and pre-artifact stalls get one fresh-process retry before final failure classification.
- 2026-04-22: fresh clean bank acceptance rerun surfaced one more qwen residual: the pre-artifact fresh-process retry could still run silently for too long after the initial stall trigger; fixed by keeping a bounded pre-artifact sentinel on the second attempt, adding explicit `retry exhausted` diagnostics, and extending regression coverage around retry stall windows.
- 2026-04-22: a follow-up clean bank rerun showed that bounded qwen retry exhaustion now behaves correctly but still surfaced one semantic misclassification: a fully silent second collect attempt was landing as `runtime_contract_failed`; fixed by reclassifying silent pre-artifact retry exhaustion to `runner_unavailable`, while preserving `runtime_contract_failed` for retries that still emit provider output before stalling.
- 2026-04-22: the next clean bank rerun immediately exposed the remaining edge-case of the same classifier: fully silent retry exhaustion could resurface as `post_artifact` once the fresh process wrote a partial artifact and then froze again; widened the reclassification so any fully silent collect retry exhaustion now lands as `runner_unavailable`, regardless of whether the second attempt died in `pre_artifact` or `post_artifact`.

### Plan ID
EP-20260422-docflow-runtime-residuals

### Context
После второго live `regres fast` collect перестал быть главным blocker, но surfaced новые residual drift classes: `step2.asis_docs` мог писать loose draft manifest и читать sibling baseline artifacts, `step3.findings` ломался на расходящемся `document_id` mapping и semantic alias duplicates, owner-gap оставался release-blocking даже без технических validator issues, а batch classifier путал terminal `validator verdict is FAIL` с `runtime_contract_failed`.

### Goals (must have)
- [x] Зафиксировать strict canonical contract для `step2.asis_docs`
- [x] Убрать sibling baseline leakage через step-local cwd + separated temp roots
- [x] Нормализовать staged docflow document IDs и semantic repo/entity aliases до validator
- [x] Перевести owner-only residual `FAIL` в non-blocking `PASS` без потери findings/questions
- [x] Классифицировать terminal `validator verdict is FAIL` как `runtime_flow_failed`
- [x] Синхронизировать README/spec/runbook/architecture с новым поведением
- [x] Прогнать DoD (`make contracts`, `make test`, `make lint`, `make build`)
- [ ] Запустить новый canonical live `regres fast` matrix и зафиксировать следующий triage cycle

### Non-goals
- [x] Не менять `schemas/*` и `internal/contracts/*`
- [x] Не добавлять новые verdict states или provider-specific compatibility repair
- [x] Не ослаблять validator или collect schemas под legacy semantic payloads

### Approach
1) Усилить shared prompt/policy/runtime-manifest validation для `step2.asis_docs`.
2) Перевести non-collect runtime cwd на step-local roots и развести headless/baseline temp roots.
3) Нормализовать `document_id` mapping, `evidence.repo` identity и semantic alias duplicates в docflow assembly/model layer.
4) Добавить owner-gap-only verdict reconciliation и скорректировать batch/report failure classification.
5) Синхронизировать docs, прогнать DoD и затем повторить canonical live `regres fast`.

### Files expected to change
- `internal/artifactquality/*`
- `internal/runtime/*`
- `internal/runtimedrafts/*`
- `internal/orchestrator/*`
- `internal/model/*`
- `internal/workspace/*`
- `scripts/full-run-ai-advent.sh`
- `scripts/full-run-batch-5x2.sh`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/PLANS.md`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

### Acceptance criteria
- [ ] `step2.asis_docs` принимает только canonical draft manifest с required outputs и без legacy top-level fields
- [ ] non-collect runtime шаги больше не стартуют из workspace root и не видят sibling baseline workspace как лёгкий template source
- [ ] staged `citation-index.json` и `final-run-index.json` используют согласованный deterministic `document_id` mapping
- [ ] semantic assembly нормализует repo aliases и детерминированно дедуплицирует entity aliases до validator/promotion
- [ ] owner-gap-only residual больше не держит `validator-verdict = FAIL`, но signal остаётся visible в findings/questions
- [ ] terminal `validator verdict is FAIL` классифицируется как `runtime_flow_failed`
- [ ] DoD проходит и следующий live `regres fast` cycle запускается

### Risks
- Главный риск — переусилить semantic dedupe и случайно слить разные сущности. Снижение риска: canonical key включает type + normalized repo identity + normalized name + primary evidence path, а alias remap остаётся детерминированным и тестируется на bank-style duplicates.

### Progress log
- 2026-04-22: started runtime/docflow residual wave for strict `step2` manifest contract, isolated non-collect cwd/layout, deterministic staged document IDs, semantic alias folding, owner-gap downgrade, and validator-fail classifier correction.

### Plan ID
EP-20260421-cleanup-owner-followups

### Context
Safe cleanup уже внедрён: убраны доказуемые code/doc хвосты, сокращены дубли и ослаблены repo-policy проверки под canonical-source model. Остались только owner-gated решения, которые нельзя принимать автоматически без риска снести intentional discoverability/history surfaces.

### Goals (must have)
- [ ] Подтвердить, должны ли `internal/docsync` и `internal/scriptsmeta` оставаться test-only пакетами под `internal/*`
- [ ] Подтвердить, нужно ли одновременно хранить `docs/archive/PLANS_ARCHIVE_2026-04.md` и `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`
- [ ] Не смешать follow-up owner decision с текущим cleanup change set

### Non-goals
- [ ] Не переносить `internal/docsync` и `internal/scriptsmeta` в этом проходе
- [ ] Не удалять `docs/archive/PLANS_SNAPSHOT_2026-04-21.md` без явного подтверждения владельца

### Approach
1) Зафиксировать спорные места как owner-gated follow-up после завершённого cleanup.
2) Не менять code/doc layout дополнительно, пока владелец не подтвердит intent.
3) После подтверждения сделать отдельный малый change set на package placement/archive retention.

### Files expected to change
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Спорные cleanup-решения зафиксированы отдельно от внедрённого safe cleanup
- [ ] Активная секция не держит уже закрытый audit-plan

### Risks
- Главный риск — ошибочно снести intentional discoverability/history surfaces. Снижение риска: держать эти вопросы owner-gated и не менять их без явного решения.

### Progress log
- 2026-04-21: cleanup внедрён; спорные решения по test-only package placement и archive retention вынесены в отдельный follow-up.

### Plan ID
EP-20260421-codex-runtime-provider

### Context
Нужно расширить MVP runtime/provider surface и trusted-host live release gate новым headless provider `codex-code`, сохранив deterministic `fake` baseline, default fallback `claude-code`, 5-profile taxonomy и shard-plan invariant между `baseline` и `parallel-default`.

### Goals (must have)
- [x] Добавить `codex-code` в runtime/config contracts, CLI/API/workspace validation и runner factory
- [x] Реализовать artifact-only headless runner через `codex exec` без codex-specific retry/watchdog policy
- [x] Расширить live batch harness/reporting/release verdict до `qwen-code + claude-code + codex-code`
- [x] Синхронизировать schema/spec/docs/examples/ADR и live runbook
- [x] Обновить Go и Python/script tests под новый provider

### Non-goals
- [x] Не добавлять hosted mode
- [x] Не менять default provider с `claude-code`
- [x] Не переименовывать legacy `full-run-batch-5x2.sh`
- [x] Не добавлять wrapper-скрипты поверх canonical matrix harness

### Approach
1) Завершить runtime/config slice: provider enum/parser, runner factory, Codex runner, workspace/API validation и schema surface.
2) Расширить live harness/reporting/preflight и release strict gate без большого generic refactor.
3) Синхронизировать ADR/docs/examples/fixtures и пересчитать release catalog totals.
4) Прогнать целевые Go и Python/script tests, затем при возможности `make contracts`, `make test`, `make lint`, `make build`.

### Files expected to change
- `internal/runtime/*`
- `internal/workspace/*`
- `internal/api/*`
- `cmd/acp/*`
- `schemas/*`
- `scripts/full-run-batch-matrix.sh`
- `scripts/full-run-batch-5x2.sh`
- `scripts/frontend-live-e2e.sh`
- `scripts/write-batch-preflight.py`
- `scripts/e2e_batch_report.py`
- `scripts/tests/*`
- `docs/*`
- `examples/*`

### Acceptance criteria
- [x] `codex-code` accepted everywhere `claude-code|qwen-code` were previously required
- [x] Release-mode matrix expects all three providers and reports frontend/backend verdicts for Codex
- [x] Regression profiles remain qwen-only unless manually filtered otherwise
- [x] Docs/runbook/ADR/examples remain synchronized with schema and tests

### Risks
- Основной риск — оставить несовпадение между runtime surface и trusted-host live gate/reporting. Снижение риска: держать provider allowlists/order explicit, расширять report fields инкрементально (`frontend_codex_status`, `frontend_cancel_codex_status`, `runtimes.codex`) и проверять это script tests.

### Progress log
- 2026-04-21: started implementation; runtime provider enum/factory/Codex runner/tests partially wired, remaining work is harness/reporting/docs synchronization.
- 2026-04-21: runtime/config, harness/reporting, docs/examples/ADR and Go/Python test slices completed; `make contracts`, `make test`, `make lint`, `make build` passed on the implementation tree.
- 2026-04-21: post-implementation audit found residual runbook drift (`release` totals and frontend three-provider acceptance wording); synchronized the runbook and started trusted-host live preflight.
- 2026-04-21: repaired canonical `/tmp/provenarch-live-e2e` path checkouts to valid pinned-SHA git heads without changing curated matrices, created a detached clean verification worktree from the implementation snapshot, and launched manual `codex-code` regression smoke through `scripts/full-run-batch-matrix.sh`.
- 2026-04-21: reran DoD on an isolated git-backed snapshot of the current working tree and attempted canonical `release fast`; the new `codex-code` path advanced through `init.step1.collect` and materialized shard outputs inside the canonical harness, while overall release verification remained blocked by trusted-host provider issues outside this slice (`claude-code` returned quota/auth `403`, `qwen-code` failed live draft contract on `single-git_url`).

### Plan ID
EP-20260420-regres-small-live-triage

### Context
Нужно выполнить canonical live `regres fast` через `scripts/full-run-batch-matrix.sh` на trusted host, непрерывно мониторить статус прогонов и после завершения собрать детальный triage-отчёт по фактическим runtime/harness проблемам без изменения public contract.

### Goals (must have)
- [ ] Запустить canonical `regres fast` matrix slice из clean worktree с нужным provider filter
- [ ] Непрерывно мониторить matrix/batch progress до terminal state
- [ ] Собрать run artifacts, classifications и release/profile reports
- [ ] Зафиксировать продуктовые/runtime/harness баги и отделить их от operational noise
- [ ] Составить итоговый triage-отчёт: что сломалось, где именно, что нужно чинить дальше

### Non-goals
- [ ] Не менять matrix taxonomy, timeout profiles и public schemas во время самого прогона
- [ ] Не подменять canonical harness wrapper-скриптом или ad-hoc env overrides
- [ ] Не пытаться чинить live run до завершения triage цикла

### Approach
1) Проверить trusted-host prerequisites, pinned path checkouts и clean worktree.
2) Выполнить `regres fast` через `scripts/full-run-batch-matrix.sh` c canonical sweep set.
3) Во время выполнения регулярно читать batch/matrix logs, status files и intermediate reports.
4) После завершения разобрать terminal artifacts: `profile-runs.jsonl`, matrix status files, batch classifications, session summaries, release/profile verdicts и raw taskrun diagnostics.
5) Сформировать детальный triage-отчёт и список необходимых follow-up fixes.

### Files expected to change
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Matrix slice дошёл до terminal state (`passed|failed`) с сохранёнными batch/matrix breadcrumbs
- [ ] Есть собранный список фактических failure classes и их root cause
- [ ] Итоговый отчёт отделяет runtime/provider проблемы от harness/reporting проблем

### Risks
- Основной риск — long-running live provider hangs или внешние transport/API failures. Снижение риска: canonical watchdog/reporting уже встроены; мониторинг идёт по нескольким источникам (`driver log`, batch status, profile status, taskrun artifacts), а не по одному summary-файлу.

### Progress log
- 2026-04-20: prerequisites и pinned path SHA проверены, подготовлен trusted-host flow для live triage.

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
| 9 Q&A capability | done (beta boundary) | Workspace-backed QA service + read-only CLI `acp qa`; публичный endpoint остаётся follow-up |
| 10 Changelog compilers | done (beta baseline) | Iteration changelog materialization в `reports/changelog/*` |
| 11 `POST /api/qa/ask` | follow-up (post-beta) | Не входит в required beta surface |
| 12–13 | out of MVP | Вне текущего beta scope |
| 14 CI trigger mode | done (beta baseline) | CLI batch required, smoke/golden jobs без live network deps |
| 15 Domain/baseline pack hardening | done (beta baseline) | Baseline skills/prompts wired и versioned в workspace |
