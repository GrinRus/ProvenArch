import { ModalDialog } from "./ModalDialog";
import { AskStagePanel } from "./StagePanels";
import type { QAProposalDraftResponse } from "../lib/qaApi";
import type { GitDiffResponse } from "../lib/appContracts";

type GitConfirmation = {
  action: "commit" | "branch";
  diff: GitDiffResponse;
};

type AppOverlaysProps = {
  askOpen: boolean;
  gitConfirmation: GitConfirmation | null;
  briefSkipConfirmationOpen: boolean;
  busy: boolean;
  onCloseAsk: () => void;
  onOpenAskCitation: (path: string) => void;
  onProposalCreated: (proposal: QAProposalDraftResponse) => void;
  onCancelGitAction: () => void;
  onConfirmGitAction: () => void;
  onCancelBriefSkip: () => void;
  onConfirmBriefSkip: () => void;
};

/** Global dialogs kept outside the route renderer so page composition stays task-focused. */
export function AppOverlays({
  askOpen,
  gitConfirmation,
  briefSkipConfirmationOpen,
  busy,
  onCloseAsk,
  onOpenAskCitation,
  onProposalCreated,
  onCancelGitAction,
  onConfirmGitAction,
  onCancelBriefSkip,
  onConfirmBriefSkip,
}: AppOverlaysProps) {
  return (
    <>
      <ModalDialog
        open={askOpen}
        title="Ask current workspace"
        description="Current workspace · read-only. Q&A execution and history do not alter Change Review or Publish acceptance."
        onCancel={onCloseAsk}
      >
        <AskStagePanel onOpenArtifact={onOpenAskCitation} onProposalCreated={onProposalCreated} />
      </ModalDialog>

      <ModalDialog
        open={gitConfirmation !== null}
        title={gitConfirmation?.action === "branch" ? "Confirm proposal branch" : "Confirm workspace commit"}
        description="This action uses the complete workspace Git inventory shown below. If branch, HEAD, or any file changes before confirmation, ACP will reject it without a Git mutation."
        confirmLabel={gitConfirmation?.action === "branch" ? "Create proposal branch" : "Commit all workspace changes"}
        busy={busy}
        onCancel={onCancelGitAction}
        onConfirm={onConfirmGitAction}
      >
        {gitConfirmation ? <GitConfirmationInventory diff={gitConfirmation.diff} /> : null}
      </ModalDialog>

      <ModalDialog
        open={briefSkipConfirmationOpen}
        title="Start without a saved analysis brief?"
        description="The run can proceed, but missing project name and scope usually reduces evidence quality and actionability."
        confirmLabel="Start with quality warning"
        busy={busy}
        onCancel={onCancelBriefSkip}
        onConfirm={onConfirmBriefSkip}
      />

    </>
  );
}

function GitConfirmationInventory({ diff }: { diff: GitDiffResponse }) {
  return (
    <div className="git-confirmation" data-testid="git-confirmation-inventory">
      <dl className="compact-defs">
        <div><dt>Branch</dt><dd>{diff.branch}</dd></div>
        <div><dt>HEAD</dt><dd><code>{diff.head_oid ?? "unborn"}</code></dd></div>
        <div><dt>Base</dt><dd>{diff.base_ref} · <code>{diff.base_oid ?? "unborn"}</code></dd></div>
        <div><dt>Fingerprint</dt><dd><code>{diff.fingerprint}</code></dd></div>
      </dl>
      {diff.files.length === 0 ? <p>No workspace changes.</p> : (
        <ul>
          {diff.files.map((file) => (
            <li key={`${file.status}:${file.original_path ?? ""}:${file.path}`}>
              <strong>{file.status}</strong> <code>{file.path}</code>
              {file.original_path ? <span> from <code>{file.original_path}</code></span> : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
