import { useMemo, useReducer, useState } from "react";

import type { Diagnostic, GuidedRepo, ValidateResponse } from "../lib/appContracts";
import { guidedReposReducer, initialGuidedRepos } from "../lib/workspaceSetupState";
import { loadWorkspaceManifest, saveWorkspaceManifest, validateWorkspaceAPI } from "../lib/workspaceApi";

type UseManifestEditorOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useManifestEditor({ setBusy, setError }: UseManifestEditorOptions) {
  const [validateResult, setValidateResult] = useState<ValidateResponse | null>(null);
  const [manifestContent, setManifestContent] = useState("");
  const [guidedRepos, dispatchGuidedRepos] = useReducer(guidedReposReducer, undefined, initialGuidedRepos);
  const [guidedDocsImportsPath, setGuidedDocsImportsPath] = useState("./docs/imports");

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

  async function loadManifest() {
    try {
      setManifestContent(await loadWorkspaceManifest());
    } catch {
      setManifestContent("");
    }
  }

  async function handleValidateWorkspace() {
    setBusy(true);
    setError(null);
    try {
      setValidateResult(await validateWorkspaceAPI());
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "workspace validation failed");
    } finally {
      setBusy(false);
    }
  }

  function updateGuidedRepo(id: string, patch: Partial<GuidedRepo>) {
    dispatchGuidedRepos({ type: "update", id, patch });
  }

  function handleAddGuidedRepo() {
    dispatchGuidedRepos({ type: "add" });
  }

  function handleRemoveGuidedRepo(id: string) {
    dispatchGuidedRepos({ type: "remove", id });
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

      lines.push(`  - name: ${name}`);
      if (repo.mode === "path") {
        lines.push(`    path: ${pathValue}`);
      } else {
        lines.push(`    git_url: ${gitURLValue}`);
      }
      if (refValue) {
        lines.push(`    ref: ${refValue}`);
      }
    }

    lines.push("docs:");
    lines.push(`  imports_path: ${importsPath}`);
    return `${lines.join("\n")}\n`;
  }

  function handleApplyGuidedWorkspaceSetup() {
    setError(null);
    try {
      setManifestContent(buildManifestFromGuidedForm());
    } catch (buildError) {
      setError(buildError instanceof Error ? buildError.message : "failed to apply guided setup");
    }
  }

  async function handleSaveManifest() {
    setBusy(true);
    setError(null);
    try {
      await saveWorkspaceManifest(manifestContent);
      await handleValidateWorkspace();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save manifest");
    } finally {
      setBusy(false);
    }
  }

  return {
    validateResult,
    validationDiagnosticsByRepo,
    manifestContent,
    guidedRepos,
    guidedDocsImportsPath,
    loadManifest,
    setManifestContent,
    setGuidedDocsImportsPath,
    updateGuidedRepo,
    handleAddGuidedRepo,
    handleRemoveGuidedRepo,
    handleApplyGuidedWorkspaceSetup,
    handleSaveManifest,
    handleValidateWorkspace,
  };
}
