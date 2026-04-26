#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
# shellcheck source=scripts/legacy-env-guard.sh
source "$PROVENARCH_ROOT/scripts/legacy-env-guard.sh"
# shellcheck source=scripts/repos-meta-fields.sh
source "$PROVENARCH_ROOT/scripts/repos-meta-fields.sh"
# shellcheck source=scripts/frontend-status-reasons.sh
source "$PROVENARCH_ROOT/scripts/frontend-status-reasons.sh"
# shellcheck source=scripts/preflight-log.sh
source "$PROVENARCH_ROOT/scripts/preflight-log.sh"
# shellcheck source=scripts/timeout-env-keys.sh
source "$PROVENARCH_ROOT/scripts/timeout-env-keys.sh"
# shellcheck source=scripts/execution-env-keys.sh
source "$PROVENARCH_ROOT/scripts/execution-env-keys.sh"
TARGET_REPOS_FILE="${TARGET_REPOS_FILE:-}"
PROFILE_ID="${PROFILE_ID:-}"
PROFILE_SOURCE_KIND="${PROFILE_SOURCE_KIND:-}"
EXPECTED_REPO_COUNT="${EXPECTED_REPO_COUNT:-}"
SWEEP_ID="${SWEEP_ID:-baseline}"
BATCH_ID="${BATCH_ID:-batch-$(date -u +'%Y%m%dT%H%M%SZ')}"
RUN_COUNT="${RUN_COUNT:-5}"
ACP_CLAUDE_CMD_BIN="${ACP_CLAUDE_CMD_BIN:-claude}"
ACP_QWEN_CMD_BIN="${ACP_QWEN_CMD_BIN:-qwen}"
ACP_CODEX_CMD_BIN="${ACP_CODEX_CMD_BIN:-codex}"
ACP_APPLY_TIMEOUTS_VIA_API="${ACP_APPLY_TIMEOUTS_VIA_API:-1}"
ACP_EXECUTION_STRATEGY="${ACP_EXECUTION_STRATEGY:-}"
ACP_MAX_PARALLEL_TASKS="${ACP_MAX_PARALLEL_TASKS:-}"
ACP_FAILURE_POLICY="${ACP_FAILURE_POLICY:-}"
ACP_SHARD_DISCOVERY_MODE="${ACP_SHARD_DISCOVERY_MODE:-}"
BATCH_PROVIDER_FILTER="${BATCH_PROVIDER_FILTER:-all}"
BATCH_RUN_SELECTION="${BATCH_RUN_SELECTION:-all}"
BATCH_SKIP_PRECHECK="${BATCH_SKIP_PRECHECK:-0}"
BATCH_FRONTEND_MODE="${BATCH_FRONTEND_MODE:-auto}"
BATCH_FRONTEND_CANCEL_MODE="${BATCH_FRONTEND_CANCEL_MODE:-once_per_provider}"
UI_E2E_HEADED="${UI_E2E_HEADED:-0}"
E2E_TMP_ROOT="${E2E_TMP_ROOT:-/tmp/provenarch-test_arch_project}"
BATCH_ROOT="${BATCH_ROOT:-$E2E_TMP_ROOT/runs/$BATCH_ID}"
REPORTS_ROOT="${REPORTS_ROOT:-$E2E_TMP_ROOT/reports}"
RESOLVED_TARGET_REPOS_FILE=""
DECLARED_REPOS_JSON=""
RUN_CLASSIFICATIONS_TSV=""
BATCH_OWNER_HEARTBEAT_SEC="${BATCH_OWNER_HEARTBEAT_SEC:-10}"
BATCH_OWNER_SENTINEL=""
BATCH_OWNER_HEARTBEAT_PID=""
declare -a STARTED_RUN_DIRS=()
declare -a STARTED_RUN_PROVIDERS=()
declare -a STARTED_RUN_INDEXES=()
# Provider surface is selectable and includes the canonical release triplet
# qwen/claude/codex.
declare -a ALL_PROVIDERS=("qwen-code" "claude-code" "codex-code")
declare -a SELECTED_PROVIDERS=()
declare -a SELECTED_RUN_INDEXES=()
SELECTED_PROVIDERS_CSV=""
SELECTED_RUN_INDEXES_CSV=""
PROFILE_SOURCE_KIND_EFFECTIVE=""
PROFILE_SOURCE_KIND_FOR_FULL_RUN=""
RUNTIME_CONTRACT_FAILURES=0
RUNNER_UNAVAILABLE_FAILURES=0
INFRA_SIGNAL_TERMINATED_FAILURES=0
INFRA_INCOMPLETE_CYCLE_FAILURES=0
RUNTIME_TIMEOUT_FAILURES=0
QUALITY_GATES_FAILED_FAILURES=0
SUMMARY_MISSING_FAILURES=0
PRECHECK_FAILED_FAILURES=0
RUNTIME_FLOW_FAILED_FAILURES=0
CANCELLATION_LIKE_FAILURES=0
OTHER_FAILURES=0
LAST_RUN_FAILURE_CLASS="none"
LAST_RUN_FAILURE_SUBCLASS="none"
LAST_RUN_CANCELLATION_LIKE=0
PRECHECK_FAILURE_RECORDED=0
FRONTEND_CANCEL_FAILURES=0
FRONTEND_CANCEL_SKIPPED=0
FRONTEND_LIVE_RESULT_FILENAME="frontend-e2e-result.json"
FRONTEND_CANCEL_RESULT_FILENAME="frontend-cancel-result.json"
TIMEOUT_PRECHECK_UNSET_KEYS=(
  "${ACP_TIMEOUT_ENV_KEYS[@]}"
  "${ACP_EXECUTION_ENV_KEYS[@]}"
  ACP_APPLY_TIMEOUTS_VIA_API
  ACP_CLAUDE_CMD_BIN
  ACP_QWEN_CMD_BIN
  ACP_CODEX_CMD_BIN
  BATCH_ID
  BATCH_PROVIDER_FILTER
  BATCH_RUN_SELECTION
  BATCH_SCRIPT
  BATCH_ROOT
  BATCH_SKIP_PRECHECK
  BATCH_FRONTEND_MODE
  BATCH_FRONTEND_CANCEL_MODE
  E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES
  E2E_MATRIX_FILE
  E2E_MATRIX_RELEASE_MODE
  E2E_TMP_ROOT
  EXPECTED_REPO_COUNT
  MATRIX_DRIVER_LOG
  MATRIX_ID
  MATRIX_ROOT
  MATRIX_TEST_SENTINEL
  MATRIX_TEST_TIMEOUT_SENTINEL
  MATRIX_TIMEOUT_PROFILE_FILE
  PROFILE_ID
  PROFILE_SOURCE_KIND
  REPORTS_ROOT
  RUN_COUNT
  SWEEP_ID
  TARGET_REPOS_FILE
  UI_E2E_HEADED
)

log() {
  printf '[batch] %s\n' "$*" >&2
}

die() {
  echo "[batch][error] $*" >&2
  exit 1
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    die "required command is unavailable: $cmd"
  fi
}

run_status_file() {
  local run_dir="$1"
  printf '%s' "$run_dir/run-status.env"
}

batch_owner_status_file() {
  printf '%s' "${BATCH_ROOT}/batch-owner.env"
}

write_batch_owner_status() {
  local state="$1"
  local process_exit="${2:-}"
  local termination_signal="${3:-none}"
  local failure_reason="${4:-none}"
  local status_file
  status_file="${BATCH_OWNER_SENTINEL:-$(batch_owner_status_file)}"
  mkdir -p "$(dirname "$status_file")"
  cat >"$status_file" <<EOF
batch_id=$BATCH_ID
profile_id=$PROFILE_ID
sweep_id=$SWEEP_ID
pid=$$
parent_pid=${PPID:-}
state=$state
process_exit=$process_exit
termination_signal=$termination_signal
failure_reason=$failure_reason
updated_at=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
EOF
}

stop_batch_owner_heartbeat() {
  if [[ -z "${BATCH_OWNER_HEARTBEAT_PID:-}" ]]; then
    return 0
  fi
  kill "${BATCH_OWNER_HEARTBEAT_PID}" >/dev/null 2>&1 || true
  wait "${BATCH_OWNER_HEARTBEAT_PID}" >/dev/null 2>&1 || true
  BATCH_OWNER_HEARTBEAT_PID=""
}

start_batch_owner_heartbeat() {
  stop_batch_owner_heartbeat
  local interval="${BATCH_OWNER_HEARTBEAT_SEC:-10}"
  if [[ ! "$interval" =~ ^[0-9]+$ ]] || [[ "$interval" -le 0 ]]; then
    write_batch_owner_status "running" "" "none" "none"
    return 0
  fi
  write_batch_owner_status "running" "" "none" "none"
  (
    while true; do
      sleep "$interval"
      write_batch_owner_status "running" "" "none" "none"
    done
  ) &
  BATCH_OWNER_HEARTBEAT_PID="$!"
}

finalize_batch_owner_status() {
  local exit_code="$1"
  stop_batch_owner_heartbeat
  if [[ ! "$exit_code" =~ ^[0-9]+$ ]]; then
    exit_code=1
  fi
  if [[ "$exit_code" -eq 0 ]]; then
    write_batch_owner_status "completed" "0" "none" "none"
    return 0
  fi
  if [[ "$exit_code" -ge 128 ]]; then
    write_batch_owner_status "signal_terminated" "$exit_code" "signal_$((exit_code - 128))" "infra_signal_terminated"
    return 0
  fi
  write_batch_owner_status "process_failed" "$exit_code" "none" "batch_exit_nonzero"
}

