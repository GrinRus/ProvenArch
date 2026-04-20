package taskresultextractor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

type TransportError struct {
	Detail string
}

func (e TransportError) Error() string {
	detail := strings.TrimSpace(e.Detail)
	if detail == "" {
		return "provider transport error"
	}
	return "provider transport error: " + detail
}

func IsTransportError(err error) bool {
	var transportErr TransportError
	return errors.As(err, &transportErr)
}

// Extract returns a schema-valid TaskResult JSON object extracted from runner stdout.
func Extract(raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("empty stdout")
	}

	var bestErr error
	if parsed, parseErr := parseFromJSON(trimmed); parseErr == nil {
		return parsed, nil
	} else {
		bestErr = preferExtractError(bestErr, parseErr)
	}
	if parsed, parseErr := parseFromText(string(trimmed)); parseErr == nil {
		return parsed, nil
	} else {
		bestErr = preferExtractError(bestErr, parseErr)
	}
	if transportErr := detectTransportError(string(trimmed)); transportErr != nil {
		return nil, fmt.Errorf("unable to extract valid TaskResult JSON from runner output: %w", transportErr)
	}
	if bestErr == nil {
		bestErr = errors.New("unknown extraction error")
	}
	return nil, fmt.Errorf("unable to extract valid TaskResult JSON from runner output: %w", bestErr)
}

func parseFromJSON(raw []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return parseCandidate(value)
}

func parseCandidate(value any) ([]byte, error) {
	switch typed := value.(type) {
	case string:
		parsed, err := parseFromText(typed)
		if err != nil {
			return nil, fmt.Errorf("string candidate parse failed: %w", err)
		}
		return parsed, nil
	case map[string]any:
		raw, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		if looksLikeTaskResultObject(typed) {
			return raw, nil
		}

		bestErr := errors.New("taskresult candidate object is missing top-level keys")
		priorityKeys := []string{"result", "response", "message", "content", "text"}
		seen := map[string]struct{}{}
		for _, key := range priorityKeys {
			seen[key] = struct{}{}
			nested, ok := typed[key]
			if !ok {
				continue
			}
			if key == "result" {
				if resultValue, isString := nested.(string); isString && strings.TrimSpace(resultValue) == "" {
					bestErr = preferExtractError(bestErr, errors.New("envelope result is empty"))
					continue
				}
			}
			if parsed, nestedErr := parseCandidate(nested); nestedErr == nil {
				return parsed, nil
			} else {
				bestErr = preferExtractError(bestErr, fmt.Errorf("envelope key %q: %w", key, nestedErr))
			}
		}
		for key, nested := range typed {
			if _, ok := seen[key]; ok {
				continue
			}
			if parsed, nestedErr := parseCandidate(nested); nestedErr == nil {
				return parsed, nil
			} else {
				bestErr = preferExtractError(bestErr, fmt.Errorf("object key %q: %w", key, nestedErr))
			}
		}
		if bestErr != nil {
			return nil, bestErr
		}
	case []any:
		var bestErr error
		for idx := len(typed) - 1; idx >= 0; idx-- {
			if parsed, err := parseCandidate(typed[idx]); err == nil {
				return parsed, nil
			} else {
				bestErr = preferExtractError(bestErr, fmt.Errorf("array item[%d]: %w", idx, err))
			}
		}
		if bestErr != nil {
			return nil, bestErr
		}
	}
	return nil, errors.New("unsupported candidate value")
}

