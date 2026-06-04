import { describe, expect, it } from "vitest";

import { pickBootstrapRun } from "./runState";

describe("pickBootstrapRun", () => {
  it("selects the newest active run before completed runs", () => {
    const selected = pickBootstrapRun([
      { run_id: "run-completed-newer", status: "succeeded", started_at: "2026-06-04T12:00:00Z" },
      { run_id: "run-active-older", status: "running", started_at: "2026-06-04T11:00:00Z" },
    ]);

    expect(selected?.run_id).toBe("run-active-older");
  });

  it("selects the newest active run when multiple active runs exist", () => {
    const selected = pickBootstrapRun([
      { run_id: "run-active-old", status: "queued", started_at: "2026-06-04T10:00:00Z" },
      { run_id: "run-active-new", status: "running", started_at: "2026-06-04T10:05:00Z" },
    ]);

    expect(selected?.run_id).toBe("run-active-new");
  });

  it("selects the newest completed run when no active run exists", () => {
    const selected = pickBootstrapRun([
      { run_id: "run-completed-old", status: "succeeded", started_at: "2026-06-04T10:00:00Z" },
      { run_id: "run-completed-new", status: "failed", started_at: "2026-06-04T10:10:00Z" },
    ]);

    expect(selected?.run_id).toBe("run-completed-new");
  });
});