write_run_status() {
  local run_dir="$1"
  local provider="$2"
  local run_index="$3"
  local state="$4"
  local process_exit="${5:-}"
  local termination_signal="${6:-none}"
  local failure_reason="${7:-}"
  local summary_written="${8:-no}"
  local status_file
  status_file="$(run_status_file "$run_dir")"
  cat >"$status_file" <<EOF
provider=$provider
run_index=$run_index
state=$state
process_exit=$process_exit
termination_signal=$termination_signal
failure_reason=$failure_reason
summary_written=$summary_written
updated_at=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
EOF
}

read_status_field() {
  local status_file="$1"
  local key="$2"
  if [[ ! -f "$status_file" ]]; then
    printf ''
    return 0
  fi
  sed -n "s/^${key}=//p" "$status_file" | tail -n1 | tr -d '\r'
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

run_classification_exists() {
  local provider="$1"
  local run_index="$2"
  if [[ ! -f "$RUN_CLASSIFICATIONS_TSV" ]]; then
    return 1
  fi
  grep -q "^${provider}"$'\t'"${run_index}"$'\t' "$RUN_CLASSIFICATIONS_TSV"
}

ensure_terminal_run_status() {
  local run_dir="$1"
  local provider="$2"
  local run_index="$3"
  local process_exit="$4"
  local run_status_path
  run_status_path="$(run_status_file "$run_dir")"
  local state
  state="$(read_status_field "$run_status_path" "state")"
  if [[ "$state" != "running" && -n "$state" ]]; then
    return 0
  fi

  local effective_process_exit="$process_exit"
  if [[ ! "$effective_process_exit" =~ ^[0-9]+$ || "$effective_process_exit" -eq 0 ]]; then
    effective_process_exit=1
  fi

  local run_status_state="process_failed"
  local run_status_signal="none"
  local failure_reason="infra_incomplete_cycle"
  if [[ "$effective_process_exit" -ge 128 ]]; then
    run_status_state="signal_terminated"
    run_status_signal="signal_$((effective_process_exit - 128))"
    failure_reason="infra_signal_terminated"
  fi
  write_run_status "$run_dir" "$provider" "$run_index" "$run_status_state" "$effective_process_exit" "$run_status_signal" "$failure_reason" "no"
}

classify_started_runs_on_signal() {
  local signal_name="$1"
  local idx
  for idx in "${!STARTED_RUN_DIRS[@]}"; do
    local run_dir="${STARTED_RUN_DIRS[$idx]}"
    local provider="${STARTED_RUN_PROVIDERS[$idx]}"
    local run_index="${STARTED_RUN_INDEXES[$idx]}"
    local run_status_path=""
    local run_status_state=""
    local process_exit=""
    [[ -z "$run_dir" || -z "$provider" || -z "$run_index" ]] && continue
    if run_classification_exists "$provider" "$run_index"; then
      continue
    fi
    run_status_path="$(run_status_file "$run_dir")"
    run_status_state="$(read_status_field "$run_status_path" "state")"
    process_exit="$(read_status_field "$run_status_path" "process_exit")"
    if [[ -z "$run_status_state" || "$run_status_state" == "running" ]]; then
      process_exit="$(signal_exit_code "$signal_name")"
      write_run_status "$run_dir" "$provider" "$run_index" "signal_terminated" "$process_exit" "$(signal_status_token "$signal_name")" "infra_signal_terminated" "no"
    elif [[ ! "$process_exit" =~ ^[0-9]+$ ]]; then
      process_exit="$(signal_exit_code "$signal_name")"
    fi
    classify_run_failure "$provider" "$run_index" "$run_dir" "$process_exit"
  done
}

on_batch_signal() {
  local signal_name="$1"
  log "received termination signal: $signal_name"
  stop_batch_owner_heartbeat
  write_batch_owner_status "signal_terminated" "$(signal_exit_code "$signal_name")" "$(signal_status_token "$signal_name")" "infra_signal_terminated"
  classify_started_runs_on_signal "$signal_name"
  exit "$(signal_exit_code "$signal_name")"
}

classify_unfinished_started_runs_on_exit() {
  local exit_code="$1"
  local idx
  for idx in "${!STARTED_RUN_DIRS[@]}"; do
    local run_dir="${STARTED_RUN_DIRS[$idx]}"
    local provider="${STARTED_RUN_PROVIDERS[$idx]}"
    local run_index="${STARTED_RUN_INDEXES[$idx]}"
    local run_status_path=""
    local process_exit=""
    [[ -z "$run_dir" || -z "$provider" || -z "$run_index" ]] && continue
    if run_classification_exists "$provider" "$run_index"; then
      continue
    fi
    run_status_path="$(run_status_file "$run_dir")"
    process_exit="$(read_status_field "$run_status_path" "process_exit")"
    if [[ ! "$process_exit" =~ ^[0-9]+$ ]]; then
      process_exit="$exit_code"
    fi
    if [[ ! "$process_exit" =~ ^[0-9]+$ || "$process_exit" -eq 0 ]]; then
      process_exit=1
    fi
    ensure_terminal_run_status "$run_dir" "$provider" "$run_index" "$process_exit"
    classify_run_failure "$provider" "$run_index" "$run_dir" "$process_exit"
  done
}

on_batch_exit() {
  local exit_code="$1"
  finalize_batch_owner_status "$exit_code"
  if [[ -z "${RUN_CLASSIFICATIONS_TSV:-}" ]]; then
    return 0
  fi
  classify_unfinished_started_runs_on_exit "$exit_code"
}

array_contains() {
  local needle="$1"
  shift
  local item
  for item in "$@"; do
    if [[ "$item" == "$needle" ]]; then
      return 0
    fi
  done
  return 1
}

run_index_selected() {
  local run_index="$1"
  array_contains "$run_index" "${SELECTED_RUN_INDEXES[@]-}"
}

first_selected_run_index() {
  if [[ "${#SELECTED_RUN_INDEXES[@]}" -eq 0 ]]; then
    printf ''
    return 0
  fi
  printf '%s' "${SELECTED_RUN_INDEXES[0]}"
}

provider_selected() {
  local provider="$1"
  array_contains "$provider" "${SELECTED_PROVIDERS[@]-}"
}

resolve_selected_providers() {
  local filter_raw="${BATCH_PROVIDER_FILTER:-all}"
  local filter
  filter="$(echo "$filter_raw" | tr -d '[:space:]')"
  if [[ -z "$filter" || "$filter" == "all" ]]; then
    SELECTED_PROVIDERS=("${ALL_PROVIDERS[@]}")
  else
    SELECTED_PROVIDERS=()
    local token
    IFS=',' read -r -a tokens <<<"$filter"
    for token in "${tokens[@]}"; do
      [[ -z "$token" ]] && continue
      case "$token" in
        qwen-code|claude-code|codex-code)
          if ! array_contains "$token" "${SELECTED_PROVIDERS[@]-}"; then
            SELECTED_PROVIDERS+=("$token")
          fi
          ;;
        *)
          die "BATCH_PROVIDER_FILTER contains unsupported provider '$token' (allowed: qwen-code, claude-code, codex-code, all)"
          ;;
      esac
    done
  fi
  if [[ "${#SELECTED_PROVIDERS[@]}" -eq 0 ]]; then
    die "BATCH_PROVIDER_FILTER resolved to an empty provider set"
  fi
  SELECTED_PROVIDERS_CSV="$(IFS=,; echo "${SELECTED_PROVIDERS[*]}")"
}

resolve_selected_run_indexes() {
  local selection_raw="${BATCH_RUN_SELECTION:-all}"
  local resolved_indexes
  resolved_indexes="$(python3 - "$RUN_COUNT" "$selection_raw" <<'PY'
import sys

try:
    run_count = int((sys.argv[1] or "").strip())
except Exception:
    raise SystemExit("RUN_COUNT must be a positive integer")

selection = (sys.argv[2] or "").strip().lower()
if selection in {"", "all"}:
    print("\n".join(str(i) for i in range(1, run_count + 1)))
    raise SystemExit(0)

values = set()
for token in [part.strip() for part in selection.split(",") if part.strip()]:
    if "-" in token:
        left, right = token.split("-", 1)
        try:
            start = int(left)
            end = int(right)
        except Exception:
            raise SystemExit(f"invalid run range token: {token}")
        if start > end:
            raise SystemExit(f"invalid descending run range token: {token}")
        for value in range(start, end + 1):
            values.add(value)
    else:
        try:
            values.add(int(token))
        except Exception:
            raise SystemExit(f"invalid run token: {token}")

if not values:
    raise SystemExit("BATCH_RUN_SELECTION resolved to an empty run set")

for value in sorted(values):
    if value < 1 or value > run_count:
        raise SystemExit(f"run index out of bounds: {value} (RUN_COUNT={run_count})")

print("\n".join(str(value) for value in sorted(values)))
PY
)"
  SELECTED_RUN_INDEXES=()
  local run_index
  while IFS= read -r run_index; do
    [[ -z "$run_index" ]] && continue
    SELECTED_RUN_INDEXES+=("$run_index")
  done <<<"$resolved_indexes"
  if [[ "${#SELECTED_RUN_INDEXES[@]}" -eq 0 ]]; then
    die "BATCH_RUN_SELECTION resolved to an empty run set"
  fi
  SELECTED_RUN_INDEXES_CSV="$(IFS=,; echo "${SELECTED_RUN_INDEXES[*]}")"
}

