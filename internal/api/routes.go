package api

import "net/http"

// apiRouteGroup keeps route ownership close to the capability it exposes. The
// Handler method owns only the transport envelope; endpoint implementation and
// session gating remain on Server methods in their domain files.
type apiRouteGroup struct {
	name   string
	routes []apiRoute
}

type apiRoute struct {
	pattern string
	handler http.HandlerFunc
}

func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	for _, group := range s.apiRouteGroups() {
		for _, route := range group.routes {
			mux.HandleFunc(route.pattern, route.handler)
		}
	}
}

func (s *Server) apiRouteGroups() []apiRouteGroup {
	return []apiRouteGroup{
		{
			name: "system-and-onboarding",
			routes: []apiRoute{
				{pattern: "/api/health", handler: s.handleHealth},
				{pattern: "/api/system/version", handler: s.handleSystemVersion},
				{pattern: "/api/onboarding/status", handler: s.handleOnboardingStatus},
				{pattern: "/api/onboarding/workspace", handler: s.handleOnboardingWorkspace},
				{pattern: "/api/onboarding/runtime", handler: s.handleOnboardingRuntime},
				{pattern: "/api/onboarding/enter-console", handler: s.handleOnboardingEnterConsole},
				{pattern: "/api/onboarding/path-suggestions", handler: s.handleOnboardingPathSuggestions},
				{pattern: "/api/onboarding/recent-workspaces/forget", handler: s.handleOnboardingRecentWorkspaceForget},
				{pattern: "/api/system/info", handler: s.handleSystemInfo},
				{pattern: "/api/system/doctor", handler: s.handleSystemDoctor},
			},
		},
		{
			name: "workspace",
			routes: []apiRoute{
				{pattern: "/api/workspace/health", handler: s.handleWorkspaceHealth},
				{pattern: "/api/workspace/validate", handler: s.handleWorkspaceValidate},
				{pattern: "/api/workspace/bundle", handler: s.handleWorkspaceBundle},
				{pattern: "/api/workspace/manifest", handler: s.handleWorkspaceManifest},
			},
		},
		{
			name: "runtime",
			routes: []apiRoute{
				{pattern: "/api/runtime/timeouts", handler: s.handleRuntimeTimeouts},
				{pattern: "/api/runtime/execution", handler: s.handleRuntimeExecution},
				{pattern: "/api/runtime/models", handler: s.handleRuntimeModels},
				{pattern: "/api/runtime/permissions", handler: s.handleRuntimePermissions},
				{pattern: "/api/runtime/profile", handler: s.handleRuntimeProfile},
			},
		},
		{
			name: "artifacts-and-git",
			routes: []apiRoute{
				{pattern: "/api/artifacts", handler: s.handleArtifacts},
				{pattern: "/api/repository-evidence", handler: s.handleRepositoryEvidence},
				{pattern: "/api/artifacts/write", handler: s.handleArtifactsWrite},
				{pattern: "/api/knowledge", handler: s.handleKnowledge},
				{pattern: "/api/architecture", handler: s.handleArchitecture},
				{pattern: "/api/git/diff", handler: s.handleGitDiff},
				{pattern: "/api/git/commit", handler: s.handleGitCommit},
				{pattern: "/api/git/proposal-branch", handler: s.handleGitProposalBranch},
			},
		},
		{
			name: "qa",
			routes: []apiRoute{
				{pattern: "/api/qa/ask", handler: s.handleQAAsk},
				{pattern: "/api/qa/runs", handler: s.handleQARuns},
				{pattern: "/api/qa/runs/", handler: s.handleQARuns},
			},
		},
		{
			name: "pipeline",
			routes: []apiRoute{
				{pattern: "/api/pipeline/init", handler: s.handlePipelineInit},
				{pattern: "/api/pipeline/refresh", handler: s.handlePipelineRefresh},
				{pattern: "/api/pipeline/runs", handler: s.handlePipelineRuns},
				{pattern: "/api/pipeline/runs/", handler: s.handlePipelineRuns},
			},
		},
		{
			name: "tasks",
			routes: []apiRoute{
				{pattern: "/api/tasks", handler: s.handleTasks},
				{pattern: "/api/tasks/", handler: s.handleTasks},
			},
		},
	}
}
