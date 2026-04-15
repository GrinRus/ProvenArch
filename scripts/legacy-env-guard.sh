#!/usr/bin/env bash
# Shared guard for legacy env inputs removed from active contracts.

ACP_LEGACY_ENV_DENYLIST=(
  TARGET_REPO
  TARGET_REPO_GIT_URL
  TARGET_REPO_NAME
  TARGET_REPO_REF
  ACP_FULL_RUN_PIPELINE_TIMEOUT_SEC
  ACP_FULL_RUN_PIPELINE_KILL_GRACE_SEC
  READY_TIMEOUT_SEC
  UI_E2E_INIT_TIMEOUT_SEC
  UI_E2E_CANCEL_TIMEOUT_SEC
)

acp_legacy_env_error_message() {
  local key="$1"
  printf "legacy env '%s' is no longer supported; use canonical TARGET_REPOS_FILE and ACP_RUNTIME_*/ACP_PIPELINE_*/ACP_API_*/ACP_UI_* variables" "$key"
}

acp_ensure_no_legacy_env_set() {
  local fail_fn="${1:-}"
  local key
  local value
  local message
  for key in "${ACP_LEGACY_ENV_DENYLIST[@]}"; do
    value="${!key:-}"
    if [[ -n "$value" ]]; then
      message="$(acp_legacy_env_error_message "$key")"
      if [[ -n "$fail_fn" ]] && declare -F "$fail_fn" >/dev/null 2>&1; then
        "$fail_fn" "$message"
        return 1
      fi
      printf '%s\n' "$message" >&2
      return 1
    fi
  done
}
