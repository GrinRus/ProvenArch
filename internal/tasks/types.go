// Package tasks contains the authoritative product Task/Attempt contracts.
//
// The package deliberately does not import the orchestrator runtime types. A
// product Task is durable user intent, while a runtime Task is an execution
// envelope. Keeping these contracts separate prevents UI and persistence code
// from deriving product identity from a run name or from provider output.
package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/validation"
)

const CurrentVersion = 1

type Lifecycle string

const (
	LifecycleOpen     Lifecycle = "open"
	LifecycleArchived Lifecycle = "archived"
)

type AttemptStatus string

const (
	AttemptQueued    AttemptStatus = "queued"
	AttemptRunning   AttemptStatus = "running"
	AttemptSucceeded AttemptStatus = "succeeded"
	AttemptFailed    AttemptStatus = "failed"
	AttemptCanceled  AttemptStatus = "canceled"
	AttemptTimeout   AttemptStatus = "timeout"
)

type Availability string

const (
	Available   Availability = "available"
	Unavailable Availability = "unavailable"
)

type PublicationState string

const (
	PublicationLinked      PublicationState = "linked"
	PublicationUnavailable PublicationState = "unavailable"
)

type RetainedEvidence string

const (
	EvidenceFull        RetainedEvidence = "full"
	EvidenceSummaryOnly RetainedEvidence = "summary_only"
	EvidenceUnavailable RetainedEvidence = "unavailable"
)

type Scope struct {
	Repositories []RepositoryScope `json:"repositories"`
}

type RepositoryScope struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

