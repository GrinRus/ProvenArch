package providercommon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
		retryPolicy.PreArtifactWallClockWindow = retryPolicy.RetryPreArtifactStallWindow
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
			if recovered, recoveredResult, recoveredErr := recoverCollectArtifactPairRepairAfterSilentRetryExhaustion(ctx, task, adapter, retryResult, retryErr); recovered {
				return true, recoveredResult, recoveredErr
			}
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
	return recoverCollectArtifactPairRepairWithOptions(ctx, task, adapter, result, validationErr, stage, collectArtifactPairRepairOptions{
		allowManifestFallback: true,
	})
}

func recoverCollectArtifactPairRepairAfterSilentRetryExhaustion(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !policy.RepairCollectArtifactPairOnce || !acpruntime.IsCollectStep(task.StepID) {
		return false, acpruntime.Result{}, nil
	}
	if !shouldClassifySilentRetryExhaustionUnavailable(policy, task, result) {
		return false, acpruntime.Result{}, nil
	}
	return recoverCollectArtifactPairRepairWithOptions(ctx, task, adapter, result, validationErr, "retry_silent_exhausted", collectArtifactPairRepairOptions{
		allowSilentNoArtifact: true,
		allowManifestFallback: true,
	})
}

type collectArtifactPairRepairOptions struct {
	allowSilentNoArtifact                    bool
	allowNoProviderDiagnostics               bool
	allowExistingAuthoredFiles               bool
	requiredExistingAuthoredDocMutationPaths map[string]struct{}
	allowManifestFallback                    bool
}

func recoverCollectArtifactPairRepairWithOptions(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string, options collectArtifactPairRepairOptions) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !policy.RepairCollectArtifactPairOnce || !acpruntime.IsCollectStep(task.StepID) {
		return false, acpruntime.Result{}, nil
	}
	snapshot := runtimeArtifactSnapshot(task)
	if snapshot.AuthoredFiles > 0 && !options.allowExistingAuthoredFiles {
		return false, acpruntime.Result{}, nil
	}
	if !options.allowSilentNoArtifact && !options.allowNoProviderDiagnostics && !resultHasProviderDiagnostics(result) {
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
					if outcomeErr := validateCollectArtifactPairRepairOutcome(task, beforeRepairFiles, options); outcomeErr != nil {
						emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", repairStalled.Diagnostic, outcomeErr)
						return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_pair_repair", "collect pair recovery did not repair referenced markdown", outcomeErr)
					}
					emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "collect_pair_repair", repairStalled.Diagnostic.StallPhase)
					return true, repairResult, nil
				}
			}
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", repairStalled.Diagnostic, repairErr)
			if options.allowManifestFallback {
				if recovered, recoveredResult, recoveredErr := recoverCollectManifestRepair(ctx, task, adapter, repairResult, repairErr, "collect_pair_repair_partial"); recovered {
					return true, recoveredResult, recoveredErr
				}
			}
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
							if outcomeErr := validateCollectArtifactPairRepairOutcome(task, beforeRepairFiles, options); outcomeErr != nil {
								emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", retryStalled.Diagnostic, outcomeErr)
								return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "collect_pair_repair", "collect pair recovery did not repair referenced markdown", outcomeErr)
							}
							emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "collect_pair_repair", retryStalled.Diagnostic.StallPhase)
							return true, retryResult, nil
						}
					}
					emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", retryStalled.Diagnostic, retryErr)
					if options.allowManifestFallback {
						if recovered, recoveredResult, recoveredErr := recoverCollectManifestRepair(ctx, task, adapter, retryResult, retryErr, "collect_pair_repair_retry_partial"); recovered {
							return true, recoveredResult, recoveredErr
						}
					}
					if shouldClassifySilentRetryExhaustionUnavailable(policy, task, retryResult) {
						return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "collect_pair_repair", retryResult, "provider unavailable during collect pair recovery", retryErr)
					}
					return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "collect_pair_repair", "collect pair recovery stalled before valid artifacts were available", retryErr)
				}
				return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, retryResult, retryErr)
			}
			if err := adapter.ValidateArtifacts(task); err != nil {
				emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
				if options.allowManifestFallback {
					if recovered, recoveredResult, recoveredErr := recoverCollectManifestRepair(ctx, task, adapter, retryResult, err, "collect_pair_repair_retry_invalid"); recovered {
						return true, recoveredResult, recoveredErr
					}
				}
				return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "collect_pair_repair", "collect pair recovery did not produce valid collect artifacts", err)
			}
			if outcomeErr := validateCollectArtifactPairRepairOutcome(task, beforeRepairFiles, options); outcomeErr != nil {
				emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), outcomeErr)
				return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "collect_pair_repair", "collect pair recovery did not repair referenced markdown", outcomeErr)
			}
			emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "collect_pair_repair", "")
			return true, retryResult, nil
		}
		emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		if options.allowManifestFallback {
			if recovered, recoveredResult, recoveredErr := recoverCollectManifestRepair(ctx, task, adapter, repairResult, err, "collect_pair_repair_invalid"); recovered {
				return true, recoveredResult, recoveredErr
			}
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_pair_repair", "collect pair recovery did not produce valid collect artifacts", err)
	}
	if outcomeErr := validateCollectArtifactPairRepairOutcome(task, beforeRepairFiles, options); outcomeErr != nil {
		emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), outcomeErr)
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_pair_repair", "collect pair recovery did not repair referenced markdown", outcomeErr)
	}
	emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "collect_pair_repair", "")
	return true, repairResult, nil
}

