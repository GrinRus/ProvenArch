import { useMemo, useReducer, useState } from "react";

import {
  type Diagnostic,
  type EditableArtifactOption,
  type GuidedRepo,
  type ValidateResponse,
  type WizardContract,
} from "../lib/appContracts";
import { splitListInput } from "../lib/runState";
import { guidedReposReducer, initialGuidedRepos } from "../lib/workspaceSetupState";
import {
  commitWorkspaceArtifacts,
  createProposalBranch,
  loadArtifactText,
  loadBaselineBundleAPI,
  loadWorkspaceManifest,
  saveEditableArtifact,
  saveWorkspaceManifest,
  validateWorkspaceAPI,
} from "../lib/workspaceApi";

type UseWorkspaceSetupOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useWorkspaceSetup({ setBusy, setError }: UseWorkspaceSetupOptions) {
  const [validateResult, setValidateResult] = useState<ValidateResponse | null>(null);
  const [manifestContent, setManifestContent] = useState("");

  const [baselineEditorArtifacts, setBaselineEditorArtifacts] = useState<EditableArtifactOption[]>([]);
  const [baselineBundleWarnings, setBaselineBundleWarnings] = useState<Diagnostic[]>([]);
  const [selectedEditorPath, setSelectedEditorPath] = useState<string>("");
  const [selectedEditorContent, setSelectedEditorContent] = useState("");
  const [editorStatus, setEditorStatus] = useState("");

  const [guidedRepos, dispatchGuidedRepos] = useReducer(guidedReposReducer, undefined, initialGuidedRepos);
  const [guidedDocsImportsPath, setGuidedDocsImportsPath] = useState("./docs/imports");

  const [wizardProjectName, setWizardProjectName] = useState("ProvenArch MVP");
  const [wizardScope, setWizardScope] = useState("payments, users, ci-cd");
  const [wizardNfr, setWizardNfr] = useState("availability, traceability");
  const [wizardRules, setWizardRules] = useState("no silent re-key, evidence-first findings");
  const [wizardStatus, setWizardStatus] = useState("");

  const [gitMessage, setGitMessage] = useState("chore: update ACP workspace artifacts");
  const [proposalBranch, setProposalBranch] = useState("proposal/beta-refresh");
  const [gitStatus, setGitStatus] = useState<string>("");

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

  async function bootstrapWorkspaceSetup() {
    try {
      setManifestContent(await loadWorkspaceManifest());
    } catch {
      setManifestContent("");
    }

    await loadBaselineBundle();
    await loadWizardContract();
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

  async function loadWizardContract() {
    try {
      const content = (await loadArtifactText("charter/wizard/step0-contract.json"))?.trim() ?? "";
      if (!content) {
        return;
      }
      const parsed = JSON.parse(content) as Partial<WizardContract>;
      if (typeof parsed.project_name === "string") {
        setWizardProjectName(parsed.project_name);
      }
      if (typeof parsed.scope === "string") {
        setWizardScope(parsed.scope);
      }
      if (Array.isArray(parsed.nfr_priorities)) {
        setWizardNfr(parsed.nfr_priorities.join(", "));
      }
      if (Array.isArray(parsed.rules)) {
        setWizardRules(parsed.rules.join(", "));
      }
    } catch {
      // Wizard contract remains optional during bootstrap.
    }
  }

  async function loadTextArtifact(path: string, setter: (value: string) => void) {
    try {
      setter((await loadArtifactText(path)) ?? "");
    } catch {
      setter("");
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

  async function handleSaveStep0WizardContract() {
    setBusy(true);
    setError(null);
    setWizardStatus("");

    const projectName = wizardProjectName.trim();
    const scope = wizardScope.trim();
    if (!projectName || !scope) {
      setBusy(false);
      setError("step0 wizard contract requires project name and scope");
      return;
    }

    const payload: WizardContract = {
      version: 1,
      project_name: projectName,
      scope,
      nfr_priorities: splitListInput(wizardNfr),
      rules: splitListInput(wizardRules),
    };

    try {
      await saveEditableArtifact("charter/wizard/step0-contract.json", `${JSON.stringify(payload, null, 2)}\n`);
      setWizardStatus("Saved charter/wizard/step0-contract.json");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save step0 wizard contract");
    } finally {
      setBusy(false);
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

  async function handleGitCommit() {
    setBusy(true);
    setError(null);
    setGitStatus("");
    try {
      const payload = await commitWorkspaceArtifacts(gitMessage);
      setGitStatus(payload.output ?? payload.message ?? payload.status);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "git commit failed");
    } finally {
      setBusy(false);
    }
  }

  async function handleCreateProposalBranch() {
    setBusy(true);
    setError(null);
    setGitStatus("");
    try {
      const payload = await createProposalBranch(proposalBranch);
      setGitStatus(`checked out ${payload.branch}`);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to create proposal branch");
    } finally {
      setBusy(false);
    }
  }

  return {
    validateResult,
    validationDiagnosticsByRepo,
    manifestContent,
    baselineEditorArtifacts,
    baselineBundleWarnings,
    selectedEditorPath,
    selectedEditorContent,
    editorStatus,
    guidedRepos,
    guidedDocsImportsPath,
    wizardProjectName,
    wizardScope,
    wizardNfr,
    wizardRules,
    wizardStatus,
    gitMessage,
    proposalBranch,
    gitStatus,
    bootstrapWorkspaceSetup,
    setManifestContent,
    setGuidedDocsImportsPath,
    setWizardProjectName,
    setWizardScope,
    setWizardNfr,
    setWizardRules,
    setSelectedEditorContent,
    setGitMessage,
    setProposalBranch,
    updateGuidedRepo,
    handleAddGuidedRepo,
    handleRemoveGuidedRepo,
    handleApplyGuidedWorkspaceSetup,
    handleSaveManifest,
    handleValidateWorkspace,
    handleSaveStep0WizardContract,
    handleEditorSelectionChange,
    handleSaveSelectedEditorArtifact,
    handleGitCommit,
    handleCreateProposalBranch,
  };
}
