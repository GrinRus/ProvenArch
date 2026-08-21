#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
# Internal backend-cycle helper for scripts/full-run-batch.sh.
# Public live E2E entrypoints are scripts/full-run-batch-matrix.sh and scripts/full-run-batch.sh.
# shellcheck source=scripts/legacy-env-guard.sh
source "$PROVENARCH_ROOT/scripts/legacy-env-guard.sh"
# shellcheck source=scripts/repos-meta-fields.sh
source "$PROVENARCH_ROOT/scripts/repos-meta-fields.sh"
# shellcheck source=scripts/timeout-env-keys.sh
source "$PROVENARCH_ROOT/scripts/timeout-env-keys.sh"
# shellcheck source=scripts/execution-env-keys.sh
source "$PROVENARCH_ROOT/scripts/execution-env-keys.sh"
# shellcheck source=scripts/run-status-heartbeat.sh
source "$PROVENARCH_ROOT/scripts/run-status-heartbeat.sh"
TARGET_REPOS_FILE="${TARGET_REPOS_FILE:-}"
PROFILE_ID="${PROFILE_ID:-}"
PROFILE_SOURCE_KIND="${PROFILE_SOURCE_KIND:-}"
PROFILE_SOURCE_KIND_EFFECTIVE="mixed"
EXPECTED_REPO_COUNT="${EXPECTED_REPO_COUNT:-}"
TMP_ROOT="${TMP_ROOT:-}"
RUN_STATUS_FILE="${RUN_STATUS_FILE:-}"
KEEP_TMP="${KEEP_TMP:-0}"
ITERATIONS="${ITERATIONS:-1}"
RUN_LOGS_TTL_HOURS="${RUN_LOGS_TTL_HOURS:-168}"
RUN_LOGS_MAX_RUNS="${RUN_LOGS_MAX_RUNS:-200}"
APPLY_TIMEOUTS_VIA_API="${ACP_APPLY_TIMEOUTS_VIA_API:-1}"
TASK_FIRST_ADMISSION="${ACP_LIVE_E2E_TASK_FIRST:-1}"

RUNTIME_STEP_TIMEOUT_SEC="${ACP_RUNTIME_STEP_TIMEOUT_SEC:-}"
RUNTIME_HEARTBEAT_SEC="${ACP_RUNTIME_HEARTBEAT_SEC:-}"
PIPELINE_TIMEOUT_SEC="${ACP_PIPELINE_TIMEOUT_SEC:-}"
PIPELINE_KILL_GRACE_SEC="${ACP_PIPELINE_KILL_GRACE_SEC:-}"
API_READY_TIMEOUT_SEC="${ACP_API_READY_TIMEOUT_SEC:-}"
API_INIT_TIMEOUT_SEC="${ACP_API_INIT_TIMEOUT_SEC:-}"
UI_INIT_POLL_TIMEOUT_SEC="${ACP_UI_INIT_POLL_TIMEOUT_SEC:-}"
UI_CANCEL_POLL_TIMEOUT_SEC="${ACP_UI_CANCEL_POLL_TIMEOUT_SEC:-}"

CREATED_TMP=0
SERVER_PID=""
FAILURE_REASON=""
TERMINATION_SIGNAL=""
API_SIM_STATUS="not_started"
API_INIT_RUN_ID=""
API_INIT_FINAL_STATUS=""
API_INIT_PROGRESS_GRACE_SEC=""
API_INIT_LAST_PROGRESS_AT=""
LAST_SIGNAL=""
HEADLESS_PROVIDER=""
HEADLESS_CMD=""
TARGET_PROFILE="generic"
RESOLVED_TARGET_REPOS_FILE=""
ORIGINAL_RESOLVED_TARGET_REPOS_FILE=""
ISOLATED_TARGET_REPOS_DIR=""
TARGET_REPOS_META_JSON=""
EXPECTED_REPO_COUNT_RESOLVED=0
EXPECTED_RUNS=0
COMPLETED_RUNS=0
EXPECTED_HEADLESS_RUNS=0
COMPLETED_HEADLESS_RUNS=0
RUNNING_RUNS_DETECTED=0
RUNNING_RUNS_BASELINE=0
RUNNING_RUNS_HEADLESS=0
MALFORMED_RUN_RESULTS_ROWS=0
SUMMARY_RESULT=""
SUMMARY_WRITTEN="no"
LAST_PIPELINE_STAGE="not_started"
LAST_RUNTIME_PROVIDER="unset"
LAST_PROGRESS_AT_UTC=""
LAST_PIPELINE_DEADLINE_AT=""
LAST_PIPELINE_TIMEOUT_ELAPSED_SEC=""
LAST_PIPELINE_DEADLINE_MISSED_BY_SEC=""
LAST_PIPELINE_LAST_PROGRESS_AT=""
LAST_WATCHDOG_TICK_AT=""
LAST_WATCHDOG_TICK_GAP_SEC=""
HOST_CLOCK_GAP_DETECTED="no"
LAST_PIPELINE_TIMED_OUT="no"
TERMINAL_PROCESS_EXIT_OVERRIDE=""
TIMEOUTS_API_APPLY_BASELINE_STATUS="not_applied"
TIMEOUTS_API_APPLY_BASELINE_EFFECTIVE=""
TIMEOUTS_API_APPLY_BASELINE_SOURCE=""
TIMEOUTS_API_APPLY_BASELINE_JSON=""
TIMEOUTS_API_APPLY_HEADLESS_STATUS="not_applied"
TIMEOUTS_API_APPLY_HEADLESS_EFFECTIVE=""
TIMEOUTS_API_APPLY_HEADLESS_SOURCE=""
TIMEOUTS_API_APPLY_HEADLESS_JSON=""

if [[ ! "$ITERATIONS" =~ ^[1-9][0-9]*$ ]]; then
  echo "ITERATIONS must be a positive integer, got: $ITERATIONS" >&2
  exit 1
fi
if [[ "$KEEP_TMP" != "0" && "$KEEP_TMP" != "1" ]]; then
  echo "KEEP_TMP must be 0 or 1, got: $KEEP_TMP" >&2
  exit 1
fi
if [[ "$APPLY_TIMEOUTS_VIA_API" != "0" && "$APPLY_TIMEOUTS_VIA_API" != "1" ]]; then
  echo "ACP_APPLY_TIMEOUTS_VIA_API must be 0 or 1, got: $APPLY_TIMEOUTS_VIA_API" >&2
  exit 1
fi
if [[ "$TASK_FIRST_ADMISSION" != "0" && "$TASK_FIRST_ADMISSION" != "1" ]]; then
  echo "ACP_LIVE_E2E_TASK_FIRST must be 0 or 1, got: $TASK_FIRST_ADMISSION" >&2
  exit 1
fi
if [[ ! "$RUN_LOGS_TTL_HOURS" =~ ^[1-9][0-9]*$ ]]; then
  echo "RUN_LOGS_TTL_HOURS must be a positive integer, got: $RUN_LOGS_TTL_HOURS" >&2
  exit 1
fi
if [[ ! "$RUN_LOGS_MAX_RUNS" =~ ^[1-9][0-9]*$ ]]; then
  echo "RUN_LOGS_MAX_RUNS must be a positive integer, got: $RUN_LOGS_MAX_RUNS" >&2
  exit 1
fi
if ! acp_ensure_no_legacy_env_set ""; then
  exit 1
fi
EXPECTED_RUNS=$((ITERATIONS * 4))
EXPECTED_HEADLESS_RUNS=$((ITERATIONS * 2))
RUN_RESULTS_EXPECTED_FIELDS=17

if [[ -z "$TMP_ROOT" ]]; then
  TMP_ROOT="$(mktemp -d -t provenarch-ai-advent.XXXXXX)"
  CREATED_TMP=1
else
  mkdir -p "$TMP_ROOT"
fi

HEADLESS_TMP_ROOT="$TMP_ROOT/headless"
BASELINE_TMP_ROOT="$TMP_ROOT/baseline"
WORKSPACE_HEADLESS="$HEADLESS_TMP_ROOT/arch-workspace"
WORKSPACE_BASELINE="$BASELINE_TMP_ROOT/arch-workspace"
LOG_DIR="$TMP_ROOT/logs"
SNAPSHOT_DIR="$TMP_ROOT/snapshots"
SUMMARY_PATH="$TMP_ROOT/session-summary.md"
FULL_RUN_LOG="$TMP_ROOT/full-run.log"
RUN_RESULTS_TSV="$TMP_ROOT/run-results.tsv"
VALIDATE_JSON="$TMP_ROOT/workspace-validate.json"
API_INIT_START_JSON="$TMP_ROOT/api-init-start.json"
API_INIT_STATUS_JSON="$TMP_ROOT/api-init-status.json"
API_INIT_ARTIFACTS_JSON="$TMP_ROOT/api-init-artifacts.json"
API_INIT_LOGS_JSON="$TMP_ROOT/api-init-logs.json"

mkdir -p "$LOG_DIR" "$SNAPSHOT_DIR"
: > "$FULL_RUN_LOG"
: > "$RUN_RESULTS_TSV"

# Keep original stdio for user-facing progress, but avoid tee/process-substitution
# in the critical path. All script output is persisted to FULL_RUN_LOG.
exec 4>&2
if [[ ! -t 4 ]]; then
  exec 4>/dev/null
fi
exec >>"$FULL_RUN_LOG" 2>&1

log() {
  local line
  line="[full-run] $*"
  printf '%s\n' "$line"
  printf '%s\n' "$line" >&4
}

require_cmd() {
  local cmd="$1"
  local hint="$2"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    FAILURE_REASON="missing required command: $cmd"
    local line
    line="missing required command: $cmd. $hint"
    echo "$line"
    echo "$line" >&4
    exit 1
  fi
}

die() {
  if [[ -z "$FAILURE_REASON" || "$FAILURE_REASON" == "unknown" ]]; then
    FAILURE_REASON="$1"
  fi
  local line
  line="[full-run][error] $1"
  echo "$line"
  echo "$line" >&4
  exit 1
}

on_termination_signal() {
  local signal_name="$1"
  TERMINATION_SIGNAL="$signal_name"
  FAILURE_REASON="infra_signal_terminated"
  log "received termination signal: $signal_name (last_pipeline_stage=$LAST_PIPELINE_STAGE last_runtime_provider=$LAST_RUNTIME_PROVIDER)"
  write_terminal_run_status "signal_terminated" "$(signal_exit_code "$signal_name")" "$(signal_status_token "$signal_name")" "$FAILURE_REASON" "$SUMMARY_WRITTEN"
  exit 1
}

