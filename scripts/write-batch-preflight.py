#!/usr/bin/env python3
"""Write batch preflight payload JSON."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path

PROVIDER_AVAILABILITY_PATTERN = re.compile(
    r"(?is)(api\s*error\s*:\s*403|permission_error|insufficient_quota|usage\s+limit|quota(?:\s+will\s+be\s+refreshed|\s+exceeded|\s+limit)|forbidden)"
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Write full-run-batch-5x2 preflight payload")
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


def compact_reason(text: str) -> str:
    cleaned = " ".join((text or "").split())
    if not cleaned:
        return ""
    if len(cleaned) <= 280:
        return cleaned
    return cleaned[:277] + "..."


def readiness_prompt(provider: str) -> str:
    return (
        "Return exactly one TaskResult JSON object and nothing else. "
        f"This is a non-mutating readiness probe for provider {provider}. "
        'Use this exact payload: {"meta":{"task_id":"preflight","step_id":"preflight","runtime":{"name":"probe","version":"probe"},"started_at":"2026-01-01T00:00:00Z"},"summary":"ok","changeset":[]}'
    )


def probe_provider_readiness(provider: str, command_path: str, provenarch_root: str) -> dict[str, str]:
    command_path = (command_path or "").strip()
    if command_path in {"", "not-selected"}:
        return {"status": "not_selected", "subclass": "", "reason": ""}

    if provider == "qwen":
        command = [
            command_path,
            "--output-format",
            "json",
            "--chat-recording",
            "false",
            "--yolo",
            "--channel",
            "CI",
            "--include-directories",
            provenarch_root,
            "--prompt",
            readiness_prompt("qwen-code"),
        ]
    elif provider == "claude":
        command = [
            command_path,
            "--output-format",
            "json",
            "--permission-mode",
            "bypassPermissions",
            "--add-dir",
            provenarch_root,
            "-p",
            readiness_prompt("claude-code"),
        ]
    else:
        return {"status": "not_selected", "subclass": "", "reason": ""}

    try:
        with tempfile.TemporaryDirectory(prefix=f"provenarch-preflight-{provider}-") as tmpdir:
            completed = subprocess.run(
                command,
                cwd=tmpdir,
                capture_output=True,
                text=True,
                timeout=30,
                check=False,
            )
    except subprocess.TimeoutExpired:
        return {"status": "indeterminate", "subclass": "probe_timeout", "reason": "provider readiness probe timed out"}
    except Exception as exc:
        return {"status": "indeterminate", "subclass": "probe_failed", "reason": compact_reason(str(exc))}

    combined = "\n".join(part for part in [completed.stdout, completed.stderr] if part).strip()
    match = PROVIDER_AVAILABILITY_PATTERN.search(combined or "")
    if match:
        return {
            "status": "unavailable",
            "subclass": "quota_or_permission",
            "reason": compact_reason(combined or match.group(0)),
        }
    if completed.returncode == 0:
        return {"status": "ready", "subclass": "", "reason": ""}
    return {
        "status": "indeterminate",
        "subclass": "probe_failed",
        "reason": compact_reason(combined or f"probe exited with code {completed.returncode}"),
    }


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
    provider_readiness = {
        "claude": probe_provider_readiness("claude", args.claude_path, args.provenarch_root),
        "qwen": probe_provider_readiness("qwen", args.qwen_path, args.provenarch_root),
    }

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
        },
        "provider_readiness": provider_readiness,
    }

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
    print(f"timeout_profile_line={timeout_profile_line}")
    print(f"execution_profile_line={execution_profile_line}")
    print(f"provider_readiness_claude_status={provider_readiness['claude']['status']}")
    print(f"provider_readiness_claude_subclass={provider_readiness['claude']['subclass']}")
    print(f"provider_readiness_qwen_status={provider_readiness['qwen']['status']}")
    print(f"provider_readiness_qwen_subclass={provider_readiness['qwen']['subclass']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
