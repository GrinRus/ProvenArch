import { useState } from "react";

import type { Diagnostic, EditableArtifactOption } from "../lib/appContracts";
import { loadArtifactText, loadBaselineBundleAPI, saveEditableArtifact } from "../lib/workspaceApi";

type UseBaselineEditorOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useBaselineEditor({ setBusy, setError }: UseBaselineEditorOptions) {
  const [baselineEditorArtifacts, setBaselineEditorArtifacts] = useState<EditableArtifactOption[]>([]);
  const [baselineBundleWarnings, setBaselineBundleWarnings] = useState<Diagnostic[]>([]);
  const [selectedEditorPath, setSelectedEditorPath] = useState<string>("");
  const [selectedEditorContent, setSelectedEditorContent] = useState("");
  const [editorStatus, setEditorStatus] = useState("");

  async function loadTextArtifact(path: string, setter: (value: string) => void) {
    try {
      setter((await loadArtifactText(path)) ?? "");
    } catch {
      setter("");
    }
  }

  async function loadBaselineBundle() {
    try {
      const payload = await loadBaselineBundleAPI();
      const artifacts = (payload.manifest?.editable_artifacts ?? []).map((artifact) => ({
        path: artifact.path,
        label: artifact.label,
      }));
      setBaselineEditorArtifacts(artifacts);
      setBaselineBundleWarnings(payload.warnings ?? []);
      const hasCurrentSelection = artifacts.some((artifact) => artifact.path === selectedEditorPath);
      const nextPath = hasCurrentSelection ? selectedEditorPath : (artifacts[0]?.path ?? "");
      setSelectedEditorPath(nextPath);
      if (nextPath) {
        await loadTextArtifact(nextPath, setSelectedEditorContent);
      } else {
        setSelectedEditorContent("");
      }
    } catch {
      setBaselineEditorArtifacts([]);
      setBaselineBundleWarnings([]);
      setSelectedEditorPath("");
      setSelectedEditorContent("");
    }
  }

  async function handleEditorSelectionChange(path: string) {
    setSelectedEditorPath(path);
    await loadTextArtifact(path, setSelectedEditorContent);
  }

  async function handleSaveSelectedEditorArtifact() {
    setBusy(true);
    setError(null);
    setEditorStatus("");
    try {
      await saveEditableArtifact(selectedEditorPath, selectedEditorContent);
      setEditorStatus(`Saved ${selectedEditorPath}`);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save editor artifact");
    } finally {
      setBusy(false);
    }
  }

  return {
    baselineEditorArtifacts,
    baselineBundleWarnings,
    selectedEditorPath,
    selectedEditorContent,
    editorStatus,
    loadBaselineBundle,
    setSelectedEditorContent,
    handleEditorSelectionChange,
    handleSaveSelectedEditorArtifact,
  };
}
