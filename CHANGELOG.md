# Changelog

All notable user-facing changes are tracked here. ProvenArch uses SemVer-style release tags, with `v0.x` treated as beta/pre-release foundation.

## v0.1.10 - 2026-08-02

Beta prerelease candidate for the local-first product shell, trustworthy artifact authorities,
impact-aware refresh, and provider-free release hardening accumulated after `v0.1.9`.

Highlights:
- Completed the evidence-oriented product shell across Architecture Home, Runs, Knowledge, Changes,
  Ask, Proposals and explicit Git publication, including responsive/mobile and recovery-state QA.
- Added server-owned run snapshot and evidence authority resolution so current, historical, QA and
  audit artifacts cannot silently fall back across trust boundaries.
- Added impact-aware refresh planning and immutable baseline preservation with source-revision,
  shard-identity and digest checks before retained artifacts are reused.
- Hardened workspace I/O against symlink escapes, made run persistence publish immutable snapshots,
  and coordinated run, session and Git mutation admission around a shared lifecycle boundary.
- Added typed validation/recovery routing, a read-only artifact integrity auditor, stronger
  claim/citation identity checks, expanded workspace-health findings and bounded evidence viewing.
- Added the explicit Ask-to-Proposal mutation and UI handoff while preserving the deterministic Q&A
  compatibility surface.
- Unified path-scope matching and tightened provider prompt/reconciliation behavior for artifact-only
  execution, focused repair, Architecture Home and collect provenance.
- Added the provider-free `make offline-closure` gate with race suites, readable fixture drift,
  contract validation, Go/Python/UI tests, rendered mock E2E, lint, build and embedded UI parity.
- Replaced the repository entrypoint with an English-first open-source README and a deterministic
  fake-runtime walkthrough, while keeping exact contracts in the canonical specifications.

Verification notes:
- Product and hardening slices through PR #199 were merged into `main`; PR #191 then passed all 11
  required/advisory checks and was rebase-merged as `68217aaba9dbd1c81814e8f5c7d23608bea5b2e3`.
- Exact Go `1.25.10`, Node `22.21.1` and npm `10.9.4` provider-free `make offline-closure` passed on
  that post-merge SHA: race suites, 90 readable fixtures, 263 Python tests, 158 UI tests, 7/7
  rendered mock scenarios, contracts, lint, build and embedded UI parity all completed successfully.
- Codex diagnostic smoke `smoke-tiny-bank-20260802T084522Z` on the preceding qualification SHA
  `5cf7ba976191b1b732ad9b49fb1b1b761d997926` passed strict execution 1/1 from snapshot artifacts
  with runtime contract and artifact quality passed, an `Excellent` verdict, and no failures,
  repairs or stalls. It is diagnostic evidence only and is not a release verdict for this candidate.
- The following Codex Bank/Open edX regression attempt was rejected as `infra_incomplete_cycle`
  after its controlling terminal session closed; no matrix result was produced and no partial output
  is accepted as evidence.

Known limitations:
- `v0.1.10` is an explicitly owner-authorized `UNQUALIFIED PRERELEASE`. It does not claim canonical
  `RELEASE READY`: the tracked tag-scoped waiver records the missing Qwen/Claude live evidence and
  composite verdict while preserving the normal fail-closed evidence path for releases without a waiver.
- Qwen quota is currently unavailable, so the required qwen/claude/codex trusted-machine composite
  has not been completed. No `release_verdict_*.json` is attached to this candidate.
- The successful Codex smoke does not replace regression, frontend init-inspect, cross-provider or
  canonical `baseline` plus `parallel-default` release evidence.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.

## v0.1.9 - 2026-07-09

Beta patch release for Console V2 UX recovery polish and rendered recovery-state QA after `v0.1.8`.

