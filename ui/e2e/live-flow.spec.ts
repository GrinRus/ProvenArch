import { expect, test, type APIRequestContext, type Locator, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const qaSmoke = (process.env.UI_E2E_QA_SMOKE ?? "0").trim() === "1";
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const initTimeoutSec = Number.parseInt(process.env.ACP_UI_INIT_POLL_TIMEOUT_SEC ?? "900", 10);
const initTimeoutMs = Number.isFinite(initTimeoutSec) && initTimeoutSec > 0 ? initTimeoutSec * 1000 : 900_000;

type RunStatusPollResponse = {
  status?: string;
  error_code?: string | null;
  current_step?: string;
  warnings?: string[] | null;
};

type RunArtifactsPollResponse = {
  artifacts?: unknown[];
};

type RunObservation = {
  status: string;
  errorCode: string;
  currentStep: string;
  warningsCount: number;
  artifactCount: number;
};

async function sleep(ms: number): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchRunObservation(api: APIRequestContext, runID: string): Promise<RunObservation> {
  const response = await api.get(`/api/pipeline/runs/${runID}`);
  const payload = (await response.json()) as RunStatusPollResponse;
  let artifactCount = 0;
  const artifactsResponse = await api.get(`/api/pipeline/runs/${runID}/artifacts`);
  if (artifactsResponse.ok()) {
    const artifactsPayload = (await artifactsResponse.json()) as RunArtifactsPollResponse;
    artifactCount = Array.isArray(artifactsPayload.artifacts) ? artifactsPayload.artifacts.length : 0;
  }
  return {
    status: (payload.status ?? "").trim(),
    errorCode: (payload.error_code ?? "").trim(),
    currentStep: (payload.current_step ?? "").trim(),
    warningsCount: Array.isArray(payload.warnings) ? payload.warnings.length : 0,
    artifactCount
  };
}

async function fetchArtifactText(api: APIRequestContext, artifactPath: string): Promise<string> {
  const response = await api.get(`/api/artifacts?path=${encodeURIComponent(artifactPath)}`);
  expect(response.ok(), `artifact API should return ${artifactPath}`).toBe(true);
  return await response.text();
}

function expectReadableArtifactText(artifactPath: string, content: string, minCharacters = 320): void {
  const text = content.trim();
  expect(text, `${artifactPath} should not be empty`).not.toBe("");
  expect(text.length, `${artifactPath} should contain evidence-backed content`).toBeGreaterThanOrEqual(minCharacters);
  expect(text, `${artifactPath} should not be a bootstrap/scaffold artifact`).not.toMatch(
    /(Provider wrote|Drafted required runtime artifacts|Select artifact to inspect|Loading\.\.\.|No findings reported\.|No proposals yet|No changelog yet)/i
  );
}

function expectReadableDiagramArtifact(artifactPath: string, content: string): void {
  const text = content.trim();
  expect(text, `${artifactPath} should contain Mermaid content`).toMatch(/^(flowchart|graph)\b|```mermaid/i);
  expect(text, `${artifactPath} should not be gap-only C4 output`).toMatch(/(Service:|Datastore:|External:|Actor:|-->|---|\buses\b|\bcalls\b)/i);
  expect(text, `${artifactPath} should not only describe missing diagram evidence`).not.toMatch(
    /Gap: no evidence-backed (services|containers|relationships|actors|external systems)[\s\S]*Gap: no evidence-backed/i
  );
}

async function expectReadableViewportPanel(
  page: Page,
  locator: Locator,
  label: string,
  minVisibleHeight = 140,
  minVisibleWidth = 240
): Promise<void> {
  await expect(locator, `${label} should be visible`).toBeVisible();
  await expect
    .poll(
      async () => {
        await locator.scrollIntoViewIfNeeded();
        const viewport = page.viewportSize();
        const box = await locator.boundingBox();
        if (!viewport || !box) {
          return 0;
        }
        const visibleWidth = Math.max(0, Math.min(box.x + box.width, viewport.width) - Math.max(box.x, 0));
        const visibleHeight = Math.max(0, Math.min(box.y + box.height, viewport.height) - Math.max(box.y, 0));
        const requiredWidth = Math.min(minVisibleWidth, Math.max(120, viewport.width - 24));
        return visibleWidth >= requiredWidth ? visibleHeight : 0;
      },
      { timeout: 10_000, message: `${label} should have readable viewport area` }
    )
    .toBeGreaterThanOrEqual(minVisibleHeight);
}

function observationShowsProductiveProgress(previous: RunObservation | null, current: RunObservation): boolean {
  if (current.status === "running" && current.currentStep !== "") {
    if (previous === null) {
      return true;
    }
    if (current.currentStep !== previous.currentStep) {
      return true;
    }
  }
  if (previous === null) {
    return current.artifactCount > 0;
  }
  return current.artifactCount > previous.artifactCount || current.warningsCount > previous.warningsCount;
}

async function waitForInitInspectRun(api: APIRequestContext, runID: string): Promise<void> {
  const initDeadline = Date.now() + initTimeoutMs;
  let lastObservation: RunObservation | null = null;
  let sawProductiveProgress = false;
  while (Date.now() < initDeadline) {
    const observation = await fetchRunObservation(api, runID);
    if (observationShowsProductiveProgress(lastObservation, observation)) {
      sawProductiveProgress = true;
    }
    if (observation.status === "succeeded") {
      return;
    }
    if (observation.status === "failed") {
      throw new Error(
        `run ${runID} terminated before inspect stage: status=failed error_code=${observation.errorCode || "-"} current_step=${observation.currentStep || "-"}`
      );
    }
    lastObservation = observation;
    await sleep(500);
  }

  if (sawProductiveProgress) {
    throw new Error(
      `ACTIVE_RUN_TIMEOUT: run ${runID} stayed productive but did not reach succeeded within ${initTimeoutSec}s current_step=${lastObservation?.currentStep || "-"} artifact_count=${lastObservation?.artifactCount ?? 0}`
    );
  }
  throw new Error(`run ${runID} did not reach succeeded within ${initTimeoutSec}s`);
}

async function captureEvidenceScreenshot(page: Page, name: string): Promise<string | null> {
  if (screenshotOutputDir === "") {
    return null;
  }
  mkdirSync(screenshotOutputDir, { recursive: true });
  const screenshotPath = path.join(screenshotOutputDir, name);
  await page.screenshot({ path: screenshotPath, fullPage: true });
  await test.info().attach(name, {
    path: screenshotPath,
    contentType: "image/png"
  });
  return screenshotPath;
}

async function expectHiddenCompatibilityControlsAbsent(page: Page): Promise<void> {
  await expect(page.getByTestId("tab-settings")).toHaveCount(0);
  await expect(page.getByTestId("setup-stepper")).toHaveCount(0);
}

async function expectOperatorInspectorSurfaces(page: Page): Promise<void> {
  await expect(page.getByTestId("blockers-panel")).toBeVisible();
  await expect(page.getByTestId("evidence-refs-panel")).toBeVisible();
  await expect(page.getByTestId("runtime-safety-panel")).toBeVisible();
  await expect(page.getByTestId("git-publication-panel")).toBeVisible();
}

async function expectAlreadyInitializedWorkspaceNavigation(page: Page): Promise<void> {
  await page.reload();
  await expect(page.getByTestId("review-panel")).toBeVisible();
  await expect(page.getByTestId("stage-review")).toHaveAttribute("aria-current", "step");

  await page.getByTestId("stage-source").click();
  await expect(page.getByTestId("source-repo-table")).toBeVisible();
  await expectOperatorInspectorSurfaces(page);

  await page.getByTestId("stage-readiness").click();
  await expect(page.getByTestId("readiness-summary-cards")).toBeVisible();
  await expect(page.getByTestId("readiness-runtime-summary")).toBeVisible();
  await expectOperatorInspectorSurfaces(page);

  await page.getByTestId("stage-analysis").click();
  await expect(page.getByTestId("analysis-run-progress")).toBeVisible();
  await expect(page.getByTestId("analysis-run-timeline")).toBeVisible();
  await expectOperatorInspectorSurfaces(page);

  await page.getByTestId("stage-review").click();
  await expect(page.getByTestId("review-panel")).toBeVisible();
  await expect(page.getByTestId("review-artifact-explorer")).toBeVisible();
  await expectOperatorInspectorSurfaces(page);

  await page.getByTestId("stage-publish").click();
  await expect(page.getByTestId("publish-panel")).toBeVisible();
  await expect(page.getByTestId("publish-gate-panel")).toBeVisible();
  await expectOperatorInspectorSurfaces(page);
}

test("live ui flow: validate -> run init -> inspect artifacts", async ({ page, request }) => {
  test.skip(scenario !== "init-inspect", `scenario ${scenario} skips init-inspect flow`);
  test.setTimeout(Math.max(initTimeoutMs + 120_000, 6 * 60 * 1000));
  const runtimeProvider = process.env.UI_E2E_RUNTIME_PROVIDER ?? "unknown";
  const expectedRepoCountRaw = Number.parseInt(process.env.UI_E2E_EXPECTED_REPO_COUNT ?? "1", 10);
  const expectedRepoCount = Number.isFinite(expectedRepoCountRaw) && expectedRepoCountRaw > 0 ? expectedRepoCountRaw : 1;

  await page.goto("/");
  await expect(page.getByTestId("console-shell")).toBeVisible();
  await expect(page.getByTestId("top-status-bar")).toContainText("Proven Arch");
  await expect(page.getByTestId("brand-version")).not.toHaveText(/v0\.1\.1 beta/i);
  await expect(page.getByTestId("brand-version")).toHaveText(/^(dev|v?\d|\w)/);
  await expect(page.getByTestId("stage-rail")).toBeVisible();
  await expect(page.getByTestId("right-inspector")).toBeVisible();
  await expect(page.getByTestId("activity-drawer")).toBeVisible();
  await page.getByTestId("stage-source").click();
  await expect(page.getByTestId("source-repo-table")).toBeVisible();
  await expectHiddenCompatibilityControlsAbsent(page);
  await expectOperatorInspectorSurfaces(page);
  await captureEvidenceScreenshot(page, "frontend-source-desktop.png");

  await page.getByTestId("stage-readiness").click();
  await expect(page.getByTestId("readiness-summary-cards")).toBeVisible();
  await expect(page.getByTestId("readiness-runtime-summary")).toBeVisible();
  await page.locator("summary").filter({ hasText: /^Advanced runtime settings$/ }).click();
  await expect(page.getByRole("heading", { name: "Settings: Runtime Timeouts" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Settings: Runtime Execution" })).toBeVisible();

  await page.getByTestId("stage-readiness").click();
  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();
  const resolvedRepoRows = page.getByTestId("workspace-validate-resolved-repos").locator("li");
  await expect(resolvedRepoRows).toHaveCount(expectedRepoCount);
  await expectOperatorInspectorSurfaces(page);
  await captureEvidenceScreenshot(page, "frontend-readiness-desktop.png");

  await page.getByTestId("stage-analysis").click();
  await expect(page.getByTestId("analysis-run-progress")).toBeVisible();
  await page.getByTestId("run-init-btn").click();
  await expect(page.getByTestId("run-status-panel")).toBeVisible();
  const runID = ((await page.getByTestId("run-status-run-id").textContent()) ?? "").trim();
  expect(runID).not.toBe("");
  console.log(`ACP_UI_E2E_RUN_ID=${runID}`);
  await test.info().attach("run-id", {
    body: runID,
    contentType: "text/plain"
  });
  await waitForInitInspectRun(request, runID);

  await page.getByTestId("stage-analysis").click();
  await expect(page.getByTestId("analysis-run-progress")).toBeVisible();
  const selectedRunButton = page.getByRole("button", { name: runID }).first();
  await expect(selectedRunButton).toBeVisible();
  await selectedRunButton.click();

  await page.getByTestId("run-logs-mode-select").selectOption("events");
  const logsContent = page.getByTestId("run-logs-content");
  await expect
    .poll(async () => (await logsContent.textContent()) ?? "", { timeout: 30_000 })
    .toContain("[EVENT]");

  await page.getByTestId("run-logs-mode-select").selectOption("raw");
  await expect
    .poll(
      async () => {
        const hasLogsContent = (await logsContent.count()) > 0;
        const content = hasLogsContent ? ((await logsContent.textContent()) ?? "") : "";
        const noLogsVisible = (await page.getByText("No run logs yet.").count()) > 0;
        return content.includes("[RAW]") || noLogsVisible;
      },
      { timeout: 30_000 }
    )
    .toBe(true);

  await page.getByTestId("run-logs-mode-select").selectOption("all");
  await expect(page.getByTestId("activity-events-table")).toBeVisible();
  await page.getByTestId("stage-analysis").click();
  await expect(page.getByTestId("analysis-run-progress")).toBeVisible();
  await expect(page.getByTestId("analysis-run-timeline")).toBeVisible();
  await expectOperatorInspectorSurfaces(page);
  await captureEvidenceScreenshot(page, "frontend-analysis-desktop.png");

  await page.getByTestId("stage-review").click();
  await expect(page.getByTestId("review-panel")).toBeVisible();
  await expect(page.getByTestId("review-artifact-explorer")).toBeVisible();
  await expect(page.getByTestId("review-evidence-preview")).toBeVisible();
  await expect(page.getByTestId("review-citation-coverage")).toBeVisible();
  const reviewArtifactExplorer = page.getByTestId("review-artifact-explorer");
  const diagramButtons = reviewArtifactExplorer.getByRole("button", { name: /reports\/diagrams\//i });
  await expect(diagramButtons.first()).toBeVisible();

  const c4ContextButton = reviewArtifactExplorer.getByRole("button", { name: /reports\/diagrams\/c4-context\.mmd/i });
  if ((await c4ContextButton.count()) > 0) {
    await c4ContextButton.first().click();
  } else {
    await diagramButtons.first().click();
  }

  const diagramPanel = page.getByTestId("run-diagram-content-panel");
  await expect(diagramPanel).toBeVisible();
  const selectedDiagramPath = page.getByTestId("run-diagram-selected-path");
  await expect
    .poll(async () => ((await selectedDiagramPath.textContent()) ?? "").trim(), { timeout: 30_000 })
    .toMatch(/reports\/diagrams\//i);
  const selectedDiagramPathText = ((await selectedDiagramPath.textContent()) ?? "").trim();
  const selectedDiagramRaw = await fetchArtifactText(request, selectedDiagramPathText);
  expectReadableDiagramArtifact(selectedDiagramPathText, selectedDiagramRaw);

  await expect
    .poll(
      async () => {
        const svgVisible = (await diagramPanel.locator(".diagram-svg svg").count()) > 0;
        const renderingVisible = (await diagramPanel.getByText(/Rendering/i).count()) > 0;
        const renderErrorVisible = (await diagramPanel.getByText(/Diagram render error:/i).count()) > 0;
        const plainTextLocator = page.getByTestId("run-diagram-content");
        const plainText =
          (await plainTextLocator.count()) > 0 ? (((await plainTextLocator.textContent()) ?? "").trim()) : "";
        return (
          svgVisible ||
          renderingVisible ||
          renderErrorVisible ||
          (plainText !== "" && !/^Select a `\.mmd` diagram artifact to preview\.$/.test(plainText))
        );
      },
      { timeout: 30_000 }
    )
    .toBe(true);
  await expectReadableViewportPanel(page, diagramPanel, "Review Mermaid/C4 preview");

  const preferredReadableArtifactButton =
    (await reviewArtifactExplorer.getByRole("button", { name: /reports\/as-is\/overview\.md/i }).count()) > 0
      ? reviewArtifactExplorer.getByRole("button", { name: /reports\/as-is\/overview\.md/i }).first()
      : (await reviewArtifactExplorer.getByRole("button", { name: /reports\/coverage\/summary\.md/i }).count()) > 0
        ? reviewArtifactExplorer.getByRole("button", { name: /reports\/coverage\/summary\.md/i }).first()
        : reviewArtifactExplorer.locator("button.link-button").first();
  await expect(preferredReadableArtifactButton).toBeVisible();
  await preferredReadableArtifactButton.click();

  const artifactContent = page.getByTestId("run-artifact-content");
  await expect(artifactContent).toBeVisible();
  await expect
    .poll(async () => (await artifactContent.textContent())?.trim() ?? "")
    .not.toMatch(/^(Select artifact to inspect\.|Loading\.\.\.)$/);
  const selectedArtifactPath = ((await page.getByTestId("run-artifact-selected-path").textContent()) ?? "").trim();
  const selectedArtifactText = ((await artifactContent.textContent()) ?? "").trim();
  expectReadableArtifactText(selectedArtifactPath, selectedArtifactText);
  await expectReadableViewportPanel(page, page.getByTestId("run-artifact-content-panel"), "Review artifact preview panel");
  await expectReadableViewportPanel(page, artifactContent, "Review artifact text preview", 120);

  await expectOperatorInspectorSurfaces(page);
  await captureEvidenceScreenshot(page, "frontend-review-desktop.png");

  if (qaSmoke) {
    await page.getByTestId("stage-ask").click();
    await expect(page.getByTestId("qa-run-history")).toBeVisible();
    await expect(page.getByTestId("qa-readonly-safety-panel")).toContainText("no canonical writes");
    await page.getByTestId("qa-question-input").fill("What are the main architecture coverage gaps?");
    await page.getByTestId("qa-ask-btn").click();

    await expect
      .poll(async () => ((await page.getByTestId("qa-run-status").textContent()) ?? "").trim(), { timeout: 120_000 })
      .toMatch(/status:\s*succeeded/i);
    await expect(page.getByTestId("qa-answer-panel")).toBeVisible();
    await expect(page.getByTestId("qa-citations-panel")).toBeVisible();
    await expect(page.getByTestId("qa-answer")).toBeVisible();
    await expect(page.getByTestId("qa-answer")).not.toContainText("No citations returned.");
    await expect(page.getByTestId("qa-citations-panel")).not.toContainText("No citations returned.");
    await expect(page.getByTestId("qa-readonly-safety-panel")).toContainText("reports/taskruns/<run_id>/qa/");
    await expect(page.getByTestId("qa-answer")).toContainText(/Confidence:\s*[1-9][0-9]*%/);
    await expect(page.getByRole("button", { name: /context-pack\.json/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /runtime-execution\.json/i })).toBeVisible();
    await captureEvidenceScreenshot(page, "frontend-ask-desktop.png");
    await page.getByTestId("stage-review").click();
  }

  await page.getByTestId("stage-publish").click();
  await expect(page.getByTestId("publish-panel")).toBeVisible();
  await expect(page.getByTestId("publish-diff-summary")).toBeVisible();
  await expect(page.getByTestId("publish-preview-panel")).toBeVisible();
  await expect(page.getByTestId("publish-preview-tabs")).toBeVisible();
  await expect(page.getByTestId("publish-gate-panel")).toBeVisible();
  await expect(page.getByTestId("publish-commit-plan")).toBeVisible();
  const publishPreviewContent = page.getByTestId("publish-selected-preview-content");
  await expect(publishPreviewContent).toBeVisible();
  await expect
    .poll(async () => (await publishPreviewContent.textContent())?.trim() ?? "", { timeout: 30_000 })
    .not.toMatch(/^(Select an artifact to load its preview in this Publish room\.|Loading\.\.\.)$/);
  expectReadableArtifactText("Publish selected artifact preview", ((await publishPreviewContent.textContent()) ?? "").trim(), 240);
  await expectReadableViewportPanel(page, page.getByTestId("publish-preview-panel"), "Publish preview panel");
  await expectReadableViewportPanel(page, publishPreviewContent, "Publish selected artifact preview", 120);
  await expect(page.getByTestId("git-publication-panel")).toContainText("proposal/beta-refresh");
  await expectOperatorInspectorSurfaces(page);
  await captureEvidenceScreenshot(page, "frontend-publish-desktop.png");

  await expectAlreadyInitializedWorkspaceNavigation(page);

  await page.getByTestId("stage-review").click();
  await page.setViewportSize({ width: 390, height: 1200 });
  await expect(page.getByTestId("review-panel")).toBeVisible();
  const mobileArtifactContent = page.getByTestId("run-artifact-content");
  await expect
    .poll(async () => (await mobileArtifactContent.textContent())?.trim() ?? "", { timeout: 30_000 })
    .not.toMatch(/^(Select artifact to inspect\.|Loading\.\.\.)$/);
  await expectReadableViewportPanel(page, page.getByTestId("review-evidence-preview"), "Mobile Review evidence preview", 180, 300);
  await expectReadableViewportPanel(page, mobileArtifactContent, "Mobile Review artifact text", 140, 300);
  const mobileBodyText = (((await page.locator("body").innerText()) ?? "").replace(/\s+/g, ""));
  expect(mobileBodyText).not.toContain("SetupBaselineRunsResultsSettingsCoverageArtifactsDiagrams");
  await captureEvidenceScreenshot(page, "frontend-review-mobile.png");

  await expect(page.locator("p.status.err")).toHaveCount(0);
  await test.info().attach("runtime-provider", {
    body: runtimeProvider,
    contentType: "text/plain"
  });
});
