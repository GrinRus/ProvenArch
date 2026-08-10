import { describe, expect, it } from "vitest";

import { buildPublishFolderSummaries, buildPublishGateItems, comparePublishArtifactPriority, gitDiffScopeHint, slugify } from "./publishUtils";

const artifact = (path: string, label = path) => ({ path, kind: "markdown", label });

describe("publish selectors", () => {
  it("sorts authoritative preview artifacts before package details", () => {
    const sorted = [artifact("proposals/orders/proposal.md"), artifact("reports/as-is/overview.md")].sort(comparePublishArtifactPriority);
    expect(sorted.map((item) => item.path)).toEqual(["reports/as-is/overview.md", "proposals/orders/proposal.md"]);
  });

  it("groups preview artifacts and keeps full-scope gate language explicit", () => {
    const folders = buildPublishFolderSummaries([artifact("reports/as-is/overview.md", "Overview"), artifact("proposals/orders/ADR.md", "ADR")]);
    expect(folders.map((folder) => folder.folder)).toEqual(["proposals", "reports/as-is"]);
    const gate = buildPublishGateItems({
      previewArtifactCount: 2,
      previewFolderCount: 2,
      gitDiff: null,
      gitDiffStatus: "Loading workspace inventory",
      gitMessage: "",
      proposalBranch: "",
      openQuestions: "- confirm owner",
    });
    expect(gate[0].tone).toBe("error");
    expect(gate[0].detail).toContain("Loading workspace inventory");
    expect(gate[1].detail).toContain("does not limit the Git commit scope");
  });

  it("provides stable scope hint and workspace-safe slugs", () => {
    expect(gitDiffScopeHint(null)).toContain("full workspace Git inventory");
    expect(slugify(" Orders & API ")).toBe("orders-api");
    expect(slugify("***")).toBe("my-service");
  });
});
