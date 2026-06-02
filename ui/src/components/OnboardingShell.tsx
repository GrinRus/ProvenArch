import type { Diagnostic, DoctorResponse, GuidedRepo, OnboardingStatusResponse, RepoSourceMode, ValidateResponse } from "../lib/appContracts";

type OnboardingShellProps = {
  busy: boolean;
  error: string | null;
  status: OnboardingStatusResponse | null;
  workspacePath: string;
  createWorkspace: boolean;
  guidedRepos: GuidedRepo[];
  guidedDocsImportsPath: string;
  validateResult: ValidateResponse | null;
  doctorResult: DoctorResponse | null;
  setupRuntime: string;
  setupRuntimeProvider: string;
  firstRunStatus: string;
  onWorkspacePathChange: (value: string) => void;
  onCreateWorkspaceChange: (value: boolean) => void;
  onSelectWorkspace: () => void;
  onRepoChange: (id: string, patch: Partial<GuidedRepo>) => void;
  onAddRepo: () => void;
  onRemoveRepo: (id: string) => void;
  onDocsImportsPathChange: (value: string) => void;
  onSaveSources: () => void;
  onRuntimeChange: (value: string) => void;
  onRuntimeProviderChange: (value: string) => void;
  onSaveRuntime: () => void;
  onCheckDoctor: () => void;
  onEnterConsole: () => void;
  onRunFirstAnalysis: () => void;
};

