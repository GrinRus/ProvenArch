import json
import os
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


class MatrixReleaseContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.matrix_driver = cls.repo_root / "scripts" / "full-run-batch-matrix.sh"

    def setUp(self) -> None:
        self.tmpdir = tempfile.TemporaryDirectory()
        self.tmp_root = Path(self.tmpdir.name)
        self.e2e_tmp_root = self.tmp_root / "e2e"
        self.sentinel_path = self.tmp_root / "batch-calls.log"
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

                mkdir -p "${REPORTS_ROOT}" "${BATCH_ROOT}/qwen-code/run1/reports/taskruns"

                run_matrix_tsv="${REPORTS_ROOT}/run_matrix_${BATCH_ID}.tsv"
                run_matrix_md="${REPORTS_ROOT}/run_matrix_${BATCH_ID}.md"
                quality_report_md="${REPORTS_ROOT}/quality_report_${BATCH_ID}.md"
                frontend_matrix_md="${REPORTS_ROOT}/frontend_e2e_matrix_${BATCH_ID}.md"
                frontend_cancel_matrix_md="${REPORTS_ROOT}/frontend_cancel_e2e_matrix_${BATCH_ID}.md"

                {
                  printf 'hard_pass\\truntime_parse\\trunner_unavailable\\truntime_timeout\\tinfra_signal_terminated\\tinfra_incomplete_cycle\\tquality_gates_failed\\tsummary_missing\\tprecheck_failed\\tcancellation_like\\truntime_flow_failed\\tsemantic_hard_fail\\toff_topic_hits\\tartifact_source\\tissues\\n'
                  for _ in $(seq 1 10); do
                    printf '1\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\t0\\tsnapshot\\t-\\n'
                  done
                } > "${run_matrix_tsv}"

                printf '# run matrix\\n' > "${run_matrix_md}"
                printf '# quality\\n' > "${quality_report_md}"

                cat > "${frontend_matrix_md}" <<'EOF'
                | provider | status | runs |
                |---|---|---|
                | qwen-code | passed | 1 |
                | claude-code | passed | 1 |
                EOF

                cat > "${frontend_cancel_matrix_md}" <<'EOF'
                | provider | status | runs |
                |---|---|---|
                | qwen-code | passed | 1 |
                | claude-code | passed | 1 |
                EOF

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
                """
            ),
            encoding="utf-8",
        )
        self.batch_script.chmod(self.batch_script.stat().st_mode | stat.S_IXUSR)

    def _write_matrix_file(self, sweeps: list[str] | None, include_profiles: list[str] | None = None) -> Path:
        ordered_profiles = ["single-path", "single-git_url", "multi-path", "multi-git_url"]
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
        env = os.environ.copy()
        env.update(
            {
                "E2E_MATRIX_FILE": str(matrix_file),
                "BATCH_SCRIPT": str(self.batch_script),
                "MATRIX_ID": matrix_id,
                "E2E_MATRIX_RELEASE_MODE": release_mode,
                "ACP_CLAUDE_CMD_BIN": "true",
                "ACP_QWEN_CMD_BIN": "true",
                "E2E_TMP_ROOT": str(self.e2e_tmp_root),
                "MATRIX_TEST_SENTINEL": str(self.sentinel_path),
            }
        )
        if extra_env:
            env.update(extra_env)
        return subprocess.run(
            [str(self.matrix_driver)],
            cwd=self.repo_root,
            env=env,
            capture_output=True,
            text=True,
        )

    def _load_verdict(self, matrix_id: str) -> dict:
        verdict_path = self.e2e_tmp_root / "reports" / f"release_verdict_{matrix_id}.json"
        self.assertTrue(verdict_path.exists(), f"missing verdict file: {verdict_path}")
        return json.loads(verdict_path.read_text(encoding="utf-8"))

    def test_release_matrix_requires_parallel_default_sweep(self) -> None:
        matrix_file = self._write_matrix_file(["baseline"])
        matrix_id = "release-test-missing-parallel"
        result = self._run_matrix(matrix_file, matrix_id)
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing required ids: parallel-default", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when release sweeps are invalid")

    def test_release_matrix_rejects_extra_sweep(self) -> None:
        matrix_file = self._write_matrix_file(["baseline", "parallel-default", "extra-sweep"])
        matrix_id = "release-test-extra-sweep"
        result = self._run_matrix(matrix_file, matrix_id)
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("contains unsupported ids: extra-sweep", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when release sweeps contain extra ids")

    def test_release_matrix_requires_all_four_profiles(self) -> None:
        matrix_file = self._write_matrix_file(
            ["baseline", "parallel-default"],
            include_profiles=["single-path", "single-git_url", "multi-path"],
        )
        matrix_id = "release-test-missing-profile"
        result = self._run_matrix(matrix_file, matrix_id)
        combined_output = result.stdout + "\n" + result.stderr
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing required profile ids: multi-git_url", combined_output)
        self.assertFalse(self.sentinel_path.exists(), "batch script should not run when release profile set is incomplete")

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
        self.assertEqual(release_contract["expected_profile_sweep_runs"], 8)
        self.assertEqual(release_contract["observed_profile_sweep_runs"], 8)
        self.assertEqual(release_contract["shard_plan_invariant_status"], "passed")

        calls = self.sentinel_path.read_text(encoding="utf-8").splitlines()
        self.assertEqual(len(calls), 8)

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
        matrix_file = self._write_matrix_file(None)
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


if __name__ == "__main__":
    unittest.main()
