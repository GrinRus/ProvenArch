# Changelog

All notable user-facing changes are tracked here. ProvenArch uses SemVer-style release tags, with `v0.x` treated as beta/pre-release foundation.

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
