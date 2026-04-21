package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type Server struct {
	mu        sync.RWMutex
	workspace workspace.Root
	service   *orchestrator.Service
}

func NewServer(ws workspace.Root, service *orchestrator.Service) *Server {
	return &Server{
		workspace: ws,
		service:   service,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/workspace/validate", s.handleWorkspaceValidate)
	mux.HandleFunc("/api/workspace/bundle", s.handleWorkspaceBundle)
	mux.HandleFunc("/api/workspace/manifest", s.handleWorkspaceManifest)
	mux.HandleFunc("/api/runtime/timeouts", s.handleRuntimeTimeouts)
	mux.HandleFunc("/api/runtime/execution", s.handleRuntimeExecution)
	mux.HandleFunc("/api/runtime/profile", s.handleRuntimeProfile)
	mux.HandleFunc("/api/artifacts", s.handleArtifacts)
	mux.HandleFunc("/api/artifacts/write", s.handleArtifactsWrite)
	mux.HandleFunc("/api/git/commit", s.handleGitCommit)
	mux.HandleFunc("/api/git/proposal-branch", s.handleGitProposalBranch)
	mux.HandleFunc("/api/pipeline/init", s.handlePipelineInit)
	mux.HandleFunc("/api/pipeline/refresh", s.handlePipelineRefresh)
	mux.HandleFunc("/api/pipeline/runs", s.handlePipelineRuns)
	mux.HandleFunc("/api/pipeline/runs/", s.handlePipelineRuns)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			mux.ServeHTTP(writer, request)
			return
		}
		serveEmbeddedUI(writer, request)
	})
}

func (s *Server) Serve(ctx context.Context, address string) error {
	httpServer := &http.Server{
		Addr:    address,
		Handler: s.Handler(),
	}

	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
	}()

	err := httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) getWorkspace() workspace.Root {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspace
}

func (s *Server) setWorkspace(ws workspace.Root) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspace = ws
}

func (s *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleWorkspaceValidate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}

	ws := s.getWorkspace()
	report := ws.Validate(request.Context(), workspace.ValidateOptions{
		ResolveRepos: true,
		FetchGit:     false,
		VerifyRefs:   true,
	})
	if !report.OK {
		writeJSON(writer, http.StatusBadRequest, map[string]any{
			"ok":        false,
			"workspace": ws.Path,
			"error": map[string]string{
				"code":    "workspace_validation_failed",
				"message": "workspace validation failed",
			},
			"errors":         report.Errors,
			"warnings":       report.Warnings,
			"resolved_repos": report.ResolvedRepos,
		})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":             true,
		"workspace":      ws.Path,
		"warnings":       report.Warnings,
		"resolved_repos": report.ResolvedRepos,
	})
}

func (s *Server) handleWorkspaceBundle(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}

	ws := s.getWorkspace()
	manifest, diagnostics := ws.EffectiveBaselineBundleManifest()
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":        true,
		"workspace": ws.Path,
		"manifest":  manifest,
		"warnings":  diagnostics,
	})
}

