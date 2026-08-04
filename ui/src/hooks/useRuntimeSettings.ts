import { useState } from "react";

import { fetchJSON } from "../lib/api";
import {
  defaultRuntimeExecutionValues,
  defaultRuntimePermissionValues,
  defaultRuntimeTimeoutValues,
  normalizeRuntimeExecutionValues,
  normalizeRuntimePermissionValues,
  normalizeRuntimeTimeoutValues,
  parseRuntimeExecutionPatch,
  parseRuntimePermissionPatch,
  parseRuntimeTimeoutPatch,
  runtimeExecutionDraftFromValues,
  runtimePermissionDraftFromValues,
  runtimeTimeoutDraftFromValues,
  type RuntimeExecutionDraft,
  type RuntimeExecutionKey,
  type RuntimeExecutionResponse,
  type RuntimeExecutionSources,
  type RuntimeExecutionValues,
  type RuntimePermissionDraft,
  type RuntimePermissionKey,
  type RuntimePermissionsResponse,
  type RuntimePermissionSources,
  type RuntimePermissionValues,
  type RuntimeProfileResponse,
  type RuntimeProviderModelDraft,
  type RuntimeProviderModels,
  type RuntimeProviderModelsResponse,
  type RuntimeStepProviderValues,
  type RuntimeTimeoutKey,
  type RuntimeTimeoutsResponse,
  type RuntimeTimeoutSources,
  type RuntimeTimeoutValues,
} from "../lib/appContracts";

type UseRuntimeSettingsOptions = {
  setBusy: (busy: boolean) => void;
  setError: (message: string | null) => void;
};

