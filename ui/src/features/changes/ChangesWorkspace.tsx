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
  const gitState = view === "proposals"
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
  } else content = <ReviewStagePanel {...review} routeView={view} />;

  return (
    <ChangesPage {...page} sourceMode={source} view={view}>
      <section className={`changes-route-view changes-route-${model.kind}`} data-testid={`changes-route-${model.kind}`}>
        <header className="changes-route-heading">
          <div>
            <p className="hint">{model.purpose}</p>
          </div>
          <span className={`status git-state-${gitState}`} data-testid="changes-git-state">Git: {gitState}</span>
          {askReturnAvailable ? <button type="button" className="link-button" onClick={onReturnToAsk}>Return to Ask</button> : null}
        </header>
        {content}
      </section>
    </ChangesPage>
  );
}
