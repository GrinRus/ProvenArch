#!/usr/bin/env python3
from __future__ import annotations

import os
import json
import signal
import subprocess
import tempfile
import textwrap
import time
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
FULL_RUN_SCRIPT = REPO_ROOT / "scripts" / "full-run-ai-advent.sh"
BATCH_SCRIPT = REPO_ROOT / "scripts" / "full-run-batch-5x2.sh"


def write_text(path: Path, content: str, mode: int = 0o644) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    path.chmod(mode)


def parse_summary_scalar(summary: str, key: str) -> str:
    needle = f"- {key}: "
    for line in summary.splitlines():
        if line.startswith(needle):
            return line[len(needle) :].strip()
    return ""


def create_full_run_stub_environment(root: Path) -> tuple[Path, Path, Path, Path, Path]:
    provenarch_root = root / "provenarch-root"
    tools_dir = root / "tools"
    target_repo = root / "target-repo"
    tmp_root = root / "tmp-run"
    repos_file = root / "inputs" / "repos.yaml"
    acp_stub = root / "acp-stub.sh"

    write_text(target_repo / "README.md", "# target repo\n")
    write_text(
        repos_file,
        textwrap.dedent(
            f"""\
            version: 1
            repos:
              - name: target-repo
                path: {target_repo}
            docs:
              imports_path: ./docs/imports
            """
        ),
    )

    write_text(
        acp_stub,
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            set -euo pipefail
            cmd="${1:-}"
            if [[ -z "$cmd" ]]; then
              echo "missing command" >&2
              exit 1
            fi
            shift || true

            if [[ "$cmd" == "init-workspace" ]]; then
              workspace=""
              while [[ $# -gt 0 ]]; do
                case "$1" in
                  --workspace)
                    workspace="$2"
                    shift 2
                    ;;
                  *)
                    shift
                    ;;
                esac
              done
              mkdir -p "$workspace/.git" "$workspace/skills" "$workspace/reports/taskruns" "$workspace/reports/as-is" "$workspace/reports/findings" "$workspace/reports/coverage"
              cat >"$workspace/workspace.yaml" <<'YAML'
            version: 1
            repos:
              - name: target-repo
                path: /tmp/target-repo
            YAML
              cat >"$workspace/skills/subagents.yaml" <<'YAML'
            version: 1
            subagents: []
            YAML
              exit 0
            fi

            if [[ "$cmd" == "serve" ]]; then
              trap 'exit 0' TERM INT HUP
              while true; do
                sleep 1
              done
            fi

            if [[ "$cmd" == "run" ]]; then
              workspace=""
              runtime=""
              pipeline=""
              runtime_provider=""
              while [[ $# -gt 0 ]]; do
                case "$1" in
                  --workspace)
                    workspace="$2"
                    shift 2
                    ;;
                  --runtime)
                    runtime="$2"
                    shift 2
                    ;;
                  --pipeline)
                    pipeline="$2"
                    shift 2
                    ;;
                  --runtime-provider)
                    runtime_provider="$2"
                    shift 2
                    ;;
                  *)
                    shift
                    ;;
                esac
              done

              mkdir -p "$workspace/reports/taskruns" "$workspace/reports/as-is" "$workspace/reports/findings" "$workspace/reports/coverage"
              counter_file="$workspace/.stub-run-counter"
              counter=0
              if [[ -f "$counter_file" ]]; then
                counter="$(cat "$counter_file")"
              fi
              counter=$((counter + 1))
              printf '%s\\n' "$counter" >"$counter_file"
              run_id="run-${runtime}-${pipeline}-${counter}"

              runtime_name="$runtime_provider"
              runtime_version=""
              if [[ "$runtime" == "fake" ]]; then
                runtime_name="fake"
                runtime_version="fake"
              elif [[ "$runtime" == "headless" ]]; then
                if [[ -z "$runtime_name" ]]; then
                  runtime_name="${ACP_RUNTIME_PROVIDER:-qwen-code}"
                fi
                case "$runtime_name" in
                  qwen-code) runtime_version="qwen-cli" ;;
                  claude-code) runtime_version="claude-cli" ;;
                  *) runtime_version="headless" ;;
                esac
              fi
              runtime_key="$runtime_name"
              if [[ -n "$runtime_version" ]]; then
                runtime_key="${runtime_name}@${runtime_version}"
              fi
              if [[ "${ACP_STUB_RUN_FORCE_EXIT_CODE:-0}" != "0" ]]; then
                exit "${ACP_STUB_RUN_FORCE_EXIT_CODE}"
              fi

              if [[ "${ACP_STUB_RUN_SLEEP:-0}" != "0" ]]; then
                sleep "${ACP_STUB_RUN_SLEEP}"
              fi

              quality_path="$workspace/reports/taskruns/${run_id}-quality.json"
              python3 - "$quality_path" "$run_id" "$pipeline" "$runtime_name" "$runtime_version" "$runtime_key" <<'PY'
            import json
            import sys
            path, run_id, pipeline, runtime_name, runtime_version, runtime_key = sys.argv[1:]
            payload = {
                "version": 1,
                "run_id": run_id,
                "pipeline": pipeline,
                "status": "succeeded",
                "generated_at": "2026-04-12T10:00:00Z",
                "runtime_versions": [runtime_key],
                "totals": {
                    "steps": 2,
                    "changeset_ops": 2,
                    "entity_upserts": 1,
                    "edge_upserts": 1,
                    "findings_added": 1,
                    "questions_count": 1,
                    "coverage_observed": 2,
                    "coverage_missing": 2,
                    "warnings_count": 0,
                    "signal_score": 9,
                },
                "steps": [
                    {
                        "step_id": f"{pipeline}.step1.collect",
                        "runtime_name": runtime_name,
                        "runtime_version": runtime_version,
                        "domain_id": "domain-main",
                        "repo_scopes": ["target-repo"],
                        "changeset_ops": 1,
                        "entity_upserts": 1,
                        "edge_upserts": 0,
                        "findings_added": 0,
                        "questions_count": 1,
                        "coverage_observed": 1,
                        "coverage_missing": 1,
                        "warnings_count": 0,
                    },
                    {
                        "step_id": f"{pipeline}.step3.findings",
                        "runtime_name": runtime_name,
                        "runtime_version": runtime_version,
                        "domain_id": "domain-main",
                        "repo_scopes": ["target-repo"],
                        "changeset_ops": 1,
                        "entity_upserts": 0,
                        "edge_upserts": 1,
                        "findings_added": 1,
                        "questions_count": 0,
                        "coverage_observed": 1,
                        "coverage_missing": 1,
                        "warnings_count": 0,
                    },
                ],
            }
            with open(path, "w", encoding="utf-8") as f:
                json.dump(payload, f, ensure_ascii=True, indent=2)
                f.write("\\n")
            PY

              cat >"$workspace/reports/as-is/overview.md" <<'MD'
            # Overview

            - services: 1
            - datastores: 1
            - integrations: 1
            - teams: 1
            MD
              cat >"$workspace/reports/findings/findings.md" <<'MD'
            # Findings

            ## Missing Owner Mapping
            - Severity: medium
            - Description: Owner mapping requires confirmation.
            MD
              cat >"$workspace/reports/coverage/summary.md" <<'MD'
            # Coverage

            ## Missing
            - owner mappings
            - ci-cd evidence
            - delta validation

            ## Notes
            - owner mapping is incomplete
            MD
              cat >"$workspace/reports/coverage/open-questions.md" <<'MD'
            # Open Questions

            - `q.owner.mapping` Which team owns target-repo?
            MD

              if [[ "${ACP_STUB_DISABLE_RUN_HISTORY:-0}" != "1" ]]; then
                run_history="$workspace/reports/taskruns/run-history.json"
                status="${ACP_STUB_RUN_HISTORY_STATUS:-succeeded}"
                history_mode="${ACP_STUB_RUN_HISTORY_MODE:-runs}"
                python3 - "$run_history" "$run_id" "$status" "$history_mode" <<'PY'
            import json
            import os
            import sys
            path, run_id, status, mode = sys.argv[1:]
            payload = {"runs": []}
            if os.path.exists(path):
                with open(path, encoding="utf-8") as f:
                    payload = json.load(f)
            key = "items" if mode == "items" else "runs"
            if mode == "items":
                payload.setdefault("version", 1)
            items = payload.get(key)
            if not isinstance(items, list):
                items = []
            items.append({"run_id": run_id, "status": status})
            payload[key] = items
            with open(path, "w", encoding="utf-8") as f:
                json.dump(payload, f, ensure_ascii=True, indent=2)
                f.write("\\n")
            PY
              fi

              echo "status: succeeded"
              echo "run_id: $run_id"
              exit 0
            fi

            echo "unsupported acp command: $cmd" >&2
            exit 1
            """
        ),
        mode=0o755,
    )

    write_text(
        provenarch_root / "Makefile",
        textwrap.dedent(
            """\
            .PHONY: build contracts test lint
            build:
            \tmkdir -p bin
            \tcp "$(ACP_STUB_SOURCE)" bin/acp
            \tchmod +x bin/acp
            contracts test lint:
            \t@:
            """
        ),
    )
    write_text(provenarch_root / ".gitkeep", "stub\n")

    write_text(
        tools_dir / "qwen",
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            if [[ "${1:-}" == "--version" ]]; then
              echo "qwen 1.0.0"
              exit 0
            fi
            echo "qwen stub"
            exit 0
            """
        ),
        mode=0o755,
    )
    write_text(
        tools_dir / "curl",
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            set -euo pipefail
            url="${@: -1}"
            if [[ "$url" == *"/api/health" ]]; then
              echo '{"ok":true}'
              exit 0
            fi
            if [[ "$url" == *"/api/workspace/validate" ]]; then
              repo_path="${ACP_STUB_TARGET_REPO:-/tmp/target-repo}"
              printf '{"ok":true,"workspace":"%s","warnings":[],"errors":[],"resolved_repos":[{"name":"target-repo","source":"path","path":"%s"}]}\\n' "${ACP_STUB_WORKSPACE:-/tmp/workspace}" "$repo_path"
              exit 0
            fi
            if [[ "$url" == *"/api/pipeline/init" ]]; then
              echo '{"run_id":"api-init-1"}'
              exit 0
            fi
            if [[ "$url" == *"/api/pipeline/runs/"*"/artifacts" ]]; then
              echo '{"artifacts":[{"path":"reports/as-is/overview.md"}]}'
              exit 0
            fi
            if [[ "$url" == *"/api/pipeline/runs/"*"/logs"* ]]; then
              echo '{"items":[{"id":"1","message":"ok"}]}'
              exit 0
            fi
            if [[ "$url" == *"/api/pipeline/runs/"* ]]; then
              echo '{"status":"succeeded"}'
              exit 0
            fi
            echo '{}'
            exit 0
            """
        ),
        mode=0o755,
    )

    return provenarch_root, tools_dir, target_repo, tmp_root, repos_file


class FullRunStabilityIntegrationTests(unittest.TestCase):
    def test_full_run_uses_isolated_baseline_and_headless_workspaces(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo, tmp_root, repos_file = create_full_run_stub_environment(root)
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TMP_ROOT": str(tmp_root),
                    "KEEP_TMP": "1",
                    "ITERATIONS": "1",
                    "RUN_QUALITY_GATES": "0",
                    "TARGET_REPOS_FILE": str(repos_file),
                    "ACP_RUNTIME_PROVIDER": "qwen-code",
                    "ACP_QWEN_CMD": str(tools_dir / "qwen"),
                    "ACP_STUB_SOURCE": str(root / "acp-stub.sh"),
                    "ACP_STUB_TARGET_REPO": str(target_repo),
                    "ACP_APPLY_TIMEOUTS_VIA_API": "0",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(FULL_RUN_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=120,
            )
            self.assertEqual(0, result.returncode, msg=result.stdout + "\n" + result.stderr)

            summary_path = tmp_root / "session-summary.md"
            self.assertTrue(summary_path.exists(), msg="session-summary.md is missing")
            summary = summary_path.read_text(encoding="utf-8")
            self.assertEqual("passed", parse_summary_scalar(summary, "result"))
            self.assertEqual(str(tmp_root / "arch-workspace-baseline"), parse_summary_scalar(summary, "workspace_baseline"))
            self.assertEqual(str(tmp_root / "arch-workspace"), parse_summary_scalar(summary, "workspace_headless"))

            run_results_path = tmp_root / "run-results.tsv"
            self.assertTrue(run_results_path.exists(), msg="run-results.tsv is missing")
            rows = [line for line in run_results_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertEqual(4, len(rows), msg="expected 4 rows for ITERATIONS=1 (fake+headless init/refresh)")

            fake_rows = []
            headless_rows = []
            for row in rows:
                cols = row.split("\t")
                self.assertGreaterEqual(len(cols), 15, msg=f"unexpected run-results row shape: {row}")
                runtime_mode = cols[1]
                quality_path = cols[14]
                if runtime_mode == "fake":
                    fake_rows.append(quality_path)
                if runtime_mode == "headless":
                    headless_rows.append(quality_path)

            self.assertEqual(2, len(fake_rows), msg=f"unexpected fake rows: {rows}")
            self.assertEqual(2, len(headless_rows), msg=f"unexpected headless rows: {rows}")
            self.assertTrue(
                all("/arch-workspace-baseline/" in path for path in fake_rows),
                msg=f"fake rows must write baseline workspace quality files: {fake_rows}",
            )
            self.assertTrue(
                all("/arch-workspace/" in path and "/arch-workspace-baseline/" not in path for path in headless_rows),
                msg=f"headless rows must write headless workspace quality files: {headless_rows}",
            )

            for quality_path in headless_rows:
                payload = json.loads(Path(quality_path).read_text(encoding="utf-8"))
                runtime_versions = payload.get("runtime_versions") or []
                self.assertTrue(runtime_versions, msg=f"missing runtime_versions in {quality_path}")
                self.assertTrue(
                    all("fake" not in str(item).lower() for item in runtime_versions),
                    msg=f"headless quality summary leaked fake runtime markers: {runtime_versions}",
                )

    def test_full_run_marks_incomplete_cycle(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo, tmp_root, repos_file = create_full_run_stub_environment(root)
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TMP_ROOT": str(tmp_root),
                    "KEEP_TMP": "1",
                    "ITERATIONS": "1",
                    "RUN_QUALITY_GATES": "0",
                    "TARGET_REPOS_FILE": str(repos_file),
                    "ACP_RUNTIME_PROVIDER": "qwen-code",
                    "ACP_QWEN_CMD": str(tools_dir / "qwen"),
                    "ACP_STUB_SOURCE": str(root / "acp-stub.sh"),
                    "ACP_STUB_TARGET_REPO": str(target_repo),
                    "ACP_STUB_DISABLE_RUN_HISTORY": "1",
                    "ACP_APPLY_TIMEOUTS_VIA_API": "0",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(FULL_RUN_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=120,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)

            summary_path = tmp_root / "session-summary.md"
            self.assertTrue(summary_path.exists(), msg="session-summary.md is missing")
            summary = summary_path.read_text(encoding="utf-8")
            self.assertEqual("failed", parse_summary_scalar(summary, "result"))
            self.assertEqual("infra_incomplete_cycle", parse_summary_scalar(summary, "failure_reason"))
            self.assertEqual("4", parse_summary_scalar(summary, "expected_runs"))
            self.assertEqual("2", parse_summary_scalar(summary, "expected_headless_runs"))
            completed_runs = parse_summary_scalar(summary, "completed_runs")
            self.assertEqual("4", completed_runs)

    def test_full_run_tracks_running_history_with_items_format(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo, tmp_root, repos_file = create_full_run_stub_environment(root)
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TMP_ROOT": str(tmp_root),
                    "KEEP_TMP": "1",
                    "ITERATIONS": "1",
                    "RUN_QUALITY_GATES": "0",
                    "TARGET_REPOS_FILE": str(repos_file),
                    "ACP_RUNTIME_PROVIDER": "qwen-code",
                    "ACP_QWEN_CMD": str(tools_dir / "qwen"),
                    "ACP_STUB_SOURCE": str(root / "acp-stub.sh"),
                    "ACP_STUB_TARGET_REPO": str(target_repo),
                    "ACP_STUB_RUN_HISTORY_MODE": "items",
                    "ACP_STUB_RUN_HISTORY_STATUS": "running",
                    "ACP_APPLY_TIMEOUTS_VIA_API": "0",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(FULL_RUN_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=120,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)

            summary_path = tmp_root / "session-summary.md"
            self.assertTrue(summary_path.exists(), msg="session-summary.md is missing")
            summary = summary_path.read_text(encoding="utf-8")
            self.assertEqual("failed", parse_summary_scalar(summary, "result"))
            self.assertEqual("infra_incomplete_cycle", parse_summary_scalar(summary, "failure_reason"))
            self.assertNotEqual("0", parse_summary_scalar(summary, "running_runs_detected"))

    def test_full_run_non_zero_pipeline_exit_keeps_pipeline_failure_reason(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo, tmp_root, repos_file = create_full_run_stub_environment(root)
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TMP_ROOT": str(tmp_root),
                    "KEEP_TMP": "1",
                    "ITERATIONS": "1",
                    "RUN_QUALITY_GATES": "0",
                    "TARGET_REPOS_FILE": str(repos_file),
                    "ACP_RUNTIME_PROVIDER": "qwen-code",
                    "ACP_QWEN_CMD": str(tools_dir / "qwen"),
                    "ACP_STUB_SOURCE": str(root / "acp-stub.sh"),
                    "ACP_STUB_TARGET_REPO": str(target_repo),
                    "ACP_STUB_RUN_FORCE_EXIT_CODE": "7",
                    "ACP_APPLY_TIMEOUTS_VIA_API": "0",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(FULL_RUN_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=120,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)

            summary_path = tmp_root / "session-summary.md"
            self.assertTrue(summary_path.exists(), msg="session-summary.md is missing")
            summary = summary_path.read_text(encoding="utf-8")
            failure_reason = parse_summary_scalar(summary, "failure_reason")
            self.assertIn("pipeline command failed for runtime=", failure_reason)
            self.assertNotIn("missing run_id", failure_reason)

    def test_full_run_marks_signal_termination(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo, tmp_root, repos_file = create_full_run_stub_environment(root)
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TMP_ROOT": str(tmp_root),
                    "KEEP_TMP": "1",
                    "ITERATIONS": "1",
                    "RUN_QUALITY_GATES": "0",
                    "TARGET_REPOS_FILE": str(repos_file),
                    "ACP_RUNTIME_PROVIDER": "qwen-code",
                    "ACP_QWEN_CMD": str(tools_dir / "qwen"),
                    "ACP_STUB_SOURCE": str(root / "acp-stub.sh"),
                    "ACP_STUB_TARGET_REPO": str(target_repo),
                    "ACP_STUB_RUN_SLEEP": "30",
                    "ACP_APPLY_TIMEOUTS_VIA_API": "0",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )

            proc = subprocess.Popen(
                [str(FULL_RUN_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                preexec_fn=os.setsid,
            )
            try:
                log_path = tmp_root / "full-run.log"
                deadline = time.time() + 20
                while time.time() < deadline:
                    if log_path.exists():
                        body = log_path.read_text(encoding="utf-8", errors="ignore")
                        if "bootstrap workspace in tmp" in body:
                            break
                    time.sleep(0.1)
                try:
                    os.killpg(proc.pid, signal.SIGTERM)
                except (PermissionError, ProcessLookupError):
                    proc.terminate()
                proc.wait(timeout=40)
            finally:
                if proc.poll() is None:
                    proc.kill()
                    proc.wait(timeout=10)

            self.assertNotEqual(proc.returncode, 0, msg="full-run script unexpectedly succeeded after SIGTERM")
            summary_path = tmp_root / "session-summary.md"
            self.assertTrue(summary_path.exists(), msg="session-summary.md is missing")
            summary = summary_path.read_text(encoding="utf-8")
            self.assertEqual("failed", parse_summary_scalar(summary, "result"))
            self.assertEqual("infra_signal_terminated", parse_summary_scalar(summary, "failure_reason"))
            self.assertIn(parse_summary_scalar(summary, "termination_signal"), {"TERM", "SIGTERM"})


def create_batch_stub_environment(root: Path) -> tuple[Path, Path, Path]:
    provenarch_root = root / "batch-root"
    tools_dir = root / "tools"
    target_repo = root / "target-repo"
    scripts_dir = provenarch_root / "scripts"
    reports_py = scripts_dir / "e2e_batch_report.py"

    write_text(target_repo / "README.md", "# target\n")
    scripts_dir.mkdir(parents=True, exist_ok=True)

    write_text(
        provenarch_root / "Makefile",
        textwrap.dedent(
            """\
            .PHONY: contracts test lint build
            contracts test lint build:
            \t@:
            """
        ),
    )
    write_text(
        scripts_dir / "full-run-ai-advent.sh",
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            set -euo pipefail
            : "${TMP_ROOT:?TMP_ROOT is required}"
            mkdir -p "$TMP_ROOT/arch-workspace/reports/as-is" "$TMP_ROOT/arch-workspace/reports/findings" "$TMP_ROOT/arch-workspace/reports/coverage" "$TMP_ROOT/arch-workspace/reports/taskruns" "$TMP_ROOT/snapshots"
            cat >"$TMP_ROOT/session-summary.md" <<'MD'
            # ProvenArch Full Run Session Summary

            - result: passed
            - quality_gates: passed
            - expected_runs: 4
            - completed_runs: 1
            - expected_headless_runs: 2
            - completed_headless_runs: 1
            - running_runs_detected: 0
            - termination_signal: none

            ## API Simulation
            - status: succeeded
            MD
            runtime_provider="${ACP_RUNTIME_PROVIDER:-qwen-code}"
            printf '1\theadless\t%s\tinit\trun-stub\tsucceeded\t5\t1\t1\t1\t1\t1\t0\t%s@test\t%s\t%s\n' \
              "$runtime_provider" "$runtime_provider" "$TMP_ROOT/arch-workspace/reports/taskruns/run-stub-quality.json" "$TMP_ROOT/logs/stub.log" >"$TMP_ROOT/run-results.tsv"
            cat >"$TMP_ROOT/arch-workspace/reports/taskruns/run-stub-quality.json" <<'JSON'
            {"status":"succeeded","totals":{"signal_score":5,"changeset_ops":1,"findings_added":1,"questions_count":1,"coverage_observed":1,"coverage_missing":1,"warnings_count":0},"runtime_versions":["qwen-code@test"],"steps":[{"step_id":"init.step1.collect","runtime_name":"qwen-code","runtime_version":"test","domain_id":"domain-main"}]}
            JSON
            cat >"$TMP_ROOT/arch-workspace/reports/as-is/overview.md" <<'MD'
            # Overview
            - services: 1
            MD
            cat >"$TMP_ROOT/arch-workspace/reports/findings/findings.md" <<'MD'
            # Findings
            ## Finding
            - Severity: medium
            - Description: stub
            MD
            cat >"$TMP_ROOT/arch-workspace/reports/coverage/summary.md" <<'MD'
            # Coverage
            ## Missing
            - owner mappings
            MD
            cat >"$TMP_ROOT/arch-workspace/reports/coverage/open-questions.md" <<'MD'
            # Open Questions
            - `q.stub` stub?
            MD
            exit 0
            """
        ),
        mode=0o755,
    )
    write_text(
        scripts_dir / "frontend-live-e2e.sh",
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            set -euo pipefail
            : "${OUTPUT_DIR:?OUTPUT_DIR is required}"
            mkdir -p "$OUTPUT_DIR"
            cat >"$OUTPUT_DIR/frontend-e2e-result.json" <<'JSON'
            {"status":"passed","runtime_provider":"stub"}
            JSON
            exit 0
            """
        ),
        mode=0o755,
    )
    write_text(
        reports_py,
        textwrap.dedent(
            """\
            #!/usr/bin/env python3
            import argparse
            from pathlib import Path

            parser = argparse.ArgumentParser()
            parser.add_argument("--batch-id", required=True)
            parser.add_argument("--batch-root", required=True)
            parser.add_argument("--reports-root", required=True)
            args = parser.parse_args()

            reports_root = Path(args.reports_root)
            reports_root.mkdir(parents=True, exist_ok=True)
            run_matrix_tsv = reports_root / f"run_matrix_{args.batch_id}.tsv"
            run_matrix_tsv.write_text(
                "provider\\trun\\thard_pass\\treliability\\tcontract\\tanalysis\\ttotal\\tverdict\\tinit_signal\\trefresh_signal\\trefresh_findings\\trefresh_questions\\trefresh_cov_missing\\tartifact_source\\tsemantic_hard_fail\\tfailure_class\\truntime_parse\\trunner_unavailable\\tinfra_signal_terminated\\tinfra_incomplete_cycle\\tsummary_missing\\toff_topic_hits\\tissues\\n"
                "qwen-code\\t1\\t0\\t10\\t10\\t10\\t30\\tPoor\\t0\\t0\\t0\\t0\\t0\\tsnapshot\\t0\\tinfra_incomplete_cycle\\t0\\t0\\t0\\t1\\t0\\t0\\treliability:infra-incomplete-cycle\\n",
                encoding="utf-8",
            )
            run_matrix_md = reports_root / f"run_matrix_{args.batch_id}.md"
            run_matrix_md.write_text("# Run Matrix\\n", encoding="utf-8")
            frontend_md = reports_root / f"frontend_e2e_matrix_{args.batch_id}.md"
            frontend_md.write_text("# Frontend Matrix\\n", encoding="utf-8")
            quality_md = reports_root / f"quality_report_{args.batch_id}.md"
            quality_md.write_text("# Quality\\n", encoding="utf-8")

            print(str(run_matrix_md))
            print(str(frontend_md))
            print(str(quality_md))
            print(str(run_matrix_tsv))
            """
        ),
        mode=0o755,
    )

    write_text(
        tools_dir / "git",
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            set -euo pipefail
            if [[ "$*" == *"rev-parse HEAD"* ]]; then
              echo "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
              exit 0
            fi
            if [[ "$*" == *"rev-parse --abbrev-ref HEAD"* ]]; then
              echo "main"
              exit 0
            fi
            echo "git-stub"
            exit 0
            """
        ),
        mode=0o755,
    )
    write_text(
        tools_dir / "npm",
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            set -euo pipefail
            exit 0
            """
        ),
        mode=0o755,
    )
    write_text(
        tools_dir / "claude",
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            if [[ "${1:-}" == "--version" ]]; then
              echo "claude 2.1.85"
              exit 0
            fi
            exit 0
            """
        ),
        mode=0o755,
    )
    write_text(
        tools_dir / "qwen",
        textwrap.dedent(
            """\
            #!/usr/bin/env bash
            if [[ "${1:-}" == "--version" ]]; then
              echo "qwen 1.0.0"
              exit 0
            fi
            exit 0
            """
        ),
        mode=0o755,
    )

    return provenarch_root, tools_dir, target_repo


