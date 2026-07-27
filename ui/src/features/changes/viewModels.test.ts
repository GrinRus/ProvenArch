import { describe, expect, it } from "vitest";

import { buildChangesRouteModel } from "./viewModels";

describe("Changes route models", () => {
  it.each([
    ["overview", "Review overview"],
    ["evidence", "Evidence"],
    ["findings", "Findings"],
    ["diff", "Workspace diff"],
    ["proposals", "Proposals"],
    ["publish", "Publish"],
  ] as const)("builds a distinct %s model", (view, title) => {
    expect(buildChangesRouteModel(view, "snapshot")).toEqual(expect.objectContaining({
      kind: view,
      source: "snapshot",
      title,
    }));
  });
});
