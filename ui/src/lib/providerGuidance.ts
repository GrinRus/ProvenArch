export function providerCommandHint(provider: string): string {
  switch (provider) {
    case "qwen-code":
      return "qwen";
    case "codex-code":
      return "codex";
    case "claude-code":
      return "claude or claude-code";
    default:
      return provider || "provider command";
  }
}

export function providerCommandEnv(provider: string): string {
  switch (provider) {
    case "qwen-code":
      return "ACP_QWEN_CMD";
    case "codex-code":
      return "ACP_CODEX_CMD";
    case "claude-code":
      return "ACP_CLAUDE_CMD";
    default:
      return "ACP_RUNTIME_PROVIDER";
  }
}

export type ProviderReadinessGuidance = {
  failureMode: string;
  probeStage: string;
  operatorFocus: string;
  nextActions: string[];
};

export function providerReadinessGuidance(provider: string, message?: string | null): ProviderReadinessGuidance {
  const providerCommand = providerCommandHint(provider);
  const envOverride = providerCommandEnv(provider);
  const normalized = (message ?? "").toLowerCase();

  if (normalized.includes("headless_probe_timeout") || normalized.includes("headless probe timed out")) {
    return {
      failureMode: "Headless probe timeout",
      probeStage: "Text readiness probe",
      operatorFocus: `${providerCommand} did not return the readiness response before the bounded preflight window. Treat this as provider/auth/quota latency, not an ACP artifact failure.`,
      nextActions: [
        `Verify ${providerCommand} can answer a short headless prompt outside ACP without an interactive login or quota prompt.`,
        `Use ${envOverride} if the working executable is outside PATH, then restart the ACP service.`,
        "Run Check local readiness again; retry Analysis only after Runtime provider passes.",
      ],
    };
  }

  if (normalized.includes("artifact") && (normalized.includes("smoke") || normalized.includes("sentinel") || normalized.includes("write"))) {
    return {
      failureMode: "Artifact smoke failed",
      probeStage: "Temp write smoke",
      operatorFocus: `${providerCommand} was reachable but did not prove it can write the sentinel artifact in the temporary ACP write surface.`,
      nextActions: [
        "Confirm the provider CLI can run non-interactively and write files in the current user session.",
        `Use ${envOverride} if ACP should call a different provider executable.`,
        "Run Check local readiness again before retrying the analysis pipeline.",
      ],
    };
  }

  if (normalized.includes("quota") || normalized.includes("usage limit") || normalized.includes("permission") || normalized.includes("auth") || normalized.includes("login")) {
    return {
      failureMode: "Auth or quota blocker",
      probeStage: "Provider request",
      operatorFocus: `${providerCommand} rejected or could not complete the readiness request. Resolve account, login, permission or quota state before retrying.`,
      nextActions: [
        `Check ${providerCommand} login/account state and quota outside ACP.`,
        `Use ${envOverride} if the authenticated executable is not the one ACP finds on PATH.`,
        "Run Check local readiness again, then return to Analysis after the provider check passes.",
      ],
    };
  }

  if (normalized.includes("not found") || normalized.includes("executable") || normalized.includes("no such file") || normalized.includes("command")) {
    return {
      failureMode: "Command unavailable",
      probeStage: "Binary discovery",
      operatorFocus: `ACP could not use ${providerCommand} as a runnable headless provider command in this service environment.`,
      nextActions: [
        `Install ${providerCommand} or point ${envOverride} at the working executable.`,
        "Restart the ACP service so the provider command environment is picked up.",
        "Run Check local readiness before starting the first live analysis.",
      ],
    };
  }

  return {
    failureMode: "Provider readiness blocker",
    probeStage: "Local readiness",
    operatorFocus: `ACP needs ${providerCommand} to pass binary, auth/quota and write-surface checks before live analysis.`,
    nextActions: [
      `Confirm ${providerCommand} is installed, logged in and available to the ACP service.`,
      `Set ${envOverride} when ACP should use a custom executable path.`,
      "Run Check local readiness again before retrying the analysis pipeline.",
    ],
  };
}
