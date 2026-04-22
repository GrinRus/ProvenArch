# internal/runtime/claudecode

Пакет содержит runtime adapters для orchestrator:
- `HeadlessRunner` — opt-in локальный запуск Claude Code headless;
- `FakeRunner` — deterministic default для required CI без live dependencies и локальных artifact fixtures.

Runtime policy в beta:
- process-scoped selector `fake|headless`;
- `fake` обязателен для required CI;
- `headless` используется только как opt-in.

Scenario/golden tests в текущем artifact-only контуре опираются на `FakeRunner` и fixture artifacts; отдельный recorded-runner seam больше не поддерживается.
