#!/usr/bin/env bash
# shellcheck disable=SC2317,SC2329 # Signal/trap callbacks and their helpers are invoked indirectly via trap handlers.
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
# shellcheck source=scripts/legacy-env-guard.sh
source "$PROVENARCH_ROOT/scripts/legacy-env-guard.sh"
# shellcheck source=scripts/repos-meta-fields.sh
source "$PROVENARCH_ROOT/scripts/repos-meta-fields.sh"
# shellcheck source=scripts/preflight-log.sh
source "$PROVENARCH_ROOT/scripts/preflight-log.sh"
# shellcheck source=scripts/timeout-env-keys.sh
source "$PROVENARCH_ROOT/scripts/timeout-env-keys.sh"
# shellcheck source=scripts/execution-env-keys.sh
source "$PROVENARCH_ROOT/scripts/execution-env-keys.sh"
BATCH_SCRIPT="${BATCH_SCRIPT:-$PROVENARCH_ROOT/scripts/full-run-batch.sh}"
E2E_MATRIX_FILE="${E2E_MATRIX_FILE:-}"
MATRIX_ID="${MATRIX_ID:-matrix-$(date -u +'%Y%m%dT%H%M%SZ')}"
RUN_COUNT="${RUN_COUNT:-1}"
E2E_TMP_ROOT="${E2E_TMP_ROOT:-/tmp/provenarch-test_arch_project}"
REPORTS_ROOT="${REPORTS_ROOT:-$E2E_TMP_ROOT/reports}"
MATRIX_ROOT="${MATRIX_ROOT:-$E2E_TMP_ROOT/matrix/$MATRIX_ID}"
ACP_CLAUDE_CMD_BIN="${ACP_CLAUDE_CMD_BIN:-claude}"
ACP_QWEN_CMD_BIN="${ACP_QWEN_CMD_BIN:-qwen}"
ACP_CODEX_CMD_BIN="${ACP_CODEX_CMD_BIN:-codex}"
if [[ -z "${ACP_CODEX_MODEL+x}" ]]; then
  ACP_CODEX_MODEL="gpt-5.5"
fi
if [[ -z "${ACP_CODEX_REASONING_EFFORT+x}" ]]; then
  ACP_CODEX_REASONING_EFFORT="xhigh"
fi
ACP_APPLY_TIMEOUTS_VIA_API="${ACP_APPLY_TIMEOUTS_VIA_API:-1}"
E2E_MATRIX_RELEASE_MODE="${E2E_MATRIX_RELEASE_MODE:-auto}"
BATCH_PROVIDER_FILTER="${BATCH_PROVIDER_FILTER:-all}"
BATCH_RUN_SELECTION="${BATCH_RUN_SELECTION:-all}"
BATCH_FRONTEND_MODE="${BATCH_FRONTEND_MODE:-}"
UI_E2E_HEADED="${UI_E2E_HEADED:-}"
MATRIX_DRIVER_LOG="${MATRIX_DRIVER_LOG:-$MATRIX_ROOT/driver.log}"
MATRIX_TIMEOUT_PROFILE_FILE="${MATRIX_TIMEOUT_PROFILE_FILE:-$MATRIX_ROOT/timeout-profile.txt}"
MATRIX_PROFILE_STATUS_HEARTBEAT_SEC="${MATRIX_PROFILE_STATUS_HEARTBEAT_SEC:-10}"
MATRIX_PROFILE_STATUS_STALE_SEC="${MATRIX_PROFILE_STATUS_STALE_SEC:-0}"
E2E_MATRIX_MIN_FREE_KB="${E2E_MATRIX_MIN_FREE_KB:-5242880}"
PROFILE_REPOS_FILE_RESOLVED=""
PROFILE_SOURCE_KIND_EFFECTIVE=""
PROFILE_EXPECTED_REPO_COUNT_RESOLVED=0
PROFILE_META_CACHE_KEY=""
PROFILE_META_CACHE_REPOS_FILE=""
PROFILE_META_CACHE_SOURCE_KIND="mixed"
PROFILE_META_CACHE_EXPECTED_REPO_COUNT=0
MATRIX_TIMEOUT_PROFILE=""
MATRIX_TIMEOUT_ENV_ASSIGNMENTS=()
declare -a MATRIX_ALL_PROVIDERS=("qwen-code" "claude-code" "codex-code")
declare -a MATRIX_SELECTED_PROVIDERS=()
declare -a MATRIX_SELECTED_RUN_INDEXES=()
MATRIX_SELECTED_PROVIDERS_CSV=""
MATRIX_SELECTED_RUN_INDEXES_CSV=""
MATRIX_STATUS_ROOT="${MATRIX_ROOT}/profile-status"
CURRENT_PROFILE_STATUS_FILE=""
CURRENT_PROFILE_ID=""
CURRENT_PROFILE_SLUG=""
CURRENT_SWEEP_ID=""
CURRENT_BATCH_ID=""
CURRENT_BATCH_ROOT=""
CURRENT_DRIVER_LOG=""
CURRENT_SOURCE_KIND=""
CURRENT_EXPECTED_REPO_COUNT=""
CURRENT_REPOS_FILE=""
CURRENT_SWEEP_STRATEGY=""
CURRENT_SWEEP_MAX_PARALLEL=""
CURRENT_SWEEP_FAILURE_POLICY=""
CURRENT_SWEEP_SHARD_MODE=""
CURRENT_PROFILE_STATUS_HEARTBEAT_PID=""

log() {
  local line
  line="[batch-matrix] $*"
  printf '%s\n' "$line" >&2
  if [[ -n "${MATRIX_DRIVER_LOG:-}" ]]; then
    mkdir -p "$(dirname "$MATRIX_DRIVER_LOG")"
    printf '%s\n' "$line" >>"$MATRIX_DRIVER_LOG"
  fi
}

die() {
  local line
  line="[batch-matrix][error] $*"
  echo "$line" >&2
  if [[ -n "${MATRIX_DRIVER_LOG:-}" ]]; then
    mkdir -p "$(dirname "$MATRIX_DRIVER_LOG")"
    printf '%s\n' "$line" >>"$MATRIX_DRIVER_LOG"
  fi
  exit 1
}

