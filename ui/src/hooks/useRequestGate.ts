import { useCallback, useEffect, useMemo, useRef } from "react";

export type RequestGateToken = {
  requestKey: string;
  signal: AbortSignal;
};

type ActiveRequest = {
  requestKey: string;
  controller: AbortController;
};

export function useRequestGate(scope: string) {
  const sequenceRef = useRef(0);
  const activeRef = useRef<ActiveRequest | null>(null);

  const abort = useCallback(() => {
    activeRef.current?.controller.abort();
    activeRef.current = null;
  }, []);

  const begin = useCallback(
    (selectionKey = ""): RequestGateToken => {
      activeRef.current?.controller.abort();
      const controller = new AbortController();
      sequenceRef.current += 1;
      const normalizedSelection = selectionKey.trim();
      const requestKey = normalizedSelection
        ? `${scope}:${normalizedSelection}:${sequenceRef.current}`
        : `${scope}:${sequenceRef.current}`;
      activeRef.current = { requestKey, controller };
      return { requestKey, signal: controller.signal };
    },
    [scope],
  );

  const isCurrent = useCallback((token: RequestGateToken) => {
    return activeRef.current?.requestKey === token.requestKey && !token.signal.aborted;
  }, []);

  const finish = useCallback((token: RequestGateToken) => {
    if (activeRef.current?.requestKey === token.requestKey) {
      activeRef.current = null;
    }
  }, []);

  const currentRequestKey = useCallback(() => activeRef.current?.requestKey ?? null, []);

  useEffect(() => abort, [abort]);

  return useMemo(
    () => ({
      abort,
      begin,
      currentRequestKey,
      finish,
      isCurrent,
    }),
    [abort, begin, currentRequestKey, finish, isCurrent],
  );
}

export function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === "AbortError";
}
