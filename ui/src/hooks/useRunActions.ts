import { useCallback } from "react";
import type { Dispatch } from "react";

import type { RunListItem, RunStatusResponse } from "../lib/appContracts";
import { getPipelineRunStatus, listPipelineRuns, requestRunCancel, startPipelineRun } from "../lib/runApi";
import type { RunExplorerAction } from "../lib/runExplorerState";
import { finalStatuses, pickBootstrapRun, reconcileSelectedRunID } from "../lib/runState";
import { isAbortError, useRequestGate } from "./useRequestGate";
import { useRunUpdatePolling } from "./useRunUpdatePolling";

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
}: RunActionsContext) {
  const runStatusRequest = useRequestGate("run-status");

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
      const token = runStatusRequest.begin(`${id}:${allowMissing ? "allow-missing" : "strict"}`);
      try {
        const typed = await getPipelineRunStatus(id, allowMissing, { signal: token.signal });
        if (!runStatusRequest.isCurrent(token)) {
          return null;
        }
        if (!typed) {
          return null;
        }
        setRunStatus(typed);
        if (finalStatuses.has(typed.status)) {
          await fetchArtifacts(id);
        }
        return typed;
      } catch (error) {
        if (isAbortError(error) || !runStatusRequest.isCurrent(token)) {
          return null;
        }
        throw error;
      } finally {
        runStatusRequest.finish(token);
      }
    },
    [fetchArtifacts, runStatusRequest, setRunStatus]
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
        setRunStatus(null);
        resetRunLogs();
        clearArtifacts();
        const status = await fetchRunStatus(id);
        if (!status || !finalStatuses.has(status.status)) {
          await fetchArtifacts(id);
        }
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
      setRunActionStatus(activeRunResumeMessage(bootstrapRun.status, bootstrapRun.run_id));
      return;
    }
    setRunActionStatus("No runs yet. Start the first analysis to create reviewable artifacts.");
  }, [handleSelectRun, loadRunList, setRunActionStatus, setRunList]);

  const pollRunUpdates = useRunUpdatePolling({
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
  });

  const handleRunPipeline = useCallback(
    async (pipeline: "init" | "refresh"): Promise<boolean> => {
      setBusy(true);
      setError(null);
      setRunActionStatus("");
      clearArtifacts();
      resetRunLogs();
      let acceptedRunID = "";
      try {
        const payload = await startPipelineRun(pipeline);
        acceptedRunID = payload.run_id;
        const provisionalRun = buildProvisionalRun(pipeline, payload);
        dispatch({
          type: "upsertRunListItem",
          item: provisionalRun,
        });
        setRunID(payload.run_id);
        setRunStatus(provisionalRun);
        setRunActionStatus(`Run ${payload.run_id} accepted; reconciling details.`);
        const status = await fetchRunStatus(payload.run_id);
        await fetchRunLogs(payload.run_id, true);
        if (status && finalStatuses.has(status.status)) {
          await fetchRunLogsUntilEOF(payload.run_id);
        }
        await loadRunList(100);
        setRunActionStatus("");
        return true;
      } catch (requestError) {
        if (acceptedRunID) {
          setRunActionStatus(`Run ${acceptedRunID} accepted; reconciling details failed: ${errorMessage(requestError, "run details are temporarily unavailable")}`);
          return true;
        }
        setError(requestError instanceof Error ? requestError.message : "failed to start pipeline");
        return false;
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
        try {
          await loadRunList(100);
          const status = await fetchRunStatus(runId);
          if (status && finalStatuses.has(status.status)) {
            await fetchRunLogsUntilEOF(runId);
          } else {
            await fetchRunLogs(runId, false);
          }
        } catch (requestError) {
          setRunActionStatus(`Cancel requested for ${runId}; reconciling details failed: ${errorMessage(requestError, "run details are temporarily unavailable")}`);
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

function activeRunResumeMessage(status: string, runID: string): string {
  if (status === "running" || status === "queued") {
    return `Resumed active run ${runID}.`;
  }
  return `Selected latest completed run ${runID}.`;
}

function buildProvisionalRun(pipeline: "init" | "refresh", payload: { run_id: string; status: string }): RunStatusResponse {
  return {
    run_id: payload.run_id,
    pipeline,
    status: normalizeRunStartStatus(payload.status),
    started_at: new Date().toISOString(),
    finished_at: null,
    warnings: [],
    error_code: null,
    error: null,
  };
}

function normalizeRunStartStatus(status: string): RunStatusResponse["status"] {
  if (status === "queued" || status === "running" || status === "succeeded" || status === "failed") {
    return status;
  }
  return "queued";
}

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}
