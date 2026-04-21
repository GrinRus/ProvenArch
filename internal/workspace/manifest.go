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
	Name     string              `yaml:"name" json:"name"`
	Path     string              `yaml:"path,omitempty" json:"path,omitempty"`
	GitURL   string              `yaml:"git_url,omitempty" json:"git_url,omitempty"`
	Ref      string              `yaml:"ref,omitempty" json:"ref,omitempty"`
	Analysis *RepoAnalysisConfig `yaml:"analysis,omitempty" json:"analysis,omitempty"`
}

type RepoAnalysisConfig struct {
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Role    string   `yaml:"role,omitempty" json:"role,omitempty"`
}

type DocsConfig struct {
	ImportsPath string `yaml:"imports_path,omitempty" json:"imports_path,omitempty"`
}

type RuntimeConfig struct {
	Profile *RuntimeProfileConfig `yaml:"profile,omitempty" json:"profile,omitempty"`
}

type RuntimeProfileConfig struct {
	Timeouts  *RuntimeTimeoutsConfig  `yaml:"timeouts,omitempty" json:"timeouts,omitempty"`
	Execution *RuntimeExecutionConfig `yaml:"execution,omitempty" json:"execution,omitempty"`
	Steps     *RuntimeStepsConfig     `yaml:"steps,omitempty" json:"steps,omitempty"`
}

