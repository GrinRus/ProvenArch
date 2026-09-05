#!/usr/bin/env bash
set -euo pipefail

SCENARIOS=(
  "analysis-failed-shard-mock"
  "happy-path-mock"
  "onboarding-recovery-mock"
  "permission-recovery-mock"
  "provider-stream-mock"
  "publish-git-recovery-mock"
  "qa-recovery-mock"
  "source-recovery-mock"
)

if [[ "${ACP_UI_MOCK_E2E_LIST:-0}" == "1" ]]; then
  printf '%s\n' "${SCENARIOS[@]}"
  exit 0
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
ui_dir="$repo_root/ui"
playwright_bin="$ui_dir/node_modules/.bin/playwright"
config="${ACP_UI_MOCK_PLAYWRIGHT_CONFIG:-playwright.mock.config.ts}"
if [[ ! -x "$playwright_bin" ]]; then
  echo "ui mock e2e: Playwright is not installed at $playwright_bin; run npm ci --prefix ui first" >&2
  exit 1
fi

results_parent="${ACP_UI_MOCK_E2E_RESULTS_DIR:-${TMPDIR:-/tmp}}"
mkdir -p "$results_parent"
results_root="$(mktemp -d "$results_parent/provenarch-ui-mock-e2e.XXXXXX")"
results_root="$(cd "$results_root" && pwd)"
log_dir="$results_root/logs"
mkdir -p "$log_dir"
echo "ui mock e2e: evidence at $results_root"

node_bin="$("$script_dir/resolve-node-tool.sh" node)"
node_dir="$(cd "$(dirname "$node_bin")" && pwd)"
export PATH="$node_dir:${PATH:-}"
if [[ -z "${UI_E2E_BASE_URL:-}" ]]; then
  port="$("$node_bin" -e '
    const server = require("node:net").createServer();
    server.on("error", error => { console.error(error.message); process.exit(1); });
    server.listen(0, "127.0.0.1", () => {
      console.log(server.address().port);
      server.close();
    });
  ')"
  export UI_E2E_BASE_URL="http://127.0.0.1:$port"
fi

playwright_results_parent="${UI_E2E_PLAYWRIGHT_OUTPUT_DIR:-$results_root/test-results}"
mkdir -p "$playwright_results_parent"
playwright_results_root="$(mktemp -d "$playwright_results_parent/run.XXXXXX")"
playwright_results_root="$(cd "$playwright_results_root" && pwd)"

passed_count=0
skipped_count=0

cd "$ui_dir"
for scenario in "${SCENARIOS[@]}"; do
  spec="e2e/${scenario}.spec.ts"
  log_path="$log_dir/${scenario}.log"

  if [[ ! -f "$spec" ]]; then
    echo "ui mock e2e: missing spec for scenario $scenario at ui/$spec" >&2
    exit 1
  fi

  echo "ui mock e2e: running $scenario"
  if ! UI_E2E_SCENARIO="$scenario" UI_E2E_OUTPUT_DIR="$results_root/screenshots/$scenario" UI_E2E_PLAYWRIGHT_OUTPUT_DIR="$playwright_results_root/$scenario" "$playwright_bin" test -c "$config" "$spec" --reporter=list >"$log_path" 2>&1; then
    cat "$log_path" >&2
    exit 1
  fi
  cat "$log_path"

  if grep -Eq '(^|[^0-9])[1-9][0-9]* skipped\b' "$log_path"; then
    echo "ui mock e2e: scenario $scenario produced skipped tests" >&2
    exit 1
  fi
  if ! grep -Eq '(^|[^0-9])1 passed\b' "$log_path"; then
    echo "ui mock e2e: scenario $scenario did not report exactly one passed test" >&2
    exit 1
  fi

  passed_count=$((passed_count + 1))
done

if [[ "$passed_count" -ne "${#SCENARIOS[@]}" ]]; then
  echo "ui mock e2e: expected ${#SCENARIOS[@]} passed scenarios, got $passed_count" >&2
  exit 1
fi

echo "ui mock e2e passed: ${passed_count} passed / ${skipped_count} skipped"
