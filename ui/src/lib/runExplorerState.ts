import type { RunListItem, RunStatusResponse } from "./appContracts";

export type RunExplorerState = {
  runId: string | null;
  runStatus: RunStatusResponse | null;
  runList: RunListItem[];
  runActionStatus: string;
  cancelBusy: boolean;
};

export type RunExplorerAction =
  | { type: "setRunID"; runId: string | null }
  | { type: "setRunStatus"; runStatus: RunStatusResponse | null }
  | { type: "setRunList"; runList: RunListItem[] }
  | { type: "upsertRunListItem"; item: RunListItem }
  | { type: "clearRunStatusForRun"; runId: string }
  | { type: "setRunActionStatus"; runActionStatus: string }
  | { type: "setCancelBusy"; cancelBusy: boolean };

export const initialRunExplorerState: RunExplorerState = {
  runId: null,
  runStatus: null,
  runList: [],
  runActionStatus: "",
  cancelBusy: false,
};

export function runExplorerReducer(state: RunExplorerState, action: RunExplorerAction): RunExplorerState {
  switch (action.type) {
    case "setRunID":
      return { ...state, runId: action.runId };
    case "setRunStatus":
      return { ...state, runStatus: action.runStatus };
    case "setRunList":
      return { ...state, runList: action.runList };
    case "upsertRunListItem":
      return {
        ...state,
        runList: [action.item, ...state.runList.filter((run) => run.run_id !== action.item.run_id)],
      };
    case "clearRunStatusForRun":
      if (state.runStatus?.run_id !== action.runId) {
        return state;
      }
      return { ...state, runStatus: null };
    case "setRunActionStatus":
      return { ...state, runActionStatus: action.runActionStatus };
    case "setCancelBusy":
      return { ...state, cancelBusy: action.cancelBusy };
    default:
      return state;
  }
}
