package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	BaselineBundleManifestPath  = "skills/bundle-manifest.json"
	baselineBundleSchemaVersion = 1
	baselineBundleVersion       = 1
)

type BaselineBundleManifest struct {
	SchemaVersion        int                         `json:"schema_version"`
	BundleVersion        int                         `json:"bundle_version"`
	PromptSurfacePolicy  BaselinePromptSurfacePolicy `json:"prompt_surface_policy"`
	PromptPacks          []string                    `json:"prompt_packs,omitempty"`
	ReferenceOnlyPrompts []string                    `json:"reference_only_prompts,omitempty"`
	EditableArtifacts    []BaselineEditableArtifact  `json:"editable_artifacts,omitempty"`
}

type BaselinePromptSurfacePolicy struct {
	LiveHeadlessSource   string `json:"live_headless_source"`
	ReferenceOnlyPattern string `json:"reference_only_pattern"`
}

type BaselineEditableArtifact struct {
	Path        string `json:"path"`
	Label       string `json:"label"`
	Category    string `json:"category"`
	PromptUsage string `json:"prompt_usage,omitempty"`
}

var baselineStaticEditableArtifacts = []BaselineEditableArtifact{
	{Path: "charter/overview.md", Label: "charter/overview.md", Category: "charter"},
	{Path: "charter/rules.yaml", Label: "charter/rules.yaml", Category: "charter"},
	{Path: "charter/nfr.yaml", Label: "charter/nfr.yaml", Category: "charter"},
	{Path: "charter/glossary.yaml", Label: "charter/glossary.yaml", Category: "charter"},
	{Path: "skills/subagents.yaml", Label: "skills/subagents.yaml", Category: "bundle"},
}

func EmbeddedBaselineBundleManifest() BaselineBundleManifest {
	promptPackPaths := make([]string, 0, len(baselinePromptPacks))
	for pack := range baselinePromptPacks {
		promptPackPaths = append(promptPackPaths, filepath.ToSlash(filepath.Join("skills", "prompt-packs", pack+".md")))
	}
	sort.Strings(promptPackPaths)

	referenceOnlyPrompts := make([]string, 0, len(baselineSkillIDs)*2)
	editableArtifacts := make([]BaselineEditableArtifact, 0, len(baselineStaticEditableArtifacts)+len(promptPackPaths)+len(baselineSkillIDs)*2)
	editableArtifacts = append(editableArtifacts, baselineStaticEditableArtifacts...)
	for _, path := range promptPackPaths {
		editableArtifacts = append(editableArtifacts, BaselineEditableArtifact{
			Path:        path,
			Label:       path,
			Category:    "prompt-pack",
			PromptUsage: "live-consumed",
		})
	}
	for _, skill := range baselineSkillIDs {
		for _, promptName := range []string{"system", "task"} {
			path := filepath.ToSlash(filepath.Join("skills", skill, "prompts", promptName+".md"))
			referenceOnlyPrompts = append(referenceOnlyPrompts, path)
			editableArtifacts = append(editableArtifacts, BaselineEditableArtifact{
				Path:        path,
				Label:       path + " (reference-only)",
				Category:    "skill-prompt",
				PromptUsage: "reference-only",
			})
		}
	}
	sort.Strings(referenceOnlyPrompts)
	sort.Slice(editableArtifacts, func(i, j int) bool {
		return editableArtifacts[i].Path < editableArtifacts[j].Path
	})

	return BaselineBundleManifest{
		SchemaVersion: baselineBundleSchemaVersion,
		BundleVersion: baselineBundleVersion,
		PromptSurfacePolicy: BaselinePromptSurfacePolicy{
			LiveHeadlessSource:   "skills/prompt-packs/*.md",
			ReferenceOnlyPattern: "skills/*/prompts/*.md",
		},
		PromptPacks:          promptPackPaths,
		ReferenceOnlyPrompts: referenceOnlyPrompts,
		EditableArtifacts:    editableArtifacts,
	}
}

