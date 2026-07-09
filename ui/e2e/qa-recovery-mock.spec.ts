import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";

const scenario = (process.env.UI_E2E_SCENARIO ?? "init-inspect").trim().toLowerCase();
const screenshotOutputDir = (process.env.UI_E2E_OUTPUT_DIR ?? "").trim();
const failedRunID = "qa-failed-recovery";
const retryRunID = "qa-retry-ok";

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

async function installQARecoveryMock(page: Page): Promise<{ postedQuestions: string[] }> {
  const postedQuestions: string[] = [];
  const failedQARun = {
    run_id: failedRunID,
    pipeline: "qa",
    status: "failed",
    started_at: "2026-04-03T12:00:00Z",
    finished_at: "2026-04-03T12:00:07Z",
    question: "Which service owns checkout and what evidence supports that ownership?",
    current_step: "qa.ask",
    runtime_provider: "qwen-code",
    provider: "qwen-code",
    answer: null,
    citations: null,
    unresolved: null,
    confidence: null,
    generated_at: null,
    warnings: ["answer artifact missing citations", "runtime-execution.json kept for audit"],
    error_code: "runtime_contract_failed",
    error: "qa-answer.json failed validation because citations were missing",
  };
  const retryQARun = {
    run_id: retryRunID,
    pipeline: "qa",
    status: "succeeded",
    started_at: "2026-04-03T12:01:00Z",
    finished_at: "2026-04-03T12:01:05Z",
    question: failedQARun.question,
    current_step: "qa.ask",
    runtime_provider: "qwen-code",
    provider: "qwen-code",
    answer: "checkout-service owns checkout orchestration; ownership evidence is in reports/as-is/overview.md.",
    citations: [{ path: "reports/as-is/overview.md", reason: "ownership evidence" }],
    unresolved: ["confirm escalation owner"],
    confidence: 0.78,
    generated_at: "2026-04-03T12:01:05Z",
    warnings: [],
    error_code: null,
    error: null,
  };

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
          workspace: "/tmp/qa-recovery-workspace",
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
      await route.fulfill({
        ...json({
          items: [
            {
              run_id: "run-analysis-succeeded",
              pipeline: "init",
              status: "succeeded",
              current_step: "init.step4.proposals",
              started_at: "2026-04-03T11:50:00Z",
              finished_at: "2026-04-03T11:58:00Z",
              warnings: [],
            },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/pipeline/runs/run-analysis-succeeded") {
      await route.fulfill({
        ...json({
          run_id: "run-analysis-succeeded",
          pipeline: "init",
          status: "succeeded",
          current_step: "init.step4.proposals",
          started_at: "2026-04-03T11:50:00Z",
          finished_at: "2026-04-03T11:58:00Z",
          warnings: [],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/pipeline/runs/run-analysis-succeeded/review-summary") {
      await route.fulfill({
        ...json({
          run_id: "run-analysis-succeeded",
          pipeline: "init",
          status: "succeeded",
          steps: [
            { step_id: "step0_constitution", label: "Charter", state: "done", provider: "qwen-code", artifact_count: 1 },
            { step_id: "step1_collect", label: "Collect", state: "done", provider: "qwen-code", artifact_count: 2 },
            { step_id: "step2_as_is", label: "As-is", state: "done", provider: "qwen-code", artifact_count: 2 },
            { step_id: "step3_findings", label: "Findings", state: "done", provider: "qwen-code", artifact_count: 1 },
            { step_id: "step4_proposals", label: "Proposals", state: "done", provider: "qwen-code", artifact_count: 1 },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/pipeline/runs/run-analysis-succeeded/artifacts") {
      await route.fulfill({
        ...json({
          run_id: "run-analysis-succeeded",
          artifacts: [
            { path: "reports/as-is/overview.md", kind: "report", label: "As-is overview" },
            { path: "reports/coverage/open-questions.md", kind: "report", label: "Open questions" },
          ],
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/pipeline/runs/run-analysis-succeeded/logs") {
      await route.fulfill({ ...json({ run_id: "run-analysis-succeeded", items: [], next_cursor: 0, eof: true }) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/qa/runs") {
      await route.fulfill({ ...json({ items: [failedQARun] }) });
      return;
    }

    if (method === "GET" && url.pathname === `/api/qa/runs/${failedRunID}`) {
      await route.fulfill({ ...json(failedQARun) });
      return;
    }

    if (method === "POST" && url.pathname === "/api/qa/runs") {
      const payload = JSON.parse(request.postData() || "{}") as { question?: string };
      postedQuestions.push(payload.question ?? "");
      await route.fulfill({ ...json({ run_id: retryRunID, status: "queued" }) });
      return;
    }

    if (method === "GET" && url.pathname === `/api/qa/runs/${retryRunID}`) {
      await route.fulfill({ ...json(retryQARun) });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/manifest") {
      await route.fulfill({
        ...json({
          content:
            'version: 1\nrepos:\n  - name: "checkout"\n    path: "/tmp/checkout"\ndocs:\n  imports_path: "./docs/imports"\nruntime:\n  selected: headless\n  provider: qwen-code\n',
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/workspace/bundle") {
      await route.fulfill({
        ...json({
          ok: true,
          workspace: "/tmp/qa-recovery-workspace",
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
          workspace: "/tmp/qa-recovery-workspace",
          warnings: [],
          errors: [],
          resolved_repos: [{ name: "checkout", source: "/tmp/checkout", ref: "main", status: "ok" }],
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
          persisted: { mode: "trusted_full_access", approval_channel: "fail_fast" },
          effective: { mode: "trusted_full_access", approval_channel: "fail_fast" },
          source: { mode: "workspace", approval_channel: "default" },
        }),
      });
      return;
    }

    if (method === "GET" && url.pathname === "/api/runtime/profile") {
      await route.fulfill({
        ...json({
          ok: true,
          permissions: {
            persisted: { mode: "trusted_full_access", approval_channel: "fail_fast" },
            effective: { mode: "trusted_full_access", approval_channel: "fail_fast" },
            source: { mode: "workspace", approval_channel: "default" },
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
      const bodies: Record<string, string> = {
        "charter/overview.md": "# Charter\n\nQA recovery fixture.\n",
        "reports/as-is/overview.md": "# As-is overview\n\ncheckout-service owns checkout orchestration.\n",
        "reports/coverage/open-questions.md": "- Confirm checkout escalation owner.\n",
        [`reports/taskruns/${failedRunID}/qa/context-pack.json`]: '{"run_id":"qa-failed-recovery","question":"checkout ownership"}',
        [`reports/taskruns/${failedRunID}/qa/qa-answer.json`]: '{"status":"failed","error":"missing citations"}',
        [`reports/taskruns/${failedRunID}/qa/runtime-execution.json`]: '{"provider":"qwen-code","status":"failed"}',
      };
      await route.fulfill({ ...(requestedPath in bodies ? text(bodies[requestedPath]) : text(`# Fixture artifact\n\n${requestedPath || "unknown"}\n`)) });
      return;
    }

    await route.fulfill({ ...json({ ok: true, ignored: apiPath }) });
  });

  return { postedQuestions };
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

test("qa recovery mock: failed Ask run remains understandable and retryable", async ({ page }) => {
  test.skip(scenario !== "qa-recovery-mock", `scenario ${scenario} skips QA recovery mock`);

  const consoleErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text());
    }
  });
  const { postedQuestions } = await installQARecoveryMock(page);

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto("/");
  await expect(page.getByTestId("console-shell")).toBeVisible();
  await page.getByTestId("stage-ask").click();
  await expect(page.getByTestId("stage-ask")).toHaveAttribute("aria-current", "step");

  const recovery = page.getByTestId("qa-failure-recovery");
  await expect(recovery).toBeVisible();
  await expect(recovery).toContainText("Recovery path");
  await expect(recovery).toContainText("runtime_contract_failed");
  await expect(recovery).toContainText("qa.ask");
  await expect(recovery).toContainText(`reports/taskruns/${failedRunID}/qa/`);
  await expect(recovery).toContainText("qa-answer.json failed validation because citations were missing");
  await expect(recovery).toContainText("answer artifact missing citations");
  await expect(recovery).toContainText("Retry starts a new Q&A run");
  await expect(page.getByTestId("qa-run-history")).toContainText("Which service owns checkout");
  await expect(page.getByTestId("qa-run-status")).toContainText("status: failed");
  await expect(page.getByTestId("qa-answer")).toContainText("No answer returned yet");
  await expect(page.getByTestId("qa-readonly-safety-panel")).toContainText("no canonical writes");
  await expect(page.getByTestId("qa-readonly-safety-panel")).toContainText("reports/taskruns/<run_id>/qa/");
  await expect(page.getByTestId("qa-readonly-safety-panel")).toContainText("context-pack.json");
  await expect(page.getByTestId("qa-readonly-safety-panel")).toContainText("qa-answer.json");
  await expect(page.getByTestId("qa-readonly-safety-panel")).toContainText("runtime-execution.json");
  await expect(page.getByTestId("qa-citations-panel")).toContainText("No citations returned");
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "qa-recovery-desktop.png");

  await page.setViewportSize({ width: 390, height: 900 });
  await expect(recovery).toBeVisible();
  await expect(page.getByTestId("qa-run-history")).toBeVisible();
  await expect(page.getByTestId("qa-readonly-safety-panel")).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "qa-recovery-mobile.png");

  await page.getByTestId("qa-retry-run-btn").click();
  await expect(page.getByTestId("qa-answer")).toContainText("checkout-service owns checkout orchestration");
  expect(postedQuestions).toEqual(["Which service owns checkout and what evidence supports that ownership?"]);
  expect(consoleErrors).toEqual([]);
});
