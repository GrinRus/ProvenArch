#!/usr/bin/env python3
"""Write batch preflight payload JSON."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path


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


def probe_provider_readiness(
    provider: str,
    command: str,
    repo_root: str,
    version_line: str = "",
    codex_config_text: str | None = None,
) -> dict[str, str]:
    command = (command or "").strip()
    if command in {"", "not-selected"}:
        return {
            "provider": provider,
            "status": "not_selected",
            "subclass": "",
            "reason": "",
        }

    env = os.environ.copy()
    try:
        completed = subprocess.run(
            [command, "--version"],
            cwd=repo_root or None,
            env=env,
            text=True,
            capture_output=True,
            timeout=15,
            check=False,
        )
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
    quota_markers = (
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
    if any(marker in normalized for marker in quota_markers):
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
    return {
        "provider": provider,
        "status": "ready",
        "subclass": "",
        "reason": combined,
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
