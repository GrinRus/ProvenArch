#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

rm -rf ui/dist ui/node_modules/.vite
./scripts/run-npm.sh run build --prefix ui

rm -rf internal/api/ui_dist/assets internal/api/ui_dist/index.html
mkdir -p internal/api/ui_dist/assets
cp -R ui/dist/assets/* internal/api/ui_dist/assets/
cp ui/dist/index.html internal/api/ui_dist/index.html

if ! git diff --quiet -- internal/api/ui_dist; then
  echo "internal/api/ui_dist is stale; run make build and commit the regenerated embedded UI bundle." >&2
  git diff --stat -- internal/api/ui_dist >&2
  exit 1
fi

echo "internal/api/ui_dist is fresh"
