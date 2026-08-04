package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func (s *Service) ReconcileStaleRunsAfterRestart() {
	s.recoverStaleRunsAfterRestart()
}

func (s *Service) StartAsyncRun(ctx context.Context, request RunRequest) (string, error) {
	_ = s.cleanupRunLogs()
	if request.Pipeline == PipelineQA && strings.TrimSpace(request.Question) == "" {
		return "", fmt.Errorf("question is required for qa runs")
	}
	intent := request.Intent
	if intent == "" {
		intent = RunIntentStart
	}
	if intent != RunIntentStart && intent != RunIntentQueue {
		return "", fmt.Errorf("unsupported run intent %q", intent)
	}
	if intent == RunIntentQueue && request.Pipeline != PipelineRefresh {
		return "", ErrQueueUnsupported
	}
	request.Intent = intent
	resolvedStepProviders, err := s.ResolveStepProviderProfile(request.Workspace.Manifest)
	if err != nil {
		return "", err
	}
	resolvedProviderModels, err := s.ResolveProviderModelProfile(request.Workspace.Manifest)
	if err != nil {
		return "", err
	}
	request.ProviderModels = &resolvedProviderModels
	runID := s.nextRunID()
	now := s.clock().UTC()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return "", ErrServiceClosed
	}
	queuedRecord := func() runRecord {
		progress := newRunProgress(request.Pipeline, now, strings.TrimSpace(request.ResumeFromStep))
		progress.Phase = "queued"
		var retry *RetryLineage
		if strings.TrimSpace(request.RetryParentRunID) != "" {
			retry = &RetryLineage{ParentRunID: strings.TrimSpace(request.RetryParentRunID), Reason: strings.TrimSpace(request.RetryReason), RequestedStep: retryRequestedStep(request), EffectiveStartStep: strings.TrimSpace(request.ResumeFromStep), RequestedScopes: retryRequestedScopes(request), EffectiveScopes: append([]string(nil), request.RetryScopes...), ReusedInputs: append([]string(nil), request.RetryReusedInputs...)}
		}
		return runRecord{
			info: RunInfo{
				RunID:                runID,
				Pipeline:             string(request.Pipeline),
				Status:               RunStatusQueued,
				StartedAt:            now,
				Question:             strings.TrimSpace(request.Question),
				RuntimeMode:          s.runtimeMode,
				StepProviders:        resolvedStepProviders.Effective.StringMap(),
				ProviderModels:       resolvedProviderModels.Effective,
				ProviderModelSources: resolvedProviderModels.Source,
				Progress:             progress,
				Retry:                retry,
			},
		}
	}
	if s.isActiveRunLocked() {
		if intent != RunIntentQueue {
			s.mu.Unlock()
			return "", ErrRunActive
		}
		if s.pendingRun != nil {
			superseded, ok := s.supersededRunRecordLocked(s.pendingRun.runID, runID)
			if !ok {
				s.mu.Unlock()
				return "", fmt.Errorf("pending run %q is missing from registry", s.pendingRun.runID)
			}
			if err := s.upsertRunsLocked(queuedRecord(), superseded); err != nil {
				s.mu.Unlock()
				return "", err
			}
			s.pendingRun = &pendingRun{runID: runID, request: request, queuedAt: now}
			s.mu.Unlock()
			return runID, nil
		}
		if err := s.upsertRunLocked(queuedRecord()); err != nil {
			s.mu.Unlock()
			return "", err
		}
		s.pendingRun = &pendingRun{
			runID:    runID,
			request:  request,
			queuedAt: now,
		}
		s.mu.Unlock()
		return runID, nil
	}
	if err := s.upsertRunLocked(queuedRecord()); err != nil {
		s.mu.Unlock()
		return "", err
	}
	s.activeRunID = runID
	s.mu.Unlock()

	s.launchAsyncRun(ctx, runID, request)
	s.appendRunLog(runID, RunLogEntry{
		Timestamp: now,
		Level:     RunLogLevelInfo,
		Message:   "run queued",
		Fields: map[string]any{
			"pipeline": string(request.Pipeline),
		},
	})

	return runID, nil
}

