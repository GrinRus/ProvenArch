export function runtimeDisplayLabel(runtimeMode: string, runtimeProvider: string, options: { compact?: boolean } = {}): string {
  if (runtimeMode === "fake") {
    return options.compact ? "Deterministic demo" : "Deterministic demo runtime";
  }
  const provider = runtimeProvider.trim() || "provider pending";
  return options.compact ? provider : `headless / ${provider}`;
}

export function providerDisplayLabel(runtimeMode: string, provider: string | null | undefined): string {
  if (runtimeMode === "fake") {
    return "Demo evidence";
  }
  return provider?.trim() || "provider pending";
}
