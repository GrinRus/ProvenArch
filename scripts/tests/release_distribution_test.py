import sys
import unittest
from pathlib import Path
from typing import Iterable


REPO_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO_ROOT / "scripts"))

from yaml_compat import load_yaml_file


class ReleaseDistributionTest(unittest.TestCase):
    def _workflow_steps(self, workflow_name: str) -> Iterable[dict]:
        workflow = load_yaml_file(REPO_ROOT / ".github" / "workflows" / workflow_name)
        for job in workflow["jobs"].values():
            for step in job.get("steps", []):
                yield step

    def test_goreleaser_config_matches_primary_distribution_contract(self) -> None:
        config = load_yaml_file(REPO_ROOT / ".goreleaser.yml")

        self.assertEqual(config["version"], 2)
        self.assertEqual(config["project_name"], "acp")

        builds = config["builds"]
        self.assertEqual(1, len(builds))
        build = builds[0]
        self.assertEqual("./cmd/acp", build["main"])
        self.assertEqual("acp", build["binary"])
        self.assertEqual(["darwin", "linux"], build["goos"])
        self.assertEqual(["amd64", "arm64"], build["goarch"])
        self.assertIn("CGO_ENABLED=0", build["env"])
        self.assertIn("-X main.version={{ .Version }}", build["ldflags"][0])
        self.assertIn("-X main.commit={{ .Commit }}", build["ldflags"][0])
        self.assertIn("-X main.date={{ .Date }}", build["ldflags"][0])

        archives = config["archives"]
        self.assertEqual(1, len(archives))
        archive = archives[0]
        self.assertEqual(["acp"], archive["ids"])
        self.assertNotIn("builds", archive)
        self.assertEqual(["tar.gz"], archive["formats"])
        self.assertEqual("acp_{{ .Os }}_{{ .Arch }}", archive["name_template"])
        self.assertIn("README.md", archive["files"])
        self.assertIn("docs/INSTALL.md", archive["files"])
        self.assertIn("docs/TROUBLESHOOTING.md", archive["files"])

        self.assertEqual("checksums.txt", config["checksum"]["name_template"])
        self.assertIn("changelog", config)
        self.assertEqual("GrinRus", config["release"]["github"]["owner"])
        self.assertEqual("ProvenArch", config["release"]["github"]["name"])

        hooks = config["before"]["hooks"]
        self.assertIn("./scripts/run-npm.sh run build --prefix ui", hooks)
        self.assertIn("cp ui/dist/index.html internal/api/ui_dist/index.html", hooks)
        self.assertIn("sboms", config)
        self.assertEqual("archive", config["sboms"][0]["artifacts"])
        self.assertFalse(config["sboms"][0]["disable"])
        self.assertTrue(config["release"]["prerelease"])

    def test_release_workflow_publishes_github_release_on_version_tags(self) -> None:
        workflow = load_yaml_file(REPO_ROOT / ".github" / "workflows" / "release.yml")

        # PyYAML treats the GitHub Actions key "on" as YAML 1.1 boolean True.
        triggers = workflow.get("on", workflow.get(True))
        self.assertEqual(["v*"], triggers["push"]["tags"])

        job = workflow["jobs"]["release"]
        self.assertEqual("ubuntu-latest", job["runs-on"])
        steps = job["steps"]

        uses = [step.get("uses", "") for step in steps]
        self.assertTrue(any(use.startswith("actions/checkout@") for use in uses))
        self.assertTrue(any(use.startswith("actions/setup-go@") for use in uses))
        self.assertTrue(any(use.startswith("actions/setup-node@") for use in uses))
        self.assertTrue(any(use.startswith("goreleaser/goreleaser-action@") for use in uses))
        self.assertTrue(any(use.startswith("anchore/sbom-action/download-syft@") for use in uses))
        self.assertTrue(any(use.startswith("actions/attest-build-provenance@") for use in uses))

        runs = [step.get("run", "") for step in steps]
        self.assertIn("npm ci --prefix ui", runs)
        self.assertIn("make contracts", runs)
        self.assertIn("make test", runs)
        self.assertIn("make lint", runs)

        release_step = next(step for step in steps if step.get("uses", "").startswith("goreleaser/goreleaser-action@"))
        self.assertEqual("release --clean", release_step["with"]["args"])
        self.assertEqual("~> v2", release_step["with"]["version"])
        self.assertIn("GITHUB_TOKEN", release_step["env"])
        self.assertEqual("github-release", job["environment"])
        self.assertEqual("write", job["permissions"]["contents"])
        self.assertEqual("write", job["permissions"]["id-token"])
        self.assertEqual("write", job["permissions"]["attestations"])

    def test_go_workflows_use_repository_go_version_file(self) -> None:
        workflow_names = (
            "backend.yml",
            "codeql.yml",
            "golden.yml",
            "release.yml",
            "smoke-api.yml",
            "smoke-cli.yml",
        )

        for workflow_name in workflow_names:
            with self.subTest(workflow=workflow_name):
                setup_go_steps = [
                    step
                    for step in self._workflow_steps(workflow_name)
                    if step.get("uses", "").startswith("actions/setup-go@")
                ]
                self.assertTrue(setup_go_steps, workflow_name)
                for step in setup_go_steps:
                    with_config = step.get("with", {})
                    self.assertEqual(".go-version", with_config.get("go-version-file"))
                    self.assertNotIn("go-version", with_config)

    def test_dependency_review_workflow_uses_known_pinned_action_release(self) -> None:
        steps = list(self._workflow_steps("dependency-review.yml"))
        review_step = next(
            step
            for step in steps
            if step.get("uses", "").startswith("actions/dependency-review-action@")
        )
        self.assertEqual(
            "actions/dependency-review-action@595b5aeba73380359d98a5e087f648dbb0edce1b",
            review_step["uses"],
        )
        self.assertEqual("moderate", review_step["with"]["fail-on-severity"])


if __name__ == "__main__":
    unittest.main()
