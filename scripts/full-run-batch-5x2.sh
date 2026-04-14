#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
TARGET_REPOS_FILE="${TARGET_REPOS_FILE:-}"
PROFILE_ID="${PROFILE_ID:-}"
PROFILE_SOURCE_KIND="${PROFILE_SOURCE_KIND:-}"
EXPECTED_REPO_COUNT="${EXPECTED_REPO_COUNT:-}"
SWEEP_ID="${SWEEP_ID:-baseline}"
BATCH_ID="${BATCH_ID:-batch-$(date -u +'%Y%m%dT%H%M%SZ')}"
RUN_COUNT="${RUN_COUNT:-5}"
ACP_CLAUDE_CMD_BIN="${ACP_CLAUDE_CMD_BIN:-claude}"
ACP_QWEN_CMD_BIN="${ACP_QWEN_CMD_BIN:-qwen}"
ACP_APPLY_TIMEOUTS_VIA_API="${ACP_APPLY_TIMEOUTS_VIA_API:-1}"
ACP_EXECUTION_STRATEGY="${ACP_EXECUTION_STRATEGY:-}"
ACP_MAX_PARALLEL_TASKS="${ACP_MAX_PARALLEL_TASKS:-}"
ACP_FAILURE_POLICY="${ACP_FAILURE_POLICY:-}"
ACP_SHARD_DISCOVERY_MODE="${ACP_SHARD_DISCOVERY_MODE:-}"
ACP_REPO_SELECTION="${ACP_REPO_SELECTION:-}"
BATCH_PROVIDER_FILTER="${BATCH_PROVIDER_FILTER:-all}"
BATCH_RUN_SELECTION="${BATCH_RUN_SELECTION:-all}"
BATCH_SKIP_PRECHECK="${BATCH_SKIP_PRECHECK:-0}"
BATCH_FRONTEND_MODE="${BATCH_FRONTEND_MODE:-auto}"
E2E_TMP_ROOT="${E2E_TMP_ROOT:-/tmp/provenarch-test_arch_project}"
BATCH_ROOT="${BATCH_ROOT:-$E2E_TMP_ROOT/runs/$BATCH_ID}"
REPORTS_ROOT="${REPORTS_ROOT:-$E2E_TMP_ROOT/reports}"
RESOLVED_TARGET_REPOS_FILE=""
DECLARED_REPOS_JSON=""
RUN_CLASSIFICATIONS_TSV=""
declare -a ALL_PROVIDERS=("qwen-code" "claude-code")
declare -a SELECTED_PROVIDERS=()
declare -a SELECTED_RUN_INDEXES=()
SELECTED_PROVIDERS_CSV=""
SELECTED_RUN_INDEXES_CSV=""
RUNTIME_PARSE_FAILURES=0
RUNNER_UNAVAILABLE_FAILURES=0
INFRA_SIGNAL_TERMINATED_FAILURES=0
INFRA_INCOMPLETE_CYCLE_FAILURES=0
RUNTIME_TIMEOUT_FAILURES=0
QUALITY_GATES_FAILED_FAILURES=0
SUMMARY_MISSING_FAILURES=0
PRECHECK_FAILED_FAILURES=0
CANCELLATION_LIKE_FAILURES=0
OTHER_FAILURES=0
LAST_RUN_FAILURE_CLASS="none"
LAST_RUN_FAILURE_SUBCLASS="none"
LAST_RUN_CANCELLATION_LIKE=0
PRECHECK_FAILURE_RECORDED=0
FRONTEND_CANCEL_FAILURES=0
FRONTEND_CANCEL_SKIPPED=0
TIMEOUT_PRECHECK_UNSET_KEYS=(
  ACP_RUNTIME_STEP_TIMEOUT_SEC
  ACP_RUNTIME_HEARTBEAT_SEC
  ACP_PIPELINE_TIMEOUT_SEC
  ACP_PIPELINE_KILL_GRACE_SEC
  ACP_API_READY_TIMEOUT_SEC
  ACP_API_INIT_TIMEOUT_SEC
  ACP_UI_INIT_POLL_TIMEOUT_SEC
  ACP_UI_CANCEL_POLL_TIMEOUT_SEC
  ACP_FULL_RUN_PIPELINE_TIMEOUT_SEC
  ACP_FULL_RUN_PIPELINE_KILL_GRACE_SEC
  READY_TIMEOUT_SEC
  UI_E2E_INIT_TIMEOUT_SEC
  UI_E2E_CANCEL_TIMEOUT_SEC
  ACP_EXECUTION_STRATEGY
  ACP_MAX_PARALLEL_TASKS
  ACP_FAILURE_POLICY
  ACP_SHARD_DISCOVERY_MODE
  ACP_REPO_SELECTION
  SWEEP_ID
)

log() {
  printf '[batch-5x2] %s\n' "$*" >&2
}

die() {
  echo "[batch-5x2][error] $*" >&2
  exit 1
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    die "required command is unavailable: $cmd"
  fi
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
        qwen-code|claude-code)
          if ! array_contains "$token" "${SELECTED_PROVIDERS[@]-}"; then
            SELECTED_PROVIDERS+=("$token")
          fi
          ;;
        *)
          die "BATCH_PROVIDER_FILTER contains unsupported provider '$token' (allowed: qwen-code, claude-code, all)"
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

