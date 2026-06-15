package providercommon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

func recoverAfterArtifactValidationFailure(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if recovered, recoveredResult, recoveredErr := recoverFocusedArtifactRepair(ctx, task, adapter, result, validationErr, "contract"); recovered {
		return true, recoveredResult, recoveredErr
	}
	if !policy.RetryInvalidOrMissingArtifactsOnce {
		return false, acpruntime.Result{}, nil
	}

	initialStructuralFailure := isStructuralArtifactContractFailure(validationErr)
	emitArtifactRetryScheduledDiagnostic(task, adapter.Provider(), validationErr)
	retryResult, retryErr := runProviderCommand(ctx, task, adapter, normalizeActivityPolicy(adapter.ActivityPolicy(task)))
	if retryErr != nil {
		var retryStalled StallError
		if errors.As(retryErr, &retryStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitRetryCompletedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic.StallPhase, "fresh_process_artifact_only")
					return true, retryResult, nil
				} else if recovered, recoveredResult, recoveredErr := recoverFocusedArtifactRepair(ctx, task, adapter, retryResult, err, "retry"); recovered {
					return true, recoveredResult, recoveredErr
				}
			}
			emitRetryExhaustedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic, "fresh_process")
			if initialStructuralFailure {
				return true, acpruntime.Result{}, wrapArtifactContractFailure(adapter, task, "retry", retryResult, "fresh-process retry stalled after an earlier malformed artifact contract", validationErr)
			}
			if shouldClassifySilentRetryExhaustionUnavailable(policy, task, retryResult) {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "retry", retryResult, "provider unavailable after fresh-process artifact retry", retryErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "retry", "fresh-process retry stalled before producing valid artifacts", retryErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, retryResult, retryErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		emitArtifactRetryExhaustedDiagnostic(task, adapter.Provider(), err)
		if initialStructuralFailure {
			return true, acpruntime.Result{}, wrapArtifactContractFailure(adapter, task, "retry", retryResult, "artifact validation failed after an earlier malformed artifact contract", validationErr)
		}
		if recovered, recoveredResult, recoveredErr := recoverFocusedArtifactRepair(ctx, task, adapter, retryResult, err, "retry"); recovered {
			return true, recoveredResult, recoveredErr
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "retry", "artifact validation failed after fresh-process retry", err)
	}
	emitArtifactRetryCompletedDiagnostic(task, adapter.Provider())
	return true, retryResult, nil
}

