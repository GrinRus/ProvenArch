package runtimedrafts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/runtime/compatibilityregistry"
)

func readRuntimeFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "runtime", "testdata", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read runtime fixture %s: %v", name, err)
	}
	return raw
}

func TestValidateRequiredManifestAcceptsCanonicalConstitutionDraft(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "charter-overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatalf("write charter overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "baseline-subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write baseline subagents: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "charter-overview.md",
      "canonical_path": "charter/overview.md",
      "kind": "charter",
      "title": "Constitution"
    },
    {
      "path": "baseline-subagents.yaml",
      "canonical_path": "skills/subagents.yaml",
      "kind": "bundle",
      "title": "Baseline Subagents"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ConstitutionManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	loaded, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step0.constitution", "constitution", []string{ConstitutionManifestFile})
	if err != nil {
		t.Fatalf("expected canonical constitution draft manifest to validate: %v", err)
	}
	if loaded.Version != 1 || len(loaded.Outputs) != 2 {
		t.Fatalf("unexpected loaded manifest: %#v", loaded)
	}
}

func TestValidateRequiredManifestRejectsObservedLegacyStep0Shapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		fixture   string
		wantError string
	}{
		{
			name:      "bank single legacy schema_version",
			fixture:   "legacy-rejection/qwen_step0_bank_single_legacy_constitution_draft.json",
			wantError: `unknown field "schema_version"`,
		},
		{
			name:      "openedx version string",
			fixture:   "legacy-rejection/qwen_step0_openedx_multi_version_string_constitution_draft.json",
			wantError: "cannot unmarshal string into Go struct field Manifest.version of type int",
		},
		{
			name:      "openstack schema_version v1 string",
			fixture:   "legacy-rejection/qwen_step0_openstack_multi_schema_version_constitution_draft.json",
			wantError: `unknown field "schema_version"`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			writeRoot := filepath.Join(tempDir, "write-root")
			draftRoot := filepath.Join(tempDir, "draft-root")
			if err := os.MkdirAll(writeRoot, 0o755); err != nil {
				t.Fatalf("mkdir write root: %v", err)
			}
			if err := os.MkdirAll(draftRoot, 0o755); err != nil {
				t.Fatalf("mkdir draft root: %v", err)
			}
			if err := os.WriteFile(filepath.Join(draftRoot, "charter-overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
				t.Fatalf("write charter overview: %v", err)
			}
			if err := os.WriteFile(filepath.Join(draftRoot, "baseline-subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
				t.Fatalf("write baseline subagents: %v", err)
			}
			if err := os.WriteFile(filepath.Join(writeRoot, ConstitutionManifestFile), readRuntimeFixture(t, tc.fixture), 0o644); err != nil {
				t.Fatalf("write fixture manifest: %v", err)
			}

			_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step0.constitution", "constitution", []string{ConstitutionManifestFile})
			if err == nil {
				t.Fatalf("expected legacy manifest to be rejected")
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected %q in error, got %v", tc.wantError, err)
			}
		})
	}
}