func (s *Server) handleWorkspaceManifest(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		ws := s.getWorkspace()
		content, err := ws.ReadFile(workspace.ManifestFileName)
		if err != nil {
			writeError(writer, http.StatusNotFound, "manifest_read_failed", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"content": string(content),
		})
	case http.MethodPut:
		var payload struct {
			Content string `json:"content"`
		}
		if err := decodeStrictJSON(request, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		if strings.TrimSpace(payload.Content) == "" {
			writeError(writer, http.StatusBadRequest, "manifest_empty", "manifest content is required")
			return
		}
		if _, err := workspace.ParseManifest([]byte(payload.Content)); err != nil {
			writeError(writer, http.StatusBadRequest, "manifest_invalid", err.Error())
			return
		}

		ws := s.getWorkspace()
		if err := ws.WriteFile(workspace.ManifestFileName, []byte(payload.Content)); err != nil {
			writeError(writer, http.StatusInternalServerError, "manifest_write_failed", err.Error())
			return
		}

		reopened, err := workspace.Open(ws.Path)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "manifest_reopen_failed", err.Error())
			return
		}
		s.setWorkspace(reopened)
		writeJSON(writer, http.StatusOK, map[string]any{"ok": true})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) handleRuntimeTimeouts(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		ws := s.getWorkspace()
		resolved := acpruntime.ResolveTimeouts(ws.Manifest)
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": resolved.Persisted,
			"effective": resolved.Effective,
			"source":    resolved.Source,
		})
	case http.MethodPut:
		var payload struct {
			Timeouts workspace.RuntimeTimeoutsConfig `json:"timeouts"`
		}
		if err := decodeStrictJSON(request, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		if payload.Timeouts.IsZero() {
			writeError(writer, http.StatusBadRequest, "runtime_timeouts_empty", "timeouts payload must include at least one field")
			return
		}
		if err := validateRuntimeTimeoutPatch(payload.Timeouts); err != nil {
			writeError(writer, http.StatusBadRequest, "runtime_timeouts_invalid", err.Error())
			return
		}

		ws := s.getWorkspace()
		manifest := ws.Manifest
		if manifest.Runtime == nil {
			manifest.Runtime = &workspace.RuntimeConfig{}
		}
		if manifest.Runtime.Profile == nil {
			manifest.Runtime.Profile = &workspace.RuntimeProfileConfig{}
		}
		if manifest.Runtime.Profile.Timeouts == nil {
			manifest.Runtime.Profile.Timeouts = &workspace.RuntimeTimeoutsConfig{}
		}
		mergeRuntimeTimeoutPatch(manifest.Runtime.Profile.Timeouts, payload.Timeouts)
		if manifest.Runtime.Profile.Timeouts.IsZero() {
			manifest.Runtime.Profile.Timeouts = nil
		}
		if manifest.Runtime.Profile.IsZero() {
			manifest.Runtime.Profile = nil
		}
		if manifest.Runtime.IsZero() {
			manifest.Runtime = nil
		}

		rawManifest, err := workspace.RenderManifest(manifest)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_timeouts_render_failed", err.Error())
			return
		}
		if err := ws.WriteFile(workspace.ManifestFileName, rawManifest); err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_timeouts_write_failed", err.Error())
			return
		}
		reopened, err := workspace.Open(ws.Path)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_timeouts_reopen_failed", err.Error())
			return
		}
		s.setWorkspace(reopened)
		resolved := acpruntime.ResolveTimeouts(reopened.Manifest)
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": resolved.Persisted,
			"effective": resolved.Effective,
			"source":    resolved.Source,
		})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func validateRuntimeTimeoutPatch(patch workspace.RuntimeTimeoutsConfig) error {
	checks := []struct {
		name  string
		value *int
	}{
		{name: "step_timeout_sec", value: patch.StepTimeoutSec},
		{name: "heartbeat_sec", value: patch.HeartbeatSec},
		{name: "pipeline_timeout_sec", value: patch.PipelineTimeoutSec},
		{name: "pipeline_kill_grace_sec", value: patch.PipelineKillGraceSec},
		{name: "api_ready_timeout_sec", value: patch.APIReadyTimeoutSec},
		{name: "api_init_timeout_sec", value: patch.APIInitTimeoutSec},
		{name: "ui_init_poll_timeout_sec", value: patch.UIInitPollTimeoutSec},
		{name: "ui_cancel_poll_timeout_sec", value: patch.UICancelPollTimeoutSec},
	}
	for _, check := range checks {
		if check.value != nil && *check.value <= 0 {
			return fmt.Errorf("%s must be > 0", check.name)
		}
	}
	return nil
}

func mergeRuntimeTimeoutPatch(dst *workspace.RuntimeTimeoutsConfig, patch workspace.RuntimeTimeoutsConfig) {
	if dst == nil {
		return
	}
	if patch.StepTimeoutSec != nil {
		dst.StepTimeoutSec = patch.StepTimeoutSec
	}
	if patch.HeartbeatSec != nil {
		dst.HeartbeatSec = patch.HeartbeatSec
	}
	if patch.PipelineTimeoutSec != nil {
		dst.PipelineTimeoutSec = patch.PipelineTimeoutSec
	}
	if patch.PipelineKillGraceSec != nil {
		dst.PipelineKillGraceSec = patch.PipelineKillGraceSec
	}
	if patch.APIReadyTimeoutSec != nil {
		dst.APIReadyTimeoutSec = patch.APIReadyTimeoutSec
	}
	if patch.APIInitTimeoutSec != nil {
		dst.APIInitTimeoutSec = patch.APIInitTimeoutSec
	}
	if patch.UIInitPollTimeoutSec != nil {
		dst.UIInitPollTimeoutSec = patch.UIInitPollTimeoutSec
	}
	if patch.UICancelPollTimeoutSec != nil {
		dst.UICancelPollTimeoutSec = patch.UICancelPollTimeoutSec
	}
}