should_run_frontend_once() {
  case "$BATCH_FRONTEND_MODE" in
    never)
      return 1
      ;;
    always)
      return 0
      ;;
    auto)
      run_index_selected "1"
      return $?
      ;;
    per_run)
      return 1
      ;;
    *)
      die "BATCH_FRONTEND_MODE must be auto|always|never|per_run (got '$BATCH_FRONTEND_MODE')"
      ;;
  esac
}

should_run_frontend_for_run() {
  local run_index="$1"
  if [[ "$BATCH_FRONTEND_MODE" != "per_run" ]]; then
    return 1
  fi
  run_index_selected "$run_index"
}

should_write_frontend_skip_result() {
  case "$BATCH_FRONTEND_MODE" in
    never)
      return 0
      ;;
    auto)
      if should_run_frontend_once; then
        return 1
      fi
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

should_run_frontend_cancel_once() {
  case "$BATCH_FRONTEND_CANCEL_MODE" in
    never)
      return 1
      ;;
    once_per_provider)
      return 0
      ;;
    per_run)
      return 1
      ;;
    *)
      die "BATCH_FRONTEND_CANCEL_MODE must be once_per_provider|per_run|never (got '$BATCH_FRONTEND_CANCEL_MODE')"
      ;;
  esac
}

should_run_frontend_cancel_for_run() {
  local run_index="$1"
  if [[ "$BATCH_FRONTEND_CANCEL_MODE" != "per_run" ]]; then
    return 1
  fi
  run_index_selected "$run_index"
}

frontend_live_output_dir() {
  local provider="$1"
  local run_index="${2:-}"
  if [[ -n "$run_index" ]]; then
    printf '%s' "$BATCH_ROOT/frontend/$provider/run${run_index}"
    return 0
  fi
  printf '%s' "$BATCH_ROOT/frontend/$provider"
}

frontend_cancel_output_dir() {
  local provider="$1"
  local run_index="${2:-}"
  if [[ -n "$run_index" ]]; then
    printf '%s' "$BATCH_ROOT/frontend-cancel/$provider/run${run_index}"
    return 0
  fi
  printf '%s' "$BATCH_ROOT/frontend-cancel/$provider"
}

resolve_frontend_live_backend_run() {
  local provider="$1"
  local run_index
  run_index="$(first_selected_run_index)"
  if [[ -z "$run_index" ]]; then
    printf '\t\n'
    return 0
  fi
  printf '%s\t%s\n' "$BATCH_ROOT/$provider/run${run_index}" "$run_index"
}

backend_workspace_candidates() {
  local run_dir="$1"
  printf '%s\n' \
    "$run_dir/headless/arch-workspace" \
    "$run_dir/arch-workspace" \
    "$run_dir/workspace"
}

first_backend_workspace_candidate() {
  local run_dir="$1"
  backend_workspace_candidates "$run_dir" | head -n1
}

resolve_backend_workspace_dir() {
  local run_dir="$1"
  local candidate
  while IFS= read -r candidate; do
    [[ -z "$candidate" ]] && continue
    if [[ -d "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done < <(backend_workspace_candidates "$run_dir")
  printf ''
}

summary_scalar() {
  local summary_path="$1"
  local key="$2"
  if [[ ! -f "$summary_path" ]]; then
    printf ''
    return 0
  fi
  sed -n "s/^- ${key}: //p" "$summary_path" | tail -n1 | tr -d '\r'
}

contains_in_files() {
  local needle="$1"
  shift
  local path
  for path in "$@"; do
    if [[ -f "$path" ]] && grep -q "$needle" "$path"; then
      return 0
    fi
  done
  return 1
}

contains_regex_in_files() {
  local pattern="$1"
  shift
  local path
  for path in "$@"; do
    if [[ -f "$path" ]] && grep -E -q "$pattern" "$path"; then
      return 0
    fi
  done
  return 1
}

contains_runner_unavailable_signature() {
  local -a paths=("$@")
  local path
  local line
  local generic_pattern="([Mm]odel( is)? at capacity|[Ss]tatus[[:space:]]*[:=][[:space:]]*429|(^|[^0-9])429([^0-9]|$)|[Rr]ate[ -]?limit(ed)?|[Tt]oo many requests)"
  local noise_pattern="(chatgpt\\.com/backend-api/plugins/featured|[Cc]loudflare|failed to renew cache ttl|state db|Operation not permitted)"
  for path in "${paths[@]}"; do
    [[ ! -f "$path" ]] && continue
    if grep -q "runner_unavailable" "$path"; then
      return 0
    fi
    while IFS= read -r line || [[ -n "$line" ]]; do
      if printf '%s\n' "$line" | grep -E -q "$generic_pattern"; then
        if printf '%s\n' "$line" | grep -E -q "$noise_pattern"; then
          continue
        fi
        return 0
      fi
    done < "$path"
  done
  return 1
}

contains_runtime_contract_parse_signature() {
  local -a paths=("$@")
  local path
  for path in "${paths[@]}"; do
    if [[ -f "$path" ]] \
      && grep -E -q "parse runtime draft manifest" "$path" \
      && grep -E -q "unknown field" "$path"; then
      return 0
    fi
  done
  return 1
}

is_explicit_failure_class() {
  local value="$1"
  case "$value" in
    runtime_timeout|runner_unavailable|runtime_contract_failed|infra_signal_terminated|quality_gates_failed|runtime_flow_failed)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

write_frontend_status_json() {
  local path="$1"
  local provider="$2"
  local scenario="$3"
  local status="$4"
  local reason="$5"
  local workspace="$6"
  local output_dir="$7"
  local runtime_command="$8"
  local server_log="${9:-}"
  local playwright_log="${10:-}"
  local run_index="${11:-}"
  acp_frontend_reason_validate "$reason" die
  python3 - "$path" "$provider" "$scenario" "$status" "$reason" "$workspace" "$output_dir" "$runtime_command" "$server_log" "$playwright_log" "$run_index" <<'PY'
import json
import sys
from datetime import datetime, timezone

path, provider, scenario, status, reason, workspace, output_dir, runtime_command, server_log, playwright_log, run_index = sys.argv[1:]
payload = {
    "started_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "finished_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "status": status,
    "reason": reason,
    "runtime_provider": provider,
    "scenario": scenario,
    "workspace": workspace,
    "output_dir": output_dir,
    "runtime_command": runtime_command,
    "server_log": server_log or "-",
    "playwright_log": playwright_log or "-",
}
if run_index.strip():
    payload["run_index"] = int(run_index)
with open(path, "w", encoding="utf-8") as f:
    json.dump(payload, f, ensure_ascii=True, indent=2)
    f.write("\n")
PY
}

runtime_cmd_for_provider() {
  local provider="$1"
  case "$provider" in
    qwen-code)
      printf '%s' "$ACP_QWEN_CMD_BIN"
      ;;
    claude-code)
      printf '%s' "$ACP_CLAUDE_CMD_BIN"
      ;;
    codex-code)
      printf '%s' "$ACP_CODEX_CMD_BIN"
      ;;
    *)
      die "unsupported provider '$provider' (allowed: qwen-code, claude-code, codex-code)"
      ;;
  esac
}

write_frontend_cancel_failed_result_json() {
  local provider="$1"
  local frontend_workspace="$2"
  local cancel_output_dir="$3"
  local runtime_cmd="$4"
  local run_index="${5:-}"
  write_frontend_status_json \
    "$cancel_output_dir/$FRONTEND_CANCEL_RESULT_FILENAME" \
    "$provider" \
    "cancel-refresh" \
    "failed" \
    "$ACP_FRONTEND_REASON_FRONTEND_LIVE_E2E_FAILED" \
    "$frontend_workspace" \
    "$cancel_output_dir" \
    "$runtime_cmd" \
    "$cancel_output_dir/server.log" \
    "$cancel_output_dir/playwright.log" \
    "$run_index"
}

