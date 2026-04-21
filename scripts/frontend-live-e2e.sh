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
UI_E2E_CANCEL_STUB_SLEEP_SEC="${UI_E2E_CANCEL_STUB_SLEEP_SEC:-90}"
UI_INIT_POLL_TIMEOUT_SEC="${ACP_UI_INIT_POLL_TIMEOUT_SEC:-}"
UI_CANCEL_POLL_TIMEOUT_SEC="${ACP_UI_CANCEL_POLL_TIMEOUT_SEC:-}"
UI_E2E_CANCEL_TIMEOUT_MARGIN_SEC="${UI_E2E_CANCEL_TIMEOUT_MARGIN_SEC:-30}"
UI_E2E_INIT_TIMEOUT_CAP_SEC="${UI_E2E_INIT_TIMEOUT_CAP_SEC:-1800}"
UI_E2E_HEADED="${UI_E2E_HEADED:-0}"
FRONTEND_RESULT_FILENAME="${FRONTEND_RESULT_FILENAME:-frontend-e2e-result.json}"

DEFAULT_UI_INIT_POLL_TIMEOUT_SEC=900
DEFAULT_UI_CANCEL_POLL_TIMEOUT_SEC=420

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
cancel_value = effective.get("ui_cancel_poll_timeout_sec")
if isinstance(init_value, int) and init_value > 0:
    print(f"init={init_value}")
if isinstance(cancel_value, int) and cancel_value > 0:
    print(f"cancel={cancel_value}")
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
        cancel)
          if [[ -z "$UI_CANCEL_POLL_TIMEOUT_SEC" ]]; then
            UI_CANCEL_POLL_TIMEOUT_SEC="$value"
          fi
          ;;
      esac
    done <<<"$resolved"
  fi
  if [[ -z "$UI_INIT_POLL_TIMEOUT_SEC" ]]; then
    UI_INIT_POLL_TIMEOUT_SEC="$DEFAULT_UI_INIT_POLL_TIMEOUT_SEC"
  fi
  if [[ -z "$UI_CANCEL_POLL_TIMEOUT_SEC" ]]; then
    UI_CANCEL_POLL_TIMEOUT_SEC="$DEFAULT_UI_CANCEL_POLL_TIMEOUT_SEC"
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
  die "RUNTIME_PROVIDER is required (claude-code|qwen-code)"
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
  init-inspect|cancel-refresh)
    ;;
  *)
    die "unsupported UI_E2E_SCENARIO '$UI_E2E_SCENARIO' (allowed: init-inspect, cancel-refresh)"
    ;;
esac
case "$UI_E2E_HEADED" in
  0|1)
    ;;
  *)
    die "UI_E2E_HEADED must be 0 or 1, got '$UI_E2E_HEADED'"
    ;;
esac

require_cmd curl
require_cmd python3
require_cmd npm
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
  *)
    die "unsupported RUNTIME_PROVIDER '$RUNTIME_PROVIDER' (allowed: claude-code, qwen-code)"
    ;;
esac

if [[ "$UI_E2E_SCENARIO" == "cancel-refresh" ]]; then
  stub_runner="$OUTPUT_DIR/runtime-cancel-stub.sh"
  cat >"$stub_runner" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
trap 'exit 130' TERM INT HUP PIPE
sleep ${UI_E2E_CANCEL_STUB_SLEEP_SEC}
cat <<'JSON'
{"meta":{"task_id":"task-stub","step_id":"refresh.step1.collect","runtime":{"name":"${RUNTIME_PROVIDER}","version":"cancel-stub"},"started_at":"2026-04-12T00:00:00Z"},"summary":"stub completion (unexpected in cancel scenario)","changeset":[]}
JSON
EOF
  chmod +x "$stub_runner"
  runtime_cmd="$stub_runner"
  case "$RUNTIME_PROVIDER" in
    claude-code)
      server_env+=("ACP_CLAUDE_CMD=$runtime_cmd")
      ;;
    qwen-code)
      server_env+=("ACP_QWEN_CMD=$runtime_cmd")
      ;;
  esac
fi

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

if ! wait_for_health "$BASE_URL"; then
  die "ACP server did not become healthy in ${API_READY_TIMEOUT_SEC}s (see $SERVER_LOG)"
