export const epic20TaskFixture = {
  snapshots: [
    { run_id: "run-old", canonical_path: "reports/as-is/overview.md", content: "Old snapshot evidence", runtime_mode: "fake" },
    { run_id: "run-new", canonical_path: "reports/as-is/overview.md", content: "New live evidence", runtime_mode: "headless" },
  ],
  runtime: { desired: "headless", effective: "fake", provider: "fake" },
  coordination: { active_run_id: "run-active", pending: { run_id: "run-pending", pipeline: "refresh" } },
  knowledge: { status: "partial", valid_entity_id: "service-a", issue: "missing_reference" },
  git: {
    fingerprint: "fixture-fingerprint",
    files: [
      { path: "reports/as-is/overview.md", status: "modified" },
      { path: "notes/outside-selected-artifact.md", status: "untracked" },
      { path: "model/entities/new.yaml", original_path: "model/entities/old.yaml", status: "renamed" },
    ],
  },
} as const;