run_frontend_live_e2e() {
  local provider="$1"
  local backend_run_dir="$2"
  local output_dir="$3"
  local run_index="${4:-}"
  local workspace
  workspace="$(resolve_backend_workspace_dir "$backend_run_dir")"
  local workspace_fallback
  workspace_fallback="$(first_backend_workspace_candidate "$backend_run_dir")"
  if [[ -z "$workspace" ]]; then
    workspace="$workspace_fallback"
  fi
  local frontend_workspace="$output_dir/frontend-workspace"
  local run_results_path="$backend_run_dir/run-results.tsv"
  local refresh_run_id=""
  local snapshot_reports=""
  local runtime_cmd
  runtime_cmd="$(runtime_cmd_for_provider "$provider")"

  mkdir -p "$output_dir"
  if [[ -f "$run_results_path" ]]; then
    refresh_run_id="$(awk -F'\t' '$2=="headless" && $4=="refresh" {print $5}' "$run_results_path" | tail -n1)"
  fi
  if [[ -n "$refresh_run_id" ]]; then
    snapshot_reports="$backend_run_dir/snapshots/$refresh_run_id/reports"
  fi

  if [[ ! -d "$workspace" ]]; then
    write_frontend_status_json \
      "$output_dir/$FRONTEND_LIVE_RESULT_FILENAME" \
      "$provider" \
      "init-inspect" \
      "failed" \
      "$ACP_FRONTEND_REASON_BACKEND_WORKSPACE_MISSING" \
      "$workspace" \
      "$output_dir" \
      "$runtime_cmd" \
      "$output_dir/server.log" \
      "$output_dir/playwright.log" \
      "$run_index"
    log "frontend e2e failed provider=$provider run=${run_index:-summary} reason=$ACP_FRONTEND_REASON_BACKEND_WORKSPACE_MISSING (workspace=$workspace)"
    return 1
  fi

  if [[ -z "$refresh_run_id" || ! -d "$snapshot_reports" ]]; then
    write_frontend_status_json \
      "$output_dir/$FRONTEND_LIVE_RESULT_FILENAME" \
      "$provider" \
      "init-inspect" \
      "failed" \
      "$ACP_FRONTEND_REASON_SNAPSHOT_REPORTS_MISSING" \
      "$snapshot_reports" \
      "$output_dir" \
      "$runtime_cmd" \
      "$output_dir/server.log" \
      "$output_dir/playwright.log" \
      "$run_index"
    log "frontend e2e failed provider=$provider run=${run_index:-summary} reason=$ACP_FRONTEND_REASON_SNAPSHOT_REPORTS_MISSING refresh_run_id=${refresh_run_id:-unknown} snapshot_reports=${snapshot_reports:-unset}"
    return 1
  fi

  rm -rf "$frontend_workspace"
  cp -a "$workspace" "$frontend_workspace"
  rm -rf "$frontend_workspace/reports"
  cp -a "$snapshot_reports" "$frontend_workspace/reports"

  log "frontend live e2e provider=$provider run=${run_index:-summary} workspace=$frontend_workspace artifact_source=snapshot refresh_run_id=${refresh_run_id:-unknown}"
  if ! (
    cd "$PROVENARCH_ROOT"
    acp_build_timeout_env_assignments
    acp_build_execution_env_assignments
    env \
      "${ACP_TIMEOUT_ENV_ASSIGNMENTS[@]}" \
      "${ACP_EXECUTION_ENV_ASSIGNMENTS[@]}" \
      "WORKSPACE=$frontend_workspace" \
      "RUNTIME_PROVIDER=$provider" \
      "OUTPUT_DIR=$output_dir" \
      "UI_E2E_EXPECTED_REPO_COUNT=$EXPECTED_REPO_COUNT_RESOLVED" \
      "ACP_CLAUDE_CMD=$ACP_CLAUDE_CMD_BIN" \
      "ACP_QWEN_CMD=$ACP_QWEN_CMD_BIN" \
      "ACP_CODEX_CMD=$ACP_CODEX_CMD_BIN" \
      "UI_E2E_HEADED=$UI_E2E_HEADED" \
      "FRONTEND_RESULT_FILENAME=$FRONTEND_LIVE_RESULT_FILENAME" \
      ./scripts/frontend-live-e2e.sh
  ) >"$output_dir/driver.log" 2>&1; then
    if [[ ! -f "$output_dir/$FRONTEND_LIVE_RESULT_FILENAME" ]]; then
      write_frontend_status_json \
        "$output_dir/$FRONTEND_LIVE_RESULT_FILENAME" \
        "$provider" \
        "init-inspect" \
        "failed" \
        "$ACP_FRONTEND_REASON_FRONTEND_LIVE_E2E_FAILED" \
        "$frontend_workspace" \
        "$output_dir" \
        "$runtime_cmd" \
        "$output_dir/server.log" \
        "$output_dir/playwright.log" \
        "$run_index"
    fi
    log "frontend e2e failed provider=$provider run=${run_index:-summary} (see $output_dir/driver.log)"
    return 1
  fi

  if [[ ! -f "$output_dir/$FRONTEND_LIVE_RESULT_FILENAME" ]]; then
    write_frontend_status_json \
      "$output_dir/$FRONTEND_LIVE_RESULT_FILENAME" \
      "$provider" \
      "init-inspect" \
      "failed" \
      "$ACP_FRONTEND_REASON_FRONTEND_LIVE_E2E_FAILED" \
      "$frontend_workspace" \
      "$output_dir" \
      "$runtime_cmd" \
      "$output_dir/server.log" \
      "$output_dir/playwright.log" \
      "$run_index"
    log "frontend e2e failed provider=$provider run=${run_index:-summary} reason=missing_result_json (see $output_dir/driver.log)"
    return 1
  fi

  return 0
}

prepare_frontend_cancel_workspace() {
  local backend_run_dir="$1"
  local output_dir="$2"
  if [[ -z "$backend_run_dir" ]]; then
    printf ''
    return 0
  fi
  local workspace
  workspace="$(resolve_backend_workspace_dir "$backend_run_dir")"
  local frontend_workspace="$output_dir/frontend-workspace"

  if [[ -z "$workspace" || ! -d "$workspace" ]]; then
    printf ''
    return 0
  fi

  mkdir -p "$output_dir"
  rm -rf "$frontend_workspace"
  cp -a "$workspace" "$frontend_workspace"
  printf '%s\n' "$frontend_workspace"
}

run_frontend_cancel_e2e() {
  local provider="$1"
  local frontend_workspace="$2"
  local output_dir="$3"
  local run_index="${4:-}"
  local runtime_cmd
  runtime_cmd="$(runtime_cmd_for_provider "$provider")"

  mkdir -p "$output_dir"
  if [[ ! -d "$frontend_workspace" ]]; then
    write_frontend_status_json \
      "$output_dir/$FRONTEND_CANCEL_RESULT_FILENAME" \
      "$provider" \
      "cancel-refresh" \
      "skipped" \
      "$ACP_FRONTEND_REASON_FRONTEND_WORKSPACE_MISSING" \
      "$frontend_workspace" \
      "$output_dir" \
      "$runtime_cmd" \
      "$output_dir/server.log" \
      "$output_dir/playwright.log" \
      "$run_index"
    log "frontend cancel smoke skipped provider=$provider run=${run_index:-summary} reason=$ACP_FRONTEND_REASON_FRONTEND_WORKSPACE_MISSING"
    return 2
  fi

  log "frontend cancel smoke provider=$provider run=${run_index:-summary} workspace=$frontend_workspace"
  if ! (
    cd "$PROVENARCH_ROOT"
    acp_build_timeout_env_assignments
    acp_build_execution_env_assignments
    env \
      "${ACP_TIMEOUT_ENV_ASSIGNMENTS[@]}" \
      "${ACP_EXECUTION_ENV_ASSIGNMENTS[@]}" \
      "WORKSPACE=$frontend_workspace" \
      "RUNTIME_PROVIDER=$provider" \
      "OUTPUT_DIR=$output_dir" \
      "UI_E2E_SCENARIO=cancel-refresh" \
      "UI_E2E_EXPECTED_REPO_COUNT=$EXPECTED_REPO_COUNT_RESOLVED" \
      "ACP_CLAUDE_CMD=$ACP_CLAUDE_CMD_BIN" \
      "ACP_QWEN_CMD=$ACP_QWEN_CMD_BIN" \
      "UI_E2E_HEADED=$UI_E2E_HEADED" \
      "FRONTEND_RESULT_FILENAME=$FRONTEND_CANCEL_RESULT_FILENAME" \
      ./scripts/frontend-live-e2e.sh
  ) >"$output_dir/driver.log" 2>&1; then
    if [[ ! -f "$output_dir/$FRONTEND_CANCEL_RESULT_FILENAME" ]]; then
      write_frontend_cancel_failed_result_json "$provider" "$frontend_workspace" "$output_dir" "$runtime_cmd" "$run_index"
    fi
    log "frontend cancel smoke failed provider=$provider run=${run_index:-summary} (see $output_dir/driver.log)"
    return 1
  fi

  if [[ ! -f "$output_dir/$FRONTEND_CANCEL_RESULT_FILENAME" ]]; then
    write_frontend_cancel_failed_result_json "$provider" "$frontend_workspace" "$output_dir" "$runtime_cmd" "$run_index"
    log "frontend cancel smoke failed provider=$provider run=${run_index:-summary} reason=missing_result_json (see $output_dir/driver.log)"
    return 1
  fi

  return 0
}

run_dod_precheck_make() {
  local -a env_cmd=("env")
  local key
  for key in "${TIMEOUT_PRECHECK_UNSET_KEYS[@]}"; do
    env_cmd+=("-u" "$key")
  done
  "${env_cmd[@]}" make contracts test lint build
}

