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
	"github.com/GrinRus/ProvenArch/internal/reports"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
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

func (s *Service) runWithID(ctx context.Context, request RunRequest, runID string) (finalInfo RunInfo, finalArtifacts []Artifact, finalErr error) {
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
	defer func() {
		if recovered := recover(); recovered != nil {
			panicErr := fmt.Errorf("run panic: %v", recovered)
			s.terminalizeActiveRunAfterUnexpectedExit(runID, panicErr, "run failed: panic")
			panic(recovered)
		}
		if finalErr != nil {
			s.terminalizeActiveRunAfterUnexpectedExit(runID, finalErr, "run failed: unexpected exit")
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
	resolvedExecution := s.ResolveExecutionProfile(request.Workspace.Manifest)
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

	execution := pipelineExecution{
		runID:     runID,
		pipeline:  request.Pipeline,
		workspace: request.Workspace,
		store:     model.NewStore(request.Workspace),
		compiler:  reports.NewCompiler(request.Workspace),
		clock:     s.clock,
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
		"repo_scopes":      execution.repoScopes(),
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
		return s.finishExecutionFailure(runID, initialInfo, &execution, err)
	}

	if len(execution.partialFailures) > 0 {
		return s.finishPartialExecutionFailure(runID, initialInfo, &execution)
	}

	return s.finishExecutionSuccess(runID, initialInfo, &execution)
}

type pipelineExecution struct {
	runID     string
	pipeline  Pipeline
	workspace workspace.Root
	store     model.Store
	compiler  reports.Compiler
	clock     func() time.Time
	pipelineRunProgressState
	pipelineArtifactRegistry
	pipelineRuntimeState
	pipelineQualityState
	pipelineSemanticDocflowState
	pipelineDraftState
}

type pipelineRunProgressState struct {
	startedAt        time.Time
	stepStatus       RunInfo
	onStep           func(stepID string)
	onLog            func(entry RunLogEntry)
	warnings         []string
	resumeFromStep   string
	resumeSourceStep string
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
	partialFailures          []runtimeShardFailure
	resolvedRepoPaths        map[string]string
	stepProviders            acpruntime.StepProviderValues
	collectOutcome           runtimeShardOutcome
	findingsOutcome          runtimeShardOutcome
	findingsSkipped          bool
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
	step0DraftManifest     *runtimeDraftManifest
	step0DraftRoot         string
	asIsDraftManifest      *runtimeDraftManifest
	asIsDraftRoot          string
	proposalsDraftManifest *runtimeDraftManifest
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
