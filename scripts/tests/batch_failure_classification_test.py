import importlib.util
import json
import os
import shlex
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
E2E_BATCH_REPORT_PATH = REPO_ROOT / "scripts" / "e2e_batch_report.py"
FULL_RUN_BATCH_SCRIPT = REPO_ROOT / "scripts" / "full-run-batch-5x2.sh"


def load_e2e_batch_report_module():
    spec = importlib.util.spec_from_file_location("e2e_batch_report", E2E_BATCH_REPORT_PATH)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def write_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


class BatchFailureClassificationTest(unittest.TestCase):
    def setUp(self) -> None:
        self.module = load_e2e_batch_report_module()
        self.tempdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tempdir.name)
        self.run_dir = self.root / "run1"
        self._create_fixture_run_dir(self.run_dir)

    def tearDown(self) -> None:
        self.tempdir.cleanup()

    def _create_fixture_run_dir(self, run_dir: Path) -> None:
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- quality_gates: passed",
                    "- failure_reason: infra_incomplete_cycle",
                    "- expected_runs: 10",
                    "- completed_runs: 2",
                    "- expected_headless_runs: 10",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(run_dir / "full-run.log", "batch ended with incomplete cycle\n")
        write_text(run_dir / "batch-driver.log", "driver completed with process_exit=1\n")
        write_text(
            run_dir / "run-results.tsv",
            "\n".join(
                [
                    "\t".join(
                        [
                            "1",
                            "headless",
                            "qwen-code",
                            "init",
                            "init-run",
                            "succeeded",
                            "8",
                            "1",
                            "0",
                            "1",
                            "2",
                            "1",
                            "0",
                            "qwen-code@stub",
                            "reports/taskruns/init-run-quality.json",
                            "reports",
                        ]
                    ),
                    "\t".join(
                        [
                            "1",
                            "headless",
                            "qwen-code",
                            "refresh",
                            "refresh-run",
                            "succeeded",
                            "9",
                            "1",
                            "1",
                            "1",
                            "2",
                            "1",
                            "0",
                            "qwen-code@stub",
                            "reports/taskruns/refresh-run-quality.json",
                            "reports",
                        ]
                    ),
                ]
            )
            + "\n",
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","error_code":"runner_parse_failed","message":"provider returned invalid envelope"}\n',
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt",
            "API Error: 403 permission_error\nrunner_parse_failed\n",
        )
        self._write_snapshot(run_dir, "init-run", "init")
        self._write_snapshot(run_dir, "refresh-run", "refresh")

    def _write_snapshot(self, run_dir: Path, run_id: str, pipeline: str) -> None:
        reports_root = run_dir / "snapshots" / run_id / "reports"
        write_text(
            reports_root / "as-is/overview.md",
            "\n".join(
                [
                    "# As-Is Overview",
                    "",
                    "- Services: 1",
                    "- Dependencies (edges): 1",
                    "- External systems: 0",
                    "- Datastores: 0",
                    "",
                ]
            ),
        )
        write_text(
            reports_root / "findings/findings.md",
            "\n".join(
                [
                    "# Findings",
                    "",
                    "## Missing owner mapping",
                    "",
                    "- ID: `finding.owner.missing`",
                    "- Severity: `medium`",
                    "- Description: owner_team_id is unknown",
                    "",
                ]
            ),
        )
        write_text(
            reports_root / "coverage/summary.md",
            "\n".join(
                [
                    "# Coverage Summary",
                    "",
                    "## Observed",
                    "",
                    "- services",
                    "- entrypoints",
                    "",
                    "## Missing",
                    "",
                    "- owner mappings",
                    "",
                    "## Notes",
                    "",
                    "- fake deterministic snapshot",
                    "",
                ]
            ),
        )
        write_text(
            reports_root / "coverage/open-questions.md",
            "\n".join(
                [
                    "# Open Questions",
                    "",
                    "- `q.owner.service` Who owns the materialized service?",
                    "",
                ]
            ),
        )
        quality_payload = {
            "runtime_versions": ["qwen-code@stub"],
            "totals": {
                "changeset_ops": 1,
                "findings_added": 0 if pipeline == "init" else 1,
                "questions_count": 1,
                "coverage_observed": 2,
                "coverage_missing": 1,
                "warnings_count": 0,
            },
            "steps": [
                {
                    "step_id": f"{pipeline}.step1.collect" if pipeline == "init" else f"{pipeline}.step3.findings",
                    "runtime_name": "qwen-code",
                }
            ],
        }
        write_json(reports_root / f"taskruns/{run_id}-quality.json", quality_payload)
        taskrun_payload = {
            "meta": {
                "runtime": {"name": "qwen-code"},
                "step_id": f"{pipeline}.step1.collect" if pipeline == "init" else f"{pipeline}.step3.findings",
            },
            "changeset": [],
        }
        write_json(
            reports_root / f"taskruns/{run_id}-{pipeline}-step{'1-collect' if pipeline == 'init' else '3-findings'}.json",
            taskrun_payload,
        )

    def _create_incomplete_fixture_run_dir(self, run_dir: Path) -> None:
        self._create_fixture_run_dir(run_dir)
        incomplete_banner = "\n".join(
            [
                "> Analysis incomplete.",
                "> Collect status: unusable (planned=2 succeeded=0 failed=2)",
                "> Findings status: skipped (planned=0 succeeded=0 failed=0)",
                "> Reasons: collect_all_shards_failed, findings_skipped_due_to_unusable_collect",
                "",
            ]
        )
        for run_id in ("init-run", "refresh-run"):
            reports_root = run_dir / "snapshots" / run_id / "reports"
            write_text(
                reports_root / "as-is/overview.md",
                "# As-Is Overview\n\n"
                + incomplete_banner
                + "- Services: 0\n- Dependencies (edges): 0\n- External systems: 0\n- Datastores: 0\n",
            )
            write_text(
                reports_root / "findings/findings.md",
                "# Findings\n\n" + incomplete_banner + "Findings unavailable because analysis did not complete.\n",
            )
            write_text(
                reports_root / "coverage/summary.md",
                "# Coverage Summary\n\n"
                + incomplete_banner
                + "## Observed\n\nUnavailable due to incomplete analysis.\n\n"
                + "## Missing\n\nUnknown due to incomplete analysis.\n\n"
                + "## Notes\n\nAnalysis incomplete. See banner above.\n",
            )
            write_text(
                reports_root / "coverage/open-questions.md",
                "# Open Questions\n\n" + incomplete_banner + "Open questions unavailable due to incomplete analysis.\n",
            )
            quality_path = reports_root / f"taskruns/{run_id}-quality.json"
            quality_payload = json.loads(quality_path.read_text(encoding="utf-8"))
            quality_payload["evidence_state"] = {
                "collect": {
                    "status": "unusable",
                    "planned_shards": 2,
                    "succeeded_shards": 0,
                    "failed_shards": 2,
                },
                "findings": {
                    "status": "skipped",
                    "planned_shards": 0,
                    "succeeded_shards": 0,
                    "failed_shards": 0,
                },
                "report_mode": "incomplete",
                "reasons": ["collect_all_shards_failed", "findings_skipped_due_to_unusable_collect"],
            }
            write_json(quality_path, quality_payload)

    def _create_partial_incomplete_fixture_run_dir(self, run_dir: Path) -> None:
        self._create_fixture_run_dir(run_dir)
        partial_banner = "\n".join(
            [
                "> Partial analysis. Some shards failed; downstream content may be incomplete.",
                "> Collect status: partial (planned=2 succeeded=1 failed=1)",
                "> Findings status: partial (planned=2 succeeded=1 failed=1)",
                "> Reasons: collect_partial_shard_failures, findings_partial_shard_failures",
                "",
            ]
        )
        for run_id in ("init-run", "refresh-run"):
            reports_root = run_dir / "snapshots" / run_id / "reports"
            write_text(
                reports_root / "as-is/overview.md",
                "# As-Is Overview\n\n"
                + partial_banner
                + "- Services: 1\n- Dependencies (edges): 1\n- External systems: 0\n- Datastores: 0\n",
            )
            write_text(
                reports_root / "findings/findings.md",
                "# Findings\n\n"
                + partial_banner
                + "## Missing owner mapping\n\n- ID: `finding.owner.missing`\n- Severity: `medium`\n- Description: owner_team_id is unknown\n",
            )
            write_text(
                reports_root / "coverage/summary.md",
                "# Coverage Summary\n\n"
                + partial_banner
                + "## Observed\n\n- services\n- entrypoints\n\n"
                + "## Missing\n\n- owner mappings\n\n"
                + "## Notes\n\n- partial shard coverage only\n",
            )
            write_text(
                reports_root / "coverage/open-questions.md",
                "# Open Questions\n\n"
                + partial_banner
                + "- `q.owner.service` Who owns the materialized service?\n",
            )
            quality_path = reports_root / f"taskruns/{run_id}-quality.json"
            quality_payload = json.loads(quality_path.read_text(encoding="utf-8"))
            quality_payload["evidence_state"] = {
                "collect": {
                    "status": "partial",
                    "planned_shards": 2,
                    "succeeded_shards": 1,
                    "failed_shards": 1,
                },
                "findings": {
                    "status": "partial",
                    "planned_shards": 2,
                    "succeeded_shards": 1,
                    "failed_shards": 1,
                },
                "report_mode": "incomplete",
                "reasons": ["collect_partial_shard_failures", "findings_partial_shard_failures"],
            }
            write_json(quality_path, quality_payload)

    def _create_artifact_quality_fixture_run_dir(self, run_dir: Path) -> None:
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: passed",
                    "- quality_gates: passed",
                    "- failure_reason: none",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(run_dir / "full-run.log", "full run completed\n")
        write_text(run_dir / "batch-driver.log", "driver completed with process_exit=0\n")
        write_text(
            run_dir / "run-results.tsv",
            "\n".join(
                [
                    "\t".join(
                        [
                            "1",
                            "headless",
                            "qwen-code",
                            "init",
                            "init-run",
                            "succeeded",
                            "10",
                            "1",
                            "0",
                            "1",
                            "2",
                            "1",
                            "0",
                            "qwen-code@stub",
                            "reports/taskruns/init-run-quality.json",
                            "reports",
                        ]
                    ),
                    "\t".join(
                        [
                            "1",
                            "headless",
                            "qwen-code",
                            "refresh",
                            "refresh-run",
                            "succeeded",
                            "11",
                            "1",
                            "1",
                            "1",
                            "2",
                            "1",
                            "0",
                            "qwen-code@stub",
                            "reports/taskruns/refresh-run-quality.json",
                            "reports",
                        ]
                    ),
                ]
            )
            + "\n",
        )
        self._write_snapshot(run_dir, "init-run", "init")
        self._write_snapshot(run_dir, "refresh-run", "refresh")
        refresh_quality_path = run_dir / "snapshots" / "refresh-run" / "reports" / "taskruns" / "refresh-run-quality.json"
        refresh_quality = json.loads(refresh_quality_path.read_text(encoding="utf-8"))
        refresh_quality["run_warnings"] = [
            "artifact_quality: refresh staged final set has 6 canonical documents but only 1 generic runtime-summary citation (cite.runtime-summary)"
        ]
        write_json(refresh_quality_path, refresh_quality)

    def test_python_report_prefers_runtime_parse_over_incomplete_cycle_classifier(self) -> None:
        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=self.run_dir,
            preflight={},
            classification_row={
                "failure_class": "infra_incomplete_cycle",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )
        self.assertEqual("runtime_parse", result.failure_class)
        self.assertTrue(result.runtime_parse)
        self.assertTrue(result.infra_incomplete_cycle)

    def test_python_report_prefers_runtime_timeout_over_runner_unavailable_when_timeout_signaled(self) -> None:
        run_dir = self.root / "run-timeout-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- quality_gates: skipped",
                    "- failure_reason: runtime_timeout",
                    "- expected_runs: 4",
                    "- completed_runs: 2",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 0",
                    "- running_runs_detected: 1",
                    "- termination_signal: timeout",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","error_code":"runner_unavailable","message":"runtime task timeout after 15s"}\n',
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_timeout",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_timeout", result.failure_class)
        self.assertTrue(result.runtime_timeout)

    def test_python_report_escalates_artifact_quality_warning_to_quality_gate_failure(self) -> None:
        run_dir = self.root / "run-artifact-quality"
        self._create_artifact_quality_fixture_run_dir(run_dir)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertEqual("quality_gates_failed", result.failure_class)
        self.assertTrue(result.quality_gates_failed)
        self.assertFalse(result.hard_pass)
        self.assertIn("quality:artifact-quality", result.issues)

    def test_shell_classifier_reads_taskrun_logs_and_returns_runtime_parse(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(self.run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_parse", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_prefers_runtime_timeout_over_runner_unavailable_logs(self) -> None:
        run_dir = self.root / "run-timeout-precedence"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- quality_gates: skipped",
                    "- failure_reason: runtime_timeout",
                    "- expected_runs: 4",
                    "- completed_runs: 2",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 0",
                    "- running_runs_detected: 1",
                    "- termination_signal: timeout",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "full-run.log",
            "pipeline timed out after 180s; runner_unavailable observed in step output\n",
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","error_code":"runner_unavailable","message":"runtime task timeout after 15s"}\n',
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-timeout.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_timeout", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_python_report_treats_incomplete_reports_as_triage_only_not_empty_analysis(self) -> None:
        run_dir = self.root / "run-incomplete"
        self._create_incomplete_fixture_run_dir(run_dir)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_parse",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_parse", result.failure_class)
        self.assertNotIn("analysis:overview", result.issues)
        self.assertNotIn("analysis:findings", result.issues)
        self.assertNotIn("analysis:coverage", result.issues)
        self.assertNotIn("analysis:questions", result.issues)

    def test_python_report_accepts_partial_incomplete_reports_with_substantive_coverage(self) -> None:
        run_dir = self.root / "run-partial-incomplete"
        self._create_partial_incomplete_fixture_run_dir(run_dir)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_parse",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_parse", result.failure_class)
        self.assertNotIn("analysis:coverage", result.issues)
        self.assertNotIn("analysis:questions", result.issues)
        self.assertNotIn("analysis:findings", result.issues)

    def test_shell_frontend_mode_helpers_support_per_run(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        command = (
            prelude
            + "\n"
            + "RUN_COUNT=5\n"
            + "BATCH_RUN_SELECTION=2,4\n"
            + "BATCH_FRONTEND_MODE=per_run\n"
            + "BATCH_FRONTEND_CANCEL_MODE=per_run\n"
            + "resolve_selected_run_indexes\n"
            + 'if should_run_frontend_once; then echo "live_once=1"; else echo "live_once=0"; fi\n'
            + 'if should_run_frontend_for_run 2; then echo "live_run2=1"; else echo "live_run2=0"; fi\n'
            + 'if should_run_frontend_for_run 1; then echo "live_run1=1"; else echo "live_run1=0"; fi\n'
            + 'if should_run_frontend_cancel_once; then echo "cancel_once=1"; else echo "cancel_once=0"; fi\n'
            + 'if should_run_frontend_cancel_for_run 4; then echo "cancel_run4=1"; else echo "cancel_run4=0"; fi\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        observed = set(line.strip() for line in completed.stdout.splitlines() if line.strip())
        self.assertSetEqual(
            {"live_once=0", "live_run2=1", "live_run1=0", "cancel_once=0", "cancel_run4=1"},
            observed,
        )

    def test_shell_frontend_mode_helpers_mark_auto_skip_when_run1_not_selected(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        command = (
            prelude
            + "\n"
            + "RUN_COUNT=5\n"
            + "BATCH_RUN_SELECTION=2,4\n"
            + "BATCH_FRONTEND_MODE=auto\n"
            + "resolve_selected_run_indexes\n"
            + 'if should_run_frontend_once; then echo "live_once=1"; else echo "live_once=0"; fi\n'
            + 'if should_write_frontend_skip_result; then echo "live_skip=1"; else echo "live_skip=0"; fi\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        observed = set(line.strip() for line in completed.stdout.splitlines() if line.strip())
        self.assertSetEqual({"live_once=0", "live_skip=1"}, observed)

    def test_shell_frontend_once_uses_first_selected_run_for_always_mode(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        command = (
            prelude
            + "\n"
            + "RUN_COUNT=5\n"
            + "BATCH_ROOT=/tmp/provenarch-batch\n"
            + "BATCH_RUN_SELECTION=2,4\n"
            + "BATCH_FRONTEND_MODE=always\n"
            + "resolve_selected_run_indexes\n"
            + 'printf "%s\\n" "$(resolve_frontend_live_backend_run qwen-code)"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("/tmp/provenarch-batch/qwen-code/run2\t2", completed.stdout.strip())

    def test_python_frontend_matrix_supports_per_run_results(self) -> None:
        batch_root = self.root / "batch"
        reports_root = self.root / "reports"
        write_json(
            batch_root / "frontend/qwen-code/run1/frontend-e2e-result.json",
            {
                "status": "passed",
                "reason": "ok",
                "runtime_provider": "qwen-code",
                "workspace": "/tmp/qwen-run1",
                "base_url": "http://127.0.0.1:18081",
                "runtime_command": "qwen",
            },
        )
        write_json(
            batch_root / "frontend/qwen-code/run3/frontend-e2e-result.json",
            {
                "status": "failed",
                "reason": "playwright_failed",
                "runtime_provider": "qwen-code",
                "workspace": "/tmp/qwen-run3",
                "base_url": "http://127.0.0.1:18083",
                "runtime_command": "qwen",
            },
        )
        write_json(
            batch_root / "frontend/claude-code/frontend-e2e-result.json",
            {
                "status": "passed",
                "reason": "ok",
                "runtime_provider": "claude-code",
                "workspace": "/tmp/claude",
                "base_url": "http://127.0.0.1:18082",
                "runtime_command": "claude",
            },
        )

        frontend = self.module.load_frontend_results(batch_root)
        self.assertEqual(3, len(frontend))

        matrix_path = reports_root / "frontend-matrix.md"
        self.module.write_frontend_matrix(matrix_path, frontend)
        matrix_text = matrix_path.read_text(encoding="utf-8")

        self.assertIn("| qwen-code | failed | 2 |", matrix_text)
        self.assertIn("| claude-code | passed | 1 |", matrix_text)
        self.assertIn("| qwen-code | 1 | passed | ok |", matrix_text)
        self.assertIn("| qwen-code | 3 | failed | playwright_failed |", matrix_text)

    def test_python_frontend_cancel_matrix_supports_per_run_results(self) -> None:
        batch_root = self.root / "batch-cancel"
        reports_root = self.root / "reports-cancel"
        write_json(
            batch_root / "frontend-cancel/qwen-code/run2/frontend-cancel-result.json",
            {
                "status": "passed",
                "reason": "ok",
                "scenario": "cancel-refresh",
                "runtime_provider": "qwen-code",
                "workspace": "/tmp/qwen-cancel-run2",
                "runtime_command": "qwen",
            },
        )
        write_json(
            batch_root / "frontend-cancel/qwen-code/run4/frontend-cancel-result.json",
            {
                "status": "skipped",
                "reason": "frontend_workspace_missing",
                "scenario": "cancel-refresh",
                "runtime_provider": "qwen-code",
                "workspace": "/tmp/qwen-cancel-run4",
                "runtime_command": "qwen",
            },
        )
        write_json(
            batch_root / "frontend-cancel/claude-code/frontend-cancel-result.json",
            {
                "status": "failed",
                "reason": "frontend_live_e2e_failed",
                "scenario": "cancel-refresh",
                "runtime_provider": "claude-code",
                "workspace": "/tmp/claude-cancel",
                "runtime_command": "claude",
            },
        )

        frontend_cancel = self.module.load_frontend_cancel_results(batch_root)
        self.assertEqual(3, len(frontend_cancel))

        matrix_path = reports_root / "frontend-cancel-matrix.md"
        self.module.write_frontend_cancel_matrix(matrix_path, frontend_cancel)
        matrix_text = matrix_path.read_text(encoding="utf-8")

        self.assertIn("| qwen-code | mixed | 2 | frontend_workspace_missing=1, ok=1 |", matrix_text)
        self.assertIn("| claude-code | failed | 1 | frontend_live_e2e_failed=1 |", matrix_text)
        self.assertIn("| qwen-code | 2 | passed | ok | cancel-refresh |", matrix_text)
        self.assertIn("| qwen-code | 4 | skipped | frontend_workspace_missing | cancel-refresh |", matrix_text)
        self.assertIn("| claude-code | - | failed | frontend_live_e2e_failed | cancel-refresh |", matrix_text)

    def test_quality_report_respects_selected_provider_surface(self) -> None:
        reports_root = self.root / "reports-selected"
        quality_path = reports_root / "quality.md"
        frontend = [
            {
                "status": "passed",
                "reason": "ok",
                "runtime_provider": "qwen-code",
                "run_index": 1,
                "workspace": "/tmp/qwen-run1",
                "runtime_command": "qwen",
            }
        ]
        frontend_cancel = [
            {
                "status": "passed",
                "reason": "ok",
                "scenario": "cancel-refresh",
                "runtime_provider": "qwen-code",
                "run_index": 1,
                "workspace": "/tmp/qwen-run1",
                "runtime_command": "qwen",
            }
        ]
        runs = [
            self.module.RunEvaluation(
                provider="qwen-code",
                run_index=1,
                run_dir=self.root / "qwen-only-run1",
                hard_pass=True,
                reliability=30,
                contract=20,
                analysis=20,
                total=70,
                verdict="PASS",
            )
        ]
        preflight = {
            "generated_at_utc": "2026-04-18T00:00:00Z",
            "provenarch_sha": "abc123",
            "target_repos_file": "examples/repos.txt",
            "declared_repos_meta": {
                "profile_id": "single-git_url",
                "profile_source_kind": "git_url",
                "expected_repo_count": 1,
                "declared_repos": [{"name": "bank-of-anthos"}],
            },
            "selected_providers": ["qwen-code"],
            "selected_run_indexes": ["1"],
            "runtimes": {
                "claude": {"version_line": "not-selected"},
                "qwen": {"version_line": "qwen 0.1"},
            },
        }

        self.module.write_quality_report(
            quality_path,
            "batch-qwen-only",
            runs,
            frontend,
            frontend_cancel,
            preflight,
            ["qwen-code"],
        )
        report = quality_path.read_text(encoding="utf-8")
        self.assertIn("`1/1` backend full-runs", report)
        self.assertIn("выбранных провайдеров (`1/1`)", report)
        self.assertNotIn("10/10", report)
        self.assertNotIn("2/2", report)
        self.assertNotIn("| claude-code |", report)

    def test_selected_surface_is_resolved_from_preflight(self) -> None:
        batch_root = self.root / "selected-surface"
        classifications = {
            ("qwen-code", 1): {"failure_class": "none"},
            ("qwen-code", 2): {"failure_class": "none"},
        }
        preflight = {
            "selected_providers": ["qwen-code"],
            "selected_run_indexes": ["1", "2"],
        }

        providers = self.module.resolve_selected_providers(preflight, classifications, batch_root)
        run_indexes = self.module.resolve_selected_run_indexes(preflight, classifications, batch_root)

        self.assertEqual(["qwen-code"], providers)
        self.assertEqual([1, 2], run_indexes)

if __name__ == "__main__":
    unittest.main()