Highlights:
- Improved Console V2 recovery states across Source, Readiness, Analysis, Review and Publish so first-time operators can distinguish blocked setup, provider readiness failures, canceled/retained runs, failed shards, partial artifacts and publish handoff states.
- Added clearer provider/runtime guidance for unavailable providers, headless probe timeouts, active provider stream output before durable artifacts, collect artifact handoff failures and repair-heavy/stalled runs.
- Improved onboarding recovery for duplicate source names, recent workspace availability, runner selection, `qwen-code` readiness failures and disabled first-analysis actions until workspace, sources, runner and local readiness are valid.
- Added rendered Playwright QA coverage for provider stream, failed shard drilldown, permission recovery, Ask recovery, Publish Git recovery, Source recovery and onboarding recovery, including desktop/mobile layout, disabled-action, console-error and horizontal-overflow checks.
- Recorded UX/UI assessment evidence from medium `regres long` diagnostics, including the remaining live runtime/provider blockers observed on `qwen` headless probe timeout and FTGO collect artifact handoff stalls.
- Updated the embedded UI bundle so release binaries serve the polished recovery UI.

Verification notes:
- PR #123 squash-merged into `main` at `90c2931` after green PR checks: `backend`, `contracts`, `ui`, `golden`, `smoke-api`, `smoke-cli`, `dependency-review` and CodeQL.
- Local validation for the UX/UI slices passed before merge: rendered Playwright recovery scenarios plus `make contracts`, `make test`, `make lint` and `make build` with exact Node.js `22.21.1`.
- Release metadata validation passed before tagging: `git diff --check`, release distribution tests, `make contracts`, `make test`, `make lint` and `make build` with exact Node.js `22.21.1`.
- Medium live diagnostic `regres-long-posthog-ftgo-20260709T162033Z` was intentionally recorded as non-release evidence. It failed strict execution on provider/runtime reliability, not on the newly rendered UI recovery slices.

Known limitations:
- `v0.1.9` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- This release does not claim canonical `RELEASE READY` status without a fresh trusted-machine `reports/release_verdict_<matrix-id>.json` plus accepted SWE UX and artifact-quality assessments.
- Live provider reliability still needs follow-up: the latest medium diagnostic showed `qwen` headless readiness timeout and `runtime_stalled_before_artifacts` during collect artifact handoff.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.

## v0.1.8 - 2026-07-07

Beta patch release for artifact-quality acceptance hardening and live E2E diagnostics after `v0.1.7`.

Highlights:
- Added product-level `artifact_quality.*` signals for weak but formally valid architecture outputs, including placeholder/scaffold artifacts, empty findings with coverage gaps, low semantic density, gap-only diagrams, proposals/findings disconnects and low-actionability proposals.
- Split live E2E verdict reporting into runtime contract status and artifact quality status, so live matrices can report `runtime passed, artifact quality failed` instead of treating valid-but-useless artifacts as an optimistic PASS.
- Made any `artifact_quality.*` signal a strict live matrix/release blocker while preserving the ProvenArch/live boundary: product logic emits generic public artifact facts, and live E2E reads only public reports and taskrun quality JSON.
- Tightened `Excellent` verdict policy: `analysis:*` issues, runtime repair-heavy paths and real stall pressure now cap the run at `Needs review` even when strict runtime/artifact gates pass.
- Added step-level Excellent blocker diagnostics with repair attempts, stall counts, validation classes and first/terminal validation excerpts for faster live-run triage.
- Hardened collect manifest repair and draft validation around citation-id uniqueness, semantic question shape, step2 as-is evidence, step4 finding linkage and proposal actionability.
- Recorded follow-up backlog slices for the remaining first-pass Codex smoke blockers: `step2.asis_docs` downstream-index wording and `step4.proposals` low-actionability first-pass output.

