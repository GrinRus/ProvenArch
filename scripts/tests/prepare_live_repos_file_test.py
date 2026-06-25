import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


class PrepareLiveReposFileTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.script = cls.repo_root / "scripts" / "prepare-live-repos-file.py"

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

    def _run_prepare(
        self,
        repos_file: Path,
        out_file: Path,
        work_dir: Path,
        *,
        make_read_only: bool = False,
    ) -> subprocess.CompletedProcess[str]:
        cmd = [
            "python3",
            str(self.script),
            "--repos-file",
            str(repos_file),
            "--work-dir",
            str(work_dir),
            "--out",
            str(out_file),
        ]
        if make_read_only:
            cmd.append("--make-read-only")
        return subprocess.run(cmd, cwd=self.repo_root, capture_output=True, text=True)

    def test_path_repo_is_rewritten_to_isolated_detached_checkout(self) -> None:
        repo, head = self._create_git_repo()
        repos_file = self.tmp_root / "repos.yaml"
        out_file = self.tmp_root / "generated" / "repos.yaml"
        work_dir = self.tmp_root / "source-repos"
        repos_file.write_text(
            textwrap.dedent(
                f"""\
                version: 1
                repos:
                  - name: sample
                    path: {repo}
                    ref: {head}
                """
            ),
            encoding="utf-8",
        )

        result = self._run_prepare(repos_file, out_file, work_dir, make_read_only=True)
        self.assertEqual(0, result.returncode, msg=result.stderr or result.stdout)
        generated = out_file.read_text(encoding="utf-8")
        self.assertIn("01-sample", generated)
        self.assertNotIn(str(repo), generated)
        isolated = work_dir / "01-sample"
        self.assertTrue(isolated.is_dir())
        actual = subprocess.check_output(["git", "-C", str(isolated), "rev-parse", "HEAD"], text=True).strip()
        self.assertEqual(head, actual)
        mode = isolated.stat().st_mode
        self.assertFalse(mode & stat.S_IWUSR)

    def test_corrupt_path_checkout_is_operational_blocker(self) -> None:
        repo = self.tmp_root / "not-git"
        repo.mkdir()
        repos_file = self.tmp_root / "repos.yaml"
        out_file = self.tmp_root / "generated" / "repos.yaml"
        repos_file.write_text(
            textwrap.dedent(
                f"""\
                version: 1
                repos:
                  - name: broken
                    path: {repo}
                    ref: {"0" * 40}
                """
            ),
            encoding="utf-8",
        )

        result = self._run_prepare(repos_file, out_file, self.tmp_root / "source-repos")
        self.assertNotEqual(0, result.returncode)
        self.assertIn("rev-parse --is-inside-work-tree failed", result.stderr or result.stdout)
        self.assertFalse(out_file.exists())


if __name__ == "__main__":
    unittest.main()
