import { useCallback, useEffect, useState } from "react";

import type { RunReviewSummaryResponse } from "../lib/appContracts";
import { getPipelineRunReviewSummary } from "../lib/runApi";
import { isAbortError, useRequestGate } from "./useRequestGate";

type UseRunReviewOptions = {
  runId: string | null;
  pollSignal: string;
};

export function useRunReview({ runId, pollSignal }: UseRunReviewOptions) {
  const [runReviewSummary, setRunReviewSummary] = useState<RunReviewSummaryResponse | null>(null);
  const [runReviewStatus, setRunReviewStatus] = useState("");
  const reviewRequest = useRequestGate("run-review-summary");

  const fetchRunReviewSummary = useCallback(
    async (id = runId, allowMissing = false) => {
      if (!id) {
        reviewRequest.abort();
        setRunReviewSummary(null);
        setRunReviewStatus("");
        return null;
      }
      const token = reviewRequest.begin(`${id}:${allowMissing ? "allow-missing" : "strict"}`);
      setRunReviewSummary((current) => (current?.run_id === id ? current : null));
      setRunReviewStatus("");
      try {
        const summary = await getPipelineRunReviewSummary(id, allowMissing, { signal: token.signal });
        if (!reviewRequest.isCurrent(token)) {
          return null;
        }
        setRunReviewSummary(summary);
        setRunReviewStatus(summary ? "" : "Run review summary is not available.");
        return summary;
      } catch (error) {
        if (isAbortError(error) || !reviewRequest.isCurrent(token)) {
          return null;
        }
        setRunReviewStatus(error instanceof Error ? error.message : "Run review summary failed to load.");
        return null;
      } finally {
        reviewRequest.finish(token);
      }
    },
    [reviewRequest, runId],
  );

  useEffect(() => {
    void fetchRunReviewSummary(runId, true);
  }, [fetchRunReviewSummary, pollSignal, runId]);

  function clearRunReviewSummary() {
    reviewRequest.abort();
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
