import { fetchJSON } from "./api";
import type { DoctorResponse } from "./appContracts";

export type SystemInfoResponse = {
  version: string;
  commit: string;
  built: string;
};

export type DoctorRequest = {
  runtime?: string;
  runtime_provider?: string;
  repo_path?: string;
  repo_git_url?: string;
};

export async function loadSystemInfo(): Promise<SystemInfoResponse> {
  return fetchJSON<SystemInfoResponse>("/api/system/info");
}

export async function loadSystemDoctor(request: DoctorRequest = {}): Promise<DoctorResponse> {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(request)) {
    const trimmed = String(value ?? "").trim();
    if (trimmed) {
      params.set(key, trimmed);
    }
  }
  const suffix = params.toString() ? `?${params.toString()}` : "";
  return fetchJSON<DoctorResponse>(`/api/system/doctor${suffix}`);
}
