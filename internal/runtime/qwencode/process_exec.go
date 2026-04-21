package qwencode

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/runnerdiag"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultbinding"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultcompat"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultextractor"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

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

func (b *commandOutputBuffer) BytesCopy() []byte {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

func (b *commandOutputBuffer) String() string {
	if b == nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func runQwenCommand(ctx context.Context, task acpruntime.Task, command string, args []string, options runQwenOptions) (acpruntime.Result, string, error, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	configureCommandProcessGroup(cmd)
	if workspace := strings.TrimSpace(task.Workspace); workspace != "" {
		cmd.Dir = workspace
	} else if writeRoot := strings.TrimSpace(task.WriteRoot); writeRoot != "" {
		cmd.Dir = writeRoot
	}

	stdout := &commandOutputBuffer{}
	stderr := &commandOutputBuffer{}
	stdoutPipe, stdoutWriter, err := os.Pipe()
	if err != nil {
		return acpruntime.Result{}, "", nil, err
	}
	stderrPipe, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		return acpruntime.Result{}, "", nil, err
	}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if err := cmd.Start(); err != nil {
		_ = stdoutPipe.Close()
		_ = stdoutWriter.Close()
		_ = stderrPipe.Close()
		_ = stderrWriter.Close()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return acpruntime.Result{}, "", nil, ctxErr
		}
		return acpruntime.Result{}, "", nil, err
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
	var stalledErr error
	var stalledMu sync.Mutex
	if options.EnableCollectStallMonitor && isCollectStep(task.StepID) && strings.TrimSpace(task.WriteRoot) != "" && cmd.Process != nil {
		monitorWG.Add(1)
		go func() {
			defer monitorWG.Done()
			stallErr, stalled := monitorCollectStall(monitorCtx, cmd.Process, task, activityTracker)
			if !stalled {
				return
			}
			stalledMu.Lock()
			stalledErr = stallErr
			stalledMu.Unlock()
			select {
			case stallCh <- stallErr:
			default:
			}
		}()
	}
	if options.EnableDraftStallMonitor && runtimedrafts.IsDraftStep(task.StepID) && strings.TrimSpace(task.WriteRoot) != "" && strings.TrimSpace(task.DraftFinalRoot) != "" && cmd.Process != nil {
		monitorWG.Add(1)
		go func() {
			defer monitorWG.Done()
			stallErr, stalled := monitorDraftArtifactStall(monitorCtx, cmd.Process, task, activityTracker)
			if !stalled {
				return
			}
			stalledMu.Lock()
			stalledErr = stallErr
			stalledMu.Unlock()
			select {
			case stallCh <- stallErr:
			default:
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
	stalledTriggered := false
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
	case <-stallCh:
		stalledTriggered = true
		waitForCommandStreams(stdoutPipe, stderrPipe, streamDone, collectStallPostTerminateDrain)
		stopMonitor()
		monitorWG.Wait()
		waitForCommandExitAsync(cmd)
	}

	if waitErr == nil && streamErr != nil {
		waitErr = streamErr
	}
	result := acpruntime.Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	stalledMu.Lock()
	currentStallErr := stalledErr
	stalledMu.Unlock()
	if currentStallErr != nil {
		stalledTriggered = true
	}
	if stalledTriggered {
		parsed, _, parseErr := parseCapturedTaskResult(task, stdout.BytesCopy(), result.Stdout, result.Stderr)
		if parseErr == nil {
			return parsed, "", nil, nil
		}
		return result, "", nil, currentStallErr
	}
	if waitErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, "", nil, ctxErr
		}
		return result, "", nil, runnerdiag.BuildExecFailure(waitErr, result.Stdout, result.Stderr)
	}

	parsed, parseStage, parseErr := parseCapturedTaskResult(task, stdout.BytesCopy(), result.Stdout, result.Stderr)
	return parsed, parseStage, parseErr, nil
}

type streamedOutputBudget struct {
	forwardedBytes int
	truncated      bool
}

func parseCapturedTaskResult(task acpruntime.Task, stdoutBytes []byte, stdoutText string, stderrText string) (acpruntime.Result, string, error) {
	rawTaskResult, err := taskresultextractor.Extract(stdoutBytes)
	if err != nil {
		parseStage := "extract"
		if taskresultextractor.IsTransportError(err) {
			parseStage = "transport"
		}
		return acpruntime.Result{
			Stdout: stdoutText,
			Stderr: stderrText,
		}, parseStage, err
	}
	if normalizedRawTaskResult, changed, normalizeErr := taskresultcompat.NormalizeRawTaskResult(task, rawTaskResult); normalizeErr == nil && changed {
		rawTaskResult = normalizedRawTaskResult
	}
	taskResult, err := contracts.ParseTaskResult(rawTaskResult)
	if err != nil {
		return acpruntime.Result{
			Stdout: stdoutText,
			Stderr: stderrText,
		}, "schema", err
	}
	if err := taskresultbinding.Validate(task, taskResult, acpruntime.ProviderQwenCode); err != nil {
		return acpruntime.Result{
			Stdout: stdoutText,
			Stderr: stderrText,
		}, "binding", err
	}
	return acpruntime.Result{
		TaskResult: taskResult,
		RawJSON:    rawTaskResult,
		Stdout:     stdoutText,
		Stderr:     stderrText,
	}, "", nil
}

func monitorCollectStall(
	ctx context.Context,
	process *os.Process,
	task acpruntime.Task,
	activity *commandActivityTracker,
) (collectStallError, bool) {
	startedAt := time.Now().UTC()
	ticker := time.NewTicker(collectStallPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return collectStallError{}, false
		case <-ticker.C:
			snapshot, err := scanCollectWriteRoot(task.WriteRoot)
			if err != nil {
				continue
			}
			lastPipeActivity := activity.LastRead()
			if lastPipeActivity.IsZero() {
				lastPipeActivity = startedAt
			}
			now := time.Now().UTC()
			lastWriteRootMutation := effectiveCollectMutationTime(snapshot.LastMutation, startedAt)

			if snapshot.ManifestPresent && snapshot.AuthoredFileCount > 0 {
				if now.Sub(lastPipeActivity) < collectPostArtifactStallWindow {
					continue
				}
				if now.Sub(lastWriteRootMutation) < collectPostArtifactStallWindow {
					continue
				}

				diagnostic := collectStallDiagnostic{
					StallPhase:            collectStallPhasePostArtifact,
					ManifestState:         strings.TrimSpace(snapshot.ManifestState),
					AuthoredFileCount:     snapshot.AuthoredFileCount,
					LastPipeActivity:      lastPipeActivity.UTC(),
					LastWriteRootMutation: lastWriteRootMutation.UTC(),
				}
				emitDiagnostic(task, "runtime task stalled after artifacts", diagnostic.fields(task))
				terminateProcessWithGrace(process)
				return collectStallError{Sentinel: errCollectStalledAfterArtifacts, Diagnostic: diagnostic}, true
			}

			if snapshot.ManifestPresent || snapshot.AuthoredFileCount > 0 {
				continue
			}
			if now.Sub(lastPipeActivity) < collectPreArtifactStallWindow {
				continue
			}
			if now.Sub(lastWriteRootMutation) < collectPreArtifactStallWindow {
				continue
			}

			diagnostic := collectStallDiagnostic{
				StallPhase:            collectStallPhasePreArtifact,
				ManifestState:         strings.TrimSpace(snapshot.ManifestState),
				AuthoredFileCount:     snapshot.AuthoredFileCount,
				LastPipeActivity:      lastPipeActivity.UTC(),
				LastWriteRootMutation: lastWriteRootMutation.UTC(),
			}
			emitDiagnostic(task, "runtime task stalled before artifacts", diagnostic.fields(task))
			terminateProcessWithGrace(process)
			return collectStallError{Sentinel: errCollectStalledBeforeArtifacts, Diagnostic: diagnostic}, true
		}
	}
}

