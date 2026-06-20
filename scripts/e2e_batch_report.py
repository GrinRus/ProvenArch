#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import sys
from collections import Counter, defaultdict
from dataclasses import dataclass, field
from pathlib import Path
from statistics import mean, pstdev
from typing import Any

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from e2e_report_classifiers import (
    extract_focused_recovery_reason_counts,
    extract_focused_recovery_reason_tags,
    failure_class_rank,
    should_ignore_classified_incomplete_for_terminal_process,
    should_ignore_stale_classified_failure,
    terminal_process_failed_summary,
    terminal_success_summary,
    text_has_raw_provider_runner_unavailable_signal,
    text_has_runner_unavailable_signal,
    text_has_collect_document_path_contract_signature,
    text_has_runtime_contract_parse_signature,
    text_has_structured_runner_unavailable_signal,
)

RUN_RESULTS_COLUMNS = [
    "iteration",
    "runtime_mode",
    "runtime_provider",
    "pipeline",
    "run_id",
    "status",
    "signal",
    "entities",
    "edges",
    "findings",
    "questions",
    "cov_obs",
    "cov_missing",
    "warnings",
    "runtime_versions",
    "quality_path",
    "output_path",
]

OFF_TOPIC_TERMS = (
    "bidding",
    "tender",
    "chinabidding",
    "power system",
    "power enterprise",
    "relay protection",
    "load flow",
    "electric analysis",
    "继电",
    "潮流",
    "电力",
    "招标",
)

POWER_TARGET_HINTS = (
    "power",
    "energy",
    "electric",
    "grid",
    "utility",
)

SYNTHETIC_EVIDENCE_PREFIXES = (
    "search_source/",
    "search_query/",
    "search_config/",
    "web_search/",
    "browser/",
)

RUNTIME_FLOW_ISSUE_TAGS = (
    "runtime:shard-artifacts",
    "runtime:shard-metadata",
    "runtime:execution-semantics",
)
ARTIFACT_QUALITY_WARNING_PREFIX = "artifact_quality:"
QUALITY_COUNTER_KEYS = (
    "repair_attempts",
    "repair_exhausted",
    "fresh_retries",
    "focused_repairs",
    "stall_count",
    "pre_artifact_stalls",
    "post_artifact_stalls",
    "zero_output_pre_artifact_stalls",
    "partial_failure_count",
)
FOCUSED_REPAIR_EXHAUSTED_REASON_TAGS = (
    "collect_pair_repair_exhausted",
    "collect_manifest_repair_exhausted",
    "validator_verdict_repair_exhausted",
    "draft_artifact_repair_exhausted",
    "draft_artifact_enrichment_exhausted",
)

FRONTEND_PROVIDERS = ("qwen-code", "claude-code", "codex-code")
FRONTEND_LIVE_RESULT_FILENAME = "frontend-e2e-result.json"


def normalize_text(value: str) -> str:
    cleaned = value.strip().lower().replace("_", " ").replace("-", " ")
    return " ".join(cleaned.split())


def workspace_candidates(run_dir: Path) -> list[Path]:
    return [
        run_dir / "headless" / "arch-workspace",
        run_dir / "arch-workspace",
        run_dir / "workspace",
    ]


def resolve_workspace(run_dir: Path) -> tuple[Path, list[Path]]:
    candidates = workspace_candidates(run_dir)
    for candidate in candidates:
        if candidate.exists() and candidate.is_dir():
            return candidate, candidates
    return candidates[0], candidates


def read_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def read_text_file(path: Path) -> str:
    return path.read_text(encoding="utf-8", errors="replace")


def parse_markdown_scalar(text: str, key: str) -> str:
    match = re.search(rf"^- {re.escape(key)}:\s*(.+)$", text, flags=re.MULTILINE)
    return match.group(1).strip() if match else ""


def parse_api_status(text: str) -> str:
    match = re.search(r"## API Simulation\s+- status:\s*(\S+)", text, flags=re.MULTILINE)
    return match.group(1).strip() if match else ""


def first_token(value: str) -> str:
    parts = value.split()
    return parts[0] if parts else ""


def parse_int(value: str, default: int = 0) -> int:
    try:
        return int(str(value).strip())
    except Exception:
        return default


def extract_artifact_quality_warnings(quality_payload: dict[str, Any]) -> list[str]:
    warnings = quality_payload.get("run_warnings") or []
    if not isinstance(warnings, list):
        return []
    extracted: list[str] = []
    for warning in warnings:
        text = str(warning).strip()
        if text.startswith(ARTIFACT_QUALITY_WARNING_PREFIX):
            extracted.append(text)
    return extracted


def extract_quality_runtime_counters(quality_payload: dict[str, Any]) -> dict[str, int]:
    totals = quality_payload.get("totals") or {}
    if not isinstance(totals, dict):
        totals = {}
    counters = {key: parse_int(totals.get(key, 0), 0) for key in QUALITY_COUNTER_KEYS}
    signals = quality_payload.get("quality_signals") or []
    counters["quality_alerts"] = len(signals) if isinstance(signals, list) else 0
    return counters


def focused_recovery_counter_floor(reason_counts: Counter[str]) -> tuple[int, int]:
    focused = sum(reason_counts.values())
    exhausted = sum(count for tag, count in reason_counts.items() if tag in FOCUSED_REPAIR_EXHAUSTED_REASON_TAGS)
    return focused, exhausted


def extract_raw_runtime_stall_counters(paths: list[Path]) -> Counter[str]:
    counters: Counter[str] = Counter()
    for path in paths:
        if not path.exists() or not path.name.endswith("-meta.json"):
            continue
        try:
            payload = read_json(path)
        except Exception:
            continue
        diagnostics = payload.get("diagnostics") or {}
        if not isinstance(diagnostics, dict):
            continue
        lifecycle = diagnostics.get("provider_lifecycle") or {}
        if not isinstance(lifecycle, dict):
            lifecycle = {}
        exit_reason = str(lifecycle.get("exit_reason", "")).strip()
        error_text = str(lifecycle.get("error", "")).strip()
        if exit_reason != "stall" and "runtime_stalled" not in error_text:
            continue
        stall_phase = str(diagnostics.get("stall_phase", "")).strip()
        if not stall_phase:
            if "runtime_stalled_before_artifacts" in error_text:
                stall_phase = "pre_artifact"
            elif "runtime_stalled_after_artifacts" in error_text:
                stall_phase = "post_artifact"
        counters["stall_count"] += 1
        if stall_phase == "pre_artifact":
            counters["pre_artifact_stalls"] += 1
        elif stall_phase == "post_artifact":
            counters["post_artifact_stalls"] += 1
        stdout_bytes = parse_int(lifecycle.get("stdout_bytes", diagnostics.get("stdout_bytes", 0)), 0)
        stderr_bytes = parse_int(lifecycle.get("stderr_bytes", diagnostics.get("stderr_bytes", 0)), 0)
        artifact_observed = bool(diagnostics.get("artifact_observed", False))
        authored_count = parse_int(diagnostics.get("authored_file_count", 0), 0)
        if (
            stall_phase == "pre_artifact"
            and stdout_bytes == 0
            and stderr_bytes == 0
            and not artifact_observed
            and authored_count == 0
        ):
            counters["zero_output_pre_artifact_stalls"] += 1
    return counters


