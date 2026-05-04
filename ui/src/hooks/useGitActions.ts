import { useState } from "react";

import { commitWorkspaceArtifacts, createProposalBranch } from "../lib/workspaceApi";

type UseGitActionsOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useGitActions({ setBusy, setError }: UseGitActionsOptions) {
  const [gitMessage, setGitMessage] = useState("chore: update ACP workspace artifacts");
  const [proposalBranch, setProposalBranch] = useState("proposal/beta-refresh");
  const [gitStatus, setGitStatus] = useState<string>("");

  async function handleGitCommit() {
    setBusy(true);
    setError(null);
    setGitStatus("");
    try {
      const payload = await commitWorkspaceArtifacts(gitMessage);
      setGitStatus(payload.output ?? payload.message ?? payload.status);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "git commit failed");
    } finally {
      setBusy(false);
    }
  }

  async function handleCreateProposalBranch() {
    setBusy(true);
    setError(null);
    setGitStatus("");
    try {
      const payload = await createProposalBranch(proposalBranch);
      setGitStatus(`checked out ${payload.branch}`);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to create proposal branch");
    } finally {
      setBusy(false);
    }
  }

  return {
    gitMessage,
    proposalBranch,
    gitStatus,
    setGitMessage,
    setProposalBranch,
    handleGitCommit,
    handleCreateProposalBranch,
  };
}
