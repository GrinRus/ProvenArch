package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	HistoryPath           = "reports/taskruns/task-history.json"
	HistoryDiagnosticsMax = 20
)

type registryWriteFile func(workspace.Root, string, []byte) error

// Registry is the durable Task/Attempt aggregate store. It publishes a new
// snapshot only after the primary current file has been atomically replaced;
// the last-good copy is a recovery aid and never becomes the in-memory source
// of truth while a current write is failing.
type Registry struct {
	mu          sync.RWMutex
	root        workspace.Root
	clock       func() time.Time
	writeFile   registryWriteFile
	snapshot    History
	diagnostics []string
}

func NewRegistry(root workspace.Root, clock func() time.Time) (*Registry, error) {
	if clock == nil {
		clock = time.Now
	}
	registry := &Registry{
		root:  root,
		clock: clock,
		writeFile: func(root workspace.Root, path string, content []byte) error {
			return root.WriteFileAtomic(path, content)
		},
	}
	if err := registry.load(); err != nil {
		return nil, err
	}
	return registry, nil
}

func NewEmptyHistory(now time.Time) History {
	return History{
		Version:     CurrentVersion,
		GeneratedAt: now.UTC().Format(time.RFC3339Nano),
		Tasks:       []Task{},
		Attempts:    []Attempt{},
		Diagnostics: []string{},
	}
}

func (r *Registry) Snapshot() History {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return CloneHistory(r.snapshot)
}

func (r *Registry) Diagnostics() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	diagnostics := make([]string, len(r.diagnostics))
	copy(diagnostics, r.diagnostics)
	return diagnostics
}

// Replace validates and durably publishes the complete registry candidate.
// The caller supplies a full candidate so that a failed write cannot expose a
// partially mutated Task/Attempt graph.
func (r *Registry) Replace(candidate History) error {
	candidate = CloneHistory(candidate)
	candidate.GeneratedAt = r.clock().UTC().Format(time.RFC3339Nano)
	if err := candidate.Validate(); err != nil {
		return err
	}
	raw, err := MarshalHistory(candidate)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.replaceLocked(candidate, raw)
}

func (r *Registry) replaceLocked(candidate History, raw []byte) error {
	if err := r.writeFile(r.root, HistoryPath, raw); err != nil {
		r.addDiagnosticLocked(fmt.Sprintf("persist task history failed: %v", err))
		return fmt.Errorf("persist task history: %w", err)
	}
	if err := r.writeFile(r.root, HistoryPath+".last-good", raw); err != nil {
		// The primary state is durable. Keep it authoritative while making the
		// backup fault explicit for the diagnostics surface.
		r.addDiagnosticLocked(fmt.Sprintf("persist task history last-good failed: %v", err))
	}
	r.snapshot = CloneHistory(candidate)
	return nil
}

