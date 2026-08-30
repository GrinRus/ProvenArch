package providercommon

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
	retryResult, retryErr := runProviderCommandWithTransition(ctx, task, adapter, normalizeActivityPolicy(adapter.ActivityPolicy(task)), "transport_retry")
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
	var validationErr error
	bootstrapDraftStall := false
	if policy.AcceptValidArtifactsAfterStop {
		if err := adapter.ValidateArtifacts(task); err == nil {
			emitRetryCompletedDiagnostic(task, adapter.Provider(), stalled.Diagnostic.StallPhase, "artifact_only")
			return true, result, nil
		} else {
			validationErr = err
			bootstrapDraftStall = shouldRetryDraftBootstrapAfterInitialStall(task, stalled.Diagnostic, err)
		}
		if zeroOutputPreArtifactStall && !canRetryZeroOutputPreArtifact {
			emitZeroOutputPreArtifactStallDiagnostic(task, adapter.Provider(), stalled.Diagnostic, "pre_artifact_fail_fast")
			return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "stall", result, "provider unavailable after zero-output pre-artifact stall", runErr)
		} else if !zeroOutputPreArtifactStall && !bootstrapDraftStall {
			if recovered, recoveredResult, recoveredErr := recoverFocusedArtifactRepair(ctx, task, adapter, result, validationErr, "stall"); recovered {
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
	if bootstrapDraftStall {
		// The initial provider invocation left only runtime bootstrap files in
		// place. Treat them as stale for the fresh process so the retry must
		// author the draft rather than immediately re-observing the scaffold.
		retryPolicy.FreshArtifactMutationAfter = time.Now().UTC()
	}
	if stalled.Diagnostic.StallPhase == StallPhasePreArtifact && retryPolicy.RetryPreArtifactStallWindow > 0 {
		retryPolicy.PreArtifactStallWindow = retryPolicy.RetryPreArtifactStallWindow
		retryPolicy.PreArtifactWallClockWindow = retryPolicy.RetryPreArtifactStallWindow
	}
	emitStallRetryScheduledDiagnostic(task, adapter.Provider(), stalled.Diagnostic)
	retryResult, retryErr := runProviderCommandWithTransition(ctx, task, adapter, retryPolicy, "transport_retry")
	if retryErr != nil {
		var retryStalled StallError
		if errors.As(retryErr, &retryStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitRetryCompletedDiagnostic(task, adapter.Provider(), retryStalled.Diagnostic.StallPhase, "fresh_process_artifact_only")
					return true, retryResult, nil
				} else if recovered, recoveredResult, recoveredErr := recoverDraftAfterSilentPreArtifactRetry(ctx, task, adapter, retryResult, retryStalled.Diagnostic); recovered {
					return true, recoveredResult, recoveredErr
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

// recoverDraftAfterSilentPreArtifactRetry gives a draft one final full
// provider invocation when both the initial call and its transport retry
// stalled silently before writing any artifacts. A focused repair prompt is
// intentionally skipped in this state: it is a narrower prompt that can
// consume the last budget while producing only the bootstrap scaffold. The
// hard invocation budget remains three (initial, retry, final fresh process).
func recoverDraftAfterSilentPreArtifactRetry(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, diagnostic StallDiagnostic) (bool, acpruntime.Result, error) {
	policy := adapter.RecoveryPolicy(task)
	if !runtimedrafts.IsDraftStep(task.StepID) ||
		!shouldClassifySilentNoFreshArtifactRepairStall(policy, result, diagnostic) {
		return false, acpruntime.Result{}, nil
	}
	if budget := ProviderInvocationBudgetFromContext(ctx); budget != nil && budget.Snapshot().Remaining <= 0 {
		return false, acpruntime.Result{}, nil
	}
	beforeWriteRoot, snapshotErr := snapshotWriteRootFiles(task.WriteRoot)
	if snapshotErr != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_manifest_missing_recovery", "draft manifest recovery write_root precheck failed", snapshotErr)
	}
	beforeDraftRoot, snapshotErr := snapshotWriteRootFiles(task.DraftFinalRoot)
	if snapshotErr != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "draft_artifact_manifest_missing_recovery", "draft manifest recovery draft_final_root precheck failed", snapshotErr)
	}

	retryPolicy := normalizeActivityPolicy(adapter.ActivityPolicy(task))
	if retryPolicy.RetryPreArtifactStallWindow > 0 {
		retryPolicy.PreArtifactStallWindow = retryPolicy.RetryPreArtifactStallWindow
		retryPolicy.PreArtifactWallClockWindow = retryPolicy.RetryPreArtifactStallWindow
	}
	emitDiagnostic(task, "draft zero-output pre-artifact retry will use a final fresh provider process", map[string]any{
		"provider":       string(adapter.Provider()),
		"recovery_mode":  "fresh_process",
		"recovery_stage": "repeated_silent_pre_artifact_stall",
		"severity":       "warning",
	})
	finalResult, finalErr := runProviderCommandWithTransition(ctx, task, adapter, retryPolicy, "transport_retry")
	if finalErr != nil {
		var finalStalled StallError
		if errors.As(finalErr, &finalStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitRetryCompletedDiagnostic(task, adapter.Provider(), finalStalled.Diagnostic.StallPhase, "final_fresh_process_artifact_only")
					return true, finalResult, nil
				} else if recovered, recoveredResult, recoveredErr := recoverDraftManifestShapeDeterministically(task, adapter, finalResult, beforeWriteRoot, beforeDraftRoot, err, "draft_manifest_missing_after_final_fresh_process"); recovered {
					return true, recoveredResult, recoveredErr
				}
			}
			if shouldClassifySilentNoFreshArtifactRepairStall(policy, finalResult, finalStalled.Diagnostic) ||
				shouldClassifySilentRetryExhaustionUnavailable(policy, task, finalResult) {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "retry", finalResult, "provider unavailable after repeated silent pre-artifact stalls", finalErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, finalResult, "retry", "final fresh-process retry stalled before producing valid draft artifacts", finalErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, finalResult, finalErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		if recovered, recoveredResult, recoveredErr := recoverDraftManifestShapeDeterministically(task, adapter, finalResult, beforeWriteRoot, beforeDraftRoot, err, "draft_manifest_missing_after_final_fresh_process"); recovered {
			return true, recoveredResult, recoveredErr
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, finalResult, "retry", "final fresh-process retry produced invalid draft artifacts", err)
	}
	emitRetryCompletedDiagnostic(task, adapter.Provider(), diagnostic.StallPhase, "final_fresh_process")
	return true, finalResult, nil
}

// shouldRetryDraftBootstrapAfterInitialStall keeps the first recovery attempt
// provider-authored. A draft bootstrap is an execution scaffold, not usable
// output; spending the initial retry on focused repair can exhaust the
// provider before it gets a chance to write the requested documents. This is
// intentionally limited to the initial post-artifact stall path. Enrichment
// stages still reject unchanged/scaffold content as a contract failure.
func shouldRetryDraftBootstrapAfterInitialStall(task acpruntime.Task, diagnostic StallDiagnostic, validationErr error) bool {
	if diagnostic.StallPhase != StallPhasePostArtifact || !runtimedrafts.IsDraftStep(task.StepID) {
		return false
	}
	if isDraftBootstrapOnlyValidationFailure(validationErr) {
		return true
	}
	// Providers sometimes write the markdown outputs before the manifest. In
	// that state validation reports only the missing manifest; a focused repair
	// can replace otherwise useful content with its bootstrap scaffold and burn
	// the remaining invocation budget. Prefer a fresh provider process whenever
	// the draft already has authored files, while retaining the bootstrap-only
	// check for callers that do not expose an authored-file diagnostic.
	if !classifyValidationIssues(validationErr).Has(issueMissingArtifact) {
		return false
	}
	return diagnostic.AuthoredFileCount > 0 || draftFinalRootHasBootstrapOnlyContent(task)
}

func draftFinalRootHasBootstrapOnlyContent(task acpruntime.Task) bool {
	root := strings.TrimSpace(task.DraftFinalRoot)
	if root == "" {
		return false
	}
	entries, err := os.ReadDir(filepath.Clean(root))
	if err != nil {
		return false
	}
	markdownCount := 0
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil || !runtimedrafts.DraftTextBootstrapOnly(string(raw)) {
			return false
		}
		markdownCount++
	}
	return markdownCount > 0
}

