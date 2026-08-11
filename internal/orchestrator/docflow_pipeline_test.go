package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestDocFirstStageThenPromoteFlow(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docflowCustomProposalRunner{}),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	finalIndex := readRunFinalRunIndex(t, ws.Path, info.RunID)
	customDocs := collectCustomProposalDocs(finalIndex)
	if len(customDocs) == 0 {
		t.Fatalf("expected custom docs-first proposal docs in final run index")
	}

	for _, document := range customDocs {
		stagedContent, stagedErr := os.ReadFile(filepath.Join(ws.Path, filepath.FromSlash(document.StagedPath)))
		if stagedErr != nil {
			t.Fatalf("read staged proposal %q: %v", document.StagedPath, stagedErr)
		}
		canonicalContent, canonicalErr := os.ReadFile(filepath.Join(ws.Path, filepath.FromSlash(document.CanonicalPath)))
		if canonicalErr != nil {
			t.Fatalf("read canonical proposal %q: %v", document.CanonicalPath, canonicalErr)
		}
		if string(stagedContent) != string(canonicalContent) {
			t.Fatalf("canonical proposal %q does not match staged content", document.CanonicalPath)
		}
	}

	manifestPaths, globErr := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", info.RunID, "staging", "shards", "*", "shard-pack-manifest.json"))
	if globErr != nil {
		t.Fatalf("glob shard manifests: %v", globErr)
	}
	if len(manifestPaths) == 0 {
		t.Fatalf("expected shard manifests in staged shard roots")
	}

	verdict := readRunValidatorVerdict(t, ws.Path, info.RunID)
	if verdict.Verdict != "PASS" {
		t.Fatalf("expected PASS verdict, got %q", verdict.Verdict)
	}

	if len(finalIndex.Semantic.Entities) == 0 {
		t.Fatalf("expected semantic entities in final run index")
	}
	entityFiles, entityGlobErr := filepath.Glob(filepath.Join(ws.Path, "model", "entities", "*.yaml"))
	if entityGlobErr != nil {
		t.Fatalf("glob model entities: %v", entityGlobErr)
	}
	if len(entityFiles) == 0 {
		t.Fatalf("expected derived model/entities files after promotion")
	}
}

func TestProposalRetryHydratesValidatedParentAndPromotesChild(t *testing.T) {
	ws := createWorkspace(t)
	service := NewService(WithRunner(docflowCustomProposalRunner{}), WithClock(func() time.Time { return time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC) }))
	parent, _, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineInit, NonInteractive: true})
	if err != nil || parent.Status != RunStatusSucceeded {
		t.Fatalf("prepare parent: %#v, %v", parent, err)
	}
	child, artifacts, err := service.Run(context.Background(), RunRequest{Workspace: ws, Pipeline: PipelineInit, NonInteractive: true, ResumeFromStep: "init.step4.proposals", RetryParentRunID: parent.RunID, RetryReason: "test", RetryRequestedStep: "init.step4.proposals", RetryReusedInputs: []string{"init.step0.constitution", "init.step1.collect", "init.step2.asis_docs", "init.step3.findings"}})
	if err != nil || child.Status != RunStatusSucceeded {
		t.Fatalf("proposal retry: %#v, %v", child, err)
	}
	if child.Retry == nil || child.Retry.ParentRunID != parent.RunID || child.Retry.EffectiveStartStep != "init.step4.proposals" {
		t.Fatalf("child lineage missing: %#v", child.Retry)
	}
	if index := readRunFinalRunIndex(t, ws.Path, child.RunID); index.RunID != child.RunID {
		t.Fatalf("child final index was not rebound: %#v", index)
	}
	foundSnapshot := false
	for _, artifact := range artifacts {
		if artifact.Kind == architectureSnapshotKind {
			foundSnapshot = true
		}
	}
	if !foundSnapshot {
		t.Fatal("successful retry did not pass the normal promotion snapshot boundary")
	}
}

