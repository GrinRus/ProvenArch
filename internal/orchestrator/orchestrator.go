package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/refreshplan"
	"github.com/GrinRus/ProvenArch/internal/reports"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type Pipeline string

const (
	PipelineInit    Pipeline = "init"
	PipelineRefresh Pipeline = "refresh"
	PipelineQA      Pipeline = "qa"
)

const (
	runHistoryPath          = "reports/taskruns/run-history.json"
	runHistoryVersion       = 1
	runHistoryRetention     = 500
	historyDiagnosticsLimit = 20
	runLogsPath             = "reports/taskruns/logs"
)

const (
	runErrorCodeCanceled               = "run_canceled"
	runErrorCodeSuperseded             = "run_superseded"
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
	RunStatusCanceled  RunStatus = "canceled"
)

var (
	ErrRunNotFound      = errors.New("run not found")
	ErrRunNotCancelable = errors.New("run not cancelable")
	ErrRunActive        = errors.New("run is already active")
	ErrQueueUnsupported = errors.New("queue intent is supported only for refresh runs")
	ErrServiceClosed    = errors.New("service is shut down")
)

type RunIntent string

const (
	RunIntentStart RunIntent = "start"
	RunIntentQueue RunIntent = "queue"
)

type Service struct {
	runnerFactory      stepRunnerFactory
	clock              func() time.Time
	executionOverrides acpruntime.ExecutionOverrides
	resumeStaleAsync   bool
	providerFallback   acpruntime.Provider
	providerSource     acpruntime.ProviderSource
	runtimeMode        string

	mu             sync.RWMutex
	runs           map[string]*runRecord
	runIDSequence  int
	debounceWindow time.Duration
	closed         bool
	activeRunID    string
	pendingRun     *pendingRun
	runCancels     map[string]context.CancelFunc
	cancelRequests map[string]struct{}

	historyWorkspace           workspace.Root
	historyEnabled             bool
	historyRetention           int
	historyRecoveryDiagnostics []string
	lastHistoryPersistenceErr  error
	historyWriteFile           func(workspace.Root, string, []byte) error

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
	RunID                string                          `json:"run_id"`
	Pipeline             string                          `json:"pipeline"`
	Status               RunStatus                       `json:"status"`
	StartedAt            time.Time                       `json:"started_at"`
	FinishedAt           *time.Time                      `json:"finished_at,omitempty"`
	Question             string                          `json:"question,omitempty"`
	CurrentStep          string                          `json:"current_step,omitempty"`
	RuntimeMode          string                          `json:"runtime_mode,omitempty"`
	StepProviders        map[string]string               `json:"step_providers,omitempty"`
	ProviderModels       acpruntime.ProviderModelValues  `json:"provider_models,omitempty"`
	ProviderModelSources acpruntime.ProviderModelSources `json:"provider_model_sources,omitempty"`
	Warnings             []string                        `json:"warnings,omitempty"`
	PendingPermissions   []acpruntime.PermissionRequest  `json:"pending_permissions,omitempty"`
	ErrorCode            string                          `json:"error_code,omitempty"`
	Error                string                          `json:"error,omitempty"`
	SupersededByRunID    string                          `json:"superseded_by_run_id,omitempty"`
	RefreshSummary       *RefreshSummary                 `json:"refresh_summary,omitempty"`
	Progress             *RunProgress                    `json:"progress,omitempty"`
	Retry                *RetryLineage                   `json:"retry,omitempty"`
}

type RunProgress struct {
	Phase           string   `json:"phase"`
	CompletedSteps  int      `json:"completed_steps"`
	TotalSteps      int      `json:"total_steps"`
	CurrentStep     string   `json:"current_step,omitempty"`
	ExpectedResult  string   `json:"expected_result,omitempty"`
	PlannedUnits    int      `json:"planned_units,omitempty"`
	RunningUnits    int      `json:"running_units,omitempty"`
	SucceededUnits  int      `json:"succeeded_units,omitempty"`
	FailedUnits     int      `json:"failed_units,omitempty"`
	CurrentScopes   []string `json:"current_scopes,omitempty"`
	StartedAt       string   `json:"started_at"`
	ElapsedMS       int64    `json:"elapsed_ms"`
	LastActivityAt  string   `json:"last_activity_at,omitempty"`
	LastProgressAt  string   `json:"last_progress_at,omitempty"`
	ArtifactState   string   `json:"artifact_state,omitempty"`
	RepairAttempt   int      `json:"repair_attempt,omitempty"`
	RepairLimit     int      `json:"repair_limit,omitempty"`
	StallDeadlineAt string   `json:"stall_deadline_at,omitempty"`
}

type RetryLineage struct {
	ParentRunID        string   `json:"parent_run_id"`
	Reason             string   `json:"reason"`
	RequestedStep      string   `json:"requested_step"`
	EffectiveStartStep string   `json:"effective_start_step"`
	RequestedScopes    []string `json:"requested_scopes,omitempty"`
	EffectiveScopes    []string `json:"effective_scopes,omitempty"`
	ReusedInputs       []string `json:"reused_inputs,omitempty"`
}

