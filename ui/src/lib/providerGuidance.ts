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
