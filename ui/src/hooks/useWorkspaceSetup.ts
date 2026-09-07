import { useWorkspaceIdentity } from "./useWorkspaceIdentity";
import { useGitActions } from "./useGitActions";
import type { GitPublicationContext } from "../lib/workspaceApi";
import { useManifestEditor } from "./useManifestEditor";
import type { GitDiffResponse } from "../lib/appContracts";

type UseWorkspaceSetupOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
  workspaceKey?: string;
  publicationContext?: GitPublicationContext;
  loadPublicationDiff?: () => Promise<GitDiffResponse | null>;
};

export function useWorkspaceSetup({ setBusy, setError, workspaceKey, publicationContext, loadPublicationDiff }: UseWorkspaceSetupOptions) {
  const manifestEditor = useManifestEditor({ setBusy, setError, workspaceKey });
  const workspaceIdentity = useWorkspaceIdentity();
  const gitActions = useGitActions({ setBusy, setError, publicationContext, loadPublicationDiff });

  async function bootstrapWorkspaceSetup(workspaceKeyOverride?: string) {
    await manifestEditor.loadManifest(workspaceKeyOverride);
    await workspaceIdentity.loadWorkspaceIdentity();
  }

  return {
    ...manifestEditor,
    ...workspaceIdentity,
    ...gitActions,
    bootstrapWorkspaceSetup,
  };
}
