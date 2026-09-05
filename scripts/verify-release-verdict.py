#!/usr/bin/env python3
"""Verify complete, fresh release evidence produced by the deterministic matrix harness."""

from __future__ import annotations

import argparse
import csv
import hashlib
import json
import math
import re
import subprocess
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

RELEASE_PROVIDERS = ["qwen-code", "claude-code", "codex-code"]
RELEASE_RUN_INDEXES = ["1"]
REQUIRED_SWEEPS = ["baseline", "parallel-default"]
REQUIRED_EXECUTION_BY_SWEEP = {
    "baseline": {
        "strategy": "sequential",
        "max_parallel_tasks": "1",
        "failure_policy": "best_effort",
        "shard_discovery_mode": "heuristics",
    },
    "parallel-default": {
        "strategy": "parallel",
        "max_parallel_tasks": "4",
        "failure_policy": "best_effort",
        "shard_discovery_mode": "heuristics",
    },
}
ALLOWED_PROFILES = ["single-path", "single-git_url", "multi-path", "multi-git_url"]
MANUAL_ASSESSMENTS = {
    "ux": "swe_ux_assessment_{matrix_id}.md",
    "artifact_quality": "swe_artifact_quality_assessment_{matrix_id}.md",
}
MATRIX_ID_PATTERN = re.compile(r"[A-Za-z0-9._-]+")
SHA_RE = re.compile(r"[0-9a-f]{40}")
TAG_RE = re.compile(r"v[0-9]+\.[0-9]+\.[0-9]+(?:[-.][A-Za-z0-9.-]+)?")
EVIDENCE_SCHEMA_VERSION = 2
DEFAULT_MAX_AGE_HOURS = 168.0
VALID_PASS_RATINGS = {"excellent", "good", "fair", "needs review"}
RELEASE_ALLOWED_NONBLOCKING_ISSUES = {
    "analysis:overview",
    "analysis:findings",
    "analysis:coverage",
    "analysis:questions",
    "execution:repair-heavy",
    "execution:repair-exhausted",
    "execution:stall-pressure",
}
REQUIRED_ZERO_ARTIFACT_FIELDS = (
    "runtime_contract_failed",
    "runner_unavailable",
    "runtime_timeout",
    "infra_signal_terminated",
    "infra_incomplete_cycle",
    "quality_gates_failed",
    "artifact_quality_failed",
    "summary_missing",
    "precheck_failed",
    "runtime_flow_failed",
    "cancellation_like",
    "semantic_hard_fail",
    "off_topic_hits",
    "artifact_quality_findings",
    "provider_budget_exhausted",
    "partial_failure_count",
)
REQUIRED_NONE_ARTIFACT_FIELDS = ("failure_class",)
REQUIRED_ZERO_BACKEND_FIELDS = (
    "artifact_non_snapshot_runs",
    "runtime_contract_failed_failures",
    "runner_unavailable_failures",
    "runtime_timeout_failures",
    "infra_signal_terminated_failures",
    "infra_incomplete_cycle_failures",
    "quality_gates_failed_failures",
    "artifact_quality_failed_failures",
    "summary_missing_failures",
    "precheck_failed_failures",
    "runtime_flow_failed_runs",
    "runtime_flow_issue_hits",
    "partial_failure_count",
)
TOP_LEVEL_FIELDS = {
    "matrix_id",
    "generated_at_utc",
    "evidence_schema_version",
    "source_sha",
    "source_tree_clean",
    "generator",
    "verdict",
    "release_state",
    "profile_sweep_runs",
    "strict_pass_runs",
    "strict_fail_runs",
    "backend",
    "excellent_blockers",
    "excellent_blockers_by_step",
    "release_contract",
    "records",
    "evidence_artifacts",
}


def load_payload(path: Path) -> dict[str, Any]:
    try:
        raw = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        raise SystemExit(f"release verdict file not found: {path}") from None
    except OSError as exc:
        raise SystemExit(f"failed to read release verdict file {path}: {exc}") from exc
    except UnicodeDecodeError as exc:
        raise SystemExit(f"release verdict file is not valid UTF-8 {path}: {exc}") from exc
    try:
        payload = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"invalid release verdict JSON {path}: {exc}") from exc
    if not isinstance(payload, dict):
        raise SystemExit(f"invalid release verdict JSON {path}: top-level payload must be an object")
    return payload


def parse_utc(value: object, field: str) -> tuple[datetime | None, str | None]:
    text = str(value or "").strip()
    if not text:
        return None, f"{field} must be present"
    try:
        parsed = datetime.fromisoformat(text.replace("Z", "+00:00"))
    except ValueError:
        return None, f"{field} must be an RFC3339 timestamp"
    if parsed.tzinfo is None:
        return None, f"{field} must include a timezone"
    return parsed.astimezone(timezone.utc), None


def git_commit_exists(sha: str, cwd: Path) -> bool:
    completed = subprocess.run(
        ["git", "cat-file", "-e", f"{sha}^{{commit}}"],
        cwd=cwd,
        capture_output=True,
        text=True,
    )
    return completed.returncode == 0


def git_tag_commit(tag: str, cwd: Path) -> str:
    completed = subprocess.run(
        ["git", "rev-parse", "--verify", f"refs/tags/{tag}^{{commit}}"],
        cwd=cwd,
        capture_output=True,
        text=True,
    )
    return completed.stdout.strip() if completed.returncode == 0 else ""


def git_is_ancestor(ancestor: str, descendant: str, cwd: Path) -> bool:
    completed = subprocess.run(
        ["git", "merge-base", "--is-ancestor", ancestor, descendant],
        cwd=cwd,
        capture_output=True,
        text=True,
    )
    return completed.returncode == 0


def git_changed_paths(ancestor: str, descendant: str, cwd: Path) -> list[str] | None:
    completed = subprocess.run(
        ["git", "diff", "--name-only", f"{ancestor}..{descendant}"],
        cwd=cwd,
        capture_output=True,
        text=True,
    )
    if completed.returncode != 0:
        return None
    return [line.strip() for line in completed.stdout.splitlines() if line.strip()]


def verify_source_binding(
    payload: dict[str, Any],
    expected_source_sha: str | None,
    expected_tag: str | None,
    cwd: Path,
) -> list[str]:
    failures: list[str] = []
    source_sha = str(payload.get("source_sha", "")).strip()
    if not SHA_RE.fullmatch(source_sha):
        failures.append("source_sha must be a 40-character lowercase Git SHA")
    elif not git_commit_exists(source_sha, cwd):
        failures.append(f"source_sha is not a commit in the checked-out repository: {source_sha}")

    if expected_source_sha is not None:
        if not SHA_RE.fullmatch(expected_source_sha):
            failures.append("expected source SHA must be a 40-character lowercase Git SHA")
        elif not git_commit_exists(expected_source_sha, cwd):
            failures.append(f"expected source SHA is not a commit in the checked-out repository: {expected_source_sha}")
        else:
            if SHA_RE.fullmatch(source_sha) and git_commit_exists(source_sha, cwd) and not git_is_ancestor(source_sha, expected_source_sha, cwd):
                failures.append(
                    f"source_sha {source_sha} must be an ancestor of release source SHA {expected_source_sha}"
                )

    if expected_tag is not None:
        if not TAG_RE.fullmatch(expected_tag):
            failures.append(f"invalid release tag: {expected_tag!r}")
        else:
            tag_sha = git_tag_commit(expected_tag, cwd)
            if not tag_sha:
                failures.append(f"release tag is not present in the checked-out repository: {expected_tag}")
            elif expected_source_sha is not None and tag_sha != expected_source_sha:
                failures.append(
                    f"release tag {expected_tag} resolves to {tag_sha}, expected {expected_source_sha}"
                )
            elif source_sha and SHA_RE.fullmatch(source_sha) and git_commit_exists(source_sha, cwd) and not git_is_ancestor(source_sha, tag_sha, cwd):
                failures.append(f"source_sha is not an ancestor of release tag {expected_tag}: {tag_sha}")
            elif source_sha and SHA_RE.fullmatch(source_sha) and git_commit_exists(source_sha, cwd):
                changed_paths = git_changed_paths(source_sha, tag_sha, cwd)
                if changed_paths is None:
                    failures.append("unable to inspect commits between qualification source and release tag")
                else:
                    non_evidence_paths = [path for path in changed_paths if not path.startswith("reports/")]
                    if non_evidence_paths:
                        failures.append(
                            "release tag contains non-evidence changes after qualification source: "
                            + ", ".join(non_evidence_paths)
                        )
    return failures


