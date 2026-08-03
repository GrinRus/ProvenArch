import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { OnboardingShell } from "./OnboardingShell";
import type { DoctorResponse, GuidedRepo, OnboardingStatusResponse, ValidateResponse } from "../lib/appContracts";

const noop = vi.fn();

const baseStatus: OnboardingStatusResponse = {
  ok: true,
  launcher_mode: true,
  workspace_selected: true,
  workspace_ready: true,
  workspace: "/work/acp",
  manifest_present: true,
  runtime: {
    selected: true,
    runtime: "fake",
    runtime_provider: "fake",
  },
  can_enter_console: false,
  recent_workspaces: [],
};

const incompleteRepo: GuidedRepo = {
  id: "repo-1",
  name: "",
  mode: "git_url",
  git_url: "",
  path: "",
  ref: "",
  analysis_include: "",
  analysis_exclude: "",
};

function renderShell(overrides: Partial<Parameters<typeof OnboardingShell>[0]> = {}) {
  const props: Parameters<typeof OnboardingShell>[0] = {
    busy: false,
    error: null,
    status: baseStatus,
    workspacePath: "/work/acp",
    createWorkspace: false,
    guidedRepos: [incompleteRepo],
    guidedDocsImportsPath: "docs/imports",
    validateResult: null,
    doctorResult: null,
    setupRuntime: "fake",
    setupRuntimeProvider: "claude-code",
    firstRunStatus: "",
    onWorkspacePathChange: noop,
    onCreateWorkspaceChange: noop,
    onSelectWorkspace: noop,
    onOpenRecentWorkspace: noop,
    onForgetRecentWorkspace: noop,
    onRepoChange: noop,
    onAddRepo: noop,
    onRemoveRepo: noop,
    onDocsImportsPathChange: noop,
    onSaveSources: noop,
    onRuntimeChange: noop,
    onRuntimeProviderChange: noop,
    onSaveRuntime: noop,
    onCheckDoctor: noop,
    onEnterConsole: noop,
    onRunFirstAnalysis: noop,
    ...overrides,
  };
  render(<OnboardingShell {...props} />);
}

describe("OnboardingShell accessibility announcements", () => {
  it("links repo field errors to their inputs", () => {
    renderShell();

    const diagnostics = screen.getByTestId("onboarding-repo-diagnostics");
    expect(diagnostics).toHaveAttribute("id", "onboardingRepoDiagnostics-repo-1");

    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Name")).toHaveAttribute("aria-describedby", "onboardingRepoDiagnostics-repo-1");
    expect(screen.getByLabelText("Repository URL")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Repository URL")).toHaveAttribute("aria-describedby", "onboardingRepoDiagnostics-repo-1");
    expect(screen.getAllByRole("alert").some((alert) => alert.textContent?.includes("repo_name_required"))).toBe(true);
  });

  it("announces errors assertively and success/progress politely", () => {
    const validateResult: ValidateResponse = {
      ok: true,
      workspace: "/work/acp",
    };
    const doctorResult: DoctorResponse = {
      ok: true,
      summary: "ready",
      checks: [],
    };
    renderShell({
      error: "Workspace open failed.",
      guidedRepos: [{ ...incompleteRepo, name: "provenarch", git_url: "https://example.test/repo.git" }],
      validateResult,
      doctorResult,
      firstRunStatus: "First analysis is starting.",
    });

    expect(screen.getByText("Workspace open failed.").closest('[role="alert"]')).toHaveAttribute("aria-live", "assertive");
    expect(screen.getByText("Sources validated.").closest('[role="status"]')).toHaveAttribute("aria-live", "polite");
    fireEvent.click(screen.getByRole("button", { name: /Runner/ }));
    expect(screen.getByText("Runner and local readiness passed.").closest('[role="status"]')).toHaveAttribute("aria-live", "polite");
    fireEvent.click(screen.getByRole("button", { name: /Review & start/ }));
    expect(screen.getByText("First analysis is starting.").closest('[role="status"]')).toHaveAttribute("aria-live", "polite");
  });
});
