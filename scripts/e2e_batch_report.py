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
    issues: list[str] = field(default_factory=list)
    issue_details: list[str] = field(default_factory=list)
    error_codes: list[str] = field(default_factory=list)


def evaluate_run(provider: str, run_index: int, run_dir: Path) -> RunEvaluation:
    summary_path = run_dir / "session-summary.md"
    run_results_path = run_dir / "run-results.tsv"
    full_run_log = run_dir / "full-run.log"
    workspace = run_dir / "arch-workspace"
    findings_path = workspace / "reports/findings/findings.md"
    overview_path = workspace / "reports/as-is/overview.md"
    coverage_path = workspace / "reports/coverage/summary.md"
    questions_path = workspace / "reports/coverage/open-questions.md"

    issues: list[str] = []
    details: list[str] = []
    error_codes: list[str] = []

    summary_text = summary_path.read_text(encoding="utf-8") if summary_path.exists() else ""
    result_value = parse_markdown_scalar(summary_text, "result").split()[0] if summary_text else ""
    quality_gates_value = parse_markdown_scalar(summary_text, "quality_gates").split()[0] if summary_text else ""
    api_status = parse_api_status(summary_text)

    rows = parse_run_results(run_results_path)
    headless_rows = parse_headless_rows(rows, provider)
    init_row = headless_rows.get("init")
    refresh_row = headless_rows.get("refresh")

    h1 = result_value == "passed" and quality_gates_value == "passed" and api_status == "succeeded"
    if not h1:
        issues.append("reliability:session")
        details.append(f"reliability/session -> {summary_path}: result={result_value} quality_gates={quality_gates_value} api={api_status}")

    h2 = bool(init_row and refresh_row and init_row["status"] == "succeeded" and refresh_row["status"] == "succeeded")
    if not h2:
        issues.append("reliability:headless-status")
        details.append(f"reliability/headless-status -> {run_results_path}: init={init_row['status'] if init_row else 'missing'} refresh={refresh_row['status'] if refresh_row else 'missing'}")

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
    hard_pass = h1 and h2 and h3 and h4

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
        quality_path = Path(str(row["quality_path"]))
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
                details.append(
                    f"contract/metrics-parity -> {quality_path}: {row_key} row={row_value} json={total_value}"
                )

        taskrun_files = sorted((workspace / "reports/taskruns").glob(f"{run_id}-{pipeline}-*.json"))
        if not taskrun_files:
            c1_runtime_name_ok = False
            details.append(f"contract/runtime-name -> {workspace / 'reports/taskruns'}: no step files for run_id={run_id}")
        for taskrun_file in taskrun_files:
            payload = read_json(taskrun_file)
            runtime_name = (
                str((payload.get("meta") or {}).get("runtime", {}).get("name", "")).strip()
            )
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

    overview_ok = False
    findings_ok = False
    coverage_ok = False
    questions_ok = False

    if overview_path.exists():
        non_empty_lines = [line for line in overview_path.read_text(encoding="utf-8").splitlines() if line.strip()]
        placeholder_hit = any("no " in line.lower() and " yet" in line.lower() for line in non_empty_lines)
        overview_ok = len(non_empty_lines) >= 4 and not placeholder_hit
        if not overview_ok:
            details.append(f"analysis/overview -> {overview_path}: non_empty_lines={len(non_empty_lines)} placeholder={int(placeholder_hit)}")
    else:
        details.append(f"analysis/overview -> missing {overview_path}")

    findings_text = findings_path.read_text(encoding="utf-8") if findings_path.exists() else ""
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

    question_ids, question_texts = parse_open_questions(questions_path)
    question_texts_norm = [normalize_text(text) for text in question_texts if text.strip()]
    question_dupes = len(question_texts_norm) != len(set(question_texts_norm))
    questions_ok = len(question_texts_norm) > 0 and not question_dupes
    if not questions_ok:
        details.append(
            f"analysis/questions -> {questions_path}: count={len(question_texts_norm)} dupes={int(question_dupes)}"
        )

    owner_gap = any(term in {"owner mappings", "owner mapping", "owner team mappings", "owner team mapping"} for term in missing_terms)
    if owner_gap and has_empty_marker:
        findings_ok = False
        details.append(
            f"analysis/findings-owner-gap -> {findings_path}: owner gap exists in {coverage_path}, findings are empty"
        )

    if not overview_ok:
        issues.append("analysis:overview")
    if not findings_ok:
        issues.append("analysis:findings")
    if not coverage_ok:
        issues.append("analysis:coverage")
    if not questions_ok:
        issues.append("analysis:questions")

    analysis = bool_score(overview_ok, 10) + bool_score(findings_ok, 10) + bool_score(coverage_ok, 10) + bool_score(questions_ok, 10)

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
        "| provider | run | hard_pass | reliability | contract | analysis | total | verdict | init_signal | refresh_signal | refresh_findings | refresh_questions | refresh_cov_missing | issues |",
        "|---|---:|---:|---:|---:|---:|---:|---|---:|---:|---:|---:|---:|---|",
    ]
    for item in runs:
        lines.append(
            "| "
            f"{item.provider} | {item.run_index} | {int(item.hard_pass)} | {item.reliability} | {item.contract} | "
            f"{item.analysis} | {item.total} | {item.verdict} | {item.init_signal} | {item.refresh_signal} | "
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
        for item in items:
            issues_counter.update(item.issues)
            error_codes_counter.update(item.error_codes)
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
                "error_codes": ", ".join(f"{code}={count}" for code, count in sorted(error_codes_counter.items())) or "-",
                "issues_top": ", ".join(f"{name}={count}" for name, count in issues_counter.most_common(3)) or "-",
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
    issue_counter = Counter()
    for run in runs:
        issue_counter.update(run.issues)

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
        "## Hard-Gates",
        f"- hard_pass_runs: {hard_pass_all}/{len(runs)}",
        "",
        "## Provider Matrix",
        "",
        "| provider | runs | pass_rate | avg_total | std_total | avg_reliability | avg_contract | avg_analysis | avg_refresh_signal | std_refresh_signal | avg_refresh_findings | avg_refresh_questions | avg_refresh_cov_missing | error_codes | frontend_live_pass_rate |",
        "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---:|",
    ]
    for row in provider_rows:
        lines.append(
            "| "
            f"{row['provider']} | {row['runs']} | {row['pass_rate']:.2f} | {row['avg_total']:.2f} | {row['std_total']:.2f} | "
            f"{row['avg_reliability']:.2f} | {row['avg_contract']:.2f} | {row['avg_analysis']:.2f} | "
            f"{row['avg_signal']:.2f} | {row['std_signal']:.2f} | {row['avg_findings']:.2f} | {row['avg_questions']:.2f} | "
            f"{row['avg_cov_missing']:.2f} | {row['error_codes']} | {row['frontend_pass_rate']:.2f} |"
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

    runs: list[RunEvaluation] = []
    for provider in ("qwen-code", "claude-code"):
        for run_index in range(1, 6):
            run_dir = batch_root / provider / f"run{run_index}"
            runs.append(evaluate_run(provider, run_index, run_dir))
    runs.sort(key=lambda item: (item.provider, item.run_index))

    frontend = load_frontend_results(batch_root)
    preflight_path = batch_root / "preflight.json"
    preflight = read_json(preflight_path) if preflight_path.exists() else {}

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