def parse_run_results(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    if not path.exists():
        return rows
    for raw in read_text_file(path).splitlines():
        if not raw.strip():
            continue
        parts = raw.split("\t")
        if len(parts) < len(RUN_RESULTS_COLUMNS):
            continue
        record = dict(zip(RUN_RESULTS_COLUMNS, parts))
        for numeric_key in ("signal", "entities", "edges", "findings", "questions", "cov_obs", "cov_missing", "warnings"):
            try:
                record[numeric_key] = int(record[numeric_key])
            except Exception:
                record[numeric_key] = 0
        rows.append(record)
    return rows


def parse_backend_classifications(batch_root: Path) -> dict[tuple[str, int], dict[str, str]]:
    path = batch_root / "backend-run-classifications.tsv"
    if not path.exists():
        return {}
    lines = [line for line in read_text_file(path).splitlines() if line.strip()]
    if len(lines) <= 1:
        return {}
    header = lines[0].split("\t")
    index = {name: idx for idx, name in enumerate(header)}
    provider_idx = index.get("provider")
    run_idx = index.get("run_index")
    if provider_idx is None or run_idx is None:
        return {}
    result: dict[tuple[str, int], dict[str, str]] = {}
    for line in lines[1:]:
        parts = line.split("\t")
        if len(parts) <= max(provider_idx, run_idx):
            continue
        provider = parts[provider_idx].strip()
        try:
            run_index = int(parts[run_idx].strip())
        except Exception:
            continue
        row: dict[str, str] = {}
        for name, idx in index.items():
            if idx < len(parts):
                row[name] = parts[idx].strip()
        result[(provider, run_index)] = row
    return result


def parse_run_status_file(path: Path) -> dict[str, str]:
    payload: dict[str, str] = {}
    if not path.exists():
        return payload
    for line in read_text_file(path).splitlines():
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        if not key:
            continue
        payload[key] = value.strip()
    return payload


def parse_run_history_status_counts(path: Path) -> dict[str, int]:
    counts: dict[str, int] = {}
    if not path.exists():
        return counts
    try:
        payload = read_json(path)
    except Exception:
        return counts
    items = []
    if isinstance(payload, dict):
        items = payload.get("items") or payload.get("runs") or []
    elif isinstance(payload, list):
        items = payload
    if not isinstance(items, list):
        return counts
    for item in items:
        if not isinstance(item, dict):
            continue
        status = str(item.get("status", "")).strip().lower()
        if not status:
            continue
        counts[status] = counts.get(status, 0) + 1
    return counts


def reconstruct_backend_classifications(
    batch_root: Path, classifications: dict[tuple[str, int], dict[str, str]]
) -> dict[tuple[str, int], dict[str, str]]:
    merged = dict(classifications)
    for status_path in sorted(batch_root.glob("*/run*/run-status.env")):
        status_payload = parse_run_status_file(status_path)
        provider = str(status_payload.get("provider", status_path.parent.parent.name)).strip()
        if provider not in FRONTEND_PROVIDERS:
            continue
        try:
            run_index = int(str(status_payload.get("run_index", status_path.parent.name.replace("run", ""))).strip())
        except Exception:
            continue
        key = (provider, run_index)
        if key in merged:
            continue

        run_dir = status_path.parent
        workspace, _ = resolve_workspace(run_dir)
        run_history_counts = parse_run_history_status_counts(workspace / "reports" / "taskruns" / "run-history.json")
        if (run_dir / "session-summary.md").exists():
            continue

        state = str(status_payload.get("state", "")).strip()
        termination_signal = str(status_payload.get("termination_signal", "")).strip()
        failure_reason = str(status_payload.get("failure_reason", "")).strip()
        process_exit = str(status_payload.get("process_exit", "")).strip() or "1"
        try:
            process_exit_int = int(process_exit)
        except Exception:
            process_exit_int = 1
            process_exit = "1"

        failure_class = "infra_incomplete_cycle"
        if (
            failure_reason == "infra_signal_terminated"
            or state == "signal_terminated"
            or (termination_signal and termination_signal != "none")
            or process_exit_int >= 128
        ):
            failure_class = "infra_signal_terminated"

        merged[key] = {
            "provider": provider,
            "run_index": str(run_index),
            "failure_class": failure_class,
            "failure_subclass": "none",
            "cancellation_like": "0",
            "process_exit": process_exit,
            "summary_result": "missing",
            "failure_reason": failure_reason or failure_class,
            "termination_signal": termination_signal or ("none" if failure_class != "infra_signal_terminated" else "unknown"),
            "run_history_running": str(run_history_counts.get("running", 0)),
        }
    return merged


def normalize_selected_providers(values: Any) -> list[str]:
    if not isinstance(values, list):
        return []
    selected: list[str] = []
    seen: set[str] = set()
    for value in values:
        provider = str(value).strip()
        if provider not in FRONTEND_PROVIDERS or provider in seen:
            continue
        seen.add(provider)
        selected.append(provider)
    return selected


def normalize_selected_run_indexes(values: Any) -> list[int]:
    if not isinstance(values, list):
        return []
    selected: list[int] = []
    seen: set[int] = set()
    for value in values:
        try:
            run_index = int(str(value).strip())
        except Exception:
            continue
        if run_index <= 0 or run_index in seen:
            continue
        seen.add(run_index)
        selected.append(run_index)
    selected.sort()
    return selected


def resolve_selected_providers(
    preflight: dict[str, Any], classifications: dict[tuple[str, int], dict[str, str]], batch_root: Path
) -> list[str]:
    selected = normalize_selected_providers(preflight.get("selected_providers"))
    if selected:
        return selected

    classified = sorted({provider for provider, _ in classifications.keys() if provider in FRONTEND_PROVIDERS})
    if classified:
        return classified

    discovered: list[str] = []
    for provider in FRONTEND_PROVIDERS:
        provider_root = batch_root / provider
        if provider_root.exists():
            discovered.append(provider)
    return discovered or list(FRONTEND_PROVIDERS)


def resolve_selected_run_indexes(
    preflight: dict[str, Any], classifications: dict[tuple[str, int], dict[str, str]], batch_root: Path
) -> list[int]:
    selected = normalize_selected_run_indexes(preflight.get("selected_run_indexes"))
    if selected:
        return selected

    classified = sorted({run_index for _, run_index in classifications.keys() if run_index > 0})
    if classified:
        return classified

    discovered: set[int] = set()
    for provider in FRONTEND_PROVIDERS:
        provider_root = batch_root / provider
        if not provider_root.exists():
            continue
        for path in provider_root.glob("run*"):
            match = re.fullmatch(r"run([1-9][0-9]*)", path.name)
            if match:
                discovered.add(int(match.group(1)))
    return sorted(discovered) or [1, 2, 3, 4, 5]


def parse_markdown_section_bullets(path: Path, section_title: str) -> list[str]:
    if not path.exists():
        return []
    lines = read_text_file(path).splitlines()
    bullets: list[str] = []
    in_section = False
    for line in lines:
        if line.startswith("## "):
            in_section = line.strip().lower() == f"## {section_title.lower()}"
            continue
        if in_section and line.strip().startswith("- "):
            bullets.append(line.strip()[2:].strip("`"))
    return bullets


def parse_open_questions(path: Path) -> tuple[list[str], list[str]]:
    if not path.exists():
        return [], []
    ids: list[str] = []
    texts: list[str] = []
    for line in read_text_file(path).splitlines():
        line = line.strip()
        if not line.startswith("- "):
            continue
        payload = line[2:].strip()
        match = re.match(r"`([^`]+)`\s*(.*)$", payload)
        if match:
            ids.append(match.group(1).strip())
            texts.append(match.group(2).strip())
        else:
            texts.append(payload)
    return ids, texts


def report_has_incomplete_banner(text: str) -> bool:
    return "Analysis incomplete." in text or "Partial analysis. Some shards failed; downstream content may be incomplete." in text


def findings_has_incomplete_fallback(text: str) -> bool:
    return "Findings unavailable because analysis did not complete." in text or "Findings may be incomplete because some shards failed." in text


def coverage_has_incomplete_fallback(text: str) -> bool:
    markers = (
        "Unavailable due to incomplete analysis.",
        "Unknown due to incomplete analysis.",
        "May be incomplete because some shards failed.",
        "Analysis incomplete. See banner above.",
    )
    return any(marker in text for marker in markers)


def open_questions_has_incomplete_fallback(text: str) -> bool:
    markers = (
        "Open questions unavailable due to incomplete analysis.",
        "Open questions may be incomplete because some shards failed.",
    )
    return any(marker in text for marker in markers)


def parse_headless_rows(rows: list[dict[str, Any]], provider: str) -> dict[str, dict[str, Any]]:
    result: dict[str, dict[str, Any]] = {}
    for row in rows:
        if row.get("runtime_mode") != "headless":
            continue
        if row.get("runtime_provider") != provider:
            continue
        pipeline = row.get("pipeline", "")
        if pipeline in ("init", "refresh"):
            result[pipeline] = row
    return result


def normalize_execution_profile(preflight: dict[str, Any]) -> dict[str, Any] | None:
    payload = preflight.get("execution_profile")
    if not isinstance(payload, dict):
        return None
    effective = payload.get("effective")
    if not isinstance(effective, dict):
        return None

    defaults = {
        "strategy": "sequential",
        "max_parallel_tasks": 1,
        "failure_policy": "best_effort",
        "shard_discovery_mode": "heuristics",
    }
    allowed = {
        "strategy": {"sequential", "parallel"},
        "failure_policy": {"fail_fast", "best_effort"},
        "shard_discovery_mode": {"heuristics", "semantic"},
    }

    strategy_raw = str(effective.get("strategy", defaults["strategy"])).strip()
    strategy = strategy_raw if strategy_raw in allowed["strategy"] else defaults["strategy"]
    max_parallel_raw = effective.get("max_parallel_tasks", defaults["max_parallel_tasks"])
    try:
        max_parallel = int(max_parallel_raw)
    except Exception:
        max_parallel = defaults["max_parallel_tasks"]
    if max_parallel <= 0:
        max_parallel = defaults["max_parallel_tasks"]
    if strategy != "parallel":
        max_parallel = 1

    failure_policy_raw = str(effective.get("failure_policy", defaults["failure_policy"])).strip()
    failure_policy = (
        failure_policy_raw if failure_policy_raw in allowed["failure_policy"] else defaults["failure_policy"]
    )
    shard_mode_raw = str(effective.get("shard_discovery_mode", defaults["shard_discovery_mode"])).strip()
    shard_mode = shard_mode_raw if shard_mode_raw in allowed["shard_discovery_mode"] else defaults["shard_discovery_mode"]

    return {
        "strategy": strategy,
        "max_parallel_tasks": max_parallel,
        "failure_policy": failure_policy,
        "shard_discovery_mode": shard_mode,
    }


def normalize_declared_repos_meta(preflight: dict[str, Any]) -> dict[str, Any]:
    meta = preflight.get("declared_repos_meta")
    if isinstance(meta, dict):
        declared = meta.get("declared_repos")
        if isinstance(declared, list) and declared:
            expected = meta.get("expected_repo_count", len(declared))
            try:
                expected_count = int(expected)
            except Exception:
                expected_count = len(declared)
            if expected_count <= 0:
                expected_count = len(declared)
            return {
                "target_repos_file": str(meta.get("target_repos_file", preflight.get("target_repos_file", "-"))),
                "profile_id": str(meta.get("profile_id", preflight.get("profile_id", "adhoc"))),
                "profile_source_kind": str(meta.get("profile_source_kind", preflight.get("profile_source_kind", "mixed"))),
                "expected_repo_count": expected_count,
                "declared_repos": declared,
            }
    expected = preflight.get("expected_repo_count", 0)
    try:
        expected_count = int(expected)
    except Exception:
        expected_count = 0
    if expected_count < 0:
        expected_count = 0
    return {
        "target_repos_file": str(preflight.get("target_repos_file", "-")),
        "profile_id": str(preflight.get("profile_id", "adhoc")),
        "profile_source_kind": str(preflight.get("profile_source_kind", "mixed")),
        "expected_repo_count": expected_count,
        "declared_repos": [],
    }


def parse_workspace_validate_resolved_roots(run_dir: Path) -> list[Path]:
    validate_path = run_dir / "workspace-validate.json"
    if not validate_path.exists():
        return []
    try:
        payload = read_json(validate_path)
    except Exception:
        return []
    repos = payload.get("resolved_repos") or []
    roots: list[Path] = []
    for item in repos:
        if not isinstance(item, dict):
            continue
        raw_path = str(item.get("path", "")).strip()
        if not raw_path:
            continue
        path_obj = Path(raw_path).expanduser().resolve()
        if path_obj.exists() and path_obj.is_dir():
            roots.append(path_obj)
    deduped: list[Path] = []
    seen: set[str] = set()
    for root in roots:
        key = str(root)
        if key in seen:
            continue
        seen.add(key)
        deduped.append(root)
    return deduped


def collect_repo_roots(run_dir: Path, declared_meta: dict[str, Any]) -> list[Path]:
    roots = parse_workspace_validate_resolved_roots(run_dir)
    if roots:
        return roots
    declared = declared_meta.get("declared_repos") or []
    declared_roots: list[Path] = []
    for item in declared:
        if not isinstance(item, dict):
            continue
        if str(item.get("source", "")).strip() != "path":
            continue
        raw_path = str(item.get("path", "")).strip()
        if not raw_path:
            continue
        path_obj = Path(raw_path).expanduser().resolve()
        if path_obj.exists() and path_obj.is_dir():
            declared_roots.append(path_obj)
    deduped: list[Path] = []
    seen: set[str] = set()
    for root in declared_roots:
        key = str(root)
        if key in seen:
            continue
        seen.add(key)
        deduped.append(root)
    return deduped


def resolve_reports_root(run_dir: Path, run_id: str) -> tuple[Path, str]:
    snapshot_reports = run_dir / "snapshots" / run_id / "reports"
    if snapshot_reports.exists():
        return snapshot_reports, "snapshot"
    for workspace_root in workspace_candidates(run_dir):
        reports_root = workspace_root / "reports"
        if (reports_root / "taskruns" / f"{run_id}-quality.json").exists():
            return reports_root, "workspace"
        if (reports_root / "taskruns" / run_id).exists():
            return reports_root, "workspace"
    return snapshot_reports, "snapshot"


def resolve_quality_json(run_dir: Path, row: dict[str, Any]) -> tuple[Path, str]:
    run_id = str(row.get("run_id", "")).strip()
    reports_root, source = resolve_reports_root(run_dir, run_id)
    return reports_root / "taskruns" / f"{run_id}-quality.json", source


def resolve_step_taskrun_files(run_dir: Path, run_id: str, pipeline: str) -> tuple[list[Path], str]:
    reports_root, source = resolve_reports_root(run_dir, run_id)
    files = sorted((reports_root / "taskruns" / run_id).glob("**/runtime-execution.json"))
    return files, source


def resolve_step_semantic_files(run_dir: Path, run_id: str) -> list[Path]:
    reports_root, _ = resolve_reports_root(run_dir, run_id)
    taskruns_root = reports_root / "taskruns" / run_id
    files: list[Path] = []
    files.extend(sorted((taskruns_root / "staging" / "shards").glob("*/shard-pack-manifest.json")))
    files.extend(sorted((taskruns_root / "staging" / "final").glob("final-run-index.json")))
    files.extend(sorted((taskruns_root / "validator").glob("validator-verdict.json")))
    return [path for path in files if path.exists()]


def is_within(path: Path, root: Path) -> bool:
    try:
        path.resolve().relative_to(root.resolve())
        return True
    except Exception:
        return False


def collect_evidence_paths(payload: Any) -> list[str]:
    result: list[str] = []

    def walk(node: Any) -> None:
        if isinstance(node, dict):
            provenance = node.get("provenance")
            if isinstance(provenance, dict):
                evidence = provenance.get("evidence")
                if isinstance(evidence, list):
                    for item in evidence:
                        if isinstance(item, dict):
                            path_value = item.get("path")
                            if isinstance(path_value, str):
                                result.append(path_value.strip())
            for value in node.values():
                walk(value)
            return
        if isinstance(node, list):
            for item in node:
                walk(item)

    walk(payload)
    return result


def normalize_evidence_path(path_value: str) -> str:
    raw = path_value.strip().strip("`")
    if not raw:
        return ""
    # Allow evidence values with URL/hash/query/line suffix decorations.
    base = raw.split("?", 1)[0].split("#", 1)[0].strip()
    if not base:
        return ""
    line_suffix = re.match(r"^(.*?):\d+(?::\d+)?$", base)
    if line_suffix:
        prefix = line_suffix.group(1)
        # Keep Windows drive form like C:\... intact.
        if not re.match(r"^[A-Za-z]:[\\/]", prefix):
            base = prefix
    return base.strip()


def evidence_path_resolves(path_value: str, repo_roots: list[Path], workspace: Path) -> tuple[bool, str]:
    raw = path_value.strip()
    if not raw:
        return False, "empty path"
    normalized_raw = normalize_text(raw)
    raw_lower = raw.lower()
    normalized_candidate = normalize_evidence_path(raw)
    if not normalized_candidate:
        return False, "empty normalized path"
    normalized_candidate_text = normalize_text(normalized_candidate)
    candidate_lower = normalized_candidate.lower()
    if raw in ("/", ".", ".."):
        return False, "path points to root/relative marker"
    if any(raw_lower.startswith(prefix) for prefix in SYNTHETIC_EVIDENCE_PREFIXES):
        return False, "synthetic prefix"
    if any(normalized_raw.startswith(normalize_text(prefix)) for prefix in SYNTHETIC_EVIDENCE_PREFIXES):
        return False, "synthetic prefix"
    if any(candidate_lower.startswith(prefix) for prefix in SYNTHETIC_EVIDENCE_PREFIXES):
        return False, "synthetic prefix"
    if any(normalized_candidate_text.startswith(normalize_text(prefix)) for prefix in SYNTHETIC_EVIDENCE_PREFIXES):
        return False, "synthetic prefix"
    if re.match(r"^[a-zA-Z][a-zA-Z0-9+.\-]*://", raw):
        return False, "uri-like path"
    if re.match(r"^[a-zA-Z][a-zA-Z0-9+.\-]*://", normalized_candidate):
        return False, "uri-like path"

    candidate = Path(normalized_candidate)
    if candidate.is_absolute():
        if not candidate.exists():
            return False, "absolute path missing"
        if any(is_within(candidate, root) for root in repo_roots) or is_within(candidate, workspace):
            return True, "ok"
        return False, "absolute path outside repos/workspace"

    candidate_variants = [candidate]
    parts = candidate.parts
    if parts and parts[0] in {"arch-workspace", "workspace"} and len(parts) > 1:
        candidate_variants.append(Path(*parts[1:]))

    roots = [workspace, *repo_roots]
    ambiguous_extensionless = False
    for variant in candidate_variants:
        for root in roots:
            resolved = root / variant
            if resolved.exists():
                return True, "ok"
            extension_matches = unique_extensionless_path_matches(resolved)
            if len(extension_matches) == 1:
                return True, "ok"
            if len(extension_matches) > 1:
                ambiguous_extensionless = True
    if ambiguous_extensionless:
        return False, "ambiguous extensionless relative path"
    return False, "relative path missing in repos/workspace"


def unique_extensionless_path_matches(resolved: Path) -> list[Path]:
    if resolved.suffix:
        return []
    parent = resolved.parent
    if not parent.exists() or not parent.is_dir():
        return []
    prefix = resolved.name + "."
    matches = [child for child in parent.iterdir() if child.is_file() and child.name.startswith(prefix)]
    return sorted(matches)


def is_power_target(repo_roots: list[Path], declared_meta: dict[str, Any]) -> bool:
    fragments: list[str] = []
    for root in repo_roots:
        fragments.append(str(root))
        fragments.append(root.name)
    for item in declared_meta.get("declared_repos") or []:
        if not isinstance(item, dict):
            continue
        for key in ("name", "path", "git_url", "ref"):
            value = str(item.get(key, "")).strip()
            if value:
                fragments.append(value)
    corpus = "\n".join(fragments).lower()
    return any(hint in corpus for hint in POWER_TARGET_HINTS)


def collect_repo_mentions(payload: dict[str, Any]) -> set[str]:
    mentions: set[str] = set()
    meta = payload.get("meta") or {}
    repo_scopes = meta.get("repo_scopes")
    if isinstance(repo_scopes, list):
        for item in repo_scopes:
            value = str(item).strip()
            if value:
                mentions.add(normalize_text(value))
    citations = payload.get("citations")
    if isinstance(citations, list):
        for item in citations:
            if not isinstance(item, dict):
                continue
            value = str(item.get("repo", "")).strip()
            if value:
                mentions.add(normalize_text(value))

    def walk(node: Any) -> None:
        if isinstance(node, dict):
            provenance = node.get("provenance")
            if isinstance(provenance, dict):
                evidence = provenance.get("evidence")
                if isinstance(evidence, list):
                    for item in evidence:
                        if isinstance(item, dict):
                            repo_name = str(item.get("repo", "")).strip()
                            if repo_name:
                                mentions.add(normalize_text(repo_name))
            for value in node.values():
                walk(value)
            return
        if isinstance(node, list):
            for item in node:
                walk(item)

    walk(payload)
    return mentions


def count_semantic_edges(payload: dict[str, Any]) -> int:
    semantic = payload.get("semantic") or {}
    if not isinstance(semantic, dict):
        return 0
    edges = semantic.get("edges") or []
    return len(edges) if isinstance(edges, list) else 0


def count_cross_repo_semantic_links(payload: dict[str, Any]) -> int:
    semantic = payload.get("semantic") or {}
    payload_repo_mentions = collect_repo_mentions(payload)
    findings: list[Any] = []
    questions: list[Any] = []
    if isinstance(semantic, dict):
        semantic_findings = semantic.get("findings") or []
        if isinstance(semantic_findings, list):
            findings.extend(semantic_findings)
        semantic_questions = semantic.get("questions") or []
        if isinstance(semantic_questions, list):
            questions.extend(semantic_questions)
    top_level_findings = payload.get("findings") or []
    if isinstance(top_level_findings, list):
        findings.extend(top_level_findings)
    top_level_questions = payload.get("questions") or []
    if isinstance(top_level_questions, list):
        questions.extend(top_level_questions)
    count = 0
    for finding in findings:
        if not isinstance(finding, dict):
            continue
        related = {
            normalize_text(str(item))
            for item in (finding.get("related_ids") or [])
            if str(item).strip()
        }
        repos: set[str] = set()
        provenance = finding.get("provenance")
        if isinstance(provenance, dict):
            evidence = provenance.get("evidence") or []
            if isinstance(evidence, list):
                for item in evidence:
                    if isinstance(item, dict):
                        repo_name = str(item.get("repo", "")).strip()
                        if repo_name:
                            repos.add(normalize_text(repo_name))
        if len(repos) >= 2 and len(related | repos) >= 2:
            count += 1
    for question in questions:
        if not isinstance(question, dict):
            continue
        related = {
            normalize_text(str(item))
            for item in (question.get("related_ids") or [])
            if str(item).strip()
        }
        if len(related & payload_repo_mentions) >= 2:
            count += 1
    return count


def collect_off_topic_hits(payload: dict[str, Any]) -> list[str]:
    fragments: list[str] = []
    summary = str(payload.get("summary", "")).strip()
    if summary:
        fragments.append(summary)

    semantic = payload.get("semantic") or {}
    if not isinstance(semantic, dict):
        semantic = {}

    questions = semantic.get("questions") or []
    if isinstance(questions, list):
        for question in questions:
            if isinstance(question, dict):
                text = str(question.get("text", "")).strip()
                if text:
                    fragments.append(text)

    entities = semantic.get("entities") or []
    if isinstance(entities, list):
        for entity in entities:
            if not isinstance(entity, dict):
                continue
            fragments.extend(
                [
                    str(entity.get("id", "")).strip(),
                    str(entity.get("type", "")).strip(),
                    str(entity.get("name", "")).strip(),
                    json.dumps(entity.get("attributes", {}), ensure_ascii=False),
                ]
            )

    corpus = "\n".join(fragment for fragment in fragments if fragment).lower()
    if not corpus:
        return []
    hits = [term for term in OFF_TOPIC_TERMS if term in corpus]
    return sorted(set(hits))


def parse_overview_counts(path: Path) -> dict[str, int]:
    counts: dict[str, int] = {}
    if not path.exists():
        return counts
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        match = re.match(r"^- ([^:]+):\s*([0-9]+)$", line)
        if not match:
            continue
        label = normalize_text(match.group(1))
        counts[label] = int(match.group(2))
    return counts


def collect_runtime_taskrun_payloads(taskruns_root: Path, run_id: str, pipeline: str) -> list[tuple[Path, dict[str, Any]]]:
    candidates = sorted((taskruns_root / run_id).glob("**/runtime-execution.json"))
    result: list[tuple[Path, dict[str, Any]]] = []
    for candidate in candidates:
        try:
            payload = read_json(candidate)
        except Exception:
            continue
        if not isinstance(payload, dict):
            continue
        meta = payload.get("meta") if isinstance(payload.get("meta"), dict) else {}
        step_id = str(payload.get("step_id") or meta.get("step_id") or "").strip()
        if step_id and not step_id.startswith(f"{pipeline}."):
            continue
        result.append((candidate, payload))
    return result


def evaluate_runtime_flow_checks(
    run_dir: Path,
    workspace: Path,
    headless_rows: dict[str, dict[str, Any]],
    expected_execution: dict[str, Any],
    summary_text: str,
    full_run_log_text: str,
    runner_unavailable_signal: bool = False,
) -> tuple[set[str], list[str]]:
    issues: set[str] = set()
    details: list[str] = []
    inspected_run_ids: set[str] = set()

    for pipeline in ("init", "refresh"):
        row = headless_rows.get(pipeline)
        if not row:
            issues.add("runtime:shard-artifacts")
            details.append(f"runtime/shard-artifacts -> missing headless {pipeline} row in run-results.tsv")
            continue

        run_id = str(row.get("run_id", "")).strip()
        if not run_id:
            issues.add("runtime:shard-artifacts")
            details.append(f"runtime/shard-artifacts -> {pipeline} row has empty run_id")
            continue

        reports_root, _ = resolve_reports_root(run_dir, run_id)
        taskruns_root = reports_root / "taskruns"
        plan_files = sorted(taskruns_root.glob(f"{run_id}-{pipeline}-*-shard-plan*.json"))
        summary_files = sorted(taskruns_root.glob(f"{run_id}-{pipeline}-*-shard-summary*.json"))
        runtime_taskruns = collect_runtime_taskrun_payloads(taskruns_root, run_id, pipeline)

        if not plan_files:
            issues.add("runtime:shard-artifacts")
            details.append(f"runtime/shard-artifacts -> missing shard-plan for run_id={run_id} pipeline={pipeline}")
        if not summary_files:
            issues.add("runtime:shard-artifacts")
            details.append(f"runtime/shard-artifacts -> missing shard-summary for run_id={run_id} pipeline={pipeline}")
        if not runtime_taskruns:
            issues.add("runtime:shard-artifacts")
            details.append(f"runtime/shard-artifacts -> missing runtime-execution metadata for run_id={run_id} pipeline={pipeline}")

        missing_shard_meta = []
        for taskrun_path, payload in runtime_taskruns:
            meta = payload.get("meta") if isinstance(payload.get("meta"), dict) else {}
            step_id = str(payload.get("step_id") or meta.get("step_id") or "").strip()
            shard_id = str(payload.get("shard_id") or meta.get("shard_id") or "").strip()
            repo_scopes = payload.get("repo_scopes") if isinstance(payload.get("repo_scopes"), list) else meta.get("repo_scopes")
            path_scopes = payload.get("path_scopes") if isinstance(payload.get("path_scopes"), list) else meta.get("path_scopes")
            if (
                (step_id.endswith("step1.collect") and not shard_id)
                or not isinstance(repo_scopes, list)
                or not isinstance(path_scopes, list)
                or len(repo_scopes) == 0
                or len(path_scopes) == 0
            ):
                missing_shard_meta.append(taskrun_path)
        if missing_shard_meta:
            issues.add("runtime:shard-metadata")
            for path in missing_shard_meta[:8]:
                details.append(f"runtime/shard-metadata -> {path}: require meta.shard_id/meta.repo_scopes/meta.path_scopes")
            if len(missing_shard_meta) > 8:
                details.append(f"runtime/shard-metadata -> +{len(missing_shard_meta) - 8} additional runtime-execution artifacts with missing shard metadata")

        for artifact_path in [*plan_files, *summary_files]:
            try:
                payload = read_json(artifact_path)
            except Exception as exc:
                issues.add("runtime:execution-semantics")
                details.append(f"runtime/execution-semantics -> {artifact_path}: invalid json ({exc})")
                continue

            strategy = str(payload.get("strategy", "")).strip() or "sequential"
            max_parallel = payload.get("max_parallel_tasks", 1)
            failure_policy = str(payload.get("failure_policy", "")).strip() or "best_effort"
            shard_mode = str(payload.get("shard_discovery_mode", "")).strip() or "heuristics"
            try:
                max_parallel_int = int(max_parallel)
            except Exception:
                max_parallel_int = 0

            mismatches: list[str] = []
            if strategy != expected_execution["strategy"]:
                mismatches.append(f"strategy expected={expected_execution['strategy']} got={strategy}")
            if max_parallel_int != int(expected_execution["max_parallel_tasks"]):
                mismatches.append(
                    f"max_parallel_tasks expected={expected_execution['max_parallel_tasks']} got={max_parallel_int}"
                )
            if failure_policy != expected_execution["failure_policy"]:
                mismatches.append(f"failure_policy expected={expected_execution['failure_policy']} got={failure_policy}")
            if shard_mode != expected_execution["shard_discovery_mode"]:
                mismatches.append(
                    f"shard_discovery_mode expected={expected_execution['shard_discovery_mode']} got={shard_mode}"
                )
            if mismatches:
                issues.add("runtime:execution-semantics")
                details.append(f"runtime/execution-semantics -> {artifact_path}: {'; '.join(mismatches)}")

        summary_has_failed_items = False
        for summary_path in summary_files:
            try:
                payload = read_json(summary_path)
            except Exception:
                continue
            items = payload.get("items")
            if not isinstance(items, list):
                continue
            if any(str(item.get("status", "")).strip() == "failed" for item in items if isinstance(item, dict)):
                summary_has_failed_items = True
                break
        if summary_has_failed_items:
            summary_blob = (summary_text + "\n" + full_run_log_text).lower()
            if expected_execution["failure_policy"] == "best_effort":
                if "run_partial_failed" not in summary_blob and not runner_unavailable_signal:
                    issues.add("runtime:execution-semantics")
                    details.append(
                        f"runtime/execution-semantics -> run_id={run_id} has failed shard items under best_effort but missing run_partial_failed signal"
                    )
                elif "run_partial_failed" not in summary_blob and runner_unavailable_signal:
                    details.append(
                        f"runtime/execution-semantics -> run_id={run_id} has failed shard items under best_effort with provider-unavailable signal; skip run_partial_failed enforcement"
                    )
            else:
                issues.add("runtime:execution-semantics")
                details.append(
                    f"runtime/execution-semantics -> run_id={run_id} has failed shard items while failure_policy={expected_execution['failure_policy']}"
                )

        if run_id in inspected_run_ids:
            continue
        inspected_run_ids.add(run_id)

    return issues, details


def bool_score(value: bool, weight: int) -> int:
    return weight if value else 0


def verdict(total: int) -> str:
    if total >= 85:
        return "Excellent"
    if total >= 70:
        return "Good"
    if total >= 50:
        return "Fair"
    return "Poor"


@dataclass
class RunEvaluation:
    provider: str
    run_index: int
    run_dir: Path
    hard_pass: bool
    reliability: int
    contract: int
    analysis: int
    total: int
    verdict: str
    init_signal: int = 0
    refresh_signal: int = 0
    refresh_findings: int = 0
    refresh_questions: int = 0
    refresh_cov_missing: int = 0
    repair_attempts: int = 0
    repair_exhausted: int = 0
    fresh_retries: int = 0
    focused_repairs: int = 0
    stall_count: int = 0
    pre_artifact_stalls: int = 0
    post_artifact_stalls: int = 0
    zero_output_pre_artifact_stalls: int = 0
    partial_failure_count: int = 0
    quality_alerts: int = 0
    artifact_source: str = "snapshot"
    semantic_hard_fail: bool = False
    off_topic_hits: int = 0
    failure_class: str = "none"
    runtime_contract_failed: bool = False
    runner_unavailable: bool = False
    runtime_timeout: bool = False
    infra_signal_terminated: bool = False
    infra_incomplete_cycle: bool = False
    summary_missing: bool = False
    precheck_failed: bool = False
    runtime_flow_failed: bool = False
    cancellation_like: bool = False
    artifact_quality_findings: int = 0
    issues: list[str] = field(default_factory=list)
    issue_details: list[str] = field(default_factory=list)
    error_codes: list[str] = field(default_factory=list)


def evaluate_run(
    provider: str,
    run_index: int,
    run_dir: Path,
    preflight: dict[str, Any],
    classification_row: dict[str, str] | None = None,
) -> RunEvaluation:
    summary_path = run_dir / "session-summary.md"
    run_results_path = run_dir / "run-results.tsv"
    full_run_log = run_dir / "full-run.log"
    workspace, workspace_roots = resolve_workspace(run_dir)
    declared_meta = normalize_declared_repos_meta(preflight)
    expected_repo_count = int(declared_meta.get("expected_repo_count", 1))
    repo_roots = collect_repo_roots(run_dir, declared_meta)
    expected_execution = normalize_execution_profile(preflight)

    issues: list[str] = []
    details: list[str] = []
    error_codes: list[str] = []
    classification_row = classification_row or {}
    classified_failure = str(classification_row.get("failure_class", "")).strip()
    classified_subclass = str(classification_row.get("failure_subclass", "")).strip()
    cancellation_like = str(classification_row.get("cancellation_like", "")).strip() in {"1", "true", "yes"}
    if classified_subclass == "cancellation_like":
        cancellation_like = True

    if classified_failure == "precheck_failed":
        issues.append("reliability:precheck-failed")
        details.append(
            f"reliability/precheck-failed -> failure_class={classified_failure} process_exit={classification_row.get('process_exit', '-')}"
        )
        for precheck_log in (
            run_dir.parent.parent / "precheck-node-toolchain.log",
            run_dir.parent.parent / "precheck-make.log",
        ):
            if precheck_log.exists():
                details.append(f"reliability/precheck-failed -> precheck_log={precheck_log}")
        return RunEvaluation(
            provider=provider,
            run_index=run_index,
            run_dir=run_dir,
            hard_pass=False,
            reliability=0,
            contract=0,
            analysis=0,
            total=0,
            verdict=verdict(0),
            artifact_source="precheck",
            semantic_hard_fail=False,
            off_topic_hits=0,
            failure_class="precheck_failed",
            runtime_contract_failed=False,
            runner_unavailable=False,
            runtime_timeout=False,
            infra_signal_terminated=False,
            infra_incomplete_cycle=False,
            summary_missing=False,
            precheck_failed=True,
            runtime_flow_failed=False,
            cancellation_like=cancellation_like,
            artifact_quality_findings=0,
            issues=issues,
            issue_details=details,
            error_codes=[],
        )

    if classified_failure == "operational_host_preflight_failed":
        issues.append("reliability:operational-host-preflight-failed")
        details.append(
            "reliability/operational-host-preflight-failed -> "
            f"failure_reason={classification_row.get('failure_reason', '-')}"
        )
        return RunEvaluation(
            provider=provider,
            run_index=run_index,
            run_dir=run_dir,
            hard_pass=False,
            reliability=0,
            contract=0,
            analysis=0,
            total=0,
            verdict=verdict(0),
            artifact_source="preflight",
            semantic_hard_fail=False,
            off_topic_hits=0,
            failure_class="operational_host_preflight_failed",
            runtime_contract_failed=False,
            runner_unavailable=False,
            runtime_timeout=False,
            infra_signal_terminated=False,
            infra_incomplete_cycle=False,
            summary_missing=False,
            precheck_failed=False,
            runtime_flow_failed=False,
            cancellation_like=cancellation_like,
            artifact_quality_findings=0,
            issues=issues,
            issue_details=details,
            error_codes=[],
        )

    summary_text = read_text_file(summary_path) if summary_path.exists() else ""
    full_run_log_text = read_text_file(full_run_log) if full_run_log.exists() else ""
    run_status_path = run_dir / "run-status.env"
    run_status = parse_run_status_file(run_status_path)
    run_history_counts = parse_run_history_status_counts(workspace / "reports" / "taskruns" / "run-history.json")
    summary_missing = not summary_path.exists()
    result_value = first_token(parse_markdown_scalar(summary_text, "result")) if summary_text else ""
    failure_reason = first_token(parse_markdown_scalar(summary_text, "failure_reason")) if summary_text else ""
    termination_signal = first_token(parse_markdown_scalar(summary_text, "termination_signal")) if summary_text else ""
    expected_runs = parse_int(parse_markdown_scalar(summary_text, "expected_runs"), 0) if summary_text else 0
    completed_runs = parse_int(parse_markdown_scalar(summary_text, "completed_runs"), 0) if summary_text else 0
    expected_headless_runs = parse_int(parse_markdown_scalar(summary_text, "expected_headless_runs"), 0) if summary_text else 0
    completed_headless_runs = parse_int(parse_markdown_scalar(summary_text, "completed_headless_runs"), 0) if summary_text else 0
    running_runs_detected = parse_int(parse_markdown_scalar(summary_text, "running_runs_detected"), 0) if summary_text else 0
    run_history_running = int(run_history_counts.get("running", 0))
    api_status = parse_api_status(summary_text)
    terminal_process_failure = terminal_process_failed_summary(
        summary_path.exists(),
        run_status.get("state", ""),
        run_status.get("summary_written", ""),
    )
    terminal_success = terminal_success_summary(
        summary_path.exists(),
        run_status_path.exists(),
        result_value,
        api_status,
        run_status.get("state", ""),
        parse_int(run_status.get("process_exit", "-1"), -1),
    )
    if should_ignore_stale_classified_failure(terminal_success, classified_failure):
        details.append(
            "reliability/classifier-override -> ignored stale classified failure "
            f"{classified_failure} because run-status.env/session-summary mark terminal success"
        )
        classified_failure = "none"
        classified_subclass = "none"
        cancellation_like = False

    rows = parse_run_results(run_results_path)
    headless_rows = parse_headless_rows(rows, provider)
    init_row = headless_rows.get("init")
    refresh_row = headless_rows.get("refresh")
    quality_counter_totals: Counter[str] = Counter()
    for row in (init_row, refresh_row):
        if not row:
            continue
        quality_path, _ = resolve_quality_json(run_dir, row)
        if not quality_path.exists():
            continue
        try:
            quality_counter_totals.update(extract_quality_runtime_counters(read_json(quality_path)))
        except Exception:
            continue
    raw_stall_sources: list[Path] = []
    for workspace_root in workspace_roots:
        raw_stall_sources.extend(sorted((workspace_root / "reports" / "taskruns" / "raw").glob("*-meta.json")))
    raw_stall_counter_totals = extract_raw_runtime_stall_counters(raw_stall_sources)
    for key in QUALITY_COUNTER_KEYS:
        quality_counter_totals[key] = max(
            int(quality_counter_totals.get(key, 0)),
            int(raw_stall_counter_totals.get(key, 0)),
        )
    has_runtime_counter_source = any(int(quality_counter_totals.get(key, 0)) > 0 for key in QUALITY_COUNTER_KEYS)

    repair_attempts = int(quality_counter_totals.get("repair_attempts", 0))
    repair_exhausted = int(quality_counter_totals.get("repair_exhausted", 0))
    fresh_retries = int(quality_counter_totals.get("fresh_retries", 0))
    focused_repairs = int(quality_counter_totals.get("focused_repairs", 0))
    stall_count = int(quality_counter_totals.get("stall_count", 0))
    pre_artifact_stalls = int(quality_counter_totals.get("pre_artifact_stalls", 0))
    post_artifact_stalls = int(quality_counter_totals.get("post_artifact_stalls", 0))
    zero_output_pre_artifact_stalls = int(quality_counter_totals.get("zero_output_pre_artifact_stalls", 0))
    partial_failure_count = int(quality_counter_totals.get("partial_failure_count", 0))
    quality_alerts = int(quality_counter_totals.get("quality_alerts", 0))
    if repair_attempts >= 2:
        issues.append("execution:repair-heavy")
        details.append(f"execution/runtime-recovery -> repair_attempts={repair_attempts} fresh_retries={fresh_retries} focused_repairs={focused_repairs}")
    if repair_exhausted > 0:
        issues.append("execution:repair-exhausted")
        details.append(f"execution/runtime-recovery -> repair_exhausted={repair_exhausted}")
    if stall_count > 0:
        issues.append("execution:stall-pressure")
        details.append(
            f"execution/runtime-stalls -> stall_count={stall_count} pre_artifact={pre_artifact_stalls} "
            f"post_artifact={post_artifact_stalls} zero_output_pre_artifact={zero_output_pre_artifact_stalls}"
        )
        if raw_stall_counter_totals:
            details.append(
                "execution/runtime-stalls-raw -> "
                f"raw_meta_stalls={int(raw_stall_counter_totals.get('stall_count', 0))} "
                f"zero_output_pre_artifact={int(raw_stall_counter_totals.get('zero_output_pre_artifact_stalls', 0))}"
            )
    if partial_failure_count > 0:
        issues.append("execution:partial-failures")
        details.append(f"execution/partial-failures -> partial_failure_count={partial_failure_count}")

    snapshot_ok = True
    artifact_source = "snapshot"
    for row in (init_row, refresh_row):
        if not row:
            continue
        run_id = str(row.get("run_id", "")).strip()
        reports_root, source = resolve_reports_root(run_dir, run_id)
        if source != "snapshot":
            artifact_source = source
            snapshot_ok = False
            details.append(f"reliability/snapshot-missing -> using non-snapshot reports_root={reports_root} source={source}")
        if not reports_root.exists():
            snapshot_ok = False
            details.append(f"reliability/snapshot-missing -> missing {reports_root}")

    if summary_missing:
        issues.append("reliability:summary-missing")
        details.append(f"reliability/summary-missing -> {summary_path} is missing")

    h1 = result_value == "passed" and api_status == "succeeded"
    if not h1:
        issues.append("reliability:session")
        details.append(
            f"reliability/session -> {summary_path}: result={result_value} api={api_status}"
        )

    h2 = bool(init_row and refresh_row and init_row["status"] == "succeeded" and refresh_row["status"] == "succeeded")
    if not h2:
        issues.append("reliability:headless-status")
        details.append(
            f"reliability/headless-status -> {run_results_path}: init={init_row['status'] if init_row else 'missing'} "
            f"refresh={refresh_row['status'] if refresh_row else 'missing'}"
        )

    runtime_contract_failed_hit = False
    runtime_contract_parse_failed_hit = False
    validator_verdict_failed_hit = False
    runner_unavailable_hit = False
    runner_error_hit = False
    parse_stages: set[str] = set()
    raw_outputs: set[str] = set()
    focused_recovery_reasons: set[str] = set()
    focused_recovery_counts: Counter[str] = Counter()
    runtime_metadata_count = 0
    runtime_log_count = 0
    structured_runner_error_sources = [summary_path, full_run_log]
    structured_runner_error_sources.extend(sorted((run_dir / "logs").glob("run-iter*-*.log")))
    raw_runner_error_sources: list[Path] = []
    for workspace_root in workspace_roots:
        taskruns_root = workspace_root / "reports" / "taskruns"
        runtime_metadata_count += sum(1 for _ in taskruns_root.glob("**/runtime-execution.json"))
        workspace_structured_logs = sorted((taskruns_root / "logs").glob("*.ndjson"))
        runtime_log_count += len(workspace_structured_logs)
        structured_runner_error_sources.extend(workspace_structured_logs)
        workspace_raw_outputs = sorted(path for path in (taskruns_root / "raw").rglob("*") if path.is_file())
        runtime_log_count += len(workspace_raw_outputs)
        raw_runner_error_sources.extend(workspace_raw_outputs)
    if not h2 and (runtime_metadata_count > 0 or runtime_log_count > 0):
        details.append(
            "reliability/headless-status -> runtime artifacts/logs found despite missing or failed run-results rows: "
            f"runtime_execution={runtime_metadata_count} task_logs={runtime_log_count}"
        )
    for source_path in structured_runner_error_sources:
        if not source_path.exists():
            continue
        text = read_text_file(source_path)
        focused_recovery_reasons.update(extract_focused_recovery_reason_tags(text))
        focused_recovery_counts.update(extract_focused_recovery_reason_counts(text))
        if not terminal_success:
            if text_has_runtime_contract_parse_signature(text):
                runtime_contract_parse_failed_hit = True
                runtime_contract_failed_hit = True
                runner_error_hit = True
                error_codes.append("runtime_contract_failed")
            if text_has_collect_document_path_contract_signature(text):
                runtime_contract_failed_hit = True
                runner_error_hit = True
                error_codes.append("runtime_contract_failed")
            if text_has_structured_runner_unavailable_signal(text):
                runner_unavailable_hit = True
                runner_error_hit = True
                error_codes.append("runner_unavailable")
            if "runtime_contract_failed" in text:
                runtime_contract_failed_hit = True
                runner_error_hit = True
                error_codes.append("runtime_contract_failed")
            if "validator verdict is FAIL" in text:
                validator_verdict_failed_hit = True
            parse_stages.update(match.group(1).strip() for match in re.finditer(r"parse_stage=([a-z_]+)", text))
            raw_outputs.update(match.group(1).strip() for match in re.finditer(r"raw_output=([^\s)]+)", text))
    for source_path in raw_runner_error_sources:
        if not source_path.exists():
            continue
        text = read_text_file(source_path)
        focused_recovery_reasons.update(extract_focused_recovery_reason_tags(text))
        focused_recovery_counts.update(extract_focused_recovery_reason_counts(text))
        if not terminal_success:
            if text_has_runtime_contract_parse_signature(text):
                runtime_contract_parse_failed_hit = True
                runtime_contract_failed_hit = True
                runner_error_hit = True
                error_codes.append("runtime_contract_failed")
            if text_has_collect_document_path_contract_signature(text):
                runtime_contract_failed_hit = True
                runner_error_hit = True
                error_codes.append("runtime_contract_failed")
            if text_has_raw_provider_runner_unavailable_signal(text):
                runner_unavailable_hit = True
                runner_error_hit = True
                error_codes.append("runner_unavailable")
    h3 = not runner_error_hit
    if not h3:
        issues.append("reliability:runner-errors")
        details.append(f"reliability/runner-errors -> {full_run_log}: detected {sorted(set(error_codes))}")
        if parse_stages:
            details.append(f"reliability/runner-errors -> parse_stages={sorted(parse_stages)}")
        if raw_outputs:
            details.append(f"reliability/runner-errors -> raw_outputs={sorted(raw_outputs)[:5]}")
    if focused_recovery_reasons:
        details.append(f"reliability/focused-recovery -> reasons={sorted(focused_recovery_reasons)}")
        focused_floor, exhausted_floor = focused_recovery_counter_floor(focused_recovery_counts)
        if not has_runtime_counter_source:
            if focused_floor > focused_repairs:
                focused_repairs = focused_floor
            if exhausted_floor > repair_exhausted:
                repair_exhausted = exhausted_floor
        if exhausted_floor > 0 and "execution:repair-exhausted" not in issues:
            issues.append("execution:repair-exhausted")
        if exhausted_floor > 0 and not has_runtime_counter_source:
            details.append(
                "execution/runtime-recovery -> "
                f"focused_repairs={focused_repairs} repair_exhausted={repair_exhausted} "
                "source=focused-recovery-reasons"
            )
    if runtime_contract_failed_hit:
        issues.append("reliability:runtime-contract-failed")
    if runner_unavailable_hit:
        issues.append("reliability:runner-unavailable")

    init_signal = int(init_row["signal"]) if init_row else 0
    refresh_signal = int(refresh_row["signal"]) if refresh_row else 0
    h4 = init_signal > 0 and refresh_signal > 0
    if not h4:
        issues.append("artifact:zero-signal")
        details.append(f"artifact/zero-signal -> {run_results_path}: init_signal={init_signal} refresh_signal={refresh_signal}")

    reliability = bool_score(h1, 15) + bool_score(h2, 15) + bool_score(h3, 10)
    if not snapshot_ok:
        issues.append("reliability:snapshot-missing")
        reliability = max(0, reliability - 10)
    if summary_missing:
        reliability = max(0, reliability - 10)

    runtime_timeout = failure_reason == "runtime_timeout" or termination_signal == "timeout"
    if failure_reason == "runtime_contract_failed":
        runtime_contract_failed_hit = True
        runner_error_hit = True
        error_codes.append("runtime_contract_failed")
    if failure_reason == "runner_unavailable":
        runner_unavailable_hit = True
        runner_error_hit = True
        error_codes.append("runner_unavailable")
    infra_signal_terminated = (
        failure_reason == "infra_signal_terminated"
        or (termination_signal not in {"", "none"} and termination_signal != "-")
    )
    infra_incomplete_cycle = failure_reason == "infra_incomplete_cycle"
    partial_failures_hit = partial_failure_count > 0
    if not terminal_process_failure:
        if expected_runs > 0 and completed_runs != expected_runs:
            infra_incomplete_cycle = True
        if expected_headless_runs > 0 and completed_headless_runs != expected_headless_runs:
            infra_incomplete_cycle = True
        if running_runs_detected > 0:
            infra_incomplete_cycle = True
        if run_history_running > 0:
            infra_incomplete_cycle = True
    if runtime_timeout:
        issues.append("reliability:runtime-timeout")
        details.append(
            f"reliability/runtime-timeout -> {summary_path}: failure_reason={failure_reason or '-'} termination_signal={termination_signal or '-'}"
        )
    if infra_signal_terminated:
        issues.append("reliability:infra-signal-terminated")
        details.append(
            f"reliability/infra-signal-terminated -> {summary_path}: failure_reason={failure_reason or '-'} termination_signal={termination_signal or '-'}"
        )
    if infra_incomplete_cycle:
        issues.append("reliability:infra-incomplete-cycle")
        details.append(
            f"reliability/infra-incomplete-cycle -> {summary_path}: expected_runs={expected_runs} completed_runs={completed_runs} "
            f"expected_headless_runs={expected_headless_runs} completed_headless_runs={completed_headless_runs} "
            f"running_runs_detected={running_runs_detected} run_history_running={run_history_running}"
        )
    if partial_failures_hit:
        issues.append("reliability:partial-failures")
        details.append(
            f"reliability/partial-failures -> partial_failure_count={partial_failure_count}"
        )
    if cancellation_like:
        issues.append("reliability:cancellation-like")
        details.append(
            f"reliability/cancellation-like -> failure_subclass={classified_subclass or '-'} process_exit={classification_row.get('process_exit', '-')}"
        )

    classified_terminal_runtime_provider_failure = classified_failure in {
        "runtime_timeout",
        "runner_unavailable",
        "runtime_contract_failed",
    }
    terminal_runtime_provider_failure = (
        (
            terminal_process_failure
            and result_value == "failed"
            and (
                runtime_timeout
                or runner_unavailable_hit
                or runtime_contract_parse_failed_hit
                or (runtime_contract_failed_hit and not validator_verdict_failed_hit)
            )
        )
        or classified_terminal_runtime_provider_failure
    )

    c1_runtime_name_ok = True
    c2_runtime_versions_ok = True
    c3_metrics_ok = True

    for row in (init_row, refresh_row):
        if not row:
            c1_runtime_name_ok = False
            c2_runtime_versions_ok = False
            c3_metrics_ok = False
            continue
        pipeline = str(row["pipeline"])
        run_id = str(row["run_id"])
        quality_path, _ = resolve_quality_json(run_dir, row)
        if not quality_path.exists():
            c2_runtime_versions_ok = False
            c3_metrics_ok = False
            details.append(f"contract/quality-json-missing -> {quality_path}")
            continue
        quality_payload = read_json(quality_path)
        runtime_versions = quality_payload.get("runtime_versions") or []
        runtime_versions_joined = ",".join(str(x) for x in runtime_versions)
        if runtime_versions_joined != str(row["runtime_versions"]):
            c3_metrics_ok = False
            details.append(
                f"contract/metrics-runtime-versions -> {quality_path}: row='{row['runtime_versions']}' json='{runtime_versions_joined}'"
            )
        for runtime_version in runtime_versions:
            rv = str(runtime_version).strip().lower()
            if rv.endswith("@") or "fake" in rv or "mock" in rv:
                c2_runtime_versions_ok = False
                details.append(f"contract/runtime-versions -> {quality_path}: invalid entry '{runtime_version}'")

        totals = quality_payload.get("totals") or {}
        pairs = (
            ("signal", "signal_score"),
            ("entities", "semantic_entities"),
            ("edges", "semantic_edges"),
            ("findings", "findings_count"),
            ("questions", "questions_count"),
            ("cov_obs", "coverage_observed"),
            ("cov_missing", "coverage_missing"),
            ("warnings", "warnings_count"),
        )
        for row_key, total_key in pairs:
            row_value = int(row[row_key])
            total_value = int(totals.get(total_key, 0))
            if row_value != total_value:
                c3_metrics_ok = False
                details.append(f"contract/metrics-parity -> {quality_path}: {row_key} row={row_value} json={total_value}")

        taskrun_files, _ = resolve_step_taskrun_files(run_dir, run_id, pipeline)
        if not taskrun_files:
            quality_steps = quality_payload.get("steps") or []
            matching_runtime_names = [
                str(step.get("runtime_name", "")).strip()
                for step in quality_steps
                if str(step.get("step_id", "")).strip().startswith(f"{pipeline}.")
            ]
            if not matching_runtime_names:
                c1_runtime_name_ok = False
                details.append(
                    f"contract/runtime-name -> missing step files and quality step runtime_name for run_id={run_id} pipeline={pipeline}"
                )
            for runtime_name in matching_runtime_names:
                if not runtime_name:
                    c1_runtime_name_ok = False
                    details.append(
                        f"contract/runtime-name -> quality step runtime_name is empty for run_id={run_id} pipeline={pipeline}"
                    )
                    continue
                if runtime_name != provider:
                    c1_runtime_name_ok = False
                    details.append(
                        f"contract/runtime-name -> quality step runtime_name mismatch for run_id={run_id} pipeline={pipeline}: "
                        f"expected={provider} got={runtime_name}"
                    )
        for taskrun_file in taskrun_files:
            payload = read_json(taskrun_file)
            runtime_name = str(payload.get("provider") or (payload.get("meta") or {}).get("runtime", {}).get("name", "")).strip()
            if not runtime_name:
                c1_runtime_name_ok = False
                details.append(f"contract/runtime-name -> {taskrun_file}: empty provider/runtime.name")
            if runtime_name != provider:
                c1_runtime_name_ok = False
                details.append(f"contract/runtime-name -> {taskrun_file}: expected={provider} got={runtime_name}")

    if not c1_runtime_name_ok:
        issues.append("contract:runtime-name")
    if not c2_runtime_versions_ok:
        issues.append("contract:runtime-versions")
    if not c3_metrics_ok:
        issues.append("contract:metrics")

    contract = bool_score(c1_runtime_name_ok, 8) + bool_score(c2_runtime_versions_ok, 6) + bool_score(c3_metrics_ok, 6)

    analysis_run_id = str((refresh_row or init_row or {}).get("run_id", "")).strip()
    analysis_reports_root = workspace / "reports"
    if analysis_run_id:
        analysis_reports_root, _ = resolve_reports_root(run_dir, analysis_run_id)
    analysis_report_mode = ""
    analysis_collect_status = ""
    analysis_findings_status = ""
    analysis_evidence_reasons: list[str] = []
    analysis_quality_row = refresh_row or init_row
    artifact_quality_warnings: list[str] = []
    if analysis_quality_row:
        analysis_quality_path, _ = resolve_quality_json(run_dir, analysis_quality_row)
        if analysis_quality_path.exists():
            analysis_quality_payload = read_json(analysis_quality_path)
            artifact_quality_warnings = extract_artifact_quality_warnings(analysis_quality_payload)
            evidence_state = analysis_quality_payload.get("evidence_state") or {}
            if isinstance(evidence_state, dict):
                analysis_report_mode = str(evidence_state.get("report_mode", "")).strip()
                collect_state = evidence_state.get("collect") or {}
                findings_state = evidence_state.get("findings") or {}
                if isinstance(collect_state, dict):
                    analysis_collect_status = str(collect_state.get("status", "")).strip()
                if isinstance(findings_state, dict):
                    analysis_findings_status = str(findings_state.get("status", "")).strip()
                evidence_reasons = evidence_state.get("reasons") or []
                if isinstance(evidence_reasons, list):
                    analysis_evidence_reasons.extend(
                        str(reason).strip() for reason in evidence_reasons if str(reason).strip()
                    )
            if artifact_quality_warnings:
                issues.append("artifact:quality-warning")
                for warning in artifact_quality_warnings:
                    details.append(f"artifact/quality-warning -> {analysis_quality_path}: {warning}")
            if analysis_evidence_reasons:
                details.append(
                    "analysis/evidence-state -> "
                    f"report_mode={analysis_report_mode or '-'} collect_status={analysis_collect_status or '-'} "
                    f"findings_status={analysis_findings_status or '-'} reasons={sorted(set(analysis_evidence_reasons))}"
                )
    findings_path = analysis_reports_root / "findings/findings.md"
    overview_path = analysis_reports_root / "as-is/overview.md"
    coverage_path = analysis_reports_root / "coverage/summary.md"
    questions_path = analysis_reports_root / "coverage/open-questions.md"

    overview_ok = False
    findings_ok = False
    coverage_ok = False
    questions_ok = False

    if overview_path.exists():
        overview_text = read_text_file(overview_path)
        non_empty_lines = [line for line in overview_text.splitlines() if line.strip()]
        placeholder_hit = any("no " in line.lower() and " yet" in line.lower() for line in non_empty_lines)
        if analysis_report_mode == "incomplete":
            overview_ok = report_has_incomplete_banner(overview_text)
        else:
            overview_ok = len(non_empty_lines) >= 4 and not placeholder_hit
        if not overview_ok:
            details.append(
                f"analysis/overview -> {overview_path}: non_empty_lines={len(non_empty_lines)} placeholder={int(placeholder_hit)} "
                f"report_mode={analysis_report_mode or 'normal'} collect_status={analysis_collect_status or '-'}"
            )
    else:
        details.append(f"analysis/overview -> missing {overview_path}")

    findings_text = read_text_file(findings_path) if findings_path.exists() else ""
    findings_text_lower = findings_text.lower()
    has_finding_heading = "## " in findings_text
    has_severity = "- Severity:" in findings_text
    has_description = "- Description:" in findings_text
    has_empty_marker = "No findings reported." in findings_text
    if analysis_report_mode == "incomplete":
        findings_ok = report_has_incomplete_banner(findings_text) and (
            (has_finding_heading and has_severity and has_description) or findings_has_incomplete_fallback(findings_text)
        )
    else:
        findings_ok = has_finding_heading and has_severity and has_description and not has_empty_marker
    if not findings_ok:
        details.append(
            f"analysis/findings -> {findings_path}: heading={int(has_finding_heading)} severity={int(has_severity)} "
            f"description={int(has_description)} empty_marker={int(has_empty_marker)} report_mode={analysis_report_mode or 'normal'} "
            f"findings_status={analysis_findings_status or '-'}"
        )

    missing_terms_raw = parse_markdown_section_bullets(coverage_path, "Missing")
    missing_terms = [normalize_text(term) for term in missing_terms_raw]
    missing_dupes = len(missing_terms) != len(set(missing_terms))
    notes_raw = parse_markdown_section_bullets(coverage_path, "Notes")
    notes_norm = [normalize_text(note) for note in notes_raw]
    notes_dupes = len(notes_norm) != len(set(notes_norm))
    coverage_text = read_text_file(coverage_path) if coverage_path.exists() else ""
    coverage_substantive_ok = len(missing_terms) > 0 and not missing_dupes and not notes_dupes
    if analysis_report_mode == "incomplete":
        coverage_ok = report_has_incomplete_banner(coverage_text) and (
            coverage_substantive_ok or coverage_has_incomplete_fallback(coverage_text)
        )
    else:
        coverage_ok = coverage_substantive_ok
    if not coverage_ok:
        details.append(
            f"analysis/coverage -> {coverage_path}: missing={len(missing_terms)} missing_dupes={int(missing_dupes)} notes_dupes={int(notes_dupes)} "
            f"report_mode={analysis_report_mode or 'normal'} collect_status={analysis_collect_status or '-'}"
        )

    questions_text = read_text_file(questions_path) if questions_path.exists() else ""
    _question_ids, question_texts = parse_open_questions(questions_path)
    question_texts_norm = [normalize_text(text) for text in question_texts if text.strip()]
    question_dupes = len(question_texts_norm) != len(set(question_texts_norm))
    questions_substantive_ok = len(question_texts_norm) > 0 and not question_dupes
    if analysis_report_mode == "incomplete":
        questions_ok = report_has_incomplete_banner(questions_text) and (
            questions_substantive_ok or open_questions_has_incomplete_fallback(questions_text)
        )
    else:
        questions_ok = questions_substantive_ok
    if not questions_ok:
        details.append(
            f"analysis/questions -> {questions_path}: count={len(question_texts_norm)} dupes={int(question_dupes)} "
            f"report_mode={analysis_report_mode or 'normal'} collect_status={analysis_collect_status or '-'}"
        )

    owner_gap = any(term in {"owner mappings", "owner mapping", "owner team mappings", "owner team mapping"} for term in missing_terms)
    if analysis_report_mode != "incomplete" and owner_gap and has_empty_marker:
        findings_ok = False
        details.append(f"analysis/findings-owner-gap -> {findings_path}: owner gap exists in {coverage_path}, findings are empty")

    semantic_hard_fail = False
    off_topic_hits = 0

    refresh_step_files: list[Path] = []
    if refresh_row:
        refresh_run_id = str(refresh_row.get("run_id", ""))
        refresh_step_files = resolve_step_semantic_files(run_dir, refresh_run_id)

    if refresh_step_files:
        non_power_target = not is_power_target(repo_roots, declared_meta)
        step1_files = [path for path in refresh_step_files if path.name == "shard-pack-manifest.json"]
        if non_power_target:
            for step_file in step1_files:
                payload = read_json(step_file)
                hits = collect_off_topic_hits(payload)
                if hits:
                    off_topic_hits += len(hits)
                    semantic_hard_fail = True
                    issues.append("analysis:off-topic")
                    details.append(f"analysis/off-topic -> {step_file}: hits={','.join(hits)}")

        invalid_evidence: list[str] = []
        for step_file in refresh_step_files:
            payload = read_json(step_file)
            for evidence_path in collect_evidence_paths(payload):
                ok, reason = evidence_path_resolves(evidence_path, repo_roots, workspace)
                if not ok:
                    invalid_evidence.append(f"{step_file} :: {evidence_path} ({reason})")
        if invalid_evidence:
            semantic_hard_fail = True
            issues.append("analysis:evidence-scope")
            for item in invalid_evidence[:8]:
                details.append(f"analysis/evidence-scope -> {item}")
            if len(invalid_evidence) > 8:
                details.append(f"analysis/evidence-scope -> +{len(invalid_evidence) - 8} more invalid evidence paths")

        if expected_repo_count >= 2:
            if terminal_runtime_provider_failure:
                details.append(
                    "analysis/cross-repo-missing -> skipped for terminal runtime/provider failure classification"
                )
            else:
                repo_mentions: set[str] = set()
                edge_upserts = 0
                cross_repo_semantic_links = 0
                for step_file in refresh_step_files:
                    payload = read_json(step_file)
                    repo_mentions.update(collect_repo_mentions(payload))
                    edge_upserts += count_semantic_edges(payload)
                    cross_repo_semantic_links += count_cross_repo_semantic_links(payload)
                missing_dimensions: list[str] = []
                if len(repo_mentions) < 2:
                    missing_dimensions.append("repo_mentions<2")
                if edge_upserts < 1 and cross_repo_semantic_links < 1:
                    missing_dimensions.append("no_semantic_edges_or_cross_repo_links")
                if missing_dimensions:
                    semantic_hard_fail = True
                    issues.append("analysis:cross-repo-missing")
                    details.append(
                        f"analysis/cross-repo-missing -> run_dir={run_dir} expected_repo_count={expected_repo_count} "
                        f"repo_mentions={len(repo_mentions)} edge_upserts={edge_upserts} "
                        f"cross_repo_semantic_links={cross_repo_semantic_links} "
                        f"missing_dimensions={','.join(missing_dimensions)} "
                        "required_fix=add repo-specific citations plus at least one semantic edge, cross-repo finding with repo/path provenance, or cross-repo question with related repo ids"
                    )
    elif expected_repo_count >= 2:
        if terminal_runtime_provider_failure:
            details.append(
                "analysis/cross-repo-missing -> skipped for terminal runtime/provider failure classification "
                "(missing refresh step runtime-execution artifacts)"
            )
        else:
            semantic_hard_fail = True
            issues.append("analysis:cross-repo-missing")
            details.append(
                f"analysis/cross-repo-missing -> run_dir={run_dir} expected_repo_count={expected_repo_count} "
                "missing refresh step runtime-execution artifacts"
            )

    overview_counts = parse_overview_counts(overview_path)
    services_count = int(overview_counts.get("services", 0))
    if services_count > 0 and "missing service definition files" in findings_text_lower:
        semantic_hard_fail = True
        issues.append("analysis:cross-doc")
        details.append(
            f"analysis/cross-doc -> {overview_path} services={services_count} conflicts with 'missing service definition files' in {findings_path}"
        )

    if semantic_hard_fail:
        issues.append("reliability:semantic-hard-fail")

    runtime_flow_failed = False
    if expected_execution is not None and not terminal_runtime_provider_failure:
        runtime_unavailable_hint = text_has_runner_unavailable_signal(summary_text + "\n" + full_run_log_text)
        runtime_flow_issues, runtime_flow_details = evaluate_runtime_flow_checks(
            run_dir,
            workspace,
            headless_rows,
            expected_execution,
            summary_text,
            full_run_log_text,
            runtime_unavailable_hint,
        )
        if runtime_flow_issues:
            runtime_flow_failed = True
            issues.extend(sorted(runtime_flow_issues))
            issues.append("reliability:runtime-flow-failed")
            details.extend(runtime_flow_details)
    if (
        terminal_process_failure
        and result_value == "failed"
        and not terminal_runtime_provider_failure
        and not runtime_timeout
        and not runner_unavailable_hit
        and not runtime_contract_failed_hit
        and not infra_signal_terminated
    ):
        runtime_flow_failed = True
        if "reliability:runtime-flow-failed" not in issues:
            issues.append("reliability:runtime-flow-failed")
            details.append(
                "reliability/runtime-flow-failed -> terminal process_failed run-status + session-summary indicate completed deterministic pipeline failure"
            )
    if terminal_process_failure and validator_verdict_failed_hit and not terminal_runtime_provider_failure:
        runtime_flow_failed = True
        if "reliability:runtime-flow-failed" not in issues:
            issues.append("reliability:runtime-flow-failed")
        details.append(
            "reliability/runtime-flow-failed -> validator verdict is FAIL in terminal task logs; classify as runtime flow failure, not runtime contract failure"
        )
    if partial_failures_hit:
        runtime_flow_failed = True
        if "reliability:runtime-flow-failed" not in issues:
            issues.append("reliability:runtime-flow-failed")
        details.append("reliability/runtime-flow-failed -> partial shard failures were recorded during execution")

    if not overview_ok:
        issues.append("analysis:overview")
    if not findings_ok:
        issues.append("analysis:findings")
    if not coverage_ok:
        issues.append("analysis:coverage")
    if not questions_ok:
        issues.append("analysis:questions")

    analysis = bool_score(overview_ok, 10) + bool_score(findings_ok, 10) + bool_score(coverage_ok, 10) + bool_score(questions_ok, 10)

    failure_class = "none"
    if summary_missing:
        failure_class = "summary_missing"
    elif runtime_timeout:
        failure_class = "runtime_timeout"
    elif runtime_contract_parse_failed_hit:
        failure_class = "runtime_contract_failed"
    elif failure_reason == "runtime_contract_failed":
        failure_class = "runtime_contract_failed"
    elif validator_verdict_failed_hit or runtime_flow_failed:
        failure_class = "runtime_flow_failed"
    elif runtime_contract_failed_hit:
        failure_class = "runtime_contract_failed"
    elif infra_signal_terminated:
        failure_class = "infra_signal_terminated"
    elif runner_unavailable_hit:
        failure_class = "runner_unavailable"
    elif infra_incomplete_cycle:
        failure_class = "infra_incomplete_cycle"

    if classified_failure and classified_failure != "none":
        if failure_class != classified_failure:
            details.append(
                f"reliability/classifier-override -> summary_class={failure_class or 'none'} classifier_class={classified_failure}"
            )
        ignore_classified_incomplete = should_ignore_classified_incomplete_for_terminal_process(
            terminal_process_failure,
            classified_failure,
            failure_reason,
        )
        ignore_classified_contract_override = runtime_contract_failed_hit and classified_failure in {
            "runner_unavailable",
            "runtime_flow_failed",
        }
        ignore_classified_partial_collect_override = (
            runtime_flow_failed
            and partial_failures_hit
            and classified_failure in {"runner_unavailable", "runtime_contract_failed"}
        )
        if ignore_classified_incomplete:
            details.append(
                "reliability/classifier-override -> ignored infra_incomplete_cycle because run-status.env marks terminal process_failed summary"
            )
        elif ignore_classified_contract_override:
            details.append(
                "reliability/classifier-override -> ignored runner/runtime-flow override because raw runtime/session-summary classified the run as runtime_contract_failed"
            )
        elif ignore_classified_partial_collect_override:
            details.append(
                "reliability/classifier-override -> ignored runner/runtime-contract override because partial collect shard failures keep runtime_flow_failed as the primary root cause"
            )
        elif failure_class == "summary_missing" and classified_failure in {
            "runtime_timeout",
            "runner_unavailable",
            "runtime_contract_failed",
            "infra_signal_terminated",
            "infra_incomplete_cycle",
            "runtime_flow_failed",
        }:
            failure_class = classified_failure
        elif failure_class == "none" or failure_class_rank(classified_failure) < failure_class_rank(failure_class):
            failure_class = classified_failure
        if not (ignore_classified_partial_collect_override and classified_failure == "runtime_contract_failed"):
            runtime_contract_failed_hit = runtime_contract_failed_hit or classified_failure == "runtime_contract_failed"
        if not ignore_classified_contract_override and not ignore_classified_partial_collect_override:
            runner_unavailable_hit = runner_unavailable_hit or classified_failure == "runner_unavailable"
        runtime_timeout = runtime_timeout or classified_failure == "runtime_timeout"
        infra_signal_terminated = infra_signal_terminated or classified_failure == "infra_signal_terminated"
        if not ignore_classified_incomplete:
            infra_incomplete_cycle = infra_incomplete_cycle or classified_failure == "infra_incomplete_cycle"
        summary_missing = summary_missing or classified_failure == "summary_missing"

    hard_pass = (
        h1
        and h2
        and h3
        and snapshot_ok
        and not runtime_flow_failed
        and not summary_missing
        and not runtime_timeout
        and not infra_signal_terminated
        and not infra_incomplete_cycle
    )

    total = reliability + contract + analysis
    return RunEvaluation(
        provider=provider,
        run_index=run_index,
        run_dir=run_dir,
        hard_pass=hard_pass,
        reliability=reliability,
        contract=contract,
        analysis=analysis,
        total=total,
        verdict=verdict(total),
        init_signal=init_signal,
        refresh_signal=refresh_signal,
        refresh_findings=int(refresh_row["findings"]) if refresh_row else 0,
        refresh_questions=int(refresh_row["questions"]) if refresh_row else 0,
        refresh_cov_missing=int(refresh_row["cov_missing"]) if refresh_row else 0,
        repair_attempts=repair_attempts,
        repair_exhausted=repair_exhausted,
        fresh_retries=fresh_retries,
        focused_repairs=focused_repairs,
        stall_count=stall_count,
        pre_artifact_stalls=pre_artifact_stalls,
        post_artifact_stalls=post_artifact_stalls,
        zero_output_pre_artifact_stalls=zero_output_pre_artifact_stalls,
        partial_failure_count=partial_failure_count,
        quality_alerts=quality_alerts,
        artifact_source=artifact_source,
        semantic_hard_fail=semantic_hard_fail,
        off_topic_hits=off_topic_hits,
        failure_class=failure_class,
        runtime_contract_failed=runtime_contract_failed_hit,
        runner_unavailable=runner_unavailable_hit,
        runtime_timeout=runtime_timeout,
        infra_signal_terminated=infra_signal_terminated,
        infra_incomplete_cycle=infra_incomplete_cycle,
        summary_missing=summary_missing,
        precheck_failed=False,
        runtime_flow_failed=runtime_flow_failed,
        cancellation_like=cancellation_like,
        artifact_quality_findings=len(artifact_quality_warnings),
        issues=sorted(set(issues)),
        issue_details=details,
        error_codes=sorted(set(error_codes)),
    )


def frontend_result_run_index(payload: dict[str, Any], path: Path) -> int | None:
    raw = payload.get("run_index")
    if isinstance(raw, int) and raw > 0:
        return raw
    if isinstance(raw, str):
        try:
            parsed = int(raw.strip())
        except Exception:
            parsed = 0
        if parsed > 0:
            return parsed
    for part in reversed(path.parts):
        match = re.fullmatch(r"run([1-9][0-9]*)", part)
        if match:
            return int(match.group(1))
    return None


def load_frontend_result_entries(
    batch_root: Path, subdir: str, result_filename: str, providers: list[str] | None = None
) -> list[dict[str, Any]]:
    entries: list[dict[str, Any]] = []
    active_providers = providers or list(FRONTEND_PROVIDERS)
    for provider in active_providers:
        provider_root = batch_root / subdir / provider
        candidate_paths: list[Path] = []
        root_result = provider_root / result_filename
        if root_result.exists():
            candidate_paths.append(root_result)
        candidate_paths.extend(sorted(provider_root.glob(f"run*/{result_filename}")))

        seen: set[str] = set()
        for path in candidate_paths:
            key = str(path.resolve())
            if key in seen:
                continue
            seen.add(key)
            payload = read_json(path)
            payload["path"] = str(path)
            payload["runtime_provider"] = str(payload.get("runtime_provider", provider) or provider)
            run_index = frontend_result_run_index(payload, path)
            if run_index is not None:
                payload["run_index"] = run_index
            entries.append(payload)

    entries.sort(
        key=lambda item: (
            str(item.get("runtime_provider", "")),
            int(item.get("run_index", 0) or 0),
            str(item.get("path", "")),
        )
    )
    return entries


def load_frontend_results(batch_root: Path, providers: list[str] | None = None) -> list[dict[str, Any]]:
    return load_frontend_result_entries(batch_root, "frontend", FRONTEND_LIVE_RESULT_FILENAME, providers)


def frontend_entries_by_provider(
    entries: list[dict[str, Any]], providers: list[str] | None = None
) -> dict[str, list[dict[str, Any]]]:
    active_providers = providers or list(FRONTEND_PROVIDERS)
    allowed = set(active_providers)
    grouped: dict[str, list[dict[str, Any]]] = {provider: [] for provider in active_providers}
    for payload in entries:
        provider = str(payload.get("runtime_provider", "")).strip()
        if provider not in allowed:
            continue
        grouped[provider].append(payload)
    for provider in grouped:
        grouped[provider].sort(key=lambda item: (int(item.get("run_index", 0) or 0), str(item.get("path", ""))))
    return grouped


def aggregate_frontend_status(items: list[dict[str, Any]]) -> str:
    if not items:
        return "missing"
    statuses = [str(item.get("status", "missing")).strip() or "missing" for item in items]
    if all(status == "passed" for status in statuses):
        return "passed"
    if all(status == "skipped" for status in statuses):
        return "skipped"
    if any(status == "failed" for status in statuses):
        return "failed"
    if any(status == "missing" for status in statuses):
        return "missing"
    return "mixed"


def aggregate_frontend_reasons(items: list[dict[str, Any]]) -> str:
    if not items:
        return "-"
    reasons = Counter(str(item.get("reason", "-")).strip() or "-" for item in items)
    return ", ".join(f"{reason}={count}" for reason, count in sorted(reasons.items()))


def ensure_parent_dir(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def write_run_matrix(path: Path, runs: list[RunEvaluation]) -> None:
    ensure_parent_dir(path)
    lines = [
        "# Run Matrix",
        "",
        "| provider | run | hard_pass | reliability | contract | analysis | total | verdict | artifact_source | semantic_hard_fail | failure_class | runtime_contract_failed | runner_unavailable | runtime_timeout | infra_signal_terminated | infra_incomplete_cycle | summary_missing | precheck_failed | runtime_flow_failed | cancellation_like | off_topic_hits | init_signal | refresh_signal | refresh_findings | refresh_questions | refresh_cov_missing | repair_attempts | repair_exhausted | fresh_retries | focused_repairs | stall_count | pre_artifact_stalls | post_artifact_stalls | zero_output_pre_artifact_stalls | partial_failure_count | quality_alerts | artifact_quality_findings | issues |",
        "|---|---:|---:|---:|---:|---:|---:|---|---|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|",
    ]
    for item in runs:
        lines.append(
            "| "
            f"{item.provider} | {item.run_index} | {int(item.hard_pass)} | {item.reliability} | {item.contract} | "
            f"{item.analysis} | {item.total} | {item.verdict} | {item.artifact_source} | {int(item.semantic_hard_fail)} | {item.failure_class} | "
            f"{int(item.runtime_contract_failed)} | {int(item.runner_unavailable)} | {int(item.runtime_timeout)} | {int(item.infra_signal_terminated)} | "
            f"{int(item.infra_incomplete_cycle)} | {int(item.summary_missing)} | {int(item.precheck_failed)} | {int(item.runtime_flow_failed)} | {int(item.cancellation_like)} | {item.off_topic_hits} | "
            f"{item.init_signal} | {item.refresh_signal} | "
            f"{item.refresh_findings} | {item.refresh_questions} | {item.refresh_cov_missing} | "
            f"{item.repair_attempts} | {item.repair_exhausted} | {item.fresh_retries} | {item.focused_repairs} | "
            f"{item.stall_count} | {item.pre_artifact_stalls} | {item.post_artifact_stalls} | "
            f"{item.zero_output_pre_artifact_stalls} | {item.partial_failure_count} | {item.quality_alerts} | {item.artifact_quality_findings} | "
            f"{', '.join(item.issues) if item.issues else '-'} |"
        )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_frontend_matrix(path: Path, frontend: list[dict[str, Any]], providers: list[str] | None = None) -> None:
    ensure_parent_dir(path)
    active_providers = providers or list(FRONTEND_PROVIDERS)
    grouped = frontend_entries_by_provider(frontend, active_providers)
    lines = [
        "# Frontend Live E2E Matrix",
        "",
        "## Summary",
        "",
        "| provider | status | runs | reasons |",
        "|---|---|---:|---|",
    ]
    for provider in active_providers:
        items = grouped.get(provider, [])
        lines.append(
            f"| {provider} | {aggregate_frontend_status(items)} | {len(items)} | {aggregate_frontend_reasons(items)} |"
        )

    lines.extend(
        [
            "",
            "## Run Details",
            "",
            "| provider | run | status | reason | runtime_details | base_url | workspace | runtime_command | server_log | playwright_log |",
            "|---|---:|---|---|---|---|---|---|---|---|",
        ]
    )
    for provider in active_providers:
        items = grouped.get(provider, [])
        if not items:
            lines.append(f"| {provider} | 0 | missing | missing_result | - | - | - | - | - | - |")
            continue
        for payload in items:
            run_index = int(payload.get("run_index", 0) or 0)
            run_label = str(run_index) if run_index > 0 else "-"
            lines.append(
                "| "
                f"{md_cell(provider)} | {run_label} | {md_cell(payload.get('status', '-'))} | {md_cell(payload.get('reason', '-'))} | "
                f"{md_cell(frontend_runtime_details(payload))} | {md_cell(payload.get('base_url', '-'))} | "
                f"{md_cell(payload.get('workspace', '-'))} | {md_cell(payload.get('runtime_command', '-'))} | "
                f"{md_cell(payload.get('server_log', '-'))} | {md_cell(payload.get('playwright_log', '-'))} |"
            )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def md_cell(value: Any) -> str:
    text = str(value if value is not None else "-").strip() or "-"
    return text.replace("|", "\\|").replace("\n", " ")


def frontend_runtime_details(payload: dict[str, Any]) -> str:
    details: list[str] = []
    run_id = str(payload.get("run_id") or "").strip()
    if run_id:
        details.append(f"run_id={run_id}")
    last_status = str(payload.get("last_run_status") or "").strip()
    if last_status:
        details.append(f"last_run_status={last_status}")
    error_code = str(payload.get("last_run_error_code") or "").strip()
    if error_code:
        details.append(f"error_code={error_code}")
    current_step = str(payload.get("last_run_current_step") or "").strip()
    if current_step:
        details.append(f"current_step={current_step}")
    diagnostic_refs = payload.get("diagnostic_refs")
    if isinstance(diagnostic_refs, dict):
        screenshots = diagnostic_refs.get("screenshots")
        if isinstance(screenshots, list) and screenshots:
            details.append(f"screenshots={len(screenshots)}")
        results_dir = str(diagnostic_refs.get("playwright_results") or "").strip()
        if results_dir:
            details.append(f"playwright_results={results_dir}")
    return "; ".join(details) if details else "-"


def frontend_live_verdict_lines(frontend: list[dict[str, Any]], providers: list[str] | None = None) -> list[str]:
    active_providers = providers or list(FRONTEND_PROVIDERS)
    grouped = frontend_entries_by_provider(frontend, active_providers)
    return [
        f"- {provider}: {aggregate_frontend_status(grouped.get(provider, []))} (runs={len(grouped.get(provider, []))})"
        for provider in active_providers
    ]

def provider_matrix_rows(
    runs: list[RunEvaluation],
    frontend: list[dict[str, Any]],
    providers: list[str] | None = None,
) -> list[dict[str, Any]]:
    active_providers = providers or list(FRONTEND_PROVIDERS)
    grouped: dict[str, list[RunEvaluation]] = defaultdict(list)
    for run in runs:
        grouped[run.provider].append(run)
    frontend_grouped = frontend_entries_by_provider(frontend, active_providers)

    rows: list[dict[str, Any]] = []
    for provider in active_providers:
        items = grouped.get(provider, [])
        totals = [item.total for item in items]
        refresh_signals = [item.refresh_signal for item in items]
        issues_counter = Counter()
        error_codes_counter = Counter()
        artifact_sources = Counter()
        for item in items:
            issues_counter.update(item.issues)
            error_codes_counter.update(item.error_codes)
            artifact_sources.update([item.artifact_source])
        frontend_items = frontend_grouped.get(provider, [])
        frontend_pass_rate = (
            sum(1 for payload in frontend_items if str(payload.get("status", "")).strip() == "passed") / len(frontend_items)
            if frontend_items
            else 0.0
        )
        rows.append(
            {
                "provider": provider,
                "runs": len(items),
                "pass_rate": (sum(1 for item in items if item.hard_pass) / len(items)) if items else 0.0,
                "avg_total": mean(totals) if totals else 0.0,
                "std_total": pstdev(totals) if len(totals) > 1 else 0.0,
                "avg_reliability": mean([item.reliability for item in items]) if items else 0.0,
                "avg_contract": mean([item.contract for item in items]) if items else 0.0,
                "avg_analysis": mean([item.analysis for item in items]) if items else 0.0,
                "avg_signal": mean(refresh_signals) if refresh_signals else 0.0,
                "std_signal": pstdev(refresh_signals) if len(refresh_signals) > 1 else 0.0,
                "avg_findings": mean([item.refresh_findings for item in items]) if items else 0.0,
                "avg_questions": mean([item.refresh_questions for item in items]) if items else 0.0,
                "avg_cov_missing": mean([item.refresh_cov_missing for item in items]) if items else 0.0,
                "repair_attempts": sum(item.repair_attempts for item in items),
                "repair_exhausted": sum(item.repair_exhausted for item in items),
                "fresh_retries": sum(item.fresh_retries for item in items),
                "focused_repairs": sum(item.focused_repairs for item in items),
                "stall_count": sum(item.stall_count for item in items),
                "pre_artifact_stalls": sum(item.pre_artifact_stalls for item in items),
                "post_artifact_stalls": sum(item.post_artifact_stalls for item in items),
                "zero_output_pre_artifact_stalls": sum(item.zero_output_pre_artifact_stalls for item in items),
                "partial_failure_count": sum(item.partial_failure_count for item in items),
                "quality_alerts": sum(item.quality_alerts for item in items),
                "off_topic_hits": sum(item.off_topic_hits for item in items),
                "semantic_hard_fail_runs": sum(1 for item in items if item.semantic_hard_fail),
                "runtime_contract_failed_failures": sum(1 for item in items if item.runtime_contract_failed),
                "runner_unavailable_failures": sum(1 for item in items if item.runner_unavailable),
                "runtime_timeout_failures": sum(1 for item in items if item.runtime_timeout),
                "infra_signal_terminated_failures": sum(1 for item in items if item.infra_signal_terminated),
                "infra_incomplete_cycle_failures": sum(1 for item in items if item.infra_incomplete_cycle),
                "artifact_quality_findings": sum(item.artifact_quality_findings for item in items),
                "summary_missing_failures": sum(1 for item in items if item.summary_missing),
                "precheck_failed_failures": sum(1 for item in items if item.precheck_failed),
                "runtime_flow_failed_failures": sum(1 for item in items if item.runtime_flow_failed),
                "cancellation_like_failures": sum(1 for item in items if item.cancellation_like),
                "error_codes": ", ".join(f"{code}={count}" for code, count in sorted(error_codes_counter.items())) or "-",
                "issues_top": ", ".join(f"{name}={count}" for name, count in issues_counter.most_common(3)) or "-",
                "artifact_sources": ", ".join(f"{name}={count}" for name, count in sorted(artifact_sources.items())) or "-",
                "frontend_pass_rate": frontend_pass_rate,
                "frontend_status": aggregate_frontend_status(frontend_items),
            }
        )
    return rows


def write_execution_report(
    path: Path,
    batch_id: str,
    runs: list[RunEvaluation],
    frontend: list[dict[str, Any]],
    preflight: dict[str, Any],
    providers: list[str] | None = None,
) -> None:
    ensure_parent_dir(path)
    active_providers = providers or list(FRONTEND_PROVIDERS)
    provider_rows = provider_matrix_rows(runs, frontend, active_providers)
    hard_pass_all = sum(1 for run in runs if run.hard_pass)
    semantic_hard_fail_runs = sum(1 for run in runs if run.semantic_hard_fail)
    runtime_flow_failed_runs = sum(1 for run in runs if run.runtime_flow_failed)
    repair_attempts_total = sum(run.repair_attempts for run in runs)
    repair_exhausted_total = sum(run.repair_exhausted for run in runs)
    fresh_retries_total = sum(run.fresh_retries for run in runs)
    focused_repairs_total = sum(run.focused_repairs for run in runs)
    stall_count_total = sum(run.stall_count for run in runs)
    pre_artifact_stalls_total = sum(run.pre_artifact_stalls for run in runs)
    post_artifact_stalls_total = sum(run.post_artifact_stalls for run in runs)
    zero_output_pre_artifact_stalls_total = sum(run.zero_output_pre_artifact_stalls for run in runs)
    partial_failure_count_total = sum(run.partial_failure_count for run in runs)
    quality_alerts_total = sum(run.quality_alerts for run in runs)
    declared_meta = normalize_declared_repos_meta(preflight)
    declared_repos = declared_meta.get("declared_repos") or []
    issue_counter = Counter()
    for run in runs:
        issue_counter.update(run.issues)
    snapshot_runs = sum(1 for run in runs if run.artifact_source == "snapshot")

    lines = [
        f"# Execution Report: {batch_id}",
        "",
        "## Context",
        f"- generated_at_utc: {preflight.get('generated_at_utc', '-')}",
        f"- provenarch_sha: {preflight.get('provenarch_sha', '-')}",
        f"- target_repos_file: {declared_meta.get('target_repos_file', '-')}",
        f"- profile_id: {declared_meta.get('profile_id', '-')}",
        f"- profile_source_kind: {declared_meta.get('profile_source_kind', '-')}",
        f"- expected_repo_count: {declared_meta.get('expected_repo_count', '-')}",
        f"- declared_repos_count: {len(declared_repos)}",
        f"- claude: {((preflight.get('runtimes') or {}).get('claude') or {}).get('version_line', '-')}",
        f"- qwen: {((preflight.get('runtimes') or {}).get('qwen') or {}).get('version_line', '-')}",
        f"- codex: {((preflight.get('runtimes') or {}).get('codex') or {}).get('version_line', '-')}",
        "",
        "## Backend Execution Verdict",
        f"- hard_pass_runs: {hard_pass_all}/{len(runs)}",
        f"- runtime_flow_failed_runs: {runtime_flow_failed_runs}/{len(runs)}",
        f"- primary_failure_classes: {', '.join(sorted({run.failure_class for run in runs if run.failure_class and run.failure_class != '-'})) or 'none'}",
        "",
        "Machine execution verdict is based on runtime/preflight/frontend execution evidence only. Artifact quality signals below are telemetry for SWE assessment and do not flip the machine execution verdict.",
        "",
        "## Runtime Recovery And Artifact Telemetry",
        f"- semantic_hard_fail_runs: {semantic_hard_fail_runs}/{len(runs)}",
        f"- artifact_source_snapshot_runs: {snapshot_runs}/{len(runs)}",
        f"- repair_attempts: {repair_attempts_total}",
        f"- repair_exhausted: {repair_exhausted_total}",
        f"- fresh_retries: {fresh_retries_total}",
        f"- focused_repairs: {focused_repairs_total}",
        f"- stall_count: {stall_count_total}",
        f"- pre_artifact_stalls: {pre_artifact_stalls_total}",
        f"- post_artifact_stalls: {post_artifact_stalls_total}",
        f"- zero_output_pre_artifact_stalls: {zero_output_pre_artifact_stalls_total}",
        f"- partial_failure_count: {partial_failure_count_total}",
        f"- quality_alerts: {quality_alerts_total}",
        f"- artifact_quality_findings: {sum(run.artifact_quality_findings for run in runs)}",
        "",
        "## Frontend Live Smoke Verdict",
        *frontend_live_verdict_lines(frontend, active_providers),
        "",
        "## Provider Matrix",
        "",
        "| provider | runs | pass_rate | avg_total | std_total | avg_reliability | avg_contract | avg_analysis | avg_refresh_signal | std_refresh_signal | avg_refresh_findings | avg_refresh_questions | avg_refresh_cov_missing | repair_attempts | repair_exhausted | fresh_retries | focused_repairs | stall_count | pre_artifact_stalls | post_artifact_stalls | zero_output_pre_artifact_stalls | partial_failure_count | quality_alerts | artifact_quality_findings | off_topic_hits | semantic_hard_fail_runs | runtime_contract_failed_failures | runner_unavailable_failures | runtime_timeout_failures | infra_signal_terminated_failures | infra_incomplete_cycle_failures | summary_missing_failures | precheck_failed_failures | runtime_flow_failed_failures | cancellation_like_failures | artifact_sources | error_codes | frontend_live_pass_rate |",
        "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---:|",
    ]
    for row in provider_rows:
        lines.append(
            "| "
            f"{row['provider']} | {row['runs']} | {row['pass_rate']:.2f} | {row['avg_total']:.2f} | {row['std_total']:.2f} | "
            f"{row['avg_reliability']:.2f} | {row['avg_contract']:.2f} | {row['avg_analysis']:.2f} | "
            f"{row['avg_signal']:.2f} | {row['std_signal']:.2f} | {row['avg_findings']:.2f} | {row['avg_questions']:.2f} | "
            f"{row['avg_cov_missing']:.2f} | {row['repair_attempts']} | {row['repair_exhausted']} | {row['fresh_retries']} | "
            f"{row['focused_repairs']} | {row['stall_count']} | {row['pre_artifact_stalls']} | {row['post_artifact_stalls']} | "
            f"{row['zero_output_pre_artifact_stalls']} | "
            f"{row['partial_failure_count']} | {row['quality_alerts']} | {row['artifact_quality_findings']} | {row['off_topic_hits']} | {row['semantic_hard_fail_runs']} | "
            f"{row['runtime_contract_failed_failures']} | {row['runner_unavailable_failures']} | {row['runtime_timeout_failures']} | {row['infra_signal_terminated_failures']} | "
            f"{row['infra_incomplete_cycle_failures']} | {row['summary_missing_failures']} | {row['precheck_failed_failures']} | {row['runtime_flow_failed_failures']} | {row['cancellation_like_failures']} | "
            f"{row['artifact_sources']} | {row['error_codes']} | {row['frontend_pass_rate']:.2f} |"
        )

    lines.extend(
        [
            "",
            "## Что Сделано Качественно",
        ]
    )
    frontend_grouped = frontend_entries_by_provider(frontend, active_providers)
    frontend_statuses = {
        provider: aggregate_frontend_status(frontend_grouped.get(provider, []))
        for provider in active_providers
    }
    frontend_all_passed = bool(active_providers) and all(status == "passed" for status in frontend_statuses.values())
    frontend_all_skipped = bool(active_providers) and all(status == "skipped" for status in frontend_statuses.values())
    frontend_has_evidence = any(frontend_grouped.get(provider, []) for provider in active_providers)
    if hard_pass_all == len(runs):
        lines.append(f"- Все `{hard_pass_all}/{len(runs)}` backend full-runs прошли hard-gates без падений pipeline и signal regression.")
    else:
        lines.append("- Часть run прошла hard-gates, но есть деградации стабильности (см. matrix).")
    if frontend_all_passed:
        lines.append(f"- Frontend live e2e прошёл для выбранных провайдеров (`{len(active_providers)}/{len(active_providers)}`).")
    elif frontend_all_skipped:
        lines.append("- Frontend live e2e не запускался для этого batch (`skipped` для выбранных провайдеров).")
    elif not frontend_has_evidence:
        lines.append("- Frontend live e2e evidence отсутствует для выбранных провайдеров.")
    else:
        lines.append(f"- Frontend live e2e не полностью прошёл для выбранных провайдеров (`<{len(active_providers)}/{len(active_providers)}`).")
    lines.append("- Контрактная совместимость runtime/report артефактов проверена автоматически для каждого run.")

    lines.extend(
        [
            "",
            "## Что Плохо",
        ]
    )
    if issue_counter:
        for issue_name, count in issue_counter.most_common():
            lines.append(f"- {issue_name}: {count} run(s).")
    else:
        lines.append("- Существенных проблем качества по рубрикатору не выявлено.")

    lines.extend(
        [
            "",
            "## Per-Run Issues (Evidence)",
        ]
    )
    detailed_runs = [run for run in runs if run.issue_details]
    if not detailed_runs:
        lines.append("- Issues с подтверждёнными evidence не обнаружены.")
    else:
        for run in detailed_runs:
            lines.append(f"- {run.provider} run{run.run_index}:")
            for detail in run.issue_details:
                lines.append(f"  - {detail}")

    lines.extend(
        [
            "",
            "## P0/P1 Actions",
            "- P0: держать nightly batch regression с direct binaries (`qwen`/`claude`/`codex`); frontend live smoke обязателен только для frontend-enabled/release surfaces.",
            "- P0: если встречается `runtime_contract_failed`/`runner_unavailable`, блокировать rollout до фикса runtime contract/provider invocation.",
            "- P1: зафиксировать artifact/UX findings в отдельных SWE-agent assessment reports, не смешивая их с execution verdict.",
        ]
    )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_meta_tsv(path: Path, runs: list[RunEvaluation]) -> None:
    ensure_parent_dir(path)
    header = [
        "provider",
        "run",
        "hard_pass",
        "reliability",
        "contract",
        "analysis",
        "total",
        "verdict",
        "init_signal",
        "refresh_signal",
        "refresh_findings",
        "refresh_questions",
        "refresh_cov_missing",
        "repair_attempts",
        "repair_exhausted",
        "fresh_retries",
        "focused_repairs",
        "stall_count",
        "pre_artifact_stalls",
        "post_artifact_stalls",
        "zero_output_pre_artifact_stalls",
        "partial_failure_count",
        "quality_alerts",
        "artifact_source",
        "semantic_hard_fail",
        "failure_class",
        "runtime_contract_failed",
        "runner_unavailable",
        "runtime_timeout",
        "infra_signal_terminated",
        "infra_incomplete_cycle",
        "summary_missing",
        "precheck_failed",
        "runtime_flow_failed",
        "cancellation_like",
        "artifact_quality_findings",
        "off_topic_hits",
        "issues",
    ]
    lines = ["\t".join(header)]
    for run in runs:
        lines.append(
            "\t".join(
                [
                    run.provider,
                    str(run.run_index),
                    str(int(run.hard_pass)),
                    str(run.reliability),
                    str(run.contract),
                    str(run.analysis),
                    str(run.total),
                    run.verdict,
                    str(run.init_signal),
                    str(run.refresh_signal),
                    str(run.refresh_findings),
                    str(run.refresh_questions),
                    str(run.refresh_cov_missing),
                    str(run.repair_attempts),
                    str(run.repair_exhausted),
                    str(run.fresh_retries),
                    str(run.focused_repairs),
                    str(run.stall_count),
                    str(run.pre_artifact_stalls),
                    str(run.post_artifact_stalls),
                    str(run.zero_output_pre_artifact_stalls),
                    str(run.partial_failure_count),
                    str(run.quality_alerts),
                    run.artifact_source,
                    str(int(run.semantic_hard_fail)),
                    run.failure_class,
                    str(int(run.runtime_contract_failed)),
                    str(int(run.runner_unavailable)),
                    str(int(run.runtime_timeout)),
                    str(int(run.infra_signal_terminated)),
                    str(int(run.infra_incomplete_cycle)),
                    str(int(run.summary_missing)),
                    str(int(run.precheck_failed)),
                    str(int(run.runtime_flow_failed)),
                    str(int(run.cancellation_like)),
                    str(run.artifact_quality_findings),
                    str(run.off_topic_hits),
                    ",".join(run.issues),
                ]
            )
        )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate run/frontend matrices and execution report for the live batch.")
    parser.add_argument("--batch-id", required=True)
    parser.add_argument("--batch-root", required=True)
    parser.add_argument("--reports-root", required=True)
    args = parser.parse_args()

    batch_root = Path(args.batch_root).resolve()
    reports_root = Path(args.reports_root).resolve()
    reports_root.mkdir(parents=True, exist_ok=True)

    preflight_path = batch_root / "preflight.json"
    preflight = read_json(preflight_path) if preflight_path.exists() else {}
    classifications = reconstruct_backend_classifications(batch_root, parse_backend_classifications(batch_root))
    selected_providers = resolve_selected_providers(preflight, classifications, batch_root)
    selected_run_indexes = resolve_selected_run_indexes(preflight, classifications, batch_root)

    runs: list[RunEvaluation] = []
    for provider in selected_providers:
        for run_index in selected_run_indexes:
            run_dir = batch_root / provider / f"run{run_index}"
            runs.append(
                evaluate_run(
                    provider,
                    run_index,
                    run_dir,
                    preflight,
                    classifications.get((provider, run_index)),
                )
            )
    runs.sort(key=lambda item: (item.provider, item.run_index))

    frontend = load_frontend_results(batch_root, selected_providers)

    run_matrix_path = reports_root / f"run_matrix_{args.batch_id}.md"
    frontend_matrix_path = reports_root / f"frontend_e2e_matrix_{args.batch_id}.md"
    execution_report_path = reports_root / f"execution_report_{args.batch_id}.md"
    meta_tsv_path = reports_root / f"run_matrix_{args.batch_id}.tsv"

    write_run_matrix(run_matrix_path, runs)
    write_frontend_matrix(frontend_matrix_path, frontend, selected_providers)
    write_execution_report(execution_report_path, args.batch_id, runs, frontend, preflight, selected_providers)
    write_meta_tsv(meta_tsv_path, runs)

    print(str(run_matrix_path))
    print(str(frontend_matrix_path))
    print(str(execution_report_path))
    print(str(meta_tsv_path))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
