package providercommon

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
)

func runProviderCommand(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, policy ActivityPolicy) (acpruntime.Result, error) {
	spec, err := adapter.CommandSpec(task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if spec.Provider == "" {
		spec.Provider = adapter.Provider()
	}
	return runCommandSpec(ctx, task, spec, policy)
}

func runCommandSpec(ctx context.Context, task acpruntime.Task, spec CommandSpec, policy ActivityPolicy) (acpruntime.Result, error) {
	commandDiag := newProviderCommandDiagnostics(spec, task)
	cmd := exec.CommandContext(ctx, strings.TrimSpace(spec.Command), append([]string(nil), spec.Args...)...)
	configureCommandProcessGroup(cmd)
	if dir := strings.TrimSpace(spec.Dir); dir != "" {
		cmd.Dir = dir
	}
	if spec.Stdin != nil {
		cmd.Stdin = spec.Stdin
	}

	stdout := &commandOutputBuffer{}
	stderr := &commandOutputBuffer{}
	stdoutPipe, stdoutWriter, err := os.Pipe()
	if err != nil {
		return acpruntime.Result{}, err
	}
	stderrPipe, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		return acpruntime.Result{}, err
	}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if err := cmd.Start(); err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		_ = stderrPipe.Close()
		_ = stderrWriter.Close()
		commandDiag.finish("start_failed", 0, 0, err)
		result := acpruntime.Result{Diagnostics: map[string]any{"provider_lifecycle": commandDiag.fields()}}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, ctxErr
		}
		return result, err
	}
	commandDiag.markStarted(cmd.Process)
	emitDiagnostic(task, "provider command started", commandDiag.fields())
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	defer closeCommandPipe(stdoutPipe)
	defer closeCommandPipe(stderrPipe)

	activityTracker := newCommandActivityTracker(time.Now().UTC())
	monitorCtx, stopMonitor := context.WithCancel(context.Background())
	defer stopMonitor()
	var monitorWG sync.WaitGroup
	stallCh := make(chan StallError, 1)
	if policy.MonitorArtifacts && strings.TrimSpace(task.WriteRoot) != "" {
		monitorWG.Add(1)
		go func() {
			defer monitorWG.Done()
			if stallErr, stalled := monitorArtifactStall(monitorCtx, task, activityTracker, policy); stalled {
				select {
				case stallCh <- stallErr:
				default:
				}
			}
		}()
	}

	var streamErr error
	var streamErrMu sync.Mutex
	captureErr := func(captureErr error) {
		if captureErr == nil {
			return
		}
		streamErrMu.Lock()
		defer streamErrMu.Unlock()
		if streamErr == nil {
			streamErr = captureErr
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	streamDone := make(chan struct{})
	go func() {
		defer wg.Done()
		captureErr(captureTrackedCommandStream(&activityTrackingReader{reader: stdoutPipe, tracker: activityTracker}, stdout, task, acpruntime.OutputStreamStdout))
	}()
	go func() {
		defer wg.Done()
		captureErr(captureTrackedCommandStream(&activityTrackingReader{reader: stderrPipe, tracker: activityTracker}, stderr, task, acpruntime.OutputStreamStderr))
	}()
	go func() {
		wg.Wait()
		close(streamDone)
	}()

	var waitErr error
	select {
	case <-streamDone:
		stopMonitor()
		monitorWG.Wait()
		waitErr = cmd.Wait()
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = killCommandProcessTree(cmd.Process)
		}
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, policy.PostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		waitErr = waitForCommandExit(cmd, policy.PostTerminateDrain)
		if waitErr == nil {
			waitErr = ctx.Err()
		}
	case stallErr := <-stallCh:
		if cmd.Process != nil {
			_ = killCommandProcessTree(cmd.Process)
		}
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, policy.PostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		_ = waitForCommandExit(cmd, policy.TerminateGrace)
		stdoutText := stdout.String()
		stderrText := stderr.String()
		stallErr.Diagnostic.StdoutBytes = len([]byte(stdoutText))
		stallErr.Diagnostic.StderrBytes = len([]byte(stderrText))
		commandDiag.finish("stall", len([]byte(stdoutText)), len([]byte(stderrText)), stallErr)
		emitDiagnostic(task, "provider command finished", commandDiag.fields())
		return acpruntime.Result{
			Stdout:      stdoutText,
			Stderr:      stderrText,
			Diagnostics: map[string]any{"provider_lifecycle": commandDiag.fields()},
		}, stallErr
	}

	if waitErr == nil && streamErr != nil {
		waitErr = streamErr
	}
	result := acpruntime.Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	exitReason := "success"
	if waitErr != nil {
		exitReason = "error"
		if ctxErr := ctx.Err(); ctxErr != nil {
			exitReason = "context_canceled"
			commandDiag.finish(exitReason, len([]byte(result.Stdout)), len([]byte(result.Stderr)), ctxErr)
			emitDiagnostic(task, "provider command finished", commandDiag.fields())
			result.Diagnostics = map[string]any{"provider_lifecycle": commandDiag.fields()}
			return result, ctxErr
		}
		execFailure := runnerdiag.BuildExecFailure(waitErr, result.Stdout, result.Stderr)
		commandDiag.finish(exitReason, len([]byte(result.Stdout)), len([]byte(result.Stderr)), execFailure)
		emitDiagnostic(task, "provider command finished", commandDiag.fields())
		result.Diagnostics = map[string]any{"provider_lifecycle": commandDiag.fields()}
		return result, execFailure
	}
	commandDiag.finish(exitReason, len([]byte(result.Stdout)), len([]byte(result.Stderr)), nil)
	emitDiagnostic(task, "provider command finished", commandDiag.fields())
	result.Diagnostics = map[string]any{"provider_lifecycle": commandDiag.fields()}
	return result, nil
}