type RefreshSummary struct {
	Mode          string   `json:"mode"`
	Decision      string   `json:"decision"`
	BaselineRunID string   `json:"baseline_run_id,omitempty"`
	ReasonCodes   []string `json:"reason_codes"`
	ArtifactPath  string   `json:"artifact_path"`
	Updated       int      `json:"updated"`
	Preserved     int      `json:"preserved"`
	Removed       int      `json:"removed"`
	Uncertain     int      `json:"uncertain"`
}

type Artifact struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type RunRequest struct {
	Workspace            workspace.Root
	Pipeline             Pipeline
	NonInteractive       bool
	Question             string
	Intent               RunIntent
	ResumeFromStep       string
	RetryParentRunID     string
	RetryReason          string
	RetryRequestedStep   string
	RetryRequestedScopes []string
	RetryScopes          []string
	RetryReusedInputs    []string
	ProviderModels       *acpruntime.ProviderModelResolution
}

type PendingRunInfo struct {
	RunID    string `json:"run_id"`
	Pipeline string `json:"pipeline"`
}

type RunCoordination struct {
	ActiveRunID string          `json:"active_run_id,omitempty"`
	Pending     *PendingRunInfo `json:"pending,omitempty"`
}

type runHistorySnapshot struct {
	Version int              `json:"version"`
	Items   []runHistoryItem `json:"items"`
}