type RuntimeExecutionConfig struct {
	Strategy       string                       `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	MaxParallel    *int                         `yaml:"max_parallel_tasks,omitempty" json:"max_parallel_tasks,omitempty"`
	FailurePolicy  string                       `yaml:"failure_policy,omitempty" json:"failure_policy,omitempty"`
	ShardDiscovery *RuntimeShardDiscoveryConfig `yaml:"shard_discovery,omitempty" json:"shard_discovery,omitempty"`
}

type RuntimeShardDiscoveryConfig struct {
	Mode string `yaml:"mode,omitempty" json:"mode,omitempty"`
}

type RuntimeStepsConfig struct {
	Step0Constitution *RuntimeStepConfig `yaml:"step0_constitution,omitempty" json:"step0_constitution,omitempty"`
	Step1Collect      *RuntimeStepConfig `yaml:"step1_collect,omitempty" json:"step1_collect,omitempty"`
	Step2AsIs         *RuntimeStepConfig `yaml:"step2_as_is,omitempty" json:"step2_as_is,omitempty"`
	Step3Findings     *RuntimeStepConfig `yaml:"step3_findings,omitempty" json:"step3_findings,omitempty"`
	Step4Proposals    *RuntimeStepConfig `yaml:"step4_proposals,omitempty" json:"step4_proposals,omitempty"`
}

type RuntimeStepConfig struct {
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
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

const (
	RepoRoleBackend  = "backend"
	RepoRoleFrontend = "frontend"
	RepoRoleMixed    = "mixed"
	RepoRoleUnknown  = "unknown"
)

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

func (cfg *RuntimeExecutionConfig) IsZero() bool {
	if cfg == nil {
		return true
	}
	return strings.TrimSpace(cfg.Strategy) == "" &&
		cfg.MaxParallel == nil &&
		strings.TrimSpace(cfg.FailurePolicy) == "" &&
		(cfg.ShardDiscovery == nil || strings.TrimSpace(cfg.ShardDiscovery.Mode) == "")
}

func (cfg *RuntimeStepConfig) IsZero() bool {
	return cfg == nil || strings.TrimSpace(cfg.Provider) == ""
}

func (cfg *RuntimeStepsConfig) IsZero() bool {
	if cfg == nil {
		return true
	}
	return (cfg.Step0Constitution == nil || cfg.Step0Constitution.IsZero()) &&
		(cfg.Step1Collect == nil || cfg.Step1Collect.IsZero()) &&
		(cfg.Step2AsIs == nil || cfg.Step2AsIs.IsZero()) &&
		(cfg.Step3Findings == nil || cfg.Step3Findings.IsZero()) &&
		(cfg.Step4Proposals == nil || cfg.Step4Proposals.IsZero())
}

func (cfg *RuntimeProfileConfig) IsZero() bool {
	if cfg == nil {
		return true
	}
	return (cfg.Timeouts == nil || cfg.Timeouts.IsZero()) &&
		(cfg.Execution == nil || cfg.Execution.IsZero()) &&
		(cfg.Steps == nil || cfg.Steps.IsZero())
}

func (cfg *RuntimeConfig) IsZero() bool {
	return cfg == nil || cfg.Profile == nil || cfg.Profile.IsZero()
}

func (cfg *RepoAnalysisConfig) IsZero() bool {
	return cfg == nil || (len(cfg.Include) == 0 && len(cfg.Exclude) == 0 && strings.TrimSpace(cfg.Role) == "")
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
	for idx := range manifest.Repos {
		manifest.Repos[idx].Name = strings.TrimSpace(manifest.Repos[idx].Name)
		manifest.Repos[idx].Path = strings.TrimSpace(manifest.Repos[idx].Path)
		manifest.Repos[idx].GitURL = strings.TrimSpace(manifest.Repos[idx].GitURL)
		manifest.Repos[idx].Ref = strings.TrimSpace(manifest.Repos[idx].Ref)
		if manifest.Repos[idx].Analysis != nil {
			manifest.Repos[idx].Analysis.Include = normalizeOrderedUniqueStrings(manifest.Repos[idx].Analysis.Include)
			manifest.Repos[idx].Analysis.Exclude = normalizeOrderedUniqueStrings(manifest.Repos[idx].Analysis.Exclude)
			manifest.Repos[idx].Analysis.Role = strings.TrimSpace(strings.ToLower(manifest.Repos[idx].Analysis.Role))
			if manifest.Repos[idx].Analysis.IsZero() {
				manifest.Repos[idx].Analysis = nil
			}
		}
	}

	if strings.TrimSpace(manifest.Docs.ImportsPath) == "" {
		manifest.Docs.ImportsPath = "./docs/imports"
	}
	if manifest.Runtime != nil {
		if manifest.Runtime.Profile != nil {
			if manifest.Runtime.Profile.Timeouts != nil && manifest.Runtime.Profile.Timeouts.IsZero() {
				manifest.Runtime.Profile.Timeouts = nil
			}
			if manifest.Runtime.Profile.Execution != nil {
				manifest.Runtime.Profile.Execution.Strategy = strings.TrimSpace(manifest.Runtime.Profile.Execution.Strategy)
				manifest.Runtime.Profile.Execution.FailurePolicy = strings.TrimSpace(manifest.Runtime.Profile.Execution.FailurePolicy)
				if manifest.Runtime.Profile.Execution.ShardDiscovery != nil {
					manifest.Runtime.Profile.Execution.ShardDiscovery.Mode = strings.TrimSpace(manifest.Runtime.Profile.Execution.ShardDiscovery.Mode)
				}
				if manifest.Runtime.Profile.Execution.IsZero() {
					manifest.Runtime.Profile.Execution = nil
				}
			}
			if manifest.Runtime.Profile.Steps != nil {
				normalizeStep := func(step *RuntimeStepConfig) *RuntimeStepConfig {
					if step == nil {
						return nil
					}
					step.Provider = strings.TrimSpace(strings.ToLower(step.Provider))
					if step.IsZero() {
						return nil
					}
					return step
				}
				manifest.Runtime.Profile.Steps.Step0Constitution = normalizeStep(manifest.Runtime.Profile.Steps.Step0Constitution)
				manifest.Runtime.Profile.Steps.Step1Collect = normalizeStep(manifest.Runtime.Profile.Steps.Step1Collect)
				manifest.Runtime.Profile.Steps.Step2AsIs = normalizeStep(manifest.Runtime.Profile.Steps.Step2AsIs)
				manifest.Runtime.Profile.Steps.Step3Findings = normalizeStep(manifest.Runtime.Profile.Steps.Step3Findings)
				manifest.Runtime.Profile.Steps.Step4Proposals = normalizeStep(manifest.Runtime.Profile.Steps.Step4Proposals)
				if manifest.Runtime.Profile.Steps.IsZero() {
					manifest.Runtime.Profile.Steps = nil
				}
			}
			if manifest.Runtime.Profile.IsZero() {
				manifest.Runtime.Profile = nil
			}
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
		if repo.Analysis != nil {
			for idx, include := range repo.Analysis.Include {
				if strings.TrimSpace(include) == "" {
					problems = append(problems, fmt.Sprintf("%s.analysis.include[%d] must not be empty", indexLabel, idx))
				}
			}
			for idx, exclude := range repo.Analysis.Exclude {
				if strings.TrimSpace(exclude) == "" {
					problems = append(problems, fmt.Sprintf("%s.analysis.exclude[%d] must not be empty", indexLabel, idx))
				}
			}
			if role := strings.TrimSpace(strings.ToLower(repo.Analysis.Role)); role != "" &&
				role != RepoRoleBackend &&
				role != RepoRoleFrontend &&
				role != RepoRoleMixed &&
				role != RepoRoleUnknown {
				problems = append(problems, "analysis.role must be one of: backend, frontend, mixed, unknown")
			}
		}
	}

	if strings.TrimSpace(manifest.Docs.ImportsPath) == "" {
		problems = append(problems, "docs.imports_path must not be empty")
	}
	if manifest.Runtime != nil && manifest.Runtime.Profile != nil {
		if manifest.Runtime.Profile.Timeouts != nil {
			timeoutChecks := []struct {
				name  string
				value *int
			}{
				{name: "runtime.profile.timeouts.step_timeout_sec", value: manifest.Runtime.Profile.Timeouts.StepTimeoutSec},
				{name: "runtime.profile.timeouts.heartbeat_sec", value: manifest.Runtime.Profile.Timeouts.HeartbeatSec},
				{name: "runtime.profile.timeouts.pipeline_timeout_sec", value: manifest.Runtime.Profile.Timeouts.PipelineTimeoutSec},
				{name: "runtime.profile.timeouts.pipeline_kill_grace_sec", value: manifest.Runtime.Profile.Timeouts.PipelineKillGraceSec},
				{name: "runtime.profile.timeouts.api_ready_timeout_sec", value: manifest.Runtime.Profile.Timeouts.APIReadyTimeoutSec},
				{name: "runtime.profile.timeouts.api_init_timeout_sec", value: manifest.Runtime.Profile.Timeouts.APIInitTimeoutSec},
				{name: "runtime.profile.timeouts.ui_init_poll_timeout_sec", value: manifest.Runtime.Profile.Timeouts.UIInitPollTimeoutSec},
				{name: "runtime.profile.timeouts.ui_cancel_poll_timeout_sec", value: manifest.Runtime.Profile.Timeouts.UICancelPollTimeoutSec},
			}
			for _, check := range timeoutChecks {
				if check.value != nil && *check.value <= 0 {
					problems = append(problems, fmt.Sprintf("%s must be > 0", check.name))
				}
			}
		}
		if manifest.Runtime.Profile.Execution != nil {
			execution := manifest.Runtime.Profile.Execution
			if strategy := strings.TrimSpace(execution.Strategy); strategy != "" && strategy != "sequential" && strategy != "parallel" {
				problems = append(problems, "runtime.profile.execution.strategy must be one of: sequential, parallel")
			}
			if execution.MaxParallel != nil && *execution.MaxParallel <= 0 {
				problems = append(problems, "runtime.profile.execution.max_parallel_tasks must be > 0")
			}
			if policy := strings.TrimSpace(execution.FailurePolicy); policy != "" && policy != "fail_fast" && policy != "best_effort" {
				problems = append(problems, "runtime.profile.execution.failure_policy must be one of: fail_fast, best_effort")
			}
			if execution.ShardDiscovery != nil {
				mode := strings.TrimSpace(execution.ShardDiscovery.Mode)
				if mode != "" && mode != "heuristics" && mode != "semantic" {
					problems = append(problems, "runtime.profile.execution.shard_discovery.mode must be one of: heuristics, semantic")
				}
			}
		}
		if manifest.Runtime.Profile.Steps != nil {
			validateStepProvider := func(label string, step *RuntimeStepConfig) {
				if step == nil {
					return
				}
				provider := strings.TrimSpace(strings.ToLower(step.Provider))
				if provider == "" {
					return
				}
				if provider != "claude-code" && provider != "qwen-code" {
					problems = append(problems, fmt.Sprintf("%s.provider must be one of: claude-code, qwen-code", label))
				}
			}
			validateStepProvider("runtime.profile.steps.step0_constitution", manifest.Runtime.Profile.Steps.Step0Constitution)
			validateStepProvider("runtime.profile.steps.step1_collect", manifest.Runtime.Profile.Steps.Step1Collect)
			validateStepProvider("runtime.profile.steps.step2_as_is", manifest.Runtime.Profile.Steps.Step2AsIs)
			validateStepProvider("runtime.profile.steps.step3_findings", manifest.Runtime.Profile.Steps.Step3Findings)
			validateStepProvider("runtime.profile.steps.step4_proposals", manifest.Runtime.Profile.Steps.Step4Proposals)
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

func normalizeOrderedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
