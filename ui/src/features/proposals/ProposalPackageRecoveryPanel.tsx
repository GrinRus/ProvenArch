import { StatusBadge } from "../../components/ConsolePrimitives";
import type { Artifact } from "../../lib/appContracts";
import { proposalPackageSuggestedFix, type ProposalReviewModel } from "./proposalUtils";

export function ProposalPackageRecoveryPanel({
  proposalReview,
  preferredArtifact,
  proposalBranch,
  gitStatus,
  onOpenArtifact,
  onGoPublish,
}: {
  proposalReview: ProposalReviewModel;
  preferredArtifact: Artifact | undefined;
  proposalBranch: string;
  gitStatus: string;
  onOpenArtifact: (path: string) => void;
  onGoPublish: () => void;
}) {
  const primaryBlocker = proposalReview.blockers[0] ?? "No proposal package blocker detected.";
  const suggestedFix = proposalPackageSuggestedFix(primaryBlocker);
  const packageState = proposalReview.proposalDocumentArtifacts.length > 0 ? `${proposalReview.packages.length} artifact groups` : "proposal missing";
  const publicationPath =
    proposalReview.blockers.length > 0
      ? "Keep Publish as review-only until proposal, changelog and evidence blockers are resolved."
      : proposalBranch
        ? `Ready for Publish review on ${proposalBranch}.`
        : "Ready for Publish review; prepare a proposal branch before handoff.";

  return (
    <section className="proposal-recovery-panel" data-testid="proposal-package-recovery">
      <div className="section-heading-row">
        <div>
          <h2>Proposal package recovery</h2>
          <p className="hint">Resolve proposal/changelog gaps before treating this run as publication-ready.</p>
        </div>
        <StatusBadge tone="warn">proposal blocked</StatusBadge>
      </div>
      <div className="proposal-recovery-grid">
        <div><span className="meta-label">Package state</span><strong>{packageState}</strong></div>
        <div><span className="meta-label">Proposal docs</span><strong>{proposalReview.proposalDocumentCount}</strong></div>
        <div><span className="meta-label">ADR/RFC</span><strong>{proposalReview.adrRfcCount}</strong></div>
        <div><span className="meta-label">Changelog</span><strong>{proposalReview.changelogArtifacts.length}</strong></div>
        <div><span className="meta-label">Evidence refs</span><strong>{proposalReview.evidenceArtifacts.length}</strong></div>
      </div>
      <dl className="compact-defs proposal-recovery-detail">
        <div><dt>Primary blocker</dt><dd>{primaryBlocker}</dd></div>
        <div><dt>Suggested fix</dt><dd>{suggestedFix}</dd></div>
        <div><dt>Publication path</dt><dd>{publicationPath}{gitStatus ? ` ${gitStatus}` : ""}</dd></div>
      </dl>
      <div className="actions proposal-recovery-actions">
        {preferredArtifact ? <button type="button" className="secondary" onClick={() => onOpenArtifact(preferredArtifact.path)}>Open available artifact</button> : null}
        <button type="button" className="secondary" onClick={onGoPublish}>Check Publish gate</button>
      </div>
    </section>
  );
}