write_matrix_operational_blocker_report() {
  local reason="$1"
  if ! command -v python3 >/dev/null 2>&1; then
    return 0
  fi
  python3 - "$MATRIX_ID" "$E2E_MATRIX_FILE" "$MATRIX_ROOT" "$REPORTS_ROOT" "$MATRIX_STATUS_ROOT" "$MATRIX_DRIVER_LOG" "$reason" "$MATRIX_SELECTED_PROVIDERS_CSV" "$MATRIX_SELECTED_RUN_INDEXES_CSV" "${RELEASE_MODE:-0}" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

matrix_id = sys.argv[1]
matrix_file = sys.argv[2]
matrix_root = Path(sys.argv[3]).resolve()
reports_root = Path(sys.argv[4]).resolve()
status_root = Path(sys.argv[5]).resolve()
driver_log = sys.argv[6]
reason = sys.argv[7]
selected_providers = [item for item in sys.argv[8].split(",") if item]
selected_run_indexes = [item for item in sys.argv[9].split(",") if item]
release_mode = str(sys.argv[10]).strip() == "1"

now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
status_root.mkdir(parents=True, exist_ok=True)
reports_root.mkdir(parents=True, exist_ok=True)
inventory_root = matrix_root / "inventory"
inventory_root.mkdir(parents=True, exist_ok=True)

status_path = status_root / "matrix-operational-preflight.json"
inventory_path = inventory_root / "matrix-operational-preflight.json"
profile_matrix_md = reports_root / f"profile_matrix_{matrix_id}.md"
profile_matrix_tsv = reports_root / f"profile_matrix_{matrix_id}.tsv"
if release_mode:
    verdict_md = reports_root / f"release_verdict_{matrix_id}.md"
    verdict_json = reports_root / f"release_verdict_{matrix_id}.json"
else:
    verdict_md = reports_root / f"matrix_result_{matrix_id}.md"
    verdict_json = reports_root / f"matrix_result_{matrix_id}.json"
records_path = matrix_root / "profile-runs.jsonl"
records_path.parent.mkdir(parents=True, exist_ok=True)

batch_id = f"{matrix_id}-operational-preflight"
inventory_payload = {
    "generated_at": now,
    "matrix_id": matrix_id,
    "matrix_file": matrix_file,
    "profile_id": "matrix-operational-preflight",
    "sweep_id": "preflight",
    "batch_id": batch_id,
    "batch_root": "-",
    "terminal_status": "failed",
    "failure_reason": "operational_host_preflight_failed",
    "operational_blocker": reason,
    "selected_providers": selected_providers,
    "selected_run_indexes": selected_run_indexes,
    "key_paths": {
        "driver_log": driver_log,
        "profile_status_file": str(status_path),
        "run_matrix_tsv": "-",
        "run_matrix_md": "-",
        "frontend_matrix_md": "-",
        "execution_report_md": "-",
    },
    "raw_output_refs": [],
}
inventory_path.write_text(json.dumps(inventory_payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")

status_payload = {
    "profile_id": "matrix-operational-preflight",
    "profile_slug": "matrix-operational-preflight",
    "batch_id": batch_id,
    "matrix_id": matrix_id,
    "matrix_file": matrix_file,
    "selected_providers": selected_providers,
    "selected_run_indexes": selected_run_indexes,
    "source_kind": "operational",
    "expected_repo_count": 0,
    "repos_file": "-",
    "status": "failed",
    "failure_reason": "operational_host_preflight_failed",
    "sweep_id": "preflight",
    "execution": {
        "strategy": "-",
        "max_parallel_tasks": 0,
        "failure_policy": "-",
        "shard_discovery_mode": "-",
    },
    "batch_root": "-",
    "run_matrix_tsv": "-",
    "run_matrix_md": "-",
    "frontend_matrix_md": "-",
    "execution_report_md": "-",
    "driver_log": driver_log,
    "inventory_json": str(inventory_path),
    "raw_output_refs": [],
    "operational_blocker": reason,
    "updated_at": now,
}
status_path.write_text(json.dumps(status_payload, ensure_ascii=True) + "\n", encoding="utf-8")

record = dict(status_payload)
record["strict_status"] = "failed"
record["blocking_reasons"] = [reason]
records_path.write_text(json.dumps(record, ensure_ascii=True) + "\n", encoding="utf-8")

profile_matrix_tsv.write_text(
    "\t".join(
        [
            "profile_id",
            "sweep_id",
            "batch_id",
            "status",
            "strict_status",
            "failure_reason",
            "blocking_reasons",
            "inventory_json",
        ]
    )
    + "\n"
    + "\t".join(
        [
            "matrix-operational-preflight",
            "preflight",
            batch_id,
            "failed",
            "failed",
            "operational_host_preflight_failed",
            reason,
            str(inventory_path),
        ]
    )
    + "\n",
    encoding="utf-8",
)
profile_matrix_md.write_text(
    "\n".join(
        [
            "# Profile Matrix",
            "",
            "| profile_id | sweep_id | batch_id | status | strict | failure_reason | blockers | inventory |",
            "|---|---|---|---|---|---|---|---|",
            f"| matrix-operational-preflight | preflight | {batch_id} | failed | failed | operational_host_preflight_failed | {reason} | {inventory_path} |",
        ]
    )
    + "\n",
    encoding="utf-8",
)

record_payload = {
    "profile_id": "matrix-operational-preflight",
    "sweep_id": "preflight",
    "batch_id": batch_id,
    "status": "failed",
    "strict_status": "failed",
    "blocking_reasons": [reason],
    "backend": {
        "hard_pass": 0,
        "total_runs": 0,
        "runtime_contract_failed_failures": 0,
        "runner_unavailable_failures": 0,
        "runtime_timeout_failures": 0,
        "precheck_failed_failures": 1,
    },
    "artifacts": {
        "driver_log": driver_log,
        "inventory_json": str(inventory_path),
        "raw_output_ref_count": 0,
    },
}
if release_mode:
    verdict_payload = {
        "matrix_id": matrix_id,
        "generated_at_utc": now,
        "verdict": "FAIL",
        "release_state": "RELEASE BLOCKED",
        "profile_sweep_runs": 1,
        "strict_pass_runs": 0,
        "strict_fail_runs": 1,
        "release_contract": {
            "mode": "release",
            "selected_providers": selected_providers,
            "selected_run_indexes": selected_run_indexes,
            "contract_status": "failed",
            "blocking_reasons": [reason],
        },
        "records": [record_payload],
    }
else:
    verdict_payload = {
        "matrix_id": matrix_id,
        "generated_at_utc": now,
        "result": "FAIL",
        "mode": "non-release",
        "profile_sweep_runs": 1,
        "strict_pass_runs": 0,
        "strict_fail_runs": 1,
        "blocking_reasons": [reason],
        "records": [record_payload],
    }
verdict_json.write_text(json.dumps(verdict_payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
title = "Release Verdict" if release_mode else "Matrix Result"
status_key = "- verdict: FAIL" if release_mode else "- result: FAIL"
extra_lines = ["- release_state: RELEASE BLOCKED", "- release_contract_status: failed"] if release_mode else ["- mode: non-release"]
verdict_md.write_text(
    "\n".join(
        [
            f"# {title}: {matrix_id}",
            "",
            f"- generated_at_utc: {now}",
            status_key,
            *extra_lines,
            "",
            "## Blocking Items",
            f"- matrix-operational-preflight / preflight ({batch_id}):",
            f"  - {reason}",
            f"  - inventory: {inventory_path} (raw_output_refs=0)",
        ]
    )
    + "\n",
    encoding="utf-8",
)
PY
}

operational_host_preflight_failed() {
  local reason="$1"
  write_matrix_operational_blocker_report "$reason" || true
  die "operational_host_preflight_failed: $reason"
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    operational_host_preflight_failed "required command is unavailable: $cmd"
  fi
}

ensure_writable_dir() {
  local path="$1"
  local label="$2"
  if ! mkdir -p "$path" 2>/dev/null; then
    operational_host_preflight_failed "cannot create $label directory at $path"
  fi
  local probe="$path/.write-probe.$$"
  if ! : >"$probe" 2>/dev/null; then
    operational_host_preflight_failed "$label directory is not writable: $path"
  fi
  rm -f "$probe" >/dev/null 2>&1 || true
}

ensure_min_free_space() {
  local path="$1"
  local label="$2"
  local min_free_kb="$3"
  if [[ ! "$min_free_kb" =~ ^[0-9]+$ ]]; then
    operational_host_preflight_failed "E2E_MATRIX_MIN_FREE_KB must be a non-negative integer, got '$min_free_kb'"
  fi
  if (( min_free_kb == 0 )); then
    return 0
  fi
  if ! mkdir -p "$path" 2>/dev/null; then
    operational_host_preflight_failed "cannot create $label directory at $path"
  fi
  local free_kb
  if ! free_kb="$(df -Pk "$path" 2>/dev/null | awk 'NR == 2 {print $4}')"; then
    operational_host_preflight_failed "failed to read free disk space for $label at $path"
  fi
  if [[ ! "$free_kb" =~ ^[0-9]+$ ]]; then
    operational_host_preflight_failed "failed to parse free disk space for $label at $path"
  fi
  if (( free_kb < min_free_kb )); then
    operational_host_preflight_failed "$label has insufficient free disk space: path=$path free_kb=$free_kb required_kb=$min_free_kb"
  fi
  log "host preflight: $label free_kb=$free_kb required_kb=$min_free_kb"
}

run_host_preflight_checks() {
  if array_contains "qwen-code" "${MATRIX_SELECTED_PROVIDERS[@]-}"; then
    local qwen_path
    qwen_path="$(command -v "$ACP_QWEN_CMD_BIN" 2>/dev/null || true)"
    if [[ -z "$qwen_path" ]]; then
      operational_host_preflight_failed "qwen binary not found in PATH (ACP_QWEN_CMD_BIN=$ACP_QWEN_CMD_BIN)"
    fi
    local qwen_version
    if ! qwen_version="$("$ACP_QWEN_CMD_BIN" --version 2>/dev/null | head -n1 | tr -d '\r')"; then
      operational_host_preflight_failed "failed to read qwen version from $ACP_QWEN_CMD_BIN"
    fi
    log "host preflight: qwen_bin=$qwen_path qwen_version=${qwen_version:-unknown}"
  fi
  ensure_writable_dir "$E2E_TMP_ROOT" "e2e_tmp_root"
  ensure_writable_dir "$REPORTS_ROOT" "reports_root"
  ensure_writable_dir "$MATRIX_ROOT" "matrix_root"
  ensure_writable_dir "$MATRIX_STATUS_ROOT" "matrix_status_root"
  ensure_min_free_space "$E2E_TMP_ROOT" "e2e_tmp_root" "$E2E_MATRIX_MIN_FREE_KB"
  ensure_min_free_space "$REPORTS_ROOT" "reports_root" "$E2E_MATRIX_MIN_FREE_KB"
  ensure_min_free_space "$MATRIX_ROOT" "matrix_root" "$E2E_MATRIX_MIN_FREE_KB"
}

normalize_release_mode() {
  local raw="${1:-auto}"
  case "$raw" in
    auto)
      if [[ "$MATRIX_ID" == release-* ]]; then
        printf '1'
      else
        printf '0'
      fi
      ;;
    1|true|TRUE|yes|YES|on|ON)
      printf '1'
      ;;
    0|false|FALSE|no|NO|off|OFF)
      printf '0'
      ;;
    *)
      die "E2E_MATRIX_RELEASE_MODE must be auto|0|1 (or boolean aliases), got '$raw'"
      ;;
  esac
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

slugify() {
  local value
  value="$(echo "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
  if [[ -z "$value" ]]; then
    value="item"
  fi
  printf '%s' "$value"
}

write_current_profile_status() {
  local status="$1"
  local failure_reason="${2:-none}"
  [[ -z "$CURRENT_PROFILE_STATUS_FILE" ]] && return 0
  mkdir -p "$(dirname "$CURRENT_PROFILE_STATUS_FILE")"
  python3 - "$CURRENT_PROFILE_STATUS_FILE" "$status" "$failure_reason" "$CURRENT_PROFILE_ID" "$CURRENT_PROFILE_SLUG" "$CURRENT_BATCH_ID" "$CURRENT_SOURCE_KIND" "$CURRENT_EXPECTED_REPO_COUNT" "$CURRENT_REPOS_FILE" "$CURRENT_SWEEP_ID" "$CURRENT_SWEEP_STRATEGY" "$CURRENT_SWEEP_MAX_PARALLEL" "$CURRENT_SWEEP_FAILURE_POLICY" "$CURRENT_SWEEP_SHARD_MODE" "$CURRENT_BATCH_ROOT" "$CURRENT_DRIVER_LOG" "$MATRIX_ID" "$E2E_MATRIX_FILE" "$MATRIX_SELECTED_PROVIDERS_CSV" "$MATRIX_SELECTED_RUN_INDEXES_CSV" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

path = Path(sys.argv[1]).resolve()
payload = {
    "profile_id": sys.argv[4],
    "profile_slug": sys.argv[5],
    "batch_id": sys.argv[6],
    "matrix_id": sys.argv[17],
    "matrix_file": sys.argv[18],
    "selected_providers": [item for item in sys.argv[19].split(",") if item],
    "selected_run_indexes": [item for item in sys.argv[20].split(",") if item],
    "source_kind": sys.argv[7],
    "expected_repo_count": int(sys.argv[8]),
    "repos_file": sys.argv[9],
    "status": sys.argv[2],
    "failure_reason": sys.argv[3],
    "sweep_id": sys.argv[10],
    "execution": {
        "strategy": sys.argv[11],
        "max_parallel_tasks": int(sys.argv[12]),
        "failure_policy": sys.argv[13],
        "shard_discovery_mode": sys.argv[14],
    },
    "batch_root": sys.argv[15],
    "run_matrix_tsv": "-",
    "run_matrix_md": "-",
    "frontend_matrix_md": "-",
    "execution_report_md": "-",
    "driver_log": sys.argv[16],
    "inventory_json": "-",
    "raw_output_refs": [],
    "updated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
}
path.write_text(json.dumps(payload, ensure_ascii=True) + "\n", encoding="utf-8")
PY
}

update_current_profile_status_artifacts() {
  local status="$1"
  local failure_reason="$2"
  local run_matrix_tsv="$3"
  local run_matrix_md="$4"
  local frontend_matrix_md="$5"
  local execution_report_md="$6"
  local inventory_json="${7:--}"
  [[ -z "$CURRENT_PROFILE_STATUS_FILE" ]] && return 0
  python3 - "$CURRENT_PROFILE_STATUS_FILE" "$status" "$failure_reason" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$execution_report_md" "$inventory_json" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

path = Path(sys.argv[1]).resolve()
payload = {}
if path.exists():
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        payload = {}
payload["status"] = sys.argv[2]
payload["failure_reason"] = sys.argv[3]
payload["run_matrix_tsv"] = sys.argv[4]
payload["run_matrix_md"] = sys.argv[5]
payload["frontend_matrix_md"] = sys.argv[6]
payload["execution_report_md"] = sys.argv[7]
payload["inventory_json"] = sys.argv[8]
if sys.argv[8] and sys.argv[8] != "-":
    try:
        inventory = json.loads(Path(sys.argv[8]).read_text(encoding="utf-8"))
        payload["raw_output_refs"] = inventory.get("raw_output_refs", [])
    except Exception:
        payload["raw_output_refs"] = []
payload["updated_at"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
path.write_text(json.dumps(payload, ensure_ascii=True) + "\n", encoding="utf-8")
PY
}

write_current_profile_inventory() {
  local status="$1"
  local failure_reason="$2"
  local run_matrix_tsv="$3"
  local run_matrix_md="$4"
  local frontend_matrix_md="$5"
  local execution_report_md="$6"
  local inventory_json="$MATRIX_ROOT/inventory/${CURRENT_BATCH_ID}.json"
  mkdir -p "$(dirname "$inventory_json")"
  python3 - "$inventory_json" "$MATRIX_ID" "$E2E_MATRIX_FILE" "$CURRENT_PROFILE_ID" "$CURRENT_SWEEP_ID" "$CURRENT_BATCH_ID" "$CURRENT_BATCH_ROOT" "$status" "$failure_reason" "$CURRENT_DRIVER_LOG" "$CURRENT_PROFILE_STATUS_FILE" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$execution_report_md" "$MATRIX_SELECTED_PROVIDERS_CSV" "$MATRIX_SELECTED_RUN_INDEXES_CSV" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

out = Path(sys.argv[1]).resolve()
batch_root = Path(sys.argv[7])
raw_output_refs = []

if batch_root.exists():
    for meta_path in sorted(batch_root.rglob("reports/taskruns/raw/*-meta.json")):
        if not meta_path.is_file():
            continue
        try:
            meta = json.loads(meta_path.read_text(encoding="utf-8"))
        except Exception as exc:
            raw_output_refs.append(
                {
                    "metadata": str(meta_path),
                    "parse_error": str(exc),
                }
            )
            continue
        task = meta.get("task") if isinstance(meta.get("task"), dict) else {}
        stdout = meta.get("stdout") if isinstance(meta.get("stdout"), dict) else {}
        stderr = meta.get("stderr") if isinstance(meta.get("stderr"), dict) else {}
        raw_output_refs.append(
            {
                "metadata": str(meta_path),
                "provider": str(meta.get("provider", "")),
                "command_family": str(meta.get("command_family", "")),
                "task_id": str(task.get("task_id", "")),
                "run_id": str(task.get("run_id", "")),
                "step_id": str(task.get("step_id", "")),
                "stdout": {
                    "relative_path": str(stdout.get("relative_path", "")),
                    "bytes": int(stdout.get("bytes", 0) or 0),
                    "stored_bytes": int(stdout.get("stored_bytes", 0) or 0),
                    "sha256": str(stdout.get("sha256", "")),
                    "truncated": bool(stdout.get("truncated", False)),
                },
                "stderr": {
                    "relative_path": str(stderr.get("relative_path", "")),
                    "bytes": int(stderr.get("bytes", 0) or 0),
                    "stored_bytes": int(stderr.get("stored_bytes", 0) or 0),
                    "sha256": str(stderr.get("sha256", "")),
                    "truncated": bool(stderr.get("truncated", False)),
                },
                "diagnostics_set": bool(meta.get("diagnostics_set", False)),
            }
        )

payload = {
    "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "matrix_id": sys.argv[2],
    "matrix_file": sys.argv[3],
    "profile_id": sys.argv[4],
    "sweep_id": sys.argv[5],
    "batch_id": sys.argv[6],
    "batch_root": sys.argv[7],
    "terminal_status": sys.argv[8],
    "failure_reason": sys.argv[9],
    "selected_providers": [item for item in sys.argv[16].split(",") if item],
    "selected_run_indexes": [item for item in sys.argv[17].split(",") if item],
    "key_paths": {
        "driver_log": sys.argv[10],
        "profile_status_file": sys.argv[11],
        "run_matrix_tsv": sys.argv[12],
        "run_matrix_md": sys.argv[13],
        "frontend_matrix_md": sys.argv[14],
        "execution_report_md": sys.argv[15],
    },
    "raw_output_refs": raw_output_refs,
}
out.write_text(json.dumps(payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
print(str(out))
PY
}

stop_current_profile_status_heartbeat() {
  if [[ -z "${CURRENT_PROFILE_STATUS_HEARTBEAT_PID:-}" ]]; then
    return 0
  fi
  kill "${CURRENT_PROFILE_STATUS_HEARTBEAT_PID}" >/dev/null 2>&1 || true
  wait "${CURRENT_PROFILE_STATUS_HEARTBEAT_PID}" >/dev/null 2>&1 || true
  CURRENT_PROFILE_STATUS_HEARTBEAT_PID=""
}

start_current_profile_status_heartbeat() {
  stop_current_profile_status_heartbeat
  [[ -z "$CURRENT_PROFILE_STATUS_FILE" ]] && return 0
  local interval="${MATRIX_PROFILE_STATUS_HEARTBEAT_SEC:-10}"
  if [[ ! "$interval" =~ ^[0-9]+$ ]] || [[ "$interval" -le 0 ]]; then
    return 0
  fi
  (
    while true; do
      sleep "$interval"
      write_current_profile_status "running" "none"
    done
  ) &
  CURRENT_PROFILE_STATUS_HEARTBEAT_PID="$!"
}

signal_number() {
  case "$1" in
    HUP) printf '1' ;;
    INT) printf '2' ;;
    TERM) printf '15' ;;
    *) printf '' ;;
  esac
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

on_matrix_signal() {
  local signal_name="$1"
  log "received termination signal: $signal_name profile=$CURRENT_PROFILE_ID sweep=$CURRENT_SWEEP_ID"
  stop_current_profile_status_heartbeat
  write_current_profile_status "failed" "infra_signal_terminated"
  exit "$(signal_exit_code "$signal_name")"
}

finalize_running_profile_statuses_on_exit() {
  local failure_reason="$1"
  [[ -d "$MATRIX_STATUS_ROOT" ]] || return 0
  python3 - "$MATRIX_STATUS_ROOT" "$failure_reason" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

root = Path(sys.argv[1]).resolve()
failure_reason = sys.argv[2]
for path in sorted(root.glob("*.json")):
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        continue
    if not isinstance(payload, dict):
        continue
    if str(payload.get("status", "")).strip() != "running":
        continue
    payload["status"] = "failed"
    payload["failure_reason"] = failure_reason
    payload["updated_at"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    path.write_text(json.dumps(payload, ensure_ascii=True) + "\n", encoding="utf-8")
PY
}

reconcile_stale_profile_statuses() {
  [[ -d "$MATRIX_STATUS_ROOT" ]] || return 0
  local changed_count
  changed_count="$(python3 - "$MATRIX_STATUS_ROOT" "$MATRIX_PROFILE_STATUS_HEARTBEAT_SEC" "$MATRIX_PROFILE_STATUS_STALE_SEC" <<'PY'
import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

root = Path(sys.argv[1]).resolve()
try:
    heartbeat_sec = int((sys.argv[2] or "").strip())
except Exception:
    heartbeat_sec = 10
try:
    stale_override = int((sys.argv[3] or "").strip())
except Exception:
    stale_override = 0
stale_sec = stale_override if stale_override > 0 else max(heartbeat_sec * 3, 30)
now = datetime.now(timezone.utc)
changed = 0


def parse_ts(value: Optional[str]) -> Optional[datetime]:
    text = str(value or "").strip()
    if not text:
        return None
    try:
        return datetime.strptime(text, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    except Exception:
        return None


def is_recent(value: Optional[datetime]) -> bool:
    if value is None:
        return False
    return (now - value).total_seconds() <= stale_sec


def pid_alive(pid_value: Optional[str]) -> bool:
    text = str(pid_value or "").strip()
    if not text:
        return False
    try:
        pid = int(text)
    except Exception:
        return False
    if pid <= 0:
        return False
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    except OSError:
        return False
    return True


def read_env(path: Path) -> dict[str, str]:
    payload: dict[str, str] = {}
    if not path.exists():
        return payload
    for line in path.read_text(encoding="utf-8").splitlines():
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        payload[key.strip()] = value.strip()
    return payload


for path in sorted(root.glob("*.json")):
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        continue
    if not isinstance(payload, dict):
        continue
    if str(payload.get("status", "")).strip() != "running":
        continue

    profile_recent = is_recent(parse_ts(payload.get("updated_at")))
    batch_root_raw = str(payload.get("batch_root", "")).strip()
    owner = read_env(Path(batch_root_raw).expanduser() / "batch-owner.env") if batch_root_raw else {}
    owner_state = str(owner.get("state", "")).strip()
    owner_recent = is_recent(parse_ts(owner.get("updated_at")))
    owner_alive = pid_alive(owner.get("pid"))

    keep_running = False
    if owner:
        keep_running = owner_state == "running" and owner_alive and owner_recent
    elif profile_recent:
        keep_running = True

    if keep_running:
        continue

    payload["status"] = "failed"
    payload["failure_reason"] = "infra_incomplete_cycle"
    payload["updated_at"] = now.strftime("%Y-%m-%dT%H:%M:%SZ")
    path.write_text(json.dumps(payload, ensure_ascii=True) + "\n", encoding="utf-8")
    changed += 1

print(changed)
PY
)"
  if [[ "$changed_count" =~ ^[1-9][0-9]*$ ]]; then
    log "reconciled stale profile statuses: count=$changed_count"
  fi
}

on_matrix_exit() {
  local exit_code="$1"
  [[ "$exit_code" =~ ^[0-9]+$ ]] || exit_code=1
  stop_current_profile_status_heartbeat
  reconcile_stale_profile_statuses
  finalize_running_profile_statuses_on_exit "infra_incomplete_cycle"
}

batch_has_incomplete_run_sentinels() {
  local batch_root="$1"
  local status_file=""
  local state=""
  if [[ ! -d "$batch_root" ]]; then
    return 1
  fi
  while IFS= read -r status_file; do
    [[ -z "$status_file" ]] && continue
    state="$(sed -n 's/^state=//p' "$status_file" | tail -n1 | tr -d '\r')"
    if [[ "$state" == "running" || -z "$state" ]]; then
      return 0
    fi
  done < <(find "$batch_root" -type f -name 'run-status.env' | LC_ALL=C sort)
  return 1
}

resolve_selected_providers() {
  local filter_raw="${BATCH_PROVIDER_FILTER:-all}"
  local filter
  filter="$(echo "$filter_raw" | tr -d '[:space:]')"
  if [[ -z "$filter" || "$filter" == "all" ]]; then
    MATRIX_SELECTED_PROVIDERS=("${MATRIX_ALL_PROVIDERS[@]}")
  else
    MATRIX_SELECTED_PROVIDERS=()
    local token
    local -a tokens=()
    IFS=',' read -r -a tokens <<<"$filter"
    for token in "${tokens[@]}"; do
      [[ -z "$token" ]] && continue
      case "$token" in
        qwen-code|claude-code|codex-code)
          if ! array_contains "$token" "${MATRIX_SELECTED_PROVIDERS[@]-}"; then
            MATRIX_SELECTED_PROVIDERS+=("$token")
          fi
          ;;
        *)
          die "BATCH_PROVIDER_FILTER contains unsupported provider '$token' (allowed: qwen-code, claude-code, codex-code, all)"
          ;;
      esac
    done
  fi
  if [[ "${#MATRIX_SELECTED_PROVIDERS[@]}" -eq 0 ]]; then
    die "BATCH_PROVIDER_FILTER resolved to an empty provider set"
  fi
  MATRIX_SELECTED_PROVIDERS_CSV="$(IFS=,; echo "${MATRIX_SELECTED_PROVIDERS[*]}")"
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
  MATRIX_SELECTED_RUN_INDEXES=()
  local run_index
  while IFS= read -r run_index; do
    [[ -z "$run_index" ]] && continue
    MATRIX_SELECTED_RUN_INDEXES+=("$run_index")
  done <<<"$resolved_indexes"
  if [[ "${#MATRIX_SELECTED_RUN_INDEXES[@]}" -eq 0 ]]; then
    die "BATCH_RUN_SELECTION resolved to an empty run set"
  fi
  MATRIX_SELECTED_RUN_INDEXES_CSV="$(IFS=,; echo "${MATRIX_SELECTED_RUN_INDEXES[*]}")"
}

