import importlib.util
import json
import os
import shlex
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
E2E_BATCH_REPORT_PATH = REPO_ROOT / "scripts" / "e2e_batch_report.py"
FULL_RUN_BATCH_SCRIPT = REPO_ROOT / "scripts" / "full-run-batch.sh"
BACKEND_CYCLE_SCRIPT = REPO_ROOT / "scripts" / "internal" / "live-e2e-backend-cycle.sh"


def load_e2e_batch_report_module():
    spec = importlib.util.spec_from_file_location("e2e_batch_report", E2E_BATCH_REPORT_PATH)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def write_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def extract_bash_function(script_text: str, name: str) -> str:
    marker = f"{name}() {{"
    start = script_text.index(marker)
    lines = script_text[start:].splitlines()
    selected: list[str] = []
    depth = 0
    for line in lines:
        selected.append(line)
        depth += line.count("{") - line.count("}")
        if selected and depth == 0:
            return "\n".join(selected)
    raise AssertionError(f"failed to extract bash function {name}")


class BatchFailureClassificationTest(unittest.TestCase):
    def setUp(self) -> None:
        self.module = load_e2e_batch_report_module()
        self.tempdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tempdir.name)
        self.run_dir = self.root / "run1"
        self._create_fixture_run_dir(self.run_dir)

    def tearDown(self) -> None:
        self.tempdir.cleanup()

    def _prepend_fake_host_npm(self, env: dict[str, str]) -> None:
        bin_dir = self.root / "fake-host-bin"
        bin_dir.mkdir(exist_ok=True)
        npm = bin_dir / "npm"
        write_text(
            npm,
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\\n' '10.9.8'; exit 0; fi\n"
            "printf '%s\\n' \"$0\"\n",
        )
        npm.chmod(0o755)
        env["PATH"] = f"{bin_dir}{os.pathsep}{env.get('PATH', '')}"

    def test_evidence_path_resolves_unique_extensionless_repo_file(self) -> None:
        workspace = self.root / "workspace"
        repo = self.root / "sample-repo"
        write_text(repo / "src/utils/db/primary-store.ts", "export const primaryStore = true\n")
        write_text(repo / "src/utils/db/cache-store.ts", "export const cacheStore = true\n")

        ok, reason = self.module.evidence_path_resolves("src/utils/db/primary-store", [repo], workspace)

        self.assertTrue(ok, reason)

    def test_evidence_path_rejects_ambiguous_extensionless_repo_file(self) -> None:
        workspace = self.root / "workspace"
        repo = self.root / "ambiguous"
        write_text(repo / "src/config.ts", "export const config = true\n")
        write_text(repo / "src/config.go", "package config\n")

        ok, reason = self.module.evidence_path_resolves("src/config", [repo], workspace)

        self.assertFalse(ok)
        self.assertEqual("ambiguous extensionless relative path", reason)

    def _create_fixture_run_dir(self, run_dir: Path) -> None:
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: infra_incomplete_cycle",
                    "- expected_runs: 10",
                    "- completed_runs: 2",
                    "- expected_headless_runs: 10",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )

        write_text(run_dir / "full-run.log", "batch ended with incomplete cycle\n")
        write_text(run_dir / "batch-driver.log", "driver completed with process_exit=1\n")
        write_text(
            run_dir / "run-results.tsv",
            "\n".join(
                [
                    "\t".join(
                        [
                            "1",
                            "headless",
                            "qwen-code",
                            "init",
                            "init-run",
                            "succeeded",
                            "8",
                            "1",
                            "0",
                            "1",
                            "2",
                            "1",
                            "0",
                            "0",
                            "qwen-code@stub",
                            "reports/taskruns/init-run-quality.json",
                            "reports",
                        ]
                    ),
                    "\t".join(
                        [
                            "1",
                            "headless",
                            "qwen-code",
                            "refresh",
                            "refresh-run",
                            "succeeded",
                            "9",
                            "1",
                            "1",
                            "1",
                            "2",
                            "1",
                            "0",
                            "0",
                            "qwen-code@stub",
                            "reports/taskruns/refresh-run-quality.json",
                            "reports",
                        ]
                    ),
                ]
            )
            + "\n",
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","error_code":"runtime_contract_failed","message":"provider did not produce required artifacts"}\n',
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt",
            "artifact validation failed\nruntime_contract_failed\n",
        )
        self._write_snapshot(run_dir, "init-run", "init")
        self._write_snapshot(run_dir, "refresh-run", "refresh")

    def test_full_run_batch_provider_preflight_failure_materializes_reports(self) -> None:
        repos_file = self.root / "repos.yaml"
        write_text(
            repos_file,
            "\n".join(
                [
                    "repos:",
                    "  - name: provider-preflight",
                    "    git_url: https://example.invalid/provider-preflight.git",
                    "    ref: 1111111111111111111111111111111111111111",
                    "",
                ]
            ),
        )
        qwen_stub = self.root / "qwen-preflight-stub.sh"
        write_text(
            qwen_stub,
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "if [ \"${1:-}\" = \"-p\" ]; then printf '%s\n' 'ACP_READY'; exit 0; fi\n"
            "if [ \"${1:-}\" = \"--chat-recording\" ]; then printf '%s\n' 'ACP_READY'; exit 0; fi\n"
            "printf 'unexpected args: %s\\n' \"$*\" >&2\n"
            "exit 2\n",
        )
        qwen_stub.chmod(0o755)

        batch_id = "provider-preflight-failure"
        e2e_root = self.root / "e2e"
        reports_root = e2e_root / "reports"
        env = {
            key: value
            for key, value in os.environ.items()
            if key
            in {
                "PATH",
                "HOME",
                "TMPDIR",
                "TMP",
                "TEMP",
                "USER",
                "LOGNAME",
                "LANG",
                "LC_ALL",
                "LC_CTYPE",
                "SHELL",
                "PYENV_ROOT",
            }
        }
        env.update(
            {
                "BATCH_ID": batch_id,
                "E2E_TMP_ROOT": str(e2e_root),
                "REPORTS_ROOT": str(reports_root),
                "TARGET_REPOS_FILE": str(repos_file),
                "PROFILE_ID": "single-git_url",
                "PROFILE_SOURCE_KIND": "git_url",
                "PROFILE_EXPECTED_REPO_COUNT": "1",
                "RUN_COUNT": "1",
                "BATCH_PROVIDER_FILTER": "qwen-code",
                "BATCH_RUN_SELECTION": "1",
                "BATCH_FRONTEND_MODE": "never",
                "ACP_QWEN_CMD_BIN": str(qwen_stub),
                "ACP_CLAUDE_CMD_BIN": "true",
                "ACP_CODEX_CMD_BIN": "true",
                "ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC": "5",
                "BATCH_OWNER_HEARTBEAT_SEC": "1",
            }
        )
        self._prepend_fake_host_npm(env)

        completed = subprocess.run(
            [str(FULL_RUN_BATCH_SCRIPT)],
            cwd=REPO_ROOT,
            env=env,
            text=True,
            capture_output=True,
            timeout=90,
            check=False,
        )

        self.assertNotEqual(0, completed.returncode, msg=completed.stdout + completed.stderr)
        self.assertIn("operational_host_preflight_failed", completed.stderr + completed.stdout)
        for path in [
            reports_root / f"run_matrix_{batch_id}.md",
            reports_root / f"run_matrix_{batch_id}.tsv",
            reports_root / f"execution_report_{batch_id}.md",
            reports_root / f"frontend_e2e_matrix_{batch_id}.md",
            e2e_root / "runs" / batch_id / "report-paths.txt",
        ]:
            self.assertTrue(path.exists(), f"missing expected preflight failure artifact: {path}")

        run_matrix = (reports_root / f"run_matrix_{batch_id}.md").read_text(encoding="utf-8")
        self.assertIn("operational_host_preflight_failed", run_matrix)
        self.assertNotIn("| qwen-code | 1 | 0 | 0 | 0 | 0 | 0 | Poor | snapshot", run_matrix)
        self.assertFalse((reports_root / f"blackbox_e2e_steps_{batch_id}.jsonl").exists())
        self.assertFalse((reports_root / f"blackbox_e2e_steps_{batch_id}.md").exists())

    def test_full_run_batch_node_toolchain_precheck_failure_records_evidence(self) -> None:
        repos_file = self.root / "repos.yaml"
        write_text(
            repos_file,
            "\n".join(
                [
                    "repos:",
                    "  - name: node-precheck",
                    "    git_url: https://example.invalid/node-precheck.git",
                    "    ref: 1111111111111111111111111111111111111111",
                    "",
                ]
            ),
        )
        qwen_stub = self.root / "qwen-ready-stub.sh"
        write_text(
            qwen_stub,
            "#!/bin/sh\n"
            "if [ \"${1:-}\" = \"--version\" ]; then printf '%s\n' 'qwen 1.0'; exit 0; fi\n"
            "if [ \"${1:-}\" = \"-p\" ]; then printf '%s\n' 'ACP_READY'; exit 0; fi\n"
            "if [ \"${1:-}\" = \"--chat-recording\" ]; then\n"
            "  mkdir -p \"$(dirname \"$ACP_PREFLIGHT_SMOKE_SENTINEL\")\"\n"
            "  printf '%s\\n' \"$ACP_PREFLIGHT_SMOKE_TEXT\" > \"$ACP_PREFLIGHT_SMOKE_SENTINEL\"\n"
            "  printf '%s\n' 'Done.'\n"
            "  exit 0\n"
            "fi\n"
            "printf 'unexpected args: %s\\n' \"$*\" >&2\n"
            "exit 2\n",
        )
        qwen_stub.chmod(0o755)

        wrong_node_dir = self.root / "wrong-node"
        wrong_node_dir.mkdir()
        wrong_node = wrong_node_dir / "node"
        write_text(
            wrong_node,
            "#!/usr/bin/env bash\n"
            "set -Eeuo pipefail\n"
            "if [[ \"${1:-}\" == '-p' ]]; then\n"
            "  case \"${2:-}\" in\n"
            "    *process.versions.node*) printf '%s\\n' '22.22.3' ;;\n"
            "    *process.arch*) printf '%s\\n' 'arm64' ;;\n"
            "    *) printf '\\n' ;;\n"
            "  esac\n"
            "  exit 0\n"
            "fi\n"
            "if [[ \"${1:-}\" == '--version' ]]; then printf '%s\\n' 'v22.22.3'; exit 0; fi\n"
            "printf '%s\\n' \"$0\"\n",
        )
        wrong_npm = wrong_node_dir / "npm"
        write_text(
            wrong_npm,
            "#!/usr/bin/env bash\n"
            "set -Eeuo pipefail\n"
            "if [[ \"${1:-}\" == '--version' ]]; then printf '%s\\n' '10.9.8'; exit 0; fi\n"
            "printf '%s\\n' \"$0\"\n",
        )
        wrong_node.chmod(0o755)
        wrong_npm.chmod(0o755)

        batch_id = "node-toolchain-precheck-failure"
        e2e_root = self.root / "e2e-node"
        reports_root = e2e_root / "reports"
        env = {
            key: value
            for key, value in os.environ.items()
            if key
            in {
                "PATH",
                "HOME",
                "TMPDIR",
                "TMP",
                "TEMP",
                "USER",
                "LOGNAME",
                "LANG",
                "LC_ALL",
                "LC_CTYPE",
                "SHELL",
                "PYENV_ROOT",
            }
        }
        env.update(
            {
                "BATCH_ID": batch_id,
                "E2E_TMP_ROOT": str(e2e_root),
                "REPORTS_ROOT": str(reports_root),
                "TARGET_REPOS_FILE": str(repos_file),
                "PROFILE_ID": "single-git_url",
                "PROFILE_SOURCE_KIND": "git_url",
                "PROFILE_EXPECTED_REPO_COUNT": "1",
                "RUN_COUNT": "1",
                "BATCH_PROVIDER_FILTER": "qwen-code",
                "BATCH_RUN_SELECTION": "1",
                "BATCH_FRONTEND_MODE": "never",
                "ACP_QWEN_CMD_BIN": str(qwen_stub),
                "ACP_CLAUDE_CMD_BIN": "true",
                "ACP_CODEX_CMD_BIN": "true",
                "ACP_NODE_TOOL_CANDIDATES": str(wrong_node_dir),
                "ACP_NODE_TOOL_CANDIDATES_ONLY": "1",
                "ACP_PREFLIGHT_HEADLESS_PROBE_TIMEOUT_SEC": "5",
                "ACP_PREFLIGHT_ARTIFACT_SMOKE_TIMEOUT_SEC": "5",
                "BATCH_OWNER_HEARTBEAT_SEC": "1",
            }
        )
        self._prepend_fake_host_npm(env)

        completed = subprocess.run(
            [str(FULL_RUN_BATCH_SCRIPT)],
            cwd=REPO_ROOT,
            env=env,
            text=True,
            capture_output=True,
            timeout=90,
            check=False,
        )

        self.assertNotEqual(0, completed.returncode, msg=completed.stdout + completed.stderr)
        self.assertIn("precheck_failed", completed.stderr + completed.stdout)
        node_log = e2e_root / "runs" / batch_id / "precheck-node-toolchain.log"
        self.assertTrue(node_log.exists())
        node_log_text = node_log.read_text(encoding="utf-8")
        self.assertIn("node_status=failed", node_log_text)
        self.assertIn("Node.js 22.21.1 is required", node_log_text)
        self.assertIn("22.22.3", node_log_text)

        run_matrix = (reports_root / f"run_matrix_{batch_id}.md").read_text(encoding="utf-8")
        self.assertIn("precheck_failed", run_matrix)
        self.assertFalse((reports_root / f"blackbox_e2e_steps_{batch_id}.jsonl").exists())
        self.assertFalse((reports_root / f"blackbox_e2e_steps_{batch_id}.md").exists())

    def _create_passed_run_dir_with_raw_runner_noise(self, run_dir: Path) -> None:
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: passed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: none",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=completed",
                    "process_exit=0",
                    "termination_signal=none",
                    "failure_reason=none",
                    "summary_written=yes",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "full-run.log",
            'recovered earlier event: {"error_code":"runner_unavailable","message":"provider unavailable during prior repair"}\n'
            "GET https://chatgpt.com/backend-api/plugins/featured -> 429 Too Many Requests\n"
            "Cloudflare\n",
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","kind":"event","error_code":"runner_unavailable","message":"recovered provider diagnostic"}\n',
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt",
            "429 Too Many Requests\n",
        )

    def _write_snapshot(self, run_dir: Path, run_id: str, pipeline: str) -> None:
        reports_root = run_dir / "snapshots" / run_id / "reports"
        write_text(
            reports_root / "as-is/overview.md",
            "\n".join(
                [
                    "# As-Is Overview",
                    "",
                    "- Services: 1",
                    "- Dependencies (edges): 1",
                    "- External systems: 0",
                    "- Datastores: 0",
                    "",
                ]
            ),
        )
        write_text(
            reports_root / "findings/findings.md",
            "\n".join(
                [
                    "# Findings",
                    "",
                    "## Missing owner mapping",
                    "",
                    "- ID: `finding.owner.missing`",
                    "- Severity: `medium`",
                    "- Description: owner_team_id is unknown",
                    "",
                ]
            ),
        )
        write_text(
            reports_root / "coverage/summary.md",
            "\n".join(
                [
                    "# Coverage Summary",
                    "",
                    "## Observed",
                    "",
                    "- services",
                    "- entrypoints",
                    "",
                    "## Missing",
                    "",
                    "- owner mappings",
                    "",
                    "## Notes",
                    "",
                    "- fake deterministic snapshot",
                    "",
                ]
            ),
        )
        write_text(
            reports_root / "coverage/open-questions.md",
            "\n".join(
                [
                    "# Open Questions",
                    "",
                    "- `q.owner.service` Who owns the materialized service?",
                    "",
                ]
            ),
        )
        quality_payload = {
            "runtime_versions": ["qwen-code@stub"],
            "totals": {
                "signal_score": 8 if pipeline == "init" else 9,
                "semantic_entities": 1,
                "semantic_edges": 0 if pipeline == "init" else 1,
                "findings_count": 1,
                "questions_count": 2,
                "coverage_observed": 1,
                "coverage_missing": 0,
                "warnings_count": 0,
            },
            "steps": [
                {
                    "step_id": f"{pipeline}.step1.collect" if pipeline == "init" else f"{pipeline}.step3.findings",
                    "runtime_name": "qwen-code",
                }
            ],
        }
        write_json(reports_root / f"taskruns/{run_id}-quality.json", quality_payload)
        taskrun_payload = {
            "version": 1,
            "task_id": f"task-{run_id}-{pipeline}",
            "run_id": run_id,
            "step_id": f"{pipeline}.step1.collect" if pipeline == "init" else f"{pipeline}.step3.findings",
            "provider": "qwen-code",
            "started_at": "2026-04-20T00:00:00Z",
            "finished_at": "2026-04-20T00:00:01Z",
            "status": "failed" if pipeline == "init" else "succeeded",
            "repo_scopes": ["demo-repo"],
            "path_scopes": ["src"],
        }
        if pipeline == "init":
            taskrun_payload["shard_id"] = "domain-a"
        write_json(
            reports_root / f"taskruns/{run_id}/runtime/{pipeline}-step{'1-collect' if pipeline == 'init' else '3-findings'}/runtime-execution.json",
            taskrun_payload,
        )
        semantic_root = reports_root / "taskruns" / run_id
        if pipeline == "refresh":
            write_json(
                semantic_root / "staging" / "shards" / "domain-a" / "shard-pack-manifest.json",
                {
                    "version": 1,
                    "run_id": run_id,
                    "step_id": "refresh.step1.collect",
                    "shard_id": "domain-a",
                    "domain_id": "domain-a",
                    "agent_role": "shard-analyst",
                    "artifact_root": "reports/taskruns/refresh-run/staging/shards/domain-a",
                    "repo_scopes": ["demo-repo"],
                    "path_scopes": ["src"],
                    "documents": [
                        {
                            "id": "doc.arch",
                            "title": "Architecture",
                            "path": "architecture.md",
                            "kind": "report",
                        }
                    ],
                    "semantic": {
                        "coverage": {
                            "observed": ["services"],
                            "missing": [],
                            "notes": ["fixture"],
                        },
                        "questions": [{"id": "q.owner.service", "text": "Who owns the materialized service?"}],
                        "entities": [],
                        "edges": [],
                        "findings": [],
                    },
                },
            )
            write_json(
                semantic_root / "validator" / "validator-verdict.json",
                {
                    "version": 1,
                    "run_id": run_id,
                    "step_id": "refresh.step3.findings",
                    "generated_at": "2026-04-20T00:00:01Z",
                    "verdict": "PASS",
                    "summary": "fixture verdict",
                    "checked_paths": ["reports/as-is/overview.md"],
                    "fixed_paths": [],
                    "findings": [
                        {
                            "id": "finding.owner.missing",
                            "title": "Missing owner mapping",
                            "severity": "medium",
                            "description": "owner_team_id is unknown",
                        }
                    ],
                    "questions": [{"id": "q.owner.service", "text": "Who owns the materialized service?"}],
                },
            )

    def _create_incomplete_fixture_run_dir(self, run_dir: Path) -> None:
        self._create_fixture_run_dir(run_dir)
        incomplete_banner = "\n".join(
            [
                "> Analysis incomplete.",
                "> Collect status: unusable (planned=2 succeeded=0 failed=2)",
                "> Findings status: skipped (planned=0 succeeded=0 failed=0)",
                "> Reasons: collect_all_shards_failed, findings_skipped_due_to_unusable_collect",
                "",
            ]
        )
        for run_id in ("init-run", "refresh-run"):
            reports_root = run_dir / "snapshots" / run_id / "reports"
            write_text(
                reports_root / "as-is/overview.md",
                "# As-Is Overview\n\n"
                + incomplete_banner
                + "- Services: 0\n- Dependencies (edges): 0\n- External systems: 0\n- Datastores: 0\n",
            )
            write_text(
                reports_root / "findings/findings.md",
                "# Findings\n\n" + incomplete_banner + "Findings unavailable because analysis did not complete.\n",
            )
            write_text(
                reports_root / "coverage/summary.md",
                "# Coverage Summary\n\n"
                + incomplete_banner
                + "## Observed\n\nUnavailable due to incomplete analysis.\n\n"
                + "## Missing\n\nUnknown due to incomplete analysis.\n\n"
                + "## Notes\n\nAnalysis incomplete. See banner above.\n",
            )
            write_text(
                reports_root / "coverage/open-questions.md",
                "# Open Questions\n\n" + incomplete_banner + "Open questions unavailable due to incomplete analysis.\n",
            )
            quality_path = reports_root / f"taskruns/{run_id}-quality.json"
            quality_payload = json.loads(quality_path.read_text(encoding="utf-8"))
            quality_payload["evidence_state"] = {
                "collect": {
                    "status": "unusable",
                    "planned_shards": 2,
                    "succeeded_shards": 0,
                    "failed_shards": 2,
                },
                "findings": {
                    "status": "skipped",
                    "planned_shards": 0,
                    "succeeded_shards": 0,
                    "failed_shards": 0,
                },
                "report_mode": "incomplete",
                "reasons": ["collect_all_shards_failed", "findings_skipped_due_to_unusable_collect"],
            }
            write_json(quality_path, quality_payload)

    def _create_partial_incomplete_fixture_run_dir(self, run_dir: Path) -> None:
        self._create_fixture_run_dir(run_dir)
        partial_banner = "\n".join(
            [
                "> Partial analysis. Some shards failed; downstream content may be incomplete.",
                "> Collect status: partial (planned=2 succeeded=1 failed=1)",
                "> Findings status: partial (planned=2 succeeded=1 failed=1)",
                "> Reasons: collect_partial_shard_failures, findings_partial_shard_failures",
                "",
            ]
        )
        for run_id in ("init-run", "refresh-run"):
            reports_root = run_dir / "snapshots" / run_id / "reports"
            write_text(
                reports_root / "as-is/overview.md",
                "# As-Is Overview\n\n"
                + partial_banner
                + "- Services: 1\n- Dependencies (edges): 1\n- External systems: 0\n- Datastores: 0\n",
            )
            write_text(
                reports_root / "findings/findings.md",
                "# Findings\n\n"
                + partial_banner
                + "## Missing owner mapping\n\n- ID: `finding.owner.missing`\n- Severity: `medium`\n- Description: owner_team_id is unknown\n",
            )
            write_text(
                reports_root / "coverage/summary.md",
                "# Coverage Summary\n\n"
                + partial_banner
                + "## Observed\n\n- services\n- entrypoints\n\n"
                + "## Missing\n\n- owner mappings\n\n"
                + "## Notes\n\n- partial shard coverage only\n",
            )
            write_text(
                reports_root / "coverage/open-questions.md",
                "# Open Questions\n\n"
                + partial_banner
                + "- `q.owner.service` Who owns the materialized service?\n",
            )
            quality_path = reports_root / f"taskruns/{run_id}-quality.json"
            quality_payload = json.loads(quality_path.read_text(encoding="utf-8"))
            quality_payload["evidence_state"] = {
                "collect": {
                    "status": "partial",
                    "planned_shards": 2,
                    "succeeded_shards": 1,
                    "failed_shards": 1,
                },
                "findings": {
                    "status": "partial",
                    "planned_shards": 2,
                    "succeeded_shards": 1,
                    "failed_shards": 1,
                },
                "report_mode": "incomplete",
                "reasons": ["collect_partial_shard_failures", "findings_partial_shard_failures"],
            }
            write_json(quality_path, quality_payload)

    def _create_artifact_quality_fixture_run_dir(self, run_dir: Path) -> None:
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: passed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: none",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(run_dir / "full-run.log", "full run completed\n")
        write_text(run_dir / "batch-driver.log", "driver completed with process_exit=0\n")
        write_text(
            run_dir / "run-results.tsv",
            "\n".join(
                [
                    "\t".join(
                        [
                            "1",
                            "headless",
                            "qwen-code",
                            "init",
                            "init-run",
                            "succeeded",
                            "10",
                            "1",
                            "0",
                            "1",
                            "2",
                            "1",
                            "0",
                            "0",
                            "qwen-code@stub",
                            "reports/taskruns/init-run-quality.json",
                            "reports",
                        ]
                    ),
                    "\t".join(
                        [
                            "1",
                            "headless",
                            "qwen-code",
                            "refresh",
                            "refresh-run",
                            "succeeded",
                            "11",
                            "1",
                            "1",
                            "1",
                            "2",
                            "1",
                            "0",
                            "0",
                            "qwen-code@stub",
                            "reports/taskruns/refresh-run-quality.json",
                            "reports",
                        ]
                    ),
                ]
            )
            + "\n",
        )
        self._write_snapshot(run_dir, "init-run", "init")
        self._write_snapshot(run_dir, "refresh-run", "refresh")
        refresh_quality_path = run_dir / "snapshots" / "refresh-run" / "reports" / "taskruns" / "refresh-run-quality.json"
        refresh_quality = json.loads(refresh_quality_path.read_text(encoding="utf-8"))
        refresh_quality["run_warnings"] = [
            "artifact_quality: refresh staged final set has 6 canonical documents but only 1 generic runtime-summary citation (cite.runtime-summary)"
        ]
        write_json(refresh_quality_path, refresh_quality)

    def test_python_report_prefers_runtime_contract_failed_over_incomplete_cycle_classifier(self) -> None:
        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=self.run_dir,
            preflight={},
            classification_row={
                "failure_class": "infra_incomplete_cycle",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )
        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertTrue(result.runtime_contract_failed)
        self.assertTrue(result.infra_incomplete_cycle)

    def test_shell_no_longer_exposes_frontend_cancel_workspace_helpers(self) -> None:
        script = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        self.assertNotIn("sanitize_frontend_cancel_workspace() {", script)
        self.assertNotIn("prepare_frontend_cancel_workspace() {", script)
        self.assertNotIn("run_frontend_cancel_e2e() {", script)

    def test_frontend_live_passes_codex_command_env(self) -> None:
        script = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        self.assertIn('"ACP_CODEX_CMD=$ACP_CODEX_CMD_BIN"', script)

    def test_python_report_aggregates_runtime_repair_stall_counters(self) -> None:
        quality_path = self.run_dir / "snapshots" / "refresh-run" / "reports" / "taskruns" / "refresh-run-quality.json"
        payload = json.loads(quality_path.read_text(encoding="utf-8"))
        payload.setdefault("totals", {}).update(
            {
                "repair_attempts": 2,
                "repair_exhausted": 1,
                "fresh_retries": 1,
                "focused_repairs": 1,
                "stall_count": 2,
                "pre_artifact_stalls": 1,
                "post_artifact_stalls": 1,
                "zero_output_pre_artifact_stalls": 1,
                "partial_failure_count": 1,
            }
        )
        payload["quality_signals"] = [
            {"code": "runtime_quality.stall_pressure", "severity": "warning", "message": "runtime_quality: provider stall pressure was observed"}
        ]
        write_json(quality_path, payload)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=self.run_dir,
            preflight={},
        )

        self.assertEqual(2, result.repair_attempts)
        self.assertEqual(1, result.repair_exhausted)
        self.assertEqual(1, result.fresh_retries)
        self.assertEqual(1, result.focused_repairs)
        self.assertEqual(2, result.stall_count)
        self.assertEqual(1, result.pre_artifact_stalls)
        self.assertEqual(1, result.post_artifact_stalls)
        self.assertEqual(1, result.zero_output_pre_artifact_stalls)
        self.assertEqual(1, result.partial_failure_count)
        self.assertEqual(1, result.quality_alerts)
        self.assertIn("execution:repair-heavy", result.issues)
        self.assertIn("execution:stall-pressure", result.issues)
        self.assertIn("execution:partial-failures", result.issues)
        self.assertTrue(result.runtime_flow_failed)
        self.assertEqual("runtime_flow_failed", result.failure_class)

    def test_python_report_uses_workspace_quality_for_non_snapshot_failed_runs(self) -> None:
        run_dir = self.root / "run-workspace-quality-fallback"
        self._create_fixture_run_dir(run_dir)
        workspace_reports = run_dir / "arch-workspace" / "reports"
        shutil.copytree(run_dir / "snapshots" / "refresh-run" / "reports", workspace_reports, dirs_exist_ok=True)
        shutil.rmtree(run_dir / "snapshots" / "refresh-run")

        quality_path = workspace_reports / "taskruns" / "refresh-run-quality.json"
        payload = json.loads(quality_path.read_text(encoding="utf-8"))
        payload.setdefault("totals", {}).update(
            {
                "repair_attempts": 2,
                "repair_exhausted": 1,
                "fresh_retries": 0,
                "focused_repairs": 2,
                "stall_count": 1,
                "pre_artifact_stalls": 0,
                "post_artifact_stalls": 1,
                "zero_output_pre_artifact_stalls": 0,
                "partial_failure_count": 0,
            }
        )
        write_json(quality_path, payload)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
        )

        self.assertEqual("workspace", result.artifact_source)
        self.assertFalse(result.hard_pass)
        self.assertEqual(2, result.repair_attempts)
        self.assertEqual(1, result.repair_exhausted)
        self.assertEqual(2, result.focused_repairs)
        self.assertEqual(1, result.post_artifact_stalls)
        self.assertIn("execution:repair-exhausted", result.issues)
        self.assertTrue(any("using non-snapshot reports_root" in detail for detail in result.issue_details))

    def test_python_report_aggregates_failed_raw_stall_metadata(self) -> None:
        write_json(
            self.run_dir / "arch-workspace/reports/taskruns/raw/qwen-step2-meta.json",
            {
                "diagnostics": {
                    "artifact_observed": False,
                    "authored_file_count": 0,
                    "provider_lifecycle": {
                        "exit_reason": "stall",
                        "error": "runtime_stalled_before_artifacts",
                        "stdout_bytes": 0,
                        "stderr_bytes": 0,
                    },
                }
            },
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=self.run_dir,
            preflight={},
        )

        self.assertEqual(1, result.stall_count)
        self.assertEqual(1, result.pre_artifact_stalls)
        self.assertEqual(0, result.post_artifact_stalls)
        self.assertEqual(1, result.zero_output_pre_artifact_stalls)
        self.assertIn("execution:stall-pressure", result.issues)
        self.assertTrue(any("execution/runtime-stalls-raw" in detail for detail in result.issue_details))

    def test_python_report_prefers_runtime_flow_failed_when_validator_verdict_failed(self) -> None:
        run_dir = self.root / "run-validator-verdict-fail-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=pipeline_command_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )
        write_text(
            run_dir / "full-run.log",
            "runtime_contract_failed in step2 diagnostics\nvalidator verdict is FAIL\n",
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_flow_failed", result.failure_class)
        self.assertTrue(result.runtime_flow_failed)

    def test_python_report_keeps_runtime_contract_failed_when_already_classified_terminal(self) -> None:
        run_dir = self.root / "run-validator-verdict-fail-python-classified-contract"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=pipeline_command_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )
        write_text(
            run_dir / "full-run.log",
            "runtime_contract_failed in step2 diagnostics\nvalidator verdict is FAIL\n",
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_contract_failed",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertFalse(result.runtime_flow_failed)

    def test_python_report_prefers_runtime_contract_failed_over_runner_unavailable_classifier(self) -> None:
        run_dir = self.root / "run-contract-vs-runner-unavailable-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: runtime_contract_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "full-run.log",
            "runtime_contract_failed while validating draft manifest\nprovider returned status=429 model at capacity\n",
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runner_unavailable",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertTrue(result.runtime_contract_failed)
        self.assertTrue(result.runner_unavailable)

    def test_python_report_classifies_missing_collect_doc_reference_as_runtime_contract_failed(self) -> None:
        run_dir = self.root / "run-missing-collect-doc-ref-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=pipeline_command_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )
        write_text(
            run_dir / "full-run.log",
            'run failed: refresh.step2.asis_docs: read shard document "doc.bank.reader": '
            "open staging/shards/bank/src-ledger-balereader-overview.md: no such file or directory\n",
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runner_unavailable",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertTrue(result.runtime_contract_failed)
        self.assertFalse(result.runner_unavailable)
        self.assertTrue(
            any("ignored runner/runtime-flow override" in detail for detail in result.issue_details),
            result.issue_details,
        )

    def test_python_report_prefers_runtime_timeout_over_runner_unavailable_when_timeout_signaled(self) -> None:
        run_dir = self.root / "run-timeout-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: runtime_timeout",
                    "- expected_runs: 4",
                    "- completed_runs: 2",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 0",
                    "- running_runs_detected: 1",
                    "- termination_signal: timeout",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","error_code":"runner_unavailable","message":"runtime task timeout after 15s"}\n',
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_timeout",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_timeout", result.failure_class)
        self.assertTrue(result.runtime_timeout)

    def test_python_report_detects_runner_unavailable_capacity_signal(self) -> None:
        run_dir = self.root / "run-capacity-python"
        self._create_fixture_run_dir(run_dir)
        write_text(run_dir / "full-run.log", "provider returned error: Selected model is at capacity\n")
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","message":"Selected model is at capacity"}\n',
        )
        write_text(run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt", "429 Too Many Requests\n")

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runner_unavailable", result.failure_class)
        self.assertTrue(result.runner_unavailable)

    def test_python_runner_unavailable_signal_ignores_codex_plugin_cloudflare_noise(self) -> None:
        text = "\n".join(
            [
                "GET https://chatgpt.com/backend-api/plugins/featured -> 429 Too Many Requests",
                "Just a moment...",
                "Cloudflare",
                "failed to renew cache TTL: Operation not permitted",
            ]
        )

        self.assertFalse(self.module.text_has_runner_unavailable_signal(text))

    def test_python_runner_unavailable_signal_keeps_real_capacity_when_noise_is_separate(self) -> None:
        text = "\n".join(
            [
                "GET https://chatgpt.com/backend-api/plugins/featured -> 429 Too Many Requests",
                "Cloudflare",
                "actual runtime error: Selected model is at capacity",
            ]
        )

        self.assertTrue(self.module.text_has_runner_unavailable_signal(text))

    def test_python_report_ignores_raw_provider_runner_unavailable_word_when_structured_contract_failed(self) -> None:
        run_dir = self.root / "run-raw-word-contract-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: runtime_contract_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt",
            "model narrative mentioned runner_unavailable as a category, but no provider capacity signal exists\n",
        )

        result = self.module.evaluate_run(
            provider="codex-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertTrue(result.runtime_contract_failed)
        self.assertFalse(result.runner_unavailable)

    def test_python_report_ignores_runtime_output_runner_unavailable_word_when_structured_contract_failed(self) -> None:
        run_dir = self.root / "run-structured-runtime-output-word-contract-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: runtime_contract_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"kind":"runtime_output","stream":"stderr","message":"model narrative mentioned runner_unavailable as a category, but no provider capacity signal exists"}\n',
        )

        result = self.module.evaluate_run(
            provider="codex-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertTrue(result.runtime_contract_failed)
        self.assertFalse(result.runner_unavailable)

    def test_python_report_ignores_raw_runner_unavailable_on_terminal_success(self) -> None:
        run_dir = self.root / "run-terminal-success-with-raw-runner-noise"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertEqual("none", result.failure_class)
        self.assertFalse(result.runner_unavailable)
        self.assertTrue(result.hard_pass)

    def test_python_report_ignores_stale_classifier_failure_on_terminal_success(self) -> None:
        run_dir = self.root / "run-terminal-success-with-stale-classifier"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)

        result = self.module.evaluate_run(
            provider="codex-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runner_unavailable",
                "failure_subclass": "cancellation_like",
                "cancellation_like": "1",
                "process_exit": "0",
            },
        )

        self.assertEqual("none", result.failure_class)
        self.assertFalse(result.runner_unavailable)
        self.assertFalse(result.cancellation_like)

    def test_python_report_requires_run_status_for_terminal_success_classifier_override(self) -> None:
        run_dir = self.root / "run-terminal-success-summary-without-run-status"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)
        (run_dir / "run-status.env").unlink()

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runner_unavailable",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runner_unavailable", result.failure_class)
        self.assertTrue(result.runner_unavailable)

    def test_python_report_keeps_structured_runner_unavailable_error_code(self) -> None:
        run_dir = self.root / "run-structured-runner-unavailable-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","kind":"event","error_code":"runner_unavailable","message":"provider unavailable during verdict-only validator repair"}\n',
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runner_unavailable", result.failure_class)
        self.assertTrue(result.runner_unavailable)

    def test_python_report_surfaces_focused_recovery_reasons(self) -> None:
        run_dir = self.root / "run-focused-recovery-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            "\n".join(
                [
                    '{"level":"error","message":"focused artifact repair exhausted","recovery_mode":"validator_verdict_repair"}',
                    '{"level":"error","message":"draft recovery wrote outside the draft artifact write set"}',
                    '{"level":"error","message":"focused artifact repair exhausted","recovery_mode":"draft_artifact_enrichment"}',
                    '{"level":"error","message":"focused artifact repair exhausted","recovery_mode":"draft_artifact_enrichment"}',
                ]
            )
            + "\n",
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_contract_failed",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        focused_details = "\n".join(result.issue_details)
        self.assertIn("reliability/focused-recovery", focused_details)
        self.assertIn("validator_verdict_repair_exhausted", focused_details)
        self.assertIn("draft_artifact_enrichment_exhausted", focused_details)
        self.assertIn("draft_artifact_repair_write_set_violation", focused_details)
        self.assertEqual(4, result.focused_repairs)
        self.assertEqual(3, result.repair_exhausted)
        self.assertIn("execution:repair-exhausted", result.issues)

    def test_python_report_does_not_double_count_focused_recovery_when_quality_counters_exist(self) -> None:
        run_dir = self.root / "run-focused-recovery-no-double-count"
        self._create_fixture_run_dir(run_dir)
        quality_path = run_dir / "snapshots" / "refresh-run" / "reports" / "taskruns" / "refresh-run-quality.json"
        payload = json.loads(quality_path.read_text(encoding="utf-8"))
        payload.setdefault("totals", {}).update(
            {
                "repair_attempts": 1,
                "repair_exhausted": 1,
                "fresh_retries": 0,
                "focused_repairs": 1,
                "stall_count": 0,
                "pre_artifact_stalls": 0,
                "post_artifact_stalls": 0,
                "zero_output_pre_artifact_stalls": 0,
                "partial_failure_count": 0,
            }
        )
        write_json(quality_path, payload)
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            "\n".join(
                [
                    '{"level":"error","message":"focused artifact repair exhausted","recovery_mode":"validator_verdict_repair"}',
                    '{"level":"error","message":"draft recovery wrote outside the draft artifact write set"}',
                    '{"level":"error","message":"focused artifact repair exhausted","recovery_mode":"draft_artifact_enrichment"}',
                    '{"level":"error","message":"focused artifact repair exhausted","recovery_mode":"draft_artifact_enrichment"}',
                ]
            )
            + "\n",
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_contract_failed",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual(1, result.focused_repairs)
        self.assertEqual(1, result.repair_exhausted)
        self.assertIn("execution:repair-exhausted", result.issues)
        self.assertTrue(any("reliability/focused-recovery" in detail for detail in result.issue_details))
        self.assertFalse(any("source=focused-recovery-reasons" in detail for detail in result.issue_details))

    def test_backend_cycle_appends_failed_headless_row_from_workspace_quality_fallback(self) -> None:
        workspace = self.root / "headless-workspace"
        taskruns = workspace / "reports" / "taskruns"
        taskruns.mkdir(parents=True)
        run_id = "run_20260615_120000_abcd"
        write_json(taskruns / f"{run_id}-quality.json", {"totals": {}})
        run_results = self.root / "run-results.tsv"
        run_results.write_text("", encoding="utf-8")
        functions = "\n".join(
            [
                extract_bash_function(BACKEND_CYCLE_SCRIPT.read_text(encoding="utf-8"), "run_result_row_exists"),
                extract_bash_function(BACKEND_CYCLE_SCRIPT.read_text(encoding="utf-8"), "resolve_failed_run_id_from_workspace"),
                extract_bash_function(BACKEND_CYCLE_SCRIPT.read_text(encoding="utf-8"), "append_run_result_row_once"),
            ]
        )
        command = "\n".join(
            [
                "set -euo pipefail",
                f"RUN_RESULTS_TSV={shlex.quote(str(run_results))}",
                "RUN_RESULTS_EXPECTED_FIELDS=17",
                "snapshot_run_artifacts() { return 0; }",
                "quality_metrics() { return 1; }",
                functions,
                (
                    "append_run_result_row_once 1 headless claude-code init '' failed "
                    f"{shlex.quote(str(workspace))} reports"
                ),
                f"cat {shlex.quote(str(run_results))}",
            ]
        )

        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

        fields = completed.stdout.strip().split("\t")
        self.assertEqual(17, len(fields), completed.stdout)
        self.assertEqual(["1", "headless", "claude-code", "init", run_id, "failed"], fields[:6])

    def test_python_report_prefers_parse_signature_contract_failure_over_runner_unavailable(self) -> None:
        run_dir = self.root / "run-parse-signature-python"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "full-run.log",
            (
                "parse runtime draft manifest: json: unknown field \"repo_scopes\"\n"
                "Selected model is at capacity\n"
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            (
                '{"level":"error","message":"parse runtime draft manifest: json: unknown field \\"repo_scopes\\""}\n'
                '{"level":"error","message":"Selected model is at capacity"}\n'
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt",
            "parse runtime draft manifest: json: unknown field \"repo_scopes\"\n429 Too Many Requests\n",
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runner_unavailable",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertTrue(result.runtime_contract_failed)
        self.assertTrue(any("ignored runner/runtime-flow override" in detail for detail in result.issue_details))

    def test_python_report_reads_runner_logs_from_headless_workspace_candidate(self) -> None:
        run_dir = self.root / "run-headless-workspace-python"
        self._create_fixture_run_dir(run_dir)
        source_workspace = run_dir / "arch-workspace"
        headless_workspace = run_dir / "headless" / "arch-workspace"
        headless_workspace.parent.mkdir(parents=True, exist_ok=True)
        source_workspace.rename(headless_workspace)
        write_text(
            headless_workspace / "reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","error_code":"runtime_contract_failed","message":"contract failure"}\n',
        )
        write_text(
            headless_workspace / "reports/taskruns/raw/runtime.stderr.txt",
            "runtime_contract_failed\n",
        )
        write_text(run_dir / "full-run.log", "run completed with contract issue\n")

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_contract_failed",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertTrue(result.runtime_contract_failed)

    def test_python_report_notes_runtime_logs_when_headless_rows_missing(self) -> None:
        run_dir = self.root / "run-headless-rows-missing-with-logs"
        self._create_fixture_run_dir(run_dir)
        (run_dir / "run-results.tsv").unlink()

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "infra_incomplete_cycle",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertTrue(
            any("runtime artifacts/logs found despite missing or failed run-results rows" in detail for detail in result.issue_details)
        )

    def test_runtime_flow_best_effort_partial_skips_run_partial_enforcement_on_runner_unavailable(self) -> None:
        run_dir = self.root / "run-best-effort-capacity"
        workspace = run_dir / "headless/arch-workspace"
        workspace.mkdir(parents=True, exist_ok=True)
        expected_execution = {
            "strategy": "parallel",
            "max_parallel_tasks": 4,
            "failure_policy": "best_effort",
            "shard_discovery_mode": "heuristics",
        }
        headless_rows = {
            "init": {"run_id": "init-run"},
            "refresh": {"run_id": "refresh-run"},
        }
        for pipeline, run_id in (("init", "init-run"), ("refresh", "refresh-run")):
            taskruns_root = run_dir / "snapshots" / run_id / "reports" / "taskruns"
            write_json(
                taskruns_root / f"{run_id}-{pipeline}-step1-collect-shard-plan.json",
                {
                    "strategy": "parallel",
                    "max_parallel_tasks": 4,
                    "failure_policy": "best_effort",
                    "shard_discovery_mode": "heuristics",
                },
            )
            write_json(
                taskruns_root / f"{run_id}-{pipeline}-step1-collect-shard-summary.json",
                {
                    "strategy": "parallel",
                    "max_parallel_tasks": 4,
                    "failure_policy": "best_effort",
                    "shard_discovery_mode": "heuristics",
                    "items": [{"status": "failed"}],
                },
            )
            write_json(
                taskruns_root / run_id / "runtime" / f"{pipeline}-step1-collect" / "runtime-execution.json",
                {
                    "step_id": f"{pipeline}.step1.collect",
                    "shard_id": "domain-a",
                    "repo_scopes": ["demo-repo"],
                    "path_scopes": ["src"],
                    "meta": {
                        "step_id": f"{pipeline}.step1.collect",
                        "shard_id": "domain-a",
                        "repo_scopes": ["demo-repo"],
                        "path_scopes": ["src"],
                    },
                },
            )

        issues, _details = self.module.evaluate_runtime_flow_checks(
            run_dir=run_dir,
            workspace=workspace,
            headless_rows=headless_rows,
            expected_execution=expected_execution,
            summary_text="provider returned 429 Too Many Requests",
            full_run_log_text="Selected model is at capacity",
            runner_unavailable_signal=True,
        )

        self.assertNotIn("runtime:execution-semantics", issues)

    def test_python_report_records_artifact_quality_warning_without_execution_failure(self) -> None:
        run_dir = self.root / "run-artifact-quality"
        self._create_artifact_quality_fixture_run_dir(run_dir)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertEqual("none", result.failure_class)
        self.assertEqual(1, result.artifact_quality_findings)
        self.assertTrue(result.hard_pass)
        self.assertIn("artifact:quality-warning", result.issues)

    def test_shell_classifier_reads_taskrun_logs_and_returns_runtime_contract_failed(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(self.run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_contract_failed", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_prefers_runtime_contract_failed_over_runner_unavailable_signature(self) -> None:
        run_dir = self.root / "run-contract-vs-runner-unavailable-shell"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: runtime_contract_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "full-run.log",
            "runtime_contract_failed while validating draft manifest\nrunner_unavailable marker: status=429\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-contract-vs-runner.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_contract_failed", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_classifies_missing_collect_doc_reference_as_runtime_contract_failed(self) -> None:
        run_dir = self.root / "run-missing-collect-doc-ref-shell"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "full-run.log",
            'run failed: refresh.step2.asis_docs: read shard document "doc.bank.reader": '
            "open staging/shards/bank/src-ledger-balereader-overview.md: no such file or directory\n"
            "provider returned status=429 model at capacity\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-missing-doc-ref.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_contract_failed", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_prefers_runtime_flow_failed_for_validator_verdict_fail(self) -> None:
        run_dir = self.root / "run-validator-verdict-fail-shell"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=pipeline_command_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )
        write_text(
            run_dir / "full-run.log",
            "runtime_contract_failed in step2 diagnostics\nvalidator verdict is FAIL\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-validator-verdict.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_flow_failed", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_prefers_runtime_timeout_over_runner_unavailable_logs(self) -> None:
        run_dir = self.root / "run-timeout-precedence"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: runtime_timeout",
                    "- expected_runs: 4",
                    "- completed_runs: 2",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 0",
                    "- running_runs_detected: 1",
                    "- termination_signal: timeout",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "full-run.log",
            "pipeline timed out after 180s; runner_unavailable observed in step output\n",
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","error_code":"runner_unavailable","message":"runtime task timeout after 15s"}\n',
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-timeout.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_timeout", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_reads_capacity_signal_from_headless_workspace(self) -> None:
        run_dir = self.root / "run-headless-workspace-capacity"
        self._create_fixture_run_dir(run_dir)
        source_workspace = run_dir / "arch-workspace"
        headless_workspace = run_dir / "headless" / "arch-workspace"
        headless_workspace.parent.mkdir(parents=True, exist_ok=True)
        source_workspace.rename(headless_workspace)
        write_text(
            headless_workspace / "reports/taskruns/logs/runtime.ndjson",
            '{"level":"error","message":"Selected model is at capacity"}\n',
        )
        write_text(
            headless_workspace / "reports/taskruns/raw/runtime.stderr.txt",
            "429 Too Many Requests\n",
        )
        write_text(run_dir / "full-run.log", "terminal pipeline failure without explicit error_code\n")

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-capacity-headless.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runner_unavailable", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_runner_unavailable_signature_ignores_codex_plugin_cloudflare_noise(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        noise_log = self.root / "codex-plugin-noise.log"
        write_text(
            noise_log,
            "\n".join(
                [
                    "GET https://chatgpt.com/backend-api/plugins/featured -> 429 Too Many Requests",
                    "Just a moment...",
                    "Cloudflare",
                    "failed to renew cache TTL: Operation not permitted",
                ]
            )
            + "\n",
        )
        command = (
            prelude
            + "\n"
            + f'if contains_runner_unavailable_signature {shlex.quote(str(noise_log))}; then echo yes; else echo no; fi\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("no", completed.stdout.strip(), completed.stdout)

    def test_shell_classifier_ignores_raw_runner_unavailable_on_terminal_success(self) -> None:
        run_dir = self.root / "run-terminal-success-with-raw-runner-noise-shell"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-terminal-success.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "0"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("none", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_runner_unavailable_signature_keeps_real_capacity_when_noise_is_separate(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        mixed_log = self.root / "codex-plugin-noise-with-real-capacity.log"
        write_text(
            mixed_log,
            "\n".join(
                [
                    "GET https://chatgpt.com/backend-api/plugins/featured -> 429 Too Many Requests",
                    "Cloudflare",
                    "actual runtime error: Selected model is at capacity",
                ]
            )
            + "\n",
        )
        command = (
            prelude
            + "\n"
            + f'if contains_runner_unavailable_signature {shlex.quote(str(mixed_log))}; then echo yes; else echo no; fi\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("yes", completed.stdout.strip(), completed.stdout)

    def test_shell_runner_unavailable_signature_ignores_unstructured_category_word(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        narrative_log = self.root / "runner-unavailable-narrative.log"
        write_text(
            narrative_log,
            "model narrative mentioned runner_unavailable as a possible category without capacity signal\n",
        )
        command = (
            prelude
            + "\n"
            + f'if contains_runner_unavailable_signature {shlex.quote(str(narrative_log))}; then echo yes; else echo no; fi\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("no", completed.stdout.strip(), completed.stdout)

    def test_shell_runner_unavailable_signature_detects_bare_429_line(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        bare_log = self.root / "bare-429.log"
        write_text(bare_log, "429\n")
        command = (
            prelude
            + "\n"
            + f'if contains_runner_unavailable_signature {shlex.quote(str(bare_log))}; then echo yes; else echo no; fi\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("yes", completed.stdout.strip(), completed.stdout)

    def test_shell_classifier_prefers_parse_signature_contract_failure_over_capacity_signal(self) -> None:
        run_dir = self.root / "run-shell-parse-signature"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "full-run.log",
            (
                "parse runtime draft manifest: json: unknown field \"repo_scopes\"\n"
                "Selected model is at capacity\n"
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            (
                '{"level":"error","message":"parse runtime draft manifest: json: unknown field \\"repo_scopes\\""}\n'
                '{"level":"error","message":"Selected model is at capacity"}\n'
            ),
        )
        write_text(
            run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt",
            "parse runtime draft manifest: json: unknown field \"repo_scopes\"\n429 Too Many Requests\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-parse-signature.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_contract_failed", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_marks_missing_summary_as_incomplete_cycle_when_batch_reaches_classifier(self) -> None:
        run_dir = self.root / "run-missing-summary"
        run_dir.mkdir(parents=True, exist_ok=True)
        write_text(run_dir / "full-run.log", "child run ended before session summary\n")
        write_text(run_dir / "batch-driver.log", "driver completed with process_exit=1\n")
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                ]
            )
            + "\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-missing-summary.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("infra_incomplete_cycle", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_marks_missing_summary_signal_termination(self) -> None:
        run_dir = self.root / "run-missing-summary-signal"
        run_dir.mkdir(parents=True, exist_ok=True)
        write_text(run_dir / "full-run.log", "terminated by signal\n")
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=signal_terminated",
                    "process_exit=143",
                    "termination_signal=signal_15",
                ]
            )
            + "\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-signal.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "143"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 7, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("infra_signal_terminated", fields[2], classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("signal_15", fields[6], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_keeps_terminal_pipeline_failure_out_of_incomplete_cycle(self) -> None:
        run_dir = self.root / "run-terminal-pipeline-failure"
        self._create_fixture_run_dir(run_dir)
        runtime_log = run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson"
        raw_stderr = run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt"
        if runtime_log.exists():
            runtime_log.unlink()
        if raw_stderr.exists():
            raw_stderr.unlink()
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 10",
                    "- completed_runs: 2",
                    "- expected_headless_runs: 10",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 1",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=pipeline_command_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )
        write_text(run_dir / "full-run.log", "step2 failed to promote staged docs\n")

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-terminal.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "1"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_flow_failed", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_shell_classifier_uses_terminal_process_failed_sentinel_even_when_outer_exit_is_zero(self) -> None:
        run_dir = self.root / "run-terminal-pipeline-failure-zero-exit"
        self._create_fixture_run_dir(run_dir)
        runtime_log = run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson"
        raw_stderr = run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt"
        if runtime_log.exists():
            runtime_log.unlink()
        if raw_stderr.exists():
            raw_stderr.unlink()
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 10",
                    "- completed_runs: 10",
                    "- expected_headless_runs: 10",
                    "- completed_headless_runs: 10",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=0",
                    "termination_signal=none",
                    "failure_reason=pipeline_command_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-terminal-zero-exit.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'classify_run_failure "qwen-code" "1" {shlex.quote(str(run_dir))} "0"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("runtime_flow_failed", fields[2], classifications_tsv.read_text(encoding="utf-8"))

    def test_batch_signal_trap_preserves_child_terminal_sentinel_as_source_of_truth(self) -> None:
        run_dir = self.root / "run-signal-after-child-terminal"
        run_dir.mkdir(parents=True, exist_ok=True)
        write_text(run_dir / "full-run.log", "child already ended before outer batch got signal\n")
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=infra_incomplete_cycle",
                    "summary_written=no",
                ]
            )
            + "\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-signal-trap.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'STARTED_RUN_DIRS=({shlex.quote(str(run_dir))})\n'
            + 'STARTED_RUN_PROVIDERS=("qwen-code")\n'
            + 'STARTED_RUN_INDEXES=("1")\n'
            + 'classify_started_runs_on_signal TERM\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 7, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("infra_incomplete_cycle", fields[2], classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("none", fields[6], classifications_tsv.read_text(encoding="utf-8"))

    def test_batch_exit_trap_classifies_running_started_run_without_summary(self) -> None:
        run_dir = self.root / "run-exit-trap-running"
        run_dir.mkdir(parents=True, exist_ok=True)
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=running",
                    "process_exit=",
                    "termination_signal=none",
                    "failure_reason=",
                    "summary_written=no",
                ]
            )
            + "\n",
        )

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        classifications_tsv = self.root / "backend-run-classifications-exit-trap.tsv"
        command = (
            prelude
            + "\n"
            + f'RUN_CLASSIFICATIONS_TSV={shlex.quote(str(classifications_tsv))}\n'
            + f'STARTED_RUN_DIRS=({shlex.quote(str(run_dir))})\n'
            + 'STARTED_RUN_PROVIDERS=("qwen-code")\n'
            + 'STARTED_RUN_INDEXES=("1")\n'
            + 'on_batch_exit 1\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        fields = classifications_tsv.read_text(encoding="utf-8").strip().split("\t")
        self.assertGreaterEqual(len(fields), 3, classifications_tsv.read_text(encoding="utf-8"))
        self.assertEqual("infra_incomplete_cycle", fields[2], classifications_tsv.read_text(encoding="utf-8"))
        run_status_text = (run_dir / "run-status.env").read_text(encoding="utf-8")
        self.assertIn("state=process_failed", run_status_text)
        self.assertIn("process_exit=1", run_status_text)
        self.assertIn("failure_reason=infra_incomplete_cycle", run_status_text)

    def test_batch_exit_trap_materializes_owner_sidecar_terminal_state(self) -> None:
        batch_root = self.root / "batch-root-sidecar"
        batch_root.mkdir(parents=True, exist_ok=True)
        owner_path = batch_root / "batch-owner.env"

        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        command = (
            prelude
            + "\n"
            + f'BATCH_ROOT={shlex.quote(str(batch_root))}\n'
            + 'BATCH_ID=batch-sidecar\n'
            + 'PROFILE_ID=single-path\n'
            + 'SWEEP_ID=baseline\n'
            + 'RUN_CLASSIFICATIONS_TSV=\n'
            + 'BATCH_OWNER_SENTINEL="$(batch_owner_status_file)"\n'
            + 'start_batch_owner_heartbeat\n'
            + 'on_batch_exit 1\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("", completed.stdout.strip(), completed.stdout)
        self.assertTrue(owner_path.exists(), f"missing owner sentinel: {owner_path}")
        owner_text = owner_path.read_text(encoding="utf-8")
        self.assertIn("state=process_failed", owner_text)
        self.assertIn("process_exit=1", owner_text)
        self.assertIn("failure_reason=batch_exit_nonzero", owner_text)

    def test_python_report_reconstructs_missing_classifier_row_from_run_status(self) -> None:
        batch_root = self.root / "batch-root"
        run_dir = batch_root / "qwen-code" / "run1"
        run_dir.mkdir(parents=True, exist_ok=True)
        write_text(run_dir / "full-run.log", "child stopped before summary persistence\n")
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=signal_terminated",
                    "process_exit=143",
                    "termination_signal=signal_15",
                    "failure_reason=infra_signal_terminated",
                    "summary_written=no",
                ]
            )
            + "\n",
        )

        reconstructed = self.module.reconstruct_backend_classifications(batch_root, {})
        self.assertIn(("qwen-code", 1), reconstructed)
        row = reconstructed[("qwen-code", 1)]
        self.assertEqual("infra_signal_terminated", row["failure_class"])
        self.assertEqual("143", row["process_exit"])
        self.assertEqual("signal_15", row["termination_signal"])

    def test_python_report_marks_run_history_running_as_incomplete_cycle(self) -> None:
        run_dir = self.root / "run-history-running"
        run_dir.mkdir(parents=True, exist_ok=True)
        write_text(run_dir / "full-run.log", "partial run root without summary\n")
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=running",
                    "process_exit=",
                    "termination_signal=none",
                    "failure_reason=",
                    "summary_written=no",
                ]
            )
            + "\n",
        )
        write_json(
            run_dir / "arch-workspace/reports/taskruns/run-history.json",
            {
                "items": [
                    {"id": "run-1", "status": "running"},
                ]
            },
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "infra_incomplete_cycle",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("infra_incomplete_cycle", result.failure_class)
        self.assertTrue(result.infra_incomplete_cycle)
        self.assertTrue(any("run_history_running=1" in detail for detail in result.issue_details))

    def test_python_report_prefers_classifier_incomplete_cycle_over_summary_missing(self) -> None:
        run_dir = self.root / "run-python-missing-summary"
        run_dir.mkdir(parents=True, exist_ok=True)
        write_text(run_dir / "full-run.log", "session ended before summary persistence\n")

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "infra_incomplete_cycle",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("infra_incomplete_cycle", result.failure_class)
        self.assertTrue(result.summary_missing)
        self.assertTrue(result.infra_incomplete_cycle)

    def test_python_report_ignores_classifier_incomplete_cycle_for_terminal_process_failed_summary(self) -> None:
        run_dir = self.root / "run-python-terminal-process-failed"
        self._create_fixture_run_dir(run_dir)
        runtime_log = run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson"
        raw_stderr = run_dir / "arch-workspace/reports/taskruns/raw/runtime.stderr.txt"
        if runtime_log.exists():
            runtime_log.unlink()
        if raw_stderr.exists():
            raw_stderr.unlink()
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 10",
                    "- completed_runs: 2",
                    "- expected_headless_runs: 10",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 1",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=pipeline_command_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )
        write_text(run_dir / "full-run.log", "pipeline failed after writing session summary\n")

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "infra_incomplete_cycle",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_flow_failed", result.failure_class)
        self.assertTrue(result.runtime_flow_failed)
        self.assertFalse(result.infra_incomplete_cycle)

    def test_python_report_skips_runtime_flow_and_cross_repo_for_terminal_runtime_provider_failure(self) -> None:
        run_dir = self.root / "run-python-terminal-provider-failure"
        self._create_fixture_run_dir(run_dir)
        write_text(
            run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=pipeline_command_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )
        write_text(
            run_dir / "session-summary.md",
            "\n".join(
                [
                    "# Session Summary",
                    "",
                    "- result: failed",
                    "- execution_gate: live runtime/frontend evidence only",
                    "- failure_reason: pipeline_command_failed",
                    "- expected_runs: 4",
                    "- completed_runs: 4",
                    "- expected_headless_runs: 2",
                    "- completed_headless_runs: 2",
                    "- running_runs_detected: 0",
                    "- termination_signal: none",
                    "",
                    "## API Simulation",
                    "- status: succeeded",
                    "",
                ]
            ),
        )
        write_text(run_dir / "full-run.log", "runtime_contract_failed in step2 diagnostics\n")

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={
                "expected_repo_count": 2,
                "execution_profile": {
                    "strategy": "parallel",
                    "max_parallel_tasks": 4,
                    "failure_policy": "best_effort",
                    "shard_discovery_mode": "heuristics",
                },
            },
            classification_row={
                "failure_class": "runtime_contract_failed",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertFalse(result.runtime_flow_failed)
        self.assertNotIn("analysis:cross-repo-missing", result.issues)

    def test_python_report_accepts_multi_repo_citations_and_cross_repo_findings_without_edges(self) -> None:
        run_dir = self.root / "run-multi-repo-citations-finding-links"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)

        declared = []
        repo_names = ["course-discovery", "credentials", "devstack", "frontend-platform", "openedx-platform"]
        for repo_name in repo_names:
            repo_root = run_dir / "repos" / repo_name
            write_text(repo_root / "README.md", f"# {repo_name}\n")
            declared.append({"name": repo_name, "source": "path", "path": str(repo_root)})

        manifest_path = (
            run_dir
            / "snapshots/refresh-run/reports/taskruns/refresh-run/staging/shards/domain-a/shard-pack-manifest.json"
        )
        write_json(
            manifest_path,
            {
                "version": 1,
                "run_id": "refresh-run",
                "step_id": "refresh.step1.collect",
                "shard_id": "domain-a",
                "domain_id": "domain-a",
                "agent_role": "shard-analyst",
                "artifact_root": "reports/taskruns/refresh-run/staging/shards/domain-a",
                "repo_scopes": ["course-discovery"],
                "path_scopes": ["src"],
                "documents": [
                    {
                        "id": "doc.arch",
                        "title": "Architecture",
                        "path": "architecture.md",
                        "kind": "report",
                    }
                ],
                "citations": [
                    {
                        "id": f"cite.{repo_name}",
                        "repo": repo_name,
                        "path": "README.md",
                        "claim_ids": ["claim.repo"],
                        "document_ids": ["doc.arch"],
                    }
                    for repo_name in repo_names
                ],
                "semantic": {
                    "coverage": {
                        "observed": repo_names,
                        "missing": ["owner mappings"],
                        "notes": ["multi-repo evidence is represented by citations and a cross-repo finding"],
                    },
                    "questions": [{"id": "q.owner", "text": "Which owners map across the Open edX repositories?"}],
                    "entities": [],
                    "edges": [],
                    "findings": [
                        {
                            "id": "finding.owner",
                            "severity": "medium",
                            "title": "Owner mapping spans repositories",
                            "description": "The same owner mapping gap appears across multiple repository scopes.",
                            "rule_id": "rule.owner.required",
                            "related_ids": ["course-discovery", "frontend-platform", "openedx-platform"],
                            "provenance": {
                                "kind": "observation",
                                "confidence": 0.8,
                                "evidence": [
                                    {"repo": "course-discovery", "path": "README.md"},
                                    {"repo": "frontend-platform", "path": "README.md"},
                                    {"repo": "openedx-platform", "path": "README.md"},
                                ],
                            },
                        }
                    ],
                },
            },
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={
                "declared_repos_meta": {
                    "expected_repo_count": 5,
                    "profile_id": "multi-path",
                    "profile_source_kind": "path",
                    "declared_repos": declared,
                }
            },
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertNotIn("analysis:cross-repo-missing", result.issues)
        self.assertFalse(result.semantic_hard_fail)
        self.assertTrue(result.hard_pass)

    def test_python_report_accepts_validator_cross_repo_finding_without_edges(self) -> None:
        run_dir = self.root / "run-multi-repo-validator-cross-finding"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)

        declared = []
        repo_names = ["course-discovery", "frontend-platform", "openedx-platform"]
        for repo_name in repo_names:
            repo_root = run_dir / "repos" / repo_name
            write_text(repo_root / "README.md", f"# {repo_name}\n")
            declared.append({"name": repo_name, "source": "path", "path": str(repo_root)})

        verdict_path = run_dir / "snapshots/refresh-run/reports/taskruns/refresh-run/validator/validator-verdict.json"
        write_json(
            verdict_path,
            {
                "version": 1,
                "run_id": "refresh-run",
                "generated_at": "2026-05-20T01:26:13Z",
                "verdict": "PASS",
                "checked_paths": ["reports/taskruns/refresh-run/staging/final/final-run-index.json"],
                "findings": [
                    {
                        "id": "finding.cross_repo_owner_gap",
                        "severity": "medium",
                        "title": "Owner mapping spans Open edX repositories",
                        "description": "Ownership evidence is spread across multiple repository scopes.",
                        "related_ids": ["course-discovery", "frontend-platform"],
                        "provenance": {
                            "kind": "observation",
                            "confidence": 0.8,
                            "evidence": [
                                {"repo": "course-discovery", "path": "README.md"},
                                {"repo": "frontend-platform", "path": "README.md"},
                            ],
                        },
                    }
                ],
                "questions": [],
                "issues": [],
            },
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={
                "declared_repos_meta": {
                    "expected_repo_count": 3,
                    "profile_id": "multi-path",
                    "profile_source_kind": "path",
                    "declared_repos": declared,
                }
            },
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertNotIn("analysis:cross-repo-missing", result.issues)
        self.assertFalse(result.semantic_hard_fail)
        self.assertTrue(result.hard_pass)

    def test_python_report_accepts_cross_repo_question_with_repo_citations(self) -> None:
        run_dir = self.root / "run-multi-repo-cross-question"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)

        declared = []
        repo_names = ["course-discovery", "frontend-platform"]
        for repo_name in repo_names:
            repo_root = run_dir / "repos" / repo_name
            write_text(repo_root / "README.md", f"# {repo_name}\n")
            declared.append({"name": repo_name, "source": "path", "path": str(repo_root)})

        manifest_path = (
            run_dir
            / "snapshots/refresh-run/reports/taskruns/refresh-run/staging/shards/domain-a/shard-pack-manifest.json"
        )
        write_json(
            manifest_path,
            {
                "version": 1,
                "run_id": "refresh-run",
                "step_id": "refresh.step1.collect",
                "shard_id": "domain-a",
                "domain_id": "domain-a",
                "agent_role": "shard-analyst",
                "artifact_root": "reports/taskruns/refresh-run/staging/shards/domain-a",
                "repo_scopes": repo_names,
                "path_scopes": ["src"],
                "documents": [{"id": "doc.arch", "title": "Architecture", "path": "architecture.md", "kind": "report"}],
                "citations": [
                    {
                        "id": f"cite.{repo_name}",
                        "repo": repo_name,
                        "path": "README.md",
                        "claim_ids": [f"claim.{repo_name}"],
                        "document_ids": ["doc.arch"],
                    }
                    for repo_name in repo_names
                ],
                "semantic": {
                    "coverage": {"observed": repo_names, "missing": ["cross-repo ownership"], "notes": []},
                    "questions": [
                        {
                            "id": "q.cross_repo_owner",
                            "text": "Which owner contract connects course-discovery and frontend-platform?",
                            "related_ids": repo_names,
                        }
                    ],
                    "entities": [],
                    "edges": [],
                    "findings": [],
                },
            },
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={
                "declared_repos_meta": {
                    "expected_repo_count": 2,
                    "profile_id": "multi-path",
                    "profile_source_kind": "path",
                    "declared_repos": declared,
                }
            },
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertNotIn("analysis:cross-repo-missing", result.issues)
        self.assertFalse(result.semantic_hard_fail)
        self.assertTrue(result.hard_pass)

    def test_python_report_explains_cross_repo_missing_dimensions(self) -> None:
        run_dir = self.root / "run-multi-repo-no-cross-links"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)

        declared = []
        repo_names = ["course-discovery", "credentials", "devstack"]
        for repo_name in repo_names:
            repo_root = run_dir / "repos" / repo_name
            write_text(repo_root / "README.md", f"# {repo_name}\n")
            declared.append({"name": repo_name, "source": "path", "path": str(repo_root)})

        manifest_path = (
            run_dir
            / "snapshots/refresh-run/reports/taskruns/refresh-run/staging/shards/domain-a/shard-pack-manifest.json"
        )
        write_json(
            manifest_path,
            {
                "version": 1,
                "run_id": "refresh-run",
                "step_id": "refresh.step1.collect",
                "shard_id": "domain-a",
                "domain_id": "domain-a",
                "agent_role": "shard-analyst",
                "artifact_root": "reports/taskruns/refresh-run/staging/shards/domain-a",
                "repo_scopes": repo_names,
                "path_scopes": ["src"],
                "documents": [{"id": "doc.arch", "title": "Architecture", "path": "architecture.md", "kind": "report"}],
                "citations": [
                    {
                        "id": f"cite.{repo_name}",
                        "repo": repo_name,
                        "path": "README.md",
                        "claim_ids": [f"claim.{repo_name}"],
                        "document_ids": ["doc.arch"],
                    }
                    for repo_name in repo_names
                ],
                "semantic": {
                    "coverage": {"observed": repo_names, "missing": [], "notes": []},
                    "questions": [],
                    "entities": [],
                    "edges": [],
                    "findings": [],
                },
            },
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={
                "declared_repos_meta": {
                    "expected_repo_count": 3,
                    "profile_id": "multi-path",
                    "profile_source_kind": "path",
                    "declared_repos": declared,
                }
            },
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertIn("analysis:cross-repo-missing", result.issues)
        self.assertTrue(result.semantic_hard_fail)
        self.assertTrue(any("missing_dimensions=no_semantic_edges_or_cross_repo_links" in detail for detail in result.issue_details))

    def test_python_report_ignores_provider_model_telemetry_diagnostics(self) -> None:
        run_dir = self.root / "run-provider-model-mismatch"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"kind":"runtime_output","stream":"stdout","message":"{\\"model\\":\\"claude-opus-4-5-20251101\\"}"}\n',
        )

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertNotIn("reliability:provider-model-mismatch", result.issues)
        self.assertFalse(any("claude-opus" in detail for detail in result.issue_details))

    def test_python_report_ignores_nested_provider_model_usage_telemetry(self) -> None:
        run_dir = self.root / "run-nested-provider-model-mismatch"
        self._create_passed_run_dir_with_raw_runner_noise(run_dir)
        write_text(
            run_dir / "arch-workspace/reports/taskruns/logs/runtime.ndjson",
            '{"kind":"runtime_output","stream":"stdout","message":"{\\"modelUsage\\":{\\"kimi-for-coding\\":{\\"inputTokens\\":1}}}"}\n',
        )

        result = self.module.evaluate_run(
            provider="claude-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "none",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "0",
            },
        )

        self.assertNotIn("reliability:provider-model-mismatch", result.issues)
        self.assertFalse(any("kimi-for-coding" in detail for detail in result.issue_details))

    def test_python_report_treats_incomplete_reports_as_triage_only_not_empty_analysis(self) -> None:
        run_dir = self.root / "run-incomplete"
        self._create_incomplete_fixture_run_dir(run_dir)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_contract_failed",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertNotIn("analysis:overview", result.issues)
        self.assertNotIn("analysis:findings", result.issues)
        self.assertNotIn("analysis:coverage", result.issues)
        self.assertNotIn("analysis:questions", result.issues)

    def test_python_report_accepts_partial_incomplete_reports_with_substantive_coverage(self) -> None:
        run_dir = self.root / "run-partial-incomplete"
        self._create_partial_incomplete_fixture_run_dir(run_dir)

        result = self.module.evaluate_run(
            provider="qwen-code",
            run_index=1,
            run_dir=run_dir,
            preflight={},
            classification_row={
                "failure_class": "runtime_contract_failed",
                "failure_subclass": "none",
                "cancellation_like": "0",
                "process_exit": "1",
            },
        )

        self.assertEqual("runtime_contract_failed", result.failure_class)
        self.assertNotIn("analysis:coverage", result.issues)
        self.assertNotIn("analysis:questions", result.issues)
        self.assertNotIn("analysis:findings", result.issues)
        self.assertTrue(any("collect_partial_shard_failures" in detail for detail in result.issue_details))

    def test_shell_frontend_mode_helpers_support_per_run(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        command = (
            prelude
            + "\n"
            + "RUN_COUNT=5\n"
            + "BATCH_RUN_SELECTION=2,4\n"
            + "BATCH_FRONTEND_MODE=per_run\n"
            + "resolve_selected_run_indexes\n"
            + 'if should_run_frontend_once; then echo "live_once=1"; else echo "live_once=0"; fi\n'
            + 'if should_run_frontend_for_run 2; then echo "live_run2=1"; else echo "live_run2=0"; fi\n'
            + 'if should_run_frontend_for_run 1; then echo "live_run1=1"; else echo "live_run1=0"; fi\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        observed = set(line.strip() for line in completed.stdout.splitlines() if line.strip())
        self.assertSetEqual(
            {"live_once=0", "live_run2=1", "live_run1=0"},
            observed,
        )

    def test_shell_frontend_mode_helpers_mark_auto_skip_when_run1_not_selected(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        command = (
            prelude
            + "\n"
            + "RUN_COUNT=5\n"
            + "BATCH_RUN_SELECTION=2,4\n"
            + "BATCH_FRONTEND_MODE=auto\n"
            + "resolve_selected_run_indexes\n"
            + 'if should_run_frontend_once; then echo "live_once=1"; else echo "live_once=0"; fi\n'
            + 'if should_write_frontend_skip_result; then echo "live_skip=1"; else echo "live_skip=0"; fi\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        observed = set(line.strip() for line in completed.stdout.splitlines() if line.strip())
        self.assertSetEqual({"live_once=0", "live_skip=1"}, observed)

    def test_shell_frontend_once_uses_first_selected_run_for_always_mode(self) -> None:
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        command = (
            prelude
            + "\n"
            + "RUN_COUNT=5\n"
            + "BATCH_ROOT=/tmp/provenarch-batch\n"
            + "BATCH_RUN_SELECTION=2,4\n"
            + "BATCH_FRONTEND_MODE=always\n"
            + "resolve_selected_run_indexes\n"
            + 'printf "%s\\n" "$(resolve_frontend_live_backend_run qwen-code)"\n'
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertEqual("/tmp/provenarch-batch/qwen-code/run2\t2", completed.stdout.strip())

    def test_shell_frontend_snapshot_missing_after_backend_failure_is_skipped(self) -> None:
        backend_run_dir = self.root / "frontend-backend-failed-run"
        output_dir = self.root / "frontend-output"
        (backend_run_dir / "arch-workspace").mkdir(parents=True, exist_ok=True)
        write_text(
            backend_run_dir / "run-status.env",
            "\n".join(
                [
                    "provider=qwen-code",
                    "run_index=1",
                    "state=process_failed",
                    "process_exit=1",
                    "termination_signal=none",
                    "failure_reason=runtime_contract_failed",
                    "summary_written=yes",
                ]
            )
            + "\n",
        )
        script_text = FULL_RUN_BATCH_SCRIPT.read_text(encoding="utf-8")
        prelude, _ = script_text.split('\nif [[ ! "$RUN_COUNT" =~', 1)
        result_path = output_dir / "frontend-e2e-result.json"
        command = (
            prelude
            + "\n"
            + f'run_frontend_live_e2e "qwen-code" {shlex.quote(str(backend_run_dir))} {shlex.quote(str(output_dir))} "1"\n'
            + "python3 - <<'PY'\n"
            + "import json\n"
            + f"from pathlib import Path\np = Path({str(result_path)!r})\n"
            + "payload = json.loads(p.read_text(encoding='utf-8'))\n"
            + "print(payload['status'] + ':' + payload['reason'])\n"
            + "PY\n"
        )
        completed = subprocess.run(
            ["bash", "-lc", command],
            check=True,
            capture_output=True,
            text=True,
            env={**os.environ, "PROVENARCH_ROOT": str(REPO_ROOT)},
        )
        self.assertIn("skipped:snapshot_reports_missing", completed.stdout.strip(), completed.stdout)

    def test_python_frontend_matrix_supports_per_run_results(self) -> None:
        batch_root = self.root / "batch"
        reports_root = self.root / "reports"
        write_json(
            batch_root / "frontend/qwen-code/run1/frontend-e2e-result.json",
            {
                "status": "passed",
                "reason": "ok",
                "runtime_provider": "qwen-code",
                "workspace": "/tmp/qwen-run1",
                "base_url": "http://127.0.0.1:18081",
                "runtime_command": "qwen",
            },
        )
        write_json(
            batch_root / "frontend/qwen-code/run3/frontend-e2e-result.json",
            {
                "status": "failed",
                "reason": "browser_closed",
                "runtime_provider": "qwen-code",
                "workspace": "/tmp/qwen-run3",
                "base_url": "http://127.0.0.1:18083",
                "runtime_command": "qwen",
            },
        )
        write_json(
            batch_root / "frontend/qwen-code/run4/frontend-e2e-result.json",
            {
                "status": "failed",
                "reason": "runtime_run_failed",
                "runtime_provider": "qwen-code",
                "workspace": "/tmp/qwen-run4",
                "base_url": "http://127.0.0.1:18084",
                "runtime_command": "qwen",
            },
        )
        write_json(
            batch_root / "frontend/claude-code/frontend-e2e-result.json",
            {
                "status": "passed",
                "reason": "ok",
                "runtime_provider": "claude-code",
                "workspace": "/tmp/claude",
                "base_url": "http://127.0.0.1:18082",
                "runtime_command": "claude",
            },
        )

        frontend = self.module.load_frontend_results(batch_root)
        self.assertEqual(4, len(frontend))

        matrix_path = reports_root / "frontend-matrix.md"
        self.module.write_frontend_matrix(matrix_path, frontend)
        matrix_text = matrix_path.read_text(encoding="utf-8")

        self.assertIn("| qwen-code | failed | 3 | browser_closed=1, ok=1, runtime_run_failed=1 |", matrix_text)
        self.assertIn("| claude-code | passed | 1 |", matrix_text)
        self.assertIn("| qwen-code | 1 | passed | ok |", matrix_text)
        self.assertIn("| qwen-code | 3 | failed | browser_closed |", matrix_text)
        self.assertIn("| qwen-code | 4 | failed | runtime_run_failed |", matrix_text)

    def test_execution_report_respects_selected_provider_surface(self) -> None:
        reports_root = self.root / "reports-selected"
        quality_path = reports_root / "quality.md"
        frontend = [
            {
                "status": "passed",
                "reason": "ok",
                "runtime_provider": "qwen-code",
                "run_index": 1,
                "workspace": "/tmp/qwen-run1",
                "runtime_command": "qwen",
            }
        ]
        runs = [
            self.module.RunEvaluation(
                provider="qwen-code",
                run_index=1,
                run_dir=self.root / "qwen-only-run1",
                hard_pass=True,
                reliability=30,
                contract=20,
                analysis=20,
                total=70,
                verdict="PASS",
            )
        ]
        preflight = {
            "generated_at_utc": "2026-04-18T00:00:00Z",
            "provenarch_sha": "abc123",
            "target_repos_file": "examples/repos.txt",
            "declared_repos_meta": {
                "profile_id": "single-git_url",
                "profile_source_kind": "git_url",
                "expected_repo_count": 1,
                "declared_repos": [{"name": "bank-of-anthos"}],
            },
            "selected_providers": ["qwen-code"],
            "selected_run_indexes": ["1"],
            "runtimes": {
                "claude": {"version_line": "not-selected"},
                "qwen": {"version_line": "qwen 0.1"},
            },
        }

        self.module.write_execution_report(
            quality_path,
            "batch-qwen-only",
            runs,
            frontend,
            preflight,
            ["qwen-code"],
        )
        report = quality_path.read_text(encoding="utf-8")
        self.assertIn("`1/1` backend full-runs", report)
        self.assertIn("выбранных провайдеров (`1/1`)", report)
        self.assertNotIn("10/10", report)
        self.assertNotIn("2/2", report)
        self.assertNotIn("| claude-code |", report)

    def test_execution_report_marks_frontend_skipped_as_not_run(self) -> None:
        reports_root = self.root / "reports-frontend-skipped"
        quality_path = reports_root / "quality.md"
        frontend = [
            {
                "status": "skipped",
                "reason": "frontend_disabled",
                "runtime_provider": "codex-code",
                "run_index": 1,
                "workspace": "/tmp/codex-run1",
                "runtime_command": "codex",
            }
        ]
        runs = [
            self.module.RunEvaluation(
                provider="codex-code",
                run_index=1,
                run_dir=self.root / "codex-run1",
                hard_pass=True,
                reliability=40,
                contract=20,
                analysis=30,
                total=90,
                verdict="PASS",
            )
        ]
        preflight = {
            "generated_at_utc": "2026-04-18T00:00:00Z",
            "provenarch_sha": "abc123",
            "target_repos_file": "examples/repos.txt",
            "declared_repos_meta": {
                "profile_id": "single-git_url",
                "profile_source_kind": "git_url",
                "expected_repo_count": 1,
                "declared_repos": [{"name": "bank-of-anthos"}],
            },
            "selected_providers": ["codex-code"],
            "selected_run_indexes": ["1"],
            "runtimes": {
                "codex": {"version_line": "codex-cli 0.131.0"},
            },
        }

        self.module.write_execution_report(
            quality_path,
            "batch-codex-skipped-frontend",
            runs,
            frontend,
            preflight,
            ["codex-code"],
        )
        report = quality_path.read_text(encoding="utf-8")
        self.assertIn("Frontend live e2e не запускался для этого batch", report)
        self.assertNotIn("Frontend live e2e не полностью стабилен", report)
        self.assertIn("frontend live smoke обязателен только для frontend-enabled/release surfaces", report)

    def test_selected_surface_is_resolved_from_preflight(self) -> None:
        batch_root = self.root / "selected-surface"
        classifications = {
            ("qwen-code", 1): {"failure_class": "none"},
            ("qwen-code", 2): {"failure_class": "none"},
        }
        preflight = {
            "selected_providers": ["qwen-code"],
            "selected_run_indexes": ["1", "2"],
        }

        providers = self.module.resolve_selected_providers(preflight, classifications, batch_root)
        run_indexes = self.module.resolve_selected_run_indexes(preflight, classifications, batch_root)

        self.assertEqual(["qwen-code"], providers)
        self.assertEqual([1, 2], run_indexes)

if __name__ == "__main__":
    unittest.main()
