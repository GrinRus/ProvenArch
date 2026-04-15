import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect-service-first").trim().toLowerCase();
const initTimeoutSec = Number.parseInt(process.env.UI_E2E_INIT_TIMEOUT_SEC ?? "900", 10);
const cancelTimeoutSec = Number.parseInt(process.env.UI_E2E_CANCEL_TIMEOUT_SEC ?? "420", 10);
const expectedRepoCountRaw = Number.parseInt(process.env.UI_E2E_EXPECTED_REPO_COUNT ?? "1", 10);
const expectedRepoCount = Number.isFinite(expectedRepoCountRaw) && expectedRepoCountRaw > 0 ? expectedRepoCountRaw : 1;
const requireChunkedService = (process.env.UI_E2E_REQUIRE_CHUNKED_SERVICE ?? "0").trim() === "1";
const expectedExecutionStrategyEnvRaw = (process.env.UI_E2E_EXPECTED_STRATEGY ?? "").trim();
const expectedExecutionStrategyLegacyRaw = (process.env.ACP_EXECUTION_STRATEGY ?? "").trim();
const expectedExecutionStrategyRaw = expectedExecutionStrategyEnvRaw || expectedExecutionStrategyLegacyRaw;
const expectedExecutionParallelEnvRaw = Number.parseInt(process.env.UI_E2E_EXPECTED_MAX_PARALLEL_TASKS ?? "", 10);
const expectedExecutionParallelLegacyRaw = Number.parseInt(process.env.ACP_MAX_PARALLEL_TASKS ?? "", 10);
const expectedExecutionParallelRaw = Number.isFinite(expectedExecutionParallelEnvRaw)
  ? expectedExecutionParallelEnvRaw
  : expectedExecutionParallelLegacyRaw;

const initTimeoutMs = Number.isFinite(initTimeoutSec) && initTimeoutSec > 0 ? initTimeoutSec * 1000 : 900_000;
const cancelTimeoutMs = Number.isFinite(cancelTimeoutSec) && cancelTimeoutSec > 0 ? cancelTimeoutSec * 1000 : 420_000;

type RunStatusPayload = {
  status?: string;
  current_step?: string;
  error_code?: string | null;
  error?: string | null;
};

type RunArtifactsPayload = {
  artifacts?: Array<{ path?: string; kind?: string; label?: string }>;
};

type ServiceInventoryShard = {
  shard_id: string;
  file_count: number;
  source_bytes: number;
};

type ServiceInventoryService = {
  service_id: string;
  file_count: number;
  source_bytes: number;
  shards: ServiceInventoryShard[];
};

type ServiceInventoryPlan = {
  mode: string;
  services: ServiceInventoryService[];
  selected_shards: Array<{ shard_id: string }>;
};

async function validateWorkspace(page: Page, request: APIRequestContext): Promise<void> {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Local-first architecture control plane" })).toBeVisible();
  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();

  const validateResponse = await request.post("/api/workspace/validate");
  expect(validateResponse.ok()).toBeTruthy();
  const payload = (await validateResponse.json()) as {
    selected_repo_scopes?: string[];
    resolved_repos?: Array<{ name?: string }>;
    repo_selection?: Array<{ name?: string }>;
  };
  const resolvedRepos = Array.isArray(payload.resolved_repos) ? payload.resolved_repos : [];
  if (resolvedRepos.length > 0) {
    expect(resolvedRepos.length).toBe(expectedRepoCount);
    return;
  }
  const selectedScopes = Array.isArray(payload.selected_repo_scopes) ? payload.selected_repo_scopes : [];
  if (selectedScopes.length > 0) {
    expect(selectedScopes.length).toBe(expectedRepoCount);
    return;
  }
  const selectionRows = Array.isArray(payload.repo_selection) ? payload.repo_selection : [];
  expect(selectionRows.length).toBeGreaterThanOrEqual(expectedRepoCount);
}

