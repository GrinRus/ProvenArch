# Local Full-Run: scenario profile (`ai_advent`-like)

Этот runbook описывает воспроизводимый полный прогон ProvenArch «как пользователь» в временном `tmp` workspace (`/tmp`) против целевого репозитория.

Сценарий используется и для первого локального запуска, и для итеративного цикла улучшений backend/frontend.

## 1) Что проверяет сценарий

- Bootstrap workspace в `tmp` через `acp init-workspace`.
- Автоматический `git init` для workspace root при отсутствии `.git`.
- API simulation: `validate -> init -> artifacts`.
- API simulation run logs: `GET /api/pipeline/runs/<run_id>/logs`.
- Полный runtime цикл на каждой итерации:
  - `init/refresh` в `fake`
  - `init/refresh` в `headless`
- Quality guardrails для headless:
  - запрет mock/fake runtime version,
  - fail при zero-signal quality summary,
  - fail при регрессии сигнала относительно предыдущей итерации для той же пары `(runtime_mode, pipeline)`,
  - fail при `No findings reported` в headless refresh при owner-related gaps в coverage,
  - fail при canonical duplicates в `coverage.missing`,
  - fail при duplicate open-question texts после нормализации,
  - ai-advent profile checks для минимально содержательного сигнала.
- Snapshot artifacts per run: `TMP_ROOT/snapshots/<run_id>/...`.
- Проверка ключевых артефактов (`as-is/findings/coverage`) и run quality summaries.
- (Опционально) quality gates: `make contracts`, `make test`, `make lint`, `make build`.

## 2) Переменные скрипта

`./scripts/full-run-ai-advent.sh` поддерживает:

- `PROVENARCH_ROOT` (default: текущий repo ProvenArch)
- `TARGET_REPO` (required: path to the repository used for full-run)
- `TMP_ROOT` (default: auto `mktemp -d -t provenarch-ai-advent.XXXXXX`)
- `ACP_RUNTIME_PROVIDER` (headless provider: `claude-code` default или `qwen-code`)
- `ACP_CLAUDE_CMD` (команда для provider `claude-code`; default `claude-code`, поддержан direct `claude` без wrapper)
- `ACP_QWEN_CMD` (команда для provider `qwen-code`; default `qwen`)
- `KEEP_TMP` (`0/1`, default `0`)
- `ITERATIONS` (default `1`)
- `RUN_QUALITY_GATES` (`0/1`, default `1`)
- `RUN_LOGS_TTL_HOURS` (default `168`)
- `RUN_LOGS_MAX_RUNS` (default `200`)

## 3) Быстрый запуск (script)

```bash
cd /path/to/ProvenArch

# Вариант 1: default headless provider (claude-code в PATH)
TARGET_REPO=/path/to/target-repo ./scripts/full-run-ai-advent.sh

# Вариант 2: direct claude без wrapper
TARGET_REPO=/path/to/target-repo ACP_RUNTIME_PROVIDER=claude-code ACP_CLAUDE_CMD=claude ./scripts/full-run-ai-advent.sh

# Вариант 3: явно задать provider=qwen-code
TARGET_REPO=/path/to/target-repo ACP_RUNTIME_PROVIDER=qwen-code ./scripts/full-run-ai-advent.sh

# Вариант 4: явно задать команду для выбранного provider
TARGET_REPO=/path/to/target-repo ACP_RUNTIME_PROVIDER=qwen-code ACP_QWEN_CMD=/abs/path/to/qwen ./scripts/full-run-ai-advent.sh

# Вариант 5: оставить tmp workspace для ручного анализа
TARGET_REPO=/path/to/target-repo KEEP_TMP=1 ./scripts/full-run-ai-advent.sh
```

Script всегда формирует:
- `TMP_ROOT/full-run.log`
- `TMP_ROOT/session-summary.md`
- `TMP_ROOT/snapshots/<run_id>/...`

При ошибке script всегда сохраняет `TMP_ROOT` для дебага независимо от `KEEP_TMP`.

## 4) CLI/API поток вручную (без скрипта)

