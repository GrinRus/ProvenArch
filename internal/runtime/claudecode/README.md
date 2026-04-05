# internal/runtime/claudecode

Пакет содержит runtime adapters для orchestrator:
- `HeadlessRunner` — opt-in локальный запуск Claude Code headless;
- `FakeRunner` — deterministic default для required CI без live dependencies;
- `RecordedRunner` — replay recorded fixtures для scenario/golden тестов.

Runtime policy в beta:
- process-scoped selector `fake|headless`;
- `fake` обязателен для required CI;
- `headless` используется только как opt-in.