type runtimeExecutionPatch struct {
	Strategy           *string                    `json:"strategy"`
	MaxParallelTasks   *int                       `json:"max_parallel_tasks"`
	FailurePolicy      *string                    `json:"failure_policy"`
	ShardDiscoveryMode *string                    `json:"shard_discovery_mode"`
	Steps              *runtimeStepProvidersPatch `json:"steps"`
}

func (patch runtimeExecutionPatch) IsZero() bool {
	return patch.Strategy == nil &&
		patch.MaxParallelTasks == nil &&
		patch.FailurePolicy == nil &&
		patch.ShardDiscoveryMode == nil &&
		(patch.Steps == nil || patch.Steps.IsZero())
}

type runtimeStepProvidersPatch struct {
	Step0Constitution *string `json:"step0_constitution"`
	Step1Collect      *string `json:"step1_collect"`
	Step2AsIs         *string `json:"step2_as_is"`
	Step3Findings     *string `json:"step3_findings"`
	Step4Proposals    *string `json:"step4_proposals"`
}

func (patch runtimeStepProvidersPatch) IsZero() bool {
	return patch.Step0Constitution == nil &&
		patch.Step1Collect == nil &&
		patch.Step2AsIs == nil &&
		patch.Step3Findings == nil &&
		patch.Step4Proposals == nil
}

func (s *Server) handleRuntimeExecution(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		ws := s.getWorkspace()
		resolved := s.service.ResolveExecutionProfile(ws.Manifest)
		stepProviders, err := s.service.ResolveStepProviderProfile(ws.Manifest)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_step_provider_resolution_failed", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": runtimeExecutionPersistedPayload(resolved.Persisted, stepProviders.Persisted),
			"effective": runtimeExecutionEffectivePayload(resolved.Effective, stepProviders.Effective),
			"source":    runtimeExecutionSourcePayload(resolved.Source, stepProviders.Source),
		})
	case http.MethodPut:
		var payload struct {
			Execution runtimeExecutionPatch `json:"execution"`
		}
		if err := decodeStrictJSON(request, &payload); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
		if payload.Execution.IsZero() {
			writeError(writer, http.StatusBadRequest, "runtime_execution_empty", "execution payload must include at least one field")
			return
		}
		if err := validateRuntimeExecutionPatch(payload.Execution); err != nil {
			writeError(writer, http.StatusBadRequest, "runtime_execution_invalid", err.Error())
			return
		}

		ws := s.getWorkspace()
		manifest := ws.Manifest
		if manifest.Runtime == nil {
			manifest.Runtime = &workspace.RuntimeConfig{}
		}
		if manifest.Runtime.Profile == nil {
			manifest.Runtime.Profile = &workspace.RuntimeProfileConfig{}
		}
		if manifest.Runtime.Profile.Execution == nil {
			manifest.Runtime.Profile.Execution = &workspace.RuntimeExecutionConfig{}
		}
		if manifest.Runtime.Profile.Steps == nil {
			manifest.Runtime.Profile.Steps = &workspace.RuntimeStepsConfig{}
		}
		mergeRuntimeExecutionPatch(manifest.Runtime.Profile.Execution, payload.Execution)
		if payload.Execution.Steps != nil {
			mergeRuntimeStepProvidersPatch(manifest.Runtime.Profile.Steps, *payload.Execution.Steps)
		}
		if manifest.Runtime.Profile.Execution.IsZero() {
			manifest.Runtime.Profile.Execution = nil
		}
		if manifest.Runtime.Profile.Steps.IsZero() {
			manifest.Runtime.Profile.Steps = nil
		}
		if manifest.Runtime.Profile.IsZero() {
			manifest.Runtime.Profile = nil
		}
		if manifest.Runtime.IsZero() {
			manifest.Runtime = nil
		}

		rawManifest, err := workspace.RenderManifest(manifest)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_execution_render_failed", err.Error())
			return
		}
		if err := ws.WriteFile(workspace.ManifestFileName, rawManifest); err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_execution_write_failed", err.Error())
			return
		}
		reopened, err := workspace.Open(ws.Path)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_execution_reopen_failed", err.Error())
			return
		}
		s.setWorkspace(reopened)
		resolved := s.service.ResolveExecutionProfile(reopened.Manifest)
		stepProviders, err := s.service.ResolveStepProviderProfile(reopened.Manifest)
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "runtime_step_provider_resolution_failed", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"ok":        true,
			"persisted": runtimeExecutionPersistedPayload(resolved.Persisted, stepProviders.Persisted),
			"effective": runtimeExecutionEffectivePayload(resolved.Effective, stepProviders.Effective),
			"source":    runtimeExecutionSourcePayload(resolved.Source, stepProviders.Source),
		})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPut)
	}
}