func TestDocFirstValidatorFailBlocksPromotionInBestEffort(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docflowFailingValidatorRunner{}),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected run error when validator verdict is FAIL")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode != "" {
		t.Fatalf("expected empty error_code for validator verdict failure, got %q", info.ErrorCode)
	}

	finalIndex := readRunFinalRunIndex(t, ws.Path, info.RunID)
	customDocs := collectCustomProposalDocs(finalIndex)
	if len(customDocs) == 0 {
		t.Fatalf("expected custom docs-first proposal docs in final run index")
	}

	for _, document := range customDocs {
		if _, stagedErr := os.Stat(filepath.Join(ws.Path, filepath.FromSlash(document.StagedPath))); stagedErr != nil {
			t.Fatalf("expected staged proposal %q: %v", document.StagedPath, stagedErr)
		}
		_, canonicalErr := os.Stat(filepath.Join(ws.Path, filepath.FromSlash(document.CanonicalPath)))
		if canonicalErr == nil {
			t.Fatalf("expected promotion to be blocked for %q", document.CanonicalPath)
		}
		if !errors.Is(canonicalErr, os.ErrNotExist) {
			t.Fatalf("unexpected canonical proposal stat error for %q: %v", document.CanonicalPath, canonicalErr)
		}
	}

	verdict := readRunValidatorVerdict(t, ws.Path, info.RunID)
	if verdict.Verdict != "FAIL" {
		t.Fatalf("expected FAIL verdict, got %q", verdict.Verdict)
	}
}

func TestDocFirstOwnerGapOnlyValidatorFailDowngradesToPass(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docflowOwnerGapValidatorRunner{}),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("expected owner-gap-only validator residual to be reconciled to PASS, got %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	providerVerdict := readRunValidatorVerdict(t, ws.Path, info.RunID)
	if providerVerdict.Verdict != "FAIL" {
		t.Fatalf("expected immutable provider FAIL draft, got %q", providerVerdict.Verdict)
	}
	effective := readRunEffectiveVerdict(t, ws.Path, info.RunID)
	if effective.Verdict != "PASS" {
		t.Fatalf("expected effective PASS verdict, got %q", effective.Verdict)
	}
	if len(effective.Findings) == 0 || len(effective.Questions) == 0 {
		t.Fatalf("expected owner-gap findings/questions to remain visible, got %+v", effective)
	}
}

func TestDocFirstSourceEvidenceValidatorFailRemainsAdvisory(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docflowEvidenceAdvisoryValidatorRunner{}),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 7, 29, 9, 12, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("expected source-evidence observation to remain advisory, got %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	providerVerdict := readRunValidatorVerdict(t, ws.Path, info.RunID)
	if providerVerdict.Verdict != "FAIL" {
		t.Fatalf("expected immutable provider FAIL draft, got %q", providerVerdict.Verdict)
	}
	effective := readRunEffectiveVerdict(t, ws.Path, info.RunID)
	if effective.Verdict != "PASS" {
		t.Fatalf("expected effective PASS verdict, got %q", effective.Verdict)
	}
	if len(effective.AdvisoryIssues) != 1 || effective.AdvisoryIssues[0].Severity != "warning" {
		t.Fatalf("expected source-evidence issue to remain advisory warning, got %+v", effective.AdvisoryIssues)
	}
}

func TestDocFirstAssemblyMaterializesRequiredCanonicalLiveDocsWhenAuthoredPrefixesArePartial(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docflowPartialCanonicalRunner{}),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	finalIndex := readRunFinalRunIndex(t, ws.Path, info.RunID)
	required := map[string]bool{
		"reports/as-is/overview.md":           false,
		"reports/coverage/summary.md":         false,
		"reports/coverage/open-questions.md":  false,
		"reports/findings/findings.md":        false,
		"reports/as-is/services/authored.md":  false,
		"reports/coverage/custom-authored.md": false,
	}
	for _, document := range finalIndex.CanonicalDocuments {
		if _, ok := required[document.CanonicalPath]; ok {
			required[document.CanonicalPath] = true
		}
	}
	for path, present := range required {
		if !present {
			t.Fatalf("expected final run index to include %s", path)
		}
	}
}

