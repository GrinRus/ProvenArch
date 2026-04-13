#!/usr/bin/env bash
set -Eeuo pipefail

PROVENARCH_ROOT="${PROVENARCH_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
BATCH_SCRIPT="${BATCH_SCRIPT:-$PROVENARCH_ROOT/scripts/full-run-batch-5x2.sh}"
E2E_MATRIX_FILE="${E2E_MATRIX_FILE:-}"
MATRIX_ID="${MATRIX_ID:-matrix-$(date -u +'%Y%m%dT%H%M%SZ')}"
RUN_COUNT="${RUN_COUNT:-5}"
E2E_TMP_ROOT="${E2E_TMP_ROOT:-/tmp/provenarch-test_arch_project}"
REPORTS_ROOT="${REPORTS_ROOT:-$E2E_TMP_ROOT/reports}"
MATRIX_ROOT="${MATRIX_ROOT:-$E2E_TMP_ROOT/matrix/$MATRIX_ID}"
ACP_CLAUDE_CMD_BIN="${ACP_CLAUDE_CMD_BIN:-claude}"
ACP_QWEN_CMD_BIN="${ACP_QWEN_CMD_BIN:-qwen}"

log() {
  printf '[batch-matrix] %s\n' "$*" >&2
}

die() {
  echo "[batch-matrix][error] $*" >&2
  exit 1
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    die "required command is unavailable: $cmd"
  fi
}

slugify() {
  local value
  value="$(echo "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
  if [[ -z "$value" ]]; then
    value="profile"
  fi
  printf '%s' "$value"
}

if [[ -z "$E2E_MATRIX_FILE" ]]; then
  die "E2E_MATRIX_FILE is required (YAML with profiles[])"
fi
if [[ ! -f "$E2E_MATRIX_FILE" ]]; then
  die "E2E_MATRIX_FILE does not exist: $E2E_MATRIX_FILE"
fi
if [[ "$RUN_COUNT" != "5" ]]; then
  die "RUN_COUNT must be 5 for matrix mode (got '$RUN_COUNT')"
fi
if [[ ! -x "$BATCH_SCRIPT" ]]; then
  die "batch script is unavailable: $BATCH_SCRIPT"
fi

require_cmd bash
require_cmd python3
require_cmd "$ACP_CLAUDE_CMD_BIN"
require_cmd "$ACP_QWEN_CMD_BIN"

mkdir -p "$MATRIX_ROOT" "$REPORTS_ROOT"

PROFILES_TSV="$MATRIX_ROOT/profiles.tsv"
RECORDS_JSONL="$MATRIX_ROOT/profile-runs.jsonl"
: > "$RECORDS_JSONL"

python3 - "$E2E_MATRIX_FILE" "$PROFILES_TSV" <<'PY'
import sys
from pathlib import Path

try:
    import yaml  # type: ignore
except Exception as exc:
    raise SystemExit(f"PyYAML is required for parsing matrix file: {exc}")

matrix_path = Path(sys.argv[1]).resolve()
out_path = Path(sys.argv[2]).resolve()
payload = yaml.safe_load(matrix_path.read_text(encoding="utf-8"))

if isinstance(payload, dict):
    profiles = payload.get("profiles")
elif isinstance(payload, list):
    profiles = payload
else:
    profiles = None

if not isinstance(profiles, list) or not profiles:
    raise SystemExit(f"matrix file {matrix_path} must contain non-empty profiles[]")

required_profiles = {
    "single-path": {"source_kind": "path", "min_repos": 1, "max_repos": 1},
    "single-git_url": {"source_kind": "git_url", "min_repos": 1, "max_repos": 1},
    "multi-path": {"source_kind": "path", "min_repos": 2, "max_repos": None},
    "multi-git_url": {"source_kind": "git_url", "min_repos": 2, "max_repos": None},
}

