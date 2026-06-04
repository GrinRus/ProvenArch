import type { RunReviewSummaryResponse, RunStatusResponse } from "./appContracts";

export function runReviewWarningCount(runStatus: RunStatusResponse | null, reviewSummary: RunReviewSummaryResponse | null): number {
  const runWarnings = new Set<string>();
  for (const warning of [...(reviewSummary?.warnings ?? []), ...(runStatus?.warnings ?? [])]) {
    const normalized = normalizeWarning(warning);
    if (normalized) {
      runWarnings.add(normalized);
    }
  }
  const stepWarnings = (reviewSummary?.steps ?? []).reduce((total, step) => total + step.warnings_count, 0);
  return runWarnings.size + stepWarnings;
}

export function runReviewErrorCount(runStatus: RunStatusResponse | null, reviewSummary: RunReviewSummaryResponse | null): number {
  const stepErrors = (reviewSummary?.steps ?? []).reduce((total, step) => total + step.errors_count, 0);
  return stepErrors + (runStatus?.error_code || reviewSummary?.error_code ? 1 : 0);
}

function normalizeWarning(warning: string): string {
  return warning.trim().replace(/\s+/g, " ");
}
