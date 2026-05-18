import unittest
from pathlib import Path


class LiveE2EBlackBoxReportTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]

    def test_batch_harness_uses_internal_backend_cycle_and_blackbox_report_fields(self) -> None:
        batch_script = (self.repo_root / "scripts" / "full-run-batch.sh").read_text(encoding="utf-8")
        self.assertIn("./scripts/internal/live-e2e-backend-cycle.sh", batch_script)
        self.assertIn('source "$PROVENARCH_ROOT/scripts/internal/live-e2e-evaluator.sh"', batch_script)
        self.assertIn("live_e2e_evaluator_write_batch_step", batch_script)
        self.assertNotIn("./scripts/full-run-ai-advent.sh", batch_script)
        for token in (
            "blackbox_e2e_steps_${BATCH_ID}.jsonl",
            "blackbox_e2e_steps_${BATCH_ID}.md",
            "batch.preflight",
            "batch.report",
            "batch.final",
            "report_generation_failed",
            "require_provider_cmd",
            "target repos metadata preflight failed",
            'print("failed\\tmissing_result_json")',
            'status not in {"passed", "failed", "skipped", "blocked"}',
        ):
            self.assertIn(token, batch_script)

    def test_internal_evaluator_helper_owns_step_report_shape(self) -> None:
        helper_path = self.repo_root / "scripts" / "internal" / "live-e2e-evaluator.sh"
        helper = helper_path.read_text(encoding="utf-8")
        self.assertEqual(0, helper_path.stat().st_mode & 0o111, "internal evaluator helper must not be executable")
        self.assertIn("source-only helper", helper)
        self.assertNotIn("./scripts/full-run-batch-matrix.sh", helper)
        self.assertNotIn("exec scripts/full-run-batch-matrix.sh", helper)
        for token in (
            "live_e2e_evaluator_init_batch_report",
            "live_e2e_evaluator_write_batch_step",
            "live_e2e_evaluator_init_matrix_report",
            "live_e2e_evaluator_write_matrix_step",
            '"step_id"',
            '"goal"',
            '"action"',
            '"observed_evidence"',
            '"status"',
            '"primary_classification"',
            '"evidence_paths"',
            '"next_decision"',
        ):
            self.assertIn(token, helper)

    def test_matrix_harness_writes_blackbox_report_fields_without_public_wrapper(self) -> None:
        matrix_script = (self.repo_root / "scripts" / "full-run-batch-matrix.sh").read_text(encoding="utf-8")
        self.assertIn('BATCH_SCRIPT="${BATCH_SCRIPT:-$PROVENARCH_ROOT/scripts/full-run-batch.sh}"', matrix_script)
        self.assertIn('source "$PROVENARCH_ROOT/scripts/internal/live-e2e-evaluator.sh"', matrix_script)
        self.assertIn("live_e2e_evaluator_write_matrix_step", matrix_script)
        self.assertNotIn("full-run-ai-advent", matrix_script)
        for token in (
            "blackbox_e2e_steps_${MATRIX_ID}.jsonl",
            "blackbox_e2e_steps_${MATRIX_ID}.md",
            "matrix.preflight",
            "matrix.plan",
            "matrix.verdict",
            "matrix_plan_failed",
            "release_guard_blocked",
        ):
            self.assertIn(token, matrix_script)

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
