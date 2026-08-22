import { useEffect, useRef } from "react";
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
  const askInvokerRef = useRef<HTMLElement | null>(null);
  const askCloseRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    if (askOpen) {
      askInvokerRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      askCloseRef.current?.focus();
      return;
    }
    const invoker = askInvokerRef.current;
    askInvokerRef.current = null;
    if (invoker && document.contains(invoker)) {
      invoker.focus();
    }
  }, [askOpen]);

  useEffect(() => {
    if (!askOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onCloseAsk();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [askOpen, onCloseAsk]);
  return (
    <>
      {askOpen ? <aside className="ask-drawer" role="dialog" aria-modal="false" aria-label="Ask current workspace">
        <header><div><p className="eyebrow">Workspace Q&A</p><h2>Ask current workspace</h2><p>Read-only Q&amp;A. Answers cite the promoted workspace and never change Change Review or Publish acceptance.</p></div><button ref={askCloseRef} type="button" onClick={onCloseAsk} aria-label="Close Ask">Close</button></header>
        <AskStagePanel onOpenArtifact={onOpenAskCitation} onProposalCreated={onProposalCreated} />
      </aside> : null}

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
  const changedFiles = diff.files.length;
  const changedFolders = diff.folders.length;
  const additions = diff.files.reduce((total, file) => total + file.additions, 0);
  const deletions = diff.files.reduce((total, file) => total + file.deletions, 0);
  return (
    <div className="git-confirmation" data-testid="git-confirmation-inventory">
      {changedFiles === 0 ? <p>No workspace changes.</p> : (
        <>
          <div className="git-confirmation-summary">
            <strong>{changedFiles} changed file{changedFiles === 1 ? "" : "s"} across {changedFolders} folder{changedFolders === 1 ? "" : "s"}</strong>
            <span>{additions} additions · {deletions} deletions · complete workspace scope</span>
          </div>
          <ul className="git-confirmation-folders">
            {diff.folders.map((folder) => <li key={folder.folder}><strong>{folder.folder}</strong><span>{folder.files} file{folder.files === 1 ? "" : "s"} · +{folder.additions} / −{folder.deletions}</span></li>)}
          </ul>
          <details>
            <summary>Technical details</summary>
            <dl className="compact-defs">
              <div><dt>Branch</dt><dd>{diff.branch}</dd></div>
              <div><dt>HEAD</dt><dd><code>{diff.head_oid ?? "unborn"}</code></dd></div>
              <div><dt>Base</dt><dd>{diff.base_ref} · <code>{diff.base_oid ?? "unborn"}</code></dd></div>
              <div><dt>Fingerprint</dt><dd><code>{diff.fingerprint}</code></dd></div>
            </dl>
          </details>
          <details>
            <summary>All files</summary>
            <ul>
              {diff.files.map((file) => (
                <li key={`${file.status}:${file.original_path ?? ""}:${file.path}`}>
                  <strong>{file.status}</strong> <code>{file.path}</code>
                  {file.original_path ? <span> from <code>{file.original_path}</code></span> : null}
                </li>
              ))}
            </ul>
          </details>
        </>
      )}
    </div>
  );
}
