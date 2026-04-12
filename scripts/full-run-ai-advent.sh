#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
TARGET_REPO="${TARGET_REPO:-}"
TARGET_REPOS_FILE="${TARGET_REPOS_FILE:-}"
TARGET_REPO_GIT_URL="${TARGET_REPO_GIT_URL:-}"
TARGET_REPO_NAME="${TARGET_REPO_NAME:-}"
TARGET_REPO_REF="${TARGET_REPO_REF:-}"
PROFILE_ID="${PROFILE_ID:-}"
PROFILE_SOURCE_KIND="${PROFILE_SOURCE_KIND:-}"
EXPECTED_REPO_COUNT="${EXPECTED_REPO_COUNT:-}"
TMP_ROOT="${TMP_ROOT:-}"
KEEP_TMP="${KEEP_TMP:-0}"
ITERATIONS="${ITERATIONS:-1}"
RUN_QUALITY_GATES="${RUN_QUALITY_GATES:-1}"
READY_TIMEOUT_SEC="${READY_TIMEOUT_SEC:-60}"
RUN_LOGS_TTL_HOURS="${RUN_LOGS_TTL_HOURS:-168}"
RUN_LOGS_MAX_RUNS="${RUN_LOGS_MAX_RUNS:-200}"

CREATED_TMP=0
SERVER_PID=""
FAILURE_REASON=""
TERMINATION_SIGNAL=""
API_SIM_STATUS="not_started"
API_INIT_RUN_ID=""
API_INIT_FINAL_STATUS=""
QUALITY_GATES_STATUS="not_run"
LAST_SIGNAL=""
HEADLESS_PROVIDER=""
HEADLESS_CMD=""
TARGET_INPUT_MODE=""
TARGET_PROFILE="generic"
RESOLVED_TARGET_REPOS_FILE=""
TARGET_REPOS_META_JSON=""
EXPECTED_RUNS=0
COMPLETED_RUNS=0
EXPECTED_HEADLESS_RUNS=0
COMPLETED_HEADLESS_RUNS=0
RUNNING_RUNS_DETECTED=0
SUMMARY_RESULT=""

if [[ ! "$ITERATIONS" =~ ^[1-9][0-9]*$ ]]; then
  echo "ITERATIONS must be a positive integer, got: $ITERATIONS" >&2
  exit 1
fi
if [[ "$KEEP_TMP" != "0" && "$KEEP_TMP" != "1" ]]; then
  echo "KEEP_TMP must be 0 or 1, got: $KEEP_TMP" >&2
  exit 1
fi
if [[ "$RUN_QUALITY_GATES" != "0" && "$RUN_QUALITY_GATES" != "1" ]]; then
  echo "RUN_QUALITY_GATES must be 0 or 1, got: $RUN_QUALITY_GATES" >&2
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
EXPECTED_RUNS=$((ITERATIONS * 4))
EXPECTED_HEADLESS_RUNS=$((ITERATIONS * 2))

if [[ -z "$TMP_ROOT" ]]; then
  TMP_ROOT="$(mktemp -d -t provenarch-ai-advent.XXXXXX)"
  CREATED_TMP=1
else
  mkdir -p "$TMP_ROOT"
fi

WORKSPACE="$TMP_ROOT/arch-workspace"
LOG_DIR="$TMP_ROOT/logs"
SNAPSHOT_DIR="$TMP_ROOT/snapshots"
SUMMARY_PATH="$TMP_ROOT/session-summary.md"
FULL_RUN_LOG="$TMP_ROOT/full-run.log"
RUN_RESULTS_TSV="$TMP_ROOT/run-results.tsv"
QUALITY_LOG="$TMP_ROOT/quality-gates.log"
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
exec 3>&1 4>&2
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
  FAILURE_REASON="$1"
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
  log "received termination signal: $signal_name"
  exit 1
}

