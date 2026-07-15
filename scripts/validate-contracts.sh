#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || (cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd))"
cd "$repo_root"

contract_tools_bin="${ACP_CONTRACT_TOOLS_BIN:-$repo_root/tools/contracts/node_modules/.bin}"

require_contract_tool() {
  local tool="$1"
  local tool_path="$contract_tools_bin/$tool"
  if [[ ! -x "$tool_path" ]]; then
    echo "Missing contract validation tool: $tool" >&2
    echo "Run: ./scripts/run-npm.sh ci --prefix tools/contracts --ignore-scripts --audit=false --fund=false" >&2
    exit 1
  fi
}

require_contract_tool ajv
require_contract_tool js-yaml

ajv_bin="$contract_tools_bin/ajv"
js_yaml_bin="$contract_tools_bin/js-yaml"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

workspace_cases=(
  "examples/workspace.example.yaml:valid"
  "fixtures/workspace/valid-path.yaml:valid"
  "fixtures/workspace/valid-git-url.yaml:valid"
  "fixtures/workspace/valid-with-runtime-timeouts.yaml:valid"
  "fixtures/workspace/valid-with-runtime-permissions.yaml:valid"
  "fixtures/workspace/invalid-both.yaml:invalid"
  "fixtures/workspace/invalid-neither.yaml:invalid"
)

for entry in "${workspace_cases[@]}"; do
  file="${entry%%:*}"
  expectation="${entry##*:}"
  json_file="$tmpdir/$(basename "$file").json"

  "$js_yaml_bin" "$file" > "$json_file"

  if [[ "$expectation" == "valid" ]]; then
    "$ajv_bin" validate --spec=draft2020 -c ajv-formats -s schemas/workspace.schema.json -d "$json_file"
  else
    invalid_log="$tmpdir/$(basename "$file").invalid.log"
    if "$ajv_bin" validate --spec=draft2020 -c ajv-formats -s schemas/workspace.schema.json -d "$json_file" >"$invalid_log" 2>&1; then
      echo "Expected invalid fixture to fail: $file"
      cat "$invalid_log"
      exit 1
    fi
    echo "$file invalid as expected"
  fi
done

docs_first_contracts=(
  "schemas/shard-pack-manifest.schema.json:examples/shard-pack-manifest.example.json"
  "schemas/final-run-index.schema.json:examples/final-run-index.example.json"
  "schemas/citation-index.schema.json:examples/citation-index.example.json"
  "schemas/validator-verdict.schema.json:examples/validator-verdict.example.json"
  "schemas/qa-answer.schema.json:examples/qa-answer.example.json"
  "schemas/source-revisions.schema.json:examples/source-revisions.example.json"
  "schemas/refresh-impact-plan.schema.json:examples/refresh-impact-plan.example.json"
  "schemas/source-revisions.schema.json:fixtures/refresh-planning/unchanged/source-revisions.json"
  "schemas/refresh-impact-plan.schema.json:fixtures/refresh-planning/unchanged/refresh-impact-plan.json"
  "schemas/refresh-impact-plan.schema.json:fixtures/refresh-planning/full-fallback/refresh-impact-plan.json"
)

for entry in "${docs_first_contracts[@]}"; do
  schema="${entry%%:*}"
  sample="${entry##*:}"
  "$ajv_bin" validate --spec=draft2020 -c ajv-formats -s "$schema" -d "$sample"
done
