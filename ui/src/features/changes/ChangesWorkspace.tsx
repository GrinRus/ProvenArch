import type { ComponentProps } from "react";

import { ChangesPage } from "../../components/ChangesPage";
import { EvidenceViewer } from "../../components/EvidenceViewer";
import { ProposalsStagePanel, PublishStagePanel, ReviewStagePanel } from "../../components/StagePanels";
import type { ChangesView, RouteSource, ViewerMode } from "../../lib/appRoutes";
import { buildChangesRouteModel } from "./viewModels";

type Props = {
  view: ChangesView;
  source: RouteSource;
  page: Omit<ComponentProps<typeof ChangesPage>, "children" | "view">;
  review: ComponentProps<typeof ReviewStagePanel>;
  proposals: ComponentProps<typeof ProposalsStagePanel>;
  publish: ComponentProps<typeof PublishStagePanel>;
  currentArtifact: { path: string; content: string } | null;
  viewerMode: ViewerMode;
  askReturnAvailable: boolean;
  onViewerModeChange: (mode: ViewerMode) => void;
  onOpenCurrentArtifact: (path: string) => void;
  onReturnToAsk: () => void;
};

export function ChangesWorkspace({ view, source, page, review, proposals, publish, currentArtifact, viewerMode, askReturnAvailable, onViewerModeChange, onOpenCurrentArtifact, onReturnToAsk }: Props) {
  const model = buildChangesRouteModel(view, source);
  const gitState = source === "current"
    ? "read-only"
    : view === "proposals"
    ? proposals.gitDiff?.state ?? "unknown"
    : view === "publish"
      ? publish.gitDiff?.state ?? "unknown"
      : review.gitDiff?.state ?? "unknown";
  let content = null;
  if (view === "proposals") content = <ProposalsStagePanel {...proposals} />;
  else if (view === "publish") content = <PublishStagePanel {...publish} />;
  else if (source === "current") {
    content = (
      <section className="panel stage-panel current-evidence" data-testid="current-workspace-evidence">
        {currentArtifact
          ? <EvidenceViewer path={currentArtifact.path} content={currentArtifact.content} sourceMode="promoted_current" mode={viewerMode} onModeChange={onViewerModeChange} onOpenArtifact={onOpenCurrentArtifact} />
          : <p className="empty-state">Choose a current workspace artifact. No historical run snapshot will be substituted.</p>}
      </section>
    );
  } else content = review.runId || review.selectedArtifact || view === "evidence" ? <ReviewStagePanel {...review} routeView={view} /> : <NoTaskReviewSelection />;

  const showGitState = source === "current"
    || view === "proposals"
    || view === "publish"
    || Boolean(review.runId || review.selectedArtifact)
    || gitState !== "unknown";

  if (source === "current" && view !== "evidence") {
    content = (
      <section className="panel stage-panel current-evidence" data-testid="current-workspace-publish-blocked">
        <h2>Current workspace is read-only</h2>
        <p className="empty-state">This view cannot show historical review or publication data. Select Evidence to inspect the current workspace artifact, or choose a historical run from Changes.</p>
      </section>
    );
  }

  return (
    <ChangesPage {...page} sourceMode={source} view={view}>
      <section className={`changes-route-view changes-route-${model.kind}`} data-testid={`changes-route-${model.kind}`}>
        <header className="changes-route-heading">
          <div>
            <p className="hint">{model.purpose}</p>
          </div>
          {showGitState ? <span className={`status git-state-${gitState}`} data-testid="changes-git-state">Git: {gitState}</span> : null}
          {askReturnAvailable ? <button type="button" className="link-button" onClick={onReturnToAsk}>Return to Ask</button> : null}
        </header>
        {content}
      </section>
    </ChangesPage>
  );
}

function NoTaskReviewSelection() {
  return <section className="panel stage-panel changes-no-selection" data-testid="changes-no-selection"><p className="eyebrow">Task review</p><h2>Select a Task to inspect its evidence</h2><p className="empty-state">Choose a completed Task above. ProvenArch keeps the review pinned to that Task and never falls back to the latest run.</p></section>;
}
