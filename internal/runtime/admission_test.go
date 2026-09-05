package runtime

import "testing"

func TestCloneAdmittedRuntimeSnapshotCopiesRepositoryScopes(t *testing.T) {
	original := &AdmittedRuntimeSnapshot{
		RepositoryScopes: []string{"payments-service", "billing-service"},
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
}