classify_run_failure() {
  local provider="$1"
  local run_index="$2"
  local run_dir="$3"
  local process_exit="$4"
  local summary_path="$run_dir/session-summary.md"
  local run_results_path="$run_dir/run-results.tsv"
  local full_log_path="$run_dir/full-run.log"
  local batch_driver_log="$run_dir/batch-driver.log"

  local summary_result=""
  local failure_reason=""
  local expected_runs=""
  local completed_runs=""
  local expected_headless_runs=""
  local completed_headless_runs=""
  local running_runs_detected=""
  local termination_signal=""
  local run_count=0
  local run_class="none"
  local run_subclass="none"
  local cancellation_like=0
  local quality_gates_status=""
  local run_status_path="$run_dir/run-status.env"
  local run_status_state=""
  local run_status_signal=""
  local run_status_process_exit=""
  local run_status_failure_reason=""
  local run_status_summary_written=""
  local terminal_pipeline_failure=0
  local validator_verdict_failed=0
  local -a classify_log_paths=("$summary_path" "$full_log_path" "$batch_driver_log")
  local workspace=""
  local -a workspace_candidates=()
  local workspace_candidate
  while IFS= read -r workspace_candidate; do
    [[ -z "$workspace_candidate" ]] && continue
    workspace_candidates+=("$workspace_candidate")
    if [[ -z "$workspace" && -d "$workspace_candidate" ]]; then
      workspace="$workspace_candidate"
    fi
  done < <(backend_workspace_candidates "$run_dir")
  if [[ -z "$workspace" && "${#workspace_candidates[@]}" -gt 0 ]]; then
    workspace="${workspace_candidates[0]}"
  fi
  local iter_log
  if [[ -d "$run_dir/logs" ]]; then
    while IFS= read -r iter_log; do
      classify_log_paths+=("$iter_log")
    done < <(find "$run_dir/logs" -maxdepth 1 -type f -name 'run-iter*-*.log' | LC_ALL=C sort)
  fi
  for workspace_candidate in "${workspace_candidates[@]}"; do
    if [[ -d "$workspace_candidate/reports/taskruns/logs" ]]; then
      while IFS= read -r iter_log; do
        classify_log_paths+=("$iter_log")
      done < <(find "$workspace_candidate/reports/taskruns/logs" -maxdepth 1 -type f -name '*.ndjson' | LC_ALL=C sort)
    fi
    if [[ -d "$workspace_candidate/reports/taskruns/raw" ]]; then
      while IFS= read -r iter_log; do
        classify_log_paths+=("$iter_log")
      done < <(find "$workspace_candidate/reports/taskruns/raw" -type f | LC_ALL=C sort)
    fi
  done

  run_status_state="$(read_status_field "$run_status_path" "state")"
  run_status_signal="$(read_status_field "$run_status_path" "termination_signal")"
  run_status_process_exit="$(read_status_field "$run_status_path" "process_exit")"
  run_status_failure_reason="$(read_status_field "$run_status_path" "failure_reason")"
  run_status_summary_written="$(read_status_field "$run_status_path" "summary_written")"
  if [[ "$run_status_process_exit" =~ ^[0-9]+$ ]]; then
    process_exit="$run_status_process_exit"
  fi

  if [[ ! -f "$summary_path" ]]; then
    summary_result="missing"
    if [[ -n "$run_status_failure_reason" && "$run_status_failure_reason" != "none" ]]; then
      failure_reason="$run_status_failure_reason"
    fi
    if [[ "$run_status_state" == "signal_terminated" || "$run_status_signal" != "" && "$run_status_signal" != "none" || "$failure_reason" == "infra_signal_terminated" || "$process_exit" -ge 128 ]]; then
      run_class="infra_signal_terminated"
      failure_reason="infra_signal_terminated"
      if [[ -z "$termination_signal" || "$termination_signal" == "none" ]]; then
        if [[ -n "$run_status_signal" && "$run_status_signal" != "none" ]]; then
          termination_signal="$run_status_signal"
        elif [[ "$process_exit" -ge 128 ]]; then
          termination_signal="signal_$((process_exit - 128))"
        fi
      fi
    else
      run_class="infra_incomplete_cycle"
      if [[ -z "$failure_reason" || "$failure_reason" == "none" ]]; then
        failure_reason="infra_incomplete_cycle"
      fi
    fi
    if [[ "$run_status_summary_written" == "yes" ]]; then
      summary_result="expected_but_missing"
    fi
  else
    summary_result="$(summary_scalar "$summary_path" "result" | awk '{print $1}')"
    quality_gates_status="$(summary_scalar "$summary_path" "quality_gates" | awk '{print $1}')"
    failure_reason="$(summary_scalar "$summary_path" "failure_reason" | awk '{print $1}')"
    expected_runs="$(summary_scalar "$summary_path" "expected_runs" | awk '{print $1}')"
    completed_runs="$(summary_scalar "$summary_path" "completed_runs" | awk '{print $1}')"
    expected_headless_runs="$(summary_scalar "$summary_path" "expected_headless_runs" | awk '{print $1}')"
    completed_headless_runs="$(summary_scalar "$summary_path" "completed_headless_runs" | awk '{print $1}')"
    running_runs_detected="$(summary_scalar "$summary_path" "running_runs_detected" | awk '{print $1}')"
    termination_signal="$(summary_scalar "$summary_path" "termination_signal" | awk '{print $1}')"
  fi

  if [[ -f "$summary_path" && "$run_status_state" == "process_failed" && "$run_status_summary_written" == "yes" ]]; then
    terminal_pipeline_failure=1
  fi
  if contains_in_files "validator verdict is FAIL" "${classify_log_paths[@]}"; then
    validator_verdict_failed=1
  fi

  if [[ -f "$run_results_path" ]]; then
    run_count="$(awk 'NF { count++ } END { print count+0 }' "$run_results_path")"
  fi

  if [[ "$run_class" == "none" ]]; then
    if is_explicit_failure_class "$failure_reason"; then
      run_class="$failure_reason"
    fi
  fi

  if [[ "$run_class" == "none" ]]; then
    if [[ "$failure_reason" == "runtime_timeout" || "$termination_signal" == "timeout" ]]; then
      run_class="runtime_timeout"
    fi
  fi

  if [[ "$run_class" == "none" ]] && contains_runtime_contract_parse_signature "${classify_log_paths[@]}"; then
    run_class="runtime_contract_failed"
  fi
  if [[ "$run_class" == "none" && "$validator_verdict_failed" == "1" ]]; then
    run_class="runtime_flow_failed"
  fi
  if [[ "$run_class" == "none" ]] && contains_in_files "runtime_contract_failed" "${classify_log_paths[@]}"; then
    run_class="runtime_contract_failed"
  fi
  if [[ "$run_class" == "none" ]] && contains_runner_unavailable_signature "${classify_log_paths[@]}"; then
    run_class="runner_unavailable"
  fi

  if [[ "$run_class" == "none" ]]; then
    if [[ "$failure_reason" == "infra_signal_terminated" ]]; then
      run_class="infra_signal_terminated"
    elif [[ "$termination_signal" != "" && "$termination_signal" != "none" ]]; then
      run_class="infra_signal_terminated"
    fi
  fi

  if [[ "$run_class" == "none" ]]; then
    if [[ "$failure_reason" == "quality" || "$quality_gates_status" == "failed" ]]; then
      run_class="quality_gates_failed"
    fi
  fi

  if [[ "$run_class" == "none" ]]; then
    if [[ "$failure_reason" == "infra_incomplete_cycle" ]]; then
      run_class="infra_incomplete_cycle"
    fi
    if [[ "$terminal_pipeline_failure" != "1" ]]; then
      if [[ "$expected_runs" =~ ^[0-9]+$ && "$completed_runs" =~ ^[0-9]+$ ]]; then
        if (( completed_runs != expected_runs )); then
          run_class="infra_incomplete_cycle"
        fi
      fi
      if [[ "$expected_headless_runs" =~ ^[0-9]+$ && "$completed_headless_runs" =~ ^[0-9]+$ ]]; then
        if (( completed_headless_runs != expected_headless_runs )); then
          run_class="infra_incomplete_cycle"
        fi
      fi
      if [[ "$running_runs_detected" =~ ^[0-9]+$ ]] && (( running_runs_detected > 0 )); then
        run_class="infra_incomplete_cycle"
      fi
      if [[ "$summary_result" != "passed" && "$run_class" == "none" ]]; then
        run_class="infra_incomplete_cycle"
      fi
      if [[ ! -f "$run_results_path" ]]; then
        run_class="infra_incomplete_cycle"
      fi
    fi
  fi

  if [[ "$run_class" == "none" && "$terminal_pipeline_failure" == "1" ]]; then
    run_class="runtime_flow_failed"
  fi

  if [[ "$process_exit" -ne 0 && "$run_class" == "none" ]]; then
    if [[ "$terminal_pipeline_failure" == "1" ]]; then
      run_class="runtime_flow_failed"
    else
      run_class="infra_incomplete_cycle"
    fi
  fi

  if contains_regex_in_files "FatalCancellationError|code[=: ]130" "${classify_log_paths[@]}"; then
    cancellation_like=1
    run_subclass="cancellation_like"
  fi
  if [[ -z "$summary_result" ]]; then
    summary_result="missing"
  fi
  if [[ -z "$failure_reason" ]]; then
    failure_reason="-"
  fi
  if [[ -z "$termination_signal" ]]; then
    termination_signal="none"
  fi
  if [[ -z "$expected_runs" ]]; then
    expected_runs="-"
  fi
  if [[ -z "$completed_runs" ]]; then
    completed_runs="-"
  fi
  if [[ -z "$expected_headless_runs" ]]; then
    expected_headless_runs="-"
  fi
  if [[ -z "$completed_headless_runs" ]]; then
    completed_headless_runs="-"
  fi
  if [[ -z "$running_runs_detected" ]]; then
    running_runs_detected="-"
  fi

  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$provider" \
    "$run_index" \
    "$run_class" \
    "$process_exit" \
    "$summary_result" \
    "$failure_reason" \
    "$termination_signal" \
    "$expected_runs" \
    "$completed_runs" \
    "$expected_headless_runs" \
    "$completed_headless_runs" \
    "$running_runs_detected" \
    "$run_count" \
    "$run_subclass" \
    "$cancellation_like" >>"$RUN_CLASSIFICATIONS_TSV"

  LAST_RUN_FAILURE_CLASS="$run_class"
  LAST_RUN_FAILURE_SUBCLASS="$run_subclass"
  LAST_RUN_CANCELLATION_LIKE="$cancellation_like"
}

