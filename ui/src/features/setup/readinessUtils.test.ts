import { describe, expect, it } from "vitest";

import { workspaceHealthLabel, workspaceHealthTone } from "./readinessUtils";

describe("readinessUtils", () => {
  it("keeps loading and failure states stronger than absent data", () => {
    expect(workspaceHealthTone(null, "loading")).toBe("info");
    expect(workspaceHealthTone(null, "error")).toBe("error");
    expect(workspaceHealthLabel(null, "loading")).toBe("scanning");
    expect(workspaceHealthLabel(null, "error")).toBe("scan failed");
  });

  it("maps loaded report severity and label", () => {
    expect(workspaceHealthTone({ status: "warn" } as never, "loaded")).toBe("warn");
    expect(workspaceHealthTone({ status: "fail" } as never, "loaded")).toBe("error");
    expect(workspaceHealthLabel({ status: "ok" } as never, "loaded")).toBe("ok");
  });
});