func renderBaselineBundleManifest() ([]byte, error) {
	payload, err := json.MarshalIndent(EmbeddedBaselineBundleManifest(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal baseline bundle manifest: %w", err)
	}
	return append(payload, '\n'), nil
}

func (r Root) EffectiveBaselineBundleManifest() (BaselineBundleManifest, []Diagnostic) {
	embedded := EmbeddedBaselineBundleManifest()
	manifestPath, err := r.Resolve(BaselineBundleManifestPath)
	if err != nil {
		return embedded, []Diagnostic{{
			Level:      DiagnosticError,
			Code:       "workspace.skills.bundle_manifest.invalid_path",
			Message:    err.Error(),
			Suggestion: "Keep skills/bundle-manifest.json inside workspace root",
		}}
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return embedded, []Diagnostic{{
				Level:      DiagnosticWarning,
				Code:       "workspace.skills.bundle_manifest.missing",
				Path:       manifestPath,
				Message:    "skills/bundle-manifest.json is missing; embedded baseline inventory will be used",
				Suggestion: "Run init-workspace or serve --auto-init to re-seed baseline bundle manifest",
			}}
		}
		return embedded, []Diagnostic{{
			Level:      DiagnosticWarning,
			Code:       "workspace.skills.bundle_manifest.unreadable",
			Path:       manifestPath,
			Message:    fmt.Sprintf("cannot read skills/bundle-manifest.json: %v", err),
			Suggestion: "Fix filesystem permissions or regenerate baseline bundle manifest",
		}}
	}
	var manifest BaselineBundleManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return embedded, []Diagnostic{{
			Level:      DiagnosticWarning,
			Code:       "workspace.skills.bundle_manifest.invalid_json",
			Path:       manifestPath,
			Message:    fmt.Sprintf("cannot parse skills/bundle-manifest.json: %v", err),
			Suggestion: "Fix JSON syntax or regenerate baseline bundle manifest",
		}}
	}
	if manifest.SchemaVersion != baselineBundleSchemaVersion || manifest.BundleVersion != baselineBundleVersion {
		return embedded, []Diagnostic{{
			Level:      DiagnosticWarning,
			Code:       "workspace.skills.bundle_manifest.stale",
			Path:       manifestPath,
			Message:    fmt.Sprintf("baseline bundle manifest version mismatch: found schema=%d bundle=%d, expected schema=%d bundle=%d", manifest.SchemaVersion, manifest.BundleVersion, baselineBundleSchemaVersion, baselineBundleVersion),
			Suggestion: "Re-seed baseline bundle manifest to align UI/editor inventory with embedded runtime bundle",
		}}
	}
	normalizeBaselineBundleManifest(&manifest)
	return manifest, nil
}

func normalizeBaselineBundleManifest(manifest *BaselineBundleManifest) {
	if manifest == nil {
		return
	}
	manifest.PromptSurfacePolicy.LiveHeadlessSource = strings.TrimSpace(manifest.PromptSurfacePolicy.LiveHeadlessSource)
	manifest.PromptSurfacePolicy.ReferenceOnlyPattern = strings.TrimSpace(manifest.PromptSurfacePolicy.ReferenceOnlyPattern)
	manifest.PromptPacks = normalizeOrderedUniquePaths(manifest.PromptPacks)
	manifest.ReferenceOnlyPrompts = normalizeOrderedUniquePaths(manifest.ReferenceOnlyPrompts)
	for idx := range manifest.EditableArtifacts {
		manifest.EditableArtifacts[idx].Path = filepath.ToSlash(strings.TrimSpace(manifest.EditableArtifacts[idx].Path))
		manifest.EditableArtifacts[idx].Label = strings.TrimSpace(manifest.EditableArtifacts[idx].Label)
		manifest.EditableArtifacts[idx].Category = strings.TrimSpace(manifest.EditableArtifacts[idx].Category)
		manifest.EditableArtifacts[idx].PromptUsage = strings.TrimSpace(manifest.EditableArtifacts[idx].PromptUsage)
	}
	sort.Slice(manifest.EditableArtifacts, func(i, j int) bool {
		return manifest.EditableArtifacts[i].Path < manifest.EditableArtifacts[j].Path
	})
}

func normalizeOrderedUniquePaths(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := filepath.ToSlash(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func (r Root) validateBaselineBundleManifest() []Diagnostic {
	_, diagnostics := r.EffectiveBaselineBundleManifest()
	return diagnostics
}
