package secretredact

import (
	"regexp"
	"strings"
)

var (
	bearerPattern          = regexp.MustCompile(`(?i)(Authorization\s*:\s*Bearer\s+)[^\s,;"']+`)
	cliFlagPattern         = regexp.MustCompile(`(?i)(--(?:api[-_]?key|token|secret|password)(?:=|\s+))("[^"]*"|'[^']*'|[^\s,;]+)`)
	doubleQuotedKeyPattern = regexp.MustCompile(`(?i)(["']?(?:api[-_]?key|token|secret|password)["']?\s*[:=]\s*")([^"]*)(")`)
	singleQuotedKeyPattern = regexp.MustCompile(`(?i)(["']?(?:api[-_]?key|token|secret|password)["']?\s*[:=]\s*')([^']*)(')`)
	bareKeyPattern         = regexp.MustCompile(`(?i)(["']?(?:api[-_]?key|token|secret|password)["']?\s*[:=]\s*)([^"'\s,;}]+)`)
)

func RedactText(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	redacted := bearerPattern.ReplaceAllString(text, `${1}<redacted>`)
	redacted = cliFlagPattern.ReplaceAllString(redacted, `${1}<redacted>`)
	redacted = doubleQuotedKeyPattern.ReplaceAllString(redacted, `${1}<redacted>${3}`)
	redacted = singleQuotedKeyPattern.ReplaceAllString(redacted, `${1}<redacted>${3}`)
	redacted = bareKeyPattern.ReplaceAllString(redacted, `${1}<redacted>`)
	return redacted
}