func recoverFocusedArtifactRepair(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string) (bool, acpruntime.Result, error) {
	if report, attempted, recoveryErr := recoverArchitectureHomeInlineHeadings(task, func() error {
		return adapter.ValidateArtifacts(task)
	}, validationErr); attempted {
		if recoveryErr == nil {
			emitArchitectureHomeInlineHeadingRecoveryCompletedDiagnostic(task, adapter.Provider(), report)
			return true, markArchitectureHomeInlineHeadingsRecovered(result, report), nil
		}
		emitArchitectureHomeInlineHeadingRecoveryFailedDiagnostic(task, adapter.Provider(), recoveryErr)
		var unsafeRestore architectureHomeUnsafeRestoreError
		if errors.As(recoveryErr, &unsafeRestore) {
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, architectureHomeInlineHeadingRecoveryMode, "Architecture Home normalization could not restore original bytes", recoveryErr)
		}
	}
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
	repairResult, repairErr, commandErr := runCollectArtifactPairRepairCommand(ctx, task, adapter, result, buildRepairSpec)
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
				} else if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, repairResult, err, "collect_pair_repair_post_artifact_invalid", beforeRepairFiles, options); recovered {
					return true, recoveredResult, recoveredErr
				}
			}
			retryStreamOnlyStall := shouldRetryStreamOnlyNoFreshCollectPairRepair(policy, repairResult, repairStalled.Diagnostic)
			if shouldRetrySilentNoFreshCollectPairRepair(policy, repairResult, repairStalled.Diagnostic) || retryStreamOnlyStall {
				emitFocusedArtifactRepairRetryScheduledDiagnostic(task, adapter.Provider(), "collect_pair_repair", repairErr)
				retrySpec := buildRepairSpec
				if retryStreamOnlyStall {
					retryValidationErr := fmt.Errorf("collect_pair_repair_stream_retry: %v: %w", validationErr, ErrStalledBeforeArtifacts)
					retrySpec = func() (CommandSpec, error) {
						return repairAdapter.CollectArtifactPairRepairCommandSpec(task, retryValidationErr)
					}
				}
				retryResult, retryErr, retryCommandErr := runCollectArtifactPairRepairCommand(ctx, task, adapter, repairResult, retrySpec)
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
							} else if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, retryResult, err, "collect_pair_repair_retry_post_artifact_invalid", beforeRepairFiles, options); recovered {
								return true, recoveredResult, recoveredErr
							}
						}
						emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", retryStalled.Diagnostic, retryErr)
						if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, retryResult, retryErr, "collect_pair_repair_retry_partial", beforeRepairFiles, options); recovered {
							return true, recoveredResult, recoveredErr
						}
						if shouldClassifySilentNoFreshArtifactRepairStall(policy, retryResult, retryStalled.Diagnostic) ||
							shouldClassifySilentRetryExhaustionUnavailable(policy, task, retryResult) {
							return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "collect_pair_repair", retryResult, "provider unavailable during collect pair recovery", retryErr)
						}
						return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, retryResult, "collect_pair_repair", "collect pair recovery stalled before valid artifacts were available", retryErr)
					}
					return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, retryResult, retryErr)
				}
				if err := adapter.ValidateArtifacts(task); err != nil {
					emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
					if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, retryResult, err, "collect_pair_repair_retry_invalid", beforeRepairFiles, options); recovered {
						return true, recoveredResult, recoveredErr
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
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", repairStalled.Diagnostic, repairErr)
			if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, repairResult, repairErr, "collect_pair_repair_partial", beforeRepairFiles, options); recovered {
				return true, recoveredResult, recoveredErr
			}
			if shouldClassifySilentNoFreshArtifactRepairStall(policy, repairResult, repairStalled.Diagnostic) ||
				shouldClassifySilentRetryExhaustionUnavailable(policy, task, repairResult) {
				return true, acpruntime.Result{}, wrapProviderUnavailable(adapter, task, "collect_pair_repair", repairResult, "provider unavailable during collect pair recovery", repairErr)
			}
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_pair_repair", "collect pair recovery stalled before valid artifacts were available", repairErr)
		}
		return true, acpruntime.Result{}, classifyCommandFailure(adapter, task, repairResult, repairErr)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		if shouldRetryTransientProviderUnavailableArtifactRepair(policy, repairResult, err, adapter.UnavailableMarkers()) {
			emitFocusedArtifactRepairRetryScheduledDiagnostic(task, adapter.Provider(), "collect_pair_repair", err)
			retryResult, retryErr, retryCommandErr := runCollectArtifactPairRepairCommand(ctx, task, adapter, repairResult, buildRepairSpec)
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
						} else if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, retryResult, err, "collect_pair_repair_retry_post_artifact_invalid", beforeRepairFiles, options); recovered {
							return true, recoveredResult, recoveredErr
						}
					}
					emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "collect_pair_repair", retryStalled.Diagnostic, retryErr)
					if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, retryResult, retryErr, "collect_pair_repair_retry_partial", beforeRepairFiles, options); recovered {
						return true, recoveredResult, recoveredErr
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
				if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, retryResult, err, "collect_pair_repair_retry_invalid", beforeRepairFiles, options); recovered {
					return true, recoveredResult, recoveredErr
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
		if recovered, recoveredResult, recoveredErr := recoverCollectManifestFallbackAfterPairRepair(ctx, task, adapter, repairResult, err, "collect_pair_repair_invalid", beforeRepairFiles, options); recovered {
			return true, recoveredResult, recoveredErr
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

func recoverCollectManifestFallbackAfterPairRepair(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, stage string, before writeRootFileSnapshot, options collectArtifactPairRepairOptions) (bool, acpruntime.Result, error) {
	if !shouldRecoverCollectManifestAfterPairRepair(task, before, options, validationErr) {
		return false, acpruntime.Result{}, nil
	}
	return recoverCollectManifestRepair(ctx, task, adapter, result, validationErr, stage)
}

func shouldRecoverCollectManifestAfterPairRepair(task acpruntime.Task, before writeRootFileSnapshot, options collectArtifactPairRepairOptions, validationErr error) bool {
	if validationErr == nil {
		return false
	}
	if options.allowManifestFallback {
		return true
	}
	if len(options.requiredExistingAuthoredDocMutationPaths) == 0 {
		return false
	}
	if err := validateCollectArtifactPairRepairOutcome(task, before, options); err != nil {
		return false
	}
	if collectWriteRootHasBootstrapOnlyAuthoredDoc(task) {
		return false
	}
	if len(collectProcessContaminatedAuthoredDocs(task)) > 0 {
		return false
	}
	if len(collectAuthoredDocsMentioningMissingRepoEvidencePaths(task, validationErr)) > 0 {
		return false
	}
	return true
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

func shouldRetrySilentNoFreshCollectPairRepair(policy RecoveryPolicy, result acpruntime.Result, diagnostic StallDiagnostic) bool {
	return policy.RetryInvalidOrMissingArtifactsOnce &&
		policy.RetryZeroOutputPreArtifactStallOnce &&
		shouldClassifySilentNoFreshArtifactRepairStall(policy, result, diagnostic)
}

func shouldRetryStreamOnlyNoFreshCollectPairRepair(policy RecoveryPolicy, result acpruntime.Result, diagnostic StallDiagnostic) bool {
	return policy.RetryInvalidOrMissingArtifactsOnce &&
		policy.RetryStreamOnlyPreArtifactStallOnce &&
		diagnostic.StallPhase == StallPhasePreArtifact &&
		strings.TrimSpace(result.Stdout) != "" &&
		!diagnostic.ArtifactObserved &&
		diagnostic.AuthoredFileCount == 0
}

func shouldClassifySilentNoFreshArtifactRepairStall(policy RecoveryPolicy, result acpruntime.Result, diagnostic StallDiagnostic) bool {
	return policy.ClassifySilentRetryExhaustionUnavailable &&
		diagnostic.StallPhase == StallPhasePreArtifact &&
		strings.TrimSpace(result.Stdout) == "" &&
		strings.TrimSpace(result.Stderr) == "" &&
		!diagnostic.ArtifactObserved &&
		diagnostic.AuthoredFileCount == 0
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
	if classifyValidationIssues(validationErr).Has(issueCollectTaskIdentity) {
		emitCollectManifestTaskIdentityRecoveryScheduledDiagnostic(task, adapter.Provider(), validationErr)
		if report, recoveryErr := recoverCollectManifestTaskIdentity(task, validationErr); recoveryErr == nil {
			if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
				return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, collectManifestTaskIdentityRecoveryMode, "task-identity recovery wrote outside shard-pack-manifest.json", err)
			}
			emitCollectManifestTaskIdentityRecoveryCompletedDiagnostic(task, adapter.Provider(), report)
			result = markCollectManifestTaskIdentityRecovered(result, report)
			return true, result, nil
		} else {
			emitCollectManifestTaskIdentityRecoveryFailedDiagnostic(task, adapter.Provider(), recoveryErr)
		}
	}
	if emptyDocs := collectWhitespaceOnlyAuthoredDocs(task); len(emptyDocs) > 0 {
		if recovered, recoveredResult, recoveredErr := recoverCollectArtifactPairRepairWithOptions(ctx, task, adapter, result, validationErr, stage+"_empty_authored_markdown", collectArtifactPairRepairOptions{
			allowNoProviderDiagnostics:               true,
			allowExistingAuthoredFiles:               true,
			requiredExistingAuthoredDocMutationPaths: stringSliceSet(emptyDocs),
			allowManifestFallback:                    false,
		}); recovered {
			return true, recoveredResult, recoveredErr
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "collect_pair_repair", "collect pair recovery is required for empty authored markdown", validationErr)
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
	if !collectWriteRootHasAuthoredMarkdownDoc(task) {
		if recovered, recoveredResult, recoveredErr := recoverCollectArtifactPairRepairWithOptions(ctx, task, adapter, result, validationErr, stage+"_missing_authored_markdown", collectArtifactPairRepairOptions{
			allowNoProviderDiagnostics: true,
			allowExistingAuthoredFiles: true,
			allowManifestFallback:      true,
		}); recovered {
			return true, recoveredResult, recoveredErr
		}
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "collect_pair_repair", "collect pair recovery is required when partial collect output contains no authored markdown", validationErr)
	}
	if isCollectManifestMissingFindingsOnlyCandidate(validationErr) {
		emitCollectManifestMissingFindingsRecoveryScheduledDiagnostic(task, adapter.Provider(), validationErr)
		if report, recoveryErr := recoverCollectManifestMissingFindings(task, validationErr); recoveryErr == nil {
			if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
				return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, collectManifestMissingFindingsRecoveryMode, "missing-findings recovery wrote outside shard-pack-manifest.json", err)
			}
			emitCollectManifestMissingFindingsRecoveryCompletedDiagnostic(task, adapter.Provider(), report)
			result = markCollectManifestMissingFindingsRecovered(result, report)
			return true, result, nil
		} else {
			emitCollectManifestMissingFindingsRecoveryFailedDiagnostic(task, adapter.Provider(), recoveryErr)
		}
	}
	if report, recoveryErr := recoverCollectManifestProvenanceKindAliases(task); recoveryErr == nil {
		if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, collectManifestProvenanceKindRecoveryMode, "provenance-kind recovery wrote outside shard-pack-manifest.json", err)
		}
		emitCollectManifestProvenanceKindRecoveryCompletedDiagnostic(task, adapter.Provider(), report)
		result = markCollectManifestProvenanceKindRecovered(result, report)
		return true, result, nil
	}
	if collectManifestFileMissing(task) ||
		isCollectManifestSemanticScaffoldFailure(validationErr) ||
		isCollectManifestEmptyPayloadFailure(validationErr) ||
		isCollectManifestCitationClaimBindingFailure(validationErr) {
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
	repairResult, repairErr := runCommandSpecWithTransition(ctx, task, spec, repairPolicy, "focused_repair")
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
		if shouldRetryCollectManifestShapeCleanup(err) {
			if cleanupResult, cleanupErr, handled := runCollectManifestShapeCleanup(ctx, task, adapter, repairAdapter, repairResult, beforeRepairFiles, err, repairPolicy, policy); handled {
				return true, cleanupResult, cleanupErr
			}
		}
		emitCollectManifestRepairExhaustedDiagnostic(task, adapter.Provider(), runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, repairResult, "collect_manifest_repair", "manifest-only collect repair did not produce valid collect artifacts", err)
	}
	emitCollectManifestRepairCompletedDiagnostic(task, adapter.Provider(), "")
	return true, repairResult, nil
}