increment_failure_class_counter() {
  local run_class="$1"
  local run_subclass="${2:-none}"
  local cancellation_like="${3:-0}"
  case "$run_class" in
    runtime_contract_failed)
      RUNTIME_CONTRACT_FAILURES=$((RUNTIME_CONTRACT_FAILURES + 1))
      ;;
    runner_unavailable)
      RUNNER_UNAVAILABLE_FAILURES=$((RUNNER_UNAVAILABLE_FAILURES + 1))
      ;;
    infra_signal_terminated)
      INFRA_SIGNAL_TERMINATED_FAILURES=$((INFRA_SIGNAL_TERMINATED_FAILURES + 1))
      ;;
    infra_incomplete_cycle)
      INFRA_INCOMPLETE_CYCLE_FAILURES=$((INFRA_INCOMPLETE_CYCLE_FAILURES + 1))
      ;;
    runtime_timeout)
      RUNTIME_TIMEOUT_FAILURES=$((RUNTIME_TIMEOUT_FAILURES + 1))
      ;;
    quality_gates_failed)
      QUALITY_GATES_FAILED_FAILURES=$((QUALITY_GATES_FAILED_FAILURES + 1))
      ;;
    summary_missing)
      SUMMARY_MISSING_FAILURES=$((SUMMARY_MISSING_FAILURES + 1))
      ;;
    precheck_failed)
      PRECHECK_FAILED_FAILURES=$((PRECHECK_FAILED_FAILURES + 1))
      ;;
    runtime_flow_failed)
      RUNTIME_FLOW_FAILED_FAILURES=$((RUNTIME_FLOW_FAILED_FAILURES + 1))
      ;;
    none)
      ;;
    *)
      OTHER_FAILURES=$((OTHER_FAILURES + 1))
      ;;
  esac
  if [[ "$run_subclass" == "cancellation_like" || "$cancellation_like" == "1" ]]; then
    CANCELLATION_LIKE_FAILURES=$((CANCELLATION_LIKE_FAILURES + 1))
  fi
}

record_precheck_failed_classifications() {
  if [[ "$PRECHECK_FAILURE_RECORDED" == "1" ]]; then
    return 0
  fi
  local provider
  local run_index
  for provider in "${SELECTED_PROVIDERS[@]}"; do
    for run_index in "${SELECTED_RUN_INDEXES[@]}"; do
      printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
        "$provider" \
        "$run_index" \
        "precheck_failed" \
        "1" \
        "precheck_failed" \
        "precheck_failed" \
        "none" \
        "-" \
        "-" \
        "-" \
        "-" \
        "-" \
        "0" \
        "none" \
        "0" >>"$RUN_CLASSIFICATIONS_TSV"
      increment_failure_class_counter "precheck_failed" "none" "0"
    done
  done
  PRECHECK_FAILURE_RECORDED=1
}

finalize_precheck_failure() {
  local reason="$1"
  record_precheck_failed_classifications
  log "precheck failed: $reason"
  log "generating quality reports for batch=$BATCH_ID (precheck_failed)"
  if (
    cd "$PROVENARCH_ROOT"
    python3 scripts/e2e_batch_report.py \
      --batch-id "$BATCH_ID" \
      --batch-root "$BATCH_ROOT" \
      --reports-root "$REPORTS_ROOT" >"$BATCH_ROOT/report-paths.txt"
  ); then
    log "report paths:"
    cat "$BATCH_ROOT/report-paths.txt"
  else
    log "report generation failed after precheck failure (see $BATCH_ROOT/report-paths.txt if present)"
  fi
  log "backend failure classes: precheck_failed=$PRECHECK_FAILED_FAILURES runtime_contract_failed=$RUNTIME_CONTRACT_FAILURES runner_unavailable=$RUNNER_UNAVAILABLE_FAILURES runtime_timeout=$RUNTIME_TIMEOUT_FAILURES infra_signal_terminated=$INFRA_SIGNAL_TERMINATED_FAILURES infra_incomplete_cycle=$INFRA_INCOMPLETE_CYCLE_FAILURES quality_gates_failed=$QUALITY_GATES_FAILED_FAILURES summary_missing=$SUMMARY_MISSING_FAILURES runtime_flow_failed=$RUNTIME_FLOW_FAILED_FAILURES cancellation_like=$CANCELLATION_LIKE_FAILURES other=$OTHER_FAILURES"
  die "batch precheck failed: reason=$reason precheck_failed=$PRECHECK_FAILED_FAILURES runtime_contract_failed=$RUNTIME_CONTRACT_FAILURES runner_unavailable=$RUNNER_UNAVAILABLE_FAILURES runtime_timeout=$RUNTIME_TIMEOUT_FAILURES infra_signal_terminated=$INFRA_SIGNAL_TERMINATED_FAILURES infra_incomplete_cycle=$INFRA_INCOMPLETE_CYCLE_FAILURES quality_gates_failed=$QUALITY_GATES_FAILED_FAILURES summary_missing=$SUMMARY_MISSING_FAILURES runtime_flow_failed=$RUNTIME_FLOW_FAILED_FAILURES cancellation_like=$CANCELLATION_LIKE_FAILURES other=$OTHER_FAILURES"
}

prepare_target_repos_file() {
  if [[ -n "$TARGET_REPOS_FILE" ]]; then
    if [[ ! -f "$TARGET_REPOS_FILE" ]]; then
      die "TARGET_REPOS_FILE does not exist: $TARGET_REPOS_FILE"
    fi
    RESOLVED_TARGET_REPOS_FILE="$(cd "$(dirname "$TARGET_REPOS_FILE")" && pwd)/$(basename "$TARGET_REPOS_FILE")"
    return 0
  fi

  die "missing target input: set TARGET_REPOS_FILE=/abs/path/to/repos.yaml"
}

collect_declared_repos() {
  DECLARED_REPOS_JSON="$BATCH_ROOT/declared-repos.json"
  python3 "$PROVENARCH_ROOT/scripts/resolve-repos-meta.py" \
    --repos-file "$RESOLVED_TARGET_REPOS_FILE" \
    --expected-repo-count "$EXPECTED_REPO_COUNT" \
    --source-kind "$PROFILE_SOURCE_KIND" \
    --profile-id "$PROFILE_ID" \
    --out "$DECLARED_REPOS_JSON"
}

read_declared_repos_meta() {
  local key
  local value
  local resolved_expected_count="0"
  local resolved_source_kind="mixed"
  while IFS='=' read -r key value; do
    case "$key" in
      expected_repo_count) resolved_expected_count="$value" ;;
      profile_source_kind) resolved_source_kind="$value" ;;
    esac
  done < <(acp_read_repos_meta_fields "$DECLARED_REPOS_JSON")

  EXPECTED_REPO_COUNT_RESOLVED="${resolved_expected_count:-0}"
  PROFILE_SOURCE_KIND_EFFECTIVE="${resolved_source_kind:-mixed}"
}

if [[ ! "$RUN_COUNT" =~ ^[1-9][0-9]*$ ]]; then
  die "RUN_COUNT must be a positive integer, got '$RUN_COUNT'"
fi
if [[ "$ACP_APPLY_TIMEOUTS_VIA_API" != "0" && "$ACP_APPLY_TIMEOUTS_VIA_API" != "1" ]]; then
  die "ACP_APPLY_TIMEOUTS_VIA_API must be 0 or 1, got '$ACP_APPLY_TIMEOUTS_VIA_API'"
fi
if ! acp_validate_execution_env_overrides die; then
  exit 1
fi
if [[ "$BATCH_SKIP_PRECHECK" != "0" && "$BATCH_SKIP_PRECHECK" != "1" ]]; then
  die "BATCH_SKIP_PRECHECK must be 0 or 1, got '$BATCH_SKIP_PRECHECK'"
fi
case "$BATCH_FRONTEND_MODE" in
  auto|always|never|per_run)
    ;;
  *)
    die "BATCH_FRONTEND_MODE must be auto|always|never|per_run (got '$BATCH_FRONTEND_MODE')"
    ;;
esac
case "$BATCH_FRONTEND_CANCEL_MODE" in
  once_per_provider|per_run|never)
    ;;
  *)
    die "BATCH_FRONTEND_CANCEL_MODE must be once_per_provider|per_run|never (got '$BATCH_FRONTEND_CANCEL_MODE')"
    ;;
