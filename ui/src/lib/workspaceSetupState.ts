import { makeGuidedRepo, type GuidedRepo } from "./appContracts";

export type GuidedReposAction =
  | { type: "update"; id: string; patch: Partial<GuidedRepo> }
  | { type: "add" }
  | { type: "remove"; id: string }
  | { type: "replace"; repos: GuidedRepo[] };

export type GuidedSetupFromManifest = {
  repos: GuidedRepo[];
  docsImportsPath?: string;
};

export function initialGuidedRepos(): GuidedRepo[] {
  return [
    makeGuidedRepo({
      name: "my-service",
      mode: "git_url",
      git_url: "https://github.com/org/my-service.git",
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
    case "replace":
      return action.repos.length > 0 ? action.repos : state;
    default:
      return state;
  }
}

export function parseGuidedSetupFromManifest(content: string): GuidedSetupFromManifest | null {
  const repos: GuidedRepo[] = [];
  let currentRepo: Partial<GuidedRepo> | null = null;
  let section = "";
  let docsImportsPath: string | undefined;

  const flushRepo = () => {
    if (!currentRepo?.name) {
      currentRepo = null;
      return;
    }
    if (currentRepo.git_url) {
      repos.push(
        makeGuidedRepo({
          name: currentRepo.name,
          mode: "git_url",
          git_url: currentRepo.git_url,
          ref: currentRepo.ref ?? "",
        }),
      );
    } else if (currentRepo.path) {
      repos.push(
        makeGuidedRepo({
          name: currentRepo.name,
          mode: "path",
          path: currentRepo.path,
          ref: currentRepo.ref ?? "",
        }),
      );
    }
    currentRepo = null;
  };

  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.replace(/\s+#.*$/, "");
    if (!line.trim()) {
      continue;
    }

    const topLevel = /^([A-Za-z_][A-Za-z0-9_]*):\s*$/.exec(line);
    if (topLevel) {
      flushRepo();
      section = topLevel[1];
      continue;
    }

    if (section === "repos") {
      const repoStart = /^\s*-\s*(.*)$/.exec(line);
      if (repoStart) {
        flushRepo();
        currentRepo = {};
        const inline = repoStart[1].trim();
        if (inline) {
          applyRepoField(currentRepo, inline);
        }
        continue;
      }

      const repoField = /^\s+([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$/.exec(line);
      if (repoField && currentRepo) {
        applyRepoField(currentRepo, `${repoField[1]}: ${repoField[2]}`);
      }
      continue;
    }

    if (section === "docs") {
      const docsField = /^\s+imports_path:\s*(.*)$/.exec(line);
      if (docsField) {
        docsImportsPath = parseYAMLScalar(docsField[1]);
      }
    }
  }

  flushRepo();
  if (repos.length === 0 && docsImportsPath === undefined) {
    return null;
  }
  return { repos, docsImportsPath };
}

function applyRepoField(repo: Partial<GuidedRepo>, field: string) {
  const match = /^([A-Za-z_][A-Za-z0-9_]*):\s*(.*)$/.exec(field);
  if (!match) {
    return;
  }
  const value = parseYAMLScalar(match[2]);
  switch (match[1]) {
    case "name":
      repo.name = value;
      break;
    case "path":
      repo.mode = "path";
      repo.path = value;
      break;
    case "git_url":
      repo.mode = "git_url";
      repo.git_url = value;
      break;
    case "ref":
      repo.ref = value;
      break;
  }
}

function parseYAMLScalar(raw: string): string {
  const value = raw.trim();
  if (value.startsWith('"') && value.endsWith('"')) {
    try {
      return JSON.parse(value) as string;
    } catch {
      return value.slice(1, -1);
    }
  }
  if (value.startsWith("'") && value.endsWith("'")) {
    return value.slice(1, -1).replace(/''/g, "'");
  }
  return value;
}
