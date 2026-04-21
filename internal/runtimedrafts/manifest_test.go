package runtimedrafts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			fixture:   "qwen_step0_bank_single_legacy_constitution_draft.json",
			wantError: "runtime draft manifest version must be 1",
		},
		{
			name:      "openedx version string",
			fixture:   "qwen_step0_openedx_multi_version_string_constitution_draft.json",
			wantError: "cannot unmarshal string into Go struct field Manifest.version of type int",
		},
		{
			name:      "openstack schema_version v1 string",
			fixture:   "qwen_step0_openstack_multi_schema_version_constitution_draft.json",
			wantError: "runtime draft manifest version must be 1",
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