type RunnerPreset struct {
	Preset      string `json:"preset"`
	Mode        string `json:"mode,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Model       string `json:"model,omitempty"`
	Effort      string `json:"effort,omitempty"`
	Permissions string `json:"permissions,omitempty"`
}

type Task struct {
	Version        int              `json:"version"`
	TaskID         string           `json:"task_id"`
	Title          string           `json:"title"`
	Goal           string           `json:"goal"`
	Context        string           `json:"context,omitempty"`
	Scope          Scope            `json:"scope"`
	DesiredRunner  RunnerPreset     `json:"desired_runner"`
	Lifecycle      Lifecycle        `json:"lifecycle"`
	Revision       int              `json:"revision"`
	CreatedAt      string           `json:"created_at"`
	UpdatedAt      string           `json:"updated_at"`
	LastActivityAt string           `json:"last_activity_at"`
	ArchivedAt     *string          `json:"archived_at"`
	Attempts       []AttemptSummary `json:"attempts"`
	Outcome        Outcome          `json:"outcome"`
	Publication    Publication      `json:"publication"`
}

type AttemptSummary struct {
	AttemptID        string           `json:"attempt_id"`
	RunID            string           `json:"run_id"`
	ParentAttemptID  *string          `json:"parent_attempt_id"`
	TaskRevision     int              `json:"task_revision"`
	Status           AttemptStatus    `json:"status"`
	AdmittedAt       string           `json:"admitted_at"`
	UpdatedAt        string           `json:"updated_at"`
	FinishedAt       *string          `json:"finished_at"`
	Summary          string           `json:"summary,omitempty"`
	RetainedEvidence RetainedEvidence `json:"retained_evidence,omitempty"`
}

type IntentSnapshot struct {
	Title         string       `json:"title"`
	Goal          string       `json:"goal"`
	Context       string       `json:"context,omitempty"`
	Scope         Scope        `json:"scope"`
	DesiredRunner RunnerPreset `json:"desired_runner"`
}

type ExecutionSettings struct {
	Strategy      string `json:"strategy,omitempty"`
	MaxParallel   int    `json:"max_parallel,omitempty"`
	FailurePolicy string `json:"failure_policy,omitempty"`
	ShardMode     string `json:"shard_mode,omitempty"`
}

type EffectiveRuntime struct {
	Mode              string                  `json:"mode"`
	Provider          string                  `json:"provider"`
	Model             string                  `json:"model,omitempty"`
	Effort            string                  `json:"effort,omitempty"`
	Permissions       string                  `json:"permissions"`
	Timeouts          map[string]int          `json:"timeouts,omitempty"`
	Scope             Scope                   `json:"scope"`
	Execution         ExecutionSettings       `json:"execution"`
	StepOverrides     map[string]RunnerPreset `json:"step_overrides,omitempty"`
	ResolutionSources map[string]string       `json:"resolution_sources"`
}

type TerminalSummary struct {
	Status           AttemptStatus    `json:"status"`
	ErrorCode        string           `json:"error_code,omitempty"`
	Error            string           `json:"error,omitempty"`
	Summary          string           `json:"summary,omitempty"`
	RetainedEvidence RetainedEvidence `json:"retained_evidence"`
}

type Attempt struct {
	Version            int              `json:"version"`
	AttemptID          string           `json:"attempt_id"`
	TaskID             string           `json:"task_id"`
	RunID              string           `json:"run_id"`
	ParentAttemptID    *string          `json:"parent_attempt_id"`
	RetryReason        string           `json:"retry_reason,omitempty"`
	Pipeline           string           `json:"pipeline,omitempty"`
	IdempotencyKey     string           `json:"idempotency_key,omitempty"`
	RequestFingerprint string           `json:"request_fingerprint,omitempty"`
	TaskRevision       int              `json:"task_revision"`
	IntentSnapshot     IntentSnapshot   `json:"intent_snapshot"`
	EffectiveRuntime   EffectiveRuntime `json:"effective_runtime"`
	Status             AttemptStatus    `json:"status"`
	AdmittedAt         string           `json:"admitted_at"`
	QueuedAt           *string          `json:"queued_at"`
	StartedAt          *string          `json:"started_at"`
	FinishedAt         *string          `json:"finished_at"`
	TerminalSummary    *TerminalSummary `json:"terminal_summary"`
	RetainedEvidence   RetainedEvidence `json:"retained_evidence"`
	Outcome            Outcome          `json:"outcome"`
	Publication        Publication      `json:"publication"`
}

type Outcome struct {
	State             Availability `json:"state"`
	AttemptID         string       `json:"attempt_id,omitempty"`
	RunID             string       `json:"run_id,omitempty"`
	SnapshotPath      string       `json:"snapshot_path,omitempty"`
	UnavailableReason string       `json:"unavailable_reason,omitempty"`
}

type Publication struct {
	State                PublicationState `json:"state"`
	AttemptID            string           `json:"attempt_id,omitempty"`
	RunID                string           `json:"run_id,omitempty"`
	Action               string           `json:"action,omitempty"`
	Branch               string           `json:"branch,omitempty"`
	Commit               string           `json:"commit,omitempty"`
	InventoryFingerprint string           `json:"inventory_fingerprint,omitempty"`
	UnavailableReason    string           `json:"unavailable_reason,omitempty"`
}

type History struct {
	Version     int       `json:"version"`
	GeneratedAt string    `json:"generated_at"`
	Tasks       []Task    `json:"tasks"`
	Attempts    []Attempt `json:"attempts"`
	Diagnostics []string  `json:"diagnostics"`
}

var opaqueIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`)

func IsOpaqueID(value string) bool { return opaqueIDPattern.MatchString(strings.TrimSpace(value)) }

func ParseTask(raw []byte) (Task, error) {
	if err := validation.ValidateRawJSON(validation.TaskSchema, raw); err != nil {
		return Task{}, fmt.Errorf("task is invalid: %w", err)
	}
	var value Task
	if err := json.Unmarshal(raw, &value); err != nil {
		return Task{}, fmt.Errorf("decode task: %w", err)
	}
	if err := value.Validate(); err != nil {
		return Task{}, err
	}
	return value, nil
}

func ParseAttempt(raw []byte) (Attempt, error) {
	if err := validation.ValidateRawJSON(validation.AttemptSchema, raw); err != nil {
		return Attempt{}, fmt.Errorf("attempt is invalid: %w", err)
	}
	var value Attempt
	if err := json.Unmarshal(raw, &value); err != nil {
		return Attempt{}, fmt.Errorf("decode attempt: %w", err)
	}
	if err := value.Validate(); err != nil {
		return Attempt{}, err
	}
	return value, nil
}

