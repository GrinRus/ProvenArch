package artifactquality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestRepairCollectManifestRejectsLegacyCompatibilityPayload(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "bank_extras_invalid_manifest.json")
	writeDoc(t, writeRoot, "extras-overview.md", "# Extras\n")

	task := testCollectTask(writeRoot, "bank-of-anthos-extras", "bank-of-anthos")
	if err := RepairCollectManifest(task); err == nil {
		t.Fatalf("expected repair to fail for legacy compatibility payload")
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read original manifest: %v", err)
	}
	if _, err := contracts.ParseShardPackManifest(raw); err == nil {
		t.Fatalf("expected original manifest to remain invalid")
	}
}

func TestRepairCollectManifestNormalizesArtifactRootPrefixedDocumentPath(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "bank_of_anthos_doubled_path_manifest.json")
	writeDoc(t, writeRoot, "iac-overview.md", "# IAC Overview\n")

	task := testCollectTask(writeRoot, "bank-of-anthos-iac", "bank-of-anthos")
	task.RunID = "run_20260420_054749_001"
	task.StepID = "init.step1.collect"
	task.ArtifactRoot = "reports/taskruns/run_20260420_054749_001/staging/shards/bank-of-anthos-iac"
	task.PathScopes = []string{"iac"}

	if err := RepairCollectManifest(task); err != nil {
		t.Fatalf("repair manifest: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read repaired manifest: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		t.Fatalf("parse repaired manifest: %v", err)
	}
	if got := manifest.Documents[0].Path; got != "iac-overview.md" {
		t.Fatalf("expected normalized document path, got %q", got)
	}

	assessment, err := LoadManifestAssessment(writeRoot)
	if err != nil {
		t.Fatalf("load manifest assessment: %v", err)
	}
	if !assessment.Rich {
		t.Fatalf("expected repaired manifest to remain rich, got %#v", assessment)
	}
}

func TestRepairCollectManifestNormalizesAbsoluteDocumentPathUnderWriteRoot(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeDoc(t, writeRoot, "service-inventory.md", "# Service Inventory\n")

	manifest := contracts.ShardPackManifest{
		Version:      1,
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		ShardID:      "openedx-platform",
		AgentRole:    "shard-analyst",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/openedx-platform",
		RepoScopes:   []string{"openedx-platform"},
		PathScopes:   []string{"."},
		Documents: []contracts.AuthoredDocument{
			{
				ID:            "doc.service-inventory",
				Kind:          "report",
				Title:         "Service Inventory",
				Path:          filepath.Join(writeRoot, "service-inventory.md"),
				CanonicalPath: "reports/as-is/service-inventory/openedx-platform.md",
				Topics:        []string{"openedx"},
				CitationIDs:   []string{"cite.openedx.readme"},
			},
		},
		Citations: []contracts.DocumentCitation{
			{
				ID:          "cite.openedx.readme",
				Repo:        "openedx-platform",
				Path:        "README.md",
				ClaimIDs:    []string{"claim.openedx.readme"},
				DocumentIDs: []string{"doc.service-inventory"},
			},
		},
		Semantic: contracts.SemanticSnapshot{
			Coverage: contracts.Coverage{
				Observed: []string{"services"},
				Missing:  []string{"owner mappings"},
				Notes:    []string{"absolute path drift"},
			},
			Questions: []contracts.Question{},
			Entities:  []contracts.Entity{},
			Edges:     []contracts.Edge{},
			Findings:  []contracts.Finding{},
		},
	}
	writeManifest(t, writeRoot, manifest)

	task := testCollectTask(writeRoot, "openedx-platform", "openedx-platform")
	if err := RepairCollectManifest(task); err != nil {
		t.Fatalf("repair manifest: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read repaired manifest: %v", err)
	}
	repaired, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		t.Fatalf("parse repaired manifest: %v", err)
	}
	if got := repaired.Documents[0].Path; got != "service-inventory.md" {
		t.Fatalf("expected absolute path to normalize to write-root relative path, got %q", got)
	}
}

func testCollectTask(writeRoot string, shardID string, repo string) acpruntime.Task {
	return acpruntime.Task{
		TaskID:       "task-" + shardID,
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    filepath.Clean(filepath.Join(writeRoot, "..", "workspace")),
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/" + shardID,
		ShardID:      shardID,
		DomainID:     shardID,
		AgentRole:    "shard-analyst",
		RepoScope:    repo,
		RepoScopes:   []string{repo},
		PathScopes:   []string{"."},
		StartedAtUTC: time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC),
	}
}

func writeFixtureManifest(t *testing.T, writeRoot string, fixtureName string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", fixtureName))
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixtureName, err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
}

func writeManifest(t *testing.T, writeRoot string, manifest contracts.ShardPackManifest) {
	t.Helper()
	raw, err := jsonMarshalManifest(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func jsonMarshalManifest(manifest contracts.ShardPackManifest) ([]byte, error) {
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func writeDoc(t *testing.T, writeRoot string, rel string, content string) {
	t.Helper()
	abs := filepath.Join(writeRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for doc %q: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write doc %q: %v", rel, err)
	}
}