func recoverAfterStall(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, runErr error) (bool, acpruntime.Result, error) {
	var stalled StallError
	if !errors.As(runErr, &stalled) {
		return false, acpruntime.Result{}, nil
	}

	policy := adapter.RecoveryPolicy(task)
	emitDiagnostic(task, "retry scheduled", stalled.Diagnostic.fields(adapter.Provider(), task, "terminate_and_validate"))
	zeroOutputPreArtifactStall := shouldClassifyZeroOutputPreArtifactStallUnavailable(policy, task, result, stalled.Diagnostic)
	canRetryZeroOutputPreArtifact := zeroOutputPreArtifactStall &&
		policy.RetryZeroOutputPreArtifactStallOnce &&
		policy.RetryInvalidOrMissingArtifactsOnce
	if policy.AcceptValidArtifactsAfterStop {
		if err := adapter.ValidateArtifacts(task); err == nil {
			emitRetryCompletedDiagnostic(task, adapter.Provider(), stalled.Diagnostic.StallPhase, "artifact_only")
			return true, result, nil
		} else if zeroOutputPreArtifactStall && !canRetryZeroOutputPreArtifact {
			emitZeroOutputPreArtifactStallDiagnostic(task, adapter.Provider(), stalled.Diagnostic, "pre_artifact_fail_fast")
			return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "stall", result, "provider unavailable after zero-output pre-artifact stall", runErr)
		} else if !zeroOutputPreArtifactStall {
			if recovered, recoveredResult, recoveredErr := recoverFocusedArtifactRepair(ctx, task, adapter, result, err, "stall"); recovered {
				return true, recoveredResult, recoveredErr
			}
		}
	}
	if zeroOutputPreArtifactStall {
		if !canRetryZeroOutputPreArtifact {
			emitZeroOutputPreArtifactStallDiagnostic(task, adapter.Provider(), stalled.Diagnostic, "pre_artifact_fail_fast")
			return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "stall", result, "provider unavailable after zero-output pre-artifact stall", runErr)
		}
		emitZeroOutputPreArtifactStallRetryDiagnostic(task, adapter.Provider(), stalled.Diagnostic)
	}
	if !policy.RetryInvalidOrMissingArtifactsOnce {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "stall", "runtime stalled before valid artifacts were available", runErr)
	}
	retryPolicy := normalizeActivityPolicy(adapter.ActivityPolicy(task))
	if stalled.Diagnostic.StallPhase == StallPhasePreArtifact && retryPolicy.RetryPreArtifactStallWindow > 0 {
		retryPolicy.PreArtifactStallWindow = retryPolicy.RetryPreArtifactStallWindow
	}
	emitStallRetryScheduledDiagnostic(task, adapter.Provider(), stalled.Diagnostic)
	retryResult, retryErr := runProviderCommand(ctx, task, adapter, retryPolicy)
	if retryErr != nil {
		var retryStalled StallError
		if errors.As(retryErr, &retryStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitRetryCompletedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic.StallPhase, "fresh_process_artifact_only")
					return true, retryResult, nil
				} else if recovered, recoveredResult, recoveredErr := recoverFocusedArtifactRepair(ctx, task, adapter, retryResult, err, "retry"); recovered {
					return true, recoveredResult, recoveredErr
				}
			}
			emitRetryExhaustedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic, "fresh_process")
			if shouldClassifySilentRetryExhaustionUnavailable(policy, task, retryResult) {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "retry", retryResult, "provider unavailable after fresh-process stall retry", retryErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "retry", "fresh-process retry stalled before producing valid artifacts", retryErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, retryResult, retryErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		if recovered, recoveredResult, recoveredErr := recoverFocusedArtifactRepair(ctx, task, adapter, retryResult, err, "retry"); recovered {
			return true, recoveredResult, recoveredErr
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "retry", "artifact validation failed after stall retry", err)
	}
	emitRetryCompletedDiagnostic(task, adapter.Provider(), stalled.Diagnostic.StallPhase, "fresh_process")
	return true, retryResult, nil
}

func recoverFocusedArtifactRepair(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string) (bool, acpruntime.Result, error) {
	if recovered, recoveredResult, recoveredErr := recoverCollectArtifactPairRepair(ctx, task, adapter, result, validationErr, stage); recovered {
		return true, recoveredResult, recoveredErr
	}
	if recovered, recoveredResult, recoveredErr := recoverCollectManifestRepair(ctx, task, adapter, result, validationErr, stage); recovered {
		return true, recoveredResult, recoveredErr
	}
	if recovered, recoveredResult, recoveredErr := recoverValidatorVerdictRepair(ctx, task, adapter, result, validationErr, stage); recovered {
		return true, recoveredResult, recoveredErr
	}
	if recovered, recoveredResult, recoveredErr := recoverDraftArtifactRepair(ctx, task, adapter, result, validationErr, stage); recovered {
		return true, recoveredResult, recoveredErr
	}
	return false, acpruntime.Result{}, nil
}

