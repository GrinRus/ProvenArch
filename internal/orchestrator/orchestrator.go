package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/reports"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type Pipeline string

const (
	PipelineInit    Pipeline = "init"
	PipelineRefresh Pipeline = "refresh"
)

const (
	runHistoryPath      = "reports/taskruns/run-history.json"
	runHistoryVersion   = 1
	runHistoryRetention = 500
	runLogsPath         = "reports/taskruns/logs"
)

const (
	runErrorCodeCanceled               = "run_canceled"
	runErrorCodeReconciledAfterRestart = "run_reconciled_after_restart"
	runtimeOutputSnippetLimitRunes     = 2000
	runtimeOutputSnippetSuffix         = " ... [truncated]"
)

type RunStatus string

const (
	RunStatusQueued    RunStatus = "queued"
	RunStatusRunning   RunStatus = "running"
	RunStatusSucceeded RunStatus = "succeeded"
	RunStatusFailed    RunStatus = "failed"
)

var (
	ErrRunNotFound      = errors.New("run not found")
	ErrRunNotCancelable = errors.New("run not cancelable")
)

type Service struct {
	runner acpruntime.Runner
	clock  func() time.Time

	mu             sync.RWMutex
	runs           map[string]*runRecord
	runIDSequence  int
	debounceWindow time.Duration
	activeRunID    string
	pendingRun     *pendingRun
	runCancels     map[string]context.CancelFunc
	cancelRequests map[string]struct{}

	historyWorkspace workspace.Root
	historyEnabled   bool
	historyRetention int

	runLogsWorkspace workspace.Root
	runLogsEnabled   bool
	runLogsTTL       time.Duration
	runLogsMaxRuns   int
}

type pendingRun struct {
	runID    string
	request  RunRequest
	queuedAt time.Time
}

type runRecord struct {
	info      RunInfo
	artifacts []Artifact
}

type RunInfo struct {
	RunID       string     `json:"run_id"`
	Pipeline    string     `json:"pipeline"`
	Status      RunStatus  `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	CurrentStep string     `json:"current_step,omitempty"`
	Warnings    []string   `json:"warnings,omitempty"`
	ErrorCode   string     `json:"error_code,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type Artifact struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type RunRequest struct {
	Workspace      workspace.Root
	Pipeline       Pipeline
	NonInteractive bool
}

type runHistorySnapshot struct {
	Version int              `json:"version"`
	Items   []runHistoryItem `json:"items"`
}

type runHistoryItem struct {
	RunID       string     `json:"run_id"`
	Pipeline    string     `json:"pipeline"`
	Status      RunStatus  `json:"status"`
	StartedAt   string     `json:"started_at"`
	FinishedAt  *string    `json:"finished_at,omitempty"`
	CurrentStep string     `json:"current_step,omitempty"`
	Warnings    []string   `json:"warnings,omitempty"`
	ErrorCode   string     `json:"error_code,omitempty"`
	Error       string     `json:"error,omitempty"`
	Artifacts   []Artifact `json:"artifacts,omitempty"`
}

type Option func(*Service)

func WithRunner(runner acpruntime.Runner) Option {
	return func(service *Service) {
		service.runner = runner
	}
}

func WithClock(clock func() time.Time) Option {
	return func(service *Service) {
		service.clock = clock
	}
}

func WithDebounceWindow(window time.Duration) Option {
	return func(service *Service) {
		if window > 0 {
			service.debounceWindow = window
		}
	}
}

func WithHistoryWorkspace(ws workspace.Root) Option {
	return func(service *Service) {
		service.historyWorkspace = ws
		service.historyEnabled = strings.TrimSpace(ws.Path) != ""
		service.runLogsWorkspace = ws
		service.runLogsEnabled = strings.TrimSpace(ws.Path) != ""
	}
}

func WithRunLogsRetention(ttl time.Duration, maxRuns int) Option {
	return func(service *Service) {
		if ttl > 0 {
			service.runLogsTTL = ttl
		}
		if maxRuns > 0 {
			service.runLogsMaxRuns = maxRuns
		}
	}
}

func NewService(options ...Option) *Service {
	service := &Service{
		runner:           claudecode.FakeRunner{},
		clock:            time.Now,
		runs:             map[string]*runRecord{},
		debounceWindow:   5 * time.Minute,
		runCancels:       map[string]context.CancelFunc{},
		cancelRequests:   map[string]struct{}{},
		historyRetention: runHistoryRetention,
		runLogsTTL:       7 * 24 * time.Hour,
		runLogsMaxRuns:   200,
	}
	for _, option := range options {
		option(service)
	}
	service.loadHistory()
	service.reconcileStaleRunsAfterRestart()
	_ = service.cleanupRunLogs()
	return service
}

func ParsePipeline(value string) (Pipeline, error) {
	switch value {
	case string(PipelineInit):
		return PipelineInit, nil
	case string(PipelineRefresh), "update":
		return PipelineRefresh, nil
	default:
		return "", fmt.Errorf("unsupported pipeline %q", value)
	}
}

func (s *Service) Run(ctx context.Context, request RunRequest) (RunInfo, []Artifact, error) {
	runID := s.nextRunID()
	return s.runWithID(ctx, request, runID)
}

func (s *Service) ValidateRuntime(ctx context.Context) error {
	if checker, ok := s.runner.(acpruntime.PreflightRunner); ok {
		return checker.Preflight(ctx)
	}
	return nil
}

func (s *Service) runWithID(ctx context.Context, request RunRequest, runID string) (RunInfo, []Artifact, error) {
	_ = s.cleanupRunLogs()
	now := s.clock().UTC()
	initialInfo := RunInfo{
		RunID:     runID,
		Pipeline:  string(request.Pipeline),
		Status:    RunStatusRunning,
		StartedAt: now,
	}
	s.storeRun(runRecord{info: initialInfo})
	s.appendRunLog(runID, RunLogEntry{
		Timestamp: now,
		Level:     RunLogLevelInfo,
		Message:   "run started",
		Fields: map[string]any{
			"pipeline": string(request.Pipeline),
		},
	})
	if err := s.ValidateRuntime(ctx); err != nil {
		finishedAt := s.clock().UTC()
		failedInfo := initialInfo
		failedInfo.Status = RunStatusFailed
		failedInfo.ErrorCode, failedInfo.Error = s.classifyRunFailure(runID, err)
		failedInfo.FinishedAt = &finishedAt
		s.storeRun(runRecord{info: failedInfo})
		s.appendRunLog(runID, RunLogEntry{
			Timestamp: finishedAt,
			Level:     RunLogLevelError,
			Message:   "run failed: runtime validation",
			Fields: map[string]any{
				"error_code": failedInfo.ErrorCode,
				"error":      failedInfo.Error,
			},
		})
		_ = s.cleanupRunLogs()
		return failedInfo, nil, err
	}
	if err := request.Workspace.EnsureLayout(); err != nil {
		finishedAt := s.clock().UTC()
		failedInfo := initialInfo
		failedInfo.Status = RunStatusFailed
		failedInfo.Error = fmt.Sprintf("ensure workspace layout: %v", err)
		failedInfo.FinishedAt = &finishedAt
		s.storeRun(runRecord{info: failedInfo})
		s.appendRunLog(runID, RunLogEntry{
			Timestamp: finishedAt,
			Level:     RunLogLevelError,
			Message:   "run failed: ensure workspace layout",
			Fields: map[string]any{
				"error": failedInfo.Error,
			},
		})
		_ = s.cleanupRunLogs()
		return failedInfo, nil, err
	}
	validation := request.Workspace.Validate(ctx, workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     true,
		VerifyRefs:   true,
	})
	if !validation.OK {
		finishedAt := s.clock().UTC()
		failedInfo := initialInfo
		failedInfo.Status = RunStatusFailed
		failedInfo.Error = formatValidationReportError(validation)
		failedInfo.Warnings = diagnosticMessages(validation.Warnings)
		failedInfo.FinishedAt = &finishedAt
		s.storeRun(runRecord{
			info: failedInfo,
		})
		s.appendRunLog(runID, RunLogEntry{
			Timestamp: finishedAt,
			Level:     RunLogLevelError,
			Message:   "run failed: workspace validation",
			Fields: map[string]any{
				"error":    failedInfo.Error,
				"warnings": failedInfo.Warnings,
			},
		})
		_ = s.cleanupRunLogs()
		return failedInfo, nil, errors.New(failedInfo.Error)
	}

	execution := pipelineExecution{
		runID:              runID,
		pipeline:           request.Pipeline,
		startedAt:          now,
		workspace:          request.Workspace,
		runner:             s.runner,
		store:              model.NewStore(request.Workspace),
		compiler:           reports.NewCompiler(request.Workspace),
		clock:              s.clock,
		artifacts:          []Artifact{},
		artifactIndex:      map[string]int{},
		findings:           []contracts.Finding{},
		questions:          []contracts.Question{},
		coverage:           nil,
		domainRuns:         map[string]domainRunSummary{},
		stepStatus:         initialInfo,
		runtimeStepMetrics: []runtimeStepQuality{},
		runtimeVersions:    map[string]struct{}{},
	}
	execution.onLog = func(entry RunLogEntry) {
		if strings.TrimSpace(entry.StepID) == "" {
			entry.StepID = execution.stepStatus.CurrentStep
		}
		s.appendRunLog(runID, entry)
	}
	execution.onStep = func(stepID string) {
		progress := initialInfo
		progress.Status = RunStatusRunning
		progress.CurrentStep = stepID
		progress.Warnings = append([]string(nil), execution.warnings...)
		s.storeRun(runRecord{
			info:      progress,
			artifacts: append([]Artifact(nil), execution.artifacts...),
		})
		execution.logInfo(stepID, "", "step started", nil)
	}

	if err := execution.run(ctx); err != nil {
		finishedAt := s.clock().UTC()
		failedInfo := initialInfo
		failedInfo.Status = RunStatusFailed
		failedInfo.ErrorCode, failedInfo.Error = s.classifyRunFailure(runID, err)
		failedInfo.CurrentStep = execution.stepStatus.CurrentStep
		failedInfo.Warnings = append([]string(nil), execution.warnings...)
		failedInfo.FinishedAt = &finishedAt
		if qualityArtifact, qualityErr := execution.writeRunQualitySummary(RunStatusFailed, failedInfo.ErrorCode, failedInfo.Error); qualityErr == nil {
			execution.addArtifacts(qualityArtifact)
		} else {
			failedInfo.Warnings = append(failedInfo.Warnings, fmt.Sprintf("run quality summary write failed: %v", qualityErr))
			execution.logWarn(execution.stepStatus.CurrentStep, "", "failed to write run quality summary", map[string]any{
				"error": qualityErr.Error(),
			})
		}
		s.storeRun(runRecord{
			info:      failedInfo,
			artifacts: execution.artifacts,
		})
		execution.logError(execution.stepStatus.CurrentStep, "", "run failed", map[string]any{
			"error_code": failedInfo.ErrorCode,
			"error":      failedInfo.Error,
		})
		_ = s.cleanupRunLogs()
		return failedInfo, execution.artifacts, err
	}

	finishedAt := s.clock().UTC()
	succeeded := initialInfo
	succeeded.Status = RunStatusSucceeded
	succeeded.CurrentStep = execution.stepStatus.CurrentStep
	succeeded.Warnings = append([]string(nil), execution.warnings...)
	succeeded.FinishedAt = &finishedAt
	if qualityArtifact, qualityErr := execution.writeRunQualitySummary(RunStatusSucceeded, "", ""); qualityErr == nil {
		execution.addArtifacts(qualityArtifact)
	} else {
		succeeded.Warnings = append(succeeded.Warnings, fmt.Sprintf("run quality summary write failed: %v", qualityErr))
		execution.logWarn(execution.stepStatus.CurrentStep, "", "failed to write run quality summary", map[string]any{
			"error": qualityErr.Error(),
		})
	}
	s.storeRun(runRecord{
		info:      succeeded,
		artifacts: execution.artifacts,
	})
	execution.logInfo(execution.stepStatus.CurrentStep, "", "run succeeded", map[string]any{
		"artifacts": len(execution.artifacts),
	})
	_ = s.cleanupRunLogs()

	return succeeded, execution.artifacts, nil
}