should_run_frontend_for_selection() {
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
    *)
      die "BATCH_FRONTEND_MODE must be auto|always|never (got '$BATCH_FRONTEND_MODE')"
      ;;
  esac
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
  python3 - "$path" "$provider" "$scenario" "$status" "$reason" "$workspace" "$output_dir" "$runtime_command" "$server_log" "$playwright_log" <<'PY'
import json
import sys
from datetime import datetime, timezone

path, provider, scenario, status, reason, workspace, output_dir, runtime_command, server_log, playwright_log = sys.argv[1:]
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
with open(path, "w", encoding="utf-8") as f:
    json.dump(payload, f, ensure_ascii=True, indent=2)
    f.write("\n")
PY
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
  local -a classify_log_paths=("$summary_path" "$full_log_path" "$batch_driver_log")
  local iter_log
  if [[ -d "$run_dir/logs" ]]; then
    while IFS= read -r iter_log; do
      classify_log_paths+=("$iter_log")
    done < <(find "$run_dir/logs" -maxdepth 1 -type f -name 'run-iter*-*.log' | LC_ALL=C sort)
  fi

  if [[ ! -f "$summary_path" ]]; then
    run_class="summary_missing"
    summary_result="missing"
    failure_reason="summary_missing"
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

  if [[ -f "$run_results_path" ]]; then
    run_count="$(awk 'NF { count++ } END { print count+0 }' "$run_results_path")"
  fi

  if [[ "$run_class" == "none" ]] && contains_in_files "runner_unavailable" "${classify_log_paths[@]}"; then
    run_class="runner_unavailable"
  fi
  if [[ "$run_class" == "none" ]] && contains_in_files "runner_parse_failed" "${classify_log_paths[@]}"; then
    run_class="runtime_parse"
  fi

  if [[ "$run_class" == "none" ]]; then
    if [[ "$failure_reason" == "runtime_timeout" || "$termination_signal" == "timeout" ]]; then
      run_class="runtime_timeout"
    fi
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

  if [[ "$process_exit" -ne 0 && "$run_class" == "none" ]]; then
    run_class="infra_incomplete_cycle"
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
    runtime_parse)
      RUNTIME_PARSE_FAILURES=$((RUNTIME_PARSE_FAILURES + 1))
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
  log "backend failure classes: precheck_failed=$PRECHECK_FAILED_FAILURES runtime_parse=$RUNTIME_PARSE_FAILURES runner_unavailable=$RUNNER_UNAVAILABLE_FAILURES runtime_timeout=$RUNTIME_TIMEOUT_FAILURES infra_signal_terminated=$INFRA_SIGNAL_TERMINATED_FAILURES infra_incomplete_cycle=$INFRA_INCOMPLETE_CYCLE_FAILURES quality_gates_failed=$QUALITY_GATES_FAILED_FAILURES summary_missing=$SUMMARY_MISSING_FAILURES cancellation_like=$CANCELLATION_LIKE_FAILURES other=$OTHER_FAILURES"
  die "batch precheck failed: reason=$reason precheck_failed=$PRECHECK_FAILED_FAILURES runtime_parse=$RUNTIME_PARSE_FAILURES runner_unavailable=$RUNNER_UNAVAILABLE_FAILURES runtime_timeout=$RUNTIME_TIMEOUT_FAILURES infra_signal_terminated=$INFRA_SIGNAL_TERMINATED_FAILURES infra_incomplete_cycle=$INFRA_INCOMPLETE_CYCLE_FAILURES quality_gates_failed=$QUALITY_GATES_FAILED_FAILURES summary_missing=$SUMMARY_MISSING_FAILURES cancellation_like=$CANCELLATION_LIKE_FAILURES other=$OTHER_FAILURES"
}

prepare_target_repos_file() {
  if [[ -n "${TARGET_REPO:-}" || -n "${TARGET_REPO_GIT_URL:-}" || -n "${TARGET_REPO_NAME:-}" || -n "${TARGET_REPO_REF:-}" ]]; then
    die "legacy target input env (TARGET_REPO*) is no longer supported; use TARGET_REPOS_FILE with repos[]"
  fi
  if [[ -z "$TARGET_REPOS_FILE" ]]; then
    die "missing target input: set TARGET_REPOS_FILE (repos[] YAML)"
  fi
  if [[ ! -f "$TARGET_REPOS_FILE" ]]; then
    die "TARGET_REPOS_FILE does not exist: $TARGET_REPOS_FILE"
  fi
  RESOLVED_TARGET_REPOS_FILE="$(cd "$(dirname "$TARGET_REPOS_FILE")" && pwd)/$(basename "$TARGET_REPOS_FILE")"
}

