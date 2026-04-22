package qwencode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const (
	collectPostArtifactStallWindow = 20 * time.Second
	collectPreArtifactStallWindow  = 75 * time.Second
	collectStallPollInterval       = 2 * time.Second
	collectStallTerminateGrace     = 2 * time.Second
	collectStallPostTerminateDrain = 500 * time.Millisecond
)

type runQwenOptions struct {
	EnableCollectStallMonitor      bool
	DisableCollectPreArtifactStall bool
	EnableDraftStallMonitor        bool
}

type collectStallPhase string

const (
	collectStallPhasePreArtifact  collectStallPhase = "pre_artifact"
	collectStallPhasePostArtifact collectStallPhase = "post_artifact"
)

type collectStallDiagnostic struct {
	StallPhase            collectStallPhase
	ManifestState         string
	AuthoredFileCount     int
	LastPipeActivity      time.Time
	LastWriteRootMutation time.Time
}

type collectStallError struct {
	Sentinel   error
	Diagnostic collectStallDiagnostic
}

func (e collectStallError) Error() string {
	if e.Sentinel == nil {
		return "collect_stalled"
	}
	return e.Sentinel.Error()
}

func (e collectStallError) Is(target error) bool {
	return target != nil && target == e.Sentinel
}

type collectWriteRootSnapshot struct {
	ManifestPresent   bool
	ManifestState     string
	AuthoredFileCount int
	LastMutation      time.Time
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

func runQwenCommand(ctx context.Context, task acpruntime.Task, command string, args []string, options runQwenOptions) (acpruntime.Result, error) {
	taskPayload, err := json.Marshal(task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	commandArgs := append([]string(nil), args...)
	if len(commandArgs) == 0 {
		commandArgs = buildDefaultQwenArgs(task, buildPrompt(task))
	}

	cmd := exec.CommandContext(ctx, command, commandArgs...)
	configureCommandProcessGroup(cmd)
	if workDir := strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)); workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Stdin = bytes.NewReader(taskPayload)

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
		if ctxErr := ctx.Err(); ctxErr != nil {
			return acpruntime.Result{}, ctxErr
		}
		return acpruntime.Result{}, err
	}
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	defer closeCommandPipe(stdoutPipe)
	defer closeCommandPipe(stderrPipe)

	activityTracker := newCommandActivityTracker(time.Now().UTC())
	monitorCtx, stopMonitor := context.WithCancel(context.Background())
	defer stopMonitor()
	var monitorWG sync.WaitGroup
	stallCh := make(chan collectStallError, 1)
	if options.EnableCollectStallMonitor && isCollectStep(task.StepID) && strings.TrimSpace(task.WriteRoot) != "" && cmd.Process != nil {
		monitorWG.Add(1)
		go func() {
			defer monitorWG.Done()
			if stallErr, stalled := monitorCollectStall(monitorCtx, cmd.Process, task, activityTracker, !options.DisableCollectPreArtifactStall); stalled {
				select {
				case stallCh <- stallErr:
				default:
				}
			}
		}()
	}
	if options.EnableDraftStallMonitor && isDraftStep(task.StepID) && strings.TrimSpace(task.WriteRoot) != "" && strings.TrimSpace(task.DraftFinalRoot) != "" && cmd.Process != nil {
		monitorWG.Add(1)
		go func() {
			defer monitorWG.Done()
			if stallErr, stalled := monitorDraftArtifactStall(monitorCtx, cmd.Process, task, activityTracker); stalled {
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
		captureErr(captureCommandStream(&activityTrackingReader{reader: stdoutPipe, tracker: activityTracker}, stdout, task, acpruntime.OutputStreamStdout))
	}()
	go func() {
		defer wg.Done()
		captureErr(captureCommandStream(&activityTrackingReader{reader: stderrPipe, tracker: activityTracker}, stderr, task, acpruntime.OutputStreamStderr))
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
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, collectStallPostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		waitErr = waitForCommandExit(cmd, collectStallPostTerminateDrain)
		if waitErr == nil {
			waitErr = ctx.Err()
		}
	case stallErr := <-stallCh:
		if cmd.Process != nil {
			_ = killCommandProcessTree(cmd.Process)
		}
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, collectStallPostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		_ = waitForCommandExit(cmd, collectStallTerminateGrace)
		return acpruntime.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, stallErr
	}

	if waitErr == nil && streamErr != nil {
		waitErr = streamErr
	}
	result := acpruntime.Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if waitErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, ctxErr
		}
		return result, waitErr
	}
	return result, nil
}

