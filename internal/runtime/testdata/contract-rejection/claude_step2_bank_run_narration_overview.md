# Bank of Anthos - As-Is Architecture

## System at a glance

Bank of Anthos is a cloud-native sample banking application. The current as-is view is derived from the run `run_20260718_080853_001` collect pass, which inspected repository entrypoints and service READMEs.

## Analyzed scope

The analysis covers `README.md`, `src/frontend/`, and `src/ledger/`.

## Domains and ownership

Default ownership is recorded in `.github/CODEOWNERS`.

## Key flows

The frontend calls ledger services described under `src/ledger/`.

## Integrations and datastores

PostgreSQL deployment evidence is recorded in `kubernetes-manifests/`.

## Where to start

Begin with `README.md` and `skaffold.yaml`.

## Safe-change guidance

Validate changes against `skaffold.yaml` and service tests.

## Evidence gaps and open questions

Service-level ownership remains unconfirmed outside `.github/CODEOWNERS`.
