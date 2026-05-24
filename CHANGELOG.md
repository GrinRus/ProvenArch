# Changelog

All notable user-facing changes are tracked here. ProvenArch uses SemVer-style release tags, with `v0.x` treated as beta/pre-release foundation.

## v0.1.1 - 2026-05-23

Beta hardening, onboarding, and live release-gate release.

Highlights:
- Reworked README onboarding around what ACP does, local install, first fake analysis, runtime choice, artifacts, workspace model, and release evidence.
- Added security reporting, support, governance, code of conduct, CODEOWNERS, PR template, and issue templates.
- Hardened GitHub Actions with pinned actions, read-only defaults, timeouts, release environment, SBOM generation, and artifact attestations.
- Added Dependabot, Dependency Review, CodeQL, and OpenSSF Scorecard workflows.
- Updated UI dependency chain to remove moderate `npm audit` findings.
- Enforced exact Node.js version from `.node-version` for source builds.
- Enforced exact Go version from `.go-version` for CI, release, local Makefile, and smoke builds while preserving `go.mod` compatibility level `go 1.20`.
- Hardened headless runtime release behavior for `qwen-code`, `claude-code`, and `codex-code` around artifact-only success, bounded silent-runner recovery, provider readiness, prompt-first artifact commands, draft repair validation, and multi-repo semantic quality gates.
- Added controlled stop after valid focused repair artifacts, so collect/validator/draft repair attempts do not wait for a still-running provider once required artifacts validate.
- Fixed embedded UI release assets so Mermaid diagram previews render from the released single binary.

Verification notes:
- Public `install.sh` smoke was verified against the current published release path with checksum validation.
- Local fake walkthrough smoke was verified with `acp doctor`, `acp serve --auto-init --dry-run`, `acp run --pipeline init --runtime fake --non-interactive`, and local UI/API endpoints.
- Trusted-machine release-fast gate `release-fast-20260523T171925Z` passed with all release providers (`qwen-code`, `claude-code`, `codex-code`), baseline and parallel-default sweeps, frontend init/cancel checks, artifact quality gates, and shard-plan invariant.

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
