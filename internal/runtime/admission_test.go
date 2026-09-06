package runtime

import "testing"

func TestCloneAdmittedRuntimeSnapshotCopiesRepositoryScopes(t *testing.T) {
	original := &AdmittedRuntimeSnapshot{
		RepositoryScopes:     []string{"payments-service", "billing-service"},
		RepositoryPathScopes: map[string][]string{"payments-service": {"src", "docs"}},
	}
	clone := CloneAdmittedRuntimeSnapshot(original)
	if clone == nil {
		t.Fatal("expected a snapshot clone")
	}

	original.RepositoryScopes[0] = "mutated-original"
	if clone.RepositoryScopes[0] != "payments-service" {
		t.Fatalf("clone shares repository scopes with source: %v", clone.RepositoryScopes)
	}

	clone.RepositoryScopes[1] = "mutated-clone"
	if original.RepositoryScopes[1] != "billing-service" {
		t.Fatalf("source shares repository scopes with clone: %v", original.RepositoryScopes)
	}
	original.RepositoryPathScopes["payments-service"][0] = "mutated-original"
	if clone.RepositoryPathScopes["payments-service"][0] != "src" {
		t.Fatalf("clone shares repository path scopes with source: %v", clone.RepositoryPathScopes)
	}
	clone.RepositoryPathScopes["payments-service"][1] = "mutated-clone"
	if original.RepositoryPathScopes["payments-service"][1] != "docs" {
		t.Fatalf("source shares repository path scopes with clone: %v", original.RepositoryPathScopes)
	}
}