export function OnboardingShell({
  busy,
  error,
  status,
  workspacePath,
  createWorkspace,
  guidedRepos,
  guidedDocsImportsPath,
  validateResult,
  doctorResult,
  setupRuntime,
  setupRuntimeProvider,
  firstRunStatus,
  onWorkspacePathChange,
  onCreateWorkspaceChange,
  onSelectWorkspace,
  onRepoChange,
  onAddRepo,
  onRemoveRepo,
  onDocsImportsPathChange,
  onSaveSources,
  onRuntimeChange,
  onRuntimeProviderChange,
  onSaveRuntime,
  onCheckDoctor,
  onEnterConsole,
  onRunFirstAnalysis,
}: OnboardingShellProps) {
  const workspaceReady = status?.workspace_selected ?? false;
  const repoDiagnosticsByID = buildRepoDiagnostics(guidedRepos, validateResult);
  const hasRepoDraftErrors = Array.from(repoDiagnosticsByID.values()).some((diagnostics) => diagnostics.some((diagnostic) => diagnostic.level === "error"));
  const sourcesReady = validateResult?.ok === true && !hasRepoDraftErrors;
  const runtimeReady = status?.runtime.selected === true;
  const canEnterConsole = status?.can_enter_console === true && sourcesReady && runtimeReady;

  return (
    <main className="onboarding-shell" data-testid="onboarding-shell">
      <section className="onboarding-panel">
        <div className="onboarding-header">
          <div>
            <p className="eyebrow">ACP local console</p>
            <h1>Set up your architecture workspace</h1>
            <p className="hint">Choose the Git-tracked ACP workspace first, then connect source repositories and select the runner.</p>
          </div>
          <div className="onboarding-status-stack" aria-label="onboarding status">
            <StatusPill label="Workspace" ready={workspaceReady} />
            <StatusPill label="Sources" ready={sourcesReady} />
            <StatusPill label="Runner" ready={runtimeReady} />
          </div>
        </div>

        {error ? <div className="error-banner">{error}</div> : null}

        <div className="onboarding-grid">
          <section className="onboarding-card" data-testid="onboarding-workspace-step">
            <div className="card-heading">
              <span className="step-index">1</span>
              <div>
                <h2>Workspace</h2>
                <p className="hint">This is where ACP writes architecture artifacts. Source repos stay read-only inputs.</p>
              </div>
            </div>
            <div className="field">
              <label htmlFor="onboardingWorkspacePath">Architecture workspace path</label>
              <input
                id="onboardingWorkspacePath"
                value={workspacePath}
                placeholder="/Users/me/acp-workspaces/my-system"
                onChange={(event) => onWorkspacePathChange(event.target.value)}
              />
            </div>
            <label className="checkbox-row">
              <input type="checkbox" checked={createWorkspace} onChange={(event) => onCreateWorkspaceChange(event.target.checked)} />
              <span>Create path if missing</span>
            </label>
            <button type="button" onClick={onSelectWorkspace} disabled={busy || !workspacePath.trim()} data-testid="onboarding-workspace-save">
              {createWorkspace ? "Create or open workspace" : "Open workspace"}
            </button>
            {status?.workspace_selected ? <p className="status">Selected: {status.workspace}</p> : null}
          </section>

          <section className="onboarding-card" data-testid="onboarding-sources-step">
            <div className="card-heading">
              <span className="step-index">2</span>
              <div>
                <h2>Sources</h2>
                <p className="hint">Add one or more repositories from Git URL or local checkout path.</p>
              </div>
            </div>
            {guidedRepos.map((repo, index) => (
              <div className="onboarding-repo-row" key={repo.id}>
                <div className="repo-row-title">
                  <strong>Repo {index + 1}</strong>
                  <button type="button" className="inline-danger" onClick={() => onRemoveRepo(repo.id)} disabled={busy || guidedRepos.length <= 1}>
                    Remove
                  </button>
                </div>
                <div className="repo-field-grid">
                  <div className="field">
                    <label htmlFor={`onboardingRepoName-${repo.id}`}>Name</label>
                    <input id={`onboardingRepoName-${repo.id}`} value={repo.name} onChange={(event) => onRepoChange(repo.id, { name: event.target.value })} />
                  </div>
                  <div className="field">
                    <label htmlFor={`onboardingRepoMode-${repo.id}`}>Source type</label>
                    <select
                      id={`onboardingRepoMode-${repo.id}`}
                      value={repo.mode}
                      onChange={(event) => onRepoChange(repo.id, { mode: event.target.value as RepoSourceMode })}
                    >
                      <option value="git_url">GitHub/GitLab URL</option>
                      <option value="path">Local folder</option>
                    </select>
                  </div>
                  <div className="field is-wide">
                    <label htmlFor={`onboardingRepoSource-${repo.id}`}>{repo.mode === "path" ? "Local checkout path" : "Repository URL"}</label>
                    <input
                      id={`onboardingRepoSource-${repo.id}`}
                      value={repo.mode === "path" ? repo.path : repo.git_url}
                      onChange={(event) => onRepoChange(repo.id, repo.mode === "path" ? { path: event.target.value } : { git_url: event.target.value })}
                    />
                  </div>
                  <div className="field">
                    <label htmlFor={`onboardingRepoRef-${repo.id}`}>ref optional</label>
                    <input id={`onboardingRepoRef-${repo.id}`} value={repo.ref} onChange={(event) => onRepoChange(repo.id, { ref: event.target.value })} />
                  </div>
                </div>
                <RepoDiagnostics diagnostics={repoDiagnosticsByID.get(repo.id) ?? []} />
              </div>
            ))}
            <div className="field">
              <label htmlFor="onboardingDocsImportsPath">docs.imports_path</label>
              <input id="onboardingDocsImportsPath" value={guidedDocsImportsPath} onChange={(event) => onDocsImportsPathChange(event.target.value)} />
            </div>
            <div className="actions">
              <button type="button" onClick={onAddRepo} disabled={busy || !workspaceReady}>
                Add repo
              </button>
              <button type="button" onClick={onSaveSources} disabled={busy || !workspaceReady || hasRepoDraftErrors} data-testid="onboarding-sources-save">
                Save and validate sources
              </button>
            </div>
            {validateResult ? <p className={validateResult.ok ? "status" : "error-text"}>{validateResult.ok ? "Sources validated." : "Sources need fixes."}</p> : null}
          </section>

          <section className="onboarding-card" data-testid="onboarding-runner-step">
            <div className="card-heading">
              <span className="step-index">3</span>
              <div>
                <h2>Runner</h2>
                <p className="hint">Use fake for the first deterministic walkthrough; live providers are explicit opt-in.</p>
              </div>
            </div>
            <div className="field">
              <label htmlFor="onboardingRuntime">Runtime</label>
              <select id="onboardingRuntime" value={setupRuntime} onChange={(event) => onRuntimeChange(event.target.value)}>
                <option value="fake">fake baseline</option>
                <option value="headless">headless provider</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="onboardingRuntimeProvider">Provider</label>
              <select id="onboardingRuntimeProvider" value={setupRuntimeProvider} disabled={setupRuntime !== "headless"} onChange={(event) => onRuntimeProviderChange(event.target.value)}>
                <option value="claude-code">claude-code</option>
                <option value="qwen-code">qwen-code</option>
                <option value="codex-code">codex-code</option>
              </select>
            </div>
            <div className="actions">
              <button type="button" onClick={onSaveRuntime} disabled={busy || !workspaceReady} data-testid="onboarding-runtime-save">
                Select runner
              </button>
              <button type="button" onClick={onCheckDoctor} disabled={busy || !workspaceReady}>
                Check readiness
              </button>
            </div>
            {doctorResult ? <p className={doctorResult.ok ? "status" : "error-text"}>{doctorResult.ok ? "Runner and local readiness passed." : "Readiness has blockers."}</p> : null}
          </section>

          <section className="onboarding-card" data-testid="onboarding-ready-step">
            <div className="card-heading">
              <span className="step-index">4</span>
              <div>
                <h2>Ready</h2>
                <p className="hint">Enter Console V2 when workspace, sources and runner are valid.</p>
              </div>
            </div>
            <ul className="checklist">
              <li className={workspaceReady ? "is-ready" : ""}>Workspace selected</li>
              <li className={sourcesReady ? "is-ready" : ""}>Sources validated</li>
              <li className={runtimeReady ? "is-ready" : ""}>Runner selected</li>
            </ul>
            <div className="actions">
              <button type="button" onClick={onEnterConsole} disabled={busy || !canEnterConsole} data-testid="onboarding-enter-console">
                Open console
              </button>
              <button type="button" onClick={onRunFirstAnalysis} disabled={busy || !canEnterConsole} data-testid="onboarding-run-first-analysis">
                Run first analysis
              </button>
            </div>
            {firstRunStatus ? <p className="status">{firstRunStatus}</p> : null}
          </section>
        </div>
      </section>
    </main>
  );
}

