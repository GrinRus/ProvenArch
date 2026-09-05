import { useCallback, useRef } from "react";
import type { Dispatch } from "react";

import type { RunCoordination, RunListItem, RunStatusResponse } from "../lib/appContracts";
import { getPipelineRunStatus, listPipelineRuns } from "../lib/runApi";
import type { RunExplorerAction } from "../lib/runExplorerState";
import { finalStatuses, pickBootstrapRun } from "../lib/runState";
import { isAbortError, useRequestGate } from "./useRequestGate";
import { useRunUpdatePolling } from "./useRunUpdatePolling";

type RunActionsContext = {
  dispatch: Dispatch<RunExplorerAction>;
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runLogsEOF: boolean;
  setError: (message: string | null) => void;
  setRunID: (runId: string | null) => void;
  setRunStatus: (runStatus: RunStatusResponse | null) => void;
  setRunList: (runList: RunListItem[]) => void;
  setCoordination: (coordination: RunCoordination) => void;
  setRunActionStatus: (status: string) => void;
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
  runLogsEOF,
  setError,
  setRunID,
  setRunStatus,
  setRunList,
  setCoordination,
  setRunActionStatus,
  resetRunLogs,
  fetchRunLogs,
  fetchRunLogsUntilEOF,
  clearArtifacts,
  fetchArtifacts,
}: RunActionsContext) {
  const runStatusRequest = useRequestGate("run-status");
  const selectionSequenceRef = useRef(0);

  const loadRunList = useCallback(
    async (limit = 100): Promise<RunListItem[]> => {
      const payload = await listPipelineRuns(limit);
      const items = payload.items ?? [];
      setRunList(items);
      setCoordination(payload.coordination ?? {});
      return items;
    },
    [setCoordination, setRunList]
  );

  const fetchRunStatus = useCallback(
    async (id: string, allowMissing = false, selectionIsCurrent?: () => boolean): Promise<RunStatusResponse | null> => {
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
        if (finalStatuses.has(typed.status) && (!selectionIsCurrent || selectionIsCurrent())) {
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
      const selectionSequence = ++selectionSequenceRef.current;
      const selectionIsCurrent = () => selectionSequenceRef.current === selectionSequence;
      try {
        setRunActionStatus("");
        setRunID(id);
        setRunStatus(null);
        resetRunLogs();
        clearArtifacts();
        const status = await fetchRunStatus(id, false, selectionIsCurrent);
        if (!selectionIsCurrent()) {
          return;
        }
        if (!status || !finalStatuses.has(status.status)) {
          await fetchArtifacts(id);
        }
        if (!selectionIsCurrent()) {
          return;
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
    setRunActionStatus("No Attempts yet. Create the first Task to start an evidence-backed run.");
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

  return {
    bootstrapRuns,
    loadRunList,
    pollRunUpdates,
    fetchRunStatus,
    handleSelectRun,
  };
}

function activeRunResumeMessage(status: string, runID: string): string {
  if (status === "running" || status === "queued") {
    return `Resumed active run ${runID}.`;
  }
  return `Selected latest completed run ${runID}.`;
}