func (s *Service) CancelRun(runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ErrRunNotFound
	}

	now := s.clock().UTC()
	var cancelFn context.CancelFunc

	s.mu.Lock()
	record, ok := s.runs[runID]
	if !ok {
		s.mu.Unlock()
		return ErrRunNotFound
	}

	switch record.info.Status {
	case RunStatusSucceeded, RunStatusFailed, RunStatusCanceled:
		s.mu.Unlock()
		return ErrRunNotCancelable
	case RunStatusQueued:
		if s.pendingRun != nil && s.pendingRun.runID == runID {
			canceledInfo := cloneRunInfo(record.info)
			canceledInfo.Status = RunStatusCanceled
			canceledInfo.ErrorCode = runErrorCodeCanceled
			canceledInfo.Error = fmt.Sprintf("run canceled while queued (previous_status=%s)", RunStatusQueued)
			canceledInfo.FinishedAt = cloneTimePointer(&now)
			copiedArtifacts := append([]Artifact(nil), record.artifacts...)
			if err := s.upsertRunLocked(runRecord{
				info:      canceledInfo,
				artifacts: copiedArtifacts,
			}); err != nil {
				s.mu.Unlock()
				return err
			}
			s.pendingRun = nil
			if s.cancelRequests != nil {
				delete(s.cancelRequests, runID)
			}
			s.mu.Unlock()
			s.appendRunLog(runID, RunLogEntry{
				Timestamp: now,
				Level:     RunLogLevelWarning,
				Message:   "run canceled while queued",
				Fields: map[string]any{
					"error_code":      runErrorCodeCanceled,
					"previous_status": string(RunStatusQueued),
				},
			})
			return nil
		}
	}

	if runID != s.activeRunID {
		s.mu.Unlock()
		return ErrRunNotCancelable
	}

	if s.cancelRequests == nil {
		s.cancelRequests = map[string]struct{}{}
	}
	s.cancelRequests[runID] = struct{}{}
	if s.runCancels != nil {
		cancelFn = s.runCancels[runID]
	}
	s.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}

	s.appendRunLog(runID, RunLogEntry{
		Timestamp: now,
		Level:     RunLogLevelInfo,
		Message:   "run cancellation requested",
		Fields: map[string]any{
			"error_code": runErrorCodeCanceled,
		},
	})
	return nil
}

func (s *Service) GetRun(runID string) (RunInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.runs[runID]
	if !ok {
		return RunInfo{}, false
	}
	return cloneRunInfo(record.info), true
}

func (s *Service) GetRunArtifacts(runID string) ([]Artifact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.runs[runID]
	if !ok {
		return nil, false
	}

	artifacts := append([]Artifact(nil), record.artifacts...)
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Kind == artifacts[j].Kind {
			return artifacts[i].Path < artifacts[j].Path
		}
		return artifacts[i].Kind < artifacts[j].Kind
	})
	return artifacts, true
}

func (s *Service) GetRunPermissions(runID string) ([]acpruntime.PermissionRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.runs[runID]
	if !ok {
		return nil, false
	}
	return clonePermissionRequests(record.info.PendingPermissions), true
}

func (s *Service) GetRunLogs(runID string, cursor int, limit int) (RunLogPage, bool, error) {
	s.mu.RLock()
	_, ok := s.runs[runID]
	s.mu.RUnlock()
	if !ok {
		return RunLogPage{}, false, nil
	}

	page, err := s.queryRunLogs(runID, cursor, limit)
	if err != nil {
		return RunLogPage{}, true, err
	}
	return page, true, nil
}

func (s *Service) ListRuns(limit int) []RunInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]RunInfo, 0, len(s.runs))
	for _, record := range s.runs {
		infos = append(infos, cloneRunInfo(record.info))
	}
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].StartedAt.Equal(infos[j].StartedAt) {
			return infos[i].RunID > infos[j].RunID
		}
		return infos[i].StartedAt.After(infos[j].StartedAt)
	})

	if limit <= 0 || limit >= len(infos) {
		return infos
	}
	return infos[:limit]
}

