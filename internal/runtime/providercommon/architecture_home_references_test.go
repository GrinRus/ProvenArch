package providercommon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
)

func TestValidateRequiredRuntimeDraftArtifactsResolvesArchitectureHomeRepoReferences(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-repo-reference-valid")
	repoRoot := filepath.Join(t.TempDir(), "bank-of-anthos")
	for _, relPath := range []string{"README.md", "src/frontend/index.py"} {
		target := filepath.Join(repoRoot, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir repository path: %v", err)
		}
		if err := os.WriteFile(target, []byte("fixture\n"), 0o644); err != nil {
			t.Fatalf("write repository path: %v", err)
		}
	}
	task.RepoScopes = []string{"bank-of-anthos"}
	task.ReadContextRoots = append(task.ReadContextRoots, repoRoot)
	writeValidAsIsDraftForRepoReferences(t, task, strings.ReplaceAll(
		strings.Join(validAsIsArchitectureHomeLines(), "\n"),
		"Repository evidence is under README.md.",
		"Repository evidence is under `bank-of-anthos:README.md` and `bank-of-anthos:src/frontend/`.",
	))

	if _, _, err := ValidateRequiredRuntimeDraftArtifacts(task); err != nil {
		t.Fatalf("expected existing file and directory references to validate: %v", err)
	}
}

func TestValidateRequiredRuntimeDraftArtifactsRejectsMissingArchitectureHomeRepoReference(t *testing.T) {
	t.Parallel()

	task := newAsIsDraftTask(t, "run-repo-reference-missing")
	repoRoot := filepath.Join(t.TempDir(), "bank-of-anthos")
	for _, relPath := range []string{"README.md", "pom.xml", "kubernetes-manifests/deployment.yaml", "src/accounts/accounts-db/schema.sql"} {
		target := filepath.Join(repoRoot, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("mkdir repository path: %v", err)
		}
		if err := os.WriteFile(target, []byte("fixture\n"), 0o644); err != nil {
			t.Fatalf("write repository path: %v", err)
		}
	}
	task.RepoScopes = []string{"bank-of-anthos"}
	task.ReadContextRoots = append(task.ReadContextRoots, repoRoot)
	overview, err := os.ReadFile(filepath.Join("..", "testdata", "contract-rejection", "claude_step2_bank_missing_repo_refs_overview.md"))
	if err != nil {
		t.Fatalf("read live-observed fixture: %v", err)
	}
	writeValidAsIsDraftForRepoReferences(t, task, string(overview))

	_, _, err = ValidateRequiredRuntimeDraftArtifacts(task)
	if err == nil {
		t.Fatalf("expected missing repository references to fail validation")
	}
	for _, want := range []string{
		`"bank-of-anthos:cloudbuild.yaml" is unavailable`,
		`"bank-of-anthos:src/accounts/pom.xml" is unavailable`,
		`"bank-of-anthos:src/ledger/pom.xml" is unavailable`,
		`"bank-of-anthos:src/user-service/" is unavailable`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestValidateRequiredRuntimeDraftArtifactsRejectsEscapingArchitectureHomeRepoReferences(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		reference string
		prepare   func(t *testing.T, repoRoot string)
		want      string
	}{
		{name: "absolute", reference: "bank-of-anthos:/etc/passwd", want: "path must be repository-relative"},
		{name: "traversal", reference: "bank-of-anthos:../outside.md", want: "path escapes the repository root"},
		{
			name:      "symlink",
			reference: "bank-of-anthos:external.md",
			prepare: func(t *testing.T, repoRoot string) {
				t.Helper()
				outside := filepath.Join(t.TempDir(), "outside.md")
				if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
					t.Fatalf("write symlink target: %v", err)
				}
				if err := os.Symlink(outside, filepath.Join(repoRoot, "external.md")); err != nil {
					t.Fatalf("create escaping symlink: %v", err)
				}
			},
			want: "path resolves outside the repository root",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task := newAsIsDraftTask(t, "run-repo-reference-"+tc.name)
			repoRoot := filepath.Join(t.TempDir(), "bank-of-anthos")
			if err := os.MkdirAll(repoRoot, 0o755); err != nil {
				t.Fatalf("mkdir repository root: %v", err)
			}
			if tc.prepare != nil {
				tc.prepare(t, repoRoot)
			}
			task.RepoScopes = []string{"bank-of-anthos"}
			task.ReadContextRoots = append(task.ReadContextRoots, repoRoot)
			overview := strings.ReplaceAll(
				strings.Join(validAsIsArchitectureHomeLines(), "\n"),
				"Repository evidence is under README.md.",
				"Repository evidence is under `"+tc.reference+"`.",
			)
			writeValidAsIsDraftForRepoReferences(t, task, overview)

			_, _, err := ValidateRequiredRuntimeDraftArtifacts(task)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s reference to fail with %q, got %v", tc.name, tc.want, err)
			}
		})
	}
}

func writeValidAsIsDraftForRepoReferences(t *testing.T, task acpruntime.Task, overview string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(task.WriteRoot, "asis-draft-manifest.json"), []byte(steppolicy.RuntimeDraftManifestTaskSkeleton(task)), 0o644); err != nil {
		t.Fatalf("write as-is manifest: %v", err)
	}
	for name, content := range map[string]string{
		"overview.md":          overview,
		"summary.md":           "# Coverage\n\nRepository coverage summary.\n",
		"architect-summary.md": "# Architect summary\n\nDecision-ready architecture summary.\n",
	} {
		if err := os.WriteFile(filepath.Join(task.DraftFinalRoot, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}
