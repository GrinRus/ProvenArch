import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
FULL_RUN_AI_ADVENT = REPO_ROOT / "scripts" / "full-run-ai-advent.sh"


class FullRunAiAdventLayoutTest(unittest.TestCase):
    def test_headless_and_baseline_workspaces_use_separate_temp_roots(self) -> None:
        script_text = FULL_RUN_AI_ADVENT.read_text(encoding="utf-8")

        self.assertIn('HEADLESS_TMP_ROOT="$TMP_ROOT/headless"', script_text)
        self.assertIn('BASELINE_TMP_ROOT="$TMP_ROOT/baseline"', script_text)
        self.assertIn('WORKSPACE_HEADLESS="$HEADLESS_TMP_ROOT/arch-workspace"', script_text)
        self.assertIn('WORKSPACE_BASELINE="$BASELINE_TMP_ROOT/arch-workspace"', script_text)
        self.assertNotIn('WORKSPACE_BASELINE="$TMP_ROOT/arch-workspace-baseline"', script_text)

    def test_run_results_rows_have_fixed_field_count_and_structured_accounting(self) -> None:
        script_text = FULL_RUN_AI_ADVENT.read_text(encoding="utf-8")

        self.assertIn('RUN_RESULTS_EXPECTED_FIELDS=17', script_text)
        self.assertIn('NF == expected_fields', script_text)
        self.assertIn('malformed_run_results_rows', script_text)
        self.assertIn("printf '%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\n'", script_text)


if __name__ == "__main__":
    unittest.main()