count_running_runs_in_history() {
  local run_history_path="$1"
  if [[ ! -f "$run_history_path" ]]; then
    printf '0'
    return 0
  fi
  python3 - "$run_history_path" <<'PY'
import json
import sys

path = sys.argv[1]
payload = json.load(open(path, encoding='utf-8'))
items = payload.get('items')
if not isinstance(items, list):
    items = payload.get('runs')
if not isinstance(items, list):
    items = payload if isinstance(payload, list) else []
running = 0
for item in items:
    if not isinstance(item, dict):
        continue
    if str(item.get('status', '')).strip().lower() == 'running':
        running += 1
print(running)
PY
}

reconcile_active_runs_in_history() {
  local workspace_path="$1"
  local error_code="$2"
  local error_message="$3"
  local run_history_path="$workspace_path/reports/taskruns/run-history.json"
  if [[ ! -f "$run_history_path" ]]; then
    return 0
  fi
  local reconciled
  reconciled="$(python3 - "$run_history_path" "$error_code" "$error_message" <<'PY'
import json
import sys
from datetime import datetime, timezone

path, error_code, error_message = sys.argv[1], sys.argv[2], sys.argv[3]
with open(path, encoding="utf-8") as f:
    payload = json.load(f)

items = payload.get("items") if isinstance(payload, dict) else None
if not isinstance(items, list):
    items = payload.get("runs") if isinstance(payload, dict) else None
if not isinstance(items, list):
    items = payload if isinstance(payload, list) else []

finished_at = datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")
updated = 0
for item in items:
    if not isinstance(item, dict):
        continue
    status = str(item.get("status", "")).strip().lower()
    if status not in {"queued", "running"}:
        continue
    item["status"] = "failed"
    item["finished_at"] = finished_at
    item["error_code"] = error_code
    item["error"] = error_message
    updated += 1

if updated:
    with open(path, "w", encoding="utf-8") as f:
        json.dump(payload, f, ensure_ascii=True, indent=2)
        f.write("\n")
print(updated)
PY
)"
  if [[ "${reconciled:-0}" != "0" ]]; then
    log "reconciled active run-history entries after child failure: workspace=$workspace_path count=$reconciled error_code=$error_code"
  fi
}

run_result_row_exists() {
  local iteration="$1"
  local runtime_mode="$2"
  local runtime_provider="$3"
  local pipeline="$4"
  if [[ ! -f "$RUN_RESULTS_TSV" ]]; then
    return 1
  fi
  awk -F'\t' \
    -v iter="$iteration" \
    -v mode="$runtime_mode" \
    -v provider="$runtime_provider" \
    -v pipeline="$pipeline" \
    -v expected_fields="$RUN_RESULTS_EXPECTED_FIELDS" \
    'NF == expected_fields && $1 == iter && $2 == mode && $3 == provider && $4 == pipeline { ok=1 } END { exit ok ? 0 : 1 }' \
    "$RUN_RESULTS_TSV"
}

resolve_failed_run_id_from_workspace() {
  local workspace_path="$1"
  local pipeline="$2"
  python3 - "$workspace_path" "$pipeline" <<'PY'
import json
import re
import sys
from pathlib import Path

workspace = Path(sys.argv[1])
pipeline = (sys.argv[2] or "").strip()
taskruns = workspace / "reports" / "taskruns"
candidates = []
if taskruns.exists():
    for path in taskruns.glob("run_*-quality.json"):
        run_id = path.name[:-len("-quality.json")]
        candidates.append((path.stat().st_mtime, run_id))
    raw_root = taskruns / "raw"
    if raw_root.exists():
        for path in raw_root.glob("run_*-meta.json"):
            text = path.name
            match = re.match(r"(run_[0-9_]+)-", text)
            if not match:
                continue
            if pipeline and f"-{pipeline}." not in text:
                continue
            candidates.append((path.stat().st_mtime, match.group(1)))

if not candidates:
    sys.exit(0)
candidates.sort()
print(candidates[-1][1])
PY
}

append_run_result_row_once() {
  local iteration="$1"
  local runtime_mode="$2"
  local runtime_provider="$3"
  local pipeline="$4"
  local run_id="$5"
  local status="$6"
  local workspace_path="$7"
  local output_path="$8"

  if [[ -z "$run_id" ]]; then
    run_id="$(resolve_failed_run_id_from_workspace "$workspace_path" "$pipeline" || true)"
  fi
  if [[ -z "$run_id" ]]; then
    return 0
  fi
  if run_result_row_exists "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline"; then
    return 0
  fi
  if [[ -z "$status" ]]; then
    status="failed"
  fi

  local runtime_label="$runtime_mode"
  if [[ "$runtime_mode" == "headless" ]]; then
    runtime_label="${runtime_mode}:${runtime_provider}"
  fi
  if [[ -d "$workspace_path" ]]; then
    snapshot_run_artifacts "$run_id" "$runtime_label" "$pipeline" "$iteration" "$workspace_path" || true
  fi

  local quality_path="$workspace_path/reports/taskruns/${run_id}-quality.json"
  local signal_score=0
  local semantic_entities=0
  local semantic_edges=0
  local findings=0
  local questions=0
  local coverage_observed=0
  local coverage_missing=0
  local warnings=0
  local artifact_quality_count=0
  local domain_collect_steps=0
  local mock_flag=0
  local zero_signal=1
  local runtime_versions="${runtime_mode}@unknown"

  if [[ -f "$quality_path" ]]; then
    local metrics
    if metrics="$(quality_metrics "$quality_path" 2>/dev/null)"; then
      local quality_status
      IFS=$'\t' read -r quality_status signal_score semantic_entities semantic_edges findings questions coverage_observed coverage_missing warnings artifact_quality_count domain_collect_steps mock_flag zero_signal runtime_versions <<<"$metrics"
    else
      warnings=1
    fi
  fi

  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "$status" "$signal_score" "$semantic_entities" "$semantic_edges" "$findings" "$questions" "$coverage_observed" "$coverage_missing" "$warnings" "$runtime_versions" "$quality_path" "$output_path" >> "$RUN_RESULTS_TSV"
}

append_api_init_run_result_row() {
  local status="${1:-failed}"
  if [[ -z "${API_INIT_RUN_ID:-}" ]]; then
    return 0
  fi
  append_run_result_row_once "1" "fake" "${HEADLESS_PROVIDER:-fake}" "init" "$API_INIT_RUN_ID" "$status" "$WORKSPACE_BASELINE" "$API_INIT_STATUS_JSON"
}

is_successful_noop_refresh() {
  local workspace_path="$1"
  local run_id="$2"
  local execution_path="$workspace_path/reports/taskruns/$run_id/refresh-execution.json"

  [[ -f "$execution_path" ]] || return 1
  python3 - "$execution_path" "$run_id" <<'PY'
import json
import sys

try:
    payload = json.load(open(sys.argv[1], encoding="utf-8"))
except (OSError, json.JSONDecodeError):
    raise SystemExit(1)

valid = (
    payload.get("version") == 1
    and payload.get("run_id") == sys.argv[2]
    and payload.get("mode") == "no_op"
    and payload.get("plan_decision") == "unchanged_candidate"
    and payload.get("provider_steps_skipped") is True
)
raise SystemExit(0 if valid else 1)
PY
}

refresh_runtime_cycle_metrics() {
  if [[ -f "$RUN_RESULTS_TSV" ]]; then
    local metrics
    metrics="$(awk -F'\t' -v expected_fields="$RUN_RESULTS_EXPECTED_FIELDS" '
      NF == 0 { next }
      NF == expected_fields {
        valid++
        if ($2 == "headless") {
          headless++
        }
        next
      }
      { malformed++ }
      END { print (valid+0) "\t" (headless+0) "\t" (malformed+0) }
    ' "$RUN_RESULTS_TSV")"
    IFS=$'\t' read -r COMPLETED_RUNS COMPLETED_HEADLESS_RUNS MALFORMED_RUN_RESULTS_ROWS <<<"$metrics"
  else
    COMPLETED_RUNS=0
    COMPLETED_HEADLESS_RUNS=0
    MALFORMED_RUN_RESULTS_ROWS=0
  fi

  local baseline_history_path="$WORKSPACE_BASELINE/reports/taskruns/run-history.json"
  local headless_history_path="$WORKSPACE_HEADLESS/reports/taskruns/run-history.json"
  RUNNING_RUNS_BASELINE="$(count_running_runs_in_history "$baseline_history_path")"
  RUNNING_RUNS_HEADLESS="$(count_running_runs_in_history "$headless_history_path")"
  RUNNING_RUNS_DETECTED=$((RUNNING_RUNS_BASELINE + RUNNING_RUNS_HEADLESS))
}

validate_runtime_cycle_completion() {
  refresh_runtime_cycle_metrics

  local failed=0
  if (( COMPLETED_RUNS != EXPECTED_RUNS )); then
    log "completion invariant failed: expected_runs=$EXPECTED_RUNS completed_runs=$COMPLETED_RUNS"
    failed=1
  fi
  if (( COMPLETED_HEADLESS_RUNS != EXPECTED_HEADLESS_RUNS )); then
    log "completion invariant failed: expected_headless_runs=$EXPECTED_HEADLESS_RUNS completed_headless_runs=$COMPLETED_HEADLESS_RUNS"
    failed=1
  fi
  if (( MALFORMED_RUN_RESULTS_ROWS > 0 )); then
    log "completion invariant failed: malformed_run_results_rows=$MALFORMED_RUN_RESULTS_ROWS expected_fields=$RUN_RESULTS_EXPECTED_FIELDS"
    failed=1
  fi

  if [[ ! -f "$RUN_RESULTS_TSV" ]]; then
    log "completion invariant failed: missing $RUN_RESULTS_TSV"
    failed=1
  else
    for iteration in $(seq 1 "$ITERATIONS"); do
      if ! awk -F'\t' -v iter="$iteration" -v expected_fields="$RUN_RESULTS_EXPECTED_FIELDS" 'NF == expected_fields && $1 == iter && $2 == "headless" && $4 == "init" { ok=1 } END { exit ok ? 0 : 1 }' "$RUN_RESULTS_TSV"; then
        log "completion invariant failed: missing headless init for iteration=$iteration"
        failed=1
      fi
      if ! awk -F'\t' -v iter="$iteration" -v expected_fields="$RUN_RESULTS_EXPECTED_FIELDS" 'NF == expected_fields && $1 == iter && $2 == "headless" && $4 == "refresh" { ok=1 } END { exit ok ? 0 : 1 }' "$RUN_RESULTS_TSV"; then
        log "completion invariant failed: missing headless refresh for iteration=$iteration"
        failed=1
      fi
    done
  fi

  local baseline_history_path="$WORKSPACE_BASELINE/reports/taskruns/run-history.json"
  local headless_history_path="$WORKSPACE_HEADLESS/reports/taskruns/run-history.json"
  if [[ ! -f "$baseline_history_path" ]]; then
    log "completion invariant failed: missing run history $baseline_history_path"
    failed=1
  fi
  if [[ ! -f "$headless_history_path" ]]; then
    log "completion invariant failed: missing run history $headless_history_path"
    failed=1
  elif (( RUNNING_RUNS_DETECTED > 0 )); then
    log "completion invariant failed: detected running runs in history (baseline=$RUNNING_RUNS_BASELINE headless=$RUNNING_RUNS_HEADLESS total=$RUNNING_RUNS_DETECTED)"
    failed=1
  fi

  if (( failed != 0 )); then
    if [[ -z "$FAILURE_REASON" || "$FAILURE_REASON" == "unknown" ]]; then
      FAILURE_REASON="infra_incomplete_cycle"
    fi
    return 1
  fi
  return 0
}

