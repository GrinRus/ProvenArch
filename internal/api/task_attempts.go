package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type taskAttemptStartRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Pipeline       string `json:"pipeline,omitempty"`
	Intent         string `json:"intent,omitempty"`
}

type taskAttemptRetryRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Pipeline       string `json:"pipeline,omitempty"`
	Intent         string `json:"intent,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type attemptAdmissionFingerprint struct {
	TaskID          string `json:"task_id"`
	TaskRevision    int    `json:"task_revision"`
	Pipeline        string `json:"pipeline"`
	Intent          string `json:"intent"`
	ParentAttemptID string `json:"parent_attempt_id,omitempty"`
	RetryReason     string `json:"retry_reason,omitempty"`
}

type attemptConflict struct {
	Code    string
	Message string
}

func (e *attemptConflict) Error() string { return e.Message }

func (s *Server) handleTaskAttempts(writer http.ResponseWriter, request *http.Request, taskID, attemptID string) {
	if attemptID == "" && request.Method == http.MethodPost {
		s.handleTaskAttemptAdmission(writer, request, taskID, taskAttemptStartRequest{})
		return
	}
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPost)
		return
	}
	registry, err := s.taskRegistrySnapshot()
	if err != nil {
		writeTaskHistoryUnavailable(writer, err)
		return
	}
	history := registry.Snapshot()
	if _, ok := findTask(history, taskID); !ok {
		writeError(writer, http.StatusNotFound, "task_not_found", "task not found")
		return
	}
	if attemptID == "" {
		items := make([]producttasks.Attempt, 0)
		for _, attempt := range history.Attempts {
			if attempt.TaskID == taskID {
				items = append(items, producttasks.CloneAttempt(attempt))
			}
		}
		writeJSON(writer, http.StatusOK, map[string]any{"items": items, "diagnostics": registry.Diagnostics()})
		return
	}
	for _, attempt := range history.Attempts {
		if attempt.TaskID == taskID && attempt.AttemptID == attemptID {
			writeJSON(writer, http.StatusOK, map[string]any{"attempt": producttasks.CloneAttempt(attempt)})
			return
		}
	}
	writeError(writer, http.StatusNotFound, "attempt_not_found", "attempt not found for task")
}

func (s *Server) handleTaskAttemptAdmission(writer http.ResponseWriter, request *http.Request, taskID string, retryDefaults taskAttemptStartRequest) {
	var payload taskAttemptStartRequest
	if retryDefaults.IdempotencyKey != "" {
		payload = retryDefaults
	} else if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid attempt admission request")
		return
	}
	key := strings.TrimSpace(payload.IdempotencyKey)
	if key == "" {
		writeError(writer, http.StatusBadRequest, "idempotency_key_required", "idempotency_key is required")
		return
	}
	pipeline, intent, err := parseAttemptAdmissionOptions(payload.Pipeline, payload.Intent)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "attempt_options_invalid", err.Error())
		return
	}
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	snapshot := s.sessionSnapshot()
	if !snapshot.RuntimeSelected || snapshot.Service == nil {
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select a workspace and runner before starting an attempt")
		return
	}
	registry, err := s.taskRegistrySnapshot()
	if err != nil {
		writeTaskHistoryUnavailable(writer, err)
		return
	}
	history := registry.Snapshot()
	task, ok := findTask(history, taskID)
	if !ok {
		writeError(writer, http.StatusNotFound, "task_not_found", "task not found")
		return
	}
	if task.Lifecycle != producttasks.LifecycleOpen {
		writeError(writer, http.StatusConflict, "task_archived", "archived tasks cannot admit new attempts")
		return
	}
	if err := validateTaskRepositoryScope(task, snapshot.Workspace); err != nil {
		writeError(writer, http.StatusBadRequest, "task_scope_invalid", err.Error())
		return
	}
	fingerprint := attemptFingerprint(attemptAdmissionFingerprint{TaskID: task.TaskID, TaskRevision: task.Revision, Pipeline: string(pipeline), Intent: string(intent)})
	if existing, found := findIdempotentAttempt(history, task.TaskID, key); found {
		if existing.RequestFingerprint != fingerprint {
			writeError(writer, http.StatusConflict, "idempotency_conflict", "idempotency_key was already used for a different attempt request")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"attempt": producttasks.CloneAttempt(existing), "idempotent": true})
		return
	}
	if conflict := validateTaskCapacity(history, snapshot.Service, intent); conflict != nil {
		writeError(writer, http.StatusConflict, conflict.Code, conflict.Message)
		return
	}
	attempt, err := s.buildAdmittedAttempt(snapshot, task, pipeline, intent, key, fingerprint, nil, "")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "attempt_admission_invalid", err.Error())
		return
	}
	if err := registry.Update(func(candidate *producttasks.History) error {
		return appendAttemptToHistory(candidate, task.TaskID, attempt)
	}); err != nil {
		writeError(writer, http.StatusInternalServerError, "attempt_persist_failed", err.Error())
		return
	}
	runID, err := snapshot.Service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace: snapshot.Workspace, Pipeline: pipeline, Intent: intent,
		TaskID: task.TaskID, AttemptID: attempt.AttemptID, RequestedRunID: attempt.RunID,
		ProviderOverride: acpruntime.Provider(attempt.EffectiveRuntime.Provider),
		ProviderModels:   attemptProviderModels(snapshot, attempt), NonInteractive: true,
	})
	if err != nil || runID != attempt.RunID {
		cleanupErr := registry.Update(func(candidate *producttasks.History) error {
			return removeAttemptFromHistory(candidate, task.TaskID, attempt.AttemptID)
		})
		if cleanupErr != nil {
			writeError(writer, http.StatusInternalServerError, "attempt_admission_inconsistent", fmt.Sprintf("run admission failed: %v; cleanup failed: %v", err, cleanupErr))
			return
		}
		if err == nil {
			err = errors.New("runtime returned a different run identity")
		}
		if errors.Is(err, orchestrator.ErrRunActive) {
			writeError(writer, http.StatusConflict, "run_active", "another run is already active")
			return
		}
		if errors.Is(err, orchestrator.ErrQueueUnsupported) {
			writeError(writer, http.StatusBadRequest, "queue_intent_not_supported", err.Error())
			return
		}
		if statusCode, code, message, ok := mapTypedRunnerAPIError(err); ok {
			writeError(writer, statusCode, code, message)
			return
		}
		writeError(writer, http.StatusBadRequest, "attempt_start_failed", err.Error())
		return
	}
	if runInfo, ok := snapshot.Service.GetRun(runID); ok {
		_ = registry.Update(func(candidate *producttasks.History) error { return updateAttemptFromRun(candidate, runInfo) })
	}
	s.watchAdmittedAttempt(snapshot.Service, registry, runID, attempt.AttemptID)
	result, _ := findAttempt(registry.Snapshot(), task.TaskID, attempt.AttemptID)
	writeJSON(writer, http.StatusAccepted, map[string]any{"attempt": result, "idempotent": false})
}

func (s *Server) handleTaskAttemptRetry(writer http.ResponseWriter, request *http.Request, taskID, attemptID string) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload taskAttemptRetryRequest
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid attempt retry request")
		return
	}
	key := strings.TrimSpace(payload.IdempotencyKey)
	if key == "" {
		writeError(writer, http.StatusBadRequest, "idempotency_key_required", "idempotency_key is required")
		return
	}
	pipeline, intent, err := parseAttemptAdmissionOptions(payload.Pipeline, payload.Intent)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "attempt_options_invalid", err.Error())
		return
	}
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	snapshot := s.sessionSnapshot()
	if !snapshot.RuntimeSelected || snapshot.Service == nil {
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select a workspace and runner before retrying an attempt")
		return
	}
	registry, err := s.taskRegistrySnapshot()
	if err != nil {
		writeTaskHistoryUnavailable(writer, err)
		return
	}
	history := registry.Snapshot()
	task, ok := findTask(history, taskID)
	if !ok {
		writeError(writer, http.StatusNotFound, "task_not_found", "task not found")
		return
	}
	parent, ok := findAttempt(history, taskID, attemptID)
	if !ok {
		writeError(writer, http.StatusNotFound, "attempt_not_found", "attempt not found for task")
		return
	}
	if !isTerminalAttempt(parent.Status) {
		writeError(writer, http.StatusConflict, "retry_parent_not_terminal", "retry is available only after the parent attempt reaches a terminal state")
		return
	}
	if strings.TrimSpace(payload.Pipeline) == "" && parent.Pipeline != "" {
		pipeline, intent, err = parseAttemptAdmissionOptions(parent.Pipeline, payload.Intent)
		if err != nil {
			writeError(writer, http.StatusBadRequest, "attempt_options_invalid", err.Error())
			return
		}
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "operator_retry"
	}
	fingerprint := attemptFingerprint(attemptAdmissionFingerprint{TaskID: task.TaskID, TaskRevision: task.Revision, Pipeline: string(pipeline), Intent: string(intent), ParentAttemptID: parent.AttemptID, RetryReason: reason})
	if existing, found := findIdempotentAttempt(history, task.TaskID, key); found {
		if existing.RequestFingerprint != fingerprint {
			writeError(writer, http.StatusConflict, "idempotency_conflict", "idempotency_key was already used for a different retry request")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"attempt": producttasks.CloneAttempt(existing), "idempotent": true})
		return
	}
	if conflict := validateTaskCapacity(history, snapshot.Service, intent); conflict != nil {
		writeError(writer, http.StatusConflict, conflict.Code, conflict.Message)
		return
	}
	attempt, err := s.buildAdmittedAttempt(snapshot, task, pipeline, intent, key, fingerprint, &parent.AttemptID, reason)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "attempt_admission_invalid", err.Error())
		return
	}
	if err := registry.Update(func(candidate *producttasks.History) error {
		return appendAttemptToHistory(candidate, task.TaskID, attempt)
	}); err != nil {
		writeError(writer, http.StatusInternalServerError, "attempt_persist_failed", err.Error())
		return
	}
	runID, err := snapshot.Service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace: snapshot.Workspace, Pipeline: pipeline, Intent: intent,
		TaskID: task.TaskID, AttemptID: attempt.AttemptID, RequestedRunID: attempt.RunID,
		ProviderOverride: acpruntime.Provider(attempt.EffectiveRuntime.Provider),
		ProviderModels:   attemptProviderModels(snapshot, attempt), NonInteractive: true,
		RetryParentRunID: parent.RunID, RetryReason: reason,
	})
	if err != nil || runID != attempt.RunID {
		cleanupErr := registry.Update(func(candidate *producttasks.History) error {
			return removeAttemptFromHistory(candidate, task.TaskID, attempt.AttemptID)
		})
		if cleanupErr != nil {
			writeError(writer, http.StatusInternalServerError, "attempt_admission_inconsistent", fmt.Sprintf("retry admission failed: %v; cleanup failed: %v", err, cleanupErr))
			return
		}
		if err == nil {
			err = errors.New("runtime returned a different run identity")
		}
		if errors.Is(err, orchestrator.ErrRunActive) {
			writeError(writer, http.StatusConflict, "run_active", "another run is already active")
			return
		}
		writeError(writer, http.StatusBadRequest, "attempt_start_failed", err.Error())
		return
	}
	if runInfo, ok := snapshot.Service.GetRun(runID); ok {
		_ = registry.Update(func(candidate *producttasks.History) error { return updateAttemptFromRun(candidate, runInfo) })
	}
	s.watchAdmittedAttempt(snapshot.Service, registry, runID, attempt.AttemptID)
	result, _ := findAttempt(registry.Snapshot(), task.TaskID, attempt.AttemptID)
	writeJSON(writer, http.StatusAccepted, map[string]any{"attempt": result, "idempotent": false})
}

func (s *Server) watchAdmittedAttempt(service *orchestrator.Service, registry *producttasks.Registry, runID, attemptID string) {
	if service == nil || registry == nil || strings.TrimSpace(runID) == "" || strings.TrimSpace(attemptID) == "" {
		return
	}
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			info, ok := service.GetRun(runID)
			if !ok {
				return
			}
			if err := registry.Update(func(candidate *producttasks.History) error { return updateAttemptFromRun(candidate, info) }); err != nil {
				return
			}
			if info.Status != orchestrator.RunStatusQueued && info.Status != orchestrator.RunStatusRunning {
				return
			}
		}
	}()
}

func parseAttemptAdmissionOptions(rawPipeline, rawIntent string) (orchestrator.Pipeline, orchestrator.RunIntent, error) {
	pipeline := orchestrator.PipelineInit
	if strings.TrimSpace(rawPipeline) != "" {
		parsed, err := orchestrator.ParsePipeline(strings.TrimSpace(rawPipeline))
		if err != nil {
			return "", "", err
		}
		pipeline = parsed
	}
	intent := orchestrator.RunIntentStart
	if strings.TrimSpace(rawIntent) != "" {
		intent = orchestrator.RunIntent(strings.TrimSpace(rawIntent))
	}
	if intent != orchestrator.RunIntentStart && intent != orchestrator.RunIntentQueue {
		return "", "", errors.New("intent must be start or queue")
	}
	if intent == orchestrator.RunIntentQueue && pipeline != orchestrator.PipelineRefresh {
		return "", "", errors.New("queue intent is supported only for refresh attempts")
	}
	return pipeline, intent, nil
}

func validateTaskCapacity(history producttasks.History, service *orchestrator.Service, intent orchestrator.RunIntent) *attemptConflict {
	coordination := service.Coordination()
	if coordination.ActiveRunID != "" && intent == orchestrator.RunIntentStart {
		return &attemptConflict{Code: "run_active", Message: "another run is already active"}
	}
	if coordination.Pending != nil {
		return &attemptConflict{Code: "attempt_queue_full", Message: "one queued pipeline attempt already exists"}
	}
	for _, attempt := range history.Attempts {
		if attempt.Status == producttasks.AttemptQueued || attempt.Status == producttasks.AttemptRunning {
			if intent == orchestrator.RunIntentStart {
				return &attemptConflict{Code: "run_active", Message: "another pipeline attempt is already active or queued"}
			}
			if attempt.Status == producttasks.AttemptQueued {
				return &attemptConflict{Code: "attempt_queue_full", Message: "one queued pipeline attempt already exists"}
			}
		}
	}
	return nil
}

func (s *Server) buildAdmittedAttempt(snapshot serverSessionSnapshot, task producttasks.Task, pipeline orchestrator.Pipeline, intent orchestrator.RunIntent, key, fingerprint string, parentID *string, reason string) (producttasks.Attempt, error) {
	if err := task.Validate(); err != nil {
		return producttasks.Attempt{}, err
	}
	provider := strings.TrimSpace(task.DesiredRunner.Provider)
	if provider == "" {
		provider = string(snapshot.RuntimeConfig.Provider)
	}
	parsedProvider, err := acpruntime.ParseProvider(provider)
	if err != nil {
		return producttasks.Attempt{}, err
	}
	mode := strings.TrimSpace(task.DesiredRunner.Mode)
	if mode == "" {
		mode = snapshot.RuntimeConfig.Mode
	}
	if mode != acpruntime.RuntimeModeFake && mode != acpruntime.RuntimeModeHeadless {
		return producttasks.Attempt{}, fmt.Errorf("unsupported runner mode %q", mode)
	}
	if task.DesiredRunner.Mode != "" && task.DesiredRunner.Mode != snapshot.RuntimeConfig.Mode {
		return producttasks.Attempt{}, fmt.Errorf("runner mode %q is unavailable in the selected runtime", task.DesiredRunner.Mode)
	}
	execution := snapshot.Service.ResolveExecutionProfile(snapshot.Workspace.Manifest)
	steps, err := snapshot.Service.ResolveStepProviderProfile(snapshot.Workspace.Manifest)
	if err != nil {
		return producttasks.Attempt{}, err
	}
	models, err := snapshot.Service.ResolveProviderModelProfile(snapshot.Workspace.Manifest)
	if err != nil {
		return producttasks.Attempt{}, err
	}
	if task.DesiredRunner.Model != "" || task.DesiredRunner.Effort != "" {
		config := models.Effective[parsedProvider]
		if task.DesiredRunner.Model != "" {
			config.Model = task.DesiredRunner.Model
		}
		if task.DesiredRunner.Effort != "" {
			config.Effort = task.DesiredRunner.Effort
		}
		if err := acpruntime.ValidateProviderModelConfig(parsedProvider, config); err != nil {
			return producttasks.Attempt{}, err
		}
		models.Effective[parsedProvider] = config
	}
	if task.DesiredRunner.Permissions != "" && task.DesiredRunner.Permissions != acpruntime.PermissionModeTrustedFullAccess && task.DesiredRunner.Permissions != acpruntime.PermissionModeManaged {
		return producttasks.Attempt{}, fmt.Errorf("unsupported permissions mode %q", task.DesiredRunner.Permissions)
	}
	permissions := acpruntime.ResolvePermissions(snapshot.Workspace.Manifest)
	permissionMode := permissions.Effective.Mode
	if task.DesiredRunner.Permissions != "" {
		permissionMode = task.DesiredRunner.Permissions
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	runID := newOpaqueID("run")
	for snapshot.Service != nil {
		if _, exists := snapshot.Service.GetRun(runID); !exists {
			break
		}
		runID = newOpaqueID("run")
	}
	attemptID := newOpaqueID("attempt")
	stepOverrides := map[string]producttasks.RunnerPreset{}
	for key := range steps.Effective {
		stepOverrides[key] = producttasks.RunnerPreset{Preset: key, Mode: mode, Provider: string(parsedProvider)}
	}
	sources := map[string]string{
		"mode": "request", "provider": "request", "permissions": normalizeResolutionSource(string(permissions.Source.Mode)),
		"execution.strategy": normalizeResolutionSource(string(execution.Source.Strategy)), "execution.max_parallel": normalizeResolutionSource(string(execution.Source.MaxParallel)),
		"execution.failure_policy": normalizeResolutionSource(string(execution.Source.FailurePolicy)), "execution.shard_mode": normalizeResolutionSource(string(execution.Source.ShardMode)),
		"model": normalizeResolutionSource(string(models.Source[parsedProvider].Model)), "effort": normalizeResolutionSource(string(models.Source[parsedProvider].Effort)),
	}
	if task.DesiredRunner.Provider != "" || task.DesiredRunner.Mode != "" || task.DesiredRunner.Model != "" || task.DesiredRunner.Effort != "" || task.DesiredRunner.Permissions != "" {
		for _, field := range []string{"provider", "mode", "model", "effort", "permissions"} {
			if (field == "provider" && task.DesiredRunner.Provider != "") || (field == "mode" && task.DesiredRunner.Mode != "") || (field == "model" && task.DesiredRunner.Model != "") || (field == "effort" && task.DesiredRunner.Effort != "") || (field == "permissions" && task.DesiredRunner.Permissions != "") {
				sources[field] = "task_preset"
			}
		}
	}
	timeouts := acpruntime.ResolveTimeouts(snapshot.Workspace.Manifest)
	return producttasks.Attempt{
		Version: producttasks.CurrentVersion, AttemptID: attemptID, TaskID: task.TaskID, RunID: runID,
		ParentAttemptID: parentID, RetryReason: reason, Pipeline: string(pipeline), IdempotencyKey: key, RequestFingerprint: fingerprint,
		TaskRevision: task.Revision, IntentSnapshot: producttasks.IntentSnapshot{Title: task.Title, Goal: task.Goal, Context: task.Context, Scope: task.Scope, DesiredRunner: task.DesiredRunner},
		EffectiveRuntime: producttasks.EffectiveRuntime{Mode: mode, Provider: string(parsedProvider), Model: models.Effective[parsedProvider].Model, Effort: models.Effective[parsedProvider].Effort, Permissions: permissionMode, Timeouts: map[string]int{
			"step_timeout_sec": timeouts.Effective.StepTimeoutSec, "heartbeat_sec": timeouts.Effective.HeartbeatSec, "pipeline_timeout_sec": timeouts.Effective.PipelineTimeoutSec, "pipeline_kill_grace_sec": timeouts.Effective.PipelineKillGraceSec, "api_ready_timeout_sec": timeouts.Effective.APIReadyTimeoutSec, "api_init_timeout_sec": timeouts.Effective.APIInitTimeoutSec, "ui_init_poll_timeout_sec": timeouts.Effective.UIInitPollTimeoutSec, "ui_cancel_poll_timeout_sec": timeouts.Effective.UICancelPollTimeoutSec,
		}, Scope: task.Scope, Execution: producttasks.ExecutionSettings{Strategy: execution.Effective.Strategy, MaxParallel: execution.Effective.MaxParallel, FailurePolicy: execution.Effective.FailurePolicy, ShardMode: execution.Effective.ShardMode}, StepOverrides: stepOverrides, ResolutionSources: sources},
		Status: producttasks.AttemptQueued, AdmittedAt: now, QueuedAt: &now, RetainedEvidence: producttasks.EvidenceUnavailable,
		Outcome: producttasks.Outcome{State: producttasks.Unavailable, UnavailableReason: "attempt has not completed"}, Publication: producttasks.Publication{State: producttasks.PublicationUnavailable, UnavailableReason: "attempt has not been published"},
	}, nil
}

func normalizeResolutionSource(source string) string {
	if source == "default" || source == "" {
		return "provider_default"
	}
	if source == "override" {
		return "request"
	}
	return source
}

func attemptProviderModels(snapshot serverSessionSnapshot, attempt producttasks.Attempt) *acpruntime.ProviderModelResolution {
	resolved, err := snapshot.Service.ResolveProviderModelProfile(snapshot.Workspace.Manifest)
	if err != nil {
		return nil
	}
	provider := acpruntime.Provider(attempt.EffectiveRuntime.Provider)
	config := resolved.Effective[provider]
	config.Model, config.Effort = attempt.EffectiveRuntime.Model, attempt.EffectiveRuntime.Effort
	resolved.Effective[provider] = config
	return &resolved
}

func validateTaskRepositoryScope(task producttasks.Task, ws workspace.Root) error {
	known := map[string]bool{}
	for _, repo := range ws.Manifest.Repos {
		known[repo.Name] = true
	}
	for _, repo := range task.Scope.Repositories {
		if !known[repo.Name] {
			return fmt.Errorf("task scope repository %q is not present in workspace manifest", repo.Name)
		}
	}
	return nil
}

func appendAttemptToHistory(history *producttasks.History, taskID string, attempt producttasks.Attempt) error {
	if _, exists := findAttempt(*history, taskID, attempt.AttemptID); exists {
		return errors.New("attempt already exists")
	}
	index := indexTask(history.Tasks, taskID)
	if index < 0 {
		return errTaskNotFound
	}
	history.Attempts = append(history.Attempts, attempt)
	now := attempt.AdmittedAt
	history.Tasks[index].Attempts = append(history.Tasks[index].Attempts, producttasks.AttemptSummary{AttemptID: attempt.AttemptID, RunID: attempt.RunID, ParentAttemptID: attempt.ParentAttemptID, TaskRevision: attempt.TaskRevision, Status: attempt.Status, AdmittedAt: attempt.AdmittedAt, UpdatedAt: now, RetainedEvidence: attempt.RetainedEvidence})
	history.Tasks[index].LastActivityAt = now
	return nil
}

func removeAttemptFromHistory(history *producttasks.History, taskID, attemptID string) error {
	index := -1
	for i, attempt := range history.Attempts {
		if attempt.TaskID == taskID && attempt.AttemptID == attemptID {
			index = i
			break
		}
	}
	if index < 0 {
		return nil
	}
	history.Attempts = append(history.Attempts[:index], history.Attempts[index+1:]...)
	taskIndex := indexTask(history.Tasks, taskID)
	if taskIndex >= 0 {
		filtered := history.Tasks[taskIndex].Attempts[:0]
		for _, summary := range history.Tasks[taskIndex].Attempts {
			if summary.AttemptID != attemptID {
				filtered = append(filtered, summary)
			}
		}
		history.Tasks[taskIndex].Attempts = filtered
	}
	return nil
}

func updateAttemptFromRun(history *producttasks.History, info orchestrator.RunInfo) error {
	for index := range history.Attempts {
		attempt := &history.Attempts[index]
		if attempt.RunID != info.RunID || attempt.AttemptID != info.AttemptID {
			continue
		}
		attempt.Status = attemptStatusFromRun(info.Status)
		if attempt.Status == producttasks.AttemptRunning && attempt.StartedAt == nil {
			value := info.StartedAt.UTC().Format(time.RFC3339Nano)
			attempt.StartedAt = &value
		}
		if info.FinishedAt != nil {
			value := info.FinishedAt.UTC().Format(time.RFC3339Nano)
			attempt.FinishedAt = &value
			attempt.TerminalSummary = &producttasks.TerminalSummary{Status: attempt.Status, ErrorCode: info.ErrorCode, Error: info.Error, RetainedEvidence: attempt.RetainedEvidence}
			if attempt.Status == producttasks.AttemptSucceeded {
				attempt.Outcome = producttasks.Outcome{State: producttasks.Available, AttemptID: attempt.AttemptID, RunID: attempt.RunID, SnapshotPath: fmt.Sprintf("reports/taskruns/%s", attempt.RunID)}
			} else {
				attempt.Outcome = producttasks.Outcome{State: producttasks.Unavailable, UnavailableReason: "terminal run did not produce a successful outcome"}
			}
		}
		for taskIndex := range history.Tasks {
			if history.Tasks[taskIndex].TaskID != attempt.TaskID {
				continue
			}
			for summaryIndex := range history.Tasks[taskIndex].Attempts {
				if history.Tasks[taskIndex].Attempts[summaryIndex].AttemptID == attempt.AttemptID {
					summary := &history.Tasks[taskIndex].Attempts[summaryIndex]
					summary.Status = attempt.Status
					summary.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
					summary.FinishedAt = attempt.FinishedAt
					history.Tasks[taskIndex].LastActivityAt = summary.UpdatedAt
					if attempt.Status == producttasks.AttemptSucceeded {
						history.Tasks[taskIndex].Outcome = attempt.Outcome
					} else if isTerminalAttempt(attempt.Status) {
						history.Tasks[taskIndex].Outcome = producttasks.Outcome{State: producttasks.Unavailable, UnavailableReason: "latest attempt did not produce a successful outcome"}
					}
				}
			}
		}
		return nil
	}
	return errors.New("attempt run linkage not found")
}

func attemptStatusFromRun(status orchestrator.RunStatus) producttasks.AttemptStatus {
	switch status {
	case orchestrator.RunStatusQueued:
		return producttasks.AttemptQueued
	case orchestrator.RunStatusRunning:
		return producttasks.AttemptRunning
	case orchestrator.RunStatusSucceeded:
		return producttasks.AttemptSucceeded
	case orchestrator.RunStatusFailed:
		return producttasks.AttemptFailed
	case orchestrator.RunStatusCanceled:
		return producttasks.AttemptCanceled
	default:
		return producttasks.AttemptRunning
	}
}

func isTerminalAttempt(status producttasks.AttemptStatus) bool {
	return status == producttasks.AttemptSucceeded || status == producttasks.AttemptFailed || status == producttasks.AttemptCanceled || status == producttasks.AttemptTimeout
}

func findAttempt(history producttasks.History, taskID, attemptID string) (producttasks.Attempt, bool) {
	for _, attempt := range history.Attempts {
		if attempt.TaskID == taskID && attempt.AttemptID == attemptID {
			return producttasks.CloneAttempt(attempt), true
		}
	}
	return producttasks.Attempt{}, false
}

func findIdempotentAttempt(history producttasks.History, taskID, key string) (producttasks.Attempt, bool) {
	for _, attempt := range history.Attempts {
		if attempt.TaskID == taskID && attempt.IdempotencyKey == key {
			return producttasks.CloneAttempt(attempt), true
		}
	}
	return producttasks.Attempt{}, false
}

func attemptFingerprint(value attemptAdmissionFingerprint) string {
	raw, _ := json.Marshal(value)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}
