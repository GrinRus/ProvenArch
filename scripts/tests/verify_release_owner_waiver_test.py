import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


class VerifyReleaseOwnerWaiverTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.script = Path(__file__).resolve().parents[1] / "verify-release-owner-waiver.py"

    def repo(self, root: Path) -> tuple[str, str]:
        subprocess.run(["git", "init", "-q"], cwd=root, check=True)
        subprocess.run(["git", "config", "user.email", "test@example.com"], cwd=root, check=True)
        subprocess.run(["git", "config", "user.name", "Test"], cwd=root, check=True)
        (root / "first").write_text("first", encoding="utf-8")
        subprocess.run(["git", "add", "first"], cwd=root, check=True)
        subprocess.run(["git", "commit", "-qm", "first"], cwd=root, check=True)
        base = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
        (root / "second").write_text("second", encoding="utf-8")
        subprocess.run(["git", "add", "second"], cwd=root, check=True)
        subprocess.run(["git", "commit", "-qm", "second"], cwd=root, check=True)
        source = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
        subprocess.run(["git", "tag", "v0.1.10"], cwd=root, check=True)
        return base, source

    def payload(self, base: str) -> dict[str, object]:
        return {
            "schema_version": 1,
            "tag": "v0.1.10",
            "decision": "owner_waived",
            "release_state": "UNQUALIFIED PRERELEASE",
            "base_qualification_sha": base,
            "approved_by": "repository-owner",
            "reason": "Owner explicitly authorized publication without unavailable providers.",
            "waived_requirements": [
                "qwen-code live evidence",
                "claude-code live evidence",
                "composite release verdict",
            ],
        }

    def run_verifier(self, root: Path, payload: dict[str, object], source: str, tag: str = "v0.1.10"):
        reports = root / "reports"
        reports.mkdir(exist_ok=True)
        waiver = reports / f"release_owner_waiver_{tag}.json"
        waiver.write_text(json.dumps(payload), encoding="utf-8")
        return subprocess.run(
            [sys.executable, str(self.script), str(waiver), "--tag", tag, "--source-sha", source],
            cwd=root,
            capture_output=True,
            text=True,
        )

    def test_accepts_exact_tag_scoped_ancestor_waiver(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            base, source = self.repo(root)
            result = self.run_verifier(root, self.payload(base), source)
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("UNQUALIFIED PRERELEASE", result.stdout)

    def test_rejects_wrong_tag_or_missing_requirement(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            base, source = self.repo(root)
            payload = self.payload(base)
            payload["waived_requirements"] = ["qwen-code live evidence"]
            result = self.run_verifier(root, payload, source, tag="v0.1.11")
        self.assertNotEqual(0, result.returncode)
        self.assertIn("tag must be", result.stderr)
        self.assertIn("waived_requirements must be exactly", result.stderr)

    def test_rejects_non_ancestor_base(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            _, source = self.repo(root)
            unrelated = "0" * 40
            result = self.run_verifier(root, self.payload(unrelated), source)
        self.assertNotEqual(0, result.returncode)
        self.assertIn("is not an ancestor", result.stderr)

    def test_rejects_over_broad_waiver_payload(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            base, source = self.repo(root)
            payload = self.payload(base)
            payload["waived_requirements"] = ["all release checks"]
            payload["extra_scope"] = "publish anything"
            result = self.run_verifier(root, payload, source)
        self.assertNotEqual(0, result.returncode)
        self.assertIn("unknown waiver fields", result.stderr)
        self.assertIn("waived_requirements must be exactly", result.stderr)

    def test_rejects_noncanonical_waiver_filename(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            base, source = self.repo(root)
            (root / "reports").mkdir(exist_ok=True)
            (root / "reports" / "waiver.json").write_text(json.dumps(self.payload(base)), encoding="utf-8")
            result = subprocess.run(
                [sys.executable, str(self.script), str(root / "reports" / "waiver.json"), "--tag", "v0.1.10", "--source-sha", source],
                cwd=root,
                capture_output=True,
                text=True,
            )
        self.assertNotEqual(0, result.returncode)
        self.assertIn("waiver filename must be", result.stderr)


if __name__ == "__main__":
    unittest.main()
