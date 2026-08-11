import { expect, test, type APIRequestContext, type Locator, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { evaluateDiagramArtifactReadability } from "./liveArtifactQuality";
import { expectNoCriticalAxeViolations } from "./axe";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const qaSmoke = (process.env.UI_E2E_QA_SMOKE ?? "0").trim() === "1";
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const artifactSource = (process.env.UI_E2E_ARTIFACT_SOURCE ?? "live").trim().toLowerCase();
const snapshotRunID = (process.env.UI_E2E_SNAPSHOT_RUN_ID ?? "").trim();
const initTimeoutSec = Number.parseInt(process.env.ACP_UI_INIT_POLL_TIMEOUT_SEC ?? "900", 10);
const initTimeoutMs = Number.isFinite(initTimeoutSec) && initTimeoutSec > 0 ? initTimeoutSec * 1000 : 900_000;
const qaPollTimeoutSec = Number.parseInt(process.env.ACP_UI_QA_POLL_TIMEOUT_SEC ?? "300", 10);
const qaPollTimeoutMs = Number.isFinite(qaPollTimeoutSec) && qaPollTimeoutSec > 0 ? qaPollTimeoutSec * 1000 : 300_000;

type RunStatusPollResponse = {
  status?: string;
  error_code?: string | null;
  current_step?: string;
  warnings?: string[] | null;
};

type RunArtifactsPollResponse = {
  artifacts?: unknown[];
};

type WorkspaceValidateResponse = {
  ok?: boolean;
  resolved_repos?: Array<{ name?: string; path?: string }>;
};

type ProductTask = {
  task_id?: string;
  title?: string;
  goal?: string;
  scope?: { repositories?: Array<{ name?: string; paths?: string[] }> };
};

type TaskAttempt = {
  attempt_id?: string;
  task_id?: string;
  run_id?: string;
  status?: string;
};

type TaskListResponse = { items?: ProductTask[]; next_cursor?: string; has_more?: boolean };
type TaskAttemptListResponse = { items?: TaskAttempt[] };

type RunObservation = {
  status: string;
  errorCode: string;
  currentStep: string;
  warningsCount: number;
  artifactCount: number;
};

type QARunPollResponse = {
  status?: string;
  error_code?: string | null;
  current_step?: string;
  warnings?: string[] | null;
};

async function sleep(ms: number): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchRunObservation(api: APIRequestContext, runID: string): Promise<RunObservation> {
  const response = await api.get(`/api/pipeline/runs/${runID}`);
  expect(response.ok(), `run status API should return ${runID}`).toBe(true);
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

async function fetchQARunObservation(api: APIRequestContext, runID: string): Promise<RunObservation> {
  const response = await api.get(`/api/qa/runs/${runID}`);
  expect(response.ok(), `qa run status API should return ${runID}`).toBe(true);
  const payload = (await response.json()) as QARunPollResponse;
  return {
    status: (payload.status ?? "").trim(),
    errorCode: (payload.error_code ?? "").trim(),
    currentStep: (payload.current_step ?? "").trim(),
    warningsCount: Array.isArray(payload.warnings) ? payload.warnings.length : 0,
    artifactCount: 0
  };
}

async function resolveTaskAttemptForRun(api: APIRequestContext, requestedRunID = ""): Promise<{ taskID: string; attemptID: string; runID: string }> {
  const deadline = Date.now() + 30_000;
  let lastPayload: unknown = null;
  while (Date.now() < deadline) {
    const response = await api.get("/api/tasks?limit=100");
    expect(response.ok(), "Task API should be available in snapshot mode").toBe(true);
    const payload = (await response.json()) as TaskListResponse;
    lastPayload = payload;
    for (const task of Array.isArray(payload.items) ? payload.items : []) {
      if (!task.task_id) continue;
      const attemptsResponse = await api.get(`/api/tasks/${encodeURIComponent(task.task_id)}/attempts`);
      if (!attemptsResponse.ok()) continue;
      const attemptsPayload = (await attemptsResponse.json()) as TaskAttemptListResponse;
      const matching = (Array.isArray(attemptsPayload.items) ? attemptsPayload.items : []).filter(
        (attempt) => attempt.task_id === task.task_id && attempt.status === "succeeded" && Boolean(attempt.attempt_id) && Boolean(attempt.run_id) && (!requestedRunID || attempt.run_id === requestedRunID)
      );
      if (matching.length === 1) {
        return { taskID: task.task_id, attemptID: matching[0].attempt_id ?? "", runID: matching[0].run_id ?? "" };
      }
      if (matching.length > 1) {
        throw new Error(`Task API returned multiple succeeded Attempts for exact run ${requestedRunID || "<any>"} under Task ${task.task_id}`);
      }
    }
    await sleep(250);
  }
  throw new Error(
    `snapshot mode could not find one succeeded public Task/Attempt for run_id=${requestedRunID || "<any>"}; last_payload=${JSON.stringify(lastPayload)}`
  );
}

async function createAndAdmitLiveTask(api: APIRequestContext, runtimeProvider: string, expectedRepoCount: number): Promise<{ taskID: string; attemptID: string; runID: string }> {
  const validationResponse = await api.post("/api/workspace/validate");
  expect(validationResponse.ok(), "workspace validation API should be available before Task admission").toBe(true);
  const validation = (await validationResponse.json()) as WorkspaceValidateResponse;
  const repositories = (Array.isArray(validation.resolved_repos) ? validation.resolved_repos : [])
    .filter((repo) => typeof repo.name === "string" && repo.name.trim() !== "")
    .map((repo) => ({ name: repo.name?.trim() ?? "", paths: ["."] }));
  expect(repositories, "Task admission should use the validated workspace repository scope").toHaveLength(expectedRepoCount);

  const taskResponse = await api.post("/api/tasks", {
    data: {
      title: "Live E2E Task-first inspection",
      goal: "Inspect the exact immutable Attempt outcome and evidence without a second provider analysis.",
      context: "W25A release-gate journey",
      scope: { repositories },
      desired_runner: { preset: `${runtimeProvider}-default`, mode: "headless", provider: runtimeProvider }
    }
  });
  expect(taskResponse.status(), "Task creation should use the authoritative public API").toBe(201);
  const taskPayload = (await taskResponse.json()) as { task?: ProductTask };
  const taskID = taskPayload.task?.task_id ?? "";
  expect(taskID, "Task creation should return an opaque Task identity").not.toBe("");

  const idempotencyKey = `live-e2e-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  const attemptResponse = await api.post(`/api/tasks/${encodeURIComponent(taskID)}/attempts`, {
    data: { idempotency_key: idempotencyKey, pipeline: "init", intent: "start" }
  });
  expect(attemptResponse.ok(), "Attempt admission should use the authoritative public API").toBe(true);
  const attemptPayload = (await attemptResponse.json()) as { attempt?: TaskAttempt };
  const attempt = attemptPayload.attempt;
  expect(attempt?.attempt_id, "Attempt admission should return an Attempt identity").toBeTruthy();
  expect(attempt?.task_id, "Attempt should retain its parent Task identity").toBe(taskID);
  expect(attempt?.run_id, "Attempt admission should return the exact linked run identity").toBeTruthy();
  return { taskID, attemptID: attempt?.attempt_id ?? "", runID: attempt?.run_id ?? "" };
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
  const readability = evaluateDiagramArtifactReadability(text);
  expect(readability.hasMermaidSyntax, `${artifactPath} should contain Mermaid content`).toBe(true);
  expect(readability.hasConcreteEvidence, `${artifactPath} should not be gap-only C4 output`).toBe(true);
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

async function captureRunFailureScreenshot(page: Page): Promise<void> {
  await captureEvidenceScreenshot(page, "frontend-task-attempt-failed-desktop.png").catch(() => undefined);
}

async function waitForInitInspectRun(api: APIRequestContext, page: Page, runID: string): Promise<void> {
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
      await captureRunFailureScreenshot(page);
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

async function resolveQARunIDFromStatus(page: Page): Promise<string> {
  await expect(page.getByTestId("qa-run-status")).toBeVisible({ timeout: 30_000 });
  let raw = "";
  await expect
    .poll(async () => {
      raw = ((await page.getByTestId("qa-run-status").textContent()) ?? "").trim();
      return raw;
    }, {
      timeout: 30_000,
      message: "QA run status should include a run id"
    })
    .toMatch(/Run\s+run_[A-Za-z0-9_]+\s+status:/);
  const match = raw.match(/Run\s+(run_[A-Za-z0-9_]+)\s+status:/);
  expect(match?.[1], "QA run id should be parseable from status text").toBeTruthy();
  return match?.[1] ?? "";
}

async function cancelRunBestEffort(api: APIRequestContext, runID: string): Promise<void> {
  if (!runID) {
    return;
  }
  await api.post(`/api/pipeline/runs/${runID}/cancel`).catch(() => undefined);
}

async function waitForQARun(api: APIRequestContext, page: Page, runID: string): Promise<void> {
  const qaDeadline = Date.now() + qaPollTimeoutMs;
  let lastObservation: RunObservation | null = null;
  let sawProductiveProgress = false;
  while (Date.now() < qaDeadline) {
    const observation = await fetchQARunObservation(api, runID);
    if (observationShowsProductiveProgress(lastObservation, observation)) {
      sawProductiveProgress = true;
    }
    if (observation.status === "succeeded") {
      return;
    }
    if (observation.status === "failed") {
      await captureEvidenceScreenshot(page, "frontend-ask-failed-desktop.png").catch(() => undefined);
      throw new Error(
        `QA run ${runID} terminated before answer: status=failed error_code=${observation.errorCode || "-"} current_step=${observation.currentStep || "-"}`
      );
    }
    lastObservation = observation;
    await sleep(500);
  }
  await captureEvidenceScreenshot(page, "frontend-ask-timeout-desktop.png").catch(() => undefined);
  const progressLabel = sawProductiveProgress ? "stayed productive but did not reach succeeded" : "did not produce observable progress";
  throw new Error(
    `ACTIVE_RUN_TIMEOUT: qa run ${runID} ${progressLabel} within ${qaPollTimeoutSec}s status=${lastObservation?.status || "-"} current_step=${lastObservation?.currentStep || "-"} warnings=${lastObservation?.warningsCount ?? 0}`
  );
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

async function expectProductShellNavigation(page: Page): Promise<void> {
  await expect(page.getByTestId("product-shell")).toBeVisible();
  for (const destination of ["tasks", "knowledge", "changes"]) {
    await expect(page.getByTestId(`destination-${destination}`)).toBeVisible();
  }
  await expect(page.getByTestId("setup-utility")).toBeVisible();
  await expect(page.getByTestId("stage-ask")).toBeVisible();
  for (const retired of ["stage-rail", "right-inspector", "activity-drawer"]) {
    await expect(page.getByTestId(retired)).toHaveCount(0);
  }
}

async function expectNoDocumentOverflow(page: Page): Promise<void> {
  await expect.poll(async () => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), {
    timeout: 10_000,
    message: "ProductShell should not create global horizontal overflow",
  }).toBe(true);
}

async function expectFirstViewportContent(page: Page, locator: Locator, label: string, maxY = 520): Promise<void> {
  await expect(locator, `${label} should be visible`).toBeVisible();
  const box = await locator.boundingBox();
  expect(box, `${label} should have layout bounds`).not.toBeNull();
  expect(box?.y ?? Number.POSITIVE_INFINITY, `${label} should start in the first viewport`).toBeLessThan(maxY);
}

async function openReviewArtifactExplorer(page: Page): Promise<Locator> {
  const explorer = page.getByTestId("review-artifact-explorer");
  await expect(explorer).toBeVisible();
  const isOpen = await explorer.evaluate((node) => ("open" in node ? Boolean(node.open) : true));
  if (!isOpen) {
    await page.getByTestId("review-artifact-explorer-toggle").click();
  }
  await expect
    .poll(async () => explorer.evaluate((node) => ("open" in node ? Boolean(node.open) : true)), { timeout: 5_000 })
    .toBe(true);
  return explorer;
}

test("live ui flow: validate -> admit Task Attempt -> inspect exact outcome", async ({ page, request }) => {
  test.skip(scenario !== "init-inspect", `scenario ${scenario} skips init-inspect flow`);
  test.setTimeout(Math.max(initTimeoutMs + 120_000, 6 * 60 * 1000));
  const runtimeProvider = process.env.UI_E2E_RUNTIME_PROVIDER ?? "unknown";
  const expectedRepoCountRaw = Number.parseInt(process.env.UI_E2E_EXPECTED_REPO_COUNT ?? "1", 10);
  const expectedRepoCount = Number.isFinite(expectedRepoCountRaw) && expectedRepoCountRaw > 0 ? expectedRepoCountRaw : 1;
  const consoleErrors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") consoleErrors.push(message.text()); });
  page.on("pageerror", (error) => consoleErrors.push(error.message));

  await page.goto("/tasks");
  await expectProductShellNavigation(page);
  await expect(page.getByTestId("task-route-inbox")).toBeVisible();
  await expect(page.getByTestId("top-status-bar")).toContainText("ProvenArch");
  const brandVersion = page.getByTestId("brand-version");
  if (await brandVersion.count() === 0) {
    await page.getByTestId("top-status-bar").getByRole("button", { name: "Details", exact: true }).click();
  }
  await expect(brandVersion).toBeVisible();
  await expect(brandVersion).not.toHaveText(/v0\.1\.1 beta/i);
  await expect(brandVersion).toHaveText(/^(dev|v?\d|\w)/);
  await expectNoDocumentOverflow(page);
  await captureEvidenceScreenshot(page, "frontend-tasks-desktop.png");

  await page.goto("/setup?step=runner");
  await expect(page.getByTestId("guided-setup-page")).toBeVisible();
  await expect(page.getByTestId("stage-readiness")).toHaveAttribute("aria-current", "page");
  await expect(page.getByTestId("readiness-summary-cards")).toBeVisible();
  await expect(page.getByTestId("readiness-runtime-summary")).toBeVisible();
  await expectHiddenCompatibilityControlsAbsent(page);
  const readinessAdvancedSettings = page.getByTestId("readiness-advanced-settings");
  await expect(readinessAdvancedSettings).toBeVisible();
  await expect(readinessAdvancedSettings).not.toHaveAttribute("open", "");
  await page.getByTestId("readiness-advanced-settings").locator("summary").click();
  await expect(page.getByRole("heading", { name: "Settings: Runtime Timeouts" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Settings: Runtime Execution" })).toBeVisible();
  await page.getByTestId("readiness-advanced-settings").locator("summary").click();
  await expect(readinessAdvancedSettings).not.toHaveAttribute("open", "");
  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();
  const resolvedRepoRows = page.getByTestId("workspace-validate-resolved-repos").locator("li");
  await expect(resolvedRepoRows).toHaveCount(expectedRepoCount);
  await captureEvidenceScreenshot(page, "frontend-setup-desktop.png");

  let taskID = "";
  let attemptID = "";
  let runID = "";
  if (artifactSource === "snapshot") {
    const selected = await resolveTaskAttemptForRun(request, snapshotRunID);
    taskID = selected.taskID;
    attemptID = selected.attemptID;
    runID = selected.runID;
    const snapshotObservation = await fetchRunObservation(request, runID);
    expect(snapshotObservation.status, `snapshot run ${runID} should be succeeded`).toBe("succeeded");
  } else {
    const selected = await createAndAdmitLiveTask(request, runtimeProvider, expectedRepoCount);
    taskID = selected.taskID;
    attemptID = selected.attemptID;
    runID = selected.runID;
  }
  expect(taskID).not.toBe("");
  expect(attemptID).not.toBe("");
  expect(runID).not.toBe("");
  console.log(`ACP_UI_E2E_RUN_ID=${runID}`);
  await test.info().attach("run-id", {
    body: runID,
    contentType: "text/plain"
  });
  if (artifactSource !== "snapshot") {
    await waitForInitInspectRun(request, page, runID);
  }

  await page.goto(`/tasks/${encodeURIComponent(taskID)}/attempts/${encodeURIComponent(attemptID)}`);
  await expect(page.getByTestId("task-route-attempt")).toBeVisible();
  await expect(page.getByTestId("task-route-identities")).toContainText(taskID);
  await expect(page.getByTestId("task-route-identities")).toContainText(attemptID);
  await expect(page.getByTestId("task-route-attempt")).toContainText(runID);
  await page.getByTestId("attempt-open-studio").click();
  await expect(page.getByTestId("task-pipeline-studio")).toBeVisible();
  await expect(page.getByTestId("task-pipeline-studio")).toContainText(taskID);
  await expect(page.getByTestId("task-pipeline-studio")).toContainText(attemptID);
  await expect(page.getByTestId("task-pipeline-studio")).toContainText(runID);
  await expectNoDocumentOverflow(page);
  await captureEvidenceScreenshot(page, "frontend-task-attempt-studio-desktop.png");

  await page.reload();
  await expect(page.getByTestId("task-pipeline-studio")).toBeVisible();
  await expect(page.getByTestId("task-pipeline-studio")).toContainText(attemptID);
  await page.goto("/tasks");
  await expect(page.getByTestId("task-route-inbox")).toBeVisible();
  await page.goBack();
  await expect(page.getByTestId("task-pipeline-studio")).toBeVisible();
  await expect(page.getByTestId("task-pipeline-studio")).toContainText(taskID);

  await page.goto(`/knowledge?view=documents&source=current&task=${encodeURIComponent(taskID)}`);
  await expect(page.getByTestId("knowledge-panel")).toBeVisible();
  await expect(page.getByTestId("destination-knowledge")).toHaveAttribute("aria-current", "page");
  await expect(page.getByTestId("knowledge-panel")).toContainText("Current workspace");
  // Architecture is documents-first. Verify the default reader and the
  // adjacent diagrams mode rather than the retired overview/artifacts tabs.
  await expect(page.getByTestId("architecture-documents")).toBeVisible();
  await page.getByRole("button", { name: "Diagrams", exact: true }).click();
  await expect(page.getByTestId("architecture-diagrams")).toBeVisible();
  await page.getByRole("button", { name: "Documents", exact: true }).click();
  await expect(page.getByTestId("architecture-documents")).toBeVisible();
  await expectNoDocumentOverflow(page);
  await captureEvidenceScreenshot(page, "frontend-knowledge-desktop.png");

  await page.goto(`/changes?task=${encodeURIComponent(taskID)}&run=${encodeURIComponent(runID)}&view=evidence&source=snapshot&mode=rendered`);
  await expect(page.getByTestId("changes-page")).toBeVisible();
  await expect(page.getByTestId("destination-changes")).toHaveAttribute("aria-current", "page");
  await expect(page.getByTestId("changes-page")).toContainText(`Run snapshot · ${runID}`);
  await expect(page.getByTestId("stage-evidence")).toHaveAttribute("aria-current", "page");
  await expect(page.getByTestId("review-panel")).toBeVisible();
  await expect(page.getByTestId("review-artifact-explorer")).toBeVisible();
  await expect(page.getByTestId("review-evidence-preview")).toBeVisible();
  await expect(page.getByTestId("review-citation-coverage")).toBeVisible();
  const reviewArtifactExplorer = await openReviewArtifactExplorer(page);
  const diagramsFilter = reviewArtifactExplorer.getByRole("tab", { name: "Diagrams" });
  await expect(diagramsFilter).toBeVisible();
  await diagramsFilter.click();
  const diagramButtons = reviewArtifactExplorer.getByRole("button", { name: /reports\/diagrams\//i });
  if ((await diagramButtons.count()) > 0) {
    const c4ContextButton = reviewArtifactExplorer.getByRole("button", { name: /reports\/diagrams\/c4-context\.mmd/i });
    if ((await c4ContextButton.count()) > 0) {
      await c4ContextButton.first().click();
    } else {
      await diagramButtons.first().click();
    }

    const diagramPanel = page.getByTestId("evidence-viewer");
    await expect(diagramPanel).toBeVisible();
    const selectedDiagramPath = diagramPanel.locator(".evidence-source-header strong");
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
          return svgVisible || renderingVisible || renderErrorVisible;
        },
        { timeout: 30_000 }
      )
      .toBe(true);
    await expectReadableViewportPanel(page, diagramPanel, "Review Mermaid/C4 preview");
  } else {
    await expect(reviewArtifactExplorer.getByRole("tabpanel", { name: "Diagrams" })).toContainText(/No selected-run artifacts/i);
  }

  const reportsFilter = reviewArtifactExplorer.getByRole("tab", { name: "Reports" });
  await expect(reportsFilter).toBeVisible();
  await reportsFilter.click();
  await expect(reviewArtifactExplorer.getByRole("button", { name: /reports\/as-is\/overview\.md/i }).first()).toBeVisible();
  const preferredReadableArtifactButton =
    (await reviewArtifactExplorer.getByRole("button", { name: /reports\/as-is\/overview\.md/i }).count()) > 0
      ? reviewArtifactExplorer.getByRole("button", { name: /reports\/as-is\/overview\.md/i }).first()
      : (await reviewArtifactExplorer.getByRole("button", { name: /reports\/coverage\/summary\.md/i }).count()) > 0
        ? reviewArtifactExplorer.getByRole("button", { name: /reports\/coverage\/summary\.md/i }).first()
        : reviewArtifactExplorer.locator("button.link-button").first();
  await expect(preferredReadableArtifactButton).toBeVisible();
  await preferredReadableArtifactButton.click();

  const artifactViewer = page.getByTestId("evidence-viewer");
  await expect(artifactViewer).toBeVisible();
  const selectedArtifactPath = ((await artifactViewer.locator(".evidence-source-header strong").textContent()) ?? "").trim();
  const selectedArtifactText = await fetchArtifactText(request, selectedArtifactPath);
  expectReadableArtifactText(selectedArtifactPath, selectedArtifactText);
  await expectReadableViewportPanel(page, artifactViewer, "Review artifact preview panel");
  await expectReadableViewportPanel(page, artifactViewer.locator(".markdown-evidence"), "Review artifact text preview", 120);
  await expectNoDocumentOverflow(page);
  await captureEvidenceScreenshot(page, "frontend-changes-evidence-desktop.png");

  const askTrigger = page.getByTestId("stage-ask");
  await askTrigger.focus();
  await askTrigger.click();
  await expect(page.getByRole("dialog", { name: "Ask current workspace" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "Ask current workspace" })).toHaveCount(0);
  await expect(askTrigger).toBeFocused();

  if (qaSmoke) {
    await askTrigger.click();
    await expect(page.getByRole("dialog", { name: "Ask current workspace" })).toBeVisible();
    await expect(page.getByTestId("qa-run-history")).toBeVisible();
    await expect(page.getByTestId("qa-readonly-safety-panel")).toContainText("no canonical writes");
    await page.getByTestId("qa-question-input").fill("What are the main architecture coverage gaps?");
    await page.getByTestId("qa-ask-btn").click();
    const qaRunID = await resolveQARunIDFromStatus(page);
    try {
      await waitForQARun(request, page, qaRunID);
    } catch (error) {
      await cancelRunBestEffort(request, qaRunID);
      throw error;
    }
    await expect
      .poll(async () => ((await page.getByTestId("qa-run-status").textContent()) ?? "").trim(), { timeout: 30_000 })
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
    const citationButton = page.getByTestId("qa-citations-panel").locator("button").first();
    await expect(citationButton).toBeVisible();
    await citationButton.click();
    await expect(page.getByTestId("current-workspace-evidence")).toBeVisible();
    await expect(page.getByRole("button", { name: "Return to Ask" })).toBeVisible();
    await page.getByRole("button", { name: "Return to Ask" }).click();
    await expect(page.getByRole("dialog", { name: "Ask current workspace" })).toBeVisible();
    await page.keyboard.press("Escape");
  }

  await page.goto(`/changes?task=${encodeURIComponent(taskID)}&run=${encodeURIComponent(runID)}&view=publish&source=snapshot&mode=rendered`);
  await expect(page.getByTestId("publish-panel")).toBeVisible();
  await expect(page.getByTestId("stage-publish")).toHaveAttribute("aria-current", "page");
  await expect(page.getByTestId("publish-diff-summary")).toBeVisible();
  await expect(page.getByTestId("publish-readiness-summary")).toBeVisible();
  await expect(page.getByTestId("publish-readiness-summary")).toContainText(/Workspace scope|Publication set/);
  await expect(page.getByTestId("publish-section-jumps")).toContainText("Preview");
  await expect(page.getByTestId("publish-preview-panel")).toBeVisible();
  await expect(page.getByTestId("publish-preview-tabs")).toBeVisible();
  await expect(page.getByTestId("publish-gate-panel")).toBeVisible();
  await expect(page.getByTestId("publish-commit-plan")).toBeVisible();
  const publishArtifactList = page.getByRole("list", { name: "publish artifact preview list" });
  const architectureHomePreview = publishArtifactList.getByRole("button", { name: /reports\/as-is\/overview\.md/i }).first();
  await expect(architectureHomePreview).toBeVisible();
  await architectureHomePreview.click();
  const publishPreviewContent = page.getByTestId("publish-selected-preview-content");
  await expect(publishPreviewContent).toBeVisible();
  await expect
    .poll(async () => (await publishPreviewContent.textContent())?.trim() ?? "", { timeout: 30_000 })
    .not.toMatch(/^(Select an artifact to load its preview in this Publish room\.|Loading\.\.\.)$/);
  expectReadableArtifactText("Publish selected artifact preview", ((await publishPreviewContent.textContent()) ?? "").trim(), 240);
  await expectReadableViewportPanel(page, page.getByTestId("publish-preview-panel"), "Publish preview panel");
  await expectReadableViewportPanel(page, publishPreviewContent, "Publish selected artifact preview", 120);
  await expectNoDocumentOverflow(page);
  await captureEvidenceScreenshot(page, "frontend-changes-publish-desktop.png");

  for (const viewport of [{ width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 }]) {
    await page.setViewportSize(viewport);
    await expect(page.getByTestId("publish-panel")).toBeVisible();
    await expectNoDocumentOverflow(page);
  }

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByTestId("publish-panel")).toBeVisible();
  await expect(page.getByTestId("publish-section-jumps")).toBeVisible();
  await expect(page.getByTestId("publish-section-jumps")).toContainText("Gate");
  await expect(page.getByTestId("publish-readiness-summary")).toBeVisible();
  await expectFirstViewportContent(page, page.getByTestId("publish-readiness-summary"), "Mobile publish state");
  await expectNoDocumentOverflow(page);
  await captureEvidenceScreenshot(page, "frontend-changes-publish-mobile.png");

  await page.goto(`/changes?task=${encodeURIComponent(taskID)}&run=${encodeURIComponent(runID)}&view=evidence&source=snapshot&mode=rendered`);
  await expect(page.getByTestId("review-panel")).toBeVisible();
  await expectFirstViewportContent(page, page.getByTestId("stage-evidence"), "Mobile evidence navigation");
  await expectNoDocumentOverflow(page);
  await captureEvidenceScreenshot(page, "frontend-changes-evidence-mobile.png");

  await expect(page.locator("p.status.err")).toHaveCount(0);
  await expectNoCriticalAxeViolations(page);
  expect(consoleErrors, "live ProductShell should not emit browser console errors").toEqual([]);
  await test.info().attach("runtime-provider", {
    body: runtimeProvider,
    contentType: "text/plain"
  });
});
