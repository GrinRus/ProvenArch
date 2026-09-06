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
	"github.com/GrinRus/ProvenArch/internal/pathscope"
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
	if taskHasAttempts(history, task.TaskID) {
		writeError(writer, http.StatusConflict, "attempt_action_required", "the Task already has an Attempt; use retry or rerun on a terminal Attempt")
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
	runtimeSnapshot, err := attemptRuntimeSnapshot(attempt)
	if err != nil {
		_ = registry.Update(func(candidate *producttasks.History) error {
			return removeAttemptFromHistory(candidate, task.TaskID, attempt.AttemptID)
		})
		writeError(writer, http.StatusBadRequest, "attempt_start_failed", err.Error())
		return
	}
	runID, err := snapshot.Service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace: snapshot.Workspace, Pipeline: pipeline, Intent: intent,
		TaskID: task.TaskID, AttemptID: attempt.AttemptID, RequestedRunID: attempt.RunID,
		RuntimeSnapshot: runtimeSnapshot, NonInteractive: true,
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

func (s *Server) handleTaskAttemptChild(writer http.ResponseWriter, request *http.Request, taskID, attemptID, action string) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	if action != "retry" && action != "rerun" {
		writeError(writer, http.StatusNotFound, "task_route_not_found", "task attempt action not found")
		return
	}
	var payload taskAttemptRetryRequest
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid child attempt request")
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
		writeError(writer, http.StatusPreconditionRequired, "workspace_not_selected", "select a workspace and runner before admitting a child attempt")
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
	if task.Lifecycle == producttasks.LifecycleArchived {
		writeError(writer, http.StatusConflict, "task_archived", "archived tasks cannot admit child attempts")
		return
	}
	parent, ok := findAttempt(history, taskID, attemptID)
	if !ok {
		writeError(writer, http.StatusNotFound, "attempt_not_found", "attempt not found for task")
		return
	}
	if action == "retry" {
		if !isRetryableAttempt(parent.Status) {
			if !isTerminalAttempt(parent.Status) {
				writeError(writer, http.StatusConflict, "retry_parent_not_terminal", "retry is available only after the parent Attempt reaches a terminal state")
			} else {
				writeError(writer, http.StatusConflict, "retry_parent_not_retryable", "retry is available only after a failed, canceled or timed-out Attempt")
			}
			return
		}
	} else if parent.Status != producttasks.AttemptSucceeded {
		writeError(writer, http.StatusConflict, "rerun_parent_not_succeeded", "rerun is available only after a successful Attempt")
		return
	}
	if strings.TrimSpace(payload.Pipeline) == "" && parent.Pipeline != "" {
		pipeline, intent, err = parseAttemptAdmissionOptions(parent.Pipeline, payload.Intent)
		if err != nil {
			writeError(writer, http.StatusBadRequest, "attempt_options_invalid", err.Error())
			return
		}
	}
	if err := validateTaskRepositoryScope(task, snapshot.Workspace); err != nil {
		writeError(writer, http.StatusBadRequest, "task_scope_invalid", err.Error())
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		if action == "rerun" {
			reason = "operator_rerun"
		} else {
			reason = "operator_retry"
		}
	}
	fingerprint := attemptFingerprint(attemptAdmissionFingerprint{TaskID: task.TaskID, TaskRevision: task.Revision, Pipeline: string(pipeline), Intent: string(intent), ParentAttemptID: parent.AttemptID, RetryReason: reason})
	if existing, found := findIdempotentAttempt(history, task.TaskID, key); found {
		if existing.RequestFingerprint != fingerprint {
			writeError(writer, http.StatusConflict, "idempotency_conflict", "idempotency_key was already used for a different child attempt request")
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
	runtimeSnapshot, err := attemptRuntimeSnapshot(attempt)
	if err != nil {
		_ = registry.Update(func(candidate *producttasks.History) error {
			return removeAttemptFromHistory(candidate, task.TaskID, attempt.AttemptID)
		})
		writeError(writer, http.StatusBadRequest, "attempt_start_failed", err.Error())
		return
	}
	runID, err := snapshot.Service.StartAsyncRun(context.Background(), orchestrator.RunRequest{
		Workspace: snapshot.Workspace, Pipeline: pipeline, Intent: intent,
		TaskID: task.TaskID, AttemptID: attempt.AttemptID, RequestedRunID: attempt.RunID,
		RuntimeSnapshot: runtimeSnapshot, NonInteractive: true,
		RetryParentRunID: parent.RunID, RetryReason: reason,
	})
	if err != nil || runID != attempt.RunID {
		cleanupErr := registry.Update(func(candidate *producttasks.History) error {
			return removeAttemptFromHistory(candidate, task.TaskID, attempt.AttemptID)
		})
		if cleanupErr != nil {
			writeError(writer, http.StatusInternalServerError, "attempt_admission_inconsistent", fmt.Sprintf("%s admission failed: %v; cleanup failed: %v", action, err, cleanupErr))
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

func isRetryableAttempt(status producttasks.AttemptStatus) bool {
	return status == producttasks.AttemptFailed || status == producttasks.AttemptCanceled || status == producttasks.AttemptTimeout
}

func taskHasAttempts(history producttasks.History, taskID string) bool {
	for _, attempt := range history.Attempts {
		if attempt.TaskID == taskID {
			return true
		}
	}
	return false
}

func (s *Server) watchAdmittedAttempt(service *orchestrator.Service, registry *producttasks.Registry, runID, attemptID string) {
	if service == nil || registry == nil || strings.TrimSpace(runID) == "" || strings.TrimSpace(attemptID) == "" {
		return
	}
	s.attemptWatchMu.Lock()
	if s.attemptWatchClosed || s.attemptWatchCtx == nil {
		s.attemptWatchMu.Unlock()
		return
	}
	watchCtx := s.attemptWatchCtx
	s.attemptWatchWG.Add(1)
	s.attemptWatchMu.Unlock()
	go func() {
		defer s.attemptWatchWG.Done()
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
			}
			info, ok := service.GetRun(runID)
			if !ok {
				return
			}
			if watchCtx.Err() != nil {
				return
			}
			history := registry.Snapshot()
			if !attemptNeedsRunUpdate(history, info) {
				if info.Status != orchestrator.RunStatusQueued && info.Status != orchestrator.RunStatusRunning {
					return
				}
				continue
			}
			if err := registry.Update(func(candidate *producttasks.History) error { return updateAttemptFromRun(candidate, info) }); err != nil {
				// A transient current/last-good write failure must not permanently
				// abandon the watcher; the next state transition retries the publish.
				if watchCtx.Err() != nil {
					return
				}
				continue
			}
			if info.Status != orchestrator.RunStatusQueued && info.Status != orchestrator.RunStatusRunning {
				return
			}
		}
	}()
}

func (s *Server) reconcileTaskAttemptsAfterRestart(service *orchestrator.Service, registry *producttasks.Registry) {
	if service == nil || registry == nil {
		return
	}
	for _, attempt := range registry.Snapshot().Attempts {
		if attempt.Status != producttasks.AttemptQueued && attempt.Status != producttasks.AttemptRunning {
			continue
		}
		info, ok := service.GetRun(attempt.RunID)
		if !ok {
			// The runtime history may have been retained less long than Task
			// history. Do not leave a durable queued/running identity forever;
			// expose an explicit unavailable terminal state instead.
			_ = registry.Update(func(candidate *producttasks.History) error {
				return markAttemptUnavailableAfterRestart(candidate, attempt.TaskID, attempt.AttemptID)
			})
			continue
		}
		if attemptNeedsRunUpdate(registry.Snapshot(), info) {
			_ = registry.Update(func(candidate *producttasks.History) error { return updateAttemptFromRun(candidate, info) })
		}
		if info.Status == orchestrator.RunStatusQueued || info.Status == orchestrator.RunStatusRunning {
			s.watchAdmittedAttempt(service, registry, attempt.RunID, attempt.AttemptID)
		}
	}
}

func markAttemptUnavailableAfterRestart(history *producttasks.History, taskID, attemptID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for index := range history.Attempts {
		attempt := &history.Attempts[index]
		if attempt.TaskID != taskID || attempt.AttemptID != attemptID {
			continue
		}
		attempt.Status = producttasks.AttemptFailed
		attempt.FinishedAt = &now
		attempt.TerminalSummary = &producttasks.TerminalSummary{
			Status:           producttasks.AttemptFailed,
			ErrorCode:        "run_reconciled_after_restart",
			Error:            "runtime run identity was not found after restart",
			RetainedEvidence: attempt.RetainedEvidence,
		}
		attempt.Outcome = producttasks.Outcome{State: producttasks.Unavailable, UnavailableReason: "runtime run identity was not found after restart"}
		for taskIndex := range history.Tasks {
			if history.Tasks[taskIndex].TaskID != taskID {
				continue
			}
			for summaryIndex := range history.Tasks[taskIndex].Attempts {
				summary := &history.Tasks[taskIndex].Attempts[summaryIndex]
				if summary.AttemptID != attemptID {
					continue
				}
				summary.Status = producttasks.AttemptFailed
				summary.UpdatedAt = now
				summary.FinishedAt = &now
				history.Tasks[taskIndex].LastActivityAt = now
				history.Tasks[taskIndex].Outcome = attempt.Outcome
				return nil
			}
		}
		return nil
	}
	return errors.New("attempt run linkage not found")
}

func attemptNeedsRunUpdate(history producttasks.History, info orchestrator.RunInfo) bool {
	for _, attempt := range history.Attempts {
		if attempt.RunID != info.RunID || attempt.AttemptID != info.AttemptID {
			continue
		}
		status := attemptStatusFromRun(info.Status)
		if attempt.Status != status || (info.FinishedAt != nil && attempt.FinishedAt == nil) || (status == producttasks.AttemptRunning && attempt.StartedAt == nil) {
			return true
		}
		if info.FinishedAt != nil && attempt.FinishedAt != nil && *attempt.FinishedAt != info.FinishedAt.UTC().Format(time.RFC3339Nano) {
			return true
		}
		return false
	}
	return false
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
	for provider, source := range models.Source {
		modelSource := normalizeResolutionSource(string(source.Model))
		effortSource := normalizeResolutionSource(string(source.Effort))
		if provider == parsedProvider {
			if task.DesiredRunner.Model != "" {
				modelSource = "task_preset"
			}
			if task.DesiredRunner.Effort != "" {
				effortSource = "task_preset"
			}
		}
		sources["provider."+string(provider)+".model"] = modelSource
		sources["provider."+string(provider)+".effort"] = effortSource
	}
	taskProviderOverride := strings.TrimSpace(task.DesiredRunner.Provider) != ""
	stepOverrides := map[string]producttasks.RunnerPreset{}
	for key, resolvedProvider := range steps.Effective {
		providerForStep := resolvedProvider
		providerSource := normalizeResolutionSource(string(steps.Source[key]))
		if taskProviderOverride {
			providerForStep = parsedProvider
			providerSource = "task_preset"
		}
		config := models.Effective[providerForStep]
		stepOverrides[key] = producttasks.RunnerPreset{Preset: key, Mode: mode, Provider: string(providerForStep), Model: config.Model, Effort: config.Effort}
		sources["step."+key+".provider"] = providerSource
	}
	timeouts := acpruntime.ResolveTimeouts(snapshot.Workspace.Manifest)
	return producttasks.Attempt{
		Version: producttasks.CurrentVersion, AttemptID: attemptID, TaskID: task.TaskID, RunID: runID,
		ParentAttemptID: parentID, RetryReason: reason, Pipeline: string(pipeline), IdempotencyKey: key, RequestFingerprint: fingerprint,
		TaskRevision: task.Revision, IntentSnapshot: taskIntentSnapshot(task),
		EffectiveRuntime: producttasks.EffectiveRuntime{Mode: mode, Provider: string(parsedProvider), Model: models.Effective[parsedProvider].Model, Effort: models.Effective[parsedProvider].Effort, Permissions: permissionMode, Timeouts: map[string]int{
			"step_timeout_sec": timeouts.Effective.StepTimeoutSec, "heartbeat_sec": timeouts.Effective.HeartbeatSec, "pipeline_timeout_sec": timeouts.Effective.PipelineTimeoutSec, "pipeline_kill_grace_sec": timeouts.Effective.PipelineKillGraceSec, "api_ready_timeout_sec": timeouts.Effective.APIReadyTimeoutSec, "api_init_timeout_sec": timeouts.Effective.APIInitTimeoutSec, "ui_init_poll_timeout_sec": timeouts.Effective.UIInitPollTimeoutSec, "ui_cancel_poll_timeout_sec": timeouts.Effective.UICancelPollTimeoutSec,
		}, Scope: cloneTaskScope(task.Scope), Execution: producttasks.ExecutionSettings{Strategy: execution.Effective.Strategy, MaxParallel: execution.Effective.MaxParallel, FailurePolicy: execution.Effective.FailurePolicy, ShardMode: execution.Effective.ShardMode}, StepOverrides: stepOverrides, ResolutionSources: sources},
		Status: producttasks.AttemptQueued, AdmittedAt: now, QueuedAt: &now, RetainedEvidence: producttasks.EvidenceUnavailable,
		Outcome: producttasks.Outcome{State: producttasks.Unavailable, UnavailableReason: "attempt has not completed"}, Publication: producttasks.Publication{State: producttasks.PublicationUnavailable, UnavailableReason: "attempt has not been published"},
	}, nil
}

// taskIntentSnapshot is the explicit inheritance allowlist for a future
// Attempt. Task lifecycle, revision, timestamps, prior attempts, outcomes and
// publication state are aggregate metadata and must never leak into a child
// Attempt snapshot.
func taskIntentSnapshot(task producttasks.Task) producttasks.IntentSnapshot {
	return producttasks.IntentSnapshot{
		Title:         task.Title,
		Goal:          task.Goal,
		Context:       task.Context,
		Scope:         cloneTaskScope(task.Scope),
		DesiredRunner: task.DesiredRunner,
	}
}

func cloneTaskScope(scope producttasks.Scope) producttasks.Scope {
	repositories := make([]producttasks.RepositoryScope, len(scope.Repositories))
	for index, repository := range scope.Repositories {
		repositories[index] = producttasks.RepositoryScope{Name: repository.Name, Paths: append([]string(nil), repository.Paths...)}
	}
	return producttasks.Scope{Repositories: repositories}
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

func attemptRuntimeSnapshot(attempt producttasks.Attempt) (*acpruntime.AdmittedRuntimeSnapshot, error) {
	provider, err := acpruntime.ParseProvider(attempt.EffectiveRuntime.Provider)
	if err != nil {
		return nil, fmt.Errorf("admitted attempt provider is invalid: %w", err)
	}
	stepProviders := acpruntime.StepProviderValues{}
	stepSources := acpruntime.StepProviderSources{}
	for stepID, preset := range attempt.EffectiveRuntime.StepOverrides {
		stepID = strings.TrimSpace(stepID)
		if stepID == "" {
			return nil, errors.New("admitted attempt contains an empty step provider identity")
		}
		stepProvider, parseErr := acpruntime.ParseProvider(preset.Provider)
		if parseErr != nil {
			return nil, fmt.Errorf("admitted attempt step %q provider is invalid: %w", stepID, parseErr)
		}
		stepProviders[stepID] = stepProvider
		stepSources[stepID] = admittedStepProviderSource(attempt.EffectiveRuntime.ResolutionSources["step."+stepID+".provider"], attempt.EffectiveRuntime.ResolutionSources["provider"])
	}
	if len(stepProviders) > 0 {
		for _, stepID := range []string{
			acpruntime.StepProviderStep0Constitution,
			acpruntime.StepProviderStep1Collect,
			acpruntime.StepProviderStep2AsIs,
			acpruntime.StepProviderStep3Findings,
			acpruntime.StepProviderStep4Proposals,
			acpruntime.StepProviderQA,
		} {
			if _, exists := stepProviders[stepID]; !exists {
				return nil, fmt.Errorf("admitted attempt is missing provider identity for step %q", stepID)
			}
		}
	}
	// Attempts written before per-step provenance was introduced still have a
	// durable global provider. Reconstruct those steps from that immutable value
	// rather than falling back to mutable workspace/env resolution.
	if len(stepProviders) == 0 {
		for _, stepID := range []string{
			acpruntime.StepProviderStep0Constitution,
			acpruntime.StepProviderStep1Collect,
			acpruntime.StepProviderStep2AsIs,
			acpruntime.StepProviderStep3Findings,
			acpruntime.StepProviderStep4Proposals,
		} {
			stepProviders[stepID] = provider
			stepSources[stepID] = admittedStepProviderSource(attempt.EffectiveRuntime.ResolutionSources["provider"], "")
		}
	}
	repositoryScopes := make([]string, 0, len(attempt.EffectiveRuntime.Scope.Repositories))
	repositoryPathScopes := make(map[string][]string, len(attempt.EffectiveRuntime.Scope.Repositories))
	for _, repository := range attempt.EffectiveRuntime.Scope.Repositories {
		name := strings.TrimSpace(repository.Name)
		if name != "" {
			repositoryScopes = append(repositoryScopes, name)
			paths := append([]string(nil), repository.Paths...)
			for index, rawPath := range paths {
				pathValue := strings.TrimSpace(rawPath)
				if _, compileErr := pathscope.Compile(pathValue); compileErr != nil {
					return nil, fmt.Errorf("admitted attempt repository %q path %d is invalid: %w", name, index, compileErr)
				}
				paths[index] = pathValue
			}
			repositoryPathScopes[name] = paths
		}
	}
	models := acpruntime.ProviderModelValues{provider: {Model: attempt.EffectiveRuntime.Model, Effort: attempt.EffectiveRuntime.Effort}}
	modelSources := acpruntime.ProviderModelSources{provider: {
		Model:  admittedProviderModelSource(attempt.EffectiveRuntime.ResolutionSources["provider."+string(provider)+".model"], attempt.EffectiveRuntime.ResolutionSources["model"]),
		Effort: admittedProviderModelSource(attempt.EffectiveRuntime.ResolutionSources["provider."+string(provider)+".effort"], attempt.EffectiveRuntime.ResolutionSources["effort"]),
	}}
	for _, stepPreset := range attempt.EffectiveRuntime.StepOverrides {
		stepProvider, parseErr := acpruntime.ParseProvider(stepPreset.Provider)
		if parseErr != nil {
			return nil, fmt.Errorf("admitted attempt step provider is invalid: %w", parseErr)
		}
		if _, exists := models[stepProvider]; !exists {
			models[stepProvider] = acpruntime.ProviderModelConfig{Model: stepPreset.Model, Effort: stepPreset.Effort}
			modelSources[stepProvider] = acpruntime.ProviderModelFieldSources{
				Model:  admittedProviderModelSource(attempt.EffectiveRuntime.ResolutionSources["provider."+string(stepProvider)+".model"], ""),
				Effort: admittedProviderModelSource(attempt.EffectiveRuntime.ResolutionSources["provider."+string(stepProvider)+".effort"], ""),
			}
		}
	}
	return &acpruntime.AdmittedRuntimeSnapshot{
		Mode:                 attempt.EffectiveRuntime.Mode,
		StepProviders:        stepProviders,
		StepProviderSources:  stepSources,
		ProviderModels:       models,
		ProviderModelSources: modelSources,
		Execution:            acpruntime.ExecutionValues{Strategy: attempt.EffectiveRuntime.Execution.Strategy, MaxParallel: attempt.EffectiveRuntime.Execution.MaxParallel, FailurePolicy: attempt.EffectiveRuntime.Execution.FailurePolicy, ShardMode: attempt.EffectiveRuntime.Execution.ShardMode},
		Permissions:          acpruntime.PermissionValues{Mode: attempt.EffectiveRuntime.Permissions, ApprovalChannel: acpruntime.PermissionApprovalFailFast},
		Timeouts:             acpruntime.TimeoutValues{StepTimeoutSec: attempt.EffectiveRuntime.Timeouts["step_timeout_sec"], HeartbeatSec: attempt.EffectiveRuntime.Timeouts["heartbeat_sec"], PipelineTimeoutSec: attempt.EffectiveRuntime.Timeouts["pipeline_timeout_sec"], PipelineKillGraceSec: attempt.EffectiveRuntime.Timeouts["pipeline_kill_grace_sec"], APIReadyTimeoutSec: attempt.EffectiveRuntime.Timeouts["api_ready_timeout_sec"], APIInitTimeoutSec: attempt.EffectiveRuntime.Timeouts["api_init_timeout_sec"], UIInitPollTimeoutSec: attempt.EffectiveRuntime.Timeouts["ui_init_poll_timeout_sec"], UICancelPollTimeoutSec: attempt.EffectiveRuntime.Timeouts["ui_cancel_poll_timeout_sec"]},
		RepositoryScopes:     repositoryScopes,
		RepositoryPathScopes: repositoryPathScopes,
	}, nil
}

func admittedStepProviderSource(source, fallback string) acpruntime.ProviderSource {
	if source == "" {
		source = fallback
	}
	switch source {
	case "workspace":
		return acpruntime.ProviderSourceWorkspace
	case "env":
		return acpruntime.ProviderSourceEnv
	case "task_preset":
		return acpruntime.ProviderSourceTaskPreset
	case "request", "override":
		return acpruntime.ProviderSourceOverride
	default:
		return acpruntime.ProviderSourceDefault
	}
}

func admittedProviderModelSource(source, fallback string) acpruntime.ProviderModelSource {
	if source == "" {
		source = fallback
	}
	switch source {
	case "env":
		return acpruntime.ProviderModelSourceEnv
	case "workspace":
		return acpruntime.ProviderModelSourceWorkspace
	case "task_preset", "request":
		return acpruntime.ProviderModelSourceTaskPreset
	default:
		return acpruntime.ProviderModelSourceDefault
	}
}

func validateTaskRepositoryScope(task producttasks.Task, ws workspace.Root) error {
	known := map[string]bool{}
	for _, repo := range ws.Manifest.Repos {
		known[repo.Name] = true
	}
	for _, repo := range task.Scope.Repositories {
		name := strings.TrimSpace(repo.Name)
		if !known[name] {
			return fmt.Errorf("task scope repository %q is not present in workspace manifest", name)
		}
		for index, rawPath := range repo.Paths {
			pathValue := strings.TrimSpace(rawPath)
			if _, err := pathscope.Compile(pathValue); err != nil {
				return fmt.Errorf("task scope repository %q path %d is invalid: %w", name, index, err)
			}
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
