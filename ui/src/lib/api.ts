type APIErrorPayload = {
  error?:
    | string
    | {
        code?: string;
        message?: string;
      };
};

export function getErrorMessage(payload: unknown, fallback: string): string {
  const typed = payload as APIErrorPayload;
  if (!typed || typeof typed !== "object") {
    return fallback;
  }
  if (typeof typed.error === "string") {
    return typed.error;
  }
  if (typed.error && typeof typed.error.message === "string") {
    return typed.error.message;
  }
  return fallback;
}

export async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  const payload = (await response.json()) as T;
  if (!response.ok) {
    throw new Error(getErrorMessage(payload, `request failed: ${url}`));
  }
  return payload;
}
