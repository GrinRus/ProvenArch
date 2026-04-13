package runnerdiag

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const defaultStdoutExcerptLimit = 320

// BuildExecFailure builds a stable execution failure message from command outputs.
// Preference order:
// 1) stderr text (if non-empty)
// 2) command error + trimmed stdout excerpt
// 3) command error text
func BuildExecFailure(runErr error, stdout string, stderr string) error {
	stderrText := strings.TrimSpace(stderr)
	if stderrText != "" {
		return errors.New(stderrText)
	}

	stdoutExcerpt := sanitizeOutputSnippet(stdout, defaultStdoutExcerptLimit)
	if stdoutExcerpt != "" {
		return fmt.Errorf("%v (stdout_excerpt=%q)", runErr, stdoutExcerpt)
	}

	return errors.New(strings.TrimSpace(runErr.Error()))
}

func sanitizeOutputSnippet(raw string, limit int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(trimmed))
	lastWasSpace := false
	for _, r := range trimmed {
		if unicode.IsSpace(r) {
			if lastWasSpace {
				continue
			}
			b.WriteRune(' ')
			lastWasSpace = true
			continue
		}
		if !unicode.IsPrint(r) {
			continue
		}
		b.WriteRune(r)
		lastWasSpace = false
	}

	snippet := strings.TrimSpace(b.String())
	if snippet == "" {
		return ""
	}
	if limit > 0 && len(snippet) > limit {
		return strings.TrimSpace(snippet[:limit]) + "..."
	}
	return snippet
}
