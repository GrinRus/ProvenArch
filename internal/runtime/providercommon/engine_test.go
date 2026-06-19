package providercommon

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
	"github.com/GrinRus/ProvenArch/internal/testutil"
)

const (
	successfulZeroOutputRetryWindow      = 30 * time.Second
	successfulZeroOutputRetryTimeout     = 45 * time.Second
	successfulArtifactWriteWindow        = 500 * time.Millisecond
	successfulCollectPairRecoveryTimeout = 30 * time.Second
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

func TestNormalizeActivityPolicyAppliesDiagnosticEnvOverrides(t *testing.T) {
	t.Setenv("ACP_PROVIDER_PRE_ARTIFACT_STALL_SEC", "301")
	t.Setenv("ACP_PROVIDER_RETRY_PRE_ARTIFACT_STALL_SEC", "302")
	t.Setenv("ACP_PROVIDER_POST_ARTIFACT_STALL_SEC", "303")
	t.Setenv("ACP_PROVIDER_PARTIAL_ARTIFACT_STALL_SEC", "304")
	t.Setenv("ACP_PROVIDER_VALID_ARTIFACT_STOP_SEC", "305")

	policy := normalizeActivityPolicy(ActivityPolicy{})

	if got, want := policy.PreArtifactStallWindow, 301*time.Second; got != want {
		t.Fatalf("pre artifact window = %s, want %s", got, want)
	}
	if got, want := policy.RetryPreArtifactStallWindow, 302*time.Second; got != want {
		t.Fatalf("retry pre artifact window = %s, want %s", got, want)
	}
	if got, want := policy.PostArtifactStallWindow, 303*time.Second; got != want {
		t.Fatalf("post artifact window = %s, want %s", got, want)
	}
	if got, want := policy.PartialArtifactStallWindow, 304*time.Second; got != want {
		t.Fatalf("partial artifact window = %s, want %s", got, want)
	}
	if got, want := policy.ValidArtifactStopWindow, 305*time.Second; got != want {
		t.Fatalf("valid artifact stop window = %s, want %s", got, want)
	}
}

func TestNormalizeActivityPolicyIgnoresInvalidDiagnosticEnvOverrides(t *testing.T) {
	t.Setenv("ACP_PROVIDER_PRE_ARTIFACT_STALL_SEC", "bad")
	t.Setenv("ACP_PROVIDER_POST_ARTIFACT_STALL_SEC", "-1")

	policy := normalizeActivityPolicy(ActivityPolicy{})

	if got, want := policy.PreArtifactStallWindow, defaultPreArtifactStallWindow; got != want {
		t.Fatalf("pre artifact window = %s, want default %s", got, want)
	}
	if got, want := policy.PostArtifactStallWindow, defaultPostArtifactStallWindow; got != want {
		t.Fatalf("post artifact window = %s, want default %s", got, want)
	}
}

func TestRunHeadlessProviderManagedPermissionsFailFastWithoutProtocol(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-managed-no-protocol")
	task.RuntimePermissions = acpruntime.PermissionValues{Mode: acpruntime.PermissionModeManaged, ApprovalChannel: acpruntime.PermissionApprovalFailFast}
	runner := testAdapter{command: writeEngineScript(t, draftScript(task, "exit 0"))}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatalf("expected managed permission protocol error")
	}
	code, _, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodePermissionRequired) {
		t.Fatalf("expected runtime_permission_required, got code=%q err=%v", code, err)
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

func TestRunHeadlessProviderAcceptsQwenDraftArtifactsAfterValidWindow(t *testing.T) {
	task := newAsIsDraftTask(t, "run-valid-draft-overrun")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	tail := `while true; do
  printf '%s\n' 'still refining as-is draft'
  printf '%s\n' '- Provider is still refining after valid artifacts.' >> "$draft_root/architect-summary.md"
  sleep 0.01
done`
	runner := testAdapter{
		command: writeEngineScript(t, asIsDraftScript(task, []string{"overview.md", "summary.md", "architect-summary.md"}, tail)),
		activity: ActivityPolicy{
			MonitorArtifacts:   true,
			MonitorPreArtifact: true,
			// This test exercises the valid-artifact stop path; keep the
			// pre-artifact budget roomy enough for loaded live prechecks.
			PreArtifactStallWindow:     20 * time.Second,
			PostArtifactStallWindow:    time.Second,
			PartialArtifactStallWindow: time.Second,
			ValidArtifactStopWindow:    50 * time.Millisecond,
			PollInterval:               5 * time.Millisecond,
			PostTerminateDrain:         10 * time.Millisecond,
			TerminateGrace:             10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{AcceptValidArtifactsAfterStop: true},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected valid draft artifact stop to recover success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	if _, _, err := ValidateRequiredRuntimeDraftArtifacts(task); err != nil {
		t.Fatalf("expected valid as-is draft artifacts: %v", err)
	}
	if !hasDiagnosticField(diagnostics, "retry completed", "recovery_mode", "artifact_only") {
		t.Fatalf("expected artifact_only recovery diagnostic, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderClassifiesSilentRetryExhaustionUnavailable(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-silent-retry")
	task.RuntimeTimeoutProfile = map[string]any{
		"step_timeout_sec":      123,
		"heartbeat_timeout_sec": 7,
	}
	runner := testAdapter{
		command:     writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
		promptBytes: 42,
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      time.Second,
			RetryPreArtifactStallWindow: 5 * time.Second,
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
	startedAt := time.Now()
	_, err := RunHeadlessProvider(ctx, task, runner)
	elapsed := time.Since(startedAt)
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
	if elapsed >= 3*time.Second {
		t.Fatalf("expected fail-fast before 5s fresh retry window, elapsed %s", elapsed)
	}
	if runnerErr.RawOutputRefs.Metadata == "" {
		t.Fatal("expected raw metadata refs to be persisted")
	}
	rawMeta, readErr := os.ReadFile(filepath.Join(task.Workspace, runnerErr.RawOutputRefs.Metadata))
	if readErr != nil {
		t.Fatalf("read raw metadata: %v", readErr)
	}
	var meta map[string]any
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		t.Fatalf("parse raw metadata: %v", err)
	}
	diagnostics, ok := meta["diagnostics"].(map[string]any)
	if !ok {
		t.Fatalf("expected diagnostics map in raw metadata: %#v", meta)
	}
	lifecycle, ok := diagnostics["provider_lifecycle"].(map[string]any)
	if !ok {
		t.Fatalf("expected provider lifecycle diagnostics: %#v", diagnostics)
	}
	if lifecycle["selected_provider"] != string(acpruntime.ProviderQwenCode) {
		t.Fatalf("expected selected provider diagnostic, got %#v", lifecycle["selected_provider"])
	}
	if _, ok := lifecycle["command_path"].(string); !ok {
		t.Fatalf("expected resolved command path diagnostic, got %#v", lifecycle["command_path"])
	}
	timeoutProfile, ok := lifecycle["timeout_profile"].(map[string]any)
	if !ok {
		t.Fatalf("expected timeout profile diagnostics: %#v", lifecycle)
	}
	if got := int(timeoutProfile["step_timeout_sec"].(float64)); got != 123 {
		t.Fatalf("expected step timeout diagnostic 123, got %d", got)
	}
	if got := int(lifecycle["prompt_bytes"].(float64)); got != 42 {
		t.Fatalf("expected prompt_bytes diagnostic 42, got %d", got)
	}
	activityPolicy, ok := lifecycle["activity_policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected activity policy diagnostics: %#v", lifecycle)
	}
	if got := int(activityPolicy["pre_artifact_stall_window_ms"].(float64)); got != 1000 {
		t.Fatalf("expected pre-artifact window diagnostic 1000ms, got %d", got)
	}
}

func TestRunHeadlessProviderRetriesZeroOutputPreArtifactStallWhenPolicyAllows(t *testing.T) {
	task := newDraftTask(t, "run-silent-then-success")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	retryScript := strings.Replace(
		compactDraftScript(task),
		"set -eu\n",
		"set -eu\nprintf '%s\\n' 'retry writing artifacts'\n",
		1,
	)
	runner := &sequenceAdapter{
		testAdapter: testAdapter{
			activity: ActivityPolicy{
				MonitorArtifacts:            true,
				MonitorPreArtifact:          true,
				PreArtifactStallWindow:      20 * time.Millisecond,
				RetryPreArtifactStallWindow: successfulZeroOutputRetryWindow,
				PostArtifactStallWindow:     20 * time.Millisecond,
				PollInterval:                5 * time.Millisecond,
				PostTerminateDrain:          10 * time.Millisecond,
				TerminateGrace:              10 * time.Millisecond,
			},
			recovery: RecoveryPolicy{
				AcceptValidArtifactsAfterStop:            true,
				RetryInvalidOrMissingArtifactsOnce:       true,
				RetryZeroOutputPreArtifactStallOnce:      true,
				ClassifySilentRetryExhaustionUnavailable: true,
			},
		},
		commands: []string{
			writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
			writeEngineScript(t, retryScript),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulZeroOutputRetryTimeout)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected retry to recover valid artifacts, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("expected succeeded execution, got %+v", result.Execution)
	}
	if !hasDiagnostic(diagnostics, "zero-output pre-artifact stall will retry", "warning") {
		t.Fatalf("expected warning diagnostic for recovered zero-output stall, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderKeepsExhaustedZeroOutputRetryUnavailable(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-silent-retry-exhausted")
	runner := testAdapter{
		command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      20 * time.Millisecond,
			RetryPreArtifactStallWindow: 20 * time.Millisecond,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			RetryZeroOutputPreArtifactStallOnce:      true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected exhausted zero-output retry to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
}

func TestRunHeadlessProviderClassifiesSilentPreArtifactStallBeforeDraftRepair(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-silent-before-draft-repair")
	repairMarker := filepath.Join(task.Workspace, "draft-repair-called")
	runner := testAdapter{
		command:            writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
		draftRepairCommand: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nprintf called > "+shellQuote(repairMarker)+"\n"),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      20 * time.Millisecond,
			RetryPreArtifactStallWindow: 5 * time.Second,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RepairDraftArtifactsOnce:                 true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	if _, statErr := os.Stat(repairMarker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("draft repair should not run for silent zero-output pre-artifact stall, statErr=%v", statErr)
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

func TestRedactArgsMasksInlineSecretFlags(t *testing.T) {
	t.Parallel()

	got := redactArgs([]string{
		"--api-key=super-secret",
		"--token",
		"next-secret",
		"--model=qwen",
	})
	joined := strings.Join(got, " ")
	if strings.Contains(joined, "super-secret") || strings.Contains(joined, "next-secret") {
		t.Fatalf("secret values leaked in redacted args: %#v", got)
	}
	if !strings.Contains(joined, "--api-key=<redacted sha256=") {
		t.Fatalf("expected inline secret flag to be redacted, got %#v", got)
	}
	if !strings.Contains(joined, "--model=qwen") {
		t.Fatalf("non-secret model arg should remain visible, got %#v", got)
	}
}

func TestRedactArgsMasksPromptPayloadFlags(t *testing.T) {
	t.Parallel()

	prompt := "You are ACP runtime provider \"qwen-code\".\n\nFIRST COLLECT ARTIFACT PAIR COMMAND:\ncat > artifact"
	got := redactArgs([]string{
		"--chat-recording",
		"false",
		"-p",
		prompt,
		"--prompt=second prompt body",
		"--model=qwen",
	})
	joined := strings.Join(got, " ")
	if strings.Contains(joined, "FIRST COLLECT ARTIFACT PAIR COMMAND") || strings.Contains(joined, "second prompt body") {
		t.Fatalf("prompt payload leaked in redacted args: %#v", got)
	}
	if !strings.Contains(joined, "-p <redacted prompt bytes=") {
		t.Fatalf("expected -p payload to be redacted with byte count, got %#v", got)
	}
	if !strings.Contains(joined, "--prompt=<redacted prompt bytes=") {
		t.Fatalf("expected inline prompt payload to be redacted with byte count, got %#v", got)
	}
	if !strings.Contains(joined, "--model=qwen") {
		t.Fatalf("non-prompt args should remain visible, got %#v", got)
	}
}

func TestRedactedEnvValueOmitsSecretLikeCommandValues(t *testing.T) {
	t.Parallel()

	safe := redactedEnvValue("ACP_QWEN_CMD", "/usr/local/bin/qwen")
	if safe["value"] != "/usr/local/bin/qwen" {
		t.Fatalf("expected safe provider command value to be visible, got %#v", safe)
	}
	secretLike := redactedEnvValue("ACP_QWEN_CMD", "qwen --api-key=super-secret")
	if _, ok := secretLike["value"]; ok {
		t.Fatalf("secret-like provider command value should not be exposed: %#v", secretLike)
	}
	if secretLike["present"] != true || secretLike["sha256"] == "" {
		t.Fatalf("expected presence/hash diagnostics for secret-like command value, got %#v", secretLike)
	}
}

func TestMergedCommandEnvAppliesOverrides(t *testing.T) {
	t.Parallel()

	env := mergedCommandEnv([]string{"A=old", "B=keep"}, map[string]string{"A": "new", "CODEX_HOME": "/tmp/codex-home"})
	got := map[string]string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			got[key] = value
		}
	}
	if got["A"] != "new" {
		t.Fatalf("expected A override, got %q from %v", got["A"], env)
	}
	if got["B"] != "keep" {
		t.Fatalf("expected existing B to remain, got %q from %v", got["B"], env)
	}
	if got["CODEX_HOME"] != "/tmp/codex-home" {
		t.Fatalf("expected CODEX_HOME override, got %q from %v", got["CODEX_HOME"], env)
	}
}

func TestForwardStreamOutputRedactsSecretLikeText(t *testing.T) {
	t.Parallel()

	var chunks []acpruntime.OutputChunk
	task := acpruntime.Task{
		OnOutput: func(chunk acpruntime.OutputChunk) {
			chunks = append(chunks, chunk)
		},
	}
	budget := &streamedOutputBudget{}

	forwardStreamOutput(task, acpruntime.OutputStreamStdout, "Authorization: Bearer stream-secret\n--token stream-token\nordinary text\n", budget)

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %#v", chunks)
	}
	combined := chunks[0].Text + "\n" + chunks[1].Text + "\n" + chunks[2].Text
	if strings.Contains(combined, "stream-secret") || strings.Contains(combined, "stream-token") {
		t.Fatalf("streamed output leaked secrets:\n%s", combined)
	}
	if !strings.Contains(combined, "ordinary text") {
		t.Fatalf("streamed output lost non-secret text:\n%s", combined)
	}
}

func TestForwardStreamOutputCountsRedactedBytesForCap(t *testing.T) {
	t.Parallel()

	var chunks []acpruntime.OutputChunk
	task := acpruntime.Task{
		OnOutput: func(chunk acpruntime.OutputChunk) {
			chunks = append(chunks, chunk)
		},
	}
	redactedLine := "--token <redacted>"
	budget := &streamedOutputBudget{
		forwardedBytes: acpruntime.RuntimeOutputStreamHardCapBytes - len([]byte(redactedLine)),
	}

	forwardStreamOutput(task, acpruntime.OutputStreamStdout, "--token "+strings.Repeat("s", 1024)+"\n", budget)

	if len(chunks) != 1 {
		t.Fatalf("expected one forwarded chunk, got %#v", chunks)
	}
	if chunks[0].Truncated {
		t.Fatalf("expected redacted line to fit output cap, got truncated chunk: %#v", chunks[0])
	}
	if chunks[0].Text != redactedLine {
		t.Fatalf("expected redacted line %q, got %q", redactedLine, chunks[0].Text)
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

func TestRunHeadlessProviderRecoversMissingCollectManifestWithAuthoredDocs(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair")
	repairMarker := filepath.Join(task.Workspace, "manifest-repair-called")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
`
	repairScript := `#!/usr/bin/env bash
set -eu
printf called > ` + shellQuote(repairMarker) + `
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
	if _, statErr := os.Stat(repairMarker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("manifest-only repair should not run when deterministic recovery can recover a missing manifest, statErr=%v", statErr)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatalf("read recovered manifest: %v", err)
	}
	if !strings.Contains(string(raw), "collect_manifest.runtime_recovery") {
		t.Fatalf("expected runtime recovery finding in manifest, got %s", raw)
	}
}

func TestRunHeadlessProviderStopsCollectManifestRepairAfterValidArtifact(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair-valid-stop")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
printf '%s\n' '{"version":1}' > ` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + `
`
	repairScript := `#!/usr/bin/env bash
set -eu
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + collectManifestJSON(task) + `
EOF
while true; do
  touch ` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + `
  sleep 0.02
done
`
	runner := testAdapter{
		command:       writeEngineScript(t, initialScript),
		repairCommand: writeEngineScript(t, repairScript),
		activity: ActivityPolicy{
			MonitorArtifacts:        true,
			ValidArtifactStopWindow: 50 * time.Millisecond,
			PollInterval:            5 * time.Millisecond,
			PostTerminateDrain:      10 * time.Millisecond,
			TerminateGrace:          10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectManifestOnce:     true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulCollectPairRecoveryTimeout)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected collect manifest repair to stop after valid artifact, got %v; snapshot=%+v", err, collectArtifactSnapshot(task.WriteRoot))
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderRecoversCollectManifestWhenProviderRepairDoesNotWriteManifest(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-runtime-recovery")
	repoRoot := filepath.Join(task.Workspace, "repos", "bank-of-anthos")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Bank of Anthos\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	task.ReadContextRoots = []string{repoRoot}
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
mkdir -p ` + shellQuote(filepath.Join(task.WriteRoot, "docs")) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + ` <<'EOF'
# Payment Runtime Overview

## Services
- **Ledger API**: records payment operations.
- **Transaction Worker**: processes queued account changes.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, "docs", "overview.md")) + ` <<'EOF'
# Runtime Dependency Overview

## Dependencies
- **Account Service**: provides account data.
EOF
`
	repairScript := `#!/usr/bin/env bash
set -eu
exit 0
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
		t.Fatalf("expected deterministic collect manifest recovery success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	if !hasRuntimeWarning(result.Execution.Warnings, "collect_manifest_runtime_recovery reconstructed shard-pack-manifest.json") {
		t.Fatalf("expected runtime recovery warning, got %#v", result.Execution.Warnings)
	}
	if recovery, ok := result.Diagnostics["collect_manifest_runtime_recovery"].(map[string]any); !ok || recovery["source"] != "runtime_recovery" || recovery["provider_authored"] != false {
		t.Fatalf("expected explicit runtime recovery diagnostics, got %#v", result.Diagnostics)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatalf("read recovered manifest: %v", err)
	}
	if !strings.Contains(string(raw), "collect_manifest.runtime_recovery") {
		t.Fatalf("expected runtime recovery finding in manifest, got %s", raw)
	}
	if !strings.Contains(string(raw), `"type": "uses"`) {
		t.Fatalf("expected non-container usage edges in recovered manifest, got %s", raw)
	}
	if strings.Contains(string(raw), `"Owner mapping not confirmed"`) {
		t.Fatalf("recovered manifest should not use bootstrap owner-mapping finding, got %s", raw)
	}
}

func TestRunHeadlessProviderRecoversScaffoldCollectManifestBeforeProviderRepair(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-scaffold-runtime-recovery")
	repairMarker := filepath.Join(task.Workspace, "scaffold-manifest-repair-called")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + ` <<'EOF'
# PostHog CLI Common Overview

## Runtime surfaces
- **PostHog CLI**: exposes developer commands for project operations.
- **HogQL parser**: parses query expressions used by product analytics flows.
- **reqwest API client**: calls the PostHog API for remote operations.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + scaffoldCollectManifestJSON(task) + `
EOF
`
	repairScript := `#!/usr/bin/env bash
set -eu
printf called > ` + shellQuote(repairMarker) + `
exit 7
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
		t.Fatalf("expected deterministic scaffold collect manifest recovery success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	if _, statErr := os.Stat(repairMarker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("manifest-only provider repair should not run for scaffold semantic recovery, statErr=%v", statErr)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatalf("read recovered manifest: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "collect_manifest.runtime_recovery") {
		t.Fatalf("expected runtime recovery finding in manifest, got %s", text)
	}
	if !strings.Contains(text, `"name": "PostHog CLI"`) || !strings.Contains(text, `"name": "HogQL parser"`) {
		t.Fatalf("expected recovered manifest to preserve concrete authored-doc concepts, got %s", text)
	}
	if !strings.Contains(text, `"type": "uses"`) {
		t.Fatalf("expected recovered manifest to include non-container usage edges, got %s", text)
	}
	if strings.Contains(text, "Owner mapping not confirmed") || strings.Contains(text, "contains scoped surface") {
		t.Fatalf("recovered manifest should not retain scaffold semantic content, got %s", text)
	}
}

func TestRunHeadlessProviderRepairsCollectManifestWithMissingReferencedDocumentPath(t *testing.T) {
	task := newCollectTask(t, "run-collect-repair-missing-doc-ref")
	badManifest := strings.Replace(collectManifestJSON(task), `"path": "overview.md"`, `"path": "overivew.md"`, 1)
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + badManifest + `
EOF
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
		t.Fatalf("expected collect manifest reference repair success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatalf("read repaired manifest: %v", err)
	}
	if strings.Contains(string(raw), "overivew.md") {
		t.Fatalf("expected repair manifest to replace missing document reference, got %s", raw)
	}
	if !strings.Contains(string(raw), `"path": "overview.md"`) {
		t.Fatalf("expected repaired manifest to reference authored document, got %s", raw)
	}
}

func TestRunHeadlessProviderRejectsStructuralInvalidCollectManifestAfterRepairExhaustion(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair-structural-invalid-exhausted")
	repoRoot := filepath.Join(task.Workspace, "repos", "bank-of-anthos")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Bank of Anthos\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	task.ReadContextRoots = []string{repoRoot}
	badManifest := strings.ReplaceAll(collectManifestJSON(task), `"path": "README.md"`, `"path": "missing-evidence.md"`)
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + badManifest + `
EOF
`
	repairScript := `#!/usr/bin/env bash
set -eu
exit 0
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
		t.Fatal("expected exhausted structural manifest repair to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "manifest-only collect repair did not produce valid collect artifacts") {
		t.Fatalf("expected manifest repair exhaustion to remain terminal, got %v", err)
	}
	if !hasDiagnosticField(diagnostics, "collect manifest repair exhausted", "recovery_mode", "collect_manifest_repair") {
		t.Fatalf("expected collect manifest repair exhausted diagnostic, got %#v", diagnostics)
	}
	if hasDiagnosticField(diagnostics, "collect manifest runtime recovery completed", "recovery_mode", "collect_manifest_runtime_recovery") {
		t.Fatalf("structural-invalid manifest repair exhaustion must not be converted into deterministic recovery success: %#v", diagnostics)
	}
}

func TestRunHeadlessProviderEscalatesMissingRepoEvidenceInMarkdownToPairRepair(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-missing-repo-evidence-pair-repair")
	repoRoot := filepath.Join(task.Workspace, "repos", "bank-of-anthos")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Bank of Anthos\n\nRuntime entrypoint.\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	task.ReadContextRoots = []string{repoRoot}
	badManifest := strings.ReplaceAll(collectManifestJSON(task), `"path": "README.md"`, `"path": "missing-evidence.md"`)
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + ` <<'EOF'
# Collect Overview

## Evidence
- missing-evidence.md is the configured runtime evidence file.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + badManifest + `
EOF
`
	manifestOnlyRepairScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' 'manifest-only repair must not run when markdown cites missing repo evidence' >&2
exit 9
`
	pairRepairScript := `#!/usr/bin/env bash
set -eu
cat >` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + ` <<'EOF'
# Collect Overview

## Evidence
- README.md identifies Bank of Anthos as the assigned runtime surface.

## Gap
- The previous missing-evidence.md reference was not present in the repo and is excluded from citations.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + collectManifestJSON(task) + `
EOF
`
	runner := testAdapter{
		command:           writeEngineScript(t, initialScript),
		repairCommand:     writeEngineScript(t, manifestOnlyRepairScript),
		pairRepairCommand: writeEngineScript(t, pairRepairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectManifestOnce:     true,
			RepairCollectArtifactPairOnce: true,
		},
	}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("expected missing repo evidence markdown to be repaired as a pair, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	if !hasDiagnosticField(diagnostics, "focused artifact repair scheduled", "recovery_mode", "collect_pair_repair") {
		t.Fatalf("expected collect_pair_repair scheduled diagnostic, got %#v", diagnostics)
	}
	if hasDiagnosticField(diagnostics, "collect manifest repair scheduled", "recovery_mode", "collect_manifest_repair") {
		t.Fatalf("manifest-only repair must not run for markdown with missing repo evidence claims: %#v", diagnostics)
	}
	docRaw, err := os.ReadFile(filepath.Join(task.WriteRoot, "overview.md"))
	if err != nil {
		t.Fatalf("read repaired doc: %v", err)
	}
	if !strings.Contains(string(docRaw), "README.md identifies Bank of Anthos") {
		t.Fatalf("expected pair repair to rewrite authored markdown, got:\n%s", docRaw)
	}
}

func TestRunHeadlessProviderRejectsMissingRepoEvidencePairRepairNoopMarkdown(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-missing-repo-evidence-pair-noop")
	repoRoot := filepath.Join(task.Workspace, "repos", "bank-of-anthos")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Bank of Anthos\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	task.ReadContextRoots = []string{repoRoot}
	badManifest := strings.ReplaceAll(collectManifestJSON(task), `"path": "README.md"`, `"path": "missing-evidence.md"`)
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + ` <<'EOF'
# Collect Overview

## Evidence
- missing-evidence.md is the configured runtime evidence file.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, "aaa-general.md")) + ` <<'EOF'
# General Note

README.md exists but this file did not contain the stale missing evidence claim.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + badManifest + `
EOF
`
	pairRepairScript := `#!/usr/bin/env bash
set -eu
cat >` + shellQuote(filepath.Join(task.WriteRoot, "aaa-general.md")) + ` <<'EOF'
# General Note

README.md was rewritten, but the stale overview.md claim remains untouched.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + collectManifestJSON(task) + `
EOF
`
	runner := testAdapter{
		command:           writeEngineScript(t, initialScript),
		pairRepairCommand: writeEngineScript(t, pairRepairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectManifestOnce:     true,
			RepairCollectArtifactPairOnce: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected pair repair that leaves stale markdown unchanged to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "collect_pair_repair_noop_or_stale_markdown") {
		t.Fatalf("expected stale markdown repair failure, got %v", err)
	}
}

func TestRunHeadlessProviderRejectsBootstrapOnlyAuthoredDocWithoutPairRepair(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-bootstrap-doc-pair-repair")
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	manifest := strings.Replace(collectManifestJSON(task), `"path": "overview.md"`, `"path": "`+docRel+`"`, 1)
	docPath := filepath.Join(task.WriteRoot, filepath.FromSlash(docRel))
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(filepath.Dir(docPath)) + `
cat >` + shellQuote(docPath) + ` <<'EOF'
# Bank Overview

<!-- ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT -->

## Observations
- Repository scope: bank-of-anthos.
- Primary scoped evidence path: ` + "`README.md`" + `.

## Evidence
- Primary evidence path: ` + "`README.md`" + `

## Follow-up
- Owner mapping evidence not confirmed from the initial scoped evidence path.
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + manifest + `
EOF
`
	manifestOnlyRepairScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' 'manifest-only repair must not run for bootstrap-only docs' >&2
exit 9
`
	pairRepairScript := `#!/usr/bin/env bash
set -eu
cat >` + shellQuote(docPath) + ` <<'EOF'
# Bank Overview

## Observations
- README.md identifies Bank of Anthos as the analyzed service surface.
- Kubernetes manifests define the deployable service boundary for review.

## Evidence
- README.md
- kubernetes-manifests
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + manifest + `
EOF
`
	runner := testAdapter{
		command:           writeEngineScript(t, initialScript),
		repairCommand:     writeEngineScript(t, manifestOnlyRepairScript),
		pairRepairCommand: writeEngineScript(t, pairRepairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectManifestOnce:     true,
			RepairCollectArtifactPairOnce: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatalf("expected bootstrap-only authored collect doc to fail without pair repair")
	}
	if !strings.Contains(err.Error(), "bootstrap-only collect document") {
		t.Fatalf("expected bootstrap-only validation error, got %v", err)
	}
	raw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read original doc: %v", err)
	}
	if !strings.Contains(string(raw), "ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT") {
		t.Fatalf("expected bootstrap doc marker to remain because pair repair must not run, got:\n%s", raw)
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
			PostArtifactStallWindow:     successfulArtifactWriteWindow,
			PartialArtifactStallWindow:  successfulArtifactWriteWindow,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectArtifactPairOnce: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulCollectPairRecoveryTimeout)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected collect pair repair success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
}

func TestRunHeadlessProviderRetriesTransientAPIErrorCollectPairRepair(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-pair-repair-api-retry")
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	manifest := strings.Replace(collectManifestJSON(task), `"path": "overview.md"`, `"path": "`+docRel+`"`, 1)
	statePath := filepath.Join(task.Workspace, "repair-attempt-state")
	initialScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' 'qwen collect started'
`
	docAbs := filepath.Join(task.WriteRoot, filepath.FromSlash(docRel))
	repairScript := `#!/usr/bin/env bash
set -eu
state=` + shellQuote(statePath) + `
if [[ ! -f "$state" ]]; then
  printf '%s\n' 'first' > "$state"
  printf '%s\n' '[API Error: Premature close]'
  exit 0
fi
mkdir -p ` + shellQuote(filepath.Dir(docAbs)) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(docAbs) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + manifest + `
EOF
`
	runner := testAdapter{
		command:           writeEngineScript(t, initialScript),
		pairRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:               true,
			RepairCollectArtifactPairOnce:               true,
			RetryTransientProviderUnavailableRepairOnce: true,
		},
	}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("expected collect pair repair retry success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	if _, err := os.Stat(filepath.Join(task.WriteRoot, ShardPackManifestFileName)); err != nil {
		t.Fatalf("expected repaired manifest: %v", err)
	}
	if _, err := os.Stat(docAbs); err != nil {
		t.Fatalf("expected repaired document: %v", err)
	}
}

func TestRunHeadlessProviderClassifiesExhaustedTransientAPIErrorCollectPairRepairUnavailable(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-pair-repair-api-exhausted")
	initialScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' 'qwen collect started'
`
	repairScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' '[API Error: Premature close]'
`
	runner := testAdapter{
		command:           writeEngineScript(t, initialScript),
		pairRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:               true,
			RepairCollectArtifactPairOnce:               true,
			RetryTransientProviderUnavailableRepairOnce: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected exhausted provider API repair to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Stdout, "Premature close") {
		t.Fatalf("expected provider API error diagnostics, got %q", runnerErr.Stdout)
	}
	if strings.TrimSpace(runnerErr.RawOutputRefs.Stdout) == "" || strings.TrimSpace(runnerErr.RawOutputRefs.Metadata) == "" {
		t.Fatalf("expected raw output refs, got %+v", runnerErr.RawOutputRefs)
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
			PostArtifactStallWindow:     successfulArtifactWriteWindow,
			PartialArtifactStallWindow:  successfulArtifactWriteWindow,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairCollectArtifactPairOnce: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulCollectPairRecoveryTimeout)
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

func TestRunHeadlessProviderRepairsSilentRetryExhaustedCollectWithPairRepair(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-pair-repair-silent-retry")
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	manifest := strings.Replace(collectManifestJSON(task), `"path": "overview.md"`, `"path": "`+docRel+`"`, 1)
	docAbs := filepath.Join(task.WriteRoot, filepath.FromSlash(docRel))
	repairScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(filepath.Dir(docAbs)) + `
cat >` + shellQuote(docAbs) + ` <<'EOF'
# Collect Overview

## Observations
- README.md identifies the assigned collect surface for this shard.
- Deployment and source files should be reviewed before downstream recommendations.

## Evidence
- README.md
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + manifest + `
EOF
`
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	runner := testAdapter{
		command:           writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 5\n"),
		pairRepairCommand: writeEngineScript(t, repairScript),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      20 * time.Millisecond,
			RetryPreArtifactStallWindow: 20 * time.Millisecond,
			PostArtifactStallWindow:     successfulArtifactWriteWindow,
			PartialArtifactStallWindow:  successfulArtifactWriteWindow,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RepairCollectArtifactPairOnce:            true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			RetryZeroOutputPreArtifactStallOnce:      true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulCollectPairRecoveryTimeout)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected silent collect retry exhaustion to run collect pair repair, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	if !hasDiagnosticField(diagnostics, "focused artifact repair scheduled", "recovery_mode", "collect_pair_repair") {
		t.Fatalf("expected collect_pair_repair scheduled diagnostic, got %#v", diagnostics)
	}
	if !hasDiagnosticField(diagnostics, "focused artifact repair completed", "recovery_mode", "collect_pair_repair") {
		t.Fatalf("expected collect_pair_repair completed diagnostic, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderFailsSilentCollectWhenPairRepairProducesNoArtifacts(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-pair-repair-silent-empty")
	runner := testAdapter{
		command:           writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 5\n"),
		pairRepairCommand: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
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
			RetryZeroOutputPreArtifactStallOnce:      true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected empty collect pair repair to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "collect_pair_repair") {
		t.Fatalf("expected collect_pair_repair failure, got %v", err)
	}
}

func TestRunHeadlessProviderRecoversManifestAfterPartialPairRepairStall(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-pair-partial-manifest-recovery")
	docRel := steppolicy.SuggestedCollectDocumentPath(task)
	docAbs := filepath.Join(task.WriteRoot, filepath.FromSlash(docRel))
	repairScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(filepath.Dir(docAbs)) + `
cat >` + shellQuote(docAbs) + ` <<'EOF'
# Collect Repair Overview

## Runtime Surfaces
- **Payments API** exposes request handlers for account activity.
- **Ledger Worker** persists balance changes.
- **Postgres** stores transaction state.

## Operational Gaps
- Ownership and escalation are not confirmed in this shard.
EOF
sleep 5
`
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	runner := testAdapter{
		command:           writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 5\n"),
		pairRepairCommand: writeEngineScript(t, repairScript),
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
			RepairCollectManifestOnce:                true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			RetryZeroOutputPreArtifactStallOnce:      true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulCollectPairRecoveryTimeout)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected partial pair repair to recover missing manifest, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	if !hasDiagnosticField(diagnostics, "focused artifact repair exhausted", "recovery_mode", "collect_pair_repair") {
		t.Fatalf("expected collect_pair_repair exhausted diagnostic, got %#v", diagnostics)
	}
	if !hasDiagnosticField(diagnostics, "collect manifest runtime recovery completed", "recovery_mode", "collect_manifest_runtime_recovery") {
		t.Fatalf("expected runtime manifest recovery diagnostic, got %#v", diagnostics)
	}
	if !hasRuntimeWarning(result.Execution.Warnings, "collect_manifest_runtime_recovery reconstructed shard-pack-manifest.json") {
		t.Fatalf("expected runtime recovery warning, got %#v", result.Execution.Warnings)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatalf("read recovered manifest: %v", err)
	}
	if !strings.Contains(string(raw), "collect_manifest.runtime_recovery") {
		t.Fatalf("expected runtime recovery finding in manifest, got %s", raw)
	}
}

func TestRunHeadlessProviderRecoversMissingCollectManifestWithNestedAuthoredDocs(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair-nested")
	nestedDocPath := filepath.Join(task.WriteRoot, "docs", "overview.md")
	repairMarker := filepath.Join(task.Workspace, "nested-manifest-repair-called")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(filepath.Dir(nestedDocPath)) + `
printf '%s\n' '# Nested Collect Overview' > ` + shellQuote(nestedDocPath) + `
`
	repairScript := `#!/usr/bin/env bash
set -eu
printf called > ` + shellQuote(repairMarker) + `
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
	if _, statErr := os.Stat(repairMarker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("manifest-only repair should not run for missing nested-doc manifest recovery, statErr=%v", statErr)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatalf("read recovered manifest: %v", err)
	}
	if !strings.Contains(string(raw), `"path": "docs/overview.md"`) {
		t.Fatalf("expected recovered manifest to reference nested authored document, got %s", raw)
	}
}

func TestRunHeadlessProviderRejectsCollectRepairExtraWrites(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-repair-extra-write")
	initialScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
printf '%s\n' '# Collect Overview' > ` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + `
printf '%s\n' '{"version":1}' > ` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + `
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
printf '%s\n' '{"version":1}' > ` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + `
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

func TestRunHeadlessProviderRecoversQwenPartialCollectArtifactsWithAuthoredDocs(t *testing.T) {
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

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("expected runtime collect manifest recovery success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	if !hasRuntimeWarning(result.Execution.Warnings, "collect_manifest_runtime_recovery reconstructed shard-pack-manifest.json") {
		t.Fatalf("expected runtime recovery warning, got %#v", result.Execution.Warnings)
	}
	raw, err := os.ReadFile(filepath.Join(task.WriteRoot, ShardPackManifestFileName))
	if err != nil {
		t.Fatalf("read recovered manifest: %v", err)
	}
	if !strings.Contains(string(raw), "runtime_recovery") {
		t.Fatalf("expected recovered manifest claim/finding marker, got %s", raw)
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
		command:                writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nfor i in 1 2 3 4 5; do printf '%s\\n' \"validator diagnostic before stall $i\"; sleep 0.05; done\nsleep 5\n"),
		validatorRepairCommand: writeEngineScript(t, repairScript),
		activity: ActivityPolicy{
			MonitorArtifacts:        true,
			MonitorPreArtifact:      true,
			PreArtifactStallWindow:  100 * time.Millisecond,
			PostArtifactStallWindow: 20 * time.Millisecond,
			PollInterval:            5 * time.Millisecond,
			PostTerminateDrain:      10 * time.Millisecond,
			TerminateGrace:          10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:      true,
			RepairValidatorVerdictOnce:         true,
			RetryInvalidOrMissingArtifactsOnce: true,
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

func TestRunHeadlessProviderRetriesZeroOutputValidatorStallWhenPolicyAllows(t *testing.T) {
	task := newValidatorTask(t, "run-validator-silent-then-success")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	verdictScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, ValidatorVerdictFileName)) + ` <<'EOF'
` + validatorVerdictJSON(task) + `
EOF
`
	runner := &sequenceAdapter{
		testAdapter: testAdapter{
			activity: ActivityPolicy{
				MonitorArtifacts:            true,
				MonitorPreArtifact:          true,
				PreArtifactStallWindow:      20 * time.Millisecond,
				RetryPreArtifactStallWindow: successfulZeroOutputRetryWindow,
				PostArtifactStallWindow:     successfulArtifactWriteWindow,
				PollInterval:                5 * time.Millisecond,
				PostTerminateDrain:          10 * time.Millisecond,
				TerminateGrace:              10 * time.Millisecond,
			},
			recovery: RecoveryPolicy{
				AcceptValidArtifactsAfterStop:            true,
				RetryInvalidOrMissingArtifactsOnce:       true,
				RetryZeroOutputPreArtifactStallOnce:      true,
				ClassifySilentRetryExhaustionUnavailable: true,
			},
		},
		commands: []string{
			writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
			writeEngineScript(t, verdictScript),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulZeroOutputRetryTimeout)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected validator retry to recover valid artifacts, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("expected succeeded execution, got %+v", result.Execution)
	}
	if !hasDiagnostic(diagnostics, "zero-output pre-artifact stall will retry", "warning") {
		t.Fatalf("expected warning diagnostic for recovered validator zero-output stall, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderRetriesZeroOutputCollectStallWhenPolicyAllows(t *testing.T) {
	task := newCollectTask(t, "run-collect-silent-then-success")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	collectScript := `#!/usr/bin/env bash
set -eu
mkdir -p ` + shellQuote(task.WriteRoot) + `
cat >` + shellQuote(filepath.Join(task.WriteRoot, "overview.md")) + ` <<'EOF'
# Bank Overview
EOF
cat >` + shellQuote(filepath.Join(task.WriteRoot, ShardPackManifestFileName)) + ` <<'EOF'
` + collectManifestJSON(task) + `
EOF
`
	runner := &sequenceAdapter{
		testAdapter: testAdapter{
			activity: ActivityPolicy{
				MonitorArtifacts:            true,
				MonitorPreArtifact:          true,
				PreArtifactStallWindow:      20 * time.Millisecond,
				RetryPreArtifactStallWindow: successfulZeroOutputRetryWindow,
				PostArtifactStallWindow:     20 * time.Millisecond,
				PollInterval:                5 * time.Millisecond,
				PostTerminateDrain:          10 * time.Millisecond,
				TerminateGrace:              10 * time.Millisecond,
			},
			recovery: RecoveryPolicy{
				AcceptValidArtifactsAfterStop:            true,
				RepairCollectManifestOnce:                true,
				RepairCollectArtifactPairOnce:            true,
				RetryInvalidOrMissingArtifactsOnce:       true,
				RetryZeroOutputPreArtifactStallOnce:      true,
				ClassifySilentRetryExhaustionUnavailable: true,
			},
		},
		commands: []string{
			writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
			writeEngineScript(t, collectScript),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulZeroOutputRetryTimeout)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected collect retry to recover valid artifacts, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("expected succeeded execution, got %+v", result.Execution)
	}
	if !hasDiagnostic(diagnostics, "zero-output pre-artifact stall will retry", "warning") {
		t.Fatalf("expected warning diagnostic for recovered collect zero-output stall, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderKeepsExhaustedZeroOutputCollectRetryUnavailable(t *testing.T) {
	t.Parallel()

	task := newCollectTask(t, "run-collect-silent-retry-exhausted")
	runner := testAdapter{
		command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      20 * time.Millisecond,
			RetryPreArtifactStallWindow: 20 * time.Millisecond,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RepairCollectManifestOnce:                true,
			RepairCollectArtifactPairOnce:            true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			RetryZeroOutputPreArtifactStallOnce:      true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected exhausted collect zero-output retry to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
}

func TestRunHeadlessProviderKeepsExhaustedZeroOutputValidatorRetryUnavailable(t *testing.T) {
	t.Parallel()

	task := newValidatorTask(t, "run-validator-silent-retry-exhausted")
	runner := testAdapter{
		command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      20 * time.Millisecond,
			RetryPreArtifactStallWindow: 20 * time.Millisecond,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			RetryZeroOutputPreArtifactStallOnce:      true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected exhausted validator zero-output retry to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
}

func TestRunHeadlessProviderKeepsInvalidValidatorAfterZeroOutputRetryAsContractFailure(t *testing.T) {
	t.Parallel()

	task := newValidatorTask(t, "run-validator-silent-then-invalid")
	invalidVerdictScript := `#!/usr/bin/env bash
set -eu
write_root=` + shellQuote(task.WriteRoot) + `
mkdir -p "$write_root"
cat >"$write_root/` + ValidatorVerdictFileName + `" <<'EOF'
{"version":1,"run_id":"` + task.RunID + `","generated_at":"2026-04-16T12:00:02Z","verdict":"FAIL","checked_paths":["reports/taskruns/` + task.RunID + `/staging/final/final-run-index.json"],"issues":[{"id":"legacy","severity":"high","title":"Legacy issue","description":"Wrong shape"}]}
EOF
printf '%s\n' 'wrote invalid validator verdict'
`
	runner := &sequenceAdapter{
		testAdapter: testAdapter{
			activity: ActivityPolicy{
				MonitorArtifacts:            true,
				MonitorPreArtifact:          true,
				PreArtifactStallWindow:      20 * time.Millisecond,
				RetryPreArtifactStallWindow: successfulZeroOutputRetryWindow,
				PostArtifactStallWindow:     20 * time.Millisecond,
				PollInterval:                5 * time.Millisecond,
				PostTerminateDrain:          10 * time.Millisecond,
				TerminateGrace:              10 * time.Millisecond,
			},
			recovery: RecoveryPolicy{
				AcceptValidArtifactsAfterStop:            true,
				RetryInvalidOrMissingArtifactsOnce:       true,
				RetryZeroOutputPreArtifactStallOnce:      true,
				ClassifySilentRetryExhaustionUnavailable: true,
			},
		},
		commands: []string{
			writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
			writeEngineScript(t, invalidVerdictScript),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulZeroOutputRetryTimeout)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected invalid validator retry artifact to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
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

func TestRunHeadlessProviderRepairsAsIsDraftArtifactsWithAllDraftFiles(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-repair")
	repairScript := asIsDraftScript(task, []string{"overview.md", "summary.md", "architect-summary.md"}, "exit 0")
	runner := testAdapter{
		command:            writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		draftRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop: true,
			RepairDraftArtifactsOnce:      true,
		},
	}

	if _, err := RunHeadlessProvider(context.Background(), task, runner); err != nil {
		t.Fatalf("expected as-is draft artifact repair success, got %v", err)
	}
}

func TestRunHeadlessProviderEnrichesBootstrapOnlyDraftAfterRepairStall(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-enrichment-after-stall")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	repairScript := asIsBootstrapDraftScript(task, "sleep 5")
	enrichmentScript := asIsDraftScript(task, []string{"overview.md", "summary.md", "architect-summary.md"}, "exit 0")
	runner := testAdapter{
		command:                writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		draftRepairCommand:     writeEngineScript(t, repairScript),
		draftEnrichmentCommand: writeEngineScript(t, enrichmentScript),
		activity: ActivityPolicy{
			MonitorArtifacts:           true,
			MonitorPreArtifact:         true,
			PreArtifactStallWindow:     50 * time.Millisecond,
			PostArtifactStallWindow:    50 * time.Millisecond,
			PartialArtifactStallWindow: 50 * time.Millisecond,
			PollInterval:               10 * time.Millisecond,
			TerminateGrace:             10 * time.Millisecond,
			PostTerminateDrain:         time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:     true,
			RepairDraftArtifactsOnce:          true,
			RepairDraftArtifactEnrichmentOnce: true,
		},
	}

	if _, err := RunHeadlessProvider(context.Background(), task, runner); err != nil {
		t.Fatalf("expected draft enrichment success after scaffold repair stall, got %v", err)
	}
	if !hasDiagnosticField(diagnostics, "focused artifact repair scheduled", "recovery_mode", "draft_artifact_enrichment") {
		t.Fatalf("expected draft_artifact_enrichment scheduled diagnostic, got %#v", diagnostics)
	}
	if !hasDiagnosticField(diagnostics, "focused artifact repair completed", "recovery_mode", "draft_artifact_enrichment") {
		t.Fatalf("expected draft_artifact_enrichment completed diagnostic, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderSkipsDraftRepairForBootstrapOnlyDraft(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-enrichment-direct")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	enrichmentScript := asIsDraftScript(task, []string{"overview.md", "summary.md", "architect-summary.md"}, "exit 0")
	runner := testAdapter{
		command:                writeEngineScript(t, asIsBootstrapDraftScript(task, "exit 0")),
		draftEnrichmentCommand: writeEngineScript(t, enrichmentScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:     true,
			RepairDraftArtifactsOnce:          true,
			RepairDraftArtifactEnrichmentOnce: true,
		},
	}

	if _, err := RunHeadlessProvider(context.Background(), task, runner); err != nil {
		t.Fatalf("expected direct draft enrichment success for bootstrap-only draft, got %v", err)
	}
	if hasDiagnosticField(diagnostics, "focused artifact repair scheduled", "recovery_mode", "draft_artifact_repair") {
		t.Fatalf("bootstrap-only draft validation must skip scaffold draft repair, got %#v", diagnostics)
	}
	if !hasDiagnosticField(diagnostics, "focused artifact repair scheduled", "recovery_mode", "draft_artifact_enrichment") {
		t.Fatalf("expected draft_artifact_enrichment scheduled diagnostic, got %#v", diagnostics)
	}
	if !hasDiagnosticField(diagnostics, "focused artifact repair completed", "recovery_mode", "draft_artifact_enrichment") {
		t.Fatalf("expected draft_artifact_enrichment completed diagnostic, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderRejectsBootstrapOnlyDraftAfterEnrichment(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-enrichment-invalid")
	repairScript := asIsBootstrapDraftScript(task, "exit 0")
	enrichmentScript := asIsBootstrapDraftScript(task, "exit 0")
	runner := testAdapter{
		command:                writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		draftRepairCommand:     writeEngineScript(t, repairScript),
		draftEnrichmentCommand: writeEngineScript(t, enrichmentScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:     true,
			RepairDraftArtifactsOnce:          true,
			RepairDraftArtifactEnrichmentOnce: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected bootstrap-only draft enrichment to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "draft_artifact_enrichment_noop_or_scaffold") {
		t.Fatalf("expected enrichment failure, got %v", err)
	}
}

func TestRunHeadlessProviderRejectsStalledBootstrapOnlyDraftAfterEnrichment(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-enrichment-stalled-scaffold")
	repairScript := asIsBootstrapDraftScript(task, "exit 0")
	enrichmentScript := asIsBootstrapDraftScript(task, "sleep 5")
	runner := testAdapter{
		command:                writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		draftRepairCommand:     writeEngineScript(t, repairScript),
		draftEnrichmentCommand: writeEngineScript(t, enrichmentScript),
		activity: ActivityPolicy{
			MonitorArtifacts:           true,
			MonitorPreArtifact:         true,
			PreArtifactStallWindow:     50 * time.Millisecond,
			PostArtifactStallWindow:    50 * time.Millisecond,
			PartialArtifactStallWindow: 50 * time.Millisecond,
			PollInterval:               10 * time.Millisecond,
			TerminateGrace:             10 * time.Millisecond,
			PostTerminateDrain:         time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:     true,
			RepairDraftArtifactsOnce:          true,
			RepairDraftArtifactEnrichmentOnce: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected stalled bootstrap-only draft enrichment to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "draft_artifact_enrichment_noop_or_scaffold") {
		t.Fatalf("expected noop/scaffold enrichment failure, got %v", err)
	}
}

func TestRunHeadlessProviderRejectsDraftEnrichmentExtraWriteRootFiles(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-enrichment-extra-write")
	repairScript := asIsBootstrapDraftScript(task, "exit 0")
	enrichmentScript := asIsDraftScript(
		task,
		[]string{"overview.md", "summary.md", "architect-summary.md"},
		"printf '%s\\n' '# Extra' > "+shellQuote(filepath.Join(task.WriteRoot, "extra.md"))+"\n",
	)
	runner := testAdapter{
		command:                writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		draftRepairCommand:     writeEngineScript(t, repairScript),
		draftEnrichmentCommand: writeEngineScript(t, enrichmentScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:     true,
			RepairDraftArtifactsOnce:          true,
			RepairDraftArtifactEnrichmentOnce: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected draft enrichment write-set contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "draft enrichment wrote outside the draft artifact write set") {
		t.Fatalf("expected draft enrichment write-set failure, got %v", err)
	}
}

func TestRunHeadlessProviderRejectsDraftEnrichmentUnreferencedDraftRootFiles(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-enrichment-extra-draft-root")
	repairScript := asIsBootstrapDraftScript(task, "exit 0")
	enrichmentScript := asIsDraftScript(
		task,
		[]string{"overview.md", "summary.md", "architect-summary.md"},
		"printf '%s\\n' '# Extra' > "+shellQuote(filepath.Join(task.DraftFinalRoot, "extra.md"))+"\n",
	)
	runner := testAdapter{
		command:                writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nexit 0\n"),
		draftRepairCommand:     writeEngineScript(t, repairScript),
		draftEnrichmentCommand: writeEngineScript(t, enrichmentScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:     true,
			RepairDraftArtifactsOnce:          true,
			RepairDraftArtifactEnrichmentOnce: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected draft enrichment write-set contract failure")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "draft enrichment wrote outside the draft artifact write set") {
		t.Fatalf("expected draft enrichment write-set failure, got %v", err)
	}
	if !strings.Contains(runnerErr.Error(), "forbidden draft_final_root files") {
		t.Fatalf("expected draft_final_root write-set detail, got %v", err)
	}
}

func TestRunHeadlessProviderRetriesTransientAPIErrorDraftArtifactRepair(t *testing.T) {
	task := newAsIsDraftTask(t, "run-asis-draft-repair-api-retry")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	statePath := filepath.Join(task.Workspace, "draft-repair-attempt-state")
	initialScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' 'qwen as-is started'
`
	successScript := asIsDraftScript(task, []string{"overview.md", "summary.md", "architect-summary.md"}, "exit 0")
	repairScript := strings.Replace(
		successScript,
		"#!/usr/bin/env bash\nset -eu\n",
		"#!/usr/bin/env bash\nset -eu\nstate="+shellQuote(statePath)+"\nif [[ ! -f \"$state\" ]]; then\n  printf '%s\\n' 'first' > \"$state\"\n  printf '%s\\n' '[API Error: Connection error. (cause: request to https://api.kimi.com/coding/v1/messages failed, reason: Client network socket disconnected before secure TLS connection was established)]'\n  exit 0\nfi\n",
		1,
	)
	runner := testAdapter{
		command:            writeEngineScript(t, initialScript),
		draftRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:               true,
			RepairDraftArtifactsOnce:                    true,
			RetryTransientProviderUnavailableRepairOnce: true,
		},
	}

	result, err := RunHeadlessProvider(context.Background(), task, runner)
	if err != nil {
		t.Fatalf("expected draft artifact repair retry success, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("unexpected execution status: %+v", result.Execution)
	}
	for _, name := range []string{"overview.md", "summary.md", "architect-summary.md"} {
		if _, err := os.Stat(filepath.Join(task.DraftFinalRoot, name)); err != nil {
			t.Fatalf("expected repaired draft file %s: %v", name, err)
		}
	}
	if !hasDiagnostic(diagnostics, "focused artifact repair retry scheduled", "warning") {
		t.Fatalf("expected warning diagnostic for transient draft repair retry, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderClassifiesExhaustedTransientAPIErrorDraftArtifactRepairUnavailable(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-repair-api-exhausted")
	initialScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' 'qwen as-is started'
`
	repairScript := `#!/usr/bin/env bash
set -eu
printf '%s\n' '[API Error: Connection error. (cause: request to https://api.kimi.com/coding/v1/messages failed, reason: Client network socket disconnected before secure TLS connection was established)]'
`
	runner := testAdapter{
		command:            writeEngineScript(t, initialScript),
		draftRepairCommand: writeEngineScript(t, repairScript),
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:               true,
			RepairDraftArtifactsOnce:                    true,
			RetryTransientProviderUnavailableRepairOnce: true,
		},
	}

	_, err := RunHeadlessProvider(context.Background(), task, runner)
	if err == nil {
		t.Fatal("expected exhausted provider API draft repair to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Stdout, "Connection error") {
		t.Fatalf("expected provider API error diagnostics, got %q", runnerErr.Stdout)
	}
	if strings.TrimSpace(runnerErr.RawOutputRefs.Stdout) == "" || strings.TrimSpace(runnerErr.RawOutputRefs.Metadata) == "" {
		t.Fatalf("expected raw output refs, got %+v", runnerErr.RawOutputRefs)
	}
}

func TestRunHeadlessProviderRejectsAsIsDraftRepairMissingReferencedFiles(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-asis-draft-repair-missing")
	repairScript := asIsDraftScript(task, []string{"summary.md"}, "exit 0")
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
		t.Fatal("expected as-is draft repair missing referenced files to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
	}
	if !strings.Contains(runnerErr.Error(), "referenced draft file") || !strings.Contains(runnerErr.Error(), "overview.md") {
		t.Fatalf("expected missing referenced draft file details, got %v", err)
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

func TestRunHeadlessProviderRetriesZeroOutputProposalsStallWhenPolicyAllows(t *testing.T) {
	task := newProposalsDraftTask(t, "run-proposals-silent-then-success")
	diagnostics := []acpruntime.DiagnosticEvent{}
	task.OnDiagnostic = func(event acpruntime.DiagnosticEvent) {
		diagnostics = append(diagnostics, event)
	}
	runner := &sequenceAdapter{
		testAdapter: testAdapter{
			activity: ActivityPolicy{
				MonitorArtifacts:            true,
				MonitorPreArtifact:          true,
				PreArtifactStallWindow:      20 * time.Millisecond,
				RetryPreArtifactStallWindow: successfulZeroOutputRetryWindow,
				PostArtifactStallWindow:     successfulArtifactWriteWindow,
				PollInterval:                5 * time.Millisecond,
				PostTerminateDrain:          10 * time.Millisecond,
				TerminateGrace:              10 * time.Millisecond,
			},
			recovery: RecoveryPolicy{
				AcceptValidArtifactsAfterStop:            true,
				RetryInvalidOrMissingArtifactsOnce:       true,
				RetryZeroOutputPreArtifactStallOnce:      true,
				ClassifySilentRetryExhaustionUnavailable: true,
			},
		},
		commands: []string{
			writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
			writeEngineScript(t, proposalsDraftScript(task, "exit 0")),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulZeroOutputRetryTimeout)
	defer cancel()
	result, err := RunHeadlessProvider(ctx, task, runner)
	if err != nil {
		t.Fatalf("expected proposals retry to recover valid artifacts, got %v", err)
	}
	if result.Execution.Status != "succeeded" {
		t.Fatalf("expected succeeded execution, got %+v", result.Execution)
	}
	if !hasDiagnostic(diagnostics, "zero-output pre-artifact stall will retry", "warning") {
		t.Fatalf("expected warning diagnostic for recovered proposals zero-output stall, got %#v", diagnostics)
	}
}

func TestRunHeadlessProviderKeepsExhaustedZeroOutputProposalsRetryUnavailable(t *testing.T) {
	t.Parallel()

	task := newProposalsDraftTask(t, "run-proposals-silent-retry-exhausted")
	runner := testAdapter{
		command: writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
		activity: ActivityPolicy{
			MonitorArtifacts:            true,
			MonitorPreArtifact:          true,
			PreArtifactStallWindow:      20 * time.Millisecond,
			RetryPreArtifactStallWindow: 20 * time.Millisecond,
			PostArtifactStallWindow:     20 * time.Millisecond,
			PollInterval:                5 * time.Millisecond,
			PostTerminateDrain:          10 * time.Millisecond,
			TerminateGrace:              10 * time.Millisecond,
		},
		recovery: RecoveryPolicy{
			AcceptValidArtifactsAfterStop:            true,
			RetryInvalidOrMissingArtifactsOnce:       true,
			RetryZeroOutputPreArtifactStallOnce:      true,
			ClassifySilentRetryExhaustionUnavailable: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected exhausted proposals zero-output retry to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerUnavailable {
		t.Fatalf("expected runner_unavailable, got %s (%v)", runnerErr.Code, err)
	}
}

func TestRunHeadlessProviderKeepsInvalidProposalsAfterZeroOutputRetryAsContractFailure(t *testing.T) {
	t.Parallel()

	task := newProposalsDraftTask(t, "run-proposals-silent-then-invalid")
	invalidDraftScript := `#!/usr/bin/env bash
set -eu
write_root=` + shellQuote(task.WriteRoot) + `
draft_root=` + shellQuote(task.DraftFinalRoot) + `
mkdir -p "$write_root" "$draft_root"
cat >"$write_root/proposals-draft-manifest.json" <<'EOF'
{"version":1,"run_id":"` + task.RunID + `","step_id":"init.step4.proposals","step_contract":"proposals","agent_role":"architect","outputs":[{"path":"missing.md","canonical_path":"proposals/runtime-recommendations.md","kind":"proposal","title":"Missing"}]}
EOF
printf '%s\n' 'wrote invalid proposals manifest'
`
	runner := &sequenceAdapter{
		testAdapter: testAdapter{
			activity: ActivityPolicy{
				MonitorArtifacts:            true,
				MonitorPreArtifact:          true,
				PreArtifactStallWindow:      20 * time.Millisecond,
				RetryPreArtifactStallWindow: successfulZeroOutputRetryWindow,
				PostArtifactStallWindow:     20 * time.Millisecond,
				PollInterval:                5 * time.Millisecond,
				PostTerminateDrain:          10 * time.Millisecond,
				TerminateGrace:              10 * time.Millisecond,
			},
			recovery: RecoveryPolicy{
				AcceptValidArtifactsAfterStop:            true,
				RetryInvalidOrMissingArtifactsOnce:       true,
				RetryZeroOutputPreArtifactStallOnce:      true,
				ClassifySilentRetryExhaustionUnavailable: true,
			},
		},
		commands: []string{
			writeEngineScript(t, "#!/usr/bin/env bash\nset -eu\nsleep 10\n"),
			writeEngineScript(t, invalidDraftScript),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), successfulZeroOutputRetryTimeout)
	defer cancel()
	_, err := RunHeadlessProvider(ctx, task, runner)
	if err == nil {
		t.Fatal("expected invalid proposals retry artifact to fail")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T: %v", err, err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRuntimeContract {
		t.Fatalf("expected runtime_contract_failed, got %s (%v)", runnerErr.Code, err)
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
	if snapshot.WriteRootAuthoredFiles != 0 || snapshot.DraftRootAuthoredFiles != 1 {
		t.Fatalf("expected split draft counts write_root=0 draft_final_root=1, got %#v", snapshot)
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

func TestMonitorArtifactStallIgnoresStaleArtifactsUntilFreshMutation(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-monitor-stale-artifacts")
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "constitution-draft.json"), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatalf("write stale manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, "charter-overview.md"), []byte("# stale\n"), 0o644); err != nil {
		t.Fatalf("write stale draft: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker := newCommandActivityTracker(time.Now().Add(-time.Second))
	done := make(chan StallError, 1)
	go func() {
		err, stalled := monitorArtifactStall(ctx, task, tracker, ActivityPolicy{
			MonitorArtifacts:           true,
			MonitorPreArtifact:         true,
			FreshArtifactMutationAfter: time.Now().UTC().Add(time.Second),
			PreArtifactStallWindow:     20 * time.Millisecond,
			PostArtifactStallWindow:    20 * time.Millisecond,
			PartialArtifactStallWindow: 20 * time.Millisecond,
			PollInterval:               5 * time.Millisecond,
		})
		if stalled {
			done <- err
		}
	}()

	select {
	case err := <-done:
		if !errors.Is(err, ErrStalledBeforeArtifacts) {
			t.Fatalf("expected stale artifacts to use pre-artifact stall, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected monitor to trip pre-artifact window for stale artifacts")
	}
}

func TestMonitorArtifactStallTripsInvalidArtifactWindowDespitePipeActivity(t *testing.T) {
	t.Parallel()

	task := newDraftTask(t, "run-monitor-invalid-active-pipe")
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "constitution-draft.json"), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatalf("write invalid manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, "charter-overview.md"), []byte("# bootstrap\n"), 0o644); err != nil {
		t.Fatalf("write invalid draft: %v", err)
	}
	old := time.Now().Add(-time.Second)
	for _, path := range []string{
		filepath.Join(task.WriteRoot, "constitution-draft.json"),
		filepath.Join(task.DraftFinalRoot, "charter-overview.md"),
	} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatalf("chtimes %s: %v", path, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker := newCommandActivityTracker(time.Now().UTC())
	keepPipeFresh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-keepPipeFresh:
				return
			case at := <-ticker.C:
				tracker.Note(at)
			}
		}
	}()
	defer close(keepPipeFresh)

	done := make(chan StallError, 1)
	go func() {
		err, stalled := monitorArtifactStall(ctx, task, tracker, ActivityPolicy{
			MonitorArtifacts:           true,
			MonitorPreArtifact:         true,
			PostArtifactStallWindow:    time.Hour,
			PartialArtifactStallWindow: 20 * time.Millisecond,
			PollInterval:               5 * time.Millisecond,
		})
		if stalled {
			done <- err
		}
	}()

	select {
	case err := <-done:
		if !errors.Is(err, ErrStalledAfterArtifacts) {
			t.Fatalf("expected invalid artifact post-artifact stall, got %v", err)
		}
		if got := err.Diagnostic.ArtifactState; got != "invalid" {
			t.Fatalf("expected invalid artifact state, got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected invalid artifact stall despite fresh pipe activity")
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

func TestAllDraftMarkdownOutputsChangedRequiresEveryMarkdownTarget(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-all-draft-markdown-changed")
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "asis-draft-manifest.json"), []byte(steppolicy.RuntimeDraftManifestTaskSkeleton(task)), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	for _, name := range []string{"overview.md", "summary.md", "architect-summary.md"} {
		if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, name), []byte("# "+name+"\n\nProvider authored as-is draft.\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	before, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		t.Fatalf("snapshot before: %v", err)
	}
	if !allDraftMarkdownOutputsChanged(task, writeRootFileSnapshot{}) {
		t.Fatal("expected initial draft files to count as changed from an empty snapshot")
	}
	if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, "summary.md"), []byte("# summary\n\nProvider authored update.\n"), 0o644); err != nil {
		t.Fatalf("rewrite summary: %v", err)
	}
	if allDraftMarkdownOutputsChanged(task, before) {
		t.Fatal("expected partial markdown rewrite to be rejected")
	}
	for _, name := range []string{"overview.md", "architect-summary.md"} {
		if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, name), []byte("# "+name+"\n\nProvider authored update.\n"), 0o644); err != nil {
			t.Fatalf("rewrite %s: %v", name, err)
		}
	}
	if !allDraftMarkdownOutputsChanged(task, before) {
		t.Fatal("expected all markdown rewrites to be accepted")
	}
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
	draftEnrichmentCommand string
	promptBytes            int
	activity               ActivityPolicy
	recovery               RecoveryPolicy
}

type sequenceAdapter struct {
	testAdapter
	commands []string
	mu       sync.Mutex
	calls    int
}

func (a *sequenceAdapter) CommandSpec(acpruntime.Task) (CommandSpec, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.commands) == 0 {
		return CommandSpec{}, errors.New("sequence command is unavailable")
	}
	index := a.calls
	if index >= len(a.commands) {
		index = len(a.commands) - 1
	}
	a.calls++
	return CommandSpec{Command: a.commands[index], PromptBytes: a.promptBytes}, nil
}

func (a testAdapter) Provider() acpruntime.Provider {
	return acpruntime.ProviderQwenCode
}

func (a testAdapter) RuntimeVersion() string {
	return "test"
}

func (a testAdapter) CommandSpec(acpruntime.Task) (CommandSpec, error) {
	return CommandSpec{Command: a.command, PromptBytes: a.promptBytes}, nil
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

func (a testAdapter) DraftArtifactEnrichmentCommandSpec(acpruntime.Task, error) (CommandSpec, error) {
	if strings.TrimSpace(a.draftEnrichmentCommand) == "" {
		return CommandSpec{}, errors.New("draft enrichment command is unavailable")
	}
	return CommandSpec{Command: a.draftEnrichmentCommand}, nil
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

func newAsIsDraftTask(t *testing.T, runID string) acpruntime.Task {
	t.Helper()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", runID, "asis")
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
		StepID:            "init.step2.asis_docs",
		StepContract:      "as_is",
		AgentRole:         "architect",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ReadContextRoots:  []string{draftRoot},
		ExpectedArtifacts: []string{"asis-draft-manifest.json"},
		StartedAtUTC:      time.Now().UTC(),
	}
}

func newProposalsDraftTask(t *testing.T, runID string) acpruntime.Task {
	t.Helper()

	workspace := t.TempDir()
	writeRoot := filepath.Join(workspace, "reports", "taskruns", runID, "proposals")
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
		StepID:            "init.step4.proposals",
		StepContract:      "proposals",
		AgentRole:         "architect",
		Workspace:         workspace,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"proposals-draft-manifest.json"},
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

func scaffoldCollectManifestJSON(task acpruntime.Task) string {
	return `{
  "version": 1,
  "run_id": "` + task.RunID + `",
  "step_id": "init.step1.collect",
  "shard_id": "posthog-cli-common",
  "domain_id": "posthog",
  "agent_role": "shard-analyst",
  "artifact_root": "` + task.ArtifactRoot + `",
  "repo_scopes": ["posthog"],
  "path_scopes": ["cli", "common"],
  "documents": [
    {
      "id": "doc.posthog.cli_common.overview",
      "kind": "report",
      "title": "PostHog CLI Common Overview",
      "path": "overview.md",
      "canonical_path": "reports/as-is/posthog-cli-common/overview.md",
      "topics": ["posthog"],
      "citation_ids": ["cite.posthog.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.posthog.readme",
      "repo": "posthog",
      "path": "README.md",
      "claim_ids": ["claim.posthog.runtime"],
      "document_ids": ["doc.posthog.cli_common.overview"]
    }
  ],
  "semantic": {
    "coverage": {
      "observed": ["scoped surface"],
      "missing": ["owner mapping evidence not confirmed from scoped repository files"],
      "notes": ["collect manifest covers the assigned shard scope and evidence paths listed in citations"]
    },
    "questions": [
      {
        "id": "q.posthog.owner",
        "text": "Who owns this scoped surface?"
      }
    ],
    "entities": [
      {
        "id": "ent.posthog.repo",
        "name": "PostHog",
        "type": "domain",
        "provenance": {
          "kind": "observation",
          "confidence": 0.8,
          "evidence": [
            {
              "repo": "posthog",
              "path": "README.md"
            }
          ]
        }
      },
      {
        "id": "ent.posthog.cli_common",
        "name": "PostHog CLI Common",
        "type": "component",
        "provenance": {
          "kind": "observation",
          "confidence": 0.7,
          "evidence": [
            {
              "repo": "posthog",
              "path": "README.md"
            }
          ]
        }
      }
    ],
    "edges": [
      {
        "id": "edge.posthog.contains.scoped_surface",
        "type": "contains",
        "from": "ent.posthog.repo",
        "to": "ent.posthog.cli_common",
        "name": "contains scoped surface",
        "provenance": {
          "kind": "observation",
          "confidence": 0.7,
          "evidence": [
            {
              "repo": "posthog",
              "path": "README.md"
            }
          ]
        }
      }
    ],
    "findings": [
      {
        "id": "finding.posthog.owner_mapping",
        "severity": "medium",
        "title": "Owner mapping not confirmed",
        "description": "Scoped evidence identifies PostHog CLI Common but does not confirm an owning team.",
        "rule_id": "analysis.owner.mapping",
        "related_ids": ["ent.posthog.cli_common"],
        "provenance": {
          "kind": "observation",
          "confidence": 0.5,
          "evidence": [
            {
              "repo": "posthog",
              "path": "README.md"
            }
          ]
        }
      }
    ]
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

func proposalsDraftScript(task acpruntime.Task, tail string) string {
	return `#!/usr/bin/env bash
set -eu
write_root=` + shellQuote(task.WriteRoot) + `
draft_root=` + shellQuote(task.DraftFinalRoot) + `
mkdir -p "$write_root" "$draft_root"
cat >"$write_root/proposals-draft-manifest.json" <<'EOF'
` + steppolicy.RuntimeDraftManifestTaskSkeleton(task) + `
EOF
cat >"$draft_root/proposal.md" <<'EOF'
# Runtime Recommendations

## Summary
- Provider authored proposal draft.
EOF
cat >"$draft_root/changelog.md" <<'EOF'
# Runtime Proposal Changelog

## Changes
- Provider authored proposal changelog.
EOF
` + tail + "\n"
}

func asIsDraftScript(task acpruntime.Task, files []string, tail string) string {
	lines := []string{
		"#!/usr/bin/env bash",
		"set -eu",
		"write_root=" + shellQuote(task.WriteRoot),
		"draft_root=" + shellQuote(task.DraftFinalRoot),
		"mkdir -p \"$write_root\" \"$draft_root\"",
		"cat >\"$write_root/asis-draft-manifest.json\" <<'EOF'",
		steppolicy.RuntimeDraftManifestTaskSkeleton(task),
		"EOF",
	}
	for _, name := range files {
		lines = append(lines,
			"cat >\"$draft_root/"+name+"\" <<'EOF'",
			"# "+strings.TrimSuffix(name, filepath.Ext(name)),
			"",
			"Provider authored as-is draft artifact.",
			"EOF",
		)
	}
	lines = append(lines, tail)
	return strings.Join(lines, "\n") + "\n"
}

func asIsBootstrapDraftScript(task acpruntime.Task, tail string) string {
	lines := []string{
		"#!/usr/bin/env bash",
		"set -eu",
		"write_root=" + shellQuote(task.WriteRoot),
		"draft_root=" + shellQuote(task.DraftFinalRoot),
		"mkdir -p \"$write_root\" \"$draft_root\"",
		"cat >\"$write_root/asis-draft-manifest.json\" <<'EOF'",
		steppolicy.RuntimeDraftManifestTaskSkeleton(task),
		"EOF",
	}
	for _, name := range []string{"overview.md", "summary.md", "architect-summary.md"} {
		lines = append(lines,
			"cat >\"$draft_root/"+name+"\" <<'EOF'",
			"# "+strings.TrimSuffix(name, filepath.Ext(name)),
			"",
			"## Scope",
			"- Run: "+task.RunID,
			"- Step: "+task.StepID,
			"",
			"## Summary",
			"- Runtime draft recovery initialized this artifact for the scoped analysis step.",
			"- Use collected shard manifests and validator output as the evidence source before final review.",
			"EOF",
		)
	}
	lines = append(lines, tail)
	return strings.Join(lines, "\n") + "\n"
}

func compactDraftScript(task acpruntime.Task) string {
	manifest := `{"version":1,"run_id":"` + task.RunID + `","step_id":"init.step0.constitution","step_contract":"constitution","agent_role":"architect","outputs":[{"path":"charter-overview.md","canonical_path":"charter/overview.md","kind":"charter","title":"Constitution"},{"path":"baseline-subagents.yaml","canonical_path":"skills/subagents.yaml","kind":"bundle","title":"Baseline Subagents"}]}`
	return "#!/usr/bin/env bash\nset -eu\n" +
		"write_root=" + shellQuote(task.WriteRoot) + "\n" +
		"draft_root=" + shellQuote(task.DraftFinalRoot) + "\n" +
		"mkdir -p \"$write_root\" \"$draft_root\"\n" +
		"printf '%s\\n' " + shellQuote(manifest) + " > \"$write_root/constitution-draft.json\"\n" +
		"printf '%s\\n' '# Constitution' > \"$draft_root/charter-overview.md\"\n" +
		"printf '%s\\n' 'version: 1' > \"$draft_root/baseline-subagents.yaml\"\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func hasDiagnostic(events []acpruntime.DiagnosticEvent, message string, severity string) bool {
	for _, event := range events {
		if event.Message != message {
			continue
		}
		if strings.TrimSpace(severity) == "" {
			return true
		}
		if got, ok := event.Fields["severity"].(string); ok && got == severity {
			return true
		}
	}
	return false
}

func hasDiagnosticField(events []acpruntime.DiagnosticEvent, message string, key string, value string) bool {
	for _, event := range events {
		if event.Message != message {
			continue
		}
		if got, ok := event.Fields[key].(string); ok && got == value {
			return true
		}
	}
	return false
}

func hasRuntimeWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
