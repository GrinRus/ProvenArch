package qwencode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

var (
	ErrRunnerUnavailable = errors.New("qwen-code runner is unavailable")
)

type HeadlessRunner struct {
	Command string
	Args    []string
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_QWEN_CMD"))
	}
	if command == "" {
		command = "qwen"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	command := r.commandName()
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", acpruntime.ProviderQwenCode, command, err),
			err,
		)
	}
	return nil
}

func (r HeadlessRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := r.Preflight(ctx); err != nil {
		return acpruntime.Result{}, err
	}
	command := r.commandName()

	taskPayload, err := json.Marshal(task)
	if err != nil {
		return acpruntime.Result{}, fmt.Errorf("marshal runner task: %w", err)
	}

	args := append([]string(nil), r.Args...)
	if len(args) == 0 {
		args = []string{"--prompt", buildPrompt(taskPayload), "--output-format", "json", "--yolo"}
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = bytes.NewReader(taskPayload)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			errText = err.Error()
		}
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("%v: %s", ErrRunnerUnavailable, errText),
			err,
		)
	}

	rawTaskResult, err := extractTaskResultJSON(stdout.Bytes())
	if err != nil {
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q returned invalid taskresult: %v", acpruntime.ProviderQwenCode, err),
			err,
		)
	}
	taskResult, err := contracts.ParseTaskResult(rawTaskResult)
	if err != nil {
		return acpruntime.Result{}, acpruntime.WrapRunnerError(
			acpruntime.ProviderQwenCode,
			acpruntime.ErrorCodeRunnerParseFailed,
			fmt.Sprintf("headless provider %q returned invalid taskresult: %v", acpruntime.ProviderQwenCode, err),
			err,
		)
	}

	return acpruntime.Result{
		TaskResult: taskResult,
		RawJSON:    rawTaskResult,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}, nil
}

func buildPrompt(taskPayload []byte) string {
	return strings.TrimSpace(fmt.Sprintf(`You are ACP runtime provider %q.
Return exactly one valid JSON object for ACP TaskResult.
Do not output markdown, code fences, or any explanatory text.
If evidence is missing, use questions/coverage/findings per ACP contract.
Task payload JSON:
%s`, acpruntime.ProviderQwenCode, strings.TrimSpace(string(taskPayload))))
}

func extractTaskResultJSON(raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("empty stdout")
	}
	if _, err := contracts.ParseTaskResult(trimmed); err == nil {
		return trimmed, nil
	}

	if parsed, err := extractFromJSONObject(trimmed); err == nil {
		return parsed, nil
	}
	if parsed, err := extractFromJSONArray(trimmed); err == nil {
		return parsed, nil
	}
	if parsed, err := parseTaskResultFromText(string(trimmed)); err == nil {
		return parsed, nil
	}

	return nil, errors.New("unable to extract valid TaskResult JSON from qwen output")
}

func extractFromJSONObject(raw []byte) ([]byte, error) {
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, err
	}

	if value, ok := object["response"]; ok {
		if parsed, err := parseCandidateValue(value); err == nil {
			return parsed, nil
		}
	}
	if value, ok := object["result"]; ok {
		if parsed, err := parseCandidateValue(value); err == nil {
			return parsed, nil
		}
	}
	if value, ok := object["message"]; ok {
		if parsed, err := parseCandidateValue(value); err == nil {
			return parsed, nil
		}
	}

	return nil, errors.New("no taskresult in object output")
}

func extractFromJSONArray(raw []byte) ([]byte, error) {
	var array []map[string]any
	if err := json.Unmarshal(raw, &array); err != nil {
		return nil, err
	}
	for idx := len(array) - 1; idx >= 0; idx-- {
		item := array[idx]
		if value, ok := item["result"]; ok {
			if parsed, err := parseCandidateValue(value); err == nil {
				return parsed, nil
			}
		}
		if message, ok := item["message"]; ok {
			if parsed, err := parseCandidateValue(message); err == nil {
				return parsed, nil
			}
		}
	}
	return nil, errors.New("no taskresult in array output")
}

func parseCandidateValue(value any) ([]byte, error) {
	switch typed := value.(type) {
	case string:
		return parseTaskResultFromText(typed)
	case map[string]any:
		raw, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		if _, err := contracts.ParseTaskResult(raw); err != nil {
			return nil, err
		}
		return raw, nil
	case []any:
		for _, item := range typed {
			if parsed, err := parseCandidateValue(item); err == nil {
				return parsed, nil
			}
		}
	}
	return nil, errors.New("unsupported candidate value")
}

func parseTaskResultFromText(text string) ([]byte, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, errors.New("empty text")
	}

	if candidate, ok := stripCodeFence(trimmed); ok {
		trimmed = candidate
	}

	if json.Valid([]byte(trimmed)) {
		raw := []byte(trimmed)
		if _, err := contracts.ParseTaskResult(raw); err == nil {
			return raw, nil
		}
	}

	candidate, ok := extractJSONObject(trimmed)
	if !ok {
		return nil, errors.New("no json object in text")
	}
	raw := []byte(candidate)
	if _, err := contracts.ParseTaskResult(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func stripCodeFence(input string) (string, bool) {
	if !strings.HasPrefix(input, "```") {
		return "", false
	}
	withoutPrefix := input[3:]
	if newline := strings.IndexByte(withoutPrefix, '\n'); newline >= 0 {
		withoutPrefix = withoutPrefix[newline+1:]
	}
	if tail := strings.LastIndex(withoutPrefix, "```"); tail >= 0 {
		withoutPrefix = withoutPrefix[:tail]
	}
	candidate := strings.TrimSpace(withoutPrefix)
	if candidate == "" {
		return "", false
	}
	return candidate, true
}

func extractJSONObject(input string) (string, bool) {
	start := strings.IndexByte(input, '{')
	for start >= 0 {
		depth := 0
		inString := false
		escapeNext := false
		for idx := start; idx < len(input); idx++ {
			ch := input[idx]
			if inString {
				if escapeNext {
					escapeNext = false
					continue
				}
				switch ch {
				case '\\':
					escapeNext = true
				case '"':
					inString = false
				}
				continue
			}
			switch ch {
			case '"':
				inString = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					candidate := strings.TrimSpace(input[start : idx+1])
					if candidate != "" && json.Valid([]byte(candidate)) {
						return candidate, true
					}
					break
				}
			}
		}
		next := strings.IndexByte(input[start+1:], '{')
		if next < 0 {
			break
		}
		start = start + 1 + next
	}
	return "", false
}
