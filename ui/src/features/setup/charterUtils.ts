import type { Diagnostic, EditableArtifactOption } from "../../lib/appContracts";

export type CharterRecoveryIssue = {
  artifactPath: string;
  artifactLabel: string;
  category: string;
  promptUsage: string;
  severity: Diagnostic["level"];
  diagnosticCode: string;
  message: string;
  suggestion: string;
};

export function buildCharterBaselineRecovery(
  baselineBundleWarnings: Diagnostic[],
  baselineEditorArtifacts: EditableArtifactOption[],
  selectedEditorPath: string,
): CharterRecoveryIssue | null {
  const diagnostic = baselineBundleWarnings.find((warning) => warning.level === "error") ?? baselineBundleWarnings[0];
  if (!diagnostic) {
    return null;
  }

  const artifact = findCharterDiagnosticArtifact(diagnostic, baselineEditorArtifacts, selectedEditorPath);
  const artifactPath = artifact?.path ?? diagnostic.path ?? selectedEditorPath ?? "baseline bundle";
  return {
    artifactPath,
    artifactLabel: artifact?.label ?? artifactPath,
    category: artifact?.category ?? (artifactPath.startsWith("charter/") ? "charter" : artifactPath.startsWith("skills/") ? "skills" : "bundle"),
    promptUsage: promptUsageLabel(artifact),
    severity: diagnostic.level,
    diagnosticCode: diagnostic.code,
    message: diagnostic.message,
    suggestion: diagnostic.suggestion || defaultCharterSuggestion(artifactPath),
  };
}

function findCharterDiagnosticArtifact(
  diagnostic: Diagnostic,
  baselineEditorArtifacts: EditableArtifactOption[],
  selectedEditorPath: string,
): EditableArtifactOption | undefined {
  const directPath = diagnostic.path?.trim();
  if (directPath) {
    const direct = baselineEditorArtifacts.find((artifact) => artifact.path === directPath);
    if (direct) {
      return direct;
    }
    const suffix = baselineEditorArtifacts.find((artifact) => directPath.endsWith(artifact.path) || artifact.path.endsWith(directPath));
    if (suffix) {
      return suffix;
    }
  }

  const messageMatch = baselineEditorArtifacts.find((artifact) => diagnostic.message.includes(artifact.path) || diagnostic.message.includes(artifact.label));
  if (messageMatch) {
    return messageMatch;
  }

  if (selectedEditorPath) {
    return baselineEditorArtifacts.find((artifact) => artifact.path === selectedEditorPath);
  }

  return undefined;
}

export function promptUsageLabel(artifact?: EditableArtifactOption): string {
  if (!artifact) {
    return "bundle diagnostic";
  }
  if (artifact.prompt_usage === "live-consumed") {
    return "live consumed";
  }
  if (artifact.prompt_usage === "reference-only") {
    return "reference only";
  }
  return artifact.path.startsWith("charter/") ? "charter context" : "editable baseline";
}

export function defaultCharterSuggestion(artifactPath: string): string {
  if (artifactPath.startsWith("skills/prompt-packs/")) {
    return "Open the live-consumed prompt pack, fix the diagnostic, then save the selected baseline artifact.";
  }
  if (artifactPath.startsWith("charter/")) {
    return "Open the charter artifact, fix the project context, then save the selected baseline artifact.";
  }
  return "Open the affected baseline artifact, fix the diagnostic, then save it before running Analysis.";
}

export function splitSummaryList(value: string): string[] {
  return value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}
