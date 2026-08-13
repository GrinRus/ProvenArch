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

    def run_verifier(self, *paths: Path) -> subprocess.CompletedProcess[str]:
        return self.run_verifier_args(*(str(path) for path in paths))

    def run_verifier_args(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(self.script), *args],
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
            "records": [{
                "strict_status": "passed",
                "public_authority": {
                    "effective_verdict_source": "orchestrator",
                    "promotion_audit_result": "pass",
                },
            }],
        }

    def write_ready_evidence(self, root: Path, matrix_id: str) -> Path:
        payload = {**self.ready_payload(), "matrix_id": matrix_id}
        path = self.write_verdict(root, payload)
        self.write_accepted_assessments(root, matrix_id)
        return path

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

    def test_accepts_composite_release_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            paths = [
                self.write_ready_evidence(root, "release-full-fast"),
                self.write_ready_evidence(root, "release-full-long"),
                self.write_ready_evidence(root, "release-full-ftgo-sentry"),
            ]

            result = self.run_verifier(*paths)

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("composite release evidence ready: 3 constituent matrices", result.stdout)

    def test_accepts_composite_matrix_ids_configuration(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            reports = root / "reports"
            reports.mkdir()
            matrix_ids = ["release-full-fast", "release-full-long", "release-full-ftgo-sentry"]
            for matrix_id in matrix_ids:
                self.write_ready_evidence(reports, matrix_id)

            result = subprocess.run(
                [sys.executable, str(self.script), "--matrix-ids", ",".join(matrix_ids)],
                cwd=root,
                capture_output=True,
                text=True,
            )

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("composite release evidence ready: 3 constituent matrices", result.stdout)

    def test_rejects_missing_constituent_from_matrix_ids_configuration(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            result = subprocess.run(
                [sys.executable, str(self.script), "--matrix-ids", "release-full-missing"],
                cwd=tmp,
                capture_output=True,
                text=True,
            )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release verdict file not found", result.stderr)

    def test_rejects_duplicate_matrix_id_in_composite_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = self.write_ready_evidence(root, "release-full-fast")

            result = self.run_verifier(path, path)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate matrix_id in composite release evidence", result.stderr)

    def test_rejects_duplicate_matrix_ids_configuration(self) -> None:
        result = self.run_verifier_args(
            "--matrix-ids",
            "release-full-fast,release-full-fast",
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate matrix id in ACP_RELEASE_MATRIX_IDS", result.stderr)

    def test_rejects_conflicting_release_evidence_modes(self) -> None:
        result = self.run_verifier_args(
            "--matrix-id",
            "release-fast",
            "--verdict-path",
            "reports/release_verdict_release-fast.json",
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("set exactly one release evidence mode", result.stderr)

    def test_rejects_empty_matrix_id_configuration(self) -> None:
        result = self.run_verifier_args("--matrix-ids", "release-full-fast,")

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("contains an empty matrix id", result.stderr)

    def test_rejects_composite_when_one_constituent_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            ready = self.write_ready_evidence(root, "release-full-fast")
            failed_payload = {
                **self.ready_payload(),
                "matrix_id": "release-full-long",
                "verdict": "FAIL",
                "release_state": "RELEASE BLOCKED",
            }
            failed = self.write_verdict(root, failed_payload)
            self.write_accepted_assessments(root, "release-full-long")

            result = self.run_verifier(ready, failed)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release_verdict_release-full-long.json", result.stderr)
        self.assertIn("verdict must be PASS", result.stderr)

    def test_rejects_composite_when_one_assessment_is_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            ready = self.write_ready_evidence(root, "release-full-fast")
            missing = self.write_verdict(
                root,
                {**self.ready_payload(), "matrix_id": "release-full-long"},
            )

            result = self.run_verifier(ready, missing)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release_verdict_release-full-long.json", result.stderr)
        self.assertIn("ux assessment file not found", result.stderr)

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
