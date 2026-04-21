import type { Diagnostic, GuidedRepo, RepoSourceMode, ValidateResponse } from "../lib/appContracts";

type SetupWorkspacePanelProps = {
  busy: boolean;
  guidedRepos: GuidedRepo[];
  guidedDocsImportsPath: string;
  manifestContent: string;
  validateResult: ValidateResponse | null;
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>;
  onRepoChange: (id: string, patch: Partial<GuidedRepo>) => void;
  onAddRepo: () => void;
  onRemoveRepo: (id: string) => void;
  onDocsImportsPathChange: (value: string) => void;
  onApplyGuidedWorkspaceSetup: () => void;
  onManifestChange: (value: string) => void;
  onSaveManifest: () => void;
  onValidateWorkspace: () => void;
};

export function SetupWorkspacePanel({
  busy,
  guidedRepos,
  guidedDocsImportsPath,
  manifestContent,
  validateResult,
  validationDiagnosticsByRepo,
  onRepoChange,
  onAddRepo,
  onRemoveRepo,
  onDocsImportsPathChange,
  onApplyGuidedWorkspaceSetup,
  onManifestChange,
  onSaveManifest,
  onValidateWorkspace,
}: SetupWorkspacePanelProps) {
  return (
    <section className="panel" data-testid="workspace-panel">
      <h2>Setup: Workspace</h2>
      <p className="hint">Guided setup writes a valid multi-repo `workspace.yaml` draft.</p>
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
            <option value="path">path</option>
            <option value="git_url">git_url</option>
          </select>

          {repo.mode === "path" ? (
            <>
              <label htmlFor={`guidedRepoPath-${repo.id}`}>path</label>
              <input id={`guidedRepoPath-${repo.id}`} value={repo.path} onChange={(event) => onRepoChange(repo.id, { path: event.target.value })} />
            </>
          ) : (
            <>
              <label htmlFor={`guidedRepoGitURL-${repo.id}`}>git_url</label>
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

      <p className="hint">`workspace.yaml` editor (path/git_url sources)</p>
      <textarea value={manifestContent} onChange={(event) => onManifestChange(event.target.value)} rows={12} />
      <div className="actions">
        <button type="button" onClick={onSaveManifest} disabled={busy} data-testid="workspace-save-btn">
          Save workspace.yaml
        </button>
        <button type="button" onClick={onValidateWorkspace} disabled={busy} data-testid="workspace-validate-btn">
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
  );
}
