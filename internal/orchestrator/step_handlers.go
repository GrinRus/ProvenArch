package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type domainCollectPreparation struct {
	DomainID     string
	RepoScope    string
	DomainScopes []string
	EnvelopePath string
	OutputPath   string
	Unresolved   []string
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
	execution, err := e.executeRuntimeTask(ctx, stepID, "constitution", e.repoScopes(), []string{"."}, "", "")
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
		preparedDomain, err := e.prepareDomainCollect(stepID, domainID)
		if err != nil {
			return err
		}
		partialFailuresBefore := len(e.partialFailures)
		executions := []runtimeTaskExecution{}
		outcome := runtimeShardOutcome{}
		executions, outcome, err = e.executeRuntimeTasksSharded(ctx, stepID, domainID, preparedDomain.DomainScopes, "domain-"+sanitizeDomainArtifactSlug(domainID))
		e.recordRuntimeStepOutcome(stepID, outcome)
		if err != nil {
			return err
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
			RepoScope:      preparedDomain.RepoScope,
			TaskEnvelope:   preparedDomain.EnvelopePath,
			OutputPath:     preparedDomain.OutputPath,
			RuntimeSummary: runtimeSummary,
			QuestionIDs:    questionIDs,
			FindingIDs:     findingIDs,
			Unresolved:     append([]string(nil), preparedDomain.Unresolved...),
		}
		e.logInfo(stepID, domainID, "domain collect completed", map[string]any{
			"repo_scope":       preparedDomain.RepoScope,
			"question_count":   len(questionIDs),
			"finding_count":    len(findingIDs),
			"unresolved_count": len(preparedDomain.Unresolved),
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
	return e.enrichCanonicalCards(domainIDs, teamCards)
}

func (e *pipelineExecution) prepareDomainCollect(stepID string, domainID string) (domainCollectPreparation, error) {
	scopeResolution, err := resolveRepoScopeForDomainCard(e.workspace, domainID, e.workspace.Manifest.Repos)
	if err != nil {
		return domainCollectPreparation{}, err
	}
	repoScope := strings.TrimSpace(scopeResolution.RepoScope)
	declaredRepoScope := strings.TrimSpace(scopeResolution.DeclaredRepoScope)
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
	if scopeResolution.HasDeclaredRepoScope && declaredRepoScope != "" && !scopeResolution.DeclaredRepoScopeKnown {
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

	domainScopes := []string{}
	if strings.TrimSpace(repoScope) != "" {
		domainScopes = append(domainScopes, repoScope)
	}
	return domainCollectPreparation{
		DomainID:     domainID,
		RepoScope:    repoScope,
		DomainScopes: domainScopes,
		EnvelopePath: fmt.Sprintf("reports/agent-outputs/domains/%s.task-envelope.json", sanitizeDomainArtifactSlug(domainID)),
		OutputPath:   fmt.Sprintf("reports/agent-outputs/domains/%s.md", domainID),
		Unresolved:   unresolved,
	}, nil
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
	return defaultRuntimeTaskExecutor{execution: e}.RunRuntimeTask(ctx, RuntimeTaskRequest{
		StepID:     stepID,
		TaskSuffix: taskSuffix,
		RepoScopes: append([]string(nil), repoScopes...),
		PathScopes: append([]string(nil), pathScopes...),
		DomainID:   domainID,
		ShardID:    shardID,
	})
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
	if e.shouldSkipFindingsRuntime() {
		reason := e.collectDownstreamSkipReason("asis_docs")
		message := e.collectDownstreamSkipMessage()
		e.addReportReason(reason)
		e.refreshReportContext()
		e.addWarning(fmt.Sprintf("%s: as-is doc step skipped because %s", stepID, message))
		e.logWarn(stepID, "", "as-is doc step skipped", map[string]any{
			"reason":         message,
			"skip_reason":    reason,
			"collect_status": e.renderContext().Collect.Status,
		})
		e.asIsDraftManifest = nil
		e.asIsDraftRoot = ""
		return e.assembleStagedDocFlow()
	}

	execution, err := e.executeRuntimeTask(ctx, stepID, "as-is", e.repoScopes(), []string{"."}, "", "")
	if err != nil {
		return err
	}
	draft, _, err := runtimedrafts.ValidateRequiredManifest(
		execution.Task.WriteRoot,
		execution.Task.DraftFinalRoot,
		execution.Task.RunID,
		execution.Task.StepID,
		execution.Task.StepContract,
		execution.Task.ExpectedArtifacts,
	)
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
	if err := e.assembleStagedDocFlow(); err != nil {
		return err
	}
	return e.preserveUnaffectedStagedDocuments()
}

func (e *pipelineExecution) runStepValidator(ctx context.Context, stepID string) error {
	if e.shouldSkipFindingsRuntime() {
		reason := e.collectDownstreamSkipReason("findings")
		message := e.collectDownstreamSkipMessage()
		e.markFindingsSkipped(reason)
		e.addWarning(fmt.Sprintf("%s: validator step skipped because %s", stepID, message))
		e.logWarn(stepID, "", "validator step skipped", map[string]any{
			"reason":         message,
			"skip_reason":    reason,
			"collect_status": e.renderContext().Collect.Status,
		})
		return nil
	}

	execution, err := e.executeRuntimeTask(ctx, stepID, "validator-findings", e.repoScopes(), []string{"."}, "", "")
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
	if e.shouldSkipFindingsRuntime() {
		reason := e.collectDownstreamSkipReason("proposals")
		message := e.collectDownstreamSkipMessage()
		e.addReportReason(reason)
		e.addWarning(fmt.Sprintf("%s: proposals step skipped because %s", stepID, message))
		e.logWarn(stepID, "", "proposals step skipped", map[string]any{
			"reason":         message,
			"skip_reason":    reason,
			"collect_status": e.renderContext().Collect.Status,
		})
		e.proposalsDraftManifest = nil
		e.proposalsDraftRoot = ""
		return nil
	}

	execution, err := e.executeRuntimeTask(ctx, stepID, "proposals", e.repoScopes(), []string{"."}, "", "")
	if err != nil {
		return err
	}
	draft, _, err := runtimedrafts.ValidateRequiredManifest(
		execution.Task.WriteRoot,
		execution.Task.DraftFinalRoot,
		execution.Task.RunID,
		execution.Task.StepID,
		execution.Task.StepContract,
		execution.Task.ExpectedArtifacts,
	)
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
		if e.proposalsDraftManifest != nil {
			if err := e.stageProposalDraftOutputsForFinalIndex(); err != nil {
				return err
			}
		}
		e.logInfo(stepID, "", "promoting validated staged artifacts", nil)
		if err := e.promoteValidatedArtifacts(); err != nil {
			return err
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

func (e *pipelineExecution) stageProposalDraftOutputsForFinalIndex() error {
	if e.proposalsDraftManifest == nil {
		return nil
	}
	if e.finalRunIndex == nil {
		return fmt.Errorf("stage proposal draft outputs: final run index is missing")
	}
	if e.citationIndex == nil {
		return fmt.Errorf("stage proposal draft outputs: citation index is missing")
	}
	stageRootRel := runtimeFinalArtifactRoot(e.runID)
	stageRootAbs, err := e.workspace.Resolve(stageRootRel)
	if err != nil {
		return err
	}
	stageRoot := workspace.Root{Path: stageRootAbs}
	draftArtifacts, err := applyRuntimeDraftOutputs(
		stageRoot,
		e.proposalsDraftRoot,
		*e.proposalsDraftManifest,
		stageRootRel,
		func(target string) bool {
			return strings.HasPrefix(target, "proposals/") || strings.HasPrefix(target, "reports/changelog/")
		},
	)
	if err != nil {
		return err
	}

	artifactsByPath := map[string]Artifact{}
	for _, document := range e.finalRunIndex.CanonicalDocuments {
		stagedPath := strings.TrimSpace(document.StagedPath)
		if stagedPath == "" {
			continue
		}
		artifactsByPath[stagedPath] = Artifact{
			Path:  stagedPath,
			Kind:  document.Kind,
			Label: document.Title,
		}
	}
	for _, artifact := range draftArtifacts {
		artifactsByPath[artifact.Path] = artifact
	}
	artifactPaths := make([]string, 0, len(artifactsByPath))
	for artifactPath := range artifactsByPath {
		artifactPaths = append(artifactPaths, artifactPath)
	}
	sort.Strings(artifactPaths)
	stageArtifacts := make([]Artifact, 0, len(artifactPaths))
	for _, artifactPath := range artifactPaths {
		stageArtifacts = append(stageArtifacts, artifactsByPath[artifactPath])
	}

	generatedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(e.finalRunIndex.GeneratedAt))
	if parseErr != nil {
		generatedAt = e.clock().UTC()
	}
	finalRunIndex, err := buildFinalRunIndex(
		e.runID,
		string(e.pipeline),
		generatedAt,
		stageArtifacts,
		e.shardPacks,
		aggregateDocumentInfos(e.shardPacks),
		*e.citationIndex,
		e.finalRunIndex.Semantic,
	)
	if err != nil {
		return err
	}
	citationIndex, citationChanged := reconcileRuntimeDerivedCitationDocuments(*e.citationIndex, finalRunIndex)
	if citationChanged {
		citationRaw, err := json.MarshalIndent(citationIndex, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal citation index with proposal drafts: %w", err)
		}
		citationRaw = append(citationRaw, '\n')
		if err := stageRoot.WriteFile(citationIndexFile, citationRaw); err != nil {
			return err
		}
		parsedCitationIndex, err := contracts.ParseCitationIndex(citationRaw)
		if err != nil {
			return err
		}
		e.citationIndex = &parsedCitationIndex
	}
	finalIndexRaw, err := json.MarshalIndent(finalRunIndex, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal final run index with proposal drafts: %w", err)
	}
	finalIndexRaw = append(finalIndexRaw, '\n')
	if err := stageRoot.WriteFile(finalRunIndexFile, finalIndexRaw); err != nil {
		return err
	}
	parsedFinalRunIndex, err := contracts.ParseFinalRunIndex(finalIndexRaw)
	if err != nil {
		return err
	}
	e.finalRunIndex = &parsedFinalRunIndex
	e.logInfo(e.stepStatus.CurrentStep, "", "proposal drafts staged into final run index", map[string]any{
		"draft_artifacts":     len(draftArtifacts),
		"canonical_documents": len(e.finalRunIndex.CanonicalDocuments),
	})
	return nil
}