type providerCommandDiagnostics struct {
	Provider       string
	Command        string
	CommandPath    string
	Args           []string
	Dir            string
	IncludeDirs    []string
	Environment    map[string]any
	TimeoutProfile map[string]any
	PID            int
	StartedAt      time.Time
	FinishedAt     time.Time
	DurationMillis int64
	ExitReason     string
	Error          string
	StdoutBytes    int
	StderrBytes    int
}

func newProviderCommandDiagnostics(spec CommandSpec, task acpruntime.Task) *providerCommandDiagnostics {
	command := strings.TrimSpace(spec.Command)
	commandPath := ""
	if command != "" {
		if resolved, err := exec.LookPath(command); err == nil {
			commandPath = resolved
		}
	}
	return &providerCommandDiagnostics{
		Provider:       strings.TrimSpace(string(spec.Provider)),
		Command:        command,
		CommandPath:    strings.TrimSpace(commandPath),
		Args:           redactArgs(spec.Args),
		Dir:            strings.TrimSpace(spec.Dir),
		IncludeDirs:    normalizeDiagnosticPaths(spec.IncludeDirs),
		Environment:    allowlistedProviderEnvDiagnostics(),
		TimeoutProfile: cloneDiagnosticMap(task.RuntimeTimeoutProfile),
		StartedAt:      time.Now().UTC(),
	}
}

func (d *providerCommandDiagnostics) markStarted(process *os.Process) {
	if d == nil || process == nil {
		return
	}
	d.PID = process.Pid
}

func (d *providerCommandDiagnostics) finish(reason string, stdoutBytes int, stderrBytes int, err error) {
	if d == nil {
		return
	}
	d.FinishedAt = time.Now().UTC()
	d.DurationMillis = d.FinishedAt.Sub(d.StartedAt).Milliseconds()
	d.ExitReason = strings.TrimSpace(reason)
	d.StdoutBytes = stdoutBytes
	d.StderrBytes = stderrBytes
	if err != nil {
		d.Error = strings.TrimSpace(err.Error())
	}
}

