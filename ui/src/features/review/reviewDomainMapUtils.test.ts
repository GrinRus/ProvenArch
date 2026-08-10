import { describe, expect, it } from "vitest";

import { deriveReviewDomainMap } from "./reviewDomainMapUtils";

const artifact = (path: string, label = path) => ({ path, label, kind: "model" });

describe("review domain map selectors", () => {
  it("builds canonical groups, typed edges and blockers from model artifacts", () => {
    const model = deriveReviewDomainMap({
      coverageSummary: "coverage",
      openQuestionCount: 1,
      artifacts: [
        artifact("reports/agent-outputs/domains/payments.md", "Payments"),
        artifact("model/entities/svc.payments.yaml", "Payments service"),
        artifact("model/entities/team.platform.yaml", "Platform"),
        artifact("model/edges/edge.svc.payments.calls.api.http.billing.yaml"),
        artifact("proposals/payments/proposal.md"),
      ],
    });
    expect(model.nodes.map((node) => node.group)).toEqual(["domains", "services", "ownership"]);
    expect(model.edges[0]).toMatchObject({ type: "calls", from: "svc.payments", to: "api.http.billing" });
    expect(model.navigationArtifacts).toHaveLength(5);
    expect(model.blockers).toContain("1 open questions remain linked to evidence review.");
  });

  it("fails closed to explicit partial states when only domain outputs exist", () => {
    const model = deriveReviewDomainMap({ artifacts: [artifact("reports/agent-outputs/domains/payments.md")], coverageSummary: "", openQuestionCount: 0 });
    expect(model.entityCount).toBe(0);
    expect(model.coverageStatus).toBe("partial: coverage summary missing");
    expect(model.blockers[0]).toContain("Derived model entities are missing");
  });
});
