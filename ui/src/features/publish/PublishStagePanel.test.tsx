import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { PublishStagePanel } from "./PublishStagePanel";

describe("PublishStagePanel", () => {
  it("keeps full-workspace publication truth visible while inventory is unavailable", () => {
    render(
      <PublishStagePanel
        busy={false}
        gitMessage=""
        proposalBranch=""
        gitStatus=""
        gitError=""
        artifacts={[]}
        selectedArtifact=""
        selectedArtifactContent=""
        openQuestions=""
        gitDiff={null}
        gitDiffStatus=""
        onGitMessageChange={vi.fn()}
        onProposalBranchChange={vi.fn()}
        onCommit={vi.fn()}
        onCreateProposalBranch={vi.fn()}
        onLoadGitDiff={vi.fn()}
        onPreviewArtifact={vi.fn()}
      />,
    );
    expect(screen.getByTestId("publish-panel")).toBeInTheDocument();
    expect(screen.getByText("Loading authoritative workspace scope")).toBeInTheDocument();
    expect(screen.getByTestId("publish-commit-selected-btn")).toBeDisabled();
  });
});

