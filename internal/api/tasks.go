package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
)

type taskCreateRequest struct {
	Title         string                    `json:"title"`
	Goal          string                    `json:"goal"`
	Context       string                    `json:"context,omitempty"`
	Scope         producttasks.Scope        `json:"scope"`
	DesiredRunner producttasks.RunnerPreset `json:"desired_runner"`
}

type taskPatchRequest struct {
	ExpectedRevision *int                       `json:"expected_revision"`
	Title            *string                    `json:"title"`
	Goal             *string                    `json:"goal"`
	Context          *string                    `json:"context"`
	Scope            *producttasks.Scope        `json:"scope"`
	DesiredRunner    *producttasks.RunnerPreset `json:"desired_runner"`
}

type taskArchiveRequest struct {
	ExpectedRevision *int `json:"expected_revision"`
}

type taskCursor struct {
	LastActivityAt string `json:"last_activity_at"`
	TaskID         string `json:"task_id"`
}

func (s *Server) handleTasks(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/api/tasks")
	path = strings.Trim(path, "/")
	if path == "" {
		switch request.Method {
		case http.MethodGet:
			s.handleTaskList(writer, request)
		case http.MethodPost:
			s.handleTaskCreate(writer, request)
		default:
			writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPost)
		}
		return
	}
	parts := strings.Split(path, "/")
	taskID := strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		s.handleTaskResource(writer, request, taskID)
		return
	}
	if len(parts) == 2 && (parts[1] == "archive" || parts[1] == "unarchive") {
		s.handleTaskArchive(writer, request, taskID, parts[1] == "archive")
		return
	}
	if parts[1] != "attempts" {
		writeError(writer, http.StatusNotFound, "task_route_not_found", "task route not found")
		return
	}
	if len(parts) == 2 {
		s.handleTaskAttempts(writer, request, taskID, "")
		return
	}
	if len(parts) == 3 {
		if strings.TrimSpace(parts[2]) == "retry" {
			s.handleTaskAttemptRetry(writer, request, taskID, "")
			return
		}
		s.handleTaskAttempts(writer, request, taskID, strings.TrimSpace(parts[2]))
		return
	}
	if len(parts) == 4 && strings.TrimSpace(parts[3]) == "retry" {
		s.handleTaskAttemptRetry(writer, request, taskID, strings.TrimSpace(parts[2]))
		return
	}
	writeError(writer, http.StatusNotFound, "task_route_not_found", "task route not found")
}

func (s *Server) handleTaskCreate(writer http.ResponseWriter, request *http.Request) {
	var payload taskCreateRequest
	if err := decodeStrictJSON(request, &payload); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "invalid task create request")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	task := producttasks.Task{
		Version:        producttasks.CurrentVersion,
		TaskID:         newOpaqueID("task"),
		Title:          payload.Title,
		Goal:           payload.Goal,
		Context:        payload.Context,
		Scope:          payload.Scope,
		DesiredRunner:  payload.DesiredRunner,
		Lifecycle:      producttasks.LifecycleOpen,
		Revision:       1,
		CreatedAt:      now,
		UpdatedAt:      now,
		LastActivityAt: now,
		Attempts:       []producttasks.AttemptSummary{},
		Outcome: producttasks.Outcome{
			State:             producttasks.Unavailable,
			UnavailableReason: "no attempt has completed",
		},
		Publication: producttasks.Publication{
			State:             producttasks.PublicationUnavailable,
			UnavailableReason: "no publication has been recorded",
		},
	}
	if err := task.Validate(); err != nil {
		writeError(writer, http.StatusBadRequest, "task_invalid", err.Error())
		return
	}
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	registry, err := s.taskRegistrySnapshot()
	if err != nil {
		writeTaskHistoryUnavailable(writer, err)
		return
	}
	if err := registry.Update(func(history *producttasks.History) error {
		history.Tasks = append(history.Tasks, task)
		return nil
	}); err != nil {
		writeError(writer, http.StatusInternalServerError, "task_persist_failed", err.Error())
		return
	}
	writeJSON(writer, http.StatusCreated, map[string]any{"task": task})
}

