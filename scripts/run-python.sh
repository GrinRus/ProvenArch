#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
required_python_version="${ACP_PYTHON_VERSION:-}"
if [[ -z "$required_python_version" && -f "$repo_root/.python-version" ]]; then
  required_python_version="$(tr -d '[:space:]' < "$repo_root/.python-version")"
fi

read_python_version() {
  local python_bin="$1"
  local version_line
  version_line="$("$python_bin" --version 2>&1 || true)"
  case "$version_line" in
    "Python "* )
      version_line="${version_line#Python }"
      printf '%s\n' "${version_line%% *}"
      ;;
    *)
      printf '\n'
      ;;
  esac
}

major_minor_version() {
  local version="$1"
  printf '%s\n' "$version" | awk -F. '{print $1 "." $2}'
}

declare -a candidate_bins=()
if [[ -n "${ACP_PYTHON_BIN:-}" ]]; then
  candidate_bins+=("${ACP_PYTHON_BIN}")
fi
if [[ -n "${ACP_PYTHON_TOOL_CANDIDATES:-}" ]]; then
  IFS=':' read -r -a candidate_dirs <<< "${ACP_PYTHON_TOOL_CANDIDATES}"
  for dir in "${candidate_dirs[@]}"; do
    [[ -n "$dir" ]] || continue
    candidate_bins+=("$dir/python3" "$dir/python")
    if [[ -n "$required_python_version" ]]; then
      candidate_bins+=("$dir/python${required_python_version}" "$dir/python$(major_minor_version "$required_python_version")")
    fi
  done
fi
candidate_bins+=("$repo_root/.venv/bin/python")
if [[ -n "${ACP_PYTHON_BASE_BIN:-}" ]]; then
  candidate_bins+=("${ACP_PYTHON_BASE_BIN}")
fi
if [[ -n "$required_python_version" ]]; then
  required_major_minor="$(major_minor_version "$required_python_version")"
  candidate_bins+=("$HOME/.pyenv/versions/$required_python_version/bin/python" "$HOME/.local/bin/python$required_python_version" "$HOME/.local/bin/python$required_major_minor")
fi
candidate_bins+=("/opt/homebrew/bin/python3" "/usr/local/bin/python3" "/usr/bin/python3")
IFS=':' read -r -a path_dirs <<< "${PATH:-}"
for dir in "${path_dirs[@]}"; do
  [[ -n "$dir" ]] || continue
  candidate_bins+=("$dir/python3" "$dir/python")
  if [[ -n "$required_python_version" ]]; then
    candidate_bins+=("$dir/python${required_python_version}" "$dir/python$(major_minor_version "$required_python_version")")
  fi
done

seen=""
first_available=""
first_mismatch=""
for python_bin in "${candidate_bins[@]}"; do
  [[ -n "$python_bin" ]] || continue
  case ":$seen:" in
    *":$python_bin:"*) continue ;;
  esac
  seen="${seen}:$python_bin"
  [[ -x "$python_bin" ]] || continue
  if [[ -z "$first_available" ]]; then
    first_available="$python_bin"
  fi
  actual_version="$(read_python_version "$python_bin")"
  if [[ -n "$required_python_version" && "$actual_version" != "$required_python_version" ]]; then
    if [[ -z "$first_mismatch" ]]; then
      first_mismatch="$python_bin ($actual_version)"
    fi
    continue
  fi
  # shellcheck disable=SC2093 # This wrapper intentionally hands off to the selected Python binary.
  exec "$python_bin" "$@"
done

if [[ -n "$required_python_version" ]]; then
  echo "Python $required_python_version is required by .python-version; no matching Python toolchain was found." >&2
  echo "Install Python $required_python_version or set ACP_PYTHON_BASE_BIN=/path/to/python$required_python_version for worktree bootstrap." >&2
  echo "ACP_PYTHON_BIN remains an explicit test-runtime override and takes precedence over the worktree venv." >&2
  if [[ -n "$first_mismatch" ]]; then
    echo "First discovered Python was $first_mismatch." >&2
  fi
  exit 1
fi

if [[ -n "$first_available" ]]; then
  # shellcheck disable=SC2093 # This wrapper intentionally hands off to the selected Python binary.
  exec "$first_available" "$@"
fi

# shellcheck disable=SC2093 # This wrapper intentionally hands off to Python.
exec python3 "$@"