Verification notes:
- PR #120 merged after green PR checks for merge commit `7f65499`.
- Local validation for the product patch passed before merge: `make contracts`, `make test`, `make lint` and `make build`.
- Release metadata validation passed before tagging: `git diff --check`, docs sync, release distribution checks, `make contracts`, `make test`, `make lint` and `make build` with exact Node.js `22.21.1`.
- Codex diagnostic smoke `smoke-tiny-bank-20260707T053308Z` reached strict `PASS` with `runtime_contract_status=passed`, `artifact_quality_status=passed`, `artifact_quality_failed=0` and no `artifact_quality.*` blockers.
- The same diagnostic smoke remained `Needs review`, not `Excellent`, because first-pass `init.step2.asis_docs` and `init.step4.proposals` still required focused repair/stall recovery.

Known limitations:
- `v0.1.8` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- This release does not claim canonical `RELEASE READY` status without a fresh trusted-machine `reports/release_verdict_<matrix-id>.json` plus accepted SWE UX and artifact-quality assessments.
- The remaining first-pass `Excellent` blockers are tracked in `docs/BACKLOG.md` as `18G Step2 first-pass Excellent blocker` and `18H Step4 first-pass actionability blocker`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.

## v0.1.7 - 2026-06-11

CI-only beta patch release for live artifact quality hardening and Review UI readability after `v0.1.6`.

Highlights:
- Added stronger live-run artifact quality gates for placeholder/scaffold-only reports, empty semantic models, gap-only C4 diagrams, incomplete citation/final indexes, sparse findings, and unusable proposal/changelog outputs.
- Hardened collect/runtime prompt contracts and recovery paths so valid structure alone is not enough: collect manifests must carry evidence-backed semantic signal, provider/tool side-effect paths are rejected, and deterministic manifest recovery is constrained to current authored shard docs plus bounded repo evidence.
- Improved generated architecture model and C4 compilation so relationship-backed context diagrams can be produced from extracted entities/edges instead of silently accepting gap-only output as success.
- Improved Review and Publish readability: artifact previews stay legible on desktop/mobile, C4/Mermaid previews use scrollable space, failed-run states point operators to the latest successful artifacts, and live frontend checks now inspect readable artifact content instead of only screenshots/selectors.
- Added `GET /api/system/version` and updated the top status bar to show the actual running build metadata instead of a hard-coded `v0.1.1 beta` label.
- Anonymized live-derived regression material so local provider paths and run-specific details do not leak into product fixtures.

Verification notes:
- PR #110 merged after green PR checks and green `main` CI for merge commit `e68d35c`.
- Local validation for the product patch passed before merge: `make contracts`, `make test`, `make lint` and `make build`.
- Release metadata validation passed before tagging: `git diff --check`, docs sync, release distribution checks, `make contracts`, `make test`, `make lint` and `make build`.
- Fresh canonical trusted-machine `release long/full` was not run for this patch. Preflight found the current host writable and provider binaries available, but canonical Open edX/OpenStack path checkouts under `/tmp/provenarch-live-e2e` were missing, so this release does not claim canonical `RELEASE READY` status without a fresh `reports/release_verdict_<matrix-id>.json`.

Known limitations:
- `v0.1.7` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.
- Canonical release readiness still requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json` verified by `scripts/verify-release-verdict.py`.

## v0.1.6 - 2026-06-06

CI-only beta patch release for provider command resolution and onboarding path picker polish after `v0.1.5`.

Highlights:
- Normalized the boundary between stable ACP provider IDs and local executable names: `claude-code` now resolves `ACP_CLAUDE_CMD`, then `claude`, then legacy `claude-code`, while `qwen-code` and `codex-code` keep their `qwen`/`codex` defaults with env override support.
- Improved readiness/onboarding copy so the UI separates provider ID from executable command and reports actionable override guidance when a command is missing.
- Added local-only onboarding path suggestions and searchable comboboxes for architecture workspace paths and local target repository paths.
- Polished onboarding rendering for long paths, diagnostics, missing recents, duplicate repo names and narrow/mobile viewports.
- Updated install, troubleshooting, API and stakeholder documentation so clean UI startup is documented as `acp serve` with default `fake` runtime.

Verification notes:
- PR #104 merged after green PR checks and green `main` CI for merge commit `47a691e`.
- Local validation for the product patch passed before merge: `git diff --check`, `make contracts`, `make test`, `make lint`, `make build`, `./bin/acp serve --dry-run` and rendered onboarding smoke.
- Release metadata validation passed before tagging: `git diff --check`, docs sync, release distribution/install tests, `make contracts`, `make test`, `make lint` and `make build`.
- Fresh trusted-machine `release-fast` was not run for this patch. This release does not claim canonical `RELEASE READY` status without a fresh `reports/release_verdict_<matrix-id>.json`.

Known limitations:
- `v0.1.6` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.
- Canonical release readiness still requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json` verified by `scripts/verify-release-verdict.py`.

