import { useRef, useState } from "react";

import type { Diagnostic, EditableArtifactOption } from "../lib/appContracts";
import { loadArtifactText, loadBaselineBundleAPI, saveEditableArtifact } from "../lib/workspaceApi";

type UseBaselineEditorOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

type EditorDraft = {
  content: string;
  loadedContent: string;
  dirty: boolean;
  loaded: boolean;
  revision: number;
};

export function useBaselineEditor({ setBusy, setError }: UseBaselineEditorOptions) {
  const [baselineEditorArtifacts, setBaselineEditorArtifacts] = useState<EditableArtifactOption[]>([]);
  const [baselineBundleWarnings, setBaselineBundleWarnings] = useState<Diagnostic[]>([]);
  const [workspaceRootPath, setWorkspaceRootPath] = useState("");
  const [selectedEditorPath, setSelectedEditorPath] = useState<string>("");
  const [selectedEditorContent, setSelectedEditorContent] = useState("");
  const [selectedEditorLoadedPath, setSelectedEditorLoadedPath] = useState("");
  const [editorStatus, setEditorStatus] = useState("");
  const selectedEditorPathRef = useRef("");
  const draftsRef = useRef(new Map<string, EditorDraft>());
  const loadSequenceRef = useRef(0);

  function getDraft(path: string): EditorDraft {
    let draft = draftsRef.current.get(path);
    if (!draft) {
      draft = { content: "", loadedContent: "", dirty: false, loaded: false, revision: 0 };
      draftsRef.current.set(path, draft);
    }
    return draft;
  }

  function setSelectedPath(path: string) {
    selectedEditorPathRef.current = path;
    setSelectedEditorPath(path);
  }

  function showSelectedDraft(path: string, draft: EditorDraft, loadedPath = path) {
    if (selectedEditorPathRef.current !== path) {
      return;
    }
    setSelectedEditorContent(draft.content);
    setSelectedEditorLoadedPath(loadedPath);
  }

  async function loadSelectedEditorContent(path = selectedEditorPathRef.current || selectedEditorPath) {
    if (!path) {
      selectedEditorPathRef.current = "";
      setSelectedEditorContent("");
      setSelectedEditorLoadedPath("");
      return;
    }

    const draft = getDraft(path);
    if (draft.dirty || draft.loaded) {
      showSelectedDraft(path, draft);
      return;
    }

    const loadID = (loadSequenceRef.current += 1);
    const startRevision = draft.revision;
    setSelectedEditorLoadedPath(path);
    try {
      const content = (await loadArtifactText(path)) ?? "";
      const currentDraft = getDraft(path);
      if (loadID !== loadSequenceRef.current || selectedEditorPathRef.current !== path) {
        return;
      }
      if (currentDraft.dirty && currentDraft.revision !== startRevision) {
        showSelectedDraft(path, currentDraft);
        return;
      }
      currentDraft.content = content;
      currentDraft.loadedContent = content;
      currentDraft.loaded = true;
      currentDraft.dirty = false;
      showSelectedDraft(path, currentDraft);
    } catch {
      const currentDraft = getDraft(path);
      if (loadID === loadSequenceRef.current && selectedEditorPathRef.current === path && !currentDraft.dirty) {
        currentDraft.content = "";
        currentDraft.loadedContent = "";
        currentDraft.loaded = true;
        currentDraft.dirty = false;
        showSelectedDraft(path, currentDraft);
      }
    }
  }

  async function loadBaselineBundle() {
    try {
      const payload = await loadBaselineBundleAPI();
      setWorkspaceRootPath(payload.workspace ?? "");
      const artifacts = (payload.manifest?.editable_artifacts ?? []).map((artifact) => ({
        path: artifact.path,
        label: artifact.label,
        category: artifact.category,
        prompt_usage: artifact.prompt_usage,
      }));
      setBaselineEditorArtifacts(artifacts);
      setBaselineBundleWarnings(payload.warnings ?? []);
      const currentPath = selectedEditorPathRef.current || selectedEditorPath;
      const hasCurrentSelection = artifacts.some((artifact) => artifact.path === currentPath);
      const nextPath = hasCurrentSelection ? currentPath : "";
      setSelectedPath(nextPath);
      if (nextPath) {
        const draft = draftsRef.current.get(nextPath);
        if (draft?.dirty || draft?.loaded) {
          showSelectedDraft(nextPath, draft);
        } else {
          setSelectedEditorContent("");
          setSelectedEditorLoadedPath("");
        }
      } else {
        setSelectedEditorContent("");
        setSelectedEditorLoadedPath("");
      }
    } catch {
      setBaselineEditorArtifacts([]);
      setBaselineBundleWarnings([]);
      setWorkspaceRootPath("");
      setSelectedPath("");
      setSelectedEditorContent("");
      setSelectedEditorLoadedPath("");
    }
  }

  async function handleEditorSelectionChange(path: string) {
    setSelectedPath(path);
    setEditorStatus("");
    if (!path) {
      setSelectedEditorContent("");
      setSelectedEditorLoadedPath("");
      return;
    }
    const draft = getDraft(path);
    showSelectedDraft(path, draft);
    await loadSelectedEditorContent(path);
  }

  function updateSelectedEditorContent(value: string) {
    const path = selectedEditorPathRef.current;
    setEditorStatus("");
    setSelectedEditorContent(value);
    if (!path) {
      return;
    }
    const draft = getDraft(path);
    draft.content = value;
    draft.dirty = value !== draft.loadedContent;
    draft.loaded = true;
    draft.revision += 1;
    setSelectedEditorLoadedPath(path);
  }

  async function handleSaveSelectedEditorArtifact() {
    const path = selectedEditorPathRef.current || selectedEditorPath;
    if (!path) {
      return;
    }
    const draft = getDraft(path);
    const saveRevision = draft.revision;
    const saveContent = draft.content;
    setBusy(true);
    setError(null);
    setEditorStatus("");
    try {
      await saveEditableArtifact(path, saveContent);
      const currentDraft = getDraft(path);
      if (currentDraft.revision === saveRevision && currentDraft.content === saveContent) {
        currentDraft.loadedContent = saveContent;
        currentDraft.dirty = false;
        currentDraft.loaded = true;
        setEditorStatus(`Saved ${path}`);
      } else {
        setEditorStatus(`Saved ${path}; newer unsaved edits remain.`);
      }
      if (selectedEditorPathRef.current === path) {
        showSelectedDraft(path, currentDraft);
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save editor artifact");
    } finally {
      setBusy(false);
    }
  }

  return {
    baselineEditorArtifacts,
    baselineBundleWarnings,
    workspaceRootPath,
    selectedEditorPath,
    selectedEditorContent,
    selectedEditorLoadedPath,
    editorStatus,
    loadBaselineBundle,
    loadSelectedEditorContent,
    setSelectedEditorContent: updateSelectedEditorContent,
    handleEditorSelectionChange,
    handleSaveSelectedEditorArtifact,
  };
}
