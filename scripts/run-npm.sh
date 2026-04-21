#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
npm_bin="${ACP_NPM_BIN:-$("$PROVENARCH_ROOT/scripts/resolve-node-tool.sh" npm)}"
npm_dir="$(cd "$(dirname "$npm_bin")" && pwd)"

PATH="$npm_dir:${PATH:-}" exec "$npm_bin" "$@"
