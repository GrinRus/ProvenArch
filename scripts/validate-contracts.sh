#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

workspace_cases=(
  "examples/workspace.example.yaml:valid"
  "fixtures/workspace/valid-path.yaml:valid"
  "fixtures/workspace/valid-git-url.yaml:valid"
  "fixtures/workspace/valid-with-runtime-timeouts.yaml:valid"
  "fixtures/workspace/invalid-both.yaml:invalid"
  "fixtures/workspace/invalid-neither.yaml:invalid"
)

for entry in "${workspace_cases[@]}"; do
  file="${entry%%:*}"
  expectation="${entry##*:}"
  json_file="$tmpdir/$(basename "$file").json"

  js-yaml "$file" > "$json_file"

  if [[ "$expectation" == "valid" ]]; then
    ajv validate --spec=draft2020 -c ajv-formats -s schemas/workspace.schema.json -d "$json_file"
  else
    if ajv validate --spec=draft2020 -c ajv-formats -s schemas/workspace.schema.json -d "$json_file" >/tmp/acp-contracts-invalid.log 2>&1; then
      echo "Expected invalid fixture to fail: $file"
      cat /tmp/acp-contracts-invalid.log
      exit 1
    fi
    echo "$file invalid as expected"
  fi
done

ajv validate --spec=draft2020 -c ajv-formats -s schemas/taskresult.schema.json -d examples/taskresult.example.json
ajv validate --spec=draft2020 -c ajv-formats -s schemas/taskresult.schema.json -d fixtures/taskresult/normalized-top-level.json
ajv validate --spec=draft2020 -c ajv-formats -s schemas/taskresult.schema.json -d fixtures/taskresult/mixed-legacy.json
