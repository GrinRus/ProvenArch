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
        matrix_id = payload.get("matrix_id") if isinstance(payload, dict) else None
        suffix = str(matrix_id).strip() if matrix_id else "test"
        path = root / f"release_verdict_{suffix}.json"
        path.write_text(json.dumps(payload), encoding="utf-8")
        return path

    def write_accepted_assessments(self, root: Path, matrix_id: str) -> None:
        (root / f"swe_ux_assessment_{matrix_id}.md").write_text(
            f"# UX\n\n- matrix_id: {matrix_id}\n- decision: accepted\n",
            encoding="utf-8",
        )
        (root / f"swe_artifact_quality_assessment_{matrix_id}.md").write_text(
            f"# Artifacts\n\n- matrix_id: {matrix_id}\n- decision: accepted\n",
            encoding="utf-8",
        )

    def ready_payload(self) -> dict[str, object]:
        return {
            "matrix_id": "test-matrix",
            "verdict": "PASS",
            "release_state": "RELEASE READY",
            "release_contract": {
                "mode": "release",
                "contract_status": "passed",
                "selected_providers": ["qwen-code", "claude-code", "codex-code"],
                "selected_run_indexes": ["1"],
            },
            "records": [{"strict_status": "passed"}],
        }

    def test_accepts_pass_ready_verdict_with_accepted_swe_reports(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = self.write_verdict(
                root,
                {
                    **self.ready_payload(),
                },
            )
            self.write_accepted_assessments(root, "test-matrix")

            result = self.run_verifier(path)

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("release evidence ready", result.stdout)

    def test_rejects_missing_swe_reports(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_verdict(Path(tmp), self.ready_payload())

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ux assessment file not found", result.stderr)
        self.assertIn("artifact_quality assessment file not found", result.stderr)

    def test_rejects_swe_report_with_wrong_decision(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            matrix_id = "test-matrix"
            path = self.write_verdict(root, self.ready_payload())
            (root / f"swe_ux_assessment_{matrix_id}.md").write_text(
                f"- matrix_id: {matrix_id}\n- decision: rejected\n",
                encoding="utf-8",
            )
            (root / f"swe_artifact_quality_assessment_{matrix_id}.md").write_text(
                "- matrix_id: other\n- decision: accepted\n",
                encoding="utf-8",
            )

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ux assessment decision/status must be accepted", result.stderr)
        self.assertIn("artifact_quality assessment matrix_id must be", result.stderr)

    def test_rejects_payload_matrix_id_that_does_not_match_verdict_filename(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = root / "release_verdict_filename-matrix.json"
            path.write_text(json.dumps(self.ready_payload()), encoding="utf-8")
            self.write_accepted_assessments(root, "test-matrix")

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("matrix_id must match verdict filename", result.stderr)

    def test_rejects_fail_verdict(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_verdict(
                Path(tmp),
                {
                    "verdict": "FAIL",
                    "release_state": "RELEASE BLOCKED",
                    "release_contract": {
                        "mode": "release",
                        "contract_status": "failed",
                        "selected_providers": ["qwen-code", "claude-code", "codex-code"],
                        "selected_run_indexes": ["1"],
                    },
                    "records": [{"strict_status": "failed"}],
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

    def test_rejects_non_release_matrix_result_shape(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_verdict(
                Path(tmp),
                {
                    "result": "PASS",
                    "mode": "non-release",
                    "records": [{"strict_status": "passed"}],
                },
            )

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("verdict must be PASS", result.stderr)
        self.assertIn("release_contract must be an object", result.stderr)

    def test_rejects_release_verdict_without_release_contract_mode(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            payload = self.ready_payload()
            payload["release_contract"] = {
                "contract_status": "passed",
                "selected_providers": ["qwen-code", "claude-code", "codex-code"],
                "selected_run_indexes": ["1"],
            }
            path = self.write_verdict(Path(tmp), payload)

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release_contract.mode must be release", result.stderr)

    def test_rejects_provider_subset_or_failed_record(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            payload = self.ready_payload()
            payload["release_contract"] = {
                "mode": "release",
                "contract_status": "passed",
                "selected_providers": ["qwen-code"],
                "selected_run_indexes": ["1"],
            }
            payload["records"] = [{"strict_status": "failed"}]
            path = self.write_verdict(Path(tmp), payload)

            result = self.run_verifier(path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("selected_providers must be", result.stderr)
        self.assertIn("records[0].strict_status must be passed", result.stderr)


if __name__ == "__main__":
    unittest.main()
