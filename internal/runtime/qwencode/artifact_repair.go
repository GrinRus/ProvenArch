package qwencode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/taskresultcompat"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

func validateRuntimeDraftArtifactsAtWriteRoot(task acpruntime.Task) (runtimedrafts.Manifest, []byte, error) {
	return runtimedrafts.ValidateRequiredManifest(
		task.WriteRoot,
		task.DraftFinalRoot,
		task.RunID,
		task.StepID,
		task.StepContract,
		task.ExpectedArtifacts,
	)
}

func reconcileRuntimeDraftOutputsAtWriteRoot(task acpruntime.Task, current acpruntime.Result) (acpruntime.Result, bool, error) {
	if !runtimedrafts.IsDraftStep(task.StepID) {
		return current, false, nil
	}
	manifestFile := runtimedrafts.ManifestFileForStep(task.StepID)
	if strings.TrimSpace(task.WriteRoot) == "" || strings.TrimSpace(task.DraftFinalRoot) == "" || strings.TrimSpace(manifestFile) == "" {
		return current, false, nil
	}
	manifest, _, err := runtimedrafts.Load(task.WriteRoot, manifestFile)
	if err != nil {
		return current, false, err
	}
	if err := runtimedrafts.ValidateManifestForTask(manifest, task.RunID, task.StepID, task.StepContract); err != nil {
		return current, false, err
	}
	changed, err := runtimedrafts.ReconcileOutputsAtDraftRoot(task.DraftFinalRoot, manifest)
	if err != nil || !changed {
		return current, changed, err
	}
	normalized, _, err := taskresultcompat.NormalizeResult(task, current)
	if err != nil {
		return current, false, err
	}
	return normalized, true, nil
}

func maybeRepairRuntimeDraftArtifacts(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
) (acpruntime.Result, error) {
	if !runtimedrafts.IsDraftStep(task.StepID) {
		return current, nil
	}
	if repaired, changed, err := reconcileRuntimeDraftOutputsAtWriteRoot(task, current); err == nil && changed {
		if _, _, validationErr := validateRuntimeDraftArtifactsAtWriteRoot(task); validationErr == nil {
			return repaired, nil
		}
		current = repaired
	}
	if _, _, err := validateRuntimeDraftArtifactsAtWriteRoot(task); err == nil {
		return current, nil
	} else {
		initialProblem := compactRetryHint(err.Error())
		repairArgs := buildDraftRepairQwenArgs(
			task,
			buildPromptWithModeAndHints(taskPayload, promptRetryDraftArtifact, buildDraftArtifactRepairHints(task, err)),
		)
		repaired, repairParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, repairArgs, runQwenOptions{})
		if runErr != nil {
			return acpruntime.Result{}, wrapRuntimeDraftContractFailure(
				task,
				"draft_repair.exec",
				repaired,
				fmt.Sprintf("runtime required draft artifact repair attempt failed after initial validation error %s: %v", initialProblem, runErr),
				runErr,
			)
		}
		if parseErr != nil {
			return acpruntime.Result{}, wrapQwenParseFailure(
				task,
				"draft_repair."+strings.TrimSpace(repairParseStage),
				parseErr,
				repaired,
				fmt.Sprintf("headless provider %q returned invalid runtime draft artifact repair result after %s", acpruntime.ProviderQwenCode, initialProblem),
			)
		}
		repaired, _, _ = reconcileRuntimeDraftOutputsAtWriteRoot(task, repaired)
		if _, _, repairErr := validateRuntimeDraftArtifactsAtWriteRoot(task); repairErr != nil {
			return acpruntime.Result{}, wrapRuntimeDraftContractFailure(
				task,
				"draft_repair.contract",
				repaired,
				fmt.Sprintf("runtime required draft artifacts remained invalid after one repair attempt: initial=%s; validation=%v", initialProblem, repairErr),
				repairErr,
			)
		}
		return repaired, nil
	}
}

