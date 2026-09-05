package providercommon

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
)

func TestRunHeadlessProviderRecoversInlineArchitectureHomeWithoutProviderRepair(t *testing.T) {
	t.Parallel()
	task := newAsIsDraftTask(t, "run-inline-architecture-home")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) { diagnostics = append(diagnostics, event) }
	script := inlineArchitectureHomeScript(t, task)
	runner := &draftEnrichmentSequenceAdapter{testAdapter: testAdapter{
		command:  writeEngineScript(t, script),
		recovery: RecoveryPolicy{AcceptValidArtifactsAfterStop: true, RepairDraftArtifactsOnce: true, RepairDraftArtifactEnrichmentOnce: true},
	}}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("run provider: %v", err)
	}
	if runner.draftCalls != 0 {
		t.Fatalf("provider enrichment calls = %d, want 0", runner.draftCalls)
	}
	if !hasDiagnosticField(diagnostics, "Architecture Home inline headings recovered", "recovery_mode", architectureHomeInlineHeadingRecoveryMode) {
		t.Fatalf("missing deterministic recovery diagnostic: %#v", diagnostics)
	}
	recovery, ok := result.Diagnostics[architectureHomeInlineHeadingRecoveryMode].(map[string]any)
	if !ok || recovery["before_digest"] == "" || recovery["after_digest"] == "" || recovery["before_digest"] == recovery["after_digest"] || recovery["operator_review_required"] != true {
		t.Fatalf("invalid recovery audit: %#v", result.Diagnostics)
	}
	if err := ValidateDraftArtifacts(task); err != nil {
		t.Fatalf("recovered draft is invalid: %v", err)
	}
}

func TestArchitectureHomeInlineHeadingRecoveryRollsBackAfterFullValidationFailure(t *testing.T) {
	t.Parallel()
	task := newAsIsDraftTask(t, "run-inline-architecture-home-rollback")
	writeInlineArchitectureHomeArtifacts(t, task)
	overviewPath := filepath.Join(task.DraftFinalRoot, "overview.md")
	original, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatal(err)
	}
	cause := ValidateDraftArtifacts(task)
	if cause == nil {
		t.Fatal("expected inline headings to fail strict validation")
	}
	_, attempted, recoveryErr := recoverArchitectureHomeInlineHeadings(task, func() error {
		return context.DeadlineExceeded
	}, cause)
	if !attempted || recoveryErr == nil {
		t.Fatalf("attempted=%v err=%v, want attempted failure", attempted, recoveryErr)
	}
	after, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatal("failed recovery did not restore original Architecture Home bytes")
	}
}

func TestArchitectureHomeInlineHeadingRecoveryRejectsAdditionalContractError(t *testing.T) {
	t.Parallel()
	task := newAsIsDraftTask(t, "run-inline-architecture-home-multiple-errors")
	writeInlineArchitectureHomeArtifacts(t, task)
	overviewPath := filepath.Join(task.DraftFinalRoot, "overview.md")
	file, err := os.OpenFile(overviewPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("\nRuntime provider generated this manifest recap.\n"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	cause := ValidateDraftArtifacts(task)
	if cause == nil {
		t.Fatal("expected multiple contract errors")
	}
	_, attempted, recoveryErr := recoverArchitectureHomeInlineHeadings(task, func() error { return nil }, cause)
	if attempted || recoveryErr != nil {
		t.Fatalf("attempted=%v err=%v, want ineligible candidate", attempted, recoveryErr)
	}
}

func TestWriteArchitectureHomeAtomicCleansTempAfterReplaceFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	target := filepath.Join(root, "overview.md")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeArchitectureHomeAtomic(target, []byte("# Architecture Home\n"), 0o644); err == nil {
		t.Fatal("expected replace failure")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "overview.md" || !entries[0].IsDir() {
		t.Fatalf("atomic write left partial files: %#v", entries)
	}
}

func inlineArchitectureHomeScript(t *testing.T, task acpruntime.Task) string {
	t.Helper()
	fixturePath, err := filepath.Abs(filepath.Join("..", "..", "..", "fixtures", "scenarios", "architecture-home-inline-headings", "overview.md"))
	if err != nil {
		t.Fatal(err)
	}
	return "#!/usr/bin/env bash\nset -eu\nmkdir -p " + shellQuote(task.WriteRoot) + " " + shellQuote(task.DraftFinalRoot) + "\n" +
		"cp " + shellQuote(fixturePath) + " " + shellQuote(filepath.Join(task.DraftFinalRoot, "overview.md")) + "\n" +
		"printf '%s\\n' '# Summary' '' 'Current architecture evidence is summarized in reports/as-is/overview.md.' > " + shellQuote(filepath.Join(task.DraftFinalRoot, "summary.md")) + "\n" +
		"printf '%s\\n' '# Architect Summary' '' 'Start with reports/as-is/overview.md and README.md.' > " + shellQuote(filepath.Join(task.DraftFinalRoot, "architect-summary.md")) + "\n" +
		"cat > " + shellQuote(filepath.Join(task.WriteRoot, "asis-draft-manifest.json")) + " <<'EOF'\n" + steppolicy.RuntimeDraftManifestTaskSkeleton(task) + "\nEOF\n"
}

func writeInlineArchitectureHomeArtifacts(t *testing.T, task acpruntime.Task) {
	t.Helper()
	command := writeEngineScript(t, inlineArchitectureHomeScript(t, task))
	runner := testAdapter{command: command}
	spec, err := runner.CommandSpec(task)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCommandSpecWithTransition(context.Background(), task, spec, ActivityPolicy{}, "normal"); err != nil {
		t.Fatalf("write inline Architecture Home artifacts: %v", err)
	}
}
