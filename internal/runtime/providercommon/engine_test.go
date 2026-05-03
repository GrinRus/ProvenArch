package providercommon

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
	"github.com/GrinRus/ProvenArch/internal/testutil"
)

func TestRunHeadlessProviderSucceedsWithValidArtifacts(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-valid")
	runner := testAdapter{command: writeEngineScript(t, draftScript(task, "exit 0"))}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("run provider: %v", err)
	}
	if result.Execution.Provider != string(acpruntime.ProviderQwenCode) {
		t.Fatalf("unexpected execution provider: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderAcceptsValidArtifactsAfterControlledStop(t *testing.T) {
	task := newDraftTask(t, "run-hang-valid")
	runner := testAdapter{
		command: writeEngineScript(t, draftScript(task, "sleep 5")),
		activity: ActivityPolicy{
			MonitorArtifacts:           true,
			PostArtifactStallWindow:    20 * time.Millisecond,
			PartialArtifactStallWindow: 500 * time.Millisecond,
			PollInterval:               5 * time.Millisecond,
			PostTerminateDrain:         10 * time.Millisecond,
			TerminateGrace:             10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{AcceptValidArtifactsAfterStop: true},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected controlled-stop artifact success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderClassifiesSilentRetryExhaustionUnavailable(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-silent-retry")
	runner := testAdapter{
		command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 5\n"),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      time.Second,
			RetryPreArtifactStallWindow: time.Second,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			RetryInvalidOrMissingArtifactsOnce:       true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected runner error")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
}

func TestRunHeadlessProviderKeepsInvalidArtifactsAsContractFailure(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-invalid")
	script := "#!/usr/bin/env bash\nset -eu\nmkdir -p " + shellQuote(task.WriteRoot) + "\nprintf '%s\\n' '{\"version\":1,\"outputs\":[]}' > " + shellQuote(filepath.Join(task.WriteRoot, "constitution-draft.json")) + "\n"
	runner := testAdapter{command: writeEngineScript(t, script)}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
}

func TestRunHeadlessProviderClassifiesUnavailableMarkerWhenArtifactsMissing(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-unavailable-marker")
	runner := testAdapter{command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' 'API Error: 429 rate limit'\n")}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected runner error")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Stdout, "429") {
		t.Fatalf("expected stdout diagnostics to be preserved, got %q", runnerErr.Stdout)
	}
}

func TestRunHeadlessProviderDoesNotTreatStdoutJSONAsSuccess(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-json-stdout")
	runner := testAdapter{command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' '{\"status\":\"ok\"}'\n")}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected missing artifacts to fail despite stdout JSON")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if strings.TrimSpace(runnerErr.RawOutputRefs.Stdout) == "" || strings.TrimSpace(runnerErr.RawOutputRefs.Metadata) == "" {
		t.Fatalf("expected stdout and metadata raw refs, got %+v", runnerErr.RawOutputRefs)
	}
	stdoutRaw, readErr := os.ReadFile(filepath.Join(task.Workspace, filepath.FromSlash(runnerErr.RawOutputRefs.Stdout)))
	if readErr != nil {
		t.Fatalf("read raw stdout: %v", readErr)
	}
	if !strings.Contains(string(stdoutRaw), `"status":"ok"`) {
		t.Fatalf("expected stdout JSON persisted as diagnostics, got %q", stdoutRaw)
	}
	metaRaw, readErr := os.ReadFile(filepath.Join(task.Workspace, filepath.FromSlash(runnerErr.RawOutputRefs.Metadata)))
	if readErr != nil {
		t.Fatalf("read raw metadata: %v", readErr)
	}
	meta := map[string]any{}
	if decodeErr := json.Unmarshal(metaRaw, &meta); decodeErr != nil {
		t.Fatalf("decode raw metadata: %v", decodeErr)
	}
	if meta["provider"] != string(acpruntime.ProviderQwenCode) {
		t.Fatalf("expected provider metadata, got %#v", meta)
	}
}

func TestRunHeadlessProviderClassifiesContextDeadlineAsRuntimeTimeout(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-timeout")
	runner := testAdapter{command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' 'started'\nsleep 5\n")}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected timeout")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeTimeout {
		t.Fatalf("expected runtime_timeout, got %s (%v)", runnerErr.Code, err)
	}
	if strings.TrimSpace(runnerErr.RawOutputRefs.Metadata) == "" {
		t.Fatal("expected raw output metadata reference")
	}
}

func TestRunHeadlessProviderRetriesArtifactValidationFailureWhenPolicyAllows(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-retry-after-invalid")
	statePath := filepath.Join(task.Workspace, "attempt-state")
	script := `#!/usr/bin/env bash
set -eu
state=` + shellQuote(statePath) + `
if [[ ! -f "$state" ]]; then
  printf 'first-attempt\n' > "$state"
  exit 0
fi
` + draftScript(task, "exit 0")
	runner := testAdapter{
		command:  writeEngineScript(t, script),
		recovery: RecoveryPolicy{RetryInvalidOrMissingArtifactsOnce: true},
	}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("expected artifact retry success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderRepairsMissingCollectManifestWithAuthoredDocs(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
`
	repairScript := `#!/usr/bin/env bash
set -eu
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + collectManifestJSON(task) + `
EOF
`
	runner := testAdapter{
		command:       writeEngineScript(t, initialScript),
		repairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectManifestOnce:     true,
		},
	}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("expected collect manifest repair success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderRepairsNonSilentNoArtifactCollectWithPairRepair(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-pair-repair")
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	manifest := strings.Replace(collectManifestJSON(task), `"path": "overview.md"`, `"path": "`+docRel+`"`, 1)
	initialScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' 'codex collect started'
`
	repairScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, filepath.FromSlash(docRel))) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + manifest + `
EOF
`
	runner := testAdapter{
		command:           writeEngineScript(t, initialScript),
		pairRepairCommand: writeEngineScript(t, repairScript),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          false,
			PreArtifactStallWindow:      250 * time.Millisecond,
			RetryPreArtifactStallWindow: 250 * time.Millisecond,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PartialArtifactStallWindow:  20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectArtifactPairOnce: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected collect pair repair success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderRejectsCollectPairRepairExtraWrites(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-pair-repair-extra-write")
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	manifest := strings.Replace(collectManifestJSON(task), `"path": "overview.md"`, `"path": "`+docRel+`"`, 1)
	initialScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' 'codex collect started'
`
	repairScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, filepath.FromSlash(docRel))) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + manifest + `
EOF
printf '%s\n' '# Repair Notes' > ` + shellQuote(filepath.Join(task.WriteRoot, "repair-notes.md")) + `
`
	runner := testAdapter{
		command:           writeEngineScript(t, initialScript),
		pairRepairCommand: writeEngineScript(t, repairScript),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          false,
			PreArtifactStallWindow:      250 * time.Millisecond,
			RetryPreArtifactStallWindow: 250 * time.Millisecond,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PartialArtifactStallWindow:  20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectArtifactPairOnce: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected collect pair repair write-set contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "collect pair recovery wrote forbidden files") {
		t.Fatalf("expected collect pair write-set failure, got %v", err)
	}
	if !strings.Contains(runnerErr.Error(), "created repair-notes.md") {
		t.Fatalf("expected forbidden file path in error, got %v", err)
	}
}

func TestCollectPairRepairDoesNotMaskSilentNoArtifactCollect(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-pair-repair-silent")
	runner := testAdapter{
		command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 5\n"),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      20 * time.Millisecond,
			RetryPreArtifactStallWindow: 20 * time.Millisecond,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PartialArtifactStallWindow:  20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RepairCollectArtifactPairOnce:            true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected silent collect to remain provider unavailable")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
	if strings.Contains(runnerErr.Error(), "collect_pair_repair") {
		t.Fatalf("silent no-artifact collect should not enter collect pair repair, got %v", err)
	}
}

func TestRunHeadlessProviderRepairsMissingCollectManifestWithNestedAuthoredDocs(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair-nested")
	nestedDocPath := filepath.Join(task.WriteRoot, "docs", "overview.md")
	nestedManifest := strings.Replace(collectManifestJSON(task), `"path": "overview.md"`, `"path": "docs/overview.md"`, 1)
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(filepath.Dir(nestedDocPath)) + `
printf '%s\n' '# Nested Collect Overview' > ` + shellQuote(nestedDocPath) + `
`
	repairScript := `#!/usr/bin/env bash
set -eu
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + nestedManifest + `
EOF
`
	runner := testAdapter{
		command:       writeEngineScript(t, initialScript),
		repairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectManifestOnce:     true,
		},
	}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("expected nested collect manifest repair success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderRejectsCollectRepairExtraWrites(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair-extra-write")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
`
	repairScript := `#!/usr/bin/env bash
set -eu
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + collectManifestJSON(task) + `
EOF
printf '%s\n' '# Repair Notes' > ` + shellQuote(filepath.Join(task.WriteRoot, "repair-notes.md")) + `
`
	runner := testAdapter{
		command:       writeEngineScript(t, initialScript),
		repairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectManifestOnce:     true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected collect repair write-set contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "manifest-only collect repair wrote forbidden files") {
		t.Fatalf("expected manifest-only write-set failure, got %v", err)
	}
	if !strings.Contains(runnerErr.Error(), "created repair-notes.md") {
		t.Fatalf("expected forbidden file path in error, got %v", err)
	}
}

func TestRunHeadlessProviderRejectsCollectRepairExtraWritesBeforeCommandFailure(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair-extra-write-exit")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
`
	repairScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(filepath.Join(task.WriteRoot, "scratch-dir")) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + collectManifestJSON(task) + `
EOF
exit 2
`
	runner := testAdapter{
		command:       writeEngineScript(t, initialScript),
		repairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectManifestOnce:     true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected collect repair write-set contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "manifest-only collect repair wrote forbidden files") {
		t.Fatalf("expected write-set failure to win over command failure, got %v", err)
	}
	if !strings.Contains(runnerErr.Error(), "created directory scratch-dir") {
		t.Fatalf("expected forbidden directory path in error, got %v", err)
	}
}

func TestRunHeadlessProviderClassifiesQwenPartialCollectArtifactsAsContractFailure(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-partial")
	script := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
`
	runner := testAdapter{
		command:       writeEngineScript(t, script),
		repairCommand: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RepairCollectManifestOnce:                true,
			RetryInvalidOrMissingArtifactsOnce:       false,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected collect contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed for partial collect artifacts, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "manifest-only collect repair") {
		t.Fatalf("expected manifest repair context in error, got %v", err)
	}
}

func TestRunHeadlessProviderRepairsMissingValidatorVerdict(t *testing.T) {
	t.Parallel()

	task := newValidatorTask(t, "run-validator-repair")
	initialScript := "#!/usr/bin/env bash\nset -eu\nexit 0\n"
	repairScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ValidatorVerdictFileName)) + ` <<'EOF'
` + validatorVerdictJSON(task) + `
EOF
`
	runner := testAdapter{
		command:                writeEngineScript(t, initialScript),
		validatorRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairValidatorVerdictOnce:    true,
		},
	}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("expected validator verdict repair success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderRepairsMissingValidatorVerdictAfterNoArtifactStall(t *testing.T) {
	t.Parallel()

	task := newValidatorTask(t, "run-validator-stall-repair")
	repairScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ValidatorVerdictFileName)) + ` <<'EOF'
` + validatorVerdictJSON(task) + `
EOF
`
	runner := testAdapter{
		command:                writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 5\n"),
		validatorRepairCommand: writeEngineScript(t, repairScript),
		activity: ActivityPolicy{
			MonitorArtifacts:        true,
			MonitorPreArtifact:      true,
			PreArtifactStallWindow:  20 * time.Millisecond,
			PostArtifactStallWindow: 20 * time.Millisecond,
			PollInterval:            5 * time.Millisecond,
			PostTerminateDrain:      10 * time.Millisecond,
			TerminateGrace:          10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RepairValidatorVerdictOnce:               true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected validator verdict repair after no-artifact stall, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderRepairsInvalidValidatorVerdict(t *testing.T) {
	t.Parallel()

	task := newValidatorTask(t, "run-validator-invalid-repair")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ValidatorVerdictFileName)) + ` <<'EOF'
{"version":1,"run_id":"` + task.RunID + `","generated_at":"2026-04-16T12:00:02Z","verdict":"FAIL","checked_paths":["reports/taskruns/` + task.RunID + `/staging/final/final-run-index.json"],"issues":[{"id":"legacy","severity":"high","title":"Legacy issue","description":"Wrong shape","rule_id":"legacy.rule","related_paths":["reports/as-is/overview.md"]}]}
EOF
`
	repairScript := `#!/usr/bin/env bash
set -eu
cat >` + shellQuote(filepath.Join(task.WriteRoot, ValidatorVerdictFileName)) + ` <<'EOF'
` + validatorVerdictJSON(task) + `
EOF
`
	runner := testAdapter{
		command:                writeEngineScript(t, initialScript),
		validatorRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairValidatorVerdictOnce:    true,
		},
	}

	if _, err := RunHeadlessProvider(context.Background(), task, runner); err != nil {
		t.Fatalf("expected invalid validator verdict repair success, got %v", err)
	}
}

func TestRunHeadlessProviderRejectsValidatorRepairExtraWrites(t *testing.T) {
	t.Parallel()

	task := newValidatorTask(t, "run-validator-repair-extra-write")
	repairScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ValidatorVerdictFileName)) + ` <<'EOF'
` + validatorVerdictJSON(task) + `
EOF
printf '%s\n' '# Notes' > ` + shellQuote(filepath.Join(task.WriteRoot, "notes.md")) + `
`
	runner := testAdapter{
		command:                writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		validatorRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairValidatorVerdictOnce:    true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected validator repair write-set contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "verdict-only validator repair wrote forbidden files") {
		t.Fatalf("expected validator write-set failure, got %v", err)
	}
}

func TestRunHeadlessProviderRepairsDraftArtifactsWithDraftFinalWrites(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-draft-repair")
	repairScript := draftScript(task, "exit 0")
	runner := testAdapter{
		command:            writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		draftRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairDraftArtifactsOnce:      true,
		},
	}

	if _, err := RunHeadlessProvider(context.Background(), task, runner); err != nil {
		t.Fatalf("expected draft artifact repair success, got %v", err)
	}
}

func TestRunHeadlessProviderRejectsDraftRepairExtraWriteRootFiles(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-draft-repair-extra-write")
	repairScript := draftScript(task, "printf '%s\\n' '# Extra' > "+shellQuote(filepath.Join(task.WriteRoot, "extra.md"))+"\n")
	runner := testAdapter{
		command:            writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		draftRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairDraftArtifactsOnce:      true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected draft repair write-set contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "draft repair wrote forbidden write_root files") {
		t.Fatalf("expected draft write-set failure, got %v", err)
	}
}

func TestSilentRetryExhaustionUnavailableRequiresNoAuthoredArtifacts(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-retry-partial")
	policy := RecoveryPolicy{ClassifySilentRetryExhaustionUnavailable: true}
	if !shouldClassifySilentRetryExhaustionUnavailable(policy, task, acpruntime.Result{}) {
		t.Fatal("empty collect write_root should be eligible for silent unavailable classification")
	}
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "overview.md"), []byte("# Collect Overview\n"), 0o644); err != nil {
		t.Fatalf("write authored collect doc: %v", err)
	}
	if shouldClassifySilentRetryExhaustionUnavailable(policy, task, acpruntime.Result{}) {
		t.Fatal("authored collect artifacts must keep silent retry exhaustion in runtime_contract_failed lane")
	}
}

func TestRunHeadlessProviderKeepsInitialMalformedArtifactAsContractFailureAfterSilentRetry(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-malformed-then-silent")
	statePath := filepath.Join(task.Workspace, "attempt-state")
	script := `#!/usr/bin/env bash
set -eu
state=` + shellQuote(statePath) + `
write_root=` + shellQuote(task.WriteRoot) + `
draft_root=` + shellQuote(task.DraftFinalRoot) + `
if [[ ! -f "$state" ]]; then
  printf 'first-attempt\n' > "$state"
  mkdir -p "$write_root"
  printf '%s\n' '{"version":1,"run_id":"` + task.RunID + `","step_id":"init.step0.constitution","step_contract":"constitution","agent_role":"architect","outputs":[]}' > "$write_root/constitution-draft.json"
  exit 0
fi
rm -rf "$write_root" "$draft_root"
mkdir -p "$write_root" "$draft_root"
exit 0
`
	runner := testAdapter{
		command: writeEngineScript(t, script),
		recovery: RecoveryPolicy{
			RetryInvalidOrMissingArtifactsOnce:       true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed to preserve initial malformed artifact, got %s (%v)", runnerErr.Code, err)
	}
}

func TestMonitorArtifactStallObservesDraftArtifactMutationWindow(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-monitor")
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "constitution-draft.json"), []byte(`{
  "version": 1,
  "run_id": "`+task.RunID+`",
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {"path":"charter-overview.md","canonical_path":"charter/overview.md"},
    {"path":"baseline-subagents.yaml","canonical_path":"skills/subagents.yaml"}
  ]
}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, "charter-overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatalf("write overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, "baseline-subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write subagents: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker := newCommandActivityTracker(time.Now().Add(-time.Second))
	done := make(chan bool, 1)
	go func() {
		_, stalled := monitorArtifactStall(ctx, task, tracker, ActivityPolicy{
			MonitorArtifacts:        true,
			PostArtifactStallWindow: 20 * time.Millisecond,
			PollInterval:            5 * time.Millisecond,
		})
		done <- stalled
	}()

	select {
	case stalled := <-done:
		if !stalled {
			t.Fatal("expected monitor stall")
		}
	case <-time.After(time.Second):
		t.Fatal("expected monitor to observe valid draft artifacts")
	}
}

func TestDraftArtifactSnapshotObservesNestedDraftFinalFiles(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-monitor-nested-draft")
	nestedPath := filepath.Join(task.DraftFinalRoot, "reports", "changelog", "runtime-proposals.md")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0o755); err != nil {
		t.Fatalf("mkdir nested draft path: %v", err)
	}
	if err := os.WriteFile(nestedPath, []byte("# Runtime Proposals\n"), 0o644); err != nil {
		t.Fatalf("write nested draft artifact: %v", err)
	}

	snapshot := draftArtifactSnapshot(task)
	if !snapshot.ArtifactObserved {
		t.Fatalf("expected nested draft final file to count as observed artifact: %#v", snapshot)
	}
	if snapshot.AuthoredFiles != 1 {
		t.Fatalf("authored draft files = %d, want 1: %#v", snapshot.AuthoredFiles, snapshot)
	}
	if snapshot.LastMutation.IsZero() {
		t.Fatalf("expected nested draft final file to update last mutation: %#v", snapshot)
	}
}

func TestMonitorArtifactStallTripsPreArtifactWindow(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-monitor-pre")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker := newCommandActivityTracker(time.Now().Add(-time.Second))
	done := make(chan bool, 1)
	go func() {
		err, stalled := monitorArtifactStall(ctx, task, tracker, ActivityPolicy{
			MonitorArtifacts:        true,
			MonitorPreArtifact:      true,
			PreArtifactStallWindow:  20 * time.Millisecond,
			PostArtifactStallWindow: 20 * time.Millisecond,
			PollInterval:            5 * time.Millisecond,
		})
		done <- stalled && errors.Is(err, ErrStalledBeforeArtifacts)
	}()

	select {
	case ok := <-done:
		if !ok {
			t.Fatal("expected pre-artifact stall")
		}
	case <-time.After(time.Second):
		t.Fatal("expected monitor to trip pre-artifact window")
	}
}

func TestMonitorArtifactStallUsesPartialArtifactWindow(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-monitor-partial")
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "partial.md"), []byte("# partial\n"), 0o644); err != nil {
		t.Fatalf("write partial artifact: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker := newCommandActivityTracker(time.Now().Add(-time.Second))
	done := make(chan bool, 1)
	go func() {
		_, stalled := monitorArtifactStall(ctx, task, tracker, ActivityPolicy{
			MonitorArtifacts:           true,
			PostArtifactStallWindow:    20 * time.Millisecond,
			PartialArtifactStallWindow: time.Second,
			PollInterval:               5 * time.Millisecond,
		})
		done <- stalled
	}()

	select {
	case <-done:
		t.Fatal("partial artifacts should keep waiting until partial grace expires")
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
}

func TestCollectArtifactSnapshotIgnoresRuntimeExecutionMetadata(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-runtime-execution-only")
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "runtime-execution.json"), []byte(`{"status":"failed"}`), 0o644); err != nil {
		t.Fatalf("write runtime execution metadata: %v", err)
	}

	snapshot := collectArtifactSnapshot(task.WriteRoot)
	if snapshot.AuthoredFiles != 0 {
		t.Fatalf("runtime-execution.json must not count as authored collect artifact, got %d", snapshot.AuthoredFiles)
	}
	if snapshot.ArtifactObserved {
		t.Fatalf("runtime-execution.json alone must not count as provider artifact: %#v", snapshot)
	}
}