function StatusPill({ label, ready }: { label: string; ready: boolean }) {
  return <span className={ready ? "status-pill is-ready" : "status-pill"}>{label}</span>;
}

function RepoDiagnostics({ diagnostics }: { diagnostics: Diagnostic[] }) {
  if (diagnostics.length === 0) {
    return null;
  }
  return (
    <div className="diagnostic-list" data-testid="onboarding-repo-diagnostics">
      {diagnostics.map((diagnostic, index) => (
        <p className={diagnostic.level === "error" ? "status err" : "status warn"} key={`${diagnostic.code}-${diagnostic.message}-${index}`}>
          {diagnostic.level === "error" ? "Error" : "Warning"} [{diagnostic.code}]: {diagnostic.message}
          {diagnostic.suggestion ? <span> Next: {diagnostic.suggestion}</span> : null}
        </p>
      ))}
    </div>
  );
}

function buildRepoDiagnostics(guidedRepos: GuidedRepo[], validateResult: ValidateResponse | null): Map<string, Diagnostic[]> {
  const diagnosticsByID = new Map<string, Diagnostic[]>();
  const normalizedNameCounts = new Map<string, number>();
  for (const repo of guidedRepos) {
    const normalizedName = repo.name.trim();
    if (normalizedName) {
      normalizedNameCounts.set(normalizedName, (normalizedNameCounts.get(normalizedName) ?? 0) + 1);
    }
  }

  const append = (repoID: string, diagnostic: Diagnostic) => {
    const diagnostics = diagnosticsByID.get(repoID) ?? [];
    diagnostics.push(diagnostic);
    diagnosticsByID.set(repoID, diagnostics);
  };

  for (const repo of guidedRepos) {
    const name = repo.name.trim();
    if (!name) {
      append(repo.id, {
        level: "error",
        code: "repo_name_required",
        message: "Repo name is required.",
        suggestion: "Add a unique workspace repo name.",
      });
    } else if ((normalizedNameCounts.get(name) ?? 0) > 1) {
      append(repo.id, {
        level: "error",
        code: "repo_name_duplicate",
        message: "Repo names must be unique inside workspace.yaml.",
        suggestion: "Rename one of the duplicate repo rows.",
      });
    }
    const sourceValue = repo.mode === "path" ? repo.path.trim() : repo.git_url.trim();
    if (!sourceValue) {
      append(repo.id, {
        level: "error",
        code: repo.mode === "path" ? "repo_path_required" : "repo_git_url_required",
        message: repo.mode === "path" ? "Local checkout path is required." : "Repository URL is required.",
        suggestion: repo.mode === "path" ? "Enter an existing checkout path." : "Enter a GitHub/GitLab URL.",
      });
    }
  }

  const serverDiagnostics = [...(validateResult?.errors ?? []), ...(validateResult?.warnings ?? [])];
  for (const diagnostic of serverDiagnostics) {
    const repoName = diagnostic.repo?.trim();
    if (!repoName) {
      continue;
    }
    for (const repo of guidedRepos) {
      if (repo.name.trim() === repoName) {
        append(repo.id, diagnostic);
      }
    }
  }

  return diagnosticsByID;
}