func ParseHistory(raw []byte) (History, error) {
	if err := validation.ValidateRawJSON(validation.TaskHistorySchema, raw); err != nil {
		return History{}, fmt.Errorf("task history is invalid: %w", err)
	}
	var value History
	if err := json.Unmarshal(raw, &value); err != nil {
		return History{}, fmt.Errorf("decode task history: %w", err)
	}
	if err := value.Validate(); err != nil {
		return History{}, err
	}
	return value, nil
}

func (value Task) Validate() error {
	problems := []string{}
	if value.Version != CurrentVersion {
		problems = append(problems, "version must be 1")
	}
	if !IsOpaqueID(value.TaskID) {
		problems = append(problems, "task_id must be an opaque server-generated id")
	}
	if strings.TrimSpace(value.Title) == "" {
		problems = append(problems, "title is required")
	}
	if strings.TrimSpace(value.Goal) == "" {
		problems = append(problems, "goal is required")
	}
	problems = append(problems, validateScope(value.Scope, "scope")...)
	problems = append(problems, validateRunner(value.DesiredRunner, "desired_runner")...)
	if value.Lifecycle != LifecycleOpen && value.Lifecycle != LifecycleArchived {
		problems = append(problems, "lifecycle must be open or archived")
	}
	if value.Revision < 1 {
		problems = append(problems, "revision must be positive")
	}
	problems = append(problems, validateTaskTimes(value)...)
	if value.Lifecycle == LifecycleArchived && value.ArchivedAt == nil {
		problems = append(problems, "archived_at is required for archived tasks")
	}
	if value.Lifecycle == LifecycleOpen && value.ArchivedAt != nil {
		problems = append(problems, "archived_at must be absent for open tasks")
	}
	problems = append(problems, validateSummaries(value.Attempts)...)
	problems = append(problems, validateOutcome(value.Outcome, "outcome", true)...)
	problems = append(problems, validatePublication(value.Publication, "publication", true)...)
	return joinProblems("task", problems)
}

func (value Attempt) Validate() error {
	problems := []string{}
	if value.Version != CurrentVersion {
		problems = append(problems, "version must be 1")
	}
	if !IsOpaqueID(value.AttemptID) {
		problems = append(problems, "attempt_id must be an opaque server-generated id")
	}
	if !IsOpaqueID(value.TaskID) {
		problems = append(problems, "task_id must be an opaque server-generated id")
	}
	if strings.TrimSpace(value.RunID) == "" {
		problems = append(problems, "run_id is required")
	}
	if value.ParentAttemptID != nil && strings.TrimSpace(*value.ParentAttemptID) == value.AttemptID {
		problems = append(problems, "parent_attempt_id cannot equal attempt_id")
	}
	if value.ParentAttemptID != nil && strings.TrimSpace(value.RetryReason) == "" {
		problems = append(problems, "retry_reason is required for child attempts")
	}
	if value.ParentAttemptID == nil && strings.TrimSpace(value.RetryReason) != "" {
		problems = append(problems, "retry_reason requires parent_attempt_id")
	}
	if value.Pipeline != "" && value.Pipeline != "init" && value.Pipeline != "refresh" {
		problems = append(problems, "pipeline must be init or refresh")
	}
	if len(value.IdempotencyKey) > 256 {
		problems = append(problems, "idempotency_key must be at most 256 characters")
	}
	if value.RequestFingerprint != "" && !regexp.MustCompile(`^[a-fA-F0-9]{64}$`).MatchString(value.RequestFingerprint) {
		problems = append(problems, "request_fingerprint must be a SHA-256 fingerprint")
	}
	if value.TaskRevision < 1 {
		problems = append(problems, "task_revision must be positive")
	}
	problems = append(problems, validateIntent(value.IntentSnapshot)...)
	problems = append(problems, validateEffectiveRuntime(value.EffectiveRuntime)...)
	if !validAttemptStatus(value.Status) {
		problems = append(problems, "status is invalid")
	}
	problems = append(problems, validateAttemptTimes(value)...)
	terminal := isTerminal(value.Status)
	if terminal && value.FinishedAt == nil {
		problems = append(problems, "finished_at is required for terminal attempts")
	}
	if terminal && value.TerminalSummary == nil {
		problems = append(problems, "terminal_summary is required for terminal attempts")
	}
	if !terminal && value.FinishedAt != nil {
		problems = append(problems, "finished_at is only allowed for terminal attempts")
	}
	if value.TerminalSummary != nil {
		if !isTerminal(value.TerminalSummary.Status) {
			problems = append(problems, "terminal_summary.status must be terminal")
		}
		if value.TerminalSummary.Status != value.Status {
			problems = append(problems, "terminal_summary.status must match status")
		}
		if value.TerminalSummary.RetainedEvidence != value.RetainedEvidence {
			problems = append(problems, "terminal_summary.retained_evidence must match retained_evidence")
		}
	}
	problems = append(problems, validateOutcome(value.Outcome, "outcome", false)...)
	problems = append(problems, validatePublication(value.Publication, "publication", false)...)
	return joinProblems("attempt", problems)
}

