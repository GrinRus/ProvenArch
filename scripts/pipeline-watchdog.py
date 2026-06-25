#!/usr/bin/env python3
"""Run one live E2E pipeline command under a hard wall-clock deadline."""

from __future__ import annotations

import argparse
import json
import os
import signal
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path


EXIT_TIMEOUT = 124


def utc_now() -> datetime:
    return datetime.now(timezone.utc).replace(microsecond=0)


def iso_from_epoch(value: float | None) -> str:
    if value is None or value <= 0:
        return ""
    return datetime.fromtimestamp(value, timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def parse_env_file(path: Path) -> dict[str, str]:
    if not path.exists():
        return {}
    values: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        if "=" not in raw:
            continue
        key, value = raw.split("=", 1)
        values[key.strip()] = value.strip()
    return values


def write_status(
    path: Path | None,
    *,
    state: str,
    process_exit: str,
    termination_signal: str,
    failure_reason: str,
    summary_written: str,
    last_pipeline_stage: str,
    last_runtime_provider: str,
    last_progress_at: str,
) -> None:
    if path is None:
        return
    previous = parse_env_file(path)
    provider = previous.get("provider", "")
    run_index = previous.get("run_index", "")
    path.parent.mkdir(parents=True, exist_ok=True)
    updated_at = utc_now().isoformat().replace("+00:00", "Z")
    path.write_text(
        "\n".join(
            [
                f"provider={provider}",
                f"run_index={run_index}",
                f"state={state}",
                f"process_exit={process_exit}",
                f"termination_signal={termination_signal}",
                f"failure_reason={failure_reason}",
                f"summary_written={summary_written}",
                f"updated_at={updated_at}",
                f"last_pipeline_stage={last_pipeline_stage}",
                f"last_runtime_provider={last_runtime_provider}",
                f"last_progress_at={last_progress_at or updated_at}",
            ]
        )
        + "\n",
        encoding="utf-8",
    )


def newest_mtime(root: Path) -> float | None:
    if not root.exists():
        return None
    latest: float | None = None
    roots = [
        root / "reports" / "taskruns",
        root / "reports",
        root / "model",
        root / "proposals",
        root / "charter",
        root / "skills",
    ]
    for scan_root in roots:
        if not scan_root.exists():
            continue
        for current, dirs, files in os.walk(scan_root):
            dirs[:] = [item for item in dirs if item not in {".git", "node_modules"}]
            current_path = Path(current)
            try:
                mtime = current_path.stat().st_mtime
            except OSError:
                mtime = None
            if mtime is not None and (latest is None or mtime > latest):
                latest = mtime
            for name in files:
                try:
                    mtime = (current_path / name).stat().st_mtime
                except OSError:
                    continue
                if latest is None or mtime > latest:
                    latest = mtime
    return latest


def kill_process_group(proc: subprocess.Popen[object], sig: signal.Signals, *, include_exited_group: bool = False) -> None:
    if proc.poll() is not None and not include_exited_group:
        return
    try:
        os.killpg(proc.pid, sig)
        return
    except ProcessLookupError:
        return
    except OSError:
        pass
    try:
        if sig == signal.SIGTERM:
            proc.terminate()
        elif sig == signal.SIGKILL:
            proc.kill()
        else:
            proc.send_signal(sig)
    except ProcessLookupError:
        return


def write_metadata(path: Path, payload: dict[str, object]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--timeout-sec", type=float, required=True)
    parser.add_argument("--grace-sec", type=float, required=True)
    parser.add_argument("--heartbeat-sec", type=float, default=60)
    parser.add_argument("--poll-sec", type=float, default=1)
    parser.add_argument("--clock-gap-threshold-sec", type=float, default=0)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--metadata", type=Path, required=True)
    parser.add_argument("--workspace", type=Path)
    parser.add_argument("--status-file", type=Path)
    parser.add_argument("--last-pipeline-stage", default="not_started")
    parser.add_argument("--last-runtime-provider", default="unset")
    parser.add_argument("--progress-label", default="pipeline")
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args(argv)

    command = list(args.command)
    if command and command[0] == "--":
        command = command[1:]
    if not command:
        parser.error("missing command after --")
    if args.timeout_sec <= 0:
        parser.error("--timeout-sec must be positive")
    if args.grace_sec < 0:
        parser.error("--grace-sec must be non-negative")
    poll_sec = max(args.poll_sec, 0.05)
    heartbeat_sec = max(args.heartbeat_sec, 0)
    clock_gap_threshold = args.clock_gap_threshold_sec
    if clock_gap_threshold <= 0:
        clock_gap_threshold = max(heartbeat_sec * 3, 30)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    started_epoch = time.time()
    deadline_epoch = started_epoch + args.timeout_sec
    last_tick_monotonic = time.monotonic()
    max_tick_gap = 0.0
    clock_gap_detected = False
    last_progress_epoch = started_epoch
    last_progress_source = "process_start"
    last_output_size = 0
    last_workspace_scan = 0.0
    next_heartbeat_epoch = started_epoch + heartbeat_sec if heartbeat_sec > 0 else 0.0
    timed_out = False
    termination_signal = "none"
    process_exit: int | None = None
    cancelled_by_signal: int | None = None

    def handle_signal(signum: int, _frame: object) -> None:
        nonlocal cancelled_by_signal
        cancelled_by_signal = signum

    previous_int = signal.signal(signal.SIGINT, handle_signal)
    previous_term = signal.signal(signal.SIGTERM, handle_signal)
    proc: subprocess.Popen[object] | None = None
    try:
        with args.output.open("ab", buffering=0) as output:
            proc = subprocess.Popen(command, stdout=output, stderr=subprocess.STDOUT, start_new_session=True)
            write_status(
                args.status_file,
                state="running",
                process_exit="",
                termination_signal="none",
                failure_reason="",
                summary_written="no",
                last_pipeline_stage=args.last_pipeline_stage,
                last_runtime_provider=args.last_runtime_provider,
                last_progress_at=iso_from_epoch(last_progress_epoch),
            )
            while True:
                rc = proc.poll()
                now_epoch = time.time()
                now_monotonic = time.monotonic()
                tick_gap = now_monotonic - last_tick_monotonic
                last_tick_monotonic = now_monotonic
                if tick_gap > max_tick_gap:
                    max_tick_gap = tick_gap
                if tick_gap > clock_gap_threshold:
                    clock_gap_detected = True

                try:
                    output_size = args.output.stat().st_size
                except OSError:
                    output_size = last_output_size
                if output_size > last_output_size:
                    last_output_size = output_size
                    last_progress_epoch = now_epoch
                    last_progress_source = "stdout_stderr"

                if args.workspace is not None and now_epoch - last_workspace_scan >= max(5.0, poll_sec):
                    last_workspace_scan = now_epoch
                    artifact_mtime = newest_mtime(args.workspace)
                    if artifact_mtime is not None and artifact_mtime > last_progress_epoch:
                        last_progress_epoch = artifact_mtime
                        last_progress_source = "workspace_artifact"

                if heartbeat_sec > 0 and now_epoch >= next_heartbeat_epoch:
                    elapsed_sec = int(now_epoch - started_epoch)
                    print(
                        "[pipeline-watchdog] progress "
                        f"{args.progress_label} elapsed_sec={elapsed_sec} "
                        f"timeout_sec={int(args.timeout_sec)} "
                        f"last_progress_at={iso_from_epoch(last_progress_epoch) or '-'} "
                        f"last_progress_source={last_progress_source}",
                        flush=True,
                    )
                    write_status(
                        args.status_file,
                        state="running",
                        process_exit="",
                        termination_signal="none",
                        failure_reason="",
                        summary_written="no",
                        last_pipeline_stage=args.last_pipeline_stage,
                        last_runtime_provider=args.last_runtime_provider,
                        last_progress_at=iso_from_epoch(last_progress_epoch),
                    )
                    while next_heartbeat_epoch <= now_epoch:
                        next_heartbeat_epoch += heartbeat_sec

                if cancelled_by_signal is not None:
                    kill_process_group(proc, signal.Signals(cancelled_by_signal))
                    try:
                        proc.wait(timeout=args.grace_sec)
                    except subprocess.TimeoutExpired:
                        kill_process_group(proc, signal.SIGKILL)
                        proc.wait(timeout=max(args.grace_sec, 1))
                    else:
                        if args.grace_sec > 0:
                            time.sleep(args.grace_sec)
                        kill_process_group(proc, signal.SIGKILL, include_exited_group=True)
                    process_exit = 128 + cancelled_by_signal
                    termination_signal = f"signal_{cancelled_by_signal}"
                    break

                if rc is not None:
                    process_exit = rc
                    break

                if now_epoch >= deadline_epoch:
                    timed_out = True
                    termination_signal = "timeout"
                    kill_process_group(proc, signal.SIGTERM)
                    try:
                        proc.wait(timeout=args.grace_sec)
                    except subprocess.TimeoutExpired:
                        kill_process_group(proc, signal.SIGKILL)
                        proc.wait(timeout=max(args.grace_sec, 1))
                    else:
                        if args.grace_sec > 0:
                            time.sleep(args.grace_sec)
                        kill_process_group(proc, signal.SIGKILL, include_exited_group=True)
                    process_exit = EXIT_TIMEOUT
                    write_status(
                        args.status_file,
                        state="process_failed",
                        process_exit=str(EXIT_TIMEOUT),
                        termination_signal="timeout",
                        failure_reason="runtime_timeout",
                        summary_written="no",
                        last_pipeline_stage=args.last_pipeline_stage,
                        last_runtime_provider=args.last_runtime_provider,
                        last_progress_at=iso_from_epoch(last_progress_epoch),
                    )
                    break

                time.sleep(poll_sec)
    finally:
        signal.signal(signal.SIGINT, previous_int)
        signal.signal(signal.SIGTERM, previous_term)

    finished_epoch = time.time()
    if process_exit is None:
        process_exit = 1
    metadata = {
        "started_at": iso_from_epoch(started_epoch),
        "finished_at": iso_from_epoch(finished_epoch),
        "started_epoch": started_epoch,
        "finished_epoch": finished_epoch,
        "pipeline_deadline_at": iso_from_epoch(deadline_epoch),
        "pipeline_deadline_epoch": deadline_epoch,
        "pipeline_timeout_sec": args.timeout_sec,
        "pipeline_kill_grace_sec": args.grace_sec,
        "pipeline_timeout_elapsed_sec": int(finished_epoch - started_epoch),
        "deadline_missed_by_sec": max(0, int(finished_epoch - deadline_epoch)),
        "last_progress_at": iso_from_epoch(last_progress_epoch),
        "last_progress_epoch": last_progress_epoch,
        "last_progress_source": last_progress_source,
        "last_watchdog_tick_at": iso_from_epoch(finished_epoch),
        "max_watchdog_tick_gap_sec": round(max_tick_gap, 3),
        "infra_host_sleep_or_clock_jump_detected": clock_gap_detected,
        "timed_out": timed_out,
        "termination_signal": termination_signal,
        "process_exit": process_exit,
        "command": command,
        "output_path": str(args.output),
    }
    write_metadata(args.metadata, metadata)
    if clock_gap_detected:
        print(
            "[pipeline-watchdog] diagnostic infra_host_sleep_or_clock_jump_detected "
            f"{args.progress_label} max_tick_gap_sec={max_tick_gap:.3f}",
            flush=True,
        )
    return EXIT_TIMEOUT if timed_out else int(process_exit)


if __name__ == "__main__":
    raise SystemExit(main())
