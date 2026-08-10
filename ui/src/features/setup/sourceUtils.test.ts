import { describe, expect, it } from "vitest";

import { buildSourceValidationRecovery, defaultSourceSuggestion, sourceTypeLabel, sourceValueLabel } from "./sourceUtils";
import type { GuidedRepo } from "../../lib/appContracts";

const pathRepo: GuidedRepo = {
  id: "repo-1",
  name: "api",
  mode: "path",
  path: "/tmp/api",
  git_url: "",
  ref: "main",
  analysis_include: "",
  analysis_exclude: "",
};

describe("sourceUtils", () => {
  it("keeps server diagnostics authoritative and maps repository context", () => {
    const issue = buildSourceValidationRecovery(
      [pathRepo],
      { ok: false, workspace: "/tmp/workspace.yaml" } as never,
      [["api", [{ code: "source_unreadable", level: "error", message: "cannot read", suggestion: "check path", repo: "api" } as never]]],
    );

    expect(issue).toMatchObject({ repoKey: "api", diagnosticLabel: "source_unreadable", sourceType: "Local folder", sourceValue: "/tmp/api", refValue: "main" });
    expect(issue?.suggestion).toBe("check path");
  });

  it("finds the first incomplete draft repo without weakening valid drafts", () => {
    const issue = buildSourceValidationRecovery(
      [pathRepo, { ...pathRepo, id: "repo-2", name: "", path: "" }],
      null,
      [],
    );

    expect(issue).toMatchObject({ repoKey: "Repo 2", diagnosticLabel: "Repo name is missing", level: "draft" });
  });

  it("uses stable labels and safe fallback suggestions", () => {
    expect(sourceTypeLabel()).toBe("Workspace manifest");
    expect(sourceValueLabel(undefined, "/tmp/workspace.yaml")).toBe("/tmp/workspace.yaml");
    expect(defaultSourceSuggestion()).toContain("workspace manifest");
  });
});
