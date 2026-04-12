#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
server_log="$tmpdir/server.log"
workspace="$tmpdir/workspace"
repo="$tmpdir/repos/payments-service"
api_port="${ACP_SMOKE_API_PORT:-}"
ready_timeout_sec="${ACP_SMOKE_API_READY_TIMEOUT_SEC:-60}"
ready_interval_sec="${ACP_SMOKE_API_READY_INTERVAL_SEC:-0.25}"
mkdir -p "$repo"

if [[ -z "$api_port" ]]; then
  api_port="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
fi

api_base="http://127.0.0.1:$api_port"

extract_error_code() {
  local payload="$1"
  PAYLOAD="$payload" python3 - <<'PY'
import json
import os
import sys

raw = os.environ.get("PAYLOAD", "")
try:
    parsed = json.loads(raw)
except Exception:
    print("")
    raise SystemExit(0)
error = parsed.get("error") or {}
code = error.get("code") or ""
print(code)
PY
}

go run ./cmd/acp serve \
  --workspace "$workspace" \
  --auto-init \
  --repo-name "payments-service" \
  --repo-path "$repo" \
  --listen "127.0.0.1:$api_port" >"$server_log" 2>&1 &
server_pid=$!
trap 'kill "$server_pid" >/dev/null 2>&1 || true; wait "$server_pid" 2>/dev/null || true; rm -rf "$tmpdir"' EXIT

ready=0
attempts="$(python3 - <<'PY' "$ready_timeout_sec" "$ready_interval_sec"
import math
import sys
timeout = float(sys.argv[1])
interval = float(sys.argv[2])
print(max(1, math.ceil(timeout / interval)))
PY
)"
for _ in $(seq 1 "$attempts"); do
  if curl -sSf "$api_base/api/health" >/dev/null 2>/dev/null; then
    ready=1
    break
  fi
  sleep "$ready_interval_sec"
done
if [[ "$ready" -ne 1 ]]; then
  echo "api smoke: server did not become healthy in time (${ready_timeout_sec}s timeout)" >&2
  cat "$server_log" >&2
  exit 1
fi

curl -sSf -X POST "$api_base/api/workspace/validate" >/dev/null
run_id="$(curl -sSf -X POST -H 'Content-Type: application/json' -d '{"trigger":"manual"}' "$api_base/api/pipeline/refresh" | sed -n 's/.*"run_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"

if [[ -z "$run_id" ]]; then
  echo "failed to parse run_id" >&2
  cat "$server_log" >&2
  exit 1
fi

run_done=0
status_payload=""
for _ in {1..80}; do
  status_payload="$(curl -sSf "$api_base/api/pipeline/runs/$run_id")"
  status="$(echo "$status_payload" | sed -n 's/.*"status"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
  if [[ "$status" == "succeeded" ]]; then
    run_done=1
    break
  fi
  if [[ "$status" == "failed" ]]; then
    echo "run failed: $status_payload" >&2
    exit 1
  fi
  sleep 0.25
done

if [[ "$run_done" -ne 1 ]]; then
  echo "run did not reach succeeded status in time (run_id=$run_id)" >&2
  if [[ -n "$status_payload" ]]; then
    echo "last status payload: $status_payload" >&2
  fi
  cat "$server_log" >&2
  exit 1
fi

curl -sSf "$api_base/api/pipeline/runs/$run_id/artifacts" >/dev/null
list_payload="$(curl -sSf "$api_base/api/pipeline/runs?limit=5")"
if ! echo "$list_payload" | grep -q "$run_id"; then
  echo "run list does not include started run id: $run_id" >&2
  echo "list payload: $list_payload" >&2
  exit 1
fi

# cancel endpoint: unknown run -> 404 run_not_found
cancel_missing_body_file="$tmpdir/cancel-missing.json"
cancel_missing_status="$(curl -sS -o "$cancel_missing_body_file" -w "%{http_code}" -X POST "$api_base/api/pipeline/runs/run-missing/cancel")"
cancel_missing_payload="$(cat "$cancel_missing_body_file")"
if [[ "$cancel_missing_status" != "404" ]]; then
  echo "expected cancel missing status 404, got $cancel_missing_status" >&2
  echo "payload: $cancel_missing_payload" >&2
  exit 1
fi
cancel_missing_code="$(extract_error_code "$cancel_missing_payload")"
if [[ "$cancel_missing_code" != "run_not_found" ]]; then
  echo "expected cancel missing error code run_not_found, got '$cancel_missing_code'" >&2
  echo "payload: $cancel_missing_payload" >&2
  exit 1
fi

# cancel endpoint: terminal run -> 409 run_not_cancelable
cancel_terminal_body_file="$tmpdir/cancel-terminal.json"
cancel_terminal_status="$(curl -sS -o "$cancel_terminal_body_file" -w "%{http_code}" -X POST "$api_base/api/pipeline/runs/$run_id/cancel")"
cancel_terminal_payload="$(cat "$cancel_terminal_body_file")"
if [[ "$cancel_terminal_status" != "409" ]]; then
  echo "expected cancel terminal status 409, got $cancel_terminal_status" >&2
  echo "payload: $cancel_terminal_payload" >&2
  exit 1
fi
cancel_terminal_code="$(extract_error_code "$cancel_terminal_payload")"
if [[ "$cancel_terminal_code" != "run_not_cancelable" ]]; then
  echo "expected cancel terminal error code run_not_cancelable, got '$cancel_terminal_code'" >&2
  echo "payload: $cancel_terminal_payload" >&2
  exit 1
fi

echo "smoke-api passed"