func (s *Server) handleTaskList(writer http.ResponseWriter, request *http.Request) {
	registry, err := s.taskRegistrySnapshot()
	if err != nil {
		writeTaskHistoryUnavailable(writer, err)
		return
	}
	limit, err := parseTaskLimit(request.URL.Query().Get("limit"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_limit", err.Error())
		return
	}
	cursor, err := decodeTaskCursor(request.URL.Query().Get("cursor"))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_cursor", err.Error())
		return
	}
	from, to, err := parseTaskTimeFilter(request)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_time_filter", err.Error())
		return
	}
	lifecycle := strings.TrimSpace(request.URL.Query().Get("lifecycle"))
	if lifecycle != "" && lifecycle != string(producttasks.LifecycleOpen) && lifecycle != string(producttasks.LifecycleArchived) {
		writeError(writer, http.StatusBadRequest, "invalid_lifecycle", "lifecycle must be open or archived")
		return
	}
	tasks := registry.Snapshot().Tasks
	filtered := make([]producttasks.Task, 0, len(tasks))
	for _, task := range tasks {
		if lifecycle != "" && string(task.Lifecycle) != lifecycle {
			continue
		}
		if runner := strings.TrimSpace(request.URL.Query().Get("runner")); runner != "" && task.DesiredRunner.Preset != runner && task.DesiredRunner.Provider != runner {
			continue
		}
		if repository := strings.TrimSpace(request.URL.Query().Get("repository")); repository != "" && !taskHasRepository(task, repository) {
			continue
		}
		lastActivity, parseErr := time.Parse(time.RFC3339Nano, task.LastActivityAt)
		if parseErr != nil {
			continue
		}
		if from != nil && lastActivity.Before(*from) {
			continue
		}
		if to != nil && lastActivity.After(*to) {
			continue
		}
		if cursor != nil && !afterTaskCursor(task, *cursor) {
			continue
		}
		filtered = append(filtered, task)
	}
	sort.SliceStable(filtered, func(left, right int) bool {
		leftTime, _ := time.Parse(time.RFC3339Nano, filtered[left].LastActivityAt)
		rightTime, _ := time.Parse(time.RFC3339Nano, filtered[right].LastActivityAt)
		if leftTime.Equal(rightTime) {
			return filtered[left].TaskID < filtered[right].TaskID
		}
		return leftTime.After(rightTime)
	})
	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}
	response := map[string]any{
		"items":       filtered,
		"next_cursor": "",
		"has_more":    hasMore,
		"diagnostics": registry.Diagnostics(),
	}
	if hasMore && len(filtered) > 0 {
		last := filtered[len(filtered)-1]
		response["next_cursor"] = encodeTaskCursor(taskCursor{LastActivityAt: last.LastActivityAt, TaskID: last.TaskID})
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleTaskResource(writer http.ResponseWriter, request *http.Request, taskID string) {
	switch request.Method {
	case http.MethodGet:
		registry, err := s.taskRegistrySnapshot()
		if err != nil {
			writeTaskHistoryUnavailable(writer, err)
			return
		}
		task, ok := findTask(registry.Snapshot(), taskID)
		if !ok {
			writeError(writer, http.StatusNotFound, "task_not_found", "task not found")
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"task": task})
	case http.MethodPatch:
		var payload taskPatchRequest
		if err := decodeStrictJSON(request, &payload); err != nil || payload.ExpectedRevision == nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body", "expected_revision and a valid task patch are required")
			return
		}
		s.admissionMu.Lock()
		defer s.admissionMu.Unlock()
		registry, err := s.taskRegistrySnapshot()
		if err != nil {
			writeTaskHistoryUnavailable(writer, err)
			return
		}
		var updated producttasks.Task
		err = registry.Update(func(history *producttasks.History) error {
			index := indexTask(history.Tasks, taskID)
			if index < 0 {
				return errTaskNotFound
			}
			task := &history.Tasks[index]
			if task.Revision != *payload.ExpectedRevision {
				return &taskRevisionConflict{Expected: *payload.ExpectedRevision, Actual: task.Revision}
			}
			if payload.Title != nil {
				task.Title = *payload.Title
			}
			if payload.Goal != nil {
				task.Goal = *payload.Goal
			}
			if payload.Context != nil {
				task.Context = *payload.Context
			}
			if payload.Scope != nil {
				task.Scope = *payload.Scope
			}
			if payload.DesiredRunner != nil {
				task.DesiredRunner = *payload.DesiredRunner
			}
			if payload.Title == nil && payload.Goal == nil && payload.Context == nil && payload.Scope == nil && payload.DesiredRunner == nil {
				return errors.New("task patch must change at least one desired field")
			}
			task.Revision++
			task.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			task.LastActivityAt = task.UpdatedAt
			updated = producttasks.CloneTask(*task)
			return nil
		})
		if err != nil {
			s.writeTaskMutationError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"task": updated})
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodPatch)
	}
}

