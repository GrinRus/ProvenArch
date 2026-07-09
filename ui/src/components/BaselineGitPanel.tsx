type BaselineGitPanelProps = {
  busy: boolean;
  gitMessage: string;
  proposalBranch: string;
  gitStatus: string;
  gitError: string;
  onGitMessageChange: (value: string) => void;
  onProposalBranchChange: (value: string) => void;
  onCommit: () => void;
  onCreateProposalBranch: () => void;
};

export function BaselineGitPanel(props: BaselineGitPanelProps) {
  const {
    busy,
    gitMessage,
    proposalBranch,
    gitStatus,
    gitError,
    onGitMessageChange,
    onProposalBranchChange,
    onCommit,
    onCreateProposalBranch,
  } = props;

  return (
    <section className="panel" data-testid="baseline-git-helper-panel">
      <h2>Baseline: Git Helper Actions</h2>
      <label htmlFor="gitMessage">Commit message</label>
      <input id="gitMessage" value={gitMessage} onChange={(event) => onGitMessageChange(event.target.value)} />
      <button type="button" onClick={onCommit} disabled={busy} data-testid="git-commit-btn">
        Commit workspace changes
      </button>

      <label htmlFor="proposalBranch">Proposal branch</label>
      <input id="proposalBranch" value={proposalBranch} onChange={(event) => onProposalBranchChange(event.target.value)} />
      <button type="button" onClick={onCreateProposalBranch} disabled={busy} data-testid="git-proposal-branch-btn">
        Create/Switch proposal branch
      </button>

      {gitStatus ? <p className="status ok">{gitStatus}</p> : null}
      {gitError ? <p className="status err">{gitError}</p> : null}
    </section>
  );
}
