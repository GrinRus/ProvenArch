import json
import os
import shlex
import stat
import subprocess
import tempfile
import unittest
from pathlib import Path

import sys

REPO_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO_ROOT / "scripts"))

from yaml_compat import load_yaml_file


EXPECTED_GOLDEN_TESTS = [
    "TestScenarioFixturesHaveTrackedGoldenSnapshots",
    "TestPersistPromotedArchitectureSnapshotCopiesOnlyArchitectureRoots",
    "TestRunPersistsRevisionImpactAndNoOpExecutionArtifacts",
    "TestRefreshSelectivelyReplaysUnaffectedBaselineShards",
    "TestRunProgressUsesOnlyDeterministicPipelineSteps",
    "TestWriteRefreshMaterializationRecordsPreservedAndRemoved",
]


class GoldenWorkflowContractTest(unittest.TestCase):
    def test_workflow_uses_fail_closed_selection_runner(self) -> None:
        workflow = load_yaml_file(REPO_ROOT / ".github" / "workflows" / "golden.yml")
        steps = workflow["jobs"]["golden"]["steps"]
        run_steps = [step.get("run", "") for step in steps]
        runner_steps = [run for run in run_steps if "scripts/run-golden-tests.sh" in run]

        self.assertEqual(1, len(runner_steps))
        for test_name in EXPECTED_GOLDEN_TESTS:
            self.assertIn(test_name, runner_steps[0])
        self.assertNotIn("TestScenarioFixturesDeterministicInitPipeline", runner_steps[0])

    def _fake_go(self, directory: Path, *, list_tests: list[str], json_events: list[dict]) -> Path:
        fake = directory / "fake-go.sh"
        list_args = " ".join(shlex.quote(line) for line in list_tests)
        json_args = " ".join(
            shlex.quote(json.dumps(event, separators=(",", ":"))) for event in json_events
        )
        script = f"""#!/usr/bin/env bash
set -Eeuo pipefail
mode=run
for arg in "$@"; do
  if [[ "$arg" == "-list" ]]; then mode=list; fi
  if [[ "$arg" == "-json" ]]; then mode=json; fi
done
if [[ "$mode" == list ]]; then
  printf '%s\\n' {list_args}
else
  printf '%s\\n' {json_args}
fi
"""
        fake.write_text(script, encoding="utf-8")
        fake.chmod(fake.stat().st_mode | stat.S_IXUSR)
        return fake

    def _run_runner(self, fake_go: Path, *tests: str) -> subprocess.CompletedProcess[str]:
        env = {
            **os.environ,
            "ACP_GOLDEN_GO_BIN": str(fake_go),
            "PROVENARCH_ROOT": str(REPO_ROOT),
        }
        return subprocess.run(
            ["bash", "scripts/run-golden-tests.sh", "./internal/orchestrator", *tests],
            cwd=REPO_ROOT,
            env=env,
            capture_output=True,
            text=True,
        )

    def test_runner_accepts_only_listed_and_passed_tests(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            fake = self._fake_go(
                Path(tmp),
                list_tests=["TestGoldenA", "TestGoldenB"],
                json_events=[
                    {"Action": "run", "Test": "TestGoldenA"},
                    {"Action": "pass", "Test": "TestGoldenA"},
                    {"Action": "run", "Test": "TestGoldenB"},
                    {"Action": "pass", "Test": "TestGoldenB"},
                ],
            )
            result = self._run_runner(fake, "TestGoldenA", "TestGoldenB")

        self.assertEqual(0, result.returncode, result.stdout + result.stderr)

    def test_runner_fails_when_test_is_renamed_or_removed(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            fake = self._fake_go(
                Path(tmp),
                list_tests=["TestGoldenA"],
                json_events=[
                    {"Action": "run", "Test": "TestGoldenA"},
                    {"Action": "pass", "Test": "TestGoldenA"},
                ],
            )
            result = self._run_runner(fake, "TestGoldenA", "TestGoldenRemoved")

        self.assertNotEqual(0, result.returncode)
        self.assertIn("selection is stale", result.stderr)

    def test_runner_fails_when_go_reports_zero_tests(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            fake = self._fake_go(
                Path(tmp),
                list_tests=["TestGoldenA"],
                json_events=[{"Action": "pass", "Package": "example.test"}],
            )
            result = self._run_runner(fake, "TestGoldenA")

        self.assertNotEqual(0, result.returncode)
        self.assertIn("did not pass", result.stderr)


if __name__ == "__main__":
    unittest.main()
