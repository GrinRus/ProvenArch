import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ReviewEvidenceWorkbench } from "./ReviewEvidenceWorkbench";

describe("ReviewEvidenceWorkbench", () => {
  it("keeps the evidence preview and review trust regions visible", () => {
    render(<ReviewEvidenceWorkbench routeView="overview" coverageSummary="Coverage ready" openQuestions="" openQuestionCount={0} trustStatus={{ label: "ready", title: "Evidence ready", detail: "Ready", tone: "ok" }} artifactGroups={[]} diagramArtifacts={[]} selectedArtifact="" selectedArtifactContent="" evidenceStatus="available" evidenceIssues={[]} selectedArtifactIsLoading={false} reviewSummary={null} demo={false} reviewQueue={[]} gitDiff={null} gitDiffStatus="" onLoadGitDiff={() => undefined} onOpenArtifact={() => undefined} />);
    expect(screen.getByTestId("review-evidence-preview")).toBeInTheDocument();
    expect(screen.getByTestId("review-citation-coverage")).toHaveTextContent("Evidence ready");
  });
});
