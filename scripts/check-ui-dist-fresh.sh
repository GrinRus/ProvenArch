#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/acp-ui-freshness.XXXXXX")"
trap 'rm -rf "$tmp_root"' EXIT
./scripts/run-npm.sh run build --prefix ui -- --outDir "$tmp_root/dist"

if ! diff -qr --exclude=README.md "$tmp_root/dist" internal/api/ui_dist; then
  echo "internal/api/ui_dist is stale; run make build to regenerate the embedded UI bundle." >&2
  exit 1
fi

echo "internal/api/ui_dist matches the working-tree UI source"
