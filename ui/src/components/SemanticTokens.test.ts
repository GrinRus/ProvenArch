import { describe, expect, it } from "vitest";
import css from "../styles.css?raw";
import tokens from "../styles/tokens.css?raw";
import primitiveSource from "./SemanticPrimitives.tsx?raw";

describe("semantic token contract", () => {
  it("defines the accepted role, spacing, radius and motion aliases", () => {
    const source = `${tokens}\n${css}`;
    for (const token of ["--color-bg-canvas", "--color-text-primary", "--color-border-default", "--color-action-primary", "--color-state-danger", "--color-context-snapshot", "--space-9", "--radius-sheet", "--duration-slow"]) {
      expect(source).toContain(`${token}:`);
    }
  });

  it("keeps raw palette values out of the shared primitive implementation", () => {
    expect(primitiveSource).not.toMatch(/#[0-9a-f]{3,8}\b/i);
    expect(primitiveSource).not.toMatch(/rgb\(/i);
  });
});
