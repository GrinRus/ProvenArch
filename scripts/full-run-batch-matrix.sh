#!/usr/bin/env bash
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
BATCH_SCRIPT="${BATCH_SCRIPT:-$PROVENARCH_ROOT/scripts/full-run-batch-5x2.sh}"
E2E_MATRIX_FILE="${E2E_MATRIX_FILE:-}"
MATRIX_ID="${MATRIX_ID:-matrix-$(date -u +'%Y%m%dT%H%M%SZ')}"
RUN_COUNT="${RUN_COUNT:-1}"
E2E_TMP_ROOT="${E2E_TMP_ROOT:-/tmp/provenarch-test_arch_project}"
REPORTS_ROOT="${REPORTS_ROOT:-$E2E_TMP_ROOT/reports}"
MATRIX_ROOT="${MATRIX_ROOT:-$E2E_TMP_ROOT/matrix/$MATRIX_ID}"
ACP_CLAUDE_CMD_BIN="${ACP_CLAUDE_CMD_BIN:-claude}"
ACP_QWEN_CMD_BIN="${ACP_QWEN_CMD_BIN:-qwen}"
ACP_APPLY_TIMEOUTS_VIA_API="${ACP_APPLY_TIMEOUTS_VIA_API:-1}"
E2E_MATRIX_RELEASE_MODE="${E2E_MATRIX_RELEASE_MODE:-auto}"
E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES="${E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES:-0}"
BATCH_PROVIDER_FILTER="${BATCH_PROVIDER_FILTER:-all}"
BATCH_RUN_SELECTION="${BATCH_RUN_SELECTION:-all}"
BATCH_FRONTEND_MODE="${BATCH_FRONTEND_MODE:-}"
BATCH_FRONTEND_CANCEL_MODE="${BATCH_FRONTEND_CANCEL_MODE:-}"
UI_E2E_HEADED="${UI_E2E_HEADED:-}"
MATRIX_DRIVER_LOG="${MATRIX_DRIVER_LOG:-$MATRIX_ROOT/driver.log}"
MATRIX_TIMEOUT_PROFILE_FILE="${MATRIX_TIMEOUT_PROFILE_FILE:-$MATRIX_ROOT/timeout-profile.txt}"
PROFILE_REPOS_FILE_RESOLVED=""
PROFILE_SOURCE_KIND_EFFECTIVE=""
PROFILE_EXPECTED_REPO_COUNT_RESOLVED=0
PROFILE_META_CACHE_KEY=""
PROFILE_META_CACHE_REPOS_FILE=""
PROFILE_META_CACHE_SOURCE_KIND="mixed"
PROFILE_META_CACHE_EXPECTED_REPO_COUNT=0
MATRIX_TIMEOUT_PROFILE=""
MATRIX_TIMEOUT_ENV_ASSIGNMENTS=()
declare -a MATRIX_ALL_PROVIDERS=("qwen-code" "claude-code")
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
  python3 - "$CURRENT_PROFILE_STATUS_FILE" "$status" "$failure_reason" "$CURRENT_PROFILE_ID" "$CURRENT_PROFILE_SLUG" "$CURRENT_BATCH_ID" "$CURRENT_SOURCE_KIND" "$CURRENT_EXPECTED_REPO_COUNT" "$CURRENT_REPOS_FILE" "$CURRENT_SWEEP_ID" "$CURRENT_SWEEP_STRATEGY" "$CURRENT_SWEEP_MAX_PARALLEL" "$CURRENT_SWEEP_FAILURE_POLICY" "$CURRENT_SWEEP_SHARD_MODE" "$CURRENT_BATCH_ROOT" "$CURRENT_DRIVER_LOG" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

