import { describe, expect, it } from "vitest";

import { buildCharterBaselineRecovery, defaultCharterSuggestion, promptUsageLabel, splitSummaryList } from "./charterUtils";

const artifact = {
  path: "charter/project.md",
  label: "Project charter",
  category: "charter",
  prompt_usage: "reference-only",
} as const;

describe("charterUtils", () => {
  it("pins a diagnostic to its direct artifact and preserves provider copy", () => {
    const issue = buildCharterBaselineRecovery(
      [{ level: "error", code: "charter_invalid", message: "bad", suggestion: "fix it", path: artifact.path } as never],
      [artifact as never],
      "",
    );

    expect(issue).toMatchObject({ artifactPath: artifact.path, artifactLabel: artifact.label, category: "charter", promptUsage: "reference only", suggestion: "fix it" });
  });

  it("falls back to the selected editor artifact when diagnostics lack a path", () => {
    const issue = buildCharterBaselineRecovery(
      [{ level: "warning", code: "bundle_warning", message: "context" } as never],
      [artifact as never],
      artifact.path,
    );

    expect(issue?.artifactPath).toBe(artifact.path);
    expect(issue?.severity).toBe("warning");
  });

  it("keeps prompt and suggestion labels deterministic", () => {
    expect(promptUsageLabel()).toBe("bundle diagnostic");
    expect(defaultCharterSuggestion("skills/prompt-packs/default.md")).toContain("prompt pack");
    expect(defaultCharterSuggestion("notes.md")).toContain("baseline artifact");
    expect(splitSummaryList("scope,\n rules,,")).toEqual(["scope", "rules"]);
  });
});
