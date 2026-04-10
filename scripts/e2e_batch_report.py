#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import math
import re
from collections import Counter, defaultdict
from dataclasses import dataclass, field
from pathlib import Path
from statistics import mean, pstdev
from typing import Any

RUN_RESULTS_COLUMNS = [
    "iteration",
    "runtime_mode",
    "runtime_provider",
    "pipeline",
    "run_id",
    "status",
    "signal",
    "changeset",
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


def normalize_text(value: str) -> str:
    cleaned = value.strip().lower().replace("_", " ").replace("-", " ")
    return " ".join(cleaned.split())


def read_json(path: Path) -> dict[str, Any]:
    with path.open(encoding="utf-8") as f:
        return json.load(f)


def parse_markdown_scalar(text: str, key: str) -> str:
    match = re.search(rf"^- {re.escape(key)}:\s*(.+)$", text, flags=re.MULTILINE)
    return match.group(1).strip() if match else ""


def parse_api_status(text: str) -> str:
    match = re.search(r"## API Simulation\s+- status:\s*(\S+)", text, flags=re.MULTILINE)
    return match.group(1).strip() if match else ""


def first_token(value: str) -> str:
    parts = value.split()
    return parts[0] if parts else ""


def parse_run_results(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    if not path.exists():
        return rows
    for raw in path.read_text(encoding="utf-8").splitlines():
        if not raw.strip():
            continue
        parts = raw.split("\t")
        if len(parts) < len(RUN_RESULTS_COLUMNS):
            continue
        record = dict(zip(RUN_RESULTS_COLUMNS, parts))
        for numeric_key in ("signal", "changeset", "findings", "questions", "cov_obs", "cov_missing", "warnings"):
            try:
                record[numeric_key] = int(record[numeric_key])
            except Exception:
                record[numeric_key] = 0
        rows.append(record)
    return rows


def parse_markdown_section_bullets(path: Path, section_title: str) -> list[str]:
    if not path.exists():
        return []
    lines = path.read_text(encoding="utf-8").splitlines()
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
    for line in path.read_text(encoding="utf-8").splitlines():
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


def resolve_reports_root(run_dir: Path, run_id: str) -> tuple[Path, str]:
    snapshot_reports = run_dir / "snapshots" / run_id / "reports"
    if snapshot_reports.exists():
        return snapshot_reports, "snapshot"
    return run_dir / "arch-workspace" / "reports", "workspace-fallback"


def resolve_quality_json(run_dir: Path, row: dict[str, Any]) -> tuple[Path, str]:
    run_id = str(row.get("run_id", "")).strip()
    reports_root, source = resolve_reports_root(run_dir, run_id)
    return reports_root / "taskruns" / f"{run_id}-quality.json", source


def resolve_step_taskrun_files(run_dir: Path, run_id: str, pipeline: str) -> tuple[list[Path], str]:
    reports_root, source = resolve_reports_root(run_dir, run_id)
    files = sorted((reports_root / "taskruns").glob(f"{run_id}-{pipeline}-*.json"))
    return files, source


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


def evidence_path_resolves(path_value: str, target_repo: Path, workspace: Path) -> tuple[bool, str]:
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
        if is_within(candidate, target_repo) or is_within(candidate, workspace):
            return True, "ok"
        return False, "absolute path outside target/workspace"

    candidate_variants = [candidate]
    parts = candidate.parts
    if parts and parts[0] in {"arch-workspace", "workspace"} and len(parts) > 1:
        candidate_variants.append(Path(*parts[1:]))

    for variant in candidate_variants:
        for resolved in (target_repo / variant, workspace / variant):
            if resolved.exists():
                return True, "ok"
    return False, "relative path missing in target/workspace"


def is_power_target(target_repo: Path) -> bool:
    text = str(target_repo).lower()
    return any(hint in text for hint in POWER_TARGET_HINTS)


def collect_off_topic_hits(payload: dict[str, Any]) -> list[str]:
    fragments: list[str] = []
    summary = str(payload.get("summary", "")).strip()
    if summary:
        fragments.append(summary)

    questions = payload.get("questions") or []
    if isinstance(questions, list):
        for question in questions:
            if isinstance(question, dict):
                text = str(question.get("text", "")).strip()
                if text:
                    fragments.append(text)

    changeset = payload.get("changeset") or []
    if isinstance(changeset, list):
        for op in changeset:
            if not isinstance(op, dict):
                continue
            if isinstance(op.get("entity"), dict):
                entity = op["entity"]
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
    artifact_source: str = "snapshot"
    semantic_hard_fail: bool = False
    off_topic_hits: int = 0
    issues: list[str] = field(default_factory=list)
    issue_details: list[str] = field(default_factory=list)
    error_codes: list[str] = field(default_factory=list)


def evaluate_run(provider: str, run_index: int, run_dir: Path, preflight: dict[str, Any]) -> RunEvaluation:
    summary_path = run_dir / "session-summary.md"
    run_results_path = run_dir / "run-results.tsv"
    full_run_log = run_dir / "full-run.log"
    workspace = run_dir / "arch-workspace"
    target_repo = Path(str(preflight.get("target_repo", ""))).resolve() if preflight.get("target_repo") else workspace

    issues: list[str] = []
    details: list[str] = []
    error_codes: list[str] = []

    summary_text = summary_path.read_text(encoding="utf-8") if summary_path.exists() else ""
    result_value = first_token(parse_markdown_scalar(summary_text, "result")) if summary_text else ""
    quality_gates_value = first_token(parse_markdown_scalar(summary_text, "quality_gates")) if summary_text else ""
    api_status = parse_api_status(summary_text)

    rows = parse_run_results(run_results_path)
    headless_rows = parse_headless_rows(rows, provider)
    init_row = headless_rows.get("init")
    refresh_row = headless_rows.get("refresh")

    snapshot_ok = True
    snapshot_semantic_fallback_used = False
    artifact_source = "snapshot"
    for row in (init_row, refresh_row):
        if not row:
            continue
        run_id = str(row.get("run_id", "")).strip()
        reports_root, source = resolve_reports_root(run_dir, run_id)
        if source != "snapshot":
            snapshot_ok = False
            artifact_source = "workspace-fallback"
            details.append(f"reliability/snapshot-missing -> {run_dir / 'snapshots' / run_id}: fallback={reports_root}")

    h1 = result_value == "passed" and quality_gates_value == "passed" and api_status == "succeeded"
    if not h1:
        issues.append("reliability:session")
        details.append(
            f"reliability/session -> {summary_path}: result={result_value} quality_gates={quality_gates_value} api={api_status}"
        )

    h2 = bool(init_row and refresh_row and init_row["status"] == "succeeded" and refresh_row["status"] == "succeeded")
    if not h2:
        issues.append("reliability:headless-status")
        details.append(
            f"reliability/headless-status -> {run_results_path}: init={init_row['status'] if init_row else 'missing'} "
            f"refresh={refresh_row['status'] if refresh_row else 'missing'}"
        )

    runner_error_hit = False
    for source_path in (summary_path, full_run_log):
        if not source_path.exists():
            continue
        text = source_path.read_text(encoding="utf-8")
        for code in ("runner_unavailable", "runner_parse_failed"):
            if code in text:
                runner_error_hit = True
                error_codes.append(code)
    h3 = not runner_error_hit
    if not h3:
        issues.append("reliability:runner-errors")
        details.append(f"reliability/runner-errors -> {full_run_log}: detected {sorted(set(error_codes))}")

    init_signal = int(init_row["signal"]) if init_row else 0
    refresh_signal = int(refresh_row["signal"]) if refresh_row else 0
    h4 = init_signal > 0 and refresh_signal > 0
    if not h4:
        issues.append("reliability:signal")
        details.append(f"reliability/signal -> {run_results_path}: init_signal={init_signal} refresh_signal={refresh_signal}")

    reliability = bool_score(h1, 10) + bool_score(h2, 10) + bool_score(h3, 10) + bool_score(h4, 10)
    if not snapshot_ok:
        issues.append("reliability:snapshot-missing")
        reliability = max(0, reliability - 10)

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
        quality_path, quality_source = resolve_quality_json(run_dir, row)
        if quality_source != "snapshot":
            artifact_source = "workspace-fallback"
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
            ("changeset", "changeset_ops"),
            ("findings", "findings_added"),
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
            runtime_name = str((payload.get("meta") or {}).get("runtime", {}).get("name", "")).strip()
            if not runtime_name:
                c1_runtime_name_ok = False
                details.append(f"contract/runtime-name -> {taskrun_file}: empty meta.runtime.name")
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
        analysis_reports_root, analysis_source = resolve_reports_root(run_dir, analysis_run_id)
        if analysis_source != "snapshot":
            artifact_source = "workspace-fallback"
    findings_path = analysis_reports_root / "findings/findings.md"
    overview_path = analysis_reports_root / "as-is/overview.md"
    coverage_path = analysis_reports_root / "coverage/summary.md"
    questions_path = analysis_reports_root / "coverage/open-questions.md"

    overview_ok = False
    findings_ok = False
    coverage_ok = False
    questions_ok = False

    if overview_path.exists():
        non_empty_lines = [line for line in overview_path.read_text(encoding="utf-8").splitlines() if line.strip()]
        placeholder_hit = any("no " in line.lower() and " yet" in line.lower() for line in non_empty_lines)
        overview_ok = len(non_empty_lines) >= 4 and not placeholder_hit
        if not overview_ok:
            details.append(
                f"analysis/overview -> {overview_path}: non_empty_lines={len(non_empty_lines)} placeholder={int(placeholder_hit)}"
            )
    else:
        details.append(f"analysis/overview -> missing {overview_path}")

    findings_text = findings_path.read_text(encoding="utf-8") if findings_path.exists() else ""
    findings_text_lower = findings_text.lower()
    has_finding_heading = "## " in findings_text
    has_severity = "- Severity:" in findings_text
    has_description = "- Description:" in findings_text
    has_empty_marker = "No findings reported." in findings_text
    findings_ok = has_finding_heading and has_severity and has_description and not has_empty_marker
    if not findings_ok:
        details.append(
            f"analysis/findings -> {findings_path}: heading={int(has_finding_heading)} severity={int(has_severity)} "
            f"description={int(has_description)} empty_marker={int(has_empty_marker)}"
        )

    missing_terms_raw = parse_markdown_section_bullets(coverage_path, "Missing")
    missing_terms = [normalize_text(term) for term in missing_terms_raw]
    missing_dupes = len(missing_terms) != len(set(missing_terms))
    notes_raw = parse_markdown_section_bullets(coverage_path, "Notes")
    notes_norm = [normalize_text(note) for note in notes_raw]
    notes_dupes = len(notes_norm) != len(set(notes_norm))
    coverage_ok = len(missing_terms) > 0 and not missing_dupes and not notes_dupes
    if not coverage_ok:
        details.append(
            f"analysis/coverage -> {coverage_path}: missing={len(missing_terms)} missing_dupes={int(missing_dupes)} notes_dupes={int(notes_dupes)}"
        )

    _question_ids, question_texts = parse_open_questions(questions_path)
    question_texts_norm = [normalize_text(text) for text in question_texts if text.strip()]
    question_dupes = len(question_texts_norm) != len(set(question_texts_norm))
    questions_ok = len(question_texts_norm) > 0 and not question_dupes
    if not questions_ok:
        details.append(f"analysis/questions -> {questions_path}: count={len(question_texts_norm)} dupes={int(question_dupes)}")

    owner_gap = any(term in {"owner mappings", "owner mapping", "owner team mappings", "owner team mapping"} for term in missing_terms)
    if owner_gap and has_empty_marker:
        findings_ok = False
        details.append(f"analysis/findings-owner-gap -> {findings_path}: owner gap exists in {coverage_path}, findings are empty")

    semantic_hard_fail = False
    off_topic_hits = 0

    refresh_step_files: list[Path] = []
    if refresh_row:
        refresh_run_id = str(refresh_row.get("run_id", ""))
        refresh_step_files, refresh_source = resolve_step_taskrun_files(run_dir, refresh_run_id, "refresh")
        if refresh_source != "snapshot":
            artifact_source = "workspace-fallback"
        if not refresh_step_files:
            fallback_glob = sorted((workspace / "reports" / "taskruns").glob(f"{refresh_run_id}-refresh-*.json"))
            if fallback_glob:
                refresh_step_files = fallback_glob
                snapshot_semantic_fallback_used = True
                artifact_source = "workspace-fallback"
                details.append(
                    f"reliability/snapshot-missing -> missing refresh step taskruns in snapshot, fallback to workspace reports for run_id={refresh_run_id}"
                )

    if refresh_step_files:
        non_power_target = not is_power_target(target_repo)
        step1_files = [path for path in refresh_step_files if "-step1-collect-" in path.name]
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
                ok, reason = evidence_path_resolves(evidence_path, target_repo, workspace)
                if not ok:
                    invalid_evidence.append(f"{step_file} :: {evidence_path} ({reason})")
        if invalid_evidence:
            semantic_hard_fail = True
            issues.append("analysis:evidence-scope")
            for item in invalid_evidence[:8]:
                details.append(f"analysis/evidence-scope -> {item}")
            if len(invalid_evidence) > 8:
                details.append(f"analysis/evidence-scope -> +{len(invalid_evidence) - 8} more invalid evidence paths")

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

    if snapshot_semantic_fallback_used and snapshot_ok:
        snapshot_ok = False
        if "reliability:snapshot-missing" not in issues:
            issues.append("reliability:snapshot-missing")
        reliability = max(0, reliability - 10)

    if not overview_ok:
        issues.append("analysis:overview")
    if not findings_ok:
        issues.append("analysis:findings")
    if not coverage_ok:
        issues.append("analysis:coverage")
    if not questions_ok:
        issues.append("analysis:questions")

    analysis = bool_score(overview_ok, 10) + bool_score(findings_ok, 10) + bool_score(coverage_ok, 10) + bool_score(questions_ok, 10)
    hard_pass = h1 and h2 and h3 and h4 and snapshot_ok and not semantic_hard_fail

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
        artifact_source=artifact_source,
        semantic_hard_fail=semantic_hard_fail,
        off_topic_hits=off_topic_hits,
        issues=sorted(set(issues)),
        issue_details=details,
        error_codes=sorted(set(error_codes)),
    )


def load_frontend_results(batch_root: Path) -> dict[str, dict[str, Any]]:
    data: dict[str, dict[str, Any]] = {}
    for provider in ("qwen-code", "claude-code"):
        path = batch_root / "frontend" / provider / "frontend-e2e-result.json"
        if path.exists():
            payload = read_json(path)
            payload["path"] = str(path)
            data[provider] = payload
    return data


def write_run_matrix(path: Path, runs: list[RunEvaluation]) -> None:
    lines = [
        "# Run Matrix",
        "",
        "| provider | run | hard_pass | reliability | contract | analysis | total | verdict | artifact_source | semantic_hard_fail | off_topic_hits | init_signal | refresh_signal | refresh_findings | refresh_questions | refresh_cov_missing | issues |",
        "|---|---:|---:|---:|---:|---:|---:|---|---|---:|---:|---:|---:|---:|---:|---:|---|",
    ]
    for item in runs:
        lines.append(
            "| "
            f"{item.provider} | {item.run_index} | {int(item.hard_pass)} | {item.reliability} | {item.contract} | "
            f"{item.analysis} | {item.total} | {item.verdict} | {item.artifact_source} | {int(item.semantic_hard_fail)} | {item.off_topic_hits} | "
            f"{item.init_signal} | {item.refresh_signal} | "
            f"{item.refresh_findings} | {item.refresh_questions} | {item.refresh_cov_missing} | "
            f"{', '.join(item.issues) if item.issues else '-'} |"
        )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_frontend_matrix(path: Path, frontend: dict[str, dict[str, Any]]) -> None:
    lines = [
        "# Frontend Live E2E Matrix",
        "",
        "| provider | status | base_url | workspace | runtime_command | server_log | playwright_log |",
        "|---|---|---|---|---|---|---|",
    ]
    for provider in ("qwen-code", "claude-code"):
        payload = frontend.get(provider)
        if not payload:
            lines.append(f"| {provider} | missing | - | - | - | - | - |")
            continue
        lines.append(
            "| "
            f"{provider} | {payload.get('status', '-') } | {payload.get('base_url', '-')} | "
            f"{payload.get('workspace', '-')} | {payload.get('runtime_command', '-')} | "
            f"{payload.get('server_log', '-')} | {payload.get('playwright_log', '-')} |"
        )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def provider_matrix_rows(
    runs: list[RunEvaluation], frontend: dict[str, dict[str, Any]]
) -> list[dict[str, Any]]:
    grouped: dict[str, list[RunEvaluation]] = defaultdict(list)
    for run in runs:
        grouped[run.provider].append(run)

    rows: list[dict[str, Any]] = []
    for provider in ("qwen-code", "claude-code"):
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
        frontend_status = frontend.get(provider, {}).get("status")
        frontend_pass_rate = 1.0 if frontend_status == "passed" else 0.0 if frontend_status else 0.0
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
                "off_topic_hits": sum(item.off_topic_hits for item in items),
                "semantic_hard_fail_runs": sum(1 for item in items if item.semantic_hard_fail),
                "error_codes": ", ".join(f"{code}={count}" for code, count in sorted(error_codes_counter.items())) or "-",
                "issues_top": ", ".join(f"{name}={count}" for name, count in issues_counter.most_common(3)) or "-",
                "artifact_sources": ", ".join(f"{name}={count}" for name, count in sorted(artifact_sources.items())) or "-",
                "frontend_pass_rate": frontend_pass_rate,
            }
        )
    return rows