path = Path(sys.argv[1]).resolve()
payload = {
    "profile_id": sys.argv[4],
    "profile_slug": sys.argv[5],
    "batch_id": sys.argv[6],
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
    "frontend_cancel_matrix_md": "-",
    "quality_report_md": "-",
    "driver_log": sys.argv[16],
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
  local frontend_cancel_matrix_md="$6"
  local quality_report_md="$7"
  [[ -z "$CURRENT_PROFILE_STATUS_FILE" ]] && return 0
  python3 - "$CURRENT_PROFILE_STATUS_FILE" "$status" "$failure_reason" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$frontend_cancel_matrix_md" "$quality_report_md" <<'PY'
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
payload["frontend_cancel_matrix_md"] = sys.argv[7]
payload["quality_report_md"] = sys.argv[8]
payload["updated_at"] = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
path.write_text(json.dumps(payload, ensure_ascii=True) + "\n", encoding="utf-8")
PY
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
  write_current_profile_status "failed" "infra_signal_terminated"
  exit "$(signal_exit_code "$signal_name")"
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
        qwen-code|claude-code)
          if ! array_contains "$token" "${MATRIX_SELECTED_PROVIDERS[@]-}"; then
            MATRIX_SELECTED_PROVIDERS+=("$token")
          fi
          ;;
        *)
          die "BATCH_PROVIDER_FILTER contains unsupported provider '$token' (allowed: qwen-code, claude-code, all)"
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
  python3 "$PROVENARCH_ROOT/scripts/resolve-repos-meta.py" \
    --repos-file "$repos_file" \
    --expected-repo-count "$expected_repo_count" \
    --source-kind "$source_kind" \
    --profile-id "$profile_id" \
    --out "$output_json"
  read_profile_repos_meta "$output_json"
}

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
if [[ ! -x "$BATCH_SCRIPT" ]]; then
  die "batch script is unavailable: $BATCH_SCRIPT"
fi

require_cmd bash
require_cmd python3
require_cmd "$ACP_CLAUDE_CMD_BIN"
require_cmd "$ACP_QWEN_CMD_BIN"
acp_ensure_no_legacy_env_set die
resolve_selected_providers
resolve_selected_run_indexes

mkdir -p "$MATRIX_ROOT" "$REPORTS_ROOT"
mkdir -p "$(dirname "$MATRIX_DRIVER_LOG")"
: > "$MATRIX_DRIVER_LOG"

ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES="$(normalize_binary_flag "$E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES" "E2E_MATRIX_ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES")"
if [[ "$RELEASE_MODE" == "1" ]]; then
  if [[ -z "$BATCH_FRONTEND_MODE" ]]; then
    BATCH_FRONTEND_MODE="per_run"
  fi
  if [[ -z "$BATCH_FRONTEND_CANCEL_MODE" ]]; then
    BATCH_FRONTEND_CANCEL_MODE="once_per_provider"
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
acp_log_release_guard log "$RELEASE_MODE" "$MATRIX_ID" "$ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES" "${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}"
if [[ "${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}" -gt 0 ]]; then
  acp_log_diagnostic_timeout_overrides log "${DIAGNOSTIC_TIMEOUT_OVERRIDES[*]}"
fi
if [[ "$RELEASE_MODE" == "1" && "$ALLOW_DIAGNOSTIC_TIMEOUT_OVERRIDES" != "1" && "${#DIAGNOSTIC_TIMEOUT_OVERRIDES[@]}" -gt 0 ]]; then
  die "$(acp_release_guard_blocked_message)"
fi
log "release frontend defaults: frontend_mode=${BATCH_FRONTEND_MODE:-default} frontend_cancel_mode=${BATCH_FRONTEND_CANCEL_MODE:-default} headed=${UI_E2E_HEADED:-default}"

COMBINATIONS_TSV="$MATRIX_ROOT/profile-sweep-combinations.tsv"
RECORDS_JSONL="$MATRIX_ROOT/profile-runs.jsonl"
mkdir -p "$MATRIX_STATUS_ROOT"
: > "$RECORDS_JSONL"
: > "$MATRIX_TIMEOUT_PROFILE_FILE"
trap 'on_matrix_signal TERM' TERM
trap 'on_matrix_signal INT' INT
trap 'on_matrix_signal HUP' HUP

python3 - "$E2E_MATRIX_FILE" "$COMBINATIONS_TSV" "$RELEASE_MODE" "$MATRIX_TIMEOUT_PROFILE_FILE" <<'PY'
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