refresh_runtime_cycle_metrics() {
  if [[ -f "$RUN_RESULTS_TSV" ]]; then
    COMPLETED_RUNS="$(awk 'NF { count++ } END { print count+0 }' "$RUN_RESULTS_TSV")"
    COMPLETED_HEADLESS_RUNS="$(awk -F'\t' 'NF && $2 == "headless" { count++ } END { print count+0 }' "$RUN_RESULTS_TSV")"
  else
    COMPLETED_RUNS=0
    COMPLETED_HEADLESS_RUNS=0
  fi

  local run_history_path="$WORKSPACE/reports/taskruns/run-history.json"
  if [[ -f "$run_history_path" ]]; then
    RUNNING_RUNS_DETECTED="$(python3 - "$run_history_path" <<'PY'
import json
import sys

path = sys.argv[1]
payload = json.load(open(path, encoding='utf-8'))
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
)"
  else
    RUNNING_RUNS_DETECTED=0
  fi
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

  if [[ ! -f "$RUN_RESULTS_TSV" ]]; then
    log "completion invariant failed: missing $RUN_RESULTS_TSV"
    failed=1
  else
    for iteration in $(seq 1 "$ITERATIONS"); do
      if ! awk -F'\t' -v iter="$iteration" '$1 == iter && $2 == "headless" && $4 == "init" { ok=1 } END { exit ok ? 0 : 1 }' "$RUN_RESULTS_TSV"; then
        log "completion invariant failed: missing headless init for iteration=$iteration"
        failed=1
      fi
      if ! awk -F'\t' -v iter="$iteration" '$1 == iter && $2 == "headless" && $4 == "refresh" { ok=1 } END { exit ok ? 0 : 1 }' "$RUN_RESULTS_TSV"; then
        log "completion invariant failed: missing headless refresh for iteration=$iteration"
        failed=1
      fi
    done
  fi

  local run_history_path="$WORKSPACE/reports/taskruns/run-history.json"
  if [[ ! -f "$run_history_path" ]]; then
    log "completion invariant failed: missing run history $run_history_path"
    failed=1
  elif (( RUNNING_RUNS_DETECTED > 0 )); then
    log "completion invariant failed: detected running runs in history ($RUNNING_RUNS_DETECTED)"
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
  local generated_dir="$TMP_ROOT/generated-inputs"
  mkdir -p "$generated_dir"

  if [[ -n "$TARGET_REPOS_FILE" ]]; then
    if [[ ! -f "$TARGET_REPOS_FILE" ]]; then
      die "TARGET_REPOS_FILE does not exist: $TARGET_REPOS_FILE"
    fi
    TARGET_INPUT_MODE="repos-file"
    RESOLVED_TARGET_REPOS_FILE="$(cd "$(dirname "$TARGET_REPOS_FILE")" && pwd)/$(basename "$TARGET_REPOS_FILE")"
    return 0
  fi

  if [[ -n "$TARGET_REPO" ]]; then
    if [[ ! -d "$TARGET_REPO" ]]; then
      die "TARGET_REPO does not exist: $TARGET_REPO"
    fi
    local repo_abs
    repo_abs="$(cd "$TARGET_REPO" && pwd)"
    local repo_name
    repo_name="$(basename "$repo_abs")"
    TARGET_INPUT_MODE="legacy-single-path"
    RESOLVED_TARGET_REPOS_FILE="$generated_dir/legacy-single-path.repos.yaml"
    cat >"$RESOLVED_TARGET_REPOS_FILE" <<EOF
version: 1
repos:
  - name: ${repo_name}
    path: ${repo_abs}
docs:
  imports_path: ./docs/imports
EOF
    return 0
  fi

  if [[ -n "$TARGET_REPO_GIT_URL" ]]; then
    if [[ -z "$TARGET_REPO_NAME" ]]; then
      die "TARGET_REPO_NAME is required when TARGET_REPO_GIT_URL is set"
    fi
    if [[ -z "$TARGET_REPO_REF" ]]; then
      die "TARGET_REPO_REF is required when TARGET_REPO_GIT_URL is set (pinned ref policy)"
    fi
    TARGET_INPUT_MODE="legacy-single-git-url"
    RESOLVED_TARGET_REPOS_FILE="$generated_dir/legacy-single-git-url.repos.yaml"
    cat >"$RESOLVED_TARGET_REPOS_FILE" <<EOF
