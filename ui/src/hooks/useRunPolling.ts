import { usePollingLoop } from "./usePollingLoop";

type UseRunPollingOptions = {
  shouldPollRunDetails: boolean;
  pollRunUpdates: (signal: AbortSignal) => Promise<boolean | void>;
};

export function useRunPolling({
  shouldPollRunDetails,
  pollRunUpdates,
}: UseRunPollingOptions) {
  usePollingLoop({ enabled: shouldPollRunDetails, poll: pollRunUpdates });
}
