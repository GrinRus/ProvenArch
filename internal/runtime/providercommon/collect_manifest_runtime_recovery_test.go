package providercommon

import (
	"encoding/json"
	"strings"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestRecoveredCollectManifestKeepsExecutionDetailsOutOfSemanticArtifacts(t *testing.T) {
	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		ShardID:      "architecture",
		DomainID:     "architecture",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/architecture",
		RepoScope:    "example",
	}
	docs := []collectManifestRuntimeRecoveryDoc{{
		RelPath: "architecture.md",
		Title:   "Architecture",
		Text:    "# Architecture\n\nThe API is a service.",
		Terms:   []string{"API", "service"},
	}}
	manifest := buildRecoveredCollectManifest(task, docs, "example", "README.md")
	raw, err := json.Marshal(manifest.Semantic)
	if err != nil {
		t.Fatalf("marshal recovered manifest: %v", err)
	}
	text := strings.ToLower(string(raw))
	for _, forbidden := range []string{
		"reports/taskruns/",
		".acp/repos/",
		"draft_artifact_enrichment",
		"current run shard",
		"runtime recovery",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("recovered semantic artifact contains internal execution detail %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(manifest.Semantic.Coverage.Missing[0], "complete provider-authored manifest was unavailable") {
		t.Fatalf("expected neutral coverage gap, got %#v", manifest.Semantic.Coverage.Missing)
	}

	// Keep the original failure reason available to operators through the
	// recovery report; it must not be serialized into the semantic artifact.
	result := markCollectManifestRuntimeRecovered(acpruntime.Result{}, collectManifestRuntimeRecoveryReport{
		RecoveryCause: "open /private/tmp/reports/taskruns/run-1: unavailable",
	})
	diagnostics, ok := result.Diagnostics["collect_manifest_runtime_recovery"].(map[string]any)
	if !ok || diagnostics["recovery_cause"] == "" {
		t.Fatalf("expected recovery cause in diagnostics, got %#v", result.Diagnostics)
	}
}
