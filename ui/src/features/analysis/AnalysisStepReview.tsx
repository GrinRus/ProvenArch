import { StatusBadge } from "../../components/ConsolePrimitives";
import { GitDiffView } from "../../components/GitDiffView";
import { TabNav, tabPanelProps } from "../../components/TabNav";
import type { GitDiffResponse, RunLogEntry, RunReviewStep } from "../../lib/appContracts";
import type { LoadGitDiffOptions } from "../../lib/gitDiffApi";
import { providerDisplayLabel } from "../../lib/runtimeDisplay";
import { capitalize, selectedStepTone, stepMatches } from "./analysisViewModels";

export function AnalysisStepReview({
  steps,
  selectedStep,
  runtimeMode,
  runReviewStatus,
  runLogs,
  gitDiff,
  gitDiffStatus,
  view,
  onViewChange,
  onSelectStep,
  onOpenArtifact,
  onLoadGitDiff,
}: {
  steps: RunReviewStep[];
  selectedStep: RunReviewStep | null;
  runtimeMode: string;
  runReviewStatus: string;
  runLogs: RunLogEntry[];
  gitDiff: GitDiffResponse | null;
  gitDiffStatus: string;
  view: "artifacts" | "logs" | "evidence" | "diff";
  onViewChange: (view: "artifacts" | "logs" | "evidence" | "diff") => void;
  onSelectStep: (stepID: string) => void;
  onOpenArtifact: (path: string) => void;
  onLoadGitDiff: (options: LoadGitDiffOptions) => void;
}) {
  const stepLogs = selectedStep ? runLogs.filter((entry) => stepMatches(entry.step_id || "", selectedStep.step_id) || stepMatches(entry.taskrun_path || "", selectedStep.key)) : [];
  return (
    <section className="analysis-step-review" data-testid="analysis-step-review-panel">
      <div className="section-heading-row">
        <div>
          <h2>Step review</h2>
          <p className="hint">Review step-level artifacts, logs, evidence and workspace diff without waiting for final publication.</p>
        </div>
        <StatusBadge tone={selectedStepTone(selectedStep?.state)}>{selectedStep?.state ?? "empty"}</StatusBadge>
      </div>
      {runReviewStatus ? <p className="status warn">{runReviewStatus}</p> : null}
      {steps.length === 0 ? (
        <p className="empty-state">No review summary is available for the selected run yet. Status and logs still appear below.</p>
      ) : (
        <>
          <div className="analysis-step-card-grid">
            {steps.map((step, index) => (
              <button
                type="button"
                key={step.step_id}
                className={`analysis-step-review-card ${step.state}${selectedStep?.step_id === step.step_id ? " is-selected" : ""}`}
                data-testid="analysis-step-review-card"
                onClick={() => onSelectStep(step.step_id)}
                aria-pressed={selectedStep?.step_id === step.step_id}
              >
                <span className="step-index">{index}</span>
                <strong>{step.label}</strong>
                <code>{step.step_id}</code>
                <span>{providerDisplayLabel(runtimeMode, step.provider)}</span>
                <span>
				  {step.artifact_count ?? step.artifact_paths?.length ?? 0} artifacts · {step.warnings_count ?? 0}/{step.errors_count ?? 0} warn/error
                </span>
              </button>
            ))}
          </div>

          <TabNav
            ariaLabel="Step review tabs"
            className="step-review-tabs"
            idBase="analysis-step-tabs"
            testId="analysis-step-tabs"
            value={view}
            onChange={(tab) => {
              onViewChange(tab);
              if (tab === "diff" && selectedStep?.step_id) {
                onLoadGitDiff({ stepId: selectedStep.step_id });
              }
            }}
            options={(["artifacts", "logs", "evidence", "diff"] as const).map((tab) => ({ id: tab, label: capitalize(tab), testId: `analysis-step-tab-${tab}` }))}
          />

          <div className="step-review-body" {...tabPanelProps("analysis-step-tabs", view)}>
            {view === "artifacts" ? (
			  selectedStep && (selectedStep.artifact_paths?.length ?? 0) > 0 ? (
                <ul className="compact-list">
				  {(selectedStep.artifact_paths ?? []).map((path) => (
                    <li key={path}>
                      <button type="button" className="link-button" onClick={() => onOpenArtifact(path)}>
                        {path}
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="empty-state">No artifacts have been attached to this step yet. Logs may still be streaming.</p>
              )
            ) : null}

            {view === "logs" ? (
              stepLogs.length > 0 ? (
                <ul className="compact-list">
                  {stepLogs.slice(-8).map((entry) => (
                    <li key={entry.cursor}>
                      <span>
                        {entry.level.toUpperCase()} · {entry.step_id || selectedStep?.step_id}
                      </span>
                      <code>{entry.message}</code>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="empty-state">{selectedStep?.state === "active" ? "Provider is silent; still waiting for logs." : "No logs are available for this step."}</p>
              )
            ) : null}

            {view === "evidence" ? (
              <div className="step-evidence-summary">
                <dl className="compact-defs">
                  <div>
                    <dt>Last message</dt>
                    <dd>{selectedStep?.last_message || "No step message yet."}</dd>
                  </div>
                  <div>
                    <dt>Taskrun refs</dt>
					<dd>{selectedStep?.taskrun_paths?.join(", ") || "No taskrun refs."}</dd>
                  </div>
                </dl>
              </div>
            ) : null}

            {view === "diff" ? <GitDiffView gitDiff={gitDiff} status={gitDiffStatus} onSelectFile={(path) => onLoadGitDiff({ path, stepId: selectedStep?.step_id })} /> : null}
          </div>
        </>
      )}
    </section>
  );
}


