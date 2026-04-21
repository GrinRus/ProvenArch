package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
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
		if qualityArtifact, qualityErr := execution.writeRunQualitySummary(
			RunStatusFailed,
			failedInfo.ErrorCode,
			failedInfo.Error,
			classifyRunFailureSummary(execution.stepStatus.CurrentStep, err),
		); qualityErr == nil {
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
		if qualityArtifact, qualityErr := execution.writeRunQualitySummary(
			RunStatusFailed,
			failedInfo.ErrorCode,
			failedInfo.Error,
			runFailureClassification{
				Class:      failedInfo.ErrorCode,
				StepID:     strings.TrimSpace(execution.stepStatus.CurrentStep),
				ShortCause: strings.TrimSpace(failedInfo.Error),
				Source:     "orchestrator.partial_failures",
			},
		); qualityErr == nil {
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
	if qualityArtifact, qualityErr := execution.writeRunQualitySummary(RunStatusSucceeded, "", "", runFailureClassification{}); qualityErr == nil {
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
		return e.applyCollectRuntimeExecution(stepID, domainID, task, execution, runtimeName, runtimeVersion)
	}

	if strings.HasSuffix(stepID, "step3.findings") {
		return e.applyValidatorRuntimeExecution(stepID, domainID, task, execution, runtimeName, runtimeVersion)
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