func markCollectManifestMissingFindingsRecovered(result acpruntime.Result, report collectManifestMissingFindingsRecoveryReport) acpruntime.Result {
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	result.Diagnostics[collectManifestMissingFindingsRecoveryMode] = map[string]any{
		"recovery_mode":            collectManifestMissingFindingsRecoveryMode,
		"source":                   "runtime_shape_recovery",
		"provider_authored":        false,
		"inserted_field":           "semantic.findings",
		"before_digest":            report.BeforeDigest,
		"after_digest":             report.AfterDigest,
		"operator_review_required": true,
	}
	warning := "runtime_recovery: collect_manifest_missing_findings_recovery inserted an empty semantic.findings collection; treat as shape recovery evidence, not artifact-quality acceptance"
	if !containsRuntimeWarning(result.Execution.Warnings, warning) {
		result.Execution.Warnings = append(result.Execution.Warnings, warning)
	}
	return result
}

func markCollectManifestProvenanceKindRecovered(result acpruntime.Result, report collectManifestProvenanceKindRecoveryReport) acpruntime.Result {
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	result.Diagnostics[collectManifestProvenanceKindRecoveryMode] = map[string]any{
		"recovery_mode":            collectManifestProvenanceKindRecoveryMode,
		"source":                   "runtime_shape_recovery",
		"provider_authored":        false,
		"replacement_count":        report.ReplacementCount,
		"before_digest":            report.BeforeDigest,
		"after_digest":             report.AfterDigest,
		"operator_review_required": true,
	}
	warning := fmt.Sprintf("runtime_recovery: collect_manifest_provenance_kind_recovery canonicalized %d lexical provenance.kind aliases; treat as shape recovery evidence, not artifact-quality acceptance", report.ReplacementCount)
	if !containsRuntimeWarning(result.Execution.Warnings, warning) {
		result.Execution.Warnings = append(result.Execution.Warnings, warning)
	}
	return result
}

func markCollectManifestTaskIdentityRecovered(result acpruntime.Result, report collectManifestTaskIdentityRecoveryReport) acpruntime.Result {
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	result.Diagnostics[collectManifestTaskIdentityRecoveryMode] = map[string]any{
		"recovery_mode":            collectManifestTaskIdentityRecoveryMode,
		"source":                   "runtime_shape_recovery",
		"provider_authored":        false,
		"corrected_fields":         append([]string(nil), report.CorrectedFields...),
		"before_digest":            report.BeforeDigest,
		"after_digest":             report.AfterDigest,
		"operator_review_required": true,
	}
	warning := fmt.Sprintf("runtime_recovery: collect_manifest_task_identity_recovery restored authoritative task identity fields %s; treat as shape recovery evidence, not artifact-quality acceptance", strings.Join(report.CorrectedFields, ","))
	if !containsRuntimeWarning(result.Execution.Warnings, warning) {
		result.Execution.Warnings = append(result.Execution.Warnings, warning)
	}
	return result
}

func runCollectManifestShapeCleanup(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, repairAdapter CollectManifestRepairAdapter, baseResult acpruntime.Result, beforeRepairFiles writeRootFileSnapshot, validationErr error, repairPolicy ActivityPolicy, policy RecoveryPolicy) (acpruntime.Result, error, bool) {
	emitCollectManifestShapeCleanupScheduledDiagnostic(task, adapter.Provider(), runtimeArtifactSnapshot(task), validationErr)
	spec, err := repairAdapter.CollectManifestRepairCommandSpec(task, validationErr)
	if err != nil {
		return acpruntime.Result{}, classifyCommandFailure(adapter, task, baseResult, err), true
	}
	cleanupResult, cleanupRunErr := runCommandSpecWithTransition(ctx, task, spec, repairPolicy, "focused_repair")
	if cleanupRunErr != nil {
		if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
			return acpruntime.Result{}, classifyArtifactFailure(adapter, task, cleanupResult, "collect_manifest_shape_cleanup", "collect manifest shape cleanup wrote outside shard-pack-manifest.json", err), true
		}
		var cleanupStalled StallError
		if errors.As(cleanupRunErr, &cleanupStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := adapter.ValidateArtifacts(task); err == nil {
					emitCollectManifestShapeCleanupCompletedDiagnostic(task, adapter.Provider(), cleanupStalled.Diagnostic.StallPhase)
					return cleanupResult, nil, true
				}
			}
			emitCollectManifestShapeCleanupExhaustedDiagnostic(task, adapter.Provider(), cleanupStalled.Diagnostic, cleanupRunErr)
			return acpruntime.Result{}, classifyArtifactFailure(adapter, task, cleanupResult, "collect_manifest_shape_cleanup", "collect manifest shape cleanup stalled before valid artifacts were available", cleanupRunErr), true
		}
		return acpruntime.Result{}, classifyCommandFailure(adapter, task, cleanupResult, cleanupRunErr), true
	}
	if err := validateCollectManifestRepairWriteSet(task, beforeRepairFiles); err != nil {
		return acpruntime.Result{}, classifyArtifactFailure(adapter, task, cleanupResult, "collect_manifest_shape_cleanup", "collect manifest shape cleanup wrote outside shard-pack-manifest.json", err), true
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		emitCollectManifestShapeCleanupExhaustedDiagnostic(task, adapter.Provider(), runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		return acpruntime.Result{}, classifyArtifactFailure(adapter, task, cleanupResult, "collect_manifest_shape_cleanup", "collect manifest shape cleanup did not produce valid collect artifacts", err), true
	}
	emitCollectManifestShapeCleanupCompletedDiagnostic(task, adapter.Provider(), "")
	return cleanupResult, nil, true
}

func shouldRetryCollectManifestShapeCleanup(err error) bool {
	issues := classifyValidationIssues(err)
	if issues.HasAny(
		issueCollectRepoEvidence,
		issueCollectProcessContent,
		issueCollectWriteSet,
		issueCollectAuthoredMarkdown,
		issueCollectBootstrap,
	) {
		return false
	}
	return issues.HasAny(
		issueCollectDuplicateCitation,
		issueCollectQuestionText,
		issueCollectClaimBinding,
		issueCollectDocumentBinding,
		issueCollectCitationReference,
	)
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
		"recovery_mode":            "collect_manifest_runtime_recovery",
		"source":                   "runtime_recovery",
		"provider_authored":        false,
		"document_count":           report.DocumentCount,
		"entity_count":             report.EntityCount,
		"edge_count":               report.EdgeCount,
		"evidence_path":            strings.TrimSpace(report.EvidencePath),
		"operator_review_required": true,
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
	return classifyValidationIssues(err).Has(issueCollectSemanticScaffold)
}

func isCollectManifestEmptyPayloadFailure(err error) bool {
	return classifyValidationIssues(err).Has(issueCollectEmptyPayload)
}