type runHistoryItem struct {
	RunID                string                          `json:"run_id"`
	Pipeline             string                          `json:"pipeline"`
	Status               RunStatus                       `json:"status"`
	StartedAt            string                          `json:"started_at"`
	FinishedAt           *string                         `json:"finished_at,omitempty"`
	Question             string                          `json:"question,omitempty"`
	CurrentStep          string                          `json:"current_step,omitempty"`
	RuntimeMode          string                          `json:"runtime_mode,omitempty"`
	StepProviders        map[string]string               `json:"step_providers,omitempty"`
	ProviderModels       acpruntime.ProviderModelValues  `json:"provider_models,omitempty"`
	ProviderModelSources acpruntime.ProviderModelSources `json:"provider_model_sources,omitempty"`
	Warnings             []string                        `json:"warnings,omitempty"`
	PendingPermissions   []acpruntime.PermissionRequest  `json:"pending_permissions,omitempty"`
	ErrorCode            string                          `json:"error_code,omitempty"`
	Error                string                          `json:"error,omitempty"`
	SupersededByRunID    string                          `json:"superseded_by_run_id,omitempty"`
	RefreshSummary       *RefreshSummary                 `json:"refresh_summary,omitempty"`
	Progress             *RunProgress                    `json:"progress,omitempty"`
	Retry                *RetryLineage                   `json:"retry,omitempty"`
	Artifacts            []Artifact                      `json:"artifacts,omitempty"`
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

func WithRuntimeMode(mode string) Option {
	return func(service *Service) {
		if normalized, err := acpruntime.NormalizeMode(mode); err == nil {
			service.runtimeMode = normalized
		}
	}
}

func NewService(options ...Option) *Service {
	service := &Service{
		runnerFactory:    stepRunnerFactoryFunc(func(acpruntime.Provider) (acpruntime.Runner, error) { return fakeruntime.Runner{}, nil }),
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
		runtimeMode:      acpruntime.RuntimeModeFake,
		historyWriteFile: func(root workspace.Root, relPath string, content []byte) error {
			return root.WriteFileAtomic(relPath, content)
		},
	}
	for _, option := range options {
		option(service)
	}
	service.loadHistory()
	_ = service.cleanupRunLogs()
	return service
}

func (s *Service) SetRuntimeMode(mode string) {
	normalized, err := acpruntime.NormalizeMode(mode)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.runtimeMode = normalized
	s.mu.Unlock()
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

func (s *Service) ResolveProviderModelProfile(manifest workspace.Manifest) (acpruntime.ProviderModelResolution, error) {
	return acpruntime.ResolveProviderModels(manifest)
}

func (s *Service) resolveRunProviderModels(request RunRequest) (acpruntime.ProviderModelResolution, error) {
	if request.ProviderModels != nil {
		return *request.ProviderModels, nil
	}
	return s.ResolveProviderModelProfile(request.Workspace.Manifest)
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

func (s *Service) runWithID(ctx context.Context, request RunRequest, runID string) (finalInfo RunInfo, finalArtifacts []Artifact, finalErr error) {
	_ = s.cleanupRunLogs()
	now := s.clock().UTC()
	resumedRecord, resumed := s.loadExistingRunRecord(runID)
	startedAt := now
	initialArtifacts := []Artifact{}
	initialWarnings := []string{}
	resumeFromStep := strings.TrimSpace(request.ResumeFromStep)
	resolvedStepProviders, resolvedStepProvidersErr := s.ResolveStepProviderProfile(request.Workspace.Manifest)
	resolvedProviderModels, resolvedProviderModelsErr := s.resolveRunProviderModels(request)
	runLogMessage := "run started"
	runLogFields := map[string]any{
		"pipeline": string(request.Pipeline),
	}
	if resumed {
		// A restart must continue with the model snapshot accepted by the original run,
		// even if provider environment variables changed while the service was down.
		if len(resumedRecord.info.ProviderModels) > 0 {
			resolvedProviderModels.Effective = cloneProviderModelValues(resumedRecord.info.ProviderModels)
			resolvedProviderModels.Source = cloneProviderModelSources(resumedRecord.info.ProviderModelSources)
			resolvedProviderModelsErr = nil
		}
		startedAt = resumedRecord.info.StartedAt.UTC()
		initialArtifacts = append([]Artifact(nil), resumedRecord.artifacts...)
		initialWarnings = append([]string(nil), resumedRecord.info.Warnings...)
		if resumeFromStep == "" {
			resumeFromStep = resumeStepForCurrentStep(request.Pipeline, resumedRecord.info.CurrentStep)
		}
		runLogMessage = "run resumed after restart"
		runLogFields["previous_current_step"] = resumedRecord.info.CurrentStep
		runLogFields["resume_from_step"] = resumeFromStep
	}
	initialInfo := RunInfo{
		RunID:         runID,
		Pipeline:      string(request.Pipeline),
		Status:        RunStatusRunning,
		StartedAt:     startedAt,
		Question:      strings.TrimSpace(request.Question),
		CurrentStep:   resumeFromStep,
		RuntimeMode:   s.runtimeMode,
		StepProviders: map[string]string{},
		Warnings:      append([]string(nil), initialWarnings...),
	}
	initialInfo.Progress = newRunProgress(request.Pipeline, startedAt, resumeFromStep)
	if strings.TrimSpace(request.RetryParentRunID) != "" {
		initialInfo.Retry = &RetryLineage{
			ParentRunID:        strings.TrimSpace(request.RetryParentRunID),
			Reason:             strings.TrimSpace(request.RetryReason),
			RequestedStep:      retryRequestedStep(request),
			EffectiveStartStep: resumeFromStep,
			RequestedScopes:    retryRequestedScopes(request),
			EffectiveScopes:    append([]string(nil), request.RetryScopes...),
			ReusedInputs:       append([]string(nil), request.RetryReusedInputs...),
		}
	}
	if resumed && strings.TrimSpace(resumedRecord.info.RuntimeMode) != "" {
		initialInfo.RuntimeMode = resumedRecord.info.RuntimeMode
	}
	if resolvedStepProvidersErr == nil {
		initialInfo.StepProviders = resolvedStepProviders.Effective.StringMap()
	}
	if resolvedProviderModelsErr == nil {
		initialInfo.ProviderModels = resolvedProviderModels.Effective
		initialInfo.ProviderModelSources = resolvedProviderModels.Source
	}
	if err := s.storeRun(runRecord{
		info:      initialInfo,
		artifacts: append([]Artifact(nil), initialArtifacts...),
	}); err != nil {
		return initialInfo, initialArtifacts, fmt.Errorf("persist initial run state: %w", err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			panicErr := fmt.Errorf("run panic: %v", recovered)
			_ = s.terminalizeActiveRunAfterUnexpectedExit(runID, panicErr, "run failed: panic")
			panic(recovered)
		}
		if finalErr != nil {
			if terminalErr := s.terminalizeActiveRunAfterUnexpectedExit(runID, finalErr, "run failed: unexpected exit"); terminalErr != nil {
				finalErr = errors.Join(finalErr, fmt.Errorf("persist terminal guard state: %w", terminalErr))
			}
		}
	}()
	s.appendRunLog(runID, RunLogEntry{
		Timestamp: now,
		Level:     RunLogLevelInfo,
		Message:   runLogMessage,
		Fields:    runLogFields,
	})
	if resolvedStepProvidersErr != nil {
		return s.failRunBeforeExecution(
			runID,
			initialInfo,
			initialArtifacts,
			resolvedStepProvidersErr,
			resolvedStepProvidersErr,
			"run failed: runtime provider resolution",
			nil,
		)
	}
	if resolvedProviderModelsErr != nil {
		return s.failRunBeforeExecution(
			runID,
			initialInfo,
			initialArtifacts,
			resolvedProviderModelsErr,
			resolvedProviderModelsErr,
			"run failed: runtime model resolution",
			nil,
		)
	}
	if err := request.Workspace.EnsureLayout(); err != nil {
		return s.failRunBeforeExecution(
			runID,
			initialInfo,
			initialArtifacts,
			fmt.Errorf("ensure workspace layout: %w", err),
			err,
			"run failed: ensure workspace layout",
			nil,
		)
	}
	if request.Pipeline == PipelineQA {
		return s.runQAWithID(ctx, request, runID, initialInfo, initialArtifacts, resolvedStepProviders, resolvedProviderModels)
	}
	if strings.TrimSpace(request.RetryParentRunID) != "" && strings.TrimSpace(resumeFromStep) != "" {
		if err := copyRetryStaging(request.Workspace, request.RetryParentRunID, runID, resumeFromStep, request.RetryScopes); err != nil {
			return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, fmt.Errorf("prepare retry inputs: %w", err), err, "run failed: retry input preparation", nil)
		}
	}
	resolvedExecution := s.ResolveExecutionProfile(request.Workspace.Manifest)
	resolvedPermissions := acpruntime.ResolvePermissions(request.Workspace.Manifest)
	stepRunnerResolver := newStepRunnerResolver(s.runnerFactory, resolvedStepProviders.Effective)
	validation := request.Workspace.Validate(ctx, workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     true,
		VerifyRefs:   true,
	})
	if !validation.OK {
		validationErr := errors.New(formatValidationReportError(validation))
		return s.failRunBeforeExecution(
			runID,
			initialInfo,
			initialArtifacts,
			validationErr,
			validationErr,
			"run failed: workspace validation",
			diagnosticMessages(validation.Warnings),
		)
	}
	baselineCandidates := []string{}
	for _, prior := range s.ListRuns(0) {
		if prior.RunID == runID || prior.Status != RunStatusSucceeded || (prior.Pipeline != string(PipelineInit) && prior.Pipeline != string(PipelineRefresh)) {
			continue
		}
		baselineCandidates = append(baselineCandidates, prior.RunID)
	}
	baseline, priorEvidence := refreshplan.LoadLatestValidBaseline(request.Workspace, baselineCandidates)
	sourceRevisions := refreshplan.CaptureSourceRevisions(ctx, request.Workspace, validation.ResolvedRepos, string(request.Pipeline), runID, s.clock(), baseline, nil)
	sourceRaw, err := refreshplan.MarshalSourceRevisions(sourceRevisions)
	if err == nil {
		_, err = refreshplan.ParseSourceRevisions(sourceRaw)
	}
	if err == nil {
		err = request.Workspace.WriteFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "source-revisions.json")), append(sourceRaw, '\n'))
	}
	if err != nil {
		return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, fmt.Errorf("persist source revisions: %w", err), err, "run failed: source revision capture", nil)
	}
	sourceArtifact := Artifact{Path: filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "source-revisions.json")), Kind: "source-revisions", Label: "Source revisions"}
	if _, exists := artifactIndexFor(initialArtifacts)[sourceArtifact.Path]; !exists {
		initialArtifacts = append(initialArtifacts, sourceArtifact)
	}
	var refreshExecution *refreshplan.RefreshExecution
	var refreshImpact *refreshplan.ImpactPlan
	if request.Pipeline == PipelineRefresh {
		impact := refreshplan.BuildImpactPlan(ctx, sourceRevisions, baseline, validation.ResolvedRepos, priorEvidence, s.clock(), nil)
		refreshImpact = &impact
		impactRaw, impactErr := refreshplan.MarshalImpactPlan(impact)
		if impactErr == nil {
			_, impactErr = refreshplan.ParseImpactPlan(impactRaw)
		}
		if impactErr == nil {
			impactErr = request.Workspace.WriteFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "refresh-impact-plan.json")), append(impactRaw, '\n'))
		}
		if impactErr != nil {
			return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, fmt.Errorf("persist refresh impact plan: %w", impactErr), impactErr, "run failed: refresh impact planning", nil)
		}
		impactArtifact := Artifact{Path: filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "refresh-impact-plan.json")), Kind: "refresh-impact-plan", Label: "Refresh impact plan"}
		if _, exists := artifactIndexFor(initialArtifacts)[impactArtifact.Path]; !exists {
			initialArtifacts = append(initialArtifacts, impactArtifact)
		}
		executionAudit := refreshplan.NewRefreshExecution(runID, impact, sourceRevisions, s.clock())
		executionRaw, executionErr := refreshplan.MarshalRefreshExecution(executionAudit)
		if executionErr == nil {
			_, executionErr = refreshplan.ParseRefreshExecution(executionRaw)
		}
		if executionErr == nil {
			executionErr = request.Workspace.WriteFile(executionAudit.ArtifactPath, append(executionRaw, '\n'))
		}
		if executionErr != nil {
			return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, fmt.Errorf("persist refresh execution: %w", executionErr), executionErr, "run failed: refresh execution planning", nil)
		}
		initialArtifacts = append(initialArtifacts, Artifact{Path: executionAudit.ArtifactPath, Kind: "refresh-execution", Label: "Refresh execution"})
		baselineID := ""
		if executionAudit.BaselineRunID != nil {
			baselineID = *executionAudit.BaselineRunID
		}
		initialInfo.RefreshSummary = &RefreshSummary{Mode: executionAudit.Mode, Decision: executionAudit.PlanDecision, BaselineRunID: baselineID, ReasonCodes: refreshplan.SummaryReasonCodes(executionAudit), ArtifactPath: executionAudit.ArtifactPath}
		refreshExecution = &executionAudit
	}
	initialInfo.Warnings = append([]string(nil), initialWarnings...)
	if err := s.storeRun(runRecord{info: initialInfo, artifacts: append([]Artifact(nil), initialArtifacts...)}); err != nil {
		return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, fmt.Errorf("store planning artifacts: %w", err), err, "run failed: planning artifact registration", nil)
	}
	if refreshExecution != nil && refreshExecution.Mode == "no_op" {
		materializationArtifact, counts, materializationErr := writeRefreshMaterialization(request.Workspace, runID, *refreshExecution, priorEvidence.AllCanonicalPaths, "preserved", priorEvidence.AllCanonicalPaths, nil, s.clock().UTC().Format(time.RFC3339))
		if materializationErr != nil {
			return s.failRunBeforeExecution(runID, initialInfo, initialArtifacts, fmt.Errorf("persist refresh materialization: %w", materializationErr), materializationErr, "run failed: refresh materialization", nil)
		}
		initialArtifacts = append(initialArtifacts, materializationArtifact)
		initialInfo.RefreshSummary.Updated, initialInfo.RefreshSummary.Preserved, initialInfo.RefreshSummary.Removed, initialInfo.RefreshSummary.Uncertain = counts.Updated, counts.Preserved, counts.Removed, counts.Uncertain
		finishedAt := s.clock().UTC()
		initialInfo.Status = RunStatusSucceeded
		initialInfo.FinishedAt = &finishedAt
		initialInfo.CurrentStep = ""
		initialInfo.Progress = terminalRunProgress(initialInfo.Progress, RunStatusSucceeded, finishedAt)
		semanticRunID := runID
		if initialInfo.RefreshSummary != nil && strings.TrimSpace(initialInfo.RefreshSummary.BaselineRunID) != "" {
			semanticRunID = initialInfo.RefreshSummary.BaselineRunID
		}
		if snapshotArtifact, snapshotErr := persistPromotedArchitectureSnapshotFrom(request.Workspace, runID, semanticRunID, finishedAt); snapshotErr != nil {
			initialInfo.Warnings = append(initialInfo.Warnings, fmt.Sprintf("promoted architecture snapshot failed: %v", snapshotErr))
		} else if snapshotArtifact != nil {
			initialArtifacts = append(initialArtifacts, *snapshotArtifact)
		}
		if err := s.storeRun(runRecord{info: initialInfo, artifacts: append([]Artifact(nil), initialArtifacts...)}); err != nil {
			return initialInfo, initialArtifacts, err
		}
		s.appendRunLog(runID, RunLogEntry{Timestamp: finishedAt, Level: RunLogLevelInfo, Message: "refresh completed without provider execution", Fields: map[string]any{"mode": "no_op", "reasons": refreshExecution.ReasonCodes}})
		_ = s.cleanupRunLogs()
		return initialInfo, initialArtifacts, nil
	}

	execution := pipelineExecution{
		runID:            runID,
		pipeline:         request.Pipeline,
		workspace:        request.Workspace,
		store:            model.NewStore(request.Workspace),
		compiler:         reports.NewCompiler(request.Workspace),
		clock:            s.clock,
		refreshExecution: refreshExecution,
		preservedArtifactCandidates: func() []string {
			if refreshImpact == nil {
				return nil
			}
			return append([]string(nil), refreshImpact.PreservedArtifactCandidates...)
		}(),
		baselineCanonicalPaths: append([]string(nil), priorEvidence.AllCanonicalPaths...),
		baselineContentDigests: managedContentDigests(request.Workspace, priorEvidence.AllCanonicalPaths),
		staleArtifactCandidates: func() []string {
			if refreshImpact == nil {
				return nil
			}
			return append([]string(nil), refreshImpact.StaleArtifactCandidates...)
		}(),
		pipelineRunProgressState: pipelineRunProgressState{
			startedAt:        startedAt,
			stepStatus:       initialInfo,
			resumeFromStep:   resumeFromStep,
			resumeSourceStep: strings.TrimSpace(resumedRecord.info.CurrentStep),
			warnings:         append([]string(nil), initialWarnings...),
		},
		pipelineArtifactRegistry: pipelineArtifactRegistry{
			artifacts:     append([]Artifact(nil), initialArtifacts...),
			artifactIndex: artifactIndexFor(initialArtifacts),
		},
		pipelineRuntimeState: pipelineRuntimeState{
			runnerResolver:    stepRunnerResolver,
			runtimeVersions:   map[string]struct{}{},
			resolvedRepoPaths: map[string]string{},
			stepProviders:     resolvedStepProviders.Effective,
			providerModels:    resolvedProviderModels.Effective,
			permissionProfile: resolvedPermissions.Effective,
			retryScopes:       append([]string(nil), request.RetryScopes...),
		},
		pipelineQualityState: pipelineQualityState{
			runtimeStepMetrics:      []runtimeStepQuality{},
			runtimeRecoveryCounters: runtimeRecoveryCounters{},
		},
		pipelineSemanticDocflowState: pipelineSemanticDocflowState{
			findings:      []contracts.Finding{},
			questions:     []contracts.Question{},
			coverage:      nil,
			domainRuns:    map[string]domainRunSummary{},
			reportContext: reports.DefaultReportRenderContext(),
		},
	}
	for _, resolvedRepo := range validation.ResolvedRepos {
		name := strings.TrimSpace(resolvedRepo.Name)
		path := strings.TrimSpace(resolvedRepo.Path)
		if name == "" || path == "" {
			continue
		}
		execution.resolvedRepoPaths[name] = path
	}
	if strings.TrimSpace(request.RetryParentRunID) != "" {
		if err := execution.hydrateRetryInputs(); err != nil {
			return s.finishExecutionFailure(runID, initialInfo, &execution, fmt.Errorf("hydrate retry inputs: %w", err))
		}
	}
	if refreshImpact != nil {
		execution.refreshIntentContext = buildRefreshIntentContext(ctx, *refreshImpact, execution.resolvedRepoPaths)
	}
	resolvedTimeouts := acpruntime.ResolveTimeouts(request.Workspace.Manifest)
	execution.executionProfile = resolvedExecution.Effective
	if resolvedTimeouts.Effective.StepTimeoutSec > 0 {
		execution.runtimeStepTimeout = time.Duration(resolvedTimeouts.Effective.StepTimeoutSec) * time.Second
	}
	if resolvedTimeouts.Effective.HeartbeatSec > 0 {
		execution.runtimeHeartbeatInterval = time.Duration(resolvedTimeouts.Effective.HeartbeatSec) * time.Second
	}
	if execution.refreshExecution != nil && execution.refreshExecution.Mode == "affected_only" {
		if selectiveErr := execution.prepareSelectiveCollectBaseline(); selectiveErr != nil {
			if cleanupRoot, resolveErr := request.Workspace.Resolve(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "shards"))); resolveErr == nil {
				_ = os.RemoveAll(cleanupRoot)
			}
			execution.refreshExecution.Mode = "full"
			execution.refreshExecution.PreservedShards = []string{}
			execution.refreshExecution.ReasonCodes = append(execution.refreshExecution.ReasonCodes, "selective_baseline_unavailable")
			if persistErr := persistRefreshExecutionAudit(request.Workspace.WriteFile, execution.refreshExecution); persistErr != nil {
				return s.finishExecutionFailure(runID, initialInfo, &execution, persistErr)
			}
			initialInfo.RefreshSummary.Mode = "full"
			initialInfo.RefreshSummary.ReasonCodes = refreshplan.SummaryReasonCodes(*execution.refreshExecution)
			execution.stepStatus = initialInfo
			execution.warnings = append(execution.warnings, fmt.Sprintf("selective refresh fell back to full: %v", selectiveErr))
		} else if persistErr := persistRefreshExecutionAudit(request.Workspace.WriteFile, execution.refreshExecution); persistErr != nil {
			return s.finishExecutionFailure(runID, initialInfo, &execution, persistErr)
		}
		if err := s.storeRun(runRecord{info: initialInfo, artifacts: append([]Artifact(nil), initialArtifacts...)}); err != nil {
			return initialInfo, initialArtifacts, err
		}
	}
	progressTracker := newRunProgressTracker(request.Pipeline, startedAt, initialInfo.CurrentStep)
	execution.progressTracker = progressTracker
	var progressPersistMu sync.Mutex
	var lastOutputProgressPersist time.Time
	execution.onLog = func(entry RunLogEntry) {
		if strings.TrimSpace(entry.StepID) == "" {
			entry.StepID = execution.stepStatus.CurrentStep
		}
		s.appendRunLog(runID, entry)
		progressSnapshot := progressTracker.observe(entry)
		if !isPersistedRunProgressEvent(entry) {
			return
		}
		if entry.Kind == RunLogKindRuntimeOutput {
			progressPersistMu.Lock()
			if !lastOutputProgressPersist.IsZero() && entry.Timestamp.Sub(lastOutputProgressPersist) < 5*time.Second {
				progressPersistMu.Unlock()
				return
			}
			lastOutputProgressPersist = entry.Timestamp
			progressPersistMu.Unlock()
		}
		if current, ok := s.GetRun(runID); ok && current.Status != RunStatusQueued && current.Status != RunStatusRunning {
			return
		}
		progress := initialInfo
		progress.Status = RunStatusRunning
		progress.CurrentStep = execution.stepStatus.CurrentStep
		progress.StepProviders = execution.stepProviders.StringMap()
		progress.Warnings = append([]string(nil), execution.warnings...)
		progress.PendingPermissions = append([]acpruntime.PermissionRequest(nil), execution.pendingPermissions...)
		progress.Progress = progressSnapshot
		if err := s.storeRun(runRecord{info: progress, artifacts: append([]Artifact(nil), execution.artifacts...)}); err != nil {
			execution.recordProgressPersistenceError(err)
		}
	}
	execution.logInfo("", "", "runtime execution profile resolved", map[string]any{
		"strategy":                     execution.executionProfile.Strategy,
		"max_parallel":                 execution.executionProfile.MaxParallel,
		"failure_policy":               execution.executionProfile.FailurePolicy,
		"shard_discovery":              execution.executionProfile.ShardMode,
		"permissions_mode":             resolvedPermissions.Effective.Mode,
		"permissions_approval_channel": resolvedPermissions.Effective.ApprovalChannel,
		"step_providers":               execution.stepProviders.StringMap(),
		"repo_scopes":                  execution.repoScopes(),
		"source_repos":                 resolvedSourceRepoEvidence(validation.ResolvedRepos),
		"timeout_step_sec":             resolvedTimeouts.Effective.StepTimeoutSec,
		"timeout_hb_sec":               resolvedTimeouts.Effective.HeartbeatSec,
	})
	execution.onStep = func(stepID string) {
		progress := initialInfo
		progress.Status = RunStatusRunning
		progress.CurrentStep = stepID
		progress.StepProviders = execution.stepProviders.StringMap()
		progress.Warnings = append([]string(nil), execution.warnings...)
		progress.PendingPermissions = append([]acpruntime.PermissionRequest(nil), execution.pendingPermissions...)
		progress.Progress = progressTracker.beginStep(stepID, request.RetryScopes, s.clock().UTC())
		if err := s.storeRun(runRecord{
			info:      progress,
			artifacts: append([]Artifact(nil), execution.artifacts...),
		}); err != nil {
			execution.recordProgressPersistenceError(err)
		}
		execution.logInfo(stepID, "", "step started", nil)
	}
	execution.onPermissions = func(pending []acpruntime.PermissionRequest) {
		progress := initialInfo
		progress.Status = RunStatusRunning
		progress.CurrentStep = execution.stepStatus.CurrentStep
		progress.StepProviders = execution.stepProviders.StringMap()
		progress.Warnings = append([]string(nil), execution.warnings...)
		progress.PendingPermissions = append([]acpruntime.PermissionRequest(nil), pending...)
		progress.Progress = progressTracker.snapshot()
		if err := s.storeRun(runRecord{
			info:      progress,
			artifacts: append([]Artifact(nil), execution.artifacts...),
		}); err != nil {
			execution.recordProgressPersistenceError(err)
		}
	}

	if err := execution.run(ctx); err != nil {
		return s.finishExecutionFailure(runID, initialInfo, &execution, err)
	}

	if len(execution.partialFailures) > 0 {
		return s.finishPartialExecutionFailure(runID, initialInfo, &execution)
	}
	if execution.refreshExecution != nil {
		paths := make([]string, 0, len(execution.artifacts))
		for _, artifact := range execution.artifacts {
			paths = append(paths, artifact.Path)
		}
		if execution.refreshExecution.Mode == "affected_only" {
			preservedSet := map[string]struct{}{}
			for _, path := range execution.preservedCanonicalPaths {
				preservedSet[path] = struct{}{}
			}
			for _, path := range execution.preservedArtifactCandidates {
				baselineDigest, ok := execution.baselineContentDigests[path]
				if !ok {
					continue
				}
				current := managedContentDigests(request.Workspace, []string{path})[path]
				if current == baselineDigest {
					preservedSet[path] = struct{}{}
				}
			}
			execution.preservedCanonicalPaths = execution.preservedCanonicalPaths[:0]
			for path := range preservedSet {
				execution.preservedCanonicalPaths = append(execution.preservedCanonicalPaths, path)
			}
			sort.Strings(execution.preservedCanonicalPaths)
		}
		current := map[string]struct{}{}
		for _, path := range paths {
			if isManagedRefreshArtifact(path) {
				current[path] = struct{}{}
			}
		}
		stale := map[string]struct{}{}
		for _, path := range execution.staleArtifactCandidates {
			stale[path] = struct{}{}
		}
		removed := []string{}
		for _, path := range execution.baselineCanonicalPaths {
			if _, exists := current[path]; exists {
				continue
			}
			if execution.refreshExecution.Mode == "full" {
				removed = append(removed, path)
				continue
			}
			if _, affected := stale[path]; affected {
				removed = append(removed, path)
			}
		}
		materializationArtifact, counts, materializationErr := writeRefreshMaterialization(request.Workspace, runID, *execution.refreshExecution, paths, "updated", execution.preservedCanonicalPaths, removed, s.clock().UTC().Format(time.RFC3339))
		if materializationErr != nil {
			return s.finishExecutionFailure(runID, initialInfo, &execution, fmt.Errorf("persist refresh materialization: %w", materializationErr))
		}
		execution.addArtifacts(materializationArtifact)
		initialInfo.RefreshSummary.Updated, initialInfo.RefreshSummary.Preserved, initialInfo.RefreshSummary.Removed, initialInfo.RefreshSummary.Uncertain = counts.Updated, counts.Preserved, counts.Removed, counts.Uncertain
	}

	return s.finishExecutionSuccess(runID, initialInfo, &execution)
}

