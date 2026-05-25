import json
import os
import socket
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


class FrontendLiveE2EContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.script_path = cls.repo_root / "scripts" / "frontend-live-e2e.sh"

    def test_init_timeout_cap_defaults_to_disabled(self) -> None:
        body = self.script_path.read_text(encoding="utf-8")
        self.assertIn('UI_E2E_INIT_TIMEOUT_CAP_SEC="${UI_E2E_INIT_TIMEOUT_CAP_SEC:-0}"', body)
        self.assertNotIn("UI_E2E_INIT_TIMEOUT_CAP_SEC:-1800", body)

    def test_pipeline_timeout_guard_only_caps_when_opted_in(self) -> None:
        body = self.script_path.read_text(encoding="utf-8")
        self.assertIn("min_init_timeout_sec=$((pipeline_timeout_sec + 30))", body)
        self.assertIn(
            "if (( init_timeout_cap_sec > 0 && min_init_timeout_sec > init_timeout_cap_sec )); then",
            body,
        )

    def test_frontend_reason_taxonomy_includes_diagnostic_failures(self) -> None:
        reasons_path = self.repo_root / "scripts" / "frontend-status-reasons.sh"
        body = reasons_path.read_text(encoding="utf-8")
        self.assertIn('ACP_FRONTEND_REASON_BROWSER_CLOSED="browser_closed"', body)
        self.assertIn('ACP_FRONTEND_REASON_API_UNREACHABLE="api_unreachable"', body)
        self.assertIn('ACP_FRONTEND_REASON_SERVER_EXITED="server_exited"', body)

    def test_live_flow_uses_independent_api_request_context(self) -> None:
        spec_path = self.repo_root / "ui" / "e2e" / "live-flow.spec.ts"
        body = spec_path.read_text(encoding="utf-8")
        self.assertIn("type APIRequestContext", body)
        self.assertIn("fetchRunObservation(api: APIRequestContext", body)
        self.assertIn('scenario !== "api-context-page-close-smoke"', body)
        self.assertNotIn("page.request", body)
        self.assertNotIn("page.waitForTimeout", body)

    def test_shell_allows_api_context_page_close_smoke(self) -> None:
        body = self.script_path.read_text(encoding="utf-8")
        self.assertIn("init-inspect|cancel-refresh|api-context-page-close-smoke)", body)

    def test_server_exit_is_classified(self) -> None:
        result = self._run_frontend_harness("server_exited", acp_mode="server_exited")
        self.assertEqual("failed", result["status"])
        self.assertEqual("server_exited", result["reason"])
        self.assertIsInstance(result["server_exit_code"], int)
        self.assertEqual("server_exited", result["health_after_failure"])
        self.assertEqual("run_stub", result["run_id"])
        self.assertEqual("running", result["last_run_status"])
        self.assertEqual("init.step1.collect", result["last_run_current_step"])

    def test_api_unreachable_is_classified(self) -> None:
        result = self._run_frontend_harness("api_unreachable")
        self.assertEqual("failed", result["status"])
        self.assertEqual("api_unreachable", result["reason"])
        self.assertIsNone(result["server_exit_code"])
        self.assertEqual("failed", result["health_after_failure"])

    def test_browser_closed_is_classified(self) -> None:
        result = self._run_frontend_harness("browser_closed")
        self.assertEqual("failed", result["status"])
        self.assertEqual("browser_closed", result["reason"])
        self.assertEqual("ok", result["health_after_failure"])
        self.assertEqual("run_stub", result["run_id"])

    def test_active_run_timeout_remains_distinct(self) -> None:
        result = self._run_frontend_harness("active_timeout")
        self.assertEqual("failed", result["status"])
        self.assertEqual("active_run_timeout", result["reason"])
        self.assertEqual("ok", result["health_after_failure"])

    def _run_frontend_harness(self, npm_mode: str, acp_mode: str = "healthy") -> dict[str, object]:
        with tempfile.TemporaryDirectory() as raw_tmp:
            tmp = Path(raw_tmp)
            workspace = tmp / "workspace"
            output_dir = tmp / "output"
            marker = tmp / "health-fail.marker"
            workspace.joinpath("reports", "taskruns").mkdir(parents=True)
            (workspace / "reports" / "taskruns" / "run-history.json").write_text(
                json.dumps(
                    {
                        "version": 1,
                        "items": [
                            {
                                "run_id": "run_stub",
                                "pipeline": "init",
                                "status": "running",
                                "current_step": "init.step1.collect",
                            }
                        ],
                    }
                ),
                encoding="utf-8",
            )
            fake_acp = tmp / "fake-acp.py"
            fake_npm = tmp / "fake-npm.py"
            self._write_executable(fake_acp, self._fake_acp_source())
            self._write_executable(fake_npm, self._fake_npm_source())
            listen = f"127.0.0.1:{self._free_port()}"
            env = {
                **os.environ,
                "ACP_BIN": str(fake_acp),
                "ACP_NPM_BIN": str(fake_npm),
                "ACP_QWEN_CMD": "/usr/bin/true",
                "ACP_API_READY_TIMEOUT_SEC": "5",
                "WORKSPACE": str(workspace),
                "RUNTIME_PROVIDER": "qwen-code",
                "OUTPUT_DIR": str(output_dir),
                "LISTEN": listen,
                "FAKE_ACP_MODE": acp_mode,
                "FAKE_NPM_MODE": npm_mode,
                "FAKE_HEALTH_FAIL_MARKER": str(marker),
            }
            completed = subprocess.run(
                ["bash", str(self.script_path)],
                cwd=self.repo_root,
                env=env,
                capture_output=True,
                text=True,
                timeout=15,
            )
            self.assertNotEqual(0, completed.returncode, completed.stdout + completed.stderr)
            result_path = output_dir / "frontend-e2e-result.json"
            self.assertTrue(result_path.is_file(), completed.stdout + completed.stderr)
            return json.loads(result_path.read_text(encoding="utf-8"))

    @staticmethod
    def _write_executable(path: Path, content: str) -> None:
        path.write_text(content, encoding="utf-8")
        path.chmod(path.stat().st_mode | stat.S_IXUSR)

    @staticmethod
    def _free_port() -> int:
        sock = socket.socket()
        sock.bind(("127.0.0.1", 0))
        port = sock.getsockname()[1]
        sock.close()
        return int(port)

    @staticmethod
    def _fake_acp_source() -> str:
        return textwrap.dedent(
            """\
            #!/usr/bin/env python3
            import json
            import os
            import signal
            import sys
            import time
            from http.server import BaseHTTPRequestHandler, HTTPServer
            from pathlib import Path

            args = sys.argv[1:]
            listen = args[args.index("--listen") + 1] if "--listen" in args else "127.0.0.1:18080"
            host, port_raw = listen.rsplit(":", 1)
            marker = os.environ.get("FAKE_HEALTH_FAIL_MARKER", "")
            mode = os.environ.get("FAKE_ACP_MODE", "healthy")
            stop = False

            class Handler(BaseHTTPRequestHandler):
                def log_message(self, *_args):
                    return

                def _json(self, status, payload):
                    body = json.dumps(payload).encode("utf-8")
                    self.send_response(status)
                    self.send_header("content-type", "application/json")
                    self.send_header("content-length", str(len(body)))
                    self.end_headers()
                    self.wfile.write(body)

                def do_GET(self):
                    if self.path == "/api/health":
                        if marker and Path(marker).exists():
                            self._json(503, {"status": "unhealthy"})
                        else:
                            self._json(200, {"status": "ok"})
                        return
                    if self.path == "/api/runtime/timeouts":
                        self._json(200, {"effective": {"ui_init_poll_timeout_sec": 900, "ui_cancel_poll_timeout_sec": 420}})
                        return
                    if self.path.startswith("/api/pipeline/runs/"):
                        self._json(200, {"status": "running", "current_step": "init.step1.collect", "warnings": []})
                        return
                    self._json(404, {"error": "not found"})

            def handle_signal(_signum, _frame):
                global stop
                stop = True

            signal.signal(signal.SIGTERM, handle_signal)
            signal.signal(signal.SIGINT, handle_signal)
            server = HTTPServer((host, int(port_raw)), Handler)
            server.timeout = 0.1
            deadline = time.time() + 1.0 if mode == "server_exited" else None
            while not stop:
                if deadline is not None and time.time() >= deadline:
                    break
                server.handle_request()
            server.server_close()
            """
        )

    @staticmethod
    def _fake_npm_source() -> str:
        return textwrap.dedent(
            """\
            #!/usr/bin/env python3
            import os
            import sys
            import time
            from pathlib import Path

            mode = os.environ.get("FAKE_NPM_MODE", "browser_closed")
            print("ACP_UI_E2E_RUN_ID=run_stub")
            if mode == "server_exited":
                time.sleep(1.5)
                print("playwright failed after server exit", file=sys.stderr)
                sys.exit(1)
            if mode == "api_unreachable":
                marker = os.environ.get("FAKE_HEALTH_FAIL_MARKER", "")
                if marker:
                    Path(marker).write_text("1", encoding="utf-8")
                print("apiRequestContext.get: connect ECONNREFUSED 127.0.0.1:12345", file=sys.stderr)
                sys.exit(1)
            if mode == "active_timeout":
                print("ACTIVE_RUN_TIMEOUT: run run_stub stayed productive", file=sys.stderr)
                sys.exit(1)
            if mode == "browser_closed":
                print("Error: page.waitForTimeout: Target page, context or browser has been closed", file=sys.stderr)
                sys.exit(1)
            sys.exit(1)
            """
        )


if __name__ == "__main__":
    unittest.main()
