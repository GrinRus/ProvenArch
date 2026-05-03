import { makeGuidedRepo, type GuidedRepo } from "./appContracts";

export type GuidedReposAction =
  | { type: "update"; id: string; patch: Partial<GuidedRepo> }
  | { type: "add" }
  | { type: "remove"; id: string };

export function initialGuidedRepos(): GuidedRepo[] {
  return [
    makeGuidedRepo({
      name: "payments-service",
      mode: "path",
      path: "/absolute/path/to/payments-service",
    }),
  ];
}

export function guidedReposReducer(state: GuidedRepo[], action: GuidedReposAction): GuidedRepo[] {
  switch (action.type) {
    case "update":
      return state.map((repo) => (repo.id === action.id ? { ...repo, ...action.patch } : repo));
    case "add":
      return [...state, makeGuidedRepo()];
    case "remove":
      if (state.length <= 1) {
        return state;
      }
      return state.filter((repo) => repo.id !== action.id);
    default:
      return state;
  }
}
