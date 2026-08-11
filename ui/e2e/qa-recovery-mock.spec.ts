import { expect, test, type Page } from "@playwright/test";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { expectNoCriticalAxeViolations } from "./axe";

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

    if (method === "GET" && url.pathname === "/api/knowledge") {
      await route.fulfill({
        ...json({
          version: 1,
          generated_at: "2026-08-03T00:00:00Z",
          source_mode: "promoted_current",
          status: "unavailable",
          entities: [],
          edges: [],
          artifacts: [],
          issues: [],
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

    if (method === "POST" && url.pathname === "/api/pipeline/runs/run-analysis-succeeded/retry-plan") {
      const payload = JSON.parse(request.postData() || "{}") as { step_id?: string };
      const requestedStep = payload.step_id || "init.step4.proposals";
      const executeSteps = requestedStep === "init.step3.findings"
        ? ["init.step3.findings", "init.step4.proposals"]
        : [requestedStep];
      await route.fulfill({ ...json({
        parent_run_id: "run-analysis-succeeded",
        pipeline: "init",
        requested_step: requestedStep,
        effective_start_step: requestedStep,
        requested_scopes: [],
        effective_scopes: [],
        reused_inputs: ["init.step0.constitution", "init.step1.collect", "init.step2.asis_docs"],
        execute_steps: executeSteps,
        invalidated_steps: executeSteps,
        estimated_units: executeSteps.length,
        widened: false,
        plan_hash: "qa-recovery-plan",
      }) });
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
  await page.goto("/home");
  await expect(page.getByTestId("product-shell")).toBeVisible();
  await expect(page.getByTestId("home-panel")).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "home-desktop.png");

  await page.getByTestId("destination-knowledge").click();
  await expect(page.getByTestId("knowledge-panel")).toBeVisible();
  await expect(page.getByText("No promoted knowledge is available.")).toBeVisible();
  await captureEvidenceScreenshot(page, "knowledge-empty-desktop.png");

  await page.setViewportSize({ width: 1024, height: 768 });
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "knowledge-empty-tablet.png");

  await page.setViewportSize({ width: 390, height: 844 });
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "knowledge-empty-mobile.png");

  await page.getByTestId("destination-home").click();
  await expect(page.getByTestId("home-panel")).toBeVisible();
  await captureEvidenceScreenshot(page, "home-mobile.png");

  await page.setViewportSize({ width: 1440, height: 980 });
  await page.getByTestId("stage-ask").click();
  await expect(page.getByTestId("qa-panel")).toBeVisible();

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

  for (const width of [1280, 1024]) {
    await page.setViewportSize({ width, height: 900 });
    await expect(page.getByRole("dialog", { name: "Ask current workspace" })).toBeVisible();
    await expect(recovery).toBeVisible();
    await expectNoHorizontalOverflow(page);
  }

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(recovery).toBeVisible();
  await expect(page.getByTestId("qa-run-history")).toBeVisible();
  await expect(page.getByTestId("qa-readonly-safety-panel")).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "qa-recovery-mobile.png");

  await page.getByTestId("qa-retry-run-btn").click();
  await expect(page.getByTestId("qa-answer")).toContainText("checkout-service owns checkout orchestration");
  expect(postedQuestions).toEqual(["Which service owns checkout and what evidence supports that ownership?"]);

  const evidence = [{ repo: "commerce", path: "services/checkout/main.go", lines: { start: 18, end: 42 } }];
  const checkout = { id: "service.checkout", name: "Checkout", type: "service", owner_team_id: "team-commerce", tags: ["domain:commerce"], confidence: 0.94, provenance_kind: "observed", evidence, path: "model/entities/service.checkout.yaml", available_levels: ["context", "container"], child_levels: ["container", "component", "code"], repositories: ["commerce"], related_findings: ["finding.checkout-timeout"], related_questions: [] };
  const external = { id: "external.payment", name: "Payment provider", type: "external.system", tags: ["domain:payments-external"], confidence: 0.81, provenance_kind: "observed", evidence, path: "model/entities/external.payment.yaml", available_levels: ["context", "container"], repositories: ["commerce"], related_findings: [], related_questions: [] };
  const component = { id: "api.checkout", name: "Checkout API", type: "api.http", owner_team_id: "team-commerce", tags: ["domain:commerce"], confidence: 0.91, provenance_kind: "observed", evidence, path: "model/entities/api.checkout.yaml", available_levels: ["component", "code"], repositories: ["commerce"], related_findings: [], related_questions: ["question.checkout-owner"] };
  const relationship = { id: "edge.checkout-payment", from: checkout.id, to: external.id, type: "calls", name: "Authorizes payment", confidence: 0.89, provenance_kind: "observed", evidence, path: "model/edges/edge.checkout-payment.yaml", repositories: ["commerce"], related_findings: [], related_questions: [] };
  const architecturePayload = { version: 1, generated_at: "2026-08-03T12:00:00Z", authority: { mode: "promoted_current", source_run_id: "run-analysis-succeeded", freshness: "current" }, status: "available", counts: { entities: 3, edges: 1, evidence: 4, issues: 0 }, views: { context: { level: "context", available: true, nodes: [checkout, external], edges: [relationship] }, container: { level: "container", available: true, nodes: [checkout, external], edges: [relationship] }, component: { level: "component", available: true, nodes: [component], edges: [] }, code: { level: "code", available: true, nodes: [component], edges: [] } }, exports: { home_path: "reports/as-is/overview.md", c4_mermaid_paths: ["reports/diagrams/c4-context.mmd"] }, comparison: { available: false, categories: { entities: { added: [], changed: [], removed: [] }, edges: { added: [], changed: [], removed: [] }, findings: { added: [], changed: [], removed: [] }, gaps: { added: [], changed: [], removed: [] } } }, review: { findings: [{ id: "finding.checkout-timeout", severity: "medium", title: "Checkout timeout is not bounded", related_ids: [checkout.id] }], questions: [{ id: "question.checkout-owner", text: "Who owns the public API?", related_ids: [component.id] }] }, coverage: { observed: ["checkout"], missing: ["timeout policy"] }, artifacts: [], issues: [] };
  await page.route("**/api/architecture", async (route) => route.fulfill({ ...json(architecturePayload) }));
	await page.route("**/api/knowledge", async (route) => route.fulfill({ ...json({ error: { code: "legacy_unavailable", message: "legacy Knowledge projection is unavailable" } }, 500) }));
  await page.setViewportSize({ width: 1440, height: 980 });
  await page.goto("/architecture/map");
  await expect(page.getByTestId("architecture-canvas")).toBeVisible();
  await page.getByText("Checkout", { exact: true }).first().click();
  await expect(page.getByTestId("knowledge-entity-detail")).toContainText("team-commerce");
  await expect(page.getByTestId("knowledge-entity-detail")).toContainText("finding.checkout-timeout");
  await page.getByText("Advanced", { exact: true }).click();
  await page.getByRole("button", { name: "Code for selected service" }).click();
	await expect(page.locator(".segmented-control details")).not.toHaveAttribute("open", "");
  await expect(page.getByText("Checkout API", { exact: true }).first()).toBeVisible();
  await expect(page.locator(".architecture-breadcrumb")).toContainText("Checkout / Code");
  await page.getByRole("button", { name: "System context", exact: true }).click();
  await expect(page.locator(".react-flow__edge")).toHaveCount(1);
  await page.getByRole("combobox", { name: "Filter by owner" }).selectOption("team-commerce");
  await expect(page.locator(".react-flow__edge")).toHaveCount(0);
  await page.getByRole("combobox", { name: "Filter by owner" }).selectOption("");
  await expect(page.locator(".react-flow__edge")).toHaveCount(1);
  await page.getByRole("combobox", { name: "Filter by domain or tag" }).selectOption("domain:commerce");
  await expect(page.locator(".react-flow__edge")).toHaveCount(0);
  await page.getByRole("combobox", { name: "Filter by domain or tag" }).selectOption("");
  await expect(page.locator(".react-flow__edge")).toHaveCount(1);
  const keyboardNode = page.locator(".react-flow__node .architecture-node-button").first();
  await expect(keyboardNode).toBeVisible();
  await keyboardNode.press("Enter");
  await expect(page.getByTestId("knowledge-entity-detail")).toBeVisible();
  await page.getByRole("button", { name: "Component", exact: true }).click();
  await expect(page.getByText("Checkout API", { exact: true }).first()).toBeVisible();
  await page.getByRole("combobox", { name: "Filter by type" }).selectOption("api.http");
  await captureEvidenceScreenshot(page, "architecture-map-desktop.png");
  await page.setViewportSize({ width: 1024, height: 768 });
  await page.evaluate(() => { document.documentElement.style.zoom = "2"; });
  await expect(page.getByRole("heading", { name: "Architecture", exact: true })).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await page.evaluate(() => { document.documentElement.style.zoom = ""; });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator(".architecture-mobile-list")).toBeVisible();
  await expect(page.getByText("Checkout API", { exact: true }).last()).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await captureEvidenceScreenshot(page, "architecture-map-mobile.png");
  await page.setViewportSize({ width: 1440, height: 980 });
  await page.getByTestId("destination-home").click();
  await expect(page.locator(".home-map-visual")).toContainText("Checkout");
  await captureEvidenceScreenshot(page, "home-architecture-desktop.png");
	await page.goto("/runs/run-analysis-succeeded");
	const targetedRerun = page.getByTestId("targeted-rerun-panel");
	await expect(targetedRerun).toBeVisible();
	await expect(targetedRerun).toContainText("Repeat only the work you need");
	await targetedRerun.getByLabel("Start from step").selectOption("init.step3.findings");
	await expect(targetedRerun.getByLabel("Start from step")).toHaveValue("init.step3.findings");
	await targetedRerun.getByRole("button", { name: "Review rerun plan" }).click();
	await expect(targetedRerun.getByTestId("retry-plan")).toContainText("Invalidated dependency closure");
	await expect(targetedRerun.getByTestId("retry-plan")).toContainText("Every dependent downstream result must be rebuilt");
	await expect(targetedRerun.getByTestId("retry-plan")).toContainText("2 estimated execution unit(s)");
	await expectNoHorizontalOverflow(page);
	await captureEvidenceScreenshot(page, "successful-targeted-rerun-desktop.png");
  await expectNoCriticalAxeViolations(page);
	expect(consoleErrors.filter((message) => !message.includes("Failed to load resource: the server responded with a status of 500"))).toEqual([]);
});