func (s *Service) Coordination() RunCoordination {
	s.mu.RLock()
	defer s.mu.RUnlock()
	coordination := RunCoordination{}
	if s.isActiveRunLocked() {
		coordination.ActiveRunID = s.activeRunID
	}
	if s.pendingRun != nil {
		coordination.Pending = &PendingRunInfo{RunID: s.pendingRun.runID, Pipeline: string(s.pendingRun.request.Pipeline)}
	}
	return coordination
}

func (s *Service) HistoryDiagnostics() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string{}, s.historyRecoveryDiagnostics...)
}

func (s *Service) HasInFlightRun() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.isActiveRunLocked() || s.pendingRun != nil
}

func (s *Service) nextRunID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		s.runIDSequence++
		runID := fmt.Sprintf("run_%s_%03d", s.clock().UTC().Format("20060102_150405"), s.runIDSequence)
		if _, exists := s.runs[runID]; !exists {
			return runID
		}
	}
}

func (s *Service) storeRun(record runRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.upsertRunLocked(record)
}

func (s *Service) terminalizeActiveRunAfterUnexpectedExit(runID string, err error, logMessage string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" || err == nil {
		return nil
	}
	errorCode, errorMessage := s.classifyRunFailure(runID, err)
	if strings.TrimSpace(errorCode) == "" {
		switch {
		case errors.Is(err, context.Canceled):
			errorCode = runErrorCodeCanceled
		case errors.Is(err, context.DeadlineExceeded):
			errorCode = string(acpruntime.ErrorCodeRuntimeTimeout)
		default:
			errorCode = "internal_failure"
		}
	}
	if strings.TrimSpace(errorMessage) == "" {
		errorMessage = err.Error()
	}
	now := s.clock().UTC()

	updated := false
	s.mu.Lock()
	record, ok := s.runs[runID]
	if ok && record != nil && (record.info.Status == RunStatusQueued || record.info.Status == RunStatusRunning) {
		failedInfo := cloneRunInfo(record.info)
		failedInfo.Status = terminalStatusForErrorCode(errorCode)
		failedInfo.ErrorCode = errorCode
		failedInfo.Error = errorMessage
		failedInfo.FinishedAt = cloneTimePointer(&now)
		if persistErr := s.upsertRunLocked(runRecord{
			info:      failedInfo,
			artifacts: append([]Artifact(nil), record.artifacts...),
		}); persistErr != nil {
			s.mu.Unlock()
			return persistErr
		}
		updated = true
	}
	s.mu.Unlock()
	if !updated {
		return nil
	}

	message := strings.TrimSpace(logMessage)
	if message == "" {
		message = "run failed: unexpected exit"
	}
	s.appendRunLog(runID, RunLogEntry{
		Timestamp: now,
		Level:     RunLogLevelError,
		Message:   message,
		Fields: map[string]any{
			"error_code": errorCode,
			"error":      errorMessage,
		},
	})
	return nil
}

func (s *Service) loadExistingRunRecord(runID string) (runRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.runs[runID]
	if !ok || record == nil {
		return runRecord{}, false
	}
	return cloneRunRecord(*record), true
}

func (s *Service) launchAsyncRun(ctx context.Context, runID string, request RunRequest) {
	runCtx, cancel := context.WithCancel(ctx)

	shouldCancelImmediately := false
	s.mu.Lock()
	if s.runCancels == nil {
		s.runCancels = map[string]context.CancelFunc{}
	}
	if s.closed {
		s.mu.Unlock()
		cancel()
		_ = s.terminalizeActiveRunAfterUnexpectedExit(runID, context.Canceled, "run failed: service shutdown")
		s.finishAsyncRun(ctx, runID)
		return
	}
	s.runCancels[runID] = cancel
	if _, requested := s.cancelRequests[runID]; requested {
		shouldCancelImmediately = true
	}
	s.mu.Unlock()

	if shouldCancelImmediately {
		cancel()
	}

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicErr := fmt.Errorf("run panic: %v", recovered)
				_ = s.terminalizeActiveRunAfterUnexpectedExit(runID, panicErr, "run failed: panic")
			}
			s.finishAsyncRun(ctx, runID)
		}()
		_, _, runErr := s.runWithID(runCtx, request, runID)
		if runErr != nil {
			_ = s.terminalizeActiveRunAfterUnexpectedExit(runID, runErr, "run failed: async execution")
		}
	}()
}

