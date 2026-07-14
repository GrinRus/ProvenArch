import json
import unittest
from pathlib import Path


class UICoverageContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.repo_root = Path(__file__).resolve().parents[2]

    def test_coverage_dependency_and_script_are_locked(self) -> None:
        package_json = json.loads((self.repo_root / "ui" / "package.json").read_text(encoding="utf-8"))
        package_lock = json.loads((self.repo_root / "ui" / "package-lock.json").read_text(encoding="utf-8"))

        self.assertEqual("vitest run --coverage", package_json["scripts"].get("coverage"))
        self.assertEqual("4.1.5", package_json["devDependencies"].get("@vitest/coverage-v8"))
        self.assertEqual("4.1.5", package_lock["packages"][""]["devDependencies"].get("@vitest/coverage-v8"))
        self.assertIn("node_modules/@vitest/coverage-v8", package_lock["packages"])

    def test_vitest_v8_coverage_includes_ui_src_without_threshold_gate(self) -> None:
        config = (self.repo_root / "ui" / "vite.config.ts").read_text(encoding="utf-8")

        self.assertIn('provider: "v8"', config)
        self.assertIn('reporter: ["text", "json-summary", "json"]', config)
        self.assertIn('include: ["src/**/*.{ts,tsx}"]', config)
        self.assertIn('exclude: ["src/**/*.test.{ts,tsx}", "src/test/**", "src/**/*.d.ts"]', config)
        self.assertNotIn("thresholds", config)


if __name__ == "__main__":
    unittest.main()
