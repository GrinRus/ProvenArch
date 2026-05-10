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

    def test_qwen_headless_probe_remains_read_only_prompt(self) -> None:
        args, stdin_text = self.module.headless_probe_invocation("qwen")

        self.assertEqual(2, len(args))
        self.assertEqual("-p", args[0])
        self.assertIn("ACP_READY", args[1])
        self.assertNotIn("--yolo", args)
        self.assertNotIn("--approval-mode", args)
        self.assertEqual("", stdin_text)

    def test_qwen_artifact_smoke_uses_runtime_write_args(self) -> None:
        sentinel_path = self.root / "write-dir" / "sentinel.txt"

        args, stdin_text = self.module.artifact_smoke_invocation("qwen", sentinel_path)

        self.assertEqual(
            ["--chat-recording", "false", "--yolo", "--channel", "CI", "-p"],
            args[:6],
        )
        self.assertIn(str(sentinel_path), args[6])
        self.assertEqual("", stdin_text)

    def test_artifact_smoke_timeout_default_is_longer_than_read_only_probe(self) -> None:
        old_probe = os.environ.pop("ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC", None)
        old_smoke = os.environ.pop("ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC", None)
        try:
            self.assertEqual(30, self.module.probe_timeout_sec())
            self.assertEqual(120, self.module.artifact_smoke_timeout_sec())
        finally:
            if old_probe is not None:
                os.environ["ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC"] = old_probe
            if old_smoke is not None:
                os.environ["ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC"] = old_smoke

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
        command = self._write_script(
            "claude-stub",
            "#!/bin/sh\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"; printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"; exit 0; fi\n"
            "printf '%s\n' '{\"ok\":true}'\n",
        )

        result = self.module.probe_provider_readiness("claude", command, str(REPO_ROOT))
        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])

    def test_probe_provider_readiness_uses_qwen_runtime_args_for_artifact_smoke(self) -> None:
        command = self._write_script(
            "qwen-runtime-smoke-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "if [ \"${1:-}\" = \"-p\" ]; then printf '%s\n' 'ACP_READY'; exit 0; fi\n"
            "if [ \"${1:-}\" = \"--chat-recording\" ] && [ \"${2:-}\" = \"false\" ] && [ \"${3:-}\" = \"--yolo\" ] && [ \"${4:-}\" = \"--channel\" ] && [ \"${5:-}\" = \"CI\" ] && [ \"${6:-}\" = \"-p\" ]; then\n"
            "  mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"\n"
            "  printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"\n"
            "  printf '%s\n' 'Done.'\n"
            "  exit 0\n"
            "fi\n"
            "printf 'unexpected args: %s\\n' \"$*\" >&2\n"
            "exit 2\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])

    def test_probe_provider_readiness_checks_headless_invocation(self) -> None:
        command = self._write_script(
            "qwen-headless-quota-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "printf '%s\n' 'API Error: 429 rate limit'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("unavailable", result["status"])
        self.assertEqual("quota_or_permission", result["subclass"])
        self.assertIn("rate limit", result["reason"])

    def test_probe_provider_readiness_blocks_provider_model_mismatch(self) -> None:
        command = self._write_script(
            "qwen-claude-model-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "printf '%s\n' '{\"model\":\"claude-opus-4-5-20251101\",\"status\":\"ok\"}'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("unavailable", result["status"])
        self.assertEqual("provider_model_mismatch", result["subclass"])
        self.assertEqual("1", result["model_mismatch"])
        self.assertIn("claude-opus", result["observed_models"])

    def test_probe_provider_readiness_blocks_escaped_provider_model_mismatch(self) -> None:
        command = self._write_script(
            "qwen-escaped-claude-model-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "printf '%s\n' '{\\\"model\\\":\\\"claude-opus-4-5-20251101\\\",\\\"status\\\":\\\"ok\\\"}'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("unavailable", result["status"])
        self.assertEqual("provider_model_mismatch", result["subclass"])
        self.assertIn("claude-opus", result["observed_models"])

    def test_probe_provider_readiness_blocks_model_mismatch_from_artifact_smoke_output(self) -> None:
        command = self._write_script(
            "qwen-kimi-smoke-model-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "if [ \"${1:-}\" = \"-p\" ]; then printf '%s\n' 'ACP_READY'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then\n"
            "  mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"\n"
            "  printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"\n"
            "  printf '%s\n' '{\"model\":\"kimi-for-coding\",\"status\":\"ok\"}'\n"
            "  exit 0\n"
            "fi\n"
            "exit 2\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("provider_model_mismatch", result["subclass"])
        self.assertEqual("1", result["model_mismatch"])
        self.assertIn("kimi-for-coding", result["observed_models"])

    def test_probe_provider_readiness_blocks_nested_model_usage_mismatch(self) -> None:
        command = self._write_script(
            "claude-kimi-model-usage-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' '2.1.85 (Claude Code)'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"; printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"; exit 0; fi\n"
            "printf '%s\n' '{\"type\":\"result\",\"modelUsage\":{\"kimi-for-coding\":{\"inputTokens\":1,\"outputTokens\":1}},\"result\":\"ACP_READY\"}'\n",
        )

        result = self.module.probe_provider_readiness("claude", command, str(REPO_ROOT))

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("provider_model_mismatch", result["subclass"])
        self.assertEqual("1", result["model_mismatch"])
        self.assertIn("kimi-for-coding", result["observed_models"])

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
        command = self._write_script(
            "codex-stub",
            "#!/bin/sh\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"; printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"; exit 0; fi\n"
            "printf '%s\n' 'codex-cli 0.125.0'\n",
        )

        result = self.module.probe_provider_readiness(
            "codex",
            command,
            str(REPO_ROOT),
            "codex-cli 0.125.0",
            'model = "gpt-5.5"\n',
        )

        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])

    def test_probe_provider_readiness_blocks_artifact_smoke_failure(self) -> None:
        command = self._write_script(
            "qwen-no-artifact-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "printf '%s\n' 'ACP_READY'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("unavailable", result["status"])
        self.assertEqual("operational_host_preflight_failed", result["subclass"])
        self.assertEqual("failed", result["artifact_smoke"])

    def test_qwen_artifact_smoke_uses_runtime_like_tool_args(self) -> None:
        command = self._write_script(
            "qwen-runtime-like-artifact-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "has_yolo=0\n"
            "has_channel_ci=0\n"
            "prev=''\n"
            "for arg in \"$@\"; do\n"
            "  if [ \"$arg\" = \"--yolo\" ]; then has_yolo=1; fi\n"
            "  if [ \"$prev\" = \"--channel\" ] && [ \"$arg\" = \"CI\" ]; then has_channel_ci=1; fi\n"
            "  prev=\"$arg\"\n"
            "done\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ] && [ \"$has_yolo\" = \"1\" ] && [ \"$has_channel_ci\" = \"1\" ]; then\n"
            "  mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"\n"
            "  printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"\n"
            "  exit 0\n"
            "fi\n"
            "printf '%s\n' 'ACP_READY'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])

    def test_selected_readiness_keys_limits_non_release_provider_filter(self) -> None:
        self.assertEqual(["qwen"], self.module.selected_readiness_keys(["qwen-code"]))
        self.assertEqual(["claude", "codex"], self.module.selected_readiness_keys(["claude-code", "codex-code"]))
        self.assertEqual(["claude", "qwen", "codex"], self.module.selected_readiness_keys([]))


if __name__ == "__main__":
    unittest.main()