func validateCollectArtifactPairRepairOutcome(task acpruntime.Task, before writeRootFileSnapshot, options collectArtifactPairRepairOptions) error {
	if len(options.requiredExistingAuthoredDocMutationPaths) == 0 {
		return nil
	}
	after, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return err
	}
	changedRequired := map[string]struct{}{}
	for path, beforeState := range before {
		if beforeState.IsDir || path == ShardPackManifestFileName || path == "runtime-execution.json" {
			continue
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			continue
		}
		if _, required := options.requiredExistingAuthoredDocMutationPaths[path]; !required {
			continue
		}
		afterState, exists := after[path]
		if exists && !afterState.IsDir && beforeState != afterState {
			changedRequired[path] = struct{}{}
		}
	}
	missing := []string{}
	for path := range options.requiredExistingAuthoredDocMutationPaths {
		if _, ok := changedRequired[path]; !ok {
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("collect_pair_repair_noop_or_stale_markdown: existing authored markdown was not rewritten: %s", strings.Join(missing, ", "))
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
	beforeRepairFiles, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "collect_manifest_repair", "manifest-only collect repair write-set precheck failed", err)
	}
	if processDocs := collectProcessContaminatedAuthoredDocs(task); len(processDocs) > 0 {
		if recovered, recoveredResult, recoveredErr := recoverCollectArtifactPairRepairWithOptions(ctx, task, adapter, result, validationErr, stage+"_process_contaminated_markdown", collectArtifactPairRepairOptions{
			allowNoProviderDiagnostics:               true,
			allowExistingAuthoredFiles:               true,
			requiredExistingAuthoredDocMutationPaths: stringSliceSet(processDocs),
			allowManifestFallback:                    false,
		}); recovered {
			return true, recoveredResult, recoveredErr
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "collect_pair_repair", "collect pair recovery is required for process-contaminated authored markdown", validationErr)
	}
	if collectWriteRootHasBootstrapOnlyAuthoredDoc(task) {
		return false, acpruntime.Result{}, nil
	}
	if staleDocs := collectAuthoredDocsMentioningMissingRepoEvidencePaths(task, validationErr); len(staleDocs) > 0 {
		if recovered, recoveredResult, recoveredErr := recoverCollectArtifactPairRepairWithOptions(ctx, task, adapter, result, validationErr, stage+"_repo_evidence_mismatch", collectArtifactPairRepairOptions{
			allowNoProviderDiagnostics:               true,
			allowExistingAuthoredFiles:               true,
			requiredExistingAuthoredDocMutationPaths: stringSliceSet(staleDocs),
			allowManifestFallback:                    false,
		}); recovered {
			return true, recoveredResult, recoveredErr
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "collect_pair_repair", "collect pair recovery is required for authored markdown that cites missing repo evidence", validationErr)
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
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_manifest_repair", "manifest-only collect repair stalled before valid artifacts were available", repairErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, repairResult, repairErr)
	}
	if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_manifest_repair", "manifest-only collect repair wrote outside shard-pack-manifest.json", err)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		emitCollectManifestRepairExhaustedDiagnostic(task, adapter.Provider(), runtimeArtifactSnapshot(task).stallDiagnostic(), err)
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
	result = markCollectManifestRuntimeRecovered(result, report)
	return true, result, nil
}

