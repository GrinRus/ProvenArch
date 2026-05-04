import { useCallback, useReducer } from "react";

import { initialRunExplorerState, runExplorerReducer } from "../lib/runExplorerState";
import type { RunListItem, RunStatusResponse } from "../lib/appContracts";
import { useRunActions } from "./useRunActions";
import { useRunArtifacts } from "./useRunArtifacts";
import { useRunLogs } from "./useRunLogs";
import { useRunPolling } from "./useRunPolling";
import { useRunSelection } from "./useRunSelection";

type UseRunExplorerOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useRunExplorer({ setBusy, setError }: UseRunExplorerOptions) {
  const [state, dispatch] = useReducer(runExplorerReducer, initialRunExplorerState);
  const { runId, runStatus, runList, runActionStatus, cancelBusy } = state;
  const artifactsState = useRunArtifacts();
  const logsState = useRunLogs({ runId });

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
    filteredRunLogs,
    diagramArtifacts,
    nonDiagramArtifacts,
    selectedArtifactIsMermaid,
    selectedRunListItem,
    selectedRunWarnings,
    selectedRunIsActive,
    runLogsRendered,
    bootstrapRuns,
    handleRunPipeline,
    handleSelectRun,
    handleCancelSelectedRun,
    handleOpenArtifact,
    handleCopyRunLogs,
    handleDownloadRunLogs,
  };
}
