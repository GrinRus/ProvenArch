#!/usr/bin/env bash
set -euo pipefail

SMOKE_API_TMPDIR=""
SMOKE_API_SERVER_PID=""

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

assert_error_response() {
  local payload="$1"
  local status="$2"
  local expected_status="$3"
  local expected_code="$4"
  local label="$5"
  local code

  if [[ "$status" != "$expected_status" ]]; then
    echo "expected $label status $expected_status, got $status" >&2
    echo "payload: $payload" >&2
    return 1
  fi
  code="$(extract_error_code "$payload")"
  if [[ "$code" != "$expected_code" ]]; then
    echo "expected $label error code $expected_code, got '$code'" >&2
    echo "payload: $payload" >&2
    return 1
  fi
}

assert_status_ok() {
  local status="$1"
  local payload="$2"
  local label="$3"

  if [[ "$status" != "200" ]]; then
    echo "expected $label status 200, got $status" >&2
    echo "payload: $payload" >&2
    return 1
  fi
}

validate_logs_page() {
  local payload="$1"
  local expected_run_id="$2"
  local min_cursor="$3"
  local limit="$4"
  local require_non_empty="$5"

  PAYLOAD="$payload" python3 - "$expected_run_id" "$min_cursor" "$limit" "$require_non_empty" <<'PY'
import json
import os
import sys

expected_run_id = sys.argv[1]
min_cursor = int(sys.argv[2])
limit = int(sys.argv[3])
require_non_empty = sys.argv[4] == "1"
raw = os.environ.get("PAYLOAD", "")

def fail(message: str) -> None:
    print(f"logs payload invalid: {message}", file=sys.stderr)
    raise SystemExit(1)

def is_int(value: object) -> bool:
    return isinstance(value, int) and not isinstance(value, bool)

try:
    parsed = json.loads(raw)
except Exception as exc:
    fail(f"malformed json: {exc}")

if not isinstance(parsed, dict):
    fail("root must be an object")
if parsed.get("run_id") != expected_run_id:
    fail(f"run_id mismatch: expected {expected_run_id!r}, got {parsed.get('run_id')!r}")

items = parsed.get("items")
if not isinstance(items, list):
    fail("items must be an array")
if require_non_empty and not items:
    fail("items must be non-empty")
if len(items) > limit:
    fail(f"items length {len(items)} exceeds requested limit {limit}")

next_cursor = parsed.get("next_cursor")
if not is_int(next_cursor) or next_cursor < 0:
    fail("next_cursor must be a non-negative integer")
eof = parsed.get("eof")
if not isinstance(eof, bool):
    fail("eof must be a boolean")

previous_cursor = min_cursor - 1
for index, item in enumerate(items):
    if not isinstance(item, dict):
        fail(f"items[{index}] must be an object")
    cursor = item.get("cursor")
    if not is_int(cursor):
        fail(f"items[{index}].cursor must be an integer")
    if cursor < min_cursor:
        fail(f"items[{index}].cursor {cursor} is before requested cursor {min_cursor}")
    if cursor <= previous_cursor:
        fail(f"items[{index}].cursor {cursor} does not advance after {previous_cursor}")
    previous_cursor = cursor

    message = item.get("message")
    if not isinstance(message, str) or not message.strip():
        fail(f"items[{index}].message must be a non-empty string")

    kind = item.get("kind")
    if kind not in ("event", "runtime_output"):
        fail(f"items[{index}].kind must be event or runtime_output")
    stream = item.get("stream", "")
    if kind == "runtime_output":
        if stream not in ("stdout", "stderr"):
            fail(f"items[{index}].stream must be stdout or stderr for runtime_output")
    elif stream not in ("", None):
        fail(f"items[{index}].stream must be empty for event entries")

    fields = item.get("fields", {})
    if fields is not None and not isinstance(fields, dict):
        fail(f"items[{index}].fields must be an object when present")

if items:
    last_cursor = items[-1]["cursor"]
    if next_cursor <= last_cursor:
        fail(f"next_cursor {next_cursor} must advance after last item cursor {last_cursor}")
elif next_cursor != min_cursor:
    fail(f"empty page next_cursor {next_cursor} must equal requested cursor {min_cursor}")

if not eof and next_cursor <= min_cursor:
    fail("non-eof page must advance next_cursor")

print(next_cursor)
PY
}

fetch_and_validate_logs_page() {
  local api_base="$1"
  local run_id="$2"
  local cursor="$3"
  local limit="$4"
  local require_non_empty="$5"
  local body_file="$6"
  local label="$7"
  local status
  local payload

  status="$(curl -sS -o "$body_file" -w "%{http_code}" "$api_base/api/pipeline/runs/$run_id/logs?cursor=$cursor&limit=$limit")"
  payload="$(cat "$body_file")"
  assert_status_ok "$status" "$payload" "$label"
  validate_logs_page "$payload" "$run_id" "$cursor" "$limit" "$require_non_empty"
}

