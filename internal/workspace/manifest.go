package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/validation"
	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Version int          `yaml:"version" json:"version"`
	Repos   []RepoSource `yaml:"repos" json:"repos"`
	Docs    DocsConfig   `yaml:"docs" json:"docs"`
}

type RepoSource struct {
	Name   string `yaml:"name" json:"name"`
	Path   string `yaml:"path,omitempty" json:"path,omitempty"`
	GitURL string `yaml:"git_url,omitempty" json:"git_url,omitempty"`
	Ref    string `yaml:"ref,omitempty" json:"ref,omitempty"`
}

type DocsConfig struct {
	ImportsPath string `yaml:"imports_path,omitempty" json:"imports_path,omitempty"`
}

func LoadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	return ParseManifest(raw)
}

func ParseManifest(raw []byte) (Manifest, error) {
	payload, err := parseYAMLPayload(raw)
	if err != nil {
		return Manifest{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Manifest{}, errors.New("manifest must contain a single YAML document")
	}

	applyManifestDefaults(&manifest)
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	if err := validation.ValidatePayload(validation.WorkspaceSchema, payload); err != nil {
		return Manifest{}, fmt.Errorf("workspace manifest is invalid: %w", err)
	}

	return manifest, nil
}

func applyManifestDefaults(manifest *Manifest) {
	if strings.TrimSpace(manifest.Docs.ImportsPath) == "" {
		manifest.Docs.ImportsPath = "./docs/imports"
	}
}

func validateManifest(manifest Manifest) error {
	var problems []string

	if manifest.Version != 1 {
		problems = append(problems, "version must be 1")
	}
	if len(manifest.Repos) == 0 {
		problems = append(problems, "repos must contain at least one entry")
	}

	seenNames := map[string]struct{}{}
	for i, repo := range manifest.Repos {
		indexLabel := fmt.Sprintf("repos[%d]", i)
		name := strings.TrimSpace(repo.Name)
		if name == "" {
			problems = append(problems, fmt.Sprintf("%s.name is required", indexLabel))
		}
		if _, exists := seenNames[name]; exists && name != "" {
			problems = append(problems, fmt.Sprintf("duplicate repo.name %q", name))
		}
		seenNames[name] = struct{}{}

		hasPath := strings.TrimSpace(repo.Path) != ""
		hasGitURL := strings.TrimSpace(repo.GitURL) != ""
		switch {
		case hasPath && hasGitURL:
			problems = append(problems, fmt.Sprintf("%s must contain exactly one of path or git_url", indexLabel))
		case !hasPath && !hasGitURL:
			problems = append(problems, fmt.Sprintf("%s must contain exactly one of path or git_url", indexLabel))
		}
	}

	if strings.TrimSpace(manifest.Docs.ImportsPath) == "" {
		problems = append(problems, "docs.imports_path must not be empty")
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("workspace manifest is invalid: %s", strings.Join(problems, "; "))
}

func parseYAMLPayload(raw []byte) (any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("manifest must contain a single YAML document")
	}

	normalized := normalizeYAMLValue(payload)
	jsonBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("serialize manifest for schema validation: %w", err)
	}

	var jsonPayload any
	if err := json.Unmarshal(jsonBytes, &jsonPayload); err != nil {
		return nil, fmt.Errorf("decode manifest json payload: %w", err)
	}
	return jsonPayload, nil
}

func normalizeYAMLValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			out[key] = normalizeYAMLValue(nested)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			out[fmt.Sprint(key)] = normalizeYAMLValue(nested)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, nested := range typed {
			out = append(out, normalizeYAMLValue(nested))
		}
		return out
	default:
		return typed
	}
}