read_profile_repos_meta() {
  local meta_json="$1"
  local key
  local value
  local resolved_repos_file=""
  local resolved_source_kind="mixed"
  local resolved_expected_count="0"
  while IFS='=' read -r key value; do
    case "$key" in
      repos_file) resolved_repos_file="$value" ;;
      profile_source_kind) resolved_source_kind="$value" ;;
      expected_repo_count) resolved_expected_count="$value" ;;
    esac
  done < <(acp_read_repos_meta_fields "$meta_json")

  PROFILE_REPOS_FILE_RESOLVED="$resolved_repos_file"
  PROFILE_SOURCE_KIND_EFFECTIVE="${resolved_source_kind:-mixed}"
  PROFILE_EXPECTED_REPO_COUNT_RESOLVED="${resolved_expected_count:-0}"
}

validate_profile_repos_meta() {
  local profile_id="$1"
  local repos_file="$2"
  local expected_repo_count="$3"
  local source_kind="$4"
  local output_json="$5"
  if ! python3 "$PROVENARCH_ROOT/scripts/resolve-repos-meta.py" \
    --repos-file "$repos_file" \
    --expected-repo-count "$expected_repo_count" \
    --source-kind "$source_kind" \
    --profile-id "$profile_id" \
    --out "$output_json"; then
    return 1
  fi
  read_profile_repos_meta "$output_json"
}

if [[ "${MATRIX_TEST_RECONCILE_ONLY:-0}" == "1" ]]; then
  mkdir -p "$MATRIX_ROOT" "$REPORTS_ROOT" "$MATRIX_STATUS_ROOT"
  reconcile_stale_profile_statuses
  exit 0
fi

if [[ -z "$E2E_MATRIX_FILE" ]]; then
  die "E2E_MATRIX_FILE is required (YAML with profiles[] and optional sweeps[]/timeout_profile)"
fi
if [[ ! -f "$E2E_MATRIX_FILE" ]]; then
  die "E2E_MATRIX_FILE does not exist: $E2E_MATRIX_FILE"
fi
RELEASE_MODE="$(normalize_release_mode "$E2E_MATRIX_RELEASE_MODE")"
if [[ ! "$RUN_COUNT" =~ ^[1-9][0-9]*$ ]]; then
  die "RUN_COUNT must be a positive integer, got '$RUN_COUNT'"
fi
if [[ "$RELEASE_MODE" == "1" && "$RUN_COUNT" != "1" ]]; then
  die "release-mode requires RUN_COUNT=1 for matrix mode (got '$RUN_COUNT')"
fi
if [[ "$ACP_APPLY_TIMEOUTS_VIA_API" != "0" && "$ACP_APPLY_TIMEOUTS_VIA_API" != "1" ]]; then
  die "ACP_APPLY_TIMEOUTS_VIA_API must be 0 or 1, got '$ACP_APPLY_TIMEOUTS_VIA_API'"
fi
DEFAULT_BATCH_SCRIPT="$PROVENARCH_ROOT/scripts/full-run-batch.sh"
if [[ "$RELEASE_MODE" == "1" && "$BATCH_SCRIPT" != "$DEFAULT_BATCH_SCRIPT" && "${ACP_TEST_ALLOW_BATCH_SCRIPT_OVERRIDE:-0}" != "1" ]]; then
  die "release-mode requires the canonical batch script: $DEFAULT_BATCH_SCRIPT"
fi
if [[ ! -x "$BATCH_SCRIPT" ]]; then
  die "batch script is unavailable: $BATCH_SCRIPT"
fi

require_cmd bash
require_cmd python3
acp_ensure_no_legacy_env_set die
resolve_selected_providers
resolve_selected_run_indexes
if [[ "$RELEASE_MODE" == "1" && "$MATRIX_SELECTED_PROVIDERS_CSV" != "qwen-code,claude-code,codex-code" ]]; then
  die "release-mode requires all providers with no BATCH_PROVIDER_FILTER override"
fi
if [[ "$RELEASE_MODE" == "1" && "$MATRIX_SELECTED_RUN_INDEXES_CSV" != "1" ]]; then
  die "release-mode requires selected run indexes to be exactly 1"
fi
if ! mkdir -p "$(dirname "$MATRIX_DRIVER_LOG")"; then
  die "operational_host_preflight_failed: cannot create matrix driver log directory for $MATRIX_DRIVER_LOG"
fi
if ! : > "$MATRIX_DRIVER_LOG"; then
  die "operational_host_preflight_failed: cannot write matrix driver log at $MATRIX_DRIVER_LOG"
fi
if [[ "$RELEASE_MODE" == "1" ]]; then
  require_cmd "$ACP_QWEN_CMD_BIN"
  require_cmd "$ACP_CLAUDE_CMD_BIN"
  require_cmd "$ACP_CODEX_CMD_BIN"
else
  if array_contains "qwen-code" "${MATRIX_SELECTED_PROVIDERS[@]-}"; then
    require_cmd "$ACP_QWEN_CMD_BIN"
  fi
  if array_contains "claude-code" "${MATRIX_SELECTED_PROVIDERS[@]-}"; then
    require_cmd "$ACP_CLAUDE_CMD_BIN"
  fi
  if array_contains "codex-code" "${MATRIX_SELECTED_PROVIDERS[@]-}"; then
    require_cmd "$ACP_CODEX_CMD_BIN"
  fi
fi

run_host_preflight_checks

mkdir -p "$MATRIX_ROOT" "$REPORTS_ROOT"
mkdir -p "$MATRIX_STATUS_ROOT"
reconcile_stale_profile_statuses

if [[ "$RELEASE_MODE" == "1" ]]; then
  if [[ -z "$BATCH_FRONTEND_MODE" ]]; then
    BATCH_FRONTEND_MODE="per_run"
  fi
  if [[ -z "$UI_E2E_HEADED" ]]; then
    UI_E2E_HEADED="1"
  fi
fi

DIAGNOSTIC_TIMEOUT_ENV_KEYS=("${ACP_TIMEOUT_ENV_KEYS[@]}")

DIAGNOSTIC_TIMEOUT_OVERRIDES=()
for key in "${DIAGNOSTIC_TIMEOUT_ENV_KEYS[@]}"; do
  value="${!key:-}"
  if [[ -n "$value" ]]; then
    DIAGNOSTIC_TIMEOUT_OVERRIDES+=("$key=$value")
  fi
done

TIMEOUT_PROFILE_CMD=(
  python3
  "$PROVENARCH_ROOT/scripts/resolve-timeout-profile.py"
  --format
  line
)
acp_log_release_guard log "$RELEASE_MODE" "$MATRIX_ID" "0" "${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}"
if [[ "${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}" -gt 0 ]]; then
  acp_log_diagnostic_timeout_overrides log "${DIAGNOSTIC_TIMEOUT_OVERRIDES[*]}"
fi
if [[ "$RELEASE_MODE" == "1" && "${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}" -gt 0 ]]; then
  die "$(acp_release_guard_blocked_message)"
