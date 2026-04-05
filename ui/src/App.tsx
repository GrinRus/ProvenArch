import { useEffect, useMemo, useState } from "react";

type Diagnostic = {
  level: "error" | "warning";
  code: string;
  message: string;
  suggestion?: string;
  path?: string;
  repo?: string;
};

type ValidateResponse = {
  ok: boolean;
  workspace: string;
  warnings?: Diagnostic[];
  errors?: Diagnostic[];
  resolved_repos?: Array<{ name: string; source: string; path: string; ref?: string }>;
};

type RunStartResponse = {
  run_id: string;
  status: string;
};

type RunStatusResponse = {
  run_id: string;
  pipeline: string;
  status: "queued" | "running" | "succeeded" | "failed";
  started_at: string;
  finished_at?: string | null;
  current_step?: string;
  warnings?: string[];
  error?: string | null;
};

type Artifact = {
  path: string;
  kind: string;
  label: string;
};

type ArtifactsResponse = {
  run_id: string;
  artifacts: Artifact[];
};

type APIErrorPayload = {
  error?:
    | string
    | {
        code?: string;
        message?: string;
      };
};

type RepoSourceMode = "path" | "git_url";

type WizardContract = {
  version: number;
  project_name: string;
  scope: string;
  nfr_priorities: string[];
  rules: string[];
};

type EditableArtifactOption = {
  path: string;
  label: string;
};

const baselineEditorArtifacts: EditableArtifactOption[] = [
  { path: "charter/overview.md", label: "charter/overview.md" },
  { path: "charter/rules.yaml", label: "charter/rules.yaml" },
  { path: "charter/nfr.yaml", label: "charter/nfr.yaml" },
  { path: "charter/glossary.yaml", label: "charter/glossary.yaml" },
  { path: "skills/subagents.yaml", label: "skills/subagents.yaml" },
  { path: "skills/prompt-packs/constitution.md", label: "skills/prompt-packs/constitution.md" },
  { path: "skills/prompt-packs/collect-context.md", label: "skills/prompt-packs/collect-context.md" },
  { path: "skills/prompt-packs/findings.md", label: "skills/prompt-packs/findings.md" },
  { path: "skills/prompt-packs/proposals.md", label: "skills/prompt-packs/proposals.md" },
  { path: "skills/prompt-packs/qa.md", label: "skills/prompt-packs/qa.md" },
  { path: "skills/service-inventory/prompts/system.md", label: "skills/service-inventory/prompts/system.md" },
  { path: "skills/service-inventory/prompts/task.md", label: "skills/service-inventory/prompts/task.md" },
  { path: "skills/interface-extraction/prompts/system.md", label: "skills/interface-extraction/prompts/system.md" },
  { path: "skills/interface-extraction/prompts/task.md", label: "skills/interface-extraction/prompts/task.md" },
  { path: "skills/integration-mapping/prompts/system.md", label: "skills/integration-mapping/prompts/system.md" },
  { path: "skills/integration-mapping/prompts/task.md", label: "skills/integration-mapping/prompts/task.md" },
  { path: "skills/datastore-mapping/prompts/system.md", label: "skills/datastore-mapping/prompts/system.md" },
  { path: "skills/datastore-mapping/prompts/task.md", label: "skills/datastore-mapping/prompts/task.md" },
  { path: "skills/cicd-mapping/prompts/system.md", label: "skills/cicd-mapping/prompts/system.md" },
  { path: "skills/cicd-mapping/prompts/task.md", label: "skills/cicd-mapping/prompts/task.md" },
  { path: "skills/ownership-coverage/prompts/system.md", label: "skills/ownership-coverage/prompts/system.md" },
  { path: "skills/ownership-coverage/prompts/task.md", label: "skills/ownership-coverage/prompts/task.md" },
  { path: "skills/findings/prompts/system.md", label: "skills/findings/prompts/system.md" },
  { path: "skills/findings/prompts/task.md", label: "skills/findings/prompts/task.md" },
  { path: "skills/proposals/prompts/system.md", label: "skills/proposals/prompts/system.md" },
  { path: "skills/proposals/prompts/task.md", label: "skills/proposals/prompts/task.md" },
  { path: "skills/qa/prompts/system.md", label: "skills/qa/prompts/system.md" },
  { path: "skills/qa/prompts/task.md", label: "skills/qa/prompts/task.md" }
];

