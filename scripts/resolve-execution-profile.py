#!/usr/bin/env python3
"""Resolve canonical ACP execution profile for batch runs."""

from __future__ import annotations

import argparse
import json
import os

DEFAULTS = {
    "strategy": "sequential",
    "max_parallel_tasks": 1,
    "failure_policy": "best_effort",
    "shard_discovery_mode": "heuristics",
}

ALLOWED = {
    "strategy": {"sequential", "parallel"},
    "failure_policy": {"fail_fast", "best_effort"},
    "shard_discovery_mode": {"heuristics", "semantic"},
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Resolve canonical ACP execution profile")
    parser.add_argument(
        "--format",
        choices=("json", "line"),
        default="json",
        help="Output format: json payload or one summary line.",
    )
    return parser.parse_args()


def parse_positive(raw: str) -> int | None:
    text = (raw or "").strip()
    if not text:
        return None
    try:
        value = int(text)
    except ValueError:
        return None
    if value <= 0:
        return None
    return value


def resolve_profile() -> dict[str, dict[str, int | str]]:
    effective: dict[str, int | str] = {}
    source: dict[str, str] = {}

    strategy_raw = (os.environ.get("ACP_EXECUTION_STRATEGY", "") or "").strip()
    strategy = strategy_raw if strategy_raw in ALLOWED["strategy"] else DEFAULTS["strategy"]
    source["strategy"] = "env" if strategy_raw in ALLOWED["strategy"] else "default"
    effective["strategy"] = strategy

    max_raw = parse_positive(os.environ.get("ACP_MAX_PARALLEL_TASKS", ""))
    max_parallel = max_raw if max_raw is not None else int(DEFAULTS["max_parallel_tasks"])
    max_source = "env" if max_raw is not None else "default"
    if strategy != "parallel":
        effective["max_parallel_tasks"] = 1
        if max_parallel != 1:
            max_source = f"{max_source}->effective(strategy={strategy})"
    else:
        effective["max_parallel_tasks"] = max_parallel
    source["max_parallel_tasks"] = max_source

    failure_policy_raw = (os.environ.get("ACP_FAILURE_POLICY", "") or "").strip()
    failure_policy = (
        failure_policy_raw if failure_policy_raw in ALLOWED["failure_policy"] else str(DEFAULTS["failure_policy"])
    )
    source["failure_policy"] = "env" if failure_policy_raw in ALLOWED["failure_policy"] else "default"
    effective["failure_policy"] = failure_policy

    shard_mode_raw = (os.environ.get("ACP_SHARD_DISCOVERY_MODE", "") or "").strip()
    shard_mode = shard_mode_raw if shard_mode_raw in ALLOWED["shard_discovery_mode"] else str(DEFAULTS["shard_discovery_mode"])
    source["shard_discovery_mode"] = "env" if shard_mode_raw in ALLOWED["shard_discovery_mode"] else "default"
    effective["shard_discovery_mode"] = shard_mode

    return {"effective": effective, "source": source}


def render_line(profile: dict[str, dict[str, int | str]]) -> str:
    effective = profile["effective"]
    source = profile["source"]
    keys = ("strategy", "max_parallel_tasks", "failure_policy", "shard_discovery_mode")
    parts = [f"{key}={effective[key]}({source[key]})" for key in keys]
    return " ".join(parts)


def main() -> int:
    args = parse_args()
    profile = resolve_profile()
    if args.format == "line":
        print(render_line(profile))
    else:
        print(json.dumps(profile, ensure_ascii=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