func maybeRepairCollectArtifacts(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
) (acpruntime.Result, collectRepairAttempt, error) {
	if task.StepID != "init.step1.collect" && task.StepID != "refresh.step1.collect" {
		return current, collectRepairAttempt{}, nil
	}
	if strings.TrimSpace(task.WriteRoot) == "" {
		return current, collectRepairAttempt{}, nil
	}

	_ = artifactquality.EnsureCanonicalCollectManifest(task, current.TaskResult)
	assessment, err := assessRetryManifestAtWriteRoot(task.WriteRoot)
	if err == nil && assessment.Rich {
		return current, collectRepairAttempt{ManifestStateBeforeRetry: "rich"}, nil
	}
	initialProblem := artifactquality.DescribeAssessmentProblem(assessment, err)
	beforeRepairState := currentCollectManifestState(task.WriteRoot)
	attempt := collectRepairAttempt{
		Attempted:                true,
		ManifestStateBeforeRetry: beforeRepairState,
		Diagnostic:               buildCollectRepairDiagnostic(task.WriteRoot),
	}

	snapshot, err := artifactquality.SnapshotWriteRoot(task.WriteRoot)
	if err != nil {
		return acpruntime.Result{}, attempt, wrapArtifactContractFailure(
			task,
			"artifact_repair.snapshot",
			current,
			fmt.Sprintf("collect artifacts require repair (%s), but write_root snapshot failed", initialProblem),
			err,
		)
	}
	defer func() {
		_ = snapshot.Cleanup()
	}()

	repairArgs := buildRetryQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryArtifact, buildArtifactRepairHints(initialProblem)))
	repaired, repairParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, repairArgs, runQwenOptions{})
	if runErr != nil {
		_ = snapshot.Restore()
		unavailableMessage := buildUnavailableFailureMessage(task, runErr, repaired)
		return acpruntime.Result{}, attempt, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: artifact repair retry failed after %s: %s", ErrRunnerUnavailable, initialProblem, unavailableMessage),
			repaired.Stdout,
			repaired.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		if contractErr := validateCollectManifestContractAtWriteRoot(task.WriteRoot); contractErr != nil {
			_ = snapshot.Restore()
			return acpruntime.Result{}, attempt, wrapArtifactContractFailure(
				task,
				"artifact_repair.contract",
				repaired,
				fmt.Sprintf("collect artifact repair returned invalid taskresult and left contract-invalid shard-pack-manifest.json: parse=%v; validation=%v", parseErr, contractErr),
				errors.Join(parseErr, contractErr),
			)
		}
		_ = snapshot.Restore()
		return acpruntime.Result{}, attempt, wrapQwenParseFailure(
			task,
			"artifact_repair."+repairParseStage,
			parseErr,
			repaired,
			fmt.Sprintf("headless provider %q returned invalid collect artifact repair result after %s", acpruntime.ProviderQwenCode, initialProblem),
		)
	}

	_ = artifactquality.EnsureCanonicalCollectManifest(task, repaired.TaskResult)
	if contractErr := validateCollectManifestContractAtWriteRoot(task.WriteRoot); contractErr != nil {
		_ = snapshot.Restore()
		return acpruntime.Result{}, attempt, wrapArtifactContractFailure(
			task,
			"artifact_repair.contract",
			repaired,
			fmt.Sprintf("collect artifacts remained invalid after one repair attempt: initial=%s; validation=%v", initialProblem, contractErr),
			contractErr,
		)
	}
	repairedAssessment, err := assessRetryManifestAtWriteRoot(task.WriteRoot)
	if err != nil || !repairedAssessment.Rich {
		_ = snapshot.Restore()
		repairedProblem := artifactquality.DescribeAssessmentProblem(repairedAssessment, err)
		return acpruntime.Result{}, attempt, wrapArtifactContractFailure(
			task,
			"artifact_repair.contract",
			repaired,
			fmt.Sprintf("collect artifacts remained invalid after one repair attempt: initial=%s; repaired=%s", initialProblem, repairedProblem),
			err,
		)
	}
	return repaired, attempt, nil
}

