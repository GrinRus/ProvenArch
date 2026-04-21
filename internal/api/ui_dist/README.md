# internal/api/ui_dist/

Каталог хранит tracked embed-ассеты UI (`ui/dist`) для `acp serve`.

Policy:
- Артефакты в `internal/api/ui_dist` намеренно versioned в git.
- Обновление выполняется через `make build` (clean UI build + копирование в `internal/api/ui_dist`).
- Эти файлы не считаются “случайным generated мусором”; это release surface для embedded UI.