slugify() {
  local value
  value="$(echo "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
  if [[ -z "$value" ]]; then
    value="run"
  fi
  printf '%s' "$value"
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

validate_target_repos_file() {
  TARGET_REPOS_META_JSON="${TARGET_REPOS_META_JSON:-$TMP_ROOT/target-repos-meta.json}"
  python3 "$PROVENARCH_ROOT/scripts/resolve-repos-meta.py" \
    --repos-file "$RESOLVED_TARGET_REPOS_FILE" \
    --expected-repo-count "$EXPECTED_REPO_COUNT" \
    --source-kind "$PROFILE_SOURCE_KIND" \
    --profile-id "$PROFILE_ID" \
    --out "$TARGET_REPOS_META_JSON"
}

read_target_repos_meta() {
  local key
  local value
  local resolved_target_profile="generic"
  local resolved_source_kind="mixed"
  local resolved_expected_count="0"
  while IFS='=' read -r key value; do
    case "$key" in
      target_profile) resolved_target_profile="$value" ;;
      profile_source_kind) resolved_source_kind="$value" ;;
      expected_repo_count) resolved_expected_count="$value" ;;
    esac
  done < <(acp_read_repos_meta_fields "$TARGET_REPOS_META_JSON")

  TARGET_PROFILE="${resolved_target_profile:-generic}"
  PROFILE_SOURCE_KIND_EFFECTIVE="${resolved_source_kind:-mixed}"
  EXPECTED_REPO_COUNT_RESOLVED="${resolved_expected_count:-0}"
}

prepare_isolated_target_repos_file() {
  case "$PROFILE_SOURCE_KIND_EFFECTIVE" in
    path|mixed)
      ;;
    *)
      return 0
      ;;
  esac

  ORIGINAL_RESOLVED_TARGET_REPOS_FILE="$RESOLVED_TARGET_REPOS_FILE"
  local isolated_repos_file="$TMP_ROOT/target-repos.live-isolated.yaml"
  local isolated_repos_dir="$TMP_ROOT/source-repos"
  python3 "$PROVENARCH_ROOT/scripts/prepare-live-repos-file.py" \
    --repos-file "$ORIGINAL_RESOLVED_TARGET_REPOS_FILE" \
    --work-dir "$isolated_repos_dir" \
    --out "$isolated_repos_file" \
    --make-read-only
  RESOLVED_TARGET_REPOS_FILE="$isolated_repos_file"
  ISOLATED_TARGET_REPOS_DIR="$isolated_repos_dir"
  TARGET_REPOS_META_JSON="$TMP_ROOT/target-repos-meta.live-isolated.json"
  validate_target_repos_file
  read_target_repos_meta
  log "target path repos isolated: original=$ORIGINAL_RESOLVED_TARGET_REPOS_FILE active=$RESOLVED_TARGET_REPOS_FILE source_repos=$isolated_repos_dir"
}

restore_isolated_target_repos_write_bits() {
  if [[ -n "$ISOLATED_TARGET_REPOS_DIR" && -d "$ISOLATED_TARGET_REPOS_DIR" ]]; then
    chmod -R u+w "$ISOLATED_TARGET_REPOS_DIR" 2>/dev/null || true
  fi
}

resolve_effective_timeouts_from_workspace() {
  local workspace_path="$1"
  local manifest_path="$workspace_path/workspace.yaml"
  if [[ ! -f "$manifest_path" ]]; then
    die "workspace manifest is missing for timeout resolution: $manifest_path"
  fi
  local resolved_lines
  resolved_lines="$(python3 "$PROVENARCH_ROOT/scripts/resolve-timeout-profile.py" \
    --workspace-manifest "$manifest_path" \
    --format kv)"
  while IFS='=' read -r key value; do
    [[ -z "$key" ]] && continue
    case "$key" in
      step_timeout_sec) RUNTIME_STEP_TIMEOUT_SEC="$value" ;;
      heartbeat_sec) RUNTIME_HEARTBEAT_SEC="$value" ;;
      pipeline_timeout_sec) PIPELINE_TIMEOUT_SEC="$value" ;;
      pipeline_kill_grace_sec) PIPELINE_KILL_GRACE_SEC="$value" ;;
      api_ready_timeout_sec) API_READY_TIMEOUT_SEC="$value" ;;
      api_init_timeout_sec) API_INIT_TIMEOUT_SEC="$value" ;;
      ui_init_poll_timeout_sec) UI_INIT_POLL_TIMEOUT_SEC="$value" ;;
      ui_cancel_poll_timeout_sec) UI_CANCEL_POLL_TIMEOUT_SEC="$value" ;;
    esac
  done <<<"$resolved_lines"
}

apply_runtime_timeouts_via_api() {
  local workspace_label="$1"
  local workspace_path="$2"
  local api_base="$3"
  local payload_path="$TMP_ROOT/runtime-timeouts-${workspace_label}-payload.json"
  local put_response_path="$TMP_ROOT/runtime-timeouts-${workspace_label}-put-response.json"
  local get_response_path="$TMP_ROOT/runtime-timeouts-${workspace_label}-get-response.json"

  cat >"$payload_path" <<EOF
{"timeouts":{"step_timeout_sec":${RUNTIME_STEP_TIMEOUT_SEC},"heartbeat_sec":${RUNTIME_HEARTBEAT_SEC},"pipeline_timeout_sec":${PIPELINE_TIMEOUT_SEC},"pipeline_kill_grace_sec":${PIPELINE_KILL_GRACE_SEC},"api_ready_timeout_sec":${API_READY_TIMEOUT_SEC},"api_init_timeout_sec":${API_INIT_TIMEOUT_SEC},"ui_init_poll_timeout_sec":${UI_INIT_POLL_TIMEOUT_SEC},"ui_cancel_poll_timeout_sec":${UI_CANCEL_POLL_TIMEOUT_SEC}}}
EOF

  if ! curl -fsS -X PUT \
    -H 'Content-Type: application/json' \
    --data @"$payload_path" \
    "$api_base/api/runtime/timeouts" >"$put_response_path"; then
    die "failed to PUT /api/runtime/timeouts for workspace=$workspace_label path=$workspace_path"
  fi

  if ! curl -fsS "$api_base/api/runtime/timeouts" >"$get_response_path"; then
    die "failed to GET /api/runtime/timeouts for workspace=$workspace_label path=$workspace_path"
  fi

  local parsed
  if ! parsed="$(python3 - "$get_response_path" "$workspace_label" \
    "$RUNTIME_STEP_TIMEOUT_SEC" "$RUNTIME_HEARTBEAT_SEC" "$PIPELINE_TIMEOUT_SEC" "$PIPELINE_KILL_GRACE_SEC" \
    "$API_READY_TIMEOUT_SEC" "$API_INIT_TIMEOUT_SEC" "$UI_INIT_POLL_TIMEOUT_SEC" "$UI_CANCEL_POLL_TIMEOUT_SEC" <<'PY'
import json
import sys

payload_path = sys.argv[1]
workspace_label = sys.argv[2]
keys = [
    "step_timeout_sec",
    "heartbeat_sec",
    "pipeline_timeout_sec",
    "pipeline_kill_grace_sec",
    "api_ready_timeout_sec",
    "api_init_timeout_sec",
    "ui_init_poll_timeout_sec",
    "ui_cancel_poll_timeout_sec",
]
expected = {key: int(value) for key, value in zip(keys, sys.argv[3:11])}
payload = json.load(open(payload_path, encoding="utf-8"))
effective = payload.get("effective") or {}
persisted = payload.get("persisted") or {}
source = payload.get("source") or {}

effective_pairs = []
source_pairs = []
for key in keys:
    if key not in effective:
        raise SystemExit(f"runtime timeouts API ({workspace_label}) missing effective.{key}")
    if key not in persisted:
        raise SystemExit(f"runtime timeouts API ({workspace_label}) missing persisted.{key}")
    try:
        effective_value = int(effective[key])
        persisted_value = int(persisted[key])
    except Exception as exc:  # pragma: no cover
        raise SystemExit(f"runtime timeouts API ({workspace_label}) invalid numeric value for {key}: {exc}")
    expected_value = expected[key]
    if effective_value != expected_value:
        raise SystemExit(
            f"runtime timeouts API ({workspace_label}) effective mismatch for {key}: expected={expected_value} got={effective_value}"
        )
    if persisted_value != expected_value:
        raise SystemExit(
            f"runtime timeouts API ({workspace_label}) persisted mismatch for {key}: expected={expected_value} got={persisted_value}"
        )
    effective_pairs.append(f"{key}={effective_value}")
    source_pairs.append(f"{key}:{source.get(key, 'unknown')}")

print("effective\t" + ",".join(effective_pairs))
print("source\t" + ",".join(source_pairs))
PY
)"; then
    die "runtime timeouts API validation failed for workspace=$workspace_label path=$workspace_path (see $get_response_path)"
  fi

  local effective_line source_line
  effective_line="$(echo "$parsed" | awk -F'\t' '$1=="effective" {print $2}' | tail -n1)"
  source_line="$(echo "$parsed" | awk -F'\t' '$1=="source" {print $2}' | tail -n1)"
  if [[ -z "$effective_line" || -z "$source_line" ]]; then
    die "runtime timeouts API validation returned empty result for workspace=$workspace_label"
  fi

  log "runtime timeouts applied via API workspace=$workspace_label path=$workspace_path effective={$effective_line} source={$source_line}"
  if [[ "$workspace_label" == "baseline" ]]; then
    TIMEOUTS_API_APPLY_BASELINE_STATUS="applied"
    TIMEOUTS_API_APPLY_BASELINE_EFFECTIVE="$effective_line"
    TIMEOUTS_API_APPLY_BASELINE_SOURCE="$source_line"
    TIMEOUTS_API_APPLY_BASELINE_JSON="$get_response_path"
  else
    TIMEOUTS_API_APPLY_HEADLESS_STATUS="applied"
    TIMEOUTS_API_APPLY_HEADLESS_EFFECTIVE="$effective_line"
    TIMEOUTS_API_APPLY_HEADLESS_SOURCE="$source_line"
    TIMEOUTS_API_APPLY_HEADLESS_JSON="$get_response_path"
  fi
}

