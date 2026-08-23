import { useState } from "react";

import type { GitDiffResponse } from "../lib/appContracts";
import { loadWorkspaceGitDiff } from "../lib/gitDiffApi";
import { commitWorkspaceArtifacts, createProposalBranch, type GitPublicationContext } from "../lib/workspaceApi";

type UseGitActionsOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
  publicationContext?: GitPublicationContext;
  loadPublicationDiff?: () => Promise<GitDiffResponse | null>;
};

export function useGitActions({ setBusy, setError, publicationContext, loadPublicationDiff }: UseGitActionsOptions) {
  const [gitMessage, setGitMessage] = useState("chore: update ACP workspace artifacts");
  const [proposalBranch, setProposalBranch] = useState("proposal/beta-refresh");
  const [gitStatus, setGitStatus] = useState<string>("");
  const [gitError, setGitError] = useState<string>("");
  const [gitConfirmation, setGitConfirmation] = useState<{ action: "commit" | "branch"; diff: GitDiffResponse } | null>(null);

  async function handleGitCommit() {
	await prepareGitConfirmation("commit");
  }

  async function handleCreateProposalBranch() {
	await prepareGitConfirmation("branch");
  }

  async function prepareGitConfirmation(action: "commit" | "branch") {
    setBusy(true);
    setError(null);
    setGitStatus("");
    setGitError("");
    try {
      if (action === "commit" && !gitMessage.trim()) {
        throw new Error("Commit message is required.");
      }
      if (action === "branch" && !proposalBranch.trim()) {
        throw new Error("Proposal branch name is required.");
      }
      const diff = await (loadPublicationDiff?.() ?? loadWorkspaceGitDiff());
      if (!diff) throw new Error("Authoritative workspace Git inventory could not be loaded.");
      if (diff.state === "blocked" || diff.state === "unknown" || diff.state === "stale") {
        throw new Error(diff.message || `Git state is ${diff.state}; refresh and resolve it before publication.`);
      }
      if (action === "commit" && diff.empty) {
        throw new Error("There are no workspace changes to commit.");
      }
      setGitConfirmation({ action, diff });
    } catch (requestError) {
      const message = requestError instanceof Error ? requestError.message : "Git confirmation failed to load";
      setGitError(`Git confirmation failed: ${message}`);
      setError(message);
    } finally {
      setBusy(false);
    }
  }

  async function confirmGitAction() {
    if (!gitConfirmation) {
      return;
    }
    setBusy(true);
    setError(null);
    setGitStatus("");
    setGitError("");
    try {
      const identity = {
        fingerprint: gitConfirmation.diff.fingerprint,
        headOID: gitConfirmation.diff.head_oid ?? "",
        sourceBranch: gitConfirmation.diff.branch,
        baseRef: gitConfirmation.diff.base_ref,
        baseOID: gitConfirmation.diff.base_oid ?? "",
      };
      if (gitConfirmation.action === "commit") {
        const payload = await commitWorkspaceArtifacts(gitMessage, identity, publicationContext);
        const publicationLabel = payload.publication?.state === "linked"
          ? `Task publication linked to Attempt ${payload.publication.attempt_id ?? "unknown"}`
          : payload.publication?.unavailable_reason;
        setGitStatus([payload.output ?? payload.message ?? payload.status, publicationLabel].filter(Boolean).join(" · "));
      } else {
        const payload = await createProposalBranch(proposalBranch, identity, publicationContext);
        setGitStatus(`checked out ${payload.branch}`);
      }
      try {
        await loadPublicationDiff?.();
      } catch {
        // The Git mutation already succeeded; leave its result visible even if
        // the follow-up inventory refresh is temporarily unavailable.
      }
      setGitConfirmation(null);
    } catch (requestError) {
      const message = requestError instanceof Error ? requestError.message : "Git mutation failed";
      setGitError(`Git mutation failed: ${message}`);
      setError(message);
      setGitConfirmation(null);
    } finally {
      setBusy(false);
    }
  }

  return {
    gitMessage,
    proposalBranch,
    gitStatus,
    gitError,
    gitConfirmation,
    setGitMessage,
    setProposalBranch,
    handleGitCommit,
    handleCreateProposalBranch,
    confirmGitAction,
    cancelGitAction: () => setGitConfirmation(null),
  };
}