def _table_cells(line: str) -> list[str]:
    if not line.lstrip().startswith("|"):
        return []
    return [cell.strip() for cell in line.strip().strip("|").split("|")]


def _is_markdown_separator(cells: list[str]) -> bool:
    return bool(cells) and all(re.fullmatch(r":?-{3,}:?", cell) for cell in cells)


def _release_blocking_issues(value: object) -> list[str]:
    """Return every non-empty issue not explicitly allowed for release evidence."""
    raw = str(value or "").strip()
    if not raw or raw in {"-", "none"}:
        return []
    issues = [item.strip() for item in raw.split(",") if item.strip()]
    return [issue for issue in issues if issue not in RELEASE_ALLOWED_NONBLOCKING_ISSUES]


def _read_artifact_text(path: Path, label: str, failures: list[str]) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except FileNotFoundError:
        failures.append(f"{label} artifact is missing: {path}")
    except OSError as exc:
        failures.append(f"failed to read {label} artifact {path}: {exc}")
    except UnicodeDecodeError as exc:
        failures.append(f"{label} artifact is not valid UTF-8 {path}: {exc}")
    return ""


def verify_record_artifact_content(
    record: dict[str, Any],
    artifact_paths: dict[str, Path],
    index: int,
    source_sha: str,
) -> list[str]:
    """Reconcile materialized evidence contents with the claims in one verdict record."""
    failures: list[str] = []
    prefix = f"records[{index}]"
    expected_pairs = {(provider, run_index) for provider in RELEASE_PROVIDERS for run_index in RELEASE_RUN_INDEXES}
    expected_binding = {
        "profile_id": str(record.get("profile_id", "")).strip(),
        "sweep_id": str(record.get("sweep_id", "")).strip(),
        "batch_id": str(record.get("batch_id", "")).strip(),
    }
    for artifact_key, artifact_path in artifact_paths.items():
        text = _read_artifact_text(artifact_path, f"{prefix}.artifacts.{artifact_key}", failures)
        for binding_key, binding_value in expected_binding.items():
            marker_pattern = rf"^# acp_record_{re.escape(binding_key)}:\s*(.*)$"
            observed_markers = re.findall(marker_pattern, text, flags=re.MULTILINE)
            if observed_markers != [binding_value]:
                failures.append(f"{prefix}.artifacts.{artifact_key} is not bound to record {binding_key}")

    tsv_path = artifact_paths.get("run_matrix_tsv")
    if tsv_path is not None and tsv_path.is_file():
        try:
            with tsv_path.open("r", encoding="utf-8", newline="") as handle:
                reader = csv.DictReader(handle, delimiter="\t")
                raw_fieldnames = reader.fieldnames or []
                if len(raw_fieldnames) != len(set(raw_fieldnames)):
                    failures.append(f"{prefix}.artifacts.run_matrix_tsv contains duplicate header columns")
                fieldnames = set(raw_fieldnames)
                required = {
                    "provider",
                    "run",
                    "hard_pass",
                    "runtime_contract_status",
                    "artifact_quality_status",
                    "verdict",
                    "artifact_source",
                    *REQUIRED_NONE_ARTIFACT_FIELDS,
                    "issues",
                    "effective_verdict_source",
                    "promotion_audit_result",
                    *REQUIRED_ZERO_ARTIFACT_FIELDS,
                }
                missing = sorted(required - fieldnames)
                if missing:
                    failures.append(f"{prefix}.artifacts.run_matrix_tsv is missing columns: {', '.join(missing)}")
                rows = list(reader)
        except (OSError, UnicodeDecodeError, csv.Error) as exc:
            failures.append(f"failed to parse {prefix}.artifacts.run_matrix_tsv: {exc}")
            rows = []
        data_rows = [
            row for row in rows if not str(row.get("provider", "")).strip().startswith("# acp_record_")
        ]
        observed_pairs: set[tuple[str, str]] = set()
        hard_pass = 0
        for row_index, row in enumerate(data_rows):
            provider = str(row.get("provider", "")).strip()
            run_index = str(row.get("run", "")).strip()
            pair = (provider, run_index)
            if pair in observed_pairs:
                failures.append(f"{prefix}.artifacts.run_matrix_tsv contains duplicate provider/run: {provider}/{run_index}")
            observed_pairs.add(pair)
            if pair not in expected_pairs:
                failures.append(f"{prefix}.artifacts.run_matrix_tsv has unexpected provider/run: {provider}/{run_index}")
            if str(row.get("hard_pass", "")).strip() not in {"1", "true", "True"}:
                failures.append(f"{prefix}.artifacts.run_matrix_tsv row {row_index} hard_pass must be 1")
            else:
                hard_pass += 1
            if str(row.get("runtime_contract_status", "")).strip().lower() != "passed":
                failures.append(f"{prefix}.artifacts.run_matrix_tsv row {row_index} runtime_contract_status must be passed")
            artifact_quality_status = str(row.get("artifact_quality_status", "")).strip().lower()
            if artifact_quality_status not in {"passed", "needs_review"}:
                failures.append(
                    f"{prefix}.artifacts.run_matrix_tsv row {row_index} artifact_quality_status must be passed or needs_review"
                )
            if str(row.get("verdict", "")).strip().lower() not in VALID_PASS_RATINGS:
                failures.append(f"{prefix}.artifacts.run_matrix_tsv row {row_index} verdict must be a passing rating")
            if str(row.get("artifact_source", "")).strip().lower() != "snapshot":
                failures.append(f"{prefix}.artifacts.run_matrix_tsv row {row_index} artifact_source must be snapshot")
            blocking_issues = _release_blocking_issues(row.get("issues"))
            if blocking_issues:
                failures.append(
                    f"{prefix}.artifacts.run_matrix_tsv row {row_index} contains release-blocking issues: "
                    + ", ".join(blocking_issues)
                )
            for field in REQUIRED_ZERO_ARTIFACT_FIELDS:
                if str(row.get(field, "")).strip() not in {"0", "false", "False"}:
                    failures.append(f"{prefix}.artifacts.run_matrix_tsv row {row_index} {field} must be 0")
            for field in REQUIRED_NONE_ARTIFACT_FIELDS:
                if str(row.get(field, "")).strip().lower() != "none":
                    failures.append(f"{prefix}.artifacts.run_matrix_tsv row {row_index} {field} must be none")
            if str(row.get("effective_verdict_source", "")).strip() != "orchestrator":
                failures.append(f"{prefix}.artifacts.run_matrix_tsv row {row_index} effective_verdict_source must be orchestrator")
            if str(row.get("promotion_audit_result", "")).strip() != "pass":
                failures.append(f"{prefix}.artifacts.run_matrix_tsv row {row_index} promotion_audit_result must be pass")
        if observed_pairs != expected_pairs:
            failures.append(f"{prefix}.artifacts.run_matrix_tsv must cover every provider/run exactly once")
        backend = record.get("backend") if isinstance(record.get("backend"), dict) else {}
        if len(data_rows) != len(expected_pairs):
            failures.append(f"{prefix}.artifacts.run_matrix_tsv must contain exactly {len(expected_pairs)} rows")
        if backend.get("total_runs") != len(data_rows):
            failures.append(f"{prefix}.artifacts.run_matrix_tsv total rows do not match record backend.total_runs")
        if backend.get("hard_pass") != hard_pass:
            failures.append(f"{prefix}.artifacts.run_matrix_tsv hard-pass rows do not match record backend.hard_pass")

    run_matrix_path = artifact_paths.get("run_matrix_md")
    if run_matrix_path is not None and run_matrix_path.is_file():
        text = _read_artifact_text(run_matrix_path, f"{prefix}.artifacts.run_matrix_md", failures)
        if "# Run Matrix" not in text:
            failures.append(f"{prefix}.artifacts.run_matrix_md must include a Run Matrix heading")
        header_index = -1
        header: list[str] = []
        lines = text.splitlines()
        for line_index, line in enumerate(lines):
            cells = _table_cells(line)
            if {
                "provider",
                "run",
                "hard_pass",
                "verdict",
                "artifact_source",
                "runtime_contract_failed",
                "runtime_timeout",
                "artifact_quality_failed",
                "failure_class",
                "off_topic_hits",
                "partial_failure_count",
                "issues",
                "effective_verdict_source",
                "promotion_audit_result",
                *REQUIRED_ZERO_ARTIFACT_FIELDS,
            }.issubset(
                {cell.lower() for cell in cells}
            ):
                lowered = [cell.lower() for cell in cells]
                if len(lowered) != len(set(lowered)):
                    failures.append(f"{prefix}.artifacts.run_matrix_md contains duplicate header columns")
                header_index, header = line_index, [cell.lower() for cell in cells]
                break
        if header_index < 0:
            failures.append(f"{prefix}.artifacts.run_matrix_md must include canonical provider/run table")
        else:
            observed_pairs: set[tuple[str, str]] = set()
            table_end = next(
                (line_index for line_index in range(header_index + 1, len(lines)) if lines[line_index].startswith("## ")),
                len(lines),
            )
            for row_index, line in enumerate(lines[header_index + 1 : table_end], start=header_index + 1):
                cells = _table_cells(line)
                if not cells or _is_markdown_separator(cells):
                    continue
                if len(cells) < len(header):
                    failures.append(f"{prefix}.artifacts.run_matrix_md row {row_index} has fewer cells than canonical header")
                    continue
                row = dict(zip(header, cells))
                provider = row.get("provider", "").strip()
                run_index = row.get("run", "").strip()
                if not provider or not run_index:
                    continue
                pair = (provider, run_index)
                if pair in observed_pairs:
                    failures.append(f"{prefix}.artifacts.run_matrix_md contains duplicate provider/run: {provider}/{run_index}")
                observed_pairs.add(pair)
                if pair not in expected_pairs:
                    failures.append(f"{prefix}.artifacts.run_matrix_md has unexpected provider/run: {provider}/{run_index}")
                for field, expected in (
                    ("hard_pass", "1"),
                    ("runtime_contract_status", "passed"),
                    ("artifact_source", "snapshot"),
                    ("runtime_contract_failed", "0"),
                    ("runtime_timeout", "0"),
                    ("artifact_quality_failed", "0"),
                    ("effective_verdict_source", "orchestrator"),
                    ("promotion_audit_result", "pass"),
                ):
                    if row.get(field, "").strip().lower() != expected:
                        failures.append(f"{prefix}.artifacts.run_matrix_md row {row_index} {field} must be {expected}")
                artifact_quality_status = row.get("artifact_quality_status", "").strip().lower()
                if artifact_quality_status not in {"passed", "needs_review"}:
                    failures.append(
                        f"{prefix}.artifacts.run_matrix_md row {row_index} artifact_quality_status must be passed or needs_review"
                    )
                if row.get("verdict", "").strip().lower() not in VALID_PASS_RATINGS:
                    failures.append(f"{prefix}.artifacts.run_matrix_md row {row_index} verdict must be a passing rating")
                blocking_issues = _release_blocking_issues(row.get("issues"))
                if blocking_issues:
                    failures.append(
                        f"{prefix}.artifacts.run_matrix_md row {row_index} contains release-blocking issues: "
                        + ", ".join(blocking_issues)
                    )
                for field in REQUIRED_ZERO_ARTIFACT_FIELDS:
                    if row.get(field, "").strip() not in {"0", "false", "False"}:
                        failures.append(f"{prefix}.artifacts.run_matrix_md row {row_index} {field} must be 0")
                for field in REQUIRED_NONE_ARTIFACT_FIELDS:
                    if row.get(field, "").strip().lower() != "none":
                        failures.append(f"{prefix}.artifacts.run_matrix_md row {row_index} {field} must be none")
            if observed_pairs != expected_pairs:
                failures.append(f"{prefix}.artifacts.run_matrix_md must cover every provider/run exactly once")

    frontend_path = artifact_paths.get("frontend_matrix_md")
    if frontend_path is not None and frontend_path.is_file():
        text = _read_artifact_text(frontend_path, f"{prefix}.artifacts.frontend_matrix_md", failures)
        if "# Frontend Live E2E Matrix" not in text or "## Summary" not in text:
            failures.append(f"{prefix}.artifacts.frontend_matrix_md must include the canonical Summary section")
        header_index = -1
        header: list[str] = []
        lines = text.splitlines()
        summary_indexes = [line_index for line_index, line in enumerate(lines) if line.strip() == "## Summary"]
        if len(summary_indexes) != 1:
            failures.append(f"{prefix}.artifacts.frontend_matrix_md must contain exactly one Summary section")
        summary_start = summary_indexes[0] if len(summary_indexes) == 1 else -1
        summary_end = next(
            (line_index for line_index in range(summary_start + 1, len(lines)) if lines[line_index].startswith("## ")),
            len(lines),
        ) if summary_start >= 0 else -1
        summary_table_count = 0
        for line_index in range(summary_start + 1, summary_end) if summary_start >= 0 else []:
            line = lines[line_index]
            cells = _table_cells(line)
            if {"provider", "status", "runs"}.issubset({cell.lower() for cell in cells}):
                summary_table_count += 1
                lowered = [cell.lower() for cell in cells]
                if len(lowered) != len(set(lowered)):
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md contains duplicate summary header columns")
                if header_index < 0:
                    header_index, header = line_index, lowered
        if summary_table_count != 1:
            failures.append(f"{prefix}.artifacts.frontend_matrix_md must contain exactly one Summary provider table")
        frontend = record.get("frontend") if isinstance(record.get("frontend"), dict) else {}
        expected_statuses = {
            "qwen-code": frontend.get("frontend_qwen_status"),
            "claude-code": frontend.get("frontend_claude_status"),
            "codex-code": frontend.get("frontend_codex_status"),
        }
        observed: set[str] = set()
        if header_index < 0:
            failures.append(f"{prefix}.artifacts.frontend_matrix_md must include canonical provider summary table")
        else:
            for row_index, line in enumerate(lines[header_index + 1 :], start=header_index + 1):
                if line.startswith("## "):
                    break
                cells = _table_cells(line)
                if not cells or _is_markdown_separator(cells):
                    continue
                if len(cells) < len(header):
                    failures.append(
                        f"{prefix}.artifacts.frontend_matrix_md Summary row {row_index} has fewer cells than canonical header"
                    )
                    continue
                row = dict(zip(header, cells))
                provider = row.get("provider", "").strip()
                if not provider:
                    continue
                if provider not in expected_statuses:
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md has unexpected summary provider: {provider}")
                    continue
                if provider in observed:
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md contains duplicate summary provider: {provider}")
                    continue
                observed.add(provider)
                if row.get("status", "").strip() != str(expected_statuses[provider]):
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md row {row_index} status does not match record frontend claim")
                if row.get("runs", "").strip() != "1":
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md row {row_index} runs must be 1")
        if observed != set(expected_statuses):
            failures.append(f"{prefix}.artifacts.frontend_matrix_md must cover every release provider")
        details_header_index = -1
        details_header: list[str] = []
        details_indexes = [
            line_index for line_index, line in enumerate(lines) if line.strip() == "## Run Details"
        ]
        if len(details_indexes) != 1:
            failures.append(f"{prefix}.artifacts.frontend_matrix_md must contain exactly one Run Details section")
        details_start = details_indexes[0] if len(details_indexes) == 1 else -1
        details_end = next(
            (line_index for line_index in range(details_start + 1, len(lines)) if lines[line_index].startswith("## ")),
            len(lines),
        ) if details_start >= 0 else -1
        details_table_count = 0
        for candidate_index in range(details_start + 1, details_end) if details_start >= 0 else []:
            cells = _table_cells(lines[candidate_index])
            if {"provider", "run", "status"}.issubset({cell.lower() for cell in cells}):
                details_table_count += 1
                lowered = [cell.lower() for cell in cells]
                if len(lowered) != len(set(lowered)):
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md contains duplicate details header columns")
                if details_header_index < 0:
                    details_header_index, details_header = candidate_index, lowered
        if details_table_count != 1:
            failures.append(f"{prefix}.artifacts.frontend_matrix_md must contain exactly one Run Details table")
        if details_header_index < 0:
            failures.append(f"{prefix}.artifacts.frontend_matrix_md must include canonical Run Details table")
        else:
            detail_pairs: set[tuple[str, str]] = set()
            for row_index, line in enumerate(lines[details_header_index + 1 : details_end], start=details_header_index + 1):
                if line.startswith("## "):
                    break
                cells = _table_cells(line)
                if not cells or _is_markdown_separator(cells):
                    continue
                if len(cells) < len(details_header):
                    failures.append(
                        f"{prefix}.artifacts.frontend_matrix_md Run Details row {row_index} has fewer cells than canonical header"
                    )
                    continue
                row = dict(zip(details_header, cells))
                provider = row.get("provider", "").strip()
                run_index = row.get("run", "").strip()
                if not provider or not run_index:
                    continue
                pair = (provider, run_index)
                if pair in detail_pairs:
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md contains duplicate detail provider/run: {provider}/{run_index}")
                detail_pairs.add(pair)
                if pair not in expected_pairs:
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md has unexpected detail provider/run: {provider}/{run_index}")
                if row.get("status", "").strip() != str(expected_statuses.get(provider, "")):
                    failures.append(f"{prefix}.artifacts.frontend_matrix_md detail row {row_index} status does not match record frontend claim")
            if detail_pairs != expected_pairs:
                failures.append(f"{prefix}.artifacts.frontend_matrix_md Run Details must cover every provider/run exactly once")

    execution_path = artifact_paths.get("execution_report_md")
    if execution_path is not None and execution_path.is_file():
        text = _read_artifact_text(execution_path, f"{prefix}.artifacts.execution_report_md", failures)
        batch_id = str(record.get("batch_id", "")).strip()
        if f"# Execution Report: {batch_id}" not in text:
            failures.append(f"{prefix}.artifacts.execution_report_md heading must match batch_id")
        if "## Backend Execution Verdict" not in text or "## Public Promotion Authority" not in text:
            failures.append(f"{prefix}.artifacts.execution_report_md must include canonical execution sections")
        provenarch_matches = re.findall(r"^\s*-\s*provenarch_sha:\s*(\S+)\s*$", text, flags=re.MULTILINE)
        if provenarch_matches != [source_sha]:
            failures.append(f"{prefix}.artifacts.execution_report_md provenarch_sha must match release source_sha exactly once")
        backend = record.get("backend") if isinstance(record.get("backend"), dict) else {}
        matches = re.findall(r"^\s*-\s*hard_pass_runs:\s*(\d+)\s*/\s*(\d+)\s*$", text, flags=re.MULTILINE)
        try:
            expected_hard_pass = int(backend.get("hard_pass", -1))
        except (TypeError, ValueError):
            expected_hard_pass = -1
        try:
            expected_total_runs = int(backend.get("total_runs", -1))
        except (TypeError, ValueError):
            expected_total_runs = -1
        if len(matches) != 1 or int(matches[0][0]) != expected_hard_pass or int(matches[0][1]) != expected_total_runs:
            failures.append(f"{prefix}.artifacts.execution_report_md hard_pass_runs does not match record backend claim")
        artifact_matches = re.findall(r"^\s*-\s*artifact_quality_failed_runs:\s*(\d+)\s*/\s*(\d+)\s*$", text, flags=re.MULTILINE)
        if len(artifact_matches) != 1 or artifact_matches[0] != ("0", str(expected_total_runs)):
            failures.append(f"{prefix}.artifacts.execution_report_md artifact_quality_failed_runs must be 0/{expected_total_runs}")
        promotion_matches = re.findall(r"^\s*-\s*promotion_audit_failed_runs:\s*(\d+)\s*/\s*(\d+)\s*$", text, flags=re.MULTILINE)
        if len(promotion_matches) != 1 or promotion_matches[0] != ("0", str(expected_total_runs)):
            failures.append(f"{prefix}.artifacts.execution_report_md promotion_audit_failed_runs must be 0/{expected_total_runs}")
        runtime_flow_matches = re.findall(r"^\s*-\s*runtime_flow_failed_runs:\s*(\d+)\s*/\s*(\d+)\s*$", text, flags=re.MULTILINE)
        if len(runtime_flow_matches) != 1 or runtime_flow_matches[0] != ("0", str(expected_total_runs)):
            failures.append(f"{prefix}.artifacts.execution_report_md runtime_flow_failed_runs must be 0/{expected_total_runs}")
        semantic_matches = re.findall(r"^\s*-\s*semantic_hard_fail_runs:\s*(\d+)\s*/\s*(\d+)\s*$", text, flags=re.MULTILINE)
        if len(semantic_matches) != 1 or semantic_matches[0] != ("0", str(expected_total_runs)):
            failures.append(f"{prefix}.artifacts.execution_report_md semantic_hard_fail_runs must be 0/{expected_total_runs}")
        snapshot_matches = re.findall(r"^\s*-\s*artifact_source_snapshot_runs:\s*(\d+)\s*/\s*(\d+)\s*$", text, flags=re.MULTILINE)
        if len(snapshot_matches) != 1 or snapshot_matches[0] != (str(expected_total_runs), str(expected_total_runs)):
            failures.append(f"{prefix}.artifacts.execution_report_md artifact_source_snapshot_runs must be {expected_total_runs}/{expected_total_runs}")
        artifact_findings_matches = re.findall(
            r"^\s*-\s*artifact_quality_findings:\s*(\d+)\s*$", text, flags=re.MULTILINE
        )
        if len(artifact_findings_matches) != 1 or artifact_findings_matches[0] != "0":
            failures.append(f"{prefix}.artifacts.execution_report_md artifact_quality_findings must be 0")
        primary_failure_matches = re.findall(
            r"^\s*-\s*primary_failure_classes:\s*(.*?)\s*$", text, flags=re.MULTILINE
        )
        if len(primary_failure_matches) != 1 or primary_failure_matches[0] != "none":
            failures.append(f"{prefix}.artifacts.execution_report_md primary_failure_classes must be none")
        partial_matches = re.findall(r"^\s*-\s*partial_failure_count:\s*(\d+)\s*$", text, flags=re.MULTILINE)
        if len(partial_matches) != 1 or partial_matches[0] != "0":
            failures.append(f"{prefix}.artifacts.execution_report_md partial_failure_count must be 0")
        budget_matches = re.findall(
            r"^\s*-\s*provider_budget_exhausted_runs:\s*(\d+)\s*/\s*(\d+)\s*$",
            text,
            flags=re.MULTILINE,
        )
        if len(budget_matches) != 1 or budget_matches[0] != ("0", str(expected_total_runs)):
            failures.append(
                f"{prefix}.artifacts.execution_report_md provider_budget_exhausted_runs must be 0/{expected_total_runs}"
            )
        authority_matches = re.findall(r"^\s*-\s*effective_verdict_sources:\s*(.*?)\s*$", text, flags=re.MULTILINE)
        if len(authority_matches) != 1 or authority_matches[0] != "orchestrator":
            failures.append(f"{prefix}.artifacts.execution_report_md must attest orchestrator promotion authority")
        for provider in RELEASE_PROVIDERS:
            if provider not in text:
                failures.append(f"{prefix}.artifacts.execution_report_md must mention provider {provider}")

        provider_matrix_heading = "## Provider Matrix"
        lines = text.splitlines()
        provider_matrix_indexes = [
            line_index for line_index, line in enumerate(lines) if line.strip() == provider_matrix_heading
        ]
        if len(provider_matrix_indexes) != 1:
            failures.append(
                f"{prefix}.artifacts.execution_report_md must contain exactly one Provider Matrix section"
            )
        else:
            matrix_start = provider_matrix_indexes[0]
            matrix_end = next(
                (line_index for line_index in range(matrix_start + 1, len(lines)) if lines[line_index].startswith("## ")),
                len(lines),
            )
            required_matrix_fields = {
                "provider",
                "runs",
                "pass_rate",
                "off_topic_hits",
                "artifact_quality_findings",
                "semantic_hard_fail_runs",
                "partial_failure_count",
                "runtime_contract_failed_failures",
                "runner_unavailable_failures",
                "runtime_timeout_failures",
                "infra_signal_terminated_failures",
                "infra_incomplete_cycle_failures",
                "quality_gates_failed_failures",
                "artifact_quality_failed_failures",
                "summary_missing_failures",
                "precheck_failed_failures",
                "runtime_flow_failed_failures",
                "cancellation_like_failures",
                "artifact_sources",
                "frontend_live_pass_rate",
            }
            matrix_header_index = -1
            matrix_header: list[str] = []
            matrix_table_count = 0
            for candidate_index in range(matrix_start + 1, matrix_end):
                cells = _table_cells(lines[candidate_index])
                lowered = [cell.lower() for cell in cells]
                if required_matrix_fields.issubset(set(lowered)):
                    matrix_table_count += 1
                    if len(lowered) != len(set(lowered)):
                        failures.append(
                            f"{prefix}.artifacts.execution_report_md Provider Matrix contains duplicate header columns"
                        )
                    if matrix_header_index < 0:
                        matrix_header_index, matrix_header = candidate_index, lowered
            if matrix_table_count != 1:
                failures.append(
                    f"{prefix}.artifacts.execution_report_md must contain exactly one Provider Matrix table"
                )
            if matrix_header_index >= 0:
                backend = record.get("backend") if isinstance(record.get("backend"), dict) else {}
                frontend = record.get("frontend") if isinstance(record.get("frontend"), dict) else {}
                try:
                    expected_total_runs = int(backend.get("total_runs", 0))
                    expected_hard_pass = int(backend.get("hard_pass", 0))
                except (TypeError, ValueError):
                    expected_total_runs, expected_hard_pass = 0, 0
                expected_provider_runs = (
                    expected_total_runs // len(RELEASE_PROVIDERS) if len(RELEASE_PROVIDERS) else 0
                )
                expected_provider_hard = (
                    expected_hard_pass // len(RELEASE_PROVIDERS) if len(RELEASE_PROVIDERS) else 0
                )
                expected_frontend = {
                    "qwen-code": frontend.get("frontend_qwen_status"),
                    "claude-code": frontend.get("frontend_claude_status"),
                    "codex-code": frontend.get("frontend_codex_status"),
                }
                expected_zero_matrix_fields = (
                    "off_topic_hits",
                    "artifact_quality_findings",
                    "semantic_hard_fail_runs",
                    "partial_failure_count",
                    "runtime_contract_failed_failures",
                    "runner_unavailable_failures",
                    "runtime_timeout_failures",
                    "infra_signal_terminated_failures",
                    "infra_incomplete_cycle_failures",
                    "quality_gates_failed_failures",
                    "artifact_quality_failed_failures",
                    "summary_missing_failures",
                    "precheck_failed_failures",
                    "runtime_flow_failed_failures",
                    "cancellation_like_failures",
                )
                observed_providers: set[str] = set()
                for row_index, line in enumerate(
                    lines[matrix_header_index + 1 : matrix_end], start=matrix_header_index + 1
                ):
                    cells = _table_cells(line)
                    if not cells or _is_markdown_separator(cells):
                        continue
                    if len(cells) < len(matrix_header):
                        failures.append(
                            f"{prefix}.artifacts.execution_report_md Provider Matrix row {row_index} has fewer cells than canonical header"
                        )
                        continue
                    row = dict(zip(matrix_header, cells))
                    provider = row.get("provider", "").strip()
                    if not provider:
                        continue
                    if provider in observed_providers:
                        failures.append(
                            f"{prefix}.artifacts.execution_report_md Provider Matrix contains duplicate provider: {provider}"
                        )
                    observed_providers.add(provider)
                    if provider not in RELEASE_PROVIDERS:
                        failures.append(
                            f"{prefix}.artifacts.execution_report_md Provider Matrix has unexpected provider: {provider}"
                        )
                        continue
                    try:
                        runs_value = int(row.get("runs", ""))
                    except ValueError:
                        runs_value = -1
                    if runs_value != expected_provider_runs:
                        failures.append(
                            f"{prefix}.artifacts.execution_report_md Provider Matrix row {row_index} runs must be {expected_provider_runs}"
                        )
                    try:
                        pass_rate = float(row.get("pass_rate", ""))
                    except ValueError:
                        pass_rate = math.nan
                    expected_pass_rate = (
                        expected_provider_hard / expected_provider_runs if expected_provider_runs else math.nan
                    )
                    if not math.isfinite(pass_rate) or not math.isclose(
                        pass_rate, expected_pass_rate, rel_tol=0.0, abs_tol=0.001
                    ):
                        failures.append(
                            f"{prefix}.artifacts.execution_report_md Provider Matrix row {row_index} pass_rate does not match backend claim"
                        )
                    for field in expected_zero_matrix_fields:
                        try:
                            value = int(row.get(field, ""))
                        except ValueError:
                            value = -1
                        if value != 0:
                            failures.append(
                                f"{prefix}.artifacts.execution_report_md Provider Matrix row {row_index} {field} must be 0"
                            )
                    if row.get("artifact_sources", "").strip() != f"snapshot={runs_value}":
                        failures.append(
                            f"{prefix}.artifacts.execution_report_md Provider Matrix row {row_index} artifact_sources must be snapshot={runs_value}"
                        )
                    expected_frontend_rate = 1.0 if expected_frontend.get(provider) == "passed" else 0.0
                    try:
                        frontend_rate = float(row.get("frontend_live_pass_rate", ""))
                    except ValueError:
                        frontend_rate = math.nan
                    if not math.isfinite(frontend_rate) or not math.isclose(
                        frontend_rate, expected_frontend_rate, rel_tol=0.0, abs_tol=0.001
                    ):
                        failures.append(
                            f"{prefix}.artifacts.execution_report_md Provider Matrix row {row_index} frontend_live_pass_rate does not match frontend claim"
                        )
                if observed_providers != set(RELEASE_PROVIDERS):
                    failures.append(
                        f"{prefix}.artifacts.execution_report_md Provider Matrix must cover every release provider exactly once"
                    )

    return failures