def write_quality_report(
    path: Path,
    batch_id: str,
    runs: list[RunEvaluation],
    frontend: dict[str, dict[str, Any]],
    preflight: dict[str, Any],
) -> None:
    provider_rows = provider_matrix_rows(runs, frontend)
    hard_pass_all = sum(1 for run in runs if run.hard_pass)
    semantic_hard_fail_runs = sum(1 for run in runs if run.semantic_hard_fail)
    issue_counter = Counter()
    for run in runs:
        issue_counter.update(run.issues)
    snapshot_runs = sum(1 for run in runs if run.artifact_source == "snapshot")
    workspace_fallback_runs = len(runs) - snapshot_runs

    lines = [
        f"# Quality Report: {batch_id}",
        "",
        "## Context",
        f"- generated_at_utc: {preflight.get('generated_at_utc', '-')}",
        f"- provenarch_sha: {preflight.get('provenarch_sha', '-')}",
        f"- target_repo: {preflight.get('target_repo', '-')}",
        f"- target_repo_sha: {preflight.get('target_repo_sha', '-')}",
        f"- claude: {((preflight.get('runtimes') or {}).get('claude') or {}).get('version_line', '-')}",
        f"- qwen: {((preflight.get('runtimes') or {}).get('qwen') or {}).get('version_line', '-')}",
        "",
        "## Backend Quality Verdict (source-of-truth)",
        f"- hard_pass_runs: {hard_pass_all}/{len(runs)}",
        f"- semantic_hard_fail_runs: {semantic_hard_fail_runs}/{len(runs)}",
        f"- artifact_source_snapshot_runs: {snapshot_runs}/{len(runs)}",
        f"- artifact_source_workspace_fallback_runs: {workspace_fallback_runs}/{len(runs)}",
        "",
        "## Frontend Live Smoke Verdict",
        f"- qwen-code: {(frontend.get('qwen-code') or {}).get('status', 'missing')}",
        f"- claude-code: {(frontend.get('claude-code') or {}).get('status', 'missing')}",
        "",
        "## Provider Matrix",
        "",
        "| provider | runs | pass_rate | avg_total | std_total | avg_reliability | avg_contract | avg_analysis | avg_refresh_signal | std_refresh_signal | avg_refresh_findings | avg_refresh_questions | avg_refresh_cov_missing | off_topic_hits | semantic_hard_fail_runs | artifact_sources | error_codes | frontend_live_pass_rate |",
        "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---:|",
    ]
    for row in provider_rows:
        lines.append(
            "| "
            f"{row['provider']} | {row['runs']} | {row['pass_rate']:.2f} | {row['avg_total']:.2f} | {row['std_total']:.2f} | "
            f"{row['avg_reliability']:.2f} | {row['avg_contract']:.2f} | {row['avg_analysis']:.2f} | "
            f"{row['avg_signal']:.2f} | {row['std_signal']:.2f} | {row['avg_findings']:.2f} | {row['avg_questions']:.2f} | "
            f"{row['avg_cov_missing']:.2f} | {row['off_topic_hits']} | {row['semantic_hard_fail_runs']} | "
            f"{row['artifact_sources']} | {row['error_codes']} | {row['frontend_pass_rate']:.2f} |"
        )

    lines.extend(
        [
            "",
            "## Что Сделано Качественно",
        ]
    )
    if hard_pass_all == len(runs):
        lines.append("- Все `10/10` backend full-runs прошли hard-gates без падений pipeline и signal regression.")
    else:
        lines.append("- Часть run прошла hard-gates, но есть деградации стабильности (см. matrix).")
    if all((frontend.get(provider) or {}).get("status") == "passed" for provider in ("qwen-code", "claude-code")):
        lines.append("- Frontend live e2e прошёл для обоих провайдеров (`2/2`).")
    else:
        lines.append("- Frontend live e2e не полностью стабилен (`<2/2`).")
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
            "- P0: держать nightly `5x2` regression с direct binaries (`qwen`/`claude`) и обязательным frontend live smoke `2/2`.",
            "- P0: если встречается `runner_parse_failed`/`runner_unavailable`, блокировать rollout до фикса runtime invocation/parsing.",
            "- P1: расширить semantic quality rubric на richer evidence density в findings (rule/evidence refs) и cross-doc consistency checks.",
        ]
    )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def write_meta_tsv(path: Path, runs: list[RunEvaluation]) -> None:
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
        "artifact_source",
        "semantic_hard_fail",
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
                    run.artifact_source,
                    str(int(run.semantic_hard_fail)),
                    str(run.off_topic_hits),
                    ",".join(run.issues),
                ]
            )
        )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate run/frontend matrices and quality report for 5x2 batch.")
    parser.add_argument("--batch-id", required=True)
    parser.add_argument("--batch-root", required=True)
    parser.add_argument("--reports-root", required=True)
    args = parser.parse_args()

    batch_root = Path(args.batch_root).resolve()
    reports_root = Path(args.reports_root).resolve()
    reports_root.mkdir(parents=True, exist_ok=True)

    preflight_path = batch_root / "preflight.json"
    preflight = read_json(preflight_path) if preflight_path.exists() else {}

    runs: list[RunEvaluation] = []
    for provider in ("qwen-code", "claude-code"):
        for run_index in range(1, 6):
            run_dir = batch_root / provider / f"run{run_index}"
            runs.append(evaluate_run(provider, run_index, run_dir, preflight))
    runs.sort(key=lambda item: (item.provider, item.run_index))

    frontend = load_frontend_results(batch_root)

    run_matrix_path = reports_root / f"run_matrix_{args.batch_id}.md"
    frontend_matrix_path = reports_root / f"frontend_e2e_matrix_{args.batch_id}.md"
    quality_report_path = reports_root / f"quality_report_{args.batch_id}.md"
    meta_tsv_path = reports_root / f"run_matrix_{args.batch_id}.tsv"

    write_run_matrix(run_matrix_path, runs)
    write_frontend_matrix(frontend_matrix_path, frontend)
    write_quality_report(quality_report_path, args.batch_id, runs, frontend, preflight)
    write_meta_tsv(meta_tsv_path, runs)

    print(str(run_matrix_path))
    print(str(frontend_matrix_path))
    print(str(quality_report_path))
    print(str(meta_tsv_path))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
