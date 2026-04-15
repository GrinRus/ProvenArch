#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
BATCH_SCRIPT="${BATCH_SCRIPT:-$PROVENARCH_ROOT/scripts/full-run-batch-5x2.sh}"
E2E_MATRIX_FILE="${E2E_MATRIX_FILE:-}"
MATRIX_ID="${MATRIX_ID:-matrix-$(date -u +'%Y%m%dT%H%M%SZ')}"
RUN_COUNT="${RUN_COUNT:-5}"
E2E_TMP_ROOT="${E2E_TMP_ROOT:-/tmp/provenarch-test_arch_project}"
REPORTS_ROOT="${REPORTS_ROOT:-$E2E_TMP_ROOT/reports}"
MATRIX_ROOT="${MATRIX_ROOT:-$E2E_TMP_ROOT/matrix/$MATRIX_ID}"
ACP_CLAUDE_CMD_BIN="${ACP_CLAUDE_CMD_BIN:-claude}"
ACP_QWEN_CMD_BIN="${ACP_QWEN_CMD_BIN:-qwen}"
ACP_APPLY_TIMEOUTS_VIA_API="${ACP_APPLY_TIMEOUTS_VIA_API:-1}"
E2E_MATRIX_RELEASE_MODE="${E2E_MATRIX_RELEASE_MODE:-auto}"
E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES="${E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES:-0}"
MATRIX_MAX_PARALLEL_COMBINATIONS="${MATRIX_MAX_PARALLEL_COMBINATIONS:-2}"
BATCH_FRONTEND_MAX_PARALLEL="${BATCH_FRONTEND_MAX_PARALLEL:-1}"
MATRIX_DRIVER_LOG="${MATRIX_DRIVER_LOG:-$MATRIX_ROOT/driver.log}"
declare -a MATRIX_ACTIVE_PIDS=()
declare -a MATRIX_ACTIVE_LABELS=()

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

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    die "required command is unavailable: $cmd"
  fi
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

normalize_binary_flag() {
  local raw="$1"
  local name="$2"
  case "$raw" in
    1|true|TRUE|yes|YES|on|ON)
      printf '1'
      ;;
    0|false|FALSE|no|NO|off|OFF|"")
      printf '0'
      ;;
    *)
      die "$name must be 0|1 (or boolean aliases), got '$raw'"
      ;;
  esac
}

slugify() {
  local value
  value="$(echo "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
  if [[ -z "$value" ]]; then
    value="item"
  fi
  printf '%s' "$value"
}

if [[ -z "$E2E_MATRIX_FILE" ]]; then
  die "E2E_MATRIX_FILE is required (YAML with profiles[] and optional sweeps[])"
fi
if [[ ! -f "$E2E_MATRIX_FILE" ]]; then
  die "E2E_MATRIX_FILE does not exist: $E2E_MATRIX_FILE"
fi
if [[ "$RUN_COUNT" != "5" ]]; then
  die "RUN_COUNT must be 5 for matrix mode (got '$RUN_COUNT')"
fi
if [[ "$ACP_APPLY_TIMEOUTS_VIA_API" != "0" && "$ACP_APPLY_TIMEOUTS_VIA_API" != "1" ]]; then
  die "ACP_APPLY_TIMEOUTS_VIA_API must be 0 or 1, got '$ACP_APPLY_TIMEOUTS_VIA_API'"
fi
if [[ ! -x "$BATCH_SCRIPT" ]]; then
  die "batch script is unavailable: $BATCH_SCRIPT"
fi
if [[ ! "$MATRIX_MAX_PARALLEL_COMBINATIONS" =~ ^[1-9][0-9]*$ ]]; then
  die "MATRIX_MAX_PARALLEL_COMBINATIONS must be a positive integer, got '$MATRIX_MAX_PARALLEL_COMBINATIONS'"
fi
if [[ ! "$BATCH_FRONTEND_MAX_PARALLEL" =~ ^[1-9][0-9]*$ ]]; then
  die "BATCH_FRONTEND_MAX_PARALLEL must be a positive integer, got '$BATCH_FRONTEND_MAX_PARALLEL'"
fi

require_cmd bash
require_cmd python3
require_cmd "$ACP_CLAUDE_CMD_BIN"
require_cmd "$ACP_QWEN_CMD_BIN"

mkdir -p "$MATRIX_ROOT" "$REPORTS_ROOT"
mkdir -p "$(dirname "$MATRIX_DRIVER_LOG")"
: > "$MATRIX_DRIVER_LOG"

RELEASE_MODE="$(normalize_release_mode "$E2E_MATRIX_RELEASE_MODE")"
ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES="$(normalize_binary_flag "$E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES" "E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES")"

DIAGNOSTIC_TIMEOUT_ENV_KEYS=(
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
)

DIAGNOSTIC_TIMEOUT_OVERRIDES=()
for key in "${DIAGNOSTIC_TIMEOUT_ENV_KEYS[@]}"; do
  value="${!key:-}"
  if [[ -n "$value" ]]; then
    DIAGNOSTIC_TIMEOUT_OVERRIDES+=("$key=$value")
  fi
done

