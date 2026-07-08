import { describe, expect, it } from "vitest";

import css from "./styles.css?raw";

describe("styles.css tokens", () => {
  it("does not reference undefined custom properties", () => {
    const defined = new Set(Array.from(css.matchAll(/--[A-Za-z0-9-]+(?=\s*:)/g), (match) => match[0]));
    const referenced = Array.from(css.matchAll(/var\(\s*(--[A-Za-z0-9-]+)/g), (match) => match[1]);
    const missing = [...new Set(referenced.filter((token) => !defined.has(token)))].sort();

    expect(missing).toEqual([]);
  });
});