func (s *Service) StartAsyncRun(ctx context.Context, request RunRequest) (string, error) {
	_ = s.cleanupRunLogs()
	if err := s.ValidateRuntime(ctx); err != nil {
		return "", err
	}
	runID := s.nextRunID()
	now := s.clock().UTC()

	s.mu.Lock()
	storeQueuedRun := func() {
		s.upsertRunLocked(runRecord{
			info: RunInfo{
				RunID:     runID,
				Pipeline:  string(request.Pipeline),
				Status:    RunStatusQueued,
				StartedAt: now,
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

func (s *Service) loadHistory() {
	if !s.historyEnabled {
		return
	}
	content, err := s.historyWorkspace.ReadFile(runHistoryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		return
	}

	var snapshot runHistorySnapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		return
	}
	if snapshot.Version != runHistoryVersion {
		return
	}

	for _, item := range snapshot.Items {
		record, ok := historyItemToRunRecord(item)
		if !ok {
			continue
		}
		s.runs[record.info.RunID] = &record
	}
}

func (s *Service) reconcileStaleRunsAfterRestart() {
	now := s.clock().UTC()
	type staleRun struct {
		runID         string
		previousState RunStatus
	}
	staleRuns := []staleRun{}

	s.mu.Lock()
	for runID, record := range s.runs {
		if record == nil {
			continue
		}
		if record.info.Status != RunStatusQueued && record.info.Status != RunStatusRunning {
			continue
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
		staleRuns = append(staleRuns, staleRun{
			runID:         runID,
			previousState: previousStatus,
		})
	}
	if len(staleRuns) > 0 {
		s.persistHistoryLocked()
	}
	s.activeRunID = ""
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

	for _, stale := range staleRuns {
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
		RunID:       record.info.RunID,
		Pipeline:    record.info.Pipeline,
		Status:      record.info.Status,
		StartedAt:   record.info.StartedAt.UTC().Format(time.RFC3339),
		CurrentStep: record.info.CurrentStep,
		Warnings:    append([]string(nil), record.info.Warnings...),
		ErrorCode:   record.info.ErrorCode,
		Error:       record.info.Error,
		Artifacts:   append([]Artifact(nil), record.artifacts...),
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
			RunID:       item.RunID,
			Pipeline:    item.Pipeline,
			Status:      item.Status,
			StartedAt:   startedAt.UTC(),
			FinishedAt:  finishedAt,
			CurrentStep: item.CurrentStep,
			Warnings:    append([]string(nil), item.Warnings...),
			ErrorCode:   item.ErrorCode,
			Error:       item.Error,
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

type pipelineExecution struct {
	runID              string
	pipeline           Pipeline
	startedAt          time.Time
	workspace          workspace.Root
	runner             acpruntime.Runner
	store              model.Store
	compiler           reports.Compiler
	clock              func() time.Time
	artifacts          []Artifact
	artifactIndex      map[string]int
	findings           []contracts.Finding
	questions          []contracts.Question
	coverage           *contracts.Coverage
	domainRuns         map[string]domainRunSummary
	stepStatus         RunInfo
	onStep             func(stepID string)
	onLog              func(entry RunLogEntry)
	warnings           []string
	runtimeStepMetrics []runtimeStepQuality
	runtimeVersions    map[string]struct{}
}

type runtimeTaskExecution struct {
	RawJSON    []byte
	Normalized contracts.TaskResult
	Apply      model.ApplyReport
}

type domainRunSummary struct {
	DomainID       string
	RepoScope      string
	TaskEnvelope   string
	OutputPath     string
	RuntimeSummary string
	QuestionIDs    []string
	FindingIDs     []string
	Unresolved     []string
}

func (e *pipelineExecution) run(ctx context.Context) error {
	stepIDs := stepIDsForPipeline(e.pipeline)
	for _, stepID := range stepIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		e.stepStatus.CurrentStep = stepID
		if e.onStep != nil {
			e.onStep(stepID)
		}
		if err := e.runStep(ctx, stepID); err != nil {
			return fmt.Errorf("%s: %w", stepID, err)
		}
	}
	return nil
}

func (e *pipelineExecution) runStep(ctx context.Context, stepID string) error {
	switch stepID {
	case "init.step0.constitution":
		return e.runStepConstitution()
	case "init.step1.collect", "refresh.step1.collect":
		return e.runRuntimeStep(ctx, stepID)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return e.runStepAsIs()
	case "init.step3.findings", "refresh.step3.findings":
		return e.runRuntimeStep(ctx, stepID)
	case "init.step4.proposals", "refresh.step4.proposals":
		return e.runStepProposals()
	default:
		return fmt.Errorf("unsupported step %q", stepID)
	}
}

func (e *pipelineExecution) runStepConstitution() error {
	e.logInfo("init.step0.constitution", "", "materializing constitution artifacts", nil)
	step0Contract, hasStep0Contract, err := loadStep0WizardContract(e.workspace)
	if err != nil {
		e.addWarning(fmt.Sprintf("step0_wizard_contract_invalid: %v; fallback baseline constitution materialization is used", err))
	}
	if !hasStep0Contract {
		e.addWarning("step0_wizard_contract_missing: charter/wizard/step0-contract.json not found; fallback baseline constitution materialization is used")
	}
	if err := e.writeConstitutionArtifacts(hasStep0Contract && err == nil, step0Contract); err != nil {
		return err
	}
	if err := writeBaselineBundle(e.workspace); err != nil {
		return err
	}

	e.addArtifacts(
		Artifact{Path: "charter/overview.md", Kind: "charter", Label: "Constitution"},
		Artifact{Path: "skills/subagents.yaml", Kind: "bundle", Label: "Baseline Subagents"},
	)
	return nil
}

func (e *pipelineExecution) runRuntimeStep(ctx context.Context, stepID string) error {
	if strings.HasSuffix(stepID, "step1.collect") {
		if err := e.runStepCollectByDomain(ctx, stepID); err != nil {
			return err
		}
	} else {
		execution, err := e.executeRuntimeTask(ctx, stepID, "", collectRepoScopes(e.workspace.Manifest.Repos), "")
		if err != nil {
			return err
		}
		taskrunPath := fmt.Sprintf("reports/taskruns/%s-%s.json", e.runID, strings.ReplaceAll(stepID, ".", "-"))
		if err := e.persistTaskRun(taskrunPath, stepID, execution.RawJSON); err != nil {
			return err
		}
	}

	coverageArtifacts, err := e.compiler.WriteCoverage(e.coverage, e.questions)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(coverageArtifacts)...)

	if strings.HasSuffix(stepID, "step3.findings") {
		findingArtifacts, err := e.compiler.WriteFindings(e.findings)
		if err != nil {
			return err
		}
		e.addArtifacts(toOrchestratorArtifacts(findingArtifacts)...)

		architectArtifacts, err := e.compiler.WriteArchitectSummary(e.renderArchitectSummary())
		if err != nil {
			return err
		}
		e.addArtifacts(toOrchestratorArtifacts(architectArtifacts)...)
	}

	return nil
}

func (e *pipelineExecution) runStepCollectByDomain(ctx context.Context, stepID string) error {
	domainIDs, err := loadCanonicalDomainIDs(e.workspace)
	if err != nil {
		return err
	}
	e.logInfo(stepID, "", "domain fan-out prepared", map[string]any{
		"domains": len(domainIDs),
	})
	if len(domainIDs) == 0 {
		e.questions = mergeQuestions(e.questions, []contracts.Question{
			{
				ID:       "q.domains.missing-canonical-cards",
				Text:     "No canonical domain cards found in charter/cards/domains; create them via Step 0 wizard.",
				Priority: "high",
			},
		})
	}

	domainReports := map[string]string{}
	domainEnvelopes := make([]reports.DomainTaskEnvelope, 0, len(domainIDs))
	for _, domainID := range domainIDs {
		e.logInfo(stepID, domainID, "domain collect start", nil)
		repoScope, declaredRepoScope, hasDeclaredRepoScope, err := resolveRepoScopeForDomainCard(e.workspace, domainID, e.workspace.Manifest.Repos)
		if err != nil {
			return err
		}
		unresolved := []string{}
		if hasDeclaredRepoScope && declaredRepoScope != "" && strings.TrimSpace(repoScope) == "" {
			unresolved = append(unresolved, "repo_scope")
			e.questions = mergeQuestions(e.questions, []contracts.Question{
				{
					ID:       fmt.Sprintf("q.domain.%s.unknown-repo-scope", slugutil.Slugify(domainID)),
					Text:     fmt.Sprintf("Canonical domain %q declares unknown repo_scope %q (not present in workspace.yaml)", domainID, declaredRepoScope),
					Priority: "high",
				},
			})
		} else if strings.TrimSpace(repoScope) == "" {
			unresolved = append(unresolved, "repo_scope")
			e.questions = mergeQuestions(e.questions, []contracts.Question{
				{
					ID:       fmt.Sprintf("q.domain.%s.missing-repo-scope", slugutil.Slugify(domainID)),
					Text:     fmt.Sprintf("Canonical domain %q has no matching repo scope in workspace.yaml", domainID),
					Priority: "high",
				},
			})
		}
		envelopePath := fmt.Sprintf("reports/agent-outputs/domains/%s.task-envelope.json", sanitizeDomainArtifactSlug(domainID))
		outputPath := fmt.Sprintf("reports/agent-outputs/domains/%s.md", domainID)
		envelope := reports.DomainTaskEnvelope{
			ContractVersion: 1,
			AgentID:         "domain-analyst",
			DomainID:        domainID,
			RepoScope:       repoScope,
			Unresolved:      unresolved,
			Inputs: reports.DomainTaskInputs{
				DomainCardPath:      fmt.Sprintf("charter/cards/domains/%s.md", domainID),
				CoverageSummaryPath: "reports/coverage/summary.md",
				QuestionsPath:       "reports/coverage/open-questions.md",
				ModelEntitiesGlob:   "model/entities/*.yaml",
				FindingsPath:        "reports/findings/findings.md",
			},
			OutputPath: outputPath,
		}
		domainEnvelopes = append(domainEnvelopes, envelope)

		domainScopes := []string{}
		if strings.TrimSpace(repoScope) != "" {
			domainScopes = append(domainScopes, repoScope)
		}
		execution, err := e.executeRuntimeTask(ctx, stepID, "domain-"+sanitizeDomainArtifactSlug(domainID), domainScopes, domainID)
		if err != nil {
			return err
		}

		taskrunPath := fmt.Sprintf(
			"reports/taskruns/%s-%s-domain-%s.json",
			e.runID,
			strings.ReplaceAll(stepID, ".", "-"),
			sanitizeDomainArtifactSlug(domainID),
		)
		if err := e.persistTaskRun(taskrunPath, stepID+"."+domainID, execution.RawJSON); err != nil {
			return err
		}

		questionIDs := extractQuestionIDs(execution.Normalized.Questions)
		findingIDs := extractFindingIDs(execution.Apply.Findings)
		domainReports[domainID] = renderDomainRuntimeOutput(
			domainID,
			repoScope,
			envelopePath,
			strings.TrimSpace(execution.Normalized.Summary),
			execution.Apply,
			questionIDs,
			findingIDs,
			unresolved,
		)
		e.domainRuns[domainID] = domainRunSummary{
			DomainID:       domainID,
			RepoScope:      repoScope,
			TaskEnvelope:   envelopePath,
			OutputPath:     outputPath,
			RuntimeSummary: strings.TrimSpace(execution.Normalized.Summary),
			QuestionIDs:    questionIDs,
			FindingIDs:     findingIDs,
			Unresolved:     append([]string(nil), unresolved...),
		}
		e.logInfo(stepID, domainID, "domain collect completed", map[string]any{
			"repo_scope":       repoScope,
			"question_count":   len(questionIDs),
			"finding_count":    len(findingIDs),
			"unresolved_count": len(unresolved),
		})
	}

	contractArtifacts, err := e.compiler.WriteDomainTaskEnvelopes(domainEnvelopes)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(contractArtifacts)...)
	agentArtifacts, err := e.compiler.WriteDomainOutputs(domainReports)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(agentArtifacts)...)

	teamCards, err := loadCanonicalTeamCards(e.workspace)
	if err != nil {
		return err
	}
	if len(teamCards) == 0 {
		e.questions = mergeQuestions(e.questions, []contracts.Question{
			{
				ID:       "q.teams.missing-canonical-cards",
				Text:     "No canonical team cards found in charter/cards/teams; create them via Step 0 wizard.",
				Priority: "high",
			},
		})
	}

	if err := e.enrichCanonicalCards(domainIDs, teamCards); err != nil {
		return err
	}
	return nil
}

func (e *pipelineExecution) executeRuntimeTask(
	ctx context.Context,
	stepID string,
	taskSuffix string,
	repoScopes []string,
	domainID string,
) (runtimeTaskExecution, error) {
	taskID := fmt.Sprintf("task-%s-%s", e.runID, strings.ReplaceAll(stepID, ".", "-"))
	taskSuffix = strings.TrimSpace(taskSuffix)
	if taskSuffix != "" {
		taskID += "-" + taskSuffix
	}
	task := acpruntime.Task{
		TaskID:       taskID,
		RunID:        e.runID,
		StepID:       stepID,
		Workspace:    e.workspace.Path,
		RepoScopes:   append([]string(nil), repoScopes...),
		StartedAtUTC: e.clock().UTC(),
	}
	e.logInfo(stepID, domainID, "runtime task started", map[string]any{
		"task_id":     task.TaskID,
		"repo_scopes": task.RepoScopes,
	})

	result, err := e.runner.Run(ctx, task)
	if err != nil {
		e.logError(stepID, domainID, "runtime task failed", runtimeFailureLogFields(task, err, "", ""))
		return runtimeTaskExecution{}, err
	}
	if len(result.RawJSON) == 0 {
		raw, marshalErr := json.MarshalIndent(result.TaskResult, "", "  ")
		if marshalErr != nil {
			return runtimeTaskExecution{}, fmt.Errorf("marshal taskresult for taskrun persistence: %w", marshalErr)
		}
		result.RawJSON = raw
	}

	parsed, err := contracts.ParseTaskResult(result.RawJSON)
	if err != nil {
		e.logError(stepID, domainID, "runtime task parse failed", runtimeFailureLogFields(task, err, result.Stdout, result.Stderr))
		return runtimeTaskExecution{}, err
	}
	normalized := contracts.NormalizeTaskResult(parsed)
	normalized = e.applySemanticGuards(stepID, domainID, task, normalized)
	normalizedRaw, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return runtimeTaskExecution{}, fmt.Errorf("marshal normalized taskresult: %w", err)
	}
	runtimeName := strings.TrimSpace(normalized.Meta.Runtime.Name)
	runtimeVersion := strings.TrimSpace(normalized.Meta.Runtime.Version)
	if runtimeName == "" {
		runtimeName = "unknown"
	}
	runtimeKey := runtimeName
	if runtimeVersion != "" {
		runtimeKey = runtimeName + "@" + runtimeVersion
	}
	e.runtimeVersions[runtimeKey] = struct{}{}

	if len(normalized.Warnings) > 0 {
		for _, runtimeWarning := range normalized.Warnings {
			warningText := strings.TrimSpace(runtimeWarning)
			if warningText == "" {
				continue
			}
			prefixedWarning := warningText
			if strings.TrimSpace(stepID) != "" {
				prefixedWarning = fmt.Sprintf("%s: %s", stepID, warningText)
			}
			e.addWarning(prefixedWarning)
			e.logWarn(stepID, domainID, "runtime warning", map[string]any{
				"warning": warningText,
			})
		}
	}

	applyReport, err := e.store.ApplyChangeset(normalized)
	if err != nil {
		e.logError(stepID, domainID, "model apply failed", map[string]any{
			"task_id": task.TaskID,
			"error":   strings.TrimSpace(err.Error()),
		})
		return runtimeTaskExecution{}, err
	}
	if len(applyReport.DocArtifacts) > 0 {
		docArtifacts, err := e.compiler.WriteDocArtifacts(applyReport.DocArtifacts)
		if err != nil {
			return runtimeTaskExecution{}, err
		}
		e.addArtifacts(toOrchestratorArtifacts(docArtifacts)...)
	}

	e.questions = mergeQuestions(e.questions, normalized.Questions)
	e.coverage = mergeCoverage(e.coverage, normalized.Coverage)
	e.findings = append(e.findings, applyReport.Findings...)
	e.runtimeStepMetrics = append(e.runtimeStepMetrics, runtimeStepQuality{
		StepID:           stepID,
		DomainID:         domainID,
		RuntimeName:      runtimeName,
		RuntimeVersion:   runtimeVersion,
		RepoScopes:       append([]string(nil), task.RepoScopes...),
		ChangesetOps:     len(normalized.Changeset),
		EntityUpserts:    applyReport.UpsertedEntities,
		EdgeUpserts:      applyReport.UpsertedEdges,
		FindingsAdded:    len(applyReport.Findings),
		QuestionsCount:   len(normalized.Questions),
		CoverageObserved: countCoverageObserved(normalized.Coverage),
		CoverageMissing:  countCoverageMissing(normalized.Coverage),
		WarningsCount:    len(normalized.Warnings),
	})
	e.logInfo(stepID, domainID, "runtime task completed", map[string]any{
		"task_id":          task.TaskID,
		"runtime_name":     runtimeName,
		"runtime_version":  runtimeVersion,
		"changeset_ops":    len(normalized.Changeset),
		"entity_upserts":   applyReport.UpsertedEntities,
		"edge_upserts":     applyReport.UpsertedEdges,
		"findings_added":   len(applyReport.Findings),
		"questions_count":  len(normalized.Questions),
		"coverage_missing": countCoverageMissing(normalized.Coverage),
		"warnings_count":   len(normalized.Warnings),
	})

	return runtimeTaskExecution{
		RawJSON:    normalizedRaw,
		Normalized: normalized,
		Apply:      applyReport,
	}, nil
}

func (e *pipelineExecution) persistTaskRun(path string, label string, raw []byte) error {
	if err := e.workspace.WriteFile(path, raw); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: path, Kind: "taskrun", Label: label})
	e.logInfo(e.stepStatus.CurrentStep, "", "taskrun persisted", map[string]any{
		"taskrun_path": path,
		"label":        label,
	})
	return nil
}

