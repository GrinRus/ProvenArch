#!/usr/bin/env python3
"""Run one live-E2E pipeline through the public Task/Attempt API.

This helper is intentionally a black-box harness boundary: it starts the public ACP server,
creates a product Task, admits one immutable Attempt, polls that Attempt, and writes the same
small CLI-shaped transcript consumed by the existing backend-cycle reporter. It never imports
product Go packages and never fabricates a Task for an existing legacy run.
"""

from __future__ import annotations

import argparse
import json
import os
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request
import uuid
from pathlib import Path
from typing import Any


def request_json(base_url: str, method: str, path: str, payload: dict[str, Any] | None = None) -> dict[str, Any]:
    data = None
    headers: dict[str, str] = {}
    if payload is not None:
        data = json.dumps(payload, ensure_ascii=True).encode("utf-8")
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(f"{base_url}{path}", data=data, headers=headers, method=method)
    with urllib.request.urlopen(request, timeout=5) as response:
        body = response.read().decode("utf-8")
    decoded = json.loads(body)
    if not isinstance(decoded, dict):
        raise RuntimeError(f"public API returned non-object JSON for {path}")
    return decoded


def wait_for_health(base_url: str, deadline: float) -> None:
    last_error = "unavailable"
    while time.monotonic() < deadline:
        try:
            payload = request_json(base_url, "GET", "/api/health")
            if str(payload.get("status") or payload.get("state") or "").strip().lower() in {"ok", "healthy", "ready"}:
                return
            last_error = json.dumps(payload, ensure_ascii=True)
        except Exception as exc:  # pragma: no cover - exercised by live process startup
            last_error = str(exc)
        time.sleep(0.25)
    raise RuntimeError(f"public ACP server did not become healthy: {last_error}")


def terminate_process(process: subprocess.Popen[str] | None) -> None:
    if process is None or process.poll() is not None:
        return
    try:
        process.send_signal(signal.SIGTERM)
        process.wait(timeout=10)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=5)


