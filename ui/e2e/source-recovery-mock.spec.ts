import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const longRepoName = "first-time-payment-service";
const longGitURL = "https://github.com/acme/platform-payment-service-with-a-deliberately-long-repository-name.git";
const longRef = "feature/customer-risk-reconciliation-with-a-very-long-ref-name";

type FulfillBody = { status: number; contentType: string; body: string };

function json(body: unknown, status = 200): FulfillBody {
  return { status, contentType: "application/json", body: JSON.stringify(body) };
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
    contentType: "image/png",
  });
  return screenshotPath;
}

async function installSourceRecoveryMock(page: Page): Promise<void> {
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method().toUpperCase();
    const apiPath = `${url.pathname}${url.search}`;

    if (method === "GET" && url.pathname === "/api/system/version") {
      await route.fulfill({ ...json({ version: "dev", commit: "mock", built: "mock", ui_bundle: "vite" }) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/onboarding/status") {
      await route.fulfill({
        ...json({
          ok: true,
          launcher_mode: false,
          workspace_selected: true,
          workspace_ready: true,
          workspace: "/tmp/source-recovery-workspace",
          manifest_present: true,
          runtime: { selected: true, runtime: "headless", runtime_provider: "qwen-code", provider_source: "override" },
          can_enter_console: true,
          recent_workspaces: [],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/pipeline/runs") {
      await route.fulfill({ ...json({ items: [] }) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/manifest") {
      await route.fulfill({
        ...json({
          content:
            `version: 1\nrepos:\n  - name: "${longRepoName}"\n    git_url: "${longGitURL}"\n    ref: "${longRef}"\ndocs:\n  imports_path: "./docs/imports"\nruntime:\n  selected: headless\n  provider: qwen-code\n`,
        }),
      });
      return;
    }

    if (method === "PUT" && url.pathname === "/api/workspace/manifest") {
      await route.fulfill({ ...json({ ok: true }) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/bundle") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: "/tmp/source-recovery-workspace",
          manifest: {
            schema_version: 1,
            bundle_version: 1,
            editable_artifacts: [{ path: "charter/overview.md", label: "charter/overview.md", category: "charter" }],
          },
          warnings: [],
        }),
      });
      return;
    }

    if (method === "POST" && url.pathname === "/api/workspace/validate") {
      await route.fulfill({
        ...json(
          {
            ok: false,
            workspace: "/tmp/source-recovery-workspace",
            warnings: [],
            errors: [
              {
                level: "error",
                code: "workspace.repo.git_url.fetch_failed",
                message: "git cannot clone this repository with the configured URL and ref.",
                suggestion: "Check the repository URL, ref and local git authentication, then save and validate sources again.",
                repo: longRepoName,
              },
            ],
            resolved_repos: [],
          },
        ),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/system/doctor") {
      await route.fulfill({
        ...json({
          ok: true,
          summary: "ready",
          checks: [
            { id: "git", label: "Git", status: "pass", message: "git found" },
            { id: "runtime_provider", label: "Runtime provider", status: "pass", message: "qwen found" },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/timeouts") {
      await route.fulfill({
        ...json({
          ok: true,
          persisted: { step_timeout_sec: 5400 },
          effective: {
            step_timeout_sec: 5400,
            heartbeat_sec: 30,
            pipeline_timeout_sec: 14400,
            pipeline_kill_grace_sec: 30,
            api_ready_timeout_sec: 60,
            api_init_timeout_sec: 120,
            ui_init_poll_timeout_sec: 1500,
            ui_cancel_poll_timeout_sec: 420,
          },
          source: { step_timeout_sec: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/execution") {
      await route.fulfill({
        ...json({
          ok: true,
          persisted: { strategy: "sequential", max_parallel_tasks: 1 },
          effective: { strategy: "sequential", max_parallel_tasks: 1, failure_policy: "best_effort", shard_discovery_mode: "heuristics" },
          source: { strategy: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/permissions") {
      await route.fulfill({
        ...json({
          ok: true,
          persisted: { mode: "trusted_full_access" },
          effective: { mode: "trusted_full_access", approval_channel: "fail_fast" },
          source: { mode: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/profile") {
      await route.fulfill({
        ...json({
          ok: true,
          permissions: {
            persisted: { mode: "trusted_full_access" },
            effective: { mode: "trusted_full_access", approval_channel: "fail_fast" },
            source: { mode: "workspace" },
          },
          step_providers: {
            persisted: {},
            effective: {
              step0_constitution: "qwen-code",
              step1_collect: "qwen-code",
              step2_as_is: "qwen-code",
              step3_findings: "qwen-code",
              step4_proposals: "qwen-code",
            },
            source: {
              step0_constitution: "default",
              step1_collect: "default",
              step2_as_is: "default",
              step3_findings: "default",
              step4_proposals: "default",
            },
          },
        }),
      });
      return;
    }

    await route.fulfill({ ...json({ ok: true, ignored: apiPath }) });
  });
}

async function expectNoHorizontalOverflow(page: Page): Promise<void> {
  await expect
    .poll(async () =>
      page.evaluate(() => {
        const root = document.documentElement;
        const body = document.body;
        return Math.max(root.scrollWidth - root.clientWidth, body.scrollWidth - body.clientWidth);
      }),
    )
    .toBeLessThanOrEqual(1);
}

test("source recovery mock: validation blockers stay actionable and readable", async ({ page }) => {
  test.skip(scenario !== "source-recovery-mock", `scenario ${scenario} skips source recovery mock`);

  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });
  await installSourceRecoveryMock(page);

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto("/");
  await expect(page.getByTestId("console-shell")).toBeVisible();
  await page.getByTestId("stage-source").click();
  await expect(page.getByTestId("stage-source")).toHaveAttribute("aria-current", "step");

  const recovery = page.getByTestId("source-validation-recovery");
  await expect(recovery).toBeVisible();
  await expect(recovery).toContainText("Source validation recovery");
  await expect(recovery).toContainText(longRepoName);
  await expect(recovery).toContainText("workspace.repo.git_url.fetch_failed");
  await expect(recovery).toContainText("Git URL");
  await expect(recovery).toContainText(longRef);
  await expect(recovery).toContainText(longGitURL);
  await expect(recovery).toContainText("Check the repository URL, ref and local git authentication");
  await expect(recovery).toContainText("Save and validate sources");

  const sourceTable = page.getByTestId("source-repo-table");
  await expect(sourceTable).toContainText(longRepoName);
  await expect(sourceTable).toContainText("blocked");
  await expect(sourceTable).toContainText(longGitURL);
  const sourceRow = sourceTable.getByRole("row").filter({ hasText: longRepoName });
  await expect(sourceRow.getByRole("cell").nth(0)).toContainText(longRepoName);
  await expect(sourceRow.getByRole("cell").nth(1)).toContainText("Git URL");
  await expect(sourceRow.getByRole("cell").nth(1)).toContainText(longGitURL);

  const validationResult = page.getByTestId("workspace-validate-result");
  await expect(validationResult).toContainText("Status: invalid");
  await expect(validationResult).toContainText("Next: Check the repository URL, ref and local git authentication");
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "source-recovery-desktop.png");

  await page.getByTestId("stage-readiness").click();
  await expect(page.getByTestId("stage-readiness")).toHaveAttribute("aria-current", "step");
  await expect(page.getByTestId("setup-run-first-btn")).toBeDisabled();
  await expect(page.getByTestId("readiness-summary-cards")).toContainText("Workspace");
  await expect(page.getByTestId("readiness-summary-cards")).toContainText("blocked");

  await page.getByTestId("stage-source").click();
  await page.setViewportSize({ width: 390, height: 900 });
  await expect(recovery).toBeVisible();
  await expect(sourceTable).toBeVisible();
  await expect(validationResult).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "source-recovery-mobile.png");

  expect(consoleErrors).toEqual([]);
});