type testAdapter struct {
	command                string
	repairCommand          string
	pairRepairCommand      string
	validatorRepairCommand string
	draftRepairCommand     string
	activity               ActivityPolicy
	recovery               RecoveryPolicy
}

func (a testAdapter) Provider() acpruntime.Provider {
	return acpruntime.ProviderQwenCode
}

func (a testAdapter) RuntimeVersion() string {
	return "test"
}

func (a testAdapter) CommandSpec(acpruntime.Task) (CommandSpec, error) {
	return CommandSpec{Command: a.command}, nil
}

func (a testAdapter) CollectManifestRepairCommandSpec(acpruntime.Task, error) (CommandSpec, error) {
	if strings.TrimSpace(a.repairCommand) == "" {
		return CommandSpec{}, errors.New("repair command is unavailable")
	}
	return CommandSpec{Command: a.repairCommand}, nil
}

func (a testAdapter) CollectArtifactPairRepairCommandSpec(acpruntime.Task, error) (CommandSpec, error) {
	if strings.TrimSpace(a.pairRepairCommand) == "" {
		return CommandSpec{}, errors.New("collect pair repair command is unavailable")
	}
	return CommandSpec{Command: a.pairRepairCommand}, nil
}

func (a testAdapter) ValidatorVerdictRepairCommandSpec(acpruntime.Task, error) (CommandSpec, error) {
	if strings.TrimSpace(a.validatorRepairCommand) == "" {
		return CommandSpec{}, errors.New("validator repair command is unavailable")
	}
	return CommandSpec{Command: a.validatorRepairCommand}, nil
}

