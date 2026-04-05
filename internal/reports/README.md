# internal/reports

Пакет компилирует и материализует workspace-отчёты ACP:
- `reports/as-is/*`
- `reports/findings/*`
- `reports/coverage/*`
- `reports/agent-outputs/*`

Детерминированный scope для strict golden compare определяется в `README.md` и `docs/TESTING_STRATEGY.md`.
Run-specific артефакты (`reports/taskruns/*`, `reports/changelog/*`) исключены из strict snapshot compare.