func TestDocFlowSkipsAsIsRuntimeWhenCollectEvidenceIsUnusable(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	runner := &collectFailureRunner{}
	service := NewService(
		WithRunner(runner),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected run error when all collect shards fail")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if runner.asIsInvocationCount() != 0 {
		t.Fatalf("expected as-is runtime to be skipped, got %d invocations", runner.asIsInvocationCount())
	}
	if runner.proposalsInvocationCount() != 0 {
		t.Fatalf("expected proposals runtime to be skipped, got %d invocations", runner.proposalsInvocationCount())
	}
	if !strings.Contains(info.Error, "step1.collect") {
		t.Fatalf("expected collect failure summary, got %q", info.Error)
	}

	quality := readRunQuality(t, ws.Path, info.RunID)
	if quality.EvidenceState.ReportMode != "incomplete" {
		t.Fatalf("expected incomplete report mode, got %#v", quality.EvidenceState)
	}
	if !containsString(quality.EvidenceState.Reasons, "collect_all_shards_failed") {
		t.Fatalf("expected collect_all_shards_failed reason, got %#v", quality.EvidenceState.Reasons)
	}
	if !containsString(quality.EvidenceState.Reasons, "asis_docs_skipped_due_to_unusable_collect") {
		t.Fatalf("expected as-is skip reason, got %#v", quality.EvidenceState.Reasons)
	}
}