allocate_free_port() {
  python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
}

wait_for_health() {
  local api_base="$1"
  local deadline
  deadline=$((SECONDS + API_READY_TIMEOUT_SEC))
  while (( SECONDS < deadline )); do
    if curl -fsS "$api_base/api/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

stop_server() {
  if [[ -n "$SERVER_PID" ]]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" 2>/dev/null || true
    SERVER_PID=""
  fi
}

copy_if_exists() {
  local src="$1"
  local dst="$2"
  if [[ -f "$src" ]]; then
    mkdir -p "$(dirname "$dst")"
    cp "$src" "$dst"
  fi
}

copy_tree_if_exists() {
  local src="$1"
  local dst="$2"
  if [[ -d "$src" ]]; then
    mkdir -p "$(dirname "$dst")"
    rm -rf "$dst"
    cp -R "$src" "$dst"
  fi
}

snapshot_run_artifacts() {
  local run_id="$1"
  local runtime="$2"
  local pipeline="$3"
  local iteration="$4"
  local workspace_path="$5"

  local dst="$SNAPSHOT_DIR/$run_id"
  mkdir -p "$dst"

  copy_if_exists "$workspace_path/reports/as-is/overview.md" "$dst/reports/as-is/overview.md"
  copy_if_exists "$workspace_path/reports/findings/findings.md" "$dst/reports/findings/findings.md"
  copy_if_exists "$workspace_path/reports/coverage/summary.md" "$dst/reports/coverage/summary.md"
  copy_if_exists "$workspace_path/reports/coverage/open-questions.md" "$dst/reports/coverage/open-questions.md"
  copy_if_exists "$workspace_path/reports/taskruns/${run_id}-quality.json" "$dst/reports/taskruns/${run_id}-quality.json"
  copy_if_exists "$workspace_path/reports/taskruns/run-history.json" "$dst/reports/taskruns/run-history.json"
  copy_if_exists "$workspace_path/reports/taskruns/run-history.json.last-good" "$dst/reports/taskruns/run-history.json.last-good"
  # W25 frontend snapshot inspection must consume the product-authored Task registry
  # produced by public Task/Attempt admission. Never reconstruct identity from legacy
  # run history when the backend workspace is copied into the frontend workspace.
  copy_if_exists "$workspace_path/reports/taskruns/task-history.json" "$dst/reports/taskruns/task-history.json"
  copy_if_exists "$workspace_path/reports/taskruns/task-history.json.last-good" "$dst/reports/taskruns/task-history.json.last-good"
  copy_tree_if_exists "$workspace_path/reports/taskruns/${run_id}" "$dst/reports/taskruns/${run_id}"
  for taskrun_json in "$workspace_path/reports/taskruns/${run_id}-"*.json; do
    if [[ -f "$taskrun_json" ]]; then
      copy_if_exists "$taskrun_json" "$dst/reports/taskruns/$(basename "$taskrun_json")"
    fi
  done

  local run_slug
  run_slug="$(slugify "$run_id")"
  copy_if_exists "$workspace_path/reports/taskruns/logs/${run_slug}.ndjson" "$dst/reports/taskruns/logs/${run_slug}.ndjson"

  cat > "$dst/snapshot-meta.txt" <<META
iteration=$iteration
runtime=$runtime
pipeline=$pipeline
run_id=$run_id
workspace=$workspace_path
captured_at=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
META
}

quality_metrics() {
  local quality_path="$1"
  python3 - "$quality_path" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, encoding='utf-8') as f:
    payload = json.load(f)

totals = payload.get('totals') or {}
steps = payload.get('steps') or []
runtime_versions = payload.get('runtime_versions') or []

def metric(name: str) -> int:
    value = totals.get(name, 0)
    if isinstance(value, bool):
        return int(value)
    if isinstance(value, (int, float)):
        return int(value)
    return 0

signal = metric('signal_score')
entities = metric('semantic_entities')
edges = metric('semantic_edges')
findings = metric('findings_count')
questions = metric('questions_count')
coverage_observed = metric('coverage_observed')
coverage_missing = metric('coverage_missing')
warnings = metric('warnings_count')
run_warnings = payload.get('run_warnings') or []
if not isinstance(run_warnings, list):
    run_warnings = []
artifact_quality_count = sum(1 for item in run_warnings if str(item).startswith('artifact_quality:'))

runtime_blob = ",".join(str(item) for item in runtime_versions)
runtime_lower = runtime_blob.lower()
mock_flag = 1 if ('mock' in runtime_lower or 'fake' in runtime_lower) else 0

signal_components = entities + edges + findings + questions + coverage_observed + coverage_missing
zero_signal = 1 if signal_components == 0 else 0

domain_collect_steps = 0
for step in steps:
    step_id = str(step.get('step_id', ''))
    domain_id = str(step.get('domain_id', '')).strip()
    if 'step1.collect' in step_id and domain_id:
        domain_collect_steps += 1

status = str(payload.get('status', ''))
print("\t".join([
    status,
    str(signal),
    str(entities),
    str(edges),
    str(findings),
    str(questions),
    str(coverage_observed),
    str(coverage_missing),
    str(warnings),
    str(artifact_quality_count),
    str(domain_collect_steps),
    str(mock_flag),
    str(zero_signal),
    runtime_blob,
]))
PY
}

check_ai_advent_text_signal() {
  local workspace_path="$1"
  local run_id="$2"
  python3 - "$workspace_path" "$run_id" <<'PY'
import os
import re
import sys

workspace = sys.argv[1]
run_id = sys.argv[2]
findings_path = os.path.join(workspace, 'reports/findings/findings.md')
questions_path = os.path.join(workspace, 'reports/coverage/open-questions.md')
overview_path = os.path.join(workspace, 'reports/as-is/overview.md')

for path in (findings_path, questions_path, overview_path):
    if not os.path.isfile(path):
        print(f"missing artifact: {path}")
        sys.exit(2)

findings = open(findings_path, encoding='utf-8').read()
questions = open(questions_path, encoding='utf-8').read()
overview = open(overview_path, encoding='utf-8').read()

findings_items = len(re.findall(r'^- ', findings, flags=re.MULTILINE))
question_items = len(re.findall(r'^- ', questions, flags=re.MULTILINE))
overview_lines = len([line for line in overview.splitlines() if line.strip()])

low_findings = bool(re.search(r'no findings|none\.?$', findings.lower(), flags=re.MULTILINE))
low_questions = bool(re.search(r'no open questions|none\.?$', questions.lower(), flags=re.MULTILINE))

if overview_lines < 3:
    print(f"low textual signal for run {run_id}: overview too short ({overview_lines} lines)")
    sys.exit(3)

if findings_items == 0 and question_items == 0 and low_findings and low_questions:
    print(f"low textual signal for run {run_id}: findings/questions look empty")
    sys.exit(4)

print(f"text_signal_ok findings_items={findings_items} questions_items={question_items} overview_lines={overview_lines}")
PY
}

check_headless_refresh_semantic_quality() {
  local workspace_path="$1"
  local run_id="$2"
  python3 - "$workspace_path" "$run_id" <<'PY'
import glob
import json
import os
import re
import sys

workspace = sys.argv[1]
run_id = sys.argv[2]
findings_path = os.path.join(workspace, "reports/findings/findings.md")
coverage_path = os.path.join(workspace, "reports/coverage/summary.md")
questions_path = os.path.join(workspace, "reports/coverage/open-questions.md")

for path in (findings_path, coverage_path, questions_path):
    if not os.path.isfile(path):
        print(f"missing artifact: {path}")
        sys.exit(2)

findings_text = open(findings_path, encoding="utf-8").read()
coverage_lines = open(coverage_path, encoding="utf-8").read().splitlines()
questions_lines = open(questions_path, encoding="utf-8").read().splitlines()

def normalize(value: str) -> str:
    value = value.strip().lower().replace("_", " ").replace("-", " ")
    value = " ".join(value.split())
    return value

def canonical_gap_category(value: str) -> str:
    text = normalize(value)
    if "owner team id" in text or ("owner" in text and any(token in text for token in ("mapping", "team", "ownership"))):
        return "owner_mapping"
    if "runbook" in text or ("operational" in text and any(token in text for token in ("handoff", "procedure", "playbook"))):
        return "operational_runbook"
    if "third party" in text or ("external" in text and any(token in text for token in ("dependency", "system", "integration"))):
        return "external_dependency"
    if any(token in text for token in ("datastore", "database", "storage", "persistence")):
        return "datastore_storage"
    if any(token in text for token in ("ci cd", "cicd", "continuous integration", "workflow", "pipeline")):
        return "cicd"
    if any(token in text for token in ("api", "interface", "endpoint")):
        return "api_interface"
    return ""

missing_values = []
in_missing = False
for line in coverage_lines:
    if line.startswith("## "):
        in_missing = line.strip().lower() == "## missing"
        continue
    if in_missing and line.strip().startswith("- "):
        missing_values.append(normalize(line.strip()[2:].strip("`")))

if len(missing_values) != len(set(missing_values)):
    print(f"semantic quality failed for run {run_id}: duplicate coverage.missing terms after canonicalization")
    sys.exit(3)

owner_gap = any(item in {"owner mappings", "owner mapping", "owner team mappings", "owner team mapping"} for item in missing_values)
gap_categories = sorted({category for item in missing_values if (category := canonical_gap_category(item))})
critical_gap = bool(gap_categories)
owner_gap = owner_gap or "owner_mapping" in gap_categories
if critical_gap and re.search(r"no findings reported\.", findings_text, flags=re.IGNORECASE):
    print(
        f"semantic quality failed for run {run_id}: critical coverage gaps exist but findings report is empty "
        f"(categories={','.join(gap_categories)})"
    )
    sys.exit(4)

question_texts = []
for line in questions_lines:
    line = line.strip()
    if not line.startswith("- "):
        continue
    body = line[2:].strip()
    match = re.match(r"`[^`]+`\s*(.*)$", body)
    text = match.group(1) if match else body
    question_texts.append(normalize(text))

if len(question_texts) != len(set(question_texts)):
    print(f"semantic quality failed for run {run_id}: duplicate open-question texts after normalization")
    sys.exit(5)

critical_marker = "semantic_guard: critical_off_topic_drift in refresh.step1.collect"
taskrun_glob = os.path.join(workspace, "reports", "taskruns", run_id, "**", "runtime-execution.json")
for taskrun_path in sorted(glob.glob(taskrun_glob, recursive=True)):
    try:
        payload = json.load(open(taskrun_path, encoding="utf-8"))
    except Exception:
        continue
    if str(payload.get("step_id") or "").strip() != "refresh.step1.collect":
        continue
    warnings = payload.get("warnings") or []
    if any(critical_marker in str(item) for item in warnings):
        print(f"semantic quality failed for run {run_id}: critical off-topic drift marker found in {taskrun_path}")
        sys.exit(6)

print(
    "semantic_quality_ok "
    f"owner_gap={int(owner_gap)} "
    f"critical_gap_categories={','.join(gap_categories) if gap_categories else '-'} "
    f"coverage_missing={len(missing_values)} "
    f"open_questions={len(question_texts)}"
)
PY
}

