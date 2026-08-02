#!/usr/bin/env python3
"""Verify a tracked, tag-scoped owner waiver for an unqualified prerelease."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Any

SHA_RE = re.compile(r"[0-9a-f]{40}")
TAG_RE = re.compile(r"v[0-9]+\.[0-9]+\.[0-9]+(?:[-.][A-Za-z0-9.-]+)?")
REQUIRED_WAIVERS = [
    "qwen-code live evidence",
    "claude-code live evidence",
    "composite release verdict",
]


def load(path: Path) -> dict[str, Any]:
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        raise SystemExit(f"release owner waiver not found: {path}") from None
    except (OSError, json.JSONDecodeError) as exc:
        raise SystemExit(f"invalid release owner waiver {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise SystemExit("release owner waiver must be a JSON object")
    return payload


def is_ancestor(base: str, source: str, cwd: Path) -> bool:
    completed = subprocess.run(
        ["git", "merge-base", "--is-ancestor", base, source],
        cwd=cwd,
        capture_output=True,
        text=True,
    )
    return completed.returncode == 0


def verify(payload: dict[str, Any], tag: str, source_sha: str, cwd: Path) -> list[str]:
    failures: list[str] = []
    if payload.get("schema_version") != 1:
        failures.append("schema_version must be 1")
    if payload.get("tag") != tag:
        failures.append(f"tag must be {tag!r}, got {payload.get('tag')!r}")
    if not TAG_RE.fullmatch(tag):
        failures.append(f"invalid release tag: {tag!r}")
    if payload.get("decision") != "owner_waived":
        failures.append("decision must be 'owner_waived'")
    if payload.get("release_state") != "UNQUALIFIED PRERELEASE":
        failures.append("release_state must be 'UNQUALIFIED PRERELEASE'")
    if payload.get("waived_requirements") != REQUIRED_WAIVERS:
        failures.append(f"waived_requirements must be exactly {REQUIRED_WAIVERS!r}")
    if not str(payload.get("approved_by", "")).strip():
        failures.append("approved_by must be present")
    if len(str(payload.get("reason", "")).strip()) < 20:
        failures.append("reason must contain at least 20 characters")
    base = str(payload.get("base_qualification_sha", "")).strip()
    if not SHA_RE.fullmatch(base):
        failures.append("base_qualification_sha must be a 40-character lowercase Git SHA")
    if not SHA_RE.fullmatch(source_sha):
        failures.append("source SHA must be a 40-character lowercase Git SHA")
    elif SHA_RE.fullmatch(base) and not is_ancestor(base, source_sha, cwd):
        failures.append(f"base qualification SHA {base} is not an ancestor of release SHA {source_sha}")
    return failures


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("waiver", type=Path)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--source-sha", required=True)
    args = parser.parse_args(argv)
    failures = verify(load(args.waiver), args.tag, args.source_sha, Path.cwd())
    if failures:
        for failure in failures:
            print(f"release owner waiver rejected: {failure}", file=sys.stderr)
        return 1
    print(f"release owner waiver accepted: {args.waiver} ({args.tag})")
    print("release state: UNQUALIFIED PRERELEASE")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
