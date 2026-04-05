# GitLab trigger-mode templates (MVP)

Этот каталог содержит reference-шаблон для запуска ACP в GitLab CI без hosted control plane.

## Что покрывает шаблон
- auto-trigger на `push` в default branch
- manual trigger через UI (`CI_PIPELINE_SOURCE=web`)
- batch mode через CLI (`acp run --non-interactive`)
- deterministic runtime default (`--runtime fake`) для required CI surface

## Файл
- `push-and-manual.gitlab-ci.yml`

Перед использованием задайте `ACP_WORKSPACE` (absolute path к workspace checkout внутри job).