run_cli_pipeline() {
  local runtime_mode="$1"
  local runtime_provider="$2"
  local pipeline="$3"
  local iteration="$4"
  local workspace_path="$5"
  local previous_signal="$6"

  local runtime_label="$runtime_mode"
  if [[ "$runtime_mode" == "headless" ]]; then
    runtime_label="${runtime_mode}:${runtime_provider}"
  fi
  LAST_PIPELINE_STAGE="iteration=${iteration} runtime=${runtime_label} pipeline=${pipeline}"
  if [[ "$runtime_mode" == "headless" ]]; then
    LAST_RUNTIME_PROVIDER="$runtime_provider"
  else
    LAST_RUNTIME_PROVIDER="fake"
  fi
  write_running_run_status_heartbeat

  local output_path="$LOG_DIR/run-iter${iteration}-${runtime_mode}-${runtime_provider}-${pipeline}.log"
  local quality_path
  local run_id
  local status

  log "run: iteration=$iteration runtime=$runtime_label pipeline=$pipeline"
  local -a run_cmd
  if [[ "$TASK_FIRST_ADMISSION" == "1" ]]; then
    local task_api_port
    task_api_port="$(allocate_free_port)"
    local task_api_listen
    task_api_listen="127.0.0.1:$task_api_port"
    run_cmd=(
      python3 "$PROVENARCH_ROOT/scripts/internal/live-task-attempt.py"
      --acp-bin "$ACP_BIN"
      --workspace "$workspace_path"
      --pipeline "$pipeline"
      --runtime "$runtime_mode"
      --runtime-provider "$runtime_provider"
      --listen "$task_api_listen"
      --output "$output_path"
      --server-log "$output_path.server.log"
      --api-ready-timeout-sec "$API_READY_TIMEOUT_SEC"
      --pipeline-timeout-sec "$PIPELINE_TIMEOUT_SEC"
    )
    log "task-first admission: public Task/Attempt API runtime=$runtime_label pipeline=$pipeline listen=$task_api_listen"
  else
    run_cmd=(
      "$ACP_BIN" run
      --workspace "$workspace_path"
      --pipeline "$pipeline"
      --runtime "$runtime_mode"
      --non-interactive
      --run-logs-ttl-hours "$RUN_LOGS_TTL_HOURS"
      --run-logs-max-runs "$RUN_LOGS_MAX_RUNS"
    )
    if [[ "$runtime_mode" == "headless" ]]; then
      run_cmd+=(--runtime-provider "$runtime_provider")
    fi
  fi

  local monitor_meta="$TMP_ROOT/.pipeline-monitor-${iteration}-${runtime_mode}-${runtime_provider}-${pipeline}.json"
  rm -f "$monitor_meta"
  LAST_PIPELINE_DEADLINE_AT=""
  LAST_PIPELINE_TIMEOUT_ELAPSED_SEC=""
  LAST_PIPELINE_DEADLINE_MISSED_BY_SEC=""
  LAST_PIPELINE_LAST_PROGRESS_AT=""
  LAST_WATCHDOG_TICK_AT=""
  LAST_WATCHDOG_TICK_GAP_SEC=""
  HOST_CLOCK_GAP_DETECTED="no"
  LAST_PIPELINE_TIMED_OUT="no"
  local watchdog_cmd=(
    python3 "$PROVENARCH_ROOT/scripts/pipeline-watchdog.py"
    --timeout-sec "$PIPELINE_TIMEOUT_SEC"
    --grace-sec "$PIPELINE_KILL_GRACE_SEC"
    --heartbeat-sec "$RUNTIME_HEARTBEAT_SEC"
    --output "$output_path"
    --metadata "$monitor_meta"
    --workspace "$workspace_path"
  )
  if [[ -n "${RUN_STATUS_FILE:-}" ]]; then
    watchdog_cmd+=(--status-file "$RUN_STATUS_FILE")
  fi
  watchdog_cmd+=(
    --last-pipeline-stage "$LAST_PIPELINE_STAGE"
    --last-runtime-provider "$LAST_RUNTIME_PROVIDER"
    --progress-label "iteration=$iteration runtime=$runtime_label pipeline=$pipeline"
    -- "${run_cmd[@]}"
  )
  local run_exit=0
  if "${watchdog_cmd[@]}"; then
    run_exit=0
  else
    run_exit=$?
  fi

  if [[ -f "$monitor_meta" ]]; then
    local monitor_exports
    monitor_exports="$(python3 - "$monitor_meta" <<'PY'
import json
import shlex
import sys

payload = json.load(open(sys.argv[1], encoding="utf-8"))
fields = {
    "LAST_PIPELINE_DEADLINE_AT": payload.get("pipeline_deadline_at", ""),
    "LAST_PIPELINE_TIMEOUT_ELAPSED_SEC": str(payload.get("pipeline_timeout_elapsed_sec", "")),
    "LAST_PIPELINE_DEADLINE_MISSED_BY_SEC": str(payload.get("deadline_missed_by_sec", "")),
    "LAST_PIPELINE_LAST_PROGRESS_AT": payload.get("last_progress_at", ""),
    "LAST_PROGRESS_AT_UTC": payload.get("last_progress_at", ""),
    "LAST_WATCHDOG_TICK_AT": payload.get("last_watchdog_tick_at", ""),
    "LAST_WATCHDOG_TICK_GAP_SEC": str(payload.get("max_watchdog_tick_gap_sec", "")),
    "HOST_CLOCK_GAP_DETECTED": "yes" if payload.get("infra_host_sleep_or_clock_jump_detected") else "no",
    "LAST_PIPELINE_TIMED_OUT": "yes" if payload.get("timed_out") else "no",
}
for key, value in fields.items():
    print(f"{key}={shlex.quote(str(value))}")
PY
)"
    eval "$monitor_exports"
  fi

  if [[ "$HOST_CLOCK_GAP_DETECTED" == "yes" ]]; then
    log "diagnostic infra_host_sleep_or_clock_jump_detected: iteration=$iteration runtime=$runtime_label pipeline=$pipeline max_tick_gap_sec=${LAST_WATCHDOG_TICK_GAP_SEC:-unknown}"
  fi

  if [[ "$run_exit" -eq 124 && "$LAST_PIPELINE_TIMED_OUT" == "yes" ]]; then
    status="$(sed -n 's/^status: //p' "$output_path" | tail -n1 | tr -d '\r')"
    run_id="$(sed -n 's/^run_id: //p' "$output_path" | tail -n1 | tr -d '\r')"
    reconcile_active_runs_in_history "$workspace_path" "runtime_timeout" "pipeline timed out after ${PIPELINE_TIMEOUT_SEC}s (grace ${PIPELINE_KILL_GRACE_SEC}s)"
    append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "${status:-failed}" "$workspace_path" "$output_path"
    FAILURE_REASON="runtime_timeout"
    TERMINATION_SIGNAL="timeout"
    TERMINAL_PROCESS_EXIT_OVERRIDE="124"
    write_terminal_run_status "process_failed" "124" "timeout" "runtime_timeout" "$SUMMARY_WRITTEN"
    die "pipeline timed out after ${PIPELINE_TIMEOUT_SEC}s (grace ${PIPELINE_KILL_GRACE_SEC}s): runtime=$runtime_label pipeline=$pipeline (see $output_path)"
  fi

  if [[ "$run_exit" -ne 0 ]]; then
    status="$(sed -n 's/^status: //p' "$output_path" | tail -n1 | tr -d '\r')"
    run_id="$(sed -n 's/^run_id: //p' "$output_path" | tail -n1 | tr -d '\r')"
    reconcile_active_runs_in_history "$workspace_path" "infra_incomplete_cycle" "pipeline command exited with code ${run_exit}"
    append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "${status:-failed}" "$workspace_path" "$output_path"
    echo "pipeline failed: runtime=$runtime_label pipeline=$pipeline (see $output_path)" >&2
    tail -n 120 "$output_path" >&2 || true
    die "pipeline command failed for runtime=$runtime_label pipeline=$pipeline"
  fi
  status="$(sed -n 's/^status: //p' "$output_path" | tail -n1 | tr -d '\r')"
  run_id="$(sed -n 's/^run_id: //p' "$output_path" | tail -n1 | tr -d '\r')"
  if [[ -z "$run_id" ]]; then
    die "missing run_id in CLI output for runtime=$runtime_label pipeline=$pipeline"
  fi
  if [[ "$status" != "succeeded" ]]; then
    append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "${status:-failed}" "$workspace_path" "$output_path"
    die "unexpected run status for $run_id: $status"
  fi

  quality_path="$workspace_path/reports/taskruns/${run_id}-quality.json"
  if [[ ! -f "$quality_path" ]]; then
    if [[ "$pipeline" == "refresh" ]] && is_successful_noop_refresh "$workspace_path" "$run_id"; then
      log "refresh run $run_id completed as validated no-op without run telemetry summary"
      append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "$status" "$workspace_path" "$output_path"
      LAST_SIGNAL="${previous_signal:-0}"
      return 0
    fi
    append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "failed" "$workspace_path" "$output_path"
    die "missing run telemetry summary for run $run_id at $quality_path"
  fi

  local metrics
  if ! metrics="$(quality_metrics "$quality_path")"; then
    append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "failed" "$workspace_path" "$output_path"
    die "invalid run telemetry summary for run $run_id at $quality_path"
  fi

  local quality_status signal_score semantic_entities semantic_edges findings questions coverage_observed coverage_missing warnings artifact_quality_count
  local domain_collect_steps mock_flag zero_signal runtime_versions
  IFS=$'\t' read -r quality_status signal_score semantic_entities semantic_edges findings questions coverage_observed coverage_missing warnings artifact_quality_count domain_collect_steps mock_flag zero_signal runtime_versions <<<"$metrics"

  if [[ "$quality_status" != "succeeded" ]]; then
    append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "$status" "$workspace_path" "$output_path"
    die "run telemetry summary status is not succeeded for run $run_id: $quality_status"
  fi

  if [[ "$runtime_mode" == "headless" ]]; then
    if [[ "$mock_flag" == "1" ]]; then
      append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "$status" "$workspace_path" "$output_path"
      die "headless run $run_id uses mock/fake runtime version ($runtime_versions)"
    fi
    if [[ "$zero_signal" == "1" ]]; then
      log "headless run $run_id produced zero-signal telemetry; leaving for SWE artifact assessment"
    fi
    if [[ "$artifact_quality_count" != "0" ]]; then
      log "headless run $run_id produced artifact_quality telemetry findings: count=$artifact_quality_count"
    fi
    if [[ "$TARGET_PROFILE" == "ai-advent" && "$domain_collect_steps" -le 0 ]]; then
      log "headless run $run_id has no domain collect telemetry signal; leaving for SWE artifact assessment"
    fi
  fi

  if [[ "$runtime_mode" == "headless" && "$pipeline" == "refresh" ]]; then
    if [[ -n "$previous_signal" ]] && (( signal_score < previous_signal )); then
      log "headless refresh telemetry signal decreased: previous=$previous_signal current=$signal_score"
    fi
    check_headless_refresh_semantic_quality "$workspace_path" "$run_id" || {
      log "headless refresh semantic telemetry checks found findings for run $run_id"
    }
    if [[ "$TARGET_PROFILE" == "ai-advent" ]]; then
      check_ai_advent_text_signal "$workspace_path" "$run_id" || {
        log "ai-advent textual telemetry check found findings for run $run_id"
      }
    fi
  fi

  append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "$status" "$workspace_path" "$output_path"

  LAST_SIGNAL="$signal_score"
  return 0
}