func recoverCollectArtifactsAfterStall(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
	stalled collectStallError,
) (acpruntime.Result, error) {
	initialProblem := errCollectStalledAfterArtifacts.Error()
	emitDiagnostic(task, "runtime task retry scheduled", stalled.Diagnostic.fields(task))
	retryContext := retryDiagnosticContext{
		LastStallPhase:           collectStallPhasePostArtifact,
		ManifestStateBeforeRetry: stalled.Diagnostic.ManifestState,
		AuthoredFileCount:        stalled.Diagnostic.AuthoredFileCount,
		LastPipeActivity:         stalled.Diagnostic.LastPipeActivity,
		LastWriteRootMutation:    stalled.Diagnostic.LastWriteRootMutation,
	}

	snapshot, err := artifactquality.SnapshotWriteRoot(task.WriteRoot)
	if err != nil {
		message := buildFailureMessage(task, "stall_snapshot", fmt.Errorf("%w: snapshot write_root: %v", errCollectStalledAfterArtifacts, err), current)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", err.Error(), stalled.Diagnostic.ManifestState))
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q collect_stalled_after_artifacts and artifact recovery setup failed: %s", acpruntime.ProviderQwenCode, message),
			current.Stdout,
			current.Stderr,
			err,
		)
	}
	defer func() {
		_ = snapshot.Cleanup()
	}()

	repairArgs := buildRetryQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryArtifact, buildArtifactRepairHints(initialProblem)))
	repaired, repairParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, repairArgs, runQwenOptions{})
	if runErr != nil {
		_ = snapshot.Restore()
		failureResult := selectFailureResult(repaired, current)
		failureMessage := buildFailureMessage(task, "stall_repair.exec", fmt.Errorf("%w: %v", errCollectStalledAfterArtifacts, runErr), failureResult)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", runErr.Error(), stalled.Diagnostic.ManifestState))
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q collect_stalled_after_artifacts and repair retry failed: %s", acpruntime.ProviderQwenCode, failureMessage),
			failureResult.Stdout,
			failureResult.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		if contractErr := validateCollectManifestContractAtWriteRoot(task.WriteRoot); contractErr != nil {
			_ = snapshot.Restore()
			emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", contractErr.Error(), currentCollectManifestState(task.WriteRoot)))
			return acpruntime.Result{}, wrapArtifactContractFailure(
				task,
				"stall_repair.contract",
				repaired,
				fmt.Sprintf("collect_stalled_after_artifacts repair returned invalid taskresult and left contract-invalid shard-pack-manifest.json: parse=%v; validation=%v", parseErr, contractErr),
				errors.Join(parseErr, contractErr),
			)
		}
		_ = snapshot.Restore()
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", parseErr.Error(), stalled.Diagnostic.ManifestState))
		return acpruntime.Result{}, wrapQwenParseFailure(
			task,
			"stall_repair."+strings.TrimSpace(repairParseStage),
			parseErr,
			repaired,
			fmt.Sprintf("headless provider %q collect_stalled_after_artifacts and repair retry returned invalid taskresult", acpruntime.ProviderQwenCode),
		)
	}

	_ = artifactquality.EnsureCanonicalCollectManifest(task, repaired.TaskResult)
	if contractErr := validateCollectManifestContractAtWriteRoot(task.WriteRoot); contractErr != nil {
		_ = snapshot.Restore()
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", contractErr.Error(), currentCollectManifestState(task.WriteRoot)))
		return acpruntime.Result{}, wrapArtifactContractFailure(
			task,
			"stall_repair.contract",
			repaired,
			fmt.Sprintf("collect_stalled_after_artifacts and collect artifacts remained invalid after repair: validation=%v", contractErr),
			errors.Join(errCollectStalledAfterArtifacts, contractErr),
		)
	}
	repairedAssessment, assessErr := assessRetryManifestAtWriteRoot(task.WriteRoot)
	if assessErr != nil || !repairedAssessment.Rich {
		_ = snapshot.Restore()
		repairedProblem := artifactquality.DescribeAssessmentProblem(repairedAssessment, assessErr)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", repairedProblem, stalled.Diagnostic.ManifestState))
		return acpruntime.Result{}, wrapArtifactContractFailure(
			task,
			"stall_repair.contract",
			repaired,
			fmt.Sprintf("collect_stalled_after_artifacts and collect artifacts remained invalid after repair: repaired=%s", repairedProblem),
			errors.Join(errCollectStalledAfterArtifacts, assessErr),
		)
	}

	emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "succeeded", "", "rich"))
	return repaired, nil
}

