#!/usr/bin/env bash

read_status_field() {
  local status_file="$1"
  local key="$2"
  if [[ ! -f "$status_file" ]]; then
    printf ''
    return 0
  fi
  sed -n "s/^${key}=//p" "$status_file" | tail -n1 | tr -d '\r'
}

read_run_status_seed_field() {
  local key="$1"
  if [[ -z "${RUN_STATUS_FILE:-}" ]]; then
    printf ''
    return 0
  fi
  read_status_field "$RUN_STATUS_FILE" "$key"
}

signal_number() {
  case "$1" in
    HUP) printf '1' ;;
    INT) printf '2' ;;
    PIPE) printf '13' ;;
    TERM) printf '15' ;;
    *) printf '' ;;
  esac
}

signal_status_token() {
  local number
  number="$(signal_number "$1")"
  if [[ -n "$number" ]]; then
    printf 'signal_%s' "$number"
    return 0
  fi
  printf '%s' "$1"
}

signal_exit_code() {
  local number
  number="$(signal_number "$1")"
  if [[ -n "$number" ]]; then
    printf '%s' "$((128 + number))"
    return 0
  fi
  printf '1'
}

write_terminal_run_status() {
  local state="$1"
  local process_exit="$2"
  local termination_signal="$3"
  local failure_reason="$4"
  local summary_written="$5"
  if [[ -z "${RUN_STATUS_FILE:-}" ]]; then
    return 0
  fi
  mkdir -p "$(dirname "$RUN_STATUS_FILE")"
  local provider
  provider="$(read_run_status_seed_field "provider")"
  local run_index
  run_index="$(read_run_status_seed_field "run_index")"
  local now_utc
  now_utc="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
  local last_progress_at
  last_progress_at="${LAST_PROGRESS_AT_UTC:-$now_utc}"
  {
    echo "provider=${provider}"
    echo "run_index=${run_index}"
    echo "state=${state}"
    echo "process_exit=${process_exit}"
    echo "termination_signal=${termination_signal}"
    echo "failure_reason=${failure_reason}"
    echo "summary_written=${summary_written}"
    echo "updated_at=${now_utc}"
    echo "last_pipeline_stage=${LAST_PIPELINE_STAGE:-not_started}"
    echo "last_runtime_provider=${LAST_RUNTIME_PROVIDER:-unset}"
    echo "last_progress_at=${last_progress_at}"
  } >"$RUN_STATUS_FILE"
}

write_running_run_status_heartbeat() {
  if [[ -z "${RUN_STATUS_FILE:-}" ]]; then
    return 0
  fi
  local state
  state="$(read_status_field "$RUN_STATUS_FILE" "state")"
  if [[ -n "$state" && "$state" != "running" ]]; then
    return 0
  fi
  mkdir -p "$(dirname "$RUN_STATUS_FILE")"
  local provider
  provider="$(read_run_status_seed_field "provider")"
  local run_index
  run_index="$(read_run_status_seed_field "run_index")"
  local process_exit
  process_exit="$(read_status_field "$RUN_STATUS_FILE" "process_exit")"
  local termination_signal
  termination_signal="$(read_status_field "$RUN_STATUS_FILE" "termination_signal")"
  if [[ -z "$termination_signal" ]]; then
    termination_signal="none"
  fi
  local failure_reason
  failure_reason="$(read_status_field "$RUN_STATUS_FILE" "failure_reason")"
  local summary_written
  summary_written="$(read_status_field "$RUN_STATUS_FILE" "summary_written")"
  if [[ -z "$summary_written" ]]; then
    summary_written="no"
  fi
  local now_utc
  now_utc="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
  local last_progress_at
  last_progress_at="${LAST_PROGRESS_AT_UTC:-$now_utc}"
  {
    echo "provider=${provider}"
    echo "run_index=${run_index}"
    echo "state=running"
    echo "process_exit=${process_exit}"
    echo "termination_signal=${termination_signal}"
    echo "failure_reason=${failure_reason}"
    echo "summary_written=${summary_written}"
    echo "updated_at=${now_utc}"
    echo "last_pipeline_stage=${LAST_PIPELINE_STAGE:-not_started}"
    echo "last_runtime_provider=${LAST_RUNTIME_PROVIDER:-unset}"
    echo "last_progress_at=${last_progress_at}"
  } >"$RUN_STATUS_FILE"
}