func recoverCollectArtifactPairRepair(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !policy.RepairCollectArtifactPairOnce || !acpruntime.IsCollectStep(task.StepID) {
		return false, acpruntime.Result{}, nil
	}
	snapshot := runtimeArtifactSnapshot(task)
	if snapshot.AuthoredFiles > 0 {
		return false, acpruntime.Result{}, nil
	}
	if !resultHasProviderDiagnostics(result) {
		return false, acpruntime.Result{}, nil
	}
	repairAdapter, ok := adapter.(CollectArtifactPairRepairAdapter)
	if !ok {
		return false, acpruntime.Result{}, nil
	}
	beforeRepairFiles, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "collect_pair_repair", "collect pair recovery write-set precheck failed", err)
	}

	emitFocusedArtifactRepairScheduledDiagnostic(task, adapter.Provider(), "collect_pair_repair", stage, snapshot, validationErr)
	buildRepairSpec := func() (CommandSpec, error) {
		return repairAdapter.CollectArtifactPairRepairCommandSpec(task, validationErr)
	}
	repairResult, repairErr, commandErr := runFocusedArtifactRepairCommand(ctx, task, adapter, result, buildRepairSpec)
	if commandErr != nil {
		return true, acpruntime.Result{}, commandErr
	}
	if writeSetErr := validateCollectArtifactPairRepairWriteSet(task, beforeRepairFiles); writeSetErr != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_pair_repair", "collect pair recovery wrote outside the collect pair write set", writeSetErr)
	}
	if repairErr != nil {
		var repairStalled StallError
		if errors.As(repairErr, &repairStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "collect_pair_repair", repairStalled.Diagnostic.StallPhase)
					return true, repairResult, nil
				}
			}
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", repairStalled.Diagnostic, repairErr)
			if shouldClassifySilentRetryExhaustionUnavailable(policy, task, repairResult) {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "collect_pair_repair", repairResult, "provider unavailable during collect pair recovery", repairErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_pair_repair", "collect pair recovery stalled before valid artifacts were available", repairErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, repairResult, repairErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		if shouldRetryTransientProviderUnavailableArtifactRepair(policy, repairResult, err, adapter.UnavailableMarkers()) {
			emitFocusedArtifactRepairRetryScheduledDiagnostic(task, adapter.Provider(), "collect_pair_repair", err)
			retryResult, retryErr, retryCommandErr := runFocusedArtifactRepairCommand(ctx, task, adapter, repairResult, buildRepairSpec)
			if retryCommandErr != nil {
				return true, acpruntime.Result{}, retryCommandErr
			}
			if writeSetErr := validateCollectArtifactPairRepairWriteSet(task, beforeRepairFiles); writeSetErr != nil {
				return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "collect_pair_repair", "collect pair recovery wrote outside the collect pair write set", writeSetErr)
			}
			if retryErr != nil {
				var retryStalled StallError
				if errors.As(retryErr, &retryStalled) {
					if policy.AcceptValidArtifactsAfterStop {
						if err := adapter.ValidateArtifacts(task); err == nil {
							emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "collect_pair_repair", retryStalled.Diagnostic.StallPhase)
							return true, retryResult, nil
						}
					}
					emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", retryStalled.Diagnostic, retryErr)
					if shouldClassifySilentRetryExhaustionUnavailable(policy, task, retryResult) {
						return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "collect_pair_repair", retryResult, "provider unavailable during collect pair recovery", retryErr)
					}
					return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "collect_pair_repair", "collect pair recovery stalled before valid artifacts were available", retryErr)
				}
				return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, retryResult, retryErr)
			}
			if err := adapter.ValidateArtifacts(task); err != nil {
				emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
				return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "collect_pair_repair", "collect pair recovery did not produce valid collect artifacts", err)
			}
			emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "collect_pair_repair", "")
			return true, retryResult, nil
		}
		emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_pair_repair", "collect pair recovery did not produce valid collect artifacts", err)
	}
	emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "collect_pair_repair", "")
	return true, repairResult, nil
}

func resultHasProviderDiagnostics(result acpruntime.Result) bool {
	return strings.TrimSpace(result.Stdout) != "" || strings.TrimSpace(result.Stderr) != ""
}

