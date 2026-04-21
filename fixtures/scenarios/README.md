# Scenario Fixtures

Этот каталог фиксирует baseline scenario surface для integration/golden tests.

Стандартная структура сценария:

```text
fixtures/scenarios/<name>/
  workspace/
  repos/
    <repo-name>/
  golden/
```

Где:
- `workspace/` содержит central workspace inputs
- `repos/` содержит synthetic repos
- `golden/` содержит ожидаемые `model/`, `reports/`, `proposals/`, `changelog`

Baseline scenarios для MVP:
- `single-service-http-postgres-gitlabci`
- `two-services-http-call-and-queue`
- `missing-owner-and-missing-cicd`
- `refresh-artifact-quality`
- `validator-duplicate-claim`

Required CI использует только synthetic scenarios, staged manifests/verdicts и golden snapshots.
Live Claude Code runs в этом контуре не требуются.
