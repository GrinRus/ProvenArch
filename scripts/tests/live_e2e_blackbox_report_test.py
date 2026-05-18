import unittest
from pathlib import Path


class LiveE2EBlackBoxReportTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]

    def test_batch_harness_uses_internal_backend_cycle_and_blackbox_report_fields(self) -> None:
        batch_script = (self.repo_root / "scripts" / "full-run-batch.sh").read_text(encoding="utf-8")
        self.assertIn("./scripts/internal/live-e2e-backend-cycle.sh", batch_script)
        self.assertNotIn("./scripts/full-run-ai-advent.sh", batch_script)
        for token in (
            "blackbox_e2e_steps_${BATCH_ID}.jsonl",
            "blackbox_e2e_steps_${BATCH_ID}.md",
            '"step_id"',
            '"goal"',
            '"action"',
            '"observed_evidence"',
            '"status"',
            '"primary_classification"',
            '"evidence_paths"',
            '"next_decision"',
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

    def test_matrix_harness_writes_blackbox_report_fields_without_public_wrapper(self) -> None:
        matrix_script = (self.repo_root / "scripts" / "full-run-batch-matrix.sh").read_text(encoding="utf-8")
        self.assertIn('BATCH_SCRIPT="${BATCH_SCRIPT:-$PROVENARCH_ROOT/scripts/full-run-batch.sh}"', matrix_script)
        self.assertNotIn("full-run-ai-advent", matrix_script)
        for token in (
            "blackbox_e2e_steps_${MATRIX_ID}.jsonl",
            "blackbox_e2e_steps_${MATRIX_ID}.md",
            '"step_id"',
            '"goal"',
            '"action"',
            '"observed_evidence"',
            '"status"',
            '"primary_classification"',
            '"evidence_paths"',
            '"next_decision"',
            "matrix.preflight",
            "matrix.plan",
            "matrix.verdict",
            "matrix_plan_failed",
            "release_guard_blocked",
        ):
            self.assertIn(token, matrix_script)

    def test_removed_legacy_live_e2e_public_surfaces_are_absent(self) -> None:
        for rel in (
            "scripts/full-run-ai-advent.sh",
            "docs/LOCAL_FULL_RUN_AI_ADVENT.md",
            "examples/e2e-matrix.regression-wave1.yaml",
        ):
            self.assertFalse((self.repo_root / rel).exists(), f"unexpected legacy live E2E surface: {rel}")


if __name__ == "__main__":
    unittest.main()
