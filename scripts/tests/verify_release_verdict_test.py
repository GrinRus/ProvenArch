import json
import hashlib
import subprocess
import sys
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path


class VerifyReleaseVerdictTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.script = cls.repo_root / "scripts" / "verify-release-verdict.py"

    def init_repo(self, root: Path, tag: str = "v0.1.0") -> str:
        subprocess.run(["git", "init", "-q"], cwd=root, check=True)
        subprocess.run(["git", "config", "user.email", "test@example.invalid"], cwd=root, check=True)
        subprocess.run(["git", "config", "user.name", "Test User"], cwd=root, check=True)
        (root / "README.md").write_text("# test\n", encoding="utf-8")
        subprocess.run(["git", "add", "README.md"], cwd=root, check=True)
        subprocess.run(["git", "commit", "-qm", "fixture"], cwd=root, check=True)
        source_sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
        subprocess.run(["git", "tag", tag], cwd=root, check=True)
        return source_sha

    def now_text(self, delta: timedelta = timedelta()) -> str:
        value = datetime.now(timezone.utc) + delta
        return value.replace(microsecond=0).isoformat().replace("+00:00", "Z")

    def run_verifier(self, cwd: Path, *paths: Path, **kwargs: str) -> subprocess.CompletedProcess[str]:
        args = [str(path) for path in paths]
        for key, value in kwargs.items():
            args.extend([f"--{key.replace('_', '-')}", value])
        return subprocess.run(
            [sys.executable, str(self.script), *args], cwd=cwd, capture_output=True, text=True
        )

    def write_verdict(self, root: Path, payload: dict[str, object]) -> Path:
        matrix_id = str(payload["matrix_id"])
        reports = root if root.name == "reports" else root / "reports"
        reports.mkdir(exist_ok=True)
        path = reports / f"release_verdict_{matrix_id}.json"
        path.write_text(json.dumps(payload, indent=2), encoding="utf-8")
        return path

    def ready_payload(self, source_sha: str, matrix_id: str = "test-matrix") -> dict[str, object]:
        generated_at = self.now_text()
        profiles = ["single-path", "multi-git_url"]
        sweeps = ["baseline", "parallel-default"]
        records = []
        for profile in profiles:
            for sweep in sweeps:
                records.append(
                    {
                        "profile_id": profile,
                        "sweep_id": sweep,
                        "batch_id": f"{matrix_id}-{profile.replace('_', '-')}-{sweep.replace('_', '-')}",
                        "status": "passed",
                        "strict_status": "passed",
                        "blocking_reasons": [],
                        "shard_plan_invariant": "passed",
                        "execution": {
                            "strategy": "sequential" if sweep == "baseline" else "parallel",
                            "max_parallel_tasks": "1" if sweep == "baseline" else "4",
                            "failure_policy": "best_effort",
                            "shard_discovery_mode": "heuristics",
                        },
                        "backend": {
                            "hard_pass": 3,
                            "total_runs": 3,
                            "artifact_non_snapshot_runs": 0,
                            "runtime_contract_failed_failures": 0,
                            "runner_unavailable_failures": 0,
                            "runtime_timeout_failures": 0,
                            "infra_signal_terminated_failures": 0,
                            "infra_incomplete_cycle_failures": 0,
                            "quality_gates_failed_failures": 0,
                            "artifact_quality_failed_failures": 0,
                            "summary_missing_failures": 0,
                            "precheck_failed_failures": 0,
                            "runtime_flow_failed_runs": 0,
                            "runtime_flow_issue_hits": 0,
                            "partial_failure_count": 0,
                        },
                        "frontend": {
                            "frontend_qwen_status": "passed",
                            "frontend_claude_status": "passed",
                            "frontend_codex_status": "passed",
                        },
                        "public_authority": {
                            "effective_verdict_source": "orchestrator",
                            "promotion_audit_result": "pass",
                        },
                        "artifacts": {
                            "run_matrix_tsv": "matrix/run.tsv",
                            "run_matrix_md": "matrix/run.md",
                            "frontend_matrix_md": "matrix/frontend.md",
                            "execution_report_md": "matrix/execution.md",
                            "artifact_digests": {
                                "run_matrix_tsv": "",
                                "run_matrix_md": "",
                                "frontend_matrix_md": "",
                                "execution_report_md": "",
                            },
                        },
                    }
                )
        return {
            "matrix_id": matrix_id,
            "generated_at_utc": generated_at,
            "evidence_schema_version": 2,
            "source_sha": source_sha,
            "source_tree_clean": True,
            "generator": "scripts/full-run-batch-matrix.sh",
            "verdict": "PASS",
            "release_state": "RELEASE READY",
            "profile_sweep_runs": 4,
            "strict_pass_runs": 4,
            "strict_fail_runs": 0,
            "backend": {
                "hard_pass": 12,
                "total_runs": 12,
                "artifact_non_snapshot_runs": 0,
                "runtime_contract_failed_failures": 0,
                "runner_unavailable_failures": 0,
                "runtime_timeout_failures": 0,
                "infra_signal_terminated_failures": 0,
                "infra_incomplete_cycle_failures": 0,
                "quality_gates_failed_failures": 0,
                "artifact_quality_failed_failures": 0,
                "summary_missing_failures": 0,
                "precheck_failed_failures": 0,
                "runtime_flow_failed_runs": 0,
                "runtime_flow_issue_hits": 0,
                "partial_failure_count": 0,
            },
            "excellent_blockers": [],
            "excellent_blockers_by_step": [],
            "release_contract": {
                "mode": "release",
                "required_sweeps": sweeps,
                "observed_sweeps": sweeps,
                "selected_providers": ["qwen-code", "claude-code", "codex-code"],
                "selected_run_indexes": ["1"],
                "required_profiles": profiles,
                "observed_profiles": profiles,
                "expected_profile_sweep_runs": 4,
                "observed_profile_sweep_runs": 4,
                "shard_plan_invariant_status": "passed",
                "blocking_reasons": [],
                "contract_status": "passed",
            },
            "records": records,
            "evidence_artifacts": {},
        }

    def write_supporting_evidence(self, root: Path, payload: dict[str, object]) -> None:
        matrix_id = str(payload["matrix_id"])
        generated_at = str(payload["generated_at_utc"])
        source_sha = str(payload["source_sha"])
        (root / f"release_verdict_{matrix_id}.md").write_text(
            "\n".join(
                [
                    f"# Release Verdict: {matrix_id}",
                    "",
                    f"- generated_at_utc: {generated_at}",
                    "- verdict: PASS",
                    "- release_state: RELEASE READY",
                    "- profile_sweep_runs: 4",
                    "- strict_pass_runs: 4",
                    "- strict_fail_runs: 0",
                    "- release_contract_status: passed",
                    "",
                    "## Release Contract",
                    "- contract_status: passed",
                ]
            )
            + "\n",
            encoding="utf-8",
        )
        profiles = ["single-path", "multi-git_url"]
        sweeps = ["baseline", "parallel-default"]
        rows = ["profile_id\tsweep_id\tstrict_status"]
        md = ["# Profile Matrix", "", "| profile_id | sweep_id | status | strict |", "|---|---|---|---|"]
        for profile in profiles:
            for sweep in sweeps:
                rows.append(f"{profile}\t{sweep}\tpassed")
                md.append(f"| {profile} | {sweep} | succeeded | passed |")
        (root / f"profile_matrix_{matrix_id}.tsv").write_text("\n".join(rows) + "\n", encoding="utf-8")
        (root / f"profile_matrix_{matrix_id}.md").write_text("\n".join(md) + "\n", encoding="utf-8")
        for record in payload["records"]:  # type: ignore[index]
            profile = str(record["profile_id"])
            sweep = str(record["sweep_id"])
            artifacts = record["artifacts"]
            assert isinstance(artifacts, dict)
            digests = {}
            for key, suffix in (("run_matrix_tsv", ".tsv"), ("run_matrix_md", ".md"), ("frontend_matrix_md", ".md"), ("execution_report_md", ".md")):
                artifact_path = root / f"artifact_{matrix_id}_{profile}_{sweep}_{key}{suffix}"
                batch_id = str(record["batch_id"])
                if key == "run_matrix_tsv":
                    content = "provider\trun\thard_pass\truntime_contract_status\tartifact_quality_status\tverdict\tartifact_source\truntime_contract_failed\trunner_unavailable\truntime_timeout\tinfra_signal_terminated\tinfra_incomplete_cycle\tquality_gates_failed\tartifact_quality_failed\tsummary_missing\tprecheck_failed\truntime_flow_failed\tcancellation_like\tsemantic_hard_fail\teffective_verdict_source\tpromotion_audit_result\n" + "\n".join(
                        f"{provider}\t1\t1\tpassed\tpassed\tGood\tsnapshot\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\torchestrator\tpass" for provider in ("qwen-code", "claude-code", "codex-code")
                    ) + "\n"
                    zero_fields = [
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
                    ]
                    header = [
                        "provider", "run", "hard_pass", "runtime_contract_status", "artifact_quality_status",
                        "verdict", "artifact_source", *zero_fields, "failure_class", "issues", "effective_verdict_source", "promotion_audit_result",
                    ]
                    rows = [
                        [provider, "1", "1", "passed", "passed", "Good", "snapshot", *(["0"] * len(zero_fields)), "none", "-", "orchestrator", "pass"]
                        for provider in ("qwen-code", "claude-code", "codex-code")
                    ]
                    content = "\n".join("\t".join(row) for row in [header, *rows]) + "\n"
                elif key == "run_matrix_md":
                    content = "\n".join(
                        [
                            "# Run Matrix",
                            "",
                            "| provider | run | hard_pass | runtime_contract_status | artifact_quality_status | verdict | artifact_source | runtime_contract_failed | runner_unavailable | runtime_timeout | infra_signal_terminated | infra_incomplete_cycle | quality_gates_failed | artifact_quality_failed | summary_missing | precheck_failed | runtime_flow_failed | cancellation_like | semantic_hard_fail | off_topic_hits | artifact_quality_findings | provider_budget_exhausted | partial_failure_count | failure_class | issues | effective_verdict_source | promotion_audit_result |",
                            "|---|---:|---:|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---|",
                            *[
                                f"| {provider} | 1 | 1 | passed | passed | Good | snapshot | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | none | - | orchestrator | pass |"
                                for provider in ("qwen-code", "claude-code", "codex-code")
                            ],
                            "",
                        ]
                    )
                elif key == "frontend_matrix_md":
                    content = "\n".join(
                        [
                            "# Frontend Live E2E Matrix",
                            "",
                            "## Summary",
                            "",
                            "| provider | status | runs | reasons |",
                            "|---|---|---:|---|",
                            *[
                                f"| {provider} | passed | 1 | ok |"
                                for provider in ("qwen-code", "claude-code", "codex-code")
                            ],
                            "",
                            "## Run Details",
                            "",
                            "| provider | run | status |",
                            "|---|---:|---|",
                            *[
                                f"| {provider} | 1 | passed |"
                                for provider in ("qwen-code", "claude-code", "codex-code")
                            ],
                            "",
                        ]
                    )
                else:
                    content = "\n".join(
                        [
                            f"# Execution Report: {batch_id}",
                            "",
                            "## Context",
                            f"- provenarch_sha: {source_sha}",
                            "",
                            "## Backend Execution Verdict",
                            "- hard_pass_runs: 3/3",
                            "- artifact_quality_failed_runs: 0/3",
                            "- artifact_quality_findings: 0",
                            "- runtime_flow_failed_runs: 0/3",
                            "- primary_failure_classes: none",
                            "- semantic_hard_fail_runs: 0/3",
                            "- artifact_source_snapshot_runs: 3/3",
                            "- partial_failure_count: 0",
                            "- provider_budget_exhausted_runs: 0/3",
                            "",
                            "## Public Promotion Authority",
                            "- effective_verdict_sources: orchestrator",
                            "- promotion_audit_failed_runs: 0/3",
                            "- providers: qwen-code, claude-code, codex-code",
                            "",
                            "## Provider Matrix",
                            "",
                            "| provider | runs | pass_rate | off_topic_hits | artifact_quality_findings | semantic_hard_fail_runs | partial_failure_count | runtime_contract_failed_failures | runner_unavailable_failures | runtime_timeout_failures | infra_signal_terminated_failures | infra_incomplete_cycle_failures | quality_gates_failed_failures | artifact_quality_failed_failures | summary_missing_failures | precheck_failed_failures | runtime_flow_failed_failures | cancellation_like_failures | artifact_sources | frontend_live_pass_rate |",
                            "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---:|",
                            *[
                                f"| {provider} | 1 | 1.00 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | snapshot=1 | 1.00 |"
                                for provider in ("qwen-code", "claude-code", "codex-code")
                            ],
                            "",
                        ]
                    )
                content += "\n".join(
                    [
                        "",
                        f"# acp_record_profile_id: {profile}",
                        f"# acp_record_sweep_id: {sweep}",
                        f"# acp_record_batch_id: {batch_id}",
                        "",
                    ]
                )
                artifact_path.write_text(content, encoding="utf-8")
                artifacts[key] = artifact_path.name
                digests[key] = hashlib.sha256(artifact_path.read_bytes()).hexdigest()
            artifacts["artifact_digests"] = digests
        for label in ("ux", "artifact_quality"):
            (root / f"swe_{label}_assessment_{matrix_id}.md").write_text(
                "\n".join(
                    [
                        f"# {label}",
                        f"- matrix_id: {matrix_id}",
                        "- decision: accepted",
                        f"- source_sha: {source_sha}",
                        "- assessed_by: test-reviewer",
                        f"- assessed_at_utc: {generated_at}",
                        "",
                        "## Evidence Inspected",
                        f"- release verdict: release_verdict_{matrix_id}.json",
                        "- execution reports: profile matrix and per-record artifacts",
                    ]
                )
                + "\n",
                encoding="utf-8",
            )
        payload["evidence_artifacts"] = {
            "verdict_markdown": {
                "path": f"release_verdict_{matrix_id}.md",
                "sha256": hashlib.sha256((root / f"release_verdict_{matrix_id}.md").read_bytes()).hexdigest(),
            },
            "profile_matrix_markdown": {
                "path": f"profile_matrix_{matrix_id}.md",
                "sha256": hashlib.sha256((root / f"profile_matrix_{matrix_id}.md").read_bytes()).hexdigest(),
            },
            "profile_matrix_tsv": {
                "path": f"profile_matrix_{matrix_id}.tsv",
                "sha256": hashlib.sha256((root / f"profile_matrix_{matrix_id}.tsv").read_bytes()).hexdigest(),
            },
        }

    def write_ready_evidence(self, root: Path, source_sha: str, matrix_id: str) -> Path:
        payload = self.ready_payload(source_sha, matrix_id)
        reports = root if root.name == "reports" else root / "reports"
        reports.mkdir(exist_ok=True)
        self.write_supporting_evidence(reports, payload)
        path = self.write_verdict(root, payload)
        return path

    def test_accepts_complete_fresh_evidence_with_tag_binding(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            path = self.write_ready_evidence(root, source_sha, "test-matrix")
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("release evidence ready", result.stdout)

    def test_rejects_non_evidence_commit_after_qualification_source(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            path = self.write_ready_evidence(root, source_sha, "test-matrix")
            (root / "README.md").write_text("# changed product source\n", encoding="utf-8")
            subprocess.run(["git", "add", "README.md"], cwd=root, check=True)
            subprocess.run(["git", "commit", "-qm", "unrelated source change"], cwd=root, check=True)
            subprocess.run(["git", "tag", "-f", "v0.1.0"], cwd=root, check=True)
            release_sha = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
            result = self.run_verifier(root, path, source_sha=release_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("non-evidence changes after qualification source", result.stderr)

    def test_accepts_composite_release_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            paths = [self.write_ready_evidence(root, source_sha, matrix_id) for matrix_id in (
                "release-full-fast", "release-full-long", "release-full-ftgo-sentry"
            )]
            result = self.run_verifier(root, *paths, source_sha=source_sha, tag="v0.1.0")
        self.assertEqual(result.returncode, 0, msg=result.stderr)
        self.assertIn("composite release evidence ready: 3 constituent matrices", result.stdout)

    def test_accepts_composite_matrix_ids_configuration(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            reports = root / "reports"
            reports.mkdir()
            matrix_ids = ["release-full-fast", "release-full-long", "release-full-ftgo-sentry"]
            for matrix_id in matrix_ids:
                self.write_ready_evidence(reports, source_sha, matrix_id)
            result = self.run_verifier(root, source_sha=source_sha, tag="v0.1.0", matrix_ids=",".join(matrix_ids))
        self.assertEqual(result.returncode, 0, msg=result.stderr)

    def test_rejects_stale_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            payload["generated_at_utc"] = self.now_text(timedelta(hours=-169))
            self.write_supporting_evidence(root, payload)
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("older than 168 hours", result.stderr)

    def test_rejects_source_or_tag_mismatch(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            path = self.write_ready_evidence(root, source_sha, "test-matrix")
            result = self.run_verifier(root, path, source_sha="0" * 40, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("expected source SHA", result.stderr)

    def test_rejects_incomplete_records_and_fabricated_payload(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            payload["records"] = payload["records"][:1]  # type: ignore[index]
            self.write_supporting_evidence(root, payload)
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("records must contain exactly 4", result.stderr)

    def test_rejects_fabricated_artifact_reference(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            first_record = payload["records"][0]  # type: ignore[index]
            assert isinstance(first_record, dict)
            artifacts = first_record["artifacts"]
            assert isinstance(artifacts, dict)
            artifacts["run_matrix_tsv"] = "missing/run.tsv"
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("must reference a non-empty file", result.stderr)

    def test_rejects_hashed_but_semantically_invalid_artifact(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            first_record = payload["records"][0]  # type: ignore[index]
            assert isinstance(first_record, dict)
            artifacts = first_record["artifacts"]
            assert isinstance(artifacts, dict)
            artifact_path = reports / str(artifacts["run_matrix_tsv"])
            artifact_path.write_text("arbitrary content\n", encoding="utf-8")
            digests = artifacts["artifact_digests"]
            assert isinstance(digests, dict)
            digests["run_matrix_tsv"] = hashlib.sha256(artifact_path.read_bytes()).hexdigest()
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("is missing columns", result.stderr)

    def test_rejects_mismatched_sweep_execution_profile(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            parallel_record = next(
                record for record in payload["records"]  # type: ignore[index]
                if record["sweep_id"] == "parallel-default"
            )
            assert isinstance(parallel_record, dict)
            execution = parallel_record["execution"]
            assert isinstance(execution, dict)
            execution["strategy"] = "sequential"
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("must be 'parallel' for sweep 'parallel-default'", result.stderr)

    def test_accepts_needs_review_artifact_quality_without_hard_failure(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            first_record = payload["records"][0]  # type: ignore[index]
            assert isinstance(first_record, dict)
            artifacts = first_record["artifacts"]
            assert isinstance(artifacts, dict)
            tsv_path = reports / str(artifacts["run_matrix_tsv"])
            tsv_path.write_text(
                tsv_path.read_text(encoding="utf-8").replace(
                    "passed\tpassed\tGood", "passed\tneeds_review\tNeeds review", 1
                ),
                encoding="utf-8",
            )
            md_path = reports / str(artifacts["run_matrix_md"])
            md_path.write_text(
                md_path.read_text(encoding="utf-8").replace(
                    "| passed | passed | Good |", "| passed | needs_review | Needs review |", 1
                ),
                encoding="utf-8",
            )
            digests = artifacts["artifact_digests"]
            assert isinstance(digests, dict)
            digests["run_matrix_tsv"] = hashlib.sha256(tsv_path.read_bytes()).hexdigest()
            digests["run_matrix_md"] = hashlib.sha256(md_path.read_bytes()).hexdigest()
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertEqual(result.returncode, 0, msg=result.stderr)

    def test_rejects_runtime_flow_issue_hits_in_backend_claims(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            payload["backend"]["runtime_flow_issue_hits"] = 1  # type: ignore[index]
            first_record = payload["records"][0]  # type: ignore[index]
            assert isinstance(first_record, dict)
            first_record["backend"]["runtime_flow_issue_hits"] = 1  # type: ignore[index]
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("backend.runtime_flow_issue_hits must be 0", result.stderr)
        self.assertIn("records[0].backend.runtime_flow_issue_hits must be 0", result.stderr)

    def test_rejects_duplicate_frontend_run_details_section(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            first_record = payload["records"][0]  # type: ignore[index]
            assert isinstance(first_record, dict)
            artifacts = first_record["artifacts"]
            assert isinstance(artifacts, dict)
            artifact_path = reports / str(artifacts["frontend_matrix_md"])
            artifact_path.write_text(
                artifact_path.read_text(encoding="utf-8")
                + "\n## Run Details\n\n| provider | run | status |\n|---|---:|---|\n| qwen-code | 1 | failed |\n",
                encoding="utf-8",
            )
            digests = artifacts["artifact_digests"]
            assert isinstance(digests, dict)
            digests["frontend_matrix_md"] = hashlib.sha256(artifact_path.read_bytes()).hexdigest()
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("must contain exactly one Run Details section", result.stderr)

    def test_rejects_contradictory_provider_matrix(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            first_record = payload["records"][0]  # type: ignore[index]
            assert isinstance(first_record, dict)
            artifacts = first_record["artifacts"]
            assert isinstance(artifacts, dict)
            artifact_path = reports / str(artifacts["execution_report_md"])
            content = artifact_path.read_text(encoding="utf-8")
            content = content.replace(
                "| qwen-code | 1 | 1.00 | 0 | 0 |",
                "| qwen-code | 1 | 0.00 | 0 | 0 |",
                1,
            )
            artifact_path.write_text(content, encoding="utf-8")
            digests = artifacts["artifact_digests"]
            assert isinstance(digests, dict)
            digests["execution_report_md"] = hashlib.sha256(artifact_path.read_bytes()).hexdigest()
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Provider Matrix row", result.stderr)

    def test_rejects_nonzero_provider_matrix_strict_counters(self) -> None:
        for field in ("off_topic_hits", "cancellation_like_failures", "artifact_quality_findings"):
            with self.subTest(field=field), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                source_sha = self.init_repo(root)
                payload = self.ready_payload(source_sha)
                reports = root / "reports"
                reports.mkdir()
                self.write_supporting_evidence(reports, payload)
                first_record = payload["records"][0]  # type: ignore[index]
                assert isinstance(first_record, dict)
                artifacts = first_record["artifacts"]
                assert isinstance(artifacts, dict)
                artifact_path = reports / str(artifacts["execution_report_md"])
                lines = artifact_path.read_text(encoding="utf-8").splitlines()
                header_index = next(index for index, line in enumerate(lines) if line.startswith("| provider | runs | pass_rate |"))
                header = [cell.strip() for cell in lines[header_index].strip().strip("|").split("|")]
                row_index = header_index + 2
                cells = [cell.strip() for cell in lines[row_index].strip().strip("|").split("|")]
                cells[header.index(field)] = "1"
                lines[row_index] = "| " + " | ".join(cells) + " |"
                artifact_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
                digests = artifacts["artifact_digests"]
                assert isinstance(digests, dict)
                digests["execution_report_md"] = hashlib.sha256(artifact_path.read_bytes()).hexdigest()
                path = self.write_verdict(root, payload)
                result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(f"Provider Matrix row {row_index} {field} must be 0", result.stderr)

    def test_rejects_provider_budget_exhaustion_in_run_and_execution_report(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            first_record = payload["records"][0]  # type: ignore[index]
            assert isinstance(first_record, dict)
            artifacts = first_record["artifacts"]
            assert isinstance(artifacts, dict)
            md_path = reports / str(artifacts["run_matrix_md"])
            md_lines = md_path.read_text(encoding="utf-8").splitlines()
            md_header_index = next(index for index, line in enumerate(md_lines) if line.startswith("| provider | run |"))
            md_header = [cell.strip() for cell in md_lines[md_header_index].strip().strip("|").split("|")]
            md_row_index = md_header_index + 2
            md_cells = [cell.strip() for cell in md_lines[md_row_index].strip().strip("|").split("|")]
            md_cells[md_header.index("provider_budget_exhausted")] = "1"
            md_lines[md_row_index] = "| " + " | ".join(md_cells) + " |"
            md_path.write_text("\n".join(md_lines) + "\n", encoding="utf-8")
            execution_path = reports / str(artifacts["execution_report_md"])
            execution_path.write_text(
                execution_path.read_text(encoding="utf-8").replace(
                    "- provider_budget_exhausted_runs: 0/3", "- provider_budget_exhausted_runs: 1/3", 1
                ),
                encoding="utf-8",
            )
            digests = artifacts["artifact_digests"]
            assert isinstance(digests, dict)
            digests["run_matrix_md"] = hashlib.sha256(md_path.read_bytes()).hexdigest()
            digests["execution_report_md"] = hashlib.sha256(execution_path.read_bytes()).hexdigest()
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("provider_budget_exhausted must be 0", result.stderr)
        self.assertIn("provider_budget_exhausted_runs must be 0/3", result.stderr)

    def test_rejects_release_blocking_issues_in_run_artifacts(self) -> None:
        cases = (
            ("run_matrix_tsv", "analysis:evidence-scope"),
            ("run_matrix_tsv", "analysis:cross-repo-missing"),
            ("run_matrix_md", "runtime:execution-semantics"),
            ("run_matrix_tsv", "analysis:cross-doc"),
            ("run_matrix_tsv", "contract:runtime-name"),
            ("run_matrix_tsv", "execution:provider-budget-exhausted"),
            ("run_matrix_tsv", "reliability:semantic-hard-fail"),
            ("run_matrix_tsv", "execution:partial-failures"),
            ("run_matrix_tsv", "unknown:unclassified"),
        )
        for artifact_key, issue in cases:
            with self.subTest(artifact_key=artifact_key, issue=issue), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                source_sha = self.init_repo(root)
                payload = self.ready_payload(source_sha)
                reports = root / "reports"
                reports.mkdir()
                self.write_supporting_evidence(reports, payload)
                first_record = payload["records"][0]  # type: ignore[index]
                assert isinstance(first_record, dict)
                artifacts = first_record["artifacts"]
                assert isinstance(artifacts, dict)
                artifact_path = reports / str(artifacts[artifact_key])
                content = artifact_path.read_text(encoding="utf-8")
                if artifact_key == "run_matrix_tsv":
                    content = content.replace("\tnone\t-\torchestrator\tpass", f"\tnone\t{issue}\torchestrator\tpass", 1)
                else:
                    content = content.replace("| none | - | orchestrator | pass |", f"| none | {issue} | orchestrator | pass |", 1)
                artifact_path.write_text(content, encoding="utf-8")
                digests = artifacts["artifact_digests"]
                assert isinstance(digests, dict)
                digests[artifact_key] = hashlib.sha256(artifact_path.read_bytes()).hexdigest()
                path = self.write_verdict(root, payload)
                result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("contains release-blocking issues", result.stderr)

    def test_accepts_explicitly_nonblocking_analysis_issue(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            first_record = payload["records"][0]  # type: ignore[index]
            assert isinstance(first_record, dict)
            artifacts = first_record["artifacts"]
            assert isinstance(artifacts, dict)
            tsv_path = reports / str(artifacts["run_matrix_tsv"])
            tsv_path.write_text(
                tsv_path.read_text(encoding="utf-8").replace("\tnone\t-\torchestrator\tpass", "\tnone\tanalysis:findings\torchestrator\tpass", 1),
                encoding="utf-8",
            )
            md_path = reports / str(artifacts["run_matrix_md"])
            md_path.write_text(
                md_path.read_text(encoding="utf-8").replace("| none | - | orchestrator | pass |", "| none | analysis:findings | orchestrator | pass |", 1),
                encoding="utf-8",
            )
            digests = artifacts["artifact_digests"]
            assert isinstance(digests, dict)
            digests["run_matrix_tsv"] = hashlib.sha256(tsv_path.read_bytes()).hexdigest()
            digests["run_matrix_md"] = hashlib.sha256(md_path.read_bytes()).hexdigest()
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertEqual(result.returncode, 0, msg=result.stderr)

    def test_rejects_short_contradictory_markdown_rows(self) -> None:
        cases = (
            ("run_matrix_md", "| qwen-code | 1 | 0 |", None, "fewer cells than canonical header"),
            ("frontend_matrix_md", "| qwen-code | failed | 1 |", "## Run Details", "fewer cells than canonical header"),
            ("frontend_matrix_md", "| qwen-code | 1 | failed |", None, "detail row"),
            ("execution_report_md", "| qwen-code | 1 | 0.00 |", None, "fewer cells than canonical header"),
        )
        for artifact_key, short_row, marker, expected_error in cases:
            with self.subTest(artifact_key=artifact_key), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                source_sha = self.init_repo(root)
                payload = self.ready_payload(source_sha)
                reports = root / "reports"
                reports.mkdir()
                self.write_supporting_evidence(reports, payload)
                first_record = payload["records"][0]  # type: ignore[index]
                assert isinstance(first_record, dict)
                artifacts = first_record["artifacts"]
                assert isinstance(artifacts, dict)
                artifact_path = reports / str(artifacts[artifact_key])
                content = artifact_path.read_text(encoding="utf-8")
                if marker is not None:
                    content = content.replace(f"\n{marker}", f"\n{short_row}\n\n{marker}", 1)
                else:
                    content += f"\n{short_row}\n"
                artifact_path.write_text(content, encoding="utf-8")
                digests = artifacts["artifact_digests"]
                assert isinstance(digests, dict)
                digests[artifact_key] = hashlib.sha256(artifact_path.read_bytes()).hexdigest()
                path = self.write_verdict(root, payload)
                result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(expected_error, result.stderr)

    def test_rejects_replayed_record_artifacts(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            reports = root / "reports"
            reports.mkdir()
            self.write_supporting_evidence(reports, payload)
            first = payload["records"][0]  # type: ignore[index]
            second = payload["records"][1]  # type: ignore[index]
            assert isinstance(first, dict) and isinstance(second, dict)
            second["batch_id"] = first["batch_id"]
            second["artifacts"] = dict(first["artifacts"])
            path = self.write_verdict(root, payload)
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate batch_id", result.stderr)
        self.assertIn("duplicate artifact reference", result.stderr)

    def test_rejects_duplicate_matrix_id_in_composite_evidence(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            path = self.write_ready_evidence(root, source_sha, "release-full-fast")
            result = self.run_verifier(root, path, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("duplicate matrix_id in composite release evidence", result.stderr)

    def test_rejects_missing_constituent_from_matrix_ids_configuration(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.init_repo(root)
            (root / "reports").mkdir()
            result = self.run_verifier(root, source_sha="0" * 40, tag="v0.1.0", matrix_ids="release-full-missing")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release verdict file not found", result.stderr)

    def test_rejects_missing_or_mismatched_assessments(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            path = self.write_ready_evidence(root, source_sha, "test-matrix")
            (root / "reports" / "swe_ux_assessment_test-matrix.md").write_text(
                "- matrix_id: test-matrix\n- decision: accepted\n", encoding="utf-8"
            )
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("assessed_by must be present", result.stderr)

    def test_rejects_filename_mismatch_and_fail_verdict(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source_sha = self.init_repo(root)
            payload = self.ready_payload(source_sha)
            payload["verdict"] = "FAIL"
            payload["release_state"] = "RELEASE BLOCKED"
            (root / "reports").mkdir()
            path = root / "reports" / "release_verdict_other.json"
            path.write_text(json.dumps(payload), encoding="utf-8")
            result = self.run_verifier(root, path, source_sha=source_sha, tag="v0.1.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("matrix_id must match verdict filename", result.stderr)
        self.assertIn("verdict must be PASS", result.stderr)

    def test_rejects_invalid_json_and_non_release_shape(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.init_repo(root)
            (root / "reports").mkdir()
            invalid = root / "reports" / "release_verdict_invalid.json"
            invalid.write_text("{not-json", encoding="utf-8")
            result = self.run_verifier(root, invalid)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("invalid release verdict JSON", result.stderr)

            path = root / "reports" / "release_verdict_non-release.json"
            path.write_text(json.dumps({"result": "PASS", "mode": "non-release"}), encoding="utf-8")
            result = self.run_verifier(root, path)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("verdict must be PASS", result.stderr)
        self.assertIn("release_contract must be an object", result.stderr)

    def test_rejects_conflicting_and_duplicate_matrix_modes(self) -> None:
        result = self.run_verifier_args("--matrix-id", "release-fast", "--verdict-path", "x.json")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("set exactly one release evidence mode", result.stderr)

    def run_verifier_args(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run([sys.executable, str(self.script), *args], cwd=self.repo_root, capture_output=True, text=True)


if __name__ == "__main__":
    unittest.main()
