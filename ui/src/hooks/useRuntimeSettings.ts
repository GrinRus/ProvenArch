import { useState } from "react";

import { fetchJSON } from "../lib/api";
import {
  defaultRuntimeExecutionValues,
  defaultRuntimeTimeoutValues,
  normalizeRuntimeExecutionValues,
  normalizeRuntimeTimeoutValues,
  parseRuntimeExecutionPatch,
  parseRuntimeTimeoutPatch,
  runtimeExecutionDraftFromValues,
  runtimeTimeoutDraftFromValues,
  type RuntimeExecutionDraft,
  type RuntimeExecutionKey,
  type RuntimeExecutionResponse,
  type RuntimeExecutionSources,
  type RuntimeExecutionValues,
  type RuntimeProfileResponse,
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

  const [runtimeStepProviderPersisted, setRuntimeStepProviderPersisted] = useState<Partial<RuntimeStepProviderValues>>({});
  const [runtimeStepProviderEffective, setRuntimeStepProviderEffective] = useState<Partial<RuntimeStepProviderValues>>({});
  const [runtimeStepProviderSource, setRuntimeStepProviderSource] = useState<Partial<RuntimeStepProviderValues>>({});

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
      setRuntimeStepProviderPersisted(payload.step_providers?.persisted ?? {});
      setRuntimeStepProviderEffective(payload.step_providers?.effective ?? {});
      setRuntimeStepProviderSource(payload.step_providers?.source ?? {});
    } catch {
      setRuntimeStepProviderPersisted({});
      setRuntimeStepProviderEffective({});
      setRuntimeStepProviderSource({});
    }
  }

  function updateRuntimeTimeoutDraft(key: RuntimeTimeoutKey, value: string) {
    setRuntimeTimeoutDraft((previous) => ({ ...previous, [key]: value }));
  }

  function updateRuntimeExecutionDraft(key: RuntimeExecutionKey, value: string) {
    setRuntimeExecutionDraft((previous) => ({ ...previous, [key]: value }));
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

  return {
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
    runtimeStepProviderPersisted,
    runtimeStepProviderEffective,
    runtimeStepProviderSource,
    loadRuntimeTimeouts,
    loadRuntimeExecution,
    loadRuntimeProfile,
    updateRuntimeTimeoutDraft,
    updateRuntimeExecutionDraft,
    handleSaveRuntimeTimeouts,
    handleResetRuntimeTimeouts,
    handleSaveRuntimeExecution,
    handleResetRuntimeExecution,
  };
}