```bash
cd /path/to/ProvenArch
make build

TMP_ROOT="$(mktemp -d -t provenarch-ai-advent.XXXXXX)"
WORKSPACE="$TMP_ROOT/arch-workspace"

./bin/acp init-workspace \
  --workspace "$WORKSPACE" \
  --repo-name ai-advent \
  --repo-path /path/to/target-repo

# API simulation
PORT=18080
./bin/acp serve --workspace "$WORKSPACE" --runtime fake --listen "127.0.0.1:$PORT"
# отдельным терминалом:
# curl -X POST http://127.0.0.1:$PORT/api/workspace/validate
# curl -X POST -H 'Content-Type: application/json' -d '{"trigger":"manual"}' http://127.0.0.1:$PORT/api/pipeline/init
# curl http://127.0.0.1:$PORT/api/pipeline/runs/<run_id>
# curl http://127.0.0.1:$PORT/api/pipeline/runs/<run_id>/artifacts

# Runtime cycle: fake + headless
./bin/acp run --workspace "$WORKSPACE" --pipeline init --runtime fake --non-interactive
./bin/acp run --workspace "$WORKSPACE" --pipeline refresh --runtime fake --non-interactive
./bin/acp run --workspace "$WORKSPACE" --pipeline init --runtime headless --runtime-provider claude-code --non-interactive
./bin/acp run --workspace "$WORKSPACE" --pipeline refresh --runtime headless --runtime-provider claude-code --non-interactive
```

Ключевые артефакты после прогона:

- `reports/as-is/overview.md`
- `reports/findings/findings.md`
- `reports/coverage/open-questions.md`

## 5) Manual UI checklist

После запуска `acp serve --workspace ...` откройте UI и проверьте минимум по секциям.

### Setup

- `Setup: Workspace`: виден guided setup для `repos[]`.
- Можно добавить/удалить repo row (`name + path|git_url + ref`).
- `Validate workspace` возвращает `resolved_repos` и diagnostics.
- Diagnostics сгруппированы по repo.

### Baseline

- `Baseline: Editors`: можно открыть/сохранить baseline artifacts.
- `Setup: Step 0 Wizard Contract`: сохраняется `charter/wizard/step0-contract.json`.

### Runs

- `Runs: Pipeline Control`: запускаются `Run init` и `Run refresh`.
- `Runs: History`: отображаются queued/running/succeeded/failed и выбор run.
- `Runs: Logs`: для выбранного run отображается live stream с полями `timestamp/level/step/domain/message`.
- В `Runs: Logs` работают quick actions:
  - `Copy logs`
  - `Download logs`
  - `Open taskrun artifact` (если путь найден в log events)

### Results

- `Results: Coverage & Questions`: показываются coverage/open questions.
- `Results: Run Artifacts`: можно открыть артефакты run.

## 6) Continuous Improvement Loop (balanced backend/frontend)

Используйте script как базовый повторяемый цикл.

1. Запустить полный прогон (`full-run-ai-advent.sh`) в новом `tmp` workspace.
2. Снять findings и разложить по корзинам:
   - backend
   - frontend
3. Приоритизировать задачи: P1/P2 first, без перекоса в один слой.
4. Внести изменения в код.
5. Повторить полный прогон script'ом с нуля.
6. Повторять до условия остановки.

Критерий остановки:

- зелёные `make contracts`, `make test`, `make lint`, `make build`
- отсутствие P1/P2 находок по последнему полному прогону;
- headless runs проходят strict quality checks (non-mock, non-zero-signal, no degradation).
- headless refresh проходит semantic checks (owner-gap+findings, coverage/question dedupe).

## 7) Диагностика типовых проблем

- Ошибка `headless runtime command ... is unavailable`:
  - проверить `ACP_RUNTIME_PROVIDER`;
  - для `claude-code`: установить `claude-code` или использовать direct `ACP_CLAUDE_CMD=claude` (либо задать `ACP_CLAUDE_CMD=/abs/path/to/runner`);
  - для `qwen-code`: установить `qwen` или задать `ACP_QWEN_CMD=/abs/path/to/runner`.
- Ошибки bootstrap (`workspace.yaml/.git/skills/subagents.yaml` не созданы):
  - проверить вывод `logs/init-workspace.log`.
- API run timeout:
  - проверить `logs/serve-fake.log` и `api-init-status.json`.
- Quality regression / zero-signal fail:
  - проверить `session-summary.md` (секция failure reason),
  - сравнить `snapshots/<run_id>/...` между последними run,
  - посмотреть `reports/taskruns/<run_id>-quality.json`.
- Провал quality gates:
  - проверить `quality-gates.log`.