func retryRequestedStep(request RunRequest) string {
	if value := strings.TrimSpace(request.RetryRequestedStep); value != "" {
		return value
	}
	return strings.TrimSpace(request.ResumeFromStep)
}
func retryRequestedScopes(request RunRequest) []string {
	if request.RetryRequestedScopes != nil {
		return append([]string(nil), request.RetryRequestedScopes...)
	}
	return append([]string(nil), request.RetryScopes...)
}

type pipelineExecution struct {
	runID                       string
	pipeline                    Pipeline
	workspace                   workspace.Root
	store                       model.Store
	compiler                    reports.Compiler
	clock                       func() time.Time
	refreshExecution            *refreshplan.RefreshExecution
	preservedArtifactCandidates []string
	preservedCanonicalPaths     []string
	baselineCanonicalPaths      []string
	baselineContentDigests      map[string]string
	staleArtifactCandidates     []string
	pipelineRunProgressState
	pipelineArtifactRegistry
	pipelineRuntimeState
	pipelineQualityState
	pipelineSemanticDocflowState
	pipelineDraftState
}

type pipelineRunProgressState struct {
	startedAt              time.Time
	stepStatus             RunInfo
	progressTracker        *runProgressTracker
	onStep                 func(stepID string)
	onLog                  func(entry RunLogEntry)
	onPermissions          func([]acpruntime.PermissionRequest)
	warnings               []string
	pendingPermissions     []acpruntime.PermissionRequest
	resumeFromStep         string
	resumeSourceStep       string
	progressPersistenceMu  sync.Mutex
	progressPersistenceErr error
}

