import { useState } from "react";

import { loadBaselineBundleAPI } from "../lib/workspaceApi";

export function useWorkspaceIdentity() {
  const [workspaceRootPath, setWorkspaceRootPath] = useState("");

  async function loadWorkspaceIdentity() {
    try {
      const payload = await loadBaselineBundleAPI();
      setWorkspaceRootPath(payload.workspace ?? "");
    } catch {
      setWorkspaceRootPath("");
    }
  }

  return { workspaceRootPath, loadWorkspaceIdentity };
}