rows: list[str] = []
seen_ids: set[str] = set()
for idx, item in enumerate(profiles, start=1):
    if not isinstance(item, dict):
        raise SystemExit(f"profiles[{idx}] must be an object")
    profile_id = str(item.get("id", "")).strip()
    repos_file_raw = str(item.get("repos_file", "")).strip()
    source_kind = str(item.get("source_kind", "")).strip()
    expected_raw = str(item.get("expected_repo_count", "")).strip()

    if not profile_id:
        raise SystemExit(f"profiles[{idx}] is missing id")
    if profile_id in seen_ids:
        raise SystemExit(f"duplicate profile id: {profile_id}")
    seen_ids.add(profile_id)

    if not repos_file_raw:
        raise SystemExit(f"profiles[{idx}] is missing repos_file")
    repos_file = Path(repos_file_raw)
    if not repos_file.is_absolute():
        repos_file = (matrix_path.parent / repos_file).resolve()
    else:
        repos_file = repos_file.resolve()
    if not repos_file.exists():
        raise SystemExit(f"profiles[{idx}] repos_file does not exist: {repos_file}")
    if source_kind not in {"path", "git_url"}:
        raise SystemExit(f"profiles[{idx}] source_kind must be path|git_url, got: {source_kind}")

    try:
        expected_count = int(expected_raw)
    except Exception:
        raise SystemExit(f"profiles[{idx}] expected_repo_count must be integer, got: {expected_raw}")
    if expected_count <= 0:
        raise SystemExit(f"profiles[{idx}] expected_repo_count must be > 0, got: {expected_count}")

    contract = required_profiles.get(profile_id)
    if contract is None:
        raise SystemExit(
            f"profiles[{idx}] id must be one of: {', '.join(sorted(required_profiles.keys()))}; got: {profile_id}"
        )
    expected_kind = str(contract["source_kind"])
    if source_kind != expected_kind:
        raise SystemExit(
            f"profiles[{idx}] id={profile_id} must use source_kind={expected_kind}, got: {source_kind}"
        )
    min_repos = int(contract["min_repos"])
    max_repos = contract["max_repos"]
    if expected_count < min_repos:
        raise SystemExit(
            f"profiles[{idx}] id={profile_id} expected_repo_count must be >= {min_repos}, got: {expected_count}"
        )
    if max_repos is not None and expected_count > int(max_repos):
        raise SystemExit(
            f"profiles[{idx}] id={profile_id} expected_repo_count must be <= {max_repos}, got: {expected_count}"
        )

    rows.append("\t".join([profile_id, str(repos_file), str(expected_count), source_kind]))

missing = sorted(set(required_profiles.keys()) - seen_ids)
extra = sorted(seen_ids - set(required_profiles.keys()))
if missing:
    raise SystemExit(f"matrix file {matrix_path} missing required profile ids: {', '.join(missing)}")
if extra:
    raise SystemExit(f"matrix file {matrix_path} has unsupported profile ids: {', '.join(extra)}")
if len(profiles) != len(required_profiles):
    raise SystemExit(
        f"matrix file {matrix_path} must contain exactly {len(required_profiles)} profiles"
    )

out_path.parent.mkdir(parents=True, exist_ok=True)
out_path.write_text("\n".join(rows) + "\n", encoding="utf-8")
PY

failures=0
while IFS=$'\t' read -r profile_id repos_file expected_repo_count source_kind; do
  [[ -z "$profile_id" ]] && continue
  profile_slug="$(slugify "$profile_id")"
  batch_id="${MATRIX_ID}-${profile_slug}"
  profile_root="$MATRIX_ROOT/profiles/$profile_slug"
  driver_log="$profile_root/driver.log"
  mkdir -p "$profile_root"

  log "running profile=$profile_id source_kind=$source_kind expected_repo_count=$expected_repo_count batch_id=$batch_id"
  status="passed"
  if ! (
    cd "$PROVENARCH_ROOT"
    BATCH_ID="$batch_id" \
    RUN_COUNT="$RUN_COUNT" \
    TARGET_REPOS_FILE="$repos_file" \
    PROFILE_ID="$profile_id" \
    PROFILE_SOURCE_KIND="$source_kind" \
    EXPECTED_REPO_COUNT="$expected_repo_count" \
    E2E_TMP_ROOT="$E2E_TMP_ROOT" \
    REPORTS_ROOT="$REPORTS_ROOT" \
    ACP_CLAUDE_CMD_BIN="$ACP_CLAUDE_CMD_BIN" \
    ACP_QWEN_CMD_BIN="$ACP_QWEN_CMD_BIN" \
    "$BATCH_SCRIPT"
  ) >"$driver_log" 2>&1; then
    status="failed"
    failures=$((failures + 1))
    log "profile failed: $profile_id (see $driver_log)"
  fi

  run_matrix_tsv="$REPORTS_ROOT/run_matrix_${batch_id}.tsv"
  run_matrix_md="$REPORTS_ROOT/run_matrix_${batch_id}.md"
  frontend_matrix_md="$REPORTS_ROOT/frontend_e2e_matrix_${batch_id}.md"
  quality_report_md="$REPORTS_ROOT/quality_report_${batch_id}.md"

  python3 - "$RECORDS_JSONL" "$profile_id" "$profile_slug" "$batch_id" "$source_kind" "$expected_repo_count" "$repos_file" "$status" "$run_matrix_tsv" "$run_matrix_md" "$frontend_matrix_md" "$quality_report_md" "$driver_log" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1]).resolve()
