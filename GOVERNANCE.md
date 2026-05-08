# Governance

ProvenArch is currently maintained by the repository owners while the project is in MVP beta.

## Maintainer responsibilities

Maintainers are responsible for:
- release decisions and GitHub Releases publication;
- security triage and coordinated disclosure;
- reviewing contract/schema changes;
- keeping required CI free of live provider dependencies;
- preserving local-first MVP boundaries.

## Decision policy

Changes that affect public CLI behavior, workspace contracts, schemas, release policy, or runtime trust boundaries require maintainer review.

Breaking changes before `v1.0.0` are allowed when they are documented in `CHANGELOG.md`, release notes, and affected specs. After `v1.0.0`, breaking changes should require a major version.

## Release policy

Release readiness requires:
- `make contracts`;
- `make test`;
- `make lint`;
- `make build`;
- `npm audit --prefix ui --audit-level=moderate`;
- `govulncheck ./...` with the release Go toolchain;
- trusted-machine live release gate evidence when claiming live release readiness.

Release gate evidence must come from `reports/release_verdict_<matrix-id>.json`; no other artifact is a release-readiness substitute.

## Contributions

External contributions are welcome when they are focused, test-covered, and aligned with `AGENTS.md`, `docs/ARCHITECTURE.md`, and `docs/spec/*`.
