package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/reports"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

func (e *pipelineExecution) runStepServiceCollect(ctx context.Context, stepID string) error {
	runs, err := e.executeServiceShards(ctx, stepID, e.servicePlan.SelectedShards)
	if err != nil {
		return err
	}
	e.serviceCollectRuns = runs

	coverageArtifacts, err := e.compiler.WriteCoverage(e.coverage, e.questions)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(coverageArtifacts)...)

	if err := e.buildDomainOutputsFromServiceRuns(stepID, runs); err != nil {
		return err
	}
	return nil
}

func (e *pipelineExecution) runStepServiceFindings(ctx context.Context, stepID string) error {
	runs, err := e.executeServiceShards(ctx, stepID, e.servicePlan.SelectedShards)
	if err != nil {
		return err
	}
	e.serviceFindingRuns = runs

	coverageArtifacts, err := e.compiler.WriteCoverage(e.coverage, e.questions)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(coverageArtifacts)...)

	findingArtifacts, err := e.compiler.WriteFindings(e.findings)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(findingArtifacts)...)
	return nil
}

func (e *pipelineExecution) runStepGlobalReview(ctx context.Context, stepID string) error {
	domainUnresolved := map[string][]string{}
	domainIDs := make([]string, 0, len(e.domainRuns))
	for domainID := range e.domainRuns {
		domainIDs = append(domainIDs, domainID)
	}
	sort.Strings(domainIDs)
	for _, domainID := range domainIDs {
		domainRun := e.domainRuns[domainID]
		if len(domainRun.Unresolved) == 0 {
			continue
		}
		domainUnresolved[domainID] = append([]string(nil), domainRun.Unresolved...)
	}
	inputPayload := map[string]any{
		"version":                    1,
		"run_id":                     e.runID,
		"pipeline":                   string(e.pipeline),
		"refresh_mode":               string(e.refreshMode),
		"service_count":              len(e.servicePlan.Services),
		"selected_shard_count":       len(e.servicePlan.SelectedShards),
		"collect_runs":               e.serviceCollectRuns,
		"findings_runs":              e.serviceFindingRuns,
		"removed_services":           e.servicePlan.RemovedServices,
		"global_unresolved_services": append([]string(nil), e.globalUnresolvedServices...),
		"domain_unresolved":          domainUnresolved,
		"warnings":                   normalizeOrderedUniqueStrings(e.servicePlan.Warnings),
		"generated_at":               e.clock().UTC().Format("2006-01-02T15:04:05Z"),
	}
	inputRaw, err := json.MarshalIndent(inputPayload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal global review input: %w", err)
	}
	inputRaw = append(inputRaw, '\n')
	inputPath := fmt.Sprintf("reports/taskruns/%s-global-review-input.json", e.runID)
	if err := e.workspace.WriteFile(inputPath, inputRaw); err != nil {
		return err
	}
	e.globalReviewInputPath = inputPath
	e.addArtifacts(Artifact{Path: inputPath, Kind: "taskrun", Label: "global review input"})

	prepared, err := e.runRuntimeTaskNormalized(ctx, stepID, "global-review", append([]string(nil), e.selectedRepoScopes...), []string{inputPath}, "", "global-review")
	if err != nil {
		return err
	}
	if len(prepared.Normalized.Warnings) > 0 {
		for _, warning := range prepared.Normalized.Warnings {
			e.addWarning(fmt.Sprintf("%s: %s", stepID, strings.TrimSpace(warning)))
		}
	}
	taskrunPath := fmt.Sprintf("reports/taskruns/%s-%s-global-review.json", e.runID, strings.ReplaceAll(stepID, ".", "-"))
	if err := e.persistTaskRun(taskrunPath, stepID+".global-review", prepared.NormalizedRaw); err != nil {
		return err
	}

	summary := e.renderGlobalReviewSummary(strings.TrimSpace(prepared.Normalized.Summary), taskrunPath)
	architectArtifacts, err := e.compiler.WriteArchitectSummary(summary)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(architectArtifacts)...)
	return nil
}

