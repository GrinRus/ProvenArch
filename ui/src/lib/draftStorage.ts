export type PersistedDraft<T> = {
  version: 1;
  savedAt: string;
  value: T;
};

export function draftStorageKey(kind: string, scope: string): string {
  const normalizedScope = scope.trim() || "unbound";
  return `provenarch:draft:v1:${kind}:${encodeURIComponent(normalizedScope)}`;
}

export function readDraft<T>(key: string): PersistedDraft<T> | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<PersistedDraft<T>>;
    if (parsed.version !== 1 || typeof parsed.savedAt !== "string" || parsed.value === undefined) return null;
    return parsed as PersistedDraft<T>;
  } catch {
    return null;
  }
}

export function writeDraft<T>(key: string, value: T): boolean {
  if (typeof window === "undefined") return false;
  try {
    window.localStorage.setItem(key, JSON.stringify({ version: 1, savedAt: new Date().toISOString(), value } satisfies PersistedDraft<T>));
    return true;
  } catch {
    return false;
  }
}

export function clearDraft(key: string): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.removeItem(key);
  } catch {
    // Storage can be disabled or full; the in-memory draft remains usable.
  }
}