export function useRuntimeSettings({ setBusy, setError }: UseRuntimeSettingsOptions) {
  const [effectiveRuntimeMode, setEffectiveRuntimeMode] = useState<"fake" | "headless" | "unknown">("unknown");
  const [effectiveRuntimeProvider, setEffectiveRuntimeProvider] = useState("unknown");
  const [effectiveProviderSource, setEffectiveProviderSource] = useState("unknown");
  const [runtimeTimeoutPersisted, setRuntimeTimeoutPersisted] = useState<Partial<RuntimeTimeoutValues>>({});
  const [runtimeTimeoutEffective, setRuntimeTimeoutEffective] = useState<RuntimeTimeoutValues>(defaultRuntimeTimeoutValues);
  const [runtimeTimeoutSource, setRuntimeTimeoutSource] = useState<Partial<RuntimeTimeoutSources>>({});
  const [runtimeTimeoutDraft, setRuntimeTimeoutDraft] = useState<Record<RuntimeTimeoutKey, string>>(
    runtimeTimeoutDraftFromValues(defaultRuntimeTimeoutValues)
  );
  const [runtimeTimeoutStatus, setRuntimeTimeoutStatus] = useState("");

  const [runtimeExecutionPersisted, setRuntimeExecutionPersisted] = useState<Partial<RuntimeExecutionValues>>({});
  const [runtimeExecutionEffective, setRuntimeExecutionEffective] = useState<RuntimeExecutionValues>(defaultRuntimeExecutionValues);
  const [runtimeExecutionSource, setRuntimeExecutionSource] = useState<Partial<RuntimeExecutionSources>>({});
  const [runtimeExecutionDraft, setRuntimeExecutionDraft] = useState<RuntimeExecutionDraft>(
    runtimeExecutionDraftFromValues(defaultRuntimeExecutionValues)
  );
  const [runtimeExecutionStatus, setRuntimeExecutionStatus] = useState("");

  const [runtimePermissionPersisted, setRuntimePermissionPersisted] = useState<Partial<RuntimePermissionValues>>({});
  const [runtimePermissionEffective, setRuntimePermissionEffective] = useState<RuntimePermissionValues>(defaultRuntimePermissionValues);
  const [runtimePermissionSource, setRuntimePermissionSource] = useState<Partial<RuntimePermissionSources>>({});
  const [runtimePermissionDraft, setRuntimePermissionDraft] = useState<RuntimePermissionDraft>(
    runtimePermissionDraftFromValues(defaultRuntimePermissionValues)
  );
  const [runtimePermissionStatus, setRuntimePermissionStatus] = useState("");

  const [runtimeStepProviderPersisted, setRuntimeStepProviderPersisted] = useState<Partial<RuntimeStepProviderValues>>({});
  const [runtimeStepProviderEffective, setRuntimeStepProviderEffective] = useState<Partial<RuntimeStepProviderValues>>({});
  const [runtimeStepProviderSource, setRuntimeStepProviderSource] = useState<Partial<RuntimeStepProviderValues>>({});
  const [runtimeProviderModels, setRuntimeProviderModels] = useState<RuntimeProviderModels>({});
  const [runtimeProviderModelDraft, setRuntimeProviderModelDraft] = useState<Record<string, RuntimeProviderModelDraft>>({});
  const [runtimeProviderModelStatus, setRuntimeProviderModelStatus] = useState("");

  async function loadRuntimeTimeouts() {
    try {
      const payload = await fetchJSON<RuntimeTimeoutsResponse>("/api/runtime/timeouts");
      const nextEffective = normalizeRuntimeTimeoutValues(payload.effective, defaultRuntimeTimeoutValues);
      setRuntimeTimeoutPersisted(payload.persisted ?? {});
      setRuntimeTimeoutEffective(nextEffective);
      setRuntimeTimeoutSource(payload.source ?? {});
      setRuntimeTimeoutDraft(runtimeTimeoutDraftFromValues(nextEffective));
    } catch {
      setRuntimeTimeoutPersisted({});
      setRuntimeTimeoutEffective(defaultRuntimeTimeoutValues);
      setRuntimeTimeoutSource({});
      setRuntimeTimeoutDraft(runtimeTimeoutDraftFromValues(defaultRuntimeTimeoutValues));
    }
  }

  async function loadRuntimeExecution() {
    try {
      const payload = await fetchJSON<RuntimeExecutionResponse>("/api/runtime/execution");
      const nextEffective = normalizeRuntimeExecutionValues(payload.effective, defaultRuntimeExecutionValues);
      setRuntimeExecutionPersisted(payload.persisted ?? {});
      setRuntimeExecutionEffective(nextEffective);
      setRuntimeExecutionSource(payload.source ?? {});
      setRuntimeExecutionDraft(runtimeExecutionDraftFromValues(nextEffective));
    } catch {
      setRuntimeExecutionPersisted({});
      setRuntimeExecutionEffective(defaultRuntimeExecutionValues);
      setRuntimeExecutionSource({});
      setRuntimeExecutionDraft(runtimeExecutionDraftFromValues(defaultRuntimeExecutionValues));
    }
  }

  async function loadRuntimeProfile() {
    try {
      const payload = await fetchJSON<RuntimeProfileResponse>("/api/runtime/profile");
      setEffectiveRuntimeMode(payload.runtime_mode ?? "unknown");
      setEffectiveRuntimeProvider(payload.runtime_provider ?? "unknown");
      setEffectiveProviderSource(payload.provider_source ?? "unknown");
      const nextPermissions = normalizeRuntimePermissionValues(payload.permissions?.effective, defaultRuntimePermissionValues);
      setRuntimePermissionPersisted(payload.permissions?.persisted ?? {});
      setRuntimePermissionEffective(nextPermissions);
      setRuntimePermissionSource(payload.permissions?.source ?? {});
      setRuntimePermissionDraft(runtimePermissionDraftFromValues(nextPermissions));
      setRuntimeStepProviderPersisted(payload.step_providers?.persisted ?? {});
      setRuntimeStepProviderEffective(payload.step_providers?.effective ?? {});
      setRuntimeStepProviderSource(payload.step_providers?.source ?? {});
      setRuntimeProviderModels(payload.provider_models ?? {});
      setRuntimeProviderModelDraft(runtimeProviderModelDraftFromPayload(payload.provider_models ?? {}));
    } catch {
      setEffectiveRuntimeMode("unknown");
      setEffectiveRuntimeProvider("unknown");
      setEffectiveProviderSource("unknown");
      setRuntimePermissionPersisted({});
      setRuntimePermissionEffective(defaultRuntimePermissionValues);
      setRuntimePermissionSource({});
      setRuntimePermissionDraft(runtimePermissionDraftFromValues(defaultRuntimePermissionValues));
      setRuntimeStepProviderPersisted({});
      setRuntimeStepProviderEffective({});
      setRuntimeStepProviderSource({});
      setRuntimeProviderModels({});
      setRuntimeProviderModelDraft({});
    }
  }

  async function loadRuntimePermissions() {
    try {
      const payload = await fetchJSON<RuntimePermissionsResponse>("/api/runtime/permissions");
      const nextEffective = normalizeRuntimePermissionValues(payload.effective, defaultRuntimePermissionValues);
      setRuntimePermissionPersisted(payload.persisted ?? {});
      setRuntimePermissionEffective(nextEffective);
      setRuntimePermissionSource(payload.source ?? {});
      setRuntimePermissionDraft(runtimePermissionDraftFromValues(nextEffective));
    } catch {
      setRuntimePermissionPersisted({});
      setRuntimePermissionEffective(defaultRuntimePermissionValues);
      setRuntimePermissionSource({});
      setRuntimePermissionDraft(runtimePermissionDraftFromValues(defaultRuntimePermissionValues));
    }
  }

  function updateRuntimeTimeoutDraft(key: RuntimeTimeoutKey, value: string) {
    setRuntimeTimeoutDraft((previous) => ({ ...previous, [key]: value }));
  }

  function updateRuntimeExecutionDraft(key: RuntimeExecutionKey, value: string) {
    setRuntimeExecutionDraft((previous) => ({ ...previous, [key]: value }));
  }

  function updateRuntimePermissionDraft(key: RuntimePermissionKey, value: string) {
    setRuntimePermissionDraft((previous) => ({ ...previous, [key]: value }));
  }

  function updateRuntimeProviderModelDraft(provider: string, field: keyof RuntimeProviderModelDraft, value: string) {
    setRuntimeProviderModelDraft((previous) => ({
      ...previous,
      [provider]: { ...(previous[provider] ?? { model: "", effort: "" }), [field]: value },
    }));
  }

  async function handleSaveRuntimeProviderModels() {
    setBusy(true);
    setError(null);
    setRuntimeProviderModelStatus("");
    try {
      const providers = Object.fromEntries(
        Object.entries(runtimeProviderModelDraft).map(([provider, config]) => [provider, { model: config.model, effort: config.effort }])
      );
      const payload = await fetchJSON<RuntimeProviderModelsResponse>("/api/runtime/models", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ providers }),
      });
      setRuntimeProviderModels(payload.providers ?? {});
      setRuntimeProviderModelDraft(runtimeProviderModelDraftFromPayload(payload.providers ?? {}));
      setRuntimeProviderModelStatus("Runtime model settings saved");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save runtime model settings");
    } finally {
      setBusy(false);
    }
  }

  async function handleResetRuntimeProviderModels() {
    setBusy(true);
    setError(null);
    setRuntimeProviderModelStatus("");
    try {
      const payload = await fetchJSON<RuntimeProviderModelsResponse>("/api/runtime/models", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ providers: {} }),
      });
      setRuntimeProviderModels(payload.providers ?? {});
      setRuntimeProviderModelDraft(runtimeProviderModelDraftFromPayload(payload.providers ?? {}));
      setRuntimeProviderModelStatus("Runtime model settings reset to provider defaults");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to reset runtime model settings");
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveRuntimeTimeouts() {
    setBusy(true);
    setError(null);
    setRuntimeTimeoutStatus("");
    try {
      const patch = parseRuntimeTimeoutPatch(runtimeTimeoutDraft);
      await fetchJSON<RuntimeTimeoutsResponse>("/api/runtime/timeouts", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ timeouts: patch }),
      });
      await loadRuntimeTimeouts();
      setRuntimeTimeoutStatus("Runtime timeouts saved");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save runtime timeouts");
    } finally {
      setBusy(false);
    }
  }

  async function handleResetRuntimeTimeouts() {
    setBusy(true);
    setError(null);
    setRuntimeTimeoutStatus("");
    try {
      await fetchJSON<RuntimeTimeoutsResponse>("/api/runtime/timeouts", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ timeouts: defaultRuntimeTimeoutValues }),
      });
      await loadRuntimeTimeouts();
      setRuntimeTimeoutStatus("Runtime timeouts reset to balanced defaults");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to reset runtime timeouts");
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveRuntimeExecution() {
    setBusy(true);
    setError(null);
    setRuntimeExecutionStatus("");
    try {
      const patch = parseRuntimeExecutionPatch(runtimeExecutionDraft);
      await fetchJSON<RuntimeExecutionResponse>("/api/runtime/execution", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ execution: patch }),
      });
      await loadRuntimeExecution();
      setRuntimeExecutionStatus("Runtime execution profile saved");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save runtime execution profile");
    } finally {
      setBusy(false);
    }
  }

  async function handleResetRuntimeExecution() {
    setBusy(true);
    setError(null);
    setRuntimeExecutionStatus("");
    try {
      await fetchJSON<RuntimeExecutionResponse>("/api/runtime/execution", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ execution: defaultRuntimeExecutionValues }),
      });
      await loadRuntimeExecution();
      setRuntimeExecutionStatus("Runtime execution profile reset to defaults");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to reset runtime execution profile");
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveRuntimePermissions() {
    setBusy(true);
    setError(null);
    setRuntimePermissionStatus("");
    try {
      const patch = parseRuntimePermissionPatch(runtimePermissionDraft);
      await fetchJSON<RuntimePermissionsResponse>("/api/runtime/permissions", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ permissions: patch }),
      });
      await loadRuntimePermissions();
      setRuntimePermissionStatus("Runtime permissions saved");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to save runtime permissions");
    } finally {
      setBusy(false);
    }
  }

  async function handleResetRuntimePermissions() {
    setBusy(true);
    setError(null);
    setRuntimePermissionStatus("");
    try {
      await fetchJSON<RuntimePermissionsResponse>("/api/runtime/permissions", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ permissions: defaultRuntimePermissionValues }),
      });
      await loadRuntimePermissions();
      setRuntimePermissionStatus("Runtime permissions reset to defaults");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "failed to reset runtime permissions");
    } finally {
      setBusy(false);
    }
  }

  return {
    effectiveRuntimeMode,
    effectiveRuntimeProvider,
    effectiveProviderSource,
    runtimeTimeoutPersisted,
    runtimeTimeoutEffective,
    runtimeTimeoutSource,
    runtimeTimeoutDraft,
    runtimeTimeoutStatus,
    runtimeExecutionPersisted,
    runtimeExecutionEffective,
    runtimeExecutionSource,
    runtimeExecutionDraft,
    runtimeExecutionStatus,
    runtimePermissionPersisted,
    runtimePermissionEffective,
    runtimePermissionSource,
    runtimePermissionDraft,
    runtimePermissionStatus,
    runtimeStepProviderPersisted,
    runtimeStepProviderEffective,
    runtimeStepProviderSource,
    runtimeProviderModels,
    runtimeProviderModelDraft,
    runtimeProviderModelStatus,
    loadRuntimeTimeouts,
    loadRuntimeExecution,
    loadRuntimePermissions,
    loadRuntimeProfile,
    updateRuntimeTimeoutDraft,
    updateRuntimeExecutionDraft,
    updateRuntimePermissionDraft,
    updateRuntimeProviderModelDraft,
    handleSaveRuntimeTimeouts,
    handleResetRuntimeTimeouts,
    handleSaveRuntimeExecution,
    handleResetRuntimeExecution,
    handleSaveRuntimePermissions,
    handleResetRuntimePermissions,
    handleSaveRuntimeProviderModels,
    handleResetRuntimeProviderModels,
  };
}

function runtimeProviderModelDraftFromPayload(payload: RuntimeProviderModels): Record<string, RuntimeProviderModelDraft> {
  const draft: Record<string, RuntimeProviderModelDraft> = {};
  for (const [provider, entry] of Object.entries(payload)) {
    draft[provider] = {
      model: entry.persisted?.model ?? "",
      effort: entry.persisted?.effort ?? "",
    };
  }
  return draft;
}
