import { useCallback, useEffect, useState } from "react";

import { listTasks, type ProductTask } from "../lib/taskApi";

type ReviewTasksState = {
  tasks: ProductTask[];
  status: "idle" | "loading" | "loaded" | "error";
  error: string;
};

export function useTaskReviewCandidates(enabled: boolean): ReviewTasksState & { reload: () => void } {
  const [state, setState] = useState<ReviewTasksState>({ tasks: [], status: "idle", error: "" });
  const [reloadToken, setReloadToken] = useState(0);
  const reload = useCallback(() => setReloadToken((value) => value + 1), []);

  useEffect(() => {
    if (!enabled) {
      setState({ tasks: [], status: "idle", error: "" });
      return;
    }
    const controller = new AbortController();
    setState((current) => ({ ...current, status: "loading", error: "" }));
    void listTasks({}, "", controller.signal)
      .then((response) => setState({ tasks: response.items ?? [], status: "loaded", error: "" }))
      .catch((requestError: unknown) => {
        if (controller.signal.aborted) return;
        setState({ tasks: [], status: "error", error: requestError instanceof Error ? requestError.message : "Task review list failed" });
      });
    return () => controller.abort();
  }, [enabled, reloadToken]);

  return { ...state, reload };
}