esac
case "$UI_E2E_HEADED" in
  0|1)
    ;;
  *)
    die "UI_E2E_HEADED must be 0 or 1, got '$UI_E2E_HEADED'"
    ;;
esac

require_cmd git
require_cmd go
require_cmd npm
require_cmd make
require_cmd python3
require_cmd curl
resolve_selected_providers
resolve_selected_run_indexes
if provider_selected "claude-code"; then
  require_cmd "$ACP_CLAUDE_CMD_BIN"
fi
if provider_selected "qwen-code"; then
  require_cmd "$ACP_QWEN_CMD_BIN"
fi
if provider_selected "codex-code"; then
  require_cmd "$ACP_CODEX_CMD_BIN"
fi

mkdir -p "$BATCH_ROOT" "$REPORTS_ROOT"
BATCH_OWNER_SENTINEL="$(batch_owner_status_file)"
start_batch_owner_heartbeat
acp_ensure_no_legacy_env_set die
prepare_target_repos_file
collect_declared_repos
RUN_CLASSIFICATIONS_TSV="$BATCH_ROOT/backend-run-classifications.tsv"
echo -e "provider\trun_index\tfailure_class\tprocess_exit\tsummary_result\tfailure_reason\ttermination_signal\texpected_runs\tcompleted_runs\texpected_headless_runs\tcompleted_headless_runs\trunning_runs_detected\trun_results_rows\tfailure_subclass\tcancellation_like" >"$RUN_CLASSIFICATIONS_TSV"
trap 'on_batch_signal TERM' TERM
trap 'on_batch_signal INT' INT
trap 'on_batch_signal HUP' HUP
trap 'on_batch_exit $?' EXIT

PROVENARCH_SHA="$(git -C "$PROVENARCH_ROOT" rev-parse HEAD)"
PROVENARCH_BRANCH="$(git -C "$PROVENARCH_ROOT" rev-parse --abbrev-ref HEAD)"
CLAUDE_PATH="not-selected"
QWEN_PATH="not-selected"
CODEX_PATH="not-selected"
CLAUDE_VERSION="not-selected"
QWEN_VERSION="not-selected"
CODEX_VERSION="not-selected"
if provider_selected "claude-code"; then
  CLAUDE_PATH="$(command -v "$ACP_CLAUDE_CMD_BIN")"
  CLAUDE_VERSION="$("$ACP_CLAUDE_CMD_BIN" --version | head -n1 | tr -d '\r')"
fi
if provider_selected "qwen-code"; then
  QWEN_PATH="$(command -v "$ACP_QWEN_CMD_BIN")"
  QWEN_VERSION="$("$ACP_QWEN_CMD_BIN" --version | head -n1 | tr -d '\r')"
fi
if provider_selected "codex-code"; then
  CODEX_PATH="$(command -v "$ACP_CODEX_CMD_BIN")"
  CODEX_VERSION="$("$ACP_CODEX_CMD_BIN" --version | head -n1 | tr -d '\r')"
fi
GENERATED_AT_UTC="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
preflight_meta_lines="$(python3 "$PROVENARCH_ROOT/scripts/write-batch-preflight.py" \
  --out "$BATCH_ROOT/preflight.json" \
  --generated-at-utc "$GENERATED_AT_UTC" \
  --provenarch-root "$PROVENARCH_ROOT" \
  --provenarch-sha "$PROVENARCH_SHA" \
  --provenarch-branch "$PROVENARCH_BRANCH" \
  --target-repos-file "$RESOLVED_TARGET_REPOS_FILE" \
  --declared-repos-meta-file "$DECLARED_REPOS_JSON" \
  --apply-timeouts-via-api "$ACP_APPLY_TIMEOUTS_VIA_API" \
  --sweep-id "$SWEEP_ID" \
  --selected-providers "$SELECTED_PROVIDERS_CSV" \
  --selected-run-indexes "$SELECTED_RUN_INDEXES_CSV" \
  --claude-path "$CLAUDE_PATH" \
  --claude-version-line "$CLAUDE_VERSION" \
  --qwen-path "$QWEN_PATH" \
  --qwen-version-line "$QWEN_VERSION" \
  --codex-path "$CODEX_PATH" \
  --codex-version-line "$CODEX_VERSION")"
TIMEOUT_PROFILE_LINE=""
EXECUTION_PROFILE_LINE=""
PROVIDER_READINESS_STATUS=""
PROVIDER_READINESS_REASON=""
while IFS='=' read -r key value; do
  case "$key" in
    timeout_profile_line) TIMEOUT_PROFILE_LINE="$value" ;;
    execution_profile_line) EXECUTION_PROFILE_LINE="$value" ;;
    provider_readiness_status) PROVIDER_READINESS_STATUS="$value" ;;
    provider_readiness_reason) PROVIDER_READINESS_REASON="$value" ;;
  esac
done <<<"$preflight_meta_lines"
if [[ -z "$TIMEOUT_PROFILE_LINE" || -z "$EXECUTION_PROFILE_LINE" ]]; then
  die "preflight helper did not return timeout/execution profile lines"
fi
if [[ "$PROVIDER_READINESS_STATUS" == "unavailable" ]]; then
  die "operational_host_preflight_failed: selected provider readiness failed: ${PROVIDER_READINESS_REASON:-unknown provider readiness failure}"
fi

read_declared_repos_meta
case "$PROFILE_SOURCE_KIND_EFFECTIVE" in
  path|git_url|mixed)
    ;;
  *)
    die "declared repos metadata has invalid profile_source_kind '$PROFILE_SOURCE_KIND_EFFECTIVE' (expected path|git_url|mixed)"
    ;;
esac
PROFILE_SOURCE_KIND_FOR_FULL_RUN="$PROFILE_SOURCE_KIND_EFFECTIVE"
if [[ "$PROFILE_SOURCE_KIND_FOR_FULL_RUN" == "mixed" ]]; then
  PROFILE_SOURCE_KIND_FOR_FULL_RUN=""
fi

log "target repos input: file=$RESOLVED_TARGET_REPOS_FILE profile_id=${PROFILE_ID:-adhoc} source_kind=$PROFILE_SOURCE_KIND_EFFECTIVE expected_repo_count=$EXPECTED_REPO_COUNT_RESOLVED"
log "preflight versions: claude='$CLAUDE_VERSION' qwen='$QWEN_VERSION' codex='$CODEX_VERSION'"
acp_log_preflight_timeout log "$ACP_APPLY_TIMEOUTS_VIA_API" "$TIMEOUT_PROFILE_LINE"
acp_log_preflight_execution log "${SWEEP_ID:-baseline}" "$EXECUTION_PROFILE_LINE"
log "batch shard selection: providers=$SELECTED_PROVIDERS_CSV runs=$SELECTED_RUN_INDEXES_CSV skip_precheck=$BATCH_SKIP_PRECHECK frontend_mode=$BATCH_FRONTEND_MODE frontend_cancel_mode=$BATCH_FRONTEND_CANCEL_MODE headed=$UI_E2E_HEADED"
if [[ "$BATCH_SKIP_PRECHECK" == "1" ]]; then
  log "skipping DoD/UI precheck (BATCH_SKIP_PRECHECK=1)"
else
  log "running DoD precheck: make contracts test lint build"
  if ! (
    cd "$PROVENARCH_ROOT"
    run_dod_precheck_make >"$BATCH_ROOT/precheck-make.log" 2>&1
  ); then
    finalize_precheck_failure "make contracts test lint build failed (see $BATCH_ROOT/precheck-make.log)"
  fi

  log "installing UI dependencies and Playwright browser"
  if ! (
    cd "$PROVENARCH_ROOT"
    npm ci --prefix ui >"$BATCH_ROOT/precheck-ui-npm.log" 2>&1
    npm exec --prefix ui playwright install chromium >"$BATCH_ROOT/precheck-playwright.log" 2>&1
  ); then
    finalize_precheck_failure "UI precheck failed (see $BATCH_ROOT/precheck-ui-npm.log and $BATCH_ROOT/precheck-playwright.log)"
  fi
fi

