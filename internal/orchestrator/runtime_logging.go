package orchestrator

import (
	"errors"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/secretredact"
)

func (e *pipelineExecution) addWarning(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	for _, existing := range e.warnings {
		if strings.EqualFold(strings.TrimSpace(existing), message) {
			return
		}
	}
	e.warnings = append(e.warnings, message)
}

func (e *pipelineExecution) logInfo(stepID string, domainID string, message string, fields map[string]any) {
	e.logRunEvent(RunLogLevelInfo, stepID, domainID, message, fields)
}

func (e *pipelineExecution) logWarn(stepID string, domainID string, message string, fields map[string]any) {
	e.logRunEvent(RunLogLevelWarning, stepID, domainID, message, fields)
}

func (e *pipelineExecution) logError(stepID string, domainID string, message string, fields map[string]any) {
	e.logRunEvent(RunLogLevelError, stepID, domainID, message, fields)
}

func (e *pipelineExecution) logRuntimeOutput(stepID string, domainID string, provider acpruntime.Provider, chunk acpruntime.OutputChunk) {
	message := secretredact.RedactText(strings.TrimRight(chunk.Text, "\r\n"))
	if strings.TrimSpace(message) == "" {
		return
	}
	fields := map[string]any{}
	level := RunLogLevelInfo
	if chunk.Truncated {
		fields["output_truncated"] = true
		fields["stream"] = strings.TrimSpace(string(chunk.Stream))
	}
	entry := RunLogEntry{
		Timestamp: e.clock().UTC(),
		Level:     level,
		Kind:      RunLogKindRuntimeOutput,
		Stream:    strings.TrimSpace(string(chunk.Stream)),
		StepID:    strings.TrimSpace(stepID),
		DomainID:  strings.TrimSpace(domainID),
		Message:   message,
	}
	if len(fields) > 0 {
		entry.Fields = fields
	}
	if e.onLog != nil {
		e.onLog(entry)
	}
}

func (e *pipelineExecution) decideRuntimePermission(task acpruntime.Task, request acpruntime.PermissionRequest) acpruntime.PermissionDecision {
	if strings.TrimSpace(request.RunID) == "" {
		request.RunID = task.RunID
	}
	if strings.TrimSpace(request.StepID) == "" {
		request.StepID = task.StepID
	}
	decision := acpruntime.DecideRuntimePermission(task, request)
	fields := map[string]any{
		"request_id":       request.RequestID,
		"action":           request.Action,
		"path_or_command":  request.PathOrCommand,
		"decision":         decision.Decision,
		"rule_id":          decision.RuleID,
		"permissions_mode": task.RuntimePermissions.Mode,
		"approval_channel": task.RuntimePermissions.ApprovalChannel,
	}
	if strings.TrimSpace(decision.Message) != "" {
		fields["message"] = decision.Message
	}
	level := RunLogLevelInfo
	message := "runtime permission decision"
	if decision.Decision == acpruntime.PermissionDecisionDenied {
		level = RunLogLevelWarning
	}
	if decision.Decision == acpruntime.PermissionDecisionNeedsUser {
		level = RunLogLevelWarning
		message = "runtime permission request requires user approval"
		storedDecision := decision
		request.Decision = &storedDecision
		e.pendingPermissions = append(e.pendingPermissions, request)
		if e.onPermissions != nil {
			e.onPermissions(append([]acpruntime.PermissionRequest(nil), e.pendingPermissions...))
		}
	}
	e.logRunEvent(level, request.StepID, task.DomainID, message, fields)
	return decision
}

func (e *pipelineExecution) logRunEvent(level RunLogLevel, stepID string, domainID string, message string, fields map[string]any) {
	if e.onLog == nil {
		return
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	entry := RunLogEntry{
		Timestamp: e.clock().UTC(),
		Level:     level,
		StepID:    strings.TrimSpace(stepID),
		DomainID:  strings.TrimSpace(domainID),
		Message:   message,
	}
	if len(fields) > 0 {
		entry.Fields = fields
		if taskrunPath, ok := fields["taskrun_path"].(string); ok {
			entry.TaskrunPath = strings.TrimSpace(taskrunPath)
		}
	}
	e.onLog(entry)
}

func runtimeFailureLogFields(task acpruntime.Task, err error, fallbackStdout string, fallbackStderr string) map[string]any {
	fields := map[string]any{
		"task_id":     task.TaskID,
		"shard_id":    task.ShardID,
		"repo_scope":  task.RepoScope,
		"repo_scopes": append([]string(nil), task.RepoScopes...),
		"path_scopes": append([]string(nil), task.PathScopes...),
		"error":       strings.TrimSpace(err.Error()),
	}
	if runtimeCode, _, ok := acpruntime.ClassifyError(err); ok {
		fields["error_code"] = runtimeCode
	}

	stdout := fallbackStdout
	stderr := fallbackStderr
	var runnerErr acpruntime.RunnerError
	if errors.As(err, &runnerErr) {
		if strings.TrimSpace(string(runnerErr.Provider)) != "" {
			fields["provider"] = string(runnerErr.Provider)
		}
		if strings.TrimSpace(stdout) == "" {
			stdout = runnerErr.Stdout
		}
		if strings.TrimSpace(stderr) == "" {
			stderr = runnerErr.Stderr
		}
	}

	appendSnippetField(fields, "stdout_snippet", stdout)
	appendSnippetField(fields, "stderr_snippet", stderr)
	return fields
}

func appendSnippetField(fields map[string]any, key string, raw string) {
	if fields == nil {
		return
	}
	snippet := sanitizeAndTruncateSnippet(raw, runtimeOutputSnippetLimitRunes)
	if snippet == "" {
		return
	}
	fields[key] = snippet
}

func sanitizeAndTruncateSnippet(raw string, limitRunes int) string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return ""
	}
	if limitRunes <= 0 {
		limitRunes = runtimeOutputSnippetLimitRunes
	}
	runes := []rune(normalized)
	if len(runes) <= limitRunes {
		return normalized
	}
	truncated := strings.TrimSpace(string(runes[:limitRunes]))
	if truncated == "" {
		truncated = string(runes[:limitRunes])
	}
	return truncated + runtimeOutputSnippetSuffix
}
