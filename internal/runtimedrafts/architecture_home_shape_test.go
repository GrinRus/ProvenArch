package runtimedrafts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeArchitectureHomeInlineHeadingsPreservesAuthoredBodies(t *testing.T) {
	t.Parallel()
	raw := []byte(strings.Replace(string(readInlineArchitectureHomeFixture(t)), "## System at a glance The application", "## System at a glance  The application", 1))
	normalized, sections, err := NormalizeArchitectureHomeInlineHeadings(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sections, ArchitectureHomeRequiredSections(); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("normalized sections = %#v, want %#v", got, want)
	}
	beforeBodies := inlineArchitectureHomeBodies(string(raw))
	afterBodies := standaloneArchitectureHomeBodies(string(normalized))
	if strings.Join(beforeBodies, "\n") != strings.Join(afterBodies, "\n") {
		t.Fatalf("authored bodies changed\nbefore=%#v\nafter=%#v", beforeBodies, afterBodies)
	}
	if missing := runtimeDraftArchitectureHomeMissingSections(string(normalized)); len(missing) != 0 {
		t.Fatalf("normalized Architecture Home still misses sections: %#v", missing)
	}
}

func TestNormalizeArchitectureHomeInlineHeadingsIsDeterministic(t *testing.T) {
	t.Parallel()
	raw := readInlineArchitectureHomeFixture(t)
	first, _, err := NormalizeArchitectureHomeInlineHeadings(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := NormalizeArchitectureHomeInlineHeadings(raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("normalization is not deterministic")
	}
}

func TestNormalizeArchitectureHomeInlineHeadingsRejectsAmbiguousShapes(t *testing.T) {
	t.Parallel()
	raw := string(readInlineArchitectureHomeFixture(t))
	tests := map[string]string{
		"partial":      strings.Replace(raw, "## Evidence gaps and open questions Unconfirmed details remain explicit gaps in `reports/coverage/summary.md`.\n", "", 1),
		"duplicate":    strings.Replace(raw, "## Analyzed scope", "## System at a glance Duplicate body.\n\n## Analyzed scope", 1),
		"out_of_order": strings.Replace(strings.Replace(raw, "## System at a glance", "## TEMP", 1), "## Analyzed scope", "## System at a glance", 1),
		"empty":        strings.Replace(raw, "## System at a glance The application exposes customer-facing banking services described by `README.md`.", "## System at a glance   ", 1),
		"standalone":   strings.Replace(raw, "## System at a glance The application", "## System at a glance\n\nThe application", 1),
		"wrong_level":  strings.Replace(raw, "## System at a glance", "### System at a glance", 1),
		"code_fence":   "```markdown\n" + raw + "\n```\n",
	}
	for name, input := range tests {
		name, input := name, input
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, _, err := NormalizeArchitectureHomeInlineHeadings([]byte(input)); err == nil {
				t.Fatal("expected ambiguous shape rejection")
			}
		})
	}
}

func readInlineArchitectureHomeFixture(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "fixtures", "scenarios", "architecture-home-inline-headings", "overview.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func inlineArchitectureHomeBodies(text string) []string {
	bodies := []string{}
	for _, line := range strings.Split(text, "\n") {
		for _, section := range ArchitectureHomeRequiredSections() {
			prefix := "## " + section + " "
			if strings.HasPrefix(line, prefix) {
				bodies = append(bodies, strings.TrimPrefix(line, prefix))
			}
		}
	}
	return bodies
}

func standaloneArchitectureHomeBodies(text string) []string {
	lines := strings.Split(text, "\n")
	bodies := []string{}
	for idx, line := range lines {
		for _, section := range ArchitectureHomeRequiredSections() {
			if line == "## "+section && idx+2 < len(lines) {
				bodies = append(bodies, lines[idx+2])
			}
		}
	}
	return bodies
}