func TestDocFlowSkipsDownstreamRuntimeWhenCollectEvidenceIsPartial(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	runner := &collectPartialFailureRunner{}
	service := NewService(
		WithRunner(runner),
		WithExecutionOverrides(acpruntime.ExecutionOverrides{
			FailurePolicy: strPtr(acpruntime.ExecutionFailurePolicyBestEffort),
		}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected run error when collect evidence is partial")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode != runErrorCodePartialFailed {
		t.Fatalf("expected partial failure error code, got %q (%s)", info.ErrorCode, info.Error)
	}
	if runner.asIsInvocationCount() != 0 {
		t.Fatalf("expected as-is runtime to be skipped, got %d invocations", runner.asIsInvocationCount())
	}
	if runner.proposalsInvocationCount() != 0 {
		t.Fatalf("expected proposals runtime to be skipped, got %d invocations", runner.proposalsInvocationCount())
	}
	if !strings.Contains(info.Error, "step1.collect") {
		t.Fatalf("expected collect partial failure summary, got %q", info.Error)
	}

	quality := readRunQuality(t, ws.Path, info.RunID)
	if quality.EvidenceState.ReportMode != "incomplete" {
		t.Fatalf("expected incomplete report mode, got %#v", quality.EvidenceState)
	}
	if got := quality.EvidenceState.Collect.Status; got != "partial" {
		t.Fatalf("expected partial collect status, got %#v", quality.EvidenceState.Collect)
	}
	if !containsString(quality.EvidenceState.Reasons, "collect_partial_shard_failures") {
		t.Fatalf("expected collect_partial_shard_failures reason, got %#v", quality.EvidenceState.Reasons)
	}
	if !containsString(quality.EvidenceState.Reasons, "asis_docs_skipped_due_to_partial_collect") {
		t.Fatalf("expected as-is partial skip reason, got %#v", quality.EvidenceState.Reasons)
	}
	if !containsString(quality.EvidenceState.Reasons, "findings_skipped_due_to_partial_collect") {
		t.Fatalf("expected findings partial skip reason, got %#v", quality.EvidenceState.Reasons)
	}
	if !containsString(quality.EvidenceState.Reasons, "proposals_skipped_due_to_partial_collect") {
		t.Fatalf("expected proposals partial skip reason, got %#v", quality.EvidenceState.Reasons)
	}
}

func TestManagedFakeRuntimeAutoApprovesEnvelopePermissionRequests(t *testing.T) {
	t.Parallel()

	ws := createManagedPermissionWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithClock(func() time.Time {
			return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run managed fake pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if len(info.PendingPermissions) != 0 {
		t.Fatalf("expected no pending permissions for auto-approved fixture requests, got %+v", info.PendingPermissions)
	}

	logs := collectRunLogsForTest(t, service, info.RunID)
	autoApproved := 0
	for _, entry := range logs {
		if entry.Message != "runtime permission decision" {
			continue
		}
		if entry.Fields["decision"] == acpruntime.PermissionDecisionAutoApproved {
			autoApproved++
		}
		if entry.Fields["permissions_mode"] != acpruntime.PermissionModeManaged {
			t.Fatalf("expected managed permission diagnostics, got %+v", entry.Fields)
		}
	}
	if autoApproved == 0 {
		t.Fatalf("expected auto-approved permission decisions in run logs; sample=%s", summarizeLogSample(logs))
	}
}

func TestManagedRuntimeNeedsUserPermissionBecomesPendingFailure(t *testing.T) {
	t.Parallel()

	ws := createManagedPermissionWorkspace(t)
	service := NewService(
		WithHistoryWorkspace(ws),
		WithRunner(permissionRequiredRunner{}),
		WithClock(func() time.Time {
			return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected managed permission request to fail non-interactive run")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}
	if info.ErrorCode != string(acpruntime.ErrorCodePermissionRequired) {
		t.Fatalf("expected %s error code, got %q (%s)", acpruntime.ErrorCodePermissionRequired, info.ErrorCode, info.Error)
	}
	if len(info.PendingPermissions) != 1 {
		t.Fatalf("expected one pending permission request, got %+v", info.PendingPermissions)
	}
	if info.PendingPermissions[0].Action != "shell" || info.PendingPermissions[0].PathOrCommand != "npm install" {
		t.Fatalf("unexpected pending permission request: %+v", info.PendingPermissions[0])
	}
	if info.PendingPermissions[0].Decision == nil ||
		info.PendingPermissions[0].Decision.Decision != acpruntime.PermissionDecisionNeedsUser ||
		info.PendingPermissions[0].Decision.RuleID != "ask_unsafe_operation" {
		t.Fatalf("expected pending permission decision metadata, got %+v", info.PendingPermissions[0].Decision)
	}
	permissions, ok := service.GetRunPermissions(info.RunID)
	if !ok {
		t.Fatalf("expected permission requests endpoint backing data for %s", info.RunID)
	}
	if len(permissions) != 1 || permissions[0].RequestID != "perm-shell" {
		t.Fatalf("unexpected run permission requests: %+v", permissions)
	}
}

type docflowCustomProposalRunner struct{}

func (docflowCustomProposalRunner) Preflight(context.Context) error { return nil }

func (docflowCustomProposalRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	result, err := fakeruntime.Runner{}.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step1.collect") {
		if err := appendCustomProposalToManifest(task); err != nil {
			return acpruntime.Result{}, err
		}
	}
	return result, nil
}

type docflowFailingValidatorRunner struct{}

type docflowPartialCanonicalRunner struct{}

type docflowOwnerGapValidatorRunner struct{}

type docflowEvidenceAdvisoryValidatorRunner struct{}

type collectFailureRunner struct {
	mu               sync.Mutex
	asIsInvoked      int
	proposalsInvoked int
}

type collectPartialFailureRunner struct {
	mu               sync.Mutex
	collectFailed    bool
	asIsInvoked      int
	proposalsInvoked int
}

type permissionRequiredRunner struct{}

func (docflowFailingValidatorRunner) Preflight(context.Context) error  { return nil }
func (docflowPartialCanonicalRunner) Preflight(context.Context) error  { return nil }
func (docflowOwnerGapValidatorRunner) Preflight(context.Context) error { return nil }
func (docflowEvidenceAdvisoryValidatorRunner) Preflight(context.Context) error {
	return nil
}
func (*collectFailureRunner) Preflight(context.Context) error        { return nil }
func (*collectPartialFailureRunner) Preflight(context.Context) error { return nil }
func (permissionRequiredRunner) Preflight(context.Context) error     { return nil }

func (docflowFailingValidatorRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	result, err := docflowCustomProposalRunner{}.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step3.findings") {
		if err := overwriteValidatorVerdict(task, "FAIL"); err != nil {
			return acpruntime.Result{}, err
		}
	}
	return result, nil
}

func (docflowPartialCanonicalRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	result, err := fakeruntime.Runner{}.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step1.collect") {
		if err := appendPartialCanonicalDocsToManifest(task); err != nil {
			return acpruntime.Result{}, err
		}
	}
	return result, nil
}

func (docflowOwnerGapValidatorRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	result, err := docflowCustomProposalRunner{}.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step3.findings") {
		if err := overwriteOwnerGapValidatorVerdict(task); err != nil {
			return acpruntime.Result{}, err
		}
	}
	return result, nil
}

func (docflowEvidenceAdvisoryValidatorRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	result, err := docflowCustomProposalRunner{}.Run(ctx, task)
	if err != nil {
		return acpruntime.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step3.findings") {
		if err := overwriteEvidenceAdvisoryValidatorVerdict(task); err != nil {
			return acpruntime.Result{}, err
		}
	}
	return result, nil
}

func (r *collectFailureRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	switch task.StepID {
	case "init.step1.collect", "refresh.step1.collect":
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRuntimeContract,
			"collect manifest contract invalid",
			errors.New("shard pack manifest is invalid"),
		)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		r.mu.Lock()
		r.asIsInvoked++
		r.mu.Unlock()
		return acpruntime.Result{}, fmt.Errorf("unexpected as-is runtime invocation")
	case "init.step4.proposals", "refresh.step4.proposals":
		r.mu.Lock()
		r.proposalsInvoked++
		r.mu.Unlock()
		return acpruntime.Result{}, fmt.Errorf("unexpected proposals runtime invocation")
	default:
		return fakeruntime.Runner{}.Run(ctx, task)
	}
}

func (r *collectPartialFailureRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	switch task.StepID {
	case "init.step1.collect", "refresh.step1.collect":
		r.mu.Lock()
		if !r.collectFailed {
			r.collectFailed = true
			r.mu.Unlock()
			return acpruntime.Result{}, acpruntime.WrapRunnerError(
				acpruntime.ProviderClaudeCode,
				acpruntime.ErrorCodeRuntimeContract,
				"collect manifest contract invalid",
				errors.New("partial shard pack manifest is invalid"),
			)
		}
		r.mu.Unlock()
		return fakeruntime.Runner{}.Run(ctx, task)
	case "init.step2.asis_docs", "refresh.step2.asis_docs":
		r.mu.Lock()
		r.asIsInvoked++
		r.mu.Unlock()
		return acpruntime.Result{}, fmt.Errorf("unexpected as-is runtime invocation")
	case "init.step4.proposals", "refresh.step4.proposals":
		r.mu.Lock()
		r.proposalsInvoked++
		r.mu.Unlock()
		return acpruntime.Result{}, fmt.Errorf("unexpected proposals runtime invocation")
	default:
		return fakeruntime.Runner{}.Run(ctx, task)
	}
}

func (permissionRequiredRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if task.RuntimePermissions.Mode != acpruntime.PermissionModeManaged {
		return acpruntime.Result{}, fmt.Errorf("expected managed runtime permissions, got %q", task.RuntimePermissions.Mode)
	}
	if task.OnPermissionRequest == nil {
		return acpruntime.Result{}, fmt.Errorf("expected runtime permission hook")
	}
	decision := task.OnPermissionRequest(acpruntime.PermissionRequest{
		RequestID:     "perm-shell",
		Action:        "shell",
		PathOrCommand: "npm install",
		Reason:        "package install requires review",
	})
	if decision.Approved() {
		return fakeruntime.Runner{}.Run(ctx, task)
	}
	return acpruntime.Result{}, acpruntime.WrapRunnerError(
		acpruntime.Provider("fake"),
		acpruntime.ErrorCodePermissionRequired,
		"runtime permission required",
		fmt.Errorf("permission request %s resolved as %s", decision.RuleID, decision.Decision),
	)
}

func (r *collectFailureRunner) asIsInvocationCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.asIsInvoked
}