collect_declared_repos() {
  DECLARED_REPOS_JSON="$BATCH_ROOT/declared-repos.json"
  python3 - "$RESOLVED_TARGET_REPOS_FILE" "$EXPECTED_REPO_COUNT" "$PROFILE_SOURCE_KIND" "$PROFILE_ID" "$DECLARED_REPOS_JSON" <<'PY'
import json
import sys
from pathlib import Path

try:
    import yaml  # type: ignore
except Exception as exc:
    raise SystemExit(f"PyYAML is required for parsing repos file: {exc}")

repos_file = Path(sys.argv[1]).resolve()
expected_raw = (sys.argv[2] or "").strip()
source_kind = (sys.argv[3] or "").strip()
profile_id = (sys.argv[4] or "").strip()
out_path = Path(sys.argv[5]).resolve()

if source_kind not in {"", "path", "git_url"}:
    raise SystemExit(f"PROFILE_SOURCE_KIND must be one of path|git_url, got: {source_kind}")

payload = yaml.safe_load(repos_file.read_text(encoding="utf-8"))
if isinstance(payload, list):
    repos = payload
elif isinstance(payload, dict):
    repos = payload.get("repos")
else:
    repos = None
if not isinstance(repos, list) or not repos:
    raise SystemExit(f"repos file {repos_file} must contain non-empty repos[]")

declared = []
for idx, item in enumerate(repos, start=1):
    if not isinstance(item, dict):
        raise SystemExit(f"repos[{idx}] must be an object")
    name = str(item.get("name", "")).strip()
    if not name:
        raise SystemExit(f"repos[{idx}] is missing name")
    path_raw = str(item.get("path", "")).strip()
    git_url = str(item.get("git_url", "")).strip()
    has_path = bool(path_raw)
    has_git = bool(git_url)
    if has_path == has_git:
        raise SystemExit(f"repos[{idx}] must set exactly one of path or git_url")
    ref = str(item.get("ref", "")).strip()
    if has_path:
        path_value = Path(path_raw)
        abs_path = (repos_file.parent / path_value).resolve() if not path_value.is_absolute() else path_value.resolve()
        if not abs_path.exists():
            raise SystemExit(f"repos[{idx}] path does not exist: {abs_path}")
        if not abs_path.is_dir():
            raise SystemExit(f"repos[{idx}] path is not a directory: {abs_path}")
        source = "path"
        entry = {"name": name, "source": source, "path": str(abs_path), "ref": ref}
    else:
        source = "git_url"
        entry = {"name": name, "source": source, "git_url": git_url, "ref": ref}
    if source_kind == "path" and source != "path":
        raise SystemExit(f"profile source_kind=path but repos[{idx}] uses git_url")
    if source_kind == "git_url":
        if source != "git_url":
            raise SystemExit(f"profile source_kind=git_url but repos[{idx}] uses path")
        if not ref:
            raise SystemExit(f"repos[{idx}] git_url entry must have pinned ref for source_kind=git_url")
    declared.append(entry)

if expected_raw:
    try:
        expected_count = int(expected_raw)
    except ValueError:
        raise SystemExit(f"EXPECTED_REPO_COUNT must be an integer, got: {expected_raw}")
    if expected_count <= 0:
        raise SystemExit(f"EXPECTED_REPO_COUNT must be > 0, got: {expected_count}")
    if len(declared) != expected_count:
        raise SystemExit(f"expected {expected_count} repos but got {len(declared)} in {repos_file}")
else:
    expected_count = len(declared)

metadata = {
    "profile_id": profile_id or "adhoc",
    "profile_source_kind": source_kind or "mixed",
    "expected_repo_count": expected_count,
    "target_repos_file": str(repos_file),
    "declared_repos": declared,
}
out_path.parent.mkdir(parents=True, exist_ok=True)
out_path.write_text(json.dumps(metadata, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
PY
}

if [[ ! "$RUN_COUNT" =~ ^[1-9][0-9]*$ ]]; then
  die "RUN_COUNT must be a positive integer, got '$RUN_COUNT'"
fi
if [[ "$RUN_COUNT" != "5" ]]; then
  die "RUN_COUNT must be 5 for this batch plan (got '$RUN_COUNT')"
fi
if [[ "$ACP_APPLY_TIMEOUTS_VIA_API" != "0" && "$ACP_APPLY_TIMEOUTS_VIA_API" != "1" ]]; then
  die "ACP_APPLY_TIMEOUTS_VIA_API must be 0 or 1, got '$ACP_APPLY_TIMEOUTS_VIA_API'"
fi
if [[ -n "$ACP_EXECUTION_STRATEGY" && "$ACP_EXECUTION_STRATEGY" != "sequential" && "$ACP_EXECUTION_STRATEGY" != "parallel" ]]; then
  die "ACP_EXECUTION_STRATEGY must be sequential|parallel (or empty), got '$ACP_EXECUTION_STRATEGY'"
fi
if [[ -n "$ACP_MAX_PARALLEL_TASKS" && ! "$ACP_MAX_PARALLEL_TASKS" =~ ^[1-9][0-9]*$ ]]; then
  die "ACP_MAX_PARALLEL_TASKS must be positive integer (or empty), got '$ACP_MAX_PARALLEL_TASKS'"
fi
if [[ -n "$ACP_FAILURE_POLICY" && "$ACP_FAILURE_POLICY" != "fail_fast" && "$ACP_FAILURE_POLICY" != "best_effort" ]]; then
  die "ACP_FAILURE_POLICY must be fail_fast|best_effort (or empty), got '$ACP_FAILURE_POLICY'"
fi
if [[ -n "$ACP_SHARD_DISCOVERY_MODE" && "$ACP_SHARD_DISCOVERY_MODE" != "heuristics" && "$ACP_SHARD_DISCOVERY_MODE" != "semantic" ]]; then
  die "ACP_SHARD_DISCOVERY_MODE must be heuristics|semantic (or empty), got '$ACP_SHARD_DISCOVERY_MODE'"
fi
if [[ -n "$ACP_REPO_SELECTION" && "$ACP_REPO_SELECTION" != "all" && "$ACP_REPO_SELECTION" != "backend_only" ]]; then
  die "ACP_REPO_SELECTION must be all|backend_only (or empty), got '$ACP_REPO_SELECTION'"
fi
if [[ "$BATCH_SKIP_PRECHECK" != "0" && "$BATCH_SKIP_PRECHECK" != "1" ]]; then
  die "BATCH_SKIP_PRECHECK must be 0 or 1, got '$BATCH_SKIP_PRECHECK'"
fi
case "$BATCH_FRONTEND_MODE" in
  auto|always|never)
    ;;
  *)
    die "BATCH_FRONTEND_MODE must be auto|always|never (got '$BATCH_FRONTEND_MODE')"
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