func recoverCollectManifestRepair(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !policy.RepairCollectManifestOnce || !acpruntime.IsCollectStep(task.StepID) {
		return false, acpruntime.Result{}, nil
	}
	snapshot := runtimeArtifactSnapshot(task)
	if snapshot.AuthoredFiles <= 0 {
		return false, acpruntime.Result{}, nil
	}
	if collectWriteRootHasBootstrapOnlyAuthoredDoc(task) {
		return false, acpruntime.Result{}, nil
	}
	beforeRepairFiles, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "collect_manifest_repair", "manifest-only collect repair write-set precheck failed", err)
	}
	if collectManifestFileMissing(task) || isCollectManifestSemanticScaffoldFailure(validationErr) {
		if recovered, recoveredResult, recoveredErr := recoverCollectManifestDeterministically(task, adapter, result, beforeRepairFiles, validationErr); recovered {
			return true, recoveredResult, recoveredErr
		}
	}
	repairAdapter, ok := adapter.(CollectManifestRepairAdapter)
	if !ok {
		return false, acpruntime.Result{}, nil
	}

	emitCollectManifestRepairScheduledDiagnostic(task, adapter.Provider(), snapshot, validationErr)
	spec, err := repairAdapter.CollectManifestRepairCommandSpec(task, validationErr)
	if err != nil {
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, result, err)
	}
	repairPolicy := normalizeActivityPolicy(adapter.ActivityPolicy(task))
	repairPolicy.MonitorArtifacts = true
	repairPolicy.MonitorPreArtifact = false
	if repairPolicy.PostArtifactStallWindow < defaultCollectRepairWindow {
		repairPolicy.PostArtifactStallWindow = defaultCollectRepairWindow
	}
	if repairPolicy.PartialArtifactStallWindow < defaultCollectRepairWindow {
		repairPolicy.PartialArtifactStallWindow = defaultCollectRepairWindow
	}
	if repairPolicy.ValidArtifactStopWindow <= 0 {
		repairPolicy.ValidArtifactStopWindow = defaultRepairValidStopWindow
	}
	repairResult, repairErr := runCommandSpec(ctx, task, spec, repairPolicy)
	if repairErr != nil {
		if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_manifest_repair", "manifest-only collect repair wrote outside shard-pack-manifest.json", err)
		}
		var repairStalled StallError
		if errors.As(repairErr, &repairStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitCollectManifestRepairCompletedDiagnostic(task, adapter.Provider(), repairStalled.Diagnostic.StallPhase)
					return true, repairResult, nil
				}
			}
			emitCollectManifestRepairExhaustedDiagnostic(task, adapter.Provider(), repairStalled.Diagnostic, repairErr)
			if recovered, recoveredResult, recoveredErr := recoverCollectManifestDeterministically(task, adapter, repairResult, beforeRepairFiles, repairErr); recovered {
				return true, recoveredResult, recoveredErr
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_manifest_repair", "manifest-only collect repair stalled before valid artifacts were available", repairErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, repairResult, repairErr)
	}
	if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_manifest_repair", "manifest-only collect repair wrote outside shard-pack-manifest.json", err)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		emitCollectManifestRepairExhaustedDiagnostic(task, adapter.Provider(), runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		if recovered, recoveredResult, recoveredErr := recoverCollectManifestDeterministically(task, adapter, repairResult, beforeRepairFiles, err); recovered {
			return true, recoveredResult, recoveredErr
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_manifest_repair", "manifest-only collect repair did not produce valid collect artifacts", err)
	}
	emitCollectManifestRepairCompletedDiagnostic(task, adapter.Provider(), "")
	return true, repairResult, nil
}

func recoverCollectManifestDeterministically(task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, beforeRepairFiles writeRootFileSnapshot, cause error) (bool, acpruntime.Result, error) {
	report, err := recoverCollectManifestFromAuthoredDocs(task, cause)
	if err != nil {
		emitCollectManifestDeterministicRecoveryFailedDiagnostic(task, adapter.Provider(), err)
		return false, acpruntime.Result{}, nil
	}
	if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "collect_manifest_runtime_recovery", "deterministic collect manifest recovery wrote outside shard-pack-manifest.json", err)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		emitCollectManifestDeterministicRecoveryFailedDiagnostic(task, adapter.Provider(), err)
		return false, acpruntime.Result{}, nil
	}
	emitCollectManifestDeterministicRecoveryCompletedDiagnostic(task, adapter.Provider(), report)
	return true, result, nil
}

