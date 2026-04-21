#!/usr/bin/env bash
set -Eeuo pipefail

tool="${1:-}"
case "$tool" in
  node|npm)
    ;;
  *)
    echo "usage: $0 <node|npm>" >&2
    exit 1
    ;;
esac

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
  if [[ -n "$desired_node_arch" && -x "$node_path" ]]; then
    actual_arch="$("$node_path" -p 'process.arch' 2>/dev/null || true)"
    if [[ "$actual_arch" == "$desired_node_arch" ]]; then
      printf '%s\n' "$tool_path"
      exit 0
    fi
  fi
done

if command -v "$tool" >/dev/null 2>&1; then
  command -v "$tool"
  exit 0
fi

if [[ -n "$first_available" ]]; then
  printf '%s\n' "$first_available"
  exit 0
fi

printf '%s\n' "$tool"
