import { useEffect, useRef, useState } from "react";

import { ArtifactPathButton, StatusBadge } from "../../components/ConsolePrimitives";
import { ModalDialog } from "../../components/ModalDialog";
import { useRequestGate, isAbortError } from "../../hooks/useRequestGate";
import {
  createQAProposalDraft,
  getQARun,
  listQARuns,
  startQAQuestion,
  type QAProposalDraftResponse,
  type QARunResponse,
} from "../../lib/qaApi";
import { formatTimestamp, runOutcomeLabel, runOutcomeTone } from "../../lib/runState";
import {
  buildProvisionalQARun,
  mergeQARunHistory,
  qaErrorMessage,
  qaRunProviderLabel,
} from "./qaUtils";
import { QAFailureRecovery } from "./QAFailureRecovery";

export function AskStagePanel({
  primaryActionSignal = 0,
  onOpenArtifact,
  onProposalCreated,
}: {
  primaryActionSignal?: number;
  onOpenArtifact: (path: string) => void;
  onProposalCreated?: (proposal: QAProposalDraftResponse) => void;
}) {
  const [question, setQuestion] = useState("");
  const [qaRun, setQARun] = useState<QARunResponse | null>(null);
  const [runHistory, setRunHistory] = useState<QARunResponse[]>([]);
  const [selectedRunID, setSelectedRunID] = useState<string | null>(null);
  const [historyStatus, setHistoryStatus] = useState("Loading Q&A history.");
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const [selectedLoading, setSelectedLoading] = useState(false);
  const [proposalConfirmationOpen, setProposalConfirmationOpen] = useState(false);
  const [proposalTitle, setProposalTitle] = useState("");
  const [proposalOperatorNote, setProposalOperatorNote] = useState("");
  const [proposalBusy, setProposalBusy] = useState(false);
  const historyRequest = useRequestGate("qa-history");
  const detailRequest = useRequestGate("qa-detail");
  const pollRequest = useRequestGate("qa-poll");
  const selectedRunIDRef = useRef<string | null>(null);
  const qaRunRef = useRef<QARunResponse | null>(null);
  const selectionSequenceRef = useRef(0);
  const qaRunActive = qaRun?.status === "queued" || qaRun?.status === "running";
  const citations = qaRun?.citations ?? [];
  const unresolved = qaRun?.unresolved ?? [];
  const confidence = typeof qaRun?.confidence === "number" ? Math.round(qaRun.confidence * 100) : 0;

  function claimQASelection(runID: string): number {
    selectionSequenceRef.current += 1;
    selectedRunIDRef.current = runID;
    setSelectedRunID(runID);
    return selectionSequenceRef.current;
  }

  function isCurrentQASelection(selectionVersion: number, runID: string): boolean {
    return selectionSequenceRef.current === selectionVersion && selectedRunIDRef.current === runID;
  }

  useEffect(() => {
    selectedRunIDRef.current = selectedRunID;
  }, [selectedRunID]);

  useEffect(() => {
    qaRunRef.current = qaRun;
  }, [qaRun]);

  useEffect(() => {
    const selectionVersion = selectionSequenceRef.current;
    const historyToken = historyRequest.begin("initial");
    async function loadHistory() {
      try {
        const history = await listQARuns(20, historyToken.signal);
        if (!historyRequest.isCurrent(historyToken)) {
          return;
        }
        const items = history.items ?? [];
        const currentRun = qaRunRef.current;
        const visibleItems = currentRun ? mergeQARunHistory(currentRun, items, "preserve") : items;
        setRunHistory(visibleItems);
        setHistoryStatus(visibleItems.length > 0 ? "" : "No Q&A runs yet.");
        if (items[0]?.run_id && selectedRunIDRef.current === null && selectionSequenceRef.current === selectionVersion) {
          const selectedVersion = claimQASelection(items[0].run_id);
          setQARun(items[0]);
          const detailToken = detailRequest.begin(items[0].run_id);
          try {
            const detail = await getQARun(items[0].run_id, detailToken.signal);
            if (detailRequest.isCurrent(detailToken) && isCurrentQASelection(selectedVersion, items[0].run_id)) {
              setQARun(detail);
              setRunHistory((current) => mergeQARunHistory(detail, current, "preserve"));
              setHistoryStatus("");
            }
          } catch (error) {
            if (!isAbortError(error) && detailRequest.isCurrent(detailToken) && isCurrentQASelection(selectedVersion, items[0].run_id)) {
              setStatus(error instanceof Error ? error.message : "Q&A run detail failed");
            }
          } finally {
            detailRequest.finish(detailToken);
          }
        }
      } catch (error) {
        if (!isAbortError(error) && historyRequest.isCurrent(historyToken)) {
          setHistoryStatus(error instanceof Error ? error.message : "Q&A history failed to load");
        }
      } finally {
        historyRequest.finish(historyToken);
      }
    }
    void loadHistory();
    return () => {
      historyRequest.abort();
      detailRequest.abort();
    };
  }, []);

  useEffect(() => {
    if (!qaRun?.run_id || !qaRunActive) {
      return;
    }
    const runID = qaRun.run_id;
    let canceled = false;
    const refresh = async () => {
      const token = pollRequest.begin(runID);
      try {
        const next = await getQARun(runID, token.signal);
        if (!canceled && pollRequest.isCurrent(token) && selectedRunIDRef.current === runID) {
          setQARun(next);
          setSelectedRunID(next.run_id);
          setRunHistory((current) => mergeQARunHistory(next, current, "preserve"));
          setHistoryStatus("");
          setStatus(next.status === "succeeded" ? "Q&A run completed." : next.status === "failed" ? "Q&A run failed." : "Q&A run is running.");
        }
      } catch (error) {
        if (!isAbortError(error) && !canceled && pollRequest.isCurrent(token) && selectedRunIDRef.current === runID) {
          setStatus(error instanceof Error ? error.message : "Q&A run polling failed");
        }
      } finally {
        pollRequest.finish(token);
      }
    };
    const interval = window.setInterval(() => void refresh(), 1000);
    return () => {
      canceled = true;
      pollRequest.abort();
      window.clearInterval(interval);
    };
  }, [qaRun?.run_id, qaRunActive]);

  async function refreshHistory() {
    const token = historyRequest.begin("manual");
    setHistoryStatus("Refreshing Q&A history.");
    try {
      const history = await listQARuns(20, token.signal);
      if (!historyRequest.isCurrent(token)) {
        return;
      }
      const items = history.items ?? [];
      const currentRun = qaRunRef.current;
      const mergedItems = currentRun ? mergeQARunHistory(currentRun, items, "preserve") : items;
      setRunHistory(mergedItems);
      setHistoryStatus(mergedItems.length > 0 ? "" : "No Q&A runs yet.");
    } catch (error) {
      if (!isAbortError(error) && historyRequest.isCurrent(token)) {
        setHistoryStatus(error instanceof Error ? error.message : "Q&A history failed to load");
      }
    } finally {
      historyRequest.finish(token);
    }
  }

  async function handleSelectRun(run: QARunResponse) {
    const selectionVersion = claimQASelection(run.run_id);
    const token = detailRequest.begin(run.run_id);
    setQARun(run);
    setSelectedLoading(true);
    setStatus("");
    try {
      const detail = await getQARun(run.run_id, token.signal);
      if (detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, run.run_id)) {
        setQARun(detail);
        setRunHistory((current) => mergeQARunHistory(detail, current, "preserve"));
        setHistoryStatus("");
        setStatus(detail.status === "succeeded" ? "Q&A run loaded." : detail.status === "failed" ? "Q&A run failed." : "Q&A run is running.");
      }
    } catch (error) {
      if (!isAbortError(error) && detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, run.run_id)) {
        setStatus(error instanceof Error ? error.message : "Q&A run detail failed");
      }
    } finally {
      if (detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, run.run_id)) {
        setSelectedLoading(false);
      }
      detailRequest.finish(token);
    }
  }

  async function reloadSelectedAnswer() {
    if (!qaRun?.run_id) return;
    await handleSelectRun(qaRun);
  }

  function openProposalConfirmation() {
    if (!qaRun?.answer_digest || qaRun.status !== "succeeded") return;
    setProposalTitle((qaRun.question || "Ask synthesis").trim());
    setProposalOperatorNote("");
    setStatus("");
    setProposalConfirmationOpen(true);
  }

  async function handleCreateProposalDraft() {
    if (!qaRun?.run_id || !qaRun.answer_digest) return;
    const title = proposalTitle.trim();
    if (!title) {
      setStatus("Proposal title is required.");
      return;
    }
    setProposalBusy(true);
    setStatus("Creating proposal draft from the immutable Ask answer.");
    try {
      const proposal = await createQAProposalDraft(qaRun.run_id, {
        title,
        expected_answer_digest: qaRun.answer_digest,
        operator_note: proposalOperatorNote.trim() || undefined,
      });
      setProposalConfirmationOpen(false);
      setStatus(`Proposal draft created at ${proposal.path}.`);
      onProposalCreated?.(proposal);
    } catch (error) {
      setStatus(error instanceof Error ? error.message : "Proposal draft creation failed");
    } finally {
      setProposalBusy(false);
    }
  }

  async function handleAsk() {
    const trimmed = question.trim();
    if (!trimmed) {
      setStatus("Question is required.");
      return;
    }
    await startQARun(trimmed);
  }

  async function handleRetryQA() {
    const retryQuestion = (qaRun?.question || question).trim();
    if (!retryQuestion) {
      setStatus("Original question is unavailable.");
      return;
    }
    setQuestion(retryQuestion);
    await startQARun(retryQuestion);
  }

  async function startQARun(trimmed: string) {
    setBusy(true);
    setStatus("Submitting Q&A run.");
    try {
      const started = await startQAQuestion(trimmed);
      const provisionalRun = buildProvisionalQARun(started, trimmed);
      const selectionVersion = claimQASelection(started.run_id);
      const token = detailRequest.begin(started.run_id);
      setQARun(provisionalRun);
      setRunHistory((current) => mergeQARunHistory(provisionalRun, current));
      setHistoryStatus("");
      setStatus(`Q&A run ${started.run_id} accepted; reconciling details.`);
      try {
        const detail = await getQARun(started.run_id, token.signal);
        if (detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, started.run_id)) {
          setQARun(detail);
          setRunHistory((current) => mergeQARunHistory(detail, current));
          setHistoryStatus("");
          if (detail.status === "succeeded") {
            setStatus("Q&A run completed.");
          } else if (detail.status === "failed") {
            setStatus("Q&A run failed.");
          } else {
            setStatus("Q&A run is running.");
          }
        }
      } catch (error) {
        if (!isAbortError(error) && detailRequest.isCurrent(token) && isCurrentQASelection(selectionVersion, started.run_id)) {
          setStatus(`Q&A run ${started.run_id} accepted; reconciling details failed: ${qaErrorMessage(error, "Q&A run detail temporarily unavailable")}`);
        }
      } finally {
        detailRequest.finish(token);
      }
    } catch (error) {
      if (!isAbortError(error)) {
        setStatus(error instanceof Error ? error.message : "Q&A request failed");
      }
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => {
    if (primaryActionSignal <= 0) {
      return;
    }
    void handleAsk();
  }, [primaryActionSignal]);

  return (
    <section className="panel stage-panel" data-testid="qa-panel">
      <div className="stage-header">
        <div>
          <h1>Ask</h1>
          <p className="hint">Ask agent-backed questions over existing workspace artifacts. Source repos and canonical outputs stay unchanged.</p>
        </div>
        <StatusBadge tone={runOutcomeTone(qaRun)}>{qaRunProviderLabel(qaRun)}</StatusBadge>
      </div>

      <div className="qa-workbench">
        <aside className="qa-run-history" data-testid="qa-run-history">
          <div className="panel-subheader">
            <div>
              <h2>Run history</h2>
              <p className="hint">Async Q&A over existing workspace evidence.</p>
            </div>
            <button type="button" className="link-button" onClick={() => void refreshHistory()}>
              Refresh
            </button>
          </div>
          {historyStatus ? <p className="hint">{historyStatus}</p> : null}
          {runHistory.length === 0 ? (
            <p className="empty-state">Ask the workspace to create the first read-only Q&A run.</p>
          ) : (
            <div className="qa-history-list" role="list">
              {runHistory.map((run) => (
                <div key={run.run_id} role="listitem">
                  <button
                    type="button"
                    className={`qa-history-row${selectedRunID === run.run_id ? " is-selected" : ""}`}
                    onClick={() => void handleSelectRun(run)}
                    aria-pressed={selectedRunID === run.run_id}
                  >
                    <span className="qa-history-question">{run.question || run.run_id}</span>
                    <span className="qa-history-meta">
                      <StatusBadge tone={runOutcomeTone(run)}>{runOutcomeLabel(run, "unknown")}</StatusBadge>
                      <span>{qaRunProviderLabel(run)}</span>
                    </span>
                    <span className="qa-history-time">{formatTimestamp(run.finished_at || run.started_at)}</span>
                  </button>
                </div>
              ))}
            </div>
          )}
        </aside>

        <div className="qa-answer-panel" data-testid="qa-answer-panel">
          <div className="qa-composer">
            <label htmlFor="qaQuestion">Architecture question</label>
            <textarea
              id="qaQuestion"
              value={question}
              onChange={(event) => setQuestion(event.target.value)}
              rows={3}
              placeholder="Ask about ownership, dependencies, findings, proposals, or coverage in this workspace."
              data-testid="qa-question-input"
            />
            <button type="button" onClick={handleAsk} disabled={busy || qaRunActive} data-testid="qa-ask-btn">
              {qaRunActive ? "Agent is answering" : "Ask workspace"}
            </button>
            {status ? <p className={qaRun?.status === "failed" ? "status err" : qaRun?.status === "succeeded" ? "status ok" : "status warn"}>{status}</p> : null}
          </div>

          {qaRun ? (
            <div className="run-summary qa-run-summary" data-testid="qa-run-status">
              <div>
                <p>
                  Run <code>{qaRun.run_id}</code> status: <strong>{runOutcomeLabel(qaRun, "unknown")}</strong>
                </p>
                <p>Runtime provider: {qaRunProviderLabel(qaRun)}</p>
              </div>
              <a className="link-button" href={`/api/pipeline/runs/${encodeURIComponent(qaRun.run_id)}/logs`} target="_blank" rel="noreferrer">
                Open run logs
              </a>
              {qaRun.error ? <p className="status err">{qaRun.error}</p> : null}
              {(qaRun.warnings ?? []).length > 0 ? <p className="status warn">Warnings: {(qaRun.warnings ?? []).join(", ")}</p> : null}
            </div>
          ) : null}

          <QAFailureRecovery qaRun={qaRun} busy={busy || qaRunActive} onRetry={() => void handleRetryQA()} />

          {qaRun ? (
            <div className="qa-answer" data-testid="qa-answer">
              <div className="panel-subheader">
                <div>
                  <h2>Answer</h2>
                  <p className="hint">{selectedLoading ? "Loading selected Q&A run." : qaRun.generated_at ? `Generated ${formatTimestamp(qaRun.generated_at)}` : "Awaiting generated answer."}</p>
                </div>
                <StatusBadge tone={confidence >= 75 ? "ok" : confidence > 0 ? "warn" : "info"}>Confidence: {confidence}%</StatusBadge>
              </div>
              {qaRun.answer ? <p>{qaRun.answer}</p> : <p className="empty-state">No answer returned yet. Check run status and logs for details.</p>}
              {unresolved.length > 0 ? <p className="status warn">Unresolved: {unresolved.join(", ")}</p> : <p className="hint">No unresolved assumptions returned.</p>}
              {qaRun.status === "succeeded" && qaRun.answer_digest ? (
                <button type="button" onClick={openProposalConfirmation} data-testid="qa-create-proposal-btn">
                  Create proposal draft
                </button>
              ) : null}
              <div className="qa-related-partial">
                <h3>Related entities and edges</h3>
                <p className="hint">Partial state: the current QA API returns citations, not a structured related-entity graph. Use the citation trail for drilldown.</p>
              </div>
            </div>
          ) : (
            <div className="qa-answer qa-empty-answer">
              <h2>Answer</h2>
              <p className="empty-state">Select a historical run or ask a new question to review the answer, confidence and assumptions.</p>
            </div>
          )}
        </div>

        <aside className="qa-side-column">
          <section className="qa-readonly-safety-panel" data-testid="qa-readonly-safety-panel">
            <div className="panel-subheader">
              <div>
                <h2>Read-only runtime safety</h2>
                <p className="hint">Ask runs are audit-scoped and do not publish changes.</p>
              </div>
              <StatusBadge tone="ok">no canonical writes</StatusBadge>
            </div>
            <ul className="compact-list">
              <li>Source repositories stay read-only inputs.</li>
              <li>Canonical workspace outputs are not mutated by Q&A.</li>
              <li>Writes are limited to `reports/taskruns/&lt;run_id&gt;/qa/` audit artifacts.</li>
            </ul>
            {qaRun ? (
              <div className="actions qa-audit-actions">
                <button type="button" className="link-button" onClick={() => onOpenArtifact(`reports/taskruns/${qaRun.run_id}/qa/context-pack.json`)}>
                  context-pack.json
                </button>
                <button type="button" className="link-button" onClick={() => onOpenArtifact(`reports/taskruns/${qaRun.run_id}/qa/qa-answer.json`)}>
                  qa-answer.json
                </button>
                <button type="button" className="link-button" onClick={() => onOpenArtifact(`reports/taskruns/${qaRun.run_id}/qa/runtime-execution.json`)}>
                  runtime-execution.json
                </button>
              </div>
            ) : (
              <p className="hint">Audit links appear after selecting or starting a Q&A run.</p>
            )}
          </section>

          <section className="qa-citations-panel" data-testid="qa-citations-panel">
            <div className="panel-subheader">
              <div>
                <h2>Citations</h2>
                <p className="hint">Evidence used by the answer.</p>
              </div>
              <StatusBadge tone={citations.length > 0 ? "ok" : qaRun ? "warn" : "info"}>{citations.length} refs</StatusBadge>
            </div>
            {qaRun && citations.length === 0 ? (
              <p>No citations returned.</p>
            ) : citations.length > 0 ? (
              <ul className="citation-list">
                {citations.map((citation) => (
                  <li key={`${citation.path}-${citation.reason}`}>
                    <ArtifactPathButton path={citation.path} onOpenArtifact={onOpenArtifact} /> <span>{citation.reason}</span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="empty-state">No selected Q&A run yet.</p>
            )}
            <div className="qa-unresolved-box">
              <h3>Unresolved assumptions</h3>
              {unresolved.length > 0 ? <p className="status warn">Unresolved: {unresolved.join(", ")}</p> : <p className="hint">No unresolved assumptions returned.</p>}
            </div>
          </section>
        </aside>
      </div>
      <ModalDialog
        open={proposalConfirmationOpen}
        title="Create proposal draft"
        description="This explicit mutation creates a new current-workspace proposal package. The selected Ask taskrun and source repositories remain read-only."
        confirmLabel="Create proposal draft"
        busy={proposalBusy}
        onCancel={() => setProposalConfirmationOpen(false)}
        onConfirm={() => void handleCreateProposalDraft()}
      >
        <label htmlFor="qaProposalTitle">Proposal title</label>
        <input id="qaProposalTitle" value={proposalTitle} onChange={(event) => setProposalTitle(event.target.value)} autoFocus />
        <label htmlFor="qaProposalNote">Operator note (optional)</label>
        <textarea id="qaProposalNote" value={proposalOperatorNote} onChange={(event) => setProposalOperatorNote(event.target.value)} rows={3} />
        <p><strong>Target:</strong> `proposals/qa-synthesis-{qaRun?.run_id ?? "run"}-&lt;slug&gt;/`</p>
        <p><strong>Citations:</strong> {citations.length}</p>
        <p><strong>Unresolved assumptions:</strong> {unresolved.length > 0 ? unresolved.join(", ") : "none"}</p>
        <p className="hint">Ask remains read-only; only the new proposal package is written.</p>
        <button type="button" className="secondary" onClick={() => void reloadSelectedAnswer()} disabled={proposalBusy}>
          Reload selected answer
        </button>
      </ModalDialog>
    </section>
  );
}
