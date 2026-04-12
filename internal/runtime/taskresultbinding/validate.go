package taskresultbinding

import (
	"fmt"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

// Validate checks that TaskResult metadata is bound to the requested runtime task.
func Validate(task acpruntime.Task, result contracts.TaskResult, provider acpruntime.Provider) error {
	problems := []string{}

	taskID := strings.TrimSpace(task.TaskID)
	if strings.TrimSpace(result.Meta.TaskID) != taskID {
		problems = append(problems, fmt.Sprintf("meta.task_id=%q expected %q", result.Meta.TaskID, taskID))
	}

	stepID := strings.TrimSpace(task.StepID)
	if strings.TrimSpace(result.Meta.StepID) != stepID {
		problems = append(problems, fmt.Sprintf("meta.step_id=%q expected %q", result.Meta.StepID, stepID))
	}

	expectedRunID := strings.TrimSpace(task.RunID)
	gotRunID := strings.TrimSpace(result.Meta.RunID)
	if gotRunID != "" && gotRunID != expectedRunID {
		problems = append(problems, fmt.Sprintf("meta.run_id=%q expected %q", result.Meta.RunID, expectedRunID))
	}

	expectedRuntimeName := string(provider)
	if strings.TrimSpace(result.Meta.Runtime.Name) != expectedRuntimeName {
		problems = append(problems, fmt.Sprintf("meta.runtime.name=%q expected %q", result.Meta.Runtime.Name, expectedRuntimeName))
	}

	if len(problems) > 0 {
		return fmt.Errorf("taskresult binding mismatch: %s", strings.Join(problems, "; "))
	}
	return nil
}
