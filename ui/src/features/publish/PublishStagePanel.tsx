import { useEffect, useState, type ComponentProps } from "react";

import { BaselineGitPanel } from "../../components/BaselineGitPanel";
import { GitDiffView } from "../../components/GitDiffView";
import { StatusBadge } from "../../components/ConsolePrimitives";
import { TabNav, tabPanelProps } from "../../components/TabNav";
import { PublishGateSection } from "./PublishGateSection";
import {
  PUBLISH_ARTIFACT_FILTERS,
  publishArtifactFilterLabel,
  publishArtifactMatchesFilter,
  type PublishArtifactFilter,
} from "../../lib/artifactFilters";
import {
  buildPublishFolderSummaries,
  buildPublishGateItems,
  comparePublishArtifactPriority,
  gitDiffScopeHint,
  gitDiffScopeTitle,
  publishTabLabel,
  type PublishGateItem,
} from "./publishUtils";
import type { Artifact, GitDiffResponse } from "../../lib/appContracts";
import type { LoadGitDiffOptions } from "../../lib/gitDiffApi";
import { countMarkdownItems } from "../review/reviewUtils";

type PublishStageProps = ComponentProps<typeof BaselineGitPanel> & {
  artifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  openQuestions: string;
  externalGateItems?: PublishGateItem[];
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  onPreviewArtifact: (path: string) => void;
};

