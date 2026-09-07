import type { RunCoordination, RunListItem, RunStatusResponse } from "./appContracts";

export type RunExplorerState = {
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runList: RunListItem[];
  coordination: RunCoordination;
  runActionStatus: string;
};

export type RunExplorerAction =
  | { type: "setRunID"; runId: string | null }
  | { type: "setRunStatus"; runStatus: RunStatusResponse | null }
  | { type: "setRunList"; runList: RunListItem[] }
  | { type: "setCoordination"; coordination: RunCoordination }
  | { type: "clearRunStatusForRun"; runId: string }
  | { type: "setRunActionStatus"; runActionStatus: string };

export const initialRunExplorerState: RunExplorerState = {
  runId: null,
  runStatus: null,
  runList: [],
  coordination: {},
  runActionStatus: "",
};

export function runExplorerReducer(state: RunExplorerState, action: RunExplorerAction): RunExplorerState {
  switch (action.type) {
    case "setRunID":
      return { ...state, runId: action.runId };
    case "setRunStatus":
      return { ...state, runStatus: action.runStatus };
    case "setRunList":
      return { ...state, runList: action.runList };
    case "setCoordination":
      return { ...state, coordination: action.coordination };
    case "clearRunStatusForRun":
      if (state.runStatus?.run_id !== action.runId) {
        return state;
      }
      return { ...state, runStatus: null };
    case "setRunActionStatus":
      return { ...state, runActionStatus: action.runActionStatus };
    default:
      return state;
  }
}
