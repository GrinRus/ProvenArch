#!/usr/bin/env python3
"""Verify an existing release_verdict_<matrix-id>.json without running live E2E."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


def load_payload(path: Path) -> dict[str, Any]:
    try:
        raw = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        raise SystemExit(f"release verdict file not found: {path}") from None
    except OSError as exc:
        raise SystemExit(f"failed to read release verdict file {path}: {exc}") from exc
    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid release verdict JSON {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise SystemExit(f"invalid release verdict JSON {path}: top-level payload must be an object")
    return payload


def verify_payload(payload: dict[str, Any]) -> list[str]:
    failures: list[str] = []
    if payload.get("verdict") != "PASS":
        failures.append(f"verdict must be PASS, got {payload.get('verdict')!r}")
    if payload.get("release_state") != "RELEASE READY":
        failures.append(f"release_state must be RELEASE READY, got {payload.get('release_state')!r}")
    release_contract = payload.get("release_contract")
    if not isinstance(release_contract, dict):
        failures.append("release_contract must be an object")
    elif release_contract.get("contract_status") != "passed":
        failures.append(
            "release_contract.contract_status must be passed, "
            f"got {release_contract.get('contract_status')!r}"
        )
    return failures


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("verdict_json", type=Path, help="Path to reports/release_verdict_<matrix-id>.json")
    args = parser.parse_args(argv)

    payload = load_payload(args.verdict_json)
    failures = verify_payload(payload)
    if failures:
        for failure in failures:
            print(f"release verdict not ready: {failure}", file=sys.stderr)
        return 1
    print(f"release verdict ready: {args.verdict_json}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
