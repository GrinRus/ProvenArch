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

func TestAdditivePromptPackSectionIgnoresReferenceOnlySkillPromptFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	referencePath := filepath.Join(root, "skills", "findings", "prompts")
	if err := os.MkdirAll(referencePath, 0o755); err != nil {
		t.Fatalf("mkdir reference prompt path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(referencePath, "system.md"), []byte("Reference-only skill prompt should not be consumed.\n"), 0o644); err != nil {
		t.Fatalf("write reference-only skill prompt: %v", err)
	}

	section, pack := AdditivePromptPackSection(acpruntime.Task{
		StepID:       "refresh.step3.findings",
		Workspace:    root,
		StartedAtUTC: time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC),
	})
	if pack.Source != "seeded-baseline" {
		t.Fatalf("expected seeded-baseline source when prompt pack is absent, got %q", pack.Source)
	}
	if strings.Contains(section, "Reference-only skill prompt should not be consumed.") {
		t.Fatalf("reference-only skill prompt leaked into effective prompt section: %q", section)
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

func TestStepPolicyFindingsExplicitlyLimitsContextToStagedSurface(t *testing.T) {
	t.Parallel()

	policy := StepPolicy("refresh.step3.findings")
	if !strings.Contains(policy, "Prioritize staged final artifacts") {
		t.Fatalf("expected staged-final priority in findings policy, got %q", policy)
	}
	if !strings.Contains(policy, "do not perform broad repository rediscovery") {
		t.Fatalf("expected no broad rediscovery guardrail in findings policy, got %q", policy)
	}
	if !strings.Contains(policy, "Do not recursively crawl whole repositories") {
		t.Fatalf("expected no full recrawl guardrail in findings policy, got %q", policy)
	}
}

func TestDocFirstFilesystemPolicyFindingsMarksStagedArtifactsAsPrimaryEvidence(t *testing.T) {
	t.Parallel()

	policy := DocFirstFilesystemPolicy(acpruntime.Task{
		StepID:           "refresh.step3.findings",
		Workspace:        "/tmp/workspace",
		ArtifactRoot:     "reports/taskruns/run-1/validator",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/validator",
		ReadContextRoots: []string{"/tmp/workspace", "/tmp/workspace/reports/taskruns/run-1/staging/final"},
	})
	if !strings.Contains(policy, "Use staged/final artifacts as primary evidence") {
		t.Fatalf("expected staged/final primary evidence rule, got %q", policy)
	}
	if !strings.Contains(policy, "do not treat full repository recrawl as default behavior in validator step") {
		t.Fatalf("expected no recrawl-by-default rule, got %q", policy)
	}
}

func TestRetryHintsExposeCompactJsonObjective(t *testing.T) {
	t.Parallel()

	parseHints := strings.Join(ParseRepairHints("extract", errSample("invalid json")), "\n")
	if !strings.Contains(parseHints, "RETRY OBJECTIVE: return exactly one minimal TaskResult JSON object") {
		t.Fatalf("expected compact retry objective in parse hints, got %q", parseHints)
	}
	if !strings.Contains(parseHints, "Do NOT include tool logs, event arrays") {
		t.Fatalf("expected anti-chatter guardrail in parse hints, got %q", parseHints)
	}

	artifactHints := strings.Join(ArtifactRepairHints("manifest invalid"), "\n")
	if !strings.Contains(artifactHints, "RETRY OBJECTIVE: repair collect artifacts deterministically") {
		t.Fatalf("expected compact retry objective in artifact hints, got %q", artifactHints)
	}
}

func errSample(msg string) error {
	return &sampleErr{msg: msg}
}

type sampleErr struct {
	msg string
}

func (e *sampleErr) Error() string {
	return e.msg
}
