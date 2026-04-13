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
	Version int            `yaml:"version" json:"version"`
	Repos   []RepoSource   `yaml:"repos" json:"repos"`
	Docs    DocsConfig     `yaml:"docs" json:"docs"`
	Runtime *RuntimeConfig `yaml:"runtime,omitempty" json:"runtime,omitempty"`
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

type RuntimeConfig struct {
	Timeouts *RuntimeTimeoutsConfig `yaml:"timeouts,omitempty" json:"timeouts,omitempty"`
}

type RuntimeTimeoutsConfig struct {
	StepTimeoutSec         *int `yaml:"step_timeout_sec,omitempty" json:"step_timeout_sec,omitempty"`
	HeartbeatSec           *int `yaml:"heartbeat_sec,omitempty" json:"heartbeat_sec,omitempty"`
	PipelineTimeoutSec     *int `yaml:"pipeline_timeout_sec,omitempty" json:"pipeline_timeout_sec,omitempty"`
	PipelineKillGraceSec   *int `yaml:"pipeline_kill_grace_sec,omitempty" json:"pipeline_kill_grace_sec,omitempty"`
	APIReadyTimeoutSec     *int `yaml:"api_ready_timeout_sec,omitempty" json:"api_ready_timeout_sec,omitempty"`
	APIInitTimeoutSec      *int `yaml:"api_init_timeout_sec,omitempty" json:"api_init_timeout_sec,omitempty"`
	UIInitPollTimeoutSec   *int `yaml:"ui_init_poll_timeout_sec,omitempty" json:"ui_init_poll_timeout_sec,omitempty"`
	UICancelPollTimeoutSec *int `yaml:"ui_cancel_poll_timeout_sec,omitempty" json:"ui_cancel_poll_timeout_sec,omitempty"`
}

func (cfg *RuntimeTimeoutsConfig) IsZero() bool {
	if cfg == nil {
		return true
	}
	return cfg.StepTimeoutSec == nil &&
		cfg.HeartbeatSec == nil &&
		cfg.PipelineTimeoutSec == nil &&
		cfg.PipelineKillGraceSec == nil &&
		cfg.APIReadyTimeoutSec == nil &&
		cfg.APIInitTimeoutSec == nil &&
		cfg.UIInitPollTimeoutSec == nil &&
		cfg.UICancelPollTimeoutSec == nil
}

func (cfg *RuntimeConfig) IsZero() bool {
	return cfg == nil || cfg.Timeouts == nil || cfg.Timeouts.IsZero()
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
	if manifest.Runtime != nil {
		if manifest.Runtime.Timeouts != nil && manifest.Runtime.Timeouts.IsZero() {
			manifest.Runtime.Timeouts = nil
		}
		if manifest.Runtime.IsZero() {
			manifest.Runtime = nil
		}
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
	if manifest.Runtime != nil && manifest.Runtime.Timeouts != nil {
		timeoutChecks := []struct {
			name  string
			value *int
		}{
			{name: "runtime.timeouts.step_timeout_sec", value: manifest.Runtime.Timeouts.StepTimeoutSec},
			{name: "runtime.timeouts.heartbeat_sec", value: manifest.Runtime.Timeouts.HeartbeatSec},
			{name: "runtime.timeouts.pipeline_timeout_sec", value: manifest.Runtime.Timeouts.PipelineTimeoutSec},
			{name: "runtime.timeouts.pipeline_kill_grace_sec", value: manifest.Runtime.Timeouts.PipelineKillGraceSec},
			{name: "runtime.timeouts.api_ready_timeout_sec", value: manifest.Runtime.Timeouts.APIReadyTimeoutSec},
			{name: "runtime.timeouts.api_init_timeout_sec", value: manifest.Runtime.Timeouts.APIInitTimeoutSec},
			{name: "runtime.timeouts.ui_init_poll_timeout_sec", value: manifest.Runtime.Timeouts.UIInitPollTimeoutSec},
			{name: "runtime.timeouts.ui_cancel_poll_timeout_sec", value: manifest.Runtime.Timeouts.UICancelPollTimeoutSec},
		}
		for _, check := range timeoutChecks {
			if check.value != nil && *check.value <= 0 {
				problems = append(problems, fmt.Sprintf("%s must be > 0", check.name))
			}
		}
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("workspace manifest is invalid: %s", strings.Join(problems, "; "))
}

func RenderManifest(manifest Manifest) ([]byte, error) {
	normalized := manifest
	applyManifestDefaults(&normalized)
	raw, err := yaml.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("serialize manifest: %w", err)
	}
	return raw, nil
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
