#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
mode="${1:-}"
if [[ "$mode" != "" && "$mode" != "--tools-only" ]]; then
  echo "usage: $0 [--tools-only]" >&2
  exit 2
fi
cd "$repo_root"

failed=0
check() {
  local label="$1"
  shift
  if "$@"; then
    printf 'OK: %s\n' "$label"
  else
    printf 'MISSING: %s\n' "$label" >&2
    failed=1
  fi
}

check "pinned Go" ./scripts/run-go.sh version
check "pinned Python" ./scripts/run-python.sh --version
if npm_version="$(./scripts/run-npm.sh --version)" && [[ "$npm_version" == 10.* ]]; then
  echo "OK: pinned Node and npm 10.x"
else
  echo "MISSING: pinned Node and npm 10.x (got ${npm_version:-unavailable})" >&2
  failed=1
fi
check "Git" git --version
check "ShellCheck" shellcheck --version

if [[ "$mode" != "--tools-only" ]]; then
  check "Python development dependencies (make bootstrap)" ./scripts/run-python.sh -c '
from importlib.metadata import version
from pathlib import Path
for requirement in Path("scripts/requirements-dev.txt").read_text().splitlines():
    if requirement.strip() and not requirement.startswith("#"):
        name, expected = requirement.split("==")
        actual = version(name)
        if actual != expected:
            raise SystemExit(f"{name}: expected {expected}, got {actual}")
'
  check "UI dependencies (make bootstrap)" test -x ui/node_modules/.bin/vitest
  check "contract dependencies (make bootstrap)" test -x tools/contracts/node_modules/.bin/ajv
fi

if [[ "$failed" != 0 ]]; then
  echo "Resolve missing toolchains using .go-version/.node-version/.python-version and the resolver diagnostics; run make bootstrap to install dependencies." >&2
fi
exit "$failed"
