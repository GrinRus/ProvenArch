import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]


def write_executable(path: Path, body: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(f"#!{sys.executable}\n" + body, encoding="utf-8")
    path.chmod(0o755)


class UIVerificationTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / "checkout"
        self.root.mkdir()
        (self.root / "scripts").mkdir()
        for script in ("check-ui-dist-fresh.sh", "verify-ui-deterministic-build.sh", "ui-mock-e2e.sh"):
            shutil.copy2(REPO_ROOT / "scripts" / script, self.root / "scripts" / script)
        (self.root / "ui/node_modules/.vite").mkdir(parents=True)
        (self.root / "ui/node_modules/.vite/keep").write_text("cache", encoding="utf-8")
        (self.root / "ui/source.txt").write_text("committed", encoding="utf-8")
        (self.root / "ui/package.json").write_text('{"name":"fixture"}', encoding="utf-8")
        (self.root / "ui/package-lock.json").write_text('{"name":"fixture","lockfileVersion":3}', encoding="utf-8")
        (self.root / "ui/dist").mkdir()
        (self.root / "ui/dist/keep").write_text("existing build", encoding="utf-8")
        (self.root / "internal/api/ui_dist/assets").mkdir(parents=True)
        (self.root / "internal/api/ui_dist/index.html").write_text("committed", encoding="utf-8")
        (self.root / "internal/api/ui_dist/assets/main.js").write_text("committed", encoding="utf-8")
        (self.root / "internal/api/ui_dist/README.md").write_text("bundle docs", encoding="utf-8")
        (self.root / ".gitignore").write_text("ui/node_modules/\nui/dist/\n", encoding="utf-8")
        write_executable(self.root / "scripts/run-npm.sh", """
import os
import pathlib
import sys
args = sys.argv[1:]
assert args[:2] == ["run", "build"], args
ui = pathlib.Path(args[args.index("--prefix") + 1])
out = pathlib.Path(args[args.index("--outDir") + 1]) if "--outDir" in args else ui / "dist"
out.mkdir(parents=True, exist_ok=True)
(out / "assets").mkdir(exist_ok=True)
body = (ui / "source.txt").read_text()
if os.environ.get("UI_TEST_EXPECTED_SOURCE"):
    assert body == os.environ["UI_TEST_EXPECTED_SOURCE"], body
(out / "index.html").write_text(body)
(out / "assets/main.js").write_text(body)
""")
        self.git("init", "-q")
        self.git("add", ".")
        self.git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "fixture")
        self.env = {key: value for key, value in os.environ.items() if not key.startswith(("UI_E2E_", "ACP_UI_MOCK_", "UI_TEST_"))}

    def git(self, *args: str) -> str:
        return subprocess.check_output(["git", *args], cwd=self.root, text=True)

    def run_script(self, name: str, *args: str, env: dict | None = None) -> subprocess.CompletedProcess:
        return subprocess.run(
            ["bash", str(self.root / "scripts" / name), *args], cwd=self.root,
            env={**self.env, **(env or {})}, capture_output=True, text=True, timeout=30,
        )

    def bundle_snapshot(self) -> dict[str, bytes]:
        bundle = self.root / "internal/api/ui_dist"
        return {str(path.relative_to(bundle)): path.read_bytes() for path in bundle.rglob("*") if path.is_file()}

    def test_freshness_compares_unstaged_bundle_without_mutating_outputs(self) -> None:
        for path in ("ui/source.txt", "internal/api/ui_dist/index.html", "internal/api/ui_dist/assets/main.js"):
            (self.root / path).write_text("edited", encoding="utf-8")
        before_bundle = self.bundle_snapshot()
        before_status = self.git("status", "--porcelain")
        result = self.run_script("check-ui-dist-fresh.sh")
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertEqual(before_bundle, self.bundle_snapshot())
        self.assertEqual(before_status, self.git("status", "--porcelain"))
        self.assertEqual("existing build", (self.root / "ui/dist/keep").read_text())
        self.assertEqual("cache", (self.root / "ui/node_modules/.vite/keep").read_text())

    def test_stale_freshness_failure_preserves_bundle_and_git_index(self) -> None:
        (self.root / "ui/source.txt").write_text("edited", encoding="utf-8")
        (self.root / "internal/api/ui_dist/assets/retained.js").write_text("user asset", encoding="utf-8")
        before_bundle = self.bundle_snapshot()
        before_status = self.git("status", "--porcelain")
        result = self.run_script("check-ui-dist-fresh.sh")
        self.assertNotEqual(0, result.returncode)
        self.assertIn("is stale", result.stderr)
        self.assertEqual(before_bundle, self.bundle_snapshot())
        self.assertEqual(before_status, self.git("status", "--porcelain"))

    def test_determinism_defaults_to_worktree_and_allows_head_without_deleting_cache(self) -> None:
        (self.root / "ui/source.txt").write_text("edited", encoding="utf-8")
        (self.root / "ui/new-file.txt").write_text("new", encoding="utf-8")
        (self.root / "internal/api/ui_dist/README.md").unlink()
        for args, expected_source in (((), "edited"), (("HEAD",), "committed")):
            with self.subTest(source=expected_source):
                result = self.run_script("verify-ui-deterministic-build.sh", *args, env={"UI_TEST_EXPECTED_SOURCE": expected_source})
                self.assertEqual(0, result.returncode, result.stderr)
                self.assertEqual("cache", (self.root / "ui/node_modules/.vite/keep").read_text())

    def test_determinism_requires_prepared_dependencies_without_installing(self) -> None:
        shutil.rmtree(self.root / "ui/node_modules")
        result = self.run_script("verify-ui-deterministic-build.sh")
        self.assertNotEqual(0, result.returncode)
        self.assertIn("run make bootstrap", result.stderr)
        self.assertFalse((self.root / "ui/node_modules").exists())

    def test_reference_determinism_rejects_different_dependency_manifests(self) -> None:
        for dependency_file in ("package.json", "package-lock.json"):
            with self.subTest(file=dependency_file):
                path = self.root / "ui" / dependency_file
                original = path.read_bytes()
                path.write_text('{"name":"different"}', encoding="utf-8")
                try:
                    result = self.run_script("verify-ui-deterministic-build.sh", "HEAD")
                    self.assertNotEqual(0, result.returncode)
                    self.assertIn(f"UI {dependency_file} differs for ref HEAD", result.stderr)
                    self.assertIn("separate checkout", result.stderr)
                finally:
                    path.write_bytes(original)

    def prepare_mock_runner(self) -> None:
        shutil.copytree(REPO_ROOT / "ui/e2e", self.root / "ui/e2e")
        write_executable(self.root / "scripts/resolve-node-tool.sh", """
import pathlib
print(pathlib.Path(__file__).parent / "fake-tools/node")
""")
        write_executable(self.root / "scripts/fake-tools/node", """
import os
print(os.environ.get("UI_TEST_PORT", "23457"))
""")
        write_executable(self.root / "ui/node_modules/.bin/playwright", """
import json
import os
import pathlib
import shutil
path = pathlib.Path(os.environ["UI_E2E_PLAYWRIGHT_OUTPUT_DIR"])
path.mkdir(parents=True)
observed = {key: value for key, value in os.environ.items() if key.startswith("UI_E2E_")}
observed["selected_node"] = shutil.which("node")
(path / "observed.json").write_text(json.dumps(observed))
print("1 passed")
""")

    def test_concurrent_mock_runs_preserve_custom_directories_and_isolate_every_scenario(self) -> None:
        self.prepare_mock_runner()
        evidence = Path(self.tmp.name) / "evidence"
        playwright = Path(self.tmp.name) / "playwright"
        for directory in (evidence, playwright):
            directory.mkdir()
            (directory / "keep").write_text("existing evidence", encoding="utf-8")
        env = {**self.env, "ACP_UI_MOCK_E2E_RESULTS_DIR": str(evidence), "UI_E2E_PLAYWRIGHT_OUTPUT_DIR": str(playwright)}
        command = ["bash", str(self.root / "scripts/ui-mock-e2e.sh")]
        runs = [subprocess.Popen(command, cwd=self.root, env={**env, "UI_TEST_PORT": str(port)}, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True) for port in (23111, 23112)]
        for run in runs:
            stdout, stderr = run.communicate(timeout=30)
            self.assertEqual(0, run.returncode, stdout + stderr)
        for directory in (evidence, playwright):
            self.assertEqual("existing evidence", (directory / "keep").read_text())
        self.assertEqual(2, len(list(evidence.glob("provenarch-ui-mock-e2e.*"))))
        self.assertEqual(16, len(list(evidence.glob("*/logs/*.log"))))
        run_dirs = list(playwright.glob("run.*"))
        self.assertEqual(2, len(run_dirs))
        urls = []
        for run_dir in run_dirs:
            observations = [json.loads(path.read_text()) for path in run_dir.glob("*/observed.json")]
            self.assertEqual(8, len(observations))
            self.assertEqual({str(self.root / "scripts/fake-tools/node")}, {observed["selected_node"] for observed in observations})
            run_urls = {observed["UI_E2E_BASE_URL"] for observed in observations}
            self.assertEqual(1, len(run_urls))
            urls.append(run_urls.pop())
        self.assertEqual(2, len(set(urls)))

    def test_mock_runner_preserves_explicit_server_overrides(self) -> None:
        self.prepare_mock_runner()
        evidence = Path(self.tmp.name) / "evidence"
        result = self.run_script("ui-mock-e2e.sh", env={
            "ACP_UI_MOCK_E2E_RESULTS_DIR": str(evidence),
            "UI_E2E_BASE_URL": "http://127.0.0.1:23456",
            "UI_E2E_WEB_SERVER_COMMAND": "custom-test-server",
        })
        self.assertEqual(0, result.returncode, result.stderr)
        observations = list(evidence.glob("*/test-results/run.*/*/observed.json"))
        self.assertEqual(8, len(observations))
        for path in observations:
            observed = json.loads(path.read_text())
            self.assertEqual("http://127.0.0.1:23456", observed["UI_E2E_BASE_URL"])
            self.assertEqual("custom-test-server", observed["UI_E2E_WEB_SERVER_COMMAND"])
            self.assertEqual(str(self.root / "scripts/fake-tools/node"), observed["selected_node"])


if __name__ == "__main__":
    unittest.main()