fi
log "release frontend defaults: frontend_mode=${BATCH_FRONTEND_MODE:-default} headed=${UI_E2E_HEADED:-default}"

COMBINATIONS_TSV="$MATRIX_ROOT/profile-sweep-combinations.tsv"
RECORDS_JSONL="$MATRIX_ROOT/profile-runs.jsonl"
mkdir -p "$MATRIX_STATUS_ROOT"
: > "$RECORDS_JSONL"
: > "$MATRIX_TIMEOUT_PROFILE_FILE"
trap 'on_matrix_signal TERM' TERM
trap 'on_matrix_signal INT' INT
trap 'on_matrix_signal HUP' HUP
trap 'on_matrix_exit $?' EXIT

if ! python3 - "$E2E_MATRIX_FILE" "$COMBINATIONS_TSV" "$RELEASE_MODE" "$MATRIX_TIMEOUT_PROFILE_FILE" <<'PY'
import sys
from pathlib import Path

try:
    import yaml  # type: ignore
except Exception as exc:
    raise SystemExit(f"PyYAML is required for parsing matrix file: {exc}")

matrix_path = Path(sys.argv[1]).resolve()
out_path = Path(sys.argv[2]).resolve()
release_mode_raw = str(sys.argv[3]).strip().lower()
release_mode = release_mode_raw in {"1", "true", "yes", "on"}
timeout_profile_path = Path(sys.argv[4]).resolve()
payload = yaml.safe_load(matrix_path.read_text(encoding="utf-8"))
allowed_timeout_profiles = {"short-window", "medium-window", "extended-window"}

if isinstance(payload, dict):
    profiles = payload.get("profiles")
    sweeps = payload.get("sweeps")
    timeout_profile = str(payload.get("timeout_profile", "")).strip()
elif isinstance(payload, list):
    profiles = payload
    sweeps = None
    timeout_profile = ""
else:
    profiles = None
    sweeps = None
    timeout_profile = ""

if not isinstance(profiles, list) or not profiles:
    raise SystemExit(f"matrix file {matrix_path} must contain non-empty profiles[]")
if timeout_profile and timeout_profile not in allowed_timeout_profiles:
    raise SystemExit(
        f"matrix file {matrix_path} timeout_profile must be one of: "
        f"{', '.join(sorted(allowed_timeout_profiles))}; got: {timeout_profile}"
    )

allowed_profiles = {
    "single-path": {"source_kind": "path", "min_repos": 1, "max_repos": 1},
    "single-git_url": {"source_kind": "git_url", "min_repos": 1, "max_repos": 1},
    "multi-path": {"source_kind": "path", "min_repos": 2, "max_repos": None},
    "multi-git_url": {"source_kind": "git_url", "min_repos": 2, "max_repos": None},
}
release_profile_families = {
    "single": ("single-path", "single-git_url"),
    "multi": ("multi-path", "multi-git_url"),
}

profile_rows: list[tuple[str, str, int, str]] = []
seen_ids: set[str] = set()
for idx, item in enumerate(profiles, start=1):
    if not isinstance(item, dict):
        raise SystemExit(f"profiles[{idx}] must be an object")
    profile_id = str(item.get("id", "")).strip()
    repos_file_raw = str(item.get("repos_file", "")).strip()
    source_kind = str(item.get("source_kind", "")).strip()
    expected_raw = str(item.get("expected_repo_count", "")).strip()

    if not profile_id:
        raise SystemExit(f"profiles[{idx}] is missing id")
    if profile_id in seen_ids:
        raise SystemExit(f"duplicate profile id: {profile_id}")
    seen_ids.add(profile_id)

    if not repos_file_raw:
        raise SystemExit(f"profiles[{idx}] is missing repos_file")
    repos_file = Path(repos_file_raw)
    if not repos_file.is_absolute():
        repos_file = (matrix_path.parent / repos_file).resolve()
    else:
        repos_file = repos_file.resolve()
    if not repos_file.exists():
        raise SystemExit(f"profiles[{idx}] repos_file does not exist: {repos_file}")
    if source_kind not in {"path", "git_url"}:
        raise SystemExit(f"profiles[{idx}] source_kind must be path|git_url, got: {source_kind}")

    try:
        expected_count = int(expected_raw)
    except Exception:
        raise SystemExit(f"profiles[{idx}] expected_repo_count must be integer, got: {expected_raw}")
    if expected_count <= 0:
        raise SystemExit(f"profiles[{idx}] expected_repo_count must be > 0, got: {expected_count}")

    contract = allowed_profiles.get(profile_id)
    if contract is None:
        raise SystemExit(
            f"profiles[{idx}] id must be one of: {', '.join(sorted(allowed_profiles.keys()))}; got: {profile_id}"
        )
    expected_kind = str(contract["source_kind"])
    if source_kind != expected_kind:
        raise SystemExit(
            f"profiles[{idx}] id={profile_id} must use source_kind={expected_kind}, got: {source_kind}"
        )
    min_repos = int(contract["min_repos"])
    max_repos = contract["max_repos"]
    if expected_count < min_repos:
        raise SystemExit(
            f"profiles[{idx}] id={profile_id} expected_repo_count must be >= {min_repos}, got: {expected_count}"
        )
    if max_repos is not None and expected_count > int(max_repos):
        raise SystemExit(
            f"profiles[{idx}] id={profile_id} expected_repo_count must be <= {max_repos}, got: {expected_count}"
        )

    profile_rows.append((profile_id, str(repos_file), expected_count, source_kind))

if release_mode:
    selected_ids = {profile_id for profile_id, _, _, _ in profile_rows}
    if len(profile_rows) != 2:
        raise SystemExit(
            "release-mode requires exactly 2 profiles: one single-* and one multi-*"
        )
    for family, options in release_profile_families.items():
        matched = [profile_id for profile_id in options if profile_id in selected_ids]
        if len(matched) != 1:
            raise SystemExit(
                "release-mode requires exactly one "
                f"{family} profile from [{', '.join(options)}], got: "
                + (", ".join(matched) if matched else "none")
            )

allowed = {
    "strategy": {"sequential", "parallel"},
    "failure_policy": {"fail_fast", "best_effort"},
    "shard_discovery_mode": {"heuristics", "semantic"},
}
required_release_sweeps = ("baseline", "parallel-default")

default_sweep = {
    "id": "baseline",
    "strategy": "sequential",
    "max_parallel_tasks": 1,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "heuristics",
}

sweep_rows: list[dict[str, object]] = []
if release_mode and (not isinstance(sweeps, list) or not sweeps):
    raise SystemExit(
        "release-mode requires explicit sweeps[] with exactly these ids: baseline, parallel-default"
    )
if not isinstance(sweeps, list) or not sweeps:
    sweep_rows = [default_sweep]
else:
    seen_sweeps: set[str] = set()
    for idx, item in enumerate(sweeps, start=1):
        if not isinstance(item, dict):
            raise SystemExit(f"sweeps[{idx}] must be an object")
        sweep_id = str(item.get("id", "")).strip()
        if not sweep_id:
            raise SystemExit(f"sweeps[{idx}] is missing id")
        if sweep_id in seen_sweeps:
            raise SystemExit(f"duplicate sweep id: {sweep_id}")
        seen_sweeps.add(sweep_id)

        strategy = str(item.get("strategy", default_sweep["strategy"]))
        if strategy not in allowed["strategy"]:
            raise SystemExit(
                f"sweeps[{idx}] strategy must be one of: {', '.join(sorted(allowed['strategy']))}; got: {strategy}"
            )

        try:
            max_parallel_raw = item.get("max_parallel_tasks", default_sweep["max_parallel_tasks"])
            max_parallel_tasks = int(max_parallel_raw)
        except Exception:
            raise SystemExit(f"sweeps[{idx}] max_parallel_tasks must be integer, got: {item.get('max_parallel_tasks')}")
        if max_parallel_tasks <= 0:
            raise SystemExit(f"sweeps[{idx}] max_parallel_tasks must be > 0, got: {max_parallel_tasks}")
        if strategy != "parallel":
            max_parallel_tasks = 1

        failure_policy = str(item.get("failure_policy", default_sweep["failure_policy"]))
        if failure_policy not in allowed["failure_policy"]:
            raise SystemExit(
                "sweeps[%d] failure_policy must be one of: %s; got: %s"
                % (idx, ", ".join(sorted(allowed["failure_policy"])), failure_policy)
            )

        shard_mode = str(item.get("shard_discovery_mode", default_sweep["shard_discovery_mode"]))
        if shard_mode not in allowed["shard_discovery_mode"]:
            raise SystemExit(
                "sweeps[%d] shard_discovery_mode must be one of: %s; got: %s"
                % (idx, ", ".join(sorted(allowed["shard_discovery_mode"])), shard_mode)
            )

        sweep_rows.append(
            {
                "id": sweep_id,
                "strategy": strategy,
                "max_parallel_tasks": max_parallel_tasks,
                "failure_policy": failure_policy,
                "shard_discovery_mode": shard_mode,
            }
        )

if release_mode:
    observed_sweep_ids = {str(item["id"]).strip() for item in sweep_rows}
    required_sweep_ids = set(required_release_sweeps)
    missing_sweeps = sorted(required_sweep_ids - observed_sweep_ids)
    extra_sweeps = sorted(observed_sweep_ids - required_sweep_ids)
    if missing_sweeps:
        raise SystemExit(
            "release-mode sweeps[] is missing required ids: " + ", ".join(missing_sweeps)
        )
    if extra_sweeps:
        raise SystemExit(
            "release-mode sweeps[] contains unsupported ids: " + ", ".join(extra_sweeps)
        )
    expected_release_combinations = len(profile_rows) * len(required_release_sweeps)
    observed_release_combinations = len(profile_rows) * len(sweep_rows)
    if observed_release_combinations != expected_release_combinations:
        raise SystemExit(
            "release-mode requires full profile×sweep matrix: "
            f"expected {expected_release_combinations} combinations "
            f"(2 profiles × 2 sweeps), got {observed_release_combinations}"
        )

rows: list[str] = []
for profile_id, repos_file, expected_count, source_kind in profile_rows:
    for sweep in sweep_rows:
        rows.append(
            "\t".join(
                [
                    profile_id,
                    repos_file,
                    str(expected_count),
                    source_kind,
                    str(sweep["id"]),
                    str(sweep["strategy"]),
                    str(sweep["max_parallel_tasks"]),
                    str(sweep["failure_policy"]),
                    str(sweep["shard_discovery_mode"]),
                ]
            )
        )

out_path.parent.mkdir(parents=True, exist_ok=True)
out_path.write_text("\n".join(rows) + "\n", encoding="utf-8")
timeout_profile_path.parent.mkdir(parents=True, exist_ok=True)
timeout_profile_path.write_text((timeout_profile + "\n") if timeout_profile else "", encoding="utf-8")
PY
then
  die "matrix planning failed: matrix_file=$E2E_MATRIX_FILE"
fi

if [[ -f "$MATRIX_TIMEOUT_PROFILE_FILE" ]]; then
  MATRIX_TIMEOUT_PROFILE="$(tr -d '\r' < "$MATRIX_TIMEOUT_PROFILE_FILE" | head -n1 | xargs)"
fi
if [[ -n "$MATRIX_TIMEOUT_PROFILE" ]]; then
  TIMEOUT_PROFILE_CMD+=(--preset "$MATRIX_TIMEOUT_PROFILE")
  while IFS='=' read -r key value; do
    [[ -z "$key" || -n "${!key:-}" ]] && continue
    MATRIX_TIMEOUT_ENV_ASSIGNMENTS+=("$key=$value")
  done < <(python3 "$PROVENARCH_ROOT/scripts/resolve-timeout-profile.py" --preset "$MATRIX_TIMEOUT_PROFILE" --format env-kv)
  log "matrix timeout profile: profile=$MATRIX_TIMEOUT_PROFILE"
fi
TIMEOUT_PROFILE_LINE="$("${TIMEOUT_PROFILE_CMD[@]}")"
acp_log_preflight_timeout log "$ACP_APPLY_TIMEOUTS_VIA_API" "$TIMEOUT_PROFILE_LINE"

