import { describe, expect, it } from "vitest";

import { buildProvisionalQARun, mergeQARunHistory, normalizeQARunStatus, qaFailureGuidance } from "./qaUtils";

describe("QA view-model selectors", () => {
  it("normalizes provisional runs and keeps the question auditable", () => {
    const run = buildProvisionalQARun({ run_id: "qa-1", status: "unexpected" }, "Where is ownership defined?");
    expect(run.status).toBe("queued");
    expect(run.current_step).toBe("qa.ask");
    expect(run.question).toBe("Where is ownership defined?");
    expect(normalizeQARunStatus("succeeded")).toBe("succeeded");
  });

  it("merges refreshed history without duplicate runs and caps retention", () => {
    const history = Array.from({ length: 21 }, (_, index) => buildProvisionalQARun({ run_id: `qa-${index}`, status: "queued" }, "question"));
    const merged = mergeQARunHistory({ ...history[0], status: "succeeded" }, history, "preserve");
    expect(merged).toHaveLength(20);
    expect(merged.filter((run) => run.run_id === "qa-0")).toHaveLength(1);
    expect(merged.find((run) => run.run_id === "qa-0")?.status).toBe("succeeded");
  });

  it("keeps failure guidance actionable for blocked runtime contracts", () => {
    expect(qaFailureGuidance("runtime_contract_failed", 0)).toContain("validation");
    expect(qaFailureGuidance("unknown", 1)).toContain("warnings");
  });
});