type draftArtifactSnapshot struct {
	ManifestPresent bool
	ManifestState   string
	DraftFileCount  int
	LastMutation    time.Time
}

func monitorDraftArtifactStall(
	ctx context.Context,
	process *os.Process,
	task acpruntime.Task,
	activity *commandActivityTracker,
) (collectStallError, bool) {
	startedAt := time.Now().UTC()
	ticker := time.NewTicker(collectStallPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return collectStallError{}, false
		case <-ticker.C:
			snapshot, err := scanDraftArtifacts(task)
			if err != nil {
				continue
			}
			if !snapshot.ManifestPresent || snapshot.DraftFileCount == 0 {
				continue
			}
			lastPipeActivity := activity.LastRead()
			if lastPipeActivity.IsZero() {
				lastPipeActivity = startedAt
			}
			now := time.Now().UTC()
			lastMutation := effectiveCollectMutationTime(snapshot.LastMutation, startedAt)
			if now.Sub(lastPipeActivity) < collectPostArtifactStallWindow {
				continue
			}
			if now.Sub(lastMutation) < collectPostArtifactStallWindow {
				continue
			}

			diagnostic := collectStallDiagnostic{
				StallPhase:            collectStallPhasePostArtifact,
				ManifestState:         strings.TrimSpace(snapshot.ManifestState),
				AuthoredFileCount:     snapshot.DraftFileCount,
				LastPipeActivity:      lastPipeActivity.UTC(),
				LastWriteRootMutation: lastMutation.UTC(),
			}
			emitDiagnostic(task, "runtime task stalled after draft artifacts", diagnostic.fields(task))
			terminateProcessWithGrace(process)
			return collectStallError{Sentinel: errDraftStalledAfterArtifacts, Diagnostic: diagnostic}, true
		}
	}
}