func isCollectManifestCitationClaimBindingFailure(err error) bool {
	issues := classifyValidationIssues(err)
	if issues.HasAny(issueCollectRepoEvidence, issueCollectProcessContent) {
		return false
	}
	return issues.Has(issueCollectClaimBinding)
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

func collectWhitespaceOnlyAuthoredDocs(task acpruntime.Task) []string {
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
		if strings.TrimSpace(string(raw)) == "" {
			found = append(found, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(found)
	return found
}

func collectWriteRootHasAuthoredMarkdownDoc(task acpruntime.Task) bool {
	root := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	if root == "" || root == "." {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || entry.IsDir() {
			return nil
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || name == ShardPackManifestFileName || name == "runtime-execution.json" {
			return nil
		}
		switch strings.ToLower(filepath.Ext(name)) {
		case ".md", ".markdown":
			found = true
			return filepath.SkipAll
		default:
			return nil
		}
	})
	return found
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
	paths := []string{}
	seen := map[string]struct{}{}
	for _, issue := range classifyValidationIssues(err).Items {
		if issue.Code != issueCollectRepoEvidence || strings.TrimSpace(issue.Path) == "" {
			continue
		}
		if _, ok := seen[issue.Path]; !ok {
			seen[issue.Path] = struct{}{}
			paths = append(paths, issue.Path)
		}
	}
	sort.Strings(paths)
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
	// A provider can leave authored markdown behind while failing to create the
	// runtime draft manifest (a common post-artifact stall shape). In that
	// state, the bootstrap repair heredoc would consume the final recovery
	// invocation and leave only scaffold content. Go straight to the bounded
	// enrichment prompt so the provider can create the manifest and rewrite the
	// authored markdown in one call.
	if shouldRecoverDraftMissingManifestWithEnrichment(task, validationErr) {
		return recoverDraftArtifactEnrichment(ctx, task, adapter, result, validationErr, draftRepairEnrichmentStage(stage, validationErr))
	}
	if shouldRecoverDraftRepairValidationWithEnrichment(task, validationErr) {
		return recoverDraftArtifactEnrichment(ctx, task, adapter, result, validationErr, draftRepairEnrichmentStage(stage, validationErr))
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
				} else if shouldRecoverDraftRepairValidationWithEnrichment(task, err) {
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
						} else if shouldRecoverDraftRepairValidationWithEnrichment(task, err) {
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
				if shouldRecoverDraftRepairValidationWithEnrichment(task, err) {
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
		if shouldRecoverDraftRepairValidationWithEnrichment(task, err) {
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
	if recovered, recoveredResult, recoveredErr := recoverArchitectureHomePlaceholderReferences(task, adapter, result, validationErr, beforeWriteRoot, beforeDraftRoot); recovered {
		return true, recoveredResult, recoveredErr
	}
	// A provider can leave only the bootstrap proposal scaffold after a
	// successful analysis.  Repeatedly asking the provider to rewrite that
	// scaffold is both expensive and unreliable (especially for headless
	// providers that return no output after the first attempt).  When the
	// validation failure is limited to bootstrap/finding linkage, produce a
	// small evidence-linked proposal pair locally.  This keeps the recovery
	// deterministic while still requiring the normal adapter validation below.
	if shouldUseDeterministicProposalDraftFallback(task, validationErr) {
		if fallbackErr := writeDeterministicProposalDraft(task); fallbackErr == nil {
			if outcomeErr := validateDraftArtifactEnrichmentOutcome(task, beforeWriteRoot, beforeDraftRoot, adapter.ValidateArtifacts(task), validationErr, stage); outcomeErr == nil {
				emitDiagnostic(task, "deterministic proposal draft fallback completed", map[string]any{
					"provider":      adapter.Provider(),
					"recovery_mode": "draft_artifact_deterministic_fallback",
					"step_id":       task.StepID,
				})
				return true, result, nil
			}
		}
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
	if writeSetErr := validateDraftArtifactRepairWriteSetForStage(task, beforeWriteRoot, beforeDraftRoot, stage); writeSetErr != nil {
		rolledBack, rollbackErr := rollbackCreatedEmptyDraftSidecars(task, beforeWriteRoot, beforeDraftRoot, stage, writeSetErr)
		if rollbackErr != nil {
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft enrichment empty-sidecar rollback failed", rollbackErr)
		}
		if rolledBack {
			emitDiagnostic(task, "draft enrichment empty sidecars rolled back", map[string]any{
				"provider":       adapter.Provider(),
				"recovery_mode":  "draft_artifact_enrichment",
				"recovery_stage": stage,
				"reason":         writeSetErr.Error(),
			})
		} else if shouldRetryDraftWriteSetCleanupEnrichment(stage, task, beforeWriteRoot, beforeDraftRoot, writeSetErr) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftWriteSetCleanupRetryError(writeSetErr), "draft_artifact_enrichment_write_set_cleanup")
		} else {
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft enrichment wrote outside the draft artifact write set", writeSetErr)
		}
	}
	if strings.TrimSpace(stage) == "draft_artifact_enrichment_write_set_cleanup" {
		if leftovers := draftWriteRootOutputFiles(task); len(leftovers) > 0 {
			err := fmt.Errorf("draft artifact enrichment write-set cleanup left misplaced write_root draft files: %s", strings.Join(leftovers, ", "))
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft enrichment wrote outside the draft artifact write set", err)
		}
	}
	if enrichmentErr != nil {
		var enrichmentStalled StallError
		if errors.As(enrichmentErr, &enrichmentStalled) {
			if policy.AcceptValidArtifactsAfterStop {
				if err := validateDraftArtifactEnrichmentOutcome(task, beforeWriteRoot, beforeDraftRoot, adapter.ValidateArtifacts(task), validationErr, stage); err == nil {
					emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "completed_after_controlled_stop", runtimeArtifactSnapshot(task))
					emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", enrichmentStalled.Diagnostic.StallPhase)
					return true, enrichmentResult, nil
				} else if recovered, recoveredResult, recoveredErr := recoverDraftManifestShapeDeterministically(task, adapter, enrichmentResult, beforeWriteRoot, beforeDraftRoot, err, stage); recovered {
					if recoveredErr != nil {
						return true, acpruntime.Result{}, recoveredErr
					}
					emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_manifest_shape_recovery", enrichmentStalled.Diagnostic.StallPhase)
					return true, recoveredResult, nil
				} else if shouldRetryDraftManifestShapeEnrichment(stage, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_manifest_shape")
				} else if shouldRetryDraftShardStatusCleanupEnrichment(stage, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_shard_status_cleanup")
				} else if shouldRetryDraftSilentWriteFirstEnrichment(stage, task, beforeDraftRoot, enrichmentResult, enrichmentStalled.Diagnostic, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftWriteFirstRetryError(err), "draft_artifact_enrichment_write_first_retry")
				} else if shouldRetryDraftCompactStep2Enrichment(stage, task, beforeDraftRoot, enrichmentResult, enrichmentStalled.Diagnostic, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftCompactStep2RetryError(err), "draft_artifact_enrichment_compact_step2_retry")
				} else if shouldRetryDraftCompactStep4Enrichment(stage, task, beforeDraftRoot, enrichmentResult, enrichmentStalled.Diagnostic, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftCompactStep4RetryError(err), "draft_artifact_enrichment_compact_step4_retry")
				} else if isDraftEnrichmentNoopOrScaffoldFailure(err) {
					if shouldRetryDraftMissingPythonEnrichment(stage, enrichmentResult, err) {
						return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_python3_retry")
					}
					if shouldRetryDraftPrintedCommandEnrichment(stage, enrichmentResult, err) {
						return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftCommandTextRetryError(err), "draft_artifact_enrichment_command_text_retry")
					}
					if shouldRetryDraftMarkerCleanupEnrichment(stage, task, beforeDraftRoot, err) {
						return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_marker_cleanup")
					}
					emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "stalled", runtimeArtifactSnapshot(task))
					emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", enrichmentStalled.Diagnostic, err)
					return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft_artifact_enrichment_noop_or_scaffold", err)
				} else if shouldRetryDraftMalformedMarkdownEnrichment(stage, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_markdown_syntax")
				} else if shouldRetryDraftDownstreamIndexClaimEnrichment(stage, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_downstream_index_retry")
				} else if shouldRetryDraftShardStatusCleanupEnrichment(stage, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_shard_status_cleanup")
				} else if shouldRetryDraftArchitectureHomeCleanupEnrichment(stage, task, beforeDraftRoot, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_architecture_home_cleanup")
				} else if shouldRetryDraftMarkerCleanupEnrichment(stage, task, beforeDraftRoot, err) {
					return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_marker_cleanup")
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
	if err := validateDraftArtifactEnrichmentOutcome(task, beforeWriteRoot, beforeDraftRoot, adapter.ValidateArtifacts(task), validationErr, stage); err != nil {
		emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "invalid", runtimeArtifactSnapshot(task))
		if recovered, recoveredResult, recoveredErr := recoverDraftManifestShapeDeterministically(task, adapter, enrichmentResult, beforeWriteRoot, beforeDraftRoot, err, stage); recovered {
			if recoveredErr != nil {
				return true, acpruntime.Result{}, recoveredErr
			}
			emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_manifest_shape_recovery", "")
			return true, recoveredResult, nil
		}
		if shouldRetryDraftManifestShapeEnrichment(stage, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_manifest_shape")
		}
		if shouldRetryDraftShardStatusCleanupEnrichment(stage, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_shard_status_cleanup")
		}
		if isDraftEnrichmentNoopOrScaffoldFailure(err) {
			if shouldRetryDraftMissingPythonEnrichment(stage, enrichmentResult, err) {
				return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_python3_retry")
			}
			if shouldRetryDraftPrintedCommandEnrichment(stage, enrichmentResult, err) {
				return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, draftCommandTextRetryError(err), "draft_artifact_enrichment_command_text_retry")
			}
			if shouldRetryDraftMarkerCleanupEnrichment(stage, task, beforeDraftRoot, err) {
				return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_marker_cleanup")
			}
			emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
			return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft_artifact_enrichment_noop_or_scaffold", err)
		}
		if shouldRetryDraftMalformedMarkdownEnrichment(stage, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_markdown_syntax")
		}
		if shouldRetryDraftDownstreamIndexClaimEnrichment(stage, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_downstream_index_retry")
		}
		if shouldRetryDraftShardStatusCleanupEnrichment(stage, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_shard_status_cleanup")
		}
		if shouldRetryDraftArchitectureHomeCleanupEnrichment(stage, task, beforeDraftRoot, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_architecture_home_cleanup")
		}
		if shouldRetryDraftMarkerCleanupEnrichment(stage, task, beforeDraftRoot, err) {
			return recoverDraftArtifactEnrichment(ctx, task, adapter, enrichmentResult, err, "draft_artifact_enrichment_marker_cleanup")
		}
		emitFocusedArtifactRepairExhaustedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", runtimeArtifactSnapshot(task).stallDiagnostic(), err)
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, enrichmentResult, "draft_artifact_enrichment", "draft artifact enrichment did not produce valid draft artifact contract", err)
	}
	emitDraftArtifactEnrichmentSnapshotDiagnostic(task, adapter.Provider(), "completed", runtimeArtifactSnapshot(task))
	emitFocusedArtifactRepairCompletedDiagnostic(task, adapter.Provider(), "draft_artifact_enrichment", "")
	return true, enrichmentResult, nil
}

