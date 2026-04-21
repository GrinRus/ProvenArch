package orchestrator

import (
	"fmt"
	"path"
	"strings"

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
	if verdict.Verdict != "PASS" {
		return runtimeTaskExecution{}, fmt.Errorf("validator verdict is %s", verdict.Verdict)
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
