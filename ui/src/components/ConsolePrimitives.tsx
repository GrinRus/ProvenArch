import type { ReactNode } from "react";

import type { Severity } from "../lib/consoleTypes";

type StatusBadgeProps = {
  tone: Severity;
  children: ReactNode;
};

export function StatusBadge({ tone, children }: StatusBadgeProps) {
  return <span className={`status-badge ${tone}`}>{children}</span>;
}

type EvidenceLinkProps = {
  path: string;
  label?: string;
  onOpenArtifact?: (path: string) => void;
};

export function EvidenceLink({ path, label, onOpenArtifact }: EvidenceLinkProps) {
  if (!onOpenArtifact) {
    return <code>{label ?? path}</code>;
  }
  return (
    <button type="button" className="evidence-link" aria-label={label ? `Open evidence reference ${label}` : "Open evidence reference"} onClick={() => onOpenArtifact(path)}>
      <span aria-hidden="true">{label ?? path}</span>
    </button>
  );
}

type PrimaryActionProps = {
  children: ReactNode;
  disabled?: boolean;
  onClick?: () => void;
  testId?: string;
};

export function PrimaryAction({ children, disabled, onClick, testId }: PrimaryActionProps) {
  return (
    <button type="button" className="primary-action" onClick={onClick} disabled={disabled} data-testid={testId}>
      {children}
    </button>
  );
}