func (s *Service) finishAsyncRun(ctx context.Context, runID string) {
	var next *pendingRun
	var cancelFn context.CancelFunc

	s.mu.Lock()
	if s.runCancels != nil {
		cancelFn = s.runCancels[runID]
		delete(s.runCancels, runID)
	}
	if s.cancelRequests != nil {
		delete(s.cancelRequests, runID)
	}
	if s.activeRunID == runID {
		record := s.runs[runID]
		if record != nil && isTerminalRunStatus(record.info.Status) {
			s.activeRunID = ""
		}
	}
	if !s.closed && strings.TrimSpace(s.activeRunID) == "" && s.pendingRun != nil {
		next = s.pendingRun
		s.pendingRun = nil
		s.activeRunID = next.runID
	}
	s.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}
	if next != nil {
		s.launchAsyncRun(ctx, next.runID, next.request)
	}
}

func (s *Service) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.Shutdown(ctx)
}

func (s *Service) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	now := s.clock().UTC()
	var activeRunID string
	var cancelFns []context.CancelFunc
	var pendingPersistenceErr error

	s.mu.Lock()
	s.closed = true
	activeRunID = s.activeRunID
	if s.pendingRun != nil {
		pendingPersistenceErr = s.failQueuedRunLocked(s.pendingRun.runID, now, runErrorCodeCanceled, "service shutdown canceled queued run")
		if pendingPersistenceErr == nil {
			s.pendingRun = nil
		}
	}
	for _, cancel := range s.runCancels {
		if cancel != nil {
			cancelFns = append(cancelFns, cancel)
		}
	}
	s.mu.Unlock()

	for _, cancel := range cancelFns {
		cancel()
	}
	if strings.TrimSpace(activeRunID) == "" {
		return pendingPersistenceErr
	}
	if err := s.waitForRunTerminal(ctx, activeRunID); err != nil {
		terminalErr := s.terminalizeActiveRunAfterUnexpectedExit(activeRunID, context.Canceled, "run failed: service shutdown")
		return errors.Join(pendingPersistenceErr, err, terminalErr)
	}
	return pendingPersistenceErr
}

func (s *Service) failQueuedRunLocked(runID string, finishedAt time.Time, errorCode string, errorMessage string) error {
	record, ok := s.runs[runID]
	if !ok || record == nil || record.info.Status != RunStatusQueued {
		return nil
	}
	info := cloneRunInfo(record.info)
	info.Status = terminalStatusForErrorCode(errorCode)
	info.ErrorCode = errorCode
	info.Error = strings.TrimSpace(errorMessage)
	if info.Error == "" {
		info.Error = "run canceled"
	}
	info.FinishedAt = cloneTimePointer(&finishedAt)
	return s.upsertRunLocked(runRecord{
		info:      info,
		artifacts: append([]Artifact(nil), record.artifacts...),
	})
}

func (s *Service) waitForRunTerminal(ctx context.Context, runID string) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		s.mu.RLock()
		record, ok := s.runs[runID]
		terminal := !ok || (record != nil && isTerminalRunStatus(record.info.Status))
		_, ownsCancel := s.runCancels[runID]
		ownsActiveSlot := s.activeRunID == runID
		s.mu.RUnlock()
		if terminal && !ownsCancel && !ownsActiveSlot {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) isActiveRunLocked() bool {
	if strings.TrimSpace(s.activeRunID) == "" {
		return false
	}
	record, ok := s.runs[s.activeRunID]
	if !ok {
		return false
	}
	return record.info.Status == RunStatusQueued || record.info.Status == RunStatusRunning
}