func (s *Server) handleTaskArchive(writer http.ResponseWriter, request *http.Request, taskID string, archive bool) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	var payload taskArchiveRequest
	if err := decodeStrictJSON(request, &payload); err != nil || payload.ExpectedRevision == nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body", "expected_revision is required")
		return
	}
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	registry, err := s.taskRegistrySnapshot()
	if err != nil {
		writeTaskHistoryUnavailable(writer, err)
		return
	}
	var updated producttasks.Task
	err = registry.Update(func(history *producttasks.History) error {
		index := indexTask(history.Tasks, taskID)
		if index < 0 {
			return errTaskNotFound
		}
		task := &history.Tasks[index]
		if task.Revision != *payload.ExpectedRevision {
			return &taskRevisionConflict{Expected: *payload.ExpectedRevision, Actual: task.Revision}
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if archive {
			task.Lifecycle = producttasks.LifecycleArchived
			task.ArchivedAt = &now
		} else {
			task.Lifecycle = producttasks.LifecycleOpen
			task.ArchivedAt = nil
		}
		task.Revision++
		task.UpdatedAt = now
		task.LastActivityAt = now
		updated = producttasks.CloneTask(*task)
		return nil
	})
	if err != nil {
		s.writeTaskMutationError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"task": updated})
}

var errTaskNotFound = errors.New("task not found")

type taskRevisionConflict struct {
	Expected int
	Actual   int
}

func (e *taskRevisionConflict) Error() string {
	return fmt.Sprintf("task revision conflict: expected %d, actual %d", e.Expected, e.Actual)
}

func (s *Server) writeTaskMutationError(writer http.ResponseWriter, err error) {
	var conflict *taskRevisionConflict
	switch {
	case errors.Is(err, errTaskNotFound):
		writeError(writer, http.StatusNotFound, "task_not_found", "task not found")
	case errors.As(err, &conflict):
		writeJSON(writer, http.StatusConflict, map[string]any{
			"error":             map[string]string{"code": "revision_conflict", "message": conflict.Error()},
			"expected_revision": conflict.Expected,
			"actual_revision":   conflict.Actual,
		})
	case strings.Contains(err.Error(), "must change"):
		writeError(writer, http.StatusBadRequest, "empty_task_patch", err.Error())
	case strings.Contains(err.Error(), "task is invalid") || strings.Contains(err.Error(), "task history is invalid"):
		writeError(writer, http.StatusBadRequest, "task_invalid", err.Error())
	default:
		writeError(writer, http.StatusInternalServerError, "task_persist_failed", err.Error())
	}
}

func writeTaskHistoryUnavailable(writer http.ResponseWriter, err error) {
	writeError(writer, http.StatusServiceUnavailable, "task_history_unavailable", err.Error())
}

func findTask(history producttasks.History, taskID string) (producttasks.Task, bool) {
	for _, task := range history.Tasks {
		if task.TaskID == taskID {
			return producttasks.CloneTask(task), true
		}
	}
	return producttasks.Task{}, false
}

func indexTask(values []producttasks.Task, taskID string) int {
	for index, task := range values {
		if task.TaskID == taskID {
			return index
		}
	}
	return -1
}

func taskHasRepository(task producttasks.Task, repository string) bool {
	for _, value := range task.Scope.Repositories {
		if value.Name == repository {
			return true
		}
	}
	return false
}

func parseTaskLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 20, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > 100 {
		return 0, errors.New("limit must be an integer between 1 and 100")
	}
	return value, nil
}

func parseTaskTimeFilter(request *http.Request) (*time.Time, *time.Time, error) {
	parse := func(key string) (*time.Time, error) {
		raw := strings.TrimSpace(request.URL.Query().Get(key))
		if raw == "" {
			return nil, nil
		}
		value, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, fmt.Errorf("%s must be RFC3339", key)
		}
		return &value, nil
	}
	from, err := parse("from")
	if err != nil {
		return nil, nil, err
	}
	to, err := parse("to")
	if err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && to.Before(*from) {
		return nil, nil, errors.New("to must not precede from")
	}
	return from, to, nil
}

func afterTaskCursor(task producttasks.Task, cursor taskCursor) bool {
	taskTime, err := time.Parse(time.RFC3339Nano, task.LastActivityAt)
	cursorTime, cursorErr := time.Parse(time.RFC3339Nano, cursor.LastActivityAt)
	if err != nil || cursorErr != nil {
		return false
	}
	if taskTime.Equal(cursorTime) {
		return task.TaskID > cursor.TaskID
	}
	return taskTime.Before(cursorTime)
}

func encodeTaskCursor(cursor taskCursor) string {
	raw, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeTaskCursor(raw string) (*taskCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, errors.New("cursor is not valid base64")
	}
	var cursor taskCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil || strings.TrimSpace(cursor.TaskID) == "" {
		return nil, errors.New("cursor payload is invalid")
	}
	if _, err := time.Parse(time.RFC3339Nano, cursor.LastActivityAt); err != nil {
		return nil, errors.New("cursor timestamp is invalid")
	}
	return &cursor, nil
}

func newOpaqueID(prefix string) string {
	bytesValue := make([]byte, 10)
	if _, err := rand.Read(bytesValue); err == nil {
		return prefix + "_" + hex.EncodeToString(bytesValue)
	}
	return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
}
