import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { expectNoCriticalAxeViolations } from "./axe";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const runID = "run-permission-recovery";

type FulfillBody = { status: number; contentType: string; body: string };

function json(body: unknown, status = 200): FulfillBody {
  return { status, contentType: "application/json", body: JSON.stringify(body) };
}

function text(body: string, status = 200): FulfillBody {
  return { status, contentType: "text/plain; charset=utf-8", body };
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

async function installPermissionRecoveryMock(page: Page): Promise<void> {
  const pendingPermission = {
    request_id: "perm-install-generated-client",
    run_id: runID,
    step_id: "init.step1.collect",
    provider: "qwen-code",
    action: "shell",
    path_or_command: "npm install --ignore-scripts @acme/generated-client",
    reason: "package install requires explicit operator review before retry",
    decision: {
      request_id: "perm-install-generated-client",
      decision: "needs_user",
      rule_id: "ask_unsafe_operation",
      message: "operation requires explicit user approval",
    },
  };
  const runStatus = {
    run_id: runID,
    pipeline: "init",
    status: "failed",
    current_step: "init.step1.collect",
    started_at: "2026-04-03T12:00:00Z",
    finished_at: "2026-04-03T12:00:04Z",
    warnings: [],
    pending_permissions: [pendingPermission],
    error_code: "runtime_permission_required",
    error: "runtime permission required",
  };

  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const method = request.method().toUpperCase();
    const apiPath = `${url.pathname}${url.search}`;

    if (method === "GET" && url.pathname === "/api/system/version") {
      await route.fulfill({
        ...json({ version: "dev", commit: "mock", built: "mock", ui_bundle: "vite" }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/onboarding/status") {
      await route.fulfill({
        ...json({
          ok: true,
          launcher_mode: false,
          workspace_selected: true,
          workspace_ready: true,
          workspace: "/tmp/permission-recovery-workspace",
          manifest_present: true,
          runtime: {
            selected: true,
            runtime: "headless",
            runtime_provider: "qwen-code",
            provider_source: "override",
          },
          can_enter_console: true,
          recent_workspaces: [],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/pipeline/runs") {
      await route.fulfill({ ...json({ items: [runStatus] }) });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}`) {
      await route.fulfill({ ...json(runStatus) });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/review-summary`) {
      await route.fulfill({
        ...json({
          ...runStatus,
          steps: [
            {
              step_id: "step0_constitution",
              label: "Charter",
              state: "done",
              provider: "qwen-code",
              artifact_count: 1,
              artifact_paths: ["charter/overview.md"],
              taskrun_paths: [],
              warnings_count: 0,
              errors_count: 0,
              last_message: "charter ready",
            },
            {
              step_id: "step1_collect",
              label: "Collect",
              state: "failed",
              provider: "qwen-code",
              artifact_count: 0,
              artifact_paths: [],
              taskrun_paths: [],
              warnings_count: 0,
              errors_count: 1,
              last_message: "runtime permission required",
            },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/artifacts`) {
      await route.fulfill({ ...json({ run_id: runID, artifacts: [] }) });
      return;
    }

    if (method === "GET" && url.pathname === `/api/pipeline/runs/${runID}/logs`) {
      await route.fulfill({
        ...json({
          run_id: runID,
          items: [
            {
              cursor: 1,
              timestamp: "2026-04-03T12:00:01Z",
              level: "info",
              kind: "event",
              step_id: "init.step1.collect",
              domain_id: "generated-client",
              message: "runtime task started",
              fields: { provider: "qwen-code", shard_id: "generated-client-root" },
            },
            {
              cursor: 2,
              timestamp: "2026-04-03T12:00:03Z",
              level: "warning",
              kind: "event",
              step_id: "init.step1.collect",
              domain_id: "generated-client",
              message: "runtime permission required for shell operation",
              fields: {
                provider: "qwen-code",
                request_id: "perm-install-generated-client",
                action: "shell",
                rule_id: "ask_unsafe_operation",
              },
            },
          ],
          next_cursor: 2,
          eof: true,
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/manifest") {
      await route.fulfill({
        ...json({
          content:
            'version: 1\nrepos:\n  - name: "generated-client"\n    path: "/tmp/generated-client"\ndocs:\n  imports_path: "./docs/imports"\nruntime:\n  selected: headless\n  provider: qwen-code\n  profile:\n    permissions:\n      mode: managed\n      approval_channel: fail_fast\n',
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/bundle") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: "/tmp/permission-recovery-workspace",
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
        ...json({
          ok: true,
          workspace: "/tmp/permission-recovery-workspace",
          warnings: [],
          errors: [],
          resolved_repos: [
            {
              name: "generated-client",
              source: "/tmp/generated-client",
              ref: "main",
              status: "ok",
            },
          ],
        }),
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
          effective: {
            strategy: "sequential",
            max_parallel_tasks: 1,
            failure_policy: "best_effort",
            shard_discovery_mode: "heuristics",
          },
          source: { strategy: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/permissions") {
      await route.fulfill({
        ...json({
          ok: true,
          persisted: { mode: "managed", approval_channel: "fail_fast" },
          effective: { mode: "managed", approval_channel: "fail_fast" },
          source: { mode: "workspace", approval_channel: "workspace" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/profile") {
      await route.fulfill({
        ...json({
          ok: true,
          permissions: {
            persisted: { mode: "managed", approval_channel: "fail_fast" },
            effective: { mode: "managed", approval_channel: "fail_fast" },
            source: { mode: "workspace", approval_channel: "workspace" },
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

    if (method === "GET" && url.pathname === "/api/artifacts") {
      const requestedPath = url.searchParams.get("path") ?? "";
      await route.fulfill({
        ...(requestedPath === "charter/overview.md"
          ? text("# Charter\n\nPermission recovery fixture.\n")
          : text(`# Fixture artifact\n\n${requestedPath || "unknown"}\n`)),
      });
      return;
    }

    await route.fulfill({
      ...json({ ok: true, ignored: apiPath }),
    });
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

test("permission recovery mock: Analysis triage and Readiness settings remain readable", async ({ page }) => {
  test.skip(scenario !== "permission-recovery-mock", `scenario ${scenario} skips permission recovery mock`);

  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });
  await installPermissionRecoveryMock(page);

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto("/home");
  await expect(page.getByTestId("product-shell")).toBeVisible();
  await page.getByTestId("destination-runs").click();
  await expect(page.getByTestId("destination-runs")).toHaveAttribute("aria-current", "page");
  await page.getByRole("button", { name: runID }).click();
  await expect(page).toHaveURL(`/runs/${runID}`);

  const recovery = page.getByTestId("analysis-failure-recovery");
  await expect(recovery).toBeVisible();
  await expect(recovery).toContainText("Resolve the pending permission request");
  await expect(recovery).toContainText("runtime_permission_required");

  const permissionRecovery = page.getByTestId("runtime-permission-recovery");
  await expect(permissionRecovery).toBeVisible();
  await expect(permissionRecovery).toContainText("Permission triage");
  await expect(permissionRecovery).toContainText("1 pending request");
  await expect(permissionRecovery).toContainText("init.step1.collect");
  await expect(permissionRecovery).toContainText("shell");
  await expect(permissionRecovery).toContainText("needs_user");
  await expect(permissionRecovery).toContainText("ask_unsafe_operation");
  await expect(permissionRecovery).toContainText("npm install --ignore-scripts @acme/generated-client");
  await expect(permissionRecovery).toContainText("package install requires explicit operator review before retry");
  await expect(permissionRecovery).toContainText("Use Readiness - Advanced runtime settings - Runtime Permissions");

  const permissionTable = page.getByTestId("runs-pending-permissions-table");
  await expect(permissionTable).toBeVisible();
  await expect(permissionTable).toContainText("perm-install-generated-client");
  await expect(permissionTable).toContainText("qwen-code");
  await expect(permissionTable).toContainText("ask_unsafe_operation");
  const permissionCards = page.getByTestId("runs-pending-permissions-cards");

  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "permission-recovery-desktop.png");

  await page.getByRole("link", { name: "Setup" }).click();
  await page.getByTestId("stage-readiness").click();
  const advancedSettings = page.getByTestId("readiness-advanced-settings");
  await advancedSettings.locator("summary").click();
  await expect(page.getByTestId("runtime-permissions-panel")).toBeVisible();
  await expect(page.getByTestId("runtime-permission-mode-select")).toHaveValue("managed");
  await expect(page.getByTestId("runtime-permission-approval-channel-select")).toHaveValue("fail_fast");
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "permission-recovery-readiness-desktop.png");

  await page.setViewportSize({ width: 390, height: 900 });
  await page.getByTestId("destination-runs").click();
  await page.getByRole("button", { name: runID }).click();
  await expect(page).toHaveURL(`/runs/${runID}`);
  await expect(permissionRecovery).toBeVisible();
  await expect(permissionCards).toBeVisible();
  await expect(permissionCards).toContainText("perm-install-generated-client");
  await expect(permissionCards).toContainText("npm install --ignore-scripts @acme/generated-client");
  await expect(permissionCards).toContainText("package install requires explicit operator review before retry");
  await expect(permissionTable).toBeHidden();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "permission-recovery-mobile.png");

  await expectNoCriticalAxeViolations(page);
  expect(consoleErrors).toEqual([]);
});
