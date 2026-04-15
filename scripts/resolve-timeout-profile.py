#!/usr/bin/env python3
"""Resolve canonical ACP timeout profile.

Supports env-only mode and env+workspace mode.
"""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
from typing import Any

KEYS = (
    "step_timeout_sec",
    "heartbeat_sec",
    "pipeline_timeout_sec",
    "pipeline_kill_grace_sec",
    "api_ready_timeout_sec",
    "api_init_timeout_sec",
    "ui_init_poll_timeout_sec",
    "ui_cancel_poll_timeout_sec",
)

DEFAULTS = {
    "step_timeout_sec": 1800,
    "heartbeat_sec": 30,
    "pipeline_timeout_sec": 2400,
    "pipeline_kill_grace_sec": 30,
    "api_ready_timeout_sec": 60,
    "api_init_timeout_sec": 120,
    "ui_init_poll_timeout_sec": 900,
    "ui_cancel_poll_timeout_sec": 420,
}

CANONICAL_ENV = {
    "step_timeout_sec": "ACP_RUNTIME_STEP_TIMEOUT_SEC",
    "heartbeat_sec": "ACP_RUNTIME_HEARTBEAT_SEC",
    "pipeline_timeout_sec": "ACP_PIPELINE_TIMEOUT_SEC",
    "pipeline_kill_grace_sec": "ACP_PIPELINE_KILL_GRACE_SEC",
    "api_ready_timeout_sec": "ACP_API_READY_TIMEOUT_SEC",
    "api_init_timeout_sec": "ACP_API_INIT_TIMEOUT_SEC",
    "ui_init_poll_timeout_sec": "ACP_UI_INIT_POLL_TIMEOUT_SEC",
    "ui_cancel_poll_timeout_sec": "ACP_UI_CANCEL_POLL_TIMEOUT_SEC",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Resolve canonical ACP timeout profile")
    parser.add_argument(
        "--workspace-manifest",
        default="",
        help="Optional workspace.yaml path. When set, precedence is env > workspace > default.",
    )
    parser.add_argument(
        "--format",
        choices=("json", "kv", "line"),
        default="json",
        help="Output format: json payload, effective key=value lines, or one summary line.",
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


def load_workspace_timeouts(workspace_manifest: str) -> dict[str, Any]:
    manifest_path = Path(workspace_manifest).resolve()
    if not manifest_path.is_file():
        raise SystemExit(f"workspace manifest does not exist: {manifest_path}")
    try:
        import yaml  # type: ignore
    except Exception as exc:  # pragma: no cover
        raise SystemExit(f"PyYAML is required for timeout resolution: {exc}")
    payload = yaml.safe_load(manifest_path.read_text(encoding="utf-8")) or {}
    runtime = payload.get("runtime") if isinstance(payload, dict) else {}
    timeouts = runtime.get("timeouts") if isinstance(runtime, dict) else {}
    return timeouts if isinstance(timeouts, dict) else {}


def resolve_profile(workspace_timeouts: dict[str, Any] | None) -> dict[str, dict[str, int | str]]:
    effective: dict[str, int] = {}
    source: dict[str, str] = {}
    for key in KEYS:
        env_value = parse_positive(os.environ.get(CANONICAL_ENV[key], ""))
        if env_value is not None:
            effective[key] = env_value
            source[key] = "env"
            continue
        if workspace_timeouts is not None:
            persisted_value = workspace_timeouts.get(key)
            if isinstance(persisted_value, int) and persisted_value > 0:
                effective[key] = persisted_value
                source[key] = "workspace"
                continue
        effective[key] = DEFAULTS[key]
        source[key] = "default"
    return {"effective": effective, "source": source}


def render_kv(profile: dict[str, dict[str, int | str]]) -> str:
    effective = profile["effective"]
    lines = [f"{key}={effective[key]}" for key in KEYS]
    return "\n".join(lines)


def render_line(profile: dict[str, dict[str, int | str]]) -> str:
    effective = profile["effective"]
    source = profile["source"]
    parts = [f"{key}={effective[key]}({source[key]})" for key in KEYS]
    return " ".join(parts)


def main() -> int:
    args = parse_args()
    workspace_manifest = (args.workspace_manifest or "").strip()
    workspace_timeouts = load_workspace_timeouts(workspace_manifest) if workspace_manifest else None
    profile = resolve_profile(workspace_timeouts)

    if args.format == "kv":
        print(render_kv(profile))
    elif args.format == "line":
        print(render_line(profile))
    else:
        print(json.dumps(profile, ensure_ascii=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
