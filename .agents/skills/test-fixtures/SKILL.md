---
name: acp-test-fixtures
description: Подобрать и обновить deterministic regression fixtures для core/model, artifact validation, extraction и materialization ACP, включая readable golden при изменении outputs.
---

## Выбрать существующую поверхность

Прочитать [fixtures README](../../../fixtures/README.md) и при scenario change —
[scenario README](../../../fixtures/scenarios/README.md). Пути ниже считаются от корня репозитория.

| Поведение | Где проверять |
| --- | --- |
| Workspace/schema inputs | `fixtures/workspace`, `fixtures/tasks`, `fixtures/api`; tests в `internal/workspace`, `internal/tasks`, `internal/contracts` |
| Materialized model/reports/proposals | `fixtures/scenarios` и existing tests в `internal/model`, `internal/reports`, `internal/orchestrator` |
| Runtime artifact rejection или repair | Минимальные artifacts в `internal/runtime/testdata/contract-rejection` или reduced `fixtures/scenarios`; relevant contract/adapter tests |
| Race/restart/path/refresh integrity | Existing Go-owned temporary fixtures в tests владельца; не замораживать internal sidecar как новый public example |

Добавить наблюдаемый regression case, который защищает изменённый invariant и отличается от valid
baseline. Для небольшого contract rejection достаточно минимального artifact; полный synthetic
repository/workspace и golden нужны только когда результат действительно зависит от них.

## Golden и evidence

- При изменении materialized outputs проверить diff ожидаемой семантики. Tracked
  `fixtures/scenarios/*/golden/readable/*` — human-readable export, связанный с `golden/snapshot.sha256`.
  Обновлять их согласованно, не принимать новый digest без просмотра output diff.
- Live incident минимизировать в локальный deterministic input; raw provider transcript, credentials,
  absolute user paths и private source content не переносить в tracked fixtures.
- Synthetic source repos остаются входом теста: test execution не должен их менять. Использовать
  temporary workspace для generated outputs.

Сначала запустить existing focused package/test через `./scripts/run-go.sh test`, для schema fixtures
также `make contracts-check`, для readable golden — `make verify-readable-fixtures`.
Required tests работают без live provider/network execution. Полный implementation DoD и дополнительные
checks находятся в [AGENTS](../../../AGENTS.md) и [TESTING_STRATEGY](../../../docs/TESTING_STRATEGY.md).
