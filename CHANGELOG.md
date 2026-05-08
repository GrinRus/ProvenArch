# Changelog

All notable user-facing changes are tracked here. ProvenArch uses SemVer-style release tags, with `v0.x` treated as beta/pre-release foundation.

## Unreleased

OSS readiness hardening:
- Added security reporting, support, governance, code of conduct, CODEOWNERS, PR template, and issue templates.
- Hardened GitHub Actions with pinned actions, read-only defaults, timeouts, release environment, SBOM generation, and artifact attestations.
- Added Dependabot, Dependency Review, CodeQL, and OpenSSF Scorecard workflows.
- Updated UI dependency chain to remove moderate `npm audit` findings.
- Enforced exact Node.js version from `.node-version` for source builds.
- Enforced exact Go version from `.go-version` for CI, release, local Makefile, and smoke builds while preserving `go.mod` compatibility level `go 1.20`.

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