func (value History) Validate() error {
	problems := []string{}
	if value.Version != CurrentVersion {
		problems = append(problems, "version must be 1")
	}
	if _, err := parseTimestamp(value.GeneratedAt); err != nil {
		problems = append(problems, "generated_at must be an RFC3339 timestamp")
	}
	if len(value.Diagnostics) > 20 {
		problems = append(problems, "diagnostics cannot contain more than 20 entries")
	}
	tasksByID := map[string]Task{}
	for index, task := range value.Tasks {
		if err := task.Validate(); err != nil {
			problems = append(problems, fmt.Sprintf("tasks[%d]: %v", index, err))
		}
		if _, exists := tasksByID[task.TaskID]; exists && task.TaskID != "" {
			problems = append(problems, fmt.Sprintf("tasks[%d].task_id must be unique", index))
		}
		tasksByID[task.TaskID] = task
	}
	attemptsByID := map[string]Attempt{}
	for index, attempt := range value.Attempts {
		if err := attempt.Validate(); err != nil {
			problems = append(problems, fmt.Sprintf("attempts[%d]: %v", index, err))
		}
		if _, exists := attemptsByID[attempt.AttemptID]; exists && attempt.AttemptID != "" {
			problems = append(problems, fmt.Sprintf("attempts[%d].attempt_id must be unique", index))
		}
		attemptsByID[attempt.AttemptID] = attempt
		if task, exists := tasksByID[attempt.TaskID]; !exists {
			problems = append(problems, fmt.Sprintf("attempts[%d] references unknown task_id %q", index, attempt.TaskID))
		} else if !summaryForAttempt(task.Attempts, attempt.AttemptID, attempt.RunID, attempt.TaskRevision, attempt.Status) {
			problems = append(problems, fmt.Sprintf("attempts[%d] is not represented by an equivalent task attempt summary", index))
		}
	}
	for index, attempt := range value.Attempts {
		if attempt.ParentAttemptID == nil {
			continue
		}
		parent, exists := attemptsByID[*attempt.ParentAttemptID]
		if !exists {
			problems = append(problems, fmt.Sprintf("attempts[%d].parent_attempt_id references missing attempt %q", index, *attempt.ParentAttemptID))
			continue
		}
		if parent.TaskID != attempt.TaskID {
			problems = append(problems, fmt.Sprintf("attempts[%d].parent_attempt_id crosses task boundary", index))
		}
	}
	for _, task := range value.Tasks {
		for _, summary := range task.Attempts {
			attempt, exists := attemptsByID[summary.AttemptID]
			if !exists {
				problems = append(problems, fmt.Sprintf("task %q references missing attempt %q", task.TaskID, summary.AttemptID))
				continue
			}
			if attempt.TaskID != task.TaskID || attempt.RunID != summary.RunID || attempt.TaskRevision != summary.TaskRevision || attempt.Status != summary.Status {
				problems = append(problems, fmt.Sprintf("task %q has inconsistent summary for attempt %q", task.TaskID, summary.AttemptID))
			}
		}
	}
	return joinProblems("task history", problems)
}

func MarshalTask(value Task) ([]byte, error) { return marshalValidated(value.Validate, value) }

func MarshalAttempt(value Attempt) ([]byte, error) { return marshalValidated(value.Validate, value) }

func MarshalHistory(value History) ([]byte, error) { return marshalValidated(value.Validate, value) }

