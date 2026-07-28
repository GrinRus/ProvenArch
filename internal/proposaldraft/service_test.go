package proposaldraft

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/qa"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestCreatePublishesImmutableProposalPackage(t *testing.T) {
	t.Parallel()

	ws := newProposalWorkspace(t)
	answerRaw, contextRaw := proposalAnswerFixture(t, "run-qa")
	beforeAnswer := append([]byte(nil), answerRaw...)
	reopened, err := workspace.Open(ws.Path)
	if err != nil {
		t.Fatalf("reopen workspace before proposal mutation: %v", err)
	}
	result, err := Create(reopened, Input{
		RunID:                "run-qa",
		Title:                "Clarify ownership",
		Slug:                 "clarify-ownership",
		OperatorNote:         "Confirm with Team A.",
		ExpectedAnswerDigest: AnswerDigest(answerRaw),
		AnswerRaw:            answerRaw,
		ContextRaw:           contextRaw,
		CreatedAt:            time.Date(2026, 7, 26, 10, 5, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	if !reflect.DeepEqual(answerRaw, beforeAnswer) {
		t.Fatal("proposal creation mutated source answer bytes")
	}
	for _, rel := range []string{result.ProposalPath, result.EvidencePath, result.SourcePath} {
		if _, err := ws.ReadFile(rel); err != nil {
			t.Fatalf("read package file %s: %v", rel, err)
		}
	}
	sourceRaw, err := ws.ReadFile(result.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	source, err := contracts.ParseSourceQAAnswer(sourceRaw)
	if err != nil {
		t.Fatalf("parse source provenance: %v", err)
	}
	if source.SourceRunID != "run-qa" || source.AnswerDigest != AnswerDigest(answerRaw) {
		t.Fatalf("unexpected source provenance: %+v", source)
	}
	if _, err := Create(ws, Input{
		RunID: "run-qa", Title: "Clarify ownership", Slug: "clarify-ownership",
		ExpectedAnswerDigest: AnswerDigest(answerRaw), AnswerRaw: answerRaw, ContextRaw: contextRaw,
	}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestCreateRejectsStaleSlugAndUnresolvedCitationWithoutMutation(t *testing.T) {
	t.Parallel()

	ws := newProposalWorkspace(t)
	answerRaw, contextRaw := proposalAnswerFixture(t, "run-qa")
	tests := []struct {
		name    string
		input   Input
		wantErr error
	}{
		{
			name: "stale",
			input: Input{RunID: "run-qa", Title: "Title", Slug: "title", ExpectedAnswerDigest: "stale",
				AnswerRaw: answerRaw, ContextRaw: contextRaw},
			wantErr: ErrStaleDigest,
		},
		{
			name: "traversal slug",
			input: Input{RunID: "run-qa", Title: "Title", Slug: "../escape", ExpectedAnswerDigest: AnswerDigest(answerRaw),
				AnswerRaw: answerRaw, ContextRaw: contextRaw},
			wantErr: ErrInvalidSlug,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Create(ws, tc.input); !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}

	var answer qa.Answer
	if err := json.Unmarshal(answerRaw, &answer); err != nil {
		t.Fatal(err)
	}
	answer.Citations[0].Path = "reports/missing.md"
	brokenRaw, _ := json.Marshal(answer)
	if _, err := Create(ws, Input{
		RunID: "run-qa", Title: "Broken", Slug: "broken", ExpectedAnswerDigest: AnswerDigest(brokenRaw),
		AnswerRaw: brokenRaw, ContextRaw: contextRaw,
	}); !errors.Is(err, ErrUnresolvedCitation) {
		t.Fatalf("expected unresolved citation, got %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(ws.Path, "proposals"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed requests left proposal artifacts: %+v", entries)
	}
}

func newProposalWorkspace(t *testing.T) workspace.Root {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "version: 1\nrepos:\n  - name: fixture\n    path: " + repo + "\n"
	if err := os.WriteFile(filepath.Join(root, workspace.ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("reports/as-is/overview.md", []byte("# Overview\n")); err != nil {
		t.Fatal(err)
	}
	return ws
}

func proposalAnswerFixture(t *testing.T, runID string) ([]byte, []byte) {
	t.Helper()
	answer := qa.Answer{
		Version: 1, RunID: runID, Question: "Who owns payments?", Answer: "Team A owns payments.",
		Citations:  []qa.Citation{{Path: "reports/as-is/overview.md", Reason: "Architecture Home"}},
		Unresolved: []string{"Escalation path"}, Confidence: 0.8, Provider: "fake",
		GeneratedAt: "2026-07-26T10:00:00Z",
	}
	contextPack := qa.ContextPack{
		Version: 1, RunID: runID, Question: answer.Question, GeneratedAt: answer.GeneratedAt,
		Documents: []qa.ContextDocument{{Path: "reports/as-is/overview.md", Weight: 1, Content: "# Overview"}},
	}
	answerRaw, err := json.Marshal(answer)
	if err != nil {
		t.Fatal(err)
	}
	contextRaw, err := json.Marshal(contextPack)
	if err != nil {
		t.Fatal(err)
	}
	return answerRaw, contextRaw
}
