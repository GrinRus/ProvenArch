import { describe, expect, it } from "vitest";

import { destinationForStage, formatAppRoute, parseAppRoute, stageForRoute } from "./appRoutes";

const location = (value: string) => {
  const url = new URL(value, "http://localhost");
  return { pathname: url.pathname, search: url.search } as Location;
};

describe("application route codec", () => {
  it("canonicalizes every route to setup before Console entry", () => {
    expect(parseAppRoute(location("/changes?run=old"), false)).toMatchObject({ destination: "setup", setupStep: "workspace" });
  });

  it("round-trips run and Changes context", () => {
    const changes = parseAppRoute(location("/changes?run=run-1&view=evidence&source=snapshot&artifact=doc.overview&mode=raw"), true);
    expect(changes).toMatchObject({ destination: "changes", runId: "run-1", changesView: "evidence", source: "snapshot", artifact: "doc.overview", mode: "raw" });
    expect(formatAppRoute(changes)).toBe("/changes?run=run-1&view=evidence&source=snapshot&artifact=doc.overview&mode=raw");
    expect(formatAppRoute(parseAppRoute(location("/runs/run-1"), true))).toBe("/runs/run-1");
  });

  it("sanitizes unsupported values without changing source", () => {
    const route = parseAppRoute(location("/changes?run=missing&view=magic&source=snapshot&mode=html"), true);
    expect(route.invalid).toEqual(["view", "mode"]);
    expect(route).toMatchObject({ runId: "missing", changesView: "overview", source: "snapshot", mode: "rendered" });
  });

  it("maps setup and stage ownership", () => {
    expect(stageForRoute(parseAppRoute(location("/setup?step=runner"), true))).toBe("readiness");
    expect(destinationForStage("publish")).toBe("changes");
  });
});