func (s *Server) handleRuntimeProfile(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	ws := s.getWorkspace()
	timeouts := acpruntime.ResolveTimeouts(ws.Manifest)
	execution := s.service.ResolveExecutionProfile(ws.Manifest)
	stepProviders, err := s.service.ResolveStepProviderProfile(ws.Manifest)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "runtime_step_provider_resolution_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok": true,
		"timeouts": map[string]any{
			"persisted": timeouts.Persisted,
			"effective": timeouts.Effective,
			"source":    timeouts.Source,
		},
		"execution": map[string]any{
			"persisted": runtimeExecutionPersistedPayload(execution.Persisted, stepProviders.Persisted),
			"effective": runtimeExecutionEffectivePayload(execution.Effective, stepProviders.Effective),
			"source":    runtimeExecutionSourcePayload(execution.Source, stepProviders.Source),
		},
		"step_providers": map[string]any{
			"persisted": runtimeStepProvidersPersistedPayload(stepProviders.Persisted),
			"effective": runtimeStepProvidersEffectivePayload(stepProviders.Effective),
			"source":    runtimeStepProvidersSourcePayload(stepProviders.Source),
		},
	})
}

func validateRuntimeExecutionPatch(patch runtimeExecutionPatch) error {
	if patch.Strategy != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.Strategy))
		if value != acpruntime.ExecutionStrategySequential && value != acpruntime.ExecutionStrategyParallel {
			return fmt.Errorf("strategy must be one of: %s, %s", acpruntime.ExecutionStrategySequential, acpruntime.ExecutionStrategyParallel)
		}
	}
	if patch.MaxParallelTasks != nil && *patch.MaxParallelTasks <= 0 {
		return errors.New("max_parallel_tasks must be > 0")
	}
	if patch.FailurePolicy != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.FailurePolicy))
		if value != acpruntime.ExecutionFailurePolicyFailFast && value != acpruntime.ExecutionFailurePolicyBestEffort {
			return fmt.Errorf("failure_policy must be one of: %s, %s", acpruntime.ExecutionFailurePolicyFailFast, acpruntime.ExecutionFailurePolicyBestEffort)
		}
	}
	if patch.ShardDiscoveryMode != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.ShardDiscoveryMode))
		if value != acpruntime.ExecutionShardDiscoveryHeuristics && value != acpruntime.ExecutionShardDiscoverySemantic {
			return fmt.Errorf("shard_discovery_mode must be one of: %s, %s", acpruntime.ExecutionShardDiscoveryHeuristics, acpruntime.ExecutionShardDiscoverySemantic)
		}
	}
	if patch.Steps != nil {
		for label, value := range map[string]*string{
			"step0_constitution": patch.Steps.Step0Constitution,
			"step1_collect":      patch.Steps.Step1Collect,
			"step2_as_is":        patch.Steps.Step2AsIs,
			"step3_findings":     patch.Steps.Step3Findings,
			"step4_proposals":    patch.Steps.Step4Proposals,
		} {
			if value == nil {
				continue
			}
			provider := strings.TrimSpace(strings.ToLower(*value))
			if provider != string(acpruntime.ProviderClaudeCode) && provider != string(acpruntime.ProviderQwenCode) {
				return fmt.Errorf("%s must be one of: %s, %s", label, acpruntime.ProviderClaudeCode, acpruntime.ProviderQwenCode)
			}
		}
	}
	return nil
}

