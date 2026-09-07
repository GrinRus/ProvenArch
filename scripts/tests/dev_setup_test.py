import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


class DevSetupTest(unittest.TestCase):
    repo = Path(__file__).resolve().parents[2]

    def fixture(self, root: Path, npm_version="10.9.4", venv_version="3.10.8"):
        scripts = root / "scripts"
        scripts.mkdir()
        for name in ("setup-dev.sh", "dev-preflight.sh", "requirements-dev.txt"):
            shutil.copyfile(self.repo / "scripts" / name, scripts / name)
        (root / ".python-version").write_text("3.10.8\n")
        (root / "go.mod").write_text("module example.invalid/test\ngo 1.20\n")
        (root / "go.sum").write_text("existing module checksums\n")

        def executable(path, body):
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("#!/usr/bin/env bash\nset -eu\n" + body + "\n")
            path.chmod(0o755)

        executable(scripts / "run-go.sh", 'echo "go $*" >> calls; echo "go version go1.25.10"')
        executable(scripts / "run-npm.sh", f'''
echo "npm $*" >> calls
if [[ "${{1:-}}" == --version ]]; then echo {npm_version}; exit; fi
mkdir -p "$3/node_modules/.bin"
touch "$3/node_modules/.bin/vitest" "$3/node_modules/.bin/ajv"
chmod +x "$3/node_modules/.bin/"*
''')
        executable(scripts / "run-python.sh", 'echo "python $*" >> calls; echo "Python 3.10.8"')
        executable(root / ".venv/bin/python", f'''
if [[ "${{1:-}}" == --version ]]; then echo "Python {venv_version}"; exit; fi
echo "venv $*" >> calls
''')
        executable(root / "bin/git", "echo git-ready")
        executable(root / "bin/shellcheck", "echo shellcheck-ready")
        return {
            **{key: value for key, value in os.environ.items() if not key.startswith("ACP_PYTHON_")},
            "PROVENARCH_ROOT": str(root),
            "PATH": str(root / "bin") + ":/usr/bin:/bin",
        }

    def run_script(self, root, name, env):
        return subprocess.run(
            ["bash", str(root / "scripts" / name)], cwd=root, env=env,
            capture_output=True, text=True,
        )

    def test_preflight_reports_missing_dependencies_without_installing(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            env = self.fixture(root)
            result = self.run_script(root, "dev-preflight.sh", env)
            self.assertNotEqual(0, result.returncode)
            self.assertIn("UI dependencies", result.stderr)
            self.assertIn("contract dependencies", result.stderr)
            calls = (root / "calls").read_text()
            self.assertNotIn("npm ci", calls)
            self.assertNotIn("pip install", calls)
            self.assertFalse((root / "ui").exists())

    def test_setup_stops_before_dependency_install_on_toolchain_mismatch(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            env = self.fixture(root, npm_version="11.0.0")
            result = self.run_script(root, "setup-dev.sh", env)
            self.assertNotEqual(0, result.returncode)
            self.assertNotIn("npm ci", (root / "calls").read_text())

    def test_setup_rejects_stale_venv_without_overwriting_it(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            env = self.fixture(root, venv_version="3.11.9")
            before = (root / ".venv/bin/python").read_bytes()
            result = self.run_script(root, "setup-dev.sh", env)
            self.assertNotEqual(0, result.returncode)
            self.assertIn("recreate this worktree's venv", result.stderr)
            self.assertEqual(before, (root / ".venv/bin/python").read_bytes())
            self.assertNotIn("npm ci", (root / "calls").read_text())

    def test_setup_uses_locked_dependencies_without_tidy_or_global_pip(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            env = self.fixture(root)
            before = [(root / p).read_bytes() for p in ("go.mod", "go.sum")]
            result = self.run_script(root, "setup-dev.sh", env)
            self.assertEqual(0, result.returncode, result.stderr)
            calls = (root / "calls").read_text()
            self.assertIn("go mod download", calls)
            self.assertNotIn("tidy", calls)
            self.assertIn("npm ci --prefix tools/contracts --ignore-scripts", calls)
            self.assertIn("npm ci --prefix ui", calls)
            self.assertIn("venv -m pip install --disable-pip-version-check -r scripts/requirements-dev.txt", calls)
            self.assertEqual(before, [(root / p).read_bytes() for p in ("go.mod", "go.sum")])

    def test_fresh_setup_creates_real_venv_from_explicit_base_and_uses_it(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            env = self.fixture(root)
            shutil.rmtree(root / ".venv")
            shutil.copyfile(self.repo / "scripts/run-python.sh", root / "scripts/run-python.sh")
            (root / "scripts/requirements-dev.txt").write_text("# No downloaded packages in this isolated fixture.\n")
            env.update({
                "ACP_PYTHON_BASE_BIN": str(Path(getattr(sys, "_base_executable", sys.executable))),
                "PIP_NO_INDEX": "1",
            })
            self.assertFalse((root / ".venv").exists())
            result = self.run_script(root, "setup-dev.sh", env)
            self.assertEqual(0, result.returncode, result.stdout + result.stderr)
            self.assertTrue((root / ".venv/pyvenv.cfg").exists())
            selected = subprocess.check_output(
                ["bash", str(root / "scripts/run-python.sh"), "-c", "import sys; print(sys.prefix)"],
                cwd=root, env=env, text=True,
            ).strip()
            self.assertEqual((root / ".venv").resolve(), Path(selected).resolve())
            # An intentional runtime override is valid when it points at this same venv.
            env["ACP_PYTHON_BIN"] = str(root / ".venv/bin/python")
            result = self.run_script(root, "setup-dev.sh", env)
            self.assertEqual(0, result.returncode, result.stdout + result.stderr)

    def test_external_runtime_override_fails_before_bootstrap_mutations(self):
        for override in ("ACP_PYTHON_BIN", "ACP_PYTHON_TOOL_CANDIDATES"):
            with self.subTest(override=override), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                env = self.fixture(root)
                shutil.rmtree(root / ".venv")
                shutil.copyfile(self.repo / "scripts/run-python.sh", root / "scripts/run-python.sh")
                base = Path(getattr(sys, "_base_executable", sys.executable))
                env[override] = str(base if override == "ACP_PYTHON_BIN" else base.parent)
                result = self.run_script(root, "setup-dev.sh", env)
                self.assertNotEqual(0, result.returncode)
                self.assertIn("ACP_PYTHON_BASE_BIN", result.stderr)
                self.assertIn("unset those runtime overrides", result.stderr)
                self.assertFalse((root / ".venv").exists())
                calls = (root / "calls").read_text()
                self.assertNotIn("npm ci", calls)
                self.assertNotIn("go mod download", calls)


if __name__ == "__main__":
    unittest.main()
