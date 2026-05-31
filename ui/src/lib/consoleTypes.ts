export type StageId = "source" | "readiness" | "charter" | "analysis" | "review" | "proposals" | "ask" | "publish";

export type StageStatus = "done" | "active" | "blocked" | "pending";

export type Severity = "info" | "ok" | "warn" | "error";

export type StageOption = {
  id: StageId;
  label: string;
  description: string;
  status: StageStatus;
  count?: number;
  testId?: string;
};

export type NextAction = {
  label: string;
  description: string;
  primaryActionId?: StageId;
  intent?: "focus-analysis-blocker" | "submit-ask";
  disabledReason?: string;
};

export type InspectorItem = {
  severity: Severity;
  label: string;
  detail: string;
  path?: string;
};
