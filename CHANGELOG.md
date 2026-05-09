# Changelog

All notable user-facing changes are tracked here. ProvenArch uses SemVer-style release tags, with `v0.x` treated as beta/pre-release foundation.

## v0.1.1 - 2026-05-09

Beta hardening and onboarding release.

Highlights:
- Reworked README onboarding around what ACP does, local install, first fake analysis, runtime choice, artifacts, workspace model, and release evidence.
- Added security reporting, support, governance, code of conduct, CODEOWNERS, PR template, and issue templates.
- Hardened GitHub Actions with pinned actions, read-only defaults, timeouts, release environment, SBOM generation, and artifact attestations.
- Added Dependabot, Dependency Review, CodeQL, and OpenSSF Scorecard workflows.
- Updated UI dependency chain to remove moderate `npm audit` findings.
- Enforced exact Node.js version from `.node-version` for source builds.
- Enforced exact Go version from `.go-version` for CI, release, local Makefile, and smoke builds while preserving `go.mod` compatibility level `go 1.20`.

Verification notes:
- Public `install.sh` smoke was verified against the current published release path with checksum validation.
- Local fake walkthrough smoke was verified with `acp doctor`, `acp serve --auto-init --dry-run`, `acp run --pipeline init --runtime fake --non-interactive`, and local UI/API endpoints.

Known limitations:
- `v0.1.1` remains a beta/pre-v1 release. Public behavior and artifact contracts can still evolve before `v1.0.0`.
- Hosted/multi-tenant mode and security/compliance enforcement remain out of scope.
- Release readiness still requires trusted-machine release gate evidence from `reports/release_verdict_<matrix-id>.json`.

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