func collectManifestFileMissing(task acpruntime.Task) bool {
	root := strings.TrimSpace(task.WriteRoot)
	if root == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(filepath.Clean(root), ShardPackManifestFileName))
	return errors.Is(err, os.ErrNotExist)
}

func isCollectManifestSemanticScaffoldFailure(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "semantic snapshot is bootstrap-only collect scaffold")
}

func collectWriteRootHasBootstrapOnlyAuthoredDoc(task acpruntime.Task) bool {
	root := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	if root == "" || root == "." {
		return false
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry == nil || entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || name == ShardPackManifestFileName || name == "runtime-execution.json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		if artifactquality.CollectDocumentBootstrapOnly(string(raw)) {
			return true
		}
	}
	return false
}

func recoverValidatorVerdictRepair(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !policy.RepairValidatorVerdictOnce || acpruntime.StepProviderKeyForStepID(task.StepID) != acpruntime.StepProviderStep3Findings {
		return false, acpruntime.Result{}, nil
	}
	repairAdapter, ok := adapter.(ValidatorVerdictRepairAdapter)
	if !ok {
		return false, acpruntime.Result{}, nil
	}
	beforeRepairFiles, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "validator_verdict_repair", "validator verdict recovery write-set precheck failed", err)
	}

	emitFocusedArtifactRepairScheduledDiagnostic(task, adapter.Provider(), "validator_verdict_repair", stage, runtimeArtifactSnapshot(task), validationErr)
	repairResult, repairErr, commandErr := runFocusedArtifactRepairCommand(ctx, task, adapter, result, func() (CommandSpec, error) {
		return repairAdapter.ValidatorVerdictRepairCommandSpec(task, validationErr)
	})
	if commandErr != nil {
		return true, acpruntime.Result{}, commandErr
	}
	if writeSetErr := validateValidatorVerdictRepairWriteSet(task, beforeRepairFiles); writeSetErr != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "validator_verdict_repair", "verdict-only validator repair wrote outside validator-verdict.json", writeSetErr)
	}
	if repairErr != nil {
		var repairStalled StallError
		if errors.As(repairErr, &repairStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "validator_verdict_repair", repairStalled.Diagnostic.StallPhase)
					return true, repairResult, nil
				}
			}
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "validator_verdict_repair", repairStalled.Diagnostic, repairErr)
			if shouldClassifySilentRetryExhaustionUnavailable(policy, task, repairResult) {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "validator_verdict_repair", repairResult, "provider unavailable during verdict-only validator repair", repairErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "validator_verdict_repair", "verdict-only validator repair stalled before valid artifacts were available", repairErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, repairResult, repairErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "validator_verdict_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "validator_verdict_repair", "verdict-only validator repair did not produce a valid validator verdict contract", err)
	}
	emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "validator_verdict_repair", "")
	return true, repairResult, nil
}