while IFS=$'\t' read -r profile_id repos_file expected_repo_count source_kind sweep_id sweep_strategy sweep_max_parallel sweep_failure_policy sweep_shard_mode; do
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

  profile_meta_key="${profile_id}|${repos_file}|${expected_repo_count}|${source_kind}"
  if [[ "$profile_meta_key" != "$PROFILE_META_CACHE_KEY" ]]; then
    validate_profile_repos_meta "$profile_id" "$repos_file" "$expected_repo_count" "$source_kind" "$profile_repos_meta_json"
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
  log "running profile=$profile_id sweep=$sweep_id source_kind=$PROFILE_META_CACHE_SOURCE_KIND expected_repo_count=$PROFILE_META_CACHE_EXPECTED_REPO_COUNT batch_id=$batch_id"
  status="passed"
  if ! (
    cd "$PROVENARCH_ROOT"
    if [[ "${#MATRIX_TIMEOUT_ENV_ASSIGNMENTS[@]}" -gt 0 ]]; then
      for assignment in "${MATRIX_TIMEOUT_ENV_ASSIGNMENTS[@]}"; do
        export "$assignment"
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
    if [[ -n "$BATCH_FRONTEND_CANCEL_MODE" ]]; then
      EXTRA_BATCH_ENV_ASSIGNMENTS+=("BATCH_FRONTEND_CANCEL_MODE=$BATCH_FRONTEND_CANCEL_MODE")
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
      "ACP_APPLY_TIMEOUTS_VIA_API=$ACP_APPLY_TIMEOUTS_VIA_API" \
      "$BATCH_SCRIPT"
  ) >"$driver_log" 2>&1; then
    status="failed"
    log "profile+sweep failed: profile=$profile_id sweep=$sweep_id (see $driver_log)"
  fi
  if [[ "$status" == "passed" ]] && batch_has_incomplete_run_sentinels "$batch_root"; then
    status="failed"
    log "profile+sweep left unfinished run sentinel: profile=$profile_id sweep=$sweep_id batch_root=$batch_root"
  fi

  run_matrix_tsv="$REPORTS_ROOT/run_matrix_${batch_id}.tsv"
  run_matrix_md="$REPORTS_ROOT/run_matrix_${batch_id}.md"
  frontend_matrix_md="$REPORTS_ROOT/frontend_e2e_matrix_${batch_id}.md"
  frontend_cancel_matrix_md="$REPORTS_ROOT/frontend_cancel_e2e_matrix_${batch_id}.md"
  quality_report_md="$REPORTS_ROOT/quality_report_${batch_id}.md"
  if [[ "$status" == "passed" ]]; then
    update_current_profile_status_artifacts "$status" "none" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$frontend_cancel_matrix_md" "$quality_report_md"
  else
    update_current_profile_status_artifacts "$status" "child_failed" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$frontend_cancel_matrix_md" "$quality_report_md"
  fi

  python3 - "$RECORDS_JSONL" \
    "$profile_id" "$profile_slug" "$batch_id" "$PROFILE_META_CACHE_SOURCE_KIND" "$PROFILE_META_CACHE_EXPECTED_REPO_COUNT" "$PROFILE_META_CACHE_REPOS_FILE" "$status" \
    "$sweep_id" "$sweep_strategy" "$sweep_max_parallel" "$sweep_failure_policy" "$sweep_shard_mode" \
    "$batch_root" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$frontend_cancel_matrix_md" "$quality_report_md" "$driver_log" <<'PY'
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
    "sweep_id": sys.argv[9],
    "execution": {
        "strategy": sys.argv[10],
        "max_parallel_tasks": int(sys.argv[11]),
        "failure_policy": sys.argv[12],
        "shard_discovery_mode": sys.argv[13],
    },
    "batch_root": sys.argv[14],
    "run_matrix_tsv": sys.argv[15],
    "run_matrix_md": sys.argv[16],
    "frontend_matrix_md": sys.argv[17],
    "frontend_cancel_matrix_md": sys.argv[18],
    "quality_report_md": sys.argv[19],
    "driver_log": sys.argv[20],
}
with path.open("a", encoding="utf-8") as f:
    f.write(json.dumps(payload, ensure_ascii=True))
    f.write("\n")
PY
done < "$COMBINATIONS_TSV"
CURRENT_PROFILE_STATUS_FILE=""

MATRIX_REPORT_MD="$REPORTS_ROOT/profile_matrix_${MATRIX_ID}.md"
MATRIX_REPORT_TSV="$REPORTS_ROOT/profile_matrix_${MATRIX_ID}.tsv"
VERDICT_MD="$REPORTS_ROOT/release_verdict_${MATRIX_ID}.md"
VERDICT_JSON="$REPORTS_ROOT/release_verdict_${MATRIX_ID}.json"
if [[ "${MATRIX_TEST_TRUNCATE_RECORDS_JSONL:-0}" == "1" ]]; then
  : > "$RECORDS_JSONL"