func mergeRuntimeExecutionPatch(dst *workspace.RuntimeExecutionConfig, patch runtimeExecutionPatch) {
	if dst == nil {
		return
	}
	if patch.Strategy != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.Strategy))
		dst.Strategy = value
	}
	if patch.MaxParallelTasks != nil {
		dst.MaxParallel = patch.MaxParallelTasks
	}
	if patch.FailurePolicy != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.FailurePolicy))
		dst.FailurePolicy = value
	}
	if patch.ShardDiscoveryMode != nil {
		value := strings.TrimSpace(strings.ToLower(*patch.ShardDiscoveryMode))
		if dst.ShardDiscovery == nil {
			dst.ShardDiscovery = &workspace.RuntimeShardDiscoveryConfig{}
		}
		dst.ShardDiscovery.Mode = value
	}
}

func mergeRuntimeStepProvidersPatch(dst *workspace.RuntimeStepsConfig, patch runtimeStepProvidersPatch) {
	if dst == nil {
		return
	}
	mergeStep := func(target **workspace.RuntimeStepConfig, raw *string) {
		if raw == nil {
			return
		}
		if *target == nil {
			*target = &workspace.RuntimeStepConfig{}
		}
		(*target).Provider = strings.TrimSpace(strings.ToLower(*raw))
		if (*target).IsZero() {
			*target = nil
		}
	}
	mergeStep(&dst.Step0Constitution, patch.Step0Constitution)
	mergeStep(&dst.Step1Collect, patch.Step1Collect)
	mergeStep(&dst.Step2AsIs, patch.Step2AsIs)
	mergeStep(&dst.Step3Findings, patch.Step3Findings)
	mergeStep(&dst.Step4Proposals, patch.Step4Proposals)
}

func runtimeExecutionPersistedPayload(persisted workspace.RuntimeExecutionConfig, steps workspace.RuntimeStepsConfig) map[string]any {
	payload := map[string]any{}
	if value := strings.TrimSpace(persisted.Strategy); value != "" {
		payload["strategy"] = value
	}
	if persisted.MaxParallel != nil {
		payload["max_parallel_tasks"] = *persisted.MaxParallel
	}
	if value := strings.TrimSpace(persisted.FailurePolicy); value != "" {
		payload["failure_policy"] = value
	}
	if persisted.ShardDiscovery != nil {
		if value := strings.TrimSpace(persisted.ShardDiscovery.Mode); value != "" {
			payload["shard_discovery_mode"] = value
		}
	}
	if runtimeSteps := runtimeStepProvidersPersistedPayload(steps); len(runtimeSteps) > 0 {
		payload["steps"] = runtimeSteps
	}
	return payload
}

func runtimeExecutionEffectivePayload(effective acpruntime.ExecutionValues, steps acpruntime.StepProviderValues) map[string]any {
	payload := map[string]any{
		"strategy":             effective.Strategy,
		"max_parallel_tasks":   effective.MaxParallel,
		"failure_policy":       effective.FailurePolicy,
		"shard_discovery_mode": effective.ShardMode,
	}
	if runtimeSteps := runtimeStepProvidersEffectivePayload(steps); len(runtimeSteps) > 0 {
		payload["steps"] = runtimeSteps
	}
	return payload
}

func runtimeExecutionSourcePayload(source acpruntime.ExecutionSources, steps acpruntime.StepProviderSources) map[string]any {
	payload := map[string]any{
		"strategy":             source.Strategy,
		"max_parallel_tasks":   source.MaxParallel,
		"failure_policy":       source.FailurePolicy,
		"shard_discovery_mode": source.ShardMode,
	}
	if runtimeSteps := runtimeStepProvidersSourcePayload(steps); len(runtimeSteps) > 0 {
		payload["steps"] = runtimeSteps
	}
	return payload
}

