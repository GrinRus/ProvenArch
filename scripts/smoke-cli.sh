#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

workspace="$tmpdir/workspace"
repo="$tmpdir/repos/payments-service"
mkdir -p "$workspace" "$repo"
cat > "$workspace/workspace.yaml" <<MANIFEST
version: 1
repos:
  - name: payments-service
    path: $repo
MANIFEST

go run ./cmd/acp run --workspace "$workspace" --pipeline init --non-interactive >/dev/null
go run ./cmd/acp run --workspace "$workspace" --pipeline refresh --non-interactive >/dev/null

test -f "$workspace/reports/as-is/overview.md"
test -f "$workspace/reports/findings/findings.md"

echo "smoke-cli passed"
