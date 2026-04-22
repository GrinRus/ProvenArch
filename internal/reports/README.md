# internal/reports

Пакет реализует deterministic compile/materialize layer для workspace-отчётов ACP:
- `reports/as-is/*`
- `reports/findings/*`
- `reports/coverage/*`
- `reports/agent-outputs/*`
- `reports/diagrams/*` + `reports/diagrams/index.md`

Дополнительно пакет держит render-context helpers для incomplete/evidence-aware narrative surfaces и C4 Mermaid builders для derived diagram exports.

Детерминированный scope для strict golden compare определяется в `README.md` и `docs/TESTING_STRATEGY.md`.
Run-specific артефакты (`reports/taskruns/*`, `reports/changelog/*`) исключены из strict snapshot compare.