type pipelineArtifactRegistry struct {
	artifacts     []Artifact
	artifactIndex map[string]int
}

type pipelineRuntimeState struct {
	runnerResolver           *stepRunnerResolver
	runtimeVersions          map[string]struct{}
	runtimeStepTimeout       time.Duration
	runtimeHeartbeatInterval time.Duration
	executionProfile         acpruntime.ExecutionValues
	permissionProfile        acpruntime.PermissionValues
	refreshIntentContext     string
	partialFailures          []runtimeShardFailure
	resolvedRepoPaths        map[string]string
	stepProviders            acpruntime.StepProviderValues
	providerModels           acpruntime.ProviderModelValues
	collectOutcome           runtimeShardOutcome
	findingsOutcome          runtimeShardOutcome
	findingsSkipped          bool
	retryScopes              []string
	runtimeWriteAuditMu      sync.Mutex
}

type pipelineQualityState struct {
	runtimeStepMetrics      []runtimeStepQuality
	runtimeRecoveryCounters runtimeRecoveryCounters
}

type pipelineSemanticDocflowState struct {
	findings         []contracts.Finding
	questions        []contracts.Question
	coverage         *contracts.Coverage
	domainRuns       map[string]domainRunSummary
	reportContext    reports.ReportRenderContext
	shardPacks       []contracts.ShardPackManifest
	finalRunIndex    *contracts.FinalRunIndex
	citationIndex    *contracts.CitationIndex
	validatorVerdict *contracts.ValidatorVerdict
	semanticBase     *contracts.SemanticSnapshot
}

