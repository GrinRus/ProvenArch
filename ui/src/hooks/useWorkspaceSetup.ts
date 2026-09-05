import { useWorkspaceIdentity } from "./useWorkspaceIdentity";
import { useGitActions } from "./useGitActions";
import type { GitPublicationContext } from "../lib/workspaceApi";
import { useManifestEditor } from "./useManifestEditor";
import type { GitDiffResponse } from "../lib/appContracts";

type UseWorkspaceSetupOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
  publicationContext?: GitPublicationContext;
  loadPublicationDiff?: () => Promise<GitDiffResponse | null>;
};

export function useWorkspaceSetup({ setBusy, setError, publicationContext, loadPublicationDiff }: UseWorkspaceSetupOptions) {
  const manifestEditor = useManifestEditor({ setBusy, setError });
  const workspaceIdentity = useWorkspaceIdentity();
  const gitActions = useGitActions({ setBusy, setError, publicationContext, loadPublicationDiff });

  async function bootstrapWorkspaceSetup() {
    await manifestEditor.loadManifest();
    await workspaceIdentity.loadWorkspaceIdentity();
  }

  return {
    ...manifestEditor,
    ...workspaceIdentity,
    ...gitActions,
    bootstrapWorkspaceSetup,
  };
}
