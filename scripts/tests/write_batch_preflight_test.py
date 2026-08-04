import importlib.util
import os
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path
from unittest import mock


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
        self._old_codex_model = os.environ.pop("ACP_CODEX_MODEL", None)
        self._old_codex_reasoning_effort = os.environ.pop("ACP_CODEX_REASONING_EFFORT", None)

    def tearDown(self) -> None:
        if self._old_codex_model is not None:
            os.environ["ACP_CODEX_MODEL"] = self._old_codex_model
        else:
            os.environ.pop("ACP_CODEX_MODEL", None)
        if self._old_codex_reasoning_effort is not None:
            os.environ["ACP_CODEX_REASONING_EFFORT"] = self._old_codex_reasoning_effort
        else:
            os.environ.pop("ACP_CODEX_REASONING_EFFORT", None)
        self.tempdir.cleanup()

    def _write_script(self, name: str, body: str) -> str:
        path = self.root / name
        path.write_text(body, encoding="utf-8")
        path.chmod(0o755)
        return str(path)

    def _probe_claude_artifact_smoke_timeout_retry(
        self,
        timeout_output: str,
    ) -> tuple[dict[str, object], int]:
        smoke_attempts = 0

        def fake_run_probe_command(
            command: str,
            args: list[str],
            repo_root: str,
            stdin_text: str = "",
            env_extra: dict[str, str] | None = None,
            timeout_sec: int | None = None,
        ) -> subprocess.CompletedProcess[str]:
            del repo_root, stdin_text
            nonlocal smoke_attempts
            if args == ["--version"]:
                return subprocess.CompletedProcess(
                    [command, *args],
                    0,
                    "2.1.85 (Claude Code)\n",
                    "",
                )

            smoke_attempts += 1
            if smoke_attempts == 1:
                raise subprocess.TimeoutExpired(
                    [command, *args],
                    timeout_sec or 0,
                    output=timeout_output,
                    stderr="",
                )

            assert env_extra is not None
            sentinel = Path(env_extra["ACP_PREFLIGHT_SMOKE_SENTINEL"])
            sentinel.parent.mkdir(parents=True, exist_ok=True)
            sentinel.write_text(env_extra["ACP_PREFLIGHT_SMOKE_TEXT"] + "\n", encoding="utf-8")
            return subprocess.CompletedProcess([command, *args], 0, "", "")

        with mock.patch.object(self.module, "run_probe_command", side_effect=fake_run_probe_command):
            result = self.module.probe_provider_readiness("claude", "/test/claude", str(REPO_ROOT))
        return result, smoke_attempts

    def test_probe_provider_readiness_returns_not_selected(self) -> None:
        result = self.module.probe_provider_readiness("qwen", "not-selected", str(REPO_ROOT))
        self.assertEqual("not_selected", result["status"])

    def test_qwen_readiness_blocks_missing_binary(self) -> None:
        result = self.module.probe_provider_readiness(
            "qwen",
            str(self.root / "missing-qwen"),
            str(REPO_ROOT),
        )

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("missing_binary", result["subclass"])

    def test_qwen_skips_text_only_headless_probe(self) -> None:
        args, stdin_text = self.module.headless_probe_invocation("qwen")

        self.assertEqual([], args)
        self.assertEqual("", stdin_text)

    def test_codex_headless_probe_uses_runtime_isolation_args(self) -> None:
        args, stdin_text = self.module.headless_probe_invocation("codex")

        self.assertEqual("exec", args[0])
        for feature in (
            "plugins",
            "remote_plugin",
            "plugin_sharing",
            "apps",
            "enable_mcp_apps",
            "tool_suggest",
            "skill_mcp_dependency_install",
        ):
            self.assertIn("--disable", args)
            self.assertIn(feature, args)
        self.assertIn("--ignore-user-config", args)
        self.assertIn("--ignore-rules", args)
        self.assertIn("--model", args)
        self.assertIn("gpt-5.6-luna", args)
        self.assertIn("-c", args)
        self.assertIn('model_reasoning_effort="high"', args)
        self.assertIn("--ephemeral", args)
        self.assertEqual("-", args[-1])
        self.assertIn("ACP_READY", stdin_text)

    def test_codex_model_args_can_be_overridden_or_disabled(self) -> None:
        os.environ["ACP_CODEX_MODEL"] = "gpt-5.4"
        os.environ["ACP_CODEX_REASONING_EFFORT"] = "high"
        args, _ = self.module.headless_probe_invocation("codex")

        self.assertIn("--model", args)
        self.assertIn("gpt-5.4", args)
        self.assertIn('model_reasoning_effort="high"', args)

        os.environ["ACP_CODEX_MODEL"] = ""
        os.environ["ACP_CODEX_REASONING_EFFORT"] = ""
        args, _ = self.module.headless_probe_invocation("codex")

        self.assertNotIn("--model", args)
        self.assertNotIn("gpt-5.6-luna", args)
        self.assertNotIn("-c", args)

    def test_qwen_artifact_smoke_uses_runtime_write_args(self) -> None:
        sentinel_path = self.root / "write-dir" / "sentinel.txt"

        args, stdin_text = self.module.artifact_smoke_invocation("qwen", sentinel_path)

        self.assertEqual(
            [
                "--chat-recording",
                "false",
                "--yolo",
                "--channel",
                "CI",
                "--output-format",
                "stream-json",
                "--include-partial-messages",
                "-p",
            ],
            args[:9],
        )
        self.assertIn(str(sentinel_path), args[9])
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
            "if [ \"${1:-}\" = \"-p\" ]; then sleep 3; exit 42; fi\n"
            "if [ \"${1:-}\" = \"--chat-recording\" ] && [ \"${2:-}\" = \"false\" ] && [ \"${3:-}\" = \"--yolo\" ] && [ \"${4:-}\" = \"--channel\" ] && [ \"${5:-}\" = \"CI\" ] && [ \"${6:-}\" = \"--output-format\" ] && [ \"${7:-}\" = \"stream-json\" ] && [ \"${8:-}\" = \"--include-partial-messages\" ] && [ \"${9:-}\" = \"-p\" ]; then\n"
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

    def test_probe_provider_readiness_checks_qwen_artifact_smoke_for_quota_markers(self) -> None:
        command = self._write_script(
            "qwen-artifact-quota-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "printf '%s\n' 'API Error: 429 rate limit'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("unavailable", result["status"])
        self.assertEqual("quota_or_permission", result["subclass"])
        self.assertIn("rate limit", result["reason"])

    def test_probe_provider_readiness_ignores_cross_family_model_telemetry(self) -> None:
        command = self._write_script(
            "qwen-claude-model-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"; printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"; exit 0; fi\n"
            "printf '%s\n' '{\"model\":\"claude-opus-4-5-20251101\",\"status\":\"ok\"}'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])
        self.assertNotIn("model_mismatch", result)
        self.assertNotIn("observed_models", result)

    def test_probe_provider_readiness_ignores_escaped_model_telemetry(self) -> None:
        command = self._write_script(
            "qwen-escaped-claude-model-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"; printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"; exit 0; fi\n"
            "printf '%s\n' '{\\\"model\\\":\\\"claude-opus-4-5-20251101\\\",\\\"status\\\":\\\"ok\\\"}'\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])

    def test_probe_provider_readiness_ignores_nested_model_usage_telemetry(self) -> None:
        command = self._write_script(
            "claude-kimi-model-usage-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' '2.1.85 (Claude Code)'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"; printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"; exit 0; fi\n"
            "printf '%s\n' '{\"type\":\"result\",\"modelUsage\":{\"kimi-for-coding\":{\"inputTokens\":1,\"outputTokens\":1}},\"result\":\"ACP_READY\"}'\n",
        )

        result = self.module.probe_provider_readiness("claude", command, str(REPO_ROOT))
        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])
        self.assertNotIn("model_mismatch", result)
        self.assertNotIn("observed_models", result)

    def test_claude_readiness_uses_artifact_smoke_not_text_probe(self) -> None:
        command = self._write_script(
            "claude-text-probe-timeout-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' '2.1.85 (Claude Code)'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"; printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"; exit 0; fi\n"
            "sleep 2\n"
            "exit 42\n",
        )

        result = self.module.probe_provider_readiness("claude", command, str(REPO_ROOT))
        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])

    def test_claude_artifact_smoke_uses_runtime_like_add_dir(self) -> None:
        sentinel = self.root / "write-dir" / "sentinel.txt"

        args, stdin_text = self.module.artifact_smoke_invocation("claude", sentinel)

        self.assertEqual("", stdin_text)
        self.assertIn("--add-dir", args)
        add_dir_index = args.index("--add-dir")
        self.assertEqual(str(sentinel.parent), args[add_dir_index + 1])
        self.assertIn("-p", args)

    def test_claude_artifact_smoke_retries_timeout_once(self) -> None:
        result, attempts = self._probe_claude_artifact_smoke_timeout_retry("")

        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])
        self.assertEqual(2, attempts)

    def test_claude_artifact_smoke_retries_timeout_even_with_output(self) -> None:
        result, attempts = self._probe_claude_artifact_smoke_timeout_retry("transient smoke output")

        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])
        self.assertEqual(2, attempts)

    def test_claude_artifact_smoke_preserves_timeout_output_after_exhausted_retry(self) -> None:
        attempts = self.root / "claude-smoke-output-exhausted-attempts"
        command = self._write_script(
            "claude-exhausted-artifact-smoke-output-stub",
            "#!/bin/sh\n"
            f"attempts='{attempts}'\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' '2.1.85 (Claude Code)'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then\n"
            "  count=0\n"
            "  if [ -f \"$attempts\" ]; then count=$(cat \"$attempts\"); fi\n"
            "  count=$((count + 1))\n"
            "  printf '%s\\n' \"$count\" > \"$attempts\"\n"
            "  if [ \"$count\" = \"1\" ]; then printf '%s\\n' 'API Error: 403 permission_error usage limit'; fi\n"
            "  sleep 5\n"
            "  exit 0\n"
            "fi\n"
            "printf '%s\n' 'ACP_READY'\n",
        )
        old_timeout = os.environ.get("ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC")
        os.environ["ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC"] = "4"
        try:
            result = self.module.probe_provider_readiness("claude", command, str(REPO_ROOT))
        finally:
            if old_timeout is None:
                os.environ.pop("ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC", None)
            else:
                os.environ["ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC"] = old_timeout

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("quota_or_permission", result["subclass"])
        self.assertEqual("failed", result["artifact_smoke"])
        self.assertEqual("2", attempts.read_text(encoding="utf-8").strip())

    def test_claude_artifact_smoke_accepts_expected_sentinel_before_timeout(self) -> None:
        command = self._write_script(
            "claude-sentinel-before-timeout-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' '2.1.85 (Claude Code)'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then\n"
            "  mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"\n"
            "  printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"\n"
            "  sleep 3\n"
            "  exit 0\n"
            "fi\n"
            "printf '%s\n' 'ACP_READY'\n",
        )
        old_timeout = os.environ.get("ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC")
        os.environ["ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC"] = "2"
        try:
            result = self.module.probe_provider_readiness("claude", command, str(REPO_ROOT))
        finally:
            if old_timeout is None:
                os.environ.pop("ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC", None)
            else:
                os.environ["ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC"] = old_timeout

        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])
        self.assertEqual("passed", result["artifact_smoke"])
        self.assertIn("wrote expected sentinel before timeout", result["reason"])

    def test_claude_artifact_smoke_exhausted_timeout_blocks_preflight(self) -> None:
        attempts = self.root / "claude-smoke-attempts"
        command = self._write_script(
            "claude-timeout-artifact-smoke-stub",
            "#!/bin/sh\n"
            f"attempts='{attempts}'\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' '2.1.85 (Claude Code)'; exit 0; fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then\n"
            "  count=0\n"
            "  if [ -f \"$attempts\" ]; then count=$(cat \"$attempts\"); fi\n"
            "  count=$((count + 1))\n"
            "  printf '%s\\n' \"$count\" > \"$attempts\"\n"
            "  sleep 3\n"
            "  exit 0\n"
            "fi\n"
            "printf '%s\n' 'ACP_READY'\n",
        )
        old_timeout = os.environ.get("ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC")
        os.environ["ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC"] = "2"
        try:
            result = self.module.probe_provider_readiness("claude", command, str(REPO_ROOT))
        finally:
            if old_timeout is None:
                os.environ.pop("ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC", None)
            else:
                os.environ["ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC"] = old_timeout

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("operational_host_preflight_failed", result["subclass"])
        self.assertEqual("failed", result["artifact_smoke"])
        self.assertIn("attempt 2/2", result["reason"])
        self.assertIn("timed out", result["reason"])

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

    def test_probe_provider_readiness_uses_live_codex_model_for_version_blocker(self) -> None:
        os.environ["ACP_CODEX_MODEL"] = "gpt-5.4"
        command = self._write_script(
            "codex-stub",
            "#!/bin/sh\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"; printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"; exit 0; fi\n"
            "printf '%s\n' 'codex-cli 0.118.0'\n",
        )

        result = self.module.probe_provider_readiness(
            "codex",
            command,
            str(REPO_ROOT),
            "codex-cli 0.118.0",
            'model = "gpt-5.5"\n',
        )

        self.assertEqual("ready", result["status"])
        self.assertEqual("", result["subclass"])

    def test_probe_provider_readiness_uses_auth_only_codex_home_and_runtime_args(self) -> None:
        source_home = self.root / "source-codex-home"
        source_home.mkdir()
        (source_home / "auth.json").write_text("{\"token\":\"redacted\"}\n", encoding="utf-8")
        (source_home / "installation_id").write_text("install-123\n", encoding="utf-8")
        (source_home / "config.toml").write_text("model = \"gpt-5.6-luna\"\n", encoding="utf-8")
        (source_home / ".tmp").mkdir()
        (source_home / ".tmp" / "plugins").mkdir()

        command = self._write_script(
            "codex-isolated-preflight-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'codex-cli 0.140.0'; exit 0; fi\n"
            "has_plugins_disable=0\n"
            "has_ignore_config=0\n"
            "has_ignore_rules=0\n"
            "has_model=0\n"
            "has_reasoning=0\n"
            "prev=''\n"
            "for arg in \"$@\"; do\n"
            "  if [ \"$prev\" = \"--disable\" ] && [ \"$arg\" = \"plugins\" ]; then has_plugins_disable=1; fi\n"
            "  if [ \"$prev\" = \"--model\" ] && [ \"$arg\" = \"gpt-5.6-luna\" ]; then has_model=1; fi\n"
            "  if [ \"$prev\" = \"-c\" ] && [ \"$arg\" = 'model_reasoning_effort=\"high\"' ]; then has_reasoning=1; fi\n"
            "  if [ \"$arg\" = \"--ignore-user-config\" ]; then has_ignore_config=1; fi\n"
            "  if [ \"$arg\" = \"--ignore-rules\" ]; then has_ignore_rules=1; fi\n"
            "  prev=\"$arg\"\n"
            "done\n"
            "if [ \"$has_plugins_disable\" != \"1\" ] || [ \"$has_ignore_config\" != \"1\" ] || [ \"$has_ignore_rules\" != \"1\" ]; then\n"
            "  printf 'missing codex isolation args: %s\\n' \"$*\" >&2\n"
            "  exit 2\n"
            "fi\n"
            "if [ \"$has_model\" != \"1\" ] || [ \"$has_reasoning\" != \"1\" ]; then\n"
            "  printf 'missing codex model args: %s\\n' \"$*\" >&2\n"
            "  exit 2\n"
            "fi\n"
            "if [ -z \"${CODEX_HOME:-}\" ] || [ ! -f \"$CODEX_HOME/auth.json\" ] || [ ! -f \"$CODEX_HOME/installation_id\" ]; then\n"
            "  printf 'missing isolated codex auth home\\n' >&2\n"
            "  exit 3\n"
            "fi\n"
            "if [ -f \"$CODEX_HOME/config.toml\" ] || [ -e \"$CODEX_HOME/.tmp\" ] || [ -e \"$CODEX_HOME/plugins\" ]; then\n"
            "  printf 'codex preflight copied user config/plugin state\\n' >&2\n"
            "  exit 4\n"
            "fi\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ]; then\n"
            "  mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"\n"
            "  printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"\n"
            "  exit 0\n"
            "fi\n"
            "printf '%s\\n' 'ACP_READY'\n",
        )
        old_home = os.environ.get("CODEX_HOME")
        os.environ["CODEX_HOME"] = str(source_home)
        try:
            result = self.module.probe_provider_readiness(
                "codex",
                command,
                str(REPO_ROOT),
                "codex-cli 0.140.0",
                'model = "gpt-5.6-luna"\n',
            )
        finally:
            if old_home is None:
                os.environ.pop("CODEX_HOME", None)
            else:
                os.environ["CODEX_HOME"] = old_home

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

    def test_qwen_artifact_smoke_timeout_without_sentinel_is_single_attempt_blocker(self) -> None:
        attempts = self.root / "qwen-smoke-attempts"
        command = self._write_script(
            "qwen-timeout-artifact-smoke-stub",
            "#!/bin/sh\n"
            f"attempts='{attempts}'\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "count=0\n"
            "if [ -f \"$attempts\" ]; then count=$(cat \"$attempts\"); fi\n"
            "printf '%s\\n' \"$((count + 1))\" > \"$attempts\"\n"
            "sleep 3\n",
        )
        old_timeout = os.environ.get("ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC")
        os.environ["ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC"] = "1"
        try:
            result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        finally:
            if old_timeout is None:
                os.environ.pop("ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC", None)
            else:
                os.environ["ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC"] = old_timeout

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("operational_host_preflight_failed", result["subclass"])
        self.assertEqual("failed", result["artifact_smoke"])
        self.assertIn("attempt 1/1", result["reason"])
        self.assertIn("timed out", result["reason"])
        self.assertEqual("1", attempts.read_text(encoding="utf-8").strip())

    def test_qwen_artifact_smoke_rejects_invalid_sentinel(self) -> None:
        command = self._write_script(
            "qwen-invalid-artifact-smoke-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"\n"
            "printf '%s\\n' 'WRONG_SENTINEL' > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("operational_host_preflight_failed", result["subclass"])
        self.assertEqual("failed", result["artifact_smoke"])
        self.assertIn("unexpected sentinel content", result["reason"])

    def test_qwen_artifact_smoke_nonzero_exit_is_blocker(self) -> None:
        command = self._write_script(
            "qwen-nonzero-artifact-smoke-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "printf '%s\\n' 'artifact smoke command failed' >&2\n"
            "exit 17\n",
        )

        result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))

        self.assertEqual("unavailable", result["status"])
        self.assertEqual("operational_host_preflight_failed", result["subclass"])
        self.assertEqual("failed", result["artifact_smoke"])
        self.assertIn("artifact smoke command failed", result["reason"])

    def test_qwen_artifact_smoke_timeout_terminates_provider_process_group(self) -> None:
        orphan_marker = self.root / "qwen-orphan-marker"
        command = self._write_script(
            "qwen-orphan-artifact-smoke-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            f"(sleep 2; printf '%s\\n' orphan > '{orphan_marker}') &\n"
            "sleep 5\n",
        )
        old_timeout = os.environ.get("ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC")
        os.environ["ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC"] = "1"
        try:
            result = self.module.probe_provider_readiness("qwen", command, str(REPO_ROOT))
        finally:
            if old_timeout is None:
                os.environ.pop("ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC", None)
            else:
                os.environ["ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC"] = old_timeout

        time.sleep(2)
        self.assertEqual("unavailable", result["status"])
        self.assertEqual("operational_host_preflight_failed", result["subclass"])
        self.assertFalse(orphan_marker.exists(), "Qwen artifact-smoke child survived process-group cleanup")

    def test_qwen_artifact_smoke_uses_runtime_like_tool_args(self) -> None:
        command = self._write_script(
            "qwen-runtime-like-artifact-stub",
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "has_yolo=0\n"
            "has_channel_ci=0\n"
            "has_stream_json=0\n"
            "has_partial_messages=0\n"
            "prev=''\n"
            "for arg in \"$@\"; do\n"
            "  if [ \"$arg\" = \"--yolo\" ]; then has_yolo=1; fi\n"
            "  if [ \"$prev\" = \"--channel\" ] && [ \"$arg\" = \"CI\" ]; then has_channel_ci=1; fi\n"
            "  if [ \"$prev\" = \"--output-format\" ] && [ \"$arg\" = \"stream-json\" ]; then has_stream_json=1; fi\n"
            "  if [ \"$arg\" = \"--include-partial-messages\" ]; then has_partial_messages=1; fi\n"
            "  prev=\"$arg\"\n"
            "done\n"
            "if [ -n \"${ACP_PREFLIGHT_SMOKE_SENTINEL:-}\" ] && [ \"$has_yolo\" = \"1\" ] && [ \"$has_channel_ci\" = \"1\" ] && [ \"$has_stream_json\" = \"1\" ] && [ \"$has_partial_messages\" = \"1\" ]; then\n"
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
