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
