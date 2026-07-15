import { describe, expect, it } from "vitest";

import { defaultStageForDestination, destinationForStage, destinationFromPath } from "./appRoutes";

describe("application routes", () => {
  it("canonicalizes every path to setup before Console entry", () => {
    expect(destinationFromPath("/changes", false)).toBe("setup");
  });

  it("maps the five direct paths and canonicalizes unknown paths to home", () => {
    expect(destinationFromPath("/runs", true)).toBe("runs");
    expect(destinationFromPath("/knowledge", true)).toBe("knowledge");
    expect(destinationFromPath("/does-not-exist", true)).toBe("home");
  });

  it("keeps stage utilities reachable from their owning destination", () => {
    expect(destinationForStage("analysis")).toBe("runs");
    expect(destinationForStage("publish")).toBe("changes");
    expect(defaultStageForDestination("changes")).toBe("review");
  });
});