log "timeout controls: apply_via_api=$ACP_APPLY_TIMEOUTS_VIA_API step=${ACP_RUNTIME_STEP_TIMEOUT_SEC:-auto} heartbeat=${ACP_RUNTIME_HEARTBEAT_SEC:-auto} pipeline=${ACP_PIPELINE_TIMEOUT_SEC:-auto} kill_grace=${ACP_PIPELINE_KILL_GRACE_SEC:-auto} api_ready=${ACP_API_READY_TIMEOUT_SEC:-auto} api_init=${ACP_API_INIT_TIMEOUT_SEC:-auto} ui_init=${ACP_UI_INIT_POLL_TIMEOUT_SEC:-auto} ui_cancel=${ACP_UI_CANCEL_POLL_TIMEOUT_SEC:-auto}"
log "release guard: mode=$RELEASE_MODE matrix_id=$MATRIX_ID allow_diagnostic_timeout_overrides=$ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES overrides_detected=${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}"
log "matrix parallelism: max_parallel_combinations=$MATRIX_MAX_PARALLEL_COMBINATIONS frontend_max_parallel_runs=$BATCH_FRONTEND_MAX_PARALLEL"
if [[ "${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}" -gt 0 ]]; then
  log "diagnostic timeout overrides: ${DIAGNOSTIC_TIMEOUT_OVERRIDES[*]}"
fi
if [[ "$RELEASE_MODE" == "1" && "$ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES" != "1" && "${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}" -gt 0 ]]; then
  die "release guard blocked diagnostic timeout overrides; clear env or set E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES=1 for explicit debug-only override"
fi

COMBINATIONS_TSV="$MATRIX_ROOT/profile-sweep-combinations.tsv"
RECORDS_JSONL="$MATRIX_ROOT/profile-runs.jsonl"
MATRIX_RECORDS_DIR="$MATRIX_ROOT/records"
mkdir -p "$MATRIX_RECORDS_DIR"
: >"$RECORDS_JSONL"

python3 - "$E2E_MATRIX_FILE" "$COMBINATIONS_TSV" <<'PY'
import sys
from pathlib import Path

try:
    import yaml  # type: ignore
except Exception as exc:
    raise SystemExit(f"PyYAML is required for parsing matrix file: {exc}")

matrix_path = Path(sys.argv[1]).resolve()
out_path = Path(sys.argv[2]).resolve()
payload = yaml.safe_load(matrix_path.read_text(encoding="utf-8"))

if isinstance(payload, dict):
    profiles = payload.get("profiles")
    sweeps = payload.get("sweeps")
elif isinstance(payload, list):
    profiles = payload
    sweeps = None
else:
    profiles = None
    sweeps = None

if not isinstance(profiles, list) or not profiles:
    raise SystemExit(f"matrix file {matrix_path} must contain non-empty profiles[]")

required_profiles = {
    "single-path": {"source_kind": "path", "min_repos": 1, "max_repos": 1},
    "single-git_url": {"source_kind": "git_url", "min_repos": 1, "max_repos": 1},
    "multi-path": {"source_kind": "path", "min_repos": 2, "max_repos": None},
    "multi-git_url": {"source_kind": "git_url", "min_repos": 2, "max_repos": None},
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

    contract = required_profiles.get(profile_id)
    if contract is None:
        raise SystemExit(
            f"profiles[{idx}] id must be one of: {', '.join(sorted(required_profiles.keys()))}; got: {profile_id}"
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

missing = sorted(set(required_profiles.keys()) - seen_ids)
extra = sorted(seen_ids - set(required_profiles.keys()))
if missing:
    raise SystemExit(f"matrix file {matrix_path} missing required profile ids: {', '.join(missing)}")
if extra:
    raise SystemExit(f"matrix file {matrix_path} has unsupported profile ids: {', '.join(extra)}")
if len(profiles) != len(required_profiles):
    raise SystemExit(
        f"matrix file {matrix_path} must contain exactly {len(required_profiles)} profiles"
    )

allowed = {
    "strategy": {"sequential", "parallel"},
    "failure_policy": {"fail_fast", "best_effort"},
    "shard_discovery_mode": {"heuristics", "semantic"},
    "repo_selection": {"all", "backend_only"},
}

default_sweep = {
    "id": "baseline",
    "strategy": "sequential",
    "max_parallel_tasks": 1,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "heuristics",
    "repo_selection": "all",
}

sweep_rows: list[dict[str, object]] = []
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

        repo_selection = str(item.get("repo_selection", default_sweep["repo_selection"]))
        if repo_selection not in allowed["repo_selection"]:
            raise SystemExit(
                "sweeps[%d] repo_selection must be one of: %s; got: %s"
                % (idx, ", ".join(sorted(allowed["repo_selection"])), repo_selection)
            )

        sweep_rows.append(
            {
                "id": sweep_id,
                "strategy": strategy,
                "max_parallel_tasks": max_parallel_tasks,
                "failure_policy": failure_policy,
                "shard_discovery_mode": shard_mode,
                "repo_selection": repo_selection,
            }
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
                    str(sweep["repo_selection"]),
                ]
            )
        )

out_path.parent.mkdir(parents=True, exist_ok=True)
out_path.write_text("\n".join(rows) + "\n", encoding="utf-8")
PY

