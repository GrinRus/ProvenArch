# PLANS.md

ExecPlan помогает агентам доставлять многошаговые изменения надёжно.
Файл хранит шаблон, индекс текущей работы и незакрытые планы с явным следующим действием.
Завершённая инженерная работа и старые status/queue snapshots сохраняются в архиве.

Исторические design/research/audit документы перечислены в [индексе архива](archive/README.md).
Исторические и закрытые планы вынесены в архив:
- `docs/archive/PLANS_ARCHIVE_2026-04.md`
- `docs/archive/PLANS_ARCHIVE_2026-05.md`
- `docs/archive/PLANS_ARCHIVE_2026-06.md`
- `docs/archive/PLANS_ARCHIVE_2026-07.md`
- `docs/archive/PLANS_ARCHIVE_2026-08.md`
- [September reconciliation archive](archive/PLANS_ARCHIVE_2026-09.md)
- `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`

Канонический статус возможностей находится в [Canonical Stakeholder Matrix](STAKEHOLDER_DOC.md#0-canonical-stakeholder-matrix-source-of-truth).
Acceptance берётся из [BACKLOG](BACKLOG.md); выбор работы, её зависимости и незакрытые gates — из
индекса ниже. Наличие исторического плана или незакрытого checkbox само по себе не запускает работу.

## Когда использовать
Используйте ExecPlan, если:
- работа затрагивает несколько модулей, или
- ожидаемое время > 30–60 минут, или
- затрагиваются контракты/схемы.

---

## Шаблон ExecPlan

## EP-YYYYMMDD-<slug>

Status: active — <current work or outstanding dependency/review>.

Next action: <one concrete unresolved action and its stop condition>.

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

## Active plan index

`active` означает незакрытый инженерный scope, `blocked` — незакрытую зависимость/проверку,
`review` — оставшееся решение владельца, admin operation или merge acceptance. Старые progress logs
сохраняют факты своего времени: команды, модели, SHA и очереди из них не заменяют текущую spec,
runbook или разрешение на новый запуск. Локальная ревизия агентской разработки завершена:
[результаты и проверки](archive/PLANS_ARCHIVE_2026-09.md#ep-20260905-agent-development-revision).
Remediation program и release gates остаются отдельными scope и этой ревизией не запускаются.

| Plan | Status | Outstanding boundary |
| --- | --- | --- |
| [EP-20260905-approved-trash-cleanup](#ep-20260905-approved-trash-cleanup) | active | approved PR merges to main, followed by final revision and closeout |
| [EP-20260905-audit-remediation-program](#ep-20260905-audit-remediation-program) | active | REM-01/REM-02/REM-06/REM-07/REM-08/REM-09/REM-10 merged; REM-11 is the current independent P1 slice while REM-03B remains authorization-gated |
| [EP-20260811-task-attempt-contracts](#ep-20260811-task-attempt-contracts) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260811-task-first-ui](#ep-20260811-task-first-ui) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260812-task-first-live-evidence-alignment](#ep-20260812-task-first-live-evidence-alignment) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260811-weak-model-validation-authority](#ep-20260811-weak-model-validation-authority) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260805-live-runtime-safety-fixes](#ep-20260805-live-runtime-safety-fixes) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260805-visual-alignment-pass](#ep-20260805-visual-alignment-pass) | active | retained scope; reconcile current implementation before selecting a new slice |
| [EP-20260805-ui-bug-cleanup](#ep-20260805-ui-bug-cleanup) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260804-ui-truth-and-document-centric-flow](#ep-20260804-ui-truth-and-document-centric-flow) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260804-runtime-model-selection](#ep-20260804-runtime-model-selection) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260803-v0.1.13-unqualified-prerelease](#ep-20260803-v0113-unqualified-prerelease) | review | recorded owner/admin or merge acceptance remains open |
| [EP-20260803-ui-ux-hierarchy-onboarding](#ep-20260803-ui-ux-hierarchy-onboarding) | review | recorded owner/admin or merge acceptance remains open |
| [EP-20260803-outcome-architecture-recovery](#ep-20260803-outcome-architecture-recovery) | review | recorded owner/admin or merge acceptance remains open |
| [EP-20260729-open-source-readme](#ep-20260729-open-source-readme) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260801-r3-collect-root-claims-repair](#ep-20260801-r3-collect-root-claims-repair) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260801-r3-live-prompt-canonical-shapes](#ep-20260801-r3-live-prompt-canonical-shapes) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260801-r3-proposal-nested-section-validation](#ep-20260801-r3-proposal-nested-section-validation) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260801-r3-collect-task-identity-recovery](#ep-20260801-r3-collect-task-identity-recovery) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260801-r3-focused-repair-fresh-mutation](#ep-20260801-r3-focused-repair-fresh-mutation) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260729-r3-validator-evidence-advisory-boundary](#ep-20260729-r3-validator-evidence-advisory-boundary) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260729-r3-cross-shard-semantic-aliases](#ep-20260729-r3-cross-shard-semantic-aliases) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260729-r3-overview-evidence-gap-quality-signal](#ep-20260729-r3-overview-evidence-gap-quality-signal) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260728-r3-qwen-citation-closed-shape](#ep-20260728-r3-qwen-citation-closed-shape) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260728-r3-shutdown-test-barrier](#ep-20260728-r3-shutdown-test-barrier) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260728-r3-qwen-citation-binding](#ep-20260728-r3-qwen-citation-binding) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260728-r3-preflight-retry-test-quiescence](#ep-20260728-r3-preflight-retry-test-quiescence) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260728-r3-qwen-scope-array-contract](#ep-20260728-r3-qwen-scope-array-contract) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260722-post-implementation-trust-audit](#ep-20260722-post-implementation-trust-audit) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260719-epic18-targeted-architecture-home-repair](#ep-20260719-epic18-targeted-architecture-home-repair) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260717-epic18-r3-product-shell-live-gate](#ep-20260717-epic18-r3-product-shell-live-gate) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260623-karpathy-adoption-roadmap](#ep-20260623-karpathy-adoption-roadmap) | active | retained scope; reconcile current implementation before selecting a new slice |
| [EP-20260710-workspace-health-snapshot](#ep-20260710-workspace-health-snapshot) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260711-run-pinned-evidence-review](#ep-20260711-run-pinned-evidence-review) | active | retained scope; reconcile current implementation before selecting a new slice |
| [EP-20260712-evidence-backed-architecture-refresh](#ep-20260712-evidence-backed-architecture-refresh) | active | retained scope; reconcile current implementation before selecting a new slice |
| [EP-20260629-live-e2e-artifact-summary-finalization](#ep-20260629-live-e2e-artifact-summary-finalization) | active | retained scope; reconcile current implementation before selecting a new slice |
| [EP-20260616-live-e2e-recovery-rerun-loop](#ep-20260616-live-e2e-recovery-rerun-loop) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260608-medium-live-e2e-quality-ui](#ep-20260608-medium-live-e2e-quality-ui) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260608-live-artifact-quality-hardening](#ep-20260608-live-artifact-quality-hardening) | active | retained scope; reconcile current implementation before selecting a new slice |
| [EP-20260508-oss-readiness-hardening](#ep-20260508-oss-readiness-hardening) | review | recorded owner/admin or merge acceptance remains open |
| [EP-20260507-trusted-live-validation](#ep-20260507-trusted-live-validation) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260618-live-e2e-quality-loop](#ep-20260618-live-e2e-quality-loop) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260507-cleanup-owner-decisions](#ep-20260507-cleanup-owner-decisions) | review | recorded owner/admin or merge acceptance remains open |
| [EP-20260704-needs-review-to-excellent-no-repair-pressure](#ep-20260704-needs-review-to-excellent-no-repair-pressure) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260715-console-trust-shell](#ep-20260715-console-trust-shell) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260715-epic18-r3-composite-release](#ep-20260715-epic18-r3-composite-release) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260717-epic18-r3-ui-route-preflight](#ep-20260717-epic18-r3-ui-route-preflight) | active | retained scope; reconcile current implementation before selecting a new slice |
| [EP-20260718-epic18-r3-install-fixture-platform](#ep-20260718-epic18-r3-install-fixture-platform) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260719-epic18-r3-snapshot-live-gate-remediation](#ep-20260719-epic18-r3-snapshot-live-gate-remediation) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260728-r3-qwen-architecture-home-marker-contract](#ep-20260728-r3-qwen-architecture-home-marker-contract) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260728-r3-overview-placeholder-line-scope](#ep-20260728-r3-overview-placeholder-line-scope) | blocked | recorded validation or trusted qualification remains open |
| [EP-20260802-r3-qwen-collect-process-provenance-contract](#ep-20260802-r3-qwen-collect-process-provenance-contract) | blocked | recorded validation or trusted qualification remains open |

Original unnumbered live follow-ups remain in [Retained live follow-up notes](#retained-live-follow-up-notes).
They are retained qualification evidence, not an additional ready-to-run queue.
Completed plan bodies and the obsolete operational/queue mirrors are preserved in the
[September archive](archive/PLANS_ARCHIVE_2026-09.md). Owner/admin decisions, incomplete DoD and
trusted release qualification remain here; this reconciliation does not close REM-25.

## Active Plans


## EP-20260905-approved-trash-cleanup

Status: active — cleanup implemented and validated; owner authorized PR merges to main and final revision.

Next action: Merge the verified stacked PRs [#270](https://github.com/GrinRus/ProvenArch/pull/270) → [#271](https://github.com/GrinRus/ProvenArch/pull/271) → [#272](https://github.com/GrinRus/ProvenArch/pull/272) → [#273](https://github.com/GrinRus/ProvenArch/pull/273). The owner explicitly authorized push and merge for every cleanup iteration; REM-02 golden-selector repair is already merged in [#274](https://github.com/GrinRus/ProvenArch/pull/274).

### Context
The owner approved removal of confirmed dead code and its isolated tests, migration of useful test-only checks, CSS cleanup, historical-document archival and broken-reference repairs. Supported compatibility is removed only when there is no remaining consumer or contract. Golden coverage must remain executable; an empty selector is not a useful gate.

### Goals (must have)
- [x] Preserve the useful golden gate from the separately merged REM-02 repair; avoid a duplicate runner.
- [x] Remove confirmed backend and frontend dead groups; preserve useful regression coverage on live paths.
- [x] Remove orphan CSS selectors and verify desktop, tablet and phone rendering.
- [x] Archive remaining completed/historical documents with repaired links and explicit expired evidence; preserve open release/owner obligations.
- [x] Complete deterministic DoD, independent review and small PR delivery.
- [ ] Merge each cleanup iteration through a checked PR into main.
- [ ] Complete the final main revision and deliver the consolidated disposition report.

### Non-goals
No runtime prompt/model defaults, schemas, supported public/legacy APIs, release matrices, readable fixture deduplication, waiver deletion, accepted semantic primitive/token removal, or live provider execution. The manually usable curated Bank manifest remains until manual-use ownership is known. This task does not start or close unrelated remediation rows.

### Acceptance criteria
- Removed symbols have no live consumers, or their tests are migrated to the live surface.
- Existing deterministic behavior and source fixture bytes are preserved.
- `make contracts`, `make test`, `make lint`, `make build` pass; UI changes also pass mock/browser and embedded parity/determinism checks.
- Each iteration uses a bounded PR with validation evidence and reversible commits, successful required CI and merge into main. No direct main writes or release.
- At least three complete audit passes are recorded; a final pass after the last merge finds no new confirmed in-scope garbage. Unproven candidates remain explicitly classified.

### Dependencies and ownership
Initial base was `origin/main` `6dbbed79`; the completion recheck integrates `b45a4d41` after PRs #274–#275. The separate stabilization checkout has local changes in `docs/ARCHITECTURE.md`, semantic/docflow implementation/tests and `ui/src/App.test.tsx`; its worktree and changes are not touched. Cleanup uses isolated worktrees and avoids those runtime behavior changes. Documentation corrections affect separate stale descriptions only. The already merged guidance/archive work in PR #268 and release-verifier work in PR #269 are rechecked rather than repeated.

### Progress log
- 2026-09-05: Owner approved the audit cleanup and small PR delivery. Fresh-main comparison found several documentation/archival findings already fixed by PR #268. Created isolated cleanup branches; source audit registers retained outside the repository for revalidation.
- 2026-09-05: Detected concurrent REM-02 implementation on `codex/rem-02-golden-selection`; excluded the overlapping golden experiment from this series. Existing readable fixture history remains unchanged. Backend removal and UI source/CSS changes passed focused tests and independent backend review.

- 2026-09-05: Completed 46 Go declaration removals and one Python wrapper removal; migrated effective-verdict, Service history and incomplete-report regressions to live paths. Independent backend and UI reviews found no regression. Removed retired frontend modules/mutation branches while preserving workspace identity, Task admission and read-only diagnostics. Removed 84 CSS classes; all 1,952 retained selector branches preserve declarations/context/order. Code, styles and tests have 3,152 fewer net lines.
- 2026-09-05: Archived 11 historical documents with provenance and repaired references. Current UI and effective-verdict prose now match implementation; stale architect-summary determinism claims were replaced with bounded existing regression evidence. Documentation checks covered 358 local links and 146 fragments; independent changed-document review found no broken moved targets. Open plans, release waivers, curated inputs, readable golden history and accepted semantic primitives/tokens remain intact.
- 2026-09-05: Full integration check found one obsolete Go docsync assertion that read the removed BaselineEditorsPanel. Removed only that copy test in the UI PR; the actual runtime prompt-pack boundary test remains. Final deterministic DoD passed: `make contracts`, `make test` (all Go packages, 304 Python tests, 245 UI tests), `make lint`, `make build`. UI determinism and embedded freshness checks passed; the final rebuild produced no diff. No live provider run was used.
- 2026-09-05: Browser QA passed all 10 source/CSS mock configurations across desktop/tablet/mobile after an ENOSPC retry. Fourteen of 18 screenshot pairs were exact; a button state, dynamic elapsed times and one unstable full-page capture bound the remaining comparisons. No universal pixel-parity claim is made. All 11 CI checks passed for the implementation PR heads; this final plan-only evidence update receives a fresh CI run. Temporary cleanup worktrees were removed after retaining commits, PRs and local audit evidence. Stop condition reached at PR delivery; no merge or release performed.

- 2026-09-05: Completion recheck confirmed all four reviewed PR heads had passed CI, but the newer main from #274–#275 conflicted with the cleanup plan-index insertion. Integrated `b45a4d41` through the stack and retained both independent plan entries/statuses. Removed two obsolete golden export/update commands that selected a deleted test; documented the boundary between stored readable digest checks and fresh pipeline generation. Corrected the REM-02 test-set count to six without changing its acceptance/status.
- 2026-09-05: Independent recheck of the unchanged cleanup implementation found no live callers among 102 reviewed declaration names/22 removed or moved paths; 393 local Markdown references including 148 anchors passed. Rechecked the integrated result with docsync, all four golden runner contract tests, the actual workflow command (six tests) and readable fixture verification (90 artifacts): all passed. Runtime/UI source and embedded assets are unchanged from the completed cleanup DoD; updated PR heads receive fresh CI checks.

- 2026-09-05: Owner extended completion through PR merge into main for every iteration. Integrated independently merged retention fix #276, preserving both plan entries and runtime changes. The initial three audit passes remain evidenced by code/document/reference inventories, caller/contract checks, and a restarted challenge of test-only islands, historical documents and curated inputs; final post-merge revision remains pending.

## EP-20260905-audit-remediation-program

Status: active — REM-01, REM-02, REM-06, REM-07, REM-08, REM-09 and REM-10 merged; REM-11 is the current independent P1 slice while REM-03B remains authorization-gated.

Next action: Reproduce and deliver REM-11 with a bounded Git change-inventory path,
then repeat stabilization/dependency checks before selecting the next ready row; keep release status
explicitly blocked until REM-03B is authorized and applied with before/after/rollback evidence.
REM-25 remains blocked by REM-03..24.

### Context

Follow-up аудит от 2026-09-05 оценил не только локальные дефекты, но и соответствие реализации
local-first mission, Task-first product flow и release claims. Базовая точка аудита — `main` на
`c6d46ce13775349fb9b038ba4fcbf39c039efce3`. Программа продолжает, но не переоткрывает завершённый
Epic 19: каждая проблема должна быть повторно доказана на актуальном `origin/main`, закрыта отдельным
reviewable slice и защищена regression evidence.

Параллельно в соседней задаче `Проверь UI и UX проекта` идёт stabilization lane: semantic
alias/collision handling, document-flow assembly и trusted Codex live qualification. На момент
создания плана соседняя задача активна и имеет незавершённые изменения в отдельном checkout. Поэтому
очередь ниже является ordered-ready queue: на каждой итерации берётся первая незакрытая задача, для
которой сняты зависимости и отсутствует пересечение с актуальным stabilization scope.

Этот planning slice не исправляет продуктовый код и не объявляет release readiness. Реализация
начинается только после отдельного запуска remediation goal владельцем проекта.

### Goals (must have)

- [ ] Закрыть все подтверждённые P0/P1 findings и явно принять, перенести или закрыть evidence для
      каждого P2 finding.
- [ ] Проводить каждую code/config доработку отдельным минимальным PR с regression test,
      независимым review и зелёными required checks; внешние admin operations подтверждать
      отдельным audit trail и rollback evidence.
- [ ] Не дублировать и не перетирать работу соседнего stabilization lane; после его merge повторно
      проверить затронутые findings на свежем `origin/main`.
- [ ] Восстановить достоверность release/golden gates до низкорисковых refactoring и UX polish.
- [ ] Довести Task-first journey `Setup -> Task -> Attempt -> Review -> Publish` до согласованного
      поведения API, UI, persistence и документации.
- [ ] Завершить программу полным deterministic DoD и отдельной trusted-machine qualification по
      canonical runbook без превращения live providers в required CI.

### Non-goals

- [ ] Не добавлять hosted mode, security/compliance enforcement или новых headless providers.
- [ ] Не менять public contracts, schemas или canonical release matrices без отдельного
      schema/spec-first решения и соответствующих guardians.
- [ ] Не переписывать большие подсистемы до появления regression coverage для исправляемого
      поведения.
- [ ] Не использовать текущий main checkout соседней задачи для remediation commits, reset, pull
      или cleanup её незавершённых файлов.
- [ ] Не обновлять canonical stakeholder status до фактического merge соответствующих slices.
- [ ] Не считать diagnostic/non-release live run доказательством `RELEASE READY`.

### Dependencies / parallel stabilization boundary

Текущий stabilization-owned set фиксируется как стартовый snapshot, а не вечный список:

- `docs/ARCHITECTURE.md`
- `internal/artifactquality/semantic.go`
- `internal/artifactquality/semantic_test.go`
- `internal/orchestrator/docflow.go`
- `internal/orchestrator/docflow_test.go`
- active Codex medium live matrix и связанные с ней runtime diagnostics

До merge соседней задачи эти файлы не изменяются remediation slices. Для близких путей
`internal/artifactquality/**`, `internal/orchestrator/docflow*`, architecture behavior docs и live
harness сначала проверяется semantic overlap, даже если имя файла не совпадает. Если overlap есть,
slice остаётся `blocked-by-stabilization`: реализация не начинается, а следующей выбирается первая
ready задача.

Перед выбором и перед merge каждого slice исполнитель обязан:

1. Получить статус и последние material changes соседней задачи, а не ориентироваться только на этот
   стартовый snapshot.
2. Выполнить `git fetch origin main --prune`, зафиксировать base SHA и сравнить изменённые пути с
   candidate scope.
3. После merge stabilization lane обновиться от `origin/main`, повторно воспроизвести finding и
   удалить из очереди уже закрытые либо переформулировать изменившиеся acceptance criteria.
4. Не запускать конкурирующую full/live matrix на том же trusted host. Узкие deterministic checks
   допустимы; полный DoD планируется так, чтобы не искажать live evidence соседней задачи.

### Ordered remediation queue

Статусы `ready` и `blocked-by-stabilization` перепроверяются на каждой итерации. Если
stabilization-sensitive P1 становится ready, он возвращается выше оставшихся P1/P2 задач независимо
от того, какая ready задача была временно пропущена.

| Order | ID | Priority | Result / acceptance boundary | Depends on | Initial readiness |
| --- | --- | --- | --- | --- | --- |
| 1 | REM-01 | P0 | Release verifier принимает только полное, свежее и связанное с release tag/source SHA evidence; stale, fabricated, incomplete, mismatched assessment и over-broad waiver fixtures fail closed. | none | merged in PR #269 |
| 2 | REM-02 | P1 | Golden workflow доказывает запуск ожидаемых test cases и падает при rename/removal или zero-match вместо успешного `[no tests to run]`. | REM-01 | merged in PR #274 |
| 3 | REM-03 | P1 | `REM-03A` versioned evidence/check PR проверяет expected required checks, ruleset и owner-waiver governance; `REM-03B` — отдельная явно авторизованная admin-only operation с before/after/rollback evidence. До обеих частей обход release truth не считается закрытым. | REM-01, REM-02; explicit authority for REM-03B | blocked-by-dependency; REM-03B authorization-gated |
| 4 | REM-04 | P1 | Runtime write audit становится deny-by-default: разрешённые roots заданы явно, unknown/unclassified writes и audit failure блокируют promotion/release evidence. | stabilization merge, reproduce finding, REM-01 | blocked-by-stabilization |
| 5 | REM-05 | P1 | Root-bounded file operations и restore/promotion защищены от symlink swap и check/use races; adversarial filesystem tests не выходят за workspace. | stabilization merge, REM-04 | blocked-by-stabilization |
| 6 | REM-06 | P1 | Retention никогда не удаляет active/queued run и его Task/Attempt evidence; restart/pressure tests подтверждают lifecycle invariant. | REM-01..03, либо REM-03 admin blocker явно сохраняет release-blocked status | ready under explicit REM-03B release blocker; current slice |
| 7 | REM-07 | P1 | Task/run watchers завершаются при shutdown/cancel, не переживают server lifecycle и не создают goroutine/race leak. | REM-06 | merged in PR #279 |
| 8 | REM-08 | P1 | Queued Attempt сохраняет immutable admission context и после restart исполняется либо fail-closed с понятной диагностикой, без silent context drift. | REM-06, REM-07 | merged in PR #281 |
| 9 | REM-09 | P1 | Remote/moving Git ref резолвится в immutable commit identity; изменение branch между validate/run обнаруживается, а evidence остаётся воспроизводимым. | REM-01..03, либо REM-03 admin blocker явно сохраняет release-blocked status | merged in PR #283 |
| 10 | REM-10 | P1 | Publication и Task/Attempt linkage имеют recoverable atomic boundary; частичный сбой не оставляет ложный `Published` или потерянный commit. | REM-09 | merged in PR #285 |
| 11 | REM-11 | P1 | Git change inventory убирает subprocess-per-file path; benchmark на representative 275-file change set имеет заданный budget и не меняет semantics. | REM-09 | ready; next independent slice |
| 12 | REM-12 | P1 | Runner/provider identity и provenance отражают фактически выполненный adapter/model, без fallback mislabeling. | REM-08 | blocked-by-dependency |
| 13 | REM-13 | P1 | Task composer и admission передают полный scope/runner contract; UI summary, API snapshot и runtime execution совпадают. | REM-12 | blocked-by-dependency |
| 14 | REM-14 | P1 | Edit/retry/rerun semantics различены: immutable Attempt не мутируется, новый Attempt наследует только явно разрешённые Task values. | REM-13 | blocked-by-dependency |
| 15 | REM-15 | P1 | Create/admit/queue transitions атомарны и честно отображаются в UI; ошибка admission не создаёт phantom active Task/Attempt. | REM-13, REM-14 | blocked-by-dependency |
| 16 | REM-16 | P1 | Architecture/Setup copy, route handoff и docs описывают один фактический Task-first flow без legacy primary-path claims. | stabilization merge, REM-12..15 | blocked-by-stabilization-and-dependency |
| 17 | REM-17 | P1 | Publish action доступен только для exact current Attempt, проверенного inventory fingerprint и свежего review evidence; stale UI state fail closed. | REM-10, REM-13..15 | blocked-by-dependency |
| 18 | REM-18 | P2 | Route/workspace changes отменяют или игнорируют устаревшие async responses; component tests покрывают out-of-order success/error. | REM-15, REM-17 | blocked-by-dependency |
| 19 | REM-19 | P2 | Polling имеет единый bounded lifecycle, backoff и visibility/offline behavior без дублированных timers и бесконечного request churn. | REM-18 | blocked-by-dependency |
| 20 | REM-20 | P2 | User drafts имеют явную persistence/recovery policy; navigation, refresh, failed save и workspace switch не приводят к silent data loss. | REM-18 | blocked-by-dependency |
| 21 | REM-21 | P2 | Keyboard/focus, landmarks, labels, contrast и reduced-motion проходят automated checks и ручной smoke ключевого journey. | REM-18..20 | blocked-by-dependency |
| 22 | REM-22 | P2 | Backend hotspots декомпозированы только после behavior locks; boundaries уменьшают coupling без изменения artifact semantics. | stabilization merge, REM-04..08 | blocked-by-stabilization-and-dependency |
| 23 | REM-23 | P2 | UI hotspots разделены по data/state/view seams, общие states унифицированы, а route-level regression suite остаётся зелёной. | REM-17..21 | blocked-by-dependency |
| 24 | REM-24 | P2 | Wall-clock sleeps/flaky waits заменены deterministic clocks/events; повторные focused runs не дают flakes. | stabilization merge, REM-06..08, REM-22 | blocked-by-stabilization-and-dependency |
| 25 | REM-25 | P2 | Specs, architecture, testing strategy, stakeholder mirror, examples и active/archive plans синхронизированы с фактом; дублированные stale claims удалены. | REM-01..24 resolved | blocked-by-program |

### Slice definition of ready

Задача готова к реализации, только если одновременно выполнено следующее:

- finding воспроизводится на актуальном `origin/main` и записан как observable failure/invariant;
- прочитаны релевантные `schemas/*`, `docs/spec/*`, architecture и tests согласно source priority;
- candidate paths не принадлежат активному stabilization lane и не имеют semantic overlap;
- определены goal, non-goals, acceptance, regression test, rollback и stop condition;
- code/config slice помещается в один reviewable PR без unrelated cleanup и новых production
  dependencies. Для admin-only `REM-03B` вместо PR заранее фиксируются exact setting delta,
  authorization, before/after evidence и rollback command.

Если finding больше не воспроизводится после соседнего merge, задача закрывается evidence-комментарием
в progress log, а не повторной реализацией того же исправления.

### Iteration protocol

Для каждого следующего пункта очереди выполняется один и тот же цикл:

Admin-only `REM-03B` использует те же research, plan, review, evidence и record gates, но не создаёт
фиктивную branch/commit/PR: GitHub setting применяется только после явной авторизации и проверки
rollback.

1. **Select.** Обновить статус соседней задачи и `origin/main`; просканировать ordered queue сверху и
   взять первую ready задачу.
2. **Research.** Детально прочитать относящиеся specs, code paths, tests, history и актуальный соседний
   diff; воспроизвести дефект либо измерить baseline.
3. **Plan.** Добавить/уточнить отдельный ExecPlan slice: goal, non-goals, acceptance, affected paths,
   regression strategy, risks, rollback и stop condition.
4. **Isolate.** До изменений создать свежую `codex/<slice>` branch в отдельном worktree от точного
   `origin/main` SHA. Это намеренно выполняется до первого commit, чтобы не переносить незакоммиченные
   изменения между ветками и не задеть соседний main checkout.
5. **Implement and verify.** Реализовать минимальный slice, сначала запустить узкие checks, затем
   требуемый deterministic DoD. Не расширять scope найденными попутно P2/P3 проблемами.
6. **Commit and review.** Сделать содержательный implementation commit, провести независимое review
   correctness/contracts/tests/maintainability; внести fixes отдельным commit только если fixes
   действительно нужны. Empty review commit не создаётся.
7. **Refresh and deliver.** Ещё раз получить соседний статус и `origin/main`, разрешить drift/overlap,
   push branch, открыть PR, дождаться required checks, исправить замечания и merge только зелёный PR.
8. **Rebase the loop.** После merge выполнить fetch `origin/main` и проверить merge SHA. Если main
   checkout занят/dirty в соседней задаче, не делать pull в нём: следующий worktree создаётся прямо
   от обновлённого `origin/main`.
9. **Record and repeat.** Записать исходный `origin/main` SHA, stabilization status/revision и owned
   paths snapshot, branch, PR URL, merge SHA, review verdict и выполненные checks; затем вернуться к
   шагу 1. Программа останавливается на blocker только после фиксации evidence и минимального
   действия для разблокировки.

### Approach

1. Сначала восстановить truthfulness автоматических gates (`REM-01..03`), чтобы последующие PR и
   release claims опирались на исполняемые проверки.
2. По мере готовности немедленно поднять stabilization-sensitive runtime safety (`REM-04..05`) выше
   оставшихся P1; до этого продолжать независимый lifecycle/state slice (`REM-06..08`).
3. Закрыть Git/publication correctness/performance (`REM-09..11`) и Task-first contract gaps
   (`REM-12..15`) до UI polish.
4. Свести publish/product-flow state machine (`REM-16..21`) и лишь затем выполнять структурные
   backend/UI refactors (`REM-22..24`).
5. Синхронизировать documentation/tracker surfaces (`REM-25`), выполнить полный deterministic DoD и
   только после этого провести canonical trusted qualification через `acp-e2e-live-gate` и
   `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.

### Files expected to change

- Release/golden evidence: `.github/workflows/{golden,release}.yml`, `scripts/verify-release-*.py`,
  `scripts/tests/*release*`, repository ruleset evidence.
- Runtime/filesystem/lifecycle: `internal/fs`, `internal/runtime`, `internal/orchestrator`,
  `internal/api`, Task/run registries and focused tests.
- Git/publication: Git resolver/inventory/publish services, recovery metadata and API tests.
- Task-first/UI: Task/Attempt contracts and routes, `ui/src/**`, component and Playwright tests.
- Closure docs: relevant `docs/spec/*`, `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md`,
  `docs/STAKEHOLDER_DOC.md`, this plan/archive and examples.

Exact files are re-derived per slice. Listing a path here does not grant a parallel slice ownership
over the stabilization-owned set.

### Acceptance criteria

- [ ] Каждая P0/P1 REM-задача имеет final state `merged` или `closed by current-main evidence`;
      иначе программа и release status остаются явно blocked. Для P2 дополнительно допустимо
      `explicitly accepted/deferred` с owner/rationale; молча пропущенных findings нет.
- [ ] Для исправленных P0/P1 существует negative regression test, который падал бы на audit
      baseline.
- [ ] Каждый merged slice основан на свежем `origin/main`, не содержит чужих stabilization changes и
      прошёл independent diff review.
- [ ] На каждом завершённом behavior slice проходят `make contracts`, `make test`, `make lint` и
      `make build` с pinned toolchain; scope-specific checks сохранены в PR evidence.
- [ ] Required CI не имеет zero-test success и не зависит от live network/provider binaries.
- [ ] Финальный Task-first smoke подтверждает Setup, create/admit, progress/recovery, review и
      publish boundaries для desktop/mobile и offline/error states.
- [ ] Trusted live gate запускается только canonical harness/profile taxonomy; fresh verifier-backed
      evidence связано с точным qualified SHA, а diagnostic runs не повышают release status.
- [ ] После финального merge `origin/main` повторно проверен, stakeholder/docs синхронизированы, а
      незакрытые residual risks перечислены явно.

### Risks

- Соседний stabilization merge может закрыть или изменить finding. Mitigation: reproduce-after-merge
  gate и запрет механического применения старого patch.
- Два worktree могут конкурировать за CPU/provider/session resources. Mitigation: не запускать
  параллельные full/live gates; deterministic checks ограничивать выбранным slice до финального DoD.
- Большие cross-cutting fixes легко превратить в unreviewable PR. Mitigation: один invariant и один
  rollback boundary на slice; refactor следует после behavior lock.
- GitHub settings и trusted qualification зависят от внешнего состояния. Mitigation: отделять code
  merge от admin/live evidence и не помечать задачу закрытой без фактической проверки.
- Existing docs содержат исторические claims. Mitigation: не переписывать backlog/history в каждом
  PR; выполнить единую fact-based reconciliation в `REM-25`.

### Progress log

### REM-01 slice plan (merged)

**Goal.** Закрыть P0 false-green boundary вокруг release evidence: verifier должен принимать только
release verdict, сформированный текущим deterministic matrix generator, с полным profile/sweep/provider
contract, свежим timestamp и source SHA квалификации, который является предком проверяемого tag commit. Companion verdict
Markdown/profile matrix и оба accepted SWE assessment должны быть рядом с JSON и взаимно согласованы.

**Non-goals.** Не запускать live providers, не менять canonical release matrices, не менять product
API/schema/runtime semantics, не принимать diagnostic/non-release results и не ослаблять owner-waiver
policy. Waiver остаётся отдельным `UNQUALIFIED PRERELEASE` escape hatch, но принимает только exact
tag-scoped filename/payload и не может быть использован как `RELEASE READY` evidence.

**Affected paths.** `scripts/full-run-batch-matrix.sh`, `scripts/verify-release-verdict.py`,
`scripts/verify-release-owner-waiver.py`, `.github/workflows/release.yml`, matching release tests,
`docs/RELEASE_LIVE_E2E_RUNBOOK.md`, `docs/TESTING_STRATEGY.md` и эта plan запись. Stabilization-owned
paths не затрагиваются.

**Implementation boundary.** Generator writes `evidence_schema_version`, `source_sha`, clean-tree
provenance, generator identity and a complete artifact manifest for release verdicts (including
operational-preflight failures). Release planning and verifier both enforce the canonical execution
profiles (`baseline=sequential/1/best_effort/heuristics`, `parallel-default=parallel/4/best_effort/heuristics`),
the exact record/profile/sweep/provider cardinality, zero runtime-flow issue counters, zero provider
budget exhaustion, and canonical Provider Matrix/Run Details evidence. It rejects artifact-quality
findings that producer would promote to a hard failure, release-blocking per-run issue tokens, and
source-tree/SHA drift detected at final aggregation, while preserving the producer's
`artifact_quality_status=needs_review` analysis signal when no hard artifact failure is present and
leaving final acceptance to the required SWE assessment. Verifier validates required non-empty artifact files with SHA-256
digests and cross-file identity, rejects stale/future evidence using a bounded age, and when invoked by
tag workflow requires the payload source SHA to be an ancestor of the exact commit resolved by
`GITHUB_SHA`/`refs/tags/<tag>`. Workflow passes tag/source context explicitly and rejects ambiguous
evidence-mode configuration. Waiver verifier additionally enforces the exact tracked
`reports/release_owner_waiver_<tag>.json` target and rejects unknown waiver fields/requirements.

**Regression strategy.** Add provider-free fixtures for missing provenance, stale/future timestamps,
source/tag mismatch, noncanonical sweep execution, non-zero runtime-flow issue counters, incomplete/
duplicate records, missing or mismatched companion files, fabricated minimal payloads, contradictory
Provider Matrix/Run Details tables, release-blocking per-run issues, mid-run source mutation,
over-broad waiver filenames/fields and valid generated evidence.
Keep matrix driver tests asserting generated release evidence carries provenance; run focused Python
suites first, then `make contracts`, `make test`, `make lint`, `make build`.

**Rollback / stop condition.** Revert the single PR if the verifier cannot distinguish a generated
release fixture from any negative fixture without changing matrix execution behavior. Stop before PR
delivery if the active stabilization task changes release-owned paths or if tag/source semantics remain
ambiguous; record the exact blocker and do not substitute a weaker freshness check.

- 2026-09-05: Создан follow-up remediation ExecPlan на baseline `c6d46ce`; соседняя задача
  `Проверь UI и UX проекта` подтверждена active во время Codex medium live matrix. Зафиксированы
  owned paths, dynamic readiness и isolated-worktree protocol. `REM-01` запущен в отдельной ветке;
  stabilization-owned paths остаются нетронутыми.
- 2026-09-05: `REM-01` реализован от свежего `origin/main` `91509c08f13c5461bf5ed6a0b209efb1a38243c4`;
  release evidence теперь fail-closed по provenance, canonical execution map, runtime/provider
  counters, artifact digests и Provider Matrix/Run Details. Полные focused suites зелёные
  (`verify_release_verdict_test` 24/24, `matrix_release_contract_test` 41/41); contracts, lint и
  build также прошли. Перед delivery повторно проверен active stabilization lane; его owned paths
  не изменялись.
- 2026-09-05: Соседний stabilization lane завершил текущую попытку в статусе blocked из-за
  внешней нехватки диска/доступности Claude; его изменения остаются незакоммиченными в отдельном
  checkout. Поэтому `REM-01` не включает semantic/docflow paths и не считает соседний результат
  merge-ready доказательством закрытия stabilization findings.
- 2026-09-05: Финальный review выявил и закрыл TOCTOU между началом matrix-run и итоговой агрегацией,
  а также fail-open parsing per-run `issues`; verifier теперь использует явный allowlist допустимых
  diagnostic analysis/recovery signals и блокирует неизвестные либо runtime/contract/artifact/
  reliability issues. Добавлены mid-run mutation и allowlist/forbidden-token regression tests.
- 2026-09-05: PR #269 (`hardening: close release evidence false greens`) прошёл все required checks,
  был squash-merged в `main` как `6dbbed79d95e1f3f1673c9a882a48611945aa242`; после merge выполнен
  `git fetch origin main --prune`, рабочее дерево чистое.

### REM-02 slice plan (merged)

**Goal.** Закрыть P1 false-green boundary golden workflow: CI должен доказуемо обнаруживать каждый
ожидаемый deterministic test case и завершаться ошибкой, если тест переименован, удалён, пропущен
или фактический запуск вернул успешный пакетный результат без тестов (`[no tests to run]`).

**Finding / baseline.** На свежем `origin/main` `6dbbed79` текущий workflow обращается к пяти старым
`TestScenario*` именам, удалённым вместе с прежним runtime contract. Команда workflow проходит с
нулевым покрытием:
`./scripts/run-go.sh test ./internal/orchestrator -run 'TestScenarioFixturesDeterministicInitPipeline|...' -count=1`
возвращает `ok ... [no tests to run]`. Это наблюдаемая false-green регрессия, а не предположение по
тексту workflow.

**Non-goals.** Не восстанавливать удалённый legacy contract, не менять product/runtime semantics,
scenario fixture outputs, canonical release matrix или stabilization-owned semantic/docflow paths;
не превращать live provider или network execution в required CI.

**Affected paths.** `.github/workflows/golden.yml`, `scripts/run-golden-tests.sh`,
`internal/orchestrator/golden_fixture_test.go`, `scripts/tests/golden_workflow_contract_test.py`,
`docs/TESTING_STRATEGY.md` и эта plan-запись.
Актуальный deterministic test set выбран из существующих orchestrator tests вне stabilization-owned
файлов: snapshot promotion, run/refresh persistence, deterministic progress и materialization.

**Implementation boundary.** Golden surface дополнительно проверяет, что три tracked scenario
snapshots имеют валидные SHA-256 и все перечисленные readable outputs существуют; это возвращает
проверяемую fixture-семантику без восстановления удалённого runtime contract. Новый runner сначала
получает compiled test list через pinned repository
Go wrapper и требует exact presence каждого переданного имени. Затем он выполняет anchored `-run`
selection с `-json` и требует top-level `pass` event для каждого имени; `skip`, `fail`, zero-match,
duplicate или malformed test name блокируют job. Golden workflow передаёт один явный список из шести
актуальных deterministic tests, поэтому rename/removal не может тихо превратиться в зелёный check.

**Regression strategy.** Contract tests проверяют workflow и runner с provider-free fake Go command:
valid list/pass проходит; renamed/removed test не проходит list preflight; package-only/zero-test JSON
не проходит pass-event gate. Реальный runner запускается на canonical six-test set. Узкие проверки
дополняются `make verify-agent-guidance`, `make contracts`, `make test`, `make lint` и `make build`.

**Rollback / stop condition.** Откатить PR, если runner не отличает valid execution от zero-match или
если выбранный набор начинает пересекаться с актуальным stabilization diff. Не возвращать старые
удалённые имена только ради зелёного workflow; при semantic overlap остановить slice и перепроверить
очередь после stabilization merge.

- 2026-09-05: Перед началом REM-02 зафиксирован свежий base `origin/main`
  `6dbbed79d95e1f3f1673c9a882a48611945aa242`; соседний stabilization lane — idle/blocked, revision 29,
  с незакоммиченными изменениями в отдельном checkout. Candidate scope с ним не пересекается.
- 2026-09-05: Реализованы fail-closed list/JSON pass gates, шесть deterministic golden checks и
  provider-free contract tests для valid, renamed/removed и zero-test случаев. Узкие checks, docs-sync,
  contracts, lint и build прошли; полный Go rerun дал два pre-existing load-sensitive lifecycle
  timeout-а, после чего оба targeted rerun прошли.
- 2026-09-05: Полный Python discovery (308) и UI suite были запущены; один unrelated Claude probe
  classification test и два UI timeout под общей нагрузкой дали flake, каждый изолированный rerun
  прошёл. Эти внешние flakes не затрагивают REM-02 paths; required CI остаётся финальным gate.
- 2026-09-05: PR #274 (`ci: fail closed on golden test selection`) прошёл повторный required CI после
  review fix и был squash-merged в `main` как `2c382a235073e9a364efbecc11b7fe0ed07ac225`; после
  merge выполнен `git fetch origin main --prune`, рабочее дерево чистое.

### REM-06 slice plan (merged)

**Goal.** Сохранить каждый `queued`/`running` run и связанную с ним Task/Attempt identity/evidence
при pressure retention; ограничивать retention только terminal run records и не допускать потери
in-flight lifecycle state.

**Finding / baseline.** На свежем `origin/main` `b45a4d41` `trimRunRegistry` сортирует все записи
по `StartedAt` и удаляет самые старые независимо от статуса. `persistHistorySnapshotLocked` затем
повторно берёт только хвост общего списка. При старом queued/running run и превышении retention это
может удалить in-flight запись из памяти и `reports/taskruns/run-history.json`, хотя TASK_SPEC требует
сохранять active/queued identity и terminal summary даже после удаления детальных run artifacts.

**Readiness exception.** `REM-03B` остаётся внешней admin-only задачей без выданной авторизации;
release status явно сохраняется `release-blocked`. Поэтому предусмотренная строкой REM-06
альтернатива зависимости разрешает независимый lifecycle slice без изменения GitHub settings и
без ослабления release gate.

**Non-goals.** Не удалять Task автоматически, не менять public Task/Attempt или run-history schema,
не менять run-log TTL/max-files cleanup, restart reconciliation semantics, stabilization-owned
semantic/docflow paths, release rulesets или owner-waiver policy.

**Affected paths.** `internal/orchestrator/service_runs.go`, новый focused
`internal/orchestrator/retention_test.go`, этот ExecPlan и при необходимости узкий раздел
`docs/spec/TASK_SPEC.md`/`docs/TESTING_STRATEGY.md` только если observed behavior wording требует
уточнения. No schema or fixture output changes are expected.

**Implementation boundary.** Introduce one retention selection helper used both by in-memory registry
trim and persisted history serialization: keep all non-terminal statuses conservatively and retain at
most configured `historyRetention` newest terminal records, ordered deterministically by
`StartedAt`/`RunID`. Add tests for old queued/running records under terminal pressure, persisted
TaskID/AttemptID linkage, all-in-flight retention, and deterministic terminal eviction.

**Regression strategy.** Run focused orchestrator lifecycle/retention tests, then `-race` for the
same package. Full implementation DoD remains `make contracts`, `make test`, `make lint`, `make build`;
no live provider or network execution is required.

**Rollback / stop condition.** Stop if preserving in-flight records changes admission capacity,
terminal ordering, restart reconciliation, or creates unbounded retention beyond the one active plus
one queued pipeline invariant. Roll back if persisted history diverges from in-memory retained IDs or
Task/Attempt linkage is absent. Do not touch stabilization-owned paths.

- 2026-09-05: Fresh base `origin/main=b45a4d41702b0f1bc57246907ed05f7eb3793158`; neighbor
  stabilization remains idle/blocked at revision 29 with uncommitted semantic/docflow/live changes in
  its separate checkout. Candidate REM-06 paths do not overlap that owned set.
- 2026-09-05: Read-only GitHub evidence confirms branch protection currently requires six deterministic
  contexts (`backend`, `contracts`, `ui`, `golden`, `smoke-cli`, `smoke-api`); `REM-03B` remains
  authorization-gated and no setting mutation is in scope for this slice.
- 2026-09-05: Baseline retention regressions reproduced on the fresh base: old queued/running records
  were evicted from memory and persisted history when terminal records exceeded the budget. After the
  helper-based fix, focused retention tests, orchestrator package tests and `-race` lifecycle subset
  pass. `make contracts`, `make lint`, `make build` and doc-sync checks pass; full `make test` reached
  the complete Go suite but was interrupted after an unrelated `matrix_release_contract_test` setup
  hung in `git add -A` under concurrent worktree load.
- 2026-09-05: PR #276 passed all required deterministic CI contexts and was squash-merged into `main`
  as `99c019fca9535cd0906648a2f02af0d57c2b1e61`; `origin/main` was fetched and verified clean.

### REM-07 slice plan (in progress)

**Goal.** Привязать Attempt registry watchers к жизненному циклу API server: watcher должен
останавливаться на server shutdown, завершаться после terminal run/cancel и не оставлять
неуправляемые goroutine или race при повторном запуске/закрытии.

**Finding / baseline.** На свежем `origin/main` `watchAdmittedAttempt` запускает `go` без context,
WaitGroup или cancellation ownership. Цикл выходит только при успешной terminal синхронизации или
потере run; при shutdown он не получает сигнал и `Server.Shutdown` не ожидает его завершения. При
неуспешной записи Task history terminal watcher может продолжать тикать после закрытия сервера.

**Readiness.** REM-06 merged as `99c019f`; REM-07 depends only on that lifecycle invariant. Neighbor
stabilization remains external/blocked at revision 30; no owned semantic/docflow/live paths overlap
this slice. REM-03B remains explicitly authorization-gated and release-blocking.

**Non-goals.** Не менять Attempt/Task schema или API shape, run retention semantics, provider
runtime, stabilization-owned paths, GitHub settings, or release policy. Do not add a new watcher
backend or polling interval redesign.

**Affected paths.** `internal/api/server.go`, `internal/api/task_attempts.go`, focused API watcher
tests, this ExecPlan and the lifecycle bullet in `docs/TESTING_STRATEGY.md` if wording needs
clarification. No schema/fixture output changes are expected.

**Implementation boundary.** Give `Server` an owned watcher context, cancellation and WaitGroup;
register every watcher before launching it; make the loop select on context and ticker; close watcher
admission under a dedicated lifecycle lock before `Server.Shutdown` publishes terminal run state,
then cancel and wait without holding the general admission lease. Serialize repeated shutdown calls
separately so long context-bound API operations do not make shutdown ignore its deadline. Preserve
retry-on-transient Task history write failures until terminal success or server cancellation.

**Acceptance / regression.** A running Attempt is mirrored to terminal state; cancellation and
server shutdown leave no active watcher; repeated Shutdown is safe; `go test ./internal/api` and
`go test -race ./internal/api` pass. Full DoD remains `make contracts`, `make test`, `make lint`,
`make build`; no live provider/network execution is required.

**Rollback / stop condition.** Stop if shutdown waits indefinitely on a provider or registry write,
if a watcher updates a replaced workspace session, or if cancellation suppresses a terminal Attempt
update that can still be durably published. Roll back if race detection shows Add/Wait overlap or
if watcher count is not quiescent after shutdown.

- 2026-09-05: Initial slice base was `origin/main=15c29fc2587402b8f235bfc82de93461ead31241`;
  after cleanup PR #270 merged during implementation, branch was merged with fresh
  `origin/main=dd686fd4bc6bed2d36618ec80e8c57a491e445e1` before PR checks. Neighbor status revision
  30 remains blocked/idle with separate uncommitted stabilization changes. No path overlap.
- 2026-09-05: Baseline code review confirms watcher ownership is absent: `watchAdmittedAttempt`
  starts an untracked ticker goroutine and `Server.Shutdown` delegates only to orchestrator service
  shutdown. Focused lifecycle regression will establish the pre-fix leak/quiescence behavior.
- 2026-09-05: Implemented server-owned watcher context/cancel/WaitGroup; shutdown now lets
  orchestrator terminalization publish first, gives watchers a bounded 500ms grace window, then
  cancels and waits. Cancellation and shutdown/repeated-shutdown tests pass, including terminal
  Attempt mirroring and watcher quiescence.
- 2026-09-05: Review found that holding the general admission lease across `service.Shutdown` could
  delay shutdown behind long git/validation handlers. Replaced that coupling with a dedicated watcher
  lifecycle lock plus serialized shutdown calls; watcher admission closes before terminalization while
  cancellation remains deferred until the bounded grace window completes. API race and package tests
  remain green.
- 2026-09-05: Focused API and `-race` API suites pass. `make contracts`, `make lint` and `make build`
  pass. Full `make test` completed all Go packages except one unrelated load-sensitive
  `internal/runtime/providercommon/TestRunHeadlessProviderRespectsGlobalInvocationBudget` assertion;
  the failing test passes in an isolated rerun, so the failure is not in REM-07 paths.
- 2026-09-05: PR #279 passed all required deterministic CI contexts and was squash-merged into
  `main` as `079f9b51c95622abb6fc417234df52b0f1a36d0a`; fresh fetch confirms `origin/main` at the
  merge commit. REM-07 is closed and REM-08 is the next ready independent P1 slice.

### REM-08 slice plan (merged)

**Goal.** Сохранить immutable admission context queued Attempt до фактического запуска и после
перезапуска сервиса либо продолжить с тем же exact context, либо fail-closed с понятной диагностикой;
исключить silent scope/runtime drift.

**Finding / baseline.** На свежем `origin/main=91e52d0c` `CloneAdmittedRuntimeSnapshot` клонировал
map-поля, но оставлял `RepositoryScopes` общим slice. `StartAsyncRun` также сохранял указатель на
переданный snapshot в `pendingRun`, поэтому мутация caller-owned slice до запуска очереди могла
изменить фактически выполненный scope. Restart rehearsal показал, что обычный queued run уже
terminalizes fail-closed с `run_reconciled_after_restart`, но этот invariant не имел API regression
coverage на exact Task/Attempt context.

**Readiness.** REM-06 и REM-07 merged; REM-08 не пересекается с соседними semantic/docflow/live
paths. Соседняя stabilization задача остаётся blocked/idle на revision 30 с незакоммиченными
изменениями в отдельном checkout. REM-03B остаётся явно authorization-gated и release-blocking.

**Non-goals.** Не менять Task/Attempt schema или API shape, queue capacity, provider defaults,
restart policy, retention, watcher lifecycle, stabilization-owned paths, GitHub settings или live
provider matrix.

**Affected paths.** `internal/runtime/admission.go` и его clone regression test,
`internal/orchestrator/service_runs.go` и queued snapshot test, API restart reconciliation test,
этот ExecPlan. Schema/examples/fixtures не меняются.

**Implementation boundary.** Глубоко клонировать все reference-поля `AdmittedRuntimeSnapshot` при
admission до сохранения `RunRequest` в pending queue. Оставить существующую безопасную политику
restart для queued runs (terminal `run_reconciled_after_restart`), а API reconcile должен сохранить
неизменными IntentSnapshot и EffectiveRuntime и связать terminal state с теми же Task/Attempt/run
IDs.

**Acceptance / regression.** Caller mutation после queue admission не меняет persisted/executed
`RepositoryScopes`; queued Attempt после service restart остаётся exact identity, сохраняет immutable
intent/effective runtime и получает terminal failed state с `run_reconciled_after_restart`; focused
runtime/orchestrator/API tests and `-race` pass. Full DoD remains `make contracts`, `make test`,
`make lint`, `make build`; no live provider/network execution is required.

**Rollback / stop condition.** Stop if a queued run reads caller-owned mutable data, restart
reconciliation changes immutable Attempt fields, loses exact Task/Attempt/run linkage, or silently
resumes a queued run without an explicit persisted snapshot. Roll back if schema validation or
existing queue/restart semantics regress.

- 2026-09-05: Fresh main `91e52d0ccea759894a3c70168b2947ac70cb924e` and neighbor revision 30 checked;
  no stabilization overlap. Reproduced the shallow `RepositoryScopes` clone and confirmed queued
  restart currently fails closed, establishing the bounded implementation slice.
- 2026-09-05: Added deep snapshot cloning at async admission plus runtime and orchestrator regression
  tests for caller mutation. Added API restart rehearsal proving queued Attempt terminal diagnostic,
  exact identity and immutable intent/effective runtime are preserved.
- 2026-09-05: PR #281 passed all 11 required checks and merged as `02086568a785284dbbecadd14c4ecb658961227a`;
  fresh `origin/main` was fetched. REM-08 is closed and REM-09 is the next independent ready slice.

### REM-09 slice plan (merged)

**Goal.** Для remote `git_url` с moving branch/tag ref фактически использовать и сохранять exact
commit identity, чтобы fetch не оставлял stale local branch и изменение удалённой ветки между
validation и execution не превращалось в silent source drift.

**Finding / baseline.** На свежем `origin/main=245f462b` resolver после `git fetch origin` проверяет
и checkout-ит запрошенный `repo.Ref` напрямую. Для plain branch (например, `main`) это может
разрешить локальную cache branch, которая осталась на старом commit, тогда как обновлённый
`origin/main` уже указывает на новый commit. `ResolvedSHA` и `source-revisions.json` в таком случае
честно описывают stale checkout, а не moving remote ref. Existing tests cover unpinned default
freshness и pinned SHA stability, но не explicit moving branch ref.

**Readiness.** REM-08 merged; REM-09 затрагивает только workspace source resolver и source-identity
regressions, не пересекается с соседними semantic/docflow/live stabilization paths. Соседняя
stabilization задача проверена на revision 30 и остаётся blocked/idle в отдельном checkout.
REM-03B остаётся явно authorization-gated и release-blocking.

**Non-goals.** Не менять `workspace.yaml` или public JSON schemas, path-source checkout safety,
credential model, provider/runtime selection, refresh planner semantics, remote network policy или
stabilization-owned files.

**Affected paths.** `internal/workspace/resolver.go`, `internal/workspace/resolver_test.go`,
`internal/orchestrator/source_resolution_test.go` only if execution-evidence coverage is needed,
`docs/spec/WORKSPACE_SPEC.md` and this ExecPlan. Existing `ResolvedRepo.ResolvedSHA`/source-revisions
contract remains the identity surface; no schema change is expected.

**Implementation boundary.** После fetch разрешать explicit moving refs через свежий
remote-tracking ref (`origin/<ref>`/`refs/remotes/origin/...`) before any stale local branch;
resolve to a full commit SHA and detach/reset only the ACP-owned cache to that SHA. Preserve direct
SHA and tag behavior, keep path sources non-mutating, and ensure persisted run evidence records the
same cache `HEAD` identity used by execution. Add local bare-remote regression that advances a
branch between two resolutions and asserts fresh content/SHA plus no stale local-branch checkout.

**Acceptance / regression.** Explicit `git_url` branch ref follows the fetched remote-tracking
commit after the branch advances; `ResolvedSHA == HEAD` and run/source evidence remain exact and
reproducible. Pinned SHA and tag behavior remains stable, path verification remains read-only,
focused resolver/orchestrator tests (including `-race` where applicable) pass, and full DoD remains
`make contracts`, `make test`, `make lint`, `make build` with no live provider/network dependency.

**Rollback / stop condition.** Stop if a moving remote ref resolves to a pre-fetch local branch,
if a cache checkout remains attached to a mutable branch, if pinned refs change unexpectedly, or if
path repositories are mutated. Roll back on source-revision/evidence mismatch, schema drift or any
unrelated stabilization overlap.

- 2026-09-05: Fresh `origin/main=245f462b` and neighbor stabilization revision 30 checked. Source
  resolver review identified the stale-local-branch path; existing tests lack moving explicit branch
  coverage. Implementation remains bounded to ACP-owned git_url caches and commit identity evidence.
- 2026-09-05: Added remote-ref candidate resolution and detached exact-SHA checkout for fetched
  `git_url` sources, plus cache `HEAD` identity mismatch diagnostics. Local bare-remote regression
  now advances explicit `main` and confirms fresh content/SHA; pinned SHA, default-head and path
  safety coverage remain green. Updated `docs/spec/WORKSPACE_SPEC.md` with the source identity and
  diagnostic contract.
- 2026-09-05: Focused and full workspace/orchestrator Go tests passed, including race checks for the
  moving-ref path. `make contracts`, `make lint` and `make build` passed with local Node 22.22.3
  override; the full `make test` Go phase passed and the Python suite passed without override. The
  override run's four node-tool failures were expected fixture-version mismatches (22.22.3 vs
  repository `.node-version` 22.21.1), not REM-09 regressions.
- 2026-09-05: PR #283 passed all 11 required checks and merged as `1155f12afdc3c99a5a62cbeee4f21131d51a5e5c`;
  fresh `origin/main` was fetched. REM-09 is closed and REM-10 is the next independent ready slice.

### REM-10 slice plan (merged)

**Goal.** Сделать публикацию Git и связь с точным Task/Attempt/run recoverable при частичном сбое:
commit или branch mutation не должен теряться, а Task/Attempt не должен показывать ложный
`Published`, если запись linkage не завершилась.

**Finding / baseline.** На свежем `origin/main=9a235cda` оба Git mutation handler'а сначала выполняют
необратимый `git commit`/`git checkout -b`, а затем вызывают `recordTaskPublication`. Если после
Git side effect запись `task-history.json` падает (ошибка диска, transient write fault или crash
между операциями), response возвращает `publication_linkage_failed`, но durable registry не содержит
ни association, ни recovery marker. Повтор операции не может надёжно восстановить уже созданный
commit, а UI/Task history не имеют доказуемой границы между `Published` и `unavailable`.

**Readiness / parallel stabilization.** REM-09 и plan-sync PR #284 merged в `origin/main=9a235cda`.
Соседний тред `Проверь UI и UX проекта` проверен непосредственно перед срезом: revision 30,
`blocked/idle`; его незакоммиченные semantic/docflow/live изменения остаются в отдельном checkout.
REM-10 не изменяет stabilization-owned paths (`docs/ARCHITECTURE.md`, semantic/docflow runtime и
live diagnostics) и не зависит от их незавершённого gate. REM-03B остаётся authorization-gated и
release-blocking.

**Non-goals.** Не менять Task/Attempt JSON schema, Git publication scope, stale-confirmation policy,
provider/runtime behavior, release settings, UI copy, remote network behavior или Git history.
Не обещать cross-process rollback Git mutation: recovery должна либо доказать exact operation по
сохранённой identity, либо оставить explicit pending/unavailable state.

**Affected paths.** `internal/api/task_publication.go`, `internal/api/server.go`, Git mutation/API
tests, `docs/spec/TASK_SPEC.md`/`docs/spec/API_SPEC.md` only where the recoverable boundary needs
clarification, and this ExecPlan. No schema or fixture shape changes are expected.

**Implementation boundary.** Перед Git side effect atomically persist a compact, structured
publication-intent marker in the ACP Git metadata journal (`acp-publication-journal.json`), containing
exact context, action, target branch and confirmed pre-mutation Git identity/fingerprint. On
successful linkage, remove that marker after the registry transaction that writes Task and Attempt
publication; either ordering remains recoverable on restart. On Git failure or no-op, clear the marker
best-effort without changing a prior linked publication. On server/workspace attach, inspect pending
markers and auto-link only when a strict proof exists: commit HEAD is exactly one commit over the
recorded parent with the recorded message, or branch action has the exact target branch and recorded
HEAD; otherwise keep the marker and leave publication unavailable for explicit recovery. The journal lives in
`.git` metadata and never becomes a workspace publication artifact.
The handlers remain fail-closed and never synthesize a publication from recency or a clean tree.

**Acceptance / regression.** Context/journal persistence failure before Git prevents mutation; a
simulated crash after Git and before linkage leaves a durable intent and a later server reconciliation restores the
exact Task/Attempt publication for both commit and branch actions; ambiguous or failed Git state
does not become `linked`; no-op/failure paths do not leave stale intents when cleanup succeeds. Existing
exact-context, partial-context and stale-confirmation tests remain green. Focused API/tasks tests,
`-race` checks and full deterministic DoD (`make contracts`, `make test`, `make lint`, `make build`)
pass without live provider/network execution.

**Rollback / stop condition.** Stop if intent persistence is not atomic, if a pending marker can be
mistaken for a successful publication, if recovery links an unrelated commit/branch, if normal
publication leaves permanent control-file drift, or if the slice touches stabilization-owned paths.
Roll back on Task/Attempt schema drift, changed Git scope or any failure to preserve a previously
linked publication during a failed subsequent mutation.

- 2026-09-05: Reproduced the partial boundary on `origin/main=9a235cda`: Git mutation is durable
  before `recordTaskPublication`, while linkage failure returns 500 without a durable recovery
  record. Neighbor stabilization remains revision 30 `blocked/idle`; no path overlap.
- 2026-09-05: Implemented an atomic-write Git metadata journal for contextual commit/branch
  intents. Handlers prepare the journal before side effects, clear it on Git failure/no-op, and
  remove it after successful Task/Attempt linkage; restart reconciliation requires exact branch/head
  or one-parent commit plus message proof and leaves ambiguous state unavailable. Added API
  regressions for commit recovery, branch recovery, ambiguous commit fail-closed behavior and clean
  no-op cleanup. Full API/tasks tests, race recovery subset, docs guidance checks, contracts and
  lint/build pass; full Go suite passes and the Python suite has only the known Node 22.22.3 override
  mismatch in four toolchain fixture assertions (pinned 22.21.1 isolated tests pass).
- 2026-09-05: Full `go test -race ./internal/api` was attempted after the slice; it remains blocked
  by pre-existing parallel-runtime races in `internal/orchestrator/pipelineExecution.addArtifacts`
  observed by unrelated API lifecycle tests. The REM-10 recovery race subset is green and no race
  touches the journal paths.
- 2026-09-05: Review hardening constrained the journal path to the exact fixed file inside Git's
  common metadata directory and added traversal/escape regressions (`4f425cb6`). PR #285 passed all
  required branch checks (backend, contracts, lint, UI, golden and smoke checks) and squash-merged as
  `0842930a`; four CodeQL path-injection annotations were reviewed as false positives after the
  containment guard and their bot review threads were resolved. Fresh `origin/main` was fetched;
  the neighboring UI/UX stabilization thread remains revision 30 `blocked/idle`, with its owned paths
  untouched. REM-10 is closed; REM-11 is the next independent P1 slice.

## EP-20260811-task-attempt-contracts

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Close this W23A foundation plan after W25 consumes fresh trusted public Task/Attempt evidence.

### Context

Epic 23 cannot start as a frontend rename of pipeline runs. The accepted target introduces a durable
product Task, immutable Attempt admission, exact legacy-run behavior and honest full-workspace
publication linkage. Decision authority is fixed in `docs/spec/TASK_SPEC.md` and the 2026-08-11
Task/Attempt ADRs; current APIs remain authoritative until this plan is implemented.

### Goals (must have)

- [x] Add schema-validated public Task, Attempt and registry contracts with full schema-guardian sync.
- [x] Implement crash-safe `task-history.json` current/last-good persistence and restart diagnostics.
- [x] Add create/list/read/update/archive/unarchive Task APIs with stable pagination and revision checks.
- [x] Add idempotent per-Attempt admission with immutable scope/runner snapshots and exact run linkage.
- [x] Preserve pre-contract runs as explicit read-only legacy evidence without synthetic Tasks.
- [x] Expose unknown/unavailable result and publication linkage without inferred `Published` state.
- [ ] Close this W23A foundation plan after W25 consumes fresh trusted public Task/Attempt evidence.

### Non-goals

- Task-first shell cutover, artifact workbench implementation or removal of current routes.
- Unlimited scheduling, hosted collaboration, Task deletion or source-repository writes.
- Epic 24 validation authority changes or trusted-provider live qualification.

### Approach

1. Land Task/Attempt/registry schemas, Go contracts, fixtures, examples and documentation as one
   schema-first review boundary.
2. Add a dedicated Task registry using the existing atomic current/last-good lifecycle pattern,
   including write-fault and restart recovery tests.
3. Add Task read/write APIs and exact Task/Attempt/run joins without changing current run endpoints.
4. Extend admission to resolve and snapshot per-Attempt runner/scope under the shared service lease;
   preserve one active plus one queued pipeline Attempt.
5. Add legacy-run and publication-linkage unavailable states, then run focused contract/API tests and
   full deterministic DoD before opening 23C frontend work.

### Files expected to change

- `schemas/*task*.schema.json`, `internal/contracts`, new Task registry/application package.
- `internal/orchestrator` admission/run linkage and `internal/api` Task handlers.
- `internal/runtime/fakeruntime` and fixtures only where admitted snapshot coverage requires them.
- `docs/spec/TASK_SPEC.md`, `docs/spec/API_SPEC.md`, `docs/spec/WORKSPACE_SPEC.md`,
  `docs/APPENDIX_SCHEMAS.md`, `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md`.

### Acceptance criteria

- [x] Task survives restart and remains the same aggregate across child retry/rerun Attempts.
- [x] Admitted Attempt config cannot change after Settings/workspace/env updates.
- [x] Invalid identity/scope/runner and queue overflow fail before provider execution.
- [x] Duplicate start token returns the same Attempt and cannot create a second run.
- [x] Primary/last-good write faults never publish partial Task state in memory.
- [x] Legacy runs remain readable and never appear as fabricated Task rows; the explicit
      `/tasks/legacy` surface suppresses start, retry, queue and cancel mutations while retaining
      status, outcome and diagnostics evidence.
- [x] `make contracts`, `make test`, `make lint` and `make build` pass.

### Risks

- Task and run registries are separate durable files; admission/terminal recovery must never invent a
  cross-file association after a partial write.
- Registry growth is unbounded in the MVP because Task deletion/retention is explicitly out of scope.
- Current single-pending replacement behavior may need a narrow compatibility migration so a queued
  Attempt belonging to another Task is never silently superseded.

### Progress log

- 2026-08-13: Re-ran the provider-independent closure on `main` at `2edb83bc` with the pinned
  Node `22.21.1` toolchain: contracts, Go tests, Python `272/272`, UI `47/47` files and `255/255`
  tests, ShellCheck, TypeScript, production build, readable-fixture drift (`90` artifacts),
  deterministic UI build, embedded UI parity and mock E2E `8/8` all passed. The live-flow contract
  still requires exact public Task/Attempt identity and explicitly forbids latest-run, synthetic
  legacy identity and second-analysis fallback. A Codex-only non-release smoke was started with
  `gpt-5.6-luna`/high reasoning after preflight, reached real headless Task/Attempt execution, and
  was stopped after a bounded observation while the provider remained in a long-running init shard;
  it is diagnostic evidence only.
- 2026-08-12: W23O closure passed contracts, Go and Python suites (267 tests), UI unit tests
  (47 files/255 tests), all eight deterministic mock E2E scenarios, lint, production build and
  embedded UI parity. The local Node override is only required because this machine has 22.22.3
  while the repository pins 22.21.1; resolver tests pass without the override.
- 2026-08-12: Closed the remaining W23A legacy-surface gap. `AnalysisStagePanel`, targeted retry and
  recovery controls now receive an explicit read-only boundary for `/tasks/legacy`; legacy evidence
  keeps bounded diagnostics and technical details but cannot start, retry, queue or cancel a run.
  App/component assertions and the failed-shard, QA-recovery and Task-first mock E2E paths cover the
  boundary without synthetic Task identity or a second provider analysis.
- 2026-08-12: Re-ran the complete deterministic DoD after the read-only change with pinned Node
  22.21.1: contracts valid, Go/Python (271 tests) and UI (47 files/255 tests) suites green, all eight
  mock E2E scenarios green, lint/build green, and `ui/dist` byte-identical to embedded `ui_dist`
  (excluding its README).
- 2026-08-12: Closed the W23A publication-linkage gap. Task-bound Changes routes now carry an exact
  Task/Attempt/run context into full-workspace Git commit/proposal-branch mutations. The server
  validates the join before mutation and, only after success, records an immutable publication
  association on both Task and Attempt with action, branch/base/head identity, resulting commit and
  the confirmed inventory fingerprint. Missing or partial context remains explicitly unavailable;
  no latest-run, clean-worktree or legacy fallback is used. Focused API, schema, route and UI tests
  cover linked, unavailable and fail-closed paths.
- 2026-08-12: W23O route-closure PR #249 merged to `main` as squash commit `3699f03b` after all
  required GitHub checks passed.
- 2026-08-11: Owner accepted Task/Attempt authority, registry path, per-Attempt admission,
  single-active/single-queued coordination, legacy-run and publication-linkage decisions. Added the
  ADR/spec package and implemented W23A1 schemas, semantic Go contracts, examples and provider-free
  fixtures. W23A2 persistence, W23A3 Task APIs and W23A4 admission/linkage are now implemented;
  W23B1–N and W23O route closure are implemented; W24G and Epic 25 remain pending.
- 2026-08-12: The decision package was verified merged through PR #223 (`1d9c56bf`); subsequent
  W23/W24 implementation and W25 release-alignment work is based on that merged authority.

## EP-20260811-task-first-ui

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Close this UI ExecPlan after the W25 release-gate evidence confirms the Task-first frontend journey on a trusted machine.

### Context

The accepted Task-first target must enter the current shell additively. W23B1 establishes typed
route/history authority and truthful containers before any primary-navigation cutover; later
slices own Task data loading, composer, Attempt outcome, Architecture/Changes workbenches and
the final removal of Home/Analyze routes. W23O leaves only explicit read-only `/tasks/legacy`
  diagnostics for pre-Task evidence.

### Goals (must have)

- [x] Add typed `/tasks`, `/tasks/new`, Task detail and Task/Attempt routes with exact identity
      round-trips.
- [x] Preserve optional Task context on Architecture and Changes routes.
- [x] Fail closed on malformed or unsafe Task identities without selecting another run/result.
- [x] Add an explicit read-only W23B1 target container with focused component coverage.
- [x] Implement the W23C New Task composer with displayed scope and inline runner readiness.
- [x] Implement the W23D Task Inbox/detail/Attempt history read vertical slice with URL-restorable
      filters, authoritative lifecycle groups and explicit archive/unarchive states.
- [x] Implement the W23E outcome-first Task detail result summary bound to the exact Attempt run
      review, including explicit semantic partial state and independent current Architecture status.
- [x] Implement the additive W23F Attempt-bound Pipeline Studio with structured steps, bounded
      blocker/diagnostics disclosure and no provider-output percentage inference.
- [x] Implement W23O closure. W23G Task-scoped Architecture context, W23H
      allowlisted Markdown reader/editor, W23I model inspector, W23J Mermaid Evidence Studio,
      W23K findings queue, W23L Changes context, W23M Ask/Runner authority boundaries and W23N
      Task state/accessibility coverage are implemented additively.
- [x] Complete the full deterministic DoD after W23O (contracts, backend/UI tests, lint, build and
      embedded UI parity) on the clean implementation branch before declaring Epic 23 closed.
- [x] Complete the 23B2 primary navigation cutover: ProductShell now exposes only
      Tasks/Architecture/Changes plus utilities, and no longer renders Home/Analyze links or
      selectors. W23O now exposes only the explicit `/tasks/legacy` read-only migration surface for
      pre-Task runs and removes the legacy shell components.
- [x] Merge the W23O route-closure implementation branch and record the post-merge verification;
      W24G and Epic 25 remain owned by their separate active plans.
- [x] Re-run the combined deterministic closure after the W25A/B release-gate changes, including
      embedded UI parity and the public Task/Attempt evidence boundary.
- [x] Record the W24G entry metric and hand off the Epic 25 release-gate slices to their follow-up
      implementation plan.
- [ ] Close this UI ExecPlan after the W25 release-gate evidence confirms the Task-first frontend
      journey on a trusted machine.

### Non-goals

- Loading or mutating Task data before its owning vertical slice is ready.
- Synthetic Tasks for historical runs or implicit latest-run selection.
- Trusted live E2E; that remains gated behind W23 `23O` and W24 `24I`.

### Acceptance criteria

- [x] Task route codec preserves exact selected identities across direct load and format/parse
      round-trips.
- [x] Invalid Task path/query identity produces an explicit notice and canonical safe route.
- [x] W23B1 target container exposes no fabricated Task/Attempt/run data.
- [x] W23C composer submits only displayed scope and routes to the exact created Task identity.
- [x] Back/Forward/reload and rendered responsive/keyboard state matrix pass after cutover.
- [x] Full deterministic DoD passes for each merged UI slice.

### Progress log

- 2026-08-11: Implemented W23B1 typed route codec, Task-aware Architecture/Changes context and
  truthful target container, then W23C New Task composer with inline runner readiness and
  authoritative create call. Merged W23C as PR #235 after full CI. W23D now loads public Task and
  Attempt identities, restores Inbox filters in URLs, derives five lifecycle groups and preserves
  archive/read-only evidence semantics. W23E now renders exact-run outcome and semantic deltas with
  explicit missing-comparison state. W23F now adds the exact Attempt-bound Pipeline Studio route
  with bounded structured diagnostics. W23G now adds an explicit Task-scoped current Architecture
  context and return path without latest-run fallback. W23H now adds an allowlisted, lossless
  Markdown reader/editor while promoted reports remain read-only. W23I now adds a structured,
  read-only entity/edge inspector with schema/version labels, path-linked validation issues and
  an advanced line-numbered source view; structured save remains gated on a future lossless
  round-trip proof. W23J now adds a read-only Mermaid Evidence Studio with Raw source fallback
  and an accessible validated relation list that never treats diagram layout as semantic truth.
  W23K now adds a filtered findings/questions/gaps queue, bounded detail and an explicit no-approval
  boundary. W23L now adds exact Task context to Changes and keeps current evidence separate from
  snapshot publication. W23M now makes the global Ask read-only authority and Runner Settings
  desired/effective/source boundary explicit, including provider-outage behavior. W23N now covers
  Task Inbox empty/error recovery, keyboard row activation, 44px filter/Attempt targets and explicit
  status semantics. The 23B2 navigation cutover now makes Tasks primary and removes Home/Analyze
  links/selectors; unknown/root console paths now resolve to Task Inbox. W23O closes the legacy
  shell by canonicalizing old run links to `/tasks/legacy` read-only diagnostics.

## EP-20260812-task-first-live-evidence-alignment

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Run canonical live validation on a trusted host and obtain fresh public Task/Attempt evidence.

### Context

W25 must preserve the canonical trusted-machine release boundary after the Task-first UI and Epic 24
authority changes. The release frontend may inspect only one exact public Task/Attempt/run snapshot;
the batch reporter may consume only persisted public audit/verdict/budget fields. A missing product
identity is evidence failure, not permission to synthesize a Task from historical runs.

### Goals (must have)

- [x] Cut `init-inspect` over to exact public Task/Attempt admission and identity-bound Pipeline Studio.
- [x] Remove latest-run and legacy-shell fallback from the release-facing Playwright flow.
- [x] Add public-only promotion-audit, effective-verdict and recovery-budget fields to batch reports.
- [x] Keep historical v1 report fields readable and preserve existing failure taxonomy.
- [x] Wire trusted snapshot preparation to product-authored Task/Attempt history without synthetic
      identities; the backend snapshot now preserves both product registry generations.
- [ ] Run canonical live validation on a trusted host and obtain fresh public Task/Attempt evidence.
- [x] Complete full deterministic DoD and embedded UI parity after the W25 changes.

### Non-goals

- No new providers, canonical matrices, curated repositories, release taxonomy or live CI workflow.
- No internal Go validator/orchestrator imports in the harness and no second provider analysis.
- No synthetic Task/Attempt rows for legacy runs.

### Approach

1. Update the frontend scenario and contract tests to resolve exact public Task/Attempt identities.
2. Consume only persisted quality-summary authority fields in batch reports and fail closed on audit or
   recovery-budget violations.
3. Synchronize the runbook, testing strategy and backlog; run focused checks and the deterministic DoD.
4. Use `acp-e2e-live-gate` for any later trusted-machine run; do not alter the canonical harness.

### Acceptance criteria

- [x] Frontend snapshot flow rejects missing/ambiguous public Task/Attempt identity and never selects
      latest run or `/tasks/legacy` as a release fallback.
- [x] Report TSV/Markdown contains effective authority, audit result, invocation budget and first-pass
      validation evidence sourced from public JSON only.
- [x] `make contracts`, `make test`, `make lint`, `make build` and embedded UI parity pass.
- [ ] Canonical live gate has fresh public Task/Attempt evidence and no second analysis.

### Progress log

- 2026-08-12: Implemented W25A Playwright Task/Attempt admission and exact snapshot joins; updated
  frontend contract tests. Snapshot mode now fails closed instead of materializing synthetic identity.
- 2026-08-12: Implemented W25B public-authority extraction and report/TSV fields; audit failure and
  provider-budget exhaustion are hard gates while historical missing fields retain legacy semantics.
- 2026-08-12: Updated release runbook, testing strategy and backlog to document the public identity
  boundary. Trusted harness seeding and the final release gate remain open until a product-authored
  Task/Attempt history is present.
- 2026-08-12: Snapshot preparation now preserves the product-authored `task-history.json` and
  `.last-good` registry beside the immutable Attempt snapshot; frontend release inspection can
  resolve the exact public Task/Attempt without synthesizing identity from legacy run history.
- 2026-08-12: Canonical `release-fast-20260812T060000Z` was launched from clean commit
  `d204a8cc`; host, exact Node/Go, disk and pinned path preflight passed, Codex readiness passed,
  but Qwen and Claude failed the provider artifact smoke with the configured Kimi billing-cycle
  `403 quota_or_permission`. The matrix fail-closed as `RELEASE BLOCKED`; no live Task/Attempt or
  frontend evidence was produced, so trusted W25 closure remains open.
- 2026-08-12: W25 deterministic closure passed `make contracts`, full Go/Python/UI tests (271 Python,
  47 UI files/255 tests), lint, build, eight deterministic mock E2E scenarios and embedded UI parity.
- 2026-08-12: Non-release `smoke-tiny-bank-20260812T062000Z` exercised the public Task/Attempt
  admission path with Codex only. The headless workspace and fake control snapshots contained
  product-authored `task-history.json` plus `.last-good` and exact Task → Attempt → run joins; the
  provider run then failed closed on a real `init.step2.asis_docs` runtime stall after its draft
  referenced a missing `overview.md` (`runtime_quality.stall_pressure=1`). This is diagnostic
  evidence that the identity boundary is wired, not trusted W25 release evidence; frontend was
  intentionally skipped and the canonical release gate remains open.
- 2026-08-12: Snapshot-backed `init-inspect` was then run against the successful fake init snapshot
  `run_264afe94e4d3cb67125b` with `UI_E2E_QA_SMOKE=0`. The public Task/Attempt/run join passed
  through Pipeline Studio, Architecture/Documents, responsive/keyboard checks and the remaining
  release-facing flow without starting a second provider analysis. The no-op fake refresh snapshot
  remains unsuitable for this scenario because its run-specific artifact list intentionally contains
  only refresh metadata; this is recorded as fixture behavior, not used as trusted live evidence.
- 2026-08-12: Re-ran the current clean branch deterministic DoD after the evidence updates: contracts,
  Go, Python `271/271`, UI `47/47` files/`255/255` tests, ShellCheck, UI typecheck, Vite/Go build,
  embedded `ui_dist` equality and mock E2E `8/8` all passed. No generated build changes remain.
- 2026-08-12: Rechecked the canonical provider artifact-smoke preflight on the trusted host: Codex
  `0.144.1` is ready and writes the sentinel, while Qwen `0.19.11` and Claude `2.1.85` both still
  return the configured Kimi billing-cycle `403 permission_error`. The release matrix was not
  relaunched because the unchanged operational blocker is fail-fast and cannot be bypassed by a
  provider, matrix or taxonomy override.
-  The backend cycle now has a public-API Task/Attempt helper so future snapshots carry product-authored
  identity without synthesizing legacy runs.
- 2026-08-12: A restart rehearsal found that cloning an empty diagnostics slice changed the persisted
  task-history shape from `[]` to `null`. W23A persistence now preserves array shape in cloned history
  and public diagnostics responses; focused registry/API coverage and the exact-Node snapshot-backed
  `init-inspect` flow pass after restart. This remains deterministic evidence, not trusted-provider
  release evidence.
- 2026-08-11: Canonical `release fast` was attempted through
  `scripts/full-run-batch-matrix.sh` with exact Node/npm 22.21.1, writable trusted-gate roots and
  32.7 GB free space. All four profile/sweep records failed closed during provider readiness:
  `claude` and `qwen` both returned `quota_or_permission` (the configured Kimi billing-cycle
  account), while Codex readiness passed. No provider-backed run, Task/Attempt snapshot or frontend
  evidence was produced. Release verdict `release-fast-20260811T233300Z` is `RELEASE BLOCKED`. The
  next action is an unchanged canonical rerun on a host/account with Claude and Qwen quota/permission;
  matrices, providers and release taxonomy remain untouched.

## EP-20260811-weak-model-validation-authority

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Close the Epic 24 authority plan after W25 consumes fresh public audit/verdict evidence on the trusted release gate.

### Context

Epic 24 must close false-accept and repair-loop classes without weakening sparse truthful output or
making provider opinion the promotion authority. The accepted chain is provider draft → deterministic
technical candidate → mandatory provider-free selected-run audit → persisted effective verdict.
This plan covers reviewable slices 24A–24I; 24G remains conditional on the recorded entry metric.

### Goals (must have)

- [x] Bind validator identity and checked paths to the exact current runtime task/run snapshot.
- [x] Enforce coherent provider draft verdicts and deterministic issue ownership/order.
- [x] Validate evidence line/excerpt/hash claims through one bounded shared implementation.
- [x] Reject unknown semantic drift and dangling/colliding cross-shard graph identities.
- [x] Run provider-free selected-run audit before the first canonical write.
- [x] Persist/expose a separately versioned orchestrator-owned effective technical verdict.
- [x] Bound provider process starts to three per runtime execution unit across normal and recovery transitions.
- [x] Complete the provider-free recovery-budget and conformance closure in W24H–W24I; keep W24G
  conditional until its entry metric is recorded.
- [x] Record the W24G entry metric and decide whether mechanical-envelope reduction is warranted;
      the provider-free corpus is 100% first-pass-valid, 0% repair-entry and p95=2, so W24G is
      deferred without changing the public envelope.
- [x] Re-run the combined deterministic DoD after downstream W25 public-report integration so the
      Epic 24 authority remains verified at the release boundary.
- [ ] Close the Epic 24 authority plan after W25 consumes fresh public audit/verdict evidence on the
      trusted release gate.

### Non-goals

- New provider/model defaults, semantic invention, hosted validation or required live network tests.
- Epic 24G mechanical-envelope reduction before its metric entry condition.
- Trusted-machine live validation and any provider/model canary.
- Changes to canonical live matrices, curated repos or release taxonomy.

### Approach

1. Deliver 24A–24D as separate focused PRs with shared typed issues and provider-free fixtures.
2. Build the internal technical candidate only from ordered deterministic issues after assembly and
   allowed deterministic repairs.
3. Reuse the exact artifact auditor scanner/options as a fail-closed pre-promotion gate for candidate
   PASS, preserving its read-only/bounded behavior.
4. Persist the versioned effective verdict after audit and migrate promotion/public diagnostics to
   that authority while preserving historical provider verdict reads.
5. Synchronize contracts/specs/fixtures/ADR rationale and run full deterministic DoD before any
   trusted-machine diagnostic.
6. Apply one shared runtime-unit invocation budget at the provider process-start seam, persist its
   counters in quality diagnostics, and keep 24G conditional until its entry metric is recorded.

### Files expected to change

- `schemas/validator-verdict.schema.json`, semantic schemas and a new effective-verdict schema.
- `internal/contracts`, `internal/artifactquality`, `internal/artifactaudit`.
- `internal/evidence` owns the provider-free CRLF/LF, line-range, exact excerpt and SHA-256 validator.
- `internal/runtime/providercommon`, `internal/orchestrator` validation/repair/promotion.
- `internal/runtime/fakeruntime`, `internal/conformance`, incident-shaped fixtures and adapter parity
  tests.
- `docs/spec/PIPELINE_SPEC.md`, `docs/spec/API_SPEC.md`, `docs/APPENDIX_SCHEMAS.md`,
  `docs/ARCHITECTURE.md`, `docs/TESTING_STRATEGY.md`.

### Acceptance criteria

- [x] Foreign run/path and contradictory provider verdicts fail before semantic merge.
- [x] All evidence consumers return the same normalized locator identity and typed issue codes.
- [x] Unknown semantic fields are never silently dropped and every promoted edge resolves.
- [x] Audit error produces zero canonical writes and leaves prior generation/Git bytes unchanged.
- [x] Provider PASS/FAIL cannot override the effective deterministic result.
- [x] Historical provider verdicts remain readable as legacy/unavailable effective authority.
- [x] Provider starts are bounded and provider-parity tested without live binaries.
- [x] `make contracts`, `make test`, `make lint` and `make build` pass.

### Risks

- Audit and effective-verdict authority can become circular unless the internal pre-audit candidate
  remains distinct from the final persisted verdict.
- Tightening v1 semantic schemas may require a v2 writer with explicit historical v1 dual-read.
- Evidence normalization must be byte-identical across CRLF/LF fixtures without trimming whitespace
  or Unicode content.

### Progress log

- 2026-08-11: Owner accepted the draft/candidate/audit/effective authority chain, exact issue
  consistency rules, evidence normalization and runtime-unit recovery-budget terminology. Added the
  accepted ADR/spec/backlog package. W24A exact run/citation/final containment checks are
  implemented; W24B coherent verdict admission, W24C shared bounded evidence validation and W24D
  semantic envelope/graph checks and the provider-free selected-run pre-promotion audit are
  implemented; W24F effective verdict persistence, public authority selection and retry rebinding
  are implemented; W24H shared three-start runtime-unit budget and W24I conformance corpus/closure
  counters are implemented; W24G entry metric is recorded in
  `fixtures/conformance/w24g-entry-metric.json` and the conditional slice is deferred.
- 2026-08-11: W24H process-start enforcement is provider-adapter agnostic: the same budget context
  counts normal, transport-retry and focused-repair starts, denies the next process before spawn,
  and persists used/remaining counters, last transition and explicit exhaustion reason in runtime
  diagnostics and the run quality summary. Provider-free Claude/Qwen/Codex parity and concurrent
  reservation tests pass; W24I adds the provider-free incident corpus, adapter issue-code parity,
  closure counters and deterministic p95 invocation measurement (two).
- 2026-08-12: Added the provider-free W24G entry metric and retained fixture. The recorded corpus
  has 20/20 first-pass-valid observations, no otherwise-valid repair entries and p95=2; ADR
  `ADR-20260812-w24g-entry-metric.md` defers mechanical-envelope reduction until a future metric
  crosses the accepted threshold.
- 2026-08-13: The combined provider-free closure after W25 remains green; no new W24 code or
  contract changes are required. Fresh public audit/effective-verdict evidence is still pending
  because Claude and Qwen fail the trusted provider readiness gate.

## EP-20260805-live-runtime-safety-fixes

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Run the full release-provider matrix for Claude/Qwen/Codex after the Open edX/OpenStack curated checkouts and provider prerequisites are available on a trusted host.

### Context
Trusted live Codex runs can still mutate protected workspace files even though the runtime task
contract forbids those writes. The audit correctly fails the step, but leaving the mutation in place
can contaminate the next run and makes a failed refresh destructive.

### Goals (must have)
- [x] Restore protected workspace files after an unexpected provider mutation when the post-run
  fingerprint is unchanged.
- [x] Never overwrite an edit made after the provider exits; surface restore conflicts explicitly.
- [x] Serialize audited provider calls inside a run so rollback cannot reintroduce another task's
  transient protected mutation.
- [x] Add regression coverage and synchronize runtime behavior documentation.
- [x] Re-run the trusted-machine Codex live refresh matrix after the host has at least 5 GiB free.
- [x] Reduce provider-authored proposal repair exhaustion in a follow-up live run without weakening
  strict draft validation or hiding the existing quality telemetry.
- [ ] Run the full release-provider matrix for Claude/Qwen/Codex after the Open edX/OpenStack curated
  checkouts and provider prerequisites are available on a trusted host.

### Non-goals
- Do not turn managed permissions into a claimed hard process sandbox.
- Do not allow runtime writes outside task-local staging roots or add a new provider.
- Do not change Git publication semantics or remove existing recovery paths.

### Acceptance criteria
- [x] Protected mutations fail the runtime contract and are restored when safe.
- [x] Post-run conflicts remain untouched and produce a dedicated audit warning/log.
- [x] Live refresh completes without `runtime_write_audit_unexpected_mutation` on the curated fixtures.

### Progress log
- 2026-08-05: Live Codex diagnostic identified refresh shard mutations under `charter/cards/*`;
  implemented serialized audited execution, fingerprint-safe rollback, conflict reporting, UI deep-link
  selection restoration, mobile Publish navigation fix and Playwright/Vitest harness isolation.
- 2026-08-05: Exact Go 1.25.10 targeted/full Go tests, contracts, vet, lint, Node 22 typecheck, 180 UI
  tests, 8 mock E2E scenarios and snapshot-backed Codex UI flow passed. Canonical refresh matrix remains
  pending on a trusted host with its required free-space budget.
- 2026-08-05: Completion audit rechecked the host: `/tmp` has 1.3 GiB free, below the canonical 5 GiB
  guard, so the release matrix was not rerun with an invalid override; the goal remains active for a
  trusted-host rerun.
- 2026-08-05: Direct Codex `regres-long` rerun passed host preflight and full deterministic precheck;
  baseline reached live `gpt-5.6-luna`/high execution. A real harness defect was found while diagnosing
  the earlier interruption: matrix signals marked the profile failed but left child batch/provider
  descendants running. The driver now tracks the active child process tree, terminates it on signals,
  and has a regression test for cleanup; targeted signal test passes.
- 2026-08-05: The live `single-git_url` baseline completed with backend hard-pass 1/1 and Codex
  snapshot-backed frontend live E2E passed. The canonical `single-path` profile stopped before a
  run because its required pinned checkout at `/tmp/provenarch-live-e2e/posthog/posthog` is absent;
  the matrix is therefore an honest `FAIL` for an operational host prerequisite, not a product
  runtime failure. Fresh shell syntax, targeted signal cleanup tests and exact Node 22 lint pass;
  the trusted refresh acceptance remains open until the curated checkout is provisioned.
- 2026-08-05: Durable profile records and reports were re-synthesized after the driver source was
  edited during the in-flight run (which caused a parser error only in the late report phase); the
  backend/frontend terminal artifacts remained intact, and no matrix/provider descendants remain.
- 2026-08-05: Canonical non-release `regres-long` matrix `matrix-20260805T113030Z` passed both
  selected profiles (curated PostHog path and FTGO git URL) with Codex pinned to `gpt-5.6-luna`/`high`:
  backend hard-pass `2/2`, init→refresh summaries present, and snapshot-backed frontend live E2E `2/2`
  with nine screenshots plus Ask coverage per run. Refresh runs had zero repairs/stalls; init proposal
  generation emitted visible non-blocking repair telemetry (the git-URL init exhausted its proposal
  repair budget once), so the quality warning remains recorded rather than hidden or used to weaken
  the validator.
- 2026-08-05: Hardened normal and focused proposals prompts against the live-observed failure modes:
  validation diagnostics/staging paths are input-only, evidence files must pass an explicit `test -f`
  check, and shell variables must resolve to literal IDs/paths before markdown writes. Added prompt
  contract coverage; the strict validator and recovery budget remain unchanged.
- 2026-08-05: Final host preflight confirms all three provider CLIs are installed and the pinned
  PostHog checkout is now present, but the canonical release matrix still cannot run from this dirty
  worktree: the Open edX/OpenStack curated checkouts are absent. The disk guard itself passes (about
  10.6 GiB free versus the 5 GiB minimum), so no release matrix override or destructive cleanup was
  used; the Codex `regres-long` evidence remains the latest trusted live result.
- 2026-08-05: A full-provider diagnostic attempt classified both Claude and Qwen as
  `operational_host_preflight_failed` (`quota_or_permission`) before provider execution. A Codex-only
  rerun then exposed a precheck-only scheduler timeout in `TestRetryEndpointRejectsPlanAfterParentStagingDrifts`;
  its test deadline was widened from 8s to 20s without changing runtime behavior, and the targeted test
  passes 3/3. The interrupted rerun's Python precheck was reproduced verbosely as 267/267 passing in
  399.950s; a fresh Codex live rerun remains required after this precheck evidence.
- 2026-08-05: Fresh Codex `gpt-5.6-luna`/high `regres-long` matrix `matrix-20260805T174919Z` completed
  the curated path profile with backend hard-pass `1/1` and snapshot-backed frontend E2E `1/1` (nine
  screenshots plus Ask). Its quality report still exposes provider-authored repair telemetry rather
  than hiding it: `repair_attempts=5`, `repair_exhausted=1`, `stall_count=4`, with step-level findings
  on Architecture Home shape, collect manifest references and proposal actionability/linkage. The
  FTGO git-URL profile reached proposal generation but failed with `runtime_contract_failed` after a
  Codex repair wrote malformed JSON (`invalid character '\\'`); the run also recorded transient Codex
  WebSocket `403` reconnects. This is an honest matrix `FAIL`, not a validator relaxation.
- 2026-08-05: Added deterministic draft-manifest shape recovery: when a provider writes malformed JSON
  or unknown manifest fields, runtime atomically restores only the normative step skeleton and reruns
  the unchanged strict validators against provider-authored markdown. Added a regression test and
  focused/full `internal/runtime/providercommon` tests pass; no semantic content is synthesized. The
  post-fix deterministic DoD is green: `make contracts test lint build`, 183 UI tests, mock E2E `8/8`,
  Python `267/267`, docs-sync, embedded dist equality and `git diff --check`.
- 2026-08-06: A direct Codex `regres-long` diagnostic with the same pinned
  `gpt-5.6-luna`/`high` settings passed host and deterministic preflight, reached headless init and
  produced five collect shard surfaces before the trusted host disk fell below the 5 GiB guard. The
  matrix driver was terminated through its signal-safe process-tree cleanup; durable profile status is
  `infra_signal_terminated` with no provider verdict. The earlier `matrix-20260805T174919Z` failure
  and its malformed-manifest evidence remain the latest completed provider result; no validator
  relaxation or matrix-file edit was used.
- 2026-08-06: Continued the behavior-preserving UI maintenance slice by extracting the live diagnostics
  panel and run history table into `AnalysisDiagnosticsPanel.tsx` and `RunHistoryTable.tsx`. Existing
  rendering, callbacks, accessibility labels and test IDs remain unchanged; UI typecheck, 183 Vitest
  tests and all 8 deterministic mock E2E scenarios pass. The larger `StagePanels.tsx`/`styles.css`
  decomposition remains open and is not being hidden behind a visual rewrite.
- 2026-08-06: Extracted the selected-run domain map renderer and its model types into
  `ReviewDomainMap.tsx`; `StagePanels.tsx` now retains only review orchestration and pure map derivation.
  The extraction is behavior-preserving (same test IDs, labels, artifact navigation and empty/partial
  states). Direct UI typecheck and 183 Vitest tests pass, and the 8-scenario mock E2E suite remains green.
  The production bundle was regenerated and embedded dist equality holds; the local Go 1.20 toolchain
  cannot complete the final Go build because this repository requires `os.Root` from Go 1.24+.
- 2026-08-06: Tightened the proposal first-pass prompt without changing validation semantics: visible
  high/medium finding previews now include exact related IDs and evidence paths alongside the finding ID
  and severity. This gives the provider the fields needed for same-line actionable bullets before any
  repair is scheduled. Added a focused steppolicy regression test; it passes on Go 1.25.10.
- 2026-08-06: Re-ran the deterministic DoD with the repository-compatible temporary Go 1.25.10 toolchain:
  `make contracts test lint build` passed, alongside embedded UI equality, docs-sync, `git diff --check`,
  and mock E2E `8/8`. The canonical live gate is still intentionally not reported as passed because the
  trusted-machine disk guard has only ~3.7 GiB free while the retained canonical live checkout is
  ~6.3 GiB and the current matrix workspace is ~474 MiB.
- 2026-08-06: Direct Codex-only `regres-long` matrix
  `regres-long-posthog-ftgo-20260805T225634Z` completed both selected profiles with
  `ACP_CODEX_MODEL=gpt-5.6-luna` and `ACP_CODEX_REASONING_EFFORT=high`, but both ended as
  `runtime_contract_failed`. PostHog failed at `init.step4.proposals` after one focused repair and
  two stall signals; FTGO failed at `init.step2.asis_docs` after two focused repairs and three stall
  signals. Raw provider stdout shows malformed nested `python3 -c` shell quoting ending in
  `zsh: unmatched \"`, which explains the artifact stalls; Codex WebSocket 403s were followed by
  HTTPS fallback and are recorded as an operational provider signal, not a validator failure.
  Reports are retained under `/tmp/provenarch-test_arch_project/reports/` (matrix JSON/Markdown,
  profile TSV and per-profile execution reports).
- 2026-08-06: Hardened normal and focused `step2.asis_docs` and `step4.proposals` prompts with an
  explicit mechanical-write contract: before the complete artifact set exists, no Python/Node/awk/
  jq/eval, generated source strings or nested quote tricks; use direct literal single-quoted
  heredocs and write markdown before the manifest. Strict validators and repair budgets are
  unchanged. Focused steppolicy/providercommon/promptcontract/docsync tests pass; a live rerun is
  still required before marking the proposal-repair goal complete, and is deferred while the host
  remains below the canonical 5 GiB free-space gate.
- 2026-08-10: The follow-up trusted-machine Codex run isolated the remaining FTGO failure to shell
  interpolation in the as-is first-write sequence: the provider reported a manifest but did not
  materialize the referenced markdown under `draft_final_root`, so the unchanged validator correctly
  failed closed. The prompt now emits absolute single-quoted targets and forbids `write_root`/
  `draft_root` shell-variable interpolation for the first as-is write; strict manifest validation and
  recovery budgets are unchanged. Narrow steppolicy/promptcontract tests pass.
- 2026-08-10: Canonical `scripts/full-run-batch-matrix.sh` rerun `regres-long-ftgo-asis-fix-20260810T090004Z`
  (non-release diagnostic matrix with the curated FTGO `single-git_url` profile, exact Node 22.21.1,
  Go 1.25.10 and `gpt-5.6-luna`/`high`) passed backend hard-gates `1/1` and Codex frontend live smoke
  `1/1` through init → refresh. Init and refresh both had `warnings_count=0`, `repair_attempts=0`,
  `repair_exhausted=0`, `stall_count=0`, `partial_failure_count=0`, and `quality_alerts=0`; the matrix
  result is strict `PASS` with no blocking items. The earlier two-profile matrix remains retained as
  historical evidence (PostHog passed; FTGO exposed the now-fixed as-is quoting defect); release mode
  was not run because Open edX/OpenStack curated checkouts are unavailable on this host.
- 2026-08-10: Provisioned and SHA-verified all 11 curated Open edX/OpenStack repositories under
  `/tmp/provenarch-live-e2e`, created a clean validation snapshot, and ran the canonical direct
  `release-fast` matrix `release-fast-20260810T114248Z` without diagnostic overrides. All four
  profile/sweep combinations reached terminal classification but failed provider readiness before
  backend execution because Qwen reported `quota_or_permission: 0.19.11`; the retained verdict is
  `RELEASE BLOCKED` with `strict_pass_runs=0`, not a product runtime failure. The full release-provider
  goal remains open until Qwen quota/permission is restored on the trusted host.
- 2026-08-10: Rechecked Qwen readiness directly with the canonical artifact-smoke invocation; Qwen
  `0.19.11` returned API `403 permission_error`/usage-limit text and did not create the sentinel.
  This confirms the release blocker is still provider-side and must not be bypassed by running a
  Codex-only release matrix.

## EP-20260805-visual-alignment-pass

Status: active — retained scope; reconcile current implementation before selecting a new slice.

Next action: Reconcile the recorded open criterion with current code/status before selecting work: Decompose the remaining `App.tsx`/`StagePanels.tsx`/`styles.css` monoliths as a follow-up maintenance slice without changing the validated visual behavior.

### Context
The approved visual references are stored in the Codex visualization bundle from the design review:
`final-home.png`, `final-analyze-initial.png`, `final-analyze.png`, `final-architecture.png`,
`final-architecture-diagrams.png`, `final-changes.png`, `final-publish.png`, `final-settings.png`,
`final-init.png` and `final-mobile-architecture.png`. Current mock E2E captures show that the
implemented truth/shell slice still differs materially from those references in shared shell craft,
screen hierarchy, density and mobile behavior.

### Goals (must have)
- [x] Match the shared visual language: 64px top bar, 224px dark navigation, teal action system,
  compact status pills, restrained cards, and consistent typography/spacing.
- [x] Make Home, Analyze, Architecture, Changes and Publish read like the target task flows rather
  than internal pipeline dashboards.
- [x] Bring Init/Setup and Settings to the target form layouts without dropping existing validation
  or recovery behavior.
- [x] Verify desktop and mobile screenshots plus keyboard/focus and reduced-motion behavior.
- [x] Add deterministic happy-path visual coverage from initialization through refresh and full-workspace
  publication.
- [ ] Decompose the remaining `App.tsx`/`StagePanels.tsx`/`styles.css` monoliths as a follow-up
  maintenance slice without changing the validated visual behavior.

### Non-goals
- [ ] Do not change backend contracts, Git safety, run authority or recovery semantics.
- [ ] Do not remove legacy URLs, test IDs or deterministic mock scenarios.

### Screen deltas recorded
- [x] Shell: current utility buttons and initial-based nav differ from the target logo/Ask/icon
  header and grouped dark sidebar.
- [x] Home: current sparse status/attention layout lacks the target outcome hero, four-state strip,
  architecture mini-map and activity feed.
- [x] Analyze: current technical run detail dominates the viewport; target leads with initial/refresh
  outcome, metrics, pipeline and compact run history.
- [x] Architecture: current legacy model capture opens by default in mock flow; target is a three-pane
  Documents reader with sibling Diagrams/Model/Findings modes.
- [x] Changes: current review embeds a large artifact workbench; target leads with delta metrics,
  meaningful change list and a decision aside.
- [x] Publish: current recovery view is tall and implementation-heavy; target has an authoritative
  scope notice, grouped inventory and one commit card.
- [x] Init/Settings: current onboarding/readiness surfaces are technically correct but do not match
  the target step rail, form sections and settings shell.
- [x] Mobile: current pages expose filter/tool internals before the main document/task; target keeps a
  compact header, bottom nav and content-first reader.

### Progress log
- 2026-08-05: Target and current screenshots inspected; visual delta inventory recorded. Shared
  shell/token alignment is the first implementation slice.
- 2026-08-05: Implemented shared shell, Home outcome, Analyze launcher/outcome fallback, Documents-first
  Architecture reader/context, Changes delta/decision view, authoritative Publish inventory/commit card,
  Settings shell and Setup step rail. Added responsive bottom navigation, focus-safe controls and reduced
  motion defaults without changing backend/recovery semantics.
- 2026-08-05: Render QA: Home desktop/mobile, Analyze success/failure, Publish desktop/tablet/mobile,
  Setup recovery, Architecture model/mobile and recovery surfaces checked through deterministic mock E2E;
  all 8 mock scenarios passed. Typecheck, lint and production build passed; repository Python unittest
  discovery was started but the pre-existing install-test subprocess hung after the live contract tests,
  so it was interrupted and recorded as an environment validation blocker.
- 2026-08-05: Added `happy-path-mock`: one browser flow now starts initial analysis, waits for the
  authoritative result, refreshes the architecture, opens Documents and Diagrams, reviews a run-pinned
  semantic delta, and completes the full-workspace Git commit confirmation. The complete mock suite is
  now 8/8 scenarios with no browser console errors, critical axe violations or horizontal overflow.
- 2026-08-05: Extracted the global Ask, Git confirmation and brief-skip dialogs into `AppOverlays` so
  route rendering no longer owns cross-cutting modal composition. Behavior, accessibility labels and
  test IDs are unchanged; the remaining `App.tsx`/`StagePanels.tsx`/`styles.css` decomposition stays
  an explicit maintenance follow-up.
- 2026-08-06: Extracted the analysis live-diagnostics panel and run-history table into typed feature
  components, preserving the existing render contracts and test IDs. UI typecheck, 183 Vitest tests
  and deterministic mock E2E `8/8` pass; the remaining work is the larger StagePanels/styles split.
- 2026-08-06: Extracted pending-permission triage/table rendering into `PendingPermissionsTable.tsx`
  as the next mechanical seam. The permission recovery surface keeps its existing labels, test IDs,
  and retry guidance; typecheck remains green and the full mock suite is the next verification gate.
- 2026-08-06: Extracted pure analysis log/format selectors into
  `ui/src/features/analysis/analysisUtils.ts`; `StagePanels.tsx` keeps the same analysis render path
  while shedding another utility cluster. Node 22 typecheck, 183 Vitest tests, full mock E2E `8/8`
  and `git diff --check` pass. The remaining broad CSS and route-component split is still intentionally
  separate from this behavior-preserving maintenance slice.
- 2026-08-06: Re-ran `make contracts test lint build` on the final maintenance tree with Go 1.25.10
  and Node 22.21.1: contracts, full Go suite, Python `267/267`, UI `183/183`, shellcheck/typecheck,
  production build and embedded UI equality all pass. Vite retains only the existing large-chunk warning.
- 2026-08-10: Extracted the pure publication and workflow-state selectors from `App.tsx` into
  `ui/src/lib/appDerived.ts`, with regression coverage for unknown Git state, active-run routing and
  question counting. The render path and recovery semantics are unchanged; Node 22 typecheck, Vitest
  `186/186`, production build and deterministic mock E2E `8/8` pass. The broader App/StagePanels/CSS
  decomposition remains intentionally open.
- 2026-08-10: Extracted proposal review derivation, package grouping, blocker guidance and tab labels
  from `StagePanels.tsx` into `ui/src/features/proposals/proposalUtils.ts`, with focused tests for
  package completeness and stable labels. This is a pure mechanical split: the proposal UI, test IDs
  and publish handoff are unchanged. Node 22 typecheck, Vitest `189/189`, production build and mock
  E2E `8/8` pass; the remaining monolith/CSS split is still open.
- 2026-08-10: Extracted Ask/QA provisional-run, history merge, provider label and failure-guidance
  selectors into `ui/src/features/qa/qaUtils.ts`, with focused tests for status normalization, capped
  history and actionable recovery copy. The Ask panel and recovery DOM remain unchanged. Node 22
  typecheck, Vitest `192/192` and mock E2E `8/8` pass; the remaining monolith/CSS split is still open.
- 2026-08-10: Extracted Review queue/trust selectors and Publish inventory/gate selectors into
  `ui/src/features/review/reviewUtils.ts` and `ui/src/features/publish/publishUtils.ts`. These pure
  seams preserve selected-run authority, full-workspace Git gating, labels and test IDs while reducing
  `StagePanels.tsx` to 4,868 lines. Node 22 typecheck, Vitest `198/198`, production build and mock
  E2E `8/8` pass; the remaining component/CSS decomposition is still open.
- 2026-08-10: Extracted the analysis timeline, shard grouping, artifact-pair classification and
  visual tone selectors into `ui/src/features/analysis/analysisViewModels.ts`, with focused tests for
  terminal/active timelines, provider and warning semantics, artifact completeness and diff tones.
  The existing Analysis DOM, test IDs and recovery behavior are unchanged; `StagePanels.tsx` is now
  4,595 lines. Node 22 typecheck, Vitest `202/202`, production build and mock E2E `8/8` pass.
- 2026-08-10: Extracted provider stream summarization, shard counter parsing and artifact-handoff
  stall detection into `ui/src/features/analysis/providerStreamUtils.ts`. The live diagnostics panel
  keeps the same telemetry wording and actions while `StagePanels.tsx` drops to 4,481 lines; focused
  provider-stream tests pass (`7/7` across the two analysis selector suites).
- 2026-08-10: Extracted the Setup/Init lifecycle route into `ui/src/components/SetupRoute.tsx`,
  keeping source, readiness, charter, review and runtime-profile callbacks typed at the route boundary.
  The setup runtime distinction between effective source settings and persisted runner settings is
  preserved explicitly. `App.tsx` is now 1,222 lines; full UI Vitest `205/205` and mock E2E `8/8`
  pass after the slice.
- 2026-08-10: Split the shared design tokens into `ui/src/styles/tokens.css` and loaded them from
  `styles.css` without changing the two-stage palette cascade. Token contract tests now inspect the
  canonical token source plus feature CSS; full UI Vitest `205/205`, mock E2E `8/8` and production
  build pass with embedded dist equality.
- 2026-08-10: Extracted selected-run model/domain map derivation into
  `ui/src/features/review/reviewDomainMapUtils.ts`, including explicit partial/blocker states and
  edge parsing. The Review DOM and artifact navigation remain unchanged; focused map tests pass and
  `StagePanels.tsx` drops to 4,301 lines.
- 2026-08-10: Extracted Source validation/recovery selectors and draft diagnostics into
  `ui/src/features/setup/sourceUtils.ts`, with focused coverage for server diagnostics, incomplete
  drafts and workspace-manifest fallbacks. The Source recovery DOM, labels and validation callbacks
  remain unchanged; `StagePanels.tsx` drops to 4,172 lines. Focused verification after this slice
  is green: typecheck, source-utils tests `3/3`, UI Vitest `32 files / 210 tests`, mock E2E `8/8`,
  `make lint`, `make build`, embedded UI equality and `git diff --check`; the immediately preceding
  full deterministic DoD also passed `make test`.
- 2026-08-10: Extracted charter baseline diagnostic selection and prompt-usage/fallback copy into
  `ui/src/features/setup/charterUtils.ts`, preserving the Charter recovery DOM and artifact
  selection semantics. `StagePanels.tsx` drops to 4,079 lines; focused charter tests `3/3`, full UI
  Vitest `33 files / 213 tests`, mock E2E `8/8`, `make lint`, `make build`, embedded UI equality and
  `git diff --check` all pass.
- 2026-08-10: Re-ran the complete deterministic closure on the final setup decomposition: contracts,
  full Go suite, Python `267/267`, UI Vitest `33 files / 213 tests`, lint/typecheck, production build,
  embedded UI equality and `git diff --check` all pass. The only unclosed qualification item is the
  trusted release provider matrix, which remains externally blocked by Qwen quota/permission.
- 2026-08-10: Extracted analysis failure guidance and retained-evidence summaries into
  `ui/src/features/analysis/analysisRecoveryUtils.ts`, preserving all timeout, contract, provider,
  permission and restart-reconciliation branches. `StagePanels.tsx` drops to 4,045 lines; focused
  tests `2/2`, full UI Vitest `34 files / 215 tests`, mock E2E `8/8`, lint, build, embedded equality
  and `git diff --check` pass.
- 2026-08-10: Direct Qwen artifact-smoke was repeated after the new slice. Qwen `0.19.11` still
  returns API `403 permission_error` for billing-cycle usage limit and does not create the sentinel;
  release qualification remains provider-blocked and no Codex-only release override was used.
- 2026-08-10: Extracted readiness workspace-health tone/label mapping into
  `ui/src/features/setup/readinessUtils.ts`, keeping loading, error, absent and report-severity
  states explicit. `StagePanels.tsx` drops to 4,017 lines; focused tests `2/2`, full UI Vitest
  `35 files / 217 tests`, mock E2E `8/8`, lint, build, embedded equality and `git diff --check` pass.
- 2026-08-10: Final exact-tree `make test` after the analysis/readiness seams passed contracts, the
  full Go suite, Python `267/267` and UI `35 files / 217 tests`; no deterministic regression is
  present. Release qualification remains the sole missing gate because Qwen readiness is external.
- 2026-08-10: Extracted the publish gate checklist renderer into
  `ui/src/features/publish/PublishGateSection.tsx`, preserving gate item tones, labels, test IDs and
  empty states. `StagePanels.tsx` drops to 3,985 lines; focused gate tests `2/2`, full UI Vitest
  `36 files / 219 tests`, mock E2E `8/8`, lint, build, embedded equality and `git diff --check` pass.
- 2026-08-10: Final exact-tree `make test` after the publish gate extraction passed contracts, full
  Go, Python `267/267` and UI `36 files / 219 tests`. No deterministic regression is present;
  release qualification is still blocked only by external Qwen usage quota.
- 2026-08-10: Extracted the shared `GitDiffView` renderer into
  `ui/src/components/GitDiffView.tsx` and moved `countMarkdownItems` into review selectors. The
  same server-authoritative diff UI is now reused by Analysis, Review and Publish; `StagePanels.tsx`
  drops to 3,893 lines. Focused GitDiff tests `2/2`, full UI `37 files / 221 tests`, mock E2E `8/8`,
  lint, build, embedded equality and `git diff --check` pass.
- 2026-08-10: Final exact-tree `make test` after the shared diff extraction passed contracts, full
  Go suite, Python `267/267` and UI `37 files / 221 tests`.
- 2026-08-10: Extracted the Review Queue renderer into
  `ui/src/features/review/ReviewQueuePanel.tsx`, preserving selected-item semantics, aria-current,
  bounded list rendering and empty-state copy. `StagePanels.tsx` drops to 3,854 lines; focused queue
  tests `2/2`, full UI `38 files / 223 tests`, mock E2E `8/8`, lint, build, embedded equality and
  `git diff --check` pass.
- 2026-08-10: Extracted the Ask failure recovery surface into
  `ui/src/features/qa/QAFailureRecovery.tsx`, preserving canceled/reconciled labels, retry gating,
  audit links and warning copy. `StagePanels.tsx` drops to 3,781 lines; focused QA tests `2/2`, full
  UI `39 files / 225 tests`, mock E2E `8/8`, lint, build, embedded equality and `git diff --check`
  pass. Stabilized the historical Architecture navigation assertion with an explicit 5s async
  bootstrap timeout; the full UI suite now completes deterministically.
- 2026-08-10: Final exact-tree `make test` after Q&A recovery extraction and navigation-test
  stabilization passed contracts, full Go suite, Python `267/267` and UI `39 files / 225 tests`.
- 2026-08-10: Extracted the full Review evidence workbench into
  `ui/src/features/review/ReviewEvidenceWorkbench.tsx`, keeping artifact explorer, evidence
  preview, server-authoritative diff mode, citation coverage and trust summary behavior intact.
  `StagePanels.tsx` drops to 3,564 lines; the focused workbench test and typecheck pass. Full UI
  Vitest is now `40 files / 226 tests`, and the deterministic mock E2E gate remains `8/8`.
- 2026-08-10: Final exact-tree `make test` after the workbench extraction passed contracts, the
  full Go suite, Python `267/267` and UI `40 files / 226 tests`; `make lint`, `make build`, embedded
  UI equality and `git diff --check` are also green. The only remaining qualification gap is the
  external Qwen-gated release provider matrix.
- 2026-08-10: Removed the isolated `ProposalPackageRecoveryPanel` from `StagePanels.tsx` into
  `ui/src/features/proposals/ProposalPackageRecoveryPanel.tsx`, preserving blocker copy, artifact
  handoff and Publish recovery actions with a focused regression. `StagePanels.tsx` is now 3,473
  lines; typecheck, focused test, full UI `41 files / 227 tests`, mock E2E `8/8`, lint, build,
  embedded equality and `git diff --check` pass.
- 2026-08-10: Final exact-tree `make test` on the current decomposition passed contracts, the full
  Go suite, Python `267/267` and UI `41 files / 227 tests`. No deterministic regression remains;
  release qualification is still externally blocked only by Qwen's billing-cycle quota/permission.
- 2026-08-10: Extracted the analysis step review room (step cards, artifacts, logs, evidence and
  step-scoped diff) into `ui/src/features/analysis/AnalysisStepReview.tsx`, preserving its existing
  callbacks, test IDs and empty-state copy. `StagePanels.tsx` is now 3,336 lines; focused and full
  UI tests pass (`42 files / 228 tests`) and mock E2E remains `8/8`.
- 2026-08-10: Repeated the exact Qwen `0.19.11` artifact smoke. CLI exits zero but returns API
  `403 permission_error` for the billing-cycle usage limit and creates no sentinel; the canonical
  Claude/Qwen/Codex release matrix therefore remains blocked and was not bypassed with Codex-only.
- 2026-08-10: Final exact-tree closure after `AnalysisStepReview` passed `make test` (contracts,
  full Go suite, Python `267/267`, UI `42 files / 228 tests`), `make lint`, `make build`, embedded
  UI equality and `git diff --check`. Current line counts are `App.tsx 1,224`, `StagePanels.tsx
  3,336`, `styles.css 5,743`.
- 2026-08-10: Extracted the complete Publish route container into
  `ui/src/features/publish/PublishStagePanel.tsx` and retained the `StagePanels` re-export for
  compatibility. Full-workspace diff truth, gate rendering, artifact preview, commit and branch
  actions are unchanged. `StagePanels.tsx` is now 2,881 lines; focused/full UI `43 files / 229
  tests`, mock E2E `8/8`, `make lint`, `make build`, embedded equality and `git diff --check` pass.
- 2026-08-10: Final exact-tree `make test` after Publish extraction passed contracts, full Go,
  Python `267/267` and UI `43 files / 229 tests`. No deterministic regression remains.
- 2026-08-10: Extracted the full Ask/Q&A route container into
  `ui/src/features/qa/AskStagePanel.tsx`, retaining the `StagePanels` re-export and all async
  request gates, retry/reconciliation behavior, citations and proposal handoff. `StagePanels.tsx`
  is now 2,387 lines; focused/full UI `44 files / 230 tests` and mock E2E `8/8` pass.
- 2026-08-10: Final exact-tree `make test` after Ask extraction passed contracts, full Go, Python
  `267/267` and UI `44 files / 230 tests`; `make lint`, `make build`, embedded equality and
  `git diff --check` are green. Only the external Qwen-gated release matrix remains unqualified.
- 2026-08-10: Extracted the Setup source/readiness route block into
  `ui/src/features/setup/SetupStagePanels.tsx`, retaining `StagePanels` compatibility re-exports
  and the existing SetupRoute behavior, recovery states, validation and test IDs. `StagePanels.tsx`
  is now 1,515 lines; focused/full UI `45 files / 231 tests` and mock E2E `8/8` pass.
- 2026-08-10: Final exact-tree closure after Setup extraction passed contracts, full Go suite,
  Python `267/267` and UI `45 files / 231 tests`; `make lint`, `make build`, embedded UI equality
  and `git diff --check` are green. Current line counts are `App.tsx 1,224`, `StagePanels.tsx
  1,515`, `styles.css 5,743`. The canonical release provider matrix remains externally blocked by
  Qwen `0.19.11` billing-cycle `403 permission_error`; no Codex-only release bypass was used.
- 2026-08-10: Fresh resumed-goal audit repeated the exact Qwen artifact smoke; CLI `0.19.11`
  still exits without a sentinel and returns the same billing-cycle `403 permission_error`. The
  current UI tree remains green on Vitest `45 files / 231 tests` and mock E2E `8/8`, so no local
  regression or unfinished deterministic fix remains. Release qualification is still waiting on
  provider quota/permission recovery.

## EP-20260805-ui-bug-cleanup

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Повторить полный release-provider matrix (Claude/Qwen/Codex) на trusted host, когда соответствующие CLI и ресурсные prerequisites доступны.

### Context
После визуального pass нужно было проверить не только рендеринг, но и достоверность действий: часть
новых экранов содержала неработающие настройки, скрытый дублирующий Home DOM и слишком сильные
утверждения о состоянии Git.

### Goals (must have)
- [x] Найти и исправить подтверждённые баги в UI-навигации, Home primary action и Publish Git gate.
- [x] Удалить безопасный скрытый дубликат Home architecture outcome и обновить его E2E-селектор.
- [x] Защитить route parser от malformed percent-encoded path segments.
- [x] Добавить регрессии для Settings navigation, clean workspace и malformed route.
- [x] Прогнать typecheck, unit tests, mock E2E, Go vet/test, lint и production build.
- [x] Повторить canonical Node 22 UI bundle freshness/determinism gate на trusted runtime.
- [ ] Повторить полный release-provider matrix (Claude/Qwen/Codex) на trusted host, когда
  соответствующие CLI и ресурсные prerequisites доступны.

### Non-goals
- Не менять API/schema/runtime semantics и не удалять legacy routes или test IDs.
- Не трогать unrelated пользовательские изменения и generated embedded assets вне обычного build.

### Progress log
- 2026-08-05: Settings теперь ведёт к реальным секциям, а действие переименовано в `Edit in Setup`,
  Home review CTA открывает run-pinned Changes, Publish не утверждает authoritative scope до загрузки
  и блокирует пустой/no-op commit на уровне UI и Git hook.
- 2026-08-05: Удалён скрытый promoted Home outcome, добавлена безопасная обработка malformed URL path,
  регрессии расширены до 177 UI tests.
- 2026-08-05: Проверки прошли: 22 Vitest files / 180 tests, mock E2E 8/8, TypeScript, Go vet и
  targeted Go tests, `make lint`, `make build`, `git diff --check`.
- 2026-08-05: Canonical matrix preflight ran the exact Node 22.21.1 bundle freshness/typecheck/lint/build
  closure; `matrix-20260805T113030Z` then passed the Codex frontend snapshot gate on both selected
  profiles. No stale embedded asset or nondeterministic bundle was reported.
- 2026-08-05: Extracted review/publish artifact grouping and filter semantics into typed
  `ui/src/lib/artifactFilters.ts` with focused tests; `StagePanels.tsx` keeps the same rendering and
  test IDs while dropping another pure helper layer.
- 2026-08-05: Added `appDerived.ts` for pure selected-run issue derivation and completed the safe
  mechanical extraction of cross-cutting overlays plus artifact filters. `App.tsx` is now 1,270 lines;
  remaining `StagePanels.tsx`/`styles.css` decomposition is still behavior-preserving follow-up work.

## EP-20260804-ui-truth-and-document-centric-flow

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Repeat the validated happy path in a full release-provider matrix once the remaining provider CLIs are available on the trusted host.

### Context
The current React console has strong recovery and Git safety primitives, but its workflow state can
present an unknown Git diff as clean, Publish can enter with a selected-run diff even though commit
is workspace-wide, and historical Changes can pair a selected run with the promoted-current semantic
comparison. The approved UX also needs an explicit initial-vs-refresh flow, a first-class Settings
surface and a documents-first Architecture workspace.

### Goals (must have)
- [x] Make Git/publication state fail-closed: `loading`/`unknown` never render as `clean`.
- [x] Make Publish inventory and copy authoritative for full-workspace Git mutations.
- [x] Prevent historical Changes from rendering a comparison whose run identity is not selected.
- [x] Add a run-pinned review contract for initial and refresh analysis outcomes.
- [x] Recompose the UI around Home, Analyze, Architecture, Changes and Settings while preserving
  existing recovery, evidence and Git confirmation behavior.
- [x] Add deterministic happy-path E2E coverage from initialization through full-workspace publish.
- [x] Complete the canonical trusted-machine live refresh matrix and record its provider-backed
  verdict after the pinned curated checkout passes preflight.
- [ ] Repeat the validated happy path in a full release-provider matrix once the remaining provider
  CLIs are available on the trusted host.

### Non-goals
- [ ] No new runtime providers, hosted mode or broad backend rewrite.
- [ ] No filename-based domain inference for the Architecture document tree.
- [ ] No replacement of the existing ReactFlow/ELK model map, EvidenceViewer or Git fingerprint
  confirmation.
- [ ] No destructive migration of existing URLs; legacy route aliases remain supported.

### Approach
1) Deliver a behavior-only P0 truth pass for workflow publication and historical comparison.
2) Add the run-pinned review contract with schema, validator, fixture, API and UI synchronization.
3) Split the App/StagePanels/CSS monoliths into behavior-preserving feature seams.
4) Introduce the product shell, first-class Settings and shared Setup/Settings view models.
5) Make Architecture documents-first, then reshape Analyze and Changes around initial/refresh
   decisions.
6) Finish Publish polish, responsive/accessibility checks and the deterministic happy path.

### Files expected to change
- First slice: `ui/src/lib/workflowState.ts`, `ui/src/lib/workflowState.test.ts`, `ui/src/App.tsx`,
  `ui/src/components/StagePanels.tsx`, focused UI tests.
- Contract slice: `schemas/`, `docs/spec/API_SPEC.md`, `docs/APPENDIX_SCHEMAS.md`, backend review
  handlers, `ui/src/lib/appContracts.ts`, fixtures and contract tests.
- Composition/UX slices: `ui/src/components/ProductShell.tsx`, `ProductPages.tsx`,
  `KnowledgePage.tsx`, new feature containers/hooks, styles and E2E specs.

### Acceptance criteria
- [x] A missing, loading or failed full Git diff renders `Unknown`/`Loading`/`Blocked`, never `Clean`.
- [x] Publish loads full workspace scope by default and clearly states that commit is workspace-wide.
- [x] A historical run cannot display a current comparison unless identities match.
- [x] Initial analysis and refresh analysis have distinct outcome copy and review payloads.
- [x] Settings is reachable without opening Readiness; Setup remains a guided lifecycle flow.
- [x] Architecture opens on Documents, with diagrams/model/findings as explicit sibling modes.
- [x] Typecheck, unit tests, UI E2E, accessibility checks and repository DoD pass for each slice.

### Risks
- Existing tests encode selected-run Publish wording and will need deliberate migration to the new
  full-workspace semantics.
- A correct historical semantic review may require an additive backend snapshot contract; the UI
  must fail closed while that contract is unavailable.
- Large component files make broad edits regression-prone, so decomposition precedes visual polish.

### Progress log
- 2026-08-04: Code/UI audit completed; no product files changed.
- 2026-08-04: P0 truth pass implemented: publication loading/unknown states fail closed, Publish
  loads authoritative full-workspace Git inventory, and selected-run comparison identity is
  fail-closed. Focused UI tests and mock recovery E2E pass.
- 2026-08-04: Added additive run-pinned `review-summary.review` contract for initial/refresh runs,
  with snapshot authority, semantic/document deltas, runtime identity and deterministic counts;
  synchronized API docs, fixture and Go/TypeScript types/tests.
- 2026-08-05: Added first-class `/settings` with shared runtime profile controls and explicit
  workspace/repository/scope/Git/diagnostic sections; renamed primary Analyze navigation while
  retaining `/runs` and existing test IDs. Architecture now defaults to a URL-backed Documents
  reader with Diagrams, Model and Findings sibling modes; legacy map/catalog/flows/evidence aliases
  remain readable. Updated architecture, README and stakeholder docs.
- 2026-08-05: Full deterministic checks passed with explicit local Go 1.25.10, Node 25.9.0 and
  Python 3.10.8 binaries: contracts, Go and Python tests (266), UI tests (175), `make lint`,
  `make build`, and all 8 mock E2E scenarios. The larger App/StagePanels decomposition
  remains a follow-up slice; no behavior was changed for that non-goal.
- 2026-08-05: Added the deterministic happy-path browser flow and folded it into `scripts/ui-mock-e2e.sh`;
  all 8 mock E2E scenarios pass, including init → refresh → run-pinned Changes → full-workspace Publish.
- 2026-08-05: Canonical non-release `regres-long` matrix `matrix-20260805T113030Z` passed `2/2`
  selected Codex profiles (path and git URL), including live init→refresh and frontend snapshot E2E;
  provider-backed reports are stored under `/tmp/provenarch-test_arch_project/reports/`.
- 2026-08-05: Extracted cross-cutting Ask/Git/brief dialogs into `ui/src/components/AppOverlays.tsx`
  and moved the live run-error copy helper out of `App.tsx`; exact Node 22 typecheck/tests/lint,
  production build, embedded-bundle equality and mock E2E 8/8 pass. The larger StagePanels/CSS split
  remains intentionally open as a maintenance follow-up rather than a risky visual rewrite.

## EP-20260804-runtime-model-selection

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Compare the pinned model's artifact quality against the former baseline in the full release provider matrix before closing the migration plan.

### Context
At slice start, the runtime could select a provider per pipeline step, but model selection was not
yet a product contract. Only `codex-code` read `ACP_CODEX_MODEL` and `ACP_CODEX_REASONING_EFFORT`
directly in its adapter. Ordinary runs inherited Codex CLI defaults when these variables were absent,
while the live E2E harness pinned `gpt-5.5` with `xhigh` in several scripts and tests.

The desired behavior is provider-scoped and explicit: an unconfigured provider uses its own native
default without ACP guessing or passing a model flag; an operator may persist a model and supported
effort in `workspace.yaml`; trusted live E2E may override that profile for reproducible evidence.
The first live Codex baseline requested for this contract is `gpt-5.6-luna` with effort `high`.
Official OpenAI model guidance identifies Luna as the efficient high-volume GPT-5.6 tier and lists
`high` among its supported efforts.

### Goals (must have)
- [x] Add optional model selection for all three existing headless providers without changing their
  native defaults when no override is configured.
- [x] Add optional effort selection where the provider CLI supports it, with fail-fast capability
  validation and actionable errors.
- [x] Make persisted, effective and source values visible and editable through the runtime API and
  UI, including an explicit `Provider default` state.
- [x] Snapshot the resolved provider model profile when a run is accepted so queued and running
  work cannot change model because `workspace.yaml` or process env changes mid-run.
- [x] Complete the trusted-host live E2E Codex migration check for the `gpt-5.6-luna/high` pin and
  accept the resulting preflight/release evidence; the canonical defaults and evidence fields are
  already migrated in code.
- [ ] Compare the pinned model's artifact quality against the former baseline in the full release
  provider matrix before closing the migration plan.
- [x] Keep required CI provider-free and deterministic; validate live compatibility only through
  the existing trusted-machine harness and runbook.

### Non-goals
- No new runtime providers, hosted model API, remote model catalog or automatic availability
  discovery.
- No per-step model overrides in the first slice. Steps continue to select a provider; all steps
  using one provider share that provider's resolved model profile for the run.
- No automatic latest-model migration, cost/quality router, fallback model, retry-on-another-model
  or silent downgrade when a model is unavailable.
- No hardcoded allowlist of model IDs. Model identifiers remain opaque provider-owned strings;
  only provider capabilities and effort values are validated.
- No GPT-5.6 Pro mode, persisted reasoning, explicit prompt caching, programmatic tool calling or
  multi-agent behavior as part of model selection.
- No claim that ACP knows the concrete model behind a provider native default. ACP reports
  `provider_default` until the provider returns trustworthy observed-model metadata.

### Proposed contract

`workspace.yaml` gains an optional provider-scoped profile:

```yaml
runtime:
  profile:
    providers:
      codex-code:
        model: gpt-5.6-luna
        effort: high
      claude-code:
        model: claude-sonnet-4-6
        effort: high
      qwen-code:
        model: example-provider-model
```

The block is illustrative, not a default workspace template. A normal workspace may omit
`runtime.profile.providers` entirely.

Resolution is performed independently for `model` and `effort`:

1. provider-specific process override;
2. `workspace.yaml.runtime.profile.providers.<provider>.<field>`;
3. provider native default, represented as `provider_default` and implemented by omitting the CLI
   argument.

Initial process overrides remain provider-specific so a global flag cannot leak across mixed
step providers:

- `ACP_CLAUDE_MODEL`, `ACP_CLAUDE_EFFORT`;
- `ACP_QWEN_MODEL`;
- existing `ACP_CODEX_MODEL`, `ACP_CODEX_REASONING_EFFORT`.

An unset or whitespace-only value is not forwarded to the provider. In the persisted API patch, an
empty string clears that field and prunes empty provider/profile objects, matching current runtime
profile patch behavior. No generic `--runtime-model` CLI flag is added in the first slice; a future
provider-qualified CLI syntax can be considered only if editing YAML and the API are insufficient.

Initial adapter capability matrix:

| Provider | Model argument | Effort argument | Accepted effort values |
| --- | --- | --- | --- |
| `claude-code` | `--model` | `--effort` | `low`, `medium`, `high`, `max` |
| `qwen-code` | `--model` | unsupported | none |
| `codex-code` | `--model` | `-c model_reasoning_effort=...` | `none`, `low`, `medium`, `high`, `xhigh`, `max` |

The workspace schema accepts a trimmed, non-control model string without enumerating model names.
Effort without an explicit model is allowed and applies to the provider's native default; provider
startup remains fail-fast if that concrete default rejects the requested effort.

### Delivery slices

#### Slice 1 — contract, resolver and adapters

1. Add `runtime.profile.providers` structs, schema, rendering/pruning and semantic validation.
2. Add a provider-model resolver returning `persisted`, `effective` and `source` per provider and
   field. Resolve process env once when the run is accepted rather than reading env inside a runner.
3. Carry the resolved model/effort as internal `runtime.Task` data and append arguments in each
   adapter. Remove direct `os.Getenv` model reads from `codex-code`.
4. Snapshot effective values into run state and structured runtime-start diagnostics. Do not expand
   the versioned `RuntimeExecution` artifact in this slice unless audit requirements demonstrate
   that run state and preflight evidence are insufficient.

#### Slice 2 — API and UI

1. Add `GET/PUT /api/runtime/models` with provider-keyed `persisted`, `effective`, `source` and
   `capabilities` payloads; also expose the read model in aggregate `/api/runtime/profile` and run
   envelopes.
2. Add provider cards to runtime settings: free-text model, capability-aware effort selector,
   effective value/source, clear action and `Provider default` placeholder.
3. Disable effort editing for `qwen-code`; never offer a stale hardcoded model dropdown. A future
   catalog may enhance the text input without becoming required for execution.
4. Ensure an env override remains visibly effective even when a different persisted value exists,
   and explain that clearing the env is required before the workspace value can take effect.

#### Slice 3 — live E2E baseline migration

1. Change the canonical Codex live defaults in `live-e2e-plan.py`, batch/matrix harnesses and
   preflight writer to `gpt-5.6-luna` and `high`.
2. Keep the pin Codex-only. Claude and Qwen continue to inherit their native defaults unless their
   own explicit profile/env values are supplied.
3. Make preflight evidence include requested model, effort and source, and fail before a matrix run
   when the installed Codex CLI cannot accept the requested combination.
4. Run a narrow non-release Codex smoke on a trusted host first. If it passes, run the canonical
   release profiles through `scripts/full-run-batch-matrix.sh`; compare artifact-quality and SWE UX
   assessments with the previous `gpt-5.5/xhigh` baseline before accepting the migration.
5. Keep rollback operational and explicit: the same harness can be invoked with
   `ACP_CODEX_MODEL=gpt-5.5` and `ACP_CODEX_REASONING_EFFORT=xhigh` without editing canonical
   matrices or curated repository lists.

### Files expected to change
- Workspace contract: `internal/workspace/manifest.go`, `schemas/workspace.schema.json`,
  `docs/spec/WORKSPACE_SPEC.md`, `examples/workspace.example.yaml`, workspace fixtures and tests.
- Resolution/orchestration: new `internal/runtime/provider_models.go` and tests,
  `internal/runtime/runtime.go`, `internal/orchestrator/service_runs.go`,
  `internal/orchestrator/orchestrator.go`, `internal/orchestrator/runtime_task_executor.go`, run-state
  persistence and API envelope tests.
- Provider adapters: `internal/runtime/claudecode/*`, `internal/runtime/qwencode/*`,
  `internal/runtime/codexcode/*`.
- Runtime mutation/API: `internal/runtimeprofile/patch_service.go`, `internal/api/server.go` and
  focused tests.
- UI: `ui/src/lib/appContracts.ts`, `ui/src/hooks/useRuntimeSettings.ts`, runtime settings panels,
  styles and component tests.
- Live evidence: `scripts/live-e2e-plan.py`, `scripts/full-run-batch.sh`,
  `scripts/full-run-batch-matrix.sh`, `scripts/write-batch-preflight.py` and their Python tests.
- Docs/rationale: `docs/spec/API_SPEC.md`, `docs/spec/PIPELINE_SPEC.md`,
  `docs/ARCHITECTURE.md`, `docs/APPENDIX_SCHEMAS.md`, `docs/TESTING_STRATEGY.md`,
  `docs/RELEASE_LIVE_E2E_RUNBOOK.md`, `docs/STAKEHOLDER_DOC.md` and a focused ADR under
  `docs/adr/`.

### Mandatory tests and fixtures
- Manifest/schema round trip for omitted profiles, each provider profile and cleared empty values.
- Resolver table tests for env > workspace > provider default, independent model/effort sources,
  invalid effort, unsupported Qwen effort and whitespace handling.
- Adapter argv tests proving omitted config adds no model/effort arguments and explicit config adds
  exactly one correctly quoted provider argument pair.
- Run lifecycle test proving a queued run keeps the model snapshot captured at acceptance time.
- API GET/PUT/clear/conflict/strict-JSON tests and aggregate/run-envelope serialization tests.
- UI tests for provider default, workspace value, env override, unsupported effort and save/reset.
- Live plan/preflight tests for `gpt-5.6-luna/high`, operator overrides and evidence serialization.
- Full completed-slice DoD: `make contracts`, `make test`, `make lint`, `make build`.

### Acceptance criteria
- [ ] With no provider model profile and no provider model env, adapter argv contains no model or
  effort override and the API/UI reports `provider_default`.
- [ ] A persisted Codex `gpt-5.6-luna/high` profile reaches every Codex task in a run and does not
  affect Claude or Qwen tasks.
- [ ] Provider-specific env values override persisted fields, with source visible in API/UI and run
  diagnostics.
- [ ] Unsupported or malformed configuration fails before provider execution with a stable,
  actionable error; no silent fallback occurs.
- [ ] Required CI uses fake/unit fixtures only and performs no live provider or model-catalog call.
- [ ] Trusted-host Codex smoke and canonical live gate capture `gpt-5.6-luna/high`; release readiness
  still requires the existing strict machine verdict and accepted human assessments.
- [ ] Schema/spec/examples/fixtures/validators/tests/ADR and behavior docs are synchronized, and the
  full DoD passes.

### Risks
- Provider CLIs may change accepted effort values or model access independently. Capability errors
  must be fail-fast and the runbook must keep an operator override/rollback path.
- Provider native defaults are mutable and may not be observable. ACP must not invent a model name
  for default-mode runs or compare them as if they were pinned.
- A provider-level model may be too coarse for future cost-tiering by step. Per-step overrides are a
  follow-up contract only after the simpler profile is evaluated in real runs.
- Free-text model IDs can be mistyped. Preflight and clear errors are safer for MVP than a stale
  built-in catalog.
- Changing the release Codex baseline affects cost, latency and artifact quality. Acceptance depends
  on measured live evidence, not only CLI compatibility.

### Progress log
- 2026-08-04: Inventoried workspace schema, runtime resolver, orchestration, API/UI and live E2E
  surfaces. Confirmed model selection is currently Codex-env-only and the live pin is duplicated as
  `gpt-5.5/xhigh`.
- 2026-08-04: Verified installed provider CLI capabilities: Claude supports `--model/--effort`,
  Qwen supports `--model`, and Codex supports `--model` plus config overrides.
- 2026-08-04: Checked current official OpenAI model guidance: `gpt-5.6-luna` is a valid model ID,
  supports `high`, and GPT-5.6 migration guidance recommends comparing the existing effort with one
  level lower on representative workloads.
- 2026-08-04: Implemented the workspace contract, resolver, adapter propagation, run/history snapshot,
  API/UI settings surface, provider fixtures, and schema/docs/ADR synchronization.
- 2026-08-04: Migrated canonical live Codex defaults and preflight evidence to `gpt-5.6-luna/high`;
  rollback remains available through explicit `gpt-5.5/xhigh` environment overrides.
- 2026-08-04: Contract validation, focused Python harness tests, and UI TypeScript checks pass.
  Go validation remains blocked by the host's Go 1.20 toolchain while the repository uses `os.Root`.
- 2026-08-05: Trusted-machine non-release matrix `matrix-20260805T113030Z` captured the exact
  `gpt-5.6-luna` model and `high` reasoning effort in Codex backend commands across both path and
  git-URL profiles; preflight, backend hard-pass and frontend snapshot E2E passed `2/2`.

## EP-20260803-v0.1.13-unqualified-prerelease

Status: review — recorded owner/admin or merge acceptance remains open.

Next action: Resolve the recorded open criterion: Merge release metadata through PR before creating the tag.

### Context
PR #207 is merged into `main` and packages the outcome-first product flow, Architecture Explorer,
structured progress and dependency-aware retry. The repository owner previously directed releases
to proceed without live-provider matrix runs. This release therefore needs an exact-tag waiver and
must remain explicitly unqualified instead of claiming canonical `RELEASE READY`.

### Goals (must have)
- [x] Select the next SemVer prerelease tag `v0.1.13` from the published `v0.1.12` line.
- [x] Record user-visible release notes and verification evidence in `CHANGELOG.md`.
- [x] Add a verifier-compatible exact-tag owner waiver for `v0.1.13`.
- [x] Pass provider-free DoD and exact-tag waiver verification.
- [ ] Merge release metadata through PR before creating the tag.

### Non-goals
- No canonical live-provider matrix or `RELEASE READY` claim.
- No product, schema, provider, matrix or release-workflow behavior changes.
- No tag or GitHub Release before the preparation PR is merged.

### Acceptance
- `scripts/verify-release-owner-waiver.py` accepts the tracked waiver for the release commit.
- `make contracts`, `make test`, `make lint` and `make build` pass with pinned toolchains.
- Release PR is clean and mergeable, and its notes identify the unqualified status.

### Progress log
- 2026-08-03: Confirmed `v0.1.12` as the latest published tag and selected `v0.1.13`.
- 2026-08-03: Reused the existing fail-closed waiver contract; no live evidence was synthesized or
  represented as qualification evidence.
- 2026-08-03: Exact-tag waiver verification passed. Full provider-free DoD passed with contracts,
  Go suites, Python `266/266`, UI `168/168`, lint/typecheck, production build and embedded UI parity.

## EP-20260803-ui-ux-hierarchy-onboarding

Status: review — recorded owner/admin or merge acceptance remains open.

Next action: Resolve the recorded open criterion: Commit/merge the completed UI slice after owner review, then move this plan to the August archive.

### Context
The current React UI is technically stable at desktop `1440x980` and phone `390x844`, but the
main product journey is still difficult to understand. The UI uses many local font sizes below a
comfortable reading threshold, exposes too many simultaneous navigation levels, repeats or
contradicts status counts, and lets Home, Runs, Setup and Changes mirror internal implementation
surfaces instead of the operator journey.

The first-run experience is the highest adoption risk. `OnboardingShell` currently renders
Workspace, Sources, Runner and Ready as one multi-card page. It does not include the recommended
Analysis brief step, asks users to distinguish `fake`, `headless`, provider ID, provider command
and doctor readiness with limited explanation, and presents `Open console` beside
`Run first analysis` as competing outcomes. A new user can select a workspace and repositories but
still not understand what ProvenArch will read, where it will write, whether the chosen provider is
actually effective, or what happens after the button is pressed.

This plan refines the existing Epic 20 migration rather than replacing it. It follows
`docs/archive/design/UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md` and
`docs/archive/design/UI_ARCHITECTURE_CHANGE_REVIEW_MIGRATION_PLAN.md`, especially `20I1`, `20J1`, `20J2`, `20K`,
`20L` and `20N`. Existing schema, validator-gated promotion, read-only source repository boundary,
single-workspace server session, process-scoped runtime/provider selection and full-workspace Git
publication remain authoritative.

### Users and primary jobs
- First-time evaluator: start `acp serve`, complete a deterministic fake walkthrough without
  learning pipeline internals, and understand that demo evidence is not live architecture analysis.
- Architect / tech lead: connect one or more repositories, record useful analysis context, choose a
  live provider when needed, understand results and publish the architecture workspace.
- Operator / maintainer: verify effective runtime/provider readiness, diagnose a failed or partial
  run, and recover without losing last-good knowledge.
- Reviewer / stakeholder: read current knowledge and trace findings to evidence without seeing raw
  provider/runtime detail by default.

### Goals (must have)
- [x] Establish a readable semantic type scale: default body `15-16px`, secondary `14px`, metadata
      `12-13px`, with no interactive or explanatory text below `12px` at 100% zoom.
- [x] Limit normal product screens to global navigation plus at most one local navigation level.
- [x] Make Home an attention-first start surface with four independent state axes, one truthful
      next action, latest run, top review risks and publication readiness.
- [x] Split Runs into `/runs` history/launcher and `/runs/<run_id>` Run Studio detail/recovery.
- [x] Make terminal runs visibly terminal; remove stale `Current step` wording and do not render an
      empty disabled blocker control when no blocker exists.
- [x] Replace the current Changes tab stack with one coherent Architecture Change Review where
      evidence opens contextually, Findings/Proposals/Diff are distinct modes, and Publish is an
      explicit handoff rather than a hidden mobile tab.
- [x] Replace the all-at-once first-run page with a URL-restorable five-step onboarding flow:
      `Workspace -> Repositories -> Analysis brief -> Runner -> Review & start`.
- [x] Explain the source/workspace boundary, fake/live difference, desired/effective provider and
      the result of the first analysis before mutation.
- [x] Give every disabled, empty, partial and recovery state a visible reason and next action.
- [x] Keep the UI free of document-level horizontal overflow and make all primary tasks completable
      at `1440x980`, `1024px` and `390x844`.
- [ ] Commit/merge the completed UI slice after owner review, then move this plan to the August
      archive.

### Non-goals
- No hosted, multi-user, security/compliance enforcement or source-repository mutation UI.
- No new headless providers beyond `claude-code`, `qwen-code` and `codex-code`.
- No schema or provider/runtime precedence change merely to simplify UI wording.
- No visual big-bang rewrite, new charting system, speculative topology or permanent pipeline
  telemetry on Home.
- No promise that `doctor` can prove provider authentication, quota or model availability beyond
  the checks exposed by the authoritative backend response.
- No per-file publication or human approval state; publication remains full-workspace Git mutation.

### Target information architecture

Primary navigation remains:

```text
Home / Runs / Knowledge / Changes
```

- Setup is a contextual first-run or workspace-settings flow, not a permanent primary destination.
- Ask is a global `Current workspace · read-only` utility.
- Settings and Diagnostics contain runtime profile, effective/desired values, version/build and raw
  operational details.
- A screen has one primary action. Recovery may replace it, but secondary actions do not compete
  visually with it.
- Evidence is a contextual viewer shared by Changes, Knowledge and Ask; normal review does not add
  another permanent navigation bar.

### Target first-run flow

#### 0. Entry
- `acp serve` opens a short local launcher introduction: what ProvenArch produces, what stays
  read-only, and why a separate architecture workspace is required.
- Primary action: `Set up a workspace`; existing workspace rows provide direct `Open` actions.
- The launcher shows actual binary/server identity in a secondary menu, not in the main task area.

#### 1. Workspace
- Present mutually clear `Create workspace` and `Open existing workspace` modes.
- Explain that this Git-tracked folder is the only normal write target and must not be a source repo.
- Reuse recent workspaces and safe local path suggestions; show missing recent paths with `Forget`.
- Primary action: `Create workspace` or `Open workspace`, matching the selected mode exactly.
- Success summary persists above later steps: workspace name/path, Git state and write boundary.

#### 2. Repositories
- Start with one compact repository row; local checkout is the recommended fast path, Git URL is an
  alternative resolved through local Git credentials.
- Auto-fill the stable repo name from a selected local path when possible.
- Keep `ref`, include/exclude globs and imports path under `Advanced scope` unless values are already
  present or invalid.
- Validate a row on save and show diagnostics beside that row. A workspace-wide summary must not
  force the user to map an error back to a repository manually.
- Always repeat `Repositories are read-only`; no copy implies source mutation.
- Primary action: `Save and validate repositories`.

#### 3. Analysis brief
- Explain that the brief improves usefulness; it does not grant filesystem/provider permissions.
- Reuse existing charter/workspace contracts for goal/scope, known domains or teams, NFRs and rules.
- Do not invent a new persistence shape. If the launcher cannot persist the current brief contract,
  land a contract-only slice before this UI.
- Allow `Skip for now`, followed at Review by an explicit quality warning; do not silently block.
- Primary action: `Save analysis brief`; secondary action: `Skip for now`.

#### 4. Runner
- First choice is task language, not implementation jargon:
  `Deterministic walkthrough` (`fake`, recommended for first use) or `Live architecture analysis`
  (`headless`, explicit opt-in).
- Selecting walkthrough explains that outputs are synthetic demo evidence and no external AI CLI is
  called.
- Selecting live reveals provider choices with human labels and command IDs:
  Claude Code (`claude-code`, default), Qwen Code (`qwen-code`), Codex (`codex-code`).
- Show desired and effective mode/provider separately. The primary action is `Check readiness`, not
  `Select runner`, once a choice is saved.
- Readiness reports executable discovery, auth/quota guidance, write-surface/permission checks and
  exact recovery. If process restart is required, show a copyable restart command and keep
  `Pending restart` until server readback confirms the requested values.
- Step-scoped provider overrides stay under Expert settings and never obscure the global fallback.

#### 5. Review & start
- Summarize architecture workspace path, read-only repositories and refs, brief/skip state,
  deterministic demo or effective live provider, and readiness result.
- Explain the next observable sequence without internal step IDs:
  `collect evidence -> validate knowledge -> prepare architecture changes`.
- Primary action: `Start first analysis`. It creates the run once and routes directly to
  `/runs/<run_id>`; double-click cannot register a second ordinary run.
- Secondary action: `Open console without running` as a text action.
- Fake completion routes to the new review package with `Deterministic demo` and `Demo evidence`
  labels; live completion routes to the same surface with persisted provider identity.

### Onboarding state and recovery matrix

| State | Explanation | Primary recovery |
| --- | --- | --- |
| Workspace path missing/invalid | Exact field or filesystem reason | Correct path and retry |
| Existing workspace selected | Show persisted repos/runtime without overwriting drafts | Continue from first incomplete step |
| Repo path is not a usable checkout | Row-level path/Git diagnostic | Choose another folder |
| Git URL cannot resolve | Local Git/auth/ref diagnostic | Correct URL/ref or use local checkout |
| Duplicate repo name | Name is stable evidence identity | Rename the conflicting row |
| Brief skipped | Results may be less decision-ready | Add brief or acknowledge and continue |
| Fake selected | Synthetic walkthrough, no live claim | Continue with demo |
| Provider executable missing | Exact command/env override | Install/configure, restart if needed, recheck |
| Provider auth/quota check fails | Provider account state, not artifact failure | Verify provider CLI, then recheck |
| Desired/effective runtime differ | Current process still uses old values | Copy restart command and reconnect |
| Doctor unavailable/failed | Last selected values remain visible | Retry readiness; preserve form state |
| First run failed/partial | Retained evidence and last-good knowledge stay available | Open Run Studio recovery |
| Backend reconnecting | Server-derived values labelled `Last known` | Retry automatically without losing route/drafts |

### Delivery slices

#### UX-0 — Baseline and shared language
- Freeze fixture-driven screenshots/task scenarios for empty first run, fake success, live provider
  unavailable, terminal success, partial run, dirty publication and mobile navigation.
- Define product copy for `workspace`, `repositories`, `analysis brief`, `walkthrough`, `live
  provider`, `snapshot`, `current knowledge` and `publish`.
- Record the current <=`0.79rem` declaration count as a regression baseline.

#### UX-1 — Semantic typography and shell foundations (`20K`, then `20I1`)
- Add semantic type, spacing, surface, border, action, status and focus tokens to `styles.css`.
- Migrate shell, headers, forms, tabs, buttons, status rows and metadata before feature-specific
  polishing; remove local `0.68-0.78rem` overrides from touched surfaces.
- Simplify `ProductShell`: workspace, effective runtime/status and Ask remain primary; build metadata
  and context move to utilities.
- Establish comfortable and compact density variants; no nested card without an independent action
  or lifecycle.

#### UX-2 — Sequential onboarding (`20J1`)
- Refactor `OnboardingShell` from the four-card grid into a single-step session with progress,
  persisted summaries and one primary action.
- Reuse `LocalPathCombobox`, repo diagnostics, onboarding status/runtime endpoints and doctor checks.
- Add/reuse Analysis brief persistence only through an existing authoritative contract; otherwise
  precede UI work with a small contract slice and spec/tests.
- Unify launcher onboarding and contextual Guided Setup so labels, validation and recovery do not
  fork into two implementations.
- After first run, Setup opens from workspace settings at the relevant step and advanced YAML/
  diagnostics remain disclosed expert surfaces.

#### UX-3 — Attention-first Home and truthful actions (`20E`, `20J1`)
- Replace the four-metric placeholder with a compact four-axis line, ordered `Needs attention`,
  latest run and current architecture summary.
- `Start analysis` must call the start action and route to the created run. A navigation-only action
  must be named `Open Runs` or `Review setup`.
- Deduplicate global banner, page status and body copy through one typed workflow/attention selector.
- Explain relationships between artifacts, findings, questions and publication risks instead of
  presenting unrelated counts.

#### UX-4 — Runs list and Run Studio (`20J1`)
- `/runs`: compact launch control, active/pending identity, filters and history.
- `/runs/<run_id>`: ordered pipeline progress, current useful activity, artifacts and one recovery
  panel; diagnostics/logs live in a contextual drawer.
- On success show `Completed` with finish time/outcome and next action; hide stale current-step copy.
- Render blocker UI only for an actual blocker. Disabled actions have adjacent visible reasons.

#### UX-5 — Architecture Change Review (`20J2`)
- Keep one review route with a single local mode control: Overview, Findings, Proposals and Diff.
- Preserve existing `view=evidence` URL compatibility by opening contextual Evidence Studio rather
  than a permanent top tab. Preserve `view=publish` as an explicit final handoff, not a scroll-hidden
  mobile tab.
- Make each mode semantically distinct. Findings does not repeat the heading or generic Review Queue.
- Overview uses a change list with inline impact, confidence and citations; summary/publication
  readiness stays in one side rail on wide screens and follows content on mobile.
- No `Approve selected evidence` or review-blocker affordance without a persisted approval/blocker
  contract.

#### UX-6 — Setup cleanup, recovery, responsive and accessibility closure (`20L`, `20N`)
- Make tab, H1 and CTA naming consistent. `Workspace` never renders a page titled `Source`.
- Move raw YAML, build metadata and broad diagnostics to Expert/Settings disclosures.
- Atlas empty/partial states link to the missing source, analysis or evidence recovery action.
- Replace unhinted horizontal mobile tabs with a select/menu, wrapped controls or separate routes;
  Publish remains directly discoverable.
- Verify keyboard-only onboarding, run recovery, evidence review and publication; focus returns from
  dialogs/drawers and async completion is announced once.

### Files expected to change
- `ui/src/styles.css`
- `ui/src/components/ProductShell.tsx`
- `ui/src/components/SemanticPrimitives.tsx`
- `ui/src/components/OnboardingShell.tsx`
- `ui/src/components/ProductPages.tsx`
- `ui/src/components/StagePanels.tsx` through smaller extracted containers
- `ui/src/components/KnowledgePage.tsx`
- `ui/src/features/changes/ChangesWorkspace.tsx`
- `ui/src/lib/workflowState.ts`, `ui/src/lib/appRoutes.ts`, onboarding/runtime view models
- focused component tests, `ui/src/App.test.tsx`, fixture-driven Playwright mock scenarios
- `docs/archive/design/UI_ARCHITECTURE_CHANGE_REVIEW_DESIGN.md` and migration-plan acceptance where the final
  evidence/publish navigation decision is refined
- `docs/spec/API_SPEC.md` and contract fixtures only if Analysis brief launcher persistence or
  another required readback is genuinely absent

### Acceptance criteria
- [ ] A first-time user can start `acp serve`, create/open a workspace, attach one local repository,
      choose fake, pass readiness and reach the first running Run Studio without using another CLI
      command.
- [ ] Before start, the UI clearly names every read-only repository, the sole architecture write
      workspace, demo/live identity and effective provider.
- [ ] The user can return Back/Forward/reload to the current onboarding step without losing already
      persisted values; unsaved destructive navigation warns explicitly.
- [ ] Every onboarding step has one H1/purpose, one primary action, completed previous-step summaries
      and at most one dominant blocker.
- [ ] Home primary CTA either performs the named action or truthfully names navigation.
- [ ] `/runs` and `/runs/<run_id>` are distinct tasks; terminal success never shows an active current
      step or an empty blocker control.
- [ ] Changes has no duplicate Findings heading, no false mode affordance and no hidden mobile
      publication path.
- [ ] No explanatory or interactive text renders below `12px`; normal body copy is `15-16px` and
      metadata is `12-13px`.
- [ ] Normal screens have no more than two navigation levels and no document-level horizontal
      overflow at `1440`, `1280`, `1024` and `390x844`.
- [ ] Disabled actions expose a persistent adjacent reason; empty Atlas and unavailable evidence
      provide a real next action.
- [ ] Fake/demo can never be mistaken for live evidence; desired runtime/provider can never be
      mistaken for effective server readback.
- [ ] Source repositories remain read-only and Git mutations exist only in Publish.
- [ ] Required CI is deterministic and provider-free; live provider checks remain optional trusted-
      machine coverage.

### Validation
- Focused TypeScript/unit tests for workflow, route, onboarding state and terminal-run view models.
- Component tests for labels, disabled reasons, keyboard/focus behavior and responsive navigation.
- Fixture-driven Playwright at `1440x980`, `1024x768` and `390x844` for first-run fake, live provider
  unavailable, terminal success, Changes review and Atlas recovery.
- Automated accessibility checks with no critical violations and manual 200% zoom completion.
- `git diff --check`
- `make contracts`
- `make test`
- `make lint`
- `make build`

### Risks and mitigations
- A large `StagePanels.tsx` rewrite could create a hidden second shell. Deliver vertical routes and
  extract view models incrementally; remove legacy composition only after route-level acceptance.
- Launcher and in-console Setup can drift. Share step definitions, copy, validation summaries and
  view models rather than duplicating forms.
- Runtime wording can overpromise live readiness. Always distinguish desired/effective values and
  backend doctor scope; preserve typed provider recovery.
- Larger typography can expose overflow previously hidden by tiny text. Verify the target viewport
  matrix in each slice rather than postponing responsive work.
- Simplifying Changes can erase source identity. Keep run snapshot/current/published context visible
  even when navigation chrome is reduced.

### Open questions
- Confirm whether the existing charter/brief write API is safe before `enter-console`; if not, decide
  whether Review & start enters Console before persisting the brief or whether a small launcher-safe
  endpoint is required.
- Confirm whether `Start first analysis` may atomically cross `enter-console` and start the run using
  existing endpoints, or must perform two visible server-confirmed transitions.
- Decide whether local checkout or Git URL is the default repository source for public onboarding;
  this plan recommends local checkout for the local-first happy path while preserving Git URL.

### Progress log
- 2026-08-03: Audited the rendered UI findings, current launcher/onboarding implementation,
  workflow/routes, provider/runtime boundaries and existing Epic 20 migration plan. Added this
  execution plan; no product code or contracts changed.
- 2026-08-03: Started the UI-only implementation slice without application, analysis, test or build
  runs at owner request. Added semantic type floors, simplified ProductShell/Home/Changes, made
  terminal run wording truthful, removed empty blocker/approval affordances and changed launcher
  onboarding to one visible step at a time while reusing the existing brief/runtime/workspace APIs.
- 2026-08-03: Completed the second UI-only hierarchy/craft pass. `/runs` is now launcher/history
  while `/runs/<run_id>` owns timeline, retained evidence, recovery and disclosed diagnostics;
  Workspace setup is a read/write-boundary overview while repository editing remains on Sources;
  Home orders up to three real attention items; Changes uses a two-column review surface with
  route-specific evidence and collapsed coverage/question detail. Added final desktop/mobile CSS
  hierarchy overrides. Per owner instruction, validation remained static: `git diff --check` only;
  no application, tests, lint, typecheck or build were run.
- 2026-08-03: Completed rendered QA and the final consistency pass. Added truthful run-detail
  routing, contextual Changes snapshot navigation, sequential onboarding recovery reasons, Home and
  Knowledge recovery coverage, tablet screenshots and responsive repository cards. Fixed a
  cross-run race so an in-flight historical selection cannot replace the latest selected snapshot.
  Fresh Playwright evidence covers seven mock scenarios at desktop, tablet and phone sizes; all
  seven pass. The complete UI unit suite, TypeScript check and production Vite build also pass.
- 2026-08-03: Final pinned DoD passed with Node `22.21.1`: contracts, full Go suite, Python
  `266/266`, UI `158/158`, shellcheck/typecheck and the embedded production build. `git diff
  --check` remains clean.

## EP-20260803-outcome-architecture-recovery

Status: review — recorded owner/admin or merge acceptance remains open.

Next action: Resolve the recorded open criterion: Merge the audited outcome-first implementation through its feature PR after required checks and review succeed.

### Context
The August UI hierarchy/onboarding slice made the shell more readable and removed several false or
duplicated controls, but it did not solve the main product comprehension problem. ProvenArch
promises evidence-backed architecture knowledge; the UI still foregrounds pipeline mechanics,
run IDs and artifact files instead of answering what the system is, how it is structured, how
trustworthy the result is, what changed and what the operator should do next.

The operator feedback to address as one coherent product flow is:

- the UI remains visually dense and does not make the primary result obvious;
- the documentation describes Architecture Home, C4 diagrams, model entities/relationships,
  coverage, findings, questions and proposals, but the UI does not compose these into one outcome;
- C4 is buried under Changes -> selected run -> Artifact explorer -> Diagrams, while Knowledge
  Atlas is a relationship table rather than an architecture map;
- the generated C4 surfaces are static Mermaid previews without level navigation, drill-down,
  evidence inspection, filtering, zoom/fit/fullscreen or useful large-graph behavior;
- a terminal run says succeeded/failed but does not clearly summarize what was produced, what is
  partial, what changed, whether the current architecture was updated or what action comes next;
- runtime execution feels like a black box: `current_step` and inferred log telemetry do not show
  durable progress, current scope, useful activity, elapsed time or impending stall/timeout;
- errors expose technical codes/messages without consistently explaining impact, retained evidence,
  the recommended fix or the safe retry scope;
- the current Retry action starts a complete `init` or `refresh`; there is no public API for
  dependency-aware retry of a failed step or collect shard;
- onboarding, provider/repository selection, naming, navigation depth, typography, disabled reasons,
  mobile discoverability and Home usefulness must remain fixed rather than regress during the next
  redesign.

Existing contracts remain authoritative: source repositories are read-only, generated knowledge is
written only to the Git-versioned workspace, stable knowledge is validator-promoted, run snapshots
are immutable review authority, and partial or failed work must not silently replace the last-good
promoted architecture. Any retry is a new auditable run, never mutation of historical run identity.

### Users and primary jobs
- Architect / tech lead: understand the system, inspect C4 levels and important flows, find gaps and
  make an evidence-backed architecture decision.
- Maintainer: see whether analysis is making useful progress, understand failures and recover only
  the invalidated work without paying for a full rerun.
- Reviewer: compare proposed architecture knowledge with the last promoted state and trace every
  important statement, node, edge or finding to evidence.
- First-time evaluator: connect a workspace/repositories/provider, run a deterministic walkthrough
  and understand exactly what fake versus live analysis produced.

### Product outcome
After a completed or partial analysis, the UI must answer these questions without requiring the
operator to browse raw artifact paths:

1. What is this system and what scope was analyzed?
2. How is it structured at Context, Container and Component levels?
3. What services, datastores, external systems, domains, teams and relationships were found?
4. Which facts are evidence-backed, which areas are partial and which are unknown?
5. What findings and open questions require attention?
6. What changed from the previous promoted architecture?
7. What should the operator review, retry or publish next?

Raw files, logs, provider output, taskrun paths and run IDs remain available as secondary evidence
and diagnostics, not as the primary product result.

### Goals (must have)
- [x] Make the terminal Run Result outcome-first: summary, produced knowledge, coverage/gaps,
      changes, publication effect and one recommended next action.
- [x] Introduce a first-class Architecture Explorer with C4 level navigation, map interaction,
      catalog/flow/evidence inspection and truthful partial states.
- [x] Promote current architecture, not a historical run artifact browser, as the default post-run
      destination.
- [x] Add structured runtime progress that distinguishes provider activity from durable artifact
      progress and exposes step/unit counts, active scopes, elapsed time and stall pressure.
- [x] Replace raw error presentation with a typed recovery model: plain-language cause, affected
      result, retained evidence, recommended fix, retry capability and disclosed technical detail.
- [x] Add dependency-aware step/shard retry that creates a child run, reuses only validated parent
      inputs and automatically reruns all invalidated downstream steps.
- [x] Make Changes a comparison/publication workflow over the current and prior architecture rather
      than the main place to discover diagrams.
- [x] Preserve the completed onboarding/readability/navigation/mobile/recovery improvements from
      `EP-20260803-ui-ux-hierarchy-onboarding`.
- [x] Keep all new progress, retry and result behavior deterministic, Git-friendly and auditable.
- [x] Post-audit: make `/api/architecture` the primary UI read path with `/api/knowledge` only as a
      compatibility fallback, and expose dependency-planned step rerun after succeeded as well as
      failed/canceled terminal parent runs. Rendered QA also made partial legacy run-step arrays
      fail-soft and closes the Advanced C4 disclosure after navigation so it cannot cover the map
      breadcrumb.
- [ ] Merge the audited outcome-first implementation through its feature PR after required checks
      and review succeed.

### Non-goals
- No hosted or multi-user control plane, source-repository writes, security/compliance enforcement
  or additional providers.
- No speculative topology: unavailable C4 levels, ownership, relationships or evidence remain
  explicit gaps.
- No arbitrary isolated downstream step execution that can leave findings/proposals inconsistent
  with changed upstream evidence.
- No mutation or continuation of a terminal historical run; retry always creates a new run ID.
- No fake numeric percentage or ETA for opaque provider work. ETA may be shown only from sufficient
  comparable historical evidence; otherwise show elapsed time, unit counts and last useful progress.
- No big-bang replacement of the artifact model or validator-gated promotion contract.

### Target information architecture

```text
Home / Architecture / Changes / Runs
                         Setup remains contextual
```

- **Home:** current architecture freshness/trust, concise system summary, map preview, top
  findings/questions, last run outcome, publication readiness and one next action.
- **Architecture:** `Map`, `Overview`, `Catalog`, `Flows`, `Evidence`. This replaces the generic
  Knowledge framing while retaining current promoted read-only authority.
- **Changes:** comparison to the previous promoted state, findings/proposals/diff and explicit
  full-workspace publication.
- **Runs:** start/history and per-run operation, progress, result, failure recovery and diagnostics.
- **Setup:** workspace, repositories, analysis brief, runner/provider and advanced configuration.

Normal screens keep global navigation plus one local level. Evidence opens contextually in an
inspector/drawer and does not introduce another persistent tab hierarchy.

### Target user flow

```text
Configure workspace and read-only sources
  -> choose fake/live provider and verify readiness
  -> run analysis with visible durable progress
  -> read a plain-language Run Result
  -> explore current Architecture Map and overview
  -> review gaps/findings and retry only invalidated work when necessary
  -> inspect architecture changes
  -> publish the Git-versioned workspace
```

### Run Result UX contract
Every terminal run presents one of these operator states:

- `Completed`: result is usable and the promoted architecture is current.
- `Completed with gaps`: valid result is usable, but named scopes/evidence are partial.
- `Failed`: the attempted result was not promoted; last-good architecture remains active.
- `Canceled`: stopped by request; retained evidence and last-good architecture remain available.

The result header must show:

- a plain-language one-sentence outcome;
- whether promoted current knowledge changed;
- produced entities/edges/diagrams/findings/questions/proposals counts;
- analyzed/partial/failed scope counts;
- comparison with the previous successful/promoted baseline;
- one recommended action such as `Explore architecture`, `Review findings`, `Retry failed scope`
  or `Review changes for publication`.

The backend should expose a structured result summary instead of making the UI infer semantic
outcomes from artifact counts and arbitrary log text. Historical/legacy runs may use an explicitly
labelled bounded derived summary.

### Runtime progress UX contract
Progress is presented at three levels:

1. **Pipeline:** current step, completed/total steps, elapsed time and the result expected from the
   active step.
2. **Step units:** planned/running/succeeded/failed/pending shards or scopes, active repository/domain
   identities and validation/repair phase.
3. **Runtime health:** last provider activity, last useful artifact progress, artifact observed/valid
   state, repair attempt, stall threshold/deadline and cancel/recovery action.

The UI must distinguish:

- `provider_working`: provider output/activity exists;
- `artifact_observed`: durable output exists but is not validated;
- `validating`: deterministic contract checks are running;
- `repairing`: bounded repair attempt N/M is active;
- `stalled`: useful artifact progress is outside the expected activity window;
- `completed` / `failed` / `canceled`.

Heartbeat or stdout alone is activity, not progress. No single continuous percentage is shown for
opaque provider work. Determinate bars are limited to real units such as steps and planned shards.

Candidate structured snapshot (exact schema to be finalized in the contract slice):

```json
{
  "step_id": "init.step1.collect",
  "phase": "provider_working",
  "completed_units": 7,
  "total_units": 12,
  "active_units": 3,
  "failed_units": 0,
  "current_scopes": ["payments/backend", "identity/api"],
  "elapsed_ms": 522000,
  "last_activity_at": "2026-08-03T10:21:27Z",
  "last_progress_at": "2026-08-03T10:21:14Z",
  "artifact_state": "partial",
  "repair_attempt": 0,
  "repair_limit": 1,
  "stall_deadline_at": "2026-08-03T10:26:14Z"
}
```

### Error and recovery UX contract
Every actionable error exposes structured fields:

- category: `setup`, `provider`, `permission`, `evidence`, `contract`, `timeout`, `infrastructure`
  or `canceled`;
- plain-language title and explanation;
- failed step and optional failed scopes/shards;
- affected user-facing results;
- retained/reusable evidence;
- recommended fix and navigation target;
- `can_retry`, recommended retry mode and calculated downstream invalidation;
- technical `error_code`, trace/log/artifact refs under an expandable disclosure.

Disabled retry/recovery controls always explain why they are unavailable. Provider setup, quota,
permission and workspace validation failures route to the exact configuration/recovery surface.

### Retry dependency policy
Retry creates a child run with `parent_run_id`, retry reason, requested scope, actual start point,
reused validated inputs and invalidated downstream steps. The backend, not the browser, calculates
the safe execution closure.

| Failure/request | Reuse | Execute |
| --- | --- | --- |
| Charter | none from current attempt | Step 0 through Step 4 |
| One or more Collect shards | validated sibling shards | failed shards, aggregate Collect, Steps 2-4 |
| Complete Collect | validated upstream Charter | Step 1 through Step 4 |
| As-is docs/model/diagrams | validated Charter and Collect | Steps 2-4 |
| Findings | validated Steps 0-2 | Steps 3-4 |
| Proposals | validated Steps 0-3 | Step 4 |
| Deterministic validation-only transient | immutable staged inputs when still valid | validation, then downstream only if bytes change |
| Provider failure before artifact | validated upstream and siblings | blocked unit/step plus downstream closure |

The first contract should support `failed_step` and `failed_scopes`; arbitrary operator-selected
step ranges remain a later extension. A successful child run publishes atomically through the
existing promotion boundary. Failed retry never replaces last-good promoted knowledge.

Candidate endpoint:

```http
POST /api/pipeline/runs/{run_id}/retry
```

```json
{
  "mode": "failed_scopes",
  "step_id": "init.step1.collect",
  "scope_ids": ["payments/backend"]
}
```

### Architecture Explorer UX contract
- Default surface is the current promoted architecture, never an implicit historical snapshot.
- `Map` offers `Context / Containers / Components / Code` level navigation with breadcrumbs.
- Selecting a node or edge opens name/type/technology/owner/confidence/evidence and related findings.
- Filters cover repository, domain, owner and entity type; controls include fit, zoom, fullscreen and
  show/hide gaps/external systems/ownership.
- Clicking a C4 node drills down when a validated lower-level view exists; otherwise it explains the
  missing evidence required to create that level.
- Large graphs use stable layout, bounded labels and progressive disclosure rather than a fixed
  minimum-width SVG with unbounded horizontal scrolling.
- `Overview`, `Catalog`, `Flows` and `Evidence` provide readable non-graph alternatives and share
  selection/deep-link identity with the map.
- Every view identifies current/snapshot authority, fake/live provenance, freshness and partial
  state. The UI must not label an evidence-poor generic flowchart as complete C4.

### Delivery slices

#### Slice 1 - Outcome-first Run Result on existing contracts
- Add a pure view model that derives a conservative summary from current run review, snapshot,
  knowledge and Git diff responses.
- Replace the terminal status-first hierarchy with outcome, produced surfaces, gaps and next action.
- Preserve raw status and diagnostics under disclosure.
- No backend/schema change in this slice.

#### Slice 2 - Structured error/recovery presentation
- Centralize error taxonomy and user-facing recovery actions.
- Replace ad hoc `error_code` substring branches across Setup/Runs/Publish with typed presentation.
- Keep unknown codes visible and safely non-retryable by default.

#### Slice 3 - Structured runtime progress contract
- Define schema/types and source-of-truth lifecycle aggregation in the orchestrator/API.
- Persist enough progress state for page reload/restart without treating heartbeat as progress.
- Render segmented step progress, shard/unit state, active scopes, elapsed/useful-progress timing and
  stall warnings. Start with bounded polling; add SSE only if measured polling latency/load requires
  it.

#### Slice 4 - Dependency-aware retry contract
- Add retry planning/validation and child-run lineage.
- Implement failed-step closure first, then failed Collect shard/scope reuse.
- Validate source revisions, parent artifacts, task identity and baseline integrity before reuse;
  fall back to a clearly explained wider retry rather than unsafe reuse.
- Add UI confirmation showing reused work, rerun work, downstream invalidation and expected cost.

#### Slice 5 - Architecture Explorer foundation
- Rename/reframe Knowledge to Architecture while keeping backward-compatible route handling.
- Add Architecture Home and promoted C4 entrypoints directly to the default surface.
- Implement the read-only interactive React Flow + ELK map immediately, including the level
  switcher, node/edge inspector, filters, fit/zoom/fullscreen and partial-state explanation.
- Keep deterministic Mermaid files as exports only; they are not the interaction source of truth.

#### Slice 6 - Catalog, flows and evidence-linked drill-down
- Add entity/relationship catalog filters, key-flow view and shared node/edge/artifact deep links.
- Connect map selection to exact model YAML, repository evidence, findings and questions.
- Add mobile list/inspector fallback instead of forcing a desktop graph canvas onto small screens.

#### Slice 7 - Architecture comparison and publication handoff
- Compare current/promoted architecture with the prior accepted baseline at entity, edge, diagram,
  finding and question level.
- Make Changes answer what changed and why before showing raw Git diff.
- Carry accepted selection to the existing full-workspace Git publication confirmation.

#### Slice 8 - Home consolidation and legacy removal
- Make Home summarize current architecture outcome, freshness, trust and next action.
- Remove duplicated legacy Review/Atlas/artifact-navigation paths only after deep links and E2E
  coverage prove the replacement surfaces.
- Complete typography, responsive, focus, empty/error/partial and 200% zoom polish across the final
  information architecture.

### Files/modules expected to change
- Contracts/specs: `schemas/*` as required, `docs/spec/PIPELINE_SPEC.md`,
  `docs/spec/MODEL_SPEC.md`, `docs/APPENDIX_SCHEMAS.md`, API/UX documentation and fixtures.
- Backend lifecycle/API: `internal/orchestrator/orchestrator.go`, `service_runs.go`, step/shard
  execution/progress modules, `internal/api/server.go`, `internal/api/review_diff.go` and tests.
- UI contracts/state: `ui/src/lib/appContracts.ts`, `runApi.ts`, `runState.ts`, `appRoutes.ts`,
  workflow/recovery/progress view models and hooks.
- UI surfaces: `ProductPages.tsx`, `KnowledgePage.tsx` or extracted Architecture pages,
  `StagePanels.tsx` or extracted Run Result/Progress components, `MermaidPreview.tsx`,
  `ChangesPage.tsx`, `ProductShell.tsx` and `styles.css`.
- Test assets: Go API/orchestrator tests, schema fixtures/goldens, UI unit/component tests and
  Playwright fake/snapshot scenarios.

### State coverage
Every new surface must cover loading, empty, partial, available, stale, running, stalled, failed,
canceled, permission-blocked, provider-unavailable, retry-planning, retry-running, retry-failed and
retry-succeeded states. Current promoted knowledge remains accessible when a new or retried run is
active or failed.

### Validation plan
- Contract/schema validation for progress, result summary, retry request/plan and run lineage.
- Orchestrator tests proving safe dependency closure, validated sibling reuse, source-revision drift
  fallback, immutable parent history and atomic last-good preservation.
- API tests for retry admission/conflict/idempotency, progress persistence, unknown errors and
  legacy run compatibility.
- UI view-model/component tests for every terminal outcome, step/unit progress phase, stall warning,
  retry confirmation and disabled reason.
- Fixture/golden updates for complete, partial, failed-shard, failed-step, retry-success and
  retry-failure scenarios.
- Playwright flows at `1440x980`, `1024x768` and `390x844`: first run, useful live progress fixture,
  provider failure, failed-scope retry, terminal result, C4 drill-down, partial architecture and
  publication handoff.
- Keyboard/focus, reduced-motion, contrast, screen-reader labels and 200% zoom checks.
- Per completed slice: `make contracts`, `make test`, `make lint`, `make build`; live provider gates
  only when the changed contract/behavior requires the project release runbook.

### Acceptance criteria
- [x] A first-time user can explain what the last run produced without opening raw artifacts/logs.
- [x] A completed run links directly to current Architecture Map and Architecture Home.
- [x] A partial/failed run states whether last-good architecture changed and exactly what remains
      usable.
- [x] During Collect, planned/completed/active/failed scopes and last useful progress are visible.
- [x] Provider output without artifact progress is never presented as completed percentage.
- [x] Every supported error offers a specific recovery action; unknown errors remain inspectable and
      do not expose an unsafe retry.
- [x] Retrying a Step 3 failure creates a child run that reuses validated Steps 0-2 and executes
      Steps 3-4; retrying failed Collect scopes reuses only validated siblings and executes all
      invalidated downstream work.
- [x] Parent run history and taskrun evidence remain immutable and navigable.
- [x] Architecture supports Context/Container/Component navigation with evidence inspector and
      truthful unavailable/partial states.
- [x] Mobile users can inspect the same architecture facts through a list/inspector fallback with no
      hidden horizontally-scrolled primary actions.
- [x] Existing onboarding, source boundary, provider selection, publication authority and Git safety
      behavior do not regress.

### Risks and mitigations
- **Unsafe retry reuse:** stale or foreign artifacts could contaminate a child run. Require exact
  parent/run/task/source identity, baseline integrity and full validation; widen retry scope on any
  uncertainty.
- **Progress fiction:** provider activity can look like advancement. Keep activity/progress clocks
  separate and derive determinate progress only from known units and validated state transitions.
- **Schema blast radius:** progress/lineage/result contracts touch docs, validators and fixtures.
  Land each contract separately and use `acp-schema-guardian` during implementation.
- **Graph usability/performance:** validate the React Flow + ELK renderer with bounded large-graph
  fixtures, deterministic read-only layout and a mobile catalog/inspector fallback.
- **Navigation migration:** renaming Knowledge can break bookmarked URLs. Preserve redirects and
  route parsing compatibility until legacy links have migration coverage.
- **StagePanels growth:** do not add more modes to the monolith. Extract RunResult, RuntimeProgress,
  RecoveryPanel and Architecture surfaces with pure view models.

### Open questions
- Resolved: successful retry uses the same automatic validator/atomic promotion gate; Git publish
  remains explicit.
- Resolved: Context/Container/Component are primary; Code is Advanced.
- Resolved: previous promoted architecture is semantic baseline and Git HEAD is publication diff.
- Resolved: retry is terminal-parent only and the interactive map is read-only with deterministic
  layout; no model/layout persistence is introduced.
- Resolved: delivery is one feature branch/PR with reviewable internal commits.

### Progress log
- 2026-08-03: Consolidated operator feedback, README/architecture promises and current UI/API/runtime
  behavior into this outcome-first Architecture/Run/recovery plan. Confirmed that public UI retry
  currently starts a full `init`/`refresh`, existing restart resume is not arbitrary user step retry,
  and live diagnostics infer progress from logs rather than a structured progress contract. No
  product code, schema or runtime behavior changed.
- 2026-08-03: Implemented the vertical product slice: promoted Architecture API and interactive
  explorer, outcome-first Home/Run Result, persisted structured progress, typed recovery, retry
  planning/plan-hash admission, immutable child lineage and staging-only reuse. Updated docs,
  contracts and deterministic tests.
- 2026-08-03: Completed rendered desktop/mobile QA, the seven-scenario mock Playwright suite,
  bounded 80-node ELK layout coverage and mandatory `make contracts`, `make test`, `make lint`,
  `make build`. No live-provider E2E was run because release-gate behavior is unchanged.
- 2026-08-03: Post-implementation gap audit closed unsafe retry reuse and staging drift, persisted
  elapsed time, requested/effective retry lineage, historical run-scoped result counts/coverage,
  conservative unknown-error recovery, semantic finding/question linkage, explicit lower-level
  reasons, service-scoped Code and keyboard/200% zoom coverage. Repeated contracts, full Go/Python/UI
  tests (`266` Python, `165` UI), lint/typecheck, embedded build and all seven mock Playwright
  scenarios. Live-provider E2E remains intentionally out of scope for this UI/API contract PR.
- 2026-08-03: A second plan-by-plan audit found four remaining correctness gaps and closed them:
  promoted Architecture semantics now live in immutable snapshot manifest v2 (including no-op
  baseline lineage), Changes compares individual findings and normalized gaps, Run Result/Progress
  exposes scope/baseline/unit/repair/stall detail with segmented real-step progress, and retry now
  revalidates, rebinds and hydrates shard/aggregate/PASS-verdict inputs into the child execution
  state. Added a real fake parent-to-Proposals-child promotion regression, public-shape
  example/fixture, and synchronized API/model/pipeline/docs contracts. Full DoD and seven mock
  Playwright scenarios were repeated; live-provider E2E remains outside this PR scope.
- 2026-08-03: A third plan-by-plan audit found that retry confirmation received
  `invalidated_steps` but did not present that closure separately, and mislabeled mixed
  step/shard `estimated_units` as pipeline steps. The confirmation now explains why downstream
  work is rebuilt and distinguishes validated reuse, child execution, invalidation, effective
  scope and execution-unit estimate; component coverage protects the operator-facing contract.
- 2026-08-03: A fourth acceptance-level audit closed two coverage gaps hidden by earlier completed
  checkboxes. Architecture now filters both canvas and mobile fallback by canonical owner and exact
  model tag/domain values without filename inference. Retry staleness is verified through the real
  HTTP admission path (`409 retry_plan_stale` after parent staging drift), and persisted progress is
  verified after run-history reload with separate activity/useful-progress timestamps intact.

## EP-20260729-open-source-readme

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Complete fresh composite trusted-machine evidence before claiming canonical `RELEASE READY`; exact-tag owner waivers may publish only an explicit `UNQUALIFIED PRERELEASE`.

### Context
The repository entrypoint is a mixed Russian/English implementation digest that makes a new
visitor reconstruct the product purpose, expected outputs, first-run path and trust boundary from
release-engineering details. The repository owner requested an English-first open-source README.
Canonical implementation and contract boundaries remain in `docs/ARCHITECTURE.md`,
`docs/STAKEHOLDER_DOC.md` and `docs/spec/*`.

### Goals (must have)
- [x] Make `README.md` an English product entrypoint explaining what ProvenArch is, who it is for,
      why it exists and what files it produces.
- [x] Provide a verified deterministic fake-first walkthrough before live provider configuration.
- [x] State beta maturity, runtime trust boundaries and MVP non-goals without overstating provider
      or release readiness.
- [x] Route detailed contracts to canonical documentation instead of duplicating them.
- [x] Align repository documentation policy and docs-sync tests with the English README exception.
- [ ] Complete fresh composite trusted-machine evidence before claiming canonical `RELEASE READY`;
      exact-tag owner waivers may publish only an explicit `UNQUALIFIED PRERELEASE`.

### Non-goals
- Translating the full primarily Russian documentation set.
- Changing runtime behavior, schemas, public APIs or release evidence.
- Claiming canonical `RELEASE READY` status.

### Validation
- `git diff --check`
- README local-link and language-policy tests
- `make contracts`
- `make test`
- `make lint`
- `make build`
- `make offline-closure`

### Risks
- A shorter entrypoint can erase important caveats. Keep explicit beta, source/workspace and live
  provider trust boundaries, and link exact behavior to canonical specifications.
- README may describe `main` behavior newer than the latest published binary. Keep the explicit
  release-notes boundary and prepare matching release metadata after merge.

### Progress log
- 2026-07-29: Drafted the English product entrypoint and synchronized documentation guardrails.
- 2026-08-01: Published draft PR #191 after focused documentation and deterministic checks passed.
- 2026-08-02: Rebased PR #191 onto qualification SHA `5cf7ba976191b1b732ad9b49fb1b1b761d997926`;
  preserved the current R3 plans while retaining the product-led README replacement.
- 2026-08-02: Exact-toolchain `make offline-closure` passed with race suites, 90 readable fixtures,
  263 Python tests, 158 UI tests, 7 rendered mock scenarios, contracts, lint, build and embedded UI
  parity. README local links and the focused docs-sync suite also passed; the PR is ready for CI.
- 2026-08-02: Backend CI correctly rejected the active plan after every goal was marked complete
  before merge. Restored the pending merge goal; this is tracker-state correction only.
- 2026-08-02: PR #191 passed all 11 checks and was rebase-merged into protected linear-history
  `main` as `68217aaba9dbd1c81814e8f5c7d23608bea5b2e3`. Started post-merge offline qualification and
  truthful `v0.1.10` candidate metadata; release tag creation remains blocked on fresh composite
  trusted-machine evidence.
- 2026-08-02: Fresh detached post-merge `make offline-closure` passed on `68217aaba9dbd1c81814e8f5c7d23608bea5b2e3`
  with the exact Go/Node/npm toolchains, including race suites, 90 readable fixtures, 263 Python
  tests, 158 UI tests, 7/7 rendered mock E2E, contracts, lint, build and embedded UI parity. The
  qualification worktree remained clean; the canonical `/tmp/provenarch-live-e2e` source checkout
  root is not currently present, so no source-repository state was mutated or available to audit.
- 2026-08-02: Owner explicitly authorized publishing `v0.1.10` without Qwen/Claude live runs.
  Trusted-host preflight found all 12 pinned path repositories and exact toolchains ready, but Qwen
  and the configured Claude-through-Kimi route both returned exhausted billing-cycle quota; native
  Claude was unauthenticated and Codex artifact smoke passed. The release is therefore authorized
  only as a tracked `UNQUALIFIED PRERELEASE`, never as canonical `RELEASE READY`.
- 2026-08-02: The `v0.1.10` tag passed waiver verification but its release job stopped before
  publication because `make test` lacked the pinned PyYAML dependency available in backend CI.
  The immutable failed tag is retained; recovery moves to `v0.1.11` with the dependency fix,
  regression coverage and a new exact-tag owner-waiver record.

---

## EP-20260801-r3-collect-root-claims-repair

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated fix, establish a new qualification SHA and restart R3 from fresh smoke.

### Context
PR #197 merged as qualification SHA `ed68d3bfdac48a83b5956abab4a396699a9e7c54`; detached
post-merge offline closure passed. Fresh diagnostic smoke `smoke-tiny-bank-20260801T213628Z`
then failed one of ten init collect shards. The bounded Qwen manifest was otherwise populated but
added forbidden top-level `claims`. Strict schema validation correctly rejected it. The generic
manifest-only repair prompt then produced about 2 MiB of stream output without a valid replacement
before `runtime_stalled_after_artifacts`, so the matrix ended `FAIL` with
`runtime_contract_failed=1`, `repair_exhausted=1` and `stall_pressure=1`. Regression and release
phases were not started; the matrix is diagnostic only.

### Goals (must have)
- [x] Make the bounded normal Qwen collect contract explicitly close the root key allowlist and
      name top-level `claims`/`claim_map` as forbidden before the atomic pair write.
- [x] Route only the exact root-level additional-property family for known forbidden wrapper keys
      to a compact Qwen read-once/write-once repair contract that removes that one root field and
      preserves every canonical nested value.
- [x] Keep unknown and nested additional-property failures on the full provider-authored,
      fail-closed manifest repair path.
- [x] Add prompt/adapter regressions, synchronize docs, and pass full deterministic DoD/offline
      closure before a fix PR.
- [ ] Merge the isolated fix, establish a new qualification SHA and restart R3 from fresh smoke.

### Non-goals
- [x] Do not change schemas, public APIs, server-side validation, provider models/commands, repair
      attempt counts, timeout budgets, matrices, repositories or release acceptance.
- [x] Do not create a generic extra-field sanitizer or alter canonical semantic content.
- [x] Do not accept the failed smoke or its partial artifacts as R3 evidence.

### Approach
1. Strengthen the normal bounded prompt with the exact root allowlist and a named `claims` check.
2. Recognize only schema diagnostics whose instance path is the root object, schema target is the
   root `additionalProperties` rule, and field is one of the already-forbidden wrapper keys.
3. For Qwen only, replace the large evidence/skeleton repair prompt with one `read_file` followed
   by one same-target `write_file`, allowing removal of exactly the diagnosed root field.
4. Prove that nested/unknown extras retain the existing full repair prompt and backend validation
   remains the only success surface.

### Files expected to change
- `internal/runtime/promptcontract/collect_repair.go`
- prompt-contract and Qwen adapter tests
- runtime architecture/spec/testing/runbook documentation and this ExecPlan

### Acceptance criteria
- [x] Normal bounded Qwen prompt remains within 6 KiB and forbids root `claims` explicitly.
- [x] Exact root `claims` diagnostics produce a compact prompt below 2.4 KiB with one read and one
      write; nested `citation_ids` diagnostics do not use it.
- [x] Focused tests pass 20 repetitions; `make contracts`, `make test`, `make lint`, `make build`
      and separate `make offline-closure` pass on pinned toolchains.
- [x] Embedded UI and all 12 source repositories remain clean.

### Risks
- Error-text routing must not mistake nested schema failures for root drift. The matcher therefore
  requires both the empty instance path and the root schema location, plus a closed known field
  set; every resulting artifact still receives full task-aware backend validation.

### Progress log
- 2026-08-01: Recorded the failed smoke in a bounded external operator report and stopped R3. The
  failure is product prompt-contract behavior, not host/auth/quota/timeout/infrastructure failure.
- 2026-08-01: Closed the normal bounded Qwen root allowlist and added the exact compact root-field
  removal prompt. Matching requires Qwen, the empty root instance path, the root schema
  `additionalProperties` location and one known forbidden wrapper; nested, unknown and non-Qwen
  cases are pinned to the existing full repair path.
- 2026-08-01: Focused normal/compact/negative prompt and adapter tests passed 20 repetitions. Exact
  Go 1.25.10 / Node 22.21.1 / npm 10.9.4 `make contracts`, `make test`, `make lint`, `make build`
  and separate `make offline-closure` completed with exit code 0, including race suites, 263 Python
  tests, 158 UI tests, seven mock E2E scenarios, 90 readable fixtures and embedded UI drift checks.
  All 12 pinned source repositories remain clean.

## EP-20260801-r3-live-prompt-canonical-shapes

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated fix, establish a new qualification SHA and restart R3 from a fresh smoke.

### Context
Qualification SHA `e458937756274c12131bc81d8ef407f52234c66a` passed detached post-merge
offline closure. Fresh diagnostic smoke `smoke-tiny-bank-20260801T194033Z` then returned machine
`PASS` with strict 1/1 success, but required three focused repairs: a collect citation omitted
mandatory `document_ids`, Architecture Home compressed concrete repository paths into wildcard
references, and a validator question added forbidden `citation_ids`. The deterministic validators
rejected each first-pass artifact correctly. `runtime_quality.repair_heavy=1` makes the smoke
unacceptable as R3 qualification, so regression and release phases were not started.

### Goals (must have)
- [x] Make the bounded Qwen collect prompt show mandatory non-empty citation `document_ids` as an
      explicit object shape and require a pre-write five-field check.
- [x] Require normal step2 prompts to reject wildcard-bearing `repo:path` tokens before the draft
      manifest write instead of relying on focused repair.
- [x] Close the validator-question prompt shape to `id`, `text`, optional `priority` and
      `related_ids`, explicitly forbidding `citation_ids`.
- [x] Add provider-free prompt regressions, synchronize runtime documentation and pass the full
      deterministic DoD/offline closure.
- [ ] Merge the isolated fix, establish a new qualification SHA and restart R3 from a fresh smoke.

### Non-goals
- [x] Do not change schemas, public APIs, validators, recovery routing, provider models/commands,
      retry or timeout budgets, matrices, curated repositories or release acceptance thresholds.
- [x] Do not accept the diagnostic smoke or its repaired artifacts as R3 evidence.
- [x] Do not synthesize or normalize provider-authored semantic content.

### Approach
1. Replace compact prose around Qwen citations with one closed five-field object guide and a
   mandatory pre-write object check while retaining the 6 KiB prompt budget.
2. Add an Architecture Home pre-manifest scan rule for glob metacharacters without weakening the
   existing server-side containment/reference validator.
3. Reuse the schema-derived validator question allowlist in both normal and focused repair prompts.
4. Pin all three live-observed shapes in shared policy, prompt-contract and adapter tests.

### Files expected to change
- `internal/artifactquality/policy.go`
- `internal/runtime/promptcontract/collect_repair.go`
- `internal/runtime/steppolicy/policy.go`
- corresponding provider-free tests
- runtime architecture/spec/testing/runbook documentation and this ExecPlan

### Acceptance criteria
- [x] Bounded Qwen collect prompt remains at most 6 KiB and names all mandatory citation fields.
- [x] Normal and focused validator prompts forbid `questions[].citation_ids` without schema change.
- [x] Normal step2 prompt requires concrete wildcard-free Architecture Home references before the
      manifest write; strict backend validation remains authoritative.
- [x] Focused tests pass 20 repetitions; `make contracts`, `make test`, `make lint`, `make build`
      and `make offline-closure` pass with exact pinned toolchains.

### Risks
- The Qwen contract has a strict prompt-size budget. New wording replaces compact prose and tests
  enforce the cap; no retry budget or provider-specific runtime behavior changes.

### Progress log
- 2026-08-01: Classified the fresh smoke as provider prompt-contract drift after strict validation
  correctly rejected all three first-pass payloads. Wrote a bounded external operator report and
  stopped R3 before regression/release matrices.
- 2026-08-01: Replaced the bounded Qwen citation prose with an explicit mandatory five-field shape
  and pre-write check, added the Architecture Home wildcard-token pre-manifest scan, and closed
  validator questions to the schema allowlist. Shared policy, prompt-contract and qwen adapter
  tests cover the three live-observed payloads without changing schemas or runtime validation.
- 2026-08-01: Focused tests passed 20 repetitions. Exact Go 1.25.10 / Node 22.21.1 / npm 10.9.4
  `make contracts`, `make test`, `make lint`, `make build` and separate `make offline-closure`
  completed with exit code 0, including race suites, 263 Python tests, 158 UI tests, seven mock E2E
  scenarios, 90 readable fixtures and embedded UI drift checks. The first DoD attempt reached the
  UI test stage before reporting missing local `ui/node_modules`; after lockfile-pinned `npm ci`,
  the complete gate was repeated successfully. All 12 pinned source repositories remain clean.

## EP-20260801-r3-proposal-nested-section-validation

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Pass full provider-free DoD/offline closure, merge, establish a new qualification SHA and restart R3 from fresh smoke.

### Context
PR #194 merged as qualification SHA `0214fc6d369bb48d409277980ac15e001f0ecb30`, and a
detached post-merge offline closure passed. Fresh diagnostic smoke
`smoke-tiny-bank-20260801T171936Z` then completed 10/10 init collect shards, Architecture Home and
validator findings before failing `init.step4.proposals`. The final `proposal.md` contains the exact
required H2 `Proposed changes or follow-up plan` and three substantive H3 Sprint subsections, but
the shared Markdown section parser ends every section at the next heading regardless of level. It
therefore treats the H2 body as empty and exhausts draft enrichment. Regression and release phases
were not started; the smoke is diagnostic only.

### Goals (must have)
- [x] Preserve nested H3-H6 subsections inside a matched H1-H5 runtime draft section.
- [x] Stop section extraction at the next heading of the same or a higher hierarchy level.
- [x] Keep empty nested headings non-substantive and prevent content from a sibling section from
      satisfying the required section.
- [x] Add provider-free live-shaped positive and negative regressions and synchronize docs.
- [ ] Pass full provider-free DoD/offline closure, merge, establish a new qualification SHA and
      restart R3 from fresh smoke.

### Non-goals
- [x] Do not change schemas, public API, required proposal sections, actionability thresholds,
      provider prompts/models, timeouts, matrices, repositories or release acceptance.
- [x] Do not accept the failed smoke as release evidence.
- [x] Do not relax final draft validation or synthesize proposal content.

### Approach
1. Parse the ATX heading level selected for a required Markdown section.
2. Include later headings only when they are deeper than the selected section; stop at the first
   same/higher-level heading.
3. Exclude heading-only lines from substantive/actionable body checks.
4. Pin the observed H2 + H3 Sprint shape, an empty nested subsection and a sibling-section leak.

### Files expected to change
- `internal/runtimedrafts/manifest.go`
- `internal/runtimedrafts/manifest_test.go`
- runtime architecture/spec/testing documentation and this ExecPlan

### Acceptance criteria
- [x] The live-observed proposal passes the required-section and actionability checks.
- [x] An H2 followed only by an H3 title remains invalid.
- [x] Actionable text in the next H2 cannot satisfy an empty required H2.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.

### Risks
- Hierarchy-aware extraction affects every runtime draft section consumer. Focused tests therefore
  cover both nested acceptance and sibling isolation, while all existing draft fixtures remain the
  compatibility baseline.

### Progress log
- 2026-08-01: Classified fresh smoke failure as product validation defect. The authored proposal
  was substantive and correctly structured; only nested-heading extraction made it appear empty.
- 2026-08-01: Added hierarchy-aware section extraction plus heading-only exclusion. The
  live-shaped positive, empty nested-heading negative and sibling-leak negative tests passed 20
  repetitions; the complete `internal/runtimedrafts` package passed.
- 2026-08-01: Exact Go 1.25.10 / Node 22.21.1 / npm 10.9.4 `make contracts test lint build` and
  a separate `make offline-closure` completed with exit code 0, including race suites, 263 Python
  tests, 158 UI tests, 7 rendered mock scenarios, 90 readable fixture artifacts, embedded UI drift
  and live/product boundary checks. The worktree contains only the expected parser/tests/docs diff
  and all 12 pinned source repositories remain clean.
- 2026-08-01: Two backend CI attempts exposed an unrelated shared API test-server shutdown race.
  The parser PR remained open while the isolated test-only fix passed its own offline closure and
  merged as PR #196 (`0d1e146435d61595becd82fd7f8b173467a508ea`); this branch now includes that
  merge and will repeat the complete provider-free gate before CI restarts.
- 2026-08-01: The post-integration exact-toolchain `make contracts`, `make test`, `make lint`,
  `make build`, and separate `make offline-closure` all completed with exit code 0. The combined
  tree passed API/provider race suites, 263 Python tests, 158 UI tests, all 7 mock E2E scenarios,
  readable-fixture and embedded-UI drift checks; the 12 pinned source repositories remain clean.
## EP-20260801-r3-collect-task-identity-recovery

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Confirm the task-identity recovery boundary in fresh R3 release constituents before archive.

### Context
Qualification SHA `bd52e7aee0793b8cd7471b8cb9507738b4a42129` passed post-merge
offline closure. Fresh diagnostic smoke `smoke-tiny-bank-20260801T192845Z` then completed all init
collect shards and draft steps without the previous stale-artifact repair failure. During refresh,
qwen authored an otherwise valid source-shard manifest whose `shard_id` ended in `dc3a1b6aa9`
instead of the assigned `dc3a1b6aa0a9`. Strict validation correctly stopped the shard; the generic
provider manifest repair did not rewrite it. The smoke is diagnostic only and R3 remains stopped.

### Goals (must have)
- [x] Recognize the existing strict task-identity validation failure as a typed recovery issue.
- [x] Correct only assigned top-level `run_id`, `step_id`, `shard_id`, `domain_id` and
      `artifact_root` values in an otherwise contract-valid collect manifest.
- [x] Keep all schema, authored-document, evidence and semantic failures fail-closed, with atomic
      write, full task-aware revalidation, rollback and explicit recovery telemetry.
- [x] Add a provider-free fixture matching the observed one-character shard ID typo and negative
      no-mutation coverage.
- [x] Pass full provider-free DoD/offline closure, merge, establish a new qualification SHA and
      restart R3 from fresh smoke.
- [ ] Confirm the task-identity recovery boundary in fresh R3 release constituents before archive.

### Non-goals
- [x] Do not change schemas, public API, provider prompts/models, timeout durations, canonical
      matrices, curated repositories or release acceptance thresholds.
- [x] Do not repair nested identities, evidence paths, semantic content or malformed manifests.
- [x] Do not accept the failed smoke as release evidence.

### Approach
1. Classify only the existing `shard pack manifest task identity is invalid` validation family.
2. Before mutation, require the manifest to pass all task-independent schema, document, evidence
   and semantic validation.
3. Replace mismatched non-empty assigned top-level identities, write atomically, run the complete
   task-aware collect validator and restore the original bytes on any failure.
4. Record before/after digests, corrected field names and an operator-review warning.

### Files expected to change
- `internal/runtime/providercommon/{artifact_recovery,collect_manifest_shape_recovery,diagnostics,validation_issues}.go`
- matching providercommon tests
- `fixtures/scenarios/collect-manifest-task-identity-typo/shard-pack-manifest.json`
- runtime architecture/spec/testing documentation and this ExecPlan

### Acceptance criteria
- [x] The live-observed one-character `shard_id` typo is restored without a provider repair call.
- [x] A second contract error prevents mutation, and failed final validation restores original bytes.
- [x] Recovery changes no manifest content beyond assigned top-level task identity fields.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.

### Risks
- Deterministic identity correction could hide a manifest copied from another task. Eligibility is
  therefore limited to a manifest whose complete task-independent contract already validates;
  the task-aware validator remains the final authority and telemetry prevents silent normalization.

### Progress log
- 2026-08-01: Stopped R3 after the fresh Bank smoke failed one refresh shard on the exact
  assigned-versus-authored shard ID mismatch. Regression and release matrices were not started.
- 2026-08-01: Added typed, atomic task-identity recovery plus the live-shaped positive fixture and
  a negative other-contract-error fixture; focused tests pass.
- 2026-08-01: Focused positive/negative/classifier regressions passed 20 repetitions; the full
  providercommon package passed. Exact Go 1.25.10 / Node 22.21.1 / npm 10.9.4
  `make contracts test lint build` and a separate `make offline-closure` completed with exit code 0,
  including race suites, 263 Python tests, 158 UI tests, 7 rendered mock scenarios, 90 readable
  fixture artifacts, embedded UI drift and live/product boundary checks. The implementation
  worktree contains only the expected code/docs/fixture diff and all 12 pinned source repositories
  remain clean.
- 2026-08-01: PR #194 merged as `0214fc6d369bb48d409277980ac15e001f0ecb30`; detached
  post-merge offline closure passed. Fresh Bank smoke completed all collect shards, including the
  previously mismatched combined source shard, then exposed the separate nested proposal-section
  validator defect tracked by the plan above.

## EP-20260801-r3-focused-repair-fresh-mutation

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Confirm the fresh-mutation boundary in the fresh R3 release constituents before archiving.

### Context
Post-merge qualification SHA `9f02f259dfb99d2f78961ec61157845d3d959623` passed offline
closure and fresh smoke `smoke-tiny-bank-20260801T121843Z`. The first `regres fast` attempt then
hit a qwen pre-artifact collect stall. Its single fresh phase rerun
`regres-fast-bank-openedx-20260801T141407Z` completed all ten collect shards, but normal
`init.step2.asis_docs` left three valid-looking markdown files without the required manifest.
The scheduled focused repair inherited those old artifact mtimes, so the shared monitor classified
the new command as a post-artifact stall after two seconds and exhausted repair before it had its
own bounded write window. R3 stopped before Open edX completed and none of these runs are evidence.

### Goals (must have)
- [x] Give every focused repair command an invocation-local fresh artifact mutation baseline.
- [x] Keep preceding partial artifacts readable as repair input without counting them as current
      command progress.
- [x] Preserve fail-closed validation, bounded pre/post-artifact windows and controlled-stop rules.
- [x] Add a provider-free regression and synchronize runtime documentation.
- [x] Pass full provider-free DoD/offline closure, merge, establish a new qualification SHA and
      restart R3 from fresh smoke.
- [ ] Confirm the fresh-mutation boundary in the fresh R3 release constituents before archiving.

### Non-goals
- [x] Do not change schemas, public API, provider commands/models, timeout durations, canonical
      matrices, curated repositories or release acceptance thresholds.
- [x] Do not accept or reuse either failed `regres fast` attempt as release evidence.
- [x] Do not synthesize provider-authored artifacts or weaken strict draft validation.

### Approach
1. Set `FreshArtifactMutationAfter` when building the common focused-repair activity policy, as
   already done for collect-pair repair and draft enrichment.
2. Pin the invocation-local threshold in providercommon policy tests; retain the existing monitor
   regression proving stale files are treated as pre-artifact state until fresh mutation.
3. Run focused tests repeatedly, then the complete deterministic DoD and offline closure.

### Files expected to change
- `internal/runtime/providercommon/artifact_recovery.go`
- `internal/runtime/providercommon/engine_test.go`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] A focused repair never enters post-artifact state solely from files last mutated before that
      repair invocation.
- [x] A fresh repair mutation is still monitored and strict artifact validation remains unchanged.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.

### Risks
- A threshold taken after a very fast provider write could hide genuine progress. The baseline is
  therefore established while constructing the policy before process launch, with the existing
  one-millisecond clock-tolerance used by the other repair modes.

### Progress log
- 2026-08-01: Stopped the dependent Open edX phase after the second Bank attempt reproduced
  repair/stall pressure. Diagnostics showed the repair process ran for two seconds while its
  `last_artifact_mutation_at` still pointed to markdown written 94 seconds before repair admission.
- 2026-08-01: Added the common focused-repair fresh-mutation baseline and pinned it alongside the
  stale-artifact monitor regression; focused tests passed 20 repetitions. Exact-toolchain
  `make contracts test lint build` and `make offline-closure` passed, including race suites,
  263 Python tests, 158 UI tests, 7 rendered mock scenarios, readable-fixture drift, embedded UI
  drift and live/product boundary checks. All 12 pinned source repositories remained clean.
- 2026-08-01: PR #193 merged as `bd52e7aee0793b8cd7471b8cb9507738b4a42129`; detached
  post-merge offline closure passed. Fresh Bank smoke progressed beyond the stale focused-repair
  failure and exposed a separate collect task-identity typo tracked by the plan above.

## EP-20260729-r3-validator-evidence-advisory-boundary

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Confirm the advisory boundary in the fresh R3 release constituents before archiving this plan.

### Context
Fresh diagnostic smoke `smoke-tiny-bank-20260729T075044Z` on qualification SHA
`aa69d16191f93311560190fc467edd534ed5e567` completed init and produced a structurally valid
refresh snapshot, but `refresh.step3.findings` returned `FAIL` only for source-repository security
observations (demo credentials and a committed example JWT key). This crossed the MVP boundary:
security/compliance observations are advisory findings, while the validator release gate is limited
to staged artifact integrity and contract correctness. R3 stopped before regression matrices.

### Goals (must have)
- [x] Make the enforced validator prompt state that source-content security/compliance/architecture
      risk observations belong in findings/questions and cannot produce blocking `issues[]`.
- [x] After deterministic staged validation succeeds, reconcile a provider `FAIL` to `PASS` only
      when every error issue resolves to current-run repository evidence rather than a staged
      artifact/document contract target; retain those issues as warnings.
- [x] Keep technical staged artifact/index/reference/document failures blocking.
- [x] Pin the live-observed Bank of Anthos verdict/citation combination in a provider-free fixture.
- [x] Pass full provider-free DoD/offline closure, merge, and restart R3 from a fresh smoke.
- [ ] Confirm the advisory boundary in the fresh R3 release constituents before archiving this plan.

### Non-goals
- [x] Do not add security/compliance enforcement, change schemas/public API/workspace contracts, or
      weaken deterministic staged artifact validation.
- [x] Do not change canonical matrices, curated repositories, provider aliases/models, retry or
      timeout profiles.
- [x] Do not accept the stopped smoke or mix its artifacts into R3 evidence.

### Approach
1. Strengthen the shared normal and focused-repair validator contract with a technical-only
   PASS/FAIL boundary shared by all providers.
2. Run deterministic staged validation before verdict acceptance and add a fail-closed
   evidence-advisory reconciliation keyed by exact current-run citation identity/path.
3. Preserve advisory issue visibility by changing only `error` to `warning`, persisting the
   reconciled verdict, and logging the reconciliation.
4. Add positive live-fixture and negative staged-target/mismatched-citation tests; synchronize
   architecture, pipeline and live-runbook documentation.

### Files expected to change
- `internal/artifactquality/policy.go`
- `internal/artifactquality/policy_test.go`
- `internal/runtime/steppolicy/policy.go`
- `internal/runtime/steppolicy/policy_test.go`
- `internal/runtime/promptcontract/validator_repair.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `internal/orchestrator/docflow_repair.go`
- `internal/orchestrator/docflow_repair_test.go`
- `internal/orchestrator/runtime_task_apply.go`
- `internal/orchestrator/testdata/bank_validator_evidence_advisory.json`
- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/STAKEHOLDER_DOC.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Live-observed source-evidence issues remain visible as warnings and the verdict becomes `PASS`.
- [x] A staged final document/index issue or mismatched citation identity remains `FAIL`.
- [x] All provider prompt bodies carry the same technical-only validator boundary.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.
- [x] The merged `main` SHA becomes the only new R3 qualification input.

### Risks
- An overly broad downgrade could hide a genuine artifact defect. Reconciliation therefore runs
  only after deterministic staged validation passes and requires every error issue to match an
  exact current-run citation path/identity; staged targets, missing paths and mismatches fail closed.

### Progress log
- 2026-07-29: Classified the fresh smoke as a product contract-boundary defect and stopped R3
  before `regres fast`; no stopped/partial evidence will be reused.
- 2026-08-01: Implemented the fail-closed technical-only validator boundary and pinned the exact
  Bank verdict/citation incident. Focused suites passed, the critical orchestrator cases passed
  20 repetitions, and the four previously clock-sensitive providercommon cases passed 10
  repetitions. Exact-toolchain `make contracts test lint build` and `make offline-closure` then
  passed, including race suites, 263 Python tests, 158 UI tests, 7 rendered mock scenarios,
  readable-fixture drift, contracts, lint, build and embedded-UI/source-repository drift checks.
  The slice is ready for review; R3 remains stopped until its merge commit is requalified.
- 2026-08-01: PR #192 merged as `9f02f259`; a detached worktree on that exact SHA passed fresh
  `make offline-closure`, source-repository and embedded-UI drift checks. Fresh qwen Bank smoke
  `smoke-tiny-bank-20260801T121843Z` passed with strict zero failures and no runtime-quality
  blockers, establishing the qualification input before the subsequent regression defect.

---

## EP-20260729-r3-cross-shard-semantic-aliases

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Pass full provider-free DoD/offline closure, merge, and restart from fresh smoke.

### Context
Fresh diagnostic smoke `smoke-tiny-bank-20260729T015647Z` on qualification SHA
`d33dac87c269b88a3854152175b64cf8fab26233` completed all collect shards and produced valid
provider-authored artifacts, but the validator correctly returned `FAIL`: cross-shard semantic
assembly retained `svc.balance-reader`/`svc.balancereader` and
`db.ledger-db`/`db.ledgerdb` because their evidence paths differed, then left
`store.ledgerdb` as a dangling edge endpoint. R3 stopped before regression matrices.

### Goals (must have)
- [x] Deterministically merge cross-shard entity aliases only when type, logical repo, normalized
      ID identity and ID leaf/name agree.
- [x] Rewrite exact aliases and uniquely resolvable endpoint tokens to the canonical entity ID.
- [x] Preserve validator fail-closed behavior for ambiguous endpoint tokens.
- [x] Pin the live-observed Bank of Anthos entity/evidence/edge combination in regression tests.
- [ ] Pass full provider-free DoD/offline closure, merge, and restart from fresh smoke.

### Non-goals
- [x] Do not change schemas, public runtime contracts, provider prompts, matrices, curated
      repositories, provider commands/models, retries, or timeout budgets.
- [x] Do not synthesize a validator verdict or accept the stopped smoke as R3 evidence.

### Approach
1. Add a conservative entity identity key independent of evidence path only when full ID variants
   normalize identically and ID leaf/name agree within the same entity type and logical repo.
2. Build an endpoint-only resolver from canonical IDs and retained aliases; use token fallback only
   when exactly one canonical entity owns that token.
3. Keep findings/questions on exact remaps and leave ambiguous edge aliases untouched for validator.
4. Run focused repetition, deterministic DoD and offline closure before merge.

### Files expected to change
- `internal/orchestrator/docflow.go`
- `internal/orchestrator/docflow_test.go`
- `internal/orchestrator/testdata/bank_cross_shard_semantic_aliases.json`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Live-observed cross-path aliases merge with all provenance retained.
- [x] `store.ledgerdb` resolves only when the normalized token has one canonical target.
- [x] Ambiguous token regression remains unresolved and validator-visible.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.
- [ ] The merged `main` SHA becomes the only new R3 qualification input.

### Risks
- Token fallback can over-map generic endpoint names. The resolver therefore applies only to edge
  endpoints, requires a unique canonical token target, and never changes ambiguous values or
  finding/question references.

### Progress log
- 2026-07-29: Classified the fresh smoke failure from its immutable validator verdict and stopped
  R3 before `regres fast`; no stopped/partial matrix evidence will be reused.
- 2026-07-29: Added conservative full-ID alias grouping, endpoint-only unique token resolution and
  ambiguity guards. Focused regressions passed 20 consecutive runs; full pinned offline closure
  passed race suites, 90 readable fixtures, UI `158/158` twice, mock E2E `7/7`, Go, Python
  `263/263`, contracts, lint, build, and deterministic embedded UI verification.

---

## EP-20260729-r3-overview-evidence-gap-quality-signal

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge and restart fresh smoke from the new qualification SHA.

### Context
Fresh diagnostic smoke on qualification SHA `718f6bb80533874f5cb36d7251f2f36524544b22`
completed successfully, but init quality emitted `artifact_quality.overview_placeholder` for a
substantive Architecture Home. The broad line heuristic matched the evidence-gap sentence
`No explicit CODEOWNERS or per-service ownership evidence has been read yet.` The subsequent
refresh artifact was clean, but R3 policy requires a provider-free fix and a new qualification SHA
before live qualification continues.

### Goals (must have)
- [x] Restrict the short `no … yet` placeholder heuristic to known empty surface labels.
- [x] Preserve detection of actual placeholders such as `Services: no services yet`.
- [x] Pin the live-observed evidence-gap sentence in a regression test.
- [x] Pass the full provider-free DoD and offline closure.
- [ ] Merge and restart fresh smoke from the new qualification SHA.

### Non-goals
- [x] Do not change schemas, runtime contracts, provider prompts, matrices, curated repositories,
      provider commands/models, retries, or timeout budgets.
- [x] Do not reuse the diagnostic smoke as R3 evidence.

### Approach
1. Replace the broad same-line substring check with a closed empty-surface subject allow-list.
2. Add positive placeholder and negative live-evidence regression fixtures.
3. Run focused tests, deterministic DoD and offline closure before merging.

### Files expected to change
- `internal/orchestrator/quality.go`
- `internal/orchestrator/quality_test.go`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Focused overview quality tests pass.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.
- [ ] The merged `main` SHA becomes the only new R3 qualification input.

### Risks
- Making the heuristic too narrow could miss novel placeholder wording. Explicit bootstrap markers
  and exact standalone `placeholder`/`todo` checks remain unchanged; this slice only narrows the
  ambiguous natural-language `no … yet` branch.

### Progress log
- 2026-07-29: Classified the fresh smoke signal as a deterministic false positive against the
  immutable init snapshot; stopped progression to `regres fast` under the R3 restart policy.
- 2026-07-29: Replaced the ambiguous substring heuristic with a closed empty-surface subject
  classifier. Focused tests passed 20 consecutive runs; full pinned offline closure passed with
  race suites, 90 readable fixtures, UI `158/158` twice, mock E2E `7/7`, Go, Python `263/263`,
  contracts, lint, build, and deterministic embedded UI verification.

---

## EP-20260728-r3-qwen-citation-closed-shape

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge, establish a new qualification SHA and restart from a fresh smoke ID.

### Context
Post-merge offline closure passed on qualification SHA
`8d0c182fdc3a63bd60f945918db71f8822520be2`. Fresh diagnostic smoke
`smoke-tiny-bank-20260728T190233Z` then failed its first live init collect shard because Qwen added
`provenance` to `citations[0]`. Schema validation correctly rejected the additional property and
scheduled `manifest_only_repair`; the repair exhausted as `runtime_stalled_after_artifacts`, so the
operator stopped the matrix. Later canceled shards are stop consequences and the matrix is not
qualification evidence.

### Goals (must have)
- [x] State the exact closed citation item field set in the bounded Qwen collect contract.
- [x] Explicitly forbid citation-level provenance while preserving semantic provenance guidance.
- [x] Pin the live-observed extra-field regression in shared prompt and adapter tests.
- [x] Pass the full pinned provider-free DoD/offline closure.
- [ ] Merge, establish a new qualification SHA and restart from a fresh smoke ID.

### Non-goals
- [x] Do not change schemas, validators, repair routing, provider commands/models, retry/timeout
      budgets, matrices or curated repositories.
- [x] Do not accept the stopped smoke or its repair output as R3 evidence.

### Approach
1. Replace the compact citation guidance with the schema's exact five-field allow-list.
2. Keep unique IDs, concrete paths, non-empty claim IDs and reciprocal document bindings.
3. Add positive closed-shape and negative citation-provenance assertions at prompt and adapter
   boundaries.
4. Synchronize architecture/runbook, pass DoD, merge and restart the full sequence.

### Files expected to change
- `internal/runtime/promptcontract/collect_repair.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `internal/runtime/qwencode/runner_test.go`
- `docs/ARCHITECTURE.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Qwen normal and first-focused collect prompts remain at most 6 KiB.
- [x] Focused prompt/adapter tests pass.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.

### Risks
- The word `provenance` remains necessary for semantic entities/edges/findings; the prompt must
  scope the prohibition to citations instead of banning provenance globally.

### Progress log
- 2026-07-28: Stopped fresh smoke after exact schema evidence:
  `/citations/0 ... additionalProperties 'provenance' not allowed`. The matrix is partial and will
  not be reused.
- 2026-07-28: Added the exact five-field citation allow-list and scoped provenance prohibition.
  Focused tests and the full pinned offline closure passed: race suites, 90 readable fixtures, UI
  `158/158`, mock E2E `7/7`, Go, Python `263/263`, contracts, lint, build and deterministic
  embedded UI verification.

---

## EP-20260728-r3-shutdown-test-barrier

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated fixture fix, establish a new qualification SHA and restart qualification.

### Context
Post-merge offline closure on qualification SHA
`388af86df5053bf515659e9f5f4e78d749a5a8dc` failed
`TestServiceShutdownCancelsActiveRunAndRejectsNewStarts`: its one-second shutdown context began
immediately after async admission, before the fixture established that the blocking runner had
started. Under the loaded full Go suite, scheduler/startup latency consumed that deadline and the
test reported `context deadline exceeded`. The earlier race pass succeeded, and no live phase was
started on this SHA. The first fix gate then exposed a second timing assumption in
`TestRunHeadlessProviderKeepsRepeatedStreamOnlyCollectPairRepairStallAsContractFailure`: its
full-engine setup could consume the outer context before entering the focused repair path, yielding
zero repair calls. The adjacent tests already cover full-engine routing; this case's intended
invariant is the two-attempt focused repair exhaustion itself.

### Goals (must have)
- [x] Establish a deterministic runner-start barrier before measuring shutdown cancellation.
- [x] Exercise repeated stream-only focused repair directly, without unrelated full-engine process
      startup/retry latency.
- [x] Keep the production shutdown path and its quiescence guarantees unchanged.
- [x] Stress both focused fixtures and pass the full pinned provider-free offline closure.
- [ ] Merge the isolated fixture fix, establish a new qualification SHA and restart qualification.

### Non-goals
- [x] Do not loosen the one-second shutdown assertion after the blocking runner is active.
- [x] Do not change production lifecycle behavior, schemas, live harness inputs, providers,
      matrices, curated repositories or timeout profiles.
- [x] Do not accept any evidence from the failed post-merge gate.

### Approach
1. Reuse the existing counting blocking runner used by the adjacent pending-run shutdown test.
2. Wait on the shared bounded runner-start helper before creating the shutdown timeout context.
3. Drive the focused collect-pair recovery helper with an explicit diagnostic result and missing
   manifest error, preserving both real stream-only repair subprocess attempts.
4. Stress both focused tests, run the full deterministic DoD/offline closure, merge and restart.

### Files expected to change
- `internal/orchestrator/run_lifecycle_test.go`
- `internal/runtime/providercommon/engine_test.go`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Focused shutdown lifecycle test passes repeatedly.
- [x] The test still requires shutdown completion within one second after runner activation.
- [x] Focused collect-pair exhaustion performs and verifies exactly two monitored repair attempts.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.

### Risks
- A broad timeout increase would hide lifecycle regressions. This slice adds only the missing
  precondition barrier and leaves the measured shutdown budget intact. The collect-pair fixture
  still executes and monitors both real stream-only repair processes; only unrelated initial
  provider routing is removed from this focused exhaustion assertion.

### Progress log
- 2026-07-28: Stopped qualification before live execution after the post-merge offline gate failed
  at the shutdown fixture. Exact failure: `shutdown service: context deadline exceeded`.
- 2026-07-28: The first fix gate passed the shutdown race suite, then stopped at a second fixture
  assumption: focused collect-pair exhaustion observed zero repair calls because the outer context
  expired in unrelated full-engine setup. No failed gate output is qualification evidence.
- 2026-07-28: Final fixtures passed stress runs: shutdown `100/100` plus race `25/25`, focused
  collect-pair exhaustion `25/25` plus race `10/10`. The full pinned offline closure then passed
  race suites, 90 readable fixtures, UI `158/158`, mock E2E `7/7`, Go, Python `263/263`,
  contracts, lint, build and deterministic embedded UI verification.

---

## EP-20260728-r3-qwen-citation-binding

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated fix, establish a new qualification SHA and restart R3 from a fresh smoke ID.

### Context
Fresh standalone regression `regres-fast-bank-openedx-20260728T164734Z` on qualification SHA
`a61aae6e8711a7dc940406b320019cfb0aeb2c3a` completed Bank init cleanly, then stopped during
Bank refresh because the Qwen-authored manifest referenced unknown citation id
`cite.backup.readme` from `documents[0].citation_ids`. Validation correctly scheduled
`manifest_only_repair`; Open edX was not run, and the partial matrix is not qualification evidence.

### Goals (must have)
- [x] Require reciprocal document/citation bindings in the bounded Qwen normal and first-focused
      collect contract.
- [x] Add provider-free prompt and adapter regressions for an omitted citation reference.
- [x] Preserve the 6 KiB prompt cap, atomic pair sequencing, semantic bounds and non-Qwen behavior.
- [x] Pass the full pinned provider-free DoD/offline closure.
- [ ] Merge the isolated fix, establish a new qualification SHA and restart R3 from a fresh smoke
      ID.

### Non-goals
- [x] Do not change schemas, validators, recovery routing, provider commands/models, canonical
      matrices, curated repositories, retry counts or timeout profiles.
- [x] Do not accept the stopped regression matrix or repaired artifacts as R3 evidence.

### Approach
1. Replace the compact citation shape hint with an explicit two-way identity invariant.
2. Pin both directions and the omitted-citation prohibition in prompt/adapter tests.
3. Synchronize runbook/architecture, pass the full DoD, merge and restart every live phase.

### Files expected to change
- `internal/runtime/promptcontract/collect_repair.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `internal/runtime/qwencode/runner_test.go`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/ARCHITECTURE.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Qwen normal and first-focused collect prompts remain at most 6 KiB.
- [x] Focused prompt/adapter tests pass.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.

### Risks
- Extra identity wording consumes the strict Qwen prompt budget; the change replaces the previous
  single-citation shape line rather than adding a schema example.

### Progress log
- 2026-07-28: Stopped the fresh regression matrix at the first provider-authored contract defect;
  the exact validator error was `documents[0] references unknown citation_id
  "cite.backup.readme"`. No stopped/partial output will be reused.
- 2026-07-28: Replaced the compact citation hint with a reciprocal document/citation invariant.
  Focused prompt/adapter tests and the full pinned offline closure passed: race suites, 90 readable
  fixtures, UI `158/158`, mock E2E `7/7`, Go, Python `263/263`, contracts, lint, build and
  deterministic embedded UI verification.

---

## EP-20260728-r3-preflight-retry-test-quiescence

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated fix and restart R3 from a new qualification SHA and fresh smoke ID.

### Context
Fresh standalone regression matrix `regres-fast-bank-openedx-20260728T144845Z` stopped before
provider execution because the deterministic precheck failed
`test_claude_artifact_smoke_retries_timeout_even_with_output`. The test used a two-second real
process timeout and assumed the first shell stub would be scheduled soon enough to persist its
attempt counter. Under the loaded full Python suite that assumption failed once; 30 isolated
repetitions then passed, confirming a timing-sensitive test fixture rather than a provider or
production harness failure. The partial matrix is not qualification evidence.

### Goals (must have)
- [x] Replace wall-clock-dependent Claude retry-policy fixtures with deterministic mocked probe
      outcomes for both empty-output and output-bearing timeouts.
- [x] Preserve assertions that exactly one retry occurs and a valid second-attempt sentinel produces
      `ready`/`artifact_smoke=passed`.
- [x] Pass focused stress plus the full pinned provider-free DoD/offline closure.
- [ ] Merge the isolated fix and restart R3 from a new qualification SHA and fresh smoke ID.

### Non-goals
- [x] Do not change production preflight logic, provider commands, timeout/retry budgets, canonical
      matrices, curated repositories, schemas or runtime contracts.
- [x] Do not accept any profile from the stopped regression matrix as R3 evidence.

### Approach
1. Mock only the process supervisor boundary in the two retry-policy unit tests.
2. Return a real version result, raise a deterministic first-attempt `TimeoutExpired`, and write the
   expected sentinel on the second artifact-smoke attempt.
3. Stress the focused tests, run the full deterministic DoD/offline closure, merge, and restart all
   live phases from the new `main` SHA.

### Files expected to change
- `scripts/tests/write_batch_preflight_test.py`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Both Claude timeout retry tests are wall-clock independent and assert exactly two smoke
      attempts.
- [x] Focused tests pass repeatedly under load.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass with
      the pinned toolchains.

### Risks
- Over-mocking could stop exercising process-group termination. This slice keeps production code
  unchanged and scopes the mocked boundary to retry-policy tests; watchdog process-group behavior
  remains covered separately.

### Progress log
- 2026-07-28: Stopped the fresh regression matrix after the bank profile reported
  `precheck_failed`; Open edX had only begun its own precheck and is not accepted. The failing test
  passed 30 consecutive isolated repetitions, confirming a scheduler-sensitive fixture.
- 2026-07-28: Replaced both wall-clock retry-policy fixtures with deterministic supervisor outcomes.
  The two focused cases passed 200 consecutive paired repetitions; the full test module passed
  `32/32`; pinned offline closure passed race suites, 90 readable fixtures, UI `158/158`, mock E2E
  `7/7`, Go, Python `263/263`, contracts, lint, build and deterministic embedded UI verification.

---

## EP-20260728-r3-qwen-scope-array-contract

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Pass the full deterministic DoD/offline closure, merge the fix, qualify the new `main` SHA and restart R3 from a fresh smoke ID.

### Context
PR #179 merged as qualification SHA `0fe8793ce179e668b2cb5cd43a51ac1cc09500d6`, and the
fresh detached-worktree offline closure passed. Fresh diagnostic smoke
`smoke-tiny-bank-20260728T111000Z` proved that Qwen now writes the markdown/manifest pair in one
assistant response, but the `bank-of-anthos-docs` manifest encoded the single
`repo_scopes`/`path_scopes` values as strings. Schema validation correctly scheduled
`collect_manifest_repair`, so the matrix was stopped and is not qualification evidence.

### Goals (must have)
- [x] Render task `repo_scopes` and `path_scopes` as exact JSON array literals in the bounded Qwen
      normal/first-focused collect contract.
- [x] State explicitly that both fields remain arrays for a single value and must never be strings.
- [x] Add provider-free prompt and adapter regressions for the live-observed single-scope shape.
- [x] Preserve the prompt size cap, atomic pair sequencing, semantic bounds and Claude/Codex
      behavior.
- [ ] Pass the full deterministic DoD/offline closure, merge the fix, qualify the new `main` SHA and
      restart R3 from a fresh smoke ID.

### Non-goals
- [x] Do not change schemas, validators, recovery routing, provider commands/models, canonical
      matrices, curated repositories, retry counts or timeout profiles.
- [x] Do not accept the stopped/partial smoke or its manifest-only repair as release evidence.

### Approach
1. Replace ambiguous comma-joined scope prose in the compact Qwen collect contract with serialized
   JSON array literals.
2. Pin exact one- and multi-scope renderings in prompt/adapter tests.
3. Synchronize runbook/architecture wording, pass DoD, merge and restart the entire R3 sequence.

### Files expected to change
- `internal/runtime/promptcontract/collect_repair.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `internal/runtime/qwencode/runner_test.go`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/ARCHITECTURE.md`
- `docs/PLANS.md`

### Acceptance criteria
- [x] Qwen normal and pre-artifact repair prompts remain at most 6 KiB.
- [x] Prompt tests require `repo_scopes=["..."]` and `path_scopes=["..."]` JSON literals and reject
      scalar guidance.
- [x] `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass.

### Risks
- Extra prompt wording consumes the strict Qwen budget; the change replaces ambiguous identity
  lines instead of adding a full schema example.

### Progress log
- 2026-07-28: Stopped fresh smoke after exact validation evidence showed
  `path_scopes` was a string and `collect_manifest_repair` had been scheduled. The same manifest
  also encoded `repo_scopes` as a string; neither repaired nor partial artifacts are accepted.
- 2026-07-28: Replaced ambiguous comma-joined scope prose with exact JSON array literals in the
  bounded Qwen normal/first-focused contract. Focused prompt/adapter tests and the full pinned
  provider-free DoD/offline closure passed; schemas, validators, recovery and live harness inputs
  remain unchanged.

## EP-20260722-post-implementation-trust-audit

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Restart Epic 18 R3 only after the Epic 22 closure gate; do not reuse stopped matrix evidence.

### Context
Epic 20 and Epic 21 are implementation-complete, and the two latest provider-free Epic 18
remediations are merged in PR #171 (`ca8c3f67`) and PR #172 (`a633e3ce`). Before restarting the
trusted-machine R3 sequence, a static/provider-free code audit and preserved historical artifact
review found additional correctness, snapshot, refresh, UI-state and bidirectional live/product
isolation gaps. They are tracked as release-blocking Epic 22 rather than reopening or duplicating
the original Epic 19–21 delivery scopes.

### Goals (must have)
- [x] Deliver Epic 22 slices `22A..22O` in order as reviewable implementation units with child ExecPlans.
- [x] Close run/history/session/Git races and filesystem/snapshot trust-boundary defects.
- [x] Make refresh scope and preserved baseline evidence deterministic and immutable.
- [x] Replace stringly recovery routing with typed, bounded validation/recovery state.
- [x] Add a provider-free artifact integrity auditor and historical incident regression corpus.
- [x] Prove the live harness and product exchange only public contracts and do not author/import one
      another's state or implementation logic.
- [x] Complete Changes/Knowledge/QA/EvidenceViewer request identity, responsive and accessibility
      acceptance offline.
- [x] Pass the combined deterministic closure gate in the current working tree.
- [x] Record clean reviewed R3 qualification SHA `e8055d65699ed63623f62ad99c3b8406f79c030d`.
- [ ] Restart Epic 18 R3 only after the Epic 22 closure gate; do not reuse stopped matrix evidence.

### Non-goals
- [x] Do not run live E2E while implementing or accepting Epic 22.
- [x] Do not add hosted mode, security/compliance enforcement or new runtime providers.
- [x] Do not start Epics 12/13 or K5–K7.
- [x] Do not make live matrix identity, verdicts, assessments or environment part of product behavior.
- [x] Do not write to analyzed source repositories or weaken artifact validation to obtain PASS.

### Approach
1. Land core integrity slices `22A..22F`: registry/history, filesystem containment, snapshot resolver,
   glob semantics, immutable selective baseline and serialized session/Git coordination.
2. Land artifact/recovery isolation slices `22G..22I`: typed recovery, provider-free auditor and
   bidirectional live/product separation.
3. Land ProductShell correctness slices `22J..22N`: route/view truth, stale-response suppression,
   Knowledge/QA evidence authorities, EvidenceViewer correctness and responsive/a11y completion.
4. Run `22O` as one provider-free offline closure gate with race/fault/path/incident/UI/boundary
   suites plus the full deterministic DoD on a host with adequate disk.
5. From the recorded clean merge commit, execute Epic 18 R3 and its bounded evidence PR. The owner
   explicitly requested that `K2b -> K4 -> K3A -> K3B -> 9D -> cleanup` be implemented locally
   before that live gate; this does not replace composite PASS.

### Delivery work breakdown

This section is the program-level implementation map. Before changing code for a selected slice,
create a focused child ExecPlan with exact goals/non-goals, the final file list and fixtures for
that PR. Do not combine adjacent slices merely because they touch the same module.

#### Phase 0 — Qualification-ready development host

Deliverable:
- use exact Go/Node/npm versions from `.go-version` and `.node-version`;
- keep at least 5 GiB free on every workspace/temp volume used by tests, with additional headroom
  before `22O`;
- preserve a clean starting tree and do not touch unrelated user-owned files;
- record baseline `make contracts`, `make test`, `make lint`, and `make build` results before `22A`.

Exit gate:
- all four deterministic commands pass through the repository tool resolvers;
- failures are classified as product, fixture, toolchain or host-capacity failures rather than
  bypassed with version/precheck overrides.

#### Phase 1 — Core integrity (`22A..22F`)

`22A — Immutable run registry and transactional history`

- Deliverable: one deep-copy boundary for every `RunInfo` read/write path and one serialized
  transition primitive that persists the candidate history before publishing it in memory.
- Primary files: `internal/orchestrator/orchestrator.go`,
  `internal/orchestrator/service_runs.go`, `internal/orchestrator/run_finalization.go`,
  run-history persistence helpers and `internal/api/server.go` only if the public canceled shape
  must change.
- Focused tests: `internal/orchestrator/run_lifecycle_test.go`, restart reconciliation tests,
  concurrent `GetRun`/`ListRuns` polling, refresh-summary mutation, queued replacement, cancel and
  injected write/rename/fsync failures under `go test -race`.
- Exit gate: failed persistence never leaks a newer in-memory state; restart reconstructs one
  active/pending/terminal truth; cancellation is stable across API, history and restart.

`22B — Symlink-safe workspace containment and atomic manifest writes`

- Deliverable: a centralized workspace-owned read/open/write primitive that resolves symlink
  identity fail-closed and an atomic `workspace.yaml` write path using same-directory temp,
  file-sync, rename and directory-sync.
- Primary files: `internal/workspace/fs.go`, `internal/workspace/root.go`,
  `internal/workspace/manifest.go`, `cmd/acp/main.go` and any direct workspace file writer found by
  the slice audit.
- Focused tests: `internal/workspace/fs_test.go`, root/manifest tests with final and ancestor
  symlinks, dangling links, replacement races, traversal, valid in-root policy and injected atomic
  write failures.
- Exit gate: no workspace spelling can read or write outside the resolved root; source repositories
  remain byte-identical; interrupted manifest replacement preserves previous valid bytes.

`22C — Server-owned selected-run snapshot resolver`

- Deliverable: one backend resolver and additive read endpoint that selects an exact run-owned final
  index, validates inventory membership and returns typed
  `available|partial|not_produced|unavailable|error` state without workspace fallback.
- Primary files: `internal/api/server.go`, a focused `internal/api` snapshot resolver module,
  `internal/contracts` only if a shared response contract is necessary,
  `ui/src/hooks/useRunArtifacts.ts` and `ui/src/lib/runApi.ts`.
- Focused tests: API tests for wrong/foreign run ID, traversal, duplicate canonical mapping, missing
  or stale index, missing artifact and out-of-root target; UI A-B-A, reload and partial-run tests.
- Exit gate: the browser no longer discovers final indexes or composes staged paths; selected-run
  bytes are either exact and indexed or represented by an explicit typed issue.

`22D — One recursive glob dialect`

- Deliverable: one package that compiles, validates and matches recursive include/exclude patterns
  for every source-scope consumer.
- Primary files: new `internal/pathscope` package, `internal/workspace/manifest.go`,
  `internal/refreshplan/plan.go`, `internal/orchestrator/sharding_planner.go`, source fingerprint,
  imports and QA collection consumers discovered by AST/usage search.
- Focused tests: a shared conformance table for root/nested files, `*`, `**`, excludes, separators,
  invalid syntax, multi-repo paths and deterministic ordering; adapter tests prove all consumers
  return the same match set.
- Exit gate: invalid scope is rejected before run admission and no consumer retains an independent
  glob implementation.

`22E — Immutable selective-refresh baseline and complete shard identity`

- Deliverable: preserve unaffected documents only from validator-promoted baseline staging bytes;
  validate full shard identity and artifact digests before deciding selective reuse.
- Primary files: `internal/refreshplan/baseline.go`, `internal/refreshplan/plan.go`,
  `internal/orchestrator/refresh_selective.go`,
  `internal/orchestrator/refresh_preservation.go`,
  `internal/orchestrator/refresh_materialization.go` and refresh audit contracts if additive
  provenance is required.
- Focused tests: canonical workspace edited after baseline, repeated shard ID under another scope,
  changed repo/path scope or revision, missing pack, digest mismatch, long logical ID/model path,
  and a positive byte-identical preservation case.
- Exit gate: every identity/digest ambiguity falls back to full refresh before provider execution;
  a valid reuse records immutable provenance and reproduces the exact baseline bytes.

`22F — Session-generation lease and Git/run coordination`

- Deliverable: one admission coordinator spanning session generation validation, run registration,
  active/pending publication and Git mutation preconditions.
- Primary files: `internal/api/server.go`, session/runtime profile handlers,
  `internal/orchestrator/service_runs.go` and existing Git confirmation/fingerprint helpers.
- Focused tests: deterministic channel/barrier tests for switch-vs-start, commit-vs-start,
  proposal-branch-vs-start, concurrent confirmations, shutdown and queued replacement.
- Exit gate: no admitted run is owned by an obsolete service generation; Git mutation cannot cross
  an active/pending boundary or use stale branch/HEAD/base/inventory evidence.

Phase 1 dependency rule:
- `22B` safe primitives must be reused by later snapshot, baseline and auditor work;
- `22C` becomes the selected-run authority consumed by `22H`, `22L` and `22M`;
- `22D` is the only scope matcher accepted by `22E`;
- do not start `22G` until the registry, filesystem, snapshot, scope and admission foundations are
  merged and their deterministic DoD is green.

#### Phase 2 — Validation, auditing and boundary isolation (`22G..22I`)

`22G — Typed validation issues and explicit recovery state machine`

- Deliverable: internal typed issue codes/classes/paths plus an explicit, bounded recovery state
  machine; public validator shapes remain unchanged unless a separate schema-first decision proves
  otherwise.
- Primary files: `internal/runtime/steppolicy/policy.go`,
  `internal/runtime/providercommon/artifact_recovery.go`,
  provider-common validation/diagnostic helpers and focused prompt-contract adapters.
- Focused tests: preserved Claude/Qwen/Codex incidents under `internal/runtime/testdata`, equivalent
  paraphrased messages, shuffled issue ordering, mixed classes, no-op repair and budget exhaustion.
- Exit gate: routing does not inspect `error.Error()` fragments; the same issue set always follows
  the same auditable transition path and terminates within a deterministic budget.

`22H — Provider-free artifact integrity auditor`

- Deliverable: new read-only `internal/artifactaudit` package with a bounded redacted JSON result
  for promoted-current and exact selected-run evidence.
- Primary files: new auditor package/fixtures, reuse-only boundaries in `internal/contracts`,
  `internal/artifactquality` and the `22C` resolver; expose only the minimal CLI/API surface required
  by backlog acceptance.
- Focused tests: one fixture per preserved historical incident plus a clean fixture; foreign run,
  broken reciprocity, missing evidence/digest, absolute/staging path, execution narration,
  Architecture Home gaps, scaffold and proposal/finding disconnect.
- Exit gate: repeated scans are byte-identical; all incident fixtures have stable issue codes; scan
  leaves workspace and source repository snapshots unchanged.

`22I — Bidirectional live E2E/product isolation`

- Deliverable: production runtime receives only canonical repository/evidence identities; live
  preparation observes public product state without synthesizing run history or importing product
  implementation helpers.
- Primary files: runtime prompt/include-dir/environment construction, `scripts/full-run-batch.sh`,
  `scripts/frontend-live-e2e.sh`, `ui/e2e/live-flow.spec.ts`,
  `scripts/tests/aor_live_boundary_test.py`, plus relocation of any live-only helper from `ui/src`.
- Focused tests: bidirectional Go/Python/TypeScript forbidden-dependency scans; provider prompt/env
  snapshots without matrix/batch/temp identities; frontend preparation test proving no product
  history/state rewrite.
- Exit gate: product has no matrix/profile/sweep/verdict/assessment vocabulary; harness uses only
  CLI/API/UI/artifact contracts and cannot author or repair product state.

#### Phase 3 — ProductShell correctness (`22J..22N`)

`22J — Changes views and workflow truth`

- Deliverable: separate typed Overview/Evidence/Findings/Diff/Proposals/Publish view models and
  server-backed Git state `clean|dirty|stale|blocked|unknown`.
- Primary files: `ui/src/features/changes/ChangesWorkspace.tsx`, stage/view components,
  `ui/src/lib/workflowState.ts`, Git API contracts/handlers and route integration in `ui/src/App.tsx`.
- Focused tests: table-driven route/view/source/Git-state rendering, action availability, refresh,
  deep reload and Back/Forward.
- Exit gate: changing a Changes route changes data and action semantics, not merely a tab label;
  client state cannot claim publication readiness without authoritative Git evidence.

`22K — URL/request identity and stale-response suppression`

- Deliverable: one immutable request identity
  `(run, source, artifact-or-entity, viewer-mode)` with generation/abort handling and explicit invalid
  URL sanitization.
- Primary files: `ui/src/lib/appRoutes.ts`, `ui/src/hooks/useRequestGate.ts`,
  `ui/src/hooks/useRunArtifacts.ts`, `ui/src/App.tsx` and affected viewer hooks.
- Focused tests: delayed A-B-A responses, source/viewer changes with the same artifact path, invalid
  explicit IDs/enums, `replaceState` canonicalization, user `pushState`, reload and popstate.
- Exit gate: a stale response cannot update another route identity and invalid explicit state is
  visible to the user rather than silently reinterpreted.

`22L — Knowledge and QA evidence authorities`

- Deliverable: shared authority resolver for
  `promoted_current|run_snapshot|qa_snapshot|qa_audit`; Knowledge excludes
  `reports/taskruns/**` and QA never falls back to another authority.
- Primary files: `internal/api/knowledge.go`, `internal/qa/service.go`,
  `internal/orchestrator/qa_runs.go`, selected-run resolver integration and Knowledge/Ask UI clients.
- Focused tests: current, historical, partial, canceled, missing, foreign and legacy snapshots;
  explicit taskrun exclusion and immutable QA context-pack/answer linkage.
- Exit gate: every response names one authority and exact evidence inventory; unavailable selected
  evidence returns a typed unavailable state rather than current-workspace content.

`22M — Evidence Viewer correctness`

- Deliverable: authority-aware link resolution, typed `Demo|Live|Unknown` provenance, explicit left
  and right diff identities and bounded read/render behavior.
- Primary files: `ui/src/components/EvidenceViewer.tsx`,
  `ui/src/components/MermaidPreview.tsx`, API artifact-link resolution from `22C/22L` and viewer
  tests.
- Focused tests: traversal, cross-run link, missing/unknown source, XSS, Mermaid failure, long line,
  oversized artifact, explicit diff sides and keyboard navigation.
- Exit gate: local links cannot escape their authority, unknown evidence is never labeled Live and
  oversized/unsafe content degrades to a usable bounded fallback.

`22N — Responsive and accessibility completion`

- Deliverable: safe-area-aware responsive shell/navigation, no hidden focus targets, accessible
  modal/tab/combobox interactions and deterministic mobile card layouts.
- Primary files: `ui/src/components/ProductShell.tsx`, `ModalDialog.tsx`, `TabNav.tsx`,
  `LocalPathCombobox.tsx`, `ui/src/styles.css`, Vitest tests and mock Playwright scenarios.
- Focused tests: rendered `1440`, `1280`, `1024`, `390x844`; keyboard-only flows, focus return/trap,
  touch targets, orientation, overflow, first-viewport action and critical axe violations.
- Exit gate: no global horizontal overflow or unreachable focused element; all required product
  destinations and primary actions remain visible and operable across the acceptance viewports.

#### Phase 4 — Offline closure (`22O`)

Deliverable:
- assemble the race/fault/symlink/glob/refresh/incident/auditor/boundary/request/UI/a11y suites into
  one deterministic provider-free closure gate;
- run `make contracts`, `make test`, `make lint`, and `make build` using pinned toolchains;
- verify generated `internal/api/ui_dist`, fixture exports and source-repository snapshots have only
  expected changes;
- merge from a clean tree and record the exact qualification commit in PLANS, stakeholder status
  and the R3 operator record.

Primary files:
- `Makefile`, existing `scripts/run-go.sh`, `scripts/run-python.sh`, `scripts/run-npm.sh`,
  test/golden ownership docs and only the smallest CI wiring needed to make the offline gate
  required;
- do not add a wrapper over `scripts/full-run-batch-matrix.sh`.

Exit gate:
- closure passes repeatedly from the same clean commit;
- no live provider or network dependency is present in required CI;
- `git status --short` after the gate contains no unexplained generated drift;
- the recorded commit is the only code input authorized for Epic 18 R3.

### Epic 18 R3 execution plan

R3 is evidence generation, not a code-remediation phase. Any product or harness fix invalidates the
qualification SHA and returns the program to the appropriate provider-free slice plus `22O`.

1. On a trusted host, verify clean tree, exact pinned toolchains, adequate disk, writable canonical
   roots, all `qwen`, `claude`, `codex` binaries, and every curated path checkout at its pinned SHA.
2. Execute a fresh direct smoke, then standalone canonical `regres fast` and `regres long`.
3. Execute fresh `release full` constituents with direct
   `scripts/full-run-batch-matrix.sh` invocations: release-fast, release-long and ftgo+sentry.
4. For every release constituent require explicit `baseline` and `parallel-default`, identical
   shard plan for the same `profile_id`, strict zero failures and snapshot-only frontend evidence.
5. Inspect public backend/UI/API/artifact evidence and write matching accepted
   `swe_ux_assessment_<matrix-id>.md` and
   `swe_artifact_quality_assessment_<matrix-id>.md`.
6. Run `scripts/verify-release-verdict.py` over all three fresh constituent verdict JSON files.
7. Publish the bounded evidence PR only when all machine verdicts are `PASS`, both assessments for
   every matrix are accepted and the qualification SHA is unchanged.

### Post-R3 product and cleanup plan

Start this queue only after composite R3 readiness:

1. `K2b`: extend provider-free Workspace Health using stable issue IDs/order without changing the
   response version. Add broken links, missing edge endpoints, duplicate aliases, unlinked
   findings, proposal-evidence gaps, malformed canonical files and orphan domain/team coverage.
2. `K4`: harden global claim/citation uniqueness, reciprocity, run isolation, concrete in-root
   evidence and deterministic IDs under existing public shapes. If a shape change is proven
   necessary, stop and split a schema-first PR.
3. `K3A`: add the Ask-to-Proposal backend mutation with answer digest precondition, immutable QA
   source package, exclusive atomic creation and full schema/spec/appendix/examples/fixtures/ADR
   synchronization.
4. `K3B`: add the ProductShell confirmation and navigation flow, focus/a11y handling, stale-answer
   recovery and Git-inventory invalidation.
5. `9D`: lock deterministic `acp qa` and `POST /api/qa/ask` compatibility through v1 in docs,
   examples and contract tests; do not add removal/deprecation behavior.
6. Cleanup PR 1: deterministic drift protection and ownership documentation for retained readable
   fixtures.
7. Cleanup PR 2: archive completed ExecPlans and reconcile tracker/PR references without product
   behavior changes.

Epics 12/13 and K5-K7 remain out of this execution plan and require a separate owner-approved Wave 1
discovery plan.

### Program verification and PR rules

- One slice equals one reviewable PR; `22A..22O` order is strict.
- Each child plan starts with a failing provider-free regression or fixture when the defect is
  reproducible, then implements the smallest code change that closes it.
- Contract/schema changes require the schema guardian workflow and synchronized
  `docs/spec/*`, `docs/APPENDIX_SCHEMAS.md`, examples, fixtures, validators, tests and ADR rationale.
- Core extraction/model/recovery changes add or update incident fixtures and golden outputs.
- Behavior changes synchronize README, architecture, pipeline/testing docs and stakeholder status
  in the same slice.
- Every implementation PR completes `make contracts`, `make test`, `make lint`, and `make build`;
  focused tests are evidence during development, not a replacement for DoD.
- Live E2E is forbidden during Epic 22 acceptance and is never made a required CI dependency.
- A failure discovered after merge reopens the owning slice and invalidates downstream
  qualification evidence; it is not patched opportunistically inside a later UI or cleanup PR.

### Files expected to change
- `docs/BACKLOG.md`, `docs/PLANS.md`, `docs/STAKEHOLDER_DOC.md` for initial program tracking.
- The program-level likely modules and tests are listed in the delivery work breakdown. The exact
  final file list is narrowed and approved by each child ExecPlan; this plan does not pre-authorize
  unrelated refactors.

### Acceptance criteria
- [x] Epic 22 scope, sequencing, boundaries and per-slice acceptance are recorded in the backlog.
- [x] Stakeholder and engineering status identify Epic 22 as the current release blocker before R3.
- [x] PR #171/#172 statuses are reconciled as merged without claiming that R3 restarted.
- [x] Every `22A..22N` slice is implemented with focused tests and full deterministic DoD.
- [x] `22O` passes all provider-free closure gates in the current working tree.
- [x] The same gate passes from clean reviewed qualification commit
      `e8055d65699ed63623f62ad99c3b8406f79c030d`.
- [ ] Epic 18 R3 obtains fresh individual and composite PASS evidence from that unchanged commit.

### Risks
- The audit spans independent correctness domains; combining slices would hide causal regressions,
  so ordering and small PR boundaries are mandatory.
- Historical raw workspaces were cleaned up; preserved minimized incidents are authoritative for
  regressions, while new acceptance evidence must be generated deterministically.
- A working-tree pass is not a qualification SHA. Any change made before review/commit requires
  rerunning `make offline-closure` on the resulting clean commit before R3.

### Progress log
- 2026-07-22: Reconciled PR #171/#172 as merged and recorded the audit findings as Epic 22.
- 2026-07-22: Confirmed that Epic 22 required CI is provider-free and that R3/live evidence cannot
  restart until the offline closure gate records a new clean qualification SHA.
- 2026-07-26: Re-audited the clean `f569585c` implementation against every open backlog item and
  expanded this program plan with per-slice deliverables, primary modules, focused tests,
  dependencies, closure gates, the exact R3 evidence sequence and the post-R3 queue.
- 2026-07-26: Completed `22A` with deep immutable run snapshots, persist-before-publish registry
  transactions, atomic pending replacement, bounded history diagnostics and consistent canceled
  terminal state. Focused race/fault/restart tests and full deterministic DoD pass; `22B` is next.
- 2026-07-26: Completed `22B` with descriptor-backed workspace containment, one atomic manifest
  writer and adversarial symlink coverage; `22C` is next.
- 2026-07-26: Completed `22C` with server-owned selected-run snapshot resolution, typed states and
  persisted late-document inventory; `22D` is next.
- 2026-07-26: Completed `22D` with one validated recursive pathscope dialect shared by refresh and
  shard planning; `22E` is next.
- 2026-07-26: Completed `22E`–`22O` and the requested post-R3 product/cleanup queue in the working
  tree. `make offline-closure` passed with 263 Python tests, 158 UI tests and 7 rendered mock
  scenarios. Live E2E was not run; a clean reviewed qualification SHA and Epic 18 R3 remain open.
- 2026-07-27: Committed the complete provider-free implementation as
  `e8055d65699ed63623f62ad99c3b8406f79c030d` and reran `make offline-closure` from an isolated
  clean detached worktree. All gates passed with no tracked drift. This SHA is the only authorized
  code input for the still-open Epic 18 R3; no live E2E was run.

## EP-20260719-epic18-targeted-architecture-home-repair

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the remediation and restart qualification from the new clean merge commit.

### Context
Post-PR #162 Claude smoke `smoke-tiny-bank-20260719T005002Z` completed init and refresh collect
`10/10`. Refresh step2 then failed after the provider removed the sole invalid Architecture Home
sentence from `overview.md`: the complete draft contract was valid, but the shared enrichment guard
still required byte changes in the already-valid `summary.md` and `architect-summary.md`.

### Goals (must have)
- [x] Accept a partial markdown write set only for the exact step2 Architecture Home validation
      error, when `overview.md` is freshly changed and the complete draft contract passes.
- [x] Preserve the all-markdown rewrite requirement for every other enrichment/retry path.
- [x] Add the live-observed targeted-rewrite regression and focused stress coverage.
- [x] Synchronize runtime/testing/live-gate documentation and pass full deterministic DoD.
- [ ] Merge the remediation and restart qualification from the new clean merge commit.

### Non-goals
- [x] Do not sanitize provider content or accept a still-invalid Architecture Home.
- [x] Do not relax write-set containment, sidecar handling, schema validation or other draft steps.
- [x] Do not change schemas, HTTP APIs, provider contracts, timeout policy or canonical matrices.

### Acceptance criteria
- [x] A fresh `overview.md` repair with byte-identical valid sibling documents is accepted only after
      strict full-set validation.
- [x] Existing partial/noop enrichment tests remain terminal outside the narrow Architecture Home case.
- [x] Focused recovery sequence passes 20 consecutive runs.
- [x] `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-19: Preserved the failed smoke, confirmed refresh collect `10/10`, and isolated the
  terminal error to the shared all-markdown-change guard after a successful targeted overview repair.
- 2026-07-19: Added the exact-error/fresh-overview/full-validation policy, proved it 20/20, and
  completed deterministic DoD with 261 Python and 142 UI tests plus lint and embedded UI build.

## EP-20260717-epic18-r3-product-shell-live-gate

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the slice, then restart trusted-host qualification from the new clean commit.

### Context
- Production UI already uses `Home / Runs / Knowledge / Changes`, contextual Setup and global
  Ask, but `ui/e2e/live-flow.spec.ts` and the release runbook still assert retired Console V2
  StageRail, right inspector and activity drawer surfaces.
- The stale live flow would fail independently of backend/provider quality and cannot produce a
  trustworthy UX assessment for the current product.

### Goals
- [x] Replace the last primary-navigation `stage-analysis` test anchor with `destination-runs`
  across production and deterministic tests.
- [x] Move live init-inspect to direct ProductShell routes, run deep-link reload/Back restoration,
  current-workspace Knowledge and snapshot-isolated Changes Evidence/Publish.
- [x] Verify Step review/Diagnostics instead of the retired global activity drawer.
- [x] Verify global Ask focus/Escape/return focus plus citation navigation and Return to Ask.
- [x] Add rendered overflow/first-viewport checks at `1440`, `1280`, `1024`, `390x844`, critical
  axe assertions and browser console error capture.
- [x] Rename durable screenshots around product destinations/tasks and synchronize harness tests,
  architecture, testing strategy and live runbook.
- [x] Pass focused TypeScript/Python/UI tests, mock E2E and full deterministic DoD.
- [ ] Merge the slice, then restart trusted-host qualification from the new clean commit.

### Non-goals
- No new public route, HTTP API, schema, workflow precedence, release matrix, runtime/provider or
  timeout contract; the existing `/home` route is only prevented from falling back to legacy
  stage-driven auto-navigation. No visual redesign.
- Minor spacing/typography polish does not block R3 unless it causes overflow, clipping,
  inaccessible focus or unreadable evidence.

### Acceptance
- Live flow has no positive dependency on `stage-rail`, `right-inspector`, `activity-drawer` or
  `stage-analysis`; explicit negative assertions keep the retired shell nodes out of production DOM.
- Home, Runs, Knowledge, Changes Evidence/Publish and Ask complete through current public routes
  and visible task anchors; snapshot source identity remains explicit.
- Supported viewports have no global horizontal overflow, dialogs remain keyboard-operable,
  critical axe violations and page/console errors are empty.
- `make contracts`, `make test`, `make lint`, `make build` and `npm run e2e:mock --prefix ui` pass.

### Result
- The production-bundle live flow passed against a Git-tracked fake-runtime workspace with nine
  current ProductShell screenshots, direct/deep routes, reload/Back restoration, snapshot evidence,
  Ask citation return, four viewport contracts, critical axe checks and an empty browser console.
- Rendered QA exposed and removed a legacy Home auto-redirect: `/home` now remains the URL source of
  truth when accepted evidence already exists. Runs history selection uses an explicit Runs route.
- The common `EvidenceViewer` is now the live assertion surface for Markdown and Mermaid; Publish
  selects the promoted Architecture Home instead of applying document-quality thresholds to a
  random runtime inventory entry.
- A pre-existing providercommon test flake was reproduced once in 25 iterations. Only its synthetic
  shell-start wall-clock cap was widened from `500ms` to `2s`; runtime/provider budgets are unchanged.
  The focused stress run then passed `20/20`.
- Final gates: 19 live-harness contract tests, 142 UI unit tests, 7/7 mock E2E, local production live
  flow PASS, `make contracts`, `make test` (261 Python tests plus all Go/UI packages), `make lint` and
  `make build` all pass on Node `22.21.1`.

### Queue routing

The July 2026 queue policy is preserved in the [September archive](archive/PLANS_ARCHIVE_2026-09.md#historical-continuous-backlog-queue-policy-july-2026).
Current work is selected through the [active plan index](#active-plan-index); release execution
requires the current canonical runbook and fresh qualification evidence. Historical commit IDs and
old queue ordering in progress logs do not authorize a new run.

## EP-20260623-karpathy-adoption-roadmap

Status: active — retained scope; reconcile current implementation before selecting a new slice.

Next action: Reconcile the recorded open criterion with current code/status before selecting work: Adopt Karpathy's "compiled knowledge artifact" framing for ACP without changing runtime contracts.

### Context
The Karpathy LLM Wiki analysis in `docs/archive/research/KARPATHY_LLM_WIKI_COMPARISON_2026-06-18.md` and `docs/archive/research/KARPATHY_ADOPTION_ANALYSIS_2026-06-18.md` concluded that ACP should not become a free-form personal/wiki system. The useful import is narrower: treat accepted architecture knowledge as a maintained, Git-versioned, provenance-backed artifact that compounds through explicit review.

Current ACP already has the stronger foundation needed for this:
- source repos and imported docs are raw/read-only inputs;
- runtime writes staged artifacts under explicit write roots;
- orchestrator validates schemas/manifests and controls canonical promotion;
- stable architecture knowledge lives in `reports/*`, `model/*`, `proposals/*`, `charter/*`, `reports/changelog/*` and run-scoped taskrun artifacts;
- `qa.ask` is intentionally run-scoped and does not mutate canonical reports/model.

The implementation roadmap must preserve MVP constraints:
- no hosted mode;
- no security/compliance enforcement expansion;
- required CI remains deterministic/fake and does not require live providers or network;
- canonical artifacts are not mutated directly by runtime providers;
- schema/contract changes require synchronized docs, validators, fixtures and tests.

This plan is a roadmap for owner selection. It does not override the existing queue unless the owner chooses one of its slices as the next engineering workstream.

### Goals (must have)
- [ ] Adopt Karpathy's "compiled knowledge artifact" framing for ACP without changing runtime contracts.
- [ ] Add a deterministic workspace health/lint surface over already published workspace artifacts.
- [ ] Add an explicit Ask-answer-to-proposal promotion path that keeps Q&A compounding reviewable and non-automatic.
- [ ] Harden citation/claim identity checks before introducing any new claim model.
- [ ] Defer claim ledger, contradiction policy and search projection until smaller slices prove the need and contracts are clear.
- [ ] Keep every slice reviewable, deterministic and covered by focused tests/fixtures.

### Non-goals
- [ ] Do not build a generic Obsidian-compatible personal wiki.
- [ ] Do not let runtime providers own or silently rewrite canonical workspace files.
- [ ] Do not auto-promote `qa.ask` answers into `reports/as-is/*`, `model/*` or `charter/*`.
- [ ] Do not add MCP memory server, hosted sharing, background autonomous rewrite, or vector DB as source of truth in MVP.
- [ ] Do not overload topology edges (`calls`, `reads`, `writes`, etc.) with general claim relations (`supports`, `contradicts`, `supersedes`).

### Approach
1) **Slice K1 - Product framing / terminology**
   - Update README, architecture overview and selected UI copy to describe `arch-workspace` as a validated compiled architecture knowledge base.
   - Keep this wording-only: no API/schema/runtime behavior changes.
   - Sync wording with the existing "not chat history" positioning.

2) **Slice K2 - Deterministic workspace health/lint**
   - Add a deterministic health scanner over current workspace artifacts, separate from `validator-verdict.json`.
   - Initial checks should use existing data only: broken artifact refs, old unresolved questions, findings without proposals, proposals without evidence refs, observation provenance gaps, duplicate entity alias candidates, orphan domain outputs and broken final/citation index links.
   - Publish a small report surface, for example `reports/health/workspace-health.json` plus `reports/health/workspace-health.md`, only if the artifact contract is documented and fixture-backed.
   - Surface the summary in `Readiness`/`Review` before adding any live-provider lint.

3) **Slice K3 - Ask answer promotion to proposal draft**
   - Add an explicit user action from a selected async QA answer to create a proposal draft package.
   - Preserve `qa.ask` read-only/canonical-non-mutating semantics.
   - Suggested output package:
     - `proposals/qa-synthesis-<run-id>-<slug>/proposal.md`
     - `proposals/qa-synthesis-<run-id>-<slug>/evidence.md`
     - `proposals/qa-synthesis-<run-id>-<slug>/source-qa-answer.json`
   - Ensure citations and unresolved assumptions are copied into the proposal package.
   - Route review through existing `Proposals`/`Publish` surfaces.

4) **Slice K4 - Citation/claim identity hardening**
   - Strengthen checks around existing `citation-index.json` and report claim IDs before adding new model directories.
   - Detect duplicate/unstable claim IDs, missing citation coverage for key report surfaces and citation refs that no longer resolve.
   - Prefer health-report warnings first; promote to validator/blocking behavior only with explicit policy.

5) **Slice K5 - Claim ledger prototype (post-MVP candidate)**
   - Only after K2/K4, consider `model/claims/*.yaml` for a narrow fact set.
   - Start with architecture facts that already map to existing model semantics: service uses datastore, service calls service/external, service exposes API, service publishes/subscribes topic.
   - Add schema/spec/fixtures/tests in the same slice; use `acp-schema-guardian`.

6) **Slice K6 - Contradiction review queue (post-claim-ledger)**
   - Add typed claim relations only after claims have stable IDs.
   - Treat contradiction as reviewable knowledge, not automatic cleanup.
   - Keep relation vocabulary small and policy-owned: `supports`, `contradicts`, `supersedes`.

7) **Slice K7 - Rebuildable search projection**
   - Add only if QA/context loading becomes a measured pain.
   - Keep search index disposable and rebuildable from canonical workspace files.
   - Do not make SQLite/FTS/vector output canonical or required external infrastructure.

### Minimal first deliverable
The first implementation slice should be K2, not K5/K6/K7. A deterministic health/lint report imports the most useful Karpathy maintenance loop while preserving ACP's current architecture and avoiding schema-heavy claim ontology work.

K1 can be bundled with K2 if the diff stays small; otherwise keep K1 as a docs/UX-only precursor.

### Files expected to change
K1:
- `README.md`
- `docs/ARCHITECTURE.md`
- selected UI copy in `ui/src/components/*` / `ui/src/lib/*` only if needed

K2:
- new or existing internal package for health scanning, for example `internal/workspacehealth/*`
- `internal/api/*` if exposing health through API
- `internal/orchestrator/*` only if health report is generated as part of run/promotion
- `ui/src/components/*`, `ui/src/hooks/*`, `ui/src/lib/*` for `Readiness`/`Review` summary
- `fixtures/scenarios/*` and/or dedicated health fixtures
- `docs/spec/PIPELINE_SPEC.md`, `docs/TESTING_STRATEGY.md`, `docs/APPENDIX_SCHEMAS.md` if a new report contract is introduced

K3:
- `internal/qa/*` and/or `internal/api/*`
- proposal package writer under orchestrator/API boundary
- `ui/src/components/*`, `ui/src/hooks/*`, `ui/src/lib/*`
- fixtures for QA answer promotion
- docs/spec/API and pipeline docs if a new endpoint is added

K4-K7:
- `schemas/*`, `docs/spec/*`, `fixtures/*`, `examples/*`, validators and ADR rationale as required by schema/contract policy

### Acceptance criteria
- [ ] K1 wording explains ACP as a compiled architecture knowledge base without weakening existing boundaries.
- [ ] K2 health checks are deterministic, fixture-backed and run without live providers/network.
- [ ] K2 does not replace step3 validator and does not block promotion unless a later policy says so.
- [ ] K3 creates proposal drafts only by explicit user action.
- [ ] K3 preserves canonical non-mutation of `qa.ask`.
- [ ] K4 detects claim/citation identity drift without introducing an unreviewed claim ontology.
- [ ] K5-K7 are not implemented without fresh schema/spec-first plans.
- [ ] Full DoD passes for each implementation slice: `make contracts`, `make test`, `make lint`, `make build`.

### Test plan
- K1:
  - `./scripts/run-go.sh test ./internal/docsync`
  - `git diff --check`
- K2:
  - unit tests for each health rule;
  - fixture workspace tests for clean/warn/error health reports;
  - API tests if health endpoint is added;
  - UI tests for summary rendering and empty/partial states;
  - docs-sync test for terminology and source-of-truth consistency.
- K3:
  - API tests for valid QA promotion, missing answer, stale run and path-safety;
  - proposal package golden/fixture tests;
  - UI tests for explicit promotion action and no automatic mutation;
  - regression asserting `qa.ask` still writes only run-scoped artifacts.
- K4-K7:
  - schema/contract tests, fixture/golden updates and `acp-schema-guardian` workflow when new contracts are introduced.

### Risks
- Health report can duplicate validator responsibilities if the scope is not explicit.
- Ask promotion can create proposal spam unless the UI makes it a deliberate review action.
- Claim ledger can become a second model if introduced before citation identity hardening.
- Search projection can become hidden source of truth unless it is rebuildable and disposable.
- Terminology changes can overpromise "memory" unless docs keep the staged/validated promotion boundary clear.

### Progress log
- 2026-06-23: Created adoption roadmap from Karpathy analysis. Selected deterministic workspace health/lint as the recommended first implementation slice, with Ask promotion and claim/citation hardening as follow-ups.

## EP-20260710-workspace-health-snapshot

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Complete full DoD for the implementation slice and archive/close the plan after owner review.

### Context
This is the concrete K2a implementation slice from `EP-20260623-karpathy-adoption-roadmap`.
It imports the Karpathy maintenance loop as a deterministic, read-only health snapshot over
accepted workspace artifacts, without turning ACP into a free-form wiki and without adding
canonical `reports/health/*` artifacts in this slice.

### Goals (must have)
- [x] Add deterministic `internal/workspacehealth` scanner over existing workspace files.
- [x] Expose `GET /api/workspace/health` as a read-only API endpoint.
- [x] Surface health status/counts in `Readiness` and detailed items in the right inspector.
- [x] Keep health advisory: no run/review/publish/Q&A blocking and no runtime/provider calls.
- [x] Update README/ARCHITECTURE/STAKEHOLDER/API docs for the implemented K2a behavior.
- [ ] Complete full DoD for the implementation slice and archive/close the plan after owner review.

### Non-goals
- [x] Do not persist `reports/health/workspace-health.json|md`.
- [x] Do not change `qa.ask` or add Ask-to-Proposal promotion in this slice.
- [x] Do not add claim ledger, contradiction policy, vector DB or search projection.
- [x] Do not introduce schemas for health output until/unless a persisted artifact is added.

### Approach
1) Implement scanner rules for observation model provenance without evidence, orphan domain outputs, proposal review sections and unresolved open-question count.
2) Add API route and tests that confirm pass/warn responses and read-only behavior.
3) Add typed frontend client/state and reuse existing `Readiness`/right-inspector surfaces.
4) Update docs and run focused backend/frontend/docs checks before full DoD.

### Files expected to change
- `internal/workspacehealth/*`
- `internal/api/server.go`, `internal/api/server_test.go`
- `ui/src/App.tsx`, `ui/src/components/StagePanels.tsx`, `ui/src/lib/*`, `ui/src/App.test.tsx`
- `README.md`, `docs/ARCHITECTURE.md`, `docs/STAKEHOLDER_DOC.md`, `docs/spec/API_SPEC.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] Clean workspace returns `status=pass` with no findings.
- [x] Missing observation evidence, orphan domain output and proposal review-section gaps return warning items.
- [x] Open questions return info item only.
- [x] Malformed model YAML returns health `status=fail` as report content, not API lifecycle failure.
- [x] UI states cover no findings, warning items and scan failed.
- [x] Health scan is read-only and does not mutate workspace files.

### Risks
- Health can duplicate validator semantics; K2a keeps rules advisory and outside promotion gates.
- Proposal section check is intentionally simple heading-based lint; richer proposal quality belongs in a later slice.
- Persisted health artifacts would require a new contract/schema/docs fixture pass and are deferred to K2b.

### Progress log
- 2026-07-10: Implemented K2a read-only workspace health snapshot, API endpoint, Readiness/right-inspector UI surface and docs sync. Focused Go/UI checks and full Go suite passed; canonical `make contracts|test|lint|build` remains blocked on this host until Node.js 22.21.1 is available.
## EP-20260711-run-pinned-evidence-review

Status: active — retained scope; reconcile current implementation before selecting a new slice.

Next action: Reconcile the recorded open criterion with current code/status before selecting work: Preserve top-level `run_id`/`generated_at` and document `staged_path` in the frontend final-run index contract.

### Context
Epic 20 originally started with the highest-risk trust defect found in Console V2: historical
review could lose staged identity and expose current canonical bytes. Epic 19 subsequently landed
typed `staged_path`, selected-run identity/containment checks and run-keyed artifact loading. This
slice now begins with a sufficiency audit and closes only the remaining explicit source-mode,
fail-closed edge semantics, atomic view-state and rendered same-path multi-run proof gaps, without
changing the canonical artifact schema.

Re-baseline note (2026-07-15): Epic 19 delivered typed staged snapshot foundations in
`appContracts.ts` and `useRunArtifacts.ts`. Before implementation, audit current behavior against
the goals below and retain only missing explicit source-mode, fail-closed, transaction and rendered
multi-run proof work; do not duplicate landed code.

### Goals (must have)
- [ ] Preserve top-level `run_id`/`generated_at` and document `staged_path` in the frontend
      final-run index contract.
- [ ] Resolve every selected-run final document from its staged path, including coverage and
      open-question artifacts when present in that run's index.
- [ ] Require index `run_id` to match the selected run and every `staged_path` to remain inside
      `reports/taskruns/<run_id>/staging/final/`.
- [ ] Load preview, coverage and open questions as one run-id-keyed snapshot transaction with
      stale-response suppression or abort behavior.
- [ ] Model `Run snapshot` and `Current workspace` as explicit, non-interchangeable source modes.
- [ ] Default selected History/Review evidence to `Run snapshot` and expose
      `Current workspace` only as an explicit user action.
- [ ] Show selected `run_id`, generated timestamp and source mode in the Review/Publish evidence
      context; runtime/provider authority remains scoped to Epic `20C`.
- [ ] Treat a missing, unreadable or incomplete staged snapshot as an explicit selected-run
      error; never fall back to canonical bytes.
- [ ] Add deterministic two-run regression coverage and rendered proof with the same canonical
      paths but different content.

### Non-goals
- [ ] Do not change `schemas/final-run-index.schema.json` or synthesize a second snapshot format.
- [ ] Do not redesign the full navigation, artifact viewer or Publish gate in this slice.
- [ ] Do not add persisted review approvals or exact file-scoped Git commit behavior.
- [ ] Do not change runtime promotion, canonical workspace writes or live E2E matrices.
- [ ] Do not require a live provider; required acceptance uses fake/recorded deterministic data.

### Approach
1) Extend the TypeScript final-index document contract with required `staged_path` and keep
   its top-level `run_id`/`generated_at` parsing aligned with the existing JSON schema.
2) Introduce a small evidence-source view model that distinguishes a selected run snapshot from
   the current canonical workspace.
3) Validate that the index belongs to the selected run and that normalized staged paths are
   contained under that run's `staging/final` root before issuing artifact reads.
4) Build one run-id-keyed snapshot load for preview, coverage and open questions; abort or ignore
   stale responses when selection changes. Do not retain stable-path fallback in snapshot mode.
5) Distinguish an optional document absent from the index (`Not produced for this run`) from an
   indexed document that cannot be read (`Snapshot unavailable`).
6) Add primary-context copy and explicit unavailable/corrupt snapshot recovery near the
   selected evidence rather than as a remote global error.
7) Add two run fixtures sharing canonical paths while staging different bytes; assert rapid
   run A -> run B -> run A switching never leaks canonical, cross-run or stale async content.
8) Run focused UI tests, typecheck/build, rendered multi-run QA and the repository Full DoD.
9) Update behavior documentation only where run-history/current-workspace semantics are
   described, then commit this slice independently before selecting `20B`.

### Files expected to change
- `ui/src/lib/appContracts.ts`
- `ui/src/hooks/useRunArtifacts.ts`
- `ui/src/hooks/useRunActions.ts`
- `ui/src/hooks/useRunExplorer.ts`
- `ui/src/App.tsx` and/or the touched Review/Publish context component
- `ui/src/App.test.tsx`
- deterministic `historical-run-snapshot` UI/Playwright scenario for same-path multi-run evidence
- `docs/ARCHITECTURE.md`, `docs/archive/design/UI_CONSOLE_V2_DESIGN.md` and testing docs only where the
  implemented source-mode behavior needs synchronization

### Acceptance criteria
- [ ] Run A and Run B use the same canonical overview/coverage/questions paths but distinct
      staged content; selecting either run shows only its own bytes and metadata.
- [ ] A final index whose `run_id` differs from the selected run is rejected, and normalized
      staged paths outside `reports/taskruns/<selected-run>/staging/final/` are never read.
- [ ] Rapid A -> B -> A selection cannot let a late response from B overwrite the final A
      snapshot.
- [ ] Changing the current canonical file after both runs does not alter either historical
      preview.
- [ ] Missing staged content renders `Snapshot unavailable` (or equivalent) with run/path
      context and does not issue a canonical-path read.
- [ ] An optional document absent from a run index renders `Not produced for this run` and is
      not treated as corrupt snapshot data.
- [ ] Review and Publish evidence context display run id, generated time and `Run snapshot`;
      explicit current-workspace mode displays `Current workspace`.
- [ ] Focused component tests, UI typecheck/build and rendered multi-run QA pass without console
      errors or document-level horizontal overflow.
- [ ] Full DoD passes: `make contracts`, `make test`, `make lint`, `make build`.
- [ ] Documentation and fixtures remain aligned with the unchanged final-run index schema.

### Risks
- Existing tests with different canonical paths per run can pass while hiding the bug; same-path,
  different-byte fixtures are mandatory.
- Some run indexes may not contain optional coverage/question documents. Snapshot mode must show
  `Not produced for this run` rather than borrowing current workspace files.
- A staged path can be syntactically present but point at another run or outside staging; path
  normalization and selected-run containment are part of the trust boundary.
- Independent async requests can resolve out of order during run switching; the selected run id
  must key the complete snapshot transaction, not just the primary preview.
- A missing staged file can be tempting to recover through canonical fallback; that would
  recreate the trust defect and is explicitly forbidden.
- Publish currently mixes run-scoped preview with workspace-scoped Git state. This slice labels
  those contexts but leaves full publication-boundary correction to `20B`.

### Progress log
- 2026-07-11: Selected `20A` as the first P0 slice after the full UX/UI trust and craft audit;
  recorded the dependency-ordered Epic 20 program in `docs/BACKLOG.md`. Implementation has not
  started.
- 2026-07-15: Epic 19 merged into `main` at `02716bb`; snapshot foundations now require a
  sufficiency audit before this implementation plan is activated.

## EP-20260712-evidence-backed-architecture-refresh

Status: active — retained scope; reconcile current implementation before selecting a new slice.

Next action: Reconcile the recorded open criterion with current code/status before selecting work: Make `reports/as-is/overview.md` the architecture workspace home without changing its path.

### Context
ProvenArch already creates validated, Git-versioned architecture artifacts in a separate workspace,
but its primary overview is not yet an intentional navigation home and refresh does not first explain
what source changed or why particular domains/artifacts must be regenerated. This Wave 1 plan adds an
evidence-backed architecture home followed by deterministic source revision and impact planning,
explainable no-op behavior, affected-only collect execution and finally safe surgical promotion.

The implementation must retain the current trust boundaries: source repos remain read-only; current
source evidence outranks Git messages; runtime outputs remain staged and validator-gated; unknown
impact causes conservative refresh; required CI remains deterministic and provider-independent.

### Goals (must have)
- [ ] Make `reports/as-is/overview.md` the architecture workspace home without changing its path.
- [ ] Add machine-checkable documentation-quality rules to `step2.asis_docs` and aligned guidance to `qa.ask`.
- [ ] Persist an exact per-repo source revision baseline for successful promoted architecture runs.
- [ ] Compute a deterministic refresh impact plan before provider collect execution.
- [ ] Add bounded recent Git intent evidence without treating commit messages as source truth.
- [ ] Support safe, explainable no-op refreshes.
- [ ] Dispatch collect only for affected shards/domains with conservative fallback for ambiguity.
- [ ] Preserve unaffected canonical artifacts byte-for-byte after dependency-aware promotion exists.
- [ ] Expose changed/preserved/uncertain decisions in existing operator surfaces.

### Non-goals
- [ ] Do not write into analyzed source repositories.
- [ ] Do not read the full Git history or let an LLM choose refresh scope without deterministic planning.
- [ ] Do not add hosted scheduling, external docs integrations, providers or security enforcement.
- [ ] Do not infer human acceptance from workspace Git history; use successful validator promotion until a separate acceptance contract exists.
- [ ] Do not call an unmapped in-scope change a no-op.
- [ ] Do not rename ProvenArch or replace `arch-workspace` with an OpenWiki-compatible layout.

### Approach
1) Deliver `21A` as the minimal useful slice: architecture home policy, docs/QA quality guidance, fake output and focused tests, with no schema changes.
2) Deliver `21B` as a contract slice for source revision baselines; use `acp-schema-guardian` and `acp-test-fixtures` and synchronize all required schema/spec/example/fixture/ADR surfaces.
3) Deliver `21C` as a pure deterministic planner over complete changed paths, prior shard/domain scopes and evidence dependencies; ambiguous input produces an explicit full-refresh plan.
4) Deliver `21D` only after planner fixtures prove no-op safety; no-op skips providers and canonical rewrites but persists taskrun rationale.
5) Deliver `21E` by filtering existing collect dispatch with the validated impact plan and passing only bounded affected-scope Git intent evidence to providers.
6) Deliver `21F` by merging freshly generated affected artifacts with explicitly selected known-good baseline artifacts, then validating the complete staged publication set before promotion.
7) Deliver `21G` after behavior exists: add operator-facing explanations, update product language and perform deterministic plus optional trusted-machine quality validation.

### Files expected to change
- `docs/BACKLOG.md`
- `docs/PLANS.md`
- `docs/STAKEHOLDER_DOC.md`
- `docs/ARCHITECTURE.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/spec/WORKSPACE_SPEC.md` only if persisted workspace configuration changes
- `docs/APPENDIX_SCHEMAS.md`, `schemas/*`, `examples/*`, `fixtures/*` for `21B/21C` contracts
- `internal/workspace/*` or a focused internal source-revision package
- `internal/orchestrator/*`
- `internal/runtime/steppolicy/*`
- `internal/runtimedrafts/*`
- `internal/runtime/fakeruntime/*`
- `internal/qa/*`
- `internal/artifactquality/*`
- focused Go/UI/scenario fixture tests for each slice

### Acceptance criteria
- [ ] Every implementation slice has focused fixtures/tests and passes Full DoD: `make contracts`, `make test`, `make lint`, `make build`.
- [ ] Contract slices synchronize schemas, specs, appendix, examples, fixtures, validators, tests and ADR rationale.
- [ ] No-op is possible only for unchanged clean revisions or out-of-scope-only changes.
- [ ] Dirty worktree, missing baseline, history rewrite, oversized range and unmapped in-scope changes are explicit conservative fallbacks.
- [ ] Planner input is never silently truncated; only provider history context is bounded.
- [ ] Commit messages remain secondary intent evidence and cannot override current source evidence.
- [ ] Selective promotion preserves unaffected canonical files byte-for-byte and validates one coherent final/citation index set.
- [ ] Required CI has no live provider or network dependency.

### Risks
- Git revision state and workspace artifact state can diverge; baseline selection must bind both to one successful promoted run.
- Path-to-domain mapping can be incomplete; correctness requires conservative fallback instead of optimistic skipping.
- Carrying forward artifacts can preserve stale evidence unless dependency and baseline provenance are explicit and validator-checked.
- Global overview/coverage/findings may depend on multiple domains, so surgical refresh must model aggregate dependencies rather than only copy files by path.
- Adding impact artifacts creates contract maintenance cost; schemas and examples must be designed once the required fields are proven by the planner slice.

### Progress log
- 2026-07-12: Created Wave 1 Epic 21 and decision-complete implementation sequence. Selected `21A` as the minimal first slice; no runtime or schema behavior changed.

---

## EP-20260629-live-e2e-artifact-summary-finalization

Status: active — retained scope; reconcile current implementation before selecting a new slice.

Next action: Reconcile the recorded open criterion with current code/status before selecting work: Reject step2 as-is markdown that claims shard manifests/evidence are absent when current-run typed shard summary has planned shards.

### Context
The latest accepted `claude-code` strict medium diagnostic proved the new execution gate shape (`execution_report_*`, selected-provider totals, no legacy mixed quality gate artifacts), but manual artifact review rejected FTGO artifact quality. The machine execution verdict was correctly separate from manual artifact quality, yet key FTGO markdown could still claim `Shard pack manifests: none observed` or stale final/citation index availability while current-run shard summaries and downstream indexes existed. The same run also showed that `step4.proposals` could pass with formally present but weak proposal/changelog markdown: empty findings/gaps sections, dangling references to findings "above", and generic follow-up actions. This slice closes those machine-checkable contradictions before opening the PR, while leaving broader truthfulness/readability to the SWE artifact-quality report.

### Goals (must have)
- [ ] Reject step2 as-is markdown that claims shard manifests/evidence are absent when current-run typed shard summary has planned shards.
- [ ] Reject stale `final-run-index` / `citation-index` availability claims, including `not yet present` variants, instead of promoting them as operator evidence.
- [ ] Require step2 overview to include concrete repo/path, citation, or staged artifact references when typed shard status is visible.
- [ ] Require step2 architect summary to include a decision-ready operator cue when typed shard status is visible.
- [ ] Require step4 proposal/changelog drafts to have non-empty required sections, current-run evidence refs and exact shard completeness when typed status is visible.
- [ ] Reject dangling proposal wording such as `findings above` and generic follow-up plans unless the proposed plan contains an explicit no-actionable-proposal gap tied to current-run evidence.
- [ ] Route these validation failures to the existing provider-authored shard-status/downstream-index enrichment retries; repeated noop/scaffold/stale output remains `runtime_contract_failed`.
- [ ] Keep `release_verdict_*` execution-only; artifact/UX acceptance remains separate SWE reports.
- [ ] Treat current `codex-code` quota/permission/readiness as an external operational blocker, not a repo-code blocker for this PR.

### Non-goals
- [ ] Do not change product UI/API behavior.
- [ ] Do not edit canonical matrix files or curated repo files.
- [ ] Do not add deterministic synthesis as a hidden success path.
- [ ] Do not commit generated live E2E evidence unless it is an intentional fixture/golden.

### Approach
1) Tighten shared runtime draft validation for FTGO-style stale/contradictory as-is markdown.
2) Update prompt contract tests and runtime draft tests for the observed failure shape.
3) Synchronize live E2E runbook, pipeline spec, architecture and testing strategy.
4) Run targeted tests, then full DoD.
5) Commit, run Claude strict medium diagnostic from a clean tree if host/provider preflight passes, push branch, open draft PR, and squash-merge only after CI is green.

### Files expected to change
- `internal/runtimedrafts/manifest.go`
- `internal/runtimedrafts/manifest_test.go`
- `internal/runtime/promptcontract/draft_repair.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `docs/PLANS.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`

### Acceptance criteria
- [ ] Targeted `go test ./internal/runtimedrafts ./internal/runtime/promptcontract` passes.
- [ ] Full DoD passes with exact Node toolchain.
- [ ] Claude strict medium diagnostic either passes with accepted manual artifact/UX assessment or records a concrete remaining blocker.
- [ ] PR body documents Codex readiness as external operational blocker if it is still not runnable.

### Risks
- Stricter markdown validation can convert previously passing provider output into `runtime_contract_failed`; this is intentional for machine-checkable contradictions, while subjective truthfulness remains manual artifact quality.
- Live rerun can still fail on host/provider quota, auth, network or timeout; classify those separately and do not bypass with test-only flags.

### Progress log
- 2026-06-29: Started finalization slice after manual review rejected the latest Claude FTGO artifacts despite clean machine execution separation.
- 2026-06-29: Added the observed weak FTGO `step4` proposal/changelog shape to the plan: proposal sections must be non-empty, evidence-backed and actionable, or explicitly state a no-actionable-proposal gap.
## EP-20260616-live-e2e-recovery-rerun-loop

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Harden step2 draft enrichment prompt/contract so enrichment reads bounded staged evidence and write-first overwrites `overview.md`, `summary.md`, and `architect-summary.md`.

### Context
Diagnostic `claude-code` medium (`regres long`) live runs exposed harness/runtime issues before a clean strict-medium acceptance loop can continue: `draft_artifact_enrichment` for `step2.asis_docs` did not rewrite all bootstrap draft files, batch reporting let a stale `runner_unavailable` classifier row override collect partial root cause, and the `f2e962f` rerun showed fully silent collect fresh retry exhaustion falling straight to `runner_unavailable` without a focused collect-pair repair attempt. The user requested fix -> DoD -> commit -> live rerun loop across `claude-code`/`codex-code` with target rotation, without product UI/API changes or canonical matrix/repo edits.

### Goals (must have)
- [ ] Harden step2 draft enrichment prompt/contract so enrichment reads bounded staged evidence and write-first overwrites `overview.md`, `summary.md`, and `architect-summary.md`.
- [ ] Keep bootstrap/noop enrichment as `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`; do not add deterministic synthesis as a hidden success path.
- [ ] Run one provider-authored collect-pair repair after exhausted fully silent collect retry before terminal `runner_unavailable`, while keeping invalid/noop repair terminal.
- [ ] Make collect-pair repair evidence-first without seed/fallback heredoc, and stop invalid observed artifacts by no-fresh-mutation window even when provider stdout remains active.
- [ ] Give collect-pair repair a fresh-mutation threshold, minimum 5-minute pre/post/partial artifact windows, and validation-specific prompt focus for directory-only evidence refs, process-contaminated markdown, and no-artifact stalls.
- [ ] Fix report classification so collect partial failures stay visible as `runtime_flow_failed`/`partial_failure_count`, while primary `failure_class` uses the concrete terminal/provider class when available.
- [ ] Ensure live UI dependency/browser precheck cannot hang indefinitely; timeout must become `precheck_failed` with log/report evidence before headless runtime starts.
- [ ] Update tests and live E2E docs/runbooks for the changed runtime/reporting behavior.
- [ ] Run full DoD, commit, then rerun strict medium live E2E from a clean tree.

### Non-goals
- [ ] Do not change product UI/API behavior.
- [ ] Do not edit canonical matrix files, curated repo files, or provider command shims.
- [ ] Do not treat diagnostic non-release evidence as release readiness.

### Approach
1) Patch promptcontract and reporting classification with focused unit/script tests.
2) Synchronize `docs/RELEASE_LIVE_E2E_RUNBOOK.md`, `docs/spec/PIPELINE_SPEC.md`, `docs/ARCHITECTURE.md`, and `docs/TESTING_STRATEGY.md`.
3) Run targeted tests, then full DoD (`make contracts`, `make test`, `make lint`, `make build`).
4) Commit the fix slice.
5) Run `regres long` medium through direct `scripts/full-run-batch-matrix.sh`, starting with `codex-code` and then `claude-code`; evaluate execution, artifact and UX quality; continue the requested provider loop if blockers remain.

### Files expected to change
- `internal/runtime/promptcontract/draft_repair.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `internal/runtime/providercommon/artifact_recovery.go`
- `internal/runtime/providercommon/engine_test.go`
- `scripts/e2e_report_classifiers.py`
- `scripts/e2e_batch_report.py`
- `scripts/tests/batch_failure_classification_test.py`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/TESTING_STRATEGY.md`
- `docs/PLANS.md`

### Acceptance criteria
- [ ] Prompt tests prove step2 enrichment has exact write-first targets and banned bootstrap scaffold.
- [ ] Providercommon tests prove exhausted fully silent collect retry invokes collect-pair repair and invalid/noop repair remains terminal.
- [ ] Script tests prove partial collect stays visible as a runtime-flow signal while provider classifier rows can set primary `runner_unavailable` / `runtime_contract_failed`.
- [ ] Script tests prove a hung UI precheck command is terminated and reported as `precheck_failed`.
- [ ] Full DoD passes before rerun.
- [ ] Strict medium rerun evidence is reported across execution quality, artifact quality and UX quality.

### Risks
- Live providers can still fail due to quota/auth/timeout, which must be classified separately from ACP regressions.
- The stricter prompt can increase provider work in medium/full runs; bounded staged evidence limits the blast radius.
- Failed backend refresh can still leave triage snapshots; frontend smoke must require a succeeded refresh row before using a snapshot, otherwise it can launch a heavy UI-side live run after an already terminal backend failure.

### Progress log
- 2026-06-16: Started fix slice from the failed `regres-long-posthog-ftgo-20260616T033256Z` diagnostic evidence.
- 2026-06-16: Committed draft enrichment/reporting fix `f2e962f`; strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260616T062005Z` still failed with partial collect (`posthog` 2 failed shards, `ftgo` 8 failed shards). Reports correctly kept primary `runtime_flow_failed`, selected-provider totals, `execution_report_*`, and no legacy mixed quality artifacts, but runtime showed repeated fully silent collect retry exhaustion without focused `collect_pair_repair`. Current fix slice adds that recovery path plus classifier/docs coverage.
- 2026-06-16: Committed collect silent-retry repair fix `532c967`; strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260616T100303Z` proved `collect_pair_repair` now schedules, but PostHog failed 14/16 shards because the pair-repair prompt still wrote seed/recovery fallback markdown and let Codex continue without fresh target mutation. The FTGO profile was manually interrupted after the same no-fresh-mutation pattern appeared in `step0.constitution` and would otherwise wait for the full medium step timeout. Current slice removes seed pair-repair prompt logic and makes invalid observed artifacts stop on the bounded partial-artifact window despite active stdout.
- 2026-06-18: After commits `5587298`, `6e84a9f`, and `8c452d0`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260618T083335Z` progressed PostHog init and refresh collect to 16/16 shards, preserving selected-provider totals and failed headless refresh row. It exposed a new primary blocker at `refresh.step2.asis_docs`: `draft_artifact_enrichment` read bounded evidence and then stopped after analysis-only prose (“I have enough bounded evidence to rewrite now”) without fresh writes, leaving `overview.md` and `architect-summary.md` bootstrap-only. Runtime correctly failed as `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`. The same run showed a secondary frontend harness issue: failed refresh had a triage snapshot directory, so frontend smoke started a heavy UI live run. Current slice makes enrichment command-first/write-first and makes frontend snapshot eligibility require latest headless refresh status `succeeded`.
- 2026-06-18: Started Live E2E Quality Loop fix slice for `codex-code`/`claude-code` medium reruns. Scope: make manifest-only collect repair provider-authored instead of hidden writer command, mark deterministic `collect_manifest_runtime_recovery` as explicit runtime warning/diagnostic evidence, preserve execution/artifact telemetry separation in `execution_report_*`, and add AOR boundary tests so live/manual release evidence names do not enter core runtime/orchestrator logic.
- 2026-06-19: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T190444Z` exposed that `posthog-ee` could still become a checkpointed shard after structural-invalid repo evidence paths (`ee/celery.py`, `ee/billing/__init__.py`, `ee/plugin-server/...`) plus exhausted `collect_manifest_repair`, because deterministic `collect_manifest_runtime_recovery` rebuilt a README-only manifest and marked the shard `succeeded`. Current slice removes deterministic post-repair fallback for structural-invalid manifests: missing/scaffold-only manifests may still use runtime recovery before provider repair, but exhausted structural manifest repair now remains terminal `runtime_contract_failed`.
- 2026-06-22: Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260622T045732Z` confirmed the step0 draft enrichment blocker was gone and selected-provider reporting/no-legacy execution reports remained correct, but both profiles failed during collect partials: PostHog primary classifier `runner_unavailable`, FTGO primary classifier `runtime_contract_failed`, with `runtime_flow_failed`/`partial_failure_count` as execution signals. Current slice persists per-shard `error_code` into shard summary JSON and aligns Python reports/docs so primary `failure_class` no longer collapses these cases back to generic `runtime_flow_failed`.
- 2026-06-22: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260622T081027Z` passed PostHog backend/frontend and completed FTGO collect 16/16 with no shard errors, but failed FTGO `init.step2.asis_docs` as `runtime_contract_failed`: `draft_artifact_enrichment` generated a filesystem command using missing `python`, so no markdown target was rewritten and strict validation rejected bootstrap-only `overview.md`/`summary.md`/`architect-summary.md`. Current slice adds a prompt-level `python3` requirement plus one provider-authored `draft_artifact_enrichment_python3_retry` for this missing-interpreter case, without deterministic draft synthesis or validation weakening.
- 2026-06-22: After commit `bebbd87`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260622T133647Z` preserved selected-provider totals, emitted `execution_report_*`, skipped frontend only because backend snapshots were missing, and kept artifact quality outside the machine verdict. It still failed both selected slots as `runtime_contract_failed`: PostHog `step4.proposals` rewrote proposal/changelog but claimed `final-run-index.json` had 0 observed documents because the provider counted a nonexistent `documents[]` field instead of current-run `canonical_documents[]`; FTGO refresh `step2.asis_docs` no-action retry printed a `python3 - <<'PY'` command as assistant text and left bootstrap markdown unchanged. Current slice adds explicit final/citation index schema instructions, rejects short `Draft surface initialized` scaffold markers, and adds one provider-authored `draft_artifact_enrichment_command_text_retry` for printed-command no-action output, without deterministic draft synthesis or validation weakening.
- 2026-06-22: Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260622T160036Z` ran from clean commit `9f921a6`. PostHog failed in `init.step0.constitution`: first enrichment and no-action retry made no fresh mutation, leaving bootstrap `charter-overview.md`; FTGO passed step0 enrichment but then hit repeated fully silent collect shard failures, with 1/16 collect shards succeeded and five consecutive `runner_unavailable` failures before the run was operator-interrupted to avoid burning the full medium window. Current slice keeps provider-authored-only success, extends step0 enrichment to a bounded 2-minute pre-artifact window with explicit repo-entrypoint targets in the compact retry, and makes sequential best-effort collect abort after five consecutive `runner_unavailable` failures while marking undispatched shards failed with the same provider class.
- 2026-06-22: Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260622T173826Z` ran from clean commit `e4208fb` and confirmed the new execution/artifact split still held (`execution_report_*`, no legacy quality gate artifacts, artifact quality not driving machine verdict), but both profiles failed in `init.step0.constitution` as `runtime_contract_failed`. PostHog successfully rewrote `charter-overview.md` with evidence-backed content after the compact retry, then drifted `constitution-draft.json` by adding unknown `status`, `content_digest`, and `enriched_at` fields. FTGO kept `charter-overview.md` bootstrap-only because its `git_url` resolved checkout existed only in `read_context_roots` under `.acp/repos`, while step0 enrichment include dirs added repo roots only from `workspace.yaml` path/validate fallback. Current slice adds one provider-authored `draft_artifact_enrichment_manifest_shape` retry without relaxing strict parsing, explicitly bans unknown manifest fields in enrichment prompts, and lets step0 enrichment include selected `git_url` repo roots from `read_context_roots` while keeping `step2/step4` bounded.
- 2026-06-18: Committed first loop slice `5587298`; diagnostic `codex-code` strict medium run `regres-long-posthog-ftgo-20260618T065744Z` reached PostHog collect and exposed a new collect-repair blocker. The first shard wrote evidence-backed `root-overview.md` during `collect_pair_repair` but then read `.test_durations` and stalled before `shard-pack-manifest.json`; the second shard exhausted pair repair with no authored output. The run was interrupted after two independent P0 collect failures to avoid spending the remaining medium window on repeated shard failures. Current slice narrows pair-repair evidence candidates, forbids extra repo reads after markdown write, and lets markdown-only pair repair partials chain into explicit manifest runtime recovery while no-artifact pair repair remains terminal.
- 2026-06-18: Committed second loop slice `6e84a9f`; diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T074654Z` confirmed the markdown-only manifest recovery path, but exposed a no-artifact collect-pair repair failure on the first PostHog shard. Raw stdout showed Codex created its own broad evidence-read command, read many root files plus `docker-compose.dev.yml`, and stalled before writing either `root-overview.md` or `shard-pack-manifest.json`. The run was interrupted as `infra_signal_terminated` after the P0 blocker was captured. The later exact read-only `collect_pair_repair_preflight` attempt also stalled preflight-only, so the current slice makes collect-pair repair one-shot write-first instead of two-phase preflight/write.
- 2026-06-18: After commit `28b5937`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T142635Z` proved the one-shot write-first prompt made Codex attempt a provider-authored filesystem command, but the provider added a brittle exact-phrase precondition (`required = [...]`, `missing expected evidence`) and aborted before writing any authored artifacts when current PostHog text did not match guessed snippets. Current slice keeps provider-authored recovery but forbids hard-coded expected-phrase gates before writes; repair may only precheck target containment, at least one allowed evidence file read, and size limits, while unsupported planned claims must be omitted or reported as coverage gaps.
- 2026-06-18: After commit `df771e5`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T145933Z` showed the root-file collect pair repair now writes valid artifacts, but the next directory shard (`posthog-bin`) failed before writing because Codex generated an over-complex Python writer with an invalid f-string (`f-string: empty expression not allowed`). Current slice keeps provider-authored recovery but tightens the first repair command to a mechanically simple writer: no Python f-strings, `.format(...)` template writers, generated Python source strings, nested quote tricks, or self-invented semantic pre-write aborts; backend validation remains the semantic gate.
- 2026-06-18: After commit `2edf6cf`, diagnostic `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260618T153313Z` confirmed the f-string failure was removed and the root-file shard still recovered, but `posthog-bin` exhausted collect pair repair with no artifacts because Codex added a file-size hard gate (`read file exceeds size limit`) after reading a candidate larger than the per-file budget. The run was interrupted once this acceptance blocker was captured. Current slice keeps the evidence-bounded contract but changes the repair instruction to read bounded prefixes or skip oversized candidates and continue; terminal no-write failure is allowed only when no allowed evidence yields bytes.
- 2026-06-19: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T074026Z` passed PostHog machine execution plus frontend smoke, but failed FTGO refresh `step2.asis_docs` as `runtime_contract_failed`: `draft_artifact_enrichment` rewrote the three markdown drafts, then rewrote `asis-draft-manifest.json.outputs[]` with an invalid `logical_path` alias. The same run still showed repeated collect manifest repair no-write/status-only behavior that runtime recovered as explicit telemetry. Current slice keeps strict parsing, forbids output aliases in draft contracts, tells enrichment to preserve existing `outputs[]` mappings, and changes manifest-only collect repair from read-only preflight to a write-first provider-authored command contract.
- 2026-06-19: After commit `9f2820d`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T172527Z` confirmed bootstrap-only draft validation now routes straight to `draft_artifact_enrichment`, but PostHog refresh accepted `posthog-bin/bin-overview.md` as a completed collect shard even though the markdown said the first bounded evidence read was interrupted, the artifact was initial-only, and it "will be repaired with concrete file-level evidence". The run was interrupted during frontend-triggered PostHog collect after this acceptance blocker was captured. Current slice makes interrupted temporary collect markdown contract-invalid regardless of semantic richness, adds live-pattern regression tests, and updates collect prompt/runbook wording so these artifacts fail before downstream `step2` can mask them.
- 2026-06-20: After commit `25bb2d1`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T064659Z` did not reach headless runtime: `npm ci --prefix ui` in UI precheck hung with no log progress while registry was reachable, requiring manual interrupt. Current slice bounds UI dependency/browser precheck via `ACP_LIVE_PRECHECK_UI_TIMEOUT_SEC`, materializes timeout as `precheck_failed`, and adds script/docs coverage so precheck hangs cannot silently block the live loop.
- 2026-06-20: After commit `b03fe11`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T071243Z` reached machine `PASS` for both `posthog` and `ftgo` with selected provider totals only and no legacy mixed quality artifacts. Manual artifact review rejected the evidence: FTGO `draft_artifact_enrichment` reported planned/failed shard counts as unknown despite current-run collect summaries showing 16/16 succeeded, and several FTGO markdown drafts mentioned PostHog/matrix context as if it were target evidence. Current slice exposes current-run typed shard-plan/summary JSON to focused draft enrichment, instructs providers to count shard-summary `items[].status`, makes target identity come from `repo_scope`/`repo_scopes`/`domain_id` rather than matrix folder names, and keeps live/manual release evidence names out of core AOR packages.
- 2026-06-20: After commit `438b1aa`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T114135Z` reached machine `PASS`, fixed current-run shard completeness and target contamination, and preserved selected-provider execution reporting. Manual artifact review still rejected the evidence because several operator-facing draft files described the enrichment as replacing placeholder content. Current slice makes generic placeholder-replacement narration a shared runtime draft validation failure and strengthens prompt/docs contracts so this remains `draft_artifact_enrichment_noop_or_scaffold`, not an accepted artifact-quality issue.
- 2026-06-20: After commit `703cb66`, strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T160443Z` reached non-release machine `PASS` for selected provider only (`backend 2/2`, frontend PASS for both profiles, no legacy mixed quality-gate artifacts). Manual artifact review rejected the result because current-run final markdown still contained generic failed/incomplete shard caveats despite `16/16` succeeded summaries, and refresh as-is markdown cited prior taskrun final/citation indexes as if they were current-run evidence. Current slice rejects foreign live `run_*` references and generic conditional shard-gap wording in shared draft validation and prompt/docs contracts.
- 2026-06-21: Committed `6c91550` to isolate `codex-code` preflight readiness with the same auth-only `CODEX_HOME`, disabled plugin/app suggestion surfaces and `--ignore-user-config`/`--ignore-rules` used by runtime. Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260621T174452Z` reached real headless execution for both PostHog and FTGO, preserved selected-provider totals, and failed both profiles as `runtime_contract_failed` because provider-authored `draft_artifact_enrichment` produced evidence-backed markdown with malformed inline-code/code-fence syntax. Current slice adds a one-shot provider-authored `draft_artifact_enrichment_markdown_syntax` retry and prompt/docs tests without deterministic artifact synthesis or validation weakening.
- 2026-06-22: Claude strict medium rerun `regres-long-posthog-ftgo-20260622T183346Z` ran from clean commit `694278f` with selected provider `claude-code` only. PostHog failed at `init.step0.constitution` because no-action enrichment rewrote `charter-overview.md` with runtime/provider/generated/final-index narration instead of pure repo-entrypoint constitution evidence; FTGO proved `git_url` repo roots and collect recovery by completing 16/16 collect shards, then failed at `init.step4.proposals` because focused enrichment/no-action retry did not mutate bootstrap `proposal.md`/`changelog.md` before the 90s pre-artifact wall-clock. Current slice separates step0 prompt/validation from downstream evidence, rejects step0 downstream/runtime-only leaks, makes step4 no-action retry bounded write-before-analysis, and raises focused draft enrichment to a minimum 3-minute pre-artifact window.
- 2026-06-22: Claude strict medium rerun `regres-long-posthog-ftgo-20260622T202736Z` from clean commit `86251c2` validated the step0/FTGO read-root fixes but failed both profiles in `init.step1.collect`: PostHog had directory-only repo evidence refs (`livestream`) and process-contaminated markdown (`posthog-share-staticfiles`), while FTGO reproduced process-contaminated markdown and one fully silent no-artifact collect repair path. Reports correctly preserved selected-provider totals and execution/artifact-quality separation, but collect-pair repair used the generic 90s focused repair window and did not give validation-specific immediate rewrite guidance. Current slice gives collect-pair repair a fresh-mutation threshold, minimum 3-minute pre/post/partial windows, and prompt focus for directory-only citations, process-contaminated markdown and no-artifact stalls.
- 2026-06-23: Claude strict medium rerun `regres-long-posthog-ftgo-20260622T224209Z` ran from clean commit `0a2cc01` with selected provider totals only and no legacy mixed quality-gate artifacts. PostHog completed collect but failed `refresh.step2.asis_docs` because evidence-backed enrichment still claimed current-run final/citation indexes were unavailable; FTGO completed collect best-effort only partially (`9/16` checkpointed, `7` collect-pair repair failures) after repeated no-artifact/directory/process-contamination recovery stalls. Current slice adds one provider-authored `draft_artifact_enrichment_downstream_index_retry` for stale downstream-index claims and switches live-observed no-artifact/directory/process collect pair repair to a compact field-contract prompt without deterministic artifact synthesis.
- 2026-06-23: Codex strict medium rerun `regres-long-posthog-ftgo-20260623T024519Z` from commit `70fbadc` reached non-release machine `PASS` for selected provider only (`2/2` backend, both frontend smoke runs passed, no legacy mixed quality artifacts). Manual artifact review still found proposal-quality defects: `proposal.md` self-reported stale non-zero final-index document counts (`77/55`) while current `final-run-index.json.canonical_documents[]` had `79/57`, and "high-signal" evidence bullets included metadata-only JSON lines such as `"version": 1`. Current slice rejects metadata-only evidence bullets and mismatched current-run final/citation index counts in strict runtime draft validation, and strengthens step4 enrichment prompts/docs to compute counts from parsed JSON variables or omit them.
- 2026-06-23: Claude strict medium rerun `regres-long-posthog-ftgo-20260623T135008Z` from clean commit `1475969` proved the marker/window fixes on PostHog backend (`init` and `refresh` succeeded with 16/16 collect shards), but frontend auto triggered a UI-side live run that failed at `init.step2.asis_docs`: focused enrichment left bootstrap markdown unchanged, then runtime incorrectly scheduled a generic `draft_artifact_enrichment_no_action_retry` before eventually failing as `draft_artifact_enrichment_noop_or_scaffold`. Current slice removes the generic no-action enrichment retry entirely; noop/scaffold enrichment now exhausts immediately as `runtime_contract_failed`, while the narrow `draft_artifact_enrichment_command_text_retry` remains only for providers that print shell/Python command text instead of mutating files.
- 2026-06-25: The latest interrupted `codex-code` strict medium diagnostic exposed an outer harness reliability defect: `pipeline_timeout_sec=14400` was not a hard boundary and refresh reached about 32700 seconds before manual interruption. Current slice adds a process-group `pipeline-watchdog.py`, timeout terminal status as `runtime_timeout`, deadline/last-progress/clock-gap reporting, and provider lifecycle progress fields without increasing canonical medium timeouts or changing product UI/API behavior.
- 2026-06-25: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260625T184127Z` confirmed the new watchdog/liveness path on PostHog backend, but frontend failed because the canonical PostHog path checkout became an unreadable Git repo (`workspace.repo.ref.invalid` for pinned SHA). Taskrun quality already recorded `runtime_write_audit_unexpected_mutation`, but runtime accepted it as warning-only. Current slice makes post-provider protected workspace/analyzed repo mutation a terminal `runtime_contract_failed` for otherwise-successful steps and makes the live backend cycle rewrite canonical `path` inputs to run-local isolated detached clones before `init-workspace`, keeping canonical `/tmp/provenarch-live-e2e/...` as a verified source prerequisite only.
- 2026-06-25: After commit `bca7093`, `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260625T205043Z` proved path-repo isolation (`target-repos.live-isolated.yaml` and clean canonical PostHog), but PostHog failed before headless runtime because baseline fake API init hit the fixed 120s `api_init_timeout_sec` while actively progressing through `init.step4.proposals`. Current slice adds bounded progress-aware grace to the API init poll loop without increasing canonical timeout presets.
- 2026-06-25: After commit `2b6c7a9`, `codex-code` strict medium rerun `regres-long-posthog-ftgo-20260625T211921Z` proved API init progress grace but failed during fake `init` before headless runtime: `runtime_write_audit_unexpected_mutation` reported `repo status unavailable after runtime` for the run-local read-only PostHog clone. The failure was a write-audit false positive caused by using writable/full `git status` against an isolated read-only large checkout. Current slice keeps read-only path-repo isolation but makes audit use a lightweight HEAD/index/mode snapshot with `GIT_OPTIONAL_LOCKS=0` for read-only clones, while preserving terminal mutation failures for protected workspace/writable repo changes.
## EP-20260608-medium-live-e2e-quality-ui

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Run host/tree/provider/path preflight for the medium live E2E profile.

### Context
Run a medium live E2E assessment on a trusted local host after a user-reported weak local result: sparse artifacts and missing C4 diagrams. This is an operator evaluation, not a product code slice. The run must follow `acp-e2e-live-gate` and `docs/RELEASE_LIVE_E2E_RUNBOOK.md`: use direct public harness commands, avoid wrapper scripts, avoid canonical matrix edits, and classify host/provider/path blockers separately from ACP product defects.

The medium non-release profile is `regres long`: `posthog` (`single-path`, medium shard bucket) plus `ftgo-application` (`single-git_url`, medium shard bucket), default qwen-only baseline with `RUN_COUNT=1`. Because the current worktree may need this plan update, execute the harness from a separate clean worktree.

### Goals (must have)
- [ ] Run host/tree/provider/path preflight for the medium live E2E profile.
- [ ] Launch `regres long` through `scripts/full-run-batch-matrix.sh` using the public planner/harness path.
- [ ] Keep frontend init inspection enabled so UI artifact readability can be assessed from real UI/API evidence.
- [ ] Inspect `run_matrix_*`, `execution_report_*`, `reports/taskruns/*-quality.json` telemetry, promoted reports, diagrams and raw taskrun metadata.
- [ ] Evaluate whether the resulting artifacts are substantive, evidence-backed and complete enough for architecture review.
- [ ] Evaluate whether the UI makes the generated artifacts discoverable and readable without relying only on filesystem inspection.

### Non-goals
- [ ] Do not change runtime contracts, schemas, canonical matrix files or curated repo files.
- [ ] Do not treat this non-release diagnostic as release readiness.
- [ ] Do not bypass provider/toolchain/path prechecks with test-only overrides.
- [ ] Do not add a new wrapper around the matrix harness.

### Approach
1) Record this ExecPlan and keep the live run itself in a separate clean worktree.
2) Run fail-fast host checks: writable roots, clean tree, exact Go/Node toolchain, provider command, canonical path repo availability for `posthog`.
3) Generate the medium command with `scripts/live-e2e-plan.py --mode regres --size long --providers qwen --frontend-mode per_run --format shell`.
4) Execute the printed `scripts/full-run-batch-matrix.sh` command and monitor profile status, driver logs, inventories and generated reports.
5) Inspect backend quality outputs and promoted workspace artifacts, including C4 diagram presence/content.
6) Inspect frontend live E2E outputs and, where possible, open the resulting UI workspace for manual readability checks.
7) Report the black-box assessment with evidence paths, failure class if any, and concrete gaps.

### Progress notes
- 2026-06-11: Re-ran medium diagnostics with extended activity windows on the selected live
  providers. One provider stopped at readiness, while another produced a substantive single-path
  `init` artifact set before a later `refresh` stopped on provider availability. Manual UI review
  found the root operator issue: Review defaulted to the latest failed partial run artifact set,
  hiding the complete successful init artifacts unless the operator switched runs from Analysis. The
  UI now keeps the failed-run blocker visible and offers a Review recovery action to open the latest
  successful artifacts; C4 preview also uses a scrollable canvas so large diagrams are not reduced to
  unreadable thumbnails.

### Files expected to change
- `docs/PLANS.md`
- Optional operator assessment under `reports/` if a durable report is useful after the run

### Acceptance criteria
- [ ] Preflight result is classified as passed or a concrete operational blocker.
- [ ] Medium harness command is recorded and launched only if preflight passes.
- [ ] Artifact quality is assessed from generated machine reports and direct artifact inspection.
- [ ] UI readability is assessed from frontend init evidence and manual UI/API inspection.
- [ ] Final answer names the primary failure class or confirms no blocking quality issue was found.

### Risks
- Medium live runs can take hours and can fail due to provider auth/quota/timeout rather than ACP defects.
- `posthog` path checkout may be missing or not pinned on the current host.
- Frontend inspection depends on successful backend snapshots; if backend fails, UI evidence may be dependent-skip rather than an independent UI result.

### Progress log
- 2026-06-08: Started medium live E2E artifact quality and UI readability assessment plan.
- 2026-06-08: Medium `regres long` diagnostic was mixed: `single-path/posthog` stopped as `operational_host_preflight_failed` because qwen headless probe timed out; `single-git_url/ftgo` backend and frontend hard-passed but produced operator-unusable artifacts. Root-cause triage points at enforced runtime prompt/quality policy, not C4 rendering: command-first collect/as-is/validator skeletons are valid with empty semantic model or placeholder drafts, qwen draft valid-artifact stop can accept those files, validator can return `PASS` with empty findings/questions for single-repo owner/dependency gaps, and live quality gates score/report this as warnings/analysis loss rather than hard failure. C4 diagrams are gap-only because promoted `final-run-index.json` has `semantic.entities=[]` and `semantic.edges=[]`.
- 2026-06-08: Added the remaining live-loop hardening surface: `/api/system/version` exposes actual running build metadata before workspace selection, Console V2 top bar no longer hard-codes `v0.1.1 beta`, and frontend live `init-inspect` now checks selected Review markdown and C4 Mermaid content for readability/placeholder/gap-only failures through `/api/artifacts`.
## EP-20260608-live-artifact-quality-hardening

Status: active — retained scope; reconcile current implementation before selecting a new slice.

Next action: Reconcile the recorded open criterion with current code/status before selecting work: Define and enforce an artifact-quality rubric across all analysis outputs: charter, collect shard manifests/docs, as-is reports, coverage/questions, findings, semantic model, C4 diagrams, proposals/changelog, taskrun indexes, citation indexes, and frontend review surfaces.

### Context
Follow-up to `EP-20260608-medium-live-e2e-quality-ui`. The medium live run showed that a provider can produce structurally valid but substantively unusable analysis artifacts: placeholder top-level reports, empty findings, empty semantic model, gap-only C4 diagrams, generic proposal/changelog files, and at least one taskrun/staging document sourced from a provider tool directory instead of repository evidence. The frontend review UI then presents these artifacts as reviewable/ready because it mostly checks artifact presence and open-question counts.

This is a product-quality slice. It must keep the existing local-first/entity-per-file contracts intact unless explicitly synchronized through schemas, docs, validators, fixtures and tests.

### Goals (must have)
- [ ] Define and enforce an artifact-quality rubric across all analysis outputs: charter, collect shard manifests/docs, as-is reports, coverage/questions, findings, semantic model, C4 diagrams, proposals/changelog, taskrun indexes, citation indexes, and frontend review surfaces.
- [ ] Audit prompt correctness for each artifact-producing step (`collect`, `as-is`, `findings/validator`, `proposals`) so prompts require evidence-backed, non-placeholder content before final acceptance.
- [ ] Make placeholder or skeleton-only promoted artifacts fail quality gates for nontrivial live targets instead of passing with a low signal score.
- [ ] Make an empty semantic model a hard quality failure when collect/as-is completed against a non-empty repository.
- [ ] Make gap-only C4 diagrams a hard quality failure when the semantic model is empty or unusable.
- [ ] Reject hidden/provider/tooling files such as `.qwen/`, `.claude/`, `.codex/`, `.git/`, and similar generated side-effects from shard document manifests and promoted canonical docs.
- [ ] Require findings/questions to reflect critical coverage gaps. For example, owner/dependency/operational gaps must not coexist with `No findings reported.` in normal report mode.
- [ ] Update frontend review/readiness so the UI surfaces artifact-quality blockers and keeps generated artifacts readable without raw runtime noise dominating the review surface.
- [ ] Replace the hard-coded UI `v0.1.1 beta` label with runtime/build metadata so live screenshots show the actual binary/UI version under test.
- [ ] Update live E2E scoring so backend and frontend hard-pass require quality-readable artifacts, not only completed runs and screenshot capture.
- [ ] Evaluate artifact quality from the actual artifacts produced by each run. Do not rely on a copied bad-run fixture as the release/live verdict source.

### Non-goals
- [ ] Do not tune the curated release matrices or repo fixtures to hide the observed qwen behavior.
- [ ] Do not create a persistent `ftgo` bad-output fixture as the main quality gate. The live gate must inspect the current run's artifact tree every time.
- [ ] Do not remove the gap-node diagram fallback; it remains useful diagnostic output, but it must not be counted as a successful C4 result.
- [ ] Do not require live network/provider calls in required CI.
- [ ] Do not introduce hosted-mode or security/compliance enforcement scope.

### Approach
1) Implement a per-run artifact inventory/evaluator that walks the actual promoted workspace and taskrun/staging outputs for the selected backend/frontend run. It must score every artifact surface produced by that run, including missing surfaces.
2) Harden artifact canonicalization and docflow validation so shard documents must be inside the allowed workspace artifact surface and must not come from hidden/provider/tool/runtime side-effect directories.
3) Extend backend run-quality evaluation to emit `artifact_quality:` warnings or blocking signals for sparse top-level reports, placeholder text (`Provider wrote...`), empty semantic entities/edges, gap-only diagrams, missing model files, empty findings with critical coverage gaps, and placeholder proposals/changelog.
4) Add prompt snapshot/contract tests that verify normal collect uses evidence-first bounded reads before writing artifacts, unchanged collect seeds/recovery fallbacks and scaffold-only manifest semantics are rejected before apply, draft first-action commands remain bootstrap-only until enriched, and final success is forbidden from generic placeholder text.
5) Tighten runtime prompt policy in `steppolicy`/`promptcontract`: normal collect must write evidence-backed doc+manifest pairs instead of seed-only first-action artifacts, draft first-action skeletons may bootstrap required files, and single-repo validator prompts must not return `PASS` with zero findings/questions when coverage gaps remain.
6) Extend `scripts/e2e_batch_report.py` and Python tests so dynamically inspected bad artifact shapes remain telemetry/manual-assessment evidence, while execution failures stay limited to host/preflight/runtime/frontend/infra classes.
7) Extend review API/UI with artifact-quality status per step and review surface. The UI should show hard blockers for empty model/gap-only C4/placeholders, distinguish baseline support files from analysis outputs, and keep artifact previews legible on desktop/mobile.
8) Update docs/runbook/testing strategy to document the quality rubric and rerun the medium live profile only after the dynamic evaluator and local tests pass.

### Files expected to change
- `internal/artifactquality/canonicalize.go`
- `internal/artifactquality/collect_bootstrap.go`
- `internal/contracts/docflow.go`
- `internal/orchestrator/docflow.go`
- `internal/orchestrator/quality.go`
- `internal/runtime/promptcontract/promptcontract.go`
- `internal/runtime/promptcontract/promptcontract_test.go`
- `internal/runtime/steppolicy/policy.go`
- `internal/runtime/steppolicy/policy_test.go`
- `internal/runtime/qwencode/runner.go`
- `internal/reports/compiler.go`
- `scripts/e2e_batch_report.py`
- `scripts/tests/batch_failure_classification_test.py`
- `scripts/tests/frontend_live_e2e_contract_test.py`
- `internal/api/review_diff.go`
- `ui/src/components/TopStatusBar.tsx`
- `ui/src/components/StagePanels.tsx`
- `docs/spec/PIPELINE_SPEC.md`
- `docs/TESTING_STRATEGY.md`
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
- Related unit/integration tests that construct temporary artifact trees or inspect generated run outputs

### Acceptance criteria
- [ ] Every live/backend/frontend run quality report includes a fresh artifact inventory and quality verdict for all produced and expected analysis surfaces.
- [ ] The observed sparse `ftgo` artifact shape fails when detected by the dynamic evaluator, without relying on a persisted copied fixture.
- [ ] Hidden/provider tool document paths in shard manifests are rejected before promotion and covered by unit tests.
- [ ] Backend quality JSON contains explicit `artifact_quality:` signals for every blocked artifact class found in the current run inventory.
- [ ] Prompt contract tests prove artifact-producing prompts require evidence-backed content, semantic entities/edges where applicable, and explicit non-placeholder final criteria.
- [ ] C4 gap-only output is visible as diagnostic output but is not counted as a successful C4 diagram in quality/readiness.
- [ ] Frontend review UI shows hard blockers for empty semantic model, gap-only diagrams, placeholder reports/proposals, and empty findings with critical coverage gaps.
- [ ] Frontend screenshots display version metadata from the running app/build, not a hard-coded release string.
- [ ] Frontend live E2E contract verifies not only screenshots/selectors but also artifact readability and quality blocker surfacing.
- [ ] Docs and runbook explain the rubric and how to inspect all resulting artifacts after a live run.
- [ ] Full DoD for the slice passes: `make contracts`, `make test`, `make lint`, `make build`.

### Risks
- Providers may need prompt changes that increase runtime duration; keep hard gates deterministic and let live gates validate provider-specific behavior.
- Empty findings can be valid for tiny deterministic or toy targets, so quality rules need an explicit nontrivial-target threshold instead of a blanket ban.
- Some generated support artifacts under `skills/` are expected baseline files. The quality rubric must separate baseline support surfaces from analysis outputs.

### Progress log
- 2026-06-08: Audited the failed medium live run artifact surfaces. Backend/frontend promoted workspaces have no `model/` files, C4 is gap-only, findings/proposals/changelog/top-level overviews are placeholders, shard manifests have citations but no entities/edges/findings, and one init shard staged a `.qwen/skills/.../SKILL.md` provider-side-effect file as an as-is document. UI review/readiness currently surfaces artifact presence but not these substantive quality failures. The top status bar also hard-codes `v0.1.1 beta`, so screenshots can mislead operators about the tested build version.
- 2026-06-08: Updated the plan to avoid making a persistent bad-run fixture the quality source of truth. The live gate must dynamically inspect and score the current run's generated artifacts every time; tests can use temporary in-test artifact trees only to protect evaluator behavior.
- 2026-06-08: Started implementation slice for dynamic backend artifact quality. Added fresh run artifact inventory to quality summary, blocking `artifact_quality:*` signals for sparse current-run surfaces, and collect manifest rejection for provider/tool side-effect document paths; no persistent bad-run fixture added.
- 2026-06-08: Completed the first backend/prompt hardening slice. Prompt policy now treats collect/as-is/proposals first-action skeletons as bootstrap-only, not final-acceptable content. Full local DoD passed with exact Node candidate: `make contracts`, `make test`, `make lint`, `make build`.
- 2026-06-08: Medium diagnostic on selected live providers reproduced product-quality blockers before
  full provider readiness: collect manifests had citations/docs but no semantic entities/edges/findings,
  C4 was gap-only, promoted reports/proposals contained placeholder text, and frontend inspection
  failed on gap-only C4 before Review/mobile readability could be assessed. Current fix slice makes
  collect skeleton/repair artifacts carry minimal evidence-backed semantic signal, removes
  provider-placeholder draft text from normal and focused repair paths, and makes backend-cycle fail
  fast on fresh `artifact_quality:*` warnings before dependent frontend.
- 2026-06-08: Follow-up fresh run proved the empty-model blocker was fixed, but operator quality was
  still not acceptable: semantic output was scaffold-like repo/shard containment with repeated
  owner-gap findings, and C4 context still showed gap-only context nodes. Added dynamic
  `artifact_quality.semantic_scaffold_only` and `artifact_quality.c4_context_scaffold_only` blockers
  from temporary in-test artifact trees, not from a persisted bad-run fixture.
- 2026-06-08/09: Several fresh diagnostics exposed collect bootstrap and repair-routing defects:
  unchanged first-action artifacts, bootstrap markdown routed to manifest-only repair, recovery-only
  low-signal docs, and early post-artifact termination before targeted evidence enrichment. The fixes
  add collect bootstrap detection, route bootstrap-only docs through pair repair, extend live adapter
  enrichment windows, and reject scaffold recovery prose in collect validation.
- 2026-06-09: Follow-up diagnostics narrowed the collect prompt issue: normal collect still depended
  on a seed heredoc and later enrichment that live execution did not reliably reach. The fix removes
  the normal seed heredoc helper, makes collect evidence-first, and rejects unchanged scaffold semantic
  skeletons even when markdown no longer contains seed prose.
- 2026-06-09: Manifest-only repair then proved too easy to accept syntactically while preserving
  wrapper-only semantic output. The fix makes manifest repair evidence-first, treats the embedded
  skeleton as schema guidance only, requires concrete named entities and non-container relationships,
  and keeps backend validation as the only success surface.
- 2026-06-09: Follow-up diagnostics showed evidence-first repair still lacked a reliable filesystem
  write surface and could stall after authored markdown already existed. The fix makes manifest repair
  command-first and adds deterministic `collect_manifest_runtime_recovery` that derives a manifest
  from provider-authored markdown under the existing write-set guard when provider repair stalls.
- 2026-06-09: Runtime recovery removed the hard contract failure but exposed inefficient ordering and
  scaffold-only recovered semantic. The fix runs deterministic recovery before provider manifest
  repair for missing/scaffold-only manifests with non-bootstrap authored docs, and strengthens recovered
  manifests with concrete service/component/datastore entities plus usage/dependency/configuration
  edges from authored markdown.
- 2026-06-09: Medium diagnostic with extended provider activity windows proved the timeout override
  slice works for long-running collect enrichment, while a later partial diagnostic showed preflight
  timeout changes did not control runtime activity windows. The fix adds diagnostic-only provider
  activity timeout env overrides that flow through matrix timeout env handling and remain blocked in
  release mode.
- 2026-06-09: Medium diagnostic with extended live activity windows proved artifacts can become
  substantive and frontend can complete, but a nontrivial target failed quality on
  `artifact_quality.c4_context_scaffold_only`: C4 context stayed gap-only despite a populated model.
  Operator screenshot review also found Review/Publish preview readability was under-tested. The fix
  adds C4 context internal-relation fallback plus viewport-visible Review/Publish preview assertions
  and layout hardening.
- 2026-06-10: Medium diagnostic with provider stall windows raised further proved old limits still cut
  valid work, but a traceable changelog hit a false `artifact_quality.placeholder_artifact` because it
  included a generic validation note. The fix narrows placeholder detection so bootstrap/generic
  proposal text remains blocked while traceable finding/question-backed changelog content passes.
- 2026-06-11: PR #110 merged into `main` at `e68d35c` after green PR checks, green post-merge `main`
  CI, and local `make contracts`, `make test`, `make lint`, `make build`. Started `v0.1.7` CI-only
  beta release metadata branch to publish the live artifact quality hardening in downloadable release
  artifacts. Canonical live release gate remains blocked on this host because Open edX/OpenStack
  canonical path checkouts under `/tmp/provenarch-live-e2e` are missing; no `RELEASE READY` claim is
  made without a fresh `release_verdict_*.json`.
- 2026-07-07: PR #120 merged into `main` at `7f65499` after green PR checks. Started `v0.1.8`
  beta release metadata branch for artifact-quality acceptance hardening, live E2E Excellent
  diagnostics and the recorded `smoke-tiny-bank-20260707T053308Z` Codex diagnostic evidence. The
  diagnostic reached strict `PASS` and artifact-quality `passed`, but stayed `Needs review` because
  `init.step2.asis_docs` and `init.step4.proposals` still required focused repair. The release can be
  prepared as beta/RC evidence, but must not claim canonical `RELEASE READY` without a fresh
  `release_verdict_*.json` plus accepted SWE UX/artifact-quality assessments.
- 2026-07-09: PR #123 squash-merged into `main` at `90c2931` after green PR checks. Started
  `v0.1.9` beta release metadata branch for Console V2 UX recovery polish, rendered recovery-state
  QA coverage and recorded medium `regres long` diagnostic evidence. The latest medium diagnostic
  remains a non-release `FAIL` because of provider/runtime reliability blockers (`qwen` headless
  readiness timeout and FTGO collect artifact handoff stalls), so this release must not claim
  canonical `RELEASE READY` without a fresh verified release verdict plus accepted SWE assessments.
## EP-20260508-oss-readiness-hardening

Status: review — recorded owner/admin or merge acceptance remains open.

Next action: Resolve the recorded open criterion: Owner/admin manual verification: GitHub still reports `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks` as disabled after API PATCH; enable them if available for the account/plan, or document the platform limitation.

### Context
Public OSS readiness audit found that the release binary path works, but repository governance, security reporting, CI trust boundaries, dependency hygiene, and README/community surfaces were below public distribution standards.

### Goals (must have)
- [x] Add security disclosure, support, governance, changelog, code of conduct, CODEOWNERS, PR template, and issue templates
- [x] Harden GitHub Actions permissions, timeouts, action pinning, release environment, SBOM/provenance, and required test alignment
- [x] Add Dependabot, CodeQL, Dependency Review, and Scorecard workflows/configuration
- [x] Fix UI dependency audit issues and enforce exact Node version from `.node-version`
- [x] Move required CI/release/local build toolchain to security-patched Go from `.go-version` while keeping `go.mod` compatibility at `go 1.20`
- [x] Update README/install/testing docs for OSS beta distribution and release evidence boundaries
- [x] Best-effort apply GitHub repository settings where current credentials have permission; otherwise record owner/admin manual tasks
- [ ] Owner/admin manual verification: GitHub still reports `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks` as disabled after API PATCH; enable them if available for the account/plan, or document the platform limitation
- [ ] Owner/admin manual verification: add a safe `creation` rule to the `v*` tag ruleset once bypass actors are confirmed, so release tag creation is restricted to maintainers without locking out the release owner

### Non-goals
- [x] Do not make Docker/npm/PyPI/Maven/crates.io a primary distribution path
- [x] Do not add hosted/SaaS mode or security/compliance enforcement
- [x] Do not run trusted live release matrix without explicit owner approval
- [x] Do not treat release readiness as passed without `reports/release_verdict_<matrix-id>.json`

### Approach
1) Add OSS community/security files and front-load README quickstart/release/security links.
2) Harden workflows with read-only defaults, job-level release writes, pinned actions, timeouts, and advisory security jobs.
3) Update UI dependencies and Node resolver so source builds are deterministic against `.node-version`.
4) Attempt admin-level GitHub settings changes through `gh`; record any settings that require owner/admin manual verification.
5) Run local DoD and targeted security/install checks.

### Files expected to change
- `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `SUPPORT.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`, `GOVERNANCE.md`
- `.github/*`
- `.goreleaser.yml`
- `scripts/resolve-node-tool.sh`, `scripts/tests/*`
- `ui/package.json`, `ui/package-lock.json`
- `docs/INSTALL.md`, `docs/TESTING_STRATEGY.md`, `docs/PLANS.md`

### Acceptance criteria
- [x] `make contracts`
- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] `npm audit --prefix ui --audit-level=moderate`
- [x] `govulncheck ./...` when available
- [x] `bash ./scripts/smoke-cli.sh`
- [x] `bash ./scripts/smoke-api.sh`
- [x] GitHub owner/admin-only settings are either applied or explicitly listed as manual residual tasks

### Risks
- Branch protection, rulesets, secret scanning, private vulnerability reporting, and environment reviewers may require owner/admin permissions.
- Exact Node enforcement can block local builds until Node `22.21.1` is installed or selected through `ACP_NODE_TOOL_CANDIDATES`.
- Exact Go enforcement can block local builds until Go from `.go-version` is installed or selected through `ACP_GO_BIN`.
- GoReleaser SBOM generation requires `syft` in the release workflow.
- Local Go 1.20.x remains useful for compatibility checks, but `govulncheck` on May 8, 2026 reports vulnerable standard-library call paths when scanning binaries built with Go 1.20.3.

### Progress log
- 2026-05-08: Started OSS readiness hardening slice from the public distribution audit plan. Live matrix was not run.
- 2026-05-08: `govulncheck` against local Go 1.20.3 reported standard-library vulnerabilities; CI/release toolchain was moved to Go 1.25.10 while preserving `go.mod` language compatibility.
- 2026-05-08: Second audit found local `make build` and smoke scripts could still use stale `go` from PATH; added `.go-version`, `scripts/run-go.sh`, and wired Makefile/smoke scripts to fail fast on mismatched Go toolchains.
- 2026-05-08: Follow-up `make test` correctly failed `TestActivePlansHaveOpenGoals` because this active plan had all goals checked. Reopened the only real residual owner/admin task for GitHub secret scanning non-provider/validity features.
- 2026-05-08: Added OSS community/security files, issue/PR templates, Dependabot, CodeQL, Dependency Review, Scorecard, pinned GitHub Actions, release environment, SBOM/provenance release hooks, UI dependency updates, and exact Node resolver enforcement.
- 2026-05-08: Applied GitHub settings with admin token: `main` branch protection, `v*` tag ruleset for deletion/non-fast-forward blocking, `github-release` environment reviewer, private vulnerability reporting, Dependabot alerts/security updates, secret scanning, push protection, repo description/topics, labels, milestones, and curated `v0.1.0` release notes. GitHub still reports `secret_scanning_non_provider_patterns` and `secret_scanning_validity_checks` as disabled after PATCH; owners should manually verify whether these features are available for the account/plan.
- 2026-05-08: Tag ruleset creation restriction was left as owner/admin verification instead of being enabled blindly. GitHub's ruleset model restricts creations to bypass actors, and the safe bypass role/team set for this personal repository should be confirmed before enforcement.
- 2026-05-08: Verification passed with Go 1.25.10 + Node 22.21.1: `make contracts`, `make test`, `make lint`, `make build`, `npm audit --prefix ui --audit-level=moderate`, `govulncheck ./...`, `bash ./scripts/smoke-cli.sh`, `bash ./scripts/smoke-api.sh`, `git diff --check`, YAML parse smoke, and `goreleaser check`.
- 2026-05-08: Final workflow audit found two release-readiness bugs: `dependency-review` was pinned to a SHA from the wrong action repository, and Go workflows duplicated the exact version instead of reading `.go-version`. Fixed both and added regression coverage.
- 2026-05-08: Second-pass verification passed after Go guard and issue-template fixes: `make contracts`, `make test`, `make lint`, `make build`, `npm audit --prefix ui --audit-level=moderate`, `govulncheck ./...`, `bash ./scripts/smoke-cli.sh`, `bash ./scripts/smoke-api.sh`, `git diff --check`, YAML parse smoke, `goreleaser check`, and targeted Go/Node resolver tests.
- 2026-05-08: Final release metadata audit found `v0.1.0` was published as a normal GitHub Release while docs call the line MVP beta/pre-release. Changed GoReleaser to keep future beta releases prerelease and fixed `install.sh` so `ACP_VERSION=latest` resolves prerelease releases through the GitHub Releases API instead of relying on `/releases/latest/download`.
- 2026-05-08: Tried marking the already-published `v0.1.0` as prerelease, but reverted the remote release to latest/non-prerelease because the current public `main/install.sh` still uses GitHub's `/releases/latest/download` endpoint. Public install smoke passed again after restoring `v0.1.0` as latest. After this installer fix is merged to `main`, owners can mark beta releases as GitHub prereleases without breaking the default install command.
- 2026-05-08: PR mergeability audit found a solo-maintainer governance deadlock: `main` required one approval plus CODEOWNERS review, but `GrinRus` is currently the only collaborator and CODEOWNER. Documented the solo-maintainer policy: required CI remains mandatory, while required approving/CODEOWNERS reviews are enabled after a second maintainer exists.
## EP-20260507-trusted-live-validation

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Prepare a clean committed trusted-host worktree with required provider binaries/auth and pinned repo/path prerequisites.

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

### Non-live prerequisite check (2026-05-07)

| Check | Status | Evidence / note |
|---|---|---|
| Clean committed local worktree | ready | `git status --short` was empty at `c1f58b5`. |
| Required provider binaries in PATH | ready for auth check | `command -v qwen`, `command -v claude`, and `command -v codex` returned local paths. Exact machine-local paths are intentionally omitted from the versioned plan; provider auth/quota was not probed. |
| Canonical harness inputs present | ready | `examples/e2e-matrix.regres-fast.bank-openedx.yaml`, `examples/e2e-matrix.regres-fast.openstack.yaml`, `scripts/full-run-batch-matrix.sh`, and `scripts/verify-release-verdict.py` exist. |
| Live execution permission | blocked | No live matrix was started in this local implementation pass. Running live validation still needs explicit trusted-machine approval and direct `scripts/full-run-batch-matrix.sh` invocation. |

### Critical analysis (2026-05-08)

This slice is not closed and must not be treated as release evidence. The local worktree check and provider binary discovery only prove that the current machine can find command names; they do not prove provider authentication, quota availability, pinned repo/path readiness, matrix pass/fail status, or release readiness.

Residual blockers:
- live matrix execution still requires explicit trusted-machine approval;
- provider auth/quota remains unverified;
- pass criteria remain completely open until direct `scripts/full-run-batch-matrix.sh` runs produce matrix/report artifacts;
- release readiness still requires `reports/release_verdict_<matrix-id>.json` plus `scripts/verify-release-verdict.py` when a release verdict is used.

### Files expected to change
- `docs/PLANS.md`
- `docs/archive/*` for execution reports or reconciliation notes
- focused code/docs/tests only if live evidence creates a new fix slice

### Acceptance criteria
- [ ] No unresolved `runtime_contract_failed`, `runner_unavailable`, `semantic_hard_fail`, independent frontend failure, stale `running`, or unexpected `analysis:cross-repo-missing` remains for fixed scenarios
- [ ] Every live run has a recorded `matrix_id`, `driver.log`, `profile_matrix_<matrix-id>.*`, quality reports, and taskrun diagnostics
- [x] Release readiness uses only `reports/release_verdict_<matrix-id>.json`; verifier exits 0 only for `PASS` / `RELEASE READY` / `passed`
- [ ] Any live failure produces a narrower follow-up fix slice instead of ad-hoc matrix edits

### Risks
- Provider outages, auth/quota limits, or missing trusted-host path prerequisites can block live gates without indicating product regression.
- Full fresh-all-6 can be expensive; run it only after smaller gates either pass or isolate residual blockers.

### Progress log
- 2026-05-07: Created by tracker reconciliation. Consolidates live rerun gates from historical active plans; no live matrix was run in this reconciliation slice.
- 2026-05-07: Performed non-live prerequisite check only. Worktree and provider binary paths look ready for an owner-approved trusted-machine run, but provider auth/quota and pinned repo/path prerequisites are unverified. Live matrix execution remains blocked until explicit approval.
## EP-20260618-live-e2e-quality-loop

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Produce clean strict medium `codex-code` evidence.

### Context
Medium live E2E validation now uses the canonical non-release `regres long` matrix (`posthog + ftgo`, `RUN_COUNT=1`) with one selected provider at a time. The current loop starts with `codex-code`, then `claude-code`, and repeats the affected provider until execution quality, artifact completeness, and UX/manual evidence are clean. Generated live evidence is diagnostic and must not be committed.

### Goals (must have)
- [x] Keep product UI/API behavior, canonical matrix files, and curated repo files unchanged
- [x] Preserve execution/UX/artifact quality separation; `artifact_quality.*` is a public black-box artifact gate, separate from runtime contract status
- [x] Fix live-observed draft enrichment no-op/scaffold failure without ACP-side deterministic product synthesis
- [x] Fix frontend snapshot eligibility so failed headless refresh rows cannot start heavy UI smoke as if they were valid snapshots
- [x] Fix live-observed collect pair repair preflight-only/status-only no-op so post-preflight provider output must mutate markdown + manifest before prose
- [x] Replace two-phase collect pair repair preflight/write contract with one-shot provider-authored write-first command
- [x] Tighten normal collect so its first bounded work unit reads limited evidence, writes doc + manifest directly, and does not rely on repair/recovery as the normal success path
- [x] Isolate default `codex-code` headless invocation from user Codex config/rules so local MCP/plugins cannot stall artifact-only tasks before model actions
- [ ] Produce clean strict medium `codex-code` evidence
- [ ] Produce clean strict medium `claude-code` evidence
- [ ] Record execution, artifact completeness/truthfulness, and UX findings for each rerun

### Non-goals
- [x] Do not add wrapper scripts around `scripts/full-run-batch-matrix.sh`
- [x] Do not bypass provider/path preflight with test-only env flags
- [x] Do not edit `examples/e2e-matrix.regres-long.yaml` or curated repo files to fit the current machine
- [x] Do not treat SWE manual UX/artifact reports as core AOR runtime inputs

### Approach
1) Implement the smallest runtime/reporting/docs/test fix slice for the current live blocker.
2) Run `make contracts`, `make test`, `make lint`, and `make build`.
3) Commit the fix before any live rerun.
4) Run direct `scripts/full-run-batch-matrix.sh` from a clean tree using planner-generated `regres long` provider env.
5) Inspect matrix/profile/run/execution/frontend reports, taskrun quality JSON, raw provider metadata, staged manifests/docs, final/citation indexes, and screenshots/logs.
6) If any P0/P1 execution bug, artifact incompleteness/misleading output, or UX blocker remains, open the next narrow fix slice and repeat.

### Acceptance criteria
- [ ] Latest strict medium `codex-code` run passes selected-provider backend totals with no synthetic provider deficits
- [ ] Latest strict medium `claude-code` run passes selected-provider backend totals with no synthetic provider deficits
- [ ] No unresolved `runtime_contract_failed`, `runner_unavailable`, `runtime_timeout`, `runtime_flow_failed`, or `frontend_failed` remains for the accepted medium evidence
- [ ] No old mixed quality gate artifacts (`quality_report_*`, `quality_gates_failed`, `failure_reason=quality`, `RUN_QUALITY_GATES`) appear in generated evidence
- [ ] Artifact completeness includes planned/succeeded/failed shard accounting, final/citation indexes, proposals, publish readiness, and decision-ready summaries
- [ ] UX evidence is accepted when frontend runs, or explicitly inconclusive/not applicable with no hidden blocker when frontend is skipped

### Progress log
- 2026-06-18: Codex strict medium run `regres-long-posthog-ftgo-20260618T083335Z` failed in `refresh.step2.asis_docs`: `draft_artifact_enrichment` printed analysis-only status and left bootstrap markdown unchanged. Fixed write-first draft enrichment prompt/contract, reporting tests, frontend snapshot eligibility, docs, and committed `7c1df8e`.
- 2026-06-18: Codex rerun `regres-long-posthog-ftgo-20260618T131738Z` proved the draft enrichment fix for `init.step0.constitution`, then exposed a collect pair repair no-op: `collect_pair_repair_preflight` returned bounded evidence, provider printed “I am now writing”/status prose, but did not execute the required filesystem write before stall. The known-bad diagnostic was stopped early to implement the next narrow fix.
- 2026-06-18: Tightened collect pair repair prompt/contract so post-preflight plan/status/analysis-only prose is invalid and the next item must be a filesystem command that writes the evidence-backed markdown plus `shard-pack-manifest.json`. Targeted prompt/adapter tests passed; full DoD passed with pinned Node 22.21.1 (`make contracts`, `make test`, `make lint`, `make build`).
- 2026-06-18: Codex rerun `regres-long-posthog-ftgo-20260618T135110Z` showed the previous fix was still insufficient: `posthog-bin` `collect_pair_repair` executed only the read-only preflight command, emitted no follow-up command or agent message, and stalled before artifacts. The run was stopped as known-bad; next fix removes the separate pair-repair preflight and requires a one-shot write-first provider-authored command.
- 2026-06-18: Replaced collect pair repair's two-phase preflight/write prompt with one-shot write-first provider-authored repair. The prompt now forbids separate `collect_pair_repair_preflight`, requires the first filesystem command to read bounded evidence and write both final artifacts before returning, and keeps manifest-only repair as the only read-only preflight path. Full DoD passed with pinned Node 22.21.1 (`make contracts`, `make test`, `make lint`, `make build`).
- 2026-06-18: Codex rerun `regres-long-posthog-ftgo-20260618T161004Z` proved init could complete only with heavy repair/recovery reliance (`repair_attempts=19`, `pre_artifact_stalls=13`, `post_artifact_stalls=25`), and refresh immediately repeated a normal collect `runtime_stalled_before_artifacts` before `collect_pair_repair`. The run was interrupted as known-bad diagnostic evidence. Current slice tightens normal collect prompt/contract so the first filesystem action must read bounded evidence and write the authored doc + `shard-pack-manifest.json` together, with no read-only preflight, analysis-only prose, broad sweep, fragile templated writer, or reliance on focused repair as the expected path.
- 2026-06-18: After commit `35eb476`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T181545Z` showed normal collect improved but still not clean: PostHog collect advanced through multiple shards, while `posthog-cli-common`, `posthog-frontend-funnel-udf`, and `posthog-nodejs-packages` each failed the normal first write because Codex generated brittle inline Ruby/Node writer scripts that crashed before artifacts (`NoMethodError`, invalid regexp flags, template literal syntax). `collect_pair_repair` recovered the first two and was running on the third when the known-bad run was stopped. Current slice changes normal collect from a single generated read+write script requirement to a bounded read/list followed by direct literal shell heredoc/printf/tee writes, forbidding Ruby/Node/Python/Perl/awk/jq inline writers before both targets exist and requiring immediate simpler literal retry if the direct write fails.
- 2026-06-18: After commit `60449b2`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T190906Z` reached the first PostHog collect shard but emitted only local Codex plugin/MCP/state diagnostics (`MCP tools/list`, stream 504, plugin manifest/rules noise) and no model command items before the runtime scheduled `collect_pair_repair`. The run was stopped as known-bad because this is not prompt content failure; it is headless Codex environment leakage. Commit `1862ba7` added `--ignore-user-config` and `--ignore-rules`, but the next rerun proved those flags only ignore `$CODEX_HOME/config.toml` and user/project rules, not plugin/skill loading from `$CODEX_HOME`.
- 2026-06-18: Codex strict medium rerun `regres-long-posthog-ftgo-20260618T193336Z` started with `--ignore-user-config --ignore-rules` but still loaded `/Users/griogrii_riabov/.codex/.tmp/plugins/...` and `codex_core_skills` before any model action; the first PostHog collect shard wrote no artifacts for 108s and went to `collect_pair_repair`. The run was stopped as known-bad. Current slice adds shared `CommandSpec.Env` support and makes default `codex-code` run under isolated auth-only `CODEX_HOME` copied from caller `auth.json`/`installation_id`, excluding `config.toml`, MCP/plugins, skills, `.tmp/plugins`, and rules; custom runner args remain caller-owned overrides.
- 2026-06-18: After commit `4a3d95f`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T195848Z` proved `CODEX_HOME` override was applied, but Codex CLI self-created `config.toml`, `.tmp/plugins`, plugin cache, system skills and sqlite state inside that isolated home; `ngs-analysis` plugin manifest warnings still appeared from the isolated path. The run was stopped as known-bad before collect acceptance because isolated auth home alone does not disable remote plugin/app sync. Current slice keeps the isolated home and adds default `codex exec --disable` flags for plugin/app suggestion surfaces (`plugins`, `remote_plugin`, `plugin_sharing`, `apps`, `enable_mcp_apps`, `tool_suggest`, `skill_mcp_dependency_install`), verified manually on a short `codex exec` prompt to avoid `.tmp/plugins` creation.
- 2026-06-18: After commit `6cbffc8`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T201715Z` proved plugin/app surfaces stayed disabled (`plugin=0`, `skill=0`, no MCP startup), but the first collect shard produced no model/action events before the shared 75s pre-artifact window and was prematurely routed into `collect_pair_repair`. The run was stopped as known-bad. Current slice keeps strict stall semantics but sets a Codex-specific 3-minute initial pre-artifact window for collect steps, matching observed medium-slice prompt latency after removing plugin/app stderr noise.
- 2026-06-18: After commit `6929237`, Codex strict medium rerun `regres-long-posthog-ftgo-20260618T203446Z` proved backend collect no longer prematurely repaired and `step2/step4` enrichment replaced bootstrap scaffold, but frontend `init-inspect` hung after runtime success because the Activity / Events drawer was closed while the live spec tried `run-logs-mode-select.selectOption("events")`; Playwright had no bounded action timeout and would have waited for the full raised init budget. Current slice opens the drawer before log mode actions and bounds live Playwright action timeouts so this class is reported as `frontend_failed` instead of a multi-hour hang.
- 2026-06-19: Codex strict medium rerun `regres-long-posthog-ftgo-20260619T211450Z` ran from clean commit `476a109`. PostHog init succeeded and refresh collect reached 16/16 shard manifests, proving the stale missing-evidence markdown repair fix for `posthog-share-staticfiles`. The profile then failed at `refresh.step4.proposals`: `draft_artifact_enrichment` rewrote `proposal.md`/`changelog.md` but copied the bootstrap draft manifest summary string `"Drafted required runtime artifacts for this step."` into final markdown, so shared draft validation correctly rejected it as `draft_artifact_enrichment_noop_or_scaffold` / `runtime_contract_failed`. The same run exposed artifact-quality drift in refresh `step2` summary: shard completeness was reported as lexical `planned=0, succeeded=0, failed/error=2` despite observed 16/16 manifests. Current slice bans copied bootstrap manifest summaries in enrichment markdown and requires typed/observed shard completeness instead of lexical failed/error counts.
- 2026-06-20: Codex strict medium rerun `regres-long-posthog-ftgo-20260619T224351Z` ran from clean commit `1bc3bc5` with selected provider `codex-code` only. Backend execution passed `2/2` and preserved selected-provider totals/no legacy mixed quality-gate artifacts, but the matrix failed because PostHog frontend smoke launched a UI-side run `run_20260619_235435_001` that failed as `run_partial_failed` at `init.step4.proposals` after shard `posthog-frontend-funnel-udf` exhausted `collect_pair_repair` with only a thread-start/no-output artifact. FTGO frontend passed, but manual artifact review rejected final markdown that narrated runtime recovery mechanics (`current draft manifest`, `manifest target remains`, `bounded staged evidence`, `enrichment read`) instead of operator-facing architecture/proposal conclusions. Current slice rejects recovery-process narration in shared draft validation/prompt contracts and adds frontend report `runtime_details` so UI-run failures surface `run_id`, error code, current step, screenshots, and Playwright result directory without digging through raw metadata.
- 2026-06-21: Codex strict medium rerun `regres-long-posthog-ftgo-20260621T004731Z` ran from clean commit `df46cda` with selected provider `codex-code` only. PostHog init succeeded and refresh collect completed `16/16` without partial failures, but `refresh.step4.proposals` failed as `runtime_contract_failed`: enrichment wrote exact all-succeeded counts, but shared draft validation over-matched the positive line `No failed or incomplete shards remain as a coverage blocker in the current-run typed shard summary` as generic shard-gap wording. Current slice narrows validation so exact negated current-run no-blocker statements pass, while conditional/rerun shard caveats remain rejected, and strengthens step4 prompts to require exact counts plus no-shard-coverage-blocker wording in both proposal and changelog.
- 2026-06-22: Codex strict medium rerun `regres-long-posthog-ftgo-20260622T111245Z` ran from clean commit `8bb1517` with selected provider `codex-code` only. PostHog init succeeded, PostHog refresh and FTGO init both reached `step4.proposals`, then failed as `runtime_contract_failed` / `draft_artifact_enrichment_noop_or_scaffold`: focused enrichment either started no filesystem command or produced diagnostics without fresh valid `proposal.md`/`changelog.md` rewrites, leaving bootstrap proposal scaffold in place. Current slice adds one compact provider-authored `draft_artifact_enrichment_no_action_retry` that requires a single `python3` filesystem command over bounded current-run evidence; repeated noop/scaffold output remains terminal and no deterministic ACP-side proposal synthesis is added.
## EP-20260507-cleanup-owner-decisions

Status: review — recorded owner/admin or merge acceptance remains open.

Next action: Resolve the recorded open criterion: Owner decision: retain/move `internal/docsync` and `internal/scriptsmeta`.

### Context
Safe cleanup is already complete, but `docs/PLANS.md` and `docs/BACKLOG.md` retain owner-gated cleanup decisions where deleting or moving tracked surfaces could damage discoverability, review UX, or hidden tooling usage. These items must remain explicit and blocked until owner intent is known.

### Goals (must have)
- [x] Gather usage evidence for whether `internal/docsync` and `internal/scriptsmeta` should remain test-only packages under `internal/*`
- [x] Gather evidence on retaining both `docs/archive/PLANS_ARCHIVE_2026-04.md` and `docs/archive/PLANS_SNAPSHOT_2026-04-21.md`
- [x] Gather evidence on persisted `fixtures/scenarios/*/golden/readable`
- [x] Gather evidence on duplicated readable scenario fixtures
- [x] Gather evidence on `docs/BACKLOG.md` role as active planning surface vs reference/history backlog
- [ ] Owner decision: retain/move `internal/docsync` and `internal/scriptsmeta`
- [ ] Owner decision: retain/remove/dedupe April plan archive + snapshot docs
- [ ] Owner decision: retain/remove/dedupe readable golden fixtures
- [ ] Owner decision: keep `docs/BACKLOG.md` as reference/acceptance backlog or change its planning-surface role
- [ ] If approved by owners, implement each cleanup as a separate small change set with tests/docs sync

### Non-goals
- [x] Do not move `internal/docsync` or `internal/scriptsmeta` without owner approval
- [x] Do not delete archive snapshots, readable fixtures, or duplicated fixtures without owner approval
- [x] Do not mix destructive cleanup with runtime/live fixes

### Approach
1) Search usage and references for each cleanup candidate.
2) Write a short evidence table with retain/remove/dedupe options and recommended default.
3) Request owner decisions; retain by default if no answer is available.
4) Implement approved cleanup items one at a time with focused tests and full DoD when tracked files change.

### Owner-decision evidence (2026-05-07)

| Candidate | Evidence gathered | Options for owner | Default until decision |
|---|---|---|---|
| `internal/docsync`, `internal/scriptsmeta` | Only test files are present under those packages: `internal/docsync/docsync_test.go`, `internal/scriptsmeta/resolve_repos_meta_test.go`. `go test ./...` runs them as repo-level consistency gates; `docs/TESTING_STRATEGY.md` names `internal/docsync` as a docs-consistency gate. | Retain under `internal/*`; move to a dedicated test-support location; collapse into script tests. | Retain under `internal/*`; moving them is layout churn without owner-confirmed value. |
| `docs/archive/PLANS_ARCHIVE_2026-04.md` + `docs/archive/PLANS_SNAPSHOT_2026-04-21.md` | `docs/PLANS.md` lists both archives; the snapshot records that completed historical plans moved to the April archive. Current sizes are 912 and 1176 lines, respectively, so they are substantial historical evidence. | Retain both; collapse snapshot into archive; keep snapshot but remove cross-links. | Retain both until owner decides the audit/history surface can be reduced. |
| `fixtures/scenarios/*/golden/readable/*` | 90 tracked readable files exist. `docs/BASELINE_POLICY.md`, `docs/TESTING_STRATEGY.md`, `fixtures/README.md`, and `internal/docsync/docsync_test.go` explicitly describe them as tracked baseline/release surface and human-readable deterministic export. | Retain tracked readable exports; remove and rely on machine-readable golden only; generate on demand without tracking. | Retain; current docs/tests make them intentional, not accidental generated output. |
| Duplicated readable scenario fixtures | Hash scan shows many identical files repeated across the three readable scenario exports, including `reports/as-is/*`, `reports/diagrams/*`, `model/entities/svc.payments.yaml`, and proposal docs. | Keep duplicated per-scenario snapshots; dedupe via shared fixture layer; remove readable exports after replacing review-diff workflow. | Retain duplicated snapshots; dedupe needs a QA/tooling owner decision because per-scenario full-tree diffs are the current review UX. |
| `docs/BACKLOG.md` role | README still points to `docs/BACKLOG.md` for epics/acceptance criteria, AGENTS instructs agents to take reviewable slices from it, and `internal/docsync/docsync_test.go` asserts several backlog truth-sync strings. The backlog itself says active engineering slices live in `docs/STAKEHOLDER_DOC.md` and `docs/PLANS.md`, while a 2026-04-22 follow-up asks owners to decide active planning surface vs reference/history role. | Keep as reference/acceptance backlog; promote it back to active planning surface; archive/freeze it after migrating acceptance criteria elsewhere. | Keep current reference/acceptance role until owners decide; active execution remains in `docs/PLANS.md`. |

### Critical analysis (2026-05-08)

This slice is evidence-complete but decision-blocked. The gathered evidence points toward retaining all cleanup candidates by default because the candidates are referenced by tests/docs or are intentionally tracked review surfaces. However, retaining by default is not the same as resolving ownership: no package move, archive collapse, fixture deletion, fixture dedupe, or runbook consolidation should happen until maintainers/docs/tooling owners choose an option.

Residual blockers:
- no owner decision has been provided for test-only package placement;
- no owner decision has been provided for archive/snapshot retention;
- no owner decision has been provided for readable fixture retention or dedupe;
- no owner decision has been provided for the long-term role of `docs/BACKLOG.md`;
- no destructive cleanup was performed, so the final implementation goal remains intentionally open.

### Files expected to change
- `docs/PLANS.md`
- `docs/BACKLOG.md` only if owner-gated backlog wording changes
- candidate files only after explicit owner approval

### Acceptance criteria
- [x] Each owner-gated item has usage evidence and an explicit retain/remove/dedupe recommendation
- [x] No destructive cleanup happens without owner approval
- [ ] Approved cleanup items are implemented in separate commits
- [x] Active plan surface keeps owner-gated decisions visible until resolved

### Risks
- Removing historical docs or readable fixtures can degrade review workflows even if tests do not fail.
- Keeping everything can preserve drift; retained items need explicit ownership or periodic review.

### Progress log
- 2026-05-07: Created by tracker reconciliation from `EP-20260421-cleanup-owner-followups` and `docs/BACKLOG.md` cleanup follow-ups.
- 2026-05-07: Gathered usage evidence and documented retain-by-default recommendations. No files were deleted, moved, or deduplicated; owner decisions remain open and blocked.
- 2026-05-18: Removed the live E2E convenience-runbook owner decision from this cleanup plan because the black-box live E2E slice deletes old live E2E surfaces without compatibility.

---

## EP-20260704-needs-review-to-excellent-no-repair-pressure

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Remove the remaining normal first-pass scaffold/finding-linkage repair path for `step2.asis_docs` and `step4.proposals` before archiving this plan as `Excellent`-ready.

### Context
The latest `smoke tiny` diagnostic (`smoke-tiny-bank-runtime-pass-20260704T121128Z`) reached strict machine `PASS` with `runtime_contract_status=passed` and `artifact_quality_status=passed`, but live E2E correctly capped the label at `Needs review`. The remaining blockers are runtime-quality signals, not artifact-quality failures: first-pass `step2.asis_docs` and `step4.proposals` drafts still need focused repair, `step0.constitution` bootstrap text leaks downstream/runtime-only wording, and valid post-artifact controlled stops are over-attributed as stall pressure for `step1.collect` and `step3.findings`.

### Goals (must have)
- [x] Make normal `step0.constitution`, `step2.asis_docs`, and `step4.proposals` prompts require same-turn completion, with no final/analysis-only response before evidence-backed draft markdown exists.
- [x] Remove downstream/runtime-only wording from the initial `charter-overview.md` scaffold while preserving strict validation that blocks unchanged bootstrap drafts.
- [x] Make normal `step2.asis_docs` prompts enumerate current-run typed shard/index evidence and require exact `planned/succeeded/failed/incomplete` counts when available.
- [x] Make normal `step4.proposals` prompts require exact nested staged findings file reads and exact finding-ID linkage/actionability in the first-pass proposal/changelog.
- [x] Make normal and focused `step4.proposals` guidance require both `proposal.md` and `runtime-proposals.md`/`changelog.md` to contain the exact literal `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` string plus an explicit `no-shard-coverage-blocker` statement when typed shard status is visible.
- [x] Treat valid-artifact controlled stop as diagnostic telemetry, not `runtime_quality.stall_pressure`; keep invalid/no-fresh/pre-artifact/repair stalls as real Excellent blockers.
- [x] Add `first_validation_error_excerpt`, `stop_kind`, and `initial_artifact_state` to live `excellent_blockers_by_step` without schema-breaking removals.
- [x] Update tests, fixtures, docs/spec/runbook and run full DoD.
- [x] Rerun diagnostic `smoke tiny` on `codex-code` after the final validation/reporting fixes and classify strict/artifact/runtime pressure status.
- [x] Route recoverable draft semantic/shape repair failures, including step4 missing proposal sections and missing proposal shard completeness, into provider-authored `draft_artifact_enrichment` instead of terminal scaffold repair failure.
- [x] Harden normal `step2.asis_docs` / `step4.proposals` prompts after live evidence so first-pass draft markdown is written before the manifest and `step4` includes validator-required proposal/changelog sections by name.
- [ ] Remove the remaining normal first-pass scaffold/finding-linkage repair path for `step2.asis_docs` and `step4.proposals` before archiving this plan as `Excellent`-ready.
- [ ] Rerun diagnostic `smoke tiny` on `codex-code` after the 2026-07-05 manifest-last/required-sections prompt hardening; continue only if the result is `strict PASS`, `artifact_quality_status=passed`, no actual `runtime_quality.repair_heavy`, no actual `runtime_quality.stall_pressure`, and verdict `Excellent`.
- [ ] Fix or explicitly route the live-observed `init.step1.collect` no-artifact pre-artifact stall on root-file shards before rerunning Phase A, because partial collect prevents `step2`/`step4` validation and blocks `Excellent`.
- [x] Harden collect manifest normal/repair prompts for structural self-checks: `semantic.questions[*].id/text`, unique `citations[].id`, non-empty citation `claim_ids`/`document_ids`, `documents[].citation_ids` references and file-level repo/provenance paths.
- [x] Add one provider-authored `collect_manifest_shape_cleanup` retry after failed manifest-only repair for clean manifest-shape errors such as missing question text, duplicate citation IDs and citation/document binding drift; repeated invalid output remains `runtime_contract_failed`.
- [x] Add `terminal_validation_error_excerpt` to live `excellent_blockers_by_step` while preserving `first_validation_error_excerpt`, so repair exhaustion reports show the final terminal cause.
- [x] Rerun diagnostic `smoke tiny` on `codex-code` after the collect manifest shape-cleanup slice and classify strict/artifact/runtime pressure status.
- [x] Remove the `init.step2.asis_docs` first-pass missing-manifest/exact-shard-completeness/operator-summary repair path: normal Codex now writes all four step2 targets and passes strict draft validation on init and refresh.
- [x] Harden normal `step2.asis_docs` first-pass prompt with a compact same-command write sequence: exact `write_root`/`draft_root`, all three markdown targets, `asis-draft-manifest.json` last, and `test -s` checks for all four targets.
- [x] Require normal `step2.asis_docs` prompt guidance to place exact typed shard completeness and `no-shard-coverage-blocker` in both `summary.md` and `architect-summary.md` when all current-run shards succeeded.
- [x] Reject shell-substituted `step2.asis_docs` markdown that contains empty evidence reference slots such as `from  and`, `checked:  and`, `under .`, or `Use  and`.
- [x] Reject `step4.proposals` actionable bullets with placeholder finding IDs such as `Finding ID: none` when current-run findings are non-empty, even if an exact finding ID appears elsewhere in the file.
- [x] Rerun diagnostic `smoke tiny codex-code` after the shell-safe step2 / placeholder finding-ID validation patch from a clean committed tree and capture the next blocker.
- [x] Align `codex-code` normal draft pre-artifact window with the existing Claude/Qwen draft budget and require `step2.asis_docs` first provider item to be command execution, not assistant/status prose.
- [x] Rerun diagnostic `smoke tiny codex-code` after the Codex draft pre-command stall fix from a clean committed tree and classify the outcome.
- [x] Rerun diagnostic `smoke tiny codex-code` after Codex quota/auth is available again; current provider usage-limit blocker prevented reaching `step2`.
- [x] Remove the remaining init-only `step2.asis_docs` first-pass stale downstream-index wording: when current-run final/citation indexes are not present yet, overview must omit downstream index status instead of saying they are unavailable.
- [x] Remove the remaining init-only `step4.proposals` first-pass low-actionability repair path: medium/high findings must link exact finding ID, affected surface/path and concrete operator action before repair.
- [ ] Run broader trusted validation only after Phase A is `Excellent` from a clean committed tree with `qwen`, `claude`, and `codex` available in `PATH`.

2026-07-15 deterministic follow-up: fake as-is evidence no longer leaks taskrun staging paths;
normal step2 first-pass guidance now explicitly omits absent downstream-index status; normal
step4 self-check requires exact same-line medium/high finding linkage before manifest write.
The two remaining prompt/fake-output implementation items above are complete in code and focused
tests; the clean-tree trusted rerun remains open and is intentionally separate from deterministic DoD.

### Non-goals
- [ ] Do not weaken draft validation or artifact-quality gates.
- [ ] Do not synthesize draft markdown deterministically inside ACP.
- [ ] Do not add live profile/provider/matrix/repo-specific strings to ProvenArch product/runtime code.
- [ ] Do not treat diagnostic smoke as release readiness.

### Acceptance criteria
- [x] Focused prompt/validation tests cover same-turn draft completion, exact step2 shard counts, exact step4 finding linkage/actionability, and clean step0 scaffold wording.
- [x] Runtime quality tests prove valid controlled stops do not emit `runtime_quality.stall_pressure`, while invalid/no-fresh/pre-artifact stalls still do.
- [x] Python batch fixtures prove `valid_artifact_controlled_stop` does not block `Excellent`, while actual repair-heavy/finding-linkage repair still does.
- [x] `make contracts`, `make test`, `make lint`, `make build` pass with exact Node toolchain.
- [x] Diagnostic `smoke tiny` on `codex-code` is rerun and evaluated for strict PASS, artifact quality PASS, repair/stall pressure and label.
- [x] Latest P1 exact shard-completeness patch has full DoD evidence.
- [x] Latest P1 exact shard-completeness patch has diagnostic `smoke tiny codex-code` evidence before any broader trusted matrix.
- [x] Latest P1 step4 repair-to-enrichment route patch has full DoD and diagnostic `smoke tiny codex-code` evidence before any broader trusted matrix.
- [x] Latest P1 manifest-last/required-sections prompt hardening has full DoD and diagnostic `smoke tiny codex-code` evidence before any broader trusted matrix.

### Progress Log
- 2026-07-04: Full DoD passed before live validation, then diagnostic `smoke-tiny-bank-20260704T135401Z` ran selected `codex-code` only. The run failed strict runtime contract at `init.step4.proposals`; artifact quality did not fail, but promotion stopped before refresh. New live diagnostics correctly exposed step-level blockers: `step4.proposals` repair exhausted, `step2.asis_docs` invalid first-pass scaffold, `step0.constitution` downstream wording, and valid controlled stops for non-blocking lifecycle telemetry.
- 2026-07-04: Follow-up patch after that live evidence made step4 actionable-finding prompts explicit that one high/medium finding must be represented by one bullet with all required fields on the same bullet line, because the provider split ID/severity/path and action/residual-gap across separate bullets. The same patch fixed live classification so raw provider stdout and placeholder-hint wording no longer mask `low_actionability` as `markdown_syntax` or `finding_linkage`.
- 2026-07-04: Step0 normal prompts now explicitly ban `later collection steps` / later-analysis / future-pipeline wording in `charter-overview.md`; unknowns must be phrased as current constitution evidence gaps.
- 2026-07-04: Diagnostic `smoke-tiny-bank-20260704T145052Z` still failed strict at `init.step4.proposals`, but the final provider-authored proposal/changelog after cleanup contained exact current-run finding IDs, severity, affected surfaces and recommended actions. The remaining terminal validation was a false positive on `findings above`; shared draft validation now allows that wording when the same draft contains substantive linked findings/proposals, while still rejecting dangling references without linked content.
- 2026-07-04: Live report aggregation now treats `retry scheduled` with action `terminate_and_validate` as valid-artifact controlled completion, not repair pressure. Normal step2/step4 prompts now include bounded public evidence hints: exact shard completeness parsed from current-run shard-summary `items[].status` and a current-run finding ID/severity preview. Full DoD passed with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`.
- 2026-07-04: Diagnostic `smoke-tiny-bank-20260704T154848Z` stopped at precheck because this active ExecPlan had no open goal after documentation updates; this was classified as docs bookkeeping, not runtime evidence, and `go test ./internal/docsync` passed after adding the open live-validation goal.
- 2026-07-04: Diagnostic `smoke-tiny-bank-20260704T155014Z` completed with non-release matrix `PASS`: `hard_pass=1`, `runtime_contract_status=passed`, `artifact_quality_status=passed`, `quality_gates_failed=0`, `artifact_quality_failed=0`, and no product `artifact_quality.*` signals. Verdict remains `Needs review` with `runtime_quality.repair_heavy` and `runtime_quality.stall_pressure`: `repair_attempts=4`, `focused_repairs=4`, `repair_exhausted=0`, `stall_count=6`, `valid_artifact_controlled_stops=6`. Step-level blockers are `init.step2.asis_docs` and `refresh.step2.asis_docs` (`placeholder_or_scaffold`: first-pass overview lacks concrete repo/path/citation/staged refs) plus `init.step4.proposals` and `refresh.step4.proposals` (`finding_linkage`: first-pass proposal lacks exact current-run finding IDs). This confirms strict/artifact-quality success and identifies the remaining no-repair hardening target for `Excellent`.
- 2026-07-05: Started the no-repair hardening slice for the remaining `step2`/`step4` blockers. Normal `step2.asis_docs` and `step4.proposals` prompts now use evidence-first first-action contracts: bounded current-run evidence read/list and validation-ready manifest/markdown writes happen inside the first filesystem work unit, while bootstrap heredoc remains limited to focused repair.
- 2026-07-05: Full DoD passed for the initial P1 prompt slice, then diagnostic `smoke-tiny-bank-20260705T104741Z` ran selected `codex-code` only through `scripts/full-run-batch-matrix.sh`. Result: strict `FAIL` with `runtime_contract_failed`; artifact quality stayed passed (`artifact_quality_failed=0`). The useful signal is that `step2.asis_docs` first-pass output became valid without focused repair, while `init.step4.proposals` exhausted repair as `manifest_shape` because both final proposal files omitted the exact current-run shard completeness literal `planned=10 succeeded=10 failed=0 incomplete=0`. The same run also showed a real `init.step1.collect` pre-artifact stall, so broader trusted matrices remain blocked until Phase A reaches `Excellent`.
- 2026-07-05: Follow-up patch tightened normal and focused `step4.proposals` prompts/repair hints so both proposal and changelog outputs must carry the exact `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` literal and explicit `no-shard-coverage-blocker` statement when typed shard status is readable. This remains provider-authored only; ACP still validates and guides retry instead of synthesizing markdown.
- 2026-07-05: Full DoD passed after the exact shard-completeness patch with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`. Coverage included contracts, Go tests, 229 Python batch/script tests, UI Vitest/typecheck/build and final Go build; Vite reported only existing large-chunk warnings.
- 2026-07-05: Diagnostic `smoke-tiny-bank-20260705T114209Z` ran selected `codex-code` after the exact shard-completeness patch. Result: strict `FAIL` with `runtime_contract_failed`, `artifact_quality_failed=0`, clean collect (`10/10` succeeded) and valid `step2.asis_docs`; remaining blocker is `init.step4.proposals` `draft_artifact_repair` exhaustion. Raw provider output did read staged findings and wrote substantive linked proposal/changelog, but validation failed because both files still omitted exact proposal shard completeness. Runtime classified this as terminal repair failure instead of routing to enrichment/cleanup, so broader trusted matrices remain blocked.
- 2026-07-05: Follow-up patch broadened draft recovery routing for recoverable semantic/shape draft failures: `runtime draft manifest outputs are invalid`, malformed markdown, missing exact shard completeness, missing proposal sections, missing finding linkage/actionability, stale structured-finding denial and process/downstream wording now route to provider-authored `draft_artifact_enrichment`; structural missing draft files still use draft repair first. The shard-status cleanup retry now also recognizes step4 `proposal shard completeness` and instructs provider to rewrite `proposal.md` and `changelog.md` with exact `planned=<n> succeeded=<n> failed=<n> incomplete=<n>` plus `no-shard-coverage-blocker`, preserving finding IDs/actionability.
- 2026-07-05: Full DoD passed after the recovery-routing patch, then diagnostic `smoke-tiny-bank-20260705T123819Z` ran selected `codex-code` through `scripts/full-run-batch-matrix.sh`. Result: non-release `PASS`, `strict_pass_runs=1`, `strict_fail_runs=0`, `artifact_quality_failed_runs=0`, and no `artifact_quality.*` blockers. It is still not `Excellent`: live reported `runtime_quality.repair_exhausted=1`, `runtime_quality.repair_heavy=1`, and `runtime_quality.stall_pressure=1`; telemetry totals were `repair_attempts=4`, `focused_repairs=4`, `repair_exhausted=1`, `stall_count=5`, `pre_artifact_stalls=1`, `post_artifact_stalls=4`, and `valid_artifact_controlled_stops=5`. Step-level blockers: `refresh.step2.asis_docs` produced a manifest-first/pre-markdown path classified as `actual_stall`, `placeholder_or_scaffold`, `repair_attempts=3`; both `init.step4.proposals` and `refresh.step4.proposals` still needed one repair retry because first-pass `proposal.md` was missing validator-required substantive sections (`Decision / recommended operator action`, `Proposed changes or follow-up plan`, etc.). Broader trusted matrices remain blocked.
- 2026-07-05: Follow-up prompt hardening after `smoke-tiny-bank-20260705T123819Z`: normal `step2.asis_docs` now requires markdown targets to be written before the manifest and explicitly calls manifest-only first writes invalid; normal `step4.proposals` does the same for proposal/changelog before `proposals-draft-manifest.json` and names all validator-required proposal/changelog sections in the first-pass contract. This remains provider-authored only; ACP does not synthesize markdown.
- 2026-07-05: Full DoD passed after the manifest-last/required-sections prompt hardening with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`. Coverage included contracts, Go tests, 229 Python batch/script tests, UI Vitest/typecheck/build and final Go build; Vite reported only existing large-chunk warnings.
- 2026-07-05: Diagnostic `smoke-tiny-bank-20260705T141233Z` ran selected `codex-code` through `scripts/full-run-batch-matrix.sh` after the manifest-last/required-sections patch. Result: matrix `FAIL`, strict `FAIL`, `runtime_contract_status=failed`, `artifact_quality_failed=0`, `artifact_quality_status=needs_review`, and no product `artifact_quality.*` blockers. The run did not validate the new `step2`/`step4` happy path because init collect stopped partial: typed collect summary was `planned=10 succeeded=9 failed=1`, with `findings/as-is/proposals` skipped under `report_mode=incomplete`. The failed shard was the root-file shard `bank-of-anthos-gitignore-pylintrc-license-makefile-readme-md-mvnw-mvnw-cmd-pom-xml-a70095e6d2df`; raw stdout only contained Codex `thread.started` / `turn.started`, stderr was empty, no authored collect files were written, and runtime ended as `collect_pair_repair` `runtime_stalled_before_artifacts`. Live reports correctly exposed `init.step1.collect` as the only step-level `Excellent` blocker (`runtime_quality.repair_exhausted`, `runtime_quality.stall_pressure`, `final_validation_class=manifest_shape`). Broader trusted matrices remain blocked until this collect no-artifact path is fixed and Phase A reaches `Excellent`.
- 2026-07-06: Scoped the timeout fix for that Codex root-file collect failure: `codex-code` collect now uses a 5-minute initial and retry pre-artifact stall window, and shared collect-pair repair uses 5-minute pre/post/partial artifact windows. This does not weaken artifact-quality gates or synthesize artifacts; it only gives live Codex collect/recovery enough bounded time after transient stream reconnects. Phase A still requires a fresh `smoke tiny codex-code` rerun to prove `Excellent`.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T062358Z` proved the 5-minute window was applied and the root-file collect shard wrote artifacts; failure moved to manifest shape. Initial manifest missed `semantic.questions[2].text`; provider-authored `collect_manifest_repair` fixed questions but wrote repeated `citations[].id` for multiple repo files. Follow-up slice hardens normal/repair prompt self-checks, passes terminal validation error into manifest repair focus, adds one provider-authored `collect_manifest_shape_cleanup` retry for clean manifest-shape errors, and surfaces terminal validation excerpts in live reports.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T074548Z` ran selected `codex-code` from clean commit `efb849d` after the collect manifest shape-cleanup slice. Result: non-release matrix `PASS`, `strict_pass_runs=1`, `runtime_contract_status=passed`, `artifact_quality_status=passed`, `quality_gates_failed=0`, `artifact_quality_failed=0`, and no product `artifact_quality.*` signals. The collect blocker is fixed for this smoke: init and refresh collect both completed `10/10`, including the previously failing root-file shard. Verdict remains `Needs review`, not `Excellent`, because `init.step2.asis_docs` still used focused repair/enrichment: the normal provider turn missed `runtime/step2_as_is/asis-draft-manifest.json`, the repair pass then missed exact current-run shard completeness (`planned=10 succeeded=10 failed=0 incomplete=0`) and a decision-ready operator summary, and enrichment recovered the final artifacts. Broader trusted matrices remain blocked until this step2 first-pass repair path is removed.
- 2026-07-06: Started the `init.step2.asis_docs` first-pass completion slice. The target is not new validation policy or live acceptance; it is to make the normal provider turn mechanically finish the existing contract by writing `overview.md`, `summary.md`, `architect-summary.md`, and `asis-draft-manifest.json` in one bounded evidence-first command, with exact typed shard completeness and `no-shard-coverage-blocker` visible in the as-is markdown before any repair path is needed.
- 2026-07-06: Implemented the step2 first-pass prompt hardening. Normal `step2.asis_docs` prompt now includes a compact same-command write sequence with exact `write_root`/`draft_root`, all three markdown targets, manifest-last write and `test -s` checks for all four required files; as-is guidance now asks both `summary.md` and `architect-summary.md` to carry exact typed shard completeness plus `no-shard-coverage-blocker` when all current-run shards succeeded. Full DoD passed with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`; Vite reported only the existing large-chunk warning. Phase A still requires a clean-tree `smoke tiny codex-code` rerun before `claude-code` cross-check.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T094200Z` ran selected `codex-code` from clean commit `33c9c16`. Init collect completed `10/10`, and `init.step2.asis_docs` did create all four required targets in the first filesystem work unit, but provider-authored markdown lost path/index refs because backticked Markdown was written through shell-expanded text; validator accepted empty slots such as `Typed shard evidence checked:  and .`, so this slice exposed a product validation gap. `init.step4.proposals` then scheduled focused repair because first-pass `Top Actionable Findings` used `Finding ID: none` despite non-empty current-run findings. Since `Excellent` was already impossible after focused repair, the diagnostic matrix was stopped during refresh collect (`2/10` checkpointed) instead of waiting for a full non-Excellent report.
- 2026-07-06: Follow-up patch after `smoke-tiny-bank-20260706T094200Z` made normal step2 prompt shell-safe by requiring single-quoted heredocs or literal Python writes and self-checking for empty evidence slots. Shared draft validation now rejects empty evidence reference slots, and shared step4 validation rejects placeholder actionable Finding ID values such as `none` when current-run findings are non-empty. Targeted Go tests cover both live-shaped failures.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T104631Z` ran selected `codex-code` from clean commit `3fe91d2`. `init.step0.constitution` completed cleanly and `init.step1.collect` completed `10/10`, confirming the collect timeout/manifest-shape fixes remained effective. The run then hit `init.step2.asis_docs`: normal Codex output only contained `thread.started` / `turn.started`, no command execution, and the default 75s draft pre-artifact window terminated it as `runtime_stalled_before_artifacts`; focused repair was scheduled with `validation_error=.../runtime/step2_as_is/asis-draft-manifest.json: no such file or directory`, so `Excellent` was already impossible and the matrix was stopped.
- 2026-07-06: Follow-up patch after `smoke-tiny-bank-20260706T104631Z` aligns `codex-code` normal draft steps with the existing 180s Claude/Qwen draft pre-artifact window while preserving collect's 5-minute window and repair/excellent blockers. Normal `step2.asis_docs` prompt now explicitly says the first provider item must be command execution and forbids a preliminary assistant/status message before the filesystem command.
- 2026-07-06: Full DoD passed after the Codex draft pre-command stall fix with `ACP_NODE_TOOL_CANDIDATES=/tmp/provenarch-node-22.21.1/node-v22.21.1-darwin-arm64/bin make contracts test lint build`. Coverage included contracts, Go tests, 230 Python batch/script tests, UI Vitest/typecheck/build and final Go build; Vite reported only the existing large-chunk warning. The batch DoD timeout fixture was also made less timing-flaky by using a 2s fake-make timeout so child evidence is consistently written before timeout classification.
- 2026-07-06: Diagnostic `smoke-tiny-bank-20260706T114544Z` ran selected `codex-code` from clean commit `3330d86`. The run did not reach `step2`: `init.step1.collect` completed 4/10 shards, then five consecutive collect shards failed as `runner_unavailable` with Codex stdout `You've hit your usage limit. Visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at Jul 7th, 2026 5:43 AM.` The scheduler aborted the remaining shard after the repeated provider failure threshold. Matrix result was strict `FAIL`, `runtime_contract_status=failed`, `artifact_quality_failed=0`, `artifact_quality_status=needs_review`, `quality_gates_failed=1`, primary `failure_class=runner_unavailable`, and `partial_failure_count=6`. This is an operational provider quota blocker, not evidence against the step2 pre-command fix; Phase A remains blocked until Codex quota/auth is available and the clean-tree smoke can reach `step2`.
- 2026-07-07: Diagnostic `smoke-tiny-bank-20260707T053308Z` ran selected `codex-code` from clean commit `91000a7` after the Codex quota window. The `/tmp/provenarch-node-22.21.1/...` toolchain had a broken `npm` symlink, so the canonical smoke used the same exact Node `22.21.1` via `/Users/griogrii_riabov/.local/share/provenarch/toolchains/node-v22.21.1-darwin-arm64/bin`; harness precheck reported Node/npm ready and completed `make contracts test lint build`. Result: non-release matrix `PASS`, `strict_pass_runs=1`, `runtime_contract_status=passed`, `artifact_quality_status=passed`, `quality_gates_failed=0`, `artifact_quality_failed=0`, and no product `artifact_quality.*` signals. Init and refresh collect both completed `10/10`; refresh was clean (`repair_attempts=0`, `stall_count=0`). The run is still `Needs review`, not `Excellent`, because init quality emitted `runtime_quality.repair_heavy` and `runtime_quality.stall_pressure`: totals `repair_attempts=3`, `focused_repairs=3`, `stall_count=2`, `post_artifact_stalls=2`, `valid_artifact_controlled_stops=4`. Step-level blockers are now narrow and init-only: `init.step2.asis_docs` required two repair attempts plus one stall after first-pass `overview.md` claimed current-run final/citation indexes were unavailable instead of omitting downstream index status (`final_validation_class=manifest_shape`), and `init.step4.proposals` required one repair for low actionability because first-pass proposal did not link medium/high findings to recommended operator action and affected surface/path. Broader trusted matrices and `claude-code` cross-check remain blocked until Codex Phase A reaches `Excellent`.

---

## EP-20260715-console-trust-shell

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the outstanding validation/qualification criterion below on an appropriate trusted host; use the current runbook before any new live execution.

### Context
- Epic 20 starts with contract truth before the Console IA cutover: selected-run evidence, Git inventory, runtime identity and run coordination must be authoritative.
- Epic 18 quality closure remains an independent release-readiness track and must not leak live-provider dependencies into required CI.

### Goals
- Deliver 20A–20I1 as small reviewable slices, ending with one path-based shell at `/setup`, `/home`, `/runs`, `/knowledge` and `/changes`.
- Close deterministic Epic 18 quality/contamination gaps and run the canonical trusted-machine live gate only after deterministic DoD.
- Keep API changes additive where compatibility permits and synchronize docs, TypeScript contracts, validators, fixtures and tests.

### Non-goals
- No 20F2 queue UI, 20I2 deep URL context, 20J–20N or Epic 21 implementation.
- No new runtime providers, hosted mode or security/compliance enforcement.
- No live network/provider dependency in required CI.

### Plan
- [x] 20A: make selected-run evidence an atomic, fail-closed `RunEvidenceSnapshot` with explicit source mode.
- [x] 20B1/20B2: publish complete Git inventory/identity/fingerprint and reject stale mutations before side effects.
- [x] 20C1/20C2: persist runtime identity and separate effective runtime from restart-only desired settings after Console entry.
- [x] 20F1: make ordinary starts fail with typed `run_active`; expose active/pending coordination and typed supersession.
- [x] 20D/20G/20H/20E: add demo identity, safe evidence rendering, accessible primitives and one workflow selector.
- [x] 20I1: atomically replace StageRail with the five-destination native History API shell.
- [x] Epic 18: synchronize quality contracts, remove contamination/actionability defects and add deterministic fixtures.
- [x] Run `make contracts`, `make test`, `make lint`, `make build` and `npm run e2e:mock --prefix ui`.
- [ ] Perform trusted-host preflight; run direct canonical matrix commands only when all prerequisites pass.

### Acceptance
- Selected-run Review/Publish never falls back to current workspace content.
- Git confirmation covers the exact full-workspace mutation set and stale confirmation has no side effects.
- Historical run identity never derives from current UI selection; active/pending coordination is public and typed.
- Only the new shell is interactive after 20I1, with direct URL and Back/Forward coverage.
- Required CI remains deterministic; release readiness is reported only from PASS verdicts plus accepted SWE UX and artifact-quality assessments.

### Results
- The agreed Epic 20 foundation scope through 20I1 is complete: snapshot evidence is fail-closed, Git mutations are fingerprint-confirmed, runtime/coordination state is server-authored, and the legacy StageRail shell is absent from the product DOM. Epic 20 as a whole remains open.
- Epic 18 R1/R2 are complete with promoted-evidence-first assessment language, contamination fixes and regression fixtures.
- Deterministic DoD passed on 2026-07-15: contracts, full Go suite, 246 Python tests, 127 Vitest tests, ShellCheck/typecheck, production build and 7/7 Playwright mock scenarios with critical axe checks.
- Production dependency audit reports zero vulnerabilities after pinning the Mermaid transitive DOMPurify dependency to `3.4.12`.
- R3 is intentionally not executed from this worktree: the canonical trusted-machine gate requires reviewed committed inputs, all three release providers in `PATH`, direct matrix invocation and accepted SWE assessment reports.
- 2026-07-15 Epic 21 preparation audit: exact Go `1.25.10` and Node `22.21.1` toolchains are available; `qwen 0.17.1`, `claude 2.1.85` and `codex-cli 0.144.1` are in `PATH`; `/tmp` is writable with more than the 5 GiB minimum. The current tree is intentionally dirty during implementation and `/tmp/provenarch-live-e2e` canonical pinned checkouts are not provisioned, so smoke/release execution remains deferred until a clean merged commit on a prepared trusted host.

---

## EP-20260715-epic18-r3-composite-release

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Run direct Codex `smoke tiny`, then standalone release fast/long, then fresh release-full fast/long/ftgo-sentry constituents; stop on the first product/provider/host failure.

### Context
- Epic 21 is complete at `b95e72f`; Epic 18 R3 is the remaining release-readiness gate.
- `release full` is three fresh constituent matrices, while the tag workflow currently accepts only
  one verdict path. The trusted host has provider CLIs, writable `/tmp` and sufficient disk, but its
  canonical pinned path checkouts are not bootstrapped.

### Goals
- [x] Add a backward-compatible composite verifier/workflow interface through
  `ACP_RELEASE_MATRIX_IDS` and reject ambiguous or duplicate configuration.
- [x] Reconcile Epic 21 and Epic 18 status across planning, stakeholder, testing and runbook docs.
- [x] Merge the composite-gate slice after full deterministic DoD.
- [x] Bootstrap and verify canonical pinned checkouts on the trusted host.
- [ ] Run direct Codex `smoke tiny`, then standalone release fast/long, then fresh release-full
  fast/long/ftgo-sentry constituents; stop on the first product/provider/host failure.
- [ ] Persist only bounded final release evidence, verify every constituent plus the composite, and
  close Epic 18 only after all verdicts are `PASS` with accepted SWE UX/artifact assessments.

### Non-goals
- No HTTP API, persisted product schema, workspace contract, provider contract or canonical matrix changes.
- No timeout overrides, wrapper scripts, hosted live workflow, release tag or new Wave 1 product epic.
- No commit of raw taskrun directories, provider logs or temporary canonical checkouts.

### Acceptance
- Single-matrix release settings remain compatible; exactly one release-evidence selection mode is required.
- Any missing/failed/duplicate constituent or missing/rejected companion assessment blocks GoReleaser.
- R3 uses only direct `scripts/full-run-batch-matrix.sh` commands from a clean merged commit.
- Release readiness requires exact providers, run index 1, baseline/parallel shard-plan equality,
  strict zero-failure, frontend evidence and three accepted full-release evidence packages.
- Each code/docs PR passes `make contracts`, `make test`, `make lint`, `make build`.

### Progress log
- 2026-07-15: Confirmed exact Go/Node toolchains and qwen/claude/codex CLIs; `/tmp` is writable with
  sufficient disk. Canonical `/tmp/provenarch-live-e2e` checkouts remain the preflight blocker.
- 2026-07-15: Implemented multi-path verifier input, duplicate matrix rejection, mutually exclusive
  workflow settings and focused deterministic tests; synchronized stale Epic 21 tracker language.
- 2026-07-15: Merged the composite gate as `89b4bb49`, bootstrapped and verified all canonical
  pinned checkouts, exact Go/Node/npm toolchains and three release provider CLIs. Direct Codex
  `smoke tiny` matrix `smoke-tiny-bank-20260715T172229Z` stopped before provider execution because
  the live backend-cycle helper treated a successful Epic 21 `no_op` refresh as missing telemetry.
  Opened a bounded remediation: accept the absent legacy quality summary only when the matching
  run-scoped `refresh-execution.json` proves `unchanged_candidate`, `no_op` and skipped providers;
  every missing, malformed or mismatched audit remains a gate failure.
- 2026-07-16: No-op harness remediation merged at `e9ee719c`; Codex smoke
  `smoke-tiny-bank-20260715T181251Z` passed. Standalone matrix
  `release-fast-20260716T043815Z` was stopped on the first live product blocker:
  `qwen-code` shard `bank-of-anthos-extras` failed with `runtime_contract_failed` after focused
  `collect_pair_repair` emitted active stream telemetry for the full pre-artifact wall-clock but
  wrote no authored files. Follow-up `EP-20260716-epic18-r3-qwen-stream-repair` is the only active
  remediation before repeating smoke and release acceptance from a new clean commit.
- 2026-07-16: Qwen stream-only remediation merged at `cae4653a`. The required fresh Codex smoke
  `smoke-tiny-bank-20260716T053113Z` then completed all 10 collect shards but stopped at
  `init.step2.asis_docs`: focused enrichment replaced bootstrap content with the pre-21A generic
  overview headings, so strict Architecture Home validation rejected every required section.
  `EP-20260716-epic18-r3-step2-home-repair` is the active bounded remediation; no release matrix
  or accepted evidence follows until it is merged and smoke passes from the new clean commit.
- 2026-07-16: Step2 Architecture Home remediation merged at `435c2ac8`; fresh Codex smoke
  `smoke-tiny-bank-20260716T065516Z` passed with strict failures `0`. Standalone release-fast
  `release-fast-20260716T085301Z` then stopped on the first terminal blocker: Qwen shard
  `bank-of-anthos-extras` exhausted the allowed stream-only collect-pair retry after ~2.49 MiB of
  thinking telemetry and zero filesystem actions. `EP-20260716-epic18-r3-qwen-tool-first-retry`
  is the only active remediation; release long/full and accepted evidence remain blocked.
- 2026-07-16: Qwen tool-call-first remediation merged at `4eddf559`. The first post-merge Codex
  smoke `smoke-tiny-bank-20260716T095549Z` stopped at refresh `9/10` with
  `runner_unavailable` after Codex exhausted WebSocket TLS reconnects and HTTP fallback; raw
  evidence classified this as an external provider transport incident, so runtime retry policy
  was not expanded.
- 2026-07-16: Requalification smoke `smoke-tiny-bank-20260716T122802Z` ran from the same clean
  `4eddf559` and passed with `strict_pass_runs=1`, `strict_fail_runs=0`, init/refresh collect
  `10/10`, no runtime/provider/contract/flow blockers and no partial failures. Standalone
  `release-fast-20260716T142658Z` then stopped before product execution:
  Qwen's bounded read-only `ACP_READY` probe timed out after 30 seconds. A separate bounded
  readiness recheck reproduced `headless_probe_timeout`; Claude and Codex readiness passed.
  This is an `operational_host_preflight_failed` provider blocker. Release long/full, SWE
  assessments, bounded evidence PR and Epic 18 closure remain blocked until Qwen readiness is
  restored on this or another trusted host.
- 2026-07-16: Qwen artifact-readiness remediation merged as `57155786`. Fresh Codex smoke
  `smoke-tiny-bank-20260716T165233Z` passed with `strict_pass_runs=1`, `strict_fail_runs=0`,
  init/refresh collect `10/10` and no execution blocker. Standalone
  `release-fast-20260716T185527Z` then stopped on the first profile before product execution:
  `qwen --version` passed, but the single canonical runtime-like artifact smoke exited without
  creating the exact sentinel. The batch correctly materialized
  `operational_host_preflight_failed`; the automatically started next sweep was interrupted, and
  release long/full plus accepted evidence remain blocked. No retry, timeout-success exception or
  product-runtime change is justified by this provider/host readiness result.

---

## EP-20260717-epic18-r3-ui-route-preflight

Status: active — retained scope; reconcile current implementation before selecting a new slice.

Next action: Recheck the unresolved acceptance below against current code and canonical status, then choose a minimal authorized slice; the historical plan is not a new implementation order.

### Context
- The clean `ba98798a` deterministic R3 preflight passed contracts, Go and Python tests but the
  full UI suite exposed a route race in the provider-unavailable recovery test.
- The test opened Changes while async run canonicalization was still settling, navigated to Runs,
  and could be returned to Changes before `run-status-panel` rendered. The focused test passed
  `3/3`; production behavior and provider/runtime policy are unchanged.

### Goals
- Start the recovery scenario directly on its stable `/runs/<run_id>` deep link.
- Preserve the existing Runs recovery, Changes publication gate and Setup readiness assertions.
- Pass the full deterministic DoD before restarting R3 from the merged clean commit.

### Non-goals
- No production UI, API, runtime, schema, canonical matrix or timeout change.
- No live provider execution from the remediation branch.

---

## EP-20260718-epic18-r3-install-fixture-platform

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated test remediation and restart Claude qualification from the new merge commit.

### Context
- The post-PR #157 Claude qualification passed the full Go precheck but stopped before provider
  execution when three installer fixture cases disagreed on the release asset architecture.
- The Python fixture created its mock archive from in-process `platform.machine()`, while the
  production installer intentionally resolves the target through child `sh`/`uname`. The live
  precheck observed Python `arm64` versus child-shell `amd64`; the installer correctly rejected the
  absent mock asset. A focused run and 500 shell probes passed afterward, so production installer
  behavior is not implicated.

### Goals
- [x] Make the installer fixture platform deterministic through a test-local `uname` executable.
- [x] Keep archive URL, checksum and installation behavior covered without changing `install.sh`.
- [x] Pass focused stress and full deterministic DoD.
- [ ] Merge the isolated test remediation and restart Claude qualification from the new merge commit.

### Non-goals
- No installer interface, release artifact naming, runtime, provider, schema, API or matrix change.
- No acceptance of missing assets or checksum failures.

### Acceptance
- All installer tests use the same explicit OS/architecture identity seen by the installer process.
- Success, checksum rejection, explicit version and latest-release URL cases retain their assertions.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-18: `smoke-tiny-bank-20260718T144339Z` stopped in deterministic Python precheck with
  three missing/mock-URL failures for `acp_darwin_amd64.tar.gz`; provider execution never started.
- 2026-07-18: All four installer cases passed 20 consecutive focused runs with the deterministic
  test-local platform. Full DoD passed with Go `1.25.10`, Node `22.21.1` and npm `10.9.4`:
  contracts, full Go, 261 Python, 142 UI, lint and embedded UI build are green.

---

## EP-20260719-epic18-r3-snapshot-live-gate-remediation

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the remediation and restart canonical qualification from the clean merge commit.

### Context
- Claude qualification `smoke-tiny-bank-20260719T042213Z` completed init and refresh successfully;
  refresh collect was `10/10` and validator-gated promotion completed.
- Matrix aggregation still failed. ProductShell selected the first final-index suffix from a mixed
  refresh artifact inventory, which was the older init index, then correctly rejected the foreign
  `run_id` and showed zero snapshot artifacts. The live flow additionally required an optional
  diagram document.
- The promoted Architecture Home is 99 lines of concrete architecture evidence, but its legitimate
  guidance to replace a checked-in demo-secret placeholder triggered a substring-only placeholder
  heuristic and produced the sole `artifact_quality.*` blocker.

### Goals
- [x] Select only the exact selected-run final index and retain strict run/staged-path isolation.
- [x] Add a mixed-inventory regression with a foreign index ordered first.
- [x] Keep diagram rendering checks when diagrams exist and require substantive Report inspection
  when the selected snapshot has no diagram documents.
- [x] Restrict placeholder detection to known scaffold/process phrases and standalone markers while
  accepting evidence-backed safe-change language.
- [x] Synchronize architecture, testing, stakeholder and live-gate documentation.
- [x] Pass focused UI/Go tests, mock E2E, keyboard/axe coverage and full deterministic DoD.
- [ ] Merge the remediation and restart canonical qualification from the clean merge commit.

### Non-goals
- No HTTP API, persisted schema, provider contract, retry/timeout, canonical matrix or artifact
  acceptance weakening.
- Runtime repair/stall telemetry remains visible and can cap the qualitative label; only the false
  promoted-artifact blocker and cross-run UI selection are corrected.
- No generated diagram is inserted into a historical run index after validation.

### Acceptance
- A refresh inventory containing init and refresh indexes renders only refresh staged documents.
- A mismatched or missing exact index remains `not_produced/error`; current workspace is never used
  as historical fallback.
- The live flow proves ProductShell routes and reads an indexed Architecture Home whether or not the
  snapshot contains diagrams.
- The observed substantive Architecture Home has no `artifact_quality.overview_placeholder` signal;
  known scaffold and standalone TODO/placeholder fixtures still fail.
- `make contracts`, `make test`, `make lint`, `make build`, and UI mock E2E pass with pinned
  toolchains.

### Progress log
- 2026-07-19: Inspected the failed ProductShell screenshot, live API and copied snapshot. The refresh
  final index contains 47 canonical documents and 166 staged files, but suffix-first lookup chose
  the older init index from the 531-entry persisted inventory. The promoted overview is substantive;
  its only heuristic trigger was the word `placeholder` in secret-rotation guidance.
- 2026-07-19: Focused Go and UI regressions pass. The ProductShell live diagnostic passes against
  the saved refresh snapshot on desktop and `390x844`; Home/Runs/Knowledge/Changes are present,
  legacy shell controls are absent, and Publish prioritizes Architecture Home. Snapshot source and
  frontend-copy indexes contain the same 47 documents with byte-identical staged-final digests,
  same-run citation identity, no foreign staged paths, and no taskrun-path contamination.
- 2026-07-19: Full deterministic DoD is green with the pinned toolchains: contracts, Go/Python/UI
  tests (including 142 UI tests), lint, embedded UI build, and mock E2E `7/7` all pass.

---

## EP-20260728-r3-qwen-architecture-home-marker-contract

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated fix, establish a new qualification SHA, and restart with a fresh smoke matrix ID.

### Context
- Fresh diagnostic matrix `smoke-tiny-bank-20260728T200211Z` ran from qualification SHA
  `96824c47206a9bca95b72f097ad97fb2fceaf36a`. Host/provider/DoD preflight passed and headless
  collect completed all 10 typed shards as `succeeded`.
- `init.step2.asis_docs` then failed strict Architecture Home validation because Qwen wrote the
  exact deterministic validator marker `current analysis covers`. The focused enrichment prompt
  described the error class but did not enumerate that literal marker; it produced stream output
  without a fresh mutation and exhausted as `runtime_stalled_before_artifacts`.
- The matrix is diagnostic failure evidence only: `runtime_contract_failed=1`,
  `repair_exhausted=1`, and `stall_pressure=1`. Regression and release phases did not start.

### Goals
- [x] Keep the Architecture Home process-narration marker set in one validator-owned source.
- [x] Copy the closed marker set into normal step2, first-action, enrichment and compact/command-text
  provider prompts so validation and provider instructions cannot drift.
- [x] Add deterministic tests for defensive-copy behavior and prompt/validator marker parity.
- [x] Synchronize architecture and live-runbook behavior.
- [x] Pass the full provider-free offline closure.
- [ ] Merge the isolated fix, establish a new qualification SHA, and restart with a fresh smoke
  matrix ID.

### Non-goals
- No validator relaxation, ACP-authored narrative sanitization, schema/API change, timeout change,
  provider alias, canonical matrix or curated repository change.
- No acceptance of the stopped smoke, stale drafts, analysis-only recovery output, repair exhaustion
  or stall pressure.

### Acceptance
- Every literal Architecture Home process marker rejected by the validator is present in the normal
  and focused step2 prompt contracts.
- Provider-authored Architecture Home content still passes the complete strict draft contract;
  invalid or unchanged recovery output remains terminal.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-28: Recorded the stopped matrix operator report at
  `/tmp/provenarch-test_arch_project/reports/operator_step_report_smoke-tiny-bank-20260728T200211Z.md`;
  no release evidence was produced or accepted.
- 2026-07-28: Centralized the 21 literal process-narration markers in `internal/runtimedrafts`,
  added a defensive-copy prompt policy line, and wired it into normal and all focused step2 prompt
  modes. Focused runtimedrafts/step-policy/prompt/providercommon tests pass.
- 2026-07-28: Full pinned offline closure passed: race suites, 90 readable fixtures, UI `158/158`,
  mock E2E `7/7`, full Go suite, Python `263/263`, contracts, lint/typecheck, build and deterministic
  embedded `ui_dist`.

---

## EP-20260728-r3-overview-placeholder-line-scope

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated fix, establish a new qualification SHA, and restart with a fresh smoke matrix ID.

### Context
- Fresh diagnostic `smoke-tiny-bank-20260728T211536Z` ran from qualification SHA
  `5a61a86c0d9d51f5466f731b1ca27e29ce27266d`. Host/provider/DoD/UI/browser preflight passed,
  headless init collect completed `10/10`, and step2 passed strict validation with all eight
  Architecture Home sections, concrete repository references, no forbidden process marker and no
  focused repair/stall pressure.
- The succeeded init quality report still emitted `artifact_quality.overview_placeholder`.
  `overviewLooksPlaceholder` searched the entire document for any `no ` and any ` yet`, joining
  separate substantive lines: `No explicit message broker is visible` and
  `Runtime behavior ... is not yet traced`.
- The operator stopped the non-evidence matrix during refresh after proving the deterministic
  product-side false positive. All matrix/provider processes terminated; regression and release
  phases did not start.

### Goals
- [x] Scope the legacy `no ... yet` placeholder phrase to one Markdown line.
- [x] Preserve real same-line placeholder detection such as `no services yet`.
- [x] Add the live-observed multiline substantive wording as a deterministic regression.
- [x] Synchronize architecture and live-runbook behavior.
- [x] Pass full provider-free offline closure.
- [ ] Merge the isolated fix, establish a new qualification SHA, and restart with a fresh smoke
  matrix ID.

### Non-goals
- No weakening of exact scaffold/placeholder/TODO markers, artifact-quality gate policy, schema/API,
  runtime/provider behavior, timeout, canonical matrix or curated repository.
- No acceptance of the stopped matrix or manual suppression of its quality signal.

### Acceptance
- Independent `no <integration>` and `not yet traced` lines do not produce
  `artifact_quality.overview_placeholder`.
- Same-line `no services yet` and existing scaffold/TODO fixtures still produce the warning.
- `make contracts`, `make test`, `make lint`, and `make build` pass with pinned toolchains.

### Progress log
- 2026-07-28: Saved
  `/tmp/provenarch-test_arch_project/reports/operator_step_report_smoke-tiny-bank-20260728T211536Z.md`
  and stopped the diagnostic before it could be mistaken for release evidence.
- 2026-07-28: Moved the `no`/`yet` conjunction into the existing per-line scan. Focused overview
  quality tests pass 50 consecutive runs, including the live-observed multiline regression and the
  original `no services yet` placeholder case.
- 2026-07-28: Full pinned offline closure passed: race suites, 90 readable fixtures, UI `158/158`,
  mock E2E `7/7`, full Go suite, Python `263/263`, contracts, lint/typecheck, build and deterministic
  embedded `ui_dist`.

---

## EP-20260802-r3-qwen-collect-process-provenance-contract

Status: blocked — recorded validation or trusted qualification remains open.

Next action: Resolve the recorded open criterion: Merge the isolated fix, establish a new qualification SHA, and restart R3 from smoke only after Qwen quota/readiness succeeds.

### Context
- Post-merge qualification SHA `a5d25b6ff20c5a9f461871abec4d092bc9dd4985` passed full offline
  closure and fresh smoke `smoke-tiny-bank-20260801T225646Z` passed strict `1/1` with zero repairs,
  retries, stalls or excellent blockers.
- Standalone `regres-fast-bank-openedx-20260801T234621Z` stopped during Bank init. One normal compact
  collect markdown used the validator-rejected phrase `in this bounded read`; its pair repair then
  emitted two lexical `inferred` provenance kinds and required deterministic shape recovery.
- The subsequent proposals call returned explicit Qwen/Kimi billing-cycle quota HTTP 403, and Open
  edX readiness confirmed the same host/provider blocker. No regression or release evidence is
  accepted; live qualification remains paused until provider readiness succeeds.

### Goals
- [x] Enumerate the live-observed process phrases in the compact Qwen collect contract.
- [x] Restrict the minimal Qwen semantic subset to exact `provenance.kind: observation` and route
  unsupported inference into an observed coverage gap.
- [x] Pin both rules in prompt-contract and Qwen adapter regressions.
- [x] Synchronize architecture, pipeline, testing and live-runbook documentation.
- [x] Pass focused stress, full deterministic DoD and offline closure.
- [ ] Merge the isolated fix, establish a new qualification SHA, and restart R3 from smoke only
  after Qwen quota/readiness succeeds.

### Non-goals
- No validator relaxation, ACP-authored semantic rewrite, schema/API, retry/timeout, canonical
  matrix, curated repository, provider alias or acceptance-threshold change.
- No attempt to bypass, suppress or reclassify the explicit provider quota error.

### Acceptance
- Normal compact Qwen collect names every live-observed process phrase before atomic pair write.
- Every compact semantic provenance object is instructed to use exact `observation`; aliases such as
  `inferred` are explicitly forbidden.
- Existing closed root/citation/task identity contracts and the 6 KiB prompt budget remain intact.
- `make contracts`, `make test`, `make lint`, `make build`, and `make offline-closure` pass with
  pinned toolchains.

### Progress log
- 2026-08-02: Recorded the stopped regression operator report at
  `/tmp/provenarch-test_arch_project/reports/operator_step_regres-fast-bank-openedx-20260801T234621Z.md`.
  Bank had independent `runtime_quality.repair_heavy` remediation evidence; Open edX never started
  a backend run after the explicit provider readiness quota failure.
- 2026-08-02: Focused promptcontract/qwencode suites passed once and with `-count=20`. Full pinned
  DoD passed (`contracts`, Go, Python `263/263`, UI `158/158`, lint/typecheck and embedded build),
  followed by independent offline closure: race suites, 90 readable fixtures, mock E2E `7/7`, full
  deterministic DoD and byte-identical embedded `ui_dist` all passed.

---

## Retained live follow-up notes

Status: blocked — historical live acceptance remains unresolved in the original notes below.

Next action: reconcile the applicable incident against current implementation and the retained W25/R3
plan before choosing a new run. Do not treat old commands, model pins or provider availability as
current instructions. These notes keep their original titles because they never had Plan IDs.

### Live E2E Draft Manifest Metadata Tolerance

#### Context
Strict medium `codex-code` diagnostic run `regres-long-posthog-ftgo-20260618T225252Z` reached PostHog `refresh.step2.asis_docs` after successful collect, but failed as `runtime_contract_failed` during `draft_artifact_enrichment`. The provider rewrote evidence-backed markdown and produced a valid publish mapping, but added top-level `updated_at` to `asis-draft-manifest.json`; the shared draft parser rejected it as an unknown field.

#### Plan
- [x] Keep runtime draft manifests strict and continue rejecting legacy/envelope fields such as `repo_scopes`, `compatibility`, `generated_at`, `pipeline`, and `proposals[]`.
- [x] Allow only bounded optional metadata `updated_at` alongside existing `summary`.
- [x] Update prompt contract wording, schema docs, runbook, and tests.
- [ ] Rerun strict medium `codex-code`; then run strict medium `claude-code` if Codex produces clean evidence.

#### Acceptance
- [x] `updated_at` in `asis-draft-manifest.json` validates without changing execution/artifact quality separation.
- [x] Existing unknown-field tests still reject legacy manifest drift.
- [ ] Latest strict medium `codex-code` and `claude-code` runs pass with artifact/UX quality accepted or explicitly non-applicable with no blocker.

#### Progress log
- 2026-06-19: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T002751Z` reached non-release machine `PASS` for selected provider totals (`2/2`) with frontend PASS and no legacy mixed quality-gate artifacts, proving execution/artifact quality separation. Manual artifact review rejected the result because promoted FTGO docs cited non-existent Maven `pom.xml` files in a Gradle repo; `artifact_quality:*` telemetry surfaced evidence-scope warnings but correctly did not flip the machine verdict.
- 2026-06-19: Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260619T194014Z` was stopped during PostHog refresh after `posthog-share-staticfiles` reproduced a narrower blocker: manifest-only repair changed invalid citations from missing `share/Caddyfile` / `staticfiles/admin/tailwind.css` to existing files, but left the authored markdown claiming those missing paths. The fix escalates structural missing-repo-evidence manifests to collect pair repair when existing markdown still names the missing path, requires fresh markdown rewrite, and treats stale/noop markdown as terminal `runtime_contract_failed`.

---

### Live E2E Collect Repo Evidence Path Strictness

#### Context
The same strict medium `codex-code` run proved that missing repo evidence paths can leak past collect validation into promoted `step2` markdown: FTGO final docs referenced `ftgo-order-service/pom.xml` and `ftgo-order-service-api/pom.xml`, while the pinned repo only contains Gradle build files for those modules. This is not a manual quality preference; it is a broken runtime evidence contract because collect citations/provenance pointed to files that do not exist under the resolved repo root.

#### Plan
- [x] Validate `citations[].repo/path` against resolved collect repo roots when task context is available.
- [x] Validate semantic provenance evidence paths for entities, edges, and findings against the same resolved repo roots.
- [x] Keep existing validation behavior for callers that do not have repo-root context, so deterministic fixtures and offline contract tests remain scoped.
- [x] Update collect prompt/contract wording so providers remove unsupported claims or record coverage gaps instead of citing guessed files.
- [x] Add targeted unit coverage for missing citation paths, missing semantic evidence paths, generated repo-root suffix aliases, and runtime collect task validation.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

#### Acceptance
- [x] A collect manifest that cites a missing repo file fails as `runtime_contract_failed` / repair input before downstream `step2` can promote the claim.
- [x] Existing `artifact_quality:*` telemetry remains non-gating for machine execution verdict; this fix only enforces concrete runtime evidence paths in collect contracts.
- [ ] Latest strict medium `codex-code` rerun no longer promotes missing repo/path evidence into final artifacts.
- [ ] Latest strict medium `claude-code` rerun remains pending after Codex produces clean strict medium evidence.

---

### Live E2E Collect Process Narration Strictness

#### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T023512Z` preserved selected-provider execution reporting but exposed a narrower collect artifact-quality contract gap: some accepted/derived markdown still narrated runtime collection mechanics (`bounded read`, guessed or expected-missing paths) instead of clean operator-facing architecture evidence. Those strings can later contaminate `step2` as-is docs even when the machine verdict separation is working.

#### Plan
- [x] Reject process-contaminated collect markdown in strict collect validation.
- [x] Route existing process-contaminated authored markdown to provider-authored `collect_pair_repair` with mandatory fresh rewrite of the same markdown target.
- [x] Keep deterministic `collect_manifest_runtime_recovery` limited to process-clean authored markdown so it cannot turn runtime narration into hidden success.
- [x] Update prompt contracts, docs, and tests for process-narration/guessed-path bans.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

#### Acceptance
- [x] Final collect docs mentioning bounded reads/passes, guessed paths/files/evidence, expected-missing path checks, recovery attempts, or later repair fail validation.
- [x] Manifest-only recovery does not accept process-contaminated markdown.
- [ ] Latest strict medium `codex-code` rerun no longer promotes process-narrated collect evidence into final artifacts.

---

### Live E2E Collect Path-Scope Evidence Fairness

#### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T051657Z` ran from clean commit `6057fbb` with selected provider `codex-code` only. The new execution/UX/artifact split held: `matrix_result_*` stayed non-release, selected-provider totals were `0/2`, `execution_report_*` replaced legacy quality reports, frontend was `skipped` because snapshots were missing, and no old `quality_report_*` / `quality_gates_failed` / `failure_reason=quality` evidence appeared. Backend failed in collect for both targets: PostHog `posthog-cli-common` cited stale missing paths (`cli/package.json`, `cli/docker-compose.yml`, `common/hogvm/python/README.md`) and `collect_pair_repair` produced only Codex lifecycle output; FTGO had 15/16 succeeded shards but `ftgo-restaurant-service-api...` was falsely routed to process-contamination repair because a legitimate missing Swagger spec coverage gap said an expected resource path was not present.

#### Plan
- [x] Narrow process-contamination detection so concrete missing path claims remain invalid, but operator-facing missing spec/scope coverage gaps are allowed.
- [x] Add concrete path-scope file candidates to normal collect prompts, selected fairly per assigned directory scope.
- [x] Apply the same per-scope candidate fairness to collect pair/manifest repair prompts.
- [x] Add tests for `cli/common`-style multi-scope candidates and the missing-spec coverage-gap false positive.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

#### Acceptance
- [x] Multi-scope candidates include real files from later scopes and do not list nonexistent stale paths.
- [x] `expected src/test/... path was not present` remains invalid process contamination.
- [x] `no OpenAPI/Swagger spec was observed under this scope` remains a valid coverage gap when not used as citation/provenance evidence.
- [ ] Latest strict medium `codex-code` rerun reaches collect completion without the `posthog-cli-common` stale-path blocker or FTGO missing-spec false positive.

---

### Live E2E Draft Artifact Readability And Index Timing

#### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260620T203021Z` reached non-release machine `PASS` for selected-provider totals (`2/2`) with frontend PASS and no legacy mixed quality-gate artifacts. Manual artifact review still rejected the result: some final `step2.asis_docs` markdown claimed current-run `final-run-index.json` / `citation-index.json` were unavailable even though final staging later contained those indexes, and some `step4` proposal/changelog content pasted raw Python-style citation dictionaries/truncated JSON fragments instead of readable operator evidence.

#### Plan
- [x] Reject stale current-run final/citation-index unavailable claims in runtime draft markdown.
- [x] Reject raw structured evidence dumps in operator-facing draft markdown.
- [x] Update draft enrichment prompts so providers omit downstream index availability during step2 when indexes are not yet present, and summarize index/citation evidence instead of pasting raw objects.
- [x] Update docs and targeted prompt/validation tests.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

#### Acceptance
- [x] `step2` no longer promotes stale `final-run-index/citation-index not found` claims.
- [x] `step4` proposal/changelog drafts cannot pass validation with raw `{'id': ...}` / `documents=[{...}]` / `citations=[{...}]` dumps.
- [ ] Latest strict medium `codex-code` rerun has accepted artifact quality for readability, index truthfulness and decision readiness.
- [ ] Latest strict medium `claude-code` rerun remains pending after Codex produces clean strict medium evidence.

---

### Live E2E Proposal Final-Index Truthfulness

#### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260621T022237Z` reached non-release machine `PASS` for selected-provider totals (`2/2`) and frontend PASS. Runtime recovery behaved correctly for bootstrap-only drafts, but manual artifact review rejected FTGO proposals: `proposals/runtime-recommendations.md` and `reports/changelog/runtime-proposals.md` stated `No current-run final-run-index document list was available` even though `final-run-index.json` was present with `51` canonical documents.

#### Plan
- [x] Extend runtime draft validation to reject stale final-index document-list availability claims.
- [x] Update `step4.proposals` enrichment prompt to require canonical document count summaries when `final-run-index.json` is present and omission when it is absent.
- [x] Add regression coverage using the observed FTGO stale phrase.
- [x] Update live E2E docs/spec/architecture wording.
- [ ] Run full DoD, commit, and rerun affected strict medium `codex-code`.

#### Acceptance
- [x] Proposal/changelog drafts cannot pass validation with `No current-run final-run-index document list was available`.
- [ ] Latest strict medium `codex-code` rerun has accepted artifact quality for proposal index truthfulness and decision readiness.
- [ ] Latest strict medium `claude-code` rerun remains pending after Codex produces clean strict medium evidence.

---

### Live E2E Proposal Count And Mermaid Collision Follow-Up

#### Context
Strict medium `codex-code` rerun `regres-long-posthog-ftgo-20260621T193026Z` reached non-release machine `PASS` for selected-provider totals (`2/2`) with frontend PASS and no legacy mixed quality-gate artifacts. Manual artifact review still found two artifact-quality issues: FTGO proposal/changelog used the variant `final-run-index.json (0 observed document entries)` even though the current `final-run-index.json` had `51` `canonical_documents`, and FTGO C4 container output duplicated a Mermaid node id for distinct service ids that normalized to the same slug.

#### Plan
- [x] Reject parenthesized/styled stale zero-document `final-run-index` claims in runtime draft markdown.
- [x] Make C4 diagram generation collision-free for sanitized Mermaid node ids.
- [x] Make generated component/code diagram artifact paths collision-free for distinct entity ids that normalize to the same slug.
- [x] Update spec/runbook/testing docs and targeted tests.
- [x] Run full DoD.
- [ ] Commit and rerun affected strict medium evidence.

#### Acceptance
- [x] Proposal/changelog drafts cannot pass validation with `final-run-index.json (0 observed document entries)` without validated zero-document evidence.
- [x] Distinct model entity ids such as `svc.foo-bar` and `svc.foo.bar` produce distinct Mermaid node ids and distinct diagram artifact paths.
- [ ] Latest strict medium `codex-code` rerun has accepted artifact quality for proposal index truthfulness and C4/Mermaid usefulness.
- [ ] Latest strict medium `claude-code` rerun remains pending after Codex produces clean strict medium evidence.

---

### Live E2E Collect Repair Canonical Semantic Shape

#### Context
Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T071328Z` reproduced a collect recovery failure before a full profile completed. The first PostHog shard failed as `runtime_contract_failed`; later shards showed the same provider behavior but were sometimes rescued by manifest-only repair. The repeated root cause was `collect_pair_repair` writing legacy `shard-pack-manifest.json` semantic shape: `semantic.coverage.notes` as a string, entities with direct `repo/path/evidence` or missing `provenance`, edges with `relation/source/target`, findings without `description`, and provenance as direct `{repo,path}` or missing `kind/confidence/evidence[]`.

#### Plan
- [x] Strengthen collect manifest contract lines and compact checklist with canonical semantic object shapes.
- [x] Add canonical semantic-shape guidance to collect pair repair and manifest-only repair prompts.
- [x] Add prompt contract tests for the exact live failure fields and forbidden aliases.
- [x] Update live E2E runbook/spec/architecture/testing docs.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium `claude-code`.

#### Acceptance
- [x] Prompt contracts explicitly reject legacy collect semantic aliases that appeared in the failed `claude-code` run.
- [ ] Latest strict medium `claude-code` rerun no longer fails from unchanged/legacy collect repair manifest shape.
- [ ] Latest strict medium `claude-code` and `codex-code` evidence both reach clean execution and accepted manual artifact/UX decisions.

---

### Live E2E Collect Repair Canonical Path Mapping

#### Context
Strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T082911Z` confirmed that the canonical semantic-shape fix worked for `posthog-bin`, but exposed the next collect repair drift. The compact `collect_pair_repair` wrote provider-authored markdown and canonical semantic objects, yet set `documents[0].canonical_path` to the live staging path `reports/taskruns/.../staging/shards/posthog-bin/bin-overview.md`. Strict contract validation correctly rejected it, then `collect_manifest_repair` stalled/no-op without replacing the manifest. The run was stopped after terminal shard failure because the matrix could no longer become acceptance evidence.

#### Plan
- [x] Centralize stable collect canonical path generation from shard slug + authored doc path.
- [x] Add exact `documents[].path -> documents[].canonical_path` mapping to manifest-only repair prompts.
- [x] Add exact `documents[0].canonical_path` to compact collect pair repair prompts and explicitly forbid `reports/taskruns/**`, `/staging/`, absolute `write_root`, raw runtime paths and duplicated artifact-root canonical paths.
- [x] Add prompt/skeleton contract tests for the live-observed drift.
- [x] Update live E2E runbook/spec/architecture/testing docs.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium `claude-code`.

#### Acceptance
- [x] Compact collect pair repair prompt no longer leaves canonical path inference open.
- [x] Manifest-only repair prompt lists stable canonical-path mappings for existing authored docs.
- [ ] Latest strict medium `claude-code` rerun no longer fails on staging/taskrun `documents[].canonical_path`.
- [ ] Latest strict medium `claude-code` and `codex-code` evidence both reach clean execution and accepted manual artifact/UX decisions.

---

### Live E2E Step0 Bounded-Read Marker And Claude Collect Window

#### Context
- 2026-06-23 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T113848Z` validated the step2 hardening: downstream `step2/3/4` no longer masked partial collect, and machine execution reports kept artifact quality telemetry out of the execution verdict.
- PostHog failed at collect as execution-only `runner_unavailable`: two shards produced no stdout/stderr and no authored files across normal collect, fresh retry and focused collect-pair repair, each bounded by the 180s pre-artifact window.
- FTGO failed at `init.step0.constitution` due a draft-validation false positive: provider-authored `charter-overview.md` was evidence-backed and decision-ready, but the hard scaffold marker `bounded read` rejected the valid coverage-gap sentence “not inspected in this bounded read.”

#### Plan
- [x] Narrow draft scaffold markers from generic `bounded read` to process-specific markers such as `bounded read roots`, `bounded read pass`, `bounded evidence read` and `bounded staged evidence`.
- [x] Add regression coverage that accepts evidence-backed step0 coverage gaps using `bounded read` and still rejects recovery-process `bounded read roots`.
- [x] Extend `claude-code` collect initial/retry pre-artifact window to 5 minutes while leaving draft/enrichment windows unchanged.
- [x] Update prompt contract wording and live E2E docs/spec/testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium `claude-code`.

#### Acceptance
- [x] Evidence-backed `charter-overview.md` with a normal bounded-read coverage gap passes draft validation.
- [x] Recovery/process bounded-read markers remain contract-invalid.
- [x] Claude collect policy exposes 5-minute initial/retry pre-artifact windows for `init|refresh.step1.collect`.
- [ ] Latest strict medium `claude-code` rerun no longer fails from the FTGO false-positive marker and has enough collect budget to test PostHog shard recovery.

---

### Live E2E Collect Pair Silent No-Fresh Retry

#### Context
- 2026-06-23 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T182236Z` ran from clean commit `f5c35cc` with selected provider `claude-code` only.
- PostHog backend passed init/refresh, frontend desktop smoke passed, `execution_report_*` replaced legacy quality reports, and artifact-quality telemetry stayed out of the machine verdict.
- FTGO failed in `init.step1.collect`: 15/16 shards succeeded, but shard `ftgo-application-ftgo-kitchen-service-api-ftgo-kitchen-service-contracts-ftgo-order-cc671c29a0aa` wrote stale/invalid collect files, then focused `collect_pair_repair` stalled pre-artifact with `stdout=0`, `stderr=0` and no fresh authored mutation. Reports preserved the root as collect partial execution failure (`runtime_contract_failed`, `partial_failure_count=1`), with frontend skipped for that profile.
- Root issue: collect-pair repair treated a silent no-fresh pre-artifact repair stall over stale artifacts as final exhaustion. It needed one bounded provider-authored retry, without deterministic artifact synthesis and without treating artifact quality as a machine gate.

#### Plan
- [x] Give collect-pair repair its own activity policy so production/default collect repair windows remain bounded for live providers while tests can exercise explicit wall-clock caps.
- [x] Add one focused retry when collect-pair repair stalls before fresh authored mutation with empty stdout/stderr and the provider policy allows zero-output retry.
- [x] Keep success provider-authored only: retry must write markdown + `shard-pack-manifest.json` and pass strict validation.
- [x] Classify repeated silent no-fresh collect-pair repair exhaustion as `runner_unavailable`; classify fresh-but-invalid repair output as `runtime_contract_failed`.
- [x] Add providercommon regression tests for successful silent/no-fresh retry and exhausted silent/no-fresh classification.
- [x] Update live E2E runbook, pipeline spec, architecture and testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

#### Acceptance
- [x] First silent/no-fresh collect-pair repair does not emit final exhausted telemetry before the allowed retry.
- [x] Second provider-authored repair can recover stale/process-contaminated collect artifacts only by fresh rewriting the markdown and manifest.
- [x] Repeated silent/no-fresh repair exhaustion is `runner_unavailable`.
- [ ] Latest strict medium `claude-code` rerun no longer fails from the FTGO silent/no-fresh collect-pair repair stall.

---

### Live E2E Draft Enrichment Marker Cleanup Retry

#### Context
- 2026-06-24 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260623T230003Z` proved the silent collect-pair retry and draft enrichment paths were active, but still failed both selected backend slots.
- PostHog `refresh.step4.proposals` fresh-rewrote `proposal.md` and `changelog.md`, but `changelog.md` leaked process wording (`bounded read roots`) into operator-facing markdown. Runtime correctly rejected it as `runtime_contract_failed`, but had no targeted cleanup retry after a real markdown mutation.
- FTGO `init.step0.constitution` fresh-rewrote `charter-overview.md`, but leaked step0-invalid process/downstream wording (`validator output`, `draft manifest`, `later passes`). Runtime correctly rejected it as `runtime_contract_failed`.

#### Plan
- [x] Add one provider-authored `draft_artifact_enrichment_marker_cleanup` retry when every referenced markdown target changed, but strict validation only rejects scaffold/process/downstream wording.
- [x] Keep unchanged scaffold/noop enrichment terminal: cleanup retry is not available when markdown targets did not fresh-mutate.
- [x] Extend enrichment prompts with marker-cleanup instructions for step0 and step4.
- [x] Add providercommon and promptcontract regression tests for proposals marker cleanup and step0 downstream/process cleanup.
- [x] Update live E2E runbook, pipeline spec, architecture and testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

#### Acceptance
- [x] Fresh-rewritten proposal/changelog content with process marker contamination gets one provider-authored cleanup retry.
- [x] Fresh-rewritten step0 constitution content with downstream/process wording gets one provider-authored cleanup retry.
- [x] Repeated marker contamination or unchanged scaffold remains `runtime_contract_failed`.
- [ ] Latest strict medium `claude-code` rerun no longer fails from PostHog `bounded read roots` or FTGO step0 marker leakage.

---

### Live E2E Draft Enrichment Shard Status And Write-Set Cleanup

#### Context
- 2026-06-24 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260624T055541Z` ran selected provider `claude-code` only with selected-provider totals preserved and no legacy mixed quality artifacts, but both backend slots failed as `runtime_contract_failed`.
- PostHog `refresh.step2.asis_docs` completed collect 16/16 and fresh-rewrote step2 drafts, but `architect-summary.md` still used generic conditional shard-gap wording instead of exact current-run typed shard status.
- FTGO completed collect 16/16 and wrote valid step2 markdown under `draft_final_root`, but duplicated `overview.md`, `summary.md`, and `architect-summary.md` in `write_root`, causing a write-set failure even though the final draft files were otherwise valid.

#### Plan
- [x] Add one provider-authored `draft_artifact_enrichment_shard_status_cleanup` retry for fresh step2 markdown rejected only by generic conditional shard-gap wording.
- [x] Add one provider-authored `draft_artifact_enrichment_write_set_cleanup` retry that deletes only byte-identical referenced markdown duplicates from `write_root` while keeping arbitrary extra writes terminal.
- [x] Keep success provider-authored only: no ACP-side synthesis of step2 draft markdown or manifest shape.
- [x] Add providercommon and promptcontract regression tests for both retry modes.
- [x] Stabilize the existing silent/no-fresh collect-pair retry test window so package tests are not timing-sensitive on loaded trusted hosts.
- [x] Update live E2E runbook, pipeline spec, architecture and testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

#### Acceptance
- [x] Fresh step2 markdown with generic shard-gap caveats gets one provider-authored shard-status cleanup retry.
- [x] Byte-identical draft markdown duplicates in `write_root` get one cleanup retry that deletes only those misplaced duplicates.
- [x] `extra.md`/unreferenced write-set violations remain `runtime_contract_failed`.
- [ ] Latest strict medium `claude-code` rerun no longer fails from PostHog generic shard wording or FTGO duplicated write_root drafts.

---

### Live E2E Final Index Document ID Collision

#### Context
- 2026-06-24 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260624T101459Z` preserved selected-provider totals and no legacy mixed quality-gate artifacts, but still failed both backend slots.
- PostHog reached `init.step4.proposals` and correctly failed scaffold/noop proposal enrichment: `proposal.md` and `changelog.md` still contained bootstrap placeholder content.
- FTGO completed collect `16/16` and step2 enrichment produced readable, decision-ready as-is markdown with exact shard completeness, but staged final assembly failed before writing `final-run-index.json`: two distinct shard documents reused provider id `doc.overview`, causing `canonical_documents[12].id must be unique`.

#### Plan
- [x] Keep unique provider-authored `manifest.Documents[*].id` values when they identify one canonical path.
- [x] Remap repeated provider-authored document ids across distinct `canonical_path` values to deterministic canonical-path-derived ids before final index validation.
- [x] Remap citation `document_ids` into the same canonical document namespace.
- [x] Add docflow regression coverage for repeated `doc.overview` across distinct canonical paths.
- [x] Update pipeline/runbook/testing docs.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

#### Acceptance
- [x] Duplicate source document ids no longer produce duplicate `final-run-index.json.canonical_documents[].id`.
- [x] Citation index references the remapped unique document ids.
- [ ] Latest strict medium `claude-code` rerun no longer fails FTGO at final-index assembly.

---

### Live E2E DoD Precheck Timeout

#### Context
- 2026-06-26 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260625T224145Z` completed the PostHog backend/frontend slot, then the FTGO profile stalled before runtime in `make contracts test lint build`.
- Existing harness hard timeout covered provider pipelines and UI dependency/browser prechecks, but DoD precheck still ran as an unbounded direct shell command.
- This left the matrix profile `running` until manual interrupt and produced no useful headless taskrun evidence because runtime had not started.

#### Plan
- [x] Add `ACP_LIVE_PRECHECK_DOD_TIMEOUT_SEC` and run `make contracts test lint build` through the existing process-group precheck timeout helper.
- [x] Keep terminal classification as `precheck_failed`, with `[precheck-timeout]` evidence in `precheck-make.log`.
- [x] Keep DoD timeout separate from provider runtime timeout and artifact/UX quality.
- [x] Add a script regression test for a hung DoD precheck.
- [x] Update runbook, pipeline spec, architecture and testing strategy.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

#### Acceptance
- [x] A hung `make contracts test lint build` no longer leaves a child batch/profile indefinitely running.
- [x] Reports retain `execution_report_<batch-id>.md`; legacy mixed quality-gate artifacts stay absent.
- [ ] Latest strict medium rerun reaches runtime or fails with a bounded, classified precheck/provider/runtime reason.

---

### Live E2E Step2 Enrichment And Ask Smoke Cleanup

#### Context
- 2026-06-30 strict medium `claude-code` rerun `regres-long-posthog-ftgo-20260630T003528Z` preserved the new execution/UX/artifact separation: `matrix_result` used selected provider totals only, `execution_report_<batch-id>.md` was produced, and no legacy mixed `quality_report_*`/`quality_gates_failed` release gate evidence was observed.
- PostHog backend still failed in `init.step2.asis_docs`: after multiple no-op/scaffold enrichment attempts, compact step2 retry finally fresh-wrote markdown, but copied sampled shard prose with unbalanced inline backticks; the syntax cleanup retry then stalled before valid artifacts and the run correctly ended as `runtime_contract_failed`.
- FTGO backend passed both init and refresh after focused collect/draft repairs, but frontend failed in default Ask smoke: the async `qa.ask` run remained running after the fixed 120s Playwright poll, was classified as generic `playwright_failed`, and teardown left an orphan `claude` provider process until manually killed.

#### Plan
- [x] Keep `release_verdict_*` execution-only and leave artifact/UX quality as separate SWE report inputs.
- [x] Add bounded QA smoke polling with `ACP_UI_QA_POLL_TIMEOUT_SEC` and `ACTIVE_RUN_TIMEOUT` marker classification.
- [x] Best-effort cancel active frontend QA runs after Playwright failure before `acp serve` teardown, preventing orphan provider processes.
- [x] Tighten step2 markdown syntax retry prompt: rewrite every referenced markdown target in one bounded command, remove sampled shard prose snippets, keep exact typed shard completeness, omit stale downstream-index claims, and avoid generic shard-gap wording.
- [x] Tighten compact step2 retry prompt so evidence bullets are path plus paraphrased architecture signal only, not copied first paragraphs.
- [x] Run full DoD.
- [ ] Commit and rerun strict medium live E2E.

#### Acceptance
- [x] QA smoke timeout is `active_run_timeout`, not generic `playwright_failed`.
- [x] Failed QA smoke does not leave an unmanaged provider process after harness teardown.
- [x] Step2 syntax cleanup is provider-authored and does not introduce deterministic ACP-side artifact synthesis.
- [ ] Latest strict medium `claude-code` rerun no longer fails PostHog step2 on malformed markdown/no-op enrichment.
- [ ] Latest strict medium frontend Ask smoke either succeeds or fails with bounded `active_run_timeout` plus cancellation evidence.

---

### Runtime Stability To Excellent

#### Context
- Current `smoke tiny` live evidence can reach `strict PASS` and `artifact_quality_status=passed`, but still receive `Needs review` because good artifacts were obtained through repair/stall pressure.
- Goal is not to weaken acceptance: `Excellent` still requires runtime contract pass, artifact-quality pass, no `runtime_quality.*`, no `analysis:*`, no repair/stall pressure, and applicable frontend evidence.
- Product and live layers remain black-box separated: ProvenArch improves generic prompt/validation contracts; live E2E reports public evidence only.

#### Plan
- [x] Strengthen normal `step0.constitution`, `step2.asis_docs`, and `step4.proposals` prompts so the same provider turn completes evidence-backed drafts before successful exit.
- [x] Make `step4.proposals` high/medium actionability markdown-safe and bullet-only: exact Finding ID, copied Severity value, Affected surface/path, Recommended operator action with concrete verb, Residual gap; reject tables and generic inspect/review/decide-only proposals.
- [x] Add live E2E `Excellent blockers by step` diagnostics from public taskrun logs/raw metadata and propagate additive `excellent_blockers_by_step` into batch/matrix JSON reports.
- [x] Update `docs/spec/PIPELINE_SPEC.md` and `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.
- [x] Run targeted tests and full DoD: `make contracts`, `make test`, `make lint`, `make build`.
- [x] Rerun diagnostic `smoke tiny` on `codex-code`.

#### Acceptance
- [x] Valid step4 actionable bullets pass validation; malformed tables/inline-code and generic review-only proposals fail before promotion.
- [x] Live reports identify which `step_id` caused repair/stall pressure that blocks `Excellent`.
- [x] No provider/profile/matrix/repo-specific logic is added to ProvenArch product code.

#### Live validation notes
- 2026-07-04: `smoke-tiny-bank-stability2-20260704T095509Z` failed with `init.step4.proposals` `low_actionability`; fixed by requiring copied high/medium Severity values and clearer validation hints.
- 2026-07-04: `smoke-tiny-bank-stability3-20260704T104303Z` still failed before promotion with `init.step4.proposals` `finding_linkage`; reports correctly identify step-level `repair_exhausted`, `repair_heavy`, and `stall_pressure`. Added exact nested findings path/backticked-ID guidance as follow-up hardening. `Excellent` remains blocked until a subsequent live run avoids focused repair/stall and passes runtime contract.
- 2026-07-04: `smoke-tiny-bank-runtime-pass-20260704T121128Z` reached non-release matrix `PASS` with `runtime_contract_status=passed`, `artifact_quality_status=passed`, `quality_gates_failed=0`, and `artifact_quality_failed=0`. Verdict remains `Needs review` because `repair_attempts=5`, `focused_repairs=5`, `stall_count=12`, and `Excellent Blockers By Step` still point at init `step0`/`step1`/`step2`/`step3`/`step4` plus refresh `step2`/`step4`. The remaining P1 blocker is same-turn provider completion: current artifacts are useful after focused repair, but the happy path still relies on repair/stall pressure.

#### Follow-up Plan: Runtime Contract PASS
- [x] Treat synthetic current-run finding placeholders (`no-current-run-finding-id`, `no structured current-run finding ID`, `finding unavailable`) as explicit `step4.proposals` draft validation failures when current-run `findings.md` contains exact IDs.
- [x] Add a live-shaped regression where nested staged `findings.md` has backticked IDs but proposal/changelog use a synthetic placeholder.
- [x] Keep live E2E classification black-box by mapping the public validation text to `finding_linkage` without importing product internals.
- [x] Sync pipeline spec and live runbook for the stricter finding-linkage contract.
- [x] Rerun `smoke tiny` on `codex-code`; observed intermediate result is `runtime_contract_status=passed`, `artifact_quality_status=passed`, strict `PASS`, and `Needs review` because repair/stall pressure remains.

---