func recoverDraftArtifactRepair(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !policy.RepairDraftArtifactsOnce || !runtimedrafts.IsDraftStep(task.StepID) {
		return false, acpruntime.Result{}, nil
	}
	repairAdapter, ok := adapter.(DraftArtifactRepairAdapter)
	if !ok {
		return false, acpruntime.Result{}, nil
	}
	beforeWriteRoot, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_repair", "draft recovery write_root precheck failed", err)
	}
	beforeDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_repair", "draft recovery draft_final_root precheck failed", err)
	}

	emitFocusedArtifactRepairScheduledDiagnostic(task, adapter.Provider(), "draft_artifact_repair", stage, runtimeArtifactSnapshot(task), validationErr)
	buildRepairSpec := func() (CommandSpec, error) {
		return repairAdapter.DraftArtifactRepairCommandSpec(task, validationErr)
	}
	repairResult, repairErr, commandErr := runFocusedArtifactRepairCommand(ctx, task, adapter, result, buildRepairSpec)
	if commandErr != nil {
		return true, acpruntime.Result{}, commandErr
	}
	if writeSetErr := validateDraftArtifactRepairWriteSet(task, beforeWriteRoot, beforeDraftRoot); writeSetErr != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "draft_artifact_repair", "draft recovery wrote outside the draft artifact write set", writeSetErr)
	}
	if repairErr != nil {
		var repairStalled StallError
		if errors.As(repairErr, &repairStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "completed_after_controlled_stop", runtimeArtifactSnapshot(task))
					emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", repairStalled.Diagnostic.StallPhase)
					return true, repairResult, nil
				} else if isDraftBootstrapOnlyValidationFailure(err) {
					emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "stalled", runtimeArtifactSnapshot(task))
					emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", repairStalled.Diagnostic, repairErr)
					return recoverDraftArtifactEnrichment(ctx, task, adapter, repairResult, err, "draft_artifact_repair_stalled")
				}
			}
			emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "stalled", runtimeArtifactSnapshot(task))
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", repairStalled.Diagnostic, repairErr)
			if shouldClassifySilentRetryExhaustionUnavailable(policy, task, repairResult) {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "draft_artifact_repair", repairResult, "provider unavailable during draft artifact repair", repairErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "draft_artifact_repair", "draft artifact repair stalled before valid artifacts were available", repairErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, repairResult, repairErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		if shouldRetryTransientProviderUnavailableArtifactRepair(policy, repairResult, err, adapter.UnavailableMarkers()) {
			emitFocusedArtifactRepairRetryScheduledDiagnostic(task, adapter.Provider(), "draft_artifact_repair", err)
			retryResult, retryErr, retryCommandErr := runFocusedArtifactRepairCommand(ctx, task, adapter, repairResult, buildRepairSpec)
			if retryCommandErr != nil {
				return true, acpruntime.Result{}, retryCommandErr
			}
			if writeSetErr := validateDraftArtifactRepairWriteSet(task, beforeWriteRoot, beforeDraftRoot); writeSetErr != nil {
				return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "draft_artifact_repair", "draft recovery wrote outside the draft artifact write set", writeSetErr)
			}
			if retryErr != nil {
				var retryStalled StallError
				if errors.As(retryErr, &retryStalled) {
					if policy.AcceptValidArtifactsAfterStop {
						if err := adapter.ValidateArtifacts(task); err == nil {
							emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "completed_after_controlled_stop", runtimeArtifactSnapshot(task))
							emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", retryStalled.Diagnostic.StallPhase)
							return true, retryResult, nil
						} else if isDraftBootstrapOnlyValidationFailure(err) {
							emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "stalled", runtimeArtifactSnapshot(task))
							emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", retryStalled.Diagnostic, retryErr)
							return recoverDraftArtifactEnrichment(ctx, task, adapter, retryResult, err, "draft_artifact_repair_retry_stalled")
						}
					}
					emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "stalled", runtimeArtifactSnapshot(task))
					emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", retryStalled.Diagnostic, retryErr)
					if shouldClassifySilentRetryExhaustionUnavailable(policy, task, retryResult) {
						return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "draft_artifact_repair", retryResult, "provider unavailable during draft artifact repair", retryErr)
					}
					return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "draft_artifact_repair", "draft artifact repair stalled before valid artifacts were available", retryErr)
				}
				return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, retryResult, retryErr)
			}
			if err := adapter.ValidateArtifacts(task); err != nil {
				emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "invalid", runtimeArtifactSnapshot(task))
				emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
				if isDraftBootstrapOnlyValidationFailure(err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, retryResult, err, "draft_artifact_repair_retry_invalid")
				}
				return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "draft_artifact_repair", "draft artifact repair did not produce valid draft artifact contract", err)
			}
			emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "completed", runtimeArtifactSnapshot(task))
			emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", "")
			return true, retryResult, nil
		}
		emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "invalid", runtimeArtifactSnapshot(task))
		emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		if isDraftBootstrapOnlyValidationFailure(err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, repairResult, err, "draft_artifact_repair_invalid")
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "draft_artifact_repair", "draft artifact repair did not produce valid draft artifact contract", err)
	}
	emitDraftArtifactRepairSnapshotDiagnostic(task, adapter.Provider(), "completed", runtimeArtifactSnapshot(task))
	emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_repair", "")
	return true, repairResult, nil
}

