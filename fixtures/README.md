# Fixtures

Этот каталог хранит baseline regression surface для ACP MVP.

Текущие baseline fixtures:
- `fixtures/workspace/valid-path.yaml`
- `fixtures/workspace/valid-git-url.yaml`
- `fixtures/workspace/invalid-both.yaml`
- `fixtures/workspace/invalid-neither.yaml`
- `fixtures/taskresult/normalized-top-level.json`

Целевая структура:
- `fixtures/workspace/` — manifest и validator cases
- `fixtures/taskresult/` — raw/normalized TaskResult cases
- `fixtures/scenarios/` — scenario integration inputs и golden outputs

Baseline scenario surface:
- `single-service-http-postgres-gitlabci`
- `two-services-http-call-and-queue`
- `missing-owner-and-missing-cicd`

Generated artifacts policy:
- `fixtures/scenarios/*/golden/readable/*` намеренно tracked в git как human-readable deterministic export.
- Эти файлы используются для review diffability и не считаются случайными артефактами.

Required CI использует только local fixtures, synthetic repos и recorded runner artifacts.
Live Claude Code runs в этом контуре не требуются.
