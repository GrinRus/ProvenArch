package api

import (
	"reflect"
	"testing"
)

func TestAPIRouteGroupsPreserveCanonicalInventory(t *testing.T) {
	server := &Server{}
	groups := server.apiRouteGroups()
	if len(groups) != 7 {
		t.Fatalf("route group count = %d, want 7", len(groups))
	}

	gotGroups := make([]string, 0, len(groups))
	gotPatterns := []string{}
	seen := map[string]string{}
	for _, group := range groups {
		if group.name == "" || len(group.routes) == 0 {
			t.Fatalf("route group must have a name and routes: %+v", group)
		}
		gotGroups = append(gotGroups, group.name)
		for _, route := range group.routes {
			if route.pattern == "" || route.handler == nil {
				t.Fatalf("route in group %q is incomplete: %+v", group.name, route)
			}
			if previous, exists := seen[route.pattern]; exists {
				t.Fatalf("route %q registered by both %q and %q", route.pattern, previous, group.name)
			}
			seen[route.pattern] = group.name
			gotPatterns = append(gotPatterns, route.pattern)
		}
	}

	wantGroups := []string{
		"system-and-onboarding",
		"workspace",
		"runtime",
		"artifacts-and-git",
		"qa",
		"pipeline",
		"tasks",
	}
	wantPatterns := []string{
		"/api/health",
		"/api/system/version",
		"/api/onboarding/status",
		"/api/onboarding/workspace",
		"/api/onboarding/runtime",
		"/api/onboarding/enter-console",
		"/api/onboarding/path-suggestions",
		"/api/onboarding/recent-workspaces/forget",
		"/api/system/info",
		"/api/system/doctor",
		"/api/workspace/health",
		"/api/workspace/validate",
		"/api/workspace/bundle",
		"/api/workspace/manifest",
		"/api/runtime/timeouts",
		"/api/runtime/execution",
		"/api/runtime/models",
		"/api/runtime/permissions",
		"/api/runtime/profile",
		"/api/artifacts",
		"/api/repository-evidence",
		"/api/artifacts/write",
		"/api/knowledge",
		"/api/architecture",
		"/api/git/diff",
		"/api/git/commit",
		"/api/git/proposal-branch",
		"/api/qa/ask",
		"/api/qa/runs",
		"/api/qa/runs/",
		"/api/pipeline/init",
		"/api/pipeline/refresh",
		"/api/pipeline/runs",
		"/api/pipeline/runs/",
		"/api/tasks",
		"/api/tasks/",
	}
	if !reflect.DeepEqual(gotGroups, wantGroups) {
		t.Fatalf("route groups changed: got=%v want=%v", gotGroups, wantGroups)
	}
	if !reflect.DeepEqual(gotPatterns, wantPatterns) {
		t.Fatalf("API route inventory changed: got=%v want=%v", gotPatterns, wantPatterns)
	}
}
