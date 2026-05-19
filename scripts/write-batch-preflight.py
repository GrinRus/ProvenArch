#!/usr/bin/env python3
"""Write batch preflight payload JSON."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

PROVIDER_UNAVAILABLE_MARKERS = (
    "permission_error",
    "permission error",
    "usage limit",
    "quota exceeded",
    "quota",
    "rate limit",
    "rate_limit",
    "api error: 403",
    "api error: 429",
    "status code: 403",
    "status code: 429",
)

ARTIFACT_SMOKE_SENTINEL_TEXT = "ACP_ARTIFACT_SMOKE_READY"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Write batch preflight payload")
    parser.add_argument("--out", required=True, help="Output preflight JSON path")
    parser.add_argument("--generated-at-utc", required=True)
    parser.add_argument("--provenarch-root", required=True)
    parser.add_argument("--provenarch-sha", required=True)
    parser.add_argument("--provenarch-branch", required=True)
    parser.add_argument("--target-repos-file", required=True)
    parser.add_argument("--declared-repos-meta-file", required=True)
    parser.add_argument("--apply-timeouts-via-api", required=True, choices=("0", "1"))
    parser.add_argument("--sweep-id", default="baseline")
    parser.add_argument("--selected-providers", default="")
    parser.add_argument("--selected-run-indexes", default="")
    parser.add_argument("--claude-path", required=True)
    parser.add_argument("--claude-version-line", required=True)
    parser.add_argument("--qwen-path", required=True)
    parser.add_argument("--qwen-version-line", required=True)
    parser.add_argument("--codex-path", required=True)
    parser.add_argument("--codex-version-line", required=True)
    return parser.parse_args()


def resolve_profile(script_path: Path) -> tuple[dict[str, object], str]:
    try:
        json_raw = subprocess.check_output(
            [sys.executable, str(script_path), "--format", "json"],
            text=True,
        ).strip()
        line_raw = subprocess.check_output(
            [sys.executable, str(script_path), "--format", "line"],
            text=True,
        ).strip()
    except subprocess.CalledProcessError as exc:
        raise SystemExit(f"failed to resolve profile via {script_path}: {exc}") from exc
    return json.loads(json_raw), line_raw


def parse_codex_model_from_config(config_text: str) -> str:
    for raw_line in (config_text or "").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        match = re.match(r'^model\s*=\s*["\']([^"\']+)["\']\s*$', line)
        if match:
            return match.group(1).strip()
    return ""


def read_codex_config_text() -> str:
    codex_home = os.environ.get("CODEX_HOME", "").strip()
    if codex_home:
        path = Path(codex_home).expanduser() / "config.toml"
    else:
        path = Path.home() / ".codex" / "config.toml"
    try:
        return path.read_text(encoding="utf-8")
    except OSError:
        return ""


def parse_semver_tuple(version_line: str) -> tuple[int, int, int] | None:
    match = re.search(r"(\d+)\.(\d+)\.(\d+)", version_line or "")
    if not match:
        return None
    return tuple(int(part) for part in match.groups())


def codex_model_version_blocker(version_line: str, config_text: str | None = None) -> str:
    model = parse_codex_model_from_config(config_text if config_text is not None else read_codex_config_text())
    if not model.startswith("gpt-5.5"):
        return ""
    parsed = parse_semver_tuple(version_line)
    if parsed is None:
        return f"codex model {model} is configured, but codex version could not be parsed: {version_line}"
    if parsed < (0, 124, 0):
        return (
            f"codex model {model} requires a newer Codex CLI than {version_line}; "
            "upgrade codex or set ACP_CODEX_CMD_BIN to a compatible binary"
        )
    return ""


def probe_timeout_sec() -> int:
    raw = os.environ.get("ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC", "").strip()
    if not raw:
        return 30
    try:
        value = int(raw)
    except ValueError:
        return 30
    return value if value > 0 else 30


def artifact_smoke_timeout_sec() -> int:
    raw = os.environ.get("ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC", "").strip()
    if not raw:
        return 120
    try:
        value = int(raw)
    except ValueError:
        return 120
    return value if value > 0 else 120


def headless_probe_invocation(provider: str) -> tuple[list[str], str]:
    prompt = "ACP live preflight: reply with exactly ACP_READY. Do not write files."
    if provider == "qwen":
        return ["-p", prompt], ""
    if provider == "claude":
        return [], ""
    if provider == "codex":
        return [
            "exec",
            "--json",
            "--color",
            "never",
            "--skip-git-repo-check",
            "--sandbox",
            "danger-full-access",
            "--ephemeral",
            "-",
        ], prompt
    return [], ""


def artifact_smoke_invocation(provider: str, sentinel_path: Path) -> tuple[list[str], str]:
    prompt = (
        "ACP live artifact smoke: create parent directories if needed, write exactly "
        f"{ARTIFACT_SMOKE_SENTINEL_TEXT} to this file, then exit: {sentinel_path}"
    )
    if provider == "qwen":
        return [
            "--chat-recording",
            "false",
            "--yolo",
            "--channel",
            "CI",
            "--output-format",
            "stream-json",
            "--include-partial-messages",
            "-p",
            prompt,
        ], ""
    if provider == "claude":
        return [
            "--output-format",
            "json",
            "--permission-mode",
            "bypassPermissions",
            "--add-dir",
            str(sentinel_path.parent),
            "-p",
            prompt,
        ], ""
    if provider == "codex":
        return [
            "exec",
            "--json",
            "--color",
            "never",
            "--skip-git-repo-check",
            "--sandbox",
            "danger-full-access",
            "--ephemeral",
            "-",
        ], prompt
    return [], ""


def run_probe_command(
    command: str,
    args: list[str],
    repo_root: str,
    stdin_text: str = "",
    env_extra: dict[str, str] | None = None,
    timeout_sec: int | None = None,
) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    if env_extra:
        env.update(env_extra)
    return subprocess.run(
        [command, *args],
        cwd=repo_root or None,
        env=env,
        input=stdin_text if stdin_text else None,
        text=True,
        capture_output=True,
        timeout=timeout_sec if timeout_sec is not None else probe_timeout_sec(),
        check=False,
    )


def run_artifact_smoke(provider: str, command: str, repo_root: str) -> tuple[bool, str, str]:
    with tempfile.TemporaryDirectory(prefix="acp-provider-smoke-") as tempdir:
        sentinel_path = Path(tempdir) / "write-dir" / "sentinel.txt"
        smoke_args, smoke_stdin = artifact_smoke_invocation(provider, sentinel_path)
        if not smoke_args:
            return True, "", ""
        max_attempts = 2 if provider == "claude" else 1
        last_reason = ""
        last_combined = ""
        for attempt in range(1, max_attempts + 1):
            timeout_sec = probe_timeout_sec() if provider == "claude" else artifact_smoke_timeout_sec()
            try:
                completed = run_probe_command(
                    command,
                    smoke_args,
                    repo_root,
                    smoke_stdin,
                    {
                        "ACP_PREFLIGHT_SMOKE_SENTINEL": str(sentinel_path),
                        "ACP_PREFLIGHT_SMOKE_TEXT": ARTIFACT_SMOKE_SENTINEL_TEXT,
                    },
                    timeout_sec=timeout_sec,
                )
            except subprocess.TimeoutExpired as exc:
                stdout = exc.stdout.decode("utf-8", errors="replace") if isinstance(exc.stdout, bytes) else (exc.stdout or "")
                stderr = exc.stderr.decode("utf-8", errors="replace") if isinstance(exc.stderr, bytes) else (exc.stderr or "")
                combined = "\n".join(part for part in [stdout, stderr] if part).strip()
                last_combined = combined
                if provider == "claude":
                    try:
                        observed = sentinel_path.read_text(encoding="utf-8").strip()
                    except OSError:
                        observed = ""
                    if observed == ARTIFACT_SMOKE_SENTINEL_TEXT:
                        timeout_reason = (
                            f"{provider} artifact smoke wrote expected sentinel before timeout "
                            f"(attempt {attempt}/{max_attempts}, timeout={exc.timeout}s)"
                        )
                        return True, timeout_reason, "\n".join(
                            part for part in [combined, timeout_reason] if part
                        ).strip()
                last_reason = (
                    f"{provider} artifact smoke timed out after {exc.timeout}s "
                    f"(attempt {attempt}/{max_attempts})"
                )
                if provider == "claude" and not combined and attempt < max_attempts:
                    continue
                return False, last_reason, combined
            except Exception as exc:  # pragma: no cover - defensive shell failure path
                return False, f"{provider} artifact smoke failed: {exc}", ""
            combined = "\n".join(part for part in [completed.stdout, completed.stderr] if part).strip()
            last_combined = combined
            if completed.returncode != 0:
                last_reason = (
                    combined
                    or f"{provider} artifact smoke exited with code {completed.returncode} "
                    f"(attempt {attempt}/{max_attempts})"
                )
                if provider == "claude" and not combined and attempt < max_attempts:
                    continue
                return False, last_reason, combined
            try:
                observed = sentinel_path.read_text(encoding="utf-8").strip()
            except OSError as exc:
                last_reason = f"{provider} artifact smoke did not create sentinel: {exc} (attempt {attempt}/{max_attempts})"
                if provider == "claude" and not combined and attempt < max_attempts:
                    continue
                return False, last_reason, combined
            if observed != ARTIFACT_SMOKE_SENTINEL_TEXT:
                return False, f"{provider} artifact smoke wrote unexpected sentinel content", combined
            return True, combined, combined
        return False, last_reason, last_combined


def probe_provider_readiness(
    provider: str,
    command: str,
    repo_root: str,
    version_line: str = "",
    codex_config_text: str | None = None,
) -> dict[str, object]:
    command = (command or "").strip()
    if command in {"", "not-selected"}:
        return {
            "provider": provider,
            "status": "not_selected",
            "subclass": "",
            "reason": "",
        }

    try:
        completed = run_probe_command(command, ["--version"], repo_root)
    except FileNotFoundError:
        return {
            "provider": provider,
            "status": "unavailable",
            "subclass": "missing_binary",
            "reason": f"command not found: {command}",
        }
    except Exception as exc:  # pragma: no cover - defensive shell failure path
        return {
            "provider": provider,
            "status": "unavailable",
            "subclass": "probe_failed",
            "reason": str(exc),
        }

    combined = "\n".join(part for part in [version_line, completed.stdout, completed.stderr] if part).strip()
    normalized = combined.lower()
    if any(marker in normalized for marker in PROVIDER_UNAVAILABLE_MARKERS):
        return {
            "provider": provider,
            "status": "unavailable",
            "subclass": "quota_or_permission",
            "reason": combined or f"{provider} probe reported quota or permission failure",
        }
    if completed.returncode != 0:
        return {
            "provider": provider,
            "status": "unavailable",
            "subclass": "command_failed",
            "reason": combined or f"{provider} probe exited with code {completed.returncode}",
        }
    if provider == "codex":
        blocker = codex_model_version_blocker(combined, codex_config_text)
        if blocker:
            return {
                "provider": provider,
                "status": "unavailable",
                "subclass": "codex_model_requires_newer_cli",
                "reason": blocker,
            }
    probe_args, probe_stdin = headless_probe_invocation(provider)
    if probe_args:
        try:
            headless_completed = run_probe_command(command, probe_args, repo_root, probe_stdin)
        except subprocess.TimeoutExpired as exc:
            return {
                "provider": provider,
                "status": "unavailable",
                "subclass": "headless_probe_timeout",
                "reason": f"{provider} headless probe timed out after {exc.timeout}s",
            }
        except Exception as exc:  # pragma: no cover - defensive shell failure path
            return {
                "provider": provider,
                "status": "unavailable",
                "subclass": "headless_probe_failed",
                "reason": str(exc),
            }
        headless_combined = "\n".join(
            part for part in [headless_completed.stdout, headless_completed.stderr] if part
        ).strip()
        if headless_combined:
            combined = "\n".join(part for part in [combined, headless_combined] if part).strip()
        normalized = combined.lower()
        if any(marker in normalized for marker in PROVIDER_UNAVAILABLE_MARKERS):
            return {
                "provider": provider,
                "status": "unavailable",
                "subclass": "quota_or_permission",
                "reason": combined or f"{provider} headless probe reported quota or permission failure",
            }
        if headless_completed.returncode != 0:
            return {
                "provider": provider,
                "status": "unavailable",
                "subclass": "headless_probe_failed",
                "reason": combined or f"{provider} headless probe exited with code {headless_completed.returncode}",
            }
    smoke_ok, smoke_reason, smoke_output = run_artifact_smoke(provider, command, repo_root)
    if smoke_output:
        combined = "\n".join(part for part in [combined, smoke_output] if part).strip()
    if smoke_reason:
        normalized_smoke = "\n".join(part for part in [smoke_reason, smoke_output] if part).lower()
        if any(marker in normalized_smoke for marker in PROVIDER_UNAVAILABLE_MARKERS):
            return {
                "provider": provider,
                "status": "unavailable",
                "subclass": "quota_or_permission",
                "reason": combined or smoke_reason,
                "artifact_smoke": "failed",
            }
    if not smoke_ok:
        return {
            "provider": provider,
            "status": "unavailable",
            "subclass": "operational_host_preflight_failed",
            "reason": smoke_reason,
            "artifact_smoke": "failed",
        }
    return {
        "provider": provider,
        "status": "ready",
        "subclass": "",
        "reason": combined,
        "artifact_smoke": "passed",
    }


def selected_readiness_keys(selected_providers: list[str]) -> list[str]:
    if not selected_providers:
        return ["claude", "qwen", "codex"]
    key_by_provider = {
        "claude-code": "claude",
        "qwen-code": "qwen",
        "codex-code": "codex",
    }
    keys: list[str] = []
    for provider in selected_providers:
        key = key_by_provider.get(provider)
        if key and key not in keys:
            keys.append(key)
    return keys


def main() -> int:
    args = parse_args()
    out_path = Path(args.out).resolve()
    scripts_dir = Path(__file__).resolve().parent
    timeout_profile_script = scripts_dir / "resolve-timeout-profile.py"
    execution_profile_script = scripts_dir / "resolve-execution-profile.py"
    if not timeout_profile_script.is_file():
        raise SystemExit(f"timeout profile resolver is missing: {timeout_profile_script}")
    if not execution_profile_script.is_file():
        raise SystemExit(f"execution profile resolver is missing: {execution_profile_script}")

    declared_repos_meta_path = Path(args.declared_repos_meta_file).resolve()
    if not declared_repos_meta_path.is_file():
        raise SystemExit(f"declared repos metadata file does not exist: {declared_repos_meta_path}")
    declared_repos_meta = json.loads(declared_repos_meta_path.read_text(encoding="utf-8"))
    timeout_profile, timeout_profile_line = resolve_profile(timeout_profile_script)
    execution_profile, execution_profile_line = resolve_profile(execution_profile_script)
    sweep_id = (args.sweep_id or "").strip() or "baseline"
    selected_providers = [item.strip() for item in (args.selected_providers or "").split(",") if item.strip()]
    selected_run_indexes = [item.strip() for item in (args.selected_run_indexes or "").split(",") if item.strip()]

    payload = {
        "generated_at_utc": args.generated_at_utc,
        "provenarch_root": args.provenarch_root,
        "provenarch_sha": args.provenarch_sha,
        "provenarch_branch": args.provenarch_branch,
        "target_repos_file": args.target_repos_file,
        "declared_repos_meta": declared_repos_meta,
        "apply_timeouts_via_api": args.apply_timeouts_via_api == "1",
        "timeout_profile": timeout_profile,
        "execution_profile": execution_profile,
        "sweep_id": sweep_id,
        "selected_providers": selected_providers,
        "selected_run_indexes": selected_run_indexes,
        "runtimes": {
            "claude": {
                "path": args.claude_path,
                "version_line": args.claude_version_line,
            },
            "qwen": {
                "path": args.qwen_path,
                "version_line": args.qwen_version_line,
            },
            "codex": {
                "path": args.codex_path,
                "version_line": args.codex_version_line,
            },
        },
    }
    provider_readiness = {
        "claude": probe_provider_readiness("claude", args.claude_path, args.provenarch_root, args.claude_version_line),
        "qwen": probe_provider_readiness("qwen", args.qwen_path, args.provenarch_root, args.qwen_version_line),
        "codex": probe_provider_readiness("codex", args.codex_path, args.provenarch_root, args.codex_version_line),
    }
    payload["provider_readiness"] = provider_readiness

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
    print(f"timeout_profile_line={timeout_profile_line}")
    print(f"execution_profile_line={execution_profile_line}")
    selected_keys = selected_readiness_keys(selected_providers)
    unavailable = [
        provider_readiness[key] for key in selected_keys
        if key in provider_readiness
        and provider_readiness[key].get("status") == "unavailable"
    ]
    if unavailable:
        reason = "; ".join(f"{item.get('provider')}: {item.get('subclass')}: {item.get('reason')}" for item in unavailable)
        print("provider_readiness_status=unavailable")
        print(f"provider_readiness_reason={reason}")
    else:
        print("provider_readiness_status=ready")
        print("provider_readiness_reason=")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