func (r *collectFailureRunner) proposalsInvocationCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.proposalsInvoked
}

func (r *collectPartialFailureRunner) asIsInvocationCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.asIsInvoked
}

func (r *collectPartialFailureRunner) proposalsInvocationCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.proposalsInvoked
}

func createManagedPermissionWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	ws := createWorkspace(t)
	ws.Manifest.Runtime = &workspace.RuntimeConfig{
		Profile: &workspace.RuntimeProfileConfig{
			Permissions: &workspace.RuntimePermissionsConfig{
				Mode:            acpruntime.PermissionModeManaged,
				ApprovalChannel: acpruntime.PermissionApprovalFailFast,
			},
		},
	}
	return ws
}

func collectRunLogsForTest(t *testing.T, service *Service, runID string) []RunLogEntry {
	t.Helper()

	out := []RunLogEntry{}
	cursor := 0
	for {
		page, ok, err := service.GetRunLogs(runID, cursor, 500)
		if err != nil {
			t.Fatalf("read run logs: %v", err)
		}
		if !ok {
			t.Fatalf("expected run logs for %s", runID)
		}
		out = append(out, page.Items...)
		if page.EOF {
			return out
		}
		cursor = page.NextCursor
	}
}

func summarizeLogSample(logs []RunLogEntry) string {
	parts := []string{}
	for _, entry := range logs {
		if entry.Message == "runtime task started" || strings.Contains(entry.Message, "permission") {
			parts = append(parts, fmt.Sprintf("%s fields=%v", entry.Message, entry.Fields))
		}
		if len(parts) >= 8 {
			break
		}
	}
	return strings.Join(parts, " | ")
}

