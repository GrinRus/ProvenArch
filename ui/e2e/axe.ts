import AxeBuilder from "@axe-core/playwright";
import { expect, type Page } from "@playwright/test";

/**
 * Keep the shared browser gate aligned with the remediation acceptance
 * boundary: critical and serious violations both block a journey.  Axe's
 * serious rules include the contrast/label/landmark failures that the old
 * critical-only helper silently ignored.
 */
export async function expectNoCriticalAxeViolations(page: Page): Promise<void> {
  const results = await new AxeBuilder({ page }).analyze();
  const blocking = results.violations.filter((violation) => violation.impact === "critical" || violation.impact === "serious");
  expect(blocking, blocking.map((violation) => `${violation.id}: ${violation.help}`).join("\n")).toEqual([]);
}

export async function expectReducedMotionRespect(page: Page): Promise<void> {
  await page.emulateMedia({ reducedMotion: "reduce" });
  const activeMotion = await page.evaluate(() => Array.from(document.querySelectorAll<HTMLElement>("*"))
    .flatMap((element) => {
      const style = getComputedStyle(element);
      const durations = [...style.transitionDuration.split(","), ...style.animationDuration.split(",")]
        .map((value) => value.trim())
        .map((value) => value.endsWith("ms") ? Number.parseFloat(value) : Number.parseFloat(value) * 1000)
        .filter((value) => Number.isFinite(value));
      return durations.some((value) => value > 0.01) ? [element.tagName.toLowerCase()] : [];
    }));
  expect(activeMotion, `reduced-motion left active transitions/animations on: ${activeMotion.join(", ")}`).toEqual([]);
  await page.emulateMedia({ reducedMotion: null });
}