write_summary() {
  local exit_code="$1"
  refresh_runtime_cycle_metrics
  local result
  result="passed"
  local completion_ok=1
  if (( COMPLETED_RUNS != EXPECTED_RUNS )); then
    completion_ok=0
  fi
  if (( COMPLETED_HEADLESS_RUNS != EXPECTED_HEADLESS_RUNS )); then
    completion_ok=0
  fi
  if (( RUNNING_RUNS_DETECTED > 0 )); then
    completion_ok=0
  fi
  if [[ "$exit_code" -ne 0 || "$completion_ok" -ne 1 || -n "$TERMINATION_SIGNAL" ]]; then
    result="failed"
  fi
  SUMMARY_RESULT="$result"

  if [[ "$result" == "failed" && -z "$FAILURE_REASON" ]]; then
    if [[ -n "$TERMINATION_SIGNAL" ]]; then
      FAILURE_REASON="infra_signal_terminated"
    elif [[ "$completion_ok" -ne 1 ]]; then
      FAILURE_REASON="infra_incomplete_cycle"
    else
      FAILURE_REASON="infra_unknown_failure"
    fi
  fi

  {
    echo "# ProvenArch Full Run Session Summary"
    echo
    echo "- generated_at: $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
    echo "- result: $result"
    echo "- provenarch_root: $PROVENARCH_ROOT"
    echo "- target_input_mode: repos-file"
    echo "- target_repos_file: ${RESOLVED_TARGET_REPOS_FILE:-unset}"
    if [[ -n "$ORIGINAL_RESOLVED_TARGET_REPOS_FILE" ]]; then
      echo "- original_target_repos_file: $ORIGINAL_RESOLVED_TARGET_REPOS_FILE"
    fi
    echo "- profile_id: ${PROFILE_ID:-adhoc}"
    echo "- profile_source_kind: $PROFILE_SOURCE_KIND_EFFECTIVE"
    echo "- expected_repo_count: $EXPECTED_REPO_COUNT_RESOLVED"
    echo "- target_profile: $TARGET_PROFILE"
    echo "- workspace_baseline: $WORKSPACE_BASELINE"
    echo "- workspace_headless: $WORKSPACE_HEADLESS"
    echo "- workspace: $WORKSPACE_HEADLESS"
    echo "- tmp_root: $TMP_ROOT"
    echo "- full_run_log: $FULL_RUN_LOG"
    echo "- iterations: $ITERATIONS"
    echo "- run_logs_ttl_hours: $RUN_LOGS_TTL_HOURS"
    echo "- run_logs_max_runs: $RUN_LOGS_MAX_RUNS"
    echo "- runtime_step_timeout_sec: $RUNTIME_STEP_TIMEOUT_SEC"
    echo "- runtime_heartbeat_sec: $RUNTIME_HEARTBEAT_SEC"
    echo "- pipeline_timeout_sec: $PIPELINE_TIMEOUT_SEC"
    echo "- pipeline_kill_grace_sec: $PIPELINE_KILL_GRACE_SEC"
    echo "- api_ready_timeout_sec: $API_READY_TIMEOUT_SEC"
    echo "- api_init_timeout_sec: $API_INIT_TIMEOUT_SEC"
    if [[ -n "$API_INIT_PROGRESS_GRACE_SEC" ]]; then
      echo "- api_init_progress_grace_sec: $API_INIT_PROGRESS_GRACE_SEC"
    fi
    if [[ -n "$API_INIT_LAST_PROGRESS_AT" ]]; then
      echo "- api_init_last_progress_at: $API_INIT_LAST_PROGRESS_AT"
    fi
    echo "- ui_init_poll_timeout_sec: $UI_INIT_POLL_TIMEOUT_SEC"
    echo "- ui_cancel_poll_timeout_sec: $UI_CANCEL_POLL_TIMEOUT_SEC"
    echo "- apply_timeouts_via_api: $APPLY_TIMEOUTS_VIA_API"
    echo "- timeouts_api_apply_baseline_status: $TIMEOUTS_API_APPLY_BASELINE_STATUS"
    if [[ -n "$TIMEOUTS_API_APPLY_BASELINE_EFFECTIVE" ]]; then
      echo "- timeouts_api_apply_baseline_effective: $TIMEOUTS_API_APPLY_BASELINE_EFFECTIVE"
    fi
    if [[ -n "$TIMEOUTS_API_APPLY_BASELINE_SOURCE" ]]; then
      echo "- timeouts_api_apply_baseline_source: $TIMEOUTS_API_APPLY_BASELINE_SOURCE"
    fi
    if [[ -n "$TIMEOUTS_API_APPLY_BASELINE_JSON" ]]; then
      echo "- timeouts_api_apply_baseline_json: $TIMEOUTS_API_APPLY_BASELINE_JSON"
    fi
    echo "- timeouts_api_apply_headless_status: $TIMEOUTS_API_APPLY_HEADLESS_STATUS"
    if [[ -n "$TIMEOUTS_API_APPLY_HEADLESS_EFFECTIVE" ]]; then
      echo "- timeouts_api_apply_headless_effective: $TIMEOUTS_API_APPLY_HEADLESS_EFFECTIVE"
    fi
    if [[ -n "$TIMEOUTS_API_APPLY_HEADLESS_SOURCE" ]]; then
      echo "- timeouts_api_apply_headless_source: $TIMEOUTS_API_APPLY_HEADLESS_SOURCE"
    fi
    if [[ -n "$TIMEOUTS_API_APPLY_HEADLESS_JSON" ]]; then
      echo "- timeouts_api_apply_headless_json: $TIMEOUTS_API_APPLY_HEADLESS_JSON"
    fi
    echo "- keep_tmp: $KEEP_TMP"
    echo "- expected_runs: $EXPECTED_RUNS"
    echo "- completed_runs: $COMPLETED_RUNS"
    echo "- expected_headless_runs: $EXPECTED_HEADLESS_RUNS"
    echo "- completed_headless_runs: $COMPLETED_HEADLESS_RUNS"
    echo "- malformed_run_results_rows: $MALFORMED_RUN_RESULTS_ROWS"
    echo "- running_runs_detected: $RUNNING_RUNS_DETECTED"
    echo "- last_pipeline_stage: $LAST_PIPELINE_STAGE"
    echo "- last_runtime_provider: $LAST_RUNTIME_PROVIDER"
    if [[ -n "$LAST_PIPELINE_DEADLINE_AT" ]]; then
      echo "- pipeline_deadline_at: $LAST_PIPELINE_DEADLINE_AT"
    fi
    if [[ -n "$LAST_PIPELINE_TIMEOUT_ELAPSED_SEC" ]]; then
      echo "- pipeline_timeout_elapsed_sec: $LAST_PIPELINE_TIMEOUT_ELAPSED_SEC"
    fi
    if [[ -n "$LAST_PIPELINE_DEADLINE_MISSED_BY_SEC" ]]; then
      echo "- deadline_missed_by_sec: $LAST_PIPELINE_DEADLINE_MISSED_BY_SEC"
    fi
    if [[ -n "$LAST_PIPELINE_LAST_PROGRESS_AT" ]]; then
      echo "- last_progress_at: $LAST_PIPELINE_LAST_PROGRESS_AT"
    fi
    if [[ -n "$LAST_WATCHDOG_TICK_AT" ]]; then
      echo "- last_watchdog_tick_at: $LAST_WATCHDOG_TICK_AT"
    fi
    if [[ -n "$LAST_WATCHDOG_TICK_GAP_SEC" ]]; then
      echo "- max_watchdog_tick_gap_sec: $LAST_WATCHDOG_TICK_GAP_SEC"
    fi
    echo "- host_clock_gap_detected: $HOST_CLOCK_GAP_DETECTED"
    if [[ -n "$TERMINATION_SIGNAL" ]]; then
      echo "- termination_signal: $TERMINATION_SIGNAL"
    else
      echo "- termination_signal: none"
    fi
    echo "- headless_provider: ${HEADLESS_PROVIDER:-unset}"
    echo "- headless_command: ${HEADLESS_CMD:-unset}"
    echo

    echo "## API Simulation"
    echo "- status: $API_SIM_STATUS"
    if [[ -n "$API_INIT_RUN_ID" ]]; then
      echo "- init_run_id: $API_INIT_RUN_ID"
    fi
    if [[ -n "$API_INIT_FINAL_STATUS" ]]; then
      echo "- init_final_status: $API_INIT_FINAL_STATUS"
    fi
    echo

    echo "## Runtime Cycle Results"
    if [[ ! -s "$RUN_RESULTS_TSV" ]]; then
      echo "- no completed runs recorded"
    else
      echo "| iteration | runtime_mode | runtime_provider | pipeline | run_id | status | signal | entities | edges | findings | questions | cov_obs | cov_missing | warnings |"
      echo "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|"
      while IFS=$'\t' read -r iter runtime_mode runtime_provider pipeline run_id status signal entities edges findings questions cov_obs cov_missing warnings _runtime_versions _quality_path _run_log _extra; do
        if [[ -n "$_extra" || -z "$run_id" ]]; then
          continue
        fi
        echo "| $iter | $runtime_mode | $runtime_provider | $pipeline | $run_id | $status | $signal | $entities | $edges | $findings | $questions | $cov_obs | $cov_missing | $warnings |"
      done < "$RUN_RESULTS_TSV"
    fi
    echo

    echo "## Workspace Diagnostics"
    if [[ -f "$VALIDATE_JSON" ]]; then
      python3 - "$VALIDATE_JSON" <<'PY'
import json
import sys

payload = json.load(open(sys.argv[1], encoding='utf-8'))
warnings = payload.get('warnings') or []
errors = payload.get('errors') or []
resolved = payload.get('resolved_repos') or []
print(f"- resolved_repos: {len(resolved)}")
if not warnings:
    print("- warnings: none")
else:
    print("- warnings:")
    for item in warnings:
        print(f"  - [{item.get('code', 'unknown')}] {item.get('message', '')}")
if not errors:
    print("- errors: none")
else:
    print("- errors:")
    for item in errors:
        print(f"  - [{item.get('code', 'unknown')}] {item.get('message', '')}")
PY
    else
      echo "- validate payload is unavailable"
    fi
    echo

    echo "## Key Artifacts"
    echo "- $WORKSPACE_HEADLESS/reports/as-is/overview.md"
    echo "- $WORKSPACE_HEADLESS/reports/findings/findings.md"
    echo "- $WORKSPACE_HEADLESS/reports/coverage/summary.md"
    echo "- $WORKSPACE_HEADLESS/reports/coverage/open-questions.md"
    echo "- $WORKSPACE_HEADLESS/reports/taskruns/run-history.json"
    echo "- $WORKSPACE_HEADLESS/reports/taskruns/logs/"
    echo "- $SNAPSHOT_DIR"
    echo "- execution_gate: live runtime/frontend evidence only"
    echo

    if [[ "$result" == "failed" ]]; then
      echo "## Failure Reason + Next Actions"
      if [[ -n "$FAILURE_REASON" ]]; then
        echo "- failure_reason: $FAILURE_REASON"
      else
        echo "- failure_reason: unknown (see full_run_log)"
      fi
      echo "- next_actions:"
      echo "  1) Inspect $FULL_RUN_LOG and per-run logs in $LOG_DIR"
      echo "  2) Compare snapshots in $SNAPSHOT_DIR to detect signal regression"
      echo "  3) Fix issues and rerun script from scratch (new tmp workspace)"
    fi
  } > "$SUMMARY_PATH"
  SUMMARY_WRITTEN="yes"
}

