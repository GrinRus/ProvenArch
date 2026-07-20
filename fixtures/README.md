# Fixtures

Этот каталог хранит baseline regression surface для ACP MVP.

Текущие baseline fixtures:
- `fixtures/workspace/valid-path.yaml`
- `fixtures/workspace/valid-git-url.yaml`
- `fixtures/workspace/valid-with-runtime-timeouts.yaml`
- `fixtures/workspace/invalid-both.yaml`
- `fixtures/workspace/invalid-neither.yaml`

Целевая структура:
- `fixtures/workspace/` — manifest и validator cases
- `fixtures/scenarios/` — scenario integration inputs и golden outputs

Baseline scenario surface:
- `single-service-http-postgres-gitlabci`
- `two-services-http-call-and-queue`
- `missing-owner-and-missing-cicd`
- `refresh-artifact-quality` — reduced frozen refresh artifact sets (`bank-collapse`, `openstack-rich-reuse`) for artifact-fidelity regression checks
- `validator-duplicate-claim` — reduced staged final surfaces for deterministic duplicate-`claim_id` repair checks

Generated artifacts policy:
- `fixtures/scenarios/*/golden/readable/*` намеренно tracked в git как human-readable deterministic export.
- Эти файлы используются для review diffability и не считаются случайными артефактами.

Required CI использует только local fixtures, synthetic repos и recorded artifacts.
Live headless provider runs в этом контуре не требуются.

Provider-free runtime contract fixtures under `internal/runtime/testdata/contract-rejection/`
include reduced authored Architecture Home examples. The concrete-evidence case records invalid
repository-root shorthand and wildcard references; tests reject them without importing matrix or
live-harness state and retain exact existing file/directory references as the valid baseline.
The step2 first-pass case separately records runtime/recovery narration in an architect summary;
tests keep that content invalid while allowing equivalent operator-facing architecture and gap text.
