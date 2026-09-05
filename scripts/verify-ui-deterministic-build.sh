#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -gt 1 || "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  echo "usage: $0 [WORKTREE|HEAD|<git-ref>] (default: WORKTREE)"
  [[ "$#" -le 1 ]]
  exit
fi
ref="${1:-WORKTREE}"
repo_root="$(git rev-parse --show-toplevel)"
if [[ ! -d "$repo_root/ui/node_modules" ]]; then
  echo "UI dependencies are missing; run make bootstrap before verifying determinism." >&2
  exit 1
fi
tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/acp-ui-determinism.XXXXXX")"

cleanup() {
  rm -rf "$tmp_root"
}
trap cleanup EXIT

prepare_root() {
  local root="$1"
  mkdir -p "$root"
  if [ "$ref" = "WORKTREE" ]; then
    (
      cd "$repo_root"
      git ls-files -z -co --exclude-standard | while IFS= read -r -d '' path; do
        if [[ -e "$path" || -L "$path" ]]; then
          printf '%s\0' "$path"
        fi
      done | tar --null -T - -cf -
    ) | tar -xf - -C "$root"
  else
    git -C "$repo_root" archive --format=tar "$ref" | tar -xf - -C "$root"
    for dependency_file in package.json package-lock.json; do
      if ! cmp -s "$root/ui/$dependency_file" "$repo_root/ui/$dependency_file"; then
        echo "UI $dependency_file differs for ref $ref; prepare dependencies in a separate checkout of that ref before verifying it." >&2
        exit 1
      fi
    done
  fi
  ln -s "$repo_root/ui/node_modules" "$root/ui/node_modules"
}

build_manifest() {
  local root="$1"
  local manifest="$2"
  rm -rf "$root/ui/dist"
  "$root/scripts/run-npm.sh" run build --prefix "$root/ui" >/dev/null
  (
    cd "$root/ui/dist"
    find . -type f -print | LC_ALL=C sort | while IFS= read -r file; do
      shasum -a 256 "$file"
    done
  ) >"$manifest"
}

left="$tmp_root/left"
right="$tmp_root/right"
prepare_root "$left"
prepare_root "$right"
build_manifest "$left" "$tmp_root/left.sha256"
build_manifest "$right" "$tmp_root/right.sha256"

if ! diff -u "$tmp_root/left.sha256" "$tmp_root/right.sha256"; then
  echo "UI build is not deterministic for ref $ref" >&2
  exit 1
fi

echo "UI build is deterministic for ref $ref"