while IFS=$'\t' read -r profile_id repos_file expected_repo_count source_kind sweep_id sweep_strategy sweep_max_parallel sweep_failure_policy sweep_shard_mode <&3; do
  [[ -z "$profile_id" ]] && continue

  profile_slug="$(slugify "$profile_id")"
  sweep_slug="$(slugify "$sweep_id")"
  batch_id="${MATRIX_ID}-${profile_slug}-${sweep_slug}"
  batch_root="$E2E_TMP_ROOT/runs/$batch_id"
  profile_base_root="$MATRIX_ROOT/profiles/$profile_slug"
  profile_root="$profile_base_root/$sweep_slug"
  profile_repos_meta_json="$profile_base_root/target-repos-meta.json"
  driver_log="$profile_root/driver.log"
  mkdir -p "$profile_root"
  CURRENT_PROFILE_ID="$profile_id"
  CURRENT_PROFILE_SLUG="$profile_slug"
  CURRENT_SWEEP_ID="$sweep_id"
  CURRENT_BATCH_ID="$batch_id"
  CURRENT_BATCH_ROOT="$batch_root"
  CURRENT_DRIVER_LOG="$driver_log"
  CURRENT_SWEEP_STRATEGY="$sweep_strategy"
  CURRENT_SWEEP_MAX_PARALLEL="$sweep_max_parallel"
  CURRENT_SWEEP_FAILURE_POLICY="$sweep_failure_policy"
  CURRENT_SWEEP_SHARD_MODE="$sweep_shard_mode"
  CURRENT_PROFILE_STATUS_FILE="$MATRIX_STATUS_ROOT/${profile_slug}--${sweep_slug}.json"
  CURRENT_SOURCE_KIND="$source_kind"
  CURRENT_EXPECTED_REPO_COUNT="$expected_repo_count"
  CURRENT_REPOS_FILE="$repos_file"

  profile_meta_key="${profile_id}|${repos_file}|${expected_repo_count}|${source_kind}"
  if [[ "$profile_meta_key" != "$PROFILE_META_CACHE_KEY" ]]; then
    if ! validate_profile_repos_meta "$profile_id" "$repos_file" "$expected_repo_count" "$source_kind" "$profile_repos_meta_json" >>"$driver_log" 2>&1; then
      status="failed"
      profile_failure_reason="operational_host_preflight_failed"
      log "profile repos preflight failed: profile=$profile_id sweep=$sweep_id repos_file=$repos_file (see $driver_log)"
      write_current_profile_status "$status" "$profile_failure_reason"

      run_matrix_tsv="$REPORTS_ROOT/run_matrix_${batch_id}.tsv"
      run_matrix_md="$REPORTS_ROOT/run_matrix_${batch_id}.md"
      frontend_matrix_md="$REPORTS_ROOT/frontend_e2e_matrix_${batch_id}.md"
      execution_report_md="$REPORTS_ROOT/execution_report_${batch_id}.md"
      inventory_json="$(write_current_profile_inventory "$status" "$profile_failure_reason" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$execution_report_md")"
      update_current_profile_status_artifacts "$status" "$profile_failure_reason" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$execution_report_md" "$inventory_json"

      python3 - "$RECORDS_JSONL" \
        "$profile_id" "$profile_slug" "$batch_id" "$source_kind" "$expected_repo_count" "$repos_file" "$status" "$profile_failure_reason" \
        "$sweep_id" "$sweep_strategy" "$sweep_max_parallel" "$sweep_failure_policy" "$sweep_shard_mode" \
        "$batch_root" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$execution_report_md" "$driver_log" "$inventory_json" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1]).resolve()
payload = {
    "profile_id": sys.argv[2],
    "profile_slug": sys.argv[3],
    "batch_id": sys.argv[4],
    "source_kind": sys.argv[5],
    "expected_repo_count": int(sys.argv[6]),
    "repos_file": sys.argv[7],
    "status": sys.argv[8],
    "failure_reason": sys.argv[9],
    "sweep_id": sys.argv[10],
    "execution": {
        "strategy": sys.argv[11],
        "max_parallel_tasks": int(sys.argv[12]),
        "failure_policy": sys.argv[13],
        "shard_discovery_mode": sys.argv[14],
    },
    "batch_root": sys.argv[15],
    "run_matrix_tsv": sys.argv[16],
    "run_matrix_md": sys.argv[17],
    "frontend_matrix_md": sys.argv[18],
    "execution_report_md": sys.argv[19],
    "driver_log": sys.argv[20],
    "inventory_json": sys.argv[21],
}
with path.open("a", encoding="utf-8") as f:
    f.write(json.dumps(payload, ensure_ascii=True))
    f.write("\n")
PY
      continue
    fi
    PROFILE_META_CACHE_KEY="$profile_meta_key"
    PROFILE_META_CACHE_REPOS_FILE="$PROFILE_REPOS_FILE_RESOLVED"
    PROFILE_META_CACHE_SOURCE_KIND="$PROFILE_SOURCE_KIND_EFFECTIVE"
    PROFILE_META_CACHE_EXPECTED_REPO_COUNT="$PROFILE_EXPECTED_REPO_COUNT_RESOLVED"
    log "profile repos preflight: profile=$profile_id repos_file=$PROFILE_META_CACHE_REPOS_FILE source_kind=$PROFILE_META_CACHE_SOURCE_KIND expected_repo_count=$PROFILE_META_CACHE_EXPECTED_REPO_COUNT"
  fi
  CURRENT_SOURCE_KIND="$PROFILE_META_CACHE_SOURCE_KIND"
  CURRENT_EXPECTED_REPO_COUNT="$PROFILE_META_CACHE_EXPECTED_REPO_COUNT"
  CURRENT_REPOS_FILE="$PROFILE_META_CACHE_REPOS_FILE"
  write_current_profile_status "running" "none"
  start_current_profile_status_heartbeat
  log "running profile=$profile_id sweep=$sweep_id source_kind=$PROFILE_META_CACHE_SOURCE_KIND expected_repo_count=$PROFILE_META_CACHE_EXPECTED_REPO_COUNT batch_id=$batch_id"
  status="passed"
  if ! (
    cd "$PROVENARCH_ROOT"
    if [[ "${#MATRIX_TIMEOUT_ENV_ASSIGNMENTS[@]}" -gt 0 ]]; then
      for assignment in "${MATRIX_TIMEOUT_ENV_ASSIGNMENTS[@]}"; do
        export "${assignment?}"
      done
    fi
    acp_build_timeout_env_assignments
    ACP_EXECUTION_STRATEGY="$sweep_strategy"
    ACP_MAX_PARALLEL_TASKS="$sweep_max_parallel"
    ACP_FAILURE_POLICY="$sweep_failure_policy"
    ACP_SHARD_DISCOVERY_MODE="$sweep_shard_mode"
    acp_build_execution_env_assignments
    EXTRA_BATCH_ENV_ASSIGNMENTS=()
    if [[ -n "$BATCH_FRONTEND_MODE" ]]; then
      EXTRA_BATCH_ENV_ASSIGNMENTS+=("BATCH_FRONTEND_MODE=$BATCH_FRONTEND_MODE")
    fi
    if [[ -n "$UI_E2E_HEADED" ]]; then
      EXTRA_BATCH_ENV_ASSIGNMENTS+=("UI_E2E_HEADED=$UI_E2E_HEADED")
    fi
    BATCH_ENV_ASSIGNMENTS=(
      "${ACP_TIMEOUT_ENV_ASSIGNMENTS[@]}"
      "${ACP_EXECUTION_ENV_ASSIGNMENTS[@]}"
    )
    if [[ "${#EXTRA_BATCH_ENV_ASSIGNMENTS[@]}" -gt 0 ]]; then
      BATCH_ENV_ASSIGNMENTS+=("${EXTRA_BATCH_ENV_ASSIGNMENTS[@]}")
    fi
    env \
      "${BATCH_ENV_ASSIGNMENTS[@]}" \
      "BATCH_ID=$batch_id" \
      "BATCH_ROOT=$batch_root" \
      "RUN_COUNT=$RUN_COUNT" \
      "TARGET_REPOS_FILE=$PROFILE_META_CACHE_REPOS_FILE" \
      "PROFILE_ID=$profile_id" \
      "PROFILE_SOURCE_KIND=$PROFILE_META_CACHE_SOURCE_KIND" \
      "EXPECTED_REPO_COUNT=$PROFILE_META_CACHE_EXPECTED_REPO_COUNT" \
      "SWEEP_ID=$sweep_id" \
      "E2E_TMP_ROOT=$E2E_TMP_ROOT" \
      "REPORTS_ROOT=$REPORTS_ROOT" \
      "ACP_CLAUDE_CMD_BIN=$ACP_CLAUDE_CMD_BIN" \
      "ACP_QWEN_CMD_BIN=$ACP_QWEN_CMD_BIN" \
      "ACP_CODEX_CMD_BIN=$ACP_CODEX_CMD_BIN" \
      "ACP_CODEX_MODEL=$ACP_CODEX_MODEL" \
      "ACP_CODEX_REASONING_EFFORT=$ACP_CODEX_REASONING_EFFORT" \
      "ACP_APPLY_TIMEOUTS_VIA_API=$ACP_APPLY_TIMEOUTS_VIA_API" \
      "$BATCH_SCRIPT" < /dev/null
  ) >"$driver_log" 2>&1; then
    status="failed"
    log "profile+sweep failed: profile=$profile_id sweep=$sweep_id (see $driver_log)"
  fi
  stop_current_profile_status_heartbeat
  profile_failure_reason="none"
  if [[ "$status" != "passed" ]]; then
    profile_failure_reason="child_failed"
    if [[ -f "$driver_log" ]] && grep -q "operational_host_preflight_failed" "$driver_log"; then
      profile_failure_reason="operational_host_preflight_failed"
    fi
  fi
  if batch_has_incomplete_run_sentinels "$batch_root"; then
    status="failed"
    profile_failure_reason="infra_incomplete_cycle"
    log "profile+sweep left unfinished run sentinel: profile=$profile_id sweep=$sweep_id batch_root=$batch_root"
  fi

  run_matrix_tsv="$REPORTS_ROOT/run_matrix_${batch_id}.tsv"
  run_matrix_md="$REPORTS_ROOT/run_matrix_${batch_id}.md"
  frontend_matrix_md="$REPORTS_ROOT/frontend_e2e_matrix_${batch_id}.md"
  execution_report_md="$REPORTS_ROOT/execution_report_${batch_id}.md"
  inventory_json="$(write_current_profile_inventory "$status" "$profile_failure_reason" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$execution_report_md")"
  update_current_profile_status_artifacts "$status" "$profile_failure_reason" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$execution_report_md" "$inventory_json"

  python3 - "$RECORDS_JSONL" \
    "$profile_id" "$profile_slug" "$batch_id" "$PROFILE_META_CACHE_SOURCE_KIND" "$PROFILE_META_CACHE_EXPECTED_REPO_COUNT" "$PROFILE_META_CACHE_REPOS_FILE" "$status" "$profile_failure_reason" \
    "$sweep_id" "$sweep_strategy" "$sweep_max_parallel" "$sweep_failure_policy" "$sweep_shard_mode" \
    "$batch_root" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$execution_report_md" "$driver_log" "$inventory_json" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1]).resolve()
payload = {
    "profile_id": sys.argv[2],
    "profile_slug": sys.argv[3],
    "batch_id": sys.argv[4],
    "source_kind": sys.argv[5],
    "expected_repo_count": int(sys.argv[6]),
    "repos_file": sys.argv[7],
    "status": sys.argv[8],
    "failure_reason": sys.argv[9],
    "sweep_id": sys.argv[10],
    "execution": {
        "strategy": sys.argv[11],
        "max_parallel_tasks": int(sys.argv[12]),
        "failure_policy": sys.argv[13],
        "shard_discovery_mode": sys.argv[14],
    },
    "batch_root": sys.argv[15],
    "run_matrix_tsv": sys.argv[16],
    "run_matrix_md": sys.argv[17],
    "frontend_matrix_md": sys.argv[18],
    "execution_report_md": sys.argv[19],
    "driver_log": sys.argv[20],
    "inventory_json": sys.argv[21],
}
with path.open("a", encoding="utf-8") as f:
    f.write(json.dumps(payload, ensure_ascii=True))
    f.write("\n")
PY
done 3< "$COMBINATIONS_TSV"
CURRENT_PROFILE_STATUS_FILE=""

reconcile_stale_profile_statuses

MATRIX_REPORT_MD="$REPORTS_ROOT/profile_matrix_${MATRIX_ID}.md"
MATRIX_REPORT_TSV="$REPORTS_ROOT/profile_matrix_${MATRIX_ID}.tsv"
if [[ "$RELEASE_MODE" == "1" ]]; then
  RESULT_MD="$REPORTS_ROOT/release_verdict_${MATRIX_ID}.md"
  RESULT_JSON="$REPORTS_ROOT/release_verdict_${MATRIX_ID}.json"
else
  RESULT_MD="$REPORTS_ROOT/matrix_result_${MATRIX_ID}.md"
  RESULT_JSON="$REPORTS_ROOT/matrix_result_${MATRIX_ID}.json"
fi
if [[ "${MATRIX_TEST_TRUNCATE_RECORDS_JSONL:-0}" == "1" ]]; then
  : > "$RECORDS_JSONL"
fi

python3 - "$RECORDS_JSONL" "$MATRIX_STATUS_ROOT" "$MATRIX_REPORT_MD" "$MATRIX_REPORT_TSV" "$RESULT_MD" "$RESULT_JSON" "$MATRIX_ID" "$RELEASE_MODE" "$RUN_COUNT" "$MATRIX_SELECTED_PROVIDERS_CSV" "$MATRIX_SELECTED_RUN_INDEXES_CSV" <<'PY'
import json
import os
import re
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

records_path = Path(sys.argv[1]).resolve()
status_root = Path(sys.argv[2]).resolve()
out_md = Path(sys.argv[3]).resolve()
out_tsv = Path(sys.argv[4]).resolve()
verdict_md = Path(sys.argv[5]).resolve()
verdict_json = Path(sys.argv[6]).resolve()
matrix_id = sys.argv[7]
release_mode = str(sys.argv[8]).strip() == "1"
run_count = int(sys.argv[9])
selected_providers = [item.strip() for item in str(sys.argv[10]).split(",") if item.strip()]
selected_run_indexes = [item.strip() for item in str(sys.argv[11]).split(",") if item.strip()]


def record_key(payload: dict[str, object]) -> tuple[str, str]:
    return (str(payload.get("profile_id", "")).strip(), str(payload.get("sweep_id", "")).strip())


records_by_key: dict[tuple[str, str], dict[str, object]] = {}
if records_path.exists():
    for line in records_path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line:
            continue
        payload = json.loads(line)
        if not isinstance(payload, dict):
            continue
        records_by_key[record_key(payload)] = payload

