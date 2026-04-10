#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
TARGET_REPO="${TARGET_REPO:-$PROVENARCH_ROOT/test_arch_project/repos/ibatulanandjp__ecommerce-microservices}"
BATCH_ID="${BATCH_ID:-batch-$(date -u +'%Y%m%dT%H%M%SZ')}"
RUN_COUNT="${RUN_COUNT:-5}"
ACP_CLAUDE_CMD_BIN="${ACP_CLAUDE_CMD_BIN:-claude}"
ACP_QWEN_CMD_BIN="${ACP_QWEN_CMD_BIN:-qwen}"
BATCH_ROOT="${BATCH_ROOT:-$PROVENARCH_ROOT/test_arch_project/runs/$BATCH_ID}"
REPORTS_ROOT="${REPORTS_ROOT:-$PROVENARCH_ROOT/test_arch_project/reports}"

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

if [[ ! "$RUN_COUNT" =~ ^[1-9][0-9]*$ ]]; then
  die "RUN_COUNT must be a positive integer, got '$RUN_COUNT'"
fi
if [[ "$RUN_COUNT" != "5" ]]; then
  die "RUN_COUNT must be 5 for this batch plan (got '$RUN_COUNT')"
fi
if [[ ! -d "$TARGET_REPO" ]]; then
  die "TARGET_REPO does not exist: $TARGET_REPO"
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

PROVENARCH_SHA="$(git -C "$PROVENARCH_ROOT" rev-parse HEAD)"
PROVENARCH_BRANCH="$(git -C "$PROVENARCH_ROOT" rev-parse --abbrev-ref HEAD)"
TARGET_REPO_SHA="$(git -C "$TARGET_REPO" rev-parse HEAD)"
CLAUDE_PATH="$(command -v "$ACP_CLAUDE_CMD_BIN")"
QWEN_PATH="$(command -v "$ACP_QWEN_CMD_BIN")"
CLAUDE_VERSION="$("$ACP_CLAUDE_CMD_BIN" --version | head -n1 | tr -d '\r')"
QWEN_VERSION="$("$ACP_QWEN_CMD_BIN" --version | head -n1 | tr -d '\r')"
GENERATED_AT_UTC="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"

export BATCH_PRE_GENERATED_AT_UTC="$GENERATED_AT_UTC"
export BATCH_PRE_PROVENARCH_ROOT="$PROVENARCH_ROOT"
export BATCH_PRE_TARGET_REPO="$TARGET_REPO"
export BATCH_PRE_PROVENARCH_SHA="$PROVENARCH_SHA"
export BATCH_PRE_PROVENARCH_BRANCH="$PROVENARCH_BRANCH"
export BATCH_PRE_TARGET_REPO_SHA="$TARGET_REPO_SHA"
export BATCH_PRE_CLAUDE_PATH="$CLAUDE_PATH"
export BATCH_PRE_CLAUDE_VERSION="$CLAUDE_VERSION"
export BATCH_PRE_QWEN_PATH="$QWEN_PATH"
export BATCH_PRE_QWEN_VERSION="$QWEN_VERSION"

python3 - "$BATCH_ROOT/preflight.json" <<'PY'
import json
import os
import sys

payload = {
    "generated_at_utc": os.environ["BATCH_PRE_GENERATED_AT_UTC"],
    "provenarch_root": os.environ["BATCH_PRE_PROVENARCH_ROOT"],
    "provenarch_sha": os.environ["BATCH_PRE_PROVENARCH_SHA"],
    "provenarch_branch": os.environ["BATCH_PRE_PROVENARCH_BRANCH"],
    "target_repo": os.environ["BATCH_PRE_TARGET_REPO"],
    "target_repo_sha": os.environ["BATCH_PRE_TARGET_REPO_SHA"],
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
      TARGET_REPO="$TARGET_REPO" \
      TMP_ROOT="$run_dir" \
      KEEP_TMP=1 \
      ITERATIONS=1 \
      RUN_QUALITY_GATES=1 \
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
  workspace="$BATCH_ROOT/$provider/run1/arch-workspace"
  output_dir="$BATCH_ROOT/frontend/$provider"
  mkdir -p "$output_dir"
  log "frontend live e2e provider=$provider workspace=$workspace"
  if ! (
    cd "$PROVENARCH_ROOT"
    WORKSPACE="$workspace" \
    RUNTIME_PROVIDER="$provider" \
    OUTPUT_DIR="$output_dir" \
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
