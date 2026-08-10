import { StatusBadge } from "./ConsolePrimitives";
import { diffFileTone } from "../features/analysis/analysisViewModels";
import type { GitDiffResponse } from "../lib/appContracts";

export function GitDiffView({
  gitDiff,
  status,
  onSelectFile,
}: {
  gitDiff: GitDiffResponse | null;
  status: string;
  onSelectFile: (path: string) => void;
}) {
  if (!gitDiff) {
    return <p className="empty-state">{status || "Workspace Git diff is not loaded yet."}</p>;
  }
  const selected = gitDiff.selected_file;
  return (
    <div className="git-diff-view" data-testid="git-diff-view">
      {status ? <p className={gitDiff.empty ? "status ok" : "status warn"}>{status}</p> : null}
      {gitDiff.empty ? <p className="empty-state">No workspace Git changes. Generated artifacts may already be committed or this run has not produced publishable file changes yet.</p> : null}
      <div className="git-diff-layout">
        <aside className="git-diff-file-list" aria-label="changed files">
          <div className="section-heading-row">
            <h3>Changed files</h3>
            <StatusBadge tone={gitDiff.files.length > 0 ? "ok" : "info"}>{gitDiff.files.length}</StatusBadge>
          </div>
          {gitDiff.folders.length > 0 ? (
            <div className="git-diff-folder-summary">
              {gitDiff.folders.map((folder) => <span key={folder.folder}>{folder.folder}: {folder.files} files, +{folder.additions}/-{folder.deletions}</span>)}
            </div>
          ) : null}
          {gitDiff.files.length === 0 ? (
            <p className="hint">No changed files match the current filter.</p>
          ) : (
            <ul>
              {gitDiff.files.slice(0, 40).map((file) => (
                <li key={file.path}>
                  <button type="button" className={`git-diff-file${selected?.path === file.path ? " is-selected" : ""}`} onClick={() => onSelectFile(file.path)} aria-pressed={selected?.path === file.path}>
                    <span><StatusBadge tone={diffFileTone(file.status)}>{file.status}</StatusBadge>{file.path}</span>
                    <code>+{file.additions} / -{file.deletions}{file.binary ? " / binary" : ""}</code>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </aside>
        <section className="git-diff-hunks" data-testid="git-diff-hunks">
          <div className="section-heading-row">
            <h3>{selected?.path || gitDiff.selected_path || "No file selected"}</h3>
            <StatusBadge tone={selected ? diffFileTone(selected.status) : "info"}>{selected?.status ?? "none"}</StatusBadge>
          </div>
          {gitDiff.message ? <p className="hint">{gitDiff.message}</p> : null}
          {selected?.binary ? <p className="empty-state">Binary/non-text diff. Review the file path and status, then use Git tooling for binary inspection.</p> : null}
          {!selected?.binary && gitDiff.hunks.length === 0 ? <p className="empty-state">No line-level hunks for the selected file.</p> : null}
          {gitDiff.hunks.map((hunk, index) => (
            <div className="git-diff-hunk" key={`${hunk.header}-${index}`}>
              <div className="git-diff-hunk-header">{hunk.header}</div>
              {hunk.lines.map((line, lineIndex) => (
                <div className={`git-diff-line ${line.kind}`} key={`${hunk.header}-${lineIndex}`}>
                  <span className="git-diff-line-number">{line.old_line ?? ""}</span>
                  <span className="git-diff-line-number">{line.new_line ?? ""}</span>
                  <code>{line.content || " "}</code>
                </div>
              ))}
            </div>
          ))}
        </section>
      </div>
    </div>
  );
}
