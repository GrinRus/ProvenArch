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

func TestValidateRequiredManifestRejectsConstitutionBootstrapDraftContent(t *testing.T) {
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
	bootstrapDraft := `# Constitution

## Scope
- Run: run-1
- Step: init.step0.constitution
- Repository scope: posthog
- Path scopes: .

## Summary
- Draft surface initialized for the scoped repository analysis.
- Final content must stay tied to collected shard evidence and validator output.
`
	if err := os.WriteFile(filepath.Join(draftRoot, "charter-overview.md"), []byte(bootstrapDraft), 0o644); err != nil {
		t.Fatalf("write charter overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "baseline-subagents.yaml"), []byte("agents: []\n"), 0o644); err != nil {
		t.Fatalf("write baseline subagents: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {"path": "charter-overview.md", "canonical_path": "charter/overview.md", "kind": "charter", "title": "Constitution"},
    {"path": "baseline-subagents.yaml", "canonical_path": "skills/subagents.yaml", "kind": "bundle", "title": "Baseline Subagents"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ConstitutionManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step0.constitution", "constitution", []string{ConstitutionManifestFile})
	if err == nil {
		t.Fatalf("expected bootstrap constitution draft content to be rejected")
	}
	if !strings.Contains(err.Error(), "bootstrap-only placeholder draft content") {
		t.Fatalf("expected bootstrap placeholder error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsConstitutionDraftOutsideAllowedPublishSurface(t *testing.T) {
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
		"charter-overview.md":     "# Constitution\n\n## Scope\n- Repo evidence: `README.md`.\n",
		"baseline-subagents.yaml": "agents: []\n",
		"extra.md":                "# Extra\n",
	} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(content), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {"path": "charter-overview.md", "canonical_path": "charter/overview.md"},
    {"path": "baseline-subagents.yaml", "canonical_path": "skills/subagents.yaml"},
    {"path": "extra.md", "canonical_path": "reports/as-is/extra.md"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ConstitutionManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step0.constitution", "constitution", []string{ConstitutionManifestFile})
	if err == nil {
		t.Fatalf("expected invalid constitution publish surface to be rejected")
	}
	if !strings.Contains(err.Error(), "outside the allowed constitution publish surface") {
		t.Fatalf("expected constitution publish surface error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsObservedContractInvalidStep0Shapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		fixture   string
		wantError string
	}{
		{
			name:      "bank single schema_version",
			fixture:   "contract-rejection/qwen_step0_bank_single_contract_invalid_constitution_draft.json",
			wantError: `unknown field "schema_version"`,
		},
		{
			name:      "openedx version string",
			fixture:   "contract-rejection/qwen_step0_openedx_multi_version_string_constitution_draft.json",
			wantError: "cannot unmarshal string into Go struct field Manifest.version of type int",
		},
		{
			name:      "openstack schema_version v1 string",
			fixture:   "contract-rejection/qwen_step0_openstack_multi_schema_version_constitution_draft.json",
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
				t.Fatalf("expected contract-invalid manifest to be rejected")
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

func TestValidateRequiredManifestAcceptsAsIsDraftUpdatedAtMetadata(t *testing.T) {
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
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte("# Evidence-backed draft\n"), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "summary": "Drafted required runtime artifacts for this step.",
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md", "kind": "report", "title": "System Overview"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md", "kind": "report", "title": "Coverage Summary"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md", "kind": "agent-output", "title": "Architect Summary"}
  ],
  "updated_at": "2026-06-19T00:12:40.516612+00:00"
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	loaded, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "refresh.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err != nil {
		t.Fatalf("expected updated_at metadata to validate: %v", err)
	}
	if loaded.UpdatedAt != "2026-06-19T00:12:40.516612+00:00" {
		t.Fatalf("unexpected updated_at: %q", loaded.UpdatedAt)
	}
}

func TestValidateRequiredManifestRejectsAsIsBootstrapDraftContent(t *testing.T) {
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
	bootstrapDraft := `# System Overview

## Scope
- Run: run-1
- Step: init.step2.asis_docs
- Repository scope: posthog
- Path scopes: .

## Summary
- Draft surface initialized for the scoped repository analysis.
- Final content must stay tied to collected shard evidence and validator output.
`
	for _, relPath := range []string{"overview.md", "summary.md", "architect-summary.md"} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(bootstrapDraft), 0o644); err != nil {
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
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md", "kind": "agent-output", "title": "Architect Summary"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err == nil {
		t.Fatalf("expected bootstrap as-is draft content to be rejected")
	}
	if !strings.Contains(err.Error(), "bootstrap-only placeholder draft content") {
		t.Fatalf("expected bootstrap placeholder error, got %v", err)
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

func TestValidateRequiredManifestRejectsOutputLogicalPathAlias(t *testing.T) {
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
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte("# Evidence-backed draft\n\nreports/as-is/overview.md\n"), 0o644); err != nil {
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
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md", "logical_path": "reports/as-is/overview.md"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err == nil {
		t.Fatalf("expected logical_path output alias to be rejected")
	}
	if !strings.Contains(err.Error(), `unknown field "logical_path"`) {
		t.Fatalf("expected strict unknown logical_path error, got %v", err)
	}
}

func TestValidateRequiredManifestAcceptsCanonicalProposalsDraft(t *testing.T) {
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
		"proposal.md":  "# Proposal\n",
		"changelog.md": "# Changelog\n",
	} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(content), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "summary": "Drafted proposals.",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "proposals/proposal-baseline/proposal.md", "kind": "proposal", "title": "Proposal"},
    {"path": "changelog.md", "canonical_path": "reports/changelog/run-1.md", "kind": "changelog", "title": "Changelog"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	loaded, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err != nil {
		t.Fatalf("expected canonical proposals draft manifest to validate: %v", err)
	}
	if got, want := len(loaded.Outputs), 2; got != want {
		t.Fatalf("unexpected output count: got=%d want=%d", got, want)
	}
}

func TestValidateRequiredManifestRejectsProposalsBootstrapDraftContent(t *testing.T) {
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
		"proposal.md": `# Runtime Recommendations

## Summary
- Current run evidence should be reviewed before promotion.
- Owner mappings and unresolved coverage gaps remain the first follow-up surfaces.

## Recommendation
- Promote only recommendations that cite collected shard manifests, validator findings, or final coverage output.
`,
		"changelog.md": `# Runtime Proposal Changelog

## Changes
- Runtime proposal surface initialized for this analysis run.
- Changes must remain traceable to collected evidence, findings, or coverage gaps before promotion.

## Notes
- Promote only after artifact validation succeeds.
`,
	} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(content), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "proposals/runtime-recommendations.md", "kind": "proposal", "title": "Runtime Recommendations"},
    {"path": "changelog.md", "canonical_path": "reports/changelog/runtime-proposals.md", "kind": "changelog", "title": "Runtime Proposal Changelog"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err == nil {
		t.Fatalf("expected bootstrap proposals draft content to be rejected")
	}
	if !strings.Contains(err.Error(), "bootstrap-only placeholder draft content") {
		t.Fatalf("expected bootstrap placeholder error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsProposalsCopiedBootstrapSummary(t *testing.T) {
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
	copiedSummary := `# Runtime Recommendations

## Decision / Recommended Operator Action
Use staged evidence from final-run-index.json and cite.posthog.services.runtime-compose before accepting proposals.

## Evidence Used
- "summary": "Drafted required runtime artifacts for this step."
- cite.posthog.services.runtime-compose
`
	for _, relPath := range []string{"proposal.md", "changelog.md"} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(copiedSummary), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "summary": "Drafted required runtime artifacts for this step.",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "proposals/runtime-recommendations.md", "kind": "proposal", "title": "Runtime Recommendations"},
    {"path": "changelog.md", "canonical_path": "reports/changelog/runtime-proposals.md", "kind": "changelog", "title": "Runtime Proposal Changelog"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "refresh.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err == nil {
		t.Fatalf("expected copied bootstrap summary in proposals markdown to be rejected")
	}
	if !strings.Contains(err.Error(), "bootstrap-only placeholder draft content") {
		t.Fatalf("expected bootstrap placeholder error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsAsIsRecoveryNarration(t *testing.T) {
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
	recoveryNarration := `# Architect Summary

## Decision-Ready Status

The required as-is draft outputs are present and freshly rewritten against bounded staged evidence. The manifest target remains runtime/step2_as_is/asis-draft-manifest.json.

## Evidence

- final-run-index.json
- cite.ftgo.accounting.contracts.authorize
`
	for _, relPath := range []string{"overview.md", "summary.md", "architect-summary.md"} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(recoveryNarration), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md", "kind": "report", "title": "System Overview"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md", "kind": "report", "title": "Coverage Summary"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md", "kind": "agent-output", "title": "Architect Summary"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "refresh.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err == nil {
		t.Fatalf("expected recovery narration in as-is markdown to be rejected")
	}
	if !strings.Contains(err.Error(), "bootstrap-only placeholder draft content") {
		t.Fatalf("expected bootstrap placeholder error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsProposalsRecoveryNarration(t *testing.T) {
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
	recoveryNarration := `# Runtime Proposal Recommendations

## Decision / recommended operator action

Promote this proposal surface only as an evidence-indexed follow-up queue. The enrichment read the current draft manifest for contract fields and used bounded staged evidence under the runtime roots.

## Evidence used

- final-run-index.json
- cite.ftgo.accounting.contracts.authorize
`
	for _, relPath := range []string{"proposal.md", "changelog.md"} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(recoveryNarration), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "proposals/runtime-recommendations.md", "kind": "proposal", "title": "Runtime Recommendations"},
    {"path": "changelog.md", "canonical_path": "reports/changelog/runtime-proposals.md", "kind": "changelog", "title": "Runtime Proposal Changelog"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "refresh.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err == nil {
		t.Fatalf("expected recovery narration in proposals markdown to be rejected")
	}
	if !strings.Contains(err.Error(), "bootstrap-only placeholder draft content") {
		t.Fatalf("expected bootstrap placeholder error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsGenericPlaceholderReplacementNarration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		stepID       string
		contract     string
		manifest     string
		files        map[string]string
		manifestFile string
	}{
		{
			name:         "as-is placeholder content",
			stepID:       "refresh.step2.asis_docs",
			contract:     "as_is",
			manifestFile: AsIsManifestFile,
			files: map[string]string{
				"overview.md":          "# Overview\n\nEvidence: reports/as-is/overview.md\n",
				"summary.md":           "# Coverage Summary\n\nThe evidence set is readable enough to replace placeholder content with an operator-facing coverage statement.\n\nEvidence: final-run-index.json\n",
				"architect-summary.md": "# Architect Summary\n\nEvidence: cite.ftgo.accounting.contracts.authorize\n",
			},
			manifest: `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md", "kind": "report", "title": "System Overview"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md", "kind": "report", "title": "Coverage Summary"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md", "kind": "agent-output", "title": "Architect Summary"}
  ]
}`,
		},
		{
			name:         "proposal placeholder content",
			stepID:       "refresh.step4.proposals",
			contract:     "proposals",
			manifestFile: ProposalsManifestFile,
			files: map[string]string{
				"proposal.md":  "# Runtime Proposal Recommendations\n\nEvidence: final-run-index.json\n",
				"changelog.md": "# Runtime Proposal Changelog\n\n- Replaced placeholder proposal content with operator-facing runtime recommendations for ftgo-application.\n\nEvidence: cite.ftgo.accounting.contracts.authorize\n",
			},
			manifest: `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "proposals/runtime-recommendations.md", "kind": "proposal", "title": "Runtime Recommendations"},
    {"path": "changelog.md", "canonical_path": "reports/changelog/runtime-proposals.md", "kind": "changelog", "title": "Runtime Proposal Changelog"}
  ]
}`,
		},
		{
			name:         "constitution placeholder text",
			stepID:       "init.step0.constitution",
			contract:     "constitution",
			manifestFile: ConstitutionManifestFile,
			files: map[string]string{
				"charter-overview.md":     "# Constitution\n\nValidation failures caused by placeholder draft text are resolved by replacing placeholders with concrete scope, evidence, and operating principles.\n\nEvidence: README.md\n",
				"baseline-subagents.yaml": "version: 1\n",
			},
			manifest: `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {"path": "charter-overview.md", "canonical_path": "charter/overview.md", "kind": "charter", "title": "Constitution"},
    {"path": "baseline-subagents.yaml", "canonical_path": "skills/subagents.yaml", "kind": "bundle", "title": "Baseline Subagents"}
  ]
}`,
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
			for relPath, content := range tc.files {
				if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte(content), 0o644); err != nil {
					t.Fatalf("write draft file %s: %v", relPath, err)
				}
			}
			if err := os.WriteFile(filepath.Join(writeRoot, tc.manifestFile), []byte(tc.manifest), 0o644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}

			_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", tc.stepID, tc.contract, []string{tc.manifestFile})
			if err == nil {
				t.Fatalf("expected generic placeholder replacement narration to be rejected")
			}
			if !strings.Contains(err.Error(), "bootstrap-only placeholder draft content") {
				t.Fatalf("expected bootstrap placeholder error, got %v", err)
			}
		})
	}
}

func TestValidateRequiredManifestRejectsForeignRunIDReferences(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(draftRoot, "overview.md"), []byte(`# As-Is Overview

Current coverage uses reports/as-is/overview.md and final-run-index.json.
Run indexes reviewed: run_20260620_181032_001/staging/final/final-run-index.json.
`), 0o644); err != nil {
		t.Fatalf("write overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "summary.md"), []byte("# Coverage\n\nEvidence: final-run-index.json\n"), 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "architect-summary.md"), []byte("# Architect Summary\n\nEvidence: cite.ftgo.accounting.contracts.authorize\n"), 0o644); err != nil {
		t.Fatalf("write architect summary: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run_20260620_184900_001",
  "step_id": "refresh.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md", "kind": "report", "title": "System Overview"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md", "kind": "report", "title": "Coverage Summary"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md", "kind": "agent-output", "title": "Architect Summary"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run_20260620_184900_001", "refresh.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err == nil {
		t.Fatalf("expected foreign run id reference to be rejected")
	}
	if !strings.Contains(err.Error(), `references foreign taskrun "run_20260620_181032_001"`) {
		t.Fatalf("expected foreign taskrun error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsGenericShardGapWording(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(draftRoot, "proposal.md"), []byte(`# Runtime Recommendations

Evidence: final-run-index.json and cite.ftgo.accounting.contracts.authorize.
Failed shards require rerun or manual review before implementation.
`), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "changelog.md"), []byte(`# Runtime Proposal Changelog

Evidence: reports/changelog/runtime-proposals.md.
Failed or incomplete shards remain coverage gaps until rerun or manually reviewed.
`), 0o644); err != nil {
		t.Fatalf("write changelog: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run_20260620_184900_001",
  "step_id": "refresh.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "proposals/runtime-recommendations.md", "kind": "proposal", "title": "Runtime Recommendations"},
    {"path": "changelog.md", "canonical_path": "reports/changelog/runtime-proposals.md", "kind": "changelog", "title": "Runtime Proposal Changelog"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run_20260620_184900_001", "refresh.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err == nil {
		t.Fatalf("expected generic shard-gap wording to be rejected")
	}
	if !strings.Contains(err.Error(), "generic conditional shard-gap wording") {
		t.Fatalf("expected generic shard-gap wording error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsStaleFinalIndexAvailabilityClaim(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(draftRoot, "overview.md"), []byte(`# As-Is Overview

Evidence: reports/as-is/overview.md and staged shard manifests.
Current-run index surfaces checked:
- No current-run final-run-index.json or citation-index.json was present in the allowed current-run locations.
`), 0o644); err != nil {
		t.Fatalf("write overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "summary.md"), []byte("# Coverage\n\nEvidence: reports/coverage/summary.md\n"), 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "architect-summary.md"), []byte("# Architect Summary\n\nEvidence: cite.ftgo.accounting.contracts.authorize\n"), 0o644); err != nil {
		t.Fatalf("write architect summary: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run_20260620_184900_001",
  "step_id": "refresh.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "outputs": [
    {"path": "overview.md", "canonical_path": "reports/as-is/overview.md", "kind": "report", "title": "System Overview"},
    {"path": "summary.md", "canonical_path": "reports/coverage/summary.md", "kind": "report", "title": "Coverage Summary"},
    {"path": "architect-summary.md", "canonical_path": "reports/agent-outputs/architect/summary.md", "kind": "agent-output", "title": "Architect Summary"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, AsIsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run_20260620_184900_001", "refresh.step2.asis_docs", "as_is", []string{AsIsManifestFile})
	if err == nil {
		t.Fatalf("expected stale final-index availability claim to be rejected")
	}
	if !strings.Contains(err.Error(), "claims current-run final/citation indexes are unavailable") {
		t.Fatalf("expected stale final-index availability error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsRawStructuredEvidenceDump(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(draftRoot, "proposal.md"), []byte(`# Runtime Recommendations

Evidence: reports/findings/findings.md and final-run-index.json.
Citation/index surface:
- {'id': 'cite.ftgo.accounting.authorize.contract', 'repo': 'ftgo-application', 'path': 'ftgo-accounting-service-contracts/src/main/resources/contracts/Authorize.groovy', 'claim_ids': ['claim.accounting.contract']}
`), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "changelog.md"), []byte("# Runtime Proposal Changelog\n\nEvidence: reports/changelog/runtime-proposals.md\n"), 0o644); err != nil {
		t.Fatalf("write changelog: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run_20260620_184900_001",
  "step_id": "refresh.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "proposals/runtime-recommendations.md", "kind": "proposal", "title": "Runtime Recommendations"},
    {"path": "changelog.md", "canonical_path": "reports/changelog/runtime-proposals.md", "kind": "changelog", "title": "Runtime Proposal Changelog"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run_20260620_184900_001", "refresh.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err == nil {
		t.Fatalf("expected raw structured evidence dump to be rejected")
	}
	if !strings.Contains(err.Error(), "raw structured evidence dumps") {
		t.Fatalf("expected raw structured evidence dump error, got %v", err)
	}
}

func TestValidateRequiredManifestAllowsReadableIndexSummary(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(draftRoot, "proposal.md"), []byte(`# Runtime Recommendations

Evidence: current-run final-run-index.json present with 51 canonical documents and citation-index.json present with 103 citations.
Selected citations: cite.ftgo.accounting.authorize.contract -> ftgo-accounting-service-contracts/src/main/resources/contracts/Authorize.groovy.
There are no failed or incomplete shards in the current typed shard summary.
`), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "changelog.md"), []byte("# Runtime Proposal Changelog\n\nEvidence: reports/changelog/runtime-proposals.md and cite.ftgo.accounting.authorize.contract.\n"), 0o644); err != nil {
		t.Fatalf("write changelog: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run_20260620_184900_001",
  "step_id": "refresh.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "proposals/runtime-recommendations.md", "kind": "proposal", "title": "Runtime Recommendations"},
    {"path": "changelog.md", "canonical_path": "reports/changelog/runtime-proposals.md", "kind": "changelog", "title": "Runtime Proposal Changelog"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run_20260620_184900_001", "refresh.step4.proposals", "proposals", []string{ProposalsManifestFile}); err != nil {
		t.Fatalf("expected readable index summary to validate: %v", err)
	}
}

func TestValidateRequiredManifestRejectsObservedContractInvalidProposalsEnvelope(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), readRuntimeFixture(t, "contract-rejection/claude_step4_bank_contract_invalid_proposals_draft.json"), 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err == nil {
		t.Fatalf("expected contract-invalid proposals envelope to be rejected")
	}
	if !strings.Contains(err.Error(), `unknown field "pipeline"`) {
		t.Fatalf("expected strict parser unknown field error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsProposalsDraftOutsideAllowedPublishSurface(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(draftRoot, "proposal.md"), []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write draft file: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {"path": "proposal.md", "canonical_path": "reports/as-is/proposal.md", "kind": "proposal", "title": "Proposal"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err == nil {
		t.Fatalf("expected invalid proposals publish surface to be rejected")
	}
	if !strings.Contains(err.Error(), "outside the allowed proposals publish surface") {
		t.Fatalf("expected proposals publish surface error, got %v", err)
	}
}

func TestValidateRequiredManifestRejectsDuplicateProposalsCanonicalPath(t *testing.T) {
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
	for _, relPath := range []string{"proposal-a.md", "proposal-b.md"} {
		if err := os.WriteFile(filepath.Join(draftRoot, relPath), []byte("# Proposal\n"), 0o644); err != nil {
			t.Fatalf("write draft file %s: %v", relPath, err)
		}
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {"path": "proposal-a.md", "canonical_path": "proposals/proposal-baseline/proposal.md", "kind": "proposal", "title": "Proposal A"},
    {"path": "proposal-b.md", "canonical_path": "proposals/proposal-baseline/proposal.md", "kind": "proposal", "title": "Proposal B"}
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, ProposalsManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := ValidateRequiredManifest(writeRoot, draftRoot, "run-1", "init.step4.proposals", "proposals", []string{ProposalsManifestFile})
	if err == nil {
		t.Fatalf("expected duplicate proposals canonical path to be rejected")
	}
	if !strings.Contains(err.Error(), "must be unique") {
		t.Fatalf("expected duplicate canonical path error, got %v", err)
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
