package contracts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSourceQAAnswerExample(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "source-qa-answer.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := ParseSourceQAAnswer(raw)
	if err != nil {
		t.Fatalf("parse source QA answer: %v", err)
	}
	if source.Version != 1 || source.SourceRunID == "" || len(source.AnswerDigest) != 64 {
		t.Fatalf("unexpected source QA answer: %+v", source)
	}
}

func TestParseSourceQAAnswerRejectsAdditionalProperties(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
	  "version": 1,
	  "source_run_id": "run-1",
	  "answer_digest": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	  "proposal_title": "Title",
	  "question": "Question?",
	  "answer_generated_at": "2026-07-26T10:00:00Z",
	  "citations": [],
	  "unresolved": [],
	  "created_at": "2026-07-26T10:05:00Z",
	  "unexpected": true
	}`)
	if _, err := ParseSourceQAAnswer(raw); err == nil {
		t.Fatal("expected closed schema rejection")
	}
}
