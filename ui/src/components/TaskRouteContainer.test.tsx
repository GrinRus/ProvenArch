import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TaskRouteContainer } from "./TaskRouteContainer";

describe("TaskRouteContainer", () => {
  it("renders a truthful inbox target without legacy run data", () => {
    render(<TaskRouteContainer view="inbox" />);
    expect(screen.getByRole("heading", { name: "Task Inbox" })).toBeInTheDocument();
    expect(screen.getByTestId("task-route-inbox")).toHaveTextContent("does not load or mutate Task data yet");
    expect(screen.getByTestId("task-route-inbox")).not.toHaveTextContent("latest run");
  });

  it("keeps exact Task and Attempt identities visible", () => {
    render(<TaskRouteContainer view="attempt" taskId="task-1" attemptId="attempt-2" />);
    expect(screen.getByTestId("task-route-identities")).toHaveTextContent("task-1");
    expect(screen.getByTestId("task-route-identities")).toHaveTextContent("attempt-2");
  });

  it("fails closed for an invalid deep link", () => {
    render(<TaskRouteContainer view="inbox" invalid={["task"]} />);
    expect(screen.getByTestId("task-route-invalid")).toHaveTextContent("No Task or Attempt was selected");
  });
});
