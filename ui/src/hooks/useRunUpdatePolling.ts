import { useCallback, useRef } from "react";
import type { Dispatch } from "react";

import type { RunListItem, RunStatusResponse } from "../lib/appContracts";
import type { RunExplorerAction } from "../lib/runExplorerState";
import { activeStatuses, reconcileSelectedRunID } from "../lib/runState";

type UseRunUpdatePollingOptions = {
  clearArtifacts: () => void;
  dispatch: Dispatch<RunExplorerAction>;
  fetchRunLogs: (runId: string, reset?: boolean, signal?: AbortSignal) => Promise<unknown>;
  fetchRunLogsUntilEOF: (runId: string, signal?: AbortSignal) => Promise<void>;
  fetchRunStatus: (id: string, allowMissing?: boolean, selectionIsCurrent?: () => boolean, signal?: AbortSignal) => Promise<RunStatusResponse | null>;
  handleSelectRun: (id: string, options?: { silentErrors?: boolean }) => Promise<void>;
  loadRunList: (limit?: number, signal?: AbortSignal) => Promise<RunListItem[]>;
  resetRunLogs: () => void;
  runId: string | null;
  runLogsEOF: boolean;
  runStatus: RunStatusResponse | null;
  setRunActionStatus: (status: string) => void;
  setRunID: (runId: string | null) => void;
};

export function useRunUpdatePolling({
  clearArtifacts,
  dispatch,
  fetchRunLogs,
  fetchRunLogsUntilEOF,
  fetchRunStatus,
  handleSelectRun,
  loadRunList,
  resetRunLogs,
  runId,
  runLogsEOF,
  runStatus,
  setRunActionStatus,
  setRunID,
}: UseRunUpdatePollingOptions) {
  const currentRunIdRef = useRef(runId);
  currentRunIdRef.current = runId;

  return useCallback(async (signal?: AbortSignal): Promise<boolean> => {
    try {
      const latestRuns = await loadRunList(100, signal);
      if (signal?.aborted || !runId || currentRunIdRef.current !== runId) {
        return true;
      }
      const nextSelectedRunID = reconcileSelectedRunID(runId, latestRuns);
      if (nextSelectedRunID !== runId) {
        if (nextSelectedRunID && latestRuns.length > 0) {
          dispatch({ type: "clearRunStatusForRun", runId });
          resetRunLogs();
          clearArtifacts();
          await handleSelectRun(nextSelectedRunID, { silentErrors: true });
          setRunActionStatus(`Selected run no longer exists; switched to ${nextSelectedRunID}.`);
          return true;
        }
        const previousStatus = runStatus?.status ?? null;
        const currentStatus = await fetchRunStatus(runId, true, undefined, signal);
        if (signal?.aborted) return true;
        if (currentStatus) {
          if (activeStatuses.has(currentStatus.status)) {
            await fetchRunLogs(runId, false, signal);
            return true;
          }
          const statusChanged = previousStatus !== currentStatus.status;
          if (statusChanged || !runLogsEOF) {
            await fetchRunLogsUntilEOF(runId, signal);
          }
          return true;
        }
        dispatch({ type: "clearRunStatusForRun", runId });
        resetRunLogs();
        clearArtifacts();
        if (nextSelectedRunID) {
          await handleSelectRun(nextSelectedRunID, { silentErrors: true });
          setRunActionStatus(`Selected run no longer exists; switched to ${nextSelectedRunID}.`);
        } else {
          setRunID(null);
        }
        return true;
      }
      const previousStatus = runStatus?.status ?? null;
      const status = await fetchRunStatus(runId, false, undefined, signal);
      if (signal?.aborted) return true;
      if (!status) {
        return true;
      }
      if (activeStatuses.has(status.status)) {
        await fetchRunLogs(runId, false, signal);
        return true;
      }
      const statusChanged = previousStatus !== status.status;
      if (statusChanged || !runLogsEOF) {
        await fetchRunLogsUntilEOF(runId, signal);
      }
      return true;
    } catch {
      // keep UI responsive even if polling fails temporarily
      return false;
    }
  }, [
    clearArtifacts,
    dispatch,
    fetchRunLogs,
    fetchRunLogsUntilEOF,
    fetchRunStatus,
    handleSelectRun,
    loadRunList,
    resetRunLogs,
    runId,
    runLogsEOF,
    runStatus?.status,
    setRunActionStatus,
    setRunID,
  ]);
}
