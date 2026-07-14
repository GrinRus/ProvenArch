package workspacehealth

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestScanReturnsPassForEmptyWorkspaceHealthSurface(t *testing.T) {
	t.Parallel()

	ws := newHealthWorkspace(t)
	report, err := Scan(context.Background(), ws, Options{Now: fixedHealthTime})
	if err != nil {
		t.Fatalf("scan workspace health: %v", err)
	}

	if report.Version != Version {
		t.Fatalf("expected version %d, got %d", Version, report.Version)
	}
	if !report.GeneratedAt.Equal(fixedHealthTime()) {
		t.Fatalf("expected fixed generated_at, got %s", report.GeneratedAt)
	}
	if report.Status != StatusPass {
		t.Fatalf("expected pass status, got %s", report.Status)
	}
	if len(report.Items) != 0 {
		t.Fatalf("expected no health items, got %+v", report.Items)
	}
	if report.Summary != (Summary{}) {
		t.Fatalf("expected empty summary, got %+v", report.Summary)
	}
}

func TestScanReportsWorkspaceHealthWarningsAndOpenQuestions(t *testing.T) {
	t.Parallel()

	ws := newHealthWorkspace(t)
	writeWorkspaceFile(t, ws, "model/entities/svc.payments.yaml", strings.Join([]string{
		"id: svc.payments",
		"type: service",
		"name: Payments",
		"provenance:",
		"  kind: observation",
		"  confidence: 0.8",
		"",
	}, "\n"))
	writeWorkspaceFile(t, ws, "model/edges/edge.good.yaml", strings.Join([]string{
		"id: edge.good",
		"type: calls",
		"from: svc.payments",
		"to: svc.ledger",
		"provenance:",
		"  kind: observation",
		"  confidence: 0.8",
		"  evidence:",
		"    - repo: payments",
		"      path: README.md",
		"",
	}, "\n"))
	writeWorkspaceFile(t, ws, "reports/agent-outputs/domains/payments.md", "# Payments\n")
	writeWorkspaceFile(t, ws, "proposals/proposal-payments/proposal.md", "# Proposal\n\n## Evidence\n- `README.md`\n")
	writeWorkspaceFile(t, ws, "reports/coverage/open-questions.md", "# Open Questions\n\n- Who owns payments?\n- Where is CI/CD defined?\n")

	before := snapshotHealthWorkspace(t, ws)
	report, err := Scan(context.Background(), ws, Options{Now: fixedHealthTime})
	if err != nil {
		t.Fatalf("scan workspace health: %v", err)
	}
	after := snapshotHealthWorkspace(t, ws)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("workspace health scan mutated workspace\nbefore=%v\nafter=%v", before, after)
	}

	if report.Status != StatusWarn {
		t.Fatalf("expected warn status, got %s", report.Status)
	}
	if report.Summary.Info != 1 || report.Summary.Warning != 3 || report.Summary.Error != 0 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	assertHealthItem(t, report, "model.observation.missing_evidence", SeverityWarning, "model/entities/svc.payments.yaml")
	assertHealthItem(t, report, "domain.output.orphan", SeverityWarning, "reports/agent-outputs/domains/payments.md")
	assertHealthItem(t, report, "proposal.missing_review_sections", SeverityWarning, "proposals/proposal-payments/proposal.md")
	assertHealthItem(t, report, "coverage.open_questions.count", SeverityInfo, "reports/coverage/open-questions.md")
}

func TestScanReportsInvalidModelYAMLAsFailingHealthItem(t *testing.T) {
	t.Parallel()

	ws := newHealthWorkspace(t)
	writeWorkspaceFile(t, ws, "model/entities/broken.yaml", "id: [\n")

	report, err := Scan(context.Background(), ws, Options{Now: fixedHealthTime})
	if err != nil {
		t.Fatalf("scan workspace health: %v", err)
	}
	if report.Status != StatusFail {
		t.Fatalf("expected fail status, got %s", report.Status)
	}
	if report.Summary.Error != 1 {
		t.Fatalf("expected one error item, got %+v", report.Summary)
	}
	assertHealthItem(t, report, "model.entity.invalid_yaml", SeverityError, "model/entities/broken.yaml")
}

func assertHealthItem(t *testing.T, report Report, id string, severity Severity, path string) {
	t.Helper()

	for _, item := range report.Items {
		if item.ID == id && item.Severity == severity && item.Path == path {
			return
		}
	}
	t.Fatalf("expected health item id=%s severity=%s path=%s in %+v", id, severity, path, report.Items)
}

func newHealthWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repo := filepath.Join(root, "repos", "payments")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("create repo: %v", err)
	}
	manifest := "version: 1\nrepos:\n  - name: payments\n    path: " + repo + "\n"
	if err := os.WriteFile(filepath.Join(root, workspace.ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func writeWorkspaceFile(t *testing.T, ws workspace.Root, rel string, content string) {
	t.Helper()

	abs, err := ws.Resolve(rel)
	if err != nil {
		t.Fatalf("resolve %s: %v", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func snapshotHealthWorkspace(t *testing.T, ws workspace.Root) map[string]string {
	t.Helper()

	snapshot := map[string]string{}
	if err := filepath.WalkDir(ws.Path, func(abs string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(ws.Path, abs)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(rel)] = string(raw)
		return nil
	}); err != nil {
		t.Fatalf("snapshot workspace: %v", err)
	}
	return snapshot
}

func fixedHealthTime() time.Time {
	return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
}
