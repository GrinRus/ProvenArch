package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
)

func TestValidateRequiredRuntimeDraftArtifactsRejectsMissingReferencedFiles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		stepID       string
		stepContract string
		manifestFile string
		canonical    string
	}{
		{
			name:         "step0 constitution",
			stepID:       "init.step0.constitution",
			stepContract: "constitution",
			manifestFile: constitutionDraftManifestFile,
			canonical:    "charter/overview.md",
		},
		{
			name:         "step2 as-is",
			stepID:       "init.step2.asis_docs",
			stepContract: "as_is",
			manifestFile: asisDraftManifestFile,
			canonical:    "reports/as-is/overview.md",
		},
		{
			name:         "step4 proposals",
			stepID:       "init.step4.proposals",
			stepContract: "proposals",
			manifestFile: proposalsDraftManifestFile,
			canonical:    "proposals/proposal-baseline/proposal.md",
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
			manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "` + tc.stepID + `",
  "step_contract": "` + tc.stepContract + `",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "missing.md",
      "canonical_path": "` + tc.canonical + `",
      "kind": "report",
      "title": "Missing"
    }
  ]
}`
			if err := os.WriteFile(filepath.Join(writeRoot, tc.manifestFile), []byte(manifest), 0o644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}

			_, _, err := validateRequiredRuntimeDraftArtifacts(acpruntime.Task{
				StepID:         tc.stepID,
				WriteRoot:      writeRoot,
				DraftFinalRoot: draftRoot,
			})
			if err == nil {
				t.Fatalf("expected missing draft file validation error")
			}
			if !strings.Contains(err.Error(), "referenced draft file") {
				t.Fatalf("expected referenced draft file error, got %v", err)
			}
		})
	}
}

func TestValidateRequiredRuntimeDraftArtifactsRejectsManifestContractMismatch(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(draftRoot, "overview.md"), []byte("# Overview\n"), 0o644); err != nil {
		t.Fatalf("write draft overview: %v", err)
	}
	manifest := `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step2.asis_docs",
  "step_contract": "collect",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "overview.md",
      "canonical_path": "reports/as-is/overview.md",
      "kind": "report",
      "title": "Overview"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(writeRoot, asisDraftManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := validateRequiredRuntimeDraftArtifacts(acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step2.asis_docs",
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{asisDraftManifestFile},
	})
	if err == nil {
		t.Fatalf("expected manifest contract mismatch validation error")
	}
	if !strings.Contains(err.Error(), `step_id must equal "init.step2.asis_docs"`) {
		t.Fatalf("expected explicit step_id mismatch, got %v", err)
	}
}

func TestValidateRequiredRuntimeDraftArtifactsRejectsEmptyOutputs(t *testing.T) {
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
  "step_id": "init.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": []
}`
	if err := os.WriteFile(filepath.Join(writeRoot, proposalsDraftManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, _, err := validateRequiredRuntimeDraftArtifacts(acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step4.proposals",
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{proposalsDraftManifestFile},
	})
	if err == nil {
		t.Fatalf("expected empty outputs validation error")
	}
	if !strings.Contains(err.Error(), "outputs must not be empty") {
		t.Fatalf("expected explicit empty outputs error, got %v", err)
	}
}

func TestRunFailsWhenStep2ReturnsMalformedTaskResultAndInvalidDraftManifest(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(step2InvalidDraftManifestParseFailureRunner{}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected init pipeline to fail on invalid step2 draft manifest")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if !strings.Contains(info.Error, "required runtime draft artifacts invalid") {
		t.Fatalf("expected runtime draft artifact validation context, got %q", info.Error)
	}
	if !strings.Contains(info.Error, "runtime draft manifest version must be 1") {
		t.Fatalf("expected explicit draft manifest validation reason, got %q", info.Error)
	}
}

type step2InvalidDraftManifestParseFailureRunner struct{}

func (step2InvalidDraftManifestParseFailureRunner) Preflight(context.Context) error { return nil }

func (step2InvalidDraftManifestParseFailureRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if task.StepID != "init.step2.asis_docs" {
		return claudecode.FakeRunner{}.Run(ctx, task)
	}
	if err := os.MkdirAll(task.WriteRoot, 0o755); err != nil {
		return acpruntime.Result{}, err
	}
	legacyManifest := `{
  "schema_version": "legacy",
  "outputs": [
    {
      "path": "overview.md",
      "canonical_path": "reports/as-is/overview.md"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(task.WriteRoot, asisDraftManifestFile), []byte(legacyManifest), 0o644); err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{
		RawJSON: []byte(`{"meta":`),
		Stdout:  `{"meta":`,
	}, nil
}
