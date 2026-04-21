import os
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path


class ResolveNodeToolTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.resolver = cls.repo_root / "scripts" / "resolve-node-tool.sh"
        host_arch = subprocess.check_output(["uname", "-m"], text=True).strip()
        if host_arch in {"arm64", "aarch64"}:
            cls.expected_host_node_arch = "arm64"
        elif host_arch in {"x86_64", "amd64"}:
            cls.expected_host_node_arch = "x64"
        else:
            cls.expected_host_node_arch = ""

    def _write_fake_toolchain(self, root: Path, node_arch: str) -> None:
        root.mkdir(parents=True, exist_ok=True)
        node_path = root / "node"
        node_path.write_text(
            "\n".join(
                [
                    "#!/usr/bin/env bash",
                    "set -Eeuo pipefail",
                    'if [[ "${1:-}" == "-p" ]]; then',
                    f"  printf '%s\\n' '{node_arch}'",
                    "  exit 0",
                    "fi",
                    "printf '%s\\n' \"$0\"",
                ]
            )
            + "\n",
            encoding="utf-8",
        )
        npm_path = root / "npm"
        npm_path.write_text(
            "\n".join(
                [
                    "#!/usr/bin/env bash",
                    "set -Eeuo pipefail",
                    "printf '%s\\n' \"$0\"",
                ]
            )
            + "\n",
            encoding="utf-8",
        )
        node_path.chmod(node_path.stat().st_mode | stat.S_IXUSR)
        npm_path.chmod(npm_path.stat().st_mode | stat.S_IXUSR)

    def test_prefers_candidate_matching_host_arch(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            x64_dir = tmp_root / "x64"
            arm64_dir = tmp_root / "arm64"
            self._write_fake_toolchain(x64_dir, "x64")
            self._write_fake_toolchain(arm64_dir, "arm64")
            env = {
                "ACP_NODE_TOOL_CANDIDATES": f"{x64_dir}:{arm64_dir}",
                "ACP_NODE_TOOL_CANDIDATES_ONLY": "1",
                "PATH": "/usr/bin:/bin",
            }
            result = subprocess.run(
                [str(self.resolver), "npm"],
                cwd=self.repo_root,
                env={**os.environ, **env},
                capture_output=True,
                text=True,
                check=True,
            )
            resolved = Path(result.stdout.strip())
            if self.expected_host_node_arch == "arm64":
                self.assertEqual(arm64_dir / "npm", resolved)
            elif self.expected_host_node_arch == "x64":
                self.assertEqual(x64_dir / "npm", resolved)
            else:
                self.assertEqual(x64_dir / "npm", resolved)

    def test_falls_back_to_first_available_when_no_matching_arch_exists(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            odd_dir = tmp_root / "odd"
            self._write_fake_toolchain(odd_dir, "weird-arch")
            result = subprocess.run(
                [str(self.resolver), "npm"],
                cwd=self.repo_root,
                env={
                    **os.environ,
                    "ACP_NODE_TOOL_CANDIDATES": str(odd_dir),
                    "ACP_NODE_TOOL_CANDIDATES_ONLY": "1",
                    "PATH": "/usr/bin:/bin",
                },
                capture_output=True,
                text=True,
                check=True,
            )
            self.assertEqual(str(odd_dir / "npm"), result.stdout.strip())


if __name__ == "__main__":
    unittest.main()