func recoverDraftArtifactEnrichment(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !policy.RepairDraftArtifactEnrichmentOnce || !runtimedrafts.IsDraftStep(task.StepID) {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_repair", "draft artifact repair did not produce valid draft artifact contract", validationErr)
	}
	enrichmentAdapter, ok := adapter.(DraftArtifactEnrichmentAdapter)
	if !ok {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_repair", "draft artifact repair did not produce valid draft artifact contract", validationErr)
	}
	beforeWriteRoot, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_enrichment", "draft enrichment write_root precheck failed", err)
	}
	beforeDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_enrichment", "draft enrichment draft_final_root precheck failed", err)
	}

	emitFocusedArtifactRepairScheduledDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", stage, runtimeArtifactSnapshot(task), validationErr)
	enrichmentResult, enrichmentErr, commandErr := runFocusedArtifactRepairCommand(ctx, task, adapter, result, func() (CommandSpec, error) {
		return enrichmentAdapter.DraftArtifactEnrichmentCommandSpec(task, validationErr)
	})
	if commandErr != nil {
		return true, acpruntime.Result{}, commandErr
	}
	if writeSetErr := validateDraftArtifactRepairWriteSet(task, beforeWriteRoot, beforeDraftRoot); writeSetErr != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft enrichment wrote outside the draft artifact write set", writeSetErr)
	}
	if enrichmentErr != nil {
		var enrichmentStalled StallError
		if errors.As(enrichmentErr, &enrichmentStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "completed_after_controlled_stop", runtimeArtifactSnapshot(task))
					emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", enrichmentStalled.Diagnostic.StallPhase)
					return true, enrichmentResult, nil
				}
			}
			emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "stalled", runtimeArtifactSnapshot(task))
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", enrichmentStalled.Diagnostic, enrichmentErr)
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft artifact enrichment stalled before valid artifacts were available", enrichmentErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, enrichmentResult, enrichmentErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "invalid", runtimeArtifactSnapshot(task))
		emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft artifact enrichment did not produce valid draft artifact contract", err)
	}
	emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "completed", runtimeArtifactSnapshot(task))
	emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", "")
	return true, enrichmentResult, nil
}

func isDraftBootstrapOnlyValidationFailure(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "bootstrap-only placeholder draft content")
}

func runFocusedArtifactRepairCommand(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, baseResult acpruntime.Result, buildSpec func() (CommandSpec, error)) (acpruntime.Result, error, error) {
	spec, err := buildSpec()
	if err != nil {
		return acpruntime.Result{}, nil, classifyCommandFailure(adapter, task, baseResult, err)
	}
	repairPolicy := focusedRepairActivityPolicy(adapter.ActivityPolicy(task), true)
	repairResult, repairErr := runCommandSpec(ctx, task, spec, repairPolicy)
	return repairResult, repairErr, nil
}

func focusedRepairActivityPolicy(base ActivityPolicy, monitorPreArtifact bool) ActivityPolicy {
	repairPolicy := normalizeActivityPolicy(base)
	repairPolicy.MonitorArtifacts = true
	repairPolicy.MonitorPreArtifact = monitorPreArtifact
	if repairPolicy.PreArtifactStallWindow < defaultFocusedRepairWindow {
		repairPolicy.PreArtifactStallWindow = defaultFocusedRepairWindow
	}
	if repairPolicy.PostArtifactStallWindow < defaultFocusedRepairWindow {
		repairPolicy.PostArtifactStallWindow = defaultFocusedRepairWindow
	}
	if repairPolicy.PartialArtifactStallWindow < defaultFocusedRepairWindow {
		repairPolicy.PartialArtifactStallWindow = defaultFocusedRepairWindow
	}
	if repairPolicy.ValidArtifactStopWindow <= 0 {
		repairPolicy.ValidArtifactStopWindow = defaultRepairValidStopWindow
	}
	return repairPolicy
}