payload = {
    "profile_id": sys.argv[2],
    "profile_slug": sys.argv[3],
    "batch_id": sys.argv[4],
    "source_kind": sys.argv[5],
    "expected_repo_count": int(sys.argv[6]),
    "repos_file": sys.argv[7],
    "status": sys.argv[8],
    "run_matrix_tsv": sys.argv[9],
    "run_matrix_md": sys.argv[10],
    "frontend_matrix_md": sys.argv[11],
    "quality_report_md": sys.argv[12],
    "driver_log": sys.argv[13],
}
with path.open("a", encoding="utf-8") as f:
    f.write(json.dumps(payload, ensure_ascii=True))
    f.write("\n")
PY
done < "$PROFILES_TSV"

MATRIX_REPORT_MD="$REPORTS_ROOT/profile_matrix_${MATRIX_ID}.md"
MATRIX_REPORT_TSV="$REPORTS_ROOT/profile_matrix_${MATRIX_ID}.tsv"
python3 - "$RECORDS_JSONL" "$MATRIX_REPORT_MD" "$MATRIX_REPORT_TSV" <<'PY'
import json
import re
import sys
from pathlib import Path

records_path = Path(sys.argv[1]).resolve()
out_md = Path(sys.argv[2]).resolve()
out_tsv = Path(sys.argv[3]).resolve()

records = []
for line in records_path.read_text(encoding="utf-8").splitlines():
    line = line.strip()
    if not line:
        continue
    records.append(json.loads(line))

header = [
    "profile_id",
    "batch_id",
    "source_kind",
    "expected_repo_count",
    "status",
    "backend_hard_pass",
    "backend_total_runs",
    "runtime_parse_failures",
    "runner_unavailable_failures",
    "infra_signal_terminated_failures",
    "infra_incomplete_cycle_failures",
    "quality_gates_failed_failures",
    "summary_missing_failures",
    "frontend_qwen_status",
    "frontend_claude_status",
    "run_matrix_tsv",
    "quality_report_md",
]
tsv_lines = ["\t".join(header)]

md_lines = [
    "# Profile Matrix",
    "",
    "| profile_id | batch_id | source_kind | expected_repo_count | status | backend_hard_pass | backend_total_runs | runtime_parse_failures | runner_unavailable_failures | infra_signal_terminated_failures | infra_incomplete_cycle_failures | quality_gates_failed_failures | summary_missing_failures | frontend_qwen | frontend_claude | run_matrix | quality_report |",
    "|---|---|---|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---|---|",
]