func runtimeStepProvidersPersistedPayload(persisted workspace.RuntimeStepsConfig) map[string]any {
	payload := map[string]any{}
	appendStep := func(key string, step *workspace.RuntimeStepConfig) {
		if step == nil {
			return
		}
		if provider := strings.TrimSpace(step.Provider); provider != "" {
			payload[key] = provider
		}
	}
	appendStep("step0_constitution", persisted.Step0Constitution)
	appendStep("step1_collect", persisted.Step1Collect)
	appendStep("step2_as_is", persisted.Step2AsIs)
	appendStep("step3_findings", persisted.Step3Findings)
	appendStep("step4_proposals", persisted.Step4Proposals)
	return payload
}

func runtimeStepProvidersEffectivePayload(effective acpruntime.StepProviderValues) map[string]any {
	payload := map[string]any{}
	for key, value := range effective.StringMap() {
		payload[key] = value
	}
	return payload
}

func runtimeStepProvidersSourcePayload(source acpruntime.StepProviderSources) map[string]any {
	payload := map[string]any{}
	for key, value := range source {
		payload[key] = value
	}
	return payload
}

func (s *Server) handleArtifactsWrite(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	path := strings.TrimSpace(payload.Path)
	if path == "" {
		writeError(writer, http.StatusBadRequest, "artifact_path_required", "path is required")
		return
	}
	if !strings.HasPrefix(path, "charter/") && !strings.HasPrefix(path, "skills/") {
		writeError(writer, http.StatusBadRequest, "artifact_path_forbidden", "only charter/* and skills/* are editable through this endpoint")
		return
	}
	ws := s.getWorkspace()
	if err := ws.WriteFile(path, []byte(payload.Content)); err != nil {
		writeError(writer, http.StatusBadRequest, "artifact_write_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGitCommit(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}
	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = "chore: update ACP workspace artifacts"
	}
	ws := s.getWorkspace()
	if _, err := runGit(request.Context(), ws.Path, "add", "-A"); err != nil {
		writeError(writer, http.StatusBadRequest, "git_add_failed", err.Error())
		return
	}
	output, err := runGit(request.Context(), ws.Path, "commit", "-m", message)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "nothing to commit") {
			writeJSON(writer, http.StatusOK, map[string]any{
				"ok":      true,
				"status":  "no_changes",
				"message": "nothing to commit",
			})
			return
		}
		writeError(writer, http.StatusBadRequest, "git_commit_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":     true,
		"status": "committed",
		"output": output,
	})
}

func (s *Server) handleGitProposalBranch(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload struct {
		Name string `json:"name"`
	}
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
		return
	}

	branch := sanitizeBranchName(payload.Name)
	if branch == "" {
		branch = "proposal/" + time.Now().UTC().Format("20060102-150405")
	}

	ws := s.getWorkspace()
	if _, err := runGit(request.Context(), ws.Path, "checkout", "-b", branch); err != nil {
		if _, fallbackErr := runGit(request.Context(), ws.Path, "checkout", branch); fallbackErr != nil {
			writeError(writer, http.StatusBadRequest, "git_branch_failed", err.Error())
			return
		}
	}

	writeJSON(writer, http.StatusOK, map[string]any{
		"ok":     true,
		"branch": branch,
	})
}

func (s *Server) handleArtifacts(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}

	relPath := strings.TrimSpace(request.URL.Query().Get("path"))
	if relPath == "" {
		writeError(writer, http.StatusBadRequest, "bad_request", "path query parameter is required")
		return
	}
	ws := s.getWorkspace()
	content, err := ws.ReadFile(relPath)
	if err != nil {
		if errors.Is(err, workspace.ErrPathTraversal) || errors.Is(err, workspace.ErrPathAbsolute) {
			writeError(writer, http.StatusBadRequest, "path_invalid", err.Error())
			return
		}
		writeError(writer, http.StatusNotFound, "artifact_not_found", err.Error())
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(relPath))
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	writer.Header().Set("Content-Type", contentType)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(content)
}

func (s *Server) handlePipelineInit(writer http.ResponseWriter, request *http.Request) {
	s.handlePipelineStart(writer, request, orchestrator.PipelineInit, "ui")
}

func (s *Server) handlePipelineRefresh(writer http.ResponseWriter, request *http.Request) {
	s.handlePipelineStart(writer, request, orchestrator.PipelineRefresh, "manual")
}

type pipelineRequest struct {
	Commit               bool   `json:"commit"`
	CreateProposalBranch bool   `json:"create_proposal_branch"`
	Trigger              string `json:"trigger"`
}

