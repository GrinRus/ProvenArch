import json
import os
import stat
import subprocess
import tempfile
import textwrap
import time
import unittest
from unittest import mock
from pathlib import Path


class MatrixReleaseContractTest(unittest.TestCase):
    SAFE_ENV_KEYS = (
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
    )
    STRIPPED_ENV_PREFIXES = (
        "ACP_",
        "BATCH_",
        "E2E_",
        "MATRIX_",
        "UI_E2E_",
    )
    STRIPPED_ENV_KEYS = (
        "RUN_COUNT",
        "TARGET_REPOS_FILE",
        "PROFILE_ID",
        "PROFILE_SOURCE_KIND",
        "PROFILE_EXPECTED_REPO_COUNT",
        "PROFILE_REPOS_FILE",
        "SWEEP_ID",
    )

    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.matrix_driver = cls.repo_root / "scripts" / "full-run-batch-matrix.sh"

    def setUp(self) -> None:
        self.tmpdir = tempfile.TemporaryDirectory()
        self.tmp_root = Path(self.tmpdir.name)
        self.e2e_tmp_root = self.tmp_root / "e2e"
        self.sentinel_path = self.tmp_root / "batch-calls.log"
        self.timeout_sentinel_path = self.tmp_root / "batch-timeouts.jsonl"
        self.batch_script = self.tmp_root / "dummy-batch.sh"
        self._write_dummy_batch_script()
        self.profile_repos_files = self._prepare_repos_files()

    def tearDown(self) -> None:
        self.tmpdir.cleanup()

    def _prepare_repos_files(self) -> dict[str, Path]:
        repos_root = self.tmp_root / "repos"
        repos_root.mkdir(parents=True, exist_ok=True)

        path_single = repos_root / "path-single"
        path_multi_a = repos_root / "path-multi-a"
        path_multi_b = repos_root / "path-multi-b"
        for path in (path_single, path_multi_a, path_multi_b):
            path.mkdir(parents=True, exist_ok=True)

        matrix_inputs = self.tmp_root / "matrix-inputs"
        matrix_inputs.mkdir(parents=True, exist_ok=True)

        single_path_file = matrix_inputs / "single-path.repos.yaml"
        single_path_file.write_text(
            textwrap.dedent(
                f"""\
                repos:
                  - name: single-path-repo
                    path: {path_single}
                    ref: local-ref
                """
            ),
            encoding="utf-8",
        )

        multi_path_file = matrix_inputs / "multi-path.repos.yaml"
        multi_path_file.write_text(
            textwrap.dedent(
                f"""\
                repos:
                  - name: multi-path-a
                    path: {path_multi_a}
                    ref: local-ref-a
                  - name: multi-path-b
                    path: {path_multi_b}
                    ref: local-ref-b
                """
            ),
            encoding="utf-8",
        )

        single_git_file = matrix_inputs / "single-git.repos.yaml"
        single_git_file.write_text(
            textwrap.dedent(
                """\
                repos:
                  - name: single-git-repo
                    git_url: https://example.invalid/single.git
                    ref: 1111111111111111111111111111111111111111
                """
            ),
            encoding="utf-8",
        )

        multi_git_file = matrix_inputs / "multi-git.repos.yaml"
        multi_git_file.write_text(
            textwrap.dedent(
                """\
                repos:
                  - name: multi-git-a
                    git_url: https://example.invalid/a.git
                    ref: 2222222222222222222222222222222222222222
                  - name: multi-git-b
                    git_url: https://example.invalid/b.git
                    ref: 3333333333333333333333333333333333333333
                """
            ),
            encoding="utf-8",
        )

        return {
            "single-path": single_path_file,
            "single-git_url": single_git_file,
            "multi-path": multi_path_file,
            "multi-git_url": multi_git_file,
        }

    def _create_git_repo(self, name: str) -> tuple[Path, str]:
        repo = self.tmp_root / "repos" / name
        repo.mkdir(parents=True, exist_ok=True)
        subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True, text=True)
        subprocess.run(["git", "config", "user.email", "test@example.invalid"], cwd=repo, check=True)
        subprocess.run(["git", "config", "user.name", "Test User"], cwd=repo, check=True)
        (repo / "README.md").write_text("# repo\n", encoding="utf-8")
        subprocess.run(["git", "add", "README.md"], cwd=repo, check=True)
        subprocess.run(["git", "commit", "-m", "init"], cwd=repo, check=True, capture_output=True, text=True)
        head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
        return repo, head

    def _write_dummy_batch_script(self) -> None:
        self.batch_script.write_text(
            textwrap.dedent(
                """\
                #!/usr/bin/env bash
                set -Eeuo pipefail

                : "${REPORTS_ROOT:?}" "${BATCH_ID:?}" "${BATCH_ROOT:?}" "${SWEEP_ID:?}" "${PROFILE_ID:?}"

                if [[ -n "${MATRIX_TEST_SENTINEL:-}" ]]; then
                  printf '%s\\n' "${PROFILE_ID}/${SWEEP_ID}" >> "${MATRIX_TEST_SENTINEL}"
                fi
                if [[ -n "${MATRIX_TEST_TIMEOUT_SENTINEL:-}" ]]; then
                  python3 - <<'PY' >> "${MATRIX_TEST_TIMEOUT_SENTINEL}"
                import json
                import os
                payload = {
                    "profile_id": os.environ.get("PROFILE_ID", ""),
                    "sweep_id": os.environ.get("SWEEP_ID", ""),
                    "step_timeout_sec": os.environ.get("ACP_RUNTIME_STEP_TIMEOUT_SEC", ""),
                    "pipeline_timeout_sec": os.environ.get("ACP_PIPELINE_TIMEOUT_SEC", ""),
                    "ui_init_poll_timeout_sec": os.environ.get("ACP_UI_INIT_POLL_TIMEOUT_SEC", ""),
                }
                print(json.dumps(payload, ensure_ascii=True))
                PY
                fi

                if [[ "${MATRIX_TEST_DRAIN_STDIN:-0}" == "1" ]]; then
                  cat >/dev/null || true
                fi

                if [[ "${MATRIX_TEST_FAIL_BEFORE_REPORT:-0}" == "1" ]]; then
                  mkdir -p "${BATCH_ROOT}/qwen-code/run1"
                  cat > "${BATCH_ROOT}/qwen-code/run1/run-status.env" <<'EOF'
                provider=qwen-code
                run_index=1
                state=running
                process_exit=
                termination_signal=none
                EOF
                  exit 1
                fi

                mkdir -p "${REPORTS_ROOT}" "${BATCH_ROOT}/qwen-code/run1/reports/taskruns"

                if [[ "${MATRIX_TEST_SLEEP_SEC:-0}" != "0" ]]; then
                  sleep "${MATRIX_TEST_SLEEP_SEC}"
                fi

                provider_filter="${BATCH_PROVIDER_FILTER:-all}"
                if [[ -z "${provider_filter}" || "${provider_filter}" == "all" ]]; then
                  selected_providers=(qwen-code claude-code codex-code)
                else
                  IFS=',' read -r -a selected_providers <<< "${provider_filter}"
                fi
                provider_count="${#selected_providers[@]}"

                run_matrix_tsv="${REPORTS_ROOT}/run_matrix_${BATCH_ID}.tsv"
                run_matrix_md="${REPORTS_ROOT}/run_matrix_${BATCH_ID}.md"
                quality_report_md="${REPORTS_ROOT}/quality_report_${BATCH_ID}.md"
                frontend_matrix_md="${REPORTS_ROOT}/frontend_e2e_matrix_${BATCH_ID}.md"
                frontend_cancel_matrix_md="${REPORTS_ROOT}/frontend_cancel_e2e_matrix_${BATCH_ID}.md"
                blackbox_steps_jsonl="${REPORTS_ROOT}/blackbox_e2e_steps_${BATCH_ID}.jsonl"
                blackbox_steps_md="${REPORTS_ROOT}/blackbox_e2e_steps_${BATCH_ID}.md"

                {
                  printf 'provider\\trun\\thard_pass\\truntime_contract_failed\\trunner_unavailable\\truntime_timeout\\tinfra_signal_terminated\\tinfra_incomplete_cycle\\tquality_gates_failed\\tsummary_missing\\tprecheck_failed\\tcancellation_like\\truntime_flow_failed\\tsemantic_hard_fail\\toff_topic_hits\\tartifact_source\\tissues\\n'
                  for provider in "${selected_providers[@]}"; do
                    for run_idx in $(seq 1 "${RUN_COUNT}"); do
                      if [[ "${MATRIX_TEST_RUNTIME_FLOW_FAILED:-0}" == "1" && "${PROFILE_ID}" == "single-path" && "${SWEEP_ID}" == "baseline" && "${provider}" == "qwen-code" && "$run_idx" -eq 1 ]]; then
                        printf '%s\\t%s\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t1\\t0\\t0\\tsnapshot\\treliability:runtime-flow-failed\\n' "${provider}" "${run_idx}"
                      elif [[ "${MATRIX_TEST_QWEN_BACKEND_FAILURE:-0}" == "1" && "${provider}" == "qwen-code" && "$run_idx" -eq 1 ]]; then
                        printf '%s\\t%s\\t0\\t0\\t1\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\tsnapshot\\treliability:runner-unavailable\\n' "${provider}" "${run_idx}"
                      else
                        printf '%s\\t%s\\t1\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\tsnapshot\\t-\\n' "${provider}" "${run_idx}"
                      fi
                    done
                  done
                } > "${run_matrix_tsv}"

                printf '# run matrix\\n' > "${run_matrix_md}"
                printf '# quality\\n' > "${quality_report_md}"
                python3 - "${blackbox_steps_jsonl}" "${blackbox_steps_md}" "${BATCH_ID}" "${PROFILE_ID}" "${SWEEP_ID}" <<'PY'
                import json
                import sys
                from pathlib import Path

                jsonl_path = Path(sys.argv[1])
                md_path = Path(sys.argv[2])
                batch_id = sys.argv[3]
                profile_id = sys.argv[4]
                sweep_id = sys.argv[5]
                record = {
                    "batch_id": batch_id,
                    "profile_id": profile_id,
                    "sweep_id": sweep_id,
                    "step_id": "batch.final",
                    "goal": "classify dummy batch",
                    "action": "write dummy batch reports",
                    "observed_evidence": [str(jsonl_path)],
                    "status": "passed",
                    "primary_classification": "none",
                    "evidence_paths": [str(jsonl_path)],
                    "next_decision": "return to matrix harness",
                }
                jsonl_path.write_text(json.dumps(record, ensure_ascii=True) + "\\n", encoding="utf-8")
                md_path.write_text(
                    "# Black-Box E2E Steps\\n\\n"
                    "| step_id | status | classification | goal | action | observed_evidence | next_decision |\\n"
                    "|---|---|---|---|---|---|---|\\n"
                    "| batch.final | passed | none | classify dummy batch | write dummy batch reports | "
                    f"{jsonl_path} | return to matrix harness |\\n",
                    encoding="utf-8",
                )
                PY

                if [[ "${MATRIX_TEST_RAW_METADATA:-0}" == "1" ]]; then
                  raw_dir="${BATCH_ROOT}/qwen-code/run1/arch-workspace/reports/taskruns/raw"
                  mkdir -p "${raw_dir}"
                  cat > "${raw_dir}/run-codex-init-step0-task-codex-codex-code-meta.json" <<'EOF'
                {
                  "generated_at": "2026-04-25T00:00:00Z",
                  "provider": "codex-code",
                  "command_family": "codex-code",
                  "diagnostics_set": true,
                  "task": {
                    "task_id": "task-codex",
                    "run_id": "run-codex",
                    "step_id": "init.step0.constitution",
                    "workspace": "/tmp/workspace",
                    "shard_id": "single",
                    "repo_scope": "repo-a",
                    "repo_scopes": ["repo-a"],
                    "path_scopes": []
                  },
                  "stdout": {
                    "path": "/tmp/workspace/reports/taskruns/raw/stdout.log",
                    "relative_path": "reports/taskruns/raw/stdout.log",
                    "bytes": 12,
                    "stored_bytes": 12,
                    "sha256": "abc",
                    "truncated": false
                  },
                  "stderr": {
                    "path": "/tmp/workspace/reports/taskruns/raw/stderr.log",
                    "relative_path": "reports/taskruns/raw/stderr.log",
                    "bytes": 34,
                    "stored_bytes": 34,
                    "sha256": "def",
                    "truncated": false
                  }
                }
                EOF
                fi

                {
                  printf '| provider | status | runs | reasons |\\n'
                  printf '|---|---|---|---|\\n'
                  for provider in "${selected_providers[@]}"; do
                    if [[ "${BATCH_FRONTEND_MODE:-}" == "never" ]]; then
                      printf '| %s | skipped | 0 | disabled=1 |\\n' "${provider}"
                    elif [[ "${MATRIX_TEST_QWEN_FRONTEND_SNAPSHOT_MISSING:-0}" == "1" && "${provider}" == "qwen-code" ]]; then
                      printf '| %s | skipped | 1 | snapshot_reports_missing=1 |\\n' "${provider}"
                    else
                      printf '| %s | passed | 1 | ok=1 |\\n' "${provider}"
                    fi
                  done
                } > "${frontend_matrix_md}"

                {
                  printf '| provider | status | runs | reasons |\\n'
                  printf '|---|---|---|---|\\n'
                  for provider in "${selected_providers[@]}"; do
                    if [[ "${BATCH_FRONTEND_CANCEL_MODE:-}" == "never" ]]; then
                      printf '| %s | skipped | 0 | disabled=1 |\\n' "${provider}"
                    else
                      printf '| %s | passed | 1 | ok=1 |\\n' "${provider}"
                    fi
                  done
                } > "${frontend_cancel_matrix_md}"

                shard_plan_path="${BATCH_ROOT}/qwen-code/run1/reports/taskruns/${BATCH_ID}-init-step1-collect-shard-plan.json"
                if [[ "${MATRIX_TEST_ARTIFACT_ERROR:-0}" == "1" && "${PROFILE_ID}" == "single-path" && "${SWEEP_ID}" == "parallel-default" ]]; then
                  printf '{invalid-json\\n' > "${shard_plan_path}"
                else
                  cat > "${shard_plan_path}" <<'EOF'
                {
                  "items": [
                    {
                      "shard_id": "shard-main",
                      "repo_scopes": ["repo-main"],
                      "path_scopes": ["services"]
                    }
                  ]
                }
                EOF
                fi

                if [[ "${MATRIX_TEST_RUNTIME_FLOW_FAILED:-0}" == "1" && "${PROFILE_ID}" == "single-path" && "${SWEEP_ID}" == "baseline" ]]; then
                  exit 1
                fi
                """
            ),
            encoding="utf-8",
        )
        self.batch_script.chmod(self.batch_script.stat().st_mode | stat.S_IXUSR)

    def _write_matrix_file(
        self,
        sweeps: list[str] | None,
        include_profiles: list[str] | None = None,
        timeout_profile: str | None = None,
    ) -> Path:
        ordered_profiles = ["single-path", "multi-git_url"]
        selected_profiles = include_profiles or ordered_profiles
        matrix_path = self.tmp_root / "matrix.yaml"

        sweep_lines: list[str] = []
        if sweeps is not None:
            for sweep in sweeps:
                if sweep == "baseline":
                    sweep_lines.append(
                        textwrap.dedent(
                            """\
                              - id: baseline
                                strategy: sequential
                                max_parallel_tasks: 1
                                failure_policy: best_effort
                                shard_discovery_mode: heuristics
                            """
                        )
                    )
                    continue
                if sweep == "parallel-default":
                    sweep_lines.append(
                        textwrap.dedent(
                            """\
                              - id: parallel-default
                                strategy: parallel
                                max_parallel_tasks: 4
                                failure_policy: best_effort
                                shard_discovery_mode: heuristics
                            """
                        )
                    )
                    continue
                sweep_lines.append(
                    textwrap.dedent(
                        f"""\
                          - id: {sweep}
                            strategy: sequential
                            max_parallel_tasks: 1
                            failure_policy: best_effort
                            shard_discovery_mode: heuristics
                        """
                    )
                )

        profile_lines: list[str] = []
        for profile_id in selected_profiles:
            repos_file = self.profile_repos_files[profile_id]
            if profile_id in {"single-path", "single-git_url"}:
                expected_repo_count = 1
            else:
                expected_repo_count = 2
            source_kind = "path" if profile_id in {"single-path", "multi-path"} else "git_url"
            profile_lines.append(
                textwrap.dedent(
                    f"""\
                      - id: {profile_id}
                        source_kind: {source_kind}
                        expected_repo_count: {expected_repo_count}
                        repos_file: {repos_file}
                    """
                )
            )

        matrix_payload = ["version: 1\n"]
        if timeout_profile:
            matrix_payload.append(f"timeout_profile: {timeout_profile}\n")
        if sweeps is not None:
            matrix_payload.append("sweeps:\n")
            matrix_payload.append("".join(sweep_lines))
        matrix_payload.append("profiles:\n")
        matrix_payload.append("".join(profile_lines))
        matrix_path.write_text("".join(matrix_payload), encoding="utf-8")
        return matrix_path

    def _run_matrix(
        self,
        matrix_file: Path,
        matrix_id: str,
        extra_env: dict[str, str] | None = None,
        release_mode: str = "1",
    ) -> subprocess.CompletedProcess[str]:
        env = self._build_subprocess_env(
            {
                "E2E_MATRIX_FILE": str(matrix_file),
                "BATCH_SCRIPT": str(self.batch_script),
                "MATRIX_ID": matrix_id,
                "E2E_MATRIX_RELEASE_MODE": release_mode,
                "ACP_CLAUDE_CMD_BIN": "true",
                "ACP_QWEN_CMD_BIN": "true",
                "ACP_CODEX_CMD_BIN": "true",
                "E2E_TMP_ROOT": str(self.e2e_tmp_root),
                "REPORTS_ROOT": str(self.e2e_tmp_root / "reports"),
                "MATRIX_ROOT": str(self.e2e_tmp_root / "matrix" / matrix_id),
                "MATRIX_TEST_SENTINEL": str(self.sentinel_path),
                "MATRIX_TEST_TIMEOUT_SENTINEL": str(self.timeout_sentinel_path),
                **(extra_env or {}),
            }
        )
        return subprocess.run(
            [str(self.matrix_driver)],
            cwd=self.repo_root,
            env=env,
            capture_output=True,
            text=True,
        )

    def _build_subprocess_env(self, owned_env: dict[str, str]) -> dict[str, str]:
        env = {
            key: value
            for key, value in os.environ.items()
            if key in self.SAFE_ENV_KEYS and value
        }
        for key in list(env):
            if any(key.startswith(prefix) for prefix in self.STRIPPED_ENV_PREFIXES):
                env.pop(key, None)
        for key in self.STRIPPED_ENV_KEYS:
            env.pop(key, None)
        env.update({key: value for key, value in owned_env.items() if value is not None})
        return env

    def _load_verdict(self, matrix_id: str) -> dict:
        verdict_path = self.e2e_tmp_root / "reports" / f"release_verdict_{matrix_id}.json"
        self.assertTrue(verdict_path.exists(), f"missing verdict file: {verdict_path}")
        return json.loads(verdict_path.read_text(encoding="utf-8"))

    def _load_blackbox_steps(self, matrix_id: str) -> list[dict]:
        steps_path = self.e2e_tmp_root / "reports" / f"blackbox_e2e_steps_{matrix_id}.jsonl"
        self.assertTrue(steps_path.exists(), f"missing black-box step report: {steps_path}")
        return [
            json.loads(line)
            for line in steps_path.read_text(encoding="utf-8").splitlines()
            if line.strip()
        ]

    def _load_yaml(self, path: Path) -> dict:
        try:
            import yaml  # type: ignore
        except Exception as exc:  # pragma: no cover - hard fail mirrors runtime requirement
            self.fail(f"PyYAML is required for parsing {path}: {exc}")
        payload = yaml.safe_load(path.read_text(encoding="utf-8"))
        self.assertIsInstance(payload, dict, f"expected YAML object in {path}")
        return payload

    def _assert_release_slice_shape(self, path: Path, expected_profiles: list[str]) -> None:
        payload = self._load_yaml(path)
        sweeps = payload.get("sweeps")
        self.assertEqual(
            [item.get("id") for item in sweeps],
            ["baseline", "parallel-default"],
            f"unexpected release sweeps in {path}",
        )
        profiles = payload.get("profiles")
        self.assertEqual([item.get("id") for item in profiles], expected_profiles)
        self.assertEqual(len(profiles), 2, f"release slice must keep exactly 2 concrete profiles: {path}")

        profile_ids = {item.get("id") for item in profiles}
        self.assertEqual(
            len([profile_id for profile_id in profile_ids if str(profile_id).startswith("single-")]),
            1,
            f"release slice must keep exactly one single-* profile: {path}",
        )
        self.assertEqual(
            len([profile_id for profile_id in profile_ids if str(profile_id).startswith("multi-")]),
            1,
            f"release slice must keep exactly one multi-* profile: {path}",
        )

    def _assert_non_release_slice_shape(self, path: Path, expected_profiles: list[str]) -> None:
        payload = self._load_yaml(path)
        self.assertNotIn("sweeps", payload, f"non-release slice should rely on implicit baseline: {path}")
        profiles = payload.get("profiles")
        self.assertEqual([item.get("id") for item in profiles], expected_profiles)

    def test_release_matrix_requires_parallel_default_sweep(self) -> None:
        matrix_file = self._write_matrix_file(["baseline"])
        matrix_id = "release-test-missing-parallel"
        result = self._run_matrix(matrix_file, matrix_id)
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing required ids: parallel-default", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when release sweeps are invalid")
        steps = self._load_blackbox_steps(matrix_id)
        self.assertEqual("matrix.plan", steps[-1]["step_id"])
        self.assertEqual("failed", steps[-1]["status"])
        self.assertEqual("matrix_plan_failed", steps[-1]["primary_classification"])

    def test_release_matrix_rejects_extra_sweep(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default", "extra-sweep"])
        matrix_id = "release-test-extra-sweep"
        result = self._run_matrix(matrix_file, matrix_id)
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("contains unsupported ids: extra-sweep", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when release sweeps contain extra ids")

    def test_release_matrix_requires_single_and_multi_profile_families(self) -> None:
        matrix_file = self._write_matrix_file(
            ["baseline", "parallel-default"],
            include_profiles=["single-path"],
        )
        matrix_id = "release-test-missing-profile"
        result = self._run_matrix(matrix_file, matrix_id)
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("exactly 2 profiles", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when release profile families are incomplete")

    def test_release_matrix_rejects_duplicate_single_family(self) -> None:
        matrix_file = self._write_matrix_file(
            ["baseline", "parallel-default"],
            include_profiles=["single-path", "single-git_url", "multi-path"],
        )
        matrix_id = "release-test-duplicate-family"
        result = self._run_matrix(matrix_file, matrix_id)
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("exactly 2 profiles", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when release profile families are duplicated")

    def test_release_matrix_requires_run_count_one(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default"])
        matrix_id = "release-test-run-count"
        result = self._run_matrix(matrix_file, matrix_id, extra_env={"RUN_COUNT": "2"})
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("requires RUN_COUNT=1", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when release RUN_COUNT is invalid")

    def test_valid_release_matrix_passes_with_release_contract(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default"])
        matrix_id = "release-test-valid"
        result = self._run_matrix(matrix_file, matrix_id)
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        self.assertEqual(verdict["release_state"], "RELEASE READY")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["contract_status"], "passed")
        self.assertEqual(release_contract["required_sweeps"], ["baseline", "parallel-default"])
        self.assertEqual(release_contract["observed_sweeps"], ["baseline", "parallel-default"])
        self.assertEqual(release_contract["required_profiles"], ["single-path", "multi-git_url"])
        self.assertEqual(release_contract["expected_profile_sweep_runs"], 4)
        self.assertEqual(release_contract["observed_profile_sweep_runs"], 4)
        self.assertEqual(release_contract["shard_plan_invariant_status"], "passed")

        calls = self.sentinel_path.read_text(encoding="utf-8").splitlines()
        self.assertEqual(len(calls), 4)

    def test_release_matrix_runs_all_combinations_when_child_batch_reads_stdin(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default"])
        matrix_id = "release-test-child-stdin"
        result = self._run_matrix(matrix_file, matrix_id, extra_env={"MATRIX_TEST_DRAIN_STDIN": "1"})
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        self.assertEqual(verdict["release_contract"]["observed_profile_sweep_runs"], 4)
        self.assertEqual(verdict["release_contract"]["observed_sweeps"], ["baseline", "parallel-default"])

        calls = self.sentinel_path.read_text(encoding="utf-8").splitlines()
        self.assertEqual(
            calls,
            [
                "single-path/baseline",
                "single-path/parallel-default",
                "multi-git_url/baseline",
                "multi-git_url/parallel-default",
            ],
        )

    def test_matrix_writes_blackbox_step_report_shape(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default"])
        matrix_id = "release-test-blackbox-steps"
        result = self._run_matrix(matrix_file, matrix_id)
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        steps = self._load_blackbox_steps(matrix_id)
        step_ids = [step["step_id"] for step in steps]
        self.assertIn("matrix.preflight", step_ids)
        self.assertIn("matrix.plan", step_ids)
        self.assertIn("matrix.profile.single-path.baseline", step_ids)
        self.assertIn("matrix.profile.multi-git-url.parallel-default", step_ids)
        self.assertEqual("matrix.verdict", step_ids[-1])

        required_keys = {
            "step_id",
            "goal",
            "action",
            "observed_evidence",
            "status",
            "primary_classification",
            "evidence_paths",
            "next_decision",
        }
        for step in steps:
            self.assertTrue(required_keys.issubset(step.keys()), step)
            self.assertIsInstance(step["observed_evidence"], list)
            self.assertIsInstance(step["evidence_paths"], list)
            self.assertIn(step["status"], {"passed", "failed", "skipped", "blocked"})

        batch_step_path = self.e2e_tmp_root / "reports" / f"blackbox_e2e_steps_{matrix_id}-single-path-baseline.jsonl"
        self.assertTrue(batch_step_path.exists(), f"missing batch black-box report: {batch_step_path}")
        batch_steps = [
            json.loads(line)
            for line in batch_step_path.read_text(encoding="utf-8").splitlines()
            if line.strip()
        ]
        self.assertEqual("batch.final", batch_steps[-1]["step_id"])

    def test_release_mode_blocks_artifact_error_invariant(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default"])
        matrix_id = "release-test-artifact-error"
        result = self._run_matrix(matrix_file, matrix_id, extra_env={"MATRIX_TEST_ARTIFACT_ERROR": "1"})
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("RELEASE BLOCKED", combined_output)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "FAIL")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["contract_status"], "failed")
        self.assertEqual(release_contract["shard_plan_invariant_status"], "failed")
        self.assertIn("artifact_error", release_contract["shard_plan_invariant_counts"])
        self.assertIn("not_compared", release_contract["shard_plan_invariant_counts"])
        self.assertTrue(
            any(
                rec.get("shard_plan_invariant") == "artifact_error" and rec.get("strict_status") == "failed"
                for rec in verdict.get("records", [])
            )
        )

    def test_non_release_allows_implicit_baseline_when_sweeps_omitted(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["single-path", "single-git_url", "multi-path", "multi-git_url"],
        )
        matrix_id = "matrix-test-implicit-baseline"
        result = self._run_matrix(matrix_file, matrix_id, release_mode="0")
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["mode"], "non-release")
        self.assertEqual(release_contract["required_sweeps"], [])
        self.assertEqual(release_contract["observed_sweeps"], ["baseline"])
        self.assertEqual(release_contract["contract_status"], "passed")
        self.assertEqual(release_contract["observed_profile_sweep_runs"], 4)
        self.assertTrue(
            any(
                rec.get("shard_plan_invariant") == "not_compared" and rec.get("strict_status") == "passed"
                for rec in verdict.get("records", [])
            )
        )

    def test_matrix_records_failed_child_even_without_downstream_reports(self) -> None:
        matrix_file = self._write_matrix_file(None, include_profiles=["single-path"])
        matrix_id = "matrix-test-failed-child"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={"MATRIX_TEST_FAIL_BEFORE_REPORT": "1"},
            release_mode="0",
        )
        self.assertNotEqual(result.returncode, 0)

        records_path = self.e2e_tmp_root / "matrix" / matrix_id / "profile-runs.jsonl"
        self.assertTrue(records_path.exists(), f"missing matrix records: {records_path}")
        records = [
            json.loads(line)
            for line in records_path.read_text(encoding="utf-8").splitlines()
            if line.strip()
        ]
        self.assertEqual(1, len(records))
        self.assertEqual("failed", records[0]["status"])
        self.assertEqual("infra_incomplete_cycle", records[0]["failure_reason"])

        verdict = self._load_verdict(matrix_id)
        self.assertEqual("FAIL", verdict["verdict"])
        self.assertEqual("failed", verdict["records"][0]["status"])

    def test_matrix_reconstructs_failed_record_from_status_files_when_jsonl_is_empty(self) -> None:
        matrix_file = self._write_matrix_file(None, include_profiles=["single-path"])
        matrix_id = "matrix-test-status-fallback"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={
                "MATRIX_TEST_FAIL_BEFORE_REPORT": "1",
                "MATRIX_TEST_TRUNCATE_RECORDS_JSONL": "1",
            },
            release_mode="0",
        )
        self.assertNotEqual(result.returncode, 0)

        records_path = self.e2e_tmp_root / "matrix" / matrix_id / "profile-runs.jsonl"
        self.assertTrue(records_path.exists(), f"missing matrix records path: {records_path}")
        self.assertEqual("", records_path.read_text(encoding="utf-8"))

        status_root = self.e2e_tmp_root / "matrix" / matrix_id / "profile-status"
        status_files = sorted(status_root.glob("*.json"))
        self.assertTrue(status_files, f"missing profile status files under {status_root}")
        status_payload = json.loads(status_files[0].read_text(encoding="utf-8"))
        self.assertEqual("failed", status_payload["status"])
        self.assertEqual("infra_incomplete_cycle", status_payload["failure_reason"])

        verdict = self._load_verdict(matrix_id)
        self.assertEqual("FAIL", verdict["verdict"])
        self.assertEqual(1, len(verdict["records"]))
        self.assertEqual("failed", verdict["records"][0]["status"])

    def test_matrix_outputs_preserve_runtime_flow_failed_backend_class(self) -> None:
        matrix_file = self._write_matrix_file(None, include_profiles=["single-path"])
        matrix_id = "matrix-test-runtime-flow-failed"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={"MATRIX_TEST_RUNTIME_FLOW_FAILED": "1"},
            release_mode="0",
        )
        self.assertNotEqual(result.returncode, 0)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual("FAIL", verdict["verdict"])
        self.assertEqual(1, len(verdict["records"]))
        record = verdict["records"][0]
        self.assertEqual("failed", record["status"])
        self.assertEqual(1, record["backend"]["runtime_flow_failed_runs"])
        self.assertEqual(0, record["backend"]["infra_incomplete_cycle_failures"])

        profile_matrix_tsv = self.e2e_tmp_root / "reports" / f"profile_matrix_{matrix_id}.tsv"
        self.assertTrue(profile_matrix_tsv.exists(), f"missing profile matrix report: {profile_matrix_tsv}")
        lines = [line for line in profile_matrix_tsv.read_text(encoding="utf-8").splitlines() if line.strip()]
        self.assertGreaterEqual(len(lines), 2, f"expected header + row in {profile_matrix_tsv}")
        header = lines[0].split("\t")
        values = lines[1].split("\t")
        row = dict(zip(header, values, strict=False))
        self.assertEqual("1", row["runtime_flow_failed_runs"])
        self.assertEqual("0", row["infra_incomplete_cycle_failures"])

    def test_matrix_writes_durable_inventory_with_raw_output_refs(self) -> None:
        matrix_file = self._write_matrix_file(None, include_profiles=["single-path"])
        matrix_id = "matrix-test-durable-inventory"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={"MATRIX_TEST_RAW_METADATA": "1"},
            release_mode="0",
        )
        self.assertEqual(0, result.returncode, msg=result.stderr or result.stdout)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual("PASS", verdict["verdict"])
        record = verdict["records"][0]
        inventory_path = Path(record["artifacts"]["inventory_json"])
        self.assertTrue(inventory_path.exists(), f"missing inventory file: {inventory_path}")
        self.assertEqual(1, record["artifacts"]["raw_output_ref_count"])

        inventory = json.loads(inventory_path.read_text(encoding="utf-8"))
        self.assertEqual(matrix_id, inventory["matrix_id"])
        self.assertEqual("single-path", inventory["profile_id"])
        self.assertEqual("passed", inventory["terminal_status"])
        refs = inventory["raw_output_refs"]
        self.assertEqual(1, len(refs))
        self.assertEqual("codex-code", refs[0]["provider"])
        self.assertEqual("init.step0.constitution", refs[0]["step_id"])
        self.assertEqual(12, refs[0]["stdout"]["bytes"])

    def test_missing_selected_provider_binary_materializes_operational_blocker_report(self) -> None:
        matrix_file = self._write_matrix_file(None, include_profiles=["single-path"])
        matrix_id = "matrix-test-missing-provider-preflight"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={
                "BATCH_PROVIDER_FILTER": "codex-code",
                "ACP_CODEX_CMD_BIN": "definitely-missing-acp-codex-command",
            },
            release_mode="0",
        )
        self.assertNotEqual(result.returncode, 0)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual("FAIL", verdict["verdict"])
        self.assertEqual("operational-preflight", verdict["release_contract"]["mode"])
        self.assertIn("required command is unavailable", verdict["release_contract"]["blocking_reasons"][0])

        status_path = self.e2e_tmp_root / "matrix" / matrix_id / "profile-status" / "matrix-operational-preflight.json"
        self.assertTrue(status_path.exists(), f"missing operational profile status: {status_path}")
        status = json.loads(status_path.read_text(encoding="utf-8"))
        self.assertEqual("failed", status["status"])
        self.assertEqual("operational_host_preflight_failed", status["failure_reason"])

        inventory_path = Path(status["inventory_json"])
        self.assertTrue(inventory_path.exists(), f"missing operational inventory: {inventory_path}")

    def test_path_sha_mismatch_materializes_profile_operational_blocker_without_child_run(self) -> None:
        repo, _head = self._create_git_repo("sha-mismatch")
        repos_file = self.tmp_root / "matrix-inputs" / "single-path-sha-mismatch.repos.yaml"
        repos_file.parent.mkdir(parents=True, exist_ok=True)
        repos_file.write_text(
            textwrap.dedent(
                f"""\
                repos:
                  - name: sha-mismatch
                    path: {repo}
                    ref: "{'0' * 40}"
                """
            ),
            encoding="utf-8",
        )
        self.profile_repos_files["single-path"] = repos_file
        matrix_file = self._write_matrix_file(None, include_profiles=["single-path"])
        matrix_id = "matrix-test-path-sha-mismatch"
        result = self._run_matrix(matrix_file, matrix_id, release_mode="0")
        self.assertNotEqual(result.returncode, 0)

        self.assertFalse(self.sentinel_path.exists(), "child batch must not start after path SHA mismatch")
        verdict = self._load_verdict(matrix_id)
        self.assertEqual("FAIL", verdict["verdict"])
        self.assertEqual(1, len(verdict["records"]))
        record = verdict["records"][0]
        self.assertEqual("failed", record["status"])

        status_path = self.e2e_tmp_root / "matrix" / matrix_id / "profile-status" / "single-path--baseline.json"
        self.assertTrue(status_path.exists(), f"missing profile status: {status_path}")
        status = json.loads(status_path.read_text(encoding="utf-8"))
        self.assertEqual("operational_host_preflight_failed", status["failure_reason"])
        inventory_path = Path(status["inventory_json"])
        self.assertTrue(inventory_path.exists(), f"missing profile inventory: {inventory_path}")
        driver_log = Path(status["driver_log"])
        self.assertIn("path SHA mismatch", driver_log.read_text(encoding="utf-8"))

    def test_matrix_updates_profile_status_while_child_batch_is_running(self) -> None:
        matrix_file = self._write_matrix_file(None, include_profiles=["single-path"])
        matrix_id = "matrix-test-profile-heartbeat"
        env = self._build_subprocess_env(
            {
                "E2E_MATRIX_FILE": str(matrix_file),
                "BATCH_SCRIPT": str(self.batch_script),
                "MATRIX_ID": matrix_id,
                "E2E_MATRIX_RELEASE_MODE": "0",
                "ACP_CLAUDE_CMD_BIN": "true",
                "ACP_QWEN_CMD_BIN": "true",
                "ACP_CODEX_CMD_BIN": "true",
                "E2E_TMP_ROOT": str(self.e2e_tmp_root),
                "REPORTS_ROOT": str(self.e2e_tmp_root / "reports"),
                "MATRIX_ROOT": str(self.e2e_tmp_root / "matrix" / matrix_id),
                "MATRIX_TEST_SENTINEL": str(self.sentinel_path),
                "MATRIX_TEST_TIMEOUT_SENTINEL": str(self.timeout_sentinel_path),
                "MATRIX_TEST_SLEEP_SEC": "3",
                "MATRIX_PROFILE_STATUS_HEARTBEAT_SEC": "1",
            }
        )
        proc = subprocess.Popen(
            [str(self.matrix_driver)],
            cwd=self.repo_root,
            env=env,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        try:
            status_path = self.e2e_tmp_root / "matrix" / matrix_id / "profile-status" / "single-path--baseline.json"
            deadline = time.time() + 5
            while time.time() < deadline and not status_path.exists():
                time.sleep(0.1)
            self.assertTrue(status_path.exists(), f"missing profile status file: {status_path}")

            first_updated_at = json.loads(status_path.read_text(encoding="utf-8"))["updated_at"]
            changed = False
            deadline = time.time() + 5
            while time.time() < deadline:
                time.sleep(1.1)
                updated_at = json.loads(status_path.read_text(encoding="utf-8"))["updated_at"]
                if updated_at != first_updated_at:
                    changed = True
                    break
            self.assertTrue(changed, f"expected profile status heartbeat to update {status_path}")
        finally:
            stdout, stderr = proc.communicate(timeout=15)
        self.assertEqual(0, proc.returncode, msg=stderr or stdout)

    def test_matrix_reconcile_only_marks_stale_running_profile_as_failed(self) -> None:
        matrix_id = "matrix-test-stale-profile-reconcile"
        status_root = self.e2e_tmp_root / "matrix" / matrix_id / "profile-status"
        batch_root = self.e2e_tmp_root / "runs" / "stale-batch"
        status_root.mkdir(parents=True, exist_ok=True)
        batch_root.mkdir(parents=True, exist_ok=True)

        status_path = status_root / "single-path--baseline.json"
        status_payload = {
            "profile_id": "single-path",
            "profile_slug": "single-path",
            "batch_id": "stale-batch",
            "source_kind": "path",
            "expected_repo_count": 1,
            "repos_file": str(self.profile_repos_files["single-path"]),
            "status": "running",
            "failure_reason": "none",
            "sweep_id": "baseline",
            "execution": {
                "strategy": "sequential",
                "max_parallel_tasks": 1,
                "failure_policy": "best_effort",
                "shard_discovery_mode": "heuristics",
            },
            "batch_root": str(batch_root),
            "updated_at": "2026-04-22T08:00:00Z",
        }
        status_path.write_text(json.dumps(status_payload, ensure_ascii=True) + "\n", encoding="utf-8")
        (batch_root / "batch-owner.env").write_text(
            "\n".join(
                [
                    "batch_id=stale-batch",
                    "profile_id=single-path",
                    "sweep_id=baseline",
                    "pid=999999",
                    "parent_pid=1",
                    "state=running",
                    "process_exit=",
                    "termination_signal=none",
                    "failure_reason=none",
                    "updated_at=2026-04-22T08:00:00Z",
                ]
            )
            + "\n",
            encoding="utf-8",
        )

        result = subprocess.run(
            [str(self.matrix_driver)],
            cwd=self.repo_root,
            env=self._build_subprocess_env(
                {
                    "MATRIX_ID": matrix_id,
                    "MATRIX_ROOT": str(self.e2e_tmp_root / "matrix" / matrix_id),
                    "E2E_TMP_ROOT": str(self.e2e_tmp_root),
                    "REPORTS_ROOT": str(self.e2e_tmp_root / "reports"),
                    "MATRIX_TEST_RECONCILE_ONLY": "1",
                    "MATRIX_PROFILE_STATUS_STALE_SEC": "1",
                }
            ),
            capture_output=True,
            text=True,
        )
        self.assertEqual(0, result.returncode, msg=result.stderr or result.stdout)

        reconciled = json.loads(status_path.read_text(encoding="utf-8"))
        self.assertEqual("failed", reconciled["status"])
        self.assertEqual("infra_incomplete_cycle", reconciled["failure_reason"])

    def test_non_release_regres_matrix_allows_two_profiles(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["single-path", "multi-git_url"],
        )
        matrix_id = "matrix-test-regres-non-release"
        result = self._run_matrix(matrix_file, matrix_id, release_mode="0")
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["mode"], "non-release")
        self.assertEqual(release_contract["observed_sweeps"], ["baseline"])
        self.assertEqual(release_contract["observed_profile_sweep_runs"], 2)

    def test_non_release_single_profile_matrix_allows_implicit_baseline(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["multi-path"],
        )
        matrix_id = "matrix-test-single-profile-non-release"
        result = self._run_matrix(matrix_file, matrix_id, release_mode="0")
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["mode"], "non-release")
        self.assertEqual(release_contract["observed_sweeps"], ["baseline"])
        self.assertEqual(release_contract["observed_profile_sweep_runs"], 1)

    def test_non_release_matrix_allows_positive_run_count_without_release_guard(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["single-path", "multi-git_url"],
        )
        matrix_id = "matrix-test-non-release-run-count-two"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={"RUN_COUNT": "2"},
            release_mode="0",
        )
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["mode"], "non-release")
        self.assertEqual(release_contract["observed_sweeps"], ["baseline"])
        self.assertEqual(release_contract["observed_profile_sweep_runs"], 2)
        self.assertTrue(
            all(
                int(rec.get("backend", {}).get("total_runs", 0)) == 6
                and int(rec.get("backend", {}).get("hard_pass", 0)) == 6
                for rec in verdict.get("records", [])
            )
        )

    def test_release_matrix_ignores_polluted_ambient_env(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default"])
        matrix_id = "release-test-hermetic-polluted-env"
        polluted_env = {
            "BATCH_PROVIDER_FILTER": "qwen-code",
            "BATCH_RUN_SELECTION": "1",
            "RUN_COUNT": "7",
            "ACP_RUNTIME_STEP_TIMEOUT_SEC": "9999",
            "ACP_PIPELINE_TIMEOUT_SEC": "9999",
            "ACP_UI_INIT_POLL_TIMEOUT_SEC": "9999",
            "E2E_TMP_ROOT": str(self.tmp_root / "ambient-e2e"),
            "MATRIX_ID": "ambient-matrix-id",
        }
        with mock.patch.dict(os.environ, polluted_env, clear=False):
            result = self._run_matrix(matrix_file, matrix_id)
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["selected_providers"], ["qwen-code", "claude-code", "codex-code"])
        self.assertEqual(release_contract["selected_run_indexes"], ["1"])
        self.assertEqual(release_contract["expected_backend_runs_per_profile_sweep"], 3)
        self.assertTrue(
            all(
                int(rec.get("backend", {}).get("total_runs", 0)) == 3
                and int(rec.get("backend", {}).get("hard_pass", 0)) == 3
                and rec.get("frontend", {}).get("frontend_qwen_status") == "passed"
                and rec.get("frontend", {}).get("frontend_claude_status") == "passed"
                and rec.get("frontend", {}).get("frontend_codex_status") == "passed"
                and rec.get("strict_status") == "passed"
                for rec in verdict.get("records", [])
            )
        )

    def test_non_release_run_count_uses_test_owned_env_under_polluted_ambient_env(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["single-path", "multi-git_url"],
        )
        matrix_id = "matrix-test-non-release-hermetic-run-count"
        polluted_env = {
            "BATCH_PROVIDER_FILTER": "qwen-code",
            "BATCH_RUN_SELECTION": "1",
            "RUN_COUNT": "9",
            "ACP_RUNTIME_STEP_TIMEOUT_SEC": "1111",
            "ACP_PIPELINE_TIMEOUT_SEC": "2222",
            "ACP_UI_INIT_POLL_TIMEOUT_SEC": "3333",
        }
        with mock.patch.dict(os.environ, polluted_env, clear=False):
            result = self._run_matrix(
                matrix_file,
                matrix_id,
                extra_env={"RUN_COUNT": "2"},
                release_mode="0",
            )
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["mode"], "non-release")
        self.assertEqual(release_contract["selected_providers"], ["qwen-code", "claude-code", "codex-code"])
        self.assertEqual(release_contract["selected_run_indexes"], ["1", "2"])
        self.assertEqual(release_contract["expected_backend_runs_per_profile_sweep"], 6)
        self.assertTrue(
            all(
                int(rec.get("backend", {}).get("total_runs", 0)) == 6
                and int(rec.get("backend", {}).get("hard_pass", 0)) == 6
                and rec.get("strict_status") == "passed"
                for rec in verdict.get("records", [])
            )
        )

    def test_non_release_qwen_only_matrix_uses_single_provider_expectations(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["single-path", "multi-git_url"],
        )
        matrix_id = "matrix-test-non-release-qwen-only"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={"BATCH_PROVIDER_FILTER": "qwen-code"},
            release_mode="0",
        )
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        release_contract = verdict["release_contract"]
        self.assertEqual(release_contract["mode"], "non-release")
        self.assertEqual(release_contract["selected_providers"], ["qwen-code"])
        self.assertEqual(release_contract["selected_run_indexes"], ["1"])
        self.assertEqual(release_contract["expected_backend_runs_per_profile_sweep"], 1)
        self.assertEqual(len(verdict.get("records", [])), int(verdict["backend"]["total_runs"]))
        self.assertEqual(len(verdict.get("records", [])), int(verdict["backend"]["hard_pass"]))
        self.assertTrue(
            all(
                int(rec.get("backend", {}).get("total_runs", 0)) == 1
                and int(rec.get("backend", {}).get("hard_pass", 0)) == 1
                and rec.get("frontend", {}).get("frontend_qwen_status") == "passed"
                and rec.get("frontend", {}).get("frontend_claude_status") == "missing"
                and rec.get("frontend", {}).get("frontend_codex_status") == "missing"
                and rec.get("strict_status") == "passed"
                for rec in verdict.get("records", [])
            )
        )

    def test_non_release_frontend_never_is_non_applicable(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["single-path"],
        )
        matrix_id = "matrix-test-non-release-frontend-never"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={
                "BATCH_PROVIDER_FILTER": "qwen-code",
                "BATCH_FRONTEND_MODE": "never",
                "BATCH_FRONTEND_CANCEL_MODE": "never",
            },
            release_mode="0",
        )
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        self.assertTrue(
            all(
                rec.get("frontend", {}).get("frontend_qwen_status") == "skipped"
                and rec.get("frontend", {}).get("frontend_cancel_qwen_status") == "skipped"
                and rec.get("strict_status") == "passed"
                and not rec.get("blocking_reasons")
                for rec in verdict.get("records", [])
            )
        )

    def test_snapshot_missing_after_backend_failure_is_not_independent_frontend_blocker(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["single-path"],
        )
        matrix_id = "matrix-test-dependent-frontend-snapshot-missing"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={
                "BATCH_PROVIDER_FILTER": "qwen-code,codex-code",
                "MATRIX_TEST_QWEN_BACKEND_FAILURE": "1",
                "MATRIX_TEST_QWEN_FRONTEND_SNAPSHOT_MISSING": "1",
            },
            release_mode="0",
        )
        self.assertNotEqual(result.returncode, 0)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "FAIL")
        record = verdict["records"][0]
        self.assertEqual(record.get("frontend", {}).get("frontend_qwen_status"), "skipped")
        frontend_matrix = Path(record["artifacts"]["frontend_matrix_md"])
        self.assertIn("snapshot_reports_missing=1", frontend_matrix.read_text(encoding="utf-8"))
        blockers = record.get("blocking_reasons", [])
        self.assertTrue(any(reason.startswith("runner_unavailable=1") for reason in blockers), blockers)
        self.assertFalse(
            any("frontend_qwen_status=skipped (expected passed)" in reason for reason in blockers),
            blockers,
        )

    def test_release_matrix_still_requires_dual_provider_execution(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default"])
        matrix_id = "release-test-qwen-only-blocked"
        result = self._run_matrix(
            matrix_file,
            matrix_id,
            extra_env={"BATCH_PROVIDER_FILTER": "qwen-code"},
        )
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("RELEASE BLOCKED", combined_output)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "FAIL")
        self.assertTrue(
            any(
                any("backend_total_runs=1 (expected 3)" in reason for reason in rec.get("blocking_reasons", []))
                or any("frontend_claude_status=missing (expected passed)" in reason for reason in rec.get("blocking_reasons", []))
                or any("frontend_codex_status=missing (expected passed)" in reason for reason in rec.get("blocking_reasons", []))
                for rec in verdict.get("records", [])
            )
        )

    def test_release_slice_with_single_git_and_multi_path_passes(self) -> None:
        matrix_file = self._write_matrix_file(
            ["baseline", "parallel-default"],
            include_profiles=["single-git_url", "multi-path"],
        )
        matrix_id = "release-test-single-git-multi-path"
        result = self._run_matrix(matrix_file, matrix_id)
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        self.assertEqual(verdict["release_contract"]["required_profiles"], ["single-git_url", "multi-path"])

    def test_release_slice_with_single_path_and_multi_path_passes(self) -> None:
        matrix_file = self._write_matrix_file(
            ["baseline", "parallel-default"],
            include_profiles=["single-path", "multi-path"],
        )
        matrix_id = "release-test-single-path-multi-path"
        result = self._run_matrix(matrix_file, matrix_id)
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        self.assertEqual(verdict["release_contract"]["required_profiles"], ["single-path", "multi-path"])

    def test_release_slice_with_single_git_and_multi_git_passes(self) -> None:
        matrix_file = self._write_matrix_file(
            ["baseline", "parallel-default"],
            include_profiles=["single-git_url", "multi-git_url"],
        )
        matrix_id = "release-test-single-git-multi-git"
        result = self._run_matrix(matrix_file, matrix_id)
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        verdict = self._load_verdict(matrix_id)
        self.assertEqual(verdict["verdict"], "PASS")
        self.assertEqual(verdict["release_contract"]["required_profiles"], ["single-git_url", "multi-git_url"])

    def test_matrix_timeout_profile_rejects_unknown_preset(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["multi-path"],
            timeout_profile="unknown-window",
        )
        matrix_id = "matrix-test-timeout-profile-invalid"
        result = self._run_matrix(matrix_file, matrix_id, release_mode="0")
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("timeout_profile must be one of", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when timeout_profile is invalid")

    def test_matrix_timeout_profile_injects_canonical_timeout_env(self) -> None:
        matrix_file = self._write_matrix_file(
            None,
            include_profiles=["multi-path"],
            timeout_profile="short-window",
        )
        matrix_id = "matrix-test-timeout-profile-short-window"
        result = self._run_matrix(matrix_file, matrix_id, release_mode="0")
        self.assertEqual(result.returncode, 0, msg=result.stderr)

        payloads = [
            json.loads(line)
            for line in self.timeout_sentinel_path.read_text(encoding="utf-8").splitlines()
            if line.strip()
        ]
        self.assertEqual(len(payloads), 1)
        self.assertEqual(payloads[0]["profile_id"], "multi-path")
        self.assertEqual(payloads[0]["sweep_id"], "baseline")
        self.assertEqual(payloads[0]["step_timeout_sec"], "3600")
        self.assertEqual(payloads[0]["pipeline_timeout_sec"], "7200")
        self.assertEqual(payloads[0]["ui_init_poll_timeout_sec"], "1200")

    def test_checked_in_profile_catalog_is_consistent(self) -> None:
        catalog_path = self.repo_root / "examples" / "e2e-profile-catalog.yaml"
        catalog = self._load_yaml(catalog_path)

        repo_sets = catalog.get("repo_sets")
        self.assertIsInstance(repo_sets, dict)
        self.assertEqual(len(repo_sets), 6)

        policies = catalog.get("execution_policies")
        self.assertIsInstance(policies, dict)

        profiles = catalog.get("profiles")
        self.assertIsInstance(profiles, list)
        self.assertEqual(len(profiles), 5)

        profile_by_slug = {item["slug"]: item for item in profiles}
        self.assertEqual(len(profile_by_slug), 5)

        all_target_repo_sets: set[str] = set()
        matrix_refs: dict[str, list[str]] = {}
        allowed_ids = {"single-path", "single-git_url", "multi-path", "multi-git_url"}
        allowed_timeout_profiles = {"short-window", "medium-window", "extended-window"}
        examples_root = self.repo_root / "examples"

        for profile in profiles:
            slug = str(profile["slug"])
            policy_name = str(profile["policy"])
            self.assertIn(policy_name, policies, f"unknown policy for {slug}")
            policy = policies[policy_name]
            providers = policy.get("providers")
            self.assertIsInstance(providers, list)
            self.assertGreater(len(providers), 0)

            target_repo_sets = profile.get("target_repo_sets")
            self.assertIsInstance(target_repo_sets, list)
            all_target_repo_sets.update(str(item) for item in target_repo_sets)

            matrix_files = profile.get("matrix_files")
            self.assertIsInstance(matrix_files, list)
            self.assertGreater(len(matrix_files), 0)

            computed_runs = 0
            for rel_matrix in matrix_files:
                matrix_path = (examples_root / str(rel_matrix).replace("./", "")).resolve()
                self.assertTrue(matrix_path.exists(), f"missing matrix file for {slug}: {matrix_path}")
                matrix_refs.setdefault(str(matrix_path), []).append(slug)

                matrix_payload = self._load_yaml(matrix_path)
                timeout_profile = matrix_payload.get("timeout_profile")
                self.assertIn(timeout_profile, allowed_timeout_profiles, f"missing/invalid timeout_profile in {matrix_path}")
                matrix_profiles = matrix_payload.get("profiles")
                self.assertIsInstance(matrix_profiles, list)
                self.assertGreater(len(matrix_profiles), 0)
                for item in matrix_profiles:
                    self.assertIn(item.get("id"), allowed_ids, f"illegal concrete profile id in {matrix_path}")

                sweep_count = len(matrix_payload.get("sweeps") or [{"id": "baseline"}])
                computed_runs += len(matrix_profiles) * sweep_count * len(providers)

                if profile["mode"] == "release":
                    self._assert_release_slice_shape(
                        matrix_path,
                        [item.get("id") for item in matrix_profiles],
                    )
                else:
                    self.assertNotIn("sweeps", matrix_payload, f"regres slice should use implicit baseline: {matrix_path}")

            self.assertEqual(
                computed_runs,
                int(profile["expected_backend_runs"]),
                f"expected backend runs drift for {slug}",
            )

        self.assertEqual(all_target_repo_sets, set(repo_sets.keys()))

        for matrix_path, refs in matrix_refs.items():
            if len(refs) == 1:
                continue
            self.assertEqual(len(refs), 2, f"unexpected matrix reuse: {matrix_path} -> {refs}")
            composite_slug = next(
                (
                    slug
                    for slug in refs
                    if profile_by_slug[slug].get("composite_of_profiles")
                ),
                None,
            )
            self.assertIsNotNone(composite_slug, f"matrix reuse must come from explicit composite profile: {matrix_path}")
            other_refs = {slug for slug in refs if slug != composite_slug}
            composite_children = set(profile_by_slug[composite_slug].get("composite_of_profiles", []))
            self.assertEqual(
                other_refs,
                composite_children & other_refs,
                f"matrix reuse mismatch for {matrix_path}: refs={refs} composite={composite_children}",
            )

        self._assert_release_slice_shape(
            self.repo_root / "examples" / "e2e-matrix.release-fast.yaml",
            ["single-git_url", "multi-path"],
        )
        self.assertEqual(
            self._load_yaml(self.repo_root / "examples" / "e2e-matrix.release-fast.yaml").get("timeout_profile"),
            "short-window",
        )
        self._assert_release_slice_shape(
            self.repo_root / "examples" / "e2e-matrix.release-long.yaml",
            ["single-path", "multi-path"],
        )
        self.assertEqual(
            self._load_yaml(self.repo_root / "examples" / "e2e-matrix.release-long.yaml").get("timeout_profile"),
            "medium-window",
        )
        self._assert_release_slice_shape(
            self.repo_root / "examples" / "e2e-matrix.release-full.ftgo-sentry.yaml",
            ["single-git_url", "multi-git_url"],
        )
        self.assertEqual(
            self._load_yaml(self.repo_root / "examples" / "e2e-matrix.release-full.ftgo-sentry.yaml").get("timeout_profile"),
            "extended-window",
        )
        self._assert_non_release_slice_shape(
            self.repo_root / "examples" / "e2e-matrix.regres-fast.bank-openedx.yaml",
            ["single-git_url", "multi-path"],
        )
        self.assertEqual(
            self._load_yaml(self.repo_root / "examples" / "e2e-matrix.regres-fast.bank-openedx.yaml").get("timeout_profile"),
            "short-window",
        )
        self._assert_non_release_slice_shape(
            self.repo_root / "examples" / "e2e-matrix.regres-fast.openstack.yaml",
            ["multi-path"],
        )
        self.assertEqual(
            self._load_yaml(self.repo_root / "examples" / "e2e-matrix.regres-fast.openstack.yaml").get("timeout_profile"),
            "short-window",
        )
        self._assert_non_release_slice_shape(
            self.repo_root / "examples" / "e2e-matrix.regres-long.yaml",
            ["single-path", "single-git_url"],
        )
        self.assertEqual(
            self._load_yaml(self.repo_root / "examples" / "e2e-matrix.regres-long.yaml").get("timeout_profile"),
            "medium-window",
        )


if __name__ == "__main__":
    unittest.main()