func renderDomainRuntimeOutput(
	domainID string,
	repoScope string,
	taskEnvelopePath string,
	runtimeSummary string,
	apply model.ApplyReport,
	questionIDs []string,
	findingIDs []string,
	unresolved []string,
) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("# Domain Analyst Output: %s\n\n", domainID))
	builder.WriteString(fmt.Sprintf("- canonical_domain_id: `%s`\n", domainID))
	builder.WriteString(fmt.Sprintf("- repo_scope: `%s`\n", repoScopeOrUnknown(repoScope)))
	builder.WriteString(fmt.Sprintf("- task_envelope: `%s`\n", taskEnvelopePath))
	if strings.TrimSpace(runtimeSummary) == "" {
		builder.WriteString("- runtime_summary: `none`\n")
	} else {
		builder.WriteString(fmt.Sprintf("- runtime_summary: `%s`\n", runtimeSummary))
	}
	builder.WriteString(fmt.Sprintf("- model_entity_upserts: %d\n", apply.UpsertedEntities))
	builder.WriteString(fmt.Sprintf("- model_edge_upserts: %d\n", apply.UpsertedEdges))
	builder.WriteString(fmt.Sprintf("- related_findings: %s\n", renderBacktickList(findingIDs)))
	builder.WriteString(fmt.Sprintf("- emitted_questions: %s\n", renderBacktickList(questionIDs)))
	builder.WriteString(fmt.Sprintf("- unresolved: %s\n", renderPlainList(unresolved)))
	return builder.String()
}

