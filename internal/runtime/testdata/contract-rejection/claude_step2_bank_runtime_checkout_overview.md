# System Overview

## System at a glance

Bank of Anthos is a cloud-native banking application. The repository under `/tmp/provenarch-live/arch-workspace/.acp/repos/bank-of-anthos-7fb01b96709b` contains its source. The current analysis covers the codebase as staged in the current run, with evidence drawn from the typed shard plan/summary and shard-pack manifests.

## Analyzed scope

The current run analyzed `/tmp/provenarch-live/arch-workspace/.acp/repos/bank-of-anthos-7fb01b96709b/src`.

## Domains and ownership

Frontend ownership evidence is under `/tmp/provenarch-live/arch-workspace/.acp/repos/bank-of-anthos-7fb01b96709b/src/frontend`.

## Key flows

Based on the staged source and manifest structure, the frontend calls ledger services.

## Integrations and datastores

PostgreSQL deployment evidence is recorded in `kubernetes-manifests/`.

## Where to start

Start with `README.md` and `skaffold.yaml`.

## Safe-change guidance

Validate service changes against `skaffold.yaml`.

## Evidence gaps and open questions

Shard completeness for this run is planned=10 succeeded=10 failed=0 incomplete=0.
