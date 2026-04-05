# internal/orchestrator

Пакет реализует orchestration pipeline `init|refresh`:
- sequencing шагов (Step 0..4 для `init`, Step 1..4 для `refresh`);
- run control (single active run + debounce queue policy);
- вызов runtime runner и обработку TaskResult;
- materialization артефактов (`model/`, `reports/`, `proposals/`).

Контракт и поведение синхронизируются с:
- `docs/spec/PIPELINE_SPEC.md`
- `docs/ARCHITECTURE.md`
- `docs/STAKEHOLDER_DOC.md` (Canonical Stakeholder Matrix)