drop_matrix_active_worker() {
  local drop_idx="$1"
  local -a next_pids
  local -a next_labels
  next_pids=()
  next_labels=()
  local idx
  for idx in "${!MATRIX_ACTIVE_PIDS[@]}"; do
    if [[ "$idx" == "$drop_idx" ]]; then
      continue
    fi
    next_pids+=("${MATRIX_ACTIVE_PIDS[$idx]}")
    next_labels+=("${MATRIX_ACTIVE_LABELS[$idx]}")
  done
  MATRIX_ACTIVE_PIDS=()
  MATRIX_ACTIVE_LABELS=()
  if [[ "${#next_pids[@]}" -gt 0 ]]; then
    MATRIX_ACTIVE_PIDS=("${next_pids[@]}")
    MATRIX_ACTIVE_LABELS=("${next_labels[@]}")
  fi
}

wait_for_matrix_worker_slot() {
  local idx
  local pid
  local label
  local exit_code
  while true; do
    for idx in "${!MATRIX_ACTIVE_PIDS[@]}"; do
      pid="${MATRIX_ACTIVE_PIDS[$idx]}"
      if ! kill -0 "$pid" >/dev/null 2>&1; then
        if wait "$pid"; then
          exit_code=0
        else
          exit_code=$?
        fi
        label="${MATRIX_ACTIVE_LABELS[$idx]}"
        drop_matrix_active_worker "$idx"
        if [[ "$exit_code" -ne 0 ]]; then
          die "matrix worker failed unexpectedly worker=$label exit=$exit_code"
        fi
        return 0
      fi
    done
    sleep 0.2
  done
}

wait_for_all_matrix_workers() {
  while [[ "${#MATRIX_ACTIVE_PIDS[@]}" -gt 0 ]]; do
    wait_for_matrix_worker_slot
  done
}

launch_matrix_worker() {
  local combination_index="$1"
  local profile_id="$2"
  local repos_file="$3"
  local expected_repo_count="$4"
  local source_kind="$5"
  local sweep_id="$6"
  local sweep_strategy="$7"
  local sweep_max_parallel="$8"
  local sweep_failure_policy="$9"
  local sweep_shard_mode="${10}"
  local sweep_repo_selection="${11}"
  local profile_slug="${12}"
  local sweep_slug="${13}"

  local batch_id="${MATRIX_ID}-${profile_slug}-${sweep_slug}"
  local profile_root="$MATRIX_ROOT/profiles/$profile_slug/$sweep_slug"
  local driver_log="$profile_root/driver.log"
  local record_file=""
  record_file="$MATRIX_RECORDS_DIR/$(printf '%03d' "$combination_index")-${profile_slug}-${sweep_slug}.json"

  mkdir -p "$profile_root"
  log "running profile=$profile_id sweep=$sweep_id source_kind=$source_kind expected_repo_count=$expected_repo_count batch_id=$batch_id"

  (
    set +e
    status="passed"
    if ! (
      cd "$PROVENARCH_ROOT"
      BATCH_ID="$batch_id" \
      RUN_COUNT="$RUN_COUNT" \
      TARGET_REPOS_FILE="$repos_file" \
      PROFILE_ID="$profile_id" \
      PROFILE_SOURCE_KIND="$source_kind" \
      EXPECTED_REPO_COUNT="$expected_repo_count" \
      SWEEP_ID="$sweep_id" \
      E2E_TMP_ROOT="$E2E_TMP_ROOT" \
      REPORTS_ROOT="$REPORTS_ROOT" \
      ACP_CLAUDE_CMD_BIN="$ACP_CLAUDE_CMD_BIN" \
      ACP_QWEN_CMD_BIN="$ACP_QWEN_CMD_BIN" \
      ACP_APPLY_TIMEOUTS_VIA_API="$ACP_APPLY_TIMEOUTS_VIA_API" \
      ACP_RUNTIME_STEP_TIMEOUT_SEC="${ACP_RUNTIME_STEP_TIMEOUT_SEC:-}" \
      ACP_RUNTIME_HEARTBEAT_SEC="${ACP_RUNTIME_HEARTBEAT_SEC:-}" \
      ACP_PIPELINE_TIMEOUT_SEC="${ACP_PIPELINE_TIMEOUT_SEC:-}" \
      ACP_PIPELINE_KILL_GRACE_SEC="${ACP_PIPELINE_KILL_GRACE_SEC:-}" \
      ACP_API_READY_TIMEOUT_SEC="${ACP_API_READY_TIMEOUT_SEC:-}" \
      ACP_API_INIT_TIMEOUT_SEC="${ACP_API_INIT_TIMEOUT_SEC:-}" \
      ACP_UI_INIT_POLL_TIMEOUT_SEC="${ACP_UI_INIT_POLL_TIMEOUT_SEC:-}" \
      ACP_UI_CANCEL_POLL_TIMEOUT_SEC="${ACP_UI_CANCEL_POLL_TIMEOUT_SEC:-}" \
      ACP_EXECUTION_STRATEGY="$sweep_strategy" \
      ACP_MAX_PARALLEL_TASKS="$sweep_max_parallel" \
      ACP_FAILURE_POLICY="$sweep_failure_policy" \
      ACP_SHARD_DISCOVERY_MODE="$sweep_shard_mode" \
      ACP_REPO_SELECTION="$sweep_repo_selection" \
      BATCH_FRONTEND_MAX_PARALLEL="$BATCH_FRONTEND_MAX_PARALLEL" \
      "$BATCH_SCRIPT"
    ) >"$driver_log" 2>&1; then
      status="failed"
      log "profile+sweep failed: profile=$profile_id sweep=$sweep_id (see $driver_log)"
    fi

    run_matrix_tsv="$REPORTS_ROOT/run_matrix_${batch_id}.tsv"
    run_matrix_md="$REPORTS_ROOT/run_matrix_${batch_id}.md"
    frontend_matrix_md="$REPORTS_ROOT/frontend_e2e_matrix_${batch_id}.md"
    frontend_cancel_matrix_md="$REPORTS_ROOT/frontend_cancel_e2e_matrix_${batch_id}.md"
    quality_report_md="$REPORTS_ROOT/quality_report_${batch_id}.md"
    documentation_audit_md="$REPORTS_ROOT/documentation_audit_${batch_id}.md"

    python3 - "$record_file" \
      "$combination_index" \
      "$profile_id" "$profile_slug" "$batch_id" "$source_kind" "$expected_repo_count" "$repos_file" "$status" \
      "$sweep_id" "$sweep_strategy" "$sweep_max_parallel" "$sweep_failure_policy" "$sweep_shard_mode" "$sweep_repo_selection" \
      "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$frontend_cancel_matrix_md" "$quality_report_md" "$documentation_audit_md" "$driver_log" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1]).resolve()
