import pathlib
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]


class AORLiveBoundaryTest(unittest.TestCase):
    def test_live_release_evidence_names_do_not_enter_core_aor_logic(self) -> None:
        forbidden = (
            "swe_ux_assessment",
            "swe_artifact_quality_assessment",
            "release_verdict",
            "execution_report",
        )
        core_dirs = (
            REPO_ROOT / "internal" / "artifactquality",
            REPO_ROOT / "internal" / "contracts",
            REPO_ROOT / "internal" / "orchestrator",
            REPO_ROOT / "internal" / "runtime",
        )
        offenders: list[str] = []
        for root in core_dirs:
            for path in root.rglob("*.go"):
                if path.name.endswith("_test.go"):
                    continue
                rel = path.relative_to(REPO_ROOT)
                text = path.read_text(encoding="utf-8", errors="replace")
                for token in forbidden:
                    if token in text:
                        offenders.append(f"{rel}: contains {token}")

        self.assertEqual([], offenders)


if __name__ == "__main__":
    unittest.main()
