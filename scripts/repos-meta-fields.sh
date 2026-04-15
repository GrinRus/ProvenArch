#!/usr/bin/env bash
# Shared parser for resolve-repos-meta.py output JSON.

acp_read_repos_meta_fields() {
  local meta_json="$1"
  python3 - "$meta_json" <<'PY'
import json
import sys

payload = json.load(open(sys.argv[1], encoding="utf-8"))
repos_file = str(payload.get("repos_file", payload.get("target_repos_file", ""))).strip()
target_profile = str(payload.get("target_profile", "generic")).strip() or "generic"
source_kind = str(payload.get("profile_source_kind", "mixed")).strip() or "mixed"
expected_count = int(payload.get("expected_repo_count", len(payload.get("declared_repos") or [])))

print(f"repos_file={repos_file}")
print(f"target_profile={target_profile}")
print(f"profile_source_kind={source_kind}")
print(f"expected_repo_count={expected_count}")
PY
}