failed_runs=0
for provider in "${SELECTED_PROVIDERS[@]}"; do
  for i in "${SELECTED_RUN_INDEXES[@]}"; do
    run_dir="$BATCH_ROOT/$provider/run${i}"
    mkdir -p "$run_dir"
    write_run_status "$run_dir" "$provider" "$i" "running" "" "none" "" "no"
    STARTED_RUN_DIRS+=("$run_dir")
    STARTED_RUN_PROVIDERS+=("$provider")
    STARTED_RUN_INDEXES+=("$i")
    log "full-run provider=$provider run=$i tmp_root=$run_dir"
    process_exit=0
    (
      cd "$PROVENARCH_ROOT"
      acp_build_timeout_env_assignments
      acp_build_execution_env_assignments
      env \
        "${ACP_TIMEOUT_ENV_ASSIGNMENTS[@]}" \
        "${ACP_EXECUTION_ENV_ASSIGNMENTS[@]}" \
        "TARGET_REPOS_FILE=$RESOLVED_TARGET_REPOS_FILE" \
        "TMP_ROOT=$run_dir" \
        "RUN_STATUS_FILE=$(run_status_file "$run_dir")" \
        "KEEP_TMP=1" \
        "ITERATIONS=1" \
        "RUN_QUALITY_GATES=1" \
        "PROFILE_ID=${PROFILE_ID:-adhoc}" \
        "PROFILE_SOURCE_KIND=$PROFILE_SOURCE_KIND_FOR_FULL_RUN" \
        "EXPECTED_REPO_COUNT=$EXPECTED_REPO_COUNT_RESOLVED" \
        "ACP_RUNTIME_PROVIDER=$provider" \
        "ACP_CLAUDE_CMD=$ACP_CLAUDE_CMD_BIN" \
        "ACP_QWEN_CMD=$ACP_QWEN_CMD_BIN" \
        "ACP_CODEX_CMD=$ACP_CODEX_CMD_BIN" \
        "ACP_APPLY_TIMEOUTS_VIA_API=$ACP_APPLY_TIMEOUTS_VIA_API" \
        "SWEEP_ID=${SWEEP_ID:-baseline}" \
        ./scripts/full-run-ai-advent.sh
    ) >"$run_dir/batch-driver.log" 2>&1 || process_exit=$?

    ensure_terminal_run_status "$run_dir" "$provider" "$i" "$process_exit"
    classify_run_failure "$provider" "$i" "$run_dir" "$process_exit"
    if [[ "$LAST_RUN_FAILURE_CLASS" != "none" ]]; then
      failed_runs=$((failed_runs + 1))
      increment_failure_class_counter "$LAST_RUN_FAILURE_CLASS" "$LAST_RUN_FAILURE_SUBCLASS" "$LAST_RUN_CANCELLATION_LIKE"
      log "run failed provider=$provider run=$i class=$LAST_RUN_FAILURE_CLASS subclass=$LAST_RUN_FAILURE_SUBCLASS cancellation_like=$LAST_RUN_CANCELLATION_LIKE (see $run_dir/batch-driver.log)"
    fi
  done
done

frontend_failures=0
for provider in "${SELECTED_PROVIDERS[@]}"; do
  if should_run_frontend_once; then
    resolved_frontend_run="$(resolve_frontend_live_backend_run "$provider")"
    backend_run_dir="${resolved_frontend_run%%$'\t'*}"
    frontend_run_index="${resolved_frontend_run#*$'\t'}"
    output_dir="$(frontend_live_output_dir "$provider")"
    if ! run_frontend_live_e2e "$provider" "$backend_run_dir" "$output_dir" "$frontend_run_index"; then
      frontend_failures=$((frontend_failures + 1))
    fi
    continue
  fi

  if [[ "$BATCH_FRONTEND_MODE" == "per_run" ]]; then
    for i in "${SELECTED_RUN_INDEXES[@]}"; do
      if ! should_run_frontend_for_run "$i"; then
        continue
      fi
      output_dir="$(frontend_live_output_dir "$provider" "$i")"
      if ! run_frontend_live_e2e "$provider" "$BATCH_ROOT/$provider/run${i}" "$output_dir" "$i"; then
        frontend_failures=$((frontend_failures + 1))
      fi
    done
    continue
  fi

  if should_write_frontend_skip_result; then
    output_dir="$(frontend_live_output_dir "$provider")"
    mkdir -p "$output_dir"
    runtime_cmd="$(runtime_cmd_for_provider "$provider")"
    write_frontend_status_json \
      "$output_dir/$FRONTEND_LIVE_RESULT_FILENAME" \
      "$provider" \
      "init-inspect" \
      "skipped" \
      "$ACP_FRONTEND_REASON_SELECTION_SKIPPED" \
      "$output_dir/frontend-workspace" \
      "$output_dir" \
      "$runtime_cmd"
    log "frontend live e2e skipped provider=$provider mode=$BATCH_FRONTEND_MODE runs=$SELECTED_RUN_INDEXES_CSV"
    continue
  fi
done

for provider in "${SELECTED_PROVIDERS[@]}"; do
  runtime_cmd="$(runtime_cmd_for_provider "$provider")"
  if should_run_frontend_cancel_once; then
    cancel_output_dir="$(frontend_cancel_output_dir "$provider")"
    resolved_frontend_run="$(resolve_frontend_live_backend_run "$provider")"
    backend_run_dir="${resolved_frontend_run%%$'\t'*}"
    cancel_run_index="${resolved_frontend_run#*$'\t'}"
    frontend_workspace="$(prepare_frontend_cancel_workspace "$backend_run_dir" "$cancel_output_dir")"
    cancel_result=0
    run_frontend_cancel_e2e "$provider" "$frontend_workspace" "$cancel_output_dir" "$cancel_run_index" || cancel_result=$?
    if [[ "$cancel_result" == "1" ]]; then
      FRONTEND_CANCEL_FAILURES=$((FRONTEND_CANCEL_FAILURES + 1))
    elif [[ "$cancel_result" == "2" ]]; then
      FRONTEND_CANCEL_SKIPPED=$((FRONTEND_CANCEL_SKIPPED + 1))
    fi
    continue
  fi

  if [[ "$BATCH_FRONTEND_CANCEL_MODE" == "per_run" ]]; then
    for i in "${SELECTED_RUN_INDEXES[@]}"; do
      if ! should_run_frontend_cancel_for_run "$i"; then
        continue
      fi
      cancel_output_dir="$(frontend_cancel_output_dir "$provider" "$i")"
      frontend_workspace="$(prepare_frontend_cancel_workspace "$BATCH_ROOT/$provider/run${i}" "$cancel_output_dir")"
      cancel_result=0
      run_frontend_cancel_e2e "$provider" "$frontend_workspace" "$cancel_output_dir" "$i" || cancel_result=$?
      if [[ "$cancel_result" == "1" ]]; then
        FRONTEND_CANCEL_FAILURES=$((FRONTEND_CANCEL_FAILURES + 1))
      elif [[ "$cancel_result" == "2" ]]; then
        FRONTEND_CANCEL_SKIPPED=$((FRONTEND_CANCEL_SKIPPED + 1))
      fi
    done
    continue
  fi

  if [[ "$BATCH_FRONTEND_CANCEL_MODE" == "never" ]]; then
    cancel_output_dir="$(frontend_cancel_output_dir "$provider")"
    mkdir -p "$cancel_output_dir"
    FRONTEND_CANCEL_SKIPPED=$((FRONTEND_CANCEL_SKIPPED + 1))
    write_frontend_status_json \
      "$cancel_output_dir/$FRONTEND_CANCEL_RESULT_FILENAME" \
      "$provider" \
      "cancel-refresh" \
      "skipped" \
      "$ACP_FRONTEND_REASON_SELECTION_SKIPPED" \
      "$BATCH_ROOT/frontend/$provider/frontend-workspace" \
      "$cancel_output_dir" \
      "$runtime_cmd"
    log "frontend cancel smoke skipped provider=$provider mode=$BATCH_FRONTEND_CANCEL_MODE runs=$SELECTED_RUN_INDEXES_CSV"
  fi
done

log "generating quality reports for batch=$BATCH_ID"
(
  cd "$PROVENARCH_ROOT"
  python3 scripts/e2e_batch_report.py \
    --batch-id "$BATCH_ID" \
    --batch-root "$BATCH_ROOT" \
    --reports-root "$REPORTS_ROOT" >"$BATCH_ROOT/report-paths.txt"
)

log "report paths:"
cat "$BATCH_ROOT/report-paths.txt"

log "backend failure classes: precheck_failed=$PRECHECK_FAILED_FAILURES runtime_contract_failed=$RUNTIME_CONTRACT_FAILURES runner_unavailable=$RUNNER_UNAVAILABLE_FAILURES runtime_timeout=$RUNTIME_TIMEOUT_FAILURES infra_signal_terminated=$INFRA_SIGNAL_TERMINATED_FAILURES infra_incomplete_cycle=$INFRA_INCOMPLETE_CYCLE_FAILURES quality_gates_failed=$QUALITY_GATES_FAILED_FAILURES summary_missing=$SUMMARY_MISSING_FAILURES runtime_flow_failed=$RUNTIME_FLOW_FAILED_FAILURES cancellation_like=$CANCELLATION_LIKE_FAILURES other=$OTHER_FAILURES"

if [[ "$failed_runs" -ne 0 || "$frontend_failures" -ne 0 || "$FRONTEND_CANCEL_FAILURES" -ne 0 ]]; then
  die "batch completed with failures: full_run_failed=$failed_runs frontend_failed=$frontend_failures frontend_cancel_failed=$FRONTEND_CANCEL_FAILURES frontend_cancel_skipped=$FRONTEND_CANCEL_SKIPPED precheck_failed=$PRECHECK_FAILED_FAILURES runtime_contract_failed=$RUNTIME_CONTRACT_FAILURES runner_unavailable=$RUNNER_UNAVAILABLE_FAILURES runtime_timeout=$RUNTIME_TIMEOUT_FAILURES infra_signal_terminated=$INFRA_SIGNAL_TERMINATED_FAILURES infra_incomplete_cycle=$INFRA_INCOMPLETE_CYCLE_FAILURES quality_gates_failed=$QUALITY_GATES_FAILED_FAILURES summary_missing=$SUMMARY_MISSING_FAILURES runtime_flow_failed=$RUNTIME_FLOW_FAILED_FAILURES cancellation_like=$CANCELLATION_LIKE_FAILURES other=$OTHER_FAILURES"
fi

log "batch completed successfully"
exit 0
