import pathlib
import re
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]


class AORLiveBoundaryTest(unittest.TestCase):
    def test_live_release_identity_does_not_enter_product_sources(self) -> None:
        forbidden_literals = (
            "swe_ux_assessment",
            "swe_artifact_quality_assessment",
            "release_verdict",
            "matrix id",
            "batch id",
            "profile id",
            "matrix_id",
            "batch_id",
            "profile_id",
        )
        core_dirs = (
            REPO_ROOT / "cmd" / "acp",
            REPO_ROOT / "internal" / "artifactquality",
            REPO_ROOT / "internal" / "contracts",
            REPO_ROOT / "internal" / "orchestrator",
            REPO_ROOT / "internal" / "runtime",
            REPO_ROOT / "ui" / "src",
        )
        offenders: list[str] = []
        for root in core_dirs:
            for path in root.rglob("*"):
                if path.suffix not in {".go", ".ts", ".tsx"}:
                    continue
                if ".test." in path.name or path.name.endswith("_test.go"):
                    continue
                rel = path.relative_to(REPO_ROOT)
                text = path.read_text(encoding="utf-8", errors="replace").lower()
                for token in forbidden_literals:
                    if token in text:
                        offenders.append(f"{rel}: contains {token}")

        self.assertEqual([], offenders)

    def test_live_harness_does_not_import_or_author_product_state(self) -> None:
        batch = (REPO_ROOT / "scripts" / "full-run-batch.sh").read_text(encoding="utf-8")
        live_flow = (REPO_ROOT / "ui" / "e2e" / "live-flow.spec.ts").read_text(encoding="utf-8")
        frontend = (REPO_ROOT / "scripts" / "frontend-live-e2e.sh").read_text(encoding="utf-8")

        self.assertNotIn("prepare_frontend_snapshot_run_history", batch)
        self.assertNotRegex(batch, re.compile(r"run-history\\.json[^\\n]*(write_text|json\\.dump)"))
        self.assertNotIn('from "../src/', live_flow)
        self.assertNotIn("internal/artifactquality", live_flow)
        self.assertNotRegex(frontend, re.compile(r"run-history\\.json[^\\n]*(write_text|json\\.dump)"))


if __name__ == "__main__":
    unittest.main()
