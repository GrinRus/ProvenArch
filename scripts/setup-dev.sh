#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$repo_root"
bash ./scripts/dev-preflight.sh --tools-only

if [[ -n "${ACP_PYTHON_BIN:-}" || -n "${ACP_PYTHON_TOOL_CANDIDATES:-}" ]]; then
  if ! ./scripts/run-python.sh -c '
from pathlib import Path
import sys
expected = Path(".venv").resolve()
actual = Path(sys.prefix).resolve()
if actual != expected:
    print(f"Selected Python runtime uses {actual}; bootstrap installs only into {expected}.", file=sys.stderr)
    raise SystemExit(1)
'; then
    echo "ACP_PYTHON_BIN/ACP_PYTHON_TOOL_CANDIDATES selects a runtime outside this worktree venv." >&2
    echo "For bootstrap, set ACP_PYTHON_BASE_BIN to the pinned base interpreter and unset those runtime overrides." >&2
    echo "To keep an intentional external test runtime, prepare its dependencies separately; bootstrap will not install into it." >&2
    exit 1
  fi
fi

if [[ ! -x .venv/bin/python ]]; then
  ./scripts/run-python.sh -m venv .venv
fi
# Reject a stale venv rather than installing dependencies into the wrong interpreter.
expected_python="Python $(tr -d '[:space:]' < .python-version)"
if [[ "$(./.venv/bin/python --version)" != "$expected_python" ]]; then
  echo ".venv must use $expected_python; recreate this worktree's venv with the pinned Python." >&2
  exit 1
fi
# Download the declared modules; setup must not tidy or rewrite module declarations.
./scripts/run-go.sh mod download
./scripts/run-npm.sh ci --prefix tools/contracts --ignore-scripts --audit=false --fund=false
./scripts/run-npm.sh ci --prefix ui
./.venv/bin/python -m pip install --disable-pip-version-check -r scripts/requirements-dev.txt
bash ./scripts/dev-preflight.sh