def parse_backend_stats(tsv_path: Path) -> dict[str, int]:
    stats = {
        "hard": 0,
        "total": 0,
        "runtime_parse": 0,
        "runner_unavailable": 0,
        "infra_signal_terminated": 0,
        "infra_incomplete_cycle": 0,
        "quality_gates_failed": 0,
        "summary_missing": 0,
    }
    if not tsv_path.exists():
        return stats
    lines = [line for line in tsv_path.read_text(encoding="utf-8").splitlines() if line.strip()]
    if len(lines) <= 1:
        return stats
    header_parts = lines[0].split("\t")
    index = {name: idx for idx, name in enumerate(header_parts)}
    hard_idx = index.get("hard_pass", 2)
    runtime_parse_idx = index.get("runtime_parse")
    runner_unavailable_idx = index.get("runner_unavailable")
    infra_signal_idx = index.get("infra_signal_terminated")
    infra_incomplete_idx = index.get("infra_incomplete_cycle")
    quality_gates_failed_idx = index.get("quality_gates_failed")
    summary_missing_idx = index.get("summary_missing")
    for line in lines[1:]:
        parts = line.split("\t")
        if len(parts) <= hard_idx:
            continue
        stats["total"] += 1
        stats["hard"] += 1 if parts[hard_idx] == "1" else 0
        if runtime_parse_idx is not None and len(parts) > runtime_parse_idx and parts[runtime_parse_idx] == "1":
            stats["runtime_parse"] += 1
        if runner_unavailable_idx is not None and len(parts) > runner_unavailable_idx and parts[runner_unavailable_idx] == "1":
            stats["runner_unavailable"] += 1
        if infra_signal_idx is not None and len(parts) > infra_signal_idx and parts[infra_signal_idx] == "1":
            stats["infra_signal_terminated"] += 1
        if infra_incomplete_idx is not None and len(parts) > infra_incomplete_idx and parts[infra_incomplete_idx] == "1":
            stats["infra_incomplete_cycle"] += 1
        if quality_gates_failed_idx is not None and len(parts) > quality_gates_failed_idx and parts[quality_gates_failed_idx] == "1":
            stats["quality_gates_failed"] += 1
        if summary_missing_idx is not None and len(parts) > summary_missing_idx and parts[summary_missing_idx] == "1":
            stats["summary_missing"] += 1
    return stats

def parse_frontend_status(path: Path, provider: str) -> str:
    if not path.exists():
        return "missing"
    match = re.search(rf"^\|\s*{re.escape(provider)}\s*\|\s*([^|]+)\|", path.read_text(encoding="utf-8"), flags=re.MULTILINE)
    return match.group(1).strip() if match else "missing"

for rec in records:
    run_matrix_tsv = Path(rec["run_matrix_tsv"])
    frontend_matrix_md = Path(rec["frontend_matrix_md"])
    backend_stats = parse_backend_stats(run_matrix_tsv)
    frontend_qwen = parse_frontend_status(frontend_matrix_md, "qwen-code")
    frontend_claude = parse_frontend_status(frontend_matrix_md, "claude-code")

    tsv_lines.append(
        "\t".join(
            [
                rec["profile_id"],
                rec["batch_id"],
                rec["source_kind"],
                str(rec["expected_repo_count"]),
                rec["status"],
                str(backend_stats["hard"]),
                str(backend_stats["total"]),
                str(backend_stats["runtime_parse"]),
                str(backend_stats["runner_unavailable"]),
                str(backend_stats["infra_signal_terminated"]),
                str(backend_stats["infra_incomplete_cycle"]),
                str(backend_stats["quality_gates_failed"]),
                str(backend_stats["summary_missing"]),
                frontend_qwen,
                frontend_claude,
                rec["run_matrix_tsv"],
                rec["quality_report_md"],
            ]
        )
    )

    md_lines.append(
        "| "
        f"{rec['profile_id']} | {rec['batch_id']} | {rec['source_kind']} | {rec['expected_repo_count']} | {rec['status']} | "
        f"{backend_stats['hard']} | {backend_stats['total']} | {backend_stats['runtime_parse']} | {backend_stats['runner_unavailable']} | "
        f"{backend_stats['infra_signal_terminated']} | {backend_stats['infra_incomplete_cycle']} | {backend_stats['quality_gates_failed']} | {backend_stats['summary_missing']} | "
        f"{frontend_qwen} | {frontend_claude} | "
        f"{rec['run_matrix_md']} | {rec['quality_report_md']} |"
    )

out_md.parent.mkdir(parents=True, exist_ok=True)
out_md.write_text("\n".join(md_lines) + "\n", encoding="utf-8")
out_tsv.write_text("\n".join(tsv_lines) + "\n", encoding="utf-8")
PY

log "profile matrix markdown: $MATRIX_REPORT_MD"
log "profile matrix tsv: $MATRIX_REPORT_TSV"

if [[ "$failures" -ne 0 ]]; then
  die "matrix finished with failures: $failures profile(s)"
fi

log "matrix completed successfully"
exit 0