func markCollectManifestRuntimeRecovered(result acpruntime.Result, report collectManifestRuntimeRecoveryReport) acpruntime.Result {
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	result.Diagnostics["collect_manifest_runtime_recovery"] = map[string]any{
		"recovery_mode":       "collect_manifest_runtime_recovery",
		"source":              "runtime_recovery",
		"provider_authored":   false,
		"document_count":      report.DocumentCount,
		"entity_count":        report.EntityCount,
		"edge_count":          report.EdgeCount,
		"evidence_path":       strings.TrimSpace(report.EvidencePath),
		"manual_quality_gate": "artifact_quality_assessment",
	}
	warning := "runtime_recovery: collect_manifest_runtime_recovery reconstructed shard-pack-manifest.json from provider-authored markdown; treat as recovery evidence, not normal provider-authored manifest success"
	if !containsRuntimeWarning(result.Execution.Warnings, warning) {
		result.Execution.Warnings = append(result.Execution.Warnings, warning)
	}
	return result
}

func containsRuntimeWarning(values []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
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

func collectProcessContaminatedAuthoredDocs(task acpruntime.Task) []string {
	root := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	if root == "" || root == "." {
		return nil
	}
	found := []string{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
			return nil
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || name == ShardPackManifestFileName || name == "runtime-execution.json" {
			return nil
		}
		if strings.ToLower(filepath.Ext(name)) != ".md" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil || strings.TrimSpace(rel) == "" || rel == "." {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if artifactquality.CollectDocumentRuntimeProcessContaminated(string(raw)) {
			found = append(found, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(found)
	return found
}

func collectAuthoredDocsMentioningMissingRepoEvidencePaths(task acpruntime.Task, err error) []string {
	missingPaths := missingRepoEvidencePathsFromError(err)
	if len(missingPaths) == 0 {
		return nil
	}
	root := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	if root == "" || root == "." {
		return nil
	}
	missing := map[string]struct{}{}
	for _, path := range missingPaths {
		path = strings.TrimSpace(path)
		if path != "" {
			missing[path] = struct{}{}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	found := []string{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
			return nil
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || name == ShardPackManifestFileName || name == "runtime-execution.json" {
			return nil
		}
		if strings.ToLower(filepath.Ext(name)) != ".md" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil || strings.TrimSpace(rel) == "" || rel == "." {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(raw)
		for missingPath := range missing {
			if strings.Contains(text, missingPath) {
				found = append(found, filepath.ToSlash(rel))
				return nil
			}
		}
		return nil
	})
	sort.Strings(found)
	return found
}

func stringSliceSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

func missingRepoEvidencePathsFromError(err error) []string {
	if err == nil {
		return nil
	}
	const marker = `repo evidence path "`
	text := err.Error()
	paths := []string{}
	seen := map[string]struct{}{}
	for {
		idx := strings.Index(text, marker)
		if idx < 0 {
			break
		}
		text = text[idx+len(marker):]
		end := strings.Index(text, `"`)
		if end < 0 {
			break
		}
		path := strings.TrimSpace(text[:end])
		if path != "" {
			if _, ok := seen[path]; !ok {
				seen[path] = struct{}{}
				paths = append(paths, path)
			}
		}
		text = text[end+1:]
	}
	return paths
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
	if isDraftBootstrapOnlyValidationFailure(validationErr) {
		return recoverDraftArtifactEnrichment(ctx, task, adapter, result, validationErr, stage+"_bootstrap_only")
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
	enrichmentResult, enrichmentErr, commandErr := runFocusedArtifactRepairCommandWithPolicy(
		ctx,
		task,
		adapter,
		result,
		func() (CommandSpec, error) {
			return enrichmentAdapter.DraftArtifactEnrichmentCommandSpec(task, validationErr)
		},
		func(policy ActivityPolicy) ActivityPolicy {
			return draftArtifactEnrichmentActivityPolicy(task, policy)
		},
	)
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
				if err := validateDraftArtifactEnrichmentOutcome(task, beforeDraftRoot, adapter.ValidateArtifacts(task)); err == nil {
					emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "completed_after_controlled_stop", runtimeArtifactSnapshot(task))
					emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", enrichmentStalled.Diagnostic.StallPhase)
					return true, enrichmentResult, nil
				} else if shouldRetryDraftManifestShapeEnrichment(stage, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_manifest_shape")
				} else if isDraftEnrichmentNoopOrScaffoldFailure(err) {
					if shouldRetryDraftMissingPythonEnrichment(stage, enrichmentResult, err) {
						return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_python3_retry")
					}
					if shouldRetryDraftPrintedCommandEnrichment(stage, enrichmentResult, err) {
						return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftCommandTextRetryError(err), "draft_artifact_enrichment_command_text_retry")
					}
					if shouldRetryDraftNoActionEnrichment(stage, err) {
						return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftNoActionRetryError(err), "draft_artifact_enrichment_no_action_retry")
					}
					emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "stalled", runtimeArtifactSnapshot(task))
					emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", enrichmentStalled.Diagnostic, err)
					return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft_artifact_enrichment_noop_or_scaffold", err)
				} else if shouldRetryDraftMalformedMarkdownEnrichment(stage, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_markdown_syntax")
				}
			}
			emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "stalled", runtimeArtifactSnapshot(task))
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", enrichmentStalled.Diagnostic, enrichmentErr)
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft artifact enrichment stalled before valid artifacts were available", enrichmentErr)
		}
		if shouldRetryDraftMissingPythonEnrichment(stage, enrichmentResult, enrichmentErr) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, enrichmentErr, "draft_artifact_enrichment_python3_retry")
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, enrichmentResult, enrichmentErr)
	}
	if err := validateDraftArtifactEnrichmentOutcome(task, beforeDraftRoot, adapter.ValidateArtifacts(task)); err != nil {
		emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "invalid", runtimeArtifactSnapshot(task))
		if shouldRetryDraftManifestShapeEnrichment(stage, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_manifest_shape")
		}
		if isDraftEnrichmentNoopOrScaffoldFailure(err) {
			if shouldRetryDraftMissingPythonEnrichment(stage, enrichmentResult, err) {
				return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_python3_retry")
			}
			if shouldRetryDraftPrintedCommandEnrichment(stage, enrichmentResult, err) {
				return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftCommandTextRetryError(err), "draft_artifact_enrichment_command_text_retry")
			}
			if shouldRetryDraftNoActionEnrichment(stage, err) {
				return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftNoActionRetryError(err), "draft_artifact_enrichment_no_action_retry")
			}
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft_artifact_enrichment_noop_or_scaffold", err)
		}
		if shouldRetryDraftMalformedMarkdownEnrichment(stage, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_markdown_syntax")
		}
		emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft artifact enrichment did not produce valid draft artifact contract", err)
	}
	emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "completed", runtimeArtifactSnapshot(task))
	emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", "")
	return true, enrichmentResult, nil
}

