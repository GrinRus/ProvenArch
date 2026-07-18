# Bank of Anthos — System Overview

## System at a glance
Bank of Anthos is a containerized retail-banking application described by `bank-of-anthos:README.md`.

## Analyzed scope
This as-is view covers the entire `bank-of-anthos` repository scoped to the current run.

## Domains and ownership
Repository-wide ownership is declared in `bank-of-anthos:.github/CODEOWNERS`.

## Key flows
The frontend-to-ledger flow is implemented under `bank-of-anthos:src/frontend/` and `bank-of-anthos:src/ledger/`.

## Integrations and datastores
PostgreSQL deployment evidence is recorded in `bank-of-anthos:kubernetes-manifests/`.

## Where to start
Start with `bank-of-anthos:README.md`.

## Safe-change guidance
Coordinate API and deployment changes across `bank-of-anthos:src/` and `bank-of-anthos:kubernetes-manifests/`.

## Evidence gaps and open questions
Operational ownership beyond `bank-of-anthos:.github/CODEOWNERS` remains an evidence gap.