async function assertExecutionDefaults(request: APIRequestContext): Promise<void> {
  const response = await request.get("/api/runtime/execution");
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as {
    effective?: { strategy?: string; max_parallel_tasks?: number };
  };

  const effectiveStrategy = payload.effective?.strategy === "parallel" ? "parallel" : "sequential";
  const effectiveMaxParallelRaw = Number(payload.effective?.max_parallel_tasks ?? 1);
  const effectiveMaxParallel =
    Number.isFinite(effectiveMaxParallelRaw) && effectiveMaxParallelRaw > 0 ? effectiveMaxParallelRaw : 1;

  const expectedStrategy =
    expectedExecutionStrategyRaw === "parallel" || expectedExecutionStrategyRaw === "sequential"
      ? expectedExecutionStrategyRaw
      : effectiveStrategy;
  let expectedMaxParallel =
    Number.isFinite(expectedExecutionParallelRaw) && expectedExecutionParallelRaw > 0
      ? expectedExecutionParallelRaw
      : effectiveMaxParallel;
  if (expectedStrategy !== "parallel") {
    expectedMaxParallel = 1;
  }

  expect(payload.effective?.strategy).toBe(expectedStrategy);
  expect(payload.effective?.max_parallel_tasks).toBe(expectedMaxParallel);
}

async function waitForRunIDChange(page: Page, previousRunID: string): Promise<string> {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    const currentRunID = ((await page.getByTestId("run-status-run-id").textContent()) ?? "").trim();
    if (currentRunID !== "" && currentRunID !== previousRunID) {
      return currentRunID;
    }
    await page.waitForTimeout(250);
  }
  throw new Error(`did not observe new run id within timeout (previous=${previousRunID || "none"})`);
}

async function currentRunID(page: Page): Promise<string> {
  const runIDLocator = page.getByTestId("run-status-run-id");
  const hasRunStatus = (await runIDLocator.count()) > 0;
  if (!hasRunStatus) {
    return "";
  }
  return ((await runIDLocator.first().textContent()) ?? "").trim();
}

async function waitForRunTerminalStatus(
  request: APIRequestContext,
  runID: string,
  timeoutMs: number
): Promise<RunStatusPayload> {
  let lastPayload: RunStatusPayload = {};
  await expect
    .poll(
      async () => {
        const response = await request.get(`/api/pipeline/runs/${runID}`);
        lastPayload = (await response.json()) as RunStatusPayload;
        return (lastPayload.status ?? "").trim();
      },
      { timeout: timeoutMs }
    )
    .toMatch(/^(succeeded|failed)$/);
  return lastPayload;
}

async function fetchRunArtifacts(request: APIRequestContext, runID: string): Promise<RunArtifactsPayload> {
  const response = await request.get(`/api/pipeline/runs/${runID}/artifacts`);
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as RunArtifactsPayload;
}

async function readArtifactJSON<T>(request: APIRequestContext, path: string): Promise<T> {
  const response = await request.get(`/api/artifacts?path=${encodeURIComponent(path)}`);
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as T;
}

function findArtifactPath(artifacts: RunArtifactsPayload, matcher: RegExp): string {
  for (const artifact of artifacts.artifacts ?? []) {
    const path = (artifact.path ?? "").trim();
    if (matcher.test(path)) {
      return path;
    }
  }
  throw new Error(`artifact path not found by matcher ${matcher}`);
}

function assertChunkingPolicy(plan: ServiceInventoryPlan): void {
  expect(plan.services.length).toBeGreaterThan(0);
  let hasChunkedLargeService = false;
  let totalShards = 0;
  for (const service of plan.services) {
    expect(service.shards.length).toBeGreaterThan(0);
    expect(service.shards.length).toBeLessThanOrEqual(8);
    const isLarge = service.file_count > 500 || service.source_bytes > 8 * 1024 * 1024;
    if (isLarge && service.shards.length > 1) {
      hasChunkedLargeService = true;
    }
    for (let idx = 0; idx < service.shards.length; idx += 1) {
      const shard = service.shards[idx];
      if (idx < service.shards.length - 1) {
        expect(shard.file_count).toBeLessThanOrEqual(200);
        expect(shard.source_bytes).toBeLessThanOrEqual(3 * 1024 * 1024);
      }
      totalShards += 1;
    }
  }
  expect(plan.selected_shards.length).toBeLessThanOrEqual(totalShards);
  if (requireChunkedService) {
    expect(hasChunkedLargeService).toBeTruthy();
  }
}