func (d *providerCommandDiagnostics) fields() map[string]any {
	if d == nil {
		return map[string]any{}
	}
	fields := map[string]any{
		"selected_provider": d.Provider,
		"command":           d.Command,
		"command_path":      d.CommandPath,
		"argv":              append([]string(nil), d.Args...),
		"cwd":               d.Dir,
		"include_dirs":      append([]string(nil), d.IncludeDirs...),
		"env":               d.Environment,
		"timeout_profile":   cloneDiagnosticMap(d.TimeoutProfile),
		"pid":               d.PID,
		"started_at":        d.StartedAt.Format(time.RFC3339Nano),
		"stdout_bytes":      d.StdoutBytes,
		"stderr_bytes":      d.StderrBytes,
	}
	if !d.FinishedAt.IsZero() {
		fields["finished_at"] = d.FinishedAt.Format(time.RFC3339Nano)
		fields["duration_ms"] = d.DurationMillis
		fields["exit_reason"] = d.ExitReason
	}
	if d.Error != "" {
		fields["error"] = d.Error
	}
	return fields
}

func redactArgs(args []string) []string {
	redacted := make([]string, 0, len(args))
	redactNext := false
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if redactNext {
			redacted = append(redacted, redactSecretLikeValue(trimmed))
			redactNext = false
			continue
		}
		if idx := strings.Index(trimmed, "="); idx > 0 && isSecretKey(trimmed[:idx]) {
			redacted = append(redacted, trimmed[:idx+1]+redactSecretLikeValue(trimmed[idx+1:]))
			continue
		}
		if isSecretFlag(trimmed) {
			redacted = append(redacted, trimmed)
			redactNext = true
			continue
		}
		redacted = append(redacted, arg)
	}
	return redacted
}

func normalizeDiagnosticPaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		if abs, err := filepath.Abs(trimmed); err == nil {
			trimmed = abs
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func cloneDiagnosticMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = value
	}
	return out
}

func allowlistedProviderEnvDiagnostics() map[string]any {
	keys := []string{
		acpruntime.RuntimeProviderEnv,
		"ACP_CLAUDE_CMD",
		"ACP_QWEN_CMD",
		"ACP_CODEX_CMD",
		acpruntime.RuntimeStepTimeoutEnv,
		acpruntime.RuntimeHeartbeatEnv,
		acpruntime.PipelineTimeoutEnv,
		acpruntime.PipelineKillGraceEnv,
		"ACP_RUNTIME_TIMEOUT_SEC",
		"ACP_RUNTIME_TIMEOUT_PROFILE",
		"ACP_HEADLESS_TIMEOUT_SEC",
		"ACP_HEADLESS_TIMEOUT_PROFILE",
	}
	out := map[string]any{}
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			out[key] = redactedEnvValue(key, value)
		} else {
			out[key] = map[string]any{"present": false}
		}
	}
	return out
}

func redactedEnvValue(key string, value string) map[string]any {
	trimmed := strings.TrimSpace(value)
	result := map[string]any{
		"present": true,
		"bytes":   len([]byte(value)),
		"sha256":  sha256Hex(value),
	}
	if trimmed == "" {
		result["value"] = ""
		return result
	}
	if !isSecretKey(key) && isRawDiagnosticEnv(key) && !isSecretLikeValue(trimmed) {
		result["value"] = trimmed
	}
	return result
}

func isRawDiagnosticEnv(key string) bool {
	switch strings.TrimSpace(key) {
	case "ACP_CLAUDE_CMD",
		"ACP_QWEN_CMD",
		"ACP_CODEX_CMD",
		acpruntime.RuntimeProviderEnv,
		acpruntime.RuntimeStepTimeoutEnv,
		acpruntime.RuntimeHeartbeatEnv,
		acpruntime.PipelineTimeoutEnv,
		acpruntime.PipelineKillGraceEnv,
		"ACP_RUNTIME_TIMEOUT_SEC",
		"ACP_RUNTIME_TIMEOUT_PROFILE",
		"ACP_HEADLESS_TIMEOUT_SEC",
		"ACP_HEADLESS_TIMEOUT_PROFILE":
		return true
	default:
		return false
	}
}