func (e *pipelineExecution) renderGlobalReviewSummary(runtimeSummary string, taskrunPath string) string {
	builder := strings.Builder{}
	builder.WriteString("# Architect Aggregation Summary\n\n")
	builder.WriteString(fmt.Sprintf("- total findings: %d\n", len(e.findings)))
	builder.WriteString(fmt.Sprintf("- total questions: %d\n", len(e.questions)))
	builder.WriteString(fmt.Sprintf("- discovered services: %d\n", len(e.servicePlan.Services)))
	builder.WriteString(fmt.Sprintf("- selected service shards: %d\n", len(e.servicePlan.SelectedShards)))
	builder.WriteString(fmt.Sprintf("- collect runtime shards: %d\n", len(e.serviceCollectRuns)))
	builder.WriteString(fmt.Sprintf("- findings runtime shards: %d\n", len(e.serviceFindingRuns)))
	if len(e.servicePlan.RemovedServices) > 0 {
		builder.WriteString(fmt.Sprintf("- removed services: %d\n", len(e.servicePlan.RemovedServices)))
	}
	if len(e.globalUnresolvedServices) > 0 {
		builder.WriteString(fmt.Sprintf("- unresolved service mappings: %d\n", len(e.globalUnresolvedServices)))
	}
	_ = taskrunPath
	if strings.TrimSpace(runtimeSummary) != "" {
		builder.WriteString(fmt.Sprintf("- runtime_global_review: %s\n", runtimeSummary))
	}

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

func (e *pipelineExecution) executeServiceShards(ctx context.Context, stepID string, shards []serviceShardPlan) ([]serviceRuntimeRun, error) {
	if len(shards) == 0 {
		e.logWarn(stepID, "", "service runtime skipped: no selected service shards", nil)
		if err := e.persistShardSummary(stepID, "services", nil); err != nil {
			return nil, err
		}
		return nil, nil
	}

	strategy := strings.TrimSpace(e.executionProfile.Strategy)
	if strategy == "" {
		strategy = "sequential"
	}
	maxParallel := e.executionProfile.MaxParallel
	if maxParallel <= 0 || strategy != "parallel" {
		maxParallel = 1
	}
	failurePolicy := strings.TrimSpace(e.executionProfile.FailurePolicy)
	if failurePolicy == "" {
		failurePolicy = "best_effort"
	}
	bestEffort := failurePolicy == "best_effort"
	e.logInfo(stepID, "", "runtime shard execution prepared", map[string]any{
		"strategy":       strategy,
		"max_parallel":   maxParallel,
		"failure_policy": failurePolicy,
		"shards":         len(shards),
	})
	plans := make([]runtimeShardPlan, 0, len(shards))
	for _, shard := range shards {
		plans = append(plans, runtimeShardPlan{
			SortKey:     shard.SortKey,
			ShardID:     shard.ShardID,
			RepoScopes:  []string{shard.RepoScope},
			PathScopes:  append([]string(nil), shard.PathScopes...),
			PrimaryRepo: shard.RepoScope,
		})
	}
	if err := e.persistShardPlan(stepID, "services", plans, nil, nil, strategy, maxParallel, failurePolicy); err != nil {
		return nil, err
	}

	type preparedResult struct {
		Plan     serviceShardPlan
		Prepared runtimePreparedExecution
		Err      error
	}
	results := make([]preparedResult, len(shards))

	if maxParallel <= 1 || len(shards) <= 1 {
		for idx, shard := range shards {
			taskSuffix := fmt.Sprintf("%s-%s", slugutil.Slugify(shard.ServiceID), slugutil.Slugify(shard.ShardID))
			prepared, err := e.runRuntimeTaskNormalized(ctx, stepID, taskSuffix, []string{shard.RepoScope}, append([]string(nil), shard.PathScopes...), shard.ServiceID, shard.ShardID)
			results[idx] = preparedResult{Plan: shard, Prepared: prepared, Err: err}
			if err != nil && !bestEffort {
				return nil, err
			}
		}
	} else {
		runCtx := ctx
		cancel := func() {}
		if !bestEffort {
			runCtx, cancel = context.WithCancel(ctx)
		}
		defer cancel()
		sem := make(chan struct{}, maxParallel)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		for idx, shard := range shards {
			wg.Add(1)
			go func(index int, plan serviceShardPlan) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				taskSuffix := fmt.Sprintf("%s-%s", slugutil.Slugify(plan.ServiceID), slugutil.Slugify(plan.ShardID))
				prepared, err := e.runRuntimeTaskNormalized(runCtx, stepID, taskSuffix, []string{plan.RepoScope}, append([]string(nil), plan.PathScopes...), plan.ServiceID, plan.ShardID)
				mu.Lock()
				results[index] = preparedResult{Plan: plan, Prepared: prepared, Err: err}
				if err != nil && !bestEffort && firstErr == nil {
					firstErr = err
					cancel()
				}
				mu.Unlock()
			}(idx, shard)
		}
		wg.Wait()
		if firstErr != nil {
			return nil, firstErr
		}
	}

	runs := make([]serviceRuntimeRun, 0, len(shards))
	summary := make([]runtimeShardSummaryEntry, 0, len(shards))
	for _, result := range results {
		if result.Err != nil {
			summary = append(summary, runtimeShardSummaryEntry{
				ShardID:    result.Plan.ShardID,
				RepoScopes: []string{result.Plan.RepoScope},
				PathScopes: append([]string(nil), result.Plan.PathScopes...),
				Status:     "failed",
				Error:      strings.TrimSpace(result.Err.Error()),
			})
			if bestEffort {
				e.partialFailures = append(e.partialFailures, runtimeShardFailure{
					StepID:     stepID,
					DomainID:   result.Plan.ServiceID,
					ShardID:    result.Plan.ShardID,
					RepoScopes: []string{result.Plan.RepoScope},
					PathScopes: append([]string(nil), result.Plan.PathScopes...),
					Message:    strings.TrimSpace(result.Err.Error()),
				})
				continue
			}
			return nil, result.Err
		}

		execution, err := e.applyRuntimeTaskExecution(stepID, result.Plan.ServiceID, result.Prepared)
		if err != nil {
			summary = append(summary, runtimeShardSummaryEntry{
				ShardID:    result.Plan.ShardID,
				RepoScopes: []string{result.Plan.RepoScope},
				PathScopes: append([]string(nil), result.Plan.PathScopes...),
				Status:     "failed",
				TaskID:     result.Prepared.Task.TaskID,
				Error:      strings.TrimSpace(err.Error()),
			})
			if bestEffort {
				e.partialFailures = append(e.partialFailures, runtimeShardFailure{
					StepID:     stepID,
					DomainID:   result.Plan.ServiceID,
					ShardID:    result.Plan.ShardID,
					RepoScopes: []string{result.Plan.RepoScope},
					PathScopes: append([]string(nil), result.Plan.PathScopes...),
					Message:    strings.TrimSpace(err.Error()),
				})
				continue
			}
			return nil, err
		}

		taskrunPath := shardTaskrunPath(
			e.runID,
			stepID,
			"service-"+sanitizeDomainArtifactSlug(result.Plan.ServiceID),
			result.Plan.ShardID,
			false,
		)
		if err := e.persistTaskRun(taskrunPath, fmt.Sprintf("%s.%s.%s", stepID, result.Plan.ServiceID, result.Plan.ShardID), result.Prepared.NormalizedRaw); err != nil {
			return nil, err
		}

		run := serviceRuntimeRun{
			RepoScope:      result.Plan.RepoScope,
			ServiceID:      result.Plan.ServiceID,
			ServiceRoot:    result.Plan.ServiceRoot,
			ShardID:        result.Plan.ShardID,
			PathScopes:     append([]string(nil), result.Plan.PathScopes...),
			TaskRunPath:    taskrunPath,
			RuntimeSummary: strings.TrimSpace(execution.Normalized.Summary),
			QuestionIDs:    extractQuestionIDs(execution.Normalized.Questions),
			FindingIDs:     extractFindingIDs(execution.Apply.Findings),
			Apply:          execution.Apply,
		}
		runs = append(runs, run)
		summary = append(summary, runtimeShardSummaryEntry{
			ShardID:    result.Plan.ShardID,
			RepoScopes: []string{result.Plan.RepoScope},
			PathScopes: append([]string(nil), result.Plan.PathScopes...),
			Status:     "succeeded",
			TaskID:     result.Prepared.Task.TaskID,
			TaskRun:    taskrunPath,
		})
	}

	if err := e.persistShardSummary(stepID, "services", summary); err != nil {
		return nil, err
	}
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].RepoScope != runs[j].RepoScope {
			return runs[i].RepoScope < runs[j].RepoScope
		}
		if runs[i].ServiceID != runs[j].ServiceID {
			return runs[i].ServiceID < runs[j].ServiceID
		}
		return runs[i].ShardID < runs[j].ShardID
	})
	return runs, nil
}