func (s *Service) supersededRunRecordLocked(oldRunID string, newRunID string) (runRecord, bool) {
	record, ok := s.runs[oldRunID]
	if !ok || record == nil {
		return runRecord{}, false
	}
	finishedAt := s.clock().UTC()
	superseded := cloneRunInfo(record.info)
	superseded.Status = RunStatusCanceled
	superseded.ErrorCode = runErrorCodeSuperseded
	superseded.Error = fmt.Sprintf("run superseded by newer event %q (last-event-wins)", newRunID)
	superseded.SupersededByRunID = newRunID
	superseded.FinishedAt = cloneTimePointer(&finishedAt)
	return runRecord{
		info:      superseded,
		artifacts: append([]Artifact(nil), record.artifacts...),
	}, true
}

func (s *Service) upsertRunLocked(record runRecord) error {
	return s.upsertRunsLocked(record)
}

func (s *Service) upsertRunsLocked(records ...runRecord) error {
	candidate := cloneRunRegistry(s.runs)
	for _, record := range records {
		cloned := cloneRunRecord(record)
		candidate[cloned.info.RunID] = &cloned
	}
	trimRunRegistry(candidate, s.historyRetention)
	if err := s.persistHistorySnapshotLocked(candidate); err != nil {
		s.recordHistoryPersistenceFailureLocked(err)
		return err
	}
	s.runs = candidate
	return nil
}

func trimRunRegistry(runs map[string]*runRecord, configuredRetention int) {
	retention := configuredRetention
	if retention <= 0 {
		retention = runHistoryRetention
	}
	if len(runs) <= retention {
		return
	}

	runIDs := make([]string, 0, len(runs))
	for runID := range runs {
		runIDs = append(runIDs, runID)
	}
	sort.Slice(runIDs, func(i, j int) bool {
		left := runs[runIDs[i]].info
		right := runs[runIDs[j]].info
		if left.StartedAt.Equal(right.StartedAt) {
			return left.RunID < right.RunID
		}
		return left.StartedAt.Before(right.StartedAt)
	})
	removeCount := len(runs) - retention
	for idx := 0; idx < removeCount; idx++ {
		delete(runs, runIDs[idx])
	}
}

func (s *Service) persistHistorySnapshotLocked(runs map[string]*runRecord) error {
	if !s.historyEnabled {
		s.lastHistoryPersistenceErr = nil
		return nil
	}

	items := make([]runHistoryItem, 0, len(runs))
	records := make([]runRecord, 0, len(runs))
	for _, record := range runs {
		records = append(records, cloneRunRecord(*record))
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].info.StartedAt.Equal(records[j].info.StartedAt) {
			return records[i].info.RunID < records[j].info.RunID
		}
		return records[i].info.StartedAt.Before(records[j].info.StartedAt)
	})

	retention := s.historyRetention
	if retention <= 0 {
		retention = runHistoryRetention
	}
	if len(records) > retention {
		records = records[len(records)-retention:]
	}

	for _, record := range records {
		items = append(items, runRecordToHistoryItem(record))
	}

	snapshot := runHistorySnapshot{
		Version: runHistoryVersion,
		Items:   items,
	}
	encoded, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run history: %w", err)
	}
	encoded = append(encoded, '\n')
	writeFile := s.historyWriteFile
	if writeFile == nil {
		writeFile = func(root workspace.Root, relPath string, content []byte) error {
			return root.WriteFileAtomic(relPath, content)
		}
	}
	if err := writeFile(s.historyWorkspace, runHistoryPath, encoded); err != nil {
		return fmt.Errorf("persist run history: %w", err)
	}
	if err := writeFile(s.historyWorkspace, runHistoryPath+".last-good", encoded); err != nil {
		diagnostic := fmt.Errorf("persist run history last-good: %w", err)
		s.lastHistoryPersistenceErr = diagnostic
		s.addHistoryDiagnosticLocked(diagnostic.Error())
		return nil
	}
	s.lastHistoryPersistenceErr = nil
	return nil
}