type pipelineDraftState struct {
	step0DraftManifest     *runtimedrafts.Manifest
	step0DraftRoot         string
	asIsDraftManifest      *runtimedrafts.Manifest
	asIsDraftRoot          string
	proposalsDraftManifest *runtimedrafts.Manifest
	proposalsDraftRoot     string
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

func cloneProviderModelValues(values acpruntime.ProviderModelValues) acpruntime.ProviderModelValues {
	if len(values) == 0 {
		return nil
	}
	cloned := make(acpruntime.ProviderModelValues, len(values))
	for provider, value := range values {
		cloned[provider] = value
	}
	return cloned
}

func cloneProviderModelSources(values acpruntime.ProviderModelSources) acpruntime.ProviderModelSources {
	if len(values) == 0 {
		return nil
	}
	cloned := make(acpruntime.ProviderModelSources, len(values))
	for provider, value := range values {
		cloned[provider] = value
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

func resolvedSourceRepoEvidence(repos []workspace.ResolvedRepo) []workspace.ResolvedRepo {
	out := make([]workspace.ResolvedRepo, 0, len(repos))
	for _, repo := range repos {
		if strings.TrimSpace(repo.Name) == "" {
			continue
		}
		out = append(out, workspace.ResolvedRepo{
			Name:        strings.TrimSpace(repo.Name),
			Source:      strings.TrimSpace(repo.Source),
			Path:        strings.TrimSpace(repo.Path),
			Ref:         strings.TrimSpace(repo.Ref),
			ResolvedSHA: strings.TrimSpace(repo.ResolvedSHA),
		})
	}
	return out
}

func (e *pipelineExecution) repoScopes() []string {
	return normalizeOrderedUniqueStrings(collectRepoScopes(e.workspace.Manifest.Repos))
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

func primaryRepoScope(scopes []string) string {
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
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
