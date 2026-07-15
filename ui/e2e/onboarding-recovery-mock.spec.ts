import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { expectNoCriticalAxeViolations } from "./axe";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const workspacePath = "/tmp/onboarding-rendered-workspace";
const repoName = "payments-platform";
const repoURL = "https://github.com/acme/payments-platform-with-a-long-first-time-onboarding-url.git";
const missingRecentWorkspace = "/Users/operator/acp-workspaces/retired-platform-migration-with-a-very-long-workspace-name";

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

function onboardingStatus(patch: Record<string, unknown> = {}) {
  return {
    ok: true,
    launcher_mode: true,
    workspace_selected: false,
    workspace_ready: false,
    workspace: "",
    manifest_present: false,
    runtime: {
      selected: false,
      runtime: "fake",
      runtime_provider: "qwen-code",
      provider_source: "default",
    },
    can_enter_console: false,
    recent_workspaces: [
      {
        path: "/tmp/previous-acp-workspace",
        exists: true,
        last_opened_at: "2026-07-08T15:12:00Z",
      },
      {
        path: missingRecentWorkspace,
        exists: false,
        last_opened_at: "2026-07-07T06:30:00Z",
      },
    ],
    ...patch,
  };
}

async function installOnboardingRecoveryMock(page: Page): Promise<void> {
  let status = onboardingStatus();

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
      await route.fulfill({ ...json(status) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/onboarding/path-suggestions") {
      const kind = url.searchParams.get("kind") ?? "workspace";
      await route.fulfill({
        ...json({
          ok: true,
          suggestions:
            kind === "workspace"
              ? [
                  { path: workspacePath, label: "onboarding-rendered-workspace", exists: true, kind: "workspace" },
                  { path: missingRecentWorkspace, label: "retired-platform-migration", exists: false, kind: "workspace" },
                ]
              : [{ path: "/tmp/payments-platform", label: repoName, exists: true, kind: "repo" }],
        }),
      });
      return;
    }

    if (method === "POST" && url.pathname === "/api/onboarding/workspace") {
      const payload = request.postDataJSON() as { path?: string; create?: boolean };
      status = onboardingStatus({
        workspace_selected: true,
        workspace_ready: true,
        workspace: payload.path || workspacePath,
        manifest_present: true,
        runtime: { selected: false, runtime: "fake", runtime_provider: "qwen-code", provider_source: "default" },
      });
      await route.fulfill({ ...json(status) });
      return;
    }

    if (method === "POST" && url.pathname === "/api/onboarding/runtime") {
      const payload = request.postDataJSON() as { runtime?: string; runtime_provider?: string };
      status = onboardingStatus({
        workspace_selected: true,
        workspace_ready: true,
        workspace: workspacePath,
        manifest_present: true,
        runtime: {
          selected: true,
          runtime: payload.runtime || "headless",
          runtime_provider: payload.runtime_provider || "qwen-code",
          provider_source: "override",
        },
        can_enter_console: false,
      });
      await route.fulfill({ ...json(status) });
      return;
    }

    if (method === "POST" && url.pathname === "/api/onboarding/recent-workspaces/forget") {
      status = {
        ...status,
        recent_workspaces: (status.recent_workspaces as Array<{ path: string }>).filter((workspace) => workspace.path !== request.postDataJSON().path),
      };
      await route.fulfill({ ...json(status) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/manifest") {
      await route.fulfill({
        ...json({
          content:
            `version: 1\nrepos:\n  - name: "${repoName}"\n    git_url: "${repoURL}"\n    ref: "main"\ndocs:\n  imports_path: "./docs/imports"\nruntime:\n  selected: fake\n`,
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
          workspace: workspacePath,
          manifest: {
            schema_version: 1,
            bundle_version: 1,
            editable_artifacts: [],
          },
          warnings: [],
        }),
      });
      return;
    }

    if (method === "POST" && url.pathname === "/api/workspace/validate") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: workspacePath,
          warnings: [],
          errors: [],
          resolved_repos: [{ name: repoName, source: repoURL, ref: "main", status: "ok" }],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/system/doctor") {
      await route.fulfill({
        ...json({
          ok: false,
          summary: "needs attention",
          checks: [
            { id: "git", label: "Git", status: "pass", message: "git found" },
            {
              id: "runtime_provider",
              label: "Runtime provider",
              status: "fail",
              message: "qwen headless_probe_timeout: qwen headless probe timed out after 30s",
              suggestion: "Confirm qwen can answer a short headless prompt, check auth/quota, or set ACP_QWEN_CMD.",
            },
          ],
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

test("onboarding recovery mock: first-time blockers stay readable and retryable", async ({ page }) => {
  test.skip(scenario !== "onboarding-recovery-mock", `scenario ${scenario} skips onboarding recovery mock`);

  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });
  await installOnboardingRecoveryMock(page);

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto("/");
  await expect(page.getByTestId("onboarding-shell")).toBeVisible();
  await expect(page.getByTestId("onboarding-progress-summary")).toContainText("Create or open a workspace");
  await expect(page.getByTestId("onboarding-ready-action-hint")).toContainText("select or create a workspace");
  await expect(page.getByTestId("onboarding-recent-workspaces")).toContainText("available");
  await expect(page.getByTestId("onboarding-recent-workspaces")).toContainText("missing");
  await expect(page.getByTestId("onboarding-recent-workspaces")).toContainText(missingRecentWorkspace);

  await page.getByLabel("Architecture workspace path").fill(workspacePath);
  await page.getByTestId("onboarding-workspace-save").click();
  await expect(page.getByText(`Selected: ${workspacePath}`)).toBeVisible();
  await expect(page.getByTestId("onboarding-progress-summary")).toContainText("Select the runner");

  await page.getByRole("button", { name: "Add repo" }).click();
  const nameInputs = page.getByLabel("Name");
  await nameInputs.nth(1).fill(repoName);
  await expect(page.getByTestId("onboarding-sources-save")).toBeDisabled();
  await expect(page.getByTestId("onboarding-progress-summary")).toContainText("Fix source fields");
  await expect(page.getByTestId("onboarding-progress-summary")).toContainText("repo_name_duplicate");
  await expect(page.getByTestId("onboarding-repo-diagnostics").first()).toContainText("Repo names must be unique inside workspace.yaml");
  await expect(page.getByTestId("onboarding-repo-diagnostics")).toHaveCount(2);
  await expect(page.getByTestId("onboarding-ready-action-hint")).toContainText("fix source diagnostic repo_name_duplicate");

  await page.getByLabel("Runtime").selectOption("headless");
  await page.getByLabel("Provider").selectOption("qwen-code");
  await page.getByTestId("onboarding-runtime-save").click();
  await expect(page.getByTestId("onboarding-runner-recovery")).toContainText("Provider setup for first analysis");
  await page.getByRole("button", { name: "Check readiness" }).click();
  const runnerRecovery = page.getByTestId("onboarding-runner-recovery");
  await expect(runnerRecovery).toContainText("qwen-code");
  await expect(runnerRecovery).toContainText("qwen");
  await expect(runnerRecovery).toContainText("ACP_QWEN_CMD");
  await expect(runnerRecovery).toContainText("Headless probe timeout");
  await expect(runnerRecovery).toContainText("Text readiness probe");
  await expect(runnerRecovery).toContainText("auth/quota latency");
  await expect(runnerRecovery).toContainText("Use fake baseline for a deterministic first walkthrough");
  await expect(page.getByTestId("onboarding-doctor-result")).toContainText("qwen headless_probe_timeout");
  await expect(page.getByTestId("onboarding-run-first-analysis")).toBeDisabled();
  await expect(page.getByTestId("onboarding-enter-console")).toBeDisabled();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "onboarding-recovery-desktop.png");

  await page.setViewportSize({ width: 390, height: 900 });
  await expect(page.getByTestId("onboarding-progress-summary")).toBeVisible();
  await expect(page.getByTestId("onboarding-progress-summary")).toContainText("repo_name_duplicate");
  await expect(page.getByTestId("onboarding-runner-recovery")).toContainText("Headless probe timeout");
  await expect(page.getByTestId("onboarding-ready-action-hint")).toContainText("fix source diagnostic repo_name_duplicate");
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "onboarding-recovery-mobile.png");

  await expectNoCriticalAxeViolations(page);
  expect(consoleErrors).toEqual([]);
});