func (e *pipelineExecution) renderArchitectSummary() string {
	builder := strings.Builder{}
	builder.WriteString("# Architect Aggregation Summary\n\n")
	builder.WriteString(fmt.Sprintf("- total findings: %d\n", len(e.findings)))
	builder.WriteString(fmt.Sprintf("- total questions: %d\n", len(e.questions)))
	builder.WriteString(fmt.Sprintf("- analyzed domains: %d\n", len(e.domainRuns)))

	domainIDs := make([]string, 0, len(e.domainRuns))
	for domainID := range e.domainRuns {
		domainIDs = append(domainIDs, domainID)
	}
	sort.Strings(domainIDs)
	if len(domainIDs) == 0 {
		builder.WriteString("- domain_outputs: none\n")
		return builder.String()
	}

	builder.WriteString("- domain_outputs:\n")
	for _, domainID := range domainIDs {
		domainRun := e.domainRuns[domainID]
		builder.WriteString(fmt.Sprintf("  - `%s` (%s)\n", domainID, repoScopeOrUnknown(domainRun.RepoScope)))
		builder.WriteString(fmt.Sprintf("    - output_path: `%s`\n", domainRun.OutputPath))
		builder.WriteString(fmt.Sprintf("    - task_envelope: `%s`\n", domainRun.TaskEnvelope))
		builder.WriteString(fmt.Sprintf("    - related_findings: %s\n", renderBacktickList(domainRun.FindingIDs)))
		builder.WriteString(fmt.Sprintf("    - emitted_questions: %s\n", renderBacktickList(domainRun.QuestionIDs)))
		builder.WriteString(fmt.Sprintf("    - unresolved: %s\n", renderPlainList(domainRun.Unresolved)))
	}
	return builder.String()
}

func extractQuestionIDs(questions []contracts.Question) []string {
	ids := make([]string, 0, len(questions))
	for _, question := range questions {
		if strings.TrimSpace(question.ID) == "" {
			continue
		}
		ids = append(ids, question.ID)
	}
	sort.Strings(ids)
	return uniqueSorted(ids)
}

func extractFindingIDs(findings []contracts.Finding) []string {
	ids := make([]string, 0, len(findings))
	for _, finding := range findings {
		if strings.TrimSpace(finding.ID) == "" {
			continue
		}
		ids = append(ids, finding.ID)
	}
	sort.Strings(ids)
	return uniqueSorted(ids)
}

func (e *pipelineExecution) runStepAsIs() error {
	e.logInfo(e.stepStatus.CurrentStep, "", "compiling as-is reports", nil)
	entities, err := e.store.ListEntities()
	if err != nil {
		return err
	}
	edges, err := e.store.ListEdges()
	if err != nil {
		return err
	}
	artifacts, err := e.compiler.CompileAsIs(entities, edges)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	e.logInfo(e.stepStatus.CurrentStep, "", "as-is reports compiled", map[string]any{
		"entities":  len(entities),
		"edges":     len(edges),
		"artifacts": len(artifacts),
	})
	return nil
}

func (e *pipelineExecution) runStepProposals() error {
	e.logInfo(e.stepStatus.CurrentStep, "", "compiling proposals", map[string]any{
		"findings": len(e.findings),
	})
	artifacts, err := e.compiler.CompileProposals(e.findings)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(artifacts)...)

	changelog, err := e.compiler.WriteIterationChangelog(
		e.runID,
		string(e.pipeline),
		toReportArtifacts(e.artifacts),
		e.startedAt,
		e.clock().UTC(),
	)
	if err != nil {
		return err
	}
	e.addArtifacts(Artifact{
		Path:  changelog.Path,
		Kind:  changelog.Kind,
		Label: changelog.Label,
	})
	e.logInfo(e.stepStatus.CurrentStep, "", "proposals and changelog compiled", map[string]any{
		"artifacts": len(artifacts) + 1,
	})
	return nil
}

func stepIDsForPipeline(pipeline Pipeline) []string {
	switch pipeline {
	case PipelineInit:
		return []string{
			"init.step0.constitution",
			"init.step1.collect",
			"init.step2.asis_docs",
			"init.step3.findings",
			"init.step4.proposals",
		}
	case PipelineRefresh:
		return []string{
			"refresh.step1.collect",
			"refresh.step2.asis_docs",
			"refresh.step3.findings",
			"refresh.step4.proposals",
		}
	default:
		return nil
	}
}

const step0WizardContractPath = "charter/wizard/step0-contract.json"

type step0WizardContract struct {
	Version       int      `json:"version"`
	ProjectName   string   `json:"project_name"`
	Scope         string   `json:"scope"`
	NFRPriorities []string `json:"nfr_priorities"`
	Rules         []string `json:"rules"`
}

