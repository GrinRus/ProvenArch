package orchestrator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func (e *pipelineExecution) applyCollectRuntimeExecution(
	stepID string,
	domainID string,
	task acpruntime.Task,
	execution contracts.RuntimeExecution,
	runtimeName string,
	runtimeVersion string,
) (runtimeTaskExecution, error) {
	manifest, _, err := loadShardPackManifestFromRoot(task.WriteRoot)
	if err != nil {
		e.logError(stepID, domainID, "shard pack manifest load failed", map[string]any{
			"task_id": task.TaskID,
			"error":   strings.TrimSpace(err.Error()),
		})
		return runtimeTaskExecution{}, err
	}
	if err := artifactquality.ValidateCollectManifestTaskIdentity(
		manifest,
		collectManifestIdentityValue(execution.RunID, task.RunID),
		collectManifestIdentityValue(execution.StepID, task.StepID),
		collectManifestIdentityValue(execution.ShardID, task.ShardID),
		collectManifestIdentityValue(execution.DomainID, task.DomainID),
		collectManifestIdentityValue(execution.ArtifactRoot, task.ArtifactRoot),
	); err != nil {
		e.logError(stepID, domainID, "shard pack manifest task identity failed", map[string]any{
			"task_id": task.TaskID,
			"error":   strings.TrimSpace(err.Error()),
		})
		return runtimeTaskExecution{}, err
	}
	if artifactquality.CollectManifestBootstrapOnly(manifest, collectManifestDocumentTexts(task.WriteRoot, manifest.Documents)) {
		warning := fmt.Sprintf(
			"artifact_quality: collect shard %s retained unchanged bootstrap first-action artifacts; provider did not enrich repository evidence",
			strings.TrimSpace(task.ShardID),
		)
		e.warnings = append(e.warnings, warning)
		e.logWarn(stepID, domainID, "collect manifest bootstrap-only quality warning", map[string]any{
			"task_id":  task.TaskID,
			"shard_id": task.ShardID,
			"warning":  warning,
		})
	}
	guardedSemantic, guardWarnings := guardRefreshCollectSemantic(stepID, task, manifest.Semantic)
	if len(guardWarnings) > 0 {
		for _, warning := range guardWarnings {
			e.addWarning(warning)
			e.logWarn(stepID, domainID, "refresh semantic guard filtered candidate", map[string]any{
				"task_id":  task.TaskID,
				"shard_id": task.ShardID,
				"warning":  warning,
			})
		}
		manifest.Semantic = guardedSemantic
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

func collectManifestIdentityValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func collectManifestDocumentTexts(writeRoot string, documents []contracts.AuthoredDocument) map[string]string {
	root := filepath.Clean(strings.TrimSpace(writeRoot))
	if root == "" || root == "." || len(documents) == 0 {
		return nil
	}
	texts := map[string]string{}
	for _, document := range documents {
		rawPath := filepath.ToSlash(strings.TrimSpace(document.Path))
		if rawPath == "" {
			continue
		}
		cleanRel := filepath.Clean(filepath.FromSlash(rawPath))
		if filepath.IsAbs(cleanRel) || cleanRel == "." || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
			continue
		}
		absPath := filepath.Join(root, cleanRel)
		relToRoot, err := filepath.Rel(root, absPath)
		if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
			continue
		}
		raw, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		texts[rawPath] = string(raw)
	}
	if len(texts) == 0 {
		return nil
	}
	return texts
}

func (e *pipelineExecution) applyValidatorRuntimeExecution(
	stepID string,
	domainID string,
	task acpruntime.Task,
	execution contracts.RuntimeExecution,
	runtimeName string,
	runtimeVersion string,
) (runtimeTaskExecution, error) {
	verdict, _, err := loadValidatorVerdictFromRoot(task.WriteRoot)
	if err != nil {
		e.logError(stepID, domainID, "validator verdict load failed", map[string]any{
			"task_id": task.TaskID,
			"error":   strings.TrimSpace(err.Error()),
		})
		return runtimeTaskExecution{}, err
	}

	e.questions = mergeQuestions(e.questions, verdict.Questions)
	e.findings = mergeFindings(e.findings, verdict.Findings)
	if err := e.assembleStagedDocFlow(); err != nil {
		return runtimeTaskExecution{}, err
	}
	if _, err := e.applyValidatorRepairStage(stepID, domainID, task.TaskID, &verdict); err != nil {
		return runtimeTaskExecution{}, err
	}
	if reconciled, err := e.reconcileOwnerGapOnlyVerdict(&verdict); err != nil {
		return runtimeTaskExecution{}, err
	} else if reconciled {
		e.logInfo(stepID, domainID, "owner-gap verdict downgraded to pass", map[string]any{
			"task_id": task.TaskID,
		})
	}

	issues := e.validateStagedArtifacts()
	if len(issues) > 0 {
		return runtimeTaskExecution{}, fmt.Errorf("validator detected staged artifact issues: %s", issues[0].Message)
	}
	if reconciled, err := e.reconcileEvidenceAdvisoryOnlyVerdict(&verdict); err != nil {
		return runtimeTaskExecution{}, err
	} else if reconciled {
		e.logInfo(stepID, domainID, "source-evidence validator issues downgraded to advisory", map[string]any{
			"task_id": task.TaskID,
		})
	}
	if verdict.Verdict != "PASS" {
		return runtimeTaskExecution{}, fmt.Errorf("validator verdict is %s", verdict.Verdict)
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
