# Fixtures

`fixtures/api/source-qa-answer.json` is the closed provenance fixture for the explicit
Ask-to-Proposal mutation. It validates with `schemas/source-qa-answer.schema.json`.

Provider-free artifact integrity incidents are minimized as Go-owned temporary workspace fixtures
in `internal/artifactaudit/audit_test.go`. They cover clean exact-run evidence, foreign identity,
oversized artifacts, execution/scaffold contamination, missing repository evidence and
promoted-current digest drift. The generated reports contain no absolute source path or raw provider
output and repeated scans must be byte-identical.

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
  Scenario baseline maintainers own these review diffs; `make verify-readable-fixtures` rejects
  orphan paths or digest drift against each adjacent `golden/snapshot.sha256`.
- Эти файлы используются для review diffability и не считаются случайными артефактами.

Required CI использует только local fixtures, synthetic repos и recorded artifacts.
Live headless provider runs в этом контуре не требуются.

Selective-refresh integrity fixtures are generated in
`internal/orchestrator/refresh_baseline_integrity_test.go`: they cover immutable staged bytes,
source-range/full-scope identity, altered digests and missing/unsafe trees without freezing an
internal sidecar as a public contract example.

Session/Git admission fixtures are deterministic in `internal/api/server_test.go`: a barrier pauses
run admission at its commit point while a workspace switch waits, and active-run Git tests assert a
typed conflict plus unchanged HEAD. Existing JSON payload fixtures remain unchanged.

Typed recovery incident fixtures live in
`internal/runtime/providercommon/validation_issues_test.go`: the same mixed issue set is shuffled
for Claude/Qwen/Codex, paths are sorted/deduplicated and a repeated transition exhausts its explicit
one-attempt budget.

Provider-free runtime contract fixtures under `internal/runtime/testdata/contract-rejection/`
include reduced authored Architecture Home examples. The concrete-evidence case records invalid
repository-root shorthand and wildcard references; tests reject them without importing matrix or
live-harness state and retain exact existing file/directory references as the valid baseline.
The step2 first-pass case separately records runtime/recovery narration in an architect summary;
tests keep that content invalid while allowing equivalent operator-facing architecture and gap text.
The normal step2 prompt-contract regression also remains provider-free: it forbids inline language
generators and nested quoting before the complete write set exists, and requires one bounded read
followed by direct single-quoted heredoc writes. Raw live command transcripts are diagnostic inputs,
not tracked product fixtures or production behavior switches.