func recoverCollectBeforeArtifactsAfterStall(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
	stalled collectStallError,
) (acpruntime.Result, error) {
	emitDiagnostic(task, "runtime task retry scheduled", stalled.Diagnostic.fields(task))
	retryContext := retryDiagnosticContext{
		LastStallPhase:           stalled.Diagnostic.StallPhase,
		ManifestStateBeforeRetry: stalled.Diagnostic.ManifestState,
		AuthoredFileCount:        stalled.Diagnostic.AuthoredFileCount,
		LastPipeActivity:         stalled.Diagnostic.LastPipeActivity,
		LastWriteRootMutation:    stalled.Diagnostic.LastWriteRootMutation,
	}

	retryArgs := buildRetryQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryCollectFresh, buildCollectFreshRetryHints(task)))
	retried, retryParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, retryArgs, runQwenOptions{
		EnableCollectStallMonitor: true,
	})
	if runErr != nil {
		var retryStalled collectStallError
		if errors.As(runErr, &retryStalled) {
			retryContext.absorbDiagnostic(retryStalled.Diagnostic)
			if errors.Is(retryStalled, errCollectStalledAfterArtifacts) {
				return recoverCollectArtifactsAfterStall(ctx, task, taskPayload, command, retried, retryStalled)
			}
		}
		failureResult := selectFailureResult(retried, current)
		failureMessage := buildFailureMessage(task, "stall_retry.exec", fmt.Errorf("%w: %v", errCollectStalledBeforeArtifacts, runErr), failureResult)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", runErr.Error(), currentCollectManifestState(task.WriteRoot)))
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q collect_stalled_before_artifacts and forced retry failed: %s", acpruntime.ProviderQwenCode, failureMessage),
			failureResult.Stdout,
			failureResult.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		if contractErr := validateCollectManifestContractAtWriteRoot(task.WriteRoot); contractErr != nil {
			emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", contractErr.Error(), currentCollectManifestState(task.WriteRoot)))
			return acpruntime.Result{}, wrapArtifactContractFailure(
				task,
				"stall_retry.contract",
				retried,
				fmt.Sprintf("collect_stalled_before_artifacts forced retry returned invalid taskresult and left contract-invalid shard-pack-manifest.json: parse=%v; validation=%v", parseErr, contractErr),
				errors.Join(parseErr, contractErr),
			)
		}
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", parseErr.Error(), currentCollectManifestState(task.WriteRoot)))
		return acpruntime.Result{}, wrapQwenParseFailure(
			task,
			"stall_retry."+strings.TrimSpace(retryParseStage),
			parseErr,
			retried,
			fmt.Sprintf("headless provider %q collect_stalled_before_artifacts and forced retry returned invalid taskresult", acpruntime.ProviderQwenCode),
		)
	}

	finalResult, repairAttempt, err := maybeRepairCollectArtifacts(ctx, task, taskPayload, command, retried)
	if repairAttempt.Attempted {
		retryContext.absorbDiagnostic(repairAttempt.Diagnostic)
		if state := strings.TrimSpace(repairAttempt.ManifestStateBeforeRetry); state != "" {
			retryContext.ManifestStateBeforeRetry = state
		}
	}
	if err != nil {
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", err.Error(), currentCollectManifestState(task.WriteRoot)))
		return acpruntime.Result{}, err
	}

	emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "succeeded", "", currentCollectManifestState(task.WriteRoot)))
	return finalResult, nil
}