def verify_payload(
    payload: dict[str, Any],
    *,
    expected_source_sha: str | None,
    expected_tag: str | None,
    cwd: Path,
    verdict_root: Path,
    max_age_hours: float,
    now: datetime | None = None,
) -> list[str]:
    failures: list[str] = []
    matrix_id = str(payload.get("matrix_id", "")).strip()
    source_sha = str(payload.get("source_sha", "")).strip()
    unknown = sorted(set(payload) - TOP_LEVEL_FIELDS)
    if unknown:
        failures.append(f"unknown top-level release evidence fields: {', '.join(unknown)}")
    for field in TOP_LEVEL_FIELDS:
        if field not in payload:
            failures.append(f"{field} must be present in release evidence")

    if payload.get("evidence_schema_version") != EVIDENCE_SCHEMA_VERSION:
        failures.append(f"evidence_schema_version must be {EVIDENCE_SCHEMA_VERSION}")
    if payload.get("source_tree_clean") is not True:
        failures.append("source_tree_clean must be true for release evidence")
    if payload.get("generator") != "scripts/full-run-batch-matrix.sh":
        failures.append("generator must identify scripts/full-run-batch-matrix.sh")
    failures.extend(verify_source_binding(payload, expected_source_sha, expected_tag, cwd))

    generated_at, timestamp_error = parse_utc(payload.get("generated_at_utc"), "generated_at_utc")
    if timestamp_error:
        failures.append(timestamp_error)
    else:
        current = now or datetime.now(timezone.utc)
        assert generated_at is not None
        if generated_at > current + timedelta(minutes=5):
            failures.append("generated_at_utc is in the future")
        elif current - generated_at > timedelta(hours=max_age_hours):
            failures.append(f"release evidence is older than {max_age_hours:g} hours")

    if payload.get("verdict") != "PASS":
        failures.append(f"verdict must be PASS, got {payload.get('verdict')!r}")
    if payload.get("release_state") != "RELEASE READY":
        failures.append(f"release_state must be RELEASE READY, got {payload.get('release_state')!r}")

    for field in ("profile_sweep_runs", "strict_pass_runs", "strict_fail_runs"):
        value = payload.get(field)
        if not isinstance(value, int) or isinstance(value, bool):
            failures.append(f"{field} must be an integer")
    if payload.get("strict_fail_runs") != 0:
        failures.append("strict_fail_runs must be 0 for RELEASE READY evidence")

    aggregate_backend = payload.get("backend")
    expected_aggregate_runs = expected_runs = 0

    release_contract = payload.get("release_contract")
    if not isinstance(release_contract, dict):
        failures.append("release_contract must be an object")
        release_contract = {}
    if release_contract.get("mode") != "release":
        failures.append(f"release_contract.mode must be release, got {release_contract.get('mode')!r}")
    if release_contract.get("contract_status") != "passed":
        failures.append(
            "release_contract.contract_status must be passed, "
            f"got {release_contract.get('contract_status')!r}"
        )
    if release_contract.get("required_sweeps") != REQUIRED_SWEEPS:
        failures.append(f"release_contract.required_sweeps must be {REQUIRED_SWEEPS!r}")
    if release_contract.get("observed_sweeps") != REQUIRED_SWEEPS:
        failures.append(f"release_contract.observed_sweeps must be {REQUIRED_SWEEPS!r}")
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
    required_profiles = release_contract.get("required_profiles")
    if (
        not isinstance(required_profiles, list)
        or not all(isinstance(profile, str) for profile in required_profiles)
        or len(required_profiles) != 2
        or len(set(required_profiles)) != 2
        or any(profile not in ALLOWED_PROFILES for profile in required_profiles)
        or not str(required_profiles[0]).startswith("single-")
        or not str(required_profiles[1]).startswith("multi-")
    ):
        failures.append(
            "release_contract.required_profiles must contain exactly one single-* and one multi-* profile"
        )
        required_profiles = []
    observed_profiles = release_contract.get("observed_profiles")
    if (
        not isinstance(observed_profiles, list)
        or not all(isinstance(profile, str) for profile in observed_profiles)
        or set(observed_profiles) != set(required_profiles)
    ):
        failures.append("release_contract.observed_profiles must exactly match required_profiles")
    expected_runs = len(required_profiles) * len(REQUIRED_SWEEPS)
    if release_contract.get("expected_profile_sweep_runs") != expected_runs:
        failures.append(f"release_contract.expected_profile_sweep_runs must be {expected_runs}")
    if release_contract.get("observed_profile_sweep_runs") != expected_runs:
        failures.append(f"release_contract.observed_profile_sweep_runs must be {expected_runs}")
    if release_contract.get("shard_plan_invariant_status") != "passed":
        failures.append("release_contract.shard_plan_invariant_status must be passed")
    if release_contract.get("blocking_reasons") != []:
        failures.append("release_contract.blocking_reasons must be an empty list")
    if payload.get("profile_sweep_runs") != expected_runs:
        failures.append(f"profile_sweep_runs must be {expected_runs}")
    if payload.get("strict_pass_runs") != expected_runs:
        failures.append(f"strict_pass_runs must be {expected_runs}")
    expected_aggregate_runs = expected_runs * len(RELEASE_PROVIDERS)
    if not isinstance(aggregate_backend, dict):
        failures.append("backend must be an object")
    else:
        if aggregate_backend.get("total_runs") != expected_aggregate_runs:
            failures.append(f"backend.total_runs must be {expected_aggregate_runs}")
        if aggregate_backend.get("hard_pass") != expected_aggregate_runs:
            failures.append(f"backend.hard_pass must be {expected_aggregate_runs}")
        for field in REQUIRED_ZERO_BACKEND_FIELDS:
            value = aggregate_backend.get(field)
            if not isinstance(value, int) or isinstance(value, bool):
                failures.append(f"backend.{field} must be an integer")
            elif value != 0:
                failures.append(f"backend.{field} must be 0 for RELEASE READY evidence")

    records = payload.get("records")
    expected_pairs = {(profile, sweep) for profile in required_profiles for sweep in REQUIRED_SWEEPS}
    observed_pairs: set[tuple[str, str]] = set()
    observed_batch_ids: set[str] = set()
    observed_artifact_refs: set[str] = set()
    if not isinstance(records, list) or len(records) != expected_runs:
        failures.append(f"records must contain exactly {expected_runs} profile/sweep records")
        records = []
    for index, record in enumerate(records):
        if not isinstance(record, dict):
            failures.append(f"records[{index}] must be an object")
            continue
        profile = str(record.get("profile_id", "")).strip()
        sweep = str(record.get("sweep_id", "")).strip()
        pair = (profile, sweep)
        if pair in observed_pairs:
            failures.append(f"records contains duplicate profile/sweep pair: {profile}/{sweep}")
        observed_pairs.add(pair)
        if pair not in expected_pairs:
            failures.append(f"records[{index}] has unexpected profile/sweep pair: {profile}/{sweep}")
        batch_id = str(record.get("batch_id", "")).strip()
        if not batch_id:
            failures.append(f"records[{index}].batch_id must be present")
        elif batch_id in observed_batch_ids:
            failures.append(f"records contains duplicate batch_id: {batch_id}")
        else:
            observed_batch_ids.add(batch_id)
        def batch_slug(value: str) -> str:
            return re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-") or "item"

        expected_batch_id = (
            f"{matrix_id}-{batch_slug(profile)}-{batch_slug(sweep)}"
            if matrix_id and profile and sweep
            else ""
        )
        if expected_batch_id and batch_id != expected_batch_id:
            failures.append(f"records[{index}].batch_id must be {expected_batch_id!r}")
        if record.get("status") != "passed":
            failures.append(f"records[{index}].status must be passed, got {record.get('status')!r}")
        if record.get("strict_status") != "passed":
            failures.append(f"records[{index}].strict_status must be passed, got {record.get('strict_status')!r}")
        if record.get("blocking_reasons") != []:
            failures.append(f"records[{index}].blocking_reasons must be an empty list")
        execution = record.get("execution")
        if not isinstance(execution, dict):
            failures.append(f"records[{index}].execution must be an object")
        else:
            for key in ("strategy", "max_parallel_tasks", "failure_policy", "shard_discovery_mode"):
                value = execution.get(key)
                if value in (None, "", "-"):
                    failures.append(f"records[{index}].execution.{key} must be present")
            try:
                if int(execution.get("max_parallel_tasks", 0)) <= 0:
                    failures.append(f"records[{index}].execution.max_parallel_tasks must be positive")
            except (TypeError, ValueError):
                failures.append(f"records[{index}].execution.max_parallel_tasks must be an integer")
            expected_execution = REQUIRED_EXECUTION_BY_SWEEP.get(sweep)
            if expected_execution is not None:
                for key, expected in expected_execution.items():
                    observed = str(execution.get(key, "")).strip()
                    if key == "max_parallel_tasks":
                        raw_max_parallel = execution.get(key)
                        if isinstance(raw_max_parallel, bool):
                            observed = str(raw_max_parallel)
                        elif isinstance(raw_max_parallel, int):
                            observed = str(raw_max_parallel)
                        elif isinstance(raw_max_parallel, str) and re.fullmatch(r"[0-9]+", raw_max_parallel.strip()):
                            observed = str(int(raw_max_parallel.strip()))
                        else:
                            observed = str(raw_max_parallel).strip()
                    if observed != expected:
                        failures.append(
                            f"records[{index}].execution.{key} must be {expected!r} for sweep {sweep!r}"
                        )
        backend = record.get("backend")
        if not isinstance(backend, dict):
            failures.append(f"records[{index}].backend must be an object")
        else:
            if backend.get("total_runs") != len(RELEASE_PROVIDERS):
                failures.append(f"records[{index}].backend.total_runs must be {len(RELEASE_PROVIDERS)}")
            if backend.get("hard_pass") != len(RELEASE_PROVIDERS):
                failures.append(f"records[{index}].backend.hard_pass must be {len(RELEASE_PROVIDERS)}")
            for field in REQUIRED_ZERO_BACKEND_FIELDS:
                value = backend.get(field)
                if not isinstance(value, int) or isinstance(value, bool):
                    failures.append(f"records[{index}].backend.{field} must be an integer")
                elif value != 0:
                    failures.append(f"records[{index}].backend.{field} must be 0")
        frontend = record.get("frontend")
        if not isinstance(frontend, dict):
            failures.append(f"records[{index}].frontend must be an object")
        else:
            for provider_key in ("frontend_qwen_status", "frontend_claude_status", "frontend_codex_status"):
                if frontend.get(provider_key) != "passed":
                    failures.append(f"records[{index}].frontend.{provider_key} must be passed")
        if record.get("shard_plan_invariant") != "passed":
            failures.append(f"records[{index}].shard_plan_invariant must be passed")
        authority = record.get("public_authority")
        if not isinstance(authority, dict):
            failures.append(f"records[{index}].public_authority must be an object")
        else:
            if authority.get("effective_verdict_source") != "orchestrator":
                failures.append(
                    f"records[{index}].public_authority.effective_verdict_source must be orchestrator"
                )
            if authority.get("promotion_audit_result") != "pass":
                failures.append(f"records[{index}].public_authority.promotion_audit_result must be pass")
        artifacts = record.get("artifacts")
        if not isinstance(artifacts, dict):
            failures.append(f"records[{index}].artifacts must be an object")
        else:
            for artifact_key in ("run_matrix_tsv", "run_matrix_md", "frontend_matrix_md", "execution_report_md"):
                if not str(artifacts.get(artifact_key, "")).strip() or artifacts.get(artifact_key) == "-":
                    failures.append(f"records[{index}].artifacts.{artifact_key} must be present")
            artifact_digests = artifacts.get("artifact_digests")
            if not isinstance(artifact_digests, dict):
                failures.append(f"records[{index}].artifacts.artifact_digests must be an object")
            else:
                resolved_artifacts: dict[str, Path] = {}
                for artifact_key in ("run_matrix_tsv", "run_matrix_md", "frontend_matrix_md", "execution_report_md"):
                    digest = str(artifact_digests.get(artifact_key, ""))
                    if not re.fullmatch(r"[0-9a-f]{64}", digest):
                        failures.append(f"records[{index}].artifacts.artifact_digests.{artifact_key} must be a SHA-256 digest")
                        continue
                    artifact_path = Path(str(artifacts.get(artifact_key, "")))
                    profile_slug = re.sub(r"[^A-Za-z0-9._-]+", "_", profile)
                    sweep_slug = re.sub(r"[^A-Za-z0-9._-]+", "_", sweep)
                    expected_suffix = Path(str(artifacts.get(artifact_key, ""))).suffix or ".txt"
                    expected_artifact_ref = f"artifact_{matrix_id}_{profile_slug}_{sweep_slug}_{artifact_key}{expected_suffix}"
                    if artifact_path.as_posix() != expected_artifact_ref:
                        failures.append(
                            f"records[{index}].artifacts.{artifact_key} must be canonical per-record path {expected_artifact_ref!r}"
                        )
                    if artifact_path.as_posix() in observed_artifact_refs:
                        failures.append(f"records contains duplicate artifact reference: {artifact_path.as_posix()}")
                    observed_artifact_refs.add(artifact_path.as_posix())
                    if artifact_path.is_absolute() or ".." in artifact_path.parts:
                        failures.append(f"records[{index}].artifacts.{artifact_key} must be a safe relative path")
                        continue
                    resolved = (verdict_root / artifact_path).resolve()
                    try:
                        resolved.relative_to(verdict_root.resolve())
                    except ValueError:
                        failures.append(f"records[{index}].artifacts.{artifact_key} escapes the verdict directory")
                        continue
                    if not resolved.is_file() or resolved.stat().st_size == 0:
                        failures.append(f"records[{index}].artifacts.{artifact_key} must reference a non-empty file")
                    elif hashlib.sha256(resolved.read_bytes()).hexdigest() != digest:
                        failures.append(f"records[{index}].artifacts.{artifact_key} digest does not match content")
                    else:
                        resolved_artifacts[artifact_key] = resolved
                if len(resolved_artifacts) == 4:
                    failures.extend(verify_record_artifact_content(record, resolved_artifacts, index, source_sha))
    if observed_pairs != expected_pairs:
        failures.append("records must cover every required profile/sweep pair exactly once")

    evidence_artifacts = payload.get("evidence_artifacts")
    matrix_id = str(payload.get("matrix_id", "")).strip()
    expected_artifact_paths = {
        "verdict_markdown": f"release_verdict_{matrix_id}.md",
        "profile_matrix_markdown": f"profile_matrix_{matrix_id}.md",
        "profile_matrix_tsv": f"profile_matrix_{matrix_id}.tsv",
    }
    if not isinstance(evidence_artifacts, dict) or set(evidence_artifacts) != set(expected_artifact_paths):
        failures.append("evidence_artifacts must contain verdict_markdown, profile_matrix_markdown and profile_matrix_tsv")
    else:
        for key, expected_path in expected_artifact_paths.items():
            entry = evidence_artifacts.get(key)
            if not isinstance(entry, dict) or set(entry) != {"path", "sha256"}:
                failures.append(f"evidence_artifacts.{key} must contain path and sha256")
                continue
            if entry.get("path") != expected_path:
                failures.append(f"evidence_artifacts.{key}.path must be {expected_path!r}")
                continue
            digest = str(entry.get("sha256", ""))
            if not re.fullmatch(r"[0-9a-f]{64}", digest):
                failures.append(f"evidence_artifacts.{key}.sha256 must be a SHA-256 digest")
                continue
            artifact_path = verdict_root / expected_path
            if not artifact_path.is_file() or artifact_path.stat().st_size == 0:
                failures.append(f"evidence_artifacts.{key} must reference a non-empty file")
            elif hashlib.sha256(artifact_path.read_bytes()).hexdigest() != digest:
                failures.append(f"evidence_artifacts.{key} digest does not match content")
    return failures


