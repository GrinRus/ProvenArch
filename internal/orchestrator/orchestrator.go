package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
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
	runErrorCodePartialFailed          = "run_partial_failed"
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
	runnerFactory      stepRunnerFactory
	clock              func() time.Time
	executionOverrides acpruntime.ExecutionOverrides
	resumeStaleAsync   bool
	providerFallback   acpruntime.Provider
	providerSource     acpruntime.ProviderSource

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
	RunID         string            `json:"run_id"`
	Pipeline      string            `json:"pipeline"`
	Status        RunStatus         `json:"status"`
	StartedAt     time.Time         `json:"started_at"`
	FinishedAt    *time.Time        `json:"finished_at,omitempty"`
	CurrentStep   string            `json:"current_step,omitempty"`
	StepProviders map[string]string `json:"step_providers,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
	ErrorCode     string            `json:"error_code,omitempty"`
	Error         string            `json:"error,omitempty"`
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
	RunID         string            `json:"run_id"`
	Pipeline      string            `json:"pipeline"`
	Status        RunStatus         `json:"status"`
	StartedAt     string            `json:"started_at"`
	FinishedAt    *string           `json:"finished_at,omitempty"`
	CurrentStep   string            `json:"current_step,omitempty"`
	StepProviders map[string]string `json:"step_providers,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
	ErrorCode     string            `json:"error_code,omitempty"`
	Error         string            `json:"error,omitempty"`
	Artifacts     []Artifact        `json:"artifacts,omitempty"`
}

type stepRunnerFactory interface {
	Build(provider acpruntime.Provider) (acpruntime.Runner, error)
}

type stepRunnerFactoryFunc func(provider acpruntime.Provider) (acpruntime.Runner, error)

func (fn stepRunnerFactoryFunc) Build(provider acpruntime.Provider) (acpruntime.Runner, error) {
	return fn(provider)
}

type Option func(*Service)

func WithRunner(runner acpruntime.Runner) Option {
	return func(service *Service) {
		service.runnerFactory = stepRunnerFactoryFunc(func(acpruntime.Provider) (acpruntime.Runner, error) {
			return runner, nil
		})
	}
}

func WithRunnerFactory(factory func(provider acpruntime.Provider) (acpruntime.Runner, error)) Option {
	return func(service *Service) {
		if factory == nil {
			return
		}
		service.runnerFactory = stepRunnerFactoryFunc(factory)
	}
}

func WithClock(clock func() time.Time) Option {
	return func(service *Service) {
		service.clock = clock
	}
}

