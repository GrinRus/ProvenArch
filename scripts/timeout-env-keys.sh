#!/usr/bin/env bash
# Canonical timeout environment variables used by full-run flows.

# shellcheck disable=SC2034 # Consumed by scripts that source this file.
ACP_TIMEOUT_ENV_KEYS=(
  ACP_RUNTIME_STEP_TIMEOUT_SEC
  ACP_RUNTIME_HEARTBEAT_SEC
  ACP_PIPELINE_TIMEOUT_SEC
  ACP_PIPELINE_KILL_GRACE_SEC
  ACP_API_READY_TIMEOUT_SEC
  ACP_API_INIT_TIMEOUT_SEC
  ACP_UI_INIT_POLL_TIMEOUT_SEC
  ACP_UI_CANCEL_POLL_TIMEOUT_SEC
)

acp_build_timeout_env_assignments() {
  ACP_TIMEOUT_ENV_ASSIGNMENTS=()
  local key
  local value
  for key in "${ACP_TIMEOUT_ENV_KEYS[@]}"; do
    value="${!key:-}"
    ACP_TIMEOUT_ENV_ASSIGNMENTS+=("$key=$value")
  done
}