def parse_markdown_scalar(text: str, key: str) -> str:
    pattern = rf"^\s*(?:[-*]\s*)?{re.escape(key)}\s*:\s*(.+?)\s*$"
    match = re.search(pattern, text, flags=re.IGNORECASE | re.MULTILINE)
    return match.group(1).strip() if match else ""


def matrix_id_from_verdict_path(path: Path) -> str:
    match = re.fullmatch(r"release_verdict_(.+)\.json", path.name)
    return match.group(1) if match else ""


def verify_verdict_markdown(path: Path, payload: dict[str, Any]) -> list[str]:
    try:
        text = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return [f"release verdict markdown file not found: {path}"]
    except OSError as exc:
        return [f"failed to read release verdict markdown {path}: {exc}"]
    except UnicodeDecodeError as exc:
        return [f"release verdict markdown is not valid UTF-8 {path}: {exc}"]
    matrix_id = str(payload.get("matrix_id", "")).strip()
    failures: list[str] = []
    if f"# Release Verdict: {matrix_id}" not in text:
        failures.append("release verdict markdown heading does not match matrix_id")
    for field in ("generated_at_utc", "verdict", "release_state"):
        if parse_markdown_scalar(text, field) != str(payload.get(field, "")):
            failures.append(f"release verdict markdown {field} does not match JSON")
    if parse_markdown_scalar(text, "release_contract_status") != "passed":
        failures.append("release verdict markdown release_contract_status must be passed")
    if "## Release Contract" not in text:
        failures.append("release verdict markdown must include Release Contract section")
    return failures