export function PublishStagePanel({
  busy,
  gitMessage,
  proposalBranch,
  gitStatus,
  gitError,
  artifacts,
  selectedArtifact,
  selectedArtifactContent,
  openQuestions,
  externalGateItems = [],
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
  onGitMessageChange,
  onProposalBranchChange,
  onCommit,
  onCreateProposalBranch,
  onPreviewArtifact,
}: PublishStageProps) {
  const [publishView, setPublishView] = useState<"preview" | "diff" | "evidence" | "changelog">("preview");
  const [localSelectedPath, setLocalSelectedPath] = useState("");
  const [copyStatus, setCopyStatus] = useState("");
  const [artifactFilter, setArtifactFilter] = useState<PublishArtifactFilter>("all");
  const publishArtifacts = artifacts
    .filter((artifact) => artifact.path.trim().length > 0)
    .sort(comparePublishArtifactPriority);
  const changedPathSet = new Set(gitDiff?.files.map((file) => file.path) ?? []);
  const filteredPublishArtifacts = publishArtifacts.filter((artifact) => publishArtifactMatchesFilter(artifact, artifactFilter, changedPathSet));
  const visibleChangedFiles = artifactFilter === "all" || artifactFilter === "changed" ? (gitDiff?.files ?? []) : [];
  const selectedDiffFilePath = gitDiff?.selected_file?.path;
  const effectiveSelectedPath =
    (localSelectedPath && publishArtifacts.some((artifact) => artifact.path === localSelectedPath) ? localSelectedPath : "") ||
    (selectedArtifact && publishArtifacts.some((artifact) => artifact.path === selectedArtifact) ? selectedArtifact : "") ||
    publishArtifacts[0]?.path ||
    "";
  const folderSummaries = buildPublishFolderSummaries(publishArtifacts);
  const changelogArtifacts = publishArtifacts.filter((artifact) => artifact.kind === "changelog" || artifact.path.startsWith("reports/changelog/"));
  const selectedChangelogPath =
    (localSelectedPath && changelogArtifacts.some((artifact) => artifact.path === localSelectedPath) ? localSelectedPath : "") ||
    (selectedArtifact && changelogArtifacts.some((artifact) => artifact.path === selectedArtifact) ? selectedArtifact : "") ||
    changelogArtifacts[0]?.path ||
    "";
  const activePreviewPath = publishView === "changelog" && selectedChangelogPath ? selectedChangelogPath : effectiveSelectedPath;
  const selectedPublishArtifact = publishArtifacts.find((artifact) => artifact.path === activePreviewPath);
  const selectedPublishContent = selectedArtifact === activePreviewPath ? selectedArtifactContent : "";
  const selectedChangelogArtifact = changelogArtifacts.find((artifact) => artifact.path === selectedChangelogPath);
  const selectedChangelogContent = selectedArtifact === selectedChangelogPath ? selectedArtifactContent : "";
  const hasAuthoritativeFullWorkspaceDiff = Boolean(
    gitDiff
      && gitDiff.scope === "full_workspace"
      && gitDiff.run_id == null
      && gitDiff.state !== "unknown"
      && gitDiff.state !== "blocked"
      && gitDiff.state !== "stale",
  );
  const gateItems = [
    ...externalGateItems,
    ...buildPublishGateItems({
      previewArtifactCount: publishArtifacts.length,
      previewFolderCount: folderSummaries.length,
      gitDiff,
      gitDiffStatus,
      gitMessage,
      proposalBranch,
      openQuestions,
    }),
  ];
  const blockingGateItems = gateItems.filter((item) => item.tone === "error");
  const openQuestionGateItems = gateItems.filter((item) => item.tone === "warn" && item.label.toLowerCase().includes("open question"));
  const warningGateItems = gateItems.filter((item) => item.tone === "warn" && !openQuestionGateItems.includes(item));
  const readyGateItems = gateItems.filter((item) => item.tone === "ok" || item.tone === "info");
  const openQuestionCount = countMarkdownItems(openQuestions);
  const gitMutationDisabled = busy || blockingGateItems.length > 0 || !hasAuthoritativeFullWorkspaceDiff;
  const gitMutationBlockedTitle =
    blockingGateItems.length > 0
      ? "Resolve publish gate blockers before changing Git publication state."
      : !hasAuthoritativeFullWorkspaceDiff
        ? "Load the full workspace Git inventory before changing Git publication state."
        : undefined;
  const realFolderSummaries =
    gitDiff?.folders.map((summary) => ({
      folder: summary.folder,
      count: summary.files,
      sample: `+${summary.additions} / -${summary.deletions}`,
    })) ?? [];
  const visibleFolderSummaries = gitDiff ? realFolderSummaries : [];
  const diffScopeTitle = gitDiffScopeTitle(gitDiff);
  const diffScopeHint = gitDiffScopeHint(gitDiff);
  const primaryPublishGateItem = blockingGateItems[0] ?? openQuestionGateItems[0] ?? warningGateItems[0];
  const publishGateTone = blockingGateItems.length > 0 ? "error" : warningGateItems.length > 0 || openQuestionGateItems.length > 0 ? "warn" : "ok";
  const publishGateLabel = blockingGateItems.length > 0 ? "blocked" : warningGateItems.length > 0 || openQuestionGateItems.length > 0 ? "review" : "ready";
  const publishGateDetail = primaryPublishGateItem ? `${primaryPublishGateItem.label}: ${primaryPublishGateItem.detail}` : "Git actions are allowed after final review.";
  const gitActionLabel = gitError ? "failed" : blockingGateItems.length > 0 ? "blocked" : gitDiff?.empty ? "no changes" : gitMessage.trim() ? "ready" : "needs message";
  const gitActionDetail =
    gitError
      ? gitError
      : blockingGateItems.length > 0
      ? `${blockingGateItems[0].label}: ${blockingGateItems[0].detail}`
      : gitDiff?.empty
        ? "The workspace is clean; there is nothing to commit."
      : gitMessage.trim()
        ? `Commit message prepared: ${gitMessage}`
        : "Commit message is empty.";
  const commitDisabled = gitMutationDisabled || Boolean(gitDiff?.empty) || !gitMessage.trim();
  const commitBlockedTitle =
    gitMutationBlockedTitle
      ?? (gitDiff?.empty ? "There are no workspace changes to commit." : !gitMessage.trim() ? "Enter a commit message before committing." : "Commit reviewed workspace artifacts.");
  const authorityNoticeTitle = hasAuthoritativeFullWorkspaceDiff ? "Authoritative workspace scope loaded" : "Loading authoritative workspace scope";

  useEffect(() => {
    if (activePreviewPath && selectedArtifact !== activePreviewPath) {
      onPreviewArtifact(activePreviewPath);
    }
  }, [activePreviewPath, selectedArtifact, onPreviewArtifact]);

  function handleSelectPublishArtifact(path: string) {
    setLocalSelectedPath(path);
    setPublishView("preview");
    onPreviewArtifact(path);
  }

  function handleSelectChangelogArtifact(path: string) {
    setLocalSelectedPath(path);
    onPreviewArtifact(path);
  }

  async function handleCopyCommitMessage() {
    if (!navigator.clipboard) {
      setCopyStatus("Clipboard unavailable in this browser.");
      return;
    }
    try {
      await navigator.clipboard.writeText(gitMessage);
      setCopyStatus("Commit message copied.");
    } catch (error) {
      setCopyStatus(error instanceof Error ? error.message : "Commit message copy failed.");
    }
  }

  return (
    <div className="stage-stack" data-testid="publish-panel">
      <section className="panel stage-panel">
        <div className="stage-header">
          <div>
            <h1>Publish</h1>
            <p className="hint">Review workspace artifacts, check publication blockers and prepare Git commit/proposal branch handoff.</p>
          </div>
          <StatusBadge tone={blockingGateItems.length > 0 ? "error" : warningGateItems.length > 0 ? "warn" : hasAuthoritativeFullWorkspaceDiff ? "ok" : "info"}>
            {blockingGateItems.length > 0 ? "blocked" : warningGateItems.length > 0 ? "review" : hasAuthoritativeFullWorkspaceDiff ? "ready" : "loading"}
          </StatusBadge>
        </div>
        <div className="publish-readiness-summary" data-testid="publish-readiness-summary" aria-label="Publish readiness summary">
          <div>
            <span className="metric-label">Workspace scope</span>
            <strong>{gitDiff ? `${gitDiff.files.length} changed` : "Unknown"}</strong>
            <span>{visibleFolderSummaries.length} folders in scope</span>
          </div>
          <div>
            <span className="metric-label">Gate</span>
            <strong>{publishGateLabel}</strong>
            <span>{publishGateDetail}</span>
          </div>
          <div>
            <span className="metric-label">Open questions</span>
            <strong>{openQuestionCount}</strong>
            <span>{openQuestionCount > 0 ? "Review before commit." : "No loaded open questions."}</span>
          </div>
          <div>
            <span className="metric-label">Git action</span>
            <strong>{gitActionLabel}</strong>
            <span>{gitActionDetail}</span>
          </div>
        </div>
        <div className={`publish-authority-notice ${hasAuthoritativeFullWorkspaceDiff ? "is-ready" : "is-loading"}`} role="status"><span aria-hidden="true">{hasAuthoritativeFullWorkspaceDiff ? "✓" : "…"}</span><div><strong>{authorityNoticeTitle}</strong><p>{hasAuthoritativeFullWorkspaceDiff ? `${gitDiff?.files.length ?? 0} changed files across ${visibleFolderSummaries.length} folders. Commit includes every uncommitted workspace file; artifact previews are for navigation only.` : "The complete workspace Git inventory is loading. Commit actions stay disabled until the authoritative full-scope response arrives."}</p></div></div>
      </section>

      <nav className="publish-section-jumps" aria-label="Publish sections" data-testid="publish-section-jumps">
        <a href="#publish-diff-summary">Diff</a>
        <a href="#publish-preview-panel">Preview</a>
        <a href="#publish-gate-panel">Gate</a>
        <a href="#publish-commit-plan">Commit</a>
      </nav>

      <div className="publish-review-room">
        <section className="publish-diff-summary" id="publish-diff-summary" data-testid="publish-diff-summary">
          <div className="panel-subheader">
            <div>
              <h2>{diffScopeTitle}</h2>
              <p className="hint">{diffScopeHint}</p>
            </div>
            <StatusBadge tone={gitDiff && !gitDiff.empty ? "ok" : gitDiff ? "info" : "warn"}>
              {gitDiff ? `${gitDiff.files.length} changed` : "Git inventory unavailable"}
            </StatusBadge>
          </div>
          <div className="actions compact-actions">
            <button type="button" className="secondary-action" onClick={() => onLoadGitDiff({ runId: null })}>
              Refresh full workspace diff
            </button>
          </div>
          {gitDiffStatus ? <p className={gitDiff?.empty ? "status ok" : "status warn"}>{gitDiffStatus}</p> : null}
          {visibleFolderSummaries.length === 0 ? (
            <p className="empty-state">No workspace Git changes are available for publication yet.</p>
          ) : (
            <div className="publish-folder-list">
              {visibleFolderSummaries.map((summary) => (
                <div key={summary.folder} className="publish-folder-row">
                  <div>
                    <strong>{summary.folder}</strong>
                    <span>{summary.count} file refs</span>
                  </div>
                  <span>{summary.sample}</span>
                </div>
              ))}
            </div>
          )}
          <TabNav
            ariaLabel="Publish artifact filters"
            className="artifact-filter-tabs"
            idBase="publish-artifact-filters"
            testId="publish-artifact-filters"
            value={artifactFilter}
            onChange={setArtifactFilter}
            options={PUBLISH_ARTIFACT_FILTERS}
          />
          <div {...tabPanelProps("publish-artifact-filters", artifactFilter)}>
            {visibleChangedFiles.length ? (
              <div className="publish-artifact-list compact" role="list" aria-label="changed workspace files">
                {visibleChangedFiles.slice(0, 16).map((file) => (
                  <div key={file.path} role="listitem">
                    <button
                      type="button"
                      className={`publish-artifact-row${selectedDiffFilePath === file.path ? " is-selected" : ""}`}
                      onClick={() => {
                        setPublishView("diff");
                        onLoadGitDiff({ runId: null, path: file.path });
                      }}
                      aria-pressed={selectedDiffFilePath === file.path}
                    >
                      <span>{file.path}</span>
                      <code>
                        {file.status} · +{file.additions}/-{file.deletions}
                      </code>
                    </button>
                  </div>
                ))}
              </div>
            ) : null}
            {filteredPublishArtifacts.length === 0 ? (
              <p className="empty-state" data-testid="publish-artifact-filter-empty">
                {publishArtifacts.length === 0
                  ? "No selected-run artifacts are available for publication preview."
                  : `No ${publishArtifactFilterLabel(artifactFilter).toLowerCase()} artifact refs are available in this publication view.`}
              </p>
            ) : (
              <div className="publish-artifact-list" role="list" aria-label="publish artifact preview list">
                {filteredPublishArtifacts.slice(0, 12).map((artifact) => (
                  <div key={artifact.path} role="listitem">
                    <button
                      type="button"
                      className={`publish-artifact-row${activePreviewPath === artifact.path ? " is-selected" : ""}`}
                      onClick={() => handleSelectPublishArtifact(artifact.path)}
                      aria-pressed={activePreviewPath === artifact.path}
                    >
                      <span>{artifact.label || artifact.path}</span>
                      <code>{artifact.path}</code>
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </section>

        <section className="publish-preview-panel" id="publish-preview-panel" data-testid="publish-preview-panel">
          <TabNav
            ariaLabel="Publish preview tabs"
            className="publish-preview-tabs"
            idBase="publish-preview-tabs"
            testId="publish-preview-tabs"
            value={publishView}
            onChange={setPublishView}
            options={(["preview", "diff", "evidence", "changelog"] as const).map((view) => ({ id: view, label: publishTabLabel(view) }))}
          />
          <div className="publish-tab-panel" data-testid="publish-tab-panel" {...tabPanelProps("publish-preview-tabs", publishView)}>
            {publishView === "preview" ? (
              <div className="publish-selected-preview" data-testid="publish-selected-preview">
                <h2>Selected artifact preview</h2>
                {selectedPublishArtifact ? (
                  <>
                    <p className="hint">{selectedPublishArtifact.path}</p>
                    {selectedPublishContent ? (
                      <pre data-testid="publish-selected-preview-content">{selectedPublishContent}</pre>
                    ) : (
                      <p className="empty-state" data-testid="publish-selected-preview-empty">Select an artifact to load its preview in this Publish room.</p>
                    )}
                  </>
                ) : (
                  <p className="empty-state" data-testid="publish-selected-preview-empty">No artifact selected for publication preview.</p>
                )}
              </div>
            ) : null}
            {publishView === "diff" ? (
              <>
                <h2>{diffScopeTitle}</h2>
                <p className="hint">{diffScopeHint}</p>
                <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ runId: null, path })} />
              </>
            ) : null}
            {publishView === "evidence" ? (
              <>
                <h2>Evidence before commit</h2>
                {openQuestions.trim() ? <pre>{openQuestions}</pre> : <p className="hint">No open-question artifact content is currently loaded.</p>}
                <p className="hint">{visibleFolderSummaries.length} workspace folders have Git diff or artifact refs in this publication view.</p>
              </>
            ) : null}
            {publishView === "changelog" ? (
              <>
                <h2>Changelog</h2>
                {changelogArtifacts.length === 0 ? (
                  <>
                    <p className="empty-state">No changelog artifact is available from the selected run.</p>
                    <p className="hint">Keep publication review focused on the selected artifact preview, evidence tab and Publish gate until a changelog artifact is generated.</p>
                  </>
                ) : (
                  <>
                    <div className="publish-artifact-list compact" role="list" aria-label="publish changelog artifacts">
                      {changelogArtifacts.map((artifact) => (
                        <div key={artifact.path} role="listitem">
                          <button
                            type="button"
                            className={`publish-artifact-row${selectedChangelogPath === artifact.path ? " is-selected" : ""}`}
                            onClick={() => handleSelectChangelogArtifact(artifact.path)}
                            aria-pressed={selectedChangelogPath === artifact.path}
                          >
                            <span>{artifact.label || artifact.path}</span>
                            <code>{artifact.path}</code>
                          </button>
                        </div>
                      ))}
                    </div>
                    <div className="publish-changelog-preview">
                      <h3>Selected changelog preview</h3>
                      {selectedChangelogArtifact ? (
                        <>
                          <p className="hint">{selectedChangelogArtifact.path}</p>
                          {selectedChangelogContent ? <pre>{selectedChangelogContent}</pre> : <p className="empty-state">Loading changelog preview...</p>}
                        </>
                      ) : (
                        <p className="empty-state">No changelog selected.</p>
                      )}
                    </div>
                  </>
                )}
              </>
            ) : null}
          </div>
        </section>

        <aside className="publish-side-column">
          <section className="publish-gate-panel" id="publish-gate-panel" data-testid="publish-gate-panel">
            <div className="panel-subheader">
              <div>
                <h2>Publish gate</h2>
                <p className="hint">Checks gate Git commit and proposal branch actions; Git commands stay explicit operator actions.</p>
              </div>
              <StatusBadge tone={publishGateTone}>{publishGateLabel}</StatusBadge>
            </div>
            <PublishGateSection testId="publish-hard-blockers" title="Hard blockers" emptyLabel="No hard blockers. Git actions are allowed." items={blockingGateItems} />
            <PublishGateSection testId="publish-review-warnings" title="Review warnings" emptyLabel="No review warnings." items={warningGateItems} />
            <PublishGateSection testId="publish-open-questions" title="Open questions" emptyLabel="No open questions loaded." items={openQuestionGateItems} />
            <PublishGateSection testId="publish-ready-checks" title="Ready checks" emptyLabel="No ready checks yet." items={readyGateItems} />
          </section>

          <section className="publish-commit-plan" id="publish-commit-plan" data-testid="publish-commit-plan">
            <div className="panel-subheader">
              <div>
                <h2>Commit plan</h2>
                <p className="hint">Prepared commit/proposal branch actions use the existing Git API.</p>
              </div>
              <StatusBadge tone={gitError ? "error" : gitStatus ? "ok" : "info"}>{gitError ? "failed" : gitStatus ? "updated" : "pending"}</StatusBadge>
            </div>
            <dl className="compact-defs">
              <div>
                <dt>Folders</dt>
                <dd>{visibleFolderSummaries.map((summary) => summary.folder).join(", ") || "No changed folders yet"}</dd>
              </div>
              <div>
                <dt>Proposal branch</dt>
                <dd>{proposalBranch || "proposal branch not prepared"}</dd>
              </div>
            </dl>
            <label htmlFor="publishGitMessage">Commit message</label>
            <input id="publishGitMessage" value={gitMessage} onChange={(event) => onGitMessageChange(event.target.value)} />
            {gitError ? (
              <div className="publish-git-recovery" data-testid="publish-git-action-recovery" role="alert">
                <strong>Git action failed</strong>
                <span>{gitError}</span>
                <p>Workspace Git state was not changed by this action. Review the message or branch name, check local Git permissions/status, then retry.</p>
              </div>
            ) : null}
            <div className="actions publish-actions">
              <button
                type="button"
                className="publish-primary-action"
                aria-label="Review workspace commit"
                onClick={onCommit}
                disabled={commitDisabled}
                title={commitBlockedTitle}
                data-testid="publish-commit-selected-btn"
              >
                  <span data-testid="git-commit-btn">Commit all workspace changes</span>
              </button>
              <button type="button" className="link-button" onClick={() => void handleCopyCommitMessage()}>
                Copy commit message
              </button>
            </div>
            <label htmlFor="publishProposalBranch">Proposal branch</label>
            <input id="publishProposalBranch" value={proposalBranch} onChange={(event) => onProposalBranchChange(event.target.value)} />
            <button type="button" onClick={onCreateProposalBranch} disabled={gitMutationDisabled} title={gitMutationBlockedTitle}>
              <span data-testid="git-proposal-branch-btn">Create/Switch proposal branch</span>
            </button>
            {copyStatus ? <p className="status ok">{copyStatus}</p> : null}
            {gitStatus ? <p className="status ok">{gitStatus}</p> : null}
          </section>
        </aside>
      </div>
    </div>
  );
}

