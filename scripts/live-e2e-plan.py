#!/usr/bin/env python3
"""Emit direct live E2E matrix harness commands from the checked-in catalog."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path
from typing import Any

from yaml_compat import load_yaml_file


ALL_PROVIDERS = ("qwen-code", "claude-code", "codex-code")
PROVIDER_ALIASES = {
    "qwen": "qwen-code",
    "qwen-code": "qwen-code",
    "claude": "claude-code",
    "claude-code": "claude-code",
    "codex": "codex-code",
    "codex-code": "codex-code",
}
FRONTEND_MODES = {"auto", "always", "never", "per_run"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Print direct scripts/full-run-batch-matrix.sh commands for live E2E selectors.",
    )
    parser.add_argument("--mode", required=True, choices=("smoke", "regres", "release"))
    parser.add_argument("--size", required=True, choices=("tiny", "fast", "long", "full"))
    parser.add_argument(
        "--providers",
        default="",
        help="Provider selector: qwen|claude|codex|all|CSV of qwen-code,claude-code,codex-code.",
    )
    parser.add_argument("--run-count", type=int, default=0)
    parser.add_argument("--run-selection", default="")
    parser.add_argument("--frontend-mode", choices=tuple(sorted(FRONTEND_MODES)), default="")
    parser.add_argument("--format", choices=("shell",), default="shell")
    parser.add_argument(
        "--catalog",
        default="examples/e2e-profile-catalog.yaml",
        help="Catalog YAML path, default examples/e2e-profile-catalog.yaml.",
    )
    return parser.parse_args()


def selector_key(mode: str, size: str) -> str:
    return f"{mode}-{size}"


def load_catalog(path: Path) -> dict[str, Any]:
    payload = load_yaml_file(path)
    if not isinstance(payload, dict):
        raise ValueError(f"catalog must be a YAML object: {path}")
    return payload


def selectors_by_key(catalog: dict[str, Any]) -> dict[str, dict[str, Any]]:
    selectors = catalog.get("selectors")
    if not isinstance(selectors, list):
        raise ValueError("catalog is missing selectors[]")
    result: dict[str, dict[str, Any]] = {}
    for raw in selectors:
        if not isinstance(raw, dict):
            continue
        key = str(raw.get("slug", "")).strip()
        if key:
            result[key] = raw
    return result


def normalize_providers(raw: str, default_values: list[Any]) -> list[str]:
    raw = (raw or "").strip()
    if raw == "":
        tokens = [str(value).strip() for value in default_values if str(value).strip()]
    elif raw.lower() == "all":
        return list(ALL_PROVIDERS)
    else:
        tokens = [part.strip() for part in raw.split(",") if part.strip()]

    providers: list[str] = []
    for token in tokens:
        provider = PROVIDER_ALIASES.get(token)
        if provider is None:
            allowed = ", ".join(["all", *PROVIDER_ALIASES.keys()])
            raise ValueError(f"unsupported provider '{token}' (allowed: {allowed})")
        if provider not in providers:
            providers.append(provider)
    if not providers:
        raise ValueError("provider selector resolved to an empty set")
    return providers


def parse_run_selection(run_count: int, selection: str) -> list[int]:
    if run_count <= 0:
        raise ValueError(f"--run-count must be positive, got {run_count}")
    selection = (selection or "").strip().lower()
    if selection in {"", "all"}:
        return list(range(1, run_count + 1))

    values: set[int] = set()
    for token in [part.strip() for part in selection.split(",") if part.strip()]:
        if "-" in token:
            left, right = token.split("-", 1)
            try:
                start = int(left)
                end = int(right)
            except Exception as exc:
                raise ValueError(f"invalid run range token: {token}") from exc
            if start > end:
                raise ValueError(f"invalid descending run range token: {token}")
            values.update(range(start, end + 1))
            continue
        try:
            values.add(int(token))
        except Exception as exc:
            raise ValueError(f"invalid run token: {token}") from exc

    if not values:
        raise ValueError("run selection resolved to an empty set")
    out_of_bounds = [value for value in values if value < 1 or value > run_count]
    if out_of_bounds:
        raise ValueError(f"run index out of bounds: {out_of_bounds[0]} (RUN_COUNT={run_count})")
    return sorted(values)


def matrix_file_to_command_path(catalog_path: Path, matrix_file: str) -> Path:
    raw = Path(matrix_file)
    if raw.is_absolute():
        return raw
    catalog_dir = catalog_path.parent
    return (catalog_dir / raw).resolve()


def repo_relative(path: Path, repo_root: Path) -> str:
    try:
        return str(path.resolve().relative_to(repo_root.resolve()))
    except ValueError:
        return str(path.resolve())


def matrix_backend_runs(matrix_path: Path, providers: list[str], run_indexes: list[int]) -> int:
    payload = load_yaml_file(matrix_path)
    if not isinstance(payload, dict):
        raise ValueError(f"matrix file must be a YAML object: {matrix_path}")
    profiles = payload.get("profiles")
    if not isinstance(profiles, list) or not profiles:
        raise ValueError(f"matrix file must contain profiles[]: {matrix_path}")
    sweeps = payload.get("sweeps")
    sweep_count = len(sweeps) if isinstance(sweeps, list) and sweeps else 1
    return len(profiles) * sweep_count * len(providers) * len(run_indexes)


def build_plan(args: argparse.Namespace) -> dict[str, Any]:
    repo_root = Path(__file__).resolve().parents[1]
    catalog_path = Path(args.catalog)
    if not catalog_path.is_absolute():
        catalog_path = (repo_root / catalog_path).resolve()
    catalog = load_catalog(catalog_path)
    selectors = selectors_by_key(catalog)
    key = selector_key(args.mode, args.size)
    selector = selectors.get(key)
    if selector is None:
        raise ValueError(f"unknown live E2E selector: {key}")

    default_providers = selector.get("default_providers")
    if not isinstance(default_providers, list):
        default_providers = ["qwen-code"] if args.mode in {"smoke", "regres"} else list(ALL_PROVIDERS)
    providers = normalize_providers(args.providers, default_providers)

    provider_policy = str(selector.get("provider_policy", "")).strip()
    if provider_policy == "exactly-one" and len(providers) != 1:
        raise ValueError(f"{key} requires exactly one provider")
    if args.mode == "release":
        if set(providers) != set(ALL_PROVIDERS):
            raise ValueError("release selectors require all providers: qwen-code,claude-code,codex-code")
        providers = list(ALL_PROVIDERS)

    default_run_count = int(selector.get("default_run_count") or 1)
    run_count = args.run_count if args.run_count else default_run_count
    if args.mode == "release" and run_count != 1:
        raise ValueError("release selectors require --run-count 1")
    default_run_selection = str(selector.get("default_run_selection", "")).strip()
    run_selection_raw = args.run_selection.strip() or default_run_selection or "all"
    run_indexes = parse_run_selection(run_count, run_selection_raw)
    if args.mode == "smoke" and (run_count != 1 or run_indexes != [1]):
        raise ValueError("smoke tiny requires one run: --run-count 1 --run-selection 1")

    frontend_mode = args.frontend_mode.strip()
    if not frontend_mode:
        frontend_mode = str(selector.get("default_frontend_mode", "")).strip()
    if frontend_mode and frontend_mode not in FRONTEND_MODES:
        raise ValueError(f"unsupported frontend mode: {frontend_mode}")
    if args.mode == "release" and frontend_mode and frontend_mode != "per_run":
        raise ValueError("release selectors keep official frontend defaults; only per_run is allowed explicitly")

    invocations = selector.get("matrix_invocations")
    if not isinstance(invocations, list) or not invocations:
        raise ValueError(f"selector {key} is missing matrix_invocations[]")

    commands: list[dict[str, Any]] = []
    total_expected = 0
    release_mode = bool(selector.get("release_mode", args.mode == "release"))
    for item in invocations:
        if not isinstance(item, dict):
            raise ValueError(f"selector {key} has invalid matrix invocation")
        matrix_file = str(item.get("matrix_file", "")).strip()
        matrix_id_prefix = str(item.get("matrix_id_prefix", "")).strip()
        if not matrix_file or not matrix_id_prefix:
            raise ValueError(f"selector {key} matrix invocation requires matrix_file and matrix_id_prefix")
        matrix_path = matrix_file_to_command_path(catalog_path, matrix_file)
        if not matrix_path.is_file():
            raise ValueError(f"matrix file does not exist: {matrix_path}")
        matrix_command_path = repo_relative(matrix_path, repo_root)
        expected_runs = matrix_backend_runs(matrix_path, providers, run_indexes)
        total_expected += expected_runs

        env: dict[str, str] = {
            "MATRIX_ID": f"{matrix_id_prefix}-$(date -u +%Y%m%dT%H%M%SZ)",
            "E2E_MATRIX_FILE": matrix_command_path,
            "E2E_MATRIX_RELEASE_MODE": "1" if release_mode else "0",
            "ACP_CLAUDE_CMD_BIN": "claude",
            "ACP_QWEN_CMD_BIN": "qwen",
            "ACP_CODEX_CMD_BIN": "codex",
            "ACP_APPLY_TIMEOUTS_VIA_API": "1",
            "RUN_COUNT": str(run_count),
        }
        if args.mode != "release":
            env["BATCH_PROVIDER_FILTER"] = ",".join(providers)
        if run_selection_raw != "all" or args.mode == "smoke":
            env["BATCH_RUN_SELECTION"] = ",".join(str(value) for value in run_indexes)
        if frontend_mode:
            env["BATCH_FRONTEND_MODE"] = frontend_mode

        commands.append(
            {
                "matrix_file": matrix_command_path,
                "matrix_id_template": env["MATRIX_ID"],
                "release_mode": release_mode,
                "expected_backend_runs": expected_runs,
                "env": env,
                "command": "./scripts/full-run-batch-matrix.sh",
            }
        )

    quality_required = bool(selector.get("quality_required", args.mode in {"regres", "release"}))
    return {
        "selector": {
            "slug": key,
            "mode": args.mode,
            "size": args.size,
            "description": str(selector.get("description", "")).strip(),
            "release_mode": release_mode,
            "diagnostic": bool(selector.get("diagnostic", not release_mode)),
        },
        "selected_providers": providers,
        "run_count": run_count,
        "selected_run_indexes": [str(value) for value in run_indexes],
        "frontend_mode": frontend_mode,
        "target_repo_sets": [str(value) for value in selector.get("target_repo_sets", [])],
        "expected_backend_runs": total_expected,
        "quality": {
            "required": quality_required,
            "run_quality_json": "reports/taskruns/<run_id>-quality.json",
            "batch_quality_report": "quality_report_<batch-id>.md",
            "blocking_failure_class": "quality_gates_failed",
            "artifact_quality_warning_prefix": "artifact_quality:",
        },
        "commands": commands,
    }


def render_shell(plan: dict[str, Any]) -> str:
    rendered: list[str] = []
    for index, command in enumerate(plan["commands"], start=1):
        if len(plan["commands"]) > 1:
            rendered.append(f"# {plan['selector']['slug']} invocation {index}/{len(plan['commands'])}")
        env = command["env"]
        keys = list(env.keys())
        for key in keys:
            rendered.append(f"{key}={env[key]} \\")
        rendered.append(str(command["command"]))
        rendered.append("")
    return "\n".join(rendered).rstrip() + "\n"


def main() -> int:
    args = parse_args()
    try:
        plan = build_plan(args)
    except Exception as exc:
        print(f"live-e2e-plan: {exc}", file=sys.stderr)
        return 2

    print(render_shell(plan), end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