func isSecretFlag(arg string) bool {
	trimmed := strings.TrimSpace(arg)
	if !strings.HasPrefix(trimmed, "-") {
		return false
	}
	return isSecretKey(trimmed)
}

func isSecretKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.TrimLeft(normalized, "-")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	for _, marker := range []string{"token", "secret", "password", "passwd", "apikey", "api_key", "cookie", "credential", "auth"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func isSecretLikeValue(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	for _, marker := range []string{"token", "secret", "password", "passwd", "apikey", "api_key", "cookie", "credential", "authorization", "bearer "} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func redactSecretLikeValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<empty>"
	}
	return "<redacted sha256=" + sha256Hex(value) + ">"
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type commandActivityTracker struct {
	mu       sync.Mutex
	lastRead time.Time
}

type activityTrackingReader struct {
	reader  io.Reader
	tracker *commandActivityTracker
}

type commandOutputBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func newCommandActivityTracker(start time.Time) *commandActivityTracker {
	return &commandActivityTracker{lastRead: start.UTC()}
}

func (t *commandActivityTracker) Note(at time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if at.After(t.lastRead) {
		t.lastRead = at.UTC()
	}
}

func (t *commandActivityTracker) LastRead() time.Time {
	if t == nil {
		return time.Time{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastRead
}

func (r *activityTrackingReader) Read(p []byte) (int, error) {
	if r == nil || r.reader == nil {
		return 0, io.EOF
	}
	n, err := r.reader.Read(p)
	if n > 0 && r.tracker != nil {
		r.tracker.Note(time.Now().UTC())
	}
	return n, err
}

func (b *commandOutputBuffer) WriteString(value string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.WriteString(value)
}

func (b *commandOutputBuffer) String() string {
	if b == nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func captureTrackedCommandStream(reader io.Reader, sink *commandOutputBuffer, task acpruntime.Task, stream acpruntime.OutputStream) error {
	if sink == nil {
		return errors.New("capture sink is nil")
	}
	bufReader := bufio.NewReader(reader)
	budget := &streamedOutputBudget{}
	for {
		part, err := bufReader.ReadString('\n')
		if len(part) > 0 {
			sink.WriteString(part)
			forwardStreamOutput(task, stream, part, budget)
		}
		if err != nil {
			if errors.Is(err, io.EOF) || isPipeClosedErr(err) {
				return nil
			}
			return err
		}
	}
}

type streamedOutputBudget struct {
	forwardedBytes int
	truncated      bool
}

func forwardStreamOutput(task acpruntime.Task, stream acpruntime.OutputStream, chunk string, budget *streamedOutputBudget) {
	if task.OnOutput == nil || budget == nil {
		return
	}
	normalized := strings.ReplaceAll(chunk, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if budget.truncated {
			continue
		}
		lineBytes := len([]byte(line))
		nextBytes := budget.forwardedBytes + lineBytes
		if nextBytes <= acpruntime.RuntimeOutputStreamHardCapBytes {
			budget.forwardedBytes = nextBytes
			task.OnOutput(acpruntime.OutputChunk{Stream: stream, Text: line})
			continue
		}
		budget.truncated = true
		task.OnOutput(acpruntime.OutputChunk{
			Stream:    stream,
			Truncated: true,
			Text:      fmt.Sprintf("%s output truncated after %d bytes (internal safeguard)", stream, acpruntime.RuntimeOutputStreamHardCapBytes),
		})
	}
}

func closeCommandPipe(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}

func waitForCommandStreams(stdoutPipe *os.File, stderrPipe *os.File, streamDone <-chan struct{}, timeout time.Duration) {
	closeCommandPipe(stdoutPipe)
	closeCommandPipe(stderrPipe)
	select {
	case <-streamDone:
	case <-time.After(timeout):
	}
}

func waitForCommandExit(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil {
		return nil
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return nil
	}
}

func isPipeClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrClosed) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "file already closed")
}
