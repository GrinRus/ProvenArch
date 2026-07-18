# System Overview

## System at a glance

Bank of Anthos is a cloud-native banking application documented in `bank-of-anthos:README.md`.

## Analyzed scope

Current-run coverage includes the following shard packs:

- `shard-pack-manifest.json` — 0 file(s)

Typed shard collection status: planned=10 succeeded=10 failed=0 incomplete=0. Current-run shard coverage is not a blocker.

## Domains and ownership

Service directories are under `bank-of-anthos:src/`.

## Key flows

The frontend calls ledger services under `bank-of-anthos:src/ledger/`.

## Integrations and datastores

PostgreSQL deployment evidence is under `bank-of-anthos:kubernetes-manifests/`.

## Where to start

Start with `bank-of-anthos:README.md` and `bank-of-anthos:skaffold.yaml`.

## Safe-change guidance

Validate service changes against `bank-of-anthos:skaffold.yaml`.

## Evidence gaps and open questions

Exact service dependency edges remain unconfirmed.
