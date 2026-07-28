package providercommon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestRunHeadlessProviderCompletesMissingSemanticFindingsWithoutProviderRepair(t *testing.T) {
	t.Parallel()
	task := newCollectTask(t, "run-collect-missing-findings")
	manifest := missingFindingsFixture(t, task)
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) { diagnostics = append(diagnostics, event) }
	script := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + ` <<'EOF'
# Bank architecture overview

The service entrypoint and bounded runtime surface are documented in README.md.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + manifest + `
EOF
`
	runner := &manifestRepairSequenceAdapter{testAdapter: testAdapter{
		command:  writeEngineScript(t, script),
		recovery: RecoveryPolicy{AcceptValidArtifactsAfterStop: true, RepairCollectManifestOnce: true},
	}}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("run provider: %v", err)
	}
	if runner.manifestRepairCalls != 0 {
		t.Fatalf("provider manifest repair calls = %d, want 0", runner.manifestRepairCalls)
	}
	if !hasDiagnosticField(diagnostics, "collect manifest missing findings recovery completed", "recovery_mode", collectManifestMissingFindingsRecoveryMode) {
		t.Fatalf("missing recovery diagnostic: %#v", diagnostics)
	}
	if !hasRuntimeWarning(result.Execution.Warnings, "missing_findings_recovery inserted an empty semantic.findings") {
		t.Fatalf("missing recovery warning: %#v", result.Execution.Warnings)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if err := artifactquality.ValidateCollectManifestInRoot(task.WriteRoot); err != nil {
		t.Fatalf("completed manifest is invalid: %v", err)
	}
	assertOnlyFindingsInserted(t, []byte(manifest), raw)
	recovery, ok := result.Diagnostics[collectManifestMissingFindingsRecoveryMode].(map[string]any)
	if !ok || recovery["before_digest"] == "" || recovery["after_digest"] == "" || recovery["before_digest"] == recovery["after_digest"] {
		t.Fatalf("invalid recovery digests: %#v", result.Diagnostics)
	}
}

func TestMissingSemanticFindingsRecoveryRollsBackWhenAnotherContractErrorRemains(t *testing.T) {
	t.Parallel()
	task := newCollectTask(t, "run-collect-missing-findings-invalid")
	if err := os.MkdirAll(task.WriteRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "overview.md"), []byte("# Bank overview\n\nObserved README.md entrypoint.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	original := strings.Replace(missingFindingsFixture(t, task), `"text": "Who owns bank-of-anthos?"`, `"question": "Who owns bank-of-anthos?"`, 1)
	manifestPath := filepath.Join(task.WriteRoot, ShardPackManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	cause := artifactquality.ValidateCollectManifestInRoot(task.WriteRoot)
	if cause == nil {
		t.Fatal("expected invalid fixture")
	}
	if _, err := recoverCollectManifestMissingFindings(task, cause); err == nil {
		t.Fatal("expected combined contract error to reject recovery")
	}
	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != original {
		t.Fatal("failed recovery did not restore original manifest bytes")
	}
}

func TestMissingSemanticFindingsRecoveryIsDeterministic(t *testing.T) {
	t.Parallel()
	var digest string
	for iteration := 0; iteration < 2; iteration++ {
		task := newCollectTask(t, "run-collect-missing-findings-deterministic")
		if err := os.MkdirAll(task.WriteRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(task.WriteRoot, "overview.md"), []byte("# Bank overview\n\nObserved README.md entrypoint.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		manifest := missingFindingsFixture(t, task)
		if err := os.WriteFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
		cause := artifactquality.ValidateCollectManifestInRoot(task.WriteRoot)
		report, err := recoverCollectManifestMissingFindings(task, cause)
		if err != nil {
			t.Fatal(err)
		}
		if iteration == 0 {
			digest = report.AfterDigest
		} else if report.AfterDigest != digest {
			t.Fatalf("after digest = %s, want %s", report.AfterDigest, digest)
		}
	}
}

func TestRunHeadlessProviderCanonicalizesProvenanceKindAliasesWithoutProviderRepair(t *testing.T) {
	t.Parallel()
	task := newCollectTask(t, "run-collect-provenance-kind-aliases")
	repoRoot := prepareProvenanceKindRecoveryTask(t, &task)
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) { diagnostics = append(diagnostics, event) }
	manifest := provenanceKindAliasFixture(t, task, "shard-pack-manifest.json")
	if err := artifactquality.ValidateCollectManifestBytes([]byte(manifest)); err == nil {
		t.Fatal("raw schema validation unexpectedly accepted lexical provenance.kind aliases")
	}
	script := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + ` <<'EOF'
# Bank architecture overview

README.md identifies the service entrypoint.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + manifest + `
EOF
`
	runner := &manifestRepairSequenceAdapter{testAdapter: testAdapter{
		command:  writeEngineScript(t, script),
		recovery: RecoveryPolicy{AcceptValidArtifactsAfterStop: true, RepairCollectManifestOnce: true},
	}}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("run provider: %v", err)
	}
	if runner.manifestRepairCalls != 0 {
		t.Fatalf("provider manifest repair calls = %d, want 0", runner.manifestRepairCalls)
	}
	if !hasDiagnosticField(diagnostics, "collect manifest provenance kind recovery completed", "recovery_mode", collectManifestProvenanceKindRecoveryMode) {
		t.Fatalf("missing recovery diagnostic: %#v", diagnostics)
	}
	if !hasRuntimeWarning(result.Execution.Warnings, "canonicalized 3 lexical provenance.kind aliases") {
		t.Fatalf("missing recovery warning: %#v", result.Execution.Warnings)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if err := artifactquality.ValidateCollectManifestInRootWithRepoRoots(task.WriteRoot, map[string]string{"bank-of-anthos": repoRoot}); err != nil {
		t.Fatalf("completed manifest is invalid: %v", err)
	}
	golden := provenanceKindAliasFixture(t, task, "shard-pack-manifest.golden.json")
	if string(raw) != golden+"\n" {
		t.Fatalf("recovered manifest differs from golden\nwant:\n%s\ngot:\n%s", golden, raw)
	}
	recovery, ok := result.Diagnostics[collectManifestProvenanceKindRecoveryMode].(map[string]any)
	if !ok || recovery["replacement_count"] != 3 || recovery["before_digest"] == "" ||
		recovery["after_digest"] == "" || recovery["before_digest"] == recovery["after_digest"] {
		t.Fatalf("invalid recovery evidence: %#v", result.Diagnostics)
	}
}

