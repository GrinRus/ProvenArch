#!/usr/bin/env python3
"""Verify release evidence produced by live E2E without running live E2E."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

RELEASE_PROVIDERS = ["qwen-code", "claude-code", "codex-code"]
RELEASE_RUN_INDEXES = ["1"]
MANUAL_ASSESSMENTS = {
    "ux": "swe_ux_assessment_{matrix_id}.md",
    "artifact_quality": "swe_artifact_quality_assessment_{matrix_id}.md",
}


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
    else:
        if release_contract.get("mode") != "release":
            failures.append(f"release_contract.mode must be release, got {release_contract.get('mode')!r}")
        if release_contract.get("contract_status") != "passed":
            failures.append(
                "release_contract.contract_status must be passed, "
                f"got {release_contract.get('contract_status')!r}"
            )
        if release_contract.get("selected_providers") != RELEASE_PROVIDERS:
            failures.append(
                "release_contract.selected_providers must be "
                f"{RELEASE_PROVIDERS!r}, got {release_contract.get('selected_providers')!r}"
            )
        if release_contract.get("selected_run_indexes") != RELEASE_RUN_INDEXES:
            failures.append(
                "release_contract.selected_run_indexes must be "
                f"{RELEASE_RUN_INDEXES!r}, got {release_contract.get('selected_run_indexes')!r}"
            )
    records = payload.get("records")
    if not isinstance(records, list) or not records:
        failures.append("records must be a non-empty list")
    else:
        for index, record in enumerate(records):
            if not isinstance(record, dict):
                failures.append(f"records[{index}] must be an object")
                continue
            if record.get("strict_status") != "passed":
                failures.append(f"records[{index}].strict_status must be passed, got {record.get('strict_status')!r}")
    return failures


def parse_markdown_scalar(text: str, key: str) -> str:
    pattern = rf"^\s*(?:[-*]\s*)?{re.escape(key)}\s*:\s*(.+?)\s*$"
    match = re.search(pattern, text, flags=re.IGNORECASE | re.MULTILINE)
    return match.group(1).strip() if match else ""


def matrix_id_from_verdict_path(path: Path) -> str:
    match = re.fullmatch(r"release_verdict_(.+)\.json", path.name)
    return match.group(1) if match else ""


def verify_manual_assessment(path: Path, matrix_id: str, label: str) -> list[str]:
    failures: list[str] = []
    try:
        text = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return [f"{label} assessment file not found: {path}"]
    except OSError as exc:
        return [f"failed to read {label} assessment file {path}: {exc}"]

    observed_matrix_id = parse_markdown_scalar(text, "matrix_id")
    if observed_matrix_id != matrix_id:
        failures.append(
            f"{label} assessment matrix_id must be {matrix_id!r}, got {observed_matrix_id!r}"
        )

    decision = parse_markdown_scalar(text, "decision") or parse_markdown_scalar(text, "status")
    if decision.lower() != "accepted":
        failures.append(f"{label} assessment decision/status must be accepted, got {decision!r}")
    return failures


def verify_manual_assessments(verdict_path: Path, matrix_id: str) -> list[str]:
    reports_root = verdict_path.parent
    failures: list[str] = []
    for label, template in MANUAL_ASSESSMENTS.items():
        path = reports_root / template.format(matrix_id=matrix_id)
        failures.extend(verify_manual_assessment(path, matrix_id, label))
    return failures


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("verdict_json", type=Path, help="Path to reports/release_verdict_<matrix-id>.json")
    args = parser.parse_args(argv)

    payload = load_payload(args.verdict_json)
    matrix_id = str(payload.get("matrix_id", "")).strip()
    path_matrix_id = matrix_id_from_verdict_path(args.verdict_json)
    failures = verify_payload(payload)
    if not matrix_id:
        failures.append("matrix_id must be present")
    elif path_matrix_id and path_matrix_id != matrix_id:
        failures.append(
            f"matrix_id must match verdict filename {path_matrix_id!r}, got {matrix_id!r}"
        )
    else:
        failures.extend(verify_manual_assessments(args.verdict_json, matrix_id))
    if failures:
        for failure in failures:
            print(f"release evidence not ready: {failure}", file=sys.stderr)
        return 1
    print(f"release evidence ready: {args.verdict_json}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
