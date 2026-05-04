import { useBaselineEditor } from "./useBaselineEditor";
import { useGitActions } from "./useGitActions";
import { useManifestEditor } from "./useManifestEditor";
import { useWizardEditor } from "./useWizardEditor";

type UseWorkspaceSetupOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useWorkspaceSetup({ setBusy, setError }: UseWorkspaceSetupOptions) {
  const manifestEditor = useManifestEditor({ setBusy, setError });
  const baselineEditor = useBaselineEditor({ setBusy, setError });
  const wizardEditor = useWizardEditor({ setBusy, setError });
  const gitActions = useGitActions({ setBusy, setError });

  async function bootstrapWorkspaceSetup() {
    await manifestEditor.loadManifest();
    await baselineEditor.loadBaselineBundle();
    await wizardEditor.loadWizardContract();
  }

  return {
    ...manifestEditor,
    ...baselineEditor,
    ...wizardEditor,
    ...gitActions,
    bootstrapWorkspaceSetup,
  };
}