const finalStatuses = new Set(["succeeded", "failed"]);

function getErrorMessage(payload: unknown, fallback: string): string {
  const typed = payload as APIErrorPayload;
  if (!typed || typeof typed !== "object") {
    return fallback;
  }
  if (typeof typed.error === "string") {
    return typed.error;
  }
  if (typed.error && typeof typed.error.message === "string") {
    return typed.error.message;
  }
  return fallback;
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  const payload = (await response.json()) as T;
  if (!response.ok) {
    throw new Error(getErrorMessage(payload, `request failed: ${url}`));
  }
  return payload;
}

function splitListInput(input: string): string[] {
  return input
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

export default function App() {
  const [validateResult, setValidateResult] = useState<ValidateResponse | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [manifestContent, setManifestContent] = useState("");
  const [selectedEditorPath, setSelectedEditorPath] = useState<string>(baselineEditorArtifacts[0].path);
  const [selectedEditorContent, setSelectedEditorContent] = useState("");
  const [editorStatus, setEditorStatus] = useState("");

  const [guidedRepoName, setGuidedRepoName] = useState("payments-service");
  const [guidedRepoMode, setGuidedRepoMode] = useState<RepoSourceMode>("path");
  const [guidedRepoPath, setGuidedRepoPath] = useState("/absolute/path/to/payments-service");
  const [guidedRepoGitURL, setGuidedRepoGitURL] = useState("https://gitlab.example.com/platform/payments-service.git");
  const [guidedRepoRef, setGuidedRepoRef] = useState("main");
  const [guidedDocsImportsPath, setGuidedDocsImportsPath] = useState("./docs/imports");

  const [wizardProjectName, setWizardProjectName] = useState("ProvenArch MVP");
  const [wizardScope, setWizardScope] = useState("payments, users, ci-cd");
  const [wizardNfr, setWizardNfr] = useState("availability, traceability");
  const [wizardRules, setWizardRules] = useState("no silent re-key, evidence-first findings");
  const [wizardStatus, setWizardStatus] = useState("");

  const [runId, setRunID] = useState<string | null>(null);
  const [runStatus, setRunStatus] = useState<RunStatusResponse | null>(null);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [selectedArtifact, setSelectedArtifact] = useState<string>("");
  const [selectedArtifactContent, setSelectedArtifactContent] = useState<string>("");

  const [coverageSummary, setCoverageSummary] = useState<string>("");
  const [openQuestions, setOpenQuestions] = useState<string>("");

  const [gitMessage, setGitMessage] = useState("chore: update ACP workspace artifacts");
  const [proposalBranch, setProposalBranch] = useState("proposal/beta-refresh");
  const [gitStatus, setGitStatus] = useState<string>("");

  const canPollRun = useMemo(() => {
    if (!runStatus) {
      return Boolean(runId);
    }
    return !finalStatuses.has(runStatus.status);
  }, [runId, runStatus]);

  useEffect(() => {
    void bootstrapEditorData();
  }, []);

  useEffect(() => {
    if (!runId || !canPollRun) {
      return;
    }

    const interval = setInterval(() => {
      void fetchRunStatus(runId);
    }, 1000);

    return () => clearInterval(interval);
  }, [runId, canPollRun]);

  async function bootstrapEditorData() {
    try {
      const manifest = await fetchJSON<{ content: string }>("/api/workspace/manifest");
      setManifestContent(manifest.content ?? "");
    } catch {
      setManifestContent("");
    }

    await loadTextArtifact(selectedEditorPath, setSelectedEditorContent);
    await loadWizardContract();
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
      const payload = (await response.json()) as ValidateResponse & APIErrorPayload;
      setValidateResult(payload);
      if (!response.ok) {
        throw new Error(getErrorMessage(payload, "workspace validation failed"));
      }
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "workspace validation failed");
    } finally {
      setBusy(false);
    }
  }

  function buildManifestFromGuidedForm(): string {
    const name = guidedRepoName.trim();
    const pathValue = guidedRepoPath.trim();
    const gitURLValue = guidedRepoGitURL.trim();
    const refValue = guidedRepoRef.trim();
    const importsPath = guidedDocsImportsPath.trim() || "./docs/imports";

    if (!name) {
      throw new Error("repo name is required");
    }
    if (guidedRepoMode === "path" && !pathValue) {
      throw new Error("path source requires absolute or workspace-relative path");
    }
    if (guidedRepoMode === "git_url" && !gitURLValue) {
      throw new Error("git_url source requires repository URL");
    }

    const sourceLine = guidedRepoMode === "path" ? `    path: ${pathValue}` : `    git_url: ${gitURLValue}`;
    const refLine = refValue ? `\n    ref: ${refValue}` : "";

    return `version: 1\nrepos:\n  - name: ${name}\n${sourceLine}${refLine}\ndocs:\n  imports_path: ${importsPath}\n`;
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

  async function handleRunPipeline(pipeline: "init" | "refresh") {
    setBusy(true);
    setError(null);
    setArtifacts([]);
    setSelectedArtifact("");
    setSelectedArtifactContent("");
    try {
      const payload = await fetchJSON<RunStartResponse>(`/api/pipeline/${pipeline}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ trigger: "ui", commit: false, create_proposal_branch: false })
      });
      setRunID(payload.run_id);
      await fetchRunStatus(payload.run_id);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to start pipeline");
    } finally {
      setBusy(false);
    }
  }

  async function fetchRunStatus(id: string) {
    const payload = await fetchJSON<RunStatusResponse>(`/api/pipeline/runs/${id}`);
    setRunStatus(payload);
    if (finalStatuses.has(payload.status)) {
      await fetchArtifacts(id);
      await loadCoverageArtifacts();
    }
  }

  async function fetchArtifacts(id: string) {
    const payload = await fetchJSON<ArtifactsResponse>(`/api/pipeline/runs/${id}/artifacts`);
    setArtifacts(payload.artifacts ?? []);
  }

  async function loadCoverageArtifacts() {
    await loadTextArtifact("reports/coverage/summary.md", setCoverageSummary);
    await loadTextArtifact("reports/coverage/open-questions.md", setOpenQuestions);
  }

  async function handleOpenArtifact(path: string) {
    setSelectedArtifact(path);
    setSelectedArtifactContent("Loading...");
    await loadTextArtifact(path, setSelectedArtifactContent);
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
          Validate workspace, edit charter/skills baseline bundle, run init/refresh pipeline, inspect artifacts and
          coverage, then commit workspace updates.
        </p>
      </section>

      <section className="panel">
        <h2>Workspace Setup</h2>
        <p className="hint">Guided repo source form writes a valid `workspace.yaml` draft.</p>
        <label htmlFor="guidedRepoName">Repo name</label>
        <input id="guidedRepoName" value={guidedRepoName} onChange={(event) => setGuidedRepoName(event.target.value)} />

        <label htmlFor="guidedRepoMode">Repo source type</label>
        <select id="guidedRepoMode" value={guidedRepoMode} onChange={(event) => setGuidedRepoMode(event.target.value as RepoSourceMode)}>
          <option value="path">path</option>
          <option value="git_url">git_url</option>
        </select>

        {guidedRepoMode === "path" ? (
          <>
            <label htmlFor="guidedRepoPath">path</label>
            <input id="guidedRepoPath" value={guidedRepoPath} onChange={(event) => setGuidedRepoPath(event.target.value)} />
          </>
        ) : (
          <>
            <label htmlFor="guidedRepoGitURL">git_url</label>
            <input id="guidedRepoGitURL" value={guidedRepoGitURL} onChange={(event) => setGuidedRepoGitURL(event.target.value)} />
          </>
        )}

        <label htmlFor="guidedRepoRef">ref (optional)</label>
        <input id="guidedRepoRef" value={guidedRepoRef} onChange={(event) => setGuidedRepoRef(event.target.value)} />

        <label htmlFor="guidedDocsImportsPath">docs.imports_path</label>
        <input id="guidedDocsImportsPath" value={guidedDocsImportsPath} onChange={(event) => setGuidedDocsImportsPath(event.target.value)} />

        <div className="actions">
          <button type="button" onClick={handleApplyGuidedWorkspaceSetup} disabled={busy}>
            Apply guided workspace form
          </button>
        </div>

        <p className="hint">`workspace.yaml` editor (path/git_url sources)</p>
        <textarea value={manifestContent} onChange={(event) => setManifestContent(event.target.value)} rows={12} />
        <div className="actions">
          <button type="button" onClick={() => void handleSaveManifest()} disabled={busy}>
            Save workspace.yaml
          </button>
          <button type="button" onClick={() => void handleValidateWorkspace()} disabled={busy}>
            Validate workspace
          </button>
        </div>
        {validateResult ? (
          <div className="status-block">
            <p>
              Workspace: <code>{validateResult.workspace}</code>
            </p>
            <p>Status: {validateResult.ok ? "valid" : "invalid"}</p>
            {(validateResult.warnings ?? []).map((warning) => (
              <p className="status warn" key={`warn-${warning.code}`}>
                Warning [{warning.code}]: {warning.message}
              </p>
            ))}
            {(validateResult.errors ?? []).map((diagnostic) => (
              <p className="status err" key={`err-${diagnostic.code}`}>
                Error [{diagnostic.code}]: {diagnostic.message}
              </p>
            ))}
          </div>
        ) : null}
      </section>

      <section className="panel">
        <h2>Step 0 Wizard Contract</h2>
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

      <section className="panel">
        <h2>Baseline Editors</h2>
        <p className="hint">Editable baseline files from `charter/*` and `skills/*`.</p>
        <label htmlFor="baselineArtifactSelect">Select artifact</label>
        <select
          id="baselineArtifactSelect"
          value={selectedEditorPath}
          onChange={(event) => {
            void handleEditorSelectionChange(event.target.value);
          }}
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
        />
        <button type="button" onClick={() => void handleSaveSelectedEditorArtifact()} disabled={busy}>
          Save selected baseline artifact
        </button>
        {editorStatus ? <p className="status ok">{editorStatus}</p> : null}
      </section>

      <section className="panel">
        <h2>Run Pipeline</h2>
        <div className="actions">
          <button type="button" onClick={() => void handleRunPipeline("init")} disabled={busy}>
            Run init
          </button>
          <button type="button" onClick={() => void handleRunPipeline("refresh")} disabled={busy}>
            Run refresh
          </button>
        </div>

        {runStatus ? (
          <div className="status-block">
            <p>
              Run <code>{runStatus.run_id}</code> status: <strong>{runStatus.status}</strong>
            </p>
            <p>Pipeline: {runStatus.pipeline}</p>
            {runStatus.current_step ? <p>Current step: {runStatus.current_step}</p> : null}
            {runStatus.error ? <p className="status err">Error: {runStatus.error}</p> : null}
          </div>
        ) : null}
      </section>

      <section className="panel">
        <h2>Coverage & Questions</h2>
        <div className="columns">
          <div>
            <h3>Coverage Summary</h3>
            <pre>{coverageSummary || "No coverage summary yet."}</pre>
          </div>
          <div>
            <h3>Open Questions</h3>
            <pre>{openQuestions || "No open questions yet."}</pre>
          </div>
        </div>
      </section>

      <section className="panel">
        <h2>Run Artifacts</h2>
        {artifacts.length === 0 ? (
          <p>No artifacts yet.</p>
        ) : (
          <div className="columns">
            <ul>
              {artifacts.map((artifact) => (
                <li key={`${artifact.kind}-${artifact.path}`}>
                  <button type="button" className="link-button" onClick={() => void handleOpenArtifact(artifact.path)}>
                    {artifact.path}
                  </button>{" "}
                  ({artifact.kind})
                </li>
              ))}
            </ul>
            <div>
              <h3>{selectedArtifact || "Artifact Content"}</h3>
              <pre>{selectedArtifactContent || "Select artifact to inspect."}</pre>
            </div>
          </div>
        )}
      </section>

      <section className="panel">
        <h2>Git Helper Actions</h2>
        <label htmlFor="gitMessage">Commit message</label>
        <input id="gitMessage" value={gitMessage} onChange={(event) => setGitMessage(event.target.value)} />
        <button type="button" onClick={() => void handleGitCommit()} disabled={busy}>
          Commit workspace changes
        </button>

        <label htmlFor="proposalBranch">Proposal branch</label>
        <input id="proposalBranch" value={proposalBranch} onChange={(event) => setProposalBranch(event.target.value)} />
        <button type="button" onClick={() => void handleCreateProposalBranch()} disabled={busy}>
          Create/Switch proposal branch
        </button>

        {gitStatus ? <p className="status ok">{gitStatus}</p> : null}
      </section>

      {error ? <p className="status err">Error: {error}</p> : null}
    </main>
  );
}