## v0.1.5 - 2026-06-04

CI-only beta patch release for release workflow cleanup after `v0.1.4`.

Highlights:
- Updated the release workflow's Syft installer action to Node 24-compatible `anchore/sbom-action/download-syft@v0.24.0` on pinned commit `e22c389904149dbc22b58101806040fa8d37a610`.
- Updated the release workflow's GoReleaser action to Node 24-compatible `goreleaser/goreleaser-action@v7.2.2` on pinned commit `5daf1e915a5f0af01ddbcd89a43b8061ff4f1a89`.
- Preserved the existing release permissions, `github-release` environment gate, SBOM/provenance flow, GoReleaser arguments, checksums, and pinned-SHA policy.
- No product behavior, backend API, CLI flag, workspace schema, runtime artifact contract, or provider contract changed in this patch.

Verification notes:
- PR #100 merged after green PR checks and green `main` CI for merge commit `afa3ff8`.
- This patch is intended to make future release runs free of Node 20 / CodeQL v3 deprecation annotations.
- Fresh trusted-machine `release-fast` was not run for this patch. This release does not claim canonical `RELEASE READY` status without a fresh `reports/release_verdict_<matrix-id>.json`.

Known limitations:
- `v0.1.5` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.
- Canonical release readiness still requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json` verified by `scripts/verify-release-verdict.py`.

## v0.1.4 - 2026-06-04

CI-only beta patch release for GitHub Actions hygiene and clean UI startup polish.

Highlights:
- Modernized pinned GitHub Actions to Node 24-compatible action versions and CodeQL v4 while preserving permissions, timeouts, release environment gates, SBOM/provenance, and pinned-SHA policy.
- Added local-only Recent workspaces to onboarding: successful workspace create/open records the workspace outside `workspace.yaml`, the launcher shows newest-first recents with `Open` and `Forget`, and missing paths are visible but not openable.
- Improved reopen flow for existing workspaces: onboarding hydrates sources from `workspace.yaml`, valid manifests can proceed to `Ready` after runner selection, and invalid manifests route the operator back to `Sources` with diagnostics.
- Improved run review resume: Console V2 selects the newest active run on bootstrap, otherwise selects the newest completed run and opens `Review` when artifacts exist.
- Fixed a rendered UI startup regression where selecting a new draft workspace could trigger `/api/workspace/bundle` before `workspace.yaml` existed, causing a noisy `428 workspace_not_selected` browser console error.

Verification notes:
- PR #97 modernized GitHub Actions pins and merged after green PR checks and green `main` CI for merge commit `bc45d1c`.
- PR #98 shipped product polish and merged after green PR checks and green `main` CI for merge commit `5337e61`.
- Local validation for product polish passed: `git diff --check`, `go test ./internal/docsync`, focused `go test ./internal/api`, UI unit suite `69/69`, `make contracts`, `make test`, `make lint`, `make build`, and Playwright rendered smoke from clean launcher to Analysis/Review/Publish with no browser console issues or HTTP >=400 responses.
- Fresh trusted-machine `release-fast` was not run for this patch. This release does not claim canonical `RELEASE READY` status without a fresh `reports/release_verdict_<matrix-id>.json`.

Known limitations:
- `v0.1.4` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.
- Canonical release readiness still requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json` verified by `scripts/verify-release-verdict.py`.