fi

python3 - "$RECORDS_JSONL" "$MATRIX_STATUS_ROOT" "$MATRIX_REPORT_MD" "$MATRIX_REPORT_TSV" "$VERDICT_MD" "$VERDICT_JSON" "$MATRIX_ID" "$RELEASE_MODE" "$RUN_COUNT" "$MATRIX_SELECTED_PROVIDERS_CSV" "$MATRIX_SELECTED_RUN_INDEXES_CSV" <<'PY'
import json
import re
import sys
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

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
            "frontend_cancel_matrix_md",
            "quality_report_md",
            "driver_log",
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
required_release_providers = ("qwen-code", "claude-code")


def parse_frontend_status(path: Path, provider: str) -> str:
    if not path.exists() or not path.is_file():
        return "missing"
    text = path.read_text(encoding="utf-8")
    match = re.search(rf"^\|\s*{re.escape(provider)}\s*\|\s*([^|]+)\|", text, flags=re.MULTILINE)
    return match.group(1).strip() if match else "missing"


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


def normalize_scope_list(values: object) -> list[str]:
    if not isinstance(values, list):
        return []
    normalized = []
    for value in values:
        text = str(value).strip()
        if text:
            normalized.append(text)
    return sorted(set(normalized))


def collect_batch_shard_plan_signature(batch_root: Path) -> tuple[str | None, list[str]]:
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
    shard_plan_invariant: str,
    release_mode: bool,
    expected_backend_runs: int,
    required_frontend_providers: list[str],
) -> list[str]:
    reasons: list[str] = []

    if str(rec.get("status", "")).strip() != "passed":
        reasons.append(f"batch_status={rec.get('status', 'missing')}")

    if int(stats["total"]) != expected_backend_runs:
        reasons.append(f"backend_total_runs={stats['total']} (expected {expected_backend_runs})")
    if int(stats["hard"]) != expected_backend_runs:
        reasons.append(f"backend_hard_pass={stats['hard']} (expected {expected_backend_runs})")

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

    frontend_provider_keys = {
        "qwen-code": ("frontend_qwen_status", "frontend_cancel_qwen_status"),
        "claude-code": ("frontend_claude_status", "frontend_cancel_claude_status"),
    }
    for provider in required_frontend_providers:
        init_key, cancel_key = frontend_provider_keys[provider]
        if frontend.get(init_key) != "passed":
            reasons.append(f"{init_key}={frontend.get(init_key, 'missing')} (expected passed)")
        if frontend.get(cancel_key) != "passed":
            reasons.append(f"{cancel_key}={frontend.get(cancel_key, 'missing')} (expected passed)")
    if release_mode and shard_plan_invariant != "passed":
        reasons.append(f"shard_plan_invariant={shard_plan_invariant} (release requires passed)")

    return reasons


def shard_plan_invariant_status(records: list[dict[str, object]]) -> tuple[dict[str, str], dict[str, list[str]]]:
    status_by_batch: dict[str, str] = {}
    blockers_by_batch: dict[str, list[str]] = {}
    signatures_by_batch: dict[str, str | None] = {}

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
    "frontend_qwen_status",
    "frontend_claude_status",
    "frontend_cancel_qwen_status",
    "frontend_cancel_claude_status",
    "blocking_reasons",
    "run_matrix_tsv",
    "quality_report_md",
]