if status_root.exists():
    for path in sorted(status_root.glob("*.json")):
        try:
            payload = json.loads(path.read_text(encoding="utf-8"))
        except Exception:
            continue
        if not isinstance(payload, dict):
            continue
        key = record_key(payload)
        existing = records_by_key.get(key)
        if existing is None:
            records_by_key[key] = payload
            continue
        for field in (
            "batch_id",
            "source_kind",
            "expected_repo_count",
            "repos_file",
            "status",
            "failure_reason",
            "batch_root",
            "run_matrix_tsv",
            "run_matrix_md",
            "frontend_matrix_md",
            "execution_report_md",
            "driver_log",
            "inventory_json",
        ):
            value = payload.get(field)
            if field == "status" and str(value).strip():
                existing[field] = value
                continue
            if existing.get(field):
                continue
            existing[field] = value
        if isinstance(payload.get("execution"), dict) and not isinstance(existing.get("execution"), dict):
            existing["execution"] = payload.get("execution")
        if isinstance(payload.get("raw_output_refs"), list) and not isinstance(existing.get("raw_output_refs"), list):
            existing["raw_output_refs"] = payload.get("raw_output_refs")

records = list(records_by_key.values())

if not records:
    raise SystemExit(f"no matrix records found in {records_path} or {status_root}")

for record in records:
    if str(record.get("status", "")).strip() == "running":
        record["status"] = "failed"
        if not str(record.get("failure_reason", "")).strip():
            record["failure_reason"] = "infra_incomplete_cycle"

required_release_sweeps = ("baseline", "parallel-default")
release_profile_order = ("single-path", "single-git_url", "multi-path", "multi-git_url")
required_release_providers = ("qwen-code", "claude-code", "codex-code")


def parse_frontend_row(path: Path, provider: str) -> dict[str, str]:
    if not path.exists() or not path.is_file():
        return {"status": "missing", "reasons": ""}
    for line in path.read_text(encoding="utf-8").splitlines():
        if not line.startswith("|"):
            continue
        cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
        if len(cells) < 2 or cells[0] != provider:
            continue
        return {
            "status": cells[1] if cells[1] else "missing",
            "reasons": cells[3] if len(cells) > 3 else "",
        }
    return {"status": "missing", "reasons": ""}

def parse_backend_stats(tsv_path: Path) -> dict[str, object]:
    stats: dict[str, object] = {
        "hard": 0,
        "total": 0,
        "runtime_contract_failed": 0,
        "runner_unavailable": 0,
        "runtime_timeout": 0,
        "infra_signal_terminated": 0,
        "infra_incomplete_cycle": 0,
        "quality_gates_failed": 0,
        "artifact_quality_failed": 0,
        "summary_missing": 0,
        "precheck_failed": 0,
        "cancellation_like": 0,
        "runtime_flow_failed": 0,
        "semantic_hard_fail": 0,
        "off_topic_hits": 0,
        "artifact_non_snapshot": 0,
        "evidence_scope_hits": 0,
        "cross_repo_missing_hits": 0,
        "runtime_flow_issue_hits": 0,
        "repair_attempts": 0,
        "repair_exhausted": 0,
        "fresh_retries": 0,
        "focused_repairs": 0,
        "stall_count": 0,
        "pre_artifact_stalls": 0,
        "post_artifact_stalls": 0,
        "valid_artifact_controlled_stops": 0,
        "zero_output_pre_artifact_stalls": 0,
        "partial_failure_count": 0,
        "quality_alerts": 0,
        "issues_counter": Counter(),
        "excellent_blockers_counter": Counter(),
        "excellent_blockers_by_step": [],
        "provider_total": Counter(),
        "provider_hard": Counter(),
    }

    if not tsv_path.exists() or not tsv_path.is_file():
        return stats
    lines = [line for line in tsv_path.read_text(encoding="utf-8").splitlines() if line.strip()]
    if len(lines) <= 1:
        return stats

    header = lines[0].split("\t")
    index = {name: idx for idx, name in enumerate(header)}

    def field(parts: list[str], name: str, default: str = "") -> str:
        idx = index.get(name)
        if idx is None or idx >= len(parts):
            return default
        return parts[idx].strip()

    def field_bool(parts: list[str], name: str) -> bool:
        return field(parts, name) in {"1", "true", "True", "yes", "YES"}

    def field_int(parts: list[str], name: str, default: int = 0) -> int:
        raw = field(parts, name, str(default))
        try:
            return int(raw)
        except Exception:
            return default

    for line in lines[1:]:
        parts = line.split("\t")
        stats["total"] = int(stats["total"]) + 1
        provider = field(parts, "provider", "")
        hard_pass = field_bool(parts, "hard_pass")
        artifact_quality_failed_field = field_bool(parts, "artifact_quality_failed")
        quality_gates_failed_field = field_bool(parts, "quality_gates_failed")
        if provider:
            provider_total: Counter = stats["provider_total"]  # type: ignore[assignment]
            provider_total.update([provider])
        if hard_pass:
            stats["hard"] = int(stats["hard"]) + 1
            if provider:
                provider_hard: Counter = stats["provider_hard"]  # type: ignore[assignment]
                provider_hard.update([provider])

        for key in (
            "runtime_contract_failed",
            "runner_unavailable",
            "runtime_timeout",
            "infra_signal_terminated",
            "infra_incomplete_cycle",
            "quality_gates_failed",
            "artifact_quality_failed",
            "summary_missing",
            "precheck_failed",
            "cancellation_like",
            "runtime_flow_failed",
            "semantic_hard_fail",
        ):
            if field_bool(parts, key):
                stats[key] = int(stats[key]) + 1

        stats["off_topic_hits"] = int(stats["off_topic_hits"]) + field_int(parts, "off_topic_hits", 0)
        for key in (
            "repair_attempts",
            "repair_exhausted",
            "fresh_retries",
            "focused_repairs",
            "stall_count",
            "pre_artifact_stalls",
            "post_artifact_stalls",
            "valid_artifact_controlled_stops",
            "zero_output_pre_artifact_stalls",
            "partial_failure_count",
            "quality_alerts",
        ):
            stats[key] = int(stats[key]) + field_int(parts, key, 0)

        artifact_source = field(parts, "artifact_source", "")
        if artifact_source != "snapshot":
            stats["artifact_non_snapshot"] = int(stats["artifact_non_snapshot"]) + 1

        issues_raw = field(parts, "issues", "")
        issue_tags = [tag.strip() for tag in issues_raw.split(",") if tag.strip() and tag.strip() != "-"]
        excellent_raw = field(parts, "excellent_blockers", "")
        excellent_tags = [tag.strip() for tag in excellent_raw.split(",") if tag.strip() and tag.strip() != "-"]
        if excellent_tags:
            excellent_counter: Counter = stats["excellent_blockers_counter"]  # type: ignore[assignment]
            excellent_counter.update(excellent_tags)
        excellent_steps_raw = field(parts, "excellent_blockers_by_step", "")
        if excellent_steps_raw:
            try:
                decoded_steps = json.loads(excellent_steps_raw)
            except Exception:
                decoded_steps = []
            if isinstance(decoded_steps, list):
                step_items: list = stats["excellent_blockers_by_step"]  # type: ignore[assignment]
                for step_item in decoded_steps:
                    if isinstance(step_item, dict):
                        step_items.append(step_item)
        runtime_flow_hit = False
        for tag in issue_tags:
            counter: Counter = stats["issues_counter"]  # type: ignore[assignment]
            counter.update([tag])
            if tag == "analysis:evidence-scope":
                stats["evidence_scope_hits"] = int(stats["evidence_scope_hits"]) + 1
            if tag == "analysis:cross-repo-missing":
                stats["cross_repo_missing_hits"] = int(stats["cross_repo_missing_hits"]) + 1
            if tag == "quality:artifact-quality" and not artifact_quality_failed_field:
                stats["artifact_quality_failed"] = int(stats["artifact_quality_failed"]) + 1
                if not quality_gates_failed_field:
                    stats["quality_gates_failed"] = int(stats["quality_gates_failed"]) + 1
            if tag.startswith("runtime:"):
                runtime_flow_hit = True
        if runtime_flow_hit:
            stats["runtime_flow_issue_hits"] = int(stats["runtime_flow_issue_hits"]) + 1

    return stats


def excellent_blockers_from_stats(stats: dict[str, object]) -> list[str]:
    counter = Counter()
    explicit = stats.get("excellent_blockers_counter")
    if isinstance(explicit, Counter):
        counter.update(explicit)
    issues = stats.get("issues_counter")
    if isinstance(issues, Counter):
        if int(stats.get("artifact_quality_failed", 0)) > 0 and not counter.get("artifact-quality blockers"):
            counter["artifact-quality blockers"] = int(stats.get("artifact_quality_failed", 0))
        issue_map = {
            "execution:repair-heavy": "runtime_quality.repair_heavy",
            "execution:repair-exhausted": "runtime_quality.repair_exhausted",
            "execution:stall-pressure": "runtime_quality.stall_pressure",
            "execution:partial-failures": "runtime_quality.partial_failures",
        }
        for issue, label in issue_map.items():
            if issues.get(issue, 0) and not counter.get(label):
                counter[label] = int(issues.get(issue, 0))
        for issue, count in issues.items():
            if str(issue).startswith("analysis:") and not counter.get(str(issue)):
                counter[str(issue)] = int(count)
    return [f"{name}={count}" for name, count in sorted(counter.items())]


def normalize_scope_list(values: object) -> list[str]:
    if not isinstance(values, list):
        return []
    normalized = []
    for value in values:
        text = str(value).strip()
        if text:
            normalized.append(text)
    return sorted(set(normalized))


def collect_batch_shard_plan_signature(batch_root: Path) -> tuple[Optional[str], list[str]]:
    if not batch_root.exists():
        return None, [f"shard_plan_artifacts=missing_batch_root:{batch_root}"]

    entries: list[dict[str, object]] = []
    for plan_path in sorted(batch_root.rglob("*-shard-plan*.json")):
        if not plan_path.is_file():
            continue
        try:
            payload = json.loads(plan_path.read_text(encoding="utf-8"))
        except Exception as exc:
            return None, [f"shard_plan_artifacts=invalid_json:{plan_path} ({exc})"]

        rel_parts = plan_path.relative_to(batch_root).parts
        provider = rel_parts[0] if len(rel_parts) >= 1 else "-"
        run_slot = rel_parts[1] if len(rel_parts) >= 2 else "-"
        match = re.search(r"-(init|refresh)-", plan_path.name)
        pipeline = match.group(1) if match else "-"

        items = payload.get("items")
        normalized_items: list[dict[str, object]] = []
        if isinstance(items, list):
            for item in items:
                if not isinstance(item, dict):
                    continue
                normalized_items.append(
                    {
                        "shard_id": str(item.get("shard_id", "")).strip(),
                        "repo_scopes": normalize_scope_list(item.get("repo_scopes")),
                        "path_scopes": normalize_scope_list(item.get("path_scopes")),
                    }
                )
        normalized_items.sort(
            key=lambda item: (
                tuple(item["repo_scopes"]),
                tuple(item["path_scopes"]),
                str(item["shard_id"]),
            )
        )
        entries.append(
            {
                "slot": f"{provider}/{run_slot}/{pipeline}",
                "items": normalized_items,
            }
        )

    if not entries:
        return None, [f"shard_plan_artifacts=missing:{batch_root}"]

    entries.sort(key=lambda item: str(item["slot"]))
    return json.dumps(entries, ensure_ascii=True, sort_keys=True), []


def strict_blockers(
    rec: dict[str, object],
    stats: dict[str, object],
    frontend: dict[str, str],
    frontend_reasons: dict[str, str],
    shard_plan_invariant: str,
    release_mode: bool,
    expected_backend_runs: int,
    expected_provider_runs: int,
    required_frontend_providers: list[str],
    frontend_init_required: bool,
) -> list[str]:
    reasons: list[str] = []

    if str(rec.get("status", "")).strip() != "passed":
        reasons.append(f"batch_status={rec.get('status', 'missing')}")

    if int(stats["total"]) != expected_backend_runs:
        reasons.append(f"backend_total_runs={stats['total']} (expected {expected_backend_runs})")
    if int(stats["hard"]) != expected_backend_runs:
        reasons.append(f"backend_hard_pass={stats['hard']} (expected {expected_backend_runs})")

    for key in (
        "runtime_contract_failed",
        "runner_unavailable",
        "runtime_timeout",
        "infra_signal_terminated",
        "infra_incomplete_cycle",
        "quality_gates_failed",
        "artifact_quality_failed",
        "summary_missing",
        "precheck_failed",
    ):
        if int(stats[key]) != 0:
            reasons.append(f"{key}={stats[key]} (expected 0)")

    if int(stats["artifact_non_snapshot"]) != 0:
        reasons.append(f"artifact_source_non_snapshot={stats['artifact_non_snapshot']} (expected 0)")
    if int(stats["partial_failure_count"]) != 0:
        reasons.append(f"partial_failure_count={stats['partial_failure_count']} (expected 0)")

    runtime_flow_violations = int(stats["runtime_flow_issue_hits"]) + int(stats["runtime_flow_failed"])
    if runtime_flow_violations != 0:
        reasons.append(
            "runtime_flow_violations="
            f"issue_hits:{stats['runtime_flow_issue_hits']} runtime_flow_failed:{stats['runtime_flow_failed']} (expected 0)"
        )

    frontend_provider_keys = {
        "qwen-code": "frontend_qwen_status",
        "claude-code": "frontend_claude_status",
        "codex-code": "frontend_codex_status",
    }

    def provider_backend_passed(provider: str) -> bool:
        provider_total = stats.get("provider_total")
        provider_hard = stats.get("provider_hard")
        if isinstance(provider_total, Counter) and provider_total:
            return (
                provider_total.get(provider, 0) == expected_provider_runs
                and isinstance(provider_hard, Counter)
                and provider_hard.get(provider, 0) == expected_provider_runs
            )
        return int(stats["total"]) == expected_backend_runs and int(stats["hard"]) == expected_backend_runs

    def frontend_depends_on_backend_failure(provider: str, key: str) -> bool:
        status = frontend.get(key, "missing")
        if status not in {"skipped", "missing"}:
            return False
        if "snapshot_reports_missing" not in frontend_reasons.get(key, ""):
            return False
        return not provider_backend_passed(provider)

    for provider in required_frontend_providers:
        init_key = frontend_provider_keys[provider]
        if frontend_init_required and frontend.get(init_key) != "passed" and not frontend_depends_on_backend_failure(provider, init_key):
            reasons.append(f"{init_key}={frontend.get(init_key, 'missing')} (expected passed)")
    if release_mode and shard_plan_invariant != "passed":
        reasons.append(f"shard_plan_invariant={shard_plan_invariant} (release requires passed)")

    return reasons


