#!/usr/bin/env python3
"""Write batch preflight payload JSON."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path


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
    }

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
    print(f"timeout_profile_line={timeout_profile_line}")
    print(f"execution_profile_line={execution_profile_line}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
