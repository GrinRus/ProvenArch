import type { SystemInfoResponse } from "../lib/systemApi";

type TopStatusBarProps = {
  buildInfo: SystemInfoResponse | null;
  workspacePath: string;
  repoCount: number;
  runtimeMode: string;
  runtimeProvider: string;
  permissionMode: string;
  gitStatus: string;
  healthLabel: string;
  onRefresh: () => void;
};

export function TopStatusBar({ buildInfo, workspacePath, repoCount, runtimeMode, runtimeProvider, permissionMode, gitStatus, healthLabel, onRefresh }: TopStatusBarProps) {
  const localTime = new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date());
  const buildLabel = buildVersionLabel(buildInfo);
  const buildTitle = buildInfo ? `commit ${buildInfo.commit || "none"} · built ${buildInfo.built || "unknown"}` : "Build metadata unavailable";

  return (
    <header className="top-status-bar" data-testid="top-status-bar">
      <div className="brand-block">
        <div className="brand-mark" aria-hidden="true">
          <svg viewBox="0 0 32 32" focusable="false">
            <path d="M6 8l6-4 6 4v8l-6 4-6-4z" />
            <path d="M18 8l8 5v10l-8 5-8-5" />
            <path d="M12 12l6 4 6-4" />
            <path d="M18 16v12" />
          </svg>
        </div>
        <div>
          <p className="brand-name">Proven Arch</p>
          <p className="brand-version" title={buildTitle}>
            {buildLabel}
          </p>
        </div>
      </div>

      <div className="top-meta" aria-label="workspace status">
        <span className="top-meta-item">
          <TopMetaIcon type="workspace" />
          Workspace <code>{workspacePath || "not loaded"}</code>
        </span>
        <span className="top-meta-item">
          <TopMetaIcon type="repos" />
          Repos {repoCount}
        </span>
        <span className="top-meta-item">
          <TopMetaIcon type="runtime" />
          Runtime {runtimeMode}
          {runtimeMode === "headless" ? ` / ${runtimeProvider}` : ""}
        </span>
        <span className="top-meta-item">
          <TopMetaIcon type="permission" />
          Permissions {permissionMode}
        </span>
        <span className="top-meta-item" title={gitStatus || "Git publication is pending review"}>
          <TopMetaIcon type="git" />
          Git {gitStatus ? "updated" : "review pending"}
        </span>
        <span className="top-meta-item">
          <TopMetaIcon type="time" />
          Local time {localTime}
        </span>
        <span className="top-meta-item server-status">
          <span className="server-dot" aria-hidden="true" />
          Server {healthLabel}
        </span>
      </div>

      <div className="top-actions">
        <button type="button" className="secondary-action" onClick={onRefresh} data-testid="console-refresh-btn">
          Refresh
        </button>
        <button type="button" className="secondary-action" disabled title="Uses the workspace path shown in the top bar">
          Open workspace
        </button>
      </div>
    </header>
  );
}

function buildVersionLabel(buildInfo: SystemInfoResponse | null): string {
  const version = buildInfo?.version?.trim() ?? "";
  if (!version || version === "dev") {
    return "dev build";
  }
  return `${version.startsWith("v") ? version : `v${version}`} beta`;
}

function TopMetaIcon({ type }: { type: "workspace" | "repos" | "runtime" | "permission" | "git" | "time" }) {
  const paths = {
    workspace: ["M4 6h6l2 2h8v10H4z", "M4 6V4h7l2 2"],
    repos: ["M8 7a3 3 0 1 0 0 6a3 3 0 0 0 0-6", "M16 4a3 3 0 1 0 0 6a3 3 0 0 0 0-6", "M16 14a3 3 0 1 0 0 6a3 3 0 0 0 0-6", "M10.5 10.5l3-2", "M10.5 12.5l3 2"],
    runtime: ["M8 6l-4 6 4 6", "M16 6l4 6-4 6", "M13 5l-2 14"],
    permission: ["M12 3l7 3v5c0 4.5-2.8 8.2-7 10-4.2-1.8-7-5.5-7-10V6z", "M9 12l2 2 4-5"],
    git: ["M6 6h.01", "M18 6h.01", "M12 18h.01", "M6 6l6 12 6-12", "M6 6h12"],
    time: ["M12 3a9 9 0 1 0 0 18a9 9 0 0 0 0-18", "M12 7v5l3 2"],
  }[type];
  return (
    <svg className="top-meta-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      {paths.map((path) => (
        <path key={path} d={path} />
      ))}
    </svg>
  );
}
