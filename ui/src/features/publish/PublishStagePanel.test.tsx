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

  it("keeps Git mutations blocked when the Task review context is stale", () => {
    render(
      <PublishStagePanel
        busy={false}
        gitMessage="publish"
        proposalBranch="proposal/current"
        gitStatus=""
        gitError=""
        artifacts={[]}
        selectedArtifact=""
        selectedArtifactContent=""
        openQuestions=""
        externalGateItems={[{ label: "Fresh review evidence", detail: "Review belongs to another run.", tone: "error" }]}
        gitDiff={{ ok: true, state: "dirty", workspace: "/workspace", scope: "full_workspace", branch: "main", head_oid: "head", base_ref: "HEAD", base_oid: "base", fingerprint: "fingerprint", run_id: null, step_id: null, selected_path: null, selected_file: null, files: [{ path: "reports/as-is/overview.md", folder: "reports/as-is", status: "modified", additions: 1, deletions: 0, binary: false, index_status: "M", worktree_status: "M", unavailable: false }], folders: [{ folder: "reports/as-is", files: 1, additions: 1, deletions: 0 }], hunks: [], empty: false }}
        gitDiffStatus=""
        onGitMessageChange={vi.fn()}
        onProposalBranchChange={vi.fn()}
        onCommit={vi.fn()}
        onCreateProposalBranch={vi.fn()}
        onLoadGitDiff={vi.fn()}
        onPreviewArtifact={vi.fn()}
      />,
    );
    expect(screen.getByTestId("publish-commit-selected-btn")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Create/Switch proposal branch" })).toBeDisabled();
  });
});
