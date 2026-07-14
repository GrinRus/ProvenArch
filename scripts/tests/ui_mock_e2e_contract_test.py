import json
import os
import subprocess
import sys
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO_ROOT / "scripts"))

from yaml_compat import load_yaml_file


class UIMockE2EContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = REPO_ROOT

    def test_runner_lists_exactly_seven_existing_mock_scenarios(self) -> None:
        result = subprocess.run(
            ["bash", "scripts/ui-mock-e2e.sh"],
            cwd=self.repo_root,
            env={**os.environ, "ACP_UI_MOCK_E2E_LIST": "1"},
            capture_output=True,
            text=True,
            check=True,
        )

        scenarios = [line.strip() for line in result.stdout.splitlines() if line.strip()]

        self.assertEqual(
            [
                "analysis-failed-shard-mock",
                "onboarding-recovery-mock",
                "permission-recovery-mock",
                "provider-stream-mock",
                "publish-git-recovery-mock",
                "qa-recovery-mock",
                "source-recovery-mock",
            ],
            scenarios,
        )
        for scenario in scenarios:
            self.assertTrue((self.repo_root / "ui" / "e2e" / f"{scenario}.spec.ts").exists(), scenario)

    def test_ui_package_exposes_canonical_mock_e2e_script(self) -> None:
        package_json = json.loads((self.repo_root / "ui" / "package.json").read_text(encoding="utf-8"))

        self.assertEqual("bash ../scripts/ui-mock-e2e.sh", package_json["scripts"].get("e2e:mock"))

    def test_ui_workflow_runs_mock_playwright_gate(self) -> None:
        workflow = load_yaml_file(self.repo_root / ".github" / "workflows" / "ui.yml")
        runs = [step.get("run", "") for step in workflow["jobs"]["ui"]["steps"]]

        self.assertIn("./ui/node_modules/.bin/playwright install --with-deps chromium", runs)
        self.assertIn("npm run e2e:mock --prefix ui", runs)


if __name__ == "__main__":
    unittest.main()
