import json
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


class ResolveReposMetaTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.script = cls.repo_root / "scripts" / "resolve-repos-meta.py"

    def setUp(self) -> None:
        self.tmpdir = tempfile.TemporaryDirectory()
        self.tmp_root = Path(self.tmpdir.name)

    def tearDown(self) -> None:
        self.tmpdir.cleanup()

    def _create_git_repo(self) -> tuple[Path, str]:
        repo = self.tmp_root / "repo"
        repo.mkdir(parents=True, exist_ok=True)
        subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True, text=True)
        subprocess.run(["git", "config", "user.email", "test@example.invalid"], cwd=repo, check=True)
        subprocess.run(["git", "config", "user.name", "Test User"], cwd=repo, check=True)
        (repo / "README.md").write_text("# repo\n", encoding="utf-8")
        subprocess.run(["git", "add", "README.md"], cwd=repo, check=True)
        subprocess.run(["git", "commit", "-m", "init"], cwd=repo, check=True, capture_output=True, text=True)
        head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
        return repo, head

    def _run_resolver(self, repos_file: Path, out_file: Path) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                "python3",
                str(self.script),
                "--repos-file",
                str(repos_file),
                "--expected-repo-count",
                "1",
                "--source-kind",
                "path",
                "--profile-id",
                "single-path",
                "--out",
                str(out_file),
            ],
            cwd=self.repo_root,
            capture_output=True,
            text=True,
        )

    def test_path_repo_pinned_sha_match_is_accepted(self) -> None:
        repo, head = self._create_git_repo()
        repos_file = self.tmp_root / "repos.yaml"
        out_file = self.tmp_root / "meta.json"
        repos_file.write_text(
            textwrap.dedent(
                f"""\
                repos:
                  - name: sample
                    path: {repo}
                    ref: {head}
                """
            ),
            encoding="utf-8",
        )

        result = self._run_resolver(repos_file, out_file)
        self.assertEqual(0, result.returncode, msg=result.stderr or result.stdout)
        payload = json.loads(out_file.read_text(encoding="utf-8"))
        self.assertEqual(head, payload["declared_repos"][0]["ref"])

    def test_path_repo_pinned_sha_mismatch_is_operational_blocker(self) -> None:
        repo, _head = self._create_git_repo()
        repos_file = self.tmp_root / "repos.yaml"
        out_file = self.tmp_root / "meta.json"
        mismatch = "0" * 40
        repos_file.write_text(
            textwrap.dedent(
                f"""\
                repos:
                  - name: sample
                    path: {repo}
                    ref: "{mismatch}"
                """
            ),
            encoding="utf-8",
        )

        result = self._run_resolver(repos_file, out_file)
        self.assertNotEqual(0, result.returncode)
        self.assertIn("path SHA mismatch", result.stderr or result.stdout)
        self.assertFalse(out_file.exists())


if __name__ == "__main__":
    unittest.main()