func appendCustomProposalToManifest(task acpruntime.Task) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is required")
	}
	manifestPath := filepath.Join(writeRoot, "shard-pack-manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read shard manifest: %w", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		return fmt.Errorf("parse shard manifest: %w", err)
	}

	shardSlug := slugutil.Slugify(strings.TrimSpace(task.ShardID))
	if shardSlug == "" {
		shardSlug = "workspace"
	}
	docID := "doc.runtime-proposal." + shardSlug
	citationID := "cite.runtime-proposal." + shardSlug
	claimID := "claim.runtime-proposal." + shardSlug
	canonicalPath := path.Join("proposals", "runtime-docs-first", "proposal-"+shardSlug+".md")

	for _, existing := range manifest.Documents {
		if strings.TrimSpace(existing.ID) == docID {
			return nil
		}
	}

	content := fmt.Sprintf("# Runtime Proposal (%s)\n\nThis proposal is authored by shard `%s`.\n", shardSlug, shardSlug)
	if err := os.WriteFile(filepath.Join(writeRoot, "custom-proposal.md"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write custom proposal: %w", err)
	}

	manifest.Documents = append(manifest.Documents, contracts.AuthoredDocument{
		ID:            docID,
		Kind:          "proposal",
		Title:         "Runtime Proposal " + shardSlug,
		Path:          "custom-proposal.md",
		CanonicalPath: canonicalPath,
		Topics:        []string{"runtime-docs-first"},
		CitationIDs:   []string{citationID},
		Status:        "staged",
	})
	manifest.Citations = append(manifest.Citations, contracts.DocumentCitation{
		ID:          citationID,
		Repo:        primaryRepoForTask(task),
		Path:        "README.md",
		ClaimIDs:    []string{claimID},
		DocumentIDs: []string{docID},
	})

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal shard manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(manifestPath, encoded, 0o644); err != nil {
		return fmt.Errorf("rewrite shard manifest: %w", err)
	}
	return nil
}

func appendPartialCanonicalDocsToManifest(task acpruntime.Task) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is required")
	}
	manifestPath := filepath.Join(writeRoot, "shard-pack-manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read shard manifest: %w", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		return fmt.Errorf("parse shard manifest: %w", err)
	}
	for _, existing := range manifest.Documents {
		if strings.TrimSpace(existing.CanonicalPath) == "reports/as-is/services/authored.md" {
			return nil
		}
	}

	authoredFiles := []struct {
		ID            string
		Title         string
		Path          string
		CanonicalPath string
		CitationID    string
		ClaimID       string
		Content       string
	}{
		{
			ID:            "doc.authored.partial.asis",
			Title:         "Authored Partial Service Doc",
			Path:          "authored-service.md",
			CanonicalPath: "reports/as-is/services/authored.md",
			CitationID:    "cite.authored.partial.asis",
			ClaimID:       "claim.authored.partial.asis",
			Content:       "# Authored Service\n\nCustom authored as-is content.\n",
		},
		{
			ID:            "doc.authored.partial.coverage",
			Title:         "Authored Partial Coverage Doc",
			Path:          "custom-coverage.md",
			CanonicalPath: "reports/coverage/custom-authored.md",
			CitationID:    "cite.authored.partial.coverage",
			ClaimID:       "claim.authored.partial.coverage",
			Content:       "# Authored Coverage\n\nCustom authored coverage content.\n",
		},
	}
	repo := primaryRepoForTask(task)
	for _, authored := range authoredFiles {
		if err := os.WriteFile(filepath.Join(writeRoot, authored.Path), []byte(authored.Content), 0o644); err != nil {
			return fmt.Errorf("write authored partial doc %q: %w", authored.Path, err)
		}
		manifest.Documents = append(manifest.Documents, contracts.AuthoredDocument{
			ID:            authored.ID,
			Kind:          "report",
			Title:         authored.Title,
			Path:          authored.Path,
			CanonicalPath: authored.CanonicalPath,
			Topics:        []string{"runtime-docs-first"},
			CitationIDs:   []string{authored.CitationID},
			Status:        "staged",
		})
		manifest.Citations = append(manifest.Citations, contracts.DocumentCitation{
			ID:          authored.CitationID,
			Repo:        repo,
			Path:        "README.md",
			ClaimIDs:    []string{authored.ClaimID},
			DocumentIDs: []string{authored.ID},
		})
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal shard manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(manifestPath, encoded, 0o644); err != nil {
		return fmt.Errorf("rewrite shard manifest: %w", err)
	}
	return nil
}