def collect_raw_output_refs(rec: dict[str, object]) -> list[object]:
    refs = rec.get("raw_output_refs")
    if isinstance(refs, list):
        return refs
    inventory_path = Path(str(rec.get("inventory_json", "")))
    if inventory_path.exists() and inventory_path.is_file():
        try:
            payload = json.loads(inventory_path.read_text(encoding="utf-8"))
        except Exception:
            return []
        refs = payload.get("raw_output_refs")
        if isinstance(refs, list):
            return refs
    return []


def shard_plan_invariant_status(records: list[dict[str, object]]) -> tuple[dict[str, str], dict[str, list[str]]]:
    status_by_batch: dict[str, str] = {}
    blockers_by_batch: dict[str, list[str]] = {}
    signatures_by_batch: dict[str, Optional[str]] = {}

    for rec in records:
        batch_id = str(rec.get("batch_id", ""))
        batch_root = Path(str(rec.get("batch_root", "")))
        signature, artifact_blockers = collect_batch_shard_plan_signature(batch_root)
        signatures_by_batch[batch_id] = signature
        if artifact_blockers:
            status_by_batch[batch_id] = "artifact_error"
            blockers_by_batch[batch_id] = list(artifact_blockers)
        else:
            status_by_batch[batch_id] = "not_compared"
            blockers_by_batch[batch_id] = []

    records_by_profile: dict[str, dict[str, dict[str, object]]] = {}
    for rec in records:
        profile_id = str(rec.get("profile_id", ""))
        sweep_id = str(rec.get("sweep_id", "baseline"))
        records_by_profile.setdefault(profile_id, {})[sweep_id] = rec

    for profile_id, sweeps in records_by_profile.items():
        baseline = sweeps.get("baseline")
        parallel_default = sweeps.get("parallel-default")
        if baseline is None or parallel_default is None:
            continue

        baseline_batch = str(baseline.get("batch_id", ""))
        parallel_batch = str(parallel_default.get("batch_id", ""))
        baseline_blockers = blockers_by_batch.get(baseline_batch, [])
        parallel_blockers = blockers_by_batch.get(parallel_batch, [])
        if baseline_blockers or parallel_blockers:
            continue

        if signatures_by_batch.get(baseline_batch) == signatures_by_batch.get(parallel_batch):
            status_by_batch[baseline_batch] = "passed"
            status_by_batch[parallel_batch] = "passed"
            continue

        reason = (
            "shard_plan_invariant=baseline_vs_parallel_default_mismatch"
            f" (profile={profile_id}, baseline={baseline_batch}, parallel_default={parallel_batch})"
        )
        status_by_batch[baseline_batch] = "failed"
        status_by_batch[parallel_batch] = "failed"
        blockers_by_batch.setdefault(baseline_batch, []).append(reason)
        blockers_by_batch.setdefault(parallel_batch, []).append(reason)

    return status_by_batch, blockers_by_batch


header = [
    "profile_id",
    "sweep_id",
    "batch_id",
    "source_kind",
    "expected_repo_count",
    "execution_strategy",
    "execution_max_parallel_tasks",
    "execution_failure_policy",
    "execution_shard_discovery_mode",
    "shard_plan_invariant",
    "status",
    "strict_status",
    "backend_hard_pass",
    "backend_total_runs",
    "semantic_hard_fail_runs",
    "off_topic_hits",
    "artifact_non_snapshot_runs",
    "runtime_contract_failed_failures",
    "runner_unavailable_failures",
    "runtime_timeout_failures",
    "infra_signal_terminated_failures",
    "infra_incomplete_cycle_failures",
    "quality_gates_failed_failures",
    "artifact_quality_failed_failures",
    "summary_missing_failures",
    "precheck_failed_failures",
    "runtime_flow_failed_runs",
    "evidence_scope_hits",
    "cross_repo_missing_hits",
    "runtime_flow_issue_hits",
    "repair_attempts",
    "repair_exhausted",
    "fresh_retries",
    "focused_repairs",
    "stall_count",
    "pre_artifact_stalls",
    "post_artifact_stalls",
    "valid_artifact_controlled_stops",
    "zero_output_pre_artifact_stalls",
    "partial_failure_count",
    "quality_alerts",
    "frontend_qwen_status",
    "frontend_claude_status",
    "frontend_codex_status",
    "blocking_reasons",
    "raw_output_ref_count",
    "run_matrix_tsv",
    "execution_report_md",
    "inventory_json",
]

tsv_lines = ["\t".join(header)]
md_lines = [
    "# Profile Matrix",
    "",
    "| profile_id | sweep_id | batch_id | status | strict | shard_plan_invariant | backend_hard/total | semantic_hard_fail | off_topic_hits | artifact_non_snapshot | artifact_quality | evidence_scope | cross_repo_missing | runtime_flow | repair/stall/partial | frontend init (qwen/claude/codex) | blockers | run_matrix | execution_report |",
    "|---|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---|---|---|",
]

invariant_status_by_batch, invariant_blockers_by_batch = shard_plan_invariant_status(records)

observed_sweeps = sorted({str(rec.get("sweep_id", "baseline")).strip() for rec in records})
observed_profiles = sorted({str(rec.get("profile_id", "")).strip() for rec in records if str(rec.get("profile_id", "")).strip()})
required_release_profiles = [profile_id for profile_id in release_profile_order if profile_id in observed_profiles]
observed_pairs = {
    (str(rec.get("profile_id", "")).strip(), str(rec.get("sweep_id", "baseline")).strip())
    for rec in records
    if str(rec.get("profile_id", "")).strip()
}
invariant_statuses = [invariant_status_by_batch.get(str(rec.get("batch_id", "")), "not_compared") for rec in records]
invariant_status_counts = Counter(invariant_statuses)
invariant_aggregate_status = "passed" if invariant_statuses and all(status == "passed" for status in invariant_statuses) else "failed"

release_contract_blockers: list[str] = []
required_sweeps: list[str] = []
expected_profile_sweep_runs = len(records)
observed_profile_sweep_runs = len(observed_pairs)
missing_profile_sweep_pairs: list[str] = []
extra_profile_sweep_pairs: list[str] = []
if release_mode:
    required_sweeps = list(required_release_sweeps)
    required_sweep_set = set(required_sweeps)
    observed_sweep_set = set(observed_sweeps)
    missing_sweeps = sorted(required_sweep_set - observed_sweep_set)
    extra_sweeps = sorted(observed_sweep_set - required_sweep_set)
    if missing_sweeps:
        release_contract_blockers.append("missing_sweeps=" + ",".join(missing_sweeps))
    if extra_sweeps:
        release_contract_blockers.append("extra_sweeps=" + ",".join(extra_sweeps))

    expected_pairs = {
        (profile_id, sweep_id)
        for profile_id in required_release_profiles
        for sweep_id in required_release_sweeps
    }
    expected_profile_sweep_runs = len(expected_pairs)
    missing_profile_sweep_pairs = sorted(f"{profile}/{sweep}" for profile, sweep in expected_pairs - observed_pairs)
    extra_profile_sweep_pairs = sorted(f"{profile}/{sweep}" for profile, sweep in observed_pairs - expected_pairs)
    observed_profile_sweep_runs = len(observed_pairs)
    if missing_profile_sweep_pairs:
        release_contract_blockers.append("missing_profile_sweep_pairs=" + ",".join(missing_profile_sweep_pairs))
    if extra_profile_sweep_pairs:
        release_contract_blockers.append("extra_profile_sweep_pairs=" + ",".join(extra_profile_sweep_pairs))
    if invariant_aggregate_status != "passed":
        release_contract_blockers.append(f"shard_plan_invariant_status={invariant_aggregate_status}")

release_contract_status = "passed" if not release_contract_blockers else "failed"

verdict_records: list[dict[str, object]] = []
strict_fail_count = 0
expected_backend_runs = run_count * len(required_release_providers)
required_frontend_providers = list(required_release_providers)
if not release_mode:
    expected_backend_runs = len(selected_run_indexes) * len(selected_providers)
    required_frontend_providers = list(selected_providers)
frontend_init_required = release_mode or os.environ.get("BATCH_FRONTEND_MODE", "").strip().lower() != "never"