def verify_profile_matrix_files(reports_root: Path, payload: dict[str, Any]) -> list[str]:
    matrix_id = str(payload.get("matrix_id", "")).strip()
    contract = payload.get("release_contract") if isinstance(payload.get("release_contract"), dict) else {}
    profiles = contract.get("required_profiles") if isinstance(contract.get("required_profiles"), list) else []
    expected_pairs = {(str(profile), sweep) for profile in profiles for sweep in REQUIRED_SWEEPS}
    failures: list[str] = []
    md_path = reports_root / f"profile_matrix_{matrix_id}.md"
    tsv_path = reports_root / f"profile_matrix_{matrix_id}.tsv"
    try:
        md_text = md_path.read_text(encoding="utf-8")
    except FileNotFoundError:
        failures.append(f"profile matrix markdown file not found: {md_path}")
        md_text = ""
    except OSError as exc:
        failures.append(f"failed to read profile matrix markdown {md_path}: {exc}")
        md_text = ""
    except UnicodeDecodeError as exc:
        failures.append(f"profile matrix markdown is not valid UTF-8 {md_path}: {exc}")
        md_text = ""
    if md_text and "# Profile Matrix" not in md_text:
        failures.append("profile matrix markdown heading is missing")
    for profile, sweep in expected_pairs:
        if f"| {profile} | {sweep} |" not in md_text:
            failures.append(f"profile matrix markdown is missing {profile}/{sweep}")
    try:
        rows = list(csv.DictReader(tsv_path.read_text(encoding="utf-8").splitlines(), delimiter="\t"))
    except FileNotFoundError:
        failures.append(f"profile matrix TSV file not found: {tsv_path}")
        rows = []
    except (OSError, UnicodeDecodeError, csv.Error) as exc:
        failures.append(f"failed to read profile matrix TSV {tsv_path}: {exc}")
        rows = []
    row_pairs: set[tuple[str, str]] = set()
    for index, row in enumerate(rows):
        pair = (str(row.get("profile_id", "")).strip(), str(row.get("sweep_id", "")).strip())
        row_pairs.add(pair)
        if row.get("strict_status") != "passed":
            failures.append(f"profile matrix TSV row {index} strict_status must be passed")
    if len(rows) != len(expected_pairs):
        failures.append("profile matrix TSV must contain exactly one row per required profile/sweep pair")
    if len(row_pairs) != len(rows):
        failures.append("profile matrix TSV contains duplicate profile/sweep pairs")
    if row_pairs != expected_pairs:
        failures.append("profile matrix TSV must cover every required profile/sweep pair exactly once")
    return failures