payload = {
    "combination_index": int(sys.argv[2]),
    "profile_id": sys.argv[3],
    "profile_slug": sys.argv[4],
    "batch_id": sys.argv[5],
    "source_kind": sys.argv[6],
    "expected_repo_count": int(sys.argv[7]),
    "repos_file": sys.argv[8],
    "status": sys.argv[9],
    "sweep_id": sys.argv[10],
    "execution": {
        "strategy": sys.argv[11],
        "max_parallel_tasks": int(sys.argv[12]),
        "failure_policy": sys.argv[13],
        "shard_discovery_mode": sys.argv[14],
        "repo_selection": sys.argv[15],
    },
    "run_matrix_tsv": sys.argv[16],
    "run_matrix_md": sys.argv[17],
    "frontend_matrix_md": sys.argv[18],
    "frontend_cancel_matrix_md": sys.argv[19],
    "quality_report_md": sys.argv[20],
    "documentation_audit_md": sys.argv[21],
    "driver_log": sys.argv[22],
}
path.write_text(json.dumps(payload, ensure_ascii=True) + "\n", encoding="utf-8")
PY
    exit 0
  ) &

  MATRIX_ACTIVE_PIDS+=("$!")
  MATRIX_ACTIVE_LABELS+=("${profile_id}/${sweep_id}")
}

combination_index=0
while IFS=$'\t' read -r profile_id repos_file expected_repo_count source_kind sweep_id sweep_strategy sweep_max_parallel sweep_failure_policy sweep_shard_mode sweep_repo_selection; do
  [[ -z "$profile_id" ]] && continue
  combination_index=$((combination_index + 1))
  profile_slug="$(slugify "$profile_id")"
  sweep_slug="$(slugify "$sweep_id")"

  launch_matrix_worker \
    "$combination_index" \
    "$profile_id" "$repos_file" "$expected_repo_count" "$source_kind" "$sweep_id" \
    "$sweep_strategy" "$sweep_max_parallel" "$sweep_failure_policy" "$sweep_shard_mode" "$sweep_repo_selection" \
    "$profile_slug" "$sweep_slug"

  while [[ "${#MATRIX_ACTIVE_PIDS[@]}" -ge "$MATRIX_MAX_PARALLEL_COMBINATIONS" ]]; do
    wait_for_matrix_worker_slot
  done
done < "$COMBINATIONS_TSV"

wait_for_all_matrix_workers

python3 - "$MATRIX_RECORDS_DIR" "$RECORDS_JSONL" "$combination_index" <<'PY'
import json
import sys
from pathlib import Path

records_dir = Path(sys.argv[1]).resolve()
records_jsonl = Path(sys.argv[2]).resolve()
expected_count = int(sys.argv[3])
record_files = sorted(records_dir.glob("*.json"))
if len(record_files) != expected_count:
    raise SystemExit(f"matrix records count mismatch: expected={expected_count} got={len(record_files)} in {records_dir}")

records = []
for path in record_files:
    payload = json.loads(path.read_text(encoding="utf-8"))
    records.append(payload)
records.sort(key=lambda item: int(item.get("combination_index", 0)))
with records_jsonl.open("w", encoding="utf-8") as out:
    for payload in records:
        out.write(json.dumps(payload, ensure_ascii=True))
        out.write("\n")
PY

MATRIX_REPORT_MD="$REPORTS_ROOT/profile_matrix_${MATRIX_ID}.md"
MATRIX_REPORT_TSV="$REPORTS_ROOT/profile_matrix_${MATRIX_ID}.tsv"
VERDICT_MD="$REPORTS_ROOT/release_verdict_${MATRIX_ID}.md"
VERDICT_JSON="$REPORTS_ROOT/release_verdict_${MATRIX_ID}.json"

python3 - "$RECORDS_JSONL" "$MATRIX_REPORT_MD" "$MATRIX_REPORT_TSV" "$VERDICT_MD" "$VERDICT_JSON" "$MATRIX_ID" <<'PY'
import json
import re
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

