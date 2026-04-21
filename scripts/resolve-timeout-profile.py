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

import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
from yaml_compat import load_yaml_file

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

PRESETS = {
    "short-window": {
        "step_timeout_sec": 3600,
        "heartbeat_sec": 30,
        "pipeline_timeout_sec": 7200,
        "pipeline_kill_grace_sec": 30,
        "api_ready_timeout_sec": 60,
        "api_init_timeout_sec": 120,
        "ui_init_poll_timeout_sec": 1200,
        "ui_cancel_poll_timeout_sec": 420,
    },
    "medium-window": {
        "step_timeout_sec": 5400,
        "heartbeat_sec": 30,
        "pipeline_timeout_sec": 14400,
        "pipeline_kill_grace_sec": 30,
        "api_ready_timeout_sec": 60,
        "api_init_timeout_sec": 120,
        "ui_init_poll_timeout_sec": 1500,
        "ui_cancel_poll_timeout_sec": 420,
    },
    "extended-window": {
        "step_timeout_sec": 10800,
        "heartbeat_sec": 30,
        "pipeline_timeout_sec": 21600,
        "pipeline_kill_grace_sec": 30,
        "api_ready_timeout_sec": 60,
        "api_init_timeout_sec": 120,
        "ui_init_poll_timeout_sec": 1800,
        "ui_cancel_poll_timeout_sec": 420,
    },
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
        "--preset",
        choices=tuple(PRESETS.keys()),
        default="",
        help="Optional canonical live E2E timeout preset. Precedence is env > preset > workspace > default.",
    )
    parser.add_argument(
        "--format",
        choices=("json", "kv", "env-kv", "line"),
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
    payload = load_yaml_file(manifest_path) or {}
    runtime = payload.get("runtime") if isinstance(payload, dict) else {}
    timeouts = runtime.get("timeouts") if isinstance(runtime, dict) else {}
    return timeouts if isinstance(timeouts, dict) else {}


def resolve_profile(
    workspace_timeouts: dict[str, Any] | None,
    preset_name: str,
) -> dict[str, dict[str, int | str]]:
    effective: dict[str, int] = {}
    source: dict[str, str] = {}
    preset = PRESETS.get(preset_name, {})
    for key in KEYS:
        env_value = parse_positive(os.environ.get(CANONICAL_ENV[key], ""))
        if env_value is not None:
            effective[key] = env_value
            source[key] = "env"
            continue
        preset_value = preset.get(key)
        if isinstance(preset_value, int) and preset_value > 0:
            effective[key] = preset_value
            source[key] = f"preset:{preset_name}"
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


def render_env_kv(profile: dict[str, dict[str, int | str]]) -> str:
    effective = profile["effective"]
    lines = [f"{CANONICAL_ENV[key]}={effective[key]}" for key in KEYS]
    return "\n".join(lines)


def render_line(profile: dict[str, dict[str, int | str]]) -> str:
    effective = profile["effective"]
    source = profile["source"]
    parts = [f"{key}={effective[key]}({source[key]})" for key in KEYS]
    return " ".join(parts)


def main() -> int:
    args = parse_args()
    workspace_manifest = (args.workspace_manifest or "").strip()
    preset_name = (args.preset or "").strip()
    workspace_timeouts = load_workspace_timeouts(workspace_manifest) if workspace_manifest else None
    profile = resolve_profile(workspace_timeouts, preset_name)

    if args.format == "kv":
        print(render_kv(profile))
    elif args.format == "env-kv":
        print(render_env_kv(profile))
    elif args.format == "line":
        print(render_line(profile))
    else:
        print(json.dumps(profile, ensure_ascii=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
