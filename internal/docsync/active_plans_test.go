package docsync

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func activePlanErrors(content string) []error {
	parts := strings.SplitN(content, "\n## Active Plans\n", 2)
	if len(parts) != 2 {
		return []error{fmt.Errorf("missing Active Plans boundary")}
	}
	var errors []error
	index := map[string]string{}
	rows := regexp.MustCompile("(?m)^\\| \\[(EP-[^]]+)\\]\\(#([^)]+)\\) \\| (active|blocked|review) \\|")
	for _, row := range rows.FindAllStringSubmatch(parts[0], -1) {
		if _, duplicate := index[row[1]]; duplicate {
			errors = append(errors, fmt.Errorf("duplicate plan index row: %s", row[1]))
		}
		anchor := strings.ToLower(strings.ReplaceAll(row[1], ".", ""))
		if row[2] != anchor {
			errors = append(errors, fmt.Errorf("wrong plan anchor for %s: %s", row[1], row[2]))
		}
		index[row[1]] = row[3]
	}
	active := parts[1]
	if strings.Contains(active, "\n### Plan ID\n") {
		errors = append(errors, fmt.Errorf("unindexed legacy Plan ID section; use a named H2 plan"))
	}
	headings := regexp.MustCompile("(?m)^## ([^\n]+)").FindAllStringSubmatchIndex(active, -1)
	seen := map[string]bool{}
	for i, heading := range headings {
		id := strings.TrimSpace(active[heading[2]:heading[3]])
		if !strings.HasPrefix(id, "EP-") {
			continue
		}
		if seen[id] {
			errors = append(errors, fmt.Errorf("duplicate plan body: %s", id))
		}
		seen[id] = true
		end := len(active)
		if i+1 < len(headings) {
			end = headings[i+1][0]
		}
		body := active[heading[1]:end]
		status := regexp.MustCompile("(?m)^Status: (active|blocked|review)(?: — [^\n]+)?\\.?$").FindStringSubmatch(body)
		if status == nil {
			errors = append(errors, fmt.Errorf("plan %s needs an explicit status", id))
		} else if index[id] != status[1] {
			errors = append(errors, fmt.Errorf("plan %s index/body status mismatch or missing index", id))
		}
		if !regexp.MustCompile("(?m)^Next action: \\S.+").MatchString(body) {
			errors = append(errors, fmt.Errorf("plan %s needs an unresolved next action", id))
		}
		if status != nil && status[1] == "active" {
			goalHeading := regexp.MustCompile("(?m)^### Goals?(?: \\(must have\\))?\n").FindStringIndex(body)
			if goalHeading != nil {
				goals := body[goalHeading[1]:]
				if next := strings.Index(goals, "\n### "); next >= 0 {
					goals = goals[:next]
				}
				if strings.Contains(goals, "- [x]") && !strings.Contains(goals, "- [ ]") {
					errors = append(errors, fmt.Errorf("plan %s has closed goals; archive or record remaining review/blocker", id))
				}
			}
		}
	}
	if len(seen) == 0 {
		errors = append(errors, fmt.Errorf("no indexed active plan bodies found"))
	}
	for id := range index {
		if !seen[id] {
			errors = append(errors, fmt.Errorf("plan index has no body: %s", id))
		}
	}
	return errors
}

func TestPlanValidationDoesNotSilentlySkipRenamedOrClosedSections(t *testing.T) {
	t.Parallel()
	index := "| [EP-20260905-example](#ep-20260905-example) | active | Work |\n"
	body := "\n## Active Plans\n\n## EP-20260905-example\nStatus: active\nNext action: Verify behavior.\n### Goals\n- [ ] Verify behavior.\n"
	valid := index + body
	if errors := activePlanErrors(valid); len(errors) != 0 {
		t.Fatalf("valid plan rejected: %v", errors)
	}
	for label, invalid := range map[string]string{
		"missing index":   body,
		"missing body":    index + "\n## Active Plans\n",
		"duplicate body":  valid + body[strings.Index(body, "\n## EP-"):],
		"duplicate index": index + valid,
		"wrong anchor":    strings.Replace(valid, "#ep-20260905-example", "#stale", 1),
		"closed goals":    strings.ReplaceAll(valid, "- [ ]", "- [x]"),
		"status drift":    strings.Replace(valid, "Status: active", "Status: blocked", 1),
		"legacy format":   index + "\n## Active Plans\n### Plan ID\nEP-20260905-example\n",
		"missing action":  strings.Replace(valid, "Next action: Verify behavior.\n", "", 1),
	} {
		if errors := activePlanErrors(invalid); len(errors) == 0 {
			t.Errorf("accepted %s", label)
		}
	}
	// Closed implementation goals do not erase a remaining qualification/review gate.
	blocked := strings.ReplaceAll(strings.ReplaceAll(valid, "active", "blocked"), "- [ ]", "- [x]")
	if errors := activePlanErrors(blocked); len(errors) != 0 {
		t.Fatalf("explicit pending gate rejected: %v", errors)
	}
}
