# Спецификация пайплайна (MVP v0)

Документ описывает pipeline ACP через input/output контракты и expected artifacts.

## Общие понятия

- **Workspace**: единый central git-репозиторий `arch-workspace/` (каноническая MVP-конвенция, Variant 2) с `workspace.yaml`, `charter/`, `skills/`, `model/`, `reports/`, `proposals/`, `docs/`.
- **Orchestrator**: управляет шагами, готовит PromptPack/ContextPack, вызывает runtime, валидирует TaskResult.
- **Runtime (MVP)**: Claude Code headless.
- **TaskResult**: структурированный JSON ответа runtime-шага (`schemas/taskresult.schema.json`).

> Несмотря на schema-гибкость, MVP policy фиксирует runtime как `claude-code`.

## Charter и Skills (MVP format)

### Charter
Хранится в `charter/`, минимально:
- `charter/overview.md`
- `charter/glossary.yaml`
- `charter/nfr.yaml`
- `charter/rules.yaml`
- `charter/templates/`

### Skills
Хранятся в `skills/` и редактируются в UI:

```text
skills/<skill_name>/
  manifest.yaml
  prompts/
    system.md
    task.md
  templates/
    adr.md
    rfc.md
```

`manifest.yaml` (minimum):
- `name`
- `version`
- `applies_to`
- `inputs`
- `outputs`

Рекомендуется отдельный mapping: `skills/subagents.yaml`.

## Canonical step IDs

### Init pipeline
- `init.step0.constitution`
- `init.step1.collect`
- `init.step2.asis_docs`
- `init.step3.findings`
- `init.step4.proposals`

### Refresh pipeline (manual)
- `refresh.step1.collect`
- `refresh.step2.asis_docs`
- `refresh.step3.findings`
- `refresh.step4.proposals`

## Init pipeline

### Step 0 — Constitution (human-in-the-loop)
Inputs:
- шаблоны charter
- пользовательские правки в UI

Outputs:
- обновлённые `charter/*`

### Step 1 — Collect context (runtime step)
Inputs:
- `workspace.yaml` из корня central `arch-workspace`
- локальные репозитории
- `docs/imports/*`, `docs/rfcs/*`, `docs/meetings/*`, `docs/decisions/*`
- `charter/*`
- `skills/*`

Runtime output (TaskResult):
- `changeset`: `upsert_entity`, `upsert_edge`, optional `add_doc_artifact`
- optional `questions` и/или `add_question`
- optional `coverage` и/или `set_coverage`

Orchestrator applies:
- обновляет `model/*`
- сохраняет taskrun under `reports/taskruns/*`
- формирует coverage/questions artifacts (planned)

### Step 2 — As-is docs (compiler step in MVP)
Inputs:
- `model/*`
- `charter/*`
- `skills/templates/*`

Outputs:
- `reports/as-is/overview.md`
- `reports/as-is/service-catalog.md`
- `reports/as-is/dependencies.md` (optional)

> В MVP это детерминированная компиляция из модели.

### Step 3 — Findings (runtime step)
Inputs:
- `model/*`
- `charter/rules.yaml`
- `skills/*`

Runtime output (TaskResult):
- `changeset`: `add_finding`
- optional `questions`/`add_question`
- optional `coverage`/`set_coverage`

Orchestrator applies:
- обновляет `reports/findings/*`

### Step 4 — Proposals (compiler/templates in MVP)
Inputs:
- `model/*`
- `reports/findings/*`
- `charter/*`
- `skills/templates/*`

Outputs:
- `proposals/<proposal-id>/proposal.md`
- `proposals/<proposal-id>/ADR.md`
- `proposals/<proposal-id>/RFC.md`
- `proposals/<proposal-id>/migration-checklist.md`

> В MVP proposals формируются без write-file операции в TaskResult, через orchestrator templates/compiler.

## Нефункциональные требования (MVP)

- детерминированность на одинаковых входах
- безопасный filesystem scope (без выхода за workspace root)
- runtime предлагает, человек подтверждает спорные решения
- запись только в workspace, не в пользовательские репозитории
