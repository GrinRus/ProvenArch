import hashlib
import os
import platform
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]


def release_asset_name() -> str:
    os_name = platform.system().lower()
    machine = platform.machine().lower()
    if machine in {"x86_64", "amd64"}:
        arch = "amd64"
    elif machine in {"arm64", "aarch64"}:
        arch = "arm64"
    else:
        raise unittest.SkipTest(f"unsupported test architecture: {machine}")
    if os_name not in {"darwin", "linux"}:
        raise unittest.SkipTest(f"unsupported test OS: {os_name}")
    return f"acp_{os_name}_{arch}.tar.gz"


def write_mock_release(release_dir: Path, archive_name: str, checksum_ok: bool = True) -> None:
    payload_dir = release_dir / "payload"
    payload_dir.mkdir()
    binary = payload_dir / "acp"
    binary.write_text(
        "#!/usr/bin/env sh\n"
        "if [ \"${1:-}\" = \"version\" ] || [ \"${1:-}\" = \"--version\" ]; then\n"
        "  echo 'acp version test'\n"
        "  echo 'commit: test'\n"
        "  echo 'built: test'\n"
        "  exit 0\n"
        "fi\n"
        "echo acp-test\n",
        encoding="utf-8",
    )
    binary.chmod(0o755)

    archive = release_dir / archive_name
    with tarfile.open(archive, "w:gz") as handle:
        handle.add(binary, arcname="acp")

    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    if not checksum_ok:
        digest = "0" * 64
    (release_dir / "checksums.txt").write_text(f"{digest}  {archive_name}\n", encoding="utf-8")


class InstallScriptTest(unittest.TestCase):
    def test_installs_from_mocked_release(self) -> None:
        archive_name = release_asset_name()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            release_dir = root / "release"
            release_dir.mkdir()
            install_dir = root / "bin"
            write_mock_release(release_dir, archive_name)

            env = os.environ.copy()
            env["ACP_INSTALL_BASE_URL"] = release_dir.as_uri()
            env["INSTALL_DIR"] = str(install_dir)
            result = subprocess.run(
                ["sh", str(REPO_ROOT / "install.sh")],
                cwd=REPO_ROOT,
                env=env,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            installed = install_dir / "acp"
            self.assertTrue(installed.exists())
            self.assertIn("acp installed", result.stdout)
            version_result = subprocess.run(
                [str(installed), "version"],
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )
            self.assertEqual(version_result.returncode, 0, version_result.stderr)
            self.assertIn("acp version test", version_result.stdout)

    def test_rejects_checksum_mismatch(self) -> None:
        archive_name = release_asset_name()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            release_dir = root / "release"
            release_dir.mkdir()
            install_dir = root / "bin"
            write_mock_release(release_dir, archive_name, checksum_ok=False)

            env = os.environ.copy()
            env["ACP_INSTALL_BASE_URL"] = release_dir.as_uri()
            env["INSTALL_DIR"] = str(install_dir)
            result = subprocess.run(
                ["sh", str(REPO_ROOT / "install.sh")],
                cwd=REPO_ROOT,
                env=env,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )

            self.assertNotEqual(result.returncode, 0)
            self.assertFalse((install_dir / "acp").exists())

    def test_version_and_repo_are_used_in_release_urls(self) -> None:
        archive_name = release_asset_name()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            release_dir = root / "release"
            release_dir.mkdir()
            install_dir = root / "bin"
            fake_bin = root / "fake-bin"
            fake_bin.mkdir()
            curl_log = root / "curl.log"
            write_mock_release(release_dir, archive_name)

            fake_curl = fake_bin / "curl"
            fake_curl.write_text(
                f"""#!/usr/bin/env sh
set -eu
url=""
target=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      shift
      target="$1"
      ;;
    http*)
      url="$1"
      ;;
  esac
  shift
done
printf '%s\\n' "$url" >> "$ACP_CURL_LOG"
case "$url" in
  */checksums.txt)
    cp "$ACP_MOCK_RELEASE_DIR/checksums.txt" "$target"
    ;;
  */{archive_name})
    cp "$ACP_MOCK_RELEASE_DIR/{archive_name}" "$target"
    ;;
  *)
    echo "unexpected URL: $url" >&2
    exit 2
    ;;
esac
""",
                encoding="utf-8",
            )
            fake_curl.chmod(0o755)

            env = os.environ.copy()
            env["PATH"] = f"{fake_bin}{os.pathsep}{env['PATH']}"
            env["ACP_VERSION"] = "v9.8.7"
            env["ACP_REPO"] = "example/repo"
            env["ACP_CURL_LOG"] = str(curl_log)
            env["ACP_MOCK_RELEASE_DIR"] = str(release_dir)
            env["INSTALL_DIR"] = str(install_dir)
            result = subprocess.run(
                ["sh", str(REPO_ROOT / "install.sh")],
                cwd=REPO_ROOT,
                env=env,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            urls = curl_log.read_text(encoding="utf-8").splitlines()
            self.assertIn(
                f"https://github.com/example/repo/releases/download/v9.8.7/{archive_name}",
                urls,
            )
            self.assertIn(
                "https://github.com/example/repo/releases/download/v9.8.7/checksums.txt",
                urls,
            )

    def test_latest_resolves_prerelease_release_via_github_api(self) -> None:
        archive_name = release_asset_name()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            release_dir = root / "release"
            release_dir.mkdir()
            install_dir = root / "bin"
            fake_bin = root / "fake-bin"
            fake_bin.mkdir()
            curl_log = root / "curl.log"
            write_mock_release(release_dir, archive_name)

            fake_curl = fake_bin / "curl"
            fake_curl.write_text(
                f"""#!/usr/bin/env sh
set -eu
url=""
target=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      shift
      target="$1"
      ;;
    http*)
      url="$1"
      ;;
  esac
  shift
done
printf '%s\\n' "$url" >> "$ACP_CURL_LOG"
case "$url" in
  https://api.github.com/repos/example/repo/releases\\?per_page=1)
    printf '%s\\n' '[{{"tag_name":"v0.1.0","prerelease":true}}]' > "$target"
    ;;
  */checksums.txt)
    cp "$ACP_MOCK_RELEASE_DIR/checksums.txt" "$target"
    ;;
  */{archive_name})
    cp "$ACP_MOCK_RELEASE_DIR/{archive_name}" "$target"
    ;;
  *)
    echo "unexpected URL: $url" >&2
    exit 2
    ;;
esac
""",
                encoding="utf-8",
            )
            fake_curl.chmod(0o755)

            env = os.environ.copy()
            env["PATH"] = f"{fake_bin}{os.pathsep}{env['PATH']}"
            env["ACP_REPO"] = "example/repo"
            env["ACP_CURL_LOG"] = str(curl_log)
            env["ACP_MOCK_RELEASE_DIR"] = str(release_dir)
            env["INSTALL_DIR"] = str(install_dir)
            result = subprocess.run(
                ["sh", str(REPO_ROOT / "install.sh")],
                cwd=REPO_ROOT,
                env=env,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
            )

            self.assertEqual(result.returncode, 0, result.stderr)
            urls = curl_log.read_text(encoding="utf-8").splitlines()
            self.assertIn("https://api.github.com/repos/example/repo/releases?per_page=1", urls)
            self.assertIn(
                f"https://github.com/example/repo/releases/download/v0.1.0/{archive_name}",
                urls,
            )
            self.assertIn(
                "https://github.com/example/repo/releases/download/v0.1.0/checksums.txt",
                urls,
            )


if __name__ == "__main__":
    unittest.main()