mkdir -p "$BATCH_ROOT" "$REPORTS_ROOT"
prepare_target_repos_file
collect_declared_repos
RUN_CLASSIFICATIONS_TSV="$BATCH_ROOT/backend-run-classifications.tsv"
echo -e "provider\trun_index\tfailure_class\tprocess_exit\tsummary_result\tfailure_reason\ttermination_signal\texpected_runs\tcompleted_runs\texpected_headless_runs\tcompleted_headless_runs\trunning_runs_detected\trun_results_rows\tfailure_subclass\tcancellation_like" >"$RUN_CLASSIFICATIONS_TSV"

PROVENARCH_SHA="$(git -C "$PROVENARCH_ROOT" rev-parse HEAD)"
PROVENARCH_BRANCH="$(git -C "$PROVENARCH_ROOT" rev-parse --abbrev-ref HEAD)"
CLAUDE_PATH="not-selected"
QWEN_PATH="not-selected"
CLAUDE_VERSION="not-selected"
QWEN_VERSION="not-selected"
if provider_selected "claude-code"; then
  CLAUDE_PATH="$(command -v "$ACP_CLAUDE_CMD_BIN")"
  CLAUDE_VERSION="$("$ACP_CLAUDE_CMD_BIN" --version | head -n1 | tr -d '\r')"
fi
if provider_selected "qwen-code"; then
  QWEN_PATH="$(command -v "$ACP_QWEN_CMD_BIN")"
  QWEN_VERSION="$("$ACP_QWEN_CMD_BIN" --version | head -n1 | tr -d '\r')"
fi
GENERATED_AT_UTC="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
DECLARED_REPOS_JSON_ESCAPED="$(python3 - "$DECLARED_REPOS_JSON" <<'PY'
import json
import sys
print(json.dumps(json.load(open(sys.argv[1], encoding="utf-8"))))
PY
)"
TIMEOUT_PROFILE_JSON="$(python3 - <<'PY'
import json
import os

defaults = {
    "step_timeout_sec": 1800,
    "heartbeat_sec": 30,
    "pipeline_timeout_sec": 2400,
    "pipeline_kill_grace_sec": 30,
    "api_ready_timeout_sec": 60,
    "api_init_timeout_sec": 120,
    "ui_init_poll_timeout_sec": 900,
    "ui_cancel_poll_timeout_sec": 420,
}
canonical = {
    "step_timeout_sec": "ACP_RUNTIME_STEP_TIMEOUT_SEC",
    "heartbeat_sec": "ACP_RUNTIME_HEARTBEAT_SEC",
    "pipeline_timeout_sec": "ACP_PIPELINE_TIMEOUT_SEC",
    "pipeline_kill_grace_sec": "ACP_PIPELINE_KILL_GRACE_SEC",
    "api_ready_timeout_sec": "ACP_API_READY_TIMEOUT_SEC",
    "api_init_timeout_sec": "ACP_API_INIT_TIMEOUT_SEC",
    "ui_init_poll_timeout_sec": "ACP_UI_INIT_POLL_TIMEOUT_SEC",
    "ui_cancel_poll_timeout_sec": "ACP_UI_CANCEL_POLL_TIMEOUT_SEC",
}
deprecated = {
    "pipeline_timeout_sec": ["ACP_FULL_RUN_PIPELINE_TIMEOUT_SEC"],
    "pipeline_kill_grace_sec": ["ACP_FULL_RUN_PIPELINE_KILL_GRACE_SEC"],
    "api_ready_timeout_sec": ["READY_TIMEOUT_SEC"],
    "ui_init_poll_timeout_sec": ["UI_E2E_INIT_TIMEOUT_SEC"],
    "ui_cancel_poll_timeout_sec": ["UI_E2E_CANCEL_TIMEOUT_SEC"],
}

def parse_positive(raw):
    try:
        value = int((raw or "").strip())
    except Exception:
        return None
    return value if value > 0 else None

effective = {}
source = {}
for key in (
    "step_timeout_sec",
    "heartbeat_sec",
    "pipeline_timeout_sec",
    "pipeline_kill_grace_sec",
    "api_ready_timeout_sec",
    "api_init_timeout_sec",
    "ui_init_poll_timeout_sec",
    "ui_cancel_poll_timeout_sec",
):
    value = parse_positive(os.environ.get(canonical[key], ""))
    if value is not None:
        effective[key] = value
        source[key] = "env"
        continue
    alias_used = None
    for alias in deprecated.get(key, []):
        alias_value = parse_positive(os.environ.get(alias, ""))
        if alias_value is not None:
            value = alias_value
            alias_used = alias
            break
    if value is not None:
        effective[key] = value
        source[key] = f"env_deprecated({alias_used})"
        continue
    effective[key] = defaults[key]
    source[key] = "default"