func TestValidateRequiredManifestAcceptsCanonicalAsIsDraft(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	for relPath, content := range map[string]string{
		"overview.md":          "# Overview\n",
		"summary.md":           "# Coverage\n",
		"architect-summary.md": "# Architect\n",
		"payments-overview.md": "# Payments\n",
	} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(content), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md", "kind": "report", "title": "System Overview"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md", "kind": "report", "title": "Coverage Summary"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md", "kind": "agent-output", "title": "Architect Summary"},
    {"path": "payments-overview.md", "canonical_path": "reports/as-is/payments/overview.md", "kind": "report", "title": "Payments Overview"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	loaded, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err != nil {
		t.Fatalf("expected canonical as-is draft manifest to validate: %v", err)
	}
	if got, want := len(loaded.Outputs), 4; got != want {
		t.Fatalf("unexpected output count: got=%d want=%d", got, want)
	}
}

func TestValidateRequiredManifestRejectsAsIsDraftUnknownTopLevelField(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	for _, relPath := range []string{"overview.md", "summary.md", "architect-summary.md"} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte("# draft\n"), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "repo_scopes": ["payments"],
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err == nil {
		t.Fatalf("expected strict parser to reject unknown top-level field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsAsIsDraftNullStepContract(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	for _, relPath := range []string{"overview.md", "summary.md", "architect-summary.md"} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte("# draft\n"), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step2.asis_docs",
  "step_contract": null,
  "agent_role": "architect",
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err == nil {
		t.Fatalf("expected null step_contract to be rejected")
	}
	if !strings.Contains(err.Error(), `runtime draft manifest step_contract must equal "as_is"`) {
		t.Fatalf("expected strict step_contract error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsAsIsDraftLegacyOutputSurface(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	for _, relPath := range []string{"overview.md", "summary.md", "architect-summary.md", "open-questions.md"} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte("# draft\n"), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md"},
    {"path": "open-questions.md", "canonical_path": "reports/coverage/open-questions.md"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err == nil {
		t.Fatalf("expected invalid as-is publish surface to be rejected")
	}
	if !strings.Contains(err.Error(), "outside the allowed as-is publish surface") {
		t.Fatalf("expected as-is publish surface error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsMissingReferencedDraftFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "charter-overview.md",
      "canonical_path": "charter/overview.md",
      "kind": "charter",
      "title": "Constitution"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ConstitutionManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step0.constitution", "constitution", []string{ConstitutionManifestFile})
	if err == nil {
		t.Fatalf("expected missing draft file validation error")
	}
	if !strings.Contains(err.Error(), "referenced draft file") {
		t.Fatalf("expected referenced draft file error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsCanonicalPathOnlyDraftFilesWithoutExplicitRepair(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(draftRoot, "charter"), 0o755); err != nil {
		t.Fatalf("mkdir draft charter dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(draftRoot, "skills"), 0o755); err != nil {
		t.Fatalf("mkdir draft skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "charter", "overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatalf("write nested charter overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "skills", "subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write nested skills bundle: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "charter-overview.md",
      "canonical_path": "charter/overview.md",
      "kind": "charter",
      "title": "Constitution"
    },
    {
      "path": "baseline-subagents.yaml",
      "canonical_path": "skills/subagents.yaml",
      "kind": "bundle",
      "title": "Baseline Subagents"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ConstitutionManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step0.constitution", "constitution", []string{ConstitutionManifestFile})
	if err == nil {
		t.Fatalf("expected canonical-path-only draft files to fail read-only validation")
	}
	if !strings.Contains(err.Error(), "referenced draft file") {
		t.Fatalf("expected referenced draft file error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(draftRoot, "charter-overview.md")); !os.IsNotExist(statErr) {
		t.Fatalf("expected read-only validation to avoid creating repaired draft file, stat err=%v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(draftRoot, "baseline-subagents.yaml")); !os.IsNotExist(statErr) {
		t.Fatalf("expected read-only validation to avoid creating repaired bundle draft file, stat err=%v", statErr)
	}
}

func TestReconcileOutputsAtDraftRootCopiesCanonicalPathDraftFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(filepath.Join(draftRoot, "charter"), 0o755); err != nil {
		t.Fatalf("mkdir draft charter dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(draftRoot, "skills"), 0o755); err != nil {
		t.Fatalf("mkdir draft skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "charter", "overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatalf("write nested charter overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "skills", "subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write nested skills bundle: %v", err)
	}

	manifest := Manifest{
		Version:      1,
		RunID:        "run-1",
		StepID:       "init.step0.constitution",
		StepContract: "constitution",
		AgentRole:    "architect",
		Outputs: []Output{
			{Path: "charter-overview.md", CanonicalPath: "charter/overview.md", Kind: "charter", Title: "Constitution"},
			{Path: "baseline-subagents.yaml", CanonicalPath: "skills/subagents.yaml", Kind: "bundle", Title: "Baseline Subagents"},
		},
	}

	changed, err := ReconcileOutputsAtDraftRoot(draftRoot, manifest)
	if err != nil {
		t.Fatalf("reconcile outputs at draft root: %v", err)
	}
	if !changed {
		t.Fatalf("expected reconciliation to materialize draft files at manifest paths")
	}
	if _, err := os.Stat(filepath.Join(draftRoot, "charter-overview.md")); err != nil {
		t.Fatalf("expected reconciled charter-overview.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(draftRoot, "baseline-subagents.yaml")); err != nil {
		t.Fatalf("expected reconciled baseline-subagents.yaml: %v", err)
	}
}

func TestReconcileOutputsAtDraftRootErrorsIncludeCompatibilityRuleID(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(draftRoot, "charter"), 0o755); err != nil {
		t.Fatalf("mkdir canonical fallback dir: %v", err)
	}
	manifest := Manifest{
		Version:      1,
		RunID:        "run-1",
		StepID:       "init.step0.constitution",
		StepContract: "constitution",
		AgentRole:    "architect",
		Outputs: []Output{
			{Path: "charter-overview.md", CanonicalPath: "charter", Kind: "charter", Title: "Constitution"},
		},
	}

	_, err := ReconcileOutputsAtDraftRoot(draftRoot, manifest)
	if err == nil {
		t.Fatalf("expected reconcile error")
	}
	if !strings.Contains(err.Error(), compatibilityregistry.RuleDraftRootReconcileExistingOutputs) {
		t.Fatalf("expected repair rule id in reconcile error, got %v", err)
	}
}
