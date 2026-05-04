import { useMemo } from "react";

import type { RunListItem, RunStatusResponse } from "../lib/appContracts";
import { activeStatuses } from "../lib/runState";

type UseRunSelectionOptions = {
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runList: RunListItem[];
  runLogsEOF: boolean;
};

export function useRunSelection({ runId, runStatus, runList, runLogsEOF }: UseRunSelectionOptions) {
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

  return {
    hasActiveRuns,
    runCounters,
    selectedRunListItem,
    selectedRunWarnings,
    selectedRunIsActive,
    shouldPollRunDetails,
  };
}