## v0.1.3 - 2026-06-04

CI-only beta patch release for clean UI onboarding fixes after `v0.1.2`.

Highlights:
- Fixed the Console V2 top status release label: the UI now reads local build metadata through `/api/system/info` instead of showing a hardcoded `v0.1.1 beta` label.
- Improved first-run onboarding workspace path handling on macOS: `/tmp/...` is accepted as an alias for the system temp root while root paths, relative paths, traversal, NUL bytes, and paths outside allowed home/temp roots remain blocked.
- Made fake runtime presentation consistent in long-run review: fake runs show provider `fake` in review summaries and Analysis step cards even when the process-level fallback provider is `claude-code`.
- Deduplicated run-level and step-level warnings so ActiveRunStrip, Analysis mission control, and `/api/pipeline/runs/<run_id>/review-summary` show the same warning total.
- Clarified Publish Git diff scope: selected-run diffs and full-workspace diffs now have explicit labels and hints.

Verification notes:
- PR #93 was merged after green PR checks and green `main` CI for merge commit `1e6087e`.
- Local validation for the patch passed: `go test ./internal/api`, `git diff --check`, `make contracts`, `make test`, `make lint`, `make build`, and `./bin/acp serve --runtime fake --dry-run`.
- Browser QA passed for clean onboarding with `/tmp/...`, local source repo setup, fake runner selection, first analysis, Review, selected-run diff, full-workspace diff, and empty browser console warnings/errors.
- Fresh trusted-machine `release-fast` was not run for this patch. This release does not claim canonical `RELEASE READY` status without a fresh `reports/release_verdict_<matrix-id>.json`.

