import { describe, expect, it } from "vitest";

import { buildChangeReviewModel, buildKnowledgeViewModel, buildPublishViewModel } from "./viewModels";

describe("workbench view models", () => {
  it("routes only authoritative successful analysis runs to Change Review", () => {
    const runs = [
      { run_id: "ok", pipeline: "refresh", status: "succeeded", authoritative_index: true },
      { run_id: "failed", pipeline: "init", status: "failed", authoritative_index: false },
      { run_id: "qa", pipeline: "qa", status: "succeeded", authoritative_index: false },
    ] as never;
    expect(buildChangeReviewModel(runs, "ok", "available").map((item) => [item.run_id, item.action])).toEqual([
      ["ok", "review"], ["failed", "run_studio"],
    ]);
  });

  it("keeps valid knowledge visible while filtering a partial response", () => {
    const knowledge = {
      status: "partial", entities: [{ id: "service-a", name: "Payments", type: "service", path: "model/entities/a.yaml" }],
      edges: [], artifacts: [], issues: [{ code: "malformed", message: "bad file" }],
    } as never;
    const model = buildKnowledgeViewModel(knowledge, false, "", "pay", "service-a");
    expect(model.status).toBe("partial");
    expect(model.filteredEntities).toHaveLength(1);
    expect(model.selectedEntity?.id).toBe("service-a");
  });

  it("keeps demo publication wording distinct", () => {
    expect(buildPublishViewModel(2, 1, 0, true).actionLabel).toBe("Commit all demo workspace changes");
    expect(buildPublishViewModel(2, 1, 0, false).actionLabel).toBe("Commit all workspace changes");
  });
});