async function assertGlobalReviewExecutedOnce(
  request: APIRequestContext,
  runID: string,
  stepPrefix: "init" | "refresh"
): Promise<void> {
  const response = await request.get(`/api/pipeline/runs/${runID}/logs?cursor=0&limit=1000`);
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as {
    items?: Array<{ step_id?: string; message?: string }>;
  };
  const matches = (payload.items ?? []).filter(
    (item) => item.step_id === `${stepPrefix}.step5.global_review` && item.message === "runtime task started"
  );
  expect(matches).toHaveLength(1);
}

async function assertServiceFirstOutputs(
  request: APIRequestContext,
  runID: string,
  stepPrefix: "init" | "refresh",
  expectedMode: "incremental" | "full" | "init"
): Promise<void> {
  const artifacts = await fetchRunArtifacts(request, runID);

  const planPath = findArtifactPath(artifacts, new RegExp(`${runID}-service-inventory-plan\\.json$`));
  const plan = await readArtifactJSON<ServiceInventoryPlan>(request, planPath);
  assertChunkingPolicy(plan);

  if (expectedMode === "incremental") {
    expect(plan.mode).toBe("incremental");
  }
  if (expectedMode === "full") {
    expect(plan.mode).toBe("full");
    const totalShards = plan.services.reduce((acc, service) => acc + service.shards.length, 0);
    expect(plan.selected_shards.length).toBe(totalShards);
  }

  const globalInputPath = findArtifactPath(artifacts, new RegExp(`${runID}-global-review-input\\.json$`));
  expect(globalInputPath.length).toBeGreaterThan(0);
  await assertGlobalReviewExecutedOnce(request, runID, stepPrefix);

  const architectSummaryPath = findArtifactPath(artifacts, /reports\/agent-outputs\/architect\/summary\.md$/);
  const summaryResponse = await request.get(`/api/artifacts?path=${encodeURIComponent(architectSummaryPath)}`);
  expect(summaryResponse.ok()).toBeTruthy();
  const summaryText = await summaryResponse.text();
  expect(summaryText).toContain("Architect Aggregation Summary");
}

async function clickRunInitAndWait(page: Page): Promise<string> {
  const previousRunID = await currentRunID(page);
  await page.getByTestId("run-init-btn").click();
  await expect(page.getByTestId("run-status-panel")).toBeVisible();
  return waitForRunIDChange(page, previousRunID);
}

async function clickRunRefreshAndWait(page: Page, mode: "incremental" | "full"): Promise<string> {
  const previousRunID = await currentRunID(page);
  await page.getByTestId("run-refresh-mode-select").selectOption(mode);
  await page.getByTestId("run-refresh-btn").click();
  await expect(page.getByTestId("run-status-panel")).toBeVisible();
  return waitForRunIDChange(page, previousRunID);
}

