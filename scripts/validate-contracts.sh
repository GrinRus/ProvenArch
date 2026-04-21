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
    invalid_log="$tmpdir/$(basename "$file").invalid.log"
    if ajv validate --spec=draft2020 -c ajv-formats -s schemas/workspace.schema.json -d "$json_file" >"$invalid_log" 2>&1; then
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
)

for entry in "${docs_first_contracts[@]}"; do
  schema="${entry%%:*}"
  sample="${entry##*:}"
  ajv validate --spec=draft2020 -c ajv-formats -s "$schema" -d "$sample"
done
