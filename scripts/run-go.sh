#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
required_go_version="${ACP_GO_VERSION:-}"
if [[ -z "$required_go_version" && -f "$repo_root/.go-version" ]]; then
  required_go_version="$(tr -d '[:space:]' < "$repo_root/.go-version")"
fi

read_go_version() {
  local go_bin="$1"
  local version_line
  version_line="$("$go_bin" version 2>/dev/null || true)"
  case "$version_line" in
    "go version go"* )
      version_line="${version_line#go version go}"
      printf '%s\n' "${version_line%% *}"
      ;;
    *)
      printf '\n'
      ;;
  esac
}

declare -a candidate_bins=()
if [[ -n "${ACP_GO_BIN:-}" ]]; then
  candidate_bins+=("${ACP_GO_BIN}")
fi
if [[ -n "${ACP_GO_TOOL_CANDIDATES:-}" ]]; then
  IFS=':' read -r -a candidate_dirs <<< "${ACP_GO_TOOL_CANDIDATES}"
  for dir in "${candidate_dirs[@]}"; do
    [[ -n "$dir" ]] || continue
    candidate_bins+=("$dir/go")
    [[ -n "$required_go_version" ]] && candidate_bins+=("$dir/go${required_go_version}")
  done
fi
if [[ -n "$required_go_version" ]]; then
  candidate_bins+=("$HOME/sdk/go${required_go_version}/bin/go" "$HOME/go/bin/go${required_go_version}")
fi
candidate_bins+=("/opt/homebrew/bin/go" "/usr/local/bin/go")
IFS=':' read -r -a path_dirs <<< "${PATH:-}"
for dir in "${path_dirs[@]}"; do
  [[ -n "$dir" ]] || continue
  candidate_bins+=("$dir/go")
  [[ -n "$required_go_version" ]] && candidate_bins+=("$dir/go${required_go_version}")
done

seen=""
first_available=""
first_mismatch=""
for go_bin in "${candidate_bins[@]}"; do
  [[ -n "$go_bin" ]] || continue
  case ":$seen:" in
    *":$go_bin:"*) continue ;;
  esac
  seen="${seen}:$go_bin"
  [[ -x "$go_bin" ]] || continue
  if [[ -z "$first_available" ]]; then
    first_available="$go_bin"
  fi
  actual_version="$(read_go_version "$go_bin")"
  if [[ -n "$required_go_version" && "$actual_version" != "$required_go_version" ]]; then
    if [[ -z "$first_mismatch" ]]; then
      first_mismatch="$go_bin ($actual_version)"
    fi
    continue
  fi
  exec "$go_bin" "$@"
done

if [[ -n "$required_go_version" ]]; then
  echo "Go $required_go_version is required by .go-version; no matching Go toolchain was found." >&2
  echo "Install Go $required_go_version or set ACP_GO_BIN=/path/to/go${required_go_version}/bin/go." >&2
  if [[ -n "$first_mismatch" ]]; then
    echo "First discovered Go was $first_mismatch." >&2
  fi
  exit 1
fi

if [[ -n "$first_available" ]]; then
  exec "$first_available" "$@"
fi

exec go "$@"
