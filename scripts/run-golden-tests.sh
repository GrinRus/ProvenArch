#!/usr/bin/env bash
set -Eeuo pipefail

if (( $# < 2 )); then
  echo "usage: $0 <go-package> <test-name> [<test-name> ...]" >&2
  exit 2
fi

package="$1"
shift
expected_tests=("$@")

seen_tests=""
for test_name in "${expected_tests[@]}"; do
  if [[ ! "$test_name" =~ ^Test[A-Za-z0-9_]+$ ]]; then
    echo "invalid golden test name: $test_name" >&2
    exit 2
  fi
  case "|$seen_tests|" in
    *"|$test_name|"*)
    echo "duplicate golden test name: $test_name" >&2
    exit 2
    ;;
  esac
  seen_tests+="|$test_name"
done

repo_root="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
if [[ -n "${ACP_GOLDEN_GO_BIN:-}" ]]; then
  go_cmd=("$ACP_GOLDEN_GO_BIN")
else
  go_cmd=("$repo_root/scripts/run-go.sh")
fi

list_output=""
if ! list_output="$("${go_cmd[@]}" test "$package" -list '^Test' 2>&1)"; then
  echo "failed to list golden tests for $package:" >&2
  printf '%s\n' "$list_output" >&2
  exit 1
fi

for test_name in "${expected_tests[@]}"; do
  if ! grep -Fqx "$test_name" <<< "$list_output"; then
    echo "golden test selection is stale: expected $test_name is not present in $package" >&2
    printf '%s\n' "$list_output" >&2
    exit 1
  fi
done

test_regex='^('
for test_name in "${expected_tests[@]}"; do
  test_regex+="$test_name|"
done
test_regex="${test_regex%|})$"

run_log="$(mktemp "${TMPDIR:-/tmp}/provenarch-golden-tests.XXXXXX")"
cleanup() {
  rm -f "$run_log"
}
trap cleanup EXIT

if ! "${go_cmd[@]}" test "$package" -json -run "$test_regex" -count=1 >"$run_log" 2>&1; then
  cat "$run_log" >&2
  exit 1
fi

has_event() {
  local action="$1"
  local test_name="$2"
  grep -Eq "\"Action\":\"$action\".*\"Test\":\"$test_name\"" "$run_log"
}

for test_name in "${expected_tests[@]}"; do
  if ! has_event pass "$test_name"; then
    echo "golden test did not pass: $test_name" >&2
    cat "$run_log" >&2
    exit 1
  fi
  if has_event skip "$test_name"; then
    echo "golden test was skipped: $test_name" >&2
    cat "$run_log" >&2
    exit 1
  fi
  if has_event fail "$test_name"; then
    echo "golden test failed: $test_name" >&2
    cat "$run_log" >&2
    exit 1
  fi
done

printf 'golden tests passed (%d):' "${#expected_tests[@]}"
printf ' %s' "${expected_tests[@]}"
printf '\n'