func (s *Service) recordHistoryPersistenceFailureLocked(err error) {
	if err == nil {
		return
	}
	s.lastHistoryPersistenceErr = err
	s.addHistoryDiagnosticLocked(err.Error())
}

func (s *Service) addHistoryDiagnosticLocked(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	for _, existing := range s.historyRecoveryDiagnostics {
		if existing == message {
			return
		}
	}
	s.historyRecoveryDiagnostics = append(s.historyRecoveryDiagnostics, message)
	if len(s.historyRecoveryDiagnostics) > historyDiagnosticsLimit {
		s.historyRecoveryDiagnostics = append([]string(nil), s.historyRecoveryDiagnostics[len(s.historyRecoveryDiagnostics)-historyDiagnosticsLimit:]...)
	}
}

func runRecordToHistoryItem(record runRecord) runHistoryItem {
	item := runHistoryItem{
		RunID:                record.info.RunID,
		Pipeline:             record.info.Pipeline,
		Status:               record.info.Status,
		StartedAt:            record.info.StartedAt.UTC().Format(time.RFC3339),
		CurrentStep:          record.info.CurrentStep,
		RuntimeMode:          record.info.RuntimeMode,
		Question:             record.info.Question,
		StepProviders:        cloneStringMap(record.info.StepProviders),
		ProviderModels:       cloneProviderModelValues(record.info.ProviderModels),
		ProviderModelSources: cloneProviderModelSources(record.info.ProviderModelSources),
		Warnings:             append([]string(nil), record.info.Warnings...),
		PendingPermissions:   clonePermissionRequests(record.info.PendingPermissions),
		ErrorCode:            record.info.ErrorCode,
		Error:                record.info.Error,
		SupersededByRunID:    record.info.SupersededByRunID,
		RefreshSummary:       cloneRefreshSummary(record.info.RefreshSummary),
		Progress:             cloneRunProgress(record.info.Progress),
		Retry:                cloneRetryLineage(record.info.Retry),
		Artifacts:            append([]Artifact(nil), record.artifacts...),
	}
	if record.info.FinishedAt != nil {
		finished := record.info.FinishedAt.UTC().Format(time.RFC3339)
		item.FinishedAt = &finished
	}
	return item
}

func historyItemToRunRecord(item runHistoryItem) (runRecord, bool) {
	startedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(item.StartedAt))
	if err != nil {
		return runRecord{}, false
	}
	var finishedAt *time.Time
	if item.FinishedAt != nil && strings.TrimSpace(*item.FinishedAt) != "" {
		parsedFinishedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(*item.FinishedAt))
		if parseErr != nil {
			return runRecord{}, false
		}
		finishedAt = &parsedFinishedAt
	}
	return runRecord{
		info: RunInfo{
			RunID:                item.RunID,
			Pipeline:             item.Pipeline,
			Status:               item.Status,
			StartedAt:            startedAt.UTC(),
			FinishedAt:           finishedAt,
			Question:             item.Question,
			CurrentStep:          item.CurrentStep,
			RuntimeMode:          item.RuntimeMode,
			StepProviders:        cloneStringMap(item.StepProviders),
			ProviderModels:       cloneProviderModelValues(item.ProviderModels),
			ProviderModelSources: cloneProviderModelSources(item.ProviderModelSources),
			Warnings:             append([]string(nil), item.Warnings...),
			PendingPermissions:   clonePermissionRequests(item.PendingPermissions),
			ErrorCode:            item.ErrorCode,
			Error:                item.Error,
			SupersededByRunID:    item.SupersededByRunID,
			RefreshSummary:       cloneRefreshSummary(item.RefreshSummary),
			Progress:             cloneRunProgress(item.Progress),
			Retry:                cloneRetryLineage(item.Retry),
		},
		artifacts: append([]Artifact(nil), item.Artifacts...),
	}, true
}

