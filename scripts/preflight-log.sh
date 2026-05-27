#!/usr/bin/env bash
# Shared preflight/release log helpers for batch scripts.

acp_log_preflight_timeout() {
  local logger_fn="$1"
  local apply_timeouts_via_api="$2"
  local timeout_profile_line="$3"
  "$logger_fn" "preflight timeout profile: apply_via_api=$apply_timeouts_via_api $timeout_profile_line"
}

acp_log_preflight_execution() {
  local logger_fn="$1"
  local sweep_id="$2"
  local execution_profile_line="$3"
  local sweep_effective="${sweep_id:-baseline}"
  "$logger_fn" "preflight execution profile: sweep_id=$sweep_effective $execution_profile_line"
}

acp_log_release_guard() {
  local logger_fn="$1"
  local mode="$2"
  local matrix_id="$3"
  local allow_diagnostic_timeout_overrides="$4"
  local overrides_detected="$5"
  "$logger_fn" \
    "release guard: mode=$mode matrix_id=$matrix_id allow_diagnostic_timeout_overrides=$allow_diagnostic_timeout_overrides overrides_detected=$overrides_detected"
}

acp_log_diagnostic_timeout_overrides() {
  local logger_fn="$1"
  local overrides_rendered="$2"
  if [[ -n "$overrides_rendered" ]]; then
    "$logger_fn" "diagnostic timeout overrides: $overrides_rendered"
  fi
}

acp_release_guard_blocked_message() {
  printf '%s' \
    "release guard blocked diagnostic timeout overrides; clear env and rerun diagnostics outside release mode"
}
