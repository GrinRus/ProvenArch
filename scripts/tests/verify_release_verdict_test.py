import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


class VerifyReleaseVerdictTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.script = cls.repo_root / "scripts" / "verify-release-verdict.py"

    def run_verifier(self, path: Path) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(self.script), str(path)],
            cwd=self.repo_root,
            capture_output=True,
            text=True,
        )

    def write_verdict(self, root: Path, payload: object) -> Path:
        path = root / "release_verdict_test.json"
        path.write_text(json.dumps(payload), encoding="utf-8")
        return path

    def test_accepts_pass_ready_verdict_with_passed_release_contract(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_verdict(
                Path(tmp),
                {
                    "verdict": "PASS",
                    "release_state": "RELEASE READY",
                    "release_contract": {"contract_status": "passed"},
                },
            )

            result = self.run_verifier(path)

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("release verdict ready", result.stdout)

    def test_rejects_fail_verdict(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_verdict(
                Path(tmp),
                {
                    "verdict": "FAIL",
                    "release_state": "RELEASE BLOCKED",
                    "release_contract": {"contract_status": "failed"},
                },
            )

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("verdict must be PASS", result.stderr)
        self.assertIn("release_contract.contract_status must be passed", result.stderr)

    def test_rejects_missing_file(self) -> None:
        result = self.run_verifier(Path("/tmp/acp-missing-release-verdict.json"))

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release verdict file not found", result.stderr)

    def test_rejects_invalid_json(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "release_verdict_invalid.json"
            path.write_text("{not-json", encoding="utf-8")

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("invalid release verdict JSON", result.stderr)

    def test_rejects_missing_release_contract(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_verdict(
                Path(tmp),
                {
                    "verdict": "PASS",
                    "release_state": "RELEASE READY",
                },
            )

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release_contract must be an object", result.stderr)


if __name__ == "__main__":
    unittest.main()
