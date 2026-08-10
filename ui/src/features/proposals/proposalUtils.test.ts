import { describe, expect, it } from "vitest";

import { deriveProposalArtifactType, deriveProposalReviewModel, proposalPackageSuggestedFix, proposalTabLabel } from "./proposalUtils";

const artifact = (path: string, kind = "markdown") => ({ path, kind, label: path });

describe("proposal review selectors", () => {
  it("groups proposal, changelog and evidence artifacts without inventing blockers", () => {
    const model = deriveProposalReviewModel({
      artifacts: [
        artifact("proposals/orders/ADR.md"),
        artifact("proposals/orders/proposal.md"),
        artifact("reports/changelog/orders.md"),
        artifact("reports/findings/orders.md"),
      ],
      openQuestionCount: 0,
    });

    expect(model.packages.map((group) => group.name)).toEqual(["proposals/orders", "reports/changelog"]);
    expect(model.proposalDocumentCount).toBe(2);
    expect(model.adrRfcCount).toBe(1);
    expect(model.evidenceArtifacts).toHaveLength(1);
    expect(model.blockers).toEqual([]);
  });

  it("keeps missing package and open-question recovery actionable", () => {
    const model = deriveProposalReviewModel({ artifacts: [], openQuestionCount: 2 });
    expect(model.blockers).toEqual([
      "No proposal package artifacts are available.",
      "2 open questions remain from evidence review.",
    ]);
    expect(proposalPackageSuggestedFix(model.blockers[0])).toContain("step4.proposals");
  });

  it("uses stable labels for proposal tabs and artifact types", () => {
    expect(proposalTabLabel("logs")).toBe("Logs");
    expect(deriveProposalArtifactType("proposals/orders/RFC.md")).toBe("RFC");
    expect(deriveProposalArtifactType("reports/changelog/orders.md")).toBe("changelog");
  });
});
