#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
# shellcheck source=scripts/legacy-env-guard.sh
source "$PROVENARCH_ROOT/scripts/legacy-env-guard.sh"
# shellcheck source=scripts/frontend-status-reasons.sh
source "$PROVENARCH_ROOT/scripts/frontend-status-reasons.sh"
ACP_BIN="${ACP_BIN:-$PROVENARCH_ROOT/bin/acp}"
WORKSPACE="${WORKSPACE:-}"
RUNTIME_PROVIDER="${RUNTIME_PROVIDER:-}"
API_READY_TIMEOUT_SEC="${ACP_API_READY_TIMEOUT_SEC:-120}"
RUN_LOGS_TTL_HOURS="${RUN_LOGS_TTL_HOURS:-168}"
RUN_LOGS_MAX_RUNS="${RUN_LOGS_MAX_RUNS:-200}"
OUTPUT_DIR="${OUTPUT_DIR:-}"
LISTEN="${LISTEN:-}"
UI_E2E_EXPECTED_REPO_COUNT="${UI_E2E_EXPECTED_REPO_COUNT:-1}"
UI_E2E_SCENARIO="${UI_E2E_SCENARIO:-init-inspect}"
UI_INIT_POLL_TIMEOUT_SEC="${ACP_UI_INIT_POLL_TIMEOUT_SEC:-}"
UI_E2E_INIT_TIMEOUT_CAP_SEC="${UI_E2E_INIT_TIMEOUT_CAP_SEC:-0}"
UI_E2E_HEADED="${UI_E2E_HEADED:-0}"
UI_E2E_QA_SMOKE="${UI_E2E_QA_SMOKE:-0}"
FRONTEND_RESULT_FILENAME="${FRONTEND_RESULT_FILENAME:-frontend-e2e-result.json}"

DEFAULT_UI_INIT_POLL_TIMEOUT_SEC=900

SERVER_PID=""
BASE_URL=""

log() {
  printf '[frontend-e2e] %s\n' "$*" >&2
}

die() {
  echo "[frontend-e2e][error] $*" >&2
  exit 1
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    die "required command is unavailable: $cmd"
  fi
}

resolve_npm_bin() {
  if [[ "$(type -t npm || true)" == "function" ]]; then
    printf '%s\n' "npm"
    return 0
  fi
  printf '%s\n' "$PROVENARCH_ROOT/scripts/run-npm.sh"
}

current_npm_bin() {
  if [[ -n "${ACP_NPM_BIN:-}" ]]; then
    printf '%s\n' "$ACP_NPM_BIN"
    return 0
  fi
  if [[ -n "${NPM_BIN:-}" ]]; then
    printf '%s\n' "$NPM_BIN"
    return 0
  fi
  resolve_npm_bin
}

