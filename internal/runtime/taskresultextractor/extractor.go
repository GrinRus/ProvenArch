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
var providerQuotaOrPermissionPattern = regexp.MustCompile(`(?is)(api\s*error\s*:\s*403|permission_error|insufficient_quota|usage\s+limit|quota(?:\s+will\s+be\s+refreshed|\s+exceeded|\s+limit)|forbidden)`)

type ProviderAvailabilitySignal struct {
	Subreason string
	Message   string
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
	if bestErr == nil {
		bestErr = errors.New("unknown extraction error")
	}
	return nil, fmt.Errorf("unable to extract valid TaskResult JSON from runner output: %w", bestErr)
}

// DetectProviderAvailabilitySignal inspects runner output for explicit provider
// availability failures (for example permission/quota API responses) so callers
// can classify them as execution failures before parse-retry.
func DetectProviderAvailabilitySignal(stdout []byte, stderr []byte) (ProviderAvailabilitySignal, bool) {
	candidates := []string{}
	addCandidate := func(raw []byte) {
		normalized := normalizeText(string(raw))
		if normalized != "" {
			candidates = append(candidates, normalized)
		}
		var value any
		if err := json.Unmarshal(raw, &value); err == nil {
			candidates = append(candidates, collectStringCandidates(value)...)
		}
	}

	addCandidate(stdout)
	addCandidate(stderr)
	for _, candidate := range candidates {
		if signal, ok := detectProviderAvailabilitySignalInText(candidate); ok {
			return signal, true
		}
	}
	return ProviderAvailabilitySignal{}, false
}

func collectStringCandidates(value any) []string {
	candidates := []string{}
	var walk func(any)
	walk = func(node any) {
		switch typed := node.(type) {
		case string:
			text := normalizeText(typed)
			if text != "" {
				candidates = append(candidates, text)
			}
		case []any:
			for _, item := range typed {
				walk(item)
			}
		case map[string]any:
			priorityKeys := []string{"result", "message", "content", "text", "error"}
			for _, key := range priorityKeys {
				if value, ok := typed[key]; ok {
					walk(value)
				}
			}
			for _, value := range typed {
				walk(value)
			}
		}
	}
	walk(value)
	return candidates
}

func detectProviderAvailabilitySignalInText(text string) (ProviderAvailabilitySignal, bool) {
	normalized := normalizeText(text)
	if normalized == "" {
		return ProviderAvailabilitySignal{}, false
	}
	match := strings.TrimSpace(providerQuotaOrPermissionPattern.FindString(normalized))
	if match == "" {
		return ProviderAvailabilitySignal{}, false
	}
	return ProviderAvailabilitySignal{
		Subreason: "quota_or_permission",
		Message:   compactSignalMessage(normalized, match),
	}, true
}

func compactSignalMessage(fullText string, match string) string {
	text := strings.TrimSpace(fullText)
	if text == "" {
		return strings.TrimSpace(match)
	}
	if len(text) <= 320 {
		return text
	}
	lowerText := strings.ToLower(text)
	lowerMatch := strings.ToLower(strings.TrimSpace(match))
	idx := strings.Index(lowerText, lowerMatch)
	if idx < 0 {
		return text[:317] + "..."
	}
	start := idx - 96
	if start < 0 {
		start = 0
	}
	end := idx + len(lowerMatch) + 160
	if end > len(text) {
		end = len(text)
	}
	window := strings.TrimSpace(text[start:end])
	if start > 0 {
		window = "..." + window
	}
	if end < len(text) {
		window += "..."
	}
	return window
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
