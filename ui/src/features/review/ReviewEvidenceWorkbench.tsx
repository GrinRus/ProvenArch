import { useEffect, useState } from "react";

import { ArtifactPathButton, StatusBadge } from "../../components/ConsolePrimitives";
import { EvidenceViewer } from "../../components/EvidenceViewer";
import { GitDiffView } from "../../components/GitDiffView";
import { TabNav, tabPanelProps } from "../../components/TabNav";
import {
  REVIEW_ARTIFACT_FILTERS,
  filterReviewArtifactGroups,
  reviewArtifactFilterLabel,
  reviewArtifactGroupCategory,
  reviewArtifactGroupCategoryLabel,
  type ArtifactGroup,
  type ReviewArtifactFilter,
} from "../../lib/artifactFilters";
import type {
  Artifact,
  GitDiffResponse,
  ReviewQueueItem,
  RunReviewSummaryResponse,
} from "../../lib/appContracts";
import type { LoadGitDiffOptions } from "../../lib/gitDiffApi";
import { reviewDecisionSummary, type ReviewTrustStatus } from "./reviewUtils";
import { ReviewQueuePanel } from "./ReviewQueuePanel";

export function ReviewEvidenceWorkbench({
  routeView,
  coverageSummary,
  openQuestions,
  openQuestionCount,
  trustStatus,
  overviewArtifact,
  findingsArtifact,
  artifactGroups,
  diagramArtifacts,
  selectedArtifact,
  selectedArtifactContent,
  evidenceStatus,
  evidenceIssues,
  selectedArtifactIsLoading,
  reviewSummary,
  demo,
  reviewQueue,
  gitDiff,
  gitDiffStatus,
  onLoadGitDiff,
  onOpenArtifact,
}: {
  routeView: "overview" | "evidence" | "findings" | "diff";
  coverageSummary: string;
  openQuestions: string;
  openQuestionCount: number;
  trustStatus: ReviewTrustStatus;
  overviewArtifact?: Artifact;
  findingsArtifact?: Artifact;
  artifactGroups: ArtifactGroup[];
  diagramArtifacts: Artifact[];
  selectedArtifact: string;
  selectedArtifactContent: string;
  evidenceStatus: "idle" | "loading" | "available" | "partial" | "not_produced" | "unavailable" | "error";
  evidenceIssues: Array<{ code: string; message: string; path?: string }>;
  selectedArtifactIsLoading: boolean;
  reviewSummary: RunReviewSummaryResponse | null;
  demo: boolean;
  reviewQueue: ReviewQueueItem[];
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
  onOpenArtifact: (path: string) => void;
}) {
  const evidenceView: "preview" | "diff" = routeView === "diff" ? "diff" : "preview";
  const [artifactFilter, setArtifactFilter] = useState<ReviewArtifactFilter>("all");
  const [artifactExplorerOpen, setArtifactExplorerOpen] = useState(false);
  const visibleArtifactGroups = filterReviewArtifactGroups(artifactGroups, artifactFilter);
  const visibleArtifactCount = visibleArtifactGroups.reduce((sum, group) => sum + group.artifacts.length, 0);

  useEffect(() => {
    if (evidenceView === "diff" && selectedArtifact) {
      onLoadGitDiff({ path: selectedArtifact });
    }
  }, [evidenceView, onLoadGitDiff, selectedArtifact]);

  return (
    <div className="review-workbench">
      <aside className="review-task-lane" id="review-task-lane" aria-label="Review tasks and supporting artifacts">
        <ReviewQueuePanel queue={reviewQueue} selectedArtifact={selectedArtifact} onOpenArtifact={onOpenArtifact} />
        <details className="review-artifact-explorer" data-testid="review-artifact-explorer" id="review-artifacts" open={artifactExplorerOpen} onToggle={(event) => setArtifactExplorerOpen(event.currentTarget.open)}>
          <summary className="review-artifact-explorer-summary" data-testid="review-artifact-explorer-toggle">
            <span className="review-artifact-summary-copy"><strong>Artifact explorer</strong><span>Secondary browser for all generated files.</span></span>
            <StatusBadge tone={visibleArtifactGroups.length > 0 ? "ok" : "info"}>{artifactFilter === "all" ? `${artifactGroups.length} groups` : `${visibleArtifactCount} refs`}</StatusBadge>
          </summary>
          <div className="review-artifact-explorer-body">
            <TabNav ariaLabel="Review artifact filters" className="artifact-filter-tabs" idBase="review-artifact-filters" testId="review-artifact-filters" value={artifactFilter} onChange={(filter) => { setArtifactFilter(filter); setArtifactExplorerOpen(true); }} options={REVIEW_ARTIFACT_FILTERS} />
            <div {...tabPanelProps("review-artifact-filters", artifactFilter)}>
              {visibleArtifactGroups.length === 0 ? (
                <p className="hint">{artifactGroups.length === 0 ? "No selected-run artifacts yet. Run Analysis before evidence review." : `No ${reviewArtifactFilterLabel(artifactFilter).toLowerCase()} artifacts are available in this run.`}</p>
              ) : (
                <div className="artifact-group-list" data-testid="results-artifacts-panel">
                  {visibleArtifactGroups.map((group) => (
                    <section key={group.name} className={`artifact-group ${reviewArtifactGroupCategory(group.name)}`}>
                      <div className="artifact-group-heading"><h3>{group.name}</h3><span>{reviewArtifactGroupCategoryLabel(group.name)}</span></div>
                      <ul data-testid={group.name === "reports/diagrams" ? "run-diagrams-list" : undefined}>
                        {group.artifacts.map((artifact) => <li key={`${artifact.kind}-${artifact.path}`}><ArtifactPathButton path={artifact.path} label={artifact.label} kind={artifact.kind} selected={artifact.path === selectedArtifact} onOpenArtifact={onOpenArtifact} /></li>)}
                      </ul>
                    </section>
                  ))}
                </div>
              )}
            </div>
          </div>
        </details>
      </aside>

      <section className="review-evidence-preview" id="review-evidence-preview" data-testid="review-evidence-preview">
        <div className="section-heading-row"><div><h2>Evidence preview</h2><p className="hint">Select an artifact to inspect the reviewable evidence body.</p></div><span className="status">Validator-approved snapshot · human review is recorded through publication</span></div>
        <div className="review-mode-content">
          {evidenceView === "preview" ? (selectedArtifactIsLoading ? <p className="hint">Loading evidence...</p> : <EvidenceViewer path={selectedArtifact} content={selectedArtifactContent} runId={reviewSummary?.run_id} sourceMode="run_snapshot" provenance={demo ? "demo" : reviewSummary ? "live" : "unknown"} status={evidenceStatus === "partial" ? "partial" : evidenceStatus === "error" ? "error" : evidenceStatus === "unavailable" || evidenceStatus === "not_produced" ? "unavailable" : "available"} issues={evidenceIssues} onOpenArtifact={onOpenArtifact} />) : null}
          {evidenceView === "diff" ? <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ path })} /> : null}
        </div>
      </section>

      <aside className="review-intel" id="review-trust" data-testid="review-citation-coverage">
        <div className="section-heading-row"><h2>Citations / coverage</h2><StatusBadge tone={trustStatus.tone}>{trustStatus.label}</StatusBadge></div>
        <div className="review-intel-grid">
          <div className="metric-tile"><span className="metric-label">Architecture overview</span><strong>{overviewArtifact ? "ready" : "missing"}</strong></div>
          <div className="metric-tile"><span className="metric-label">Coverage summary</span><strong>{coverageSummary ? "ready" : "missing"}</strong></div>
          <div className="metric-tile"><span className="metric-label">Findings</span><strong>{findingsArtifact ? "ready" : "missing"}</strong></div>
          <div className="metric-tile"><span className="metric-label">Open questions</span><strong>{openQuestionCount}</strong></div>
          <div className="metric-tile"><span className="metric-label">Diagrams</span><strong>{diagramArtifacts.length}</strong></div>
        </div>
        <div className="trust-panel"><strong>{trustStatus.title}</strong><span>{trustStatus.detail}</span><span className="review-decision-summary" data-testid="review-decision-summary">{reviewDecisionSummary(trustStatus, openQuestionCount)}</span></div>
        <div className="review-source-lists"><details><summary>Coverage summary</summary><pre data-testid="coverage-summary-content">{coverageSummary || "No coverage summary yet."}</pre></details><details><summary>Open questions · {openQuestionCount}</summary><pre data-testid="open-questions-content">{openQuestions || "No open questions yet."}</pre></details></div>
      </aside>
    </div>
  );
}