cleanup_smoke_api() {
  if [[ -n "${SMOKE_API_SERVER_PID:-}" ]]; then
    kill "$SMOKE_API_SERVER_PID" >/dev/null 2>&1 || true
    wait "$SMOKE_API_SERVER_PID" 2>/dev/null || true
  fi
  if [[ -n "${SMOKE_API_TMPDIR:-}" ]]; then
    rm -rf "$SMOKE_API_TMPDIR"
  fi
}

main() {
  SMOKE_API_TMPDIR="$(mktemp -d)"
  local tmpdir="$SMOKE_API_TMPDIR"
  local server_log="$tmpdir/server.log"
  local go_bin="${GO:-./scripts/run-go.sh}"
  local workspace="$tmpdir/workspace"
  local repo="$tmpdir/repos/payments-service"
  local api_port="${ACP_SMOKE_API_PORT:-}"
  local ready_timeout_sec="${ACP_SMOKE_API_READY_TIMEOUT_SEC:-60}"
  local ready_interval_sec="${ACP_SMOKE_API_READY_INTERVAL_SEC:-0.25}"
  local api_base
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

  "$go_bin" run ./cmd/acp serve \
    --workspace "$workspace" \
    --auto-init \
    --repo-name "payments-service" \
    --repo-path "$repo" \
    --listen "127.0.0.1:$api_port" >"$server_log" 2>&1 &
  SMOKE_API_SERVER_PID=$!
  trap cleanup_smoke_api EXIT

  local ready=0
  local attempts
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
  local run_id
  run_id="$(curl -sSf -X POST -H 'Content-Type: application/json' -d '{"trigger":"manual"}' "$api_base/api/pipeline/refresh" | sed -n 's/.*"run_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"

  if [[ -z "$run_id" ]]; then
    echo "failed to parse run_id" >&2
    cat "$server_log" >&2
    exit 1
  fi

  local run_done=0
  local status_payload=""
  local status=""
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

  local first_logs_body_file="$tmpdir/logs-page-1.json"
  local second_logs_body_file="$tmpdir/logs-page-2.json"
  local next_cursor
  next_cursor="$(fetch_and_validate_logs_page "$api_base" "$run_id" "0" "2" "1" "$first_logs_body_file" "run logs first page")"
  fetch_and_validate_logs_page "$api_base" "$run_id" "$next_cursor" "2" "0" "$second_logs_body_file" "run logs second page" >/dev/null

  local invalid_logs_body_file="$tmpdir/logs-invalid-cursor.json"
  local invalid_logs_status
  local invalid_logs_payload
  invalid_logs_status="$(curl -sS -o "$invalid_logs_body_file" -w "%{http_code}" "$api_base/api/pipeline/runs/$run_id/logs?cursor=-1&limit=2")"
  invalid_logs_payload="$(cat "$invalid_logs_body_file")"
  assert_error_response "$invalid_logs_payload" "$invalid_logs_status" "400" "invalid_cursor" "logs invalid cursor"

  local list_payload
  list_payload="$(curl -sSf "$api_base/api/pipeline/runs?limit=5")"
  if ! echo "$list_payload" | grep -q "$run_id"; then
    echo "run list does not include started run id: $run_id" >&2
    echo "list payload: $list_payload" >&2
    exit 1
  fi

  # cancel endpoint: unknown run -> 404 run_not_found
  local cancel_missing_body_file="$tmpdir/cancel-missing.json"
  local cancel_missing_status
  local cancel_missing_payload
  cancel_missing_status="$(curl -sS -o "$cancel_missing_body_file" -w "%{http_code}" -X POST "$api_base/api/pipeline/runs/run-missing/cancel")"
  cancel_missing_payload="$(cat "$cancel_missing_body_file")"
  assert_error_response "$cancel_missing_payload" "$cancel_missing_status" "404" "run_not_found" "cancel missing"

  # cancel endpoint: terminal run -> 409 run_not_cancelable
  local cancel_terminal_body_file="$tmpdir/cancel-terminal.json"
  local cancel_terminal_status
  local cancel_terminal_payload
  cancel_terminal_status="$(curl -sS -o "$cancel_terminal_body_file" -w "%{http_code}" -X POST "$api_base/api/pipeline/runs/$run_id/cancel")"
  cancel_terminal_payload="$(cat "$cancel_terminal_body_file")"
  assert_error_response "$cancel_terminal_payload" "$cancel_terminal_status" "409" "run_not_cancelable" "cancel terminal"

  echo "smoke-api passed"
}

if [[ "${ACP_SMOKE_API_LIB_ONLY:-0}" != "1" ]]; then
  main "$@"
fi
