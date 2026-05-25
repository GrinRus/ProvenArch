import { RuntimeStepProvidersPanel } from "./RuntimeStepProvidersPanel";

type RuntimeProfileSettingsPanelProps = {
  busy: boolean;
  runtimeTimeoutKeys: string[];
  runtimeTimeoutLabels: Record<string, string>;
  runtimeTimeoutDraft: Record<string, string>;
  runtimeTimeoutPersisted: Record<string, number | undefined>;
  runtimeTimeoutEffective: Record<string, number>;
  runtimeTimeoutSource: Record<string, string | undefined>;
  runtimeTimeoutStatus: string;
  onReloadTimeouts: () => void;
  onSaveTimeouts: () => void;
  onResetTimeouts: () => void;
  onTimeoutChange: (key: string, value: string) => void;
  runtimeExecutionLabels: Record<string, string>;
  runtimeExecutionDraft: Record<string, string>;
  runtimeExecutionPersisted: Record<string, string | number | undefined>;
  runtimeExecutionEffective: Record<string, string | number>;
  runtimeExecutionSource: Record<string, string | undefined>;
  runtimeExecutionStatus: string;
  onReloadExecution: () => void;
  onSaveExecution: () => void;
  onResetExecution: () => void;
  onExecutionChange: (key: string, value: string) => void;
  runtimePermissionLabels: Record<string, string>;
  runtimePermissionDraft: Record<string, string>;
  runtimePermissionPersisted: Record<string, string | undefined>;
  runtimePermissionEffective: Record<string, string>;
  runtimePermissionSource: Record<string, string | undefined>;
  runtimePermissionStatus: string;
  onReloadPermissions: () => void;
  onSavePermissions: () => void;
  onResetPermissions: () => void;
  onPermissionChange: (key: string, value: string) => void;
  stepProviderLabels: Record<string, string>;
  stepProviderOrder: string[];
  stepProviderPersisted: Partial<Record<string, string>>;
  stepProviderEffective: Partial<Record<string, string>>;
  stepProviderSource: Partial<Record<string, string>>;
  onReloadProfile: () => void;
};

