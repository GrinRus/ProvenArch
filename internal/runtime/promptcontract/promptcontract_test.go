package promptcontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestAdditivePromptPackSectionUsesWorkspaceOverrideWhenPresent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	packPath := filepath.Join(root, "skills", "prompt-packs")
	if err := os.MkdirAll(packPath, 0o755); err != nil {
		t.Fatalf("mkdir prompt pack path: %v", err)
	}
	custom := "Custom collect-context override.\n"
	if err := os.WriteFile(filepath.Join(packPath, "collect-context.md"), []byte(custom), 0o644); err != nil {
		t.Fatalf("write custom prompt pack: %v", err)
	}

	section, pack := AdditivePromptPackSection(acpruntime.Task{
		StepID:       "refresh.step1.collect",
		Workspace:    root,
		StartedAtUTC: time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC),
	})
	if pack.Source != "workspace" {
		t.Fatalf("expected workspace prompt pack source, got %q", pack.Source)
	}
	if !strings.Contains(section, "Custom collect-context override.") {
		t.Fatalf("expected custom prompt pack content in section, got %q", section)
	}
}

func TestAdditivePromptPackSectionFallsBackToSeededBaselineWithWarning(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	section, pack := AdditivePromptPackSection(acpruntime.Task{
		StepID:       "refresh.step3.findings",
		Workspace:    root,
		StartedAtUTC: time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC),
	})
	if pack.Source != "seeded-baseline" {
		t.Fatalf("expected seeded-baseline source, got %q", pack.Source)
	}
	if strings.TrimSpace(pack.Warning) == "" {
		t.Fatalf("expected fallback warning when workspace prompt pack is missing")
	}
	if !strings.Contains(section, "Prompt pack warning:") {
		t.Fatalf("expected warning banner in prompt section, got %q", section)
	}
	if !strings.Contains(section, "Findings Prompt Pack") {
		t.Fatalf("expected seeded findings prompt pack content in section, got %q", section)
	}
}

func TestSharedContractAndRetryGuardrailsExposeCoreInvariants(t *testing.T) {
	t.Parallel()

	contract := strings.Join(SharedTaskResultContractLines(), "\n")
	if !strings.Contains(contract, `meta required keys: "task_id", "step_id", "runtime", "started_at"`) {
		t.Fatalf("expected shared taskresult contract to include meta keys invariant")
	}
	if !strings.Contains(contract, `coverage.missing MUST use canonical terms only`) {
		t.Fatalf("expected shared taskresult contract to include canonical coverage invariant")
	}

	retry := strings.Join(SharedRetryGuardrailLines(), "\n")
	if !strings.Contains(retry, `Unknown changeset[].op values are forbidden`) {
		t.Fatalf("expected shared retry guardrails to include changeset op whitelist")
	}
	if !strings.Contains(retry, `For op="add_doc_artifact", the payload key MUST be "doc_artifact"; never use "artifact".`) {
		t.Fatalf("expected shared retry guardrails to include doc_artifact key invariant")
	}
}
