import unittest
from pathlib import Path


class LiveE2EBlackBoxReportTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]

    def test_batch_harness_keeps_machine_evidence_without_operator_decision_reports(self) -> None:
        batch_script = (self.repo_root / "scripts" / "full-run-batch.sh").read_text(encoding="utf-8")
        self.assertIn("./scripts/internal/live-e2e-backend-cycle.sh", batch_script)
        self.assertNotIn("./scripts/full-run-ai-advent.sh", batch_script)
        for token in ("blackbox_e2e_steps", "live_e2e_evaluator", "batch.preflight", "batch.report", "batch.final"):
            self.assertNotIn(token, batch_script)
        for token in ("require_provider_cmd", "target repos metadata preflight failed", 'print("failed\\tmissing_result_json")'):
            self.assertIn(token, batch_script)

    def test_internal_evaluator_helper_is_deleted(self) -> None:
        helper_path = self.repo_root / "scripts" / "internal" / "live-e2e-evaluator.sh"
        self.assertFalse(helper_path.exists())

    def test_backend_cycle_fail_fasts_headless_artifact_quality_blockers(self) -> None:
        helper = (self.repo_root / "scripts" / "internal" / "live-e2e-backend-cycle.sh").read_text(encoding="utf-8")
        self.assertIn("artifact_quality_count", helper)
        self.assertIn("startswith('artifact_quality:')", helper)
        self.assertIn("headless run $run_id produced artifact_quality blockers", helper)
        self.assertIn('FAILURE_REASON="quality"', helper)

    def test_matrix_harness_has_no_script_authored_operator_decisions(self) -> None:
        matrix_script = (self.repo_root / "scripts" / "full-run-batch-matrix.sh").read_text(encoding="utf-8")
        self.assertIn('BATCH_SCRIPT="${BATCH_SCRIPT:-$PROVENARCH_ROOT/scripts/full-run-batch.sh}"', matrix_script)
        self.assertNotIn("full-run-ai-advent", matrix_script)
        for token in ("blackbox_e2e_steps", "live_e2e_evaluator", "matrix.preflight", "matrix.plan", "matrix.verdict"):
            self.assertNotIn(token, matrix_script)
        self.assertIn("matrix_result_${MATRIX_ID}.json", matrix_script)
        self.assertIn("release_verdict_${MATRIX_ID}.json", matrix_script)

    def test_unwanted_live_e2e_public_surfaces_are_absent(self) -> None:
        for rel in (
            "scripts/full-run-ai-advent.sh",
            "docs/LOCAL_FULL_RUN_AI_ADVENT.md",
            "examples/e2e-matrix.regression-wave1.yaml",
            ".github/workflows/manual-live-e2e.yml",
            ".github/workflows/manual-live-e2e.yaml",
        ):
            self.assertFalse((self.repo_root / rel).exists(), f"unexpected live E2E public surface: {rel}")


if __name__ == "__main__":
    unittest.main()
