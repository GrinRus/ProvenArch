import json
import importlib.util
import os
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path


class PipelineWatchdogTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.watchdog = cls.repo_root / "scripts" / "pipeline-watchdog.py"
        spec = importlib.util.spec_from_file_location("pipeline_watchdog", cls.watchdog)
        if spec is None or spec.loader is None:
            raise RuntimeError("failed to load pipeline-watchdog module")
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        cls.watchdog_module = module

    def test_timeout_kills_process_group_and_writes_terminal_status(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            output = root / "pipeline.log"
            metadata = root / "watchdog.json"
            status_file = root / "run-status.env"
            child_marker = root / "child-survived"
            status_file.write_text("provider=codex-code\nrun_index=1\nstate=running\n", encoding="utf-8")
            command = "\n".join(
                [
                    "import os, signal, subprocess, sys, time",
                    "marker = sys.argv[1]",
                    "child = subprocess.Popen([sys.executable, '-c', \"import pathlib, signal, sys, time; signal.signal(signal.SIGTERM, signal.SIG_IGN); time.sleep(1); pathlib.Path(sys.argv[1]).write_text('survived', encoding='utf-8')\", marker])",
                    "print('started', flush=True)",
                    "child.wait()",
                ]
            )

            result = subprocess.run(
                [
                    sys.executable,
                    str(self.watchdog),
                    "--timeout-sec",
                    "0.2",
                    "--grace-sec",
                    "0.1",
                    "--heartbeat-sec",
                    "0.1",
                    "--poll-sec",
                    "0.05",
                    "--output",
                    str(output),
                    "--metadata",
                    str(metadata),
                    "--status-file",
                    str(status_file),
                    "--last-pipeline-stage",
                    "iteration=1 runtime=headless:codex-code pipeline=refresh",
                    "--last-runtime-provider",
                    "codex-code",
                    "--",
                    sys.executable,
                    "-c",
                    command,
                    str(child_marker),
                ],
                cwd=self.repo_root,
                text=True,
                capture_output=True,
            )

            self.assertEqual(124, result.returncode, result.stderr + result.stdout)
            payload = json.loads(metadata.read_text(encoding="utf-8"))
            self.assertTrue(payload["timed_out"], payload)
            self.assertEqual("timeout", payload["termination_signal"])
            self.assertEqual(124, payload["process_exit"])
            self.assertGreaterEqual(payload["pipeline_timeout_elapsed_sec"], 0)
            status = status_file.read_text(encoding="utf-8")
            self.assertIn("state=process_failed", status)
            self.assertIn("termination_signal=timeout", status)
            self.assertIn("failure_reason=runtime_timeout", status)
            time.sleep(1.0)
            self.assertFalse(child_marker.exists(), "child process group survived watchdog timeout")

    def test_clock_gap_diagnostic_is_recorded_without_timeout(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            output = root / "pipeline.log"
            metadata = root / "watchdog.json"
            result = subprocess.run(
                [
                    sys.executable,
                    str(self.watchdog),
                    "--timeout-sec",
                    "2",
                    "--grace-sec",
                    "0.1",
                    "--heartbeat-sec",
                    "1",
                    "--poll-sec",
                    "0.1",
                    "--clock-gap-threshold-sec",
                    "0.05",
                    "--output",
                    str(output),
                    "--metadata",
                    str(metadata),
                    "--",
                    sys.executable,
                    "-c",
                    "import time; time.sleep(0.25)",
                ],
                cwd=self.repo_root,
                text=True,
                capture_output=True,
            )

            self.assertEqual(0, result.returncode, result.stderr + result.stdout)
            payload = json.loads(metadata.read_text(encoding="utf-8"))
            self.assertFalse(payload["timed_out"], payload)
            self.assertTrue(payload["infra_host_sleep_or_clock_jump_detected"], payload)
            self.assertGreater(payload["max_watchdog_tick_gap_sec"], 0)
            self.assertIn("infra_host_sleep_or_clock_jump_detected", result.stdout)

    def test_clean_process_exit_is_useful_progress(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            output = root / "pipeline.log"
            metadata = root / "watchdog.json"

            result = subprocess.run(
                [
                    sys.executable,
                    str(self.watchdog),
                    "--timeout-sec",
                    "2",
                    "--grace-sec",
                    "0.1",
                    "--heartbeat-sec",
                    "0",
                    "--poll-sec",
                    "0.05",
                    "--output",
                    str(output),
                    "--metadata",
                    str(metadata),
                    "--",
                    sys.executable,
                    "-c",
                    "import time; time.sleep(0.1)",
                ],
                cwd=self.repo_root,
                text=True,
                capture_output=True,
            )

            self.assertEqual(0, result.returncode, result.stderr + result.stdout)
            payload = json.loads(metadata.read_text(encoding="utf-8"))
            self.assertFalse(payload["timed_out"], payload)
            self.assertEqual("process_exit", payload["last_progress_source"], payload)
            self.assertGreater(payload["last_progress_epoch"], payload["started_epoch"], payload)

    def test_runtime_heartbeat_and_raw_logs_are_not_useful_progress(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            workspace = root / "workspace"
            output = root / "pipeline.log"
            metadata = root / "watchdog.json"
            command = "\n".join(
                [
                    "import pathlib, sys, time",
                    "workspace = pathlib.Path(sys.argv[1])",
                    "run = workspace / 'reports' / 'taskruns' / 'run_heartbeat'",
                    "(run / 'logs').mkdir(parents=True, exist_ok=True)",
                    "(run / 'raw').mkdir(parents=True, exist_ok=True)",
                    "(run / 'logs' / 'live.ndjson').write_text('runtime task heartbeat\\n', encoding='utf-8')",
                    "(run / 'raw' / 'metadata.json').write_text('{\"status\":\"heartbeat\"}\\n', encoding='utf-8')",
                    "(workspace / 'reports' / 'taskruns' / 'run-history.json').write_text('{\"status\":\"running\"}\\n', encoding='utf-8')",
                    "print('runtime task heartbeat run_id=run_heartbeat', flush=True)",
                    "print('[pipeline-watchdog] progress nested elapsed_sec=1', flush=True)",
                    "time.sleep(2)",
                ]
            )

            result = subprocess.run(
                [
                    sys.executable,
                    str(self.watchdog),
                    "--timeout-sec",
                    "0.4",
                    "--grace-sec",
                    "0.1",
                    "--heartbeat-sec",
                    "0",
                    "--poll-sec",
                    "0.05",
                    "--output",
                    str(output),
                    "--metadata",
                    str(metadata),
                    "--workspace",
                    str(workspace),
                    "--",
                    sys.executable,
                    "-c",
                    command,
                    str(workspace),
                ],
                cwd=self.repo_root,
                text=True,
                capture_output=True,
            )

            self.assertEqual(124, result.returncode, result.stderr + result.stdout)
            payload = json.loads(metadata.read_text(encoding="utf-8"))
            self.assertTrue(payload["timed_out"], payload)
            self.assertEqual("process_start", payload["last_progress_source"], payload)
            self.assertGreater(payload["last_output_activity_epoch"], payload["started_epoch"], payload)

    def test_artifact_scan_ignores_raw_surfaces_but_counts_staged_files(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            workspace = Path(tmpdir) / "workspace"
            run = workspace / "reports" / "taskruns" / "run_1"
            (run / "logs").mkdir(parents=True)
            (run / "raw").mkdir(parents=True)
            (run / "logs" / "live.ndjson").write_text("runtime task heartbeat\n", encoding="utf-8")
            (run / "raw" / "metadata.json").write_text("{}\n", encoding="utf-8")
            (workspace / "reports" / "taskruns" / "run-history.json").write_text("{}\n", encoding="utf-8")

            self.assertIsNone(self.watchdog_module.newest_mtime(workspace))

            artifact = run / "staging" / "final" / "reports" / "as-is" / "overview.md"
            artifact.parent.mkdir(parents=True)
            artifact.write_text("# Overview\n\nEvidence-backed content.\n", encoding="utf-8")

            self.assertIsNotNone(self.watchdog_module.newest_mtime(workspace))


if __name__ == "__main__":
    unittest.main()