records_path = Path(sys.argv[1]).resolve()
out_md = Path(sys.argv[2]).resolve()
out_tsv = Path(sys.argv[3]).resolve()
verdict_md = Path(sys.argv[4]).resolve()
verdict_json = Path(sys.argv[5]).resolve()
matrix_id = sys.argv[6]

records = []
for line in records_path.read_text(encoding="utf-8").splitlines():
    line = line.strip()
    if not line:
        continue
    records.append(json.loads(line))

if not records:
    raise SystemExit(f"no matrix records found in {records_path}")


def parse_frontend_matrix(path: Path, expected_runs: int | None = None) -> dict[str, object]:
    result: dict[str, object] = {
        "rows": [],
        "non_passed_rows": [],
        "by_provider": {
            "qwen-code": {"status": "missing", "passed": 0, "total": 0},
            "claude-code": {"status": "missing", "passed": 0, "total": 0},
        },
    }
    if not path.exists():
        return result

    lines = [line for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]
    row_entries: list[dict[str, object]] = []
    has_run_dimension = False
    for line in lines:
        if not line.startswith("|"):
            continue
        columns = [item.strip() for item in line.split("|")]
        if len(columns) < 4:
            continue
        provider = columns[1]
        run_raw = columns[2] if len(columns) > 2 else "-"
        status = columns[3] if len(columns) > 3 else "missing"
        if provider not in {"qwen-code", "claude-code"}:
            continue
        run_index = None
        if run_raw in {"passed", "failed", "skipped", "missing"}:
            status = run_raw
            run_index = 1
        elif run_raw and run_raw != "-":
            try:
                run_index = int(run_raw)
                has_run_dimension = True
            except Exception:
                run_index = None
        elif run_raw == "-":
            run_index = 1
        entry = {"provider": provider, "run_index": run_index, "status": status}
        row_entries.append(entry)
        if status != "passed":
            result["non_passed_rows"].append(entry)

    result["rows"] = row_entries

    for provider in ("qwen-code", "claude-code"):
        provider_rows = [item for item in row_entries if item["provider"] == provider]
        total = len(provider_rows)
        passed = sum(1 for item in provider_rows if str(item.get("status", "")).strip() == "passed")
        summary_status = "missing"
        if total > 0:
            summary_status = "passed" if passed == total else "failed"
        if has_run_dimension and expected_runs is not None and expected_runs > total:
            summary_status = "failed" if total > 0 else "missing"
        result["by_provider"][provider] = {"status": summary_status, "passed": passed, "total": total}

    if has_run_dimension and expected_runs is not None:
        for provider in ("qwen-code", "claude-code"):
            provider_rows = [item for item in row_entries if item["provider"] == provider]
            provider_run_indexes = {
                int(item["run_index"])
                for item in provider_rows
                if isinstance(item.get("run_index"), int)
            }
            for run_index in range(1, expected_runs + 1):
                if run_index not in provider_run_indexes:
                    result["non_passed_rows"].append(
                        {"provider": provider, "run_index": run_index, "status": "missing"}
                    )

    return result


def parse_documentation_audit(path: Path) -> dict[str, str]:
    result = {
        "auto_status": "missing",
        "manual_status": "missing",
        "implementation_audit_status": "missing",
    }
    if not path.exists():
        return result
    text = path.read_text(encoding="utf-8")
    for key in ("auto_status", "manual_status", "implementation_audit_status"):
        match = re.search(rf"^- {re.escape(key)}:\s*(.+)$", text, flags=re.MULTILINE)
        if match:
            result[key] = match.group(1).strip()
    return result


def parse_backend_stats(tsv_path: Path) -> dict[str, object]:
    stats: dict[str, object] = {
        "hard": 0,
        "total": 0,
        "runtime_parse": 0,
        "runner_unavailable": 0,
        "runtime_timeout": 0,
        "infra_signal_terminated": 0,
        "infra_incomplete_cycle": 0,
        "quality_gates_failed": 0,
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
        "issues_counter": Counter(),
    }

    if not tsv_path.exists():
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
        if field_bool(parts, "hard_pass"):
            stats["hard"] = int(stats["hard"]) + 1

        for key in (
            "runtime_parse",
            "runner_unavailable",
            "runtime_timeout",
            "infra_signal_terminated",
            "infra_incomplete_cycle",
            "quality_gates_failed",
            "summary_missing",
            "precheck_failed",
            "cancellation_like",
            "runtime_flow_failed",
            "semantic_hard_fail",
        ):
            if field_bool(parts, key):
                stats[key] = int(stats[key]) + 1

        stats["off_topic_hits"] = int(stats["off_topic_hits"]) + field_int(parts, "off_topic_hits", 0)

        artifact_source = field(parts, "artifact_source", "")
        if artifact_source != "snapshot":
            stats["artifact_non_snapshot"] = int(stats["artifact_non_snapshot"]) + 1

        issues_raw = field(parts, "issues", "")
        issue_tags = [tag.strip() for tag in issues_raw.split(",") if tag.strip() and tag.strip() != "-"]
        runtime_flow_hit = False
        for tag in issue_tags:
            counter: Counter = stats["issues_counter"]  # type: ignore[assignment]
            counter.update([tag])
            if tag == "analysis:evidence-scope":
                stats["evidence_scope_hits"] = int(stats["evidence_scope_hits"]) + 1
            if tag == "analysis:cross-repo-missing":
                stats["cross_repo_missing_hits"] = int(stats["cross_repo_missing_hits"]) + 1
            if tag.startswith("runtime:"):
                runtime_flow_hit = True
        if runtime_flow_hit:
            stats["runtime_flow_issue_hits"] = int(stats["runtime_flow_issue_hits"]) + 1

    return stats


