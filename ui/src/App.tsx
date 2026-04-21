import { Suspense, lazy, useEffect, useMemo, useState } from "react";

import { BaselineGitPanel } from "./components/BaselineGitPanel";
import { RunStatusPanel } from "./components/RunStatusPanel";
import { RuntimeProfileSettingsPanel } from "./components/RuntimeProfileSettingsPanel";
import { TabNav, type TabOption } from "./components/TabNav";
import { fetchJSON, getErrorMessage } from "./lib/api";
import {
  makeGuidedRepo,
  runtimeExecutionLabels,
  runtimeStepProviderLabels,
  runtimeStepProviderOrder,
  runtimeTimeoutKeys,
  runtimeTimeoutLabels,
  type BaselineBundleResponse,
  type Diagnostic,
  type EditableArtifactOption,
  type GuidedRepo,
  type RepoSourceMode,
  type RuntimeExecutionKey,
  type RuntimeTimeoutKey,
  type ValidateResponse,
  type WizardContract,
} from "./lib/appContracts";
import {
  formatTimestamp,
  splitListInput,
} from "./lib/runState";
import { useRunExplorer } from "./hooks/useRunExplorer";
import { useRuntimeSettings } from "./hooks/useRuntimeSettings";

const MermaidPreview = lazy(async () => {
  const module = await import("./components/MermaidPreview");
  return { default: module.MermaidPreview };
});

type TopTab = "setup" | "baseline" | "runs" | "results" | "settings";
type ResultsTab = "coverage" | "artifacts" | "diagrams";
type RunLogsMode = "events" | "raw" | "all";

const topTabOptions: Array<TabOption<TopTab>> = [
  { id: "setup", label: "Setup", testId: "tab-setup" },
  { id: "baseline", label: "Baseline", testId: "tab-baseline" },
  { id: "runs", label: "Runs", testId: "tab-runs" },
  { id: "results", label: "Results", testId: "tab-results" },
  { id: "settings", label: "Settings", testId: "tab-settings" },
];

const resultsTabOptions: Array<TabOption<ResultsTab>> = [
  { id: "coverage", label: "Coverage", testId: "results-tab-coverage" },
  { id: "artifacts", label: "Artifacts", testId: "results-tab-artifacts" },
  { id: "diagrams", label: "Diagrams", testId: "results-tab-diagrams" },
];

