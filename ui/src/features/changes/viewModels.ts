import type { ChangesView, RouteSource } from "../../lib/appRoutes";

export type ChangesRouteModel =
  | { kind: "overview"; source: RouteSource; title: "Review overview"; purpose: string }
  | { kind: "evidence"; source: RouteSource; title: "Evidence"; purpose: string }
  | { kind: "findings"; source: RouteSource; title: "Findings"; purpose: string }
  | { kind: "diff"; source: RouteSource; title: "Workspace diff"; purpose: string }
  | { kind: "proposals"; source: RouteSource; title: "Proposals"; purpose: string }
  | { kind: "publish"; source: RouteSource; title: "Publish"; purpose: string };

export function buildChangesRouteModel(view: ChangesView, source: RouteSource): ChangesRouteModel {
  switch (view) {
    case "overview":
      return { kind: view, source, title: "Review overview", purpose: "Run identity, coverage and review readiness." };
    case "evidence":
      return { kind: view, source, title: "Evidence", purpose: "Selected immutable evidence and artifact navigation." };
    case "findings":
      return { kind: view, source, title: "Findings", purpose: "Findings, questions and decision blockers." };
    case "diff":
      return { kind: view, source, title: "Workspace diff", purpose: "Server-authoritative current Git changes." };
    case "proposals":
      return { kind: view, source, title: "Proposals", purpose: "Proposed follow-up changes and evidence." };
    case "publish":
      return { kind: view, source, title: "Publish", purpose: "Publication gate and Git mutations." };
  }
}