def write_transcript(path: Path, values: dict[str, str]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    lines = [f"{key}: {value}" for key, value in values.items()]
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def resolve_pipeline_timeout_sec(explicit: int | None) -> int:
    value = explicit
    if value is None:
        raw = os.environ.get("PIPELINE_TIMEOUT_SEC", "1800")
        try:
            value = int(raw)
        except ValueError as exc:
            raise RuntimeError(f"pipeline timeout must be an integer, got {raw!r}") from exc
    if value <= 0:
        raise RuntimeError(f"pipeline timeout must be positive, got {value}")
    return value


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--acp-bin", required=True)
    parser.add_argument("--workspace", required=True)
    parser.add_argument("--pipeline", choices=("init", "refresh"), required=True)
    parser.add_argument("--runtime", choices=("fake", "headless"), required=True)
    parser.add_argument("--runtime-provider", required=True)
    parser.add_argument("--listen", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--server-log", required=True)
    parser.add_argument("--api-ready-timeout-sec", type=int, default=120)
    parser.add_argument(
        "--pipeline-timeout-sec",
        type=int,
        default=None,
        help="Effective pipeline polling deadline supplied by the backend-cycle harness.",
    )
    parser.add_argument("--poll-sec", type=float, default=0.25)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    workspace = str(Path(args.workspace).resolve())
    base_url = f"http://{args.listen}"
    server_log = Path(args.server_log)
    output = Path(args.output)
    server_log.parent.mkdir(parents=True, exist_ok=True)

    server_command = [
        args.acp_bin,
        "serve",
        "--workspace",
        workspace,
        "--runtime",
        args.runtime,
        "--runtime-provider",
        args.runtime_provider,
        "--listen",
        args.listen,
        "--run-logs-ttl-hours",
        os.environ.get("RUN_LOGS_TTL_HOURS", "168"),
        "--run-logs-max-runs",
        os.environ.get("RUN_LOGS_MAX_RUNS", "200"),
    ]
    server: subprocess.Popen[str] | None = None
    run_id = ""
    task_id = ""
    attempt_id = ""
    terminal_status = "failed"
    error_message = ""
    try:
        pipeline_timeout_sec = resolve_pipeline_timeout_sec(args.pipeline_timeout_sec)
        with server_log.open("w", encoding="utf-8") as log_file:
            server = subprocess.Popen(
                server_command,
                stdout=log_file,
                stderr=subprocess.STDOUT,
                text=True,
            )
        wait_for_health(base_url, time.monotonic() + max(1, args.api_ready_timeout_sec))
        validation = request_json(base_url, "POST", "/api/workspace/validate")
        resolved_repos = validation.get("resolved_repos")
        if not isinstance(resolved_repos, list) or not resolved_repos:
            raise RuntimeError("public workspace validation returned no resolved repositories")
        repositories = []
        for repo in resolved_repos:
            if not isinstance(repo, dict) or not str(repo.get("name") or "").strip():
                raise RuntimeError("public workspace validation returned an invalid repository identity")
            repositories.append({"name": str(repo["name"]).strip(), "paths": ["."]})

        task_response = request_json(
            base_url,
            "POST",
            "/api/tasks",
            {
                "title": f"Live E2E {args.pipeline} Task",
                "goal": "Inspect the exact immutable Attempt outcome and public runtime evidence.",
                "context": "Canonical trusted-machine live E2E Task-first admission.",
                "scope": {"repositories": repositories},
                "desired_runner": {
                    "preset": f"{args.runtime_provider}-default",
                    "mode": args.runtime,
                    "provider": args.runtime_provider,
                },
            },
        )
        task = task_response.get("task")
        if not isinstance(task, dict) or not str(task.get("task_id") or "").strip():
            raise RuntimeError("public Task admission returned no opaque task_id")
        task_id = str(task["task_id"]).strip()
        idempotency_key = f"live-e2e-{args.pipeline}-{uuid.uuid4().hex}"
        attempt_response = request_json(
            base_url,
            "POST",
            f"/api/tasks/{task_id}/attempts",
            {"idempotency_key": idempotency_key, "pipeline": args.pipeline, "intent": "start"},
        )
        attempt = attempt_response.get("attempt")
        if not isinstance(attempt, dict):
            raise RuntimeError("public Attempt admission returned no attempt")
        attempt_id = str(attempt.get("attempt_id") or "").strip()
        run_id = str(attempt.get("run_id") or "").strip()
        if not attempt_id or not run_id or str(attempt.get("task_id") or "").strip() != task_id:
            raise RuntimeError("public Attempt admission returned an inconsistent Task/Attempt/run join")

        deadline = time.monotonic() + pipeline_timeout_sec
        while time.monotonic() < deadline:
            current_response = request_json(base_url, "GET", f"/api/tasks/{task_id}/attempts/{attempt_id}")
            current = current_response.get("attempt")
            if not isinstance(current, dict):
                raise RuntimeError("public Attempt read returned no attempt")
            current_run_id = str(current.get("run_id") or "").strip()
            if current_run_id != run_id or str(current.get("task_id") or "").strip() != task_id:
                raise RuntimeError("public Attempt identity changed while polling")
            terminal_status = str(current.get("status") or "").strip()
            if terminal_status in {"succeeded", "failed", "canceled"}:
                if terminal_status != "succeeded":
                    error_message = str(current.get("error") or current.get("terminal_summary") or "attempt failed")
                break
            time.sleep(max(0.05, args.poll_sec))
        else:
            error_message = f"Attempt did not reach terminal state within PIPELINE_TIMEOUT_SEC={pipeline_timeout_sec}"
            try:
                request_json(base_url, "POST", f"/api/pipeline/runs/{run_id}/cancel", {})
            except Exception:
                pass
            terminal_status = "failed"

        write_transcript(
            output,
            {
                "workspace": workspace,
                "run_id": run_id,
                "task_id": task_id,
                "attempt_id": attempt_id,
                "pipeline": args.pipeline,
                "status": terminal_status,
                "runtime mode": args.runtime,
                "runtime provider": args.runtime_provider,
                "task admission": "public-api",
                **({"error": error_message} if error_message else {}),
            },
        )
        return 0 if terminal_status == "succeeded" else 1
    except (OSError, RuntimeError, urllib.error.URLError, json.JSONDecodeError) as exc:
        error_message = str(exc)
        write_transcript(
            output,
            {
                "workspace": workspace,
                "run_id": run_id,
                "task_id": task_id,
                "attempt_id": attempt_id,
                "pipeline": args.pipeline,
                "status": "failed",
                "runtime mode": args.runtime,
                "runtime provider": args.runtime_provider,
                "task admission": "public-api",
                "error": error_message,
            },
        )
        return 1
    finally:
        terminate_process(server)


if __name__ == "__main__":
    raise SystemExit(main())
