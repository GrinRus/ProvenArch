package artifactquality

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestEnsureCanonicalCollectManifestRepairsInvalidExtrasFixture(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "bank_extras_invalid_manifest.json")
	writeDoc(t, writeRoot, "extras-overview.md", "# Extras\n")

	task := testCollectTask(writeRoot, "bank-of-anthos-extras", "bank-of-anthos")
	result := testCollectResult(task.TaskID, task.StepID, task.RunID, "bank-of-anthos")

	if err := EnsureCanonicalCollectManifest(task, result); err != nil {
		t.Fatalf("canonicalize extras fixture: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read canonicalized manifest: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		t.Fatalf("parse canonicalized manifest: %v", err)
	}
	if got := manifest.Compatibility.Coverage.Observed; len(got) == 0 || got[0] != "services" {
		t.Fatalf("expected compatibility coverage from TaskResult snapshot, got %#v", got)
	}
	assessment, err := ValidateCollectManifestAtWriteRoot(writeRoot)
	if err != nil {
		t.Fatalf("validate collect manifest readability: %v", err)
	}
	if !assessment.Rich {
		t.Fatalf("expected repaired extras manifest to be rich, got %#v", assessment)
	}
}

func TestEnsureCanonicalCollectManifestRepairsInvalidKubernetesFixture(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "bank_kubernetes_invalid_manifest.json")
	writeDoc(t, writeRoot, "kubernetes-manifests.md", "# Kubernetes\n")

	task := testCollectTask(writeRoot, "bank-of-anthos-kubernetes-manifests", "bank-of-anthos")
	result := testCollectResult(task.TaskID, task.StepID, task.RunID, "bank-of-anthos")

	if err := EnsureCanonicalCollectManifest(task, result); err != nil {
		t.Fatalf("canonicalize kubernetes fixture: %v", err)
	}

	assessment, err := ValidateCollectManifestAtWriteRoot(writeRoot)
	if err != nil {
		t.Fatalf("validate collect manifest readability: %v", err)
	}
	if !assessment.Rich {
		t.Fatalf("expected repaired kubernetes manifest to be rich, got %#v", assessment)
	}
}

func TestEnsureCanonicalCollectManifestSynthesizesMissingManifestFromWriteRootDocs(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeDoc(t, writeRoot, "services.md", "# Services\n")

	task := testCollectTask(writeRoot, "bank-of-anthos-src", "bank-of-anthos")
	result := testCollectResult(task.TaskID, task.StepID, task.RunID, "bank-of-anthos")

	if err := EnsureCanonicalCollectManifest(task, result); err != nil {
		t.Fatalf("synthesize missing manifest: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read synthesized manifest: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		t.Fatalf("parse synthesized manifest: %v", err)
	}
	if len(manifest.Documents) == 0 {
		t.Fatalf("expected synthesized manifest documents")
	}
	if got := strings.TrimSpace(manifest.Documents[0].Path); got != "services.md" {
		t.Fatalf("expected synthesized doc path services.md, got %q", got)
	}
	if _, err := ValidateCollectManifestAtWriteRoot(writeRoot); err != nil {
		t.Fatalf("validate synthesized manifest readability: %v", err)
	}
}

func TestEnsureCanonicalCollectManifestNormalizesStagingPrefixedDocumentPaths(t *testing.T) {
	t.Parallel()

	writeRoot := t.TempDir()
	writeFixtureManifest(t, writeRoot, "openedx_staging_prefixed_path_manifest.json")
	writeDoc(t, writeRoot, "service-inventory.md", "# Service Inventory\n")

	task := testCollectTask(writeRoot, "openedx-platform", "openedx-platform")
	task.ArtifactRoot = "reports/taskruns/run-1/staging/shards/openedx-platform"
	result := testCollectResult(task.TaskID, task.StepID, task.RunID, "openedx-platform")

	if err := EnsureCanonicalCollectManifest(task, result); err != nil {
		t.Fatalf("normalize staging-prefixed path: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(writeRoot, shardPackManifestFile))
	if err != nil {
		t.Fatalf("read normalized manifest: %v", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		t.Fatalf("parse normalized manifest: %v", err)
	}
	if len(manifest.Documents) != 1 {
		t.Fatalf("expected one normalized document, got %d", len(manifest.Documents))
	}
	if got := manifest.Documents[0].Path; got != "service-inventory.md" {
		t.Fatalf("expected normalized relative document path, got %q", got)
	}
	if _, err := ValidateCollectManifestAtWriteRoot(writeRoot); err != nil {
		t.Fatalf("validate normalized manifest readability: %v", err)
	}
}

func testCollectTask(writeRoot string, shardID string, repo string) acpruntime.Task {
	return acpruntime.Task{
		TaskID:       "task-" + shardID,
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    tTempWorkspace(writeRoot),
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/" + shardID,
		ShardID:      shardID,
		DomainID:     shardID,
		RepoScope:    repo,
		RepoScopes:   []string{repo},
		PathScopes:   []string{"."},
		StartedAtUTC: time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC),
	}
}

func testCollectResult(taskID string, stepID string, runID string, repo string) contracts.TaskResult {
	return contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:    taskID,
			StepID:    stepID,
			RunID:     runID,
			Runtime:   contracts.RuntimeMeta{Name: "qwen-code", Version: "test"},
			StartedAt: "2026-04-19T09:00:00Z",
			RepoScope: repo,
		},
		Summary:   "collect canonicalized",
		Changeset: []contracts.Operation{},
		Questions: []contracts.Question{
			{ID: "q.refresh.delta", Text: "What changed?", Priority: "high"},
		},
		Coverage: &contracts.Coverage{
			Observed: []string{"services"},
			Missing:  []string{"owner mappings", "runtime metrics", "dependencies"},
			Notes:    []string{"canonicalized from fixture"},
		},
	}
}

func writeFixtureManifest(t *testing.T, writeRoot string, fixtureName string) {
	t.Helper()
	raw := readFixture(t, fixtureName)
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
}

func readFixture(t *testing.T, fixtureName string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", fixtureName))
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixtureName, err)
	}
	return raw
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

func tTempWorkspace(writeRoot string) string {
	return filepath.Clean(filepath.Join(writeRoot, "..", "workspace"))
}