var supportedTriggers = map[string]struct{}{
	"ui":         {},
	"manual":     {},
	"hook":       {},
	"automation": {},
}

func (s *Server) handlePipelineStart(writer http.ResponseWriter, request *http.Request, pipeline orchestrator.Pipeline, defaultTrigger string) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}

	body := pipelineRequest{Trigger: defaultTrigger}
	if request.Body != nil {
		decoder := json.NewDecoder(request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil && !errors.Is(err, context.Canceled) {
			if !errors.Is(err, io.EOF) {
				writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
				return
			}
		}
		var trailing any
		if err := decoder.Decode(&trailing); err == nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		} else if !errors.Is(err, io.EOF) {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
	}
	if strings.TrimSpace(body.Trigger) == "" {
		body.Trigger = defaultTrigger
	}
	if _, ok := supportedTriggers[body.Trigger]; !ok {
		writeError(writer, http.StatusBadRequest, "trigger_unsupported", "trigger must be one of: ui, manual, hook, automation")
		return
	}
	if body.Commit || body.CreateProposalBranch {
		writeError(writer, http.StatusNotImplemented, "not_supported", "commit/create_proposal_branch is not supported in this slice")
		return
	}

	runID, err := s.service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace:      s.getWorkspace(),
		Pipeline:       pipeline,
		NonInteractive: true,
	})
	if err != nil {
		if statusCode, code, message, ok := mapTypedRunnerAPIError(err); ok {
			writeError(writer, statusCode, code, message)
			return
		}
		writeError(writer, http.StatusBadRequest, "run_start_failed", err.Error())
		return
	}

	writeJSON(writer, http.StatusAccepted, map[string]any{
		"run_id": runID,
		"status": "started",
	})
}

func (s *Server) handlePipelineRuns(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		s.handlePipelineRunsGet(writer, request)
	case http.MethodPost:
		s.handlePipelineRunsPost(writer, request)
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPost)
	}
}

func (s *Server) handlePipelineRunsGet(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/api/pipeline/runs" || request.URL.Path == "/api/pipeline/runs/" {
		const (
			defaultLimit = 50
			maxLimit     = 500
		)
		limit := defaultLimit
		rawLimit := strings.TrimSpace(request.URL.Query().Get("limit"))
		if rawLimit != "" {
			parsedLimit, err := strconv.Atoi(rawLimit)
			if err != nil || parsedLimit <= 0 {
				writeError(writer, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer")
				return
			}
			if parsedLimit > maxLimit {
				parsedLimit = maxLimit
			}
			limit = parsedLimit
		}

		runs := s.service.ListRuns(limit)
		items := make([]map[string]any, 0, len(runs))
		for _, runInfo := range runs {
			items = append(items, formatRunInfoPayload(runInfo))
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"items": items,
		})
		return
	}

	rest := strings.TrimPrefix(request.URL.Path, "/api/pipeline/runs/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(writer, http.StatusNotFound, "run_id_required", "run id is required")
		return
	}
	runID := parts[0]

	if len(parts) == 1 {
		runInfo, ok := s.service.GetRun(runID)
		if !ok {
			writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
			return
		}
		writeJSON(writer, http.StatusOK, formatRunInfoPayload(runInfo))
		return
	}

	if len(parts) == 2 && parts[1] == "artifacts" {
		artifacts, ok := s.service.GetRunArtifacts(runID)
		if !ok {
			writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"run_id":    runID,
			"artifacts": artifacts,
		})
		return
	}

	if len(parts) == 2 && parts[1] == "logs" {
		cursor := 0
		rawCursor := strings.TrimSpace(request.URL.Query().Get("cursor"))
		if rawCursor != "" {
			parsedCursor, err := strconv.Atoi(rawCursor)
			if err != nil || parsedCursor < 0 {
				writeError(writer, http.StatusBadRequest, "invalid_cursor", "cursor must be a non-negative integer")
				return
			}
			cursor = parsedCursor
		}

		limit := 200
		rawLimit := strings.TrimSpace(request.URL.Query().Get("limit"))
		if rawLimit != "" {
			parsedLimit, err := strconv.Atoi(rawLimit)
			if err != nil || parsedLimit <= 0 {
				writeError(writer, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer")
				return
			}
			if parsedLimit > 500 {
				parsedLimit = 500
			}
			limit = parsedLimit
		}

		page, ok, err := s.service.GetRunLogs(runID, cursor, limit)
		if !ok {
			writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
			return
		}
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "run_logs_unavailable", err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"run_id":      page.RunID,
			"items":       page.Items,
			"next_cursor": page.NextCursor,
			"eof":         page.EOF,
		})
		return
	}

	writeError(writer, http.StatusNotFound, "endpoint_not_found", "endpoint not found")
}

