import copy
import importlib.util
import json
import sys
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO_ROOT / "scripts"))

_MODULE_SPEC = importlib.util.spec_from_file_location(
    "verify_release_governance", REPO_ROOT / "scripts" / "verify-release-governance.py"
)
assert _MODULE_SPEC and _MODULE_SPEC.loader
_MODULE = importlib.util.module_from_spec(_MODULE_SPEC)
_MODULE_SPEC.loader.exec_module(_MODULE)
load_manifest = _MODULE.load_manifest
validate_manifest = _MODULE.validate_manifest


class ReleaseGovernanceTest(unittest.TestCase):
    def test_versioned_governance_snapshot_matches_repository_surfaces(self) -> None:
        self.assertEqual([], validate_manifest(load_manifest(REPO_ROOT), REPO_ROOT))

    def test_required_context_drift_fails_closed(self) -> None:
        payload = copy.deepcopy(load_manifest(REPO_ROOT))
        payload["required_status_checks"]["contexts"] = ["backend", "contracts"]
        failures = validate_manifest(payload, REPO_ROOT)
        self.assertTrue(any("required_status_checks.contexts" in failure for failure in failures))

    def test_ruleset_bypass_drift_fails_closed(self) -> None:
        payload = copy.deepcopy(load_manifest(REPO_ROOT))
        payload["tag_ruleset"]["bypass_actors"] = [{"actor_id": 1}]
        failures = validate_manifest(payload, REPO_ROOT)
        self.assertTrue(any("tag_ruleset.bypass_actors" in failure for failure in failures))

    def test_snapshot_is_json_and_tied_to_current_main_evidence(self) -> None:
        payload = json.loads((REPO_ROOT / "docs/release-governance.json").read_text(encoding="utf-8"))
        self.assertEqual("7b7388deccbea93a5e76a5a54a075f2f0ebf7d00", payload["source_sha"])
        self.assertEqual("GrinRus/ProvenArch", payload["repository"])


if __name__ == "__main__":
    unittest.main()
