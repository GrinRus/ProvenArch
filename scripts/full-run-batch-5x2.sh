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
BATCH_ID="${BATCH_ID:-batch-$(date -u +'%Y%m%dT%H%M%SZ')}"
RUN_COUNT="${RUN_COUNT:-5}"
ACP_CLAUDE_CMD_BIN="${ACP_CLAUDE_CMD_BIN:-claude}"
ACP_QWEN_CMD_BIN="${ACP_QWEN_CMD_BIN:-qwen}"
E2E_TMP_ROOT="${E2E_TMP_ROOT:-/tmp/provenarch-test_arch_project}"
BATCH_ROOT="${BATCH_ROOT:-$E2E_TMP_ROOT/runs/$BATCH_ID}"
REPORTS_ROOT="${REPORTS_ROOT:-$E2E_TMP_ROOT/reports}"
RESOLVED_TARGET_REPOS_FILE=""
DECLARED_REPOS_JSON=""

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

prepare_target_repos_file() {
  local generated_dir="$BATCH_ROOT/generated-inputs"
  mkdir -p "$generated_dir"

  if [[ -n "$TARGET_REPOS_FILE" ]]; then
    if [[ ! -f "$TARGET_REPOS_FILE" ]]; then
      die "TARGET_REPOS_FILE does not exist: $TARGET_REPOS_FILE"
    fi
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

require_cmd git
require_cmd go
require_cmd npm
require_cmd make
require_cmd python3
require_cmd curl
require_cmd "$ACP_CLAUDE_CMD_BIN"
require_cmd "$ACP_QWEN_CMD_BIN"

mkdir -p "$BATCH_ROOT" "$REPORTS_ROOT"
prepare_target_repos_file
collect_declared_repos

PROVENARCH_SHA="$(git -C "$PROVENARCH_ROOT" rev-parse HEAD)"
PROVENARCH_BRANCH="$(git -C "$PROVENARCH_ROOT" rev-parse --abbrev-ref HEAD)"
CLAUDE_PATH="$(command -v "$ACP_CLAUDE_CMD_BIN")"
QWEN_PATH="$(command -v "$ACP_QWEN_CMD_BIN")"
CLAUDE_VERSION="$("$ACP_CLAUDE_CMD_BIN" --version | head -n1 | tr -d '\r')"
QWEN_VERSION="$("$ACP_QWEN_CMD_BIN" --version | head -n1 | tr -d '\r')"
GENERATED_AT_UTC="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
DECLARED_REPOS_JSON_ESCAPED="$(python3 - "$DECLARED_REPOS_JSON" <<'PY'
import json
import sys
print(json.dumps(json.load(open(sys.argv[1], encoding="utf-8"))))
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
log "running DoD precheck: make contracts test lint build"
(
  cd "$PROVENARCH_ROOT"
  make contracts test lint build >"$BATCH_ROOT/precheck-make.log" 2>&1
)

log "installing UI dependencies and Playwright browser"
(
  cd "$PROVENARCH_ROOT"
  npm ci --prefix ui >"$BATCH_ROOT/precheck-ui-npm.log" 2>&1
  npm exec --prefix ui playwright install chromium >"$BATCH_ROOT/precheck-playwright.log" 2>&1
)

failed_runs=0
for provider in qwen-code claude-code; do
  for i in $(seq 1 "$RUN_COUNT"); do
    run_dir="$BATCH_ROOT/$provider/run${i}"
    mkdir -p "$run_dir"
    log "full-run provider=$provider run=$i tmp_root=$run_dir"
    if ! (
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
      ./scripts/full-run-ai-advent.sh
    ) >"$run_dir/batch-driver.log" 2>&1; then
      failed_runs=$((failed_runs + 1))
      log "run failed provider=$provider run=$i (see $run_dir/batch-driver.log)"
    fi
  done
done

frontend_failures=0
for provider in qwen-code claude-code; do
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
    ./scripts/frontend-live-e2e.sh
  ) >"$output_dir/driver.log" 2>&1; then
    frontend_failures=$((frontend_failures + 1))
    log "frontend e2e failed provider=$provider (see $output_dir/driver.log)"
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

if [[ "$failed_runs" -ne 0 || "$frontend_failures" -ne 0 ]]; then
  die "batch completed with failures: full_run_failed=$failed_runs frontend_failed=$frontend_failures"
fi

log "batch completed successfully"
exit 0