func (s *Server) handlePipelineRunsPost(writer http.ResponseWriter, request *http.Request) {
	rest := strings.TrimPrefix(request.URL.Path, "/api/pipeline/runs/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "cancel" {
		writeError(writer, http.StatusNotFound, "endpoint_not_found", "endpoint not found")
		return
	}
	runID := strings.TrimSpace(parts[0])

	if request.Body != nil {
		var payload struct{}
		if err := decodeStrictJSON(request, &payload); err != nil && !errors.Is(err, io.EOF) {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid request body")
			return
		}
	}

	if err := s.service.CancelRun(runID); err != nil {
		if errors.Is(err, orchestrator.ErrRunNotFound) {
			writeError(writer, http.StatusNotFound, "run_not_found", "run not found")
			return
		}
		if errors.Is(err, orchestrator.ErrRunNotCancelable) {
			writeError(writer, http.StatusConflict, "run_not_cancelable", "run is already terminal")
			return
		}
		writeError(writer, http.StatusInternalServerError, "run_cancel_failed", err.Error())
		return
	}

	writeJSON(writer, http.StatusAccepted, map[string]any{
		"run_id": runID,
		"status": "cancel_requested",
	})
}

func runGit(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func sanitizeBranchName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, " ", "-")
	allowed := make([]rune, 0, len(name))
	for _, r := range name {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isDigit || r == '/' || r == '-' || r == '_' {
			allowed = append(allowed, r)
			continue
		}
		allowed = append(allowed, '-')
	}
	branch := strings.Trim(strings.ReplaceAll(string(allowed), "//", "/"), "/-")
	if branch == "" {
		return ""
	}
	if !strings.HasPrefix(branch, "proposal/") {
		return "proposal/" + branch
	}
	return branch
}

func writeMethodNotAllowed(writer http.ResponseWriter, allowedMethod string) {
	writer.Header().Set("Allow", allowedMethod)
	writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}

func decodeStrictJSON(request *http.Request, payload any) error {
	if request == nil || request.Body == nil {
		return io.EOF
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(payload); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("unexpected trailing JSON tokens")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func writeError(writer http.ResponseWriter, statusCode int, code string, message string) {
	writeJSON(writer, statusCode, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func formatOptionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func formatOptionalString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func formatRunInfoPayload(runInfo orchestrator.RunInfo) map[string]any {
	return map[string]any{
		"run_id":         runInfo.RunID,
		"pipeline":       runInfo.Pipeline,
		"status":         runInfo.Status,
		"started_at":     runInfo.StartedAt.UTC().Format(time.RFC3339),
		"finished_at":    formatOptionalTime(runInfo.FinishedAt),
		"current_step":   runInfo.CurrentStep,
		"step_providers": runInfo.StepProviders,
		"warnings":       runInfo.Warnings,
		"error_code":     formatOptionalString(runInfo.ErrorCode),
		"error":          formatOptionalString(runInfo.Error),
	}
}

func mapTypedRunnerAPIError(err error) (statusCode int, code string, message string, ok bool) {
	runnerCode, runnerMessage, classified := acpruntime.ClassifyError(err)
	if !classified {
		return 0, "", "", false
	}
	switch runnerCode {
	case string(acpruntime.ErrorCodeRunnerUnavailable):
		return http.StatusServiceUnavailable, runnerCode, runnerMessage, true
	case string(acpruntime.ErrorCodeRunnerParseFailed):
		// Parse failures are surfaced as run-level failures (`error_code`) after async start.
		return 0, "", "", false
	default:
		return http.StatusBadRequest, runnerCode, runnerMessage, true
	}
}
