export function runtimeDisplayLabel(runtimeMode: string, runtimeProvider: string, options: { compact?: boolean } = {}): string {
  if (runtimeMode === "fake") {
    return options.compact ? "fake" : "fake baseline";
  }
  const provider = runtimeProvider.trim() || "provider pending";
  return options.compact ? provider : `headless / ${provider}`;
}

export function providerDisplayLabel(runtimeMode: string, provider: string | null | undefined): string {
  if (runtimeMode === "fake") {
    return "fake";
  }
  return provider?.trim() || "provider pending";
}