parse_positive_int_or_die() {
  local raw="$1"
  local name="$2"
  local numeric
  if [[ ! "$raw" =~ ^[0-9]+$ ]]; then
    die "$name must be a positive integer, got '$raw'"
  fi
  numeric=$((10#$raw))
  if (( numeric <= 0 )); then
    die "$name must be > 0, got '$raw'"
  fi
  printf '%s' "$numeric"
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
  local deadline=$((SECONDS + API_READY_TIMEOUT_SEC))
  while (( SECONDS < deadline )); do
    if curl -fsS "$api_base/api/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

server_is_running() {
  [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" >/dev/null 2>&1
}

capture_server_exit_code_if_stopped() {
  if [[ -z "$SERVER_PID" ]]; then
    printf ''
    return 0
  fi
  if server_is_running; then
    printf ''
    return 0
  fi
  local exit_code
  set +e
  wait "$SERVER_PID" >/dev/null 2>&1
  exit_code="$?"
  set -e
  SERVER_PID=""
  printf '%s' "$exit_code"
}

check_health_once() {
  if curl -fsS "$BASE_URL/api/health" >/dev/null 2>&1; then
    printf 'ok'
  else
    printf 'failed'
  fi
}

extract_frontend_run_id() {
  if [[ ! -f "$PLAYWRIGHT_LOG" ]]; then
    printf ''
    return 0
  fi
  grep -Eo 'ACP_UI_E2E_RUN_ID=[A-Za-z0-9_.:-]+' "$PLAYWRIGHT_LOG" 2>/dev/null \
    | tail -n 1 \
    | cut -d= -f2- \
    || true
}

resolve_last_run_snapshot() {
  local run_id="$1"
  local health_status="$2"
  local api_payload=""
  if [[ -n "$run_id" && "$health_status" == "ok" ]]; then
    api_payload="$(curl -fsS "$BASE_URL/api/pipeline/runs/$run_id" 2>/dev/null || true)"
  fi
  python3 - "$WORKSPACE" "$run_id" "$api_payload" <<'PY'
import json
import sys
from pathlib import Path

workspace = Path(sys.argv[1])
run_id = sys.argv[2].strip()
api_payload = sys.argv[3]

candidate = None
if api_payload:
    try:
        payload = json.loads(api_payload)
        if isinstance(payload, dict):
            candidate = payload
    except Exception:
        candidate = None

if candidate is None:
    history_path = workspace / "reports" / "taskruns" / "run-history.json"
    try:
        history = json.loads(history_path.read_text(encoding="utf-8"))
    except Exception:
        history = {}
    items = history.get("items") if isinstance(history, dict) else []
    if isinstance(items, list):
        if run_id:
            candidate = next((item for item in items if isinstance(item, dict) and item.get("run_id") == run_id), None)
        if candidate is None and items:
            dict_items = [item for item in items if isinstance(item, dict)]
            if dict_items:
                candidate = dict_items[0]

if not isinstance(candidate, dict):
    candidate = {}
print(f"last_run_status={str(candidate.get('status') or '').strip()}")
print(f"last_run_error_code={str(candidate.get('error_code') or '').strip()}")
print(f"last_run_current_step={str(candidate.get('current_step') or '').strip()}")
PY
}

resolve_ui_poll_timeouts() {
  local payload
  payload="$(curl -fsS "$BASE_URL/api/runtime/timeouts" 2>/dev/null || true)"
  if [[ -n "$payload" ]]; then
    local resolved
    resolved="$(python3 - "$payload" <<'PY'
import json
import sys

raw = sys.argv[1]
try:
    payload = json.loads(raw)
except Exception:
    payload = {}
effective = payload.get("effective") if isinstance(payload, dict) else {}
if not isinstance(effective, dict):
    effective = {}
init_value = effective.get("ui_init_poll_timeout_sec")
if isinstance(init_value, int) and init_value > 0:
    print(f"init={init_value}")
PY
)"
    while IFS='=' read -r key value; do
      [[ -z "$key" ]] && continue
      case "$key" in
        init)
          if [[ -z "$UI_INIT_POLL_TIMEOUT_SEC" ]]; then
            UI_INIT_POLL_TIMEOUT_SEC="$value"
          fi
          ;;
      esac
    done <<<"$resolved"
  fi
  if [[ -z "$UI_INIT_POLL_TIMEOUT_SEC" ]]; then
    UI_INIT_POLL_TIMEOUT_SEC="$DEFAULT_UI_INIT_POLL_TIMEOUT_SEC"
  fi
}
# shellcheck disable=SC2329
cleanup() {
  if [[ -n "$SERVER_PID" ]]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" 2>/dev/null || true
    SERVER_PID=""
  fi
}
trap cleanup EXIT

if [[ -z "$WORKSPACE" ]]; then
  die "WORKSPACE is required"
fi
if [[ -z "$RUNTIME_PROVIDER" ]]; then
  die "RUNTIME_PROVIDER is required (claude-code|qwen-code|codex-code)"
fi
if [[ ! -d "$WORKSPACE" ]]; then
  die "WORKSPACE does not exist: $WORKSPACE"
fi
if [[ ! -x "$ACP_BIN" ]]; then
  die "ACP binary is unavailable: $ACP_BIN (run 'make build')"
fi

if [[ -z "$OUTPUT_DIR" ]]; then
  OUTPUT_DIR="$(mktemp -d -t provenarch-frontend-e2e.XXXXXX)"
else
  mkdir -p "$OUTPUT_DIR"
fi

case "$UI_E2E_SCENARIO" in
  init-inspect)
    ;;
  *)
    die "unsupported UI_E2E_SCENARIO '$UI_E2E_SCENARIO' (allowed: init-inspect)"
    ;;