version: 1
repos:
  - name: ${TARGET_REPO_NAME}
    git_url: ${TARGET_REPO_GIT_URL}
    ref: ${TARGET_REPO_REF}
docs:
  imports_path: ./docs/imports
EOF
    return 0
  fi

  die "missing target input: set TARGET_REPOS_FILE (canonical) or legacy TARGET_REPO / TARGET_REPO_GIT_URL+TARGET_REPO_NAME+TARGET_REPO_REF"
}

validate_target_repos_file() {
  TARGET_REPOS_META_JSON="$TMP_ROOT/target-repos-meta.json"
  python3 - "$RESOLVED_TARGET_REPOS_FILE" "$EXPECTED_REPO_COUNT" "$PROFILE_SOURCE_KIND" "$PROFILE_ID" "$TARGET_REPOS_META_JSON" <<'PY'
import json
import os
import sys
from pathlib import Path

try:
    import yaml  # type: ignore
except Exception as exc:  # pragma: no cover
    raise SystemExit(f"PyYAML is required for parsing repos file: {exc}")

repos_file = Path(sys.argv[1]).resolve()
expected_raw = (sys.argv[2] or "").strip()
source_kind = (sys.argv[3] or "").strip()
profile_id = (sys.argv[4] or "").strip()
meta_path = Path(sys.argv[5]).resolve()

allowed_kinds = {"", "path", "git_url"}
if source_kind not in allowed_kinds:
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
        source = "path"
        path_value = Path(path_raw)
        abs_path = (repos_file.parent / path_value).resolve() if not path_value.is_absolute() else path_value.resolve()
        if not abs_path.exists():
            raise SystemExit(f"repos[{idx}] path does not exist: {abs_path}")
        if not abs_path.is_dir():
            raise SystemExit(f"repos[{idx}] path is not a directory: {abs_path}")
        entry = {
            "name": name,
            "source": source,
            "path": str(abs_path),
            "ref": ref,
        }
    else:
        source = "git_url"
        entry = {
            "name": name,
            "source": source,
            "git_url": git_url,
            "ref": ref,
        }
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

target_profile = "generic"
for repo in declared:
    haystack = " ".join(
        part for part in [repo.get("name", ""), repo.get("path", ""), repo.get("git_url", ""), repo.get("ref", "")] if part
    ).lower()
    if "ai_advent_challenge_new" in haystack:
        target_profile = "ai-advent"
        break

meta = {
    "repos_file": str(repos_file),
    "profile_id": profile_id or "adhoc",
    "profile_source_kind": source_kind or "mixed",
    "expected_repo_count": expected_count,
    "target_profile": target_profile,
    "declared_repos": declared,
}
meta_path.parent.mkdir(parents=True, exist_ok=True)
meta_path.write_text(json.dumps(meta, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
PY
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
  deadline=$((SECONDS + READY_TIMEOUT_SEC))
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

snapshot_run_artifacts() {
  local run_id="$1"
  local runtime="$2"
  local pipeline="$3"
  local iteration="$4"

  local dst="$SNAPSHOT_DIR/$run_id"
  mkdir -p "$dst"

  copy_if_exists "$WORKSPACE/reports/as-is/overview.md" "$dst/reports/as-is/overview.md"
  copy_if_exists "$WORKSPACE/reports/findings/findings.md" "$dst/reports/findings/findings.md"
  copy_if_exists "$WORKSPACE/reports/coverage/summary.md" "$dst/reports/coverage/summary.md"
  copy_if_exists "$WORKSPACE/reports/coverage/open-questions.md" "$dst/reports/coverage/open-questions.md"
  copy_if_exists "$WORKSPACE/reports/taskruns/${run_id}.json" "$dst/reports/taskruns/${run_id}.json"
  copy_if_exists "$WORKSPACE/reports/taskruns/${run_id}-quality.json" "$dst/reports/taskruns/${run_id}-quality.json"
  for taskrun_json in "$WORKSPACE/reports/taskruns/${run_id}-"*.json; do
    if [[ -f "$taskrun_json" ]]; then
      copy_if_exists "$taskrun_json" "$dst/reports/taskruns/$(basename "$taskrun_json")"
    fi
  done

  local run_slug
  run_slug="$(slugify "$run_id")"
  copy_if_exists "$WORKSPACE/reports/taskruns/logs/${run_slug}.ndjson" "$dst/reports/taskruns/logs/${run_slug}.ndjson"

  cat > "$dst/snapshot-meta.txt" <<META
iteration=$iteration
runtime=$runtime
pipeline=$pipeline
run_id=$run_id
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
changeset = metric('changeset_ops')
findings = metric('findings_added')
questions = metric('questions_count')
coverage_observed = metric('coverage_observed')
coverage_missing = metric('coverage_missing')
warnings = metric('warnings_count')
entity_upserts = metric('entity_upserts')
edge_upserts = metric('edge_upserts')

runtime_blob = ",".join(str(item) for item in runtime_versions)
runtime_lower = runtime_blob.lower()
mock_flag = 1 if ('mock' in runtime_lower or 'fake' in runtime_lower) else 0

signal_components = changeset + findings + questions + coverage_observed + coverage_missing + entity_upserts + edge_upserts
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
    str(changeset),
    str(findings),
    str(questions),
    str(coverage_observed),
    str(coverage_missing),
    str(warnings),
    str(domain_collect_steps),
    str(mock_flag),
    str(zero_signal),
    runtime_blob,
]))
PY
}

