import type { ArchitectureLevel } from "./appContracts";

export function levelLabel(level: ArchitectureLevel): string {
  return level === "context" ? "System context" : level.charAt(0).toUpperCase() + level.slice(1);
}
