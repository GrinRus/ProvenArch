# ProvenArch Code Quality Audit

> Archived 2026-09-05 from `docs/CODE_AUDIT_2026-07-10.md`. Findings and line references describe the recorded July baseline, not the current tree. Epic 19 remediation status belongs to the [canonical matrix](../../STAKEHOLDER_DOC.md); the original audit evidence is preserved below.

Дата: 2026-07-10

Baseline: 122e4c9b5a91b29e243677c0dac0fe2ebfca226b

Режим: read-only quality audit; исправления production-кода, API, schemas и contracts не выполнялись.

## Executive summary

Итоговый verdict: **deterministic baseline зелёный, но reliability/release readiness требует обязательных P1-доработок**.

- Blocker не найден.
- Основной реестр: **29 findings** — 19 BUG, 3 REF, 7 QUAL.
- Impact: **19 Major**, 10 Normal.
- Confidence: **27 Confirmed**, 2 High.
- Priority: **19 P1**, 10 P2.
- Confirmed dead-code register: **13 записей / 60 identifiers или assignments** — 46 Go, 10 TypeScript/React, 4 shell.
- Наиболее существенные риски: неатомарные promotion/persistence, lifecycle активных runs, unsynchronized service replacement, run-scoped UI state races, неполная проверка collect evidence, release/generated-artifact false-green gates.

Положительный baseline:

- make contracts, make test и make lint прошли.
- Go suite и go test -race -shuffle=on -count=1 ./... прошли без race markers.
- Backend statement coverage: 70.8%.
- TypeScript typecheck и 93 Vitest tests прошли.
- Семь deterministic mock Playwright scenarios прошли на desktop и mobile без console errors или page-level overflow.
- 230 Python tests, Python compileall и bash syntax checks прошли.

Области вне quality-scope не оценивались.

## Scope и метод

| Поток | Проверенная область | Результат |
|---|---|---|
| Static | reachability, dead code, symbol/dependency wiring, schemas/spec drift, hotspots | 4 findings, 8 dead-code groups |
| Backend | API, orchestrator, runtime, workspace, init/refresh/cancel/restart, persistence/recovery | 7 findings |
| UI | React state, run scoping, async requests, accessibility, desktop/mobile mock QA | 10 findings, 2 dead-code groups |
| Tooling | Makefile, CI, shell/Python harness, false-green tests, generated UI drift | 8 findings, 3 dead-code groups |

