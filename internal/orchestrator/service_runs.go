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
	resolvedStepProviders, err := s.ResolveStepProviderProfile(request.Workspace.Manifest)
	if err != nil {
		return "", err
	}
	runID := s.nextRunID()
	now := s.clock().UTC()

	s.mu.Lock()
	storeQueuedRun := func() {
		s.upsertRunLocked(runRecord{
			info: RunInfo{
				RunID:         runID,
				Pipeline:      string(request.Pipeline),
				Status:        RunStatusQueued,
				StartedAt:     now,
				StepProviders: resolvedStepProviders.Effective.StringMap(),
			},
		})
	}
	if s.isActiveRunLocked() {
		if s.pendingRun != nil {
			if now.Sub(s.pendingRun.queuedAt) <= s.debounceWindow {
				storeQueuedRun()
				s.markRunSupersededLocked(s.pendingRun.runID, runID)
				s.pendingRun = &pendingRun{
					runID:    runID,
					request:  request,
					queuedAt: now,
				}
				s.mu.Unlock()
				return runID, nil
			}
			s.mu.Unlock()
			return "", fmt.Errorf("run is already active and pending queue is outside debounce window")
		}
		storeQueuedRun()
		s.pendingRun = &pendingRun{
			runID:    runID,
			request:  request,
			queuedAt: now,
		}
		s.mu.Unlock()
		return runID, nil
	}
	storeQueuedRun()
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
	case RunStatusSucceeded, RunStatusFailed:
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
			s.upsertRunLocked(runRecord{
				info:      failedInfo,
				artifacts: copiedArtifacts,
			})
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

func (s *Service) storeRun(record runRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.upsertRunLocked(record)
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
			RunID:       record.info.RunID,
			Pipeline:    record.info.Pipeline,
			Status:      record.info.Status,
			StartedAt:   record.info.StartedAt,
			FinishedAt:  record.info.FinishedAt,
			CurrentStep: record.info.CurrentStep,
			Warnings:    append([]string(nil), record.info.Warnings...),
			ErrorCode:   record.info.ErrorCode,
			Error:       record.info.Error,
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
	s.runCancels[runID] = cancel
	if _, requested := s.cancelRequests[runID]; requested {
		shouldCancelImmediately = true
	}
	s.mu.Unlock()

	if shouldCancelImmediately {
		cancel()
	}

	go func() {
		_, _, _ = s.runWithID(runCtx, request, runID)
		s.finishAsyncRun(ctx, runID)
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
	if s.pendingRun != nil {
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

func (s *Service) markRunSupersededLocked(oldRunID string, newRunID string) {
	record, ok := s.runs[oldRunID]
	if !ok {
		return
	}
	finishedAt := s.clock().UTC()
	superseded := record.info
	superseded.Status = RunStatusFailed
	superseded.ErrorCode = ""
	superseded.Error = fmt.Sprintf("run superseded by newer event %q (last-event-wins)", newRunID)
	superseded.FinishedAt = &finishedAt
	s.upsertRunLocked(runRecord{
		info:      superseded,
		artifacts: record.artifacts,
	})
}

func (s *Service) upsertRunLocked(record runRecord) {
	s.runs[record.info.RunID] = &record
	s.trimRunRegistryLocked()
	s.persistHistoryLocked()
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

func (s *Service) persistHistoryLocked() {
	if !s.historyEnabled {
		return
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
		return
	}
	encoded = append(encoded, '\n')
	_ = s.historyWorkspace.WriteFile(runHistoryPath, encoded)
}

func runRecordToHistoryItem(record runRecord) runHistoryItem {
	item := runHistoryItem{
		RunID:         record.info.RunID,
		Pipeline:      record.info.Pipeline,
		Status:        record.info.Status,
		StartedAt:     record.info.StartedAt.UTC().Format(time.RFC3339),
		CurrentStep:   record.info.CurrentStep,
		StepProviders: cloneStringMap(record.info.StepProviders),
		Warnings:      append([]string(nil), record.info.Warnings...),
		ErrorCode:     record.info.ErrorCode,
		Error:         record.info.Error,
		Artifacts:     append([]Artifact(nil), record.artifacts...),
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
			RunID:         item.RunID,
			Pipeline:      item.Pipeline,
			Status:        item.Status,
			StartedAt:     startedAt.UTC(),
			FinishedAt:    finishedAt,
			CurrentStep:   item.CurrentStep,
			StepProviders: cloneStringMap(item.StepProviders),
			Warnings:      append([]string(nil), item.Warnings...),
			ErrorCode:     item.ErrorCode,
			Error:         item.Error,
		},
		artifacts: append([]Artifact(nil), item.Artifacts...),
	}, true
}

func classifyExecutionError(err error) (code string, message string) {
	message = strings.TrimSpace(err.Error())
	if runtimeCode, runtimeMessage, ok := acpruntime.ClassifyError(err); ok {
		if strings.TrimSpace(runtimeMessage) != "" {
			message = runtimeMessage
		}
		return runtimeCode, message
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
