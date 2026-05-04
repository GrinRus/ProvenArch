import sys
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO_ROOT / "scripts"))

from yaml_compat import load_yaml_file


class ReleaseDistributionTest(unittest.TestCase):
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
        self.assertIn("npm run build --prefix ui", hooks)
        self.assertIn("cp ui/dist/index.html internal/api/ui_dist/index.html", hooks)

    def test_release_workflow_publishes_github_release_on_version_tags(self) -> None:
        workflow = load_yaml_file(REPO_ROOT / ".github" / "workflows" / "release.yml")

        # PyYAML treats the GitHub Actions key "on" as YAML 1.1 boolean True.
        triggers = workflow.get("on", workflow.get(True))
        self.assertEqual(["v*"], triggers["push"]["tags"])

        job = workflow["jobs"]["release"]
        self.assertEqual("ubuntu-latest", job["runs-on"])
        steps = job["steps"]

        uses = [step.get("uses", "") for step in steps]
        self.assertIn("actions/checkout@v4", uses)
        self.assertIn("actions/setup-go@v5", uses)
        self.assertIn("actions/setup-node@v4", uses)
        self.assertIn("goreleaser/goreleaser-action@v6", uses)

        runs = [step.get("run", "") for step in steps]
        self.assertIn("npm ci --prefix ui", runs)
        self.assertIn("make contracts", runs)
        self.assertIn("make test", runs)
        self.assertIn("make lint", runs)

        release_step = next(step for step in steps if step.get("uses") == "goreleaser/goreleaser-action@v6")
        self.assertEqual("release --clean", release_step["with"]["args"])
        self.assertIn("GITHUB_TOKEN", release_step["env"])


if __name__ == "__main__":
    unittest.main()