// recoverArchitectureHomePlaceholderReferences repairs only explicit `/...`
// repository references in an Architecture Home overview. The validator
// remains strict for all other missing references; this path simply converts
// provider shorthand into a concrete existing evidence file before spending
// another provider invocation on enrichment.
func recoverArchitectureHomePlaceholderReferences(task acpruntime.Task, adapter ProviderAdapter, result acpruntime.Result, validationErr error, beforeWriteRoot, beforeDraftRoot writeRootFileSnapshot) (bool, acpruntime.Result, error) {
	stepID := strings.TrimSpace(task.StepID)
	if stepID != "init.step2.asis_docs" && stepID != "refresh.step2.asis_docs" {
		return false, acpruntime.Result{}, nil
	}
	if !isArchitectureHomeRepositoryReferenceError(validationErr) {
		return false, acpruntime.Result{}, nil
	}
	manifest, _, err := runtimedrafts.Load(task.WriteRoot, runtimedrafts.AsIsManifestFile)
	if err != nil {
		return false, acpruntime.Result{}, nil
	}
	overviewPath := ""
	for _, output := range manifest.Outputs {
		if filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath))) != "reports/as-is/overview.md" {
			continue
		}
		if overviewPath != "" {
			return false, acpruntime.Result{}, nil
		}
		overviewPath = filepath.Join(filepath.Clean(task.DraftFinalRoot), filepath.Clean(output.Path))
	}
	if overviewPath == "" {
		return false, acpruntime.Result{}, nil
	}
	original, err := os.ReadFile(overviewPath)
	if err != nil {
		return false, acpruntime.Result{}, nil
	}
	repaired, replacements, changed := normalizeArchitectureHomePlaceholderReferences(original, collectTaskRepoRoots(task))
	if !changed || len(replacements) == 0 {
		return false, acpruntime.Result{}, nil
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(overviewPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := writeArchitectureHomeAtomic(overviewPath, repaired, mode); err != nil {
		return true, acpruntime.Result{}, classifyArtifactFailure(adapter, task, result, "architecture_home_placeholder_recovery", "Architecture Home placeholder recovery could not write overview", err)
	}
	if err := adapter.ValidateArtifacts(task); err != nil {
		_ = writeArchitectureHomeAtomic(overviewPath, original, mode)
		return false, acpruntime.Result{}, nil
	}
	if err := validateDraftArtifactEnrichmentOutcome(task, beforeWriteRoot, beforeDraftRoot, nil, validationErr, "draft_artifact_enrichment_architecture_home_placeholder_recovery"); err != nil {
		_ = writeArchitectureHomeAtomic(overviewPath, original, mode)
		return false, acpruntime.Result{}, nil
	}
	emitDiagnostic(task, "Architecture Home placeholder references recovered", map[string]any{
		"provider":        adapter.Provider(),
		"recovery_mode":   "architecture_home_placeholder_recovery",
		"replacements":    append([]string(nil), replacements...),
		"operator_review": true,
	})
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	result.Diagnostics["architecture_home_placeholder_recovery"] = map[string]any{
		"recovery_mode": "architecture_home_placeholder_recovery",
		"replacements":  append([]string(nil), replacements...),
	}
	return true, result, nil
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

func validateDraftArtifactEnrichmentOutcome(task acpruntime.Task, beforeWriteRoot writeRootFileSnapshot, beforeDraftRoot writeRootFileSnapshot, validationErr error, recoveryCause error, stage string) error {
	if validationErr != nil {
		if isDraftBootstrapOnlyValidationFailure(validationErr) {
			return fmt.Errorf("draft_artifact_enrichment_noop_or_scaffold: %w", validationErr)
		}
		return validationErr
	}
	if strings.TrimSpace(stage) == "draft_artifact_enrichment_write_set_cleanup" {
		if draftWriteRootDuplicateCleanupOccurred(task, beforeWriteRoot, beforeDraftRoot) ||
			draftStep0CanonicalDuplicateCleanupOccurred(task, beforeDraftRoot) {
			return nil
		}
	}
	if !allDraftMarkdownOutputsChanged(task, beforeDraftRoot) {
		if targetedArchitectureHomeRewriteIsValid(task, beforeDraftRoot, recoveryCause) {
			return nil
		}
		return fmt.Errorf("draft_artifact_enrichment_noop_or_scaffold: not all referenced markdown draft files changed")
	}
	return nil
}

func targetedArchitectureHomeRewriteIsValid(task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot, recoveryCause error) bool {
	stepID := strings.TrimSpace(task.StepID)
	if stepID != "init.step2.asis_docs" && stepID != "refresh.step2.asis_docs" {
		return false
	}
	if !classifyValidationIssues(recoveryCause).Has(issueDraftArchitectureHome) && !isArchitectureHomeRepositoryReferenceError(recoveryCause) {
		return false
	}
	afterDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		return false
	}
	beforeState, beforeExists := beforeDraftRoot["overview.md"]
	afterState, afterExists := afterDraftRoot["overview.md"]
	return afterExists && (beforeExists != afterExists || beforeState != afterState)
}

func isArchitectureHomeRepositoryReferenceError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "architecture home repository reference")
}

func shouldRetryDraftWriteSetCleanupEnrichment(stage string, task acpruntime.Task, beforeWriteRoot writeRootFileSnapshot, beforeDraftRoot writeRootFileSnapshot, err error) bool {
	if strings.TrimSpace(stage) == "draft_artifact_enrichment_write_set_cleanup" || err == nil {
		return false
	}
	if shouldRetryDraftStep0CanonicalDuplicateCleanup(task, beforeWriteRoot, beforeDraftRoot, err) {
		return true
	}
	if !classifyValidationIssues(err).Has(issueDraftWriteRootForbidden) {
		return false
	}
	afterWriteRoot, writeErr := snapshotWriteRootFiles(task.WriteRoot)
	if writeErr != nil {
		return false
	}
	afterDraftRoot, draftErr := snapshotWriteRootFiles(task.DraftFinalRoot)
	if draftErr != nil {
		return false
	}
	allowedDraftRootChange := allowedDraftRootRepairMutation(task)
	draftRootChanges := unexpectedRepairMutationDetails(beforeDraftRoot, afterDraftRoot, func(change repairMutation) bool {
		return allowedDraftRootChange(change.Path, change.State)
	})
	if len(draftRootChanges) > 0 {
		return false
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	changes := unexpectedRepairMutationDetails(beforeWriteRoot, afterWriteRoot, func(change repairMutation) bool {
		return strings.TrimSpace(manifestFile) != "" && change.Path == manifestFile
	})
	if len(changes) == 0 {
		return false
	}
	for _, change := range changes {
		if change.Op != "created" && change.Op != "modified" {
			return false
		}
		if !isAllowedDraftMarkdownOutputPath(task, change.Path) || change.State.IsDir {
			return false
		}
		draftState, ok := afterDraftRoot[change.Path]
		if !ok || !sameWriteRootFileContent(change.State, draftState) {
			return false
		}
	}
	return true
}

func rollbackCreatedEmptyDraftSidecars(task acpruntime.Task, beforeWriteRoot writeRootFileSnapshot, beforeDraftRoot writeRootFileSnapshot, stage string, writeSetErr error) (bool, error) {
	if writeSetErr == nil || !strings.Contains(writeSetErr.Error(), "draft repair wrote forbidden draft_final_root files") {
		return false, nil
	}
	afterWriteRoot, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return false, err
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	writeRootChanges := unexpectedRepairMutationDetails(beforeWriteRoot, afterWriteRoot, func(change repairMutation) bool {
		return strings.TrimSpace(manifestFile) != "" && change.Path == manifestFile
	})
	if len(writeRootChanges) > 0 {
		return false, nil
	}
	afterDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		return false, err
	}
	allowedDraftRootChange := allowedDraftRootRepairMutation(task)
	changes := unexpectedRepairMutationDetails(beforeDraftRoot, afterDraftRoot, func(change repairMutation) bool {
		return allowedDraftRootChange(change.Path, change.State)
	})
	if len(changes) == 0 {
		return false, nil
	}
	for _, change := range changes {
		if change.Op != "created" || change.State.IsDir || !change.State.Mode.IsRegular() || change.State.Size != 0 {
			return false, nil
		}
	}
	cleanRoot := filepath.Clean(strings.TrimSpace(task.DraftFinalRoot))
	for _, change := range changes {
		path := filepath.Join(cleanRoot, filepath.FromSlash(change.Path))
		contained, err := filepath.Rel(cleanRoot, path)
		if err != nil || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) || filepath.IsAbs(contained) {
			return false, fmt.Errorf("refuse empty-sidecar rollback outside draft_final_root: %s", change.Path)
		}
		if err := os.Remove(path); err != nil {
			return false, fmt.Errorf("remove created empty draft sidecar %q: %w", change.Path, err)
		}
	}
	if err := validateDraftArtifactRepairWriteSetForStage(task, beforeWriteRoot, beforeDraftRoot, stage); err != nil {
		return false, fmt.Errorf("revalidate draft write set after empty-sidecar rollback: %w", err)
	}
	return true, nil
}

