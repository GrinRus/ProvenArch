# ADR-20260329-implementation-stack-go.md

- **ADR ID:** ADR-20260329-implementation-stack-go
- **Status:** superseded
- **Date:** 2026-03-29
- **Owners:** ACP maintainers
- **Superseded by:** [ADR-20260410-headless-runtime-multi-provider](ADR-20260410-headless-runtime-multi-provider.md)

## Context

ACP is a local-first MVP:
- runs locally on developer machine
- runtime for extraction/analysis was initially constrained to **Claude Code headless** (MVP draft assumption)
- workspace is a Git repository with entity-per-file model and generated reports

We evaluated monorepo implementation stacks:
1) Java/Kotlin + Spring
2) Go
3) Node (TypeScript)

Local-first distribution and robust process/FS control are primary drivers.

## Decision

We choose **Go** for the backend/orchestrator/server.

UI choice (implementation detail):
- **React + TypeScript + Vite**
- UI is embedded into the Go binary for local distribution (serving `ui/dist`).

## Rationale

- Go is well-suited for **local-first packaging** (single binary) and dependable execution.
- Strong control over filesystem and process execution (headless runtime runner).
- Operational simplicity for early adopters: “download → run → open localhost”.

## Consequences

- UI still requires Node toolchain for build, but runtime distribution remains a single Go binary.
- We will shell out to `git` CLI in MVP for predictable behavior.
- Runtime scope in this ADR is historical and superseded by multi-provider MVP policy in ADR-20260410.

## Follow-ups

- Decide schema validation library in Go (if we enforce JSON Schema in code in MVP).
- Implement server skeleton + embedded UI placeholder (E1).
