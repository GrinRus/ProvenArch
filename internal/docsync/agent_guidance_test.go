package docsync

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Validate discoverability and concrete references without prescribing skill prose.
func TestAgentGuidanceStructureAndReferences(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	skills, err := filepath.Glob(filepath.Join(root, ".agents/skills/*/SKILL.md"))
	if err != nil || len(skills) == 0 {
		t.Fatalf("discover repository skills: %v", err)
	}
	names := map[string]string{}
	paths := []string{"AGENTS.md", "CONTRIBUTING.md", "docs/DOCS_POLICY.md", "docs/AGENT_DEVELOPMENT.md"}
	for _, path := range skills {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		name, err := skillName(string(body))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		if previous, exists := names[name]; exists {
			t.Errorf("duplicate skill name %q in %s and %s", name, previous, path)
		}
		names[name] = path
		rel, _ := filepath.Rel(root, path)
		paths = append(paths, rel)
	}
	for _, path := range paths {
		for _, err := range guidanceReferenceErrors(root, path, readDoc(t, path)) {
			t.Errorf("%s: %v", path, err)
		}
	}
}

func skillName(content string) (string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return "", fmt.Errorf("missing YAML frontmatter")
	}
	parts := strings.SplitN(content[4:], "\n---", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unterminated YAML frontmatter")
	}
	var metadata map[string]interface{}
	if err := yaml.Unmarshal([]byte(parts[0]), &metadata); err != nil {
		return "", err
	}
	name, _ := metadata["name"].(string)
	description, _ := metadata["description"].(string)
	if !regexp.MustCompile("^[a-z0-9]+(-[a-z0-9]+)*$").MatchString(name) || len(name) > 64 {
		return "", fmt.Errorf("invalid skill name %q", name)
	}
	if strings.TrimSpace(description) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("description and instructions must be nonempty")
	}
	return name, nil
}

func guidanceReferenceErrors(root, rel, content string) []error {
	var errors []error
	// Markdown links are relative to the source file; commands are repository-relative.
	for _, match := range regexp.MustCompile("\\[[^\\]\n]*\\]\\(([^)]+)\\)").FindAllStringSubmatch(content, -1) {
		target := strings.Trim(match[1], "<>")
		u, err := url.Parse(target)
		if err != nil {
			errors = append(errors, fmt.Errorf("invalid link %q", target))
			continue
		}
		if u.Scheme != "" || u.Host != "" {
			continue
		}
		path := filepath.Join(root, filepath.Dir(rel), filepath.FromSlash(u.Path))
		if u.Path == "" {
			path = filepath.Join(root, rel)
		}
		if _, err := os.Stat(path); err != nil {
			errors = append(errors, fmt.Errorf("unresolved local link %q: %w", target, err))
			continue
		}
		if u.Fragment != "" {
			document, err := os.ReadFile(path)
			if err != nil || !markdownFragments(string(document))[u.Fragment] {
				errors = append(errors, fmt.Errorf("unresolved local fragment %q", target))
			}
		}
	}
	for _, path := range regexp.MustCompile("scripts/[a-zA-Z0-9_-]+\\.(?:sh|py)").FindAllString(content, -1) {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			errors = append(errors, fmt.Errorf("unresolved script %q: %w", path, err))
		}
	}
	return errors
}

func markdownFragments(content string) map[string]bool {
	fragments := map[string]bool{}
	counts := map[string]int{}
	fence := ""
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if fence == "" {
				fence = trimmed[:3]
			} else if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if heading := regexp.MustCompile("^#{1,6} +(.+?) *#*$").FindStringSubmatch(line); heading != nil {
			var slug strings.Builder
			for _, char := range strings.ToLower(heading[1]) {
				if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' {
					slug.WriteRune(char)
				} else if char == ' ' {
					slug.WriteByte('-')
				}
			}
			base := slug.String()
			anchor := base
			if counts[base] > 0 {
				anchor = fmt.Sprintf("%s-%d", base, counts[base])
			}
			counts[base]++
			fragments[anchor] = true
		}
		for _, id := range regexp.MustCompile("<(?:a|[hH][1-6])[^>]*\\bid=[\"']([^\"']+)[\"']").FindAllStringSubmatch(line, -1) {
			fragments[id[1]] = true
		}
	}
	return fragments
}

func TestSkillMetadataRejectsUndiscoverableInstructions(t *testing.T) {
	t.Parallel()
	for _, content := range []string{
		"# Missing metadata",
		"---\nname: acp-example\n",
		"---\nname: acp-example\nname: duplicate\ndescription: Example\n---\nRun checks.",
		"---\nname: Wrong_Name\ndescription: Example\n---\nRun checks.",
		"---\nname: acp-example\ndescription: []\n---\nRun checks.",
		"---\nname: acp-example\ndescription: Example\n---\n",
	} {
		if _, err := skillName(content); err == nil {
			t.Errorf("accepted invalid skill: %q", content)
		}
	}
	if name, err := skillName("---\nname: acp-example\ndescription: Example\nmetadata:\n  short-description: Example\n---\nRun checks."); err != nil || name != "acp-example" {
		t.Fatalf("rejected skill with optional metadata: %q, %v", name, err)
	}
}

func TestGuidanceReferencesResolveFromTheirOwningFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("Instructions"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs/guide.md"), []byte("## Workflow\n## Карта изменений\n## Workflow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	valid := "[rules](../AGENTS.md) [web](https://example.com/guide) [section](#workflow) [duplicate](#workflow-1) [map](#карта-изменений)"
	if errors := guidanceReferenceErrors(root, "docs/guide.md", valid); len(errors) != 0 {
		t.Fatalf("valid relative references rejected: %v", errors)
	}
	invalid := "[bad](missing.md) Run ./scripts/missing.sh [fragment](#missing) [other](../AGENTS.md#missing)"
	if errors := guidanceReferenceErrors(root, "docs/guide.md", invalid); len(errors) != 4 {
		t.Fatalf("expected missing document, script and fragment diagnostics, got %v", errors)
	}
}
