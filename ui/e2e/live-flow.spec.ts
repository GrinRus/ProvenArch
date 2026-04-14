import { expect, test } from "@playwright/test";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const initTimeoutSec = Number.parseInt(process.env.UI_E2E_INIT_TIMEOUT_SEC ?? "900", 10);
const cancelTimeoutSec = Number.parseInt(process.env.UI_E2E_CANCEL_TIMEOUT_SEC ?? "420", 10);
const initTimeoutMs = Number.isFinite(initTimeoutSec) && initTimeoutSec > 0 ? initTimeoutSec * 1000 : 900_000;
const cancelTimeoutMs = Number.isFinite(cancelTimeoutSec) && cancelTimeoutSec > 0 ? cancelTimeoutSec * 1000 : 420_000;

test("live ui flow: validate -> run init -> inspect artifacts", async ({ page }) => {
  test.skip(scenario !== "init-inspect", `scenario ${scenario} skips init-inspect flow`);
  test.setTimeout(Math.max(initTimeoutMs + 120_000, 6 * 60 * 1000));
  const runtimeProvider = process.env.UI_E2E_RUNTIME_PROVIDER ?? "unknown";
  const expectedRepoCountRaw = Number.parseInt(process.env.UI_E2E_EXPECTED_REPO_COUNT ?? "1", 10);
  const expectedRepoCount = Number.isFinite(expectedRepoCountRaw) && expectedRepoCountRaw > 0 ? expectedRepoCountRaw : 1;

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Local-first architecture control plane" })).toBeVisible();

  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();
  const resolvedRepoRows = page.getByTestId("workspace-validate-resolved-repos").locator("li");
  await expect(resolvedRepoRows).toHaveCount(expectedRepoCount);
  const repoSelectionRows = page.getByTestId("workspace-validate-repo-selection").locator("li");
  await expect
    .poll(async () => repoSelectionRows.count())
    .toBeGreaterThan(0);

  await page.getByTestId("run-init-btn").click();
  await expect(page.getByTestId("run-status-panel")).toBeVisible();
  const runID = ((await page.getByTestId("run-status-run-id").textContent()) ?? "").trim();
  expect(runID).not.toBe("");
  await expect
    .poll(
      async () => {
        const response = await page.request.get(`/api/pipeline/runs/${runID}`);
        const payload = (await response.json()) as { status?: string };
        return (payload.status ?? "").trim();
      },
      { timeout: initTimeoutMs }
    )
    .toBe("succeeded");

  const selectedRunButton = page.getByRole("button", { name: runID }).first();
  await expect(selectedRunButton).toBeVisible();
  await selectedRunButton.click();

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

test("live ui flow: run refresh -> cancel -> failed(run_canceled)", async ({ page }) => {
  test.skip(scenario !== "cancel-refresh", `scenario ${scenario} skips cancel-refresh flow`);
  test.setTimeout(Math.max(cancelTimeoutMs + 120_000, 6 * 60 * 1000));
  const runtimeProvider = process.env.UI_E2E_RUNTIME_PROVIDER ?? "unknown";

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Local-first architecture control plane" })).toBeVisible();

  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();

  const previousRunID = ((await page.getByTestId("run-status-run-id").textContent()) ?? "").trim();
  await page.getByTestId("run-refresh-btn").click();
  await expect(page.getByTestId("run-status-panel")).toBeVisible();

  const runID = await test.step("wait for new refresh run id", async () => {
    const deadline = Date.now() + 30_000;
    while (Date.now() < deadline) {
      const currentRunID = ((await page.getByTestId("run-status-run-id").textContent()) ?? "").trim();
      if (currentRunID !== "" && currentRunID !== previousRunID) {
        return currentRunID;
      }
      await page.waitForTimeout(250);
    }
    throw new Error(`did not observe new refresh run id within timeout (previous=${previousRunID || "none"})`);
  });
  expect(runID).not.toBe("");
  await expect
    .poll(
      async () => {
        const response = await page.request.get(`/api/pipeline/runs/${runID}`);
        const payload = (await response.json()) as { status?: string };
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
