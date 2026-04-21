import tempfile
import textwrap
import unittest
from pathlib import Path
import sys


REPO_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO_ROOT / "scripts"))

from yaml_compat import load_yaml_file, load_yaml_text


class YAMLCompatTest(unittest.TestCase):
    def test_load_yaml_text_supports_indentless_sequence_under_mapping_key(self) -> None:
        payload = load_yaml_text(
            textwrap.dedent(
                """\
                version: 1
                sweeps:
                - id: baseline
                  strategy: sequential
                - id: parallel-default
                  strategy: parallel
                profiles:
                - id: single-git_url
                  expected_repo_count: 1
                """
            )
        )
        self.assertEqual(1, payload["version"])
        self.assertEqual("baseline", payload["sweeps"][0]["id"])
        self.assertEqual("parallel", payload["sweeps"][1]["strategy"])
        self.assertEqual("single-git_url", payload["profiles"][0]["id"])

    def test_load_yaml_text_supports_nested_mappings_and_lists(self) -> None:
        payload = load_yaml_text(
            textwrap.dedent(
                """\
                version: 1
                timeout_profile: short-window
                sweeps:
                  - id: baseline
                    strategy: sequential
                  - id: parallel-default
                    strategy: parallel
                profiles:
                  - id: single-git_url
                    expected_repo_count: 1
                    repos_file: ./repos/github/mono-bank-of-anthos.repos.yaml
                """
            )
        )
        self.assertEqual(1, payload["version"])
        self.assertEqual("short-window", payload["timeout_profile"])
        self.assertEqual("baseline", payload["sweeps"][0]["id"])
        self.assertEqual("parallel", payload["sweeps"][1]["strategy"])
        self.assertEqual(1, payload["profiles"][0]["expected_repo_count"])

    def test_load_yaml_file_supports_catalog_like_payload(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / "catalog.yaml"
            path.write_text(
                textwrap.dedent(
                    """\
                    version: 1
                    execution_policies:
                      release-default:
                        release_mode: true
                        providers:
                          - qwen-code
                          - claude-code
                          - codex-code
                    """
                ),
                encoding="utf-8",
            )
            payload = load_yaml_file(path)
        self.assertTrue(payload["execution_policies"]["release-default"]["release_mode"])
        self.assertEqual(
            ["qwen-code", "claude-code", "codex-code"],
            payload["execution_policies"]["release-default"]["providers"],
        )


if __name__ == "__main__":
    unittest.main()
