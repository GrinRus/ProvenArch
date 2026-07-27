import { useCallback, useState } from "react";

import type { GitDiffResponse } from "../lib/appContracts";
import { loadWorkspaceGitDiff, type LoadGitDiffOptions } from "../lib/gitDiffApi";
import { isAbortError, useRequestGate } from "./useRequestGate";

export function useGitDiff() {
  const [gitDiff, setGitDiff] = useState<GitDiffResponse | null>(null);
  const [gitDiffStatus, setGitDiffStatus] = useState("");
  const diffRequest = useRequestGate("git-diff");

  const loadGitDiff = useCallback(async (options: LoadGitDiffOptions = {}) => {
    const token = diffRequest.begin(gitDiffRequestKey(options));
    setGitDiff(null);
    setGitDiffStatus("Loading workspace Git diff.");
    try {
      const payload = await loadWorkspaceGitDiff({ ...options, signal: token.signal });
      if (!diffRequest.isCurrent(token)) {
        return null;
      }
      setGitDiff(payload);
      setGitDiffStatus(gitTruthStatus(payload));
      return payload;
    } catch (error) {
      if (isAbortError(error) || !diffRequest.isCurrent(token)) {
        return null;
      }
      setGitDiffStatus(error instanceof Error ? error.message : "Workspace Git diff failed to load.");
      return null;
    } finally {
      diffRequest.finish(token);
    }
  }, [diffRequest]);

  return {
    gitDiff,
    gitDiffStatus,
    loadGitDiff,
  };
}

function gitDiffRequestKey(options: LoadGitDiffOptions): string {
  return [
    options.runId ?? "",
    options.path ?? "",
    options.folder ?? "",
    options.stepId ?? "",
    options.fingerprint ?? "",
  ].join("|");
}

function gitTruthStatus(payload: GitDiffResponse): string {
  switch (payload.state) {
    case "clean": return "No workspace Git changes.";
    case "stale": return "Workspace Git state changed after the supplied confirmation.";
    case "blocked": return "Git publication is blocked while analysis is active or queued.";
    case "unknown": return payload.message || "Workspace Git state is unavailable.";
    case "dirty": return "";
  }
}
