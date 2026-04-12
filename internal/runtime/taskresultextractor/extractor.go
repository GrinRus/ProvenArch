package taskresultextractor

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

// Extract returns a schema-valid TaskResult JSON object extracted from runner stdout.
func Extract(raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("empty stdout")
	}
	if _, err := contracts.ParseTaskResult(trimmed); err == nil {
		return trimmed, nil
	}

	if parsed, err := parseFromJSON(trimmed); err == nil {
		return parsed, nil
	}
	if parsed, err := parseFromText(string(trimmed)); err == nil {
		return parsed, nil
	}

	return nil, errors.New("unable to extract valid TaskResult JSON from runner output")
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
		return parseFromText(typed)
	case map[string]any:
		raw, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		if _, err := contracts.ParseTaskResult(raw); err == nil {
			return raw, nil
		}

		priorityKeys := []string{"result", "response", "message", "content", "text"}
		seen := map[string]struct{}{}
		for _, key := range priorityKeys {
			seen[key] = struct{}{}
			nested, ok := typed[key]
			if !ok {
				continue
			}
			if parsed, nestedErr := parseCandidate(nested); nestedErr == nil {
				return parsed, nil
			}
		}
		for key, nested := range typed {
			if _, ok := seen[key]; ok {
				continue
			}
			if parsed, nestedErr := parseCandidate(nested); nestedErr == nil {
				return parsed, nil
			}
		}
	case []any:
		for idx := len(typed) - 1; idx >= 0; idx-- {
			if parsed, err := parseCandidate(typed[idx]); err == nil {
				return parsed, nil
			}
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

	if json.Valid([]byte(trimmed)) {
		if parsed, err := parseFromJSON([]byte(trimmed)); err == nil {
			return parsed, nil
		}
	}
	if parsed, err := parseFromNDJSON(trimmed); err == nil {
		return parsed, nil
	}

	candidates := extractJSONObjects(trimmed)
	if len(candidates) == 0 {
		return nil, errors.New("no json object found in text")
	}
	for _, candidate := range candidates {
		if parsed, err := parseFromJSON([]byte(candidate)); err == nil {
			return parsed, nil
		}
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
	for idx := len(candidates) - 1; idx >= 0; idx-- {
		if parsed, err := parseFromJSON([]byte(candidates[idx])); err == nil {
			return parsed, nil
		}
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