for rec in records:
    run_matrix_tsv = Path(str(rec["run_matrix_tsv"]))
    frontend_matrix_md = Path(str(rec["frontend_matrix_md"]))

    stats = parse_backend_stats(run_matrix_tsv)
    frontend_rows = {
        "frontend_qwen_status": parse_frontend_row(frontend_matrix_md, "qwen-code"),
        "frontend_claude_status": parse_frontend_row(frontend_matrix_md, "claude-code"),
        "frontend_codex_status": parse_frontend_row(frontend_matrix_md, "codex-code"),
    }
    frontend_statuses = {
        key: row["status"]
        for key, row in frontend_rows.items()
    }
    frontend_reasons = {
        key: row["reasons"]
        for key, row in frontend_rows.items()
    }
    excellent_blockers = excellent_blockers_from_stats(stats)
    excellent_blockers_by_step = list(stats.get("excellent_blockers_by_step", []))

    shard_plan_invariant = invariant_status_by_batch.get(str(rec["batch_id"]), "not_compared")
    blockers = strict_blockers(
        rec,
        stats,
        frontend_statuses,
        frontend_reasons,
        shard_plan_invariant,
        release_mode,
        expected_backend_runs,
        len(selected_run_indexes) if not release_mode else run_count,
        required_frontend_providers,
        frontend_init_required,
    )
    blockers.extend(invariant_blockers_by_batch.get(str(rec["batch_id"]), []))
    strict_status = "passed" if not blockers else "failed"
    if blockers:
        strict_fail_count += 1

    execution = rec.get("execution") if isinstance(rec.get("execution"), dict) else {}
    execution_strategy = str((execution or {}).get("strategy", "-"))
    execution_max_parallel = str((execution or {}).get("max_parallel_tasks", "-"))
    execution_failure_policy = str((execution or {}).get("failure_policy", "-"))
    execution_shard_mode = str((execution or {}).get("shard_discovery_mode", "-"))
    raw_output_refs = collect_raw_output_refs(rec)

    tsv_lines.append(
        "\t".join(
            [
                str(rec["profile_id"]),
                str(rec.get("sweep_id", "baseline")),
                str(rec["batch_id"]),
                str(rec["source_kind"]),
                str(rec["expected_repo_count"]),
                execution_strategy,
                execution_max_parallel,
                execution_failure_policy,
                execution_shard_mode,
                shard_plan_invariant,
                str(rec["status"]),
                strict_status,
                str(stats["hard"]),
                str(stats["total"]),
                str(stats["semantic_hard_fail"]),
                str(stats["off_topic_hits"]),
                str(stats["artifact_non_snapshot"]),
                str(stats["runtime_contract_failed"]),
                str(stats["runner_unavailable"]),
                str(stats["runtime_timeout"]),
                str(stats["infra_signal_terminated"]),
                str(stats["infra_incomplete_cycle"]),
                str(stats["quality_gates_failed"]),
                str(stats["artifact_quality_failed"]),
                str(stats["summary_missing"]),
                str(stats["precheck_failed"]),
                str(stats["runtime_flow_failed"]),
                str(stats["evidence_scope_hits"]),
                str(stats["cross_repo_missing_hits"]),
                str(stats["runtime_flow_issue_hits"]),
                str(stats["repair_attempts"]),
                str(stats["repair_exhausted"]),
                str(stats["fresh_retries"]),
                str(stats["focused_repairs"]),
                str(stats["stall_count"]),
                str(stats["pre_artifact_stalls"]),
                str(stats["post_artifact_stalls"]),
                str(stats["valid_artifact_controlled_stops"]),
                str(stats["zero_output_pre_artifact_stalls"]),
                str(stats["partial_failure_count"]),
                str(stats["quality_alerts"]),
                frontend_statuses["frontend_qwen_status"],
                frontend_statuses["frontend_claude_status"],
                frontend_statuses["frontend_codex_status"],
                "; ".join(blockers) if blockers else "-",
                str(len(raw_output_refs)),
                str(rec["run_matrix_tsv"]),
                str(rec["execution_report_md"]),
                str(rec.get("inventory_json", "-")),
            ]
        )
    )

    md_lines.append(
        "| "
        f"{rec['profile_id']} | {rec.get('sweep_id', 'baseline')} | {rec['batch_id']} | {rec['status']} | {strict_status} | "
        f"{shard_plan_invariant} | "
        f"{stats['hard']}/{stats['total']} | {stats['semantic_hard_fail']} | {stats['off_topic_hits']} | {stats['artifact_non_snapshot']} | "
        f"{stats['artifact_quality_failed']} | "
        f"{stats['evidence_scope_hits']} | {stats['cross_repo_missing_hits']} | "
        f"{int(stats['runtime_flow_failed']) + int(stats['runtime_flow_issue_hits'])} | "
        f"repair={stats['repair_attempts']}; exhausted={stats['repair_exhausted']}; focused={stats['focused_repairs']}; "
        f"stalls={stats['stall_count']} (pre={stats['pre_artifact_stalls']}; post={stats['post_artifact_stalls']}); "
        f"valid_controlled={stats['valid_artifact_controlled_stops']}; zero_pre={stats['zero_output_pre_artifact_stalls']}; partial={stats['partial_failure_count']}; alerts={stats['quality_alerts']} | "
        f"{frontend_statuses['frontend_qwen_status']}/{frontend_statuses['frontend_claude_status']}/{frontend_statuses['frontend_codex_status']} | "
        f"{'; '.join(blockers) if blockers else '-'} | {rec['run_matrix_md']} | {rec['execution_report_md']} |"
    )

    verdict_records.append(
        {
            "profile_id": rec["profile_id"],
            "sweep_id": rec.get("sweep_id", "baseline"),
            "batch_id": rec["batch_id"],
            "status": rec["status"],
            "strict_status": strict_status,
            "blocking_reasons": blockers,
            "excellent_blockers": excellent_blockers,
            "excellent_blockers_by_step": excellent_blockers_by_step,
            "shard_plan_invariant": shard_plan_invariant,
            "execution": {
                "strategy": execution_strategy,
                "max_parallel_tasks": execution_max_parallel,
                "failure_policy": execution_failure_policy,
                "shard_discovery_mode": execution_shard_mode,
            },
            "backend": {
                "hard_pass": int(stats["hard"]),
                "total_runs": int(stats["total"]),
                "artifact_non_snapshot_runs": int(stats["artifact_non_snapshot"]),
                "runtime_contract_failed_failures": int(stats["runtime_contract_failed"]),
                "runner_unavailable_failures": int(stats["runner_unavailable"]),
                "runtime_timeout_failures": int(stats["runtime_timeout"]),
                "infra_signal_terminated_failures": int(stats["infra_signal_terminated"]),
                "infra_incomplete_cycle_failures": int(stats["infra_incomplete_cycle"]),
                "quality_gates_failed_failures": int(stats["quality_gates_failed"]),
                "artifact_quality_failed_failures": int(stats["artifact_quality_failed"]),
                "summary_missing_failures": int(stats["summary_missing"]),
                "precheck_failed_failures": int(stats["precheck_failed"]),
                "runtime_flow_failed_runs": int(stats["runtime_flow_failed"]),
                "runtime_flow_issue_hits": int(stats["runtime_flow_issue_hits"]),
                "repair_attempts": int(stats["repair_attempts"]),
                "repair_exhausted": int(stats["repair_exhausted"]),
                "fresh_retries": int(stats["fresh_retries"]),
                "focused_repairs": int(stats["focused_repairs"]),
                "stall_count": int(stats["stall_count"]),
                "pre_artifact_stalls": int(stats["pre_artifact_stalls"]),
                "post_artifact_stalls": int(stats["post_artifact_stalls"]),
                "valid_artifact_controlled_stops": int(stats["valid_artifact_controlled_stops"]),
                "zero_output_pre_artifact_stalls": int(stats["zero_output_pre_artifact_stalls"]),
                "partial_failure_count": int(stats["partial_failure_count"]),
            },
            "frontend": frontend_statuses,
            "artifacts": {
                "run_matrix_tsv": rec["run_matrix_tsv"],
                "run_matrix_md": rec["run_matrix_md"],
                "frontend_matrix_md": rec["frontend_matrix_md"],
                "execution_report_md": rec["execution_report_md"],
                "driver_log": rec["driver_log"],
                "inventory_json": rec.get("inventory_json", "-"),
                "raw_output_ref_count": len(raw_output_refs),
            },
        }
    )

release_contract_failed = release_mode and release_contract_status != "passed"
verdict = "PASS" if strict_fail_count == 0 and not release_contract_failed else "FAIL"
release_state = "RELEASE READY" if verdict == "PASS" else "RELEASE BLOCKED"

backend_aggregate: dict[str, int] = {}
for rec in verdict_records:
    backend = rec.get("backend")
    if not isinstance(backend, dict):
        continue
    for key, value in backend.items():
        try:
            backend_aggregate[key] = backend_aggregate.get(key, 0) + int(value)
        except Exception:
            continue
matrix_excellent_blockers = [
    f"{item['profile_id']}/{item['sweep_id']}:{blocker}"
    for item in verdict_records
    for blocker in item.get("excellent_blockers", [])
]
matrix_excellent_blockers_by_step = [
    {
        **step_entry,
        "profile_id": item["profile_id"],
        "sweep_id": item["sweep_id"],
        "batch_id": item["batch_id"],
    }
    for item in verdict_records
    for step_entry in item.get("excellent_blockers_by_step", [])
    if isinstance(step_entry, dict)
]

out_md.parent.mkdir(parents=True, exist_ok=True)
out_md.write_text("\n".join(md_lines) + "\n", encoding="utf-8")
out_tsv.write_text("\n".join(tsv_lines) + "\n", encoding="utf-8")

generated_at = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
if release_mode:
    verdict_lines = [
        f"# Release Verdict: {matrix_id}",
        "",
        f"- generated_at_utc: {generated_at}",
        f"- verdict: {verdict}",
        f"- release_state: {release_state}",
        f"- profile_sweep_runs: {len(verdict_records)}",
        f"- strict_pass_runs: {len(verdict_records) - strict_fail_count}",
        f"- strict_fail_runs: {strict_fail_count}",
        "- release_mode: 1",
        f"- release_contract_status: {release_contract_status}",
        "",
        "## Blocking Items",
    ]
else:
    verdict_lines = [
        f"# Matrix Result: {matrix_id}",
        "",
        f"- generated_at_utc: {generated_at}",
        f"- result: {verdict}",
        "- mode: non-release",
        f"- profile_sweep_runs: {len(verdict_records)}",
        f"- strict_pass_runs: {len(verdict_records) - strict_fail_count}",
        f"- strict_fail_runs: {strict_fail_count}",
        "",
        "## Blocking Items",
    ]

if strict_fail_count == 0 and not release_contract_failed:
    verdict_lines.append("- none")
else:
    for item in verdict_records:
        if item["strict_status"] != "failed":
            continue
        verdict_lines.append(
            f"- {item['profile_id']} / {item['sweep_id']} ({item['batch_id']}):"
        )
        for reason in item["blocking_reasons"]:
            verdict_lines.append(f"  - {reason}")
        artifacts = item["artifacts"]
        verdict_lines.append(f"  - run_matrix: {artifacts['run_matrix_md']}")
        verdict_lines.append(f"  - execution_report: {artifacts['execution_report_md']}")
        verdict_lines.append(f"  - frontend_matrix: {artifacts['frontend_matrix_md']}")
        verdict_lines.append(f"  - inventory: {artifacts['inventory_json']} (raw_output_refs={artifacts['raw_output_ref_count']})")
    if release_mode and release_contract_failed:
        verdict_lines.append("- release_contract:")
        for blocker in release_contract_blockers:
            verdict_lines.append(f"  - {blocker}")

verdict_lines.extend(["", "## Excellent Blockers"])
excellent_records = [item for item in verdict_records if item.get("excellent_blockers")]
if not excellent_records:
    verdict_lines.append("- none")
else:
    for item in excellent_records:
        verdict_lines.append(
            f"- {item['profile_id']} / {item['sweep_id']} ({item['batch_id']}): "
            f"{'; '.join(str(blocker) for blocker in item.get('excellent_blockers', []))}"
        )
verdict_lines.extend(["", "## Excellent Blockers By Step"])
if not matrix_excellent_blockers_by_step:
    verdict_lines.append("- none")
else:
    for step_entry in matrix_excellent_blockers_by_step:
        verdict_lines.append(
            "- "
            f"{step_entry.get('profile_id', '-')}/{step_entry.get('sweep_id', '-')} "
            f"({step_entry.get('batch_id', '-')}): "
            f"step={step_entry.get('step_id', '-')}; blocker={step_entry.get('blocker_code', '-')}; "
            f"repair_attempts={step_entry.get('repair_attempts', 0)}; "
            f"stall_count={step_entry.get('stall_count', 0)}; "
            f"valid_artifact_controlled_stops={step_entry.get('valid_artifact_controlled_stops', 0)}; "
            f"stop_kind={step_entry.get('stop_kind', '-')}; "
            f"initial_artifact_state={step_entry.get('initial_artifact_state', '-')}; "
            f"final_validation_class={step_entry.get('final_validation_class', 'unknown')}"
        )

if release_mode:
    verdict_lines.extend(
        [
            "",
            "## Release Contract",
            f"- required_sweeps: {', '.join(required_sweeps) if required_sweeps else '-'}",
            f"- observed_sweeps: {', '.join(observed_sweeps) if observed_sweeps else '-'}",
            f"- expected_profile_sweep_runs: {expected_profile_sweep_runs}",
            f"- observed_profile_sweep_runs: {observed_profile_sweep_runs}",
            f"- shard_plan_invariant_status: {invariant_aggregate_status}",
            f"- contract_status: {release_contract_status}",
        ]
    )
    verdict_payload = {
        "matrix_id": matrix_id,
        "generated_at_utc": generated_at,
        "verdict": verdict,
        "release_state": release_state,
        "profile_sweep_runs": len(verdict_records),
        "strict_pass_runs": len(verdict_records) - strict_fail_count,
        "strict_fail_runs": strict_fail_count,
        "backend": backend_aggregate,
        "excellent_blockers": matrix_excellent_blockers,
        "excellent_blockers_by_step": matrix_excellent_blockers_by_step,
        "release_contract": {
            "mode": "release",
            "required_sweeps": required_sweeps,
            "observed_sweeps": observed_sweeps,
            "expected_profile_sweep_runs": expected_profile_sweep_runs,
            "observed_profile_sweep_runs": observed_profile_sweep_runs,
            "selected_providers": selected_providers,
            "selected_run_indexes": selected_run_indexes,
            "expected_backend_runs_per_profile_sweep": expected_backend_runs,
            "required_profiles": list(required_release_profiles),
            "observed_profiles": observed_profiles,
            "missing_profile_sweep_pairs": missing_profile_sweep_pairs,
            "extra_profile_sweep_pairs": extra_profile_sweep_pairs,
            "shard_plan_invariant_status": invariant_aggregate_status,
            "shard_plan_invariant_counts": dict(invariant_status_counts),
            "contract_status": release_contract_status,
            "blocking_reasons": release_contract_blockers,
        },
        "records": verdict_records,
    }
else:
    matrix_blockers = [
        f"{item['profile_id']}/{item['sweep_id']}:{reason}"
        for item in verdict_records
        for reason in item.get("blocking_reasons", [])
    ]
    verdict_payload = {
        "matrix_id": matrix_id,
        "generated_at_utc": generated_at,
        "result": verdict,
        "mode": "non-release",
        "profile_sweep_runs": len(verdict_records),
        "strict_pass_runs": len(verdict_records) - strict_fail_count,
        "strict_fail_runs": strict_fail_count,
        "backend": backend_aggregate,
        "excellent_blockers": matrix_excellent_blockers,
        "excellent_blockers_by_step": matrix_excellent_blockers_by_step,
        "selected_providers": selected_providers,
        "selected_run_indexes": selected_run_indexes,
        "blocking_reasons": matrix_blockers,
        "records": verdict_records,
    }

verdict_md.write_text("\n".join(verdict_lines) + "\n", encoding="utf-8")
verdict_json.write_text(json.dumps(verdict_payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
PY

log "profile matrix markdown: $MATRIX_REPORT_MD"
log "profile matrix tsv: $MATRIX_REPORT_TSV"
if [[ "$RELEASE_MODE" == "1" ]]; then
  log "release verdict markdown: $RESULT_MD"
  log "release verdict json: $RESULT_JSON"
else
  log "matrix result markdown: $RESULT_MD"
  log "matrix result json: $RESULT_JSON"
fi

MATRIX_RESULT="$(python3 - "$RESULT_JSON" "$RELEASE_MODE" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding="utf-8"))
release_mode = str(sys.argv[2]).strip() == "1"
key = "verdict" if release_mode else "result"
print(str(payload.get(key, "FAIL")).strip().upper())
PY
)"

if [[ "$MATRIX_RESULT" != "PASS" ]]; then
  if [[ "$RELEASE_MODE" == "1" ]]; then
    die "RELEASE BLOCKED for matrix=$MATRIX_ID (see $RESULT_MD)"
  fi
  die "matrix result failed for matrix=$MATRIX_ID (see $RESULT_MD)"
fi

if [[ "$RELEASE_MODE" == "1" ]]; then
  log "matrix completed successfully (RELEASE READY)"
else
  log "matrix completed successfully (non-release PASS)"
fi
exit 0