esac
case "$UI_E2E_HEADED" in
  0|1)
    ;;
  *)
    die "UI_E2E_HEADED must be 0 or 1, got '$UI_E2E_HEADED'"
    ;;
esac
case "$UI_E2E_QA_SMOKE" in
  0|1)
    ;;
  *)
    die "UI_E2E_QA_SMOKE must be 0 or 1, got '$UI_E2E_QA_SMOKE'"
    ;;
esac

require_cmd curl
require_cmd python3
NPM_BIN="$(current_npm_bin)"
require_cmd "$NPM_BIN"
acp_ensure_no_legacy_env_set die

runtime_cmd=""
declare -a server_env
server_env=()
case "$RUNTIME_PROVIDER" in
  claude-code)
    runtime_cmd="${ACP_CLAUDE_CMD:-claude}"
    ;;
  qwen-code)
    runtime_cmd="${ACP_QWEN_CMD:-qwen}"
    ;;
  codex-code)
    runtime_cmd="${ACP_CODEX_CMD:-codex}"
    ;;
  *)
    die "unsupported RUNTIME_PROVIDER '$RUNTIME_PROVIDER' (allowed: claude-code, qwen-code, codex-code)"
    ;;
esac

require_cmd "$runtime_cmd"

if [[ -z "$LISTEN" ]]; then
  port="$(allocate_free_port)"
  LISTEN="127.0.0.1:${port}"
