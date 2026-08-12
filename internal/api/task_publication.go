package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
)

// taskPublicationContext is intentionally all-or-nothing. A partial context
// is never completed from the latest run or any other heuristic source.
type taskPublicationContext struct {
	TaskID    string
	AttemptID string
	RunID     string
}

func (value taskPublicationContext) provided() bool {
	return strings.TrimSpace(value.TaskID) != "" || strings.TrimSpace(value.AttemptID) != "" || strings.TrimSpace(value.RunID) != ""
}

func (value taskPublicationContext) validateShape() error {
	if !value.provided() {
		return nil
	}
	if !producttasks.IsOpaqueID(value.TaskID) || !producttasks.IsOpaqueID(value.AttemptID) || strings.TrimSpace(value.RunID) == "" {
		return errors.New("task_id, attempt_id and run_id are required and must identify an exact Task/Attempt/run join")
	}
	return nil
}

func validateTaskPublicationContext(history producttasks.History, value taskPublicationContext) (producttasks.Task, producttasks.Attempt, error) {
	if err := value.validateShape(); err != nil {
		return producttasks.Task{}, producttasks.Attempt{}, err
	}
	if !value.provided() {
		return producttasks.Task{}, producttasks.Attempt{}, nil
	}
	task, found := findTask(history, value.TaskID)
	if !found {
		return producttasks.Task{}, producttasks.Attempt{}, fmt.Errorf("task %q was not found", value.TaskID)
	}
	attempt, found := findAttempt(history, value.TaskID, value.AttemptID)
	if !found {
		return producttasks.Task{}, producttasks.Attempt{}, fmt.Errorf("attempt %q was not found for task %q", value.AttemptID, value.TaskID)
	}
	if attempt.RunID != strings.TrimSpace(value.RunID) {
		return producttasks.Task{}, producttasks.Attempt{}, fmt.Errorf("attempt %q is linked to run %q, not %q", value.AttemptID, attempt.RunID, value.RunID)
	}
	return task, attempt, nil
}

func publicationUnavailable(reason string) producttasks.Publication {
	return producttasks.Publication{
		State:             producttasks.PublicationUnavailable,
		UnavailableReason: strings.TrimSpace(reason),
	}
}

func buildTaskPublication(action string, context taskPublicationContext, before, after gitWorkspaceState) producttasks.Publication {
	return producttasks.Publication{
		State:                producttasks.PublicationLinked,
		AttemptID:            strings.TrimSpace(context.AttemptID),
		RunID:                strings.TrimSpace(context.RunID),
		Action:               action,
		Branch:               after.Identity.Branch,
		BaseRef:              before.Identity.BaseRef,
		BaseOID:              before.Identity.BaseOID,
		HeadOID:              after.Identity.HeadOID,
		Commit:               after.Identity.HeadOID,
		InventoryFingerprint: before.Fingerprint,
	}
}

func recordTaskPublication(registry *producttasks.Registry, context taskPublicationContext, publication producttasks.Publication) error {
	if registry == nil {
		return errors.New("task history is not available")
	}
	if err := context.validateShape(); err != nil {
		return err
	}
	if !context.provided() {
		return nil
	}
	return registry.Update(func(history *producttasks.History) error {
		task, attempt, err := validateTaskPublicationContext(*history, context)
		if err != nil {
			return err
		}
		if publication.AttemptID != attempt.AttemptID || publication.RunID != attempt.RunID {
			return errors.New("publication does not match the exact Attempt/run join")
		}
		if index := indexTask(history.Tasks, task.TaskID); index >= 0 {
			history.Tasks[index].Publication = publication
			history.Tasks[index].LastActivityAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		for index := range history.Attempts {
			if history.Attempts[index].TaskID == attempt.TaskID && history.Attempts[index].AttemptID == attempt.AttemptID {
				history.Attempts[index].Publication = publication
				return nil
			}
		}
		return errors.New("attempt disappeared while recording publication")
	})
}
