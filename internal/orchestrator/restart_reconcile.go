package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type staleRunReconciliation struct {
	runID         string
	previousState RunStatus
}

type staleRunResumeTarget struct {
	runID   string
	request RunRequest
}

func (s *Service) loadHistory() {
	if !s.historyEnabled {
		return
	}
	content, err := s.historyWorkspace.ReadFile(runHistoryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			content, err = s.historyWorkspace.ReadLastGoodFile(runHistoryPath)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					s.addHistoryRecoveryDiagnostic(fmt.Sprintf("load run history last-good failed: %v", err))
				}
				return
			}
			s.addHistoryRecoveryDiagnostic(fmt.Sprintf("recovered run history from %s", runHistoryPath+".last-good"))
		} else {
			lastGood, lastGoodErr := s.historyWorkspace.ReadLastGoodFile(runHistoryPath)
			if lastGoodErr != nil {
				s.addHistoryRecoveryDiagnostic(fmt.Sprintf("load run history failed: %v", err))
				return
			}
			content = lastGood
			s.addHistoryRecoveryDiagnostic(fmt.Sprintf("recovered run history from %s after read failure: %v", runHistoryPath+".last-good", err))
		}
	}

	snapshot, err := parseRunHistorySnapshot(content)
	if err != nil {
		lastGood, lastGoodErr := s.historyWorkspace.ReadLastGoodFile(runHistoryPath)
		if lastGoodErr != nil {
			s.addHistoryRecoveryDiagnostic(fmt.Sprintf("parse run history failed: %v", err))
			return
		}
		recoveredSnapshot, recoveredErr := parseRunHistorySnapshot(lastGood)
		if recoveredErr != nil {
			s.addHistoryRecoveryDiagnostic(fmt.Sprintf("parse run history and last-good failed: current=%v last_good=%v", err, recoveredErr))
			return
		}
		snapshot = recoveredSnapshot
		s.addHistoryRecoveryDiagnostic(fmt.Sprintf("recovered run history from %s after invalid current: %v", runHistoryPath+".last-good", err))
	}

	for _, item := range snapshot.Items {
		record, ok := historyItemToRunRecord(item)
		if !ok {
			continue
		}
		s.runs[record.info.RunID] = &record
	}
}

func parseRunHistorySnapshot(content []byte) (runHistorySnapshot, error) {
	var snapshot runHistorySnapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		return runHistorySnapshot{}, fmt.Errorf("decode run history: %w", err)
	}
	if snapshot.Version != runHistoryVersion {
		return runHistorySnapshot{}, fmt.Errorf("unsupported run history version %d", snapshot.Version)
	}
	return snapshot, nil
}

func (s *Service) addHistoryRecoveryDiagnostic(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	s.historyRecoveryDiagnostics = append(s.historyRecoveryDiagnostics, message)
}

func (s *Service) recoverStaleRunsAfterRestart() {
	now := s.clock().UTC()
	reconciledRuns := []staleRunReconciliation{}
	var resumeTarget *staleRunResumeTarget

	s.mu.Lock()
	candidateRunID := ""
	if s.resumeStaleAsync {
		candidateRunID = s.findResumableRunningRunLocked()
	}
	for runID, record := range s.runs {
		if record == nil {
			continue
		}
		switch record.info.Status {
		case RunStatusQueued:
		case RunStatusRunning:
		default:
			continue
		}
		if record.info.Status == RunStatusRunning && runID == candidateRunID {
			if pipeline, err := ParsePipeline(record.info.Pipeline); err == nil {
				resumeTarget = &staleRunResumeTarget{
					runID: runID,
					request: RunRequest{
						Workspace:      s.historyWorkspace,
						Pipeline:       pipeline,
						NonInteractive: true,
					},
				}
				continue
			}
		}
		previousStatus := record.info.Status
		finishedAt := now
		reconciledInfo := record.info
		reconciledInfo.Status = RunStatusFailed
		reconciledInfo.ErrorCode = runErrorCodeReconciledAfterRestart
		reconciledInfo.Error = fmt.Sprintf("run reconciled after service restart (stale status=%s)", previousStatus)
		reconciledInfo.FinishedAt = &finishedAt
		record.info = reconciledInfo
		s.runs[runID] = record
		reconciledRuns = append(reconciledRuns, staleRunReconciliation{
			runID:         runID,
			previousState: previousStatus,
		})
	}
	if len(reconciledRuns) > 0 {
		_ = s.persistHistoryLocked()
	}
	if resumeTarget != nil {
		s.activeRunID = resumeTarget.runID
	} else {
		s.activeRunID = ""
	}
	s.pendingRun = nil
	if s.runCancels == nil {
		s.runCancels = map[string]context.CancelFunc{}
	}
	for runID, cancel := range s.runCancels {
		if cancel != nil {
			cancel()
		}
		delete(s.runCancels, runID)
	}
	if s.cancelRequests == nil {
		s.cancelRequests = map[string]struct{}{}
	}
	for runID := range s.cancelRequests {
		delete(s.cancelRequests, runID)
	}
	s.mu.Unlock()

	for _, stale := range reconciledRuns {
		s.appendRunLog(stale.runID, RunLogEntry{
			Timestamp: now,
			Level:     RunLogLevelWarning,
			Message:   "run reconciled after restart",
			Fields: map[string]any{
				"error_code":      runErrorCodeReconciledAfterRestart,
				"previous_status": string(stale.previousState),
			},
		})
	}
	if resumeTarget != nil {
		s.launchAsyncRun(context.Background(), resumeTarget.runID, resumeTarget.request)
	}
}

func (s *Service) findResumableRunningRunLocked() string {
	if !s.resumeStaleAsync || !s.historyEnabled || strings.TrimSpace(s.historyWorkspace.Path) == "" {
		return ""
	}

	candidates := make([]runRecord, 0, len(s.runs))
	for _, record := range s.runs {
		if record == nil || record.info.Status != RunStatusRunning {
			continue
		}
		if !s.isRunningRunResumable(*record) {
			continue
		}
		candidates = append(candidates, *record)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].info.StartedAt.Equal(candidates[j].info.StartedAt) {
			return candidates[i].info.RunID > candidates[j].info.RunID
		}
		return candidates[i].info.StartedAt.After(candidates[j].info.StartedAt)
	})
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].info.RunID
}

func (s *Service) isRunningRunResumable(record runRecord) bool {
	pipeline, err := ParsePipeline(record.info.Pipeline)
	if err != nil {
		return false
	}
	currentStep := strings.TrimSpace(record.info.CurrentStep)
	if currentStep == "" {
		return false
	}
	resumeStep := resumeStepForCurrentStep(pipeline, currentStep)
	if resumeStep == "" {
		return false
	}
	if !isRuntimeOrLaterStep(pipeline, currentStep) {
		return true
	}
	return hasShardArtifactsForRun(s.historyWorkspace.Path, record.info.RunID, resumeStep)
}