func parseFromText(text string) ([]byte, error) {
	trimmed := normalizeText(text)
	if trimmed == "" {
		return nil, errors.New("empty text")
	}
	if candidate, ok := stripCodeFence(trimmed); ok {
		trimmed = normalizeText(candidate)
	}

	var bestErr error
	if json.Valid([]byte(trimmed)) {
		if parsed, err := parseFromJSON([]byte(trimmed)); err == nil {
			return parsed, nil
		} else {
			bestErr = preferExtractError(bestErr, err)
		}
	}
	if parsed, err := parseFromNDJSON(trimmed); err == nil {
		return parsed, nil
	} else {
		bestErr = preferExtractError(bestErr, err)
	}

	candidates := extractJSONObjects(trimmed)
	if len(candidates) == 0 {
		if bestErr != nil {
			return nil, bestErr
		}
		return nil, errors.New("no json object found in text")
	}
	for _, candidate := range candidates {
		if parsed, err := parseFromJSON([]byte(candidate)); err == nil {
			return parsed, nil
		} else {
			bestErr = preferExtractError(bestErr, err)
		}
	}
	if bestErr != nil {
		return nil, bestErr
	}
	return nil, errors.New("unable to parse taskresult from extracted json objects")
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

func normalizeText(input string) string {
	if input == "" {
		return ""
	}
	normalized := strings.ReplaceAll(input, "\ufeff", "")
	normalized = ansiEscapePattern.ReplaceAllString(normalized, "")
	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, r := range normalized {
		if r == '\n' || r == '\r' || r == '\t' || r >= 0x20 {
			builder.WriteRune(r)
		}
	}
	return strings.TrimSpace(builder.String())
}

func parseFromNDJSON(text string) ([]byte, error) {
	lines := strings.Split(text, "\n")
	if len(lines) <= 1 {
		return nil, errors.New("not ndjson")
	}
	candidates := make([]string, 0, len(lines))
	for _, line := range lines {
		candidate := normalizeNDJSONLine(line)
		if candidate == "" || !json.Valid([]byte(candidate)) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil, errors.New("no ndjson candidates")
	}
	var bestErr error
	for idx := len(candidates) - 1; idx >= 0; idx-- {
		if parsed, err := parseFromJSON([]byte(candidates[idx])); err == nil {
			return parsed, nil
		} else {
			bestErr = preferExtractError(bestErr, err)
		}
	}
	if bestErr != nil {
		return nil, bestErr
	}
	return nil, errors.New("unable to parse taskresult from ndjson")
}

func normalizeNDJSONLine(line string) string {
	candidate := normalizeText(line)
	if candidate == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(candidate), "data:") {
		candidate = strings.TrimSpace(candidate[5:])
	}
	if strings.HasPrefix(strings.ToLower(candidate), "event:") {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(candidate), "id:") {
		return ""
	}
	if idx := strings.IndexAny(candidate, "{["); idx > 0 {
		candidate = strings.TrimSpace(candidate[idx:])
	}
	return candidate
}

func extractJSONObjects(input string) []string {
	candidates := make([]string, 0, 4)
	start := strings.IndexByte(input, '{')
	for start >= 0 {
		depth := 0
		inString := false
		escapeNext := false
		found := false
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
						candidates = append(candidates, candidate)
					}
					found = true
					break
				}
			}
		}
		if !found {
			break
		}
		next := strings.IndexByte(input[start+1:], '{')
		if next < 0 {
			break
		}
		start = start + 1 + next
	}
	return candidates
}

func preferExtractError(current error, candidate error) error {
	if candidate == nil {
		return current
	}
	if current == nil {
		return candidate
	}
	currentText := strings.ToLower(strings.TrimSpace(current.Error()))
	candidateText := strings.ToLower(strings.TrimSpace(candidate.Error()))
	if strings.Contains(candidateText, "envelope result is empty") {
		return candidate
	}
	if strings.Contains(candidateText, "envelope key \"result\"") {
		return candidate
	}
	if strings.Contains(currentText, "unsupported candidate value") {
		return candidate
	}
	if strings.Contains(currentText, "unable to parse taskresult") && !strings.Contains(candidateText, "unable to parse taskresult") {
		return candidate
	}
	if strings.Contains(candidateText, "taskresult") && !strings.Contains(currentText, "taskresult") {
		return candidate
	}
	return current
}

func looksLikeTaskResultObject(value map[string]any) bool {
	if value == nil {
		return false
	}
	if _, ok := value["meta"]; ok {
		return true
	}
	if _, ok := value["changeset"]; ok {
		return true
	}
	return false
}

func detectTransportError(text string) error {
	normalized := normalizeText(text)
	if normalized == "" {
		return nil
	}
	lower := strings.ToLower(normalized)
	index := strings.LastIndex(lower, "[api error:")
	if index < 0 {
		index = strings.LastIndex(lower, "api error:")
	}
	if index < 0 {
		return nil
	}

	excerpt := strings.TrimSpace(normalized[index:])
	if newline := strings.IndexAny(excerpt, "\r\n"); newline >= 0 {
		excerpt = strings.TrimSpace(excerpt[:newline])
	}
	lowerExcerpt := strings.ToLower(excerpt)
	transportMarkers := []string{
		"ssl",
		"tls",
		"certificate",
		"connection",
		"transport",
		"timeout",
		"socket",
		"network",
		"http2",
		"packet length too long",
		"econn",
	}
	for _, marker := range transportMarkers {
		if strings.Contains(lowerExcerpt, marker) {
			return TransportError{Detail: compactTransportExcerpt(excerpt)}
		}
	}
	return nil
}

func compactTransportExcerpt(value string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(normalized) <= 320 {
		return normalized
	}
	return normalized[:317] + "..."
}
