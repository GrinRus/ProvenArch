package runtimedrafts

import (
	"fmt"
	"strings"
)

var architectureHomeRequiredSections = []string{
	"System at a glance",
	"Analyzed scope",
	"Domains and ownership",
	"Key flows",
	"Integrations and datastores",
	"Where to start",
	"Safe-change guidance",
	"Evidence gaps and open questions",
}

// ArchitectureHomeRequiredSections returns the canonical section labels in
// their expected order. The returned slice is detached from validator state.
func ArchitectureHomeRequiredSections() []string {
	return append([]string(nil), architectureHomeRequiredSections...)
}

// NormalizeArchitectureHomeInlineHeadings repairs one narrow Markdown shape:
// every required H2 exists exactly once and in canonical order, but its
// substantive body starts on the same physical line. It never authors or
// rewrites body content.
func NormalizeArchitectureHomeInlineHeadings(raw []byte) ([]byte, []string, error) {
	text := string(raw)
	separator := "\n"
	if strings.Contains(text, "\r\n") {
		if strings.Contains(strings.ReplaceAll(text, "\r\n", ""), "\r") {
			return nil, nil, fmt.Errorf("Architecture Home uses mixed line endings")
		}
		separator = "\r\n"
	} else if strings.Contains(text, "\r") {
		return nil, nil, fmt.Errorf("Architecture Home uses unsupported line endings")
	}

	lines := strings.Split(text, separator)
	normalized := make([]string, 0, len(lines)+2*len(architectureHomeRequiredSections))
	found := make([]string, 0, len(architectureHomeRequiredSections))
	nextRequired := 0
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
		}
		matched := false
		for _, section := range architectureHomeRequiredSections {
			heading := "## " + section
			if line == heading {
				return nil, nil, fmt.Errorf("Architecture Home already contains standalone required heading %q", section)
			}
			if !strings.HasPrefix(line, heading) || len(line) == len(heading) {
				continue
			}
			separatorByte := line[len(heading)]
			if separatorByte != ' ' && separatorByte != '\t' {
				continue
			}
			if inFence {
				return nil, nil, fmt.Errorf("Architecture Home required heading %q appears inside a code fence", section)
			}
			if nextRequired >= len(architectureHomeRequiredSections) || architectureHomeRequiredSections[nextRequired] != section {
				return nil, nil, fmt.Errorf("Architecture Home inline required headings are duplicated or out of order at %q", section)
			}
			body := line[len(heading)+1:]
			if strings.TrimSpace(body) == "" {
				return nil, nil, fmt.Errorf("Architecture Home inline required heading %q has an empty body", section)
			}
			normalized = append(normalized, heading, "", body)
			found = append(found, section)
			nextRequired++
			matched = true
			break
		}
		if !matched {
			normalized = append(normalized, line)
		}
	}
	if inFence {
		return nil, nil, fmt.Errorf("Architecture Home has an unclosed code fence")
	}
	if len(found) != len(architectureHomeRequiredSections) {
		return nil, nil, fmt.Errorf("Architecture Home has %d eligible inline required headings; want %d", len(found), len(architectureHomeRequiredSections))
	}
	return []byte(strings.Join(normalized, separator)), found, nil
}