func scanDraftArtifacts(task acpruntime.Task) (draftArtifactSnapshot, error) {
	snapshot := draftArtifactSnapshot{ManifestState: "missing"}
	writeRoot := strings.TrimSpace(task.WriteRoot)
	draftRoot := strings.TrimSpace(task.DraftFinalRoot)
	manifestFile := strings.TrimSpace(runtimedrafts.ManifestFileForStep(task.StepID))
	if writeRoot == "" || draftRoot == "" || manifestFile == "" {
		return snapshot, nil
	}

	manifestPath := filepath.Join(filepath.Clean(writeRoot), manifestFile)
	if info, err := os.Stat(manifestPath); err == nil && !info.IsDir() {
		snapshot.ManifestPresent = true
		snapshot.LastMutation = info.ModTime().UTC()
		if _, _, loadErr := runtimedrafts.Load(writeRoot, manifestFile); loadErr == nil {
			snapshot.ManifestState = "present"
		} else {
			snapshot.ManifestState = "invalid"
		}
	} else if err != nil && !os.IsNotExist(err) {
		return draftArtifactSnapshot{}, err
	}

	walkErr := filepath.WalkDir(filepath.Clean(draftRoot), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		snapshot.DraftFileCount++
		if info, infoErr := d.Info(); infoErr == nil && info.ModTime().After(snapshot.LastMutation) {
			snapshot.LastMutation = info.ModTime().UTC()
		}
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		return draftArtifactSnapshot{}, walkErr
	}
	return snapshot, nil
}

func effectiveCollectMutationTime(lastMutation time.Time, startedAt time.Time) time.Time {
	if lastMutation.IsZero() || lastMutation.Before(startedAt) {
		return startedAt.UTC()
	}
	return lastMutation.UTC()
}

func currentCollectManifestState(writeRoot string) string {
	snapshot, err := scanCollectWriteRoot(writeRoot)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(snapshot.ManifestState)
}

func currentDraftManifestState(task acpruntime.Task) string {
	snapshot, err := scanDraftArtifacts(task)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(snapshot.ManifestState)
}

