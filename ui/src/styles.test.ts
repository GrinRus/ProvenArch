import { describe, expect, it } from "vitest";

import css from "./styles.css?raw";
import tokens from "./styles/tokens.css?raw";

describe("styles.css tokens", () => {
  it("does not reference undefined custom properties", () => {
    const source = `${tokens}\n${css}`;
    const defined = new Set(Array.from(source.matchAll(/--[A-Za-z0-9-]+(?=\s*:)/g), (match) => match[0]));
    const referenced = Array.from(source.matchAll(/var\(\s*(--[A-Za-z0-9-]+)/g), (match) => match[1]);
    const missing = [...new Set(referenced.filter((token) => !defined.has(token)))].sort();

    expect(missing).toEqual([]);
  });
});