func loadStep0WizardContract(ws workspace.Root) (step0WizardContract, bool, error) {
	content, err := ws.ReadFile(step0WizardContractPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return step0WizardContract{}, false, nil
		}
		return step0WizardContract{}, false, fmt.Errorf("read %s: %w", step0WizardContractPath, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()

	var contract step0WizardContract
	if err := decoder.Decode(&contract); err != nil {
		return step0WizardContract{}, true, fmt.Errorf("parse %s: %w", step0WizardContractPath, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return step0WizardContract{}, true, fmt.Errorf("parse %s: unexpected trailing JSON payload", step0WizardContractPath)
	} else if !errors.Is(err, io.EOF) {
		return step0WizardContract{}, true, fmt.Errorf("parse %s: %w", step0WizardContractPath, err)
	}

	contract.ProjectName = strings.TrimSpace(contract.ProjectName)
	contract.Scope = strings.TrimSpace(contract.Scope)
	contract.NFRPriorities = normalizeOrderedUniqueStrings(contract.NFRPriorities)
	contract.Rules = normalizeOrderedUniqueStrings(contract.Rules)

	validationProblems := []string{}
	if contract.Version != 1 {
		validationProblems = append(validationProblems, "version must be 1")
	}
	if contract.ProjectName == "" {
		validationProblems = append(validationProblems, "project_name is required")
	}
	if contract.Scope == "" {
		validationProblems = append(validationProblems, "scope is required")
	}
	if len(validationProblems) > 0 {
		sort.Strings(validationProblems)
		return step0WizardContract{}, true, fmt.Errorf("invalid %s: %s", step0WizardContractPath, strings.Join(validationProblems, "; "))
	}

	return contract, true, nil
}

func (e *pipelineExecution) writeConstitutionArtifacts(useWizardContract bool, contract step0WizardContract) error {
	projectName := ""
	scope := ""
	nfrPriorities := []string{}
	rules := []string{}
	if useWizardContract {
		projectName = contract.ProjectName
		scope = contract.Scope
		nfrPriorities = append([]string(nil), contract.NFRPriorities...)
		rules = append([]string(nil), contract.Rules...)
	}

	overview := "# Project Constitution\n\nGenerated baseline charter for ACP MVP.\n"
	glossary := "terms: []\n"
	if useWizardContract {
		overview = strings.TrimSpace(fmt.Sprintf(
			"# Project Constitution\n\n- project_name: `%s`\n- scope: `%s`\n\nGenerated from `%s`.\n",
			projectName,
			scope,
			step0WizardContractPath,
		)) + "\n"
		scopeTerms := splitAndNormalizeList(scope)
		glossary = renderStringListYAML("terms", scopeTerms)
	}
	nfrContent := renderStringListYAML("nfr", nfrPriorities)
	rulesContent := renderStringListYAML("rules", rules)

	if err := e.workspace.WriteFile("charter/overview.md", []byte(overview)); err != nil {
		return err
	}
	if err := e.workspace.WriteFile("charter/glossary.yaml", []byte(glossary)); err != nil {
		return err
	}
	if err := e.workspace.WriteFile("charter/nfr.yaml", []byte(nfrContent)); err != nil {
		return err
	}
	if err := e.workspace.WriteFile("charter/rules.yaml", []byte(rulesContent)); err != nil {
		return err
	}

	for _, repo := range e.workspace.Manifest.Repos {
		slug := slugutil.Slugify(repo.Name)
		domainPath := fmt.Sprintf("charter/cards/domains/%s.md", slug)
		domainBody := strings.TrimSpace(fmt.Sprintf("# Domain: %s\n\n- id: `%s`\n- repo_scope: `%s`\n", repo.Name, slug, repo.Name))
		if useWizardContract {
			domainBody += fmt.Sprintf("\n- charter_project: `%s`\n- charter_scope: `%s`\n", projectName, scope)
		}
		domainBody += "\n"
		if err := e.workspace.WriteFile(domainPath, []byte(domainBody)); err != nil {
			return err
		}
	}

	teamBody := "# Team: Platform\n\n- id: `team.platform`\n"
	if useWizardContract {
		teamBody = strings.TrimSpace(teamBody+fmt.Sprintf("- charter_project: `%s`\n", projectName)) + "\n"
	}
	if err := e.workspace.WriteFile("charter/cards/teams/platform.md", []byte(teamBody)); err != nil {
		return err
	}

	return nil
}

func renderStringListYAML(key string, values []string) string {
	values = normalizeOrderedUniqueStrings(values)
	if len(values) == 0 {
		return fmt.Sprintf("%s: []\n", strings.TrimSpace(key))
	}

	builder := strings.Builder{}
	builder.WriteString(strings.TrimSpace(key))
	builder.WriteString(":\n")
	for _, value := range values {
		builder.WriteString("  - ")
		builder.WriteString(strconv.Quote(value))
		builder.WriteString("\n")
	}
	return builder.String()
}

func splitAndNormalizeList(raw string) []string {
	replacer := strings.NewReplacer(",", "\n", ";", "\n", "\t", "\n")
	values := strings.Split(replacer.Replace(raw), "\n")
	return normalizeOrderedUniqueStrings(values)
}

func normalizeOrderedUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func writeBaselineBundle(ws workspace.Root) error {
	return ws.EnsureBaselineBundle()
}

func collectRepoScopes(repos []workspace.RepoSource) []string {
	scopes := make([]string, 0, len(repos))
	for _, repo := range repos {
		scopes = append(scopes, repo.Name)
	}
	sort.Strings(scopes)
	return scopes
}

func toOrchestratorArtifacts(artifacts []reports.Artifact) []Artifact {
	out := make([]Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		out = append(out, Artifact{
			Path:  artifact.Path,
			Kind:  artifact.Kind,
			Label: artifact.Label,
		})
	}
	return out
}

func toReportArtifacts(artifacts []Artifact) []reports.Artifact {
	out := make([]reports.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		out = append(out, reports.Artifact{
			Path:  artifact.Path,
			Kind:  artifact.Kind,
			Label: artifact.Label,
		})
	}
	return out
}

func (e *pipelineExecution) addArtifacts(artifacts ...Artifact) {
	if e.artifactIndex == nil {
		e.artifactIndex = map[string]int{}
	}
	for _, artifact := range artifacts {
		key := artifact.Kind + "|" + artifact.Path
		if existingIndex, exists := e.artifactIndex[key]; exists {
			e.artifacts[existingIndex] = artifact
			continue
		}
		e.artifactIndex[key] = len(e.artifacts)
		e.artifacts = append(e.artifacts, artifact)
	}
}

func (e *pipelineExecution) addWarning(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	for _, existing := range e.warnings {
		if existing == message {
			return
		}
	}
	e.warnings = append(e.warnings, message)
	e.logWarn(e.stepStatus.CurrentStep, "", "run warning", map[string]any{
		"warning": message,
	})
}

func (e *pipelineExecution) logInfo(stepID string, domainID string, message string, fields map[string]any) {
	e.logRunEvent(RunLogLevelInfo, stepID, domainID, message, fields)
}

func (e *pipelineExecution) logWarn(stepID string, domainID string, message string, fields map[string]any) {
	e.logRunEvent(RunLogLevelWarning, stepID, domainID, message, fields)
}

func (e *pipelineExecution) logError(stepID string, domainID string, message string, fields map[string]any) {
	e.logRunEvent(RunLogLevelError, stepID, domainID, message, fields)
}

func (e *pipelineExecution) logRunEvent(level RunLogLevel, stepID string, domainID string, message string, fields map[string]any) {
	if e.onLog == nil {
		return
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	entry := RunLogEntry{
		Timestamp: e.clock().UTC(),
		Level:     level,
		StepID:    strings.TrimSpace(stepID),
		DomainID:  strings.TrimSpace(domainID),
		Message:   message,
	}
	if len(fields) > 0 {
		entry.Fields = fields
		if taskrunPath, ok := fields["taskrun_path"].(string); ok {
			entry.TaskrunPath = strings.TrimSpace(taskrunPath)
		}
	}
	e.onLog(entry)
}

func runtimeFailureLogFields(task acpruntime.Task, err error, fallbackStdout string, fallbackStderr string) map[string]any {
	fields := map[string]any{
		"task_id":     task.TaskID,
		"repo_scopes": append([]string(nil), task.RepoScopes...),
		"error":       strings.TrimSpace(err.Error()),
	}
	if runtimeCode, _, ok := acpruntime.ClassifyError(err); ok {
		fields["error_code"] = runtimeCode
	}

	stdout := fallbackStdout
	stderr := fallbackStderr
	var runnerErr acpruntime.RunnerError
	if errors.As(err, &runnerErr) {
		if strings.TrimSpace(string(runnerErr.Provider)) != "" {
			fields["provider"] = string(runnerErr.Provider)
		}
		if strings.TrimSpace(stdout) == "" {
			stdout = runnerErr.Stdout
		}
		if strings.TrimSpace(stderr) == "" {
			stderr = runnerErr.Stderr
		}
	}

	appendSnippetField(fields, "stdout_snippet", stdout)
	appendSnippetField(fields, "stderr_snippet", stderr)
	return fields
}

func appendSnippetField(fields map[string]any, key string, raw string) {
	if fields == nil {
		return
	}
	snippet := sanitizeAndTruncateSnippet(raw, runtimeOutputSnippetLimitRunes)
	if snippet == "" {
		return
	}
	fields[key] = snippet
}

func sanitizeAndTruncateSnippet(raw string, limitRunes int) string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return ""
	}
	if limitRunes <= 0 {
		limitRunes = runtimeOutputSnippetLimitRunes
	}
	runes := []rune(normalized)
	if len(runes) <= limitRunes {
		return normalized
	}
	truncated := strings.TrimSpace(string(runes[:limitRunes]))
	if truncated == "" {
		truncated = string(runes[:limitRunes])
	}
	return truncated + runtimeOutputSnippetSuffix
}

func countCoverageObserved(coverage *contracts.Coverage) int {
	if coverage == nil {
		return 0
	}
	return len(coverage.Observed)
}

func countCoverageMissing(coverage *contracts.Coverage) int {
	if coverage == nil {
		return 0
	}
	return len(coverage.Missing)
}

func mergeQuestions(existing []contracts.Question, incoming []contracts.Question) []contracts.Question {
	byID := map[string]contracts.Question{}
	byText := map[string]string{}
	order := make([]string, 0, len(existing)+len(incoming))

	appendQuestion := func(question contracts.Question) {
		id := canonicalizeQuestionID(question.ID)
		if id == "" {
			return
		}
		question.ID = id
		textKey := normalizeQuestionText(question.Text)
		if textKey != "" {
			if existingID, exists := byText[textKey]; exists {
				if existingID != id {
					return
				}
			}
		}
		if _, exists := byID[id]; exists {
			return
		}
		byID[id] = question
		order = append(order, id)
		if textKey != "" {
			byText[textKey] = id
		}
	}
	for _, question := range existing {
		appendQuestion(question)
	}
	for _, question := range incoming {
		appendQuestion(question)
	}

	merged := make([]contracts.Question, 0, len(order))
	for _, id := range order {
		merged = append(merged, byID[id])
	}
	return merged
}

func mergeCoverage(existing *contracts.Coverage, incoming *contracts.Coverage) *contracts.Coverage {
	if existing == nil && incoming == nil {
		return nil
	}
	if existing == nil {
		copyCoverage := *incoming
		copyCoverage.Observed = dedupeSemanticStrings(copyCoverage.Observed)
		copyCoverage.Missing = dedupeSemanticStrings(canonicalizeCoverageMissing(copyCoverage.Missing))
		copyCoverage.Notes = dedupeSemanticStrings(copyCoverage.Notes)
		return &copyCoverage
	}
	if incoming == nil {
		copyCoverage := *existing
		copyCoverage.Observed = dedupeSemanticStrings(copyCoverage.Observed)
		copyCoverage.Missing = dedupeSemanticStrings(canonicalizeCoverageMissing(copyCoverage.Missing))
		copyCoverage.Notes = dedupeSemanticStrings(copyCoverage.Notes)
		return &copyCoverage
	}

	merged := &contracts.Coverage{
		Observed: dedupeSemanticStrings(append(existing.Observed, incoming.Observed...)),
		Missing:  dedupeSemanticStrings(canonicalizeCoverageMissing(append(existing.Missing, incoming.Missing...))),
		Notes:    dedupeSemanticStrings(append(existing.Notes, incoming.Notes...)),
	}
	return merged
}

func canonicalizeCoverageMissing(values []string) []string {
	canonical := make([]string, 0, len(values))
	for _, value := range values {
		canonical = append(canonical, canonicalizeCoverageMissingValue(value))
	}
	return canonical
}

func canonicalizeCoverageMissingValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ToLower(trimmed)
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")

	switch normalized {
	case "owner mappings", "owner mapping", "owner team mapping", "owner team mappings":
		return "owner mappings"
	case "ci cd evidence", "ci cd pipelines", "ci cd pipeline evidence", "cicd evidence", "ci pipelines":
		return "ci-cd evidence"
	case "delta validation":
		return "delta validation"
	case "dependency graph":
		return "dependency graph"
	case "runtime metrics":
		return "runtime metrics"
	case "api contracts", "api contract", "api contracts drift", "api specification drift", "api specs":
		return "api contracts"
	case "deployment configs", "deployment config", "deployment configuration", "deployment configurations":
		return "deployment configs"
	case "integration edges", "service integrations":
		return "integration edges"
	case "datastore bindings", "database bindings":
		return "datastore bindings"
	case "dependencies", "dependency drift":
		return "dependencies"
	default:
		return trimmed
	}
}