func shouldRetryDraftStep0CanonicalDuplicateCleanup(task acpruntime.Task, beforeWriteRoot writeRootFileSnapshot, beforeDraftRoot writeRootFileSnapshot, err error) bool {
	if !classifyValidationIssues(err).Has(issueDraftFinalRootForbidden) {
		return false
	}
	afterWriteRoot, writeErr := snapshotWriteRootFiles(task.WriteRoot)
	if writeErr != nil {
		return false
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	writeRootChanges := unexpectedRepairMutationDetails(beforeWriteRoot, afterWriteRoot, func(change repairMutation) bool {
		return strings.TrimSpace(manifestFile) != "" && change.Path == manifestFile
	})
	if len(writeRootChanges) > 0 {
		return false
	}
	afterDraftRoot, draftErr := snapshotWriteRootFiles(task.DraftFinalRoot)
	if draftErr != nil {
		return false
	}
	allowedDraftRootChange := allowedDraftRootRepairMutation(task)
	draftRootChanges := unexpectedRepairMutationDetails(beforeDraftRoot, afterDraftRoot, func(change repairMutation) bool {
		return allowedDraftRootChange(change.Path, change.State)
	})
	if len(draftRootChanges) == 0 {
		return false
	}
	hasCanonicalDuplicate := false
	for _, change := range draftRootChanges {
		if !isStep0CanonicalDraftDuplicateCreation(task, change) {
			return false
		}
		if filepath.ToSlash(filepath.Clean(change.Path)) == "skills/subagents.yaml" && !change.State.IsDir {
			hasCanonicalDuplicate = true
		}
	}
	return hasCanonicalDuplicate
}

func draftWriteRootDuplicateCleanupOccurred(task acpruntime.Task, beforeWriteRoot writeRootFileSnapshot, beforeDraftRoot writeRootFileSnapshot) bool {
	afterWriteRoot, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return false
	}
	for _, output := range loadAllowedDraftOutputs(task) {
		rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		if rel == "" || rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
			continue
		}
		if strings.ToLower(filepath.Ext(rel)) != ".md" {
			continue
		}
		if !writeRootPathWasDraftDuplicate(beforeWriteRoot, beforeDraftRoot, rel) {
			continue
		}
		if _, stillExists := afterWriteRoot[rel]; !stillExists {
			return true
		}
	}
	return false
}

func draftStep0CanonicalDuplicateCleanupOccurred(task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot) bool {
	afterDraftRoot, err := snapshotWriteRootFiles(task.DraftFinalRoot)
	if err != nil {
		return false
	}
	filePath := "skills/subagents.yaml"
	beforeState, beforeOK := beforeDraftRoot[filePath]
	if !beforeOK || beforeState.IsDir {
		return false
	}
	_, afterOK := afterDraftRoot[filePath]
	return !afterOK
}

func isStep0CanonicalDraftDuplicateCreation(task acpruntime.Task, change repairMutation) bool {
	return change.Op == "created" && step0CanonicalDraftDuplicatePathMatches(task, change.Path, change.State.IsDir)
}

func isStep0CanonicalDraftDuplicateCleanupMutation(task acpruntime.Task, change repairMutation) bool {
	return change.Op == "deleted" && step0CanonicalDraftDuplicatePathMatches(task, change.Path, change.State.IsDir)
}

func step0CanonicalDraftDuplicatePathMatches(task acpruntime.Task, rawPath string, isDir bool) bool {
	if strings.TrimSpace(task.StepID) != "init.step0.constitution" {
		return false
	}
	path := filepath.ToSlash(filepath.Clean(strings.TrimSpace(rawPath)))
	for _, output := range loadAllowedDraftOutputs(task) {
		outputPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		canonicalPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.CanonicalPath)))
		if outputPath != "baseline-subagents.yaml" || canonicalPath != "skills/subagents.yaml" {
			continue
		}
		if isDir {
			return path == "skills"
		}
		return path == "skills/subagents.yaml"
	}
	return false
}

func draftWriteRootOutputFiles(task acpruntime.Task) []string {
	writeRoot, err := snapshotWriteRootFiles(task.WriteRoot)
	if err != nil {
		return nil
	}
	out := make([]string, 0)
	for _, output := range loadAllowedDraftOutputs(task) {
		rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		if rel == "" || rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
			continue
		}
		if strings.ToLower(filepath.Ext(rel)) != ".md" {
			continue
		}
		if state, ok := writeRoot[rel]; ok && !state.IsDir {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out
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
	return classifyValidationIssues(err).Has(issueDraftNoopScaffold)
}

func shouldRetryDraftMalformedMarkdownEnrichment(stage string, err error) bool {
	return classifyValidationIssues(err).Allows(stage, recoveryTransition{
		IssueCode: issueDraftMalformedMarkdown, TargetStage: "draft_artifact_enrichment_markdown_syntax", MaxAttempts: 1,
	})
}

func shouldRetryDraftManifestShapeEnrichment(stage string, err error) bool {
	issues := classifyValidationIssues(err)
	return issues.Allows(stage, recoveryTransition{
		IssueCode: issueDraftUnknownField, TargetStage: "draft_artifact_enrichment_manifest_shape", MaxAttempts: 1,
	}) &&
		issues.HasAny(issueDraftManifestParse, issueDraftManifestOutputs)
}

func shouldRetryDraftDownstreamIndexClaimEnrichment(stage string, err error) bool {
	issues := classifyValidationIssues(err)
	return issues.Has(issueDraftManifestOutputs) && issues.Allows(stage, recoveryTransition{
		IssueCode: issueDraftDownstreamIndex, TargetStage: "draft_artifact_enrichment_downstream_index_retry", MaxAttempts: 1,
	}) &&
		!issues.HasAny(
			issueDraftMalformedMarkdown,
			issueDraftShardStatus,
			issueDraftArchitectureHome,
			issueDraftMarkerCleanup,
			issueDraftFindingLinkage,
			issueDraftProposalSection,
			issueDraftEvidence,
			issueDraftOperatorSummary,
			issueDraftEmptyShardEvidence,
		)
}

func shouldRetryDraftShardStatusCleanupEnrichment(stage string, err error) bool {
	if isDraftEnrichmentNoopOrScaffoldFailure(err) {
		return false
	}
	return classifyValidationIssues(err).Allows(stage, recoveryTransition{
		IssueCode: issueDraftShardStatus, TargetStage: "draft_artifact_enrichment_shard_status_cleanup", MaxAttempts: 1,
	})
}

func shouldRetryDraftArchitectureHomeCleanupEnrichment(stage string, task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot, err error) bool {
	if err == nil {
		return false
	}
	stepID := strings.TrimSpace(task.StepID)
	if stepID != "init.step2.asis_docs" && stepID != "refresh.step2.asis_docs" {
		return false
	}
	if !classifyValidationIssues(err).Allows(stage, recoveryTransition{
		IssueCode: issueDraftArchitectureHome, TargetStage: "draft_artifact_enrichment_architecture_home_cleanup", MaxAttempts: 1,
	}) {
		return false
	}
	return allDraftMarkdownOutputsChanged(task, beforeDraftRoot)
}

func shouldRetryDraftMarkerCleanupEnrichment(stage string, task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot, err error) bool {
	if err == nil {
		return false
	}
	issues := classifyValidationIssues(err)
	if !issues.Allows(stage, recoveryTransition{
		IssueCode: issueDraftBootstrap, TargetStage: "draft_artifact_enrichment_marker_cleanup", MaxAttempts: 1,
	}) && !issues.Allows(stage, recoveryTransition{
		IssueCode: issueDraftMarkerCleanup, TargetStage: "draft_artifact_enrichment_marker_cleanup", MaxAttempts: 1,
	}) {
		return false
	}
	return allDraftMarkdownOutputsChanged(task, beforeDraftRoot) && draftMarkdownContainsMarkerCleanupCandidate(task)
}

func draftMarkdownContainsMarkerCleanupCandidate(task acpruntime.Task) bool {
	for _, output := range loadAllowedDraftOutputs(task) {
		rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		if rel == "" || rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
			continue
		}
		if strings.ToLower(filepath.Ext(rel)) != ".md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(strings.TrimSpace(task.DraftFinalRoot), rel))
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(raw))
		if draftMarkdownHasDirectScaffoldMarker(lower) {
			continue
		}
		if draftMarkdownHasProcessMarkerCleanupCandidate(lower) {
			return true
		}
	}
	return false
}