func TestProvenanceKindRecoveryRejectsArbitraryAliasesWithoutMutation(t *testing.T) {
	t.Parallel()
	task := newCollectTask(t, "run-collect-provenance-kind-arbitrary")
	prepareProvenanceKindRecoveryTask(t, &task)
	original := strings.Replace(provenanceKindAliasFixture(t, task, "shard-pack-manifest.json"), `"kind": "observed"`, `"kind": "measured"`, 1)
	manifestPath := filepath.Join(task.WriteRoot, ShardPackManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := recoverCollectManifestProvenanceKindAliases(task); err == nil {
		t.Fatal("expected arbitrary provenance kind to reject recovery")
	}
	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != original {
		t.Fatal("rejected recovery mutated original manifest")
	}

	for _, nonExact := range []string{"Observed", " observed"} {
		candidate := strings.Replace(provenanceKindAliasFixture(t, task, "shard-pack-manifest.json"), `"kind": "observed"`, `"kind": "`+nonExact+`"`, 1)
		if err := os.WriteFile(manifestPath, []byte(candidate), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := recoverCollectManifestProvenanceKindAliases(task); err == nil {
			t.Fatalf("expected non-exact alias %q to reject recovery", nonExact)
		}
		after, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != candidate {
			t.Fatalf("rejected non-exact alias %q mutated original manifest", nonExact)
		}
	}
}

func TestProvenanceKindRecoveryRollsBackWhenEvidenceValidationFails(t *testing.T) {
	t.Parallel()
	task := newCollectTask(t, "run-collect-provenance-kind-rollback")
	prepareProvenanceKindRecoveryTask(t, &task)
	original := strings.ReplaceAll(provenanceKindAliasFixture(t, task, "shard-pack-manifest.json"), `"path": "README.md"`, `"path": "MISSING.md"`)
	manifestPath := filepath.Join(task.WriteRoot, ShardPackManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := recoverCollectManifestProvenanceKindAliases(task); err == nil {
		t.Fatal("expected missing evidence path to reject recovery")
	}
	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != original {
		t.Fatal("failed recovery did not restore original manifest bytes")
	}
}

func TestWriteCollectManifestAtomicCleansTemporaryFileAfterReplaceFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	target := filepath.Join(root, ShardPackManifestFileName)
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeCollectManifestAtomic(target, []byte("{}\n"), 0o644); err == nil {
		t.Fatal("expected replace failure when target is a directory")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != ShardPackManifestFileName || !entries[0].IsDir() {
		t.Fatalf("atomic write left partial files: %#v", entries)
	}
}

func prepareProvenanceKindRecoveryTask(t *testing.T, task *acpruntime.Task) string {
	t.Helper()
	if err := os.MkdirAll(task.WriteRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Join(task.Workspace, "repos", "bank-of-anthos")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Bank of Anthos\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "overview.md"), []byte("# Bank architecture overview\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	task.ReadContextRoots = []string{repoRoot}
	task.RepoScopes = []string{"bank-of-anthos"}
	task.PathScopes = []string{"README.md"}
	return repoRoot
}

func provenanceKindAliasFixture(t *testing.T, task acpruntime.Task, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "fixtures", "scenarios", "collect-manifest-provenance-kind-aliases", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	value := strings.ReplaceAll(string(raw), "__RUN_ID__", task.RunID)
	value = strings.ReplaceAll(value, "__ARTIFACT_ROOT__", task.ArtifactRoot)
	return strings.TrimSuffix(value, "\n")
}

func missingFindingsFixture(t *testing.T, task acpruntime.Task) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "fixtures", "scenarios", "collect-manifest-missing-findings", "shard-pack-manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	value := strings.ReplaceAll(string(raw), "__RUN_ID__", task.RunID)
	return strings.ReplaceAll(value, "__ARTIFACT_ROOT__", task.ArtifactRoot)
}

func assertOnlyFindingsInserted(t *testing.T, before []byte, after []byte) {
	t.Helper()
	var beforeValue map[string]any
	var afterValue map[string]any
	if err := json.Unmarshal(before, &beforeValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(after, &afterValue); err != nil {
		t.Fatal(err)
	}
	afterSemantic := afterValue["semantic"].(map[string]any)
	findings, ok := afterSemantic["findings"].([]any)
	if !ok || len(findings) != 0 {
		t.Fatalf("semantic.findings = %#v, want []", afterSemantic["findings"])
	}
	delete(afterSemantic, "findings")
	beforeJSON, _ := json.Marshal(beforeValue)
	afterJSON, _ := json.Marshal(afterValue)
	if string(beforeJSON) != string(afterJSON) {
		t.Fatalf("recovery changed existing manifest values\nbefore=%s\nafter=%s", beforeJSON, afterJSON)
	}
}