func recoverDraftArtifactsAfterStall(
	ctx context.Context,
	task acpruntime.Task,
	taskPayload []byte,
	command string,
	current acpruntime.Result,
	stalled collectStallError,
) (acpruntime.Result, error) {
	emitDiagnostic(task, "runtime task retry scheduled", stalled.Diagnostic.fields(task))
	retryContext := retryDiagnosticContext{
		LastStallPhase:           collectStallPhasePostArtifact,
		ManifestStateBeforeRetry: stalled.Diagnostic.ManifestState,
		AuthoredFileCount:        stalled.Diagnostic.AuthoredFileCount,
		LastPipeActivity:         stalled.Diagnostic.LastPipeActivity,
		LastWriteRootMutation:    stalled.Diagnostic.LastWriteRootMutation,
	}

	retryHints := append([]string{
		`- Previous attempt stalled after writing draft artifacts; do NOT continue repository exploration on this retry.`,
		`- Reuse the existing draft manifest and draft files from write_root/draft_final_root, fix only draft contract drift, and return the final TaskResult JSON immediately.`,
	}, buildDraftArtifactRepairHints(task, errDraftStalledAfterArtifacts)...)
	repairArgs := buildDraftRepairQwenArgs(task, buildPromptWithModeAndHints(taskPayload, promptRetryDraftArtifact, retryHints))
	repaired, repairParseStage, parseErr, runErr := runQwenCommand(ctx, task, command, repairArgs, runQwenOptions{})
	if runErr != nil {
		failureResult := selectFailureResult(repaired, current)
		failureMessage := buildFailureMessage(task, "draft_stall_repair.exec", fmt.Errorf("%w: %v", errDraftStalledAfterArtifacts, runErr), failureResult)
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", runErr.Error(), currentDraftManifestState(task)))
		return acpruntime.Result{}, acpruntime.WrapRunnerErrorWithOutput(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q draft_stalled_after_artifacts and repair retry failed: %s", acpruntime.ProviderQwenCode, failureMessage),
			failureResult.Stdout,
			failureResult.Stderr,
			runErr,
		)
	}
	if parseErr != nil {
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", parseErr.Error(), currentDraftManifestState(task)))
		return acpruntime.Result{}, wrapQwenParseFailure(
			task,
			"draft_stall_repair."+strings.TrimSpace(repairParseStage),
			parseErr,
			repaired,
			fmt.Sprintf("headless provider %q draft_stalled_after_artifacts and repair retry returned invalid taskresult", acpruntime.ProviderQwenCode),
		)
	}
	repaired, _, _ = reconcileRuntimeDraftOutputsAtWriteRoot(task, repaired)
	if _, _, err := validateRuntimeDraftArtifactsAtWriteRoot(task); err != nil {
		emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "failed", err.Error(), currentDraftManifestState(task)))
		return acpruntime.Result{}, wrapRuntimeDraftContractFailure(
			task,
			"draft_stall_repair.contract",
			repaired,
			fmt.Sprintf("draft_stalled_after_artifacts and runtime draft artifacts remained invalid after repair: validation=%v", err),
			errors.Join(errDraftStalledAfterArtifacts, err),
		)
	}
	emitDiagnostic(task, "runtime task retry completed", retryDiagnosticFields(task, stalled, retryContext, "succeeded", "", currentDraftManifestState(task)))
	return repaired, nil
}

func validateCollectManifestContractAtWriteRoot(writeRoot string) error {
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is empty")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(writeRoot), "shard-pack-manifest.json"))
	if err != nil {
		return fmt.Errorf("read shard pack manifest: %w", err)
	}
	if _, err := contracts.ParseShardPackManifest(raw); err != nil {
		return err
	}
	return nil
}

func buildCollectRepairDiagnostic(writeRoot string) collectStallDiagnostic {
	snapshot, err := scanCollectWriteRoot(writeRoot)
	if err != nil {
		return collectStallDiagnostic{}
	}
	return collectStallDiagnostic{
		ManifestState:         strings.TrimSpace(snapshot.ManifestState),
		AuthoredFileCount:     snapshot.AuthoredFileCount,
		LastWriteRootMutation: snapshot.LastMutation.UTC(),
	}
}