func normalizeSemanticKey(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func dedupeSemanticStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := normalizeSemanticKey(trimmed)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func canonicalizeQuestionID(id string) string {
	canonical := strings.TrimSpace(id)
	for {
		dot := strings.LastIndex(canonical, ".")
		if dot <= 0 {
			break
		}
		suffix := canonical[dot+1:]
		if _, err := strconv.Atoi(suffix); err != nil {
			break
		}
		canonical = canonical[:dot]
	}
	return strings.TrimSpace(canonical)
}

func normalizeQuestionText(text string) string {
	return normalizeSemanticKey(text)
}

func isOwnerMappingsMissing(coverage *contracts.Coverage) bool {
	if coverage == nil {
		return false
	}
	for _, missing := range coverage.Missing {
		if canonicalizeCoverageMissingValue(missing) == "owner mappings" {
			return true
		}
	}
	return false
}

func shouldFilterRefreshCollectEntityType(entityType string) bool {
	switch normalizeSemanticKey(entityType) {
	case "runtime provider", "runtime", "runtime meta", "metadata":
		return true
	default:
		return false
	}
}

func offTopicCollectTerms() []string {
	return []string{
		"bidding",
		"tender",
		"chinabidding",
		"power system",
		"power enterprise",
		"relay protection",
		"load flow",
		"electric analysis",
		"继电",
		"潮流",
		"电力",
		"招标",
	}
}

func isLikelyPowerScope(value string) bool {
	normalized := normalizeSemanticKey(value)
	if normalized == "" {
		return false
	}
	for _, hint := range []string{"power", "energy", "electric", "grid", "utility"} {
		if strings.Contains(normalized, hint) {
			return true
		}
	}
	return false
}

func shouldApplyOffTopicGuard(task acpruntime.Task) bool {
	if isLikelyPowerScope(task.Workspace) {
		return false
	}
	for _, scope := range task.RepoScopes {
		if isLikelyPowerScope(scope) {
			return false
		}
	}
	return true
}

func detectOffTopicTerms(text string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}
	hits := []string{}
	for _, term := range offTopicCollectTerms() {
		if strings.Contains(normalized, strings.ToLower(term)) {
			hits = append(hits, term)
		}
	}
	return dedupeSemanticStrings(hits)
}

func entitySemanticText(entity *contracts.Entity) string {
	if entity == nil {
		return ""
	}
	parts := []string{
		strings.TrimSpace(entity.ID),
		strings.TrimSpace(entity.Type),
		strings.TrimSpace(entity.Name),
	}
	if len(entity.Aliases) > 0 {
		parts = append(parts, strings.Join(entity.Aliases, " "))
	}
	if entity.Attributes != nil {
		if raw, err := json.Marshal(entity.Attributes); err == nil {
			parts = append(parts, string(raw))
		}
	}
	return strings.Join(parts, " ")
}

