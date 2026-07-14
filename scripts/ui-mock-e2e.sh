#!/usr/bin/env bash
set -euo pipefail

SCENARIOS=(
  "analysis-failed-shard-mock"
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
results_root="${ACP_UI_MOCK_E2E_RESULTS_DIR:-/tmp/provenarch-ui-mock-e2e}"
log_dir="$results_root/logs"

if [[ ! -x "$playwright_bin" ]]; then
  echo "ui mock e2e: Playwright is not installed at $playwright_bin; run npm ci --prefix ui first" >&2
  exit 1
fi

rm -rf "$results_root"
mkdir -p "$log_dir"

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
  if ! UI_E2E_SCENARIO="$scenario" UI_E2E_OUTPUT_DIR="$results_root/screenshots/$scenario" "$playwright_bin" test -c "$config" "$spec" --reporter=list >"$log_path" 2>&1; then
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
