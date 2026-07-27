import subprocess
import unittest
from pathlib import Path


class ReadableFixtureDriftTest(unittest.TestCase):
    def test_tracked_readable_exports_match_machine_snapshot_digests(self) -> None:
        root = Path(__file__).resolve().parents[2]
        result = subprocess.run(
            [str(root / "scripts" / "run-python.sh"), str(root / "scripts" / "verify-readable-fixture-drift.py")],
            cwd=root,
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("readable fixture drift check passed", result.stdout)


if __name__ == "__main__":
    unittest.main()
