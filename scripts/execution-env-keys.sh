#!/usr/bin/env bash
# Canonical execution environment variables used by batch/matrix flows.

# shellcheck disable=SC2034 # Consumed by scripts that source this file.
ACP_EXECUTION_ENV_KEYS=(
  ACP_EXECUTION_STRATEGY
  ACP_MAX_PARALLEL_TASKS
  ACP_FAILURE_POLICY
  ACP_SHARD_DISCOVERY_MODE
  ACP_REPO_SELECTION
)

acp_validate_execution_env_overrides() {
  local fail_fn="${1:-}"
  local message=""

  if [[ -n "${ACP_EXECUTION_STRATEGY:-}" && "${ACP_EXECUTION_STRATEGY}" != "sequential" && "${ACP_EXECUTION_STRATEGY}" != "parallel" ]]; then
    message="ACP_EXECUTION_STRATEGY must be sequential|parallel (or empty), got '${ACP_EXECUTION_STRATEGY}'"
  elif [[ -n "${ACP_MAX_PARALLEL_TASKS:-}" && ! "${ACP_MAX_PARALLEL_TASKS}" =~ ^[1-9][0-9]*$ ]]; then
    message="ACP_MAX_PARALLEL_TASKS must be positive integer (or empty), got '${ACP_MAX_PARALLEL_TASKS}'"
  elif [[ -n "${ACP_FAILURE_POLICY:-}" && "${ACP_FAILURE_POLICY}" != "fail_fast" && "${ACP_FAILURE_POLICY}" != "best_effort" ]]; then
    message="ACP_FAILURE_POLICY must be fail_fast|best_effort (or empty), got '${ACP_FAILURE_POLICY}'"
  elif [[ -n "${ACP_SHARD_DISCOVERY_MODE:-}" && "${ACP_SHARD_DISCOVERY_MODE}" != "heuristics" && "${ACP_SHARD_DISCOVERY_MODE}" != "semantic" ]]; then
    message="ACP_SHARD_DISCOVERY_MODE must be heuristics|semantic (or empty), got '${ACP_SHARD_DISCOVERY_MODE}'"
  elif [[ -n "${ACP_REPO_SELECTION:-}" && "${ACP_REPO_SELECTION}" != "all" && "${ACP_REPO_SELECTION}" != "backend_only" ]]; then
    message="ACP_REPO_SELECTION must be all|backend_only (or empty), got '${ACP_REPO_SELECTION}'"
  fi

  if [[ -z "$message" ]]; then
    return 0
  fi

  if [[ -n "$fail_fn" ]] && declare -F "$fail_fn" >/dev/null 2>&1; then
    "$fail_fn" "$message"
  else
    printf '%s\n' "$message" >&2
  fi
  return 1
}

acp_build_execution_env_assignments() {
  ACP_EXECUTION_ENV_ASSIGNMENTS=()
  local key
  local value
  for key in "${ACP_EXECUTION_ENV_KEYS[@]}"; do
    value="${!key:-}"
    ACP_EXECUTION_ENV_ASSIGNMENTS+=("$key=$value")
  done
}