print(json.dumps({"effective": effective, "source": source}, ensure_ascii=True))
PY
)"
TIMEOUT_PROFILE_LINE="$(python3 - "$TIMEOUT_PROFILE_JSON" <<'PY'
import json
import sys
payload = json.loads(sys.argv[1])
keys = (
    "step_timeout_sec",
    "heartbeat_sec",
    "pipeline_timeout_sec",
    "pipeline_kill_grace_sec",
    "api_ready_timeout_sec",
    "api_init_timeout_sec",
    "ui_init_poll_timeout_sec",
    "ui_cancel_poll_timeout_sec",
)
parts = []
for key in keys:
    parts.append(f"{key}={payload['effective'][key]}({payload['source'][key]})")
print(" ".join(parts))
PY
)"
EXECUTION_PROFILE_JSON="$(python3 - <<'PY'
import json
import os

defaults = {
    "strategy": "sequential",
    "max_parallel_tasks": 1,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "heuristics",
    "repo_selection": "all",
}
allowed = {
    "strategy": {"sequential", "parallel"},
    "failure_policy": {"fail_fast", "best_effort"},
    "shard_discovery_mode": {"heuristics", "semantic"},
    "repo_selection": {"all", "backend_only"},
}

def parse_positive(raw):
    try:
        value = int((raw or "").strip())
    except Exception:
        return None
    return value if value > 0 else None

effective = {}
source = {}

strategy_raw = (os.environ.get("ACP_EXECUTION_STRATEGY", "") or "").strip()
strategy = strategy_raw if strategy_raw in allowed["strategy"] else defaults["strategy"]
source["strategy"] = "env" if strategy_raw in allowed["strategy"] else "default"
effective["strategy"] = strategy

max_raw = parse_positive(os.environ.get("ACP_MAX_PARALLEL_TASKS", ""))
max_parallel = max_raw if max_raw is not None else defaults["max_parallel_tasks"]
max_source = "env" if max_raw is not None else "default"
if strategy != "parallel":
    effective["max_parallel_tasks"] = 1
    if max_parallel != 1:
        max_source = f"{max_source}->effective(strategy={strategy})"
else:
    effective["max_parallel_tasks"] = max_parallel
source["max_parallel_tasks"] = max_source

failure_policy_raw = (os.environ.get("ACP_FAILURE_POLICY", "") or "").strip()
failure_policy = failure_policy_raw if failure_policy_raw in allowed["failure_policy"] else defaults["failure_policy"]
source["failure_policy"] = "env" if failure_policy_raw in allowed["failure_policy"] else "default"
effective["failure_policy"] = failure_policy

shard_mode_raw = (os.environ.get("ACP_SHARD_DISCOVERY_MODE", "") or "").strip()
shard_mode = shard_mode_raw if shard_mode_raw in allowed["shard_discovery_mode"] else defaults["shard_discovery_mode"]
source["shard_discovery_mode"] = "env" if shard_mode_raw in allowed["shard_discovery_mode"] else "default"
effective["shard_discovery_mode"] = shard_mode

repo_selection_raw = (os.environ.get("ACP_REPO_SELECTION", "") or "").strip()
repo_selection = repo_selection_raw if repo_selection_raw in allowed["repo_selection"] else defaults["repo_selection"]
source["repo_selection"] = "env" if repo_selection_raw in allowed["repo_selection"] else "default"
effective["repo_selection"] = repo_selection

print(json.dumps({"effective": effective, "source": source}, ensure_ascii=True))
PY
)"
EXECUTION_PROFILE_LINE="$(python3 - "$EXECUTION_PROFILE_JSON" <<'PY'
import json
import sys
payload = json.loads(sys.argv[1])
keys = (
    "strategy",
    "max_parallel_tasks",
    "failure_policy",
    "shard_discovery_mode",
    "repo_selection",
)
parts = []
for key in keys:
    parts.append(f"{key}={payload['effective'][key]}({payload['source'][key]})")
print(" ".join(parts))
PY
)"

export BATCH_PRE_GENERATED_AT_UTC="$GENERATED_AT_UTC"
export BATCH_PRE_PROVENARCH_ROOT="$PROVENARCH_ROOT"
export BATCH_PRE_TARGET_REPOS_FILE="$RESOLVED_TARGET_REPOS_FILE"
export BATCH_PRE_PROVENARCH_SHA="$PROVENARCH_SHA"
export BATCH_PRE_PROVENARCH_BRANCH="$PROVENARCH_BRANCH"
export BATCH_PRE_CLAUDE_PATH="$CLAUDE_PATH"
export BATCH_PRE_CLAUDE_VERSION="$CLAUDE_VERSION"
export BATCH_PRE_QWEN_PATH="$QWEN_PATH"
export BATCH_PRE_QWEN_VERSION="$QWEN_VERSION"
export BATCH_PRE_DECLARED_REPOS_JSON="$DECLARED_REPOS_JSON_ESCAPED"
export BATCH_PRE_TIMEOUT_PROFILE_JSON="$TIMEOUT_PROFILE_JSON"
export BATCH_PRE_APPLY_TIMEOUTS_VIA_API="$ACP_APPLY_TIMEOUTS_VIA_API"
export BATCH_PRE_EXECUTION_PROFILE_JSON="$EXECUTION_PROFILE_JSON"
export BATCH_PRE_SWEEP_ID="$SWEEP_ID"

python3 - "$BATCH_ROOT/preflight.json" <<'PY'
import json
import os
import sys

