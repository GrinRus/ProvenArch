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
      setGitDiffStatus(payload.empty ? "No workspace Git changes." : "");
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
  ].join("|");
}