def verify_manual_assessment(
    path: Path,
    matrix_id: str,
    label: str,
    source_sha: str,
    verdict_generated_at: datetime | None,
    max_age_hours: float,
    now: datetime,
) -> list[str]:
    failures: list[str] = []
    try:
        text = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return [f"{label} assessment file not found: {path}"]
    except OSError as exc:
        return [f"failed to read {label} assessment file {path}: {exc}"]
    except UnicodeDecodeError as exc:
        return [f"{label} assessment file is not valid UTF-8 {path}: {exc}"]

    observed_matrix_id = parse_markdown_scalar(text, "matrix_id")
    if observed_matrix_id != matrix_id:
        failures.append(f"{label} assessment matrix_id must be {matrix_id!r}, got {observed_matrix_id!r}")
    decision = parse_markdown_scalar(text, "decision") or parse_markdown_scalar(text, "status")
    if decision.lower() != "accepted":
        failures.append(f"{label} assessment decision/status must be accepted, got {decision!r}")
    if parse_markdown_scalar(text, "source_sha") != source_sha:
        failures.append(f"{label} assessment source_sha must match release source SHA")
    assessed_by = parse_markdown_scalar(text, "assessed_by")
    if not assessed_by:
        failures.append(f"{label} assessment assessed_by must be present")
    assessed_at, timestamp_error = parse_utc(parse_markdown_scalar(text, "assessed_at_utc"), f"{label} assessed_at_utc")
    if timestamp_error:
        failures.append(timestamp_error)
    else:
        assert assessed_at is not None
        if assessed_at > now + timedelta(minutes=5):
            failures.append(f"{label} assessed_at_utc is in the future")
        elif now - assessed_at > timedelta(hours=max_age_hours):
            failures.append(f"{label} assessment is older than {max_age_hours:g} hours")
        if verdict_generated_at is not None and assessed_at < verdict_generated_at:
            failures.append(f"{label} assessed_at_utc predates release verdict evidence")
    expected_verdict_ref = f"release_verdict_{matrix_id}.json"
    if parse_markdown_scalar(text, "release verdict") != expected_verdict_ref:
        failures.append(f"{label} assessment must reference {expected_verdict_ref}")
    evidence_section = re.search(
        r"^##\s+Evidence Inspected\s*$([\s\S]*?)(?=^##\s+|\Z)",
        text,
        flags=re.IGNORECASE | re.MULTILINE,
    )
    if not evidence_section:
        failures.append(f"{label} assessment must include an Evidence Inspected section")
    elif not any(
        re.match(r"^\s*[-*]\s+[^:]+:\s*\S+", line)
        for line in evidence_section.group(1).splitlines()
    ):
        failures.append(f"{label} assessment Evidence Inspected section must name inspected evidence")
    return failures