func (e *pipelineExecution) applySemanticGuards(stepID string, domainID string, task acpruntime.Task, normalized contracts.TaskResult) contracts.TaskResult {
	if stepID == "refresh.step1.collect" {
		filtered := make([]contracts.Operation, 0, len(normalized.Changeset))
		droppedByType := map[string]int{}
		droppedOffTopicEntities := 0
		droppedOffTopicQuestions := 0
		droppedOffTopicTerms := []string{}
		offTopicGuard := shouldApplyOffTopicGuard(task)
		for _, op := range normalized.Changeset {
			if op.Op == "upsert_entity" && op.Entity != nil && shouldFilterRefreshCollectEntityType(op.Entity.Type) {
				droppedByType[op.Entity.Type]++
				continue
			}
			if offTopicGuard && op.Op == "upsert_entity" && op.Entity != nil {
				hits := detectOffTopicTerms(entitySemanticText(op.Entity))
				if len(hits) > 0 {
					droppedOffTopicEntities++
					droppedOffTopicTerms = append(droppedOffTopicTerms, hits...)
					continue
				}
			}
			filtered = append(filtered, op)
		}
		if len(droppedByType) > 0 {
			normalized.Changeset = filtered
			parts := make([]string, 0, len(droppedByType))
			for typ, count := range droppedByType {
				parts = append(parts, fmt.Sprintf("%s=%d", strings.TrimSpace(typ), count))
			}
			sort.Strings(parts)
			normalized.Warnings = append(
				normalized.Warnings,
				fmt.Sprintf("semantic_guard: dropped refresh.step1.collect entity types [%s]", strings.Join(parts, ", ")),
			)
		} else {
			normalized.Changeset = filtered
		}
		if offTopicGuard {
			questions := make([]contracts.Question, 0, len(normalized.Questions))
			for _, question := range normalized.Questions {
				hits := detectOffTopicTerms(question.Text)
				if len(hits) > 0 {
					droppedOffTopicQuestions++
					droppedOffTopicTerms = append(droppedOffTopicTerms, hits...)
					continue
				}
				questions = append(questions, question)
			}
			normalized.Questions = questions
		}
		if droppedOffTopicEntities > 0 || droppedOffTopicQuestions > 0 {
			normalized.Warnings = append(
				normalized.Warnings,
				fmt.Sprintf(
					"semantic_guard: dropped refresh.step1.collect off-topic artifacts entities=%d questions=%d terms=[%s]",
					droppedOffTopicEntities,
					droppedOffTopicQuestions,
					strings.Join(dedupeSemanticStrings(droppedOffTopicTerms), ", "),
				),
			)
			if len(normalized.Changeset) == 0 {
				normalized.Warnings = append(
					normalized.Warnings,
					"semantic_guard: critical_off_topic_drift in refresh.step1.collect",
				)
			}
		}
	}

	if stepID != "refresh.step3.findings" {
		return normalized
	}
	hasFinding := false
	for _, op := range normalized.Changeset {
		if op.Op == "add_finding" && op.Finding != nil {
			hasFinding = true
			break
		}
	}
	if hasFinding {
		return normalized
	}
	if !isOwnerMappingsMissing(normalized.Coverage) {
		return normalized
	}

	entities, err := e.store.ListEntities()
	if err != nil {
		normalized.Warnings = append(normalized.Warnings, fmt.Sprintf("semantic_guard: owner-gap check failed: %v", err))
		return normalized
	}
	scopeSet := map[string]struct{}{}
	for _, scope := range task.RepoScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		scopeSet[scope] = struct{}{}
	}

	var candidate contracts.Entity
	found := false
	for _, entity := range entities {
		if entity.Type != "service" {
			continue
		}
		if strings.TrimSpace(entity.OwnerTeamID) != "" {
			continue
		}
		if len(scopeSet) > 0 {
			attributes, ok := entity.Attributes.(map[string]any)
			if !ok {
				continue
			}
			repoScope, _ := attributes["repo_scope"].(string)
			repoScope = strings.TrimSpace(repoScope)
			if repoScope == "" {
				continue
			}
			if _, ok := scopeSet[repoScope]; !ok {
				continue
			}
		}
		candidate = entity
		found = true
		break
	}
	if !found {
		repo := "unknown"
		if len(task.RepoScopes) > 0 && strings.TrimSpace(task.RepoScopes[0]) != "" {
			repo = strings.TrimSpace(task.RepoScopes[0])
		}
		relatedID := "scope." + slugutil.Slugify(repo)
		if relatedID == "scope." {
			relatedID = "scope.unknown"
		}
		findingID := "finding.missing-owner." + slugutil.Slugify(relatedID) + ".refresh"
		if strings.TrimSpace(domainID) != "" {
			findingID = findingID + "." + slugutil.Slugify(domainID)
		}
		normalized.Changeset = append(normalized.Changeset, contracts.Operation{
			Op: "add_finding",
			Finding: &contracts.Finding{
				ID:          findingID,
				Severity:    "medium",
				Title:       "Missing owner mapping",
				Description: fmt.Sprintf("owner mappings are unresolved for repo scope %q", repo),
				RuleID:      "rule.owner.required",
				RelatedIDs:  []string{relatedID},
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.62,
					Evidence: []contracts.Evidence{
						{
							Repo: repo,
							Path: "README.md",
						},
					},
				},
			},
		})
		normalized.Warnings = append(
			normalized.Warnings,
			fmt.Sprintf("semantic_guard: added fallback owner-mapping finding %q", findingID),
		)
		return normalized
	}

	evidence := append([]contracts.Evidence(nil), candidate.Provenance.Evidence...)
	if len(evidence) == 0 {
		repo := "unknown"
		if len(task.RepoScopes) > 0 && strings.TrimSpace(task.RepoScopes[0]) != "" {
			repo = strings.TrimSpace(task.RepoScopes[0])
		}
		evidence = []contracts.Evidence{
			{
				Repo: repo,
				Path: "README.md",
			},
		}
	}
	relatedID := strings.TrimSpace(candidate.ID)
	if relatedID == "" {
		relatedID = "svc.unknown"
	}
	findingID := "finding.missing-owner." + slugutil.Slugify(relatedID) + ".refresh"
	if strings.TrimSpace(domainID) != "" {
		findingID = findingID + "." + slugutil.Slugify(domainID)
	}
	normalized.Changeset = append(normalized.Changeset, contracts.Operation{
		Op: "add_finding",
		Finding: &contracts.Finding{
			ID:          findingID,
			Severity:    "medium",
			Title:       "Missing owner mapping",
			Description: fmt.Sprintf("owner_team_id is not confirmed for service %q", relatedID),
			RuleID:      "rule.owner.required",
			RelatedIDs:  []string{relatedID},
			Provenance: contracts.Provenance{
				Kind:       "inference",
				Confidence: 0.66,
				Evidence:   evidence,
			},
		},
	})
	normalized.Warnings = append(
		normalized.Warnings,
		fmt.Sprintf("semantic_guard: added fallback owner-mapping finding %q", findingID),
	)
	return normalized
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func loadCanonicalDomainIDs(ws workspace.Root) ([]string, error) {
	domainsDir, err := ws.Resolve("charter/cards/domains")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(domainsDir); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	domainSet := map[string]struct{}{}
	if err := filepath.WalkDir(domainsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		base := strings.TrimSuffix(entry.Name(), ".md")
		base = strings.TrimSpace(base)
		if base == "" {
			return nil
		}
		domainSet[base] = struct{}{}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan canonical domain cards: %w", err)
	}

	domains := make([]string, 0, len(domainSet))
	for domainID := range domainSet {
		domains = append(domains, domainID)
	}
	sort.Strings(domains)
	return domains, nil
}

func repoScopeForDomain(domainID string, repos []workspace.RepoSource) string {
	domainSlug := slugutil.Slugify(domainID)
	for _, repo := range repos {
		repoSlug := slugutil.Slugify(repo.Name)
		if repoSlug == domainSlug {
			return repo.Name
		}
	}
	return ""
}

func resolveRepoScopeForDomainCard(ws workspace.Root, domainID string, repos []workspace.RepoSource) (repoScope string, declaredRepoScope string, hasDeclaredRepoScope bool, err error) {
	cardPath := fmt.Sprintf("charter/cards/domains/%s.md", domainID)
	contentBytes, err := ws.ReadFile(cardPath)
	if err != nil {
		return "", "", false, err
	}
	content := normalizeLineEndings(string(contentBytes))
	declaredRepoScope = strings.TrimSpace(extractCardField(content, "repo_scope"))
	if declaredRepoScope != "" {
		hasDeclaredRepoScope = true
		if repoScopeExists(declaredRepoScope, repos) {
			return declaredRepoScope, declaredRepoScope, true, nil
		}
		return "", declaredRepoScope, true, nil
	}
	repoScope = strings.TrimSpace(repoScopeForDomain(domainID, repos))
	return repoScope, "", false, nil
}

func repoScopeExists(scope string, repos []workspace.RepoSource) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	for _, repo := range repos {
		if strings.TrimSpace(repo.Name) == scope {
			return true
		}
	}
	return false
}

func repoScopeOrUnknown(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "unknown"
	}
	return scope
}

func sanitizeDomainArtifactSlug(domainID string) string {
	return slugutil.Slugify(domainID)
}

type canonicalTeamCard struct {
	Slug   string
	TeamID string
}

func (e *pipelineExecution) enrichCanonicalCards(domainIDs []string, teamCards []canonicalTeamCard) error {
	entities, err := e.store.ListEntities()
	if err != nil {
		return err
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].ID < entities[j].ID })

	if err := e.enrichDomainCards(domainIDs, entities); err != nil {
		return err
	}
	if err := e.enrichTeamCards(teamCards, entities); err != nil {
		return err
	}

	teamIDSet := map[string]struct{}{}
	for _, teamCard := range teamCards {
		teamIDSet[normalizeID(teamCard.TeamID)] = struct{}{}
	}
	missingTeamQuestions := []contracts.Question{}
	for _, entity := range entities {
		ownerTeamID := normalizeID(entity.OwnerTeamID)
		if ownerTeamID == "" {
			continue
		}
		if _, ok := teamIDSet[ownerTeamID]; ok {
			continue
		}
		slug := slugutil.Slugify(ownerTeamID)
		missingTeamQuestions = append(missingTeamQuestions, contracts.Question{
			ID:       fmt.Sprintf("q.team.%s.missing-canonical-card", slug),
			Text:     fmt.Sprintf("Owner team %q for entity %q has no canonical card in charter/cards/teams", ownerTeamID, entity.ID),
			Priority: "high",
		})
	}
	if len(missingTeamQuestions) > 0 {
		e.questions = mergeQuestions(e.questions, missingTeamQuestions)
	}
	return nil
}

func (e *pipelineExecution) enrichDomainCards(domainIDs []string, entities []contracts.Entity) error {
	for _, domainID := range domainIDs {
		cardPath := fmt.Sprintf("charter/cards/domains/%s.md", domainID)
		contentBytes, err := e.workspace.ReadFile(cardPath)
		if err != nil {
			return err
		}
		content := normalizeLineEndings(string(contentBytes))

		repoScope := strings.TrimSpace(repoScopeForDomain(domainID, e.workspace.Manifest.Repos))
		if repoScope == "" {
			repoScope = strings.TrimSpace(extractCardField(content, "repo_scope"))
		}

		relatedEntities := collectDomainEntities(domainID, repoScope, entities)
		relatedEntitySet := map[string]struct{}{}
		for entityID := range relatedEntities {
			relatedEntitySet[entityID] = struct{}{}
		}

		relatedFindings := collectRelatedFindings(e.findings, relatedEntitySet)
		relatedQuestions := collectRelatedQuestions(e.questions, relatedEntitySet)
		evidenceRefs := collectEvidenceRefs(relatedEntities)
		coverageMissing := []string{}
		if e.coverage != nil {
			coverageMissing = append([]string(nil), e.coverage.Missing...)
		}

		derived := renderDomainDerivedSection(domainID, repoScope, sortedKeys(relatedEntitySet), relatedFindings, relatedQuestions, coverageMissing, evidenceRefs)
		updated := mergeDerivedSection(content, derived)
		if err := e.workspace.WriteFile(cardPath, []byte(updated)); err != nil {
			return err
		}
	}
	return nil
}