class BatchPostRunValidationIntegrationTests(unittest.TestCase):
    def test_batch_marks_incomplete_cycle_even_when_full_run_exit_zero(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-stability-test",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)

            classification_path = e2e_tmp_root / "runs" / "batch-stability-test" / "backend-run-classifications.tsv"
            self.assertTrue(classification_path.exists(), msg="backend-run-classifications.tsv is missing")
            rows = [line for line in classification_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertGreaterEqual(len(rows), 2, msg="classification file must contain header + rows")
            self.assertTrue(
                any("\tinfra_incomplete_cycle\t" in line for line in rows[1:]),
                msg="expected infra_incomplete_cycle classification for incomplete run results",
            )
            self.assertIn("infra_incomplete_cycle=", result.stderr)

    def test_batch_records_precheck_failed_when_dod_precheck_fails(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            write_text(
                provenarch_root / "Makefile",
                textwrap.dedent(
                    """\
                    .PHONY: contracts test lint build
                    contracts test lint build:
                    \t@echo "forced precheck fail" >&2
                    \t@exit 2
                    """
                ),
            )
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-precheck-fail-test",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            classification_path = e2e_tmp_root / "runs" / "batch-precheck-fail-test" / "backend-run-classifications.tsv"
            self.assertTrue(classification_path.exists(), msg="backend-run-classifications.tsv is missing")
            rows = [line for line in classification_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertGreaterEqual(len(rows), 11, msg="expected 10 precheck_failed rows + header")
            self.assertTrue(
                all("\tprecheck_failed\t" in line for line in rows[1:]),
                msg="expected all classification rows to be precheck_failed",
            )
            self.assertIn("precheck_failed=", result.stderr)

    def test_batch_precheck_failed_respects_shard_selection(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            write_text(
                provenarch_root / "Makefile",
                textwrap.dedent(
                    """\
                    .PHONY: contracts test lint build
                    contracts test lint build:
                    \t@echo "forced precheck fail for shard selection test" >&2
                    \t@exit 2
                    """
                ),
            )
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-precheck-shard-selection",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "BATCH_PROVIDER_FILTER": "qwen-code",
                    "BATCH_RUN_SELECTION": "2,4",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            classification_path = e2e_tmp_root / "runs" / "batch-precheck-shard-selection" / "backend-run-classifications.tsv"
            self.assertTrue(classification_path.exists(), msg="backend-run-classifications.tsv is missing")
            rows = [line for line in classification_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertEqual(len(rows), 3, msg="expected header + 2 shard rows")
            payload_rows = [line.split("\t") for line in rows[1:]]
            self.assertEqual({"qwen-code"}, {item[0] for item in payload_rows})
            self.assertEqual({"2", "4"}, {item[1] for item in payload_rows})
            self.assertTrue(all(item[2] == "precheck_failed" for item in payload_rows))

    def test_batch_precheck_ignores_timeout_tuning_env(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            write_text(
                provenarch_root / "Makefile",
                textwrap.dedent(
                    """\
                    .PHONY: contracts test lint build
                    contracts test lint build:
                    \t@if [ -n "$$ACP_RUNTIME_STEP_TIMEOUT_SEC$$ACP_PIPELINE_TIMEOUT_SEC$$ACP_API_INIT_TIMEOUT_SEC$$READY_TIMEOUT_SEC$$UI_E2E_INIT_TIMEOUT_SEC$$UI_E2E_CANCEL_TIMEOUT_SEC" ]; then \
                    \t\techo "timeout env leaked into precheck" >&2; \
                    \t\texit 3; \
                    \tfi
                    """
                ),
            )
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-precheck-timeout-env-isolation",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "ACP_RUNTIME_STEP_TIMEOUT_SEC": "2700",
                    "ACP_PIPELINE_TIMEOUT_SEC": "3600",
                    "ACP_API_INIT_TIMEOUT_SEC": "180",
                    "READY_TIMEOUT_SEC": "99",
                    "UI_E2E_INIT_TIMEOUT_SEC": "100",
                    "UI_E2E_CANCEL_TIMEOUT_SEC": "100",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            classification_path = (
                e2e_tmp_root / "runs" / "batch-precheck-timeout-env-isolation" / "backend-run-classifications.tsv"
            )
            self.assertTrue(classification_path.exists(), msg="backend-run-classifications.tsv is missing")
            rows = [line for line in classification_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertGreaterEqual(len(rows), 2, msg="classification file must contain header + rows")
            self.assertFalse(
                any("\tprecheck_failed\t" in line for line in rows[1:]),
                msg="expected timeout env to be isolated from precheck",
            )
            self.assertTrue(
                any("\tinfra_incomplete_cycle\t" in line for line in rows[1:]),
                msg="expected non-precheck run classification rows",
            )

    def test_batch_frontend_auto_mode_skips_when_run1_not_selected(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-frontend-auto-skip",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "BATCH_PROVIDER_FILTER": "claude-code",
                    "BATCH_RUN_SELECTION": "2",
                    "BATCH_FRONTEND_MODE": "auto",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            frontend_result_path = (
                e2e_tmp_root / "runs" / "batch-frontend-auto-skip" / "frontend" / "claude-code" / "frontend-e2e-result.json"
            )
            self.assertTrue(frontend_result_path.exists(), msg="frontend-e2e-result.json is missing")
            frontend_result = json.loads(frontend_result_path.read_text(encoding="utf-8"))
            self.assertEqual("skipped", frontend_result.get("status"))
            self.assertIn("frontend_mode_auto", str(frontend_result.get("reason", "")))

            cancel_result_path = (
                e2e_tmp_root
                / "runs"
                / "batch-frontend-auto-skip"
                / "frontend-cancel"
                / "claude-code"
                / "frontend-cancel-result.json"
            )
            self.assertTrue(cancel_result_path.exists(), msg="frontend-cancel-result.json is missing")
            cancel_result = json.loads(cancel_result_path.read_text(encoding="utf-8"))
            self.assertEqual("skipped", cancel_result.get("status"))
            self.assertIn("frontend_mode_auto", str(cancel_result.get("reason", "")))
            self.assertIn("frontend_failed=0", result.stderr)
            self.assertIn("frontend_cancel_skipped=1", result.stderr)

    def test_batch_rejects_unknown_provider_filter(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-invalid-provider-filter",
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "BATCH_PROVIDER_FILTER": "qwen-code,unknown-provider",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            self.assertIn("unsupported provider", result.stderr)

    def test_batch_provider_filter_does_not_require_unselected_runtime_binary(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-provider-filter-runtime-binary",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "missing-claude-bin",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "BATCH_PROVIDER_FILTER": "qwen-code",
                    "BATCH_RUN_SELECTION": "1",
                    "BATCH_SKIP_PRECHECK": "1",
                    "BATCH_FRONTEND_MODE": "never",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            self.assertNotIn("required command is unavailable: missing-claude-bin", result.stderr)
            classification_path = (
                e2e_tmp_root / "runs" / "batch-provider-filter-runtime-binary" / "backend-run-classifications.tsv"
            )
            self.assertTrue(classification_path.exists(), msg="backend-run-classifications.tsv is missing")
            rows = [line for line in classification_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertEqual(len(rows), 2, msg="expected header + one run row")
            payload = rows[1].split("\t")
            self.assertEqual("qwen-code", payload[0])
            self.assertEqual("1", payload[1])

    def test_batch_rejects_out_of_bounds_run_selection(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-invalid-run-selection",
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "BATCH_RUN_SELECTION": "1-6",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            self.assertIn("run index out of bounds", result.stderr)

    def test_batch_skip_precheck_bypasses_failing_makefile(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            write_text(
                provenarch_root / "Makefile",
                textwrap.dedent(
                    """\
                    .PHONY: contracts test lint build
                    contracts test lint build:
                    \t@echo "forced precheck fail that must be skipped" >&2
                    \t@exit 7
                    """
                ),
            )
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-skip-precheck",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "BATCH_PROVIDER_FILTER": "qwen-code",
                    "BATCH_RUN_SELECTION": "1",
                    "BATCH_SKIP_PRECHECK": "1",
                    "BATCH_FRONTEND_MODE": "never",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            self.assertIn("skipping DoD/UI precheck", result.stderr)
            self.assertNotIn("batch precheck failed", result.stderr)
            self.assertIn("precheck_failed=0", result.stderr)

    def test_batch_frontend_never_mode_marks_both_smokes_skipped(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-frontend-never-skip",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "BATCH_PROVIDER_FILTER": "claude-code",
                    "BATCH_RUN_SELECTION": "1",
                    "BATCH_FRONTEND_MODE": "never",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            frontend_result_path = (
                e2e_tmp_root / "runs" / "batch-frontend-never-skip" / "frontend" / "claude-code" / "frontend-e2e-result.json"
            )
            cancel_result_path = (
                e2e_tmp_root
                / "runs"
                / "batch-frontend-never-skip"
                / "frontend-cancel"
                / "claude-code"
                / "frontend-cancel-result.json"
            )
            self.assertTrue(frontend_result_path.exists(), msg="frontend-e2e-result.json is missing")
            self.assertTrue(cancel_result_path.exists(), msg="frontend-cancel-result.json is missing")
            frontend_result = json.loads(frontend_result_path.read_text(encoding="utf-8"))
            cancel_result = json.loads(cancel_result_path.read_text(encoding="utf-8"))
            self.assertEqual("skipped", frontend_result.get("status"))
            self.assertEqual("skipped", cancel_result.get("status"))
            self.assertIn("frontend_mode_never", str(frontend_result.get("reason", "")))
            self.assertIn("frontend_mode_never", str(cancel_result.get("reason", "")))
            self.assertIn("frontend_failed=0", result.stderr)
            self.assertIn("frontend_cancel_skipped=1", result.stderr)

    def test_batch_classifies_signal_terminated_without_runner_unavailable(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            write_text(
                provenarch_root / "scripts/full-run-ai-advent.sh",
                textwrap.dedent(
                    """\
                    #!/usr/bin/env bash
                    set -euo pipefail
                    : "${TMP_ROOT:?TMP_ROOT is required}"
                    mkdir -p "$TMP_ROOT"
                    cat >"$TMP_ROOT/session-summary.md" <<'MD'
                    # ProvenArch Full Run Session Summary

                    - result: failed
                    - quality_gates: passed
                    - expected_runs: 4
                    - completed_runs: 4
                    - expected_headless_runs: 2
                    - completed_headless_runs: 2
                    - running_runs_detected: 0
                    - failure_reason: infra_signal_terminated
                    - termination_signal: TERM

                    ## API Simulation
                    - status: succeeded
                    MD
                    : >"$TMP_ROOT/run-results.tsv"
                    echo "signal terminated" >"$TMP_ROOT/full-run.log"
                    exit 1
                    """
                ),
                mode=0o755,
            )
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-signal-term-test",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            classification_path = e2e_tmp_root / "runs" / "batch-signal-term-test" / "backend-run-classifications.tsv"
            self.assertTrue(classification_path.exists(), msg="backend-run-classifications.tsv is missing")
            rows = [line for line in classification_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertGreaterEqual(len(rows), 2, msg="classification file must contain header + rows")
            self.assertTrue(
                any("\tinfra_signal_terminated\t" in line for line in rows[1:]),
                msg="expected infra_signal_terminated classification",
            )
            self.assertFalse(
                any("\trunner_unavailable\t" in line for line in rows[1:]),
                msg="runner_unavailable must not be reported for signal termination scenario",
            )

    def test_batch_prioritizes_runtime_parse_over_infra_incomplete_cycle(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            provenarch_root, tools_dir, target_repo = create_batch_stub_environment(root)
            write_text(
                provenarch_root / "scripts/full-run-ai-advent.sh",
                textwrap.dedent(
                    """\
                    #!/usr/bin/env bash
                    set -euo pipefail
                    : "${TMP_ROOT:?TMP_ROOT is required}"
                    provider="${ACP_RUNTIME_PROVIDER:-qwen-code}"
                    mkdir -p "$TMP_ROOT/logs" "$TMP_ROOT/arch-workspace/reports/taskruns/raw"
                    cat >"$TMP_ROOT/session-summary.md" <<'MD'
                    # ProvenArch Full Run Session Summary

                    - result: failed
                    - quality_gates: passed
                    - expected_runs: 4
                    - completed_runs: 1
                    - expected_headless_runs: 2
                    - completed_headless_runs: 0
                    - running_runs_detected: 0
                    - failure_reason: infra_incomplete_cycle
                    - termination_signal: none

                    ## API Simulation
                    - status: succeeded
                    MD
                    : >"$TMP_ROOT/run-results.tsv"
                    raw_path="$TMP_ROOT/arch-workspace/reports/taskruns/raw/iter1-stdout.log"
                    echo '{"not":"taskresult"}' >"$raw_path"
                    cat >"$TMP_ROOT/logs/run-iter1-headless-${provider}-refresh.log" <<LOG
                    run failed: error_code=runner_parse_failed parse_stage=schema raw_output=${raw_path}
                    LOG
                    echo "runner_parse_failed for ${provider}" >"$TMP_ROOT/full-run.log"
                    exit 1
                    """
                ),
                mode=0o755,
            )
            e2e_tmp_root = root / "e2e"
            env = os.environ.copy()
            env.update(
                {
                    "PROVENARCH_ROOT": str(provenarch_root),
                    "TARGET_REPO": str(target_repo),
                    "BATCH_ID": "batch-runtime-parse-priority-test",
                    "E2E_TMP_ROOT": str(e2e_tmp_root),
                    "ACP_CLAUDE_CMD_BIN": "claude",
                    "ACP_QWEN_CMD_BIN": "qwen",
                    "PATH": f"{tools_dir}:{env.get('PATH', '')}",
                }
            )
            result = subprocess.run(
                [str(BATCH_SCRIPT)],
                env=env,
                cwd=str(REPO_ROOT),
                check=False,
                capture_output=True,
                text=True,
                timeout=180,
            )
            self.assertNotEqual(result.returncode, 0, msg=result.stdout + "\n" + result.stderr)
            classification_path = (
                e2e_tmp_root / "runs" / "batch-runtime-parse-priority-test" / "backend-run-classifications.tsv"
            )
            self.assertTrue(classification_path.exists(), msg="backend-run-classifications.tsv is missing")
            rows = [line for line in classification_path.read_text(encoding="utf-8").splitlines() if line.strip()]
            self.assertGreaterEqual(len(rows), 2, msg="classification file must contain header + rows")
            failure_classes = [line.split("\t")[2] for line in rows[1:]]
            self.assertTrue(
                any(item == "runtime_parse" for item in failure_classes),
                msg="expected runtime_parse classification",
            )
            self.assertFalse(
                any(item == "infra_incomplete_cycle" for item in failure_classes),
                msg="runtime_parse must take priority over infra_incomplete_cycle",
            )


if __name__ == "__main__":
    unittest.main()