func draftMarkdownHasDirectScaffoldMarker(lower string) bool {
	markers := []string{
		"provider wrote this draft artifact",
		"drafted required runtime artifacts",
		"draft surface initialized",
		"runtime proposal surface initialized",
		"runtime draft recovery initialized",
		"draft recovery initialized",
		"treat this as diagnostic evidence until",
		"bootstrap-only placeholder",
		"placeholder draft content",
		"placeholder draft text",
		"placeholder content",
		"placeholder proposal content",
		"replace placeholder",
		"replaced placeholder",
		"replacing placeholders",
		"current run evidence should be reviewed before promotion",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func draftMarkdownHasProcessMarkerCleanupCandidate(lower string) bool {
	markers := []string{
		"reports/taskruns/",
		"staging/final/",
		"staging/shards/",
		"bounded read root",
		"bounded read roots",
		"bounded staged evidence",
		"bounded evidence read",
		"bounded read pass",
		"current draft manifest",
		"draft manifest",
		"draft space",
		"manifest target",
		"manifest retains",
		"recovery pass",
		"recovery action",
		"enrichment read",
		"enrichment pass",
		"validator output",
		"pipeline artifacts",
		"later passes",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func shouldRetryDraftPrintedCommandEnrichment(stage string, result acpruntime.Result, err error) bool {
	if strings.TrimSpace(stage) == "draft_artifact_enrichment_command_text_retry" || !isDraftEnrichmentNoopOrScaffoldFailure(err) {
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

func shouldRetryDraftSilentWriteFirstEnrichment(stage string, task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot, result acpruntime.Result, diagnostic StallDiagnostic, err error) bool {
	trimmedStage := strings.TrimSpace(stage)
	if strings.HasPrefix(trimmedStage, "draft_artifact_enrichment_") || err == nil {
		return false
	}
	if diagnostic.StallPhase != StallPhasePreArtifact {
		return false
	}
	if strings.TrimSpace(result.Stdout) != "" || strings.TrimSpace(result.Stderr) != "" {
		return false
	}
	return !allDraftMarkdownOutputsChanged(task, beforeDraftRoot)
}

func shouldRetryDraftCompactStep2Enrichment(stage string, task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot, result acpruntime.Result, diagnostic StallDiagnostic, err error) bool {
	if strings.TrimSpace(stage) != "draft_artifact_enrichment_write_first_retry" || err == nil {
		return false
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
	default:
		return false
	}
	if diagnostic.StallPhase != StallPhasePreArtifact {
		return false
	}
	if strings.TrimSpace(result.Stdout) != "" || strings.TrimSpace(result.Stderr) != "" {
		return false
	}
	return !allDraftMarkdownOutputsChanged(task, beforeDraftRoot)
}

func shouldRetryDraftCompactStep4Enrichment(stage string, task acpruntime.Task, beforeDraftRoot writeRootFileSnapshot, result acpruntime.Result, diagnostic StallDiagnostic, err error) bool {
	if strings.TrimSpace(stage) != "draft_artifact_enrichment_write_first_retry" || err == nil {
		return false
	}
	switch strings.TrimSpace(task.StepID) {
	case "init.step4.proposals", "refresh.step4.proposals":
	default:
		return false
	}
	if diagnostic.StallPhase != StallPhasePreArtifact {
		return false
	}
	if strings.TrimSpace(result.Stdout) != "" || strings.TrimSpace(result.Stderr) != "" {
		return false
	}
	return !allDraftMarkdownOutputsChanged(task, beforeDraftRoot)
}

func draftWriteFirstRetryError(err error) error {
	if err == nil {
		return errors.New("draft_artifact_enrichment_write_first_retry")
	}
	return fmt.Errorf("draft_artifact_enrichment_write_first_retry: %w", err)
}

func draftCompactStep2RetryError(err error) error {
	if err == nil {
		return errors.New("draft_artifact_enrichment_compact_step2_retry")
	}
	return fmt.Errorf("draft_artifact_enrichment_compact_step2_retry: %w", err)
}

func draftCompactStep4RetryError(err error) error {
	if err == nil {
		return errors.New("draft_artifact_enrichment_compact_step4_retry")
	}
	return fmt.Errorf("draft_artifact_enrichment_compact_step4_retry: %w", err)
}

func draftWriteSetCleanupRetryError(err error) error {
	if err == nil {
		return errors.New("draft_artifact_enrichment_write_set_cleanup")
	}
	return fmt.Errorf("draft_artifact_enrichment_write_set_cleanup: %w", err)
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
	return classifyValidationIssues(err).Has(issueDraftBootstrap)
}

func shouldRecoverDraftRepairValidationWithEnrichment(task acpruntime.Task, err error) bool {
	if err == nil || !runtimedrafts.IsDraftStep(task.StepID) {
		return false
	}
	issues := classifyValidationIssues(err)
	return issues.HasAny(
		issueDraftBootstrap,
		issueDraftManifestOutputs,
		issueDraftManifestParse,
		issueDraftMalformedMarkdown,
		issueDraftShardStatus,
		issueDraftFindingLinkage,
		issueDraftProposalSection,
		issueDraftEvidence,
		issueDraftOperatorSummary,
		issueDraftEmptyShardEvidence,
		issueDraftMarkerCleanup,
	)
}

func shouldRecoverDraftMissingManifestWithEnrichment(task acpruntime.Task, err error) bool {
	if err == nil || !runtimedrafts.IsDraftStep(task.StepID) {
		return false
	}
	issues := classifyValidationIssues(err)
	if !issues.Has(issueMissingArtifact) {
		return false
	}
	text := strings.ToLower(err.Error())
	if !strings.Contains(text, "read runtime draft manifest") &&
		!strings.Contains(text, "parse runtime draft manifest") {
		return false
	}
	// Only take this path when the provider left authored markdown behind. A
	// truly silent run must retain the existing provider-unavailable recovery
	// classification rather than being converted into an enrichment failure.
	snapshot, snapshotErr := snapshotWriteRootFiles(task.DraftFinalRoot)
	if snapshotErr != nil {
		return false
	}
	for path := range snapshot {
		if strings.EqualFold(filepath.Ext(path), ".md") {
			return true
		}
	}
	return false
}

func shouldUseDeterministicProposalDraftFallback(task acpruntime.Task, err error) bool {
	if err == nil || (task.StepID != "init.step4.proposals" && task.StepID != "refresh.step4.proposals") {
		return false
	}
	issues := classifyValidationIssues(err)
	// Only replace the recovery scaffold itself.  Substantive provider-authored
	// proposals that need linkage/section cleanup must still go through the
	// focused provider enrichment path.
	if issues.Has(issueDraftBootstrap) {
		return proposalDraftHasRecoveryScaffold(task)
	}
	// A provider can produce a fully formed-looking proposal while claiming
	// that no structured findings exist.  That claim is still a recovery
	// scaffold when the current-run findings report is non-empty: retaining it
	// would discard actionable finding IDs and repeatedly invoke enrichment.
	if !issues.Has(issueDraftFindingLinkage) || !proposalDraftClaimsNoStructuredFindings(task) {
		return false
	}
	return true
}

func proposalDraftClaimsNoStructuredFindings(task acpruntime.Task) bool {
	root := filepath.Clean(strings.TrimSpace(task.DraftFinalRoot))
	if root == "" || root == "." {
		return false
	}
	for _, output := range loadAllowedDraftOutputs(task) {
		rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		if rel != "proposal.md" && rel != "changelog.md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return false
		}
		lower := strings.ToLower(string(raw))
		if strings.Contains(lower, "no structured finding summary") ||
			strings.Contains(lower, "no structured findings") ||
			strings.Contains(lower, "no actionable finding was available") {
			return true
		}
	}
	return false
}

func proposalDraftHasRecoveryScaffold(task acpruntime.Task) bool {
	root := filepath.Clean(strings.TrimSpace(task.DraftFinalRoot))
	if root == "" || root == "." {
		return false
	}
	found := false
	for _, output := range loadAllowedDraftOutputs(task) {
		rel := filepath.ToSlash(filepath.Clean(strings.TrimSpace(output.Path)))
		if rel != "proposal.md" && rel != "changelog.md" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return false
		}
		lower := strings.ToLower(string(raw))
		if !strings.Contains(lower, "runtime draft recovery initialized") &&
			!strings.Contains(lower, "drafted required runtime artifacts") &&
			!strings.Contains(lower, "draft surface initialized") {
			return false
		}
		found = true
	}
	return found
}

type deterministicProposalFinding struct {
	id       string
	severity string
	related  string
	evidence string
}

// writeDeterministicProposalDraft writes a minimal, operator-facing proposal
// pair from the current findings markdown. It is deliberately limited to the
// two proposal outputs and is used only when the provider left bootstrap-only
// content behind. The normal draft validator remains the source of truth.
func writeDeterministicProposalDraft(task acpruntime.Task) error {
	findingsText, _ := readProposalFindingsMarkdown(task)
	findings := parseDeterministicProposalFindings(findingsText)
	shardLine := deterministicProposalShardCompleteness(task)

	var proposal strings.Builder
	proposal.WriteString("# Runtime Recommendations\n\n")
	proposal.WriteString("## Decision / recommended operator action\n")
	proposal.WriteString("The analysis findings require explicit follow-up actions before the proposal package is accepted.\n\n")
	proposal.WriteString("## Evidence used\n")
	proposal.WriteString("- reports/findings/findings.md\n")
	for _, finding := range findings {
		if finding.evidence != "" {
			proposal.WriteString("- ")
			proposal.WriteString(finding.evidence)
			proposal.WriteByte('\n')
		}
	}
	if shardLine != "" {
		proposal.WriteString("- ")
		proposal.WriteString(shardLine)
		proposal.WriteByte('\n')
	}
	proposal.WriteString("\n## Proposed changes or follow-up plan\n")
	if len(findings) == 0 {
		proposal.WriteString("- No actionable proposal evidence was present in the findings; keep the operator decision anchored to reports/findings/findings.md.\n")
	} else {
		proposal.WriteString("### Top Actionable Findings\n")
		for _, finding := range findings {
			severity := finding.severity
			if severity == "" {
				severity = "low"
			}
			affected := finding.related
			if affected == "" {
				affected = finding.evidence
			}
			if affected == "" {
				affected = "reports/findings/findings.md"
			}
			action := "monitor the documented gap and confirm ownership"
			if severity == "high" || severity == "medium" {
				action = "document an owner and acceptance evidence for the affected surface"
			}
			fmt.Fprintf(&proposal, "- Finding ID: `%s`; Severity: `%s`; Affected surface/path: %s; Recommended operator action: %s; Residual gap: implementation and operational ownership remain to be confirmed.\n", finding.id, severity, affected, action)
		}
	}
	proposal.WriteString("\n## Risks, gaps, and out-of-scope notes\n")
	proposal.WriteString("- The recommendations are limited to the evidence recorded in reports/findings/findings.md; implementation acceptance remains an operator decision.\n")
	if shardLine != "" {
		proposal.WriteString("- ")
		proposal.WriteString(shardLine)
		proposal.WriteByte('\n')
	}

	var changelog strings.Builder
	changelog.WriteString("# Runtime Proposal Changelog\n\n")
	changelog.WriteString("## Updated architecture/proposal surfaces\n")
	changelog.WriteString("- proposals/runtime-recommendations.md records evidence-linked follow-up actions.\n\n")
	changelog.WriteString("## Findings/proposals summary\n")
	if len(findings) == 0 {
		changelog.WriteString("- No actionable proposal evidence was present in the findings; no source change is approved by this artifact.\n")
	} else {
		for _, finding := range findings {
			fmt.Fprintf(&changelog, "- Finding ID: `%s` is linked to a documented follow-up action.\n", finding.id)
		}
	}
	changelog.WriteString("\n## Evidence index or citation references\n- reports/findings/findings.md\n")
	if shardLine != "" {
		changelog.WriteString("- ")
		changelog.WriteString(shardLine)
		changelog.WriteByte('\n')
	}
	changelog.WriteString("\n## Residual coverage gaps\n- Operational ownership and implementation acceptance require confirmation outside this proposal artifact.\n")

	root := filepath.Clean(strings.TrimSpace(task.DraftFinalRoot))
	if root == "" || root == "." {
		return errors.New("draft proposal root is empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "proposal.md"), []byte(proposal.String()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "changelog.md"), []byte(changelog.String()), 0o644)
}

func readProposalFindingsMarkdown(task acpruntime.Task) (string, bool) {
	root := filepath.Clean(strings.TrimSpace(task.DraftFinalRoot))
	if root == "" || root == "." {
		return "", false
	}
	candidates := []string{
		filepath.Join(root, "reports", "findings", "findings.md"),
		filepath.Join(root, "..", "final", "reports", "findings", "findings.md"),
		filepath.Join(root, "..", "..", "final", "reports", "findings", "findings.md"),
		filepath.Join(root, "..", "..", "..", "staging", "final", "reports", "findings", "findings.md"),
	}
	for _, candidate := range candidates {
		if raw, err := os.ReadFile(filepath.Clean(candidate)); err == nil {
			return string(raw), true
		}
	}
	return "", false
}

func parseDeterministicProposalFindings(markdown string) []deterministicProposalFinding {
	findings := []deterministicProposalFinding{}
	current := deterministicProposalFinding{}
	flush := func() {
		if strings.TrimSpace(current.id) != "" {
			findings = append(findings, current)
		}
		current = deterministicProposalFinding{}
	}
	for _, rawLine := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(rawLine)
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "- id:"):
			flush()
			current.id = deterministicProposalFieldValue(line[len("- id:"):])
		case strings.HasPrefix(lower, "- severity:"):
			current.severity = strings.ToLower(deterministicProposalFieldValue(line[len("- severity:"):]))
		case strings.HasPrefix(lower, "- related ids:"):
			current.related = deterministicProposalFieldValue(line[len("- related ids:"):])
		case strings.HasPrefix(lower, "- evidence:"):
			current.evidence = deterministicProposalFieldValue(line[len("- evidence:"):])
		}
	}
	flush()
	seen := map[string]struct{}{}
	unique := findings[:0]
	for _, finding := range findings {
		if _, ok := seen[finding.id]; ok {
			continue
		}
		seen[finding.id] = struct{}{}
		unique = append(unique, finding)
	}
	return unique
}

func deterministicProposalFieldValue(value string) string {
	value = strings.TrimSpace(value)
	if start := strings.Index(value, "`"); start >= 0 {
		if end := strings.Index(value[start+1:], "`"); end >= 0 {
			return strings.TrimSpace(value[start+1 : start+1+end])
		}
	}
	value = strings.Trim(value, "` \t:;,")
	return strings.TrimSpace(value)
}

func deterministicProposalShardCompleteness(task acpruntime.Task) string {
	root := filepath.Clean(filepath.Join(strings.TrimSpace(task.DraftFinalRoot), "..", "..", "..", ".."))
	pattern := filepath.Join(root, strings.TrimSpace(task.RunID)+"-*-step1-collect-shard-summary-*.json")
	matches, _ := filepath.Glob(pattern)
	for _, match := range matches {
		raw, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		var summary struct {
			Items []struct {
				Status string `json:"status"`
			} `json:"items"`
		}
		if json.Unmarshal(raw, &summary) != nil || len(summary.Items) == 0 {
			continue
		}
		failed, incomplete := 0, 0
		for _, item := range summary.Items {
			switch strings.ToLower(strings.TrimSpace(item.Status)) {
			case "succeeded":
			case "failed":
				failed++
			default:
				incomplete++
			}
		}
		return fmt.Sprintf("Shard completeness: %d/%d succeeded; failed=%d incomplete=%d", len(summary.Items)-failed-incomplete, len(summary.Items), failed, incomplete)
	}
	return ""
}

func draftRepairEnrichmentStage(stage string, err error) string {
	suffix := "draft_semantic"
	if isDraftBootstrapOnlyValidationFailure(err) {
		suffix = "bootstrap_only"
	}
	base := strings.TrimSpace(stage)
	if base == "" {
		base = "contract"
	}
	return base + "_" + suffix
}

func runFocusedArtifactRepairCommand(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, baseResult acpruntime.Result, buildSpec func() (CommandSpec, error)) (acpruntime.Result, error, error) {
	return runFocusedArtifactRepairCommandWithPolicy(ctx, task, adapter, baseResult, buildSpec, nil)
}

func runCollectArtifactPairRepairCommand(ctx context.Context, task acpruntime.Task, adapter ProviderAdapter, baseResult acpruntime.Result, buildSpec func() (CommandSpec, error)) (acpruntime.Result, error, error) {
	spec, err := buildSpec()
	if err != nil {
		return acpruntime.Result{}, nil, classifyCommandFailure(adapter, task, baseResult, err)
	}
	repairPolicy := normalizeActivityPolicy(adapter.ActivityPolicy(task))
	repairPolicy.MonitorArtifacts = true
	repairPolicy.MonitorPreArtifact = true
	repairPolicy = collectArtifactPairRepairActivityPolicy(repairPolicy)
	repairResult, repairErr := runCommandSpecWithTransition(ctx, task, spec, repairPolicy, "focused_repair")
	return repairResult, repairErr, nil
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
	repairResult, repairErr := runCommandSpecWithTransition(ctx, task, spec, repairPolicy, "focused_repair")
	return repairResult, repairErr, nil
}

func collectArtifactPairRepairActivityPolicy(policy ActivityPolicy) ActivityPolicy {
	policy.FreshArtifactMutationAfter = time.Now().UTC().Add(-time.Millisecond)
	if policy.PreArtifactStallWindow < defaultCollectRepairWindow {
		policy.PreArtifactStallWindow = defaultCollectRepairWindow
	}
	if policy.PreArtifactWallClockWindow <= 0 {
		policy.PreArtifactWallClockWindow = defaultCollectRepairWindow
	}
	if policy.PostArtifactStallWindow < defaultCollectRepairWindow {
		policy.PostArtifactStallWindow = defaultCollectRepairWindow
	}
	if policy.PartialArtifactStallWindow < defaultCollectRepairWindow {
		policy.PartialArtifactStallWindow = defaultCollectRepairWindow
	}
	if policy.ValidArtifactStopWindow <= 0 {
		policy.ValidArtifactStopWindow = defaultRepairValidStopWindow
	}
	return policy
}

func focusedRepairActivityPolicy(base ActivityPolicy, monitorPreArtifact bool) ActivityPolicy {
	repairPolicy := normalizeActivityPolicy(base)
	repairPolicy.MonitorArtifacts = true
	repairPolicy.MonitorPreArtifact = monitorPreArtifact
	repairPolicy.FreshArtifactMutationAfter = time.Now().UTC().Add(-time.Millisecond)
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
