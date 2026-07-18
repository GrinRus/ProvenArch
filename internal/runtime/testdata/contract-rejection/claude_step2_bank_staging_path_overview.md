# Bank of Anthos - As-Is Architecture Overview

## System at a glance
Bank of Anthos is a service-oriented banking application described by `README.md`.

## Analyzed scope
The analyzed scope includes `src/frontend` and `src/ledger`.

## Domains and ownership
Repository ownership is recorded in `.github/CODEOWNERS`.

## Key flows
The frontend calls ledger services described under `src/ledger`.

## Integrations and datastores
Datastore deployment evidence is recorded in `kubernetes-manifests`.

## Where to start
Review the staged shard overviews under `reports/taskruns/run_20260718_063551_001/staging/shards/`.

## Safe-change guidance
Validate service changes against `skaffold.yaml`.

## Evidence gaps and open questions
Service-level ownership remains unconfirmed outside `.github/CODEOWNERS`.