cleanup() {
  local exit_code="$1"
  stop_server
  SUMMARY_WRITTEN="no"
  if ! write_summary "$exit_code"; then
    SUMMARY_WRITTEN="no"
  fi

  local terminal_state="completed"
  local terminal_signal="none"
  local terminal_exit="$exit_code"
  if [[ "$TERMINATION_SIGNAL" == "timeout" ]]; then
    terminal_state="process_failed"
    terminal_signal="timeout"
    terminal_exit="${TERMINAL_PROCESS_EXIT_OVERRIDE:-124}"
  elif [[ -n "$TERMINATION_SIGNAL" ]]; then
    terminal_state="signal_terminated"
    terminal_signal="$(signal_status_token "$TERMINATION_SIGNAL")"
    terminal_exit="$(signal_exit_code "$TERMINATION_SIGNAL")"
  elif [[ "$exit_code" -ne 0 || "$SUMMARY_RESULT" != "passed" ]]; then
    terminal_state="process_failed"
  fi
  write_terminal_run_status "$terminal_state" "$terminal_exit" "$terminal_signal" "${FAILURE_REASON:-none}" "$SUMMARY_WRITTEN"

  if [[ "$exit_code" -ne 0 || "$SUMMARY_RESULT" != "passed" ]]; then
    log "run failed; keeping artifacts for debugging at $TMP_ROOT"
    log "summary: $SUMMARY_PATH"
    return
  fi

  if [[ "$KEEP_TMP" == "1" || "$CREATED_TMP" -eq 0 ]]; then
    log "artifacts kept at $TMP_ROOT"
    log "summary: $SUMMARY_PATH"
    return
  fi

  cat "$SUMMARY_PATH" >&4
  restore_isolated_target_repos_write_bits
  rm -rf "$TMP_ROOT"
  log "temporary artifacts removed (set KEEP_TMP=1 to keep)"
}
trap 'on_termination_signal TERM' TERM
trap 'on_termination_signal INT' INT
trap 'on_termination_signal HUP' HUP
trap 'on_termination_signal PIPE' PIPE
trap 'cleanup $?' EXIT

require_cmd git "Install git and ensure it is available in PATH."
"$PROVENARCH_ROOT/scripts/run-go.sh" version >/dev/null || die "Install Go version from .go-version or set ACP_GO_BIN=/path/to/go."
require_cmd npm "Install Node.js/npm and ensure it is available in PATH."
require_cmd make "Install make and ensure it is available in PATH."
require_cmd curl "Install curl and ensure it is available in PATH."
require_cmd python3 "Install python3 and ensure it is available in PATH."

if [[ ! -d "$PROVENARCH_ROOT" ]]; then
  die "PROVENARCH_ROOT does not exist: $PROVENARCH_ROOT"
fi

prepare_target_repos_file
validate_target_repos_file
read_target_repos_meta
prepare_isolated_target_repos_file
case "$PROFILE_SOURCE_KIND_EFFECTIVE" in
  path|git_url|mixed)
    ;;
  *)
    die "invalid target repos metadata profile_source_kind '$PROFILE_SOURCE_KIND_EFFECTIVE' (expected path|git_url|mixed)"
    ;;
esac
if [[ -n "$ORIGINAL_RESOLVED_TARGET_REPOS_FILE" ]]; then
  log "target input resolved: mode=repos-file repos_file=$RESOLVED_TARGET_REPOS_FILE original_repos_file=$ORIGINAL_RESOLVED_TARGET_REPOS_FILE profile=$TARGET_PROFILE"
else
  log "target input resolved: mode=repos-file repos_file=$RESOLVED_TARGET_REPOS_FILE profile=$TARGET_PROFILE"
fi

HEADLESS_PROVIDER="${ACP_RUNTIME_PROVIDER:-claude-code}"
case "$HEADLESS_PROVIDER" in
  claude-code)
    HEADLESS_CMD="${ACP_CLAUDE_CMD:-claude-code}"
    export ACP_CLAUDE_CMD="$HEADLESS_CMD"
    ;;
  qwen-code)
    HEADLESS_CMD="${ACP_QWEN_CMD:-qwen}"
    export ACP_QWEN_CMD="$HEADLESS_CMD"
    ;;
  codex-code)
    HEADLESS_CMD="${ACP_CODEX_CMD:-codex}"
    export ACP_CODEX_CMD="$HEADLESS_CMD"
    ;;
  *)
    die "unsupported ACP_RUNTIME_PROVIDER '$HEADLESS_PROVIDER' (allowed: claude-code, qwen-code, codex-code)"
    ;;
esac
if ! command -v "$HEADLESS_CMD" >/dev/null 2>&1; then
  die "headless runtime command '$HEADLESS_CMD' is unavailable for provider '$HEADLESS_PROVIDER'. Install command or set ACP_CLAUDE_CMD/ACP_QWEN_CMD/ACP_CODEX_CMD"
fi
export ACP_RUNTIME_PROVIDER="$HEADLESS_PROVIDER"

log "build ProvenArch binary"
(
  cd "$PROVENARCH_ROOT"
  make build >"$LOG_DIR/make-build.log" 2>&1
)
ACP_BIN="$PROVENARCH_ROOT/bin/acp"
if [[ ! -x "$ACP_BIN" ]]; then
  die "acp binary was not built at $ACP_BIN (see $LOG_DIR/make-build.log)"
fi

log "bootstrap baseline workspace in tmp"
"$ACP_BIN" init-workspace \
  --workspace "$WORKSPACE_BASELINE" \
  --repos-file "$RESOLVED_TARGET_REPOS_FILE" >"$LOG_DIR/init-workspace.log" 2>&1

if [[ ! -f "$WORKSPACE_BASELINE/workspace.yaml" ]]; then
  die "workspace bootstrap failed: missing $WORKSPACE_BASELINE/workspace.yaml"
fi
if [[ ! -d "$WORKSPACE_BASELINE/.git" ]]; then
  die "workspace bootstrap failed: missing $WORKSPACE_BASELINE/.git"
fi
if [[ ! -f "$WORKSPACE_BASELINE/skills/subagents.yaml" ]]; then
  die "workspace bootstrap failed: missing baseline bundle artifact skills/subagents.yaml"
fi

resolve_effective_timeouts_from_workspace "$WORKSPACE_BASELINE"
log "resolved timeout config: step=${RUNTIME_STEP_TIMEOUT_SEC}s heartbeat=${RUNTIME_HEARTBEAT_SEC}s pipeline=${PIPELINE_TIMEOUT_SEC}s kill_grace=${PIPELINE_KILL_GRACE_SEC}s api_ready=${API_READY_TIMEOUT_SEC}s api_init=${API_INIT_TIMEOUT_SEC}s ui_init=${UI_INIT_POLL_TIMEOUT_SEC}s ui_cancel=${UI_CANCEL_POLL_TIMEOUT_SEC}s"

API_PORT="$(allocate_free_port)"
API_BASE="http://127.0.0.1:${API_PORT}"
SERVER_LOG="$LOG_DIR/serve-fake.log"

log "start API server for validate/init simulation"
LAST_PIPELINE_STAGE="api_server.bootstrap"
LAST_RUNTIME_PROVIDER="fake"
write_running_run_status_heartbeat
"$ACP_BIN" serve \
  --workspace "$WORKSPACE_BASELINE" \
  --runtime fake \
  --listen "127.0.0.1:${API_PORT}" \
  --run-logs-ttl-hours "$RUN_LOGS_TTL_HOURS" \
  --run-logs-max-runs "$RUN_LOGS_MAX_RUNS" >"$SERVER_LOG" 2>&1 &
SERVER_PID="$!"

if ! wait_for_health "$API_BASE"; then
  die "ACP API did not become healthy in ${API_READY_TIMEOUT_SEC}s (see $SERVER_LOG)"
fi

if [[ "$APPLY_TIMEOUTS_VIA_API" == "1" ]]; then
  log "apply runtime timeouts via API for workspace=baseline"
  apply_runtime_timeouts_via_api "baseline" "$WORKSPACE_BASELINE" "$API_BASE"
else
  log "runtime timeouts API apply disabled for workspace=baseline (ACP_APPLY_TIMEOUTS_VIA_API=0)"
  TIMEOUTS_API_APPLY_BASELINE_STATUS="skipped"
fi

log "POST /api/workspace/validate"
LAST_PIPELINE_STAGE="api.workspace.validate"
write_running_run_status_heartbeat
curl -fsS -X POST "$API_BASE/api/workspace/validate" > "$VALIDATE_JSON"
python3 - "$VALIDATE_JSON" "$EXPECTED_REPO_COUNT_RESOLVED" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding='utf-8'))
if not payload.get('ok'):
    raise SystemExit('workspace validate returned ok=false')
resolved = payload.get('resolved_repos') or []
if not resolved:
    raise SystemExit('workspace validate returned empty resolved_repos')
