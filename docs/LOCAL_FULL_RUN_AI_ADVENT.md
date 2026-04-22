# Local Full-Run: scenario/full-run profile

Этот документ описывает только `scenario/full-run` convenience flow для локальной итерации.
Он не является canonical release runbook и не должен использоваться как источник `release verdict`.

Для strict pre-release gate используйте отдельный trusted machine runbook:
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`

Shared defaults, timeout profiles, matrix taxonomy и полный каталог batch/matrix env vars намеренно не дублируются здесь.
Source of truth для этих частей:
- `README.md`
- `docs/TESTING_STRATEGY.md`

## 1) Когда использовать этот flow

- нужен полный локальный цикл на временном `tmp` workspace;
- требуется прогнать `API simulation + runtime fake + runtime headless`;
- нужен repeatable debug loop с `full-run.log`, `session-summary.md` и snapshot-артефактами;
- нужно быстро проверить local изменения без перехода в release gate.

Этот flow не заменяет release path и не даёт `PASS|FAIL` решения для релиза.

## 2) Что здесь уникально относительно других runbook'ов

`./scripts/full-run-ai-advent.sh` делает именно локальный scenario/full-run цикл:
- bootstrap workspace в `tmp`;
- API simulation (`validate -> init -> artifacts`);
- runtime итерации `fake + headless`;
- optional quality gates (`make contracts`, `make test`, `make lint`, `make build`);
- guaranteed debug artifacts:
  - `TMP_ROOT/full-run.log`
  - `TMP_ROOT/session-summary.md`
  - `TMP_ROOT/snapshots/<run_id>/...`

Boundary с batch/matrix flow:
- canonical matrix entry использует `E2E_MATRIX_FILE`;
- approved profile ids для matrix docs/tests: `single-path`, `single-git_url`, `multi-path`, `multi-git_url`;
- batch-only semantic checks вроде `analysis:cross-repo-missing` относятся к matrix/batch flow и описаны в `docs/TESTING_STRATEGY.md`;
- strict release guidance и `release verdict` остаются только в `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.

## 3) Canonical references

- `README.md`
  Используйте для shared examples, quickstart и developer entrypoints.
- `docs/TESTING_STRATEGY.md`
  Используйте как source of truth для `E2E_MATRIX_FILE`, timeout profiles, matrix taxonomy, batch semantics и required CI/test boundaries.
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md`
  Используйте для trusted machine release flow, canonical release slices и единственного допустимого release verdict.

## 4) Minimal inputs и env vars

`./scripts/full-run-ai-advent.sh` требует и/или поддерживает только scenario-relevant набор:

- `TARGET_REPOS_FILE`
  Канонический input; обязателен. Формат совпадает с `repos[]` в `workspace.yaml`.
- `PROVENARCH_ROOT`
  Опционально; default — текущий repo ProvenArch.
- `ACP_RUNTIME_PROVIDER`
  Опционально; `claude-code` default, `qwen-code` или `codex-code`.
- `ACP_CLAUDE_CMD`
  Опционально; override команды для `claude-code`.
- `ACP_QWEN_CMD`
  Опционально; override команды для `qwen-code`.
- `ACP_CODEX_CMD`
  Опционально; override команды для `codex-code`.
- `KEEP_TMP`
  `0/1`; оставить временный workspace для ручного анализа.
- `ITERATIONS`
  Число полных циклов.
- `RUN_QUALITY_GATES`
  `0/1`; включать ли `make contracts/test/lint/build`.
- `RUN_LOGS_TTL_HOURS`, `RUN_LOGS_MAX_RUNS`
  Retention knobs для runtime run logs.

Shared timeout/env/matrix knobs намеренно не перечисляются в этом документе повторно.
Используйте:
- `README.md` для quick examples;
- `docs/TESTING_STRATEGY.md` для canonical timeout/env semantics;
- `docs/RELEASE_LIVE_E2E_RUNBOOK.md` для release-oriented matrix settings.

## 5) Быстрый запуск

Базовый local full-run:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml ./scripts/full-run-ai-advent.sh
```

Claude direct:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_RUNTIME_PROVIDER=claude-code \
ACP_CLAUDE_CMD=claude \
./scripts/full-run-ai-advent.sh
```

Qwen:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_RUNTIME_PROVIDER=qwen-code \
./scripts/full-run-ai-advent.sh
```

Codex:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml \
ACP_RUNTIME_PROVIDER=codex-code \
ACP_CODEX_CMD=codex \
./scripts/full-run-ai-advent.sh
```

Сохранить временный workspace:

```bash
TARGET_REPOS_FILE=/abs/path/to/repos.yaml KEEP_TMP=1 ./scripts/full-run-ai-advent.sh
```

Matrix/batch запуск не описывается здесь детально; canonical reference:

```bash
E2E_MATRIX_FILE=/abs/path/to/e2e-matrix.yaml \
ACP_CLAUDE_CMD_BIN=claude \
ACP_QWEN_CMD_BIN=qwen \
ACP_CODEX_CMD_BIN=codex \
./scripts/full-run-batch-matrix.sh
```

Для matrix taxonomy, `single-path|single-git_url|multi-path|multi-git_url`, timeout profiles и release-mode используйте `README.md` и `docs/TESTING_STRATEGY.md`.

## 6) Output semantics и boundary

Script всегда materialize-ит:
- `TMP_ROOT/full-run.log`
- `TMP_ROOT/session-summary.md`
- `TMP_ROOT/snapshots/<run_id>/...`

При runtime failure raw-output diagnostics остаются в workspace taskrun surface:
- `reports/taskruns/raw/*`

Scenario/full-run boundary:
- документ описывает convenience flow для локальной разработки;
- release verdict отсюда не принимается;
- если нужен `PASS|FAIL` gate, переключайтесь на `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.

## 7) Continuous Improvement Loop

1. Запустить `full-run-ai-advent.sh` в новом `tmp` workspace.
2. Разобрать findings и quality/debug artifacts.
3. Внести изменения в backend/frontend.
4. Повторить полный прогон.
5. Останавливать цикл только после зелёных DoD-checks и прохождения локальных quality constraints.

## 8) Диагностика

- `headless runtime command ... is unavailable`
  Проверьте `ACP_RUNTIME_PROVIDER`, `ACP_CLAUDE_CMD`, `ACP_QWEN_CMD`, `ACP_CODEX_CMD`.
- bootstrap/layout failures
  Смотрите `full-run.log` и `logs/init-workspace.log`.
- timeout/hang
  Сверяйте `session-summary.md`, `full-run.log` и effective timeout values из `README.md` / `docs/TESTING_STRATEGY.md`.
- quality regression
  Сравнивайте `snapshots/<run_id>/...` и `reports/taskruns/<run_id>-quality.json`.

Если проблема относится к batch/matrix orchestration или release-mode semantics, этот документ уже не является source of truth:
- переходите в `README.md`, `docs/TESTING_STRATEGY.md` и `docs/RELEASE_LIVE_E2E_RUNBOOK.md`.

## 9) Owner follow-up after cleanup slice

Этот документ intentionally сохранён в cleanup-slice как convenience runbook, а не как canonical release surface.

Отдельный owner-confirmation follow-up после merge всё ещё нужен:
- оставить ли `docs/LOCAL_FULL_RUN_AI_ADVENT.md` отдельным документом;
- либо схлопнуть его дальше в pointer/appendix после подтверждения, что unique local full-run delta покрыт другими canonical runbook'ами.