func (a testAdapter) DraftArtifactRepairCommandSpec(acpruntime.Task, error) (CommandSpec, error) {
	if strings.TrimSpace(a.draftRepairCommand) == "" {
		return CommandSpec{}, errors.New("draft repair command is unavailable")
	}
	return CommandSpec{Command: a.draftRepairCommand}, nil
}

func (a testAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return ValidateRuntimeArtifacts(task, a.Provider())
}

func (a testAdapter) ActivityPolicy(acpruntime.Task) ActivityPolicy {
	return a.activity
}

func (a testAdapter) RecoveryPolicy(acpruntime.Task) RecoveryPolicy {
	return a.recovery
}

func (a testAdapter) UnavailableMarkers() []string {
	return DefaultUnavailableMarkers()
}

func newDraftTask(t *testing.T, runID string) acpruntime.Task {
	t.Helper()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", runID, "constitution")
	draftRoot := filepath.Join(workspace, "reports", "taskruns", runID, "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	return acpruntime.Task{
		TaskID:            "task-" + runID,
		RunID:             runID,
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		AgentRole:         "architect",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
		StartedAtUTC:      time.Now().UTC(),
	}
}

func newCollectTask(t *testing.T, runID string) acpruntime.Task {
	t.Helper()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", runID, "staging", "shards", "bank")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	return acpruntime.Task{
		TaskID:            "task-" + runID,
		RunID:             runID,
		StepID:            "init.step1.collect",
		StepContract:      "collect",
		AgentRole:         "shard-analyst",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		ArtifactRoot:      "reports/taskruns/" + runID + "/staging/shards/bank",
		ShardID:           "bank",
		DomainID:          "bank",
		RepoScopes:        []string{"bank-of-anthos"},
		PathScopes:        []string{"."},
		ExpectedArtifacts: []string{ShardPackManifestFileName},
		StartedAtUTC:      time.Now().UTC(),
	}
}