const minDraftArtifactEnrichmentPreArtifactWindow = 3 * time.Minute

func draftArtifactEnrichmentActivityPolicy(task acpruntime.Task, policy ActivityPolicy) ActivityPolicy {
	policy.FreshArtifactMutationAfter = time.Now().UTC().Add(-time.Millisecond)
	if policy.PreArtifactStallWindow < minDraftArtifactEnrichmentPreArtifactWindow {
		policy.PreArtifactStallWindow = minDraftArtifactEnrichmentPreArtifactWindow
	}
	if policy.PreArtifactWallClockWindow < minDraftArtifactEnrichmentPreArtifactWindow {
		policy.PreArtifactWallClockWindow = minDraftArtifactEnrichmentPreArtifactWindow
	}
	return policy
}

func validateDraftArtifactEnrichmentOutcome(task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot, validationErr error) error {
	if validationErr != nil {
		if isDraftBootstrapOnlyValidationFailure(validationErr) {
			return fmt.Errorf("draft_artifact_enrichment_noop_or_scaffold: %w", validationErr)
		}
		return validationErr
	}
	if !allDraftMarkdownOutputsChanged(task, beforeDraftRoot) {
		return fmt.Errorf("draft_artifact_enrichment_noop_or_scaffold: not all referenced markdown draft files changed")
	}
	return nil
}

