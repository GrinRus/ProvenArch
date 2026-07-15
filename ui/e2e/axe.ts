import AxeBuilder from "@axe-core/playwright";
import { expect, type Page } from "@playwright/test";

export async function expectNoCriticalAxeViolations(page: Page): Promise<void> {
  const results = await new AxeBuilder({ page }).analyze();
  expect(results.violations.filter((violation) => violation.impact === "critical")).toEqual([]);
}
