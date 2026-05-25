import type { Diagnostic, DoctorResponse, GuidedRepo, RepoSourceMode, ValidateResponse } from "../lib/appContracts";

type SetupWorkspacePanelProps = {
  busy: boolean;
  guidedRepos: GuidedRepo[];
  guidedDocsImportsPath: string;
  manifestContent: string;
  validateResult: ValidateResponse | null;
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>;
  doctorResult: DoctorResponse | null;
  doctorStatus: string;
  firstRunStatus: string;
  setupRuntime: string;
  setupRuntimeProvider: string;
  onRepoChange: (id: string, patch: Partial<GuidedRepo>) => void;
  onAddRepo: () => void;
  onRemoveRepo: (id: string) => void;
  onDocsImportsPathChange: (value: string) => void;
  onApplyGuidedWorkspaceSetup: () => void;
  onManifestChange: (value: string) => void;
  onSaveManifest: () => void;
  onValidateWorkspace: () => void;
  onSetupRuntimeChange: (value: string) => void;
  onSetupRuntimeProviderChange: (value: string) => void;
  onCheckDoctor: () => void;
  onRunFirstAnalysis: () => void;
};

export function SetupWorkspacePanel({
  busy,
  guidedRepos,
  guidedDocsImportsPath,
  manifestContent,
  validateResult,
  validationDiagnosticsByRepo,
  doctorResult,
  doctorStatus,
  firstRunStatus,
  setupRuntime,
  setupRuntimeProvider,
  onRepoChange,
  onAddRepo,
  onRemoveRepo,
  onDocsImportsPathChange,
  onApplyGuidedWorkspaceSetup,
  onManifestChange,
  onSaveManifest,
  onValidateWorkspace,
  onSetupRuntimeChange,
  onSetupRuntimeProviderChange,
  onCheckDoctor,
  onRunFirstAnalysis,
}: SetupWorkspacePanelProps) {
  const firstRepo = guidedRepos[0];
  const suggestedWorkspace = `~/acp-workspaces/${slugify(firstRepo?.name || "my-service")}`;
  const validated = validateResult?.ok === true;

  return (
    <section className="panel" data-testid="workspace-panel">
      <h2>Setup: First run</h2>
      <p className="hint">Connect one GitHub/GitLab URL or local checkout, validate readiness, then run the first architecture analysis.</p>

      <ol className="setup-steps" data-testid="setup-stepper">
        <li className="active">Source</li>
        <li>Workspace</li>
        <li>Runtime</li>
        <li>Validate</li>
        <li>Run</li>
      </ol>

      <div className="setup-band">
        <h3>1. Source</h3>
        <p className="hint">Start with a repository URL if the project only exists on GitHub/GitLab. Private repos use your local git authentication.</p>
      </div>
      {guidedRepos.map((repo, index) => (
        <div className="repo-card" key={repo.id}>
          <div className="repo-card-head">
            <h3>Repo {index + 1}</h3>
            <button type="button" className="inline-danger" onClick={() => onRemoveRepo(repo.id)} disabled={busy || guidedRepos.length <= 1}>
              Remove
            </button>
          </div>

          <label htmlFor={`guidedRepoName-${repo.id}`}>Repo name</label>
          <input id={`guidedRepoName-${repo.id}`} value={repo.name} onChange={(event) => onRepoChange(repo.id, { name: event.target.value })} />

          <label htmlFor={`guidedRepoMode-${repo.id}`}>Repo source type</label>
          <select
            id={`guidedRepoMode-${repo.id}`}
            value={repo.mode}
            onChange={(event) => onRepoChange(repo.id, { mode: event.target.value as RepoSourceMode })}
          >
            <option value="git_url">GitHub/GitLab URL</option>
            <option value="path">Local folder</option>
          </select>

          {repo.mode === "path" ? (
            <>
              <label htmlFor={`guidedRepoPath-${repo.id}`}>Local checkout path</label>
              <input id={`guidedRepoPath-${repo.id}`} value={repo.path} onChange={(event) => onRepoChange(repo.id, { path: event.target.value })} />
            </>
          ) : (
            <>
              <label htmlFor={`guidedRepoGitURL-${repo.id}`}>Repository URL</label>
              <input
                id={`guidedRepoGitURL-${repo.id}`}
                value={repo.git_url}
                onChange={(event) => onRepoChange(repo.id, { git_url: event.target.value })}
              />
            </>
          )}

          <label htmlFor={`guidedRepoRef-${repo.id}`}>ref (optional)</label>
          <input
            id={`guidedRepoRef-${repo.id}`}
            value={repo.ref}
            onChange={(event) => onRepoChange(repo.id, { ref: event.target.value })}
            placeholder="Leave empty to use current checkout"
          />
        </div>
      ))}

      <div className="setup-band">
        <h3>2. Workspace</h3>
        <p className="hint">
          The workspace path is selected when starting `acp serve`. Recommended default for this source: <code>{suggestedWorkspace}</code>.
        </p>
      </div>
      <label htmlFor="guidedDocsImportsPath">docs.imports_path</label>
      <input id="guidedDocsImportsPath" value={guidedDocsImportsPath} onChange={(event) => onDocsImportsPathChange(event.target.value)} />

      <div className="actions">
        <button type="button" onClick={onAddRepo} disabled={busy}>
          Add repo
        </button>
        <button type="button" onClick={onApplyGuidedWorkspaceSetup} disabled={busy}>
          Apply guided workspace form
        </button>
      </div>

      <div className="setup-band">
        <h3>3. Runtime</h3>
        <p className="hint">Use fake for the first deterministic walkthrough. Headless requires a local provider command.</p>
      </div>
      <div className="columns compact">
        <div>
          <label htmlFor="setupRuntime">Runtime mode</label>
          <select id="setupRuntime" value={setupRuntime} onChange={(event) => onSetupRuntimeChange(event.target.value)}>
            <option value="fake">fake</option>
            <option value="headless">headless</option>
          </select>
        </div>
        <div>
          <label htmlFor="setupRuntimeProvider">Headless provider</label>
          <select
            id="setupRuntimeProvider"
            value={setupRuntimeProvider}
            onChange={(event) => onSetupRuntimeProviderChange(event.target.value)}
            disabled={setupRuntime !== "headless"}
          >
            <option value="claude-code">claude-code</option>
            <option value="qwen-code">qwen-code</option>
            <option value="codex-code">codex-code</option>
          </select>
        </div>
      </div>
      {setupRuntime === "headless" ? (
        <p className="status warn">
          Headless mode is process-scoped. If this service was started with `--runtime fake`, restart it with `--runtime headless --runtime-provider {setupRuntimeProvider}`.
        </p>
      ) : null}

      <details className="advanced-block">
        <summary>Advanced workspace.yaml editor</summary>
        <textarea
          id="legacyWorkspaceManifestEditor"
          name="legacyWorkspaceManifestEditor"
          aria-label="workspace.yaml content"
          value={manifestContent}
          onChange={(event) => onManifestChange(event.target.value)}
          rows={12}
        />
      </details>

      <div className="setup-band">
        <h3>4. Validate</h3>
        <p className="hint">Save the generated manifest, validate workspace layout and repo access, then run doctor for local prerequisites.</p>
      </div>
      <div className="actions">
        <button type="button" onClick={onSaveManifest} disabled={busy} data-testid="workspace-save-btn">
          Save and validate workspace.yaml
        </button>
        <button type="button" onClick={onValidateWorkspace} disabled={busy} data-testid="workspace-validate-btn">
          Validate workspace
        </button>
        <button type="button" onClick={onCheckDoctor} disabled={busy} data-testid="setup-doctor-btn">
          Check local readiness
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
                  {diagnostic.suggestion ? <span> Next: {diagnostic.suggestion}</span> : null}
                </p>
              ))}
            </div>
          ))}
        </div>
      ) : null}
      {doctorStatus ? <p className="status">{doctorStatus}</p> : null}
      {doctorResult ? (
        <div className="status-block" data-testid="setup-doctor-result">
          <p>Status: {doctorResult.ok ? "ready" : "needs attention"}</p>
          <ul className="check-list">
            {doctorResult.checks.map((check) => (
              <li className={`check ${check.status}`} key={check.id}>
                <strong>{check.label}:</strong> {check.message}
                {check.suggestion ? <span> Next: {check.suggestion}</span> : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="setup-band">
        <h3>5. Run</h3>
        <p className="hint">After validation passes, start `init` and inspect the generated coverage, artifacts and diagrams.</p>
      </div>
      <div className="actions">
        <button type="button" onClick={onRunFirstAnalysis} disabled={busy || !validated} data-testid="setup-run-first-btn">
          Run first analysis
        </button>
      </div>
      {firstRunStatus ? <p className="status ok">{firstRunStatus}</p> : null}
    </section>
  );
}

function slugify(value: string): string {
  const normalized = value.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
  return normalized || "my-service";
}