test("live ui flow: init-inspect-service-first", async ({ page }) => {
  test.skip(scenario !== "init-inspect-service-first" && scenario !== "init-inspect", `scenario ${scenario} skips init-inspect-service-first flow`);
  test.setTimeout(Math.max(initTimeoutMs + 120_000, 6 * 60 * 1000));

  await validateWorkspace(page, page.request);
  await assertExecutionDefaults(page.request);

  const runID = await clickRunInitAndWait(page);
  const status = await waitForRunTerminalStatus(page.request, runID, initTimeoutMs);
  expect(status.status).toBe("succeeded");
  expect(status.current_step).toBe("init.step6.proposals");

  await assertServiceFirstOutputs(page.request, runID, "init", "init");

  const selectedRunButton = page.getByRole("button", { name: runID }).first();
  await expect(selectedRunButton).toBeVisible();
  await selectedRunButton.click();
  const firstArtifactButton = page.getByTestId("results-artifacts-panel").locator("button.link-button").first();
  await expect(firstArtifactButton).toBeVisible();
  await firstArtifactButton.click();
  const artifactContent = page.getByTestId("run-artifact-content");
  await expect(artifactContent).toBeVisible();
  await expect.poll(async () => (await artifactContent.textContent())?.trim() ?? "").not.toMatch(/^(Select artifact to inspect\.|Loading\.\.\.)$/);
  await expect(page.locator("p.status.err")).toHaveCount(0);
});

test("live ui flow: refresh-incremental", async ({ page }) => {
  test.skip(scenario !== "refresh-incremental", `scenario ${scenario} skips refresh-incremental flow`);
  test.setTimeout(Math.max(initTimeoutMs + 120_000, 6 * 60 * 1000));

  await validateWorkspace(page, page.request);
  await assertExecutionDefaults(page.request);

  const initRunID = await clickRunInitAndWait(page);
  const initStatus = await waitForRunTerminalStatus(page.request, initRunID, initTimeoutMs);
  expect(initStatus.status).toBe("succeeded");

  const refreshRunID = await clickRunRefreshAndWait(page, "incremental");
  const refreshStatus = await waitForRunTerminalStatus(page.request, refreshRunID, initTimeoutMs);
  expect(refreshStatus.status).toBe("succeeded");
  expect(refreshStatus.current_step).toBe("refresh.step6.proposals");

  await assertServiceFirstOutputs(page.request, refreshRunID, "refresh", "incremental");
});

test("live ui flow: refresh-full", async ({ page }) => {
  test.skip(scenario !== "refresh-full", `scenario ${scenario} skips refresh-full flow`);
  test.setTimeout(Math.max(initTimeoutMs + 120_000, 6 * 60 * 1000));

  await validateWorkspace(page, page.request);
  await assertExecutionDefaults(page.request);

  const initRunID = await clickRunInitAndWait(page);
  const initStatus = await waitForRunTerminalStatus(page.request, initRunID, initTimeoutMs);
  expect(initStatus.status).toBe("succeeded");

  const refreshRunID = await clickRunRefreshAndWait(page, "full");
  const refreshStatus = await waitForRunTerminalStatus(page.request, refreshRunID, initTimeoutMs);
  expect(refreshStatus.status).toBe("succeeded");
  expect(refreshStatus.current_step).toBe("refresh.step6.proposals");

  await assertServiceFirstOutputs(page.request, refreshRunID, "refresh", "full");
});

test("live ui flow: cancel-refresh", async ({ page }) => {
  test.skip(scenario !== "cancel-refresh", `scenario ${scenario} skips cancel-refresh flow`);
  test.setTimeout(Math.max(cancelTimeoutMs + 120_000, 6 * 60 * 1000));

  await validateWorkspace(page, page.request);
  await assertExecutionDefaults(page.request);

  const runID = await clickRunRefreshAndWait(page, "incremental");
  await expect
    .poll(
      async () => {
        const response = await page.request.get(`/api/pipeline/runs/${runID}`);
        const payload = (await response.json()) as RunStatusPayload;
        return (payload.status ?? "").trim();
      },
      { timeout: 60_000 }
    )
    .toBe("running");

  const cancelButton = page.getByTestId("run-cancel-btn");
  await expect(cancelButton).toBeEnabled({ timeout: 30_000 });
  await cancelButton.click();

  await expect
    .poll(
      async () => {
        const response = await page.request.get(`/api/pipeline/runs/${runID}`);
        const payload = (await response.json()) as RunStatusPayload;
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
});