payload = {
    "generated_at_utc": os.environ["BATCH_PRE_GENERATED_AT_UTC"],
    "provenarch_root": os.environ["BATCH_PRE_PROVENARCH_ROOT"],
    "provenarch_sha": os.environ["BATCH_PRE_PROVENARCH_SHA"],
    "provenarch_branch": os.environ["BATCH_PRE_PROVENARCH_BRANCH"],
    "target_repos_file": os.environ["BATCH_PRE_TARGET_REPOS_FILE"],
    "declared_repos_meta": json.loads(os.environ["BATCH_PRE_DECLARED_REPOS_JSON"]),
    "apply_timeouts_via_api": os.environ["BATCH_PRE_APPLY_TIMEOUTS_VIA_API"] == "1",
    "timeout_profile": json.loads(os.environ["BATCH_PRE_TIMEOUT_PROFILE_JSON"]),
    "execution_profile": json.loads(os.environ["BATCH_PRE_EXECUTION_PROFILE_JSON"]),
    "sweep_id": os.environ.get("BATCH_PRE_SWEEP_ID", "baseline") or "baseline",
    "runtimes": {
        "claude": {
            "path": os.environ["BATCH_PRE_CLAUDE_PATH"],
            "version_line": os.environ["BATCH_PRE_CLAUDE_VERSION"],
        },
        "qwen": {
            "path": os.environ["BATCH_PRE_QWEN_PATH"],
            "version_line": os.environ["BATCH_PRE_QWEN_VERSION"],
        },
    },
}
with open(sys.argv[1], "w", encoding="utf-8") as f:
    json.dump(payload, f, ensure_ascii=True, indent=2)
    f.write("\n")
PY

EXPECTED_REPO_COUNT_RESOLVED="$(python3 - "$DECLARED_REPOS_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding="utf-8"))
print(int(payload.get("expected_repo_count", len(payload.get("declared_repos") or []))))
PY
)"

