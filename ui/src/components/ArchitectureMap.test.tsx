import { describe, expect, it } from "vitest";

import { boundedLargeArchitecture } from "../test/fixtures/architectureLarge";
import { layoutArchitecture } from "./ArchitectureMap";

describe("ArchitectureMap layout", () => {
  it("lays out the bounded large-graph fixture with finite readable positions", async () => {
    const fixture = boundedLargeArchitecture(80);
    const layout = await layoutArchitecture(fixture.nodes, fixture.edges);
    expect(layout.nodes).toHaveLength(80);
    expect(layout.edges).toHaveLength(79);
    expect(layout.nodes.every((node) => Number.isFinite(node.position.x) && Number.isFinite(node.position.y))).toBe(true);
    expect(new Set(layout.nodes.map((node) => `${node.position.x}:${node.position.y}`)).size).toBe(80);
  });
});
