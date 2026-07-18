# Architecture Home

## System at a glance
Deployment evidence is under `bank-of-anthos:kubernetes-manifests/`.

## Analyzed scope
Build evidence includes `bank-of-anthos:pom.xml` and the absent `bank-of-anthos:cloudbuild.yaml`.

## Domains and ownership
Accounts live under `bank-of-anthos:src/accounts/`; the claimed `bank-of-anthos:src/user-service/` path is absent.

## Key flows
Start at `bank-of-anthos:README.md`.

## Integrations and datastores
The claimed manifests `bank-of-anthos:src/accounts/pom.xml` and `bank-of-anthos:src/ledger/pom.xml` are absent.

## Where to start
Read `bank-of-anthos:README.md`.

## Safe-change guidance
Validate deployment changes against `bank-of-anthos:kubernetes-manifests/`.

## Evidence gaps and open questions
Unconfirmed interfaces remain explicit gaps.