expected_count = int(sys.argv[2])
if len(resolved) != expected_count:
    raise SystemExit(f"workspace validate resolved_repos count mismatch: expected={expected_count} got={len(resolved)}")
print(f"resolved_repos={len(resolved)}")
PY

log "POST /api/pipeline/init"
LAST_PIPELINE_STAGE="api.pipeline.init"
write_running_run_status_heartbeat
curl -fsS -X POST -H 'Content-Type: application/json' -d '{"trigger":"manual"}' "$API_BASE/api/pipeline/init" > "$API_INIT_START_JSON"
API_INIT_RUN_ID="$(python3 - "$API_INIT_START_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding='utf-8'))
run_id = payload.get('run_id', '')
if not run_id:
    raise SystemExit('missing run_id in /api/pipeline/init response')
print(run_id)
PY
)"

init_status=""
init_current_step=""
init_artifact_count="0"
init_warning_count="0"
api_init_started_at=$SECONDS
api_init_deadline=$((SECONDS + API_INIT_TIMEOUT_SEC))
API_INIT_PROGRESS_GRACE_SEC=$((RUNTIME_HEARTBEAT_SEC * 2))
if (( API_INIT_PROGRESS_GRACE_SEC < 60 )); then
  API_INIT_PROGRESS_GRACE_SEC=60
fi
api_init_hard_deadline=$((api_init_deadline + API_INIT_PROGRESS_GRACE_SEC))
last_api_init_progress=0
api_init_last_signature=""
api_init_last_progress_second=$SECONDS
API_INIT_LAST_PROGRESS_AT="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
while :; do
  if (( SECONDS >= api_init_deadline )); then
    api_init_last_progress_age=$((SECONDS - api_init_last_progress_second))
    if [[ "$init_status" != "running" && "$init_status" != "queued" && -n "$init_status" ]]; then
      break
    fi
    if (( SECONDS >= api_init_hard_deadline || api_init_last_progress_age > API_INIT_PROGRESS_GRACE_SEC )); then
      break
    fi
  fi
  curl -fsS "$API_BASE/api/pipeline/runs/$API_INIT_RUN_ID" > "$API_INIT_STATUS_JSON"
  init_fields="$(python3 - "$API_INIT_STATUS_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding='utf-8'))
status = str(payload.get('status') or '')
step = str(payload.get('current_step') or '')
artifacts = payload.get('artifacts') or []
warnings = payload.get('warnings') or []
print("\t".join([status, step, str(len(artifacts)), str(len(warnings))]))
PY
)"
  IFS=$'\t' read -r init_status init_current_step init_artifact_count init_warning_count <<< "$init_fields"
  init_signature="${init_status}|${init_current_step}|${init_artifact_count}|${init_warning_count}"
  if [[ -n "$init_signature" && "$init_signature" != "$api_init_last_signature" ]]; then
    if [[ -n "$api_init_last_signature" ]]; then
      log "api init observed progress: run_id=$API_INIT_RUN_ID status=$init_status current_step=${init_current_step:-unknown} artifacts=$init_artifact_count warnings=$init_warning_count"
    fi
    api_init_last_signature="$init_signature"
    api_init_last_progress_second=$SECONDS
    API_INIT_LAST_PROGRESS_AT="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
  fi
  if [[ "$init_status" == "succeeded" ]]; then
    break
  fi
  if [[ "$init_status" == "failed" ]]; then
    API_INIT_FINAL_STATUS="$init_status"
    append_api_init_run_result_row "$init_status"
    die "API init run failed (see $API_INIT_STATUS_JSON and $SERVER_LOG)"
  fi
  api_init_elapsed=$((SECONDS - api_init_started_at))
  if (( RUNTIME_HEARTBEAT_SEC > 0 )) && (( api_init_elapsed > 0 )) && (( api_init_elapsed % RUNTIME_HEARTBEAT_SEC == 0 )) && (( api_init_elapsed != last_api_init_progress )); then
    log "api init progress: run_id=$API_INIT_RUN_ID status=$init_status current_step=${init_current_step:-unknown} elapsed_sec=$api_init_elapsed timeout_sec=$API_INIT_TIMEOUT_SEC progress_grace_sec=$API_INIT_PROGRESS_GRACE_SEC"
    LAST_PIPELINE_STAGE="api.pipeline.init.poll"
    write_running_run_status_heartbeat
    last_api_init_progress="$api_init_elapsed"
  fi
  sleep 0.25
done
if [[ "$init_status" != "succeeded" ]]; then
  API_INIT_FINAL_STATUS="$init_status"
  append_api_init_run_result_row "${init_status:-failed}"
  api_init_elapsed=$((SECONDS - api_init_started_at))
  api_init_last_progress_age=$((SECONDS - api_init_last_progress_second))
  die "API init run did not finish in time (run_id=$API_INIT_RUN_ID elapsed_sec=$api_init_elapsed timeout_sec=$API_INIT_TIMEOUT_SEC progress_grace_sec=$API_INIT_PROGRESS_GRACE_SEC last_progress_age_sec=$api_init_last_progress_age current_step=${init_current_step:-unknown})"
fi
API_INIT_FINAL_STATUS="$init_status"

curl -fsS "$API_BASE/api/pipeline/runs/$API_INIT_RUN_ID/artifacts" > "$API_INIT_ARTIFACTS_JSON"
python3 - "$API_INIT_ARTIFACTS_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding='utf-8'))
artifacts = payload.get('artifacts') or []
if not artifacts:
    raise SystemExit('init artifacts payload is empty')
print(f"api_artifacts={len(artifacts)}")
PY

curl -fsS "$API_BASE/api/pipeline/runs/$API_INIT_RUN_ID/logs?cursor=0&limit=200" > "$API_INIT_LOGS_JSON"
python3 - "$API_INIT_LOGS_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding='utf-8'))
items = payload.get('items') or []
if not items:
    raise SystemExit('api init logs payload is empty')
print(f"api_logs_items={len(items)}")
PY

API_SIM_STATUS="succeeded"
log "stop API server"
stop_server

log "bootstrap headless workspace in tmp"
LAST_PIPELINE_STAGE="headless.workspace.bootstrap"
LAST_RUNTIME_PROVIDER="$HEADLESS_PROVIDER"
write_running_run_status_heartbeat
"$ACP_BIN" init-workspace \
  --workspace "$WORKSPACE_HEADLESS" \
  --repos-file "$RESOLVED_TARGET_REPOS_FILE" >"$LOG_DIR/init-workspace-headless.log" 2>&1
if [[ ! -f "$WORKSPACE_HEADLESS/workspace.yaml" ]]; then
  die "headless workspace bootstrap failed: missing $WORKSPACE_HEADLESS/workspace.yaml"
fi
if [[ ! -d "$WORKSPACE_HEADLESS/.git" ]]; then
  die "headless workspace bootstrap failed: missing $WORKSPACE_HEADLESS/.git"
fi

resolve_effective_timeouts_from_workspace "$WORKSPACE_HEADLESS"
if [[ "$APPLY_TIMEOUTS_VIA_API" == "1" ]]; then
  log "start temporary API server for timeout apply workspace=headless"
  LAST_PIPELINE_STAGE="headless.timeouts.apply"
  LAST_RUNTIME_PROVIDER="$HEADLESS_PROVIDER"
  write_running_run_status_heartbeat
  HEADLESS_API_PORT="$(allocate_free_port)"
  HEADLESS_API_BASE="http://127.0.0.1:${HEADLESS_API_PORT}"
  HEADLESS_SERVER_LOG="$LOG_DIR/serve-headless-timeouts.log"
  "$ACP_BIN" serve \
    --workspace "$WORKSPACE_HEADLESS" \
    --runtime fake \
    --listen "127.0.0.1:${HEADLESS_API_PORT}" \
    --run-logs-ttl-hours "$RUN_LOGS_TTL_HOURS" \
    --run-logs-max-runs "$RUN_LOGS_MAX_RUNS" >"$HEADLESS_SERVER_LOG" 2>&1 &
  SERVER_PID="$!"
  if ! wait_for_health "$HEADLESS_API_BASE"; then
    die "headless timeout API did not become healthy in ${API_READY_TIMEOUT_SEC}s (see $HEADLESS_SERVER_LOG)"
  fi
  apply_runtime_timeouts_via_api "headless" "$WORKSPACE_HEADLESS" "$HEADLESS_API_BASE"
  stop_server
else
  log "runtime timeouts API apply disabled for workspace=headless (ACP_APPLY_TIMEOUTS_VIA_API=0)"
  TIMEOUTS_API_APPLY_HEADLESS_STATUS="skipped"
fi

log "run runtime cycles: fake + headless(provider=$HEADLESS_PROVIDER)"
LAST_PIPELINE_STAGE="runtime.cycles"
LAST_RUNTIME_PROVIDER="$HEADLESS_PROVIDER"
write_running_run_status_heartbeat
prev_fake_init_signal=""
prev_fake_refresh_signal=""
prev_headless_init_signal=""
prev_headless_refresh_signal=""
for iteration in $(seq 1 "$ITERATIONS"); do
  run_cli_pipeline "fake" "$HEADLESS_PROVIDER" "init" "$iteration" "$WORKSPACE_BASELINE" "$prev_fake_init_signal"
  prev_fake_init_signal="$LAST_SIGNAL"

  run_cli_pipeline "fake" "$HEADLESS_PROVIDER" "refresh" "$iteration" "$WORKSPACE_BASELINE" "$prev_fake_refresh_signal"
  prev_fake_refresh_signal="$LAST_SIGNAL"

  run_cli_pipeline "headless" "$HEADLESS_PROVIDER" "init" "$iteration" "$WORKSPACE_HEADLESS" "$prev_headless_init_signal"
  prev_headless_init_signal="$LAST_SIGNAL"

  run_cli_pipeline "headless" "$HEADLESS_PROVIDER" "refresh" "$iteration" "$WORKSPACE_HEADLESS" "$prev_headless_refresh_signal"
  prev_headless_refresh_signal="$LAST_SIGNAL"
done

if ! validate_runtime_cycle_completion; then
  if [[ -z "$FAILURE_REASON" || "$FAILURE_REASON" == "unknown" ]]; then
    FAILURE_REASON="infra_incomplete_cycle"
  fi
  local_line="[full-run][error] runtime cycle completion invariants failed"
  echo "$local_line"
  echo "$local_line" >&4
  exit 1
fi

for path in \
  "$WORKSPACE_HEADLESS/reports/as-is/overview.md" \
  "$WORKSPACE_HEADLESS/reports/findings/findings.md" \
  "$WORKSPACE_HEADLESS/reports/coverage/open-questions.md"; do
  if [[ ! -f "$path" ]]; then
    die "missing expected artifact after run cycle: $path"
  fi
done

log "full run completed"