fi
resolve_ui_poll_timeouts
if [[ "$UI_E2E_SCENARIO" == "init-inspect" ]]; then
  init_timeout_sec="$(parse_positive_int_or_die "$UI_INIT_POLL_TIMEOUT_SEC" "ACP_UI_INIT_POLL_TIMEOUT_SEC")"
  pipeline_timeout_sec=0
  init_timeout_cap_sec="$(parse_positive_int_or_die "$UI_E2E_INIT_TIMEOUT_CAP_SEC" "UI_E2E_INIT_TIMEOUT_CAP_SEC")"
  if [[ -n "${ACP_PIPELINE_TIMEOUT_SEC:-}" ]]; then
    pipeline_timeout_sec="$(parse_positive_int_or_die "$ACP_PIPELINE_TIMEOUT_SEC" "ACP_PIPELINE_TIMEOUT_SEC")"
  fi
  if (( pipeline_timeout_sec > 0 )); then
    min_init_timeout_sec=$((pipeline_timeout_sec + 30))
    if (( min_init_timeout_sec > init_timeout_cap_sec )); then
      log "init-inspect timeout guard: suggested=${min_init_timeout_sec}s exceeds cap=${init_timeout_cap_sec}s; applying bounded cap"
      min_init_timeout_sec="$init_timeout_cap_sec"
    fi
    if (( init_timeout_sec < min_init_timeout_sec )); then
      log "init-inspect timeout guard: bump ACP_UI_INIT_POLL_TIMEOUT_SEC ${init_timeout_sec}s -> ${min_init_timeout_sec}s (pipeline_timeout=${pipeline_timeout_sec}s, cap=${init_timeout_cap_sec}s)"
      UI_INIT_POLL_TIMEOUT_SEC="$min_init_timeout_sec"
    fi
  fi
fi
log "effective UI polling timeouts: init=${UI_INIT_POLL_TIMEOUT_SEC}s cancel=${UI_CANCEL_POLL_TIMEOUT_SEC}s"
if [[ "$UI_E2E_SCENARIO" == "cancel-refresh" ]]; then
  cancel_timeout_sec="$(parse_positive_int_or_die "$UI_CANCEL_POLL_TIMEOUT_SEC" "ACP_UI_CANCEL_POLL_TIMEOUT_SEC")"
  cancel_stub_sleep_sec="$(parse_positive_int_or_die "$UI_E2E_CANCEL_STUB_SLEEP_SEC" "UI_E2E_CANCEL_STUB_SLEEP_SEC")"
  cancel_margin_sec="$(parse_positive_int_or_die "$UI_E2E_CANCEL_TIMEOUT_MARGIN_SEC" "UI_E2E_CANCEL_TIMEOUT_MARGIN_SEC")"
  min_cancel_timeout_sec=$((cancel_stub_sleep_sec + cancel_margin_sec))
  log "cancel-refresh timeout guard: timeout=${cancel_timeout_sec}s stub_sleep=${cancel_stub_sleep_sec}s margin=${cancel_margin_sec}s min_required=${min_cancel_timeout_sec}s"
  if (( cancel_timeout_sec < min_cancel_timeout_sec )); then
    die "cancel-refresh preflight failed: ACP_UI_CANCEL_POLL_TIMEOUT_SEC=${cancel_timeout_sec}s must be >= ${min_cancel_timeout_sec}s (UI_E2E_CANCEL_STUB_SLEEP_SEC=${cancel_stub_sleep_sec}s + margin=${cancel_margin_sec}s)"
  fi
fi

status="passed"
reason="$ACP_FRONTEND_REASON_OK"
playwright_cmd=(npm run --prefix ui e2e:live)
if [[ "$UI_E2E_HEADED" == "1" ]]; then
  playwright_cmd+=(-- --headed)
fi
log "playwright headed mode: $UI_E2E_HEADED"
if ! (
  cd "$PROVENARCH_ROOT"
  UI_E2E_BASE_URL="$BASE_URL" \
  UI_E2E_RUNTIME_PROVIDER="$RUNTIME_PROVIDER" \
  UI_E2E_SCENARIO="$UI_E2E_SCENARIO" \
  UI_E2E_EXPECTED_REPO_COUNT="$UI_E2E_EXPECTED_REPO_COUNT" \
  ACP_UI_INIT_POLL_TIMEOUT_SEC="$UI_INIT_POLL_TIMEOUT_SEC" \
  ACP_UI_CANCEL_POLL_TIMEOUT_SEC="$UI_CANCEL_POLL_TIMEOUT_SEC" \
  UI_E2E_OUTPUT_DIR="$PLAYWRIGHT_RESULTS_DIR" \
  "${playwright_cmd[@]}"
) >"$PLAYWRIGHT_LOG" 2>&1; then
  status="failed"
  reason="$ACP_FRONTEND_REASON_PLAYWRIGHT_FAILED"
fi

finished_at="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
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
}
with open(path, "w", encoding="utf-8") as f:
    json.dump(payload, f, ensure_ascii=True, indent=2)
    f.write("\n")
PY

log "frontend e2e status=$status"
log "server_log=$SERVER_LOG"
log "playwright_log=$PLAYWRIGHT_LOG"
log "result_json=$RESULT_JSON"

if [[ "$status" != "passed" ]]; then
  tail -n 80 "$PLAYWRIGHT_LOG" >&2 || true
  exit 1
fi

exit 0
