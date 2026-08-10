import { describe, expect, it } from "vitest";

import { failureEvidenceSummary, failureRecoveryGuidance } from "./analysisRecoveryUtils";

describe("analysisRecoveryUtils", () => {
  it("keeps evidence summaries explicit for each retained state", () => {
    expect(failureEvidenceSummary(2, 4)).toBe("2 artifact refs kept");
    expect(failureEvidenceSummary(0, 3)).toBe("3 diagnostic rows");
    expect(failureEvidenceSummary(0, 0)).toBe("status and logs only");
  });

  it("prioritizes actionable recovery guidance", () => {
    expect(failureRecoveryGuidance("runtime_failed", 1)).toContain("pending permission");
    expect(failureRecoveryGuidance("runtime_timeout", 0)).toContain("time budget");
    expect(failureRecoveryGuidance("runtime_contract_failed", 0)).toContain("validation");
  });
});
