import json
import subprocess
import sys
import unittest
from pathlib import Path


class LiveE2EPlanTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.script = cls.repo_root / "scripts" / "live-e2e-plan.py"

    def run_plan(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(self.script), *args],
            cwd=self.repo_root,
            capture_output=True,
            text=True,
        )

    def load_json_plan(self, *args: str) -> dict[str, object]:
        result = self.run_plan(*args, "--format", "json")
        self.assertEqual(result.returncode, 0, msg=result.stderr)
        return json.loads(result.stdout)

    def test_smoke_tiny_generates_one_repo_one_run_one_provider_command(self) -> None:
        payload = self.load_json_plan("--mode", "smoke", "--size", "tiny", "--providers", "codex")

        self.assertEqual(payload["expected_backend_runs"], 1)
        self.assertEqual(payload["selected_providers"], ["codex-code"])
        self.assertEqual(payload["selected_run_indexes"], ["1"])
        self.assertEqual(payload["target_repo_sets"], ["bank-of-anthos"])
        self.assertTrue(payload["quality"]["required"])

        commands = payload["commands"]
        self.assertEqual(len(commands), 1)
        env = commands[0]["env"]
        self.assertEqual(env["E2E_MATRIX_FILE"], "examples/e2e-matrix.smoke-tiny.bank.yaml")
        self.assertEqual(env["E2E_MATRIX_RELEASE_MODE"], "0")
        self.assertEqual(env["RUN_COUNT"], "1")
        self.assertEqual(env["BATCH_RUN_SELECTION"], "1")
        self.assertEqual(env["BATCH_PROVIDER_FILTER"], "codex-code")
        self.assertEqual(env["BATCH_FRONTEND_MODE"], "never")
        self.assertEqual(env["BATCH_FRONTEND_CANCEL_MODE"], "never")

    def test_regres_fast_accepts_provider_shorthand_csv(self) -> None:
        payload = self.load_json_plan("--mode", "regres", "--size", "fast", "--providers", "qwen,codex")

        self.assertEqual(payload["selected_providers"], ["qwen-code", "codex-code"])
        self.assertEqual(payload["expected_backend_runs"], 6)
        self.assertEqual(len(payload["commands"]), 2)
        for command in payload["commands"]:
            env = command["env"]
            self.assertEqual(env["BATCH_PROVIDER_FILTER"], "qwen-code,codex-code")
            self.assertEqual(env["E2E_MATRIX_RELEASE_MODE"], "0")

    def test_regres_full_is_non_release_and_includes_all_repo_sets(self) -> None:
        payload = self.load_json_plan(
            "--mode",
            "regres",
            "--size",
            "full",
            "--providers",
            "claude",
            "--frontend-mode",
            "never",
        )

        self.assertEqual(payload["selected_providers"], ["claude-code"])
        self.assertEqual(payload["expected_backend_runs"], 6)
        self.assertEqual(
            payload["target_repo_sets"],
            [
                "bank-of-anthos",
                "openedx-ecosystem",
                "openstack-ecosystem",
                "posthog",
                "ftgo-application",
                "sentry-ecosystem",
            ],
        )
        matrix_files = [command["matrix_file"] for command in payload["commands"]]
        self.assertIn("examples/e2e-matrix.diagnostic.sentry.yaml", matrix_files)
        for command in payload["commands"]:
            env = command["env"]
            self.assertEqual(env["E2E_MATRIX_RELEASE_MODE"], "0")
            self.assertEqual(env["BATCH_PROVIDER_FILTER"], "claude-code")
            self.assertEqual(env["BATCH_FRONTEND_MODE"], "never")
            self.assertEqual(env["BATCH_FRONTEND_CANCEL_MODE"], "never")

    def test_release_selector_rejects_provider_subset(self) -> None:
        result = self.run_plan(
            "--mode",
            "release",
            "--size",
            "fast",
            "--providers",
            "codex",
            "--format",
            "json",
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release selectors require all providers", result.stderr)

    def test_release_selector_accepts_all_providers_in_any_order(self) -> None:
        payload = self.load_json_plan(
            "--mode",
            "release",
            "--size",
            "fast",
            "--providers",
            "codex,qwen,claude",
        )

        self.assertEqual(payload["selected_providers"], ["qwen-code", "claude-code", "codex-code"])
        self.assertEqual(payload["expected_backend_runs"], 12)
        for command in payload["commands"]:
            self.assertNotIn("BATCH_PROVIDER_FILTER", command["env"])

    def test_release_full_preserves_three_provider_direct_matrix_surface(self) -> None:
        payload = self.load_json_plan("--mode", "release", "--size", "full")

        self.assertEqual(payload["selected_providers"], ["qwen-code", "claude-code", "codex-code"])
        self.assertEqual(payload["expected_backend_runs"], 36)
        self.assertEqual(len(payload["commands"]), 3)
        for command in payload["commands"]:
            env = command["env"]
            self.assertEqual(env["E2E_MATRIX_RELEASE_MODE"], "1")
            self.assertNotIn("BATCH_PROVIDER_FILTER", env)
            self.assertTrue(command["release_mode"])

    def test_regres_and_release_plans_declare_existing_quality_gate(self) -> None:
        for args in (
            ("--mode", "regres", "--size", "fast", "--providers", "codex"),
            ("--mode", "release", "--size", "full"),
        ):
            payload = self.load_json_plan(*args)
            quality = payload["quality"]
            self.assertTrue(quality["required"])
            self.assertEqual("reports/taskruns/<run_id>-quality.json", quality["run_quality_json"])
            self.assertEqual("quality_report_<batch-id>.md", quality["batch_quality_report"])
            self.assertEqual("quality_gates_failed", quality["blocking_failure_class"])
            self.assertEqual("artifact_quality:", quality["artifact_quality_warning_prefix"])

    def test_shell_output_contains_direct_harness_commands_only(self) -> None:
        result = self.run_plan("--mode", "regres", "--size", "fast", "--providers", "codex")

        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("./scripts/full-run-batch-matrix.sh", result.stdout)
        self.assertIn("BATCH_PROVIDER_FILTER=codex-code", result.stdout)
        self.assertNotIn("live-e2e-plan.py", result.stdout)


if __name__ == "__main__":
    unittest.main()
