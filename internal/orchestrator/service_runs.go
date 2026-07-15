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
	runID := s.nextRunID()
	now := s.clock().UTC()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return "", ErrServiceClosed
	}
	storeQueuedRun := func() error {
		return s.upsertRunLocked(runRecord{
			info: RunInfo{
				RunID:         runID,
				Pipeline:      string(request.Pipeline),
				Status:        RunStatusQueued,
				StartedAt:     now,
				Question:      strings.TrimSpace(request.Question),
				RuntimeMode:   s.runtimeMode,
				StepProviders: resolvedStepProviders.Effective.StringMap(),
			},
		})
	}
	if s.isActiveRunLocked() {
		if intent != RunIntentQueue {
			s.mu.Unlock()
			return "", ErrRunActive
		}
		if s.pendingRun != nil {
			if err := storeQueuedRun(); err != nil {
				s.mu.Unlock()
				return "", err
			}
			if err := s.markRunSupersededLocked(s.pendingRun.runID, runID); err != nil {
				s.mu.Unlock()
				return "", err
			}
			s.pendingRun = &pendingRun{runID: runID, request: request, queuedAt: now}
			s.mu.Unlock()
			return runID, nil
		}
		if err := storeQueuedRun(); err != nil {
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
	if err := storeQueuedRun(); err != nil {
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
			s.pendingRun = nil
			if s.cancelRequests != nil {
				delete(s.cancelRequests, runID)
			}
			failedInfo := record.info
			failedInfo.Status = RunStatusFailed
			failedInfo.ErrorCode = runErrorCodeCanceled
			failedInfo.Error = fmt.Sprintf("run canceled while queued (previous_status=%s)", RunStatusQueued)
			failedInfo.FinishedAt = &now
			copiedArtifacts := append([]Artifact(nil), record.artifacts...)
			if err := s.upsertRunLocked(runRecord{
				info:      failedInfo,
				artifacts: copiedArtifacts,
			}); err != nil {
				s.mu.Unlock()
				return err
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
	return record.info, true
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
	return append([]acpruntime.PermissionRequest(nil), record.info.PendingPermissions...), true
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
		info := record.info
		info.Warnings = append([]string(nil), record.info.Warnings...)
		info.PendingPermissions = append([]acpruntime.PermissionRequest(nil), record.info.PendingPermissions...)
		infos = append(infos, info)
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
	coordination := RunCoordination{ActiveRunID: s.activeRunID}
	if s.pendingRun != nil {
		coordination.Pending = &PendingRunInfo{RunID: s.pendingRun.runID, Pipeline: string(s.pendingRun.request.Pipeline)}
	}
	return coordination
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

func (s *Service) terminalizeActiveRunAfterUnexpectedExit(runID string, err error, logMessage string) {
	runID = strings.TrimSpace(runID)
	if runID == "" || err == nil {
		return
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
		failedInfo := record.info
		failedInfo.Status = RunStatusFailed
		failedInfo.ErrorCode = errorCode
		failedInfo.Error = errorMessage
		failedInfo.FinishedAt = &now
		s.upsertRunLocked(runRecord{
			info:      failedInfo,
			artifacts: append([]Artifact(nil), record.artifacts...),
		})
		updated = true
	}
	s.mu.Unlock()
	if !updated {
		return
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
}

func (s *Service) loadExistingRunRecord(runID string) (runRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.runs[runID]
	if !ok || record == nil {
		return runRecord{}, false
	}
	return runRecord{
		info: RunInfo{
			RunID:              record.info.RunID,
			Pipeline:           record.info.Pipeline,
			Status:             record.info.Status,
			StartedAt:          record.info.StartedAt,
			FinishedAt:         record.info.FinishedAt,
			Question:           record.info.Question,
			CurrentStep:        record.info.CurrentStep,
			RuntimeMode:        record.info.RuntimeMode,
			StepProviders:      cloneStringMap(record.info.StepProviders),
			Warnings:           append([]string(nil), record.info.Warnings...),
			PendingPermissions: append([]acpruntime.PermissionRequest(nil), record.info.PendingPermissions...),
			ErrorCode:          record.info.ErrorCode,
			Error:              record.info.Error,
			SupersededByRunID:  record.info.SupersededByRunID,
		},
		artifacts: append([]Artifact(nil), record.artifacts...),
	}, true
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
		s.terminalizeActiveRunAfterUnexpectedExit(runID, context.Canceled, "run failed: service shutdown")
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
				s.terminalizeActiveRunAfterUnexpectedExit(runID, panicErr, "run failed: panic")
			}
			s.finishAsyncRun(ctx, runID)
		}()
		_, _, _ = s.runWithID(runCtx, request, runID)
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
		s.activeRunID = ""
	}
	if !s.closed && s.pendingRun != nil {
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

	s.mu.Lock()
	s.closed = true
	activeRunID = s.activeRunID
	if s.pendingRun != nil {
		s.failQueuedRunLocked(s.pendingRun.runID, now, runErrorCodeCanceled, "service shutdown canceled queued run")
		s.pendingRun = nil
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
		return nil
	}
	if err := s.waitForRunTerminal(ctx, activeRunID); err != nil {
		s.terminalizeActiveRunAfterUnexpectedExit(activeRunID, context.Canceled, "run failed: service shutdown")
		return err
	}
	return nil
}

func (s *Service) failQueuedRunLocked(runID string, finishedAt time.Time, errorCode string, errorMessage string) {
	record, ok := s.runs[runID]
	if !ok || record == nil || record.info.Status != RunStatusQueued {
		return
	}
	info := record.info
	info.Status = RunStatusFailed
	info.ErrorCode = errorCode
	info.Error = strings.TrimSpace(errorMessage)
	if info.Error == "" {
		info.Error = "run canceled"
	}
	info.FinishedAt = &finishedAt
	_ = s.upsertRunLocked(runRecord{
		info:      info,
		artifacts: append([]Artifact(nil), record.artifacts...),
	})
}

func (s *Service) waitForRunTerminal(ctx context.Context, runID string) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		info, ok := s.GetRun(runID)
		if !ok || info.Status == RunStatusSucceeded || info.Status == RunStatusFailed || info.Status == RunStatusCanceled {
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

func (s *Service) markRunSupersededLocked(oldRunID string, newRunID string) error {
	record, ok := s.runs[oldRunID]
	if !ok {
		return nil
	}
	finishedAt := s.clock().UTC()
	superseded := record.info
	superseded.Status = RunStatusCanceled
	superseded.ErrorCode = runErrorCodeSuperseded
	superseded.Error = fmt.Sprintf("run superseded by newer event %q (last-event-wins)", newRunID)
	superseded.SupersededByRunID = newRunID
	superseded.FinishedAt = &finishedAt
	return s.upsertRunLocked(runRecord{
		info:      superseded,
		artifacts: record.artifacts,
	})
}

func (s *Service) upsertRunLocked(record runRecord) error {
	s.runs[record.info.RunID] = &record
	s.trimRunRegistryLocked()
	if err := s.persistHistoryLocked(); err != nil {
		s.lastHistoryPersistenceErr = err
		s.addHistoryPersistenceWarningLocked(record.info.RunID, err)
		return err
	}
	s.lastHistoryPersistenceErr = nil
	return nil
}

func (s *Service) trimRunRegistryLocked() {
	retention := s.historyRetention
	if retention <= 0 {
		retention = runHistoryRetention
	}
	if len(s.runs) <= retention {
		return
	}

	runIDs := make([]string, 0, len(s.runs))
	for runID := range s.runs {
		runIDs = append(runIDs, runID)
	}
	sort.Slice(runIDs, func(i, j int) bool {
		left := s.runs[runIDs[i]].info
		right := s.runs[runIDs[j]].info
		if left.StartedAt.Equal(right.StartedAt) {
			return left.RunID < right.RunID
		}
		return left.StartedAt.Before(right.StartedAt)
	})
	removeCount := len(s.runs) - retention
	for idx := 0; idx < removeCount; idx++ {
		delete(s.runs, runIDs[idx])
	}
}

func (s *Service) persistHistoryLocked() error {
	if !s.historyEnabled {
		return nil
	}

	items := make([]runHistoryItem, 0, len(s.runs))
	records := make([]runRecord, 0, len(s.runs))
	for _, record := range s.runs {
		records = append(records, runRecord{
			info:      record.info,
			artifacts: append([]Artifact(nil), record.artifacts...),
		})
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
	if err := s.historyWorkspace.WriteFileAtomicWithLastGood(runHistoryPath, encoded); err != nil {
		return fmt.Errorf("persist run history: %w", err)
	}
	return nil
}

func (s *Service) addHistoryPersistenceWarningLocked(runID string, err error) {
	if err == nil {
		return
	}
	record, ok := s.runs[runID]
	if !ok || record == nil {
		return
	}
	warning := fmt.Sprintf("run history persistence failed: %v", err)
	for _, existing := range record.info.Warnings {
		if existing == warning {
			return
		}
	}
	record.info.Warnings = append(record.info.Warnings, warning)
}

func runRecordToHistoryItem(record runRecord) runHistoryItem {
	item := runHistoryItem{
		RunID:              record.info.RunID,
		Pipeline:           record.info.Pipeline,
		Status:             record.info.Status,
		StartedAt:          record.info.StartedAt.UTC().Format(time.RFC3339),
		CurrentStep:        record.info.CurrentStep,
		RuntimeMode:        record.info.RuntimeMode,
		Question:           record.info.Question,
		StepProviders:      cloneStringMap(record.info.StepProviders),
		Warnings:           append([]string(nil), record.info.Warnings...),
		PendingPermissions: append([]acpruntime.PermissionRequest(nil), record.info.PendingPermissions...),
		ErrorCode:          record.info.ErrorCode,
		Error:              record.info.Error,
		SupersededByRunID:  record.info.SupersededByRunID,
		RefreshSummary:     cloneRefreshSummary(record.info.RefreshSummary),
		Artifacts:          append([]Artifact(nil), record.artifacts...),
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
			RunID:              item.RunID,
			Pipeline:           item.Pipeline,
			Status:             item.Status,
			StartedAt:          startedAt.UTC(),
			FinishedAt:         finishedAt,
			Question:           item.Question,
			CurrentStep:        item.CurrentStep,
			RuntimeMode:        item.RuntimeMode,
			StepProviders:      cloneStringMap(item.StepProviders),
			Warnings:           append([]string(nil), item.Warnings...),
			PendingPermissions: append([]acpruntime.PermissionRequest(nil), item.PendingPermissions...),
			ErrorCode:          item.ErrorCode,
			Error:              item.Error,
			SupersededByRunID:  item.SupersededByRunID,
			RefreshSummary:     cloneRefreshSummary(item.RefreshSummary),
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