def strict_blockers(
    rec: dict[str, object],
    stats: dict[str, object],
    frontend_init: dict[str, object],
    frontend_cancel: dict[str, object],
    documentation_audit: dict[str, str],
) -> list[str]:
    reasons: list[str] = []

    if str(rec.get("status", "")).strip() != "passed":
        reasons.append(f"batch_status={rec.get('status', 'missing')}")

    if int(stats["total"]) != 10:
        reasons.append(f"backend_total_runs={stats['total']} (expected 10)")
    if int(stats["hard"]) != 10:
        reasons.append(f"backend_hard_pass={stats['hard']} (expected 10)")

    for key in (
        "runtime_parse",
        "runner_unavailable",
        "runtime_timeout",
        "infra_signal_terminated",
        "infra_incomplete_cycle",
        "quality_gates_failed",
        "summary_missing",
        "precheck_failed",
    ):
        if int(stats[key]) != 0:
            reasons.append(f"{key}={stats[key]} (expected 0)")

    if int(stats["semantic_hard_fail"]) != 0:
        reasons.append(f"semantic_hard_fail={stats['semantic_hard_fail']} (expected 0)")
    if int(stats["off_topic_hits"]) != 0:
        reasons.append(f"off_topic_hits={stats['off_topic_hits']} (expected 0)")
    if int(stats["artifact_non_snapshot"]) != 0:
        reasons.append(f"artifact_source_non_snapshot={stats['artifact_non_snapshot']} (expected 0)")
    if int(stats["evidence_scope_hits"]) != 0:
        reasons.append(f"analysis:evidence-scope hits={stats['evidence_scope_hits']} (expected 0)")
    if int(stats["cross_repo_missing_hits"]) != 0:
        reasons.append(f"analysis:cross-repo-missing hits={stats['cross_repo_missing_hits']} (expected 0)")

    runtime_flow_violations = int(stats["runtime_flow_issue_hits"]) + int(stats["runtime_flow_failed"])
    if runtime_flow_violations != 0:
        reasons.append(
            "runtime_flow_violations="
            f"issue_hits:{stats['runtime_flow_issue_hits']} runtime_flow_failed:{stats['runtime_flow_failed']} (expected 0)"
        )

    init_non_passed = frontend_init.get("non_passed_rows")
    if isinstance(init_non_passed, list) and init_non_passed:
        sample = []
        for item in init_non_passed[:6]:
            if not isinstance(item, dict):
                continue
            sample.append(
                f"{item.get('provider', '-')}#run{item.get('run_index', '-')}={item.get('status', 'missing')}"
            )
        reasons.append(
            f"frontend_init_non_passed={len(init_non_passed)} (expected 0); sample={', '.join(sample) if sample else '-'}"
        )

    init_by_provider = frontend_init.get("by_provider")
    if isinstance(init_by_provider, dict):
        for provider in ("qwen-code", "claude-code"):
            payload = init_by_provider.get(provider)
            if not isinstance(payload, dict):
                reasons.append(f"frontend_init_{provider}_status=missing (expected passed)")
                continue
            status = str(payload.get("status", "missing")).strip() or "missing"
            if status != "passed":
                reasons.append(f"frontend_init_{provider}_status={status} (expected passed)")

    cancel_by_provider = frontend_cancel.get("by_provider")
    if isinstance(cancel_by_provider, dict):
        for provider in ("qwen-code", "claude-code"):
            payload = cancel_by_provider.get(provider)
            if not isinstance(payload, dict):
                reasons.append(f"frontend_cancel_{provider}_status=missing (expected passed)")
                continue
            status = str(payload.get("status", "missing")).strip() or "missing"
            if status != "passed":
                reasons.append(f"frontend_cancel_{provider}_status={status} (expected passed)")

    if documentation_audit.get("auto_status") != "passed":
        reasons.append(f"documentation_audit_auto_status={documentation_audit.get('auto_status', 'missing')} (expected passed)")
    if documentation_audit.get("manual_status") != "passed":
        reasons.append(f"documentation_audit_manual_status={documentation_audit.get('manual_status', 'missing')} (expected passed)")

    return reasons


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
    "execution_repo_selection",
    "status",
    "strict_status",
    "backend_hard_pass",
    "backend_total_runs",
    "semantic_hard_fail_runs",
    "off_topic_hits",
    "artifact_non_snapshot_runs",
    "runtime_parse_failures",
    "runner_unavailable_failures",
    "runtime_timeout_failures",
    "infra_signal_terminated_failures",
    "infra_incomplete_cycle_failures",
    "quality_gates_failed_failures",
    "summary_missing_failures",
    "precheck_failed_failures",
    "runtime_flow_failed_runs",
    "evidence_scope_hits",
    "cross_repo_missing_hits",
    "runtime_flow_issue_hits",
    "frontend_init_non_passed_rows",
    "frontend_init_qwen_status",
    "frontend_init_claude_status",
    "frontend_cancel_qwen_status",
    "frontend_cancel_claude_status",
    "documentation_audit_auto_status",
    "documentation_audit_manual_status",
    "documentation_audit_implementation_status",
    "blocking_reasons",
    "run_matrix_tsv",
    "quality_report_md",
    "documentation_audit_md",
]

