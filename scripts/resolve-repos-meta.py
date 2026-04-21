#!/usr/bin/env python3
"""Resolve repos-file metadata used by full-run/batch scripts.

Outputs normalized JSON with validated declared repos and effective source kind.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
from yaml_compat import load_yaml_file


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Resolve/validate TARGET_REPOS_FILE metadata")
    parser.add_argument("--repos-file", required=True, help="Path to repos YAML file")
    parser.add_argument("--expected-repo-count", default="", help="Optional expected repo count")
    parser.add_argument("--source-kind", default="", help="Optional source kind override: path|git_url")
    parser.add_argument("--profile-id", default="", help="Optional profile id")
    parser.add_argument("--out", required=True, help="Path to write normalized metadata JSON")
    return parser.parse_args()


def load_repos_payload(repos_file: Path) -> list[dict[str, Any]]:
    payload = load_yaml_file(repos_file)
    repos: Any
    if isinstance(payload, list):
        repos = payload
    elif isinstance(payload, dict):
        repos = payload.get("repos")
    else:
        repos = None
    if not isinstance(repos, list) or not repos:
        raise SystemExit(f"repos file {repos_file} must contain non-empty repos[]")
    return repos


def normalize_declared_repos(repos_file: Path, repos: list[dict[str, Any]], source_kind: str) -> list[dict[str, Any]]:
    declared: list[dict[str, Any]] = []
    for idx, item in enumerate(repos, start=1):
        if not isinstance(item, dict):
            raise SystemExit(f"repos[{idx}] must be an object")
        name = str(item.get("name", "")).strip()
        if not name:
            raise SystemExit(f"repos[{idx}] is missing name")
        path_raw = str(item.get("path", "")).strip()
        git_url = str(item.get("git_url", "")).strip()
        has_path = bool(path_raw)
        has_git = bool(git_url)
        if has_path == has_git:
            raise SystemExit(f"repos[{idx}] must set exactly one of path or git_url")
        ref = str(item.get("ref", "")).strip()
        if has_path:
            source = "path"
            path_value = Path(path_raw)
            abs_path = (repos_file.parent / path_value).resolve() if not path_value.is_absolute() else path_value.resolve()
            if not abs_path.exists():
                raise SystemExit(f"repos[{idx}] path does not exist: {abs_path}")
            if not abs_path.is_dir():
                raise SystemExit(f"repos[{idx}] path is not a directory: {abs_path}")
            entry = {"name": name, "source": source, "path": str(abs_path), "ref": ref}
        else:
            source = "git_url"
            entry = {"name": name, "source": source, "git_url": git_url, "ref": ref}
        if source_kind == "path" and source != "path":
            raise SystemExit(f"profile source_kind=path but repos[{idx}] uses git_url")
        if source_kind == "git_url" and source != "git_url":
            raise SystemExit(f"profile source_kind=git_url but repos[{idx}] uses path")
        declared.append(entry)
    return declared


def detect_effective_source_kind(source_kind: str, declared: list[dict[str, Any]]) -> str:
    if source_kind:
        return source_kind
    declared_sources = {str(item.get("source", "")).strip() for item in declared}
    if len(declared_sources) == 1:
        return next(iter(declared_sources))
    return "mixed"


def ensure_git_url_refs_pinned(effective_source_kind: str, declared: list[dict[str, Any]]) -> None:
    if effective_source_kind != "git_url":
        return
    for idx, repo in enumerate(declared, start=1):
        source = str(repo.get("source", "")).strip()
        ref = str(repo.get("ref", "")).strip()
        if source != "git_url":
            raise SystemExit(f"profile source_kind=git_url but repos[{idx}] uses path")
        if not ref:
            raise SystemExit(f"repos[{idx}] git_url entry must have pinned ref for source_kind=git_url")


def resolve_expected_count(raw: str, declared: list[dict[str, Any]], repos_file: Path) -> int:
    expected_raw = (raw or "").strip()
    if not expected_raw:
        return len(declared)
    try:
        expected_count = int(expected_raw)
    except ValueError:
        raise SystemExit(f"EXPECTED_REPO_COUNT must be an integer, got: {expected_raw}") from None
    if expected_count <= 0:
        raise SystemExit(f"EXPECTED_REPO_COUNT must be > 0, got: {expected_count}")
    if len(declared) != expected_count:
        raise SystemExit(f"expected {expected_count} repos but got {len(declared)} in {repos_file}")
    return expected_count


def detect_target_profile(declared: list[dict[str, Any]]) -> str:
    target_profile = "generic"
    for repo in declared:
        haystack = " ".join(
            part for part in [repo.get("name", ""), repo.get("path", ""), repo.get("git_url", ""), repo.get("ref", "")] if part
        ).lower()
        if "ai_advent_challenge_new" in haystack:
            target_profile = "ai-advent"
            break
    return target_profile


def main() -> int:
    args = parse_args()
    source_kind = (args.source_kind or "").strip()
    if source_kind not in {"", "path", "git_url"}:
        raise SystemExit(f"PROFILE_SOURCE_KIND must be one of path|git_url, got: {source_kind}")

    repos_file = Path(args.repos_file).resolve()
    out_path = Path(args.out).resolve()
    profile_id = (args.profile_id or "").strip() or "adhoc"

    repos = load_repos_payload(repos_file)
    declared = normalize_declared_repos(repos_file, repos, source_kind)
    effective_source_kind = detect_effective_source_kind(source_kind, declared)
    ensure_git_url_refs_pinned(effective_source_kind, declared)
    expected_count = resolve_expected_count(args.expected_repo_count, declared, repos_file)

    payload = {
        "repos_file": str(repos_file),
        "target_repos_file": str(repos_file),
        "profile_id": profile_id,
        "profile_source_kind": effective_source_kind,
        "expected_repo_count": expected_count,
        "target_profile": detect_target_profile(declared),
        "declared_repos": declared,
    }
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(payload, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