log "target repos input: file=$RESOLVED_TARGET_REPOS_FILE profile_id=${PROFILE_ID:-adhoc} source_kind=${PROFILE_SOURCE_KIND:-mixed} expected_repo_count=$EXPECTED_REPO_COUNT_RESOLVED"
log "preflight versions: claude='$CLAUDE_VERSION' qwen='$QWEN_VERSION'"
log "preflight timeout profile: apply_via_api=$ACP_APPLY_TIMEOUTS_VIA_API $TIMEOUT_PROFILE_LINE"
log "preflight execution profile: sweep_id=${SWEEP_ID:-baseline} $EXECUTION_PROFILE_LINE"
log "batch shard selection: providers=$SELECTED_PROVIDERS_CSV runs=$SELECTED_RUN_INDEXES_CSV skip_precheck=$BATCH_SKIP_PRECHECK frontend_mode=$BATCH_FRONTEND_MODE"
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
    log "full-run provider=$provider run=$i tmp_root=$run_dir"
    process_exit=0
    (
      cd "$PROVENARCH_ROOT"
      TARGET_REPOS_FILE="$RESOLVED_TARGET_REPOS_FILE" \
      TMP_ROOT="$run_dir" \
      KEEP_TMP=1 \
      ITERATIONS=1 \
      RUN_QUALITY_GATES=1 \
      PROFILE_ID="${PROFILE_ID:-adhoc}" \
      PROFILE_SOURCE_KIND="${PROFILE_SOURCE_KIND:-mixed}" \
      EXPECTED_REPO_COUNT="$EXPECTED_REPO_COUNT_RESOLVED" \
      ACP_RUNTIME_PROVIDER="$provider" \
      ACP_CLAUDE_CMD="$ACP_CLAUDE_CMD_BIN" \
      ACP_QWEN_CMD="$ACP_QWEN_CMD_BIN" \
      ACP_APPLY_TIMEOUTS_VIA_API="$ACP_APPLY_TIMEOUTS_VIA_API" \
      SWEEP_ID="${SWEEP_ID:-baseline}" \
      ACP_RUNTIME_STEP_TIMEOUT_SEC="${ACP_RUNTIME_STEP_TIMEOUT_SEC:-}" \
      ACP_RUNTIME_HEARTBEAT_SEC="${ACP_RUNTIME_HEARTBEAT_SEC:-}" \
      ACP_PIPELINE_TIMEOUT_SEC="${ACP_PIPELINE_TIMEOUT_SEC:-}" \
      ACP_PIPELINE_KILL_GRACE_SEC="${ACP_PIPELINE_KILL_GRACE_SEC:-}" \
      ACP_API_READY_TIMEOUT_SEC="${ACP_API_READY_TIMEOUT_SEC:-}" \
      ACP_API_INIT_TIMEOUT_SEC="${ACP_API_INIT_TIMEOUT_SEC:-}" \
      ACP_UI_INIT_POLL_TIMEOUT_SEC="${ACP_UI_INIT_POLL_TIMEOUT_SEC:-}" \
      ACP_UI_CANCEL_POLL_TIMEOUT_SEC="${ACP_UI_CANCEL_POLL_TIMEOUT_SEC:-}" \
      ACP_EXECUTION_STRATEGY="${ACP_EXECUTION_STRATEGY:-}" \
      ACP_MAX_PARALLEL_TASKS="${ACP_MAX_PARALLEL_TASKS:-}" \
      ACP_FAILURE_POLICY="${ACP_FAILURE_POLICY:-}" \
      ACP_SHARD_DISCOVERY_MODE="${ACP_SHARD_DISCOVERY_MODE:-}" \
      ACP_REPO_SELECTION="${ACP_REPO_SELECTION:-}" \
      ./scripts/full-run-ai-advent.sh
    ) >"$run_dir/batch-driver.log" 2>&1 || process_exit=$?

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
  if ! should_run_frontend_for_selection; then
    output_dir="$BATCH_ROOT/frontend/$provider"
    mkdir -p "$output_dir"
    runtime_cmd="$ACP_QWEN_CMD_BIN"
    if [[ "$provider" == "claude-code" ]]; then
      runtime_cmd="$ACP_CLAUDE_CMD_BIN"
    fi
    write_frontend_status_json \
      "$output_dir/frontend-e2e-result.json" \
      "$provider" \
      "init-inspect" \
      "skipped" \
      "frontend_mode_${BATCH_FRONTEND_MODE}_selection_${SELECTED_RUN_INDEXES_CSV}" \
      "$output_dir/frontend-workspace" \
      "$output_dir" \
      "$runtime_cmd"
    log "frontend live e2e skipped provider=$provider mode=$BATCH_FRONTEND_MODE runs=$SELECTED_RUN_INDEXES_CSV"
    continue
  fi

  backend_run_dir="$BATCH_ROOT/$provider/run1"
  workspace="$backend_run_dir/arch-workspace"
  output_dir="$BATCH_ROOT/frontend/$provider"
  frontend_workspace="$output_dir/frontend-workspace"
  run_results_path="$backend_run_dir/run-results.tsv"
  refresh_run_id=""
  snapshot_reports=""
  mkdir -p "$output_dir"
  if [[ -f "$run_results_path" ]]; then
    refresh_run_id="$(awk -F'\t' '$2=="headless" && $4=="refresh" {print $5}' "$run_results_path" | tail -n1)"
  fi
  if [[ -n "$refresh_run_id" ]]; then
    snapshot_reports="$backend_run_dir/snapshots/$refresh_run_id/reports"
  fi

  if [[ ! -d "$workspace" ]]; then
    frontend_failures=$((frontend_failures + 1))
    write_frontend_status_json \
      "$output_dir/frontend-e2e-result.json" \
      "$provider" \
      "init-inspect" \
      "failed" \
      "backend_workspace_missing" \
      "$workspace" \
      "$output_dir" \
      "${ACP_CLAUDE_CMD_BIN}/${ACP_QWEN_CMD_BIN}"
    log "frontend e2e failed provider=$provider reason=backend_workspace_missing (workspace=$workspace)"
    continue
  fi

  rm -rf "$frontend_workspace"
  cp -a "$workspace" "$frontend_workspace"
  frontend_source="workspace-fallback"
  if [[ -d "$snapshot_reports" ]]; then
    rm -rf "$frontend_workspace/reports"
    cp -a "$snapshot_reports" "$frontend_workspace/reports"
    frontend_source="snapshot"
  fi

  log "frontend live e2e provider=$provider workspace=$frontend_workspace artifact_source=$frontend_source refresh_run_id=${refresh_run_id:-unknown}"
  if ! (
    cd "$PROVENARCH_ROOT"
    WORKSPACE="$frontend_workspace" \
    RUNTIME_PROVIDER="$provider" \
    OUTPUT_DIR="$output_dir" \
    UI_E2E_EXPECTED_REPO_COUNT="$EXPECTED_REPO_COUNT_RESOLVED" \
    ACP_CLAUDE_CMD="$ACP_CLAUDE_CMD_BIN" \
    ACP_QWEN_CMD="$ACP_QWEN_CMD_BIN" \
    ACP_UI_INIT_POLL_TIMEOUT_SEC="${ACP_UI_INIT_POLL_TIMEOUT_SEC:-}" \
    ACP_UI_CANCEL_POLL_TIMEOUT_SEC="${ACP_UI_CANCEL_POLL_TIMEOUT_SEC:-}" \
    ACP_EXECUTION_STRATEGY="${ACP_EXECUTION_STRATEGY:-}" \
    ACP_MAX_PARALLEL_TASKS="${ACP_MAX_PARALLEL_TASKS:-}" \
    ACP_FAILURE_POLICY="${ACP_FAILURE_POLICY:-}" \
    ACP_SHARD_DISCOVERY_MODE="${ACP_SHARD_DISCOVERY_MODE:-}" \
    ACP_REPO_SELECTION="${ACP_REPO_SELECTION:-}" \
    ./scripts/frontend-live-e2e.sh
  ) >"$output_dir/driver.log" 2>&1; then
    frontend_failures=$((frontend_failures + 1))
    log "frontend e2e failed provider=$provider (see $output_dir/driver.log)"
  fi
done

