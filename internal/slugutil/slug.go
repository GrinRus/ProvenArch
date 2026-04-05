package slugutil

import "strings"

// Slugify normalizes arbitrary user/runtime text into stable lowercase slug tokens.
func Slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}

	var out []rune
	prevDash := false
	for _, r := range value {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isDigit {
			out = append(out, r)
			prevDash = false
			continue
		}
		if prevDash {
			continue
		}
		out = append(out, '-')
		prevDash = true
	}

	slug := strings.Trim(string(out), "-")
	if slug == "" {
		return "unknown"
	}
	return slug
}