В основной реестр включены только Confirmed и High findings. Автоматические сигналы использовались как кандидаты: ограничения go vet, race detector и Staticcheck учитывались по официальной документации: [Go vet](https://go.dev/cmd/vet/?m=old), [Go race detector](https://go.dev/doc/articles/race_detector), [Staticcheck U1000](https://staticcheck.dev/changes/2020.2/). Coverage использовался как карта непроверенных путей, а не как самостоятельная оценка качества: [Vitest coverage](https://vitest.dev/guide/coverage.html).

Не запускались live matrix/providers, fuzzing, dependency advisory/security scans и специальные boundary-проверки. Raw logs и generated inputs в отчёт не включены.

## Статус проверок

| Проверка | Статус | Примечание |
|---|---|---|
| Baseline HEAD / clean start | PASS | exact commit; до аудита tree был clean |
| ast-index | PASS с limitation | global internal pass встретил ENOSPC; module-scoped passes и Staticcheck закрыли reachability |
| Staticcheck 2026.1 | PASS как анализатор | подтвердил 46 unreachable production-Go identifiers |
| go vet ./... | PASS для production scope | пять test-only loop-capture diagnostics исключены |
| go mod tidy -diff | PASS | drift отсутствует |
| TypeScript typecheck | PASS | normal project configuration |
| TypeScript noUnusedLocals/noUnusedParameters | FAIL ожидаемо | три static UI identifiers из DEAD-008 |
| go test ./... | PASS | initial ENOSPC был host limitation; повтор прошёл |
| Go race/shuffle | PASS | race markers отсутствуют |
| Go coverage | PASS | total 70.8%; fake runtime 47.4% — самый низкий package result |
| Targeted lifecycle/recovery tests | PASS | существующие tests не покрывают перечисленные failure-injection paths |
| Vitest | PASS | 6 files / 93 tests |
| Mock Playwright | PASS | 7/7 scenarios, desktop 1440×980 и mobile 390×900 |
| V8 UI coverage | NOT RUN | @vitest/coverage-v8 отсутствует в locked dependencies |
| Python unittest | PASS | 230 tests |
| Python compileall | PASS | cache вынесен в /tmp |
| bash -n | PASS | 18 scripts |
| ShellCheck 0.11.0 | FAIL | actionable dead/unused shell code; trap callbacks требуют точечных suppressions |
| make contracts | PASS | validator toolchain при этом разрешается как mutable latest |
| make test | PASS | Go + 230 Python + 93 Vitest |
| make lint | PASS | не включает ShellCheck |
| temp exact-commit make build | PASS | current worktree не изменялся |
| generated UI comparison | FAIL | 109 changed files / 185 path-level differences |

## Приоритизированный реестр

| ID | Категория | Impact | Confidence | Priority | Effort | Кратко |
|---|---|---|---|---|---|---|
| BUG-001 | BUG | Major | Confirmed | P1 | L | Promotion меняет canonical tree без transaction/rollback |
| BUG-002 | BUG | Major | Confirmed | P1 | M | Shutdown serve не владеет lifecycle активных runs |
| BUG-003 | BUG | Major | Confirmed | P1 | S | Panic async runner завершает API process |
| BUG-004 | BUG | Major | Confirmed | P1 | M | Unpinned git_url refresh остаётся на старом checkout |
| BUG-005 | BUG | Major | Confirmed | P1 | M | Recovery state пишется неатомарно, history errors теряются |
| BUG-006 | BUG | Major | Confirmed | P1 | M | Runner reselection заменяет Service без quiesce |
| BUG-007 | BUG | Major | Confirmed | P1 | S | API handlers читают mutable service без синхронизации |
| BUG-008 | BUG | Major | Confirmed | P1 | M | Collect принимает evidence-empty shard pack |
| BUG-009 | BUG | Major | Confirmed | P1 | M | Reverse citation.document_ids membership не проверяется |
| BUG-010 | BUG | Normal | Confirmed | P2 | M | Step 1 card enrichment недостижим |
| BUG-011 | BUG | Major | Confirmed | P1 | M | Historical run показывает текущий canonical content |
| BUG-012 | BUG | Major | Confirmed | P1 | M | Быстрый run switch смешивает status/artifacts/logs |
| BUG-013 | BUG | Major | Confirmed | P1 | M | Accepted mutation показывается failed из-за follow-up GET |
| BUG-014 | BUG | Major | Confirmed | P1 | M | Artifact preview и Git diff принимают stale response |
| BUG-015 | BUG | Major | Confirmed | P1 | M | Async Q&A теряет accepted run и stale history selection |
| BUG-016 | BUG | Major | Confirmed | P1 | M | Form edits во время save помечаются saved/valid |
| BUG-017 | BUG | Major | Confirmed | P1 | M | Charter loader затирает новый dirty text |
| BUG-018 | BUG | Major | Confirmed | P1 | M | Embedded UI не имеет freshness/reproducibility gate |
| BUG-019 | BUG | Major | High | P1 | M | Release workflow не проверяет composite verdict |
| REF-001 | REF | Normal | Confirmed | P2 | M | Refresh semantic guard не участвует в pipeline |
| REF-002 | REF | Major | Confirmed | P1 | M | Contract validator dependencies разрешаются как latest |
| REF-003 | REF | Normal | Confirmed | P2 | M | Python toolchain не закреплён |
| QUAL-001 | QUAL | Normal | Confirmed | P2 | M | TabNav реализует неполный ARIA tabs pattern |
| QUAL-002 | QUAL | Normal | Confirmed | P2 | M | LocalPathCombobox не имеет standard keyboard control |
| QUAL-003 | QUAL | Normal | Confirmed | P2 | S | Async statuses/errors не объявляются assistive tech |
| QUAL-004 | QUAL | Normal | Confirmed | P2 | S | PR CI не запускает repository lint target |
| QUAL-005 | QUAL | Normal | Confirmed | P2 | S | make lint не покрывает production shell scripts |
| QUAL-006 | QUAL | Normal | Confirmed | P2 | S | smoke-api не проверяет logs endpoint |
| QUAL-007 | QUAL | Normal | High | P2 | M | Deterministic mock Playwright не оркестрируется CI |

## Подробные findings

### BUG-001 — Promotion не имеет транзакционной границы и rollback

- **Location:** internal/orchestrator/docflow_promotion.go:32, :114, :142
- **Сценарий:** после validator PASS во время обычной promotion заканчивается место, теряется доступ к target либо падает model/diagram materialization.
- **Expected:** canonical set переключается целиком; failure оставляет coherent previous или coherent incomplete generation.
- **Actual:** документы перезаписываются и удаляются in-place, model directories удаляются до rebuild; failure оставляет смесь поколений.
- **Evidence:** между write/delete/rebuild стадиями отсутствуют staging transaction, journal, backup и rollback.
- **Root cause:** promotion реализована как последовательная мутация live canonical tree.
- **Recommendation:** собрать полный managed snapshot в sibling staging tree, валидировать и атомарно переключать; для платформ без directory exchange вести rollback journal.
- **Acceptance test:** fault injection на каждом N-м write/remove/model/diagram step; после failure canonical workspace совпадает ровно с одним поколением.

### BUG-002 — Остановка serve не владеет lifecycle активных runs

- **Location:** cmd/acp/main.go:226; internal/api/server.go:1157; internal/runtime/providercommon/procgroup_unix.go:15
- **Сценарий:** operator или service manager посылает SIGINT/SIGTERM во время async headless run.
- **Expected:** server context отменяет run, завершает provider process group и сохраняет terminal status.
- **Actual:** serve и API runs используют background contexts; bounded shutdown orchestration отсутствует, history может остаться running.
- **Evidence:** signal-aware context есть у synchronous run, но Server.Serve не закрывает orchestrator service.
- **Root cause:** async run context не связан с lifetime сервера.
- **Recommendation:** signal-aware server context плюс bounded Service.Close/Shutdown для active/pending runs и provider cleanup.
- **Acceptance test:** blocking provider + server cancellation/SIGTERM; server завершается bounded, provider отсутствует, history terminal, post-shutdown writes отсутствуют.

### BUG-003 — Panic в async runner завершает API process

- **Location:** internal/orchestrator/service_runs.go:366; internal/orchestrator/orchestrator.go:334
- **Сценарий:** runner/orchestrator паникует внутри API-started init, refresh или QA run.
- **Expected:** один run получает failed/internal_failure, API остаётся доступным, active slot освобождается.
- **Actual:** runWithID terminalize-ит history, повторно вызывает panic, а внешняя async goroutine не делает recover/guaranteed cleanup.
- **Evidence:** finishAsyncRun вызывается только после обычного return.
- **Root cause:** synchronous panic propagation переиспользована без async isolation boundary.
- **Recommendation:** recover на внешней goroutine и finishAsyncRun через defer; synchronous Run может сохранить текущую semantics.
- **Acceptance test:** panic-runner через StartAsyncRun; API жив, history terminal, slot освобождён, pending run продолжен.

### BUG-004 — git_url без ref остаётся на старом checkout после fetch

- **Location:** internal/workspace/resolver.go:249
- **Сценарий:** remote default branch получает новый commit, затем запускается refresh unpinned git_url source.
- **Expected:** повторное resolve анализирует новый remote default commit.
- **Actual:** cache выполняет fetch --prune, но checkout/reset выполняется только при non-empty ref.
- **Evidence:** empty-ref path не двигает HEAD после fetch.
- **Root cause:** fetch ошибочно считается обновлением working tree.
- **Recommendation:** resolve remote default HEAD, reset ACP-owned cache на fetched SHA и сохранять resolved SHA в evidence.
- **Acceptance test:** второй resolve после нового remote commit обновляет cached HEAD и читаемый файл.

### BUG-005 — Critical recovery state записывается неатомарно

- **Location:** internal/workspace/fs.go:79; internal/orchestrator/service_runs.go:504; restart_reconcile.go:36; sharding_artifacts.go:135
- **Сценарий:** disk-full, crash или I/O failure во время history/shard-summary update.
- **Expected:** restart читает последнюю целую snapshot либо выдаёт явный actionable blocker.
- **Actual:** os.WriteFile truncates live file; history write error игнорируется; malformed history становится empty; corrupt summary блокирует resume.
- **Evidence:** отсутствуют temp+fsync+rename/last-good copy и error propagation.
- **Root cause:** общий non-atomic Root.WriteFile используется для recovery-critical state.
- **Recommendation:** atomic persistence primitive для history, summaries и checkpoints с surfaced diagnostics.
- **Acceptance test:** fault injection после truncate/до rename; restart читает last-good snapshot или явный blocker, checkpoint replayable.

### BUG-006 — Runner reselection заменяет Service без quiesce

- **Location:** internal/api/server.go:251; internal/api/onboarding.go:119
- **Сценарий:** resumed run активен, затем onboarding сохраняет другой runner.
- **Expected:** изменение отклоняется до terminal либо coordinated migration сохраняет ownership.
- **Actual:** service заменяется без cancel/transfer/reconcile; старый goroutine продолжает писать, новый service держит stale history; direct mode сообщает config без effective rebuild.
- **Evidence:** replacement не проверяет active/pending registry.
- **Root cause:** runtime configuration и orchestration lifecycle заменяются независимо.
- **Recommendation:** conflict при active/pending либо coordinated shutdown/migration; direct mode должен сохранять reported/effective equality.
- **Acceptance test:** blocking resumed run + runtime selection даёт conflict или coherent migration; следующий run использует новый provider.

### BUG-007 — API handlers читают mutable service без синхронизации

- **Location:** internal/api/server.go:1249; competing writes server.go:216, :251
- **Сценарий:** UI polling совпадает со сменой workspace/runner из другой вкладки.
- **Expected:** request использует coherent session snapshot либо получает 428/409.
- **Actual:** writers используют mutex, но handlers читают s.service напрямую; возможны race, TOCTOU и nil dereference.
- **Evidence:** runs/status/artifacts/logs/cancel/runtime-profile/QA handlers обходят synchronized accessor.
- **Root cause:** middleware readiness check и handler execution не связаны одной session generation.
- **Recommendation:** immutable session snapshot под RLock и coordinated swap.
- **Acceptance test:** concurrent polling и onboarding mutation под race detector без race/panic; каждый response принадлежит одной generation.

### BUG-008 — Collect принимает evidence-empty shard pack

- **Location:** schemas/shard-pack-manifest.schema.json:62, :68; internal/contracts/docflow.go:306, :420; artifactquality/canonicalize.go:17; collect_bootstrap.go:225; runtime_task_apply.go:31, :45
- **Сценарий:** provider возвращает формально корректный manifest с empty documents/citations и разреженным semantic snapshot.
- **Expected:** collect gate отклоняет пакет и запускает focused repair до checkpoint.
- **Actual:** arrays не имеют minItems; validators проверяют только существующие элементы; manifest может checkpoint/apply.
- **Evidence:** ManifestAssessment.Rich не участвует в success gate, bootstrap heuristic не является общей completeness validation.
- **Root cause:** корректность формы принята за достаточную evidence cardinality.
- **Recommendation:** потребовать non-empty documents/citations и citation_ids для authored docs; синхронизировать schema, validators, fixtures и spec.
- **Acceptance test:** negative contract/provider-engine tests отклоняют empty evidence; succeeded checkpoint/model apply отсутствуют.

### BUG-009 — Reverse citation.document_ids membership не проверяется

- **Location:** internal/contracts/docflow.go:306, :420; internal/orchestrator/docflow.go:1549, :1709; docs/spec/PIPELINE_SPEC.md:256, :286
- **Сценарий:** document ссылается на citation, но citation содержит typo/unknown document ID.
- **Expected:** collect отклоняется или repair-ится.
- **Actual:** document-to-citation side проверяется, citation-to-document membership — нет; unknown ID переживает remap/staged validation.
- **Evidence:** validateCitationSet требует только non-empty DocumentIDs.
- **Root cause:** двунаправленный контракт реализован асимметрично.
- **Recommendation:** membership и symmetry validation до и после remap.
- **Acceptance test:** unknown/asymmetric bindings fail; valid duplicate-ID remap cases остаются зелёными.

### BUG-010 — Step 1 card enrichment недостижим

- **Location:** internal/orchestrator/step_handlers.go:185, :198; semantic_cards.go:218, :258, :293; docs/spec/PIPELINE_SPEC.md:301
- **Сценарий:** refresh добавляет entities/findings/questions, пользователь открывает canonical domain/team cards.
- **Expected:** Step 1 идемпотентно добавляет Derived section с evidence refs.
- **Actual:** active handler возвращается без enrichment; весь enrichment call tree не имеет production references.
- **Evidence:** Staticcheck U1000 и ast-index refs/usages подтверждают недостижимость при обязательном spec behavior.
- **Root cause:** pipeline refactor отключил path без синхронизации контракта.
- **Recommendation:** восстановить call в contract-safe точке либо отдельным contract decision удалить behavior и dead cluster.
- **Acceptance test:** fake init/refresh integration создаёт ровно одну Derived section с evidence refs.

### REF-001 — Refresh semantic guard не участвует в pipeline

- **Location:** internal/orchestrator/semantic_utils.go:202, :211, :228, :241, :253, :267; docflow.go:894, :908; docs/ARCHITECTURE.md:137
- **Сценарий:** refresh provider добавляет runtime metadata или off-topic semantic entity с формальной repo evidence.
- **Expected:** заявленный guard фильтрует либо явно маркирует placeholder/off-topic material.
- **Actual:** normalisation делает только trim/dedupe/remap; guard/filter helpers недостижимы.
- **Evidence:** Staticcheck U1000 и нулевые refs/usages для шести helpers.
- **Root cause:** незавершённый refactor оставил документацию и implementation в разных состояниях.
- **Recommendation:** сформулировать общий refresh policy; подключить guard с diagnostics/tests либо удалить обещание и dead code синхронным docs/ADR изменением.
- **Acceptance test:** refresh маркирует/фильтрует runtime metadata, сохраняет legitimate same-domain entity и явно отличается от init.

### BUG-011 — Historical run открывает текущий canonical content

- **Location:** ui/src/hooks/useRunArtifacts.ts:68; ui/src/lib/appContracts.ts:132
- **Сценарий:** оператор выбирает старый completed run после публикации нового run с теми же canonical paths.
- **Expected:** preview, coverage и questions принадлежат выбранному run.
- **Actual:** UI отбрасывает staged_path и читает текущие reports paths; old run показывает new content.
- **Evidence:** FinalRunIndexDocument не моделирует staged_path; mapping сохраняет только canonical_path, coverage читает stable paths.
- **Root cause:** artifact identity не включает run-scoped read path.
- **Recommendation:** разделить canonical label и run-scoped staged_path; historical review читать из staged snapshot.
- **Acceptance test:** два run с одинаковым canonical path и разным content; выбор старого не запрашивает canonical file.

### BUG-012 — Быстрый выбор run смешивает status, artifacts и logs

- **Location:** ui/src/hooks/useRunActions.ts:77; useRunLogs.ts:88; useRunArtifacts.ts:58
- **Сценарий:** оператор быстро выбирает run A, затем B; responses A задерживаются.
- **Expected:** последнее действие выигрывает, все panels показывают B.
- **Actual:** late A responses без generation/run check записываются в shared state при selected B.
- **Evidence:** loaders не используют AbortController/request generation и не сверяют response run ID.
- **Root cause:** async state global, а не request/run scoped.
- **Recommendation:** abort/token per selection и reducer actions с обязательным runId.
- **Acceptance test:** deferred A resolve после B; status/artifacts/logs/Activity остаются строго B.

### BUG-013 — Accepted mutation ошибочно показывается failed

- **Location:** ui/src/hooks/useRunActions.ts:155, :208
- **Сценарий:** start/cancel POST принят, но следующий status/list/log GET временно падает.
- **Expected:** UI сохраняет queued/cancel requested и отдельно сообщает reconciliation failure.
- **Actual:** общий catch сообщает failed to start/cancel, провоцируя duplicate action.
- **Evidence:** mutation и follow-up reads находятся в одном try/catch.
- **Root cause:** mutation acknowledgement не отделён от reconciliation.
- **Recommendation:** commit accepted state сразу после POST; follow-up reads выполнять recoverable background path.
- **Acceptance test:** POST success + first GET failure сохраняет run ID и восстанавливает polling без второго POST.

### BUG-014 — Artifact preview и Git diff принимают stale response

- **Location:** ui/src/hooks/useRunArtifacts.ts:107; ui/src/hooks/useGitDiff.ts:11
- **Сценарий:** оператор быстро открывает artifact/diff A, затем B.
- **Expected:** selected key и payload всегда относятся к B.
- **Actual:** поздний A response перезаписывает content/diff под header B.
- **Evidence:** setters после await не проверяют request key/sequence.
- **Root cause:** key и payload хранятся раздельно без atomic request identity.
- **Recommendation:** state {requestKey,status,data}, abort previous и key check before commit.
- **Acceptance test:** B resolve первым, A последним; header и content остаются B.

### BUG-015 — Async Q&A теряет accepted run и stale history selection

- **Location:** ui/src/components/StagePanels.tsx:4689, :4726
- **Сценарий:** Q&A POST принят, первый detail GET падает; либо selection A→B получает late A.
- **Expected:** provisional accepted run появляется сразу, polling продолжается, last selection wins.
- **Actual:** qaRun остаётся null до successful detail, polling не стартует; late A может заменить B.
- **Evidence:** provisional record создаётся только после getQARun; selection handler без generation guard.
- **Root cause:** lifecycle зависит от немедленного detail response и не request-scoped.
- **Recommendation:** materialize provisional record из start response, poll по ID, добавить selection token.
- **Acceptance test:** failed-then-success detail восстанавливает тот же run; deferred A не меняет selected B answer.

### BUG-016 — Изменения формы во время save помечаются saved и valid

- **Location:** ui/src/hooks/useManifestEditor.ts:174, :190
- **Сценарий:** пользователь нажимает Save и меняет repo/manifest до завершения request.
- **Expected:** новый draft остаётся dirty и требует validation.
- **Actual:** completion старого request устанавливает validation и setupDirty=false для изменённого UI.
- **Evidence:** fields editable; completion без revision check безусловно сбрасывает dirty.
- **Root cause:** save snapshot не связан с form revision.
- **Recommendation:** revision counter/snapshot comparison; stale completion не меняет dirty/validation.
- **Acceptance test:** изменить URL во время deferred save; второй save отправляет новый URL, validation остаётся stale до него.

### BUG-017 — Charter editor затирает новый dirty text

- **Location:** ui/src/hooks/useBaselineEditor.ts:65; ui/src/App.tsx:216
- **Сценарий:** пользователь выбирает artifact и начинает редактировать после первого load.
- **Expected:** один load на selection; late responses не трогают dirty draft.
- **Actual:** selection handler и App effect оба запускают load; второй/старый response перезаписывает content.
- **Evidence:** два call sites, loader всегда вызывает setSelectedEditorContent.
- **Root cause:** два владельца загрузки и нет per-path request/draft sequencing.
- **Recommendation:** один loading effect, request token и dirty drafts per path.
- **Acceptance test:** две deferred loads одного path; late response не меняет введённый после первой текст.

### QUAL-001 — TabNav реализует неполный ARIA tabs pattern

- **Location:** ui/src/components/TabNav.tsx:19
- **Сценарий:** keyboard/screen-reader пользователь переключает tabs.
- **Expected:** roving tabIndex, Arrow/Home/End и tab↔tabpanel relationship.
- **Actual:** все tabs в Tab order; arrows не работают; aria-controls/labelledby/tabpanel отсутствуют.
- **Evidence:** component задаёт только tablist/tab/aria-selected/click.
- **Root cause:** visual buttons получили tab roles без полного interaction contract.
- **Recommendation:** reusable WAI-ARIA tab controller и stable panel IDs.
- **Acceptance test:** active tab единственный tabIndex=0; arrows/Home/End управляют focus/selection; panel связан с tab.

### QUAL-002 — Path combobox нельзя управлять стандартными клавишами

- **Location:** ui/src/components/LocalPathCombobox.tsx:82
- **Сценарий:** keyboard пользователь выбирает suggested workspace/repo path.
- **Expected:** ArrowDown/Up, Enter, Escape и aria-activedescendant.
- **Actual:** input не имеет key handlers/active option; options доступны pointer или последовательным Tab.
- **Evidence:** есть focus/blur/click state, но нет composite keyboard state.
- **Root cause:** ARIA roles добавлены без keyboard contract.
- **Recommendation:** active-index state, activedescendant и Arrow/Enter/Escape; options убрать из отдельного Tab order.
- **Acceptance test:** полный keyboard-only selection и корректное закрытие Escape.

### QUAL-003 — Async errors/statuses не объявляются assistive tech

- **Location:** ui/src/App.tsx:1115; ui/src/components/OnboardingShell.tsx:133
- **Сценарий:** screen-reader пользователь получает validation/save/start failure без focus change.
- **Expected:** error объявляется, field связан с diagnostic.
- **Actual:** большинство messages обычные p/div; alert/live semantics есть только в отдельных surfaces.
- **Evidence:** UI содержит лишь один role=alert и один aria-live при множестве async status blocks.
- **Root cause:** status presentation не централизована как accessible primitive.
- **Recommendation:** error summary role=alert, success/progress aria-live=polite, inline diagnostics через describedby/invalid.
- **Acceptance test:** async failure создаёт accessible alert; invalid field связан и объявляется без ручной навигации.

### BUG-018 — Versioned embedded UI не имеет freshness/reproducibility gate

- **Location:** .github/workflows/ui.yml:38; Makefile:42; .goreleaser.yml:5
- **Сценарий:** ui/src меняется без обновления tracked internal/api/ui_dist.
- **Expected:** PR падает; normal Go build и release build exact commit встраивают одинаковый UI.
- **Actual:** UI workflow строит только ignored ui/dist; Go build использует tracked bundle, GoReleaser пересобирает его.
- **Evidence:** exact-commit temp build дал 109 changed files / 185 path-level differences; normalized main JS/CSS семантически совпали, что указывает на chunk-hash churn.
- **Root cause:** generated surface поддерживается manual checklist, deterministic drift gate отсутствует.
- **Recommendation:** стабилизировать chunk generation и добавить clean-build job с git diff --exit-code для embed.
- **Acceptance test:** два independent clean builds имеют одинаковые manifests/digests; stale embed PR падает.

### BUG-019 — Release workflow не проверяет composite verdict

- **Location:** .github/workflows/release.yml:45, :54
- **Сценарий:** tag v* создаётся без accepted release verdict и SWE assessments.
- **Expected:** publication блокируется до успешного offline verify-release-verdict.
- **Actual:** после contracts/test/lint workflow сразу вызывает GoReleaser; evidence/verifier отсутствуют.
- **Evidence:** publication path не имеет evidence input или verifier call; external environment rules из repo не проверяемы.
- **Root cause:** обязательный pre-tag gate остаётся out-of-band.
- **Recommendation:** до write permissions загрузить canonical evidence и запустить verifier либо блокировать release tag без verified workflow.
- **Acceptance test:** missing/FAIL/mismatched SWE evidence не запускает GoReleaser; полный accepted комплект запускает.

### REF-002 — Contract validator dependencies разрешаются как mutable latest

- **Location:** Makefile:17; .github/workflows/contracts.yml:23
- **Сценарий:** один commit проверяется до и после нового release ajv-cli/ajv-formats/js-yaml.
- **Expected:** один locked validator toolchain.
- **Actual:** npm exec package names без versions/lockfile каждый раз разрешают registry latest.
- **Evidence:** contract run использовал transient package set; exact versions/integrity metadata отсутствуют.
- **Root cause:** contract tooling не оформлен как versioned dependency bundle.
- **Recommendation:** locked tools package или exact versions с integrity; запуск через installed lockfile.
- **Acceptance test:** contract gate работает offline после npm ci; tool upgrade требует explicit lockfile diff.

### QUAL-004 — PR CI не запускает repository lint target

- **Location:** .github/workflows/backend.yml:27; ui.yml:32; release.yml:51
- **Сценарий:** PR содержит компилируемый, но неформатированный Go-код.
- **Expected:** required PR CI выполняет canonical make lint.
- **Actual:** PR workflows запускают tests/build/typecheck; полный lint есть только после release tag.
- **Evidence:** go test/build не требуют gofmt-clean source, а PR path не вызывает Makefile lint target.
- **Root cause:** lint policy разделена между local checklist и tag-only workflow.
- **Recommendation:** required PR lint job через make lint.
- **Acceptance test:** intentionally unformatted Go file делает PR check красным.

### QUAL-005 — make lint не покрывает production shell scripts

- **Location:** Makefile:28
- **Сценарий:** batch/harness script содержит unused variable или dead helper.
- **Expected:** canonical lint обнаруживает shell diagnostics.
- **Actual:** make lint проверяет gofmt и TypeScript; target PASS при non-zero ShellCheck.
- **Evidence:** подтверждены unused frontend status assignments и два dead helpers; trap callbacks требуют точечных suppressions.
- **Root cause:** ShellCheck не включён в Makefile/workflow.
- **Recommendation:** добавить ShellCheck с documented suppressions для indirect trap callbacks и valid export idiom.
- **Acceptance test:** current tree проходит без actionable warnings; новый unused shell variable ломает make lint.

### REF-003 — Python toolchain required scripts не закреплён

- **Location:** .github/workflows/backend.yml:30
- **Сценарий:** ubuntu-latest меняет default Python minor.
- **Expected:** Python version закреплён как Go/Node и проверяется fail-fast.
- **Actual:** workflows вызывают bare python3; setup-python, .python-version и wrapper отсутствуют.
- **Evidence:** local suite использовал Python 3.10, CI version определяется runner image.
- **Root cause:** Python считался system utility, хотя является значимой частью tooling.
- **Recommendation:** pin version, actions/setup-python и общий version-check wrapper.
- **Acceptance test:** все workflows сообщают одну exact version; wrong interpreter падает до tests.

### QUAL-006 — smoke-api не проверяет logs endpoint

- **Location:** scripts/smoke-api.sh:111; docs/TESTING_STRATEGY.md:193, :271
- **Сценарий:** /api/pipeline/runs/id/logs ломается при рабочем status/artifacts API.
- **Expected:** required smoke обнаруживает routing/pagination regression.
- **Actual:** script проверяет artifacts/list/cancel, но не вызывает logs.
- **Evidence:** malformed/5xx logs response не влияет ни на один assertion.
- **Root cause:** smoke script отстал от documented baseline.
- **Recommendation:** запросить logs с cursor/limit и проверить wire shape/cursor.
- **Acceptance test:** stubbed 5xx или malformed logs response делает smoke-api красным.

### QUAL-007 — Deterministic mock Playwright scenarios не оркестрируются CI

- **Location:** ui/package.json:12; .github/workflows/ui.yml:29; ui/e2e/analysis-failed-shard-mock.spec.ts:447
- **Сценарий:** recovery flow ломается на browser interaction, уже покрытом одним из mock specs.
- **Expected:** provider-free mocks регулярно выполняются required CI.
- **Actual:** make test/UI workflow запускают только Vitest; shared manual/live entrypoint пропускает mocks без explicit scenarios.
- **Evidence:** audit выполнил 93 Vitest tests и отдельно 7 mocks; canonical workflows не запускают последние.
- **Root cause:** deterministic mocks не имеют отдельного runner matrix.
- **Recommendation:** e2e:mock target с local Vite server и seven-scenario matrix, без providers/network.
- **Acceptance test:** семь scenarios passed, skipped=0; broken selector валит required UI check.

## Confirmed dead-code register

### DEAD-001 — Semantic card rendering/enrichment cluster

- **Impact / confidence / priority / effort:** Normal / Confirmed / P2 / M
- **Location:** internal/orchestrator/semantic_cards.go:18, :46, :201, :218, :258, :293, :373, :389, :402, :417, :432, :449, :471, :492, :505, :514, :526, :535, :547, :577
- **Сценарий:** разработчик меняет Step 1 cards, считая этот двадцатисимвольный cluster частью production flow.
- **Expected:** cluster достижим и защищён integration tests либо отсутствует.
- **Actual:** production references отсутствуют; spec при этом продолжает обещать enrichment.
- **Evidence:** Staticcheck U1000 и ast-index refs/usages; exports/reflection/embed/build tags/codegen/test-only wiring исключены.
- **Root cause:** отключённый pipeline enrichment path.
- **Recommendation:** разрешить BUG-010 — восстановить behavior или удалить cluster вместе с contract change.
- **Acceptance test:** Step 1 integration проходит, перечисленные U1000 отсутствуют.

### DEAD-002 — Refresh semantic guard cluster

- **Impact / confidence / priority / effort:** Normal / Confirmed / P2 / M
- **Location:** internal/orchestrator/semantic_utils.go:202, :211, :228, :241, :253, :267
- **Сценарий:** разработчик рассчитывает на существующий refresh guard при изменении semantic normalization.
- **Expected:** policy вызывается и тестируется либо удалена.
- **Actual:** шесть helpers не имеют production callers.
- **Evidence:** Staticcheck U1000 и нулевые ast-index refs/usages.
- **Root cause:** незавершённая migration normalisation.
- **Recommendation:** разрешить REF-001; не сохранять parallel unreachable policy.
- **Acceptance test:** выбранная policy покрыта refresh tests, U1000 cluster исчез.

### DEAD-003 — Runtime-draft compatibility wrappers

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** internal/orchestrator/runtime_drafts.go:18, :22, :33, :37, :41, :56
- **Сценарий:** maintainer меняет wrapper, ожидая влияние на runtime draft validation.
- **Expected:** один canonical call path.
- **Actual:** шесть wrappers не вызываются после migration в internal/runtimedrafts.
- **Evidence:** Staticcheck U1000 и empty refs/usages.
- **Root cause:** package migration оставила forwarding surface.
- **Recommendation:** удалить wrappers, использовать canonical package напрямую.
- **Acceptance test:** runtime-draft tests проходят; symbols отсутствуют в Staticcheck.

### DEAD-004 — Sharding planner/artifact wrappers

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** internal/orchestrator/sharding_artifacts.go:171; sharding_planner.go:177, :199, :246
- **Сценарий:** maintainer исправляет legacy planner helper вместо active input-oriented path.
- **Expected:** только active planner API остаётся в package.
- **Actual:** четыре wrappers недостижимы.
- **Evidence:** U1000 плюс empty refs/usages.
- **Root cause:** transition к input-oriented planner helpers.
- **Recommendation:** удалить wrappers без новых forwarding aliases.
- **Acceptance test:** deterministic sharding tests проходят; U1000 отсутствуют.

### DEAD-005 — Provider default-argument wrappers

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** internal/runtime/claudecode/runner.go:170; codexcode/runner.go:178; qwencode/prompt_policy.go:10
- **Сценарий:** developer меняет default args wrapper, ожидая изменение provider invocation.
- **Expected:** один permission-aware builder path.
- **Actual:** три старых entry points не вызываются.
- **Evidence:** U1000 и empty refs/usages; active adapters используют новые builders.
- **Root cause:** migration к permission-aware builders.
- **Recommendation:** удалить wrappers.
- **Acceptance test:** adapter argument tests проходят; три symbols исчезают из Staticcheck.

### DEAD-006 — Docflow compatibility helpers

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** internal/orchestrator/docflow.go:510, :514
- **Сценарий:** maintainer предполагает, что local helpers участвуют в artifact quality gate.
- **Expected:** canonical validation path однозначен.
- **Actual:** две проверки перенесены в artifact-quality layer, local helpers не вызываются.
- **Evidence:** U1000 и empty refs/usages.
- **Root cause:** incomplete cleanup после migration.
- **Recommendation:** удалить helpers после сохранения canonical package tests.
- **Acceptance test:** docflow/artifact-quality tests проходят; U1000 отсутствуют.

### DEAD-007 — Независимые остатки partial refactors

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** internal/api/review_diff.go:788; internal/model/store.go:191; internal/orchestrator/quality.go:241; internal/reports/compiler.go:266; internal/runtime/promptcontract/collect_repair.go:213
- **Сценарий:** maintainer читает пять standalone helpers как действующие extension points.
- **Expected:** helpers имеют consumers либо удалены.
- **Actual:** call sites заменены, helpers остались недостижимыми.
- **Evidence:** U1000 и empty refs/usages для каждого.
- **Root cause:** локальные refactors не завершили cleanup.
- **Recommendation:** удалять независимо, без новых wrappers.
- **Acceptance test:** package tests проходят; пять symbols отсутствуют в Staticcheck.

### DEAD-008 — TypeScript unused import и props

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** ui/src/App.tsx:25; ui/src/components/StagePanels.tsx:1545, :3380
- **Сценарий:** UI maintainer считает Diagnostic, issueCount и nonDiagramArtifacts частью active rendering contract.
- **Expected:** import/props читаются либо отсутствуют.
- **Actual:** consumers удалены, type/call-site wiring осталось.
- **Evidence:** clean tsc noUnusedLocals/noUnusedParameters указывает ровно эти три записи.
- **Root cause:** UI refactors не очистили signatures/imports.
- **Recommendation:** удалить import и props вместе с call-site wiring.
- **Acceptance test:** strict noUnused typecheck проходит.

### DEAD-009 — Legacy QA client

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** ui/src/lib/qaApi.ts:8, :51
- **Сценарий:** maintainer исправляет legacy POST QA wrapper, но активный UI использует async Q&A API.
- **Expected:** documented fallback имеет call site либо wrapper отсутствует.
- **Actual:** QAAskResponse, QAAskRawResponse и askArchitectureQuestion не импортируются и не вызываются.
- **Evidence:** reference search empty; normal typecheck/Vitest pass.
- **Root cause:** migration на async Q&A оставила legacy UI client.
- **Recommendation:** удалить wrapper либо явно подключить документированный fallback.
- **Acceptance test:** reference search остаётся empty до удаления; typecheck/Vitest проходят после cleanup.

### DEAD-010 — Неиспользуемые facade cleanup members

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** ui/src/hooks/useRunReview.ts:39; useGitDiff.ts:9, :25; useRunExplorer.ts:164
- **Сценарий:** developer рассчитывает, что facade cleanup members сбрасывают state при run switch.
- **Expected:** clearRunReviewSummary, clearGitDiff и selectedDiffPath state имеют consumers либо отсутствуют.
- **Actual:** App/consumers их не читают; selectedDiffPath создаёт лишние rerenders.
- **Evidence:** reference search показывает только facade propagation/internal declarations.
- **Root cause:** hook facade пережил consumer cleanup.
- **Recommendation:** удалить surface либо подключить к explicit run-switch cleanup.
- **Acceptance test:** reference search содержит только active consumers; typecheck/Vitest зелёные.

### DEAD-011 — normalize_binary_flag

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** scripts/full-run-batch-matrix.sh:408
- **Сценарий:** maintainer меняет binary flag normalization, ожидая влияние на matrix parsing.
- **Expected:** helper вызывается либо отсутствует.
- **Actual:** вызовов нет.
- **Evidence:** ShellCheck/reference validation.
- **Root cause:** parser refactor оставил старый helper.
- **Recommendation:** удалить либо подключить к фактическому parsing path.
- **Acceptance test:** ShellCheck clean; matrix contract tests проходят.

### DEAD-012 — run_dod_precheck_make

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** scripts/full-run-batch.sh:1036
- **Сценарий:** maintainer исправляет dead helper вместо active dod_precheck_cmd path.
- **Expected:** один precheck implementation.
- **Actual:** function не вызывается; logic дублируется active path около строки 1729.
- **Evidence:** ShellCheck/reference validation.
- **Root cause:** replacement не удалил old function.
- **Recommendation:** удалить dead helper.
- **Acceptance test:** batch tests и ShellCheck проходят.

### DEAD-013 — Unused frontend status assignments

- **Impact / confidence / priority / effort:** Minor / Confirmed / P3 / S
- **Location:** scripts/full-run-batch.sh:1838, :1855
- **Сценарий:** maintainer считает frontend_status/frontend_reason частью итоговой classification.
- **Expected:** values читаются downstream либо не вычисляются.
- **Actual:** assignments не имеют consumers; frontend_result_summary может оставаться validation call.
- **Evidence:** ShellCheck/reference validation.
- **Root cause:** reporting path удалил consumers, assignments остались.
- **Recommendation:** удалить assignments или явно использовать values.
- **Acceptance test:** behavior tests проходят, ShellCheck не сообщает unused assignments.

## Карта необходимых доработок

| Кластер | Findings | Предлагаемая граница refactor |
|---|---|---|
| Crash-safe workspace generations | BUG-001, BUG-005 | atomic snapshot writer + journal/last-good recovery primitive |
| Server/session lifecycle | BUG-002, BUG-003, BUG-006, BUG-007 | one owner for server context, service generation and async cleanup |
| Collect contract integrity | BUG-008, BUG-009 | symmetric evidence validator before checkpoint/apply and after remap |
| Run-scoped UI data | BUG-011–BUG-017 | request generation tokens, abort, provisional mutations and keyed payload state |
| Semantic pipeline contract | BUG-010, REF-001, DEAD-001/002 | explicit decision: restore behavior or remove contract/dead implementation |
| Release/reproducibility | BUG-018, BUG-019, REF-002 | locked toolchain, deterministic embed gate, verified release evidence |
| Required CI quality | QUAL-004–QUAL-007, REF-003 | canonical lint/smoke/mock targets with pinned runtimes |
| Accessible interaction primitives | QUAL-001–QUAL-003 | shared tabs, combobox and status-announcement controllers |
| Cleanup | DEAD-003–DEAD-013 | small package-scoped deletion slices after behavior decisions |

## Remediation roadmap

Canonical implementation slicing and dependency order are maintained in
`docs/BACKLOG.md` → **Epic 19 — Code Quality Audit Remediation (Local-first MVP)**.

### P1 — до следующего release

1. **Crash safety:** объединить BUG-001 и BUG-005 в один design slice с atomic generation writer, failure injection и last-good recovery.
2. **Async lifecycle:** BUG-002/003/006/007 — server-owned context, panic isolation, service-generation locking и hot-swap conflict policy.
3. **Collect correctness:** BUG-008/009 — schema/validator/fixtures/spec synchronization; при реализации обязательно использовать schema-guardian workflow.
4. **Source freshness:** BUG-004 — deterministic remote-default SHA resolution для unpinned git_url.
5. **UI consistency:** BUG-011–017 — единый request-scoped state layer; начать с run selection и mutation acknowledgement, затем editors/Q&A.
6. **Release gates:** BUG-018/019 и REF-002 — deterministic embedded UI drift check, verified release verdict и locked contract tools.

### P2 — ближайшие quality slices

1. Принять одно архитектурное решение по BUG-010/REF-001 и удалить либо восстановить DEAD-001/002.
2. Реализовать accessible tabs/combobox/status primitives для QUAL-001–003.
3. Добавить required PR lint, ShellCheck, logs smoke и mock Playwright matrix для QUAL-004–007.
4. Закрепить Python runtime по REF-003.
5. Добавить missing failure-injection/concurrency tests и isolated UI hook tests с deferred/out-of-order responses.

### P3 — opportunistic cleanup

Удалить DEAD-003–DEAD-013 небольшими package-scoped slices. Каждый slice должен сохранять make contracts, make test, make lint и make build; forwarding wrappers вместо удаления не добавлять.

## Residual test gaps

- Existing green race suite не исполняет concurrent workspace/runner replacement path из BUG-007.
- Promotion/persistence tests не делают systematic failure injection по каждому write/remove boundary.
- UI hooks не имеют isolated deferred/out-of-order response tests; current mock E2E в основном проверяет rendered happy/recovery states.
- Historical staged-vs-canonical artifact content не тестируется.
- V8 UI coverage не собран, потому что coverage provider отсутствует в lockfile.
- Fake runtime package coverage 47.4% указывает на область для дополнительного test design, но сам процент не классифицирован как finding.

## Audit completion

- Production code, public APIs, schemas и contracts не изменены.
- Live providers/matrices не запускались.
- Исправления findings не входили в scope.
- Итоговые tracked changes должны состоять только из этого отчёта и quality-only ExecPlan в docs/PLANS.md.
