import os
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path


class RunGoTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.runner = cls.repo_root / "scripts" / "run-go.sh"

    def _write_fake_go(self, root: Path, version: str, name: str = "go") -> Path:
        root.mkdir(parents=True, exist_ok=True)
        go_path = root / name
        go_path.write_text(
            "\n".join(
                [
                    "#!/usr/bin/env bash",
                    "set -Eeuo pipefail",
                    'if [[ "${1:-}" == "version" ]]; then',
                    f"  printf '%s\\n' 'go version go{version} fake/arch'",
                    "  exit 0",
                    "fi",
                    'printf "%s\\n" "$@"',
                ]
            )
            + "\n",
            encoding="utf-8",
        )
        go_path.chmod(go_path.stat().st_mode | stat.S_IXUSR)
        return go_path

    def _runner_env(self, **overrides: str) -> dict[str, str]:
        """Keep runner-selection tests independent of the invoking shell."""
        env = os.environ.copy()
        for key in ("ACP_GO_BIN", "ACP_GO_VERSION", "ACP_GO_TOOL_CANDIDATES"):
            env.pop(key, None)
        env.update(overrides)
        return env

    def test_uses_candidate_matching_required_go_version(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            good_dir = tmp_root / "good"
            self._write_fake_go(good_dir, "1.25.10")
            result = subprocess.run(
                [str(self.runner), "version"],
                cwd=self.repo_root,
                env=self._runner_env(
                    HOME=str(tmp_root / "home"),
                    ACP_GO_TOOL_CANDIDATES=str(good_dir),
                    PATH="/usr/bin:/bin",
                ),
                capture_output=True,
                text=True,
                check=True,
            )
            self.assertIn("go version go1.25.10", result.stdout)

    def test_rejects_wrong_go_version_when_no_exact_toolchain_exists(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            old_dir = tmp_root / "old"
            self._write_fake_go(old_dir, "1.20.3")
            result = subprocess.run(
                [str(self.runner), "version"],
                cwd=self.repo_root,
                env=self._runner_env(
                    HOME=str(tmp_root / "home"),
                    ACP_GO_TOOL_CANDIDATES=str(old_dir),
                    PATH="/usr/bin:/bin",
                ),
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(0, result.returncode)
            self.assertIn("Go 1.25.10 is required", result.stderr)