check_ai_advent_text_signal() {
  local run_id="$1"
  python3 - "$WORKSPACE" "$run_id" <<'PY'
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
  local run_id="$1"
  python3 - "$WORKSPACE" "$run_id" <<'PY'
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
if owner_gap and re.search(r"no findings reported\.", findings_text, flags=re.IGNORECASE):
    print(f"semantic quality failed for run {run_id}: owner-related gap exists but findings report is empty")
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
taskrun_glob = os.path.join(workspace, "reports", "taskruns", f"{run_id}-refresh-step1-collect-*.json")
for taskrun_path in sorted(glob.glob(taskrun_glob)):
    try:
        payload = json.load(open(taskrun_path, encoding="utf-8"))
    except Exception:
        continue
    warnings = payload.get("warnings") or []
    if any(critical_marker in str(item) for item in warnings):
        print(f"semantic quality failed for run {run_id}: critical off-topic drift marker found in {taskrun_path}")
        sys.exit(6)

print(
    "semantic_quality_ok "
    f"owner_gap={int(owner_gap)} "
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
  local previous_signal="$5"

  local runtime_label="$runtime_mode"
  if [[ "$runtime_mode" == "headless" ]]; then
    runtime_label="${runtime_mode}:${runtime_provider}"
  fi

  local output_path="$LOG_DIR/run-iter${iteration}-${runtime_mode}-${runtime_provider}-${pipeline}.log"
  local quality_path
  local run_id
  local status

  log "run: iteration=$iteration runtime=$runtime_label pipeline=$pipeline"
  local run_cmd=(
    "$ACP_BIN" run
    --workspace "$WORKSPACE"
    --pipeline "$pipeline"
    --runtime "$runtime_mode"
    --non-interactive
    --run-logs-ttl-hours "$RUN_LOGS_TTL_HOURS"
    --run-logs-max-runs "$RUN_LOGS_MAX_RUNS"
  )
  if [[ "$runtime_mode" == "headless" ]]; then
    run_cmd+=(--runtime-provider "$runtime_provider")
  fi

  if ! "${run_cmd[@]}" >"$output_path" 2>&1; then
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
    die "unexpected run status for $run_id: $status"
  fi

  quality_path="$WORKSPACE/reports/taskruns/${run_id}-quality.json"
  if [[ ! -f "$quality_path" ]]; then
    die "missing quality summary for run $run_id at $quality_path"
  fi

  local metrics
  metrics="$(quality_metrics "$quality_path")"

  local quality_status signal_score changeset findings questions coverage_observed coverage_missing warnings
  local domain_collect_steps mock_flag zero_signal runtime_versions
  IFS=$'\t' read -r quality_status signal_score changeset findings questions coverage_observed coverage_missing warnings domain_collect_steps mock_flag zero_signal runtime_versions <<<"$metrics"

  if [[ "$quality_status" != "succeeded" ]]; then
    die "quality summary status is not succeeded for run $run_id: $quality_status"
  fi

  if [[ "$runtime_mode" == "headless" ]]; then
    if [[ "$mock_flag" == "1" ]]; then
      die "headless run $run_id uses mock/fake runtime version ($runtime_versions)"
    fi
    if [[ "$zero_signal" == "1" ]]; then
      die "headless run $run_id produced zero-signal quality summary"
    fi
    if [[ "$TARGET_PROFILE" == "ai-advent" && "$domain_collect_steps" -le 0 ]]; then
      die "headless run $run_id has no domain collect signal in quality summary"
    fi
  fi

  if [[ "$runtime_mode" == "headless" && "$pipeline" == "refresh" ]]; then
    if [[ -n "$previous_signal" ]] && (( signal_score < previous_signal )); then
      die "quality regression: last run signal ($signal_score) is lower than previous run signal ($previous_signal) in iteration $iteration"
    fi
    check_headless_refresh_semantic_quality "$run_id" || die "headless refresh semantic quality checks failed for run $run_id"
    if [[ "$TARGET_PROFILE" == "ai-advent" ]]; then
      check_ai_advent_text_signal "$run_id" || die "ai-advent textual quality check failed for run $run_id"
    fi
  fi

  snapshot_run_artifacts "$run_id" "$runtime_label" "$pipeline" "$iteration"

  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "$status" "$signal_score" "$changeset" "$findings" "$questions" "$coverage_observed" "$coverage_missing" "$warnings" "$runtime_versions" "$quality_path" "$output_path" >> "$RUN_RESULTS_TSV"

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
    echo "- target_input_mode: ${TARGET_INPUT_MODE:-unknown}"
    echo "- target_repos_file: ${RESOLVED_TARGET_REPOS_FILE:-unset}"
    if [[ -n "$TARGET_REPO" ]]; then
      echo "- target_repo_legacy_path: $TARGET_REPO"
    fi
    if [[ -n "$TARGET_REPO_GIT_URL" ]]; then
      echo "- target_repo_legacy_git_url: $TARGET_REPO_GIT_URL"
      echo "- target_repo_legacy_name: ${TARGET_REPO_NAME:-unset}"
      echo "- target_repo_legacy_ref: ${TARGET_REPO_REF:-unset}"
    fi
    echo "- profile_id: ${PROFILE_ID:-adhoc}"
    echo "- profile_source_kind: ${PROFILE_SOURCE_KIND:-mixed}"
    echo "- expected_repo_count: ${EXPECTED_REPO_COUNT:-auto}"
    echo "- target_profile: $TARGET_PROFILE"
    echo "- workspace: $WORKSPACE"
    echo "- tmp_root: $TMP_ROOT"
    echo "- full_run_log: $FULL_RUN_LOG"
    echo "- iterations: $ITERATIONS"
    echo "- run_logs_ttl_hours: $RUN_LOGS_TTL_HOURS"
    echo "- run_logs_max_runs: $RUN_LOGS_MAX_RUNS"
    echo "- keep_tmp: $KEEP_TMP"
    echo "- expected_runs: $EXPECTED_RUNS"
    echo "- completed_runs: $COMPLETED_RUNS"
    echo "- expected_headless_runs: $EXPECTED_HEADLESS_RUNS"
    echo "- completed_headless_runs: $COMPLETED_HEADLESS_RUNS"
    echo "- running_runs_detected: $RUNNING_RUNS_DETECTED"
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
      echo "| iteration | runtime_mode | runtime_provider | pipeline | run_id | status | signal | changeset | findings | questions | cov_obs | cov_missing | warnings |"
      echo "|---|---|---|---|---|---|---|---|---|---|---|---|---|"
      while IFS=$'\t' read -r iter runtime_mode runtime_provider pipeline run_id status signal changeset findings questions cov_obs cov_missing warnings _runtime_versions _quality_path _run_log; do
        echo "| $iter | $runtime_mode | $runtime_provider | $pipeline | $run_id | $status | $signal | $changeset | $findings | $questions | $cov_obs | $cov_missing | $warnings |"
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
    echo "- $WORKSPACE/reports/as-is/overview.md"
    echo "- $WORKSPACE/reports/findings/findings.md"
    echo "- $WORKSPACE/reports/coverage/summary.md"
    echo "- $WORKSPACE/reports/coverage/open-questions.md"
    echo "- $WORKSPACE/reports/taskruns/run-history.json"
    echo "- $WORKSPACE/reports/taskruns/logs/"
    echo "- $SNAPSHOT_DIR"
    if [[ "$QUALITY_GATES_STATUS" == "passed" ]]; then
      echo "- quality_gates: passed ($QUALITY_LOG)"
    elif [[ "$QUALITY_GATES_STATUS" == "failed" ]]; then
      echo "- quality_gates: failed ($QUALITY_LOG)"
    else
      echo "- quality_gates: skipped"
    fi
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
}

cleanup() {
  local exit_code="$1"
  stop_server
  write_summary "$exit_code"

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
  rm -rf "$TMP_ROOT"
  log "temporary artifacts removed (set KEEP_TMP=1 to keep)"
}
trap 'on_termination_signal TERM' TERM
trap 'on_termination_signal INT' INT
trap 'on_termination_signal HUP' HUP
trap 'on_termination_signal PIPE' PIPE
trap 'cleanup $?' EXIT

require_cmd git "Install git and ensure it is available in PATH."
require_cmd go "Install Go 1.20+ and ensure it is available in PATH."
require_cmd npm "Install Node.js/npm and ensure it is available in PATH."
require_cmd make "Install make and ensure it is available in PATH."
require_cmd curl "Install curl and ensure it is available in PATH."
require_cmd python3 "Install python3 and ensure it is available in PATH."

if [[ ! -d "$PROVENARCH_ROOT" ]]; then
  die "PROVENARCH_ROOT does not exist: $PROVENARCH_ROOT"
fi

prepare_target_repos_file
validate_target_repos_file
TARGET_PROFILE="$(python3 - "$TARGET_REPOS_META_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding='utf-8'))
print(payload.get("target_profile", "generic"))
PY
)"
log "target input resolved: mode=$TARGET_INPUT_MODE repos_file=$RESOLVED_TARGET_REPOS_FILE profile=$TARGET_PROFILE"

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
  *)
    die "unsupported ACP_RUNTIME_PROVIDER '$HEADLESS_PROVIDER' (allowed: claude-code, qwen-code)"
    ;;
