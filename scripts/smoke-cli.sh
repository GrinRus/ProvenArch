#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
GO_BIN="${GO:-./scripts/run-go.sh}"

workspace="$tmpdir/workspace"
repo="$tmpdir/repos/payments-service"
mkdir -p "$workspace" "$repo"
printf '# payments-service\n' >"$repo/README.md"

"$GO_BIN" run ./cmd/acp init-workspace \
  --workspace "$workspace" \
  --repo-name "payments-service" \
  --repo-path "$repo" \
  --force >/dev/null

"$GO_BIN" run ./cmd/acp run --workspace "$workspace" --pipeline init --non-interactive >/dev/null
"$GO_BIN" run ./cmd/acp run --workspace "$workspace" --pipeline refresh --non-interactive >/dev/null

test -f "$workspace/reports/as-is/overview.md"
test -f "$workspace/reports/findings/findings.md"

echo "smoke-cli passed"