def verify_manual_assessments(
    verdict_path: Path,
    matrix_id: str,
    payload: dict[str, Any],
    max_age_hours: float,
    now: datetime,
) -> list[str]:
    reports_root = verdict_path.parent
    verdict_generated_at, _ = parse_utc(payload.get("generated_at_utc"), "generated_at_utc")
    source_sha = str(payload.get("source_sha", "")).strip()
    failures: list[str] = []
    for label, template in MANUAL_ASSESSMENTS.items():
        path = reports_root / template.format(matrix_id=matrix_id)
        failures.extend(
            verify_manual_assessment(
                path,
                matrix_id,
                label,
                source_sha,
                verdict_generated_at,
                max_age_hours,
                now,
            )
        )
    return failures


def verify_verdict_path(
    verdict_path: Path,
    *,
    expected_source_sha: str | None,
    expected_tag: str | None,
    cwd: Path,
    max_age_hours: float,
    now: datetime,
) -> tuple[str, list[str]]:
    payload = load_payload(verdict_path)
    matrix_id = str(payload.get("matrix_id", "")).strip()
    path_matrix_id = matrix_id_from_verdict_path(verdict_path)
    failures = verify_payload(
        payload,
        expected_source_sha=expected_source_sha,
        expected_tag=expected_tag,
        cwd=cwd,
        verdict_root=verdict_path.parent,
        max_age_hours=max_age_hours,
        now=now,
    )
    if not matrix_id:
        failures.append("matrix_id must be present")
    elif not MATRIX_ID_PATTERN.fullmatch(matrix_id):
        failures.append("matrix_id contains invalid characters")
    expected_filename = f"release_verdict_{matrix_id}.json" if matrix_id else ""
    if verdict_path.parent.name != "reports":
        failures.append("release verdict must be stored directly under a reports directory")
    if not path_matrix_id:
        failures.append("release verdict filename must be release_verdict_<matrix_id>.json")
    elif path_matrix_id != matrix_id:
        failures.append(f"matrix_id must match verdict filename {path_matrix_id!r}, got {matrix_id!r}")
    if matrix_id and path_matrix_id == matrix_id and verdict_path.name == expected_filename:
        failures.extend(verify_verdict_markdown(verdict_path.parent / f"release_verdict_{matrix_id}.md", payload))
        failures.extend(verify_profile_matrix_files(verdict_path.parent, payload))
        failures.extend(verify_manual_assessments(verdict_path, matrix_id, payload, max_age_hours, now))
    return matrix_id, failures


