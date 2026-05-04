import { useEffect } from "react";

type UseRunPollingOptions = {
  shouldPollRunDetails: boolean;
  runId: string | null;
  runLogsCursor: number;
  runLogsEOF: boolean;
  pollRunUpdates: () => Promise<void>;
};

export function useRunPolling({
  shouldPollRunDetails,
  runId,
  runLogsCursor,
  runLogsEOF,
  pollRunUpdates,
}: UseRunPollingOptions) {
  useEffect(() => {
    if (!shouldPollRunDetails) {
      return;
    }

    const interval = setInterval(() => {
      void pollRunUpdates();
    }, 1000);

    return () => clearInterval(interval);
  }, [shouldPollRunDetails, runId, runLogsCursor, runLogsEOF, pollRunUpdates]);
}
