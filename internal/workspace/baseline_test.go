package workspace

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestEnsureBaselineBundleCreatesMissingArtifacts(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		t.Fatalf("ensure baseline bundle: %v", err)
	}

	required := []string{
		"skills/subagents.yaml",
		"skills/bundle-manifest.json",
		"skills/prompt-packs/collect-context.md",
		"skills/service-inventory/prompts/system.md",
		"skills/service-inventory/prompts/task.md",
		"skills/service-inventory/templates/adr.md",
		"skills/service-inventory/templates/rfc.md",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(ws.Path, rel)); err != nil {
			t.Fatalf("expected %s to be seeded: %v", rel, err)
		}
	}
}

func TestEnsureBaselineSupportBundleDoesNotSeedCanonicalSubagentsOutput(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.EnsureBaselineSupportBundle(); err != nil {
		t.Fatalf("ensure baseline support bundle: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ws.Path, "skills/subagents.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected support bundle to avoid seeding canonical subagents output, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.Path, "skills/prompt-packs/collect-context.md")); err != nil {
		t.Fatalf("expected support bundle to keep prompt packs available: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.Path, BaselineBundleManifestPath)); err != nil {
		t.Fatalf("expected support bundle to keep baseline bundle manifest available: %v", err)
	}
}

func TestEnsureBaselineBundleSeedsMachineReadableManifest(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		t.Fatalf("ensure baseline bundle: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(ws.Path, BaselineBundleManifestPath))
	if err != nil {
		t.Fatalf("read baseline bundle manifest: %v", err)
	}
	var manifest BaselineBundleManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode baseline bundle manifest: %v", err)
	}
	if manifest.SchemaVersion != baselineBundleSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", baselineBundleSchemaVersion, manifest.SchemaVersion)
	}
	if manifest.BundleVersion != baselineBundleVersion {
		t.Fatalf("expected bundle version %d, got %d", baselineBundleVersion, manifest.BundleVersion)
	}
	if len(manifest.EditableArtifacts) == 0 {
		t.Fatalf("expected editable_artifacts in baseline bundle manifest")
	}
}

