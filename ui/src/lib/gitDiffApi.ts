import { fetchJSON } from "./api";
import type { GitDiffResponse } from "./appContracts";

export type LoadGitDiffOptions = {
  path?: string;
  folder?: string;
  runId?: string | null;
  stepId?: string | null;
  signal?: AbortSignal;
};

export async function loadWorkspaceGitDiff(options: LoadGitDiffOptions = {}): Promise<GitDiffResponse> {
  const query = new URLSearchParams();
  if (options.path) {
    query.set("path", options.path);
  }
  if (options.folder) {
    query.set("folder", options.folder);
  }
  if (options.runId) {
    query.set("run_id", options.runId);
  }
  if (options.stepId) {
    query.set("step_id", options.stepId);
  }
  const suffix = query.toString() ? `?${query.toString()}` : "";
  return fetchJSON<GitDiffResponse>(`/api/git/diff${suffix}`, { signal: options.signal });
}