func cloneRefreshSummary(value *RefreshSummary) *RefreshSummary {
	if value == nil {
		return nil
	}
	clone := *value
	clone.ReasonCodes = append([]string(nil), value.ReasonCodes...)
	return &clone
}

func cloneRunProgress(value *RunProgress) *RunProgress {
	if value == nil {
		return nil
	}
	clone := *value
	clone.CurrentScopes = append([]string(nil), value.CurrentScopes...)
	return &clone
}

func cloneRetryLineage(value *RetryLineage) *RetryLineage {
	if value == nil {
		return nil
	}
	clone := *value
	clone.RequestedScopes = append([]string(nil), value.RequestedScopes...)
	clone.EffectiveScopes = append([]string(nil), value.EffectiveScopes...)
	clone.ReusedInputs = append([]string(nil), value.ReusedInputs...)
	return &clone
}

func cloneRunInfo(value RunInfo) RunInfo {
	clone := value
	clone.FinishedAt = cloneTimePointer(value.FinishedAt)
	clone.StepProviders = cloneStringMap(value.StepProviders)
	clone.ProviderModels = cloneProviderModelValues(value.ProviderModels)
	clone.ProviderModelSources = cloneProviderModelSources(value.ProviderModelSources)
	clone.Warnings = append([]string(nil), value.Warnings...)
	clone.PendingPermissions = clonePermissionRequests(value.PendingPermissions)
	clone.RefreshSummary = cloneRefreshSummary(value.RefreshSummary)
	clone.Progress = cloneRunProgress(value.Progress)
	clone.Retry = cloneRetryLineage(value.Retry)
	return clone
}

func cloneRunRecord(value runRecord) runRecord {
	return runRecord{
		info:      cloneRunInfo(value.info),
		artifacts: append([]Artifact(nil), value.artifacts...),
	}
}

func cloneRunRegistry(values map[string]*runRecord) map[string]*runRecord {
	clone := make(map[string]*runRecord, len(values))
	for runID, record := range values {
		if record == nil {
			continue
		}
		copied := cloneRunRecord(*record)
		clone[runID] = &copied
	}
	return clone
}

func clonePermissionRequests(values []acpruntime.PermissionRequest) []acpruntime.PermissionRequest {
	if values == nil {
		return nil
	}
	clone := make([]acpruntime.PermissionRequest, len(values))
	for idx, value := range values {
		clone[idx] = value
		if value.Decision != nil {
			decision := *value.Decision
			clone[idx].Decision = &decision
		}
	}
	return clone
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func terminalStatusForErrorCode(errorCode string) RunStatus {
	if strings.TrimSpace(errorCode) == runErrorCodeCanceled {
		return RunStatusCanceled
	}
	return RunStatusFailed
}

func isTerminalRunStatus(status RunStatus) bool {
	return status == RunStatusSucceeded || status == RunStatusFailed || status == RunStatusCanceled
}

func classifyExecutionError(err error) (code string, message string) {
	message = strings.TrimSpace(err.Error())
	if runtimeCode, runtimeMessage, ok := acpruntime.ClassifyError(err); ok {
		if strings.TrimSpace(runtimeMessage) != "" {
			message = runtimeMessage
		}
		return runtimeCode, message
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return string(acpruntime.ErrorCodeRuntimeTimeout), message
	case errors.Is(err, context.Canceled):
		return runErrorCodeCanceled, message
	}
	return "", message
}

func (s *Service) classifyRunFailure(runID string, err error) (string, string) {
	if s.isCancelRequested(runID) {
		if errors.Is(err, context.Canceled) {
			return runErrorCodeCanceled, "run canceled by request"
		}
		message := strings.TrimSpace(err.Error())
		if message == "" {
			message = "run canceled by request"
		} else {
			message = fmt.Sprintf("run canceled by request (%s)", message)
		}
		return runErrorCodeCanceled, message
	}
	return classifyExecutionError(err)
}

func (s *Service) isCancelRequested(runID string) bool {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.cancelRequests[runID]
	return ok
}
