#!/usr/bin/env bash
# Internal source-only helper for live E2E black-box evaluator step evidence.
# Public live E2E entrypoints remain scripts/full-run-batch-matrix.sh and scripts/full-run-batch.sh.

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  echo "live-e2e-evaluator.sh is an internal source-only helper; do not execute it directly." >&2
  exit 64
fi

live_e2e_evaluator_init_batch_report() {
  local jsonl_path="$1"
  local md_path="$2"
  local batch_id="$3"
  local profile_id="$4"
  local sweep_id="$5"

  mkdir -p "$(dirname "$jsonl_path")" "$(dirname "$md_path")"
  : >"$jsonl_path"
  python3 - "$md_path" "$batch_id" "$profile_id" "$sweep_id" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1])
batch_id = sys.argv[2]
profile_id = sys.argv[3]
sweep_id = sys.argv[4]
path.write_text(
    "\n".join(
        [
            f"# Black-Box E2E Steps: {batch_id}",
            "",
            f"- profile_id: {profile_id}",
            f"- sweep_id: {sweep_id}",
            "",
            "| step_id | status | classification | goal | action | observed_evidence | next_decision |",
            "|---|---|---|---|---|---|---|",
        ]
    )
    + "\n",
    encoding="utf-8",
)
PY
}

live_e2e_evaluator_write_batch_step() {
  local jsonl_path="$1"
  local md_path="$2"
  local batch_id="$3"
  local profile_id="$4"
  local sweep_id="$5"
  local step_id="$6"
  local goal="$7"
  local action="$8"
  local status="$9"
  local primary_classification="${10}"
  local next_decision="${11}"
  shift 11

  python3 - "$jsonl_path" "$md_path" "$batch_id" "$profile_id" "$sweep_id" "$step_id" "$goal" "$action" "$status" "$primary_classification" "$next_decision" "$@" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

jsonl_path = Path(sys.argv[1])
md_path = Path(sys.argv[2])
batch_id = sys.argv[3]
profile_id = sys.argv[4]
sweep_id = sys.argv[5]
step_id = sys.argv[6]
goal = sys.argv[7]
action = sys.argv[8]
status = sys.argv[9]
primary_classification = sys.argv[10]
next_decision = sys.argv[11]
evidence = [item for item in sys.argv[12:] if item and item != "-"]
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
payload = {
    "generated_at_utc": now,
    "batch_id": batch_id,
    "profile_id": profile_id,
    "sweep_id": sweep_id,
    "step_id": step_id,
    "goal": goal,
    "action": action,
    "observed_evidence": evidence,
    "status": status,
    "primary_classification": primary_classification,
    "evidence_paths": evidence,
    "next_decision": next_decision,
}
with jsonl_path.open("a", encoding="utf-8") as f:
    f.write(json.dumps(payload, ensure_ascii=True, sort_keys=True))
    f.write("\n")

def cell(value: object) -> str:
    text = str(value).replace("\n", " ").replace("|", "\\|").strip()
    return text if text else "-"

with md_path.open("a", encoding="utf-8") as f:
    f.write(
        "| "
        + " | ".join(
            [
                cell(step_id),
                cell(status),
                cell(primary_classification),
                cell(goal),
                cell(action),
                cell("; ".join(evidence)),
                cell(next_decision),
            ]
        )
        + " |\n"
    )
PY
}

live_e2e_evaluator_init_matrix_report() {
  local jsonl_path="$1"
  local md_path="$2"
  local matrix_id="$3"
  local matrix_file="$4"

  mkdir -p "$(dirname "$jsonl_path")" "$(dirname "$md_path")"
  : >"$jsonl_path"
  python3 - "$md_path" "$matrix_id" "$matrix_file" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1])
matrix_id = sys.argv[2]
matrix_file = sys.argv[3]
path.write_text(
    "\n".join(
        [
            f"# Black-Box E2E Matrix Steps: {matrix_id}",
            "",
            f"- matrix_file: {matrix_file}",
            "",
            "| step_id | status | classification | goal | action | observed_evidence | next_decision |",
            "|---|---|---|---|---|---|---|",
        ]
    )
    + "\n",
    encoding="utf-8",
)
PY
}

live_e2e_evaluator_write_matrix_step() {
  local jsonl_path="$1"
  local md_path="$2"
  local matrix_id="$3"
  local matrix_file="$4"
  local step_id="$5"
  local goal="$6"
  local action="$7"
  local status="$8"
  local primary_classification="$9"
  local next_decision="${10}"
  shift 10

  python3 - "$jsonl_path" "$md_path" "$matrix_id" "$matrix_file" "$step_id" "$goal" "$action" "$status" "$primary_classification" "$next_decision" "$@" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

jsonl_path = Path(sys.argv[1])
md_path = Path(sys.argv[2])
matrix_id = sys.argv[3]
matrix_file = sys.argv[4]
step_id = sys.argv[5]
goal = sys.argv[6]
action = sys.argv[7]
status = sys.argv[8]
primary_classification = sys.argv[9]
next_decision = sys.argv[10]
evidence = [item for item in sys.argv[11:] if item and item != "-"]
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
payload = {
    "generated_at_utc": now,
    "matrix_id": matrix_id,
    "matrix_file": matrix_file,
    "step_id": step_id,
    "goal": goal,
    "action": action,
    "observed_evidence": evidence,
    "status": status,
    "primary_classification": primary_classification,
    "evidence_paths": evidence,
    "next_decision": next_decision,
}
with jsonl_path.open("a", encoding="utf-8") as f:
    f.write(json.dumps(payload, ensure_ascii=True, sort_keys=True))
    f.write("\n")

def cell(value: object) -> str:
    text = str(value).replace("\n", " ").replace("|", "\\|").strip()
    return text if text else "-"

with md_path.open("a", encoding="utf-8") as f:
    f.write(
        "| "
        + " | ".join(
            [
                cell(step_id),
                cell(status),
                cell(primary_classification),
                cell(goal),
                cell(action),
                cell("; ".join(evidence)),
                cell(next_decision),
            ]
        )
        + " |\n"
    )
PY
}
