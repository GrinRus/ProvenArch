#!/usr/bin/env sh
set -eu

repo="${ACP_REPO:-GrinRus/ProvenArch}"
version="${ACP_VERSION:-latest}"
install_dir="${INSTALL_DIR:-$HOME/.local/bin}"
base_url="${ACP_INSTALL_BASE_URL:-}"

os_name="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch_name="$(uname -m)"

case "$os_name" in
  darwin|linux)
    ;;
  *)
    echo "unsupported OS: $os_name" >&2
    exit 1
    ;;
esac

case "$arch_name" in
  x86_64|amd64)
    arch_name="amd64"
    ;;
  arm64|aarch64)
    arch_name="arm64"
    ;;
  *)
    echo "unsupported architecture: $arch_name" >&2
    exit 1
    ;;
esac

archive_name="acp_${os_name}_${arch_name}.tar.gz"
checksum_name="checksums.txt"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

download() {
  url="$1"
  target="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$target"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$target"
    return
  fi
  echo "curl or wget is required" >&2
  exit 1
}

resolve_latest_version() {
  releases_path="$tmp_dir/releases.json"
  download "https://api.github.com/repos/${repo}/releases?per_page=1" "$releases_path"
  resolved_version="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$releases_path" | head -n 1)"
  if [ -z "$resolved_version" ]; then
    echo "could not resolve latest release for ${repo}" >&2
    exit 1
  fi
  printf '%s\n' "$resolved_version"
}

if [ -z "$base_url" ]; then
  if [ "$version" = "latest" ]; then
    version="$(resolve_latest_version)"
  fi
  base_url="https://github.com/${repo}/releases/download/${version}"
fi

archive_path="$tmp_dir/$archive_name"
checksum_path="$tmp_dir/$checksum_name"

download "${base_url}/${archive_name}" "$archive_path"
download "${base_url}/${checksum_name}" "$checksum_path"

checksum_line="$(grep "  ${archive_name}\$" "$checksum_path" || true)"
if [ -z "$checksum_line" ]; then
  echo "checksum for ${archive_name} not found in ${checksum_name}" >&2
  exit 1
fi
printf '%s\n' "$checksum_line" > "$tmp_dir/checksum.selected"

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmp_dir" && sha256sum -c checksum.selected)
elif command -v shasum >/dev/null 2>&1; then
  (cd "$tmp_dir" && shasum -a 256 -c checksum.selected)
else
  echo "sha256sum or shasum is required for checksum verification" >&2
  exit 1
fi

extract_dir="$tmp_dir/extract"
mkdir -p "$extract_dir"
tar -xzf "$archive_path" -C "$extract_dir"

if [ ! -f "$extract_dir/acp" ]; then
  echo "archive did not contain acp binary" >&2
  exit 1
fi

mkdir -p "$install_dir"
cp "$extract_dir/acp" "$install_dir/acp"
chmod 0755 "$install_dir/acp"

echo "acp installed to ${install_dir}/acp"
case ":$PATH:" in
  *":$install_dir:"*)
    ;;
  *)
    echo "add ${install_dir} to PATH to run acp from any directory"
    ;;
esac
