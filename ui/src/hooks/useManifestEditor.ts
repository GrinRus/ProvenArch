import { useMemo, useReducer, useRef, useState } from "react";

import type { Diagnostic, GuidedRepo, ValidateResponse } from "../lib/appContracts";
import { splitAnalysisScopeLines } from "../lib/analysisScope";
import { guidedReposReducer, initialGuidedRepos, parseGuidedSetupFromManifest } from "../lib/workspaceSetupState";
import { loadWorkspaceManifest, saveWorkspaceManifest, validateWorkspaceAPI } from "../lib/workspaceApi";

type UseManifestEditorOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useManifestEditor({ setBusy, setError }: UseManifestEditorOptions) {
  const [validateResult, setValidateResult] = useState<ValidateResponse | null>(null);
  const [manifestContent, setManifestContent] = useState("");
  const [manifestStatus, setManifestStatus] = useState("");
  const [guidedRepos, dispatchGuidedRepos] = useReducer(guidedReposReducer, undefined, initialGuidedRepos);
  const [guidedDocsImportsPath, setGuidedDocsImportsPath] = useState("./docs/imports");
  const setupDirtyRef = useRef(false);
  const [hasUnsavedManifestDraft, setHasUnsavedManifestDraft] = useState(false);
  const manifestContentRef = useRef("");
  const formRevisionRef = useRef(0);

  const validationDiagnosticsByRepo = useMemo(() => {
    if (!validateResult) {
      return [];
    }
    const grouped = new Map<string, Diagnostic[]>();
    const diagnostics = [...(validateResult.warnings ?? []), ...(validateResult.errors ?? [])];
    for (const diagnostic of diagnostics) {
      const key = diagnostic.repo?.trim() ? diagnostic.repo : "__workspace__";
      const existing = grouped.get(key) ?? [];
      existing.push(diagnostic);
      grouped.set(key, existing);
    }
    return Array.from(grouped.entries()).sort((left, right) => left[0].localeCompare(right[0]));
  }, [validateResult]);

  function markSetupDirty() {
    setupDirtyRef.current = true;
    setHasUnsavedManifestDraft(true);
    formRevisionRef.current += 1;
    setValidateResult(null);
    setManifestStatus("");
  }

  function isCurrentManifestSave(revision: number, content: string): boolean {
    return formRevisionRef.current === revision && manifestContentRef.current === content;
  }

  async function loadManifest() {
    try {
      const content = await loadWorkspaceManifest();
      if (setupDirtyRef.current) {
        return;
      }
      manifestContentRef.current = content;
      setManifestContent(content);
      const guidedSetup = parseGuidedSetupFromManifest(content);
      if (guidedSetup?.repos.length) {
        dispatchGuidedRepos({ type: "replace", repos: guidedSetup.repos });
      }
      if (guidedSetup?.docsImportsPath) {
        setGuidedDocsImportsPath(guidedSetup.docsImportsPath);
      }
    } catch (requestError) {
      manifestContentRef.current = "";
      setManifestContent("");
      throw requestError;
    }
  }

  async function handleValidateWorkspace(): Promise<ValidateResponse | null> {
    setBusy(true);
    setError(null);
    try {
      const result = await validateWorkspaceAPI();
      setValidateResult(result);
      return result;
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "workspace validation failed");
      return null;
    } finally {
      setBusy(false);
    }
  }

  function updateGuidedRepo(id: string, patch: Partial<GuidedRepo>) {
    markSetupDirty();
    dispatchGuidedRepos({ type: "update", id, patch });
  }

  function handleAddGuidedRepo() {
    markSetupDirty();
    dispatchGuidedRepos({ type: "add" });
  }

  function handleRemoveGuidedRepo(id: string) {
    markSetupDirty();
    dispatchGuidedRepos({ type: "remove", id });
  }

  function updateGuidedDocsImportsPath(value: string) {
    markSetupDirty();
    setGuidedDocsImportsPath(value);
  }

  function updateManifestContent(value: string) {
    markSetupDirty();
    manifestContentRef.current = value;
    setManifestContent(value);
  }

  function buildManifestFromGuidedForm(): string {
    const importsPath = guidedDocsImportsPath.trim() || "./docs/imports";
    const names = new Set<string>();
    const lines = ["version: 1", "repos:"];

    if (guidedRepos.length === 0) {
      throw new Error("at least one repo entry is required");
    }

    for (const repo of guidedRepos) {
      const name = repo.name.trim();
      const pathValue = repo.path.trim();
      const gitURLValue = repo.git_url.trim();
      const refValue = repo.ref.trim();
      const includeGlobs = splitAnalysisScopeLines(repo.analysis_include);
      const excludeGlobs = splitAnalysisScopeLines(repo.analysis_exclude);

      if (!name) {
        throw new Error("repo name is required for every entry");
      }
      if (names.has(name)) {
        throw new Error(`duplicate repo name "${name}" in guided setup`);
      }
      names.add(name);

      if (repo.mode === "path" && !pathValue) {
        throw new Error(`repo "${name}" with path source requires non-empty path`);
      }
      if (repo.mode === "git_url" && !gitURLValue) {
        throw new Error(`repo "${name}" with git_url source requires repository URL`);
      }

      lines.push(`  - name: ${yamlScalar(name)}`);
      if (repo.mode === "path") {
        lines.push(`    path: ${yamlScalar(pathValue)}`);
      } else {
        lines.push(`    git_url: ${yamlScalar(gitURLValue)}`);
      }
      if (refValue) {
        lines.push(`    ref: ${yamlScalar(refValue)}`);
      }
      if (includeGlobs.length > 0 || excludeGlobs.length > 0) {
        lines.push("    analysis:");
        if (includeGlobs.length > 0) {
          lines.push("      include:");
          for (const pattern of includeGlobs) {
            lines.push(`        - ${yamlScalar(pattern)}`);
          }
        }
        if (excludeGlobs.length > 0) {
          lines.push("      exclude:");
          for (const pattern of excludeGlobs) {
            lines.push(`        - ${yamlScalar(pattern)}`);
          }
        }
      }
    }

    lines.push("docs:");
    lines.push(`  imports_path: ${yamlScalar(importsPath)}`);
    return `${lines.join("\n")}\n`;
  }

  function handleApplyGuidedWorkspaceSetup() {
    setError(null);
    try {
      markSetupDirty();
      const nextManifest = buildManifestFromGuidedForm();
      manifestContentRef.current = nextManifest;
      setManifestContent(nextManifest);
      setValidateResult(null);
    } catch (buildError) {
      setError(buildError instanceof Error ? buildError.message : "failed to apply guided setup");
    }
  }

  async function handleSaveGuidedWorkspaceSetup() {
    setBusy(true);
    setError(null);
    try {
      const nextManifest = buildManifestFromGuidedForm();
      markSetupDirty();
      const saveRevision = formRevisionRef.current;
      manifestContentRef.current = nextManifest;
      setManifestContent(nextManifest);
      await saveWorkspaceManifest(nextManifest);
      const validation = await validateWorkspaceAPI();
      if (isCurrentManifestSave(saveRevision, nextManifest)) {
        setValidateResult(validation);
        setupDirtyRef.current = false;
        setHasUnsavedManifestDraft(false);
        setManifestStatus("Saved workspace.yaml");
      } else {
        setupDirtyRef.current = true;
        setManifestStatus("Saved workspace.yaml; newer unsaved edits remain.");
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save guided workspace setup");
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveManifest() {
    setBusy(true);
    setError(null);
    const saveRevision = formRevisionRef.current;
    const saveContent = manifestContentRef.current;
    try {
      await saveWorkspaceManifest(saveContent);
      const validation = await validateWorkspaceAPI();
      if (isCurrentManifestSave(saveRevision, saveContent)) {
        setValidateResult(validation);
        setupDirtyRef.current = false;
        setHasUnsavedManifestDraft(false);
        setManifestStatus("Saved workspace.yaml");
      } else {
        setupDirtyRef.current = true;
        setManifestStatus("Saved workspace.yaml; newer unsaved edits remain.");
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save manifest");
    } finally {
      setBusy(false);
    }
  }

  return {
    hasUnsavedManifestDraft,
    validateResult,
    validationDiagnosticsByRepo,
    manifestContent,
    manifestStatus,
    guidedRepos,
    guidedDocsImportsPath,
    loadManifest,
    setManifestContent: updateManifestContent,
    setGuidedDocsImportsPath: updateGuidedDocsImportsPath,
    updateGuidedRepo,
    handleAddGuidedRepo,
    handleRemoveGuidedRepo,
    handleApplyGuidedWorkspaceSetup,
    handleSaveGuidedWorkspaceSetup,
    handleSaveManifest,
    handleValidateWorkspace,
  };
}

function yamlScalar(value: string): string {
  return JSON.stringify(value);
}
