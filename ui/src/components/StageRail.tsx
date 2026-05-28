import { useState, type KeyboardEvent } from "react";

import type { StageId, StageOption } from "../lib/consoleTypes";

type StageRailProps = {
  stages: StageOption[];
  activeStage: StageId;
  onStageChange: (stage: StageId) => void;
};

const stageGlyphs: Record<StageOption["status"], string> = {
  done: "✓",
  active: "●",
  blocked: "!",
  pending: "○",
};

const stageIconPaths: Record<StageId, string[]> = {
  source: ["M4 7h5l2 2h9v9H4z", "M4 7V5h6l2 2"],
  readiness: ["M12 3a9 9 0 1 0 0 18a9 9 0 0 0 0-18", "M8 12l2.5 2.5L16 9"],
  charter: ["M7 3h7l4 4v14H7z", "M14 3v5h5", "M10 12h5", "M10 16h6"],
  analysis: ["M11 4a7 7 0 1 0 0 14a7 7 0 0 0 0-14", "M16 16l4 4"],
  review: ["M3 12s3.5-5.5 9-5.5S21 12 21 12s-3.5 5.5-9 5.5S3 12 3 12", "M12 9a3 3 0 1 0 0 6a3 3 0 0 0 0-6"],
  proposals: ["M6 4v16", "M6 8h7a5 5 0 0 1 5 5v7", "M6 16h6"],
  ask: ["M9 9a3 3 0 1 1 4.2 2.8c-.9.5-1.2 1.2-1.2 2.2", "M12 18h.01", "M4 5h16v16H4z"],
  publish: ["M12 16V4", "M7 9l5-5 5 5", "M5 20h14"],
};

export function StageRail({ stages, activeStage, onStageChange }: StageRailProps) {
  const [collapsed, setCollapsed] = useState(false);

  function focusStage(stage: StageOption) {
    const testId = stage.testId ?? `stage-${stage.id}`;
    window.requestAnimationFrame(() => {
      document.querySelector<HTMLButtonElement>(`[data-testid="${testId}"]`)?.focus();
    });
  }

  function moveFocusToStage(index: number) {
    const boundedIndex = Math.max(0, Math.min(stages.length - 1, index));
    const nextStage = stages[boundedIndex];
    onStageChange(nextStage.id);
    focusStage(nextStage);
  }

  function handleStageKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    switch (event.key) {
      case "ArrowDown":
      case "ArrowRight":
        event.preventDefault();
        moveFocusToStage(index + 1);
        break;
      case "ArrowUp":
      case "ArrowLeft":
        event.preventDefault();
        moveFocusToStage(index - 1);
        break;
      case "Home":
        event.preventDefault();
        moveFocusToStage(0);
        break;
      case "End":
        event.preventDefault();
        moveFocusToStage(stages.length - 1);
        break;
    }
  }

  return (
    <nav className={`stage-rail ${collapsed ? "is-collapsed" : ""}`} aria-label="Proven Arch workflow" data-testid="stage-rail">
      <p className="rail-title">Workflow</p>
      {stages.map((stage, index) => (
        <button
          key={stage.id}
          type="button"
          className={`stage-row ${stage.status} ${stage.id === activeStage ? "is-selected" : ""}`}
          aria-label={`${stage.label}: ${stage.description}; ${stage.status}`}
          aria-current={stage.id === activeStage ? "step" : undefined}
          onClick={() => onStageChange(stage.id)}
          onKeyDown={(event) => handleStageKeyDown(event, index)}
          data-testid={stage.testId ?? `stage-${stage.id}`}
          title={`${stage.label}: ${stage.description}`}
        >
          <span className="stage-index" aria-hidden="true">
            {index + 1}
          </span>
          <StageIcon stage={stage.id} />
          <span className="stage-copy" aria-hidden="true">
            <span className="stage-label">{stage.label}</span>
            <span className="stage-description">{stage.description}</span>
          </span>
          <span className="stage-state" aria-hidden="true">
            {stage.count ? stage.count : stageGlyphs[stage.status]}
          </span>
        </button>
      ))}
      <button
        type="button"
        className="rail-collapse-btn"
        aria-pressed={collapsed}
        aria-label={collapsed ? "Expand workflow rail" : "Collapse workflow rail"}
        onClick={() => setCollapsed((value) => !value)}
        data-testid="stage-rail-collapse-btn"
      >
        <span className="rail-collapse-icon" aria-hidden="true">
          {collapsed ? "›" : "‹"}
        </span>
        <span className="rail-collapse-label">{collapsed ? "Expand" : "Collapse"}</span>
      </button>
    </nav>
  );
}

function StageIcon({ stage }: { stage: StageId }) {
  return (
    <span className="stage-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" role="img" focusable="false">
        {stageIconPaths[stage].map((path) => (
          <path key={path} d={path} />
        ))}
      </svg>
    </span>
  );
}
