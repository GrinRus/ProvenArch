import json
import os
import subprocess
import unittest
from pathlib import Path


class SmokeApiLogsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]

    def _run_helper(self, command: str, payload: object) -> subprocess.CompletedProcess[str]:
        env = {
            **os.environ,
            "ACP_SMOKE_API_LIB_ONLY": "1",
            "PAYLOAD": payload if isinstance(payload, str) else json.dumps(payload),
        }
        return subprocess.run(
            ["bash", "-c", f"source scripts/smoke-api.sh; {command}"],
            cwd=self.repo_root,
            env=env,
            capture_output=True,
            text=True,
        )

    def test_logs_page_validator_accepts_non_empty_page_and_returns_next_cursor(self) -> None:
        payload = {
            "run_id": "run-1",
            "items": [
                {
                    "cursor": 0,
                    "timestamp": "2026-07-14T00:00:00Z",
                    "level": "info",
                    "kind": "event",
                    "message": "pipeline started",
                },
                {
                    "cursor": 1,
                    "timestamp": "2026-07-14T00:00:01Z",
                    "level": "info",
                    "kind": "runtime_output",
                    "stream": "stdout",
                    "message": "collecting evidence",
                    "fields": {"provider": "fake"},
                },
            ],
            "next_cursor": 2,
            "eof": False,
        }

        result = self._run_helper('validate_logs_page "$PAYLOAD" run-1 0 2 1', payload)

        self.assertEqual(0, result.returncode, result.stderr)
        self.assertEqual("2", result.stdout.strip())

    def test_logs_page_validator_accepts_empty_tail_page(self) -> None:
        payload = {
            "run_id": "run-1",
            "items": [],
            "next_cursor": 2,
            "eof": True,
        }

        result = self._run_helper('validate_logs_page "$PAYLOAD" run-1 2 2 0', payload)

        self.assertEqual(0, result.returncode, result.stderr)
        self.assertEqual("2", result.stdout.strip())

    def test_logs_page_validator_rejects_malformed_payload(self) -> None:
        payload = {
            "run_id": "run-1",
            "items": {"cursor": 0},
            "next_cursor": 1,
            "eof": True,
        }

        result = self._run_helper('validate_logs_page "$PAYLOAD" run-1 0 2 0', payload)

        self.assertNotEqual(0, result.returncode)
        self.assertIn("items must be an array", result.stderr)

    def test_logs_status_check_rejects_non_2xx_response(self) -> None:
        payload = {"error": {"code": "run_logs_unavailable", "message": "boom"}}

        result = self._run_helper('assert_status_ok 500 "$PAYLOAD" "run logs first page"', payload)

        self.assertNotEqual(0, result.returncode)
        self.assertIn("expected run logs first page status 200, got 500", result.stderr)

    def test_invalid_cursor_response_requires_expected_error_code(self) -> None:
        payload = {"error": {"code": "invalid_cursor", "message": "cursor must be non-negative"}}

        result = self._run_helper('assert_error_response "$PAYLOAD" 400 400 invalid_cursor "logs invalid cursor"', payload)

        self.assertEqual(0, result.returncode, result.stderr)

    def test_invalid_cursor_response_rejects_wrong_error_code(self) -> None:
        payload = {"error": {"code": "run_not_found", "message": "run not found"}}

        result = self._run_helper('assert_error_response "$PAYLOAD" 400 400 invalid_cursor "logs invalid cursor"', payload)

        self.assertNotEqual(0, result.returncode)
        self.assertIn("expected logs invalid cursor error code invalid_cursor", result.stderr)


if __name__ == "__main__":
    unittest.main()
