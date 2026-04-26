# internal/runtime/claudecode

Пакет содержит runtime adapters для orchestrator:
- `HeadlessRunner` — opt-in локальный запуск Claude Code headless.

Runtime policy в beta:
- process-scoped selector `fake|headless`;
- `fake` обязателен для required CI;
- `headless` используется только как opt-in.

Deterministic `fake` runtime живёт в provider-neutral `internal/runtime/fakeruntime` и используется вместе с artifact fixtures; отдельный recorded-runner seam больше не поддерживается.
