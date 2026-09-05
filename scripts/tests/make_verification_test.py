import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


class MakeVerificationTest(unittest.TestCase):
    def test_parallel_make_does_not_validate_during_dependency_installation(self):
        repo = Path(__file__).resolve().parents[2]
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            shutil.copyfile(repo / "Makefile", root / "Makefile")
            (root / "scripts").mkdir()
            installer = root / "npm-fixture"
            installer.write_text(
                '#!/usr/bin/env bash\nset -eu\n'
                'if [[ "$1" == ci ]]; then\n'
                '  touch installing\n'
                '  sleep 0.2\n'
                '  touch installed\n'
                '  rm installing\n'
                'fi\n'
            )
            installer.chmod(0o755)
            (root / "scripts/validate-contracts.sh").write_text(
                '#!/usr/bin/env bash\nset -eu\n'
                '[[ -f installed && ! -f installing ]] || { echo install-race >&2; exit 1; }\n'
                'echo validated >> checks\n'
            )
            result = subprocess.run(
                ["make", "-j2", "contracts", "test", "GO=true", "PYTHON=true", f"NPM={installer}"],
                cwd=root, env={**os.environ, "MAKEFLAGS": ""},
                capture_output=True, text=True,
            )
            self.assertEqual(0, result.returncode, result.stdout + result.stderr)
            self.assertNotIn("install-race", result.stderr)
            self.assertTrue((root / "checks").read_text().strip())


if __name__ == "__main__":
    unittest.main()
