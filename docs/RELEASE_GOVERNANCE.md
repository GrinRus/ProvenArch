# Release governance evidence

`release-governance.json` is a versioned, read-only snapshot of the GitHub controls that
protect release truth. It is tied to the `source_sha` from which the evidence was captured;
it is not itself a release verdict and it never grants a waiver.

The deterministic check validates the snapshot, the release workflow, the six expected required
contexts, and every tracked owner-waiver payload:

```bash
make verify-release-governance
```

To compare the current GitHub settings with the snapshot on a trusted machine, provide a token
with read access to repository settings:

```bash
GITHUB_TOKEN="$TOKEN" ./scripts/run-python.sh scripts/verify-release-governance.py --live
```

`--live` is read-only. A mismatch is a governance failure that must be fixed by an owner/admin or
by a follow-up versioned evidence PR. It must not be repaired by weakening the expected checks in
the snapshot. The release workflow remains fail-closed on exact-tag evidence and owner waivers;
this snapshot does not change that policy.

The current evidence records:

- strict `main` protection with `backend`, `contracts`, `golden`, `smoke-api`, `smoke-cli`, and
  `ui` required checks;
- active `protect release tags` ruleset for `refs/tags/v*`, blocking deletion and non-fast-forward
  updates with no bypass actors;
- `github-release` environment review by `GrinRus`;
- exact tracked owner-waiver path, schema, release state, and allowed waived requirements.

The separately authorized REM-03B governance operation was a no-op: the live snapshot already
matched this manifest, so no GitHub setting was mutated. Before/after and rollback evidence is
recorded in [the REM-03B audit artifact](audits/REM-03B_GITHUB_GOVERNANCE_2026-09-08.json).
