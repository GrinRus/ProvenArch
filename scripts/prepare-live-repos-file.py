#!/usr/bin/env python3
"""Prepare a run-local repos file for trusted-machine live E2E.

Canonical path checkouts remain host prerequisites and source-of-truth inputs,
but live provider runtimes must not operate directly on those checkouts.
This helper verifies each declared path checkout, creates a local detached clone
under the batch temp root, and writes a generated repos file that points ACP at
the isolated clone.
"""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from yaml_compat import load_yaml_file


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Create an isolated live E2E repos file")
    parser.add_argument("--repos-file", required=True, help="Input repos YAML file")
    parser.add_argument("--work-dir", required=True, help="Directory for run-local repo clones")
    parser.add_argument("--out", required=True, help="Path to write generated repos YAML")
    parser.add_argument(
        "--make-read-only",
        action="store_true",
        help="Remove write bits from isolated path checkouts after cloning",
    )
    return parser.parse_args()


def run_git(cwd: Path, *args: str) -> str:
    try:
        return subprocess.check_output(["git", "-C", str(cwd), *args], text=True, stderr=subprocess.STDOUT).strip()
    except subprocess.CalledProcessError as exc:
        output = (exc.output or "").strip()
        detail = f": {output}" if output else ""
        raise SystemExit(f"git -C {cwd} {' '.join(args)} failed{detail}") from None


def run_cmd(args: list[str]) -> str:
    try:
        return subprocess.check_output(args, text=True, stderr=subprocess.STDOUT).strip()
    except subprocess.CalledProcessError as exc:
        output = (exc.output or "").strip()
        detail = f": {output}" if output else ""
        raise SystemExit(f"{' '.join(args)} failed{detail}") from None


def load_repos_payload(repos_file: Path) -> tuple[Any, list[dict[str, Any]]]:
    payload = load_yaml_file(repos_file)
    if isinstance(payload, list):
        repos = payload
    elif isinstance(payload, dict):
        repos = payload.get("repos")
    else:
        repos = None
    if not isinstance(repos, list) or not repos:
        raise SystemExit(f"repos file {repos_file} must contain non-empty repos[]")
    normalized: list[dict[str, Any]] = []
    for idx, item in enumerate(repos, start=1):
        if not isinstance(item, dict):
            raise SystemExit(f"repos[{idx}] must be an object")
        normalized.append(dict(item))
    return payload, normalized


def slugify(value: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-")
    return slug or "repo"


def make_tree_writable(root: Path) -> None:
    if not root.exists():
        return
    for path in sorted(root.rglob("*"), reverse=True):
        try:
            if path.is_symlink():
                continue
            mode = path.stat().st_mode
            path.chmod(mode | stat.S_IWUSR)
        except OSError:
            pass
    try:
        root.chmod(root.stat().st_mode | stat.S_IWUSR)
    except OSError:
        pass


def make_tree_read_only(root: Path) -> None:
    for path in sorted(root.rglob("*"), reverse=True):
        try:
            if path.is_symlink():
                continue
            mode = path.stat().st_mode
            path.chmod(mode & ~stat.S_IWUSR & ~stat.S_IWGRP & ~stat.S_IWOTH)
        except OSError:
            pass
    try:
        mode = root.stat().st_mode
        root.chmod(mode & ~stat.S_IWUSR & ~stat.S_IWGRP & ~stat.S_IWOTH)
    except OSError:
        pass


def resolve_path_entry(repos_file: Path, idx: int, repo: dict[str, Any]) -> tuple[Path, str]:
    path_raw = str(repo.get("path", "")).strip()
    ref = str(repo.get("ref", "")).strip()
    if not path_raw:
        raise SystemExit(f"repos[{idx}] path entry is missing path")
    source = Path(path_raw)
    source = (repos_file.parent / source).resolve() if not source.is_absolute() else source.resolve()
    if not source.exists():
        raise SystemExit(f"repos[{idx}] path does not exist: {source}")
    if not source.is_dir():
        raise SystemExit(f"repos[{idx}] path is not a directory: {source}")
    inside = run_git(source, "rev-parse", "--is-inside-work-tree")
    if inside != "true":
        raise SystemExit(f"repos[{idx}] path is not a readable git checkout: {source}")
    head = run_git(source, "rev-parse", "HEAD")
    checkout_ref = head
    if ref:
        checkout_ref = run_git(source, "rev-parse", "--verify", f"{ref}^{{commit}}")
        if re.fullmatch(r"[0-9a-fA-F]{40}", ref) and head != ref:
            raise SystemExit(f"repos[{idx}] path SHA mismatch: {source} expected={ref} got={head}")
    return source, checkout_ref


def clone_path_entry(source: Path, checkout_ref: str, dest: Path, make_read_only: bool) -> None:
    if dest.exists():
        make_tree_writable(dest)
        shutil.rmtree(dest)
    dest.parent.mkdir(parents=True, exist_ok=True)
    run_cmd(["git", "clone", "--shared", "--no-checkout", str(source), str(dest)])
    run_git(dest, "checkout", "--detach", checkout_ref)
    actual = run_git(dest, "rev-parse", "HEAD")
    if actual != checkout_ref:
        raise SystemExit(f"isolated checkout SHA mismatch: {dest} expected={checkout_ref} got={actual}")
    run_git(dest, "status", "--porcelain", "--untracked-files=all")
    if make_read_only:
        make_tree_read_only(dest)
        # Verify read-only permissions did not break ordinary git inspection.
        run_git(dest, "status", "--porcelain", "--untracked-files=all")


def yaml_string(value: str) -> str:
    return json.dumps(value, ensure_ascii=True)


def write_repos_yaml(out_path: Path, repos: list[dict[str, str]]) -> None:
    lines = ["version: 1", "repos:"]
    for repo in repos:
        lines.append(f"  - name: {yaml_string(repo['name'])}")
        if repo.get("path"):
            lines.append(f"    path: {yaml_string(repo['path'])}")
        if repo.get("git_url"):
            lines.append(f"    git_url: {yaml_string(repo['git_url'])}")
        if repo.get("ref"):
            lines.append(f"    ref: {yaml_string(repo['ref'])}")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    args = parse_args()
    repos_file = Path(args.repos_file).resolve()
    work_dir = Path(args.work_dir).resolve()
    out_path = Path(args.out).resolve()

    _, repos = load_repos_payload(repos_file)
    generated: list[dict[str, str]] = []
    for idx, repo in enumerate(repos, start=1):
        name = str(repo.get("name", "")).strip()
        if not name:
            raise SystemExit(f"repos[{idx}] is missing name")
        path_raw = str(repo.get("path", "")).strip()
        git_url = str(repo.get("git_url", "")).strip()
        if bool(path_raw) == bool(git_url):
            raise SystemExit(f"repos[{idx}] must set exactly one of path or git_url")
        ref = str(repo.get("ref", "")).strip()
        if git_url:
            generated.append({"name": name, "git_url": git_url, "ref": ref})
            continue
        source, checkout_ref = resolve_path_entry(repos_file, idx, repo)
        dest = work_dir / f"{idx:02d}-{slugify(name)}"
        clone_path_entry(source, checkout_ref, dest, args.make_read_only)
        generated.append({"name": name, "path": str(dest), "ref": checkout_ref})

    write_repos_yaml(out_path, generated)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
