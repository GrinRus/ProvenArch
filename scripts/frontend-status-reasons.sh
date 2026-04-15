#!/usr/bin/env bash

# Canonical reason codes for frontend e2e status artifacts.
ACP_FRONTEND_REASON_OK="ok"
ACP_FRONTEND_REASON_PLAYWRIGHT_FAILED="playwright_failed"
ACP_FRONTEND_REASON_SELECTION_SKIPPED="frontend_selection_skipped"
ACP_FRONTEND_REASON_BACKEND_WORKSPACE_MISSING="backend_workspace_missing"
ACP_FRONTEND_REASON_SNAPSHOT_REPORTS_MISSING="snapshot_reports_missing"
ACP_FRONTEND_REASON_FRONTEND_WORKSPACE_MISSING="frontend_workspace_missing"
ACP_FRONTEND_REASON_FRONTEND_LIVE_E2E_FAILED="frontend_live_e2e_failed"
ACP_FRONTEND_REASON_UNKNOWN="unknown"

acp_frontend_reason_is_allowed() {
  local reason="$1"
  case "$reason" in
    "$ACP_FRONTEND_REASON_OK"|\
      "$ACP_FRONTEND_REASON_PLAYWRIGHT_FAILED"|\
      "$ACP_FRONTEND_REASON_SELECTION_SKIPPED"|\
      "$ACP_FRONTEND_REASON_BACKEND_WORKSPACE_MISSING"|\
      "$ACP_FRONTEND_REASON_SNAPSHOT_REPORTS_MISSING"|\
      "$ACP_FRONTEND_REASON_FRONTEND_WORKSPACE_MISSING"|\
      "$ACP_FRONTEND_REASON_FRONTEND_LIVE_E2E_FAILED"|\
      "$ACP_FRONTEND_REASON_UNKNOWN")
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

acp_frontend_reason_validate() {
  local reason="$1"
  local on_error="${2:-echo}"
  if acp_frontend_reason_is_allowed "$reason"; then
    return 0
  fi
  local message="unsupported frontend status reason '$reason'"
  if declare -F "$on_error" >/dev/null 2>&1; then
    "$on_error" "$message"
  else
    echo "$message" >&2
  fi
  return 1
}
