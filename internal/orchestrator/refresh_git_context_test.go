package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/refreshplan"
)

func TestBuildRefreshIntentContextIsBoundedAndEvidenceFirst(t *testing.T) {
	changes := make([]refreshplan.PathChange, 0, 10_000)
	for i := 0; i < 10_000; i++ {
		changes = append(changes, refreshplan.PathChange{Status: "modified", Path: fmt.Sprintf("src/component/%05d/implementation.go", i), InScope: true})
	}
	contextText := buildRefreshIntentContext(context.Background(), refreshplan.ImpactPlan{RepoDeltas: []refreshplan.RepoDelta{{Repo: "repo", Changes: changes}}}, nil)
	if len(contextText) > maxRefreshIntentBytes {
		t.Fatalf("context exceeded bound: %d", len(contextText))
	}
	if !strings.Contains(contextText, "changed=modified:src/component/00000/implementation.go") || !strings.Contains(contextText, "observed evidence are authoritative") {
		excerpt := contextText
		if len(excerpt) > 300 {
			excerpt = excerpt[:300]
		}
		t.Fatalf("missing evidence-first context: %s", excerpt)
	}
}