func readRunQuality(t *testing.T, workspacePath string, runID string) runQualitySummary {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(workspacePath, "reports", "taskruns", runID+"-quality.json"))
	if err != nil {
		t.Fatalf("read run quality: %v", err)
	}
	var report runQualitySummary
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode run quality: %v", err)
	}
	return report
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func overwriteValidatorVerdict(task acpruntime.Task, verdictValue string) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is required")
	}
	checkedPaths := []string{
		path.Join("reports", "taskruns", task.RunID, "staging", "final", "final-run-index.json"),
		path.Join("reports", "taskruns", task.RunID, "staging", "final", "citation-index.json"),
	}
	verdict := contracts.ValidatorVerdict{
		Version:      1,
		RunID:        strings.TrimSpace(task.RunID),
		GeneratedAt:  task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
		Verdict:      verdictValue,
		Summary:      "Synthetic validator verdict for docs-first promotion test.",
		CheckedPaths: checkedPaths,
	}
	if verdictValue == "FAIL" {
		verdict.Issues = []contracts.ValidatorIssue{
			{
				Code:     "synthetic_validator_fail",
				Severity: "error",
				Message:  "validator forced fail for best-effort path test",
				Path:     checkedPaths[0],
			},
		}
	}
	encoded, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal validator verdict: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(writeRoot, "validator-verdict.json"), encoded, 0o644); err != nil {
		return fmt.Errorf("write validator verdict: %w", err)
	}
	return nil
}

func overwriteOwnerGapValidatorVerdict(task acpruntime.Task) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is required")
	}
	checkedPaths := []string{
		path.Join("reports", "taskruns", task.RunID, "staging", "final", "final-run-index.json"),
		path.Join("reports", "taskruns", task.RunID, "staging", "final", "citation-index.json"),
	}
	verdict := contracts.ValidatorVerdict{
		Version:      1,
		RunID:        strings.TrimSpace(task.RunID),
		GeneratedAt:  task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
		Verdict:      "FAIL",
		Summary:      "Synthetic owner-gap-only validator verdict.",
		CheckedPaths: checkedPaths,
		Findings: []contracts.Finding{
			{
				ID:          "finding.owner.synthetic",
				Severity:    "medium",
				Title:       "Owner mapping remains unresolved",
				Description: "Validator could not confirm an owning team from staged evidence.",
				RuleID:      "rule.owner.required",
				RelatedIDs:  []string{"svc.checkout"},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence: []contracts.Evidence{{
						Repo: "checkout",
						Path: "README.md",
					}},
				},
			},
		},
		Questions: []contracts.Question{
			{
				ID:         "q.owner.synthetic",
				Text:       "Which team owns the checkout service?",
				RelatedIDs: []string{"svc.checkout"},
			},
		},
	}
	encoded, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal validator verdict: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(writeRoot, "validator-verdict.json"), encoded, 0o644); err != nil {
		return fmt.Errorf("write validator verdict: %w", err)
	}
	return nil
}

