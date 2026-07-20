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

Provider-free contract regressions могут хранить минимальную authored artifact pair без полного
workspace/golden дерева. `collect-manifest-wrong-artifact-root` фиксирует structurally valid collect
manifest с foreign task identity и существующим authored document; runtime обязан reject-нуть его
до downstream materialization и разрешает только обычный provider-authored repair.

Required CI использует только synthetic scenarios, staged manifests/verdicts и golden snapshots.
Live headless provider runs в этом контуре не требуются.
