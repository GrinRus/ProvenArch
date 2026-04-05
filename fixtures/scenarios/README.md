# Scenario Fixtures (planned)

Этот каталог фиксирует baseline scenario surface для integration/golden tests.

Стандартная структура сценария:

```text
fixtures/scenarios/<name>/
  workspace/
  repos/
    <repo-name>/
  runner/
  golden/
```

Где:
- `workspace/` содержит central workspace inputs
- `repos/` содержит synthetic repos
- `runner/` содержит recorded raw TaskResult per step
- `golden/` содержит ожидаемые `model/`, `reports/`, `proposals/`, `changelog`

Baseline scenarios для MVP:
- `single-service-http-postgres-gitlabci`
- `two-services-http-call-and-queue`
- `missing-owner-and-missing-cicd`

Required CI использует только synthetic scenarios и recorded runner artifacts.
Live Claude Code runs в этом контуре не требуются.
