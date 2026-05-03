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
        self.assertIn('run_result_row_exists()', script_text)
        self.assertIn('append_run_result_row_once()', script_text)
        self.assertIn('NF == expected_fields', script_text)
        self.assertIn('malformed_run_results_rows', script_text)
        self.assertIn("printf '%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\t%s\\n'", script_text)

    def test_failed_pipeline_paths_persist_run_results_metadata_before_die(self) -> None:
        script_text = FULL_RUN_AI_ADVENT.read_text(encoding="utf-8")

        self.assertIn('snapshot_run_artifacts "$run_id" "$runtime_label" "$pipeline" "$iteration" "$workspace_path" || true', script_text)
        self.assertIn('if metrics="$(quality_metrics "$quality_path" 2>/dev/null)"; then', script_text)
        self.assertIn('append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "failed" "$workspace_path" "$output_path"', script_text)
        self.assertIn('append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "${status:-failed}" "$workspace_path" "$output_path"', script_text)
        self.assertIn('append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "$status" "$workspace_path" "$output_path"', script_text)
        self.assertLess(
            script_text.index('append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "${status:-failed}" "$workspace_path" "$output_path"'),
            script_text.index('die "pipeline command failed for runtime=$runtime_label pipeline=$pipeline"'),
        )
        self.assertLess(
            script_text.index('append_run_result_row_once "$iteration" "$runtime_mode" "$runtime_provider" "$pipeline" "$run_id" "failed" "$workspace_path" "$output_path"'),
            script_text.index('die "missing quality summary for run $run_id at $quality_path"'),
        )


if __name__ == "__main__":
    unittest.main()
