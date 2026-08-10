import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { deriveProposalReviewModel } from "./proposalUtils";
import { ProposalPackageRecoveryPanel } from "./ProposalPackageRecoveryPanel";

describe("ProposalPackageRecoveryPanel", () => {
  it("keeps proposal blockers and recovery actions explicit", () => {
    const proposalReview = deriveProposalReviewModel({ artifacts: [], openQuestionCount: 1 });
    const onOpenArtifact = vi.fn();
    const onGoPublish = vi.fn();
    render(
      <ProposalPackageRecoveryPanel
        proposalReview={proposalReview}
        preferredArtifact={{ path: "reports/as-is/overview.md", kind: "markdown", label: "Overview" }}
        proposalBranch=""
        gitStatus="workspace clean"
        onOpenArtifact={onOpenArtifact}
        onGoPublish={onGoPublish}
      />,
    );

    expect(screen.getByTestId("proposal-package-recovery")).toHaveTextContent("No proposal package artifacts are available.");
    screen.getByRole("button", { name: "Open available artifact" }).click();
    screen.getByRole("button", { name: "Check Publish gate" }).click();
    expect(onOpenArtifact).toHaveBeenCalledWith("reports/as-is/overview.md");
    expect(onGoPublish).toHaveBeenCalledTimes(1);
  });
});
