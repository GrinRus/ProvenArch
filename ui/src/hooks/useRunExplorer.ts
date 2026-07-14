import { useCallback, useMemo, useReducer } from "react";

import { initialRunExplorerState, runExplorerReducer } from "../lib/runExplorerState";
import type { RunListItem, RunStatusResponse } from "../lib/appContracts";
import { useRunActions } from "./useRunActions";
import { useRunArtifacts } from "./useRunArtifacts";
import { useRunLogs } from "./useRunLogs";
import { useRunPolling } from "./useRunPolling";
import { useRunReview } from "./useRunReview";
import { useRunSelection } from "./useRunSelection";
import { useGitDiff } from "./useGitDiff";

type UseRunExplorerOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useRunExplorer({ setBusy, setError }: UseRunExplorerOptions) {
  const [state, dispatch] = useReducer(runExplorerReducer, initialRunExplorerState);
  const { runId, runStatus, runList, runActionStatus, cancelBusy } = state;
  const artifactsState = useRunArtifacts();
  const logsState = useRunLogs({ runId });
  const gitDiffState = useGitDiff();

  const {
    artifacts,
    selectedArtifact,
    selectedArtifactContent,
    coverageSummary,
    openQuestions,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedArtifactIsMermaid,
    fetchArtifacts,
    loadCoverageArtifacts,
    handleOpenArtifact,
    clearArtifacts,
  } = artifactsState;
  const {
    runLogs,
    runLogsCursor,
    runLogsEOF,
    runLogsStatus,
    runLogsViewMode,
    runLogsMode,
    runLogTaskrunPaths,
    filteredRunLogs,
    runLogsRendered,
    setRunLogsViewMode,
    setRunLogsMode,
    resetRunLogs,
    fetchRunLogs,
    fetchRunLogsUntilEOF,
    handleCopyRunLogs,
    handleDownloadRunLogs,
  } = logsState;
  const reviewPollSignal = useMemo(
    () => [runId ?? "", runStatus?.status ?? "", runStatus?.current_step ?? "", runLogs.length, artifacts.length].join("|"),
    [artifacts.length, runId, runLogs.length, runStatus?.current_step, runStatus?.status],
  );
  const { runReviewSummary, runReviewStatus, fetchRunReviewSummary } = useRunReview({
    runId,
    pollSignal: reviewPollSignal,
  });
  const { gitDiff, gitDiffStatus, loadGitDiff } = gitDiffState;

  const setRunID = useCallback((nextRunID: string | null) => dispatch({ type: "setRunID", runId: nextRunID }), []);
  const setRunStatus = useCallback(
    (nextRunStatus: RunStatusResponse | null) =>
      dispatch({ type: "setRunStatus", runStatus: nextRunStatus }),
    []
  );
  const setRunList = useCallback(
    (nextRunList: RunListItem[]) => dispatch({ type: "setRunList", runList: nextRunList }),
    []
  );
  const setRunActionStatus = useCallback(
    (nextRunActionStatus: string) =>
      dispatch({ type: "setRunActionStatus", runActionStatus: nextRunActionStatus }),
    []
  );
  const setCancelBusy = useCallback(
    (nextCancelBusy: boolean) => dispatch({ type: "setCancelBusy", cancelBusy: nextCancelBusy }),
    []
  );

  const {
    hasActiveRuns,
    runCounters,
    selectedRunListItem,
    selectedRunWarnings,
    selectedRunIsActive,
    shouldPollRunDetails,
  } = useRunSelection({ runId, runStatus, runList, runLogsEOF });

  const {
    bootstrapRuns,
    pollRunUpdates,
    handleRunPipeline,
    handleSelectRun,
    handleCancelSelectedRun,
  } = useRunActions({
    dispatch,
    runId,
    runStatus,
    selectedRunIsActive,
    runLogsEOF,
    setBusy,
    setError,
    setRunID,
    setRunStatus,
    setRunList,
    setRunActionStatus,
    setCancelBusy,
    resetRunLogs,
    fetchRunLogs,
    fetchRunLogsUntilEOF,
    clearArtifacts,
    fetchArtifacts,
    loadCoverageArtifacts,
  });

  useRunPolling({
    shouldPollRunDetails,
    runId,
    runLogsCursor,
    runLogsEOF,
    pollRunUpdates,
  });

  return {
    runId,
    setRunID,
    runStatus,
    runList,
    artifacts,
    selectedArtifact,
    selectedArtifactContent,
    runLogsStatus,
    runLogsViewMode,
    setRunLogsViewMode,
    runLogsMode,
    setRunLogsMode,
    runActionStatus,
    cancelBusy,
    coverageSummary,
    openQuestions,
    hasActiveRuns,
    runCounters,
    runLogTaskrunPaths,
    runLogs,
    filteredRunLogs,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedArtifactIsMermaid,
    selectedRunListItem,
    selectedRunWarnings,
    selectedRunIsActive,
    runLogsRendered,
    runReviewSummary,
    runReviewStatus,
    gitDiff,
    gitDiffStatus,
    bootstrapRuns,
    fetchRunReviewSummary,
    loadGitDiff,
    handleRunPipeline,
    handleSelectRun,
    handleCancelSelectedRun,
    handleOpenArtifact,
    handleCopyRunLogs,
    handleDownloadRunLogs,
  };
}
