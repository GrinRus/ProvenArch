import { useCallback, useState } from "react";

import type { GitDiffResponse } from "../lib/appContracts";
import { loadWorkspaceGitDiff, type LoadGitDiffOptions } from "../lib/gitDiffApi";

export function useGitDiff() {
  const [gitDiff, setGitDiff] = useState<GitDiffResponse | null>(null);
  const [gitDiffStatus, setGitDiffStatus] = useState("");
  const [selectedDiffPath, setSelectedDiffPath] = useState("");

  const loadGitDiff = useCallback(async (options: LoadGitDiffOptions = {}) => {
    setGitDiffStatus("Loading workspace Git diff.");
    try {
      const payload = await loadWorkspaceGitDiff(options);
      setGitDiff(payload);
      setSelectedDiffPath(payload.selected_file?.path ?? payload.selected_path ?? options.path ?? "");
      setGitDiffStatus(payload.empty ? "No workspace Git changes." : "");
      return payload;
    } catch (error) {
      setGitDiffStatus(error instanceof Error ? error.message : "Workspace Git diff failed to load.");
      return null;
    }
  }, []);

  const clearGitDiff = useCallback(() => {
    setGitDiff(null);
    setGitDiffStatus("");
    setSelectedDiffPath("");
  }, []);

  return {
    gitDiff,
    gitDiffStatus,
    selectedDiffPath,
    setSelectedDiffPath,
    loadGitDiff,
    clearGitDiff,
  };
}