esac
if ! command -v "$HEADLESS_CMD" >/dev/null 2>&1; then
  die "headless runtime command '$HEADLESS_CMD' is unavailable for provider '$HEADLESS_PROVIDER'. Install command or set ACP_CLAUDE_CMD/ACP_QWEN_CMD"
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

log "bootstrap workspace in tmp"
"$ACP_BIN" init-workspace \
  --workspace "$WORKSPACE" \
  --repos-file "$RESOLVED_TARGET_REPOS_FILE" >"$LOG_DIR/init-workspace.log" 2>&1

if [[ ! -f "$WORKSPACE/workspace.yaml" ]]; then
  die "workspace bootstrap failed: missing $WORKSPACE/workspace.yaml"
fi
if [[ ! -d "$WORKSPACE/.git" ]]; then
  die "workspace bootstrap failed: missing $WORKSPACE/.git"
fi
if [[ ! -f "$WORKSPACE/skills/subagents.yaml" ]]; then
  die "workspace bootstrap failed: missing baseline bundle artifact skills/subagents.yaml"
fi

API_PORT="$(allocate_free_port)"
API_BASE="http://127.0.0.1:${API_PORT}"
SERVER_LOG="$LOG_DIR/serve-fake.log"

log "start API server for validate/init simulation"
"$ACP_BIN" serve \
  --workspace "$WORKSPACE" \
  --runtime fake \
  --listen "127.0.0.1:${API_PORT}" \
  --run-logs-ttl-hours "$RUN_LOGS_TTL_HOURS" \
  --run-logs-max-runs "$RUN_LOGS_MAX_RUNS" >"$SERVER_LOG" 2>&1 &
