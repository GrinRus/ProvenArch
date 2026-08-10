import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AnalysisStepReview } from "./AnalysisStepReview";

describe("AnalysisStepReview", () => {
  it("keeps empty and review states explicit", () => {
    render(<AnalysisStepReview steps={[]} selectedStep={null} runtimeMode="fake" runReviewStatus="Review is pending" runLogs={[]} gitDiff={null} gitDiffStatus="" view="artifacts" onViewChange={vi.fn()} onSelectStep={vi.fn()} onOpenArtifact={vi.fn()} onLoadGitDiff={vi.fn()} />);
    expect(screen.getByTestId("analysis-step-review-panel")).toHaveTextContent("Review is pending");
    expect(screen.getByText(/No review summary is available for the selected run yet/)).toBeInTheDocument();
  });
});
