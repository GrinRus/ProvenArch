import os
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path


class RunPythonTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.runner = cls.repo_root / "scripts" / "run-python.sh"

    def _write_fake_python(self, root: Path, version: str, name: str = "python3") -> Path:
        root.mkdir(parents=True, exist_ok=True)
        python_path = root / name
        python_path.write_text(
            "\n".join(
                [
                    "#!/usr/bin/env bash",
                    "set -Eeuo pipefail",
                    'if [[ "${1:-}" == "--version" || "${1:-}" == "-V" ]]; then',
                    f"  printf '%s\\n' 'Python {version}'",
                    "  exit 0",
                    "fi",
                    'printf "%s\\n" "$@"',
                ]
            )
            + "\n",
            encoding="utf-8",
        )
        python_path.chmod(python_path.stat().st_mode | stat.S_IXUSR)
        return python_path

    def test_uses_candidate_matching_required_python_version(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            good_dir = tmp_root / "good"
            self._write_fake_python(good_dir, "3.10.8")
            result = subprocess.run(
                ["bash", str(self.runner), "--version"],
                cwd=self.repo_root,
                env={
                    **os.environ,
                    "HOME": str(tmp_root / "home"),
                    "ACP_PYTHON_TOOL_CANDIDATES": str(good_dir),
                    "PATH": "/usr/bin:/bin",
                },
                capture_output=True,
                text=True,
                check=True,
            )
            self.assertIn("Python 3.10.8", result.stdout)

    def test_rejects_wrong_python_version_before_running_command(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            old_dir = tmp_root / "old"
            self._write_fake_python(old_dir, "3.11.9")
            result = subprocess.run(
                ["bash", str(self.runner), "-c", "print('should-not-run')"],
                cwd=self.repo_root,
                env={
                    **os.environ,
                    "HOME": str(tmp_root / "home"),
                    "ACP_PYTHON_VERSION": "3.10.9",
                    "ACP_PYTHON_TOOL_CANDIDATES": str(old_dir),
                    "PATH": "/usr/bin:/bin",
                },
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(0, result.returncode)
            self.assertIn("Python 3.10.9 is required", result.stderr)
            self.assertNotIn("should-not-run", result.stdout)


if __name__ == "__main__":
    unittest.main()
