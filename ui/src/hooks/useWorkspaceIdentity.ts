import { useState } from "react";

import { loadBaselineBundleAPI } from "../lib/workspaceApi";
import { isAbortError, useRequestGate } from "./useRequestGate";

export function useWorkspaceIdentity() {
  const [workspaceRootPath, setWorkspaceRootPath] = useState("");
  const identityRequest = useRequestGate("workspace-identity");

  async function loadWorkspaceIdentity() {
    const token = identityRequest.begin();
    try {
      const payload = await loadBaselineBundleAPI({ signal: token.signal });
      if (!identityRequest.isCurrent(token)) return;
      setWorkspaceRootPath(payload.workspace ?? "");
    } catch (error) {
      if (isAbortError(error) || !identityRequest.isCurrent(token)) return;
      setWorkspaceRootPath("");
    } finally {
      identityRequest.finish(token);
    }
  }

  return { workspaceRootPath, loadWorkspaceIdentity };
}
