#!/usr/bin/env bash

# Canonical reason codes for frontend e2e status artifacts.
ACP_FRONTEND_REASON_OK="ok"
ACP_FRONTEND_REASON_PLAYWRIGHT_FAILED="playwright_failed"
ACP_FRONTEND_REASON_ACTIVE_RUN_TIMEOUT="active_run_timeout"
ACP_FRONTEND_REASON_RUNTIME_RUN_FAILED="runtime_run_failed"
ACP_FRONTEND_REASON_BROWSER_CLOSED="browser_closed"
ACP_FRONTEND_REASON_API_UNREACHABLE="api_unreachable"
ACP_FRONTEND_REASON_SERVER_EXITED="server_exited"
ACP_FRONTEND_REASON_SELECTION_SKIPPED="frontend_selection_skipped"
ACP_FRONTEND_REASON_BACKEND_WORKSPACE_MISSING="backend_workspace_missing"
ACP_FRONTEND_REASON_SNAPSHOT_REPORTS_MISSING="snapshot_reports_missing"
ACP_FRONTEND_REASON_FRONTEND_WORKSPACE_MISSING="frontend_workspace_missing"
ACP_FRONTEND_REASON_FRONTEND_LIVE_E2E_FAILED="frontend_live_e2e_failed"
ACP_FRONTEND_REASON_ARTIFACT_PREVIEW_UNREADABLE="artifact_preview_unreadable"
ACP_FRONTEND_REASON_NAVIGATION_CONFUSING="navigation_confusing"
ACP_FRONTEND_REASON_PUBLISH_DECISION_BLOCKED="publish_decision_blocked"
ACP_FRONTEND_REASON_ASK_FLOW_BLOCKED="ask_flow_blocked"
ACP_FRONTEND_REASON_MOBILE_REVIEW_UNUSABLE="mobile_review_unusable"
ACP_FRONTEND_REASON_PARTIAL_STATE_UNCLEAR="partial_state_unclear"
ACP_FRONTEND_REASON_UNKNOWN="unknown"

acp_frontend_reason_is_allowed() {
  local reason="$1"
  case "$reason" in
    "$ACP_FRONTEND_REASON_OK"|\
      "$ACP_FRONTEND_REASON_PLAYWRIGHT_FAILED"|\
      "$ACP_FRONTEND_REASON_ACTIVE_RUN_TIMEOUT"|\
      "$ACP_FRONTEND_REASON_RUNTIME_RUN_FAILED"|\
      "$ACP_FRONTEND_REASON_BROWSER_CLOSED"|\
      "$ACP_FRONTEND_REASON_API_UNREACHABLE"|\
      "$ACP_FRONTEND_REASON_SERVER_EXITED"|\
      "$ACP_FRONTEND_REASON_SELECTION_SKIPPED"|\
      "$ACP_FRONTEND_REASON_BACKEND_WORKSPACE_MISSING"|\
      "$ACP_FRONTEND_REASON_SNAPSHOT_REPORTS_MISSING"|\
      "$ACP_FRONTEND_REASON_FRONTEND_WORKSPACE_MISSING"|\
      "$ACP_FRONTEND_REASON_FRONTEND_LIVE_E2E_FAILED"|\
      "$ACP_FRONTEND_REASON_ARTIFACT_PREVIEW_UNREADABLE"|\
      "$ACP_FRONTEND_REASON_NAVIGATION_CONFUSING"|\
      "$ACP_FRONTEND_REASON_PUBLISH_DECISION_BLOCKED"|\
      "$ACP_FRONTEND_REASON_ASK_FLOW_BLOCKED"|\
      "$ACP_FRONTEND_REASON_MOBILE_REVIEW_UNUSABLE"|\
      "$ACP_FRONTEND_REASON_PARTIAL_STATE_UNCLEAR"|\
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
