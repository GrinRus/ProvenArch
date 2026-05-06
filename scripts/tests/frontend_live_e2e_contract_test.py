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


if __name__ == "__main__":
    unittest.main()
