#!/usr/bin/env python3
"""Validate versioned release-governance evidence and optionally compare GitHub settings."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import sys
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

from yaml_compat import load_yaml_file

MANIFEST_PATH = Path("docs/release-governance.json")
EXPECTED_CONTEXTS = ["backend", "contracts", "golden", "smoke-api", "smoke-cli", "ui"]
EXPECTED_RULES = ["deletion", "non_fast_forward"]
EXPECTED_WAIVER_FIELDS = {
    "schema_version",
    "tag",
    "decision",
    "release_state",
    "waived_requirements",
    "approved_by",
    "reason",
    "base_qualification_sha",
}
SHA_RE = re.compile(r"^[0-9a-f]{40}$")
TAG_RE = re.compile(r"^v[0-9]+\.[0-9]+\.[0-9]+(?:[-.][A-Za-z0-9.-]+)?$")


def load_manifest(repo_root: Path) -> dict[str, Any]:
    path = repo_root / MANIFEST_PATH
    payload = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(payload, dict):
        raise ValueError("release governance manifest must be an object")
    return payload


def _expect(actual: Any, expected: Any, label: str, failures: list[str]) -> None:
    if actual != expected:
        failures.append(f"{label}: expected {expected!r}, got {actual!r}")


def _iso_timestamp(value: Any) -> bool:
    if not isinstance(value, str):
        return False
    try:
        dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return False
    return value.endswith("Z")


def _validate_waivers(repo_root: Path, policy: dict[str, Any], failures: list[str]) -> None:
    waiver_files = sorted((repo_root / "reports").glob("release_owner_waiver_*.json"))
    if not waiver_files:
        failures.append("owner_waiver: no tracked release owner waivers found")
        return
    required_fields = set(policy.get("required_fields", []))
    expected_requirements = policy.get("waived_requirements")
    for path in waiver_files:
        try:
            payload = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as exc:
            failures.append(f"{path}: invalid JSON ({exc})")
            continue
        if not isinstance(payload, dict):
            failures.append(f"{path}: payload must be an object")
            continue
        if set(payload) - EXPECTED_WAIVER_FIELDS:
            failures.append(f"{path}: unknown fields {sorted(set(payload) - EXPECTED_WAIVER_FIELDS)!r}")
        if set(payload) != required_fields:
            failures.append(f"{path}: fields must be exactly {sorted(required_fields)!r}")
        tag = payload.get("tag")
        expected_tag = path.stem.removeprefix("release_owner_waiver_")
        if tag != expected_tag or not isinstance(tag, str) or not TAG_RE.fullmatch(tag):
            failures.append(f"{path}: tag must match its exact versioned filename")
        _expect(payload.get("schema_version"), 1, f"{path}: schema_version", failures)
        _expect(payload.get("decision"), policy.get("decision"), f"{path}: decision", failures)
        _expect(payload.get("release_state"), policy.get("release_state"), f"{path}: release_state", failures)
        _expect(payload.get("waived_requirements"), expected_requirements, f"{path}: waived_requirements", failures)


def validate_manifest(payload: dict[str, Any], repo_root: Path) -> list[str]:
    failures: list[str] = []
    expected_top_level = {
        "schema_version",
        "repository",
        "captured_at",
        "source_sha",
        "required_status_checks",
        "branch_protection",
        "tag_ruleset",
        "release_environment",
        "owner_waiver",
        "evidence_sources",
    }
    unknown = sorted(set(payload) - expected_top_level)
    if unknown:
        failures.append(f"manifest: unknown fields {unknown!r}")
    _expect(payload.get("schema_version"), 1, "manifest.schema_version", failures)
    _expect(payload.get("repository"), "GrinRus/ProvenArch", "manifest.repository", failures)
    if not _iso_timestamp(payload.get("captured_at")):
        failures.append("manifest.captured_at must be an ISO-8601 UTC timestamp")
    if not SHA_RE.fullmatch(str(payload.get("source_sha", ""))):
        failures.append("manifest.source_sha must be a 40-character lowercase Git SHA")

    checks = payload.get("required_status_checks", {})
    if not isinstance(checks, dict):
        failures.append("required_status_checks must be an object")
    else:
        _expect(checks.get("strict"), True, "required_status_checks.strict", failures)
        _expect(checks.get("contexts"), EXPECTED_CONTEXTS, "required_status_checks.contexts", failures)

    protection = payload.get("branch_protection", {})
    if not isinstance(protection, dict):
        failures.append("branch_protection must be an object")
    else:
        _expect(protection.get("enforce_admins"), True, "branch_protection.enforce_admins", failures)
        reviews = protection.get("required_pull_request_reviews", {})
        if not isinstance(reviews, dict):
            failures.append("branch_protection.required_pull_request_reviews must be an object")
        else:
            for key, value in {
                "dismiss_stale_reviews": True,
                "require_code_owner_reviews": False,
                "required_approving_review_count": 0,
            }.items():
                _expect(reviews.get(key), value, f"branch_protection.required_pull_request_reviews.{key}", failures)

    ruleset = payload.get("tag_ruleset", {})
    if not isinstance(ruleset, dict):
        failures.append("tag_ruleset must be an object")
    else:
        for key, value in {
            "name": "protect release tags",
            "target": "tag",
            "enforcement": "active",
            "include": ["refs/tags/v*"],
            "rules": EXPECTED_RULES,
            "bypass_actors": [],
        }.items():
            _expect(ruleset.get(key), value, f"tag_ruleset.{key}", failures)

    environment = payload.get("release_environment", {})
    if not isinstance(environment, dict):
        failures.append("release_environment must be an object")
    else:
        _expect(environment.get("name"), "github-release", "release_environment.name", failures)
        _expect(environment.get("required_reviewers"), ["GrinRus"], "release_environment.required_reviewers", failures)
        _expect(environment.get("prevent_self_review"), False, "release_environment.prevent_self_review", failures)

    policy = payload.get("owner_waiver", {})
    if not isinstance(policy, dict):
        failures.append("owner_waiver must be an object")
    else:
        _expect(policy.get("path_template"), "reports/release_owner_waiver_<tag>.json", "owner_waiver.path_template", failures)
        _expect(policy.get("decision"), "owner_waived", "owner_waiver.decision", failures)
        _expect(policy.get("release_state"), "UNQUALIFIED PRERELEASE", "owner_waiver.release_state", failures)
        _expect(policy.get("unknown_fields_rejected"), True, "owner_waiver.unknown_fields_rejected", failures)
        _expect(
            sorted(policy.get("required_fields", [])),
            sorted(EXPECTED_WAIVER_FIELDS),
            "owner_waiver.required_fields",
            failures,
        )
        _expect(
            policy.get("waived_requirements"),
            ["qwen-code live evidence", "claude-code live evidence", "composite release verdict"],
            "owner_waiver.waived_requirements",
            failures,
        )
        _validate_waivers(repo_root, policy, failures)

    sources = payload.get("evidence_sources")
    if not isinstance(sources, list) or not sources:
        failures.append("evidence_sources must be a non-empty list")

    for context in EXPECTED_CONTEXTS:
        workflow_path = repo_root / ".github" / "workflows" / f"{context}.yml"
        if not workflow_path.exists():
            failures.append(f"required workflow is missing: {workflow_path}")
            continue
        workflow = load_yaml_file(workflow_path)
        if not isinstance(workflow, dict):
            failures.append(f"workflow {context}: expected an object")
            continue
        _expect(workflow.get("name"), context, f"workflow {context}.name", failures)
        triggers = workflow.get("on", workflow.get(True, {}))
        if not isinstance(triggers, dict) or "pull_request" not in triggers:
            failures.append(f"workflow {context}: pull_request trigger is required")

    release_workflow = load_yaml_file(repo_root / ".github" / "workflows" / "release.yml")
    jobs = release_workflow.get("jobs", {}) if isinstance(release_workflow, dict) else {}
    verify_job = jobs.get("verify-release-evidence", {}) if isinstance(jobs, dict) else {}
    release_job = jobs.get("release", {}) if isinstance(jobs, dict) else {}
    if not isinstance(verify_job, dict) or not isinstance(release_job, dict):
        failures.append("release workflow must define verify-release-evidence and release jobs")
    else:
        _expect(release_job.get("needs"), "verify-release-evidence", "release.needs", failures)
        verify_runs = "\n".join(
            step.get("run", "") for step in verify_job.get("steps", []) if isinstance(step, dict)
        )
        for token in ("verify-release-verdict.py", "verify-release-owner-waiver.py", "--tag", "--source-sha"):
            if token not in verify_runs:
                failures.append(f"release verify job is missing {token!r}")
    return failures


def _github_get(url: str, token: str) -> dict[str, Any] | list[Any]:
    request = Request(
        url,
        headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {token}",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    with urlopen(request, timeout=20) as response:
        payload = json.load(response)
    if not isinstance(payload, (dict, list)):
        raise ValueError(f"GitHub response at {url} must be an object or list")
    return payload


def fetch_live_snapshot(repository: str, token: str) -> dict[str, Any]:
    api_root = f"https://api.github.com/repos/{repository}"
    protection = _github_get(f"{api_root}/branches/main/protection", token)
    rulesets = _github_get(f"{api_root}/rulesets?includes_parents=false", token)
    environments = _github_get(f"{api_root}/environments/github-release", token)
    if not isinstance(protection, dict) or not isinstance(rulesets, list) or not isinstance(environments, dict):
        raise ValueError("unexpected GitHub governance response shape")
    ruleset = next((item for item in rulesets if isinstance(item, dict) and item.get("name") == "protect release tags"), None)
    if not isinstance(ruleset, dict) or not ruleset.get("id"):
        raise ValueError("protect release tags ruleset was not found")
    ruleset_detail = _github_get(f"{api_root}/rulesets/{ruleset['id']}", token)
    if not isinstance(ruleset_detail, dict):
        raise ValueError("ruleset detail response must be an object")

    status_checks = protection.get("required_status_checks") or {}
    reviews = protection.get("required_pull_request_reviews") or {}
    ref_name = (ruleset_detail.get("conditions") or {}).get("ref_name") or {}
    rules = ruleset_detail.get("rules") or []
    reviewer_names = sorted(
        reviewer.get("reviewer", {}).get("login", "")
        for rule in environments.get("protection_rules", [])
        if isinstance(rule, dict) and rule.get("type") == "required_reviewers"
        for reviewer in rule.get("reviewers", [])
        if isinstance(reviewer, dict) and isinstance(reviewer.get("reviewer"), dict)
    )
    return {
        "required_status_checks": {
            "strict": status_checks.get("strict"),
            "contexts": sorted(status_checks.get("contexts") or []),
        },
        "branch_protection": {
            "enforce_admins": (protection.get("enforce_admins") or {}).get("enabled"),
            "required_pull_request_reviews": {
                "dismiss_stale_reviews": reviews.get("dismiss_stale_reviews"),
                "require_code_owner_reviews": reviews.get("require_code_owner_reviews"),
                "required_approving_review_count": reviews.get("required_approving_review_count"),
            },
        },
        "tag_ruleset": {
            "name": ruleset_detail.get("name"),
            "target": ruleset_detail.get("target"),
            "enforcement": ruleset_detail.get("enforcement"),
            "include": sorted(ref_name.get("include") or []),
            "rules": sorted(item.get("type") for item in rules if isinstance(item, dict)),
            "bypass_actors": ruleset_detail.get("bypass_actors") or [],
        },
        "release_environment": {
            "name": "github-release",
            "required_reviewers": reviewer_names,
            "prevent_self_review": next(
                (
                    rule.get("prevent_self_review")
                    for rule in environments.get("protection_rules", [])
                    if isinstance(rule, dict) and rule.get("type") == "required_reviewers"
                ),
                None,
            ),
        },
    }


def compare_live(payload: dict[str, Any], live: dict[str, Any]) -> list[str]:
    failures: list[str] = []
    for section in ("required_status_checks", "branch_protection", "tag_ruleset", "release_environment"):
        expected = payload.get(section)
        actual = live.get(section)
        if expected != actual:
            failures.append(f"live {section} differs from versioned evidence: expected {expected!r}, got {actual!r}")
    return failures


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", type=Path, default=MANIFEST_PATH)
    parser.add_argument("--live", action="store_true", help="read and compare current GitHub settings")
    parser.add_argument("--repository", help="override the repository for --live")
    args = parser.parse_args(argv)
    repo_root = Path.cwd()
    try:
        payload = json.loads((repo_root / args.manifest).read_text(encoding="utf-8"))
        if not isinstance(payload, dict):
            raise ValueError("manifest must be an object")
    except (OSError, json.JSONDecodeError, ValueError) as exc:
        print(f"release governance rejected: {exc}", file=sys.stderr)
        return 1
    failures = validate_manifest(payload, repo_root)
    if args.live:
        token = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")
        if not token:
            print("release governance rejected: --live requires GITHUB_TOKEN or GH_TOKEN", file=sys.stderr)
            return 1
        try:
            live = fetch_live_snapshot(args.repository or payload.get("repository", ""), token)
            failures.extend(compare_live(payload, live))
        except (HTTPError, URLError, OSError, ValueError) as exc:
            failures.append(f"live GitHub governance read failed: {exc}")
    if failures:
        for failure in failures:
            print(f"release governance rejected: {failure}", file=sys.stderr)
        return 1
    print(f"release governance accepted: {(repo_root / args.manifest).resolve()}")
    if args.live:
        print("live GitHub settings match versioned evidence (read-only)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