func overwriteEvidenceAdvisoryValidatorVerdict(task acpruntime.Task) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return fmt.Errorf("write_root is required")
	}
	var citationRaw []byte
	var citationPath string
	for _, root := range task.ReadContextRoots {
		candidate := filepath.Join(strings.TrimSpace(root), "citation-index.json")
		raw, err := os.ReadFile(candidate)
		if err == nil {
			citationRaw = raw
			citationPath = candidate
			break
		}
	}
	if len(citationRaw) == 0 {
		return fmt.Errorf("read citation index from current read_context_roots")
	}
	citationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		return fmt.Errorf("parse citation index %s: %w", citationPath, err)
	}
	if len(citationIndex.Citations) == 0 {
		return fmt.Errorf("citation index must contain source evidence")
	}
	citation := citationIndex.Citations[0]
	verdict := contracts.ValidatorVerdict{
		Version:     1,
		RunID:       strings.TrimSpace(task.RunID),
		GeneratedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
		Verdict:     "FAIL",
		Summary:     "Synthetic source-repository observation incorrectly marked blocking.",
		CheckedPaths: []string{
			path.Join("reports", "taskruns", task.RunID, "staging", "final", "final-run-index.json"),
			path.Join("reports", "taskruns", task.RunID, "staging", "final", "citation-index.json"),
		},
		Issues: []contracts.ValidatorIssue{
			{
				Code:       "source_content_policy_observation",
				Severity:   "error",
				Message:    "Source evidence contains an operational policy observation.",
				Path:       citation.Path,
				CitationID: citation.ID,
			},
		},
	}
	encoded, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal evidence advisory verdict: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(writeRoot, "validator-verdict.json"), encoded, 0o644); err != nil {
		return fmt.Errorf("write evidence advisory verdict: %w", err)
	}
	return nil
}

func primaryRepoForTask(task acpruntime.Task) string {
	if value := strings.TrimSpace(task.RepoScope); value != "" {
		return value
	}
	for _, scope := range task.RepoScopes {
		if value := strings.TrimSpace(scope); value != "" {
			return value
		}
	}
	return "workspace"
}

func readRunFinalRunIndex(t *testing.T, workspacePath string, runID string) contracts.FinalRunIndex {
	t.Helper()

	finalIndexPath := filepath.Join(workspacePath, "reports", "taskruns", runID, "staging", "final", "final-run-index.json")
	raw, err := os.ReadFile(finalIndexPath)
	if err != nil {
		t.Fatalf("read final run index %q: %v", finalIndexPath, err)
	}
	index, err := contracts.ParseFinalRunIndex(raw)
	if err != nil {
		t.Fatalf("parse final run index %q: %v", finalIndexPath, err)
	}
	return index
}

func readRunValidatorVerdict(t *testing.T, workspacePath string, runID string) contracts.ValidatorVerdict {
	t.Helper()

	verdictPath := filepath.Join(workspacePath, "reports", "taskruns", runID, "validator", "validator-verdict.json")
	raw, err := os.ReadFile(verdictPath)
	if err != nil {
		t.Fatalf("read validator verdict %q: %v", verdictPath, err)
	}
	verdict, err := contracts.ParseValidatorVerdict(raw)
	if err != nil {
		t.Fatalf("parse validator verdict %q: %v", verdictPath, err)
	}
	return verdict
}

func readRunEffectiveVerdict(t *testing.T, workspacePath string, runID string) contracts.EffectiveVerdict {
	t.Helper()

	verdictPath := filepath.Join(workspacePath, "reports", "taskruns", runID, "validator", "effective-verdict.json")
	raw, err := os.ReadFile(verdictPath)
	if err != nil {
		t.Fatalf("read effective verdict %q: %v", verdictPath, err)
	}
	verdict, err := contracts.ParseEffectiveVerdict(raw)
	if err != nil {
		t.Fatalf("parse effective verdict %q: %v", verdictPath, err)
	}
	return verdict
}

func collectCustomProposalDocs(index contracts.FinalRunIndex) []contracts.FinalRunDocument {
	documents := make([]contracts.FinalRunDocument, 0, len(index.CanonicalDocuments))
	for _, document := range index.CanonicalDocuments {
		if strings.HasPrefix(strings.TrimSpace(document.CanonicalPath), "proposals/runtime-docs-first/") {
			documents = append(documents, document)
		}
	}
	return documents
}

func strPtr(value string) *string {
	return &value
}
