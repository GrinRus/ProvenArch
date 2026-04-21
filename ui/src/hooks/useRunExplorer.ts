import { useEffect, useMemo, useState } from "react";

import { fetchJSON, getErrorMessage } from "../lib/api";
import {
  activeStatuses,
  finalStatuses,
  pickBootstrapRun,
  reconcileSelectedRunID,
} from "../lib/runState";
import type {
  RunListItem,
  RunListResponse,
  RunStartResponse,
  RunStatusResponse,
} from "../lib/appContracts";
import { useRunArtifacts } from "./useRunArtifacts";
import { useRunLogs } from "./useRunLogs";

type UseRunExplorerOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useRunExplorer({ setBusy, setError }: UseRunExplorerOptions) {
  const [runId, setRunID] = useState<string | null>(null);
  const [runStatus, setRunStatus] = useState<RunStatusResponse | null>(null);
  const [runList, setRunList] = useState<RunListItem[]>([]);
  const [runActionStatus, setRunActionStatus] = useState("");
  const [cancelBusy, setCancelBusy] = useState(false);
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

  const hasActiveRuns = useMemo(() => runList.some((run) => activeStatuses.has(run.status)), [runList]);
  const runCounters = useMemo(
    () =>
      runList.reduce(
        (acc, run) => {
          if (run.status === "running" || run.status === "queued") {
            acc.running += 1;
          } else if (run.status === "succeeded") {
            acc.succeeded += 1;
          } else if (run.status === "failed") {
            acc.failed += 1;
          }
          return acc;
        },
        { running: 0, succeeded: 0, failed: 0 }
      ),
    [runList]
  );
  const selectedRunListItem = useMemo(() => {
    if (!runId) {
      return null;
    }
    return runList.find((item) => item.run_id === runId) ?? null;
  }, [runId, runList]);
  const selectedRunWarnings = useMemo(() => {
    if (runStatus && runId && runStatus.run_id === runId) {
      return runStatus.warnings ?? [];
    }
    return selectedRunListItem?.warnings ?? [];
  }, [runId, runStatus, selectedRunListItem]);
  const selectedRunIsActive = useMemo(() => {
    if (runStatus && runId && runStatus.run_id === runId) {
      return activeStatuses.has(runStatus.status);
    }
    if (selectedRunListItem) {
      return activeStatuses.has(selectedRunListItem.status);
    }
    return false;
  }, [runId, runStatus, selectedRunListItem]);
  const shouldPollRunDetails = useMemo(() => {
    return hasActiveRuns || selectedRunIsActive || (runId !== null && !runLogsEOF);
  }, [hasActiveRuns, selectedRunIsActive, runId, runLogsEOF]);
  useEffect(() => {
    if (!shouldPollRunDetails) {
      return;
    }

    const interval = setInterval(() => {
      void pollRunUpdates();
    }, 1000);

    return () => clearInterval(interval);
  }, [shouldPollRunDetails, runId, runLogsCursor, runLogsEOF]);

  async function bootstrapRuns() {
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
  }

  async function loadRunList(limit = 100): Promise<RunListItem[]> {
    const payload = await fetchJSON<RunListResponse>(`/api/pipeline/runs?limit=${limit}`);
    const items = payload.items ?? [];
    setRunList(items);
    return items;
  }

  async function pollRunUpdates() {
    try {
      const latestRuns = await loadRunList(100);
      if (!runId) {
        return;
      }
      const nextSelectedRunID = reconcileSelectedRunID(runId, latestRuns);
      if (nextSelectedRunID !== runId) {
        if (nextSelectedRunID && latestRuns.length > 0) {
          setRunStatus((previous) => {
            if (previous && previous.run_id === runId) {
              return null;
            }
            return previous;
          });
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
        setRunStatus((previous) => {
          if (previous && previous.run_id === runId) {
            return null;
          }
          return previous;
        });
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
  }

  async function handleRunPipeline(pipeline: "init" | "refresh") {
    setBusy(true);
    setError(null);
    setRunActionStatus("");
    clearArtifacts();
    resetRunLogs();
    try {
      const payload = await fetchJSON<RunStartResponse>(`/api/pipeline/${pipeline}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ trigger: "ui", commit: false, create_proposal_branch: false }),
      });
      setRunList((previous) => [
        {
          run_id: payload.run_id,
          pipeline,
          status: "queued",
          started_at: new Date().toISOString(),
          finished_at: null,
          warnings: [],
          error_code: null,
          error: null,
        },
        ...previous.filter((run) => run.run_id !== payload.run_id),
      ]);
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
  }

  async function fetchRunStatus(id: string, allowMissing = false): Promise<RunStatusResponse | null> {
    const response = await fetch(`/api/pipeline/runs/${id}`);
    const payload = await response.json();
    if (response.status === 404 && allowMissing) {
      return null;
    }
    if (!response.ok) {
      throw new Error(getErrorMessage(payload, `request failed: /api/pipeline/runs/${id}`));
    }
    const typed = payload as RunStatusResponse;
    setRunStatus(typed);
    if (finalStatuses.has(typed.status)) {
      await fetchArtifacts(id);
      await loadCoverageArtifacts();
    }
    return typed;
  }

  async function handleSelectRun(
    id: string,
    options?: {
      silentErrors?: boolean;
    }
  ) {
    try {
      setRunActionStatus("");
      setRunID(id);
      resetRunLogs();
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
  }

  async function handleCancelSelectedRun() {
    if (!runId || !selectedRunIsActive) {
      return;
    }

    setCancelBusy(true);
    setError(null);
    setRunActionStatus("");
    try {
      const response = await fetch(`/api/pipeline/runs/${runId}/cancel`, {
        method: "POST",
      });
      const payload = await response.json();

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
          setRunStatus((previous) => {
            if (previous && previous.run_id === runId) {
              return null;
            }
            return previous;
          });
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

      throw new Error(getErrorMessage(payload, "failed to cancel selected run"));
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to cancel selected run");
    } finally {
      setCancelBusy(false);
    }
  }

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
