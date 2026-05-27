# ADR-20260421-codex-runtime-provider-expansion.md

- **ADR ID:** ADR-20260421-codex-runtime-provider-expansion
- **Status:** accepted
- **Date:** 2026-04-21
- **Owners:** ACP maintainers
- **Supersedes:** none
- **Amends:** [ADR-20260410-headless-runtime-multi-provider](ADR-20260410-headless-runtime-multi-provider.md)

## Context

После ADR-20260410 MVP runtime policy уже был зафиксирован как `fake` baseline + headless multi-provider surface с двумя production providers:
- `claude-code` (default fallback)
- `qwen-code`

Новый release/live gate slice требует добавить третий provider `codex-code` без смены default provider, без hosted-mode и без расширения required CI на live network dependencies.

Одновременно нужно сохранить:
- deterministic `fake` baseline для required CI;
- precedence `workspace step override > CLI/env global provider > claude-code`;
- artifact-only runtime contract и существующие error codes;
- canonical 5-profile live taxonomy с invariant `baseline == parallel-default` по shard-plan для одного `profile_id`.

## Decision

ACP MVP runtime policy расширяется до трёх headless providers:
- `claude-code` (default fallback)
- `qwen-code`
- `codex-code`

Дополнительные решения:
- `codex-code` становится full release peer для manual trusted-host release gate;
- canonical release acceptance теперь требует `qwen + claude + codex`;
- regression profiles остаются qwen-only, а claude/codex regression runs остаются manual diagnostics через `BATCH_PROVIDER_FILTER=<provider>`;
- provider command envs фиксируются как:
  - `ACP_CLAUDE_CMD` (default `claude-code`)
  - `ACP_QWEN_CMD` (default `qwen`)
  - `ACP_CODEX_CMD` (default `codex`)
- live batch harness/preflight command envs фиксируются как:
  - `ACP_CLAUDE_CMD_BIN` (default `claude`)
  - `ACP_QWEN_CMD_BIN` (default `qwen`)
  - `ACP_CODEX_CMD_BIN` (default `codex`)

## Rationale

- `codex-code` нужен как отдельный production runtime surface для trusted-machine live validation, а не как замена `claude-code`.
- Сохранение `claude-code` как default fallback минимизирует churn в CLI/API/workspace semantics.
- Release gate должен проверять всех release peers симметрично, иначе matrix verdict перестаёт быть canonical readiness signal.
- Regression profiles остаются qwen-only, чтобы быстрый diagnostic контур не раздувался по времени и operational cost.

## Consequences

- Runtime/config/docs/schema surfaces должны принимать `codex-code` везде, где ранее были допустимы `claude-code|qwen-code`.
- Live harness/reporting расширяется инкрементально через явные поля `runtimes.codex` и `frontend_codex_status`, без большого generic refactor. Frontend cancellation больше не является live release-gate signal.
- Release catalog totals пересчитываются:
  - `release-fast=12`
  - `release-long=12`
  - `release-full=36`
- Required CI по-прежнему использует only `fake` / fixtures / recorded outputs и не зависит от наличия `codex`.

## Follow-ups

- Любое следующее расширение provider list beyond `claude-code|qwen-code|codex-code` требует нового отдельного slice/ADR.
- Если позже появится hosted/runtime-routing mode, он должен рассматриваться отдельно от current local-first artifact-only contract.
