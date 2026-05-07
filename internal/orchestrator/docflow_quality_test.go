package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestAssessRefreshArtifactWarningsFlagsBankLikeCollapseFixture(t *testing.T) {
	t.Parallel()

	manifests, finalIndex, citationIndex, verdict := loadRefreshArtifactFixtureSet(
		t,
		filepath.Join("..", "..", "fixtures", "scenarios", "refresh-artifact-quality", "bank-collapse"),
	)
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected canonical PASS validator verdict in frozen bank fixture, got %q", verdict.Verdict)
	}

	warnings := assessRefreshArtifactWarnings(manifests, finalIndex, citationIndex)
	if len(warnings) != 2 {
		t.Fatalf("expected 2 artifact-quality warnings, got %d: %#v", len(warnings), warnings)
	}
	if !containsArtifactWarning(warnings, "only 1 generic runtime-summary citation") {
		t.Fatalf("expected runtime-summary collapse warning, got %#v", warnings)
	}
	if !containsArtifactWarning(warnings, "reuse-only and preserve no repo-specific citations") {
		t.Fatalf("expected reuse-only warning, got %#v", warnings)
	}
}

func TestAssessRefreshArtifactWarningsAllowsOpenstackRichReuseFixture(t *testing.T) {
	t.Parallel()

	manifests, finalIndex, citationIndex, verdict := loadRefreshArtifactFixtureSet(
		t,
		filepath.Join("..", "..", "fixtures", "scenarios", "refresh-artifact-quality", "openstack-rich-reuse"),
	)
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected canonical PASS validator verdict in frozen openstack fixture, got %q", verdict.Verdict)
	}

	warnings := assessRefreshArtifactWarnings(manifests, finalIndex, citationIndex)
	if len(warnings) != 0 {
		t.Fatalf("expected no artifact-quality warnings for acceptable rich reuse fixture, got %#v", warnings)
	}
}

func TestRuntimeDiagnosticCountersSurfaceRepairStallPressure(t *testing.T) {
	t.Parallel()

	execution := &pipelineExecution{}
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "retry scheduled",
		Fields: map[string]any{
			"action":      "terminate_and_validate",
			"stall_phase": "pre_artifact",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "retry scheduled",
		Fields: map[string]any{
			"action":        "fresh_process_after_stall",
			"recovery_mode": "fresh_process",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "focused artifact repair scheduled",
		Fields: map[string]any{
			"action": "focused_artifact_repair",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "zero-output pre-artifact stall classified unavailable",
		Fields: map[string]any{
			"zero_output_pre_artifact_stall": true,
			"stall_phase":                    "pre_artifact",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "focused artifact repair exhausted",
		Fields: map[string]any{
			"stall_phase":      "post_artifact",
			"validation_error": "validator verdict is invalid",
		},
	})
	execution.recordRuntimeDiagnosticCounters(acpruntime.DiagnosticEvent{
		Message: "retry exhausted",
		Fields: map[string]any{
			"stall_phase":      "pre_artifact",
			"validation_error": "runtime_stalled_before_artifacts",
		},
	})

	counters := execution.runtimeRecoveryCounters
	if counters.RepairAttempts != 2 || counters.FreshRetries != 1 || counters.FocusedRepairs != 1 {
		t.Fatalf("unexpected repair counters: %+v", counters)
	}
	if counters.RepairExhausted != 2 {
		t.Fatalf("expected exhausted repair counter, got %+v", counters)
	}
	if counters.StallCount != 2 || counters.PreArtifactStalls != 2 || counters.PostArtifactStalls != 0 {
		t.Fatalf("unexpected stall counters: %+v", counters)
	}
	if counters.ZeroOutputPreArtifactStalls != 1 {
		t.Fatalf("expected zero-output pre-artifact counter, got %+v", counters)
	}

	signals := runtimeRecoveryQualitySignals(counters, 1)
	if !hasRunQualitySignal(signals, "runtime_quality.repair_heavy") ||
		!hasRunQualitySignal(signals, "runtime_quality.stall_pressure") ||
		!hasRunQualitySignal(signals, "runtime_quality.zero_output_pre_artifact_stalls") ||
		!hasRunQualitySignal(signals, "runtime_quality.partial_failures") {
		t.Fatalf("expected runtime quality signals, got %#v", signals)
	}
}

func loadRefreshArtifactFixtureSet(
	t *testing.T,
	root string,
) ([]contracts.ShardPackManifest, contracts.FinalRunIndex, contracts.CitationIndex, contracts.ValidatorVerdict) {
	t.Helper()

	manifestPaths, err := filepath.Glob(filepath.Join(root, "manifests", "*.json"))
	if err != nil {
		t.Fatalf("glob manifests: %v", err)
	}
	if len(manifestPaths) == 0 {
		t.Fatalf("expected manifests in %s", root)
	}

	manifests := make([]contracts.ShardPackManifest, 0, len(manifestPaths))
	for _, manifestPath := range manifestPaths {
		raw, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			t.Fatalf("read manifest %s: %v", manifestPath, readErr)
		}
		manifest, parseErr := contracts.ParseShardPackManifest(raw)
		if parseErr != nil {
			t.Fatalf("parse manifest %s: %v", manifestPath, parseErr)
		}
		manifests = append(manifests, manifest)
	}

	finalIndexRaw, err := os.ReadFile(filepath.Join(root, "final-run-index.json"))
	if err != nil {
		t.Fatalf("read final-run-index: %v", err)
	}
	finalIndex, err := contracts.ParseFinalRunIndex(finalIndexRaw)
	if err != nil {
		t.Fatalf("parse final-run-index: %v", err)
	}

	citationIndexRaw, err := os.ReadFile(filepath.Join(root, "citation-index.json"))
	if err != nil {
		t.Fatalf("read citation-index: %v", err)
	}
	citationIndex, err := contracts.ParseCitationIndex(citationIndexRaw)
	if err != nil {
		t.Fatalf("parse citation-index: %v", err)
	}

	validatorVerdictRaw, err := os.ReadFile(filepath.Join(root, "validator-verdict.json"))
	if err != nil {
		t.Fatalf("read validator-verdict: %v", err)
	}
	validatorVerdict, err := contracts.ParseValidatorVerdict(validatorVerdictRaw)
	if err != nil {
		t.Fatalf("parse validator-verdict: %v", err)
	}

	return manifests, finalIndex, citationIndex, validatorVerdict
}

func hasRunQualitySignal(signals []runQualitySignal, code string) bool {
	for _, signal := range signals {
		if signal.Code == code {
			return true
		}
	}
	return false
}

func containsArtifactWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
