import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { QAFailureRecovery } from "./QAFailureRecovery";

describe("QAFailureRecovery", () => {
  it("stays absent for a non-failed answer run", () => {
    const { container } = render(<QAFailureRecovery qaRun={null} busy={false} onRetry={() => undefined} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("keeps failed answer recovery actionable", () => {
    render(<QAFailureRecovery qaRun={{ run_id: "qa-1", status: "failed", error_code: "runtime_failed", current_step: "qa.ask", question: "Why?", warnings: ["warning"], error: "failed" } as never} busy={false} onRetry={() => undefined} />);
    expect(screen.getByTestId("qa-failure-recovery")).toHaveTextContent("Retry question");
    expect(screen.getByTestId("qa-retry-run-btn")).toBeEnabled();
  });
});
