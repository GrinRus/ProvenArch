import type { ReactNode } from "react";

import type { Severity } from "../lib/consoleTypes";

type StatusBadgeProps = {
  tone: Severity;
  children: ReactNode;
};

export function StatusBadge({ tone, children }: StatusBadgeProps) {
  return <span className={`status-badge ${tone}`}>{children}</span>;
}

type ArtifactPathButtonProps = {
  path: string;
  label?: string;
  kind?: string;
  actionLabel?: string;
  selected?: boolean;
  onOpenArtifact: (path: string) => void;
};

export function ArtifactPathButton({ path, label, kind, actionLabel = "Open artifact", selected = false, onOpenArtifact }: ArtifactPathButtonProps) {
  const trimmed = path.trim();
  const parts = trimmed.split("/").filter(Boolean);
  const basename = label?.trim() || parts[parts.length - 1] || trimmed;
  const context = parts.length > 1 ? parts.slice(0, -1).join("/") : "";
  const accessibleLabel = kind ? `${actionLabel}: ${trimmed} (${kind})` : `${actionLabel}: ${trimmed}`;
  return (
    <button
      type="button"
      className={`artifact-path-button link-button${selected ? " is-selected" : ""}`}
      title={trimmed}
      aria-label={accessibleLabel}
      aria-current={selected ? "true" : undefined}
      onClick={() => onOpenArtifact(trimmed)}
    >
      <span className="artifact-path-name">{basename}</span>
      {context ? <span className="artifact-path-context">{context}</span> : null}
    </button>
  );
}
