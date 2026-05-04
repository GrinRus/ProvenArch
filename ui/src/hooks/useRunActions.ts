import { useCallback } from "react";
import type { Dispatch } from "react";

import type { RunListItem, RunStatusResponse } from "../lib/appContracts";
import { getPipelineRunStatus, listPipelineRuns, requestRunCancel, startPipelineRun } from "../lib/runApi";
import type { RunExplorerAction } from "../lib/runExplorerState";
import { activeStatuses, finalStatuses, pickBootstrapRun, reconcileSelectedRunID } from "../lib/runState";

type RunActionsContext = {
  dispatch: Dispatch<RunExplorerAction>;
  runId: string | null;
  runStatus: RunStatusResponse | null;
  selectedRunIsActive: boolean;
  runLogsEOF: boolean;
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
  setRunID: (runId: string | null) => void;
  setRunStatus: (runStatus: RunStatusResponse | null) => void;
  setRunList: (runList: RunListItem[]) => void;
  setRunActionStatus: (status: string) => void;
  setCancelBusy: (busy: boolean) => void;
  resetRunLogs: () => void;
  fetchRunLogs: (runId: string, reset?: boolean) => Promise<unknown>;
  fetchRunLogsUntilEOF: (runId: string) => Promise<void>;
  clearArtifacts: () => void;
  fetchArtifacts: (runId: string) => Promise<void>;
  loadCoverageArtifacts: () => Promise<void>;
};

export function useRunActions({
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
}: RunActionsContext) {
  const loadRunList = useCallback(
    async (limit = 100): Promise<RunListItem[]> => {
      const payload = await listPipelineRuns(limit);
      const items = payload.items ?? [];
      setRunList(items);
      return items;
    },
    [setRunList]
  );

  const fetchRunStatus = useCallback(
    async (id: string, allowMissing = false): Promise<RunStatusResponse | null> => {
      const typed = await getPipelineRunStatus(id, allowMissing);
      if (!typed) {
        return null;
      }
      setRunStatus(typed);
      if (finalStatuses.has(typed.status)) {
        await fetchArtifacts(id);
        await loadCoverageArtifacts();
      }
      return typed;
    },
    [fetchArtifacts, loadCoverageArtifacts, setRunStatus]
  );

  const handleSelectRun = useCallback(
    async (
      id: string,
      options?: {
        silentErrors?: boolean;
      }
    ) => {
      try {
        setRunActionStatus("");
        setRunID(id);
        resetRunLogs();
        clearArtifacts();
        const status = await fetchRunStatus(id);
        await fetchArtifacts(id);
        await fetchRunLogs(id, true);
        if (status && finalStatuses.has(status.status)) {
          await fetchRunLogsUntilEOF(id);
        }
      } catch (requestError) {
        if (!options?.silentErrors) {
          setError(requestError instanceof Error ? requestError.message : "failed to load run details");
        }
      }
    },
    [
      clearArtifacts,
      fetchArtifacts,
      fetchRunLogs,
      fetchRunLogsUntilEOF,
      fetchRunStatus,
      resetRunLogs,
      setError,
      setRunActionStatus,
      setRunID,
    ]
  );

  const bootstrapRuns = useCallback(async () => {
    let initialRuns: RunListItem[] = [];
    try {
      initialRuns = await loadRunList(100);
    } catch {
      setRunList([]);
    }

    const bootstrapRun = pickBootstrapRun(initialRuns);
    if (bootstrapRun) {
      await handleSelectRun(bootstrapRun.run_id, { silentErrors: true });
    }
  }, [handleSelectRun, loadRunList, setRunList]);

  const pollRunUpdates = useCallback(async () => {
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

  const handleRunPipeline = useCallback(
    async (pipeline: "init" | "refresh") => {
      setBusy(true);
      setError(null);
      setRunActionStatus("");
      clearArtifacts();
      resetRunLogs();
      try {
        const payload = await startPipelineRun(pipeline);
        dispatch({
          type: "upsertRunListItem",
          item: {
            run_id: payload.run_id,
            pipeline,
            status: "queued",
            started_at: new Date().toISOString(),
            finished_at: null,
            warnings: [],
            error_code: null,
            error: null,
          },
        });
        setRunID(payload.run_id);
        const status = await fetchRunStatus(payload.run_id);
        await fetchRunLogs(payload.run_id, true);
        if (status && finalStatuses.has(status.status)) {
          await fetchRunLogsUntilEOF(payload.run_id);
        }
        await loadRunList(100);
      } catch (requestError) {
        setError(requestError instanceof Error ? requestError.message : "failed to start pipeline");
      } finally {
        setBusy(false);
      }
    },
    [
      clearArtifacts,
      dispatch,
      fetchRunLogs,
      fetchRunLogsUntilEOF,
      fetchRunStatus,
      loadRunList,
      resetRunLogs,
      setBusy,
      setError,
      setRunActionStatus,
      setRunID,
    ]
  );

  const handleCancelSelectedRun = useCallback(async () => {
    if (!runId || !selectedRunIsActive) {
      return;
    }

    setCancelBusy(true);
    setError(null);
    setRunActionStatus("");
    try {
      const response = await requestRunCancel(runId);

      if (response.status === 202) {
        setRunActionStatus(`Cancel requested for ${runId}`);
        await loadRunList(100);
        const status = await fetchRunStatus(runId);
        if (status && finalStatuses.has(status.status)) {
          await fetchRunLogsUntilEOF(runId);
        } else {
          await fetchRunLogs(runId, false);
        }
        return;
      }

      if (response.status === 404) {
        const latestRuns = await loadRunList(100);
        const nextSelectedRunID = reconcileSelectedRunID(runId, latestRuns);
        if (nextSelectedRunID !== runId) {
          dispatch({ type: "clearRunStatusForRun", runId });
          resetRunLogs();
          clearArtifacts();
          if (nextSelectedRunID) {
            await handleSelectRun(nextSelectedRunID, { silentErrors: true });
            setRunActionStatus(`Selected run no longer exists; switched to ${nextSelectedRunID}.`);
          } else {
            setRunID(null);
            setRunActionStatus("Selected run no longer exists.");
          }
        } else {
          setRunActionStatus("Selected run no longer exists.");
        }
        return;
      }

      if (response.status === 409) {
        setRunActionStatus("Selected run is already terminal.");
        await loadRunList(100);
        const status = await fetchRunStatus(runId);
        if (status && finalStatuses.has(status.status)) {
          await fetchRunLogsUntilEOF(runId);
        }
        return;
      }

      throw new Error("failed to cancel selected run");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to cancel selected run");
    } finally {
      setCancelBusy(false);
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
    selectedRunIsActive,
    setCancelBusy,
    setError,
    setRunActionStatus,
    setRunID,
  ]);

  return {
    bootstrapRuns,
    loadRunList,
    pollRunUpdates,
    handleRunPipeline,
    fetchRunStatus,
    handleSelectRun,
    handleCancelSelectedRun,
  };
}
