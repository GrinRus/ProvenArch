package orchestrator

import (
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/refreshplan"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestWriteRefreshMaterializationRecordsPreservedAndRemoved(t *testing.T) {
	ws := workspace.Root{Path: t.TempDir()}
	if err := ws.WriteFile("reports/as-is/current.md", []byte("current\n")); err != nil {
		t.Fatal(err)
	}
	baseline := "run-base"
	audit := refreshplan.RefreshExecution{Version: 1, RunID: "run-next", GeneratedAt: time.Now().UTC().Format(time.RFC3339), BaselineRunID: &baseline, PlanDecision: "selective_candidate", Mode: "affected_only", ReasonCodes: []string{}, SourceRanges: []refreshplan.SourceRange{}, AffectedShards: []string{"payments"}, PreservedShards: []string{"users"}, ArtifactPath: "reports/taskruns/run-next/refresh-execution.json"}
	_, counts, err := writeRefreshMaterialization(ws, "run-next", audit, []string{"reports/as-is/current.md"}, "updated", []string{"reports/as-is/current.md"}, []string{"reports/as-is/removed.md"}, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	if counts.Preserved != 1 || counts.Removed != 1 || counts.Updated != 0 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
}
