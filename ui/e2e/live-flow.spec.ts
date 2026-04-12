import { expect, test } from "@playwright/test";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();

test("live ui flow: validate -> run init -> inspect artifacts", async ({ page }) => {
  test.skip(scenario !== "init-inspect", `scenario ${scenario} skips init-inspect flow`);
  const runtimeProvider = process.env.UI_E2E_RUNTIME_PROVIDER ?? "unknown";
  const expectedRepoCountRaw = Number.parseInt(process.env.UI_E2E_EXPECTED_REPO_COUNT ?? "1", 10);
  const expectedRepoCount = Number.isFinite(expectedRepoCountRaw) && expectedRepoCountRaw > 0 ? expectedRepoCountRaw : 1;

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Local-first architecture control plane" })).toBeVisible();

  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();
  const resolvedRepoRows = page.getByTestId("workspace-validate-result").locator(".repo-summary ul li");
  await expect(resolvedRepoRows).toHaveCount(expectedRepoCount);

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
      { timeout: 10 * 60 * 1000 }
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
  const runtimeProvider = process.env.UI_E2E_RUNTIME_PROVIDER ?? "unknown";

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Local-first architecture control plane" })).toBeVisible();

  await page.getByTestId("workspace-validate-btn").click();
  await expect(page.getByTestId("workspace-validate-result")).toBeVisible();
  await expect(page.getByText("Status: valid")).toBeVisible();

  await page.getByTestId("run-refresh-btn").click();
  await expect(page.getByTestId("run-status-panel")).toBeVisible();
  const runID = ((await page.getByTestId("run-status-run-id").textContent()) ?? "").trim();
  expect(runID).not.toBe("");

  const cancelButton = page.getByTestId("run-cancel-btn");
  await expect(cancelButton).toBeEnabled({ timeout: 30_000 });
  await cancelButton.click();
  await expect(page.getByText(new RegExp(`Cancel requested for ${runID}`, "i"))).toBeVisible();

  await expect
    .poll(
      async () => {
        const response = await page.request.get(`/api/pipeline/runs/${runID}`);
        const payload = (await response.json()) as { status?: string; error_code?: string | null };
        const status = (payload.status ?? "").trim();
        const errorCode = (payload.error_code ?? "").trim();
        return `${status}|${errorCode}`;
      },
      { timeout: 3 * 60 * 1000 }
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