export default function App() {
  const [activeTab, setActiveTab] = useState<TopTab>("setup");
  const [resultsTab, setResultsTab] = useState<ResultsTab>("coverage");
  const [validateResult, setValidateResult] = useState<ValidateResponse | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [manifestContent, setManifestContent] = useState("");
  const [baselineEditorArtifacts, setBaselineEditorArtifacts] = useState<EditableArtifactOption[]>([]);
  const [baselineBundleWarnings, setBaselineBundleWarnings] = useState<Diagnostic[]>([]);
  const [selectedEditorPath, setSelectedEditorPath] = useState<string>("");
  const [selectedEditorContent, setSelectedEditorContent] = useState("");
  const [editorStatus, setEditorStatus] = useState("");

  const [guidedRepos, setGuidedRepos] = useState<GuidedRepo[]>(() => [
    makeGuidedRepo({
      name: "payments-service",
      mode: "path",
      path: "/absolute/path/to/payments-service"
    })
  ]);
  const [guidedDocsImportsPath, setGuidedDocsImportsPath] = useState("./docs/imports");

  const [wizardProjectName, setWizardProjectName] = useState("ProvenArch MVP");
  const [wizardScope, setWizardScope] = useState("payments, users, ci-cd");
  const [wizardNfr, setWizardNfr] = useState("availability, traceability");
  const [wizardRules, setWizardRules] = useState("no silent re-key, evidence-first findings");
  const [wizardStatus, setWizardStatus] = useState("");

  const [gitMessage, setGitMessage] = useState("chore: update ACP workspace artifacts");
  const [proposalBranch, setProposalBranch] = useState("proposal/beta-refresh");
  const [gitStatus, setGitStatus] = useState<string>("");

  const runtimeSettings = useRuntimeSettings({
    setBusy,
    setError,
  });
  const runExplorer = useRunExplorer({
    setBusy,
    setError,
  });

  const {
    runtimeTimeoutPersisted,
    runtimeTimeoutEffective,
    runtimeTimeoutSource,
    runtimeTimeoutDraft,
    runtimeTimeoutStatus,
    runtimeExecutionPersisted,
    runtimeExecutionEffective,
    runtimeExecutionSource,
    runtimeExecutionDraft,
    runtimeExecutionStatus,
    runtimeStepProviderPersisted,
    runtimeStepProviderEffective,
    runtimeStepProviderSource,
    loadRuntimeTimeouts,
    loadRuntimeExecution,
    loadRuntimeProfile,
    updateRuntimeTimeoutDraft,
    updateRuntimeExecutionDraft,
    handleSaveRuntimeTimeouts,
    handleResetRuntimeTimeouts,
    handleSaveRuntimeExecution,
    handleResetRuntimeExecution,
  } = runtimeSettings;

  const {
    runId,
    runStatus,
    runList,
    artifacts,
    selectedArtifact,
    selectedArtifactContent,
    runLogsStatus,
    runLogsViewMode,
    setRunLogsViewMode,
    runLogsMode,
    setRunLogsMode,
    runActionStatus,
    cancelBusy,
    coverageSummary,
    openQuestions,
    runCounters,
    runLogTaskrunPaths,
    filteredRunLogs,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedArtifactIsMermaid,
    selectedRunWarnings,
    selectedRunIsActive,
    runLogsRendered,
    bootstrapRuns,
    handleRunPipeline,
    handleSelectRun,
    handleCancelSelectedRun,
    handleOpenArtifact,
    handleCopyRunLogs,
    handleDownloadRunLogs,
  } = runExplorer;

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

  useEffect(() => {
    void bootstrapEditorData();
  }, []);

  async function bootstrapEditorData() {
    await bootstrapRuns();

    try {
      const manifest = await fetchJSON<{ content: string }>("/api/workspace/manifest");
      setManifestContent(manifest.content ?? "");
    } catch {
      setManifestContent("");
    }

    await loadBaselineBundle();
    await loadWizardContract();
    await loadRuntimeTimeouts();
    await loadRuntimeExecution();
    await loadRuntimeProfile();
  }

  async function loadBaselineBundle() {
    try {
      const payload = await fetchJSON<BaselineBundleResponse>("/api/workspace/bundle");
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
      const response = await fetch("/api/artifacts?path=charter/wizard/step0-contract.json");
      if (!response.ok) {
        return;
      }
      const content = (await response.text()).trim();
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
      // no-op: wizard contract is optional during bootstrap
    }
  }

  async function loadTextArtifact(path: string, setter: (value: string) => void) {
    try {
      const response = await fetch(`/api/artifacts?path=${encodeURIComponent(path)}`);
      if (!response.ok) {
        setter("");
        return;
      }
      setter(await response.text());
    } catch {
      setter("");
    }
  }

  async function handleValidateWorkspace() {
    setBusy(true);
    setError(null);
    try {
      const response = await fetch("/api/workspace/validate", {
        method: "POST",
        headers: { "Content-Type": "application/json" }
      });
      const payload = await response.json();
      setValidateResult(payload as ValidateResponse);
      if (!response.ok) {
        throw new Error(getErrorMessage(payload, "workspace validation failed"));
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "workspace validation failed");
    } finally {
      setBusy(false);
    }
  }

  function updateGuidedRepo(id: string, patch: Partial<GuidedRepo>) {
    setGuidedRepos((previous) => previous.map((repo) => (repo.id === id ? { ...repo, ...patch } : repo)));
  }

  function handleAddGuidedRepo() {
    setGuidedRepos((previous) => [...previous, makeGuidedRepo()]);
  }

  function handleRemoveGuidedRepo(id: string) {
    setGuidedRepos((previous) => {
      if (previous.length <= 1) {
        return previous;
      }
      return previous.filter((repo) => repo.id !== id);
    });
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
      await fetchJSON<{ ok: boolean }>("/api/workspace/manifest", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content: manifestContent })
      });
      await handleValidateWorkspace();
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save manifest");
    } finally {
      setBusy(false);
    }
  }

  async function saveEditableArtifact(path: string, content: string) {
    await fetchJSON<{ ok: boolean }>("/api/artifacts/write", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path, content })
    });
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
      rules: splitListInput(wizardRules)
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
      const payload = await fetchJSON<{ status: string; message?: string; output?: string }>("/api/git/commit", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: gitMessage })
      });
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
      const payload = await fetchJSON<{ branch: string }>("/api/git/proposal-branch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: proposalBranch })
      });
      setGitStatus(`checked out ${payload.branch}`);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to create proposal branch");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="shell">
      <section className="hero">
        <p className="eyebrow">ACP Beta Surface</p>
        <h1>Local-first architecture control plane</h1>
        <p className="lead">
          Validate workspace, tune settings, edit baseline prompts, run init/refresh pipelines, inspect logs, diagrams,
          and artifacts, then commit workspace updates.
        </p>
      </section>

      <TabNav value={activeTab} onChange={setActiveTab} options={topTabOptions} testId="top-tabs" />

      {activeTab === "setup" ? (
        <>
          <section className="panel" data-testid="workspace-panel">
            <h2>Setup: Workspace</h2>
            <p className="hint">Guided setup writes a valid multi-repo `workspace.yaml` draft.</p>
            {guidedRepos.map((repo, index) => (
              <div className="repo-card" key={repo.id}>
                <div className="repo-card-head">
                  <h3>Repo {index + 1}</h3>
                  <button type="button" className="inline-danger" onClick={() => handleRemoveGuidedRepo(repo.id)} disabled={busy || guidedRepos.length <= 1}>
                    Remove
                  </button>
                </div>

                <label htmlFor={`guidedRepoName-${repo.id}`}>Repo name</label>
                <input
                  id={`guidedRepoName-${repo.id}`}
                  value={repo.name}
                  onChange={(event) => updateGuidedRepo(repo.id, { name: event.target.value })}
                />

                <label htmlFor={`guidedRepoMode-${repo.id}`}>Repo source type</label>
                <select
                  id={`guidedRepoMode-${repo.id}`}
                  value={repo.mode}
                  onChange={(event) => updateGuidedRepo(repo.id, { mode: event.target.value as RepoSourceMode })}
                >
                  <option value="path">path</option>
                  <option value="git_url">git_url</option>
                </select>

                {repo.mode === "path" ? (
                  <>
                    <label htmlFor={`guidedRepoPath-${repo.id}`}>path</label>
                    <input
                      id={`guidedRepoPath-${repo.id}`}
                      value={repo.path}
                      onChange={(event) => updateGuidedRepo(repo.id, { path: event.target.value })}
                    />
                  </>
                ) : (
                  <>
                    <label htmlFor={`guidedRepoGitURL-${repo.id}`}>git_url</label>
                    <input
                      id={`guidedRepoGitURL-${repo.id}`}
                      value={repo.git_url}
                      onChange={(event) => updateGuidedRepo(repo.id, { git_url: event.target.value })}
                    />
                  </>
                )}

                <label htmlFor={`guidedRepoRef-${repo.id}`}>ref (optional)</label>
                <input
                  id={`guidedRepoRef-${repo.id}`}
                  value={repo.ref}
                  onChange={(event) => updateGuidedRepo(repo.id, { ref: event.target.value })}
                  placeholder="Leave empty to use current checkout"
                />
              </div>
            ))}

            <label htmlFor="guidedDocsImportsPath">docs.imports_path</label>
            <input id="guidedDocsImportsPath" value={guidedDocsImportsPath} onChange={(event) => setGuidedDocsImportsPath(event.target.value)} />

            <div className="actions">
              <button type="button" onClick={handleAddGuidedRepo} disabled={busy}>
                Add repo
              </button>
              <button type="button" onClick={handleApplyGuidedWorkspaceSetup} disabled={busy}>
                Apply guided workspace form
              </button>
            </div>

            <p className="hint">`workspace.yaml` editor (path/git_url sources)</p>
            <textarea value={manifestContent} onChange={(event) => setManifestContent(event.target.value)} rows={12} />
            <div className="actions">
              <button type="button" onClick={() => void handleSaveManifest()} disabled={busy} data-testid="workspace-save-btn">
                Save workspace.yaml
              </button>
              <button
                type="button"
                onClick={() => void handleValidateWorkspace()}
                disabled={busy}
                data-testid="workspace-validate-btn"
              >
                Validate workspace
              </button>
            </div>
            {validateResult ? (
              <div className="status-block" data-testid="workspace-validate-result">
                <p>
                  Workspace: <code>{validateResult.workspace}</code>
                </p>
                <p>Status: {validateResult.ok ? "valid" : "invalid"}</p>

                {(validateResult.resolved_repos ?? []).length > 0 ? (
                  <div className="repo-summary" data-testid="workspace-validate-resolved-repos">
                    <p className="hint">Resolved repos</p>
                    <ul>
                      {(validateResult.resolved_repos ?? []).map((repo) => (
                        <li key={`resolved-${repo.name}-${repo.path}`}>
                          <code>{repo.name}</code> ({repo.source}) {repo.path}
                          {repo.ref ? ` @ ${repo.ref}` : ""}
                        </li>
                      ))}
                    </ul>
                  </div>
                ) : null}

                {validationDiagnosticsByRepo.map(([repoKey, diagnostics]) => (
                  <div key={`diag-group-${repoKey}`} className="repo-summary">
                    <p className="hint">{repoKey === "__workspace__" ? "Workspace diagnostics" : `Diagnostics for ${repoKey}`}</p>
                    {diagnostics.map((diagnostic, index) => (
                      <p className={diagnostic.level === "error" ? "status err" : "status warn"} key={`${repoKey}-${diagnostic.code}-${diagnostic.message}-${index}`}>
                        {diagnostic.level === "error" ? "Error" : "Warning"} [{diagnostic.code}]: {diagnostic.message}
                      </p>
                    ))}
                  </div>
                ))}
              </div>
            ) : null}
          </section>

          <section className="panel">
            <h2>Setup: Step 0 Wizard Contract</h2>
            <p className="hint">Structured contract persisted as `charter/wizard/step0-contract.json`.</p>

            <label htmlFor="wizardProjectName">Project name</label>
            <input id="wizardProjectName" value={wizardProjectName} onChange={(event) => setWizardProjectName(event.target.value)} />

            <label htmlFor="wizardScope">Scope</label>
            <textarea id="wizardScope" value={wizardScope} onChange={(event) => setWizardScope(event.target.value)} rows={3} />

            <label htmlFor="wizardNfr">NFR priorities (comma/newline)</label>
            <textarea id="wizardNfr" value={wizardNfr} onChange={(event) => setWizardNfr(event.target.value)} rows={3} />

            <label htmlFor="wizardRules">Rules (comma/newline)</label>
            <textarea id="wizardRules" value={wizardRules} onChange={(event) => setWizardRules(event.target.value)} rows={3} />

            <button type="button" onClick={() => void handleSaveStep0WizardContract()} disabled={busy}>
              Save Step 0 wizard contract
            </button>

            {wizardStatus ? <p className="status ok">{wizardStatus}</p> : null}
          </section>
        </>
      ) : null}

      {activeTab === "settings" ? (
        <RuntimeProfileSettingsPanel
          busy={busy}
          runtimeTimeoutKeys={[...runtimeTimeoutKeys]}
          runtimeTimeoutLabels={runtimeTimeoutLabels}
          runtimeTimeoutDraft={runtimeTimeoutDraft}
          runtimeTimeoutPersisted={runtimeTimeoutPersisted}
          runtimeTimeoutEffective={runtimeTimeoutEffective}
          runtimeTimeoutSource={runtimeTimeoutSource}
          runtimeTimeoutStatus={runtimeTimeoutStatus}
          onReloadTimeouts={() => void loadRuntimeTimeouts()}
          onSaveTimeouts={() => void handleSaveRuntimeTimeouts()}
          onResetTimeouts={() => void handleResetRuntimeTimeouts()}
          onTimeoutChange={(key, value) => updateRuntimeTimeoutDraft(key as RuntimeTimeoutKey, value)}
          runtimeExecutionLabels={runtimeExecutionLabels}
          runtimeExecutionDraft={runtimeExecutionDraft}
          runtimeExecutionPersisted={runtimeExecutionPersisted}
          runtimeExecutionEffective={runtimeExecutionEffective}
          runtimeExecutionSource={runtimeExecutionSource}
          runtimeExecutionStatus={runtimeExecutionStatus}
          onReloadExecution={() => void loadRuntimeExecution()}
          onSaveExecution={() => void handleSaveRuntimeExecution()}
          onResetExecution={() => void handleResetRuntimeExecution()}
          onExecutionChange={(key, value) => updateRuntimeExecutionDraft(key as RuntimeExecutionKey, value)}
          stepProviderLabels={runtimeStepProviderLabels}
          stepProviderOrder={[...runtimeStepProviderOrder]}
          stepProviderPersisted={runtimeStepProviderPersisted}
          stepProviderEffective={runtimeStepProviderEffective}
          stepProviderSource={runtimeStepProviderSource}
          onReloadProfile={() => void loadRuntimeProfile()}
        />
      ) : null}

      {activeTab === "baseline" ? (
        <>
          <section className="panel">
            <h2>Baseline: Editors</h2>
            <p className="hint">
              Editable baseline files from `charter/*` and `skills/*`. Live headless runtime customization consumes prompt packs for
              `collect`/`findings`; `skills/*/prompts/*.md` stay editable here as reference-only seeded assets.
            </p>
            {baselineBundleWarnings.map((diagnostic, index) => (
              <p
                key={`${diagnostic.code}-${diagnostic.message}-${index}`}
                className={diagnostic.level === "error" ? "status err" : "status warn"}
              >
                {diagnostic.level === "error" ? "Error" : "Warning"} [{diagnostic.code}]: {diagnostic.message}
              </p>
            ))}
            <label htmlFor="baselineArtifactSelect">Select artifact</label>
            <select
              id="baselineArtifactSelect"
              value={selectedEditorPath}
              onChange={(event) => {
                void handleEditorSelectionChange(event.target.value);
              }}
              disabled={baselineEditorArtifacts.length === 0}
            >
              {baselineEditorArtifacts.map((artifact) => (
                <option key={artifact.path} value={artifact.path}>
                  {artifact.label}
                </option>
              ))}
            </select>
            <label htmlFor="baselineArtifactEditor">{selectedEditorPath}</label>
            <textarea
              id="baselineArtifactEditor"
              value={selectedEditorContent}
              onChange={(event) => setSelectedEditorContent(event.target.value)}
              rows={10}
              disabled={!selectedEditorPath}
            />
            <button type="button" onClick={() => void handleSaveSelectedEditorArtifact()} disabled={busy || !selectedEditorPath}>
              Save selected baseline artifact
            </button>
            {editorStatus ? <p className="status ok">{editorStatus}</p> : null}
          </section>

          <BaselineGitPanel
            busy={busy}
            gitMessage={gitMessage}
            proposalBranch={proposalBranch}
            gitStatus={gitStatus}
            onGitMessageChange={setGitMessage}
            onProposalBranchChange={setProposalBranch}
            onCommit={() => void handleGitCommit()}
            onCreateProposalBranch={() => void handleCreateProposalBranch()}
          />
        </>
      ) : null}

      {activeTab === "runs" ? (
        <>
          <section className="panel" data-testid="runs-control-panel">
            <h2>Runs: Pipeline Control</h2>
            <div className="actions">
              <button type="button" onClick={() => void handleRunPipeline("init")} disabled={busy} data-testid="run-init-btn">
                Run init
              </button>
              <button
                type="button"
                onClick={() => void handleRunPipeline("refresh")}
                disabled={busy}
                data-testid="run-refresh-btn"
              >
                Run refresh
              </button>
              <button
                type="button"
                onClick={() => void handleCancelSelectedRun()}
                disabled={busy || cancelBusy || !runId || !selectedRunIsActive}
                data-testid="run-cancel-btn"
              >
                Cancel selected run
              </button>
            </div>
            {runActionStatus ? <p className="status warn">{runActionStatus}</p> : null}

            <RunStatusPanel runStatus={runStatus} warnings={selectedRunWarnings} />
          </section>

          <section className="panel" data-testid="runs-history-panel">
            <h2>Runs: History</h2>
            <p className="hint">
              Running: {runCounters.running} | Succeeded: {runCounters.succeeded} | Failed: {runCounters.failed}
            </p>
            {runList.length === 0 ? (
              <p>No runs yet.</p>
            ) : (
              <div className="run-table-wrap">
                <table className="run-table" data-testid="runs-history-table">
                  <thead>
                    <tr>
                      <th>Run ID</th>
                      <th>Status</th>
                      <th>Pipeline</th>
                      <th>Started</th>
                      <th>Finished</th>
                      <th>Error code</th>
                      <th>Warnings</th>
                    </tr>
                  </thead>
                  <tbody>
                    {runList.map((run) => (
                      <tr
                        key={run.run_id}
                        className={runId === run.run_id ? "selected" : ""}
                        onClick={() => void handleSelectRun(run.run_id)}
                      >
                        <td>
                          <button
                            type="button"
                            className="link-button"
                            onClick={(event) => {
                              event.stopPropagation();
                              void handleSelectRun(run.run_id);
                            }}
                          >
                            {run.run_id}
                          </button>
                        </td>
                        <td>{run.status}</td>
                        <td>{run.pipeline}</td>
                        <td>{formatTimestamp(run.started_at)}</td>
                        <td>{formatTimestamp(run.finished_at)}</td>
                        <td>{run.error_code || "-"}</td>
                        <td>{run.warnings?.length ?? 0}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="panel" data-testid="runs-logs-panel">
            <h2>Runs: Logs</h2>
            <div className="actions">
              <label htmlFor="runLogsMode">Mode</label>
              <select
                id="runLogsMode"
                value={runLogsMode}
                onChange={(event) => setRunLogsMode(event.target.value as RunLogsMode)}
                className="inline-select"
                data-testid="run-logs-mode-select"
              >
                <option value="all">all</option>
                <option value="events">event timeline</option>
                <option value="raw">raw agent stream</option>
              </select>
              <label htmlFor="runLogsViewMode">View</label>
              <select
                id="runLogsViewMode"
                value={runLogsViewMode}
                onChange={(event) => setRunLogsViewMode(event.target.value as "line" | "line+fields")}
                className="inline-select"
                data-testid="run-logs-view-select"
              >
                <option value="line">line</option>
                <option value="line+fields">line+fields</option>
              </select>
              <button
                type="button"
                onClick={() => void handleCopyRunLogs()}
                disabled={filteredRunLogs.length === 0}
                data-testid="run-logs-copy-btn"
              >
                Copy logs
              </button>
              <button
                type="button"
                onClick={() => handleDownloadRunLogs()}
                disabled={filteredRunLogs.length === 0 || !runId}
                data-testid="run-logs-download-btn"
              >
                Download logs
              </button>
            </div>
            {runLogsStatus ? <p className="status ok">{runLogsStatus}</p> : null}
            {runLogTaskrunPaths.length > 0 ? (
              <div className="actions">
                {runLogTaskrunPaths.map((path) => (
                  <button key={`taskrun-log-open-${path}`} type="button" onClick={() => void handleOpenArtifact(path)}>
                    Open runtime execution artifact: {path}
                  </button>
                ))}
              </div>
            ) : null}
            {filteredRunLogs.length === 0 ? (
              <p>No run logs yet.</p>
            ) : (
              <pre data-testid="run-logs-content">{runLogsRendered}</pre>
            )}
          </section>
        </>
      ) : null}

      {activeTab === "results" ? (
        <>
          <TabNav value={resultsTab} onChange={setResultsTab} options={resultsTabOptions} testId="results-tabs" />

          {resultsTab === "coverage" ? (
            <section className="panel" data-testid="results-coverage-panel">
              <h2>Results: Coverage & Questions</h2>
              <div className="columns">
                <div>
                  <h3>Coverage Summary</h3>
                  <pre data-testid="coverage-summary-content">{coverageSummary || "No coverage summary yet."}</pre>
                </div>
                <div>
                  <h3>Open Questions</h3>
                  <pre data-testid="open-questions-content">{openQuestions || "No open questions yet."}</pre>
                </div>
              </div>
            </section>
          ) : null}

          {resultsTab === "artifacts" ? (
            <section className="panel" data-testid="results-artifacts-panel">
              <h2>Results: Run Artifacts</h2>
              {nonDiagramArtifacts.length === 0 ? (
                <p>No non-diagram artifacts yet.</p>
              ) : (
                <div className="columns">
                  <ul data-testid="run-artifacts-list">
                    {nonDiagramArtifacts.map((artifact) => (
                      <li key={`${artifact.kind}-${artifact.path}`}>
                        <button type="button" className="link-button" onClick={() => void handleOpenArtifact(artifact.path)}>
                          {artifact.path}
                        </button>{" "}
                        ({artifact.kind})
                      </li>
                    ))}
                  </ul>
                  <div data-testid="run-artifact-content-panel">
                    <h3 data-testid="run-artifact-selected-path">{selectedArtifact || "Artifact Content"}</h3>
                    <pre data-testid="run-artifact-content">{selectedArtifactContent || "Select artifact to inspect."}</pre>
                  </div>
                </div>
              )}
            </section>
          ) : null}

          {resultsTab === "diagrams" ? (
            <section className="panel" data-testid="results-diagrams-panel">
              <h2>Results: Diagrams</h2>
              {diagramArtifacts.length === 0 ? (
                <p>No diagram artifacts yet.</p>
              ) : (
                <div className="columns">
                  <ul data-testid="run-diagrams-list">
                    {diagramArtifacts.map((artifact) => (
                      <li key={`${artifact.kind}-${artifact.path}`}>
                        <button type="button" className="link-button" onClick={() => void handleOpenArtifact(artifact.path)}>
                          {artifact.path}
                        </button>{" "}
                        ({artifact.kind})
                      </li>
                    ))}
                  </ul>
                  <div data-testid="run-diagram-content-panel">
                    <h3 data-testid="run-diagram-selected-path">{selectedArtifact || "Diagram Preview"}</h3>
                    {selectedArtifactIsMermaid ? (
                      <Suspense fallback={<p className="hint">Loading diagram renderer...</p>}>
                        <MermaidPreview source={selectedArtifactContent} title={selectedArtifact || "diagram"} />
                      </Suspense>
                    ) : (
                      <pre data-testid="run-diagram-content">
                        {selectedArtifactContent || "Select a `.mmd` diagram artifact to preview."}
                      </pre>
                    )}
                  </div>
                </div>
              )}
            </section>
          ) : null}
        </>
      ) : null}

      {error ? <p className="status err">Error: {error}</p> : null}
    </main>
  );
}
