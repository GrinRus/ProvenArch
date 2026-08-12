import { useBaselineEditor } from "./useBaselineEditor";
import { useGitActions } from "./useGitActions";
import type { GitPublicationContext } from "../lib/workspaceApi";
import { useManifestEditor } from "./useManifestEditor";
import { useWizardEditor } from "./useWizardEditor";

type UseWorkspaceSetupOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
  publicationContext?: GitPublicationContext;
};

export function useWorkspaceSetup({ setBusy, setError, publicationContext }: UseWorkspaceSetupOptions) {
  const manifestEditor = useManifestEditor({ setBusy, setError });
  const baselineEditor = useBaselineEditor({ setBusy, setError });
  const wizardEditor = useWizardEditor({ setBusy, setError });
  const gitActions = useGitActions({ setBusy, setError, publicationContext });

  async function bootstrapWorkspaceSetup() {
    await manifestEditor.loadManifest();
    await baselineEditor.loadBaselineBundle();
  }

  return {
    ...manifestEditor,
    ...baselineEditor,
    ...wizardEditor,
    ...gitActions,
    bootstrapWorkspaceSetup,
  };
}
