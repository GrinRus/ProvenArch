import { describe, expect, it } from "vitest";

import { getConsoleRouteKind } from "./AppConsoleView";
import type { AppRoute } from "../lib/appRoutes";

function route(overrides: Partial<AppRoute>): AppRoute {
  return { destination: "tasks", invalid: [], ...overrides };
}

describe("getConsoleRouteKind", () => {
  it.each([
    [route({ destination: "setup" }), "setup"],
    [route({ destination: "knowledge" }), "knowledge"],
    [route({ destination: "changes" }), "changes"],
    [route({ destination: "settings" }), "settings"],
    [route({ destination: "tasks", taskView: "new" }), "task-composer"],
    [route({ destination: "tasks", taskView: "inbox" }), "tasks"],
    [route({ destination: "tasks", taskView: "detail", taskId: "task-1" }), "tasks"],
    [route({ destination: "tasks", taskView: "legacy" }), "legacy-analysis"],
  ] as const)("maps %j to %s", (input, expected) => {
    expect(getConsoleRouteKind(input, "analysis")).toBe(expected);
  });

  it("does not render a legacy route when its stage is no longer analysis", () => {
    expect(getConsoleRouteKind(route({ destination: "tasks", taskView: "legacy" }), "review")).toBe("empty");
  });
});
