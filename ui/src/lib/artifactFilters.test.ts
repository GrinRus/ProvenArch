import { describe, expect, it } from "vitest";

import {
  filterReviewArtifactGroups,
  groupArtifactsByFolder,
  publishArtifactMatchesFilter,
  reviewArtifactGroupCategory,
  reviewArtifactGroupCategoryLabel,
  type ArtifactGroup,
} from "./artifactFilters";
import type { Artifact } from "./appContracts";

const artifact = (path: string, kind = "report"): Artifact => ({ path, kind, label: path });

describe("artifact filter helpers", () => {
  it("groups review artifacts by stable folder priority and path order", () => {
    const groups = groupArtifactsByFolder([
      artifact("reports/findings/findings.md", "finding"),
      artifact("reports/as-is/overview.md"),
      artifact("reports/as-is/details.md"),
      artifact("model/entities/service.yaml", "model"),
    ]);

    expect(groups.map((group) => group.name)).toEqual(["reports/as-is", "reports/findings", "model/entities"]);
    expect(groups[0]?.artifacts.map((item) => item.path)).toEqual(["reports/as-is/details.md", "reports/as-is/overview.md"]);
  });

  it("filters review groups without mutating the source group list", () => {
    const groups: ArtifactGroup[] = [
      { name: "reports/as-is", artifacts: [artifact("reports/as-is/overview.md")] },
      { name: "reports/diagrams", artifacts: [artifact("reports/diagrams/context.mmd", "diagram")] },
    ];

    const filtered = filterReviewArtifactGroups(groups, "diagrams");

    expect(filtered).toEqual([{ name: "reports/diagrams", artifacts: [artifact("reports/diagrams/context.mmd", "diagram")] }]);
    expect(groups).toHaveLength(2);
  });

  it("uses the same path semantics for Publish changed and artifact categories", () => {
    const changed = new Set(["reports/as-is/overview.md"]);

    expect(publishArtifactMatchesFilter(artifact("reports/as-is/overview.md"), "changed", changed)).toBe(true);
    expect(publishArtifactMatchesFilter(artifact("reports/as-is/other.md"), "changed", changed)).toBe(false);
    expect(publishArtifactMatchesFilter(artifact("reports/taskruns/run.json", "taskrun"), "taskruns", changed)).toBe(true);
    expect(reviewArtifactGroupCategory("reports/diagrams")).toBe("is-diagram-group");
    expect(reviewArtifactGroupCategoryLabel("model/entities")).toBe("model");
  });
});