SERVER_PID="$!"

if ! wait_for_health "$API_BASE"; then
  die "ACP API did not become healthy in ${READY_TIMEOUT_SEC}s (see $SERVER_LOG)"
fi

log "POST /api/workspace/validate"
curl -fsS -X POST "$API_BASE/api/workspace/validate" > "$VALIDATE_JSON"
python3 - "$VALIDATE_JSON" "$TARGET_REPOS_META_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding='utf-8'))
if not payload.get('ok'):
    raise SystemExit('workspace validate returned ok=false')
resolved = payload.get('resolved_repos') or []
if not resolved:
    raise SystemExit('workspace validate returned empty resolved_repos')
meta = json.load(open(sys.argv[2], encoding='utf-8'))
expected_count = int(meta.get('expected_repo_count', len(resolved)))
if len(resolved) != expected_count:
    raise SystemExit(f"workspace validate resolved_repos count mismatch: expected={expected_count} got={len(resolved)}")
print(f"resolved_repos={len(resolved)}")
PY

log "POST /api/pipeline/init"
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
for _ in $(seq 1 240); do
  curl -fsS "$API_BASE/api/pipeline/runs/$API_INIT_RUN_ID" > "$API_INIT_STATUS_JSON"
  init_status="$(python3 - "$API_INIT_STATUS_JSON" <<'PY'