export function RuntimeProfileSettingsPanel({
  busy,
  runtimeTimeoutKeys,
  runtimeTimeoutLabels,
  runtimeTimeoutDraft,
  runtimeTimeoutPersisted,
  runtimeTimeoutEffective,
  runtimeTimeoutSource,
  runtimeTimeoutStatus,
  onReloadTimeouts,
  onSaveTimeouts,
  onResetTimeouts,
  onTimeoutChange,
  runtimeExecutionLabels,
  runtimeExecutionDraft,
  runtimeExecutionPersisted,
  runtimeExecutionEffective,
  runtimeExecutionSource,
  runtimeExecutionStatus,
  onReloadExecution,
  onSaveExecution,
  onResetExecution,
  onExecutionChange,
  runtimePermissionLabels,
  runtimePermissionDraft,
  runtimePermissionPersisted,
  runtimePermissionEffective,
  runtimePermissionSource,
  runtimePermissionStatus,
  onReloadPermissions,
  onSavePermissions,
  onResetPermissions,
  onPermissionChange,
  stepProviderLabels,
  stepProviderOrder,
  stepProviderPersisted,
  stepProviderEffective,
  stepProviderSource,
  onReloadProfile,
}: RuntimeProfileSettingsPanelProps) {
  return (
    <>
      <section className="panel" data-testid="runtime-timeouts-panel">
        <h2>Settings: Runtime Timeouts</h2>
        <p className="hint">Persisted in `workspace.yaml` (`runtime.profile.timeouts`) with precedence `env &gt; workspace &gt; defaults`.</p>
        <div className="actions">
          <button type="button" onClick={onReloadTimeouts} disabled={busy}>
            Reload runtime timeouts
          </button>
          <button type="button" onClick={onSaveTimeouts} disabled={busy} data-testid="runtime-timeouts-save-btn">
            Save runtime timeouts
          </button>
          <button type="button" onClick={onResetTimeouts} disabled={busy}>
            Reset balanced defaults
          </button>
        </div>
        {runtimeTimeoutKeys.map((key) => (
          <div key={`timeout-${key}`}>
            <label htmlFor={`runtime-timeout-${key}`}>{runtimeTimeoutLabels[key]}</label>
            <input
              id={`runtime-timeout-${key}`}
              data-testid={`runtime-timeout-input-${key}`}
              value={runtimeTimeoutDraft[key]}
              onChange={(event) => onTimeoutChange(key, event.target.value)}
            />
            <p className="hint">
              persisted: {runtimeTimeoutPersisted[key] ?? "-"} | effective: {runtimeTimeoutEffective[key]} | source: {runtimeTimeoutSource[key] ?? "default"}
            </p>
          </div>
        ))}
        {runtimeTimeoutStatus ? <p className="status ok">{runtimeTimeoutStatus}</p> : null}
      </section>

      <section className="panel" data-testid="runtime-execution-panel">
        <h2>Settings: Runtime Execution</h2>
        <p className="hint">Persisted in `workspace.yaml` (`runtime.profile.execution`) with precedence `CLI &gt; env &gt; workspace &gt; defaults`.</p>
        <div className="actions">
          <button type="button" onClick={onReloadExecution} disabled={busy}>
            Reload runtime execution
          </button>
          <button type="button" onClick={onSaveExecution} disabled={busy} data-testid="runtime-execution-save-btn">
            Save runtime execution
          </button>
          <button type="button" onClick={onResetExecution} disabled={busy}>
            Reset execution defaults
          </button>
        </div>

        <label htmlFor="runtime-execution-strategy">{runtimeExecutionLabels.strategy}</label>
        <select
          id="runtime-execution-strategy"
          data-testid="runtime-execution-strategy-select"
          value={runtimeExecutionDraft.strategy}
          onChange={(event) => onExecutionChange("strategy", event.target.value)}
        >
          <option value="sequential">sequential</option>
          <option value="parallel">parallel</option>
        </select>
        <p className="hint">
          persisted: {String(runtimeExecutionPersisted.strategy ?? "-")} | effective: {String(runtimeExecutionEffective.strategy)} | source:{" "}
          {runtimeExecutionSource.strategy ?? "default"}
        </p>

        <label htmlFor="runtime-execution-max-parallel">{runtimeExecutionLabels.max_parallel_tasks}</label>
        <input
          id="runtime-execution-max-parallel"
          data-testid="runtime-execution-max-parallel-input"
          value={runtimeExecutionDraft.max_parallel_tasks}
          onChange={(event) => onExecutionChange("max_parallel_tasks", event.target.value)}
        />
        <p className="hint">
          persisted: {String(runtimeExecutionPersisted.max_parallel_tasks ?? "-")} | effective: {String(runtimeExecutionEffective.max_parallel_tasks)} | source:{" "}
          {runtimeExecutionSource.max_parallel_tasks ?? "default"}
        </p>

        <label htmlFor="runtime-execution-failure">{runtimeExecutionLabels.failure_policy}</label>
        <select
          id="runtime-execution-failure"
          data-testid="runtime-execution-failure-policy-select"
          value={runtimeExecutionDraft.failure_policy}
          onChange={(event) => onExecutionChange("failure_policy", event.target.value)}
        >
          <option value="best_effort">best_effort</option>
          <option value="fail_fast">fail_fast</option>
        </select>
        <p className="hint">
          persisted: {String(runtimeExecutionPersisted.failure_policy ?? "-")} | effective: {String(runtimeExecutionEffective.failure_policy)} | source:{" "}
          {runtimeExecutionSource.failure_policy ?? "default"}
        </p>

        <label htmlFor="runtime-execution-shard-mode">{runtimeExecutionLabels.shard_discovery_mode}</label>
        <select
          id="runtime-execution-shard-mode"
          data-testid="runtime-execution-shard-mode-select"
          value={runtimeExecutionDraft.shard_discovery_mode}
          onChange={(event) => onExecutionChange("shard_discovery_mode", event.target.value)}
        >
          <option value="heuristics">heuristics</option>
          <option value="semantic">semantic</option>
        </select>
        <p className="hint">
          persisted: {String(runtimeExecutionPersisted.shard_discovery_mode ?? "-")} | effective:{" "}
          {String(runtimeExecutionEffective.shard_discovery_mode)} | source: {runtimeExecutionSource.shard_discovery_mode ?? "default"}
        </p>

        {runtimeExecutionStatus ? <p className="status ok">{runtimeExecutionStatus}</p> : null}
      </section>

      <section className="panel" data-testid="runtime-permissions-panel">
        <h2>Settings: Runtime Permissions</h2>
        <p className="hint">Persisted in `workspace.yaml` (`runtime.profile.permissions`) with precedence `workspace &gt; defaults`.</p>
        <div className="actions">
          <button type="button" onClick={onReloadPermissions} disabled={busy}>
            Reload runtime permissions
          </button>
          <button type="button" onClick={onSavePermissions} disabled={busy} data-testid="runtime-permissions-save-btn">
            Save runtime permissions
          </button>
          <button type="button" onClick={onResetPermissions} disabled={busy}>
            Reset permission defaults
          </button>
        </div>

        <label htmlFor="runtime-permission-mode">{runtimePermissionLabels.mode}</label>
        <select
          id="runtime-permission-mode"
          data-testid="runtime-permission-mode-select"
          value={runtimePermissionDraft.mode}
          onChange={(event) => onPermissionChange("mode", event.target.value)}
        >
          <option value="trusted_full_access">trusted_full_access</option>
          <option value="managed">managed</option>
        </select>
        <p className="hint">
          persisted: {String(runtimePermissionPersisted.mode ?? "-")} | effective: {String(runtimePermissionEffective.mode)} | source:{" "}
          {runtimePermissionSource.mode ?? "default"}
        </p>

        <label htmlFor="runtime-permission-approval-channel">{runtimePermissionLabels.approval_channel}</label>
        <select
          id="runtime-permission-approval-channel"
          data-testid="runtime-permission-approval-channel-select"
          value={runtimePermissionDraft.approval_channel}
          onChange={(event) => onPermissionChange("approval_channel", event.target.value)}
        >
          <option value="fail_fast">fail_fast</option>
          <option value="ui">ui</option>
        </select>
        <p className="hint">
          persisted: {String(runtimePermissionPersisted.approval_channel ?? "-")} | effective:{" "}
          {String(runtimePermissionEffective.approval_channel)} | source: {runtimePermissionSource.approval_channel ?? "default"}
        </p>

        {runtimePermissionStatus ? <p className="status ok">{runtimePermissionStatus}</p> : null}
      </section>

      <section className="panel" data-testid="runtime-step-providers-panel">
        <h2>Settings: Step Providers</h2>
        <p className="hint">Effective provider resolution per step from `/api/runtime/profile` with precedence `step override &gt; CLI/env global &gt; default`.</p>
        <div className="actions">
          <button type="button" onClick={onReloadProfile} disabled={busy}>
            Reload runtime profile
          </button>
        </div>
        <RuntimeStepProvidersPanel
          labels={stepProviderLabels}
          order={[...stepProviderOrder]}
          persisted={stepProviderPersisted}
          effective={stepProviderEffective}
          source={stepProviderSource}
        />
      </section>
    </>
  );
}
