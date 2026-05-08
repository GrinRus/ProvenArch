#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tool="${1:-}"
case "$tool" in
  node|npm)
    ;;
  *)
    echo "usage: $0 <node|npm>" >&2
    exit 1
    ;;
esac

required_node_version="${ACP_NODE_VERSION:-}"
if [[ -z "$required_node_version" && -f "$repo_root/.node-version" ]]; then
  required_node_version="$(tr -d '[:space:]' < "$repo_root/.node-version")"
fi
node_version_check="${ACP_NODE_VERSION_CHECK:-1}"

host_arch="$(uname -m 2>/dev/null || true)"
desired_node_arch=""
case "$host_arch" in
  arm64|aarch64)
    desired_node_arch="arm64"
    ;;
  x86_64|amd64)
    desired_node_arch="x64"
    ;;
esac

read_node_attr() {
  local node_path="$1"
  local expr="$2"
  "$node_path" -p "$expr" 2>/dev/null || true
}

node_matches_required_version() {
  local node_path="$1"
  if [[ "$node_version_check" != "1" || -z "$required_node_version" ]]; then
    return 0
  fi
  local actual_version
  actual_version="$(read_node_attr "$node_path" "process.versions.node")"
  [[ "$actual_version" == "$required_node_version" ]]
}

declare -a candidate_dirs=()
if [[ -n "${ACP_NODE_TOOL_CANDIDATES:-}" ]]; then
  IFS=':' read -r -a candidate_dirs <<< "${ACP_NODE_TOOL_CANDIDATES}"
fi
if [[ "${ACP_NODE_TOOL_CANDIDATES_ONLY:-0}" != "1" ]]; then
  case "$desired_node_arch" in
    arm64)
      candidate_dirs+=("/opt/homebrew/bin" "/usr/local/bin")
      ;;
    x64)
      candidate_dirs+=("/usr/local/bin" "/opt/homebrew/bin")
      ;;
    *)
      candidate_dirs+=("/opt/homebrew/bin" "/usr/local/bin")
      ;;
  esac
  IFS=':' read -r -a path_dirs <<< "${PATH:-}"
  candidate_dirs+=("${path_dirs[@]}")
fi

seen_dirs=""
first_available=""
first_version_compatible=""
first_version_mismatch=""
for dir in "${candidate_dirs[@]}"; do
  [[ -n "$dir" ]] || continue
  case ":$seen_dirs:" in
    *":$dir:"*)
      continue
      ;;
  esac
  seen_dirs="${seen_dirs}:$dir"
  if [[ ! -x "$dir/$tool" ]]; then
    continue
  fi
  tool_path="$dir/$tool"
  node_path="$dir/node"
  if [[ -z "$first_available" ]]; then
    first_available="$tool_path"
  fi
  if [[ ! -x "$node_path" ]]; then
    continue
  fi
  if ! node_matches_required_version "$node_path"; then
    if [[ -z "$first_version_mismatch" ]]; then
      first_version_mismatch="$node_path"
    fi
    continue
  fi
  if [[ -z "$first_version_compatible" ]]; then
    first_version_compatible="$tool_path"
  fi
  if [[ -n "$desired_node_arch" ]]; then
    actual_arch="$(read_node_attr "$node_path" "process.arch")"
    if [[ "$actual_arch" != "$desired_node_arch" ]]; then
      continue
    fi
  fi
  printf '%s\n' "$tool_path"
  exit 0
done

if [[ -n "$first_version_compatible" ]]; then
  printf '%s\n' "$first_version_compatible"
  exit 0
fi

if command -v "$tool" >/dev/null 2>&1; then
  tool_path="$(command -v "$tool")"
  tool_dir="$(cd "$(dirname "$tool_path")" && pwd)"
  node_path="$tool_dir/node"
  if [[ -x "$node_path" ]] && node_matches_required_version "$node_path"; then
    printf '%s\n' "$tool_path"
    exit 0
  fi
fi

if [[ "$node_version_check" == "1" && -n "$required_node_version" ]]; then
  echo "Node.js $required_node_version is required by .node-version; no matching $tool toolchain was found." >&2
  echo "Install Node.js $required_node_version or set ACP_NODE_TOOL_CANDIDATES=/path/to/node-$required_node_version/bin." >&2
  if [[ -n "$first_version_mismatch" ]]; then
    actual_version="$(read_node_attr "$first_version_mismatch" "process.versions.node")"
    echo "First discovered node was $first_version_mismatch ($actual_version)." >&2
  fi
  exit 1
fi

printf '%s\n' "$tool"