func CloneAttempt(value Attempt) Attempt {
	clone := value
	clone.ParentAttemptID = cloneStringPointer(value.ParentAttemptID)
	clone.QueuedAt = cloneStringPointer(value.QueuedAt)
	clone.StartedAt = cloneStringPointer(value.StartedAt)
	clone.FinishedAt = cloneStringPointer(value.FinishedAt)
	clone.EffectiveRuntime.Timeouts = cloneIntMap(value.EffectiveRuntime.Timeouts)
	clone.EffectiveRuntime.ResolutionSources = cloneStringMap(value.EffectiveRuntime.ResolutionSources)
	clone.EffectiveRuntime.StepOverrides = cloneRunnerMap(value.EffectiveRuntime.StepOverrides)
	clone.EffectiveRuntime.Scope.Repositories = cloneRepositories(value.EffectiveRuntime.Scope.Repositories)
	clone.IntentSnapshot.Scope.Repositories = cloneRepositories(value.IntentSnapshot.Scope.Repositories)
	clone.IntentSnapshot.DesiredRunner = value.IntentSnapshot.DesiredRunner
	if value.TerminalSummary != nil {
		summary := *value.TerminalSummary
		clone.TerminalSummary = &summary
	}
	return clone
}

func marshalValidated(validate func() error, value any) ([]byte, error) {
	if err := validate(); err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal task contract: %w", err)
	}
	return append(raw, '\n'), nil
}

func validateScope(scope Scope, label string) []string {
	problems := []string{}
	if len(scope.Repositories) == 0 {
		return []string{label + ".repositories must not be empty"}
	}
	seen := map[string]struct{}{}
	for index, repository := range scope.Repositories {
		name := strings.TrimSpace(repository.Name)
		if name == "" {
			problems = append(problems, fmt.Sprintf("%s.repositories[%d].name is required", label, index))
		}
		if _, exists := seen[name]; exists && name != "" {
			problems = append(problems, fmt.Sprintf("%s.repositories[%d].name must be unique", label, index))
		}
		seen[name] = struct{}{}
		for pathIndex, path := range repository.Paths {
			clean := strings.TrimSpace(path)
			if clean == "" {
				problems = append(problems, fmt.Sprintf("%s.repositories[%d].paths[%d] is required", label, index, pathIndex))
				continue
			}
			if strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
				problems = append(problems, fmt.Sprintf("%s.repositories[%d].paths[%d] must be workspace-relative", label, index, pathIndex))
			}
		}
	}
	return problems
}

func validateRunner(runner RunnerPreset, label string) []string {
	problems := []string{}
	if strings.TrimSpace(runner.Preset) == "" {
		problems = append(problems, label+".preset is required")
	}
	if runner.Mode != "" && runner.Mode != "fake" && runner.Mode != "headless" {
		problems = append(problems, label+".mode is invalid")
	}
	if runner.Provider != "" && runner.Provider != "claude-code" && runner.Provider != "qwen-code" && runner.Provider != "codex-code" {
		problems = append(problems, label+".provider is invalid")
	}
	if runner.Permissions != "" && runner.Permissions != "trusted_full_access" && runner.Permissions != "managed" {
		problems = append(problems, label+".permissions is invalid")
	}
	return problems
}

func validateIntent(value IntentSnapshot) []string {
	problems := []string{}
	if strings.TrimSpace(value.Title) == "" {
		problems = append(problems, "intent_snapshot.title is required")
	}
	if strings.TrimSpace(value.Goal) == "" {
		problems = append(problems, "intent_snapshot.goal is required")
	}
	problems = append(problems, validateScope(value.Scope, "intent_snapshot.scope")...)
	problems = append(problems, validateRunner(value.DesiredRunner, "intent_snapshot.desired_runner")...)
	return problems
}

