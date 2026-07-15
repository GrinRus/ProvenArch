import { fetchJSON } from "./api";
import type { OnboardingPathSuggestionsResponse, OnboardingStatusResponse } from "./appContracts";

export async function loadOnboardingStatus(): Promise<OnboardingStatusResponse> {
  return fetchJSON<OnboardingStatusResponse>("/api/onboarding/status");
}

export async function selectOnboardingWorkspace(path: string, create: boolean): Promise<OnboardingStatusResponse> {
  return fetchJSON<OnboardingStatusResponse>("/api/onboarding/workspace", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, create }),
  });
}

export async function selectOnboardingRuntime(runtime: string, runtimeProvider: string): Promise<OnboardingStatusResponse> {
  return fetchJSON<OnboardingStatusResponse>("/api/onboarding/runtime", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ runtime, runtime_provider: runtimeProvider }),
  });
}

export async function enterOnboardingConsole(): Promise<OnboardingStatusResponse> {
  return fetchJSON<OnboardingStatusResponse>("/api/onboarding/enter-console", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
}

export async function forgetOnboardingRecentWorkspace(path: string): Promise<OnboardingStatusResponse> {
  return fetchJSON<OnboardingStatusResponse>("/api/onboarding/recent-workspaces/forget", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });
}

export async function loadOnboardingPathSuggestions(kind: "workspace" | "repo", query: string): Promise<OnboardingPathSuggestionsResponse> {
  const params = new URLSearchParams({ kind, query });
  return fetchJSON<OnboardingPathSuggestionsResponse>(`/api/onboarding/path-suggestions?${params.toString()}`);
}