Known limitations:
- `v0.1.3` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.
- Canonical release readiness still requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json` verified by `scripts/verify-release-verdict.py`.

## v0.1.2 - 2026-06-02

CI-only beta release for onboarding-first startup and long-run review UX.

Highlights:
- Added UI-first launcher mode for `acp serve` without `--workspace`: operators can create/open an architecture workspace, configure sources, select a runner, review readiness, and then enter Console V2.
- Kept direct-mode compatibility for scripts, CI, live E2E, and experienced users through `acp serve --workspace <path>`.
- Added multi-repo source setup in onboarding and Source, using the existing `workspace.yaml.repos[]` model for local paths or Git URLs with optional refs.
- Added mandatory runner selection in onboarding for `fake`, `claude-code`, `qwen-code`, and `codex-code`, with `fake` as the recommended deterministic first walkthrough.
- Added long-run review UX: persistent active-run strip, canonical step review model, step-level artifacts/logs/evidence/diff tabs, and clearer partial/failure states for long provider runs.
- Added Review Queue and integrated real workspace Git diff into Review, Proposals, and Publish, including a fuller Git Review Room for publication decisions.
- Added local UI APIs for onboarding, run review summaries, and workspace Git diff: `/api/onboarding/*`, `/api/pipeline/runs/<run_id>/review-summary`, and `/api/git/diff`.
- Hardened frontend live E2E and already-initialized workspace regressions while preserving the release-facing `UI_E2E_SCENARIO=init-inspect` shell.
- Hardened launcher workspace path handling for UI onboarding: launcher paths are normalized, constrained to user-home or system temp roots, and documented; direct-mode remains the advanced path.

Verification notes:
- PR #91 was merged after green PR checks and green `main` CI for merge commit `24301e2`.
- Local release-prep checks for the release metadata patch passed: `git diff --check`, release distribution/install tests, `make contracts`, `make test`, `make lint`, and `make build`.
- Fresh trusted-machine `release-fast` was skipped by owner decision for this prerelease. This release does not claim canonical `RELEASE READY` status without a fresh `reports/release_verdict_<matrix-id>.json`.

Known limitations:
- `v0.1.2` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.
- Canonical release readiness still requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json` verified by `scripts/verify-release-verdict.py`.

## v0.1.1 - 2026-06-01

Beta hardening, onboarding, Console V2, and no-gate prerelease candidate.

Highlights:
- Reworked README onboarding around what ACP does, local install, first fake analysis, runtime choice, artifacts, workspace model, and release evidence.
- Added security reporting, support, governance, code of conduct, CODEOWNERS, PR template, and issue templates.
- Hardened GitHub Actions with pinned actions, read-only defaults, timeouts, release environment, SBOM generation, and artifact attestations.
- Added Dependabot, Dependency Review, CodeQL, and OpenSSF Scorecard workflows.
- Updated UI dependency chain to remove moderate `npm audit` findings.
- Enforced exact Node.js version from `.node-version` for source builds.
- Enforced exact Go version from `.go-version` for CI, release, local Makefile, and smoke builds while preserving `go.mod` compatibility level `go 1.20`.
- Redesigned the embedded UI as Console V2 with mission-control layout, stage rail, health strip, right inspector, activity drawer, review workbench, publish gate, and saved visual references.
- Refined Console V2 daily-use UX: hard blockers, review warnings, and open questions are distinct; Source and Readiness have clearer responsibilities; Review chooses better default artifacts; Activity Drawer has explicit empty/recovery states.
- Hardened frontend live E2E `init-inspect` for already-initialized workspaces and preserved deterministic fake-runtime coverage for required CI.
- Hardened headless runtime release behavior for `qwen-code`, `claude-code`, and `codex-code` around artifact-only success, bounded silent-runner recovery, provider readiness, prompt-first artifact commands, draft repair validation, and multi-repo semantic quality gates.
- Added controlled stop after valid focused repair artifacts, so collect/validator/draft repair attempts do not wait for a still-running provider once required artifacts validate.
- Fixed embedded UI release assets so Mermaid diagram previews render from the released single binary.

Verification notes:
- Public `install.sh` smoke was verified against the current published release path with checksum validation.
- Local fake walkthrough smoke was verified with `acp doctor`, `acp serve --auto-init --dry-run`, `acp run --pipeline init --runtime fake --non-interactive`, and local UI/API endpoints.
- Main CI was green for the final `v0.1.1` release candidate branch before tagging.
- Local release-prep checks passed: `git diff --check`, release distribution/install tests, `make contracts`, `make test`, `make lint`, and `make build`.
- Non-release codex diagnostic smoke `smoke-tiny-bank-20260601T123637Z` passed with frontend `init-inspect` screenshots for Source, Readiness, Analysis, Review, Publish, and mobile Review.
- Fresh trusted-machine `release-fast` was skipped by owner decision for this prerelease. This release does not claim canonical `RELEASE READY` status for the final tag SHA.

Known limitations:
- `v0.1.1` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.
- Canonical release readiness still requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json` verified by `scripts/verify-release-verdict.py`.

## v0.1.0 - 2026-05-04

Initial public beta release.

Highlights:
- Single-binary `acp` distribution for macOS/Linux on `amd64` and `arm64`.
- Checksum-aware `install.sh`.
- Local-first workspace setup with `acp doctor`, `acp serve`, and fake runtime walkthrough.
- Go backend/orchestrator with embedded React/TypeScript UI.
- Docs-first runtime pipeline and deterministic fake baseline for required CI.
- Headless runtime provider surface for `claude-code`, `qwen-code`, and `codex-code`.

Known limitations:
- MVP beta: public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode is out of scope.
- Security/compliance enforcement is out of scope.
- Live release readiness requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json`.

Upgrade notes:
- No previous public release.
- Install with:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/GrinRus/ProvenArch/main/install.sh | sh
  ```