func validateEffectiveRuntime(value EffectiveRuntime) []string {
	problems := []string{}
	if value.Mode != "fake" && value.Mode != "headless" {
		problems = append(problems, "effective_runtime.mode is invalid")
	}
	if value.Provider != "claude-code" && value.Provider != "qwen-code" && value.Provider != "codex-code" {
		problems = append(problems, "effective_runtime.provider is invalid")
	}
	if value.Permissions != "trusted_full_access" && value.Permissions != "managed" {
		problems = append(problems, "effective_runtime.permissions is invalid")
	}
	problems = append(problems, validateScope(value.Scope, "effective_runtime.scope")...)
	if len(value.ResolutionSources) == 0 {
		problems = append(problems, "effective_runtime.resolution_sources must not be empty")
	}
	for key, source := range value.ResolutionSources {
		if strings.TrimSpace(key) == "" {
			problems = append(problems, "effective_runtime.resolution_sources keys must not be empty")
		}
		if source != "env" && source != "workspace" && source != "provider_default" && source != "task_preset" && source != "request" {
			problems = append(problems, fmt.Sprintf("effective_runtime.resolution_sources[%q] is invalid", key))
		}
	}
	for key, timeout := range value.Timeouts {
		if strings.TrimSpace(key) == "" || timeout < 1 {
			problems = append(problems, "effective_runtime.timeouts must contain positive values with non-empty keys")
		}
	}
	for key, runner := range value.StepOverrides {
		problems = append(problems, validateRunner(runner, "effective_runtime.step_overrides."+key)...)
	}
	return problems
}

func validateTaskTimes(value Task) []string {
	problems := []string{}
	created, createdErr := parseTimestamp(value.CreatedAt)
	updated, updatedErr := parseTimestamp(value.UpdatedAt)
	activity, activityErr := parseTimestamp(value.LastActivityAt)
	if createdErr != nil {
		problems = append(problems, "created_at must be an RFC3339 timestamp")
	}
	if updatedErr != nil {
		problems = append(problems, "updated_at must be an RFC3339 timestamp")
	}
	if activityErr != nil {
		problems = append(problems, "last_activity_at must be an RFC3339 timestamp")
	}
	if createdErr == nil && updatedErr == nil && updated.Before(created) {
		problems = append(problems, "updated_at must not precede created_at")
	}
	if updatedErr == nil && activityErr == nil && activity.Before(updated) {
		problems = append(problems, "last_activity_at must not precede updated_at")
	}
	if value.ArchivedAt != nil {
		archived, err := parseTimestamp(*value.ArchivedAt)
		if err != nil {
			problems = append(problems, "archived_at must be an RFC3339 timestamp")
		} else if activityErr == nil && archived.Before(activity) {
			problems = append(problems, "archived_at must not precede last_activity_at")
		}
	}
	return problems
}

func validateAttemptTimes(value Attempt) []string {
	problems := []string{}
	admitted, admittedErr := parseTimestamp(value.AdmittedAt)
	if admittedErr != nil {
		problems = append(problems, "admitted_at must be an RFC3339 timestamp")
	}
	previous := admitted
	previousLabel := "admitted_at"
	for _, entry := range []struct {
		label string
		value *string
	}{
		{label: "queued_at", value: value.QueuedAt},
		{label: "started_at", value: value.StartedAt},
		{label: "finished_at", value: value.FinishedAt},
	} {
		label, pointer := entry.label, entry.value
		if pointer == nil {
			continue
		}
		parsed, err := parseTimestamp(*pointer)
		if err != nil {
			problems = append(problems, label+" must be an RFC3339 timestamp")
			continue
		}
		if admittedErr == nil && parsed.Before(previous) {
			problems = append(problems, label+" must not precede "+previousLabel)
		}
		previous, previousLabel = parsed, label
	}
	return problems
}

func validateSummaries(values []AttemptSummary) []string {
	problems := []string{}
	seen := map[string]struct{}{}
	for index, value := range values {
		if !IsOpaqueID(value.AttemptID) {
			problems = append(problems, fmt.Sprintf("attempts[%d].attempt_id is invalid", index))
		}
		if strings.TrimSpace(value.RunID) == "" {
			problems = append(problems, fmt.Sprintf("attempts[%d].run_id is required", index))
		}
		if _, exists := seen[value.AttemptID]; exists && value.AttemptID != "" {
			problems = append(problems, fmt.Sprintf("attempts[%d].attempt_id must be unique", index))
		}
		seen[value.AttemptID] = struct{}{}
		if value.TaskRevision < 1 {
			problems = append(problems, fmt.Sprintf("attempts[%d].task_revision must be positive", index))
		}
		if !validAttemptStatus(value.Status) {
			problems = append(problems, fmt.Sprintf("attempts[%d].status is invalid", index))
		}
		if _, err := parseTimestamp(value.AdmittedAt); err != nil {
			problems = append(problems, fmt.Sprintf("attempts[%d].admitted_at is invalid", index))
		}
		if _, err := parseTimestamp(value.UpdatedAt); err != nil {
			problems = append(problems, fmt.Sprintf("attempts[%d].updated_at is invalid", index))
		}
	}
	return problems
}

