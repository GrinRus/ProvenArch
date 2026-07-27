package pathscope

import (
	"fmt"
	"path"
	"strings"
)

type Pattern struct {
	raw      string
	segments []string
	subtree  bool
}

func Compile(value string) (Pattern, error) {
	raw := strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if raw == "" || strings.HasPrefix(raw, "/") {
		return Pattern{}, fmt.Errorf("path scope pattern %q must be non-empty and relative", value)
	}
	if len(raw) >= 3 && raw[1] == ':' && raw[2] == '/' {
		return Pattern{}, fmt.Errorf("path scope pattern %q must not use an absolute drive path", value)
	}
	clean := strings.TrimPrefix(path.Clean(raw), "./")
	if clean == "." && raw == "." {
		return Pattern{raw: "."}, nil
	}
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != strings.TrimPrefix(raw, "./") {
		return Pattern{}, fmt.Errorf("path scope pattern %q is not normalized", value)
	}
	segments := strings.Split(clean, "/")
	for _, segment := range segments {
		if strings.Contains(segment, "**") && segment != "**" {
			return Pattern{}, fmt.Errorf("path scope pattern %q may use ** only as a complete segment", value)
		}
		if segment != "**" {
			if _, err := path.Match(segment, "candidate"); err != nil {
				return Pattern{}, fmt.Errorf("invalid path scope pattern %q: %w", value, err)
			}
		}
	}
	return Pattern{raw: clean, segments: segments, subtree: !strings.ContainsAny(clean, "*?[")}, nil
}

func (p Pattern) Match(candidate string) bool {
	normalized := strings.TrimPrefix(path.Clean(strings.ReplaceAll(strings.TrimSpace(candidate), "\\", "/")), "./")
	if normalized == "." {
		return p.raw == "."
	}
	if normalized == ".." || strings.HasPrefix(normalized, "../") || strings.HasPrefix(normalized, "/") {
		return false
	}
	if matchSegments(p.segments, strings.Split(normalized, "/")) {
		return true
	}
	return p.subtree && strings.HasPrefix(normalized, p.raw+"/")
}

func Match(pattern, candidate string) bool {
	compiled, err := Compile(pattern)
	return err == nil && compiled.Match(candidate)
}

func matchSegments(pattern, candidate []string) bool {
	if len(pattern) == 0 {
		return len(candidate) == 0
	}
	if pattern[0] == "**" {
		return matchSegments(pattern[1:], candidate) ||
			(len(candidate) > 0 && matchSegments(pattern, candidate[1:]))
	}
	if len(candidate) == 0 {
		return false
	}
	matched, err := path.Match(pattern[0], candidate[0])
	return err == nil && matched && matchSegments(pattern[1:], candidate[1:])
}
