#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ACP_BIN="${ACP_BIN:-$PROVENARCH_ROOT/bin/acp}"
WORKSPACE="${WORKSPACE:-}"
RUNTIME_PROVIDER="${RUNTIME_PROVIDER:-}"
READY_TIMEOUT_SEC="${READY_TIMEOUT_SEC:-120}"
RUN_LOGS_TTL_HOURS="${RUN_LOGS_TTL_HOURS:-168}"
RUN_LOGS_MAX_RUNS="${RUN_LOGS_MAX_RUNS:-200}"
OUTPUT_DIR="${OUTPUT_DIR:-}"
LISTEN="${LISTEN:-}"

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
  local deadline=$((SECONDS + READY_TIMEOUT_SEC))
  while (( SECONDS < deadline )); do
    if curl -fsS "$api_base/api/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

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

require_cmd curl
require_cmd python3
require_cmd npm

runtime_cmd=""
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
require_cmd "$runtime_cmd"

if [[ -z "$LISTEN" ]]; then
  port="$(allocate_free_port)"
  LISTEN="127.0.0.1:${port}"
fi
BASE_URL="http://${LISTEN}"
SERVER_LOG="$OUTPUT_DIR/serve-${RUNTIME_PROVIDER}.log"
PLAYWRIGHT_LOG="$OUTPUT_DIR/playwright-${RUNTIME_PROVIDER}.log"
RESULT_JSON="$OUTPUT_DIR/frontend-e2e-result.json"
started_at="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"

log "starting ACP server (provider=$RUNTIME_PROVIDER listen=$LISTEN)"
"$ACP_BIN" serve \
  --workspace "$WORKSPACE" \
  --runtime headless \
  --runtime-provider "$RUNTIME_PROVIDER" \
  --listen "$LISTEN" \
  --run-logs-ttl-hours "$RUN_LOGS_TTL_HOURS" \
  --run-logs-max-runs "$RUN_LOGS_MAX_RUNS" >"$SERVER_LOG" 2>&1 &
SERVER_PID="$!"

if ! wait_for_health "$BASE_URL"; then
  die "ACP server did not become healthy in ${READY_TIMEOUT_SEC}s (see $SERVER_LOG)"
fi

status="passed"
if ! (
  cd "$PROVENARCH_ROOT"
  UI_E2E_BASE_URL="$BASE_URL" UI_E2E_RUNTIME_PROVIDER="$RUNTIME_PROVIDER" npm run --prefix ui e2e:live
) >"$PLAYWRIGHT_LOG" 2>&1; then
  status="failed"
fi

finished_at="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
export FRONTEND_E2E_STARTED_AT="$started_at"
export FRONTEND_E2E_FINISHED_AT="$finished_at"
export FRONTEND_E2E_STATUS="$status"
export FRONTEND_E2E_PROVIDER="$RUNTIME_PROVIDER"
export FRONTEND_E2E_BASE_URL="$BASE_URL"
export FRONTEND_E2E_WORKSPACE="$WORKSPACE"
export FRONTEND_E2E_RUNTIME_CMD="$runtime_cmd"
export FRONTEND_E2E_SERVER_LOG="$SERVER_LOG"
export FRONTEND_E2E_PLAYWRIGHT_LOG="$PLAYWRIGHT_LOG"
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