tsv_lines = ["\t".join(header)]
md_lines = [
    "# Profile Matrix",
    "",
    "| profile_id | sweep_id | batch_id | status | strict | backend_hard/total | semantic_hard_fail | off_topic_hits | artifact_non_snapshot | evidence_scope | cross_repo_missing | runtime_flow | frontend init non-passed | frontend init (qwen/claude) | frontend cancel (qwen/claude) | documentation audit (auto/manual/impl) | blockers | run_matrix | quality_report | documentation_audit |",
    "|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---|---|---|---|---|",
]

verdict_records: list[dict[str, object]] = []
strict_fail_count = 0

for rec in records:
    run_matrix_tsv = Path(str(rec["run_matrix_tsv"]))
    frontend_matrix_md = Path(str(rec["frontend_matrix_md"]))
    frontend_cancel_matrix_md = Path(str(rec["frontend_cancel_matrix_md"]))
    documentation_audit_md = Path(str(rec["documentation_audit_md"]))

    stats = parse_backend_stats(run_matrix_tsv)
    frontend_init = parse_frontend_matrix(frontend_matrix_md, expected_runs=5)
    frontend_cancel = parse_frontend_matrix(frontend_cancel_matrix_md)
    documentation_audit = parse_documentation_audit(documentation_audit_md)

    blockers = strict_blockers(rec, stats, frontend_init, frontend_cancel, documentation_audit)
    strict_status = "passed" if not blockers else "failed"
    if blockers:
        strict_fail_count += 1

    execution = rec.get("execution") if isinstance(rec.get("execution"), dict) else {}
    execution_strategy = str((execution or {}).get("strategy", "-"))
    execution_max_parallel = str((execution or {}).get("max_parallel_tasks", "-"))
    execution_failure_policy = str((execution or {}).get("failure_policy", "-"))
    execution_shard_mode = str((execution or {}).get("shard_discovery_mode", "-"))
    execution_repo_selection = str((execution or {}).get("repo_selection", "-"))

    init_non_passed_rows = frontend_init.get("non_passed_rows")
    init_non_passed_count = len(init_non_passed_rows) if isinstance(init_non_passed_rows, list) else 0
    init_by_provider = frontend_init.get("by_provider") if isinstance(frontend_init.get("by_provider"), dict) else {}
    cancel_by_provider = (
        frontend_cancel.get("by_provider") if isinstance(frontend_cancel.get("by_provider"), dict) else {}
    )
    frontend_init_qwen_status = str((init_by_provider.get("qwen-code") or {}).get("status", "missing"))
    frontend_init_claude_status = str((init_by_provider.get("claude-code") or {}).get("status", "missing"))
    frontend_cancel_qwen_status = str((cancel_by_provider.get("qwen-code") or {}).get("status", "missing"))
    frontend_cancel_claude_status = str((cancel_by_provider.get("claude-code") or {}).get("status", "missing"))

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
                execution_repo_selection,
                str(rec["status"]),
                strict_status,
                str(stats["hard"]),
                str(stats["total"]),
                str(stats["semantic_hard_fail"]),
                str(stats["off_topic_hits"]),
                str(stats["artifact_non_snapshot"]),
                str(stats["runtime_parse"]),
                str(stats["runner_unavailable"]),
                str(stats["runtime_timeout"]),
                str(stats["infra_signal_terminated"]),
                str(stats["infra_incomplete_cycle"]),
                str(stats["quality_gates_failed"]),
                str(stats["summary_missing"]),
                str(stats["precheck_failed"]),
                str(stats["runtime_flow_failed"]),
                str(stats["evidence_scope_hits"]),
                str(stats["cross_repo_missing_hits"]),
                str(stats["runtime_flow_issue_hits"]),
                str(init_non_passed_count),
                frontend_init_qwen_status,
                frontend_init_claude_status,
                frontend_cancel_qwen_status,
                frontend_cancel_claude_status,
                documentation_audit["auto_status"],
                documentation_audit["manual_status"],
                documentation_audit["implementation_audit_status"],
                "; ".join(blockers) if blockers else "-",
                str(rec["run_matrix_tsv"]),
                str(rec["quality_report_md"]),
                str(rec["documentation_audit_md"]),
            ]
        )
    )

    md_lines.append(
        "| "
        f"{rec['profile_id']} | {rec.get('sweep_id', 'baseline')} | {rec['batch_id']} | {rec['status']} | {strict_status} | "
        f"{stats['hard']}/{stats['total']} | {stats['semantic_hard_fail']} | {stats['off_topic_hits']} | {stats['artifact_non_snapshot']} | "
        f"{stats['evidence_scope_hits']} | {stats['cross_repo_missing_hits']} | "
        f"{int(stats['runtime_flow_failed']) + int(stats['runtime_flow_issue_hits'])} | "
        f"{init_non_passed_count} | "
        f"{frontend_init_qwen_status}/{frontend_init_claude_status} | "
        f"{frontend_cancel_qwen_status}/{frontend_cancel_claude_status} | "
        f"{documentation_audit['auto_status']}/{documentation_audit['manual_status']}/{documentation_audit['implementation_audit_status']} | "
        f"{'; '.join(blockers) if blockers else '-'} | {rec['run_matrix_md']} | {rec['quality_report_md']} | {rec['documentation_audit_md']} |"
    )

    verdict_records.append(
        {
            "profile_id": rec["profile_id"],
            "sweep_id": rec.get("sweep_id", "baseline"),
            "batch_id": rec["batch_id"],
            "status": rec["status"],
            "strict_status": strict_status,
            "blocking_reasons": blockers,
            "execution": {
                "strategy": execution_strategy,
                "max_parallel_tasks": execution_max_parallel,
                "failure_policy": execution_failure_policy,
                "shard_discovery_mode": execution_shard_mode,
                "repo_selection": execution_repo_selection,
            },
            "backend": {
                "hard_pass": int(stats["hard"]),
                "total_runs": int(stats["total"]),
                "semantic_hard_fail_runs": int(stats["semantic_hard_fail"]),
                "off_topic_hits": int(stats["off_topic_hits"]),
                "artifact_non_snapshot_runs": int(stats["artifact_non_snapshot"]),
                "runtime_parse_failures": int(stats["runtime_parse"]),
                "runner_unavailable_failures": int(stats["runner_unavailable"]),
                "runtime_timeout_failures": int(stats["runtime_timeout"]),
                "infra_signal_terminated_failures": int(stats["infra_signal_terminated"]),
                "infra_incomplete_cycle_failures": int(stats["infra_incomplete_cycle"]),
                "quality_gates_failed_failures": int(stats["quality_gates_failed"]),
                "summary_missing_failures": int(stats["summary_missing"]),
                "precheck_failed_failures": int(stats["precheck_failed"]),
                "runtime_flow_failed_runs": int(stats["runtime_flow_failed"]),
                "evidence_scope_hits": int(stats["evidence_scope_hits"]),
                "cross_repo_missing_hits": int(stats["cross_repo_missing_hits"]),
                "runtime_flow_issue_hits": int(stats["runtime_flow_issue_hits"]),
            },
            "frontend": {
                "init": frontend_init,
                "cancel": frontend_cancel,
            },
            "documentation_audit": documentation_audit,
            "artifacts": {
                "run_matrix_tsv": rec["run_matrix_tsv"],
                "run_matrix_md": rec["run_matrix_md"],
                "frontend_matrix_md": rec["frontend_matrix_md"],
                "frontend_cancel_matrix_md": rec["frontend_cancel_matrix_md"],
                "quality_report_md": rec["quality_report_md"],
                "documentation_audit_md": rec["documentation_audit_md"],
                "driver_log": rec["driver_log"],
            },
        }
    )