func scanCollectWriteRoot(writeRoot string) (collectWriteRootSnapshot, error) {
	root := strings.TrimSpace(writeRoot)
	if root == "" {
		return collectWriteRootSnapshot{ManifestState: "missing"}, nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return collectWriteRootSnapshot{ManifestState: "missing"}, nil
		}
		return collectWriteRootSnapshot{}, err
	}
	if !info.IsDir() {
		return collectWriteRootSnapshot{}, fmt.Errorf("write_root is not a directory: %s", root)
	}

	snapshot := collectWriteRootSnapshot{ManifestState: "missing"}
	if walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if info, infoErr := d.Info(); infoErr == nil && info.ModTime().After(snapshot.LastMutation) {
			snapshot.LastMutation = info.ModTime().UTC()
		}
		if rel == "shard-pack-manifest.json" {
			snapshot.ManifestPresent = true
			return nil
		}
		snapshot.AuthoredFileCount++
		return nil
	}); walkErr != nil {
		return collectWriteRootSnapshot{}, walkErr
	}

	if snapshot.ManifestPresent {
		assessment, assessErr := assessRetryManifestAtWriteRoot(root)
		if assessErr == nil && assessment.Rich {
			snapshot.ManifestState = "rich"
		} else {
			snapshot.ManifestState = "invalid"
		}
	}
	return snapshot, nil
}

func terminateProcessWithGrace(process *os.Process) {
	if process == nil {
		return
	}
	if err := signalCommandProcessTree(process, syscall.SIGTERM); err != nil && !isProcessDoneErr(err) {
		_ = killCommandProcessTree(process)
		return
	}
	if collectStallTerminateGrace > 0 {
		time.Sleep(collectStallTerminateGrace)
	}
	_ = killCommandProcessTree(process)
}

func isProcessDoneErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrProcessDone) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "already finished") || strings.Contains(lower, "process already finished") || strings.Contains(lower, "no such process")
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
			if errors.Is(err, io.EOF) {
				return nil
			}
			if isPipeClosedErr(err) {
				return nil
			}
			return err
		}
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
			task.OnOutput(acpruntime.OutputChunk{
				Stream: stream,
				Text:   line,
			})
			continue
		}
		remaining := acpruntime.RuntimeOutputStreamHardCapBytes - budget.forwardedBytes
		if remaining > 0 {
			trimmed := line
			if len([]byte(trimmed)) > remaining {
				trimmedBytes := []byte(trimmed)
				if remaining < len(trimmedBytes) {
					trimmed = string(trimmedBytes[:remaining])
				}
			}
			trimmed = strings.TrimSpace(trimmed)
			if trimmed != "" {
				task.OnOutput(acpruntime.OutputChunk{
					Stream: stream,
					Text:   trimmed,
				})
			}
		}
		budget.truncated = true
		task.OnOutput(acpruntime.OutputChunk{
			Stream:    stream,
			Truncated: true,
			Text:      fmt.Sprintf("%s output truncated after %d bytes (internal safeguard)", stream, acpruntime.RuntimeOutputStreamHardCapBytes),
		})
	}
}

func (c *retryDiagnosticContext) absorbDiagnostic(diagnostic collectStallDiagnostic) {
	if c == nil {
		return
	}
	if diagnostic.StallPhase != "" {
		c.LastStallPhase = diagnostic.StallPhase
	}
	if state := strings.TrimSpace(diagnostic.ManifestState); state != "" {
		c.ManifestStateBeforeRetry = state
	}
	if diagnostic.AuthoredFileCount > 0 {
		c.AuthoredFileCount = diagnostic.AuthoredFileCount
	}
	if !diagnostic.LastPipeActivity.IsZero() {
		c.LastPipeActivity = diagnostic.LastPipeActivity.UTC()
	}
	if !diagnostic.LastWriteRootMutation.IsZero() {
		c.LastWriteRootMutation = diagnostic.LastWriteRootMutation.UTC()
	}
}

func waitForCommandStreams(stdoutPipe io.ReadCloser, stderrPipe io.ReadCloser, streamDone <-chan struct{}, timeout time.Duration) {
	if timeout <= 0 {
		<-streamDone
		return
	}
	select {
	case <-streamDone:
		return
	case <-time.After(timeout):
		closeCommandPipe(stdoutPipe)
		closeCommandPipe(stderrPipe)
		select {
		case <-streamDone:
		case <-time.After(timeout):
		}
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
	if timeout <= 0 {
		return <-done
	}
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

func closeCommandPipe(pipe io.Closer) {
	if pipe == nil {
		return
	}
	_ = pipe.Close()
}
