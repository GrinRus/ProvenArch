import { fetchJSON } from "./api";
import type { OnboardingStatusResponse } from "./appContracts";

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