import json
import sys
payload = json.load(open(sys.argv[1], encoding='utf-8'))
print(payload.get('status', ''))
PY
)"
  if [[ "$init_status" == "succeeded" ]]; then
    break
  fi
  if [[ "$init_status" == "failed" ]]; then
    API_INIT_FINAL_STATUS="$init_status"
    die "API init run failed (see $API_INIT_STATUS_JSON and $SERVER_LOG)"
  fi
  sleep 0.25
done
if [[ "$init_status" != "succeeded" ]]; then
  API_INIT_FINAL_STATUS="$init_status"
  die "API init run did not finish in time (run_id=$API_INIT_RUN_ID)"
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

log "run runtime cycles: fake + headless(provider=$HEADLESS_PROVIDER)"
prev_fake_init_signal=""
prev_fake_refresh_signal=""
prev_headless_init_signal=""
prev_headless_refresh_signal=""
for iteration in $(seq 1 "$ITERATIONS"); do
  run_cli_pipeline "fake" "$HEADLESS_PROVIDER" "init" "$iteration" "$prev_fake_init_signal"
  prev_fake_init_signal="$LAST_SIGNAL"

  run_cli_pipeline "fake" "$HEADLESS_PROVIDER" "refresh" "$iteration" "$prev_fake_refresh_signal"
  prev_fake_refresh_signal="$LAST_SIGNAL"

  run_cli_pipeline "headless" "$HEADLESS_PROVIDER" "init" "$iteration" "$prev_headless_init_signal"
  prev_headless_init_signal="$LAST_SIGNAL"

  run_cli_pipeline "headless" "$HEADLESS_PROVIDER" "refresh" "$iteration" "$prev_headless_refresh_signal"
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

if [[ "$RUN_QUALITY_GATES" == "1" ]]; then
  log "run quality gates: make contracts test lint build"
  if ! (
    cd "$PROVENARCH_ROOT"
    # Run project gates with neutral runtime env so defaults in tests are stable.
    env -u ACP_RUNTIME_PROVIDER -u ACP_QWEN_CMD -u ACP_CLAUDE_CMD \
      make contracts test lint build >"$QUALITY_LOG" 2>&1
  ); then
    QUALITY_GATES_STATUS="failed"
    die "quality gates failed (see $QUALITY_LOG)"
  fi
  QUALITY_GATES_STATUS="passed"
else
  QUALITY_GATES_STATUS="skipped"
fi

for path in \
  "$WORKSPACE/reports/as-is/overview.md" \
  "$WORKSPACE/reports/findings/findings.md" \
  "$WORKSPACE/reports/coverage/open-questions.md"; do
  if [[ ! -f "$path" ]]; then
    die "missing expected artifact after run cycle: $path"
  fi
done

log "full run completed"
