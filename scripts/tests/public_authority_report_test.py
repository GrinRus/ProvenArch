import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
REPORT_PATH = REPO_ROOT / "scripts" / "e2e_batch_report.py"


def load_report_module():
    spec = importlib.util.spec_from_file_location("e2e_batch_report_public_authority", REPORT_PATH)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class PublicAuthorityReportTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.module = load_report_module()

    def _run(self):
        return self.module.RunEvaluation(
            provider="qwen-code",
            run_index=1,
            run_dir=Path("/tmp/public-authority-run"),
            hard_pass=True,
            reliability=40,
            contract=30,
            analysis=30,
            total=100,
            verdict="Excellent",
            runtime_contract_status="passed",
            effective_verdict_source="orchestrator",
            promotion_audit_result="pass",
            provider_invocations=2,
            provider_invocation_budget_max=3,
            provider_invocation_remaining=1,
            validation_first_pass_valid=4,
            validation_first_pass_invalid=1,
        )

    def test_extracts_public_authority_fields_and_preserves_legacy_missing_fields(self) -> None:
        evidence = self.module.extract_public_authority_evidence(
            {
                "totals": {
                    "effective_verdict_source": "orchestrator",
                    "promotion_audit_result": "pass",
                    "provider_invocations": 2,
                    "provider_invocation_budget_max": 3,
                    "provider_invocation_remaining": 1,
                    "provider_budget_exhausted": False,
                    "provider_terminal_exhaustion_reason": "",
                    "validation_first_pass_valid": 4,
                    "validation_first_pass_invalid": 1,
                }
            }
        )
        self.assertEqual("orchestrator", evidence["effective_verdict_source"])
        self.assertEqual("pass", evidence["promotion_audit_result"])
        self.assertEqual(2, evidence["provider_invocations"])
        self.assertEqual(4, evidence["validation_first_pass_valid"])

        legacy = self.module.extract_public_authority_evidence({"totals": {}})
        self.assertEqual("", legacy["effective_verdict_source"])
        self.assertFalse(legacy["provider_budget_exhausted"])

    def test_release_authority_gate_requires_explicit_public_evidence(self) -> None:
        gate, reasons = self.module.public_authority_gate("", "", False, release_mode=True)
        self.assertTrue(gate)
        self.assertIn("execution:effective-verdict-missing", reasons)
        self.assertIn("execution:promotion-audit-missing-or-failed", reasons)

        gate, reasons = self.module.public_authority_gate("orchestrator", "pass", False, release_mode=True)
        self.assertFalse(gate)
        self.assertEqual([], reasons)

        diagnostic_gate, diagnostic_reasons = self.module.public_authority_gate("orchestrator", "warn", False)
        self.assertFalse(diagnostic_gate)
        self.assertEqual([], diagnostic_reasons)

    def test_report_and_tsv_publish_public_authority_without_internal_imports(self) -> None:
        run = self._run()
        with tempfile.TemporaryDirectory() as tempdir:
            root = Path(tempdir)
            report_path = root / "execution.md"
            tsv_path = root / "meta.tsv"
            self.module.write_execution_report(
                report_path,
                "authority-fixture",
                [run],
                [],
                {"generated_at_utc": "2026-08-12T00:00:00Z"},
                ["qwen-code"],
            )
            self.module.write_meta_tsv(tsv_path, [run])
            report = report_path.read_text(encoding="utf-8")
            tsv = tsv_path.read_text(encoding="utf-8")

        self.assertIn("## Public Promotion Authority", report)
        self.assertIn("promotion_audit_failed_runs: 0/1", report)
        self.assertIn("effective_verdict_source", tsv.splitlines()[0])
        self.assertIn("orchestrator", tsv.splitlines()[1])
        source = REPORT_PATH.read_text(encoding="utf-8")
        self.assertNotIn("internal.artifact", source)
        self.assertNotIn("internal.orchestrator", source)


if __name__ == "__main__":
    unittest.main()