func captureCommandStream(reader io.Reader, sink *commandOutputBuffer, task acpruntime.Task, stream acpruntime.OutputStream) error {
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

func waitForCommandExitAsync(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		_ = cmd.Wait()
	}()
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

func monitorCollectStall(ctx context.Context, process *os.Process, task acpruntime.Task, tracker *commandActivityTracker, monitorPreArtifact bool) (collectStallError, bool) {
	for {
		select {
		case <-ctx.Done():
			return collectStallError{}, false
		case <-time.After(collectStallPollInterval):
		}
		snapshot := collectWriteRootState(task.WriteRoot)
		lastPipe := tracker.LastRead()
		lastMutation := snapshot.LastMutation
		if !snapshot.ManifestPresent && !lastMutation.IsZero() && time.Since(lastMutation) < collectPreArtifactStallWindow {
			continue
		}
		if !lastPipe.IsZero() && time.Since(lastPipe) < collectPreArtifactStallWindow {
			continue
		}
		if snapshot.ManifestPresent {
			if !lastPipe.IsZero() && time.Since(lastPipe) < collectPostArtifactStallWindow {
				continue
			}
			return collectStallError{
				Sentinel: errCollectStalledAfterArtifacts,
				Diagnostic: collectStallDiagnostic{
					StallPhase:            collectStallPhasePostArtifact,
					ManifestState:         snapshot.ManifestState,
					AuthoredFileCount:     snapshot.AuthoredFileCount,
					LastPipeActivity:      lastPipe,
					LastWriteRootMutation: lastMutation,
				},
			}, true
		}
		if !monitorPreArtifact {
			continue
		}
		return collectStallError{
			Sentinel: errCollectStalledBeforeArtifacts,
			Diagnostic: collectStallDiagnostic{
				StallPhase:            collectStallPhasePreArtifact,
				ManifestState:         snapshot.ManifestState,
				AuthoredFileCount:     snapshot.AuthoredFileCount,
				LastPipeActivity:      lastPipe,
				LastWriteRootMutation: lastMutation,
			},
		}, true
	}
}

func monitorDraftArtifactStall(ctx context.Context, process *os.Process, task acpruntime.Task, tracker *commandActivityTracker) (collectStallError, bool) {
	for {
		select {
		case <-ctx.Done():
			return collectStallError{}, false
		case <-time.After(collectStallPollInterval):
		}
		if _, _, err := validateRuntimeDraftArtifactsAtWriteRoot(task); err != nil {
			continue
		}
		lastPipe := tracker.LastRead()
		if !lastPipe.IsZero() && time.Since(lastPipe) < collectPostArtifactStallWindow {
			continue
		}
		return collectStallError{
			Sentinel: errDraftStalledAfterArtifacts,
			Diagnostic: collectStallDiagnostic{
				StallPhase:            collectStallPhasePostArtifact,
				ManifestState:         "valid",
				AuthoredFileCount:     countDraftFiles(task.DraftFinalRoot),
				LastPipeActivity:      lastPipe,
				LastWriteRootMutation: latestDraftMutation(task.WriteRoot, task.DraftFinalRoot),
			},
		}, true
	}
}

func collectWriteRootState(writeRoot string) collectWriteRootSnapshot {
	snapshot := collectWriteRootSnapshot{}
	if strings.TrimSpace(writeRoot) == "" {
		return snapshot
	}
	manifestPath := filepath.Join(writeRoot, "shard-pack-manifest.json")
	if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
		snapshot.ManifestPresent = true
		snapshot.LastMutation = info.ModTime().UTC()
		snapshot.ManifestState = "present"
	}
	entries, err := os.ReadDir(writeRoot)
	if err != nil {
		return snapshot
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "shard-pack-manifest.json" {
			continue
		}
		snapshot.AuthoredFileCount++
		if info, statErr := entry.Info(); statErr == nil {
			modTime := info.ModTime().UTC()
			if modTime.After(snapshot.LastMutation) {
				snapshot.LastMutation = modTime
			}
		}
	}
	if snapshot.ManifestPresent {
		raw, err := os.ReadFile(manifestPath)
		if err == nil {
			if _, parseErr := contracts.ParseShardPackManifest(raw); parseErr == nil {
				snapshot.ManifestState = "valid"
			} else {
				snapshot.ManifestState = "invalid"
			}
		}
	}
	return snapshot
}

func countDraftFiles(draftRoot string) int {
	draftRoot = strings.TrimSpace(draftRoot)
	if draftRoot == "" {
		return 0
	}
	entries, err := os.ReadDir(draftRoot)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		count++
	}
	return count
}

func latestDraftMutation(writeRoot string, draftRoot string) time.Time {
	latest := collectWriteRootState(writeRoot).LastMutation
	draftRoot = strings.TrimSpace(draftRoot)
	if draftRoot == "" {
		return latest
	}
	entries, err := os.ReadDir(draftRoot)
	if err != nil {
		return latest
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		modTime := info.ModTime().UTC()
		if modTime.After(latest) {
			latest = modTime
		}
	}
	return latest
}