func newValidatorTask(t *testing.T, runID string) acpruntime.Task {
	t.Helper()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", runID, "validator")
	stagedRoot := filepath.Join(workspace, "reports", "taskruns", runID, "staging", "final")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(stagedRoot, 0o755); err != nil {
		t.Fatalf("mkdir staged root: %v", err)
	}
	for _, name := range []string{"final-run-index.json", "citation-index.json"} {
		if err := os.WriteFile(filepath.Join(stagedRoot, name), []byte(`{"version":1}`+"\n"), 0o644); err != nil {
			t.Fatalf("write staged %s: %v", name, err)
		}
	}
	return acpruntime.Task{
		TaskID:            "task-" + runID,
		RunID:             runID,
		StepID:            "init.step3.findings",
		StepContract:      "validator",
		AgentRole:         "architect",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    stagedRoot,
		ReadContextRoots:  []string{stagedRoot},
		ExpectedArtifacts: []string{ValidatorVerdictFileName},
		StartedAtUTC:      time.Now().UTC(),
	}
}

func validatorVerdictJSON(task acpruntime.Task) string {
	return `{
  "version": 1,
  "run_id": "` + task.RunID + `",
  "generated_at": "2026-04-16T12:00:02Z",
  "verdict": "PASS",
  "summary": "No blocking technical validator issues remain.",
  "checked_paths": [
    "reports/taskruns/` + task.RunID + `/staging/final/final-run-index.json",
    "reports/taskruns/` + task.RunID + `/staging/final/citation-index.json"
  ],
  "fixed_paths": [],
  "findings": [],
  "questions": [],
  "issues": []
}`
}

