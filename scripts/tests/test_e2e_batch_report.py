#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path


def load_report_module():
    repo_root = Path(__file__).resolve().parents[2]
    module_path = repo_root / "scripts" / "e2e_batch_report.py"
    spec = importlib.util.spec_from_file_location("e2e_batch_report", module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"failed to load module spec: {module_path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


report = load_report_module()


def write_text(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def write_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def write_workspace_validate(path: Path, resolved_repos: list[dict]) -> None:
    write_json(
        path,
        {
            "ok": True,
            "workspace": str(path.parent / "arch-workspace"),
            "warnings": [],
            "errors": [],
            "resolved_repos": resolved_repos,
        },
    )


def make_session_summary(path: Path) -> None:
    write_text(
        path,
        "\n".join(
            [
                "# ProvenArch Full Run Session Summary",
                "",
                "- result: passed",
                "- quality_gates: passed",
                "",
                "## API Simulation",
                "- status: succeeded",
                "",
            ]
        ),
    )


def make_run_results_line(
    provider: str,
    pipeline: str,
    run_id: str,
    signal: int,
    changeset: int,
    findings: int,
    questions: int,
    cov_obs: int,
    cov_missing: int,
    warnings: int,
    runtime_versions: str,
) -> str:
    return "\t".join(
        [
            "1",
            "headless",
            provider,
            pipeline,
            run_id,
            "succeeded",
            str(signal),
            str(changeset),
            str(findings),
            str(questions),
            str(cov_obs),
            str(cov_missing),
            str(warnings),
            runtime_versions,
            "/tmp/quality.json",
            "/tmp/output.log",
        ]
    )


def make_quality_payload(
    run_id: str,
    pipeline: str,
    provider: str,
    signal: int,
    changeset: int,
    findings: int,
    questions: int,
    cov_obs: int,
    cov_missing: int,
    warnings: int,
    runtime_version: str,
    step_ids: list[str],
) -> dict:
    return {
        "version": 1,
        "run_id": run_id,
        "pipeline": pipeline,
        "status": "succeeded",
        "generated_at": "2026-04-10T10:00:00Z",
        "runtime_versions": [f"{provider}@{runtime_version}"],
        "totals": {
            "steps": len(step_ids),
            "changeset_ops": changeset,
            "entity_upserts": max(changeset - findings, 0),
            "edge_upserts": 0,
            "findings_added": findings,
            "questions_count": questions,
            "coverage_observed": cov_obs,
            "coverage_missing": cov_missing,
            "warnings_count": warnings,
            "signal_score": signal,
        },
        "steps": [
            {
                "step_id": step_id,
                "runtime_name": provider,
                "runtime_version": runtime_version,
                "repo_scopes": ["repo-main"],
                "changeset_ops": 1,
                "entity_upserts": 1,
                "edge_upserts": 0,
                "findings_added": 0 if "step1" in step_id else 1,
                "questions_count": 1,
                "coverage_observed": 1,
                "coverage_missing": 1,
                "warnings_count": 0,
            }
            for step_id in step_ids
        ],
    }


def make_step_payload(provider: str, run_id: str, step_id: str, evidence_path: str, summary: str = "ok") -> dict:
    return {
        "meta": {
            "task_id": f"task-{run_id}-{step_id}",
            "run_id": run_id,
            "step_id": step_id,
            "runtime": {"name": provider, "version": "1.2.3"},
            "started_at": "2026-04-10T10:00:00Z",
        },
        "summary": summary,
        "changeset": [
            {
                "op": "upsert_entity",
                "entity": {
                    "id": "svc.orders",
                    "type": "service",
                    "name": "Orders Service",
                    "provenance": {
                        "kind": "observation",
                        "confidence": 0.8,
                        "evidence": [{"repo": "repo-main", "path": evidence_path}],
                    },
                },
            }
        ],
        "questions": [{"id": "q.refresh.owner", "text": "Who owns orders service?"}],
        "coverage": {"observed": ["services"], "missing": ["owner_mappings"], "notes": ["owner mapping is unclear"]},
    }


def write_analysis_reports(reports_root: Path, *, findings_text: str) -> None:
    write_text(
        reports_root / "as-is/overview.md",
        "\n".join(
            [
                "# Overview",
                "",
                "- services: 2",
                "- datastores: 1",
                "- integrations: 1",
                "- teams: 1",
                "",
            ]
        ),
    )
    write_text(reports_root / "findings/findings.md", findings_text)
    write_text(
        reports_root / "coverage/summary.md",
        "\n".join(
            [
                "# Coverage",
                "",
                "## Missing",
                "- owner_mappings",
                "- ci_cd_evidence",
                "",
                "## Notes",
                "- owner mapping remains unclear",
                "",
            ]
        ),
    )
    write_text(
        reports_root / "coverage/open-questions.md",
        "\n".join(["# Open Questions", "", "- `q.owner.1` Who is the owning team for orders service?", ""]),
    )


class EvaluateRunTests(unittest.TestCase):
    def test_evidence_path_resolves_workspace_prefixed_reference(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            target_repo = root / "target-repo"
            workspace = root / "arch-workspace"
            write_text(target_repo / "README.md", "# repo\n")
            write_text(workspace / "reports/as-is/overview.md", "# overview\n")

            ok, reason = report.evidence_path_resolves(
                "arch-workspace/reports/as-is/overview.md#L12",
                [target_repo],
                workspace,
            )
            self.assertTrue(ok, msg=reason)

    def test_evidence_path_resolves_against_multiple_repo_roots(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            repo_a = root / "repo-a"
            repo_b = root / "repo-b"
            workspace = root / "arch-workspace"
            write_text(repo_a / "README.md", "# repo a\n")
            write_text(repo_b / "services/orders/main.go", "package orders\n")
            write_text(workspace / "reports/as-is/overview.md", "# overview\n")

            ok_a, reason_a = report.evidence_path_resolves("README.md", [repo_a, repo_b], workspace)
            self.assertTrue(ok_a, msg=reason_a)

            ok_b, reason_b = report.evidence_path_resolves("services/orders/main.go:12", [repo_a, repo_b], workspace)
            self.assertTrue(ok_b, msg=reason_b)

            ok_bad, reason_bad = report.evidence_path_resolves("search_source/example.com", [repo_a, repo_b], workspace)
            self.assertFalse(ok_bad, msg=reason_bad)

    def test_normalize_declared_repos_meta_supports_new_and_legacy_preflight(self) -> None:
        new_meta = report.normalize_declared_repos_meta(
            {
                "declared_repos_meta": {
                    "target_repos_file": "/tmp/matrix/single-path.yaml",
                    "profile_id": "single-path",
                    "profile_source_kind": "path",
                    "expected_repo_count": 2,
                    "declared_repos": [
                        {"name": "repo-a", "source": "path", "path": "/tmp/repo-a"},
                        {"name": "repo-b", "source": "path", "path": "/tmp/repo-b"},
                    ],
                }
            }
        )
        self.assertEqual(2, new_meta["expected_repo_count"])
        self.assertEqual("single-path", new_meta["profile_id"])
        self.assertEqual(2, len(new_meta["declared_repos"]))

        legacy_meta = report.normalize_declared_repos_meta({"target_repo": "/tmp/legacy-repo"})
        self.assertEqual(1, legacy_meta["expected_repo_count"])
        self.assertEqual(1, len(legacy_meta["declared_repos"]))
        self.assertEqual("/tmp/legacy-repo", legacy_meta["declared_repos"][0]["path"])

    def test_prefers_snapshot_artifacts_for_quality(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            target_repo = root / "target-repo"
            write_text(target_repo / "README.md", "# demo\n")

            run_dir = root / "batch/qwen-code/run1"
            run_dir.mkdir(parents=True, exist_ok=True)
            make_session_summary(run_dir / "session-summary.md")
            write_text(run_dir / "full-run.log", "")

            init_id = "run_init_1"
            refresh_id = "run_refresh_1"
            runtime_versions = "qwen-code@1.2.3"
            write_text(
                run_dir / "run-results.tsv",
                "\n".join(
                    [
                        make_run_results_line("qwen-code", "init", init_id, 12, 2, 1, 1, 2, 2, 0, runtime_versions),
                        make_run_results_line("qwen-code", "refresh", refresh_id, 14, 2, 1, 1, 2, 2, 0, runtime_versions),
                    ]
                )
                + "\n",
            )

            init_reports = run_dir / "snapshots" / init_id / "reports"
            refresh_reports = run_dir / "snapshots" / refresh_id / "reports"

            write_json(
                init_reports / "taskruns" / f"{init_id}-quality.json",
                make_quality_payload(init_id, "init", "qwen-code", 12, 2, 1, 1, 2, 2, 0, "1.2.3", ["init.step1.collect"]),
            )
            write_json(
                refresh_reports / "taskruns" / f"{refresh_id}-quality.json",
                make_quality_payload(
                    refresh_id,
                    "refresh",
                    "qwen-code",
                    14,
                    2,
                    1,
                    1,
                    2,
                    2,
                    0,
                    "1.2.3",
                    ["refresh.step1.collect", "refresh.step3.findings"],
                ),
            )

            write_json(
                init_reports / "taskruns" / f"{init_id}-init-step1-collect-domain-main.json",
                make_step_payload("qwen-code", init_id, "init.step1.collect", "README.md:10"),
            )
            write_json(
                refresh_reports / "taskruns" / f"{refresh_id}-refresh-step1-collect-domain-main.json",
                make_step_payload("qwen-code", refresh_id, "refresh.step1.collect", "README.md#L3"),
            )
            write_json(
                refresh_reports / "taskruns" / f"{refresh_id}-refresh-step3-findings.json",
                make_step_payload("qwen-code", refresh_id, "refresh.step3.findings", "README.md"),
            )

            write_analysis_reports(
                refresh_reports,
                findings_text="\n".join(
                    [
                        "# Findings",
                        "",
                        "## Missing Owner Mapping",
                        "- Severity: high",
                        "- Description: Ownership is missing for orders service.",
                        "",
                    ]
                ),
            )

            result = report.evaluate_run("qwen-code", 1, run_dir, {"target_repo": str(target_repo)})
            self.assertTrue(result.hard_pass)
            self.assertEqual("snapshot", result.artifact_source)
            self.assertFalse(result.semantic_hard_fail)
            self.assertNotIn("reliability:snapshot-missing", result.issues)
            self.assertNotIn("analysis:evidence-scope", result.issues)

    def test_semantic_hard_fail_on_off_topic_evidence_scope_and_cross_doc(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            target_repo = root / "target-repo"
            write_text(target_repo / "README.md", "# demo\n")

            run_dir = root / "batch/qwen-code/run1"
            run_dir.mkdir(parents=True, exist_ok=True)
            make_session_summary(run_dir / "session-summary.md")
            write_text(run_dir / "full-run.log", "")

            init_id = "run_init_2"
            refresh_id = "run_refresh_2"
            runtime_versions = "qwen-code@1.2.3"
            write_text(
                run_dir / "run-results.tsv",
                "\n".join(
                    [
                        make_run_results_line("qwen-code", "init", init_id, 12, 2, 1, 1, 2, 2, 0, runtime_versions),
                        make_run_results_line("qwen-code", "refresh", refresh_id, 14, 2, 1, 1, 2, 2, 0, runtime_versions),
                    ]
                )
                + "\n",
            )

            init_reports = run_dir / "snapshots" / init_id / "reports"
            refresh_reports = run_dir / "snapshots" / refresh_id / "reports"

            write_json(
                init_reports / "taskruns" / f"{init_id}-quality.json",
                make_quality_payload(init_id, "init", "qwen-code", 12, 2, 1, 1, 2, 2, 0, "1.2.3", ["init.step1.collect"]),
            )
            write_json(
                refresh_reports / "taskruns" / f"{refresh_id}-quality.json",
                make_quality_payload(
                    refresh_id,
                    "refresh",
                    "qwen-code",
                    14,
                    2,
                    1,
                    1,
                    2,
                    2,
                    0,
                    "1.2.3",
                    ["refresh.step1.collect", "refresh.step3.findings"],
                ),
            )
            write_json(
                init_reports / "taskruns" / f"{init_id}-init-step1-collect-domain-main.json",
                make_step_payload("qwen-code", init_id, "init.step1.collect", "README.md"),
            )

            step1_payload = make_step_payload(
                "qwen-code",
                refresh_id,
                "refresh.step1.collect",
                "search_source/chinabidding.cn",
                summary="Collected bidding and tender records for power system analysis",
            )
            step1_payload["changeset"][0]["entity"]["id"] = "external.chinabidding"
            step1_payload["changeset"][0]["entity"]["name"] = "China Bidding"
            step1_payload["questions"] = [{"id": "q.bidding.1", "text": "Which tender announcements changed this week?"}]
            write_json(refresh_reports / "taskruns" / f"{refresh_id}-refresh-step1-collect-domain-main.json", step1_payload)
            write_json(
                refresh_reports / "taskruns" / f"{refresh_id}-refresh-step3-findings.json",
                make_step_payload("qwen-code", refresh_id, "refresh.step3.findings", "README.md"),
            )

            write_analysis_reports(
                refresh_reports,
                findings_text="\n".join(
                    [
                        "# Findings",
                        "",
                        "## Service Contracts Drift",
                        "- Severity: high",
                        "- Description: missing service definition files block validation.",
                        "",
                    ]
                ),
            )

            result = report.evaluate_run("qwen-code", 1, run_dir, {"target_repo": str(target_repo)})
            self.assertFalse(result.hard_pass)
            self.assertTrue(result.semantic_hard_fail)
            self.assertGreater(result.off_topic_hits, 0)
            self.assertIn("analysis:off-topic", result.issues)
            self.assertIn("analysis:evidence-scope", result.issues)
            self.assertIn("analysis:cross-doc", result.issues)
            self.assertIn("reliability:semantic-hard-fail", result.issues)

    def test_multi_repo_cross_repo_missing_is_hard_fail(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            repo_a = root / "repo-a"
            repo_b = root / "repo-b"
            write_text(repo_a / "README.md", "# repo a\n")
            write_text(repo_b / "README.md", "# repo b\n")

            run_dir = root / "batch/qwen-code/run1"
            run_dir.mkdir(parents=True, exist_ok=True)
            make_session_summary(run_dir / "session-summary.md")
            write_text(run_dir / "full-run.log", "")
            write_workspace_validate(
                run_dir / "workspace-validate.json",
                [
                    {"name": "repo-a", "source": "path", "path": str(repo_a)},
                    {"name": "repo-b", "source": "path", "path": str(repo_b)},
                ],
            )

            init_id = "run_init_m1"
            refresh_id = "run_refresh_m1"
            runtime_versions = "qwen-code@1.2.3"
            write_text(
                run_dir / "run-results.tsv",
                "\n".join(
                    [
                        make_run_results_line("qwen-code", "init", init_id, 12, 2, 1, 1, 2, 2, 0, runtime_versions),
                        make_run_results_line("qwen-code", "refresh", refresh_id, 13, 2, 1, 1, 2, 2, 0, runtime_versions),
                    ]
                )
                + "\n",
            )

            init_reports = run_dir / "snapshots" / init_id / "reports"
            refresh_reports = run_dir / "snapshots" / refresh_id / "reports"
            write_json(
                init_reports / "taskruns" / f"{init_id}-quality.json",
                make_quality_payload(init_id, "init", "qwen-code", 12, 2, 1, 1, 2, 2, 0, "1.2.3", ["init.step1.collect"]),
            )
            write_json(
                refresh_reports / "taskruns" / f"{refresh_id}-quality.json",
                make_quality_payload(
                    refresh_id,
                    "refresh",
                    "qwen-code",
                    13,
                    2,
                    1,
                    1,
                    2,
                    2,
                    0,
                    "1.2.3",
                    ["refresh.step1.collect", "refresh.step3.findings"],
                ),
            )
            write_json(
                init_reports / "taskruns" / f"{init_id}-init-step1-collect-domain-main.json",
                make_step_payload("qwen-code", init_id, "init.step1.collect", "README.md"),
            )
            step1 = make_step_payload("qwen-code", refresh_id, "refresh.step1.collect", "README.md")
            step1["meta"]["repo_scopes"] = ["repo-a"]
            write_json(refresh_reports / "taskruns" / f"{refresh_id}-refresh-step1-collect-domain-main.json", step1)
            step3 = make_step_payload("qwen-code", refresh_id, "refresh.step3.findings", "README.md")
            step3["meta"]["repo_scopes"] = ["repo-a"]
            write_json(refresh_reports / "taskruns" / f"{refresh_id}-refresh-step3-findings.json", step3)

            write_analysis_reports(
                refresh_reports,
                findings_text="\n".join(
                    [
                        "# Findings",
                        "",
                        "## Missing Owner Mapping",
                        "- Severity: medium",
                        "- Description: Ownership is partially unknown.",
                        "",
                    ]
                ),
            )

            preflight = {
                "declared_repos_meta": {
                    "target_repos_file": str(root / "profiles/multi-path.yaml"),
                    "profile_id": "multi-path",
                    "profile_source_kind": "path",
                    "expected_repo_count": 2,
                    "declared_repos": [
                        {"name": "repo-a", "source": "path", "path": str(repo_a), "ref": ""},
                        {"name": "repo-b", "source": "path", "path": str(repo_b), "ref": ""},
                    ],
                }
            }
            result = report.evaluate_run("qwen-code", 1, run_dir, preflight)
            self.assertFalse(result.hard_pass)
            self.assertTrue(result.semantic_hard_fail)
            self.assertIn("analysis:cross-repo-missing", result.issues)

    def test_multi_repo_cross_repo_signal_passes_when_mentions_and_edge_exist(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            repo_a = root / "repo-a"
            repo_b = root / "repo-b"
            write_text(repo_a / "README.md", "# repo a\n")
            write_text(repo_b / "README.md", "# repo b\n")

            run_dir = root / "batch/claude-code/run1"
            run_dir.mkdir(parents=True, exist_ok=True)
            make_session_summary(run_dir / "session-summary.md")
            write_text(run_dir / "full-run.log", "")
            write_workspace_validate(
                run_dir / "workspace-validate.json",
                [
                    {"name": "repo-a", "source": "path", "path": str(repo_a)},
                    {"name": "repo-b", "source": "path", "path": str(repo_b)},
                ],
            )

            init_id = "run_init_m2"
            refresh_id = "run_refresh_m2"
            runtime_versions = "claude-code@1.2.3"
            write_text(
                run_dir / "run-results.tsv",
                "\n".join(
                    [
                        make_run_results_line("claude-code", "init", init_id, 11, 2, 1, 1, 2, 2, 0, runtime_versions),
                        make_run_results_line("claude-code", "refresh", refresh_id, 15, 3, 1, 1, 2, 2, 0, runtime_versions),
                    ]
                )
                + "\n",
            )

            init_reports = run_dir / "snapshots" / init_id / "reports"
            refresh_reports = run_dir / "snapshots" / refresh_id / "reports"
            write_json(
                init_reports / "taskruns" / f"{init_id}-quality.json",
                make_quality_payload(init_id, "init", "claude-code", 11, 2, 1, 1, 2, 2, 0, "1.2.3", ["init.step1.collect"]),
            )
            write_json(
                refresh_reports / "taskruns" / f"{refresh_id}-quality.json",
                make_quality_payload(
                    refresh_id,
                    "refresh",
                    "claude-code",
                    15,
                    3,
                    1,
                    1,
                    2,
                    2,
                    0,
                    "1.2.3",
                    ["refresh.step1.collect", "refresh.step3.findings"],
                ),
            )
            write_json(
                init_reports / "taskruns" / f"{init_id}-init-step1-collect-domain-main.json",
                make_step_payload("claude-code", init_id, "init.step1.collect", "README.md"),
            )

            step1 = make_step_payload("claude-code", refresh_id, "refresh.step1.collect", "README.md")
            step1["meta"]["repo_scopes"] = ["repo-a", "repo-b"]
            step1["changeset"][0]["entity"]["provenance"]["evidence"] = [
                {"repo": "repo-a", "path": "README.md"},
                {"repo": "repo-b", "path": "README.md"},
            ]
            write_json(refresh_reports / "taskruns" / f"{refresh_id}-refresh-step1-collect-domain-main.json", step1)

            step3 = make_step_payload("claude-code", refresh_id, "refresh.step3.findings", "README.md")
            step3["meta"]["repo_scopes"] = ["repo-a", "repo-b"]
            step3["changeset"].append(
                {
                    "op": "upsert_edge",
                    "edge": {
                        "id": "edge.cross.repo-a.repo-b",
                        "type": "calls",
                        "from": "svc.repo-a.orders",
                        "to": "svc.repo-b.users",
                        "provenance": {
                            "kind": "observation",
                            "confidence": 0.8,
                            "evidence": [{"repo": "repo-b", "path": "README.md"}],
                        },
                    },
                }
            )
            write_json(refresh_reports / "taskruns" / f"{refresh_id}-refresh-step3-findings.json", step3)

            write_analysis_reports(
                refresh_reports,
                findings_text="\n".join(
                    [
                        "# Findings",
                        "",
                        "## Integration Contract Risk",
                        "- Severity: high",
                        "- Description: Cross-repo contract needs explicit owner and SLA.",
                        "",
                    ]
                ),
            )

            preflight = {
                "declared_repos_meta": {
                    "target_repos_file": str(root / "profiles/multi-path.yaml"),
                    "profile_id": "multi-path",
                    "profile_source_kind": "path",
                    "expected_repo_count": 2,
                    "declared_repos": [
                        {"name": "repo-a", "source": "path", "path": str(repo_a), "ref": ""},
                        {"name": "repo-b", "source": "path", "path": str(repo_b), "ref": ""},
                    ],
                }
            }
            result = report.evaluate_run("claude-code", 1, run_dir, preflight)
            self.assertTrue(result.hard_pass)
            self.assertNotIn("analysis:cross-repo-missing", result.issues)
            self.assertFalse(result.semantic_hard_fail)

    def test_marks_snapshot_missing_when_only_workspace_reports_exist(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            target_repo = root / "target-repo"
            write_text(target_repo / "README.md", "# demo\n")

            run_dir = root / "batch/claude-code/run1"
            run_dir.mkdir(parents=True, exist_ok=True)
            make_session_summary(run_dir / "session-summary.md")
            write_text(run_dir / "full-run.log", "")

            init_id = "run_init_3"
            refresh_id = "run_refresh_3"
            runtime_versions = "claude-code@1.2.3"
            write_text(
                run_dir / "run-results.tsv",
                "\n".join(
                    [
                        make_run_results_line("claude-code", "init", init_id, 10, 2, 1, 1, 2, 2, 0, runtime_versions),
                        make_run_results_line("claude-code", "refresh", refresh_id, 11, 2, 1, 1, 2, 2, 0, runtime_versions),
                    ]
                )
                + "\n",
            )

            reports_root = run_dir / "arch-workspace" / "reports"
            write_json(
                reports_root / "taskruns" / f"{init_id}-quality.json",
                make_quality_payload(init_id, "init", "claude-code", 10, 2, 1, 1, 2, 2, 0, "1.2.3", ["init.step1.collect"]),
            )
            write_json(
                reports_root / "taskruns" / f"{refresh_id}-quality.json",
                make_quality_payload(
                    refresh_id,
                    "refresh",
                    "claude-code",
                    11,
                    2,
                    1,
                    1,
                    2,
                    2,
                    0,
                    "1.2.3",
                    ["refresh.step1.collect", "refresh.step3.findings"],
                ),
            )
            write_json(
                reports_root / "taskruns" / f"{init_id}-init-step1-collect-domain-main.json",
                make_step_payload("claude-code", init_id, "init.step1.collect", "README.md"),
            )
            write_json(
                reports_root / "taskruns" / f"{refresh_id}-refresh-step1-collect-domain-main.json",
                make_step_payload("claude-code", refresh_id, "refresh.step1.collect", "README.md"),
            )
            write_json(
                reports_root / "taskruns" / f"{refresh_id}-refresh-step3-findings.json",
                make_step_payload("claude-code", refresh_id, "refresh.step3.findings", "README.md"),
            )
            write_analysis_reports(
                reports_root,
                findings_text="\n".join(
                    [
                        "# Findings",
                        "",
                        "## Missing Owner Mapping",
                        "- Severity: medium",
                        "- Description: Ownership is partially unknown.",
                        "",
                    ]
                ),
            )

            result = report.evaluate_run("claude-code", 1, run_dir, {"target_repo": str(target_repo)})
            self.assertFalse(result.hard_pass)
            self.assertEqual("workspace-fallback", result.artifact_source)
            self.assertIn("reliability:snapshot-missing", result.issues)

    def test_classifies_infra_incomplete_cycle_from_summary_counters(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            target_repo = root / "target-repo"
            write_text(target_repo / "README.md", "# demo\n")

            run_dir = root / "batch/qwen-code/run1"
            run_dir.mkdir(parents=True, exist_ok=True)
            write_text(
                run_dir / "session-summary.md",
                "\n".join(
                    [
                        "# ProvenArch Full Run Session Summary",
                        "",
                        "- result: passed",
                        "- quality_gates: passed",
                        "- expected_runs: 4",
                        "- completed_runs: 2",
                        "- expected_headless_runs: 2",
                        "- completed_headless_runs: 1",
                        "- running_runs_detected: 1",
                        "- failure_reason: infra_incomplete_cycle",
                        "",
                        "## API Simulation",
                        "- status: succeeded",
                        "",
                    ]
                ),
            )
            write_text(run_dir / "full-run.log", "")
            write_text(run_dir / "run-results.tsv", "")

            result = report.evaluate_run("qwen-code", 1, run_dir, {"target_repo": str(target_repo)})
            self.assertFalse(result.hard_pass)
            self.assertEqual("infra_incomplete_cycle", result.failure_class)
            self.assertTrue(result.infra_incomplete_cycle)
            self.assertIn("reliability:infra-incomplete-cycle", result.issues)

    def test_classifies_summary_missing(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            target_repo = root / "target-repo"
            write_text(target_repo / "README.md", "# demo\n")

            run_dir = root / "batch/claude-code/run1"
            run_dir.mkdir(parents=True, exist_ok=True)
            write_text(run_dir / "full-run.log", "")
            write_text(run_dir / "run-results.tsv", "")

            result = report.evaluate_run("claude-code", 1, run_dir, {"target_repo": str(target_repo)})
            self.assertFalse(result.hard_pass)
            self.assertEqual("summary_missing", result.failure_class)
            self.assertTrue(result.summary_missing)
            self.assertIn("reliability:summary-missing", result.issues)


if __name__ == "__main__":
    unittest.main()
