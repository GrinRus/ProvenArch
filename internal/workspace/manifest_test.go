package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsManifestWithDuplicateRepoNames(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: duplicate
    path: /tmp/one
  - name: duplicate
    git_url: https://example.com/repo.git
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), `duplicate repo.name "duplicate"`) {
		t.Fatalf("expected duplicate repo error, got %v", err)
	}
}

func TestOpenRejectsManifestMissingPathAndGitURL(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: invalid
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "must contain exactly one of path or git_url") {
		t.Fatalf("expected path/git_url invariant error, got %v", err)
	}
}

func TestOpenRejectsManifestWithNonPositiveRuntimeTimeout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: /tmp/payments
runtime:
  profile:
    timeouts:
      step_timeout_sec: 0
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "runtime.profile.timeouts.step_timeout_sec must be > 0") {
		t.Fatalf("expected runtime timeout validation error, got %v", err)
	}
}

func TestOpenRejectsLegacyRuntimeTimeoutsShape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: /tmp/payments
runtime:
  timeouts:
    step_timeout_sec: 10
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "additionalProperties 'timeouts' not allowed") {
		t.Fatalf("expected schema rejection for legacy runtime.timeouts shape, got %v", err)
	}
}

func TestOpenRejectsManifestWithInvalidRepoAnalysisRole(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: /tmp/payments
    analysis:
      role: data
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "analysis.role must be one of: backend, frontend, mixed, unknown") {
		t.Fatalf("expected analysis.role validation error, got %v", err)
	}
}

func TestOpenRejectsManifestWithLegacyRepoSelectionField(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: /tmp/payments
runtime:
  profile:
    execution:
      repo_selection: all
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "additionalProperties 'repo_selection' not allowed") {
		t.Fatalf("expected schema rejection for legacy repo_selection field, got %v", err)
	}
}

func TestOpenRejectsManifestWithInvalidStepProvider(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: /tmp/payments
runtime:
  profile:
    steps:
      step2_as_is:
        provider: bogus
`)

	_, err := Open(root)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "runtime.profile.steps.step2_as_is.provider") {
		t.Fatalf("expected step provider validation error, got %v", err)
	}
}

func TestRenderManifestIncludesRuntimeTimeouts(t *testing.T) {
	t.Parallel()

	step := 1800
	pipeline := 2400
	raw, err := RenderManifest(Manifest{
		Version: 1,
		Repos: []RepoSource{
			{Name: "payments", Path: "/tmp/payments"},
		},
		Runtime: &RuntimeConfig{
			Profile: &RuntimeProfileConfig{
				Timeouts: &RuntimeTimeoutsConfig{
					StepTimeoutSec:     &step,
					PipelineTimeoutSec: &pipeline,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("render manifest: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "runtime:") || !strings.Contains(text, "profile:") || !strings.Contains(text, "timeouts:") {
		t.Fatalf("expected runtime timeout section, got:\n%s", text)
	}
	if !strings.Contains(text, "step_timeout_sec: 1800") {
		t.Fatalf("expected step timeout in manifest, got:\n%s", text)
	}
	if !strings.Contains(text, "pipeline_timeout_sec: 2400") {
		t.Fatalf("expected pipeline timeout in manifest, got:\n%s", text)
	}
}

func TestRenderManifestIncludesRuntimeStepProviders(t *testing.T) {
	t.Parallel()

	raw, err := RenderManifest(Manifest{
		Version: 1,
		Repos: []RepoSource{
			{Name: "payments", Path: "/tmp/payments"},
		},
		Runtime: &RuntimeConfig{
			Profile: &RuntimeProfileConfig{
				Steps: &RuntimeStepsConfig{
					Step1Collect:   &RuntimeStepConfig{Provider: "qwen-code"},
					Step4Proposals: &RuntimeStepConfig{Provider: "claude-code"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("render manifest: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "steps:") {
		t.Fatalf("expected runtime steps section, got:\n%s", text)
	}
	if !strings.Contains(text, "step1_collect:") || !strings.Contains(text, "provider: qwen-code") {
		t.Fatalf("expected step1 provider in manifest, got:\n%s", text)
	}
	if !strings.Contains(text, "step4_proposals:") || !strings.Contains(text, "provider: claude-code") {
		t.Fatalf("expected step4 provider in manifest, got:\n%s", text)
	}
}

func TestRenderManifestIncludesCodexRuntimeStepProvider(t *testing.T) {
	t.Parallel()

	raw, err := RenderManifest(Manifest{
		Version: 1,
		Repos: []RepoSource{
			{Name: "payments", Path: "/tmp/payments"},
		},
		Runtime: &RuntimeConfig{
			Profile: &RuntimeProfileConfig{
				Steps: &RuntimeStepsConfig{
					Step3Findings: &RuntimeStepConfig{Provider: "codex-code"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("render manifest: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "step3_findings:") || !strings.Contains(text, "provider: codex-code") {
		t.Fatalf("expected codex step provider in manifest, got:\n%s", text)
	}
}

func TestResolveRejectsAbsoluteAndTraversal(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	if _, err := ws.Resolve("/etc/passwd"); err != ErrPathAbsolute {
		t.Fatalf("expected ErrPathAbsolute, got %v", err)
	}
	if _, err := ws.Resolve("../outside"); err != ErrPathTraversal {
		t.Fatalf("expected ErrPathTraversal, got %v", err)
	}
}

func TestEnsureLayoutCreatesRequiredDirectories(t *testing.T) {
	t.Parallel()

	root := writeValidWorkspaceRoot(t)
	ws, err := Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	for _, rel := range []string{
		"model/entities",
		"reports/as-is/services",
		"reports/changelog",
		"charter/cards/domains",
	} {
		abs := filepath.Join(root, rel)
		info, err := os.Stat(abs)
		if err != nil {
			t.Fatalf("stat %q: %v", rel, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", rel)
		}
	}
}

func writeValidWorkspaceRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeManifestFile(t, root, `
version: 1
repos:
  - name: payments
    path: /tmp/payments
`)
	return root
}

func writeManifestFile(t *testing.T, root string, content string) {
	t.Helper()
	manifestPath := filepath.Join(root, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}