verdict = "PASS" if strict_fail_count == 0 else "FAIL"
release_state = "RELEASE READY" if verdict == "PASS" else "RELEASE BLOCKED"

out_md.parent.mkdir(parents=True, exist_ok=True)
out_md.write_text("\n".join(md_lines) + "\n", encoding="utf-8")
out_tsv.write_text("\n".join(tsv_lines) + "\n", encoding="utf-8")

verdict_lines = [
    f"# Release Verdict: {matrix_id}",
    "",
    f"- generated_at_utc: {datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')}",
    f"- verdict: {verdict}",
    f"- release_state: {release_state}",
    f"- profile_sweep_runs: {len(verdict_records)}",
    f"- strict_pass_runs: {len(verdict_records) - strict_fail_count}",
    f"- strict_fail_runs: {strict_fail_count}",
    "",
    "## Blocking Items",
]

if strict_fail_count == 0:
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
        verdict_lines.append(f"  - quality_report: {artifacts['quality_report_md']}")
        verdict_lines.append(f"  - frontend_matrix: {artifacts['frontend_matrix_md']}")
        verdict_lines.append(f"  - frontend_cancel_matrix: {artifacts['frontend_cancel_matrix_md']}")
        verdict_lines.append(f"  - documentation_audit: {artifacts['documentation_audit_md']}")

verdict_payload = {
    "matrix_id": matrix_id,
    "generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "verdict": verdict,
    "release_state": release_state,
    "profile_sweep_runs": len(verdict_records),
    "strict_pass_runs": len(verdict_records) - strict_fail_count,
    "strict_fail_runs": strict_fail_count,
    "records": verdict_records,
}

verdict_md.write_text("\n".join(verdict_lines) + "\n", encoding="utf-8")
verdict_json.write_text(json.dumps(verdict_payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
PY

log "profile matrix markdown: $MATRIX_REPORT_MD"
log "profile matrix tsv: $MATRIX_REPORT_TSV"
log "release verdict markdown: $VERDICT_MD"
log "release verdict json: $VERDICT_JSON"

MATRIX_VERDICT="$(python3 - "$VERDICT_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding="utf-8"))
print(str(payload.get("verdict", "FAIL")).strip().upper())
PY
)"

if [[ "$MATRIX_VERDICT" != "PASS" ]]; then
  die "RELEASE BLOCKED for matrix=$MATRIX_ID (see $VERDICT_MD)"
fi

log "matrix completed successfully (RELEASE READY)"
exit 0