func (e *pipelineExecution) enrichTeamCards(teamCards []canonicalTeamCard, entities []contracts.Entity) error {
	for _, teamCard := range teamCards {
		cardPath := fmt.Sprintf("charter/cards/teams/%s.md", teamCard.Slug)
		contentBytes, err := e.workspace.ReadFile(cardPath)
		if err != nil {
			return err
		}
		content := normalizeLineEndings(string(contentBytes))

		teamID := normalizeID(teamCard.TeamID)
		if teamID == "" {
			teamID = "team." + slugutil.Slugify(teamCard.Slug)
		}

		relatedServices := collectTeamServices(teamID, entities)
		relatedServiceSet := map[string]struct{}{}
		for serviceID := range relatedServices {
			relatedServiceSet[serviceID] = struct{}{}
		}
		relatedFindings := collectRelatedFindings(e.findings, relatedServiceSet)
		relatedQuestions := collectRelatedQuestions(e.questions, relatedServiceSet)
		evidenceRefs := collectEvidenceRefs(relatedServices)
		coverageMissing := []string{}
		if e.coverage != nil {
			coverageMissing = append([]string(nil), e.coverage.Missing...)
		}

		derived := renderTeamDerivedSection(teamID, sortedKeys(relatedServiceSet), relatedFindings, relatedQuestions, coverageMissing, evidenceRefs)
		updated := mergeDerivedSection(content, derived)
		if err := e.workspace.WriteFile(cardPath, []byte(updated)); err != nil {
			return err
		}
	}
	return nil
}

func loadCanonicalTeamCards(ws workspace.Root) ([]canonicalTeamCard, error) {
	teamsDir, err := ws.Resolve("charter/cards/teams")
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(teamsDir); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	cards := []canonicalTeamCard{}
	if err := filepath.WalkDir(teamsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		slug := strings.TrimSpace(strings.TrimSuffix(entry.Name(), ".md"))
		if slug == "" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		teamID := strings.TrimSpace(extractCardField(normalizeLineEndings(string(content)), "id"))
		if teamID == "" {
			teamID = "team." + slugutil.Slugify(slug)
		}
		cards = append(cards, canonicalTeamCard{
			Slug:   slug,
			TeamID: teamID,
		})
		return nil
	}); err != nil {
		return nil, fmt.Errorf("scan canonical team cards: %w", err)
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Slug < cards[j].Slug
	})
	return cards, nil
}

func collectDomainEntities(domainID string, repoScope string, entities []contracts.Entity) map[string]contracts.Entity {
	domainSlug := slugutil.Slugify(domainID)
	related := map[string]contracts.Entity{}
	for _, entity := range entities {
		if strings.TrimSpace(repoScope) != "" && hasEvidenceRepo(entity, repoScope) {
			related[entity.ID] = entity
			continue
		}
		entitySlug := slugutil.Slugify(entity.ID)
		if strings.Contains(entitySlug, domainSlug) {
			related[entity.ID] = entity
		}
	}
	return related
}

func collectTeamServices(teamID string, entities []contracts.Entity) map[string]contracts.Entity {
	services := map[string]contracts.Entity{}
	for _, entity := range entities {
		if entity.Type != "service" {
			continue
		}
		if normalizeID(entity.OwnerTeamID) == teamID {
			services[entity.ID] = entity
		}
	}
	return services
}

func collectRelatedFindings(findings []contracts.Finding, relatedIDs map[string]struct{}) []string {
	ids := []string{}
	for _, finding := range findings {
		if !hasRelatedIntersection(finding.RelatedIDs, relatedIDs) {
			continue
		}
		if strings.TrimSpace(finding.ID) == "" {
			continue
		}
		ids = append(ids, finding.ID)
	}
	sort.Strings(ids)
	return uniqueSorted(ids)
}

func collectRelatedQuestions(questions []contracts.Question, relatedIDs map[string]struct{}) []string {
	ids := []string{}
	for _, question := range questions {
		if !hasRelatedIntersection(question.RelatedIDs, relatedIDs) {
			continue
		}
		if strings.TrimSpace(question.ID) == "" {
			continue
		}
		ids = append(ids, question.ID)
	}
	sort.Strings(ids)
	return uniqueSorted(ids)
}

func collectEvidenceRefs(entities map[string]contracts.Entity) []string {
	refs := []string{}
	for _, id := range sortedKeys(entities) {
		entity := entities[id]
		for _, evidence := range entity.Provenance.Evidence {
			repo := strings.TrimSpace(evidence.Repo)
			path := strings.TrimSpace(evidence.Path)
			if repo == "" || path == "" {
				continue
			}
			refs = append(refs, fmt.Sprintf("%s:%s", repo, path))
		}
	}
	sort.Strings(refs)
	return uniqueSorted(refs)
}

func renderDomainDerivedSection(domainID string, repoScope string, entityIDs []string, findingIDs []string, questionIDs []string, coverageMissing []string, evidenceRefs []string) string {
	coverageMissing = uniqueSorted(append([]string(nil), coverageMissing...))

	builder := strings.Builder{}
	builder.WriteString("## Derived (ACP Step1)\n\n")
	builder.WriteString(fmt.Sprintf("- domain_id: `%s`\n", domainID))
	builder.WriteString(fmt.Sprintf("- repo_scope: `%s`\n", repoScopeOrUnknown(repoScope)))
	builder.WriteString(fmt.Sprintf("- related_entities: %s\n", renderBacktickList(entityIDs)))
	builder.WriteString(fmt.Sprintf("- related_findings: %s\n", renderBacktickList(findingIDs)))
	builder.WriteString(fmt.Sprintf("- open_questions: %s\n", renderBacktickList(questionIDs)))
	builder.WriteString(fmt.Sprintf("- coverage_missing: %s\n", renderPlainList(coverageMissing)))
	if len(evidenceRefs) == 0 {
		builder.WriteString("- evidence_refs: none\n")
	} else {
		builder.WriteString("- evidence_refs:\n")
		for _, ref := range evidenceRefs {
			builder.WriteString(fmt.Sprintf("  - `%s`\n", ref))
		}
	}
	return strings.TrimRight(builder.String(), "\n")
}

func renderTeamDerivedSection(teamID string, serviceIDs []string, findingIDs []string, questionIDs []string, coverageMissing []string, evidenceRefs []string) string {
	coverageMissing = uniqueSorted(append([]string(nil), coverageMissing...))

	builder := strings.Builder{}
	builder.WriteString("## Derived (ACP Step1)\n\n")
	builder.WriteString(fmt.Sprintf("- team_id: `%s`\n", teamID))
	builder.WriteString(fmt.Sprintf("- related_services: %s\n", renderBacktickList(serviceIDs)))
	builder.WriteString(fmt.Sprintf("- related_findings: %s\n", renderBacktickList(findingIDs)))
	builder.WriteString(fmt.Sprintf("- open_questions: %s\n", renderBacktickList(questionIDs)))
	builder.WriteString(fmt.Sprintf("- coverage_missing: %s\n", renderPlainList(coverageMissing)))
	if len(evidenceRefs) == 0 {
		builder.WriteString("- evidence_refs: none\n")
	} else {
		builder.WriteString("- evidence_refs:\n")
		for _, ref := range evidenceRefs {
			builder.WriteString(fmt.Sprintf("  - `%s`\n", ref))
		}
	}
	return strings.TrimRight(builder.String(), "\n")
}

func mergeDerivedSection(content string, derivedSection string) string {
	const heading = "## Derived (ACP Step1)"
	content = normalizeLineEndings(content)
	trimmed := strings.TrimRight(content, "\n")
	if idx := strings.Index(trimmed, heading); idx >= 0 {
		trimmed = strings.TrimRight(trimmed[:idx], "\n")
	}
	if strings.TrimSpace(trimmed) == "" {
		return derivedSection + "\n"
	}
	return trimmed + "\n\n" + derivedSection + "\n"
}

func hasEvidenceRepo(entity contracts.Entity, repoScope string) bool {
	for _, evidence := range entity.Provenance.Evidence {
		if strings.TrimSpace(evidence.Repo) == strings.TrimSpace(repoScope) {
			return true
		}
	}
	return false
}

func hasRelatedIntersection(related []string, universe map[string]struct{}) bool {
	if len(universe) == 0 {
		return false
	}
	for _, id := range related {
		if _, ok := universe[id]; ok {
			return true
		}
	}
	return false
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderBacktickList(values []string) string {
	values = uniqueSorted(append([]string(nil), values...))
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("`%s`", value))
	}
	return strings.Join(parts, ", ")
}

func renderPlainList(values []string) string {
	values = uniqueSorted(append([]string(nil), values...))
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func normalizeLineEndings(input string) string {
	return strings.ReplaceAll(input, "\r\n", "\n")
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func extractCardField(content string, field string) string {
	prefix := "- " + strings.TrimSpace(field) + ":"
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		value = strings.Trim(value, "`")
		value = strings.Trim(value, `"'`)
		return strings.TrimSpace(value)
	}
	return ""
}

func formatValidationReportError(report workspace.ValidationReport) string {
	if len(report.Errors) == 0 {
		return "workspace validation failed"
	}
	messages := make([]string, 0, len(report.Errors))
	for _, diagnostic := range report.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", diagnostic.Code, diagnostic.Message))
	}
	return strings.Join(messages, "; ")
}

func diagnosticMessages(diagnostics []workspace.Diagnostic) []string {
	if len(diagnostics) == 0 {
		return nil
	}
	out := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, fmt.Sprintf("%s: %s", diagnostic.Code, diagnostic.Message))
	}
	return out
}