func (r *Registry) Update(mutator func(*History) error) error {
	if mutator == nil {
		return errors.New("task history mutator is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	candidate := CloneHistory(r.snapshot)
	if err := mutator(&candidate); err != nil {
		return err
	}
	candidate.GeneratedAt = r.clock().UTC().Format(time.RFC3339Nano)
	if err := candidate.Validate(); err != nil {
		return err
	}
	raw, err := MarshalHistory(candidate)
	if err != nil {
		return err
	}
	return r.replaceLocked(candidate, raw)
}

func (r *Registry) load() error {
	current, currentErr := r.root.ReadFile(HistoryPath)
	if currentErr == nil {
		if parsed, migrated, parseErr := parseHistoryWithNormalization(current); parseErr == nil {
			r.snapshot = parsed
			if migrated {
				r.persistNormalizedHistory(parsed)
			}
			return nil
		} else {
			currentErr = parseErr
		}
	}

	lastGood, lastGoodErr := r.root.ReadLastGoodFile(HistoryPath)
	if lastGoodErr == nil {
		parsed, migrated, parseErr := parseHistoryWithNormalization(lastGood)
		if parseErr == nil {
			r.snapshot = parsed
			if migrated {
				r.persistNormalizedHistory(parsed)
			}
			r.addDiagnosticLocked(fmt.Sprintf("recovered task history from %s", HistoryPath+".last-good"))
			if currentErr != nil && !errors.Is(currentErr, os.ErrNotExist) {
				r.addDiagnosticLocked(fmt.Sprintf("current task history unavailable: %v", currentErr))
			}
			return nil
		}
		lastGoodErr = parseErr
	}

	if errors.Is(currentErr, os.ErrNotExist) && errors.Is(lastGoodErr, os.ErrNotExist) {
		r.snapshot = NewEmptyHistory(r.clock())
		return nil
	}
	if currentErr == nil {
		currentErr = errors.New("current task history is unavailable")
	}
	if lastGoodErr == nil {
		lastGoodErr = errors.New("last-good task history is unavailable")
	}
	return fmt.Errorf("load task history failed: current=%v last_good=%v", currentErr, lastGoodErr)
}

func parseHistoryWithNormalization(raw []byte) (History, bool, error) {
	parsed, err := ParseHistory(raw)
	if err == nil {
		return parsed, false, nil
	}
	normalized, changed, normalizeErr := normalizeLegacyRepositoryPaths(raw)
	if normalizeErr != nil || !changed {
		return History{}, false, err
	}
	parsed, normalizedErr := ParseHistory(normalized)
	if normalizedErr != nil {
		return History{}, false, err
	}
	return parsed, true, nil
}

func normalizeLegacyRepositoryPaths(raw []byte) ([]byte, bool, error) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, false, err
	}
	changed := false
	normalizeScope := func(value any) {
		scope, ok := value.(map[string]any)
		if !ok {
			return
		}
		repositories, ok := scope["repositories"].([]any)
		if !ok {
			return
		}
		for _, item := range repositories {
			repository, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if paths, present := repository["paths"]; present && paths == nil {
				repository["paths"] = []any{}
				changed = true
			}
		}
	}
	if tasks, ok := document["tasks"].([]any); ok {
		for _, item := range tasks {
			if task, ok := item.(map[string]any); ok {
				normalizeScope(task["scope"])
			}
		}
	}
	if attempts, ok := document["attempts"].([]any); ok {
		for _, item := range attempts {
			if attempt, ok := item.(map[string]any); ok {
				if intent, ok := attempt["intent_snapshot"].(map[string]any); ok {
					normalizeScope(intent["scope"])
				}
				if runtime, ok := attempt["effective_runtime"].(map[string]any); ok {
					normalizeScope(runtime["scope"])
				}
			}
		}
	}
	if !changed {
		return raw, false, nil
	}
	normalized, err := json.Marshal(document)
	return normalized, true, err
}

func (r *Registry) persistNormalizedHistory(history History) {
	raw, err := MarshalHistory(history)
	if err != nil {
		r.addDiagnosticLocked(fmt.Sprintf("normalize task history failed: %v", err))
		return
	}
	if err := r.writeFile(r.root, HistoryPath, raw); err != nil {
		r.addDiagnosticLocked(fmt.Sprintf("persist normalized task history failed: %v", err))
		return
	}
	if err := r.writeFile(r.root, HistoryPath+".last-good", raw); err != nil {
		r.addDiagnosticLocked(fmt.Sprintf("persist normalized task history last-good failed: %v", err))
	}
	r.addDiagnosticLocked("normalized legacy null repository paths in task history")
}

func (r *Registry) addDiagnosticLocked(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	for _, existing := range r.diagnostics {
		if existing == message {
			return
		}
	}
	r.diagnostics = append(r.diagnostics, message)
	if len(r.diagnostics) > HistoryDiagnosticsMax {
		r.diagnostics = append([]string(nil), r.diagnostics[len(r.diagnostics)-HistoryDiagnosticsMax:]...)
	}
}

func CloneHistory(value History) History {
	clone := value
	clone.Tasks = make([]Task, len(value.Tasks))
	for index, task := range value.Tasks {
		clone.Tasks[index] = CloneTask(task)
	}
	clone.Attempts = make([]Attempt, len(value.Attempts))
	for index, attempt := range value.Attempts {
		clone.Attempts[index] = CloneAttempt(attempt)
	}
	// Keep the JSON array shape stable even when the source is an empty,
	// non-nil slice. append onto a nil slice turns that valid empty array into
	// nil, which json.Marshal encodes as `null` and makes a persisted history
	// fail the task-history schema on restart.
	clone.Diagnostics = make([]string, len(value.Diagnostics))
	copy(clone.Diagnostics, value.Diagnostics)
	return clone
}

func CloneTask(value Task) Task {
	clone := value
	clone.ArchivedAt = cloneStringPointer(value.ArchivedAt)
	clone.Scope.Repositories = cloneRepositories(value.Scope.Repositories)
	clone.Attempts = make([]AttemptSummary, len(value.Attempts))
	for index, summary := range value.Attempts {
		clone.Attempts[index] = summary
		clone.Attempts[index].ParentAttemptID = cloneStringPointer(summary.ParentAttemptID)
		clone.Attempts[index].FinishedAt = cloneStringPointer(summary.FinishedAt)
	}
	return clone
}
