import importlib.util
import os
import sys
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
WRITE_BATCH_PREFLIGHT_PATH = REPO_ROOT / "scripts" / "write-batch-preflight.py"


def load_module():
    spec = importlib.util.spec_from_file_location("write_batch_preflight", WRITE_BATCH_PREFLIGHT_PATH)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class WriteBatchPreflightTest(unittest.TestCase):
    def setUp(self) -> None:
        self.module = load_module()
        self.tempdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tempdir.name)

    def tearDown(self) -> None:
        self.tempdir.cleanup()

    def _write_script(self, name: str, body: str) -> str:
        path = self.root / name
        path.write_text(body, encoding="utf-8")
        path.chmod(0o755)
        return str(path)

    def test_probe_provider_readiness_returns_not_selected(self) -> None:
        result = self.module.probe_provider_readiness("qwen", "not-selected", str(REPO_ROOT))
        self.assertEqual("not_selected", result["status"])

    def test_probe_provider_readiness_marks_quota_signal_unavailable(self) -> None:
        command = self._write_script(
            "qwen-stub",
            "#!/bin/sh\n"
            "printf '%s\n' '[{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"API Error: 403 permission_error usage limit\"}]}}]'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("unavailable", result["status"])
        self.assertEqual("quota_or_permission", result["subclass"])
        self.assertIn("permission_error", result["reason"])

    def test_probe_provider_readiness_marks_successful_probe_ready(self) -> None:
        command = self._write_script("claude-stub", "#!/bin/sh\nprintf '%s\n' '{\"ok\":true}'\n")

        result = self.module.probe_provider_readiness("claude", command, str(REPO_ROOT))
        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])

    def test_probe_provider_readiness_blocks_old_codex_for_gpt55(self) -> None:
        command = self._write_script("codex-stub", "#!/bin/sh\nprintf '%s\n' 'codex-cli 0.118.0'\n")

        result = self.module.probe_provider_readiness(
            "codex",
            command,
            str(REPO_ROOT),
            "codex-cli 0.118.0",
            'model = "gpt-5.5"\n',
        )

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("codex_model_requires_newer_cli", result["subclass"])

    def test_probe_provider_readiness_allows_updated_codex_for_gpt55(self) -> None:
        command = self._write_script("codex-stub", "#!/bin/sh\nprintf '%s\n' 'codex-cli 0.125.0'\n")

        result = self.module.probe_provider_readiness(
            "codex",
            command,
            str(REPO_ROOT),
            "codex-cli 0.125.0",
            'model = "gpt-5.5"\n',
        )

        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])

    def test_selected_readiness_keys_limits_non_release_provider_filter(self) -> None:
        self.assertEqual(["qwen"], self.module.selected_readiness_keys(["qwen-code"]))
        self.assertEqual(["claude", "codex"], self.module.selected_readiness_keys(["claude-code", "codex-code"]))
        self.assertEqual(["claude", "qwen", "codex"], self.module.selected_readiness_keys([]))


if __name__ == "__main__":
    unittest.main()