func TestEnsureBaselineBundleDoesNotOverwriteExistingFiles(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	customPrompt := "custom prompt content\n"
	if err := ws.WriteFile("skills/prompt-packs/collect-context.md", []byte(customPrompt)); err != nil {
		t.Fatalf("write custom prompt pack: %v", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		t.Fatalf("ensure baseline bundle: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "skills/prompt-packs/collect-context.md"))
	if err != nil {
		t.Fatalf("read prompt pack: %v", err)
	}
	if string(content) != customPrompt {
		t.Fatalf("expected custom prompt pack to stay unchanged, got %q", string(content))
	}
}

func TestValidateWarnsOnStaleBaselineBundleManifest(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		t.Fatalf("ensure baseline bundle: %v", err)
	}
	stale := EmbeddedBaselineBundleManifest()
	stale.BundleVersion = 0
	raw, err := json.MarshalIndent(stale, "", "  ")
	if err != nil {
		t.Fatalf("marshal stale bundle manifest: %v", err)
	}
	if err := ws.WriteFile(BaselineBundleManifestPath, append(raw, '\n')); err != nil {
		t.Fatalf("write stale bundle manifest: %v", err)
	}

	report := ws.Validate(context.Background(), ValidateOptions{})
	found := false
	for _, warning := range report.Warnings {
		if warning.Code == "workspace.skills.bundle_manifest.stale" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected stale baseline bundle manifest warning, got %+v", report.Warnings)
	}
}

func TestEnsureBaselineBundleSeedsStructuredPromptDefaults(t *testing.T) {
	t.Parallel()

	ws := writeBaselineWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	if err := ws.EnsureBaselineBundle(); err != nil {
		t.Fatalf("ensure baseline bundle: %v", err)
	}

	requiredSections := []string{
		"## Goal",
		"## Inputs",
		"## Required Output Shape",
		"## Evidence Policy",
		"## Forbidden Behavior",
		"## Fallback When Unknown",
	}

	promptPaths := make([]string, 0, len(baselinePromptPacks)+len(baselineSkillIDs)*2)
	for pack := range baselinePromptPacks {
		promptPaths = append(promptPaths, filepath.Join("skills", "prompt-packs", pack+".md"))
	}
	for _, skill := range baselineSkillIDs {
		promptPaths = append(promptPaths,
			filepath.Join("skills", skill, "prompts", "system.md"),
			filepath.Join("skills", skill, "prompts", "task.md"),
		)
	}
	sort.Strings(promptPaths)

	for _, rel := range promptPaths {
		content, err := os.ReadFile(filepath.Join(ws.Path, rel))
		if err != nil {
			t.Fatalf("read prompt %s: %v", rel, err)
		}
		body := string(content)
		for _, section := range requiredSections {
			if !strings.Contains(body, section) {
				t.Fatalf("prompt %s missing required section %q", rel, section)
			}
		}
		if words := len(strings.Fields(body)); words < 70 {
			t.Fatalf("prompt %s is too short: %d words", rel, words)
		}
	}
}

func TestBaselineCollectPromptPackEncodesLegacyHygieneRules(t *testing.T) {
	t.Parallel()

	content, ok := BaselinePromptPack("collect-context")
	if !ok {
		t.Fatalf("expected collect-context baseline prompt pack")
	}

	expected := []string{
		"semantic.coverage.observed/missing/notes",
		"Each semantic provenance.evidence[*] item must include non-empty repo/path fields",
		"Treat schemas/spec plus the enforced runtime prompt as the only manifest schema source of truth",
		"Do not read reports/taskruns, raw runtime logs, archived plans, or prior shard-pack manifests as schema templates",
		"semantic.coverage MUST use observed/missing/notes; do NOT use covered_topics or alternate coverage keys.",
		"semantic.questions[*] MUST use id + text; do NOT use question or other alias keys.",
	}
	for _, token := range expected {
		if !strings.Contains(content, token) {
			t.Fatalf("expected collect prompt pack to contain %q, got:\n%s", token, content)
		}
	}
}

func TestBaselineFindingsPromptPackRequiresCanonicalVerdictMetadata(t *testing.T) {
	t.Parallel()

	content, ok := BaselinePromptPack("findings")
	if !ok {
		t.Fatalf("expected findings baseline prompt pack")
	}

	expected := []string{
		"`validator-verdict.json` only, with version/run_id/generated_at/verdict/summary/checked_paths",
		"issues[] items use only code/severity/message plus optional path/document_id/citation_id; severity is error|warning",
		"Do NOT put legacy finding-shaped fields inside issues[]: id, title, description, rule_id, related_paths, related_ids, provenance.",
		"Observation provenance evidence must include repo/path for every cited file-level fact",
		"If owner linkage is missing, surface owner-gap finding with explicit uncertainty without forcing FAIL on its own",
	}
	for _, token := range expected {
		if !strings.Contains(content, token) {
			t.Fatalf("expected findings prompt pack to contain %q, got:\n%s", token, content)
		}
	}
}

func TestBaselineProposalsPromptPackRequiresCanonicalDraftManifest(t *testing.T) {
	t.Parallel()

	content, ok := BaselinePromptPack("proposals")
	if !ok {
		t.Fatalf("expected proposals baseline prompt pack")
	}

	expected := []string{
		"proposals-draft-manifest.json MUST include version=1, run_id, step_id, step_contract=\"proposals\", agent_role, outputs[], and optional summary/updated_at.",
		"outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.",
		"Do NOT add legacy top-level fields such as pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps.",
		"Do NOT emit final-index-like proposal envelopes; proposals-draft-manifest.json is only the runtime draft publish map.",
	}
	for _, token := range expected {
		if !strings.Contains(content, token) {
			t.Fatalf("expected proposals prompt pack to contain %q, got:\n%s", token, content)
		}
	}

	system := renderSkillSystemPrompt("proposals")
	for _, token := range []string{
		"step_contract=\"proposals\"",
		"outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.",
		"Do NOT add legacy top-level fields such as pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps.",
	} {
		if !strings.Contains(system, token) {
			t.Fatalf("expected proposals skill prompt to contain %q, got:\n%s", token, system)
		}
	}
}

func writeBaselineWorkspace(t *testing.T) Root {
	t.Helper()

	root := t.TempDir()
	manifest := strings.TrimSpace(`
version: 1
repos:
  - name: sample
    path: /tmp/sample
`) + "\n"
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}