fi
BASE_URL="http://${LISTEN}"
SERVER_LOG="$OUTPUT_DIR/serve-${RUNTIME_PROVIDER}.log"
PLAYWRIGHT_LOG="$OUTPUT_DIR/playwright-${RUNTIME_PROVIDER}.log"
PLAYWRIGHT_RESULTS_DIR="$OUTPUT_DIR/playwright-results"
if [[ -z "$FRONTEND_RESULT_FILENAME" || "$FRONTEND_RESULT_FILENAME" == */* ]]; then
  die "FRONTEND_RESULT_FILENAME must be a non-empty file name without '/'"
fi
RESULT_JSON="$OUTPUT_DIR/$FRONTEND_RESULT_FILENAME"
started_at="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
mkdir -p "$PLAYWRIGHT_RESULTS_DIR"

write_frontend_result_json() {
  export FRONTEND_E2E_STARTED_AT="$started_at"
  export FRONTEND_E2E_FINISHED_AT="$finished_at"
  export FRONTEND_E2E_STATUS="$status"
  export FRONTEND_E2E_PROVIDER="$RUNTIME_PROVIDER"
  export FRONTEND_E2E_BASE_URL="$BASE_URL"
  export FRONTEND_E2E_WORKSPACE="$WORKSPACE"
  export FRONTEND_E2E_RUNTIME_CMD="$runtime_cmd"
  export FRONTEND_E2E_SCENARIO="$UI_E2E_SCENARIO"
  export FRONTEND_E2E_SERVER_LOG="$SERVER_LOG"
  export FRONTEND_E2E_PLAYWRIGHT_LOG="$PLAYWRIGHT_LOG"
  export FRONTEND_E2E_REASON="$reason"
  export FRONTEND_E2E_SERVER_PID="$SERVER_PID_STARTED"
  export FRONTEND_E2E_SERVER_EXIT_CODE="$server_exit_code"
  export FRONTEND_E2E_HEALTH_AFTER_FAILURE="$health_after_failure"
  export FRONTEND_E2E_RUN_ID="$frontend_run_id"
  export FRONTEND_E2E_LAST_RUN_STATUS="$last_run_status"
  export FRONTEND_E2E_LAST_RUN_ERROR_CODE="$last_run_error_code"
  export FRONTEND_E2E_LAST_RUN_CURRENT_STEP="$last_run_current_step"
  export FRONTEND_E2E_PLAYWRIGHT_RESULTS_DIR="$PLAYWRIGHT_RESULTS_DIR"
  acp_frontend_reason_validate "$FRONTEND_E2E_REASON" die
  python3 - "$RESULT_JSON" <<'PY'
import json
import os
import sys

path = sys.argv[1]
payload = {
    "started_at": os.environ.get("FRONTEND_E2E_STARTED_AT"),
    "finished_at": os.environ.get("FRONTEND_E2E_FINISHED_AT"),
    "status": os.environ.get("FRONTEND_E2E_STATUS"),
    "runtime_provider": os.environ.get("FRONTEND_E2E_PROVIDER"),
    "base_url": os.environ.get("FRONTEND_E2E_BASE_URL"),
    "workspace": os.environ.get("FRONTEND_E2E_WORKSPACE"),
    "runtime_command": os.environ.get("FRONTEND_E2E_RUNTIME_CMD"),
    "scenario": os.environ.get("FRONTEND_E2E_SCENARIO"),
    "reason": os.environ.get("FRONTEND_E2E_REASON", "unknown"),
    "server_log": os.environ.get("FRONTEND_E2E_SERVER_LOG"),
    "playwright_log": os.environ.get("FRONTEND_E2E_PLAYWRIGHT_LOG"),
    "server_pid": int(os.environ["FRONTEND_E2E_SERVER_PID"]) if os.environ.get("FRONTEND_E2E_SERVER_PID", "").isdigit() else None,
    "server_exit_code": int(os.environ["FRONTEND_E2E_SERVER_EXIT_CODE"]) if os.environ.get("FRONTEND_E2E_SERVER_EXIT_CODE", "").isdigit() else None,
    "health_after_failure": os.environ.get("FRONTEND_E2E_HEALTH_AFTER_FAILURE"),
    "run_id": os.environ.get("FRONTEND_E2E_RUN_ID") or None,
    "last_run_status": os.environ.get("FRONTEND_E2E_LAST_RUN_STATUS") or None,
    "last_run_error_code": os.environ.get("FRONTEND_E2E_LAST_RUN_ERROR_CODE") or None,
    "last_run_current_step": os.environ.get("FRONTEND_E2E_LAST_RUN_CURRENT_STEP") or None,
    "diagnostic_refs": {
        "server_log": os.environ.get("FRONTEND_E2E_SERVER_LOG"),
        "playwright_log": os.environ.get("FRONTEND_E2E_PLAYWRIGHT_LOG"),
        "playwright_results": os.environ.get("FRONTEND_E2E_PLAYWRIGHT_RESULTS_DIR"),
        "screenshots": [],
        "run_history": os.path.join(os.environ.get("FRONTEND_E2E_WORKSPACE", ""), "reports", "taskruns", "run-history.json"),
    },
}
screenshots_dir = os.environ.get("FRONTEND_E2E_PLAYWRIGHT_RESULTS_DIR", "")
if screenshots_dir and os.path.isdir(screenshots_dir):
    payload["diagnostic_refs"]["screenshots"] = [
        os.path.join(screenshots_dir, name)
        for name in sorted(os.listdir(screenshots_dir))
        if name.endswith(".png") and name.startswith("frontend-")
    ]
with open(path, "w", encoding="utf-8") as f:
    json.dump(payload, f, ensure_ascii=True, indent=2)
    f.write("\n")
PY
}

log "starting ACP server (provider=$RUNTIME_PROVIDER listen=$LISTEN)"
if [[ "${#server_env[@]}" -gt 0 ]]; then
  env "${server_env[@]}" "$ACP_BIN" serve \
    --workspace "$WORKSPACE" \
    --runtime headless \
    --runtime-provider "$RUNTIME_PROVIDER" \
    --listen "$LISTEN" \
    --run-logs-ttl-hours "$RUN_LOGS_TTL_HOURS" \
    --run-logs-max-runs "$RUN_LOGS_MAX_RUNS" >"$SERVER_LOG" 2>&1 &
else
  "$ACP_BIN" serve \
    --workspace "$WORKSPACE" \
    --runtime headless \
    --runtime-provider "$RUNTIME_PROVIDER" \
    --listen "$LISTEN" \
    --run-logs-ttl-hours "$RUN_LOGS_TTL_HOURS" \
    --run-logs-max-runs "$RUN_LOGS_MAX_RUNS" >"$SERVER_LOG" 2>&1 &
fi
SERVER_PID="$!"
SERVER_PID_STARTED="$SERVER_PID"

if ! wait_for_health "$BASE_URL"; then
  status="failed"
  reason="$ACP_FRONTEND_REASON_API_UNREACHABLE"
  health_after_failure="failed"
  server_exit_code="$(capture_server_exit_code_if_stopped)"
  if [[ -n "$server_exit_code" ]]; then
    reason="$ACP_FRONTEND_REASON_SERVER_EXITED"
    health_after_failure="server_exited"
  fi
  frontend_run_id=""
  last_run_status=""
  last_run_error_code=""
  last_run_current_step=""
  while IFS='=' read -r key value; do
    case "$key" in
      last_run_status)
        last_run_status="$value"
        ;;
      last_run_error_code)
        last_run_error_code="$value"
        ;;
      last_run_current_step)
        last_run_current_step="$value"
        ;;
    esac
  done < <(resolve_last_run_snapshot "" "$health_after_failure")
  finished_at="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
  write_frontend_result_json
  log "frontend e2e status=$status"
  log "server_log=$SERVER_LOG"
  log "playwright_log=$PLAYWRIGHT_LOG"
  log "result_json=$RESULT_JSON"
  tail -n 80 "$SERVER_LOG" >&2 || true
  exit 1
fi
resolve_ui_poll_timeouts
if [[ "$UI_E2E_SCENARIO" == "init-inspect" ]]; then
  init_timeout_sec="$(parse_positive_int_or_die "$UI_INIT_POLL_TIMEOUT_SEC" "ACP_UI_INIT_POLL_TIMEOUT_SEC")"
  pipeline_timeout_sec=0
  if [[ ! "$UI_E2E_INIT_TIMEOUT_CAP_SEC" =~ ^[0-9]+$ ]]; then
    die "UI_E2E_INIT_TIMEOUT_CAP_SEC must be a non-negative integer, got '$UI_E2E_INIT_TIMEOUT_CAP_SEC'"
  fi
  init_timeout_cap_sec=$((10#$UI_E2E_INIT_TIMEOUT_CAP_SEC))
  if [[ -n "${ACP_PIPELINE_TIMEOUT_SEC:-}" ]]; then
    pipeline_timeout_sec="$(parse_positive_int_or_die "$ACP_PIPELINE_TIMEOUT_SEC" "ACP_PIPELINE_TIMEOUT_SEC")"
  fi
  if (( pipeline_timeout_sec > 0 )); then
    min_init_timeout_sec=$((pipeline_timeout_sec + 30))
    if (( init_timeout_cap_sec > 0 && min_init_timeout_sec > init_timeout_cap_sec )); then
      log "init-inspect timeout guard: suggested=${min_init_timeout_sec}s exceeds cap=${init_timeout_cap_sec}s; applying bounded cap"
      min_init_timeout_sec="$init_timeout_cap_sec"
    fi
    if (( init_timeout_sec < min_init_timeout_sec )); then
      log "init-inspect timeout guard: bump ACP_UI_INIT_POLL_TIMEOUT_SEC ${init_timeout_sec}s -> ${min_init_timeout_sec}s (pipeline_timeout=${pipeline_timeout_sec}s, cap=${init_timeout_cap_sec}s)"
      UI_INIT_POLL_TIMEOUT_SEC="$min_init_timeout_sec"
    fi
  fi
fi
log "effective UI polling timeouts: init=${UI_INIT_POLL_TIMEOUT_SEC}s"

status="passed"
reason="$ACP_FRONTEND_REASON_OK"
server_exit_code=""
health_after_failure="not_checked"
frontend_run_id=""
last_run_status=""
last_run_error_code=""
last_run_current_step=""
playwright_cmd=("$NPM_BIN" run --prefix ui e2e:live)
if [[ "$UI_E2E_HEADED" == "1" ]]; then
  playwright_cmd+=(-- --headed)
fi
log "playwright headed mode: $UI_E2E_HEADED"
if ! (
  cd "$PROVENARCH_ROOT"
  UI_E2E_BASE_URL="$BASE_URL" \
  UI_E2E_RUNTIME_PROVIDER="$RUNTIME_PROVIDER" \
  UI_E2E_SCENARIO="$UI_E2E_SCENARIO" \
  UI_E2E_QA_SMOKE="$UI_E2E_QA_SMOKE" \
  UI_E2E_EXPECTED_REPO_COUNT="$UI_E2E_EXPECTED_REPO_COUNT" \
  ACP_UI_INIT_POLL_TIMEOUT_SEC="$UI_INIT_POLL_TIMEOUT_SEC" \
  UI_E2E_OUTPUT_DIR="$PLAYWRIGHT_RESULTS_DIR" \
  "${playwright_cmd[@]}"
) >"$PLAYWRIGHT_LOG" 2>&1; then
  status="failed"
  reason="$ACP_FRONTEND_REASON_PLAYWRIGHT_FAILED"
  frontend_run_id="$(extract_frontend_run_id)"
  server_exit_code="$(capture_server_exit_code_if_stopped)"
  if [[ -n "$server_exit_code" ]]; then
    reason="$ACP_FRONTEND_REASON_SERVER_EXITED"
    health_after_failure="server_exited"
  else
    health_after_failure="$(check_health_once)"
    if grep -q "ACTIVE_RUN_TIMEOUT:" "$PLAYWRIGHT_LOG"; then
      reason="$ACP_FRONTEND_REASON_ACTIVE_RUN_TIMEOUT"
    elif [[ "$health_after_failure" != "ok" ]]; then
      reason="$ACP_FRONTEND_REASON_API_UNREACHABLE"
    elif grep -E -q "Target (page, context or browser|page|context|browser) has been closed" "$PLAYWRIGHT_LOG"; then
      reason="$ACP_FRONTEND_REASON_BROWSER_CLOSED"
    fi
  fi
  while IFS='=' read -r key value; do
    case "$key" in
      last_run_status)
        last_run_status="$value"
        ;;
      last_run_error_code)
        last_run_error_code="$value"
        ;;
      last_run_current_step)
        last_run_current_step="$value"
        ;;
    esac
  done < <(resolve_last_run_snapshot "$frontend_run_id" "$health_after_failure")
  if [[ "$reason" == "$ACP_FRONTEND_REASON_PLAYWRIGHT_FAILED" && "$last_run_status" == "failed" ]]; then
    reason="$ACP_FRONTEND_REASON_RUNTIME_RUN_FAILED"
  elif [[ "$reason" == "$ACP_FRONTEND_REASON_PLAYWRIGHT_FAILED" ]] && grep -q "terminated before inspect stage: status=failed" "$PLAYWRIGHT_LOG"; then
    reason="$ACP_FRONTEND_REASON_RUNTIME_RUN_FAILED"
  fi
else
  health_after_failure="not_applicable"
  frontend_run_id="$(extract_frontend_run_id)"
fi

finished_at="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
write_frontend_result_json

log "frontend e2e status=$status"
log "server_log=$SERVER_LOG"
log "playwright_log=$PLAYWRIGHT_LOG"
log "result_json=$RESULT_JSON"

if [[ "$status" != "passed" ]]; then
  if [[ "$reason" == "$ACP_FRONTEND_REASON_SERVER_EXITED" ]]; then
    tail -n 80 "$SERVER_LOG" >&2 || true
  fi
  tail -n 80 "$PLAYWRIGHT_LOG" >&2 || true
  exit 1
fi

exit 0