for provider in "${SELECTED_PROVIDERS[@]}"; do
  cancel_output_dir="$BATCH_ROOT/frontend-cancel/$provider"
  mkdir -p "$cancel_output_dir"
  runtime_cmd="$ACP_QWEN_CMD_BIN"
  if [[ "$provider" == "claude-code" ]]; then
    runtime_cmd="$ACP_CLAUDE_CMD_BIN"
  fi
  if ! should_run_frontend_for_selection; then
    FRONTEND_CANCEL_SKIPPED=$((FRONTEND_CANCEL_SKIPPED + 1))
    write_frontend_status_json \
      "$cancel_output_dir/frontend-cancel-result.json" \
      "$provider" \
      "cancel-refresh" \
      "skipped" \
      "frontend_mode_${BATCH_FRONTEND_MODE}_selection_${SELECTED_RUN_INDEXES_CSV}" \
      "$BATCH_ROOT/frontend/$provider/frontend-workspace" \
      "$cancel_output_dir" \
      "$runtime_cmd"
    log "frontend cancel smoke skipped provider=$provider mode=$BATCH_FRONTEND_MODE runs=$SELECTED_RUN_INDEXES_CSV"
    continue
  fi

  frontend_workspace="$BATCH_ROOT/frontend/$provider/frontend-workspace"

  if [[ ! -d "$frontend_workspace" ]]; then
    FRONTEND_CANCEL_SKIPPED=$((FRONTEND_CANCEL_SKIPPED + 1))
    write_frontend_status_json \
      "$cancel_output_dir/frontend-cancel-result.json" \
      "$provider" \
      "cancel-refresh" \
      "skipped" \
      "frontend_workspace_missing" \
      "$frontend_workspace" \
      "$cancel_output_dir" \
      "$runtime_cmd"
    log "frontend cancel smoke skipped provider=$provider reason=frontend_workspace_missing"
    continue
  fi

  log "frontend cancel smoke provider=$provider workspace=$frontend_workspace"
  if ! (
    cd "$PROVENARCH_ROOT"
    WORKSPACE="$frontend_workspace" \
    RUNTIME_PROVIDER="$provider" \
    OUTPUT_DIR="$cancel_output_dir" \
    UI_E2E_SCENARIO="cancel-refresh" \
    UI_E2E_EXPECTED_REPO_COUNT="$EXPECTED_REPO_COUNT_RESOLVED" \
    ACP_CLAUDE_CMD="$ACP_CLAUDE_CMD_BIN" \
    ACP_QWEN_CMD="$ACP_QWEN_CMD_BIN" \
    ACP_UI_INIT_POLL_TIMEOUT_SEC="${ACP_UI_INIT_POLL_TIMEOUT_SEC:-}" \
    ACP_UI_CANCEL_POLL_TIMEOUT_SEC="${ACP_UI_CANCEL_POLL_TIMEOUT_SEC:-}" \
    ACP_EXECUTION_STRATEGY="${ACP_EXECUTION_STRATEGY:-}" \
    ACP_MAX_PARALLEL_TASKS="${ACP_MAX_PARALLEL_TASKS:-}" \
    ACP_FAILURE_POLICY="${ACP_FAILURE_POLICY:-}" \
    ACP_SHARD_DISCOVERY_MODE="${ACP_SHARD_DISCOVERY_MODE:-}" \
    ACP_REPO_SELECTION="${ACP_REPO_SELECTION:-}" \
    ./scripts/frontend-live-e2e.sh
  ) >"$cancel_output_dir/driver.log" 2>&1; then
    FRONTEND_CANCEL_FAILURES=$((FRONTEND_CANCEL_FAILURES + 1))
    if [[ -f "$cancel_output_dir/frontend-e2e-result.json" ]]; then
      cp "$cancel_output_dir/frontend-e2e-result.json" "$cancel_output_dir/frontend-cancel-result.json"
    else
      write_frontend_status_json \
        "$cancel_output_dir/frontend-cancel-result.json" \
        "$provider" \
        "cancel-refresh" \
        "failed" \
        "frontend_live_e2e_failed" \
        "$frontend_workspace" \
        "$cancel_output_dir" \
        "$runtime_cmd" \
        "$cancel_output_dir/server.log" \
        "$cancel_output_dir/playwright.log"
    fi
    log "frontend cancel smoke failed provider=$provider (see $cancel_output_dir/driver.log)"
    continue
  fi

  if [[ ! -f "$cancel_output_dir/frontend-cancel-result.json" ]] && [[ -f "$cancel_output_dir/frontend-e2e-result.json" ]]; then
    cp "$cancel_output_dir/frontend-e2e-result.json" "$cancel_output_dir/frontend-cancel-result.json"
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

log "backend failure classes: precheck_failed=$PRECHECK_FAILED_FAILURES runtime_parse=$RUNTIME_PARSE_FAILURES runner_unavailable=$RUNNER_UNAVAILABLE_FAILURES runtime_timeout=$RUNTIME_TIMEOUT_FAILURES infra_signal_terminated=$INFRA_SIGNAL_TERMINATED_FAILURES infra_incomplete_cycle=$INFRA_INCOMPLETE_CYCLE_FAILURES quality_gates_failed=$QUALITY_GATES_FAILED_FAILURES summary_missing=$SUMMARY_MISSING_FAILURES cancellation_like=$CANCELLATION_LIKE_FAILURES other=$OTHER_FAILURES"

if [[ "$failed_runs" -ne 0 || "$frontend_failures" -ne 0 || "$FRONTEND_CANCEL_FAILURES" -ne 0 ]]; then
  die "batch completed with failures: full_run_failed=$failed_runs frontend_failed=$frontend_failures frontend_cancel_failed=$FRONTEND_CANCEL_FAILURES frontend_cancel_skipped=$FRONTEND_CANCEL_SKIPPED precheck_failed=$PRECHECK_FAILED_FAILURES runtime_parse=$RUNTIME_PARSE_FAILURES runner_unavailable=$RUNNER_UNAVAILABLE_FAILURES runtime_timeout=$RUNTIME_TIMEOUT_FAILURES infra_signal_terminated=$INFRA_SIGNAL_TERMINATED_FAILURES infra_incomplete_cycle=$INFRA_INCOMPLETE_CYCLE_FAILURES quality_gates_failed=$QUALITY_GATES_FAILED_FAILURES summary_missing=$SUMMARY_MISSING_FAILURES cancellation_like=$CANCELLATION_LIKE_FAILURES other=$OTHER_FAILURES"
fi

log "batch completed successfully"
exit 0
