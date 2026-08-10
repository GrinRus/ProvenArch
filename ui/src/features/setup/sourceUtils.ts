import type { Diagnostic, GuidedRepo, ValidateResponse } from "../../lib/appContracts";

export type SourceRecoveryIssue = {
  repoKey: string;
  diagnosticLabel: string;
  level: Diagnostic["level"] | "draft";
  message: string;
  suggestion: string;
  sourceType: string;
  sourceValue: string;
  refValue: string;
};

export function buildSourceValidationRecovery(
  guidedRepos: GuidedRepo[],
  validateResult: ValidateResponse | null,
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>,
): SourceRecoveryIssue | null {
  const serverIssue = firstSourceDiagnostic(guidedRepos, validateResult, validationDiagnosticsByRepo);
  return serverIssue ?? firstDraftSourceIssue(guidedRepos);
}

function firstSourceDiagnostic(
  guidedRepos: GuidedRepo[],
  validateResult: ValidateResponse | null,
  validationDiagnosticsByRepo: Array<[string, Diagnostic[]]>,
): SourceRecoveryIssue | null {
  if (!validateResult) {
    return null;
  }

  const diagnosticEntry =
    validationDiagnosticsByRepo
      .flatMap(([repoKey, diagnostics]) => diagnostics.map((diagnostic) => ({ repoKey, diagnostic })))
      .find(({ diagnostic }) => diagnostic.level === "error") ??
    (!validateResult.ok
      ? validationDiagnosticsByRepo.flatMap(([repoKey, diagnostics]) => diagnostics.map((diagnostic) => ({ repoKey, diagnostic })))[0]
      : undefined);

  if (!diagnosticEntry) {
    return null;
  }

  const repo = guidedRepos.find((candidate) => candidate.name === diagnosticEntry.repoKey || candidate.name === diagnosticEntry.diagnostic.repo);
  return {
    repoKey: diagnosticEntry.repoKey === "__workspace__" ? "workspace.yaml" : diagnosticEntry.repoKey,
    diagnosticLabel: diagnosticEntry.diagnostic.code,
    level: diagnosticEntry.diagnostic.level,
    message: diagnosticEntry.diagnostic.message,
    suggestion: diagnosticEntry.diagnostic.suggestion || defaultSourceSuggestion(repo),
    sourceType: sourceTypeLabel(repo),
    sourceValue: sourceValueLabel(repo, validateResult.workspace),
    refValue: repo?.ref || "current/default",
  };
}

function firstDraftSourceIssue(guidedRepos: GuidedRepo[]): SourceRecoveryIssue | null {
  const nameCounts = new Map<string, number>();
  guidedRepos.forEach((repo) => {
    const name = repo.name.trim().toLowerCase();
    if (name) {
      nameCounts.set(name, (nameCounts.get(name) ?? 0) + 1);
    }
  });

  for (const [index, repo] of guidedRepos.entries()) {
    const repoKey = repo.name.trim() || `Repo ${index + 1}`;
    const duplicateName = repo.name.trim() && (nameCounts.get(repo.name.trim().toLowerCase()) ?? 0) > 1;
    const sourceMissing = repo.mode === "path" ? repo.path.trim() === "" : repo.git_url.trim() === "";
    const nameMissing = repo.name.trim() === "";

    if (!nameMissing && !duplicateName && !sourceMissing) {
      continue;
    }

    const diagnosticLabel = nameMissing ? "Repo name is missing" : duplicateName ? "Repo name is duplicated" : "Repository source is missing";
    const message = nameMissing
      ? "This repository needs a stable name before `workspace.yaml` can be saved and validated."
      : duplicateName
        ? "Repository names must be unique before Source can resolve each repo."
        : `${sourceTypeLabel(repo)} is empty, so Source cannot resolve this repository.`;
    const suggestion = nameMissing
      ? "Enter a short unique repository name, then save and validate sources."
      : duplicateName
        ? "Rename one of the duplicate repositories, then save and validate sources."
        : repo.mode === "path"
          ? "Enter the local checkout path, then save and validate sources."
          : "Enter the GitHub/GitLab URL and make sure local git authentication can reach it.";

    return {
      repoKey,
      diagnosticLabel,
      level: "draft",
      message,
      suggestion,
      sourceType: sourceTypeLabel(repo),
      sourceValue: sourceValueLabel(repo),
      refValue: repo.ref || "current/default",
    };
  }

  return null;
}

export function sourceTypeLabel(repo?: GuidedRepo): string {
  if (!repo) {
    return "Workspace manifest";
  }
  return repo.mode === "path" ? "Local folder" : "Git URL";
}

export function sourceValueLabel(repo?: GuidedRepo, workspace?: string): string {
  if (!repo) {
    return workspace || "workspace.yaml";
  }
  const sourceValue = repo.mode === "path" ? repo.path : repo.git_url;
  return sourceValue.trim() || `${sourceTypeLabel(repo)} missing`;
}

export function defaultSourceSuggestion(repo?: GuidedRepo): string {
  if (!repo) {
    return "Fix the workspace manifest entry, then save and validate sources again.";
  }
  return repo.mode === "path"
    ? "Check the local checkout path and filesystem access, then save and validate sources again."
    : "Check the repository URL and your local git authentication, then save and validate sources again.";
}