func validateOutcome(value Outcome, label string, includeAttempt bool) []string {
	problems := []string{}
	if value.State != Available && value.State != Unavailable {
		return []string{label + ".state is invalid"}
	}
	if value.State == Available {
		if includeAttempt && !IsOpaqueID(value.AttemptID) {
			problems = append(problems, label+".attempt_id is required when available")
		}
		if strings.TrimSpace(value.RunID) == "" {
			problems = append(problems, label+".run_id is required when available")
		}
		if strings.TrimSpace(value.SnapshotPath) == "" {
			problems = append(problems, label+".snapshot_path is required when available")
		}
		if strings.TrimSpace(value.UnavailableReason) != "" {
			problems = append(problems, label+".unavailable_reason is not allowed when available")
		}
	} else if strings.TrimSpace(value.UnavailableReason) == "" {
		problems = append(problems, label+".unavailable_reason is required when unavailable")
	}
	return problems
}

func validatePublication(value Publication, label string, includeAttempt bool) []string {
	problems := []string{}
	if value.State != PublicationLinked && value.State != PublicationUnavailable {
		return []string{label + ".state is invalid"}
	}
	if value.State == PublicationLinked {
		if includeAttempt && !IsOpaqueID(value.AttemptID) {
			problems = append(problems, label+".attempt_id is required when linked")
		}
		if strings.TrimSpace(value.RunID) == "" {
			problems = append(problems, label+".run_id is required when linked")
		}
		if value.Action != "commit" && value.Action != "branch" && value.Action != "pull_request" {
			problems = append(problems, label+".action is required when linked")
		}
		if strings.TrimSpace(value.Branch) == "" || strings.TrimSpace(value.Commit) == "" {
			problems = append(problems, label+".branch and commit are required when linked")
		}
		if !regexp.MustCompile(`^[a-fA-F0-9]{64}$`).MatchString(value.InventoryFingerprint) {
			problems = append(problems, label+".inventory_fingerprint must be a SHA-256 fingerprint when linked")
		}
	} else if strings.TrimSpace(value.UnavailableReason) == "" {
		problems = append(problems, label+".unavailable_reason is required when unavailable")
	}
	return problems
}

func summaryForAttempt(values []AttemptSummary, attemptID, runID string, revision int, status AttemptStatus) bool {
	for _, value := range values {
		if value.AttemptID == attemptID && value.RunID == runID && value.TaskRevision == revision && value.Status == status {
			return true
		}
	}
	return false
}

func parseTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
}

func validAttemptStatus(value AttemptStatus) bool {
	return value == AttemptQueued || value == AttemptRunning || isTerminal(value)
}

func isTerminal(value AttemptStatus) bool {
	return value == AttemptSucceeded || value == AttemptFailed || value == AttemptCanceled || value == AttemptTimeout
}

func joinProblems(kind string, problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("%s is invalid: %s", kind, strings.Join(problems, "; "))
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
func cloneStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	clone := make(map[string]string, len(value))
	for key, item := range value {
		clone[key] = item
	}
	return clone
}
func cloneIntMap(value map[string]int) map[string]int {
	if value == nil {
		return nil
	}
	clone := make(map[string]int, len(value))
	for key, item := range value {
		clone[key] = item
	}
	return clone
}
func cloneRunnerMap(value map[string]RunnerPreset) map[string]RunnerPreset {
	if value == nil {
		return nil
	}
	clone := make(map[string]RunnerPreset, len(value))
	for key, item := range value {
		clone[key] = item
	}
	return clone
}
func cloneRepositories(value []RepositoryScope) []RepositoryScope {
	if value == nil {
		return nil
	}
	clone := make([]RepositoryScope, len(value))
	for index, item := range value {
		clone[index] = item
		clone[index].Paths = append([]string(nil), item.Paths...)
	}
	return clone
}

// EqualJSON is useful for immutable snapshot tests and compares canonical JSON
// values rather than pointer identity.
func EqualJSON(left, right any) bool {
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftRaw, rightRaw)
}
