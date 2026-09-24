import json
import sys
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO_ROOT / "scripts"))

from yaml_compat import load_yaml_file


PR_ONLY_WORKFLOWS = {"contracts", "ui", "golden", "smoke-api", "smoke-cli"}
MAIN_PUSH_WORKFLOWS = {"backend", "lint", "codeql"}


def _workflow_triggers(name: str) -> dict:
    workflow = load_yaml_file(REPO_ROOT / ".github" / "workflows" / f"{name}.yml")
    # PyYAML's YAML 1.1 loader may parse the unquoted key `on` as boolean True.
    return workflow.get("on", workflow.get(True, {}))


class CITriggerPolicyTest(unittest.TestCase):
    def test_all_required_contexts_are_pull_request_checks(self) -> None:
        governance = json.loads(
            (REPO_ROOT / "docs" / "release-governance.json").read_text(encoding="utf-8")
        )
        contexts = governance["required_status_checks"]["contexts"]

        self.assertCountEqual(
            {"backend", "contracts", "golden", "smoke-api", "smoke-cli", "ui"},
            contexts,
        )
        for name in contexts:
            with self.subTest(workflow=name):
                self.assertIn("pull_request", _workflow_triggers(name))

    def test_short_required_checks_are_pr_only(self) -> None:
        for name in PR_ONLY_WORKFLOWS:
            with self.subTest(workflow=name):
                triggers = _workflow_triggers(name)
                self.assertIn("pull_request", triggers)
                self.assertNotIn("push", triggers)

    def test_backend_lint_and_codeql_keep_main_pushes(self) -> None:
        for name in MAIN_PUSH_WORKFLOWS:
            with self.subTest(workflow=name):
                triggers = _workflow_triggers(name)
                self.assertIn("pull_request", triggers)
                self.assertEqual(["main"], triggers["push"]["branches"])

    def test_scorecard_keeps_weekly_and_manual_runs_without_push(self) -> None:
        triggers = _workflow_triggers("scorecard")

        self.assertNotIn("push", triggers)
        self.assertIn("schedule", triggers)
        self.assertEqual(1, len(triggers["schedule"]))
        self.assertIn("workflow_dispatch", triggers)


if __name__ == "__main__":
    unittest.main()
