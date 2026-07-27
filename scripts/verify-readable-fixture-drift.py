#!/usr/bin/env python3
"""Verify tracked readable scenario exports against machine snapshot digests."""

from __future__ import annotations

import hashlib
from pathlib import Path
import sys


def main() -> int:
    repo_root = Path(__file__).resolve().parents[1]
    scenarios_root = repo_root / "fixtures" / "scenarios"
    errors: list[str] = []
    checked = 0
    for readable_root in sorted(scenarios_root.glob("*/golden/readable")):
        snapshot_path = readable_root.parent / "snapshot.sha256"
        if not snapshot_path.is_file():
            errors.append(f"{readable_root.relative_to(repo_root)}: missing snapshot.sha256")
            continue
        snapshot: dict[str, str] = {}
        for line_number, raw_line in enumerate(snapshot_path.read_text(encoding="utf-8").splitlines(), 1):
            parts = raw_line.split(maxsplit=1)
            if len(parts) != 2 or len(parts[0]) != 64:
                errors.append(f"{snapshot_path.relative_to(repo_root)}:{line_number}: invalid digest line")
                continue
            digest, rel = parts
            if rel in snapshot:
                errors.append(f"{snapshot_path.relative_to(repo_root)}:{line_number}: duplicate path {rel}")
            snapshot[rel] = digest.lower()
        readable_paths = sorted(path for path in readable_root.rglob("*") if path.is_file())
        if not readable_paths:
            errors.append(f"{readable_root.relative_to(repo_root)}: empty readable export")
        for artifact in readable_paths:
            rel = artifact.relative_to(readable_root).as_posix()
            expected = snapshot.get(rel)
            if expected is None:
                errors.append(f"{artifact.relative_to(repo_root)}: absent from machine snapshot")
                continue
            actual = hashlib.sha256(artifact.read_bytes()).hexdigest()
            if actual != expected:
                errors.append(
                    f"{artifact.relative_to(repo_root)}: digest drift expected={expected} actual={actual}"
                )
            checked += 1
    if checked == 0:
        errors.append("no readable scenario artifacts found")
    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    print(f"readable fixture drift check passed ({checked} artifacts)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
