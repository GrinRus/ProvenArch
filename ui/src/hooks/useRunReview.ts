import { useCallback, useEffect, useState } from "react";

import type { RunReviewSummaryResponse } from "../lib/appContracts";
import { getPipelineRunReviewSummary } from "../lib/runApi";

type UseRunReviewOptions = {
  runId: string | null;
  pollSignal: string;
};

export function useRunReview({ runId, pollSignal }: UseRunReviewOptions) {
  const [runReviewSummary, setRunReviewSummary] = useState<RunReviewSummaryResponse | null>(null);
  const [runReviewStatus, setRunReviewStatus] = useState("");

  const fetchRunReviewSummary = useCallback(
    async (id = runId, allowMissing = false) => {
      if (!id) {
        setRunReviewSummary(null);
        setRunReviewStatus("");
        return null;
      }
      try {
        const summary = await getPipelineRunReviewSummary(id, allowMissing);
        setRunReviewSummary(summary);
        setRunReviewStatus(summary ? "" : "Run review summary is not available.");
        return summary;
      } catch (error) {
        setRunReviewStatus(error instanceof Error ? error.message : "Run review summary failed to load.");
        return null;
      }
    },
    [runId],
  );

  useEffect(() => {
    void fetchRunReviewSummary(runId, true);
  }, [fetchRunReviewSummary, pollSignal, runId]);

  function clearRunReviewSummary() {
    setRunReviewSummary(null);
    setRunReviewStatus("");
  }

  return {
    runReviewSummary,
    runReviewStatus,
    fetchRunReviewSummary,
    clearRunReviewSummary,
  };
}
