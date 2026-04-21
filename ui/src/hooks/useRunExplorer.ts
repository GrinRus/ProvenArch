import { useEffect, useMemo, useState } from "react";

import { fetchJSON, getErrorMessage } from "../lib/api";
import {
  activeStatuses,
  dedupeArtifactsByPath,
  finalStatuses,
  formatTimestamp,
  indexArtifactPath,
  pickBootstrapRun,
  reconcileSelectedRunID,
  runLogsPageLimit,
} from "../lib/runState";
import type {
  Artifact,
  ArtifactsResponse,
  FinalRunIndex,
  RunListItem,
  RunListResponse,
  RunLogEntry,
  RunLogsResponse,
  RunStartResponse,
  RunStatusResponse,
} from "../lib/appContracts";

type RunLogsMode = "events" | "raw" | "all";
type RunLogsViewMode = "line" | "line+fields";

type UseRunExplorerOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useRunExplorer({ setBusy, setError }: UseRunExplorerOptions) {
  const [runId, setRunID] = useState<string | null>(null);
  const [runStatus, setRunStatus] = useState<RunStatusResponse | null>(null);
  const [runList, setRunList] = useState<RunListItem[]>([]);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [selectedArtifact, setSelectedArtifact] = useState<string>("");
  const [selectedArtifactContent, setSelectedArtifactContent] = useState<string>("");
  const [runLogs, setRunLogs] = useState<RunLogEntry[]>([]);
  const [runLogsCursor, setRunLogsCursor] = useState(0);
  const [runLogsEOF, setRunLogsEOF] = useState(false);
  const [runLogsStatus, setRunLogsStatus] = useState("");
  const [runLogsViewMode, setRunLogsViewMode] = useState<RunLogsViewMode>("line");
  const [runLogsMode, setRunLogsMode] = useState<RunLogsMode>("all");
  const [runActionStatus, setRunActionStatus] = useState("");
  const [cancelBusy, setCancelBusy] = useState(false);
  const [coverageSummary, setCoverageSummary] = useState("");
  const [openQuestions, setOpenQuestions] = useState("");

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
  const runLogTaskrunPaths = useMemo(() => {
    const paths = new Set<string>();
    for (const entry of runLogs) {
      if (entry.taskrun_path && entry.taskrun_path.trim().length > 0) {
        paths.add(entry.taskrun_path.trim());
      }
    }
    return Array.from(paths).sort((left, right) => left.localeCompare(right));
  }, [runLogs]);
  const filteredRunLogs = useMemo(() => {
    if (runLogsMode === "all") {
      return runLogs;
    }
    if (runLogsMode === "events") {
      return runLogs.filter((entry) => (entry.kind ?? "event") !== "runtime_output");
    }
    return runLogs.filter((entry) => (entry.kind ?? "event") === "runtime_output");
  }, [runLogs, runLogsMode]);
  const diagramArtifacts = useMemo(() => {
    return artifacts
      .filter((artifact) => artifact.kind === "diagram" || artifact.kind === "diagram-index" || artifact.path.startsWith("reports/diagrams/"))
      .sort((left, right) => left.path.localeCompare(right.path));
  }, [artifacts]);
  const nonDiagramArtifacts = useMemo(() => {
    return artifacts
      .filter((artifact) => !(artifact.kind === "diagram" || artifact.kind === "diagram-index" || artifact.path.startsWith("reports/diagrams/")))
      .sort((left, right) => left.path.localeCompare(right.path));
  }, [artifacts]);
  const selectedArtifactIsMermaid = useMemo(() => {
    if (!selectedArtifact) {
      return false;
    }
    if (selectedArtifact.endsWith(".mmd")) {
      return true;
    }
    const text = selectedArtifactContent.trim();
    if (text.startsWith("flowchart") || text.startsWith("graph") || text.startsWith("sequenceDiagram") || text.startsWith("classDiagram")) {
      return true;
    }
    return text.includes("```mermaid");
  }, [selectedArtifact, selectedArtifactContent]);
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
  const runLogsRendered = useMemo(() => {
    const includeFields = runLogsViewMode === "line+fields";
    return filteredRunLogs
      .map((entry) => {
        const line = formatRunLogLine(entry);
        if (!includeFields) {
          return line;
        }
        if (!entry.fields || Object.keys(entry.fields).length === 0) {
          return line;
        }
        const serialized = JSON.stringify(entry.fields, null, 2);
        if (!serialized) {
          return line;
        }
        return `${line}\n${serialized}`;
      })
      .join(includeFields ? "\n\n" : "\n");
  }, [filteredRunLogs, runLogsViewMode]);

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

  function resetRunLogs() {
    setRunLogs([]);
    setRunLogsCursor(0);
    setRunLogsEOF(false);
    setRunLogsStatus("");
  }

  function mergeRunLogsPayload(payload: RunLogsResponse, reset: boolean, fallbackCursor: number) {
    setRunLogs((previous) => {
      const seed = reset ? [] : previous;
      const merged = [...seed];
      const seen = new Set(merged.map((entry) => entry.cursor));
      for (const entry of payload.items ?? []) {
        if (seen.has(entry.cursor)) {
          continue;
        }
        merged.push(entry);
        seen.add(entry.cursor);
      }
      merged.sort((left, right) => left.cursor - right.cursor);
      return merged;
    });
    setRunLogsCursor(payload.next_cursor ?? fallbackCursor);
    setRunLogsEOF(Boolean(payload.eof));
  }

  async function fetchRunLogs(id: string, reset = false): Promise<RunLogsResponse | null> {
    if (!id) {
      return null;
    }
    const cursor = reset ? 0 : runLogsCursor;
    if (!reset && runLogsEOF) {
      return null;
    }
    const payload = await fetchJSON<RunLogsResponse>(
      `/api/pipeline/runs/${id}/logs?cursor=${cursor}&limit=${runLogsPageLimit}`
    );
    mergeRunLogsPayload(payload, reset, cursor);
    return payload;
  }

  async function fetchRunLogsUntilEOF(id: string) {
    if (!id) {
      return;
    }
    let cursor = 0;
    let reset = true;
    for (let page = 0; page < 25; page += 1) {
      const payload = await fetchJSON<RunLogsResponse>(
        `/api/pipeline/runs/${id}/logs?cursor=${cursor}&limit=${runLogsPageLimit}`
      );
      mergeRunLogsPayload(payload, reset, cursor);
      if (payload.eof) {
        return;
      }
      const nextCursor = payload.next_cursor ?? cursor;
      if (nextCursor <= cursor) {
        return;
      }
      cursor = nextCursor;
      reset = false;
    }
  }

  async function pollRunUpdates() {
    try {
      const latestRuns = await loadRunList(100);
      if (!runId) {
        return;
      }
      const nextSelectedRunID = reconcileSelectedRunID(runId, latestRuns);
      if (nextSelectedRunID !== runId) {
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
        setSelectedArtifact("");
        setSelectedArtifactContent("");
        setRunStatus((previous) => {
          if (previous && previous.run_id === runId) {
            return null;
          }
          return previous;
        });
        setArtifacts([]);
        resetRunLogs();
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

  async function loadTextArtifact(path: string, setter: (value: string) => void) {
    try {
      const response = await fetch(`/api/artifacts?path=${encodeURIComponent(path)}`);
      if (!response.ok) {
        setter("");
        return;
      }
      setter(await response.text());
    } catch {
      setter("");
    }
  }

  async function handleRunPipeline(pipeline: "init" | "refresh") {
    setBusy(true);
    setError(null);
    setRunActionStatus("");
    setArtifacts([]);
    setSelectedArtifact("");
    setSelectedArtifactContent("");
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
          setSelectedArtifact("");
          setSelectedArtifactContent("");
          setRunStatus((previous) => {
            if (previous && previous.run_id === runId) {
              return null;
            }
            return previous;
          });
          setArtifacts([]);
          resetRunLogs();
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

  async function fetchArtifacts(id: string) {
    const payload = await fetchJSON<ArtifactsResponse>(`/api/pipeline/runs/${id}/artifacts`);
    const runArtifacts = payload.artifacts ?? [];
    const finalRunIndexPath = indexArtifactPath(runArtifacts, "/staging/final/final-run-index.json");
    if (!finalRunIndexPath) {
      setArtifacts(runArtifacts);
      return;
    }

    try {
      const finalRunIndex = await fetchJSON<FinalRunIndex>(`/api/artifacts?path=${encodeURIComponent(finalRunIndexPath)}`);
      const canonicalArtifacts: Artifact[] = (finalRunIndex.canonical_documents ?? [])
        .map((document) => {
          const canonicalPath = String(document.canonical_path ?? "").trim();
          if (!canonicalPath) {
            return null;
          }
          return {
            path: canonicalPath,
            kind: String(document.kind ?? "report").trim() || "report",
            label: String(document.title ?? canonicalPath).trim() || canonicalPath,
          } satisfies Artifact;
        })
        .filter((artifact): artifact is Artifact => artifact !== null);

      const indexArtifacts: Artifact[] = [
        {
          path: finalRunIndexPath,
          kind: "taskrun",
          label: "Final Run Index",
        },
      ];
      const citationIndexPath = String(finalRunIndex.citation_index_path ?? "").trim();
      if (citationIndexPath.length > 0) {
        indexArtifacts.push({
          path: citationIndexPath,
          kind: "taskrun",
          label: "Citation Index",
        });
      }
      setArtifacts(dedupeArtifactsByPath([...canonicalArtifacts, ...indexArtifacts]));
    } catch {
      setArtifacts(runArtifacts);
    }
  }

  async function loadCoverageArtifacts() {
    await loadTextArtifact("reports/coverage/summary.md", setCoverageSummary);
    await loadTextArtifact("reports/coverage/open-questions.md", setOpenQuestions);
  }

  async function handleOpenArtifact(path: string) {
    setSelectedArtifact(path);
    setSelectedArtifactContent("Loading...");
    await loadTextArtifact(path, setSelectedArtifactContent);
  }

  function formatRunLogLine(entry: RunLogEntry): string {
    const kind = entry.kind ?? "event";
    const stream = entry.stream ? `[${entry.stream.toUpperCase()}]` : "";
    const parts = [
      formatTimestamp(entry.timestamp),
      entry.level.toUpperCase(),
      kind === "runtime_output" ? "[RAW]" : "[EVENT]",
      stream,
      entry.step_id ? `[${entry.step_id}]` : "",
      entry.domain_id ? `(${entry.domain_id})` : "",
      entry.message,
    ].filter((value) => value && value.trim().length > 0);
    return parts.join(" ");
  }

  async function handleCopyRunLogs() {
    if (filteredRunLogs.length === 0 || runLogsRendered.trim().length === 0) {
      return;
    }
    if (!navigator.clipboard || !navigator.clipboard.writeText) {
      setRunLogsStatus("Clipboard API is not available in this browser context.");
      return;
    }

    try {
      await navigator.clipboard.writeText(runLogsRendered);
      setRunLogsStatus("Run logs copied to clipboard");
    } catch (requestError) {
      setRunLogsStatus(requestError instanceof Error ? requestError.message : "Run logs copy failed");
    }
  }

  function handleDownloadRunLogs() {
    if (filteredRunLogs.length === 0 || !runId || runLogsRendered.trim().length === 0) {
      return;
    }
    const blob = new Blob([runLogsRendered], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${runId}.logs.txt`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    setRunLogsStatus(`Downloaded ${runId}.logs.txt`);
  }

  return {
    runId,
    setRunID,
    runStatus,
    runList,
    artifacts,
    selectedArtifact,
    selectedArtifactContent,
    runLogs,
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
