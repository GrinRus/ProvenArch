import subprocess
import tempfile
import textwrap
import time
import unittest
from pathlib import Path


class RunStatusHeartbeatTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.helper = cls.repo_root / "scripts" / "run-status-heartbeat.sh"

    def test_running_heartbeat_refreshes_progress_metadata(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            run_status = tmp_root / "run-status.env"
            run_status.write_text(
                textwrap.dedent(
                    """\
                    provider=qwen-code
                    run_index=1
                    state=running
                    process_exit=
                    termination_signal=none
                    failure_reason=
                    summary_written=no
                    updated_at=2026-04-20T00:00:00Z
                    last_pipeline_stage=not_started
                    last_runtime_provider=unset
                    last_progress_at=2026-04-20T00:00:00Z
                    """
                ),
                encoding="utf-8",
            )

            time.sleep(1)
            script = textwrap.dedent(
                f"""\
                set -Eeuo pipefail
                source "{self.helper}"
                RUN_STATUS_FILE="{run_status}"
                LAST_PIPELINE_STAGE="step1.collect"
                LAST_RUNTIME_PROVIDER="qwen-code"
                write_running_run_status_heartbeat
                """
            )
            subprocess.run(["bash", "-lc", script], check=True, cwd=self.repo_root)

            fields = {}
            for raw_line in run_status.read_text(encoding="utf-8").splitlines():
                if "=" not in raw_line:
                    continue
                key, value = raw_line.split("=", 1)
                fields[key] = value

            self.assertEqual("running", fields.get("state"))
            self.assertEqual("step1.collect", fields.get("last_pipeline_stage"))
            self.assertEqual("qwen-code", fields.get("last_runtime_provider"))
            self.assertNotEqual("2026-04-20T00:00:00Z", fields.get("updated_at"))
            self.assertNotEqual("2026-04-20T00:00:00Z", fields.get("last_progress_at"))

    def test_running_heartbeat_does_not_override_terminal_status(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            run_status = tmp_root / "run-status.env"
            original = textwrap.dedent(
                """\
                provider=qwen-code
                run_index=1
                state=process_failed
                process_exit=1
                termination_signal=none
                failure_reason=runtime_parse
                summary_written=yes
                updated_at=2026-04-20T00:00:00Z
                last_pipeline_stage=step2.asis_docs
                last_runtime_provider=qwen-code
                last_progress_at=2026-04-20T00:00:00Z
                """
            )
            run_status.write_text(original, encoding="utf-8")

            script = textwrap.dedent(
                f"""\
                set -Eeuo pipefail
                source "{self.helper}"
                RUN_STATUS_FILE="{run_status}"
                LAST_PIPELINE_STAGE="step4.proposals"
                LAST_RUNTIME_PROVIDER="claude-code"
                write_running_run_status_heartbeat
                """
            )
            subprocess.run(["bash", "-lc", script], check=True, cwd=self.repo_root)

            self.assertEqual(original, run_status.read_text(encoding="utf-8"))