func collectManifestJSON(task acpruntime.Task) string {
	return `{
  "version": 1,
  "run_id": "` + task.RunID + `",
  "step_id": "init.step1.collect",
  "shard_id": "bank",
  "domain_id": "bank",
  "agent_role": "shard-analyst",
  "artifact_root": "` + task.ArtifactRoot + `",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["."],
  "documents": [
    {
      "id": "doc.bank.overview",
      "kind": "report",
      "title": "Bank Overview",
      "path": "overview.md",
      "canonical_path": "reports/as-is/bank/overview.md",
      "topics": ["bank"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.bank.readme"],
      "document_ids": ["doc.bank.overview"]
    }
  ],
  "semantic": {
    "coverage": {
      "observed": ["repository entrypoints"],
      "missing": ["owner mapping"],
      "notes": ["repair test fixture"]
    },
    "questions": [
      {
        "id": "q.bank.owner",
        "text": "Who owns bank-of-anthos?"
      }
    ],
    "entities": [
      {
        "id": "svc.bank",
        "name": "bank",
        "type": "service",
        "provenance": {
          "kind": "observation",
          "confidence": 0.8,
          "evidence": [
            {
              "repo": "bank-of-anthos",
              "path": "README.md"
            }
          ]
        }
      }
    ],
    "edges": [],
    "findings": []
  }
}`
}

func writeEngineScript(t *testing.T, script string) string {
	t.Helper()
	return testutil.WriteExecutableScript(t, "provider-stub.sh", script)
}

func draftScript(task acpruntime.Task, tail string) string {
	return `#!/usr/bin/env bash
set -eu
write_root=` + shellQuote(task.WriteRoot) + `
draft_root=` + shellQuote(task.DraftFinalRoot) + `
run_id=` + shellQuote(task.RunID) + `
mkdir -p "$write_root" "$draft_root"
cat >"$write_root/constitution-draft.json" <<EOF
{
  "version": 1,
  "run_id": "$run_id",
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
}
EOF
cat >"$draft_root/charter-overview.md" <<'EOF'
# Constitution
EOF
cat >"$draft_root/baseline-subagents.yaml" <<'EOF'
version: 1
EOF
` + tail + "\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