def parse_matrix_ids(value: str) -> list[str]:
    matrix_ids = [item.strip() for item in value.split(",")]
    if not matrix_ids or any(not item for item in matrix_ids):
        raise SystemExit("ACP_RELEASE_MATRIX_IDS contains an empty matrix id")
    invalid = [item for item in matrix_ids if not MATRIX_ID_PATTERN.fullmatch(item)]
    if invalid:
        raise SystemExit(f"invalid matrix id in ACP_RELEASE_MATRIX_IDS: {invalid[0]!r}")
    duplicates = sorted({item for item in matrix_ids if matrix_ids.count(item) > 1})
    if duplicates:
        raise SystemExit(f"duplicate matrix id in ACP_RELEASE_MATRIX_IDS: {duplicates[0]!r}")
    return matrix_ids


def resolve_verdict_paths(args: argparse.Namespace) -> list[Path]:
    configured = [bool(args.verdict_json), bool(args.matrix_ids), bool(args.matrix_id), bool(args.verdict_path)]
    if sum(configured) != 1:
        raise SystemExit(
            "set exactly one release evidence mode: positional verdict paths, "
            "--matrix-ids, --matrix-id or --verdict-path"
        )
    if args.verdict_json:
        return list(args.verdict_json)
    if args.matrix_ids:
        return [Path("reports") / f"release_verdict_{item}.json" for item in parse_matrix_ids(args.matrix_ids)]
    if args.matrix_id:
        matrix_ids = parse_matrix_ids(args.matrix_id)
        if len(matrix_ids) != 1:
            raise SystemExit("--matrix-id accepts exactly one matrix id")
        return [Path("reports") / f"release_verdict_{matrix_ids[0]}.json"]
    return [args.verdict_path]


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("verdict_json", type=Path, nargs="*", help="release_verdict_<matrix-id>.json paths")
    parser.add_argument("--matrix-ids", help="Comma-separated matrix IDs resolved under reports/")
    parser.add_argument("--matrix-id", help="Single matrix ID resolved under reports/")
    parser.add_argument("--verdict-path", type=Path, help="Explicit single verdict path")
    parser.add_argument("--source-sha", help="Expected Git source SHA (required by tag release workflow)")
    parser.add_argument("--tag", help="Expected SemVer release tag (required by tag release workflow)")
    parser.add_argument(
        "--max-age-hours",
        type=float,
        default=DEFAULT_MAX_AGE_HOURS,
        help=f"Maximum evidence age in hours (default: {DEFAULT_MAX_AGE_HOURS:g})",
    )
    args = parser.parse_args(argv)
    if not math.isfinite(args.max_age_hours) or args.max_age_hours <= 0:
        raise SystemExit("--max-age-hours must be greater than zero")
    verdict_paths = resolve_verdict_paths(args)
    now = datetime.now(timezone.utc)

    failures: list[str] = []
    matrix_ids: set[str] = set()
    for verdict_path in verdict_paths:
        matrix_id, verdict_failures = verify_verdict_path(
            verdict_path,
            expected_source_sha=args.source_sha,
            expected_tag=args.tag,
            cwd=Path.cwd(),
            max_age_hours=args.max_age_hours,
            now=now,
        )
        if matrix_id:
            if matrix_id in matrix_ids:
                failures.append(f"duplicate matrix_id in composite release evidence: {matrix_id!r}")
            matrix_ids.add(matrix_id)
        failures.extend(f"{verdict_path}: {failure}" for failure in verdict_failures)
    if failures:
        for failure in failures:
            print(f"release evidence not ready: {failure}", file=sys.stderr)
        return 1
    for verdict_path in verdict_paths:
        print(f"release evidence ready: {verdict_path}")
    if len(verdict_paths) > 1:
        print(f"composite release evidence ready: {len(verdict_paths)} constituent matrices")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
