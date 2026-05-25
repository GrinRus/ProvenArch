import { expect, test, type APIRequestContext } from "@playwright/test";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const initTimeoutSec = Number.parseInt(process.env.ACP_UI_INIT_POLL_TIMEOUT_SEC ?? "900", 10);
const cancelTimeoutSec = Number.parseInt(process.env.ACP_UI_CANCEL_POLL_TIMEOUT_SEC ?? "420", 10);
const initTimeoutMs = Number.isFinite(initTimeoutSec) && initTimeoutSec > 0 ? initTimeoutSec * 1000 : 900_000;
const cancelTimeoutMs = Number.isFinite(cancelTimeoutSec) && cancelTimeoutSec > 0 ? cancelTimeoutSec * 1000 : 420_000;

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

test("live ui flow: validate -> run init -> inspect artifacts", async ({ page, request }) => {
  test.skip(scenario !== "init-inspect", `scenario ${scenario} skips init-inspect flow`);
  test.setTimeout(Math.max(initTimeoutMs + 120_000, 6 * 60 * 1000));
  const runtimeProvider = process.env.UI_E2E_RUNTIME_PROVIDER ?? "unknown";
  const expectedRepoCountRaw = Number.parseInt(process.env.UI_E2E_EXPECTED_REPO_COUNT ?? "1", 10);
  const expectedRepoCount = Number.isFinite(expectedRepoCountRaw) && expectedRepoCountRaw > 0 ? expectedRepoCountRaw : 1;

  await page.goto("/");
  await expect(page.getByTestId("console-shell")).toBeVisible();
  await expect(page.getByTestId("top-status-bar")).toContainText("Proven Arch");

  await page.getByTestId("stage-readiness").click();
  await page.getByText("Advanced runtime settings").click();
  await expect(page.getByRole("heading", { name: "Settings: Runtime Timeouts" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Settings: Runtime Execution" })).toBeVisible();

  await page.getByTestId("stage-readiness").click();
  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();
  const resolvedRepoRows = page.getByTestId("workspace-validate-resolved-repos").locator("li");
  await expect(resolvedRepoRows).toHaveCount(expectedRepoCount);

  await page.getByTestId("stage-analysis").click();
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

  await page.getByTestId("stage-review").click();
  const diagramButtons = page.getByTestId("run-diagrams-list").locator("button.link-button");
  await expect(diagramButtons.first()).toBeVisible();

  const c4ContextButton = page.getByRole("button", { name: /reports\/diagrams\/c4-context\.mmd/i });
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

  const firstArtifactButton = page.getByTestId("results-artifacts-panel").locator("button.link-button").first();
  await expect(firstArtifactButton).toBeVisible();
  await firstArtifactButton.click();

  const artifactContent = page.getByTestId("run-artifact-content");
  await expect(artifactContent).toBeVisible();
  await expect
    .poll(async () => (await artifactContent.textContent())?.trim() ?? "")
    .not.toMatch(/^(Select artifact to inspect\.|Loading\.\.\.)$/);

  await expect(page.locator("p.status.err")).toHaveCount(0);
  await test.info().attach("runtime-provider", {
    body: runtimeProvider,
    contentType: "text/plain"
  });
});

test("live ui flow: run refresh -> cancel -> failed(run_canceled)", async ({ page, request }) => {
  test.skip(scenario !== "cancel-refresh", `scenario ${scenario} skips cancel-refresh flow`);
  test.setTimeout(Math.max(cancelTimeoutMs + 120_000, 6 * 60 * 1000));
  const runtimeProvider = process.env.UI_E2E_RUNTIME_PROVIDER ?? "unknown";

  await page.goto("/");
  await expect(page.getByTestId("console-shell")).toBeVisible();
  await expect(page.getByTestId("top-status-bar")).toContainText("Proven Arch");

  await page.getByTestId("stage-readiness").click();
  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();

  await page.getByTestId("stage-analysis").click();
  const previousRunIDLocator = page.getByTestId("run-status-run-id");
  const previousRunID =
    (await previousRunIDLocator.count()) > 0 ? ((await previousRunIDLocator.first().textContent()) ?? "").trim() : "";
  await page.getByTestId("run-refresh-btn").click();
  await expect(page.getByTestId("run-status-panel")).toBeVisible();

  const runID = await test.step("wait for new refresh run id", async () => {
    const deadline = Date.now() + 30_000;
    while (Date.now() < deadline) {
      const currentRunID = ((await page.getByTestId("run-status-run-id").textContent()) ?? "").trim();
      if (currentRunID !== "" && currentRunID !== previousRunID) {
        return currentRunID;
      }
      await sleep(250);
    }
    throw new Error(`did not observe new refresh run id within timeout (previous=${previousRunID || "none"})`);
  });
  expect(runID).not.toBe("");
  console.log(`ACP_UI_E2E_RUN_ID=${runID}`);
  await test.info().attach("run-id", {
    body: runID,
    contentType: "text/plain"
  });
  await expect
    .poll(
      async () => {
        const response = await request.get(`/api/pipeline/runs/${runID}`);
        const payload = (await response.json()) as { status?: string };
        return (payload.status ?? "").trim();
      },
      { timeout: 60_000 }
    )
    .toMatch(/^(queued|running)$/);
  const activeDeadline = Date.now() + Math.max(cancelTimeoutMs, 180_000);
  let activeStatus = "";
  while (Date.now() < activeDeadline) {
    const response = await request.get(`/api/pipeline/runs/${runID}`);
    const payload = (await response.json()) as { status?: string; error_code?: string | null };
    const status = (payload.status ?? "").trim();
    const errorCode = (payload.error_code ?? "").trim();
    if (status === "running") {
      activeStatus = status;
      break;
    }
    if (status === "failed") {
      throw new Error(`run ${runID} terminated before cancel: status=failed error_code=${errorCode || "-"}`);
    }
    await sleep(1000);
  }
  if (activeStatus !== "running") {
    throw new Error(`run ${runID} did not reach running status before cancel deadline (${Math.max(cancelTimeoutSec, 180)}s)`);
  }

  const cancelButton = page.getByTestId("run-cancel-btn");
  await expect(cancelButton).toBeEnabled({ timeout: 30_000 });
  await cancelButton.click();

  await expect
    .poll(
      async () => {
        const response = await request.get(`/api/pipeline/runs/${runID}`);
        const payload = (await response.json()) as { status?: string; error_code?: string | null };
        const status = (payload.status ?? "").trim();
        const errorCode = (payload.error_code ?? "").trim();
        return `${status}|${errorCode}`;
      },
      { timeout: cancelTimeoutMs }
    )
    .toBe("failed|run_canceled");

  await expect(page.getByTestId("run-status-value")).toHaveText("failed");
  await expect(page.getByText(/Error code:\s*run_canceled/i)).toBeVisible();
  await expect(page.getByTestId("run-cancel-btn")).toBeDisabled();

  const selectedRunButton = page.getByRole("button", { name: runID }).first();
  await expect(selectedRunButton).toBeVisible();
  await selectedRunButton.click();

  await page.getByTestId("run-logs-view-select").selectOption("line+fields");
  const logsContent = page.getByTestId("run-logs-content");
  await expect
    .poll(async () => (await logsContent.textContent()) ?? "", { timeout: 30_000 })
    .toContain("\"error_code\": \"run_canceled\"");

  await test.info().attach("runtime-provider", {
    body: runtimeProvider,
    contentType: "text/plain"
  });
});

test("live ui diagnostic: api request context survives page close", async ({ page, request }) => {
  test.skip(
    scenario !== "api-context-page-close-smoke",
    `scenario ${scenario} skips api-context-page-close-smoke flow`
  );
  test.setTimeout(60_000);

  const beforeClose = await request.get("/api/health");
  expect(beforeClose.ok()).toBe(true);

  await page.close();

  const deadline = Date.now() + 10_000;
  let lastStatus = beforeClose.status();
  while (Date.now() < deadline) {
    const response = await request.get("/api/health");
    lastStatus = response.status();
    if (response.ok()) {
      return;
    }
    await sleep(250);
  }

  throw new Error(`api request context did not survive page close: last_health_status=${lastStatus}`);
});