func allDraftMarkdownOutputsChanged(task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot) bool {
	afterDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		return false
	}
	seenMarkdown := false
	for _, output := range loadAllowedDraftOutputs(task) {
		rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		if rel == "" || rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
			continue
		}
		if strings.ToLower(filepath.Ext(rel)) != ".md" {
			continue
		}
		seenMarkdown = true
		beforeState, beforeExists := beforeDraftRoot[rel]
		afterState, afterExists := afterDraftRoot[rel]
		if beforeExists == afterExists && (!afterExists || beforeState == afterState) {
			return false
		}
	}
	return seenMarkdown
}

func isDraftEnrichmentNoopOrScaffoldFailure(err error) bool {
	return err != nil && strings.Contains(err.Error(), "draft_artifact_enrichment_noop_or_scaffold")
}

func shouldRetryDraftMalformedMarkdownEnrichment(stage string, err error) bool {
	return strings.TrimSpace(stage) != "draft_artifact_enrichment_markdown_syntax" &&
		err != nil &&
		strings.Contains(err.Error(), "malformed markdown inline-code")
}

func shouldRetryDraftManifestShapeEnrichment(stage string, err error) bool {
	if strings.TrimSpace(stage) == "draft_artifact_enrichment_manifest_shape" || err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "parse runtime draft manifest: json: unknown field") ||
		(strings.Contains(text, "runtime draft manifest outputs are invalid:") && strings.Contains(text, "unknown field"))
}

func shouldRetryDraftNoActionEnrichment(stage string, err error) bool {
	trimmedStage := strings.TrimSpace(stage)
	return trimmedStage != "draft_artifact_enrichment_no_action_retry" &&
		trimmedStage != "draft_artifact_enrichment_command_text_retry" &&
		isDraftEnrichmentNoopOrScaffoldFailure(err)
}

func draftNoActionRetryError(err error) error {
	if err == nil {
		return errors.New("draft_artifact_enrichment_no_action_retry")
	}
	return fmt.Errorf("draft_artifact_enrichment_no_action_retry: %w", err)
}

func shouldRetryDraftPrintedCommandEnrichment(stage string, result acpruntime.Result, err error) bool {
	if strings.TrimSpace(stage) != "draft_artifact_enrichment_no_action_retry" || !isDraftEnrichmentNoopOrScaffoldFailure(err) {
		return false
	}
	text := strings.ToLower(strings.Join([]string{result.Stdout, result.Stderr, errorText(err)}, "\n"))
	markers := []string{
		"python3 - <<",
		"python3 <<",
		"python3 -c ",
		"```bash",
		"```sh",
		"```python",
		"here-doc",
		"heredoc",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func draftCommandTextRetryError(err error) error {
	if err == nil {
		return errors.New("draft_artifact_enrichment_command_text_retry")
	}
	return fmt.Errorf("draft_artifact_enrichment_command_text_retry: %w", err)
}

func shouldRetryDraftMissingPythonEnrichment(stage string, result acpruntime.Result, err error) bool {
	if strings.TrimSpace(stage) == "draft_artifact_enrichment_python3_retry" {
		return false
	}
	text := strings.ToLower(strings.Join([]string{result.Stdout, result.Stderr, errorText(err)}, "\n"))
	return strings.Contains(text, "command not found: python") ||
		strings.Contains(text, "python: command not found") ||
		strings.Contains(text, "python: not found")
}

func isDraftBootstrapOnlyValidationFailure(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "bootstrap-only placeholder draft content")
}

func runFocusedArtifactRepairCommand(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, baseResult acpruntime.Result, buildSpec func() (CommandSpec, error)) (acpruntime.Result, error, error) {
	return runFocusedArtifactRepairCommandWithPolicy(ctx, task, adapter, baseResult, buildSpec, nil)
}

func runFocusedArtifactRepairCommandWithPolicy(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, baseResult acpruntime.Result, buildSpec func() (CommandSpec, error), configure func(ActivityPolicy) ActivityPolicy) (acpruntime.Result, error, error) {
	spec, err := buildSpec()
	if err != nil {
		return acpruntime.Result{}, nil, classifyCommandFailure(adapter, task, baseResult, err)
	}
	repairPolicy := focusedRepairActivityPolicy(adapter.ActivityPolicy(task), true)
	if configure != nil {
		repairPolicy = configure(repairPolicy)
	}
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
	repairPolicy.PreArtifactWallClockWindow = defaultFocusedRepairWindow
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
