package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type retryRequest struct {
	StepID   string   `json:"step_id,omitempty"`
	ScopeIDs []string `json:"scope_ids,omitempty"`
	PlanHash string   `json:"plan_hash,omitempty"`
}

type retryPlan struct {
	ParentRunID        string   `json:"parent_run_id"`
	Pipeline           string   `json:"pipeline"`
	RequestedStep      string   `json:"requested_step"`
	EffectiveStartStep string   `json:"effective_start_step"`
	RequestedScopes    []string `json:"requested_scopes"`
	EffectiveScopes    []string `json:"effective_scopes"`
	ReusedInputs       []string `json:"reused_inputs"`
	ExecuteSteps       []string `json:"execute_steps"`
	InvalidatedSteps   []string `json:"invalidated_steps"`
	EstimatedUnits     int      `json:"estimated_units"`
	Widened            bool     `json:"widened"`
	WidenReason        string   `json:"widen_reason,omitempty"`
	PlanHash           string   `json:"plan_hash"`
}

func (s *Server) handlePipelineRetryPlan(writer http.ResponseWriter, request *http.Request, parentRunID string) {
	var payload retryRequest
	if err := decodeStrictJSON(request, &payload); err != nil && !errors.Is(err, io.EOF) {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid retry-plan request body")
		return
	}
	snapshot := s.sessionSnapshot()
	plan, status, code, err := buildRetryPlan(snapshot, parentRunID, payload)
	if err != nil {
		writeError(writer, status, code, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, plan)
}

func (s *Server) handlePipelineRetry(writer http.ResponseWriter, request *http.Request, parentRunID string) {
	var payload retryRequest
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid retry request body")
		return
	}
	if strings.TrimSpace(payload.PlanHash) == "" {
		writeError(writer, http.StatusBadRequest, "retry_plan_hash_required", "plan_hash is required")
		return
	}
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	snapshot := s.sessionSnapshot()
	plan, status, code, err := buildRetryPlan(snapshot, parentRunID, payload)
	if err != nil {
		writeError(writer, status, code, err.Error())
		return
	}
	if !strings.EqualFold(strings.TrimSpace(payload.PlanHash), plan.PlanHash) {
		writeError(writer, http.StatusConflict, "retry_plan_stale", "retry inputs changed; calculate a new retry plan")
		return
	}
	pipeline, err := orchestrator.ParsePipeline(plan.Pipeline)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "retry_pipeline_invalid", err.Error())
		return
	}
	if current := s.sessionSnapshot(); current.Generation != snapshot.Generation || current.Service != snapshot.Service {
		writeError(writer, http.StatusConflict, "session_generation_changed", "workspace session changed before retry admission")
		return
	}
	runID, err := snapshot.Service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace: snapshot.Workspace, Pipeline: pipeline, NonInteractive: true,
		ResumeFromStep: plan.EffectiveStartStep, RetryParentRunID: parentRunID,
		RetryReason: "operator_retry", RetryRequestedStep: plan.RequestedStep, RetryRequestedScopes: plan.RequestedScopes,
		RetryScopes: plan.EffectiveScopes, RetryReusedInputs: plan.ReusedInputs,
	})
	if err != nil {
		if errors.Is(err, orchestrator.ErrRunActive) {
			writeError(writer, http.StatusConflict, "run_active", "another run is already active")
			return
		}
		writeError(writer, http.StatusBadRequest, "retry_start_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusAccepted, map[string]any{"run_id": runID, "status": "started", "parent_run_id": parentRunID, "retry_plan": plan})
}

func buildRetryPlan(snapshot serverSessionSnapshot, parentRunID string, request retryRequest) (retryPlan, int, string, error) {
	if snapshot.Service == nil {
		return retryPlan{}, http.StatusPreconditionRequired, "workspace_not_selected", fmt.Errorf("select a workspace before planning retry")
	}
	parent, ok := snapshot.Service.GetRun(strings.TrimSpace(parentRunID))
	if !ok || parent.Pipeline == string(orchestrator.PipelineQA) {
		return retryPlan{}, http.StatusNotFound, "run_not_found", fmt.Errorf("run not found")
	}
	if !retryParentTerminal(parent.Status) {
		return retryPlan{}, http.StatusConflict, "retry_parent_not_terminal", fmt.Errorf("retry is available only after the parent run reaches a terminal state")
	}
	steps := pipelineSteps(parent.Pipeline)
	requested := strings.TrimSpace(request.StepID)
	if requested == "" {
		requested = strings.TrimSpace(parent.CurrentStep)
	}
	index := indexString(steps, requested)
	if index < 0 {
		index = 0
		requested = steps[0]
	}
	effectiveIndex := index
	widenReason := ""
	scopes := normalizeRetryScopes(request.ScopeIDs)
	needsParentInputs := effectiveIndex > 0 || (strings.Contains(steps[effectiveIndex], "collect") && len(scopes) > 0)
	if needsParentInputs {
		if err := orchestrator.ValidateRetryStaging(snapshot.Workspace, parent.RunID, steps[effectiveIndex], scopes); err != nil {
			effectiveIndex = 0
			scopes = nil
			widenReason = "Validated parent inputs are unavailable or no longer pass validation, so retry must restart from the first pipeline step."
		}
	}
	reused := append([]string(nil), steps[:effectiveIndex]...)
	execute := append([]string(nil), steps[effectiveIndex:]...)
	if strings.Contains(steps[effectiveIndex], "collect") && len(scopes) > 0 {
		reused = append(reused, "validated_sibling_collect_scopes")
	}
	estimatedUnits := len(execute)
	if strings.Contains(steps[effectiveIndex], "collect") && len(scopes) > 0 {
		estimatedUnits = len(scopes) + len(execute) - 1
	}
	plan := retryPlan{ParentRunID: parent.RunID, Pipeline: parent.Pipeline, RequestedStep: requested, EffectiveStartStep: steps[effectiveIndex], RequestedScopes: normalizeRetryScopes(request.ScopeIDs), EffectiveScopes: scopes, ReusedInputs: reused, ExecuteSteps: execute, InvalidatedSteps: execute, EstimatedUnits: estimatedUnits, Widened: effectiveIndex != index || widenReason != "", WidenReason: widenReason}
	artifacts, _ := snapshot.Service.GetRunArtifacts(parent.RunID)
	plan.PlanHash = retryPlanHash(snapshot, parent, artifacts, plan)
	return plan, http.StatusOK, "", nil
}

func retryParentTerminal(status orchestrator.RunStatus) bool {
	return status == orchestrator.RunStatusSucceeded || status == orchestrator.RunStatusFailed || status == orchestrator.RunStatusCanceled
}

func retryPlanHash(snapshot serverSessionSnapshot, parent orchestrator.RunInfo, artifacts []orchestrator.Artifact, plan retryPlan) string {
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	digests := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		content, err := os.ReadFile(filepath.Join(snapshot.Workspace.Path, filepath.FromSlash(artifact.Path)))
		if err != nil {
			digests = append(digests, artifact.Path+":unavailable")
			continue
		}
		digest := sha256.Sum256(content)
		digests = append(digests, artifact.Path+":"+hex.EncodeToString(digest[:]))
	}
	payload := struct {
		Parent            orchestrator.RunInfo `json:"parent"`
		Plan              retryPlan            `json:"plan"`
		Digests           []string             `json:"digests"`
		StagingDigests    []string             `json:"staging_digests"`
		SourceFingerprint string               `json:"source_fingerprint"`
	}{Parent: parent, Plan: plan, Digests: digests, StagingDigests: retryStagingDigests(snapshot.Workspace.Path, parent.RunID), SourceFingerprint: retrySourceFingerprint(snapshot)}
	payload.Plan.PlanHash = ""
	raw, _ := json.Marshal(payload)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func retryStagingDigests(root, runID string) []string {
	staging := filepath.Join(root, "reports", "taskruns", runID, "staging")
	result := []string{}
	_ = filepath.WalkDir(staging, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			result = append(result, "unavailable:"+current)
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(staging, current)
		if err != nil {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			result = append(result, filepath.ToSlash(rel)+":symlink")
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		raw, err := os.ReadFile(current)
		if err != nil {
			result = append(result, filepath.ToSlash(rel)+":unavailable")
			return nil
		}
		digest := sha256.Sum256(raw)
		result = append(result, filepath.ToSlash(rel)+":"+hex.EncodeToString(digest[:]))
		return nil
	})
	sort.Strings(result)
	return result
}

func retrySourceFingerprint(snapshot serverSessionSnapshot) string {
	resolved, diagnostics := snapshot.Workspace.ResolveRepoSources(context.Background(), workspace.ResolveOptions{FetchGit: false, VerifyRefs: true})
	payload := struct {
		Manifest    workspace.Manifest       `json:"manifest"`
		Resolved    []workspace.ResolvedRepo `json:"resolved"`
		Diagnostics []workspace.Diagnostic   `json:"diagnostics"`
	}{Manifest: snapshot.Workspace.Manifest, Resolved: resolved, Diagnostics: diagnostics}
	raw, _ := json.Marshal(payload)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func pipelineSteps(pipeline string) []string {
	if pipeline == string(orchestrator.PipelineRefresh) {
		return []string{"refresh.step1.collect", "refresh.step2.asis_docs", "refresh.step3.findings", "refresh.step4.proposals"}
	}
	return []string{"init.step0.constitution", "init.step1.collect", "init.step2.asis_docs", "init.step3.findings", "init.step4.proposals"}
}

func indexString(items []string, target string) int {
	for index, item := range items {
		if item == target {
			return index
		}
	}
	return -1
}
func normalizeRetryScopes(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