func WithExecutionOverrides(overrides acpruntime.ExecutionOverrides) Option {
	return func(service *Service) {
		service.executionOverrides = overrides
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

func WithResumeStaleAsyncRuns() Option {
	return func(service *Service) {
		service.resumeStaleAsync = true
	}
}

func WithProviderFallback(provider acpruntime.Provider, source acpruntime.ProviderSource) Option {
	return func(service *Service) {
		if provider != "" {
			service.providerFallback = provider
		}
		if source != "" {
			service.providerSource = source
		}
	}
}

func NewService(options ...Option) *Service {
	service := &Service{
		runnerFactory:    stepRunnerFactoryFunc(func(acpruntime.Provider) (acpruntime.Runner, error) { return claudecode.FakeRunner{}, nil }),
		clock:            time.Now,
		runs:             map[string]*runRecord{},
		debounceWindow:   5 * time.Minute,
		runCancels:       map[string]context.CancelFunc{},
		cancelRequests:   map[string]struct{}{},
		historyRetention: runHistoryRetention,
		runLogsTTL:       7 * 24 * time.Hour,
		runLogsMaxRuns:   200,
		providerFallback: acpruntime.ProviderClaudeCode,
		providerSource:   acpruntime.ProviderSourceDefault,
	}
	for _, option := range options {
		option(service)
	}
	service.loadHistory()
	_ = service.cleanupRunLogs()
	return service
}

func (s *Service) ReconcileStaleRunsAfterRestart() {
	s.recoverStaleRunsAfterRestart()
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

func (s *Service) ResolveExecutionProfile(manifest workspace.Manifest) acpruntime.ExecutionResolution {
	return acpruntime.ResolveExecution(manifest, s.executionOverrides)
}

func (s *Service) ResolveStepProviderProfile(manifest workspace.Manifest) (acpruntime.StepProviderResolution, error) {
	return acpruntime.ResolveStepProviders(manifest, s.providerFallback, s.providerSource)
}

func (s *Service) Run(ctx context.Context, request RunRequest) (RunInfo, []Artifact, error) {
	runID := s.nextRunID()
	return s.runWithID(ctx, request, runID)
}

func (s *Service) ValidateRuntime(ctx context.Context, manifests ...workspace.Manifest) error {
	manifest := workspace.Manifest{}
	if len(manifests) > 0 {
		manifest = manifests[0]
	}
	resolved, err := s.ResolveStepProviderProfile(manifest)
	if err != nil {
		return err
	}
	resolver := newStepRunnerResolver(s.runnerFactory, resolved.Effective)
	return resolver.Preflight(ctx)
}

func (s *Service) runWithID(ctx context.Context, request RunRequest, runID string) (RunInfo, []Artifact, error) {
	_ = s.cleanupRunLogs()
	now := s.clock().UTC()
	resumedRecord, resumed := s.loadExistingRunRecord(runID)
	startedAt := now
	initialArtifacts := []Artifact{}
	initialWarnings := []string{}
	resumeFromStep := ""
	resolvedStepProviders, resolvedStepProvidersErr := s.ResolveStepProviderProfile(request.Workspace.Manifest)
	runLogMessage := "run started"
	runLogFields := map[string]any{
		"pipeline": string(request.Pipeline),
	}
	if resumed {
		startedAt = resumedRecord.info.StartedAt.UTC()
		initialArtifacts = append([]Artifact(nil), resumedRecord.artifacts...)
		initialWarnings = append([]string(nil), resumedRecord.info.Warnings...)
		resumeFromStep = resumeStepForCurrentStep(request.Pipeline, resumedRecord.info.CurrentStep)
		runLogMessage = "run resumed after restart"
		runLogFields["previous_current_step"] = resumedRecord.info.CurrentStep
		runLogFields["resume_from_step"] = resumeFromStep
	}
	initialInfo := RunInfo{
		RunID:         runID,
		Pipeline:      string(request.Pipeline),
		Status:        RunStatusRunning,
		StartedAt:     startedAt,
		CurrentStep:   resumeFromStep,
		StepProviders: map[string]string{},
		Warnings:      append([]string(nil), initialWarnings...),
	}
	if resolvedStepProvidersErr == nil {
		initialInfo.StepProviders = resolvedStepProviders.Effective.StringMap()
	}
	s.storeRun(runRecord{
		info:      initialInfo,
		artifacts: append([]Artifact(nil), initialArtifacts...),
	})
	s.appendRunLog(runID, RunLogEntry{
		Timestamp: now,
		Level:     RunLogLevelInfo,
		Message:   runLogMessage,
		Fields:    runLogFields,
	})
	if resolvedStepProvidersErr != nil {
		finishedAt := s.clock().UTC()
		failedInfo := initialInfo
		failedInfo.Status = RunStatusFailed
		failedInfo.ErrorCode, failedInfo.Error = s.classifyRunFailure(runID, resolvedStepProvidersErr)
		failedInfo.FinishedAt = &finishedAt
		s.storeRun(runRecord{
			info:      failedInfo,
			artifacts: append([]Artifact(nil), initialArtifacts...),
		})
		s.appendRunLog(runID, RunLogEntry{
			Timestamp: finishedAt,
			Level:     RunLogLevelError,
			Message:   "run failed: runtime provider resolution",
			Fields: map[string]any{
				"error_code": failedInfo.ErrorCode,
				"error":      failedInfo.Error,
			},
		})
		_ = s.cleanupRunLogs()
		return failedInfo, nil, resolvedStepProvidersErr
	}
	if err := request.Workspace.EnsureLayout(); err != nil {
		finishedAt := s.clock().UTC()
		failedInfo := initialInfo
		failedInfo.Status = RunStatusFailed
		failedInfo.ErrorCode, failedInfo.Error = s.classifyRunFailure(runID, fmt.Errorf("ensure workspace layout: %w", err))
		failedInfo.FinishedAt = &finishedAt
		s.storeRun(runRecord{
			info:      failedInfo,
			artifacts: append([]Artifact(nil), initialArtifacts...),
		})
		s.appendRunLog(runID, RunLogEntry{
			Timestamp: finishedAt,
			Level:     RunLogLevelError,
			Message:   "run failed: ensure workspace layout",
			Fields: map[string]any{
				"error_code": failedInfo.ErrorCode,
				"error":      failedInfo.Error,
			},
		})
		_ = s.cleanupRunLogs()
		return failedInfo, nil, err
	}
	resolvedExecution := s.ResolveExecutionProfile(request.Workspace.Manifest)
	stepRunnerResolver := newStepRunnerResolver(s.runnerFactory, resolvedStepProviders.Effective)
	validation := request.Workspace.Validate(ctx, workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     true,
		VerifyRefs:   true,
	})
	if !validation.OK {
		finishedAt := s.clock().UTC()
		failedInfo := initialInfo
		failedInfo.Status = RunStatusFailed
		validationErr := errors.New(formatValidationReportError(validation))
		failedInfo.ErrorCode, failedInfo.Error = s.classifyRunFailure(runID, validationErr)
		failedInfo.Warnings = diagnosticMessages(validation.Warnings)
		failedInfo.FinishedAt = &finishedAt
		s.storeRun(runRecord{
			info:      failedInfo,
			artifacts: append([]Artifact(nil), initialArtifacts...),
		})
		s.appendRunLog(runID, RunLogEntry{
			Timestamp: finishedAt,
			Level:     RunLogLevelError,
			Message:   "run failed: workspace validation",
			Fields: map[string]any{
				"error_code": failedInfo.ErrorCode,
				"error":      failedInfo.Error,
				"warnings":   failedInfo.Warnings,
			},
		})
		_ = s.cleanupRunLogs()
		return failedInfo, nil, validationErr
	}

	execution := pipelineExecution{
		runID:              runID,
		pipeline:           request.Pipeline,
		startedAt:          startedAt,
		workspace:          request.Workspace,
		runnerResolver:     stepRunnerResolver,
		store:              model.NewStore(request.Workspace),
		compiler:           reports.NewCompiler(request.Workspace),
		clock:              s.clock,
		artifacts:          append([]Artifact(nil), initialArtifacts...),
		artifactIndex:      artifactIndexFor(initialArtifacts),
		findings:           []contracts.Finding{},
		questions:          []contracts.Question{},
		coverage:           nil,
		domainRuns:         map[string]domainRunSummary{},
		stepStatus:         initialInfo,
		runtimeStepMetrics: []runtimeStepQuality{},
		runtimeVersions:    map[string]struct{}{},
		resumeFromStep:     resumeFromStep,
		resumeSourceStep:   strings.TrimSpace(resumedRecord.info.CurrentStep),
		warnings:           append([]string(nil), initialWarnings...),
		resolvedRepoPaths:  map[string]string{},
		repoSelectionMode:  "all",
		selectedRepoScopes: collectRepoScopes(request.Workspace.Manifest.Repos),
		reportContext:      reports.DefaultReportRenderContext(),
		stepProviders:      resolvedStepProviders.Effective,
	}
	for _, resolvedRepo := range validation.ResolvedRepos {
		name := strings.TrimSpace(resolvedRepo.Name)
		path := strings.TrimSpace(resolvedRepo.Path)
		if name == "" || path == "" {
			continue
		}
		execution.resolvedRepoPaths[name] = path
	}
	resolvedTimeouts := acpruntime.ResolveTimeouts(request.Workspace.Manifest)
	execution.executionProfile = resolvedExecution.Effective
	if resolvedTimeouts.Effective.StepTimeoutSec > 0 {
		execution.runtimeStepTimeout = time.Duration(resolvedTimeouts.Effective.StepTimeoutSec) * time.Second
	}
	if resolvedTimeouts.Effective.HeartbeatSec > 0 {
		execution.runtimeHeartbeatInterval = time.Duration(resolvedTimeouts.Effective.HeartbeatSec) * time.Second
	}
	execution.onLog = func(entry RunLogEntry) {
		if strings.TrimSpace(entry.StepID) == "" {
			entry.StepID = execution.stepStatus.CurrentStep
		}
		s.appendRunLog(runID, entry)
	}
	execution.logInfo("", "", "runtime execution profile resolved", map[string]any{
		"strategy":         execution.executionProfile.Strategy,
		"max_parallel":     execution.executionProfile.MaxParallel,
		"failure_policy":   execution.executionProfile.FailurePolicy,
		"shard_discovery":  execution.executionProfile.ShardMode,
		"step_providers":   execution.stepProviders.StringMap(),
		"selected_scopes":  append([]string(nil), execution.selectedRepoScopes...),
		"timeout_step_sec": resolvedTimeouts.Effective.StepTimeoutSec,
		"timeout_hb_sec":   resolvedTimeouts.Effective.HeartbeatSec,
	})
	execution.onStep = func(stepID string) {
		progress := initialInfo
		progress.Status = RunStatusRunning
		progress.CurrentStep = stepID
		progress.StepProviders = execution.stepProviders.StringMap()
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
		execution.rewriteTerminalReports(RunStatusFailed)
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

	if len(execution.partialFailures) > 0 {
		finishedAt := s.clock().UTC()
		failedInfo := initialInfo
		failedInfo.Status = RunStatusFailed
		failedInfo.ErrorCode = runErrorCodePartialFailed
		failedInfo.Error = summarizePartialFailures(execution.partialFailures)
		failedInfo.CurrentStep = execution.stepStatus.CurrentStep
		execution.rewriteTerminalReports(RunStatusFailed)
		failedInfo.Warnings = append([]string(nil), execution.warnings...)
		failedInfo.FinishedAt = &finishedAt
		if qualityArtifact, qualityErr := execution.writeRunQualitySummary(RunStatusFailed, failedInfo.ErrorCode, failedInfo.Error); qualityErr == nil {
			execution.addArtifacts(qualityArtifact)
		} else {
			failedInfo.Warnings = append(failedInfo.Warnings, fmt.Sprintf("run quality summary write failed: %v", qualityErr))
		}
		s.storeRun(runRecord{
			info:      failedInfo,
			artifacts: execution.artifacts,
		})
		execution.logError(execution.stepStatus.CurrentStep, "", "run failed: partial shard failures detected", map[string]any{
			"error_code":            failedInfo.ErrorCode,
			"partial_failure_count": len(execution.partialFailures),
		})
		_ = s.cleanupRunLogs()
		return failedInfo, execution.artifacts, errors.New(failedInfo.Error)
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

type pipelineExecution struct {
	runID                    string
	pipeline                 Pipeline
	startedAt                time.Time
	workspace                workspace.Root
	runnerResolver           *stepRunnerResolver
	store                    model.Store
	compiler                 reports.Compiler
	clock                    func() time.Time
	artifacts                []Artifact
	artifactIndex            map[string]int
	findings                 []contracts.Finding
	questions                []contracts.Question
	coverage                 *contracts.Coverage
	domainRuns               map[string]domainRunSummary
	stepStatus               RunInfo
	onStep                   func(stepID string)
	onLog                    func(entry RunLogEntry)
	warnings                 []string
	runtimeStepMetrics       []runtimeStepQuality
	runtimeVersions          map[string]struct{}
	runtimeStepTimeout       time.Duration
	runtimeHeartbeatInterval time.Duration
	executionProfile         acpruntime.ExecutionValues
	partialFailures          []runtimeShardFailure
	resumeFromStep           string
	resumeSourceStep         string
	resolvedRepoPaths        map[string]string
	repoSelectionMode        string
	selectedRepoScopes       []string
	stepProviders            acpruntime.StepProviderValues
	collectOutcome           runtimeShardOutcome
	findingsOutcome          runtimeShardOutcome
	findingsSkipped          bool
	reportContext            reports.ReportRenderContext
	step0DraftManifest       *runtimeDraftManifest
	step0DraftRoot           string
	asIsDraftManifest        *runtimeDraftManifest
	asIsDraftRoot            string
	proposalsDraftManifest   *runtimeDraftManifest
	proposalsDraftRoot       string
	shardPacks               []contracts.ShardPackManifest
	finalRunIndex            *contracts.FinalRunIndex
	citationIndex            *contracts.CitationIndex
	validatorVerdict         *contracts.ValidatorVerdict
	semanticBase             *contracts.SemanticSnapshot
}

type runtimeTaskExecution struct {
	Task             acpruntime.Task
	RuntimeName      string
	RuntimeVersion   string
	Execution        contracts.RuntimeExecution
	Apply            model.ApplyReport
	ShardManifest    *contracts.ShardPackManifest
	ValidatorVerdict *contracts.ValidatorVerdict
}

type runtimePreparedExecution struct {
	Task           acpruntime.Task
	Execution      contracts.RuntimeExecution
	ExecutionRaw   []byte
	RuntimeName    string
	RuntimeVersion string
}

type runtimeShardFailure struct {
	StepID     string
	DomainID   string
	ShardID    string
	RepoScopes []string
	PathScopes []string
	Message    string
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
	startIdx := 0
	if strings.TrimSpace(e.resumeFromStep) != "" {
		if idx := indexOfPipelineStep(stepIDs, e.resumeFromStep); idx >= 0 {
			startIdx = idx
		}
	}
	for _, stepID := range stepIDs[startIdx:] {
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
		return e.runStepConstitution(ctx, stepID)
	case "init.step1.collect", "refresh.step1.collect":
		return e.runRuntimeStep(ctx, stepID)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		return e.runStepAsIs(ctx, stepID)
	case "init.step3.findings", "refresh.step3.findings":
		return e.runStepValidator(ctx, stepID)
	case "init.step4.proposals", "refresh.step4.proposals":
		return e.runStepProposals(ctx, stepID)
	default:
		return fmt.Errorf("unsupported step %q", stepID)
	}
}

func (e *pipelineExecution) runStepConstitution(ctx context.Context, stepID string) error {
	selectedScopes := normalizeOrderedUniqueStrings(e.selectedRepoScopes)
	execution, err := e.executeRuntimeTask(ctx, stepID, "constitution", selectedScopes, []string{"."}, "", "")
	if err != nil {
		return err
	}
	if err := e.publishValidatedConstitutionDrafts(execution); err != nil {
		return err
	}
	if err := e.materializeConstitutionSupportSurface(stepID); err != nil {
		return err
	}
	return nil
}

func (e *pipelineExecution) runRuntimeStep(ctx context.Context, stepID string) error {
	return e.runStepCollectByDomain(ctx, stepID)
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

	for _, domainID := range domainIDs {
		e.logInfo(stepID, domainID, "domain collect start", nil)
		scopeResolution, err := resolveRepoScopeForDomainCard(e.workspace, domainID, e.workspace.Manifest.Repos)
		if err != nil {
			return err
		}
		repoScope := strings.TrimSpace(scopeResolution.RepoScope)
		declaredRepoScope := strings.TrimSpace(scopeResolution.DeclaredRepoScope)
		hasDeclaredRepoScope := scopeResolution.HasDeclaredRepoScope
		unresolved := []string{}
		if scopeResolution.DomainIDMismatch {
			declaredDomainID := strings.TrimSpace(scopeResolution.DeclaredDomainID)
			questionID := fmt.Sprintf("q.domain.%s.id-mismatch", slugutil.Slugify(domainID))
			e.questions = mergeQuestions(e.questions, []contracts.Question{
				{
					ID:       questionID,
					Text:     fmt.Sprintf("Canonical domain card filename %q conflicts with declared id %q; runtime keeps filename as canonical id for deterministic artifacts", domainID, declaredDomainID),
					Priority: "high",
				},
			})
			e.logWarn(stepID, domainID, "domain card id mismatch", map[string]any{
				"filename_domain_id": domainID,
				"declared_domain_id": declaredDomainID,
			})
		}
		if hasDeclaredRepoScope && declaredRepoScope != "" && !scopeResolution.DeclaredRepoScopeKnown {
			unresolved = appendUniqueStrings(unresolved, "repo_scope")
			e.questions = mergeQuestions(e.questions, []contracts.Question{
				{
					ID:       fmt.Sprintf("q.domain.%s.unknown-repo-scope", slugutil.Slugify(domainID)),
					Text:     fmt.Sprintf("Canonical domain %q declares unknown repo_scope %q (not present in workspace.yaml)", domainID, declaredRepoScope),
					Priority: "high",
				},
			})
		} else if strings.TrimSpace(repoScope) == "" {
			unresolved = appendUniqueStrings(unresolved, "repo_scope")
			e.questions = mergeQuestions(e.questions, []contracts.Question{
				{
					ID:       fmt.Sprintf("q.domain.%s.missing-repo-scope", slugutil.Slugify(domainID)),
					Text:     fmt.Sprintf("Canonical domain %q has no matching repo scope in workspace.yaml", domainID),
					Priority: "high",
				},
			})
		}
		skipReason := ""
		if repoScope != "" && !e.isRepoScopeSelected(repoScope) {
			unresolved = appendUniqueStrings(unresolved, "repo_scope")
			questionText := fmt.Sprintf(
				"Canonical domain %q repo_scope %q is excluded by runtime repo_selection=%q; domain task is skipped",
				domainID,
				repoScope,
				e.repoSelectionMode,
			)
			if hasDeclaredRepoScope && scopeResolution.DeclaredRepoScopeKnown {
				questionText = fmt.Sprintf(
					"Canonical domain %q declares repo_scope %q, but it is excluded by runtime repo_selection=%q; domain task is skipped",
					domainID,
					declaredRepoScope,
					e.repoSelectionMode,
				)
			} else if hasDeclaredRepoScope {
				questionText = fmt.Sprintf(
					"Canonical domain %q declares unknown repo_scope %q; resolved fallback repo_scope %q is excluded by runtime repo_selection=%q; domain task is skipped",
					domainID,
					declaredRepoScope,
					repoScope,
					e.repoSelectionMode,
				)
			}
			e.questions = mergeQuestions(e.questions, []contracts.Question{
				{
					ID:       fmt.Sprintf("q.domain.%s.repo-scope-excluded-by-selection", slugutil.Slugify(domainID)),
					Text:     questionText,
					Priority: "high",
				},
			})
			skipReason = fmt.Sprintf("repo_scope %q excluded by runtime repo_selection=%q", repoScope, e.repoSelectionMode)
		}
		if skipReason == "" && len(normalizeOrderedUniqueStrings(e.selectedRepoScopes)) == 0 {
			unresolved = appendUniqueStrings(unresolved, "repo_scope")
			e.questions = mergeQuestions(e.questions, []contracts.Question{
				{
					ID:       fmt.Sprintf("q.domain.%s.repo-selection-empty", slugutil.Slugify(domainID)),
					Text:     fmt.Sprintf("Canonical domain %q is skipped because runtime repo_selection=%q selected zero repo scopes", domainID, e.repoSelectionMode),
					Priority: "high",
				},
			})
			skipReason = fmt.Sprintf("runtime repo_selection=%q selected zero repo scopes", e.repoSelectionMode)
		}
		envelopePath := fmt.Sprintf("reports/agent-outputs/domains/%s.task-envelope.json", sanitizeDomainArtifactSlug(domainID))
		outputPath := fmt.Sprintf("reports/agent-outputs/domains/%s.md", domainID)
		domainScopes := []string{}
		if strings.TrimSpace(repoScope) != "" && skipReason == "" {
			domainScopes = append(domainScopes, repoScope)
		}
		partialFailuresBefore := len(e.partialFailures)
		executions := []runtimeTaskExecution{}
		outcome := runtimeShardOutcome{}
		if skipReason == "" {
			executions, outcome, err = e.executeRuntimeTasksSharded(ctx, stepID, domainID, domainScopes, "domain-"+sanitizeDomainArtifactSlug(domainID))
			e.recordRuntimeStepOutcome(stepID, outcome)
			if err != nil {
				return err
			}
		} else {
			e.logWarn(stepID, domainID, "domain collect skipped", map[string]any{
				"repo_scope":       repoScope,
				"repo_selection":   e.repoSelectionMode,
				"selection_reason": skipReason,
			})
		}
		partialFailuresAfter := len(e.partialFailures)
		domainFailedShards := partialFailuresAfter - partialFailuresBefore
		if domainFailedShards < 0 {
			domainFailedShards = 0
		}
		if outcome.FailedShards > 0 {
			domainFailedShards = outcome.FailedShards
		}
		aggregatedApply := model.ApplyReport{}
		aggregatedQuestions := make([]contracts.Question, 0, len(executions))
		aggregatedFindings := make([]contracts.Finding, 0, len(executions))
		summaries := make([]string, 0, len(executions))
		for _, execution := range executions {
			if execution.ShardManifest != nil {
				aggregatedApply.UpsertedEntities += len(execution.ShardManifest.Semantic.Entities)
				aggregatedApply.UpsertedEdges += len(execution.ShardManifest.Semantic.Edges)
				aggregatedQuestions = append(aggregatedQuestions, execution.ShardManifest.Semantic.Questions...)
				aggregatedFindings = append(aggregatedFindings, execution.ShardManifest.Semantic.Findings...)
			}
			summary := ""
			if execution.ShardManifest != nil {
				summary = strings.TrimSpace(execution.ShardManifest.Summary)
			}
			if summary != "" {
				summaries = append(summaries, summary)
			}
		}
		questionIDs := extractQuestionIDs(aggregatedQuestions)
		findingIDs := extractFindingIDs(aggregatedFindings)
		domainTotalShards := outcome.PlannedShards
		if domainTotalShards == 0 {
			domainTotalShards = len(executions) + domainFailedShards
		}
		runtimeSummary := strings.Join(normalizeOrderedUniqueStrings(summaries), " | ")
		if runtimeSummary == "" {
			runtimeSummary = "none"
		}
		if skipReason != "" {
			runtimeSummary = "skipped: " + skipReason
		}
		if domainTotalShards > 1 || domainFailedShards > 0 {
			runtimeSummary = fmt.Sprintf(
				"%s [shards_total=%d succeeded=%d failed=%d]",
				runtimeSummary,
				domainTotalShards,
				outcome.SucceededShards,
				domainFailedShards,
			)
		}
		e.domainRuns[domainID] = domainRunSummary{
			DomainID:       domainID,
			RepoScope:      repoScope,
			TaskEnvelope:   envelopePath,
			OutputPath:     outputPath,
			RuntimeSummary: runtimeSummary,
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
	return nil
}

func (e *pipelineExecution) executeRuntimeTask(
	ctx context.Context,
	stepID string,
	taskSuffix string,
	repoScopes []string,
	pathScopes []string,
	domainID string,
	shardID string,
) (runtimeTaskExecution, error) {
	prepared, err := e.runRuntimeTaskNormalized(ctx, stepID, taskSuffix, repoScopes, pathScopes, domainID, shardID)
	if err != nil {
		return runtimeTaskExecution{}, err
	}
	executionPath := runtimeExecutionMetadataPathForTask(prepared.Task)
	executionLabel := strings.TrimSpace(stepID) + ".runtime-execution"
	if err := e.persistRuntimeExecutionArtifact(executionPath, executionLabel, prepared.ExecutionRaw); err != nil {
		return runtimeTaskExecution{}, err
	}
	return e.applyRuntimeTaskExecution(stepID, domainID, prepared)
}

func (e *pipelineExecution) runRuntimeTaskNormalized(
	ctx context.Context,
	stepID string,
	taskSuffix string,
	repoScopes []string,
	pathScopes []string,
	domainID string,
	shardID string,
) (runtimePreparedExecution, error) {
	taskID := fmt.Sprintf("task-%s-%s", e.runID, strings.ReplaceAll(stepID, ".", "-"))
	taskSuffix = strings.TrimSpace(taskSuffix)
	if taskSuffix != "" {
		taskID += "-" + taskSuffix
	}
	repoScope := primaryRepoScope(repoScopes)
	artifactRootRel, writeRootAbs, draftFinalRootAbs, readContextRoots, err := e.runtimeArtifactContext(stepID, strings.TrimSpace(shardID), repoScopes)
	if err != nil {
		return runtimePreparedExecution{}, err
	}
	resolvedProvider := acpruntime.ProviderClaudeCode
	runner := acpruntime.Runner(nil)
	if e.runnerResolver != nil {
		resolvedProvider, runner, err = e.runnerResolver.ReadyRunnerForStep(ctx, stepID)
		if err != nil {
			return runtimePreparedExecution{}, err
		}
	}
	task := acpruntime.Task{
		TaskID:            taskID,
		RunID:             e.runID,
		StepID:            stepID,
		ShardID:           strings.TrimSpace(shardID),
		DomainID:          strings.TrimSpace(domainID),
		Workspace:         e.workspace.Path,
		ArtifactRoot:      artifactRootRel,
		WriteRoot:         writeRootAbs,
		DraftFinalRoot:    draftFinalRootAbs,
		ReadContextRoots:  append([]string(nil), readContextRoots...),
		AgentRole:         runtimeAgentRole(stepID),
		StepContract:      runtimeStepContract(stepID),
		ExpectedArtifacts: append([]string(nil), runtimeExpectedArtifacts(stepID)...),
		RepoScope:         repoScope,
		RepoScopes:        append([]string(nil), repoScopes...),
		PathScopes:        append([]string(nil), pathScopes...),
		StartedAtUTC:      e.clock().UTC(),
		OnOutput: func(chunk acpruntime.OutputChunk) {
			e.logRuntimeOutput(stepID, domainID, chunk)
		},
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			e.logInfo(stepID, domainID, event.Message, event.Fields)
		},
	}
	e.logInfo(stepID, domainID, "runtime task started", map[string]any{
		"task_id":            task.TaskID,
		"shard_id":           task.ShardID,
		"repo_scope":         task.RepoScope,
		"repo_scopes":        task.RepoScopes,
		"path_scopes":        task.PathScopes,
		"provider":           resolvedProvider,
		"artifact_root":      task.ArtifactRoot,
		"write_root":         task.WriteRoot,
		"draft_final_root":   task.DraftFinalRoot,
		"read_context_roots": task.ReadContextRoots,
	})

	taskCtx := ctx
	cancel := func() {}
	if e.runtimeStepTimeout > 0 {
		taskCtx, cancel = context.WithTimeout(ctx, e.runtimeStepTimeout)
	}
	defer cancel()

	if runner == nil {
		return runtimePreparedExecution{}, fmt.Errorf("runtime runner resolver is not configured")
	}

	var heartbeatStop chan struct{}
	var heartbeatWG sync.WaitGroup
	if e.runtimeHeartbeatInterval > 0 {
		heartbeatStop = make(chan struct{})
		heartbeatTicker := time.NewTicker(e.runtimeHeartbeatInterval)
		startedAt := e.clock().UTC()
		heartbeatWG.Add(1)
		go func() {
			defer heartbeatWG.Done()
			defer heartbeatTicker.Stop()
			for {
				select {
				case <-heartbeatStop:
					return
				case <-heartbeatTicker.C:
					e.logInfo(stepID, domainID, "runtime task heartbeat", map[string]any{
						"task_id":     task.TaskID,
						"shard_id":    task.ShardID,
						"repo_scope":  task.RepoScope,
						"repo_scopes": task.RepoScopes,
						"path_scopes": task.PathScopes,
						"elapsed_sec": int(time.Since(startedAt).Seconds()),
					})
				}
			}
		}()
	}
	stopHeartbeat := func() {
		if heartbeatStop != nil {
			close(heartbeatStop)
		}
		heartbeatWG.Wait()
		heartbeatStop = nil
	}
	defer stopHeartbeat()
	result, err := runner.Run(taskCtx, task)
	if err != nil {
		if isDraftOnlyRuntimeStep(stepID) {
			if _, _, draftErr := validateRequiredRuntimeDraftArtifacts(task); draftErr != nil {
				e.logError(stepID, domainID, "runtime draft artifact validation failed", map[string]any{
					"task_id": task.TaskID,
					"error":   strings.TrimSpace(draftErr.Error()),
				})
				err = fmt.Errorf("%w: required runtime draft artifacts invalid: %v", err, draftErr)
			}
		}
		if errors.Is(taskCtx.Err(), context.DeadlineExceeded) {
			err = fmt.Errorf("runtime task timeout after %ds: %w", int(e.runtimeStepTimeout.Seconds()), err)
		}
		if failedExecution, ok := runtimeExecutionFromFailure(task, resolvedProvider, err, e.clock().UTC()); ok {
			failedExecutionLabel := strings.TrimSpace(stepID) + ".runtime-execution"
			if raw, marshalErr := json.MarshalIndent(failedExecution, "", "  "); marshalErr != nil {
				e.logWarn(stepID, domainID, "marshal failed runtime execution metadata failed", map[string]any{
					"task_id": task.TaskID,
					"error":   marshalErr.Error(),
				})
			} else if persistErr := e.persistRuntimeExecutionArtifact(runtimeExecutionMetadataPathForTask(task), failedExecutionLabel, raw); persistErr != nil {
				e.logWarn(stepID, domainID, "persist failed runtime execution metadata failed", map[string]any{
					"task_id": task.TaskID,
					"error":   persistErr.Error(),
				})
			}
		}
		e.logError(stepID, domainID, "runtime task failed", runtimeFailureLogFields(task, err, "", ""))
		return runtimePreparedExecution{}, err
	}
	execution := contracts.NormalizeRuntimeExecution(result.Execution)
	if strings.TrimSpace(execution.TaskID) == "" {
		execution = acpruntime.NewExecution(task, resolvedProvider, "", "succeeded", e.clock().UTC(), nil)
	}
	if isDraftOnlyRuntimeStep(stepID) {
		if _, _, draftErr := validateRequiredRuntimeDraftArtifacts(task); draftErr != nil {
			e.logError(stepID, domainID, "runtime draft artifact validation failed", map[string]any{
				"task_id": task.TaskID,
				"error":   strings.TrimSpace(draftErr.Error()),
			})
			return runtimePreparedExecution{}, fmt.Errorf("runtime required draft artifacts invalid: %w", draftErr)
		}
	}
	executionRaw, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return runtimePreparedExecution{}, fmt.Errorf("marshal runtime execution: %w", err)
	}
	runtimeName := strings.TrimSpace(execution.Provider)
	runtimeVersion := strings.TrimSpace(execution.RuntimeVersion)
	if runtimeName == "" {
		runtimeName = string(resolvedProvider)
	}

	return runtimePreparedExecution{
		Task:           task,
		Execution:      execution,
		ExecutionRaw:   executionRaw,
		RuntimeName:    runtimeName,
		RuntimeVersion: runtimeVersion,
	}, nil
}

func (e *pipelineExecution) applyRuntimeTaskExecution(
	stepID string,
	domainID string,
	prepared runtimePreparedExecution,
) (runtimeTaskExecution, error) {
	task := prepared.Task
	execution := prepared.Execution
	runtimeName := prepared.RuntimeName
	runtimeVersion := prepared.RuntimeVersion
	runtimeKey := runtimeName
	if runtimeVersion != "" {
		runtimeKey = runtimeName + "@" + runtimeVersion
	}
	e.runtimeVersions[runtimeKey] = struct{}{}

	if len(execution.Warnings) > 0 {
		for _, runtimeWarning := range execution.Warnings {
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

	if strings.HasSuffix(stepID, "step1.collect") {
		manifest, _, err := loadShardPackManifestFromRoot(task.WriteRoot)
		if err != nil {
			e.logError(stepID, domainID, "shard pack manifest load failed", map[string]any{
				"task_id": task.TaskID,
				"error":   strings.TrimSpace(err.Error()),
			})
			return runtimeTaskExecution{}, err
		}
		e.shardPacks = append(e.shardPacks, manifest)
		applyReport, err := e.store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
			Entities: manifest.Semantic.Entities,
			Edges:    manifest.Semantic.Edges,
		})
		if err != nil {
			e.logError(stepID, domainID, "semantic model apply failed", map[string]any{
				"task_id": task.TaskID,
				"error":   strings.TrimSpace(err.Error()),
			})
			return runtimeTaskExecution{}, err
		}
		e.questions = mergeQuestions(e.questions, manifest.Semantic.Questions)
		e.coverage = mergeCoverage(e.coverage, &manifest.Semantic.Coverage)
		e.findings = mergeFindings(e.findings, manifest.Semantic.Findings)
		manifestPath := path.Join(task.ArtifactRoot, shardPackManifestFile)
		e.addArtifacts(Artifact{
			Path:  manifestPath,
			Kind:  "taskrun",
			Label: "Shard Pack Manifest",
		})
		coverage := &manifest.Semantic.Coverage
		e.runtimeStepMetrics = append(e.runtimeStepMetrics, runtimeStepQuality{
			StepID:           stepID,
			DomainID:         domainID,
			RuntimeName:      runtimeName,
			RuntimeVersion:   runtimeVersion,
			RepoScopes:       append([]string(nil), task.RepoScopes...),
			SemanticEntities: len(manifest.Semantic.Entities),
			SemanticEdges:    len(manifest.Semantic.Edges),
			FindingsCount:    len(manifest.Semantic.Findings),
			QuestionsCount:   len(manifest.Semantic.Questions),
			CoverageObserved: countCoverageObserved(coverage),
			CoverageMissing:  countCoverageMissing(coverage),
			WarningsCount:    len(execution.Warnings),
		})
		e.logInfo(stepID, domainID, "runtime shard pack collected", map[string]any{
			"task_id":           task.TaskID,
			"shard_id":          task.ShardID,
			"artifact_root":     task.ArtifactRoot,
			"manifest_path":     manifestPath,
			"documents":         len(manifest.Documents),
			"citations":         len(manifest.Citations),
			"semantic_entities": len(manifest.Semantic.Entities),
			"semantic_edges":    len(manifest.Semantic.Edges),
			"semantic_findings": len(manifest.Semantic.Findings),
		})
		e.logInfo(stepID, domainID, "runtime task completed", map[string]any{
			"task_id":         task.TaskID,
			"shard_id":        task.ShardID,
			"runtime_name":    runtimeName,
			"runtime_version": runtimeVersion,
		})
		return runtimeTaskExecution{
			Task:           task,
			RuntimeName:    runtimeName,
			RuntimeVersion: runtimeVersion,
			Execution:      execution,
			Apply:          applyReport,
			ShardManifest:  &manifest,
		}, nil
	}

	if strings.HasSuffix(stepID, "step3.findings") {
		verdict, _, err := loadValidatorVerdictFromRoot(task.WriteRoot)
		if err != nil {
			e.logError(stepID, domainID, "validator verdict load failed", map[string]any{
				"task_id": task.TaskID,
				"error":   strings.TrimSpace(err.Error()),
			})
			return runtimeTaskExecution{}, err
		}
		if verdict.Verdict != "PASS" {
			return runtimeTaskExecution{}, fmt.Errorf("validator verdict is %s", verdict.Verdict)
		}
		e.questions = mergeQuestions(e.questions, verdict.Questions)
		e.findings = mergeFindings(e.findings, verdict.Findings)
		if err := e.assembleStagedDocFlow(); err != nil {
			return runtimeTaskExecution{}, err
		}
		if err := e.repairValidatorScopedArtifacts(&verdict); err != nil {
			e.logError(stepID, domainID, "validator scoped repair failed", map[string]any{
				"task_id": task.TaskID,
				"error":   strings.TrimSpace(err.Error()),
			})
			return runtimeTaskExecution{}, err
		}
		issues := e.validateStagedArtifacts()
		if len(issues) > 0 {
			return runtimeTaskExecution{}, fmt.Errorf("validator detected staged artifact issues: %s", issues[0].Message)
		}
		e.validatorVerdict = &verdict
		e.addArtifacts(Artifact{
			Path:  runtimeValidatorVerdictPath(e.runID),
			Kind:  "taskrun",
			Label: "Validator Verdict",
		})
		e.runtimeStepMetrics = append(e.runtimeStepMetrics, runtimeStepQuality{
			StepID:           stepID,
			DomainID:         domainID,
			RuntimeName:      runtimeName,
			RuntimeVersion:   runtimeVersion,
			RepoScopes:       append([]string(nil), task.RepoScopes...),
			FindingsCount:    len(verdict.Findings),
			QuestionsCount:   len(verdict.Questions),
			CoverageObserved: countCoverageObserved(e.coverage),
			CoverageMissing:  countCoverageMissing(e.coverage),
			WarningsCount:    len(execution.Warnings),
		})
		e.logInfo(stepID, domainID, "validator verdict accepted", map[string]any{
			"task_id":     task.TaskID,
			"checked":     len(verdict.CheckedPaths),
			"fixed_paths": len(verdict.FixedPaths),
		})
		e.logInfo(stepID, domainID, "runtime task completed", map[string]any{
			"task_id":         task.TaskID,
			"shard_id":        task.ShardID,
			"runtime_name":    runtimeName,
			"runtime_version": runtimeVersion,
		})
		return runtimeTaskExecution{
			Task:             task,
			RuntimeName:      runtimeName,
			RuntimeVersion:   runtimeVersion,
			Execution:        execution,
			ValidatorVerdict: &verdict,
		}, nil
	}

	if isDraftOnlyRuntimeStep(stepID) {
		e.runtimeStepMetrics = append(e.runtimeStepMetrics, runtimeStepQuality{
			StepID:         stepID,
			DomainID:       domainID,
			RuntimeName:    runtimeName,
			RuntimeVersion: runtimeVersion,
			RepoScopes:     append([]string(nil), task.RepoScopes...),
			WarningsCount:  len(execution.Warnings),
		})
		e.logInfo(stepID, domainID, "runtime draft step completed", map[string]any{
			"task_id":         task.TaskID,
			"shard_id":        task.ShardID,
			"runtime_name":    runtimeName,
			"runtime_version": runtimeVersion,
			"warnings_count":  len(execution.Warnings),
		})
		return runtimeTaskExecution{
			Task:           task,
			RuntimeName:    runtimeName,
			RuntimeVersion: runtimeVersion,
			Execution:      execution,
		}, nil
	}

	return runtimeTaskExecution{
		Task:           task,
		RuntimeName:    runtimeName,
		RuntimeVersion: runtimeVersion,
		Execution:      execution,
	}, nil
}

func (e *pipelineExecution) replayRuntimeTaskExecution(
	stepID string,
	domainID string,
	prepared runtimePreparedExecution,
) (runtimeTaskExecution, error) {
	return e.applyRuntimeTaskExecution(stepID, domainID, prepared)
}

func loadPreparedExecutionFromPersistedRuntimeExecution(raw []byte) (runtimePreparedExecution, error) {
	execution, err := contracts.ParseRuntimeExecution(raw)
	if err != nil {
		return runtimePreparedExecution{}, err
	}
	startedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(execution.StartedAt))
	if err != nil {
		return runtimePreparedExecution{}, fmt.Errorf("parse persisted runtime execution start time: %w", err)
	}
	runtimeName := strings.TrimSpace(execution.Provider)
	if runtimeName == "" {
		runtimeName = "unknown"
	}
	return runtimePreparedExecution{
		Task: acpruntime.Task{
			TaskID:         execution.TaskID,
			RunID:          execution.RunID,
			StepID:         execution.StepID,
			ShardID:        execution.ShardID,
			DomainID:       execution.DomainID,
			ArtifactRoot:   execution.ArtifactRoot,
			WriteRoot:      execution.WriteRoot,
			DraftFinalRoot: execution.DraftFinalRoot,
			RepoScope:      execution.RepoScope,
			RepoScopes:     append([]string(nil), execution.RepoScopes...),
			PathScopes:     append([]string(nil), execution.PathScopes...),
			StartedAtUTC:   startedAt.UTC(),
		},
		Execution:      execution,
		ExecutionRaw:   append([]byte(nil), raw...),
		RuntimeName:    runtimeName,
		RuntimeVersion: strings.TrimSpace(execution.RuntimeVersion),
	}, nil
}

func runtimeExecutionFromFailure(task acpruntime.Task, fallbackProvider acpruntime.Provider, err error, finishedAt time.Time) (contracts.RuntimeExecution, bool) {
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		return contracts.RuntimeExecution{}, false
	}
	provider := runnerErr.Provider
	if strings.TrimSpace(string(provider)) == "" {
		provider = fallbackProvider
	}
	status := "failed"
	switch runnerErr.Code {
	case acpruntime.ErrorCodeRuntimeTimeout:
		status = "timeout"
	case acpruntime.ErrorCodeRunCanceled:
		status = "canceled"
	}
	execution := acpruntime.NewExecution(task, provider, "", status, finishedAt, nil)
	execution.RawOutputRefs = runnerErr.RawOutputRefs
	return contracts.NormalizeRuntimeExecution(execution), true
}

func (e *pipelineExecution) persistRuntimeExecutionArtifact(path string, label string, raw []byte) error {
	if err := e.workspace.WriteFile(path, raw); err != nil {
		return err
	}
	e.addArtifacts(Artifact{Path: path, Kind: "runtime-execution", Label: label})
	e.logInfo(e.stepStatus.CurrentStep, "", "runtime execution persisted", map[string]any{
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

func (e *pipelineExecution) runStepAsIs(ctx context.Context, stepID string) error {
	if e.shouldReplayAsIsWithoutRuntime(stepID) {
		e.logInfo(stepID, "", "rebuilding staged doc flow from persisted collect artifacts", map[string]any{
			"resume_source_step": e.resumeSourceStep,
		})
		e.asIsDraftManifest = nil
		e.asIsDraftRoot = ""
		return e.assembleStagedDocFlow()
	}

	selectedScopes := normalizeOrderedUniqueStrings(e.selectedRepoScopes)
	execution, err := e.executeRuntimeTask(ctx, stepID, "as-is", selectedScopes, []string{"."}, "", "")
	if err != nil {
		return err
	}
	draft, _, err := validateRequiredRuntimeDraftArtifacts(execution.Task)
	if err != nil {
		return err
	}
	e.asIsDraftManifest = &draft
	e.asIsDraftRoot = execution.Task.DraftFinalRoot
	e.addArtifacts(Artifact{
		Path:  path.Join(execution.Task.ArtifactRoot, asisDraftManifestFile),
		Kind:  "taskrun",
		Label: "As-Is Draft Manifest",
	})
	e.logInfo(stepID, "", "assembling staged doc flow", nil)
	return e.assembleStagedDocFlow()
}

func (e *pipelineExecution) runStepValidator(ctx context.Context, stepID string) error {
	if e.shouldSkipFindingsRuntime() {
		e.markFindingsSkipped("findings_skipped_due_to_unusable_collect")
		e.addWarning(fmt.Sprintf("%s: validator step skipped because collect evidence is unusable", stepID))
		e.logWarn(stepID, "", "validator step skipped", map[string]any{
			"reason":         "collect evidence is unusable",
			"collect_status": e.renderContext().Collect.Status,
		})
		return nil
	}

	selectedScopes := normalizeOrderedUniqueStrings(e.selectedRepoScopes)
	if len(selectedScopes) == 0 {
		e.addWarning(fmt.Sprintf("%s: validator step skipped because repo_selection=%q selected zero repo scopes", stepID, e.repoSelectionMode))
		e.logWarn(stepID, "", "validator step skipped", map[string]any{
			"repo_selection_mode": e.repoSelectionMode,
			"selected_scopes":     append([]string(nil), e.selectedRepoScopes...),
		})
		return nil
	}
	execution, err := e.executeRuntimeTask(ctx, stepID, "validator-findings", append([]string(nil), selectedScopes...), []string{"."}, "", "")
	outcome := runtimeShardOutcome{PlannedShards: 1}
	if err != nil {
		outcome.FailedShards = 1
		e.recordRuntimeStepOutcome(stepID, outcome)
		return err
	}
	if execution.ValidatorVerdict != nil {
		outcome.SucceededShards = 1
	}
	e.recordRuntimeStepOutcome(stepID, outcome)
	return nil
}

func (e *pipelineExecution) runStepProposals(ctx context.Context, stepID string) error {
	selectedScopes := normalizeOrderedUniqueStrings(e.selectedRepoScopes)
	execution, err := e.executeRuntimeTask(ctx, stepID, "proposals", selectedScopes, []string{"."}, "", "")
	if err != nil {
		return err
	}
	draft, _, err := validateRequiredRuntimeDraftArtifacts(execution.Task)
	if err != nil {
		return err
	}
	e.proposalsDraftManifest = &draft
	e.proposalsDraftRoot = execution.Task.DraftFinalRoot
	e.addArtifacts(Artifact{
		Path:  path.Join(execution.Task.ArtifactRoot, proposalsDraftManifestFile),
		Kind:  "taskrun",
		Label: "Proposals Draft Manifest",
	})
	if e.validatorVerdict != nil {
		e.logInfo(stepID, "", "promoting validated staged artifacts", nil)
		if err := e.promoteValidatedArtifacts(); err != nil {
			return err
		}
		if e.proposalsDraftManifest != nil {
			if draftManifestHasPrefix(e.proposalsDraftManifest, "proposals/") && e.finalRunIndex != nil {
				removed := draftManifestCanonicalPathsWithPrefix(e.proposalsDraftManifest, "proposals/")
				for _, canonicalPath := range removed {
					target, resolveErr := e.workspace.Resolve(canonicalPath)
					if resolveErr != nil {
						return resolveErr
					}
					if removeErr := os.Remove(target); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
						return fmt.Errorf("remove deterministic proposal %q: %w", canonicalPath, removeErr)
					}
				}
				e.removeArtifactsByPath(removed...)
			}
			artifacts, err := applyRuntimeDraftOutputs(
				e.workspace,
				e.proposalsDraftRoot,
				*e.proposalsDraftManifest,
				"",
				func(target string) bool {
					return strings.HasPrefix(target, "proposals/") || strings.HasPrefix(target, "reports/changelog/")
				},
			)
			if err != nil {
				return err
			}
			e.addArtifacts(artifacts...)
		}
	} else if e.renderContext().IsIncomplete() || len(e.partialFailures) > 0 || e.findingsSkipped {
		e.addWarning(fmt.Sprintf("%s: canonical promotion skipped because validator verdict is missing", e.stepStatus.CurrentStep))
		e.logWarn(e.stepStatus.CurrentStep, "", "canonical promotion skipped", map[string]any{
			"reason":      "validator verdict is missing",
			"report_mode": e.renderContext().ReportMode,
		})
	} else {
		return fmt.Errorf("promote validated artifacts: validator verdict is missing")
	}

	if !draftManifestHasPrefix(e.proposalsDraftManifest, "reports/changelog/") {
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
	}
	e.logInfo(e.stepStatus.CurrentStep, "", "proposals and changelog compiled", map[string]any{
		"artifacts": len(e.artifacts),
	})
	return nil
}

func (e *pipelineExecution) rewriteTerminalReports(status RunStatus) {
	renderCtx := e.terminalRenderContext(status)
	if !renderCtx.IsIncomplete() {
		return
	}

	reportStep := strings.TrimSpace(e.stepStatus.CurrentStep)
	if reportStep == "" {
		reportStep = string(e.pipeline) + ".terminal"
	}

	logRewriteWarning := func(stage string, err error) {
		e.addWarning(fmt.Sprintf("terminal report rewrite failed (%s): %v", stage, err))
		e.logWarn(reportStep, "", "terminal report rewrite failed", map[string]any{
			"stage": stage,
			"error": err.Error(),
		})
	}

	if artifacts, err := e.compiler.WriteCoverage(e.coverage, e.questions, renderCtx); err != nil {
		logRewriteWarning("coverage", err)
	} else {
		e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	}

	if artifacts, err := e.compiler.WriteFindings(e.findings, renderCtx); err != nil {
		logRewriteWarning("findings", err)
	} else {
		e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	}

	if domainReports, err := e.authoredDomainReports(); err != nil {
		logRewriteWarning("domain-outputs.prepare", err)
	} else if artifacts, err := e.compiler.WriteDomainOutputs(domainReports); err != nil {
		logRewriteWarning("domain-outputs", err)
	} else {
		e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	}

	if artifacts, err := e.compiler.WriteDomainTaskEnvelopes(e.stagedDomainEnvelopes()); err != nil {
		logRewriteWarning("domain-task-envelopes", err)
	} else {
		e.addArtifacts(toOrchestratorArtifacts(artifacts)...)
	}

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

func indexOfPipelineStep(stepIDs []string, stepID string) int {
	target := strings.TrimSpace(stepID)
	for idx, candidate := range stepIDs {
		if candidate == target {
			return idx
		}
	}
	return -1
}

func resumeStepForCurrentStep(pipeline Pipeline, currentStep string) string {
	currentStep = strings.TrimSpace(currentStep)
	if currentStep == "" {
		return ""
	}
	stepIDs := stepIDsForPipeline(pipeline)
	if indexOfPipelineStep(stepIDs, currentStep) < 0 {
		return ""
	}
	firstRuntime := firstRuntimeStepForPipeline(pipeline)
	if firstRuntime == "" {
		return currentStep
	}
	if isRuntimeOrLaterStep(pipeline, currentStep) {
		return firstRuntime
	}
	return currentStep
}

func firstRuntimeStepForPipeline(pipeline Pipeline) string {
	switch pipeline {
	case PipelineInit:
		return "init.step1.collect"
	case PipelineRefresh:
		return "refresh.step1.collect"
	default:
		return ""
	}
}

func isRuntimeOrLaterStep(pipeline Pipeline, stepID string) bool {
	stepIDs := stepIDsForPipeline(pipeline)
	currentIdx := indexOfPipelineStep(stepIDs, stepID)
	runtimeIdx := indexOfPipelineStep(stepIDs, firstRuntimeStepForPipeline(pipeline))
	return currentIdx >= 0 && runtimeIdx >= 0 && currentIdx >= runtimeIdx
}

func artifactIndexFor(artifacts []Artifact) map[string]int {
	index := make(map[string]int, len(artifacts))
	for idx, artifact := range artifacts {
		index[artifact.Kind+"|"+artifact.Path] = idx
	}
	return index
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func hasShardArtifactsForRun(workspaceRoot string, runID string, stepID string) bool {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	runID = strings.TrimSpace(runID)
	stepID = strings.TrimSpace(stepID)
	if workspaceRoot == "" || runID == "" || stepID == "" {
		return false
	}
	stepSlug := strings.ReplaceAll(stepID, ".", "-")
	patterns := []string{
		filepath.Join(workspaceRoot, "reports", "taskruns", fmt.Sprintf("%s-%s-shard-summary*.json", runID, stepSlug)),
		filepath.Join(workspaceRoot, "reports", "taskruns", fmt.Sprintf("%s-%s-shard-plan*.json", runID, stepSlug)),
		filepath.Join(workspaceRoot, "reports", "taskruns", fmt.Sprintf("%s-%s-shard-*.json", runID, stepSlug)),
		filepath.Join(workspaceRoot, "reports", "taskruns", fmt.Sprintf("%s-%s-domain-*-shard-*.json", runID, stepSlug)),
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if _, statErr := os.Stat(match); statErr == nil {
				return true
			}
		}
	}
	return false
}

func summarizePartialFailures(failures []runtimeShardFailure) string {
	if len(failures) == 0 {
		return ""
	}
	parts := make([]string, 0, len(failures))
	for _, failure := range failures {
		step := strings.TrimSpace(failure.StepID)
		if step == "" {
			step = "runtime"
		}
		shard := strings.TrimSpace(failure.ShardID)
		if shard == "" {
			shard = "shard"
		}
		domain := strings.TrimSpace(failure.DomainID)
		message := strings.TrimSpace(failure.Message)
		if message == "" {
			message = "unknown shard error"
		}
		if domain != "" {
			parts = append(parts, fmt.Sprintf("%s[%s/%s]: %s", step, domain, shard, message))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s[%s]: %s", step, shard, message))
	}
	sort.Strings(parts)
	return fmt.Sprintf("partial shard failures (%d): %s", len(parts), strings.Join(parts, "; "))
}

func writeBaselineSupportBundle(ws workspace.Root) error {
	return ws.EnsureBaselineSupportBundle()
}

func collectRepoScopes(repos []workspace.RepoSource) []string {
	scopes := make([]string, 0, len(repos))
	for _, repo := range repos {
		scopes = append(scopes, repo.Name)
	}
	sort.Strings(scopes)
	return scopes
}

func appendUniqueStrings(values []string, additions ...string) []string {
	out := append([]string(nil), values...)
	seen := map[string]struct{}{}
	for _, value := range out {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		seen[trimmed] = struct{}{}
	}
	for _, candidate := range additions {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (e *pipelineExecution) isRepoScopeSelected(scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	selected := normalizeOrderedUniqueStrings(e.selectedRepoScopes)
	if len(selected) == 0 {
		mode := strings.ToLower(strings.TrimSpace(e.repoSelectionMode))
		return mode == "" || mode == "all"
	}
	for _, candidate := range selected {
		if candidate == scope {
			return true
		}
	}
	return false
}

func primaryRepoScope(scopes []string) string {
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
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

func (e *pipelineExecution) removeArtifactsByPath(paths ...string) {
	if len(paths) == 0 || len(e.artifacts) == 0 {
		return
	}
	removeSet := map[string]struct{}{}
	for _, item := range paths {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			removeSet[trimmed] = struct{}{}
		}
	}
	if len(removeSet) == 0 {
		return
	}
	filtered := make([]Artifact, 0, len(e.artifacts))
	for _, artifact := range e.artifacts {
		if _, exists := removeSet[artifact.Path]; exists {
			continue
		}
		filtered = append(filtered, artifact)
	}
	e.artifacts = filtered
	e.artifactIndex = artifactIndexFor(filtered)
}

func (e *pipelineExecution) shouldReplayAsIsWithoutRuntime(stepID string) bool {
	if !strings.HasSuffix(strings.TrimSpace(stepID), "step2.asis_docs") {
		return false
	}
	if len(e.shardPacks) == 0 {
		return false
	}
	resumeSourceStep := strings.TrimSpace(e.resumeSourceStep)
	if resumeSourceStep == "" {
		return false
	}
	stepIDs := stepIDsForPipeline(e.pipeline)
	sourceIdx := indexOfPipelineStep(stepIDs, resumeSourceStep)
	asIsIdx := indexOfPipelineStep(stepIDs, stepID)
	return asIsIdx >= 0 && sourceIdx > asIsIdx
}

func isDraftOnlyRuntimeStep(stepID string) bool {
	switch strings.TrimSpace(stepID) {
	case "init.step0.constitution", "init.step2.asis_docs", "refresh.step2.asis_docs", "init.step4.proposals", "refresh.step4.proposals":
		return true
	default:
		return false
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

func (e *pipelineExecution) logRuntimeOutput(stepID string, domainID string, chunk acpruntime.OutputChunk) {
	message := strings.TrimRight(chunk.Text, "\r\n")
	if strings.TrimSpace(message) == "" {
		return
	}
	entry := RunLogEntry{
		Timestamp: e.clock().UTC(),
		Level:     RunLogLevelInfo,
		Kind:      RunLogKindRuntimeOutput,
		Stream:    strings.TrimSpace(string(chunk.Stream)),
		StepID:    strings.TrimSpace(stepID),
		DomainID:  strings.TrimSpace(domainID),
		Message:   message,
	}
	if chunk.Truncated {
		entry.Fields = map[string]any{
			"output_truncated": true,
			"stream":           entry.Stream,
		}
	}
	if e.onLog != nil {
		e.onLog(entry)
	}
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
		"shard_id":    task.ShardID,
		"repo_scope":  task.RepoScope,
		"repo_scopes": append([]string(nil), task.RepoScopes...),
		"path_scopes": append([]string(nil), task.PathScopes...),
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

func (e *pipelineExecution) applySemanticGuards(stepID string, domainID string, task acpruntime.Task, normalized contracts.SemanticSnapshot) contracts.SemanticSnapshot {
	return normalized
}

func (e *pipelineExecution) ensureOwnerGapFallback(domainID string, task acpruntime.Task, normalized contracts.SemanticSnapshot) contracts.SemanticSnapshot {
	return normalized
}

func (e *pipelineExecution) ensureCrossRepoEdgeFallback(domainID string, task acpruntime.Task, normalized contracts.SemanticSnapshot) contracts.SemanticSnapshot {
	return normalized
}

func (e *pipelineExecution) applyEvidencePathSemanticGuard(stepID string, task acpruntime.Task, normalized contracts.SemanticSnapshot) contracts.SemanticSnapshot {
	return normalized
}

func entityRepoScope(entity contracts.Entity, task acpruntime.Task) string {
	if attributes, ok := entity.Attributes.(map[string]any); ok {
		if repoScope, ok := attributes["repo_scope"].(string); ok {
			repoScope = strings.TrimSpace(repoScope)
			if repoScope != "" {
				return repoScope
			}
		}
	}
	for _, evidence := range entity.Provenance.Evidence {
		repo := strings.TrimSpace(evidence.Repo)
		if repo != "" {
			return repo
		}
	}
	if len(task.RepoScopes) > 0 {
		return strings.TrimSpace(task.RepoScopes[0])
	}
	return ""
}

func fallbackEvidenceForEntity(entity contracts.Entity, repoScope string, workspaceRoot string, repoRoots map[string]string) (contracts.Evidence, bool) {
	for _, evidence := range entity.Provenance.Evidence {
		candidate := evidence
		repo := strings.TrimSpace(candidate.Repo)
		if repo == "" {
			repo = strings.TrimSpace(repoScope)
		}
		candidate.Repo = repo
		if evidencePathResolvesInScope(candidate, workspaceRoot, repoRoots) {
			return contracts.Evidence{
				Repo: strings.TrimSpace(candidate.Repo),
				Ref:  strings.TrimSpace(candidate.Ref),
				Path: strings.TrimSpace(candidate.Path),
			}, true
		}
	}

	repo := strings.TrimSpace(repoScope)
	if repo == "" {
		repo = "unknown"
	}
	for _, candidatePath := range []string{"README.md", "README", "go.mod", "package.json", "pom.xml"} {
		candidate := contracts.Evidence{
			Repo: repo,
			Path: candidatePath,
		}
		if evidencePathResolvesInScope(candidate, workspaceRoot, repoRoots) {
			return candidate, true
		}
	}
	return contracts.Evidence{}, false
}

func sanitizeProvenanceEvidence(provenance *contracts.Provenance, defaultRepo string, workspaceRoot string, repoRoots map[string]string) (removed int, downgraded bool) {
	if provenance == nil {
		return 0, false
	}
	filtered := make([]contracts.Evidence, 0, len(provenance.Evidence))
	for _, evidence := range provenance.Evidence {
		candidate := evidence
		if strings.TrimSpace(candidate.Repo) == "" && strings.TrimSpace(defaultRepo) != "" {
			candidate.Repo = strings.TrimSpace(defaultRepo)
		}
		if evidencePathResolvesInScope(candidate, workspaceRoot, repoRoots) {
			filtered = append(filtered, candidate)
			continue
		}
		removed++
	}
	provenance.Evidence = filtered
	if strings.TrimSpace(provenance.Kind) == "observation" && len(provenance.Evidence) == 0 {
		provenance.Kind = "inference"
		downgraded = true
	}
	return removed, downgraded
}

func declaredRepoRoots(ws workspace.Root) map[string]string {
	roots := map[string]string{}
	for _, repo := range ws.Manifest.Repos {
		name := strings.TrimSpace(repo.Name)
		if name == "" {
			continue
		}
		root := strings.TrimSpace(repo.Path)
		if root == "" && strings.TrimSpace(repo.GitURL) != "" {
			root = filepath.Join(ws.Path, ".acp", "repos", slugutil.Slugify(name))
		}
		if root == "" {
			continue
		}
		if !filepath.IsAbs(root) {
			root = filepath.Join(ws.Path, root)
		}
		root = filepath.Clean(root)
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		roots[normalizeSemanticKey(name)] = root
	}
	return roots
}

func evidencePathResolvesInScope(evidence contracts.Evidence, workspaceRoot string, repoRoots map[string]string) bool {
	raw := strings.TrimSpace(evidence.Path)
	if raw == "" {
		return false
	}
	if raw == "." || raw == "/" || raw == ".." {
		return false
	}

	normalized := normalizeEvidencePath(raw)
	if normalized == "" {
		return false
	}
	normalizedLower := strings.ToLower(normalized)
	for _, prefix := range []string{"search_source/", "search_query/", "search_config/", "web_search/", "external_search/"} {
		if strings.HasPrefix(strings.ToLower(raw), prefix) || strings.HasPrefix(normalizedLower, prefix) {
			return false
		}
	}
	if strings.Contains(normalizedLower, "://") {
		return false
	}

	candidate := filepath.Clean(filepath.FromSlash(normalized))
	if candidate == "." || candidate == ".." {
		return false
	}
	if filepath.IsAbs(candidate) {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			return false
		}
		if isPathWithinRoot(candidate, workspaceRoot) {
			return true
		}
		for _, root := range repoRoots {
			if isPathWithinRoot(candidate, root) {
				return true
			}
		}
		return false
	}

	var roots []string
	repoScope := normalizeSemanticKey(strings.TrimSpace(evidence.Repo))
	if repoScope != "" {
		if root, ok := repoRoots[repoScope]; ok {
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		roots = append(roots, workspaceRoot)
		for _, root := range repoRoots {
			roots = append(roots, root)
		}
	} else {
		roots = append(roots, workspaceRoot)
	}

	candidateVariants := []string{candidate}
	normalizedSlash := filepath.ToSlash(candidate)
	if strings.HasPrefix(normalizedSlash, "arch-workspace/") || strings.HasPrefix(normalizedSlash, "workspace/") {
		parts := strings.SplitN(normalizedSlash, "/", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			candidateVariants = append(candidateVariants, filepath.Clean(filepath.FromSlash(parts[1])))
		}
	}

	for _, variant := range candidateVariants {
		for _, root := range roots {
			absCandidate := filepath.Join(root, variant)
			info, err := os.Stat(absCandidate)
			if err == nil && !info.IsDir() {
				return true
			}
		}
	}
	return false
}

func normalizeEvidencePath(pathValue string) string {
	value := strings.TrimSpace(strings.Trim(pathValue, "`"))
	if value == "" {
		return ""
	}
	if queryIndex := strings.IndexByte(value, '?'); queryIndex >= 0 {
		value = value[:queryIndex]
	}
	if hashIndex := strings.IndexByte(value, '#'); hashIndex >= 0 {
		value = value[:hashIndex]
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if colon := strings.LastIndex(value, ":"); colon > 0 {
		prefix := value[:colon]
		suffix := value[colon+1:]
		if !looksLikeWindowsDrive(prefix) && looksLikeLineSuffix(suffix) {
			value = strings.TrimSpace(prefix)
		}
	}
	return strings.TrimSpace(value)
}

func looksLikeLineSuffix(value string) bool {
	parts := strings.Split(value, ":")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func looksLikeWindowsDrive(prefix string) bool {
	if len(prefix) < 2 {
		return false
	}
	letter := prefix[0]
	if (letter < 'A' || letter > 'Z') && (letter < 'a' || letter > 'z') {
		return false
	}
	return prefix[1] == ':'
}

func isPathWithinRoot(candidate string, root string) bool {
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absCandidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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

type domainRepoScopeResolution struct {
	DomainFileID           string
	DeclaredDomainID       string
	HasDeclaredDomainID    bool
	DomainIDMismatch       bool
	RepoScope              string
	DeclaredRepoScope      string
	HasDeclaredRepoScope   bool
	DeclaredRepoScopeKnown bool
}

func resolveRepoScopeForDomainCard(ws workspace.Root, domainID string, repos []workspace.RepoSource) (domainRepoScopeResolution, error) {
	cardPath := fmt.Sprintf("charter/cards/domains/%s.md", domainID)
	contentBytes, err := ws.ReadFile(cardPath)
	if err != nil {
		return domainRepoScopeResolution{}, err
	}
	return resolveRepoScopeForDomainCardContent(domainID, normalizeLineEndings(string(contentBytes)), repos), nil
}

func resolveRepoScopeForDomainCardContent(domainID string, content string, repos []workspace.RepoSource) domainRepoScopeResolution {
	resolution := domainRepoScopeResolution{
		DomainFileID: strings.TrimSpace(domainID),
	}
	declaredDomainID := strings.TrimSpace(extractCardField(content, "id"))
	if declaredDomainID != "" {
		resolution.DeclaredDomainID = declaredDomainID
		resolution.HasDeclaredDomainID = true
		if slugutil.Slugify(declaredDomainID) != slugutil.Slugify(domainID) {
			resolution.DomainIDMismatch = true
		}
	}

	declaredRepoScope := strings.TrimSpace(extractCardField(content, "repo_scope"))
	if declaredRepoScope != "" {
		resolution.DeclaredRepoScope = declaredRepoScope
		resolution.HasDeclaredRepoScope = true
		if repoScopeExists(declaredRepoScope, repos) {
			resolution.DeclaredRepoScopeKnown = true
			resolution.RepoScope = declaredRepoScope
			return resolution
		}
	}
	if strings.TrimSpace(resolution.RepoScope) == "" {
		resolution.RepoScope = strings.TrimSpace(repoScopeForDomain(domainID, repos))
	}
	return resolution
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

		scopeResolution := resolveRepoScopeForDomainCardContent(domainID, content, e.workspace.Manifest.Repos)
		repoScope := strings.TrimSpace(scopeResolution.RepoScope)

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
