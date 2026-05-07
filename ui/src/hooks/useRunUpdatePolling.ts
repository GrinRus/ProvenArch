import { useCallback } from "react";
import type { Dispatch } from "react";

import type { RunListItem, RunStatusResponse } from "../lib/appContracts";
import type { RunExplorerAction } from "../lib/runExplorerState";
import { activeStatuses, reconcileSelectedRunID } from "../lib/runState";

type UseRunUpdatePollingOptions = {
  clearArtifacts: () => void;
  dispatch: Dispatch<RunExplorerAction>;
  fetchRunLogs: (runId: string, reset?: boolean) => Promise<unknown>;
  fetchRunLogsUntilEOF: (runId: string) => Promise<void>;
  fetchRunStatus: (id: string, allowMissing?: boolean) => Promise<RunStatusResponse | null>;
  handleSelectRun: (id: string, options?: { silentErrors?: boolean }) => Promise<void>;
  loadRunList: (limit?: number) => Promise<RunListItem[]>;
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
  return useCallback(async () => {
    try {
      const latestRuns = await loadRunList(100);
      if (!runId) {
        return;
      }
      const nextSelectedRunID = reconcileSelectedRunID(runId, latestRuns);
      if (nextSelectedRunID !== runId) {
        if (nextSelectedRunID && latestRuns.length > 0) {
          dispatch({ type: "clearRunStatusForRun", runId });
          resetRunLogs();
          clearArtifacts();
          await handleSelectRun(nextSelectedRunID, { silentErrors: true });
          setRunActionStatus(`Selected run no longer exists; switched to ${nextSelectedRunID}.`);
          return;
        }
        const previousStatus = runStatus?.status ?? null;
        const currentStatus = await fetchRunStatus(runId, true);
        if (currentStatus) {
          if (activeStatuses.has(currentStatus.status)) {
            await fetchRunLogs(runId, false);
            return;
          }
          const statusChanged = previousStatus !== currentStatus.status;
          if (statusChanged || !runLogsEOF) {
            await fetchRunLogsUntilEOF(runId);
          }
          return;
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
        return;
      }
      const previousStatus = runStatus?.status ?? null;
      const status = await fetchRunStatus(runId);
      if (!status) {
        return;
      }
      if (activeStatuses.has(status.status)) {
        await fetchRunLogs(runId, false);
        return;
      }
      const statusChanged = previousStatus !== status.status;
      if (statusChanged || !runLogsEOF) {
        await fetchRunLogsUntilEOF(runId);
      }
    } catch {
      // keep UI responsive even if polling fails temporarily
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