tsv_lines = ["\t".join(header)]
md_lines = [
    "# Profile Matrix",
    "",
    "| profile_id | sweep_id | batch_id | status | strict | shard_plan_invariant | backend_hard/total | semantic_hard_fail | off_topic_hits | artifact_non_snapshot | evidence_scope | cross_repo_missing | runtime_flow | frontend init (qwen/claude) | frontend cancel (qwen/claude) | blockers | run_matrix | quality_report |",
    "|---|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|---|---|---|---|",
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

for rec in records:
    run_matrix_tsv = Path(str(rec["run_matrix_tsv"]))
    frontend_matrix_md = Path(str(rec["frontend_matrix_md"]))
    frontend_cancel_matrix_md = Path(str(rec["frontend_cancel_matrix_md"]))

    stats = parse_backend_stats(run_matrix_tsv)
    frontend_statuses = {
        "frontend_qwen_status": parse_frontend_status(frontend_matrix_md, "qwen-code"),
        "frontend_claude_status": parse_frontend_status(frontend_matrix_md, "claude-code"),
        "frontend_cancel_qwen_status": parse_frontend_status(frontend_cancel_matrix_md, "qwen-code"),
        "frontend_cancel_claude_status": parse_frontend_status(frontend_cancel_matrix_md, "claude-code"),
    }

    shard_plan_invariant = invariant_status_by_batch.get(str(rec["batch_id"]), "not_compared")
    blockers = strict_blockers(
        rec,
        stats,
        frontend_statuses,
        shard_plan_invariant,
        release_mode,
        expected_backend_runs,
        required_frontend_providers,
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
                frontend_statuses["frontend_qwen_status"],
                frontend_statuses["frontend_claude_status"],
                frontend_statuses["frontend_cancel_qwen_status"],
                frontend_statuses["frontend_cancel_claude_status"],
                "; ".join(blockers) if blockers else "-",
                str(rec["run_matrix_tsv"]),
                str(rec["quality_report_md"]),
            ]
        )
    )

    md_lines.append(
        "| "
        f"{rec['profile_id']} | {rec.get('sweep_id', 'baseline')} | {rec['batch_id']} | {rec['status']} | {strict_status} | "
        f"{shard_plan_invariant} | "
        f"{stats['hard']}/{stats['total']} | {stats['semantic_hard_fail']} | {stats['off_topic_hits']} | {stats['artifact_non_snapshot']} | "
        f"{stats['evidence_scope_hits']} | {stats['cross_repo_missing_hits']} | "
        f"{int(stats['runtime_flow_failed']) + int(stats['runtime_flow_issue_hits'])} | "
        f"{frontend_statuses['frontend_qwen_status']}/{frontend_statuses['frontend_claude_status']} | "
        f"{frontend_statuses['frontend_cancel_qwen_status']}/{frontend_statuses['frontend_cancel_claude_status']} | "
        f"{'; '.join(blockers) if blockers else '-'} | {rec['run_matrix_md']} | {rec['quality_report_md']} |"
    )

    verdict_records.append(
        {
            "profile_id": rec["profile_id"],
            "sweep_id": rec.get("sweep_id", "baseline"),
            "batch_id": rec["batch_id"],
            "status": rec["status"],
            "strict_status": strict_status,
            "blocking_reasons": blockers,
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
            "frontend": frontend_statuses,
            "artifacts": {
                "run_matrix_tsv": rec["run_matrix_tsv"],
                "run_matrix_md": rec["run_matrix_md"],
                "frontend_matrix_md": rec["frontend_matrix_md"],
                "frontend_cancel_matrix_md": rec["frontend_cancel_matrix_md"],
                "quality_report_md": rec["quality_report_md"],
                "driver_log": rec["driver_log"],
            },
        }
    )

release_contract_failed = release_mode and release_contract_status != "passed"
verdict = "PASS" if strict_fail_count == 0 and not release_contract_failed else "FAIL"
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
    f"- release_mode: {'1' if release_mode else '0'}",
    f"- release_contract_status: {release_contract_status}",
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
        verdict_lines.append(f"  - quality_report: {artifacts['quality_report_md']}")
        verdict_lines.append(f"  - frontend_matrix: {artifacts['frontend_matrix_md']}")
        verdict_lines.append(f"  - frontend_cancel_matrix: {artifacts['frontend_cancel_matrix_md']}")
    if release_contract_failed:
        verdict_lines.append("- release_contract:")
        for blocker in release_contract_blockers:
            verdict_lines.append(f"  - {blocker}")

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
    "generated_at_utc": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "verdict": verdict,
    "release_state": release_state,
    "profile_sweep_runs": len(verdict_records),
    "strict_pass_runs": len(verdict_records) - strict_fail_count,
    "strict_fail_runs": strict_fail_count,
    "release_contract": {
        "mode": "release" if release_mode else "non-release",
        "required_sweeps": required_sweeps,
        "observed_sweeps": observed_sweeps,
        "expected_profile_sweep_runs": expected_profile_sweep_runs,
        "observed_profile_sweep_runs": observed_profile_sweep_runs,
        "selected_providers": selected_providers,
        "selected_run_indexes": selected_run_indexes,
        "expected_backend_runs_per_profile_sweep": expected_backend_runs,
        "required_profiles": list(required_release_profiles) if release_mode else [],
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