func (e *pipelineExecution) buildDomainOutputsFromServiceRuns(stepID string, runs []serviceRuntimeRun) error {
	domainIDs, err := loadCanonicalDomainIDs(e.workspace)
	if err != nil {
		return err
	}
	if len(domainIDs) == 0 {
		e.questions = mergeQuestions(e.questions, []contracts.Question{{
			ID:       "q.domains.missing-canonical-cards",
			Text:     "No canonical domain cards found in charter/cards/domains; create them via Step 0 wizard.",
			Priority: "high",
		}})
	}
	teamCards, err := loadCanonicalTeamCards(e.workspace)
	if err != nil {
		return err
	}
	if len(teamCards) == 0 {
		e.questions = mergeQuestions(e.questions, []contracts.Question{{
			ID:       "q.teams.missing-canonical-cards",
			Text:     "No canonical team cards found in charter/cards/teams; create them via Step 0 wizard.",
			Priority: "high",
		}})
	}

	type domainBucket struct {
		runs       []serviceRuntimeRun
		unresolved []string
	}
	type domainContext struct {
		repoScope              string
		declaredRepoScope      string
		hasDeclaredRepoScope   bool
		declaredRepoScopeKnown bool
		skipReason             string
		unresolved             []string
	}
	domainBuckets := map[string]*domainBucket{}
	domainContexts := map[string]domainContext{}
	for _, domainID := range domainIDs {
		domainBuckets[domainID] = &domainBucket{}
	}
	globalUnresolved := []string{}

	domainByRepo := map[string][]string{}
	selectedScopes := normalizeOrderedUniqueStrings(e.selectedRepoScopes)
	for _, domainID := range domainIDs {
		resolution, err := resolveRepoScopeForDomainCard(e.workspace, domainID, e.workspace.Manifest.Repos)
		if err != nil {
			return err
		}
		repoScope := strings.TrimSpace(resolution.RepoScope)
		declaredRepoScope := strings.TrimSpace(resolution.DeclaredRepoScope)
		hasDeclaredRepoScope := resolution.HasDeclaredRepoScope
		unresolved := []string{}
		skipReason := ""

		if resolution.DomainIDMismatch {
			declaredDomainID := strings.TrimSpace(resolution.DeclaredDomainID)
			e.questions = mergeQuestions(e.questions, []contracts.Question{{
				ID:       fmt.Sprintf("q.domain.%s.id-mismatch", slugutil.Slugify(domainID)),
				Text:     fmt.Sprintf("Canonical domain card filename %q conflicts with declared id %q; runtime keeps filename as canonical id for deterministic artifacts", domainID, declaredDomainID),
				Priority: "high",
			}})
		}
		if hasDeclaredRepoScope && declaredRepoScope != "" && !resolution.DeclaredRepoScopeKnown {
			unresolved = appendUniqueStrings(unresolved, "repo_scope")
			e.questions = mergeQuestions(e.questions, []contracts.Question{{
				ID:       fmt.Sprintf("q.domain.%s.unknown-repo-scope", slugutil.Slugify(domainID)),
				Text:     fmt.Sprintf("Canonical domain %q declares unknown repo_scope %q (not present in workspace.yaml)", domainID, declaredRepoScope),
				Priority: "high",
			}})
		} else if repoScope == "" {
			unresolved = appendUniqueStrings(unresolved, "repo_scope")
			e.questions = mergeQuestions(e.questions, []contracts.Question{{
				ID:       fmt.Sprintf("q.domain.%s.missing-repo-scope", slugutil.Slugify(domainID)),
				Text:     fmt.Sprintf("Canonical domain %q has no matching repo scope in workspace.yaml", domainID),
				Priority: "high",
			}})
		}
		if repoScope != "" && !e.isRepoScopeSelected(repoScope) {
			unresolved = appendUniqueStrings(unresolved, "repo_scope")
			questionText := fmt.Sprintf(
				"Canonical domain %q repo_scope %q is excluded by runtime repo_selection=%q; domain task is skipped",
				domainID,
				repoScope,
				e.repoSelectionMode,
			)
			if hasDeclaredRepoScope && resolution.DeclaredRepoScopeKnown {
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
			e.questions = mergeQuestions(e.questions, []contracts.Question{{
				ID:       fmt.Sprintf("q.domain.%s.repo-scope-excluded-by-selection", slugutil.Slugify(domainID)),
				Text:     questionText,
				Priority: "high",
			}})
			skipReason = fmt.Sprintf("repo_scope %q excluded by runtime repo_selection=%q", repoScope, e.repoSelectionMode)
		}
		if skipReason == "" && len(selectedScopes) == 0 {
			unresolved = appendUniqueStrings(unresolved, "repo_scope")
			e.questions = mergeQuestions(e.questions, []contracts.Question{{
				ID:       fmt.Sprintf("q.domain.%s.repo-selection-empty", slugutil.Slugify(domainID)),
				Text:     fmt.Sprintf("Canonical domain %q is skipped because runtime repo_selection=%q selected zero repo scopes", domainID, e.repoSelectionMode),
				Priority: "high",
			}})
			skipReason = fmt.Sprintf("runtime repo_selection=%q selected zero repo scopes", e.repoSelectionMode)
		}
		domainContexts[domainID] = domainContext{
			repoScope:              repoScope,
			declaredRepoScope:      declaredRepoScope,
			hasDeclaredRepoScope:   hasDeclaredRepoScope,
			declaredRepoScopeKnown: resolution.DeclaredRepoScopeKnown,
			skipReason:             skipReason,
			unresolved:             append([]string(nil), unresolved...),
		}
		if skipReason != "" || repoScope == "" {
			continue
		}
		domainByRepo[repoScope] = append(domainByRepo[repoScope], domainID)
	}
	for scope := range domainByRepo {
		sort.Strings(domainByRepo[scope])
	}

	for _, run := range runs {
		domains := append([]string(nil), domainByRepo[run.RepoScope]...)
		if len(domains) == 0 {
			globalUnresolved = append(globalUnresolved, fmt.Sprintf("%s(no-domain:%s)", run.ServiceID, run.RepoScope))
			continue
		}
		if len(domains) == 1 {
			domainBuckets[domains[0]].runs = append(domainBuckets[domains[0]].runs, run)
			continue
		}
		matched := matchDomainsByTokens(domains, run)
		if len(matched) == 1 {
			domainBuckets[matched[0]].runs = append(domainBuckets[matched[0]].runs, run)
			continue
		}
		globalUnresolved = append(globalUnresolved, fmt.Sprintf("%s(ambiguous:%s)", run.ServiceID, strings.Join(domains, ",")))
		for _, domainID := range domains {
			domainBuckets[domainID].unresolved = appendUniqueStrings(domainBuckets[domainID].unresolved, "service_mapping_ambiguous")
		}
		e.questions = mergeQuestions(e.questions, []contracts.Question{{
			ID:       fmt.Sprintf("q.domain.%s.service-mapping-ambiguous", slugutil.Slugify(run.ServiceID)),
			Text:     fmt.Sprintf("Service %q in repo_scope %q ambiguously maps to domains %q", run.ServiceID, run.RepoScope, strings.Join(domains, ", ")),
			Priority: "high",
		}})
	}
	if len(globalUnresolved) > 0 {
		e.questions = mergeQuestions(e.questions, []contracts.Question{{
			ID:       "q.domain.global.unresolved-service-mappings",
			Text:     fmt.Sprintf("Unresolved service mappings: %s", strings.Join(normalizeOrderedUniqueStrings(globalUnresolved), "; ")),
			Priority: "medium",
		}})
	}
	e.globalUnresolvedServices = normalizeOrderedUniqueStrings(globalUnresolved)

	domainReports := map[string]string{}
	domainEnvelopes := make([]reports.DomainTaskEnvelope, 0, len(domainIDs))
	e.domainRuns = map[string]domainRunSummary{}

	for _, domainID := range domainIDs {
		bucket := domainBuckets[domainID]
		ctx := domainContexts[domainID]
		repoScope := strings.TrimSpace(ctx.repoScope)
		unresolved := append([]string(nil), bucket.unresolved...)
		unresolved = appendUniqueStrings(unresolved, ctx.unresolved...)

		apply := model.ApplyReport{}
		summaries := []string{}
		questionIDs := []string{}
		findingIDs := []string{}
		for _, run := range bucket.runs {
			apply.UpsertedEntities += run.Apply.UpsertedEntities
			apply.UpsertedEdges += run.Apply.UpsertedEdges
			apply.Findings = append(apply.Findings, run.Apply.Findings...)
			questionIDs = append(questionIDs, run.QuestionIDs...)
			findingIDs = append(findingIDs, run.FindingIDs...)
			if strings.TrimSpace(run.RuntimeSummary) != "" {
				summaries = append(summaries, run.RuntimeSummary)
			}
		}
		questionIDs = uniqueSorted(questionIDs)
		findingIDs = uniqueSorted(findingIDs)
		runtimeSummary := strings.Join(normalizeOrderedUniqueStrings(summaries), " | ")
		if strings.TrimSpace(runtimeSummary) == "" {
			runtimeSummary = "none"
		}
		if strings.TrimSpace(ctx.skipReason) != "" {
			runtimeSummary = "skipped: " + strings.TrimSpace(ctx.skipReason)
		}

		envelopePath := fmt.Sprintf("reports/agent-outputs/domains/%s.task-envelope.json", sanitizeDomainArtifactSlug(domainID))
		outputPath := fmt.Sprintf("reports/agent-outputs/domains/%s.md", domainID)
		envelope := reports.DomainTaskEnvelope{
			ContractVersion: 1,
			AgentID:         "domain-analyst",
			DomainID:        domainID,
			RepoScope:       repoScope,
			Unresolved:      append([]string(nil), unresolved...),
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
		domainReports[domainID] = renderDomainRuntimeOutput(domainID, repoScope, envelopePath, runtimeSummary, apply, questionIDs, findingIDs, unresolved)
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

	if err := e.enrichCanonicalCards(domainIDs, teamCards); err != nil {
		return err
	}
	e.logInfo(stepID, "", "domain outputs aggregated from service runs", map[string]any{
		"domains": len(domainReports),
		"runs":    len(runs),
	})
	return nil
}

func matchDomainsByTokens(candidates []string, run serviceRuntimeRun) []string {
	serviceTokens := tokenizeServiceMapping(run.ServiceID + " " + run.ServiceRoot)
	matched := []string{}
	for _, candidate := range candidates {
		domainTokens := tokenizeServiceMapping(candidate)
		if tokensOverlap(serviceTokens, domainTokens) {
			matched = append(matched, candidate)
		}
	}
	sort.Strings(matched)
	return matched
}

func tokenizeServiceMapping(value string) map[string]struct{} {
	replacer := strings.NewReplacer("/", " ", "-", " ", "_", " ", ".", " ")
	normalized := strings.ToLower(strings.TrimSpace(replacer.Replace(value)))
	parts := strings.Fields(normalized)
	set := map[string]struct{}{}
	for _, part := range parts {
		if len(part) < 2 {
			continue
		}
		set[part] = struct{}{}
	}
	return set
}

func tokensOverlap(left map[string]struct{}, right map[string]struct{}) bool {
	for token := range left {
		if _, ok := right[token]; ok {
			return true
		}
	}
	return false
}
